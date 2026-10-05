package termcap

import (
	"fmt"
	"image/color"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/internal/termevent"
)

// DefaultTimeout is how long the probe waits for its sentinel.
const DefaultTimeout = 2 * time.Second

// CapsMsg carries the probe's result. A Prober delivers it once: when the
// DA1 sentinel answers, or when the timeout fires first.
type CapsMsg struct{ Caps Caps }

// ColorSchemeMsg reports the terminal's light or dark scheme, each time a
// DSR 997 report arrives after the probe has ended.
type ColorSchemeMsg struct{ Dark bool }

// Reply is a message as a query sees it: the tea message, and for any
// sequence ultraviolet did not decode, its raw bytes.
type Reply struct {
	Msg tea.Msg
	Raw string // an unknown CSI, OSC, DCS or APC reply; "" otherwise
}

// Query is one probe. Programs and later records add their own with
// WithQuery; each is sent after the built-in queries and before the
// sentinel.
type Query struct {
	Name  string
	Seq   func(env Env, mux Mux) string // the bytes to send; "" skips it
	Gated bool                          // only behind the heuristic
	Parse func(r Reply, c *Caps) bool   // true when r answered it
}

// Option configures a Prober.
type Option func(*Prober)

// WithTimeout sets how long the probe waits for its sentinel. The default
// is DefaultTimeout.
func WithTimeout(d time.Duration) Option { return func(p *Prober) { p.timeout = d } }

// WithQuery adds a query to the batch, after the built-in queries and
// before the sentinel.
func WithQuery(q Query) Option { return func(p *Prober) { p.added = append(p.added, q) } }

// WithoutHeuristic sends the gated queries to every terminal, even those
// the heuristic would spare: Apple Terminal, and SSH peers the environment
// does not name.
func WithoutHeuristic() Option { return func(p *Prober) { p.ungated = true } }

// WithOverride sets facts the program knows better, from a flag or a file.
// f runs on the facts the prober starts from, and again on every Caps it
// hands out. f should set each fact with origin Override, so that no reply
// replaces it.
func WithOverride(f func(*Caps)) Option {
	return func(p *Prober) { p.overrides = append(p.overrides, f) }
}

// WithoutColorSchemeUpdates never subscribes to mode 2031, so the terminal
// sends no DSR 997 reports after the probe, and Restore is always empty.
func WithoutColorSchemeUpdates() Option { return func(p *Prober) { p.noScheme = true } }

// WithoutBackgroundRequest leaves the OSC 11 background query out of the
// batch, and tea.RequestBackgroundColor out of the follow-up to each DSR 997
// report.
func WithoutBackgroundRequest() Option { return func(p *Prober) { p.noBackground = true } }

// WithDisabled sends nothing. Init returns nil, and the first tea.EnvMsg
// delivers a CapsMsg holding only the environment's facts. A program run
// without input, tea.WithInput(nil), must not probe: tea skips its own
// queries then, but the replies to the prober's would reach the shell.
func WithDisabled() Option { return func(p *Prober) { p.disabled = true } }

// WithGOOS reads the terminal's identity for goos instead of
// runtime.GOOS: for a test, or for a wish server, whose own operating
// system is not the SSH client's.
func WithGOOS(goos string) Option { return func(p *Prober) { p.goos = goos } }

// WithAppearanceEnv names an environment variable the program sets to
// "dark" or "light", such as MYAPP_APPEARANCE. It is also read with an LC_
// prefix, because a default sshd forwards LC_* variables. It outranks
// COLORFGBG and the desktop hook, and a terminal's reply outranks it.
func WithAppearanceEnv(name string) Option { return func(p *Prober) { p.appearanceEnv = name } }

// WithAppearanceHook asks the desktop whether it is dark, through f: the
// macOS appearance, the XDG portal or the Windows registry, which need a
// process or an OS binding the library does not take. f runs once, as a
// command, after the first tea.EnvMsg; it reports ok false when it cannot
// tell. Its answer outranks COLORFGBG only.
func WithAppearanceHook(f func() (dark, ok bool)) Option {
	return func(p *Prober) { p.appearanceHook = f }
}

// WithConsoleHost asks, on Windows, whether the console is the classic
// console host rather than ConPTY, through f, which a later Windows record
// supplies. f runs once, as a command, after the first tea.EnvMsg; it
// reports ok false when it cannot tell.
func WithConsoleHost(f func() (classic, ok bool)) Option {
	return func(p *Prober) { p.consoleHost = f }
}

// Prober learns the terminal's capabilities from one batch of queries that
// ends with DA1. A program keeps one beside its workspace, returns its Init
// from the program's Init, and passes every message to its Update. It
// never hides a message from the program.
//
// The batch goes out on the first tea.EnvMsg, because the gated queries and
// tmux passthrough depend on the environment, and tea sends that message
// unordered with Init's command
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md A3).
//
// The prober never sends what tea asks for itself (DECRQM 2026 and 2027,
// XTGETTCAP), because tea acts on those replies. It records them from tea's
// own messages instead.
//
// By default it subscribes to mode 2031 once the terminal supports it, so
// the program must restore the terminal before it exits: quit with Quit,
// or send Restore's bytes first.
type Prober struct {
	timeout      time.Duration
	added        []Query
	ungated      bool
	overrides    []func(*Caps)
	noScheme     bool
	noBackground bool
	disabled     bool

	goos           string // runtime.GOOS, or WithGOOS
	appearanceEnv  string
	appearanceHook func() (dark, ok bool)
	consoleHost    func() (classic, ok bool)

	caps    Caps
	sent    []*probe // the batch's queries, in order, once sent
	started bool     // the first tea.EnvMsg arrived
	done    bool     // CapsMsg was delivered
	pending bool     // CapsMsg is due, and waits for the colour profile
	profile bool     // tea.ColorProfileMsg arrived
	dsr997  bool     // Dark came from a DSR 997 report
	set2031 bool     // the prober subscribed to mode 2031

	frag    byte // the kind of split reply IsReplyFragment is inside, or 0
	fragLen int  // its length so far
	fragEsc bool // its last byte was ESC
}

// probe is a query in the batch.
type probe struct {
	Query
	silent   func(c *Caps) // what no reply before the sentinel means; nil for nothing
	answered bool

	// prefix starts the raw reply this query gets, and fact is the fact a
	// reply over maxReply marks; both only for the built-in queries whose
	// replies ultraviolet does not decode.
	prefix string
	fact   func(*Caps) *Fact[Support]
}

// maxReply is the longest raw reply the prober parses.
const maxReply = 1024

// timeoutMsg is the deadline Init starts.
type timeoutMsg struct{ p *Prober }

// appearanceMsg is the appearance hook's answer.
type appearanceMsg struct {
	p        *Prober
	dark, ok bool
}

// consoleMsg is the console-host hook's answer.
type consoleMsg struct {
	p           *Prober
	classic, ok bool
}

// New returns a Prober.
func New(o ...Option) *Prober {
	p := &Prober{timeout: DefaultTimeout, goos: runtime.GOOS}
	for _, f := range o {
		f(p)
	}
	for _, f := range p.overrides {
		f(&p.caps)
	}
	return p
}

// Init starts the deadline. The batch follows the first tea.EnvMsg.
func (p *Prober) Init() tea.Cmd {
	if p.disabled {
		return nil
	}
	return tea.Tick(p.timeout, func(time.Time) tea.Msg { return timeoutMsg{p} })
}

// Caps is a copy of what is known now, with the overrides applied.
func (p *Prober) Caps() Caps {
	c := p.caps
	c.Attributes = slices.Clone(c.Attributes)
	for _, f := range p.overrides {
		f(&c)
	}
	return c
}

// Restore is the bytes that reset every mode the prober set: mode 2031's
// reset when it subscribed, and "" otherwise.
func (p *Prober) Restore() string {
	if p.set2031 {
		return ansi.ResetModeLightDark
	}
	return ""
}

// Quit restores every mode the prober set, then quits.
func (p *Prober) Quit() tea.Cmd {
	if s := p.Restore(); s != "" {
		return tea.Sequence(tea.Raw(s), tea.Quit)
	}
	return tea.Quit
}

// Update observes msg, and returns what the probe sends next: the batch on
// the first tea.EnvMsg, CapsMsg when the probe ends, mode 2031 once the
// terminal supports it, and ColorSchemeMsg after the probe.
func (p *Prober) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	switch m := msg.(type) {
	case tea.EnvMsg:
		cmds = append(cmds, p.start(Env(m)))
	case timeoutMsg:
		if m.p == p && !p.done {
			p.caps.TimedOut = true
			cmds = append(cmds, p.deliver())
		}
	case appearanceMsg:
		if m.p == p && m.ok {
			p.caps.Dark.SetReason(m.dark, Heuristic, ReasonDesktopAppearance)
		}
	case consoleMsg:
		if m.p == p && m.ok {
			p.caps.LegacyConsole.Set(m.classic, Queried)
		}
	case tea.ColorProfileMsg:
		p.caps.Profile = m.Profile
		p.profile = true
		if p.pending {
			cmds = append(cmds, p.deliver())
		}
	case tea.ModeReportMsg:
		switch m.Mode {
		case ansi.ModeSynchronizedOutput:
			p.caps.SyncOutput.Set(support(m.Value), Queried)
		case ansi.ModeUnicodeCore:
			p.caps.GraphemeWidth.Set(support(m.Value), Queried)
		}
	case tea.BackgroundColorMsg:
		p.caps.Background = m.Color
		if !p.dsr997 {
			p.caps.Dark.Set(m.IsDark(), Queried)
		}
	}

	ev, decoded := termevent.Decode(msg)
	r := Reply{Msg: msg}
	if decoded && ev.Kind == termevent.Unknown {
		r.Raw = ev.Raw
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

	if decoded {
		switch ev.Kind {
		case termevent.ColorScheme:
			p.dsr997 = true
			p.caps.Dark.Set(ev.Dark, Queried)
			if p.done {
				cmds = append(cmds, p.schemeChanged(ev.Dark))
			}
		case termevent.DeviceAttributes:
			p.caps.Attributes = ev.Attrs
			p.caps.Sixel.Set(supportIf(slices.Contains(ev.Attrs, 4)), Queried)
			appleFingerprint(&p.caps)
			cmds = append(cmds, p.sentinel())
		}
	}

	if !p.disabled && !p.noScheme && !p.set2031 && p.caps.ColorSchemeReports.Value == Supported {
		p.set2031 = true
		cmds = append(cmds, tea.Raw(ansi.SetModeLightDark))
	}
	return tea.Batch(cmds...)
}

// start reads the environment's facts and sends the batch, once.
func (p *Prober) start(env Env) tea.Cmd {
	if p.started {
		return nil
	}
	p.started = true
	p.caps.setEnv(env, p.goos, p.appearanceEnv)
	hooks := p.hooks()
	if p.disabled {
		return tea.Batch(append(hooks, p.deliver())...)
	}
	// JetBrains paints a query as text: send nothing, and say why each
	// fact a query would have set is unknown (MADR A1, Q7).
	if p.caps.EnvBrand.Value == BrandJetBrains {
		for _, f := range queryFacts(&p.caps) {
			f.SetReason(Unknown, NotQueried, ReasonJetBrainsPaints)
		}
		return tea.Batch(append(hooks, p.deliver())...)
	}
	// An editor's terminal answers for the editor, so the gated queries,
	// which would describe the user's terminal, are skipped there.
	inEditor := p.caps.Editor.Value != EditorNone
	gated := p.ungated || gatedAllowed(env) && !inEditor
	if inEditor && !gated {
		p.caps.DesktopNotify.SetReason(Unknown, NotQueried, ReasonEditorTerminal)
		p.caps.KittyGraphics.SetReason(Unknown, NotQueried, ReasonEditorTerminal)
	}
	var b strings.Builder
	for _, q := range p.batch() {
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
	return tea.Batch(append(hooks, tea.Raw(b.String()))...)
}

// queryFacts are the support facts the batch's queries set.
func queryFacts(c *Caps) []*Fact[Support] {
	return []*Fact[Support]{
		&c.KittyKeyboard, &c.ColorSchemeReports, &c.InBandResize, &c.FocusEvents,
		&c.DesktopNotify, &c.KittyGraphics, &c.Sixel,
	}
}

// tooLong handles a raw reply over maxReply: it is not parsed, and the
// built-in query it would answer is marked with the reason.
func (p *Prober) tooLong(raw string) {
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

// Kinds of split reply.
const (
	fragCSI    = 'c' // ends with a final byte, 0x40-0x7e
	fragString = 's' // OSC, DCS or APC: ends with BEL or ST
)

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
	if !p.awaiting() {
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

// awaiting reports whether the probe may still get a reply: it sent its
// batch, and the sentinel or a query is unanswered.
func (p *Prober) awaiting() bool {
	if len(p.sent) == 0 {
		return false
	}
	if !p.caps.Complete {
		return true
	}
	return slices.ContainsFunc(p.sent, func(q *probe) bool { return !q.answered })
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

// SetTmux records what the program learned from running TmuxQuery.
func (p *Prober) SetTmux(f TmuxFacts) { p.caps.Tmux = f }

// hooks are the commands that run the program's appearance and console
// hooks, once.
func (p *Prober) hooks() []tea.Cmd {
	var cmds []tea.Cmd
	if f := p.appearanceHook; f != nil {
		cmds = append(cmds, func() tea.Msg {
			dark, ok := f()
			return appearanceMsg{p: p, dark: dark, ok: ok}
		})
	}
	if f := p.consoleHost; f != nil && p.goos == goosWindows {
		cmds = append(cmds, func() tea.Msg {
			classic, ok := f()
			return consoleMsg{p: p, classic: classic, ok: ok}
		})
	}
	return cmds
}

// sentinel ends the probe when DA1 answers the batch: every query sent
// before it with no reply is unsupported. A DA1 nobody asked for, or a
// second one, changes nothing more.
func (p *Prober) sentinel() tea.Cmd {
	if len(p.sent) == 0 || p.caps.Complete {
		return nil
	}
	p.caps.Complete = true
	for _, q := range p.sent {
		if !q.answered && q.silent != nil {
			q.silent(&p.caps)
		}
	}
	return p.deliver()
}

// deliver sends CapsMsg, once, when tea's colour profile has arrived: tea
// sends it unordered with tea.EnvMsg. The deadline delivers regardless.
func (p *Prober) deliver() tea.Cmd {
	if p.done {
		return nil
	}
	if !p.profile && !p.caps.TimedOut {
		p.pending = true
		return nil
	}
	p.done = true
	c := p.Caps()
	return func() tea.Msg { return CapsMsg{Caps: c} }
}

// schemeChanged reports a DSR 997 after the probe, and asks for the exact
// background, so a theme that follows it gets the colour.
func (p *Prober) schemeChanged(dark bool) tea.Cmd {
	msg := func() tea.Msg { return ColorSchemeMsg{Dark: dark} }
	if p.noBackground {
		return msg
	}
	return tea.Sequence(msg, tea.RequestBackgroundColor)
}

// batch is every query in the batch's order, before the sentinel: the
// Kitty keyboard request, the DECRQM set and DSR 996, the OSC 11 background
// query unless declined, the gated queries, then each added query. The
// background query is written here rather than sent as
// tea.RequestBackgroundColor, which tea.Batch could run after the sentinel
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md A3).
func (p *Prober) batch() []*probe {
	unsupported := func(f func(*Caps) *Fact[Support]) func(*Caps) {
		return func(c *Caps) { f(c).Set(Unsupported, Queried) }
	}
	qs := []*probe{
		{Name: "kitty-keyboard", Seq: fixed(ansi.RequestKittyKeyboard), Parse: parseKittyKeyboard,
			silent: unsupported(func(c *Caps) *Fact[Support] { return &c.KittyKeyboard })},
		modeProbe("color-scheme-reports", ansi.ModeLightDark, ansi.RequestModeLightDark,
			func(c *Caps) *Fact[Support] { return &c.ColorSchemeReports }),
		modeProbe("in-band-resize", ansi.ModeInBandResize, ansi.RequestModeInBandResize,
			func(c *Caps) *Fact[Support] { return &c.InBandResize }),
		modeProbe("focus-events", ansi.ModeFocusEvent, ansi.RequestModeFocusEvent,
			func(c *Caps) *Fact[Support] { return &c.FocusEvents }),
		{Name: "color-scheme", Seq: fixed(ansi.RequestLightDarkReport), Parse: parseColorScheme},
		{Name: "background", Seq: p.background, Parse: parseBackground},
		{Name: "secondary-attributes", Seq: fixed(ansi.RequestSecondaryDeviceAttributes), Parse: parseDA2},
		{Name: "terminal-version", Seq: fixed(ansi.RequestNameVersion), Gated: true, Parse: parseVersion},
		{Name: "desktop-notify", Seq: wrapped(osc99Query), Gated: true, Parse: parseOSC99,
			silent: unsupported(func(c *Caps) *Fact[Support] { return &c.DesktopNotify }),
			prefix: "\x1b]99;", fact: func(c *Caps) *Fact[Support] { return &c.DesktopNotify }},
		{Name: "kitty-graphics", Seq: wrapped(kittyGraphicsQuery), Gated: true, Parse: parseKittyGraphics,
			silent: unsupported(func(c *Caps) *Fact[Support] { return &c.KittyGraphics })},
		{Name: "foreground", Seq: fixed(ansi.RequestForegroundColor), Gated: true, Parse: parseForeground},
		{Name: "palette", Seq: fixed(paletteQuery), Gated: true, Parse: parsePalette, prefix: "\x1b]4;"},
	}
	for _, q := range p.added {
		qs = append(qs, &probe{Query: q})
	}
	return qs
}

// background is the OSC 11 query, unless WithoutBackgroundRequest.
func (p *Prober) background(Env, Mux) string {
	if p.noBackground {
		return ""
	}
	return ansi.RequestBackgroundColor
}

// parseBackground answers OSC 11. Update records the colour itself.
func parseBackground(r Reply, _ *Caps) bool {
	_, ok := r.Msg.(tea.BackgroundColorMsg)
	return ok
}

// fixed is a query's bytes that do not depend on the environment.
func fixed(seq string) func(Env, Mux) string { return func(Env, Mux) string { return seq } }

// wrapped is a query only the outer terminal can answer: inside tmux it
// goes out through tmux's passthrough, which needs allow-passthrough.
func wrapped(seq string) func(Env, Mux) string {
	return func(_ Env, m Mux) string {
		if m == Tmux {
			return ansi.TmuxPassthrough(seq)
		}
		return seq
	}
}

// modeProbe asks with DECRQM whether the terminal knows a mode.
func modeProbe(name string, mode ansi.DECMode, seq string, fact func(*Caps) *Fact[Support]) *probe {
	return &probe{
		Name: name, Seq: fixed(seq), Parse: func(r Reply, c *Caps) bool {
			m, ok := r.Msg.(tea.ModeReportMsg)
			if !ok || m.Mode != mode {
				return false
			}
			fact(c).Set(support(m.Value), Queried)
			return true
		},
		silent: func(c *Caps) { fact(c).Set(Unsupported, Queried) },
	}
}

// support reads a DECRQM reply: a mode the terminal does not recognise, or
// holds permanently off, is unsupported.
func support(v ansi.ModeSetting) Support {
	return supportIf(!v.IsNotRecognized() && !v.IsPermanentlyReset())
}

func supportIf(ok bool) Support {
	if ok {
		return Supported
	}
	return Unsupported
}

func parseKittyKeyboard(r Reply, c *Caps) bool {
	m, ok := r.Msg.(tea.KeyboardEnhancementsMsg)
	if !ok {
		return false
	}
	c.KittyKeyboard.Set(Supported, Queried)
	c.KeyboardFlags = m.Flags
	return true
}

// parseColorScheme answers DSR 996. Update records the scheme itself.
func parseColorScheme(r Reply, _ *Caps) bool {
	e, ok := termevent.Decode(r.Msg)
	return ok && e.Kind == termevent.ColorScheme
}

func parseVersion(r Reply, c *Caps) bool {
	m, ok := r.Msg.(tea.TerminalVersionMsg)
	if !ok {
		return false
	}
	c.Terminal.Set(m.Name, Queried)
	if strings.HasPrefix(strings.ToLower(m.Name), "tmux") {
		c.Mux.Set(Tmux, Queried)
	}
	if b := brandFromVersion(m.Name); b != BrandUnknown {
		c.Brand.Set(b, Queried)
	}
	return true
}

// parseDA2 reads the secondary device attributes, which name Alacritty's
// version and, with DA1, Apple Terminal over SSH.
func parseDA2(r Reply, c *Caps) bool {
	e, ok := termevent.Decode(r.Msg)
	if !ok || e.Kind != termevent.SecondaryDeviceAttributes {
		return false
	}
	c.SecondaryAttributes = e.Attrs
	appleFingerprint(c)
	return true
}

// appleFingerprint names Apple Terminal from DA1 1;2 with DA2 1;95;0,
// which over SSH is the only sign of it.
func appleFingerprint(c *Caps) {
	if slices.Equal(c.Attributes, []int{1, 2}) && slices.Equal(c.SecondaryAttributes, []int{1, 95, 0}) {
		c.Brand.Set(BrandAppleTerminal, Queried)
	}
}

func parseForeground(r Reply, c *Caps) bool {
	m, ok := r.Msg.(tea.ForegroundColorMsg)
	if !ok {
		return false
	}
	c.Foreground = m.Color
	return true
}

// paletteQuery asks for the 16 ANSI palette colours (OSC 4), so a later
// theme record can build a theme from them.
var paletteQuery = func() string {
	var b strings.Builder
	for i := range 16 {
		fmt.Fprintf(&b, "\x1b]4;%d;?\x07", i)
	}
	return b.String()
}()

// parsePalette reads one OSC 4 reply, OSC 4 ; index ; colour ST, which
// ultraviolet does not decode.
func parsePalette(r Reply, c *Caps) bool {
	body, ok := strings.CutPrefix(r.Raw, "\x1b]4;")
	if !ok {
		return false
	}
	body = strings.TrimSuffix(strings.TrimSuffix(body, "\x07"), "\x1b\\")
	idx, spec, ok := strings.Cut(body, ";")
	i, err := strconv.Atoi(idx)
	if !ok || err != nil || i < 0 || i >= len(c.Palette) {
		return false
	}
	col, ok := parseXColor(spec)
	if !ok {
		return false
	}
	c.Palette[i] = col
	c.PaletteKnown = !slices.Contains(c.Palette[:], nil)
	return true
}

// parseXColor reads an X11 colour, rgb:R/G/B with one to four hex digits
// per component, or #rrggbb.
func parseXColor(s string) (color.Color, bool) {
	if h, ok := strings.CutPrefix(s, "#"); ok && len(h) == 6 {
		var r, g, b uint8
		if _, err := fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b); err != nil {
			return nil, false
		}
		return color.RGBA{R: r, G: g, B: b, A: 0xff}, true
	}
	spec, ok := strings.CutPrefix(s, "rgb:")
	if !ok {
		return nil, false
	}
	parts := strings.Split(spec, "/")
	if len(parts) != 3 {
		return nil, false
	}
	var rgb [3]uint8
	for i, p := range parts {
		if p == "" || len(p) > 4 {
			return nil, false
		}
		v, err := strconv.ParseUint(p, 16, 16)
		if err != nil {
			return nil, false
		}
		// Scale n hex digits to 8 bits.
		scaled := v * 255 / (1<<(4*len(p)) - 1)
		if scaled > math.MaxUint8 {
			return nil, false
		}
		rgb[i] = uint8(scaled)
	}
	return color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 0xff}, true
}

// osc99ID names the prober's OSC 99 query, so its reply is told apart.
const osc99ID = "termcap"

// osc99Query asks which desktop notification features the terminal has
// (the Kitty desktop notifications protocol: metadata p=?).
var osc99Query = ansi.DesktopNotification("", "i="+osc99ID, "p=?")

// parseOSC99 reads the reply to osc99Query: OSC 99 ; i=termcap:p=? ; <keys>
// ST. Any reply means the protocol is supported; the keys are not read yet.
func parseOSC99(r Reply, c *Caps) bool {
	body, ok := strings.CutPrefix(r.Raw, "\x1b]99;")
	if !ok {
		return false
	}
	meta, _, _ := strings.Cut(body, ";")
	keys := strings.Split(meta, ":")
	if !slices.Contains(keys, "p=?") || !slices.Contains(keys, "i="+osc99ID) {
		return false
	}
	c.DesktopNotify.Set(Supported, Queried)
	return true
}

// kittyGraphicsQuery is the Kitty graphics protocol's support query: a
// one-pixel image with action q, which the terminal checks and does not
// store.
var kittyGraphicsQuery = ansi.KittyGraphics([]byte("AAAA"), "i=31", "s=1", "v=1", "a=q", "t=d", "f=24")

// parseKittyGraphics reads the reply to kittyGraphicsQuery: OK when the
// terminal can show images this way, an error otherwise.
func parseKittyGraphics(r Reply, c *Caps) bool {
	e, ok := termevent.Decode(r.Msg)
	if !ok || e.Kind != termevent.KittyGraphics {
		return false
	}
	c.KittyGraphics.Set(supportIf(e.Raw == "OK"), Queried)
	return true
}

// gatedAllowed reports whether the gated queries may go out. Some terminals
// print a query they do not understand, so they go only where the
// environment gives reason to expect an answer: not to Apple Terminal, and
// over SSH only to a terminal the environment names.
func gatedAllowed(env Env) bool {
	if env.Getenv("TERM_PROGRAM") == "Apple_Terminal" {
		return false
	}
	_, tty := env.LookupEnv("SSH_TTY")
	_, conn := env.LookupEnv("SSH_CONNECTION")
	if !tty && !conn {
		return true
	}
	if env.Getenv("LC_TERMINAL") == "iTerm2" {
		return true
	}
	term := env.Getenv("TERM")
	for _, b := range []Brand{BrandKitty, BrandGhostty, BrandWezTerm, BrandAlacritty, BrandFoot, BrandRio, BrandContour} {
		if strings.Contains(term, b.String()) {
			return true
		}
	}
	return false
}
