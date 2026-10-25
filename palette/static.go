package palette

import (
	"context"
	"encoding/json/jsontext"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
)

// Item is one entry of a Static provider: a theme, a layout preset, a
// model or a mode.
type Item struct {
	ID   string // stable; keys recents and de-duplication
	Text string // the line shown, and what the query matches
	Help string // the selected item's description line
	Hint string // a key hint
	// Run is what accepting the item does, on the event loop, or nil.
	Run func() tea.Cmd
}

// Static is a provider of a small fixed set of items, named name
// (0008-MADR §5). An empty query lists them all, in their order; a typed
// one matches their Text.
func Static(name string, items []Item) Provider {
	return &static{name: name, items: slices.Clone(items)}
}

type static struct {
	name  string
	items []Item
}

func (s *static) Name() string { return s.name }

func (s *static) Search(ctx context.Context, q Search, yield func(Hit) bool) error {
	for _, it := range s.items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if sc, ok := q.Matcher.Score(it.Text); ok && !yield(it.hit(sc)) {
			return nil
		}
	}
	return nil
}

// Discover lists every item, the first best, so they keep their order.
func (s *static) Discover(ctx context.Context, yield func(Hit) bool) error {
	for i, it := range s.items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !yield(it.hit(fuzzy.Score{Value: len(s.items) - i})) {
			return nil
		}
	}
	return nil
}

func (it Item) hit(sc fuzzy.Score) Hit {
	h := Hit{ID: it.ID, Text: it.Text, Help: it.Help, Hint: it.Hint, Score: sc}
	if run := it.Run; run != nil {
		h.Run = func(jsontext.Value) tea.Cmd { return run() }
	}
	return h
}
