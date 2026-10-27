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
  `TestCount`).
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
  `CompleterFunc`, `EditEvent`, `EditEventKind`, as 0023-MADR D6;
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
  wrap or a re-wrap; `NewScreen(columns, rows, method)`; pending wrap; CSI
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
