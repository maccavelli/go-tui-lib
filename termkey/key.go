package termkey

import (
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "termkey"

// Mod is a set of modifier keys. Each bit has ultraviolet's value, which
// Bubble Tea's tea.KeyMod takes. The Kitty keyboard protocol numbers super
// and meta the other way round; the decoder maps them.
type Mod uint16

// The modifiers, and the lock keys' states.
const (
	ModShift Mod = 1 << iota
	ModAlt
	ModCtrl
	ModMeta
	ModHyper
	ModSuper // the Windows or Command key
	ModCapsLock
	ModNumLock
	ModScrollLock
)

var modNames = enum.Bits[Mod]{
	Pkg: pkgName, Type: "Mod", None: "none",
	Tokens: []string{"shift", "alt", "ctrl", "meta", "hyper", "super", "capslock", "numlock", "scrolllock"},
}

// String is the set's names joined with "|", or "none". Bits with no name
// come last, in hex.
func (m Mod) String() string { return modNames.String(m) }

// MarshalText is the set's names joined with "|", or "none". A bit with no
// name is an error.
func (m Mod) MarshalText() ([]byte, error) { return modNames.Marshal(m) }

// UnmarshalText reads what MarshalText wrote, the names in any order.
func (m *Mod) UnmarshalText(b []byte) error { return modNames.Unmarshal(b, m) }

// Contains reports whether m holds every modifier of mods.
func (m Mod) Contains(mods Mod) bool { return m&mods == mods }

// KeyEvent is one key a terminal sent: its code, its modifiers, the text it
// types, and, with the Kitty keyboard protocol, its shifted and base codes
// and whether it repeats or is released. It is tea.Key's fields, and
// Release, which Bubble Tea gives as a message type of its own.
type KeyEvent struct {
	Code        rune   // a key code (KeyEnter, KeyF1) or the rune of a printable key, lower case
	Mod         Mod    // the modifiers held
	Text        string // what the key types; "" for a key that types nothing
	ShiftedCode rune   // the shifted key's rune, "A" for shift+a, when the terminal says
	BaseCode    rune   // the key on a US PC-101 layout, when the terminal says
	IsRepeat    bool   // the key is held down, and repeats
	Release     bool   // the key was released
}

// String is the text the key types, when it types any but a space, and
// otherwise Keystroke, as tea.Key's String is: "?" rather than "shift+/".
func (k KeyEvent) String() string {
	if k.Text != "" && k.Text != " " {
		return k.Text
	}
	return k.Keystroke()
}

// Keystroke is the key as a binding names it, as tea.Key's Keystroke is:
// the modifiers, always in the order ctrl, alt, shift, meta, hyper and
// super, each but the key's own, then the key's name, or its base code's
// rune, or its own. A space is "space".
func (k KeyEvent) Keystroke() string {
	var sb strings.Builder
	for _, m := range []struct {
		mod         Mod
		name        string
		left, right rune
	}{
		{ModCtrl, "ctrl+", KeyLeftCtrl, KeyRightCtrl},
		{ModAlt, "alt+", KeyLeftAlt, KeyRightAlt},
		{ModShift, "shift+", KeyLeftShift, KeyRightShift},
		{ModMeta, "meta+", KeyLeftMeta, KeyRightMeta},
		{ModHyper, "hyper+", KeyLeftHyper, KeyRightHyper},
		{ModSuper, "super+", KeyLeftSuper, KeyRightSuper},
	} {
		if k.Mod.Contains(m.mod) && k.Code != m.left && k.Code != m.right {
			sb.WriteString(m.name)
		}
	}
	if name, ok := keyNames[k.Code]; ok {
		sb.WriteString(name)
		return sb.String()
	}
	code := k.Code
	if k.BaseCode != 0 {
		code = k.BaseCode // the key on a US layout
	}
	switch code {
	case KeySpace:
		sb.WriteString("space")
	case KeyExtended:
		sb.WriteString(k.Text)
	default:
		sb.WriteRune(code)
	}
	return sb.String()
}
