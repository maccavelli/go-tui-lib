package launch_test

// The tests of launch's exported API, written as a program's own tests are,
// on launchtest: its first consumer
// (docs/decisions/0013-PLAN-cli-integration-helpers.md Step 6, deviation
// D6). The tests that need launch's seams stay in package launch.

import (
	"context"
	"encoding"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/launch"
	"github.com/maccavelli/go-tui-lib/launch/launchtest"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/termcap"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// model is a program's model whose behaviour each test sets.
type model struct {
	keys       string // the keys it has seen
	loops      int    // the LoopMsgs it has run
	quitOnSize bool   // quit on the first WindowSizeMsg
	quitOnKey  string // quit on this key
	panicOn    string // "view", or "key:<k>"
	epilogue   string // what Epilogue returns
}

type (
	quitMsg  struct{}
	panicMsg struct{}
)

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		k := msg.String()
		if m.panicOn == "key:"+k {
			panic("a panic in Update")
		}
		m.keys += k
		if k == m.quitOnKey {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if m.quitOnSize {
			return m, tea.Quit
		}
	case quitMsg:
		return m, tea.Quit
	case panicMsg:
		panic("a panic in Update")
	case command.LoopMsg:
		m.loops++
		return m, msg.Run()
	}
	return m, nil
}

func (m model) View() tea.View {
	if m.panicOn == "view" {
		panic("a panic in View")
	}
	return tea.NewView("model:" + m.keys)
}

func (m model) Epilogue() string { return m.epilogue }

// idle is a terminal with no keys typed, closed when the test ends.
func idle(t *testing.T) *launchtest.Terminal {
	term := launchtest.NewTerminal(80, 24)
	t.Cleanup(func() { _ = term.Close() })
	return term
}

// interactive is a decision to start the TUI on In and Out.
func interactive() launch.Decision {
	return launch.Decision{Interactive: true, Reason: launch.ReasonTerminal, In: launch.TargetStream, UI: launch.TargetStream,
		Profile: colorprofile.ASCII, UIProfile: colorprofile.ASCII, Width: 80, Height: 24}
}

// within runs f, failing t if it takes longer than d.
func within[T any](t *testing.T, d time.Duration, f func() T) T {
	t.Helper()
	ch := make(chan T, 1)
	go func() { ch <- f() }()
	select {
	case v := <-ch:
		return v
	case <-time.After(d):
		t.Fatalf("did not return within %s", d)
		var zero T
		return zero
	}
}

type ran struct {
	m   model
	err error
}

// run runs m on s under d with a five-second guard.
func run(t *testing.T, ctx context.Context, s launch.Streams, d launch.Decision, m model, opts ...launch.Option) ran {
	t.Helper()
	return within(t, 5*time.Second, func() ran {
		got, err := launch.Run(ctx, s, d, m, opts...)
		return ran{got, err}
	})
}

// The interfaces each framework asks of a flag value
// (docs/reports/0014-REPORT-api-assessment-and-integration-research.md §3.2).
var (
	_ flag.Value               = new(launch.Choice)
	_ flag.Getter              = new(launch.Choice)
	_ encoding.TextMarshaler   = launch.ChoiceAuto
	_ encoding.TextUnmarshaler = new(launch.Choice)
	_ interface {
		flag.Value
		Type() string // pflag
	} = new(launch.Choice)
	_ interface {
		MarshalFlag() (string, error)
		UnmarshalFlag(string) error
	} = new(launch.Choice) // go-flags
)

func TestChoiceText(t *testing.T) {
	readers := map[string]func(*launch.Choice, string) error{
		"Set":           (*launch.Choice).Set,
		"UnmarshalText": func(v *launch.Choice, s string) error { return v.UnmarshalText([]byte(s)) },
		"UnmarshalFlag": (*launch.Choice).UnmarshalFlag,
	}
	for _, c := range []struct {
		choice launch.Choice
		text   string
	}{{launch.ChoiceAuto, "auto"}, {launch.ChoiceTUI, "tui"}, {launch.ChoicePlain, "plain"}} {
		if got := c.choice.String(); got != c.text {
			t.Errorf("String() = %q, want %q", got, c.text)
		}
		if got := c.choice.Type(); got != "mode" {
			t.Errorf("Type() = %q, want \"mode\"", got)
		}
		if got := c.choice.Get(); got != c.choice {
			t.Errorf("Get() = %v, want %v", got, c.choice)
		}
		b, err := c.choice.MarshalText()
		if err != nil || string(b) != c.text {
			t.Errorf("MarshalText() = %q, %v", b, err)
		}
		s, err := c.choice.MarshalFlag()
		if err != nil || s != c.text {
			t.Errorf("MarshalFlag() = %q, %v", s, err)
		}
		for name, read := range readers {
			v := launch.Choice(9)
			if err := read(&v, c.text); err != nil || v != c.choice {
				t.Errorf("%s(%q) = %v, %v", name, c.text, v, err)
			}
		}
	}
	if launch.ChoiceAuto != 0 {
		t.Error("ChoiceAuto is not the zero Choice")
	}

	// An invalid value: refused, with launch's error, and v left alone.
	for name, read := range readers {
		v := launch.ChoicePlain
		if err := read(&v, "TUI"); err == nil || err.Error() != `launch: unknown Choice "TUI"` || v != launch.ChoicePlain {
			t.Errorf("%s(\"TUI\") = %v, left %s", name, err, v)
		}
	}
	if _, err := launch.Choice(9).MarshalText(); err == nil || err.Error() != "launch: Choice 9 has no name" {
		t.Errorf("Choice(9).MarshalText(): %v", err)
	}
	if _, err := launch.Choice(9).MarshalFlag(); err == nil {
		t.Error("Choice(9).MarshalFlag() gave no error")
	}
}

func TestTargetText(t *testing.T) {
	for i, want := range []string{"none", "stream", "err", "tty"} {
		v := launch.Target(i)
		b, err := v.MarshalText()
		if v.String() != want || err != nil || string(b) != want {
			t.Errorf("Target(%d) = %q, %q, %v; want %q", i, v.String(), b, err, want)
		}
		var back launch.Target
		if err := back.UnmarshalText(b); err != nil || back != v {
			t.Errorf("UnmarshalText(%q) = %v, %v", b, back, err)
		}
	}
	if launch.TargetNone != 0 {
		t.Error("TargetNone is not the zero Target")
	}
}

func TestFlagsRegister(t *testing.T) {
	for _, c := range []struct {
		args    []string
		want    launch.Choice
		wantErr bool
	}{
		{nil, launch.ChoiceAuto, false},
		{[]string{"-tui"}, launch.ChoiceTUI, false},
		{[]string{"--tui"}, launch.ChoiceTUI, false},
		{[]string{"-tui=false"}, launch.ChoiceAuto, false},
		{[]string{"-mode=plain"}, launch.ChoicePlain, false},
		{[]string{"-mode", "tui"}, launch.ChoiceTUI, false},
		// --tui wins over --mode, whatever the order.
		{[]string{"-mode=plain", "-tui"}, launch.ChoiceTUI, false},
		{[]string{"-tui", "-mode=plain"}, launch.ChoiceTUI, false},
		{[]string{"-mode=bogus"}, launch.ChoiceAuto, true},
	} {
		var f launch.Flags
		fs := flag.NewFlagSet("prog", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f.RegisterFlags(fs)
		err := fs.Parse(c.args)
		if (err != nil) != c.wantErr {
			t.Errorf("%q: Parse = %v, want an error: %v", c.args, err, c.wantErr)
			continue
		}
		if got := f.Resolve(); !c.wantErr && got != c.want {
			t.Errorf("%q: Resolve() = %s, want %s", c.args, got, c.want)
		}
	}

	// The bare -tui is a boolean flag, so it takes no value.
	var f launch.Flags
	fs := flag.NewFlagSet("prog", flag.ContinueOnError)
	f.RegisterFlags(fs)
	if bf, ok := fs.Lookup("tui").Value.(interface{ IsBoolFlag() bool }); !ok || !bf.IsBoolFlag() {
		t.Error("-tui is not a boolean flag")
	}
	if got := fs.Lookup("mode").DefValue; got != "auto" {
		t.Errorf("-mode's default is %q, want \"auto\"", got)
	}
}

// source is what a *cobra.Command has.
type source struct {
	in  io.Reader
	out io.Writer
	err io.Writer
}

func (s source) InOrStdin() io.Reader   { return s.in }
func (s source) OutOrStdout() io.Writer { return s.out }
func (s source) ErrOrStderr() io.Writer { return s.err }

func TestFromSource(t *testing.T) {
	src := source{launchtest.NewPipe(""), launchtest.NewTerminal(80, 24), launchtest.NewPipe("")}
	s := launch.FromSource(src, launchtest.Env("TERM=xterm"))
	if s.In != src.in || s.Out != src.out || s.Err != src.err {
		t.Error("FromSource did not take the source's three streams")
	}
	if got := s.Env.Getenv("TERM"); got != "xterm" {
		t.Errorf("FromSource's Env: TERM = %q", got)
	}
}

func TestRunQuits(t *testing.T) {
	out := launchtest.NewTerminal(80, 24)
	r := run(t, t.Context(), launch.Streams{In: idle(t), Out: out}, interactive(), model{quitOnSize: true, keys: "x"})
	if r.err != nil || r.m.keys != "x" {
		t.Fatalf("Run = %+v, %v; want the typed model and nil", r.m, r.err)
	}
	if !strings.Contains(out.Output(), "model:x") {
		t.Errorf("Out does not hold the view: %q", out.Output())
	}
}

func TestRunUsesTheGivenInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	in := idle(t)
	in.Type("abq")
	r := run(t, ctx, launch.Streams{In: in, Out: launchtest.NewTerminal(80, 24)}, interactive(), model{quitOnKey: "q"})
	if r.err != nil || r.m.keys != "abq" {
		t.Fatalf("Run = %+v, %v; want the keys typed on s.In", r.m, r.err)
	}
}

func TestRunRegistryDetached(t *testing.T) {
	for _, end := range []string{"quit", "crash", "cancel"} {
		t.Run(end, func(t *testing.T) {
			var calls atomic.Int32
			r := command.NewRegistry()
			ping, err := command.New("app.ping", "Ping", func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
				calls.Add(1)
				return command.Result{Text: "pong"}, nil
			}, command.WithDanger(command.ReadOnly))
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Register(ping); err != nil {
				t.Fatal(err)
			}
			req := command.Request{ID: "app.ping", Origin: command.OriginAgent}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			started := make(chan *tea.Program, 1)
			done := make(chan ran, 1)
			go func() {
				got, err := launch.Run(ctx, launch.Streams{In: idle(t), Out: launchtest.NewTerminal(80, 24)}, interactive(), model{},
					launch.WithRegistry(r), launch.OnStart(func(p *tea.Program) { started <- p }))
				done <- ran{got, err}
			}()
			p := within(t, 5*time.Second, func() *tea.Program { return <-started })

			// While the TUI runs, an agent's Loop command runs on its loop.
			res := within(t, 5*time.Second, func() string {
				res, err := r.Run(t.Context(), req)
				if err != nil {
					t.Error(err)
				}
				return res.Text
			})
			if res != "pong" {
				t.Errorf("the agent's command gave %q", res)
			}
			switch end {
			case "quit":
				p.Send(quitMsg{})
			case "crash":
				p.Send(panicMsg{})
			case "cancel":
				cancel()
			}
			out := within(t, 5*time.Second, func() ran { return <-done })
			if out.m.loops != 1 {
				t.Errorf("the model ran %d LoopMsgs, want 1", out.m.loops)
			}

			// After Run, the same command runs on its caller, at once.
			before := calls.Load()
			text := within(t, time.Second, func() string {
				res, err := r.Run(context.Background(), req)
				if err != nil {
					t.Error(err)
				}
				return res.Text
			})
			if text != "pong" || calls.Load() != before+1 {
				t.Errorf("after Run: %q, with %d runs", text, calls.Load()-before)
			}
		})
	}
}

func TestRunRestorerAfterCrash(t *testing.T) {
	p := termcap.New()
	p.Update(tea.ColorProfileMsg{Profile: colorprofile.TrueColor})
	p.Update(tea.EnvMsg{"TERM=xterm-kitty"})
	p.Update(tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: ansi.ModeReset})
	if p.Restore() != ansi.ResetModeLightDark {
		t.Fatal("the prober did not set mode 2031")
	}
	in := idle(t)
	in.Type("p")
	out := launchtest.NewTerminal(80, 24)
	r := run(t, t.Context(), launch.Streams{In: in, Out: out}, interactive(), model{panicOn: "key:p"}, launch.WithRestorer(p))
	if !errors.Is(r.err, launch.ErrCrashed) {
		t.Fatalf("err = %v, want a crash", r.err)
	}
	if !strings.HasSuffix(out.Output(), ansi.ResetModeLightDark) {
		t.Errorf("the drawing stream does not end with mode 2031's reset: %q", out.Output())
	}
}

func TestRunOnStart(t *testing.T) {
	var calls atomic.Int32
	var mu sync.Mutex
	var seen *tea.Program
	r := run(t, t.Context(), launch.Streams{In: idle(t), Out: launchtest.NewTerminal(80, 24)}, interactive(), model{},
		launch.OnStart(func(p *tea.Program) {
			calls.Add(1)
			mu.Lock()
			seen = p
			mu.Unlock()
			go p.Send(quitMsg{}) // delivered once the program runs
		}))
	if r.err != nil {
		t.Fatalf("err = %v; the message OnStart sent did not quit the program", r.err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls.Load() != 1 || seen == nil {
		t.Errorf("OnStart was called %d times, with %v", calls.Load(), seen)
	}
}

func TestRunEpilogue(t *testing.T) {
	out := launchtest.NewTerminal(80, 24)
	r := run(t, t.Context(), launch.Streams{In: idle(t), Out: out}, interactive(), model{quitOnSize: true, epilogue: "resume with --continue\n"})
	if r.err != nil {
		t.Fatal(r.err)
	}
	if s := out.Output(); !strings.HasSuffix(s, "resume with --continue\n") || strings.Index(s, "model:") > strings.Index(s, "resume") {
		t.Errorf("the epilogue is not written after the TUI's last byte: %q", s)
	}

	out = launchtest.NewTerminal(80, 24)
	r = run(t, t.Context(), launch.Streams{In: idle(t), Out: out}, interactive(), model{panicOn: "view", epilogue: "resume"})
	if !errors.Is(r.err, launch.ErrCrashed) || strings.Contains(out.Output(), "resume") {
		t.Errorf("after a crash: %v, and Out holds %q", r.err, out.Output())
	}
}

func TestWithFilterSeesTheModel(t *testing.T) {
	var sawModel atomic.Bool
	in := idle(t)
	in.Type("axbq")
	r := run(t, t.Context(), launch.Streams{In: in, Out: launchtest.NewTerminal(80, 24)}, interactive(), model{quitOnKey: "q"},
		launch.WithFilter(func(m model, msg tea.Msg) tea.Msg {
			sawModel.Store(true)
			if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "x" {
				return nil // dropped
			}
			return msg
		}))
	if r.err != nil || r.m.keys != "abq" {
		t.Errorf("Run = %+v, %v; want x filtered out", r.m, r.err)
	}
	if !sawModel.Load() {
		t.Error("the filter never saw the program's own model")
	}
}

// other is a model of another type.
type other struct{}

func (other) Init() tea.Cmd                         { return nil }
func (o other) Update(tea.Msg) (tea.Model, tea.Cmd) { return o, nil }
func (other) View() tea.View                        { return tea.NewView("other") }

// becomes turns into other on its first window size, and quits.
type becomes struct{ model }

func (b becomes) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		return other{}, tea.Quit
	}
	return b, nil
}

func TestRunFinalModelOfAnotherType(t *testing.T) {
	type out struct {
		b   becomes
		err error
	}
	r := within(t, 5*time.Second, func() out {
		b, err := launch.Run(t.Context(), launch.Streams{In: idle(t), Out: launchtest.NewTerminal(80, 24)}, interactive(), becomes{model{keys: "kept"}})
		return out{b, err}
	})
	if r.err == nil || !strings.Contains(r.err.Error(), "launch_test.other") || !strings.Contains(r.err.Error(), "launch_test.becomes") {
		t.Errorf("err = %v; want one naming both types", r.err)
	}
	if r.b.keys != "kept" {
		t.Errorf("the model is %+v; want the last good one", r.b)
	}
}

// text is a pane that shows its lines.
type text struct{ lines []string }

func (p text) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return p, nil }
func (p text) View(_, _ int) string                     { return strings.Join(p.lines, "\n") }

// board is the program around a workspace.
type board struct{ ws *workspace.Workspace }

func (b board) Init() tea.Cmd                           { return b.ws.Init() }
func (b board) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return b, b.ws.Update(msg) }
func (b board) View() tea.View                          { return tea.NewView(b.ws.Render()) }

// TestFrameGolden draws a three-pane workspace once, as a plain status
// board. The tree is fixed, not a responsive preset, so every case shows the
// three panes. The workspace starts with a TrueColor theme and follows the
// profile Frame sends, so a no-colour case is plain only if Frame sends it.
func TestFrameGolden(t *testing.T) {
	tuitest.Golden(t, "frame-workspace", tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
		p := colorprofile.ASCII
		if c.Color {
			p = colorprofile.TrueColor
		}
		g := glyph.For(c.UTF8)
		build := func(p colorprofile.Profile, bg theme.Background) theme.Theme { return theme.New(p, bg, g) }
		root := layout.Split{Axis: layout.Vertical, Children: []layout.Child{
			{Node: layout.Split{Axis: layout.Horizontal, Children: []layout.Child{
				{Node: layout.Pane{ID: "main"}},
				{Node: layout.Pane{ID: "side"}, Size: layout.Fixed(18)},
			}}},
			{Node: layout.Pane{ID: "bottom"}, Size: layout.Fixed(3)},
		}}
		ws := workspace.New(
			root,
			map[layout.PaneID]workspace.Pane{
				"main":   text{[]string{"build 1832 passed", "deploy to staging: done"}},
				"side":   text{[]string{"3 agents", "0 failing"}},
				"bottom": text{[]string{"last run 12:04"}},
			},
			workspace.WithTheme(theme.New(colorprofile.TrueColor, theme.Dark, g)),
			workspace.WithThemeBuilder(build),
		)
		return launch.Frame(board{ws}, c.Width, 12, p)
	})
}

// commands is a model that records whether Init ran, or any command it
// returned did.
type commands struct{ init, ran *bool }

func (m commands) Init() tea.Cmd {
	*m.init = true
	return func() tea.Msg { *m.ran = true; return nil }
}

func (m commands) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, func() tea.Msg { *m.ran = true; return nil }
}

func (m commands) View() tea.View { return tea.NewView("drawn") }

func TestFrameRunsNoCommand(t *testing.T) {
	var init, ranCmd bool
	if got := launch.Frame(commands{&init, &ranCmd}, 20, 2, colorprofile.ASCII); got != "drawn" {
		t.Errorf("Frame = %q", got)
	}
	if init || ranCmd {
		t.Errorf("Frame called Init (%v) or ran a command (%v)", init, ranCmd)
	}
}

// exitCoder is a program's own error with a status.
type exitCoder struct{}

func (exitCoder) Error() string { return "custom" }
func (exitCoder) ExitCode() int { return 42 }

func TestExitCode(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"ErrNotStarted", launch.ErrNotStarted, 2},
		{"tea.ErrInterrupted", tea.ErrInterrupted, 130},
		{"context.DeadlineExceeded", context.DeadlineExceeded, 124},
		{"context.Canceled", context.Canceled, 130},
		{"tea.ErrProgramPanic", tea.ErrProgramPanic, 2},
		{"another error", errors.New("x"), 1},
		{"ExitError", &launch.ExitError{Code: 7}, 7},
		{"a type with ExitCode", exitCoder{}, 42},
	} {
		if got := launch.ExitCode(c.err); got != c.want {
			t.Errorf("%s: ExitCode = %d, want %d", c.name, got, c.want)
		}
		if c.err == nil {
			continue
		}
		if got := launch.ExitCode(fmt.Errorf("wrapped: %w", c.err)); got != c.want {
			t.Errorf("%s, wrapped: ExitCode = %d, want %d", c.name, got, c.want)
		}
	}

	// As Run returns them.
	crash := fmt.Errorf("%w: %w", launch.ErrCrashed, fmt.Errorf("%w: %w", tea.ErrProgramKilled, tea.ErrProgramPanic))
	if got := launch.ExitCode(crash); got != 2 {
		t.Errorf("a crash from a panic: %d, want 2", got)
	}
	if got := launch.ExitCode(fmt.Errorf("%w: %w", tea.ErrProgramKilled, context.Canceled)); got != 130 {
		t.Errorf("a cancel: %d, want 130", got)
	}

	// An ExitError inside ErrCrashed, around a panic: the ExitError wins.
	inside := fmt.Errorf("%w: %w", launch.ErrCrashed, &launch.ExitError{Code: 7, Err: tea.ErrProgramPanic})
	if got := launch.ExitCode(inside); got != 7 {
		t.Errorf("an ExitError inside a crash: %d, want 7", got)
	}
}

func TestExitError(t *testing.T) {
	cause := errors.New("cause")
	e := &launch.ExitError{Code: 3, Err: cause}
	if e.Error() != "cause" || !errors.Is(e, cause) || e.ExitCode() != 3 {
		t.Errorf("ExitError{3, cause} = %q, Is %v, %d", e.Error(), errors.Is(e, cause), e.ExitCode())
	}
	if got := (&launch.ExitError{Code: 4}).Error(); got != "exit status 4" {
		t.Errorf("ExitError{4} = %q", got)
	}
	if (&launch.ExitError{Code: 4}).Unwrap() != nil {
		t.Error("an ExitError with no Err unwraps to something")
	}
}
