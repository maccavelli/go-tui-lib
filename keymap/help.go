package keymap

import (
	"slices"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"

	"github.com/maccavelli/go-tui-lib/v2/command"
)

// Describer gives a command's title for help (MADR §7, amendment A4).
type Describer interface {
	// Describe is id's title, and false when there is none.
	Describe(id command.ID) (title string, ok bool)
}

// DescriberFunc is a function that is a Describer.
type DescriberFunc func(id command.ID) (title string, ok bool)

// Describe calls f.
func (f DescriberFunc) Describe(id command.ID) (string, bool) { return f(id) }

// FromRegistry describes commands through Registry.Lookup and
// Command.Title. A command it does not hold, or one with no title, has no
// title.
func FromRegistry(r *command.Registry) Describer {
	return DescriberFunc(func(id command.ID) (string, bool) {
		c, ok := r.Lookup(id)
		if !ok || c.Title == "" {
			return "", false
		}
		return c.Title, true
	})
}

// title is id's title from d, or the ID itself when d is nil or has none.
func title(id command.ID, d Describer) string {
	if d != nil {
		if t, ok := d.Describe(id); ok {
			return t
		}
	}
	return string(id)
}

// KeyBinding is a bubbles key.Binding for id along a path (MADR §7,
// amendment A7). Its keys are every BubblesKeys spelling of each
// one-stroke press sequence Bindings gives, so that key.Matches agrees
// with the Dispatcher wherever BubblesKeys agrees with Matches. A chord,
// and a release or repeat stroke, which a key.Binding cannot tell from a
// press, are listed as their full text, such as "ctrl+x n": no key prints
// it, so key.Matches never matches it, and help shows it. A key.Binding
// has no when, so a binding with one matches whether it holds or not.
//
// Its help is the primary sequence and the command's title, from d, or
// the ID when d is nil or has no title. A command bound to nothing along
// the path, or removed, gives a disabled binding, which help skips.
func (k *Keymap) KeyBinding(id command.ID, p Path, d Describer) key.Binding {
	return keyBinding(k.shown(id, p), id, d)
}

// keyBinding is KeyBinding's binding for the entries shown.
func keyBinding(shown []*entry, id command.ID, d Describer) key.Binding {
	if len(shown) == 0 {
		return key.NewBinding(key.WithDisabled())
	}
	var keys []string
	for _, e := range shown {
		spell := []string{e.Keys.String()}
		if len(e.Keys) == 1 && e.Keys[0].Event == Press {
			spell = e.Keys[0].BubblesKeys()
		}
		for _, s := range spell {
			if !slices.Contains(keys, s) {
				keys = append(keys, s)
			}
		}
	}
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(shown[0].Keys.String(), title(id, d)))
}

// Help is a help.KeyMap for the commands ids along a path (amendment A7).
// The short help is each command's KeyBinding, in the order given. The
// full help lists the same commands, each once, in a column for the scope
// its primary sequence is bound in, innermost scope first. With no ids,
// both list every command bound along the path, innermost scope first and
// in resolution order within a scope. A command bound to nothing is left
// out of the full help, and help skips its disabled binding in the short.
func (k *Keymap) Help(p Path, d Describer, ids ...command.ID) help.KeyMap {
	if len(ids) == 0 {
		for _, scope := range slices.Backward(p) {
			for _, e := range k.scopes[scope] {
				if !slices.Contains(ids, e.Command) {
					ids = append(ids, e.Command)
				}
			}
		}
	} else {
		ids = slices.Clone(ids)
	}
	h := &helpMap{}
	cols := map[command.Scope][]key.Binding{}
	for _, id := range ids {
		shown := k.shown(id, p)
		b := keyBinding(shown, id, d)
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
