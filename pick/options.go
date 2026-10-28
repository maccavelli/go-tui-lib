package pick

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// The defaults.
const (
	// DefaultHeight is the most rows of entries shown, gum's and huh's
	// (docs/decisions/0023-MADR-line-mode-components.md A3.5, Q8).
	DefaultHeight = 10
	// DefaultIndexAbove is how many entries a Chooser ranks in full; above
	// it a fuzzy.Index narrows them first, as palette.DefaultIndexAbove
	// does for a walk's paths.
	DefaultIndexAbove = 5000
)

// Option configures NewChooser. It is opaque: only this package's functions
// make one.
type Option interface{ apply(*Chooser) }

// optionFunc is a function used as an Option.
type optionFunc func(*Chooser)

func (f optionFunc) apply(c *Chooser) { f(c) }

// WithHeight sets the most rows of entries shown. Zero or less keeps
// DefaultHeight. The list is that tall, or as tall as the entries when
// there are fewer, for the whole pick.
func WithHeight(n int) Option {
	return optionFunc(func(c *Chooser) {
		if n > 0 {
			c.height = n
		}
	})
}

// WithPrompt sets the prompt before the query, and before the line left in
// the scrollback: the glyphs' Prompt by default. An empty prompt keeps it.
func WithPrompt(s string) Option { return optionFunc(func(c *Chooser) { c.prompt = s }) }

// WithQuery sets the query the pick starts with, its cursor at its end.
func WithQuery(s string) Option { return optionFunc(func(c *Chooser) { c.initial = s }) }

// WithMethod sets how the rows are measured: ansi.WcWidth by default, as a
// terminal measures until it reports grapheme clustering (mode 2027), and
// ansi.GraphemeWidth after. It is the region's too, when the Chooser builds
// its own.
func WithMethod(m ansi.Method) Option { return optionFunc(func(c *Chooser) { c.method = m }) }

// WithGlyphs sets the glyphs the rows are drawn with: glyph.Unicode() by
// default, glyph.ASCII() where the terminal cannot show Unicode.
func WithGlyphs(g glyph.Set) Option { return optionFunc(func(c *Chooser) { c.glyphs = g }) }

// WithIndexAbove narrows a list of more than n entries through a
// fuzzy.Index before it is ranked. Zero or less keeps DefaultIndexAbove.
func WithIndexAbove(n int) Option {
	return optionFunc(func(c *Chooser) {
		if n > 0 {
			c.indexAbove = n
		}
	})
}

// WithRegion draws the pick in r, the host's region, which it shares among
// its components and sets inline's options on; one engine draws a region
// at a time. Without it, or with nil, the Chooser builds its own, by
// inline.NewRegion and WithMethod's method
// (docs/decisions/0023-MADR-line-mode-components.md A2 item 3).
func WithRegion(r *inline.Region) Option { return optionFunc(func(c *Chooser) { c.region = r }) }

// WithCatalog gives the Chooser's own words in the program's: see Messages.
func WithCatalog(cat msgcat.Catalog) Option {
	return optionFunc(func(c *Chooser) { c.catalog = cat })
}

// WithRules sets how the entries' text, the prompt and a typed query are
// cleaned: by r, and then by safetext's Line rules, the floor, so that no
// rules clean less than Line does
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D2).
// safetext.Line() by default.
func WithRules(r safetext.Rules) Option { return optionFunc(func(c *Chooser) { c.rules = r }) }

// WithKeymap resolves the Chooser's keys in k, the program's keymap, in
// ChooserScope and then lineedit.ComposerScope, whose editing commands
// edit the query, so a user's file rebinds them; k holds DefaultRules and
// lineedit.DefaultRules among its defaults. Without it, or with nil, the
// Chooser builds a keymap of the two alone.
func WithKeymap(k *keymap.Keymap) Option { return optionFunc(func(c *Chooser) { c.keys = k }) }
