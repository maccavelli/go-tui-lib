package ask

import "github.com/maccavelli/go-tui-lib/v2/termkey"

// Question is what is asked: the notes before it, its title, the lines
// after the title, the rows of answers, and the answer that cancels.
type Question struct {
	Notes  []string // lines before the title: why only "once" is offered, a limit
	Title  string   // wrapped at spaces, continuation rows indented two
	After  []string // lines after the title: "Patterns: …"
	Rows   []Row
	Cancel Answer // the key that cancels, and its label: "[c] cancel"; it ends the last row
}

// Row is one row of answers, under its label: "Allow", "Reject", or "" for
// a row of its own.
type Row struct {
	Label   string
	Answers []Answer
}

// Answer is one answer the question offers.
type Answer struct {
	ID     string // the caller's
	Key    string // one key, as keymap's notation names it, "1", "f", "shift+n", or the text it types, "N"
	Label  string // "once", "reject and say why"
	Echo   string // the answer in words, left in the scrollback: "allow for this session"; Label when empty
	Reason bool   // a reason is read after the key, up to enter
}

// Response is how the question was answered.
type Response struct {
	Answer   Answer
	Reason   string // what was typed after an answer with Reason, its spaces trimmed
	Canceled bool   // the cancel key, ctrl+c, or the host's Cancel
}

// matches reports whether k chooses an answer whose key is key: k named
// as a binding names it, or, for a key held with no modifier but shift
// and the locks, the text it types.
func matches(key string, k termkey.KeyEvent) bool {
	if key == "" || k.Release {
		return false
	}
	if k.Keystroke() == key {
		return true
	}
	plain := k.Mod&^(termkey.ModShift|termkey.ModCapsLock|termkey.ModNumLock|termkey.ModScrollLock) == 0
	return plain && k.Text != "" && k.Text == key
}
