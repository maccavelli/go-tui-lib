package command

import (
	"context"
	"fmt"
	"sync"
)

// Decision is a gate's answer, with ACP's four permission option kinds.
// Its zero value is "no decision", which refuses.
type Decision uint8

// The decisions.
const (
	AllowOnce    Decision = iota + 1 // run this time
	AllowAlways                      // run, and do not ask again for this command and caller
	RejectOnce                       // refuse this time
	RejectAlways                     // refuse, and do not ask again for this command and caller
)

// ACPKind is d as ACP's PermissionOptionKind: "allow_once", "allow_always",
// "reject_once" or "reject_always"; "" for no decision.
func (d Decision) ACPKind() string {
	switch d {
	case AllowOnce:
		return "allow_once"
	case AllowAlways:
		return "allow_always"
	case RejectOnce:
		return "reject_once"
	case RejectAlways:
		return "reject_always"
	}
	return ""
}

func (d Decision) allows() bool { return d == AllowOnce || d == AllowAlways }

// Gate is asked before a command runs when the policy says to ask:
// typically a permission dialog. Dispatch asks it on the goroutine that
// calls Dispatch, and Run on Run's; a gate that waits for the user is
// reached through Run, from the agent's goroutine. A gate's panic refuses
// the request, with an error that wraps ErrRefused and a *PanicError.
type Gate interface {
	Decide(ctx context.Context, inv *Invocation) (Decision, error)
}

// GateFunc is a function used as a Gate.
type GateFunc func(ctx context.Context, inv *Invocation) (Decision, error)

// Decide calls f.
func (f GateFunc) Decide(ctx context.Context, inv *Invocation) (Decision, error) {
	return f(ctx, inv)
}

// AllowIf is a Gate that answers AllowOnce when ok, and RejectOnce
// otherwise: a program's own command line passes its --yes flag, as
// Request.Gate.
func AllowIf(ok bool) Gate {
	return GateFunc(func(context.Context, *Invocation) (Decision, error) {
		if ok {
			return AllowOnce, nil
		}
		return RejectOnce, nil
	})
}

// asks reports whether the policy asks the gate before a command of danger
// d runs for origin o: a mutating command from an agent, and a destructive
// one from an agent or the shell.
func asks(d Danger, o Origin) bool {
	switch d {
	case Mutating:
		return o == OriginAgent
	case Destructive:
		return o == OriginAgent || o == OriginCLI
	}
	return false
}

// alwaysKey is what AllowAlways and RejectAlways are remembered by.
type alwaysKey struct {
	id     ID
	caller string
}

// policy holds the gate and the remembered answers.
type policy struct {
	gate   Gate
	mu     sync.Mutex
	always map[alwaysKey]Decision
}

// decide is the policy's answer for inv: AllowOnce when it does not ask;
// else the request's own gate's, when it has one; else a remembered
// answer, or the registry's gate's. A refusal is an error wrapping
// ErrRefused; a gate's panic is a refusal that also wraps a *PanicError.
func (p *policy) decide(ctx context.Context, inv *Invocation, own Gate) (Decision, error) {
	if !asks(inv.Command.Danger, inv.Origin) {
		return AllowOnce, nil
	}
	if own != nil {
		d, err := askGate(ctx, own, inv)
		return verdict(inv, d, err)
	}
	key := alwaysKey{inv.Command.ID, inv.Caller}
	p.mu.Lock()
	d, ok := p.always[key]
	p.mu.Unlock()
	if ok {
		return verdict(inv, d, nil)
	}
	if p.gate == nil {
		return RejectOnce, fmt.Errorf("%w: %s %s from %s needs a gate, and there is none",
			ErrRefused, inv.Command.Danger, inv.Command.ID, inv.Origin)
	}
	d, err := askGate(ctx, p.gate, inv)
	if err == nil && (d == AllowAlways || d == RejectAlways) {
		p.mu.Lock()
		p.always[key] = d
		p.mu.Unlock()
	}
	return verdict(inv, d, err)
}

// verdict is a gate's answer, or its error, as the policy's: anything but
// AllowOnce or AllowAlways refuses.
func verdict(inv *Invocation, d Decision, err error) (Decision, error) {
	if err != nil {
		return RejectOnce, fmt.Errorf("%w: %s: the gate failed: %w", ErrRefused, inv.Command.ID, err)
	}
	if !d.allows() {
		if d != RejectAlways {
			d = RejectOnce
		}
		return d, fmt.Errorf("%w: %s: the gate said %s", ErrRefused, inv.Command.ID, d.ACPKind())
	}
	return d, nil
}
