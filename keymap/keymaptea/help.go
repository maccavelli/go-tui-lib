package keymaptea

import (
	"slices"
	"unicode"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// title is id's title from d, or the ID itself when d is nil or has none.
func title(id command.ID, d keymap.Describer) string {
	if d != nil {
		if t, ok := d.Describe(id); ok {
			return t
		}
	}
	return string(id)
}

// KeyBinding is a bubbles key.Binding for id along a path (0007-MADR §7,
// amendment A7). Its keys are every BubblesKeys spelling of each
// one-stroke press sequence k.Bindings gives, so that key.Matches agrees
// with the Dispatcher wherever BubblesKeys agrees with Matches. A chord,
// and a release or repeat stroke, which a key.Binding cannot tell from a
// press, are listed as their full text, such as "ctrl+x n": no key prints
// it, so key.Matches never matches it, and help shows it. A key.Binding
// has no when, so a binding with one matches whether it holds or not.
//
// Its help is the primary sequence and the command's title, from d, or
// the ID when d is nil or has no title. A command bound to nothing along
// the path, or removed, gives a disabled binding, which help skips.
func KeyBinding(k *keymap.Keymap, id command.ID, p keymap.Path, d keymap.Describer) key.Binding {
	return bindingFor(k.Shown(id, p), id, d)
}

// bindingFor is KeyBinding's binding for the bindings shown.
func bindingFor(shown []keymap.Binding, id command.ID, d keymap.Describer) key.Binding {
	if len(shown) == 0 {
		return key.NewBinding(key.WithDisabled())
	}
	var keys []string
	for _, b := range shown {
		spell := []string{b.Keys.String()}
		if len(b.Keys) == 1 && b.Keys[0].Event == keymap.Press {
			spell = BubblesKeys(b.Keys[0])
		}
		for _, s := range spell {
			if !slices.Contains(keys, s) {
				keys = append(keys, s)
			}
		}
	}
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(shown[0].Keys.String(), title(id, d)))
}

// Help is a help.KeyMap for the commands ids along a path (0007-MADR
// amendment A7). The short help is each command's KeyBinding, in the order
// given. The full help lists the same commands, each once, in a column for
// the scope its primary sequence is bound in, innermost scope first. With
// no ids, both list every command bound along the path, innermost scope
// first and in resolution order within a scope: the later binding first,
// as k.Rules, which is in the order New took them, read backward. A
// command bound to nothing is left out of the full help, and help skips
// its disabled binding in the short.
func Help(k *keymap.Keymap, p keymap.Path, d keymap.Describer, ids ...command.ID) help.KeyMap {
	if len(ids) == 0 {
		rules := slices.Collect(k.Rules())
		for _, scope := range slices.Backward(p) {
			for _, b := range slices.Backward(rules) {
				if b.Scope == scope && !slices.Contains(ids, b.Command) {
					ids = append(ids, b.Command)
				}
			}
		}
	} else {
		ids = slices.Clone(ids)
	}
	h := &helpMap{}
	cols := map[command.Scope][]key.Binding{}
	for _, id := range ids {
		shown := k.Shown(id, p)
		b := bindingFor(shown, id, d)
		h.short = append(h.short, b)
		if len(shown) > 0 {
			cols[shown[0].Scope] = append(cols[shown[0].Scope], b)
		}
	}
	for _, scope := range slices.Backward(p) {
		if c, ok := cols[scope]; ok {
			h.full = append(h.full, c)
			delete(cols, scope) // a path may list a scope twice
		}
	}
	return h
}

// helpMap is Help's help.KeyMap, built once: the keymap does not change.
type helpMap struct {
	short []key.Binding
	full  [][]key.Binding
}

// ShortHelp implements help.KeyMap.
func (h *helpMap) ShortHelp() []key.Binding { return slices.Clone(h.short) }

// FullHelp implements help.KeyMap.
func (h *helpMap) FullHelp() [][]key.Binding {
	out := make([][]key.Binding, len(h.full))
	for i, c := range h.full {
		out[i] = slices.Clone(c)
	}
	return out
}

// BubblesKeys is every k.String() a stroke can produce, for a key.Binding:
// its Keystroke() form, and for a printable stroke with no modifier but
// shift, the text the key types. Three keys are beyond it, since bubbles
// compares a key's String(), its typed text: a caps-lock letter, which
// bubbles cannot tell from shift; a shifted symbol written as shift and
// its key, whose text is the layout's (write the symbol, "?"); and a Scan
// stroke, which names a position, not the text typed.
func BubblesKeys(s keymap.Stroke) []string {
	// A stroke of the key alone, pressed, writes the key's name, as
	// termkey's Keystroke does, or its rune; the rune itself for a Scan
	// stroke's key, which is a printable rune with no name.
	keys := []string{keymap.Stroke{Mod: s.Mod, Code: s.Code}.String()}
	named := keymap.Stroke{Code: s.Code}.String() != string(s.Code)
	if !s.Scan && !named && unicode.IsPrint(s.Code) && s.Mod&^termkey.ModShift == 0 {
		text := string(s.Code)
		if s.Mod.Contains(termkey.ModShift) {
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
