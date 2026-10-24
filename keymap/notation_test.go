package keymap

import (
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
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
		s.Code = termkey.KeyF1 + rune(r.IntN(63))
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
	for name, want := range map[string]rune{"enter": termkey.KeyEnter, "up": termkey.KeyUp, "esc": termkey.KeyEscape,
		"space": termkey.KeySpace, "pgdown": termkey.KeyPgDown, "plus": termkey.KeyKpPlus, "f63": termkey.KeyF63} {
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
		{"<leader> <leader> <leader> <leader> a", leader, "5 strokes"},
		{"<leader> a b c d", nil, "5 strokes"}, {"a ctrl+foo", nil, `"foo"`},
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

// TestLeaderPlaceholder: with no leader, <leader> stays a placeholder that
// both spellings write back, for New to expand, and that matches no key
// (amendment A6).
func TestLeaderPlaceholder(t *testing.T) {
	q, err := ParseSequence("<LEADER> f", nil)
	if err != nil || len(q) != 2 || q[0].Code != keyLeader {
		t.Fatalf(`ParseSequence("<LEADER> f", nil) = %v, %v; want the placeholder and f`, q, err)
	}
	for _, text := range []string{q.String(), q.VSCode()} {
		if text != "<leader> f" {
			t.Errorf("the placeholder writes %q, want %q", text, "<leader> f")
		}
		if got, err := ParseSequence(text, nil); err != nil || !slices.Equal(got, q) {
			t.Errorf("ParseSequence(%q) = %v, %v; want %v", text, got, err, q)
		}
	}
	for _, k := range []termkey.KeyEvent{{}, {Code: keyLeader}, {Code: 'f'}} {
		if q[0].Matches(k) {
			t.Errorf("the placeholder matches %+v", k)
		}
	}
	if _, err := ParseStroke("<leader>"); !errors.Is(err, ErrNotation) {
		t.Errorf(`ParseStroke("<leader>") = %v; want ErrNotation: the placeholder is a sequence's`, err)
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
