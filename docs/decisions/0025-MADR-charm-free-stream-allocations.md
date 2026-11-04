---
status: proposed
date: 2026-11-04
decision-makers: the repository owner
consulted: "allocation profiles, escape analysis and scratch prototypes of stream, transcript and safetext at a8ff40f, measured by the agent"
informed: gobble-cli and pi-go, the library's consumers
---
# Draw a Doc's frame without copying its prefix, stream a Printer's chunks without copying the line again, and keep no sanitizer in a finished reply

## Context and Problem Statement

On 2026-11-04 the owner asked:

> assess the stream support in the cli code, evaluate it and find me 3
> optimizations we can make for consumers that do not import charm.

and, after the assessment:

> draft the stream optimization records, ensure the plan is accurate and
> reflects the current repo and codebase facts, and ensure the plan is
> actionable, detailed, and comprehensive. prove all assertions.

A consumer that does not import Charm uses `stream`'s cores directly:
the `Printer` and the `Writer` over it, a `Doc` drawn by a host's own
loop, and `transcript`, which keeps every reply's `Printer`.
[0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
A5 made `stream` a Charm-free core, and
[0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)
D7 gave the line mode its `Printer`. At `a8ff40f`, `go list -deps` of
`stream`, `transcript`, `frame`, `inline`, `lineedit`, `ask`, `pick`,
`safetext` and `termkey` names no Charm module but `x/ansi`.

This record decides three changes inside `stream` that cut what those
paths copy and keep, records the options that were prototyped and
refused, with what refused them, and names what is left out. Nothing a
program sees changes but one method added.

### How every number here was got

* **The tree:** `a8ff40f`, the tip of `main`, identical to
  `origin/main`. `v2.11.0` is the latest tag.
* **The copies:** each change was prototyped on a scratch export of
  `a8ff40f` (`git archive`), outside the repository; one export kept as
  the baseline, one for each change alone, and one with all three. In
  each, `go test ./...` of the root module passed, `internal/conformance`
  among it.
* **Benchmarks:** `GOWORK=off go test -run '^$' -bench … -benchmem
  -count 5`. Bytes and allocations an operation are what this record
  rests on; a benchmark's repeat to within 0.01 % from run to run, and a
  heap figure to within 0.6 % (a kept reply measured 3,099 to 3,113
  bytes). Times are the median of five, taken while the machine's load
  average was 11 to 24, and are indicative only.
* **Three benchmarks the tree does not have yet:** gobble's benchmark
  reply (`stream/testdata/reply.txt`, 921 bytes) in 8-byte chunks; a
  4,060-byte paragraph line in 4-byte chunks; and `BenchmarkDoc`'s
  document drawn by `RenderParts`.
  [0025-PLAN-charm-free-stream-allocations.md](0025-PLAN-charm-free-stream-allocations.md)
  Steps 2 and 5 add them, and Steps 3 to 5 turn the heap and allocation
  probes below into budget tests.
* **Heap held:** `runtime.MemStats.HeapAlloc` after `runtime.GC()`,
  before and after building 2,000 objects, divided by 2,000; and once
  for `transcript`'s `benchTranscript`, 4 MiB of kept replies
  (`transcript/bench_test.go`).
* **Where an allocation is made:** `-memprofile` at
  `-memprofilerate 1`, read with `go tool pprof -list`; escape analysis
  with `go build -gcflags=-m`.

### Finding 1: a `Doc`'s frame copies the whole committed prefix

* `Render` caches the committed segments' renders a width, appending
  each new segment once (`stream/document.go:243`, `:274-281`), and
  renders only the tail each frame (`:257`). It then joins them:
  `d.frame = prefix + "\n\n" + tail` (`:266`), a new string of the whole
  frame, each time the version has moved.
* **The cost:** in `BenchmarkDoc` (200 KB in 20-byte chunks, a frame
  after each, `stream/bench_test.go:62-72`), line `:266` allocates
  1.01 GB of the 1.05 GB the run allocates, 96 %, by `pprof -list`. A
  frame's cost so grows with the document, and a document's with its
  square.
* **Who pays it:** a host that draws frames itself, as the streaming
  guide's scheduler loop does (`frame/example_test.go:34`,
  `_ = d.Render(40)`). Such a host shows a window of rows, an inline
  region or a pane, and needs the frame's last rows, not a fresh copy of
  all of it. `stream/streamtea`'s `DocPane` calls `Render` too
  (`stream/streamtea/pane.go:181`); it is a Bubble Tea component, outside
  this record.

### Finding 2: a `Printer` copies the partial line again at every chunk

* `take` copies what the sanitizers wrote from the Printer's `clean`
  buffer into its Source (`stream/printer.go:193-195`), and then joins
  the partial line held from the last chunk to it,
  `s = p.partial + s` (`:196`), so the whole incomplete line is copied
  again at each chunk. A line streamed in small chunks costs the square
  of its length.
* Each draw of a Source at a width (`At` and `Printed` through `draw`,
  `:138`; `Plain`'s `Render` and `RenderView`, `stream/renderer.go:45`
  and `stream/view.go:273`) writes its input into `clean`, and `take`
  copies it into the Source again.
* **The cost:** in `BenchmarkReplyRender`'s profile, by line, `:196`
  makes 39,158 of 537,403 allocations, 7.3 %, more than any other line of
  the library's own code; only `strings.Builder`'s methods, which every
  caller shares, and the benchmark's own line make more. A 20-byte
  `Write` takes 2.964 allocations; the 4,060-byte line in 4-byte chunks
  allocates 2,276,572 bytes.
* **Not a cost:** `p.san.Write([]byte(chunk))` (`:153`) and `Doc`'s
  `d.san.Write([]byte(s))` (`stream/document.go:110`) copy nothing:
  escape analysis reports "zero-copy string->[]byte conversion" at both.
  The assessment before this record said the conversion copies each
  chunk; it does not, and that claim is withdrawn.

### Finding 3: a finished reply keeps sanitizers it never uses again

* A `safetext.Sanitizer` holds its parser's state by value
  (`safetext/sanitizer.go:18`), among it two fixed buffers, `csi
  [sgrMax]byte` and `osc [oscMax]byte` (`safetext/clean.go:204`, `:209`;
  128 and 2,048 bytes, `:172`, `:184`): 2,424 bytes a `Sanitizer` by
  `unsafe.Sizeof`. Under the default rules, `safetext.Text()`, which
  keep no SGR (`:229`), neither buffer is read: a CSI's parameters are
  read only after `dispatchCSI`'s `!m.keepSGR` return (`:732-736`), and
  an OSC's body is written only while `oscKeep` holds, which only
  `keepSGR` sets (`:684`).
* `NewPrinter` makes two sanitizers (`stream/printer.go:82`, `open` at
  `:87-89`), and every `Flush` closes them and makes two more (`:166`),
  which a finished reply never writes to. A `Transcript` keeps every
  reply's `Printer` (`transcript/transcript.go:43`, `:92`). `Doc.Finish`
  closes its two and keeps them (`stream/document.go:126-133`).
* **The cost:** a kept reply, gobble's benchmark reply written and
  flushed, holds 9,512 bytes of heap for 921 bytes of source; the 4 MiB
  `benchTranscript` holds 43,550,976 bytes; a finished `Doc` of that
  reply, drawn once, 12,646; a `Printer` never written, 7,462.

### Measured

Each row is the baseline, `a8ff40f`, then each change alone, then all
three; bytes and allocations an operation, and heap held. D1 changes no
`Printer` path and D2 and D3 no frame join, so the columns add as the
rows show.

| Measure | `a8ff40f` | D1 | D2 | D3 | All three |
| :--- | ---: | ---: | ---: | ---: | ---: |
| `BenchmarkReplyRender`, 64-byte chunks: B/op, allocs/op | 33,160, 205 | 33,160, 205 | 30,216, 176 | 27,784, 203 | 24,840, 174 |
| `BenchmarkWriter`: B/op, allocs/op | 30,600, 221 | 30,600, 221 | 26,712, 164 | 25,224, 219 | 21,336, 162 |
| The reply in 8-byte chunks: B/op, allocs/op | 36,800, 396 | 36,800, 396 | 26,576, 165 | 31,424, 394 | 21,200, 163 |
| A 4,060-byte line in 4-byte chunks: B/op, allocs/op | 2,276,572, 2,160 | 2,276,572, 2,160 | 65,336, 129 | 2,271,196, 2,158 | 59,960, 127 |
| A 20-byte `Printer.Write`: mallocs | 2.964 | 2.964 | 0.984 | 2.964 | 0.984 |
| `BenchmarkDoc` by `Render`: B/op, allocs/op | 1,125,833,192, 518,567 | 1,125,833,416, 518,570 | 1,123,324,409, 507,658 | 1,125,833,696, 518,574 | 1,123,324,637, 507,665 |
| The same document by `RenderParts`: B/op, allocs/op | — | 40,078,168, 508,304 | — | — | 37,568,770, 497,393 |
| `Plain`'s `Render`, `RenderView` of a short document: allocs | 23, 30 | — | 22, 29 | 23, 30 | 22, 29 |
| `BenchmarkReprint`: B/op, allocs/op | 6,550,594, 44,480 | 6,550,594, 44,480 | 6,230,093, 44,167 | 6,550,935, 44,481 | 6,230,599, 44,168 |
| Heap a kept reply holds, bytes | 9,512 | 9,513 | 8,489 | 4,137 | 3,110 |
| Heap the 4 MiB `benchTranscript` holds, bytes | 43,550,976 | 43,550,864 | 38,898,784 | 19,068,688 | 14,400,424 |
| Heap a finished `Doc` holds, bytes | 12,646 | 12,648 | 12,648 | 4,966 | 4,966 |
| Heap a `Printer` never written holds, bytes | 7,462 | 7,462 | 7,462 | 2,081 | 2,086 |

* **Times,** the median of five, indicative under the load above: the
  4,060-byte line 427 µs at `a8ff40f`, 127 µs with all three;
  `BenchmarkDoc`'s document 189 ms by `Render` at `a8ff40f`, 50 ms by
  `RenderParts` with all three. The rest are within the noise.
* **D3's one allocation more in `BenchmarkReprint`** (44,480 to 44,481)
  is not D3's code. The profiles' difference is `fmt.newPrinter` and
  `sync.(*Pool).pinSlow` under `msgcat.Format`, the notice's formatting:
  with less garbage, the collector runs at other moments and `fmt`'s
  printer pool is refilled at others. D3's own difference is 9,108 fewer
  `safetext.NewSanitizer` calls over the run, from `Flush`.
* **The budgets hold** with all three: `TestReplyBudget`, 257
  allocations (`stream/bench_test.go:90`, gobble-cli's, 0023-MADR A9
  item 5), and `TestReprintBudget`, 49,696 allocations and 7,274,618
  bytes (`transcript/bench_test.go:56-58`).

### The question

Can these paths copy and keep less, with no change to what any of them
prints or draws, no new default, and additions only to the API?

## Decision Drivers

* **Output byte for byte as today.** Every existing test, golden file
  and example passes unchanged.
* **API conventions** (AGENTS.md): additions only, so a minor release;
  `make apicheck` reports 0 incompatible changes against `v2.11.0`.
* **0020-MADR D9's bar holds:** clean input through `safetext.Clean`
  allocates nothing, cleaning allocates once, and `Sanitizer.Write`
  allocates nothing in the steady state
  ([0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md),
  `:737-739`), as `safetext`'s `TestAllocations` holds it
  (`safetext/bench_test.go:31-54`).
* **The existing budgets hold:** `TestReplyBudget` and
  `TestReprintBudget`.
* **Each saving is held by a test** that fails on the code before the
  change.
* **No default added** (API convention 12): none of these is a
  behaviour a program could want to refuse.

## Considered Options

* **D1, the frame:**
  * D1-A: `Doc.RenderParts(width) (prefix, tail string)`, the frame's two
    parts, cached as `Render` is.
  * D1-B: `Doc.RenderTo(w io.Writer, width int)`, the frame written to a
    writer.
  * D1-C: keep `Render` alone, and document its cost.
* **D2, the chunk path:**
  * D2-A: the `Printer` reads its lines from its Source; the floor's
    sanitizer and every draw write into the Source.
  * D2-B: D2-A, and `safetext.Sanitizer.WriteString`, for `Write`'s
    `string` chunk.
  * D2-C: keep the chunk path.
* **D3, the sanitizers a finished reply keeps:**
  * D3-A: a `Printer` opens its sanitizers at its first `Write` after
    `NewPrinter` or a `Flush`, and drops them at the `Flush`; `Doc.Finish`
    drops its own.
  * D3-B: shrink the sanitizer: its CSI and OSC buffers made only when
    its rules keep SGR.
  * D3-C: `Sanitizer.Reset`, so that `Flush` reuses the two it has.
  * D3-D: keep the sanitizers.

## Decision Outcome

Chosen options: "D1-A", "D2-A" and "D3-A", because each removes a cost
the profiles show, changes no output, keeps 0020-MADR D9's bar and the
existing budgets, and adds one method between them. The prototypes of all
three pass every test of the root module.

### D1. `Doc.RenderParts`

```go
// RenderParts is Render's frame in its two parts, not joined: prefix, the
// committed segments' renders, and tail, the tail's. Render(width) is
// prefix and tail joined by one blank line, either alone when the other
// is "". After Finish, prefix is the whole frame and tail is "". Like
// Render, it is cached by width and version; a second call at the same
// width and version allocates nothing.
func (d *Doc) RenderParts(width int) (prefix, tail string)
```

* **The cache:** the parts are kept by width and version, where
  `Render`'s frame is kept today (`stream/document.go:268`). `Render`
  joins them on its first call at a version and keeps the joined frame,
  so its own cost, its output and its cache are as today.
* **The prefix is not copied:** it is `widthCache`'s builder's string,
  which `strings.Builder` gives without a copy and never changes after
  (`strings/builder.go` in Go 1.27.3: `String` is a view of the buffer,
  `:47`; `WriteString` only appends, `:114`; `Reset` drops the buffer,
  `:61`). It is safe to keep, as any Go string is.
* **What a host does with it:** draws the frame's last rows from the
  tail and, when the tail is short, the end of the prefix, with no copy
  of the rest. `Render` stays for a host that wants one string.
* **What it does not promise:** that the prefix only grows. It is
  drawn anew when a reference definition is committed
  (`stream/document.go:316-318`, 0009-PLAN D16), at a width the cache has
  dropped (two are kept, `:306`), after `SetRevision` or `Reset`, and is
  the whole frame after `Finish`. A host that printed the prefix once
  into its scrollback and redrew only the tail would need that promise;
  it is not decided here.
* **The name** is new in `stream`, a method, and clashes with nothing in
  [docs/glossary.md](../glossary.md); the glossary governs type names,
  and no type is added.

### D2. The chunk path reads its lines from the Source

* The floor's sanitizer writes into the Source itself
  (`safetext.NewSanitizer(&p.source, floor)`; `*strings.Builder` is an
  `io.Writer` whose `Write` never fails).
* `take` scans the Source from `p.at`, the partial line's start, which
  is already its offset there (`stream/printer.go:202-204`), and sets
  `partial` to `src[p.at:]`, a view and not a copy; nothing joins the
  partial line to the next chunk.
* `draw`, `Plain`'s `drawWith` and `viewWith` write their input into the
  new Printer's Source.
* The Printer's `clean` field goes (`:46`), and with it `Idle`'s
  `p.clean.b.Len() == 0` (`:147`), which is 0 at every return from
  `Write` and `Flush` today, since `take` empties `clean` each time it
  runs. The type `cleaned` stays: `Doc` uses it (`stream/document.go:57`).
* `Write` keeps `p.san.Write([]byte(chunk))`, which copies nothing.

### D3. A finished reply keeps no sanitizer

* `NewPrinter` makes none; `Write` calls `open` when the Printer has
  none; `Flush` closes and drops them, when it has them, and makes no
  more. A `Write` after a `Flush` starts afresh, as `Flush`'s
  documentation says today: a new pair is made then, as one is made at
  the `Flush` today.
* `Doc.Finish` drops its two after closing them. `Append` after `Finish`
  does nothing and `Write` returns `ErrClosed` before either touches
  them (`stream/document.go:106-107`, `Write`'s `finished` check);
  `Reset` makes a new pair (`:138-141`).

### Consequences

* Good, because a host drawing a `Doc`'s frame itself stops paying a
  copy of the document each frame: 28 times fewer bytes for the
  benchmark's 200 KB document by D1 alone, 30 times with all three.
* Good, because a reply streamed in small chunks costs in proportion to
  its length, not its square: 38 times fewer bytes for the 4,060-byte
  line.
* Good, because a kept reply holds a third of the heap it does today, and
  the default 4 MiB transcript of the benchmark reply 14.4 MB in place
  of 43.6 MB.
* Good, because the `Writer`, the reprint and `Plain` gain with the
  `Printer`, with no change of their own.
* Neutral, because `Render`'s own cost is unchanged: only a host that
  calls `RenderParts` gains D1.
* Neutral, because a `Printer` made and written again after each
  `Flush` makes its sanitizers at the `Write` in place of the `Flush`:
  the same number.
* Bad, because the `Printer`'s `partial` is a view of its Source, so the
  Source's buffer may be held a while longer by a string the Printer no
  longer needs; the Source is kept for the reply's life anyway
  (`Source`, `stream/printer.go:98`), so nothing more is held.

### Confirmation

* [0025-PLAN-charm-free-stream-allocations.md](0025-PLAN-charm-free-stream-allocations.md)
  adds the three benchmarks above, and for each change a budget test
  that a mutation reverting the change fails, by assertion.
* `TestRenderParts` holds `RenderParts` joined to `Render` and to a twin
  `Doc` drawn only at the one width, after every chunk of documents with
  a relaxed cut, a fence, a table and a definition committed midway, at
  three widths at each version, through `SetRevision` and `Finish`; and
  a second call at the same width and version to 0 allocations. A
  prototype of it passed on the copy with all three changes, and four
  mutations of the cache failed it (the PLAN's "What the prototypes
  showed").
* Every existing test, golden and example passes unchanged;
  `FuzzPrinter` and `transcript`'s fuzz target run 5 minutes each.
* `make apicheck` against `v2.11.0`: 0 incompatible changes, one
  addition.

## Pros and Cons of the Options

### D1-A: `Doc.RenderParts`

* Good, because the frame's parts are already strings the `Doc` holds:
  returning them copies nothing.
* Good, because `Render` keeps its meaning and its cost, and is `RenderParts`
  joined, which a test holds.
* Neutral, because a host that wants one string still calls `Render`.
* Bad, because it is one method more to learn.

### D1-B: `Doc.RenderTo(w io.Writer, width int)`

* Good, because it allocates no frame string.
* Bad, because it still writes every byte of the prefix each frame, the
  O(document) cost D1 removes, now as I/O.
* Bad, because a host showing only the last rows cannot tell the writer
  where to start without the parts.

### D1-C: keep `Render`, document its cost

* Good, because nothing is added.
* Bad, because a host has no way around the copy: a Go string is
  immutable, so a frame that holds the prefix and a new tail is a new
  string of both.

### D2-A: lines read from the Source

* Good, because it removes the square-law copy and the draw's second
  copy, with no API change.
* Good, because the Source already holds every byte the scan needs, at
  the offsets the Printer keeps.
* Neutral, because `partial` becomes a view of the Source (Consequences).

### D2-B: D2-A and `Sanitizer.WriteString`

* Bad, because it saves nothing: the conversion it would avoid copies
  nothing today (Finding 2).
* Bad, because its prototype, `Write` and `WriteString` each a wrapper
  over one generic `write[T string | []byte]`, made `Doc.Append`'s
  `[]byte(s)` escape: `pprof -list` showed 30,738 allocations at
  `stream/document.go:110` over three `BenchmarkDoc` runs, one a chunk,
  which the `[]byte` callers would pay.

### D2-C: keep the chunk path

* Bad, because a long line in small chunks keeps costing its square.

### D3-A: open at the first `Write`, drop at `Flush` and `Finish`

* Good, because a finished reply or document holds no parser state, and
  `Flush` makes no sanitizers a finished reply never uses.
* Good, because it is inside `stream`, so `safetext` and its bar are not
  touched.
* Neutral, because a live, streaming `Printer` holds its two, as today.

### D3-B: shrink the sanitizer

* Good, because every `Sanitizer` would shrink from 2,424 bytes to 296,
  by `unsafe.Sizeof` in the prototype.
* Bad, because it breaks 0020-MADR D9's bar. As slices made in `init`
  when SGR is kept, `TestAllocations` failed under `StatusLine`, which
  keeps it: `Clean` 2 or 3 allocations where 0 or 1 are wanted, and
  `IsClean` 2 where 0 is. As slices of buffers on `Clean`'s and
  `IsClean`'s stack, escape analysis reported "moved to heap: b" at both
  declarations, an allocation more in every `Clean` and `IsClean`,
  under every rule, `TestAllocations` failing under all four presets.
  `-gcflags=-m=2` traces the flow: the buffers into the parser
  (`m ← &b`), then the parser's contents to the heap through `run`'s
  generic call (`{heap} ← *{temp}`, at `run[go.shape.string](…, &m, s)`);
  sixteen of the parser's methods are reported "leaking param content:
  m". Why each leaks was not established. Assigning the slices in `Clean`
  and `IsClean` themselves, not in `init`, changed nothing. Keeping a
  pointer in the parser on the stack would mean changing what those
  methods do with it.
* Bad, because D3-A already removes what a finished reply holds.

### D3-C: `Sanitizer.Reset`

* Good, because `Flush` would make no new sanitizers.
* Bad, because a kept reply would still hold its two: 5,375 bytes of the
  9,512, the difference D3-A makes (9,512 to 4,137), each `Sanitizer`'s
  2,424 bytes held in a larger size class.
* Bad, because it adds a method to `safetext` for what `stream` can do
  alone.

### D3-D: keep the sanitizers

* Bad, because a transcript at its default bound holds about three times
  the heap its replies need.

## More Information

### Left out, with the evidence

* **`Doc`'s own `clean` buffer.** `Doc.Append` takes 1.006 allocations a
  20-byte chunk, the buffer regrowing after `take` empties it
  (`stream/document.go:184-189`; `(*cleaned).Write` in the profile,
  30,731 of `BenchmarkDoc`'s 1,449,517 allocations over three runs,
  2.1 %). D2's way, the floor writing into the Source, would remove it;
  it is left for a record of its own, since `take`'s version and line
  bookkeeping would change with it.
* **`DocPane` calling `RenderParts`.** `stream/streamtea` is a Bubble Tea
  package (`stream/streamtea/pane.go:181`), outside this record's
  consumers.
* **The `Writer`'s `string(b)`** (`stream/writer.go:38`) does not escape,
  by escape analysis, and gains with the `Printer`: nothing of its own
  is decided.

### Records

* [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
  §4, the `Doc` and its `Render` "cached by (width, version)", and A5.
* [0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  D9, `safetext` and its allocation bar.
* [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)
  D7, the `Printer`; A9 item 5, the reply and reprint budgets; A15,
  `Printed`.
* [0025-PLAN-charm-free-stream-allocations.md](0025-PLAN-charm-free-stream-allocations.md),
  the implementation.
