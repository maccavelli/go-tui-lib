package launch

import (
	"io"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
)

// Option configures Run.
type Option interface {
	apply(*options)
}

type options struct {
	registry  *command.Registry
	restorers []Restorer
	onStart   []func(*tea.Program)
	program   []tea.ProgramOption
	filter    func(tea.Model, tea.Msg) tea.Msg // on the program's own model
	epilogue  io.Writer                        // WithEpilogueWriter's; nil is Out
	noEpi     bool                             // WithoutEpilogue
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) { f(o) }

// WithRegistry attaches r to the program's loop while the TUI runs, so a
// Loop command an agent or a CLI handler runs reaches the model on its own
// loop. Run detaches it on every outcome before it returns; a Loop command
// started after that runs on its caller, exactly once
// (docs/decisions/0013-MADR-cli-integration-helpers.md A1.1).
func WithRegistry(r *command.Registry) Option {
	return optionFunc(func(o *options) { o.registry = r })
}

// WithRestorer adds r, whose bytes Run writes to the stream the TUI drew on,
// on every outcome. Several may be given; each is written in turn.
func WithRestorer(r Restorer) Option {
	return optionFunc(func(o *options) { o.restorers = append(o.restorers, r) })
}

// OnStart adds f, which Run calls with the program once it exists and
// before it runs, so a goroutine of the program's own can Send to it. Each
// is called once, in the order given.
func OnStart(f func(*tea.Program)) Option {
	return optionFunc(func(o *options) { o.onStart = append(o.onStart, f) })
}

// WithProgramOptions adds Bubble Tea options, after Run's own, so they can
// add to them or replace them. A filter goes through WithFilter instead:
// tea.WithFilter would see Run's wrapper, not the program's model.
func WithProgramOptions(opts ...tea.ProgramOption) Option {
	return optionFunc(func(o *options) { o.program = append(o.program, opts...) })
}

// WithoutEpilogue stops Run writing the final model's Epilogue(), for a
// program that writes it itself, from the model Run returns
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D4). It wins
// over WithEpilogueWriter.
func WithoutEpilogue() Option { return optionFunc(func(o *options) { o.noEpi = true }) }

// WithEpilogueWriter has Run write the epilogue to w instead of Out. A nil
// w keeps Out.
func WithEpilogueWriter(w io.Writer) Option {
	return optionFunc(func(o *options) { o.epilogue = w })
}

// WithFilter is tea.WithFilter, given the program's own model. A model that
// is not an M passes every message through unchanged.
func WithFilter[M tea.Model](f func(M, tea.Msg) tea.Msg) Option {
	return optionFunc(func(o *options) {
		o.filter = func(m tea.Model, msg tea.Msg) tea.Msg {
			if mm, ok := m.(M); ok {
				return f(mm, msg)
			}
			return msg
		}
	})
}
