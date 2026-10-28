---
status: in-progress
date: 2026-10-27
associated-madr: "0023-MADR-line-mode-components.md"
---
# Implement the line-mode question, picker, composer and transcript

Associated MADR: [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)

## Goal

A line-mode CLI that links no Charm module but `x/ansi` asks an inline
question, picks from a filtered list, edits and submits a line, and
streams a reply's Markdown into a transcript that reprints on a resize;
the TUI asks the same question in an overlay, edits in the same composer,
and picks from the same entries in the palette. Every component is an
engine that does no I/O, every default is listed with its control
(0023-MADR D13), and every string it shows comes from the program's
catalogue.

Done means every item under Verification holds, CI is green on the pushed
tree, each of the four releases is tagged by the owner, and each release's
consumer smoke test passes.

## Scope

### Facts re-read before execution

Step 1 re-reads, at the tree it starts on, each fact 0023-MADR's Context
cites in this repository (`termkey`, `termcap.ParseReply`, `keymap`'s
`Dispatcher`, `fuzzy`, `palette`, `internal/conformance`), and records a
change as a deviation before any code. gobble's facts are its records' and
code's at `5fb6928`; a step that relies on one cites it.

### In scope

| Step | Paths | What | Release |
| :--- | :--- | :--- | :--- |
| 1 | `inline/`, `inline/inlinetest/`, `termcap/reply.go`, conformance | the live region, the reader, the test screen, `ReplyCursorPosition` | |
| 2 | `ask/`, `ask/asktea/` | the question, and its overlay | the next minor |
| 3 | `pick/`, `palette/entries.go`, conformance | the picker, and `palette.FromEntries` (A3) | the next minor |
| 4 | `lineedit/`, `lineedit/lineedittea/` | the composer, and its pane | the next minor |
| 5 | `stream/`, `stream/streamtea/`, `docs/decisions/0009-*` | the Charm-free core, `Printer`, `Markup`; 0009's amendment | |
| 6 | `transcript/` | the bounded, reprintable conversation | the next minor |
| 7 | `docs/`, `README.md` | the guide, the close-out | |

Each step's package joins `cores` and the `charmfree` files in the commit
that builds it; the `<pkg>tea` packages are not cores. Each step adds its
defaults' rows to `docs/guides/defaults.md`, and its message IDs and
English defaults, through `msgcat`.

### Out of scope

* What 0023-MADR D10 leaves out: a line-mode multi-select, mouse input in
  the line mode, win32-input-mode, undo and a kill ring, a `picktea`.
* `stream/glamourmd`, 0009's glamour adapter, which 0009's own PLAN builds.
* `git push` and tags, which are the owner's.

## Prerequisites

* [0024-PLAN-command-and-control-opt-outs.md](0024-PLAN-command-and-control-opt-outs.md)
  is `complete`: rule 12, `msgcat` and the defaults inventory exist, and
  `v2.2.0` is tagged.
* The owner has approved this PLAN.

## Rules for every step

0024-PLAN's rules 1 to 8, and:

1. **Engines are functions of their inputs and the time.** No step adds a
   goroutine, a clock read or an environment read to a core, except
   `inline.Reader`, whose one goroutine is 0023-MADR D3's.
2. **Goldens on `inlinetest.Screen`,** across {colour, no colour} ×
   {UTF-8, ASCII} at widths 60 and 100, each line within its width under
   the method, read before they are committed.
3. **Every default has its row** in `docs/guides/defaults.md`, and every
   string its message ID, in the step that adds it.
4. **A release step** runs 0008-PLAN Step 9's procedure: the API listed by
   apidiff, release notes, the disclosure guard, the owner's push and tag,
   CI green on both, and a consumer smoke test from the proxy in a scratch
   module whose closure, listed for Linux, macOS and Windows, holds no
   Charm module but `x/ansi`, with a negative control seen to fail.

## Implementation Steps

### Step 1: `inline`, `inlinetest` and the cursor report

* `termcap`: `ReplyCursorPosition`, with `Reply.Row` and `Col`, read from
  `CSI row ; col R` and DECXCPR's `CSI ? row ; col R`.
* `inline.Region` (0023-MADR D3): `Draw`, `Print`, `Commit`, `Clear`,
  `Resize`, `HandleInput`, `Timeout`, `HandleTimeout`, `Restore`; options
  `WithMethod`, `WithSynchronized`, `WithReflow`, `WithoutCursorQuery`,
  `WithHeight`; `Reflow` and its tokens from `internal/enum`.
* `inline.Reader`: `NewReader(io.Reader)`, `Read(ctx, p)`: one read in
  flight at most, never closed.
* `inlinetest.Screen`: gobble's `termtest.Screen` rebuilt on `x/ansi`
  widths, by grapheme cluster, with `Resize(columns, Reflow)` and an
  answered cursor report; its text form for goldens.
* **Tests:**
  * a diff redraw writes only changed rows; a one-key change is at most
    the changed text plus 40 bytes;
  * `Print` leaves the text above and the region whole below it, on the
    screen;
  * each re-wrap kind, with a report and without, and keys typed while a
    report is awaited, which reach the component in order; a row-1 report
    (`CSI 1 ; 5 R`) taken as a report while awaited, and as `ctrl+f3`
    otherwise;
  * `WithoutCursorQuery` sends no `CSI 6 n`;
  * `Reader` holds one read in flight, and no file of `inline` calls
    `Close` on it (a scan, as gobble's `TestNoCloseOnStdin`);
  * fuzz: any input and resize stream never panics, and no drawn row is
    wider than the width.
* **Mutations:** the diff rewrites every row; `Print` does not redraw the
  region; the cut kind moves by the re-wrapped rows; the awaited report is
  passed on as a key; `Reader` starts a second read; the budget is
  exceeded by an erase before each draw.

### Step 2: `ask` and `ask/asktea` (issue #7), and a release

* `ask`: `Question`, `Row`, `Answer`, `Response`, `Layout`, `Asker`,
  `AskEvent`, `AskEventKind`, as 0023-MADR D4; the reason read through a
  one-row composer, bounded at 2048 bytes by default (`WithReasonLimit`).
* `ask/asktea`: an overlay pane drawing `Layout` at its width and driving
  the same `Asker`.
* **Tests:** `Layout` goldens of gobble's permission question
  (0029-MADR D1's sketch) at 60, 100 and 177, and of the trust, plan-mode,
  grant and tool-call questions; an unknown key ignored; the cancel key,
  ctrl+c and the end of input cancel; a reason echoed, edited and bounded;
  the answer committed in words; fuzz: `Layout` never splits an answer
  and never writes a row wider than the width.
* **Mutations:** a row breaks inside an answer; a continuation is not
  indented; an unknown key answers; the echo is the label, not `Echo`.
* **The release,** under rule 4, with gobble's issue #7 told where the
  question is.

### Step 3: `pick` and `palette.FromEntries` (issue #8), and a release

*Amended 2026-10-27, before the step began, to ground it on 0023-MADR A3.
As first written, the step named `PickEvent` and `PickEventKind`
(renamed by A2 item 6), the keys up, down, page up, page down, enter and
esc, and three mutations; it said nothing of the layout, the count, the
height, the line left in the scrollback, the index's count, the
messages, the rules or the region.*

**Before any code.**

1. *Done 2026-10-27:* the owner answered A3's Q8 to Q11, each
   "(Recommended)", and A3 is accepted: the height is 10 rows, and the
   keys are `pick`'s own commands in the program's keymap.
2. A3.1's facts about this repository are re-read at the tree the step
   starts on: `go doc ./fuzzy` (`Rank`, `NewMatcher`, `Index`,
   `Candidates`, `DefaultMaxCandidates`), `palette/engine.go`'s `before`
   and `insert`, `palette/static.go`, `palette/view.go`'s `minText` and
   `drawRow`, `palette/keys.go`, and `go doc ./keymap Dispatcher`. A
   change is a deviation, raised before code.
3. `python3 -m tools.verify names` over `pick`'s types (`Entry`,
   `Chooser`, `ChooserEvent`, `ChooserEventKind`, `Option`) reports clean.

**`pick/`, a core** (`inline`, `internal/lineinput`, `fuzzy`, `glyph`,
`keymap`, `msgcat`, `safetext`, `termkey`, `internal/enum`,
`internal/sanitize`; `x/ansi`):

| File | Holds |
| :--- | :--- |
| `doc.go` | the package documentation: the layout, the keys, the host loop, the line left behind; its stability line |
| `entry.go` | `Entry{ID, Text, Detail}`: `Text` matched and drawn, `Detail` drawn and not matched; IDs the caller's, distinct where `palette.FromEntries` takes the same list |
| `event.go` | `ChooserEventKind`, `PickNone`, `PickWrite`, `PickDone`, tokens `none`, `write`, `done`, from `internal/enum`; `ChooserEvent{Kind, Data, Entry, Canceled}` |
| `options.go` | `Option`, opaque; `WithHeight` and `DefaultHeight` (Q8), `WithPrompt`, `WithQuery`, `WithMethod`, `WithGlyphs`, `WithIndexAbove` and `DefaultIndexAbove` (5000), `WithRegion`, `WithCatalog`, `WithRules`, and, by Q9, `WithKeymap` |
| `keys.go` | by Q9: `SelectNextID`, `SelectPrevID`, `PageNextID`, `PagePrevID`, `AcceptID`, `CancelID`, `ChooserScope` (`kind:chooser`) and `DefaultRules()` |
| `messages.go` | `MsgCount` (`%[1]d of %[2]d`), `MsgCountMore` (`%[1]d of %[2]d+`), `MsgNoResults` (`no results`), `MsgCanceled` (`cancel`); `Messages()` |
| `chooser.go` | `NewChooser(entries, …)`, `Resize`, `HandleInput`, `Timeout`, `HandleTimeout`, `Cancel`, `NextEvent` |

**What the `Chooser` does,** each item proved by the test named after it:

* **Cleaning:** each entry's `Text` and `Detail` is cleaned once, at
  `NewChooser`, by `WithRules`' rules and then `internal/sanitize.Line`,
  the floor; the `ChooserEvent` carries the caller's `Entry` as given
  (`TestCleaned`, `TestRules`).
* **The layout** (A3.4 item 1): the query row, `WithPrompt` (default
  `glyph.Set.Prompt`), a space, the query's window from
  `lineinput.Line.View`, and the count right-aligned, dropped when the
  query would keep fewer than 8 cells; then `min(height, len(entries))`
  rows, each `glyph.Set.Focus` on the selected row and its width in
  spaces on the others, a space, the `Text` cut by
  `internal/sanitize.Truncate` with `glyph.Set.Ellipsis`, and the `Detail`
  right-aligned after at least one space, dropped when the `Text` would
  keep fewer than 8 cells. Rows past the matches are blank. The cursor
  is on the query row (`TestLayoutGolden`, `TestDetail`, `TestHeight`).
* **Ranking** (A3.4 items 4, 5): each change of the query compiles one
  `fuzzy.NewMatcher` and ranks with `Rank`, limit 0, over `Text`; above
  `WithIndexAbove` entries the candidates come from one `fuzzy.Index`
  built at `NewChooser`. The count is the selected row's place and the
  matches', `MsgCountMore` when the index yielded
  `fuzzy.DefaultMaxCandidates`, and `MsgNoResults` for none
  (`TestRankLikeMatcher`, `TestEmptyQueryOrder`, `TestIndex`,
  `TestCount`). *Amended by D11, 2026-10-27:* `Rank` breaks a tie by the
  shorter key, so the `Chooser` scores each candidate with
  `Matcher.Score` and sorts the matches stably by grade and value, ties
  in the input order, and `fuzzy.Rank`'s documentation says its own tie
  order (0023-MADR A4).
* **Keys** (A3.4 item 3, Q9): through a `keymap.Dispatcher` over
  `WithKeymap`'s keymap, or one built from `DefaultRules()` alone, on the
  path `[ChooserScope]`: up and ctrl+p, down and ctrl+n, a row, clamped,
  with no wrap; pgup and pgdown, a page of the visible rows; enter
  accepts the selected entry; esc and ctrl+g cancel. A pending
  sequence's `Wait` is folded into `Timeout`, and `HandleTimeout` gives
  the dispatcher its `Expire`. ctrl+c cancels before the dispatcher, as
  in `ask`. A key the dispatcher does not take goes to the query's
  `lineinput.Line`, and a paste is inserted there. The query's bound is
  1024 bytes. The selection returns to the first row when the query
  changes, and the window keeps it on the screen (`TestMoves`,
  `TestKeymap`, `TestCancel`, `TestQueryEdit`).
* **The end** (A3.4 item 6): enter with a match commits one line, the
  prompt, a space and the entry's cleaned `Text`, and sends `PickDone`
  with the caller's `Entry`; enter with no match does nothing; a cancel
  commits the prompt and `MsgCanceled`, and sends `PickDone` with
  `Canceled`. Nothing is taken after (`TestAccept`, `TestAcceptNoMatch`,
  `TestCancel`).
* **The engine contract** (D2, A2 items 2, 3): `Resize` draws at once the
  first time and through the region's cursor query after; the region is
  `WithRegion`'s or its own, by `inline.NewRegion` and `WithMethod`'s
  method; a cursor report awaited is the region's, never a key
  (`TestResize`, `TestWithRegion`, `TestRow1Report`).
* **Words:** every string through `WithCatalog` and `Messages`
  (`TestCatalog`).

**`palette/entries.go`:** `FromEntries(name string, entries []pick.Entry)
Provider`, `Static`'s form: `Name` is name; `Search` scores each entry's
`Text` with the search's matcher, in order; `Discover` lists every entry,
first best; each hit is `Hit{ID, Text, Hint: Detail, Score}` with no
`Run`, so accepting sends `SelectedMsg` (`TestFromEntries`,
`TestFromEntriesRankedLikeChooser`: for each query of a fixture of 40
entries with distinct IDs, the palette's hit IDs, through a picker on a
workspace in a `synctest` bubble, equal the `Chooser`'s ranked IDs).

**The gates and the docs:** `pick` joins `cores` and the `charmfree`
files; `pick.ChooserEventKind` joins the enum table; the defaults guide
gains a `pick` table, a row for each option, the keys (`built`, by Q9) and
ctrl+c (`decided`, D4's rule kept), and a `palette` row for
`FromEntries`; the glossary moves `Chooser`, `Entry`, `ChooserEvent` and
`ChooserEventKind` to "Built".

**Tests beyond those named above:**

* `TestLayoutGolden` on `inlinetest.Screen`, across the matrix at 60 and
  100, each line within its width under the method: a models list of 40
  entries with context and prices in `Detail`, and a sessions list by
  title and time, each with an empty query, `son`, a query with no
  match, and the index on (`WithIndexAbove(10)`): two lists, four
  queries and the matrix's eight cases, 64 files, read before they are
  kept.
* `FuzzChooser`: any key, paste and resize stream never panics, no drawn
  row is wider than the width but a row of one cluster wider than it, and
  the selection is always a drawn row.
* `BenchmarkKeystroke`, one typed key and its drawing, at 5000 entries,
  a full scan, and at 100,000, through the index; its figures are recorded
  in the execution record, as the baseline Step 7's guide cites.

**Mutations,** each killed by a named test, its failure quoted in the
record: the index is never built (`TestIndex`); enter commits the query,
not the entry (`TestAccept`); `FromEntries` matches `Detail`
(`TestFromEntries`); the `Chooser` matches `Detail` (`TestDetail`); ties
broken by length, not input order (`TestRankLikeMatcher`); the selection
kept on a query change (`TestMoves`); esc does not cancel (`TestCancel`);
`WithKeymap` ignored (`TestKeymap`); the count ignores the index's cap
(`TestCount`); the list's height follows the matches (`TestHeight`);
`WithRules` replaces the floor (`TestRules`); `pick` imports Bubble Tea
(`TestCoresAreCharmFree`).

**The release, `v2.4.0`,** under rule 4: the API listed by `make
apicheck` against `v2.3.0`, 0 incompatible changes; "Migrating to
`v2.4.0`", the README's Status and `docs/architecture.md`; the gates on
macOS, Linux and Windows; the disclosure guard; the owner's push, CI, tag
and the tag's CI; the consumer smoke test from the proxy in a scratch
module, which picks from the models list on an `inlinetest.Screen`,
types `son`, accepts, checks the `Entry` and the line left, lists the
closure for Linux, macOS and Windows (`x/ansi` alone of Charm), and runs
two negative controls, each seen failing: the program expecting another
entry, and the closure check over a program that imports `palette`. Issue
#8 is told, for the owner to close.

### Step 4: `lineedit` and `lineedit/lineedittea` (issue #9), and a release

* `lineedit`: `Composer`, `History`, `MemHistory`, `Completer`,
  `CompleterFunc`, `EditEvent`, `EditEventKind` (named `ComposerEvent` and
  `ComposerEventKind` by A2 item 6), as 0023-MADR D6;
  `DefaultRules()` in `kind:composer`, gobble's bindings; `PasteInsert`
  and `PasteSubmit`.
* `lineedit/lineedittea`: a workspace pane over the same `Composer`.
* **Tests:**
  * every binding gobble's editor has (0023-MADR's list), and a parity
    test against x/term's `Terminal` over gobble's 55 keys
    (`buffer_test.go:210-366`);
  * a grapheme cluster, a ZWJ sequence and a flag are one cursor step and
    their true width;
  * whole-word layout, a wider word left whole, continuation under the
    prompt;
  * history up and down, by visual row first; `MemHistory` drops
    consecutive duplicates, bounds at 1000, and round-trips its file
    form;
  * `EditExternal` only `WithExternalEditor`; `SetText` after it;
  * a user binding in the keymap rebinds a key;
  * the budget per key, with a benchmark and its baseline recorded;
  * fuzz: any key stream never panics, and the buffer is valid UTF-8.
* **Mutations:** a word motion stops inside a cluster; up recalls history
  from a middle row; a duplicate is kept; a paste submits under
  `PasteInsert`; the external editor's event is sent without its option.
* **By 0023-MADR A3 (items 9 to 12),** added 2026-10-27: alt+f, alt+b,
  alt+d and alt+backspace by Readline's words, a cluster's first rune a
  letter or a digit (`TestWords`, with `can't`, `32.3`, a CJK run and an
  accented word), and ctrl+w to whitespace (`TestCtrlW`); alt+enter
  inserts a newline, beside Shift+Enter and the trailing `\`
  (`TestNewline`); a paste is cleaned by `safetext` with newlines and
  tabs kept, an `ESC [ 2 0 1 ~` inside its text removed, and no
  completion run on it (`TestPaste`); `MemHistory` drops a line equal to
  the previous one and an empty one, and writes `\n` and `\\`
  (`TestMemHistory`, `FuzzMemHistory`). The `History` interface's methods
  and their doc are x/term v0.47.0's, quoted in A3.2.
* **By 0023-MADR A5,** added 2026-10-27, by D13 to D19 below: a word
  wider than its row broken at the width; a forward word motion to the
  word's end; each row measured as it is drawn (D19); the parity
  test compares with a recorded reference of x/term v0.47.0, written by a
  generator under `lineedit/testdata/`; ctrl+l clears only after
  `WithClearScreen()`; ctrl+x ctrl+e sends `EditExternal` only with
  `WithExternalEditor`; and `ask`'s reason and `pick`'s query move to
  one-row `Composer`s, `ask` gaining `WithKeymap`.
* **The release,** under rule 4.

### Step 5: `stream`'s core, `Printer` and `Markup`

* `docs/decisions/0009-MADR-streaming-content-engine.md` gains its
  amendment, as 0023-MADR D7 says, and its PLAN is amended to build on
  this step.
* `stream`: `Scanner`, the block scanner of 0009 §4; `Printer`, the
  line-stable printer of 0023-MADR D7; `Markup` and `Style`. `Doc`,
  `Renderer`, `RendererFunc` and `Plain` are 0009's PLAN's; this step
  builds what the line mode needs, and leaves `Doc` to that PLAN, which
  builds it on the same `Scanner`.
* **Tests:** gobble's styling and wrapping cases (`render/mdstyle.go`,
  `render/wrap.go`, their tests) as goldens; the zero `Markup` gives the
  input byte for byte; a table held until a blank line; a fence's lines
  printed as they complete; the output the same for any split of the
  input (property test); fuzz: never panics, never a row wider than the
  width outside a code or table row.
* **Mutations:** a table row printed before the table ends; a marker
  dropped by the styler; a row starting with a word that opens a block;
  the output depends on the split.

* **By 0023-MADR A3 (items 7, 13, 14, 16),** added 2026-10-27: the
  owner answered Q10 "Hold one line (Recommended)": the `Printer` holds a
  paragraph line until the next line or the end, tested on every example of CommonMark 0.31.2 §4.3 (setext
  headings) and of GFM 0.29-gfm §4.10 (tables), their numbers read from
  the specifications at the step's start; an inline delimiter still open when
  its line prints is printed as it is, and no printed line is styled
  again (`TestInlineOpen`); a reference link is styled only when its
  definition came first (`TestReferenceLater`); a CJK run is one word,
  left whole (`TestWrapCJK`, the limit UAX #14 §3.1 names). `pick` gains
  `WithMarkup`, its matched cells underlined, and a golden of it.

### Step 6: `transcript` (issue #10), and a release

* `transcript`: `Transcript` and its options, as 0023-MADR D9, with
  `WithoutScrollbackClear` (D13).
* **Tests:** the bound drops the oldest pieces; a reprint draws the newest
  back to the row limit, after the notice when it cut; the settle waits
  for the last resize and for the host's idle; `WithoutScrollbackClear`
  writes `CSI 2 J` without `CSI 3 J`; a 10,000-row reprint benchmark, its
  time recorded beside gobble's (0030-MADR).
* **Mutations:** the bound keeps the oldest; the reprint does not wait to
  settle; the notice is written when nothing was cut.
* **By 0023-MADR A3 (item 15),** added 2026-10-27: the owner answered
  Q11 "On, WithoutScrollbackClear (Recommended)", D13 as accepted; the
  reprint writes `SGR 0` first, and `CSI 2 J` before `CSI
  3 J` (`TestReprintOrder`).
* **The release,** under rule 4; issue #10 told.

### Step 7: documentation and the close-out

* `docs/guides/line-mode.md`: the host loop (read, decode, feed, drain,
  write, arm the deadline), raw mode as the program's, the reader, the
  question, the picker, the composer and the transcript, with examples
  compiled and run (`ExampleAsker`, `ExampleChooser`, `ExampleComposer`,
  `ExamplePrinter`).
* `docs/architecture.md`, `docs/glossary.md`, `docs/README.md`, README's
  Status; issues #7 to #10 told, for the owner to close.
* This PLAN becomes `complete`, and its row says so.

## Verification

* Every step's mutations are killed, each failure quoted in the record.
* Each step from 3 re-reads, at the tree it starts on, the repository
  facts 0023-MADR A3 cites for it, and records a change as a deviation
  before any code.
* `python3 -m tools.verify gates` passes on macOS, Linux and Windows for
  every step.
* `TestCoresAreCharmFree` passes with `inline`, `inlinetest`, `ask`,
  `pick`, `lineedit`, `stream` and `transcript`, and fails on a scratch
  copy that imports Bubble Tea into one.
* `TestDefaultsInventory` passes with every component's rows.
* `make apicheck` reports 0 incompatible changes at each release.
* Each release's smoke test passes from the proxy, its closure clean.
* The identifier scan finds nothing.

## Rollout and Rollback

* **Rollout.** Four releases, after Steps 2, 3, 4 and 6, in gobble's
  order; gobble takes each as it lands.
* **Rollback.** Before a push, each step is one local commit. After it, a
  patch release fixes forward. Every change is an addition.

## Execution Record

*2026-10-27.* The owner approved this PLAN: "approved to proceed", after
[0024-PLAN-command-and-control-opt-outs.md](0024-PLAN-command-and-control-opt-outs.md)
was released as `v2.2.0` and closed out (`9a13872`).

### Step 1: `inline`, `inlinetest` and the cursor report

* **Deviation D1, 2026-10-27: the facts re-read changed only by this
  PLAN's prerequisite.**
  * **Found,** re-reading the MADR's facts at `9a13872`: `git diff
    65d437b` shows no change to `termkey`, `keymap` or `fuzzy`, and none
    to `termcap` outside its tests; `palette/provider.go:87`'s `Hit.Run`
    is still a `tea.Cmd`. A scratch program over `termkey` and `termcap`
    gives the "Probed here" table's four results again. What changed is
    0024-PLAN's: `msgcat` joined `TestCoresAreCharmFree`'s cores,
    `palette`'s command IDs and `Opener` are exported, and API convention
    12 and `docs/guides/defaults.md` exist.
  * **Resolutions offered:** record it here, with no change to a decision
    (recommended); or also amend the MADR's Context.
  * **Chosen by the owner,** 2026-10-27: "Record as D1, no change
    (Recommended)".
* **Deviation D2, 2026-10-27: the report's wait gets a control.**
  * **Found:** MADR D3 has the region wait "by default 150 ms" for the
    cursor report, and this step's option list gives no control, which
    convention 12 asks of a timeout.
  * **Resolutions offered:** `inline.WithReportWait` (recommended); or a
    `decided` row with no control.
  * **Chosen by the owner,** 2026-10-27: "Add WithReportWait
    (Recommended)". MADR A1 item 1. This step's scope gains the option,
    its test and a mutation.
* **Deviation D3, 2026-10-27: the height follows the terminal.**
  * **Found:** D3 windows a region taller than its height, this step
    gives `WithHeight` alone, and `Resize` takes the width alone.
  * **Resolutions offered:** `Region.SetHeight` beside `WithHeight`
    (recommended); or `WithHeight` only, a new region on a height change.
  * **Chosen by the owner,** 2026-10-27: "Add Region.SetHeight
    (Recommended)". MADR A1 item 2.
* **Deviation D4, 2026-10-27: `inlinetest` imports `inline`.**
  * **Found:** D1 has `inlinetest` import `x/ansi` alone, while this step
    gives `Screen` a `Resize(columns, Reflow)`, and `Reflow` is `inline`'s;
    a second exported `Reflow` would fail conformance's names check, and no
    other name is reserved.
  * **Resolutions offered:** `inlinetest` imports `inline` (recommended);
    or an `inlinetest.Narrowing` of its own.
  * **Chosen by the owner,** 2026-10-27: "inlinetest imports inline
    (Recommended)". MADR A1 item 3.
* **`termcap/reply.go`:** `ReplyCursorPosition`, the last kind, token
  `cursor-position`, with `Reply.Row` and `Reply.Col`, each from 1, read
  from `CSI row ; col R` and DECXCPR's `CSI ? row ; col R`, with exactly
  two parameters, each at least 1. `ParseReply`'s documentation says the
  report on the first row reads as a modified F3 to a decoder. The probe
  ignores the kind, as it ignores every kind it does not ask for.
* **`inline/`:**
  * `Region` (`region.go`): `NewRegion`, `Draw`, `Print`, `Commit`,
    `Clear`, `Resize`, `HandleInput`, `Timeout`, `HandleTimeout`,
    `Restore`, and `SetHeight` (D3). Drawing is gobble's renderer: an
    unchanged row skipped, a changed row rewritten from what it shares
    with the row before, by relative moves, never erased first, each
    drawing one synchronized update with the cursor hidden. Two things it
    adds to gobble's: the shared part is cut between whole escape
    sequences and grapheme clusters, under the width method, never inside
    one; and the escape sequences in it are written again before the
    changed text, with a reset after a styled row, so a rewrite keeps the
    styles in force there. A row wider than the width is cut to it, by
    `internal/sanitize.Truncate`.
  * The resize is gobble's `lift`: the report's column decides cut or
    re-wrap when the cursor was past the new width, and `WithReflow`'s
    kind when it does not; a keep-row terminal is redrawn as a re-wrap.
    While the report is awaited, `Draw` keeps its rows and `Print` its
    text, and the report's redraw writes both; every input but a cursor
    report goes on to the component, in order. A report that comes after
    its wait ended is taken and dropped, until another input comes, so a
    slow terminal's report is not a key.
  * Options (`options.go`): `WithMethod`, `WithSynchronized`,
    `WithReflow`, `WithoutCursorQuery`, `WithReportWait` (D2) and
    `WithHeight`; `DefaultReportWait`, 150 ms. `Option` is opaque.
  * `Reflow` (`reflow.go`), its tokens from `internal/enum`: `unknown`,
    `down`, `keep-row`, `cut`.
  * `Reader` (`reader.go`): `NewReader(io.Reader)` and `Read(ctx, p)`. Its
    goroutine starts at the first `Read`, and makes one read of the input
    for each `Read` that finds nothing kept, with a buffer the size of that
    `Read`'s, where gobble's reads 512 bytes: the size is the caller's, and
    needs no option. One `Read` waits at a time; `ctx` ends the wait, not
    the read. Nothing in `inline` calls `Close`.
* **`inline/inlinetest/`:** `Screen`, gobble's `termtest.Screen` rebuilt on
  `x/ansi`'s `DecodeSequenceInString` under the screen's width method, a
  cell per grapheme cluster, a wide one two cells and never split by a
  wrap or a re-wrap (*corrected by Step 3's D12:* a cluster takes as many
  cells as it is wide, a ZWJ sequence under `ansi.WcWidth` four or six, and
  `Resize`, a cluster split and an erase each read it so); `NewScreen(columns, rows, method)`; pending wrap; CSI
  A, B, C, D, H, J, 2 J, 3 J, K and 2 K; DECTCEM and mode 2026, as
  `Hidden` and `Synchronized`; `Answer` for `CSI 6 n`; `Cooked`;
  `Resize(columns, inline.Reflow)`, by D4; a scrollback, which a scroll
  and a keep-row re-wrap push rows into; `Lines`, `Scrollback`, and
  `Text`, the form a golden holds. A sequence cut between two writes
  waits for the rest.
* **The gates:** `inline` and `inline/inlinetest` join `cores` and the
  `charmfree` files; `internal/conformance`'s enum table gains
  `inline.Reflow`.
* **`docs/guides/defaults.md`:** an `inline` table of nine rows: the
  method, the synchronized drawing, the cursor query, the wait, the
  re-wrap kind and the height, `built`; a row wider than the width and the
  cursor report, `floor`; reading the input, `decided` by the MADR's D3.
  `inline` shows no text of its own, so it has no message IDs.
* **`docs/glossary.md`:** `inline`'s `Region`, `Reflow` and `Reader`, and
  `inlinetest`'s `Screen`, move from "Reserved" to "Built".
* **Tests:**
  * `termcap`'s `TestParseReply` gains eight cases: a report, in 7-bit
    and 8-bit CSI, the first row's, DECXCPR's, and four that are not
    reports (one parameter, three, row 0, none).
  * `inlinetest`: the writes, the line feed raw and cooked and the
    scrollback, wide and ZWJ clusters, the cursor report at a pending
    wrap, a sequence across two writes, and the three re-wrap kinds.
  * `inline`, on a `Screen` through a host loop: `TestDrawDiff` (only the
    changed cell written, a shorter region erased, an unchanged draw
    writes nothing); `TestDrawBudget`, gobble's three cases, at most the
    changed text plus 40 bytes; a styled row and a ZWJ cluster rewritten
    from the right place; `TestPrint`; `TestCommit` and its clear;
    `TestResize`, each re-wrap kind by the report, by a timeout and with
    no query, and `TestResizeRowsAbove`; `TestKeysWhileAwaited`;
    `TestRow1Report`; `TestLateReport`; `TestWithoutCursorQuery`;
    `TestReportWait`; `TestHeight`; `TestUnsynchronized`; `TestWiderRow`;
    `TestReader`; and `TestNoClose`, a scan of `inline`'s files for any
    `Close` call.
  * `FuzzRegion`: draws, prints, commits, clears, resizes, inputs,
    timeouts and heights never panic, and no drawn row is wider than the
    width. Its first 20 s found a property stated wrong: while a resize
    waits for its report, the rows drawn at the old width stand until the
    redraw; the check skips that state, and the input is kept as a seed
    (`inline/testdata/fuzz/FuzzRegion/3af0be9c1f8f2983`). Then 60 s, 1.28
    million runs, clean.
  * Three tests failed on their own first expectations, and were
    corrected: the cursor report after a raw line feed (`3;4`, not
    `3;6`), the re-wrapped screen too short to keep its top row, and the
    budget test's layout, which first cut rows every 80 bytes where
    gobble's lays out whole words, and placed the old cursor at the end
    where gobble's places it at the edit.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 12 killed of 12. Mutation 1.4's first form did not compile
  (`rep` unused), so it was rewritten and run again, and killed by a test:

  | Mutation | Killed by |
  | :--- | :--- |
  | the diff rewrites every row | `TestDrawDiff`: `an unchanged draw wrote "\x1b[?2026h\x1b[?25l\x1b[1A\r\x1b[5C\n\r\x1b[4C\r\x1b[4C…"`; `TestDrawBudget`: `59 bytes for 1 changed, more than 41` |
  | `Print` does not redraw the region | `TestPrint`: the screen `["above" "> hel" "reply one" "reply two" …]`, want `["above" "reply one" "reply two" "> hel" "  lo"]` |
  | the cut kind moves by the re-wrapped rows | `TestResize` (cut, by a timeout and with no query) and `TestResizeRowsAbove/cut`: `the screen and scrollback are …` |
  | the awaited report is passed on as a key | `TestRow1Report`: `an awaited report on row 1 reached the component as [{key ctrl+f3 …}]`, and `TestResize`, `TestKeysWhileAwaited` |
  | `Reader` starts a second read | `TestReader`: `a read of the input started before a Read asked for it` |
  | an erase before each draw | `TestDrawBudget`: `270 bytes for 1 changed, more than 41`; `TestDrawDiff` |
  | `WithReportWait` ignored | `TestReportWait`: `the wait ends after 150ms, true; want 300ms` |
  | `SetHeight` ignored | `TestHeight`: `height 2: the screen is …` |
  | the reader closes the input at its end | `TestNoClose`: `reader.go calls Close: the input is never closed` |
  | `WithoutCursorQuery` ignored | `TestResize` (no query): `the resize wrote "\x1b[6n"` |
  | DECXCPR's report not read | `TestParseReply`: `DECXCPR's report: ParseReply("\x1b[?12;40R") = {Kind:unknown …}` |
  | `inline` imports Bubble Tea | `TestCoresAreCharmFree`: `inline (GOOS=linux) links charm.land/bubbletea/v2: inline → charm.land/bubbletea/v2`, and `inline/inlinetest` through `inline` |

* **Names:** no type name is new beyond D3's four, which the glossary
  reserved and `tools.verify names` checked when the MADR was written;
  `Reflow` has one meaning, by D4.
* **Gates:** `python3 -m tools.verify gates` over the step's 14 Go files,
  `.golangci.yml`, the guide, the glossary, the index and 2 records. The
  first run failed one gate on every host, `modernize`: `go fix` asked for
  `for c := range row` in place of a counted loop in `inlinetest`'s
  `Screen.Resize`, whose body never changes `c`; the loop was changed, and
  every gate run again. macOS 19 passed, 0 failed, in 638 s,
  `release-check` "335 file(s) clean in 1 module(s)", with `apicheck`
  against `v2.2.0` reporting no incompatible change and the examples gate
  passing (its count is the tracked files, so `inline`'s 10 untracked
  files are outside it, but in its lint, vet and tests of `./...`, and in
  the per-file gates); `remote linux` 18 passed, 0 failed, 1 skipped
  (`gofmt`, which is given no files there), in 1128 s, "345 file(s)
  clean", `inline`'s among them; `remote windows` 18 passed, 0 failed,
  1 skipped, in 2256 s, "345 file(s) clean". `make pre-add-check` on the
  14 Go files is clean. The identifier scan of the change, untracked files
  included, is clean.

### Step 2: `ask` and `ask/asktea` (issue #7), and a release

*2026-10-27.* The owner committed Step 1 (`c843891`) and said
"committed, proceed".

* **Deviation D5, 2026-10-27: `ask` needs a line editor before
  `lineedit` is built.**
  * **Found,** before any code: the MADR's D1 has `ask` and `pick` import
    `lineedit`, and D4 reads a reason "through a one-row
    `lineedit.Composer`", while D12 and this PLAN build `lineedit` in
    Step 4, after `ask`'s release (Step 2) and `pick`'s (Step 3). The
    conflict is in the record as written; Step 1 did not cause it.
  * **Resolutions offered:** an unexported one-row editor under
    `internal/`, used by `ask` now and `pick` in Step 3, on which Step 4
    builds `lineedit`, the two moving to the `Composer` with no exported
    change (recommended; until Step 4 the reason's keys are fixed, not the
    keymap's); building `lineedit` first, moving every later step and
    release and gobble's order; or releasing a partial
    `lineedit.Composer` in this step, its shape frozen in `v2` before its
    design is built.
  * **Chosen by the owner,** 2026-10-27: "Internal editor now
    (Recommended)". MADR A2 item 1. This step's scope gains
    `internal/lineinput/`, a core.
* **Deviation D6, 2026-10-27: the host ends an engine's wait.**
  * **Found:** D4 has "the cancel key, ctrl+c or the end of input" cancel,
    but D2's contract takes `HandleInput`, `Resize` and `HandleTimeout`
    alone, and `termkey` has no input for the end of input; gobble's
    `askPermission` also cancels when its context ends
    (`internal/cli/permission.go:209`, at `5fb6928`).
  * **Resolutions offered:** `Cancel()` in D2's contract, the end of input
    or the host's own reason, ask and pick ending canceled, `lineedit`
    sending `EditEOF` (recommended); `HandleEOF(now)`, the end of input
    alone; or `Asker.Cancel()` alone, each later engine deciding its own.
  * **Chosen by the owner,** 2026-10-27: "Cancel() in the contract
    (Recommended)". MADR A2 item 2.
* **Deviation D7, 2026-10-27: the Asker draws in the host's region.**
  * **Found:** D4 says "the region commits the question" and does not say
    whose region; gobble asks its permission question while its composer
    is live, so a region is on the screen already.
  * **Resolutions offered:** `ask.WithRegion(*inline.Region)`, the host's
    region, shared among its components, the Asker building
    `inline.NewRegion(inline.WithMethod(m))` without it (recommended); or
    an Asker always building its own, with `WithRegionOptions`, a second
    region under a live composer.
  * **Chosen by the owner,** 2026-10-27: "ask.WithRegion (Recommended)".
    MADR A2 item 3.
* **Deviation D8, 2026-10-27: `asktea` draws the Asker's own rows.**
  * **Found:** D4 has `asktea` draw `Layout` and drive the same `Asker`,
    whose only output is the terminal's bytes, so an overlay has no rows
    for a reason being typed. `asktea`'s type names were not among the
    MADR's 31: `python3 -m tools.verify names` over `ask`'s eight,
    `asktea`'s `Dialog`, `AnsweredMsg` and `Option`, and
    `internal/lineinput`'s `Line` reports "12 planned name(s), clean".
  * **Resolutions offered:** `Asker.View(width int) (rows []string, row,
    col int)`, the rows and the cursor the Asker draws, read without a
    change, which the line mode draws through too (recommended); or
    `Asker.Chosen` and `Asker.Reason`, `asktea` drawing the reason row
    itself.
  * **Chosen by the owner,** 2026-10-27: "Asker.View (Recommended)".
    MADR A2 item 4.
* **Deviation D9, 2026-10-27: a word or an answer wider than its row.**
  * **Found:** D4 leaves "a word wider than a row … whole" and never
    breaks inside an answer, while D11's fuzz property has `Layout` never
    split an answer and never write a row wider than the width; both
    cannot hold for a word or an answer wider than the width. Step 1's
    region cuts a row wider than the width, a floor, so a long path in a
    command would be cut on the screen; gobble's 0029-PLAN P2 asks for
    "the whole command, no word cut", which its terminal's own hard wrap
    gives it.
  * **Resolutions offered:** a word wider than the whole row broken at the
    width by grapheme cluster, its pieces on rows of their own, and an
    answer too wide for any row on a row of its own, broken alike, so no
    row is wider than the width and the whole text shows (recommended); or
    D4 as written, the region cutting the row.
  * **Chosen by the owner,** 2026-10-27: "Break it at the width
    (Recommended)". MADR A2 item 5.
* **Deviation D10, 2026-10-27: the events are named for their engines.**
  * **Found,** by `make pre-add-check`'s lint of the first build: revive's
    `exported` rule at `ask/event.go:9` and `:37`, "type name will be used
    as ask.AskEventKind by other packages, and that stutters; consider
    calling this EventKind", and the same of `AskEvent`; D5's `PickEvent`
    and `PickEventKind` would fail alike, and `lineedit`'s `EditEvent`
    does not. `ask.Event` clashes: `python3 -m tools.verify names` reports
    "ask.Event: also in keymap". No file of the repository suppresses a
    revive finding.
  * **Resolutions offered:** each event named for its engine,
    `ask.AskerEvent` and `AskerEventKind`, `pick.ChooserEvent` and
    `ChooserEventKind`, `lineedit.ComposerEvent` and `ComposerEventKind`,
    the kind constants and `NextEvent` unchanged ("6 planned name(s),
    clean") (recommended); or one shared `Output` and `OutputKind` in every
    engine, with `NextOutput`. A per-line suppression was not offered: it
    silences a gate.
  * **Chosen by the owner,** 2026-10-27: "Named for the engine
    (Recommended)". MADR A2 item 6.
* **`internal/lineinput/`** (D5): `Line`, one row of grapheme clusters
  under the width method (`ansi.FirstGraphemeCluster`, which keeps a ZWJ
  sequence and a flag whole under both methods, by a probe); `Insert`,
  cleaned by `internal/sanitize.Line`, cut by whole clusters at its bound,
  and re-segmented, so a mark typed after its base joins its cluster;
  `Key`, the fixed set: text, backspace and ctrl+h, delete and ctrl+d,
  left and ctrl+b, right and ctrl+f, home and ctrl+a, end and ctrl+e,
  ctrl+u, ctrl+k, ctrl+w and alt+backspace; `View(width)`, the part around
  the cursor that fits, its cell counted. A core.
* **`ask/`:**
  * `Question`, `Row`, `Answer`, `Response` (`question.go`). An answer's
    `Key` matches a key as a binding names it, `shift+n`, or, held with no
    modifier but shift and the locks, the text it types, `N`.
  * `Layout` (`layout.go`), pure: the notes, the title and the lines after
    it wrapped at spaces, continuations indented two; each row's label
    padded to the widest, two cells before its answers; answers `[k]
    label`, two cells apart, broken between answers, continuations under
    the first; the cancel answer ending the last row three cells after the
    answer before it, as gobble's sketch draws it; then `glyph.Set.Prompt`.
    A word or an answer wider than its whole row is broken at the width by
    cluster, onto rows of its own, by D9; a cluster wider than the room the
    hang leaves goes to the row's first column. Every string is cleaned by
    `internal/sanitize.Line`.
  * `Asker` (`asker.go`): `NewAsker`, `Resize` (the first draws),
    `HandleInput`, `Timeout`, `HandleTimeout`, `Cancel` (D6), `NextEvent`
    and `View` (D8). The question is cleaned once, at `NewAsker`, by
    `WithRules`' rules and then `Line`'s; the response carries the caller's
    `Answer` as given. A reason is read through a `lineinput.Line`, its
    lead the answer's label in `MsgReasonPrompt`, cut to half the width
    when wider; enter gives it, its spaces trimmed. The answer is committed
    with the question: the prompt and the echo, `Echo` or `Label`, with the
    reason in `MsgReasonEcho`, wrapped at the width; a cancel's is the
    cancel answer's echo, its label, or `MsgCanceled`.
  * `AskerEvent` and `AskerEventKind` (`event.go`, D10): `AskNone`,
    `AskWrite`, `AskDone`, tokens `none`, `write`, `done`, from
    `internal/enum`.
  * Options (`options.go`): `WithMethod`, `WithGlyphs` (`glyph.Unicode()`
    by default), `WithRegion` (D7), `WithReasonLimit` with
    `DefaultReasonLimit`, 2048 bytes, `WithCatalog`, `WithRules`. `Option`
    is opaque.
  * Messages (`messages.go`): `MsgReasonPrompt` (`%s: `), `MsgReasonEcho`
    (`%[1]s: %[2]s`), `MsgCanceled` (`cancel`), and `Messages()`.
* **`ask/asktea/`:** `Dialog`, from `NewDialog(ws, q, …)`: `Open` builds a
  fresh `Asker` with the workspace's width method and its theme's glyphs,
  then `WithAsk`'s options, and pushes a modal overlay as wide as the
  workspace or `WithWidth`'s and as tall as the question is there;
  `Update` gives the `Asker` the keys, through `keymaptea.FromTea`, and the
  pastes, drops its bytes, and on its answer closes the overlay and sends
  `AnsweredMsg`; `OverlayClosedMsg`, esc's among them, cancels; `View`
  draws `Asker.View` at the pane's width; `Cursor` places the cursor.
  `DefaultID`, `WithID`, `WithAnchor`, `WithWidth`, `WithAsk`. Not a core.
* **The gates:** `ask` and `internal/lineinput` join `cores` and the
  `charmfree` files; the enum table gains `ask.AskerEventKind`.
* **`docs/guides/defaults.md`:** an `ask` table of twelve rows: the method,
  the glyphs, the region, the reason's bound, the words, the rules and the
  overlay's ID, place, width and question, `built`; what cancels, the
  reason's keys and a word wider than its row, `decided` by the MADR's D4
  and A2; esc in the overlay, `decided` by
  [0008-MADR-command-palette.md](0008-MADR-command-palette.md), the
  workspace's rule for every overlay. An `ask/asktea` control is named by
  its path, as `command/commandtea`'s is: the inventory's first run failed
  on `asktea.WithID`, "no package asktea".
* **`docs/glossary.md`:** `ask`'s seven names move to "Built", with
  `asktea`'s `Dialog` and `AnsweredMsg`; `pick`'s and `lineedit`'s
  reserved events take D10's names.
* **The release's notes:** `docs/guides/migrating.md`, "Migrating to
  `v2.3.0`", Steps 1 and 2's additions; `README.md`'s Status, the release
  and a bullet for the question; `docs/architecture.md`, `inline`,
  `inline/inlinetest`, `ask`, `ask/asktea` and `internal/lineinput` in the
  package map, the table, the cores and the tree, as `v2.2.0`'s close-out
  did for its packages.
* **Tests:**
  * `lineinput`: `TestKeys`, each key as a terminal sends it, through
    `termkey.Decoder`; `TestClusters`, a ZWJ sequence, a flag and a
    combining mark one step each under both methods; `TestLimit`;
    `TestView`; `FuzzLine`.
  * `ask`: `TestSketch`, the permission question at 66 columns, gobble's
    0029-MADR D1 sketch row for row; `TestLayoutGolden`, drawn by an
    `Asker` on an `inlinetest.Screen`, each line within its width, the
    permission question at 60, 100 and 177, and the trust, plan-mode,
    grant and tool-call questions at 60 and 100, across colour and
    charset: 44 files, 18 distinct, each read before it was kept. `ask`
    draws no colour, so a case's colour and no-colour files are the same.
    `TestLayoutBreaks`, `TestLayoutCleaned`, `TestLayoutWide`;
    `TestAnswer`, `TestEchoLabel`, `TestUnknownKey`, `TestCancel` (the
    cancel key, ctrl+c, `Cancel`, ctrl+c in a reason, and no cancel
    answer), `TestShiftKey`, `TestReason`, `TestReasonLimit`,
    `TestReasonView`, `TestView`, `TestResize`, `TestWithRegion`,
    `TestCatalog`, `TestRules`, `TestRow1Report`; `FuzzLayout`: no row
    wider than the width but one cluster alone, and an answer that fits a
    row never split.
  * `asktea`: `TestDialog`, `TestDialogReason`, `TestDialogCancel` (esc
    and ctrl+c), `TestDialogWidth`, on a workspace in a `synctest` bubble,
    as `palette`'s tests drive one.
  * My expectations were wrong on review, before a first run, and were
    corrected: in `lineinput`'s tests, a ctrl+u case that typed a stray
    `u`, and two view cases that left out the space at the row's end; the
    long title's pieces in `TestLayoutBreaks`, which carry the
    continuation indent; and in `ask`'s tests, a reason edit whose ctrl+w
    deleted the whole word, a catalogue test at a width that cut its lead,
    an `Asker.Question` that does not exist, a region check that could not
    tell two regions apart, and a first-row report test that left the
    screen's own reply to be fed. The same review found two faults in
    `flow`, fixed before the first run: the pieces of a first word wider
    than the row overran by the indent, and a blank lead could leave an
    empty row.
  * Fuzzing, 60 s each, clean: `FuzzLayout` 781,371 runs, `FuzzLine`
    5,112,265; an earlier 30 s of `FuzzLayout`, 291,932 runs, clean too.
    No input was kept.
* **Lint,** at the first pre-add check: the stuttering names (D10);
  gocritic's `evalOrder` in `Dialog.Update`; revive's `confusing-naming`,
  an unexported `cancel` beside `Cancel`, renamed `endCanceled`;
  staticcheck's ST1018, a literal ZWJ in a test, written `\u200d`.
  `make modernize` then asked for `strings.FieldsSeq`, a `strings.Builder`
  in `flow`'s unbounded row, `slices.Backward` and `strings.SplitSeq`;
  `go fix ./ask/...` applied them.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 15 killed of 15. Two needed a second run: 2.1's first form did
  not compile (`flush` unused), and was rewritten to keep it; 2.7
  survived, since `Layout`, `words` and `Line` each clean by `Line` again,
  so the floor showed only in the lead before a reason, which `View` draws
  as it is: `TestRules` gained that case, a label and a translation
  holding escapes under weaker rules, and it killed 2.7:

  | Mutation | Killed by |
  | :--- | :--- |
  | a row breaks inside an answer | `TestLayoutGolden`: `permission color.utf8.60: line 1 differs`, and the other three charset and colour cases at 60 |
  | a continuation is not indented | `TestSketch`: `at 66 columns: …`; `TestLayoutGolden`: `permission color.utf8.60: line 2 differs` |
  | an unknown key answers | `TestUnknownKey`: `answered &{Answer:{ID:yes …}} by keys no answer shows`; `TestEchoLabel`, `TestCancel` |
  | the echo is the label, not `Echo` | `TestAnswer`: `left: "Go on?\n  [y] yes  [n] no   [c] cancel\n> yes\n"; want the echo after the prompt`; `TestReason`, `TestCatalog` |
  | `Cancel` ignored | `TestCancel`: `Cancel: response <nil>; want canceled`; `TestDialogCancel`: `esc: answers []; want one, canceled` |
  | `WithReasonLimit` ignored | `TestReasonLimit`: `reason of 2048 bytes; want 5` |
  | `WithRules` replaces the floor | `TestRules`: `the lead under weaker rules: "> say\x1b[31m why\n: "; want it cleaned to the floor` |
  | a word wider than its row left whole | `TestLayoutBreaks`: `"/home/<user>/a-very-long-path-that-is-wider-than-the-row/file.txt" is 65 cells, in 30`; `TestLayoutWide` |
  | the line steps by rune, not by cluster | `TestClusters`: `"a👨\u200d👩\u200d👧🇫🇷éb" is 11 clusters; want 5` |
  | esc in the overlay does not cancel | `TestDialogCancel`: `esc: answers []; want one, canceled` |
  | the overlay draws at another width | `TestDialogWidth`: `"  [y] allow  [f] reject and say why   [c] cancel" is 48 cells, in 22` |
  | `WithRegion` ignored | `TestWithRegion`: `the host's region, made WithoutCursorQuery, asked for the cursor: "\x1b[3;3R"` |
  | `WithCatalog` ignored | `TestCatalog`: `while typed: "> reject and say why: non"` |
  | the awaited report is passed on as a key | `TestRow1Report`: `the awaited report answered as a key: &{Answer:{ID:f3 Key:ctrl+f3 …}}` |
  | `ask` imports Bubble Tea | `TestCoresAreCharmFree`: `ask (GOOS=linux) links charm.land/bubbletea/v2: ask → charm.land/bubbletea/v2`, and darwin and windows |

* **Names:** `python3 -m tools.verify names` over the step's twelve, "12
  planned name(s), clean", then over D10's six, "6 planned name(s),
  clean"; conformance's names check passes with `Dialog` and
  `AnsweredMsg`.
* **The API:** `make apicheck`, "against v2.2.0, 0 incompatible
  change(s)".
* **Gates:** `python3 -m tools.verify gates` over the step's 18 Go files,
  `.golangci.yml`, the README, the index, the glossary, three guides,
  `docs/architecture.md` and 2 records: macOS 19 passed, 0 failed, in
  549 s, `release-check` "345 file(s) clean in 1 module(s)", with
  `apicheck` against `v2.2.0` and the examples gate (its count is the
  tracked files, so this step's 16 untracked Go files are outside it, but
  in its lint, vet and tests of `./...`, and in the per-file gates);
  `remote linux` 18 passed, 0 failed, 1 skipped (`gofmt`, which is given
  no files there), in 1193 s, "361 file(s) clean", this step's among
  them; `remote windows` 18 passed, 0 failed, 1 skipped, in 2340 s, "361
  file(s) clean". `make pre-add-check` on the 18 Go files is clean. The
  identifier scan of the change, untracked files included, is clean.
* **The release, `v2.3.0`,** under rule 4:
  * The owner committed Steps 1 and 2 (`c843891`, `c73eee6`). The
    disclosure guard passed over both, and the owner pushed `main`. CI run
    40566281470 passed: `modules`, `gates`, and `test` on macOS 15, Ubuntu
    24.04 and Windows 2025; Docs run 40566281383 passed.
  * The owner tagged `c73eee6` `v2.3.0`, annotated, and pushed the tag;
    its CI run, 40568967724, passed the same five jobs.
  * The proxy serves `v2.3.0` at `c73eee6`. The consumer smoke test, in a
    scratch module outside the repository with `GOWORK` empty, required it
    by `go get`, then ran `go mod tidy`, `go vet` and `go build`, and ran a
    program that asks gobble's permission question on an
    `inlinetest.Screen` through `ask` and `inline`, types `x`, `f`, a
    reason and enter, and checks the answer, `why` with "not now", and the
    screen, ending `> reject: not now`, each line within 60 cells. Its
    closure, listed for Linux, macOS and Windows, is seven modules, and
    `x/ansi` alone of Charm.
  * Its two negative controls failed as they must: the program run with
    `-control`, which expects the answer `allow-once`, exited 1 with
    `response &{Answer:{ID:why …} Reason:not now …}; want allow-once`; and
    the same closure check over a program that imports `ask/asktea` found
    `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`,
    `charm.land/lipgloss/v2`, `colorprofile`, `ultraviolet` and three more
    of `github.com/charmbracelet/x`, on each system.
  * Issue #7 is to be told where the question is, with the four places the
    library differs from its text: D9's break of a word wider than its row,
    no width cap (D2), keys fed to an engine (D2, D3), and the reason's
    fixed keys until `lineedit` (A2 item 1).
* **Step 2 complete, 2026-10-27.**

### Step 3: `pick` and `palette.FromEntries` (issue #8), and a release

*2026-10-27.* The owner committed the A3 records (`4341af2`) and said
"committed, approved to proceed".

* **Before any code** (items 2 and 3): `git diff --stat c73eee6 4341af2`
  over `fuzzy`, `palette`, `keymap`, `glyph`, `msgcat`, `safetext`,
  `inline`, `internal` and `termkey` is empty, so A3.1's facts about this
  repository, read at `699ecb4`, stand at the tree the step starts on; they
  were read again, `go doc -all ./fuzzy` and `./keymap`,
  `palette/engine.go`'s `insert` and `before` (`:420-456`),
  `palette/static.go:38-61`, `palette/view.go`'s `minText` (`:18`),
  `drawRow` (`:72-152`) and `helpLine` (`:205-239`), `palette/keys.go:45-72`
  and `palette/options.go`'s `DefaultLimit`, 200 (`:17`), and agree with
  A3.1, but for its first fact, which D11 corrects. `keymap` is a core, so
  `pick`'s closure holds `command` and `when` through it, both cores.
  `python3 -m tools.verify names` over `pick`'s five types: "5 planned
  name(s), clean".
* **Deviation D11, 2026-10-27: `Rank` breaks a tie by the shorter key.**
  * **Found** by `TestFromEntriesRankedLikeChooser`, the step's first run
    of it: `"gpt": the palette ranks ["m06" "m07" "m08" "m09" "m10"]; the
    Chooser ["m06" "m09" "m07" "m08" "m10"]`. A probe in a scratch module
    showed the five score `prefix 84` alike, and `Rank` returning them by
    length; `fuzzy/rank.go:23-35` orders by grade, value, the shorter key,
    then the input order, while its documentation says "stable on ties",
    which A3.1 quoted and A3.4 item 4 read as the input order. The fault is
    in the record's fact, pre-existing at `4341af2`, not in this step's
    code; the other eight queries of the test agreed.
  * **Resolutions offered:** the input order, the `Chooser` scoring with
    `Matcher.Score` and a stable sort as the palette's engine orders, with
    A3.1 corrected and `Rank`'s documentation made to say its tie order
    (recommended); `Rank`'s order kept and `FromEntries` yielding in it,
    a sessions list's ties then by title length, an accepted decision
    changed; or a tie-order option in `fuzzy`, new exported API.
  * **Chosen by the owner,** 2026-10-27: "Input order (Recommended)".
    0023-MADR A4. This step's scope gains `fuzzy/rank.go`'s documentation,
    a comment only.
* **Deviation D12, 2026-10-27: `inlinetest.Screen` takes a cluster as at
  most two cells wide.**
  * **Found** by `FuzzChooser`, its saved input
    `pick/testdata/fuzz/FuzzChooser/17ead44044a2b0f6`: `width 73, row 0:
    the screen shows "❯ 👨\u200d👩K,  7EL _👧 … no results"; want "❯
    👨\u200d👩K,7EL _👧 … no results"`. A replay of the input through the
    public API in a scratch module drew the row right until the screen's
    own `Resize`, at the same width, between the input's two halves; after
    it the `Chooser` moved to column 8, the right column, and the screen
    put the text two cells further on. Under `ansi.WcWidth` `👨\u200d👩` is
    one cluster four cells wide, by `StringWidth`, `FirstGraphemeCluster`
    and `DecodeSequenceInString` alike (a probe). `Screen.Resize`
    (`inline/inlinetest/screen.go:261-264`) reads a cluster's width as 1,
    or 2 when the next cell continues it, so a wider one is re-laid two
    cells wide; `put` (`:221-226`) and `eraseRight` (`:191-193`) blank one
    cell beside a cluster they split, not all of it. The fault is Step 1's,
    released in `v2.3.0`; the `Region` and `pick` measure as the screen
    writes, and are not at fault. On a scratch copy of the tree, `Resize`
    counting every continuation cell made the four saved inputs of
    `FuzzChooser` pass, and `inline`'s and `inlinetest`'s tests too.
  * **Resolutions offered:** the fix in this step, with a test of a four-
    and a six-cell cluster under `ansi.WcWidth` across a resize, a split
    and an erase, seen failing first, and a mutation, shipped in `v2.4.0`
    as a fix (recommended); or a patch release, `v2.3.1`, before this step
    goes on.
  * **Chosen by the owner,** 2026-10-27: "Fix it in Step 3 (Recommended)".
    This step's scope gains `inline/inlinetest/screen.go` and its test. No
    decision of the MADR changes: D3 has the screen measure by the width
    method, which the fix makes it do.
* **`pick/`,** a core:
  * `Entry{ID, Text, Detail}` (`entry.go`). `ChooserEvent{Kind, Data,
    Entry, Canceled}` and `ChooserEventKind` (`event.go`): `PickNone`,
    `PickWrite`, `PickDone`, tokens `none`, `write`, `done`, from
    `internal/enum`.
  * `Chooser` (`chooser.go`): `NewChooser(entries, …)`, `Resize` (the
    first draws), `HandleInput`, `Timeout`, `HandleTimeout`, `Cancel`,
    `NextEvent`. The entries' `Text` and `Detail`, and the prompt, are
    cleaned once, at `NewChooser`, by `WithRules`' rules and then
    `internal/sanitize.Line`; the event carries the caller's `Entry`. The
    query is an `internal/lineinput.Line` of at most 1024 bytes; a typed
    key's text and a paste are cleaned the same way before they go in.
    Each change of the query scores the entries' `Text` with one
    `fuzzy.Matcher`, through the index's candidates, sorted back into the
    order given, when there are more than `WithIndexAbove` entries, and
    sorts the matches stably by grade and then value (D11); an empty query
    keeps every entry, in order, past the index, so the index's cap never
    hides the list. The selection returns to the first row.
  * The keys go through a `keymap.Dispatcher` over `WithKeymap`'s keymap,
    or one of `DefaultRules()` alone, on the path `[ChooserScope]`, with
    an empty `when.Map`; ctrl+c cancels before it. A key it does not take,
    and a sequence's replayed keys, go to the query. A pending sequence's
    `Wait` is folded into `Timeout`, the earlier of it and the region's,
    and `HandleTimeout` gives the dispatcher its `Expire` when it is due.
    A command of another ID in `ChooserScope` is taken and does nothing.
  * The rows (`view`): the query row, the prompt (`WithPrompt`, or the
    glyphs' `Prompt`), cut to half the width when wider, a space, the
    query's window, and the count right-aligned, left out below 8 cells
    of query; then `min(height, entries)` rows, `glyph.Set.Focus` or its
    width in spaces, a space, the `Text` cut by `internal/sanitize.Truncate`
    with the ellipsis, and the `Detail` right-aligned, left out below 8
    cells of text; blank past the matches. The cursor is on the query row.
  * The end: enter with a match commits one line, the prompt, a space and
    the entry's cleaned `Text`, cut to the width with the ellipsis; enter
    with none does nothing; a cancel commits the prompt and
    `MsgCanceled`. Nothing is taken after.
  * Options (`options.go`), opaque: `WithHeight` and `DefaultHeight` (10),
    `WithPrompt`, `WithQuery`, `WithMethod`, `WithGlyphs`, `WithIndexAbove`
    and `DefaultIndexAbove` (5000), `WithRegion`, `WithCatalog`,
    `WithRules`, `WithKeymap`; a zero or nil value keeps the default.
    Keys (`keys.go`): `SelectNextID`, `SelectPrevID`, `PageNextID`,
    `PagePrevID`, `AcceptID`, `CancelID`, `ChooserScope` and
    `DefaultRules()`. Messages (`messages.go`): `MsgCount`, `MsgCountMore`,
    `MsgNoResults`, `MsgCanceled`, `Messages()`.
  * Its closure holds `command` and `when` through `keymap`, as the facts
    re-read said; both are cores. `pick` imports `palette` nowhere, so
    `palette` can import it.
* **`palette/entries.go`:** `FromEntries(name, entries)`, `Static`'s form,
  each entry's `Text` cleaned by `internal/sanitize.Line` at construction,
  so it is scored as the `Chooser` scores it under its default rules;
  `Search` scores `Text` alone, `Discover` lists the entries first best,
  each hit `Hit{ID, Text, Hint: Detail, Score}` with no `Run`.
* **`fuzzy/rank.go`** (D11): `Rank`'s documentation says its tie order.
* **`inline/inlinetest/screen.go`** (D12): `Resize` counts every cell that
  continues a cluster; `put` and `eraseRight` blank all of a cluster they
  split, by `blankCluster`; the type's documentation says a cluster takes
  as many cells as it is wide.
* **The gates:** `pick` joins `cores` and the `charmfree` files; the enum
  table gains `pick.ChooserEventKind`, whose absence the table's test
  reported first, `pick.ChooserEventKind is an enum the table lacks: add
  it to enums`.
* **`docs/guides/defaults.md`:** a `pick` table of fifteen rows: the
  method, the glyphs, the prompt, the starting query, the height, the
  index, the region, the keys, the command IDs and scope, the words and
  the rules, `built`; what cancels besides the keys (D4), the query's
  editing keys (A2) and equal matches in the order given (A4), `decided`;
  the query's 1024 bytes, `floor`. `palette` gains a `FromEntries` row.
* **`docs/glossary.md`:** `pick`'s four names move from "Reserved" to
  "Built".
* **The release's notes:** `docs/guides/migrating.md`, "Migrating to
  `v2.4.0`", with `pick` and `FromEntries` added, and D12's fix and D11's
  documentation under "Fixed"; `README.md`'s Status, the release and a
  bullet for the picker; `docs/architecture.md`, `pick` in the package map,
  the table and the cores, `palette`'s imports and providers, and
  `internal/lineinput`'s row.
* **Tests:**
  * `pick`, on an `inlinetest.Screen` through a host loop:
    `TestLayoutGolden`, the models list of 40 and the sessions list of 16,
    each with an empty query, `son`, `zzqx` and `son` through the index
    (`WithIndexAbove(10)`), across the matrix at 60 and 100: 64 files, 20
    distinct, since `pick` draws no colour, each read before it was kept;
    the index's files equal `son`'s, as a three-cluster query keeps every
    candidate holding its three clusters. `TestDetail`, `TestRankLikeMatcher`
    (an oracle of `Matcher.Score` and a stable sort over six queries, and a
    tie of `qabzzzzzz` before `qab`, the order given), `TestEmptyQueryOrder`,
    `TestIndex` (a sparse match, `s1o2n3n4e5t6`, that a full ranking finds
    and the index drops, and the index's matches in the full ranking's
    order), `TestCount`, `TestMoves`, `TestKeymap` (a user's key, a removed
    one, a binding in another scope, and a sequence, `g g`, replayed when
    it matches nothing and when its wait ends), `TestCancel` (esc, ctrl+g,
    ctrl+c and `Cancel`, and nothing taken after), `TestQueryEdit`,
    `TestAccept`, `TestAcceptNoMatch`, `TestCleaned`, `TestRules`,
    `TestResize`, `TestWithRegion`, `TestRow1Report`, `TestCatalog`,
    `TestDefaultRules`; and, in the package, `TestHeight`, which reads the
    rows the `Chooser` gives its region, as a blank row and no row look
    alike on a screen.
  * `palette`: `TestFromEntries`; `TestFromEntriesRankedLikeChooser`, for
    nine queries of the 40 models, the hits' IDs through a picker on a
    workspace in a `synctest` bubble equal the IDs a `Chooser` chooses,
    each found by moving to it and pressing enter.
  * `inlinetest`: `TestWiderClusters`, four- and six-cell clusters under
    `ansi.WcWidth` across a resize to the same width and a narrower one,
    a cell inside one written over, its first cell written over, and an
    erase from inside it; all five cases failed on the screen as it was,
    `after a resize to the same width, c at column 5: "a👨‍👩b c"`,
    before D12's fix.
  * `FuzzChooser`, in the package: keys, pastes and a resize between the
    input's halves never panic; the screen shows the `Chooser`'s rows,
    each cut to the width, pieces of no width aside, so none is wider and
    the redraws never drift; and the selection is a drawn row. As first
    written it checked that no screen line was wider than the width, which
    a screen of that width always holds; it was made the stronger check
    before its first run. Its first runs found the check stated wrong
    three times, each a piece of no width, a lone ZWJ, `0` and a ZWJ split
    by the screen's decoder, and spaces before a ZWJ trimmed in the wrong
    order, and found D12; each input is kept as a seed under
    `pick/testdata/fuzz/FuzzChooser/`. Then 60 s, 1,040,436 runs, clean,
    with `-fuzzminimizetime 0`: a run with minimization showed 0 runs a
    second for 37 s, the fuzzer minimizing new inputs, which a run without
    it did not. `FuzzRegion`, 30 s, 840,924 runs, and `FuzzLayout`, 30 s,
    850,732, clean over the fixed screen.
  * `BenchmarkKeystroke`, one typed key and its drawing, the query between
    `so` and `son`, six counts on an Apple M1 Pro, Go 1.27.3: at 5000
    entries, ranked in full, 1.59 to 1.64 ms, 84.9 KB and 80 allocations
    a key; at 100,000, through the index, 1.03 to 1.07 ms, 304 KB and 119
    allocations. Both are under a 16 ms frame: the baseline Step 7's guide
    cites.
  * My expectations were wrong, on review or at a first run, and were
    corrected: `TestMoves`' second pgdown, `openai/o3`, and the first row
    for a query, which the matcher decides; `TestEmptyQueryOrder` drew 41
    rows on a screen of 40, the host's to fit by the region's height;
    `TestCleaned`'s padding, 27 spaces, not 26; `TestRankLikeMatcher`'s tie
    taken as the top two rows, where `ai21/jamba-large` ranks first; and
    the smoke program's count, 3 matches of its 7 entries, not 4.
* **Lint,** at the first pre-add check: gofmt, blank lines left at the end
  of `pick/chooser_test.go`. `make modernize` then asked for `keymap.Default`'s
  promoted fields in `bind`'s literal, as `palette/keys.go` writes them.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 14 killed of 14, the twelve the step names and two for D12:

  | Mutation | Killed by |
  | :--- | :--- |
  | the index is never built | `TestIndex`: `through the index: ["anthropic/claude-sonnet-5-5" "anthropic/claude-sonnet-4-5" "sparse"]; want the sparse match dropped`; `TestCount` |
  | enter commits the query, not the entry | `TestAccept`: `left "model son\n"; want the prompt and the entry's text`; `TestCleaned` |
  | `FromEntries` matches `Detail` | `TestFromEntries`: `zulu, a Detail, matched ["a"]`; `TestFromEntriesRankedLikeChooser` |
  | the `Chooser` matches `Detail` | `TestDetail`: `the query beta matched "> alpha                   beta"`; `TestRankLikeMatcher` |
  | ties broken by length, not input order | `TestRankLikeMatcher`: `a tie: ["ai21/jamba-large" "short" "long" …]; want long before short, the order given`; `TestLayoutGolden`, `sessions-son`; `TestFromEntriesRankedLikeChooser` |
  | the selection kept on a query change | `TestMoves`: `after "m": "" selected; want "mistral/mistral-large-2"` |
  | esc does not cancel | `TestCancel`: `esc: <nil>; want canceled`; `TestCatalog`, `TestDefaultRules` |
  | `WithKeymap` ignored | `TestKeymap`: `ctrl+t, the user's: ">  …  1 of 40"; want the selection moved`; `TestRow1Report` |
  | the count ignores the index's cap | `TestCount`: `through a capped index: "> x  …  1 of 2000"; want 1 of 2000+` |
  | the list's height follows the matches | `TestHeight`: `default, query "entry 01": 2 rows; want the query and 10` |
  | `WithRules` replaces the floor | `TestRules`: `weaker: "> a", writing "…\r> a\nb\x1b[31mc\x1b[m…"; want "> a bc"` |
  | `pick` imports Bubble Tea | `TestCoresAreCharmFree`: `pick (GOOS=linux) links charm.land/bubbletea/v2: pick → charm.land/bubbletea/v2`, and darwin and windows |
  | the screen's re-wrap takes a cluster as at most two cells (D12) | `TestWiderClusters`: `after a resize to the same width, c at column 5: "a👨‍👩b c", column 6; want b overwritten`; `FuzzChooser`'s seed `17ead44044a2b0f6` |
  | the screen blanks one cell of a cluster it splits (D12) | `TestWiderClusters`: `a cell inside it: "x👨‍👩‍👧 z   y"; want "x  z   y"` |

* **Names:** `python3 -m tools.verify names` over `pick`'s five types, "5
  planned name(s), clean", before any code; conformance's names check
  passes with them.
* **The API:** `make apicheck`, "against v2.3.0, 0 incompatible
  change(s)".
* **Gates:** `python3 -m tools.verify gates` over the step's 18 Go files,
  `.golangci.yml`, the README, the index, the glossary, two guides,
  `docs/architecture.md` and 2 records: macOS 19 passed, 0 failed, in
  624 s, `release-check` "361 file(s) clean in 1 module(s)", with
  `apicheck` against `v2.3.0` and the examples gate (its count is the
  tracked files, so this step's untracked files are outside it, but in its
  lint, vet and tests of `./...`, and in the per-file gates), markdownlint
  over 23 files with 0 issues; `remote linux` 18 passed, 0 failed,
  1 skipped (`gofmt`, which is given no files there), in 1207 s, "374
  file(s) clean", this step's among them; `remote windows` 18 passed,
  0 failed, 1 skipped, in 2366 s, "374 file(s) clean". `make pre-add-check`
  on the 18 Go files is clean. The identifier scan of the change, untracked
  files included, is clean.
* **The smoke test, rehearsed** against the tree by a `replace`, before the
  release: a program picks from seven models on an `inlinetest.Screen`
  through `pick` and `inline`, types `son` and enter, and checks the
  `Entry`, `perplexity/sonar-pro`, the rows while picking, "1 of 3", each
  line within 60 cells, and the line left, `model perplexity/sonar-pro`.
  Its closure, listed for Linux, macOS and Windows, is seven modules, and
  `x/ansi` alone of Charm. Its two controls failed as they must: `-control`,
  expecting `anthropic/claude-sonnet-5-5`, exited 1; and the closure check
  over a program that imports `palette` found `charm.land/bubbletea/v2`
  and seven more Charm modules on each system. The same test runs from the
  proxy once `v2.4.0` is tagged.
* **The release, `v2.4.0`,** under rule 4:
  * The owner committed the step (`51997cb`). The disclosure guard passed
    over it and `4341af2`, and the owner pushed `main`. CI run
    40609447730 passed: `modules`, `gates`, and `test` on macOS 15, Ubuntu
    24.04 and Windows 2025; Docs run 40609447605 passed.
  * The owner tagged `51997cb` `v2.4.0`, annotated, and pushed the tag;
    its CI run, 40611335195, passed the same five jobs.
  * The proxy serves `v2.4.0` at `51997cb`. The consumer smoke test, the
    rehearsal's program in a scratch module outside the repository with
    `GOWORK` empty, required it by `go get`, then ran `go mod tidy`,
    `go vet` and `go build`, and the program: it chose
    `perplexity/sonar-pro` from `son`, "1 of 3", and left `model
    perplexity/sonar-pro`. Its closure, listed for Linux, macOS and
    Windows, is seven modules, and `x/ansi` alone of Charm.
  * Its two negative controls failed as they must: `-control` exited 1
    with `chose &{Kind:done … Entry:{ID:perplexity/sonar-pro …}}; want
    anthropic/claude-sonnet-5-5`; and the closure check over a program
    that imports `palette` found `charm.land/bubbletea/v2`,
    `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`, `colorprofile`,
    `ultraviolet` and three more of `github.com/charmbracelet/x` on Linux
    and macOS, and two more on Windows, which has no `x/termios`.
  * Issue #8 is to be told where the picker is, with the six places the
    library differs from its text: the keys a keymap's commands (A3.4 item
    3), a height of 10 (item 2), ties in the order given (A4), no
    highlight until Step 5 (item 7), the query's fixed keys until
    `lineedit` (A2 item 1), and the palette as the TUI's form (D5).
* **Step 3 complete, 2026-10-27.**

### Step 4: `lineedit` and `lineedit/lineedittea` (issue #9), and a release

*2026-10-27.* The owner committed and pushed Step 3's release record
(`b37df5a`), and said "approved proceed".

* **Before any code:** `git diff --stat 4341af2 b37df5a` touches no package
  D6 cites but `fuzzy`'s documentation and `inlinetest` (D11, D12). A
  scratch program over `termkey` and `keymap` read, at `b37df5a`: CR as
  `enter`; LF as `ctrl+j`, so CR LF is two keys; `ESC CR` as `alt+enter`;
  Kitty's `CSI 13 ; 2 u` and modifyOtherKeys' `CSI 27 ; 2 ; 13 ~` as
  `shift+enter`; `ESC b`, `ESC f`, `ESC d` and `ESC DEL` as `alt+b`,
  `alt+f`, `alt+d` and `alt+backspace`; ctrl+x then ctrl+e as two keys;
  and `keymap.New` builds `shift+enter`, `alt+enter`, `ctrl+j` and
  `ctrl+x ctrl+e` in `kind:composer` without an error. x/term v0.47.0's
  `History` (`terminal.go:40-58`) is A3.2's quotation, word for word, and
  its `go.mod` requires `golang.org/x/sys` v0.49.0. gobble's parity table
  is 55 cases (`buffer_test.go:144-204`, at `5fb6928`), its reference
  x/term run live (`:58-90`); its external editor opens from a slash
  command, `/edit`, which calls `compose` (`slash.go:26`, `:105-107`), not
  a key; its ctrl+l
  writes `CSI 2 J CSI H` and redraws (`editor.go:214-222`).
  `python3 -m tools.verify names` over `lineedit`'s nine types, with
  `PasteMode` for `PasteInsert` and `PasteSubmit`, and
  `lineedit/lineedittea`'s `ComposerPane`, `EventMsg` and `Option`: "12
  planned name(s), clean".
* **Deviation D13, 2026-10-27: the parity test's reference.**
  * **Found:** the step's parity test is against x/term's `Terminal`,
    which gobble runs live; `golang.org/x/term` is not in the root's
    `go.mod`, D1 says no module is added, and requiring v0.47.0 would raise
    `golang.org/x/sys` from v0.47.0 to v0.49.0 for the module. Seven of the
    55 cases differ by D6's design: six bracketed pastes and ctrl+c.
    *Corrected 2026-10-28, when the reference was first read:* gobble's
    table has five paste cases, so six differ by D6's design, not the
    seven the question put to the owner said; with D18's two, the test
    leaves out eight by name and compares 47.
  * **Resolutions offered:** a recorded reference, written by a generator
    under `lineedit/testdata/` run in a temporary module, the cases D6
    decides otherwise left out by name (recommended); x/term required for the tests by an
    amendment; or a hand-written table.
  * **Chosen by the owner,** 2026-10-27: "Recorded reference
    (Recommended)". 0023-MADR A5 item 1.
* **Deviation D14, 2026-10-27: ctrl+l.**
  * **Found:** gobble and x/term clear the screen and redraw the line on
    ctrl+l; D6 has no event for it, and API convention 12 keeps a terminal
    write added after `v2.1.0` off until a `With…` option.
  * **Resolutions offered:** `WithClearScreen()`, ctrl+l bound to
    `ClearID` and doing nothing without it (recommended); on by default
    with `WithoutClearScreen()`; or an `EditClear` event and a redraw
    method for the host.
  * **Chosen by the owner,** 2026-10-27: "WithClearScreen opt-in
    (Recommended)". 0023-MADR A5 item 2.
* **Deviation D15, 2026-10-27: the external editor's key.**
  * **Found:** D6 has `WithExternalEditor` and `EditExternal` and names no
    key; gobble opens its editor by a slash command.
  * **Resolutions offered:** ctrl+x ctrl+e, inert without the option
    (recommended); no default key; or alt+e, fish's. The question called
    ctrl+x ctrl+e "Readline's edit-and-execute-command"; it is bash's
    (bash's manual page, at bash 3.2.57), and not in the Readline user
    manual, read afterwards; the choice stands, the attribution corrected
    in A5 item 3.
  * **Chosen by the owner,** 2026-10-27: "ctrl+x ctrl+e (Recommended)".
    0023-MADR A5 item 3.
* **Deviation D16, 2026-10-27: `ask` and `pick` move to the `Composer`.**
  * **Found:** A2 item 1 has this step move `ask`'s reason and `pick`'s
    query to the `Composer`, their keys the keymap's after; the step as
    written does not list it, and `ask` has no `WithKeymap`.
  * **Resolutions offered:** move them in this step, as one-row
    composers, `ask` gaining `WithKeymap` (recommended); in a later step
    of this PLAN; or leave them on `internal/lineinput`.
  * **Chosen by the owner,** 2026-10-27: "Move them in Step 4
    (Recommended)". 0023-MADR A5 item 4. This step's scope gains `ask/`
    and `pick/`, with their tests.
* **Deviation D17, 2026-10-27: a word wider than its row.**
  * **Found,** reading gobble's layout at `5fb6928` for the port: D6 has "a
    wider word left whole", and the MADR's Context says gobble's layout
    "leaves a word wider than a row whole"; gobble's code cuts it at the
    row's end (`internal/cli/editor/layout.go:30-31`, `rowEnd` at
    `:73-88`). Left whole, the region cuts the row at the width, a floor,
    so the rest of the word and the cursor over it are off the screen.
  * **Resolutions offered:** break it at the width by grapheme cluster, as
    gobble's code and A2 item 5 do, correcting D6 and the Context
    (recommended); or leave it whole, as D6 says.
  * **Chosen by the owner,** 2026-10-27: "Break it at the width
    (Recommended)". 0023-MADR A5 item 5.
* **Deviation D18, 2026-10-28: where a forward word motion stops.**
  * **Found,** porting the word keys: A3 item 9 has alt+f and the others go
    by Readline's words, and Readline 8.3's `forward-word` moves "to the end
    of the next word"; x/term's Alt+Right, this step's parity reference,
    moves to the start of the next one (gobble's `toRightWord`,
    `buffer.go:312-323`, its port). Two of gobble's 55 cases tell them
    apart: "Alt+Right twice" and "Alt+Right from a space". The step as
    written asks for both.
  * **Resolutions offered:** Readline's, the word's end, the two cases left
    out by name (recommended); or x/term's, the next word's start, with
    Readline's letters and digits.
  * **Chosen by the owner,** 2026-10-28: "Readline: the word's end
    (Recommended)". 0023-MADR A5 item 6.
* **Deviation D19, 2026-10-28: a mark with no base joins the space before
  it.**
  * **Found** by `FuzzLayout`, 30 s over `ask` after its move to the
    keymap: `a row 4 cells wide at width 3: "  ️0"`, its input kept at
    `ask/testdata/fuzz/FuzzLayout/b7c759da2f54632c`. The same input fails
    on a copy of `b37df5a`, so the fault is Step 2's, released in
    `v2.3.0`, not this step's. A probe of `x/ansi` v0.11.8: U+FE0F alone
    is 0 cells under both methods, and `" \ufe0f"`, one cluster, is 2
    under `ansi.GraphemeWidth`; so a layout that measures a word apart and
    draws it after a space writes a row a cell wider than it measured.
    `ask.Layout` splits words with `strings.Fields`, which splits a space
    and a mark after it apart; `pick`'s rows and the composer draw text
    after a lead ending in a space. Unicode 18.0.0 says such a mark "may"
    be presented as a base (§3.6.1, D52) and shows one in isolation on
    U+00A0 (§2.11.4), which `strings.Fields` also splits at.
  * **Resolutions offered:** measure each row as it is drawn, in this
    step, `ask` breaking only at a lone space, `pick` and the composer
    measuring their lead with the text (recommended); drop such marks when
    cleaning; or a patch release, `v2.4.1`, first.
  * **Chosen by the owner,** 2026-10-28: "Measure rows as drawn
    (Recommended)". 0023-MADR A5 item 7. This step's scope gains
    `ask/layout.go` and `pick/chooser.go`'s rows, with a test each, and
    the release notes a "Fixed" entry for both.
  * *Note, 2026-10-28, at the mutation proofs:* `ask`'s fix was built as
    both halves of the resolution, words broken only at a lone space and
    each row measured as it joins it. Mutation 4.16, the row measured by
    its pieces again, survived; a scratch copy with the words split by
    `strings.Fields` again and the joined measure kept passed too. Each
    half makes the other unreachable, so neither could be seen failing.
    `flow` keeps Step 2's arithmetic, which adds up right once no cluster
    is split, and `words` is the fix: its mutation, a space split from the
    mark that joins it, is killed by `TestLayoutMarks`. 0023-MADR A5 item
    7 reads so.
* **`internal/lineinput/`** (A2 item 1, A5): `Line` is the text of the
  `Composer`, `ask`'s reason and `pick`'s query: grapheme clusters under the
  width method, a line break a cluster of its own, bounded in bytes (`New`)
  or runes (`NewRunes`); `Insert`, the caller's cleaned text, cut by whole
  clusters at the bound and re-segmented; `SetText`, `SetPos`, `Clusters`,
  `Pos`, `View`; and `Do(id)`, the fifteen commands it runs by `lineedit`'s
  IDs, listed by `Commands`: the moves by cluster, by Readline's words (a
  run of clusters whose first rune is a letter or a digit, alt+f to a
  word's end and alt+b to its start, A5 item 6) and to the line's ends,
  each line ending at a line break; the deletions back and forward, by
  those words, back to white space (ctrl+w), and to the line's ends;
  ctrl+d's deletion, its end of input the `Composer`'s; and Readline's
  transpose. The fixed set of keys, `Key`, is gone: every key is a
  keymap's command.
* **`lineedit/`,** a core:
  * `Composer` (`composer.go`): `NewComposer`, `Resize`, `HandleInput`,
    `Timeout`, `HandleTimeout`, `Cancel`, `NextEvent`, `View`, `Text`,
    `SetText`. A key goes through a `keymap.Dispatcher` over `WithKeymap`'s
    keymap or `DefaultRules()` alone, on `[ComposerScope]`, with an empty
    `when.Map`; ctrl+c sends `EditInterrupt` with the text, outside the
    keymap; a ctrl+j right after enter's submission is dropped, so a CR LF
    is one enter, as x/term reads it; a key the keymap does not take types
    its text. The commands: the `Line`'s by `Do`; up and down by row, from
    the layout's places, and from the first or the last row the history's
    entries and the user's draft back (gobble's Q1); ctrl+d's end of input
    on an empty line; enter's submission, or a line break for a trailing
    backslash; alt+enter's and shift+enter's line break; tab's completion,
    the line and the cursor's byte offset to the `Completer`, its result
    cleaned as one line; ctrl+l's `CSI 2 J CSI H` and a redraw at the top,
    only `WithClearScreen` (A5 item 2); ctrl+x ctrl+e's `EditExternal`, only
    `WithExternalEditor` (A5 item 3). A submission is committed to the
    scrollback with its header, added to the history when not empty, sent
    as `EditSubmit`, and an empty prompt drawn below it, the header read
    again. A paste is cleaned by the rules with line breaks and tabs kept,
    then by the paste's floor, printable text with line breaks and tabs
    (A3 item 11), and inserted, or submitted with the text under
    `PasteSubmit`; no completion runs on it. A recalled entry and `SetText`'s
    text are cleaned so too. A pending key sequence's wait is folded into
    `Timeout`.
  * The layout (`layout.go`), gobble's by cluster: each line after its
    prompt, the first `WithPrompt`'s and the rest `WithContinue`'s, rows
    broken at the last space the width reaches, the break taking the
    space, a word wider than its row broken at the width (D17), each row
    after a line's first indented by its prompt's width; a tab drawn as a
    space; a row's lead and its first cluster measured together, as drawn
    (D19); the cursor's place at each cluster, a cursor past a full last
    row on a row below, which `View` adds while the cursor is there, and
    one past a full row before a line break at its last column, so the
    rows never depend on the cursor. `View` puts the header row first.
  * `History` (`history.go`), x/term v0.47.0's interface and its
    documentation; `MemHistory`, `Max` and `DefaultHistorySize` (1000), an
    empty entry and a repeat of the newest dropped, `At` panicking out of
    range, `WriteTo` and `ReadFrom` its file form, a line break `\n` and a
    backslash `\\`, a line's `\r` ending dropped and an unknown escape
    kept; `Completer` and `CompleterFunc`.
  * `ComposerEvent` and `ComposerEventKind` (`event.go`): `EditNone`,
    `EditWrite`, `EditSubmit`, `EditEOF`, `EditInterrupt`, `EditExternal`,
    tokens `none`, `write`, `submit`, `eof`, `interrupt`, `external`.
    `PasteMode` (`options.go`), `PasteInsert` and `PasteSubmit`, tokens
    `insert` and `submit`.
  * Options (`options.go`), opaque: `WithPrompt`, `WithContinue`,
    `WithHeader`, `WithHistory`, `WithCompleter`, `WithKeymap`, `WithLimit`
    and `DefaultLimit` (4096 runes), `WithPaste`, `WithExternalEditor`,
    `WithClearScreen`, `WithMethod`, `WithGlyphs`, `WithRegion`,
    `WithRules`. Keys (`keys.go`): twenty-two command IDs, `CharLeftID` to
    `ExternalID`, `ComposerScope` (`kind:composer`) and `DefaultRules()`,
    35 bindings; shift+enter binds where the keymap's features and facts
    say the terminal sends it apart from enter, as `keymap` binds a default
    it can deliver (`go doc ./keymap Default`), which `TestNewline` showed:
    a keymap with no features left it unbound.
* **`lineedit/lineedittea/`:** `ComposerPane`, from `NewComposerPane(ws,
  …)`, which a program puts in its workspace with `SetPane`: the same
  `Composer`, measured with the workspace's width method and drawn with its
  theme's glyphs, then `WithComposer`'s options, its region asking the
  terminal nothing; `Update` gives it the pane's size, the keys through
  `keymaptea.FromTea` and the pastes, and sends every event but the writes
  as an `EventMsg`, a key sequence's wait a `tea.Tick`; `View` draws the
  rows around the cursor at the pane's size; `Cursor`; `Composer`, for
  `SetText` after the external editor. `DefaultID`, `WithID`,
  `WithComposer`. Not a core.
* **`ask` and `pick`** (A5 item 4): a reason's and a query's keys go
  through the keymap, `ask` on `[ComposerScope]` with `WithKeymap`, new,
  and `pick` on `[ComposerScope, ChooserScope]`, its own scope first;
  `lineedit`'s editing commands run on their `Line`, and in a reason its
  submit gives it. Their defaults build `lineedit.DefaultRules()`, and
  `pick`'s with its own. Every test and golden of both passes unchanged,
  but `pick`'s `TestKeymap`, whose keymap held `pick.DefaultRules()` alone,
  as the new documentation of `WithKeymap` says it may not. D19's fixes:
  `ask`'s words break only at a cluster that is a space alone; `pick`
  measures an entry's text with its marker, the query with the prompt, and
  a count or detail placed right with `right`, which drops it when it does
  not fit.
* **The parity reference** (D13): `lineedit/testdata/xtermparity/main.go`
  runs x/term v0.47.0's `Terminal` live over gobble's 55 cases, as gobble's
  `buffer_test.go:58-90` does, from its own module file, `xterm.mod` with
  `xterm.sum`, by `GOWORK=off GOFLAGS=-mod=mod go run
  -modfile=lineedit/testdata/xtermparity/xterm.mod
  ./lineedit/testdata/xtermparity > lineedit/testdata/xterm-parity.json`;
  the root's `go.mod` and `go.sum` are unchanged. The file holds each
  case's input and x/term's outcome.
* **The gates:** `lineedit` joins `cores` and the `charmfree` files; the
  enum table gains `lineedit.ComposerEventKind` and `lineedit.PasteMode`,
  whose absence its test reported first: `lineedit.ComposerEventKind is an
  enum the table lacks: add it to enums`, and the same of `PasteMode`.
* **`docs/guides/defaults.md`:** a `lineedit` table of 21 rows, among them
  shift+enter's binding, the paste's floor (`floor`), ctrl+c's
  `EditInterrupt` and a `MemHistory`'s repeats (`decided`), and two of
  `lineedittea`; `ask`'s and `pick`'s rows for a reason's and a query's
  keys move from `decided` to `built`, with `WithKeymap` and
  `lineedit.DefaultRules`.
* **`docs/glossary.md`:** `lineedit`'s seven names move to "Built", with
  `PasteMode`, `lineedittea`'s `ComposerPane` and `EventMsg`.
* **The release's notes:** "Migrating to `v2.5.0`", `lineedit`,
  `lineedittea` and `ask.WithKeymap` added; `ask`'s and `pick`'s keys
  changed, alt+backspace by Readline's words where it deleted to a space;
  D19's fix for both; `README.md`'s Status and a bullet for the composer;
  `docs/architecture.md`, the package map, the table, the cores.
* **Tests:**
  * `internal/lineinput`: `TestDo`, each command as a terminal sends its
    key; `TestLines`, the ends and kills at a line break; `TestCommands`;
    `TestClusters`, `TestLimit` in bytes and runes, `TestView`, `FuzzLine`.
  * `lineedit`: `TestKeysMatchXterm`, 47 of gobble's 55 cases at 200
    columns and at 12 but for the nine that move between rows, against the
    reference, eight left out by name: four pastes and the control keys
    inside a paste by D6 and A3 item 11, ctrl+c by D6, and two of alt+right
    by A5 item 6; `TestIDs`, the `Line`'s commands are `lineedit`'s IDs;
    `TestDefaultRules`; `TestWords` (`can't`, `32.3`, a CJK run, an
    accented word composed and decomposed, the arrows with alt and ctrl,
    alt+d, alt+backspace, a ZWJ sequence one step); `TestCtrlW`;
    `TestNewline`; `TestPaste`; `TestMemHistory`; `TestHistoryRows`;
    `TestLayout`; `TestLayoutMarks`; `TestLayoutGolden`, an empty prompt, a
    long text, three lines and a header above a recalled entry, across the
    matrix at 60 and 100: 32 files, 12 distinct, each read before it was
    kept; `TestExternal`, `TestClear`, `TestInterrupt`, `TestEOF`,
    `TestCompleter`, `TestKeymap`, `TestRules`, `TestResize`,
    `TestWithRegion`, `TestRow1Report`; `TestKeyDrawBudget`, gobble's three
    cases, a key at a long text's end, in its middle and a move, each at
    most the changed text plus 40 bytes.
  * `lineedit/lineedittea`: `TestComposerPane`, `TestComposerPaneWindow`,
    `TestComposerPaneExternal`, on a workspace in a `synctest` bubble.
  * `ask`: `TestReasonKeymap`, `TestLayoutMarks`; `pick`: `TestKeymap`'s
    user binding in `lineedit.ComposerScope`, `TestRowsMeasuredAsDrawn`.
    `TestLayoutMarks` of `ask` and `TestRowsMeasuredAsDrawn` of `pick`
    were run on a copy of `b37df5a` and failed there, `"a ️0" is 4 cells`
    at width 3 and `"▸ ️0…" is 5 cells` at width 4; the first form of
    `pick`'s passed there, its query matching no entry, and was rewritten
    to reach an entry cut to its room and a query that fills its window.
  * Fuzzing, with `-fuzzminimizetime 0`: `FuzzComposer`, 60 s, 891,011
    runs, clean, after its first seed showed its own check stated wrong,
    the screen's rows compared from the screen's first row where a
    submission had left rows above the region; `FuzzMemHistory` 30 s,
    4,449,160 runs; `FuzzLine` 30 s, 3,905,715; `FuzzChooser` 30 s,
    446,727, clean over the moved `pick`; and `FuzzLayout` found D19 at
    19 s, its input kept as a seed, clean after the fix in the step's
    test runs.
  * `BenchmarkKeystroke`, a typed key and a backspace in turn at the end of
    a five-row text at 80 columns, six counts on an Apple M1 Pro, Go
    1.27.3: 24.8 to 28.9 µs, 19.7 KB and 81 allocations a key, the layout
    and the segmenting of the whole text in it; gobble's 1.5 µs baseline
    on darwin (its `perf_test.go:14-18`) times the drawing alone, so the
    two are not alike. Under a 16 ms frame: the baseline Step 7's guide
    cites.
  * My expectations were wrong, at a first run, and were corrected:
    `TestDefaultRules`' count, 35 bindings, not 39; `TestWords`' ZWJ case,
    one left, not two; `TestHistoryRows`, up from the last row stopping
    before the space at rune 15, as gobble's own table has it; `TestLayout`,
    a tab typed is tab's completion, so a pasted one; `TestNewline`, which
    needed a keymap with Kitty's features for shift+enter; the pane's
    tests, `View`'s trailing space, the pane's own size and a sleep past the
    keymap's wait in the `synctest` bubble; and the smoke program's
    history step, whose ctrl+u cut a recalled line's first row only, as
    documented.
* **Lint,** at the first pre-add check: gofmt in three files, and
  gocritic's `evalOrder` at `lineedittea`'s `return p, p.drain(now)`,
  split as `asktea`'s was. staticcheck's ST1018 was met before lint ran:
  literal U+200D, U+0301 and U+FE0F in test strings, written as escapes.
  `make modernize` asked for nothing. A first layout that measured every
  prefix whole took the tests from 1.6 s to 16 s, quadratic over gobble's
  4100-key case, and was replaced by measuring a row's lead and first
  cluster together and summing the rest, as before.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 23 killed of 23, the five the step names, the ones A3 and A5 add,
  and those of D19; 4.16 first survived, and was replaced, as D19's note
  says; 4.23 survived until `TestRowsMeasuredAsDrawn` checked that a
  detail is kept:

  | Mutation | Killed by |
  | :--- | :--- |
  | a word motion stops inside a cluster | `TestClusters`: `"a👨\u200d👩\u200d👧🇫🇷éb" is 10 clusters; want 5`; `TestLimit` |
  | up recalls history from a middle row | `TestHistoryRows`: `up from the last row: "the quick brown fox jumps|"; want "the quick brown| fox jumps"` |
  | a duplicate is kept | `TestHistoryRows`: `after three submits: 3 entries, up twice gives "second"; want 2, first` |
  | a paste submits under `PasteInsert` | `TestPaste`: `PasteInsert: "", submitted ["say a\tb\ncd"] …; want the text kept, nothing submitted`; `TestLayout`, `TestRules` |
  | the external editor's event is sent without its option | `TestExternal`: `EditExternal sent without WithExternalEditor` |
  | alt+f stops at the next word's start (D18) | `TestDo`: `alt+f to the word's end: … gives "one |two"; want "one| two"`; `TestWords` |
  | ctrl+w stops at a word, not white space | `TestDo`: `ctrl+w over punctuation: … gives "a b-c.|"; want "a |"`; `TestCtrlW` |
  | alt+enter inserts no newline | `TestNewline`: `text "onetwothree\nfour"`; `TestLayoutGolden` |
  | a trailing backslash does not continue the text | `TestNewline`: `text "four"` |
  | a paste is cleaned by the rules alone, below its floor | `TestRules`: `under weaker rules: "x p\x1b[32mq\nr", …` |
  | ctrl+l clears without `WithClearScreen` | `TestClear`: `WithClearScreen false: cleared true, text "abc"` |
  | `MemHistory` writes a line break as it is | `TestMemHistory`: `file "a\ntwo\nlines\n…"; want "a\ntwo\\nlines\n…"` |
  | a CR LF is two enters | `TestKeysMatchXterm`: `CR LF is one Enter at 200: submitted ["ab" ""], eof false; want ["ab"], false` |
  | a submission is not added to the history | `TestHistoryRows`: `after three submits: 0 entries, up twice gives ""; want 2, first` |
  | the composer measures a row's lead and first cluster apart (D19) | `TestLayoutMarks`: `method 1, width 9: "> ️0 first" is 10 cells` |
  | `ask`'s words split a space from the mark that joins it (D19) | `TestLayoutMarks`: `method 1, width 3: "a ️0" is 4 cells` |
  | `pick` measures an entry's text apart from its marker (D19) | `TestRowsMeasuredAsDrawn`: `method 1, width 4, keys "", row 1: "▸ ️0…" is 5 cells` |
  | `pick`'s query row ignores the join (D19) | `TestRowsMeasuredAsDrawn`: `… row 0: "❯ ️qq" is 5 cells` |
  | `pick`'s detail placed by a sum, not as drawn (D19) | `TestRowsMeasuredAsDrawn`: plain's detail dropped |
  | `ask`'s `WithKeymap` ignored | `TestReasonKeymap`: `response <nil>; want the reason now, backspace removed and ctrl+o submitting` |
  | `pick`'s query takes no `lineedit` command | `TestKeymap`: `ctrl+o, the user's lineedit command: "> son … 1 of 4"; want the query emptied`; `TestEmptyQueryOrder`, `TestQueryEdit` |
  | the pane sends no `EventMsg` | `TestComposerPane`: `events []; want one EditSubmit of hello world`; `TestComposerPaneExternal` |
  | `lineedit` imports Bubble Tea | `TestCoresAreCharmFree`: `lineedit (GOOS=linux) links charm.land/bubbletea/v2: lineedit → charm.land/bubbletea/v2`, and `ask` and `pick` through it, on each system |

* **Names:** `python3 -m tools.verify names` over the step's twelve, "12
  planned name(s), clean", before any code; conformance's names check
  passes with them.
* **The API:** `make apicheck`, "against v2.4.0, 0 incompatible
  change(s)".
* **Gates:** `python3 -m tools.verify gates` over the step's 41 files but
  its goldens, its fuzz seeds and the parity reference: macOS 19 passed,
  0 failed, in 667 s, `release-check` "374 file(s) clean in 1 module(s)",
  with `apicheck` against `v2.4.0` and the examples gate (its count is the
  tracked files, so this step's untracked files are outside it, but in its
  lint, vet and tests of `./...`, and in the per-file gates); `remote
  linux` 18 passed, 0 failed, 1 skipped (`gofmt`, which is given no files
  there), in 1834 s, "391 file(s) clean", this step's among them; `remote
  windows` 18 passed, 0 failed, 1 skipped, in 2436 s, "391 file(s)
  clean". `make pre-add-check` on the 32 Go files is clean. The
  identifier scan of the change, untracked files included, is clean.
* **The smoke test, rehearsed** against the tree by a `replace`: a program
  on an `inlinetest.Screen` through `lineedit` and `inline` types a text,
  alt+enter and a second line, and enter, then `second` and enter, then up,
  and checks the two submissions, the history of two with `second` newest
  and recalled, each line within 40 cells, and the screen, the first
  submission wrapped and its line break after the continuation prompt. Its
  closure, listed for Linux, macOS and Windows, is seven modules, and
  `x/ansi` alone of Charm. Its controls failed as they must: `-control`
  exited 1, and the closure check over a program that imports
  `lineedit/lineedittea` found `charm.land/bubbletea/v2` and seven more
  Charm modules on Linux and macOS, six on Windows.
* **The release, `v2.5.0`,** under rule 4:
  * The owner committed the step (`8c0fa9c`) and pushed `main` before the
    disclosure guard was run by hand, as this PLAN's rule 4 has it; the
    owner's own pre-push hook runs the same guard on every push. CI run
    40719119627 passed: `modules`, `gates`, and `test` on macOS 15, Ubuntu
    24.04 and Windows 2025; Docs run 40719119549 passed.
  * The owner tagged `8c0fa9c` `v2.5.0`, annotated, and pushed the tag;
    its CI run, 40725471833, passed the same five jobs.
  * The proxy serves `v2.5.0` at `8c0fa9c`. The consumer smoke test, the
    rehearsal's program in a scratch module outside the repository with
    `GOWORK` empty, required it by `go get`, then ran `go mod tidy`,
    `go vet` and `go build`, and the program: two submissions, the first
    wrapped with its line break after the continuation prompt, a history
    of two with `second` newest and recalled, and the screen as the
    rehearsal had it. Its closure, listed for Linux, macOS and Windows, is
    seven modules, and `x/ansi` alone of Charm.
  * Its two negative controls failed as they must: `-control` exited 1
    with `submitted ["explain the resize redraw in the inline region\nand
    fix it" "second"]; want ["explain the resize" "second"]`; and the
    closure check over a program that imports `lineedit/lineedittea` found
    `charm.land/bubbletea/v2` and seven more Charm modules on Linux and
    macOS, six on Windows.
  * Issue #9 is to be told where the composer is, with the eight places
    the library differs from its text and from gobble's editor: the paste
    (D6, A3 item 11), ctrl+c (D6), Readline's words (A3 item 9, A5 item
    6), a word wider than its row broken (A5 item 5), the line breaks (A3
    item 10), the external editor's key and event (D6, A5 item 3), ctrl+l
    (A5 item 2), and no width cap (D2).
* **Step 4 complete, 2026-10-28.**
