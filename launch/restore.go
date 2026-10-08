package launch

import "github.com/charmbracelet/x/term"

// Restorer undoes a terminal mode a program set outside Bubble Tea, such as
// termcap's mode 2031: Restore returns the bytes that reset it, or "" when
// there is nothing to undo. *termcap.Prober satisfies it. Run writes the
// bytes to the stream the TUI drew on, on every outcome, after Bubble Tea
// has shut down; resetting a mode that is already off is harmless.
type Restorer interface {
	Restore() string
}

// save is the saved-state seam: it saves v's terminal state, and returns
// what puts it back. Tests replace it.
var save = saveState

// saveState saves the state of a stream with a terminal descriptor, with
// term.GetState, and does nothing for any other stream.
func saveState(v any) (restore func() error, err error) {
	f, ok := v.(interface{ Fd() uintptr })
	if !ok || !term.IsTerminal(f.Fd()) {
		return func() error { return nil }, nil
	}
	st, err := term.GetState(f.Fd())
	if err != nil {
		return nil, err
	}
	return func() error { return term.Restore(f.Fd(), st) }, nil
}
