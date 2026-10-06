package workspace

import (
	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/theme"
)

// SetBackground pins the background the theme is built for, or with
// theme.Unknown returns to following the terminal
// (docs/decisions/0006-MADR-command-registry.md A2, Q5).
//
// theme.Dark or theme.Light rebuilds the theme for that background, and a
// later tea.BackgroundColorMsg changes only what the workspace remembers
// the terminal reported. theme.Unknown rebuilds the theme for the
// background last reported, and returns tea.RequestBackgroundColor to ask
// again, unless the workspace was built WithoutBackgroundQuery. A
// workspace built WithTheme does not follow the terminal: it records the
// choice and returns nil, and its theme does not change.
func (w *Workspace) SetBackground(bg theme.Background) tea.Cmd {
	w.pinnedBg = bg
	if !w.follow {
		return nil
	}
	w.rebuildTheme()
	if bg == theme.Unknown && !w.noQuery {
		return tea.RequestBackgroundColor
	}
	return nil
}

// background is the background the theme is built for: the pinned one,
// or else the one the terminal last reported.
func (w *Workspace) background() theme.Background {
	if w.pinnedBg != theme.Unknown {
		return w.pinnedBg
	}
	return w.bg
}
