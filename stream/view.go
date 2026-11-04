package stream

import (
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// RenderView is a frame with its side-tables, for copying a code block,
// hit-testing a link and selecting text while a reply streams
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §4;
// docs/decisions/0009-PLAN-streaming-content-engine.md D19).
type RenderView struct {
	// Lines is the frame's rows: Render's, split at its line breaks.
	Lines []string
	// Offsets is each row's source offset: the start of the source line it
	// was drawn from; of its segment, when the Renderer tells none; the
	// blank row between two segments, the next segment's.
	Offsets []int
	// Links is the links' parts, in row order.
	Links []ViewLink
	// Code is the code blocks, in source order.
	Code []ViewCode
	// Tables is the tables, in source order.
	Tables []ViewTable
}

// ViewLink is a link's part on one row: a link wrapped over rows has a part
// a row, all with one ID.
type ViewLink struct {
	// ID is the source offset of the link's "[": the same in every frame,
	// at every width and for any chunking.
	ID int
	// URL is its destination, or its reference's, as the source has it.
	URL string
	// Row is its row, and From and To its cells in the row, To excluded,
	// measured by the Renderer's method.
	Row, From, To int
}

// ViewCode is a code block: its info string and its body's source, for
// "copy code block", and the rows it was drawn on, its fences included.
type ViewCode struct {
	// Info is the fence's info string; "" for indented code.
	Info string
	// Start and End are its body's source bytes, the lines between its
	// fences.
	Start, End int
	// Row is its first row, and Rows how many it was drawn on.
	Row, Rows int
}

// ViewTable is a table's cells as the source has them, its header first,
// for copying, and the rows it was drawn on.
type ViewTable struct {
	Cells     [][]string
	Row, Rows int
}

// ViewRenderer is a Renderer that tells its rows' lines and its
// side-tables, its offsets in src. Plain is one.
type ViewRenderer interface {
	Renderer
	RenderView(src string, width int) (RenderView, error)
}

// viewBuilder is a Printer's view of what it prints: each row's source
// offset, as it emits it, and the side-tables.
type viewBuilder struct {
	v       RenderView
	defURL  map[string]string // the input's reference definitions, by label
	code    int               // the open fence's index in v.Code, or -1
	indent  int               // the open indented block's index in v.Code, or -1
	indentE int               // the open indented block's end so far
	indentR int               // the rows begun when its last line was printed
}

func newViewBuilder(src string) *viewBuilder {
	b := &viewBuilder{defURL: map[string]string{}, code: -1, indent: -1}
	var s Scanner
	for l := range strings.SplitSeq(src, "\n") {
		if blk := s.Scan(l); blk.Kind == BlockDefinition && !strings.HasPrefix(blk.Label, "^") {
			_, n := indentOf(l)
			if end := closingBracket(l[n:], 0) + n; end+2 <= len(l) {
				if f := strings.Fields(l[end+2:]); len(f) > 0 {
					b.defURL[blk.Label] = strings.Trim(f[0], "<>")
				}
			}
		}
	}
	return b
}

// rows is how many rows have begun.
func (b *viewBuilder) rows() int { return len(b.v.Offsets) }

// emitted records the rows s begins, each from the source line at.
func (b *viewBuilder) emitted(s string, at int) {
	for range strings.Count(s, "\n") + 1 {
		b.v.Offsets = append(b.v.Offsets, at)
	}
}

// links records the links of the source line l, at offset at, as wrapped
// is its drawing, beginning at row; cont is a continuation row's prefix.
func (b *viewBuilder) links(m ansi.Method, l string, at int, wrapped string, row int) {
	spans := linkSpans(l, b.defURL)
	if len(spans) == 0 {
		return
	}
	plain := ansi.Strip(l)
	n := prosePrefix(plain)
	cont := strings.Map(func(r rune) rune {
		if r == '>' || r == '\t' {
			return r
		}
		return ' '
	}, plain[:n])
	si := 0
	for k, r := range strings.Split(ansi.Strip(wrapped), "\n") {
		col := 0
		if k > 0 {
			if !strings.HasPrefix(r, cont) {
				return
			}
			r, col = r[len(cont):], m.StringWidth(cont)
			for si < len(plain) && (plain[si] == ' ' || plain[si] == '\t') {
				si++
			}
		}
		if !strings.HasPrefix(plain[si:], r) {
			return // not the line's own bytes: no link is placed
		}
		for _, sp := range spans {
			a, z := max(sp.start, si), min(sp.end, si+len(r))
			if a < z {
				from := col + m.StringWidth(plain[si:a])
				b.v.Links = append(b.v.Links, ViewLink{ID: at + sp.start, URL: sp.url, Row: row + k,
					From: from, To: from + m.StringWidth(plain[a:z])})
			}
		}
		si += len(r)
	}
}

// openCode starts the fence whose opening line l, of run, is at offset at.
func (b *viewBuilder) openCode(l, run string, at int) {
	info := strings.TrimSpace(l[strings.Index(l, run)+len(run):])
	b.v.Code = append(b.v.Code, ViewCode{Info: info, Start: at + len(l) + 1, Row: b.rows()})
	b.code = len(b.v.Code) - 1
}

// closeCode ends the open fence, its body ending at end, its closing line
// printed.
func (b *viewBuilder) closeCode(end int) {
	if b.code < 0 {
		return
	}
	c := &b.v.Code[b.code]
	c.End, c.Rows = max(end, c.Start), b.rows()-c.Row
	b.code = -1
}

// indented adds the indented line of n bytes at at to the open indented
// block, or starts one.
func (b *viewBuilder) indented(at, n int) {
	if b.indent < 0 {
		b.v.Code = append(b.v.Code, ViewCode{Start: at, Row: b.rows()})
		b.indent = len(b.v.Code) - 1
	}
	b.indentE = at + n
}

// b2i is 1 for true, a line's break.
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// closeIndented ends the open indented block after its last line.
func (b *viewBuilder) closeIndented() {
	if b.indent < 0 {
		return
	}
	c := &b.v.Code[b.indent]
	c.End, c.Rows = b.indentE, b.indentR-c.Row
	b.indent = -1
}

// span is a link in a source line: its bytes and its URL.
type span struct {
	start, end int
	url        string
}

// linkSpans is the links of the source line l outside its code spans: an
// inline link or image, and a reference whose definition defs has.
func linkSpans(l string, defs map[string]string) []span {
	var out []span
	for i := 0; i < len(l); {
		switch l[i] {
		case '\\':
			i += 2
			continue
		case '`':
			run := l[i : i+len(l[i:])-len(strings.TrimLeft(l[i:], "`"))]
			if end := closingRun(l[i+len(run):], len(run)); end >= 0 {
				i += len(run) + end + len(run)
				continue
			}
			i += len(run)
			continue
		case '[':
			if sp, ok := linkSpan(l, i, defs); ok {
				out = append(out, sp)
				i = sp.end
				continue
			}
		}
		i++
	}
	return out
}

// linkSpan is the link whose text's bracket is at l[i], as linkAt reads one.
func linkSpan(l string, i int, defs map[string]string) (span, bool) {
	shut := closingBracket(l, i)
	if shut < 0 {
		return span{}, false
	}
	rest := l[shut+1:]
	text := label(l[i+1 : shut])
	switch {
	case strings.HasPrefix(rest, "("):
		if end := closingParen(rest); end > 0 {
			dest := strings.Fields(rest[1:end])
			url := ""
			if len(dest) > 0 {
				url = strings.Trim(dest[0], "<>")
			}
			return span{i, shut + 1 + end + 1, url}, true
		}
	case strings.HasPrefix(rest, "[]"):
		if u, ok := defs[text]; ok {
			return span{i, shut + 3, u}, true
		}
		return span{}, false
	case strings.HasPrefix(rest, "["):
		if end := closingBracket(rest, 0); end > 0 {
			if u, ok := defs[label(rest[1:end])]; ok {
				return span{i, shut + 1 + end + 1, u}, true
			}
		}
	}
	if u, ok := defs[text]; ok && !strings.HasPrefix(rest, "(") && !strings.HasPrefix(rest, "[") {
		return span{i, shut + 1, u}, true
	}
	return span{}, false
}

// RenderView draws src at width cells, as Render does, with its view.
func (r plain) RenderView(src string, width int) (RenderView, error) {
	return r.viewWith(src, width, nil), nil
}

// viewWith is drawWith's drawing of src, and its view.
func (r plain) viewWith(src string, width int, h *highlight) RenderView {
	q := &Printer{method: r.method, width: width, noDefs: true, fit: true, defs: map[string]bool{}, hl: h}
	q.view = newViewBuilder(src)
	q.source.WriteString(src)
	q.end()
	out := q.drain()
	v := q.view.v
	if out == "" {
		return RenderView{}
	}
	v.Lines = strings.Split(out, "\n")
	for len(v.Offsets) < len(v.Lines) {
		v.Offsets = append(v.Offsets, len(src))
	}
	v.Offsets = v.Offsets[:len(v.Lines)]
	return v
}

// viewCache is a width's views of the committed segments, joined, of the
// definitions as they were at gen.
type viewCache struct {
	gen  uint64
	outs int
	v    RenderView
}

// View is Render's frame with its side-tables
// (docs/decisions/0009-PLAN-streaming-content-engine.md D19). A segment's
// view is kept with its render; the tail's is taken anew each frame, so
// nothing of an earlier tail is carried; a code block a relaxed cut split
// is one ViewCode; after Finish, it is the Renderer's view of the whole
// source. With a Renderer that is not a ViewRenderer, a row's offset is
// its segment's, and there is no link, code block or table.
func (d *Doc) View(width int) RenderView {
	src := d.source.String()
	if d.finished {
		return d.viewOf(src, width, 0, false, 0, len(src))
	}
	c := d.viewCache(width)
	for ; c.outs < len(d.segs); c.outs++ {
		sg := d.segs[c.outs]
		in := d.input(sg.open, src[sg.start:sg.end], sg.close, true)
		joinView(&c.v, trimView(d.viewOf(in, width, inputBase(sg.start, sg.open), false, sg.start, sg.end)), sg.start, sg.open != "")
	}
	out := copyView(c.v)
	if width != d.codeWidth {
		d.code.reset()
		d.codeWidth = width
	}
	base := inputBase(d.tailStart, d.tailOpen)
	if t := d.tail(src); t != "" {
		joinView(&out, trimView(d.viewOf(t, width, base, true, d.tailStart, len(src))), d.tailStart, d.tailOpen != "")
	}
	return out
}

// viewCache is width's view cache, emptied when a definition has been
// committed since it was filled; the views of the two widths last used are
// kept, as their renders are.
func (d *Doc) viewCache(width int) *viewCache {
	c, ok := d.views[width]
	if !ok {
		if len(d.viewWidths) == 2 {
			delete(d.views, d.viewWidths[0])
			d.viewWidths = d.viewWidths[1:]
		}
		c = &viewCache{gen: d.defGen}
		d.views[width] = c
		d.viewWidths = append(d.viewWidths, width)
	}
	if c.gen != d.defGen {
		*c = viewCache{gen: d.defGen}
	}
	return c
}

// viewOf is src's view at width, its offsets made the document's by base
// and kept within start and end, where src is the input of the source
// between them: Plain's, a ViewRenderer's, or the Renderer's rows alone,
// each at start; and Plain's when the Renderer fails.
func (d *Doc) viewOf(src string, width, base int, keep bool, start, end int) RenderView {
	var v RenderView
	p, isPlain := d.r.(plain)
	vr, isView := d.r.(ViewRenderer)
	switch {
	case isPlain:
		var h *highlight
		if d.highlighter != nil {
			h = &highlight{h: d.highlighter, lim: d.lim, plain: overLimits(src, base, d.lim), base: base}
			if keep {
				h.cache = &d.code
			}
		}
		v = p.viewWith(src, width, h)
	case isView:
		var err error
		if v, err = vr.RenderView(src, width); err != nil {
			v = plain{d.method}.viewWith(src, width, nil)
		}
	default:
		if out := d.draw(src, width, base, keep); out != "" {
			v.Lines = strings.Split(out, "\n")
			v.Offsets = make([]int, len(v.Lines))
			for i := range v.Offsets {
				v.Offsets[i] = start
			}
		}
		return v
	}
	clamp := func(o int) int { return min(max(base+o, start), end) }
	for i := range v.Offsets {
		v.Offsets[i] = clamp(v.Offsets[i])
	}
	for i := range v.Links {
		v.Links[i].ID += base
	}
	for i := range v.Code {
		v.Code[i].Start, v.Code[i].End = clamp(v.Code[i].Start), clamp(v.Code[i].End)
	}
	return v
}

// trimView is v without its blank rows at its start and end, as trimBlank
// trims a render.
func trimView(v RenderView) RenderView {
	empty := func(l string) bool { return strings.TrimSpace(ansi.Strip(l)) == "" }
	i, j := 0, len(v.Lines)
	for i < j && empty(v.Lines[i]) {
		i++
	}
	for j > i && empty(v.Lines[j-1]) {
		j--
	}
	out := RenderView{Lines: v.Lines[i:j], Offsets: v.Offsets[i:j]}
	for _, l := range v.Links {
		if l.Row >= i && l.Row < j {
			l.Row -= i
			out.Links = append(out.Links, l)
		}
	}
	for _, c := range v.Code {
		c.Row, c.Rows = c.Row-i, min(c.Row+c.Rows, j)-c.Row
		if c.Row < 0 {
			c.Rows, c.Row = c.Rows+c.Row, 0
		}
		out.Code = append(out.Code, c)
	}
	for _, t := range v.Tables {
		t.Row -= i
		out.Tables = append(out.Tables, t)
	}
	return out
}

// joinView adds part to dst after one blank row, whose offset is start,
// part's source; inside is true when part begins inside a fence dst's
// last code block is, which the two then share.
func joinView(dst *RenderView, part RenderView, start int, inside bool) {
	if len(part.Lines) == 0 {
		return
	}
	if len(dst.Lines) > 0 {
		dst.Lines = append(dst.Lines, "")
		dst.Offsets = append(dst.Offsets, start)
	}
	shift := len(dst.Lines)
	dst.Lines = append(dst.Lines, part.Lines...)
	dst.Offsets = append(dst.Offsets, part.Offsets...)
	for _, l := range part.Links {
		l.Row += shift
		dst.Links = append(dst.Links, l)
	}
	for k, c := range part.Code {
		c.Row += shift
		if n := len(dst.Code); k == 0 && inside && n > 0 && c.Start == dst.Code[n-1].End {
			last := &dst.Code[n-1]
			last.End, last.Rows = c.End, c.Row+c.Rows-last.Row
			continue
		}
		dst.Code = append(dst.Code, c)
	}
	for _, t := range part.Tables {
		t.Row += shift
		dst.Tables = append(dst.Tables, t)
	}
}

// copyView is v with slices of its own, so that joining to it leaves v as
// it was.
func copyView(v RenderView) RenderView {
	return RenderView{
		Lines: slices.Clone(v.Lines), Offsets: slices.Clone(v.Offsets), Links: slices.Clone(v.Links),
		Code: slices.Clone(v.Code), Tables: slices.Clone(v.Tables),
	}
}
