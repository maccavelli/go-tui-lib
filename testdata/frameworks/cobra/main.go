// Command cobra runs a registry command from a command line built with
// Cobra: the program docs/guides/commands.md shows in "Run commands from
// your own CLI". scripts/go-examples.sh builds and runs it.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

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
			return runCLI(c.Context(), r, "session.save", saveArgs{Name: a[0]}, yes)
		},
	}
	save.Flags().BoolVar(&yes, "yes", false, "approve without asking")
	// guide:end
	root.AddCommand(save)
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
