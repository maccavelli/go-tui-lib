// Command kong runs a registry command from a command line built with Kong:
// the program docs/guides/commands.md shows in "Run commands from your own
// CLI". scripts/go-examples.sh builds and runs it.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/launch"
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

// session is the program's TUI: here, a stand-in that quits on q.
type session struct{ keys string }

func (m session) Init() tea.Cmd { return nil }

func (m session) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if k.String() == "q" {
			return m, tea.Quit
		}
		m.keys += k.String()
	}
	return m, nil
}

func (m session) View() tea.View { return tea.NewView("session: " + m.keys) }

// tui runs the TUI for --tui. When it cannot start, or crashes, it says why
// on Err and returns the model to continue from, with fallBack true: the
// program goes on in its own CLI mode, and never repeats what the TUI did.
// An end the user or the program chose ends the program, with its status.
func tui(ctx context.Context, s launch.Streams, r *command.Registry, m session) (session, int, bool) {
	d := launch.Decide(s, launch.Config{Choice: launch.ChoiceTUI})
	final, err := launch.Run(ctx, s, d, m, launch.WithRegistry(r))
	switch {
	case errors.Is(err, launch.ErrNotStarted):
		fmt.Fprintf(s.Err, "Warning: --tui unavailable (%v); using the CLI\n", err)
		return m, 0, true
	case errors.Is(err, launch.ErrCrashed):
		fmt.Fprintf(s.Err, "Warning: the TUI stopped (%v); continuing in the CLI\n", err)
		return final, 0, true
	default:
		return final, launch.ExitCode(err), false
	}
}

// lineMode is the program's own CLI mode, continuing from m.
func lineMode(s launch.Streams, m session) { fmt.Fprintf(s.Out, "line mode, from %q\n", m.keys) }

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
		launch.Flags `embed:""` // --mode, --tui and --no-tui

		Yes  bool     `help:"approve without asking"`
		Save saveArgs `cmd:"" help:"Save the session"`
		Line struct{} `cmd:"" default:"1" help:"Run the line mode"`
	}
	kctx := kong.Parse(&cli)
	if kctx.Command() == "save <name>" {
		kctx.FatalIfErrorf(runCLI(ctx, r, "session.save", cli.Save, cli.Yes))
	}
	// guide:end
	// guide:kong-tui
	if kctx.Command() == "line" {
		s := launch.Streams{In: os.Stdin, Out: kctx.Stdout, Err: kctx.Stderr, Env: os.Environ()}
		m := session{}
		if cli.Resolve() == launch.ChoiceTUI {
			final, code, fallBack := tui(ctx, s, r, m)
			if !fallBack {
				os.Exit(code)
			}
			m = final
		}
		lineMode(s, m)
	}
	// guide:end
}
