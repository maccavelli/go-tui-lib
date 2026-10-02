package layout

// Span is where a preset's bottom pane runs.
type Span uint8

const (
	// FullWidth runs the bottom pane under the main pane and the sidebar.
	FullWidth Span = iota
	// UnderMain runs the bottom pane under the main pane only; the sidebar
	// keeps the full height.
	UnderMain
)

// The names of the presets' splits, and so of their separators in
// State.Resize: "sidebar:0" is the boundary between the main pane and the
// sidebar (whichever side it is on), and "bottom:0" the boundary above the
// bottom pane.
const (
	SplitSidebar = "sidebar"
	SplitBottom  = "bottom"
	SplitFooter  = "footer"
	SplitFolded  = "folded"
)

// PresetOption changes a preset.
type PresetOption func(*preset)

type preset struct {
	sidebar, bottom, main Size
	span                  Span
	footer                PaneID
	footerRows            int
	gap                   int
	foldBelow, hideBelow  int // widths
	bottomBelow           int // height
	responsive            bool
}

func defaults() preset {
	return preset{
		main:        Fill(1).AtLeast(30).ShrinkOrder(1),
		sidebar:     Percent(30).AtLeast(24).AtMost(56),
		bottom:      Percent(30).AtLeast(5).AtMost(20).ShrinkFirst(),
		gap:         1,
		foldBelow:   100,
		hideBelow:   70,
		bottomBelow: 16,
		responsive:  true,
	}
}

// SidebarWidth sets the sidebar's claim on the width (default 30%, between
// 24 and 56 cells).
func SidebarWidth(s Size) PresetOption { return func(p *preset) { p.sidebar = s } }

// BottomHeight sets the bottom pane's claim on the height (default 30%,
// between 5 and 20 rows).
func BottomHeight(s Size) PresetOption { return func(p *preset) { p.bottom = s } }

// MainSize sets the main pane's claim (default Fill(1), at least 30 cells).
func MainSize(s Size) PresetOption { return func(p *preset) { p.main = s } }

// BottomSpan sets where the bottom pane runs (default FullWidth).
func BottomSpan(s Span) PresetOption { return func(p *preset) { p.span = s } }

// Footer adds a footer pane of rows rows under everything, such as a status
// line. It is never resized and never folded.
func Footer(id PaneID, rows int) PresetOption {
	return func(p *preset) { p.footer, p.footerRows = id, rows }
}

// Gap sets the cells between neighbouring panes (default 1, for a
// separator). Use 0 when the panes draw their own borders.
func Gap(n int) PresetOption { return func(p *preset) { p.gap = n } }

// Breakpoints sets the responsive folds: below foldWidth columns the
// sidebar moves under the main pane; below hideWidth it is hidden; below
// hideBottomHeight rows the bottom pane is hidden. Defaults: 100, 70, 16.
func Breakpoints(foldWidth, hideWidth, hideBottomHeight int) PresetOption {
	return func(p *preset) { p.foldBelow, p.hideBelow, p.bottomBelow = foldWidth, hideWidth, hideBottomHeight }
}

// NoResponsive keeps the full arrangement at every size; the solver still
// shrinks and hides panes that do not fit, in shrink order.
func NoResponsive() PresetOption { return func(p *preset) { p.responsive = false } }

// SidebarRight is a main pane with a sidebar on its right.
func SidebarRight(main, side PaneID, o ...PresetOption) Node {
	return build(main, side, "", false, o)
}

// SidebarLeft is a main pane with a sidebar on its left.
func SidebarLeft(main, side PaneID, o ...PresetOption) Node {
	return build(main, side, "", true, o)
}

// SidebarRightBottom is SidebarRight with a bottom pane.
func SidebarRightBottom(main, side, bottom PaneID, o ...PresetOption) Node {
	return build(main, side, bottom, false, o)
}

// SidebarLeftBottom is SidebarLeft with a bottom pane.
func SidebarLeftBottom(main, side, bottom PaneID, o ...PresetOption) Node {
	return build(main, side, bottom, true, o)
}

func build(main, side, bottom PaneID, left bool, opts []PresetOption) Node {
	p := defaults()
	for _, o := range opts {
		o(&p)
	}
	full := p.arrange(main, side, bottom, left, true, bottom != "")
	root := full
	if p.responsive {
		r := Responsive{}
		withBottom := bottom != ""
		if withBottom {
			r.Rules = []Rule{
				{When: And(MinWidth(p.foldBelow), MinHeight(p.bottomBelow)), Use: full},
				{When: MinWidth(p.foldBelow), Use: p.arrange(main, side, bottom, left, true, false)},
				{When: And(MinWidth(p.hideBelow), MinHeight(p.bottomBelow)), Use: p.folded(main, side, bottom, true)},
				{When: MinWidth(p.hideBelow), Use: p.folded(main, side, bottom, false)},
				{When: MinHeight(p.bottomBelow), Use: p.stack(main, bottom)},
			}
		} else {
			r.Rules = []Rule{
				{When: MinWidth(p.foldBelow), Use: full},
				{When: MinWidth(p.hideBelow), Use: p.folded(main, side, "", false)},
			}
		}
		r.Else = Pane{ID: main}
		root = r
	}
	if p.footer == "" {
		return root
	}
	return Split{Name: SplitFooter, Axis: Vertical, Children: []Child{
		{Node: root, Size: Fill(1)},
		{Node: Pane{ID: p.footer}, Size: Fixed(p.footerRows).AtLeast(p.footerRows).ShrinkOrder(2)},
	}}
}

// arrange is the full arrangement: main and sidebar side by side, and the
// bottom pane, when withBottom, across both or under main.
func (p preset) arrange(main, side, bottom PaneID, left, withSide, withBottom bool) Node {
	mainNode := Node(Pane{ID: main})
	if withBottom && p.span == UnderMain {
		mainNode = p.stack(main, bottom)
	}
	row := mainNode
	if withSide {
		kids := []Child{{Node: mainNode, Size: p.main}, {Node: Pane{ID: side}, Size: p.sidebar}}
		if left {
			kids[0], kids[1] = kids[1], kids[0]
		}
		row = Split{Name: SplitSidebar, Axis: Horizontal, Gap: p.gap, Children: kids}
	}
	if withBottom && p.span == FullWidth {
		return Split{Name: SplitBottom, Axis: Vertical, Gap: p.gap, Children: []Child{
			{Node: row, Size: Fill(1).AtLeast(5).ShrinkOrder(1)},
			{Node: Pane{ID: bottom}, Size: p.bottom},
		}}
	}
	return row
}

// folded stacks the sidebar under the main pane, and the bottom pane under
// both when withBottom.
func (p preset) folded(main, side, bottom PaneID, withBottom bool) Node {
	kids := []Child{
		{Node: Pane{ID: main}, Size: Fill(2).AtLeast(5).ShrinkOrder(1)},
		{Node: Pane{ID: side}, Size: Fill(1).AtLeast(3)},
	}
	if withBottom {
		kids = append(kids, Child{Node: Pane{ID: bottom}, Size: p.bottom})
	}
	return Split{Name: SplitFolded, Axis: Vertical, Gap: p.gap, Children: kids}
}

// stack is the main pane over the bottom pane.
func (p preset) stack(main, bottom PaneID) Node {
	return Split{Name: SplitBottom, Axis: Vertical, Gap: p.gap, Children: []Child{
		{Node: Pane{ID: main}, Size: Fill(1).AtLeast(5).ShrinkOrder(1)},
		{Node: Pane{ID: bottom}, Size: p.bottom},
	}}
}
