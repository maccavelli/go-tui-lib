package launch_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/launch"
	"github.com/maccavelli/go-tui-lib/termcap"
)

// Decide's rules, on streams that are not terminals, under each choice.
func ExampleDecide() {
	s := launch.Streams{
		In:  strings.NewReader(""),
		Out: io.Discard,
		Err: io.Discard,
		Env: termcap.Env{"TERM=xterm-256color", "LANG=en_US.UTF-8", "CI=true"},
	}
	for _, c := range []launch.Choice{launch.ChoicePlain, launch.ChoiceAuto, launch.ChoiceTUI} {
		d := launch.Decide(s, launch.Config{Choice: c})
		fmt.Println(c, d.Interactive, d.Reason, d.Glyphs)
	}
	// Output:
	// plain false launch.requested-plain unicode
	// auto false launch.ci unicode
	// tui false launch.input-not-terminal unicode
}

// The shared flags in the standard flag package: --tui wins over --mode.
func ExampleFlags() {
	var f launch.Flags
	fs := flag.NewFlagSet("prog", flag.ContinueOnError)
	f.RegisterFlags(fs)
	_ = fs.Parse([]string{"-mode", "plain"})
	fmt.Println(f.Resolve())
	_ = fs.Parse([]string{"-tui"})
	fmt.Println(f.Resolve())
	// Output:
	// plain
	// tui
}

// session is the program's model: here, a TUI over a session that its CLI
// mode can resume.
type session struct{ id string }

func (s session) Init() tea.Cmd                       { return nil }
func (s session) Update(tea.Msg) (tea.Model, tea.Cmd) { return s, tea.Quit }
func (s session) View() tea.View                      { return tea.NewView("session " + s.id) }

// runCLI is the program's own CLI mode.
func runCLI(context.Context, launch.Streams, session) int { return 0 }

// The --tui pattern, for a program whose default is its own CLI: a TUI that
// cannot start, or crashes, falls back to the CLI, which continues where the
// TUI was and never resends. Compiled, not run: running it needs a terminal.
func ExampleRun() {
	var f launch.Flags
	fs := flag.NewFlagSet("prog", flag.ExitOnError)
	f.RegisterFlags(fs)
	_ = fs.Parse(os.Args[1:])

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	s := launch.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Env: os.Environ()}
	sess := session{id: "new"}

	if f.Resolve() == launch.ChoiceTUI {
		d := launch.Decide(s, launch.Config{Choice: launch.ChoiceTUI})
		final, err := launch.Run(ctx, s, d, sess)
		switch {
		case errors.Is(err, launch.ErrNotStarted):
			fmt.Fprintf(s.Err, "Warning: --tui unavailable (%v); using the CLI\n", err)
		case errors.Is(err, launch.ErrCrashed):
			fmt.Fprintf(s.Err, "Warning: the TUI stopped (%v); continuing in the CLI\n", err)
			sess = final // continue where the TUI was
		default:
			os.Exit(launch.ExitCode(err)) // the user or the program ended it
		}
	}
	os.Exit(runCLI(ctx, s, sess))
}
