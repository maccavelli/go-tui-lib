// Command cobra runs a registry command from a command line built with
// Cobra: the program docs/guides/commands.md shows in "Run commands from
// your own CLI". scripts/go-examples.sh builds and runs it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

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
	// guide:cobra-tui
	var f launch.Flags
	root := &cobra.Command{
		Use:   "pi",
		Short: "A program with a TUI beside its CLI",
		RunE: func(c *cobra.Command, _ []string) error {
			s := launch.FromSource(c, os.Environ())
			m := session{}
			if f.Resolve() == launch.ChoiceTUI {
				final, code, fallBack := tui(c.Context(), s, r, m)
				if !fallBack {
					return exitWith(code)
				}
				m = final
			}
			lineMode(s, m)
			return nil
		},
	}
	fs := flag.NewFlagSet("pi", flag.ContinueOnError)
	f.RegisterFlags(fs)
	root.PersistentFlags().AddGoFlagSet(fs) // keeps --tui a bare boolean
	// guide:end
	// guide:cobra
	var yes bool
	save := &cobra.Command{
		Use:   "save NAME",
		Short: "Save the session",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			return runCLI(c.Context(), c.OutOrStdout(), r, "session.save", saveArgs{Name: a[0]}, yes)
		},
	}
	save.Flags().BoolVar(&yes, "yes", false, "approve without asking")
	// guide:end
	// guide:cobra-run
	var format command.Format
	run := &cobra.Command{
		Use:   "run ID [ARGS...]",
		Short: "Run any command by its ID",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			return runAny(c.Context(), c.OutOrStdout(), r, a, yes, format)
		},
		// The IDs, then each command's flags and their values.
		ValidArgsFunction: func(_ *cobra.Command, a []string, partial string) ([]string, cobra.ShellCompDirective) {
			if len(a) == 0 {
				return r.Complete("", nil, partial), cobra.ShellCompDirectiveNoFileComp
			}
			return r.Complete(command.ID(a[0]), a[1:], partial), cobra.ShellCompDirectiveNoFileComp
		},
	}
	run.Flags().BoolVar(&yes, "yes", false, "approve without asking")
	run.Flags().TextVar(&format, "format", command.FormatText, "text or json")
	run.Flags().SetInterspersed(false) // the flags after the ID are the command's
	// guide:end
	root.AddCommand(save, run)
	if err := root.ExecuteContext(context.Background()); err != nil {
		os.Exit(launch.ExitCode(err))
	}
}

// exitWith is nil for status 0, and an ExitError otherwise.
func exitWith(code int) error {
	if code == 0 {
		return nil
	}
	return &launch.ExitError{Code: code}
}
