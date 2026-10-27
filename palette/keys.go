package palette

import (
	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
)

// The commands a picker's keys are bound to (0008-MADR A3 item 1),
// exported so a program binds them under its own keys and scopes
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D4). The
// program runs palette.open, through OpenCommand; the picker runs the
// others itself, as a workspace.CommandConsumer, in the order their keys
// were pressed.
const (
	OpenID       command.ID = "palette.open"        // opens the picker: the program's OpenCommand
	SelectNextID command.ID = "palette.select.next" // moves the selection down a row
	SelectPrevID command.ID = "palette.select.prev" // moves the selection up a row
	PageNextID   command.ID = "palette.page.next"   // moves the selection down a page
	PagePrevID   command.ID = "palette.page.prev"   // moves the selection up a page
	AcceptID     command.ID = "palette.accept"      // runs the selected hit, or accepts it in a pop-up
	CompleteID   command.ID = "palette.complete"    // completes the query to the selected hit
)

// PickerScope is the key scope an open picker names, as a
// workspace.KeyContexter.
const PickerScope command.Scope = "kind:picker"

// seq is a default's keys, which are fixed: an error is a bug.
func seq(s string) keymap.Sequence {
	q, err := keymap.ParseSequence(s, nil)
	if err != nil {
		panic(err)
	}
	return q
}

// bind is a default binding of keys to id in scope, where when holds.
func bind(scope command.Scope, keys string, id command.ID, when string) keymap.Default {
	return keymap.Default{
		Keys: seq(keys), Command: id, Scope: scope, When: when,
		Location: keymap.Location{Layer: keymap.DefaultLayer},
	}
}

// DefaultRules is the palette's default bindings (0008-MADR A3 item 1),
// for a program's one keymap beside workspace.DefaultRules, so that one
// user file rebinds both: ctrl+p runs palette.open, the program's
// OpenCommand, where no modal overlay is open; in an open picker, the
// "kind:picker" scope, up and ctrl+p, and down and ctrl+n, move the
// selection, pgup and pgdown page, enter runs the selected hit, and tab
// completes the query to it; in a completion pop-up, "overlay:completion",
// the same moves, and tab and enter accept, shadowing the pane's own
// bindings of those keys only while it is open; CompletionRules gives a
// pop-up under another ID its own. esc stays the workspace's:
// a picker takes it while its query is not empty, to clear it, and esc
// closes a pop-up. None is ctrl+c, which is the program's.
//
// Without these rules a picker's keys reach its input, and nothing moves
// the selection.
func DefaultRules() []keymap.Default {
	return append([]keymap.Default{
		bind(command.Global, "ctrl+p", OpenID, "!workspace.modal"),
		bind(PickerScope, "up", SelectPrevID, ""),
		bind(PickerScope, "ctrl+p", SelectPrevID, ""),
		bind(PickerScope, "down", SelectNextID, ""),
		bind(PickerScope, "ctrl+n", SelectNextID, ""),
		bind(PickerScope, "pgup", PagePrevID, ""),
		bind(PickerScope, "pgdown", PageNextID, ""),
		bind(PickerScope, "enter", AcceptID, ""),
		bind(PickerScope, "tab", CompleteID, ""),
	}, CompletionRules(completionID)...)
}

// CompletionRules is the completion pop-up's bindings in the scope
// "overlay:<id>": up and ctrl+p, and down and ctrl+n, move the selection,
// and tab and enter accept. A non-modal overlay's own scopes are not on the
// key path, its "overlay:<id>" is (0008-MADR A3 item 6), so a pop-up opened
// WithID(id) needs these for its id; DefaultRules holds them for
// "completion", the default.
func CompletionRules(id string) []keymap.Default {
	scope := command.Scope("overlay:" + id)
	return []keymap.Default{
		bind(scope, "up", SelectPrevID, ""),
		bind(scope, "ctrl+p", SelectPrevID, ""),
		bind(scope, "down", SelectNextID, ""),
		bind(scope, "ctrl+n", SelectNextID, ""),
		bind(scope, "tab", AcceptID, ""),
		bind(scope, "enter", AcceptID, ""),
	}
}
