// Command cobra runs a registry command from a command line built with
// Cobra: the program docs/guides/commands.md shows in "Run commands from
// your own CLI". scripts/go-examples.sh builds and runs it.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/spf13/cobra"
)

// yesGate approves a command that asks, when --yes was given.
type yesGate bool

func (y yesGate) Decide(context.Context, *command.Invocation) (command.Decision, error) {
	if y {
		return command.AllowOnce, nil
	}
	return command.RejectOnce, nil
}

// runCLI runs a registry command from the program's own command line.
func runCLI(ctx context.Context, r *command.Registry, id command.ID, args any, yes bool) error {
	raw, err := json.Marshal(args)
	if err != nil {
		return err
	}
	res, err := r.Run(ctx, command.Request{
		ID: id, Args: raw, Origin: command.OriginCLI, Caller: "pi", Gate: yesGate(yes),
	})
	if err != nil {
		return err
	}
	fmt.Println(res.Text)
	return nil
}

type saveArgs struct {
	Name string `json:"name" arg:"" help:"the session's name"`
}

// newRegistry holds the one command the example runs: a Destructive save,
// which the policy asks the gate about when the CLI runs it.
func newRegistry() (*command.Registry, error) {
	save, err := command.New("session.save", "Save the session",
		func(_ context.Context, _ *command.Invocation, a saveArgs) (command.Result, error) {
			return command.Result{Text: "saved " + a.Name, Value: a.Name}, nil
		},
		command.WithDanger(command.Destructive),
		command.WithSurfaces(command.SurfaceCLI),
	)
	if err != nil {
		return nil, err
	}
	r := command.NewRegistry()
	return r, r.Register(save)
}

func main() {
	r, err := newRegistry()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root := &cobra.Command{Use: "pi", Short: "A program with a TUI beside its CLI"}
	// guide:cobra
	var yes bool
	save := &cobra.Command{
		Use:   "save NAME",
		Short: "Save the session",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			return runCLI(c.Context(), r, "session.save", saveArgs{Name: a[0]}, yes)
		},
	}
	save.Flags().BoolVar(&yes, "yes", false, "approve without asking")
	// guide:end
	root.AddCommand(save)
	if err := root.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
