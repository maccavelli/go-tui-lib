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
// when.Check. The functions return them, and the workspace reads only
// these, so no program can change what it publishes
// (docs/decisions/0014-PLAN-hardening.md Step 8, finding H11).
var (
	ctxFocusedPane = when.NewKey[string]("workspace.focusedPane", "the focused pane's ID")
	ctxZoomed      = when.NewKey[bool]("workspace.zoomed", "whether a pane is zoomed")
	ctxHiddenPanes = when.NewKey[[]string]("workspace.hiddenPanes", "the panes the layout knows and is not showing")
	ctxOverlay     = when.NewKey[string]("workspace.overlay", "the top overlay's ID")
	ctxModal       = when.NewKey[bool]("workspace.modal", "whether the top overlay is modal")
	ctxWidth       = when.NewKey[int]("workspace.width", "the workspace's width in cells")
	ctxHeight      = when.NewKey[int]("workspace.height", "the workspace's height in cells")
)

// FocusedPaneKey is the key of the focused pane's ID.
func FocusedPaneKey() when.Key[string] { return ctxFocusedPane }

// ZoomedKey is the key of whether a pane is zoomed.
func ZoomedKey() when.Key[bool] { return ctxZoomed }

// HiddenPanesKey is the key of every pane the layout knows and is not
// showing: hidden by the user, dropped by a responsive rule, or squeezed
// out.
func HiddenPanesKey() when.Key[[]string] { return ctxHiddenPanes }

// OverlayKey is the key of the top overlay's ID, or "" with none open.
func OverlayKey() when.Key[string] { return ctxOverlay }

// ModalKey is the key of whether the top overlay is modal.
func ModalKey() when.Key[bool] { return ctxModal }

// WidthKey is the key of the workspace's width in cells.
func WidthKey() when.Key[int] { return ctxWidth }

// HeightKey is the key of the workspace's height in cells.
func HeightKey() when.Key[int] { return ctxHeight }

// The workspace's context keys as variables. Reassigning one changes
// nothing the workspace publishes. They are removed in v0.9.0.
var (
	// KeyFocusedPane is the key of the focused pane's ID.
	//
	// Deprecated: use FocusedPaneKey.
	KeyFocusedPane = ctxFocusedPane

	// KeyZoomed is the key of whether a pane is zoomed.
	//
	// Deprecated: use ZoomedKey.
	KeyZoomed = ctxZoomed

	// KeyHiddenPanes is the key of every pane the layout knows and is not
	// showing: hidden by the user, dropped by a responsive rule, or
	// squeezed out.
	//
	// Deprecated: use HiddenPanesKey.
	KeyHiddenPanes = ctxHiddenPanes

	// KeyOverlay is the key of the top overlay's ID, or "" with none open.
	//
	// Deprecated: use OverlayKey.
	KeyOverlay = ctxOverlay

	// KeyModal is the key of whether the top overlay is modal.
	//
	// Deprecated: use ModalKey.
	KeyModal = ctxModal

	// KeyWidth is the key of the workspace's width in cells.
	//
	// Deprecated: use WidthKey.
	KeyWidth = ctxWidth

	// KeyHeight is the key of the workspace's height in cells.
	//
	// Deprecated: use HeightKey.
	KeyHeight = ctxHeight
)

// ContextKeys is the workspace's keys and their kinds, for when.Check.
func ContextKeys() when.Keys {
	return when.Keys{
		ctxFocusedPane.Name: ctxFocusedPane.Kind(),
		ctxZoomed.Name:      ctxZoomed.Kind(),
		ctxHiddenPanes.Name: ctxHiddenPanes.Kind(),
		ctxOverlay.Name:     ctxOverlay.Kind(),
		ctxModal.Name:       ctxModal.Kind(),
		ctxWidth.Name:       ctxWidth.Kind(),
		ctxHeight.Name:      ctxHeight.Kind(),
	}
}

// WhenContext is the context a command's When is evaluated in: the top
// overlay's keys, if its content is a Contexter, then the focused pane's,
// then the workspace's own. The first that holds a key wins.
func (w *Workspace) WhenContext() when.Context {
	m := when.Map{}
	ctxFocusedPane.Set(m, string(w.focus))
	ctxZoomed.Set(m, w.state.Zoom != "")
	ctxHiddenPanes.Set(m, paneNames(w.plan.Hidden))
	ctxWidth.Set(m, w.width)
	ctxHeight.Set(m, w.height)
	top, modal := "", false
	var layers []when.Context
	if n := len(w.overlays); n > 0 {
		o := w.overlays[n-1]
		top, modal = o.ID, o.Modal
		if c, ok := o.Pane.(Contexter); ok {
			layers = append(layers, c.WhenContext())
		}
	}
	ctxOverlay.Set(m, top)
	ctxModal.Set(m, modal)
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
