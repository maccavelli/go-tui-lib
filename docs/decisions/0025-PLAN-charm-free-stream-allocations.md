---
status: in-progress
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
| 4 | `stream/printer.go`, `stream/renderer.go`, `stream/view.go`, `stream/bench_test.go`, `transcript/bench_test.go` | D2: the chunk path; its budgets |
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
    committed.
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
    there with no `go mod tidy`: the build fails, `RenderParts`
    undefined.
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
