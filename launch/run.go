package launch

import (
	"context"
	"errors"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
)

// newProgram is the program constructor seam. Tests replace it.
var newProgram = tea.NewProgram

// Run starts the TUI on the decision's targets, and returns the final model
// as an M (docs/decisions/0013-MADR-cli-integration-helpers.md §7, A1.1 to
// A1.4). The program passes its own ctx, typically from
// signal.NotifyContext: launch installs no signal handler.
//
// What it returns:
//   - the model quitting: the final model, and nil;
//   - a plain decision, a controlling terminal that does not open, or
//     Bubble Tea failing before the model's Init: m, and an error wrapping
//     ErrNotStarted. Nothing was drawn by the model, and the terminal is
//     as it was;
//   - tea.ErrInterrupted, or ctx ending: the final model, or the last good
//     one when Bubble Tea returned none, and Bubble Tea's error as it is;
//   - anything else after Init, such as a panic Bubble Tea recovered: the
//     last good model, the model after the last Update that returned, and
//     an error wrapping ErrCrashed. The terminal was restored.
//
// On every outcome, after Bubble Tea returns, Run detaches WithRegistry's
// registry, writes each Restorer's bytes to the stream the TUI drew on,
// puts back the terminal states it saved when the TUI did not start or
// crashed, and closes the controlling terminal it opened. On a clean end, a
// final model with an Epilogue() string method has the epilogue written to
// Out, after the TUI's last byte. A failure in any of these is joined onto
// the error Run returns.
//
// The program is built with the decision's input and output,
// WithEnvironment(s.Env), WithColorProfile(d.UIProfile),
// WithoutSignalHandler, and WithWindowSize(d.Width, d.Height) when the
// drawing stream is not a terminal file; WithProgramOptions's options
// follow. Run never sets the alternate screen: that is the model's choice,
// in its View.
func Run[M tea.Model](ctx context.Context, s Streams, d Decision, m M, opts ...Option) (M, error) {
	return runWith(ctx, s, d, m, openTTY, opts)
}

// runWith is Run, with open as the controlling terminal.
func runWith[M tea.Model](ctx context.Context, s Streams, d Decision, m M, open opener, opts []Option) (M, error) {
	if !d.Interactive {
		return m, fmt.Errorf("%w: %s", ErrNotStarted, d.Reason)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var o options
	for _, f := range opts {
		f.apply(&o)
	}

	t, err := openTargets(s, d, open)
	if err != nil {
		return m, fmt.Errorf("%w: %w", ErrNotStarted, err)
	}
	restores, err := saveAll(t.in, t.out)
	if err != nil {
		return m, errors.Join(fmt.Errorf("%w: saving the terminal's state: %w", ErrNotStarted, err), t.close())
	}

	st := &state[M]{last: m}
	popts := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithInput(t.in),
		tea.WithOutput(t.out),
		tea.WithEnvironment(s.Env),
		tea.WithColorProfile(d.UIProfile),
		tea.WithoutSignalHandler(),
	}
	if !terminalFile(t.out) {
		popts = append(popts, tea.WithWindowSize(d.Width, d.Height))
	}
	if o.filter != nil {
		popts = append(popts, tea.WithFilter(func(pm tea.Model, msg tea.Msg) tea.Msg {
			if w, ok := pm.(wrapped[M]); ok {
				return o.filter(w.inner, msg)
			}
			return o.filter(pm, msg)
		}))
	}
	popts = append(popts, o.program...)
	p := newProgram(wrapped[M]{inner: m, st: st}, popts...)

	var detach func()
	if o.registry != nil {
		detach = o.registry.Attach(p.Send)
	}
	for _, f := range o.onStart {
		f(p)
	}
	final, runErr := p.Run()

	// On every outcome, in this order: detach, the restorers, the saved
	// states when the TUI did not start or crashed, the opened terminal,
	// and the epilogue on a clean end.
	if detach != nil {
		detach()
	}
	var extra []error
	for _, r := range o.restorers {
		if b := r.Restore(); b != "" {
			if _, err := io.WriteString(t.out, b); err != nil {
				extra = append(extra, fmt.Errorf("launch: writing a restorer's reset: %w", err))
			}
		}
	}
	model, end, err := classify(ctx, m, st, final, runErr)
	if end == endNotStarted || end == endCrashed {
		for _, restore := range restores {
			if err := restore(); err != nil {
				extra = append(extra, fmt.Errorf("launch: restoring the terminal's state: %w", err))
			}
		}
	}
	if err := t.close(); err != nil {
		extra = append(extra, err)
	}
	if end == endClean {
		if e, ok := any(model).(interface{ Epilogue() string }); ok && s.Out != nil {
			if _, err := io.WriteString(s.Out, e.Epilogue()); err != nil {
				extra = append(extra, fmt.Errorf("launch: writing the epilogue: %w", err))
			}
		}
	}
	if len(extra) == 0 {
		return model, err
	}
	return model, errors.Join(append([]error{err}, extra...)...)
}

// end is how a run ended.
type end uint8

const (
	endClean      end = iota // the model quit
	endNotStarted            // Bubble Tea failed before Init
	endEnded                 // an interrupt, or ctx ended
	endCrashed               // anything else, after Init
)

// classify maps Bubble Tea's result to Run's.
func classify[M tea.Model](ctx context.Context, m M, st *state[M], final tea.Model, runErr error) (M, end, error) {
	switch {
	case runErr == nil:
		got, ok := unwrap[M](final)
		if !ok {
			return st.last, endClean, fmt.Errorf("launch: the final model is a %T, not a %T", final, m)
		}
		return got, endClean, nil
	case !st.started:
		return m, endNotStarted, fmt.Errorf("%w: %w", ErrNotStarted, runErr)
	case errors.Is(runErr, tea.ErrInterrupted) || ctx.Err() != nil:
		if final == nil {
			return st.last, endEnded, runErr
		}
		got, ok := unwrap[M](final)
		if !ok {
			return st.last, endEnded, errors.Join(runErr, fmt.Errorf("launch: the final model is a %T, not a %T", final, m))
		}
		return got, endEnded, runErr
	default:
		return st.last, endCrashed, fmt.Errorf("%w: %w", ErrCrashed, runErr)
	}
}

// targets are the streams the TUI reads and draws on, and the controlling
// terminal Run opened for them, if any.
type targets struct {
	in       io.Reader
	out      io.Writer
	ttyIn    io.ReadCloser
	ttyOut   io.WriteCloser
	ttyOpen  bool
	ttyClose bool // close has run
}

// openTargets resolves d's targets on s, opening the controlling terminal
// once when either needs it.
func openTargets(s Streams, d Decision, open opener) (*targets, error) {
	t := &targets{}
	if d.In == TargetTTY || d.UI == TargetTTY {
		in, out, err := open()
		if err != nil {
			return nil, err
		}
		t.ttyIn, t.ttyOut, t.ttyOpen = in, out, true
	}
	switch d.In {
	case TargetStream:
		t.in = s.In
	case TargetTTY:
		t.in = t.ttyIn
	}
	switch d.UI {
	case TargetStream:
		t.out = s.Out
	case TargetErr:
		t.out = s.Err
	case TargetTTY:
		t.out = t.ttyOut
	}
	if t.in == nil || t.out == nil {
		return nil, errors.Join(fmt.Errorf("launch: the decision has no input or no drawing stream (in %s, ui %s)", d.In, d.UI), t.close())
	}
	return t, nil
}

// close closes each distinct file openTargets opened, once.
func (t *targets) close() error {
	if !t.ttyOpen || t.ttyClose {
		return nil
	}
	t.ttyClose = true
	var errs []error
	if err := t.ttyIn.Close(); err != nil {
		errs = append(errs, err)
	}
	if any(t.ttyOut) != any(t.ttyIn) {
		if err := t.ttyOut.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("launch: closing the controlling terminal: %w", err)
	}
	return nil
}

// saveAll saves the state of in and out, once each when they are one
// stream, and returns what puts them back.
func saveAll(in io.Reader, out io.Writer) ([]func() error, error) {
	streams := []any{in}
	if any(out) != any(in) {
		streams = append(streams, out)
	}
	var restores []func() error
	for _, v := range streams {
		r, err := save(v)
		if err != nil {
			return nil, err
		}
		restores = append(restores, r)
	}
	return restores, nil
}

// terminalFile reports whether w is a terminal file, whose size Bubble Tea
// reads itself.
func terminalFile(w io.Writer) bool {
	f, ok := w.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(f.Fd())
}
