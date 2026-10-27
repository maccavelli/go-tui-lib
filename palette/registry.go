package palette

import (
	"context"
	"encoding/json/jsontext"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/command/commandtea"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// FromRegistry is a provider of reg's commands (0008-MADR §5, A3 items 4
// to 7): those Available on the palette surface, under the when values at
// the time the picker opened. A command matches by its title, its
// "Category: Title", its slash name or an alias, whichever scores best, and
// is drawn as "Category: Title". Its hit carries the command's ID, Danger,
// description and parameters; running it dispatches the command through
// reg, with command.OriginPalette, so the registry's gate and its
// PromptMsg and ResultMsg follow. It owns the prefix ">", and lists every
// such command for an empty query.
func FromRegistry(reg *command.Registry) Provider { return &registry{reg: reg} }

type registry struct {
	reg *command.Registry
	at  atomic.Pointer[when.Context] // the when values at the last opening
}

func (*registry) Name() string   { return "commands" }
func (*registry) Prefix() string { return ">" }

// Opened keeps the when values for Discover, which is not given them.
func (r *registry) Opened(c when.Context) { r.at.Store(&c) }

func (r *registry) Search(ctx context.Context, s Search, yield func(Hit) bool) error {
	for c := range r.reg.Available(s.WhenContext, command.SurfacePalette) {
		if err := ctx.Err(); err != nil {
			return err
		}
		sc, ok := bestOf(s.Matcher, names(c))
		if ok && !yield(r.hit(c, sc, s.WhenContext)) {
			return nil
		}
	}
	return nil
}

// Discover lists every command available where the picker opened, in ID
// order; with no opening seen, none.
func (r *registry) Discover(ctx context.Context, yield func(Hit) bool) error {
	at := r.at.Load()
	if at == nil {
		return nil
	}
	for c := range r.reg.Available(*at, command.SurfacePalette) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !yield(r.hit(c, fuzzy.Score{}, *at)) {
			return nil
		}
	}
	return nil
}

// names is what a command matches by: its title, "Category: Title", its
// slash name and its aliases.
func names(c command.Command) []string {
	out := []string{c.Title}
	if c.Category != "" {
		out = append(out, c.Category+": "+c.Title)
	}
	if c.Slash != "" {
		out = append(out, c.Slash)
	}
	return append(out, c.Aliases...)
}

// bestOf is m's best score over candidates, and whether any matched.
func bestOf(m *fuzzy.Matcher, candidates []string) (fuzzy.Score, bool) {
	var best fuzzy.Score
	found := false
	for _, s := range candidates {
		if sc, ok := m.Score(s); ok && (!found || better(sc, best)) {
			best, found = sc, true
		}
	}
	return best, found
}

// hit is c's hit, whose Run dispatches it through the registry, with the
// when values it was found under.
func (r *registry) hit(c command.Command, sc fuzzy.Score, at when.Context) Hit {
	text := c.Title
	if c.Category != "" {
		text = c.Category + ": " + c.Title
	}
	reg, id := r.reg, c.ID
	h := Hit{
		ID: string(c.ID), Text: text, Score: sc, Help: c.Description,
		Command: c.ID, Danger: c.Danger,
		Run: func(args jsontext.Value) tea.Cmd {
			return commandtea.Dispatch(context.Background(), reg, command.Request{
				ID: id, Args: args, Origin: command.OriginPalette, WhenContext: at,
			})
		},
	}
	// A schema whose parameters cannot be read has no Params and no Run:
	// accepting its hit sends SelectedMsg, and nothing runs (0008-MADR A3
	// item 5).
	params, err := command.Params(c)
	if err != nil {
		h.Run = nil
		return h
	}
	h.Params = params
	return h
}
