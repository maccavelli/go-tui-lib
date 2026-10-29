// Package transcript keeps what a line-mode CLI has shown, and prints it
// again at the terminal's new width once resizing settles
// (docs/decisions/0023-MADR-line-mode-components.md D9, D13, A3 item 15,
// A8), as gobble-cli's line session does (its 0030-MADR D3 and D4).
//
// A Transcript holds pieces in the order they were shown: text kept as it
// was written, cleaned (AddText); a function of the width, drawn anew at
// each width, such as a help listing or a log record (AddDrawn); and a
// reply, a stream.Printer, drawn again from its source by Printer.At
// (AddReply). It keeps at most WithMaxBytes bytes, 4 MiB by default; past
// that, the oldest pieces go.
//
// The host gives it the width with Resize. The first Resize sets the width;
// a later one with another width starts the settle, WithSettle after the
// last resize, 150 ms by default, whose deadline Timeout gives. Once it has
// passed, HandleTimeout gives the reprint when the host says its output is
// idle: at a line's start, its Printer idle. A reprint still due waits for
// the next HandleTimeout(now, true), so the host calls it whenever its
// output becomes idle; it gives "" when nothing is due. Timeout gives no
// deadline then, so a host never waits on one that has passed.
//
// The reprint is SGR 0, then, as one synchronized update (mode 2026, which
// WithSynchronized turns off), CSI 2 J, CSI 3 J, which clears the
// terminal's scrollback and with it what came before the program, and the
// cursor home; then the newest pieces drawn at the width back to
// WithMaxRows rows, 10,000 by default, after the notice line
// MsgEarlierOutput when anything was left out. Its line breaks are CR LF,
// so it reads alike in raw mode. The clear is the price of a reprint that
// lines up: the terminal's own re-wrap of a table, a diff or a code block
// cannot. WithoutScrollbackClear leaves CSI 3 J out, keeping the scrollback
// and the old rows in it. kitty clears the scrollback on CSI 3 J though its
// terminfo leaves E3 out, and iTerm2 may ask the user first.
//
// Before writing the reprint, a host removes its inline.Region with Clear,
// and after it draws the region again. A Transcript does no I/O and is used
// on one goroutine.
//
// The package links no Charm module but x/ansi.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package transcript
