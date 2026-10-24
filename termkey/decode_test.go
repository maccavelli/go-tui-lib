package termkey

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// decodeCase is bytes in and the inputs a Decoder gives for them, read
// whole and flushed. A want's Raw is compared for InputUnknown only; a
// key's Raw is checked by the rows' bytes adding up.
type decodeCase struct {
	name string
	in   string
	want []Input
}

func kc(code rune, mod Mod) Input { return Input{Kind: InputKey, Key: KeyEvent{Code: code, Mod: mod}} }

func kt(code rune, text string) Input {
	return Input{Kind: InputKey, Key: KeyEvent{Code: code, Text: text}}
}

func ke(k KeyEvent) Input { return Input{Kind: InputKey, Key: k} }

func unk(raw string) Input { return Input{Kind: InputUnknown, Raw: []byte(raw)} }

func pst(text string) Input { return Input{Kind: InputPaste, Text: text} }

// decodeAll is what d gives for s, read whole and flushed.
func decodeAll(d *Decoder, s string) []Input { return d.Flush(d.Decode([]byte(s), nil)) }

// sameInputs reports whether got is want, Raw compared for InputUnknown.
func sameInputs(got, want []Input) bool {
	return slices.EqualFunc(got, want, func(g, w Input) bool {
		if g.Kind != w.Kind || g.Key != w.Key || g.Text != w.Text || g.Truncated != w.Truncated {
			return false
		}
		return g.Kind != InputUnknown || bytes.Equal(g.Raw, w.Raw)
	})
}

// show is ins as a test failure prints them.
func show(ins []Input) string {
	var parts []string
	for _, in := range ins {
		switch in.Kind {
		case InputKey:
			parts = append(parts, fmt.Sprintf("key{%+v raw %q}", in.Key, in.Raw))
		case InputPaste:
			parts = append(parts, fmt.Sprintf("paste{%q truncated %v}", in.Text, in.Truncated))
		default:
			parts = append(parts, fmt.Sprintf("unknown{%q truncated %v}", in.Raw, in.Truncated))
		}
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func textCases() []decodeCase {
	return []decodeCase{
		{"a letter", "a", []Input{kt('a', "a")}},
		{"an upper-case letter", "A", []Input{ke(KeyEvent{Code: 'a', Mod: ModShift, ShiftedCode: 'A', Text: "A"})}},
		{"a digit", "1", []Input{kt('1', "1")}},
		{"a tilde", "~", []Input{kt('~', "~")}},
		{"two bytes of UTF-8", "é", []Input{kt('é', "é")}},
		{"two CJK characters", "日本", []Input{kt('日', "日"), kt('本', "本")}},
		{"four bytes of UTF-8", "\xf0\x9f\x99\x82", []Input{kt(0x1f642, "\xf0\x9f\x99\x82")}},
		{"the replacement character itself", "\xef\xbf\xbd", []Input{kt(0xfffd, "\xef\xbf\xbd")}},
		{"a byte no UTF-8 begins with", "\xff", []Input{unk("\xff")}},
		// A continuation byte on its own is a C1 control, as ultraviolet reads it.
		{"a cut character before a letter", "\xe2\x82a", []Input{unk("\xe2"), kc('B', ModCtrl|ModAlt), kt('a', "a")}},
		{"a cut character at the end", "\xe2\x82", []Input{unk("\xe2"), kc('B', ModCtrl|ModAlt)}},
		{"a letter, then a combining accent", "e\xcc\x81", []Input{kt('e', "e"), kt(0x301, "\xcc\x81")}},
		{"a grapheme of two runes", "\xf0\x9f\x91\x8d\xf0\x9f\x8f\xbd", []Input{kt(0x1f44d, "\xf0\x9f\x91\x8d"), kt(0x1f3fd, "\xf0\x9f\x8f\xbd")}},
		{"a C1 control", "\x85", []Input{kc('E', ModCtrl|ModAlt)}},
	}
}

func controlCases() []decodeCase {
	cs := []decodeCase{
		{"NUL", "\x00", []Input{kc(KeySpace, ModCtrl)}},
		{"BS", "\x08", []Input{kc('h', ModCtrl)}},
		{"HT", "\x09", []Input{kc(KeyTab, 0)}},
		{"CR", "\x0d", []Input{kc(KeyEnter, 0)}},
		{"ESC", "\x1b", []Input{kc(KeyEscape, 0)}},
		{"DEL", "\x7f", []Input{kc(KeyBackspace, 0)}},
		{"SP", " ", []Input{kt(KeySpace, " ")}},
		{"FS", "\x1c", []Input{kc('\\', ModCtrl)}},
		{"GS", "\x1d", []Input{kc(']', ModCtrl)}},
		{"RS", "\x1e", []Input{kc('^', ModCtrl)}},
		{"US", "\x1f", []Input{kc('_', ModCtrl)}},
	}
	for c := byte(0x01); c <= 0x1a; c++ {
		if c == 0x08 || c == 0x09 || c == 0x0d {
			continue
		}
		letter := rune('a' + c - 1)
		cs = append(cs, decodeCase{fmt.Sprintf("ctrl+%c", letter), string([]byte{c}), []Input{kc(letter, ModCtrl)}})
	}
	return cs
}

func altCases() []decodeCase {
	return []decodeCase{
		{"alt+a", "\x1ba", []Input{kc('a', ModAlt)}},
		{"alt+shift+a", "\x1bA", []Input{ke(KeyEvent{Code: 'a', Mod: ModShift | ModAlt, ShiftedCode: 'A'})}},
		{"alt+é", "\x1bé", []Input{kc('é', ModAlt)}},
		{"alt+space", "\x1b ", []Input{kc(KeySpace, ModAlt)}},
		{"ctrl+alt+a", "\x1b\x01", []Input{kc('a', ModCtrl|ModAlt)}},
		{"alt+escape", "\x1b\x1b", []Input{kc(KeyEscape, ModAlt)}},
		{"alt+up", "\x1b\x1b[A", []Input{kc(KeyUp, ModAlt)}},
		{"alt+[", "\x1b[", []Input{kc('[', ModAlt)}},
		{"alt+shift+o", "\x1bO", []Input{kc('o', ModShift|ModAlt)}},
		{"alt+shift+p", "\x1bP", []Input{kc('p', ModShift|ModAlt)}},
		{"alt+]", "\x1b]", []Input{kc(']', ModAlt)}},
		{"alt+_", "\x1b_", []Input{kc('_', ModAlt)}},
		{"alt+^", "\x1b^", []Input{kc('^', ModAlt)}},
		{"alt+shift+x", "\x1bX", []Input{kc('x', ModShift|ModAlt)}},
		{"ESC before a byte no key begins with", "\x1b\xff", []Input{kc(KeyEscape, 0), unk("\xff")}},
		{"ESC before a reply", "\x1b\x1b[?1u", []Input{kc(KeyEscape, 0), unk("\x1b[?1u")}},
		{"ESC before a paste", "\x1b\x1b[200~ab\x1b[201~", []Input{kc(KeyEscape, 0), pst("ab")}},
	}
}

// xtermMods is each xterm modifier parameter's modifiers: 1 plus the bits.
var xtermMods = map[int]Mod{
	2: ModShift, 3: ModAlt, 4: ModShift | ModAlt, 5: ModCtrl,
	6: ModShift | ModCtrl, 7: ModAlt | ModCtrl, 8: ModShift | ModAlt | ModCtrl,
}

// tildeKeys is each CSI n ~ key.
var tildeKeys = map[int]rune{
	1: KeyHome, 2: KeyInsert, 3: KeyDelete, 4: KeyEnd, 5: KeyPgUp, 6: KeyPgDown, 7: KeyHome, 8: KeyEnd,
	11: KeyF1, 12: KeyF2, 13: KeyF3, 14: KeyF4, 15: KeyF5,
	17: KeyF6, 18: KeyF7, 19: KeyF8, 20: KeyF9, 21: KeyF10,
	23: KeyF11, 24: KeyF12, 25: KeyF13, 26: KeyF14,
	28: KeyF15, 29: KeyF16, 31: KeyF17, 32: KeyF18, 33: KeyF19, 34: KeyF20,
}

func legacyCases() []decodeCase {
	var cs []decodeCase
	for _, l := range []struct {
		fin  byte
		code rune
	}{
		{'A', KeyUp}, {'B', KeyDown}, {'C', KeyRight}, {'D', KeyLeft}, {'E', KeyBegin},
		{'F', KeyEnd}, {'H', KeyHome}, {'P', KeyF1}, {'Q', KeyF2}, {'R', KeyF3}, {'S', KeyF4},
	} {
		cs = append(cs, decodeCase{fmt.Sprintf("CSI %c", l.fin), "\x1b[" + string(l.fin), []Input{kc(l.code, 0)}})
		for m := 2; m <= 8; m++ {
			cs = append(cs, decodeCase{fmt.Sprintf("CSI 1;%d %c", m, l.fin), fmt.Sprintf("\x1b[1;%d%c", m, l.fin), []Input{kc(l.code, xtermMods[m])}})
		}
	}
	for _, n := range slices.Sorted(maps.Keys(tildeKeys)) {
		code := tildeKeys[n]
		cs = append(cs, decodeCase{fmt.Sprintf("CSI %d ~", n), fmt.Sprintf("\x1b[%d~", n), []Input{kc(code, 0)}})
		for m := 2; m <= 8; m++ {
			cs = append(cs, decodeCase{fmt.Sprintf("CSI %d;%d ~", n, m), fmt.Sprintf("\x1b[%d;%d~", n, m), []Input{kc(code, xtermMods[m])}})
		}
	}
	return append(cs,
		decodeCase{"shift+up, rxvt", "\x1b[a", []Input{kc(KeyUp, ModShift)}},
		decodeCase{"shift+left, rxvt", "\x1b[d", []Input{kc(KeyLeft, ModShift)}},
		decodeCase{"shift+tab", "\x1b[Z", []Input{kc(KeyTab, ModShift)}},
		decodeCase{"ctrl+insert, urxvt", "\x1b[2^", []Input{kc(KeyInsert, ModCtrl)}},
		decodeCase{"ctrl+shift+delete, urxvt", "\x1b[3@", []Input{kc(KeyDelete, ModCtrl|ModShift)}},
		decodeCase{"shift+page up, urxvt, at the end", "\x1b[5$", []Input{kc(KeyPgUp, ModShift)}},
		decodeCase{"shift+page up, urxvt, then up", "\x1b[5$\x1b[A", []Input{kc(KeyPgUp, ModShift), kc(KeyUp, 0)}},
		decodeCase{"up, repeating", "\x1b[1;1:2A", []Input{ke(KeyEvent{Code: KeyUp, IsRepeat: true})}},
		decodeCase{"ctrl+up, released", "\x1b[1;5:3A", []Input{ke(KeyEvent{Code: KeyUp, Mod: ModCtrl, Release: true})}},
		decodeCase{"delete, repeating", "\x1b[3;1:2~", []Input{ke(KeyEvent{Code: KeyDelete, IsRepeat: true})}},
		decodeCase{"up, 8-bit CSI", "\x9bA", []Input{kc(KeyUp, 0)}},
		decodeCase{"delete, 8-bit CSI", "\x9b3~", []Input{kc(KeyDelete, 0)}},
		decodeCase{"a cursor position report on row 2", "\x1b[2;5R", []Input{unk("\x1b[2;5R")}},
		decodeCase{"a code no key has", "\x1b[9~", []Input{unk("\x1b[9~")}},
	)
}

func ss3Cases() []decodeCase {
	cs := []decodeCase{
		{"SS3 up", "\x1bOA", []Input{kc(KeyUp, 0)}},
		{"SS3 down", "\x1bOB", []Input{kc(KeyDown, 0)}},
		{"SS3 right", "\x1bOC", []Input{kc(KeyRight, 0)}},
		{"SS3 left", "\x1bOD", []Input{kc(KeyLeft, 0)}},
		{"SS3 begin", "\x1bOE", []Input{kc(KeyBegin, 0)}},
		{"SS3 end", "\x1bOF", []Input{kc(KeyEnd, 0)}},
		{"SS3 home", "\x1bOH", []Input{kc(KeyHome, 0)}},
		{"SS3 F1", "\x1bOP", []Input{kc(KeyF1, 0)}},
		{"SS3 F2", "\x1bOQ", []Input{kc(KeyF2, 0)}},
		{"SS3 F3", "\x1bOR", []Input{kc(KeyF3, 0)}},
		{"SS3 F4", "\x1bOS", []Input{kc(KeyF4, 0)}},
		{"SS3 ctrl+up, rxvt", "\x1bOa", []Input{kc(KeyUp, ModCtrl)}},
		{"SS3 keypad enter", "\x1bOM", []Input{kc(KeyKpEnter, 0)}},
		{"SS3 keypad equal", "\x1bOX", []Input{kc(KeyKpEqual, 0)}},
		{"SS3 keypad multiply", "\x1bOj", []Input{kc(KeyKpMultiply, 0)}},
		{"SS3 keypad 0", "\x1bOp", []Input{kc(KeyKp0, 0)}},
		{"SS3 keypad 9", "\x1bOy", []Input{kc(KeyKp9, 0)}},
		{"SS3 F1, 8-bit", "\x8fP", []Input{kc(KeyF1, 0)}},
		{"SS3 a final no key has", "\x1bOz", []Input{unk("\x1bOz")}},
	}
	for m := 2; m <= 8; m++ {
		cs = append(cs, decodeCase{fmt.Sprintf("SS3 %d P", m), fmt.Sprintf("\x1bO%dP", m), []Input{kc(KeyF1, xtermMods[m])}})
	}
	return cs
}

func otherKeysCases() []decodeCase {
	return []decodeCase{
		{"modifyOtherKeys ctrl+a", "\x1b[27;5;97~", []Input{kc('a', ModCtrl)}},
		{"modifyOtherKeys shift+A", "\x1b[27;2;65~", []Input{ke(KeyEvent{Code: 'A', Mod: ModShift, Text: "A"})}},
		{"modifyOtherKeys a, no modifier", "\x1b[27;1;97~", []Input{kt('a', "a")}},
		{"modifyOtherKeys ctrl+enter", "\x1b[27;5;13~", []Input{kc(KeyEnter, ModCtrl)}},
		{"modifyOtherKeys alt+tab", "\x1b[27;3;9~", []Input{kc(KeyTab, ModAlt)}},
		{"modifyOtherKeys ctrl+backspace, DEL", "\x1b[27;5;127~", []Input{kc(KeyBackspace, ModCtrl)}},
		{"modifyOtherKeys ctrl+shift+backspace, BS", "\x1b[27;6;8~", []Input{kc(KeyBackspace, ModCtrl|ModShift)}},
		{"modifyOtherKeys ctrl+escape", "\x1b[27;5;27~", []Input{kc(KeyEscape, ModCtrl)}},
		{"modifyOtherKeys with two parameters", "\x1b[27;5~", []Input{unk("\x1b[27;5~")}},
		{"formatOtherKeys ctrl+a", "\x1b[97;5u", []Input{kc('a', ModCtrl)}},
		{"formatOtherKeys shift+enter", "\x1b[13;2u", []Input{kc(KeyEnter, ModShift)}},
		{"formatOtherKeys ctrl+tab", "\x1b[9;5u", []Input{kc(KeyTab, ModCtrl)}},
	}
}

func kittyCases() []decodeCase {
	cs := []decodeCase{
		{"Kitty a", "\x1b[97u", []Input{kt('a', "a")}},
		{"Kitty a, pressed", "\x1b[97;1:1u", []Input{kt('a', "a")}},
		{"Kitty a, repeating", "\x1b[97;1:2u", []Input{ke(KeyEvent{Code: 'a', Text: "a", IsRepeat: true})}},
		{"Kitty a, released", "\x1b[97;1:3u", []Input{ke(KeyEvent{Code: 'a', Text: "a", Release: true})}},
		{"Kitty ctrl+a, released", "\x1b[97;5:3u", []Input{ke(KeyEvent{Code: 'a', Mod: ModCtrl, Release: true})}},
		{"Kitty shift+a, its shifted key", "\x1b[97:65;2u", []Input{ke(KeyEvent{Code: 'a', Mod: ModShift, ShiftedCode: 'A', Text: "A"})}},
		{"Kitty ctrl+ф, its base key", "\x1b[1092::97;5u", []Input{ke(KeyEvent{Code: 'ф', Mod: ModCtrl, BaseCode: 'a'})}},
		{"Kitty shift+ф, both keys", "\x1b[1092:1060:97;2u", []Input{ke(KeyEvent{Code: 'ф', Mod: ModShift, ShiftedCode: 'Ф', BaseCode: 'a', Text: "Ф"})}},
		{"Kitty shift+a, its text", "\x1b[97;2;65u", []Input{ke(KeyEvent{Code: 'a', Mod: ModShift, Text: "A"})}},
		{"Kitty a, two runes of text", "\x1b[97;;65:66u", []Input{kt('a', "AB")}},
		{"Kitty escape", "\x1b[27u", []Input{kc(KeyEscape, 0)}},
		{"Kitty enter", "\x1b[13u", []Input{kc(KeyEnter, 0)}},
		{"Kitty backspace", "\x1b[127u", []Input{kc(KeyBackspace, 0)}},
		{"Kitty ctrl+a as C0, WezTerm", "\x1b[1u", []Input{kc('a', ModCtrl)}},
		{"Kitty F1, private use", "\x1b[57364u", []Input{kc(KeyF1, 0)}},
		{"Kitty ctrl+F1", "\x1b[57364;5u", []Input{kc(KeyF1, ModCtrl)}},
		{"Kitty keypad 0", "\x1b[57399u", []Input{kt(KeyKp0, "0")}},
		{"Kitty keypad enter", "\x1b[57414u", []Input{kc(KeyKpEnter, 0)}},
		{"Kitty left shift, pressed", "\x1b[57441;2u", []Input{kc(KeyLeftShift, ModShift)}},
		{"Kitty, a code that is no rune", "\x1b[1114112u", []Input{kt(0xfffd, "\xef\xbf\xbd")}},
		{"Kitty with no parameters", "\x1b[u", []Input{unk("\x1b[u")}},
	}
	for _, b := range []struct {
		bit  int
		mod  Mod
		text string
	}{
		{1, ModShift, "A"}, {2, ModAlt, ""}, {4, ModCtrl, ""}, {8, ModSuper, ""},
		{16, ModHyper, ""}, {32, ModMeta, ""}, {64, ModCapsLock, "A"}, {128, ModNumLock, "a"},
	} {
		cs = append(cs, decodeCase{fmt.Sprintf("Kitty modifier bit %d", b.bit), fmt.Sprintf("\x1b[97;%du", b.bit+1),
			[]Input{ke(KeyEvent{Code: 'a', Mod: b.mod, Text: b.text})}})
	}
	return cs
}

func pasteCases() []decodeCase {
	return []decodeCase{
		{"a paste", "\x1b[200~hello\x1b[201~", []Input{pst("hello")}},
		{"a paste of controls", "\x1b[200~a\r\nb\tc\x1b[201~", []Input{pst("a\r\nb\tc")}},
		{"an empty paste", "\x1b[200~\x1b[201~", []Input{pst("")}},
		{"a paste with invalid UTF-8", "\x1b[200~a\xffb\x1b[201~", []Input{pst("ab")}},
		{"a paste holding a key's sequence", "\x1b[200~a\x1b[Ab\x1b[201~", []Input{pst("a\x1b[Ab")}},
		{"a paste holding an OSC", "\x1b[200~a\x1b]0;t\x07b\x1b[201~", []Input{pst("a\x1b]0;t\x07b")}},
		{"keys around a paste", "x\x1b[200~p\x1b[201~y", []Input{kt('x', "x"), pst("p"), kt('y', "y")}},
		{"a paste not ended", "\x1b[200~abc", []Input{pst("abc")}},
		{"a paste end with no start", "\x1b[201~", []Input{unk("\x1b[201~")}},
	}
}

func replyCases() []decodeCase {
	return []decodeCase{
		{"the Kitty flags reply", "\x1b[?1u", []Input{unk("\x1b[?1u")}},
		{"the Kitty flags reply, then a key", "\x1b[?31ua", []Input{unk("\x1b[?31u"), kt('a', "a")}},
		{"DA1", "\x1b[?62;22c", []Input{unk("\x1b[?62;22c")}},
		{"DECRPM", "\x1b[?2026;2$y", []Input{unk("\x1b[?2026;2$y")}},
		{"OSC 11, ST", "\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\", []Input{unk("\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\")}},
		{"OSC 11, BEL", "\x1b]11;rgb:0/0/0\x07", []Input{unk("\x1b]11;rgb:0/0/0\x07")}},
		{"OSC 11, 8-bit", "\x9d11;x\x9c", []Input{unk("\x9d11;x\x9c")}},
		{"OSC 11, a bare ESC at the end", "\x1b]11;x\x1b", []Input{unk("\x1b]11;x\x1b")}},
		{"OSC, cancelled by CAN", "\x1b]11;x\x18a", []Input{unk("\x1b]11;x\x18"), kt('a', "a")}},
		{"OSC, cut by an ESC", "\x1b]11;x\x1bA", []Input{unk("\x1b]11;x"), ke(KeyEvent{Code: 'a', Mod: ModShift | ModAlt, ShiftedCode: 'A'})}},
		{"OSC, not ended", "\x1b]11;x", []Input{unk("\x1b]11;x")}},
		{"ESC ] ESC: alt+], then alt+x", "\x1b]\x1bx", []Input{kc(']', ModAlt), kc('x', ModAlt)}},
		{"DCS XTVERSION", "\x1bP>|kitty(0.39.1)\x1b\\", []Input{unk("\x1bP>|kitty(0.39.1)\x1b\\")}},
		{"APC Kitty graphics", "\x1b_Gi=31;OK\x1b\\", []Input{unk("\x1b_Gi=31;OK\x1b\\")}},
		{"PM", "\x1b^x\x1b\\", []Input{unk("\x1b^x\x1b\\")}},
		{"SOS", "\x1bXx\x9c", []Input{unk("\x1bXx\x9c")}},
		{"a focus report", "\x1b[I", []Input{unk("\x1b[I")}},
		{"an X10 mouse report", "\x1b[M !!a", []Input{unk("\x1b[M !!"), kt('a', "a")}},
		{"an SGR mouse report", "\x1b[<0;1;1M", []Input{unk("\x1b[<0;1;1M")}},
		{"a win32-input-mode key", "\x1b[65;30;97;1;0;1_", []Input{unk("\x1b[65;30;97;1;0;1_")}},
		{"a CSI cut by a control", "\x1b[1;\x01", []Input{unk("\x1b[1;"), kc('a', ModCtrl)}},
		{"33 parameters", "\x1b[" + strings.Repeat("1;", 32) + "1A", []Input{unk("\x1b[" + strings.Repeat("1;", 32) + "1A")}},
	}
}

// tables is every table of D7's encodings.
func tables() map[string][]decodeCase {
	return map[string][]decodeCase{
		"text": textCases(), "control": controlCases(), "alt": altCases(), "legacy": legacyCases(),
		"ss3": ss3Cases(), "otherkeys": otherKeysCases(), "kitty": kittyCases(), "paste": pasteCases(),
		"reply": replyCases(),
	}
}

// allCases is every table's rows.
func allCases() []decodeCase {
	var all []decodeCase
	for _, name := range slices.Sorted(maps.Keys(tables())) {
		all = append(all, tables()[name]...)
	}
	return all
}

func TestDecode(t *testing.T) {
	for name, cases := range tables() {
		t.Run(name, func(t *testing.T) {
			for _, c := range cases {
				var d Decoder
				got := decodeAll(&d, c.in)
				if !sameInputs(got, c.want) {
					t.Errorf("%s: %q gives %s, want %s", c.name, c.in, show(got), show(c.want))
					continue
				}
				if d.Pending() {
					t.Errorf("%s: Pending after Flush", c.name)
				}
				if slices.ContainsFunc(got, func(in Input) bool { return in.Kind == InputPaste }) {
					continue
				}
				var raw []byte
				for _, in := range got {
					raw = append(raw, in.Raw...)
				}
				if string(raw) != c.in {
					t.Errorf("%s: the inputs' bytes add up to %q, want %q", c.name, raw, c.in)
				}
			}
		})
	}
}

// TestCorpus keeps testdata/golden/corpus.golden the tables' inputs, a row
// a line, its name and its quoted bytes. internal/termevent's differential
// test replays it through ultraviolet, which only that package may import
// (MADR D7).
func TestCorpus(t *testing.T) {
	var b strings.Builder
	seen := map[string]bool{}
	for _, c := range allCases() {
		if seen[c.name] {
			t.Errorf("two rows are named %q", c.name)
		}
		seen[c.name] = true
		fmt.Fprintf(&b, "%s\t%s\n", c.name, strconv.QuoteToASCII(c.in))
	}
	tuitest.Text(t, "corpus", b.String())
}

func TestSplit(t *testing.T) {
	for _, c := range allCases() {
		for i := 1; i < len(c.in); i++ {
			var d Decoder
			got := d.Decode([]byte(c.in[:i]), nil)
			got = d.Flush(d.Decode([]byte(c.in[i:]), got))
			if !sameInputs(got, c.want) {
				t.Errorf("%s: %q split at %d gives %s, want %s", c.name, c.in, i, show(got), show(c.want))
			}
		}
		var d Decoder
		var got []Input
		for i := range len(c.in) {
			got = d.Decode([]byte{c.in[i]}, got)
		}
		if got = d.Flush(got); !sameInputs(got, c.want) {
			t.Errorf("%s: %q a byte at a time gives %s, want %s", c.name, c.in, show(got), show(c.want))
		}
	}
}

func TestFlushEscape(t *testing.T) {
	var d Decoder
	if got := d.Decode([]byte("\x1b"), nil); len(got) != 0 || !d.Pending() {
		t.Fatalf("a lone ESC gives %s, Pending %v; want nothing yet, and Pending", show(got), d.Pending())
	}
	if got := d.Flush(nil); !sameInputs(got, []Input{kc(KeyEscape, 0)}) || d.Pending() {
		t.Errorf("Flush gives %s, Pending %v; want escape, and nothing pending", show(got), d.Pending())
	}
	for _, c := range []decodeCase{
		{"ESC [", "\x1b[", []Input{kc('[', ModAlt)}},
		{"a CSI cut by a read", "\x1b[1;", []Input{unk("\x1b[1;")}},
		{"an OSC cut by a read", "\x1b]11;rgb:0", []Input{unk("\x1b]11;rgb:0")}},
		{"a paste cut by a read", "\x1b[200~abc\x1b[20", []Input{pst("abc\x1b[20")}},
		{"a UTF-8 character cut by a read", "\xe6\x97", []Input{unk("\xe6"), kc('W', ModCtrl|ModAlt)}},
	} {
		var d Decoder
		if got := d.Decode([]byte(c.in), nil); len(got) != 0 || !d.Pending() {
			t.Errorf("%s: Decode gives %s, Pending %v; want nothing yet, and Pending", c.name, show(got), d.Pending())
			continue
		}
		if got := d.Flush(nil); !sameInputs(got, c.want) || d.Pending() {
			t.Errorf("%s: Flush gives %s, Pending %v; want %s", c.name, show(got), d.Pending(), show(c.want))
		}
	}
	d = Decoder{}
	if got := d.Decode([]byte("a"), nil); d.Pending() || !sameInputs(got, []Input{kt('a', "a")}) {
		t.Errorf("a letter gives %s, Pending %v; want a, and nothing pending", show(got), d.Pending())
	}
}

func TestPasteBounded(t *testing.T) {
	long := "\x1b[200~0123456789abcdef\x1b[201~x"
	want := []Input{{Kind: InputPaste, Text: "01234567", Truncated: true}, kt('x', "x")}
	d := Decoder{MaxPaste: 8}
	if got := decodeAll(&d, long); !sameInputs(got, want) {
		t.Errorf("a 16-byte paste over MaxPaste 8 gives %s, want %s", show(got), show(want))
	}
	d = Decoder{MaxPaste: 8}
	var got []Input
	for i := range len(long) {
		got = d.Decode([]byte{long[i]}, got)
	}
	if got = d.Flush(got); !sameInputs(got, want) {
		t.Errorf("the paste a byte at a time gives %s, want %s", show(got), show(want))
	}
	d = Decoder{MaxPaste: 8}
	if got := decodeAll(&d, "\x1b[200~01234567\x1b[201~"); !sameInputs(got, []Input{pst("01234567")}) {
		t.Errorf("a paste of exactly MaxPaste gives %s, want it whole and not truncated", show(got))
	}
	d = Decoder{}
	got = decodeAll(&d, "\x1b[200~"+strings.Repeat("y", 1<<20+10)+"\x1b[201~")
	if len(got) != 1 || len(got[0].Text) != 1<<20 || !got[0].Truncated {
		t.Errorf("a paste over the default bound gives %d inputs, the first of %d bytes, truncated %v; want one of 1 MiB, truncated",
			len(got), len(got[0].Text), got[0].Truncated)
	}
}

func TestSequenceBounded(t *testing.T) {
	cut := func(s string, n int) Input { return Input{Kind: InputUnknown, Raw: []byte(s[:n]), Truncated: true} }
	osc := "\x1b]11;" + strings.Repeat("z", 40) + "\x1b\\"
	csi := "\x1b[" + strings.Repeat("1", 40) + "A"
	ss3 := "\x1bO" + strings.Repeat("1", 40) + "P"
	for _, c := range []decodeCase{
		{"an OSC over the bound", osc + "x", []Input{cut(osc, 16), kt('x', "x")}},
		{"a CSI over the bound", csi + "x", []Input{cut(csi, 16), kt('x', "x")}},
		{"an SS3 over the bound", ss3 + "x", []Input{cut(ss3, 16), kt('x', "x")}},
		{"a CSI of exactly the bound", "\x1b[" + strings.Repeat("1", 13) + "~", []Input{unk("\x1b[" + strings.Repeat("1", 13) + "~")}},
		{"a CSI one over the bound", "\x1b[" + strings.Repeat("1", 14) + "~x", []Input{cut("\x1b["+strings.Repeat("1", 14), 16), kt('x', "x")}},
		{"an OSC of exactly the bound", "\x1b]" + strings.Repeat("z", 12) + "\x1b\\", []Input{unk("\x1b]" + strings.Repeat("z", 12) + "\x1b\\")}},
		// The ESC of an ST the bound cuts is left out of the cut.
		{"an OSC whose ST crosses the bound", "\x1b]" + strings.Repeat("z", 13) + "\x1b\\x", []Input{cut("\x1b]"+strings.Repeat("z", 13), 15), kt('x', "x")}},
	} {
		d := Decoder{MaxSequence: 16}
		if got := decodeAll(&d, c.in); !sameInputs(got, c.want) {
			t.Errorf("%s: gives %s, want %s", c.name, show(got), show(c.want))
		}
		d = Decoder{MaxSequence: 16}
		var got []Input
		for i := range len(c.in) {
			got = d.Decode([]byte{c.in[i]}, got)
			if len(d.buf) > 16 {
				t.Errorf("%s: holds %d bytes, over MaxSequence", c.name, len(d.buf))
				break
			}
		}
		if got = d.Flush(got); !sameInputs(got, c.want) {
			t.Errorf("%s: a byte at a time gives %s, want %s", c.name, show(got), show(c.want))
		}
	}
	d := Decoder{}
	got := decodeAll(&d, "\x1b]"+strings.Repeat("z", 5000)+"\x07")
	if len(got) != 1 || len(got[0].Raw) != 4<<10 || !got[0].Truncated {
		t.Errorf("an OSC over the default bound gives %d inputs, the first of %d bytes; want one input of 4 KiB, truncated", len(got), len(got[0].Raw))
	}
	d = Decoder{}
	whole := "\x1b]" + strings.Repeat("z", 4<<10-4) + "\x1b\\"
	if got := decodeAll(&d, whole); !sameInputs(got, []Input{unk(whole)}) {
		t.Errorf("an OSC of exactly the default bound gives %d inputs; want it whole", len(got))
	}
}

func FuzzDecode(f *testing.F) {
	for _, c := range allCases() {
		f.Add([]byte(c.in), uint16(len(c.in)/2))
	}
	f.Fuzz(func(t *testing.T, data []byte, split uint16) {
		var whole Decoder
		want := whole.Flush(whole.Decode(data, nil))
		if whole.Pending() {
			t.Fatalf("%q: Pending after Flush", data)
		}
		i := int(split) % (len(data) + 1)
		var parts Decoder
		got := parts.Decode(data[:i], nil)
		got = parts.Flush(parts.Decode(data[i:], got))
		sameRaw := slices.EqualFunc(got, want, func(g, w Input) bool { return bytes.Equal(g.Raw, w.Raw) })
		if !sameInputs(got, want) || !sameRaw {
			t.Fatalf("%q split at %d gives %s, whole gives %s", data, i, show(got), show(want))
		}
	})
}
