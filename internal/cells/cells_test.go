package cells

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
)

// lines is the frame's rendered rows, padded to the frame's height, since
// Render trims trailing spaces.
func lines(f *Frame) []string {
	out := strings.Split(f.Render(), "\n")
	for len(out) < f.Height() {
		out = append(out, "")
	}
	return out
}

// fill writes c into every cell, so a test can see which cells changed.
func fill(f *Frame, c string) {
	row := strings.Repeat(c, f.Width())
	rows := make([]string, f.Height())
	for i := range rows {
		rows[i] = row
	}
	f.Draw(strings.Join(rows, "\n"), layout.Rect{W: f.Width(), H: f.Height()})
}

func TestDrawClipsToItsRectangle(t *testing.T) {
	f := NewFrame(10, 3, ansi.WcWidth)
	fill(f, "x")
	f.Draw("hello world\nsecond line", layout.Rect{X: 2, Y: 1, W: 4, H: 1})
	want := []string{"xxxxxxxxxx", "xxhellxxxx", "xxxxxxxxxx"}
	if got := lines(f); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("frame\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestDrawClearsItsRectangleFirst(t *testing.T) {
	f := NewFrame(10, 1, ansi.WcWidth)
	fill(f, "x")
	r := layout.Rect{X: 2, W: 6, H: 1}
	f.Draw("abcdef", r)
	f.Draw("ab", r)
	if got := lines(f)[0]; got != "xxab    xx" {
		t.Fatalf("after a shorter draw: %q, want %q: no tail of the longer one", got, "xxab    xx")
	}
}

func TestClearAndResize(t *testing.T) {
	f := NewFrame(6, 2, ansi.WcWidth)
	fill(f, "x")
	f.Resize(6, 2)
	if f.Width() != 6 || f.Height() != 2 || lines(f)[1] != "xxxxxx" {
		t.Fatalf("Resize to the same size changed the frame: %dx%d %q", f.Width(), f.Height(), lines(f))
	}
	f.Clear()
	if got := strings.TrimRight(f.Render(), "\n"); got != "" {
		t.Fatalf("after Clear: %q, want an empty frame", got)
	}
	f.Resize(9, 4)
	if f.Width() != 9 || f.Height() != 4 {
		t.Fatalf("Resize(9, 4): %dx%d", f.Width(), f.Height())
	}
	f.Resize(-1, -3)
	if f.Width() != 0 || f.Height() != 0 {
		t.Fatalf("a negative size: %dx%d, want 0x0", f.Width(), f.Height())
	}
}

// The family emoji joins three emoji with zero-width joiners; the heart is
// U+2764 with the VS16 selector. wcwidth and grapheme widths disagree on
// both, which is the misalignment 0004-MADR §3 fixes.
const family, heart = "\U0001F468\u200D\U0001F469\u200D\U0001F467", "\u2764\uFE0F"

func TestTheMethodDecidesWhereTheRestLands(t *testing.T) {
	for _, s := range []string{family, heart, "ab"} {
		widths := map[ansi.Method]int{}
		for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
			f := NewFrame(12, 1, ansi.WcWidth)
			f.SetMethod(m)
			if f.Method() != m {
				t.Fatalf("Method() is %v after SetMethod(%v)", f.Method(), m)
			}
			f.Draw("a"+s+"b", layout.Rect{W: 10, H: 1})
			f.Draw("|", layout.Rect{X: 10, W: 1, H: 1})
			row := lines(f)[0]
			// The spaces between b and | are what is left of the 10 cells.
			gap := strings.Index(row, "|") - strings.LastIndex(row, "b") - 1
			want := 10 - 2 - m.StringWidth(s)
			if gap != want {
				t.Fatalf("%q at method %v: %d cells between b and |, want %d (row %q)", s, m, gap, want, row)
			}
			widths[m] = m.StringWidth(s)
		}
		differ := widths[ansi.WcWidth] != widths[ansi.GraphemeWidth]
		if differ != (s != "ab") {
			t.Fatalf("%q: wcwidth %d, grapheme %d; the emoji must differ and ASCII must not", s, widths[ansi.WcWidth], widths[ansi.GraphemeWidth])
		}
	}
}

// TestRenderMatchesLipglossCanvas draws the same layers as lipgloss's canvas
// and compositor do, as the workspace drew its frame up to v0.1.6, and
// requires the same bytes.
func TestRenderMatchesLipglossCanvas(t *testing.T) {
	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).Foreground(lipgloss.Color("#ff8800"))
	type placed struct {
		content string
		x, y    int
	}
	layers := []placed{
		{border.Width(14).Render("main pane\n" + family + " and " + heart + "\n日本語のテキスト"), 0, 0},
		{border.Width(10).Render("side\nwrapped text that is long"), 16, 0},
		{lipgloss.NewStyle().Bold(true).Render("│\n│\n│\n│\n│\n│"), 15, 0},
		{border.Width(12).Background(lipgloss.Color("236")).Render("dialog\nover both"), 8, 2},
	}
	canvas := lipgloss.NewCanvas(32, 9)
	var ls []*lipgloss.Layer
	for i, l := range layers {
		ls = append(ls, lipgloss.NewLayer(l.content).X(l.x).Y(l.y).Z(i))
	}
	canvas.Compose(lipgloss.NewCompositor(ls...))

	f := NewFrame(32, 9, ansi.GraphemeWidth) // lipgloss's canvas measures in graphemes
	for _, l := range layers {
		f.Draw(l.content, layout.Rect{X: l.x, Y: l.y, W: lipgloss.Width(l.content), H: lipgloss.Height(l.content)})
	}
	if got, want := f.Render(), canvas.Render(); got != want {
		t.Fatalf("Render differs from lipgloss's canvas:\n got %q\nwant %q", got, want)
	}
	for i, row := range strings.Split(f.Render(), "\n") {
		if strings.HasSuffix(ansi.Strip(row), " ") {
			t.Fatalf("row %d ends in a space: %q", i, row)
		}
	}
}
