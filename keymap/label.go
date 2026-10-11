package keymap

import (
	"maps"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/glyph"
)

// keyAliases and modAliases are the names ParseStroke reads on input only,
// each with the canonical name it stands for (MADR amendment A1).
var (
	keyAliases = map[string]string{
		"return": "enter", "escape": "esc", "pageup": "pgup", "pagedown": namePgDown, "del": "delete",
	}
	modAliases = map[string]string{"option": "alt", "cmd": nameSuper, "win": nameSuper}
)

// displayAliases are the shorter names Label writes, by canonical name.
var displayAliases = map[string]string{namePgDown: "pgdn", "delete": "del"}

// ParseAliases is the names ParseStroke reads besides the canonical ones,
// each with the canonical name it stands for: "return", "escape",
// "pageup", "pagedown", "del", "option", "cmd" and "win". String never
// writes them; VSCode writes VS Code's own, "escape", "pageup",
// "pagedown" and "cmd". The map is the caller's.
func ParseAliases() map[string]string {
	m := maps.Clone(keyAliases)
	maps.Copy(m, modAliases)
	return m
}

// DisplayAliases is the shorter names Label writes, each by the canonical
// name it stands for: "pgdn" and "del". Bubble Tea's own "pgup" is short
// already. String never writes them. The map is the caller's.
func DisplayAliases() map[string]string { return maps.Clone(displayAliases) }

// CopyStrokes are the keys a terminal may send for copy: ctrl+c, super+c
// and ctrl+shift+c, for a program's own selection scope (amendment A1).
// keymap binds none of them, and ctrl+c stays the program's.
func CopyStrokes() []Stroke {
	return []Stroke{
		{Code: 'c', Mod: tea.ModCtrl},
		{Code: 'c', Mod: tea.ModSuper},
		{Code: 'c', Mod: tea.ModCtrl | tea.ModShift},
	}
}

// Label is the stroke as a person reads it, on goos, the system the user
// types on, as runtime.GOOS names it (amendments A1 and A8):
//   - on darwin, ctrl, alt, shift and super are g.Keys' ⌃ ⌥ ⇧ ⌘, or their
//     ASCII twins "ctrl+", "opt+", "shift+" and "cmd+", in that order and
//     before the key; meta and hyper, which have no glyph, are "meta+" and
//     "hyper+";
//   - elsewhere, each modifier is its canonical name and "+";
//   - the arrows are g.Keys' ↑ ↓ ← → or "up", "down", "left" and "right",
//     and DisplayAliases shortens "pgdown" and "delete";
//   - a scan stroke keeps its code, a release or repeat stroke its suffix,
//     and the leader placeholder is "<leader>".
//
// A label is for display: ParseStroke does not read it.
func (s Stroke) Label(goos string, g glyph.Set) string {
	if s.Code == keyLeader {
		return leaderName
	}
	mac := map[tea.KeyMod]string{
		tea.ModCtrl: g.Keys.Control, tea.ModAlt: g.Keys.Option, tea.ModShift: g.Keys.Shift, tea.ModSuper: g.Keys.Command,
	}
	var b strings.Builder
	for _, m := range modifiers {
		if !s.Mod.Contains(m.mod) {
			continue
		}
		if l, ok := mac[m.mod]; ok && goos == "darwin" {
			b.WriteString(l)
		} else {
			b.WriteString(m.canon + "+")
		}
	}
	b.WriteString(s.keyLabel(g))
	if s.Event != Press {
		b.WriteString(":" + s.Event.String())
	}
	return b.String()
}

// keyLabel is the stroke's key as Label writes it.
func (s Stroke) keyLabel(g glyph.Set) string {
	if !s.Scan {
		switch s.Code {
		case tea.KeyUp:
			return g.Keys.Up
		case tea.KeyDown:
			return g.Keys.Down
		case tea.KeyLeft:
			return g.Keys.Left
		case tea.KeyRight:
			return g.Keys.Right
		}
	}
	text := s.text()
	if d, ok := displayAliases[text]; ok && !s.Scan {
		return d
	}
	return text
}

// Shortcut is the label of id's primary sequence along a path, or "" when
// nothing binds it there, so a hint built from it is dropped rather than
// left stale after a rebind (amendment A1).
func (k *Keymap) Shortcut(id command.ID, p Path, goos string, g glyph.Set) string {
	qs := k.Bindings(id, p)
	if len(qs) == 0 {
		return ""
	}
	return qs[0].Label(goos, g)
}

// Label is the strokes' labels, joined by one space. A keymap's bindings
// have <leader> expanded, so their labels show the leader's own keys.
func (q Sequence) Label(goos string, g glyph.Set) string {
	return q.join(func(s Stroke) string { return s.Label(goos, g) })
}
