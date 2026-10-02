package workspace

import (
	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/layout"
)

// PaneFocusMsg tells a pane, through its Update, that it now has the
// keyboard: it is the focused pane and no modal overlay is open, or it is
// the top modal overlay's pane. Because it arrives through Update, a pane
// with value semantics, such as a bubbles model, keeps the change in the
// value it returns. It is not tea.FocusMsg, which says the terminal gained
// focus.
type PaneFocusMsg struct{}

// PaneBlurMsg tells a pane, through its Update, that it no longer has the
// keyboard. It is not tea.BlurMsg, which says the terminal lost focus.
type PaneBlurMsg struct{}

// focusTarget is who has the keyboard: the top modal overlay's pane, or
// else the focused pane. The zero value is nobody.
type focusTarget struct {
	overlay bool
	id      string
}

func (w *Workspace) focusTarget() focusTarget {
	if o := w.modal(); o != nil {
		return focusTarget{overlay: true, id: o.ID}
	}
	return focusTarget{id: string(w.focus)}
}

// moveFocus tells was that it lost the keyboard and the current target that
// it has it, when they differ. Every change of focus, of the modal overlay
// stack, and of which panes are shown goes through it.
func (w *Workspace) moveFocus(was focusTarget) tea.Cmd {
	now := w.focusTarget()
	if now == was {
		return nil
	}
	return tea.Batch(w.lose(was), w.gain(now))
}

// gain gives t the keyboard: its Focuser first, so a pane written for
// v0.1.0 works unchanged, then PaneFocusMsg through its Update.
func (w *Workspace) gain(t focusTarget) tea.Cmd {
	if t.overlay {
		i := w.overlayIndex(t.id)
		if i < 0 {
			return nil
		}
		var cmd tea.Cmd
		if f, ok := w.overlays[i].Pane.(Focuser); ok {
			cmd = f.Focus()
		}
		return tea.Batch(cmd, w.updateOverlay(i, PaneFocusMsg{}))
	}
	id := layout.PaneID(t.id)
	p, ok := w.panes[id]
	if !ok {
		return nil
	}
	var cmd tea.Cmd
	if f, ok := p.(Focuser); ok {
		cmd = f.Focus()
	}
	return tea.Batch(cmd, w.Send(id, PaneFocusMsg{}))
}

// lose takes the keyboard from t: its Focuser's Blur first, then PaneBlurMsg
// through its Update. A target that no longer exists is told nothing.
func (w *Workspace) lose(t focusTarget) tea.Cmd {
	if t.overlay {
		i := w.overlayIndex(t.id)
		if i < 0 {
			return nil
		}
		if f, ok := w.overlays[i].Pane.(Focuser); ok {
			f.Blur()
		}
		return w.updateOverlay(i, PaneBlurMsg{})
	}
	id := layout.PaneID(t.id)
	p, ok := w.panes[id]
	if !ok {
		return nil
	}
	if f, ok := p.(Focuser); ok {
		f.Blur()
	}
	return w.Send(id, PaneBlurMsg{})
}
