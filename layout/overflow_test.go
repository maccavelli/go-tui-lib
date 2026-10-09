package layout

import (
	"maps"
	"math"
	"testing"
)

// TestLayoutOverflow: claims, weights, minimums and gaps whose products or
// sums pass an int are solved exactly, and every pane stays in its area
// (docs/decisions/0014-PLAN-hardening.md Step 7, H10, D6 and D7).
func TestLayoutOverflow(t *testing.T) {
	three := func(gap int, sz Size) Split {
		return Split{Axis: Horizontal, Gap: gap, Children: []Child{
			{Node: Pane{ID: "a"}, Size: sz}, {Node: Pane{ID: "b"}, Size: sz}, {Node: Pane{ID: "c"}, Size: sz},
		}}
	}
	two := func(a, b Size) Split {
		return Split{Axis: Horizontal, Children: []Child{{Node: Pane{ID: "a"}, Size: a}, {Node: Pane{ID: "b"}, Size: b}}}
	}
	for _, tc := range []struct {
		name string
		root Split
		w    int
		want map[PaneID]int // each pane's width; a pane not placed is 0
	}{
		{"Ratio(2^40, 2^41) of 2^24", two(Ratio(1<<40, 1<<41), Fill(1)), 1 << 24, map[PaneID]int{"a": 8388608, "b": 8388608}},
		// 2^24*(2^40+1)/(2^41+3) is 8,388,607.99…, floored.
		{"Ratio(2^40+1, 2^41+3) of 2^24", two(Ratio(1<<40+1, 1<<41+3), Fill(1)), 1 << 24, map[PaneID]int{"a": 8388607, "b": 8388609}},
		{"two Fill(2^40) of 2^24", two(Fill(1<<40), Fill(1<<40)), 1 << 24, map[PaneID]int{"a": 8388608, "b": 8388608}},
		{"Percent(50) of 2^62", two(Percent(50), Fill(1)), 1 << 62, map[PaneID]int{"a": 1 << 61, "b": 1 << 61}},
		// The weights are shifted until they sum within an int; the one
		// cell left over goes to the first, as for any equal weights.
		{"three Fill(MaxInt) of 100", three(0, Fill(math.MaxInt)), 100, map[PaneID]int{"a": 34, "b": 33, "c": 33}},
		// The one cell left goes to the larger remainder: b's is 0.5 of
		// 2^61+1, scaled to 500, which r*1000 would overflow; a's is 333.
		{"remainders of a 2^61 ratio", Split{Axis: Horizontal, Children: []Child{
			{Node: Pane{ID: "a"}, Size: Ratio(1, 3)},
			{Node: Pane{ID: "b"}, Size: Ratio(1037629354146162304, 1<<61+1)},
			{Node: Pane{ID: "c"}, Size: Fixed(2)},
		}}, 10, map[PaneID]int{"a": 3, "b": 5, "c": 2}},
		// D6: the claims' sum passes an int; the later children shrink first.
		{"three Fixed(2^62) of 100", three(0, Fixed(1<<62)), 100, map[PaneID]int{"a": 100, "b": 0, "c": 0}},
		// D6: the minimums' sum passes an int; the later two are hidden.
		{"three Min 2^62 of 2^62+5", three(0, Fixed(1<<62).AtLeast(1<<62)), 1<<62 + 5, map[PaneID]int{"a": 1<<62 + 5}},
		// D7: the gaps' product passes an int; the gaps do not fit.
		{"gap 2^62+1 of 1", three(1<<62+1, Fill(1)), 1, map[PaneID]int{"a": 1}},
	} {
		area := Rect{W: tc.w, H: 1}
		p, err := Solve(tc.root, area, State{})
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		got := map[PaneID]int{}
		for id, r := range p.Panes {
			if r.W != 0 || tc.want[id] != 0 {
				got[id] = r.W
			}
		}
		want := maps.Clone(tc.want)
		maps.DeleteFunc(want, func(_ PaneID, w int) bool { return w == 0 })
		if !maps.Equal(got, want) {
			t.Errorf("%s: widths %v, want %v", tc.name, got, want)
			continue
		}
		check(t, tc.name, p, area, true)
	}
}
