package command

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// Panics in handlers and gates, as errors
// (docs/decisions/0014-PLAN-component-native-forms.md Step 2).

// secret is a panic value that must never reach an error's text.
const secret = "s3cr3t-panic-value"

// panicking is a command of mode m whose handler panics with secret.
func panicking(m Mode) Command {
	return cmd("p", UI, func(c *Command) {
		c.Mode = m
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			panic(secret)
		})
	})
}

// isPanic reports whether err is a handler's *PanicError for p, holding
// secret and a stack, with secret kept out of its text.
func isPanic(t *testing.T, how string, err error) {
	t.Helper()
	pe, ok := errors.AsType[*PanicError](err)
	switch {
	case !ok:
		t.Errorf("%s: error %v is not a *PanicError", how, err)
	case pe.ID != "p" || pe.Value != secret || len(pe.Stack) == 0:
		t.Errorf("%s: PanicError = {%s %v %d stack bytes}", how, pe.ID, pe.Value, len(pe.Stack))
	case !errors.Is(err, ErrPanicked):
		t.Errorf("%s: %v does not wrap ErrPanicked", how, err)
	case err.Error() != "command: p: handler panicked":
		t.Errorf("%s: text %q", how, err.Error())
	}
}

func TestHandlerPanicIsAnError(t *testing.T) {
	t.Run("Loop through Dispatch", func(t *testing.T) {
		r := registryOf(t, nil, panicking(Loop))
		c := r.Dispatch(t.Context(), Request{ID: "p", Origin: OriginKey})
		isPanic(t, "Dispatch", resultOf(t, collect(c)).Err)
	})
	t.Run("Async inside its command", func(t *testing.T) {
		r := registryOf(t, nil, panicking(Async))
		c := r.Dispatch(t.Context(), Request{ID: "p", Origin: OriginKey})
		isPanic(t, "the Async command", resultOf(t, collect(c)).Err)
		if n := len(r.running.runs); n != 0 {
			t.Errorf("%d commands still running after the panic", n)
		}
	})
	for _, m := range []Mode{Loop, Async} {
		t.Run("Run "+m.String(), func(t *testing.T) {
			r := registryOf(t, nil, panicking(m))
			_, err := r.Run(t.Context(), Request{ID: "p", Origin: OriginCLI})
			isPanic(t, "Run", err)
		})
	}
	t.Run("CallMCP", func(t *testing.T) {
		r := registryOf(t, nil, panicking(Loop))
		res := r.CallMCP(t.Context(), "p", nil, "agent")
		if !res.IsError || len(res.Content) != 1 || !strings.Contains(res.Content[0].Text, "command: p: handler panicked") {
			t.Errorf("CallMCP = %+v", res)
		}
	})
}

func TestGatePanicRefuses(t *testing.T) {
	ran := false
	r := registryOf(t, []RegistryOption{WithGate(&gateFunc{next: func(*Invocation) (Verdict, error) {
		panic(secret)
	}})}, cmd("p", Destructive, func(c *Command) {
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			ran = true
			return Result{}, nil
		})
	}))
	_, err := r.Run(t.Context(), Request{ID: "p", Origin: OriginAgent})
	if ran {
		t.Error("the handler ran after its gate panicked")
	}
	if !errors.Is(err, ErrRefused) || !errors.Is(err, ErrPanicked) {
		t.Fatalf("err = %v; want ErrRefused and ErrPanicked", err)
	}
	pe, _ := errors.AsType[*PanicError](err)
	if pe.ID != "p" || pe.Value != secret || pe.Error() != "command: p: gate panicked" {
		t.Errorf("PanicError = %q, %v", pe.Error(), pe.Value)
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "handler") {
		t.Errorf("err's text %q", err.Error())
	}
}

func TestPanicErrorHidesValue(t *testing.T) {
	var log bytes.Buffer
	r := registryOf(t, []RegistryOption{WithAuditor(SlogAuditor(slog.New(slog.NewTextHandler(&log, nil))))}, panicking(Loop))
	_, err := r.Run(t.Context(), Request{ID: "p", Origin: OriginCLI})
	mcp := r.CallMCP(t.Context(), "p", nil, "agent")
	pe, ok := errors.AsType[*PanicError](err)
	if !ok || pe.Value != secret {
		t.Fatalf("err = %v; want a *PanicError holding the value", err)
	}
	for what, s := range map[string]string{
		"Error()": err.Error(), "the audit log": log.String(), "CallMCP": mcp.Content[0].Text,
	} {
		if strings.Contains(s, secret) {
			t.Errorf("%s holds the panic's value: %q", what, s)
		}
		if !strings.Contains(s, "command: p: handler panicked") {
			t.Errorf("%s does not name the panic: %q", what, s)
		}
	}
	if !strings.Contains(string(pe.Stack), "panic_test.go") {
		t.Error("the stack does not reach the panicking handler")
	}
}

func TestSentinelsStillCompare(t *testing.T) {
	r := registryOf(t, nil, cmd("off", UI, func(c *Command) { c.When = "never" }))
	for _, tc := range []struct {
		req  Request
		want error
	}{
		{Request{ID: "nope", Origin: OriginCLI}, ErrUnknown},
		{Request{ID: "off", Origin: OriginCLI}, ErrUnavailable},
		{Request{ID: "off"}, ErrRefused},
	} {
		_, err := r.Run(t.Context(), tc.req)
		if !errors.Is(err, tc.want) {
			t.Errorf("%+v: %v is not %v", tc.req, err, tc.want)
		}
		for _, other := range []error{ErrUnknown, ErrUnavailable, ErrRefused, ErrPanicked} {
			if !errors.Is(other, tc.want) && errors.Is(err, other) {
				t.Errorf("%+v: %v is also %v", tc.req, err, other)
			}
		}
	}
	if ErrUnknown.Error() != "command: unknown command" || ErrUnavailable.Error() != "command: not available" ||
		ErrRefused.Error() != "command: refused" {
		t.Error("a sentinel's text changed")
	}
}
