package stream

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Highlighter highlights code (docs/decisions/0009-MADR-streaming-content-engine.md
// A1 §4): lines, continuing from state, and the state after them, which a
// later call continues from, or nil when it cannot resume. out holds a line
// for each of lines, the same text styled; any other out is not used, and
// the lines show plain. A nil state is a block's start.
type Highlighter interface {
	Highlight(lang string, lines []string, state any) (out []string, next any)
}

// HighlighterFunc is a function used as a Highlighter.
type HighlighterFunc func(lang string, lines []string, state any) (out []string, next any)

// Highlight calls f.
func (f HighlighterFunc) Highlight(lang string, lines []string, state any) ([]string, any) {
	return f(lang, lines, state)
}

// The highlighting limits, codex's: a block past any shows plain.
const (
	defaultCodeBytes     = 512 << 10
	defaultCodeLines     = 10_000
	defaultCodeLineBytes = 4 << 10
)

// limits is when code is too much to highlight.
type limits struct{ bytes, lines, line int }

func defaultLimits() limits {
	return limits{bytes: defaultCodeBytes, lines: defaultCodeLines, line: defaultCodeLineBytes}
}

// highlight is how a Printer highlights its fences' lines: by h, within
// lim; a Doc's tail's lines kept in cache, and the fences a Doc found past
// a limit, by their opening line's offset in the document, plain. base is
// the input's offset in the document.
type highlight struct {
	h     Highlighter
	lim   limits
	cache *codeCache
	plain map[int]bool
	base  int
}

// fenceState is the fence a Printer is in: its opening line's offset in
// the input, its language, its lines so far and their bytes, the state
// after the last, and whether it shows plain.
type fenceState struct {
	at     int
	lang   string
	lines  []string
	bytes  int
	state  any
	plain  bool
	inside bool
}

// lang is a fence's language: its info string's first word.
func lang(open, run string) string {
	f := strings.Fields(open[strings.Index(open, run)+len(run):])
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// open starts the fence whose opening line l, of run, is at offset at.
func (p *Printer) openFence(l, run string, at int) {
	p.fence = fenceState{at: at, lang: lang(l, run), inside: true}
	if h := p.hl; h != nil && h.plain[h.base+at] {
		p.fence.plain = true
	}
}

// code is the fence's next line l highlighted, or as it is: with no
// Highlighter, once the fence is past a limit, or when the Highlighter's
// out is not the line. nl is false for a line not yet complete, which a
// Doc's cache does not keep.
func (p *Printer) code(l string, nl bool) string {
	h, f := p.hl, &p.fence
	if h == nil || f.plain {
		return l
	}
	f.bytes += len(l) + 1
	if len(l) > h.lim.line || len(f.lines)+1 > h.lim.lines || f.bytes > h.lim.bytes {
		f.plain = true // a Printer's block past a limit: plain from this line on
		return l
	}
	k := len(f.lines)
	f.lines = append(f.lines, l)
	if c := h.cache; c != nil {
		if out, state, ok := c.get(h.base+f.at, f.lang, k, l); ok {
			f.state = state
			return out
		}
	}
	var out []string
	var next any
	if k == 0 || f.state != nil {
		out, next = h.h.Highlight(f.lang, []string{l}, f.state)
	} else { // the Highlighter could not resume: every line again, from the block's start
		out, next = h.h.Highlight(f.lang, f.lines, nil)
		if len(out) == len(f.lines) {
			out = out[k:]
		}
	}
	s := l
	if len(out) == 1 && ansi.Strip(out[0]) == l {
		s = out[0]
	}
	f.state = next
	if c := h.cache; c != nil && nl {
		c.put(h.base+f.at, f.lang, k, l, s, next, 2*h.lim.bytes)
	}
	return s
}

// codeCache is a Doc's tail's highlighted lines, by fence, its opening
// line's offset in the document; a fence's lines in order, each with the
// state after it.
type codeCache struct {
	fences map[int]*fenceLines
	size   int
}

type fenceLines struct {
	lang   string
	src    []string
	out    []string
	states []any
}

// get is fence at's line k, l, as kept, and the state after it.
func (c *codeCache) get(at int, lang string, k int, l string) (string, any, bool) {
	e := c.fences[at]
	if e == nil || e.lang != lang || k >= len(e.src) || e.src[k] != l {
		return "", nil, false
	}
	return e.out[k], e.states[k], true
}

// put keeps fence at's line k: l, drawn as out, and the state after it;
// every line is dropped when they pass budget bytes.
func (c *codeCache) put(at int, lang string, k int, l, out string, state any, budget int) {
	if c.fences == nil {
		c.fences = map[int]*fenceLines{}
	}
	e := c.fences[at]
	if e == nil || e.lang != lang || k != len(e.src) {
		if e != nil {
			c.size -= e.bytes()
		}
		e = &fenceLines{lang: lang}
		c.fences[at] = e
		if k != 0 {
			return // a line out of order: kept from the block's start only
		}
	}
	e.src, e.out, e.states = append(e.src, l), append(e.out, out), append(e.states, state)
	c.size += len(l) + len(out)
	if c.size > budget {
		c.reset()
	}
}

func (e *fenceLines) bytes() int {
	n := 0
	for i := range e.src {
		n += len(e.src[i]) + len(e.out[i])
	}
	return n
}

// reset drops every kept line.
func (c *codeCache) reset() { c.fences, c.size = nil, 0 }

// keep drops the fences that begin before from, which a cut committed.
func (c *codeCache) keep(from int) {
	for at, e := range c.fences {
		if at < from {
			c.size -= e.bytes()
			delete(c.fences, at)
		}
	}
}

// overLimits is the fences of src, by their opening line's offset plus
// base, that pass a limit: a whole block past any shows plain in a Doc.
func overLimits(src string, base int, lim limits) map[int]bool {
	var s Scanner
	var out map[int]bool
	at, lines, bytes, over := -1, 0, 0, false
	for off := 0; off < len(src); {
		end := strings.IndexByte(src[off:], '\n')
		l := src[off:]
		if end >= 0 {
			l = src[off : off+end]
		}
		switch s.Scan(l).Kind {
		case BlockFence:
			at, lines, bytes, over = off, 0, 0, false
		case BlockCode:
			lines++
			bytes += len(l) + 1
			over = over || len(l) > lim.line || lines > lim.lines || bytes > lim.bytes
			if over {
				if out == nil {
					out = map[int]bool{}
				}
				out[base+at] = true
			}
		}
		if end < 0 {
			break
		}
		off += end + 1
	}
	return out
}
