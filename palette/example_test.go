package palette_test

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/palette"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// blank is a pane that draws nothing.
type blank struct{}

func (blank) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return blank{}, nil }
func (blank) View(int, int) string                     { return "" }

// loop stands in for a Bubble Tea program here: it runs each command, and
// gives its message back to the workspace, until none is left.
func loop(w *workspace.Workspace, cmd tea.Cmd) {
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch m := c().(type) {
		case nil:
		case tea.BatchMsg:
			queue = append(queue, m...)
		default:
			queue = append(queue, w.Update(m))
		}
	}
}

// A picker over three themes: the program builds one keymap from the
// workspace's and the palette's rules, registers a provider, and opens the
// picker. Typing filters the hits, down moves the selection, and enter runs
// the selected hit and closes the picker.
func ExamplePicker() {
	km, err := keymap.New(keymap.Settings{}, append(workspace.DefaultRules(), palette.DefaultRules()...))
	if err != nil {
		fmt.Println(err)
		return
	}
	w := workspace.New(layout.Pane{ID: "main"}, map[layout.PaneID]workspace.Pane{"main": blank{}},
		workspace.WithBindings(km),
		workspace.WithTheme(theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII())),
		workspace.WithoutBackgroundQuery())
	loop(w, w.Init())
	loop(w, w.Update(tea.WindowSizeMsg{Width: 60, Height: 12}))

	chosen := "none"
	choose := func(id string) func() tea.Cmd { return func() tea.Cmd { chosen = id; return nil } }
	p := palette.NewPicker(w)
	if err := p.Register(palette.Static("themes", []palette.Item{
		{ID: "dark", Text: "Dark theme", Run: choose("dark")},
		{ID: "light", Text: "Light theme", Run: choose("light")},
		{ID: "contrast", Text: "High contrast", Run: choose("contrast")},
	}), palette.Availability{}); err != nil {
		fmt.Println(err)
		return
	}

	loop(w, p.Open())
	for _, r := range "th" {
		loop(w, w.Update(tea.KeyPressMsg{Code: r, Text: string(r)}))
	}
	loop(w, w.Update(tea.KeyPressMsg{Code: tea.KeyDown}))
	for line := range strings.SplitSeq(ansi.Strip(p.View(40, 4)), "\n") {
		fmt.Println(strings.TrimRight(line, " "))
	}
	loop(w, w.Update(tea.KeyPressMsg{Code: tea.KeyEnter}))
	fmt.Println("chosen:", chosen)
	// Output:
	// > th
	//   Dark theme
	// > Light theme
	//                                   2 of 2
	// chosen: light
}
