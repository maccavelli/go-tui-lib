// Package lineedittea is lineedit's composer as a workspace pane, for a
// Bubble Tea program (docs/decisions/0023-MADR-line-mode-components.md D6).
//
// A ComposerPane drives the same lineedit.Composer the line mode does,
// giving it the keys and the pastes that reach the pane, and draws the
// Composer's View at the pane's size, so the TUI's composer and the CLI's
// edit alike. What the Composer gives, a submission, the end of input,
// ctrl+c and the external editor's request, comes to the program as an
// EventMsg; the program answers the editor's with the pane's Composer's
// SetText.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package lineedittea
