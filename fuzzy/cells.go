package fuzzy

import "github.com/charmbracelet/x/ansi"

// CellRange is the cells [Start, End) of a line, as lipgloss.StyleRanges
// takes them.
type CellRange struct{ Start, End int }

// Cells turns byte ranges of s, such as a Match's, into the cells they
// occupy when s is drawn with method: the workspace's width method, so a
// highlight lands where the text does. s is plain text, with no escape
// sequence or control character, as internal/sanitize's Line leaves a
// row. Widths are method.StringWidth's, the measure the workspace's cutter
// uses. A range out of order or outside s is skipped; one with no width
// gives no cell range.
func Cells(s string, ranges []Range, method ansi.Method) []CellRange {
	out := make([]CellRange, 0, len(ranges))
	at, cell := 0, 0
	for _, r := range ranges {
		if r.Start < at || r.End < r.Start || r.End > len(s) {
			continue
		}
		cell += method.StringWidth(s[at:r.Start])
		w := method.StringWidth(s[r.Start:r.End])
		if w > 0 {
			out = append(out, CellRange{Start: cell, End: cell + w})
		}
		cell += w
		at = r.End
	}
	return out
}
