package workspace

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
)

func TestModalOverlayBelowCursor(t *testing.T) {
	r := newRig(t, 120, 40)
	r.main.cursor = tea.NewCursor(2, 1)
	in := r.w.content("main", r.w.plan.Panes["main"])
	// On v0.1.0 this Push never returns: overlayRect and Cursor call each
	// other until the stack overflows.
	r.w.Push(Overlay{ID: "picker", Pane: &fake{id: "picker"}, Anchor: Anchor{Kind: BelowCursor}, Width: 20, Height: 5, Modal: true})
	got := r.w.overlayRect(r.w.overlays[0])
	if got.X != in.X+2 || got.Y != in.Y+2 {
		t.Fatalf("modal BelowCursor overlay at (%d,%d), want one row under the cursor at (%d,%d)", got.X, got.Y, in.X+2, in.Y+2)
	}
	if c := r.w.Cursor(); c != nil {
		t.Fatalf("with a modal overlay whose pane has no cursor, Cursor() = %+v, want nil", c)
	}
}

func TestSetPaneDropsTheCachedView(t *testing.T) {
	first := &stable{fake: &fake{id: "main", body: "FIRST"}}
	w := New(layout.Pane{ID: "main"}, map[layout.PaneID]Pane{"main": first}, WithTheme(asciiTheme))
	w.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	w.Render()
	w.SetPane("main", &stable{fake: &fake{id: "main", body: "SECOND"}})
	if frame := ansi.Strip(w.Render()); !strings.Contains(frame, "SECOND") || strings.Contains(frame, "FIRST") {
		t.Fatalf("after SetPane, an unchanged replacement shows the old view:\n%s", frame)
	}
}

func TestPaneAndOverlayWithOneID(t *testing.T) {
	pane := &stable{fake: &fake{id: "main", body: "PANEVIEW"}}
	over := &stable{fake: &fake{id: "main", body: "OVERLAYVIEW"}}
	w := New(layout.Pane{ID: "main"}, map[layout.PaneID]Pane{"main": pane}, WithTheme(asciiTheme))
	w.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	// A non-modal overlay of the pane's size: both are focused, at one
	// content size, so on v0.1.0 they share the cache entry "box:main".
	w.Push(Overlay{ID: "main", Pane: over, Width: 40, Height: 10})
	frame := ansi.Strip(w.Render())
	if !strings.Contains(frame, "OVERLAYVIEW") || pane.views != 1 || over.views != 1 {
		t.Fatalf("pane viewed %d times, overlay %d times; frame:\n%s", pane.views, over.views, frame)
	}
	type ping struct{}
	w.SendOverlay("main", ping{})
	if pane.got(ping{}) || !over.got(ping{}) {
		t.Fatal("SendOverlay did not reach only the overlay")
	}
	type poke struct{}
	w.Send("main", poke{})
	if !pane.got(poke{}) || over.got(poke{}) {
		t.Fatal("Send did not try the pane first")
	}
}

func TestOverlayIsToldItsNewSize(t *testing.T) {
	r := newRig(t, 120, 40)
	dlg := &fake{id: "dialog"}
	r.w.Push(Overlay{ID: "dialog", Pane: dlg, Width: 100, Height: 30, Modal: true})
	r.w.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	if s := dlg.sizes(); len(s) != 2 || s[0] != (SizeMsg{Width: 98, Height: 28}) || s[1] != (SizeMsg{Width: 58, Height: 18}) {
		t.Fatalf("overlay sizes %v, want 98x28 then 58x18", s)
	}
	r.w.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	if n := len(dlg.sizes()); n != 2 {
		t.Fatalf("an unchanged size was sent again: %d sizes", n)
	}
}

func TestPushReplacesAnOpenID(t *testing.T) {
	r := newRig(t, 120, 40)
	r.w.Push(Overlay{ID: "x", Pane: &fake{id: "x", body: "OLDBODY"}, Width: 30, Height: 6})
	r.w.Push(Overlay{ID: "y", Pane: &fake{id: "y"}, Width: 10, Height: 4})
	second := &fake{id: "x", body: "NEWBODY"}
	r.w.Push(Overlay{ID: "x", Pane: second, Width: 30, Height: 6})
	if got := r.w.Overlays(); !slices.Equal(got, []string{"y", "x"}) {
		t.Fatalf("overlays %v, want [y x]: the open x replaced and moved to the top", got)
	}
	frame := ansi.Strip(r.w.Render())
	if !strings.Contains(frame, "NEWBODY") || strings.Contains(frame, "OLDBODY") {
		t.Fatalf("the replaced overlay is still drawn:\n%s", frame)
	}
	if s := second.sizes(); len(s) != 1 {
		t.Fatalf("the replacement got sizes %v, want one", s)
	}
}

func TestPopEvictsTheOverlay(t *testing.T) {
	r := newRig(t, 120, 40)
	r.w.Push(Overlay{ID: "qqpopped", Pane: &fake{id: "qqpopped"}, Width: 30, Height: 6})
	r.w.Render()
	r.w.Pop()
	for k := range r.w.cache {
		if strings.Contains(fmt.Sprint(k), "qqpopped") {
			t.Fatalf("after Pop the cache still holds %v", k)
		}
	}
}
