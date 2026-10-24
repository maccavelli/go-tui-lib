package termcaptea

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// KeyboardEnhancements is what a program should put in
// View.KeyboardEnhancements: event types only where the terminal reports
// them cleanly, which c.Keyboard().Request says. Tea always asks for
// disambiguation as well, which a program cannot turn off through the
// View (docs/decisions/0005-MADR-terminal-capabilities-and-services.md
// A4); Keyboard's reason says when the terminal would rather have none.
func KeyboardEnhancements(c termcap.Caps) tea.KeyboardEnhancements {
	return tea.KeyboardEnhancements{ReportEventTypes: c.Keyboard().Request&ansi.KittyReportEventTypes != 0}
}
