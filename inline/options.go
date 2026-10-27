package inline

import (
	"time"

	"github.com/charmbracelet/x/ansi"
)

// DefaultReportWait is how long a resize waits for the cursor report
// before it redraws by WithReflow's kind (gobble-cli's
// 0030-MADR-resize-and-performance.md D2, as amended).
const DefaultReportWait = 150 * time.Millisecond

// Option configures NewRegion. It is opaque: only this package's functions
// make one.
type Option interface{ apply(*Region) }

// optionFunc is a function used as an Option.
type optionFunc func(*Region)

func (f optionFunc) apply(r *Region) { f(r) }

// WithMethod sets how the region measures text: ansi.WcWidth by default,
// as a terminal measures until it reports grapheme clustering (mode 2027),
// and ansi.GraphemeWidth after.
func WithMethod(m ansi.Method) Option { return optionFunc(func(r *Region) { r.method = m }) }

// WithSynchronized sets whether each drawing is one synchronized update
// (mode 2026) with the cursor hidden while it moves. It is on by default;
// a terminal without mode 2026 ignores it.
func WithSynchronized(on bool) Option { return optionFunc(func(r *Region) { r.sync = on }) }

// WithReflow sets how the terminal re-wraps its rows on a resize, for when
// the cursor report does not say: ReflowUnknown, redrawn as a re-wrap, by
// default. A host sets it from what it knows, such as termcap's identity of
// xterm, which cuts.
func WithReflow(k Reflow) Option { return optionFunc(func(r *Region) { r.reflow = k }) }

// WithoutCursorQuery stops a resize asking the terminal where its cursor
// is (CSI 6 n): the region redraws at once, by WithReflow's kind alone
// (docs/decisions/0023-MADR-line-mode-components.md D13).
func WithoutCursorQuery() Option { return optionFunc(func(r *Region) { r.noQuery = true }) }

// WithReportWait sets how long a resize waits for the cursor report. Zero
// or less keeps DefaultReportWait
// (docs/decisions/0023-MADR-line-mode-components.md A1).
func WithReportWait(d time.Duration) Option {
	return optionFunc(func(r *Region) {
		if d > 0 {
			r.wait = d
		}
	})
}

// WithHeight sets the most rows the region shows: a taller region is drawn
// from the rows around the cursor's, so that row is on the screen. Zero or
// less shows every row, the default. SetHeight changes it later.
func WithHeight(rows int) Option { return optionFunc(func(r *Region) { r.SetHeight(rows) }) }
