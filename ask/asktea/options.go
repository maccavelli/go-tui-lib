package asktea

import (
	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// Option configures NewDialog. It is opaque: only this package's functions
// make one.
type Option interface{ apply(*config) }

type config struct {
	id     string
	anchor workspace.Anchor
	width  int
	ask    []ask.Option
}

// optionFunc is a function used as an Option.
type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

func newConfig(o []Option) config {
	c := config{id: DefaultID}
	for _, opt := range o {
		opt.apply(&c)
	}
	return c
}

// WithID names the overlay, DefaultID by default; its AnsweredMsg carries
// the name.
func WithID(id string) Option {
	return optionFunc(func(c *config) {
		if id != "" {
			c.id = id
		}
	})
}

// WithAnchor places the overlay, centred by default.
func WithAnchor(a workspace.Anchor) Option { return optionFunc(func(c *config) { c.anchor = a }) }

// WithWidth sets the overlay's width, its border included: the
// workspace's by default. Zero or less keeps the default.
func WithWidth(cells int) Option { return optionFunc(func(c *config) { c.width = cells }) }

// WithAsk gives the Asker its options, after the workspace's width method
// and glyphs: a reason's bound, the program's catalogue, the rules text is
// cleaned by.
func WithAsk(o ...ask.Option) Option {
	return optionFunc(func(c *config) { c.ask = append(c.ask, o...) })
}
