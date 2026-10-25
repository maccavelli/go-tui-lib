package workspace

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// closedMsg is what a closing overlay's command sends, so a test sees that
// the workspace returned it.
type closedMsg struct{ id string }

// closing is a fake whose Update answers OverlayClosedMsg with closedMsg.
type closing struct{ *fake }

func (c *closing) Update(msg tea.Msg) (Pane, tea.Cmd) {
	c.fake.Update(msg)
	if _, ok := msg.(OverlayClosedMsg); ok {
		id := c.id
		return c, func() tea.Msg { return closedMsg{id: id} }
	}
	return c, nil
}

// closes counts the OverlayClosedMsgs f got, and reports whether the first
// came after a PaneBlurMsg.
func closes(f *fake) (n int, afterBlur bool) {
	blur := slices.Index(f.msgs, tea.Msg(PaneBlurMsg{}))
	at := slices.Index(f.msgs, tea.Msg(OverlayClosedMsg{}))
	return len(filter[OverlayClosedMsg](f.msgs)), blur >= 0 && at > blur
}

// TestOverlayClosedMsg: each way the workspace closes an overlay, Pop,
// Close, the esc binding and a Push that replaces its ID, sends it
// OverlayClosedMsg once, after PaneBlurMsg when it had the keyboard, and
// returns its command; an overlay still open is sent none
// (docs/decisions/0008-MADR-command-palette.md A6).
func TestOverlayClosedMsg(t *testing.T) {
	cases := []struct {
		name  string
		modal bool
		close func(w *Workspace) tea.Cmd
	}{
		{"pop", true, func(w *Workspace) tea.Cmd { return w.Pop() }},
		{"close", true, func(w *Workspace) tea.Cmd { return w.Update(Close("dlg")) }},
		{"esc", true, func(w *Workspace) tea.Cmd { return w.Update(press("esc")) }},
		{"push", true, func(w *Workspace) tea.Cmd {
			return w.Push(Overlay{ID: "dlg", Pane: &fake{id: "new"}, Width: 20, Height: 6, Modal: true})
		}},
		{"close non-modal", false, func(w *Workspace) tea.Cmd { return w.Update(Close("dlg")) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t, 120, 40)
			under := &fake{id: "under"}
			r.w.Push(Overlay{ID: "under", Pane: under, Width: 10, Height: 4})
			dlg := &closing{fake: &fake{id: "dlg"}}
			r.w.Push(Overlay{ID: "dlg", Pane: dlg, Width: 20, Height: 6, Modal: tc.modal})
			if n, _ := closes(dlg.fake); n != 0 {
				t.Fatalf("an open overlay got %d OverlayClosedMsgs", n)
			}
			msgs := run(tc.close(r.w))
			n, afterBlur := closes(dlg.fake)
			if n != 1 {
				t.Fatalf("the closed overlay got %d OverlayClosedMsgs; want 1", n)
			}
			if tc.modal && !afterBlur {
				t.Errorf("OverlayClosedMsg came before PaneBlurMsg: %v", dlg.msgs)
			}
			if !slices.Contains(msgs, tea.Msg(closedMsg{id: "dlg"})) {
				t.Errorf("the closed overlay's command was not returned: %v", msgs)
			}
			if n, _ := closes(under); n != 0 {
				t.Errorf("the overlay beneath, still open, got %d OverlayClosedMsgs", n)
			}
			if slices.Contains(r.w.Overlays(), "dlg") && tc.name != "push" {
				t.Errorf("dlg is still open: %v", r.w.Overlays())
			}
		})
	}

	// An overlay closed beneath another, without the keyboard, is told too.
	r := newRig(t, 120, 40)
	low := &closing{fake: &fake{id: "low"}}
	r.w.Push(Overlay{ID: "low", Pane: low, Width: 20, Height: 6, Modal: true})
	r.w.Push(Overlay{ID: "top", Pane: &fake{id: "top"}, Width: 20, Height: 6, Modal: true})
	low.reset()
	run(r.w.Update(Close("low")))
	if n, _ := closes(low.fake); n != 1 || low.got(PaneBlurMsg{}) {
		t.Errorf("a covered overlay closed: %d OverlayClosedMsgs, blur %v; want 1 and no blur", n, low.got(PaneBlurMsg{}))
	}
}
