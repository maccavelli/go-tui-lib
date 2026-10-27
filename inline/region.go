package inline

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// The sequences a region writes. A drawing is one synchronized update
// (mode 2026) with the cursor hidden while it moves (DECTCEM).
const (
	syncOn      = "\x1b[?2026h"
	syncOff     = "\x1b[?2026l"
	cursorHide  = "\x1b[?25l"
	cursorShow  = "\x1b[?25h"
	cursorQuery = "\x1b[6n"
	eraseBelow  = "\x1b[J"
	eraseRight  = "\x1b[K"
	resetStyle  = "\x1b[m"
)

// Region is the rows a component draws below the terminal's scrollback,
// and redraws in place (docs/decisions/0023-MADR-line-mode-components.md
// D3). Each method returns the bytes to write to the terminal, which the
// host writes in order; a Region writes nothing itself, reads no clock and
// starts no goroutine. One goroutine drives it.
//
// It draws by relative moves only, from where the cursor stands: an
// unchanged row is skipped, a changed one rewritten from what it shares
// with the row before, never erased first, and every drawing is one
// synchronized update with the cursor hidden, unless WithSynchronized
// turns that off. Give it the terminal's width with Resize before the
// first Draw; a row wider than the width is cut to it.
type Region struct {
	method  ansi.Method
	sync    bool
	reflow  Reflow
	noQuery bool
	wait    time.Duration
	height  int

	width int // the terminal's, in cells; 0 until Resize gives it

	drawn []string // the rows on the screen as drawn; nil when none are
	at    int      // the cursor's row among drawn
	col   int      // the cursor's column, in cells
	up    int      // for the next first drawing: the rows from the cursor up to the region's first row

	// rows, row and rcol are the last Draw's, for a redraw after a resize
	// or a Print; has says there are some.
	rows      []string
	row, rcol int
	has       bool

	awaiting bool      // a resize waits for the cursor report
	deadline time.Time // when it stops waiting
	queries  int       // cursor queries sent and not answered
	held     []string  // what Print was given while a report was awaited
}

// NewRegion returns an empty region, measuring with ansi.WcWidth, each
// drawing synchronized, a resize asking for the cursor report.
func NewRegion(o ...Option) *Region {
	r := &Region{method: ansi.WcWidth, sync: true, wait: DefaultReportWait}
	for _, f := range o {
		f.apply(r)
	}
	return r
}

// SetHeight sets the most rows the region shows, as WithHeight does, for a
// host whose terminal changed height. It writes nothing: the next Draw
// shows the rows around the cursor's.
func (r *Region) SetHeight(rows int) { r.height = max(rows, 0) }

// Draw returns the bytes that show rows in place of the rows drawn last,
// with the cursor at row and col, in cells. While a resize waits for the
// cursor report it returns nothing, and the report's redraw shows these
// rows.
func (r *Region) Draw(rows []string, row, col int) string {
	r.rows, r.row, r.rcol, r.has = slices.Clone(rows), row, col, true
	if r.awaiting {
		return ""
	}
	return r.wrap(r.paint(""))
}

// Print returns the bytes that write text above the region and draw the
// region again below it, as x/term's Terminal.Write does: a streaming
// reply prints above a composer so. Its line ends are written as CRLF, and
// it ends on a line of its own. While a resize waits for the cursor
// report, the text waits too, and the report's redraw writes it.
func (r *Region) Print(text string) string {
	t := lines(text)
	if r.awaiting {
		r.held = append(r.held, t)
		return ""
	}
	if r.drawn == nil {
		return r.wrap(t)
	}
	r.up, r.drawn = r.at, nil
	return r.wrap(r.paint(t))
}

// Commit returns the bytes that leave rows in the scrollback, whole, and
// the cursor at the start of the row below them: the region is empty, and
// the next Draw starts there.
func (r *Region) Commit(rows []string) string {
	var b strings.Builder
	b.WriteString(r.lift())
	b.WriteString(r.flushHeld())
	for _, row := range rows {
		row = r.fit(row)
		b.WriteString(row)
		if styled(row) {
			b.WriteString(resetStyle)
		}
		b.WriteString("\r\n")
	}
	r.forget()
	return r.wrap(b.String())
}

// Clear returns the bytes that remove the region: the cursor is left where
// its first row was.
func (r *Region) Clear() string {
	s := r.lift() + r.flushHeld()
	r.forget()
	return r.wrap(s)
}

// lift returns the bytes that move the cursor to the region's first row
// and erase from there down, ending a wait for the cursor report by
// WithReflow's kind. With nothing drawn it returns nothing.
func (r *Region) lift() string {
	if r.awaiting {
		r.up, r.drawn, r.awaiting = r.liftRows(0, false), nil, false
	} else if r.drawn != nil {
		r.up, r.drawn = r.at, nil
	}
	if r.up == 0 && !r.has {
		return ""
	}
	var b strings.Builder
	if r.up > 0 {
		b.WriteString("\x1b[" + strconv.Itoa(r.up) + "A")
	}
	b.WriteString("\r" + eraseBelow)
	r.up = 0
	return b.String()
}

// flushHeld returns what Print held while a report was awaited.
func (r *Region) flushHeld() string {
	s := strings.Join(r.held, "")
	r.held = nil
	return s
}

// forget empties the region: the next Draw is a first one, from the
// cursor's row.
func (r *Region) forget() {
	r.drawn, r.at, r.col, r.up = nil, 0, 0, 0
	r.rows, r.row, r.rcol, r.has = nil, 0, 0, false
}

// Resize returns the bytes for a terminal now width cells wide: with rows
// on the screen, the cursor query (CSI 6 n), whose report HandleInput
// reads, or the redraw at once under WithoutCursorQuery. A width of 0 or
// less, or the width the region has, changes nothing.
func (r *Region) Resize(width int, now time.Time) string {
	if width <= 0 || (width == r.width && !r.awaiting) {
		return ""
	}
	r.width = width
	if r.drawn == nil {
		return ""
	}
	if r.noQuery {
		return r.redraw(0, false)
	}
	r.awaiting, r.deadline = true, now.Add(r.wait)
	r.queries++
	return cursorQuery
}

// HandleInput takes the cursor report a resize asked for out of the input:
// ok is true when in was one, and the bytes are the redraw. Every other
// input, and a cursor report nobody asked for, is the component's: ok is
// false. The report on the first row, which termkey reads as a modified F3
// (CSI 1 ; mod R), is taken as a report while one is asked for. A report
// that arrives after its wait ended is taken and dropped, until another
// input arrives.
func (r *Region) HandleInput(in termkey.Input, now time.Time) (string, bool) {
	if r.queries == 0 {
		return "", false
	}
	rep, ok := cursorReport(in)
	if !ok {
		if !r.awaiting {
			r.queries = 0 // the late reports did not come
		}
		return "", false
	}
	r.queries--
	if !r.awaiting || r.queries > 0 {
		return "", true
	}
	return r.redraw(rep.Col, true), true
}

// cursorReport is in as a cursor position report, if it is one.
func cursorReport(in termkey.Input) (termcap.Reply, bool) {
	if in.Kind != termkey.InputKey && in.Kind != termkey.InputUnknown {
		return termcap.Reply{}, false
	}
	rep := termcap.ParseReply(string(in.Raw))
	return rep, rep.Kind == termcap.ReplyCursorPosition
}

// Timeout is when the wait for a cursor report ends, while there is one.
func (r *Region) Timeout() (time.Time, bool) { return r.deadline, r.awaiting }

// HandleTimeout returns the redraw by WithReflow's kind once the wait for
// the cursor report has ended at now; before then, or with no wait, it
// returns nothing.
func (r *Region) HandleTimeout(now time.Time) string {
	if !r.awaiting || now.Before(r.deadline) {
		return ""
	}
	return r.redraw(0, false)
}

// Restore returns the bytes that undo what the region may have left set
// had its drawing been cut short: the cursor shown, and mode 2026 off. It
// is launch.Restorer's shape.
func (r *Region) Restore() string {
	if !r.sync {
		return ""
	}
	return syncOff + cursorShow
}

// redraw draws the last rows again after a resize, from where the
// region's first row now is: col is the cursor report's column, from 1,
// when known.
func (r *Region) redraw(col int, known bool) string {
	r.up, r.drawn, r.awaiting = r.liftRows(col, known), nil, false
	return r.wrap(r.paint(r.flushHeld()))
}

// liftRows is how many rows above the cursor the region's first row is now
// that the terminal is r.width cells wide (gobble-cli's
// 0030-MADR-resize-and-performance.md D2, as amended). A terminal re-wraps
// each drawn row at the new width, unless it cuts them; the report's
// column tells which when the cursor's column was past the new width, and
// WithReflow's kind when it does not. A terminal that keeps the cursor's
// row is redrawn as a re-wrap: moving up stops at the screen's first row.
func (r *Region) liftRows(col int, known bool) int {
	w := r.width
	if r.drawn == nil || w <= 0 {
		return 0
	}
	cut := r.reflow == ReflowCut
	if known && r.col >= w && r.col%w != w-1 {
		cut = col-1 == w-1
	}
	if cut {
		return r.at
	}
	up := r.col / w
	for _, d := range r.drawn[:r.at] {
		up += max((r.method.StringWidth(d)+w-1)/w, 1)
	}
	return up
}

// paint returns the moves and text that show the last rows in place of
// the rows drawn, unwrapped; a first drawing writes prefix, a Print's
// text, after it erases.
func (r *Region) paint(prefix string) string {
	rows, at, col := r.window()
	var b strings.Builder
	first := r.drawn == nil
	cur := r.at
	if first {
		if r.up > 0 {
			b.WriteString("\x1b[" + strconv.Itoa(r.up) + "A")
		}
		b.WriteString("\r" + eraseBelow + prefix)
		cur = 0
	}
	r.up = 0
	for i, row := range rows {
		was := ""
		if !first && i < len(r.drawn) {
			if was = r.drawn[i]; was == row {
				continue
			}
		}
		cur = moveTo(&b, cur, i)
		r.writeRow(&b, was, row)
	}
	if !first && len(r.drawn) > len(rows) {
		cur = moveTo(&b, cur, len(rows))
		b.WriteString("\r" + eraseBelow)
	}
	if b.Len() > 0 || cur != at || r.col != col {
		moveTo(&b, cur, at)
		b.WriteByte('\r')
		if col > 0 {
			b.WriteString("\x1b[" + strconv.Itoa(col) + "C")
		}
	}
	if rows == nil {
		rows = []string{}
	}
	r.drawn, r.at, r.col = rows, at, col
	return b.String()
}

// window is the last rows cut to the width, and the cursor among them:
// with a height, the rows around the cursor's.
func (r *Region) window() (rows []string, at, col int) {
	rows = make([]string, len(r.rows))
	for i, row := range r.rows {
		rows[i] = r.fit(row)
	}
	at = min(max(r.row, 0), max(len(rows)-1, 0))
	col = max(r.rcol, 0)
	if r.width > 0 {
		col = min(col, r.width-1)
	}
	if h := r.height; h > 0 && len(rows) > h {
		from := min(max(at-h+1, 0), len(rows)-h)
		rows, at = rows[from:from+h], at-from
	}
	return rows, at, col
}

// fit is row cut to the width, when the width is known.
func (r *Region) fit(row string) string {
	if r.width <= 0 {
		return row
	}
	return sanitize.Truncate(row, r.width, r.method, "")
}

// writeRow writes row over was, from the start of the row the cursor is
// on: the shared part is skipped by a move, the styles in force there are
// set again, and what is left of a longer was is erased.
func (r *Region) writeRow(b *strings.Builder, was, row string) {
	n, escapes := r.shared(was, row)
	b.WriteByte('\r')
	if cells := r.method.StringWidth(row[:n]); cells > 0 {
		b.WriteString("\x1b[" + strconv.Itoa(cells) + "C")
	}
	b.WriteString(escapes)
	b.WriteString(row[n:])
	if styled(row) {
		b.WriteString(resetStyle)
	}
	if r.method.StringWidth(row) < r.method.StringWidth(was) {
		b.WriteString(eraseRight)
	}
}

// shared is how many bytes a and b start with alike, cut between whole
// escape sequences and grapheme clusters, and the escape sequences among
// them, which set the styles in force where it ends.
func (r *Region) shared(a, b string) (n int, escapes string) {
	var sa, sb byte
	var esc strings.Builder
	for n < len(a) && n < len(b) {
		seqA, _, na, nextA := r.method.DecodeSequenceInString(a[n:], sa, nil)
		seqB, _, nb, nextB := r.method.DecodeSequenceInString(b[n:], sb, nil)
		if na == 0 || na != nb || seqA != seqB {
			break
		}
		if escape(seqA) {
			esc.WriteString(seqA)
		}
		n += na
		sa, sb = nextA, nextB
	}
	return n, esc.String()
}

// escape reports whether seq is an escape sequence, by its 7-bit or 8-bit
// introducer.
func escape(seq string) bool {
	return seq != "" && (seq[0] == ansi.ESC || (seq[0] >= 0x80 && seq[0] <= 0x9f))
}

// styled reports whether row holds an escape sequence, whose styles a
// reset ends. Cleaned text holds only 7-bit ones.
func styled(row string) bool { return strings.IndexByte(row, ansi.ESC) >= 0 }

// moveTo moves the cursor from row from of the region to row to: up by a
// sequence, down by line feeds, which scroll when the region grows past
// the screen's last row. The column is left to the caller.
func moveTo(b *strings.Builder, from, to int) int {
	switch {
	case to < from:
		b.WriteString("\x1b[" + strconv.Itoa(from-to) + "A")
	case to > from:
		b.WriteString(strings.Repeat("\n", to-from))
	}
	return to
}

// wrap makes s one synchronized update with the cursor hidden, unless the
// region draws unsynchronized; nothing stays nothing.
func (r *Region) wrap(s string) string {
	if s == "" || !r.sync {
		return s
	}
	return syncOn + cursorHide + s + cursorShow + syncOff
}

// lines is text with its line ends written as CRLF, ending on a line of
// its own, and its styles reset when it holds any.
func lines(text string) string {
	if text == "" {
		return ""
	}
	t := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
	if styled(t) {
		t = strings.TrimSuffix(t, "\r\n") + resetStyle + "\r\n"
	}
	if !strings.HasSuffix(t, "\r\n") {
		t += "\r\n"
	}
	return t
}
