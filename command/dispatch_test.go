package command

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

type doneMsg struct{}

func TestDispatchLoop(t *testing.T) {
	var ran atomic.Bool
	r := registryOf(t, nil, cmd("a", UI, func(c *Command) {
		c.When = "!off"
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			ran.Store(true)
			return Result{Value: 7, Text: "done", Cmd: msgCmd(doneMsg{})}, nil
		})
	}))
	c := r.Dispatch(t.Context(), Request{ID: "a", Origin: OriginKey})
	if !ran.Load() {
		t.Fatal("a Loop command did not run during Dispatch")
	}
	msgs := collect(c)
	if !slices.Contains(msgs, tea.Msg(doneMsg{})) {
		t.Errorf("Result.Cmd's message is not among %v", msgs)
	}
	if res := resultOf(t, msgs); res.Err != nil || res.Result.Text != "done" || res.Result.Value != 7 {
		t.Errorf("ResultMsg = %+v", res)
	}
	for _, m := range msgs {
		if _, ok := m.(PromptMsg); ok {
			t.Error("an Action sent a PromptMsg")
		}
	}
	ran.Store(false)
	res := resultOf(t, collect(r.Dispatch(t.Context(), Request{ID: "a", Origin: OriginKey, Context: mapOf("off")})))
	if ran.Load() || !errors.Is(res.Err, ErrUnavailable) {
		t.Errorf("with its When false: ran %v, %v", ran.Load(), res.Err)
	}
}

func TestDispatchAsync(t *testing.T) {
	var ran atomic.Bool
	r := registryOf(t, nil, cmd("a", UI, func(c *Command) {
		c.Mode = Async
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			ran.Store(true)
			return Result{Text: "done"}, nil
		})
	}))
	c := r.Dispatch(t.Context(), Request{ID: "a", Origin: OriginKey})
	if ran.Load() {
		t.Fatal("an Async command ran during Dispatch")
	}
	res := resultOf(t, collect(c))
	if !ran.Load() || res.Result.Text != "done" {
		t.Errorf("after the tea.Cmd ran: ran %v, %+v", ran.Load(), res)
	}
}

func TestRunMatchesDispatch(t *testing.T) {
	fail := errors.New("failed")
	for _, mode := range []Mode{Loop, Async} {
		for _, err := range []error{nil, fail} {
			r := registryOf(t, nil, cmd("a", UI, func(c *Command) {
				c.Mode = mode
				c.Handler = HandlerFunc(func(_ context.Context, inv *Invocation) (Result, error) {
					return Result{Value: map[string]any{"args": string(inv.Args)}, Text: inv.Raw}, err
				})
			}))
			req := Request{ID: "a", Args: []byte(`{"x":1}`), Raw: "x=1", Origin: OriginSlash}
			got, gerr := r.Run(t.Context(), req)
			res := resultOf(t, collect(r.Dispatch(t.Context(), req)))
			if !reflect.DeepEqual(got, res.Result) || !errors.Is(gerr, err) || !errors.Is(res.Err, err) {
				t.Errorf("%s, %v: Run %+v, %v; Dispatch %+v, %v", mode, err, got, gerr, res.Result, res.Err)
			}
		}
	}
}

func TestPromptAndForward(t *testing.T) {
	r := registryOf(t, nil,
		cmd("p", UI, func(c *Command) {
			c.Kind = Prompt
			c.Handler = HandlerFunc(func(_ context.Context, inv *Invocation) (Result, error) {
				return Result{Text: "review " + inv.Raw}, nil
			})
		}),
		cmd("acp.agent.review", UI, func(c *Command) { c.Kind, c.Handler, c.Slash = Forward, nil, "review" }),
		cmd("acp.agent.plan", UI, func(c *Command) { c.Kind, c.Handler = Forward, nil }),
	)
	for id, want := range map[ID]string{"p": "review a b", "acp.agent.review": "/review a b", "acp.agent.plan": "/acp.agent.plan a b"} {
		req := Request{ID: id, Raw: " a b ", Origin: OriginSlash}
		if id == "p" {
			req.Raw = "a b"
		}
		msgs := collect(r.Dispatch(t.Context(), req))
		var prompts []PromptMsg
		for _, m := range msgs {
			if p, ok := m.(PromptMsg); ok {
				prompts = append(prompts, p)
			}
		}
		if len(prompts) != 1 || prompts[0].Text != want || prompts[0].Command.ID != id || prompts[0].Request.ID != id {
			t.Errorf("%s: PromptMsgs %+v, want one with %q", id, prompts, want)
		}
		if res, err := r.Run(t.Context(), req); err != nil || res.Text != want {
			t.Errorf("%s: Run = %q, %v", id, res.Text, err)
		}
	}
}

// blocker is an async command that reports its start on started and runs
// until its context ends.
func blocker(id ID, exclusive bool, started chan<- ID) Command {
	return cmd(id, UI, func(c *Command) {
		c.Mode, c.Exclusive = Async, exclusive
		c.Handler = HandlerFunc(func(ctx context.Context, inv *Invocation) (Result, error) {
			started <- inv.Command.ID
			<-ctx.Done()
			return Result{}, ctx.Err()
		})
	})
}

// runAsync runs c's message in a goroutine and returns a function that
// waits for its ResultMsg's error.
func runAsync(t *testing.T, c tea.Cmd) func() error {
	ch := make(chan []tea.Msg, 1)
	go func() { ch <- collect(c) }()
	return func() error {
		t.Helper()
		select {
		case msgs := <-ch:
			return resultOf(t, msgs).Err
		case <-time.After(5 * time.Second):
			t.Fatal("the async command did not end")
			return nil
		}
	}
}

func TestCancel(t *testing.T) {
	started := make(chan ID, 4)
	r := registryOf(t, nil, blocker("a", false, started), blocker("b", false, started))
	a := runAsync(t, r.Dispatch(t.Context(), Request{ID: "a", Origin: OriginKey}))
	b := runAsync(t, r.Dispatch(t.Context(), Request{ID: "b", Origin: OriginKey}))
	<-started
	<-started
	r.Cancel("a")
	if err := a(); !errors.Is(err, context.Canceled) {
		t.Errorf("a after Cancel(a): %v", err)
	}
	r.Cancel("missing")
	runErr := make(chan error, 1)
	go func() {
		_, err := r.Run(t.Context(), Request{ID: "a", Origin: OriginAgent})
		runErr <- err
	}()
	<-started
	r.CancelAll()
	if err := b(); !errors.Is(err, context.Canceled) {
		t.Errorf("b after CancelAll: %v", err)
	}
	if err := <-runErr; !errors.Is(err, context.Canceled) {
		t.Errorf("Run of a after CancelAll: %v", err)
	}
}

func TestExclusive(t *testing.T) {
	started := make(chan ID, 4)
	r := registryOf(t, nil, blocker("x", true, started), blocker("y", false, started))
	x1 := runAsync(t, r.Dispatch(t.Context(), Request{ID: "x", Origin: OriginKey}))
	<-started
	x2 := runAsync(t, r.Dispatch(t.Context(), Request{ID: "x", Origin: OriginKey}))
	<-started
	if err := x1(); !errors.Is(err, context.Canceled) {
		t.Errorf("the first exclusive run after the second began: %v", err)
	}
	y1 := runAsync(t, r.Dispatch(t.Context(), Request{ID: "y", Origin: OriginKey}))
	<-started
	y2 := runAsync(t, r.Dispatch(t.Context(), Request{ID: "y", Origin: OriginKey}))
	<-started
	r.Cancel("x")
	if err := x2(); !errors.Is(err, context.Canceled) {
		t.Errorf("the second exclusive run after Cancel: %v", err)
	}
	// A non-exclusive command's runs go on side by side until cancelled.
	r.Cancel("y")
	if e1, e2 := y1(), y2(); !errors.Is(e1, context.Canceled) || !errors.Is(e2, context.Canceled) {
		t.Errorf("the non-exclusive runs: %v, %v", e1, e2)
	}
}

func TestNonExclusiveRunsTogether(t *testing.T) {
	started := make(chan ID, 4)
	r := registryOf(t, nil, blocker("y", false, started))
	y1 := runAsync(t, r.Dispatch(t.Context(), Request{ID: "y", Origin: OriginKey}))
	<-started
	_ = runAsync(t, r.Dispatch(t.Context(), Request{ID: "y", Origin: OriginKey}))
	<-started
	r.running.mu.Lock()
	n := len(r.running.runs["y"])
	r.running.mu.Unlock()
	if n != 2 {
		t.Errorf("%d runs of y in flight, want 2", n)
	}
	r.CancelAll()
	if err := y1(); !errors.Is(err, context.Canceled) {
		t.Errorf("y1: %v", err)
	}
}
