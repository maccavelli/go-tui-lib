package stream_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// wideTable is a table whose cells hold a number, a URL and prose.
var wideTable = []string{
	"| id | link | note |",
	"| --: | --- | --- |",
	"| 1234567 | https://example.com/a/long/path | a note with several words in it |",
	"| 8 | https://e.x | short |",
}

// fit is FitTable at width in the Markdown form, measured by wcwidth.
func fit(rows []string, width int) []string {
	return stream.FitTable(rows, stream.TableOptions{Width: width, Method: ansi.WcWidth})
}

// TestFitTableFits: a table that fits is its columns as wide as their
// widest cells, as the Printer has always laid one out; with no width,
// the same.
func TestFitTableFits(t *testing.T) {
	want := []string{
		"| a   |  bb |",
		"| --- | --: |",
		"| ccc |   d |",
	}
	for _, w := range []int{0, 13, 80} {
		if got := fit([]string{"a | bb", "--- | --:", "ccc | d"}, w); strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("at %d: %q; want %q", w, got, want)
		}
	}
}

// TestFitTableShrinks: past the width the columns shrink toward their
// longest word, the prose wraps at its spaces, and the number and the URL
// stay whole; no row is wider than the width.
func TestFitTableShrinks(t *testing.T) {
	got := fit(wideTable, 60)
	for _, r := range got {
		if w := ansi.StringWidth(r); w > 60 {
			t.Errorf("a row of %d cells at 60: %q", w, r)
		}
	}
	all := strings.Join(got, "\n")
	for _, word := range []string{"1234567", "https://example.com/a/long/path"} {
		if !strings.Contains(all, word) {
			t.Errorf("%q broken: %q", word, got)
		}
	}
	if len(got) <= len(wideTable) {
		t.Errorf("%q; want the note wrapped over rows", got)
	}
}

// records is a table of three columns, whose rules take more cells than
// its pairs: 24 cells at the least as a grid, 8 a pair.
var records = []string{"| k | v | w |", "| --- | --- | --- |", "| alpha | beta | gamma |", "| a | b | c |"}

// TestFitTableFallbacks: as the width shrinks, the table becomes key/value
// records, then stacked records, then its source.
func TestFitTableFallbacks(t *testing.T) {
	for _, c := range []struct {
		width int
		want  []string
	}{
		{24, []string{"| k     | v    | w     |", "| ----- | ---- | ----- |", "| alpha | beta | gamma |", "| a     | b    | c     |"}},
		{20, []string{"k: alpha", "v: beta", "w: gamma", "", "k: a", "v: b", "w: c"}},
		{7, []string{"k", "  alpha", "v", "  beta", "w", "  gamma", "", "k", "  a", "v", "  b", "w", "  c"}},
		{6, records}, // the source as it came
	} {
		rows := records
		if got := fit(rows, c.width); strings.Join(got, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("at %d: %q; want %q", c.width, got, c.want)
		}
	}
}

// TestFitTableWide: a row of wide glyphs is padded by cells, so every row
// of the table is the same width.
func TestFitTableWide(t *testing.T) {
	got := fit([]string{"| 名前 | x |", "| --- | --- |", "| 東京都 | y |", "| a | z |"}, 0)
	w := ansi.StringWidth(got[0])
	for _, r := range got {
		if ansi.StringWidth(r) != w {
			t.Errorf("rows of %d and %d cells: %q", w, ansi.StringWidth(r), got)
		}
	}
}

// TestPlainFitsTables: Plain fits a table to its width; a Printer's table
// is never wrapped, as 0023-MADR decided
// (docs/decisions/0009-PLAN-streaming-content-engine.md D20).
func TestPlainFitsTables(t *testing.T) {
	src := strings.Join(records[:3], "\n") + "\n"
	if got, _ := stream.Plain(ansi.WcWidth).Render(src, 20); got != "k: alpha\nv: beta\nw: gamma\n" {
		t.Errorf("Plain at 20: %q; want records", got)
	}
	p := stream.NewPrinter(stream.WithWidth(20))
	if got := p.Write(src) + p.Flush(); got != "| k     | v    | w     |\n| ----- | ---- | ----- |\n| alpha | beta | gamma |\n" {
		t.Errorf("a Printer at 20: %q; want the table unwrapped", got)
	}
}

// TestFitTableGolden: a table in a box of the glyph set's Light border,
// and in the Markdown form, across the matrix.
func TestFitTableGolden(t *testing.T) {
	tuitest.Golden(t, "fittable", tuitest.Matrix{Widths: []int{30, 60}}, func(c tuitest.Case) string {
		b := c.Glyphs().Light
		boxed := stream.FitTable(wideTable, stream.TableOptions{Width: c.Width, Method: ansi.WcWidth, Border: &b})
		md := fit(wideTable, c.Width)
		return strings.Join(boxed, "\n") + "\n\n" + strings.Join(md, "\n") + "\n"
	})
}

var _ = glyph.ASCII // the glyph sets the golden's cases draw with
