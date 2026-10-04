package termcap

import (
	"slices"
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

// WithoutBackgroundRequest leaves tea.RequestBackgroundColor out of the
// batch, and out of the follow-up to each DSR 997 report.
func WithoutBackgroundRequest() Option { return func(p *Prober) { p.noBackground = true } }

// WithDisabled sends nothing. Init returns nil, and the first tea.EnvMsg
// delivers a CapsMsg holding only the environment's facts. A program run
// without input, tea.WithInput(nil), must not probe: tea skips its own
// queries then, but the replies to the prober's would reach the shell.
func WithDisabled() Option { return func(p *Prober) { p.disabled = true } }

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

	caps    Caps
	sent    []*probe // the batch's queries, in order, once sent
	started bool     // the first tea.EnvMsg arrived
	done    bool     // CapsMsg was delivered
	dsr997  bool     // Dark came from a DSR 997 report
	set2031 bool     // the prober subscribed to mode 2031
}

// probe is a query in the batch.
type probe struct {
	Query
	silent   func(c *Caps) // what no reply before the sentinel means; nil for nothing
	answered bool
}

// timeoutMsg is the deadline Init starts.
type timeoutMsg struct{ p *Prober }

// New returns a Prober.
func New(o ...Option) *Prober {
	p := &Prober{timeout: DefaultTimeout}
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
	case tea.ColorProfileMsg:
		p.caps.Profile = m.Profile
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
	for _, q := range p.sent {
		if q.Parse(r, &p.caps) {
			q.answered = true
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
	p.caps.setEnv(env)
	if p.disabled {
		return p.deliver()
	}
	gated := p.ungated || gatedAllowed(env)
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
	cmds := []tea.Cmd{tea.Raw(b.String())}
	if !p.noBackground {
		cmds = append(cmds, tea.RequestBackgroundColor)
	}
	return tea.Batch(cmds...)
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

// deliver sends CapsMsg, once.
func (p *Prober) deliver() tea.Cmd {
	if p.done {
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
// Kitty keyboard request, the DECRQM set and DSR 996, the gated queries,
// then each added query.
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
		{Name: "terminal-version", Seq: fixed(ansi.RequestNameVersion), Gated: true, Parse: parseVersion},
		{Name: "desktop-notify", Seq: wrapped(osc99Query), Gated: true, Parse: parseOSC99,
			silent: unsupported(func(c *Caps) *Fact[Support] { return &c.DesktopNotify })},
		{Name: "kitty-graphics", Seq: wrapped(kittyGraphicsQuery), Gated: true, Parse: parseKittyGraphics,
			silent: unsupported(func(c *Caps) *Fact[Support] { return &c.KittyGraphics })},
	}
	for _, q := range p.added {
		qs = append(qs, &probe{Query: q})
	}
	return qs
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
	return true
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
	for _, name := range []string{"kitty", "ghostty", "wezterm", "alacritty", "foot", "rio", "contour"} {
		if strings.Contains(term, name) {
			return true
		}
	}
	return false
}
