// Package layout turns a tree of panes and a terminal size into rectangles.
//
// It is pure geometry, with no terminal and no Charm import
// (docs/decisions/0002-MADR-multi-pane-workspace-layouts.md §2), so a layout
// can be tested, persisted and reused by any front end:
//
//   - a Node tree describes the arrangement: Pane leaves, Split nodes with a
//     Size per child, Responsive nodes that choose a tree by size, or a
//     program's own Node;
//   - Solve allocates cells deterministically, in integers, so the children
//     of a split always add up to it, and no rectangle leaves the area;
//   - State holds what a user changed (resized separators, hidden panes, a
//     zoomed pane) as JSON-ready data that Solve re-applies, and re-clamps,
//     at every size;
//   - the presets build the common arrangements, a main pane with a sidebar
//     on either side, each with or without a bottom pane and a footer, as
//     ordinary trees a program can change.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package layout

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/maccavelli/go-tui-lib/internal/enum"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "layout"

// Rect is a rectangle of terminal cells. X and Y are its top-left cell.
type Rect struct{ X, Y, W, H int }

// Empty reports whether r has no cells.
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// Contains reports whether cell (x, y) is in r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Within reports whether r lies entirely inside outer.
func (r Rect) Within(outer Rect) bool {
	return r.X >= outer.X && r.Y >= outer.Y && r.X+r.W <= outer.X+outer.W && r.Y+r.H <= outer.Y+outer.H
}

// Overlaps reports whether r and o share a cell.
func (r Rect) Overlaps(o Rect) bool {
	if r.Empty() || o.Empty() {
		return false
	}
	return r.X < o.X+o.W && o.X < r.X+r.W && r.Y < o.Y+o.H && o.Y < r.Y+r.H
}

// Axis is the direction a split lays its children out in.
type Axis uint8

const (
	// Horizontal lays children side by side, dividing the width.
	Horizontal Axis = iota
	// Vertical stacks children, dividing the height.
	Vertical
)

var axisNames = enum.Names[Axis]{
	Pkg: pkgName, Type: "Axis",
	Tokens: []string{"horizontal", "vertical"},
}

// String is the axis's token.
func (a Axis) String() string { return axisNames.String(a) }

// MarshalText is the axis's token. An axis with no token is an error.
func (a Axis) MarshalText() ([]byte, error) { return axisNames.Marshal(a) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (a *Axis) UnmarshalText(b []byte) error { return axisNames.Unmarshal(b, a) }

// PaneID names a pane. IDs are unique within a tree.
type PaneID string

// Node is a node of a layout tree. Pane, Split and Responsive are the
// built-in kinds. A program's own kind, such as a grid or a tab stack,
// implements Node by placing its leaves through ctx, and Leaves if it wants
// its panes reported as hidden when they are not placed.
type Node interface {
	Arrange(area Rect, ctx *Context) error
}

// Leaver is implemented by a Node that can list every pane it may place,
// in order, whether or not it places them at a given size.
type Leaver interface {
	Leaves() []PaneID
}

// Pane is a leaf: one pane, placed in the whole area it is given.
type Pane struct{ ID PaneID }

// Arrange places the pane.
func (p Pane) Arrange(area Rect, ctx *Context) error { return ctx.Place(p.ID, area) }

// Leaves returns the pane's ID.
func (p Pane) Leaves() []PaneID { return []PaneID{p.ID} }

// Child is a node of a split, with its claim on the split's length.
type Child struct {
	Node Node
	Size Size
}

// Split divides its area among its children along Axis, with Gap cells
// between neighbours for a separator or a border. Name, when set, names
// the split's separators "<name>:<i>" (between child i and i+1), so a
// resize in State survives a change of tree; otherwise they are named by
// position, as "/<path>:<i>".
//
// A name is used by at most one split in each Solve, or Solve fails with
// ErrBadSplitName. Splits in different Responsive rules, or in trees a
// program swaps, may share a name, and then share the resize: the presets
// do this so a resize survives a change of breakpoint. A name must not
// start with "/", which positional IDs use.
type Split struct {
	Name     string
	Axis     Axis
	Children []Child
	Gap      int
}

// Leaves returns every pane of the split's children, in order.
func (s Split) Leaves() []PaneID {
	var out []PaneID
	for _, c := range s.Children {
		out = append(out, leaves(c.Node)...)
	}
	return out
}

// Condition decides whether a responsive rule applies to an area.
type Condition func(area Rect) bool

// MinWidth holds when the area is at least n cells wide.
func MinWidth(n int) Condition { return func(a Rect) bool { return a.W >= n } }

// MinHeight holds when the area is at least n cells tall.
func MinHeight(n int) Condition { return func(a Rect) bool { return a.H >= n } }

// And holds when every condition holds.
func And(cs ...Condition) Condition {
	return func(a Rect) bool {
		for _, c := range cs {
			if !c(a) {
				return false
			}
		}
		return true
	}
}

// Or holds when any condition holds.
func Or(cs ...Condition) Condition {
	return func(a Rect) bool {
		return slices.ContainsFunc(cs, func(c Condition) bool { return c(a) })
	}
}

// Not holds when c does not.
func Not(c Condition) Condition { return func(a Rect) bool { return !c(a) } }

// Rule is one choice of a Responsive node: Use, when When holds.
type Rule struct {
	When Condition
	Use  Node
}

// Responsive arranges the first rule whose condition holds for the area,
// or Else when none does. A nil Else places nothing.
type Responsive struct {
	Rules []Rule
	Else  Node
}

// Arrange arranges the chosen node.
func (r Responsive) Arrange(area Rect, ctx *Context) error {
	if n := r.choose(area); n != nil {
		return ctx.Arrange(n, area)
	}
	return nil
}

func (r Responsive) choose(area Rect) Node {
	for _, rule := range r.Rules {
		if rule.When == nil || rule.When(area) {
			return rule.Use
		}
	}
	return r.Else
}

// Leaves returns every pane any rule, or Else, may place, each once, in the
// order first seen.
func (r Responsive) Leaves() []PaneID {
	var out []PaneID
	for _, rule := range r.Rules {
		out = append(out, leaves(rule.Use)...)
	}
	out = append(out, leaves(r.Else)...)
	return dedupe(out)
}

func leaves(n Node) []PaneID {
	if l, ok := n.(Leaver); ok {
		return l.Leaves()
	}
	return nil
}

func dedupe(ids []PaneID) []PaneID {
	seen := make(map[PaneID]bool, len(ids))
	out := ids[:0:0]
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// Separator is the boundary between two neighbouring children of a split:
// the Gap cells between them, which a host draws and lets the user drag.
type Separator struct {
	// ID names the separator for State.Resize.
	ID string
	// Axis is the axis of the split it divides. A Horizontal split's
	// separators are vertical lines.
	Axis Axis
	// Rect is the gap's cells. With a Gap of 0 it is empty, at the boundary.
	Rect Rect
	// Resizable is true for a named split's separators, the only ones
	// State.Resize moves. An unnamed split's separator has a positional ID,
	// which changes when the tree does, so a host must not resize it.
	Resizable bool
}

// Plan is the result of Solve.
type Plan struct {
	// Panes holds the rectangle of every placed pane.
	Panes map[PaneID]Rect
	// Order lists the placed panes in tree order: the default focus order.
	Order []PaneID
	// Separators lists every separator between placed children.
	Separators []Separator
	// Hidden lists, in tree order, the panes the tree knows of but did not
	// place: hidden by State, dropped by a responsive rule, or squeezed out.
	Hidden []PaneID
	// Resize holds, for each separator that State.Resize moved, the delta
	// Solve applied after clamping it to the children's bounds. A host
	// stores this, not the delta it asked for, so a drag or a held key past
	// a limit leaves no dead zone to work back through. A separator with
	// nothing applied is absent, and Resize is nil when none is present.
	Resize map[string]int
}

// All yields each placed pane and its rectangle, in Order, the tree's
// order (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §6).
func (p Plan) All() iter.Seq2[PaneID, Rect] {
	return func(yield func(PaneID, Rect) bool) {
		for _, id := range p.Order {
			if !yield(id, p.Panes[id]) {
				return
			}
		}
	}
}

// Context carries the state and collects the plan while a tree is
// arranged. A custom Node uses Place for its leaves and Arrange for its
// children.
type Context struct {
	state  State
	plan   *Plan
	path   []int
	splits map[string]bool // split names claimed in this solve
}

// Errors returned by Solve.
var (
	ErrDuplicatePane = errors.New("layout: a pane is placed twice")
	ErrBadArea       = errors.New("layout: the area has a negative size")
	ErrBadSize       = errors.New("layout: a size is invalid")
	ErrBadSplitName  = errors.New("layout: a split name is used twice, or is reserved")
)

// claimSplit records that a split named name is arranged in this solve. An
// empty name claims nothing. A name already claimed, or one that starts
// with "/", is an error, as a pane placed twice is.
func (c *Context) claimSplit(name string) error {
	switch {
	case name == "":
		return nil
	case strings.HasPrefix(name, "/"):
		return fmt.Errorf("%w: %q starts with \"/\", which positional separator IDs use", ErrBadSplitName, name)
	case c.splits[name]:
		return fmt.Errorf("%w: %q names two splits arranged in one layout", ErrBadSplitName, name)
	}
	if c.splits == nil {
		c.splits = map[string]bool{}
	}
	c.splits[name] = true
	return nil
}

// Place records pane id at r. A pane hidden by State, or given no cells, is
// not placed.
func (c *Context) Place(id PaneID, r Rect) error {
	if _, dup := c.plan.Panes[id]; dup {
		return fmt.Errorf("%w: %q", ErrDuplicatePane, id)
	}
	if c.state.IsHidden(id) || r.Empty() {
		return nil
	}
	c.plan.Panes[id] = r
	c.plan.Order = append(c.plan.Order, id)
	return nil
}

// Arrange arranges child n in area, as the i-th child of the current node.
func (c *Context) Arrange(n Node, area Rect) error {
	if n == nil {
		return nil
	}
	return n.Arrange(area, c)
}

// State returns the state Solve was given.
func (c *Context) State() State { return c.state }

// Solve arranges root in area, applying st.
func Solve(root Node, area Rect, st State) (Plan, error) {
	if area.W < 0 || area.H < 0 {
		return Plan{}, ErrBadArea
	}
	if err := st.valid(); err != nil {
		return Plan{}, err
	}
	plan := Plan{Panes: map[PaneID]Rect{}}
	ctx := &Context{state: st, plan: &plan}
	known := leaves(root)
	if len(dedupe(known)) != len(known) {
		return Plan{}, ErrDuplicatePane
	}
	if st.Zoom != "" && slices.Contains(known, st.Zoom) && !area.Empty() {
		plan.Panes[st.Zoom] = area
		plan.Order = []PaneID{st.Zoom}
	} else if err := ctx.Arrange(root, area); err != nil {
		return Plan{}, err
	}
	for _, id := range known {
		if _, ok := plan.Panes[id]; !ok {
			plan.Hidden = append(plan.Hidden, id)
		}
	}
	return plan, nil
}
