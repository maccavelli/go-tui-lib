// Package inline draws a component's live rows below the terminal's
// scrollback, and redraws them in place, for a line-mode CLI that links no
// TUI framework (docs/decisions/0023-MADR-line-mode-components.md D3).
//
// A Region is an engine that does no I/O: each method returns the bytes
// to write, and the host writes them, in order, to the terminal. It draws
// by relative moves only, rewriting a changed row from what it shares with
// the row before; it prints text above itself, as x/term's Terminal.Write
// does; and on a resize it asks the terminal where its cursor is, reads
// the report from the input, and redraws by how the terminal re-wrapped
// its rows. Its rows are laid out by the component that drew them, and
// cleaned by it: each is one line of display text, with no line break, and
// no control but its SGR and OSC 8 sequences.
//
// A Reader reads the host's input only when asked, one read at a time, and
// never closes it: on Windows a console read keeps the mode it started in,
// and closing console input while a read is pending hangs the process.
//
// The terminal's mode, its size and the resize signal stay the host's: the
// package reads no environment, opens no file and starts no process.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package inline
