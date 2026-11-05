---
status: complete
date: 2026-11-04
associated-madr: "0025-MADR-charm-free-stream-allocations.md"
---
# Implement a Doc's frame without the prefix copied, a Printer's chunks without the line copied again, and no sanitizer kept by a finished reply

Associated MADR: [0025-MADR-charm-free-stream-allocations.md](0025-MADR-charm-free-stream-allocations.md)

## Goal

A host that draws a `Doc`'s frame itself can take it as two strings
that copy nothing (`Doc.RenderParts`, MADR D1); a `Printer`, and with it
the `Writer`, the reprint and `Plain`, streams a reply without copying
the partial line again at each chunk or a draw's input twice (D2); and a
finished `Printer` or `Doc` keeps no sanitizer (D3). Every output is the
same byte for byte, and each saving is held by a budget test.

Done means every item under Verification holds, CI is green on the pushed
tree and on the tag, the release is tagged by the owner, and its smoke
test passes from the proxy.

## Scope

### Facts re-read before execution (at `a8ff40f`)

Each is re-read at the tree the step starts on; one that has changed is a
deviation.

| Fact | Step |
| :--- | :--- |
| `Render` caches the committed segments' renders a width in `widthCache`'s builder, appending each segment once (`stream/document.go:243`, `join` at `:274-281`), renders the tail (`:257`), and joins them, `d.frame = prefix + "\n\n" + tail` (`:266`), keeping the frame by width and version (`:268`) | 5 |
| A width's cache is emptied when a definition is committed (`:316-318`); two widths are kept (`:306`) | 5 |
| `Doc.Finish` closes its sanitizers and keeps them (`:126-133`); `Append` returns at once after `Finish` (`:106-107`), `Write` returns `ErrClosed`; `Reset` makes a new pair (`:138-141`) | 3 |
| `NewPrinter` calls `open` (`stream/printer.go:82`), which makes two sanitizers, the floor's writing into `clean` (`:87-89`); `Flush` closes them and calls `open` again (`:162-166`) | 3 |
| `Write` is `p.san.Write([]byte(chunk))`, then `take` (`:152-155`); the conversion is zero-copy by `go build -gcflags=-m` | 4 |
| `take` copies `clean` into the Source and joins the partial line to it, `s = p.partial + s` (`:192-208`, the join at `:196`) | 4 |
| `draw` writes its input into `clean` (`:138`), as do `Plain`'s `drawWith` (`stream/renderer.go:45`) and `viewWith` (`stream/view.go:273`) | 4 |
| `Idle` reads `p.clean.b.Len()` (`:147`); the Printer's `clean` field is `:46`; the type `cleaned` (`:62-67`) is also `Doc`'s (`stream/document.go:57`) | 4 |
| No other file names the Printer's `clean`, `san` or `base`: `stream/printer.go`, `stream/renderer.go:45` and `stream/view.go:273` alone | 3, 4 |
| `TestReplyBudget`, 257 allocations (`stream/bench_test.go:90-98`); `TestReprintBudget`, 49,696 allocations and 7,274,618 bytes, skipped under `-short`, measured by `perRun` (`transcript/bench_test.go:56-82`, `perRun` at `:85`) | 2 to 5 |
| `BenchmarkReplyRender` feeds 64-byte chunks (`stream/bench_test.go:20`); `BenchmarkDoc` 200 KB in 20-byte chunks with a frame after each (`:62-72`) | 2 |
| The streaming guide's "Feed a Doc" (`docs/guides/streaming-content.md:21`), its excerpt of `stream/document_test.go#doc` (`guide:doc` at `:227`), and "A frame is cheap" (`:38`) | 6 |
| `v2.11.0` is the latest tag; `README.md:21` names it the current release | 6 |

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0025-*`, `docs/README.md` | the MADR accepted, this PLAN `in-progress` |
| 2 | `stream/bench_test.go` | two benchmarks~~, and the helpers the budgets use~~ (D1: each helper with its first use) |
| 3 | `stream/printer.go`, `stream/document.go`, `stream/bench_test.go`, `stream/printer_test.go`, `transcript/bench_test.go` | D3: no sanitizer kept by a finished reply; its budgets |
| 4 | `stream/printer.go`, `stream/renderer.go`, `stream/view.go`, `stream/bench_test.go`, `transcript/bench_test.go`; by D3, `stream/build_race_test.go`, `stream/build_norace_test.go` | D2: the chunk path; its budgets |
| 5 | `stream/document.go`, `stream/document_test.go`, `stream/bench_test.go` | D1: `Doc.RenderParts`; its tests, budget, benchmark and example |
| 6 | `docs/guides/streaming-content.md`, `docs/guides/migrating.md`, `README.md`, this PLAN, the MADR, `docs/README.md` | the guide, the release, the close-out |

No module is added or required. `safetext` is not changed (MADR D2-B and
D3-B were measured and refused). No type is added, so `python3 -m
tools.verify names` has nothing to check; no default is added, so
`docs/guides/defaults.md` gains no row (API convention 12).

### Out of scope

* `Doc`'s own `clean` buffer, `DocPane` and the `Writer`'s conversion
  (MADR, More Information).
* A promise that `RenderParts`' prefix only grows, for a host that prints
  it into its scrollback (MADR D1).
* `git push` and tags, which are the owner's.

### What the prototypes showed (2026-11-04)

The three changes were prototyped on scratch exports of `a8ff40f`, with
probes standing in for this PLAN's tests (MADR, "How every number here
was got"). Each mutation below was applied to the copy with all three
changes, and its failure read.

| Mutation | Probe or test | What it did |
| :--- | :--- | :--- |
| 3.1 `NewPrinter` opens | heap of a Printer never written | 7,451 bytes; Step 3's budget is 3,072 |
| 3.2 `Flush` opens again | heap of a kept reply | 8,475 bytes; budgets 5,120, then 3,584 |
| 3.3 `Flush` keeps the closed pair | `TestSource` | killed: `printer_test.go:530: Source = "one\ntw"` |
| 3.4 `Finish` keeps its pair | heap of a finished `Doc` | 12,646 bytes; budget 6,144 |
| 4.1 the partial line cloned at each chunk | `BenchmarkLongLine`'s work | 2,258,984 bytes and 1,142 allocations; budgets 65,536 and 144 (69,632 and 144 by D2) |
| 4.2 `drawWith` clones its input | `Plain`'s allocations | `Render` 23, `RenderView` 29; budgets 22 and 29 |
| 4.3 each line printed with its line break | `TestStyleCases` | killed at `printer_test.go:92` ("heading"); `TestAnySplit` alone let it survive |
| 4.4 the partial line lost | `TestStyleCases` | killed: `no trailing newline: "# T" gives ""` |
| 5.2 the parts kept across widths | `TestRenderParts`' probe | killed by the twin: `doc 0, size 1, at 40, width 40: RenderParts joined differs from a twin's frame`; one width a chunk, it survived |
| 5.3 `joined` left set | the same | killed: `Render differs from RenderParts joined` |
| 5.4 the finished frame as the tail | the same | killed: `RenderParts is not (Render, "")` |
| 5.5 the parts drawn again at one version | the same | killed: `a second RenderParts allocates 11` |

Three of this PLAN's first drafts were corrected by them: 4.3 and 4.4
named `TestAnySplit`, which a fault common to the split and the whole
input passes; and `TestRenderParts` drew one width a chunk against the
`Doc` itself, which 5.2 passed. The relaxed cut of `TestRenderParts`'
second document was checked by an internal probe: one segment committed
before any blank line. 3.5 is a nil `*safetext.Sanitizer`'s `Close`,
which reads `s.closed` (`safetext/sanitizer.go:66-67`) and so panics;
`TestFlushUnwritten` recovers and asserts.

## Prerequisites

* The owner has accepted the MADR and approved this PLAN.
* `v2.11.0` is the latest tag, and `make apicheck` compares with it. A
  release made between this approval and Step 6, such as
  [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
  Step 7's, moves the base and the name; Step 6 then uses the next minor.

## Rules for every step

[0024-PLAN-command-and-control-opt-outs.md](0024-PLAN-command-and-control-opt-outs.md)'s
rules 1 to 8, with rule 4's base `v2.11.0`, and these:

9. **Output byte for byte.** Every existing test, golden file and example
   passes unchanged. One that needs a change is a deviation.
10. **A budget is a ceiling set above the measured value,** and is seen
    failing twice before it is committed: by its step's mutation, through
    `python3 -m tools.verify mutate`, and by running the step's new tests
    on a scratch export of the commit before the step. Both failures are
    quoted in the record. A budget that fails on any host or under
    `-race`, `-shuffle` or `LC_ALL=C` is a deviation; it is never raised
    to pass.
11. **Bytes and allocations are what a budget holds,** never time.
12. **A heap budget** builds its objects with `GOMAXPROCS` 1, after
    `runtime.GC()`, and reads `runtime.MemStats.HeapAlloc` before and
    after a second `runtime.GC()`, keeping the objects alive past the
    read. No test of the package runs in parallel with it: no test in
    `stream` or `transcript` calls `t.Parallel()` at `a8ff40f`. Under
    `-race`, the prototype's heap probes read within 0.3 % of a plain run
    (a kept reply 3,102 bytes against 3,110, a finished `Doc` 4,963
    against 4,966, a Printer never written 2,075 against 2,086), so the
    budgets hold there with the same margin.

## Implementation Steps

### Step 1: records

* The MADR `accepted`, with the owner's words; this PLAN `in-progress`;
  `docs/README.md`'s two rows.
* Records only.

### Step 2: the measurements in the tree

The benchmarks the MADR's table rests on, so that each later step's
numbers are measured in the tree, before and after.

* `stream/bench_test.go`:
  * `BenchmarkReplyChunks`: `readReply`'s reply through a `Printer` with
    `DefaultMarkup` at 100 columns, in 8-byte chunks, then `Flush`.
  * `BenchmarkLongLine`: one paragraph line,
    `strings.Repeat("a word or two of a long line ", 140)` and a line
    break (4,061 bytes), through the same `Printer` in 4-byte chunks.
  * ~~`perRun(runs int, f func()) (allocs, bytes float64)`, as
    `transcript/bench_test.go:85` has it, for the budgets' bytes.~~
    *Moved to Step 4 by deviation D1, 2026-11-04.*
  * ~~`heldHeap(n int, build func() any) float64`, the heap `n` built
    objects hold, each, by rule 12.~~ *Moved to Step 3 by deviation D1.*
* **Recorded in the execution record:** each benchmark's B/op and
  allocs/op at the step's tree, `-count 5`, with the MADR's table
  beside them; a difference of more than 1 % in bytes or allocations is
  a deviation.
* No library code changes; nothing to mutate.

### Step 3: D3, no sanitizer kept by a finished reply

* **Code,** as MADR D3:
  * `stream/printer.go`: `NewPrinter` calls no `open`; `Write` calls
    `open` when `p.san` is nil; `Flush` closes and drops the pair when
    it has one, and calls no `open`. `Flush`'s documentation keeps its
    words; `open`'s says when it runs.
  * `stream/document.go`: `Finish` sets `d.san` and `d.base` to nil
    after closing them.
* **Tests:**
  * *Added by deviation D1:* `stream/bench_test.go`'s
    `heldHeap(n int, build func() any) float64`, the heap `n` built
    objects hold, each, by rule 12.
  * `stream/bench_test.go`, `TestHeldHeap`, three cases by `heldHeap`
    over 2,000 objects:
    * a kept reply, gobble's reply written and flushed: at most 5,120
      bytes (4,137 in the prototype; 9,512 before);
    * a `Printer` never written: at most 3,072 (2,081; 7,462);
    * a finished `Doc`, the reply appended, finished and drawn at 80: at
      most 6,144 (4,966; 12,646).
  * `transcript/bench_test.go`, `TestTranscriptHeap`: the heap
    `benchTranscript` holds, at most 20 MiB, 20,971,520 bytes (19,068,688;
    43,550,976), skipped under `-short`.
  * `stream/printer_test.go`, `TestFlushUnwritten`: `Flush` of a
    `Printer` never written gives "" and does not panic, recovered and
    asserted; a `Write` after it prints as a new Printer's would.
* **Mutations:**
  * 3.1 `NewPrinter` calls `open` again: `TestHeldHeap`, a Printer never
    written.
  * 3.2 `Flush` calls `open` after dropping the pair: `TestHeldHeap`, a
    kept reply.
  * 3.3 `Flush` closes the pair and keeps it: `TestSource`
    (`stream/printer_test.go:518`), whose `Write` after a `Flush` then
    goes to a closed sanitizer and prints nothing.
  * 3.4 `Finish` keeps the pair: `TestHeldHeap`, a finished `Doc`.
  * 3.5 `Flush` with no nil check: `TestFlushUnwritten`.
* **The budgets' second failure:** Step 3's new tests on a scratch export
  of Step 2's commit, where each heap budget fails.
* **Gates,** rule 4; and the budgets, by rule 10, under `-race`.

### Step 4: D2, the chunk path

* **Code,** as MADR D2:
  * `stream/printer.go`: `open` makes the floor's sanitizer write into
    `&p.source`; `take` scans `p.source.String()` from `p.at` and sets
    `p.partial = src[p.at:]`; `draw` writes its input into `q.source`;
    the `clean` field goes, and `Idle`'s term with it; the `nolint`
    comments that name `cleaned`'s `Write` name the Source's.
  * `stream/renderer.go:45` and `stream/view.go:273`:
    `q.source.WriteString(src)`.
  * `Write` keeps `p.san.Write([]byte(chunk))`.
* **Tests:**
  * *Added by deviation D1:* `stream/bench_test.go`'s
    `perRun(runs int, f func()) (allocs, bytes float64)`, as
    `transcript/bench_test.go:85` has it, for the budgets' bytes.
  * `stream/bench_test.go`:
    * `TestHeldHeap`'s kept reply tightened to at most 3,584 bytes
      (3,110 in the prototype; 4,137 at Step 3's tree);
    * ~~`TestChunkBudget`: `BenchmarkReplyChunks`' work at most 176
      allocations and 23,552 bytes (163 and 21,200; 394 and 31,424 at
      Step 3's tree);~~ *Re-set by deviation D2, 2026-11-04:* at most 176
      allocations and 25,600 bytes (167 and 23,792 with all three changes;
      398 and 34,016 at Step 3's tree), the in-tree benchmark, which keeps
      the output;
    * ~~`TestLongLineBudget`: `BenchmarkLongLine`'s work at most 144
      allocations and 65,536 bytes (127 and 59,960; 2,158 and
      2,271,196);~~ *Re-set by D2:* at most 144 allocations and 69,632
      bytes (128 and 64,056; 2,159 and 2,275,294 at Step 3's tree);
      *by deviation D3,* both with a second ceiling for a race build,
      `TestChunkBudget` at most 192 allocations and 27,648 bytes,
      `TestLongLineBudget` at most 144 and 77,824, chosen by `raceBuild`,
      a constant of `stream/build_race_test.go` (`//go:build race`) and
      `stream/build_norace_test.go`;
    * `TestPlainAllocs`: `Plain`'s `Render` and `RenderView` of
      "A first paragraph, long enough to wrap at forty columns.\n\nThen a
      **second** one.\n" at 40, at most 22 and 29 allocations (22 and 29;
      23 and 30).
  * `transcript/bench_test.go`, `TestTranscriptHeap` tightened to at most
    16 MiB, 16,777,216 bytes (14,400,424; 19,068,688).
  * `TestReplyBudget` and `TestReprintBudget` hold: in the prototype, the
    reply 174 allocations of 257, and the reprint 44,168 allocations and
    6,230,599 bytes of 49,696 and 7,274,618.
* **Mutations:**
  * 4.1 `p.partial = strings.Clone(src[p.at:])`, the partial line copied
    again: `TestLongLineBudget`.
  * 4.2 `drawWith` writes `strings.Clone(src)`, a draw's second copy:
    `TestPlainAllocs`.
  * 4.3 `take` prints each line with its line break,
    `p.line(src[p.at:p.at+i+1], true)`: `TestStyleCases`
    (`stream/printer_test.go:89`), whose wants are fixed strings.
    `TestAnySplit` does not kill it: it compares a split input with the
    whole, and both print the same wrong lines.
  * 4.4 `take` loses the partial line, `p.partial = ""`:
    `TestStyleCases`, its case with no trailing newline.
* **The budgets' second failure:** Step 4's tests on a scratch export of
  Step 3's commit.
* **Fuzz:** `FuzzPrinter`, `FuzzWriter`, `FuzzDoc` and `FuzzTranscript`,
  5 minutes each, no failure and no file added to the tree.
* **Gates,** rule 4; the budgets under `-race`.

### Step 5: D1, `Doc.RenderParts`

* **Code,** as MADR D1, in `stream/document.go`:
  * `Doc` keeps the parts, `partPrefix` and `partTail`, and `joined`,
    beside `frame`, `frameW`, `frameV` and `framed`. The names avoid the
    method `tail` (`stream/document.go:358`), which a field of that name
    would clash with.
  * `parts(width)`: unless `framed` at this width and version, today's
    `Render` body to the tail's render, then the parts kept, `framed` set
    and `joined` cleared.
  * `Render`: `whole` after `Finish`; else `parts`, then the join, once a
    version, into `frame`.
  * `RenderParts(width) (prefix, tail string)`: `(whole(width), "")`
    after `Finish`; else `parts`, then the kept parts. Its documentation
    as MADR D1.
* **Tests,** in `stream/document_test.go` unless named:
  * `TestRenderParts`: over three documents, in chunks of 1, 7 and 64
    bytes, with `SetRevision(7)` at the 51st chunk. The documents: the
    first 12 KiB of `paragraphs()`; 120 lines of one paragraph,
    `"a line of a long paragraph\n"` each, 3,240 bytes, past
    `RelaxAfter`'s 2 KiB, so that a relaxed cut commits a segment before
    any blank line, then a blank line and `next`; and one with a
    paragraph, a fence with a blank line inside, a table and a reference
    used before its definition. After each chunk, at each of widths 40,
    80 and 120, in an order that turns with the chunk, so that three
    widths are asked at one version, one more than the two a `Doc`
    caches:
    * `Render(w)` is `RenderParts(w)` joined;
    * both are what a twin `Doc`, given the same chunks and drawn only
      at `w`, gives: the reference that a cache keyed wrongly cannot
      share;
    * a second `RenderParts(w)` allocates nothing
      (`testing.AllocsPerRun`).

    After `Finish`, at each width, `RenderParts(w)` is `(Render(w), "")`.
  * `stream/bench_test.go`: `BenchmarkDocParts`, `BenchmarkDoc` with
    `RenderParts`; `TestRenderPartsBudget`, its work at most 40 MiB,
    41,943,040 bytes (37,568,770 in the prototype; 1,123,324,637 by
    `Render`).
  * `ExampleDoc_RenderParts`, a guide region `parts`: a paragraph and a
    blank line appended, then a second paragraph still streaming; the
    two parts printed with `%q`. Its output is read before it is
    committed. *By deviation D4,* a blank line between its
    `// guide:end` and its `// Output:`, and `ExampleDoc`'s, so that each
    is run.
  * Every existing `Doc`, `DocPane` and guide test passes unchanged.
* **Mutations:**
  * 5.1 `RenderParts` returns `(d.Render(width), "")`: `TestRenderPartsBudget`.
  * 5.2 `parts` keeps its parts across widths (the `frameW == width`
    test dropped): `TestRenderParts`, by the twin. Drawn at one width a
    chunk, and against itself alone, it survives: each chunk moves the
    version, so the cache is never asked for two widths at one.
  * 5.3 `parts` leaves `joined` set: `TestRenderParts`, `Render` then a
    stale frame.
  * 5.4 `RenderParts` after `Finish` returns `("", whole)`:
    `TestRenderParts`.
  * 5.5 `parts` draws again at the same version (its early return
    dropped): `TestRenderParts`, the second call's allocations.
* **The budget's second failure:** `TestRenderPartsBudget` with
  `RenderParts` replaced by `Render` on a scratch copy, as 5.1 is.
* **Fuzz:** `FuzzDoc`, 5 minutes.
* **Gates,** rule 4; `make apicheck`: 0 incompatible changes, one
  addition, `(*Doc).RenderParts`.

### Step 6: the guide, the release and the close-out

* `docs/guides/streaming-content.md`, "Feed a Doc": a bullet after "A
  frame is cheap", that the committed part is rendered once but
  `Render`'s frame is a new string each version, and that a host drawing
  the frame's last rows takes `RenderParts`; the `parts` excerpt.
  `scripts/go-examples.py --check-guide` checks it.
* `docs/guides/migrating.md`, "Migrating to `v2.12.0`": `RenderParts`
  added; nothing changed in output; the `Printer`'s, the `Writer`'s and
  the reprint's allocations fewer, and a finished reply's or document's
  heap smaller, with the MADR's numbers.
* `README.md`: the current release, and a sentence in "Streaming
  content".
* **The release,** `v2.12.0` or the owner's name, by 0023-PLAN's rule 4:
  * the owner's commit; the disclosure guard over the outgoing commits;
    the owner's push, CI and Docs green; the owner's annotated tag, CI
    green on it;
  * a smoke test from the proxy: a scratch module, `GOWORK` empty, that
    requires the tag, streams a document into a `Doc` and checks at each
    chunk that `RenderParts` joined is `Render`, and streams the reply
    through a `Printer` and a `Writer`; `go vet`, `go build` and its run
    pass, and its closure for Linux, macOS and Windows holds no Charm
    module but `x/ansi`;
  * its control against `v2.11.0` from the proxy, the requirement held
    there ~~with no `go mod tidy`~~ (*by deviation D5:* after `go mod
    tidy`, the requirement checked to be `v2.11.0` still): the build
    fails, `RenderParts` undefined.
* The MADR's status and this PLAN's `complete`; `docs/README.md`.

## Verification

* Each step's tests pass, and each budget was seen failing twice (rule
  10), both failures quoted.
* Every mutation of Steps 3 to 5 is killed by the test named, by an
  assertion, its failure quoted.
* The MADR's table holds in the tree: Step 2's benchmarks and the budgets
  at each step within 1 % of the MADR's bytes and allocations.
* `TestReplyBudget`, `TestReprintBudget` and `safetext`'s
  `TestAllocations` pass at every step.
* Every existing test, golden and example passes unchanged.
* The fuzz targets of Steps 4 and 5 ran 5 minutes each with no failure.
* `python3 -m tools.verify gates` passes on macOS, and through `remote
  linux` and `remote windows`, at Steps 3, 4 and 5.
* `make apicheck` against `v2.11.0`: 0 incompatible changes.
* The identifier scan of each step's diff is clean.
* CI is green on the pushed tree and on the tag, and the smoke test
  passes from the proxy, its control failing against `v2.11.0`.

## Rollout and Rollback

* One commit a step, by the owner (rule 7), so each change is reverted by
  its own commit. Steps 3 and 4 change no API; reverting either before
  the release restores today's code exactly, and their budget tests go
  with them.
* Step 5 adds `RenderParts`. Before the tag, its commit is reverted with
  its tests. After the tag, it stays (API convention 1); a fault in it is
  fixed forward in a patch release.
* Nothing persists: no file format, no state outside the process.

## Execution record

### Step 1: records

*2026-11-04.* The owner committed the MADR and this PLAN as proposed
(`4e1a51f`) and said "accept the madr, approved to proceed". The MADR is
`accepted`, with those words; this PLAN `in-progress`; `docs/README.md`'s
two rows. Records only. **Step 1 complete, 2026-11-04.**

### Step 2: the measurements in the tree

* **Deviation D1, 2026-11-04: the helpers had no caller.**
  * **Found,** at the pre-add check of Step 2 as written:
    `make pre-add-check FILES="stream/bench_test.go"` exited 2,
    `stream/bench_test.go:83:6: func perRun is unused (unused)` and
    `:98:6: func heldHeap is unused (unused)`, under each GOOS.
    `.golangci.yml` enables `unused` with `run.tests: true`; the helpers'
    first callers are Step 3's `TestHeldHeap` and Step 4's budgets.
  * **Resolutions offered:** each helper added with its first use,
    `heldHeap` in Step 3 and `perRun` in Step 4 (recommended); or Step 2
    merged into Step 3. A `nolint` was not offered: it is a workaround.
  * **Chosen by the owner,** 2026-11-04: "Add each helper with its first
    use". Steps 2 to 4 annotated above; no decision changes, so the MADR
    is not amended.

* **Deviation D2, 2026-11-04: the in-tree benchmarks keep the output.**
  * **Found,** measuring Step 2's benchmarks in the tree, `-count 5`:
    `BenchmarkReplyChunks` 39,392 B/op and 400 allocs/op, against the
    MADR's 36,800 and 396, +7.0 % in bytes, past this step's 1 %.
    `BenchmarkLongLine` 2,280,668 and 2,161 against 2,276,572 and 2,160,
    within it. On fresh scratch copies with this step's
    `stream/bench_test.go`, the cause: `chunked` collects the Printer's
    output in a builder, as `replyRender` does, where the MADR's probe
    discarded it: +2,592 bytes and +4 allocations to the reply, +4,096
    bytes and +1 allocation to the line, before and after the changes
    alike. With all three changes the in-tree benchmarks measure
    23,792 and 167, and 64,056 and 128, so Step 4's `TestChunkBudget`,
    23,552 bytes, would fail, and `TestLongLineBudget`'s 65,536 would
    hold by 2.3 %. With D3 alone, Step 3's tree: 34,016 and 398, and
    2,275,294 and 2,159.
  * **Resolutions offered:** keep the output, the in-tree numbers the
    baseline, Step 4's ceilings re-set from them (recommended); or
    discard the output, as the probe did, the ceilings as written.
  * **Chosen by the owner,** 2026-11-04: "Keep the output; re-set the
    ceilings". Step 4 annotated above; the MADR's table gains a dated
    note.
* **What Step 2 adds:** `stream/bench_test.go`: `chunked(src string,
  size int) string`, a `Printer` with `DefaultMarkup` at 100 columns fed
  `src` in chunks of `size` bytes, its output collected, then flushed;
  `BenchmarkReplyChunks`, the benchmark reply in 8-byte chunks;
  `longLine`, 4,060 bytes of one paragraph line and its line break; and
  `BenchmarkLongLine`, `longLine` in 4-byte chunks. No library code
  changes.
* **The baseline in the tree,** at `08f35fe` with this step, `-count 5`,
  each run's bytes and allocations:

  | Benchmark | B/op | allocs/op |
  | :--- | ---: | ---: |
  | `BenchmarkReplyRender` | 33,160 | 205 |
  | `BenchmarkReplyChunks` | 39,392 | 400 |
  | `BenchmarkLongLine` | 2,280,668 to 2,280,676 | 2,161 |
  | `BenchmarkWriter` | 30,600 | 221 |
  | `BenchmarkDoc` | 1,125,833,370 to 1,125,834,077 | 518,568 to 518,575 |
  | `BenchmarkReprint` | 6,550,587 to 6,550,598 | 44,480 |

  `BenchmarkReplyRender`, `BenchmarkWriter`, `BenchmarkDoc` and
  `BenchmarkReprint` are the MADR's, within 0.01 %.
* **Checks:** `make pre-add-check FILES="stream/bench_test.go"`: clean
  (gofmt, golangci-lint for each GOOS, go vet, go test, go mod tidy,
  govulncheck); the documents' checks and the identifier scan clean.
* **Step 2 complete, 2026-11-04.**

### Step 3: D3, no sanitizer kept by a finished reply

*2026-11-04.* The owner committed Step 2 (`8ef3ff3`) and said "committed,
proceed".

**Facts re-read** at `8ef3ff3`, by the same script as at `a8ff40f`: every
line cited in `stream/printer.go`, `stream/document.go`, `transcript/`
and `safetext/` is where it was; only `stream/bench_test.go`'s moved, by
Step 2's additions.

**What changed:**

* `stream/printer.go`: `NewPrinter` calls no `open`; `Write` calls `open`
  when `p.san` is nil; `Flush` closes and drops the pair when it has one,
  and calls no `open`. `open`'s documentation says when it runs.
* `stream/document.go`: `Finish` drops its pair after closing it.
* `stream/bench_test.go`: `heldHeap`, by D1, and `TestHeldHeap`, which
  logs each case's heap.
* `transcript/bench_test.go`: `transcriptHeap`, 20 MiB, and
  `TestTranscriptHeap`, skipped under `-short`.
* `stream/printer_test.go`: `TestFlushUnwritten`.

**Measured** (`go test -v`):

| Budget | Ceiling | At this step | At `8ef3ff3` |
| :--- | ---: | ---: | ---: |
| a kept reply's heap | 5,120 | 4,124 | 9,500 |
| a `Printer` never written | 3,072 | 2,095 | 7,471 |
| a finished `Doc` | 6,144 | 4,975 | 12,655 |
| the 4 MiB transcript | 20,971,520 | 19,046,608 | 43,528,896 |

Under `-race`: 4,124, 2,095, 4,975 and 19,047,936. `TestReplyBudget` and
`TestReprintBudget` pass, under `-race` too.

**Mutations** (`python3 -m tools.verify mutate`): 6 killed, 0 survived,
0 errors, each by an assertion:

* 3.1 `NewPrinter` calls `open`: `bench_test.go:188: a Printer never
  written holds 7471 bytes of heap; want at most 3072`.
* 3.2 `Flush` calls `open` after dropping the pair: `bench_test.go:188: a
  kept reply holds 9500 bytes of heap; want at most 5120`; and in
  `transcript`, `bench_test.go:106: the 4 MiB transcript holds 43527392
  bytes of heap; want at most 20971520`.
* 3.3 `Flush` keeps the closed pair: `printer_test.go:549: Source =
  "one\ntw"`.
* 3.4 `Finish` keeps its pair: `bench_test.go:188: a finished Doc holds
  12655 bytes of heap; want at most 6144`.
* 3.5 `Flush` with no nil check: `printer_test.go:527: Flush of a Printer
  never written: "", panicked true; want "" and no panic`.

**The budgets' second failure** (rule 10): this step's three test files on
a scratch export of `8ef3ff3`: `TestHeldHeap` fails each case, "a kept
reply holds 9500 bytes of heap; want at most 5120", "a Printer never
written holds 7471 … want at most 3072", "a finished Doc holds 12655 …
want at most 6144"; `TestTranscriptHeap`, "the 4 MiB transcript holds
43528896 bytes of heap; want at most 20971520". `TestFlushUnwritten`
passes there, as it must: that code opens at `NewPrinter`.

**Gates:**

* `make pre-add-check` on the five Go files: clean.
* `python3 -m tools.verify gates` on macOS: 22 passed, 0 failed, 0
  skipped in 855 s.
* `remote linux`: 21 passed, 0 failed, 1 skipped (`gofmt`, which a
  remote run is given no files for) in 1,198 s. `remote windows`: the
  same, in 2,881 s.
* The documents' checks and the identifier scan: clean.

**Step 3 complete, 2026-11-04.**

### Step 4: D2, the chunk path

*2026-11-04.* The owner committed Step 3 (`8f22f83`) and said "committed,
proceed".

**Facts re-read** at `8f22f83`: as the table says, at lines moved by
Step 3's additions (the Printer's `clean` field `:46`, `open`'s floor
`:90`, `draw`'s `:140`, `Idle` `:149`, `take` `:199-215` with the join at
`:203`); `stream/renderer.go:45` and `stream/view.go:273` unchanged.

* **Deviation D3, 2026-11-04: the race build allocates more.**
  * **Found,** running the step's budgets under `-race`, which CI runs:
    `TestChunkBudget` 182 allocations and 25,184 bytes against 176 and
    25,600; `TestLongLineBudget` 134 and 72,640 against 144 and 69,632.
    Without `-race`, 167 and 23,792, and 128 and 64,056. The race
    readings repeated exactly over three runs. A profile of
    `BenchmarkReplyChunks` under each build put the 15 allocations more
    in the race build in `strings.Builder` (`WriteByte`, `WriteString`,
    `Write`), `bytes.Buffer` (`grow`, `growSlice`, `String`) and
    `Style.Apply`'s `stream/style.go:76`, none a site of this step's: the
    race build compiles with other inlining and escape decisions. Before
    the change, under `-race`: 413 and 36,336, and 2,165 and 2,292,000.
    The heap budgets, `TestPlainAllocs`, `TestReplyBudget` and
    `TestReprintBudget` pass under `-race`.
  * **Resolutions offered:** a ceiling for each build, chosen by a
    constant a `//go:build race` file sets (recommended); or one ceiling
    for both, set from the race build. Skipping the budgets under
    `-race` was not offered: it is a workaround. Rule 10 forbids raising
    a ceiling to pass, so the plain build's stand.
  * **Chosen by the owner,** 2026-11-04: "A ceiling for each build".
    `stream/build_race_test.go` and `stream/build_norace_test.go` added
    to the step; Step 4 annotated above. No decision changes, so the
    MADR is not amended.

**What changed:**

* `stream/printer.go`: `open` makes the floor's sanitizer write into
  `&p.source`; `take` scans the Source from `p.at` and sets `p.partial`
  to `src[p.at:]`; `draw` writes its input into `q.source`; the `clean`
  field goes, with `Idle`'s term; `base`'s and `partial`'s field comments,
  `cleaned`'s documentation (now `Doc`'s) and the `nolint` comments say
  so.
* `stream/renderer.go`, `stream/view.go`: `q.source.WriteString(src)`.
* `stream/bench_test.go`: `perRun`, by D1; `TestChunkBudget`,
  `TestLongLineBudget` and `TestPlainAllocs`; `TestHeldHeap`'s kept reply
  at 3,584 bytes.
* `stream/build_race_test.go`, `stream/build_norace_test.go`: `raceBuild`,
  by D3.
* `transcript/bench_test.go`: `transcriptHeap` at 16 MiB.

**Measured** (`go test -v`; under `-race` where it differs):

| Budget | Ceiling | At this step | Under `-race` (its ceiling) | At `8f22f83` |
| :--- | ---: | ---: | ---: | ---: |
| `TestChunkBudget`, allocations and bytes | 176, 25,600 | 167, 23,792 | 182, 25,184 (192, 27,648) | 398, 34,016 |
| `TestLongLineBudget` | 144, 69,632 | 128, 64,056 | 134, 72,640 (144, 77,824) | 2,159, 2,275,283 |
| `TestPlainAllocs`, `Render` and `RenderView` | 22, 29 | 22, 29 | 22, 29 | 23, 30 |
| `TestHeldHeap`, a kept reply | 3,584 | 3,081 | 3,100 | 4,124 |
| `TestTranscriptHeap` | 16,777,216 | 14,384,656 | 14,384,624 | 19,046,624 |

`TestHeldHeap`'s other cases are Step 3's, 2,095 and 4,975.
`TestReplyBudget` and `TestReprintBudget` pass, under `-race` too.

**Mutations:** 4 killed, 0 survived, 0 errors, each by an assertion; and
4.1 twice more, in both builds, against both chunk budgets:

* 4.1 `p.partial = strings.Clone(src[p.at:])`: `bench_test.go:226: the
  long line in 4-byte chunks: 1143 allocations and 2263138 bytes; want at
  most 144 and 69632`. Against both budgets, plain: "the reply in 8-byte
  chunks: 282 allocations and 32097 bytes; want at most 176 and 25600",
  and the long line's as above; under `-race`: "297 allocations and
  33632 bytes; want at most 192 and 27648" and "1149 allocations and
  2271734 bytes; want at most 144 and 77824".
* 4.2 `drawWith` writes `strings.Clone(src)`: `bench_test.go:240: Plain:
  Render 23 and RenderView 29 allocations; want at most 22 and 29`.
* 4.3 each line printed with its line break: `TestStyleCases`,
  `printer_test.go:92: heading: "## Title\n" gives
  "\x1b[1m## Title\n\x1b[0m\n"; want "\x1b[1m## Title\x1b[0m\n"`, and its
  other cases.
* 4.4 the partial line lost: `TestStyleCases`, `printer_test.go:92: no
  trailing newline: "# T" gives ""; want "\x1b[1m# T\x1b[0m"`.

**The budgets' second failure** (rule 10): this step's `stream` and
`transcript` test files on a scratch export of `8f22f83`: every new or
tightened budget fails, at the "At `8f22f83`" column's values, each
"want at most" the ceiling above; and under `-race`, with the two build
files, `TestChunkBudget` "413 allocations and 36336 bytes; want at most
192 and 27648" and `TestLongLineBudget` "2165 allocations and 2292002
bytes; want at most 144 and 77824".

**Fuzz,** 5 minutes each, no failure, no file added to the tree:
`FuzzPrinter` 19,579,827 runs, `FuzzWriter` 8,854,534, `FuzzDoc`
7,884,402, `FuzzTranscript` 9,418,264.

**Gates:**

* `make pre-add-check` on the seven Go files: clean.
* `python3 -m tools.verify gates` on macOS: 22 passed, 0 failed, 0
  skipped in 838 s; `release-check` clean over 514 files, `apicheck`
  among it.
* `remote linux`: 21 passed, 0 failed, 1 skipped (`gofmt`) in 1,280 s.
* `remote windows`: the first run ended `remote: windows: exit=255` after
  its `vuln` gate, every gate before it passed (`race`, `shuffle` and
  `lc-all-c` among them), the ssh session lost; run again whole: 21
  passed, 0 failed, 1 skipped in 2,916 s.
* The documents' checks and the identifier scan: clean.

**Step 4 complete, 2026-11-04.**

### Step 5: D1, `Doc.RenderParts`

*2026-11-04.* The owner committed Step 4 (`a63eec5`) and said "committed,
proceed".

**Facts re-read** at `a63eec5`: `Render` at `stream/document.go:239`, the
cache's `:246`, the join at `:269`, the frame kept at `:271`, the
definition reset at `:319`, two widths at `:309`, the method `tail` at
`:339`: as the table says, 3 lines down by Step 3's comment in `Finish`.

* **Deviation D4, 2026-11-04: `ExampleDoc` was never run.**
  * **Found,** writing `ExampleDoc_RenderParts`: it was not run, and
    `go test -list Example ./stream/` listed only `ExamplePrinter` and
    `ExampleWriter`, at `a63eec5` too. Go checks an example's output only
    when its last comment group begins `Output:`; `// guide:end` directly
    above `// Output:` makes them one group, and the example is compiled
    and not run. A scan of the 28 examples: `ExampleDoc`
    (`stream/document_test.go:226`) and the new example alone; the five
    others not collected have no output comment, compile-only by design
    (`ExampleWait`, `ExampleFilter`, `ExampleRun`, `ExampleProber`,
    `ExampleWorkspace_program`; `ExampleFilter` had none before
    `72b2a95` either). `ExampleDoc` was run from `a9fcfaf`, which added
    it, until `72b2a95`, 0009-PLAN Step 8, which put its guide region
    around it. On a scratch export, a blank line between the two lines
    made it run, and it passed.
  * **Resolutions offered:** both examples fixed in this step, with a note
    in 0009-PLAN (recommended); or the new one alone, `ExampleDoc` left
    for an amendment of 0009.
  * **Chosen by the owner,** 2026-11-04: "Fix both in Step 5". A dated
    note is added to
    [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)'s
    Step 8 record. No decision changes.

**What changed:**

* `stream/document.go`: `partPrefix`, `partTail` and `joined` beside
  `frame`; `parts(width)`, today's `Render` body to the tail's render,
  keeping the parts; `Render` joining them once a version;
  `RenderParts(width) (prefix, tail string)`, documented as MADR D1.
* `stream/document_test.go`: `joinParts`, `TestRenderParts`,
  `ExampleDoc_RenderParts` (region `parts`), and D4's blank line in
  `ExampleDoc`.
* `stream/bench_test.go`: `docParts`, `BenchmarkDocParts` and
  `TestRenderPartsBudget`.

**The API:** `make apicheck` against `v2.11.0`: 0 incompatible changes.
`go doc ./stream Doc` lists `func (d *Doc) RenderParts(width int)
(prefix, tail string)`, which `v2.11.0`'s `stream/document.go` does not
name; it is the step's one exported addition, and Steps 3 and 4 added
none.

**Measured:** `TestRenderPartsBudget`, 37,568,624 bytes, at most
41,943,040; under `-race`, 37,963,024, within the same ceiling, so this
budget needs no race ceiling. `TestRenderParts` passes in 12 s, 40 s under
`-race`. `ExampleDoc_RenderParts` prints `"A first paragraph."` and
`"Then a second, whose first line is\nwhole,\nand whose next is still
stream"`, read from its run against a placeholder before it was written.
`go test -list Example ./stream/` now lists `ExampleDoc` and
`ExampleDoc_RenderParts`. Every earlier budget passes, under `-race` too.

**Mutations:** 6 killed, 0 survived, 0 errors, each by an assertion:

* 5.1 `RenderParts` returns `(d.Render(width), "")`: `bench_test.go:283:
  the document by RenderParts: 1123317624 bytes; want at most 41943040`,
  the budget's second failure by rule 10 as planned.
* 5.2 the parts kept across widths: `document_test.go:316: doc 0, size 1,
  at 40, width 40: RenderParts joined "A line of a paragraph in a
  streamed reply"; a twin's frame "A line of a paragraph in a
  streamed\nreply"`.
* 5.3 `joined` left set: `document_test.go:313: doc 0, size 1, at 1,
  width 80: Render "A"; RenderParts joined "A "`.
* 5.4 the finished frame as the tail: `document_test.go:326: doc 0, size
  1, finished, width 40: RenderParts "", "A line of a paragraph …"; want
  (Render, "")`.
* 5.5 the parts drawn again at one version: `document_test.go:319: doc 0,
  size 1, at 0, width 40: a second RenderParts allocates 11 times`.
* 5.6, by D4, `holdback` returning the partial line whole: `--- FAIL:
  ExampleDoc`, its got and want printed: the example runs.

**Fuzz:** `FuzzDoc`, 5 minutes, 8,355,794 runs, no failure, no file added
to the tree.

**Gates:**

* `make pre-add-check` on the three Go files: clean, the examples gate
  among it.
* `python3 -m tools.verify gates` on macOS: 22 passed, 0 failed, 0
  skipped in 962 s.
* `remote linux`: 21 passed, 0 failed, 1 skipped (`gofmt`) in 1,509 s.
* `remote windows`: 20 passed, 1 failed, 1 skipped in 2,880 s. The one
  failed was `markdownlint`, `exit=2` in 1 s: Node.js v26.13.0 threw
  `SyntaxError: Unexpected number` loading markdownlint-cli2's own
  modules (`compileSourceTextModule`), before any file was read; the step
  changes no Markdown, and macOS and Linux passed it. Run again alone
  through `remote windows -- --only markdownlint`: 1 passed, `Summary: 0
  issues in 0 files`. Every other gate passed in the full run, `race`,
  `shuffle`, `lc-all-c` and the examples gate among them.
* The documents' checks and the identifier scan: clean.

**Step 5 complete, 2026-11-04.**

### Step 6: the guide, the release and the close-out

*2026-11-04.* The owner committed Step 5 (`7818042`) and said "committed
proceed".

**Measured in the tree** at `7818042`, `-count 3`, the release's numbers:
`BenchmarkReplyRender` 24,840 B/op and 174 allocs/op;
`BenchmarkReplyChunks` 23,792 and 167; `BenchmarkLongLine` 64,056 and
128; `BenchmarkWriter` 21,336 and 162; `BenchmarkDoc` 1,123,324,403 to
1,123,327,573 and 507,659 to 507,688; `BenchmarkDocParts` 37,568,684 to
37,568,883 and 497,392 to 497,394; `BenchmarkReprint` 6,230,561 to
6,230,585 and 44,168. Heap: a kept reply 3,100 bytes, a Printer never
written 2,095, a finished `Doc` 4,975, the 4 MiB transcript 14,383,296.
The MADR's "All three" column, within its stated repeat.

**What changed:**

* `docs/guides/streaming-content.md`, "Feed a Doc": the bullet "A frame
  need not be copied", and after the list the `parts` excerpt with what
  it prints and when the prefix is drawn anew. The guide's excerpts
  checked: 20, the new one among them.
* `docs/guides/migrating.md`: "Migrating to `v2.12.0`", its numbers the
  tree's above.
* `README.md`: the current release `v2.12.0`, and a sentence in
  "Streaming content".

* **Deviation D5, 2026-11-04: the control needs `go.sum`.**
  * **Found,** rehearsing the smoke test (`smoke212.py local`, a scratch
    module outside the repository, the tree by a replace): the control
    against `v2.11.0` with no `go mod tidy` failed for another reason,
    `parts/main.go:13:2: missing go.sum entry for module providing
    package github.com/charmbracelet/x/ansi`, which the script refused as
    a pass. The rule against tidy was from `v2.10.0`'s control
    (0009-PLAN Step 8), whose program imported packages `v2.9.0` did not
    have, so tidy moved its requirement; here every package
    `./parts` imports is in `v2.11.0`.
  * **Resolutions offered:** `go mod tidy`, the requirement then checked
    to be `v2.11.0` (recommended); or no tidy, the missing `go.sum` line
    got by hand at the version `v2.11.0` selects.
  * **Chosen by the owner,** 2026-11-04: "Tidy, the requirement checked".
    Step 6 annotated above.

**The smoke test, rehearsed** (`smoke212.py local`): `go mod tidy`, `go
vet` and `go build` exit 0; `./parts`' closure for Linux, macOS and
Windows is the module, the root, `x/ansi`, `displaywidth`, `uax29`,
`go-colorful` and `go-runewidth`, no Charm module but `x/ansi`; "smoke:
PASS parts, 1224 frames by RenderParts, the reply through a Printer and a
Writer". Its control against `v2.11.0` from the proxy, after `go mod
tidy`, requires `v2.11.0` still, and fails as it must:
`parts/main.go:46:21: d.RenderParts undefined (type *stream.Doc has no
field or method RenderParts)`.

~~**Not done here:** the owner's commit; the disclosure guard; the owner's
push, CI and Docs on it; the annotated tag `v2.12.0` and CI on it; the
smoke test from the proxy, `smoke212.py proxy`; and the close-out. This
step stays open for them, and the PLAN `in-progress`.~~ Done, below.

**The release, `v2.12.0`:**

* **The push.** The owner committed the step (`bbfd52f`). The disclosure
  guard over the seven outgoing commits, `4e1a51f` to `bbfd52f`, exited 0
  with nothing to report, and the identifier scan against `origin/main`
  was clean; then the owner pushed `main`. CI run 41931958557 passed:
  `modules`, `gates`, and `test` on macOS 15, Ubuntu 24.04 and Windows
  2025. Docs run 41931958615 passed.
* **The tag.** The owner tagged `bbfd52f` `v2.12.0`, annotated, and
  pushed it. Its CI run, 41935275773, passed every job at its first
  attempt.
* **The smoke test from the proxy** (`smoke212.py proxy`, a scratch
  module outside the repository, `GOWORK` empty): `go get
  github.com/maccavelli/go-tui-lib/v2@v2.12.0`, `go mod tidy`, `go vet` and
  `go build` exit 0, the requirement `v2.12.0`; `./parts`' closure for
  Linux, macOS and Windows holds no Charm module but `x/ansi`; "smoke:
  PASS parts, 1224 frames by RenderParts, the reply through a Printer and
  a Writer".
* **The control,** the same program against `v2.11.0` from the proxy,
  after `go mod tidy` (D5), the requirement still `v2.11.0`: `go build`
  exit 1, `parts/main.go:46:21: d.RenderParts undefined (type *stream.Doc
  has no field or method RenderParts)`.
* **Step 6 complete, 2026-11-04.**
* **This PLAN is complete, 2026-11-04.** Its Verification holds: every
  budget seen failing twice, the race ceilings among them; every mutation
  of Steps 3 to 5 killed by an assertion, each failure quoted; the MADR's
  table held in the tree (Step 6's measurements); `TestReplyBudget`,
  `TestReprintBudget` and `safetext`'s `TestAllocations` passing at every
  step; every existing test, golden and example unchanged, and
  `ExampleDoc` checked again (D4); the fuzz targets 5 minutes each with no
  failure; the gates on macOS, Linux and Windows at Steps 3, 4 and 5;
  `make apicheck` 0 incompatible changes and one addition; the identifier
  scans clean; CI green on the push and the tag; the smoke test passed
  from the proxy and its control failed against `v2.11.0`. Five
  deviations, D1 to D5, each the owner's choice, are recorded above; none
  changed a decision of the MADR.
