package inlinetest

import (
	"slices"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/inline"
)

func write(s *Screen, text string) { _, _ = s.Write([]byte(text)) }

// TestWrites: text, the pending wrap, the moves, the erases, DECTCEM and
// mode 2026.
func TestWrites(t *testing.T) {
	s := NewScreen(10, 4, ansi.WcWidth)
	write(s, "0123456789")
	if s.Row != 0 || s.Col != 9 || !s.pending {
		t.Fatalf("a full row: cursor %d,%d, pending %v; want 0,9 with a wrap pending", s.Row, s.Col, s.pending)
	}
	write(s, "a")
	if got := s.Lines()[:2]; !slices.Equal(got, []string{"0123456789", "a"}) || s.Row != 1 || s.Col != 1 {
		t.Fatalf("after the pending wrap: %q, cursor %d,%d", got, s.Row, s.Col)
	}
	write(s, "\r\x1b[1A\x1b[3C\x1b[K")
	if got := s.Lines()[0]; got != "012" {
		t.Errorf("CSI K after CSI 3 C: %q", got)
	}
	write(s, "\x1b[2;2H\x1b[J")
	if got := s.Lines()[:2]; !slices.Equal(got, []string{"012", "a"}) || s.Row != 1 || s.Col != 1 {
		t.Errorf("CSI 2;2 H then CSI J: %q, cursor %d,%d", got, s.Row, s.Col)
	}
	write(s, "\x1b[?25l\x1b[?2026h")
	if !s.Hidden || !s.Synchronized {
		t.Errorf("hidden %v, synchronized %v; want both", s.Hidden, s.Synchronized)
	}
	write(s, "\x1b[?2026l\x1b[?25h\x1b[1;31mred\x1b[m")
	if s.Hidden || s.Synchronized || s.Lines()[1] != "ared" {
		t.Errorf("hidden %v, synchronized %v, row %q", s.Hidden, s.Synchronized, s.Lines()[1])
	}
	write(s, "\x1b[2J")
	if got := s.Text(); got != "" {
		t.Errorf("CSI 2 J left %q", got)
	}
}

// TestLineFeed: raw mode keeps the column; Cooked returns to the start; the
// bottom row scrolls into the scrollback, which CSI 3 J clears.
func TestLineFeed(t *testing.T) {
	s := NewScreen(10, 2, ansi.WcWidth)
	write(s, "ab\ncd")
	if got := s.Lines(); !slices.Equal(got, []string{"ab", "  cd"}) {
		t.Errorf("raw: %q", got)
	}
	write(s, "\r\nef")
	if got, back := s.Lines(), s.Scrollback(); !slices.Equal(got, []string{"  cd", "ef"}) || !slices.Equal(back, []string{"ab"}) {
		t.Errorf("scrolled: %q, scrollback %q", got, back)
	}
	write(s, "\x1b[3J")
	if back := s.Scrollback(); len(back) != 0 {
		t.Errorf("CSI 3 J kept %q", back)
	}
	c := NewScreen(10, 2, ansi.WcWidth)
	c.Cooked = true
	write(c, "ab\ncd")
	if got := c.Lines(); !slices.Equal(got, []string{"ab", "cd"}) {
		t.Errorf("cooked: %q", got)
	}
}

// TestClusters: a wide cluster takes two cells and is never split by a
// wrap; under GraphemeWidth a ZWJ family is one cluster of two; text over
// half a wide cluster blanks its other half.
func TestClusters(t *testing.T) {
	s := NewScreen(5, 3, ansi.GraphemeWidth)
	write(s, "abcd世")
	if got := s.Lines()[:2]; !slices.Equal(got, []string{"abcd", "世"}) || s.Col != 2 {
		t.Errorf("a wide cluster at the edge: %q, column %d", got, s.Col)
	}
	write(s, "\r\n👨‍👩‍👧x")
	if got := s.Lines()[2]; got != "👨‍👩‍👧x" || s.Col != 3 {
		t.Errorf("a ZWJ family: %q, column %d; want two cells and x", got, s.Col)
	}
	write(s, "\x1b[1A\r\x1b[1Cz")
	if got := s.Lines()[1]; got != " z" {
		t.Errorf("z over the second cell of 世: %q", got)
	}
}

// TestAnswer: the cursor report is the cursor's place, from 1, and at a
// pending wrap the last column.
func TestAnswer(t *testing.T) {
	s := NewScreen(10, 4, ansi.WcWidth)
	var got []string
	s.Answer = func(r string) { got = append(got, r) }
	write(s, "\n\nabc\x1b[6n0123456\x1b[6n")
	if !slices.Equal(got, []string{"\x1b[3;4R", "\x1b[3;10R"}) {
		t.Errorf("reports %q", got)
	}
}

// TestSplitWrite: a sequence cut between two writes is applied whole.
func TestSplitWrite(t *testing.T) {
	s := NewScreen(10, 2, ansi.WcWidth)
	write(s, "abc\x1b[")
	write(s, "2Dx")
	if got := s.Lines()[0]; got != "axc" {
		t.Errorf("CSI 2 D across two writes: %q", got)
	}
}

// TestResize: the three ways a terminal narrows, from gobble-cli's
// 0030-PLAN P1, with the cursor past the new width.
func TestResize(t *testing.T) {
	start := func() *Screen {
		s := NewScreen(10, 8, ansi.WcWidth)
		write(s, "top\r\n0123456789\r\nabcdef")
		return s
	}
	for _, c := range []struct {
		how      inline.Reflow
		lines    []string
		row, col int
		back     []string
	}{
		{inline.ReflowDown, []string{"top", "0123", "4567", "89", "abcd", "ef", "", ""}, 5, 2, nil},
		{inline.ReflowUnknown, []string{"top", "0123", "4567", "89", "abcd", "ef", "", ""}, 5, 2, nil},
		{inline.ReflowKeepRow, []string{"89", "abcd", "ef", "", "", "", "", ""}, 2, 2, []string{"top", "0123", "4567"}},
		{inline.ReflowCut, []string{"top", "0123", "abcd", "", "", "", "", ""}, 2, 3, nil},
	} {
		s := start()
		s.Resize(4, c.how)
		if got := s.Lines(); !slices.Equal(got, c.lines) || s.Row != c.row || s.Col != c.col || !slices.Equal(s.Scrollback(), c.back) {
			t.Errorf("%s: %q, cursor %d,%d, scrollback %q; want %q, %d,%d, %q", c.how, got, s.Row, s.Col, s.Scrollback(), c.lines, c.row, c.col, c.back)
		}
	}
	w := NewScreen(5, 2, ansi.GraphemeWidth)
	write(w, "ab世c")
	w.Resize(3, inline.ReflowDown)
	if got := w.Lines(); !slices.Equal(got, []string{"ab", "世c"}) {
		t.Errorf("a wide cluster re-wrapped: %q; want it whole on the next row", got)
	}
}

// TestWiderClusters: under WcWidth a ZWJ sequence is one cluster as wide as
// its emoji, four or six cells, and keeps that width across a resize, and
// text over any of its cells blanks all of it
// (0023-PLAN-line-mode-components.md Step 3, D12).
func TestWiderClusters(t *testing.T) {
	const pair, family = "👨‍👩", "👨‍👩‍👧" // 4 and 6 cells under WcWidth
	s := NewScreen(10, 3, ansi.WcWidth)
	write(s, "a"+pair+"b")
	s.Resize(10, inline.ReflowDown)
	write(s, "\r\x1b[5Cc")
	if got := s.Lines()[0]; got != "a"+pair+"c" || s.Col != 6 {
		t.Errorf("after a resize to the same width, c at column 5: %q, column %d; want b overwritten", got, s.Col)
	}
	s.Resize(5, inline.ReflowDown)
	if got := s.Lines()[:2]; !slices.Equal(got, []string{"a" + pair, "c"}) {
		t.Errorf("re-wrapped at 5: %q; want the 4-cell pair kept whole on the first row", got)
	}
	for _, c := range []struct {
		name, keys, want string
	}{
		{"a cell inside it", "\r\x1b[3Cz", "x  z   y"},
		{"its first cell", "\r\x1b[1Cw", "xw     y"},
		{"an erase from inside it", "\r\x1b[4C\x1b[K", "x"},
	} {
		s := NewScreen(10, 2, ansi.WcWidth)
		write(s, "x"+family+"y"+c.keys)
		if got := s.Lines()[0]; got != c.want {
			t.Errorf("%s: %q; want %q, the 6-cell family blanked whole", c.name, got, c.want)
		}
	}
}
