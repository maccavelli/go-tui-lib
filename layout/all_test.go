package layout

import (
	"slices"
	"testing"
)

func TestPlanAllYieldsInOrderAndStops(t *testing.T) {
	p, err := Solve(SidebarRightBottom("main", "side", "logs", Footer("footer", 1)), Rect{W: 120, H: 40}, State{})
	if err != nil {
		t.Fatal(err)
	}
	var ids []PaneID
	for id, r := range p.All() {
		if r != p.Panes[id] {
			t.Fatalf("%s: rectangle %v, want %v", id, r, p.Panes[id])
		}
		ids = append(ids, id)
	}
	if !slices.Equal(ids, p.Order) {
		t.Fatalf("All yielded %v, want Order %v", ids, p.Order)
	}
	// A loop that breaks stops the iterator.
	n := 0
	for range p.All() {
		n++
		if n == 2 {
			break
		}
	}
	if n != 2 {
		t.Fatalf("a loop broken after 2 panes ran %d times", n)
	}
}
