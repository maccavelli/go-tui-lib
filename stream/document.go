package stream

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// defaultRelaxAfter is how many bytes a Doc's tail holds with no cut before
// a relaxed one, crush's (docs/decisions/0009-MADR-streaming-content-engine.md
// §4).
const defaultRelaxAfter = 2 << 10

// segment is a committed part of a Doc's source, rendered once a width: its
// bytes, the fence's opening line when it begins inside a fence a relaxed
// cut split, and the fence's closing run when it ends inside one.
type segment struct {
	start, end  int
	open, close string
}

// widthCache is a width's renders of the committed segments, trimmed, of
// the definitions as they were at gen, and those not empty joined, so that
// a frame copies the prefix once.
type widthCache struct {
	gen    uint64
	outs   int
	joined strings.Builder
}

// Doc is one streamed Markdown document, drawn by a Renderer a stable
// prefix at a time (docs/decisions/0009-MADR-streaming-content-engine.md
// §4). Its source is cleaned as it comes, by a Sanitizer of its rules and
// then of safetext's Text rules, so nothing in it, its caches or its frames
// holds a sequence the stream sent. Each complete line is read once by a
// Scanner, and a blank line where nothing stays open is a cut: the source
// before it is a segment, rendered once a width and cached, and only the
// tail after the last cut is rendered each frame. A tail that has gone
// without a cut for RelaxAfter bytes is cut at the next line where its
// block can be split. A frame is the segments' renders and the tail's,
// each trimmed of its blank lines at its start and end, joined by one
// blank line; after Finish, a frame is the Renderer's of the whole source,
// byte for byte. A Doc is used on one goroutine; a Buffer is the way in
// from others.
type Doc struct {
	r                     Renderer
	method                ansi.Method
	rules                 safetext.Rules
	relaxAfter            int
	noHoldback, noPartial bool
	math                  bool        // the holdback holds math too, by WithMath
	highlighter           Highlighter // WithHighlighter's, used when the Renderer is Plain
	lim                   limits      // WithHighlightLimits'

	clean     cleaned
	san, base *safetext.Sanitizer
	source    strings.Builder
	scan      Scanner
	done      int      // the source's bytes in complete lines, each scanned
	tailStart int      // where the tail begins
	tailOpen  string   // the fence's opening line the tail begins inside, after a relaxed cut
	since     int      // the tail's bytes in complete lines
	tailDefs  []string // the definition lines in the tail
	segs      []segment
	defs      []string // the definition lines committed, which each input carries (D16)
	defGen    uint64   // raised when a definition is committed
	widths    []int    // the widths cached, the latest last
	cache     map[int]*widthCache
	final     map[int]string
	version   uint64
	finished  bool

	frame       string
	frameW      int
	frameV      uint64
	framed      bool
	finalWidths []int

	code      codeCache // the tail's highlighted lines
	codeWidth int       // the width they were drawn at
	rev       uint64    // SetRevision's

	views      map[int]*viewCache // the committed segments' views, by width
	viewWidths []int              // the widths whose views are kept, the latest last
}

// NewDoc is a Doc that renders with r, Plain by the Doc's method when r is
// nil.
func NewDoc(r Renderer, o ...Option) *Doc {
	d := &Doc{rules: safetext.Text(), relaxAfter: defaultRelaxAfter, lim: defaultLimits()}
	for _, opt := range o {
		opt.applyDoc(d)
	}
	d.r = r
	if r == nil {
		d.r = Plain(d.method)
	}
	d.Reset()
	return d
}

// Append adds s, a chunk of the document, cleaned. After Finish it does
// nothing.
func (d *Doc) Append(s string) {
	if d.finished || s == "" {
		return
	}
	d.san.Write([]byte(s)) //nolint:errcheck // it fails only as cleaned's Write does, which never fails
	d.take()
}

// Write is Append, as an io.Writer. After Finish it returns 0 and
// ErrClosed.
func (d *Doc) Write(p []byte) (int, error) {
	if d.finished {
		return 0, ErrClosed
	}
	d.Append(string(p))
	return len(p), nil
}

// Finish ends the document: what the sanitizers hold is cleaned and
// added, and from then on a frame is the Renderer's of the whole source.
func (d *Doc) Finish() {
	if d.finished {
		return
	}
	d.san.Close()  //nolint:errcheck // as Append: cleaned's Write never fails
	d.base.Close() //nolint:errcheck // as Append: cleaned's Write never fails
	d.take()
	d.finished = true
	d.version++
}

// Reset empties the Doc, as a new one with its Renderer and options.
func (d *Doc) Reset() {
	d.clean.b.Reset()
	d.base = safetext.NewSanitizer(&d.clean, floor)
	d.san = safetext.NewSanitizer(d.base, d.rules)
	d.source.Reset()
	d.scan = Scanner{}
	d.done, d.tailStart, d.tailOpen, d.since = 0, 0, "", 0
	d.tailDefs, d.segs, d.defs = nil, nil, nil
	d.defGen++
	d.widths, d.cache = nil, map[int]*widthCache{}
	d.final, d.finalWidths = map[int]string{}, nil
	d.finished, d.framed = false, false
	d.code.reset()
	d.views, d.viewWidths = map[int]*viewCache{}, nil
	d.version++
}

// SetRevision is the revision of what the Renderer and the Highlighter draw
// with, such as a theme.Theme's ID, which stream does not import
// (docs/decisions/0009-PLAN-streaming-content-engine.md D19): a different
// one drops every cache, the segments', the frame's and the highlighted
// lines, so the next frame is drawn anew, and raises Version.
func (d *Doc) SetRevision(rev uint64) {
	if rev == d.rev {
		return
	}
	d.rev = rev
	d.widths, d.cache = nil, map[int]*widthCache{}
	d.final, d.finalWidths = map[int]string{}, nil
	d.framed = false
	d.code.reset()
	d.views, d.viewWidths = map[int]*viewCache{}, nil
	d.version++
}

// Version is raised by every change to what Render gives.
func (d *Doc) Version() uint64 { return d.version }

// Finished reports whether Finish has ended the document.
func (d *Doc) Finished() bool { return d.finished }

// Source is the cleaned source so far.
func (d *Doc) Source() string { return d.source.String() }

// take reads what the sanitizers wrote, and scans each line it completes.
func (d *Doc) take() {
	s := d.clean.b.String()
	if s == "" {
		return
	}
	d.clean.b.Reset()
	d.source.WriteString(s)
	d.version++
	src := d.source.String()
	for {
		i := strings.IndexByte(src[d.done:], '\n')
		if i < 0 {
			return
		}
		d.line(d.done, src[d.done:d.done+i])
		d.done += i + 1
	}
}

// line scans the complete line l, which begins at start, and cuts before it
// where the Scanner says, or where the tail has gone RelaxAfter bytes with
// no cut and its block can be split.
func (d *Doc) line(start int, l string) {
	b := d.scan.Scan(l)
	switch {
	case b.Cut:
		d.commit(start, "", "")
	case b.Split && d.since > d.relaxAfter && b.Kind == BlockCode:
		d.commit(start, b.Fence, fenceRun(b.Fence))
	case b.Split && d.since > d.relaxAfter:
		d.commit(start, "", "")
	}
	d.since += len(l) + 1
	if b.Kind == BlockDefinition {
		d.tailDefs = append(d.tailDefs, l)
	}
}

// commit makes the tail before at a segment, closed by closer; the tail
// after it begins inside the fence open opens.
func (d *Doc) commit(at int, open, closer string) {
	if at > d.tailStart {
		d.segs = append(d.segs, segment{start: d.tailStart, end: at, open: d.tailOpen, close: closer})
	}
	if len(d.tailDefs) > 0 {
		d.defs = append(d.defs, d.tailDefs...)
		d.tailDefs = nil
		d.defGen++
	}
	d.tailStart, d.tailOpen, d.since = at, open, 0
}

// Render is the frame at width cells.
func (d *Doc) Render(width int) string {
	if d.finished {
		return d.whole(width)
	}
	if d.framed && d.frameW == width && d.frameV == d.version {
		return d.frame
	}
	c := d.widthCache(width)
	src := d.source.String()
	for ; c.outs < len(d.segs); c.outs++ {
		sg := d.segs[c.outs]
		join(&c.joined, trimBlank(d.draw(d.input(sg.open, src[sg.start:sg.end], sg.close, true), width, inputBase(sg.start, sg.open), false)))
	}
	if width != d.codeWidth {
		d.code.reset()
		d.codeWidth = width
	}
	base := inputBase(d.tailStart, d.tailOpen)
	d.code.keep(base)
	var tail string
	if t := d.tail(src); t != "" {
		tail = trimBlank(d.draw(t, width, base, true))
	}
	prefix := c.joined.String()
	switch {
	case tail == "":
		d.frame = prefix
	case prefix == "":
		d.frame = tail
	default:
		d.frame = prefix + "\n\n" + tail
	}
	d.frameW, d.frameV, d.framed = width, d.version, true
	return d.frame
}

// join appends part to b after one blank line, and nothing for an empty
// part.
func join(b *strings.Builder, part string) {
	if part == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString(part)
}

// whole is the finished document's frame at width: the Renderer's of the
// whole source, cached for the last two widths.
func (d *Doc) whole(width int) string {
	if s, ok := d.final[width]; ok {
		return s
	}
	if len(d.finalWidths) == 2 {
		delete(d.final, d.finalWidths[0])
		d.finalWidths = d.finalWidths[1:]
	}
	s := d.draw(d.source.String(), width, 0, false)
	d.final[width] = s
	d.finalWidths = append(d.finalWidths, width)
	return s
}

// widthCache is width's cache, emptied when a definition has been
// committed since it was filled; the caches of the two widths last used
// are kept, so that a width changed and back renders no segment again.
func (d *Doc) widthCache(width int) *widthCache {
	c, ok := d.cache[width]
	if !ok {
		if len(d.widths) == 2 {
			delete(d.cache, d.widths[0])
			d.widths = d.widths[1:]
		}
		c = &widthCache{gen: d.defGen}
		d.cache[width] = c
		d.widths = append(d.widths, width)
	} else if i := indexOf(d.widths, width); i >= 0 && i != len(d.widths)-1 {
		d.widths = append(append(d.widths[:i:i], d.widths[i+1:]...), width)
	}
	if c.gen != d.defGen {
		c.gen, c.outs = d.defGen, 0
		c.joined.Reset()
	}
	return c
}

func indexOf(s []int, v int) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// tail is the tail's render input: its complete lines, and the partial
// last line, shown as WithoutPartialLine and the holdback say. A Buffer
// gives the partial line only once no complete line is queued, so it never
// shows out of order.
func (d *Doc) tail(src string) string {
	body := src[d.tailStart:d.done]
	if partial := src[d.done:]; partial != "" && !d.noPartial {
		if !d.noHoldback && d.scan.fence == "" {
			partial = holdback(partial, d.math)
		}
		body += partial
	}
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return d.input(d.tailOpen, body, "", d.scan.fence == "" && d.scan.html == 0)
}

// input is a segment's render input: the fence it opens inside, its body,
// the fence it closes, and, where withDefs, the definitions committed so
// far after a blank line (docs/decisions/0009-PLAN-streaming-content-engine.md
// D16).
func (d *Doc) input(open, body, closer string, withDefs bool) string {
	var b strings.Builder
	if open != "" {
		b.WriteString(open)
		b.WriteByte('\n')
	}
	b.WriteString(body)
	if closer != "" {
		if !strings.HasSuffix(body, "\n") {
			b.WriteByte('\n')
		}
		b.WriteString(closer)
		b.WriteByte('\n')
	}
	if withDefs && len(d.defs) > 0 {
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
		for _, def := range d.defs {
			b.WriteString(def)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// inputBase is the document offset of a render input's start, for the
// input of a segment or a tail that begins at start, inside the fence open
// opens when it is not "": its opening line, given again, is before start.
func inputBase(start int, open string) int {
	if open == "" {
		return start
	}
	return start - len(open) - 1
}

// draw is the Renderer's render of src at width, or, when it fails, src
// drawn by Plain: no text is lost. With a Highlighter and Plain, src's
// fences are highlighted, a block past a limit plain whole; base is src's
// offset in the document, and keep keeps the lines highlighted, as the
// tail's are.
func (d *Doc) draw(src string, width, base int, keep bool) string {
	if p, ok := d.r.(plain); ok && d.highlighter != nil {
		h := &highlight{h: d.highlighter, lim: d.lim, plain: overLimits(src, base, d.lim), base: base}
		if keep {
			h.cache = &d.code
		}
		return p.drawWith(src, width, h)
	}
	out, err := d.r.Render(src, width)
	if err != nil {
		out, _ = Plain(d.method).Render(src, width) //nolint:errcheck // Plain never fails
	}
	return out
}

// trimBlank is s without its blank lines at its start and end, a line blank
// when nothing but spaces is left of it with its sequences stripped.
func trimBlank(s string) string {
	lines := strings.Split(s, "\n")
	empty := func(l string) bool { return strings.TrimSpace(ansi.Strip(l)) == "" }
	i, j := 0, len(lines)
	for i < j && empty(lines[i]) {
		i++
	}
	for j > i && empty(lines[j-1]) {
		j--
	}
	return strings.Join(lines[i:j], "\n")
}
