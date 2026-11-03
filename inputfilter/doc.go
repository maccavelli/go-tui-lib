// Package inputfilter coalesces a terminal's noisy input before it queues
// (docs/decisions/0009-MADR-streaming-content-engine.md §6, A6 item 8;
// docs/decisions/0009-PLAN-streaming-content-engine.md D10): wheel notches
// summed, mouse motion throttled, and, where a terminal does not bracket
// its pastes, a paste told from typing by its speed.
//
// The package has two layers. The pure layer takes the time from its
// caller and reads no clock, and each of its types is used on one
// goroutine:
//
//   - a Wheel sums Scroll events, a change of direction starting the sum
//     again; the first passes at once and later ones at most one an
//     interval, each carrying the sum, and Residual takes what is held;
//   - a Throttle passes one motion sample an interval;
//   - a PasteBurst reads termkey.Input: a run of characters that come
//     faster than anyone types is held, and given as one
//     termkey.InputPaste, an Enter or a Tab inside it a newline or a tab;
//     a bracketed paste turns it off for good, since the terminal has
//     shown that it brackets them;
//   - a Reassembler joins a mouse report or a probe's reply that the
//     terminal's reader gave up on at its escape timeout, from the
//     unknown head it gave up on, repairs an X10 report ConPTY
//     re-encoded, and drops the replies a ReplyMatcher knows;
//   - ParseReport reads a mouse or focus report, Typeahead keeps the
//     typing of a batch, NormalizePaste makes a paste's line ends LF, and
//     PastedPath reads a dropped path.
//
// The Flusher is the driver over the stages, and its methods are safe
// for concurrent use: it runs a Reassembler, the typeahead (Capture,
// Replay and Quarantine), paste normalisation and a PasteBurst in turn.
// When something it holds falls due, it says so on Flushes, and the
// caller takes it with Flush on its own goroutine, so that nothing leaves
// the stages behind a key typed after it. A line-mode
// host selects on Flushes beside its input; a Bubble Tea program uses
// inputfilter/inputfiltertea, whose Filter is given to tea.WithFilter.
//
// The package links no Charm module.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package inputfilter
