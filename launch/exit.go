package launch

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// ExitError carries an exit status through any framework: Kong's
// FatalIfErrorf finds it with errors.As, and urfave/cli's HandleExitCoder
// through its ExitCode method. Err may be nil.
type ExitError struct {
	Code int
	Err  error
}

// Error is Err's text, or "exit status <Code>" when Err is nil.
func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit status %d", e.Code)
	}
	return e.Err.Error()
}

// Unwrap is Err.
func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode is Code.
func (e *ExitError) ExitCode() int { return e.Code }

// ExitCode is the process exit status for err, as a shell expects it
// (docs/decisions/0013-MADR-cli-integration-helpers.md §9). An error with
// an ExitCode() int method anywhere in its chain, such as an ExitError,
// gives that. Otherwise:
//   - nil: 0;
//   - ErrNotStarted: 2, a usage error, as the flag package gives;
//   - tea.ErrInterrupted: 130, 128 + SIGINT;
//   - context.DeadlineExceeded: 124, as timeout(1) gives;
//   - context.Canceled: 130, since a cancel is how the program's own
//     interrupt arrives;
//   - tea.ErrProgramPanic: 2, an unrecovered panic's status;
//   - anything else: 1.
//
// A program whose cancel came from SIGTERM, and that wants 143, checks for
// that first.
func ExitCode(err error) int {
	if c, ok := errors.AsType[interface {
		error
		ExitCode() int
	}](err); ok {
		return c.ExitCode()
	}
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrNotStarted):
		return 2
	case errors.Is(err, tea.ErrInterrupted):
		return 130
	case errors.Is(err, context.DeadlineExceeded):
		return 124
	case errors.Is(err, context.Canceled):
		return 130
	case errors.Is(err, tea.ErrProgramPanic):
		return 2
	default:
		return 1
	}
}
