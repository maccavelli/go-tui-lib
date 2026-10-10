package workspace

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/internal/enum"
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

var anchorNames = enum.Names[AnchorKind]{
	Pkg: pkgName, Type: "AnchorKind",
	Tokens: []string{"center", "below-cursor", "on-pane"},
}

// String is the anchor kind's token.
func (k AnchorKind) String() string { return anchorNames.String(k) }

// MarshalText is the anchor kind's token. An anchor kind with no token is an
// error.
func (k AnchorKind) MarshalText() ([]byte, error) { return anchorNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *AnchorKind) UnmarshalText(b []byte) error { return anchorNames.Unmarshal(b, k) }

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

// Push opens an overlay above any already open and tells it its size. A
// modal overlay takes the keyboard: the pane that had it is sent
// PaneBlurMsg, and the overlay's pane PaneFocusMsg. A non-modal overlay
// changes no focus, because the focused pane drives it. An overlay already
// open with the same ID is closed first, so the new one replaces it, on top.
func (w *Workspace) Push(o Overlay) tea.Cmd {
	was := w.focusTarget()
	var cmds []tea.Cmd
	if i := w.overlayIndex(o.ID); i >= 0 {
		if was == (focusTarget{overlay: true, id: o.ID}) {
			// The replaced overlay had the keyboard: it loses it now, and
			// nothing beneath gains it in between.
			cmds = append(cmds, w.lose(was))
			was = focusTarget{}
		}
		w.removeOverlay(i)
	}
	w.overlays = append(w.overlays, o)
	w.dirty = true
	sz := w.overlayContent(o)
	w.osizes[o.ID] = sz
	cmds = append(cmds, w.updateOverlay(len(w.overlays)-1, sz), w.moveFocus(was))
	return tea.Batch(cmds...)
}

// Pop closes the top overlay. If it had the keyboard, its pane is sent
// PaneBlurMsg, and whoever has the keyboard now, the next modal overlay or
// the focused pane, PaneFocusMsg.
func (w *Workspace) Pop() tea.Cmd {
	if n := len(w.overlays); n > 0 {
		return w.closeOverlay(n - 1)
	}
	return nil
}

// closeOverlay closes overlay i, moving the keyboard on if it had it.
func (w *Workspace) closeOverlay(i int) tea.Cmd {
	was := w.focusTarget()
	var cmd tea.Cmd
	if was == (focusTarget{overlay: true, id: w.overlays[i].ID}) {
		cmd = w.lose(was)
		was = focusTarget{}
	}
	w.removeOverlay(i)
	return tea.Batch(cmd, w.moveFocus(was))
}

// removeOverlay removes overlay i and forgets its size and its cached
// views. It sends no message.
func (w *Workspace) removeOverlay(i int) {
	id := w.overlays[i].ID
	w.overlays = slices.Delete(w.overlays, i, i+1)
	delete(w.osizes, id)
	w.forget(overlayView, id)
	w.dirty = true
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
	w.touched(w.overlays[i].Pane)
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
