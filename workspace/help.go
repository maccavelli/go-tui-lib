package workspace

import (
	"slices"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/keymap/keymaptea"
)

// Workspace is a help.KeyMap, so help.Model renders its bindings
// (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §5).
var _ help.KeyMap = (*Workspace)(nil)

// helpDescs are the workspace's commands' descriptions in its help.
var helpDescs = map[command.ID]string{
	"workspace.focus.next": "next pane", "workspace.focus.prev": "previous pane", "workspace.focus.index": "pane",
	"workspace.zoom": "zoom", cmdResizeDirection: "resize", cmdOverlayClose: "close",
}

// helpBinding is the binding of the workspace's command id as the engine
// has it along the key path, with "overlay" on it so that the overlays'
// key is listed while none is open (docs/decisions/0007-MADR-keymap-engine.md
// A11). A command bound to nothing gives a disabled binding.
func (w *Workspace) helpBinding(id command.ID) key.Binding {
	p := w.KeyPath()
	if !slices.Contains(p, scopeOverlay) {
		p = append(p, scopeOverlay)
	}
	return keymaptea.KeyBinding(w.km, id, p, keymap.DescriberFunc(func(id command.ID) (string, bool) {
		d, ok := helpDescs[id]
		return d, ok
	}))
}

// ShortHelp implements help.KeyMap: the bindings of whoever has the
// keyboard, the top modal overlay's pane or else the focused pane, then
// focus-next and zoom, as the keymap binds them. Disabled bindings are
// left out.
func (w *Workspace) ShortHelp() []key.Binding {
	return enabled(append(w.targetKeys(), w.helpBinding("workspace.focus.next"), w.helpBinding("workspace.zoom")))
}

// FullHelp implements help.KeyMap: four columns, the bindings of whoever
// has the keyboard, then focus, then layout (zoom and resize), then
// overlays (close), as the keymap binds them. Disabled bindings, and
// columns left empty, are left out.
func (w *Workspace) FullHelp() [][]key.Binding {
	b := w.helpBinding
	cols := [][]key.Binding{
		w.targetKeys(),
		{b("workspace.focus.next"), b("workspace.focus.prev"), b("workspace.focus.index")},
		{b("workspace.zoom"), b(cmdResizeDirection)},
		{b(cmdOverlayClose)},
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
