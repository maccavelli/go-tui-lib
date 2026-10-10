package command

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ArgsOf, MustNew, GateFunc, AllowIf and WatchContext
// (docs/decisions/0014-PLAN-component-native-forms.md Step 3).

type waitArgs struct {
	Name string        `json:"name"`
	Wait time.Duration `json:"wait"`
}

func TestArgsOfDuration(t *testing.T) {
	raw, err := ArgsOf(waitArgs{Name: "x", Wait: 90 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"name":"x","wait":"1m30s"}` {
		t.Errorf("ArgsOf = %s", raw)
	}
	var got waitArgs
	r := registryOf(t, nil, MustNew("w", "Wait", func(_ context.Context, _ *Invocation, a waitArgs) (Result, error) {
		got = a
		return Result{}, nil
	}, WithDanger(UI)))
	if _, err := r.Run(t.Context(), Request{ID: "w", Args: raw, Origin: OriginCLI}); err != nil {
		t.Fatalf("New's rule refused ArgsOf's arguments: %v", err)
	}
	if got.Wait != 90*time.Second || got.Name != "x" {
		t.Errorf("the handler got %+v", got)
	}
}

func TestArgsOfDeterministic(t *testing.T) {
	m := map[string]int{"b": 2, "a": 1, "c": 3, "e": 5, "d": 4}
	first, err := ArgsOf(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != `{"a":1,"b":2,"c":3,"d":4,"e":5}` {
		t.Errorf("ArgsOf = %s; want sorted members", first)
	}
	for range 20 {
		if again, _ := ArgsOf(m); string(again) != string(first) {
			t.Fatalf("ArgsOf gave %s, then %s", first, again)
		}
	}
	if _, err := ArgsOf(func() {}); err == nil || !strings.HasPrefix(err.Error(), "command: arguments: ") {
		t.Errorf("ArgsOf(a func) = %v; want an error", err)
	}
}

func TestMustNewPanics(t *testing.T) {
	run := func(context.Context, *Invocation, NoArgs) (Result, error) { return Result{}, nil }
	if c := MustNew("ok", "OK", run, WithDanger(ReadOnly)); c.ID != "ok" || c.Danger != ReadOnly {
		t.Errorf("MustNew = %+v", c)
	}
	defer func() {
		p := recover()
		err, ok := p.(error)
		if !ok || !strings.Contains(err.Error(), "no WithDanger") {
			t.Errorf("MustNew without a danger panicked with %v; want New's error", p)
		}
	}()
	MustNew("bad", "Bad", run)
	t.Error("MustNew did not panic")
}

func TestAllowIf(t *testing.T) {
	r := registryOf(t, nil, cmd("rm", Destructive))
	for _, ok := range []bool{true, false} {
		d, err := AllowIf(ok).Decide(t.Context(), &Invocation{})
		want := RejectOnce
		if ok {
			want = AllowOnce
		}
		if d != want || err != nil {
			t.Errorf("AllowIf(%v) = %v, %v", ok, d, err)
		}
		_, err = r.Run(t.Context(), Request{ID: "rm", Origin: OriginCLI, Gate: AllowIf(ok)})
		if ok != (err == nil) || (!ok && !errors.Is(err, ErrRefused)) {
			t.Errorf("Run with AllowIf(%v): %v", ok, err)
		}
	}
}

func TestGateFunc(t *testing.T) {
	var saw *Invocation
	fail := errors.New("no")
	g := GateFunc(func(_ context.Context, inv *Invocation) (Verdict, error) {
		saw = inv
		return AllowAlways, fail
	})
	inv := &Invocation{Caller: "me"}
	if d, err := g.Decide(t.Context(), inv); d != AllowAlways || !errors.Is(err, fail) || saw != inv {
		t.Errorf("GateFunc.Decide = %v, %v, saw %p want %p", d, err, saw, inv)
	}
	var _ Gate = g
}

func TestWatchContextEnds(t *testing.T) {
	r := NewRegistry()
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(t.Context())
	got := make(chan any, 1)
	w := r.WatchContext(ctx)
	go func() { got <- w() }()
	select {
	case m := <-got:
		t.Fatalf("WatchContext returned %v before any change or cancel", m)
	case <-time.After(20 * time.Millisecond):
	}
	cancel()
	select {
	case m := <-got:
		if m != nil {
			t.Errorf("WatchContext returned %v on cancel; want nil", m)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("WatchContext did not return within 100 ms of cancel")
	}
	deadline := time.Now().Add(time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > before {
		t.Errorf("%d goroutines after cancel, %d before", n, before)
	}

	// A change still wakes it, as Watch.
	w = r.WatchContext(t.Context())
	if err := r.Register(cmd("a", UI)); err != nil {
		t.Fatal(err)
	}
	if m := w(); m != (ChangedMsg{Version: 1}) {
		t.Errorf("after Register: %v, want ChangedMsg{1}", m)
	}
}
