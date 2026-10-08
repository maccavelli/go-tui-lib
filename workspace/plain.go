package workspace

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
)

// A plain, linear rendering, for a pipe, a log or a screen reader, and help
// drawn with the workspace's glyphs
// (docs/decisions/0014-MADR-native-integration-api.md W2).

// PlainViewer gives a pane its own plain rendering for RenderPlain: text at
// most width cells wide, with no escape sequences and no padding. Without
// it, RenderPlain uses the pane's View with its escape sequences removed.
type PlainViewer interface{ PlainView(width int) string }

// RenderPlain renders the workspace as plain lines, one pane after another,
// for output that is read rather than looked at. It lays the workspace out
// at width × the current height (80 × 24 before any size), then writes each
// pane the layout shows, the focus ring's first in its order, then the rest
// in tree order:
//  1. its title and badge, on one line;
//  2. its PlainView(width), or else its View at width × its content height
//     with escape sequences removed; trailing spaces and blank lines are
//     dropped;
//  3. a blank line.
//
// Hidden and zero-size panes and overlays are left out. It changes nothing:
// not the layout, the panes' sizes, the frame or the view cache.
func (w *Workspace) RenderPlain(width int) string {
	if width <= 0 {
		return ""
	}
	height := w.height
	if height <= 0 {
		height = fallbackHeight
	}
	plan, err := layout.Solve(w.withMinimums(w.root), layout.Rect{W: width, H: height}, w.state)
	if err != nil {
		plan = w.plan
	}
	var b strings.Builder
	for _, id := range plainOrder(plan, w.ring) {
		p := w.panes[id]
		if p == nil {
			continue
		}
		b.WriteString(w.plainTitle(p, string(id)))
		b.WriteByte('\n')
		var body string
		if pv, ok := p.(PlainViewer); ok {
			body = pv.PlainView(width)
		} else {
			body = p.View(width, w.content(id, plan.Panes[id]).H)
		}
		if body = plainText(body); body != "" {
			b.WriteString(body)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// plainOrder is the panes plan shows: ring's first, in its order, then the
// rest in tree order.
func plainOrder(plan layout.Plan, ring []layout.PaneID) []layout.PaneID {
	out := make([]layout.PaneID, 0, len(plan.Order))
	for _, id := range ring {
		if _, shown := plan.Panes[id]; shown && slices.Contains(plan.Order, id) && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	for _, id := range plan.Order {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

// plainTitle is a pane's title and badge, without escapes, on one line.
func (w *Workspace) plainTitle(p Pane, id string) string {
	name := id
	if tp, ok := p.(Titled); ok {
		name = tp.Title()
	}
	if bp, ok := p.(Badged); ok && bp.Badge() != "" {
		g := w.theme.Glyphs
		name += " " + g.BadgeOpen + bp.Badge() + g.BadgeClose
	}
	return strings.Join(strings.Fields(ansi.Strip(name)), " ")
}

// plainText is s without escape sequences, trailing spaces on each line,
// or trailing blank lines.
func plainText(s string) string {
	lines := strings.Split(ansi.Strip(s), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t\r")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// Help is a bubbles help model for the workspace's bindings, drawn with its
// glyphs and styles: the short help's separator is the glyph set's Bullet,
// its ellipsis the glyph set's Ellipsis, so an ASCII glyph set draws only
// ASCII. The full help keeps bubbles' separator of four spaces. Its width
// is unset, as help.New's is.
func (w *Workspace) Help() help.Model {
	g, st := w.theme.Glyphs, w.theme.Styles
	m := help.New()
	m.ShortSeparator = " " + g.Bullet + " "
	m.Ellipsis = g.Ellipsis
	m.Styles = help.Styles{
		Ellipsis:       st.Muted,
		ShortKey:       st.Body,
		ShortDesc:      st.Muted,
		ShortSeparator: st.Muted,
		FullKey:        st.Body,
		FullDesc:       st.Muted,
		FullSeparator:  st.Muted,
	}
	return m
}
