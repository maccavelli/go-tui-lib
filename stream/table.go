package stream

import (
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
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

// TableOptions is how FitTable lays a table out
// (docs/decisions/0009-PLAN-streaming-content-engine.md D19).
type TableOptions struct {
	// Width is the cells the table may take; 0 or less, no limit, each
	// column as wide as its widest cell.
	Width int
	// Method measures the cells.
	Method ansi.Method
	// Border draws the table as a box of its glyphs, such as glyph.Set's
	// Light, whose ASCII twin glyph.ASCII() has; nil draws it in the
	// Markdown form, pipes and a delimiter row, as Plain does.
	Border *glyph.Border
}

// FitTable lays out a closed table, rows its source lines: the header, the
// delimiter row and the body (docs/decisions/0009-MADR-streaming-content-engine.md
// A1 §4). A row of fewer cells is padded and one of more cut, as GFM's
// §4.10 reads them. Each column starts at its widest cell and shrinks no
// further than its longest word, the space left shared in proportion to
// what each still wants; a cell wraps at its spaces and no word is broken,
// so numbers and URLs stay whole; each row is padded by grapheme cluster
// under Method, so a wide glyph leaves no ghost cell. When the longest
// words do not fit, the rows are key/value records, "header: value", a
// blank line between records; when a pair does not fit one line, stacked
// records, the header on its line and the value under it indented two
// cells, wrapped; when a word is wider than Width, the source rows as they
// came. The rows have no line break.
func FitTable(rows []string, o TableOptions) []string {
	if len(rows) < 2 {
		return rows
	}
	delim := cells(rows[1])
	cols := len(delim)
	aligns := make([]align, cols)
	for c, d := range delim {
		aligns[c] = alignOf(d)
	}
	least := 3 // a delimiter's cell, ":-:" at the least
	if o.Border != nil {
		least = 1
	}
	widest, words := make([]int, cols), make([]int, cols)
	grid := make([][]string, 0, len(rows)-1)
	for i, r := range rows {
		if i == 1 {
			continue
		}
		cs := cells(r)
		cs = append(cs, make([]string, max(0, cols-len(cs)))...)[:cols]
		for c, v := range cs {
			widest[c] = max(widest[c], least, o.Method.StringWidth(v))
			words[c] = max(words[c], least, longestWord(o.Method, v))
		}
		grid = append(grid, cs)
	}
	widths, wrap := widest, false
	if o.Width > 0 && tableWidth(widest) > o.Width {
		if tableWidth(words) > o.Width {
			return records(rows, grid, o)
		}
		widths, wrap = share(words, widest, o.Width-tableWidth(words)), true
	}
	return drawTable(o, grid, widths, aligns, wrap)
}

// tableWidth is the cells a table of columns widths takes: each column and
// the space either side of it, and a rule before each and after the last.
func tableWidth(widths []int) int {
	n := 1
	for _, w := range widths {
		n += w + 3
	}
	return n
}

// longestWord is the widest of v's space-separated words under m.
func longestWord(m ansi.Method, v string) int {
	n := 0
	for w := range strings.FieldsSeq(v) {
		n = max(n, m.StringWidth(w))
	}
	return n
}

// share is the columns from least, given extra cells shared in proportion
// to what each wants to reach most, the cells a share leaves over given one
// a column, left to right, to those still wanting.
func share(least, most []int, extra int) []int {
	out := slices.Clone(least)
	wants := 0
	for c := range out {
		wants += most[c] - least[c]
	}
	if wants == 0 {
		return out
	}
	given := 0
	for c := range out {
		add := extra * (most[c] - least[c]) / wants
		out[c] += add
		given += add
	}
	for c := 0; given < extra && c < len(out); c++ {
		if out[c] < most[c] {
			out[c]++
			given++
		}
	}
	return out
}

// wrapCell is v in lines of at most width cells under m, broken at its
// spaces; no word in v is wider than width.
func wrapCell(m ansi.Method, v string, width int) []string {
	var out []string
	line, n := "", 0
	for w := range strings.FieldsSeq(v) {
		ww := m.StringWidth(w)
		switch {
		case line == "":
			line, n = w, ww
		case n+1+ww <= width:
			line, n = line+" "+w, n+1+ww
		default:
			out = append(out, line)
			line, n = w, ww
		}
	}
	return append(out, line)
}

// drawTable draws grid, its header first, its columns widths wide, in the
// Markdown form or the box o.Border gives; a cell wraps only where wrap,
// since otherwise every cell fits its column.
func drawTable(o TableOptions, grid [][]string, widths []int, aligns []align, wrap bool) []string {
	b := o.Border
	var out []string
	if b != nil {
		out = append(out, rule(b.TopLeft, b.Top, b.MiddleTop, b.TopRight, widths))
	}
	for i, cs := range grid {
		if !wrap {
			out = append(out, tableRow(o.Method, cs, widths, aligns, b))
			out = appendRule(out, i, o, widths, aligns)
			continue
		}
		lines := make([][]string, len(cs))
		height := 1
		for c, v := range cs {
			lines[c] = wrapCell(o.Method, v, widths[c])
			height = max(height, len(lines[c]))
		}
		for k := range height {
			row := make([]string, len(cs))
			for c := range cs {
				if k < len(lines[c]) {
					row[c] = lines[c][k]
				}
			}
			out = append(out, tableRow(o.Method, row, widths, aligns, b))
		}
		out = appendRule(out, i, o, widths, aligns)
	}
	if b != nil {
		out = append(out, rule(b.BottomLeft, b.Bottom, b.MiddleBottom, b.BottomRight, widths))
	}
	return out
}

// appendRule is out with, after the header, row 0, its rule: a box's
// middle line, or the Markdown form's delimiter row.
func appendRule(out []string, i int, o TableOptions, widths []int, aligns []align) []string {
	switch b := o.Border; {
	case i != 0:
		return out
	case b != nil:
		return append(out, rule(b.MiddleLeft, b.Top, b.Middle, b.MiddleRight, widths))
	}
	return append(out, tableDelimiter(widths, aligns))
}

// rule is a box's horizontal line: left, fill across each column and the
// space either side of it, cross between columns, and right.
func rule(left, fill, cross, right string, widths []int) string {
	var s strings.Builder
	s.WriteString(left)
	for c, w := range widths {
		if c > 0 {
			s.WriteString(cross)
		}
		s.WriteString(strings.Repeat(fill, w+2))
	}
	s.WriteString(right)
	return s.String()
}

// records is a table too wide for o.Width as records, a body row each: a
// line "header: value" a cell, or, when one does not fit, the header and
// its value under it indented two cells, wrapped; or, when a word is wider
// than that, rows, its source, as it came.
func records(rows []string, grid [][]string, o TableOptions) []string {
	head, body := grid[0], grid[1:]
	pairs := true
	for _, cs := range body {
		for c, v := range cs {
			if o.Method.StringWidth(pair(head[c], v)) > o.Width {
				pairs = false
			}
		}
	}
	var out []string
	for r, cs := range body {
		if r > 0 {
			out = append(out, "")
		}
		for c, v := range cs {
			if pairs {
				out = append(out, pair(head[c], v))
				continue
			}
			if o.Method.StringWidth(head[c]) > o.Width || longestWord(o.Method, v) > o.Width-2 {
				return rows
			}
			out = append(out, head[c])
			for _, l := range wrapCell(o.Method, v, o.Width-2) {
				if l != "" {
					out = append(out, "  "+l)
				}
			}
		}
	}
	return out
}

// pair is a key/value record's line.
func pair(key, value string) string {
	if value == "" {
		return key + ":"
	}
	return key + ": " + value
}

// tableRow is one line of a row: its cells padded to their columns and
// aligned, between pipes, or between b's sides when b is not nil.
func tableRow(m ansi.Method, cs []string, widths []int, aligns []align, b *glyph.Border) string {
	bar := "|"
	if b != nil {
		bar = b.Left
	}
	var s strings.Builder
	s.WriteString(bar)
	for c, v := range cs {
		pad := max(widths[c]-m.StringWidth(v), 0)
		var l, r int
		switch aligns[c] {
		case alignRight:
			l = pad
		case alignCenter:
			l, r = pad/2, pad-pad/2
		default:
			r = pad
		}
		s.WriteString(" " + strings.Repeat(" ", l) + v + strings.Repeat(" ", r) + " " + bar)
	}
	return s.String()
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
