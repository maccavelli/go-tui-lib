package workspace

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/theme"
)

// The view cache's bound (docs/decisions/0014-PLAN-hardening.md Step 3,
// finding H1).

func TestViewCacheBounded(t *testing.T) {
	w, main := followingWorkspace()
	w.Push(Overlay{ID: "dialog", Pane: &stable{fake: &fake{id: "dialog"}}, Width: 20, Height: 5})
	bound := 2 * (len(w.panes) + len(w.overlays))
	for i := range 500 {
		w.Update(tea.WindowSizeMsg{Width: 60 + i%40, Height: 12 + i%7})
		w.Render()
	}
	for i := range 200 {
		w.SetTheme(theme.New(colorprofile.ASCII, theme.Background(i%3), glyph.ASCII()))
		w.Render()
	}
	for _, m := range []ansi.Method{ansi.GraphemeWidth, ansi.WcWidth} {
		w.setMethod(m)
		w.Render()
	}
	if n := len(w.cache); n > bound {
		t.Fatalf("%d cached views, want at most %d (2 × (panes + overlays))", n, bound)
	}

	// The hit path: an unchanged Changer is not asked again.
	w.dirty = true
	w.Render()
	before := main.views
	w.dirty = true
	w.Render()
	if main.views != before {
		t.Errorf("an unchanged Changer was drawn again: %d views, then %d", before, main.views)
	}
	// A change of size misses, and replaces the slot.
	w.Update(tea.WindowSizeMsg{Width: 99, Height: 19})
	w.Render()
	if main.views == before {
		t.Error("a resized Changer was not drawn again")
	}
	if n := len(w.cache); n > bound {
		t.Errorf("%d cached views after a miss, want at most %d", n, bound)
	}
}
