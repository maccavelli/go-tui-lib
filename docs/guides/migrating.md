# Migrating between releases

Within a major version nothing is removed or retyped: a renamed name keeps
working, as a deprecated wrapper, alias or field, until the next major
version, and `staticcheck` reports each use of one as SA1019. A removal
needs a major version, at a new module path
([AGENTS.md, API conventions](../../AGENTS.md#api-conventions), rule 1).
Up to `v0.12.0`, a deprecated name was removed in the next minor release.

## Migrating to `v2.7.0`

`v2.7.0` is a minor release. It adds four names to `termcap` and
deprecates one, and makes the line mode's editing and drawing faster.
Nothing is removed or retyped, and every drawing is byte for byte what
it was: `make apicheck` against `v2.6.2` reports 0 incompatible changes
([0023-PLAN-line-mode-components](../decisions/0023-PLAN-line-mode-components.md)
Step 7c;
[0024-PLAN-terminal-text-and-convention-opt-outs](../decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md)
Step 1). A program on `v2.6.2` needs no change:

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.7.0
```

### Added

- **`termcap.WithoutQuery(names...)`** leaves the named queries out of the
  probe's batch and sends the rest. The built-in queries' names are
  `kitty-keyboard`, `mode-2031`, `mode-2048`, `mode-1004`, `dsr-996`,
  `background`, `da2`, `xtversion`, `notify-query`, `kitty-graphics`,
  `foreground`, `palette`, `mode-2026` and `mode-2027`. A `WithQuery`'s
  `Name` names it too. DA1, the sentinel, has no name. A name no query has
  sends nothing, and each query fact says why, with
  **`termcap.ReasonUnknownQuery`**
  ([0024-MADR](../decisions/0024-MADR-opt-in-and-opt-out-coverage.md) D5).
- **`termcap.WithoutGatedQueries()`** never sends the gated queries, even
  where the heuristic would; it wins over `WithoutHeuristic`.
- **`termcap.WithoutQueries()`** sends nothing, as `WithDisabled` did, by
  the rule's name for an option that turns a default off.

### Deprecated

- **`termcap.WithDisabled`:** use `WithoutQueries`. It stays until `v3`.

### Changed

- **The probe reads its environment as its own system does.** A probe
  given `WithGOOS("windows")` matches variable names in any case, as
  Windows does, on any host; one given another system matches them
  exactly. `termcap.FromEnv` matches by its `goos` argument the same way.
  Only a probe whose system a program set to another changes.
  `Env.LookupEnv`, on its own, follows the running system as before
  ([0024-MADR](../decisions/0024-MADR-opt-in-and-opt-out-coverage.md) A2).
- **`lineedit`, `inline` and the line `ask` and `pick` edit are faster,
  and allocate less** ([0023-MADR](../decisions/0023-MADR-line-mode-components.md)
  A12, for issue #13).
  - A key typed and drawn, the issue's benchmark, takes 1.43 µs, 4
    allocations and 264 bytes on an Apple M1 Pro, from 14.5 µs, 52 and
    10,092.
  - A key at the end of a text near the 4,096-rune bound takes 4.4 µs, from
    229 µs; one at the start of twenty rows, 7.1 µs, from 83 µs.
  - On the Linux and Windows hosts the issue names, the key takes 5.0 µs
    and 2.3 µs, against gobble-cli's budgets of 15 µs and 4.2 µs.

  The Composer keeps its layout and lays out again only the rows an edit
  reaches. The line splices its clusters and keeps their widths. The
  region draws each changed row once into one buffer. The dispatcher
  skips the bindings a key cannot match. Tests hold each key to 8
  allocations and 808 bytes.

## Migrating to `v2.6.2`

`v2.6.2` is a patch: nothing exported changes, no default changes, and
every reply and drawing that did not panic is read and drawn as before:
`make apicheck` against `v2.6.1` reports 0 incompatible changes
([0023-PLAN-line-mode-components](../decisions/0023-PLAN-line-mode-components.md)
Step 7a). A program on `v2.6.1` needs no change:

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.6.2
```

### Fixed

- **`termcap.ParseReply` no longer panics on a sequence with more than 32
  parameters.** x/ansi's parser stores a digit past its parameter buffer,
  and `ParseReply`'s bound missed an ESC followed by intermediate bytes
  before `[` or `P`, such as `ESC SP [`; it panicked so since `v2.0.0`.
  Every path to the parameters is now counted, and such a reply is
  `ReplyUnknown`
  ([0023-MADR](../decisions/0023-MADR-line-mode-components.md) A10).
- **`inlinetest.Screen` no longer panics** on such a sequence written to
  it: it shows nothing for it, and draws the text around it.

## Migrating to `v2.6.1`

`v2.6.1` is a patch: nothing exported changes, no default changes, and
every output is as it was: `make apicheck` against `v2.6.0` reports 0
incompatible changes
([0023-PLAN-line-mode-components](../decisions/0023-PLAN-line-mode-components.md)
Step 6a). A program on `v2.6.0` needs no change:

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.6.1
```

### Changed

- **`stream` and `transcript` are faster, and allocate less.** On an Apple
  M1 Pro, a reprint of 4 MiB of replies back to 10,000 rows at 60 columns
  takes 5.2 ms, 6.5 MB and 43,541 allocations, from `v2.6.0`'s 12.3 ms,
  15.6 MB and 186,585; a reply streamed in 64-byte chunks at 100 columns
  takes about 24 µs and 200 allocations, from 34 µs and 636. The word wrapper
  takes each word as a slice of its line and measures it once;
  `Printer.At` draws its source, which is cleaned already, without
  cleaning it again; and a line no longer in bytes than the width is
  counted one row without being measured
  ([0023-MADR](../decisions/0023-MADR-line-mode-components.md) A9). Tests
  hold each to an allocation budget.

## Migrating to `v2.6.0`

`v2.6.0` adds two packages and an option, removes or retypes nothing, and
changes no default: `make apicheck` against `v2.5.0` reports 0
incompatible changes
([0023-PLAN-line-mode-components](../decisions/0023-PLAN-line-mode-components.md)
Steps 5 and 6). A program on `v2.5.0` needs no change to build, and
behaves as it did:

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.6.0
```

### Added

- **`stream`,** a streamed reply's Markdown printed line by line, each
  line once (issue #10): `Printer`, from `NewPrinter` with `WithMarkup`,
  `WithWidth`, `WithMethod`, `WithRules` and `WithoutTableLayout`, and
  its `Write`, `Flush`, `SetWidth`, `Source`, `Idle` and `At`, its source
  drawn again at another width; `Markup`, a `Style` for each part of a
  reply, and `DefaultMarkup`, gobble-cli's; `Style`, a set of SGR
  attributes, with `Apply`. A paragraph's line is held until the next, so
  a setext heading and a table's header are drawn as what they are; a
  table is held until it ends, and laid out; prose is wrapped at whole
  words, every Markdown marker kept. The zero `Markup` styles nothing. It
  links no Charm module but `x/ansi`
  ([0023-MADR](../decisions/0023-MADR-line-mode-components.md) D7, A3,
  A6 to A8). 0009-MADR's `Doc`, `Buffer` and `Pane` are still to come,
  `Buffer` and `Pane` in `stream/streamtea` (its A5).
- **`transcript`,** what a line-mode CLI has shown, printed again at the
  new width once a resize settles (issue #10): `Transcript`, from
  `NewTranscript` with `WithMaxBytes`, `WithMaxRows`, `WithSettle`,
  `WithMethod`, `WithoutScrollbackClear`, `WithSynchronized`, `WithRules`,
  `WithCatalog` and `WithNoticeStyle`; `AddText`, `AddDrawn`,
  `AddReply`, `Resize`, `Timeout` and `HandleTimeout`; the defaults
  `DefaultMaxBytes` (4 MiB), `DefaultMaxRows` (10,000) and
  `DefaultSettle` (150 ms); `Messages` and `MsgEarlierOutput`. A reprint
  clears the terminal's scrollback, and with it what came before the
  program, unless `WithoutScrollbackClear`
  ([0023-MADR](../decisions/0023-MADR-line-mode-components.md) D9, D13,
  A8). It links no Charm module but `x/ansi`.
- **`pick`:** `WithMarkup`, a `stream.Markup` whose `Match` style marks
  the cells of each entry the query matched; nothing is styled by
  default.

## Migrating to `v2.5.0`

`v2.5.0` adds a package and its adapter, and an option, and removes or
retypes nothing: `make apicheck` against `v2.4.0` reports 0 incompatible
changes
([0023-PLAN-line-mode-components](../decisions/0023-PLAN-line-mode-components.md)
Step 4). A program on `v2.4.0` needs no change to build. Two behaviours
change, as below: `ask`'s reason and `pick`'s query are edited by
`lineedit`'s commands in the program's keymap, the same keys by default;
and the layouts of both are measured as they are drawn.

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.5.0
```

### Added

- **`lineedit`,** the line a user types, inline (issue #9): `Composer`,
  the line as an engine that does no I/O, laid out in whole words, with
  `ComposerEvent`, `ComposerEventKind`, `View`, `Text` and `SetText`;
  `History`, x/term's interface, and `MemHistory`, one in memory with its
  file form, `DefaultHistorySize`; `Completer` and `CompleterFunc`;
  `PasteMode`, `PasteInsert` and `PasteSubmit`; the options `WithPrompt`,
  `WithContinue`, `WithHeader`, `WithHistory`, `WithCompleter`,
  `WithKeymap`, `WithLimit` with `DefaultLimit`, `WithPaste`,
  `WithExternalEditor`, `WithClearScreen`, `WithMethod`, `WithGlyphs`,
  `WithRegion` and `WithRules`; its keys, `DefaultRules`, the command IDs
  from `CharLeftID` to `ExternalID`, and `ComposerScope`. It links no
  Charm module but `x/ansi`.
- **`lineedit/lineedittea`,** the same composer in a workspace pane:
  `ComposerPane`, `EventMsg`, `DefaultID`, and the options `WithID` and
  `WithComposer`.
- **`ask`:** `WithKeymap`, the program's keymap for a reason's keys.

### Changed

- **`ask` and `pick`:** a reason's and a query's editing keys are
  `lineedit`'s commands, in `lineedit.ComposerScope`, resolved after
  `pick`'s own. With the defaults, every key before does what it did,
  but alt+backspace, which deleted back to a space and now deletes back
  by Readline's words, letters and digits, as ctrl+w still deletes to a
  space; alt+f, alt+b and alt+d are new, by the same words, and ctrl+t
  transposes. A program that gives `pick` its own keymap holds
  `lineedit.DefaultRules` in it beside `pick.DefaultRules`
  (0023-MADR A5 item 4).

### Fixed

- **`ask` and `pick`:** a mark or a variation selector with no base, which
  joins the space drawn before it, is measured where it lands, so a row of
  `ask.Layout` or of a `pick.Chooser` is never wider than the width as the
  terminal draws it; before, under `ansi.GraphemeWidth`, such a row could
  be a cell too wide, and lose its last cell to the region
  ([0023-MADR](../decisions/0023-MADR-line-mode-components.md) A5 item 7).

## Migrating to `v2.4.0`

`v2.4.0` adds a package and a palette provider, removes or retypes
nothing, and changes no default: `make apicheck` against `v2.3.0` reports
0 incompatible changes
([0023-PLAN-line-mode-components](../decisions/0023-PLAN-line-mode-components.md)
Step 3). A program on `v2.3.0` needs no change to build, and behaves as it
did, but for the test screen's fix below:

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.4.0
```

### Added

- **`pick`,** a picker drawn inline (issue #8): `Entry`, and `Chooser`,
  the pick as an engine that does no I/O, with `ChooserEvent`,
  `ChooserEventKind` and `Cancel`; the options `WithHeight` with
  `DefaultHeight`, `WithPrompt`, `WithQuery`, `WithMethod`, `WithGlyphs`,
  `WithIndexAbove` with `DefaultIndexAbove`, `WithRegion`, `WithCatalog`,
  `WithRules` and `WithKeymap`; its keys, `DefaultRules`, the command IDs
  `SelectNextID`, `SelectPrevID`, `PageNextID`, `PagePrevID`, `AcceptID`
  and `CancelID`, and `ChooserScope`; its messages, `Messages`,
  `MsgCount`, `MsgCountMore`, `MsgNoResults` and `MsgCanceled`. It links
  no Charm module but `x/ansi`.
- **`palette`:** `FromEntries`, a provider of the same `[]pick.Entry`,
  ranked as a `pick.Chooser` ranks them.

### Fixed

- **`inline/inlinetest`:** `Screen` keeps a cluster wider than two cells,
  such as a ZWJ sequence under `ansi.WcWidth`, as wide as it is across a
  `Resize`, and blanks all of it when text covers one of its cells; before,
  a resize laid it out two cells wide, and the text after it two cells
  early (0023-PLAN Step 3, D12).
- **`fuzzy`:** `Matcher.Rank`'s documentation states its tie order, the
  shorter key and then the earlier item, which it has always had
  ([0023-MADR](../decisions/0023-MADR-line-mode-components.md) A4).

## Migrating to `v2.3.0`

`v2.3.0` adds packages and a reply kind, removes or retypes nothing, and
changes no default: `make apicheck` against `v2.2.0` reports 0
incompatible changes
([0023-PLAN-line-mode-components](../decisions/0023-PLAN-line-mode-components.md)
Steps 1 and 2). A program on `v2.2.0` needs no change to build, and
behaves as it did:

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.3.0
```

### Added

- **`inline`,** the live rows below a line-mode CLI's scrollback:
  `Region`, which draws and redraws them in place by relative moves and
  follows the terminal's re-wrap on a resize, with `Reflow`, and `Reader`,
  which reads the input only when asked and never closes it; the options
  `WithMethod`, `WithSynchronized`, `WithReflow`, `WithoutCursorQuery`,
  `WithReportWait` with `DefaultReportWait`, and `WithHeight`. It links no
  Charm module but `x/ansi`.
- **`inline/inlinetest`,** `Screen`, a terminal's screen for tests, by
  grapheme cluster, with the three re-wrap kinds and a cursor report.
- **`ask`,** a question asked inline (issue #7): `Question`, `Row`,
  `Answer` and `Response`; `Layout`, the question laid out at a width; and
  `Asker`, the question as an engine that does no I/O, with `AskerEvent`,
  `AskerEventKind`, `Cancel` and `View`; the options `WithMethod`,
  `WithGlyphs`, `WithRegion`, `WithReasonLimit` with `DefaultReasonLimit`,
  `WithCatalog` and `WithRules`; its messages, `Messages`,
  `MsgReasonPrompt`, `MsgReasonEcho` and `MsgCanceled`. It links no Charm
  module but `x/ansi`.
- **`ask/asktea`,** the same question in a workspace overlay: `Dialog`,
  `AnsweredMsg`, `DefaultID`, and the options `WithID`, `WithAnchor`,
  `WithWidth` and `WithAsk`.
- **`termcap`:** `ReplyCursorPosition`, a cursor position report, with
  `Reply.Row` and `Reply.Col`.

## Migrating to `v2.2.0`

`v2.2.0` adds a package, options, helpers and exported names, removes or
retypes nothing, and changes no default: `make apicheck` against `v2.1.0`
reports 0 incompatible changes
([0024-PLAN-command-and-control-opt-outs](../decisions/0024-PLAN-command-and-control-opt-outs.md)
Steps 2 to 8). A program on `v2.1.0` needs no change to build, and behaves
as it did until it sets one of the new options:

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.2.0
```

### Added

- **The defaults inventory,** [the defaults guide](defaults.md): every
  default the library applies, with the control that refuses or changes
  it, and the floors that have none, each with its reason
  ([AGENTS.md, API conventions](../../AGENTS.md#api-conventions), rule
  12).
- **`msgcat`,** a message catalogue: `MessageID`, `Catalog`,
  `CatalogFunc`, `Table`, `Format` and `Check`. It imports the standard
  library alone. The packages' own text moves onto it in a later release.
- **`command`:** `WithGatePolicy`, with `GatePolicy`, `GatePolicyFunc` and
  `DefaultGatePolicy`, which says which requests ask the gate;
  `Registry.Forget` and `Registry.ForgetAll`, which drop remembered
  answers; `WithoutBuiltins`, a registry without its three commands; and
  their IDs, `BuiltinList`, `BuiltinDescribe` and `BuiltinQuit`.
- **`workspace.WithoutOwnCommands`,** which sends a key's `workspace.*`
  command back as a `RequestMsg`, for the program's registry.
- **`launch`:** `Config.AllowDumb` and `Config.IgnoreCI`; `ColorProfile`
  and `GlyphTier`, the helpers `Decide` uses; and `WithoutEpilogue` and
  `WithEpilogueWriter`.
- **`palette`:** the picker's command IDs, `OpenID`, `SelectNextID`,
  `SelectPrevID`, `PageNextID`, `PagePrevID`, `AcceptID` and
  `CompleteID`, and `PickerScope`; `CompletionRules`, a pop-up's bindings
  under its own ID; and `Opener`, which `Open` calls on each provider that
  has it. A provider of your own that already has a method
  `Opened(when.Context)` is now called when the picker opens.

## Migrating to `v2.1.0`

`v2.1.0` adds a package and a message, and removes or retypes nothing:
`make apicheck` against `v2.0.0` reports 0 incompatible changes
([0008-PLAN-command-palette](../decisions/0008-PLAN-command-palette.md)
Steps 5 to 9). A program on `v2.0.0` needs no change to build:

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.1.0
```

### Changed

- **An overlay's pane is told when it closes.** Whenever the workspace
  removes an overlay, by `Pop`, `Close`, the `esc` binding or a `Push`
  that replaces its ID, the pane is sent `workspace.OverlayClosedMsg`
  through its `Update`, after `PaneBlurMsg` if it had the keyboard, and its
  command is batched with the closing's. A pane that ignores messages it
  does not know sees no difference.

### Added

- **`palette`,** the command palette
  ([the command palette guide](command-palette.md)): `Picker`, from
  `NewPicker`, with `Register`, `Open`, `OpenCommand` and `DefaultRules`;
  the providers `FromRegistry`, `Static`, `Panes`, `Files` and `Recents`;
  `Provider` and its optional capabilities; the argument stage over
  `command.Params`; `Frecency`, the recents store, with its JSON form; and
  completion pop-ups, `WithoutInput`, `SetQuery` and `SelectedMsg`.
- **`workspace.OverlayClosedMsg`,** above.

## Migrating to `v2.0.0`

`v2.0.0` is a new major version, at the module path
`github.com/maccavelli/go-tui-lib/v2`. A program on `v1.13.0` keeps it:
`go get -u` never moves a `v1` user to `v2`. It splits `termcap`,
`termsvc`, `command` and `keymap` into a core that links no Charm module
but `x/ansi`, and a sub-package that holds the core's Bubble Tea forms
([0020-MADR](../decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
D2 to D7, A2, A4.1). It adds `termkey`, a terminal's keys and a decoder
for them.

### The import path

Every import of the library moves to `/v2`:

```go
import "github.com/maccavelli/go-tui-lib/workspace"    // v1.13.0
import "github.com/maccavelli/go-tui-lib/v2/workspace" // v2.0.0
```

```bash
go get github.com/maccavelli/go-tui-lib/v2@v2.0.0
```

The packages keep their names, so the only change in a file is the import
line, then the names below.

### Moved and retyped

Each name `v2.0.0` moves or retypes, with the change a caller makes. A
Bubble Tea form now takes the core value it used to be a method of.

| `v1.13.0` | `v2.0.0` | The change |
| :--- | :--- | :--- |
| `termcap.Prober`, `termcap.NewProber(opts...)` | `termcaptea.Prober`, `termcaptea.NewProber(opts...)` | import `termcap/termcaptea`; the options stay `termcap`'s, and the methods are the same |
| `termcap.KeyboardFlags(caps)`, `KeyboardCaps.Enhancements` | `termcaptea.KeyboardEnhancements(caps)` | what goes in `View.KeyboardEnhancements`; a line-mode CLI reads `KeyboardCaps.Request` |
| `termcap.Caps.Profile`, a `colorprofile.Profile` | a `termcap.ColorProfile` | `termcaptea.ToColorProfile(caps.Profile)` where a `colorprofile.Profile` is wanted, and `FromColorProfile` back |
| `termcap.Reply{Msg, Raw}`, a Bubble Tea message or an undecoded reply's bytes | `termcap.Reply`, parsed: `Kind`, `Raw` and the reply's fields | a custom query's `Parse` reads the parsed fields; `termcap.ParseReply(raw)` reads a reply's bytes; `Reply` is no longer comparable |
| `termcap.ReportSchemaVersion` 1 | 2 | see "Changed" |
| `n.Notify(x)`, `n.NotifyContext(ctx, x)` | `termsvctea.Notify(n, x)`, `termsvctea.NotifyContext(ctx, n, x)` | import `termsvc/termsvctea`; a CLI writes `n.Prepare(ctx, x)`'s bytes and calls its `deliver` |
| `n.Update(msg)`, whose `tea.Cmd` was always nil | `termsvctea.Observe(n, msg)`, which returns nothing | or `n.SetCaps(caps)` and `n.SetFocus(focused)` |
| `termsvc.Copy(caps, text)`, `CopyContext(ctx, caps, text)` | `termsvctea.Copy(caps, text)`, `CopyContext(ctx, caps, text)` | a CLI uses `termsvc.PrepareCopy` or `CopySequence` |
| `termsvc.ActivityBeacon()` | `termsvctea.ActivityBeacon()` | the package only |
| `policy.Open(url)` | `termsvctea.Open(policy, url)` | still delivers `OpenURLMsg` |
| `r.Dispatch(ctx, req)`, a `tea.Cmd` | `commandtea.Dispatch(ctx, r, req)` | the core's `r.Dispatch` returns a `command.Effect` |
| `r.Watch()`, `r.WatchContext(ctx)` | `commandtea.Watch(r)`, `commandtea.WatchContext(ctx, r)` | the core's `r.Watch(ctx)` returns an `Effect` |
| `r.Attach(p.Send)`, `command.WithLoop(p.Send)` | `commandtea.Attach(r, p.Send)`, `commandtea.WithLoop(p.Send)` | the core's take a `func(any)` |
| `msg.Run()` of a `command.LoopMsg`, a `tea.Cmd` | `commandtea.RunLoop(msg)` | `msg.Run()` returns an `Effect` |
| `command.Result.Cmd`, a `tea.Cmd` | a `command.Effect` | `Result{Cmd: commandtea.Effect(cmd)}` in a handler |
| `d.Update(msg, path, ctx)` of a `keymap.Dispatcher` | `keymaptea.Update(d, msg, path, ctx)` | import `keymap/keymaptea`; the core's `d.Key(k, path, ctx)` and `d.Expire(t)` |
| `keymap.Outcome.Replay`, `[]tea.KeyPressMsg` | `[]termkey.KeyEvent` | `tea.KeyPressMsg(keymaptea.ToTea(k))` for each |
| `keymap.Settings.Tick` | removed | a test sets `Settings.Timeout`; `keymaptea.Update` starts `tea.Tick` |
| `km.KeyBinding(id, path, d)`, `km.Help(path, d, ids...)` | `keymaptea.KeyBinding(km, id, path, d)`, `keymaptea.Help(km, path, d, ids...)` | `km.Shown(id, path)` is the bindings they show |
| `stroke.BubblesKeys()` | `keymaptea.BubblesKeys(stroke)` | the package only |
| `keymap.Normalize`, `Stroke.Matches`, `Dispatcher.Explain`, `Explanation.Raw` and `Normalized`, over a `tea.Key` | over a `termkey.KeyEvent` | `keymaptea.FromTea(k)` converts, field for field |
| `keymap.Stroke.Mod`, a `tea.KeyMod` | a `termkey.Mod` | the same bit values; `termkey.Mod(m)` and `tea.KeyMod(s.Mod)` convert |

### Changed

- **`termcap.Report`'s JSON** has `"schema_version": 2`, and writes the
  colour profile by its token, `truecolor`, `ansi256`, `ansi`, `ascii`,
  `notty` or `unknown`, where `v1.13.0` wrote `TrueColor`, `ANSI256` and
  the rest. The text report writes the same tokens. Reading `Caps` from
  JSON takes either spelling.

### Added

- **`termkey`:** `KeyEvent`, `Mod`, the key codes with Bubble Tea's values
  and names, and a `Decoder` from the bytes a terminal sends to keys,
  pastes and the sequences it does not decode, for a line-mode CLI.
- **`termcap`:** `Probe`, from `NewProbe`, the probe as an engine that
  does no I/O, with `ProbeEvent`, `ProbeEventKind` and `ProbeHook`;
  `ParseReply`, `ReplyKind` and `ModeSetting`; `ColorProfile`;
  `WithoutModeQuery`; `KeyboardCaps.Request`.
- **`termsvc`:** `Notifier.Prepare`, `SetCaps` and `SetFocus`;
  `PrepareCopy`.
- **`command`:** `Effect`.
- **`keymap`:** `Dispatcher.Key` and `Expire`, `Outcome.Wait` and
  `Timeout`, and `Keymap.Shown`.
- **The adapters:** `termcap/termcaptea`, `termsvc/termsvctea`,
  `command/commandtea` and `keymap/keymaptea`; and
  `termcaptest.Terminal.Reply`.

`command`, `fuzzy`, `glyph`, `keymap`, `layout`, `safetext`, `termcap`,
`termkey`, `termsvc` and `when` link no Charm module but `x/ansi`, and a
test holds them to it, so a line-mode CLI that uses only these builds
without Bubble Tea, Lip Gloss or Bubbles.

## Migrating to `v1.13.0`

`v1.13.0` follows `v0.12.0` directly. It was tagged in place of
`v0.13.0` and kept, and there is no `v0.13.0`
([0020-MADR](../decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
A4). `go get github.com/maccavelli/go-tui-lib@latest` gives it.

`v1.13.0` adds packages and fields, and removes nothing
([0020-PLAN-charm-free-cores-glyphs-and-safetext](../decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md)
Step 6). A program needs no change to build. What it shows changes where
it puts untrusted text on the screen that is not well formed, as
"Changed" lists.

### Changed

- **Invalid UTF-8 shows as U+FFFD,** one for each maximal subpart (the
  Unicode Standard, §3.9.6), where the library's output dropped it: in a
  pane's title and badge, a notification, a window title, a link's text,
  a beacon's vendor and tmux's reply. A raw byte 0x80 to 0x9F that
  continues no character is invalid UTF-8, so it shows as U+FFFD and
  opens no sequence.
- **Malformed escape sequences are read as the DEC parser reads them**
  ([0020-MADR](../decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  A3):
  - a C1 control encoded as UTF-8, U+0080 to U+009F, is its 7-bit form:
    U+009B opens a CSI, U+0090 a DCS, U+009C ends a string, and the
    sequence goes whole, where only the code point was dropped before;
  - BEL ends a DCS, SOS, PM or APC, so the text after it shows;
  - text other than ASCII inside a DCS, SOS, PM or APC is part of it;
  - a character inside an OSC does not end it, even one whose UTF-8 holds
    the byte 0x9C, such as `Ü`;
  - a pane's view splits a malformed sequence as a title always did: an
    nF escape such as `ESC SP X` is removed, and the text after it stays;
  - a pane's view keeps no SGR or OSC 8 link written with a C1 control.

  Well-formed text, styles and links come out as before.

### Added

- **`safetext`:** `Clean` and `IsClean` by `Rules`, with the presets
  `Text`, `Strict`, `Line`, `Command` and `StatusLine`; `Sanitizer`, which
  cleans a stream; `SafeURL` and `URLs`. It imports only the standard
  library and `glyph`, so a line-mode CLI can use it without Bubble Tea.
  `internal/sanitize`, which cleans what the library shows, now runs on
  it.
- **`fuzzy`:** a `Matcher` that matches and ranks by grapheme cluster,
  with grades and fzf's scoring, and an `Index` for large candidate lists.
- **`glyph.Set`:** `Done`, `Failed`, `Running` and `Pending`, each its own
  shape in both sets; `Heading`; `MeterFill` and `MeterTrack`; `Control`;
  `Prompt` and `Mask`.
- **`workspace`:** `Bindings` and `Theme`, so a pane can show a command's
  key and draw as the workspace does; `CommandConsumer` and `CommandMsg`,
  for a pane or an overlay that runs some commands itself; and `Close`,
  an overlay's request to close itself.

`fuzzy`, `glyph`, `layout`, `safetext` and `when` link no Charm module but
`x/ansi`. A test holds them to that, so a program that uses only these
builds without Bubble Tea, Lip Gloss or Bubbles.

## Migrating to `v0.12.0`

`v0.12.0` removes the names `v0.11.0` deprecated
([0007-PLAN-keymap-engine](../decisions/0007-PLAN-keymap-engine.md) Step
12). A program that builds on `v0.11.x` with no SA1019 report needs no
change.

### Removed

| Removed | Use |
| :--- | :--- |
| `workspace.KeyMap` and its `Bindings` | a keymap: `keymap.New` with `workspace.DefaultRules()` as its defaults, and its `Bindings`, or `keymaptea.KeyBinding` from `v2.0.0` |
| `workspace.DefaultKeyMap` | `workspace.DefaultRules()`, the same keys |
| `workspace.WithKeyMap` | `workspace.WithBindings`, and `SetBindings` for a rebuilt keymap |

[The keymaps guide](keymaps.md#moving-from-workspacekeymap) maps each
field to its command, and writes one rebinding both ways.

## Migrating to `v0.11.0`

`v0.11.0` adds `keymap`, and moves the workspace's keys onto it
([0007-MADR](../decisions/0007-MADR-keymap-engine.md)). A program that
sets no key option needs no change: the default keys are the same.

### Deprecated

These still work in `v0.11.x`, and are removed in `v0.12.0`.

| Deprecated | Use |
| :--- | :--- |
| `workspace.KeyMap` | a keymap: `keymap.New` with `workspace.DefaultRules()` as its defaults |
| `workspace.DefaultKeyMap` | `workspace.DefaultRules()`, the same keys |
| `workspace.WithKeyMap` | `workspace.WithBindings`, and `SetBindings` for a rebuilt keymap |

Each field is now a command: `FocusNext` and `FocusPrev` are
`workspace.focus.next` and `.prev`, `FocusPane[i]` is
`workspace.focus.index` with `{"index": i+1}`, `Zoom` is `workspace.zoom`,
the four `Resize…` fields are `workspace.resize.direction` with
`{"direction": "left"}` and the rest, and `Close` is
`workspace.overlay.close` in the `overlay` scope.
[The keymaps guide](keymaps.md#moving-from-workspacekeymap) writes one
rebinding both ways.

### Changed

- **The workspace's help lists a command once.** `alt+1` to `alt+9` are
  one entry, `alt+1 pane`, where the full help listed nine, and the four
  resize keys are one, `alt+shift+left resize`. Under a modal overlay,
  the focus and zoom keys, which never reached through it, are no longer
  listed.
- **A global binding reaches a modal overlay.** A modal overlay still
  takes every key no binding on its path takes, and the layout's keys
  still do not reach through it.
- **`glyph.Set` has `Keys`,** a `KeyGlyphs` of the macOS modifier and
  arrow labels. Their ASCII twins are words, such as `opt+`: the one
  exception to a glyph's single cell.

### Added

- **`keymap`:** notation, layered keymaps with conflicts, chords and a
  leader, VS Code-format user files with a JSON Schema, terminal-aware
  defaults, labels, help, which-key data and a key explanation.
- **`workspace`:** `DefaultRules`, `WithBindings`, `SetBindings`,
  `KeyPath`, `KeyContexter`, `PushMode` and `RequestMsg`; and two
  commands, `workspace.focus.index` and `workspace.resize.direction`.

## Migrating to `v0.10.0`

`v0.10.0` removes the names `v0.9.0` deprecated
([0014-PLAN-canonicalization](../decisions/0014-PLAN-canonicalization.md)
Step 9). A program that builds on `v0.9.x` with no SA1019 report needs no
change, except for the audit log's key.

### Removed

| Removed | Use |
| :--- | :--- |
| `command.Decision` | `command.Verdict` |
| `command.Record.Decision` | `Record.Verdict` |
| `command.Request.Context`, `Invocation.Context` | `WhenContext` |
| `termcap.New` | `termcap.NewProber` |
| `layout.SidebarWidth`, `BottomHeight`, `MainSize`, `BottomSpan`, `Footer`, `Gap`, `Breakpoints`, `NoResponsive` | `layout.WithSidebarWidth`, `WithBottomHeight`, `WithMainSize`, `WithBottomSpan`, `WithFooter`, `WithGap`, `WithBreakpoints`, `WithoutResponsive` |
| `workspace.WithMouse` | `workspace.WithoutMouse()`, or nothing: the mouse is handled by default |
| `termsvc.WithGate` | `termsvc.WithNotifyFilter` |

### Changed

- **`SlogAuditor` logs the verdict under the key `verdict`,** where it was
  `decision`. A query or alert on the old key needs the new one.

## Migrating to `v0.9.0`

`v0.9.0` gives the API one name for each idea
([0014-PLAN-canonicalization](../decisions/0014-PLAN-canonicalization.md)).

### Renamed

The old names still work in `v0.9.x`, and are removed in `v0.10.0`.

| Old | New |
| :--- | :--- |
| `command.Decision` | `command.Verdict`, with the same four constants |
| `command.Record.Decision` | `Record.Verdict`; the registry sets both |
| `command.Request.Context`, `Invocation.Context` | `WhenContext`; the registry reads `WhenContext`, else `Context` |
| `termcap.New` | `termcap.NewProber` |
| `layout.SidebarWidth`, `BottomHeight`, `MainSize`, `BottomSpan`, `Footer`, `Gap`, `Breakpoints` | `layout.WithSidebarWidth`, `WithBottomHeight`, `WithMainSize`, `WithBottomSpan`, `WithFooter`, `WithGap`, `WithBreakpoints` |
| `layout.NoResponsive` | `layout.WithoutResponsive` |
| `workspace.WithMouse(false)` | `workspace.WithoutMouse()` |
| `workspace.WithMouse(true)` | nothing: the mouse is handled by default |
| `termsvc.WithGate` | `termsvc.WithNotifyFilter` |

`SlogAuditor` still logs the verdict under the key `decision` in
`v0.9.x`; the key is `verdict` from `v0.10.0`.

### Removed

These were deprecated in `v0.8.0`.

| Removed | Use |
| :--- | :--- |
| `termcaptest.RunTimeout` | `(*termcaptest.Terminal).SetRunTimeout`, or `DefaultRunTimeout` |
| `workspace.KeyFocusedPane`, `KeyZoomed`, `KeyHiddenPanes`, `KeyOverlay`, `KeyModal`, `KeyWidth`, `KeyHeight` | `workspace.FocusedPaneKey()`, `ZoomedKey()`, `HiddenPanesKey()`, `OverlayKey()`, `ModalKey()`, `WidthKey()`, `HeightKey()` |

### Changed

- **Option types are opaque.** `command.Option` and `RegistryOption`,
  `termcap.Option`, `termsvc.NotifyOption`, and `workspace.Option` and
  `WrapOption` are interfaces that only their package's functions make.
  A program's own `func(*T)` no longer compiles as one; build options with
  the `With…`, `Without…` and `On…` functions.
- **Every exported enum has text forms.** 21 enums gain `MarshalText` and
  `UnmarshalText`, so their JSON is their token, such as `"read-only"`,
  where it was a number. A value with no token prints `Type(N)`, such as
  `Mux(9)`, where some printed the number alone.
- **`termcap`'s JSON uses `encoding/json/v2`.** It no longer escapes `<`,
  `>` and `&`. Reading is stricter: a duplicate name or invalid UTF-8 is
  an error, and a name matches in its own case only. Invalid UTF-8 in a
  program's own `Caps` values is written as U+FFFD.
- **`termcap.ParseTmux` cleans its fields** of escape sequences, controls
  and invalid UTF-8, as every other source of a terminal's text is.
- **`when.Parse`'s errors are `*when.SyntaxError`,** with the offset, the
  message and any error beneath, such as a regex's. Their texts are
  unchanged.
- **`termcap`'s errors for an unknown name wrap `termcap.ErrUnknownName`,**
  with their texts unchanged.
- **`command`'s raw JSON is spelled `jsontext.Value`,** which is the type
  `json.RawMessage` names, so no code changes.

### Added

- `command.AuditorFunc`, `termsvc.BackendFunc` and `termsvc.ClipboardFunc`,
  to use a function as a hook.
- `when.SyntaxError` and `termcap.ErrUnknownName`.
