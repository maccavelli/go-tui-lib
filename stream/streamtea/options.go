package streamtea

import (
	"github.com/maccavelli/go-tui-lib/v2/frame"
	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// Option configures NewDocPane. It is opaque: only this package's
// functions make one.
type Option interface{ apply(*config) }

type config struct {
	title    string
	noFollow bool
	buf      *stream.Buffer
	pacer    frame.Pacer
}

// optionFunc is a function used as an Option.
type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

func newConfig(o []Option) config {
	var c config
	for _, opt := range o {
		opt.apply(&c)
	}
	if c.pacer == nil {
		c.pacer = frame.Adaptive()
	}
	return c
}

// WithTitle is the pane's title, which the workspace's chrome shows; it is
// empty without it.
func WithTitle(s string) Option {
	return optionFunc(func(c *config) { c.title = s })
}

// WithoutFollow keeps the pane where the user scrolled it: it starts at
// the document's start, and never follows the tail, even at the end.
func WithoutFollow() Option {
	return optionFunc(func(c *config) { c.noFollow = true })
}

// WithBuffer is the Buffer the pane pulls into its Doc on each
// frame.DueMsg. Without it the pane draws the Doc as the program fills it.
func WithBuffer(b *stream.Buffer) Option {
	return optionFunc(func(c *config) { c.buf = b })
}

// WithPacer releases the Buffer's lines a frame at a time; it is a new
// frame.Adaptive by default, and a nil p keeps that. A pacer keeps its
// state between frames, so each pane needs its own.
func WithPacer(p frame.Pacer) Option {
	return optionFunc(func(c *config) {
		if p != nil {
			c.pacer = p
		}
	})
}
