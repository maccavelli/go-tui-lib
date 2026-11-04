package stream

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Renderer turns Markdown into styled text at a width in cells
// (docs/decisions/0009-MADR-streaming-content-engine.md §4). A Doc gives it
// cleaned source, one call at a time, so it need not be safe for
// concurrent use.
type Renderer interface {
	Render(src string, width int) (string, error)
}

// RendererFunc is a function used as a Renderer.
type RendererFunc func(src string, width int) (string, error)

// Render calls f.
func (f RendererFunc) Render(src string, width int) (string, error) { return f(src, width) }

// plain is Plain's Renderer.
type plain struct{ method ansi.Method }

// Plain is a Renderer that styles nothing: the Printer's drawing with the
// zero Markup, prose wrapped at whole words to the width measured by m,
// and tables laid out in aligned columns by FitTable at the width, every
// marker kept; where the Printer's tables are never wrapped, a pane, which
// clips a row, needs them to fit
// (docs/decisions/0009-PLAN-streaming-content-engine.md D20)
// (docs/decisions/0009-MADR-streaming-content-engine.md A6 item 6). It
// prints no reference or footnote definition, which a Doc gives each
// segment it renders (docs/decisions/0009-PLAN-streaming-content-engine.md
// D16). Its source is cleaned already, as a Doc's is; it never fails.
func Plain(m ansi.Method) Renderer { return plain{m} }

// Render draws src at width cells.
func (r plain) Render(src string, width int) (string, error) { return r.drawWith(src, width, nil), nil }

// drawWith draws src at width cells, its code highlighted by h when it is
// not nil.
func (r plain) drawWith(src string, width int, h *highlight) string {
	q := &Printer{method: r.method, width: width, noDefs: true, fit: true, defs: map[string]bool{}, hl: h}
	q.clean.b.WriteString(src)
	q.end()
	return q.drain()
}

// holdback is a partial line as a Doc shows it: up to the first inline
// construct it opens and has not closed, as goose holds a streamed line
// (docs/reports/0003-REPORT-agent-tui-ecosystem-research.md §3.5): an
// emphasis or strong run of * or _, a strikethrough ~~, a run of backticks
// with no run of as many after it, or a link whose ]( has come and whose )
// has not; and, with math, a \( with no \) after it, or a $$ with no $$
// after it (docs/decisions/0009-MADR-streaming-content-engine.md A1 §4).
// "Hello **wor" is shown as "Hello ".
func holdback(s string, math bool) string {
	type open struct {
		at    int
		delim string
	}
	var stack []open
	cut := len(s)
	for i := 0; i < len(s); {
		c := s[i]
		if math {
			switch n, open := mathSpan(s[i:]); {
			case open:
				return s[:min(cut, i)]
			case n > 0:
				i += n
				continue
			}
		}
		switch c {
		case '\\':
			i += 2
			continue
		case '`':
			j, closed := codeSpanEnd(s, i)
			if !closed {
				return s[:min(cut, i)]
			}
			i = j
			continue
		case '*', '_', '~':
			n := holdRun(s, i)
			if c == '~' && n != 2 {
				i += n
				continue
			}
			d := s[i : i+n]
			before, after := holdBefore(s, i), holdAt(s, i+n)
			opens, closes := !holdSpace(after), !holdSpace(before)
			if c == '_' && holdWord(before) && holdWord(after) {
				opens, closes = false, false // inside a word: snake_case
			}
			if k := len(stack) - 1; closes && k >= 0 && stack[k].delim == d {
				stack = stack[:k]
			} else if opens {
				stack = append(stack, open{i, d})
			}
			i += n
			continue
		case '[':
			stack = append(stack, open{i, "["})
		case ']':
			k := len(stack) - 1
			for k >= 0 && stack[k].delim != "[" {
				k--
			}
			if k >= 0 {
				start := stack[k].at
				stack = stack[:k]
				if strings.HasPrefix(s[i+1:], "(") {
					end := strings.IndexByte(s[i+2:], ')')
					if end < 0 {
						return s[:min(cut, start)]
					}
					i += 3 + end
					continue
				}
			}
		}
		i++
	}
	for _, o := range stack {
		if o.delim != "[" {
			cut = min(cut, o.at)
		}
	}
	return s[:cut]
}

// codeSpanEnd is the offset after the code span whose backtick run begins
// at s[i], and whether a run of as many closes it.
func codeSpanEnd(s string, i int) (int, bool) {
	n := holdRun(s, i)
	for j := i + n; j < len(s); {
		k := strings.IndexByte(s[j:], '`')
		if k < 0 {
			break
		}
		j += k
		m := holdRun(s, j)
		if j += m; m == n {
			return j, true
		}
	}
	return 0, false
}

// mathSpan is the length of the math s starts with, \( to \) or $$ to $$,
// and whether it is open, its closer not come; 0 when s starts with none.
func mathSpan(s string) (n int, open bool) {
	for _, d := range [][2]string{{`\(`, `\)`}, {"$$", "$$"}} {
		if strings.HasPrefix(s, d[0]) {
			end := strings.Index(s[2:], d[1])
			if end < 0 {
				return 2, true
			}
			return 4 + end, false
		}
	}
	return 0, false
}

// holdRun is the length of the run of s[i]'s byte at i.
func holdRun(s string, i int) int {
	n := 1
	for i+n < len(s) && s[i+n] == s[i] {
		n++
	}
	return n
}

// holdBefore is the byte before i, or a space at the start.
func holdBefore(s string, i int) byte {
	if i == 0 {
		return ' '
	}
	return s[i-1]
}

// holdAt is the byte at i, or a space at the end.
func holdAt(s string, i int) byte {
	if i >= len(s) {
		return ' '
	}
	return s[i]
}

// holdSpace reports a space or a tab, which a run must not be next to on
// the side it opens or closes.
func holdSpace(c byte) bool { return c == ' ' || c == '\t' }

// holdWord reports a byte of a word, between which an _ opens nothing.
func holdWord(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}
