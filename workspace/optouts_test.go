package workspace

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// The mouse, the bare overlay and the focus fallback
// (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md Step 4;
// 0024-MADR D6, A1).

// dragged reports how far a 5-cell drag of the sidebar's separator moved
// it, and whether a click on side focused it, under opts.
func dragged(t *testing.T, opts ...Option) (resize int, focused bool, clicks int) {
	t.Helper()
	r := newRig(t, 120, 30, append([]Option{WithChrome(Separators)}, opts...)...)
	r.w.SetLayout(layout.SidebarRight("main", "side"))
	r.w.Render()
	var sep layout.Separator
	for _, s := range r.w.plan.Separators {
		if s.ID == "sidebar:0" {
			sep = s
		}
	}
	r.w.Update(tea.MouseClickMsg{X: sep.Rect.X, Y: 3, Button: tea.MouseLeft})
	r.w.Update(tea.MouseMotionMsg{X: sep.Rect.X + 5, Y: 3, Button: tea.MouseLeft})
	r.w.Update(tea.MouseReleaseMsg{X: sep.Rect.X + 5, Y: 3, Button: tea.MouseLeft})
	r.w.Render()
	x, y := r.at("side", 1, 1)
	r.w.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	r.w.Update(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown})
	if len(filter[tea.MouseWheelMsg](r.side.msgs)) != 1 {
		t.Errorf("the wheel did not reach side under %d options", len(opts))
	}
	return r.w.State().Resize["sidebar:0"], r.w.Focused() == "side", len(filter[tea.MouseClickMsg](r.side.msgs))
}

func TestMouseOptionsEachDropOneBehaviour(t *testing.T) {
	for _, c := range []struct {
		name    string
		opts    []Option
		resize  int
		focused bool
	}{
		{"default", nil, 5, true},
		{"WithoutMouseFocus", []Option{WithoutMouseFocus()}, 5, false},
		{"WithoutMouseResize", []Option{WithoutMouseResize()}, 0, true},
		{"both", []Option{WithoutMouseFocus(), WithoutMouseResize()}, 0, false},
	} {
		resize, focused, clicks := dragged(t, c.opts...)
		if resize != c.resize || focused != c.focused || clicks != 1 {
			t.Errorf("%s: resize %d, focused %v, side's clicks %d; want %d, %v, 1", c.name, resize, focused, clicks, c.resize, c.focused)
		}
	}
}

func TestFocusFallback(t *testing.T) {
	var lost layout.PaneID
	var ring []layout.PaneID
	for _, c := range []struct {
		name string
		opts []Option
		want layout.PaneID
	}{
		{"default", nil, "main"},
		{"to logs", []Option{WithFocusFallback(FocusFallbackFunc(func(l layout.PaneID, r []layout.PaneID) layout.PaneID {
			lost, ring = l, r
			return "logs"
		}))}, "logs"},
		{"outside the ring", []Option{WithFocusFallback(FocusFallbackFunc(func(layout.PaneID, []layout.PaneID) layout.PaneID {
			return "footer"
		}))}, "main"},
	} {
		r := newRig(t, 120, 40, c.opts...)
		r.w.Focus("side")
		r.w.Toggle("side")
		if got := r.w.Focused(); got != c.want {
			t.Errorf("%s: focus %s after side hid, want %s", c.name, got, c.want)
		}
	}
	if lost != "side" || !slices.Equal(ring, []layout.PaneID{"main", "logs"}) {
		t.Errorf("the fallback was given %q and %v; want side and [main logs]", lost, ring)
	}
}

// bare is a workspace of 40 by 12 with a bare overlay of 10 by 3 on it.
func bare(t *testing.T, c tuitest.Case, o Overlay) (*Workspace, *fake) {
	t.Helper()
	f := &fake{id: "pop"}
	o.ID, o.Pane = "pop", f
	w := New(layout.SidebarRight("main", "side"), map[layout.PaneID]Pane{"main": &fake{id: "main"}, "side": &fake{id: "side"}}, WithTheme(caseTheme(c)))
	w.Update(tea.WindowSizeMsg{Width: c.Width, Height: 12})
	w.Push(o)
	w.Render()
	return w, f
}

func TestBareOverlay(t *testing.T) {
	c := tuitest.Case{UTF8: true, Width: 40}
	w, f := bare(t, c, Overlay{Width: 10, Height: 3, Bare: true, Modal: true})
	if s := f.sizes(); len(s) == 0 || s[len(s)-1] != (SizeMsg{Width: 10, Height: 3}) {
		t.Errorf("a bare overlay's sizes %v; want the last 10x3", s)
	}
	framed, ff := bare(t, c, Overlay{Width: 10, Height: 3, Modal: true})
	if s := ff.sizes(); len(s) == 0 || s[len(s)-1] != (SizeMsg{Width: 8, Height: 1}) {
		t.Errorf("a framed overlay's sizes %v; want the last 8x1", s)
	}
	if w.Render() == framed.Render() {
		t.Error("the bare overlay drew as the framed one")
	}
	r := w.overlayRect(w.overlays[0])
	f.cursor = tea.NewCursor(0, 0)
	if cur := w.Cursor(); cur == nil || cur.X != r.X || cur.Y != r.Y {
		t.Errorf("the cursor %+v; want the overlay's corner (%d,%d)", cur, r.X, r.Y)
	}
	w.Update(tea.MouseClickMsg{X: r.X, Y: r.Y, Button: tea.MouseLeft})
	if c := filter[tea.MouseClickMsg](f.msgs); len(c) != 1 || c[0].X != 0 || c[0].Y != 0 {
		t.Errorf("a click on the corner reached the overlay as %v; want one at (0,0)", c)
	}
	tiny, tf := bare(t, c, Overlay{Width: 1, Height: 1, Bare: true})
	if s := tf.sizes(); len(s) == 0 || s[len(s)-1] != (SizeMsg{Width: 1, Height: 1}) || tiny.overlayRect(tiny.overlays[0]).W != 1 {
		t.Errorf("a 1x1 bare overlay's sizes %v", s)
	}
}

func TestBareOverlayGolden(t *testing.T) {
	tuitest.Golden(t, "overlay-bare", tuitest.Matrix{Widths: []int{40, 60}}, func(c tuitest.Case) string {
		w, _ := bare(t, c, Overlay{Width: 10, Height: 3, Bare: true, Modal: true})
		return w.Render()
	})
}
