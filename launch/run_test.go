package launch

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/command"
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
