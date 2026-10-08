package launch

import "errors"

// The two ways a TUI the program asked for does not run to its end. Run
// wraps each around the cause, so errors.Is finds it.
var (
	// ErrNotStarted: the TUI did not start. The decision was plain, the
	// controlling terminal did not open, or Bubble Tea failed before the
	// model's Init ran. Nothing was drawn by the model, and the terminal is
	// as it was, so the program can run its CLI mode instead.
	ErrNotStarted = errors.New("launch: the TUI did not start")
	// ErrCrashed: the TUI started, then stopped on its own: a panic Bubble
	// Tea recovered, or a failure reading input. The terminal was restored,
	// and Run returns the last good model, so the program can continue in
	// its CLI mode from where the TUI was.
	ErrCrashed = errors.New("launch: the TUI stopped")
)
