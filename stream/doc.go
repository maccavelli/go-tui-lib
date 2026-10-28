// Package stream prints a streamed reply's Markdown in a line-mode CLI's
// scrollback, styled and wrapped, each line once
// (docs/decisions/0023-MADR-line-mode-components.md D7, D8, A3, A6, A7;
// docs/decisions/0009-MADR-streaming-content-engine.md A5).
//
// A Printer takes the reply as it arrives and gives the lines that are
// stable: cleaned by safetext, styled by a Markup, every Markdown marker
// kept, and a prose line wrapped at whole words to the width. A paragraph's
// line waits for the next, so a setext heading and a table's header are
// drawn as what they are; a table waits until it ends, then is laid out in
// aligned columns, or printed as it came under WithoutTableLayout; a
// fence's lines print as they come. Nothing printed is changed later, so a
// scrollback that cannot be redrawn reads right.
//
// A Markup is a Style for each part a reply has, an SGR attribute set; the
// zero Markup styles nothing, and DefaultMarkup is gobble-cli's. With the
// zero Markup, no width and WithoutTableLayout, the output is the cleaned
// input, byte for byte. A link's
// text is underlined, and a reference link only once its definition has
// been printed.
//
// The package links no Charm module but x/ansi.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package stream
