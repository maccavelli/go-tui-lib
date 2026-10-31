package workspace

import (
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
)

// The workspace's own words, by message ID: its commands' results and
// reasons, and its help
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9, A4).
const (
	// MsgNoPaneIndex is workspace.focus.index's result when the focus ring
	// has no pane at the place asked: the place.
	MsgNoPaneIndex msgcat.MessageID = "workspace.no-pane-index"
	// MsgNoOverlay is workspace.overlay.close's result when none is open.
	MsgNoOverlay msgcat.MessageID = "workspace.no-overlay"
	// MsgNotFocusable is the reason a pane cannot take focus.
	MsgNotFocusable msgcat.MessageID = "workspace.not-focusable"
	// MsgNotAPane is the reason an ID names no pane of the layout.
	MsgNotAPane msgcat.MessageID = "workspace.not-a-pane"
	// MsgNotADirection is the reason a direction is not one of the four.
	MsgNotADirection msgcat.MessageID = "workspace.not-a-direction"
	// MsgNotASplit is the reason a name is no resizable split on screen.
	MsgNotASplit msgcat.MessageID = "workspace.not-a-split"
	// MsgManySplits is the reason a split's name is not enough: the
	// separators on screen, then their IDs.
	MsgManySplits msgcat.MessageID = "workspace.many-splits"
	// MsgNotALayout is the reason a name is no layout's.
	MsgNotALayout msgcat.MessageID = "workspace.not-a-layout"
	// MsgPaneFocused marks the focused pane in workspace.panes's text.
	MsgPaneFocused msgcat.MessageID = "workspace.pane-focused"
	// MsgPaneHidden marks a hidden pane in workspace.panes's text.
	MsgPaneHidden msgcat.MessageID = "workspace.pane-hidden"
	// MsgHelpNextPane is workspace.focus.next in the help.
	MsgHelpNextPane msgcat.MessageID = "workspace.help-next-pane"
	// MsgHelpPrevPane is workspace.focus.prev in the help.
	MsgHelpPrevPane msgcat.MessageID = "workspace.help-prev-pane"
	// MsgHelpPane is workspace.focus.index in the help.
	MsgHelpPane msgcat.MessageID = "workspace.help-pane"
	// MsgHelpZoom is workspace.zoom in the help.
	MsgHelpZoom msgcat.MessageID = "workspace.help-zoom"
	// MsgHelpResize is workspace.resize.direction in the help.
	MsgHelpResize msgcat.MessageID = "workspace.help-resize"
	// MsgHelpClose is workspace.overlay.close in the help.
	MsgHelpClose msgcat.MessageID = "workspace.help-close"
)

// Messages is the workspace's words in English, its defaults, by ID:
// where a program starts its translation.
func Messages() msgcat.Table {
	return msgcat.Table{
		MsgNoPaneIndex:   "No pane %d in the focus ring.",
		MsgNoOverlay:     "No overlay is open.",
		MsgNotFocusable:  "is not a pane on screen that takes focus",
		MsgNotAPane:      "is not a pane of the layout",
		MsgNotADirection: "is not left, right, up or down",
		MsgNotASplit:     "is not a resizable split on screen",
		MsgManySplits:    "has %[1]d separators on screen; name one of %[2]s",
		MsgNotALayout:    "is not a layout's name",
		MsgPaneFocused:   "focused",
		MsgPaneHidden:    "hidden",
		MsgHelpNextPane:  "next pane",
		MsgHelpPrevPane:  "previous pane",
		MsgHelpPane:      "pane",
		MsgHelpZoom:      "zoom",
		MsgHelpResize:    "resize",
		MsgHelpClose:     "close",
	}
}

// WithCatalog gives the workspace's own words in the program's: see
// Messages.
func WithCatalog(c msgcat.Catalog) Option { return optionFunc(func(w *Workspace) { w.catalog = c }) }

// say is the workspace's text for id, from its catalogue or the default,
// as one line.
func (w *Workspace) say(id msgcat.MessageID, args ...any) string {
	return sanitize.Line(msgcat.Format(w.catalog, id, Messages()[id], args...))
}
