package launch

import (
	"context"
	"errors"
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
	"github.com/maccavelli/go-tui-lib/termcap"
)

// tm is a model whose behaviour each test sets.
type tm struct {
	keys       string  // the keys it has seen
	loops      int     // the LoopMsgs it has run
	quitOnSize bool    // quit on the first WindowSizeMsg
	quitOnKey  string  // quit on this key
	panicOn    string  // "init", "view", or "key:<k>"
	initCmd    tea.Cmd // what Init returns
	epilogue   string  // what Epilogue returns
}

type (
	quitMsg  struct{}
	panicMsg struct{}
)

func (m tm) Init() tea.Cmd {
	if m.panicOn == "init" {
		panic("a panic in Init")
	}
	return m.initCmd
}

func (m tm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m tm) View() tea.View {
	if m.panicOn == "view" {
		panic("a panic in View")
	}
	return tea.NewView("tm:" + m.keys)
}

func (m tm) Epilogue() string { return m.epilogue }

// blocking is an input that has nothing to read until the test ends.
func blocking(t *testing.T) io.Reader {
	r, w := io.Pipe()
	t.Cleanup(func() { _ = w.Close() })
	return r
}

// interactive is a decision to start the TUI on In and Out.
func interactive() Decision {
	return Decision{Interactive: true, Reason: ReasonTerminal, In: TargetStream, UI: TargetStream,
		Profile: colorprofile.ASCII, UIProfile: colorprofile.ASCII, Width: 80, Height: 24}
}

// saves replaces the saved-state seam for one test, and counts its saves
// and restores.
type saves struct{ saved, restored atomic.Int32 }

func countSaves(t *testing.T) *saves {
	c := &saves{}
	old := save
	save = func(any) (func() error, error) {
		c.saved.Add(1)
		return func() error { c.restored.Add(1); return nil }, nil
	}
	t.Cleanup(func() { save = old })
	return c
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

type result struct {
	m   tm
	err error
}

// runTM runs m on s under d with a five-second guard.
func runTM(t *testing.T, ctx context.Context, s Streams, d Decision, m tm, opts ...Option) result {
	t.Helper()
	return within(t, 5*time.Second, func() result {
		got, err := Run(ctx, s, d, m, opts...)
		return result{got, err}
	})
}

type resetRestorer struct{}

func (resetRestorer) Restore() string { return "<RESET>" }

func TestRunPlainWritesNothing(t *testing.T) {
	var built atomic.Int32
	old := newProgram
	newProgram = func(m tea.Model, o ...tea.ProgramOption) *tea.Program {
		built.Add(1)
		return old(m, o...)
	}
	t.Cleanup(func() { newProgram = old })
	in, out, errw := terminal(80, 24), terminal(80, 24), terminal(80, 24)
	d := Decision{Reason: ReasonCI}
	got, err := Run(t.Context(), Streams{In: in, Out: out, Err: errw}, d, tm{keys: "start"}, WithRestorer(resetRestorer{}))
	if !errors.Is(err, ErrNotStarted) || !strings.Contains(err.Error(), "launch.ci") {
		t.Errorf("err = %v; want ErrNotStarted naming the reason", err)
	}
	if got.keys != "start" {
		t.Errorf("the model came back as %+v, want the one given", got)
	}
	if built.Load() != 0 {
		t.Error("a plain decision built a program")
	}
	if in.String() != "" || out.String() != "" || errw.String() != "" {
		t.Errorf("a plain decision wrote %q, %q, %q", in.String(), out.String(), errw.String())
	}
}

func TestRunQuits(t *testing.T) {
	out := terminal(80, 24)
	r := runTM(t, t.Context(), Streams{In: blocking(t), Out: out}, interactive(), tm{quitOnSize: true, keys: "x"})
	if r.err != nil || r.m.keys != "x" {
		t.Fatalf("Run = %+v, %v; want the typed model and nil", r.m, r.err)
	}
	if !strings.Contains(out.String(), "tm:x") {
		t.Errorf("Out does not hold the view: %q", out.String())
	}
}

func TestRunUsesTheGivenInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	r := runTM(t, ctx, Streams{In: strings.NewReader("abq"), Out: terminal(80, 24)}, interactive(), tm{quitOnKey: "q"})
	if r.err != nil || r.m.keys != "abq" {
		t.Fatalf("Run = %+v, %v; want the keys typed on s.In", r.m, r.err)
	}
}

func TestRunNotStarted(t *testing.T) {
	c := countSaves(t)
	old := newProgram
	// A program that fails before Init: Bubble Tea refuses a nil model.
	newProgram = func(_ tea.Model, o ...tea.ProgramOption) *tea.Program { return old(nil, o...) }
	t.Cleanup(func() { newProgram = old })
	out := terminal(80, 24)
	r := runTM(t, t.Context(), Streams{In: blocking(t), Out: out}, interactive(), tm{keys: "given"}, WithRestorer(resetRestorer{}))
	if !errors.Is(r.err, ErrNotStarted) || !strings.Contains(r.err.Error(), "InitialModel cannot be nil") {
		t.Errorf("err = %v; want ErrNotStarted wrapping Bubble Tea's", r.err)
	}
	if errors.Is(r.err, ErrCrashed) {
		t.Error("a failure before Init is a crash")
	}
	if r.m.keys != "given" {
		t.Errorf("the model is %+v, want the one given", r.m)
	}
	if c.restored.Load() != c.saved.Load() || c.saved.Load() != 2 {
		t.Errorf("saved %d, restored %d; want both of In and Out", c.saved.Load(), c.restored.Load())
	}
	if !strings.HasSuffix(out.String(), "<RESET>") {
		t.Errorf("the Restorer was not written: %q", out.String())
	}
}

func TestRunCrash(t *testing.T) {
	boom := func() tea.Msg { panic("a panic in a command") }
	for _, c := range []struct {
		name  string
		m     tm
		input string
		keys  string // the last good model's
	}{
		{"Init", tm{panicOn: "init"}, "", ""},
		{"Update, after a good Update", tm{panicOn: "key:p"}, "ap", "a"},
		{"View", tm{panicOn: "view"}, "", ""},
		{"a command", tm{initCmd: boom}, "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			saved := countSaves(t)
			out := terminal(80, 24)
			in := blocking(t)
			if c.input != "" {
				in = io.MultiReader(strings.NewReader(c.input), blocking(t))
			}
			r := runTM(t, t.Context(), Streams{In: in, Out: out}, interactive(), c.m, WithRestorer(resetRestorer{}))
			if !errors.Is(r.err, ErrCrashed) || !errors.Is(r.err, tea.ErrProgramPanic) {
				t.Fatalf("err = %v; want ErrCrashed wrapping tea.ErrProgramPanic", r.err)
			}
			if errors.Is(r.err, ErrNotStarted) {
				t.Error("a crash is also not-started")
			}
			if r.m.keys != c.keys {
				t.Errorf("the model's keys are %q, want the last good model's %q", r.m.keys, c.keys)
			}
			if saved.restored.Load() != 2 {
				t.Errorf("restored %d saved states, want 2", saved.restored.Load())
			}
			if !strings.HasSuffix(out.String(), "<RESET>") {
				t.Errorf("the Restorer was not written last: %q", out.String())
			}
		})
	}
}

func TestRunEndsAreNotCrashes(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	deadline, stop := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer stop()
	for _, c := range []struct {
		name string
		ctx  context.Context
		m    tm
		want error
	}{
		{"tea.Quit", t.Context(), tm{quitOnSize: true}, nil},
		{"tea.Interrupt", t.Context(), tm{initCmd: tea.Interrupt}, tea.ErrInterrupted},
		{"a cancelled ctx", cancelled, tm{initCmd: func() tea.Msg { cancel(); return nil }}, context.Canceled},
		{"a passed deadline", deadline, tm{}, context.DeadlineExceeded},
	} {
		t.Run(c.name, func(t *testing.T) {
			saved := countSaves(t)
			out := terminal(80, 24)
			r := runTM(t, c.ctx, Streams{In: blocking(t), Out: out}, interactive(), c.m, WithRestorer(resetRestorer{}))
			if c.want == nil && r.err != nil || c.want != nil && !errors.Is(r.err, c.want) {
				t.Fatalf("err = %v, want %v", r.err, c.want)
			}
			if errors.Is(r.err, ErrCrashed) || errors.Is(r.err, ErrNotStarted) {
				t.Errorf("err = %v carries a sentinel", r.err)
			}
			if saved.restored.Load() != 0 {
				t.Errorf("restored %d saved states on an end the program chose", saved.restored.Load())
			}
			if !strings.HasSuffix(out.String(), "<RESET>") {
				t.Errorf("the Restorer was not written: %q", out.String())
			}
		})
	}
}

func TestRunRegistryDetached(t *testing.T) {
	for _, end := range []string{"quit", "crash", "cancel"} {
		t.Run(end, func(t *testing.T) {
			var ran atomic.Int32
			r := command.NewRegistry()
			ping, err := command.New("app.ping", "Ping", func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
				ran.Add(1)
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
			done := make(chan result, 1)
			go func() {
				got, err := Run(ctx, Streams{In: blocking(t), Out: terminal(80, 24)}, interactive(), tm{},
					WithRegistry(r), OnStart(func(p *tea.Program) { started <- p }))
				done <- result{got, err}
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
			out := within(t, 5*time.Second, func() result { return <-done })
			if out.m.loops != 1 {
				t.Errorf("the model ran %d LoopMsgs, want 1", out.m.loops)
			}

			// After Run, the same command runs on its caller, at once.
			before := ran.Load()
			text := within(t, time.Second, func() string {
				res, err := r.Run(context.Background(), req)
				if err != nil {
					t.Error(err)
				}
				return res.Text
			})
			if text != "pong" || ran.Load() != before+1 {
				t.Errorf("after Run: %q, with %d runs", text, ran.Load()-before)
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
	countSaves(t)
	out := terminal(80, 24)
	r := runTM(t, t.Context(), Streams{In: io.MultiReader(strings.NewReader("p"), blocking(t)), Out: out},
		interactive(), tm{panicOn: "key:p"}, WithRestorer(p))
	if !errors.Is(r.err, ErrCrashed) {
		t.Fatalf("err = %v, want a crash", r.err)
	}
	if !strings.HasSuffix(out.String(), ansi.ResetModeLightDark) {
		t.Errorf("the drawing stream does not end with mode 2031's reset: %q", out.String())
	}
}

func TestRunOnStart(t *testing.T) {
	var calls atomic.Int32
	var mu sync.Mutex
	var seen *tea.Program
	r := runTM(t, t.Context(), Streams{In: blocking(t), Out: terminal(80, 24)}, interactive(), tm{},
		OnStart(func(p *tea.Program) {
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
	out := terminal(80, 24)
	r := runTM(t, t.Context(), Streams{In: blocking(t), Out: out}, interactive(), tm{quitOnSize: true, epilogue: "resume with --continue\n"})
	if r.err != nil {
		t.Fatal(r.err)
	}
	if s := out.String(); !strings.HasSuffix(s, "resume with --continue\n") || strings.Index(s, "tm:") > strings.Index(s, "resume") {
		t.Errorf("the epilogue is not written after the TUI's last byte: %q", s)
	}

	countSaves(t)
	out = terminal(80, 24)
	r = runTM(t, t.Context(), Streams{In: blocking(t), Out: out}, interactive(), tm{panicOn: "view", epilogue: "resume"})
	if !errors.Is(r.err, ErrCrashed) || strings.Contains(out.String(), "resume") {
		t.Errorf("after a crash: %v, and Out holds %q", r.err, out.String())
	}
}

func TestRunOpensAndClosesTTY(t *testing.T) {
	tty := func() Decision {
		d := interactive()
		d.In, d.UI = TargetTTY, TargetTTY
		return d
	}
	for _, c := range []struct {
		name     string
		distinct bool
		m        tm
		failOpen bool
		failNew  bool
		wantErr  error
	}{
		{name: "one file, quit", m: tm{quitOnSize: true}},
		{name: "two files, quit", distinct: true, m: tm{quitOnSize: true}},
		{name: "two files, crash", distinct: true, m: tm{panicOn: "view"}, wantErr: ErrCrashed},
		{name: "two files, not started", distinct: true, failNew: true, wantErr: ErrNotStarted},
		{name: "the open fails", failOpen: true, wantErr: ErrNotStarted},
	} {
		t.Run(c.name, func(t *testing.T) {
			countSaves(t)
			if c.failNew {
				old := newProgram
				newProgram = func(_ tea.Model, o ...tea.ProgramOption) *tea.Program { return old(nil, o...) }
				t.Cleanup(func() { newProgram = old })
			}
			ttyIn := terminal(100, 50)
			ttyOut := ttyIn
			if c.distinct {
				ttyOut = terminal(100, 50)
			}
			o := &ttyOpener{in: ttyIn, out: ttyOut, fail: c.failOpen}
			out := terminal(80, 24)
			r := within(t, 5*time.Second, func() result {
				got, err := runWith(t.Context(), Streams{In: pipe(), Out: out}, tty(), c.m, o.open, nil)
				return result{got, err}
			})
			if c.wantErr == nil && r.err != nil || c.wantErr != nil && !errors.Is(r.err, c.wantErr) {
				t.Fatalf("err = %v, want %v", r.err, c.wantErr)
			}
			if o.opens != 1 {
				t.Errorf("opened %d times, want 1", o.opens)
			}
			want := 1
			if c.failOpen {
				want = 0
			}
			if ttyIn.closes() != want || c.distinct && ttyOut.closes() != want {
				t.Errorf("closes: in %d, out %d; want %d each", ttyIn.closes(), ttyOut.closes(), want)
			}
			if out.String() != "" {
				t.Errorf("Out was written, though the TUI drew on the terminal: %q", out.String())
			}
		})
	}
}

func TestWithFilterSeesTheModel(t *testing.T) {
	var sawModel atomic.Bool
	r := runTM(t, t.Context(), Streams{In: io.MultiReader(strings.NewReader("axbq"), blocking(t)), Out: terminal(80, 24)},
		interactive(), tm{quitOnKey: "q"},
		WithFilter(func(m tm, msg tea.Msg) tea.Msg {
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
type becomes struct{ tm }

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
		b, err := Run(t.Context(), Streams{In: blocking(t), Out: terminal(80, 24)}, interactive(), becomes{tm{keys: "kept"}})
		return out{b, err}
	})
	if r.err == nil || !strings.Contains(r.err.Error(), "launch.other") || !strings.Contains(r.err.Error(), "launch.becomes") {
		t.Errorf("err = %v; want one naming both types", r.err)
	}
	if r.b.keys != "kept" {
		t.Errorf("the model is %+v; want the last good one", r.b)
	}
}
