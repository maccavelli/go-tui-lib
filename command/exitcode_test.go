package command_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/launch"
)

// TestExitCodes: each of command's errors carries its exit status
// (docs/decisions/0014-MADR-native-integration-api.md A1), bare and
// wrapped, through launch.ExitCode and through errors.As. It is an
// external test, since launch imports command.
func TestExitCodes(t *testing.T) {
	r := command.NewRegistry(command.WithGate(gate(func() (command.Decision, error) { panic("boom") })))
	panics, err := command.New("panics", "Panics", func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
		panic("boom")
	}, command.WithDanger(command.UI))
	if err != nil {
		t.Fatal(err)
	}
	gated, err := command.New("gated", "Gated", func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
		return command.Result{}, nil
	}, command.WithDanger(command.Destructive))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Register(panics, gated); err != nil {
		t.Fatal(err)
	}
	run := func(req command.Request) error {
		_, err := r.Run(t.Context(), req)
		return err
	}
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"ErrUnknown", command.ErrUnknown, 2},
		{"ErrUnavailable", command.ErrUnavailable, 1},
		{"ErrRefused", command.ErrRefused, 3},
		{"ErrPanicked", command.ErrPanicked, 2},
		{"*ArgError", &command.ArgError{Path: "/x", Reason: "bad"}, 2},
		{"*PanicError", &command.PanicError{ID: "x", Value: 1}, 2},
		{"an unknown command", run(command.Request{ID: "nope", Origin: command.OriginCLI}), 2},
		{"no origin", run(command.Request{ID: "panics"}), 3},
		{"bad arguments", run(command.Request{ID: "panics", Origin: command.OriginCLI, Args: []byte(`{"x":1}`)}), 2},
		{"a handler's panic", run(command.Request{ID: "panics", Origin: command.OriginCLI}), 2},
		// A gate's panic is a refusal first, and a panic second.
		{"a gate's panic", run(command.Request{ID: "gated", Origin: command.OriginAgent}), 3},
	} {
		for _, err := range []error{tc.err, fmt.Errorf("the program: %w", tc.err)} {
			if got := launch.ExitCode(err); got != tc.want {
				t.Errorf("%s: launch.ExitCode(%v) = %d, want %d", tc.name, err, got, tc.want)
			}
			var c interface{ ExitCode() int }
			if !errors.As(err, &c) || c.ExitCode() != tc.want {
				t.Errorf("%s: errors.As(%v) found %v", tc.name, err, c)
			}
		}
	}
}

// gate is a function as a command.Gate.
type gate func() (command.Decision, error)

func (g gate) Decide(context.Context, *command.Invocation) (command.Decision, error) { return g() }
