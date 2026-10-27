package inline_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// host is a line-mode host on a test screen: what the region returns is
// written to the screen, and what the screen answers is decoded and given
// to the region first; what the region does not take is the component's.
type host struct {
	t       *testing.T
	scr     *inlinetest.Screen
	r       *inline.Region
	dec     termkey.Decoder
	replies []byte
	now     time.Time
	keys    []termkey.Input
}

func newHost(t *testing.T, columns, rows int, m ansi.Method, o ...inline.Option) *host {
	t.Helper()
	h := &host{t: t, scr: inlinetest.NewScreen(columns, rows, m), now: time.Unix(1, 0)}
	h.r = inline.NewRegion(append([]inline.Option{inline.WithMethod(m)}, o...)...)
	h.scr.Answer = func(reply string) { h.replies = append(h.replies, reply...) }
	h.write(h.r.Resize(columns, h.now))
	return h
}

func (h *host) write(s string) { _, _ = h.scr.Write([]byte(s)) }

// feed decodes b, as typed or answered, and gives each input to the
// region, then to the component.
func (h *host) feed(b []byte) {
	ins := h.dec.Flush(h.dec.Decode(b, nil))
	for _, in := range ins {
		if out, ok := h.r.HandleInput(in, h.now); ok {
			h.write(out)
			continue
		}
		h.keys = append(h.keys, in)
	}
}

// answer feeds what the screen answered since the last answer.
func (h *host) answer() {
	b := h.replies
	h.replies = nil
	h.feed(b)
}

// chunk lays text out in rows of width cells, as a component would, one
// cell a byte; the cursor is at the end.
func chunk(text string, width int) (rows []string, row, col int) {
	for len(text) > width {
		rows = append(rows, text[:width])
		text = text[width:]
	}
	rows = append(rows, text)
	return rows, len(rows) - 1, len(text)
}

// words lays text out in rows of width cells at its spaces, as gobble-cli's
// editor does (its layout.go), one cell a byte: a row holds whole words and
// the space after each. at is the row and column of the byte offset off.
func words(text string, width, off int) (rows []string, row, col int) {
	start, last := 0, 0
	var starts []int
	for i := range len(text) {
		if i-start == width {
			cut := last
			if cut <= start {
				cut = i
			}
			rows, starts, start = append(rows, text[start:cut]), append(starts, start), cut
		}
		if text[i] == ' ' {
			last = i + 1
		}
	}
	rows, starts = append(rows, text[start:]), append(starts, start)
	for i, st := range starts {
		if st <= off {
			row, col = i, off-st
		}
	}
	return rows, row, col
}

func (h *host) wantLines(what string, want ...string) {
	h.t.Helper()
	if got := h.scr.Lines()[:len(want)]; !slices.Equal(got, want) {
		h.t.Fatalf("%s: the screen is\n%q\nwant\n%q", what, h.scr.Lines(), want)
	}
	for _, l := range h.scr.Lines()[len(want):] {
		if l != "" {
			h.t.Fatalf("%s: the screen is\n%q\nwant nothing below %q", what, h.scr.Lines(), want)
		}
	}
}

// TestDrawDiff: a redraw writes the changed rows only, from what they
// share with the row before; a shorter region erases what it no longer
// has; an unchanged one writes nothing.
func TestDrawDiff(t *testing.T) {
	h := newHost(t, 40, 6, ansi.WcWidth)
	h.write("above\r\n")
	h.write(h.r.Draw([]string{"> one", "  two", "  three"}, 2, 7))
	h.wantLines("the first draw", "above", "> one", "  two", "  three")
	if h.scr.Row != 3 || h.scr.Col != 7 {
		t.Fatalf("the cursor is at %d,%d; want 3,7", h.scr.Row, h.scr.Col)
	}
	out := h.r.Draw([]string{"> one", "  twx", "  three"}, 1, 5)
	if strings.Contains(out, "one") || strings.Contains(out, "three") || strings.Contains(out, "tw") || !strings.Contains(out, "\x1b[4Cx") {
		t.Errorf("one changed cell wrote %q; want only the x, after a move past what is shared", out)
	}
	h.write(out)
	h.wantLines("one cell changed", "above", "> one", "  twx", "  three")
	h.write(h.r.Draw([]string{"> one", "  tw"}, 1, 4))
	h.wantLines("a shorter region", "above", "> one", "  tw")
	if out := h.r.Draw([]string{"> one", "  tw"}, 1, 4); out != "" {
		t.Errorf("an unchanged draw wrote %q", out)
	}
	if h.scr.Row != 2 || h.scr.Col != 4 || h.scr.Hidden || h.scr.Synchronized {
		t.Errorf("the cursor at %d,%d, hidden %v, synchronized %v; want 2,4 shown, mode 2026 off", h.scr.Row, h.scr.Col, h.scr.Hidden, h.scr.Synchronized)
	}
}

// TestDrawBudget: a key's drawing writes at most the text it changed and 40
// bytes besides, the synchronized update, the hidden cursor and the moves
// (gobble-cli's 0030-MADR D5): a key at a long line's end, one in its
// middle, and one that only moves the cursor.
func TestDrawBudget(t *testing.T) {
	text := "> " + strings.Repeat("words fill the prompt ", 10)
	for _, c := range []struct {
		name          string
		before, after string
		was, cursor   int // the cursor's byte offset in before, and in after
	}{
		{"a key at the end", text, text + "x", len(text), len(text) + 1},
		{"a key in the middle", text, text[:40] + "x" + text[40:], 40, 41},
		{"the cursor moved", text, text, len(text), len(text) - 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := inline.NewRegion()
			r.Resize(80, time.Unix(1, 0))
			old, orow, ocol := words(c.before, 80, c.was)
			now, row, col := words(c.after, 80, c.cursor)
			r.Draw(old, orow, ocol)
			out := r.Draw(now, row, col)
			changed := 0
			for i, row := range now {
				was := ""
				if i < len(old) {
					was = old[i]
				}
				n := 0
				for n < len(was) && n < len(row) && was[n] == row[n] {
					n++
				}
				changed += len(row) - n
			}
			if len(out) > changed+40 {
				t.Fatalf("%d bytes for %d changed, more than %d: %q", len(out), changed, changed+40, out)
			}
		})
	}
}

// TestStyledRow: a row rewritten after a style sets the style again before
// its changed text, and resets it after.
func TestStyledRow(t *testing.T) {
	h := newHost(t, 20, 3, ansi.WcWidth)
	h.write(h.r.Draw([]string{"\x1b[1mhello\x1b[m"}, 0, 5))
	out := h.r.Draw([]string{"\x1b[1mhellx\x1b[m"}, 0, 5)
	if !strings.Contains(out, "\r\x1b[4C\x1b[1mx\x1b[m") {
		t.Errorf("the rewrite is %q; want the bold set again before x", out)
	}
	h.write(out)
	h.wantLines("a styled row", "hellx")
}

// TestClusterPrefix: the shared part ends before a grapheme cluster that
// differs, never inside it.
func TestClusterPrefix(t *testing.T) {
	h := newHost(t, 20, 3, ansi.GraphemeWidth)
	h.write(h.r.Draw([]string{"a👨‍👩‍👧b"}, 0, 4))
	out := h.r.Draw([]string{"a👨‍👩‍👦b"}, 0, 4)
	if !strings.Contains(out, "\r\x1b[1C👨‍👩‍👦b") {
		t.Errorf("the rewrite is %q; want it from the cluster's start", out)
	}
	h.write(out)
	h.wantLines("a cluster changed", "a👨‍👩‍👦b")
}

// TestPrint: text prints above the region, which is drawn whole again
// below it, the cursor back in place.
func TestPrint(t *testing.T) {
	h := newHost(t, 30, 8, ansi.WcWidth)
	h.write("above\r\n")
	h.write(h.r.Draw([]string{"> hel", "  lo"}, 1, 4))
	h.write(h.r.Print("reply one\nreply two"))
	h.wantLines("a print", "above", "reply one", "reply two", "> hel", "  lo")
	if h.scr.Row != 4 || h.scr.Col != 4 {
		t.Errorf("the cursor is at %d,%d; want 4,4", h.scr.Row, h.scr.Col)
	}
	h.write(h.r.Draw([]string{"> help", "  lo"}, 0, 6))
	h.wantLines("a draw after the print", "above", "reply one", "reply two", "> help", "  lo")
	empty := newHost(t, 30, 4, ansi.WcWidth)
	empty.write(empty.r.Print("only text"))
	empty.wantLines("a print with nothing drawn", "only text")
}

// TestCommit: the rows stay in the scrollback, whole, the cursor below
// them, and the next draw starts there.
func TestCommit(t *testing.T) {
	h := newHost(t, 30, 8, ansi.WcWidth)
	h.write(h.r.Draw([]string{"> a question"}, 0, 12))
	h.write(h.r.Commit([]string{"> a question", "allow once"}))
	h.write(h.r.Draw([]string{"> next"}, 0, 6))
	h.wantLines("a commit", "> a question", "allow once", "> next")
	h.write(h.r.Clear())
	h.wantLines("a clear", "> a question", "allow once")
	if h.scr.Row != 2 || h.scr.Col != 0 {
		t.Errorf("after the clear the cursor is at %d,%d; want 2,0", h.scr.Row, h.scr.Col)
	}
}

// TestResize: each re-wrap kind, read from the cursor report, or taken
// from WithReflow when it is not asked for or does not come; the region is
// drawn once, where it now is, and the line above it kept.
func TestResize(t *testing.T) {
	const line = "> abcdefghijklm" // 15 cells, the cursor after it, past the new width
	want := "above\n> abcdefgh\nijklm\n"
	for _, how := range []inline.Reflow{inline.ReflowDown, inline.ReflowKeepRow, inline.ReflowCut} {
		for _, mode := range []string{"report", "timeout", "no query"} {
			t.Run(how.String()+", "+mode, func(t *testing.T) {
				var o []inline.Option
				switch mode {
				case "timeout":
					o = append(o, inline.WithReflow(how))
				case "no query":
					o = append(o, inline.WithReflow(how), inline.WithoutCursorQuery())
				}
				h := newHost(t, 20, 6, ansi.WcWidth, o...)
				h.write("above\r\n")
				rows, row, col := chunk(line, 20)
				h.write(h.r.Draw(rows, row, col))
				h.scr.Resize(10, how)
				out := h.r.Resize(10, h.now)
				if got := strings.Contains(out, "\x1b[6n"); got != (mode != "no query") {
					t.Fatalf("the resize wrote %q", out)
				}
				h.write(out)
				rows, row, col = chunk(line, 10)
				h.write(h.r.Draw(rows, row, col)) // the component's layout at the new width
				switch mode {
				case "report":
					h.answer()
				case "timeout":
					h.replies = nil // the terminal does not answer
					h.write(h.r.HandleTimeout(h.now.Add(inline.DefaultReportWait)))
				}
				if got := h.scr.Text(); got != want {
					t.Errorf("the screen and scrollback are\n%s\nwant\n%s", got, want)
				}
				if h.scr.Col != 5 || h.scr.Lines()[h.scr.Row] != "ijklm" {
					t.Errorf("the cursor is at %d,%d", h.scr.Row, h.scr.Col)
				}
			})
		}
	}
}

// TestResizeRowsAbove: with the cursor on the region's second row, the
// rows above it count as the terminal re-wrapped them, or as cut.
func TestResizeRowsAbove(t *testing.T) {
	want := "above\n> abcdefgh\nijklmnop\n  qrstu\n"
	for _, how := range []inline.Reflow{inline.ReflowDown, inline.ReflowKeepRow, inline.ReflowCut} {
		t.Run(how.String(), func(t *testing.T) {
			h := newHost(t, 20, 8, ansi.WcWidth, inline.WithReflow(how))
			h.write("above\r\n")
			h.write(h.r.Draw([]string{"> abcdefghijklmnop", "  qrstu"}, 1, 7))
			h.scr.Resize(10, how)
			h.write(h.r.Resize(10, h.now))
			h.answer()
			h.write(h.r.Draw([]string{"> abcdefgh", "ijklmnop", "  qrstu"}, 2, 7))
			if got := h.scr.Text(); got != want {
				t.Errorf("the screen and scrollback are\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// TestKeysWhileAwaited: keys typed while the report is awaited reach the
// component in order, its drawings wait, and the report's redraw shows the
// last.
func TestKeysWhileAwaited(t *testing.T) {
	h := newHost(t, 20, 6, ansi.WcWidth)
	h.write(h.r.Draw([]string{"> ab"}, 0, 4))
	h.write(h.r.Resize(10, h.now))
	h.scr.Resize(10, inline.ReflowDown)
	h.feed([]byte("cd"))
	var typed string
	for _, k := range h.keys {
		typed += k.Key.String()
	}
	if typed != "cd" {
		t.Fatalf("the component got %q; want c then d", typed)
	}
	if out := h.r.Draw([]string{"> abcd"}, 0, 6); out != "" {
		t.Fatalf("a draw while the report is awaited wrote %q", out)
	}
	if out := h.r.Print("printed"); out != "" {
		t.Fatalf("a print while the report is awaited wrote %q", out)
	}
	h.answer()
	h.wantLines("after the report", "printed", "> abcd")
}

// TestRow1Report: CSI 1 ; 5 R is the report on the first row while one is
// awaited, and the key ctrl+f3 otherwise.
func TestRow1Report(t *testing.T) {
	h := newHost(t, 20, 4, ansi.WcWidth)
	h.write(h.r.Draw([]string{"> ab"}, 0, 4))
	if out := h.r.Resize(10, h.now); !strings.Contains(out, "\x1b[6n") {
		t.Fatalf("the resize wrote %q", out)
	}
	h.feed([]byte("\x1b[1;5R"))
	if len(h.keys) != 0 {
		t.Errorf("an awaited report on row 1 reached the component as %v", h.keys)
	}
	if _, waiting := h.r.Timeout(); waiting {
		t.Errorf("the region still waits after the report")
	}
	h.feed([]byte("\x1b[1;5R"))
	if len(h.keys) != 1 || h.keys[0].Kind != termkey.InputKey || h.keys[0].Key.String() != "ctrl+f3" {
		t.Errorf("CSI 1 ; 5 R with no report awaited gave %v; want the key ctrl+f3", h.keys)
	}
}

// TestLateReport: a report that comes after its wait is dropped, until
// another input comes.
func TestLateReport(t *testing.T) {
	h := newHost(t, 20, 4, ansi.WcWidth)
	h.write(h.r.Draw([]string{"> ab"}, 0, 4))
	h.write(h.r.Resize(10, h.now))
	if out := h.r.HandleTimeout(h.now.Add(inline.DefaultReportWait - time.Millisecond)); out != "" {
		t.Fatalf("the wait ended early: %q", out)
	}
	h.write(h.r.HandleTimeout(h.now.Add(inline.DefaultReportWait)))
	h.answer()
	if len(h.keys) != 0 {
		t.Errorf("a late report reached the component as %v", h.keys)
	}
	h.feed([]byte("x\x1b[1;5R"))
	if len(h.keys) != 2 || h.keys[1].Key.String() != "ctrl+f3" {
		t.Errorf("after another input: %v; want x and ctrl+f3", h.keys)
	}
}

// TestWithoutCursorQuery: a resize sends no CSI 6 n, and redraws at once.
func TestWithoutCursorQuery(t *testing.T) {
	h := newHost(t, 20, 4, ansi.WcWidth, inline.WithoutCursorQuery())
	h.write(h.r.Draw([]string{"> ab"}, 0, 4))
	out := h.r.Resize(10, h.now)
	if strings.Contains(out, "\x1b[6n") || !strings.Contains(out, "> ab") {
		t.Errorf("the resize wrote %q; want the redraw and no query", out)
	}
	if _, waiting := h.r.Timeout(); waiting {
		t.Errorf("the region waits for a report it did not ask for")
	}
}

// TestReportWait: the wait is DefaultReportWait, or WithReportWait's when
// it is positive.
func TestReportWait(t *testing.T) {
	now := time.Unix(1, 0)
	for _, c := range []struct {
		o    []inline.Option
		want time.Duration
	}{
		{nil, 150 * time.Millisecond},
		{[]inline.Option{inline.WithReportWait(300 * time.Millisecond)}, 300 * time.Millisecond},
		{[]inline.Option{inline.WithReportWait(0)}, 150 * time.Millisecond},
		{[]inline.Option{inline.WithReportWait(-time.Second)}, 150 * time.Millisecond},
	} {
		r := inline.NewRegion(c.o...)
		r.Resize(20, now)
		r.Draw([]string{"x"}, 0, 1)
		r.Resize(10, now)
		if at, ok := r.Timeout(); !ok || at.Sub(now) != c.want {
			t.Errorf("the wait ends after %v, %v; want %v", at.Sub(now), ok, c.want)
		}
	}
}

// TestHeight: a region taller than its height shows the rows around the
// cursor's; SetHeight changes it.
func TestHeight(t *testing.T) {
	h := newHost(t, 20, 8, ansi.WcWidth, inline.WithHeight(2))
	rows := []string{"r0", "r1", "r2", "r3"}
	h.write(h.r.Draw(rows, 3, 2))
	h.wantLines("height 2", "r2", "r3")
	h.r.SetHeight(3)
	h.write(h.r.Draw(rows, 3, 2))
	h.wantLines("height 3", "r1", "r2", "r3")
	h.r.SetHeight(0)
	h.write(h.r.Draw(rows, 0, 2))
	h.wantLines("every row", "r0", "r1", "r2", "r3")
}

// TestUnsynchronized: WithSynchronized(false) writes no mode 2026 and no
// DECTCEM, and Restore has nothing to undo.
func TestUnsynchronized(t *testing.T) {
	r := inline.NewRegion(inline.WithSynchronized(false))
	r.Resize(20, time.Unix(1, 0))
	if out := r.Draw([]string{"x"}, 0, 1); strings.Contains(out, "\x1b[?") {
		t.Errorf("an unsynchronized draw wrote %q", out)
	}
	if got := r.Restore(); got != "" {
		t.Errorf("Restore = %q; want nothing", got)
	}
	if got := inline.NewRegion().Restore(); got != "\x1b[?2026l\x1b[?25h" {
		t.Errorf("Restore = %q; want mode 2026 off and the cursor shown", got)
	}
}

// TestWiderRow: a row wider than the width is cut to it.
func TestWiderRow(t *testing.T) {
	h := newHost(t, 10, 3, ansi.WcWidth)
	h.write(h.r.Draw([]string{"0123456789abc", "x"}, 1, 1))
	h.wantLines("a row too wide", "0123456789", "x")
}
