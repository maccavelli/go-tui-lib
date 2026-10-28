package lineedit

import (
	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
)

// The commands a Composer's keys are bound to, exported so a program binds
// them under its own keys (docs/decisions/0023-MADR-line-mode-components.md
// D6). The Composer runs them itself; a command of another ID that a key in
// ComposerScope reaches is taken and does nothing. ask's reason and pick's
// query run the ones that edit one line (A5 item 4).
const (
	CharLeftID            command.ID = "lineedit.char.left"              // a cluster left
	CharRightID           command.ID = "lineedit.char.right"             // a cluster right
	WordLeftID            command.ID = "lineedit.word.left"              // to the start of this or the last word
	WordRightID           command.ID = "lineedit.word.right"             // to the end of this or the next word
	LineStartID           command.ID = "lineedit.line.start"             // to the line's start
	LineEndID             command.ID = "lineedit.line.end"               // to the line's end
	UpID                  command.ID = "lineedit.up"                     // a row up, or the history's older entry from the first row
	DownID                command.ID = "lineedit.down"                   // a row down, or the history's newer entry from the last row
	DeleteBackID          command.ID = "lineedit.delete.back"            // the cluster before the cursor
	DeleteForwardID       command.ID = "lineedit.delete.forward"         // the cluster at the cursor
	DeleteOrEOFID         command.ID = "lineedit.delete.forward-or-eof"  // the cluster at the cursor, or the end of input on an empty line
	DeleteWordBackID      command.ID = "lineedit.delete.word-back"       // back to the start of this or the last word
	DeleteWordForwardID   command.ID = "lineedit.delete.word-forward"    // to the end of this or the next word
	DeleteSpaceWordBackID command.ID = "lineedit.delete.space-word-back" // back to white space, as unix-word-rubout
	DeleteToLineEndID     command.ID = "lineedit.delete.to-line-end"     // to the line's end
	DeleteToLineStartID   command.ID = "lineedit.delete.to-line-start"   // to the line's start
	TransposeID           command.ID = "lineedit.transpose"              // the cluster before the cursor over the one at it
	CompleteID            command.ID = "lineedit.complete"               // the Completer's completion
	SubmitID              command.ID = "lineedit.submit"                 // the text, or a newline after a trailing backslash
	NewlineID             command.ID = "lineedit.newline"                // a line break at the cursor
	ClearID               command.ID = "lineedit.clear"                  // the screen cleared, with WithClearScreen
	ExternalID            command.ID = "lineedit.external"               // the external editor, with WithExternalEditor
)

// ComposerScope is the key scope a Composer's keys are resolved in. ask
// and pick resolve their own scope first, then this one.
const ComposerScope command.Scope = "kind:composer"

// seq is a default's keys, which are fixed: an error is a bug.
func seq(s string) keymap.Sequence {
	q, err := keymap.ParseSequence(s, nil)
	if err != nil {
		panic(err)
	}
	return q
}

// bind is a default binding of keys to id in ComposerScope.
func bind(keys string, id command.ID) keymap.Default {
	return keymap.Default{
		Keys: seq(keys), Command: id, Scope: ComposerScope,
		Location: keymap.Location{Layer: keymap.DefaultLayer},
	}
}

// DefaultRules is the Composer's default bindings, in ComposerScope, for a
// program's one keymap beside its others, so that one user file rebinds
// them all: gobble's editor's keys, Readline's words for alt+b, alt+f,
// alt+d and alt+backspace (A3 item 9, A5 item 6), alt+enter and
// shift+enter for a newline (A3 item 10), shift+enter bound only where the
// keymap's features and facts say the terminal sends it apart from enter,
// as keymap binds a default it can deliver; ctrl+l to clear the screen (A5
// item 2) and ctrl+x ctrl+e for the external editor (A5 item 3). ctrl+c is
// not among them: it interrupts outside the keymap, and stays the
// program's.
func DefaultRules() []keymap.Default {
	var out []keymap.Default
	for _, b := range []struct {
		id   command.ID
		keys []string
	}{
		{CharLeftID, []string{"left", "ctrl+b"}},
		{CharRightID, []string{"right", "ctrl+f"}},
		{WordLeftID, []string{"alt+b", "alt+left", "ctrl+left"}},
		{WordRightID, []string{"alt+f", "alt+right", "ctrl+right"}},
		{LineStartID, []string{"home", "ctrl+a"}},
		{LineEndID, []string{"end", "ctrl+e"}},
		{UpID, []string{"up", "ctrl+p"}},
		{DownID, []string{"down", "ctrl+n"}},
		{DeleteBackID, []string{"backspace", "ctrl+h"}},
		{DeleteForwardID, []string{"delete"}},
		{DeleteOrEOFID, []string{"ctrl+d"}},
		{DeleteWordBackID, []string{"alt+backspace"}},
		{DeleteWordForwardID, []string{"alt+d"}},
		{DeleteSpaceWordBackID, []string{"ctrl+w"}},
		{DeleteToLineEndID, []string{"ctrl+k"}},
		{DeleteToLineStartID, []string{"ctrl+u"}},
		{TransposeID, []string{"ctrl+t"}},
		{CompleteID, []string{"tab"}},
		{SubmitID, []string{"enter", "ctrl+j"}},
		{NewlineID, []string{"alt+enter", "shift+enter"}},
		{ClearID, []string{"ctrl+l"}},
		{ExternalID, []string{"ctrl+x ctrl+e"}},
	} {
		for _, k := range b.keys {
			out = append(out, bind(k, b.id))
		}
	}
	return out
}

// defaultKeymap is DefaultRules alone, which always build.
func defaultKeymap() *keymap.Keymap {
	k, err := keymap.New(keymap.Settings{}, DefaultRules())
	if err != nil {
		panic(err)
	}
	return k
}
