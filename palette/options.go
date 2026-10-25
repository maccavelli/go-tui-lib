package palette

import (
	"log/slog"
	"runtime"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "palette"

// DefaultLimit is how many hits a picker keeps, best first, across its
// providers.
const DefaultLimit = 200

// Grouping is how a picker orders its hits.
type Grouping uint8

// The groupings. ByScore is the zero value.
const (
	// ByScore orders every hit by its score, best first.
	ByScore Grouping = iota
	// ByProvider orders the hits by provider, in the order they were
	// registered, and each provider's by score.
	ByProvider
)

var groupingNames = enum.Names[Grouping]{
	Pkg: pkgName, Type: "Grouping",
	Tokens: []string{"score", "provider"},
}

// String is the grouping's token.
func (g Grouping) String() string { return groupingNames.String(g) }

// MarshalText is the grouping's token. A grouping with no token is an
// error.
func (g Grouping) MarshalText() ([]byte, error) { return groupingNames.Marshal(g) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (g *Grouping) UnmarshalText(b []byte) error { return groupingNames.Unmarshal(b, g) }

// Option configures a picker.
type Option interface{ apply(*config) }

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// config is a picker's options, applied.
type config struct {
	limit       int
	grouping    Grouping
	logger      *slog.Logger
	goos        string // the system key hints are labelled for
	placeholder string
	noInput     bool      // a completion pop-up: no input row, non-modal
	recents     *Frecency // where accepted hits are recorded, or nil
}

func newConfig(opts []Option) config {
	c := config{limit: DefaultLimit, logger: slog.New(slog.DiscardHandler), goos: runtime.GOOS}
	for _, o := range opts {
		o.apply(&c)
	}
	return c
}

// WithLimit keeps the best n hits across providers; below 1 keeps
// DefaultLimit.
func WithLimit(n int) Option {
	return optionFunc(func(c *config) {
		if n > 0 {
			c.limit = n
		}
	})
}

// WithGrouping orders the hits by g.
func WithGrouping(g Grouping) Option { return optionFunc(func(c *config) { c.grouping = g }) }

// WithLogger sends a failing provider's error to h. Without it, errors are
// dropped; a failure's row still shows on the screen, naming the provider.
func WithLogger(h slog.Handler) Option {
	return optionFunc(func(c *config) {
		if h != nil {
			c.logger = slog.New(h)
		}
	})
}

// WithGOOS labels the key hints for goos, as runtime.GOOS names a system,
// such as "darwin" for ⌘ and ⌥: for a program serving a remote terminal,
// or a test. Without it, or with "", they are labelled for runtime.GOOS.
func WithGOOS(goos string) Option {
	return optionFunc(func(c *config) {
		if goos != "" {
			c.goos = goos
		}
	})
}

// WithPlaceholder is the input's text while the query is empty.
func WithPlaceholder(s string) Option {
	return optionFunc(func(c *config) { c.placeholder = sanitize.Line(s) })
}

// WithRecents records the ID of each hit the picker accepts in f, which
// Recents reads to put those hits first (0008-MADR §6). Only the ID is
// kept, never a hit's arguments. nil records nothing.
func WithRecents(f *Frecency) Option { return optionFunc(func(c *config) { c.recents = f }) }

// WithoutInput makes the picker a completion pop-up, with no input row of
// its own (0008-MADR §5, A3 item 6). It opens non-modal, so the focused
// pane keeps the keyboard and sends it SetQuery as its "@" or "/" token
// changes, as the overlay "completion" unless WithID names another, under
// the pane's cursor. Its keys are DefaultRules' "overlay:completion"
// bindings; a program that picks another ID binds its own. Accepting a
// hit sends SelectedMsg, runs nothing, and closes the pop-up.
func WithoutInput() Option { return optionFunc(func(c *config) { c.noInput = true }) }

// OverlayOption places a picker's overlay when it opens.
type OverlayOption interface{ applyOverlay(*overlayConfig) }

type overlayFunc func(*overlayConfig)

func (f overlayFunc) applyOverlay(c *overlayConfig) { f(c) }

// The overlay a picker opens in, unless an OverlayOption says otherwise:
// its ID, and its size, chrome included, which the workspace clamps to
// itself; and a completion pop-up's.
const (
	defaultID     = "palette"
	defaultWidth  = 72
	defaultHeight = 16

	completionID     = "completion"
	completionWidth  = 48
	completionHeight = 10
)

// overlayConfig is an opening's OverlayOptions, applied.
type overlayConfig struct {
	id            string
	anchor        workspace.Anchor
	width, height int
}

// newOverlayConfig applies opts over the defaults: a picker's, or a
// completion pop-up's, which sits under the focused pane's cursor.
func newOverlayConfig(opts []OverlayOption, completion bool) overlayConfig {
	c := overlayConfig{id: defaultID, width: defaultWidth, height: defaultHeight}
	if completion {
		c = overlayConfig{
			id: completionID, anchor: workspace.Anchor{Kind: workspace.BelowCursor},
			width: completionWidth, height: completionHeight,
		}
	}
	for _, o := range opts {
		o.applyOverlay(&c)
	}
	return c
}

// WithID names the overlay, for workspace.Close, SendOverlay and its
// "overlay:<id>" key scope; "" keeps the default, "palette", or a
// completion pop-up's "completion".
func WithID(id string) OverlayOption {
	return overlayFunc(func(c *overlayConfig) {
		if id != "" {
			c.id = id
		}
	})
}

// WithAnchor places the overlay at a; without it a picker is centred, and
// a completion pop-up is under the focused pane's cursor.
func WithAnchor(a workspace.Anchor) OverlayOption {
	return overlayFunc(func(c *overlayConfig) { c.anchor = a })
}

// WithSize is the overlay's size in cells, chrome included; a side below 1
// keeps its default, 72 by 16, or 48 by 10 for a completion pop-up. The
// workspace clamps it to its own size.
func WithSize(width, height int) OverlayOption {
	return overlayFunc(func(c *overlayConfig) {
		if width > 0 {
			c.width = width
		}
		if height > 0 {
			c.height = height
		}
	})
}
