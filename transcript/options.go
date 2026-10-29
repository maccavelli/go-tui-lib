package transcript

import (
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// The defaults, gobble-cli's (its 0030-MADR D3 and D4).
const (
	// DefaultMaxBytes is the most the Transcript keeps, in bytes.
	DefaultMaxBytes = 4 << 20
	// DefaultMaxRows is the most rows a reprint draws.
	DefaultMaxRows = 10000
	// DefaultSettle is how long after the last resize the reprint is due.
	DefaultSettle = 150 * time.Millisecond
)

// Option configures NewTranscript. It is opaque: only this package's
// functions make one.
type Option interface{ apply(*Transcript) }

// optionFunc is a function used as an Option.
type optionFunc func(*Transcript)

func (f optionFunc) apply(t *Transcript) { f(t) }

// WithMaxBytes keeps at most n bytes of pieces; past that, the oldest go.
// Zero or less keeps DefaultMaxBytes.
func WithMaxBytes(n int) Option {
	return optionFunc(func(t *Transcript) {
		if n > 0 {
			t.maxBytes = n
		}
	})
}

// WithMaxRows draws at most n rows in a reprint, the newest. Zero or less
// keeps DefaultMaxRows.
func WithMaxRows(n int) Option {
	return optionFunc(func(t *Transcript) {
		if n > 0 {
			t.maxRows = n
		}
	})
}

// WithSettle makes the reprint due d after the last resize. Zero or less
// keeps DefaultSettle.
func WithSettle(d time.Duration) Option {
	return optionFunc(func(t *Transcript) {
		if d > 0 {
			t.settle = d
		}
	})
}

// WithMethod sets how a row's cells are counted against WithMaxRows:
// ansi.WcWidth by default, as a terminal measures until it reports grapheme
// clustering (mode 2027), and ansi.GraphemeWidth after.
func WithMethod(m ansi.Method) Option { return optionFunc(func(t *Transcript) { t.method = m }) }

// WithoutScrollbackClear leaves CSI 3 J out of the reprint: the terminal's
// scrollback is kept, and with it the rows shown before the reprint, at
// their old width (docs/decisions/0023-MADR-line-mode-components.md D13).
func WithoutScrollbackClear() Option {
	return optionFunc(func(t *Transcript) { t.keepScrollback = true })
}

// WithSynchronized sets whether the reprint is one synchronized update
// (mode 2026). It is on by default; a terminal without mode 2026 ignores it.
func WithSynchronized(on bool) Option { return optionFunc(func(t *Transcript) { t.sync = on }) }

// WithRules sets how AddText's text and the notice are cleaned: by r, and
// then by safetext's Text rules, the floor, so that no rules clean less
// than Text does (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md
// D2). safetext.Text() by default. AddDrawn's text is the host's own
// drawing, and is written as drawn; a reply is cleaned by its Printer.
func WithRules(r safetext.Rules) Option { return optionFunc(func(t *Transcript) { t.rules = r }) }

// WithCatalog gives the Transcript's own words in the program's: see
// Messages.
func WithCatalog(c msgcat.Catalog) Option { return optionFunc(func(t *Transcript) { t.catalog = c }) }

// WithNoticeStyle draws the notice line in s: unstyled by default, as
// stream's zero Markup is (docs/decisions/0023-MADR-line-mode-components.md
// A8 item 3). gobble dims it.
func WithNoticeStyle(s stream.Style) Option { return optionFunc(func(t *Transcript) { t.notice = s }) }
