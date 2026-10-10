// Command urfave runs a registry command from a command line built with
// urfave/cli v3: the program docs/guides/commands.md shows in "Run commands
// from your own CLI". scripts/go-examples.py builds and runs it.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/urfave/cli/v3"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/launch"
)

// runCLI runs a registry command from the program's own command line, and
// writes its result to w. AllowIf(yes) approves a command that asks, when
// --yes was given.
func runCLI(ctx context.Context, w io.Writer, r *command.Registry, id command.ID, args any, yes bool) error {
	raw, err := command.ArgsOf(args)
	if err != nil {
		return err
	}
	res, err := r.Run(ctx, command.Request{
		ID: id, Args: raw, Origin: command.OriginCLI, Caller: "pi", Gate: command.AllowIf(yes),
	})
	if err != nil {
		return err
	}
	return command.WriteResult(w, res, command.FormatText)
}

// runAny runs any command offered to the CLI, by its ID and its arguments
// as the shell split them: "run session.save notes", or "run
// session.save --name=notes".
func runAny(ctx context.Context, w io.Writer, r *command.Registry, args []string, yes bool, f command.Format) error {
	if len(args) == 0 {
		return &command.ArgError{Reason: "run: name a command"}
	}
	req, err := r.ParseArgs(command.ID(args[0]), args[1:], command.OriginCLI)
	if err != nil {
		return err
	}
	req.Caller, req.Gate = "pi", command.AllowIf(yes)
	res, err := r.Run(ctx, req)
	if err != nil {
		return err
	}
	return command.WriteResult(w, res, f)
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
	r, err := newRegistry()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var f launch.Flags
	var format command.Format
	one := 1
	// guide:urfave
	app := &cli.Command{
		Name: "pi",
		Flags: []cli.Flag{
			&cli.GenericFlag{Name: "mode", Value: &f.Mode, Usage: "auto, tui or plain"},
			&cli.BoolFlag{Name: "tui", Destination: &f.TUI, Usage: "start the TUI"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			s := launch.Streams{In: c.Reader, Out: c.Writer, Err: c.ErrWriter, Env: os.Environ()}
			m := session{}
			if f.Resolve() == launch.ChoiceTUI {
				final, code, fallBack := tui(ctx, s, r, m)
				if !fallBack {
					return exitWith(code)
				}
				m = final
			}
			lineMode(s, m)
			return nil
		},
		Commands: []*cli.Command{{
			Name:      "save",
			Usage:     "Save the session",
			ArgsUsage: "NAME",
			Flags:     []cli.Flag{&cli.BoolFlag{Name: "yes", Usage: "approve without asking"}},
			Action: func(ctx context.Context, c *cli.Command) error {
				return runCLI(ctx, c.Root().Writer, r, "session.save", saveArgs{Name: c.Args().First()}, c.Bool("yes"))
			},
		}, {
			Name:      "run",
			Usage:     "Run any command by its ID",
			ArgsUsage: "ID [ARGS...]",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "yes", Usage: "approve without asking"},
				&cli.TextFlag{Name: "format", Value: &format, Usage: "text or json"},
			},
			StopOnNthArg: &one, // the flags after the ID are the command's
			Action: func(ctx context.Context, c *cli.Command) error {
				return runAny(ctx, c.Root().Writer, r, c.Args().Slice(), c.Bool("yes"), format)
			},
		}},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(launch.ExitCode(err))
	}
	// guide:end
}

// exitWith is nil for status 0, and an ExitError otherwise.
func exitWith(code int) error {
	if code == 0 {
		return nil
	}
	return &launch.ExitError{Code: code}
}
