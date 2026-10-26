// Package msgcat is the shape of a program's message catalogue: the text
// the library shows a user or an agent, in the program's own words
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9).
//
// Each package that shows text names its messages with MessageID
// constants, "<package>.<name>", keeps their English defaults, and takes a
// program's Catalog through an option of its own. A Catalog gives a
// message's text, or false to keep the default; Table is one read from a
// map, and CatalogFunc adapts a function. Format picks the text and places
// its arguments with fmt, so a translation may reorder them with fmt's
// explicit indexes, such as "%[2]d of %[1]d", and writes a literal percent
// sign as "%%". Check reports a translation whose verbs differ from its
// default's, for a program's own test.
//
// msgcat imports the standard library alone, so every package, the
// narrowest core among them, may use it. It keeps no catalogue of its
// own: the program passes one.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package msgcat
