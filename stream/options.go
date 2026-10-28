package stream

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// Option configures NewPrinter. It is opaque: only this package's
// functions make one.
type Option interface{ apply(*Printer) }

// optionFunc is a function used as an Option.
type optionFunc func(*Printer)

func (f optionFunc) apply(p *Printer) { f(p) }

// WithMarkup styles the reply by m: the zero Markup by default, which
// styles nothing, so a program whose output has no colour, under NO_COLOR
// or an ASCII profile, keeps it; DefaultMarkup is gobble-cli's.
func WithMarkup(m Markup) Option { return optionFunc(func(p *Printer) { p.markup = m }) }

// WithoutTableLayout prints a table's rows as they came, styled by the
// Markup, where by default the Printer lays a table out: its columns as
// wide as their widest cell, aligned as its delimiter row says. A table is
// held until it ends either way. With it and the zero Markup, the output
// is the input, byte for byte
// (docs/decisions/0023-MADR-line-mode-components.md A7).
func WithoutTableLayout() Option { return optionFunc(func(p *Printer) { p.asIs = true }) }

// WithWidth wraps prose at n cells: 0, no wrapping, by default.
func WithWidth(n int) Option { return optionFunc(func(p *Printer) { p.width = n }) }

// WithMethod sets how text is measured: ansi.WcWidth by default, as a
// terminal measures until it reports grapheme clustering (mode 2027), and
// ansi.GraphemeWidth after.
func WithMethod(m ansi.Method) Option { return optionFunc(func(p *Printer) { p.method = m }) }

// WithRules sets how the reply is cleaned: by r, and then by safetext's
// Text rules, the floor, so that no rules clean less than Text does
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D2).
// safetext.Text() by default.
func WithRules(r safetext.Rules) Option { return optionFunc(func(p *Printer) { p.rules = r }) }
