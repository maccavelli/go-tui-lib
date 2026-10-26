package command

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
)

// Verdict is a gate's answer, with ACP's four permission option kinds.
// Its zero value is "no verdict", which refuses.
type Verdict uint8

// The verdicts.
const (
	AllowOnce    Verdict = iota + 1 // run this time
	AllowAlways                     // run, and do not ask again for this command and caller
	RejectOnce                      // refuse this time
	RejectAlways                    // refuse, and do not ask again for this command and caller
)

var verdictNames = enum.Names[Verdict]{
	Pkg: pkgName, Type: "Verdict",
	Tokens: []string{"", "allow_once", "allow_always", "reject_once", "reject_always"},
}

// String is the verdict's token, its ACPKind: "" for no verdict.
func (v Verdict) String() string { return verdictNames.String(v) }

// MarshalText is the verdict's token. A verdict with no token is an error.
func (v Verdict) MarshalText() ([]byte, error) { return verdictNames.Marshal(v) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (v *Verdict) UnmarshalText(b []byte) error { return verdictNames.Unmarshal(b, v) }

// ACPKind is v as ACP's PermissionOptionKind: "allow_once", "allow_always",
// "reject_once" or "reject_always"; "" for no verdict, and for a v with no
// token.
func (v Verdict) ACPKind() string {
	if v > RejectAlways {
		return ""
	}
	return v.String()
}

func (v Verdict) allows() bool { return v == AllowOnce || v == AllowAlways }

// Gate is asked before a command runs when the policy says to ask:
// typically a permission dialog. Dispatch asks it on the goroutine that
// calls Dispatch, and Run on Run's; a gate that waits for the user is
// reached through Run, from the agent's goroutine. A gate's panic refuses
// the request, with an error that wraps ErrRefused and a *PanicError.
type Gate interface {
	Decide(ctx context.Context, inv *Invocation) (Verdict, error)
}

// GateFunc is a function used as a Gate.
type GateFunc func(ctx context.Context, inv *Invocation) (Verdict, error)

// Decide calls f.
func (f GateFunc) Decide(ctx context.Context, inv *Invocation) (Verdict, error) {
	return f(ctx, inv)
}

// AllowIf is a Gate that answers AllowOnce when ok, and RejectOnce
// otherwise: a program's own command line passes its --yes flag, as
// Request.Gate.
func AllowIf(ok bool) Gate {
	return GateFunc(func(context.Context, *Invocation) (Verdict, error) {
		if ok {
			return AllowOnce, nil
		}
		return RejectOnce, nil
	})
}

// GatePolicy says which requests are put to a gate before they run
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D4). Asks is
// called for each request the registry admits, on the goroutine that calls
// Dispatch or Run, and must not block. A policy's panic refuses the request,
// with an error that wraps ErrRefused and a *PanicError.
type GatePolicy interface {
	Asks(c Command, o Origin) bool
}

// GatePolicyFunc is a function used as a GatePolicy.
type GatePolicyFunc func(c Command, o Origin) bool

// Asks calls f.
func (f GatePolicyFunc) Asks(c Command, o Origin) bool { return f(c, o) }

// DefaultGatePolicy is the policy a registry has without WithGatePolicy: a
// mutating command from an agent, and a destructive one from an agent or
// the shell, ask the gate (docs/decisions/0006-MADR-command-registry.md
// §7). A program's own policy may call it, and change only what it means
// to.
func DefaultGatePolicy() GatePolicy {
	return GatePolicyFunc(func(c Command, o Origin) bool { return defaultAsks(c.Danger, o) })
}

// WithGatePolicy sets the policy that says which requests ask the gate. A
// nil policy keeps DefaultGatePolicy's.
func WithGatePolicy(p GatePolicy) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		if p != nil {
			r.policy.rule = p
		}
	})
}

// defaultAsks is DefaultGatePolicy's table, for a command of danger d and
// origin o.
func defaultAsks(d Danger, o Origin) bool {
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

// policy holds the gate policy, the gate and the remembered answers.
type policy struct {
	rule   GatePolicy // nil: DefaultGatePolicy's table
	gate   Gate
	mu     sync.Mutex
	always map[alwaysKey]Verdict
}

// asks reports whether the policy asks a gate about inv, and returns the
// rule's panic as a *PanicError.
func (p *policy) asks(inv *Invocation) (ask bool, err error) {
	if p.rule == nil {
		return defaultAsks(inv.Command.Danger, inv.Origin), nil
	}
	defer func() {
		if v := recover(); v != nil {
			ask, err = true, &PanicError{ID: inv.Command.ID, Value: v, Stack: debug.Stack(), gate: true}
		}
	}()
	return p.rule.Asks(inv.Command, inv.Origin), nil
}

// forget drops the answers remembered for id and caller, or every answer
// when all is set.
func (p *policy) forget(id ID, caller string, all bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if all {
		clear(p.always)
		return
	}
	delete(p.always, alwaysKey{id, caller})
}

// decide is the policy's answer for inv: AllowOnce when it does not ask;
// else the request's own gate's, when it has one; else a remembered
// answer, or the registry's gate's. A refusal is an error wrapping
// ErrRefused; a policy's or a gate's panic is a refusal that also wraps a
// *PanicError.
func (p *policy) decide(ctx context.Context, inv *Invocation, own Gate) (Verdict, error) {
	ask, err := p.asks(inv)
	if err != nil {
		return settle(inv, RejectOnce, err)
	}
	if !ask {
		return AllowOnce, nil
	}
	if own != nil {
		d, err := askGate(ctx, own, inv)
		return settle(inv, d, err)
	}
	key := alwaysKey{inv.Command.ID, inv.Caller}
	p.mu.Lock()
	d, ok := p.always[key]
	p.mu.Unlock()
	if ok {
		return settle(inv, d, nil)
	}
	if p.gate == nil {
		return RejectOnce, fmt.Errorf("%w: %s %s from %s needs a gate, and there is none",
			ErrRefused, inv.Command.Danger, inv.Command.ID, inv.Origin)
	}
	d, err = askGate(ctx, p.gate, inv)
	if err == nil && (d == AllowAlways || d == RejectAlways) {
		p.mu.Lock()
		p.always[key] = d
		p.mu.Unlock()
	}
	return settle(inv, d, err)
}

// settle is a gate's answer, or its error, as the policy's: anything but
// AllowOnce or AllowAlways refuses.
func settle(inv *Invocation, d Verdict, err error) (Verdict, error) {
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
