package workspace

import (
	"maps"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// RenderPlain and Help
// (docs/decisions/0014-PLAN-component-native-forms.md Step 7).

// plainBubble is a bubbles-shaped model with its own plain view.
type plainBubble struct{ text string }

func (b plainBubble) Update(tea.Msg) (plainBubble, tea.Cmd) { return b, nil }
func (b plainBubble) View() string                          { return "\x1b[1m" + b.text + "\x1b[m   " }
func (b plainBubble) PlainView(width int) string {
	return b.text + " (plain, " + strings.Repeat("w", min(width, 3)) + ")"
}

// styledBubble has no plain view of its own.
type styledBubble struct{}

func (styledBubble) Update(tea.Msg) (styledBubble, tea.Cmd) { return styledBubble{}, nil }
func (styledBubble) View() string                           { return "\x1b[31mred\x1b[m line\n\n\n" }

// plainWorkspace is the golden frames' workspace, with a styled pane, a
// wrapped bubble with a PlainView, a footer that refuses focus, and an
// overlay open.
func plainWorkspace(th theme.Theme, opts ...Option) *Workspace {
	panes := map[layout.PaneID]Pane{
		"main":   &fake{id: "main", title: "Session", body: th.Styles.Title.Render("user: hello") + "\nagent: hi there   \n\n"},
		"side":   &fake{id: "side", title: "Metrics\nand more", badge: "3", body: "tokens 1200\ncost $0.01"},
		"logs":   Wrap(plainBubble{text: "INFO started"}),
		"footer": &fake{id: "footer", noFocus: true, body: "model x | mode ask | ctx 12%"},
	}
	root := layout.SidebarRightBottom("main", "side", "logs", layout.WithFooter("footer", 1), layout.WithGap(0))
	w := New(root, panes, append([]Option{WithTheme(th), WithPaneChrome("footer", None)}, opts...)...)
	w.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	w.Push(Overlay{ID: "permission", Pane: &fake{id: "permission", body: "overlay text"}, Width: 30, Height: 5, Modal: true})
	return w
}

func TestRenderPlainGolden(t *testing.T) {
	tuitest.Golden(t, "plain", tuitest.Matrix{Widths: []int{40, 80}}, func(c tuitest.Case) string {
		return plainWorkspace(caseTheme(c)).RenderPlain(c.Width)
	})
}

func TestRenderPlainNoEscapes(t *testing.T) {
	for _, c := range (tuitest.Matrix{Widths: []int{40}}).Cases() {
		w := plainWorkspace(caseTheme(c))
		w.Render()
		cache, plan, last := maps.Clone(w.cache), w.plan, w.last
		out := w.RenderPlain(c.Width)
		if strings.ContainsRune(out, '\x1b') {
			t.Errorf("%s: an escape sequence in %q", c.Name(), out)
		}
		for l := range strings.SplitSeq(out, "\n") {
			if strings.HasSuffix(l, " ") {
				t.Errorf("%s: trailing spaces in %q", c.Name(), l)
			}
		}
		if strings.Contains(out, "overlay text") {
			t.Errorf("%s: an overlay was written", c.Name())
		}
		if !maps.Equal(cache, w.cache) || !reflect.DeepEqual(plan, w.plan) || w.Render() != last {
			t.Errorf("%s: RenderPlain changed the cache, the plan or the frame", c.Name())
		}
	}
	w := New(layout.Pane{ID: "s"}, map[layout.PaneID]Pane{"s": Wrap(styledBubble{})}, WithTheme(asciiTheme))
	if got := w.RenderPlain(20); got != "s\nred line\n\n" {
		t.Errorf("a wrapped bubble without PlainView: %q", got)
	}
	if got := w.RenderPlain(0); got != "" {
		t.Errorf("RenderPlain(0) = %q", got)
	}
}

func TestRenderPlainOrder(t *testing.T) {
	titles := func(out string) []string {
		var ts []string
		for block := range strings.SplitSeq(strings.TrimSuffix(out, "\n\n"), "\n\n") {
			ts = append(ts, strings.SplitN(block, "\n", 2)[0])
		}
		return ts
	}
	th := theme.New(asciiTheme.Profile, theme.Unknown, glyph.ASCII())
	for _, tc := range []struct {
		name  string
		opts  []Option
		setup func(*Workspace)
		want  []string
	}{
		{"tree order, the footer last", nil, nil, []string{"Session", "Metrics and more [3]", "logs", "footer"}},
		{"WithFocusRing first", []Option{WithFocusRing("logs", "main")}, nil, []string{"logs", "Session", "Metrics and more [3]", "footer"}},
		{"a hidden pane", nil, func(w *Workspace) { w.Toggle("side") }, []string{"Session", "logs", "footer"}},
		{"zoomed", nil, func(w *Workspace) { w.Zoom("logs") }, []string{"logs"}},
	} {
		w := plainWorkspace(th, tc.opts...)
		if tc.setup != nil {
			tc.setup(w)
		}
		if got := titles(w.RenderPlain(100)); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestHelpUsesGlyphs(t *testing.T) {
	for _, tc := range []struct {
		g          glyph.Set
		has, hasNo string
	}{
		{glyph.ASCII(), " * ", "•…"},
		{glyph.Unicode(), " • ", ""},
	} {
		w := plainWorkspace(theme.New(asciiTheme.Profile, theme.Unknown, tc.g))
		w.Pop()
		h := w.Help()
		wide := h.View(w)
		h.SetWidth(20)
		narrow := h.View(w)
		h.ShowAll = true
		full := h.View(w)
		if !strings.Contains(wide, tc.has) {
			t.Errorf("the short help %q has no %q", wide, tc.has)
		}
		if tc.hasNo != "" && strings.ContainsAny(wide+narrow+full, tc.hasNo) {
			t.Errorf("ASCII glyphs drew %q, %q, %q", wide, narrow, full)
		}
		if !strings.HasSuffix(strings.TrimRight(narrow, " "), tc.g.Ellipsis) {
			t.Errorf("the truncated help %q does not end in %q", narrow, tc.g.Ellipsis)
		}
		if h.FullSeparator != "    " {
			t.Errorf("FullSeparator = %q; want bubbles' four spaces", h.FullSeparator)
		}
	}
}
