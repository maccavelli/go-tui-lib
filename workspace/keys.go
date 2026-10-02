package workspace

import (
	"fmt"

	"charm.land/bubbles/v2/key"
)

// KeyMap is the workspace's own bindings. Every binding can be rebound with
// SetKeys, or removed with Unbind or SetEnabled(false). None is ctrl+c: the
// program owns it (docs/decisions/0001-MADR-scaffold-charm-tui-library.md
// §6, rule 2). The defaults use alt so that tab, the arrows and alt+arrows
// stay with the focused pane, such as an editor.
type KeyMap struct {
	// FocusNext and FocusPrev move focus around the ring.
	FocusNext, FocusPrev key.Binding
	// FocusPane focuses the n-th pane of the ring, 1 to 9.
	FocusPane [9]key.Binding
	// Zoom gives the focused pane the whole area, and restores it.
	Zoom key.Binding
	// ResizeLeft, ResizeRight, ResizeUp and ResizeDown move the separator
	// nearest the focused pane by one cell.
	ResizeLeft, ResizeRight, ResizeUp, ResizeDown key.Binding
	// Close closes the top overlay.
	Close key.Binding
}

// DefaultKeyMap returns the default bindings: alt+] and alt+[ to move focus,
// alt+1 to alt+9 to focus a pane, alt+z to zoom, alt+shift+arrows to resize,
// and esc to close an overlay.
func DefaultKeyMap() KeyMap {
	k := KeyMap{
		FocusNext:   key.NewBinding(key.WithKeys("alt+]"), key.WithHelp("alt+]", "next pane")),
		FocusPrev:   key.NewBinding(key.WithKeys("alt+["), key.WithHelp("alt+[", "previous pane")),
		Zoom:        key.NewBinding(key.WithKeys("alt+z"), key.WithHelp("alt+z", "zoom")),
		ResizeLeft:  key.NewBinding(key.WithKeys("alt+shift+left"), key.WithHelp("alt+shift+left", "resize")),
		ResizeRight: key.NewBinding(key.WithKeys("alt+shift+right"), key.WithHelp("alt+shift+right", "resize")),
		ResizeUp:    key.NewBinding(key.WithKeys("alt+shift+up"), key.WithHelp("alt+shift+up", "resize")),
		ResizeDown:  key.NewBinding(key.WithKeys("alt+shift+down"), key.WithHelp("alt+shift+down", "resize")),
		Close:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
	}
	for i := range k.FocusPane {
		n := fmt.Sprintf("alt+%d", i+1)
		k.FocusPane[i] = key.NewBinding(key.WithKeys(n), key.WithHelp(n, fmt.Sprintf("pane %d", i+1)))
	}
	return k
}

// Bindings lists every binding, for a help footer.
func (k KeyMap) Bindings() []key.Binding {
	out := []key.Binding{k.FocusNext, k.FocusPrev, k.Zoom, k.ResizeLeft, k.ResizeRight, k.ResizeUp, k.ResizeDown, k.Close}
	return append(out, k.FocusPane[:]...)
}
