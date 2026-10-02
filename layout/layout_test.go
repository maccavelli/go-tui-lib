package layout

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/tuitest"
)

func mustSolve(t *testing.T, root Node, w, h int, st State) Plan {
	t.Helper()
	p, err := Solve(root, Rect{W: w, H: h}, st)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func widths(p Plan, ids ...PaneID) []int {
	out := make([]int, len(ids))
	for i, id := range ids {
		out[i] = p.Panes[id].W
	}
	return out
}

func row(gap int, kids ...Child) Split {
	return Split{Name: "row", Axis: Horizontal, Gap: gap, Children: kids}
}
func leaf(id PaneID, s Size) Child { return Child{Node: Pane{ID: id}, Size: s} }

func TestClaims(t *testing.T) {
	for _, c := range []struct {
		name  string
		split Split
		w     int
		want  []int
	}{
		{"fixed and fill", row(0, leaf("a", Fixed(10)), leaf("b", Fill(1))), 50, []int{10, 40}},
		{"percent and fill", row(0, leaf("a", Percent(30)), leaf("b", Fill(1))), 50, []int{15, 35}},
		{"ratio", row(0, leaf("a", Ratio(1, 3)), leaf("b", Fill(1))), 30, []int{10, 20}},
		{"weights", row(0, leaf("a", Fill(1)), leaf("b", Fill(3))), 40, []int{10, 30}},
		{"remainders largest first", row(0, leaf("a", Fill(1)), leaf("b", Fill(1)), leaf("c", Fill(1))), 10, []int{4, 3, 3}},
		{"percent remainders", row(0, leaf("a", Percent(50)), leaf("b", Percent(50))), 11, []int{6, 5}},
		{"gap", row(2, leaf("a", Fill(1)), leaf("b", Fill(1))), 12, []int{5, 5}},
		{"max caps a fill", row(0, leaf("a", Fill(1).AtMost(5)), leaf("b", Fill(1))), 30, []int{5, 25}},
		{"min lifts a percent", row(0, leaf("a", Percent(10).AtLeast(8)), leaf("b", Fill(1))), 40, []int{8, 32}},
		{"leftover goes to the last child", row(0, leaf("a", Fixed(5)), leaf("b", Fixed(5))), 20, []int{5, 15}},
	} {
		ids := []PaneID{"a", "b", "c"}[:len(c.want)]
		if got := widths(mustSolve(t, c.split, c.w, 1, State{}), ids...); !slices.Equal(got, c.want) {
			t.Errorf("%s: widths %v, want %v", c.name, got, c.want)
		}
	}
}

func TestShrinkOrder(t *testing.T) {
	// 60 asked for, 40 available: b shrinks first (ShrinkFirst), then c,
	// then a, each down to its Min.
	s := row(0, leaf("a", Fixed(20).AtLeast(10)), leaf("b", Fixed(20).AtLeast(15).ShrinkFirst()), leaf("c", Fixed(20).AtLeast(10)))
	if got := widths(mustSolve(t, s, 40, 1, State{}), "a", "b", "c"); !slices.Equal(got, []int{15, 15, 10}) {
		t.Fatalf("widths %v, want [15 15 10]", got)
	}
}

func TestHideWhenMinimumsDoNotFit(t *testing.T) {
	s := row(1, leaf("main", Fill(1).AtLeast(30).ShrinkOrder(1)), leaf("side", Fixed(20).AtLeast(20)))
	p := mustSolve(t, s, 40, 5, State{})
	if _, ok := p.Panes["side"]; ok || p.Panes["main"].W != 40 || !slices.Equal(p.Hidden, []PaneID{"side"}) {
		t.Fatalf("plan %+v, want side hidden and main the whole width", p)
	}
	if len(p.Separators) != 0 {
		t.Fatalf("a hidden neighbour left a separator: %+v", p.Separators)
	}
}

func TestResponsiveChoosesTheFirstRule(t *testing.T) {
	r := Responsive{
		Rules: []Rule{{When: MinWidth(100), Use: row(0, leaf("a", Fill(1)), leaf("b", Fill(1)))}},
		Else:  Pane{ID: "a"},
	}
	if p := mustSolve(t, r, 120, 10, State{}); len(p.Panes) != 2 {
		t.Fatalf("at 120: %+v", p.Panes)
	}
	p := mustSolve(t, r, 80, 10, State{})
	if len(p.Panes) != 1 || p.Panes["a"].W != 80 || !slices.Equal(p.Hidden, []PaneID{"b"}) {
		t.Fatalf("at 80: %+v", p)
	}
	if !Not(MinWidth(5))(Rect{W: 4}) || !Or(MinWidth(9), MinHeight(1))(Rect{W: 1, H: 1}) {
		t.Fatal("Not / Or")
	}
}

func TestResizeIsClampedAtEverySize(t *testing.T) {
	s := row(1, leaf("main", Fill(1).AtLeast(30)), leaf("side", Percent(30).AtLeast(20).AtMost(50)))
	base := mustSolve(t, s, 200, 10, State{})
	wide := mustSolve(t, s, 200, 10, State{}.WithResize("row:0", -40))
	if got := wide.Panes["side"].W; got != 50 {
		t.Fatalf("side after a -40 drag from %d = %d, want its Max 50", base.Panes["side"].W, got)
	}
	// The same state at 60 columns: main keeps its Min.
	small := mustSolve(t, s, 60, 10, State{}.WithResize("row:0", -40))
	if small.Panes["main"].W < 30 || small.Panes["main"].W+small.Panes["side"].W+1 != 60 {
		t.Fatalf("at 60: %v", widths(small, "main", "side"))
	}
	// A positional (unnamed) split ignores resizes.
	anon := s
	anon.Name = ""
	if p := mustSolve(t, anon, 200, 10, State{}.WithResize("row:0", -40)); p.Panes["side"] != base.Panes["side"] {
		t.Fatal("an unnamed split was resized")
	}
}

func TestPlanReportsAppliedResize(t *testing.T) {
	s := row(1, leaf("main", Fill(1).AtLeast(30)), leaf("side", Percent(30).AtLeast(20).AtMost(50)))
	base := mustSolve(t, s, 200, 10, State{})
	if base.Resize != nil {
		t.Fatalf("no resize asked for, Plan.Resize = %v", base.Resize)
	}
	// +1000 asks to move the separator far past side's Min of 20.
	far := mustSolve(t, s, 200, 10, State{}.WithResize("row:0", 1000))
	want := base.Panes["side"].W - 20
	if got, ok := far.Resize["row:0"]; !ok || got != want {
		t.Fatalf("Plan.Resize[row:0] = %d, %v; want the applied %d, not the asked 1000", got, ok, want)
	}
	if far.Panes["side"].W != 20 {
		t.Fatalf("side = %d, want its Min 20", far.Panes["side"].W)
	}
	// Storing the applied delta, then moving back by 5, moves the separator
	// 5 cells: there is no dead zone of 1000 - want cells.
	back := mustSolve(t, s, 200, 10, State{}.WithResize("row:0", far.Resize["row:0"]-5))
	if back.Panes["side"].W != 25 {
		t.Fatalf("after -5 from the applied delta, side = %d, want 25", back.Panes["side"].W)
	}
	// side starts at its Max, so growing it is clamped to nothing, and a
	// delta clamped to 0 is absent.
	if base.Panes["side"].W != 50 {
		t.Fatalf("side = %d, want it at its Max 50 for the next case", base.Panes["side"].W)
	}
	none := mustSolve(t, s, 200, 10, State{}.WithResize("row:0", -7))
	if _, ok := none.Resize["row:0"]; ok || none.Panes["side"].W != 50 {
		t.Fatalf("a delta clamped to 0: Plan.Resize = %v, side = %d", none.Resize, none.Panes["side"].W)
	}
}

// TestAppliedResizeReproducesThePlan checks the property a host relies on
// when it stores Plan.Resize: solving again with the applied deltas gives
// the same plan, and any deltas leave the split tiled.
func TestAppliedResizeReproducesThePlan(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for i := range 2000 {
		next := 0
		// random names splits after the pane count, so two splits can share
		// a name, and with it their separators' keys. A program's names are
		// unique, so the tree is renamed before the property is checked.
		splits := 0
		root := uniqueNames(random(r, 4, &next, 8, false), &splits)
		area := Rect{W: 1 + r.IntN(300), H: 1 + r.IntN(100)}
		label := fmt.Sprintf("tree %d at %dx%d", i, area.W, area.H)
		plain, err := Solve(root, area, State{})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		st := State{}
		for _, s := range plain.Separators {
			if s.Resizable {
				st = st.WithResize(s.ID, r.IntN(401)-200)
			}
		}
		asked, err := Solve(root, area, st)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		check(t, label+" resized", asked, area, true)
		applied, err := Solve(root, area, State{Resize: asked.Resize})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if !reflect.DeepEqual(applied.Panes, asked.Panes) || !reflect.DeepEqual(applied.Resize, asked.Resize) {
			t.Fatalf("%s: solving with the applied deltas %v gives a different plan than asking for %v", label, asked.Resize, st.Resize)
		}
	}
}

func TestSeparatorResizable(t *testing.T) {
	named := row(1, leaf("a", Fill(1)), leaf("b", Fill(1)), leaf("c", Fill(1)))
	anon := named
	anon.Name = ""
	for _, c := range []struct {
		name string
		root Split
		want bool
	}{{"named", named, true}, {"unnamed", anon, false}} {
		p := mustSolve(t, c.root, 60, 5, State{})
		if len(p.Separators) != 2 {
			t.Fatalf("%s: %d separators, want 2", c.name, len(p.Separators))
		}
		for _, s := range p.Separators {
			if s.Resizable != c.want {
				t.Errorf("%s split: separator %s Resizable = %v, want %v", c.name, s.ID, s.Resizable, c.want)
			}
		}
	}
}

func TestStateJSON(t *testing.T) {
	st := State{}.WithHidden("logs", true).WithHidden("a", true).WithResize("sidebar:0", 7).WithZoom("main")
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"version":1,"resize":{"sidebar:0":7},"hidden":["a","logs"],"zoom":"main"}` {
		t.Fatalf("JSON = %s", b)
	}
	var back State
	if err := json.Unmarshal(b, &back); err != nil || !reflect.DeepEqual(back, st) {
		t.Fatalf("round trip = %+v, %v", back, err)
	}
	if err := json.Unmarshal([]byte(`{"version":2}`), &back); !errors.Is(err, ErrStateVersion) {
		t.Fatalf("version 2: %v", err)
	}
	if _, err := Solve(Pane{ID: "a"}, Rect{W: 1, H: 1}, State{Version: 9}); !errors.Is(err, ErrStateVersion) {
		t.Fatalf("Solve with version 9: %v", err)
	}
	if st.WithHidden("logs", false).IsHidden("logs") || st.WithResize("sidebar:0", -7).Resize != nil {
		t.Fatal("WithHidden false / WithResize back to 0")
	}
}

func TestHiddenAndZoom(t *testing.T) {
	s := row(1, leaf("a", Fill(1)), leaf("b", Fill(1)), leaf("c", Fill(1)))
	p := mustSolve(t, s, 31, 5, State{}.WithHidden("b", true))
	if _, ok := p.Panes["b"]; ok || p.Panes["a"].W+p.Panes["c"].W+1 != 31 || !slices.Equal(p.Order, []PaneID{"a", "c"}) {
		t.Fatalf("hidden b: %+v", p)
	}
	z := mustSolve(t, s, 31, 5, State{}.WithZoom("c"))
	if len(z.Panes) != 1 || z.Panes["c"] != (Rect{W: 31, H: 5}) || !slices.Equal(z.Hidden, []PaneID{"a", "b"}) {
		t.Fatalf("zoom c: %+v", z)
	}
	if u := mustSolve(t, s, 31, 5, State{}.WithZoom("nope")); len(u.Panes) != 3 {
		t.Fatal("zooming an unknown pane changed the plan")
	}
}

func TestErrors(t *testing.T) {
	dup := row(0, leaf("a", Fill(1)), leaf("a", Fill(1)))
	if _, err := Solve(dup, Rect{W: 10, H: 1}, State{}); !errors.Is(err, ErrDuplicatePane) {
		t.Errorf("duplicate: %v", err)
	}
	for _, sz := range []Size{Fixed(-1), Percent(101), Ratio(2, 1), Ratio(1, 0), Fill(1).AtLeast(5).AtMost(4)} {
		if _, err := Solve(row(0, leaf("a", sz)), Rect{W: 10, H: 1}, State{}); !errors.Is(err, ErrBadSize) {
			t.Errorf("size %+v: %v", sz, err)
		}
	}
	if _, err := Solve(Pane{ID: "a"}, Rect{W: -1}, State{}); !errors.Is(err, ErrBadArea) {
		t.Errorf("negative area: %v", err)
	}
}

// grid is a custom Node: its panes in equal columns of one row.
type grid struct{ ids []PaneID }

func (g grid) Leaves() []PaneID { return g.ids }
func (g grid) Arrange(area Rect, ctx *Context) error {
	kids := make([]Child, len(g.ids))
	for i, id := range g.ids {
		kids[i] = leaf(id, Fill(1))
	}
	return ctx.Arrange(Split{Axis: Horizontal, Children: kids}, area)
}

func TestCustomNode(t *testing.T) {
	root := Split{Axis: Vertical, Children: []Child{
		{Node: grid{ids: []PaneID{"x", "y", "z"}}, Size: Fill(1)},
		leaf("log", Fixed(3)),
	}}
	p := mustSolve(t, root, 30, 10, State{})
	if len(p.Panes) != 4 || p.Panes["y"] != (Rect{X: 10, W: 10, H: 7}) || p.Panes["log"] != (Rect{Y: 7, W: 30, H: 3}) {
		t.Fatalf("custom node plan: %+v", p.Panes)
	}
}

// diagram draws a plan: each pane filled with the first letter of its ID,
// separators as '|' or '-', unplaced cells as '.'.
func diagram(p Plan, w, h int) string {
	cells := make([][]rune, h)
	for y := range cells {
		cells[y] = []rune(strings.Repeat(".", w))
	}
	for _, id := range p.Order {
		r, ch := p.Panes[id], []rune(string(id))[0]
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				cells[y][x] = ch
			}
		}
	}
	for _, s := range p.Separators {
		ch := '|'
		if s.Axis == Vertical {
			ch = '-'
		}
		for y := s.Rect.Y; y < s.Rect.Y+s.Rect.H; y++ {
			for x := s.Rect.X; x < s.Rect.X+s.Rect.W; x++ {
				if cells[y][x] == '|' || cells[y][x] == '-' {
					cells[y][x] = '+'
				} else {
					cells[y][x] = ch
				}
			}
		}
	}
	var b strings.Builder
	for _, row := range cells {
		b.WriteString(string(row) + "\n")
	}
	return b.String()
}

func presets() map[string]Node {
	return map[string]Node{
		"sidebar-right":             SidebarRight("main", "side"),
		"sidebar-left":              SidebarLeft("main", "side"),
		"sidebar-right-bottom":      SidebarRightBottom("main", "side", "logs"),
		"sidebar-left-bottom":       SidebarLeftBottom("main", "side", "logs"),
		"sidebar-right-bottom-main": SidebarRightBottom("main", "side", "logs", BottomSpan(UnderMain)),
		"sidebar-left-bottom-main":  SidebarLeftBottom("main", "side", "logs", BottomSpan(UnderMain)),
		"sidebar-right-footer":      SidebarRightBottom("main", "side", "logs", Footer("footer", 1)),
	}
}

// presetNames lists presets() in sorted order, so tests that walk it do so
// the same way every run.
func presetNames() []string { return slices.Sorted(maps.Keys(presets())) }

func TestPresetDiagrams(t *testing.T) {
	all := presets()
	for _, name := range presetNames() {
		root := all[name]
		var b strings.Builder
		for _, h := range []int{20, 40} {
			for _, w := range []int{60, 80, 120, 200} {
				p := mustSolve(t, root, w, h, State{})
				fmt.Fprintf(&b, "== %dx%d  hidden %v\n", w, h, p.Hidden)
				b.WriteString(diagram(p, w, h))
			}
		}
		tuitest.Text(t, "preset-"+name, b.String())
	}
}

func TestPresetsKeepTheMainPane(t *testing.T) {
	all := presets()
	for _, name := range presetNames() {
		root := all[name]
		for _, w := range []int{20, 60, 99, 100, 200} {
			for _, h := range []int{3, 15, 16, 60} {
				if p := mustSolve(t, root, w, h, State{}); p.Panes["main"].Empty() {
					t.Errorf("%s at %dx%d: no main pane: %+v", name, w, h, p.Panes)
				}
			}
		}
	}
}

// random builds a tree of depth at most d with at most *budget panes.
func random(r *rand.Rand, d int, next *int, budget int, withMax bool) Node {
	if d == 0 || *next >= budget || r.IntN(3) == 0 {
		*next++
		return Pane{ID: PaneID(fmt.Sprintf("p%d", *next))}
	}
	s := Split{Name: fmt.Sprintf("s%d", *next), Axis: Axis(r.IntN(2)), Gap: r.IntN(3)}
	for range 2 + r.IntN(3) {
		if *next >= budget {
			break
		}
		var sz Size
		switch r.IntN(4) {
		case 0:
			sz = Fixed(r.IntN(30))
		case 1:
			sz = Percent(r.IntN(101))
		case 2:
			sz = Ratio(1+r.IntN(3), 4)
		default:
			sz = Fill(1 + r.IntN(3))
		}
		if r.IntN(3) == 0 {
			sz = sz.AtLeast(r.IntN(10))
		}
		if withMax && r.IntN(3) == 0 {
			sz = sz.AtMost(sz.Min + 1 + r.IntN(40))
		}
		sz = sz.ShrinkOrder(r.IntN(3) - 1)
		s.Children = append(s.Children, Child{Node: random(r, d-1, next, budget, withMax), Size: sz})
	}
	if len(s.Children) == 0 {
		*next++
		return Pane{ID: PaneID(fmt.Sprintf("p%d", *next))}
	}
	return s
}

// uniqueNames returns n with every split renamed "u<k>", in tree order.
func uniqueNames(n Node, k *int) Node {
	s, ok := n.(Split)
	if !ok {
		return n
	}
	out := s
	out.Name = fmt.Sprintf("u%d", *k)
	*k++
	out.Children = make([]Child, len(s.Children))
	for i, c := range s.Children {
		out.Children[i] = Child{Node: uniqueNames(c.Node, k), Size: c.Size}
	}
	return out
}

// check verifies the invariants every plan must hold.
func check(t *testing.T, label string, p Plan, area Rect, tiled bool) {
	t.Helper()
	var rects []Rect
	for _, id := range p.Order {
		rects = append(rects, p.Panes[id])
	}
	for _, s := range p.Separators {
		rects = append(rects, s.Rect)
	}
	cells := 0
	for i, r := range rects {
		if !r.Empty() && !r.Within(area) {
			t.Fatalf("%s: %+v escapes %+v", label, r, area)
		}
		for _, o := range rects[i+1:] {
			if r.Overlaps(o) {
				t.Fatalf("%s: %+v overlaps %+v", label, r, o)
			}
		}
		if !r.Empty() {
			cells += r.W * r.H
		}
	}
	if tiled && len(p.Order) > 0 && cells != area.W*area.H {
		t.Fatalf("%s: %d of %d cells covered", label, cells, area.W*area.H)
	}
}

func TestSolverProperties(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for i := range 3000 {
		withMax := i%2 == 0
		next := 0
		root := random(r, 4, &next, 8, withMax)
		area := Rect{W: 1 + r.IntN(300), H: 1 + r.IntN(100)}
		label := fmt.Sprintf("tree %d at %dx%d", i, area.W, area.H)
		p, err := Solve(root, area, State{})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		// Only a tree with no Max must tile: with every child of a split at
		// its Max, the rest of the split stays empty by design.
		check(t, label, p, area, !withMax)
		again, _ := Solve(root, area, State{})
		if !reflect.DeepEqual(p, again) {
			t.Fatalf("%s: two solves differ", label)
		}
		if len(p.Order)+len(p.Hidden) != next {
			t.Fatalf("%s: %d placed + %d hidden of %d panes", label, len(p.Order), len(p.Hidden), next)
		}
	}
}

// decode builds a small tree from fuzz bytes, deterministically.
func decode(data []byte) Node {
	i, n := 0, 0
	nextByte := func() int {
		if i >= len(data) {
			return 0
		}
		i++
		return int(data[i-1])
	}
	var build func(depth int) Node
	build = func(depth int) Node {
		b := nextByte()
		if depth == 0 || b%3 == 0 || n >= 12 {
			n++
			return Pane{ID: PaneID(fmt.Sprintf("p%d", n))}
		}
		s := Split{Name: fmt.Sprintf("s%d", n), Axis: Axis(b % 2), Gap: nextByte() % 4}
		for range 1 + nextByte()%4 {
			sz := Size{Kind: SizeKind(nextByte() % 4), N: nextByte(), D: 1 + nextByte(), Min: nextByte() % 40, Shrink: nextByte()%3 - 1}
			if sz.Kind == KindPercent {
				sz.N %= 101
			}
			if sz.Kind == KindRatio {
				sz.N %= sz.D + 1
			}
			if m := nextByte(); m%2 == 0 {
				sz.Max = sz.Min + m%60
			}
			s.Children = append(s.Children, Child{Node: build(depth - 1), Size: sz})
		}
		return s
	}
	return build(4)
}

func FuzzSolve(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}, 80, 24)
	f.Add([]byte{4, 1, 3, 1, 50, 2, 10, 1, 7, 0, 9}, 200, 60)
	f.Fuzz(func(t *testing.T, data []byte, w, h int) {
		w, h = abs(w)%400, abs(h)%200
		area := Rect{W: w, H: h}
		p, err := Solve(decode(data), area, State{})
		if err != nil {
			if errors.Is(err, ErrBadSize) || errors.Is(err, ErrDuplicatePane) {
				return
			}
			t.Fatal(err)
		}
		check(t, "fuzz", p, area, false)
	})
}

func abs(x int) int {
	if x < 0 {
		return -(x + 1) // avoids overflow on the minimum int
	}
	return x
}

func BenchmarkSolvePresets(b *testing.B) {
	all := presets()
	for _, name := range presetNames() {
		root := all[name]
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				if _, err := Solve(root, Rect{W: 200, H: 60}, State{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
