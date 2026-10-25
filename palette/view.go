package palette

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// minText is the narrowest a row's text is cut to before its hint, and
// then its badge, give way.
const minText = 8

// restyle takes the input's prompt, mask, styles and width from the
// workspace's theme now, so a theme change shows without reopening.
func (p *Picker) restyle() {
	t := p.w.Theme()
	p.input.Prompt = t.Glyphs.Prompt + " "
	if r := []rune(t.Glyphs.Mask); len(r) > 0 {
		p.input.EchoCharacter = r[0]
	}
	st := textinput.StyleState{Text: t.Styles.Body, Placeholder: t.Styles.Muted, Prompt: t.Styles.Accent}
	p.input.SetStyles(textinput.Styles{Focused: st, Blurred: st})
	p.input.SetWidth(max(p.width-p.w.WidthMethod().StringWidth(p.input.Prompt)-1, 1))
}

// View draws the picker into width × height cells: the input, a window of
// the results, and a help line with the selected hit's description and a
// count. Rows are measured and cut in the workspace's width method
// (0008-MADR §7, A5.3).
func (p *Picker) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	p.restyle()
	t, m := p.w.Theme(), p.w.WidthMethod()
	out := []string{sanitize.Truncate(p.input.View(), width, m, "")}
	if height == 1 {
		return out[0]
	}
	rows, ls := p.eng.rows(), p.lines()
	for i := p.top; i < min(p.top+height-2, len(ls)); i++ {
		l := ls[i]
		if l.row < 0 {
			out = append(out, t.Styles.Muted.Render(sanitize.Truncate(l.header, width, m, t.Glyphs.Ellipsis)))
			continue
		}
		out = append(out, p.drawRow(rows[l.row], l.row == p.sel, width))
	}
	for len(out) < height-1 {
		out = append(out, "")
	}
	return strings.Join(append(out, p.helpLine(rows, width)), "\n")
}

// drawRow draws one row in width cells: the selection marker, the text with
// its matched cells highlighted, a danger badge, and a key hint at the
// right.
func (p *Picker) drawRow(r row, selected bool, width int) string {
	t, m := p.w.Theme(), p.w.WidthMethod()
	g := t.Glyphs
	base := t.Styles.Body
	if r.failed || r.hit.Disabled != "" {
		base = t.Styles.Muted
	}
	hl := t.Styles.Accent.Underline(true)
	marker := strings.Repeat(" ", m.StringWidth(g.Focus))
	if selected {
		base, hl = base.Bold(true), hl.Bold(true)
		marker = g.Focus
	}
	marker += " "

	text, badge, hint := r.hit.Text, "", r.hit.Hint
	var badgeStyle lipgloss.Style
	switch {
	case r.failed:
		text = g.Failed + " " + text + " failed"
	case p.eng.prefix:
		hint = r.hit.Group
	default:
		switch r.hit.Danger {
		case command.Mutating:
			badge, badgeStyle = g.BadgeOpen+r.hit.Danger.String()+g.BadgeClose, t.Styles.Warning
		case command.Destructive:
			badge, badgeStyle = g.BadgeOpen+r.hit.Danger.String()+g.BadgeClose, t.Styles.Error
		}
		if hint == "" && r.hit.Command != "" {
			hint = p.w.Bindings().Shortcut(r.hit.Command, p.path, p.cfg.goos, g)
		}
		if r.hit.Disabled != "" {
			hint = r.hit.Disabled // shown with its reason, in words, where a key would be
		}
	}

	room := width - m.StringWidth(marker)
	right := func() int {
		n := 0
		if badge != "" {
			n += 1 + m.StringWidth(badge)
		}
		if hint != "" {
			n += 1 + m.StringWidth(hint)
		}
		return n
	}
	if hint != "" && room-right() < minText {
		hint = ""
	}
	if badge != "" && room-right() < minText {
		badge = ""
	}
	avail := max(room-right(), 0)

	var cells []fuzzy.CellRange
	if mt := p.eng.search.Matcher; mt != nil && !r.failed && !p.eng.prefix {
		p.matched++
		if hit, ok := mt.Match(text); ok {
			cells = fuzzy.Cells(text, hit.Ranges, m)
		}
	}
	body := styled(text, cells, avail, m, base, hl, g.Ellipsis)

	var b strings.Builder
	b.WriteString(marker)
	b.WriteString(body)
	used := m.StringWidth(marker) + m.StringWidth(ansi.Strip(body))
	if badge != "" {
		b.WriteString(" ")
		b.WriteString(badgeStyle.Render(badge))
		used += 1 + m.StringWidth(badge)
	}
	if hint != "" {
		pad := width - used - m.StringWidth(hint)
		b.WriteString(strings.Repeat(" ", max(pad, 1)))
		b.WriteString(t.Styles.Muted.Render(hint))
	}
	return b.String()
}

// styled draws s in at most cells cells measured with m: the cells cs
// cover in hl, the rest in base, ended with ellipsis when s is wider, and
// a highlight in the part cut off not drawn. Each piece is cut by the
// repository's cutter, so a highlight lands on the cells the workspace
// measures, under either width method (0008-MADR A5.3).
func styled(s string, cs []fuzzy.CellRange, cells int, m ansi.Method, base, hl lipgloss.Style, ellipsis string) string {
	end, tail := m.StringWidth(s), ""
	if end > cells {
		if ew := m.StringWidth(ellipsis); ew <= cells {
			end, tail = cells-ew, ellipsis
		} else {
			end = cells
		}
	}
	var b strings.Builder
	put := func(st lipgloss.Style, piece string) {
		if piece != "" {
			b.WriteString(st.Render(piece))
		}
	}
	at := 0
	for _, c := range cs {
		from, to := max(c.Start, at), min(c.End, end)
		if from >= to {
			continue
		}
		put(base, cut(s, at, from, m))
		put(hl, cut(s, from, to, m))
		at = to
	}
	put(base, cut(s, at, end, m))
	put(base, tail)
	return b.String()
}

// cut is the text of cells [a, b) of s under m: sanitize.Truncate(s, b)
// less its prefix sanitize.Truncate(s, a), as fuzzy's Cells oracle cuts
// (0008-MADR A4 item 1).
func cut(s string, a, b int, m ansi.Method) string {
	if a >= b {
		return ""
	}
	head := sanitize.Truncate(s, a, m, "")
	return strings.TrimPrefix(sanitize.Truncate(s, b, m, ""), head)
}

// helpLine is the selected hit's description, or a disabled hit's reason,
// at the left, and the count at the right: "3 of 40", and while providers
// still search, ", searching" and the ellipsis glyph, in words, as nothing
// animates (0008-MADR §7).
func (p *Picker) helpLine(rows []row, width int) string {
	t, m := p.w.Theme(), p.w.WidthMethod()
	count := "no results"
	if len(rows) > 0 {
		count = fmt.Sprintf("%d of %d", p.sel+1, len(rows))
	}
	if p.eng.pending > 0 {
		if len(rows) == 0 {
			count = "searching" + t.Glyphs.Ellipsis
		} else {
			count += ", searching" + t.Glyphs.Ellipsis
		}
	}
	count = sanitize.Truncate(count, width, m, t.Glyphs.Ellipsis)
	help := ""
	if r, ok := p.selected(); ok && !r.failed {
		help = r.hit.Help
		if r.hit.Disabled != "" {
			help = r.hit.Disabled
		}
	}
	room := width - m.StringWidth(count) - 1
	if room < 1 || help == "" {
		return strings.Repeat(" ", max(width-m.StringWidth(count), 0)) + t.Styles.Muted.Render(count)
	}
	help = sanitize.Truncate(help, room, m, t.Glyphs.Ellipsis)
	pad := width - m.StringWidth(help) - m.StringWidth(count)
	return t.Styles.Muted.Render(help) + strings.Repeat(" ", max(pad, 1)) + t.Styles.Muted.Render(count)
}
