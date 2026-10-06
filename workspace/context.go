package workspace

import (
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/when"
)

// Contexter is a pane or overlay content that publishes when keys of its
// own, such as an editor's mode. WhenContext layers them above the
// workspace's.
type Contexter interface {
	WhenContext() when.Context
}

// The workspace's context keys (docs/decisions/0006-MADR-command-registry.md
// §8, A8). A command's When reads them; ContextKeys lists them for
// when.Check.
var (
	// KeyFocusedPane is the focused pane's ID.
	KeyFocusedPane = when.NewKey[string]("workspace.focusedPane", "the focused pane's ID")
	// KeyZoomed is whether a pane is zoomed.
	KeyZoomed = when.NewKey[bool]("workspace.zoomed", "whether a pane is zoomed")
	// KeyHiddenPanes is every pane the layout knows and is not showing:
	// hidden by the user, dropped by a responsive rule, or squeezed out.
	KeyHiddenPanes = when.NewKey[[]string]("workspace.hiddenPanes", "the panes the layout knows and is not showing")
	// KeyOverlay is the top overlay's ID, or "" with none open.
	KeyOverlay = when.NewKey[string]("workspace.overlay", "the top overlay's ID")
	// KeyModal is whether the top overlay is modal.
	KeyModal = when.NewKey[bool]("workspace.modal", "whether the top overlay is modal")
	// KeyWidth is the workspace's width in cells.
	KeyWidth = when.NewKey[int]("workspace.width", "the workspace's width in cells")
	// KeyHeight is the workspace's height in cells.
	KeyHeight = when.NewKey[int]("workspace.height", "the workspace's height in cells")
)

// ContextKeys is the workspace's keys and their kinds, for when.Check.
func ContextKeys() when.Keys {
	return when.Keys{
		KeyFocusedPane.Name: KeyFocusedPane.Kind(),
		KeyZoomed.Name:      KeyZoomed.Kind(),
		KeyHiddenPanes.Name: KeyHiddenPanes.Kind(),
		KeyOverlay.Name:     KeyOverlay.Kind(),
		KeyModal.Name:       KeyModal.Kind(),
		KeyWidth.Name:       KeyWidth.Kind(),
		KeyHeight.Name:      KeyHeight.Kind(),
	}
}

// WhenContext is the context a command's When is evaluated in: the top
// overlay's keys, if its content is a Contexter, then the focused pane's,
// then the workspace's own. The first that holds a key wins.
func (w *Workspace) WhenContext() when.Context {
	m := when.Map{}
	KeyFocusedPane.Set(m, string(w.focus))
	KeyZoomed.Set(m, w.state.Zoom != "")
	KeyHiddenPanes.Set(m, paneNames(w.plan.Hidden))
	KeyWidth.Set(m, w.width)
	KeyHeight.Set(m, w.height)
	top, modal := "", false
	var layers []when.Context
	if n := len(w.overlays); n > 0 {
		o := w.overlays[n-1]
		top, modal = o.ID, o.Modal
		if c, ok := o.Pane.(Contexter); ok {
			layers = append(layers, c.WhenContext())
		}
	}
	KeyOverlay.Set(m, top)
	KeyModal.Set(m, modal)
	if c, ok := w.panes[w.focus].(Contexter); ok {
		layers = append(layers, c.WhenContext())
	}
	return when.Layered(append(layers, m)...)
}

func paneNames(ids []layout.PaneID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}
