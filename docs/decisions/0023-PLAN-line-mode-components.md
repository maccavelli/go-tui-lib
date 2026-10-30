---
status: in-progress
date: 2026-10-29
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
| 6a | `stream/`, `transcript/` | the reprint's and the printer's speed, and their budgets (D27, 0023-MADR A9) | a patch, `v2.6.1` |
| 7 | `docs/`, `README.md`; *annotated 2026-10-29:* the four examples' `example_test.go` in `ask/`, `pick/`, `lineedit/` and `stream/`, which the step names | the guide, the close-out | |
| 7a | `internal/sanitize/`, `termcap/reply.go`, `inline/inlinetest/`, `docs/`, `README.md` | a reply's parameters counted on every path to them (D28, 0023-MADR A10) | a patch, `v2.6.2` |

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
* `stream`: `Scanner`, the block scanner of 0009 §4 (*unexported, by
  D21:* 0009's PLAN exports it when `Doc` is built on it); `Printer`, the
  line-stable printer of 0023-MADR D7; `Markup` and `Style`. `Doc`,
  `Renderer`, `RendererFunc` and `Plain` are 0009's PLAN's; this step
  builds what the line mode needs, and leaves `Doc` to that PLAN, which
  builds it on the same `Scanner`.
* **Tests:** gobble's styling and wrapping cases (`render/mdstyle.go`,
  `render/wrap.go`, their tests) as goldens; the zero `Markup` gives the
  input byte for byte; a table held until a blank line; a fence's lines
  printed as they complete; the output the same for any split of the
  input (property test); fuzz: never panics, never a row wider than the
  width outside a code or table row. *Annotated 2026-10-28:* the zero
  `Markup` is byte for byte with `WithoutTableLayout` too (D22, 0023-MADR
  A7); the fuzz property is D7's, a wider word and a block-opening word
  excepted (the execution record's note).
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
  `WithoutScrollbackClear` (D13). *Annotated 2026-10-28, by 0023-MADR A8
  (D23 to D26):* `stream.Printer.At` draws a reply again;
  `WithSynchronized` and `WithNoticeStyle` join the options; a reprint
  still due waits for the host's next `HandleTimeout(now, true)`.
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
* **The release,** under rule 4; issue #10 told. *Annotated 2026-10-29
  (D27):* issue #10 is told after Step 6a's release, with its numbers.

### Step 6a: the reprint's speed, and a patch release

*Added 2026-10-29 by deviation D27, 0023-MADR A9.* Nothing exported
changes.

* **Code:**
  * `stream/wrap.go`: `wrapProse` takes each word and gap as a slice of
    its line, not a byte at a time; `place` measures each word once and
    keeps the held word's width; a word of printable ASCII alone is as wide
    as it is long (A9 item 4).
  * `stream/printer.go`: `At` gives the cleaned `Source` to a new
    `Printer`'s line reader at the width, with no sanitizer, and ends it;
    `Flush`'s ending of the blocks becomes the `end` both share (A9 item
    2).
  * `transcript/transcript.go`: a line no longer in bytes than the width
    takes one row (A9 item 3); `render` walks a piece's lines back without
    splitting it; the reprint writes CR LF as it is built.
* **Tests:**
  * `transcript`: `TestReprintBudget`, the benchmark's reprint at most
    49,696 allocations and 7,274,618 bytes (A9 item 5), skipped under
    `-short`; `TestRowsByBytes`, lines of exactly the width and one more in
    ASCII, a CJK line longer in bytes than the width but not in cells, and
    one wider in cells, each its rows.
  * `stream`: `TestReplyBudget`, gobble's reply in 64-byte chunks at 100
    columns at most 257 allocations; `BenchmarkReplyRender`, gobble's
    `benchReplyRender`.
  * Every existing test of `stream` and `transcript` passes unchanged,
    `TestAt` among them; `FuzzPrinter` and `FuzzTranscript` run 120 s
    each.
  * On a scratch copy only, not committed: a differential fuzz of the new
    `wrapProse` against `v2.6.0`'s, kept beside it in a test file, 120 s,
    every output byte for byte the same.
* **Mutations:** both budgets fail on `v2.6.0`'s `wrap.go` alone, and the
  reprint's on `v2.6.0`'s `At` alone; the byte rule off by one; `Source` keeping
  each chunk as written, before the sanitizers, so that `At`, which no
  longer cleans, would draw what was never cleaned; the CR dropped from a
  CR LF; the ASCII rule taking a byte above `0x7e`.
* **The benchmarks:** `BenchmarkReprint` and `BenchmarkReplyRender`, five
  runs each on macOS, recorded beside gobble's, run the same day.
* **The release,** `v2.6.1`, under rule 4: notes in
  `docs/guides/migrating.md` ("Migrating to `v2.6.1`": nothing to change,
  the numbers); `README.md`'s current release; the smoke test of Step 6,
  against `v2.6.1`; issue #10 told, with the numbers, for the owner to
  close.

### Step 7: documentation and the close-out

* `docs/guides/line-mode.md`: the host loop (read, decode, feed, drain,
  write, arm the deadline), raw mode as the program's, the reader, the
  question, the picker, the composer and the transcript, with examples
  compiled and run (`ExampleAsker`, `ExampleChooser`, `ExampleComposer`,
  `ExamplePrinter`).
* `docs/architecture.md`, `docs/glossary.md`, `docs/README.md`, README's
  Status; issues #7 to #10 told, for the owner to close.
* This PLAN becomes `complete`, and its row says so.

### Step 7a: a reply's parameters, and a patch release

*Added 2026-10-29 by deviation D28, 0023-MADR A10.* Nothing exported
changes.

* **Code:**
  * `internal/sanitize`: the parameter count of the sequence at a
    string's start, on each path to x/ansi's parameter state (ESC, any
    intermediates, then `[` or `P`; 0x9B; 0x90), and the bound, 32.
  * `termcap/reply.go`: `ParseReply` takes the count from there in place
    of its own `params`; its bound is the same.
  * `inline/inlinetest/screen.go`: a sequence at or over the bound is
    decoded with no parser and shows nothing; the parser is sized one
    over the bound, as `ParseReply`'s.
  * `docs/architecture.md`: the `internal/sanitize` row names the count.
* **Tests:**
  * `internal/sanitize`: the count on each path, and none for a sequence
    of no parameters.
  * `termcap`: every path with 32, 33 and 40 separators and a digit is
    `ReplyUnknown` with no panic; a DA1 reply of 31 separators is still
    read; CI's input joins `replySeeds`.
  * `inline/inlinetest`: every path with 40 separators and a digit, between
    two letters, leaves the two letters drawn and no panic; a cursor move
    with 31 separators still moves.
  * `FuzzParseReply` 10 minutes.
* **Mutations:** the count without the intermediates' skip; without
  0x90; off by one at the bound; `Screen` with its guard removed.
* **The release,** `v2.6.2`, under rule 4: notes in
  `docs/guides/migrating.md` ("Migrating to `v2.6.2`": nothing to change);
  `README.md`'s current release; the smoke test of Step 6, against
  `v2.6.2`, with `ParseReply` given CI's input, its negative control the
  same program against `v2.6.1`, which panics.

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
  order; gobble takes each as it lands. *Added 2026-10-29 (D27):* and a
  patch, `v2.6.1`, after Step 6a. *Added 2026-10-29 (D28):* and a patch,
  `v2.6.2`, after Step 7a.
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

### Step 5: `stream`'s core, `Printer` and `Markup`

*2026-10-28.* The owner committed and pushed Step 4's release record
(`919ecef`), and said "post and close. then proceed"; issue #9 has its note
and is closed.

* **Before any code:** the specifications' examples, read at the step's
  start: CommonMark 0.31.2's `spec.json`, section "Setext headings", 27
  examples, 80 to 106, example 95 `Foo\nBar\n---` giving
  `<h2>Foo\nBar</h2>`; and cmark-gfm 0.29.0.gfm.13's `test/spec.txt`, 672
  examples, "Tables (extension)" examples 198 to 205. gobble's renderer at
  `5fb6928` read for the port: `render/mdstream.go` (the holdback),
  `mdstyle.go` (the styling, which styles no link), `wrap.go` (the
  wrapping), `table.go` (the layout, whose `isRow` takes only a row that
  starts with a pipe), `term/style.go` (the SGR attributes), and their
  order in a reply: `term.Sanitize`, `Stream.Push`, `Tables`, `Style`,
  `Wrap.Push` (`internal/cli/view.go:132`, `:275-278`). 0009's MADR §1, §4
  and PLAN Steps 5, 6 and A1.4 read for its amendment; 0009's PLAN is
  `proposed`, and nothing of `stream`, `frame` or `inputfilter` is built.
* **Deviation D20, 2026-10-28: links.**
  * **Found:** A3 item 14 has a reference link styled only when its
    definition came first; D7's styling, gobble's, styles no link.
  * **Resolutions offered:** a link's text underlined and its URL dim, a
    reference link only after its definition, no OSC 8 (recommended); or
    no link styling, A3 item 14's clause corrected.
  * **Chosen by the owner,** 2026-10-28: "Underline a link's text
    (Recommended)". 0023-MADR A6 item 1.
* **Deviation D21, 2026-10-28: the scanner's API.**
  * **Found:** the step exports `stream.Scanner`, which only the `Printer`
    uses here; 0009's `Doc`, its second consumer, is that PLAN's to build.
  * **Resolutions offered:** unexported until `Doc` (recommended); or a
    minimal exported `Scanner` now.
  * **Chosen by the owner,** 2026-10-28: "Unexported until Doc
    (Recommended)". 0023-MADR A6 item 2; 0009-MADR A5; 0009-PLAN's note of
    2026-10-28.
* **Recorded with them** (0023-MADR A6 items 3, 4), as facts the tests
  expect: the examples' numbers; a multi-line setext heading's earlier
  lines print as a paragraph, one line held (Q10); a table is held whole
  until a blank line or another block, its rows with or without outer
  pipes; an HTML block's lines print as they come, unstyled and unwrapped.
* **Deviation D22, 2026-10-28: a table and the zero `Markup`.**
  * **Found,** writing `TestZeroMarkup`: 0023-MADR D7 has a table "laid
    out" and the zero `Markup`'s output "the input, byte for byte", which
    a reply with a table cannot both satisfy. The `Printer` follows
    gobble's order, `Tables` before `Style` (`render/table.go:76`;
    `mdstyle.go:12`'s "byte for byte" is `Style`'s alone), and the test
    passed only with the table taken out of its input. New in this step.
  * **Resolutions offered:** lay a table out under every `Markup`, with
    `stream.WithoutTableLayout` to turn it off, rule 12's control
    (recommended); lay out only under a `Markup` that styles; or always
    lay out, listed as a floor.
  * **Chosen by the owner,** 2026-10-28: "Lay out; WithoutTableLayout".
    0023-MADR A7. Added to the step: `WithoutTableLayout` in
    `stream/options.go`, its test, its defaults row and a mutation.
* **Note, 2026-10-28: the fuzz property's form.** The step's "never a row
  wider than the width outside a code or table row" cannot hold beside
  D7's own "a wider word whole" and "no row beginning with a word that
  would open a block". The owner chose "Record D7's form": `FuzzPrinter`
  asserts, when every line of the cleaned source is prose, that each word
  on a row but its first ends within the width or would open a block
  there.
* **Code:**
  * `stream`, a Charm-free core of ten Go files: `printer.go`, the
    `Printer` (`NewPrinter`, `Write`, `Flush`, `SetWidth`, `Source`,
    `Idle`), which cleans each chunk through the rules' `safetext.Sanitizer`
    and then the floor's, `safetext.Text`, and prints each source line once
    it is stable; `scan.go`, the unexported block scanner (D21): fences,
    indented code, ATX headings, setext underlines, thematic breaks, list
    and quote markers, HTML blocks by CommonMark §4.6's seven kinds and
    their end conditions, reference definitions, and what starts a block;
    `table.go`, GFM §4.10's rows with or without outer pipes, the delimiter
    row, a header of as many cells, and the layout; `inline.go`, code spans,
    then links (inline, images, and full, collapsed and shortcut references
    once defined), then gobble's emphasis; `wrap.go`, gobble's whole-word
    wrapping on one styled line; `style.go`, `Style` (an `internal/enum`
    bit set, with `Apply`), `Markup` and `DefaultMarkup`; `options.go`,
    `WithMarkup`, `WithWidth`, `WithMethod`, `WithRules` and, by D22,
    `WithoutTableLayout`; `doc.go`, with its stability line.
  * `pick`: `WithMarkup(stream.Markup)`. The `Chooser` keeps the query's
    `fuzzy.Matcher`; a row's `Text`, once cut to its room, has the cells the
    query matched in the whole text styled in the `Match` style: the
    matcher's ranges, within the bytes the cut kept, turned into cells by
    `fuzzy.Cells` in the width method, and each cluster that starts in one
    styled. The ellipsis is not.
  * The gates and the docs: `stream` joins `cores` and the `charmfree`
    files; the enum table gains `stream.Style`, a bit set, whose absence its
    test reported first: `stream.Style is an enum the table lacks: add it to
    enums`, and then, before it was listed in `bitSets`,
    `stream.Style(2049).String() = "bold|white", want "Style(2049)"`. `docs/guides/defaults.md` gains a
    `stream` table of eight rows and `pick`'s row for `WithMarkup`;
    `docs/glossary.md` moves `Printer`, `Markup` and `Style` to "Built",
    `Scanner` still reserved; `docs/architecture.md`, the package map, the
    table and the cores; `pick`'s and `stream`'s package documentation. No
    release: Step 6 releases, and its notes take this step's changes.
* **Found by the fuzz, fixed in the step's code** (before any commit, so
  not deviations from the PLAN):
  * A marker's reset taken for a word: a list marker followed by a tab,
    `0)\t000`, cut by `cutPlain` before the marker's `SGR 0`, so the wrapper
    placed the reset as a word of no width and broke the row after it; the
    styled output differed from the plain. `cutPlain` keeps the escapes
    right after the prefix with it. The input is kept as a seed,
    `stream/testdata/fuzz/FuzzPrinter/e336175f3f1a9887`.
  * A table that ends the reply gained a line break: `endTable` gave every
    row one. The `Printer` keeps whether the table's last row had one
    (`tableNL`). The seed `cbf9b04895404c98` is `0|0\n-|-`. The golden
    `TestTableLayout` writes is a row per line and cannot show a last line
    break, so the test checks the end of the output itself: gobble's own
    golden ends `| no   | newline |` with none.
* **Lint,** at the first run of the step's lint on a scratch copy: errcheck
  on the three discarded results of the sanitizers' `Write` and `Close`,
  marked `//nolint:errcheck` with the reason, since `cleaned`'s `Write`
  never fails, as the repository marks other impossible results
  (`keymap/keymaptea/keymaptea.go:54`); revive's `redefines-builtin-id` at
  `close`, renamed `shut`; staticcheck's QF1001 and QF1002, a `letter`
  helper and a tagged switch. `make modernize` then asked for
  `strings.SplitSeq` in three loops of `printer_test.go`.
* **Found, not changed (outside the step):** depguard does not report a
  blank import, `import _ "charm.land/bubbletea/v2"`: on a scratch copy it
  passed lint in `lineedit` and in `stream`, and a used, named import of the
  same package failed it, `import 'charm.land/bubbletea/v2' is not allowed
  from list 'charmfree'`, in both. `TestCoresAreCharmFree` reports a blank
  import through the closure, as mutation 5.21 shows, so no core can ship
  one; the gap is only lint's.
* **Tests** (`stream/printer_test.go`, `stream/examples_test.go`, written by
  a script from the specifications' JSON and text):
  * gobble's styling cases, 24, and five of links, through the `Printer`
    with `DefaultMarkup` (`TestStyleCases`); gobble's styling input as a
    golden (`TestStyleGolden`), which matches gobble's `style.golden` line
    for line but for its table, now laid out; gobble's 30 wrapping cases
    (`TestWrap`), the table case laid out; gobble's table golden
    (`TestTableLayout`), which matches gobble's `tables.golden` line for
    line, and its other text left alone (`TestNotTables`); `TestApply`;
  * the zero `Markup` byte for byte under `WithoutTableLayout`, a table laid
    out without it, its rows as they came and styled with it, and
    `DefaultMarkup` less its escapes the zero `Markup`'s
    (`TestZeroMarkup`);
  * any split: every split in two at a rune's start and 20 random splits of
    every input, plain and styled (`TestAnySplit`);
  * every setext example, 80 to 106, and every table example, 198 to 205,
    as goldens with ESC as `<ESC>` (`TestSetext`, `TestTables`), each read
    against the specification's HTML: 81, 82 and 95 draw only their last
    line as the heading (A6 item 4); the examples the spec gives no
    heading, 85, 87, 88, 92, 93's lazy line, 94, 97 to 101 and 104 to 106,
    print as the paragraph, code or rule they are; 91's code span and HTML
    lines as the spec has them; 199's alignment; 201 ended by a quote and
    202 by a blank line, its row of one cell padded; 203 no table; 204 a
    short row padded and a long one cut;
  * `TestHoldOneLine`, `TestTableHeld`, `TestFenceLines`, `TestInlineOpen`,
    `TestReferenceLater`, `TestWrapCJK`, `TestRules` (weaker rules than the
    floor, and stricter), `TestSource`, `TestHTML` (an HTML block to its
    end, by kind, and a kind 7 tag that does not interrupt a paragraph),
    `TestParagraphContinues`;
  * `FuzzPrinter`: no panic; a split in two at any byte gives the whole's
    output; `DefaultMarkup`'s less its escapes is the plain; the zero
    `Markup` with no width and `WithoutTableLayout` gives the cleaned
    source; and, when every line is prose, D7's form of the width (the
    note above). Its run, after the two fixes and the lint's: 300 s, 15.9
    million inputs, clean.
  * `pick`: `TestMarkupGolden`, the rows the `Chooser` gives its region
    across the matrix at 30 and 60, underlined under colour and plain
    without, each within its width; `TestMarkupKeepsLayout`, the screen the
    same with and without the `Markup`, for both lists, four queries and
    three widths; `FuzzChooser` now draws with `DefaultMarkup`, so its check
    of the screen against each row covers the styled rows. Its run:
    120 s, 743 thousand inputs, clean.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 22 killed of 22: the four the step names, setext, links, lazy
  continuation and HTML, D22's control, the two bugs the fuzz found, and the
  rest of the `Printer`, `pick`'s styling and the core's closure. 5.11 and
  5.12 first failed to build, which is no kill, and were rewritten and
  killed. Every new test fails under one of them:

  | Mutation | Killed by |
  | :--- | :--- |
  | 5.1 a table row printed before the table ends | `printer_test.go:103: tuitest: style color.utf8.0: line 25 differs` |
  | 5.2 a marker dropped by the styler | `printer_test.go:91: list marker: "- item\n" gives " item\n"; want "\x1b[36m-\x1b[0m item\n"` |
  | 5.3 a row starts with a word that opens a block | `printer_test.go:191: a lone dash does not start a row:` |
  | 5.4 the output depends on the split | `TestAnySplit`, on gobble's styling input; `TestHoldOneLine`, `TestFenceLines`, `TestSource`; `FuzzPrinter`'s seeds: `split at 13: "aaaa bbbb cccc dddd, eeee\n"; whole: "aaaa bbbb cccc dddd,\neeee\n"` |
  | 5.5 an underline does not make a heading | `printer_test.go:250: tuitest: setext color.utf8.0: line 8 differs` |
  | 5.6 a reference link styled before its definition | `printer_test.go:91: an undefined reference is text: "[a][b] and [c]\n" gives "\x1b[4m[a]\x1b[0m\x1b[2m[b]\x1b[0m and [c]\n"; want "[a][b] and [c]\n"` |
  | 5.7 a lazy line is taken for a setext heading's text | `printer_test.go:250: tuitest: setext color.utf8.0: line 148 differs` |
  | 5.8 a kind 7 tag interrupts a paragraph | `printer_test.go:484: "text\n<div>\n*a*\n\ntext\n<span>\n*a*\n"` |
  | 5.9 an HTML block never ends by its condition | `printer_test.go:477: "<div class=\"x\">\n**not styled** and a long line here\n</div>\n\n<!-- a\n**not** -->\n*yes*\n"` |
  | 5.10 WithoutTableLayout ignored (D22) | `printer_test.go:116: the zero Markup without table layout changed the text:` |
  | 5.11 a table that ends the reply gains a line break | `printer_test.go:280: a table that ends the reply with no line break: "\\|\n\\| ---- \\| ------- \\|\n\\| no   \\| newline \\|\n"` |
  | 5.12 a marker's reset taken for a word | `printer_test.go:432: styled less its escapes: "0)\n  000"; plain: "0)\t000"` |
  | 5.13 indented code interrupts a paragraph | `printer_test.go:494: "Foo\n    bar\n===\n"; want "Foo\n\x1b[1m    bar\x1b[0m\n\x1b[1m===\x1b[0m\n"` |
  | 5.14 the floor is the rules | `printer_test.go:379: under weaker rules: "ab \x1b[31mred\x1b[0m\n"; want the floor's` |
  | 5.15 a Style applies every colour | `printer_test.go:82: red\\|cyan.Apply("x") = "\x1b[31;36mx\x1b[0m"; want "\x1b[31mx\x1b[0m"` |
  | 5.16 a table's delimiter of another cell count starts it | `printer_test.go:260: tuitest: tables color.utf8.0: line 61 differs` |
  | 5.17 a held line is lost at the end | `printer_test.go:91: not a heading: "#hashtag\n" gives ""; want "#hashtag\n"` |
  | 5.18 pick's WithMarkup ignored | `internal_test.go:31: tuitest: markup color.utf8.30: line 2 differs` |
  | 5.19 pick styles the text before the cut | `internal_test.go:40: tuitest: line 3 is 47 cells wide, over 30: "  a long session name about \x1b[4mson\x1b[0mnets and the sea"` |
  | 5.20 pick styles the ellipsis | `internal_test.go:31: tuitest: markup color.utf8.30: line 3 differs` |
  | 5.21 stream imports Bubble Tea | `charmfree_test.go:186: pick (GOOS=linux) links charm.land/bubbletea/v2: pick → stream → charm.land/bubbletea/v2` |
  | 5.22 pick drops a match that ends the text | `internal_test.go:31: tuitest: markup color.utf8.30: line 3 differs` |

  `TestDefaultsInventory` reads the new rows: on a scratch copy, a
  misspelt control in each fails it, `docs/guides/defaults.md:311 (A table's
  layout): built, but stream exports no WithoutTableLayouts`, and the same
  of `pick.WithMarkups`. depguard's `charmfree` entry for `stream`, on a
  scratch copy with a used import, `stream/zz_dg.go:3:8: import
  'charm.land/bubbletea/v2' is not allowed from list 'charmfree'`.
* **Names:** `python3 -m tools.verify names`, "4 planned name(s), clean",
  before any code; conformance's names check passes with them.
* **The API:** `make apicheck`, "against v2.5.0, 0 incompatible
  change(s)".
* **Gates:** `python3 -m tools.verify gates` over the step's 27 files but
  its goldens and its fuzz seeds: macOS 19 passed, 0 failed, in 648 s,
  `release-check` "391 file(s) clean in 1 module(s)", with `apicheck`
  against `v2.5.0` and the examples gate (its count is the tracked
  files, so `stream`'s are outside it, but in its lint, vet and tests of
  `./...`, and in the per-file gates), markdownlint over 23 files, 0
  issues; `remote linux` 18 passed, 0 failed, 1 skipped (`gofmt`, which
  is given no files there), in 1166 s, "401 file(s) clean", this
  step's among them; `remote windows` 18 passed, 0 failed, 1 skipped, in
  2720 s, "401 file(s) clean". The identifier scan of the change,
  untracked files included, is clean.
* **No release:** Step 6 releases `stream` with `transcript`, and its
  notes take this step's changes. The step is done when the owner
  commits it.
* **Step 5 complete, 2026-10-28:** the owner committed it (`5b713cf`).

### Step 6: `transcript` (issue #10), and a release

*2026-10-28.* The owner said "committed, approved to proceed".

* **Before any code:** gobble's `internal/cli/transcript.go`,
  `transcript_test.go` and `view.go` at `5fb6928`, and its
  `0030-MADR-resize-and-performance.md` (D3 to D5) and PLAN (P3, P4):
  the pieces, each a source and a function of the width; the 4 MiB bound,
  oldest first; the render back to 10,000 rows, the oldest piece cut, a
  dim notice when anything was left out; the reprint at a line's start
  with both wrappers idle, 150 ms after the last resize, as `CSI ?2026h
  CSI 2J CSI 3J CSI H`, the text, `CSI ?2026l`; and `BenchmarkReprint`, a
  4 MB transcript of its 921-byte reply drawn back to 10,000 rows at 60
  columns, 9.4 ms on macOS, 29 ms on the Linux machine and 12 ms on the
  Windows laptop (0030-PLAN, P4's baselines).
* **Deviation D23, 2026-10-28: A7's fact.**
  * **Found,** reading `view.go`: gobble runs `Tables` and `Style` only
    when colour is on (`:269-276`, `:508-513`), so A7 item 1's "its tables
    are laid out with colour off too", written in Step 5, is false, and
    D22 rested partly on it. Nothing is released.
  * **Resolutions offered:** keep D22 and correct the fact
    (recommended); or follow gobble, a table laid out only under a
    `Markup` that styles, `WithoutTableLayout` removed before `v2.6.0`.
  * **Chosen by the owner,** 2026-10-28: "Keep D22, correct A7". 0023-MADR
    A8 item 1.
* **Deviation D24, 2026-10-28: drawing a reply again.**
  * **Found:** D9's `AddReply(p *stream.Printer)` draws a reply at each
    reprint's width, and `stream` exports no way to draw a `Printer`'s
    source with its options at another width.
  * **Resolutions offered:** `stream.Printer.At(width)` (recommended); or
    no `AddReply`, the host passing `AddDrawn` a function of its own.
  * **Chosen by the owner,** 2026-10-28: "Printer.At(width)". 0023-MADR A8
    item 2. Added to the step: `stream/printer.go`'s `At`, and its test.
* **Deviation D25, 2026-10-28: the controls rule 12 asks for.**
  * **Found:** the reprint writes mode 2026 and a notice line, which D9
    and D13 give no control.
  * **Resolutions offered:** `WithSynchronized(on bool)` and the notice in
    `WithNoticeStyle(stream.Style)`, from the catalogue (recommended); or
    `WithSynchronized` alone, the notice unstyled.
  * **Chosen by the owner,** 2026-10-28: "Synchronized + style".
    0023-MADR A8 item 3.
* **Deviation D26, 2026-10-28: a reprint due while output is mid-line.**
  * **Found:** a `Timeout` that kept the passed settle deadline would make
    a host spin while its output is not idle.
  * **Resolutions offered:** `Timeout` gives nothing once the deadline has
    passed, and the host calls `HandleTimeout(now, true)` at each idle
    point (recommended); or a `Due()` method as well.
  * **Chosen by the owner,** 2026-10-28: "Call it when idle". 0023-MADR
    A8 item 4.
* **Code:**
  * `transcript`, a Charm-free core: `transcript.go`, the `Transcript`
    (`NewTranscript`, `AddText`, `AddDrawn`, `AddReply`, `Resize`,
    `Timeout`, `HandleTimeout`): pieces of text cleaned by the rules and
    then the floor, `safetext.Text`; functions of the width, written as
    drawn, as `inline.Region` writes a host's rows; and replies, drawn by
    `stream.Printer.At`. The bound drops the oldest pieces past
    `WithMaxBytes`, the newest always kept, a reply counted by its source
    as it grows, checked as pieces are added and at a reprint. The render
    walks back from the newest piece to `WithMaxRows` rows, counted as the
    terminal shows them at the width under `WithMethod`, cuts the oldest
    piece it shows by its first lines, and draws no piece it does not
    show. The reprint is `SGR 0`, then mode 2026 set, `CSI 2 J`, `CSI 3 J`,
    `CSI H`, the notice when anything was left out, the text with CR LF
    line breaks, and mode 2026 reset. The first `Resize` sets the width; a
    later one with another width starts the settle. `options.go`, the nine
    options and `DefaultMaxBytes`, `DefaultMaxRows` and `DefaultSettle`;
    `messages.go`, `MsgEarlierOutput` and `Messages`; `doc.go`, the host's
    part, kitty's and iTerm2's handling of `CSI 3 J`, and the stability
    line.
  * `stream`: `Printer.At` (D24).
  * The gates and the docs: `transcript` joins `cores` and the `charmfree`
    files; `docs/guides/defaults.md` gains a `transcript` table of nine
    rows; `docs/glossary.md` moves `Transcript` to "Built";
    `docs/architecture.md`, the package map, the table and the cores, and
    `At` in `stream`'s row.
  * The release's notes: "Migrating to `v2.6.0`", `stream` (Step 5),
    `transcript` and `pick.WithMarkup` added, nothing changed;
    `README.md`'s Status, the current release `v2.6.0` and a bullet for
    the streamed reply and its transcript.
* **Rows, not line breaks:** gobble's render counts line breaks against its
  10,000; this one counts the rows each line takes at the width, as D9's
  `WithMethod` is for, so a line kept whole and wider than the width, a
  heading, a code line or a table row, counts as the rows the terminal
  wraps it into. The benchmark's 10,000 rows at 60 columns are 8,123 lines.
* **Found while writing the tests, fixed in the step's code:** `FuzzTranscript`'s
  first form read an op's argument byte only for a resize, so its seeds
  never reached a reprint, and no mutation failed it; every op now takes
  its argument, both seeds reach a reprint, and the mutations below fail
  them. `TestHostReprint` showed the reprint waiting while the reply's
  last paragraph line is held (Q10), as D26 has it, and now checks that it
  comes at the host's idle call after the reply ends.
* **Lint:** ineffassign in `fuzz_test.go`'s `check`, and `make modernize`'s
  `slices.Backward` in the render's loop.
* **Tests** (`transcript`):
  * gobble's four, through the public API: `TestRendersAtAWidth`,
    `TestNewestRows`, `TestDrawsOnlyWhatItShows` (10 functions called for
    10 rows of 100 pieces), `TestBound` (and the newest piece kept alone
    past the bound);
  * `TestCutPiece`, `TestWrappedRows` (a 30-cell line takes three rows at
    10), `TestReplyGrows`, `TestReprintOrder`, `TestWithoutScrollbackClear`,
    `TestWithSynchronized`, `TestSettle` (three resizes reprint once, at the
    last one's settle; not before it; not while the host is not idle, with
    no deadline meanwhile; then at its idle call; and once), `TestNoNotice`,
    `TestNotice` (the catalogue's, cleaned to one line, styled, and
    `msgcat.Check`), `TestRules`, `TestCRLF`;
  * `TestReprintGolden`, across the matrix at 60 and 100, read: at 60 the
    banner and the prompt are left out, the notice first, the reply cut by
    its first lines and wrapped at 60, its heading, code line, table rows
    and URL wrapped by the screen, 30 rows; at 100 nothing is cut and there
    is no notice. The colour and plain goldens are alike, the screen
    showing no SGR;
  * `TestHostReprint`: a host on `inlinetest.Screen` streams a reply above
    a one-row `inline.Region`, the screen narrows from 60 to 40, the
    reprint waits for the reply's held line, and after the host's
    `Clear`, the reprint and `Draw`, the screen holds the conversation at
    40, the prompt row below it, and no scrollback;
  * `FuzzTranscript`: any ops never panic, no passed deadline is given,
    and each reprint is framed, CR LF, at most the row limit after the
    notice. Its run: 120 s, 2.7 million inputs,
    clean.
  * `stream`: `TestAt`, every input fed a byte at a time under three
    option sets, `At` equal to what was printed at each idle point and
    after the `Flush`, and the `Printer` unchanged by it.
* **The benchmark:** `BenchmarkReprint`, 4 MiB of gobble's 921-byte reply,
  4,554 replies each a `Printer` with `DefaultMarkup`, reprinted at 60
  columns back to 10,000 rows, five runs on macOS (Apple M1 Pro): 12.28 to
  12.45 ms, 15.6 MB and 186,585 allocations a reprint, beside gobble's
  9.4 ms, 7.3 MB and 49,696 allocations on macOS (0030-PLAN P4). Each reply
  shown is drawn by a fresh `Printer` through `At`, with its sanitizers.
  `TestBenchTranscript` checks the reprint: the notice, then 10,000 rows at
  60 columns. Not measured on the Linux and Windows hosts: `tools.verify
  remote` runs the gates, which run no benchmark.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 22 killed of 22: the three the step names, A3 item 15's order,
  D13's and A8's controls, the settle and the idle, the render, the
  cleaning, `At`, and the core's closure. 6.3 first failed to build, which
  is no kill, and was rewritten and killed. Every new test fails under one
  of them; `TestReprintGolden`, written after the run, fails under 6.3,
  6.12 and 6.18 run against it alone:

  | Mutation | Killed by |
  | :--- | :--- |
  | 6.1 the bound keeps the oldest | `transcript_test.go:136: the oldest piece kept past 100 bytes, or not said to be gone: "(earlier output is not shown again)\nthe very first\nyyyyyyy…` |
  | 6.2 the reprint does not wait to settle | `transcript_test.go:215: reprinted before the last resize settled: "\x1b[0m\x1b[?2026h\x1b[2J\x1b[3J\x1b[Hx\r\n\x1b[?2026l"` |
  | 6.3 the notice written when nothing was cut | `host_test.go:87: row 0: "(earlier output is not shown again)"; want "> explain the reprint"` |
  | 6.4 CSI 3 J before CSI 2 J | `bench_test.go:63: reprint "\x1b[0m\x1b[?2026h\x1b[3J\x1b[2J\x1b[H(earlier output is not shown again)\r\nfunc main() { fmt.Println(\"a long line of …` |
  | 6.5 no SGR 0 first | `bench_test.go:63: reprint "\x1b[?2026h\x1b[2J\x1b[3J\x1b[H(earlier output is not shown again)\r\nfunc main() { fmt.Println(\"a long line of code th…` |
  | 6.6 WithoutScrollbackClear ignored | `transcript_test.go:180: "\x1b[0m\x1b[?2026h\x1b[2J\x1b[3J\x1b[Hx\r\n\x1b[?2026l"` |
  | 6.7 WithSynchronized ignored | `transcript_test.go:189: "\x1b[0m\x1b[?2026h\x1b[2J\x1b[3J\x1b[Hx\r\n"` |
  | 6.8 the reprint does not wait for idle | `host_test.go:64: reprinted with the reply's line held: "\x1b[0m\x1b[?2026h\x1b[2J\x1b[3J\x1b[H> explain the reprint\r\nThe transcript keeps every p…` |
  | 6.9 Timeout keeps a passed deadline (D26) | `transcript_test.go:221: a deadline, passed, while the reprint waits for idle` |
  | 6.10 the first Resize starts a settle | `transcript_test.go:200: the first Resize, which only sets the width, reprinted` |
  | 6.11 every piece drawn, shown or not | `transcript_test.go:122: 100 pieces drawn for 10 rows; want 10` |
  | 6.12 rows counted by line breaks, not cells | `bench_test.go:73: 12309 rows at 60 columns, in 10000 lines; want at most 10000, and close to it` |
  | 6.13 line breaks left LF | `host_test.go:87: row 1: "                     The transcript keep"; want "The transcript keeps every piece the"` |
  | 6.14 AddText cleaned by the rules alone, below its floor | `transcript_test.go:261: "ab \x1b[31mred\x1b[0m\n\x1b[2mdim\x1b[0m\n"` |
  | 6.15 a reply's size fixed when added | `transcript_test.go:161: the grown reply not counted against the bound: "before\nfirst line\na second line, longer\n"` |
  | 6.16 the notice not styled | `transcript_test.go:247: "(plus tôt )\nb\n"` |
  | 6.17 the catalogue ignored | `transcript_test.go:247: "\x1b[2m(earlier output is not shown again)\x1b[0m\nb\n"` |
  | 6.18 a cut piece keeps its first lines | `transcript_test.go:92: "(earlier output is not shown again)\none\ntwo\nthree\nfive\n"` |
  | 6.19 At ignores the table setting | `TestAt`, under `WithoutTableLayout`: At's output differs from what was printed, at an idle point |
  | 6.20 At ignores the Markup | `TestAt`, under `DefaultMarkup`: At's output differs from what was printed, at an idle point |
  | 6.21 At does not flush | `TestAt`: after the `Flush`, At's output lacks the held last line |
  | 6.22 transcript imports Bubble Tea | `charmfree_test.go:187: transcript (GOOS=linux) links charm.land/bubbletea/v2: transcript → charm.land/bubbletea/v2` |

  `TestDefaultsInventory` reads the new rows: on a scratch copy a misspelt
  control fails it, `docs/guides/defaults.md:326 (The reprint as one
  synchronized update): built, but transcript exports no
  WithSynchronised`, and a `decided` row citing no record that exists,
  `decided by 0099-MADR-line-mode-components.md, which does not exist`; a
  `decided` row's control is not resolved, by that test's design
  (`internal/conformance/defaults_test.go:103-107`). depguard's `charmfree`
  entry for `transcript`, on a scratch copy with a used import,
  `transcript/zz_dg.go:3:8: import 'charm.land/bubbletea/v2' is not allowed
  from list 'charmfree'`.
* **Names:** `python3 -m tools.verify names` over `Transcript` and `Option`,
  "2 planned name(s), clean", before any code.
* **The API:** `make apicheck`, "against v2.5.0, 0 incompatible
  change(s)".
* **Gates:** `python3 -m tools.verify gates` over the step's 21 files but
  its goldens: macOS 19 passed, 0 failed, in 755 s, `release-check`
  "401 file(s) clean in 1 module(s)", with `apicheck` against `v2.5.0`
  and the examples gate; `remote linux` 18 passed, 0 failed, 1 skipped
  (`gofmt`, which is given no files there), in 1259 s, "409 file(s)
  clean", this step's among them; `remote windows` 18 passed, 0 failed,
  1 skipped, in 2435 s, "409 file(s) clean". A first run on the three
  hosts was stopped part way to add `TestReprintGolden`, which rule 2
  asks for, and run again whole. The identifier scan of the change,
  untracked files included, is clean.
* **The smoke test, rehearsed** against the tree by a `replace`: a program
  on an `inlinetest.Screen` streams a reply with a table through a
  `Printer` with `DefaultMarkup`, keeps it in a `Transcript`, narrows the
  screen from 60 to 40, and checks the reprint, the reply wrapped at 40
  and the table laid out, and the scrollback cleared. Its closure, listed
  for Linux, macOS and Windows, is seven modules, and `x/ansi` alone of
  Charm. Its controls failed as they must: `-control`, expecting the rows
  at 60, exited 1, and the closure check over a program that imports
  `lineedit/lineedittea` found `charm.land/bubbletea/v2` and seven more
  Charm modules on Linux and macOS, six on Windows.
* **The release, `v2.6.0`,** under rule 4:
  * The owner committed the step (`140ca44`), which carried Step 5's
    commit (`5b713cf`) with it, and pushed `main` before the disclosure
    guard was run by hand, as this PLAN's rule 4 has it; the owner's own
    pre-push hook runs the same guard on every push, and run by hand
    afterwards over `919ecef..140ca44` it exited 0 with nothing to report.
    CI run 40812722310 passed: `modules`, `gates`, and `test` on macOS 15,
    Ubuntu 24.04 and Windows 2025; Docs run 40812722271 passed.
  * The owner tagged `140ca44` `v2.6.0`, annotated, and pushed the tag;
    its CI run, 40882028238, passed the same five jobs.
  * The proxy serves `v2.6.0` at `140ca44`. The consumer smoke test, the
    rehearsal's program in a scratch module outside the repository with
    `GOWORK` empty, required it by `go get`, then ran `go mod tidy`,
    `go vet` and `go build`, and the program: a reply with a table
    streamed through a `Printer` with `DefaultMarkup` and kept in a
    `Transcript`, the screen narrowed from 60 to 40, and the reprint, 7
    rows at 40 columns, the reply wrapped and the table laid out, the
    scrollback cleared. Its closure, listed for Linux, macOS and Windows,
    is seven modules, and `x/ansi` alone of Charm.
  * Its two negative controls failed as they must: `-control` exited 1
    with `row 0: "The **transcript** keeps every piece the"; want "The
    **transcript** keeps every piece the program has shown,"`; and the
    closure check over a program that imports `lineedit/lineedittea`
    found `charm.land/bubbletea/v2` and seven more Charm modules on Linux
    and macOS, six on Windows.
* **Deviation D27, 2026-10-29: the reprint's speed.**
  * **Found,** after the tag, when the owner read the draft note for
    issue #10 ("The benchmark comparison is not flattering") and asked
    "assess how gobble does it, we need to get these numbers down": the
    reprint is slower than gobble's and allocates about 3.8 times as much,
    and a streamed reply allocates 2.5 times as much. The profiles of both
    benchmarks, the four causes and the numbers are 0023-MADR A9 item 1.
    No web research was needed: the profiles answered the question. New
    in Steps 5 and 6.
  * **Prototyped** on a scratch export of `140ca44`, each change on the one
    before, `stream`'s and `transcript`'s tests passing at each, five runs
    of `BenchmarkReprint` each:

    | Change | Reprint | Allocations | Bytes |
    | :--- | :--- | :--- | :--- |
    | `v2.6.0` | 12.2 to 12.3 ms | 186,585 | 15,604,126 |
    | the wrapper's words and gaps as slices, each word measured once | 7.6 to 7.8 ms | 50,117 | 13,515,654 |
    | `At` with no sanitizer | 6.0 ms | 43,856 | 7,040,030 |
    | one row for a line no longer in bytes than the width; no split | 5.8 to 5.9 ms | 43,542 | 6,899,217 |
    | CR LF written as the reprint is built | 5.7 to 5.8 ms | 43,541 | 6,530,575 |
    | a word of printable ASCII as wide as it is long | 5.3 to 5.4 ms | 43,541 | 6,530,575 |

    gobble's, the same day: 9.6 to 11.9 ms, 49,696 allocations, 7,274,618
    bytes. A reply streamed in 64-byte chunks at 100 columns: `v2.6.0`
    33.7 µs and 636 allocations, the prototype 18.2 µs and 200, gobble's
    32.9 µs and 257.
  * **Resolutions offered:** a step of its own, Step 6a, with budgets and
    a patch release, `v2.6.1` (recommended); the same code released with
    Step 7; or the wrapper's fix alone, in `v2.6.1`. And for issue #10's
    note: held for the fix (recommended), or posted now with `v2.6.0`'s
    numbers.
  * **Chosen by the owner,** 2026-10-29: "Step 6a, v2.6.1 (Recommended)"
    and "After the fix (Recommended)". 0023-MADR A9. Added to the PLAN:
    Step 6a, its row in "In scope", and its patch in "Rollout".
  * **Rule breaches while investigating, told to the owner:** one `grep`,
    read-only, over gobble's tree; one heredoc, blocked by the hook before
    it ran.
* **Step 6 complete, 2026-10-29;** issue #10 is told after Step 6a's
  release (D27).

### Step 6a: the reprint's speed, and a patch release

*2026-10-29.* The owner committed the amendment (`37151bd`) and said
"committed proceed".

* **Code** (0023-MADR A9 items 2 to 4), the prototype's five changes,
  applied to the tree by the same script that made them on the scratch
  copy, each landing once:
  * `stream/wrap.go`: `wrapProse` takes each word, with the escapes in it,
    and each gap as a slice of the line; `place` measures a word once, by
    `wordWidth`, and keeps the held word's width in `heldW`; `wordWidth`
    is a word's length when it is printable ASCII alone, `0x20` to `0x7e`.
  * `stream/printer.go`: `At` builds a `Printer` with this one's options
    and no sanitizer, gives it the cleaned `Source`, and ends it with
    `end`, which is `Flush`'s ending of the blocks, now shared.
  * `transcript/transcript.go`: `rowsOf` gives one row, unmeasured, to a
    line no longer in bytes than the width; `render` walks a piece back by
    `strings.LastIndexByte` and cuts it by slicing, with no split or join;
    `writeCRLF` writes the text's line breaks as CR LF into the reprint's
    builder, in place of `crlf`'s two copies.
* **Tests:**
  * `transcript`: `TestReprintBudget` (A9 item 5), the benchmark's reprint
    10 times after one to warm it, counted by `runtime.MemStats` as
    `testing.AllocsPerRun` counts, at most 49,696 allocations and
    7,274,618 bytes; skipped under `-short`. It passes under `-race`.
    `TestRowsByBytes`: ten ASCII bytes at 10 columns one row, eleven two;
    five CJK characters, 15 bytes and 10 cells, one row; six, two.
  * `stream`: `TestReplyBudget`, gobble's reply in 64-byte chunks at 100
    columns, `testing.AllocsPerRun` over 20, at most 257;
    `BenchmarkReplyRender`; gobble's reply copied to
    `stream/testdata/reply.txt`, the same bytes (`cmp` against gobble's
    `internal/cli/render/testdata/reply.txt`).
  * `TestAt` gained one input, with an OSC title, an SGR, a NUL, a BEL and
    a `CSI 2 J`: none of its inputs had a control, so no test showed that
    `At`, which no longer cleans, draws what was printed, cleaned. Every
    other test of `stream` and `transcript` is unchanged and passes.
  * `FuzzPrinter`, 120 s, 10.7 million inputs, and `FuzzTranscript`,
    120 s, 1.7 million, clean.
  * **The differential fuzz,** on a scratch copy of the working tree only:
    `FuzzWrapSame` in a test file beside `v2.6.0`'s `wrapProse`, copied as
    `wrapProseOld` with its wrapper, under both methods at widths 0 to 99:
    120 s, 5.9 million inputs, every output byte for byte the same. Seen
    failing first, with `wordWidth` taking bytes to `0xfe` as ASCII, at its
    seed `"世界 世界世界 a\tb  c   d"`: new `"世界\n世界世界\na\tb\nc   d"`,
    old `"世界\n世界世界 a\nb  c   d"`.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 8 killed of 8. Every new test fails under one of them:

  | Mutation | Killed by |
  | :--- | :--- |
  | 6a.1 `v2.6.0`'s `wrap.go` | `bench_test.go:52: a streamed reply: 636 allocations; want at most 257`; `bench_test.go:79: a reprint: 180009 allocations and 8618659 bytes; want at most 49696 and 7274618` |
  | 6a.2 `v2.6.0`'s `At` | `bench_test.go:79: a reprint: 49802 allocations and 13005666 bytes; want at most 49696 and 7274618` |
  | 6a.3 the byte rule off by one | `transcript_test.go:129: ASCII a byte over: "old\naaaaaaaaaaa\n"; want "(earlier output is not shown again)\naaaaaaaaaaa\n", the line 2 row(s) at 10` |
  | 6a.4 `Source` keeps each chunk as written | `printer_test.go:416:` `TestAt`'s new input, `At` giving the OSC, the NUL and the BEL that were printed cleaned; `printer_test.go:451: Source = "one\x1b[2J\ntwthree\n"` |
  | 6a.5 the CR dropped from a CR LF | `host_test.go:29: tuitest: reprint color.utf8.60: line 4 differs` |
  | 6a.6 the ASCII rule takes a byte above `0x7e` | `printer_test.go:191: wide characters take two columns:` |
  | 6a.7 a cut piece keeps its first lines | `host_test.go:29: tuitest: reprint color.utf8.60: line 2 differs` |
  | 6a.8 the held word's width not kept | `printer_test.go:191: wide characters take two columns:`, and three more of gobble's wrapping cases |

* **The benchmarks,** five runs each on macOS (Apple M1 Pro), beside
  gobble's at `ccd8aa8`, run the same hour, each after the other, with the
  machine otherwise idle but for the remote gates:

  | Benchmark | This tree | gobble |
  | :--- | :--- | :--- |
  | `BenchmarkReprint` | 5.18 to 5.26 ms, 6,530,580 bytes, 43,541 allocations | 9.45 to 9.77 ms, 7,274,150 bytes, 49,696 allocations |
  | `BenchmarkReplyRender` | 20.1 to 28.1 µs, median 23.5 µs, 32,792 bytes, 200 allocations | 32.4 to 36.2 µs, median 34.9 µs, 257 allocations (its `TestPerfBudget`) |

  An earlier run the same day, under load from the macOS gates, gave 5.7
  to 6.9 ms and 22.6 to 31.4 µs here, and 10.4 to 16.8 ms and 35.9 to
  41.2 µs for gobble; the allocation counts are the same in every run.
  `v2.6.0`'s were 12.2 to 12.3 ms and 186,585 allocations, and 33.7 µs and
  636 (D27). Not measured on the Linux and Windows hosts: `tools.verify
  remote` runs the gates, which run no benchmark; `TestReprintBudget` and
  `TestReplyBudget`, which do not depend on time, run there in the gates.
* **The API:** `make apicheck`, "against v2.6.0, 0 incompatible
  change(s)".
* **The release's notes:** "Migrating to `v2.6.1`" in
  `docs/guides/migrating.md`, nothing to change and the numbers;
  `README.md`'s current release `v2.6.1`.
* **Gates:** `python3 -m tools.verify gates` over the step's 10 files:
  macOS 19 passed, 0 failed, in 754 s, `release-check` "409 file(s) clean
  in 1 module(s)", with `apicheck` against `v2.6.0` and the examples
  gate; `remote linux` 18 passed, 0 failed, 1 skipped (`gofmt`, which is
  given no files there), in 1513 s, "410 file(s) clean", this step's
  among them; `remote windows` 18 passed, 0 failed, 1 skipped, in 2420 s,
  "410 file(s) clean". The race, shuffle and `LC_ALL=C` runs on each
  host ran the budgets. "Migrating to `v2.6.1`" took the final numbers
  after the macOS run, a change to its prose alone, linted again after.
  The identifier scan of the change, untracked files included, is clean.
* **The smoke test, rehearsed** against the tree by a `replace`, Step 6's
  program as it is, required as `v2.6.1`: the reprint, 7 rows at 40
  columns, the scrollback cleared; its closure, listed for Linux, macOS
  and Windows, seven modules, and `x/ansi` alone of Charm; `-control`
  exited 1, and the closure check over a program that imports
  `lineedit/lineedittea` found `charm.land/bubbletea/v2` and seven more
  Charm modules.
* **The release, `v2.6.1`,** under rule 4:
  * The owner committed the step (`1af21f0`) and pushed `main`, with the
    amendment (`37151bd`), before the disclosure guard was run by hand;
    the owner's own pre-push hook runs the same guard on every push, and
    run by hand afterwards over `140ca44..1af21f0` it exited 0 with
    nothing to report. CI run 40912691677 passed: `modules`, `gates`, and
    `test` on macOS 15, Ubuntu 24.04 and Windows 2025; Docs run
    40912691492 passed.
  * The owner tagged `1af21f0` `v2.6.1`, annotated, and pushed the tag;
    its CI run, 40921640410, passed the same five jobs.
  * The proxy serves `v2.6.1` at `1af21f0`. The consumer smoke test,
    the rehearsal's program in a scratch module outside the repository
    with `GOWORK` empty, required it by `go get`, then ran `go mod tidy`,
    `go vet` and `go build`, and the program: the reprint, 7 rows at 40
    columns, the scrollback cleared. Its closure, listed for Linux, macOS
    and Windows, is seven modules, and `x/ansi` alone of Charm.
  * Its two negative controls failed as they must: `-control` exited 1
    with `row 0: "The **transcript** keeps every piece the"; want "The
    **transcript** keeps every piece the program has shown,"`; and the
    closure check over a program that imports `lineedit/lineedittea`
    found `charm.land/bubbletea/v2` and seven more Charm modules on Linux
    and macOS, six on Windows.
* **Step 6a complete, 2026-10-29;** issue #10's note, with `v2.6.1`'s
  numbers, is posted when the owner asks (D27).

### Step 7: documentation and the close-out

*2026-10-29.* The owner committed and pushed Step 6a's release record
(`185fb67`) and said "post #10 and close. proceed with step 7". The note
on issue #10, with `v2.6.1`'s numbers, is posted, and the issue is closed
as completed. Issues #7, #8 and #9 have their notes, the ones Steps 2 to 4
drafted, and were closed on 2026-10-27, 2026-10-28 and 2026-10-28
(`gh issue view`), so nothing is left for the owner to close.

* **Before any writing:** the seven packages' documentation read again at
  `185fb67`, `go doc -all` of `inline`, `ask`, `pick`, `lineedit`,
  `stream`, `transcript` and `termkey`, and their host tests,
  `lineedit/host_test.go` and `transcript/host_test.go`; the guide says
  only what these say. 0023-MADR A3 cites no repository fact for this
  step.
* **The examples,** one `example_test.go` in each package, in its `_test`
  package, each run by `go test` against its `// Output:`:
  * `ExampleComposer` is the host loop, `run`, whole: it drains the
    Composer's events, writing what it asks; reads an `inline.Reader`
    only when asked, until the Composer's deadline, the escape timeout
    while the decoder holds bytes, or the next look at the width, 200 ms;
    flushes the decoder when the input goes quiet; feeds each input in;
    cancels at `io.EOF`; and resizes when the width changes. The example
    types two lines on an `inlinetest.Screen` and ends at the input's end.
  * `ExampleAsker`: a question of two rows, the reason typed after `3`;
    `ExampleChooser`: three models, the query `son`, then enter;
    `ExamplePrinter`: a heading, a paragraph wrapped at 32 columns and a
    table laid out at the reply's end, chunk by chunk.
  * The step's row in "In scope" names `docs/` and `README.md`; the
    examples the step names are Go example functions, which live in their
    packages, so the step's Go files are these four, all tests.
* **The guide,** `docs/guides/line-mode.md`: what is the program's (D10);
  the host loop, `ExampleComposer`'s `run`; raw mode with
  `golang.org/x/term`, which the library does not require, and the
  region's `Restore`; one reader and one region, `Print`, the re-wrap
  kinds and the width method; the question, the picker, the composer, the
  printer and the transcript, each with its options and defaults; testing
  on `inlinetest.Screen`; and the defaults guide's rows. Its four output
  blocks are the examples' `// Output:`.
* **The guide's code, checked in a scratch module** outside the
  repository, requiring the tree by a `replace` with `GOWORK` off: `run`
  copied from `lineedit/example_test.go` by a script, the guide's
  `session` over `golang.org/x/term` v0.47.0, and its transcript `host`.
  `go vet` and `go build` passed for macOS, and `go build` for Linux and
  Windows. Its tests passed under `-race`: the guide's `host` on a test
  screen, a line submitted, a reply streamed above the composer, the
  screen narrowed from 60 to 40, and the reprint once the reply ended,
  the reply at 40 columns and no scrollback; and two of the loop, each
  seen failing on a scratch copy of `run`:
  * a lone ESC, then 300 ms of quiet, then enter submits `ab`; with the
    flush removed, `run = [], <nil>; want [ab] <nil>`;
  * the width changed from 40 to 12 while the loop waits draws the line
    again at 12; with the resize removed, `the line was not drawn again
    at 12 columns`.

  The binary ran on a pseudo-terminal (`script`), typed `hello there`,
  enter, a left arrow, `x`, enter and ctrl+d: "2 line(s)", the region's
  `Restore` and `CSI ?2004l` written last.
* **The guide's blocks against their sources:** a script read each of the
  guide's eight code blocks and found each, line for line, in a source
  that compiled and ran, the four Go blocks in the examples and the
  scratch host, and the four text blocks in an example's `// Output:`:
  "8 block(s), 0 failed". On a scratch copy of the guide with one Go line
  and one output line changed it gave `FAIL block 1 (go, 73 lines)` and
  `FAIL block 4 (text, 4 lines)`, and exited 1. No gate in the repository
  runs this check: the guide's excerpts can drift from the examples
  unless a later record adds the guide to `scripts/go-examples.py`, as
  the commands guide is.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 5 killed of 5:

  | Mutation | Killed by |
  | :--- | :--- |
  | 7.1 the Asker given `2` in place of `3` | `ExampleAsker`: got `session ""` and `> allowed for this session` |
  | 7.2 the Chooser given the query `o` | `ExampleChooser`: got `> o … 1 of 3`, three rows, and `chose opus` |
  | 7.3 the loop drops each submission | `ExampleComposer`: got `[] <nil>` |
  | 7.4 the loop writes what the Composer asks to `io.Discard` | `ExampleComposer`: got the submissions and no screen rows |
  | 7.5 the reply's end not printed | `ExamplePrinter`: got `-- flush` and no table |

* **The other documents:**
  * `docs/architecture.md`: "Thirty-one packages, and ten internal ones",
    counted by `go list ./...` with `GOWORK=off`, in place of "Ten
    packages, and five internal ones", stale since before `v1`; the line
    mode among what they are for; `ask`'s row gains `WithKeymap`; the
    tree gains `pick/`, `lineedit/`, `lineedit/lineedittea/`, `stream/`
    and `transcript/`, and the guides' line lists all nine; "What is not
    here" no longer lists a line-mode picker, built in `v2.4.0`, or
    `inline`, built in `v2.3.0`, and says a line-mode CLI sets raw mode
    itself.
  * `docs/README.md`: seven rows in "I want to…", for the host loop,
    what the program owns, the question and the picker, the composer, the
    reply and its reprint, testing on a screen, and 0023-MADR; this
    PLAN's row `complete (v2.6.1)`.
  * `README.md`: the `v2.6.0` bullet says what `v2.6.1` changed and links
    the guide; a row in "I want to…".
  * `docs/glossary.md` needed no change: every name 0023-MADR reserved is
    under "Built from those records", and `stream`'s `Scanner` stays
    reserved, unexported.
* **Gates:** `python3 -m tools.verify gates` over the step's eight files:
  macOS 19 passed, 0 failed, in 826 s, `release-check` "410 file(s) clean
  in 1 module(s)", with `apicheck` and the examples gate; the four
  examples, untracked then, by `make pre-add-check FILES=…`, "4 file(s)
  clean"; `remote linux` 18 passed, 0 failed, 1 skipped (`gofmt`, which
  is given no files there), in 1309 s, "414 file(s) clean", the examples
  among them; `remote windows` 18 passed, 0 failed, 1 skipped, in 2453 s,
  "414 file(s) clean". markdownlint over the guide, `architecture.md` and
  both READMEs: 0 issues; its globs leave the records out. The identifier
  scan of the change, untracked files included, is clean.
* **Verification,** each item, from this record:
  * every step's mutations killed, each failure quoted: 12, 15, 14, 23,
    22 and 22 in Steps 1 to 6, 8 in Step 6a and 5 in Step 7;
  * each step from 3 re-read its facts before any code ("Before any
    code" in Steps 3 to 6; above for Step 7), and recorded a change as a
    deviation (D11, D23 among them);
  * `tools.verify gates` passed on macOS, Linux and Windows for every
    step;
  * `TestCoresAreCharmFree` passes with `inline`, `inlinetest`, `ask`,
    `pick`, `lineedit`, `stream` and `transcript`, and failed on a scratch
    copy that imported Bubble Tea into each (Steps 1 to 6);
  * `TestDefaultsInventory` passes with every component's rows;
  * `make apicheck` reported 0 incompatible changes at each release,
    `v2.3.0` to `v2.6.1`;
  * each release's smoke test passed from the proxy, its closure `x/ansi`
    alone of Charm: `v2.3.0`, `v2.4.0`, `v2.5.0`, `v2.6.0` and `v2.6.1`.
    0023-MADR's Confirmation asks for a smoke test that asks, picks,
    edits and streams: the releases' tests did each in turn, its own
    component, and no one program did all four;
  * the identifier scan found nothing, at every step.
* ~~**This PLAN is complete, 2026-10-29,** once CI passes on this step's
  push, which the handoff reports.~~ *Struck 2026-10-29 by D28, below.*
  Done as its Goal says: every Verification item holds, CI passed on each pushed tree, the owner
  tagged each release, `v2.3.0` to `v2.6.0` and the patch `v2.6.1`, and
  each release's smoke test passed. Left as the PLAN left it: what D10
  leaves out, and `stream/glamourmd`, which 0009's PLAN builds.
* **Deviation D28, 2026-10-29: a reply's parameters.**
  * **Found** by CI on this step's push, run 40969366150, whose `gates`
    job failed in `fuzz`: `FuzzParseReply` panicked in x/ansi's parser on
    `"\x1b [;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;0"`, the input its
    `fuzz-corpus` artifact kept; `modules` and the three `test` jobs
    passed, and Docs run 40969366185 passed. Reproduced in a scratch
    module at `5dd7285` and at `v2.2.0`, and with x/ansi v0.11.9; the same
    panic in `inlinetest.Screen`. Not this step's code: the facts are
    0023-MADR A10 item 1.
  * **Resolutions offered:** amend this record with a Step 7a and a patch
    release, `v2.6.2` (recommended); the same fix under a new pair for
    `termcap`'s parser; or a later record, `main`'s CI failing until then.
  * **Chosen by the owner,** 2026-10-29: "Amend 0023, patch v2.6.2
    (Recommended)". 0023-MADR A10. Added to the PLAN: Step 7a, its row in
    "In scope", and its patch in "Rollout". The PLAN is `in-progress`
    again: CI did not pass on this step's push, so the line above does not
    hold.

### Step 7a: a reply's parameters, and a patch release

*2026-10-29.* The owner committed the amendment (`dc7c912`) and said
"committed, proceed".

* **Tests first,** each run on the code as it was:
  * `termcap`'s `TestParseReplyParamBound`: CI's input, and each of seven
    paths, `ESC [`, `ESC SP [`, `ESC SP ! " [`, 0x9B, `ESC P`, `ESC SP P`
    and 0x90, with 32, 33 and 40 separators, a digit and a final byte,
    each `ReplyUnknown` with no panic. Before the fix it failed seven
    times, each `ESC SP` form with 33 or 40 separators and CI's input,
    `ParseReply("\x1b [;;;…;0") panicked: runtime error: index out of
    range [33] with length 33`; 32 separators fit the parser's 33.
  * `inline/inlinetest`'s `TestManyParams`: each of the seven paths with
    40 separators and a digit between `a` and `b` draws `ab`, and `CSI 3
    C` with 31 separators still moves three cells. Before the fix it
    failed on all seven, `Write("a\x1b[;;…;0Cb") panicked: runtime error:
    index out of range [40] with length 32`.
  * `internal/sanitize`'s `TestSeparators`: the count on each path, with
    `:` as a separator, and none for `ESC [ m`, a bare `ESC [`, an OSC, a
    charset designation, `ESC ESC [`, text and "". It could not compile
    before `Separators` existed; mutations 7a.1 and 7a.2 below are its
    failures.
  * CI's input joins `replySeeds`.
* **Code:**
  * `internal/sanitize/separators.go`: `Separators`, the separators x/ansi's
    parser reads for the sequence at a string's start, on each path, and
    `MaxSeparators`, 32, the bound, with why a parser is one larger.
  * `termcap/reply.go`: `ParseReply` takes both from there; its `params`
    and `maxParams` are gone, and its documentation says a reply over the
    bound is `ReplyUnknown`.
  * `inline/inlinetest/screen.go`: the parser is sized
    `MaxSeparators + 1`; a sequence at or over the bound is decoded with
    no parser and not applied, so it shows nothing and the text after it
    is drawn.
* **Docs:** "Migrating to `v2.6.2`" in `docs/guides/migrating.md`, nothing
  to change and the two fixes, `ParseReply`'s panic dated to `v2.0.0`: a
  scratch module requiring `v2.0.0` from the proxy panics on CI's input,
  `index out of range [33] with length 33`; `README.md`'s current release
  `v2.6.2`; `docs/architecture.md`'s `internal/sanitize` row names
  `Separators` and `MaxSeparators`.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 4 killed of 4:

  | Mutation | Killed by |
  | :--- | :--- |
  | 7a.1 the count without the intermediates' skip | `separators_test.go:34: Separators("\x1b [1;2m") = 0; want 1`, and `TestParseReplyParamBound` and `TestManyParams` |
  | 7a.2 the count without 0x90 | `separators_test.go:34: Separators("\x90>;\|v\x9c") = 0; want 1`; `reply_test.go:137: ParseReply("\x90;;…;0q\x9c") panicked` |
  | 7a.3 `ParseReply`'s bound off by one, `>` for `>=` | `reply_test.go:108: 33 parameters: ParseReply("\x1b[?1;1;…;1c") = {Kind:primary-attributes …}`, the existing case: 32 separators fit the parser and do not panic, and the bound keeps them unknown |
  | 7a.4 `Screen` with its guard removed | `screen_test.go:128: Write("a\x1b[;;…;0Cb") panicked: runtime error: index out of range [40] with length 33`, on all seven paths |

* **Fuzz:** `FuzzParseReply` for 600 s, on a scratch copy of the tree,
  33.3 million inputs, clean, no failing input written.
* **The API:** `make apicheck`, "against v2.6.1, 0 incompatible
  change(s)".
* **Pre-add:** `make pre-add-check` over the seven Go files, "7 file(s)
  clean in 1 module(s)".
* **Gates:** `python3 -m tools.verify gates` over the step's ten files:
  macOS 19 passed, 0 failed, in 760 s, `release-check` "414 file(s) clean
  in 1 module(s)", with `apicheck` and the examples gate; `remote linux`
  18 passed, 0 failed, 1 skipped (`gofmt`, which is given no files
  there), in 1185 s, "416 file(s) clean", this step's two new files
  among them; `remote windows` 18 passed, 0 failed, 1 skipped, in
  2451 s, "416 file(s) clean". markdownlint over the three documents: 0
  issues. The identifier scan of the change, untracked files included,
  is clean.
* **The smoke test, rehearsed** against the tree by a `replace`: Step 6's
  program as it is, required as `v2.6.2`, the reprint 7 rows at 40
  columns, the scrollback cleared; `./reply`, `termcap.ParseReply` on
  CI's input, "ParseReply: unknown"; the closure, listed for Linux, macOS
  and Windows, seven modules, and `x/ansi` alone of Charm. Its three
  controls failed as they must: `-control` exited 1; the closure check
  over `lineedit/lineedittea` found `charm.land/bubbletea/v2` and seven
  more Charm modules; and `./reply` in a module requiring `v2.6.1` from
  the proxy exited 1, `panic: runtime error: index out of range [33] with
  length 33`.
* **The release,** `v2.6.2`, waits for the owner's commit, push and tag
  under rule 4.
