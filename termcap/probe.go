package termcap

import (
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
)

// DefaultTimeout is how long the probe waits for its sentinel.
const DefaultTimeout = 2 * time.Second

// CapsMsg carries the probe's result to a Bubble Tea program.
// termcaptea's Prober delivers it once: when the DA1 sentinel answers, or
// when the timeout fires first.
type CapsMsg struct{ Caps Caps }

// ColorSchemeMsg reports the terminal's light or dark scheme to a Bubble
// Tea program, each time a DSR 997 report arrives after the probe has
// ended.
type ColorSchemeMsg struct{ Dark bool }

// Probe learns a terminal's capabilities from one batch of queries that
// ends with DA1. It does no I/O, reads no clock and starts no goroutine:
// the host gives it the environment, each reply and the time, and drains
// the events it returns. It is not safe for concurrent use: one goroutine,
// the TUI's loop or the CLI's, drives it.
//
// A host drives it so:
//
//  1. Start, with the environment and the time;
//  2. drain NextEvent until it returns ProbeNoEvent: write each ProbeWrite's
//     Data to the terminal, in order; run each ProbeRunHook's Run where it
//     likes, and pass its answer to HandleHook; ProbeDone carries the
//     facts; and ignore a kind it does not know, since a later version may
//     add one;
//  3. give each reply to HandleReply as it arrives, whole, in order, and
//     drain again;
//  4. while Timeout reports a deadline, call HandleTimeout once it has
//     passed, and drain again.
//
// termcaptea's Prober is that host for a Bubble Tea program. A line-mode
// CLI is its own host, and takes on what Bubble Tea does for a TUI: it
// probes only from raw mode, and never without input, since a reply
// nobody reads reaches the user's input line; it reads every reply the
// batch asks for; and, reading with termkey, it does not Flush while
// Awaiting, so a reply split across reads is joined rather than given up
// in halves (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
// A2.1, A2.3 and A2.4).
//
// By default it subscribes to mode 2031 once the terminal supports it, so
// the host must restore the terminal before it exits, by writing Restore.
type Probe struct {
	timeout      time.Duration
	added        []Query
	ungated      bool
	overrides    []func(*Caps)
	noScheme     bool
	noBackground bool
	disabled     bool
	noGated      bool     // WithoutGatedQueries
	dropped      []string // WithoutQuery
	hostModes    []int    // WithoutModeQuery

	goos           string // runtime.GOOS, or WithGOOS
	appearanceEnv  string
	appearanceHook func() (dark, ok bool)
	consoleHost    func() (classic, ok bool)

	caps     Caps
	sent     []*probe // the batch's queries, in order, once sent
	started  bool     // Start ran
	done     bool     // ProbeDone was queued
	dsr997   bool     // Dark came from a DSR 997 report
	set2031  bool     // the probe subscribed to mode 2031
	deadline time.Time
	events   []ProbeEvent // queued, oldest first
}

// ProbeEventKind is what a ProbeEvent asks of the host. A host ignores a
// kind it does not know: a later version adds kinds, and one that needs
// the host to act comes only after an option asks for it.
type ProbeEventKind uint8

const (
	// ProbeNoEvent is the zero value: nothing to do.
	ProbeNoEvent ProbeEventKind = iota
	// ProbeWrite asks the host to write Data to the terminal, in order.
	ProbeWrite
	// ProbeRunHook asks the host to run Run, then pass its answer to
	// HandleHook with Hook. It comes only for a hook an option set.
	ProbeRunHook
	// ProbeDone says the probe ended; Caps holds the facts. It comes once.
	ProbeDone
	// ProbeColorScheme reports a DSR 997 after the probe; Dark says which.
	// A host may drop it.
	ProbeColorScheme
)

var probeEventKindNames = enum.Names[ProbeEventKind]{
	Pkg: pkgName, Type: "ProbeEventKind", Unknown: ErrUnknownName,
	Tokens: []string{tokenNone, "write", "run-hook", "done", tokenColorScheme},
}

// String is the kind's name.
func (k ProbeEventKind) String() string { return probeEventKindNames.String(k) }

// MarshalText is the kind's name.
func (k ProbeEventKind) MarshalText() ([]byte, error) { return probeEventKindNames.Marshal(k) }

// UnmarshalText reads a name MarshalText wrote.
func (k *ProbeEventKind) UnmarshalText(b []byte) error { return probeEventKindNames.Unmarshal(b, k) }

// ProbeEvent is one thing the probe asks of its host. Only the fields its
// Kind names are set. Fields are added, never changed, so build one with a
// keyed literal.
type ProbeEvent struct {
	Kind ProbeEventKind
	Data string              // ProbeWrite: the bytes
	Hook ProbeHook           // ProbeRunHook: which hook, for HandleHook
	Run  func() (v, ok bool) // ProbeRunHook: the program's hook
	Caps Caps                // ProbeDone: Caps when NextEvent returned it
	Dark bool                // ProbeColorScheme
}

// ProbeHook names a program's hook, which the probe hands to its host to
// run, since a hook may start a process.
type ProbeHook uint8

const (
	// HookAppearance is WithAppearanceHook's: whether the desktop is dark.
	HookAppearance ProbeHook = iota + 1
	// HookConsoleHost is WithConsoleHost's: whether the Windows console is
	// the classic console host.
	HookConsoleHost
)

var probeHookNames = enum.Names[ProbeHook]{
	Pkg: pkgName, Type: "ProbeHook", Unknown: ErrUnknownName,
	Tokens: []string{tokenNone, "appearance", "console-host"},
}

// String is the hook's name.
func (h ProbeHook) String() string { return probeHookNames.String(h) }

// MarshalText is the hook's name.
func (h ProbeHook) MarshalText() ([]byte, error) { return probeHookNames.Marshal(h) }

// UnmarshalText reads a name MarshalText wrote.
func (h *ProbeHook) UnmarshalText(b []byte) error { return probeHookNames.Unmarshal(b, h) }

// NewProbe returns a Probe.
func NewProbe(o ...Option) *Probe {
	p := &Probe{timeout: DefaultTimeout, goos: runtime.GOOS}
	for _, f := range o {
		f.apply(p)
	}
	for _, f := range p.overrides {
		f(&p.caps)
	}
	return p
}

// Start reads the environment's facts and queues the batch, once: one
// ProbeWrite of every query and DA1, then a ProbeRunHook for each hook an
// option set. The deadline runs from now. A probe WithoutQuery gave a
// name no query has, a disabled probe, and one in a JetBrains terminal,
// which paints a query as text, write nothing and end at once.
func (p *Probe) Start(env Env, now time.Time) {
	if p.started {
		return
	}
	p.started = true
	p.caps.setEnv(env, p.goos, p.appearanceEnv)
	// A name no query has is the program's mistake: send nothing rather
	// than a batch it did not mean, and say why on each query fact.
	if p.unknownQuery() {
		for _, f := range queryFacts(&p.caps) {
			f.SetReason(Unknown, NotQueried, ReasonUnknownQuery)
		}
		p.queueHooks()
		p.finish()
		return
	}
	// JetBrains paints a query as text: send nothing, and say why each
	// fact a query would have set is unknown (MADR A1, Q7), whether or not
	// the probe is disabled (docs/decisions/0014-PLAN-component-native-forms.md
	// D9).
	if p.caps.EnvBrand.Value == BrandJetBrains {
		for _, f := range queryFacts(&p.caps) {
			f.SetReason(Unknown, NotQueried, ReasonJetBrainsPaints)
		}
		p.queueHooks()
		p.finish()
		return
	}
	if p.disabled {
		p.queueHooks()
		p.finish()
		return
	}
	// An editor's terminal answers for the editor, so the gated queries,
	// which would describe the user's terminal, are skipped there.
	inEditor := p.caps.Editor.Value != EditorNone
	heuristic := p.ungated || gatedAllowed(env, p.goos) && !inEditor
	if inEditor && !heuristic && !p.noGated {
		p.caps.DesktopNotify.SetReason(Unknown, NotQueried, ReasonEditorTerminal)
		p.caps.KittyGraphics.SetReason(Unknown, NotQueried, ReasonEditorTerminal)
	}
	gated := heuristic && !p.noGated
	var b strings.Builder
	for _, q := range p.batch(env) {
		if q.Gated && !gated {
			continue
		}
		s := q.Seq(env, p.caps.Mux.Value)
		if s == "" {
			continue
		}
		b.WriteString(s)
		p.sent = append(p.sent, q)
	}
	b.WriteString(ansi.RequestPrimaryDeviceAttributes)
	p.deadline = now.Add(p.timeout)
	p.queue(ProbeEvent{Kind: ProbeWrite, Data: b.String()})
	p.queueHooks()
}

// queryFacts are the support facts the batch's queries set.
func queryFacts(c *Caps) []*Fact[Support] {
	return []*Fact[Support]{
		&c.KittyKeyboard, &c.ColorSchemeReports, &c.InBandResize, &c.FocusEvents,
		&c.DesktopNotify, &c.KittyGraphics, &c.Sixel,
	}
}

// queueHooks hands each hook an option set to the host, once; the console
// host's only on Windows.
func (p *Probe) queueHooks() {
	if f := p.appearanceHook; f != nil {
		p.queue(ProbeEvent{Kind: ProbeRunHook, Hook: HookAppearance, Run: f})
	}
	if f := p.consoleHost; f != nil && p.goos == goosWindows {
		p.queue(ProbeEvent{Kind: ProbeRunHook, Hook: HookConsoleHost, Run: f})
	}
}

// HandleReply reads one reply, in the order it arrived. A reply over
// 1 KiB is not parsed, and the query it would answer is marked
// ReasonReplyTooLong. Every query sent is offered the rest. DA1 is the
// sentinel: it ends the probe, and each query still unanswered then is
// unsupported. A mode WithoutModeQuery names, and modes 2026 and 2027,
// are read whoever asked for them.
func (p *Probe) HandleReply(r Reply, _ time.Time) {
	switch r.Kind {
	case ReplyMode:
		if r.Mode == int(ansi.ModeSynchronizedOutput) || r.Mode == int(ansi.ModeUnicodeCore) || slices.Contains(p.hostModes, r.Mode) {
			if f := modeFact(&p.caps, r.Mode); f != nil {
				f.Set(supportIf(r.Setting.supported()), Queried)
			}
		}
	case ReplyColor:
		if r.OSC == 11 {
			p.caps.Background = r.Color
			if !p.dsr997 {
				p.caps.Dark.Set(isDark(r.Color), Queried)
			}
		}
	}

	if len(r.Raw) > maxReply {
		p.tooLong(r.Raw)
	} else {
		for _, q := range p.sent {
			if q.Parse(r, &p.caps) {
				q.answered = true
			}
		}
	}

	switch r.Kind {
	case ReplyColorScheme:
		p.dsr997 = true
		p.caps.Dark.Set(r.Dark, Queried)
		if p.done {
			p.queue(ProbeEvent{Kind: ProbeColorScheme, Dark: r.Dark})
			// The exact background, so a theme that follows the scheme
			// gets the colour.
			if !p.noBackground && !p.drops("background") {
				p.queue(ProbeEvent{Kind: ProbeWrite, Data: ansi.RequestBackgroundColor})
			}
		}
	case ReplyPrimaryAttributes:
		p.caps.Attributes = slices.Clone(r.Params)
		p.caps.Sixel.Set(supportIf(slices.Contains(r.Params, 4)), Queried)
		appleFingerprint(&p.caps)
		p.sentinel()
	}

	if !p.disabled && !p.noScheme && !p.set2031 && p.caps.ColorSchemeReports.Value == Supported {
		p.set2031 = true
		p.queue(ProbeEvent{Kind: ProbeWrite, Data: ansi.SetModeLightDark})
	}
}

// tooLong handles a reply over maxReply: it is not parsed, and the
// built-in query it would answer is marked with the reason.
func (p *Probe) tooLong(raw string) {
	if strings.HasPrefix(raw, "\x9d") {
		raw = "\x1b]" + raw[1:] // the 8-bit OSC
	}
	for _, q := range p.sent {
		if q.prefix == "" || !strings.HasPrefix(raw, q.prefix) {
			continue
		}
		q.answered = true
		if q.fact != nil {
			q.fact(&p.caps).SetReason(Unknown, Queried, ReasonReplyTooLong)
		}
	}
}

// sentinel ends the probe when DA1 answers the batch: every query sent
// before it with no reply is unsupported. A DA1 nobody asked for, or a
// second one, changes nothing more.
func (p *Probe) sentinel() {
	if len(p.sent) == 0 || p.caps.Complete {
		return
	}
	p.caps.Complete = true
	for _, q := range p.sent {
		if !q.answered && q.silent != nil {
			q.silent(&p.caps)
		}
	}
	p.finish()
}

// finish queues ProbeDone, once.
func (p *Probe) finish() {
	if p.done {
		return
	}
	p.done = true
	p.queue(ProbeEvent{Kind: ProbeDone})
}

// HandleTimeout ends the probe with TimedOut when now is at or past the
// deadline Timeout gave, and the probe has not ended; otherwise it does
// nothing.
func (p *Probe) HandleTimeout(now time.Time) {
	deadline, ok := p.Timeout()
	if !ok || now.Before(deadline) {
		return
	}
	p.caps.TimedOut = true
	p.finish()
}

// HandleHook takes a hook's answer, from the Run of a ProbeRunHook event:
// the appearance hook's sets Dark, as a heuristic with
// ReasonDesktopAppearance; the console host's sets LegacyConsole, as
// queried. ok false, the hook's "cannot tell", changes nothing.
func (p *Probe) HandleHook(h ProbeHook, v, ok bool) {
	if !ok {
		return
	}
	switch h {
	case HookAppearance:
		p.caps.Dark.SetReason(v, Heuristic, ReasonDesktopAppearance)
	case HookConsoleHost:
		p.caps.LegacyConsole.Set(v, Queried)
	}
}

// SetColorProfile records the host's own detection of the colour profile,
// which termcap does not detect.
func (p *Probe) SetColorProfile(c ColorProfile) { p.caps.Profile = c }

// SetTmux records what the program learned from running TmuxQuery.
func (p *Probe) SetTmux(f TmuxFacts) { p.caps.Tmux = f }

// Timeout is the deadline by which HandleTimeout is due, and true, from
// Start until ProbeDone; then the zero time and false. A disabled probe
// has none.
func (p *Probe) Timeout() (deadline time.Time, ok bool) {
	if p.done || p.deadline.IsZero() {
		return time.Time{}, false
	}
	return p.deadline, true
}

// NextEvent is the oldest event the host has not taken, or the zero
// ProbeEvent, of kind ProbeNoEvent, when none is left.
func (p *Probe) NextEvent() ProbeEvent {
	if len(p.events) == 0 {
		return ProbeEvent{}
	}
	e := p.events[0]
	p.events[0] = ProbeEvent{}
	p.events = p.events[1:]
	if e.Kind == ProbeDone {
		e.Caps = p.Caps()
	}
	return e
}

// queue adds an event for the host.
func (p *Probe) queue(e ProbeEvent) { p.events = append(p.events, e) }

// Awaiting reports whether a reply may still come: the batch went out, and
// the sentinel or a query is unanswered.
func (p *Probe) Awaiting() bool {
	if len(p.sent) == 0 {
		return false
	}
	if !p.caps.Complete {
		return true
	}
	return slices.ContainsFunc(p.sent, func(q *probe) bool { return !q.answered })
}

// Caps is a copy of what is known now, with the overrides applied.
func (p *Probe) Caps() Caps {
	c := p.caps
	c.Attributes = slices.Clone(c.Attributes)
	for _, f := range p.overrides {
		f(&c)
	}
	return c
}

// Restore is the bytes that reset every mode the probe set: mode 2031's
// reset when it subscribed, and "" otherwise.
func (p *Probe) Restore() string {
	if p.set2031 {
		return ansi.ResetModeLightDark
	}
	return ""
}
