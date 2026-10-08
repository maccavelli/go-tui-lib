package launch

import "github.com/charmbracelet/x/term"

// isTerminal reports whether v is a terminal: its own answer when it has an
// IsTerminal() bool method, as a test's fake or an SSH session's PTY does,
// else term.IsTerminal on its descriptor. A value with neither is not one.
// The file mode is never asked: /dev/null is a character device and not a
// terminal (docs/decisions/0013-MADR-cli-integration-helpers.md §4, P7).
func isTerminal(v any) bool {
	if t, ok := v.(interface{ IsTerminal() bool }); ok {
		return t.IsTerminal()
	}
	if f, ok := v.(interface{ Fd() uintptr }); ok {
		return term.IsTerminal(f.Fd())
	}
	return false
}

// size is v's size in cells: its own answer when it has a Size() (width,
// height int) method, else term.GetSize on its descriptor (A1.4). It is 0,
// 0 when neither answers.
func size(v any) (width, height int) {
	if s, ok := v.(interface{ Size() (int, int) }); ok {
		return s.Size()
	}
	if f, ok := v.(interface{ Fd() uintptr }); ok {
		if w, h, err := term.GetSize(f.Fd()); err == nil {
			return w, h
		}
	}
	return 0, 0
}
