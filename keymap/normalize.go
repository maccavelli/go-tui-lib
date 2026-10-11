package keymap

import (
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// Normalize is k as the bindings name it, and whether that changed it
// (MADR amendment A1). goos is the system the keys come from, as
// runtime.GOOS names it. Unless f has Disambiguate, a legacy terminal's
// encodings are read as their keys:
//   - a C0 code with no modifier is ctrl and its letter, as ctrl+a for
//     0x01, ctrl+\ to ctrl+_ for 0x1c to 0x1f, and ctrl+space for NUL; tab,
//     enter and escape keep their names;
//   - a raw BS (0x08) is backspace, whatever the modifiers;
//   - an upper-case letter with ctrl and no shift is ctrl+shift and the
//     lower-case letter;
//   - ctrl+5 is ctrl+], and ctrl+4 is ctrl+\, as a legacy terminal sends
//     them.
//
// Two rules apply whatever f holds:
//   - on Windows, a key with ctrl and alt and printable Text is AltGr: it
//     is that text, with no modifier, and the Dispatcher gives it to the
//     focused pane, never to a binding;
//   - a key with super loses any meta and hyper beside it, which some
//     terminals report with it.
//
// Ultraviolet, Bubble Tea's decoder, already reads most C0 bytes as ctrl
// and a letter, and BS as ctrl+h; these rules hold for a key from any
// source.
func Normalize(k tea.Key, f Features, goos string) (tea.Key, bool) {
	if isAltGr(k, goos) {
		r, _ := utf8.DecodeRuneInString(k.Text)
		n := tea.Key{Code: r, Text: k.Text, Mod: k.Mod & locks, BaseCode: k.BaseCode, IsRepeat: k.IsRepeat}
		return n, n != k
	}
	n := k
	if n.Mod.Contains(tea.ModSuper) {
		n.Mod &^= tea.ModMeta | tea.ModHyper
	}
	if f&Disambiguate == 0 {
		mods := n.Mod &^ locks
		switch c := n.Code; {
		case c == 0x08:
			n.Code = tea.KeyBackspace
		case mods != 0:
		case c >= 0x01 && c <= 0x1a && c != tea.KeyTab && c != tea.KeyEnter:
			n.Code, n.Mod, n.Text = c+0x60, n.Mod|tea.ModCtrl, ""
		case c >= 0x1c && c <= 0x1f:
			n.Code, n.Mod, n.Text = c+0x40, n.Mod|tea.ModCtrl, ""
		case c == 0 && n.BaseCode == 0 && n.Text == "":
			// A Windows console key with no code carries its rune in
			// BaseCode, and is not NUL.
			n.Code, n.Mod = tea.KeySpace, n.Mod|tea.ModCtrl
		}
		if mods = n.Mod &^ locks; mods.Contains(tea.ModCtrl) {
			switch c := n.Code; {
			case c < unicode.MaxRune && unicode.IsUpper(c) && !mods.Contains(tea.ModShift):
				n.Code, n.Mod = unicode.ToLower(c), n.Mod|tea.ModShift
			case c == '5':
				n.Code = ']'
			case c == '4':
				n.Code = '\\'
			}
		}
	}
	return n, n != k
}

// isAltGr reports whether k is AltGr on Windows: ctrl and alt, with text
// that is printable.
func isAltGr(k tea.Key, goos string) bool {
	if goos != "windows" || !k.Mod.Contains(tea.ModCtrl|tea.ModAlt) || k.Text == "" {
		return false
	}
	for _, r := range k.Text {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
