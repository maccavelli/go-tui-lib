package workspace

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Workspace is a help.KeyMap, so help.Model renders its bindings
// (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §5).
var _ help.KeyMap = (*Workspace)(nil)

// ShortHelp implements help.KeyMap: the bindings of whoever has the
// keyboard, the top modal overlay's pane or else the focused pane, then
// focus-next and zoom. Disabled bindings are left out.
func (w *Workspace) ShortHelp() []key.Binding {
	return enabled(append(w.targetKeys(), w.keys.FocusNext, w.keys.Zoom))
}

// FullHelp implements help.KeyMap: four columns, the bindings of whoever
// has the keyboard, then focus, then layout (zoom and resize), then
// overlays (close). Disabled bindings, and columns left empty, are left out.
func (w *Workspace) FullHelp() [][]key.Binding {
	k := w.keys
	cols := [][]key.Binding{
		w.targetKeys(),
		append([]key.Binding{k.FocusNext, k.FocusPrev}, k.FocusPane[:]...),
		{k.Zoom, k.ResizeLeft, k.ResizeRight, k.ResizeUp, k.ResizeDown},
		{k.Close},
	}
	var out [][]key.Binding
	for _, c := range cols {
		if c = enabled(c); len(c) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// targetKeys are the bindings of whoever has the keyboard, when it is a
// KeyMapper.
func (w *Workspace) targetKeys() []key.Binding {
	t := w.focusTarget()
	var p Pane
	if t.overlay {
		if i := w.overlayIndex(t.id); i >= 0 {
			p = w.overlays[i].Pane
		}
	} else {
		p = w.panes[w.focus]
	}
	if k, ok := p.(KeyMapper); ok {
		return k.Keys()
	}
	return nil
}

// enabled is bs without its disabled bindings.
func enabled(bs []key.Binding) []key.Binding {
	out := make([]key.Binding, 0, len(bs))
	for _, b := range bs {
		if b.Enabled() {
			out = append(out, b)
		}
	}
	return out
}

// View is the frame as a tea.View, with the cursor of whoever has the
// keyboard. The program sets AltScreen, MouseMode and the rest on the
// value it returns; the workspace never sets them
// (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rule 2).
func (w *Workspace) View() tea.View {
	v := tea.NewView(w.Render())
	v.Cursor = w.Cursor()
	return v
}
