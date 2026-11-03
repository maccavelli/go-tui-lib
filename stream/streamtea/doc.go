// Package streamtea is stream's Doc as a workspace pane, for a Bubble Tea
// program (docs/decisions/0009-MADR-streaming-content-engine.md §5, A7
// items 2 to 4).
//
// A DocPane draws a stream.Doc at the pane's width, measured with the
// workspace's width method and marked with its theme's glyphs, and follows
// the document's tail until the user scrolls up, and again once they
// scroll back to the end. Given a stream.Buffer, it pulls the Buffer's
// lines on each frame.DueMsg, as its pacer releases them. Its keys are
// commands: DefaultRules binds them in the "kind:docpane" scope, for the
// program's one keymap beside workspace.DefaultRules, and the pane runs
// them itself, as a workspace.CommandConsumer.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package streamtea
