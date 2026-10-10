package command

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/internal/teamsg"
	"github.com/maccavelli/go-tui-lib/when"
)

// Dispatch runs a request from a Bubble Tea program's Update. It finds the
// command, checks its When and Surfaces against the request, and asks the
// policy, on the calling goroutine. A Loop command then runs at once, and
// Dispatch returns its Result.Cmd batched with a ResultMsg. An Async
// command runs only when the returned tea.Cmd runs, under a context
// Cancel reaches. A Prompt or Forward command also sends a PromptMsg. A
// request that cannot run returns a ResultMsg whose Err says why.
func (r *Registry) Dispatch(ctx context.Context, req Request) tea.Cmd {
	e, inv, d, err := r.admit(ctx, req)
	if err != nil {
		r.audit(req, e, d, outcome{started: time.Now()}, err)
		return teamsg.Cmd(ResultMsg{Request: req, Err: err})
	}
	if inv.Command.Mode == Async {
		return func() tea.Msg {
			o := r.execute(ctx, inv)
			r.audit(req, e, d, o, o.err)
			return tea.Batch(effects(req, inv, o)...)()
		}
	}
	o := r.execute(ctx, inv)
	r.audit(req, e, d, o, o.err)
	return tea.Batch(effects(req, inv, o)...)
}

// Run runs a request to completion, whatever the command's mode, with the
// checks Dispatch makes, and returns its Result; the caller decides what
// to do with Result.Cmd. It is the shell's, agents' and tests' way in. An
// Async command's run is one Cancel reaches.
//
// With WithLoop, a Loop command runs on the program's event loop: Run
// sends a LoopMsg and waits until the loop has run it or ctx ends. Its
// Result.Cmd goes to the program, and the Result Run returns has none.
// Without WithLoop, a Loop command runs on Run's caller.
func (r *Registry) Run(ctx context.Context, req Request) (Result, error) {
	e, inv, d, err := r.admit(ctx, req)
	if err != nil {
		r.audit(req, e, d, outcome{started: time.Now()}, err)
		return Result{}, err
	}
	var o outcome
	if l := r.loop.Load(); inv.Command.Mode == Loop && l != nil {
		o = r.onLoop(ctx, inv, l)
	} else {
		o = r.execute(ctx, inv)
	}
	r.audit(req, e, d, o, o.err)
	return o.res, o.err
}

// Attach makes send the registry's loop until detach is called. A Loop
// command started while attached runs on the loop; once detached, or when
// the loop does not take it, it runs on the caller instead, exactly once.
//
// send is a program's Send. A LoopMsg is sent from a goroutine of its own,
// which ends when send returns, and detach cannot end a send that is
// blocked: Bubble Tea's Send returns once the program has ended, so detach
// after the program's Run has returned, never before it has started. A
// command that runs on the caller after detach keeps its Result.Cmd, which
// no program will run.
//
// Attach replaces the current loop, WithLoop's included, and detach then
// leaves the registry with none. A second detach does nothing.
func (r *Registry) Attach(send func(tea.Msg)) (detach func()) {
	l := &loopState{send: send, done: make(chan struct{})}
	r.mu.Lock()
	r.loop.Store(l)
	r.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			r.mu.Lock()
			r.loop.CompareAndSwap(l, nil)
			r.mu.Unlock()
			close(l.done)
		})
	}
}

// onLoop runs inv on l through a LoopMsg, and waits for its outcome. Either
// the loop or this caller claims the run, never both:
//   - the LoopMsg runs the command only if it wins the claim;
//   - when l is detached, a command the loop has not claimed runs here;
//   - when ctx ends, a command the loop has not claimed is cancelled, and
//     one it has claimed is waited for, so its outcome is kept.
//
// A LoopMsg the loop reaches after the claim was lost runs nothing.
func (r *Registry) onLoop(ctx context.Context, inv *Invocation, l *loopState) outcome {
	var claimed atomic.Bool
	done := make(chan outcome, 1)
	msg := LoopMsg{run: func() tea.Cmd {
		if !claimed.CompareAndSwap(false, true) {
			return nil
		}
		// execute recovers the handler's panic into the outcome, so the
		// outcome is always sent, and the loop never sees the panic
		// (docs/decisions/0014-PLAN-component-native-forms.md Step 2).
		if err := ctx.Err(); err != nil {
			done <- outcome{started: time.Now(), err: err}
			return nil
		}
		o := r.execute(ctx, inv)
		cmd := o.res.Cmd
		o.res.Cmd = nil
		done <- o
		return cmd
	}}
	go l.send(msg) // Send blocks until the program takes it, or ends
	select {
	case o := <-done:
		return o
	case <-l.done:
		if claimed.CompareAndSwap(false, true) {
			return r.execute(ctx, inv)
		}
		return <-done
	case <-ctx.Done():
		if claimed.CompareAndSwap(false, true) {
			return outcome{started: time.Now(), err: ctx.Err()}
		}
		return <-done
	}
}

// Cancel cancels the running async invocations of id.
func (r *Registry) Cancel(id ID) { r.running.cancel(id) }

// CancelAll cancels every running async invocation.
func (r *Registry) CancelAll() { r.running.cancelAll() }

// admit finds req's command and decides whether it may run: the origin is
// set, the command is offered on the origin's surface, its When holds, its
// arguments fit its schema (and gain its defaults), and the policy allows
// it. The entry is nil when no command has the ID.
func (r *Registry) admit(ctx context.Context, req Request) (*entry, *Invocation, Verdict, error) {
	s := r.snap.Load()
	i, ok := s.byID[req.ID]
	if !ok {
		return nil, nil, 0, fmt.Errorf("%w: %q", ErrUnknown, req.ID)
	}
	e := &s.entries[i]
	// The size first, before anything reads the arguments.
	if err := r.tooLarge(max(len(req.Args), len(req.Raw))); err != nil {
		return e, nil, 0, err
	}
	c := req.WhenContext
	if c == nil {
		c = req.Context // the deprecated field, through v0.9.x
	}
	if c == nil {
		c = when.Map(nil)
	}
	inv := &Invocation{
		Command: e.cmd, Args: req.Args, Raw: req.Raw,
		Origin: req.Origin, Caller: req.Caller, WhenContext: c, Context: c,
	}
	if req.Origin == 0 || req.Origin > OriginProgram {
		return e, nil, 0, fmt.Errorf("%w: %s: the request's origin is %s", ErrRefused, req.ID, req.Origin)
	}
	if sf := req.Origin.surface(); sf != 0 && e.cmd.surfaces()&sf == 0 {
		return e, nil, 0, fmt.Errorf("%w: %s is not offered on the %s surface", ErrUnavailable, req.ID, sf)
	}
	if !e.holds(c) {
		return e, nil, 0, fmt.Errorf("%w: %s: when %q is false", ErrUnavailable, req.ID, e.cmd.When)
	}
	if e.args != nil {
		args, err := e.args.prepare(req.Args)
		if err != nil {
			return e, nil, 0, err
		}
		inv.Args = args
	}
	d, err := r.policy.decide(ctx, inv, req.Gate)
	if err != nil {
		return e, nil, d, err
	}
	return e, inv, d, nil
}

// outcome is one run of a handler.
type outcome struct {
	res     Result
	err     error
	started time.Time
	dur     time.Duration
}

// execute runs inv: its handler, or for a Forward its slash text. An
// Async command runs under a context Cancel reaches, and an Exclusive one
// cancels its earlier runs first. A handler's panic is the outcome's
// error, a *PanicError.
func (r *Registry) execute(ctx context.Context, inv *Invocation) outcome {
	c := &inv.Command
	if c.Mode == Async {
		var done func()
		ctx, done = r.running.start(ctx, c.ID, c.Exclusive)
		defer done()
	}
	o := outcome{started: time.Now()}
	if c.Kind == Forward {
		o.res = Result{Text: forwardText(inv)}
	} else {
		o.res, o.err = runHandler(ctx, c, inv)
	}
	o.dur = time.Since(o.started)
	return o
}

// forwardText is a Forward command as the agent reads it: "/name args",
// with the agent's own name from Meta["acp"] when FromACP kept it
// (docs/decisions/0006-MADR-command-registry.md A6), else the slash name,
// else the ID.
func forwardText(inv *Invocation) string {
	name := inv.Command.Slash
	if acp, ok := inv.Command.Meta[metaACP].(map[string]any); ok {
		if n, ok := acp["name"].(string); ok && n != "" {
			name = n
		}
	}
	if name == "" {
		name = string(inv.Command.ID)
	}
	if raw := strings.TrimSpace(inv.Raw); raw != "" {
		return "/" + name + " " + raw
	}
	return "/" + name
}

// effects is what a run sends to the program: its Result.Cmd, a
// ResultMsg, and for a Prompt or Forward that succeeded a PromptMsg.
func effects(req Request, inv *Invocation, o outcome) []tea.Cmd {
	cmds := []tea.Cmd{o.res.Cmd, teamsg.Cmd(ResultMsg{Request: req, Result: o.res, Err: o.err, Duration: o.dur})}
	if o.err == nil && inv.Command.Kind != Action {
		cmds = append(cmds, teamsg.Cmd(PromptMsg{Text: o.res.Text, Command: inv.Command, Request: req}))
	}
	return cmds
}

// audit records a request when there is an auditor, with the values its
// command's schema marks secret masked. e is nil for an unknown command.
func (r *Registry) audit(req Request, e *entry, d Verdict, o outcome, err error) {
	if r.auditor == nil {
		return
	}
	args := req.Args
	switch {
	case r.tooLarge(len(args)) != nil:
		args = nil // too large to record, or to read for masking
	case e != nil:
		args = e.args.mask(args)
	}
	r.auditor.Audit(Record{
		ID: req.ID, Origin: req.Origin, Caller: req.Caller, Args: args,
		Verdict: d, Decision: d, Started: o.started, Duration: o.dur, Err: err,
	})
}

// running is the set of async invocations in flight, by command.
type running struct {
	mu   sync.Mutex
	next uint64
	runs map[ID]map[uint64]context.CancelFunc
}

// start adds a run of id under a context derived from ctx, cancelling id's
// other runs first when exclusive. done removes it and releases its
// context.
func (rn *running) start(ctx context.Context, id ID, exclusive bool) (_ context.Context, done func()) {
	ctx, cancel := context.WithCancel(ctx)
	rn.mu.Lock()
	if exclusive {
		rn.cancelLocked(id)
	}
	rn.next++
	n := rn.next
	if rn.runs[id] == nil {
		rn.runs[id] = map[uint64]context.CancelFunc{}
	}
	rn.runs[id][n] = cancel
	rn.mu.Unlock()
	return ctx, func() {
		rn.mu.Lock()
		if m := rn.runs[id]; m != nil {
			delete(m, n)
			if len(m) == 0 {
				delete(rn.runs, id)
			}
		}
		rn.mu.Unlock()
		cancel()
	}
}

func (rn *running) cancelLocked(id ID) {
	for _, cancel := range rn.runs[id] {
		cancel()
	}
	delete(rn.runs, id)
}

func (rn *running) cancel(id ID) {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	rn.cancelLocked(id)
}

func (rn *running) cancelAll() {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	for id := range rn.runs {
		rn.cancelLocked(id)
	}
}
