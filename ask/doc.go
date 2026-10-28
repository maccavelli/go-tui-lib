// Package ask asks a question inline, below a line-mode CLI's scrollback:
// rows of answers, each chosen by one key, at most one of which reads a
// reason after it (docs/decisions/0023-MADR-line-mode-components.md D4).
//
// Layout lays a Question out at a width, a pure function: the title
// wrapped at spaces, its continuation rows indented two; each row's label
// padded to the widest; each answer "[k] label", two spaces apart, a row
// that does not fit broken between answers, its continuation under its
// first answer; then the prompt. A word or an answer wider than its whole
// row is broken at the width, so no row is wider and every word is shown
// (A2 item 5).
//
// An Asker is the question as an engine that does no I/O, as every
// line-mode component is (D2): the host gives it the width, each decoded
// input and the time, and drains NextEvent, writing each AskWrite's bytes
// to the terminal, until AskDone gives the Response. It draws in an
// inline.Region, the host's through WithRegion or its own. The question
// and its answer, in words, stay in the scrollback.
//
// A key no answer shows is ignored. The cancel answer's key, ctrl+c, and
// the host's Cancel, at the end of its input or for a reason of its own,
// cancel. An answer with Reason reads a line, up to lineedit's submit,
// enter by default, edited by lineedit's editing commands in the
// program's keymap (WithKeymap), bounded at DefaultReasonLimit bytes by
// default.
//
// The text shown, the caller's and a model's, is cleaned once, on the way
// in, never below safetext's Line rules. Its own words, such as the label
// before a reason, come from the program's catalogue through WithCatalog;
// Messages gives their English defaults.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package ask
