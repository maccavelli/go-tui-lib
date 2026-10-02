package workspace

import (
	"slices"

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
	// ID names the overlay. Overlay IDs are a namespace of their own, so an
	// overlay may share an ID with a pane. SendOverlay reaches it by this ID,
	// and Send and To do when no pane has the ID. Pushing an ID already open
	// replaces that overlay.
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

// Push opens an overlay above any already open, tells it its size, and
// focuses its pane. An overlay already open with the same ID is closed
// first, so the new one replaces it, on top.
func (w *Workspace) Push(o Overlay) tea.Cmd {
	if i := w.overlayIndex(o.ID); i >= 0 {
		w.closeOverlay(i)
	}
	w.overlays = append(w.overlays, o)
	sz := w.overlayContent(o)
	w.osizes[o.ID] = sz
	cmds := []tea.Cmd{w.updateOverlay(len(w.overlays)-1, sz)}
	if f, ok := o.Pane.(Focuser); ok {
		cmds = append(cmds, f.Focus())
	}
	return tea.Batch(cmds...)
}

// Pop closes the top overlay.
func (w *Workspace) Pop() tea.Cmd {
	if n := len(w.overlays); n > 0 {
		w.closeOverlay(n - 1)
	}
	return nil
}

// closeOverlay blurs overlay i, removes it, and forgets its size and its
// cached view.
func (w *Workspace) closeOverlay(i int) {
	o := w.overlays[i]
	if f, ok := o.Pane.(Focuser); ok {
		f.Blur()
	}
	w.overlays = slices.Delete(w.overlays, i, i+1)
	delete(w.osizes, o.ID)
	delete(w.cache, viewKey{overlayView, o.ID})
}

// overlayIndex is the position of the open overlay with id, or -1.
func (w *Workspace) overlayIndex(id string) int {
	return slices.IndexFunc(w.overlays, func(o Overlay) bool { return o.ID == id })
}

// SendOverlay delivers msg to the open overlay id now, and returns its
// command. Unlike Send, it never reaches a pane.
func (w *Workspace) SendOverlay(id string, msg tea.Msg) tea.Cmd {
	if i := w.overlayIndex(id); i >= 0 {
		return w.updateOverlay(i, msg)
	}
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
		// The focused pane's cursor, never Cursor(): with a modal overlay
		// open, Cursor() asks for this rectangle, and the two would recurse.
		if c := w.paneCursor(); c != nil {
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
