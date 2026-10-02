package workspace

import (
	"slices"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/layout"
)

// valuePane is a pane with value semantics: its Update returns a changed
// copy, as bubbles models do. Focus reaches it only as a message, through
// the value Update returns. log is shared, so the test sees every copy's
// messages in order.
type valuePane struct {
	id      string
	log     *[]string
	focused bool
}

func (v valuePane) Update(msg tea.Msg) (Pane, tea.Cmd) {
	switch msg.(type) {
	case PaneFocusMsg:
		v.focused = true
		*v.log = append(*v.log, v.id+" focus")
	case PaneBlurMsg:
		v.focused = false
		*v.log = append(*v.log, v.id+" blur")
	}
	return v, nil
}

func (v valuePane) View(int, int) string { return v.id }

// focusedValue reports what the workspace's stored copy of pane id believes.
func focusedValue(t *testing.T, w *Workspace, id layout.PaneID) bool {
	t.Helper()
	v, ok := w.panes[id].(valuePane)
	if !ok {
		t.Fatalf("pane %s is %T", id, w.panes[id])
	}
	return v.focused
}

func TestFocusReachesValuePanes(t *testing.T) {
	var log []string
	panes := map[layout.PaneID]Pane{}
	for _, id := range []string{"a", "b", "c"} {
		panes[layout.PaneID(id)] = valuePane{id: id, log: &log}
	}
	w := New(layout.SidebarRightBottom("a", "b", "c", layout.Gap(0)), panes, WithTheme(asciiTheme))
	w.Init()
	w.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	w.FocusNext()
	w.Render()
	in := w.content("c", w.plan.Panes["c"])
	w.Update(tea.MouseClickMsg{X: in.X + 1, Y: in.Y, Button: tea.MouseLeft})
	w.Toggle("c")
	want := []string{"a focus", "a blur", "b focus", "b blur", "c focus", "c blur", "a focus"}
	if !slices.Equal(log, want) {
		t.Fatalf("focus messages %v, want %v", log, want)
	}
	if !focusedValue(t, w, "a") || focusedValue(t, w, "b") || focusedValue(t, w, "c") {
		t.Fatal("the stored panes do not hold the focus their messages gave them")
	}
}

func TestModalOverlayMovesFocus(t *testing.T) {
	var log []string
	w := New(layout.Pane{ID: "a"}, map[layout.PaneID]Pane{"a": valuePane{id: "a", log: &log}}, WithTheme(asciiTheme))
	w.Init()
	log = nil
	w.Push(Overlay{ID: "pop", Pane: valuePane{id: "pop", log: &log}, Width: 10, Height: 4})
	if len(log) != 0 {
		t.Fatalf("a non-modal Push sent %v", log)
	}
	w.Pop()
	if len(log) != 0 {
		t.Fatalf("closing a non-modal overlay sent %v", log)
	}
	w.Push(Overlay{ID: "dlg", Pane: valuePane{id: "dlg", log: &log}, Width: 20, Height: 6, Modal: true})
	w.Push(Overlay{ID: "dlg2", Pane: valuePane{id: "dlg2", log: &log}, Width: 20, Height: 6, Modal: true})
	w.Pop()
	w.Pop()
	want := []string{"a blur", "dlg focus", "dlg blur", "dlg2 focus", "dlg2 blur", "dlg focus", "dlg blur", "a focus"}
	if !slices.Equal(log, want) {
		t.Fatalf("focus messages %v, want %v", log, want)
	}
	// Focus moved under a modal overlay reaches the pane only when the
	// overlay closes.
	panes := map[layout.PaneID]Pane{"a": valuePane{id: "a", log: &log}, "b": valuePane{id: "b", log: &log}}
	w = New(layout.SidebarRight("a", "b"), panes, WithTheme(asciiTheme))
	w.Init()
	w.Push(Overlay{ID: "dlg", Pane: valuePane{id: "dlg", log: &log}, Width: 20, Height: 6, Modal: true})
	log = nil
	w.Focus("b")
	if len(log) != 0 {
		t.Fatalf("Focus under a modal overlay sent %v", log)
	}
	w.Pop()
	if want := []string{"dlg blur", "b focus"}; !slices.Equal(log, want) {
		t.Fatalf("after Pop: %v, want %v", log, want)
	}
}

func TestFocuserIsCalledBeforeTheMessage(t *testing.T) {
	r := newRig(t, 120, 40)
	if !r.main.focused || !r.main.got(PaneFocusMsg{}) {
		t.Fatalf("after Init: Focuser %v, message %v", r.main.focused, r.main.got(PaneFocusMsg{}))
	}
	r.w.FocusNext()
	if r.main.focused || !r.main.got(PaneBlurMsg{}) || !r.side.focused || !r.side.got(PaneFocusMsg{}) {
		t.Fatal("a focus change did not reach both the Focuser and the message")
	}
}

func TestWrapFocusesATextinput(t *testing.T) {
	// The terminal cursor, not bubbles' default virtual one, which draws
	// itself in the view and gives no Cursor.
	ti := textinput.New()
	ti.SetVirtualCursor(false)
	w := New(layout.Pane{ID: "in"}, map[layout.PaneID]Pane{"in": Wrap(ti)}, WithTheme(asciiTheme))
	w.Init()
	w.Update(tea.WindowSizeMsg{Width: 40, Height: 5})
	w.Update(press("h"))
	w.Update(press("i"))
	m, ok := w.panes["in"].(*Model[textinput.Model])
	if !ok {
		t.Fatalf("pane is %T", w.panes["in"])
	}
	if !m.M.Focused() || m.M.Value() != "hi" {
		t.Fatalf("textinput focused %v, value %q; want focused with \"hi\"", m.M.Focused(), m.M.Value())
	}
	in := w.content("in", w.plan.Panes["in"])
	if m.M.Width() != in.W {
		t.Fatalf("textinput width %d, want the content width %d", m.M.Width(), in.W)
	}
	own := m.M.Cursor()
	got := w.Cursor()
	if own == nil || got == nil || got.X != in.X+own.X || got.Y != in.Y+own.Y {
		t.Fatalf("cursor %+v, want the textinput's %+v offset by (%d,%d)", got, own, in.X, in.Y)
	}
	w.Update(tea.WindowSizeMsg{Width: 40, Height: 5}) // no change, no blur
	w.Push(Overlay{ID: "dlg", Pane: &fake{id: "dlg"}, Width: 20, Height: 4, Modal: true})
	if m := w.panes["in"].(*Model[textinput.Model]); m.M.Focused() {
		t.Fatal("a modal overlay did not blur the textinput")
	}
}

func TestWrapSizesAViewport(t *testing.T) {
	w := New(layout.Pane{ID: "v"}, map[layout.PaneID]Pane{"v": Wrap(viewport.New())}, WithTheme(asciiTheme))
	w.Init()
	w.Update(tea.WindowSizeMsg{Width: 50, Height: 12})
	m := w.panes["v"].(*Model[viewport.Model])
	in := w.content("v", w.plan.Panes["v"])
	if m.M.Width() != in.W || m.M.Height() != in.H {
		t.Fatalf("viewport %dx%d, want %dx%d", m.M.Width(), m.M.Height(), in.W, in.H)
	}
}

func TestWrapOptionsOverride(t *testing.T) {
	var sized [2]int
	var events []string
	cur := tea.NewCursor(1, 0)
	keys := []key.Binding{key.NewBinding(key.WithKeys("x"))}
	m := Wrap(textinput.New(),
		OnSize(func(_ *textinput.Model, w, h int) { sized = [2]int{w, h} }),
		OnFocus(func(*textinput.Model) tea.Cmd { events = append(events, "focus"); return nil }),
		OnBlur(func(*textinput.Model) { events = append(events, "blur") }),
		WithCursor(func(textinput.Model) *tea.Cursor { return cur }),
		WithKeys(func(textinput.Model) []key.Binding { return keys }),
	)
	m.Update(SizeMsg{Width: 7, Height: 3})
	m.Update(PaneFocusMsg{})
	m.Update(PaneBlurMsg{})
	if sized != [2]int{7, 3} || !slices.Equal(events, []string{"focus", "blur"}) {
		t.Fatalf("OnSize %v, events %v", sized, events)
	}
	if m.M.Focused() || m.M.Width() == 7 {
		t.Fatal("an option did not replace the found method")
	}
	if m.Cursor() != cur || len(m.Keys()) != 1 {
		t.Fatal("WithCursor or WithKeys was not used")
	}
}
