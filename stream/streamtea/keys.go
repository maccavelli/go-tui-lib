package streamtea

import (
	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
)

// The commands a DocPane's keys are bound to, exported so a program binds
// them under its own keys and scopes
// (docs/decisions/0009-MADR-streaming-content-engine.md A7 item 2). The
// pane runs them itself, as a workspace.CommandConsumer, in the order
// their keys were pressed.
const (
	ScrollUpID   command.ID = "stream.scroll.up"   // scrolls up a row
	ScrollDownID command.ID = "stream.scroll.down" // scrolls down a row
	PageUpID     command.ID = "stream.page.up"     // scrolls up a page, the pane's height
	PageDownID   command.ID = "stream.page.down"   // scrolls down a page
	TopID        command.ID = "stream.top"         // goes to the document's start
	BottomID     command.ID = "stream.bottom"      // goes to its end, which the pane then follows
)

// DocPaneScope is the key scope a DocPane names, as a
// workspace.KeyContexter.
const DocPaneScope command.Scope = "kind:docpane"

// seq is a default's keys, which are fixed: an error is a bug.
func seq(s string) keymap.Sequence {
	q, err := keymap.ParseSequence(s, nil)
	if err != nil {
		panic(err)
	}
	return q
}

// bind is a default binding of keys to id in the DocPane's scope.
func bind(keys string, id command.ID) keymap.Default {
	return keymap.Default{
		Keys: seq(keys), Command: id, Scope: DocPaneScope,
		Location: keymap.Location{Layer: keymap.DefaultLayer},
	}
}

// DefaultRules is the DocPane's default bindings, for a program's one
// keymap beside workspace.DefaultRules, so that one user file rebinds
// both: in the "kind:docpane" scope, up and down scroll a row, pgup and
// pgdown a page, and home and end go to the document's start and end.
// None is ctrl+c, which is the program's.
//
// Without these rules no key scrolls a DocPane, and the wheel alone does.
func DefaultRules() []keymap.Default {
	return []keymap.Default{
		bind("up", ScrollUpID),
		bind("down", ScrollDownID),
		bind("pgup", PageUpID),
		bind("pgdown", PageDownID),
		bind("home", TopID),
		bind("end", BottomID),
	}
}
