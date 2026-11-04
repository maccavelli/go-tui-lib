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
	crlf   bool // lines end with CR LF, by WithCRLF
	noDefs bool // a definition prints nothing, as Plain draws
	fit    bool // a table is fitted to the width, as Plain draws (0009-PLAN D20)

	highlighter Highlighter // WithHighlighter's
	lim         limits      // WithHighlightLimits'
	hl          *highlight  // how code is highlighted, or nil
	fence       fenceState  // the fence the lines are in
	at          int         // the source offset of partial's start
	lineAt      int         // the source offset of the line being printed

	view    *viewBuilder // Plain's view of what it prints, or nil
	emitAt  int          // the source offset of the line an emit draws
	heldAt  int          // the held paragraph line's
	tableAt int          // the held table's first line's

	clean   cleaned             // the sanitizers' output, read after each write
	san     *safetext.Sanitizer // the rules', writing into base
	base    *safetext.Sanitizer // the floor's, writing into clean
	source  strings.Builder     // the cleaned source so far
	partial string              // the incomplete last line

	scan    Scanner  // the block state, line to line
	held    string   // a paragraph's line, until the next line
	holding bool     // held is a line
	table   []string // a table's rows, until it ends
	tableNL bool     // the table's last row has its line break
	defs    map[string]bool

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
	p := &Printer{rules: safetext.Text(), defs: map[string]bool{}, lim: defaultLimits()}
	for _, opt := range o {
		opt.apply(p)
	}
	if p.highlighter != nil {
		p.hl = &highlight{h: p.highlighter, lim: p.lim}
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
	q := &Printer{markup: p.markup, method: p.method, width: width, rules: p.rules, asIs: p.asIs, crlf: p.crlf, defs: map[string]bool{}}
	if p.hl != nil {
		q.hl = &highlight{h: p.hl.h, lim: p.hl.lim}
	}
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
		p.lineAt = p.at
		p.line(p.partial, false)
		p.at += len(p.partial)
		p.partial = ""
	}
	p.endHeld()
	p.endTable()
	p.scan = Scanner{}
	p.fence = fenceState{}
	if v := p.view; v != nil {
		v.closeCode(p.at)
		v.closeIndented()
	}
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
		p.lineAt = p.at
		p.line(s[:i], true)
		p.at += i + 1
		s = s[i+1:]
	}
	p.partial = s
}

func (p *Printer) drain() string {
	s := p.out.String()
	p.out.Reset()
	return s
}

// emit writes one printed line, its rows and its end as WithCRLF says.
// The cleaned text holds no CR, so no line end is written twice.
func (p *Printer) emit(s string, nl bool) {
	if p.view != nil {
		p.view.emitted(s, p.emitAt)
	}
	if p.crlf {
		s = strings.ReplaceAll(s, "\n", "\r\n")
	}
	p.out.WriteString(s)
	switch {
	case nl && p.crlf:
		p.out.WriteString("\r\n")
	case nl:
		p.out.WriteByte('\n')
	}
}

// line takes one complete source line, nl false for a last line with no
// line break, as its Scanner tells its block.
func (p *Printer) line(l string, nl bool) {
	b := p.scan.Scan(l)
	if len(p.table) > 0 && b.Kind != BlockTableRow {
		p.endTable()
	}
	if p.holding && b.Kind != BlockSetext && b.Kind != BlockTable {
		p.endHeld()
	}
	p.emitAt = p.lineAt
	if v := p.view; v != nil && b.Kind != BlockIndented && b.Kind != BlockBlank {
		v.closeIndented()
	}
	switch b.Kind {
	case BlockTableRow:
		p.table, p.tableNL = append(p.table, l), nl
	case BlockSetext:
		p.holding = false
		p.emitAt = p.heldAt
		p.emit(p.heading(p.held), true)
		p.emitAt = p.lineAt
		p.emit(p.markup.Heading.Apply(l), nl)
	case BlockTable:
		p.holding = false
		p.table, p.tableNL, p.tableAt = []string{p.held, l}, nl, p.heldAt
	case BlockFence:
		p.openFence(l, fenceRun(l), p.lineAt)
		if v := p.view; v != nil {
			v.openCode(l, fenceRun(l), p.lineAt)
		}
		p.emit(p.fenceLine(l, fenceRun(l)), nl)
	case BlockCode:
		p.emit(p.code(l, nl), nl)
	case BlockFenceEnd:
		p.fence = fenceState{}
		p.emit(p.fenceLine(l, ""), nl)
		if v := p.view; v != nil {
			v.closeCode(p.lineAt)
		}
	case BlockIndented:
		if v := p.view; v != nil {
			v.indented(p.lineAt, len(l)+b2i(nl))
		}
		p.emit(l, nl)
		if v := p.view; v != nil {
			v.indentR = v.rows()
		}
	case BlockHeading:
		p.emit(p.heading(l), nl)
	case BlockList, BlockQuote, BlockContinuation:
		p.emit(p.prose(l), nl)
	case BlockDefinition:
		p.defs[b.Label] = true
		if !p.noDefs {
			p.emit(p.definitionLine(l), nl)
		}
	case BlockParagraph:
		if !nl {
			p.emit(p.prose(l), false) // the reply's last line: nothing can follow it
			return
		}
		p.held, p.holding, p.heldAt = l, true, p.lineAt
	default: // blank, a break, HTML
		p.emit(l, nl)
	}
}

// heading is a heading's line in the Heading style, its links in the view.
func (p *Printer) heading(l string) string {
	if p.view != nil {
		p.view.links(p.method, l, p.emitAt, l, p.view.rows())
	}
	return p.markup.Heading.Apply(l)
}

// endHeld prints a held paragraph line: nothing came to make it more.
func (p *Printer) endHeld() {
	if p.holding {
		p.holding = false
		p.emitAt = p.heldAt
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
		width := 0 // a line-mode table is never wrapped (0023-MADR's Context)
		if p.fit {
			width = p.width
		}
		rows = FitTable(rows, TableOptions{Width: width, Method: p.method})
	}
	p.emitAt = p.tableAt
	first := 0
	if p.view != nil {
		first = p.view.rows()
	}
	for i, row := range rows {
		p.emit(inline(p.markup, row, p.defs), i < len(rows)-1 || p.tableNL) // the reply's last line keeps having no break
	}
	if v := p.view; v != nil {
		t := ViewTable{Row: first, Rows: v.rows() - first}
		for i, r := range p.table {
			if i != 1 {
				t.Cells = append(t.Cells, cells(r))
			}
		}
		v.v.Tables = append(v.v.Tables, t)
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
	out := wrapProse(p.method, p.width, styled)
	if p.view != nil {
		p.view.links(p.method, l, p.emitAt, out, p.view.rows())
	}
	return out
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
