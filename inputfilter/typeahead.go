package inputfilter

import (
	"unicode"
	"unicode/utf8"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// maxCapture bounds what a Flusher's Capture holds, which is untrusted.
const maxCapture = 4096

// Typeahead keeps the typing in ins, as grok keeps a prompt typed while it
// starts (grok xai-grok-pager/src/app/event_loop.rs:53-124): text, a
// Backspace, a Shift or Alt Enter, and pastes, up to the first Esc, which
// cuts the rest. A terminal's reports and the other keys are dropped.
func Typeahead(ins []termkey.Input) []termkey.Input {
	var out []termkey.Input
	for _, in := range ins {
		if in.Kind == termkey.InputKey && !in.Key.Release && in.Key.Code == termkey.KeyEscape {
			break
		}
		if typing(in) {
			out = append(out, in)
		}
	}
	return out
}

// typing reports whether in is typing Typeahead keeps.
func typing(in termkey.Input) bool {
	switch {
	case in.Kind == termkey.InputPaste:
		return true
	case in.Kind != termkey.InputKey || in.Key.Release:
		return false
	}
	k := in.Key
	switch k.Code {
	case termkey.KeyBackspace:
		return true
	case termkey.KeyEnter:
		return k.Mod&(termkey.ModShift|termkey.ModAlt) != 0
	}
	if k.Text == "" || k.Mod&^(locks|termkey.ModShift) != 0 {
		return false
	}
	c, _ := utf8.DecodeRuneInString(k.Text)
	return !unicode.IsControl(c)
}
