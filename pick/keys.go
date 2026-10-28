package pick

import (
	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
)

// The commands a Chooser's keys are bound to
// (docs/decisions/0023-MADR-line-mode-components.md A3.4 item 3), exported
// so a program binds them under its own keys. The Chooser runs them
// itself; a command of another ID that a key in ChooserScope reaches is
// taken and does nothing.
const (
	SelectNextID command.ID = "pick.select.next" // moves the selection down a row
	SelectPrevID command.ID = "pick.select.prev" // moves the selection up a row
	PageNextID   command.ID = "pick.page.next"   // moves the selection down a page of rows
	PagePrevID   command.ID = "pick.page.prev"   // moves the selection up a page of rows
	AcceptID     command.ID = "pick.accept"      // chooses the selected entry
	CancelID     command.ID = "pick.cancel"      // ends the pick, nothing chosen
)

// ChooserScope is the key scope a Chooser's keys are resolved in, and the
// only one: a binding in another scope of the program's keymap does not
// reach it.
const ChooserScope command.Scope = "kind:chooser"

// seq is a default's keys, which are fixed: an error is a bug.
func seq(s string) keymap.Sequence {
	q, err := keymap.ParseSequence(s, nil)
	if err != nil {
		panic(err)
	}
	return q
}

// bind is a default binding of keys to id in ChooserScope.
func bind(keys string, id command.ID) keymap.Default {
	return keymap.Default{
		Keys: seq(keys), Command: id, Scope: ChooserScope,
		Location: keymap.Location{Layer: keymap.DefaultLayer},
	}
}

// DefaultRules is the Chooser's default bindings, in ChooserScope, for a
// program's one keymap beside its others, so that one user file rebinds
// them all: up and ctrl+p, and down and ctrl+n, move the selection, pgup
// and pgdown page, enter chooses, and esc and ctrl+g cancel, as fzf and
// inquire bind them. ctrl+j and ctrl+q are left unbound, since the tools
// disagree on them; ctrl+c is not among them, and cancels outside the
// keymap (A3.4 item 3).
func DefaultRules() []keymap.Default {
	return []keymap.Default{
		bind("up", SelectPrevID),
		bind("ctrl+p", SelectPrevID),
		bind("down", SelectNextID),
		bind("ctrl+n", SelectNextID),
		bind("pgup", PagePrevID),
		bind("pgdown", PageNextID),
		bind("enter", AcceptID),
		bind("esc", CancelID),
		bind("ctrl+g", CancelID),
	}
}

// defaultKeymap is DefaultRules alone, which always build.
func defaultKeymap() *keymap.Keymap {
	k, err := keymap.New(keymap.Settings{}, DefaultRules())
	if err != nil {
		panic(err)
	}
	return k
}
