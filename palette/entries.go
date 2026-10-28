package palette

import (
	"context"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/pick"
)

// FromEntries is a provider of a line-mode picker's entries, named name,
// so that one []pick.Entry serves a pick.Chooser in the line mode and the
// palette in the TUI, ranked by one matcher
// (docs/decisions/0023-MADR-line-mode-components.md D5, A3.4 item 8). It is
// Static's form: an empty query lists the entries in their order; a typed
// one matches their Text, as one line of display text, and never their
// Detail. Each hit's ID is the entry's, its Hint the entry's Detail, and it
// has no Run, so accepting it sends SelectedMsg, and the program acts on
// the ID.
//
// The palette ranks one provider's hits as pick.Chooser ranks the same
// entries under its default rules, through its first DefaultLimit hits,
// when their IDs are distinct: it keeps one hit per ID.
func FromEntries(name string, entries []pick.Entry) Provider {
	p := &fromEntries{name: name}
	for _, e := range entries {
		e.Text = sanitize.Line(e.Text)
		p.entries = append(p.entries, e)
	}
	return p
}

type fromEntries struct {
	name    string
	entries []pick.Entry // their Text cleaned, as pick.Chooser ranks it
}

func (p *fromEntries) Name() string { return p.name }

func (p *fromEntries) Search(ctx context.Context, q Search, yield func(Hit) bool) error {
	for _, e := range p.entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if sc, ok := q.Matcher.Score(e.Text); ok && !yield(entryHit(e, sc)) {
			return nil
		}
	}
	return nil
}

// Discover lists every entry, the first best, so they keep their order.
func (p *fromEntries) Discover(ctx context.Context, yield func(Hit) bool) error {
	for i, e := range p.entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !yield(entryHit(e, fuzzy.Score{Value: len(p.entries) - i})) {
			return nil
		}
	}
	return nil
}

func entryHit(e pick.Entry, sc fuzzy.Score) Hit {
	return Hit{ID: e.ID, Text: e.Text, Hint: e.Detail, Score: sc}
}
