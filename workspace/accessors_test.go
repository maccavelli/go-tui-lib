package workspace

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-tui-lib/layout"
)

func TestPanesYieldsTheFocusRing(t *testing.T) {
	r := newRig(t, 120, 40) // the footer is not focusable
	var ids []layout.PaneID
	for id, p := range r.w.Panes() {
		if p != r.w.panes[id] {
			t.Fatalf("%s: yielded %v, want the hosted pane", id, p)
		}
		ids = append(ids, id)
	}
	if !slices.Equal(ids, r.w.focusRing()) || !slices.Equal(ids, []layout.PaneID{"main", "side", "logs"}) {
		t.Fatalf("Panes yielded %v, want the focus ring %v", ids, r.w.focusRing())
	}
	ringed := newRig(t, 120, 40, WithFocusRing("logs", "main"))
	ids = ids[:0]
	for id := range ringed.w.Panes() {
		ids = append(ids, id)
		break // and a loop that breaks stops it
	}
	if !slices.Equal(ids, []layout.PaneID{"logs"}) {
		t.Fatalf("with WithFocusRing(logs, main), a broken loop got %v, want [logs]", ids)
	}
}

func TestPaneAs(t *testing.T) {
	r := newRig(t, 120, 40)
	pop := &stable{fake: &fake{id: "pop"}}
	r.w.Push(Overlay{ID: "pop", Pane: pop, Width: 20, Height: 5})
	if p, ok := r.w.PaneAs[*fake]("main"); !ok || p != r.main {
		t.Fatalf("PaneAs[*fake](main) = %v, %v; want the main pane", p, ok)
	}
	if p, ok := r.w.PaneAs[*stable]("pop"); !ok || p != pop {
		t.Fatalf("PaneAs[*stable](pop) = %v, %v; want the overlay's pane", p, ok)
	}
	if p, ok := r.w.PaneAs[*stable]("main"); ok || p != nil {
		t.Fatalf("PaneAs[*stable](main) = %v, %v; want false for the wrong type", p, ok)
	}
	if p, ok := r.w.PaneAs[*fake]("absent"); ok || p != nil {
		t.Fatalf("PaneAs[*fake](absent) = %v, %v; want false", p, ok)
	}
	// A pane and an overlay may share an ID; PaneAs finds the pane first,
	// as Send does.
	r.w.Push(Overlay{ID: "main", Pane: &stable{fake: &fake{id: "main"}}, Width: 20, Height: 5})
	if p, ok := r.w.PaneAs[*fake]("main"); !ok || p != r.main {
		t.Fatal("with an overlay named main open, PaneAs did not find the pane first")
	}
}
