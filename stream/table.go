package stream

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// cells splits a table row on its unescaped pipes, without its outer pipes,
// trimming each cell (GFM 0.29-gfm §4.10): a row has them or not.
func cells(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, `\|`) {
		s = s[:len(s)-1]
	}
	var out []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '|':
			cur.WriteString(`\|`)
			i++
		case s[i] == '|':
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(s[i])
		}
	}
	return append(out, strings.TrimSpace(cur.String()))
}

// delimiterRow reports a table's delimiter row: cells of hyphens, each with
// a colon or not at either end, and at least one pipe, so a lone run of
// hyphens stays a setext underline.
func delimiterRow(line string) bool {
	if !strings.Contains(line, "|") {
		return false
	}
	for _, c := range cells(line) {
		c = strings.TrimSuffix(strings.TrimPrefix(c, ":"), ":")
		if c == "" || strings.Trim(c, "-") != "" {
			return false
		}
	}
	return true
}

// tableStart reports whether head, a paragraph's line, and delim start a
// table: delim is a delimiter row with as many cells as head (§4.10: "If
// not, a table will not be recognized").
func tableStart(head, delim string) bool {
	return delimiterRow(delim) && len(cells(head)) == len(cells(delim))
}

type align int

const (
	alignLeft align = iota
	alignCenter
	alignRight
)

func alignOf(delim string) align {
	left, right := strings.HasPrefix(delim, ":"), strings.HasSuffix(delim, ":")
	switch {
	case left && right:
		return alignCenter
	case right:
		return alignRight
	}
	return alignLeft
}

// layoutTable lays rows out, the header first and the delimiter row second:
// cells padded to their column's width under m and aligned as the delimiter
// row's colons say, a row of fewer cells padded and one of more cut, as
// §4.10 reads them (gobble-cli's render/table.go, its rows with or without
// outer pipes). The rows have no line break; the Printer gives each its
// own.
func layoutTable(m ansi.Method, rows []string) []string {
	delim := cells(rows[1])
	cols := len(delim)
	aligns := make([]align, cols)
	widths := make([]int, cols)
	for c, d := range delim {
		aligns[c] = alignOf(d)
		widths[c] = 3
	}
	grid := make([][]string, 0, len(rows)-1)
	for i, r := range rows {
		if i == 1 {
			continue
		}
		cs := cells(r)
		cs = append(cs, make([]string, max(0, cols-len(cs)))...)[:cols]
		for c, v := range cs {
			widths[c] = max(widths[c], m.StringWidth(v))
		}
		grid = append(grid, cs)
	}
	var out []string
	for i, cs := range grid {
		out = append(out, tableRow(m, cs, widths, aligns))
		if i == 0 {
			out = append(out, tableDelimiter(widths, aligns))
		}
	}
	return out
}

func tableRow(m ansi.Method, cs []string, widths []int, aligns []align) string {
	var b strings.Builder
	b.WriteString("|")
	for c, v := range cs {
		pad := widths[c] - m.StringWidth(v)
		var l, r int
		switch aligns[c] {
		case alignRight:
			l = pad
		case alignCenter:
			l, r = pad/2, pad-pad/2
		default:
			r = pad
		}
		b.WriteString(" " + strings.Repeat(" ", l) + v + strings.Repeat(" ", r) + " |")
	}
	return b.String()
}

func tableDelimiter(widths []int, aligns []align) string {
	var b strings.Builder
	b.WriteString("|")
	for c, w := range widths {
		d := strings.Repeat("-", w)
		switch aligns[c] {
		case alignRight:
			d = d[:w-1] + ":"
		case alignCenter:
			d = ":" + d[:w-2] + ":"
		}
		b.WriteString(" " + d + " |")
	}
	return b.String()
}
