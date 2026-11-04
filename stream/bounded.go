package stream

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// Bounded's defaults, the agent's, with no source
// (docs/decisions/0009-PLAN-streaming-content-engine.md D19).
const (
	defaultBoundedBytes = 64 << 10
	defaultHeadLines    = 20
	defaultTailLines    = 20
	defaultLineBytes    = 4 << 10
)

// boundedFloor is what a Bounded keeps at the least: SGR, and OSC 8 links
// to http, https and mailto targets, on one line.
var boundedFloor = safetext.StatusLine()

// Bounded holds live output, such as a tool's
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §4): all of it
// up to a byte limit, then its first lines and its last, each line capped,
// so that output with no line break stays bounded too; the lines between
// are counted, in lines, which no width changes. Each line is cleaned by
// its rules and then by safetext's StatusLine rules, its floor. It is used
// on one goroutine.
type Bounded struct {
	method                     ansi.Method
	rules                      safetext.Rules
	glyphs                     glyph.Set
	catalog                    msgcat.Catalog
	maxBytes, head, tail, line int

	kept    []string // every line, while within maxBytes; then the head
	last    []string // the last tail lines, once past maxBytes
	size    int      // the bytes of kept, line breaks included
	over    bool     // maxBytes was passed
	omitted int
	partial []byte // the last line as far as it has come, at most line bytes
	cut     bool   // the last line passed line bytes, and the rest of it is dropped
}

// NewBounded is a Bounded with nothing written, by WithMaxBytes,
// WithHeadLines, WithTailLines, WithMaxLineBytes, WithGlyphs, WithCatalog,
// WithRules and WithMethod; the other options it ignores.
func NewBounded(o ...Option) *Bounded {
	b := &Bounded{
		rules: safetext.Text(), glyphs: glyph.Unicode(),
		maxBytes: defaultBoundedBytes, head: defaultHeadLines, tail: defaultTailLines, line: defaultLineBytes,
	}
	for _, opt := range o {
		opt.applyBounded(b)
	}
	return b
}

// Write takes the next of the output; it never fails.
func (b *Bounded) Write(p []byte) (int, error) {
	for rest := p; len(rest) > 0; {
		i := bytes.IndexByte(rest, '\n')
		chunk := rest
		if i >= 0 {
			chunk = rest[:i]
		}
		b.hold(chunk)
		if i < 0 {
			break
		}
		b.add(b.clean())
		b.partial, b.cut = b.partial[:0], false
		rest = rest[i+1:]
	}
	return len(p), nil
}

// hold adds s to the last line, up to its cap, cut at a character's start.
func (b *Bounded) hold(s []byte) {
	if b.cut {
		return
	}
	if room := b.line - len(b.partial); len(s) > room {
		for room > 0 && !utf8.RuneStart(s[room]) {
			room--
		}
		s, b.cut = s[:room], true
	}
	b.partial = append(b.partial, s...)
}

// clean is the last line cleaned, with the glyph set's Ellipsis when it was
// cut.
func (b *Bounded) clean() string {
	s := safetext.Clean(safetext.Clean(string(b.partial), b.rules), boundedFloor)
	if b.cut {
		s += b.glyphs.Ellipsis
	}
	return s
}

// add keeps a complete line: in kept while the bytes are within the limit;
// past it, the head stays in kept, the last lines in last, and the lines
// between are counted.
func (b *Bounded) add(l string) {
	if !b.over {
		b.kept = append(b.kept, l)
		b.size += len(l) + 1
		if b.size <= b.maxBytes {
			return
		}
		b.over = true
		if len(b.kept) > b.head {
			spill := b.kept[b.head:]
			b.kept = b.kept[:b.head:b.head]
			for _, x := range spill {
				b.keepLast(x)
			}
		}
		return
	}
	b.keepLast(l)
}

// keepLast keeps l among the last lines, counting the one it pushes out.
func (b *Bounded) keepLast(l string) {
	b.last = append(b.last, l)
	if len(b.last) > b.tail {
		b.last = b.last[1:]
		b.omitted++
	}
}

// Omitted is how many lines were left out.
func (b *Bounded) Omitted() int { return b.omitted }

// Lines is what is kept: the head; the omitted line, when lines were left
// out, after closing any SGR the head left open; the tail; and the last
// line as far as it has come.
func (b *Bounded) Lines() []string {
	out := append([]string(nil), b.kept...)
	if b.omitted > 0 {
		if n := len(out); n > 0 && strings.Contains(strings.Join(out, ""), "\x1b[") {
			out[n-1] += "\x1b[m"
		}
		def := Messages()[MsgOmitted]
		out = append(out, msgcat.Format(b.catalog, MsgOmitted, def, b.omitted))
	}
	out = append(out, b.last...)
	if len(b.partial) > 0 || b.cut {
		out = append(out, b.clean())
	}
	return out
}

// Render is Lines, each wrapped at width cells by the method, a width of 0
// or less wrapping nothing.
func (b *Bounded) Render(width int) string {
	ls := b.Lines()
	if width > 0 {
		for i, l := range ls {
			ls[i] = b.method.Hardwrap(l, width, true)
		}
	}
	return strings.Join(ls, "\n")
}
