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

// OverlayOption places a picker's overlay when it opens.
type OverlayOption interface{ applyOverlay(*overlayConfig) }

type overlayFunc func(*overlayConfig)

func (f overlayFunc) applyOverlay(c *overlayConfig) { f(c) }

// The overlay a picker opens in, unless an OverlayOption says otherwise:
// its ID, and its size, chrome included, which the workspace clamps to
// itself.
const (
	defaultID     = "palette"
	defaultWidth  = 72
	defaultHeight = 16
)

// overlayConfig is an opening's OverlayOptions, applied.
type overlayConfig struct {
	id            string
	anchor        workspace.Anchor
	width, height int
}

func newOverlayConfig(opts []OverlayOption) overlayConfig {
	c := overlayConfig{id: defaultID, width: defaultWidth, height: defaultHeight}
	for _, o := range opts {
		o.applyOverlay(&c)
	}
	return c
}

// WithID names the overlay, for workspace.Close, SendOverlay and its
// "overlay:<id>" key scope; "" keeps the default, "palette".
func WithID(id string) OverlayOption {
	return overlayFunc(func(c *overlayConfig) {
		if id != "" {
			c.id = id
		}
	})
}

// WithAnchor places the overlay at a; it is centred without it.
func WithAnchor(a workspace.Anchor) OverlayOption {
	return overlayFunc(func(c *overlayConfig) { c.anchor = a })
}

// WithSize is the overlay's size in cells, chrome included; a side below 1
// keeps its default, 72 by 16. The workspace clamps it to its own size.
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
