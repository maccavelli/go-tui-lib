package stream

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// Option configures NewPrinter, NewDoc and NewBounded. It is opaque: only
// this package's functions make one. An option that means nothing to one
// of them is ignored by it, as each option's documentation says
// (docs/decisions/0009-MADR-streaming-content-engine.md A6 item 4;
// docs/decisions/0009-PLAN-streaming-content-engine.md D19).
type Option interface {
	apply(*Printer)
	applyDoc(*Doc)
	applyBounded(*Bounded)
}

// option is an Option from a function for each of Printer, Doc and
// Bounded; a nil one ignores the option.
type option struct {
	printer func(*Printer)
	doc     func(*Doc)
	bounded func(*Bounded)
}

func (o option) apply(p *Printer) {
	if o.printer != nil {
		o.printer(p)
	}
}

func (o option) applyDoc(d *Doc) {
	if o.doc != nil {
		o.doc(d)
	}
}

func (o option) applyBounded(b *Bounded) {
	if o.bounded != nil {
		o.bounded(b)
	}
}

// printerOnly is an Option a Doc and a Bounded ignore.
func printerOnly(f func(*Printer)) Option { return option{printer: f} }

// docOnly is an Option a Printer and a Bounded ignore.
func docOnly(f func(*Doc)) Option { return option{doc: f} }

// boundedOnly is an Option a Printer and a Doc ignore.
func boundedOnly(f func(*Bounded)) Option { return option{bounded: f} }

// size is the option that sets *at to n, when n is above zero.
func size(n int, at func(*Bounded) *int) Option {
	return boundedOnly(func(b *Bounded) {
		if n > 0 {
			*at(b) = n
		}
	})
}

// WithMarkup styles the reply by m: the zero Markup by default, which
// styles nothing, so a program whose output has no colour, under NO_COLOR
// or an ASCII profile, keeps it; DefaultMarkup is gobble-cli's. A Doc,
// whose Renderer styles it, ignores it.
func WithMarkup(m Markup) Option { return printerOnly(func(p *Printer) { p.markup = m }) }

// WithoutTableLayout prints a table's rows as they came, styled by the
// Markup, where by default the Printer lays a table out: its columns as
// wide as their widest cell, aligned as its delimiter row says. A table is
// held until it ends either way. With it and the zero Markup, the output
// is the input, byte for byte
// (docs/decisions/0023-MADR-line-mode-components.md A7). A Doc ignores it.
func WithoutTableLayout() Option { return printerOnly(func(p *Printer) { p.asIs = true }) }

// WithWidth wraps prose at n cells: 0, no wrapping, by default. A Doc,
// given its width at each Render, ignores it.
func WithWidth(n int) Option { return printerOnly(func(p *Printer) { p.width = n }) }

// WithMethod sets how text is measured: ansi.WcWidth by default, as a
// terminal measures until it reports grapheme clustering (mode 2027), and
// ansi.GraphemeWidth after. A Doc measures by it what it wraps itself, a
// segment its Renderer failed on; a Bounded, the rows it wraps.
func WithMethod(m ansi.Method) Option {
	return option{
		printer: func(p *Printer) { p.method = m }, doc: func(d *Doc) { d.method = m },
		bounded: func(b *Bounded) { b.method = m },
	}
}

// WithCRLF ends each printed line, and each row of a wrapped one, with CR
// LF in place of LF, for a terminal in raw mode written without an
// inline.Region; a Writer's output too
// (docs/decisions/0009-MADR-streaming-content-engine.md A8). A Doc
// ignores it.
func WithCRLF() Option { return printerOnly(func(p *Printer) { p.crlf = true }) }

// WithRules sets how the reply is cleaned: by r, and then by safetext's
// Text rules, the floor, so that no rules clean less than Text does
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D2).
// safetext.Text() by default. A Bounded cleans each line by r and then by
// safetext's StatusLine rules, its floor, which keep SGR and safe links:
// WithRules(safetext.StatusLine()) keeps a tool's colours.
func WithRules(r safetext.Rules) Option {
	return option{
		printer: func(p *Printer) { p.rules = r }, doc: func(d *Doc) { d.rules = r },
		bounded: func(b *Bounded) { b.rules = r },
	}
}

// WithHighlighter highlights fenced code by h, its language the info
// string's first word (docs/decisions/0009-MADR-streaming-content-engine.md
// A1 §4): a Printer each code line as it prints it, from the state its
// last left, a block past a limit plain from that line on, since a line
// is printed once; a Doc whose Renderer is Plain, nil or stream.Plain, its
// blocks, a block past a limit plain whole. In a Doc, a committed
// segment's blocks are highlighted when it is rendered, once a width; the
// tail's lines are kept, each with the state after it, so each line is
// highlighted once, up to twice the limit's bytes, past which all are
// dropped; a width change and a new SetRevision drop them. A Doc with
// another Renderer ignores it, as does a Bounded. No code is highlighted
// by default.
func WithHighlighter(h Highlighter) Option {
	return option{printer: func(p *Printer) { p.highlighter = h }, doc: func(d *Doc) { d.highlighter = h }}
}

// WithHighlightLimits is when code is too much to highlight, and shows
// plain: past maxBytes, maxLines, or a line of maxLineBytes; 512 KiB,
// 10,000 lines and 4 KiB by default, codex's, and zero or less keeps each.
// A Bounded ignores it.
func WithHighlightLimits(maxBytes, maxLines, maxLineBytes int) Option {
	set := func(l *limits) {
		for _, v := range []struct {
			n  int
			at *int
		}{{maxBytes, &l.bytes}, {maxLines, &l.lines}, {maxLineBytes, &l.line}} {
			if v.n > 0 {
				*v.at = v.n
			}
		}
	}
	return option{printer: func(p *Printer) { set(&p.lim) }, doc: func(d *Doc) { set(&d.lim) }}
}

// WithMaxBytes is the bytes a Bounded keeps all of: 64 KiB by default,
// and zero or less keeps it. Past it, it keeps its head and tail lines. A
// Printer and a Doc ignore it.
func WithMaxBytes(n int) Option { return size(n, func(b *Bounded) *int { return &b.maxBytes }) }

// WithHeadLines is the first lines a Bounded keeps past its bytes: 20 by
// default, and zero or less keeps it. A Printer and a Doc ignore it.
func WithHeadLines(n int) Option { return size(n, func(b *Bounded) *int { return &b.head }) }

// WithTailLines is the last lines a Bounded keeps past its bytes: 20 by
// default, and zero or less keeps it. A Printer and a Doc ignore it.
func WithTailLines(n int) Option { return size(n, func(b *Bounded) *int { return &b.tail }) }

// WithMaxLineBytes is the bytes a Bounded keeps of one line, the rest of
// it dropped and the line ended with the Ellipsis: 4 KiB by default, and
// zero or less keeps it. A Printer and a Doc ignore it.
func WithMaxLineBytes(n int) Option { return size(n, func(b *Bounded) *int { return &b.line }) }

// WithGlyphs sets the glyphs a Bounded draws with, its Ellipsis:
// glyph.Unicode() by default. A Printer and a Doc ignore it.
func WithGlyphs(g glyph.Set) Option { return boundedOnly(func(b *Bounded) { b.glyphs = g }) }

// WithCatalog gives a Bounded's words, MsgOmitted, in the program's: see
// Messages for the defaults. A Printer and a Doc ignore it.
func WithCatalog(c msgcat.Catalog) Option { return boundedOnly(func(b *Bounded) { b.catalog = c }) }

// WithRelaxAfter is how many bytes a Doc's tail may hold with no cut
// before the next line where a block can be split is cut: 2 KiB by
// default, and zero or less keeps it
// (docs/decisions/0009-MADR-streaming-content-engine.md §4). A Printer
// ignores it.
func WithRelaxAfter(n int) Option {
	return docOnly(func(d *Doc) {
		if n > 0 {
			d.relaxAfter = n
		}
	})
}

// WithoutInlineHoldback shows a Doc's partial last line whole, where by
// default it stops before an inline construct the line opens and has not
// closed: emphasis, a strikethrough, a code span or a link. A Printer
// ignores it.
func WithoutInlineHoldback() Option { return docOnly(func(d *Doc) { d.noHoldback = true }) }

// WithMath makes a Doc's holdback hold an unclosed \( or $$ in the partial
// line too, as it holds the other constructs, until it closes, the line
// completes or Finish: off by default
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §4). A
// Printer, which prints whole lines only, ignores it.
func WithMath() Option { return docOnly(func(d *Doc) { d.math = true }) }

// WithoutPartialLine shows only a Doc's complete lines, where by default
// it shows the partial last line too, when no complete line is still
// queued. A Printer ignores it.
func WithoutPartialLine() Option { return docOnly(func(d *Doc) { d.noPartial = true }) }
