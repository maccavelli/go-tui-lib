package layout

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// SizeKind is how a child claims its split's length.
type SizeKind uint8

const (
	// KindFill takes a share, by weight N, of what the others leave.
	KindFill SizeKind = iota
	// KindFixed takes N cells.
	KindFixed
	// KindPercent takes N percent of the split's length.
	KindPercent
	// KindRatio takes N/D of the split's length.
	KindRatio
)

// Size is a child's claim on its split's length, with bounds. The zero Size
// is Fill(1).
type Size struct {
	Kind SizeKind
	// N and D are the claim: cells, a percentage, a ratio N/D, or a fill
	// weight. A fill weight of 0 counts as 1.
	N, D int
	// Min and Max bound the child, in cells. Max 0 means unbounded.
	Min, Max int
	// Shrink orders the children that give up cells when the split is too
	// short: lowest first, and among equals the later child first. A child
	// already at its Min is then hidden in the same order.
	Shrink int
}

// Fixed claims n cells.
func Fixed(n int) Size { return Size{Kind: KindFixed, N: n} }

// Percent claims p percent of the split.
func Percent(p int) Size { return Size{Kind: KindPercent, N: p} }

// Ratio claims n/d of the split.
func Ratio(n, d int) Size { return Size{Kind: KindRatio, N: n, D: d} }

// Fill claims a share of the rest, by weight w.
func Fill(w int) Size { return Size{Kind: KindFill, N: w} }

// AtLeast returns s with a minimum of n cells.
func (s Size) AtLeast(n int) Size { s.Min = n; return s }

// AtMost returns s with a maximum of n cells.
func (s Size) AtMost(n int) Size { s.Max = n; return s }

// ShrinkFirst returns s marked to give up cells before children with the
// default order.
func (s Size) ShrinkFirst() Size { s.Shrink = -1; return s }

// ShrinkOrder returns s with shrink order k.
func (s Size) ShrinkOrder(k int) Size { s.Shrink = k; return s }

func (s Size) valid() error {
	switch {
	case s.N < 0 || s.Min < 0 || s.Max < 0:
		return fmt.Errorf("%w: negative value in %+v", ErrBadSize, s)
	case s.Max > 0 && s.Min > s.Max:
		return fmt.Errorf("%w: Min %d above Max %d", ErrBadSize, s.Min, s.Max)
	case s.Kind == KindPercent && s.N > 100:
		return fmt.Errorf("%w: %d percent", ErrBadSize, s.N)
	case s.Kind == KindRatio && (s.D <= 0 || s.N > s.D):
		return fmt.Errorf("%w: ratio %d/%d", ErrBadSize, s.N, s.D)
	case s.Kind > KindRatio:
		return fmt.Errorf("%w: kind %d", ErrBadSize, s.Kind)
	}
	return nil
}

func (s Size) clamp(n int) int {
	n = max(n, s.Min)
	if s.Max > 0 {
		n = min(n, s.Max)
	}
	return n
}

func (s Size) weight() int {
	if s.N == 0 {
		return 1
	}
	return s.N
}

// Arrange divides area among the children that have a visible pane.
func (s Split) Arrange(area Rect, ctx *Context) error {
	var kids []int
	for i, c := range s.Children {
		if err := c.Size.valid(); err != nil {
			return err
		}
		if s.visible(c.Node, ctx) {
			kids = append(kids, i)
		}
	}
	if s.Gap < 0 {
		return fmt.Errorf("%w: gap %d", ErrBadSize, s.Gap)
	}
	length := area.W
	if s.Axis == Vertical {
		length = area.H
	}
	sizes, kept := allocate(s.Children, kids, length, s.Gap)
	s.resize(ctx, kept, sizes)
	pos := area.X
	if s.Axis == Vertical {
		pos = area.Y
	}
	for k, i := range kept {
		r := area
		if s.Axis == Horizontal {
			r.X, r.W = pos, sizes[k]
		} else {
			r.Y, r.H = pos, sizes[k]
		}
		ctx.path = append(ctx.path, i)
		err := ctx.Arrange(s.Children[i].Node, r)
		ctx.path = ctx.path[:len(ctx.path)-1]
		if err != nil {
			return err
		}
		pos += sizes[k]
		if k < len(kept)-1 {
			sep := area
			if s.Axis == Horizontal {
				sep.X, sep.W = pos, s.Gap
			} else {
				sep.Y, sep.H = pos, s.Gap
			}
			ctx.plan.Separators = append(ctx.plan.Separators, Separator{ID: s.sepID(ctx, i), Axis: s.Axis, Rect: sep, Resizable: s.Name != ""})
			pos += s.Gap
		}
	}
	return nil
}

// visible reports whether n would place any pane: false for a subtree whose
// every known pane is hidden by State.
func (s Split) visible(n Node, ctx *Context) bool {
	ids := leaves(n)
	if len(ids) == 0 {
		return n != nil
	}
	return slices.ContainsFunc(ids, func(id PaneID) bool { return !ctx.state.IsHidden(id) })
}

func (s Split) sepID(ctx *Context, i int) string {
	if s.Name != "" {
		return s.Name + ":" + strconv.Itoa(i)
	}
	parts := make([]string, 0, len(ctx.path)+1)
	for _, p := range ctx.path {
		parts = append(parts, strconv.Itoa(p))
	}
	return "/" + strings.Join(parts, "/") + ":" + strconv.Itoa(i)
}

// resize applies State.Resize to the boundaries between kept children: a
// positive delta moves cells from the later child to the earlier one, as
// far as both children's bounds allow. What it applied goes in Plan.Resize.
func (s Split) resize(ctx *Context, kept []int, sizes []int) {
	for k := 0; k+1 < len(kept); k++ {
		key := s.resizeKey(kept[k])
		d, ok := ctx.state.Resize[key]
		if !ok || d == 0 {
			continue
		}
		a, b := s.Children[kept[k]].Size, s.Children[kept[k+1]].Size
		if d > 0 {
			d = min(d, sizes[k+1]-b.Min)
			if a.Max > 0 {
				d = min(d, a.Max-sizes[k])
			}
		} else {
			d = max(d, a.Min-sizes[k])
			if b.Max > 0 {
				d = max(d, sizes[k+1]-b.Max)
			}
		}
		if d == 0 {
			continue
		}
		sizes[k] += d
		sizes[k+1] -= d
		if ctx.plan.Resize == nil {
			ctx.plan.Resize = map[string]int{}
		}
		ctx.plan.Resize[key] = d
	}
}

// resizeKey is the State.Resize key of the separator after child i. Only a
// named split can be resized: a positional name changes when the tree does.
func (s Split) resizeKey(i int) string {
	if s.Name == "" {
		return ""
	}
	return s.Name + ":" + strconv.Itoa(i)
}

// allocate gives each child in kids a length out of total, less a gap
// between neighbours, and returns the lengths of the children it kept, in
// order. Children it could not fit are dropped, in shrink order.
func allocate(children []Child, kids []int, total, gap int) (sizes []int, kept []int) {
	kept = slices.Clone(kids)
	for {
		sizes = fit(children, kept, total-gap*max(len(kept)-1, 0))
		if sizes != nil || len(kept) <= 1 {
			break
		}
		victim := dropFirst(children, kept)
		kept = slices.DeleteFunc(kept, func(i int) bool { return i == victim })
	}
	if sizes == nil { // one child left, whose Min exceeds the space
		sizes = make([]int, len(kept))
		if len(kept) == 1 {
			sizes[0] = max(total, 0)
		}
	}
	return sizes, kept
}

// dropFirst is the kept child to hide first: the lowest Shrink, and among
// equals the latest.
func dropFirst(children []Child, kept []int) int {
	best := kept[len(kept)-1]
	for _, i := range slices.Backward(kept) {
		if children[i].Size.Shrink < children[best].Size.Shrink {
			best = i
		}
	}
	return best
}

// fit sizes the kept children to exactly avail cells, or returns nil when
// their minimums do not fit.
func fit(children []Child, kept []int, avail int) []int {
	n := len(kept)
	sizes := make([]int, n)
	minSum := 0
	for k, i := range kept {
		minSum += children[i].Size.Min
		sizes[k] = children[i].Size.Min
	}
	if avail < 0 || minSum > avail {
		return nil
	}
	// Claims: fixed cells, then percentages and ratios floored, each
	// clamped. Remainders are kept for the largest-remainder pass.
	rem := make([]int, n) // remainder numerators, scaled to a common base
	for k, i := range kept {
		sz := children[i].Size
		switch sz.Kind {
		case KindFixed:
			sizes[k] = sz.clamp(sz.N)
		case KindPercent:
			sizes[k] = sz.clamp(avail * sz.N / 100)
			rem[k] = (avail * sz.N % 100) * 1000 / 100
		case KindRatio:
			sizes[k] = sz.clamp(avail * sz.N / sz.D)
			rem[k] = (avail * sz.N % sz.D) * 1000 / sz.D
		case KindFill:
			sizes[k] = sz.clamp(0)
		}
	}
	left := avail - sum(sizes)
	if left < 0 {
		return shrink(children, kept, sizes, -left)
	}
	left = share(children, kept, sizes, left, func(s Size) bool { return s.Kind == KindFill })
	left = byRemainder(children, kept, sizes, rem, left)
	// Whatever is still left goes to the last child that can take it, so
	// the split stays tiled; with every child at its Max it stays empty.
	for k := n - 1; k >= 0 && left > 0; k-- {
		sz := children[kept[k]].Size
		room := left
		if sz.Max > 0 {
			room = min(room, sz.Max-sizes[k])
		}
		sizes[k] += room
		left -= room
	}
	return sizes
}

// share gives left cells to the children eligible takes, by weight, with the
// largest remainders first and no child past its Max, and returns what it
// could not place.
func share(children []Child, kept []int, sizes []int, left int, takes func(Size) bool) int {
	for left > 0 {
		var open []int
		weights := 0
		for k, i := range kept {
			sz := children[i].Size
			if takes(sz) && (sz.Max == 0 || sizes[k] < sz.Max) {
				open = append(open, k)
				weights += sz.weight()
			}
		}
		if len(open) == 0 {
			return left
		}
		given := 0
		type part struct{ k, rem int }
		parts := make([]part, 0, len(open))
		for _, k := range open {
			sz := children[kept[k]].Size
			n := left * sz.weight() / weights
			if sz.Max > 0 {
				n = min(n, sz.Max-sizes[k])
			}
			sizes[k] += n
			given += n
			parts = append(parts, part{k, left * sz.weight() % weights})
		}
		rest := left - given
		slices.SortStableFunc(parts, func(a, b part) int { return cmp.Compare(b.rem, a.rem) })
		for _, p := range parts {
			if rest == 0 {
				break
			}
			sz := children[kept[p.k]].Size
			if sz.Max == 0 || sizes[p.k] < sz.Max {
				sizes[p.k]++
				rest--
			}
		}
		if rest == left {
			return left
		}
		left = rest
	}
	return 0
}

// byRemainder gives left cells, one each, to the percentage and ratio
// children with the largest flooring remainders, and returns the rest.
func byRemainder(children []Child, kept []int, sizes []int, rem []int, left int) int {
	order := make([]int, 0, len(kept))
	for k, i := range kept {
		if kind := children[i].Size.Kind; kind == KindPercent || kind == KindRatio {
			order = append(order, k)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(rem[b], rem[a]) })
	for _, k := range order {
		if left == 0 {
			break
		}
		sz := children[kept[k]].Size
		if sz.Max == 0 || sizes[k] < sz.Max {
			sizes[k]++
			left--
		}
	}
	return left
}

// shrink takes over cells from the children, in shrink order, down to their
// Min. fit has checked that the minimums fit, so it always succeeds.
func shrink(children []Child, kept []int, sizes []int, over int) []int {
	order := make([]int, len(kept))
	for k := range order {
		order[k] = k
	}
	slices.SortStableFunc(order, func(a, b int) int {
		if c := cmp.Compare(children[kept[a]].Size.Shrink, children[kept[b]].Size.Shrink); c != 0 {
			return c
		}
		return cmp.Compare(b, a) // later first
	})
	for _, k := range order {
		take := min(over, sizes[k]-children[kept[k]].Size.Min)
		sizes[k] -= take
		over -= take
		if over == 0 {
			break
		}
	}
	return sizes
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}
