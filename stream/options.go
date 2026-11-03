package stream

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// Option configures NewPrinter and NewDoc. It is opaque: only this
// package's functions make one. An option that means nothing to one of
// them is ignored by it, as each option's documentation says
// (docs/decisions/0009-MADR-streaming-content-engine.md A6 item 4).
type Option interface {
	apply(*Printer)
	applyDoc(*Doc)
}

// option is an Option from a function for each of Printer and Doc; a nil
// one ignores the option.
type option struct {
	printer func(*Printer)
	doc     func(*Doc)
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

// printerOnly is an Option a Doc ignores.
func printerOnly(f func(*Printer)) Option { return option{printer: f} }

// docOnly is an Option a Printer ignores.
func docOnly(f func(*Doc)) Option { return option{doc: f} }

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
// segment its Renderer failed on.
func WithMethod(m ansi.Method) Option {
	return option{printer: func(p *Printer) { p.method = m }, doc: func(d *Doc) { d.method = m }}
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
// safetext.Text() by default.
func WithRules(r safetext.Rules) Option {
	return option{printer: func(p *Printer) { p.rules = r }, doc: func(d *Doc) { d.rules = r }}
}

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

// WithoutPartialLine shows only a Doc's complete lines, where by default
// it shows the partial last line too, when no complete line is still
// queued. A Printer ignores it.
func WithoutPartialLine() Option { return docOnly(func(d *Doc) { d.noPartial = true }) }
