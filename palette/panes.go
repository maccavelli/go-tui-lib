package palette

import (
	"context"
	"encoding/json/jsontext"
	"slices"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/when"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// Panes is a provider of w's panes (0008-MADR §5): each pane of the focus
// ring, which accepting focuses, and each pane the layout hides, which
// accepting shows and focuses. A pane is drawn by its title, as a
// workspace.Titled gives it, or by its ID. It reads w when the picker opens,
// on the event loop, and lists them all for an empty query.
func Panes(w *workspace.Workspace) Provider { return &panes{w: w} }

type panes struct {
	w  *workspace.Workspace
	at atomic.Pointer[[]paneEntry] // the panes at the last opening
}

// paneEntry is one pane as the picker opened.
type paneEntry struct {
	id     layout.PaneID
	title  string
	hidden bool
}

func (*panes) Name() string { return "panes" }

// opened reads the workspace's panes, which only the event loop may.
func (p *panes) opened(when.Context) {
	var out []paneEntry
	hidden := p.w.Plan().Hidden
	for id, pane := range p.w.Panes() {
		if !slices.Contains(hidden, id) {
			out = append(out, paneEntry{id: id, title: titleOf(id, pane)})
		}
	}
	for _, id := range hidden {
		if pane, ok := p.w.Pane(id); ok {
			out = append(out, paneEntry{id: id, title: titleOf(id, pane), hidden: true})
		}
	}
	p.at.Store(&out)
}

func titleOf(id layout.PaneID, p workspace.Pane) string {
	if t, ok := p.(workspace.Titled); ok && t.Title() != "" {
		return t.Title()
	}
	return string(id)
}

func (p *panes) entries() []paneEntry {
	if at := p.at.Load(); at != nil {
		return *at
	}
	return nil
}

func (p *panes) Search(ctx context.Context, q Search, yield func(Hit) bool) error {
	for _, e := range p.entries() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if sc, ok := q.Matcher.Score(e.title); ok && !yield(p.hit(e, sc)) {
			return nil
		}
	}
	return nil
}

// Discover lists every pane, the ring's in its order, then the hidden.
func (p *panes) Discover(ctx context.Context, yield func(Hit) bool) error {
	es := p.entries()
	for i, e := range es {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !yield(p.hit(e, fuzzy.Score{Value: len(es) - i})) {
			return nil
		}
	}
	return nil
}

// hit is e's hit: accepting it focuses the pane, showing it first if it
// is hidden. Its Run calls the workspace, on the event loop.
func (p *panes) hit(e paneEntry, sc fuzzy.Score) Hit {
	w, id := p.w, e.id
	h := Hit{ID: "pane:" + string(id), Text: e.title, Score: sc, Help: "Focus the pane"}
	h.Run = func(jsontext.Value) tea.Cmd { return w.Focus(id) }
	if e.hidden {
		h.Help = "Show the hidden pane, and focus it"
		h.Run = func(jsontext.Value) tea.Cmd { return tea.Batch(w.Toggle(id), w.Focus(id)) }
	}
	return h
}
