package workspace

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
)

// TestPrependPadding: under GraphemeWidth, a Unicode Prepend character joins
// the next character, so padding counted by adding widths comes out short,
// and a full line that ends with one joins the border after it
// (docs/decisions/0014-PLAN-hardening.md Step 9, D8, found by FuzzClip). It
// uses only names that existed before the fix.
func TestPrependPadding(t *testing.T) {
	m := ansi.GraphemeWidth
	for _, c := range []struct {
		s string
		w int
	}{
		{"\u0605", 3}, {"abc\u0605", 6}, {"abc\u0605", 3}, {"\u0d4e", 2}, {"ab\u0d4e", 3}, {"\u0600123", 5},
		{"\u0d40", 1}, {"\u0940ab", 3}, {"\x1b[1m\u0d40ab\x1b[m", 3}, {"\ufe0f", 3}, {"\ufe0fab", 3},
	} {
		got := clip(m, c.s, c.w, 1)
		if m.StringWidth(got) != c.w || m.StringWidth(got+"|") != c.w+1 || m.StringWidth("|"+got) != c.w+1 {
			t.Errorf("clip(%q, %d) = %q: %d cells, %d with a border after it, %d with one before it",
				c.s, c.w, got, m.StringWidth(got), m.StringWidth(got+"|"), m.StringWidth("|"+got))
		}
	}
	for _, chrome := range []Chrome{Borders, Separators} {
		root := layout.Split{Axis: layout.Horizontal, Children: []layout.Child{
			{Node: layout.Pane{ID: "a"}, Size: layout.Fixed(5)}, {Node: layout.Pane{ID: "b"}},
		}}
		w := New(root, map[layout.PaneID]Pane{
			"a": &fake{id: "a", title: "t\u0605", body: "abc\u0605\n\u0605"},
			"b": &fake{id: "b", title: "u\u0605", body: "x\u0605"},
		}, WithTheme(asciiTheme), WithWidthMethod(m), WithChrome(chrome))
		w.Update(tea.WindowSizeMsg{Width: 12, Height: 4})
		for i, l := range strings.Split(w.Render(), "\n") {
			// The frame drops a line's trailing blanks, so only a bordered
			// line is the whole width; the title line is checked below.
			if got := m.StringWidth(l); chrome == Borders && got != 12 {
				t.Errorf("chrome %v, row %d: %d cells, want 12: %q", chrome, i, got, ansi.Strip(l))
			}
		}
		if chrome == Separators {
			line := w.title(&fake{id: "a", title: "t\u0605"}, "a", false, 8)
			if got := m.StringWidth(line); got != 8 {
				t.Errorf("title: %d cells, want 8: %q", got, ansi.Strip(line))
			}
		}
	}
}

// TestClipNeverWider: under either method, a clipped line is exactly its
// width, whatever x/ansi's Truncate makes of a ZWJ emoji sequence, Hangul
// jamo, a conjunct or a Prepend character; and a bordered pane holding
// them keeps its border (docs/decisions/0014-PLAN-hardening.md Step 9,
// D8). It uses only names that existed before the fix.
func TestClipNeverWider(t *testing.T) {
	texts := []string{
		"\U0001f468\u200d\U0001f469\u200d\U0001f467", "\u1100\u1161\u11a8", "\u0915\u094d\u0937",
		"0\u0605000", "\u06050", "\u0d4e0", "\x1b[1m\U0001f468\u200d\U0001f469\x1b[m",
	}
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		for _, s := range texts {
			for w := range 7 {
				for l := range strings.SplitSeq(clip(m, s, w, 1), "\n") {
					if got := m.StringWidth(l); got != w {
						t.Errorf("method %v: clip(%q, %d) = %q, %d cells", m, s, w, l, got)
					}
				}
			}
		}
		root := layout.Split{Axis: layout.Horizontal, Children: []layout.Child{
			{Node: layout.Pane{ID: "a"}, Size: layout.Fixed(5)}, {Node: layout.Pane{ID: "b"}},
		}}
		body := strings.Join(texts, "\n")
		w := New(root, map[layout.PaneID]Pane{
			"a": &fake{id: "a", title: texts[0], body: body}, "b": &fake{id: "b", title: texts[1], body: body},
		}, WithTheme(asciiTheme), WithWidthMethod(m))
		w.Update(tea.WindowSizeMsg{Width: 12, Height: 9})
		for i, l := range strings.Split(w.Render(), "\n") {
			if got := m.StringWidth(l); got != 12 || !strings.HasSuffix(ansi.Strip(l), "|") && !strings.HasSuffix(ansi.Strip(l), "+") {
				t.Errorf("method %v, row %d: %d cells, or no border at its end: %q", m, i, got, ansi.Strip(l))
			}
		}
	}
}
