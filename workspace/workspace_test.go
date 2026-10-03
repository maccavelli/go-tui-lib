package workspace

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
)

// fake is a recording pane. It is a pointer, so the test keeps seeing it
// after the workspace stores what Update returns.
type fake struct {
	id         string
	title      string
	badge      string
	msgs       []tea.Msg
	views      int
	focused    bool
	noFocus    bool
	minW, minH int
	cursor     *tea.Cursor
	body       string
	esc        bool
	cmd        tea.Cmd
}

func (f *fake) Update(msg tea.Msg) (Pane, tea.Cmd) {
	f.msgs = append(f.msgs, msg)
	return f, f.cmd
}

func (f *fake) View(w, h int) string {
	f.views++
	if f.body != "" {
		return f.body
	}
	return f.id
}
func (f *fake) Title() string {
	if f.title == "" {
		return f.id
	}
	return f.title
}
func (f *fake) Badge() string           { return f.badge }
func (f *fake) Focus() tea.Cmd          { f.focused = true; return nil }
func (f *fake) Blur()                   { f.focused = false }
func (f *fake) Focusable() bool         { return !f.noFocus }
func (f *fake) Cursor() *tea.Cursor     { return f.cursor }
func (f *fake) MinSize() (int, int)     { return f.minW, f.minH }
func (f *fake) ConsumesEsc() bool       { return f.esc }
func (f *fake) Keys() []key.Binding     { return nil }
func (f *fake) got(want tea.Msg) bool   { return slices.Contains(f.msgs, want) }
func (f *fake) sizes() (out []SizeMsg)  { return filter[SizeMsg](f.msgs) }
func (f *fake) keys() []tea.KeyPressMsg { return filter[tea.KeyPressMsg](f.msgs) }
func (f *fake) reset()                  { f.msgs = nil }
func filter[T any](msgs []tea.Msg) []T {
	var out []T
	for _, m := range msgs {
		if v, ok := m.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// stable is a fake that is also a Changer.
type stable struct {
	*fake
	dirty bool
}

func (s *stable) Changed() bool { return s.dirty }
func (s *stable) Update(msg tea.Msg) (Pane, tea.Cmd) {
	s.msgs = append(s.msgs, msg)
	return s, nil
}

type rig struct {
	w                        *Workspace
	main, side, logs, footer *fake
}

var asciiTheme = theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII())

func newRig(t *testing.T, width, height int, opts ...Option) rig {
	t.Helper()
	r := rig{main: &fake{id: "main"}, side: &fake{id: "side"}, logs: &fake{id: "logs"}, footer: &fake{id: "footer", noFocus: true}}
	root := layout.SidebarRightBottom("main", "side", "logs", layout.Footer("footer", 1), layout.Gap(0))
	opts = append([]Option{WithTheme(asciiTheme)}, opts...)
	r.w = New(root, map[layout.PaneID]Pane{"main": r.main, "side": r.side, "logs": r.logs, "footer": r.footer}, opts...)
	r.w.Init()
	r.w.Update(tea.WindowSizeMsg{Width: width, Height: height})
	r.w.Render()
	return r
}

func press(s string) tea.KeyPressMsg {
	switch s {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "alt+z":
		return tea.KeyPressMsg{Code: 'z', Mod: tea.ModAlt}
	case "alt+2":
		return tea.KeyPressMsg{Code: '2', Mod: tea.ModAlt}
	case "alt+shift+left":
		return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt | tea.ModShift}
	case "alt+shift+right":
		return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt | tea.ModShift}
	case "alt+.":
		return tea.KeyPressMsg{Code: '.', Mod: tea.ModAlt}
	case "alt+,":
		return tea.KeyPressMsg{Code: ',', Mod: tea.ModAlt}
	}
	if len(s) != 1 {
		panic("press: no key named " + s)
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

func TestEachPaneIsToldItsSize(t *testing.T) {
	r := newRig(t, 120, 40)
	for _, f := range []*fake{r.main, r.side, r.logs} {
		in := r.w.content(layout.PaneID(f.id), r.w.plan.Panes[layout.PaneID(f.id)])
		if s := f.sizes(); len(s) == 0 || s[len(s)-1] != (SizeMsg{Width: in.W, Height: in.H}) {
			t.Errorf("%s: sizes %v, want the last to be %dx%d", f.id, s, in.W, in.H)
		}
	}
	n := len(r.main.sizes())
	r.w.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if len(r.main.sizes()) != n {
		t.Error("an unchanged size was sent again")
	}
}

func TestFocusCycles(t *testing.T) {
	r := newRig(t, 120, 40)
	var seen []layout.PaneID
	for range 4 {
		seen = append(seen, r.w.Focused())
		r.w.Update(press("alt+."))
	}
	if !slices.Equal(seen, []layout.PaneID{"main", "side", "logs", "main"}) {
		t.Fatalf("focus order %v (the footer is not focusable)", seen)
	}
	// Four moves left focus on side; one back is main.
	r.w.Update(press("alt+,"))
	if r.w.Focused() != "main" || !r.main.focused || r.side.focused {
		t.Fatalf("after alt+,: %s, main focused %v, side focused %v", r.w.Focused(), r.main.focused, r.side.focused)
	}
	r.w.Update(press("alt+,"))
	if r.w.Focused() != "logs" {
		t.Fatalf("alt+, from main wrapped to %s, want logs", r.w.Focused())
	}
	r.w.Update(press("alt+2"))
	if r.w.Focused() != "side" {
		t.Fatalf("alt+2 focused %s", r.w.Focused())
	}
	r.w.Toggle("side")
	if r.w.Focused() == "side" || !slices.Contains([]layout.PaneID{"main", "logs"}, r.w.Focused()) {
		t.Fatalf("focus stayed on a hidden pane: %s", r.w.Focused())
	}
}

func TestKeysReachOnlyTheFocusedPane(t *testing.T) {
	r := newRig(t, 120, 40)
	r.w.Update(press("x"))
	if len(r.main.keys()) != 1 || len(r.side.keys())+len(r.logs.keys()) != 0 {
		t.Fatalf("keys: main %v, side %v, logs %v", r.main.keys(), r.side.keys(), r.logs.keys())
	}
}

func TestCtrlCIsNeverConsumed(t *testing.T) {
	for _, b := range DefaultKeyMap().Bindings() {
		if slices.Contains(b.Keys(), "ctrl+c") {
			t.Fatalf("a default binding uses ctrl+c: %v", b.Keys())
		}
	}
	r := newRig(t, 120, 40)
	st, focus := r.w.State(), r.w.Focused()
	r.w.Update(press("ctrl+c"))
	if r.w.Focused() != focus || len(r.w.State().Hidden) != len(st.Hidden) || r.w.State().Zoom != st.Zoom {
		t.Fatal("ctrl+c changed the workspace")
	}
	if k := r.main.keys(); len(k) != 1 || k[0].String() != "ctrl+c" {
		t.Fatalf("ctrl+c did not reach the focused pane: %v", k)
	}
}

func TestModalOverlayTrapsKeys(t *testing.T) {
	r := newRig(t, 120, 40)
	dlg := &fake{id: "dialog"}
	r.w.Push(Overlay{ID: "dialog", Pane: dlg, Width: 40, Height: 8, Modal: true})
	r.w.Update(press("y"))
	r.w.Update(press("alt+."))
	if len(dlg.keys()) != 2 || len(r.main.keys()) != 0 || r.w.Focused() != "main" {
		t.Fatalf("dialog keys %v, main keys %v, focus %s", dlg.keys(), r.main.keys(), r.w.Focused())
	}
	if c := dlg.sizes(); len(c) != 1 || c[0] != (SizeMsg{Width: 38, Height: 6}) {
		t.Fatalf("overlay size %v", c)
	}
	r.w.Update(press("esc"))
	if len(r.w.Overlays()) != 0 {
		t.Fatal("esc did not close the overlay")
	}
	keeper := &fake{id: "picker", esc: true}
	r.w.Push(Overlay{ID: "picker", Pane: keeper, Width: 30, Height: 6, Modal: true})
	r.w.Update(press("esc"))
	if len(r.w.Overlays()) != 1 || len(keeper.keys()) != 1 {
		t.Fatal("an overlay that consumes esc was closed")
	}
}

func TestNonModalOverlayLeavesKeysToThePane(t *testing.T) {
	r := newRig(t, 120, 40)
	pop := &fake{id: "completion"}
	r.w.Push(Overlay{ID: "completion", Pane: pop, Anchor: Anchor{Kind: BelowCursor}, Width: 20, Height: 5})
	r.w.Update(press("a"))
	if len(r.main.keys()) != 1 || len(pop.keys()) != 0 {
		t.Fatalf("main %v, pop-up %v", r.main.keys(), pop.keys())
	}
}

// at returns a screen cell inside pane id's content area.
func (r rig) at(id layout.PaneID, dx, dy int) (int, int) {
	in := r.w.content(id, r.w.plan.Panes[id])
	return in.X + dx, in.Y + dy
}

func TestMouseRouting(t *testing.T) {
	r := newRig(t, 120, 40)
	x, y := r.at("side", 2, 1)
	r.w.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if r.w.Focused() != "side" {
		t.Fatalf("a click focused %s, want side", r.w.Focused())
	}
	if c := filter[tea.MouseClickMsg](r.side.msgs); len(c) != 1 || c[0].X != 2 || c[0].Y != 1 {
		t.Fatalf("side got clicks %v, want one at (2,1)", c)
	}
	x, y = r.at("logs", 0, 0)
	r.w.Update(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown})
	if len(filter[tea.MouseWheelMsg](r.logs.msgs)) != 1 || len(filter[tea.MouseWheelMsg](r.side.msgs)) != 0 {
		t.Fatal("the wheel did not reach the pane under the pointer")
	}
	// A click on a border focuses, and is not forwarded.
	b := r.w.plan.Panes["main"]
	r.w.Update(tea.MouseClickMsg{X: b.X, Y: b.Y, Button: tea.MouseLeft})
	if r.w.Focused() != "main" || len(filter[tea.MouseClickMsg](r.main.msgs)) != 0 {
		t.Fatalf("a border click: focus %s, main clicks %v", r.w.Focused(), filter[tea.MouseClickMsg](r.main.msgs))
	}
	// A modal overlay blocks the mouse beneath it.
	r.w.Push(Overlay{ID: "dialog", Pane: &fake{id: "dialog"}, Width: 20, Height: 5, Modal: true})
	r.w.Render()
	x, y = r.at("side", 2, 1)
	r.w.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if r.w.Focused() != "main" {
		t.Fatal("a click reached a pane under a modal overlay")
	}
	off := newRig(t, 120, 40, WithMouse(false))
	x, y = off.at("side", 2, 1)
	off.w.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if off.w.Focused() != "main" {
		t.Fatal("WithMouse(false) still routed a click")
	}
}

func TestSeparatorDrag(t *testing.T) {
	r := newRig(t, 120, 30, WithChrome(Separators))
	r.w.SetLayout(layout.SidebarRight("main", "side"))
	r.w.Render()
	var sep layout.Separator
	for _, s := range r.w.plan.Separators {
		if s.ID == "sidebar:0" {
			sep = s
		}
	}
	if sep.ID == "" {
		t.Fatalf("no sidebar separator in %+v", r.w.plan.Separators)
	}
	before := r.w.plan.Panes["side"].W
	r.w.Update(tea.MouseClickMsg{X: sep.Rect.X, Y: 3, Button: tea.MouseLeft})
	r.w.Update(tea.MouseMotionMsg{X: sep.Rect.X + 5, Y: 3, Button: tea.MouseLeft})
	r.w.Update(tea.MouseReleaseMsg{X: sep.Rect.X + 5, Y: 3, Button: tea.MouseLeft})
	if r.w.State().Resize["sidebar:0"] != 5 || r.w.plan.Panes["side"].W != before-5 {
		t.Fatalf("after a 5-cell drag: resize %v, side %d → %d", r.w.State().Resize, before, r.w.plan.Panes["side"].W)
	}
	r.w.Update(tea.MouseMotionMsg{X: sep.Rect.X + 9, Y: 3})
	if r.w.State().Resize["sidebar:0"] != 5 {
		t.Fatal("motion after the release still resized")
	}
}

func TestResizeByKey(t *testing.T) {
	r := newRig(t, 120, 40)
	r.w.Update(press("alt+shift+left"))
	if r.w.State().Resize["sidebar:0"] != -1 {
		t.Fatalf("alt+shift+left on main: resize %v", r.w.State().Resize)
	}
}

func TestZoomAndToggle(t *testing.T) {
	r := newRig(t, 120, 40)
	r.w.Update(press("alt+z"))
	if p := r.w.Plan(); len(p.Panes) != 1 || p.Panes["main"] != (layout.Rect{W: 120, H: 40}) {
		t.Fatalf("zoom: %+v", p.Panes)
	}
	r.w.Update(press("alt+z"))
	if len(r.w.Plan().Panes) != 4 {
		t.Fatal("zoom did not restore")
	}
	r.w.Toggle("logs")
	if _, ok := r.w.Plan().Panes["logs"]; ok {
		t.Fatal("toggle did not hide logs")
	}
	r.w.Toggle("logs")
	if _, ok := r.w.Plan().Panes["logs"]; !ok {
		t.Fatal("toggle did not show logs")
	}
}

func TestCursorIsOffsetByThePane(t *testing.T) {
	r := newRig(t, 120, 40)
	r.main.cursor = tea.NewCursor(2, 1)
	in := r.w.content("main", r.w.plan.Panes["main"])
	if c := r.w.Cursor(); c == nil || c.X != in.X+2 || c.Y != in.Y+1 {
		t.Fatalf("cursor %+v, want (%d,%d)", c, in.X+2, in.Y+1)
	}
	r.main.cursor = tea.NewCursor(in.W, 0)
	if r.w.Cursor() != nil {
		t.Fatal("a cursor outside the pane was shown")
	}
	r.main.cursor = nil
	if r.w.Cursor() != nil {
		t.Fatal("a nil cursor was shown")
	}
	r.w.Update(press("alt+."))
	r.side.cursor = tea.NewCursor(0, 0)
	r.w.Toggle("side")
	if c := r.w.Cursor(); c != nil && r.w.Focused() == "side" {
		t.Fatal("a hidden pane's cursor was shown")
	}
}

func TestChangerSkipsTheView(t *testing.T) {
	s := &stable{fake: &fake{id: "main"}}
	w := New(layout.SidebarRight("main", "side"), map[layout.PaneID]Pane{"main": s, "side": &fake{id: "side"}}, WithTheme(asciiTheme))
	w.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	w.Render()
	w.Render()
	if s.views != 1 {
		t.Fatalf("an unchanged Changer was viewed %d times", s.views)
	}
	s.dirty = true
	w.Render()
	if s.views != 2 {
		t.Fatalf("a changed Changer was viewed %d times", s.views)
	}
	s.dirty = false
	w.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	w.Render()
	if s.views != 3 {
		t.Fatalf("after a resize, viewed %d times", s.views)
	}
}

func TestPanesAreClipped(t *testing.T) {
	r := newRig(t, 120, 40, WithChrome(None))
	r.main.body = strings.Repeat(strings.Repeat("M", 300)+"\n", 100)
	frame := strings.Split(r.w.Render(), "\n")
	if len(frame) > 40 {
		t.Fatalf("%d rows", len(frame))
	}
	sx := r.w.plan.Panes["side"].X
	for i, line := range frame[:20] {
		if w := ansi.StringWidth(line); w > 120 {
			t.Fatalf("row %d is %d cells", i, w)
		}
		if cell := ansi.Strip(line); len(cell) > sx && cell[sx] == 'M' {
			t.Fatalf("row %d: main painted into side: %q", i, cell)
		}
	}
	// Panes are drawn left to right, so a pane's overflow would sit under
	// its neighbour. An overlay is drawn above everything: its overflow
	// would show.
	wide := &fake{id: "wide", body: strings.Repeat("O", 200) + "\n" + strings.Repeat("O", 200)}
	r.w.Push(Overlay{ID: "wide", Pane: wide, Width: 20, Height: 5, Modal: true})
	box := r.w.overlayRect(r.w.overlays[0])
	frame = strings.Split(ansi.Strip(r.w.Render()), "\n")
	for y := box.Y; y < box.Y+box.H; y++ {
		if row := []rune(frame[y]); len(row) > box.X+box.W && strings.ContainsRune(string(row[box.X+box.W:]), 'O') {
			t.Fatalf("row %d: the overlay painted past its box: %q", y, frame[y])
		}
	}
}

func TestFocusIsVisibleWithoutColour(t *testing.T) {
	r := newRig(t, 120, 40)
	frame := r.w.Render()
	if !strings.Contains(frame, "> main") || strings.Contains(frame, "> side") {
		t.Fatalf("the focused title is not marked:\n%s", frame)
	}
	if !strings.Contains(frame, "\x1b[1m") {
		t.Fatal("the focused title is not bold under the ASCII profile")
	}
	if strings.Contains(frame, "38;") {
		t.Fatal("the ASCII profile drew colour")
	}
}

func TestSizerRaisesTheMinimum(t *testing.T) {
	r := newRig(t, 120, 40)
	r.side.minW = 60
	r.w.SetPane("side", r.side)
	if got := r.w.plan.Panes["side"].W; got < 62 {
		t.Fatalf("side is %d wide, want at least 62 (60 + borders)", got)
	}
}

func TestSendBroadcastAndTo(t *testing.T) {
	r := newRig(t, 120, 40)
	type ping struct{}
	r.w.Update(ping{})
	if !r.main.got(ping{}) || !r.side.got(ping{}) || !r.footer.got(ping{}) {
		t.Fatal("an unknown message was not broadcast")
	}
	r.main.reset()
	r.side.reset()
	type poke struct{}
	r.w.Update(To("side", poke{}))
	if r.main.got(poke{}) || !r.side.got(poke{}) {
		t.Fatal("To did not target one pane")
	}
	r.w.Send("logs", poke{})
	if !r.logs.got(poke{}) {
		t.Fatal("Send did not reach logs")
	}
}

type tick struct{ n int }

// TestCommandsAreIndependent runs a broadcast's commands on goroutines, as
// Bubble Tea does, and feeds their messages back to Update one at a time,
// as Bubble Tea does too. Under -race, a command that touched the
// workspace or a pane would fail it.
func TestCommandsAreIndependent(t *testing.T) {
	r := newRig(t, 120, 40)
	for i, f := range []*fake{r.main, r.side, r.logs} {
		f.cmd = func() tea.Msg { return tick{i} }
	}
	for _, m := range run(r.w.Broadcast(struct{}{})) {
		r.w.Update(m)
		_ = r.w.Render()
	}
	if len(filter[tick](r.side.msgs)) != 3 {
		t.Fatalf("side saw %d ticks", len(filter[tick](r.side.msgs)))
	}
}

// run executes cmd, and every command in a batch, each on its own
// goroutine.
func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	b, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var mu sync.Mutex
	var out []tea.Msg
	var wg sync.WaitGroup
	for _, c := range b {
		wg.Go(func() {
			ms := run(c)
			mu.Lock()
			out = append(out, ms...)
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
}

func BenchmarkRender(b *testing.B) {
	for _, withChanger := range []bool{false, true} {
		name := "views"
		if withChanger {
			name = "changer"
		}
		b.Run(name, func(b *testing.B) {
			panes := map[layout.PaneID]Pane{}
			for _, id := range []string{"main", "side", "logs", "footer"} {
				f := &fake{id: id, body: strings.Repeat(id+" line\n", 60)}
				if withChanger {
					panes[layout.PaneID(id)] = &stable{fake: f}
				} else {
					panes[layout.PaneID(id)] = f
				}
			}
			w := New(layout.SidebarRightBottom("main", "side", "logs", layout.Footer("footer", 1), layout.Gap(0)), panes, WithTheme(asciiTheme))
			w.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
			for b.Loop() {
				_ = w.Render()
			}
		})
	}
}
