package ask

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// The layout's spacing, as gobble-cli's 0029-MADR-permission-prompt-layout.md
// D1 draws it.
const (
	indent    = 2 // a title's continuation rows, and the rows of answers
	gap       = 2 // between a row's label and its answers, and between answers
	cancelGap = 3 // before the cancel answer, at the end of the last row
)

// Layout is q laid out at width cells, measured with m, its prompt from g:
// the notes, the title, the lines after it, the rows of answers, and the
// prompt row. Each string is cleaned as one line of display text. A width
// of 0 or less wraps nothing.
func Layout(q Question, width int, m ansi.Method, g glyph.Set) []string {
	var out []string
	text := func(s string) {
		out = append(out, flow(m, width, "", words(s), indent)...)
	}
	for _, n := range q.Notes {
		text(n)
	}
	text(q.Title)
	for _, a := range q.After {
		text(a)
	}
	labelW := 0
	for _, r := range q.Rows {
		if r.Label != "" {
			labelW = max(labelW, m.StringWidth(sanitize.Line(r.Label)))
		}
	}
	cancel := token{s: answerText(q.Cancel), sep: cancelGap}
	for i, r := range q.Rows {
		lead := strings.Repeat(" ", indent)
		if r.Label != "" {
			l := sanitize.Line(r.Label)
			lead += l + strings.Repeat(" ", labelW-m.StringWidth(l)+gap)
		}
		var toks []token
		for _, a := range r.Answers {
			toks = append(toks, token{s: answerText(a), sep: gap})
		}
		if i == len(q.Rows)-1 && q.Cancel.Key != "" {
			toks = append(toks, cancel)
		}
		out = append(out, flow(m, width, lead, toks, m.StringWidth(lead))...)
	}
	if len(q.Rows) == 0 && q.Cancel.Key != "" {
		out = append(out, flow(m, width, strings.Repeat(" ", indent), []token{cancel}, indent)...)
	}
	return append(out, g.Prompt)
}

// answerText is "[k] label", cleaned.
func answerText(a Answer) string {
	return "[" + sanitize.Line(a.Key) + "] " + sanitize.Line(a.Label)
}

// token is one word or answer, and the cells that separate it from the one
// before it on a row.
type token struct {
	s   string
	sep int
}

// words is s cleaned, as tokens one space apart: broken at each grapheme
// cluster that is a space alone, so a space and a mark that joins it stay
// one cluster, inside a word (0023-MADR A5 item 7).
func words(s string) []token {
	var toks []token
	var w strings.Builder
	end := func() {
		if w.Len() > 0 {
			toks = append(toks, token{s: w.String(), sep: 1})
			w.Reset()
		}
	}
	for rest := sanitize.Line(s); rest != ""; {
		c, _ := ansi.FirstGraphemeCluster(rest, ansi.GraphemeWidth)
		rest = rest[len(c):]
		if c == " " {
			end()
			continue
		}
		w.WriteString(c)
	}
	end()
	return toks
}

// flow lays toks out in rows of at most width cells: the first row starts
// with lead, each row after it with hang cells of space. A token is never
// broken unless it is wider than a row, when it is broken at the width by
// grapheme cluster, its pieces on rows of their own; a hang that leaves no
// room is dropped, and so is a lead wider than the width, onto rows of its
// own. A width of 0 or less is one row. A row ends with no space.
func flow(m ansi.Method, width int, lead string, toks []token, hang int) []string {
	if width <= 0 {
		var row strings.Builder
		row.WriteString(lead)
		for i, t := range toks {
			if i > 0 {
				row.WriteString(strings.Repeat(" ", t.sep))
			}
			row.WriteString(t.s)
		}
		return []string{strings.TrimRight(row.String(), " ")}
	}
	if hang >= width {
		hang = 0
	}
	pad := strings.Repeat(" ", hang)
	var rows []string
	cur, curW, empty := lead, m.StringWidth(lead), true
	if curW > width {
		if l := strings.TrimRight(lead, " "); l != "" {
			rows = append(rows, pieces(m, l, width, width)...)
		}
		cur, curW = pad, hang
	}
	flush := func() {
		if strings.TrimSpace(cur) != "" {
			rows = append(rows, strings.TrimRight(cur, " "))
		}
		cur, curW, empty = pad, hang, true
	}
	for _, t := range toks {
		tw := m.StringWidth(t.s)
		sep := t.sep
		if empty {
			sep = 0
		}
		if curW+sep+tw <= width {
			cur += strings.Repeat(" ", sep) + t.s
			curW += sep + tw
			empty = false
			continue
		}
		switch {
		case !empty || strings.TrimSpace(cur) != "":
			flush() // a row with answers, or a label alone, ends
		case curW >= width:
			cur, curW = pad, hang // a blank lead that fills the row gives way
		}
		if curW+tw <= width {
			cur += t.s
			curW += tw
			empty = false
			continue
		}
		for i, p := range pieces(m, t.s, width-curW, width-hang) {
			if i > 0 {
				if strings.TrimSpace(cur) != "" {
					rows = append(rows, strings.TrimRight(cur, " "))
				}
				cur, curW = pad, hang
			}
			pw := m.StringWidth(p)
			if curW+pw > width && strings.TrimSpace(cur) == "" {
				cur, curW = "", 0 // a cluster wider than the room the hang leaves
			}
			cur += p
			curW += pw
		}
		empty = false
	}
	if !empty || strings.TrimSpace(cur) != "" {
		rows = append(rows, strings.TrimRight(cur, " "))
	}
	return rows
}

// pieces is s broken by grapheme cluster into a first piece of at most
// first cells and the rest of at most rest; a cluster wider than its
// piece's room is a piece of its own.
func pieces(m ansi.Method, s string, first, rest int) []string {
	var out []string
	cur, curW, room := "", 0, first
	for s != "" {
		c, _ := ansi.FirstGraphemeCluster(s, m)
		s = s[len(c):]
		cw := m.StringWidth(c)
		if curW+cw > room && (cur != "" || len(out) == 0 && room < rest) {
			out = append(out, cur)
			cur, curW, room = "", 0, rest
		}
		cur += c
		curW += cw
	}
	return append(out, cur)
}
