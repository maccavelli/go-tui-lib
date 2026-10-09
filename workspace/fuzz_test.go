package workspace

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
)

// Fuzz targets (docs/decisions/0014-PLAN-hardening.md Step 9, finding H9).

// nonNegative is n with its sign dropped, without overflowing at the
// minimum int.
func nonNegative(n int) int {
	if n < 0 {
		return -(n + 1)
	}
	return n
}

// cellAt is what starts at cell x of line, a line with no escape
// sequences: a rune under WcWidth, as a terminal that does not cluster
// draws it; under GraphemeWidth, a grapheme cut as ultraviolet's renderer
// cuts it, with the decoder and, after an ASCII byte that a non-ASCII one
// follows, ansi.FirstGraphemeCluster (ultraviolet styled.go). What is zero
// cells wide belongs to no cell of its own. It does not use Method.Cut,
// whose WcWidth form ignores left in x/ansi v0.11.8.
func cellAt(m ansi.Method, line string, x int) string {
	col := 0
	var state byte
	for line != "" {
		seq, n := string([]rune(line)[0]), 0
		if m == ansi.GraphemeWidth {
			seq, _, n, state = m.DecodeSequenceInString(line, state, nil)
			if n == 0 {
				return ""
			}
			if n == 1 && seq[0] > 0x1f && seq[0] < 0x7f && len(line) > 1 && line[1] >= 0xc0 {
				if cluster, _ := ansi.FirstGraphemeCluster(line, m); len(cluster) > 1 {
					seq, n = cluster, len(cluster)
				}
			}
		} else {
			n = len(seq)
		}
		line = line[n:]
		w := m.StringWidth(seq)
		if w > 0 && col == x {
			return seq
		}
		if col += w; col > x {
			return ""
		}
	}
	return ""
}

// FuzzClip: clip gives exactly h lines, each exactly w cells under its
// method, none starting with what joins the character before it or ending
// with what joins the character after it, with no carriage return.
func FuzzClip(f *testing.F) {
	for _, c := range []struct {
		s    string
		w, h int
	}{
		{"\x1b", 3, 1}, {"a\x1b[1mb\x1b[m", 4, 1}, {"a\u202eb\x07c", 3, 1}, {"a\tb", 3, 1}, {"x\x1b[31", 2, 1},
		{"\u65e5\u672c\u8a9e\n\u65e5\u672c", 5, 3}, {"a\r\nb\rc", 4, 2}, {"\x1b]8;;http://x\x07link\x1b]8;;\x07", 3, 1},
		{"\U0001f468\u200d\U0001f469\u200d\U0001f467", 1, 1}, {"", 0, 0}, {"line", 0, 2}, {strings.Repeat("ab\n", 50), 7, 4},
		{"\u0605", 3, 2}, {"abc\u0605", 6, 1}, {"\u0d4e", 2, 1}, // D8: Prepend characters
		// What fuzzing found (docs/decisions/0014-PLAN-hardening.md Step 9,
		// D8 and D9).
		{"\u0605", 3, 109}, {"\u06050", 0, 136}, {"\x1b]\r\a", 3, 1}, {"\U0001f468\u200d\U0001f469\u200d\U0001f467", 3, 1},
		{"\u0d40", 1, 1}, {"\x1b[1m\u0d40ab\x1b[m", 3, 1}, {"\ufe0f", 36, 11},
	} {
		f.Add(c.s, c.w, c.h, false)
		f.Add(c.s, c.w, c.h, true)
	}
	f.Fuzz(func(t *testing.T, s string, w, h int, grapheme bool) {
		w, h = nonNegative(w)%201, nonNegative(h)%41
		m := ansi.WcWidth
		if grapheme {
			m = ansi.GraphemeWidth
		}
		out := clip(m, s, w, h)
		if h == 0 {
			if out != "" {
				t.Fatalf("clip(%q, %d, 0) = %q, want nothing", s, w, out)
			}
			return
		}
		lines := strings.Split(out, "\n")
		if len(lines) != h {
			t.Fatalf("clip(%q, %d, %d): %d lines", s, w, h, len(lines))
		}
		for i, l := range lines {
			if got := m.StringWidth(l); got != w {
				t.Fatalf("clip(%q, %d, %d): line %d is %d cells: %q", s, w, h, i, got, l)
			}
			if w > 0 && m.StringWidth(l+"|") != w+1 {
				t.Fatalf("clip(%q, %d, %d): line %d, %q, joins the character after it", s, w, h, i, l)
			}
			if w > 0 && m.StringWidth("|"+l) != w+1 {
				t.Fatalf("clip(%q, %d, %d): line %d, %q, joins the character before it", s, w, h, i, l)
			}
		}
		if strings.ContainsRune(out, '\r') {
			t.Fatalf("clip(%q, %d, %d) = %q holds a carriage return", s, w, h, out)
		}
	})
}

// FuzzRender: a workspace of bordered panes with fuzzed titles, badges and
// bodies, at a fuzzed size, under either width method, draws every pane's
// four corners, its left, right and bottom edges, and the start of its top
// edge intact.
func FuzzRender(f *testing.F) {
	for _, c := range []struct {
		title, badge, body string
		w, h               int
	}{
		{"main", "", "body", 80, 24},
		{"evil\nline", "b\r\x1b[2J\u202e", "body \u202etext\x1b", 60, 12},
		{"\u65e5\u672c\u8a9e", "\u65e5", strings.Repeat("\u65e5", 50) + "\n" + strings.Repeat("x", 300), 41, 9},
		{strings.Repeat("t", 200), "99+", "\x1b[31mred\x1b[m\n\t\ttab", 7, 5},
		{"\x1b]8;;http://x\x07link\x1b]8;;\x07", "\x1b]0;title\x07", "\x1b[2J\x1b[H", 120, 40},
		{"", "", "", 3, 3},
		{"t\u0605", "\u0605", "abc\u0605\n\u0600123\nabcdefghi\u0d4e", 11, 5}, // D8
		// What fuzzing found (docs/decisions/0014-PLAN-hardening.md Step 9,
		// D8 and D9, and the oracle's segmentation).
		{"0", "\r\u202e", "000\x1b X0000000", 60, 12}, {"\u0605", "\u0605", "0\u0605000", 11, 100},
		{"0", "\u0940", "0", 31, 5}, {"0", "0", "\U0001f468\u200d\U0001f469\u200d\U0001f467\n\u1100\u1161\u11a8", 12, 6},
		{"0", "0", "\u0d40", 8, 54}, {"0", "0", "\x1b[" + strings.Repeat(":", 32) + "0m", 80, 67},
	} {
		for _, flags := range []byte{0, 1, 2, 3} {
			f.Add(c.title, c.badge, c.body, c.w, c.h, flags)
		}
	}
	f.Fuzz(func(t *testing.T, title, badge, body string, width, height int, flags byte) {
		unicode, grapheme := flags&1 != 0, flags&2 != 0
		width, height = 1+nonNegative(width)%240, 1+nonNegative(height)%60
		th := theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII())
		if unicode {
			th = theme.New(colorprofile.TrueColor, theme.Unknown, glyph.Unicode())
		}
		root := layout.Split{Axis: layout.Horizontal, Children: []layout.Child{
			{Node: layout.Pane{ID: "a"}},
			{Node: layout.Split{Axis: layout.Vertical, Children: []layout.Child{
				{Node: layout.Pane{ID: "b"}}, {Node: layout.Pane{ID: "c"}},
			}}},
		}}
		panes := map[layout.PaneID]Pane{}
		for _, id := range []string{"a", "b", "c"} {
			panes[layout.PaneID(id)] = &fake{id: id, title: title, badge: badge, body: body}
		}
		m := ansi.WcWidth
		if grapheme {
			m = ansi.GraphemeWidth
		}
		w := New(root, panes, WithTheme(th), WithChrome(Borders), WithFocus("a"), WithWidthMethod(m))
		w.Update(tea.WindowSizeMsg{Width: width, Height: height})
		frame := strings.Split(ansi.Strip(w.Render()), "\n")
		cell := func(x, y int) string {
			if y >= len(frame) {
				return ""
			}
			return cellAt(m, frame[y], x)
		}
		type mark struct {
			x, y int
			g    string
		}
		b := w.theme.Border(w.border)
		for _, id := range w.plan.Order {
			r := w.plan.Panes[id]
			if insetBorder(r).W == 0 {
				continue // too small for a border: drawn blank
			}
			x0, y0, x1, y1 := r.X, r.Y, r.X+r.W-1, r.Y+r.H-1
			want := []mark{
				{x0, y0, b.TopLeft}, {x0 + 1, y0, b.Top}, {x1, y0, b.TopRight},
				{x0, y1, b.BottomLeft}, {x1, y1, b.BottomRight},
			}
			for y := y0 + 1; y < y1; y++ {
				want = append(want, mark{x0, y, b.Left}, mark{x1, y, b.Right})
			}
			for x := x0 + 1; x < x1; x++ {
				want = append(want, mark{x, y1, b.Bottom})
			}
			for _, c := range want {
				// A combining mark may sit on an edge's glyph; the glyph
				// is still there, in its cell.
				if got := cell(c.x, c.y); !strings.HasPrefix(got, c.g) {
					t.Fatalf("%dx%d, method %v, pane %s at %+v: cell (%d, %d) is %q, want %q\n%s",
						width, height, m, id, r, c.x, c.y, got, c.g, strings.Join(frame, "\n"))
				}
			}
		}
	})
}
