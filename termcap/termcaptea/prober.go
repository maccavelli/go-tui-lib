package termcaptea

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/teamsg"
	"github.com/maccavelli/go-tui-lib/v2/internal/termevent"
	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// Prober learns the terminal's capabilities from one batch of queries that
// ends with DA1, through a termcap.Probe. A program keeps one beside its
// workspace, returns its Init from the program's Init, and passes every
// message to its Update. It never hides a message from the program.
//
// The batch goes out on the first tea.EnvMsg, because the gated queries and
// tmux passthrough depend on the environment, and tea sends that message
// unordered with Init's command
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md A3). The
// deadline runs from Init.
//
// The prober never sends what tea asks for itself (DECRQM 2026 and 2027,
// XTGETTCAP), because tea acts on those replies: it builds its probe with
// termcap.WithoutModeQuery(2026, 2027), and records them from tea's own
// messages instead.
//
// By default it subscribes to mode 2031 once the terminal supports it, so
// the program must restore the terminal before it exits: quit with Quit,
// or send Restore's bytes first.
type Prober struct {
	opts  []termcap.Option
	probe *termcap.Probe

	init      bool      // Init returned a deadline
	t0        time.Time // when Init ran
	tick      time.Time // the deadline a pending tick is for, or zero
	expired   bool      // a tick fired
	profile   bool      // tea.ColorProfileMsg arrived
	pending   bool      // the probe ended, and CapsMsg waits for the profile
	delivered bool      // CapsMsg was delivered

	frag    byte // the kind of split reply IsReplyFragment is inside, or 0
	fragLen int  // its length so far
	fragEsc bool // its last byte was ESC
}

// timeoutMsg is the probe's deadline.
type timeoutMsg struct{ p *Prober }

// hookMsg is a hook's answer.
type hookMsg struct {
	p     *Prober
	hook  termcap.ProbeHook
	v, ok bool
}

// NewProber returns a Prober.
func NewProber(o ...termcap.Option) *Prober {
	opts := append(slices.Clone(o), termcap.WithoutModeQuery(int(ansi.ModeSynchronizedOutput), int(ansi.ModeUnicodeCore)))
	return &Prober{opts: opts, probe: termcap.NewProbe(opts...)}
}

// Init starts the deadline. The batch follows the first tea.EnvMsg. A
// disabled prober returns nil.
func (p *Prober) Init() tea.Cmd {
	// What deadline a probe of these options started now would have: none
	// when it is disabled.
	t0 := time.Now()
	scratch := termcap.NewProbe(p.opts...)
	scratch.Start(nil, t0)
	deadline, ok := scratch.Timeout()
	if !ok {
		return nil
	}
	p.init, p.t0, p.tick = true, t0, deadline
	return tea.Tick(deadline.Sub(t0), func(time.Time) tea.Msg { return timeoutMsg{p} })
}

// Caps is a copy of what is known now, with the overrides applied.
func (p *Prober) Caps() termcap.Caps { return p.probe.Caps() }

// Restore is the bytes that reset every mode the prober set: mode 2031's
// reset when it subscribed, and "" otherwise.
func (p *Prober) Restore() string { return p.probe.Restore() }

// Quit restores every mode the prober set, then quits.
func (p *Prober) Quit() tea.Cmd {
	if s := p.Restore(); s != "" {
		return tea.Sequence(tea.Raw(s), tea.Quit)
	}
	return tea.Quit
}

// SetTmux records what the program learned from running termcap.TmuxQuery.
func (p *Prober) SetTmux(f termcap.TmuxFacts) { p.probe.SetTmux(f) }

// Update observes msg, and returns what the probe sends next: the batch on
// the first tea.EnvMsg, CapsMsg when the probe ends, mode 2031 once the
// terminal supports it, and ColorSchemeMsg after the probe.
func (p *Prober) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case tea.EnvMsg:
		t0 := p.t0
		if !p.init {
			t0 = time.Now()
		}
		p.probe.Start(termcap.Env(m), t0)
	case tea.ColorProfileMsg:
		p.probe.SetColorProfile(FromColorProfile(m.Profile))
		p.profile = true
	case timeoutMsg:
		if m.p == p {
			p.tick, p.expired = time.Time{}, true
			if deadline, ok := p.probe.Timeout(); ok {
				p.probe.HandleTimeout(deadline)
			}
		}
	case hookMsg:
		if m.p == p {
			p.probe.HandleHook(m.hook, m.v, m.ok)
		}
	}
	if r, ok := reply(msg); ok {
		p.probe.HandleReply(r, time.Now())
	}
	return p.drain()
}

// drain turns every event the probe queued into a command: its writes and
// messages in order, its hooks beside them. It delivers CapsMsg, once,
// when the probe has ended and tea's colour profile has arrived, since tea
// sends that unordered with tea.EnvMsg; the deadline delivers regardless.
func (p *Prober) drain() tea.Cmd {
	var ordered, beside []tea.Cmd
	for e := p.probe.NextEvent(); e.Kind != termcap.ProbeNoEvent; e = p.probe.NextEvent() {
		switch e.Kind {
		case termcap.ProbeWrite:
			ordered = append(ordered, tea.Raw(e.Data))
		case termcap.ProbeRunHook:
			run, hook := e.Run, e.Hook
			beside = append(beside, func() tea.Msg {
				v, ok := run()
				return hookMsg{p: p, hook: hook, v: v, ok: ok}
			})
		case termcap.ProbeDone:
			p.pending = true
		case termcap.ProbeColorScheme:
			ordered = append(ordered, teamsg.Cmd(termcap.ColorSchemeMsg{Dark: e.Dark}))
		}
	}
	if p.pending && !p.delivered {
		c := p.probe.Caps()
		if p.profile || c.TimedOut || p.expired {
			// A deadline that passed while CapsMsg waited for the profile
			// ends the wait, as a timeout.
			c.TimedOut = c.TimedOut || !p.profile
			p.delivered = true
			ordered = append(ordered, teamsg.Cmd(termcap.CapsMsg{Caps: c}))
		}
	}
	// A deadline Init's tick does not cover: the environment came after
	// that tick fired. A program that never ran Init has no deadline.
	if deadline, ok := p.probe.Timeout(); ok && p.init && !deadline.Equal(p.tick) {
		p.tick = deadline
		beside = append(beside, tea.Tick(time.Until(deadline), func(time.Time) tea.Msg { return timeoutMsg{p} }))
	}
	return tea.Batch(append([]tea.Cmd{tea.Sequence(ordered...)}, beside...)...)
}

// Kinds of split reply.
const (
	fragCSI    = 'c' // ends with a final byte, 0x40-0x7e
	fragString = 's' // OSC, DCS or APC: ends with BEL or ST
)

// maxReply is the longest reply the probe parses.
const maxReply = 1024

// IsReplyFragment reports whether msg is part of a reply to the probe that
// arrived split across reads: an unknown event, which is what the reader
// gives up on at its escape timeout, whose bytes begin a reply the probe
// is still waiting for, and each key press after it up to the byte that
// ends that reply, within 1 KiB (docs/decisions/0005-MADR-terminal-capabilities-and-services.md
// A2). An input filter, such as one passed to tea.WithFilter, calls it once
// for each message, in order, to drop the fragments. The prober never
// hides a message itself.
func (p *Prober) IsReplyFragment(msg tea.Msg) bool {
	if p.frag != 0 {
		b, ok := keyByte(msg)
		if !ok || p.fragLen >= maxReply {
			p.frag = 0
			return false
		}
		p.fragLen++
		switch {
		case p.frag == fragCSI && b >= 0x40 && b <= 0x7e,
			p.frag == fragString && b == ansi.BEL,
			p.frag == fragString && p.fragEsc && b == '\\':
			p.frag = 0
		}
		p.fragEsc = b == ansi.ESC
		return true
	}
	if !p.probe.Awaiting() {
		return false
	}
	ev, ok := termevent.Decode(msg)
	if !ok || ev.Kind != termevent.Unknown {
		return false
	}
	if p.frag = fragKind(ev.Raw); p.frag == 0 {
		return false
	}
	p.fragLen, p.fragEsc = len(ev.Raw), strings.HasSuffix(ev.Raw, "\x1b")
	return true
}

// fragKind is the kind of reply raw begins and does not finish, or 0.
func fragKind(raw string) byte {
	switch {
	case strings.HasPrefix(raw, "\x1b[?"), strings.HasPrefix(raw, "\x1b[>"):
		if last := raw[len(raw)-1]; last >= 0x40 && last <= 0x7e {
			return 0
		}
		return fragCSI
	case strings.HasPrefix(raw, "\x1b]"), strings.HasPrefix(raw, "\x1bP>|"), strings.HasPrefix(raw, "\x1b_G"):
		if strings.HasSuffix(raw, "\x07") || strings.HasSuffix(raw, "\x1b\\") {
			return 0
		}
		return fragString
	}
	return 0
}

// keyByte is the byte a key press stands for in a split reply: its text,
// ESC, or BEL as ctrl+g.
func keyByte(msg tea.Msg) (byte, bool) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return 0, false
	}
	switch {
	case len(k.Text) == 1:
		return k.Text[0], true
	case k.Code == tea.KeyEscape && k.Mod == 0:
		return ansi.ESC, true
	case k.Code == 'g' && k.Mod == tea.ModCtrl:
		return ansi.BEL, true
	}
	return 0, false
}
