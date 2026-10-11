package keymap

import (
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// randomStroke is a canonical stroke: any modifier set, a named key, an F
// key, a printable rune that is not an upper-case letter, or a scan code,
// with any event.
func randomStroke(r *rand.Rand) Stroke {
	var s Stroke
	for _, m := range modifiers {
		if r.IntN(3) == 0 {
			s.Mod |= m.mod
		}
	}
	switch r.IntN(4) {
	case 0:
		names := slices.Sorted(func(yield func(string) bool) {
			for n := range nameCodes {
				if !yield(n) {
					return
				}
			}
		})
		s.Code = nameCodes[names[r.IntN(len(names))]]
	case 1:
		s.Code = tea.KeyF1 + rune(r.IntN(63))
	case 2:
		for {
			c := rune(0x21 + r.IntN(0x2000))
			if unicode.IsPrint(c) && !unicode.IsUpper(c) && unicode.ToLower(c) == c {
				if _, named := codeNames[c]; !named {
					s.Code = c
					break
				}
			}
		}
	default:
		codes := slices.Sorted(func(yield func(string) bool) {
			for n := range scanCodes {
				if !yield(n) {
					return
				}
			}
		})
		s.Code, s.Scan = scanCodes[codes[r.IntN(len(codes))]], true
	}
	s.Event = Event(r.IntN(3))
	return s
}

// TestRoundTrips: a canonical stroke or sequence reads back from its
// canonical and its VS Code spelling, events included (amendment A5).
func TestRoundTrips(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 13))
	for range 5000 {
		s := randomStroke(r)
		if got, err := ParseStroke(s.String()); err != nil || got != s {
			t.Fatalf("ParseStroke(%q) = %+v, %v; want %+v", s.String(), got, err, s)
		}
		if got, err := ParseStroke(s.VSCode()); err != nil || got != s {
			t.Fatalf("ParseStroke(%q) = %+v, %v; want %+v", s.VSCode(), got, err, s)
		}
		q := Sequence{s}
		for range r.IntN(MaxStrokes) {
			q = append(q, randomStroke(r))
		}
		for _, text := range []string{q.String(), q.VSCode()} {
			if got, err := ParseSequence(text, nil); err != nil || !slices.Equal(got, q) {
				t.Fatalf("ParseSequence(%q) = %v, %v; want %v", text, got, err, q)
			}
		}
	}
}

// TestNames: every named key's name is Bubble Tea's own, and a name reads
// as the main key, not the keypad's.
func TestNames(t *testing.T) {
	for _, c := range namedKeys {
		if n := codeNames[c]; n == "" || n == string(c) {
			t.Errorf("key %U has no name of Bubble Tea's: %q", c, n)
		}
	}
	for name, want := range map[string]rune{"enter": tea.KeyEnter, "up": tea.KeyUp, "esc": tea.KeyEscape,
		"space": tea.KeySpace, "pgdown": tea.KeyPgDown, "plus": tea.KeyKpPlus, "f63": tea.KeyF63} {
		if got := nameCodes[name]; got != want {
			t.Errorf("%q reads as %U, want %U", name, got, want)
		}
	}
}

func mustStroke(t *testing.T, s string) Stroke {
	t.Helper()
	st, err := ParseStroke(s)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// TestNormalization: the spellings §2 and A5 read as one stroke, and the
// inputs they refuse, each named.
func TestNormalization(t *testing.T) {
	for in, want := range map[string]string{
		"A": "shift+a", "Ctrl+Shift+A": "ctrl+shift+a", "shift+ctrl+a": "ctrl+shift+a",
		"ctrl+shift+a": "ctrl+shift+a", "Ctrl+A": "ctrl+shift+a", "escape": "esc", "pageup": "pgup",
		"pagedown": "pgdown", "cmd+a": "super+a", "win+a": "super+a", "option+a": "alt+a", "ENTER": "enter",
		"[keya]": "[KeyA]", "ctrl+[KeyA]": "ctrl+[KeyA]", "?": "?", "ctrl++": "ctrl++", "+": "+",
		"a:release": "a:release", "ctrl+x:REPEAT": "ctrl+x:repeat", "::release": "::release", "shift+tab": "shift+tab",
	} {
		if got := mustStroke(t, in).String(); got != want {
			t.Errorf("%q reads as %q, want %q", in, got, want)
		}
	}
	for _, c := range []struct{ in, part string }{
		{"capslock+a", "capslock"}, {"numlock+a", "numlock"}, {"ScrollLock+a", "ScrollLock"},
		{"", "empty"}, {"ctrl+", "no key"}, {"ctrl+ctrl+a", "twice"}, {"ctrl+foo", `"foo"`},
		{"[KeyZz]", "[KeyZz]"}, {":release", `":release"`},
	} {
		_, err := ParseStroke(c.in)
		if !errors.Is(err, ErrNotation) || !strings.Contains(err.Error(), c.part) {
			t.Errorf("ParseStroke(%q) = %v; want ErrNotation naming %q", c.in, err, c.part)
		}
	}
	leader := Sequence{mustStroke(t, "ctrl+x")}
	for _, c := range []struct {
		in     string
		leader Sequence
		part   string
	}{
		{"a b c d e", nil, "5 strokes"}, {"a  b", nil, "empty stroke"}, {" a", nil, "empty stroke"},
		{"<leader> a", nil, "no leader"}, {"<leader> <leader> <leader> <leader> a", leader, "5 strokes"},
		{"a ctrl+foo", nil, `"foo"`},
	} {
		_, err := ParseSequence(c.in, c.leader)
		if !errors.Is(err, ErrNotation) || !strings.Contains(err.Error(), c.part) {
			t.Errorf("ParseSequence(%q) = %v; want ErrNotation naming %q", c.in, err, c.part)
		}
	}
	q, err := ParseSequence("<Leader> q", leader)
	if err != nil || q.String() != "ctrl+x q" {
		t.Errorf("<leader> q = %v, %v; want ctrl+x q", q, err)
	}
}

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
// strokes it should, with the lock modifiers ignored.
func TestMatches(t *testing.T) {
	for _, c := range keyCases {
		for text, want := range c.strokes {
			if got := mustStroke(t, text).Matches(c.key); got != want[0] {
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
	for _, c := range keyCases {
		msg := tea.KeyPressMsg(c.key)
		for text, want := range c.strokes {
			s := mustStroke(t, text)
			b := key.NewBinding(key.WithKeys(s.BubblesKeys()...))
			got := key.Matches(msg, b)
			if got != want[1] {
				t.Errorf("%s: key.Matches(%q, %q) = %v, want %v", c.name, msg.String(), s.BubblesKeys(), got, want[1])
			}
			locked := c.key.Mod&locks != 0
			symbol := strings.HasPrefix(text, "shift+") && !unicode.IsLetter(s.Code) && s.printable()
			if !locked && !symbol && s.Event == Press && !s.Scan && want[0] != want[1] {
				t.Errorf("%s: %q: Matches and key.Matches differ on a key they should agree on", c.name, text)
			}
		}
	}
}

// TestEventText: the event's tokens.
func TestEventText(t *testing.T) {
	for e, want := range map[Event]string{Press: "press", Release: "release", Repeat: "repeat"} {
		if e.String() != want {
			t.Errorf("%d = %q", e, e.String())
		}
	}
}
