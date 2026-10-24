package keymaptea

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
)

// The helpers are keymap's own tests', over its exported API.

var global = keymap.Path{command.Global}

// allFeatures is every Kitty keyboard feature.
const allFeatures = keymap.Disambiguate | keymap.EventTypes | keymap.AlternateKeys | keymap.AllKeysAsEscapes | keymap.AssociatedText

func seq(t testing.TB, s string) keymap.Sequence {
	t.Helper()
	q, err := keymap.ParseSequence(s, nil)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// bind is a binding of keys to id in scope, made in code in layer l.
func bind(t testing.TB, l keymap.Layer, scope command.Scope, keys string, id command.ID) keymap.Binding {
	t.Helper()
	return keymap.Binding{Keys: seq(t, keys), Command: id, Scope: scope, Location: keymap.Location{Layer: l}}
}

// mustNew builds a keymap that must have no error.
func mustNew(t testing.TB, s keymap.Settings, defaults []keymap.Default, layers ...[]keymap.Binding) *keymap.Keymap {
	t.Helper()
	k, err := keymap.New(s, defaults, layers...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return k
}

func defaults(bs ...keymap.Binding) []keymap.Default {
	ds := make([]keymap.Default, len(bs))
	for i, b := range bs {
		ds[i] = keymap.Default{Binding: b}
	}
	return ds
}

func mustStroke(t *testing.T, s string) keymap.Stroke {
	t.Helper()
	st, err := keymap.ParseStroke(s)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// press and release are the messages for a stroke's key.
func press(t *testing.T, s string) tea.KeyPressMsg {
	t.Helper()
	st := mustStroke(t, s)
	return tea.KeyPressMsg{Code: st.Code, Mod: tea.KeyMod(st.Mod)}
}

func release(t *testing.T, s string) tea.KeyReleaseMsg {
	t.Helper()
	return tea.KeyReleaseMsg(press(t, s))
}
