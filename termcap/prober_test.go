package termcap

import (
	"fmt"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/internal/termevent/termeventtest"
)

// The batch, byte by byte, for a local terminal the heuristic allows.
const (
	kittyKeyboardQ = "\x1b[?u"
	modesQ         = "\x1b[?2031$p\x1b[?2048$p\x1b[?1004$p"
	dsr996Q        = "\x1b[?996n"
	osc11Q         = "\x1b]11;?\x07"
	xtversionQ     = "\x1b[>q"
	osc99Q         = "\x1b]99;i=termcap:p=?;\x07"
	kittyGraphicsQ = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
	da1Q           = "\x1b[c"
)

var local = Env{"TERM=xterm-kitty"}

// run executes cmd and every command it batches or sequences, in order, and
// returns the messages. It never meets a timer: only Init returns one.
func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	v := reflect.ValueOf(msg)
	if v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
		var out []tea.Msg
		for i := range v.Len() {
			out = append(out, run(v.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// raw is every tea.Raw write among msgs, joined.
func raw(msgs []tea.Msg) string {
	var b strings.Builder
	for _, m := range msgs {
		if r, ok := m.(tea.RawMsg); ok {
			fmt.Fprint(&b, r.Msg)
		}
	}
	return b.String()
}

var backgroundRequest = reflect.TypeOf(tea.RequestBackgroundColor())

func count[T any](msgs []tea.Msg) int {
	n := 0
	for _, m := range msgs {
		if _, ok := m.(T); ok {
			n++
		}
	}
	return n
}

func backgroundRequests(msgs []tea.Msg) int {
	n := 0
	for _, m := range msgs {
		if reflect.TypeOf(m) == backgroundRequest {
			n++
		}
	}
	return n
}

func capsOf(t *testing.T, msgs []tea.Msg) Caps {
	t.Helper()
	var got []Caps
	for _, m := range msgs {
		if c, ok := m.(CapsMsg); ok {
			got = append(got, c.Caps)
		}
	}
	if len(got) != 1 {
		t.Fatalf("%d CapsMsg in %v, want 1", len(got), msgs)
	}
	return got[0]
}

// start returns a prober that has seen env, and what it sent.
func start(env Env, o ...Option) (*Prober, []tea.Msg) {
	p := New(o...)
	return p, run(p.Update(tea.EnvMsg(env)))
}

// feed passes each message to p and returns everything p sent back.
func feed(p *Prober, msgs ...tea.Msg) []tea.Msg {
	var out []tea.Msg
	for _, m := range msgs {
		out = append(out, run(p.Update(m))...)
	}
	return out
}

func TestBatchBytes(t *testing.T) {
	tmux := func(seq string) string {
		return "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	safe := kittyKeyboardQ + modesQ + dsr996Q + osc11Q
	cases := []struct {
		name string
		env  Env
		o    []Option
		want string
	}{
		{"local", local, nil, safe + xtversionQ + osc99Q + kittyGraphicsQ + da1Q},
		{"inside tmux", Env{"TERM=tmux-256color", "TMUX=/tmp/tmux-1/default,1,0"}, nil,
			safe + xtversionQ + tmux(osc99Q) + tmux(kittyGraphicsQ) + da1Q},
		{"unknown SSH peer", Env{"TERM=xterm-256color", "SSH_TTY=/dev/pts/1"}, nil, safe + da1Q},
		{"unknown SSH peer, no heuristic", Env{"TERM=xterm-256color", "SSH_TTY=/dev/pts/1"}, []Option{WithoutHeuristic()},
			safe + xtversionQ + osc99Q + kittyGraphicsQ + da1Q},
		{"Apple Terminal", Env{"TERM=xterm-256color", "TERM_PROGRAM=Apple_Terminal"}, nil, safe + da1Q},
		{"no background query", local, []Option{WithoutBackgroundRequest()},
			kittyKeyboardQ + modesQ + dsr996Q + xtversionQ + osc99Q + kittyGraphicsQ + da1Q},
	}
	for _, c := range cases {
		_, msgs := start(c.env, c.o...)
		if got := raw(msgs); got != c.want {
			t.Errorf("%s: batch\n %q\nwant\n %q", c.name, got, c.want)
		}
		// The OSC 11 query is in the batch, before DA1; a separate command
		// could be written after it (MADR A3).
		if n := backgroundRequests(msgs); n != 0 {
			t.Errorf("%s: %d background commands beside the batch, want 0", c.name, n)
		}
	}
}

func TestBatchNeverAsksWhatTeaAsks(t *testing.T) {
	for _, env := range []Env{local, {"TERM=xterm", "TMUX=x"}, {"TERM=xterm", "SSH_TTY=x"}} {
		_, msgs := start(env, WithoutHeuristic())
		b := raw(msgs)
		for _, q := range []string{ansi.RequestModeSynchronizedOutput, ansi.RequestModeUnicodeCore, "\x1bP+q"} {
			if strings.Contains(b, q) {
				t.Errorf("%v: the batch asks %q, which tea asks and acts on", env, q)
			}
		}
	}
}

func TestBatchGoesOutOnceOnTheFirstEnvironment(t *testing.T) {
	p := New()
	if msgs := feed(p, tea.WindowSizeMsg{Width: 80, Height: 24}, tea.ColorProfileMsg{}); raw(msgs) != "" {
		t.Fatalf("sent %q before tea.EnvMsg", raw(msgs))
	}
	if b := raw(feed(p, tea.EnvMsg(local))); !strings.HasSuffix(b, da1Q) {
		t.Fatalf("the first tea.EnvMsg sent %q", b)
	}
	if msgs := feed(p, tea.EnvMsg(local)); len(msgs) != 0 {
		t.Fatalf("a second tea.EnvMsg sent %v", msgs)
	}
}

func TestGatedAllowed(t *testing.T) {
	cases := []struct {
		env  Env
		want bool
	}{
		{Env{"TERM=xterm-256color"}, true},
		{Env{}, true},
		{Env{"TERM_PROGRAM=Apple_Terminal", "TERM=xterm-256color"}, false},
		{Env{"TERM=xterm-256color", "SSH_TTY=/dev/pts/0"}, false},
		{Env{"TERM=xterm-256color", "SSH_CONNECTION=a 1 b 22"}, false},
		{Env{"TERM=xterm-kitty", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=xterm-ghostty", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=wezterm", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=alacritty", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=foot", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=xterm-256color", "LC_TERMINAL=iTerm2", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=tmux-256color", "TMUX=x", "SSH_TTY=/dev/pts/0"}, false},
	}
	for _, c := range cases {
		if got := gatedAllowed(c.env); got != c.want {
			t.Errorf("gatedAllowed(%v) = %v, want %v", c.env, got, c.want)
		}
	}
}

func TestSentinelAloneMarksEveryQueryUnsupported(t *testing.T) {
	p, _ := start(local)
	c := capsOf(t, feed(p, termeventtest.DeviceAttributes(62, 22)))
	if !c.Complete || c.TimedOut {
		t.Fatalf("Complete %v, TimedOut %v; want true, false", c.Complete, c.TimedOut)
	}
	no := Fact[Support]{Unsupported, Queried}
	for name, f := range map[string]Fact[Support]{
		"KittyKeyboard": c.KittyKeyboard, "ColorSchemeReports": c.ColorSchemeReports,
		"InBandResize": c.InBandResize, "FocusEvents": c.FocusEvents,
		"DesktopNotify": c.DesktopNotify, "KittyGraphics": c.KittyGraphics, "Sixel": c.Sixel,
	} {
		if f != no {
			t.Errorf("%s = %+v after DA1 alone, want unsupported from a query", name, f)
		}
	}
	if c.Terminal != (Fact[string]{"xterm-kitty", Environment}) || c.Dark.Origin != NotQueried {
		t.Errorf("silence changed facts no query owns: Terminal %+v, Dark %+v", c.Terminal, c.Dark)
	}
	if c.SyncOutput.Origin != NotQueried || c.GraphemeWidth.Origin != NotQueried {
		t.Errorf("tea's facts changed with no report: %+v %+v", c.SyncOutput, c.GraphemeWidth)
	}
	if !reflect.DeepEqual(c.Attributes, []int{62, 22}) {
		t.Errorf("Attributes = %v", c.Attributes)
	}
}

func TestEveryReplySupports(t *testing.T) {
	p, _ := start(local)
	msgs := feed(p,
		tea.KeyboardEnhancementsMsg{Flags: 1},
		tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: ansi.ModeReset},
		tea.ModeReportMsg{Mode: ansi.ModeInBandResize, Value: ansi.ModeReset},
		tea.ModeReportMsg{Mode: ansi.ModeFocusEvent, Value: ansi.ModeSet},
		termeventtest.DarkColorScheme(),
		tea.TerminalVersionMsg{Name: "kitty(0.39.1)"},
		termeventtest.UnknownOsc("\x1b]99;i=termcap:p=?;a=focus,report:o=always,unfocused\x1b\\"),
		termeventtest.KittyGraphics("OK"),
		tea.BackgroundColorMsg{Color: color.RGBA{R: 0xfa, G: 0xfa, B: 0xfa, A: 0xff}},
		termeventtest.DeviceAttributes(62, 4, 22),
	)
	c := capsOf(t, msgs)
	yes := Fact[Support]{Supported, Queried}
	for name, f := range map[string]Fact[Support]{
		"KittyKeyboard": c.KittyKeyboard, "ColorSchemeReports": c.ColorSchemeReports,
		"InBandResize": c.InBandResize, "FocusEvents": c.FocusEvents,
		"DesktopNotify": c.DesktopNotify, "KittyGraphics": c.KittyGraphics, "Sixel": c.Sixel,
	} {
		if f != yes {
			t.Errorf("%s = %+v, want supported from a query", name, f)
		}
	}
	if c.KeyboardFlags != 1 {
		t.Errorf("KeyboardFlags = %d, want 1", c.KeyboardFlags)
	}
	if c.Terminal != (Fact[string]{"kitty(0.39.1)", Queried}) {
		t.Errorf("Terminal = %+v, want the XTVERSION reply", c.Terminal)
	}
	// DSR 997 said dark; the light background that came after does not
	// overrule it.
	if c.Dark != (Fact[bool]{true, Queried}) {
		t.Errorf("Dark = %+v, want the DSR 997 report", c.Dark)
	}
	if c.Background == nil {
		t.Error("Background not recorded")
	}
	if count[ColorSchemeMsg](msgs) != 0 {
		t.Error("ColorSchemeMsg sent for the probe's own DSR 996 reply")
	}
}

func TestRepliesThatSayNo(t *testing.T) {
	p, _ := start(local)
	c := capsOf(t, feed(p,
		tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: ansi.ModeNotRecognized},
		tea.ModeReportMsg{Mode: ansi.ModeFocusEvent, Value: ansi.ModePermanentlyReset},
		termeventtest.KittyGraphics("EINVAL:bad format"),
		termeventtest.UnknownOsc("\x1b]99;i=someone-else:p=?;a=focus\x1b\\"),
		termeventtest.DeviceAttributes(62),
	))
	no := Fact[Support]{Unsupported, Queried}
	if c.ColorSchemeReports != no || c.FocusEvents != no || c.KittyGraphics != no || c.DesktopNotify != no {
		t.Fatalf("got %+v %+v %+v %+v, want each unsupported", c.ColorSchemeReports, c.FocusEvents, c.KittyGraphics, c.DesktopNotify)
	}
}

func TestXTVersionOfTmuxIsAMux(t *testing.T) {
	p, _ := start(Env{"TERM=screen-256color"})
	feed(p, tea.TerminalVersionMsg{Name: "tmux 3.4"})
	if c := p.Caps(); c.Mux != (Fact[Mux]{Tmux, Queried}) {
		t.Fatalf("Mux = %+v, want tmux from the reply", c.Mux)
	}
}

func TestReplyAfterTheSentinelStillCounts(t *testing.T) {
	p, _ := start(local)
	feed(p, termeventtest.DeviceAttributes(62))
	feed(p, termeventtest.UnknownOsc("\x1b]99;i=termcap:p=?;a=focus\x07"))
	if c := p.Caps(); c.DesktopNotify != (Fact[Support]{Supported, Queried}) {
		t.Fatalf("DesktopNotify = %+v after a late reply, want supported", c.DesktopNotify)
	}
}

func TestTimeoutEndsTheProbe(t *testing.T) {
	for _, d := range []time.Duration{0, 50 * time.Millisecond} {
		synctest.Test(t, func(t *testing.T) {
			var o []Option
			want := DefaultTimeout
			if d != 0 {
				o, want = []Option{WithTimeout(d)}, d
			}
			p := New(o...)
			began := time.Now()
			deadline := p.Init()
			feed(p, tea.EnvMsg(local))
			msg := deadline()
			if got := time.Since(began); got != want {
				t.Errorf("the deadline fired after %v, want %v", got, want)
			}
			c := capsOf(t, feed(p, msg))
			if !c.TimedOut || c.Complete {
				t.Errorf("TimedOut %v, Complete %v; want true, false", c.TimedOut, c.Complete)
			}
			if c.KittyKeyboard != (Fact[Support]{}) || c.DesktopNotify != (Fact[Support]{}) {
				t.Errorf("unanswered facts are %+v %+v, want unknown", c.KittyKeyboard, c.DesktopNotify)
			}
		})
	}
}

func TestCapsMsgIsDeliveredOnce(t *testing.T) {
	da1 := termeventtest.DeviceAttributes(62)
	for name, order := range map[string][]tea.Msg{
		"sentinel, then timeout": {da1, timeoutMsg{}, da1},
		"timeout, then sentinel": {timeoutMsg{}, da1, da1},
	} {
		p, _ := start(local)
		var msgs []tea.Msg
		for _, m := range order {
			if tm, ok := m.(timeoutMsg); ok {
				tm.p = p
				m = tm
			}
			msgs = append(msgs, feed(p, m)...)
		}
		if n := count[CapsMsg](msgs); n != 1 {
			t.Errorf("%s: %d CapsMsg, want 1", name, n)
		}
	}
}

func TestAnotherProbersTimeoutIsIgnored(t *testing.T) {
	p, _ := start(local)
	if msgs := feed(p, timeoutMsg{New()}); len(msgs) != 0 {
		t.Fatalf("another prober's deadline ended this probe: %v", msgs)
	}
}

func TestTeasOwnModeReports(t *testing.T) {
	p, _ := start(local)
	feed(p,
		tea.ModeReportMsg{Mode: ansi.ModeSynchronizedOutput, Value: ansi.ModeReset},
		tea.ModeReportMsg{Mode: ansi.ModeUnicodeCore, Value: ansi.ModeNotRecognized},
	)
	c := p.Caps()
	if c.SyncOutput != (Fact[Support]{Supported, Queried}) || c.GraphemeWidth != (Fact[Support]{Unsupported, Queried}) {
		t.Fatalf("SyncOutput %+v, GraphemeWidth %+v", c.SyncOutput, c.GraphemeWidth)
	}
}

func TestColorSchemeAfterTheProbe(t *testing.T) {
	for _, o := range [][]Option{nil, {WithoutBackgroundRequest()}} {
		p, _ := start(local, o...)
		feed(p, termeventtest.DeviceAttributes(62))
		msgs := feed(p, termeventtest.LightColorScheme())
		var scheme []ColorSchemeMsg
		for _, m := range msgs {
			if s, ok := m.(ColorSchemeMsg); ok {
				scheme = append(scheme, s)
			}
		}
		if len(scheme) != 1 || scheme[0].Dark {
			t.Errorf("options %d: ColorSchemeMsg %v, want one, light", len(o), scheme)
		}
		want := 1
		if len(o) > 0 {
			want = 0
		}
		if n := backgroundRequests(msgs); n != want {
			t.Errorf("options %d: %d background requests after DSR 997, want %d", len(o), n, want)
		}
		if c := p.Caps(); c.Dark != (Fact[bool]{false, Queried}) {
			t.Errorf("Dark = %+v, want light from DSR 997", c.Dark)
		}
	}
}

func TestBackgroundSetsDarkWithoutDSR997(t *testing.T) {
	p, _ := start(local)
	feed(p, tea.BackgroundColorMsg{Color: color.RGBA{A: 0xff}})
	if c := p.Caps(); c.Dark != (Fact[bool]{true, Queried}) {
		t.Fatalf("Dark = %+v, want dark from OSC 11", c.Dark)
	}
}

func TestColorSchemeReportsSubscription(t *testing.T) {
	report := func(v ansi.ModeSetting) tea.ModeReportMsg {
		return tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: v}
	}
	cases := []struct {
		name    string
		o       []Option
		reply   ansi.ModeSetting
		set     string
		restore string
	}{
		{"default, supported", nil, ansi.ModeReset, ansi.SetModeLightDark, ansi.ResetModeLightDark},
		{"default, not recognised", nil, ansi.ModeNotRecognized, "", ""},
		{"declined", []Option{WithoutColorSchemeUpdates()}, ansi.ModeReset, "", ""},
	}
	for _, c := range cases {
		p, _ := start(local, c.o...)
		if got := raw(feed(p, report(c.reply), report(c.reply))); got != c.set {
			t.Errorf("%s: wrote %q, want %q once", c.name, got, c.set)
		}
		if got := p.Restore(); got != c.restore {
			t.Errorf("%s: Restore() = %q, want %q", c.name, got, c.restore)
		}
	}
}

func TestQuitRestoresFirst(t *testing.T) {
	p, _ := start(local)
	feed(p, tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: ansi.ModeReset})
	msgs := run(p.Quit())
	if len(msgs) != 2 || raw(msgs[:1]) != ansi.ResetModeLightDark {
		t.Fatalf("Quit() = %v, want the reset, then quit", msgs)
	}
	if _, ok := msgs[1].(tea.QuitMsg); !ok {
		t.Fatalf("Quit()'s second message is %T, want tea.QuitMsg", msgs[1])
	}

	q, _ := start(local)
	if msgs := run(q.Quit()); len(msgs) != 1 || count[tea.QuitMsg](msgs) != 1 {
		t.Fatalf("Quit() with nothing to restore = %v, want tea.QuitMsg alone", msgs)
	}
}

func TestDisabledSendsNothing(t *testing.T) {
	p := New(WithDisabled())
	if p.Init() != nil {
		t.Fatal("Init returned a command")
	}
	msgs := feed(p, tea.EnvMsg(Env{"TERM=xterm", "TMUX=x"}))
	if raw(msgs) != "" || backgroundRequests(msgs) != 0 {
		t.Fatalf("sent %v", msgs)
	}
	c := capsOf(t, msgs)
	if c.Complete || c.TimedOut || c.Mux != (Fact[Mux]{Tmux, Environment}) {
		t.Fatalf("Caps = %+v, want the environment's facts only", c)
	}
	if b := raw(feed(p, tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: ansi.ModeReset})); b != "" {
		t.Fatalf("a disabled prober wrote %q", b)
	}
}

func TestAddedQuery(t *testing.T) {
	const seq = "\x1b[?5n"
	q := Query{
		Name: "dsr-5",
		Seq:  fixed(seq),
		Parse: func(r Reply, c *Caps) bool {
			if r.Raw != "\x1b[0n" {
				return false
			}
			c.Terminal.Set("answered", Queried)
			return true
		},
	}
	gated := Query{Name: "gated", Seq: fixed("\x1b[?6n"), Gated: true, Parse: func(Reply, *Caps) bool { return false }}
	p, msgs := start(Env{"TERM=xterm", "SSH_TTY=x"}, WithQuery(q), WithQuery(gated))
	b := raw(msgs)
	if !strings.HasSuffix(b, osc11Q+seq+da1Q) {
		t.Errorf("batch %q: want the added query after the built-in ones and before DA1", b)
	}
	if strings.Contains(b, "\x1b[?6n") {
		t.Errorf("batch %q holds a gated query the heuristic refused", b)
	}
	feed(p, termeventtest.UnknownCsi("\x1b[0n"))
	if c := p.Caps(); c.Terminal != (Fact[string]{"answered", Queried}) {
		t.Errorf("the added query's reply was not parsed: %+v", c.Terminal)
	}
}

func TestOverrideBeatsAQuery(t *testing.T) {
	p, _ := start(local, WithOverride(func(c *Caps) {
		c.DesktopNotify.Set(Unsupported, Override)
		c.KeyboardFlags = 0
	}))
	c := capsOf(t, feed(p,
		termeventtest.UnknownOsc("\x1b]99;i=termcap:p=?;a=focus\x1b\\"),
		tea.KeyboardEnhancementsMsg{Flags: 31},
		termeventtest.DeviceAttributes(62),
	))
	if c.DesktopNotify != (Fact[Support]{Unsupported, Override}) || c.KeyboardFlags != 0 {
		t.Fatalf("DesktopNotify %+v, KeyboardFlags %d; want the override", c.DesktopNotify, c.KeyboardFlags)
	}
}

func TestCapsIsACopy(t *testing.T) {
	p, _ := start(local)
	feed(p, termeventtest.DeviceAttributes(62, 4))
	c := p.Caps()
	c.Attributes[0] = 1
	if p.Caps().Attributes[0] != 62 {
		t.Fatal("Caps shares Attributes with the prober")
	}
}
