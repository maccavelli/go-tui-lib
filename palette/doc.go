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
// A Picker draws it all in a modal overlay of a workspace: an input, a
// window of the hits with the matched cells underlined, danger badges and
// key hints, and a help line with a count. Its keys are commands in one
// keymap with the workspace's: build it with DefaultRules beside
// workspace.DefaultRules, and register OpenCommand's command, which ctrl+p
// runs. The picker runs its own moves as a workspace.CommandConsumer, so
// they take effect in the order their keys were pressed. Accepting a hit
// runs its Run on the event loop and closes the picker; closing it, by any
// means, cancels its searches and stops its providers.
//
// A hit with Params opens an argument stage in the same overlay first: it
// asks for each string, integer, number and boolean parameter, and any
// with an enum, in order, from a list where there is one; it holds them to
// Required, Default, Min and Max, with the reason for a refused value in
// the help line; it skips Hidden ones and masks Secret ones; and it runs
// the hit with the values as one JSON object. A hit with any other
// parameter is sent as SelectedMsg, and nothing runs.
//
// A Frecency remembers the hits a picker made WithRecents runs, by ID
// only, with a score that halves every half-life; Recents puts those hits
// first. WriteTo and ReadFrom keep it between runs as versioned JSON.
//
// Four providers are built in: FromRegistry's commands, under the prefix
// ">", which run through the registry with command.OriginPalette; Static's
// fixed items; Panes' panes to focus or show; and Files' paths, under "@",
// walked from an fs.FS the caller gives and ranked by their last element.
// A picker made WithoutInput is a completion pop-up for an editor pane,
// which drives it with SetQuery and is sent SelectedMsg.
//
// palette is a Bubble Tea package: a hit's Run returns a tea.Cmd, and the
// picker is a workspace pane.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package palette
