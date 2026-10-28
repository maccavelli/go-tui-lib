package ask

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// DefaultReasonLimit bounds a reason, in bytes, as gobble-cli bounds its
// own (internal/cli/permission.go's maxWhy).
const DefaultReasonLimit = 2048

// Option configures NewAsker. It is opaque: only this package's functions
// make one.
type Option interface{ apply(*Asker) }

// optionFunc is a function used as an Option.
type optionFunc func(*Asker)

func (f optionFunc) apply(a *Asker) { f(a) }

// WithMethod sets how the question is measured: ansi.WcWidth by default,
// as a terminal measures until it reports grapheme clustering (mode 2027),
// and ansi.GraphemeWidth after. It is the region's too, when the Asker
// builds its own.
func WithMethod(m ansi.Method) Option { return optionFunc(func(a *Asker) { a.method = m }) }

// WithGlyphs sets the glyphs the question is drawn with: glyph.Unicode()
// by default, glyph.ASCII() where the terminal cannot show Unicode.
func WithGlyphs(g glyph.Set) Option { return optionFunc(func(a *Asker) { a.glyphs = g }) }

// WithRegion draws the question in r, the host's region, which it shares
// among its components and sets inline's options on; one engine draws a
// region at a time. Without it, or with nil, the Asker builds its own, by
// inline.NewRegion and WithMethod's method
// (docs/decisions/0023-MADR-line-mode-components.md A2 item 3).
func WithRegion(r *inline.Region) Option { return optionFunc(func(a *Asker) { a.region = r }) }

// WithReasonLimit bounds a reason, in bytes. Zero or less keeps
// DefaultReasonLimit.
func WithReasonLimit(n int) Option {
	return optionFunc(func(a *Asker) {
		if n > 0 {
			a.limit = n
		}
	})
}

// WithKeymap resolves a reason's keys in k, the program's keymap, in
// lineedit.ComposerScope, so a user's file rebinds them; k holds
// lineedit.DefaultRules among its defaults. An answer's own key is its
// Key, not the keymap's. Without it, or with nil, the Asker builds a
// keymap of lineedit.DefaultRules alone
// (docs/decisions/0023-MADR-line-mode-components.md A5 item 4).
func WithKeymap(k *keymap.Keymap) Option { return optionFunc(func(a *Asker) { a.keys = k }) }

// WithCatalog gives the Asker's own words in the program's: see Messages.
func WithCatalog(c msgcat.Catalog) Option { return optionFunc(func(a *Asker) { a.catalog = c }) }

// WithRules sets how the question's text and a typed reason are cleaned:
// by r, and then by safetext's Line rules, the floor, so that no rules
// clean less than Line does (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md
// D2). safetext.Line() by default.
func WithRules(r safetext.Rules) Option { return optionFunc(func(a *Asker) { a.rules = r }) }
