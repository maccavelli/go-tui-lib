package workspace

import (
	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/layout"
)

// AnchorKind is where an overlay is placed.
type AnchorKind uint8

const (
	// Center centres the overlay in the workspace.
	Center AnchorKind = iota
	// BelowCursor places the overlay under the focused pane's cursor, as a
	// completion pop-up is.
	BelowCursor
	// OnPane places the overlay at an offset from a pane's content area.
	OnPane
)

// Anchor places an overlay.
type Anchor struct {
	Kind AnchorKind
	// Pane and DX, DY place an OnPane overlay.
	Pane   layout.PaneID
	DX, DY int
}

// Overlay is a pane drawn above the layout: a dialog, a picker, a pop-up.
type Overlay struct {
	// ID names the overlay; Send and To reach it by this ID.
	ID string
	// Pane is the overlay's content.
	Pane Pane
	// Anchor places it.
	Anchor Anchor
	// Width and Height are its size, chrome included. They are clamped to
	// the workspace.
	Width, Height int
	// Modal overlays take every key and every mouse event until closed. A
	// non-modal overlay takes only esc and the mouse events over it; other
	// keys still reach the focused pane, which drives it.
	Modal bool
}

// Push opens an overlay above any already open, and focuses its pane.
func (w *Workspace) Push(o Overlay) tea.Cmd {
	w.overlays = append(w.overlays, o)
	cmds := []tea.Cmd{w.updateOverlay(len(w.overlays)-1, w.overlayContent(o))}
	if f, ok := o.Pane.(Focuser); ok {
		cmds = append(cmds, f.Focus())
	}
	return tea.Batch(cmds...)
}

// Pop closes the top overlay.
func (w *Workspace) Pop() tea.Cmd {
	n := len(w.overlays)
	if n == 0 {
		return nil
	}
	if f, ok := w.overlays[n-1].Pane.(Focuser); ok {
		f.Blur()
	}
	w.overlays = w.overlays[:n-1]
	return nil
}

// Overlays returns the open overlays' IDs, bottom first.
func (w *Workspace) Overlays() []string {
	out := make([]string, len(w.overlays))
	for i, o := range w.overlays {
		out[i] = o.ID
	}
	return out
}

func (w *Workspace) modal() *Overlay {
	if n := len(w.overlays); n > 0 && w.overlays[n-1].Modal {
		return &w.overlays[n-1]
	}
	return nil
}

func (w *Workspace) updateOverlay(i int, msg tea.Msg) tea.Cmd {
	next, cmd := w.overlays[i].Pane.Update(msg)
	if next != nil {
		w.overlays[i].Pane = next
	}
	return cmd
}

// overlayRect places an overlay in the workspace.
func (w *Workspace) overlayRect(o Overlay) layout.Rect {
	ow, oh := min(max(o.Width, 3), w.width), min(max(o.Height, 3), w.height)
	r := layout.Rect{X: (w.width - ow) / 2, Y: (w.height - oh) / 2, W: ow, H: oh}
	switch o.Anchor.Kind {
	case BelowCursor:
		if c := w.Cursor(); c != nil {
			r.X, r.Y = c.X, c.Y+1
		}
	case OnPane:
		if p, ok := w.plan.Panes[o.Anchor.Pane]; ok {
			in := w.content(o.Anchor.Pane, p)
			r.X, r.Y = in.X+o.Anchor.DX, in.Y+o.Anchor.DY
		}
	}
	// Keep it on screen.
	r.X = max(0, min(r.X, w.width-r.W))
	r.Y = max(0, min(r.Y, w.height-r.H))
	return r
}

// overlayContent is an overlay's content size: its rectangle less a border.
func (w *Workspace) overlayContent(o Overlay) SizeMsg {
	in := insetBorder(w.overlayRect(o))
	return SizeMsg{Width: in.W, Height: in.H}
}
