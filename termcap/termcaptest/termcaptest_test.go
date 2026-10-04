package termcaptest

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/termcap"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// app is the smallest program that embeds a prober: it quits through the
// prober's Quit on StopMsg, as every program must.
type app struct {
	p         *termcap.Prober
	plainQuit bool // quit with tea.Quit, skipping the restore: only a mutation does this
	schemes   []termcap.ColorSchemeMsg
	waitFor   int // ColorSchemeMsg to see before quitting
	stopped   bool
}

func newApp(o ...termcap.Option) *app { return &app{p: termcap.New(o...)} }

func (a *app) Init() tea.Cmd { return a.p.Init() }

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := a.p.Update(msg)
	switch m := msg.(type) {
	case termcap.ColorSchemeMsg:
		a.schemes = append(a.schemes, m)
	case StopMsg:
		a.stopped = true
	}
	if a.stopped && len(a.schemes) >= a.waitFor {
		if a.plainQuit {
			return a, tea.Batch(cmd, tea.Quit)
		}
		return a, tea.Batch(cmd, a.p.Quit())
	}
	return a, cmd
}

func (a *app) View() tea.View { return tea.NewView("") }

type fact = termcap.Fact[termcap.Support]

var (
	yes = fact{Value: termcap.Supported, Origin: termcap.Queried}
	no  = fact{Value: termcap.Unsupported, Origin: termcap.Queried}
)

func TestEachProfile(t *testing.T) {
	cases := []struct {
		p          Profile
		terminal   string
		mux        termcap.Mux
		dark       bool
		keyboard   fact
		scheme     fact
		resize     fact
		focus      fact
		notify     fact
		graphics   fact
		syncOutput fact
		profile    colorprofile.Profile
	}{
		{Kitty(), "kitty(0.39.1)", termcap.NoMux, true, yes, yes, yes, yes, yes, yes, yes, colorprofile.TrueColor},
		{XTerm(), "xterm-256color", termcap.NoMux, false, no, no, no, yes, no, no, no, colorprofile.ANSI256},
		{Tmux(true), "tmux 3.4", termcap.Tmux, true, no, no, no, yes, yes, yes, no, colorprofile.TrueColor},
		{Tmux(false), "tmux 3.4", termcap.Tmux, true, no, no, no, yes, no, no, no, colorprofile.TrueColor},
		{DA1Only(), "xterm-256color", termcap.NoMux, false, no, no, no, no, no, no, fact{}, colorprofile.ANSI},
	}
	for _, c := range cases {
		got := Run(t, newApp(), c.p)
		if !got.Complete || got.TimedOut {
			t.Errorf("%s: Complete %v, TimedOut %v; want the sentinel", c.p.Name, got.Complete, got.TimedOut)
		}
		if got.Terminal.Value != c.terminal || got.Mux.Value != c.mux || got.Profile != c.profile {
			t.Errorf("%s: Terminal %+v, Mux %+v, Profile %v; want %q, %v, %v", c.p.Name, got.Terminal, got.Mux, got.Profile, c.terminal, c.mux, c.profile)
		}
		if c.p.Background != nil && got.Dark.Value != c.dark {
			t.Errorf("%s: Dark = %+v, want %v", c.p.Name, got.Dark, c.dark)
		}
		for name, f := range map[string][2]fact{
			"KittyKeyboard":      {got.KittyKeyboard, c.keyboard},
			"ColorSchemeReports": {got.ColorSchemeReports, c.scheme},
			"InBandResize":       {got.InBandResize, c.resize},
			"FocusEvents":        {got.FocusEvents, c.focus},
			"DesktopNotify":      {got.DesktopNotify, c.notify},
			"KittyGraphics":      {got.KittyGraphics, c.graphics},
		} {
			if f[0] != f[1] {
				t.Errorf("%s: %s = %+v, want %+v", c.p.Name, name, f[0], f[1])
			}
		}
		// SyncOutput is tea's query: answered, or not recognised, when the
		// profile answers DECRQM at all.
		if got.SyncOutput != c.syncOutput {
			t.Errorf("%s: SyncOutput = %+v, want %+v", c.p.Name, got.SyncOutput, c.syncOutput)
		}
	}
}

func TestSilentEndsByTimeout(t *testing.T) {
	got := Run(t, newApp(termcap.WithTimeout(100*time.Millisecond)), Silent())
	if !got.TimedOut || got.Complete {
		t.Fatalf("TimedOut %v, Complete %v; want a timeout", got.TimedOut, got.Complete)
	}
	if got.KittyKeyboard != (fact{}) || got.DesktopNotify != (fact{}) {
		t.Fatalf("unanswered facts %+v %+v, want unknown", got.KittyKeyboard, got.DesktopNotify)
	}
}

func TestOnlyTeaAsksForModes2026And2027(t *testing.T) {
	for _, p := range []Profile{Kitty(), XTerm(), Tmux(true)} {
		term := NewTerminal(p)
		term.Run(t, newApp())
		out := term.Output()
		for _, q := range []string{ansi.RequestModeSynchronizedOutput, ansi.RequestModeUnicodeCore} {
			if n := strings.Count(out, q); n != 1 {
				t.Errorf("%s: %q written %d times, want once, by tea", p.Name, q, n)
			}
		}
	}
}

func TestColorSchemeReportMidRun(t *testing.T) {
	term := NewTerminal(Kitty())
	a := newApp()
	a.waitFor = 1
	go func() {
		// Wait for the subscription, which follows the probe, then switch
		// to light.
		for deadline := time.Now().Add(RunTimeout); !strings.Contains(term.Output(), ansi.SetModeLightDark); {
			if time.Now().After(deadline) {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		term.Send("\x1b[?997;2n")
	}()
	term.Run(t, a)
	if len(a.schemes) != 1 || a.schemes[0].Dark {
		t.Fatalf("ColorSchemeMsg %v, want one, light", a.schemes)
	}
}

// modeEvents is each set or reset of mode 2031 in the output, in order.
func modeEvents(term *Terminal) []string {
	var out []string
	for _, s := range term.Sequences() {
		switch s {
		case ansi.SetModeLightDark:
			out = append(out, "set")
		case ansi.ResetModeLightDark:
			out = append(out, "reset")
		}
	}
	return out
}

func TestResetBeforeExit(t *testing.T) {
	term := NewTerminal(Kitty())
	term.Run(t, newApp())
	if got := strings.Join(modeEvents(term), ","); got != "set,reset" {
		t.Fatalf("mode 2031: %s, want set after the probe and reset before exit", got)
	}
	seqs := term.Sequences()
	if seqs[len(seqs)-1] != ansi.ResetModeLightDark {
		t.Errorf("the last sequence before exit is %q, want the 2031 reset", seqs[len(seqs)-1])
	}

	declined := NewTerminal(Kitty())
	declined.Run(t, newApp(termcap.WithoutColorSchemeUpdates()))
	if got := modeEvents(declined); len(got) != 0 {
		t.Fatalf("with WithoutColorSchemeUpdates, mode 2031: %v, want neither", got)
	}
}

// program embeds a prober beside a workspace, as the guide says (MADR A2,
// Q8): the workspace's own background query is turned off.
type program struct {
	*app
	ws *workspace.Workspace
}

type blank struct{}

func (blank) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return blank{}, nil }
func (blank) View(int, int) string                     { return "" }

func newProgram(o ...workspace.Option) *program {
	ws := workspace.New(layout.Pane{ID: "main"}, map[layout.PaneID]workspace.Pane{"main": blank{}}, o...)
	return &program{app: newApp(), ws: ws}
}

func (p *program) Init() tea.Cmd { return tea.Batch(p.app.Init(), p.ws.Init()) }

func (p *program) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	wcmd := p.ws.Update(msg)
	_, cmd := p.app.Update(msg)
	return p, tea.Batch(wcmd, cmd)
}

func TestOneBackgroundQueryBesideAWorkspace(t *testing.T) {
	for _, c := range []struct {
		o    []workspace.Option
		want int
	}{
		{[]workspace.Option{workspace.WithoutBackgroundQuery()}, 1},
		{nil, 2}, // what the guide's rule prevents
	} {
		term := NewTerminal(Kitty())
		term.Run(t, newProgram(c.o...))
		if n := strings.Count(term.Output(), "\x1b]11;?"); n != c.want {
			t.Errorf("workspace options %d: %d OSC 11 queries, want %d", len(c.o), n, c.want)
		}
	}
}

func TestSeqLen(t *testing.T) {
	cases := map[string]int{
		"\x1b[c":                     3,
		"\x1b[?2031$p":               9,
		"\x1b[?2031":                 0,
		"\x1b]11;?\x07":              7,
		"\x1b]99;a;b\x1b\\":          10,
		"\x1b]99;a":                  0,
		"\x1bPtmux;\x1b\x1b]9\x1b\\": 13,
		"\x1bPtmux;\x1b\x1b":         0,
		"\x1b_Gi=1;OK\x1b\\":         11,
		"\x1b7":                      2,
		"\x1b":                       0,
	}
	for s, want := range cases {
		if got := seqLen(s); got != want {
			t.Errorf("seqLen(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestSequenceSplitAcrossWrites(t *testing.T) {
	term := NewTerminal(DA1Only())
	for _, part := range []string{"text\x1b", "[", "c more"} {
		if _, err := term.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if got := term.Sequences(); len(got) != 1 || got[0] != "\x1b[c" {
		t.Fatalf("sequences %q, want one DA1", got)
	}
	if r := <-term.input; r != "\x1b[?1c" {
		t.Fatalf("reply %q, want the DA1 answer", r)
	}
}
