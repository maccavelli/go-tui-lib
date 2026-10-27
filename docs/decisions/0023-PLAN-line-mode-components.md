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
| 3 | `pick/`, `palette/entries.go` | the picker, and `palette.FromEntries` | the next minor |
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

* `pick`: `Entry`, `Chooser`, `PickEvent`, `PickEventKind`, as 0023-MADR
  D5; ranking by `fuzzy`, with an index above `WithIndexAbove`.
* `palette.FromEntries(name string, entries []pick.Entry) Provider`, as
  D5 amended by Q5: `Detail` in the hint slot, no `Run`, `SelectedMsg` on
  accept, the entries in order for an empty query.
* **Tests:** typing filters and ranks as `fuzzy.Rank` does; up, down, page
  up and page down; enter chooses and commits one line; esc cancels; the
  index's best hits equal the full scan's; the same entries ranked alike
  by the `Chooser` and by `FromEntries` for each query of a fixture.
* **Mutations:** the index is never built; enter commits the query, not
  the entry; `FromEntries` matches `Detail`.
* **The release,** under rule 4.

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
