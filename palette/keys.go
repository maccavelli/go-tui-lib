package palette

import (
	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
)

// The commands a picker's keys are bound to (0008-MADR A3 item 1). The
// program runs palette.open, through OpenCommand; the picker runs the
// others itself, as a workspace.CommandConsumer, in the order their keys
// were pressed.
const (
	openID     command.ID = "palette.open"
	nextID     command.ID = "palette.select.next"
	prevID     command.ID = "palette.select.prev"
	pageNextID command.ID = "palette.page.next"
	pagePrevID command.ID = "palette.page.prev"
	acceptID   command.ID = "palette.accept"
	completeID command.ID = "palette.complete"
)

// pickerScope is the key scope an open picker names, as a
// workspace.KeyContexter.
const pickerScope command.Scope = "kind:picker"

// completionScope is a completion pop-up's key scope under its default ID:
// a non-modal overlay's own scopes are not on the key path, its
// "overlay:<id>" is (0008-MADR A3 item 6).
const completionScope command.Scope = "overlay:" + completionID

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
// bindings of those keys only while it is open. esc stays the workspace's:
// a picker takes it while its query is not empty, to clear it, and esc
// closes a pop-up. None is ctrl+c, which is the program's.
//
// Without these rules a picker's keys reach its input, and nothing moves
// the selection.
func DefaultRules() []keymap.Default {
	return []keymap.Default{
		bind(command.Global, "ctrl+p", openID, "!workspace.modal"),
		bind(pickerScope, "up", prevID, ""),
		bind(pickerScope, "ctrl+p", prevID, ""),
		bind(pickerScope, "down", nextID, ""),
		bind(pickerScope, "ctrl+n", nextID, ""),
		bind(pickerScope, "pgup", pagePrevID, ""),
		bind(pickerScope, "pgdown", pageNextID, ""),
		bind(pickerScope, "enter", acceptID, ""),
		bind(pickerScope, "tab", completeID, ""),
		bind(completionScope, "up", prevID, ""),
		bind(completionScope, "ctrl+p", prevID, ""),
		bind(completionScope, "down", nextID, ""),
		bind(completionScope, "ctrl+n", nextID, ""),
		bind(completionScope, "tab", acceptID, ""),
		bind(completionScope, "enter", acceptID, ""),
	}
}
