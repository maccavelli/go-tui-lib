// Package fuzzy finds a query in candidate strings and ranks them, the same
// way for every list a program filters: a command palette, slash and path
// completion, and pickers (docs/decisions/0008-MADR-command-palette.md §2).
//
// A Matcher is a compiled query. It matches grapheme clusters, never parts
// of one, so a range it reports never splits an emoji or a letter and its
// combining mark. Two clusters match when their runes are equal under
// simple case folding (unicode.SimpleFold), after the Normalizer when one is
// given. Smart case is the default: a query with an upper-case letter is
// case-sensitive.
//
// A match has a Grade and a value. The grade comes first: Exact (the
// candidate, its last path element or its stem equals the query), Prefix,
// Boundary (a whole word or path segment equals the query, or every query
// cluster lands on a word boundary), then Subsequence. Within a grade the
// value orders matches: an alignment of the query, scored with fzf's
// constants, that rewards consecutive clusters, word and path boundaries,
// camelCase and digit transitions and the first cluster, and penalises
// gaps.
//
// The package imports only the standard library and x/ansi, so a program
// can use it without Bubble Tea. Widths are cells in the method the caller
// passes to Cells.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package fuzzy
