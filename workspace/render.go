package workspace

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
)

// Render composes the frame: every placed pane in its rectangle with its
// chrome, the separators, and the overlays on top. Each pane is clipped to
// its rectangle. The program puts the result in its tea.View.
func (w *Workspace) Render() string {
	if w.width <= 0 || w.height <= 0 {
		return ""
	}
	var layers []*lipgloss.Layer
	w.inner = map[string]layout.Rect{}
	for _, id := range w.plan.Order {
		r := w.plan.Panes[id]
		lid := layerID("pane", string(id))
		w.inner[lid] = w.content(id, r)
		layers = append(layers, lipgloss.NewLayer(w.renderPane(id, r)).X(r.X).Y(r.Y).ID(lid))
	}
	for _, s := range w.plan.Separators {
		if s.Rect.Empty() {
			continue
		}
		layers = append(layers, lipgloss.NewLayer(w.renderSeparator(s)).X(s.Rect.X).Y(s.Rect.Y).Z(1).ID(layerID("sep", s.ID)))
	}
	for i, o := range w.overlays {
		r := w.overlayRect(o)
		lid := layerID("overlay", o.ID)
		w.inner[lid] = insetBorder(r)
		layers = append(layers, lipgloss.NewLayer(w.renderBox(r, o.Pane, viewKey{overlayView, o.ID}, true)).X(r.X).Y(r.Y).Z(10+i).ID(lid))
	}
	w.hits = lipgloss.NewCompositor(layers...)
	canvas := lipgloss.NewCanvas(w.width, w.height)
	canvas.Compose(w.hits)
	return canvas.Render()
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
// at the same size and focus as its cached view under k.
func (w *Workspace) view(k viewKey, p Pane, width, height int, focused bool) string {
	if c, ok := p.(Changer); ok && !c.Changed() {
		if hit, ok := w.cache[k]; ok && hit.width == width && hit.height == height && hit.focused == focused {
			return hit.view
		}
	}
	v := clip(p.View(width, height), width, height)
	w.cache[k] = cached{width: width, height: height, focused: focused, view: v}
	return v
}

// clip makes s exactly width × height cells: lines truncated, padded and
// limited, so a pane can never paint outside its area.
func clip(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, l := range lines {
		l = ansi.Truncate(l, width, "")
		if pad := width - ansi.StringWidth(l); pad > 0 {
			l += strings.Repeat(" ", pad)
		}
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}

func (w *Workspace) renderPane(id layout.PaneID, r layout.Rect) string {
	p := w.panes[id]
	k := viewKey{paneView, string(id)}
	focused := id == w.focus && w.modal() == nil
	switch w.chromeOf(id) {
	case Borders:
		return w.renderBox(r, p, k, focused)
	case Separators:
		in := w.content(id, r)
		head := w.title(p, string(id), focused, r.W)
		if in.H == 0 {
			return head
		}
		return head + "\n" + w.view(k, p, in.W, in.H, focused)
	}
	return w.view(k, p, r.W, r.H, focused)
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
	label = ansi.Truncate(label, width, t.Glyphs.Ellipsis)
	if pad := width - ansi.StringWidth(label); pad > 0 {
		label += strings.Repeat(" ", pad)
	}
	return style.Render(label)
}

// renderBox draws p in r with a border, its title in the top edge. k names
// its cached view, and k.id is the title when p has none of its own.
func (w *Workspace) renderBox(r layout.Rect, p Pane, k viewKey, focused bool) string {
	t := w.theme
	b := t.Border(w.border)
	edge := t.Styles.Border
	if focused {
		edge = t.Styles.BorderFocus
	}
	in := insetBorder(r)
	if in.W == 0 {
		return clip("", r.W, r.H)
	}
	name := k.id
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
	label = ansi.Truncate(label, max(in.W-1, 0), t.Glyphs.Ellipsis)
	fill := in.W - 1 - ansi.StringWidth(label)
	var out strings.Builder
	out.WriteString(edge.Render(b.TopLeft+b.Top) + tstyle.Render(label) + edge.Render(strings.Repeat(b.Top, max(fill, 0))+b.TopRight))
	if in.H > 0 {
		body := w.view(k, p, in.W, in.H, focused)
		for _, line := range strings.Split(body, "\n") {
			out.WriteString("\n" + edge.Render(b.Left) + line + edge.Render(b.Right))
		}
	}
	out.WriteString("\n" + edge.Render(b.BottomLeft+strings.Repeat(b.Bottom, in.W)+b.BottomRight))
	return out.String()
}

func (w *Workspace) renderSeparator(s layout.Separator) string {
	g := w.theme.Glyphs
	if w.chrome != Separators {
		return clip("", s.Rect.W, s.Rect.H)
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
	if w.hits == nil {
		return nil
	}
	kind, id := splitLayerID(w.hits.Hit(m.X, m.Y).ID())
	if o := w.modal(); o != nil && (kind != "overlay" || id != o.ID) {
		return nil // a modal overlay takes the mouse; nothing beneath it is reached
	}
	switch kind {
	case "sep":
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
	case "overlay":
		for i, o := range w.overlays {
			if o.ID == id {
				if lm, ok := local(msg, w.inner[layerID("overlay", id)]); ok {
					return w.updateOverlay(i, lm)
				}
			}
		}
		return nil
	case "pane":
		pid := layout.PaneID(id)
		var cmds []tea.Cmd
		if _, ok := msg.(tea.MouseClickMsg); ok {
			cmds = append(cmds, w.Focus(pid))
		}
		if lm, ok := local(msg, w.inner[layerID("pane", id)]); ok {
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
