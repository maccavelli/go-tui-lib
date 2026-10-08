package launch

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
)

// Frame is m drawn once, at width × height cells in profile p, for a
// program's plain output: a status board, a summary, a final screen
// (docs/decisions/0013-MADR-cli-integration-helpers.md §8). It sends m a
// tea.ColorProfileMsg with p, then a tea.WindowSizeMsg, each through
// Update, and returns View().Content. It calls no Init and runs no command,
// so it sends no query and reads nothing. A program with a better plain
// form, such as a command result's text, prints that instead.
func Frame(m tea.Model, width, height int, p colorprofile.Profile) string {
	m, _ = m.Update(tea.ColorProfileMsg{Profile: p})
	m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return m.View().Content
}
