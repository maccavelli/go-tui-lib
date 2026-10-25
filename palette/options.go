package palette

import (
	"log/slog"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
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
	limit    int
	grouping Grouping
	logger   *slog.Logger
}

func newConfig(opts []Option) config {
	c := config{limit: DefaultLimit, logger: slog.New(slog.DiscardHandler)}
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
