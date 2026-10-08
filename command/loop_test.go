package command

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// The registry's loop, attached and detached
// (docs/decisions/0013-PLAN-cli-integration-helpers.md Step 2c).

type runOutcome struct {
	res Result
	err error
}

// runOffLoop runs req on r from a goroutine of its own, as an agent or a CLI
// does, and delivers what Run returned.
func runOffLoop(ctx context.Context, r *Registry, req Request) <-chan runOutcome {
	got := make(chan runOutcome, 1)
	go func() {
		res, err := r.Run(ctx, req)
		got <- runOutcome{res, err}
	}()
	return got
}

// inSecond receives from ch, failing t after one second.
func inSecond[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(time.Second):
		t.Fatal(what)
		var zero T
		return zero
	}
}

// counting is a Loop command whose handler counts its runs.
func counting(n *atomic.Int32, text string) Command {
	return cmd("loop", UI, func(c *Command) {
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			n.Add(1)
			return Result{Text: text}, nil
		})
	})
}

func TestAttachRunsOnLoop(t *testing.T) {
	var onLoop atomic.Bool // true only while the test's loop runs a message
	var sawLoop atomic.Bool
	r := registryOf(t, nil, cmd("loop", UI, func(c *Command) {
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			sawLoop.Store(onLoop.Load())
			return Result{Text: "done"}, nil
		})
	}))
	sent := make(chan tea.Msg, 1)
	detach := r.Attach(func(m tea.Msg) { sent <- m })
	defer detach()

	got := runOffLoop(t.Context(), r, Request{ID: "loop", Origin: OriginAgent})
	lm, ok := inSecond(t, sent, "Run sent nothing to the attached loop").(LoopMsg)
	if !ok {
		t.Fatal("Run sent something other than a LoopMsg")
	}
	onLoop.Store(true)
	lm.Run()
	onLoop.Store(false)
	if out := inSecond(t, got, "Run did not return"); out.err != nil || out.res.Text != "done" {
		t.Errorf("Run = %+v, %v", out.res, out.err)
	}
	if !sawLoop.Load() {
		t.Error("the handler did not run on the loop")
	}
}

func TestDetachRunsOnCaller(t *testing.T) {
	var n atomic.Int32
	r := registryOf(t, nil, counting(&n, "here"))
	// A loop that never takes its message, as a program that has ended
	// before it reads one, or never started.
	never := make(chan struct{})
	t.Cleanup(func() { close(never) })
	sent := make(chan struct{}, 1)
	detach := r.Attach(func(tea.Msg) { sent <- struct{}{}; <-never })

	got := runOffLoop(context.Background(), r, Request{ID: "loop", Origin: OriginAgent})
	inSecond(t, sent, "Run sent nothing to the attached loop")
	detach()
	out := inSecond(t, got, "Run still waits on a detached loop")
	if out.err != nil || out.res.Text != "here" || n.Load() != 1 {
		t.Errorf("Run = %+v, %v, with %d runs; want one run on the caller", out.res, out.err, n.Load())
	}
	detach() // a second detach does nothing

	// Detached, the registry has no loop, and runs on the caller at once.
	if res, err := r.Run(t.Context(), Request{ID: "loop", Origin: OriginAgent}); err != nil || res.Text != "here" || n.Load() != 2 {
		t.Errorf("after detach: %+v, %v, with %d runs", res, err, n.Load())
	}
}

func TestDetachRaceRunsOnce(t *testing.T) {
	var n atomic.Int32
	r := registryOf(t, nil, counting(&n, "once"))
	for i := range 1000 {
		n.Store(0)
		var delivered sync.WaitGroup
		var detach func()
		ready := make(chan struct{})
		detach = r.Attach(func(m tea.Msg) {
			delivered.Add(1)
			defer delivered.Done()
			<-ready
			go detach() // detach fires while the loop takes the message
			m.(LoopMsg).Run()
		})
		close(ready)
		out := inSecond(t, runOffLoop(t.Context(), r, Request{ID: "loop", Origin: OriginAgent}), "Run did not return")
		detach()
		delivered.Wait()
		if out.err != nil || out.res.Text != "once" {
			t.Fatalf("iteration %d: Run = %+v, %v", i, out.res, out.err)
		}
		if got := n.Load(); got != 1 {
			t.Fatalf("iteration %d: the handler ran %d times, want exactly 1", i, got)
		}
	}
}

// recording is an Auditor that keeps every record.
type recording struct {
	mu   sync.Mutex
	recs []Record
}

func (a *recording) Audit(rec Record) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.recs = append(a.recs, rec)
}

func TestCancelAfterStartKeepsOutcome(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	audit := &recording{}
	r := registryOf(t, []RegistryOption{WithAuditor(audit)}, cmd("loop", UI, func(c *Command) {
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			close(started)
			<-release
			return Result{Text: "kept"}, nil
		})
	}))
	sent := make(chan tea.Msg, 1)
	detach := r.Attach(func(m tea.Msg) { sent <- m })
	defer detach()

	ctx, cancel := context.WithCancel(t.Context())
	got := runOffLoop(ctx, r, Request{ID: "loop", Origin: OriginAgent})
	lm := inSecond(t, sent, "Run sent nothing").(LoopMsg)
	go lm.Run()
	inSecond(t, started, "the loop did not start the command")
	cancel() // the caller gives up after the loop has claimed the run
	time.Sleep(10 * time.Millisecond)
	close(release)
	out := inSecond(t, got, "Run did not return")
	if out.err != nil || out.res.Text != "kept" {
		t.Errorf("Run = %+v, %v; want the handler's outcome, not the cancellation", out.res, out.err)
	}
	audit.mu.Lock()
	defer audit.mu.Unlock()
	if len(audit.recs) != 1 || audit.recs[0].Err != nil {
		t.Errorf("audit = %+v; want one record of the kept outcome", audit.recs)
	}
}

func TestLoopPanicReleasesCaller(t *testing.T) {
	r := registryOf(t, nil, cmd("loop", UI, func(c *Command) {
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			panic("boom")
		})
	}))
	sent := make(chan tea.Msg, 1)
	detach := r.Attach(func(m tea.Msg) { sent <- m })
	defer detach()

	got := runOffLoop(t.Context(), r, Request{ID: "loop", Origin: OriginAgent})
	lm := inSecond(t, sent, "Run sent nothing").(LoopMsg)
	recovered := make(chan any, 1)
	go func() {
		// The program's loop, which recovers a panic as Bubble Tea does.
		defer func() { recovered <- recover() }()
		lm.Run()
	}()
	out := inSecond(t, got, "a panic on the loop left Run waiting")
	if out.err == nil || !strings.Contains(out.err.Error(), "panicked on the loop: boom") {
		t.Errorf("Run's error = %v; want one naming the panic", out.err)
	}
	if p := inSecond(t, recovered, "the loop did not finish"); p != "boom" {
		t.Errorf("the loop recovered %v; want the panic passed on", p)
	}
}

// TestWithLoopUnchanged: WithLoop is a permanent Attach. TestRunOnLoop and
// TestDispatchLoop hold its behaviour, unchanged by Step 2c; this test adds
// that it is never detached, and that Attach and detach replace it and
// leave no loop.
func TestWithLoopUnchanged(t *testing.T) {
	var n atomic.Int32
	sent := make(chan tea.Msg, 4)
	r := registryOf(t, []RegistryOption{WithLoop(func(m tea.Msg) { sent <- m })}, counting(&n, "x"))
	for range 2 {
		got := runOffLoop(t.Context(), r, Request{ID: "loop", Origin: OriginAgent})
		inSecond(t, sent, "WithLoop's loop got nothing").(LoopMsg).Run()
		inSecond(t, got, "Run did not return")
	}
	detach := r.Attach(func(tea.Msg) { t.Error("the attached loop was used after detach") })
	detach()
	if res, err := r.Run(t.Context(), Request{ID: "loop", Origin: OriginAgent}); err != nil || res.Text != "x" || n.Load() != 3 {
		t.Errorf("after Attach and detach: %+v, %v, %d runs", res, err, n.Load())
	}
}
