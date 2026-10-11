// Package keymap binds keys to command IDs through a context-aware keymap,
// with chords, a leader key and VS Code-format user keymaps
// (docs/decisions/0007-MADR-keymap-engine.md).
//
// A Stroke is one key, as a binding names it, and a Sequence is one stroke,
// a chord or a leader sequence. The canonical notation is Bubble Tea's own
// Keystroke() form, "ctrl+alt+shift+a"; VS Code's spelling parses too, and
// each spelling round-trips. A release or repeat stroke ends in ":release"
// or ":repeat" (amendment A5).
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package keymap

import (
	"cmp"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/internal/enum"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "keymap"

// Event is when a stroke fires: when its key is pressed, released, or
// repeated while it is held.
type Event uint8

// The events. Press is the zero value.
const (
	Press Event = iota
	Release
	Repeat
)

var eventNames = enum.Names[Event]{
	Pkg: pkgName, Type: "Event",
	Tokens: []string{"press", "release", "repeat"},
}

// String is the event's token.
func (e Event) String() string { return eventNames.String(e) }

// MarshalText is the event's token. An event with no token is an error.
func (e Event) MarshalText() ([]byte, error) { return eventNames.Marshal(e) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (e *Event) UnmarshalText(b []byte) error { return eventNames.Unmarshal(b, e) }

// Stroke is one key, as a binding names it.
type Stroke struct {
	Mod   tea.KeyMod // ctrl, alt, shift, meta, hyper, super; never a lock key
	Code  rune       // a tea key constant (tea.KeyEnter), or a lower-case rune
	Scan  bool       // match the US-layout position (BaseCode), as VS Code's [KeyA]
	Event Event      // Press (the default), Release, or Repeat
}

// Sequence is one stroke, a chord, or a leader sequence; at most MaxStrokes.
type Sequence []Stroke

// MaxStrokes is the most strokes a Sequence holds.
const MaxStrokes = 4

// ErrNotation is wrapped by every error ParseStroke and ParseSequence
// return; the error's text names the input and its bad part.
var ErrNotation = errors.New("keymap: bad key notation")

// locks are the lock modifiers: state, not a chord.
const locks = tea.ModCapsLock | tea.ModNumLock | tea.ModScrollLock

// modifiers, in Keystroke() order, with their canonical names, and VS
// Code's where it differs.
var modifiers = []struct {
	mod           tea.KeyMod
	canon, vscode string
}{
	{tea.ModCtrl, "ctrl", ""},
	{tea.ModAlt, "alt", ""},
	{tea.ModShift, "shift", ""},
	{tea.ModMeta, "meta", ""},
	{tea.ModHyper, "hyper", ""},
	{tea.ModSuper, "super", "cmd"},
}

// modifierNames are the names a modifier is read from, lower case: each
// canonical and VS Code name, and the aliases "option" and "win".
var modifierNames = func() map[string]tea.KeyMod {
	names := map[string]tea.KeyMod{"option": tea.ModAlt, "win": tea.ModSuper}
	for _, m := range modifiers {
		names[m.canon] = m.mod
		if m.vscode != "" {
			names[m.vscode] = m.mod
		}
	}
	return names
}()

// lockNames are the lock modifiers' names, refused as modifiers.
var lockNames = map[string]bool{"capslock": true, "numlock": true, "scrolllock": true}

// namedKeys are the keys Bubble Tea names, the main keys before the keypad's,
// so that a name parses to the main key. Each name is Bubble Tea's own,
// from Keystroke(): this package copies no name.
var namedKeys = []rune{
	tea.KeyBackspace, tea.KeyTab, tea.KeyEnter, tea.KeyEscape, tea.KeySpace,
	tea.KeyUp, tea.KeyDown, tea.KeyRight, tea.KeyLeft, tea.KeyBegin, tea.KeyFind,
	tea.KeyInsert, tea.KeyDelete, tea.KeySelect, tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd,
	tea.KeyF1, tea.KeyF2, tea.KeyF3, tea.KeyF4, tea.KeyF5, tea.KeyF6, tea.KeyF7, tea.KeyF8, tea.KeyF9,
	tea.KeyF10, tea.KeyF11, tea.KeyF12, tea.KeyF13, tea.KeyF14, tea.KeyF15, tea.KeyF16, tea.KeyF17,
	tea.KeyF18, tea.KeyF19, tea.KeyF20, tea.KeyF21, tea.KeyF22, tea.KeyF23, tea.KeyF24, tea.KeyF25,
	tea.KeyF26, tea.KeyF27, tea.KeyF28, tea.KeyF29, tea.KeyF30, tea.KeyF31, tea.KeyF32, tea.KeyF33,
	tea.KeyF34, tea.KeyF35, tea.KeyF36, tea.KeyF37, tea.KeyF38, tea.KeyF39, tea.KeyF40, tea.KeyF41,
	tea.KeyF42, tea.KeyF43, tea.KeyF44, tea.KeyF45, tea.KeyF46, tea.KeyF47, tea.KeyF48, tea.KeyF49,
	tea.KeyF50, tea.KeyF51, tea.KeyF52, tea.KeyF53, tea.KeyF54, tea.KeyF55, tea.KeyF56, tea.KeyF57,
	tea.KeyF58, tea.KeyF59, tea.KeyF60, tea.KeyF61, tea.KeyF62, tea.KeyF63,
	tea.KeyCapsLock, tea.KeyScrollLock, tea.KeyNumLock, tea.KeyPrintScreen, tea.KeyPause, tea.KeyMenu,
	tea.KeyMediaPlay, tea.KeyMediaPause, tea.KeyMediaPlayPause, tea.KeyMediaReverse, tea.KeyMediaStop,
	tea.KeyMediaFastForward, tea.KeyMediaRewind, tea.KeyMediaNext, tea.KeyMediaPrev, tea.KeyMediaRecord,
	tea.KeyLowerVol, tea.KeyRaiseVol, tea.KeyMute,
	tea.KeyLeftShift, tea.KeyLeftAlt, tea.KeyLeftCtrl, tea.KeyLeftSuper, tea.KeyLeftHyper, tea.KeyLeftMeta,
	tea.KeyRightShift, tea.KeyRightAlt, tea.KeyRightCtrl, tea.KeyRightSuper, tea.KeyRightHyper,
	tea.KeyRightMeta, tea.KeyIsoLevel3Shift, tea.KeyIsoLevel5Shift,
	tea.KeyKpEnter, tea.KeyKpEqual, tea.KeyKpMultiply, tea.KeyKpPlus, tea.KeyKpComma, tea.KeyKpMinus,
	tea.KeyKpDecimal, tea.KeyKpDivide, tea.KeyKp0, tea.KeyKp1, tea.KeyKp2, tea.KeyKp3, tea.KeyKp4,
	tea.KeyKp5, tea.KeyKp6, tea.KeyKp7, tea.KeyKp8, tea.KeyKp9, tea.KeyKpSep, tea.KeyKpUp, tea.KeyKpDown,
	tea.KeyKpLeft, tea.KeyKpRight, tea.KeyKpPgUp, tea.KeyKpPgDown, tea.KeyKpHome, tea.KeyKpEnd,
	tea.KeyKpInsert, tea.KeyKpDelete, tea.KeyKpBegin,
}

// codeNames is each named key's name, and nameCodes each name's first
// code: the main key's.
var codeNames, nameCodes = keyNames()

func keyNames() (map[rune]string, map[string]rune) {
	names, codes := map[rune]string{}, map[string]rune{}
	for _, c := range namedKeys {
		n := tea.Key{Code: c}.Keystroke()
		names[c] = n
		if _, ok := codes[n]; !ok && utf8.RuneCountInString(n) > 1 {
			codes[n] = c
		}
	}
	return names, codes
}

// vscodeNames are the key names VS Code spells another way; the reverse
// are read as aliases.
var vscodeNames = map[string]string{"esc": "escape", "pgup": "pageup", "pgdown": "pagedown"}

// scanCodes are VS Code's scan codes, each the US-layout key it names.
var scanCodes = scanTable()

func scanTable() map[string]rune {
	t := map[string]rune{
		"Minus": '-', "Equal": '=', "BracketLeft": '[', "BracketRight": ']', "Backslash": '\\',
		"Semicolon": ';', "Quote": '\'', "Backquote": '`', "Comma": ',', "Period": '.', "Slash": '/',
	}
	for r := 'a'; r <= 'z'; r++ {
		t["Key"+string(unicode.ToUpper(r))] = r
	}
	for r := '0'; r <= '9'; r++ {
		t["Digit"+string(r)] = r
	}
	return t
}

// scanNames is each scan code's name, by the key it names.
var scanNames = func() map[rune]string {
	n := map[rune]string{}
	for name, r := range scanCodes {
		n[r] = name
	}
	return n
}()

// scanLower is each scan code by its lower-case name, for input.
var scanLower = func() map[string]rune {
	l := map[string]rune{}
	for name, r := range scanCodes {
		l[strings.ToLower(name)] = r
	}
	return l
}()

func bad(input, format string, a ...any) error {
	return fmt.Errorf("%w %q: %s", ErrNotation, input, fmt.Sprintf(format, a...))
}

// ParseStroke reads one stroke, in the canonical or the VS Code spelling:
// modifiers and a key joined by "+", and an optional ":release" or
// ":repeat". Modifier and key names are case-insensitive, and an upper-case
// letter means shift and the letter (amendment A5). A lock modifier, an
// unknown name and an empty stroke are errors wrapping ErrNotation, naming
// the bad part.
func ParseStroke(s string) (Stroke, error) {
	var st Stroke
	rest := s
	lower := strings.ToLower(rest)
	for _, e := range []Event{Release, Repeat} {
		suffix := ":" + e.String()
		if strings.HasSuffix(lower, suffix) && len(rest) > len(suffix) {
			st.Event = e
			rest = rest[:len(rest)-len(suffix)]
			break
		}
	}
	if rest == "" {
		return Stroke{}, bad(s, "an empty stroke")
	}
	for {
		i := strings.Index(rest, "+")
		if i <= 0 {
			break // no modifier left; a key may be "+" itself
		}
		word := strings.ToLower(rest[:i])
		if lockNames[word] {
			return Stroke{}, bad(s, "%q is a lock, not a modifier", rest[:i])
		}
		m, ok := modifierNames[word]
		if !ok {
			break
		}
		if st.Mod.Contains(m) {
			return Stroke{}, bad(s, "the modifier %q twice", rest[:i])
		}
		st.Mod |= m
		rest = rest[i+1:]
	}
	if rest == "" {
		return Stroke{}, bad(s, "no key after the modifiers")
	}
	switch {
	case len(rest) > 2 && rest[0] == '[' && rest[len(rest)-1] == ']':
		r, ok := scanLower[strings.ToLower(rest[1:len(rest)-1])]
		if !ok {
			return Stroke{}, bad(s, "the unknown scan code %q", rest)
		}
		st.Code, st.Scan = r, true
	case utf8.RuneCountInString(rest) == 1:
		r, _ := utf8.DecodeRuneInString(rest)
		if !unicode.IsPrint(r) || r == ' ' {
			return Stroke{}, bad(s, "the key %q is not printable", rest)
		}
		if lo := unicode.ToLower(r); lo != r && unicode.IsUpper(r) {
			r = lo
			st.Mod |= tea.ModShift
		}
		st.Code = r
	default:
		name := strings.ToLower(rest)
		for canon, vs := range vscodeNames {
			if name == vs {
				name = canon
			}
		}
		c, ok := nameCodes[name]
		if !ok {
			return Stroke{}, bad(s, "the unknown key %q", rest)
		}
		st.Code = c
	}
	return st, nil
}

// leaderName is the token that stands for the leader sequence.
const leaderName = "<leader>"

// keyLeader is the code of the placeholder stroke that stands for the
// leader until New expands it (amendment A6): no key has it.
const keyLeader rune = -1

// ParseSequence reads strokes separated by one space, expanding <leader> to
// leader. With no leader, <leader> stays a placeholder stroke, which
// prints "<leader>" and matches no key; New expands it with
// Settings.Leader. An empty stroke and more than MaxStrokes strokes are
// errors wrapping ErrNotation.
func ParseSequence(s string, leader Sequence) (Sequence, error) {
	var q Sequence
	for part := range strings.SplitSeq(s, " ") {
		switch {
		case part == "":
			return nil, bad(s, "an empty stroke; strokes are separated by one space")
		case strings.EqualFold(part, leaderName):
			if len(leader) == 0 {
				q = append(q, Stroke{Code: keyLeader})
				continue
			}
			q = append(q, leader...)
		default:
			st, err := ParseStroke(part)
			if err != nil {
				return nil, err
			}
			q = append(q, st)
		}
	}
	if len(q) > MaxStrokes {
		return nil, bad(s, "%d strokes, more than %d", len(q), MaxStrokes)
	}
	return q, nil
}

// keyName is a key's name as Bubble Tea prints it: its name, or the rune;
// and the leader placeholder's, "<leader>".
func keyName(c rune) string {
	if n, ok := codeNames[c]; ok {
		return n
	}
	if c == keyLeader {
		return leaderName
	}
	return string(c)
}

// write is the stroke with each modifier spelled by name and the key by
// keyText.
func (s Stroke) write(name func(canon, vscode string) string, keyText string) string {
	var b strings.Builder
	for _, m := range modifiers {
		if s.Mod.Contains(m.mod) {
			b.WriteString(name(m.canon, m.vscode))
			b.WriteByte('+')
		}
	}
	b.WriteString(keyText)
	if s.Event != Press {
		b.WriteString(":" + s.Event.String())
	}
	return b.String()
}

// text is the stroke's key: its scan code, name or rune.
func (s Stroke) text() string {
	if s.Scan {
		return "[" + scanNames[s.Code] + "]"
	}
	return keyName(s.Code)
}

// String is the canonical spelling: "ctrl+alt+shift+a".
func (s Stroke) String() string {
	return s.write(func(canon, _ string) string { return canon }, s.text())
}

// VSCode is VS Code's spelling: "cmd" for super, "escape", "pageup" and
// "pagedown".
func (s Stroke) VSCode() string {
	key := s.text()
	if vs, ok := vscodeNames[key]; ok && !s.Scan {
		key = vs
	}
	return s.write(func(canon, vscode string) string { return cmp.Or(vscode, canon) }, key)
}

// String is the strokes in the canonical spelling, joined by one space.
func (q Sequence) String() string { return q.join(Stroke.String) }

// VSCode is the strokes in VS Code's spelling, joined by one space.
// ParseSequence reads it back unchanged.
func (q Sequence) VSCode() string { return q.join(Stroke.VSCode) }

func (q Sequence) join(f func(Stroke) string) string {
	parts := make([]string, len(q))
	for i, s := range q {
		parts[i] = f(s)
	}
	return strings.Join(parts, " ")
}

// printable reports whether the stroke's key is a printable rune, with no
// name.
func (s Stroke) printable() bool {
	_, named := codeNames[s.Code]
	return !s.Scan && !named && unicode.IsPrint(s.Code)
}

// Matches reports whether k is this stroke's key. It ignores the message's
// lock modifiers, and a letter caps lock made upper case. A stroke matches
// when:
//   - the modifiers are equal, and Bubble Tea prints both keys with one
//     name, so the keypad's Enter matches enter (amendment A5);
//   - for a printable stroke without shift, the message's Text is the
//     stroke's rune and its modifiers, less shift, are equal;
//   - for a Scan stroke, the message's BaseCode, or Code when there is no
//     BaseCode, is the stroke's key.
//
// A Repeat stroke matches only a repeated key. Whether the message is a
// press or a release is the message's type, which Matches does not see.
func (s Stroke) Matches(k tea.Key) bool {
	if s.Code == keyLeader || s.Event == Repeat && !k.IsRepeat {
		return false
	}
	mod := k.Mod &^ locks
	code := k.Code
	if k.Mod.Contains(tea.ModCapsLock) && unicode.IsLetter(code) {
		code = unicode.ToLower(code)
	}
	if s.Scan {
		base := k.BaseCode
		if base == 0 {
			base = code
		}
		return mod == s.Mod && unicode.ToLower(base) == s.Code
	}
	if mod == s.Mod && keyName(code) == keyName(s.Code) {
		return true
	}
	return s.printable() && !s.Mod.Contains(tea.ModShift) &&
		k.Text == string(s.Code) && mod&^tea.ModShift == s.Mod
}

// BubblesKeys is every k.String() this stroke can produce, for a
// key.Binding: its Keystroke() form, and for a printable stroke with no
// modifier but shift, the text the key types. Three keys are beyond it,
// since bubbles compares a key's String(), its typed text: a caps-lock
// letter, which bubbles cannot tell from shift; a shifted symbol written as
// shift and its key, whose text is the layout's (write the symbol, "?");
// and a Scan stroke, which names a position, not the text typed.
func (s Stroke) BubblesKeys() []string {
	key := keyName(s.Code)
	if s.Scan {
		key = string(s.Code)
	}
	plain := Stroke{Mod: s.Mod}
	keys := []string{plain.write(func(canon, _ string) string { return canon }, key)}
	if s.printable() && s.Mod&^tea.ModShift == 0 {
		text := string(s.Code)
		if s.Mod.Contains(tea.ModShift) {
			if !unicode.IsLetter(s.Code) {
				return keys
			}
			text = string(unicode.ToUpper(s.Code))
		}
		if text != keys[0] {
			keys = append(keys, text)
		}
	}
	return keys
}
