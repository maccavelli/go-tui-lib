package workspace

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/internal/cells"
	"github.com/maccavelli/go-tui-lib/layout"
)

// regionKind is what a region of the frame shows.
type regionKind uint8

const (
	paneRegion regionKind = iota
	sepRegion
	overlayRegion
)

// region is a rectangle of the last frame, for hit testing: what it shows,
// where, and its content area inside any chrome.
type region struct {
	kind  regionKind
	id    string
	rect  layout.Rect
	inner layout.Rect
}

// Render composes the frame: every placed pane in its rectangle with its
// chrome, the separators, and the overlays on top. Each pane is clipped to
// its rectangle. The program puts the result in its tea.View.
//
// The frame is drawn into one reused buffer, and only when something changed
// since the last one; otherwise the last frame is returned as it is (see
// Pane).
func (w *Workspace) Render() string {
	if w.width <= 0 || w.height <= 0 {
		return ""
	}
	if !w.dirty && !w.changersChanged() {
		return w.last
	}
	if w.frame == nil {
		w.frame = cells.NewFrame(w.width, w.height, w.method)
	}
	w.frame.Resize(w.width, w.height)
	w.frame.SetMethod(w.method)
	w.frame.Clear()
	w.regions = w.regions[:0]
	for _, id := range w.plan.Order {
		r := w.plan.Panes[id]
		w.frame.Draw(w.renderPane(id, r), r)
		w.regions = append(w.regions, region{kind: paneRegion, id: string(id), rect: r, inner: w.content(id, r)})
	}
	for _, s := range w.plan.Separators {
		if s.Rect.Empty() {
			continue
		}
		w.frame.Draw(w.renderSeparator(s), s.Rect)
		w.regions = append(w.regions, region{kind: sepRegion, id: s.ID, rect: s.Rect, inner: s.Rect})
	}
	for _, o := range w.overlays {
		r := w.overlayRect(o)
		w.frame.Draw(w.renderBox(r, o.Pane, overlayView, o.ID, true), r)
		w.regions = append(w.regions, region{kind: overlayRegion, id: o.ID, rect: r, inner: insetBorder(r)})
	}
	slices.Reverse(w.regions) // top first: the last overlay, then separators, then panes
	w.last = w.frame.Render()
	w.dirty = false
	return w.last
}

// changersChanged reports whether a shown pane or an open overlay that is a
// Changer reports a change.
func (w *Workspace) changersChanged() bool {
	for _, id := range w.plan.Order {
		if c, ok := w.panes[id].(Changer); ok && c.Changed() {
			return true
		}
	}
	for _, o := range w.overlays {
		if c, ok := o.Pane.(Changer); ok && c.Changed() {
			return true
		}
	}
	return false
}

// hit is the top region of the last frame under (x, y).
func (w *Workspace) hit(x, y int) (region, bool) {
	for _, r := range w.regions {
		if r.rect.Contains(x, y) {
			return r, true
		}
	}
	return region{}, false
}

// Cursor returns the terminal cursor for the focused pane, or for the top
// modal overlay, in screen cells; nil when that pane has none, or it falls
// outside the pane.
func (w *Workspace) Cursor() *tea.Cursor {
	if o := w.modal(); o != nil {
		return placeCursor(o.Pane, insetBorder(w.overlayRect(*o)))
	}
	return w.paneCursor()
}

// paneCursor is the focused pane's cursor in screen cells, whatever
// overlays are open. A BelowCursor overlay is placed by it.
func (w *Workspace) paneCursor() *tea.Cursor {
	r, ok := w.plan.Panes[w.focus]
	if !ok {
		return nil
	}
	return placeCursor(w.panes[w.focus], w.content(w.focus, r))
}

// placeCursor moves p's cursor from its own cells into the screen cells of
// its content area in, or returns nil when it has none or it falls outside.
func placeCursor(p Pane, in layout.Rect) *tea.Cursor {
	c, ok := p.(Cursorer)
	if !ok {
		return nil
	}
	cur := c.Cursor()
	if cur == nil || cur.X < 0 || cur.Y < 0 || cur.X >= in.W || cur.Y >= in.H {
		return nil
	}
	out := *cur
	out.X += in.X
	out.Y += in.Y
	return &out
}

// view asks a pane for its view, unless it is a Changer reporting no change
// and its view is cached under the same size, focus, width method and theme.
func (w *Workspace) view(kind viewKind, id string, p Pane, width, height int, focused bool) string {
	k := viewKey{kind: kind, id: id, width: width, height: height, focused: focused, method: w.method, themeGen: w.themeGen}
	if c, ok := p.(Changer); ok && !c.Changed() {
		if v, ok := w.cache[k]; ok {
			return v
		}
	}
	v := clip(w.method, p.View(width, height), width, height)
	w.cache[k] = v
	return v
}

// clip makes s exactly width × height cells, measured with m: lines
// truncated, padded and limited, so a pane can never paint outside its area.
// It writes one builder, with no allocation per line.
func clip(m ansi.Method, s string, width, height int) string {
	var b strings.Builder
	b.Grow(len(s) + height*(width+1))
	n := 0
	for line := range strings.SplitSeq(s, "\n") {
		if n == height {
			break
		}
		if n > 0 {
			b.WriteByte('\n')
		}
		line = m.Truncate(line, width, "")
		b.WriteString(line)
		pad(&b, width-m.StringWidth(line))
		n++
	}
	for ; n < height; n++ {
		if n > 0 {
			b.WriteByte('\n')
		}
		pad(&b, width)
	}
	return b.String()
}

// spaces is the run pad writes from.
const spaces = "                                                                "

// pad writes n spaces to b.
func pad(b *strings.Builder, n int) {
	for n > 0 {
		k := min(n, len(spaces))
		b.WriteString(spaces[:k])
		n -= k
	}
}

func (w *Workspace) renderPane(id layout.PaneID, r layout.Rect) string {
	p := w.panes[id]
	focused := id == w.focus && w.modal() == nil
	switch w.chromeOf(id) {
	case Borders:
		return w.renderBox(r, p, paneView, string(id), focused)
	case Separators:
		in := w.content(id, r)
		head := w.title(p, string(id), focused, r.W)
		if in.H == 0 {
			return head
		}
		return head + "\n" + w.view(paneView, string(id), p, in.W, in.H, focused)
	}
	return w.view(paneView, string(id), p, r.W, r.H, focused)
}

// title renders a pane's title line: the focus marker, the title and the
// badge, truncated to width cells.
func (w *Workspace) title(p Pane, id string, focused bool, width int) string {
	t := w.theme
	name := id
	if tp, ok := p.(Titled); ok {
		name = tp.Title()
	}
	mark := " "
	style := t.Styles.Title
	if focused {
		mark, style = t.Glyphs.Focus, t.Styles.FocusTitle
	}
	label := mark + " " + name
	if b, ok := p.(Badged); ok && b.Badge() != "" {
		label += " " + t.Glyphs.BadgeOpen + b.Badge() + t.Glyphs.BadgeClose
	}
	label = w.method.Truncate(label, width, t.Glyphs.Ellipsis)
	if pad := width - w.method.StringWidth(label); pad > 0 {
		label += strings.Repeat(" ", pad)
	}
	return style.Render(label)
}

// renderBox draws p in r with a border, its title in the top edge. kind and
// id name its cached view, and id is the title when p has none of its own.
func (w *Workspace) renderBox(r layout.Rect, p Pane, kind viewKind, id string, focused bool) string {
	t := w.theme
	b := t.Border(w.border)
	edge := t.Styles.Border
	if focused {
		edge = t.Styles.BorderFocus
	}
	in := insetBorder(r)
	if in.W == 0 {
		return clip(w.method, "", r.W, r.H)
	}
	name := id
	if tp, ok := p.(Titled); ok {
		name = tp.Title()
	}
	mark := ""
	tstyle := t.Styles.Title
	if focused {
		mark, tstyle = t.Glyphs.Focus+" ", t.Styles.FocusTitle
	}
	label := " " + mark + name
	if bd, ok := p.(Badged); ok && bd.Badge() != "" {
		label += " " + t.Glyphs.BadgeOpen + bd.Badge() + t.Glyphs.BadgeClose
	}
	label += " "
	label = w.method.Truncate(label, max(in.W-1, 0), t.Glyphs.Ellipsis)
	fill := in.W - 1 - w.method.StringWidth(label)
	var out strings.Builder
	out.WriteString(edge.Render(b.TopLeft+b.Top) + tstyle.Render(label) + edge.Render(strings.Repeat(b.Top, max(fill, 0))+b.TopRight))
	if in.H > 0 {
		// The edges are styled once per box, not once per row.
		left, right := edge.Render(b.Left), edge.Render(b.Right)
		body := w.view(kind, id, p, in.W, in.H, focused)
		out.Grow(len(body) + in.H*(len(left)+len(right)+1))
		for line := range strings.SplitSeq(body, "\n") {
			out.WriteByte('\n')
			out.WriteString(left)
			out.WriteString(line)
			out.WriteString(right)
		}
	}
	out.WriteString("\n" + edge.Render(b.BottomLeft+strings.Repeat(b.Bottom, in.W)+b.BottomRight))
	return out.String()
}

func (w *Workspace) renderSeparator(s layout.Separator) string {
	g := w.theme.Glyphs
	if w.chrome != Separators {
		return clip(w.method, "", s.Rect.W, s.Rect.H)
	}
	cell := strings.Repeat(g.SeparatorHorizontal, s.Rect.W)
	if s.Axis == layout.Horizontal { // a vertical line between side-by-side panes
		cell = strings.Repeat(g.SeparatorVertical, s.Rect.W)
	}
	rows := make([]string, s.Rect.H)
	for i := range rows {
		rows[i] = cell
	}
	return w.theme.Styles.Border.Render(strings.Join(rows, "\n"))
}

// mouseEvent routes a mouse event by what is under the pointer.
func (w *Workspace) mouseEvent(msg tea.MouseMsg) tea.Cmd {
	m := msg.Mouse()
	if w.drag != nil {
		switch msg.(type) {
		case tea.MouseMotionMsg:
			pos := m.X
			if w.drag.sep.Axis == layout.Vertical {
				pos = m.Y
			}
			d := pos - w.drag.last
			w.drag.last = pos
			if d != 0 {
				return w.Resize(w.drag.sep.ID, d)
			}
			return nil
		case tea.MouseReleaseMsg:
			w.drag = nil
			return nil
		}
	}
	reg, ok := w.hit(m.X, m.Y)
	if o := w.modal(); o != nil && (!ok || reg.kind != overlayRegion || reg.id != o.ID) {
		return nil // a modal overlay takes the mouse; nothing beneath it is reached
	}
	if !ok {
		return nil
	}
	id := reg.id
	switch reg.kind {
	case sepRegion:
		if _, ok := msg.(tea.MouseClickMsg); ok {
			for _, s := range w.plan.Separators {
				if s.ID == id {
					pos := m.X
					if s.Axis == layout.Vertical {
						pos = m.Y
					}
					w.drag = &drag{sep: s, last: pos}
				}
			}
		}
		return nil
	case overlayRegion:
		for i, o := range w.overlays {
			if o.ID == id {
				if lm, ok := local(msg, reg.inner); ok {
					return w.updateOverlay(i, lm)
				}
			}
		}
		return nil
	case paneRegion:
		pid := layout.PaneID(id)
		var cmds []tea.Cmd
		if _, ok := msg.(tea.MouseClickMsg); ok {
			cmds = append(cmds, w.Focus(pid))
		}
		if lm, ok := local(msg, reg.inner); ok {
			cmds = append(cmds, w.Send(pid, lm))
		}
		return tea.Batch(cmds...)
	}
	return nil
}

// local translates a mouse event into area's cells, or reports false when
// the pointer is outside area, such as on a border or a title.
func local(msg tea.MouseMsg, area layout.Rect) (tea.Msg, bool) {
	m := msg.Mouse()
	if !area.Contains(m.X, m.Y) {
		return nil, false
	}
	m.X -= area.X
	m.Y -= area.Y
	switch msg.(type) {
	case tea.MouseClickMsg:
		return tea.MouseClickMsg(m), true
	case tea.MouseReleaseMsg:
		return tea.MouseReleaseMsg(m), true
	case tea.MouseMotionMsg:
		return tea.MouseMotionMsg(m), true
	case tea.MouseWheelMsg:
		return tea.MouseWheelMsg(m), true
	}
	return nil, false
}
