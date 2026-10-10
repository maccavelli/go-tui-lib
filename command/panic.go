package command

import (
	"context"
	"runtime/debug"
)

// PanicError is a handler's or a gate's panic, as an error
// (docs/decisions/0014-MADR-native-integration-api.md W2). Its text names
// the command and leaves out Value, which may hold a secret; the program
// reads Value and Stack from the error itself. It wraps ErrPanicked.
type PanicError struct {
	ID    ID
	Value any    // what was passed to panic
	Stack []byte // the panicking goroutine's stack, from debug.Stack

	gate bool // the gate panicked, not the handler
}

func (e *PanicError) Error() string {
	if e.gate {
		return "command: " + string(e.ID) + ": gate panicked"
	}
	return "command: " + string(e.ID) + ": handler panicked"
}

// Unwrap returns ErrPanicked.
func (e *PanicError) Unwrap() error { return ErrPanicked }

// ExitCode is 2, the status of an unrecovered Go panic.
func (e *PanicError) ExitCode() int { return 2 }

// runHandler runs c's handler, and returns its panic as a *PanicError.
func runHandler(ctx context.Context, c *Command, inv *Invocation) (res Result, err error) {
	defer func() {
		if p := recover(); p != nil {
			res, err = Result{}, &PanicError{ID: c.ID, Value: p, Stack: debug.Stack()}
		}
	}()
	return c.Handler.Run(ctx, inv)
}

// askGate asks g about inv, and returns its panic as a *PanicError.
func askGate(ctx context.Context, g Gate, inv *Invocation) (d Verdict, err error) {
	defer func() {
		if p := recover(); p != nil {
			d, err = RejectOnce, &PanicError{ID: inv.Command.ID, Value: p, Stack: debug.Stack(), gate: true}
		}
	}()
	return g.Decide(ctx, inv)
}
