package workspace

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// keyed is a pane with its own bindings, one of them disabled.
type keyed struct {
	*fake
	keys []key.Binding
}

func (k *keyed) Keys() []key.Binding { return k.keys }
func (k *keyed) Update(msg tea.Msg) (Pane, tea.Cmd) {
	k.msgs = append(k.msgs, msg)
	return k, nil
}

func bind(keys, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys), key.WithHelp(keys, desc))
}

func disabled(keys, desc string) key.Binding {
	b := bind(keys, desc)
	b.SetEnabled(false)
	return b
}

func TestHelpListsWhoeverHasTheKeyboardFirst(t *testing.T) {
	main := &keyed{fake: &fake{id: "main"}, keys: []key.Binding{bind("ctrl+s", "save"), disabled("ctrl+q", "secret")}}
	w := New(layout.SidebarRight("main", "side"), map[layout.PaneID]Pane{"main": main, "side": &fake{id: "side"}}, WithTheme(asciiTheme))
	w.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	view := ansi.Strip(help.New().View(w))
	save, next := strings.Index(view, "ctrl+s save"), strings.Index(view, "alt+. next pane")
	if save < 0 || next < 0 || save > next {
		t.Fatalf("short help %q: want the pane's ctrl+s before the workspace's alt+.", view)
	}
	if strings.Contains(view, "secret") {
		t.Fatalf("short help %q lists a disabled binding", view)
	}
	full := w.FullHelp()
	if len(full) != 4 || full[0][0].Help().Desc != "save" || full[3][0].Help().Desc != "close" {
		t.Fatalf("full help columns %v: want the pane's keys, focus, layout and overlays", full)
	}
	// With a modal overlay open, its keys come first, not the pane's.
	dlg := &keyed{fake: &fake{id: "dlg"}, keys: []key.Binding{bind("y", "allow")}}
	w.Push(Overlay{ID: "dlg", Pane: dlg, Width: 30, Height: 6, Modal: true})
	view = ansi.Strip(help.New().View(w))
	if !strings.HasPrefix(view, "y allow") || strings.Contains(view, "save") {
		t.Fatalf("short help with a modal overlay %q: want the overlay's keys, not the pane's", view)
	}
	// A pane with no bindings leaves no empty column.
	w.Pop()
	w.Focus("side")
	if full := w.FullHelp(); len(full) != 3 {
		t.Fatalf("full help for a pane with no keys has %d columns, want 3", len(full))
	}
}

func TestViewCarriesTheFrameAndCursorOnly(t *testing.T) {
	r := newRig(t, 120, 40)
	r.main.cursor = tea.NewCursor(2, 1)
	v := r.w.View()
	if v.Content != r.w.Render() || v.Cursor == nil || *v.Cursor != *r.w.Cursor() {
		t.Fatalf("View: content or cursor differs from Render and Cursor")
	}
	if !reflect.DeepEqual(v, tea.View{Content: v.Content, Cursor: v.Cursor}) {
		t.Fatalf("View set a field the program owns: %+v", v)
	}
}

// chromeFrame is a sidebar layout with Separators on the main pane only,
// None everywhere else, and a footer.
func chromeFrame(c tuitest.Case) string {
	panes := map[layout.PaneID]Pane{
		"main":   &fake{id: "main", title: "Session", body: "user: hello\nagent: hi there"},
		"side":   &fake{id: "side", body: "tokens 1200\ncost $0.01"},
		"footer": &fake{id: "footer", noFocus: true, body: "model x | ctx 12%"},
	}
	root := layout.SidebarRight("main", "side", layout.Footer("footer", 1), layout.NoResponsive())
	w := New(root, panes, WithTheme(caseTheme(c)), WithChrome(None), WithPaneChrome("main", Separators))
	w.Update(tea.WindowSizeMsg{Width: c.Width, Height: 12})
	return w.Render()
}

// junctionFrame lays panes out so that its separators meet in a cross and
// in each of the four tees: the column separators of rows 1 and 2 line up
// (a cross between them, a tee up below row 2), row 4's starts below row 3
// (a tee down), and a split inside row 1's right column and row 2's left
// column meets the column separator (a tee right and a tee left).
func junctionFrame(c tuitest.Case) string {
	pane := func(id string) layout.Child {
		return layout.Child{Node: layout.Pane{ID: layout.PaneID(id)}, Size: layout.Fill(1)}
	}
	col := func(n layout.Node, size layout.Size) layout.Child { return layout.Child{Node: n, Size: size} }
	stack := func(a, b string) layout.Node {
		return layout.Split{Axis: layout.Vertical, Gap: 1, Children: []layout.Child{pane(a), pane(b)}}
	}
	row := func(h int, children ...layout.Child) layout.Child {
		return layout.Child{Node: layout.Split{Axis: layout.Horizontal, Gap: 1, Children: children}, Size: layout.Fixed(h)}
	}
	root := layout.Split{Axis: layout.Vertical, Gap: 1, Children: []layout.Child{
		row(5, col(layout.Pane{ID: "a"}, layout.Fixed(30)), col(stack("b1", "b2"), layout.Fill(1))),
		row(5, col(stack("c1", "c2"), layout.Fixed(30)), col(layout.Pane{ID: "d"}, layout.Fill(1))),
		{Node: layout.Pane{ID: "e"}, Size: layout.Fixed(2)},
		row(2, col(layout.Pane{ID: "f"}, layout.Fixed(45)), col(layout.Pane{ID: "g"}, layout.Fill(1))),
	}}
	panes := map[layout.PaneID]Pane{}
	for _, id := range []string{"a", "b1", "b2", "c1", "c2", "d", "e", "f", "g"} {
		panes[layout.PaneID(id)] = &fake{id: id}
	}
	w := New(root, panes, WithTheme(caseTheme(c)), WithChrome(Separators))
	w.Update(tea.WindowSizeMsg{Width: c.Width, Height: 18})
	return w.Render()
}

func TestChromeGolden(t *testing.T) {
	m := tuitest.Matrix{Widths: []int{80, 120}}
	tuitest.Golden(t, "chrome-mixed", m, chromeFrame)
	tuitest.Golden(t, "chrome-junctions", m, junctionFrame)
}

func TestJunctionFrameHasEveryJunction(t *testing.T) {
	frame := junctionFrame(tuitest.Case{UTF8: true, Width: 80})
	for _, g := range []string{"┼", "┬", "┴", "├", "┤"} {
		if !strings.Contains(frame, g) {
			t.Errorf("the junction frame has no %q:\n%s", g, frame)
		}
	}
}

func TestSeparatorFollowsPaneChrome(t *testing.T) {
	// With None everywhere, no separator is drawn; with Separators on main
	// only, the one beside main is.
	frame := func(opts ...Option) string {
		w := New(layout.SidebarRight("main", "side", layout.NoResponsive()),
			map[layout.PaneID]Pane{"main": &fake{id: "main"}, "side": &fake{id: "side"}},
			append([]Option{WithTheme(asciiTheme), WithChrome(None)}, opts...)...)
		w.Update(tea.WindowSizeMsg{Width: 60, Height: 6})
		return w.Render()
	}
	if strings.Contains(frame(), "|") {
		t.Fatal("a separator between two None panes was drawn")
	}
	if !strings.Contains(frame(WithPaneChrome("main", Separators)), "|") {
		t.Fatal("the separator beside a Separators pane was not drawn")
	}
}
