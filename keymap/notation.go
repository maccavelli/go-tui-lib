// Package keymap binds keys to command IDs through a context-aware keymap,
// with chords, a leader key and VS Code-format user keymaps
// (docs/decisions/0007-MADR-keymap-engine.md).
//
// A Stroke is one key, as a binding names it, and a Sequence is one stroke,
// a chord or a leader sequence. The canonical notation is termkey's
// Keystroke() form, which is Bubble Tea's, "ctrl+alt+shift+a"; VS Code's
// spelling parses too, and each spelling round-trips. A release or repeat
// stroke ends in ":release" or ":repeat" (amendment A5).
//
// The package links no Bubble Tea: a Dispatcher takes termkey's keys, and
// says how long a pending sequence waits. Its Bubble Tea forms, the
// dispatcher's Update, help and key.Binding, are in keymap/keymaptea
// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md D7).
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

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "keymap"

// nameSuper and namePgDown are canonical names that several of the
// spelling tables give: the modifiers', VS Code's, and the aliases'.
const (
	nameSuper  = "super"
	namePgDown = "pgdown"
)

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
	Mod   termkey.Mod // ctrl, alt, shift, meta, hyper, super; never a lock key
	Code  rune        // a tea key constant (termkey.KeyEnter), or a lower-case rune
	Scan  bool        // match the US-layout position (BaseCode), as VS Code's [KeyA]
	Event Event       // Press (the default), Release, or Repeat
}

// Sequence is one stroke, a chord, or a leader sequence; at most MaxStrokes.
type Sequence []Stroke

// MaxStrokes is the most strokes a Sequence holds.
const MaxStrokes = 4

// ErrNotation is wrapped by every error ParseStroke and ParseSequence
// return; the error's text names the input and its bad part.
var ErrNotation = errors.New("keymap: bad key notation")

// locks are the lock modifiers: state, not a chord.
const locks = termkey.ModCapsLock | termkey.ModNumLock | termkey.ModScrollLock

// modifiers, in Keystroke() order, with their canonical names, and VS
// Code's where it differs.
var modifiers = []struct {
	mod           termkey.Mod
	canon, vscode string
}{
	{termkey.ModCtrl, "ctrl", ""},
	{termkey.ModAlt, "alt", ""},
	{termkey.ModShift, "shift", ""},
	{termkey.ModMeta, "meta", ""},
	{termkey.ModHyper, "hyper", ""},
	{termkey.ModSuper, nameSuper, "cmd"},
}

// modifierNames are the names a modifier is read from, lower case: each
// canonical name, and each of modAliases, VS Code's "cmd" among them.
var modifierNames = func() map[string]termkey.Mod {
	names := map[string]termkey.Mod{}
	for _, m := range modifiers {
		names[m.canon] = m.mod
	}
	for alias, canon := range modAliases {
		names[alias] = names[canon]
	}
	return names
}()

// lockNames are the lock modifiers' names, refused as modifiers.
var lockNames = map[string]bool{"capslock": true, "numlock": true, "scrolllock": true}

// namedKeys are the keys termkey names, the main keys before the keypad's,
// so that a name parses to the main key. Each name is termkey's own, from
// Keystroke(), which is Bubble Tea's: this package copies no name.
var namedKeys = []rune{
	termkey.KeyBackspace, termkey.KeyTab, termkey.KeyEnter, termkey.KeyEscape, termkey.KeySpace,
	termkey.KeyUp, termkey.KeyDown, termkey.KeyRight, termkey.KeyLeft, termkey.KeyBegin, termkey.KeyFind,
	termkey.KeyInsert, termkey.KeyDelete, termkey.KeySelect, termkey.KeyPgUp, termkey.KeyPgDown, termkey.KeyHome, termkey.KeyEnd,
	termkey.KeyF1, termkey.KeyF2, termkey.KeyF3, termkey.KeyF4, termkey.KeyF5, termkey.KeyF6, termkey.KeyF7, termkey.KeyF8, termkey.KeyF9,
	termkey.KeyF10, termkey.KeyF11, termkey.KeyF12, termkey.KeyF13, termkey.KeyF14, termkey.KeyF15, termkey.KeyF16, termkey.KeyF17,
	termkey.KeyF18, termkey.KeyF19, termkey.KeyF20, termkey.KeyF21, termkey.KeyF22, termkey.KeyF23, termkey.KeyF24, termkey.KeyF25,
	termkey.KeyF26, termkey.KeyF27, termkey.KeyF28, termkey.KeyF29, termkey.KeyF30, termkey.KeyF31, termkey.KeyF32, termkey.KeyF33,
	termkey.KeyF34, termkey.KeyF35, termkey.KeyF36, termkey.KeyF37, termkey.KeyF38, termkey.KeyF39, termkey.KeyF40, termkey.KeyF41,
	termkey.KeyF42, termkey.KeyF43, termkey.KeyF44, termkey.KeyF45, termkey.KeyF46, termkey.KeyF47, termkey.KeyF48, termkey.KeyF49,
	termkey.KeyF50, termkey.KeyF51, termkey.KeyF52, termkey.KeyF53, termkey.KeyF54, termkey.KeyF55, termkey.KeyF56, termkey.KeyF57,
	termkey.KeyF58, termkey.KeyF59, termkey.KeyF60, termkey.KeyF61, termkey.KeyF62, termkey.KeyF63,
	termkey.KeyCapsLock, termkey.KeyScrollLock, termkey.KeyNumLock, termkey.KeyPrintScreen, termkey.KeyPause, termkey.KeyMenu,
	termkey.KeyMediaPlay, termkey.KeyMediaPause, termkey.KeyMediaPlayPause, termkey.KeyMediaReverse, termkey.KeyMediaStop,
	termkey.KeyMediaFastForward, termkey.KeyMediaRewind, termkey.KeyMediaNext, termkey.KeyMediaPrev, termkey.KeyMediaRecord,
	termkey.KeyLowerVol, termkey.KeyRaiseVol, termkey.KeyMute,
	termkey.KeyLeftShift, termkey.KeyLeftAlt, termkey.KeyLeftCtrl, termkey.KeyLeftSuper, termkey.KeyLeftHyper, termkey.KeyLeftMeta,
	termkey.KeyRightShift, termkey.KeyRightAlt, termkey.KeyRightCtrl, termkey.KeyRightSuper, termkey.KeyRightHyper,
	termkey.KeyRightMeta, termkey.KeyIsoLevel3Shift, termkey.KeyIsoLevel5Shift,
	termkey.KeyKpEnter, termkey.KeyKpEqual, termkey.KeyKpMultiply, termkey.KeyKpPlus, termkey.KeyKpComma, termkey.KeyKpMinus,
	termkey.KeyKpDecimal, termkey.KeyKpDivide, termkey.KeyKp0, termkey.KeyKp1, termkey.KeyKp2, termkey.KeyKp3, termkey.KeyKp4,
	termkey.KeyKp5, termkey.KeyKp6, termkey.KeyKp7, termkey.KeyKp8, termkey.KeyKp9, termkey.KeyKpSep, termkey.KeyKpUp, termkey.KeyKpDown,
	termkey.KeyKpLeft, termkey.KeyKpRight, termkey.KeyKpPgUp, termkey.KeyKpPgDown, termkey.KeyKpHome, termkey.KeyKpEnd,
	termkey.KeyKpInsert, termkey.KeyKpDelete, termkey.KeyKpBegin,
}

// codeNames is each named key's name, and nameCodes each name's first
// code: the main key's.
var codeNames, nameCodes = keyNames()

func keyNames() (map[rune]string, map[string]rune) {
	names, codes := map[rune]string{}, map[string]rune{}
	for _, c := range namedKeys {
		n := termkey.KeyEvent{Code: c}.Keystroke()
		names[c] = n
		if _, ok := codes[n]; !ok && utf8.RuneCountInString(n) > 1 {
			codes[n] = c
		}
	}
	return names, codes
}

// vscodeNames are the key names VS Code spells another way. ParseStroke
// reads VS Code's names among keyAliases.
var vscodeNames = map[string]string{"esc": "escape", "pgup": "pageup", namePgDown: "pagedown"}

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
			st.Mod |= termkey.ModShift
		}
		st.Code = r
	default:
		name := strings.ToLower(rest)
		if canon, ok := keyAliases[name]; ok {
			name = canon
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

// keyName is a key's name as termkey prints it: its name, or the rune;
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

// foldCode is k's code as Matches compares it: a letter caps lock made
// upper case is lowered.
func foldCode(k termkey.KeyEvent) rune {
	if k.Mod.Contains(termkey.ModCapsLock) && unicode.IsLetter(k.Code) {
		return unicode.ToLower(k.Code)
	}
	return k.Code
}

// matchName is the name Matches compares for k, for the dispatcher's skip
// (docs/decisions/0023-MADR-line-mode-components.md A12 item 4.9).
func matchName(k termkey.KeyEvent) string { return keyName(foldCode(k)) }

// Matches reports whether k is this stroke's key. It ignores the key's
// lock modifiers, and a letter caps lock made upper case. A stroke matches
// when:
//   - the modifiers are equal, and termkey prints both keys with one
//     name, so the keypad's Enter matches enter (amendment A5);
//   - for a printable stroke without shift, the key's Text is the
//     stroke's rune and its modifiers, less shift, are equal;
//   - for a Scan stroke, the key's BaseCode, or Code when there is no
//     BaseCode, is the stroke's key.
//
// A Repeat stroke matches only a repeated key. Matches does not look at
// whether the key is a press or a release; the Dispatcher does.
func (s Stroke) Matches(k termkey.KeyEvent) bool {
	if s.Code == keyLeader || s.Event == Repeat && !k.IsRepeat {
		return false
	}
	mod := k.Mod &^ locks
	code := foldCode(k)
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
	return mod&^termkey.ModShift == s.Mod && !s.Mod.Contains(termkey.ModShift) &&
		k.Text == string(s.Code) && s.printable()
}
