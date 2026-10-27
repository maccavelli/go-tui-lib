// Package asktea asks ask's question in a workspace overlay, for a Bubble
// Tea program (docs/decisions/0023-MADR-line-mode-components.md D4).
//
// A Dialog is the overlay's pane. It drives the same ask.Asker the line
// mode does, giving it the keys, and draws the Asker's View at the
// overlay's width, so the question reads alike in both modes. The answer
// comes to the program as an AnsweredMsg; esc, which closes any overlay,
// ends the question canceled.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package asktea
