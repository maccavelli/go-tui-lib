// Package lineedit is the line a line-mode CLI's user types, below its
// scrollback: a prompt, the text in whole words to the width, history,
// completion, bracketed paste and an external editor
// (docs/decisions/0023-MADR-line-mode-components.md D6, A3, A5).
//
// A Composer is the line as an engine that does no I/O, as every
// line-mode component is (D2): the host gives it the width, each decoded
// input and the time, and drains NextEvent, writing each EditWrite's bytes
// to the terminal. EditSubmit gives each submission, which stays in the
// scrollback, an empty prompt drawn below it; EditEOF the end of input;
// EditInterrupt ctrl+c, whose meaning is the host's. It draws in an
// inline.Region, the host's through WithRegion or its own, so a reply the
// host prints through the region's Print goes above the line being typed.
//
// The text is grapheme clusters under the width method, so a combining
// mark, a ZWJ sequence or a flag is one cursor step and its true width.
// It is laid out in whole words, each row after the first indented by the
// prompt's width; a word wider than a whole row is broken at its end. A
// line break, from alt+enter, shift+enter where the terminal reports it,
// or a trailing backslash before enter, starts a line after the
// continuation prompt.
//
// Every key but ctrl+c is a command in the program's keymap, in the scope
// ComposerScope: DefaultRules binds gobble's editor's keys, Readline's
// words for alt+b, alt+f, alt+d and alt+backspace, and ctrl+w to white
// space; WithKeymap takes the program's keymap, a user's file among its
// layers. Up and down move by row, and from the first or the last row
// recall the History, x/term's interface; MemHistory keeps one in memory,
// with its file form. ctrl+l clears the screen only WithClearScreen, and
// ctrl+x ctrl+e asks the host for the external editor only
// WithExternalEditor: the library starts no process.
//
// Typed text, a paste, a completion and a recalled entry are cleaned on the
// way in, never below safetext's Line rules, a paste and an entry keeping
// their line breaks and tabs.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package lineedit
