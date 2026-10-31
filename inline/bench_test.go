package inline_test

import (
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inline"
)

// keyRows is a Composer's three rows, and the same with a key typed at the
// end, which a draw shows in turn: issue #13's drawing alone
// (docs/decisions/0023-MADR-line-mode-components.md A12).
func keyRows() (r *inline.Region, draw func(i int)) {
	r = inline.NewRegion(inline.WithHeight(24))
	r.Resize(80, time.Unix(1, 0))
	a := []string{
		"> words fill the prompt words fill the prompt words fill the prompt words fill",
		"  the prompt words fill the prompt words fill the prompt words fill the prompt",
		"  words fill the prompt words fill the prompt ",
	}
	z := append([]string(nil), a...)
	z[2] += "x"
	return r, func(i int) {
		if i%2 == 0 {
			r.Draw(z, 2, len(z[2]))
		} else {
			r.Draw(a, 2, len(a[2]))
		}
	}
}

// BenchmarkRegionDraw: a region 24 rows tall at 80 columns drawing a
// Composer's three rows, the last of which a key changes and takes back, in
// turn.
func BenchmarkRegionDraw(b *testing.B) {
	_, draw := keyRows()
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		draw(i)
		i++
	}
}

// TestRegionDrawBudget: a drawing that changes one row allocates once, its
// bytes: the rows and the drawn slices are reused, and its builder grows
// once.
func TestRegionDrawBudget(t *testing.T) {
	_, draw := keyRows()
	i := 0
	if n := testing.AllocsPerRun(100, func() { draw(i); i++ }); n > 1 {
		t.Errorf("a drawing: %.1f allocations; want at most 1", n)
	}
}
