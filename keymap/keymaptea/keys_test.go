package keymaptea

import (
	"strings"
	"testing"
	"unicode"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/keymap"
)

// keyCase is one key as ultraviolet decodes it, and what each stroke makes
// of it: Matches, and bubbles' key.Matches over BubblesKeys. Line numbers
// are of ultraviolet v0.0.0-20260811164956-006e29f97886's decoder.go.
type keyCase struct {
	name, decoder string
	key           tea.Key
	strokes       map[string][2]bool // stroke: {Matches, key.Matches}
}

var keyCases = []keyCase{
	{"legacy upper-case A", "parseUtf8, :1097-1102: lower-case code, shift, ShiftedCode, Text",
		tea.Key{Code: 'a', ShiftedCode: 'A', Mod: tea.ModShift, Text: "A"},
		map[string][2]bool{"shift+a": {true, true}, "a": {false, false}, "A": {true, true}}},
	{"legacy shifted symbol", "parseUtf8, :1094-1095: the symbol, no shift",
		tea.Key{Code: '?', Text: "?"},
		map[string][2]bool{"?": {true, true}, "shift+/": {false, false}}},
	{"legacy alt+a", "the ESC prefix, :264-265: Text cleared, alt",
		tea.Key{Code: 'a', Mod: tea.ModAlt},
		map[string][2]bool{"alt+a": {true, true}, "a": {false, false}}},
	{"legacy esc", "the ESC prefix, :243: KeyEscape",
		tea.Key{Code: tea.KeyEscape},
		map[string][2]bool{"esc": {true, true}, "escape": {true, true}}},
	{"Kitty shift+a", "parseKittyKeyboard, :1433-1437 ShiftedCode, :1512-1513 Text",
		tea.Key{Code: 'a', ShiftedCode: 'A', Mod: tea.ModShift, Text: "A"},
		map[string][2]bool{"shift+a": {true, true}, "a": {false, false}}},
	{"Kitty shift+/ typing ?", "parseKittyKeyboard, :1512-1513: Text is the shifted key",
		tea.Key{Code: '/', ShiftedCode: '?', Mod: tea.ModShift, Text: "?"},
		// shift+/ matches, but bubbles sees "?": a shifted symbol's text is
		// the layout's, which BubblesKeys cannot know; §2 writes it "?".
		map[string][2]bool{"?": {true, true}, "shift+/": {true, false}, "/": {false, false}}},
	{"Kitty ctrl+a", "parseKittyKeyboard, :1442-1447: Text cleared past shift",
		tea.Key{Code: 'a', Mod: tea.ModCtrl},
		map[string][2]bool{"ctrl+a": {true, true}, "a": {false, false}}},
	{"Kitty AZERTY a, at the US q", "parseKittyKeyboard, :1415-1424: BaseCode",
		tea.Key{Code: 'a', BaseCode: 'q', Text: "a"},
		// A scan stroke names a position; bubbles compares the typed text,
		// so it sees [KeyA] here, not [KeyQ].
		map[string][2]bool{"a": {true, true}, "[KeyQ]": {true, false}, "[KeyA]": {false, true}}},
	{"Kitty keypad Enter", "kittyKeyMap: KeyKpEnter, named enter (amendment A5)",
		tea.Key{Code: tea.KeyKpEnter},
		map[string][2]bool{"enter": {true, true}}},
	{"Kitty keypad 1", "parseKittyKeyboard, :1482-1485: keypad Text",
		tea.Key{Code: tea.KeyKp1, Text: "1"},
		map[string][2]bool{"1": {true, true}}},
	{"Kitty repeated a", "parseKittyKeyboard, :1453-1455: IsRepeat",
		tea.Key{Code: 'a', Text: "a", IsRepeat: true},
		map[string][2]bool{"a": {true, true}, "a:repeat": {true, true}}},
	{"Kitty a, not repeated", "parseKittyKeyboard",
		tea.Key{Code: 'a', Text: "a"},
		map[string][2]bool{"a:repeat": {false, true}}},
	{"Kitty a under caps lock", "parseKittyKeyboard, :1480, :1508-1515: Text upper case",
		tea.Key{Code: 'a', Mod: tea.ModCapsLock, Text: "A"},
		// The lock is ignored, so a matches; bubbles sees "A", which it
		// cannot tell from shift+a (D1).
		map[string][2]bool{"a": {true, false}, "shift+a": {false, true}}},
	{"Windows console a under caps lock", "parseWin32InputKeyEvent, :2030-2048; ensureKeyCase, :2058-2075",
		tea.Key{Code: 'A', Mod: tea.ModCapsLock, Text: "A"},
		map[string][2]bool{"a": {true, false}, "shift+a": {false, true}}},
	{"Windows console shift+a under caps lock", "ensureKeyCase, :2063-2068: ShiftedCode, Text",
		tea.Key{Code: 'a', ShiftedCode: 'A', Mod: tea.ModShift | tea.ModCapsLock, Text: "A"},
		map[string][2]bool{"shift+a": {true, true}, "a": {false, false}}},
	{"Windows console ctrl+a under num lock", "translateControlKeyState, :2082-2101",
		tea.Key{Code: 'a', Mod: tea.ModCtrl | tea.ModNumLock},
		map[string][2]bool{"ctrl+a": {true, true}}},
}

// TestMatches: each key, decoded as ultraviolet decodes it, matches the
// strokes it should through FromTea, with the lock modifiers ignored.
func TestMatches(t *testing.T) {
	for _, c := range keyCases {
		for text, want := range c.strokes {
			if got := mustStroke(t, text).Matches(FromTea(c.key)); got != want[0] {
				t.Errorf("%s (%s): %q.Matches = %v, want %v", c.name, c.decoder, text, got, want[0])
			}
		}
	}
}

// TestBubblesInterop: bubbles' key.Matches over BubblesKeys agrees with
// Matches on every key without a lock modifier and with a written-out
// symbol. Where it cannot, under caps lock and for a shifted symbol written
// as shift and its key, the table pins the difference (D1).
func TestBubblesInterop(t *testing.T) {
	const locks = tea.ModCapsLock | tea.ModNumLock | tea.ModScrollLock
	for _, c := range keyCases {
		msg := tea.KeyPressMsg(c.key)
		for text, want := range c.strokes {
			s := mustStroke(t, text)
			b := key.NewBinding(key.WithKeys(BubblesKeys(s)...))
			got := key.Matches(msg, b)
			if got != want[1] {
				t.Errorf("%s: key.Matches(%q, %q) = %v, want %v", c.name, msg.String(), BubblesKeys(s), got, want[1])
			}
			locked := c.key.Mod&locks != 0
			printable := !s.Scan && unicode.IsPrint(s.Code) && keymap.Stroke{Code: s.Code}.String() == string(s.Code)
			symbol := strings.HasPrefix(text, "shift+") && !unicode.IsLetter(s.Code) && printable
			if !locked && !symbol && s.Event == keymap.Press && !s.Scan && want[0] != want[1] {
				t.Errorf("%s: %q: Matches and key.Matches differ on a key they should agree on", c.name, text)
			}
		}
	}
}
