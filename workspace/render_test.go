package workspace

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
)

// views is how many times any of fs was asked for its view.
func views(fs ...*fake) int {
	n := 0
	for _, f := range fs {
		n += f.views
	}
	return n
}

func TestCleanFrameSkipsEveryView(t *testing.T) {
	r := newRig(t, 120, 40)
	first := r.w.Render()
	before := views(r.main, r.side, r.logs, r.footer)
	if again := r.w.Render(); again != first {
		t.Fatal("an unchanged frame was drawn differently")
	}
	if got := views(r.main, r.side, r.logs, r.footer); got != before {
		t.Fatalf("an unchanged frame asked for %d views", got-before)
	}
}

// changers is a rig whose panes are all unchanged Changers, so a message to
// a pane marks nothing dirty and each case below is the only cause of a
// redraw.
func changers(t *testing.T) (*Workspace, map[string]*stable) {
	t.Helper()
	ps := map[string]*stable{}
	panes := map[layout.PaneID]Pane{}
	for _, id := range []string{"main", "side", "logs"} {
		ps[id] = &stable{fake: &fake{id: id}}
		panes[layout.PaneID(id)] = ps[id]
	}
	w := New(layout.SidebarRightBottom("main", "side", "logs", layout.WithGap(0)), panes, WithTheme(asciiTheme))
	w.Init()
	w.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	w.Render()
	return w, ps
}

func TestEachDirtyCaseRedraws(t *testing.T) {
	cases := []struct {
		name   string
		before func(w *Workspace)
		act    func(w *Workspace)
	}{
		{"size", nil, func(w *Workspace) { w.Update(tea.WindowSizeMsg{Width: 100, Height: 30}) }},
		{"layout", nil, func(w *Workspace) { w.SetLayout(layout.SidebarLeftBottom("main", "side", "logs", layout.WithGap(0))) }},
		{"state", nil, func(w *Workspace) { w.Toggle("logs") }},
		{"focus", nil, func(w *Workspace) { w.FocusNext() }},
		{"overlay push", nil, func(w *Workspace) {
			w.Push(Overlay{ID: "o", Pane: &stable{fake: &fake{id: "o"}}, Width: 20, Height: 5})
		}},
		{"overlay pop", func(w *Workspace) {
			w.Push(Overlay{ID: "o", Pane: &stable{fake: &fake{id: "o"}}, Width: 20, Height: 5})
		}, func(w *Workspace) { w.Pop() }},
		{"replacing push", func(w *Workspace) {
			w.Push(Overlay{ID: "o", Pane: &stable{fake: &fake{id: "o"}}, Width: 20, Height: 5})
		}, func(w *Workspace) {
			w.Push(Overlay{ID: "o", Pane: &stable{fake: &fake{id: "o2"}}, Width: 30, Height: 6})
		}},
		{"a message to a non-Changer pane", func(w *Workspace) {
			w.SetPane("logs", &fake{id: "logs"})
		}, func(w *Workspace) { w.Send("logs", struct{}{}) }},
		{"a message to a non-Changer overlay", func(w *Workspace) {
			w.Push(Overlay{ID: "o", Pane: &fake{id: "o"}, Width: 20, Height: 5})
		}, func(w *Workspace) { w.SendOverlay("o", struct{}{}) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, _ := changers(t)
			if c.before != nil {
				c.before(w)
			}
			w.Render()
			if w.dirty {
				t.Fatal("still dirty after Render")
			}
			c.act(w)
			if !w.dirty {
				t.Fatalf("%s did not mark the frame dirty", c.name)
			}
			w.Render()
		})
	}
	// An unchanged Changer that gets a message marks nothing, and its view
	// is not asked for again.
	w, ps := changers(t)
	n := ps["main"].views
	w.Send("main", struct{}{})
	w.Render()
	if w.dirty || ps["main"].views != n {
		t.Fatal("a message to an unchanged Changer redrew the frame")
	}
}

func TestHidingEveryPaneClearsTheFrame(t *testing.T) {
	r := newRig(t, 60, 20)
	if strings.TrimSpace(ansi.Strip(r.w.Render())) == "" {
		t.Fatal("the first frame is empty; the test proves nothing")
	}
	for _, id := range []layout.PaneID{"main", "side", "logs", "footer"} {
		r.w.Toggle(id)
	}
	if got := strings.TrimSpace(ansi.Strip(r.w.Render())); got != "" {
		t.Fatalf("with every pane hidden the frame still shows %q", got)
	}
}

func TestOverlayWinsTheHitTest(t *testing.T) {
	r := newRig(t, 120, 40)
	pop := &fake{id: "pop"}
	r.w.Push(Overlay{ID: "pop", Pane: pop, Width: 20, Height: 6})
	r.w.Render()
	box := r.w.overlayRect(r.w.overlays[0])
	in := insetBorder(box)
	if !r.w.plan.Panes["main"].Contains(in.X, in.Y) {
		t.Fatal("the overlay is not over main; the test proves nothing")
	}
	r.main.reset()
	r.w.Update(tea.MouseClickMsg{X: in.X, Y: in.Y, Button: tea.MouseLeft})
	clicks := filter[tea.MouseClickMsg](pop.msgs)
	if len(clicks) != 1 || clicks[0].X != 0 || clicks[0].Y != 0 || len(filter[tea.MouseClickMsg](r.main.msgs)) != 0 {
		t.Fatalf("a click inside the overlay: overlay got %v, main got %v", clicks, filter[tea.MouseClickMsg](r.main.msgs))
	}
	// A click on the overlay's border reaches nothing inside it, and
	// nothing beneath it.
	pop.reset()
	r.w.Update(tea.MouseClickMsg{X: box.X, Y: box.Y, Button: tea.MouseLeft})
	if len(pop.msgs) != 0 || len(filter[tea.MouseClickMsg](r.main.msgs)) != 0 || r.w.Focused() != "main" {
		t.Fatalf("a click on the overlay's border: overlay %v, main %v, focus %s", pop.msgs, r.main.msgs, r.w.Focused())
	}
}

func TestRenderAllocs(t *testing.T) {
	w := benchWorkspace(false, asciiTheme)
	w.Render()
	full := testing.AllocsPerRun(50, func() {
		w.dirty = true
		_ = w.Render()
	})
	if full > 650 {
		t.Errorf("a full 200 x 60 frame: %.0f allocations, want at most 650", full)
	}
	clean := testing.AllocsPerRun(50, func() { _ = w.Render() })
	if clean > 2 {
		t.Errorf("an unchanged frame: %.0f allocations, want at most 2", clean)
	}
	t.Logf("allocations: full %.0f, clean %.0f", full, clean)
}
