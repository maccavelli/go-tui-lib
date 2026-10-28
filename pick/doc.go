// Package pick picks one entry from a filtered list, inline, below a
// line-mode CLI's scrollback: a model, a session, a file
// (docs/decisions/0023-MADR-line-mode-components.md D5, A3).
//
// A Chooser is the picker as an engine that does no I/O, as every
// line-mode component is (D2): the host gives it the width, each decoded
// input and the time, and drains NextEvent, writing each PickWrite's bytes
// to the terminal, until PickDone gives the Entry chosen, or Canceled. It
// draws in an inline.Region, the host's through WithRegion or its own.
//
// The layout is fzf's reverse layout (A3.4 item 1): the query row first,
// the prompt before the query and the count after it, at the right; then
// the rows, as tall as WithHeight, or the entries when fewer, for the
// whole question. Each row is the palette's: the focus glyph on the
// selected row, so the selection never rests on colour, the entry's Text,
// and its Detail at the right, dropped when the Text would keep fewer than
// 8 cells.
//
// Typing edits the query, through a fixed set of editing keys; each change
// ranks the entries' Text again with fuzzy's matcher, best first, ties in
// the order given, and an empty query keeps that order. Above WithIndexAbove
// entries a fuzzy.Index narrows the candidates first. The other keys are
// commands in the program's keymap, in the scope ChooserScope: DefaultRules
// binds up and ctrl+p, down and ctrl+n, pgup and pgdown, enter, and esc and
// ctrl+g, and WithKeymap takes the program's keymap, a user's file among
// its layers. ctrl+c cancels, outside the keymap, as the host's Cancel
// does.
//
// One line is left in the scrollback: the prompt and the entry chosen, or
// the prompt and the word for a cancel. Every entry's text is cleaned once,
// on the way in, never below safetext's Line rules; the event carries the
// caller's Entry as given. The Chooser's own words come from the program's
// catalogue through WithCatalog; Messages gives their English defaults.
//
// The same []Entry serves the TUI through palette.FromEntries, ranked by the
// same matcher.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package pick
