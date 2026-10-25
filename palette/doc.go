// Package palette finds and runs anything from one overlay: commands,
// files, panes, themes, or a program's own items, through providers that
// search off the event loop and stream their hits back
// (docs/decisions/0008-MADR-command-palette.md).
//
// A Provider yields Hits for a Search; the picker owns the goroutines.
// Each keystroke starts a new generation: the last search is cancelled,
// its late hits are dropped, and the providers available where the
// palette opened search again, each after its own debounce. Their hits
// reach the program's Update in batches, are kept best first up to a
// limit, and are de-duplicated by ID. A provider that fails or panics is
// one muted row naming it; its error goes to the logger, never the screen.
//
// palette is a Bubble Tea package: a hit's Run returns a tea.Cmd, and the
// picker is a workspace pane.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package palette
