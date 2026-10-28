package lineedittea

import "github.com/maccavelli/go-tui-lib/v2/lineedit"

// Option configures NewComposerPane. It is opaque: only this package's
// functions make one.
type Option interface{ apply(*config) }

type config struct {
	id       string
	composer []lineedit.Option
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

// WithID names the pane in its EventMsgs, DefaultID by default.
func WithID(id string) Option {
	return optionFunc(func(c *config) {
		if id != "" {
			c.id = id
		}
	})
}

// WithComposer gives the Composer its options, after the workspace's width
// method and glyphs: its prompt, history, completer and keymap.
func WithComposer(o ...lineedit.Option) Option {
	return optionFunc(func(c *config) { c.composer = append(c.composer, o...) })
}
