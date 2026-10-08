package launchtest_test

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/launch"
	"github.com/maccavelli/go-tui-lib/launch/launchtest"
)

// picker is a program's model: it quits on q, and remembers the keys.
type picker struct{ keys string }

func (p picker) Init() tea.Cmd { return nil }

func (p picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		p.keys += k.String()
		if k.String() == "q" {
			return p, tea.Quit
		}
	}
	return p, nil
}

func (p picker) View() tea.View { return tea.NewView("pick: " + p.keys) }

// A CLI's --tui path, tested end to end: the decision on a fake terminal,
// then a run that quits on a typed q.
func ExampleTerminal() {
	term := launchtest.NewTerminal(80, 24)
	defer term.Close()
	s := launchtest.Streams(term, term, launchtest.NewPipe(""), "TERM=xterm-256color", "LANG=en_US.UTF-8")

	d := launch.Decide(s, launch.Config{Choice: launch.ChoiceTUI})
	fmt.Println(d.Interactive, d.Reason, d.Width, d.Height, d.Glyphs)

	term.Type("abq")
	final, err := launch.Run(context.Background(), s, d, picker{})
	fmt.Println(final.keys, err, launch.ExitCode(err))
	// Output:
	// true launch.terminal 80 24 unicode
	// abq <nil> 0
}
