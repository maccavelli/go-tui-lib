// Command kong runs a registry command from a command line built with Kong:
// the program docs/guides/commands.md shows in "Run commands from your own
// CLI". scripts/go-examples.sh builds and runs it.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/maccavelli/go-tui-lib/command"
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
	ctx := context.Background()
	r, err := newRegistry()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// guide:kong
	var cli struct {
		Yes  bool     `help:"approve without asking"`
		Save saveArgs `cmd:"" help:"Save the session"`
		TUI  struct{} `cmd:"" default:"1" help:"Run the TUI"`
	}
	kctx := kong.Parse(&cli)
	if kctx.Command() == "save <name>" {
		kctx.FatalIfErrorf(runCLI(ctx, r, "session.save", cli.Save, cli.Yes))
	}
	// guide:end
}
