package lineedit

import (
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// place is where the cursor stands on the screen: its row among the text's
// rows, and its column.
type place struct{ row, col int }

// laid is the text as it is drawn: its rows, the place of the cursor at
// each cluster index from 0 to the text's end, and the indent of a row
// below the last, where the cursor at the text's end stands when the last
// row is full. For each row, starts holds its first cluster, bases its
// line's first cluster, and reach the last cluster the row depends on: the
// last its break read, or the next row's start when that is later, or for
// a line's last row the line's end. So a change lays out again only the
// rows that reach it.
type laid struct {
	rows   []string
	places []place
	starts []int
	reach  []int
	bases  []int // each row's line's first cluster
	pad    string

	pads     [2]string // the indents under the prompt and under cont, once made
	old      oldRows   // the rows after the restart, kept for resync
	syncFrom int       // the first cluster of the tail no edit touched
}

// oldRows is the rows a relay laid out again, as they were: the first was
// row firstRow, its first place the place of cluster placeBase, of a text
// of n clusters.
type oldRows struct {
	rows      []string
	places    []place
	starts    []int
	reach     []int
	bases     []int
	n         int
	firstRow  int
	placeBase int
}

// layout lays the text's clusters out in rows of at most width cells,
// gobble's layout (its 0027-MADR D2) by cluster: the first line after
// prompt, each after a line break after cont, and each line's rows after
// the first indented by its prompt's width. A row ends at the last space
// it reaches, which the break takes and the row does not draw; a word
// wider than a whole row is broken at the row's end, so its pieces are on
// rows of their own (0023-MADR A5 item 5). A cursor that would stand past
// the last column goes to the next row's start: at the text's end, a row
// below the last, which the drawing adds only while the cursor is there;
// at a line break's, the last column, so the rows never depend on the
// cursor. A row's lead and its first cluster are measured together, as
// they are drawn, so a mark with no base that joins the lead's space is
// counted where it lands (0023-MADR A5 item 7); the text's clusters, cut
// from it whole, join nothing. ws is each cluster's width, as the Line
// measured it. A width of 0 or less is one row a line.
func layout(m ansi.Method, width int, prompt, cont string, text []string, ws []int) laid {
	var out laid
	out.relay(m, width, prompt, cont, text, ws, 0, 0)
	return out
}

// relay lays out again the rows of l from the first one whose break looked
// at cluster from or past it, keeping the rows before it, their places and
// their strings, which no change at from or after can move (0023-MADR
// A12). from 0 lays out everything. tail is how many clusters at the
// end are as they were: once a row laid out again starts inside them where
// an old row started, under the same lead, the rows from there are the
// old ones, their places shifted (see sync). The rows are gobble's layout
// (its 0027-MADR D2) by cluster, as before.
func (l *laid) relay(m ansi.Method, width int, prompt, cont string, text []string, ws []int, from, tail int) {
	r := 0
	for r < len(l.reach) && l.reach[r] < from {
		r++
	}
	start := len(text) + 1
	if r < len(l.starts) {
		start = l.starts[r]
	}
	if r == 0 {
		start = 0
	}
	start = min(start, len(l.places))
	o := &l.old
	o.n, o.firstRow, o.placeBase = len(l.places)-1, r, start
	o.rows, o.places = append(o.rows[:0], l.rows[r:]...), append(o.places[:0], l.places[start:]...)
	o.starts, o.reach, o.bases = append(o.starts[:0], l.starts[r:]...), append(o.reach[:0], l.reach[r:]...), append(o.bases[:0], l.bases[r:]...)
	if o.n < 0 {
		tail = 0
	}
	base := 0
	if r < len(l.bases) {
		base = l.bases[r]
	}
	l.rows, l.starts, l.reach, l.bases, l.places = l.rows[:r], l.starts[:r], l.reach[:r], l.bases[:r], l.places[:start]
	l.syncFrom = len(text) - tail
	lead, which := prompt, 0
	if base > 0 {
		lead, which = cont, 1
	}
	for {
		end := base
		for end < len(text) && text[end] != "\n" {
			end++
		}
		final := end == len(text)
		if l.layLine(m, width, lead, which, text, ws, base, end, start, final) || final {
			return
		}
		base, lead, which = end+1, cont, 1
		start = base
	}
}

// layLine adds the rows of the line text[base:end] from cluster first on,
// and their places, from the place before first to the place before the
// line break, or the end, after it.
func (l *laid) layLine(m ansi.Method, width int, lead string, which int, text []string, ws []int, base, end, first int, final bool) bool {
	line, lw := text[base:end], ws[base:end]
	indent := sanitize.StringWidth(m, lead)
	if len(l.pads[which]) != indent {
		l.pads[which] = strings.Repeat(" ", indent)
	}
	pad := l.pads[which]
	if final {
		l.pad = pad
	}
	for start := first - base; ; {
		if l.sync(base+start, len(text), base) {
			return true
		}
		head := pad
		if start == 0 {
			head = lead
		}
		with := indent // the head with the row's first cluster, measured together
		if start < len(line) {
			with = headWidth(m, head, indent, line[start], lw[start])
		}
		stop0, next, fit := rowEnd(line, lw, start, with, width)
		row := len(l.rows)
		last := stop0 == len(line) && next == len(line)
		// A row depends on the clusters it measured, and on whether the line
		// ends where the next row starts: the line's last row, whose next is
		// the line's end, holds the places to its end.
		fit = max(fit, next)
		l.rows = append(l.rows, rowText(head, line[start:stop0]))
		l.starts = append(l.starts, base+start)
		l.reach = append(l.reach, base+fit)
		l.bases = append(l.bases, base)
		stop := next
		if last {
			stop = len(line) + 1
		}
		col := indent
		for p := start; p < stop; p++ {
			switch {
			case p == start+1 && p <= stop0:
				col = with
			case p > start && p <= stop0:
				col += cellWidth(line[p-1], lw[p-1])
			}
			if width > 0 && col >= width {
				if p == len(line) && !final {
					l.places = append(l.places, place{row, width - 1})
				} else {
					l.places = append(l.places, place{row + 1, indent})
				}
				continue
			}
			l.places = append(l.places, place{row, col})
		}
		if last {
			return false
		}
		start = next
	}
}

// sync keeps the old rows from the one that started where a row would now
// start, at s, shifted by the change in the text's length, when s is past
// the first cluster of the tail no edit touched, so that the row and the
// one before it read the same clusters as before, and the row's line has
// the same lead. A row's break and places depend only on its own clusters,
// its line's lead and the line's end, so each old row from there is the
// row the layout would make, and each place moves by the shift
// (0023-MADR A12).
func (l *laid) sync(s, n, base int) bool {
	o := &l.old
	if s <= l.syncFrom || len(o.starts) == 0 {
		return false
	}
	shift := n - o.n
	k, ok := slices.BinarySearch(o.starts, s-shift)
	if !ok || (o.bases[k] == 0) != (base == 0) {
		return false
	}
	rowShift := len(l.rows) - (o.firstRow + k)
	l.rows = append(l.rows, o.rows[k:]...)
	for i := k; i < len(o.starts); i++ {
		l.starts = append(l.starts, o.starts[i]+shift)
		l.reach = append(l.reach, o.reach[i]+shift)
	}
	// The row's own line may start before the change: its rows take the
	// line's start as laid now; a later line's moves by the shift.
	for _, b := range o.bases[k:] {
		if b == o.bases[k] {
			l.bases = append(l.bases, base)
		} else {
			l.bases = append(l.bases, b+shift)
		}
	}
	for _, pl := range o.places[o.starts[k]-o.placeBase:] {
		l.places = append(l.places, place{pl.row + rowShift, pl.col})
	}
	return true
}

// headWidth is head's width with cluster c after it, measured together, as
// a mark with no base joins the head's last space (0023-MADR A5 item 7). A
// cluster that starts with printable ASCII joins nothing before it, so its
// width is added.
func headWidth(m ansi.Method, head string, w int, c string, cw int) int {
	if c[0] >= 0x20 && c[0] <= 0x7e {
		return w + cellWidth(c, cw)
	}
	return sanitize.StringWidth(m, head+c)
}

// rowEnd is where the row that starts at start ends in line, where the next
// row starts, and fit, the last cluster it looked at: at the last space the
// width reaches, the space taken by the break, or with no such space at
// the width, cutting the word; a cluster wider than the room is a row of
// its own. with is the row's head and first cluster, measured together.
func rowEnd(line []string, lw []int, start, with, width int) (end, next, fit int) {
	if width <= 0 {
		return len(line), len(line), len(line)
	}
	fit, used := start, 0
	if fit < len(line) {
		if used = with; used <= width {
			fit++
		}
	}
	for fit > start && fit < len(line) {
		w := cellWidth(line[fit], lw[fit])
		if used+w > width {
			break
		}
		used += w
		fit++
	}
	if fit == len(line) {
		return fit, fit, fit
	}
	for k := fit; k > start; k-- {
		if line[k] == " " {
			return k, k + 1, fit
		}
	}
	return max(fit, start+1), max(fit, start+1), fit
}

// cellWidth is the cells cluster c takes, w as the Line measured it: a tab
// is drawn as one space.
func cellWidth(c string, w int) int {
	if c == "\t" {
		return 1
	}
	return w
}

// rowText is head and the clusters as a row draws them, a tab as one
// space, in one allocation.
func rowText(head string, clusters []string) string {
	n := len(head)
	for _, c := range clusters {
		n += len(c)
	}
	var b strings.Builder
	b.Grow(n)
	b.WriteString(head)
	for _, c := range clusters {
		if c == "\t" {
			c = " "
		}
		b.WriteString(c)
	}
	return b.String()
}

// nearest is the cluster index on row whose column is col, or the last one
// before it, or the row's first.
func nearest(places []place, row, col int) int {
	best := -1
	for p, pl := range places {
		if pl.row != row {
			continue
		}
		if best == -1 || pl.col <= col {
			best = p
		}
	}
	return best
}
