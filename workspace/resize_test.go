package workspace

import (
	"maps"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/layout"
)

// separator returns the plan's separator with id.
func separator(t *testing.T, w *Workspace, id string) layout.Separator {
	t.Helper()
	for _, s := range w.plan.Separators {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no separator %q in %+v", id, w.plan.Separators)
	return layout.Separator{}
}

func TestHeldResizeKeyHasNoDeadZone(t *testing.T) {
	r := newRig(t, 120, 40)
	for range 50 {
		r.w.Update(press("alt+shift+right"))
	}
	limit := r.w.plan.Panes["main"].W
	if got, applied := r.w.State().Resize["sidebar:0"], r.w.plan.Resize["sidebar:0"]; got != applied {
		t.Fatalf("after 50 presses against the limit: stored %d, applied %d; the store must be what was applied", got, applied)
	}
	r.w.Update(press("alt+shift+left"))
	if got := r.w.plan.Panes["main"].W; got != limit-1 {
		t.Fatalf("one press back from the limit: main is %d cells wide, want %d", got, limit-1)
	}
}

func TestDragPastTheLimitAndBack(t *testing.T) {
	r := newRig(t, 120, 30, WithChrome(Separators))
	r.w.SetLayout(layout.SidebarRight("main", "side"))
	r.w.Render()
	x := separator(t, r.w, "sidebar:0").Rect.X
	r.w.Update(tea.MouseClickMsg{X: x, Y: 3, Button: tea.MouseLeft})
	r.w.Update(tea.MouseMotionMsg{X: x + 100, Y: 3, Button: tea.MouseLeft})
	limit := r.w.plan.Panes["main"].W
	if got, applied := r.w.State().Resize["sidebar:0"], r.w.plan.Resize["sidebar:0"]; got != applied {
		t.Fatalf("after a drag past the limit: stored %d, applied %d; the store must be what was applied", got, applied)
	}
	r.w.Update(tea.MouseMotionMsg{X: x + 99, Y: 3, Button: tea.MouseLeft})
	if got := r.w.plan.Panes["main"].W; got != limit-1 {
		t.Fatalf("a drag back one cell from past the limit: main is %d cells wide, want %d", got, limit-1)
	}
}

func TestWindowResizeKeepsTheStoredLayout(t *testing.T) {
	r := newRig(t, 120, 40)
	r.w.Resize("sidebar:0", -10)
	want := maps.Clone(r.w.plan.Panes)
	stored := r.w.State()
	r.w.Update(tea.WindowSizeMsg{Width: 50, Height: 40})
	if r.w.plan.Panes["main"] == want["main"] {
		t.Fatal("the narrow window did not change the layout; the test proves nothing")
	}
	r.w.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if !maps.Equal(r.w.plan.Panes, want) || !maps.Equal(r.w.State().Resize, stored.Resize) {
		t.Fatalf("after shrinking and growing: panes %v, want %v; state %v, want %v", r.w.plan.Panes, want, r.w.State().Resize, stored.Resize)
	}
}

func TestResizeAtAWindowLimit(t *testing.T) {
	r := newRig(t, 120, 40)
	for range 50 {
		r.w.Update(press("alt+shift+right"))
	}
	stored := r.w.State().Resize["sidebar:0"]
	r.w.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if applied := r.w.plan.Resize["sidebar:0"]; applied == 0 || applied >= stored {
		t.Fatalf("at 100 cells: applied %d of a stored %d; the test needs a smaller, non-zero clamp", applied, stored)
	}
	// Toward the limit the window imposed: nothing moves, nothing is stored.
	r.w.Update(press("alt+shift+right"))
	if got := r.w.State().Resize["sidebar:0"]; got != stored {
		t.Fatalf("a press at the window's limit stored %d over %d", got, stored)
	}
	// Away from it: the separator moves at once, from where it is shown.
	narrow := r.w.plan.Panes["main"].W
	r.w.Update(press("alt+shift+left"))
	if got := r.w.plan.Panes["main"].W; got != narrow-1 {
		t.Fatalf("a press back from the window's limit: main is %d cells wide, want %d", got, narrow-1)
	}
}

func TestUnnamedSeparatorIsNotResized(t *testing.T) {
	root := layout.Split{Axis: layout.Horizontal, Gap: 1, Children: []layout.Child{
		{Node: layout.Pane{ID: "a"}}, {Node: layout.Pane{ID: "b"}},
	}}
	a, b := &fake{id: "a"}, &fake{id: "b"}
	w := New(root, map[layout.PaneID]Pane{"a": a, "b": b}, WithTheme(asciiTheme), WithChrome(Separators))
	w.Init()
	w.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	w.Render()
	sep := w.plan.Separators[0]
	if sep.Resizable || !strings.HasPrefix(sep.ID, "/") {
		t.Fatalf("separator %+v: want an unnamed, positional one", sep)
	}
	want := maps.Clone(w.plan.Panes)
	check := func(how string) {
		t.Helper()
		if w.State().Resize != nil || !maps.Equal(w.plan.Panes, want) {
			t.Fatalf("%s resized an unnamed separator: state %v, panes %v", how, w.State().Resize, w.plan.Panes)
		}
	}
	w.Resize(sep.ID, 3)
	check("Resize")
	w.Update(press("alt+shift+right"))
	check("the keyboard")
	w.Update(tea.MouseClickMsg{X: sep.Rect.X, Y: 3, Button: tea.MouseLeft})
	w.Update(tea.MouseMotionMsg{X: sep.Rect.X + 5, Y: 3, Button: tea.MouseLeft})
	w.Update(tea.MouseReleaseMsg{X: sep.Rect.X + 5, Y: 3, Button: tea.MouseLeft})
	check("a drag")
}

func TestKeyboardResizeSkipsAnUnnamedSeparator(t *testing.T) {
	// b's trailing separator is the unnamed inner split's; its leading one
	// is outer:0, which the keyboard moves instead.
	inner := layout.Split{Axis: layout.Horizontal, Gap: 1, Children: []layout.Child{
		{Node: layout.Pane{ID: "b"}}, {Node: layout.Pane{ID: "c"}},
	}}
	root := layout.Split{Name: "outer", Axis: layout.Horizontal, Gap: 1, Children: []layout.Child{
		{Node: layout.Pane{ID: "a"}}, {Node: inner},
	}}
	panes := map[layout.PaneID]Pane{"a": &fake{id: "a"}, "b": &fake{id: "b"}, "c": &fake{id: "c"}}
	w := New(root, panes, WithTheme(asciiTheme), WithChrome(None), WithFocus("b"))
	w.Init()
	w.Update(tea.WindowSizeMsg{Width: 90, Height: 20})
	w.Update(press("alt+shift+right"))
	if got := w.State().Resize; len(got) != 1 || got["outer:0"] != 1 {
		t.Fatalf("alt+shift+right on b: resize %v, want outer:0 moved by 1", got)
	}
}
