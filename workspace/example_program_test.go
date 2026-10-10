package workspace_test

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// model is the program around a workspace, as
// docs/guides/building-workspaces.md shows it. The program, not the
// workspace, owns ctrl+c, the alternate screen and the mouse mode.
type model struct{ ws *workspace.Workspace }

func (m model) Init() tea.Cmd { return m.ws.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+c" {
		return m, tea.Quit
	}
	return m, m.ws.Update(msg)
}

func (m model) View() tea.View {
	v := tea.NewView(m.ws.Render())
	v.Cursor = m.ws.Cursor()
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// The program around a workspace. It is compiled, not run: running it needs
// a terminal.
func ExampleWorkspace_program() {
	th := theme.New(colorprofile.TrueColor, theme.Unknown, glyph.Unicode())
	ws := workspace.New(
		layout.SidebarRightBottom("session", "metrics", "logs", layout.WithFooter("footer", 1), layout.WithGap(0)),
		map[layout.PaneID]workspace.Pane{
			"session": &transcript{}, "metrics": &metrics{}, "logs": &logs{}, "footer": &footer{text: "ready"},
		},
		workspace.WithTheme(th), workspace.WithPaneChrome("footer", workspace.None),
	)
	_ = tea.NewProgram(model{ws: ws}) // then: p.Run()
}
