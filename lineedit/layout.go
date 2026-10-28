package lineedit

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// place is where the cursor stands on the screen: its row among the text's
// rows, and its column.
type place struct{ row, col int }

// laid is the text as it is drawn: its rows, the place of the cursor at
// each cluster index from 0 to the text's end, and the indent of a row
// below the last, where the cursor at the text's end stands when the last
// row is full.
type laid struct {
	rows   []string
	places []place
	pad    string
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
// from it whole, join nothing. A width of 0 or less is one row a line.
func layout(m ansi.Method, width int, prompt, cont string, text []string) laid {
	var out laid
	base := 0
	lead := prompt
	for {
		end := base
		for end < len(text) && text[end] != "\n" {
			end++
		}
		final := end == len(text)
		layLine(&out, m, width, lead, text[base:end], final)
		if final {
			return out
		}
		base, lead = end+1, cont
	}
}

// layLine adds one line's rows and its places, from the place before its
// first cluster to the place before the line break, or the end, after it.
func layLine(out *laid, m ansi.Method, width int, lead string, line []string, final bool) {
	indent := m.StringWidth(lead)
	pad := strings.Repeat(" ", indent)
	if final {
		out.pad = pad
	}
	for start := 0; ; {
		head := pad
		if start == 0 {
			head = lead
		}
		end, next := rowEnd(m, line, start, head, width)
		row := len(out.rows)
		out.rows = append(out.rows, head+draw(line[start:end]))
		last := end == len(line) && next == len(line)
		stop := next
		if last {
			stop = len(line) + 1
		}
		col := m.StringWidth(head)
		for p := start; p < stop; p++ {
			switch {
			case p == start+1 && p <= end:
				col = m.StringWidth(head + draw(line[start:p]))
			case p > start && p <= end:
				col += cellWidth(m, line[p-1])
			}
			if width > 0 && col >= width {
				if p == len(line) && !final {
					out.places = append(out.places, place{row, width - 1})
				} else {
					out.places = append(out.places, place{row + 1, indent})
				}
				continue
			}
			out.places = append(out.places, place{row, col})
		}
		if last {
			return
		}
		start = next
	}
}

// rowEnd is where the row that starts at start, after head, ends in line,
// and where the next row starts: at the last space the width reaches, the
// space taken by the break, or with no such space at the width, cutting
// the word; a cluster wider than the room is a row of its own.
func rowEnd(m ansi.Method, line []string, start int, head string, width int) (end, next int) {
	if width <= 0 {
		return len(line), len(line)
	}
	fit, used := start, 0
	if fit < len(line) {
		if used = m.StringWidth(head + draw(line[start:start+1])); used <= width {
			fit++
		}
	}
	for fit > start && fit < len(line) && used+cellWidth(m, line[fit]) <= width {
		used += cellWidth(m, line[fit])
		fit++
	}
	if fit == len(line) {
		return fit, fit
	}
	for k := fit; k > start; k-- {
		if line[k] == " " {
			return k, k + 1
		}
	}
	return max(fit, start+1), max(fit, start+1)
}

// cellWidth is the cells cluster c takes: a tab is drawn as one space.
func cellWidth(m ansi.Method, c string) int {
	if c == "\t" {
		return 1
	}
	return m.StringWidth(c)
}

// draw is clusters as a row draws them, a tab as one space.
func draw(clusters []string) string {
	var b strings.Builder
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
