package commandtea

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
)

type (
	aMsg   struct{}
	bMsg   struct{}
	effMsg struct{}
)

// run executes cmd and every command a tea.BatchMsg it returns holds, in
// order, and returns the messages.
func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	m := cmd()
	if b, ok := m.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range b {
			out = append(out, run(c)...)
		}
		return out
	}
	return []tea.Msg{m}
}

// registry is a registry holding cmds.
func registry(t *testing.T, o []command.RegistryOption, cmds ...command.Command) *command.Registry {
	t.Helper()
	r := command.NewRegistry(o...)
	if err := r.Register(cmds...); err != nil {
		t.Fatal(err)
	}
	return r
}

// ui is a UI command id whose handler returns res, and runs in mode.
func ui(id command.ID, mode command.Mode, res command.Result, ran *atomic.Int32) command.Command {
	return command.Command{
		ID: id, Title: string(id), Danger: command.UI, Mode: mode,
		Handler: command.HandlerFunc(func(context.Context, *command.Invocation) (command.Result, error) {
			if ran != nil {
				ran.Add(1)
			}
			return res, nil
		}),
	}
}

// TestCmdBatches: an Effect whose value is two Effects gives a
// tea.BatchMsg of two commands, whose messages are theirs, in order; a
// nil one among them is skipped, and a batch of nils is no message.
func TestCmdBatches(t *testing.T) {
	a := func() any { return aMsg{} }
	b := func() any { return bMsg{} }
	msg := Cmd(func() any { return []command.Effect{a, b} })()
	batch, ok := msg.(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("a batch of two: %#v, want a tea.BatchMsg of two", msg)
	}
	if batch[0]() != (aMsg{}) || batch[1]() != (bMsg{}) {
		t.Errorf("the batch's messages: %v, %v", batch[0](), batch[1]())
	}
	if got := run(Cmd(func() any { return []command.Effect{nil, a, nil} })); !reflect.DeepEqual(got, []tea.Msg{aMsg{}}) {
		t.Errorf("a batch with nils: %v, want aMsg alone", got)
	}
	if got := Cmd(func() any { return []command.Effect{nil} })(); got != nil {
		t.Errorf("a batch of nils: %v, want no message", got)
	}
	nested := Cmd(func() any { return []command.Effect{func() any { return []command.Effect{a, b} }, a} })
	if got := run(nested); !reflect.DeepEqual(got, []tea.Msg{aMsg{}, bMsg{}, aMsg{}}) {
		t.Errorf("a nested batch: %v", got)
	}
	if got := Cmd(a)(); got != (aMsg{}) {
		t.Errorf("one message: %v", got)
	}
}

// TestEffectNil: nil stays nil both ways, and a command and its Effect
// give the same message.
func TestEffectNil(t *testing.T) {
	if Effect(nil) != nil {
		t.Error("Effect(nil) is not nil")
	}
	if Cmd(nil) != nil {
		t.Error("Cmd(nil) is not nil")
	}
	c := func() tea.Msg { return aMsg{} }
	if got := Cmd(Effect(c))(); got != (aMsg{}) {
		t.Errorf("a round trip: %v", got)
	}
	if got := Effect(tea.Batch(c, c))(); reflect.TypeOf(got) != reflect.TypeFor[tea.BatchMsg]() {
		t.Errorf("a tea.Batch as an Effect gives %T, want its tea.BatchMsg", got)
	}
}

// TestDispatch: a Loop command's effect is one tea.BatchMsg of its
// Result.Cmd's message and its ResultMsg, a Prompt's with its PromptMsg
// too, and an unknown command's a ResultMsg alone.
func TestDispatch(t *testing.T) {
	prompt := ui("p", command.Loop, command.Result{Text: "say hi"}, nil)
	prompt.Kind = command.Prompt
	r := registry(t, nil, ui("a", command.Loop, command.Result{Text: "done", Cmd: Effect(func() tea.Msg { return effMsg{} })}, nil), prompt)
	msg := Dispatch(t.Context(), r, command.Request{ID: "a", Origin: command.OriginKey})()
	if _, ok := msg.(tea.BatchMsg); !ok {
		t.Fatalf("Dispatch's message is %T, want a tea.BatchMsg", msg)
	}
	got := run(func() tea.Msg { return msg })
	if len(got) != 2 || got[0] != (effMsg{}) {
		t.Fatalf("Dispatch sent %v, want the effect's message, then the ResultMsg", got)
	}
	if res, ok := got[1].(command.ResultMsg); !ok || res.Result.Text != "done" || res.Err != nil {
		t.Errorf("then %+v", got[1])
	}
	p := run(Dispatch(t.Context(), r, command.Request{ID: "p", Origin: command.OriginKey}))
	if len(p) != 2 {
		t.Fatalf("a prompt sent %v, want a ResultMsg and a PromptMsg", p)
	}
	if pm, ok := p[1].(command.PromptMsg); !ok || pm.Text != "say hi" {
		t.Errorf("the prompt's second message %+v", p[1])
	}
	u := run(Dispatch(t.Context(), r, command.Request{ID: "missing", Origin: command.OriginKey}))
	if len(u) != 1 {
		t.Fatalf("an unknown command sent %v", u)
	}
	if res, ok := u[0].(command.ResultMsg); !ok || res.Err == nil {
		t.Errorf("an unknown command: %+v", u[0])
	}
}

// TestDispatchAsync: an Async command runs only when the tea.Cmd runs.
func TestDispatchAsync(t *testing.T) {
	var ran atomic.Int32
	r := registry(t, nil, ui("a", command.Async, command.Result{Text: "later"}, &ran))
	cmd := Dispatch(t.Context(), r, command.Request{ID: "a", Origin: command.OriginKey})
	if ran.Load() != 0 {
		t.Fatal("the async command ran before its tea.Cmd did")
	}
	got := run(cmd)
	if ran.Load() != 1 || len(got) != 1 {
		t.Fatalf("after the tea.Cmd: ran %d times, sent %v", ran.Load(), got)
	}
	if res, ok := got[0].(command.ResultMsg); !ok || res.Result.Text != "later" {
		t.Errorf("the async result %+v", got[0])
	}
}

// within runs cmd in a goroutine, and gives its message, or fails after d.
func within(t *testing.T, cmd tea.Cmd, d time.Duration) tea.Msg {
	t.Helper()
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case m := <-ch:
		return m
	case <-time.After(d):
		t.Fatal("the command did not return")
		return nil
	}
}

// TestWatch: a change is ChangedMsg; a change that renamed a slash name
// is a tea.BatchMsg of ChangedMsg and ConflictMsg; a cancelled
// WatchContext gives no message.
func TestWatch(t *testing.T) {
	r := command.NewRegistry()
	w := Watch(r)
	if err := r.Register(ui("a", command.Loop, command.Result{}, nil)); err != nil {
		t.Fatal(err)
	}
	if m := within(t, w, 2*time.Second); m != (command.ChangedMsg{Version: 1}) {
		t.Errorf("after Register: %v", m)
	}

	clash := ui("builtin.review", command.Loop, command.Result{}, nil)
	clash.Slash = "review"
	if err := r.Register(clash); err != nil {
		t.Fatal(err)
	}
	w = Watch(r)
	loaded := ui("user.review", command.Loop, command.Result{}, nil)
	loaded.Slash, loaded.Kind = "review", command.Prompt
	r.ReplaceSource(command.Source{Kind: command.User, Name: "user"}, []command.Command{loaded})
	m := within(t, w, 2*time.Second)
	if _, ok := m.(tea.BatchMsg); !ok {
		t.Fatalf("a renaming change: %T, want a tea.BatchMsg", m)
	}
	got := run(func() tea.Msg { return m })
	if len(got) != 2 {
		t.Fatalf("a renaming change sent %v", got)
	}
	if _, ok := got[0].(command.ChangedMsg); !ok {
		t.Errorf("first %T, want ChangedMsg", got[0])
	}
	if c, ok := got[1].(command.ConflictMsg); !ok || len(c.Conflicts) != 1 {
		t.Errorf("then %+v, want one conflict", got[1])
	}

	ctx, cancel := context.WithCancel(t.Context())
	w = WatchContext(ctx, r)
	cancel()
	if m := within(t, w, 2*time.Second); m != nil {
		t.Errorf("a cancelled WatchContext: %v", m)
	}
}

// TestAttachAndRunLoop: a program's Send, through Attach, gets the
// LoopMsg of a Loop command Run hands over; RunLoop runs it on the loop,
// and its tea.Cmd is the command's Result.Cmd. WithLoop does the same.
func TestAttachAndRunLoop(t *testing.T) {
	var ran atomic.Int32
	res := command.Result{Text: "on the loop", Cmd: Effect(func() tea.Msg { return effMsg{} })}
	for _, how := range []string{"Attach", "WithLoop"} {
		sent := make(chan tea.Msg, 1)
		send := func(m tea.Msg) { sent <- m }
		var o []command.RegistryOption
		if how == "WithLoop" {
			o = append(o, WithLoop(send))
		}
		r := registry(t, o, ui("loop", command.Loop, res, &ran))
		if how == "Attach" {
			detach := Attach(r, send)
			defer detach()
		}
		before := ran.Load()
		done := make(chan command.Result, 1)
		go func() {
			got, _ := r.Run(t.Context(), command.Request{ID: "loop", Origin: command.OriginAgent})
			done <- got
		}()
		var lm command.LoopMsg
		select {
		case m := <-sent:
			var ok bool
			if lm, ok = m.(command.LoopMsg); !ok {
				t.Fatalf("%s: the loop got %T, want a LoopMsg", how, m)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: the loop got nothing", how)
		}
		if ran.Load() != before {
			t.Fatalf("%s: the command ran before the loop ran it", how)
		}
		if got := run(RunLoop(lm)); !reflect.DeepEqual(got, []tea.Msg{effMsg{}}) {
			t.Errorf("%s: RunLoop's tea.Cmd sent %v, want the effect's message", how, got)
		}
		select {
		case got := <-done:
			if got.Text != "on the loop" || got.Cmd != nil {
				t.Errorf("%s: Run returned %+v, want the text and no effect, which went to the loop", how, got)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: Run did not return", how)
		}
	}
	var zero command.LoopMsg
	if RunLoop(zero) != nil {
		t.Error("the zero LoopMsg gave a command")
	}
}
