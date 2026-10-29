package stream

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// Printer prints a streamed reply's Markdown line by line, each line once,
// for a scrollback that cannot be redrawn
// (docs/decisions/0023-MADR-line-mode-components.md D7, A3, A6). Write
// takes the next chunk and gives what is now stable; Flush gives the rest
// at the reply's end. Each source line is printed once it is complete and
// what it is can no longer change: a paragraph's line once the next line
// has come, since an underline after it makes it a setext heading and a
// delimiter row a table's header (A3 item 13, Q10); a table once a blank
// line or another block ends it; a fence's lines as each completes. A
// printed line is cleaned, styled by the Markup, every marker kept, and a
// prose line wrapped at whole words to the width; a line is never printed
// or styled again. The output is the same for any split of the input. A
// Printer does no I/O and is used on one goroutine.
type Printer struct {
	markup Markup
	method ansi.Method
	width  int
	rules  safetext.Rules
	asIs   bool // a table's rows printed as they came, by WithoutTableLayout

	clean   cleaned             // the sanitizers' output, read after each write
	san     *safetext.Sanitizer // the rules', writing into base
	base    *safetext.Sanitizer // the floor's, writing into clean
	source  strings.Builder     // the cleaned source so far
	partial string              // the incomplete last line

	held      string   // a paragraph's line, until the next line
	holding   bool     // held is a line
	table     []string // a table's rows, until it ends
	tableNL   bool     // the table's last row has its line break
	fence     string   // the open fence's marker run, or ""
	html      int      // the open HTML block's kind, 1 to 7, or 0
	paragraph bool     // the last line was a paragraph's
	lazy      bool     // the last line was a list item's or a quote's text, which a paragraph's line continues
	defs      map[string]bool

	out strings.Builder
}

// cleaned is where the sanitizers write: the floor's sanitizer writes
// here, and the rules' sanitizer into the floor's. Its Write never fails,
// as a strings.Builder's does not.
type cleaned struct{ b strings.Builder }

func (c *cleaned) Write(p []byte) (int, error) { return c.b.Write(p) }

// floor is what a reply keeps at the least: safetext's Text rules, printable
// text, line breaks, tabs and the bidirectional controls prose needs.
var floor = safetext.Text()

// NewPrinter is a Printer with nothing printed.
func NewPrinter(o ...Option) *Printer {
	p := &Printer{rules: safetext.Text(), defs: map[string]bool{}}
	for _, opt := range o {
		opt.apply(p)
	}
	p.open()
	return p
}

// open starts the sanitizers afresh.
func (p *Printer) open() {
	p.base = safetext.NewSanitizer(&p.clean, floor)
	p.san = safetext.NewSanitizer(p.base, p.rules)
}

// SetWidth wraps the lines printed from now on at width cells; 0 or less
// wraps nothing. A line printed already stays as it was.
func (p *Printer) SetWidth(width int) { p.width = width }

// Source is the cleaned source written so far, across Flushes: what a
// transcript keeps to print the reply again at another width.
func (p *Printer) Source() string { return p.source.String() }

// At is the Source drawn afresh at width cells: given to a new Printer with
// this one's options at that width, with no sanitizer, since it is cleaned
// already, then ended as Flush ends a reply. It changes nothing in this
// Printer. While this one is Idle, At gives what it would have printed so
// far had its width been width all along; a transcript draws a reply again
// so on a resize (docs/decisions/0023-MADR-line-mode-components.md A8 item
// 2, A9 item 2).
func (p *Printer) At(width int) string {
	q := &Printer{markup: p.markup, method: p.method, width: width, rules: p.rules, asIs: p.asIs, defs: map[string]bool{}}
	q.clean.b.WriteString(p.Source()) // cleaned already, by the rules and then the floor
	q.end()
	return q.drain()
}

// Idle reports whether the Printer holds nothing back: what it printed
// ends a line, and no line, table or part of one waits.
func (p *Printer) Idle() bool {
	return p.partial == "" && !p.holding && len(p.table) == 0 && p.clean.b.Len() == 0
}

// Write takes the next chunk of the reply, and gives what is now stable,
// as whole lines.
func (p *Printer) Write(chunk string) string {
	p.san.Write([]byte(chunk)) //nolint:errcheck // it fails only as cleaned's Write does, which never fails
	p.take()
	return p.drain()
}

// Flush ends the reply: it gives everything still held, the last line
// with no line break when the reply had none. A Write after it starts
// afresh, at a line's start with no block open; the source and the
// definitions printed are kept.
func (p *Printer) Flush() string {
	p.san.Close()  //nolint:errcheck // as Write: cleaned's Write never fails
	p.base.Close() //nolint:errcheck // as Write: cleaned's Write never fails
	p.end()
	p.open()
	return p.drain()
}

// end prints what the sanitizers wrote and everything still held, and
// closes every block.
func (p *Printer) end() {
	p.take()
	if p.partial != "" {
		p.line(p.partial, false)
		p.partial = ""
	}
	p.endHeld()
	p.endTable()
	p.fence, p.html, p.paragraph, p.lazy = "", 0, false, false
}

// take reads what the sanitizers wrote, and prints each line it
// completes.
func (p *Printer) take() {
	s := p.clean.b.String()
	p.clean.b.Reset()
	p.source.WriteString(s)
	s = p.partial + s
	for {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			break
		}
		p.line(s[:i], true)
		s = s[i+1:]
	}
	p.partial = s
}

func (p *Printer) drain() string {
	s := p.out.String()
	p.out.Reset()
	return s
}

// emit writes one printed line.
func (p *Printer) emit(s string, nl bool) {
	p.out.WriteString(s)
	if nl {
		p.out.WriteByte('\n')
	}
}

// line takes one complete source line, nl false for a last line with no
// line break.
func (p *Printer) line(l string, nl bool) {
	if p.fence != "" {
		if closesFence(l, p.fence) {
			p.fence = ""
			p.emit(p.fenceLine(l, ""), nl)
			return
		}
		p.emit(l, nl)
		return
	}
	if p.html > 0 {
		end := htmlEnds(p.html, l)
		if end && p.html >= 6 {
			p.html = 0 // the blank line is the block's end, and is printed as one
		} else {
			if end {
				p.html = 0
			}
			p.emit(l, nl)
			return
		}
	}
	if len(p.table) > 0 {
		if !blank(l) && !startsBlock(l) {
			p.table, p.tableNL = append(p.table, l), nl
			return
		}
		p.endTable()
	}
	if p.holding {
		h := p.held
		p.holding = false
		switch {
		case setextUnderline(l):
			p.emit(p.markup.Heading.Apply(h), true)
			p.emit(p.markup.Heading.Apply(l), nl)
			p.paragraph = false
			return
		case tableStart(h, l):
			p.table, p.tableNL = []string{h, l}, nl
			p.paragraph = false
			return
		}
		p.emit(p.prose(h), true)
	}
	p.block(l, nl)
}

// block takes a line with nothing held before it.
func (p *Printer) block(l string, nl bool) {
	cont, lazy := p.paragraph, p.lazy
	p.paragraph, p.lazy = false, false
	if lazy && !blank(l) && !startsBlock(l) && !indentedCode(l) {
		p.lazy = true // a lazy continuation, never a setext heading's text (CommonMark §4.3)
		p.emit(p.prose(l), nl)
		return
	}
	switch {
	case blank(l):
		p.emit(l, nl)
	case fenceRun(l) != "":
		p.fence = fenceRun(l)
		p.emit(p.fenceLine(l, p.fence), nl)
	case indentedCode(l) && !cont:
		p.emit(l, nl)
	case atxHeading(l):
		p.emit(p.markup.Heading.Apply(l), nl)
	case thematicBreak(l):
		p.emit(l, nl)
	case htmlStart(l, cont) > 0:
		k := htmlStart(l, cont)
		if !htmlEnds(k, l) || k >= 6 {
			p.html = k
		}
		p.emit(l, nl)
	case container(l):
		p.lazy = !blank(strings.TrimLeft(containerText(l), ">"))
		p.emit(p.prose(l), nl)
	default:
		if lbl, ok := definition(l); ok && !cont {
			p.defs[lbl] = true
			p.emit(p.definitionLine(l), nl)
			return
		}
		p.paragraph = true
		if !nl {
			p.emit(p.prose(l), false) // the reply's last line: nothing can follow it
			return
		}
		p.held, p.holding = l, true
	}
}

// endHeld prints a held paragraph line: nothing came to make it more.
func (p *Printer) endHeld() {
	if p.holding {
		p.holding = false
		p.emit(p.prose(p.held), true)
	}
}

// endTable lays out and prints a held table, or prints its rows as they
// came under WithoutTableLayout.
func (p *Printer) endTable() {
	if len(p.table) == 0 {
		return
	}
	rows := p.table
	if !p.asIs {
		rows = layoutTable(p.method, rows)
	}
	for i, row := range rows {
		p.emit(inline(p.markup, row, p.defs), i < len(rows)-1 || p.tableNL) // the reply's last line keeps having no break
	}
	p.table = nil
}

// prose is a paragraph's, a list item's or a quote's line, its marker
// styled, its text styled, wrapped at the width.
func (p *Printer) prose(l string) string {
	cols, n := indentOf(l)
	body := l[n:]
	var styled string
	switch k := markerLen(body); {
	case cols <= 3 && k > 0:
		styled = l[:n] + p.markup.Marker.Apply(body[:k]) + inline(p.markup, body[k:], p.defs)
	default:
		styled = inline(p.markup, l, p.defs)
	}
	return wrapProse(p.method, p.width, styled)
}

// fenceLine is a fence's opening line, its run dim and its language in
// the FenceInfo style, or its closing line, dim.
func (p *Printer) fenceLine(l, run string) string {
	if run == "" {
		return p.markup.Fence.Apply(l)
	}
	i := strings.Index(l, run) + len(run)
	marker, info := l[:i], l[i:]
	lang := strings.TrimSpace(info)
	if lang == "" {
		return p.markup.Fence.Apply(l)
	}
	j := strings.Index(info, lang)
	return p.markup.Fence.Apply(marker+info[:j]) + p.markup.FenceInfo.Apply(lang) + info[j+len(lang):]
}

// definitionLine is a reference definition's line: its label in the Link
// style, the rest in the URL style.
func (p *Printer) definitionLine(l string) string {
	_, n := indentOf(l)
	end := closingBracket(l[n:], 0) + n
	return l[:n] + p.markup.Link.Apply(l[n:end+1]) + ":" + p.markup.URL.Apply(l[end+2:])
}

// containerText is a list item's or a quote's line after its markers.
func containerText(l string) string {
	return l[prosePrefix(l):]
}
