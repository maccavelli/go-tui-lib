---
status: in-progress
date: 2026-11-01
associated-madr: "0009-MADR-streaming-content-engine.md"
---
# Implement the streaming content engine

Associated MADR: [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)

*Revised 2026-10-02.* The owner answered the MADR's Q1–Q5, each with its
recommendation, and the MADR is accepted. This PLAN stays proposed until the
owner approves execution. MADR amendment A1, which is proposed, adds Steps
A1.1 to A1.4. They run after Step 7 and before Step 8, and only once A1 is
accepted. Its questions Q6–Q8 decide where three of its types live.

*Later on 2026-10-02:* the owner accepted A1 and answered Q6–Q8 with the
recommendations: `Interpret` in `safetext`, `Writer` and `Upgrader` in
`frame`, and `FitTable` in `stream`. Steps A1.1 to A1.4 are in scope, after
Step 7 and before Step 8.

*Revised again 2026-10-02, for MADR amendment A2 (proposed).* The owner
decided that the glamour adapter is a nested module
([0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)).
Step 7 now creates `stream/glamourmd` as its own module, with its own
`go.mod`, after the root release that carries every other step. The order
is therefore Steps 1–6, A1.1–A1.4 and 8, the root tag, and then Step 7; the
earlier notes' "A1.1 to A1.4 run after Step 7" is superseded. The root
`go.mod` never gains glamour. The scope table, Step 7, Verification and
Rollout changed to match. Step 7 waits for A2 and 0010-MADR to be accepted,
and for the tooling steps of
[0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md)
to be complete.

*2026-10-03:* the owner accepted A2, and approved
[0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md),
whose Phases 2–6 run after `v0.1.5`. Nothing in this PLAN's steps
changes. Step 7 still waits for 0010-PLAN's tooling phases. This PLAN
stays proposed until the owner approves its execution.

*2026-10-07, for MADR amendment A3:*

* [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md)
  is complete, so Step 7's precondition is met.
* The theme revision that Step 7's renderer cache and the code caches
  are keyed by is the MADR's owner question Q9. It is answered before the
  first step that builds one.
* `workspace/render.go`'s `local` is now at `:460`.

*2026-10-23, for
[0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)
D9:* Step 2 (`safetext`) is carried out by
[0020-PLAN-charm-free-cores-glyphs-and-safetext.md](0020-PLAN-charm-free-cores-glyphs-and-safetext.md)
Step 4, with the names of 0009-MADR A4 (`Rules`, `Sanitizer`,
`NewSanitizer`) and 0020-MADR D9's decisions: what is removed is at least
what `internal/sanitize` removes, invalid UTF-8 is one U+FFFD per maximal
subpart, `Flatten` collapses runs, `Escape` keeps what `Tabs` and
`Newlines` keep, `IsClean` is added, and `internal/sanitize` wraps it.
`glyph.Set.Control` is added there too. Step A1.1 here then adds
`Interpret` and nothing else. This PLAN stays proposed; its Step 2 is
not run from here.

*2026-10-28, for
[0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
A5, by
[0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)
D7 and A6:* `stream` is a Charm-free core. Step 5 builds `Doc`, `Renderer`,
`RendererFunc`, `Plain` and the cut scanner in `stream`, on the block
scanner
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 5 builds there first, unexported, and exports it as `Scanner` once
`Doc` uses it; `Buffer` and `Doc.Pull` are built in `stream/streamtea`, and
so is Step 6's `Pane`, which imports `workspace`, `frame` and Bubble Tea.
`stream` joins `TestCoresAreCharmFree` in 0023-PLAN's Step 5, before this
PLAN's Step 5 runs. This PLAN stays proposed.

*2026-11-01, for
[0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
A6:* the pair re-read against the tree at `114bee8`. The section "Order
and changes, as revised on 2026-11-01" under Implementation Steps is how
the steps below now read. *Later on 2026-11-01:* the owner answered Q9 to
Q12 with the recommendations, and A6 is accepted: `frame` and
`inputfilter` are cores with adapters, `Interpret`'s row is
`InterpretRow`, the order is the revised one, and `theme.New` stamps an
ID. In short:

* **Done elsewhere:** Step 2, and A1.1 but for `Interpret`, by
  [0020-PLAN-charm-free-cores-glyphs-and-safetext.md](0020-PLAN-charm-free-cores-glyphs-and-safetext.md)
  Step 4 (`v1.13.0`); A5's split and `stream`'s `Printer`, by
  [0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
  Step 5 (`v2.6.0`).
* **The names are A4's** throughout, with A6 item 4's option names and
  Q11's for `Interpret`'s row, which collides with `ask.Row`.
* **The tools are today's:** Go 1.27.2, `scripts/go-precheck.py`,
  `python3 -m tools.verify`, and the defaults inventory and message
  catalogue of 0024-MADR.
* **Facts moved:** `local` is at `workspace/render.go:538-556`;
  ultraviolet joins input split within 50 ms, which Step A1.3 measures
  before building its reassembly.

*Rewritten later on 2026-11-01, for
[0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
A6 as committed (`6365db5`) and A7 (proposed):* the Goal, Scope, rules,
Implementation Steps, Verification and Rollout below replace the text
those notes annotated, so that each step can be run as written. The text
they replace, with every note it carried, is this file at `6365db5`. Each
step keeps its number, since records cite them (0010-MADR Step 7,
0020-MADR Steps 2 and A1.1, and 0009-MADR Steps 7 and A1.1 to A1.4); the
steps run in the order the
Scope table gives. Where a step names a type, it uses A4's, A6's and A7's
names, and `Interpret`'s package is `stream`, Q13's answer.

## Goal

Build what 0009-MADR decides and is not yet built, as amended to A7:

* `frame` and `inputfilter`, Charm-free cores, each a pure layer and a
  goroutine-backed driver over it (A6 item 8), with their adapters
  `frame/frametea` and `inputfilter/inputfiltertea`;
* `stream`'s `Scanner`, `Doc`, `Renderer`, `Plain`, `Buffer`,
  `Interpret` (Q13) and A1's additions, with the test kit
  `stream/streamtest`;
* `stream/streamtea`'s `DocPane`, keyed to `theme.Theme.ID` (A6 Q9);
* the workspace's extension point for mouse messages (§6);
* `stream/glamourmd`, a nested module (A2).

So that a TUI streams an agent's Markdown into a workspace pane at
O(delta) cost a chunk, with an exact final frame, with nothing from the
stream reaching the terminal as a control sequence, and with wheel floods
and typed pastes coalesced before they queue; and so that a line-mode host
paces its output, feeds a `Doc` from another goroutine and detects a paste
with no Bubble Tea.

Done means every item under Verification holds, CI is green on each
pushed step, the owner has tagged the root's minor release and the
adapter's `stream/glamourmd/v0.1.0`, and each release's consumer smoke
test has passed from the proxy.

## Scope

### Prerequisites

* 0009-MADR A7 accepted, with Q13 answered (done, 2026-11-01), and this
  PLAN approved by the owner.
* Done already: Step 2 (`safetext`, by
  [0020-PLAN-charm-free-cores-glyphs-and-safetext.md](0020-PLAN-charm-free-cores-glyphs-and-safetext.md)
  Step 4, `v1.13.0`); `stream` a core, with `Printer`, `Markup` and
  `Style` (by
  [0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
  Step 5, `v2.6.0`); the nested-module tooling (by
  [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md)).

### In scope, in the order the steps run

| Order | Step | Packages and paths | What |
| :--- | :--- | :--- | :--- |
| 1 | 1 | `docs/decisions/0009-*`, `docs/README.md`, `docs/guides/defaults.md`, `docs/glossary.md` | records, the planned defaults, the reserved names |
| — | 2 | `safetext/`, `glyph/` | done by 0020-PLAN Step 4 |
| 2 | A1.1 | `stream/` | `Interpret`, `InterpretOptions`, `InterpretRow` (Q13) |
| 2a | A8.1 | `stream/` | `stream.Writer`, a plain streaming writer (MADR A8; *waits for the owner's approval*) |
| 3 | 3 | `frame/`, `frame/frametea/`, `internal/conformance/`, *`.golangci.yml` (D5)* | the pacing core, the scheduler, and their adapter |
| 4 | A1.2 | `frame/`, `frame/frametea/` | `Coalescer`, `Demand`, ~~`Writer`~~ `Output` (Q14), `Upgrader` |
| 5 | 4 | `inputfilter/`, `inputfilter/inputfiltertea/`, `internal/conformance/`, `workspace/render.go`, *`.golangci.yml` (D5)* | the wheel, motion and paste-burst cores, their driver, the filter, and `local`'s extension point |
| 6 | A1.3 | `inputfilter/`, `inputfilter/inputfiltertea/` | A1's input stages, measured first |
| 7 | 5 | `stream/`, `stream/streamtest/` | `Scanner` out of the `Printer`; `Doc`, `Renderer`, `Plain`, `Buffer`, `Doc.Pull`; `CheckRenderer` |
| 8 | 6 | `theme/`, `stream/streamtea/` | `Theme.ID`; `DocPane` |
| 9 | A1.4 | `stream/`, `stream/streamtea/` | `RenderView`, the dialect, `Highlighter`, `FitTable`, `Bounded` |
| 10 | 8 | examples, `docs/`, `README.md` | the guide and the root's minor release |
| 11 | 7 | `stream/glamourmd/`, `go.work`, `docs/`, `docs/glossary.md`, `internal/conformance/` | the glamour adapter, a nested module, and its release |

* The root `go.mod` gains no requirement in any step. `stream/glamourmd`'s
  `go.mod` requires the root's tag and `charm.land/glamour/v2`, in Step 7
  alone (A2).
* Each new core joins `cores` in `internal/conformance/charmfree_test.go`
  in the commit that builds it; an adapter does not. `stream`'s allowed
  imports grow by `frame` in Step 5 (A7 item 9).
* Each step adds its defaults' rows to `docs/guides/defaults.md`, turning
  Step 1's `planned` rows `built`, and its message IDs and English
  defaults, through `msgcat`, where it shows a user text.

### Out of scope

* A virtualized transcript of many `Doc`s, watermark pruning, and inline
  scrollback with OSC 133; the line mode's `transcript` is 0023's.
* A syntax-highlighting lexer: A1.4 adds the `Highlighter` interface only,
  and a lexer needs a dependency record of its own.
* Re-sending the mouse modes on focus-in, the planned `termmode` record's.
* Images, and the terminal services of
  [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md).
* Moving the tree to Bubble Tea `v2.1.x` or `x/ansi` `v0.11.9` (A6 item
  2): a change of its own.
* Any change in pi-go or gobble-cli.
* `git push` and tags, which are the owner's.

## Rules for every step

1. **One step, one commit, the owner's.** The agent stops at the end of
   each step with its evidence in the Execution Record, stages nothing,
   and commits nothing; the owner commits with `git commit --no-edit` and
   pushes. The next step starts after the owner's "proceed".
2. **Facts first.** A step re-reads, at the tree it starts on, each fact
   it names under "Facts re-read", and records a change as a dated
   deviation, with the owner's choice, before any code.
3. **Names first.** `python3 -m tools.verify names` over the step's
   planned type names reports no clash before any code.
4. **Every new test is seen failing.** Each mutation the step names runs
   through `python3 -m tools.verify mutate` on scratch copies, and must
   fail the test it names; the failure is quoted in the record. A mutation
   that survives, or does not compile, is replaced and recorded.
5. **Gates.** `make pre-add-check FILES="…"` over the step's Go files;
   `python3 -m tools.verify gates` with the step's files on macOS, and
   through `remote linux` and `remote windows`; `make apicheck` at 0
   incompatible changes; `python3 -m tools.verify scan --base HEAD`; and
   `make markdownlint check-records docs-check disclosure-check` when the
   step changes a document. Each fuzz target the step adds or changes
   runs 5 minutes.
6. **Conventions.** AGENTS.md's TUI conventions 1 to 6 and API conventions
   1 to 12 hold for every exported name: opaque options named `With…`,
   `Without…` or `On…`; a hook with its `…Func`; enums from
   `internal/enum`; `Err…` sentinels; each package's stability line; each
   default a row with its control; each string a user is shown a message
   ID.
7. **Cores and adapters** (0020-MADR D1 to D3): a declaration whose
   signature, fields or methods name a Charm type is the adapter's, and
   only then; every effect has a core form the adapter wraps; the adapter
   holds conversions, not logic.

## Go practice for every step

* **Zero values.** A type whose zero value can be ready is (`Queue[T]`,
  `Scanner`, `InterpretOptions`); one with an invariant has its `NewX`.
  An options struct's zero value takes the defaults.
* **Concurrency is stated.** A type's documentation says either "safe for
  concurrent use" or "used on one goroutine". A driver starts each
  goroutine with `sync.WaitGroup.Go` and waits for it in `Close`; `Close`
  is idempotent and returns an `error`, as `io.Closer` does; no goroutine
  runs without work; a channel is exposed receive-only, and closed by its
  sender alone; a `time.Timer` is stopped and reset without draining, as
  Go's timers have allowed since 1.23.
* **Contexts.** A job that may be abandoned takes a `context.Context` as
  its first parameter, cancelled when it is replaced and on `Close`.
* **Time.** The pure layer takes `now time.Time` and never reads a clock;
  a driver reads the real clock, and stamps what it emits, so the host's
  `now` and the driver's agree.
* **Generics and iterators.** `Queue[T any]`, `Coalescer[T any]` and
  `Upgrader[K comparable, R any]`; a drain is an `iter.Seq[T]`; `slices`
  and `maps` over hand-written loops.
* **Errors and panics.** An error wraps its cause with `%w`; untrusted
  input never panics; a panic marks a programmer's error only, as
  `palette`'s fixed key sequences do.
* **No global mutable state, no `init`.** `theme.Theme.ID`'s counter is
  the one package-level variable, a `sync/atomic` `Uint64`.
* **Tests.** Table-driven; drivers inside `testing/synctest.Test`, where a
  goroutine left blocked fails the test; `-race`; fuzz targets for every
  parser and every "the same for any split" property; goldens through
  `tuitest`'s matrix, {colour, no colour} × {UTF-8, ASCII} × 60 and 100
  columns; benchmarks with `b.Loop`, and allocation budgets held by a test
  where the MADR sets one.
* **Documentation.** Every exported name has a comment that begins with
  its name; each package has an `Example…`; a guide's Go block is a
  `// guide:` region of one, which the examples gate holds.

## Implementation Steps

### Step 1: records, the planned defaults and the reserved names

* **Records:** 0009-MADR A7 `accepted`, with Q13's answer (done,
  2026-11-01); this PLAN `in-progress`; `docs/README.md`'s rows.
* **`docs/guides/defaults.md`:** a section for each of `frame`,
  `frame/frametea`, `inputfilter`, `inputfilter/inputfiltertea` and
  `stream/streamtea`, and rows added to `stream`'s, headed `Behaviour |
  Default | Control | Status`, each `planned`, its control named as the
  step will build it:
  * `frame`: 60 fps (`frame.WithFPS`); each gear's threshold (its
    `AdaptiveOption`); a scheduler's timer goroutine, from `NewScheduler`
    to `Close` (not building one, `Scheduler.Close`); the `Coalescer`'s
    100 ms wait (`NewCoalescer`'s `maxWait`); `Demand`'s 83 ms `Slow` tick
    ~~(`frame.WithSlowTick`)~~ (`frame.Ticks`, D1); ~~`Writer`'s~~ `Output`'s (Q14) 5 s stall (`frame.WithStallAfter`);
    `Upgrader`'s goroutine a job (`Upgrader.Close`);
  * `inputfilter`: wheel coalescing and the motion throttle on, at 16 ms
    (`inputfiltertea.WithoutWheelCoalescing`,
    `inputfiltertea.WithoutMotionThrottle`, `inputfiltertea.WithInterval`);
    paste bursts off (`inputfiltertea.WithPasteBurst`), with codex's 3
    characters, 8 ms gap and 8 ms idle, 60 ms on Windows
    (`inputfilter.PasteBurstConfig`); the flush driver's timer
    (`inputfilter.Flusher.Close`); typeahead off; one event a notch
    (`inputfiltertea.WithWheelProfile`);
  * `stream`: `RelaxAfter` 2 KiB (`stream.WithRelaxAfter`); inline
    holdback and the partial line on (`stream.WithoutInlineHoldback`,
    `stream.WithoutPartialLine`); `Interpret`'s 50,000 rows and 8,192
    columns (`stream.InterpretOptions`); the highlighter's limits and
    `Bounded`'s (their options, named in Step A1.4);
  * `stream/streamtea`: following the tail (`streamtea.WithoutFollow`);
    the pane's keys (`streamtea.DefaultRules`); its pacer, `Adaptive`
    (`streamtea.WithPacer`).
* **`docs/glossary.md`,** "Names the planned records must not take": the
  names of A6 and A7 for `frame`, `frame/frametea`, `inputfilter`,
  `inputfilter/inputfiltertea`, `stream` and `stream/streamtea`, and
  `theme`'s `ID` method.
* **Checks:** `python3 -m tools.verify names` over every type this PLAN
  plans reports no clash; `TestDefaultsInventory` passes with the
  `planned` rows; the documents' checks.
* **A records commit,** with no source, under AGENTS.md's bootstrap
  exception.

### Step 2: `safetext`

*Done by 0020-PLAN Step 4, released in `v1.13.0` (MADR A6 item 1).*
`safetext`'s own tests are its checks.

### Step A1.1: `Interpret`, in `stream` (Q13)

* **Facts re-read:** `stream`'s imports; `x/ansi`'s `Method.StringWidth`
  and its parser's `DecodeSequence`; `safetext`'s handling of invalid
  UTF-8, which `Interpret` matches.
* **The API:**

  ```go
  // InterpretOptions bounds Interpret. The zero value takes the defaults.
  type InterpretOptions struct {
  	Method  ansi.Method // how a cell is counted; the zero value is ansi.WcWidth
  	MaxRows int         // 0 or less: DefaultInterpretRows
  	MaxCols int         // 0 or less: DefaultInterpretCols
  }

  // InterpretRow is one row of interpreted output.
  type InterpretRow struct {
  	Styled string // the row with its SGR, reset at its end
  	Plain  string // the row's text alone, for copy and search
  }

  const (
  	DefaultInterpretRows = 50_000
  	DefaultInterpretCols = 8_192
  )

  // Interpret replays captured terminal output into rows (0009-MADR A1).
  func Interpret(raw []byte, o InterpretOptions) []InterpretRow
  ```

* **Behaviour,** as A1 sets it: a grid of cells, each advanced by its
  width under `Method`, ~~a wide cluster taking two and a half-overwritten
  one blanked~~ *(corrected 2026-11-01, D3)* a cluster taking every cell
  its width measures, and one written over in any cell blanked whole; SGR
  in its `;` and `:` forms kept, and written once per
  change of style; CR to column 0, overwriting; BS one cell back; HT to the
  next multiple of 8; `CSI K` and `J` with 0, 1 and 2, and `A`, `B`, `C`,
  `D` and `G`, each with a default of 1; every other sequence, OSC among
  them, and every other control dropped; invalid UTF-8 one U+FFFD per
  maximal subpart, as `safetext` writes it. Past `MaxCols`, a row is cut;
  past `MaxRows`, the oldest rows go, so the output's tail is kept. Each
  call starts afresh, so its result is a function of its input.
* **Tests:** a table with one row per feature above; a progress bar
  rewritten by CR ends as its last state; a wide rune advances two cells;
  each cap; `Interpret` twice over the same bytes is equal.
  `FuzzInterpret`: no row wider than `MaxCols` cells under `Method`;
  `Styled` holds no sequence but SGR; `Plain` is `ansi.Strip(Styled)`;
  `Plain` passes `safetext.IsClean` under `safetext.Line()`.
* **Benchmark:** 1 MB of `go test -v` output with colour and a CR progress
  bar, its allocations reported.
* **Mutations:** one column a rune (the wide-rune row); CR not returning
  to column 0 (the progress bar); HT to a multiple of 4; OSC kept
  (`FuzzInterpret`); `MaxRows` ignored; the `:` form of SGR not read.
* **Docs:** `stream`'s package documentation; `docs/architecture.md`'s
  `stream` row.

### Step A8.1: `stream.Writer`, a plain streaming writer (0009-MADR A8)

*Added 2026-11-01 for 0009-MADR A8, accepted with Q14 answered:* it runs
after Step A1.1 and before Step 3, once the owner approves this step. The
type is `stream.Writer` (Q14).

* **Facts re-read:** the `Printer`'s API (`NewPrinter`, `Write`, `Flush`)
  and options; `TestAnySplit` and `FuzzPrinter`; `io.Writer`'s contract
  and `testing/iotest`'s readers and writers.
* **The API,** as A8 item 2 gives it, with `ErrClosed`, and the option
  `WithCRLF()`, which the `Printer` honours too.
* **Behaviour:** each `Write` and `WriteString` passes its input to the
  `Printer` and writes what is now stable in one `dst.Write`; `Close`
  writes the `Printer`'s `Flush`, leaves `dst` open, and makes a later
  `Write` return `ErrClosed`; a second `Close` returns nil; a `dst` error
  is sticky.
* **Tests:**
  * `io.Copy` of `testdata/reply.txt` into a `Writer` over a
    `bytes.Buffer`, through `iotest.OneByteReader`, `iotest.HalfReader`
    and a plain reader, gives the `Printer`'s `Write` and `Flush` output
    byte for byte, at widths 0, 60 and 100, with and without
    `DefaultMarkup`;
  * ~~with no option, the output is the input cleaned by `safetext.Text`;~~
    *(corrected 2026-11-01, D4)* with no option, the output is a
    `Printer`'s with no option, a table laid out; with
    `WithoutTableLayout`, the input cleaned by `safetext.Text`, byte for
    byte;
  * `WriteString` equals `Write`;
  * `WithCRLF`: every line ends with CR LF and no LF stands alone, from the
    `Writer` and from the `Printer`;
  * a destination that fails after some bytes: that `Write` returns
    `len(p)` and the error, and every later `Write` and `Close` the same
    error;
  * `Close` leaves a destination that counts its `Close` calls at 0; a
    `Write` after `Close` returns `ErrClosed`; a second `Close` nil;
  * `FuzzWriter`: any split of any input, through `Write`, equals the
    `Printer`'s output for the whole;
  * `ExampleWriter`: `io.Copy` of a Markdown reply from a reader.
* **Benchmark:** `io.Copy` of `testdata/reply.txt` through the `Writer`,
  its allocations beside `BenchmarkReplyRender`'s.
* **Mutations:** `Write` holding stable lines back (the copy test);
  `Close` not flushing (the copy test's tail); the error not sticky;
  `Close` closing `dst`; `WithCRLF` ignored; a `Write` after `Close`
  accepted.
* **Defaults:** rows built in `stream`'s section: each line's end, LF
  (`stream.WithCRLF`); the destination at `Close`, left open (decided,
  0009-MADR A8).
* **Docs:** `stream`'s package documentation; the line-mode guide's "Stream
  a reply", a writer for plain output; `docs/architecture.md`'s `stream`
  row; `docs/glossary.md`, Q14's names.
* **Gates,** as rule 5.

### Step 3: `frame`, its scheduler, and `frame/frametea`

* **Facts re-read:** Bubble Tea's frame-rate bounds (`renderer.go:13-14`,
  60 and 120); `tea.Cmd`'s shape; `testing/synctest`'s API.
* **The pure layer** (`frame`), used on one goroutine, time passed in:

  ```go
  type DueMsg struct {
  	At  time.Time // when the frame fell due
  	Seq uint64    // one more for each frame delivered
  }

  type Queue[T any] struct{ /* a ring of (T, time.Time) */ } // zero value ready
  func (q *Queue[T]) Push(v T, at time.Time)
  func (q *Queue[T]) Len() int
  func (q *Queue[T]) Snapshot(now time.Time) Snapshot
  func (q *Queue[T]) Drain(p Pacer, now time.Time) iter.Seq[T]

  type Snapshot struct {
  	Depth  int
  	Oldest time.Duration
  }
  type Pacer interface{ Pace(s Snapshot, now time.Time) Release }
  type PacerFunc func(s Snapshot, now time.Time) Release
  type Release struct {
  	Take int
  	Gear Gear
  }
  type Gear uint8 // Smooth, CatchUp: tokens "smooth", "catch-up"

  func Adaptive(o ...AdaptiveOption) Pacer // codex's two gears
  func Immediate() Pacer                   // the whole queue each frame
  ```

  `Adaptive`'s options are `WithCatchUpDepth` (8), `WithCatchUpAge`
  (120 ms), `WithSmoothDepth` (2), `WithSmoothAge` (40 ms), `WithHold`
  (250 ms), `WithSevereDepth` (64) and `WithSevereAge` (300 ms), each
  keeping its default at zero or less. The frame-rate arithmetic, when a
  frame wanted at a time may be drawn after the last, is an unexported
  function the scheduler calls, exported only if a step needs it.
* **The driver** (`frame`), safe for concurrent use:

  ```go
  func NewScheduler(o ...Option) *Scheduler // WithFPS(n): 60; clamped to 1..120
  func (s *Scheduler) Request()                  // as soon as the cap allows
  func (s *Scheduler) RequestIn(d time.Duration) // the earliest deadline wins
  func (s *Scheduler) Frames() <-chan DueMsg   // capacity one: requests merge
  func (s *Scheduler) Close() error              // closes Frames; idempotent
  ```

  One `time.Timer` under a mutex; a goroutine started by `WaitGroup.Go`
  sends a due frame, stamped with the time it fell due, and stops; `Close`
  stops the timer, waits, and closes `Frames`.
* **The adapter** (`frame/frametea`):

  ```go
  func Wait(s *frame.Scheduler) tea.Cmd                           // one DueMsg; nil once closed
  func Forward(s *frame.Scheduler, send func(tea.Msg)) (stop func()) // push delivery, §3's WithSend
  ```

  A program re-arms `Wait` on each `DueMsg`:
  `return m, tea.Batch(m.pane.Update(msg), frametea.Wait(m.frames))`.
  *Added 2026-11-01 (D2):* `Forward` starts one goroutine, which reads
  `Frames` and calls `send`, started by `WaitGroup.Go` and ended by its
  `stop` or by the scheduler's `Close`; `stop` waits for it, and is
  idempotent.
* **Tests,** inside `synctest.Test`:
  * 1,000 `Request`s from 10 goroutines within one frame deliver one
    `DueMsg`;
  * frames come no closer than 1/fps, at 60 and 120 fps; `WithFPS(500)`
    is 120 and `WithFPS(0)` is 60;
  * `RequestIn(50ms)` alone delivers at 50 ms, and with a `Request` after
    it at once, the cap allowing;
  * `Seq` rises by one a frame, and `At` is the fake clock's due time;
  * `Close` closes `Frames`, a second `Close` returns nil, and no
    goroutine is left (the bubble ends);
  * `frametea.Wait` returns the frame, and nil after `Close`;
    `Forward`'s `stop` ends delivery; *(D2)* `Forward`'s goroutine is gone
    after `stop`, and after `Close` with no `stop`, the bubble ending.
* **The pacer's table,** codex's transitions: CatchUp entered at depth 8
  and at age 120 ms; left only when depth ≤ 2 and age ≤ 40 ms have held
  250 ms; no re-entry within 250 ms but at depth 64 or age 300 ms; an
  empty queue back to Smooth. `PacerFunc` adapts a function.
* **The queue:** `Drain` takes from the head, in order; `Snapshot` gives
  depth and the oldest's age; once grown, `Push` and `Drain` allocate
  nothing an item (`testing.AllocsPerRun`).
* **Mutations:** `Frames` unbuffered, or of capacity 2 (the merge test);
  the cap skipped; the exit hold ignored; the severe bypass ignored;
  `Drain` from the tail; `Close` not waiting for the goroutine (the
  bubble); `Seq` not raised.
* **Conformance:** `frame` joins `cores`; `TestCoresAreCharmFree` passes,
  and fails on a scratch copy that imports Bubble Tea into `frame`.
* **Defaults:** the `frame` rows of Step 1 that this step builds become
  `built`.

### Step A1.2: `Coalescer`, `Demand`, `Output` and `Upgrader`

*Renamed 2026-11-01 (0009-MADR A8, Q14):* A1's `frame.Writer` is
`frame.Output`, with `NewOutput`, `OutputOption` and `OutputEvent`, so
that `stream.Writer` (Step A8.1) is the one `Writer`. This step's text
uses the new names.

* **Facts re-read:** 0003-REPORT §8.17's numbers (83 ms, 5 s, 100 ms);
  Bubble Tea's renderer writing changes only (`cursed_renderer.go:643`;
  ultraviolet `terminal_renderer.go:111-117`), on which `Output` rests
  (A7 item 10).
* **The API** (`frame` unless named):

  ```go
  // Coalescer batches mergeable items a frame at a time; any other item is
  // a barrier, which flushes the batch first. Used on one goroutine.
  func NewCoalescer[T any](apply func([]T), mergeable func(T) bool, maxWait time.Duration) *Coalescer[T]
  func (c *Coalescer[T]) Push(v T, now time.Time) (due time.Time, ok bool) // a deadline, when one is newly needed
  func (c *Coalescer[T]) Flush()

  type Demand uint8 // None, Slow, Fast: tokens "none", "slow", "fast"
  func Combine(d ...Demand) Demand // the greatest
  // Ticks is how often each demand ticks; a zero field takes its default
  // (D1).
  type Ticks struct {
  	Slow time.Duration // 0: 83 ms
  	Fast time.Duration // 0: one frame at 60 fps
  }
  func (t Ticks) Interval(d Demand) time.Duration // 0 for None

  // Output wraps the output a program gives tea.WithOutput. Safe for
  // concurrent use.
  func NewOutput(w io.Writer, o ...OutputOption) *Output // WithStallAfter(5 s)
  func (o *Output) Write(p []byte) (int, error)
  func (o *Output) Events() <-chan OutputEvent // Stalled, Resumed
  func (o *Output) Close() error

  // Upgrader runs costly renders off the event loop. Safe for concurrent use.
  func NewUpgrader[K comparable, R any]() *Upgrader[K, R]
  func (u *Upgrader[K, R]) Submit(key K, gen uint64, fn func(context.Context) R)
  func (u *Upgrader[K, R]) Results() <-chan UpgradeMsg[K, R]
  func (u *Upgrader[K, R]) Close() error
  type UpgradeMsg[K comparable, R any] struct {
  	Key    K
  	Gen    uint64
  	Result R
  }
  ```

  *Corrected 2026-11-01 (D1):* ~~`Demand.Interval(fps int)`~~ is
  `Ticks.Interval`, above. `frame/frametea` adds ~~`TickCmd(d frame.Demand,
  fps int) tea.Cmd`~~ `TickCmd(d frame.Demand, t frame.Ticks) tea.Cmd`, nil
  for `None`; `WaitUpgrade(u) tea.Cmd`; and `CoalesceCmd(due time.Time)
  tea.Cmd`, the tick for a `Coalescer`'s deadline.
  *Added 2026-11-02 (D7):* `TickCmd`'s message is
  `frame.TickMsg{At time.Time; Demand Demand}`, and `CoalesceCmd`'s
  `frame.FlushMsg{At time.Time}`, on which a program calls `Flush`; both
  are the core's, beside `DueMsg`.
* **`Output`** writes on its own goroutine; while a write is in flight it
  appends later writes to one pending buffer, written whole after, and
  drops nothing (A7 item 10). `Write` returns `len(p)` and the error of an
  earlier write, if any. A write in flight longer than the stall time
  sends `Stalled` once, and `Resumed` when it completes.
  *Amended 2026-11-02 (D9):* `Output` has `Fd() uintptr` and
  `Read(p []byte) (int, error)`, forwarded to what it wraps when that has
  them, so that it is a `term.File` and Bubble Tea still finds the
  terminal; and `Write` waits for its own bytes to be written up to
  `WithWriteWait(d)` (100 ms), returning sooner when they are, so that on
  a terminal that keeps up Bubble Tea's last write lands before it
  restores the terminal. Only a write slower than the wait lets the
  program go on, its later draws merging.
* **`Upgrader`:** a newer `Submit` for a key cancels the older job's
  context; a result whose generation is older than the newest submitted
  for its key is dropped; `Close` cancels every job and waits.
  *Read 2026-11-02 (D8):* "the newest submitted" is the key's job while it
  runs. A `Submit` whose generation is lower than the running job's is
  ignored; an equal or higher one cancels and replaces it; a result is
  delivered only while its job is still the key's; and the key is
  forgotten when its job ends.
* **Tests,** inside `synctest.Test`: a barrier between deltas is applied
  after the earlier and before the later, for every interleaving of a
  generated sequence; the maximum wait applies a batch when no frame
  comes; `Combine` is the greatest; `Interval(None)` is 0 and `TickCmd`
  nil; an idle program whose demand is `None` arms no timer; `Output` over
  a writer that blocks makes many draws one write, sends one `Stalled`
  after 5 s and one `Resumed`, and the bytes written equal the bytes
  given, in order; `Upgrader` replaces, cancels, drops stale results, and
  leaves no goroutine after `Close`.
* **Mutations:** the barrier not flushing first; `TickCmd(None)` a
  command; `Output` dropping the pending buffer (the byte count);
  `Output` writing a second time while one is in flight; `Upgrader`
  keeping a stale result; `Upgrader` not cancelling the replaced job.

### Step 4: `inputfilter`, its driver, `inputfilter/inputfiltertea`, and the extension point

* **Facts re-read:** `tea.MouseMsg` (`mouse.go:46`) and the paste
  messages (`paste.go`); `local` at `workspace/render.go:538-556`;
  `keymaptea.FromTea`, which converts a `tea.Key` to a
  `termkey.KeyEvent`.
* **The pure layer** (`inputfilter`), used on one goroutine, time passed
  in:
  * a wheel sum: notches added by direction, a change of direction
    resetting it, the first passing at once and later ones at most one an
    interval, carrying the sum, and a residual a caller can take;
  * a motion throttle: one sample an interval;
  * `PasteBurst`, over `termkey.Input`, with `PasteBurstConfig` (codex's
    3 characters, 8 ms gap, 8 ms idle, 60 ms on Windows, the platform
    from the config's `GOOS`), `Key(in termkey.Input, now time.Time)
    BurstAction` and `Tick(now time.Time) (termkey.Input, bool)`, which
    gives a burst as one `termkey.InputPaste`; Enter inside a burst a
    newline; a bracketed paste turning detection off for good.
* **The driver** (`inputfilter.Flusher`), safe for concurrent use: the
  timed flushes, a wheel residual and a burst's idle end, on a
  receive-only channel, with `Close`.
* *Added 2026-11-02 (D10, D10a):* the core's API, which the bullets above
  left open, is:

  ```go
  // inputfilter, pure: time in, no clock; each used on one goroutine
  type Scroll struct{ X, Y, DX, DY int; Mod termkey.Mod }
  type Wheel struct{ /* zero value ready: 16 ms */ }
  func (w *Wheel) Add(s Scroll, now time.Time) (Scroll, bool)
  func (w *Wheel) Residual() (Scroll, bool)
  func (w *Wheel) Due() (time.Time, bool)
  type Throttle struct{ /* zero value ready: 16 ms */ }
  func (t *Throttle) Allow(now time.Time) bool
  type PasteBurstConfig struct{ MinChars int; Gap, Idle time.Duration; GOOS string }
  func NewPasteBurst(c PasteBurstConfig) *PasteBurst
  func (p *PasteBurst) Key(in termkey.Input, now time.Time) BurstAction // Pass, Hold, Release
  func (p *PasteBurst) Tick(now time.Time) (termkey.Input, bool)
  func (p *PasteBurst) Due() (time.Time, bool)

  // inputfilter, the driver: safe for concurrent use
  func NewFlusher(o ...Option) *Flusher // WithInterval, WithoutWheelCoalescing,
  	// WithoutMotionThrottle, WithPasteBurst
  func (f *Flusher) Scroll(s Scroll, now time.Time) (Scroll, bool)
  func (f *Flusher) Motion(now time.Time) bool
  func (f *Flusher) Input(in termkey.Input, now time.Time) []termkey.Input
  func (f *Flusher) Flushes() <-chan time.Time // a flush fell due (D10a)
  func (f *Flusher) Flush(now time.Time) Flush  // what is due, taken by the caller
  func (f *Flusher) Close() error
  type Flush struct{ Inputs []termkey.Input; Scroll Scroll; HasScroll bool }

  // inputfilter/inputfiltertea
  func NewFilter(o ...inputfilter.Option) *Filter
  ```

  The options are the core's, and the adapter's `NewFilter` takes them.
  `Flushes` says only that a flush fell due, and the caller takes it with
  `Flush` on its own goroutine, so that the stages give up nothing off
  the event loop. *(D11)* A burst starts at `MinChars`, 3, the
  characters before it held, and an Enter after a burst's flush is a key.
* **The adapter** (`inputfilter/inputfiltertea`):

  ```go
  func NewFilter(o ...Option) *Filter // WithInterval(16 ms), WithoutWheelCoalescing,
  	// WithoutMotionThrottle, WithPasteBurst(inputfilter.PasteBurstConfig), WithWheelProfile(n)
  	// D10: ...inputfilter.Option; D12: WithWheelProfile is Step A1.3's
  func (f *Filter) Filter(m tea.Model, msg tea.Msg) tea.Msg // for tea.WithFilter
  func (f *Filter) Attach(send func(tea.Msg)) (stop func()) // the driver's flushes, and the FIFO's drain

  type CoalescedWheelMsg struct {
  	Pointer tea.Mouse
  	DX, DY  int
  }
  func (m CoalescedWheelMsg) Mouse() tea.Mouse
  func (m CoalescedWheelMsg) String() string
  func (m CoalescedWheelMsg) WithMouse(t tea.Mouse) tea.Msg
  func (m CoalescedWheelMsg) WheelDelta() (dx, dy int)
  ```

  A burst reaches the model as `tea.PasteMsg`. Order is kept by §6's FIFO,
  drained through `send` from a goroutine the filter starts, so the event
  loop never blocks on itself. `WheelDelta` lets a pane read any coalesced
  wheel message without importing this package.
* **`workspace`'s extension point:** `local` gains one case, a
  `tea.MouseMsg` with `WithMouse(tea.Mouse) tea.Msg`, translated by calling
  it; its comment and `docs/guides/building-workspaces.md` say so.
* **Tests:** the pure layer with given times: 100 notches within 16 ms
  give the full sum once the residual is taken, the first at once;
  up then down resets; motion one an interval; clicks, releases and keys
  untouched; `PasteBurst` passes codex's cases, a 3-character burst,
  Enter inside, a burst ended by an arrow, Windows' longer idle, and
  typing 50 ms apart passing through. The driver, inside `synctest.Test`:
  the residual flushed after the last notch; no goroutine after `Close`.
  The filter: a burst arrives as one `tea.PasteMsg`; after a
  `tea.PasteStartMsg` detection is off; interleaved keys, wheel events and
  drains keep their order; `send` is never called on the calling
  goroutine. The workspace: a `CoalescedWheelMsg` over a pane arrives in
  its local cells; a program's own type with `WithMouse` does too; one
  without it is dropped.
* **Mutations:** the direction reset removed; every motion sample passed;
  the FIFO bypassed by new input; `local` ignoring `WithMouse`; a
  `PasteStartMsg` leaving detection on; `Close` not waiting.
* **Conformance:** `inputfilter` joins `cores`, and imports `termkey`, a
  core.

### Step A1.3: the input stages, measured first

* **Measured before any code,** on a scratch module against the tree:
  Bubble Tea's reading of an SGR mouse report and a focus report split at
  each byte with gaps of 10, 60 and 200 ms, and of an X10 report past
  column 95, fed through a pipe; and of a bracketed paste with CRLF and
  lone CR. The record quotes what the model received. A stage the
  measurement shows Bubble Tea already does, as ultraviolet's 50 ms join
  may (A6 item 2), is not built, and the owner is told, as a deviation.
* **The stages,** each over `termkey.Input` in `inputfilter`, with its
  conversion in `inputfiltertea`, as A1 lists them: fragment reassembly
  past the measured window; X10 repair; late probe replies swallowed by
  matcher functions the caller gives, so `inputfilter` imports no
  `termcap`; a bare Esc flushing what any stage holds; `Capture` and
  `Quarantine` typeahead, neither on by default; paste normalisation,
  CRLF and lone CR to LF, and an empty bracketed paste to
  `ClipboardImageRequestMsg`, a plain struct in the core;
  `PastedPath(s, goos string) (string, bool)`; grok's PowerShell burst
  cases as `PasteBurstConfig` fields; and `WithWheelProfile`'s events a
  notch, 1 by default. *(D10, D12)* `WithWheelProfile` is an
  `inputfilter.Option`, which the adapter's `NewFilter` takes.
* **Tests:** each report split at every byte and in every two-way split
  reaches the model as one message, or none for a swallowed reply, and no
  stray key; a truncated report at the input's end is held, then given as
  keys by the next bare Esc, never as a mouse event; `Capture` keeps text,
  Backspace, Shift and Alt+Enter and pastes, and cuts at the first Esc;
  `Quarantine` drops all input to its deadline; the paste cases;
  `PastedPath`'s quoted, `file://`, drive and UNC forms, and prose
  refused; a profile of 3 makes 3 events one notch.
* **Mutations:** reassembly skipped; X10 repair skipped; late replies
  passed as keys; Esc not flushing; CR left in a paste.
* *Written 2026-11-02 (D13, D14), for the owner's review before any
  code:* the step's API. Each stage is pure, time in, used on one
  goroutine, and has `PasteBurst`'s shape: `Key(in, now) Action`, `Tick(now)
  (termkey.Input, bool)` and `Due() (time.Time, bool)`. The `Flusher` runs
  them in this order, each one's output the next one's input:
  `Reassembler`, the typeahead, paste normalisation, `PasteBurst`.

  ```go
  // inputfilter: one action for every stage. BurstAction, Step 4's, is
  // renamed: inputfilter is in no release, so no deprecation is owed.
  type Action uint8 // Pass, Hold, Release: tokens "pass", "hold", "release"

  // Report is a terminal report the stages decode.
  type Report struct {
  	Kind   ReportKind
  	Button MouseButton
  	X, Y   int         // cells, from 0
  	Mod    termkey.Mod
  }
  type ReportKind uint8  // MousePress, MouseRelease, MouseMotion, MouseWheel,
  	// FocusIn, FocusOut: tokens "press", "release", "motion", "wheel",
  	// "focus-in", "focus-out"
  type MouseButton uint8 // ButtonNone, ButtonLeft, ButtonMiddle, ButtonRight,
  	// WheelUp, WheelDown, WheelLeft, WheelRight, ButtonBackward,
  	// ButtonForward, Button10, Button11: ultraviolet's values, which
  	// tea.MouseButton has, so the adapter converts value for value
  func ParseReport(raw string) (Report, bool) // SGR mouse, X10 mouse, focus

  // Reassembler joins a report or a reply split past the reader's 50 ms:
  // from an unknown head (D14), an InputUnknown that begins a CSI, OSC, DCS
  // or APC sequence and does not end it, each key after it that is one
  // byte of the sequence is held, up to 1 KiB, until the sequence ends, a
  // key that is not its next byte comes, or the window passes. A sequence
  // joined is a reply the matcher knows, and dropped; or a Report, given
  // as one InputUnknown with its whole Raw; or neither, and given back as
  // it came. An X10 report ConPTY re-encoded, an InputUnknown "\x1b[M"
  // and three runes, one of U+0080 to U+00FF, and the keys it ran into, is
  // repaired, each rune back to its byte. Anything else ends what is held
  // first: a bare Esc gives it back at once.
  type ReassembleConfig struct {
  	Window  time.Duration // how long a head waits for the rest: 500 ms
  	Replies ReplyMatcher  // nil: no reply is dropped
  	NoX10   bool          // leave X10 reports as they came
  }
  type ReplyMatcher interface{ IsReply(raw string) bool }
  type ReplyMatcherFunc func(raw string) bool
  func NewReassembler(c ReassembleConfig) *Reassembler

  // Typeahead keeps grok's typing from ins: text, Backspace, Shift or
  // Alt+Enter, and pastes, cut at the first Esc.
  func Typeahead(ins []termkey.Input) []termkey.Input

  // NormalizePaste makes a paste's CRLF and lone CR an LF.
  func NormalizePaste(s string) string

  // ClipboardImageRequestMsg is an empty bracketed paste, which Windows
  // Terminal before 1.25 sends for a clipboard holding only an image.
  type ClipboardImageRequestMsg struct{}

  // PastedPath reads s as a dropped or pasted path: trimmed, its matching
  // quotes removed, a file:// URL decoded, and accepted only when it begins
  // with a path's anchor, "/", "~/", a drive ("C:\" or "C:/") or a UNC
  // "\\" (grok's starts_with_drop_anchor); backslash escapes are undone
  // except where goos is "windows", and except in a drive or UNC path;
  // an unquoted space not escaped refuses it, but in a drive or UNC path.
  func PastedPath(s, goos string) (string, bool)

  // PasteBurstConfig gains grok's PowerShell rules, each off at its zero:
  //	FirstGap     time.Duration // how soon a run's second key must follow: Gap
  //	LinesOnly    bool          // a burst with no character after an Enter is
  //	                           // given back as its keys, unless path-shaped
  //	PathMinChars int           // with GOOS "windows", a path-shaped burst of at
  //	                           // least this many is a paste under LinesOnly
  //	EnterCtrlJ   bool          // an Enter then a Ctrl+J is one "\n", a paste
  func PowerShellPasteBurst() PasteBurstConfig // grok's: FirstGap 2 ms, Gap 10 ms,
  	// MinChars 3, LinesOnly, PathMinChars 8, EnterCtrlJ

  // Wheel gains EventsPerNotch int: 1 at zero or less. Whole notches pass;
  // the events left over carry into the next in the same direction.

  // The Flusher's new options and methods.
  func WithoutReassembly() Option
  func WithReassemblyWindow(d time.Duration) Option // 500 ms; zero or less keeps it
  func WithReplies(m ReplyMatcher) Option
  func WithoutX10Repair() Option
  func WithoutPasteNormalization() Option
  func WithWheelProfile(eventsPerNotch int) Option // 1
  func (f *Flusher) Capture()                     // hold keys and pastes until Replay
  func (f *Flusher) Replay() []termkey.Input       // what Capture held, by Typeahead
  func (f *Flusher) Quarantine(until time.Time)   // drop all input until then

  // inputfilter/inputfiltertea: the conversions.
  //   - a uv.UnknownEvent, read through internal/termevent.Decode, becomes
  //     an InputUnknown; an InputUnknown the Reassembler gave back as it
  //     came becomes the message it was;
  //   - a joined Report becomes tea's MouseClickMsg, MouseReleaseMsg,
  //     MouseMotionMsg, MouseWheelMsg, FocusMsg or BlurMsg;
  //   - an X10 report tea decoded, a MouseMsg whose X+33 is a UTF-8 lead
  //     byte, 0xC2 or 0xC3, with the keys after it, is given to the
  //     Reassembler as the InputUnknown it was, re-encoded from the Mouse;
  //   - an empty tea.PasteMsg becomes inputfilter.ClipboardImageRequestMsg,
  //     unless WithoutClipboardImageRequest;
  //   - Filter gains Capture, Replay, which gives what was held through the
  //     drains, and Quarantine.
  func WithoutClipboardImageRequest() inputfilter.Option
  ```

  *Corrected 2026-11-02 (D15):* ~~`inputfiltertea.WithoutClipboardImageRequest`~~
  is `inputfilter.WithoutClipboardImageRequest() Option`, the core's, with
  `func (f *Flusher) ClipboardImageRequest(in termkey.Input) bool`, which
  reports whether an input the Flusher gave is the request: an empty
  bracketed paste, unless the option built it. The adapter turns such a
  paste into `ClipboardImageRequestMsg`; a line-mode host asks the same.

  *Limits stated, by D14 and the measurement:* a report split after its
  first one or two bytes reaches the model as `esc` or `alt+[` and keys,
  and is not rejoined, so that a bare Esc never waits; a focus report,
  three bytes long, has no unknown head to begin from, and is never
  rejoined; a probe reply Bubble Tea decodes whole after its deadline,
  such as a `tea.BackgroundColorMsg`, is a message, not keys, and passes.
  *Numbers the agent chose, with no source:* the 500 ms window, long
  against the measured 60 and 200 ms gaps and short against a person's
  typing; the 1 KiB cap is the probe's (`termcaptea/prober.go:182`).
* *(D13)* **Tests,** besides those above: every two-way split of an SGR
  report and of a DA1, OSC 11 and kitty-keyboard reply, past 50 ms,
  whose head is three bytes or more, reaches the model as one message or
  none; the X10 cases measured, columns 96, 120, 200 and 223, repaired,
  in termkey's form and in tea's; a head with nothing after it given back
  at the window's end; a bare Esc after a head giving it back at once;
  `Typeahead` on grok's cases (`event_loop.rs:56-124`); `Quarantine`
  dropping a burst of keys and passing the first key after its deadline;
  `NormalizePaste`; `PastedPath` on codex's cases
  (`clipboard_paste.rs:399-560`) with prose, two paths and a URL that is
  not a file refused; `PowerShellPasteBurst` on grok's cases
  (`event_loop.rs:3587-3715`): a single-line run given back, a multi-line
  one a paste, an Enter then Ctrl+J a CRLF, a Windows path of eight; a
  profile of 3 making 3 events one notch; `Action`'s tokens.
* *(D13)* **Mutations,** besides those above: the window ignored; a
  reply matched and still given; a Report's coordinates off by one; a
  rune of U+0080 to U+00FF not repaired; `Typeahead` not cutting at Esc;
  `Quarantine` passing input before its deadline; `PastedPath` accepting
  prose; `LinesOnly` ignored; `EnterCtrlJ` ignored; the wheel's leftover
  events dropped.

### Step 5: `Scanner`, `Doc`, `Renderer`, `Plain`, `Buffer` and `stream/streamtest`

Four parts, each passing its tests before the next.

1. **`Scanner`, moved out of the `Printer`** (A6 item 5; 0023-MADR A6
   item 2). The block state the `Printer` keeps in its own fields
   (`fence`, `html`, `table`, `tableNL`, `paragraph`, `lazy`, `defs`) moves
   into `Scanner`, whose zero value is a document's start:

   ```go
   type Scanner struct{ /* the open fence, HTML block, table, paragraph, containers */ }
   func (s *Scanner) Scan(line string) Block // one complete line
   type Block struct {
   	Kind BlockKind // tokens from internal/enum
   	Cut  bool      // a cut may fall before the next line, by §4's rule
   	// … the fields §4's decisions need
   }
   ```

   The `Printer` is rebuilt on it, and every one of its tests passes
   unchanged, its goldens byte for byte: `TestApply` to
   `TestParagraphContinues`, `FuzzPrinter`, `ExamplePrinter`, and
   `TestReplyBudget`'s allocation budget.
2. **`Doc`,** on `Scanner`, as §4 decides, with A4's and A6's names:

   ```go
   type Renderer interface{ Render(src string, width int) (string, error) }
   type RendererFunc func(src string, width int) (string, error)
   func Plain(m ansi.Method) Renderer // wraps with wrapProse, lays tables out with layoutTable

   func NewDoc(r Renderer, o ...Option) *Doc // stream.Option: WithRules, WithMethod,
   	// WithRelaxAfter(2 KiB), WithoutInlineHoldback, WithoutPartialLine
   func (d *Doc) Append(s string)
   func (d *Doc) Write(p []byte) (int, error)
   func (d *Doc) Finish()
   func (d *Doc) Reset()
   func (d *Doc) Render(width int) string
   func (d *Doc) Version() uint64
   func (d *Doc) Finished() bool
   func (d *Doc) Source() string
   ```

   `stream.Option` gains a second unexported method so that one option
   applies to a `Printer` and a `Doc` (A6 item 4, A7 item 6); an option
   that means nothing to one of them is documented as ignored by it.
   *Added 2026-11-03 (D16):* each segment's render input carries the
   reference and footnote definitions committed so far, after a blank
   line, so that a reference resolves wherever its definition is; a
   committed definition drops the cache, and the segments render once
   more. `Plain` prints no definition line, so that its frames and its
   one-shot render agree.
3. **`Buffer` and `Doc.Pull`,** in `stream` (A7 item 9):

   ```go
   func NewBuffer(s *frame.Scheduler) *Buffer           // safe for concurrent use
   func (b *Buffer) Write(p []byte) (int, error)          // whole lines timestamped; a line requests a frame
   func (b *Buffer) Close() error                        // the stream's end: Pull then calls Finish
   func (d *Doc) Pull(b *Buffer, p frame.Pacer, now time.Time) bool // true while more is queued
   ```

   *Amended 2026-11-03 (D17, in Step 6):* while lines stay queued, `Pull`
   also asks `b`'s Scheduler for the next frame.

4. **`stream/streamtest`** (A2): `CheckRenderer(t testing.TB, r
   stream.Renderer, o ...stream.Option)`, built on `stream`'s exported API
   alone, streaming generated documents in random chunks and reporting
   chunking invariance and `Finish` exactness through `t`.

* **Tests:** the scanner's table (fences of both characters, longer
  closers, indented openers, info strings, an unclosed fence; list items
  with continuations, loose lists, a new list after a blank; quotes; HTML
  blocks of types 1 to 7; indented code; tables; setext headings;
  reference and footnote definitions). The properties, over a block
  grammar and random chunkings, through `CheckRenderer` with `Plain`:
  chunking invariance; `Finish` exactness; the stated differences only,
  with a fake renderer and no relaxed cut; the cost, by a counting
  renderer over 200 KB of prose with paragraphs, prose with none, one
  fence and one list, every committed byte rendered once a width and no
  `Render` given more than `RelaxAfter` and a line of tail, a 200 KB table
  the stated exception; a width changed and back rendering no segment
  more than twice; no frame holding 0x1B that the renderer did not add.
  Holdback: `Hello **wor` shows `Hello `; a closed `**world**` shows; an
  open backtick run, `~~` and `[label](http…` are held; the partial line
  hidden while lines are queued. `Buffer`: 8 goroutines writing under
  `-race` lose no byte and keep each writer's order; a completed line
  requests a frame; `Close` makes `Pull` call `Finish`. `FuzzDoc` over
  text and split points: no panic, chunking invariance, `Finish`
  exactness.
* **Benchmark:** 200 KB in 20-byte chunks through `Plain`, its
  allocations, against a whole re-render per chunk.
* **Mutations:** a cut inside a fence; the list's next-line check
  skipped; a reference definition keeping the cache; `Finish` keeping the
  segmented render; the relaxed cut never taken; a relaxed cut inside a
  table; a list's relaxed cut losing its number; `Append` skipping the
  sanitizer; `CheckRenderer` with a renderer whose output depends on
  where the source was cut (it must fail); the `Printer` built on a
  `Scanner` that forgets the lazy continuation (`TestParagraphContinues`).

### Step 6: `theme.Theme.ID` and `DocPane`

* **Facts re-read:** `theme.New`, `theme.Theme`'s fields; the workspace's
  `Theme()` and `WidthMethod()`; `palette`'s keys (`palette/keys.go`) as
  the pattern of A7 item 2.
* **`theme.Theme.ID`** (A6 Q9, A7 item 4): `func (t Theme) ID() uint64`,
  stamped by `New` from a `sync/atomic` counter, unique a process; a
  literal `Theme` has 0. `Theme`'s and `ID`'s documentation say that a
  copy keeps its ID, and that a changed theme is built with `New`.
* **`DocPane`** (`stream/streamtea`), used on the event loop:

  ```go
  func NewDocPane(w *workspace.Workspace, d *stream.Doc, o ...Option) *DocPane
  	// WithTitle(s), WithoutFollow(), WithBuffer(*stream.Buffer), WithPacer(frame.Pacer)
  func (p *DocPane) Update(msg tea.Msg) (workspace.Pane, tea.Cmd)
  func (p *DocPane) View(width, height int) string
  func (p *DocPane) Changed() bool
  func (p *DocPane) Title() string
  func (p *DocPane) KeyContexts() []command.Scope // DocPaneScope, "kind:docpane"
  func (p *DocPane) ConsumesCommand(id command.ID) bool
  func DefaultRules() []keymap.Default // up, down, pgup, pgdown, home, end
  const (
  	ScrollUpID   command.ID = "stream.scroll.up"
  	ScrollDownID command.ID = "stream.scroll.down"
  	PageUpID     command.ID = "stream.page.up"
  	PageDownID   command.ID = "stream.page.down"
  	TopID        command.ID = "stream.top"
  	BottomID     command.ID = "stream.bottom"
  )
  ```

  It follows the tail until scrolled up, and again at the bottom; it pulls
  from its `Buffer` on each `frame.DueMsg` with its pacer, `Adaptive` by
  default; it scrolls by a `tea.MouseWheelMsg`, and by any mouse message
  with `WheelDelta() (dx, dy int)`; it reads `w.Theme()` and
  `w.WidthMethod()` at each draw, and keeps its lines by width, method and
  theme ID, an ID of 0 never reused; its scroll marks come from the
  theme's glyphs; `Changed` is true only after a new `Version`, size,
  scroll or key.
  *Added 2026-11-03 (D17):* `Doc.Pull` asks its `Buffer`'s Scheduler for
  the next frame while lines stay queued, so that the pane, which has no
  scheduler, drains its queue a frame at a time. Step 5's `Pull` gains
  it here, with a test, and the mutation "`Pull` not asking for the next
  frame".
* **Tests:** `ID`: two `New`s differ, a copy keeps its ID, a literal is 0,
  concurrent `New`s under `-race` are unique. `DocPane`, in a real
  workspace with `workspace.DefaultRules()` and `DefaultRules()` in its
  keymap: following, stopping and resuming; `Changed` false after an
  unrelated message; a frame pulls; both wheel messages scroll; each
  command moves as named, and a rebound key does too; a theme change
  redraws, and a literal theme is never served from the cache; `View`
  at 10,000 lines within 2× of 100 lines (a benchmark). Goldens across the
  matrix: mid-stream with a partial line, inside a table, finished, and
  scrolled up with its marks.
* **Mutations:** `Changed` always true; following ignoring the user's
  scroll; the cache keyed without the theme ID; an ID of 0 reused;
  `ConsumesCommand` false for a scroll command; the marks not from the
  glyphs.
* **Conformance:** `stream/streamtea` is not a core; `theme`'s API
  diff is one addition.

### Step A1.4: `RenderView`, the dialect, `Highlighter`, `FitTable` and `Bounded`

* **The API,** A1's, with A4's names: `Doc.View(width int) RenderView`,
  its lines with their source offsets, links with IDs and cell ranges,
  code blocks with their info and ranges, and table copy data;
  `Highlighter` and `HighlighterFunc`; `FitTable` from `layoutTable`, with
  A1's fallbacks to key/value records, stacked records and the source;
  `Bounded`, with its byte, line and line-length limits as options.
  `Bounded`'s omitted-lines marker is the message `stream.MsgOmitted`,
  with `Messages()` and `WithCatalog`, `stream`'s first text a user is
  shown; `DocPane` passes its catalogue on.
* **Tests:** A1's: chunking invariance of `RenderView`; a wrapped link
  keeping one ID across a tail re-render; `~**10%**` not struck, `~~x~~`
  struck, an open `\(` held until `Finish` with math on; a counting
  `Highlighter` highlighting each line of a growing fence once, both caches
  dropped by a theme ID or width change, each limit showing code plain;
  `FitTable` keeping numbers and URLs whole, padding a wide-glyph row
  exactly, falling back as the width shrinks, with goldens across the
  matrix and the ASCII border set; `Bounded` keeping head and tail,
  closing a cut escape, and giving one omitted count at two widths.
* **Mutations:** `RenderView` not cut back on a tail re-render; a single
  `~` striking; the fence re-highlighted whole each line; `FitTable`
  clipping by byte; `Bounded` counting rows instead of lines; the marker
  not from the catalogue.
* *Written 2026-11-03 (D19), for the owner's review before any code:*
  the step's API. Every type is `stream`'s, a core, which gains the
  imports `glyph` (A1) and `msgcat`; `stream.Option` gains a third
  unexported method, for `Bounded`, compatibly, as A7 item 6 found for
  the second, and an option that means nothing to a `Printer`, a `Doc` or
  a `Bounded` is documented as ignored by it.

  ```go
  // RenderView is a frame with its side-tables, for copying a code block,
  // hit-testing a link and selecting text while a reply streams (A1 §4).
  type RenderView struct {
  	Lines   []string    // the frame's rows: Render's, split at its line breaks
  	Offsets []int       // a row's source offset: the start of the source line it was
  	                    // drawn from; of its segment, when the Renderer tells none; the
  	                    // blank row between two segments, the next segment's
  	Links   []ViewLink  // in row order
  	Code    []ViewCode  // in source order
  	Tables  []ViewTable // in source order
  }

  // ViewLink is a link's part on one row: a link wrapped over rows has a
  // part a row, all with one ID.
  type ViewLink struct {
  	ID       int    // the source offset of the link's "[": the same in every frame,
  	                // at every width and for any chunking
  	URL      string // its destination, or its reference's, as the source has it
  	Row      int
  	From, To int    // its cells in the row, To excluded, under the Doc's method
  }

  // ViewCode is a code block: its info string and its body's source, for
  // "copy code block", and the rows it was drawn on, its fences included.
  type ViewCode struct {
  	Info       string // the fence's info string; "" for indented code
  	Start, End int    // its body's source bytes, the lines between its fences
  	Row, Rows  int
  }

  // ViewTable is a table's cells as the source has them, its header first,
  // for copying, and the rows it was drawn on.
  type ViewTable struct {
  	Cells     [][]string
  	Row, Rows int
  }

  // ViewRenderer is a Renderer that tells its rows' lines and its
  // side-tables, its offsets in src. Plain is one.
  type ViewRenderer interface {
  	Renderer
  	RenderView(src string, width int) (RenderView, error)
  }

  // View is Render's frame with its side-tables. A segment's view is kept
  // with its render; the tail's is taken anew each frame, so nothing of an
  // earlier tail is carried; a code block a relaxed cut split is one
  // ViewCode; after Finish, it is the Renderer's view of the whole source.
  // With a Renderer that is not a ViewRenderer, a row's offset is its
  // segment's, and there is no link, code block or table.
  func (d *Doc) View(width int) RenderView

  // Highlighter highlights code: lines, continuing from state, and the
  // state after them, which a later call continues from, or nil when it
  // cannot resume. out holds a line for each of lines, the same text
  // styled; any other out is not used, and the lines show plain.
  type Highlighter interface {
  	Highlight(lang string, lines []string, state any) (out []string, next any)
  }
  type HighlighterFunc func(lang string, lines []string, state any) (out []string, next any)

  // WithHighlighter highlights code blocks, lang the info string's first
  // word: a Printer each code line as it prints it, from the state its
  // last left; a Doc whose Renderer is Plain, nil or stream.Plain, its
  // blocks, and a Doc with another Renderer ignores it. In a Doc:
  //   - a committed segment's blocks are highlighted when it is rendered,
  //     once a width, as the segment is;
  //   - the tail's open fence resumes from the state after its last
  //     complete line, so each line is highlighted once; under a nil state,
  //     from its first line each frame;
  //   - the tail's closed fences are kept by (info, body), up to twice the
  //     byte limit, and all dropped when that overflows;
  //   - a width change and a new revision drop both caches.
  func WithHighlighter(h Highlighter) Option
  // A block over any limit shows plain: 512 KiB, 10,000 lines, 4 KiB in a
  // line, each kept by zero or less.
  func WithHighlightLimits(maxBytes, maxLines, maxLineBytes int) Option

  // SetRevision is the revision of what the Renderer and the Highlighter
  // draw with, such as a theme.Theme's ID, which stream does not import; a
  // different one drops every cache, the segments' and the frame's too.
  func (d *Doc) SetRevision(rev uint64)

  // WithMath makes a Doc's holdback hold an unclosed \( or $$ in the
  // partial line too, as the other constructs, until it closes, the line
  // completes or Finish; off by default. A Printer, which prints whole
  // lines only, ignores it.
  func WithMath() Option

  // TableOptions is how FitTable lays a table out.
  type TableOptions struct {
  	Width  int           // cells; 0 or less: no limit, each column its widest cell
  	Method ansi.Method
  	Border *glyph.Border // nil: the Markdown form, pipes and a delimiter row, as
  	                     // Plain draws; else a box of its glyphs, such as glyph.Set's
  	                     // Light, whose ASCII twin glyph.ASCII() has
  }

  // FitTable lays out a closed table, rows its source lines: the header,
  // the delimiter row and the body. Each column starts at its widest cell
  // and shrinks no further than its longest word, the space left shared in
  // proportion to what each still wants; a cell wraps at its spaces and no
  // word is broken, so numbers and URLs stay whole; each row is padded by
  // grapheme cluster under Method, so a wide glyph leaves no ghost cell.
  // When the longest words do not fit, the rows are key/value records,
  // "header: value", a blank line between records; when a pair does not
  // fit one line, stacked records, the header on its line and the value
  // under it indented two cells, wrapped; when a word is wider than Width,
  // the source rows as they came.
  func FitTable(rows []string, o TableOptions) []string
  // A Printer and Plain lay their tables out by FitTable at their width,
  // in the Markdown form: a table wider than the width is a record each
  // row, where it ran past the width before. WithoutTableLayout keeps the
  // rows as they came.
  // Corrected 2026-11-03 (D20): Plain alone fits its tables at its width;
  // a Printer lays them out by FitTable with no width, as wide as their
  // widest cells, as 0023-MADR decided.

  // Bounded holds live output, such as a tool's (A1 §4): all of it up to a
  // byte limit, then its first lines and its last, each line capped, the
  // lines between counted. The count is in lines, which no width changes.
  // It is used on one goroutine.
  type Bounded struct{ /* ... */ }
  func NewBounded(o ...Option) *Bounded
  func (b *Bounded) Write(p []byte) (int, error) // split into lines, each cleaned by its
  	// rules, then StatusLine's, whose rules make one line
  func (b *Bounded) Lines() []string // the head; the omitted line, when lines are left out;
  	// the tail, the last line as far as it has come. A head line an SGR
  	// left open is closed before the omitted line.
  func (b *Bounded) Omitted() int    // the lines left out
  func (b *Bounded) Render(width int) string // Lines wrapped at width cells, by its method
  func WithMaxBytes(n int) Option     // 64 KiB: all is kept up to it
  func WithHeadLines(n int) Option    // 20: the first lines kept past it
  func WithTailLines(n int) Option    // 20: the last lines kept past it
  func WithMaxLineBytes(n int) Option // 4 KiB: a longer line is cut, ending with the Ellipsis
  func WithGlyphs(g glyph.Set) Option // Bounded's Ellipsis: glyph.Unicode() by default
  func WithCatalog(c msgcat.Catalog) Option
  	// each With… of a size: zero or less keeps the default
  	// WithRules and WithMethod apply to a Bounded too: its rules
  	// safetext.Text() by default, and never less than safetext.StatusLine(),
  	// its floor, which keeps SGR, and OSC 8 links to http, https and mailto
  	// targets, alone; so a program keeps a tool's colours with
  	// WithRules(safetext.StatusLine())

  // stream's first text a user is shown (0024-MADR D9).
  const MsgOmitted msgcat.MessageID = "stream.omitted" // "(%d lines not shown)"
  func Messages() msgcat.Table
  ```

  *The agent's readings, for the owner's review:*
  * **The dialect needs no code:** the `Printer`, and so `Plain`, already
    strike `~~` alone (`stream/inline.go:182-186`), so the step's tests of
    it are tests only.
  * **A `Renderer` reports through `ViewRenderer`,** an interface beside
    `Renderer`, so that a renderer such as Step 7's glamour adapter, which
    reports nothing, still works, as A1 says "A `Renderer` that cannot
    report a table returns none, and `Doc` derives the line map itself".
  * **A link's ID is its source offset,** which no tail render, width or
    chunking changes; A1 asks only that the IDs continue.
  * **The theme reaches the `Doc` as a number,** `SetRevision`, since A6
    Q9 keeps `theme` out of `stream`. `DocPane` sets it to its theme's ID
    at each draw, and, under a theme of ID 0, to a new revision of its
    own each draw, so that a literal theme draws right at the cost of
    every cache, as A7 item 4 says.
  * ~~**`FitTable` changes the `Printer`'s and `Plain`'s tables** past their
    width: from a row running past it to records. That is what A1 Q8,
    "so `Plain` can lay out tables", chose; it changes the line mode's
    output, which a release note says.~~ *Corrected 2026-11-03 (D20):*
    `Plain`'s alone; the `Printer`'s are as 0023-MADR decided.
  * **`Bounded`'s floor is `StatusLine`, not `Text`:** A1 closes "an
    escape sequence cut off at the head", which `Text`, removing every
    sequence, would leave none of. Its default is still `Text`.
  * **The numbers the agent chose, with no source:** `Bounded`'s 64 KiB,
    20 and 20 lines and 4 KiB a line, and the closed fences' budget, twice
    the byte limit. The 512 KiB, 10,000 lines and 4 KiB are A1's, codex's.
  * **`DocPane` takes no catalogue:** it draws no `Bounded` and no text of
    `stream`'s, so the step's "`DocPane` passes its catalogue on" has
    nothing to pass. Should a pane draw a `Bounded` later, it takes one
    then.
* *(D19)* **Tests,** besides those above: `View`'s rows joined are
  `Render`'s; its offsets and side-tables the same for any chunking,
  after `Finish` too; a fence split by a relaxed cut one `ViewCode`;
  `ViewTable`'s cells as the source has them; a Renderer that is not a
  `ViewRenderer` giving segment offsets and no side-table; `SetRevision`
  rendering every segment again; `WithHighlightLimits`; `FitTable` with a
  box border and with none, each fallback at the width that brings it;
  the `Printer`'s table past its width as records; `Bounded` under the
  byte limit keeping all, over it head and tail with the omitted line, a
  line past its cap cut with the Ellipsis, a line with no newline
  bounded, `WithRules(safetext.StatusLine())` keeping colour and the
  floor removing what `StatusLine` does, `WithCatalog`'s text; and
  `Messages()` through `msgcat.Check`.
* *(D19)* **Mutations,** besides those above: a link's ID by its row;
  `SetRevision` keeping the segments; the closed-fence cache never
  dropped; the highlight limits ignored; `FitTable` breaking a word; a
  fallback skipped; `Bounded`'s floor `Text`, or none; a line's cap
  ignored.

### Step 8: the examples, the guide, and the root's release

* **Examples,** each with `// guide:` regions: `ExampleDoc` (`stream`),
  `ExampleScheduler` (`frame`), `ExampleFilter` (`inputfiltertea`), and
  `ExampleDocPane` (`streamtea`), which streams an answer from a goroutine
  through a `Buffer` into a `DocPane` in a workspace, and shows the one
  line that re-arms `frametea.Wait`.
* **`docs/guides/streaming-content.md`:** feeding a `Doc` from the event
  loop or from goroutines; the scheduler, and why `Wait` is re-armed;
  choosing a pacer; the filter, `Attach` and paste bursts; what `safetext`
  removes and adding trusted links with `termsvc.Linker`; what may differ
  before `Finish`; the line mode's use of the cores; `Interpret`,
  `RenderView` and `Bounded`; and the `DocPane`'s commands.
* **`docs/architecture.md`,** `README.md` and `docs/README.md`'s rows;
  "Migrating to `v2.N.0`" in `docs/guides/migrating.md`.
* **The release,** a minor of the root, under
  [0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)'s
  rule 4 and `docs/guides/releasing.md`: the API listed by `apidiff`; the
  notes; the disclosure guard; the owner's push and tag; CI green on both;
  and a consumer smoke test from the proxy, in a scratch module, that
  streams a document into a `DocPane` and paces a line-mode reply with
  `frame` and no Bubble Tea, whose closure for Linux, macOS and Windows
  holds no Charm module but `x/ansi` for the cores, with a control that
  fails against the release before.
  *Added 2026-11-03 (D21):* the step fixes `DocPane`'s last row under a
  finished `Doc`, which its rehearsal found: `stream/streamtea/pane.go`
  and its test join the step's files.

### Step 7: `stream/glamourmd`, a nested module

* **Runs after Step 8's tag** (0010-MADR §3).
* **Facts re-read,** each recorded before the requirement is added:
  glamour's newest release (`v2.0.1` on 2026-11-01) and its `go.mod`; A7
  item 5's lines in its source; each added module's licence, from its
  licence file; `GOWORK=off govulncheck ./...` in the module, on a scratch
  copy; the `StyleConfig` fields that set bullets, rules, quote bars and
  table separators, and any glyph glamour draws that no field controls. A
  finding stops the step for the owner.
* **The module:** `stream/glamourmd/go.mod`, `module
  github.com/maccavelli/go-tui-lib/stream/glamourmd`, `go 1.27.2`,
  requiring `github.com/maccavelli/go-tui-lib/v2` at Step 8's tag and
  `charm.land/glamour/v2`, with no `replace`; `go work use
  ./stream/glamourmd` in the same commit; it imports the root's exported
  packages only.
* **The API** (A2, A4):

  ```go
  func NewRenderer(o ...Option) *Renderer // WithTheme(theme.Theme), WithStyle(ansi.StyleConfig),
  	// WithEmoji(), WithChromaFormatter(name)
  func (r *Renderer) Render(src string, width int) (string, error) // safe for concurrent use
  ```

  `WithTheme` builds the whole `StyleConfig` from the theme and its glyphs:
  `Document.Margin` 0, `CodeBlock.Chroma` nil and `CodeBlock.Theme` a
  built-in chroma style chosen by the background, the table's three
  separators from `glyph`, `CenterSeparator` always set with the others.
  The renderers are an LRU of four, keyed by width, `theme.Theme.ID` and
  emoji, each behind its own lock. It never calls
  `WithEnvironmentConfig`, `WithStylePath` or `WithStylesFromJSONFile`.
* **Tests:** the cache by its key, at most four, the least recently used
  out; concurrent `Render`s under `-race` at one key and at two; chroma's
  registry never holds "charm" after renders under two themes, each
  theme's code in its own colours; `WithTheme` under the dark, light and
  unknown palettes and the ASCII and no-colour profiles, goldens across
  the matrix; the ASCII glyphs give an all-ASCII list, rule, quote and
  table; a table renders with no panic; no document margin;
  `streamtest.CheckRenderer` with this renderer, chunking invariance and
  `Finish` exactness, and the stated differences recorded construct by
  construct.
* **Mutations:** the cache ignoring the width; ignoring the theme ID;
  the lock removed (`-race`); the background ignored; `CodeBlock.Chroma`
  left set (the two-theme test); `CenterSeparator` unset (the table test).
* **Glossary:** `Renderer` joins "Deliberate" and `sharedNames` (the
  glossary's planned pair).
* **Checks,** per 0010-MADR §4, in `stream/glamourmd` with `GOWORK=off`:
  vet, `-race`, `LC_ALL=C`, golangci-lint for three systems, `go mod
  tidy -diff`, govulncheck; the tests once in workspace mode; the root's
  gates unchanged, its `go.mod` with no glamour line.
* **The release:** the owner's tag `stream/glamourmd/v0.1.0`; a smoke
  test from the proxy, `go get` of the adapter in a scratch module and a
  render; and a second scratch module with the root alone, whose `go.mod`,
  `go.sum` and `go list -m all` hold no glamour, goldmark, chroma or
  bluemonday.

## Verification

* Every step's mutations fail their named tests, each failure quoted.
* `python3 -m tools.verify gates` passes on macOS, Linux and Windows for
  every step, with the step's files.
* `TestCoresAreCharmFree` passes with `stream`, `frame` and `inputfilter`,
  and fails on a scratch copy that imports Bubble Tea into each;
  `TestNarrowClosures` still holds `safetext` to its rule.
* `TestDefaultsInventory` passes, with no row of this PLAN left `planned`
  at Step 8.
* `make apicheck` reports 0 incompatible changes at every step.
* `python3 -m tools.verify names` reports no clash at every step.
* Every driver's test runs inside `testing/synctest`, and none leaves a
  goroutine; no adapter starts one but `inputfiltertea`'s drain send *and,
  added 2026-11-01 (D2), `frametea.Forward`'s reader*.
* The root `go.mod` has no glamour, goldmark, chroma or bluemonday line;
  `stream/glamourmd/go.mod` requires exactly the root's tag and glamour,
  with no `replace`; `go mod tidy -diff` is clean in both modules.
* The identifier scan finds nothing; CI is green on every push and on both
  tags; both smoke tests pass from the proxy.

## Rollout and Rollback

* **Rollout.** Steps 1 to 8 land one commit each, unreleased until Step
  8's minor release of the root; Step 7 then tags the adapter. A program
  takes the root release with no change: every addition is new API.
* **Rollback.** Before a push, a step is one local commit. After it, a
  fix goes forward in a patch. If glamour's graph proves unacceptable, the
  adapter is not tagged again, its directory and `go.work` line go, and
  `stream` keeps working with `Plain` or a program's own `Renderer`; tags
  are never deleted.

## Execution Record

*2026-11-01.* The owner committed A7 and the rewritten PLAN (`7c23536`)
and said "committed, approved to proceed".

### Step 1: records, the planned defaults and the reserved names

* **Deviation D1, 2026-11-01: the slow tick's control.**
  * **Found** while naming Step 1's rows: Step 1 named
    `frame.WithSlowTick`, and Step A1.2's API had no such option, its
    `Demand.Interval(fps)` fixing 83 ms, so the default had no control
    (AGENTS.md, API convention 12). The PLAN's own text, not the tree.
  * **Resolutions offered:** a `frame.Ticks` value, its zero fields taking
    the defaults (recommended); options on `TickCmd`; or a `decided` row
    with no control.
  * **Chosen by the owner,** 2026-11-01: "A frame.Ticks value
    (Recommended)". Step A1.2's sketch and Step 1's row are corrected;
    0009-MADR A7 item 11 is annotated. `python3 -m tools.verify names`
    over the 41 planned names, `Ticks` among them: "names: 41 planned
    name(s), clean". No file is added to the step.
* **Deviation D2, 2026-11-01: `frametea.Forward`'s goroutine.**
  * **Found** with D1: `Forward` must read `Frames` on a goroutine to call
    `send`, and Verification allowed no adapter goroutine but
    `inputfiltertea`'s drain send.
  * **Resolutions offered:** keep `Forward` and list its goroutine
    (recommended); drop it; or move it to the core.
  * **Chosen by the owner,** 2026-11-01: "Keep Forward, list it
    (Recommended)". Step 3's adapter and tests, Verification, and Step
    1's rows say so. No file is added to the step.
* **A new requirement, 2026-11-01,** during this step: the owner's "we
  need a plain streaming wrapper to enhance extensibility and
  flexibility", then "An io.Writer around the stream (Recommended)" and,
  for Q14, "stream.Writer; frame's → Output (Recommended)". 0009-MADR A8
  records it, accepted; Step A8.1 is added, after Step A1.1, and waits
  for the owner's approval; Step A1.2 is renamed for Q14. This step's
  records carry them: A8, Step A8.1, the `stream.WithCRLF` row, and the
  reserved names. No source changes.

**What changed:**

* This PLAN `in-progress`; `docs/README.md`'s rows for the MADR (A6 to A8
  accepted) and the PLAN (in progress, Step 1).
* `docs/guides/defaults.md`: seven rows added to `stream`'s section,
  three for Step 5, one each for Steps A1.1 and A8.1, and two for Step
  A1.4's limits, whose options that step names; and three sections, `frame and frame/frametea`
  (10 rows), `inputfilter and inputfilter/inputfiltertea` (7) and
  `stream/streamtea` (3). Every row is `planned`, citing this PLAN and
  its step.
* `docs/glossary.md`, "Names the planned records must not take": the
  names A6 to A8 and D1 reserve, and why `safetext.Row` is not one.
* 0009-MADR: A7 item 11 annotated (D1); A8 and its answer.

**Checks:**

* `TestDefaultsInventory` passes with the new rows. Two mutations, on
  scratch copies, show it reads them:
  * a `frame` row marked `built`: "docs/guides/defaults.md:329 (A frame's
    rate): built, but no package frame: no required module provides
    package github.com/maccavelli/go-tui-lib/v2/frame";
  * a `stream/streamtea` row citing a PLAN that does not exist:
    "docs/guides/defaults.md:358 (How queued lines are released): planned
    by 0009-PLAN-streaming-engine.md, which does not exist".

  `mutate: 2 killed, 0 survived, 0 errors`.
* `python3 -m tools.verify names` over the PLAN's planned types with
  `Ticks` (41) and over A8's four: clean.
* `make markdownlint check-records docs-check disclosure-check`: clean;
  the identifier scan: clean.
* **Gates,** `python3 -m tools.verify gates` with the step's five files,
  over the records as this step leaves them, A8 included:

  | Host | `gates` |
  | :--- | :--- |
  | macOS | 21 passed, 0 failed, 1 skipped (`gofmt`, no Go files) in 848 s |
  | `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1214 s |
  | `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2608 s |

  A first run on macOS and Linux, before A8 was written, passed the same;
  the first Windows run was stopped for the rerun above.
* **Step 1 complete, 2026-11-01,** for the owner's commit. Step A1.1 is
  next; Step A8.1, after it, waits for the owner's approval.

### Step A1.1: `Interpret`, in `stream` (Q13)

*2026-11-01.* The owner committed Step 1 (`8626c9f`) and said
"committed, approved for A8.1 and proceed to A1.1". Step A8.1 is approved;
it runs after this step.

**Facts re-read:**

* `stream` imports `safetext`, `glyph` (through it), `internal/enum` and
  `x/ansi` (`go list -deps`); `Interpret` adds no import.
* `x/ansi`'s `Method.DecodeSequenceInString(data, state, p)` splits text
  into grapheme clusters with their widths; its zero state is
  `NormalState`, and with a nil `Parser` it stores no parameter. Its
  parser, given a `Parser`, misreads 32 or more parameter separators
  (`internal/sanitize/separators.go`, 0023-MADR A10). So `Interpret` reads
  a sequence's parameters itself, at most 32 for an SGR, as `safetext`'s
  `KeepSGR` keeps, and gives `x/ansi` text alone.
* `safetext` writes invalid UTF-8 as one U+FFFD for each maximal subpart
  (`safetext/utf8.go`, unexported; `safetext`'s narrow rule keeps it in
  that package). `Interpret` cleans each run of printable bytes with
  `safetext.Clean` under the zero `Rules`, the strictest, which writes
  invalid UTF-8 so and removes format characters too; no logic is written
  twice. A run ends at an ASCII control byte, which no maximal subpart
  spans.
* `safetext`'s grammar for C1 code points, U+009B a CSI, U+009D an OSC,
  U+0090, U+0098, U+009E and U+009F strings, U+009C an ST, holds for
  `Interpret` too, so the two read a sequence alike.

* **Deviation D3, 2026-11-01: a cluster wider than two cells.**
  * **Found** by `FuzzInterpret`'s second 5-minute run, after 112 s, on
    the agent's own code: the input "ౙ్ౙ్ౙ00", 6 columns, `ansi.WcWidth`,
    gave "row 0 is 7 cells, over 6: "ౙ్ౙ్ౙ  00"". Under `ansi.WcWidth`,
    `x/ansi` gives a cluster the sum of its runes' widths: in a scratch
    module, `DecodeSequenceInString` measured the Telugu conjunct "ౙ్ౙ్ౙ"
    at 3 cells and the family "👨‍👩‍👧" at 6, and under `ansi.GraphemeWidth`
    at 1 and 2. The step's text said "a wide cluster taking two", and the
    code placed only one continuation cell, padding the rest with blanks.
  * **Resolutions offered:** a cluster takes every cell it measures
    (recommended); clamp a cluster to 2 cells, which makes the row's
    measured width disagree with its cells, a workaround; or split such a
    cluster into runes, which breaks the cluster model.
  * **Chosen by the owner,** 2026-11-01: "Any width, as measured
    (Recommended)". The step's behaviour is corrected above. The fuzz
    input is kept as a seed, `testdata/fuzz/FuzzInterpret/06a722286caffb19`,
    added to the step's files; table rows cover clusters of 3 and 6 cells.

**What changed:**

* `stream/interpret.go`: `Interpret`, `InterpretOptions`, `InterpretRow`,
  `DefaultInterpretRows` and `DefaultInterpretCols`. It reads its input as
  `safetext`'s grammar does, ESC and the C1 code points opening sequences,
  and keeps SGR (its own `;` and `:` parser, at most 32 groups of at most
  8 parts), CR, BS, HT, LF, and CSI A, B, C, D, G, K and J. Every other
  sequence and control is dropped; an OSC, DCS, SOS, PM or APC string over
  `safetext.DefaultMaxString` bytes is abandoned there. Text runs are
  cleaned by `safetext.Clean` under the zero `Rules`, then split into
  clusters by `Method.DecodeSequenceInString` with a nil `Parser`. A style
  is a comparable value, written as one canonical SGR at each change. Past
  twice `MaxRows`, the grid keeps its newest `MaxRows` rows, so dropping
  old rows costs one copy a `MaxRows` rows; the result is the newest
  `MaxRows`, without trailing empty rows.
* `stream/doc.go`: a paragraph on `Interpret`. `docs/architecture.md`:
  `stream`'s row. `docs/guides/defaults.md`: `Interpret`'s bounds `built`.
* Found by lint and fixed before the gates: gocyclo, `sgr`'s complexity
  44 over 40, so the set-and-clear attributes became the table
  `attrCodes`; gosec G115 on `uint8(g[1])`, so `underlines` is a lookup;
  staticcheck ST1018 on a test literal holding U+200B and U+202E as
  characters, so the test's format and combining characters are escapes.
* Found by the table, after D3's fix and before the commit: the agent's
  `clearCluster` read its loop bound from the cluster's first cell, which
  the first pass had blanked, so it cleared one cell alone and left the
  rest as continuations; the row "writing in a 3-cell cluster blanks it
  whole" failed: "" yx"; want [" y x"]". The bound is read once; mutation
  A1.1.10 holds it.

**Tests,** in `stream/interpret_test.go`:

* `TestInterpret`, 39 rows: text, LF, CR, BS, HT, each erase, each move,
  each dropped sequence and control, C1, invalid UTF-8, format characters,
  wide and combining clusters, clusters of 3 and 6 cells under
  `ansi.WcWidth` and of 2 under `ansi.GraphemeWidth` (D3), both caps.
* `TestInterpretSGR`, 16 rows: the `;` and `:` forms of 38, 48 and 58,
  4:n, a change of style mid-row, the reset, the C1 CSI, a 33-parameter
  SGR and an index past 255, each dropped.
* `TestInterpretString`, `TestInterpretDefaults` and
  `TestInterpretDeterministic`.
* `FuzzInterpret`: rows within `MaxRows`; no row wider than `MaxCols`
  under the method; `Styled` holds SGR and nothing else; `Plain` is
  `ansi.Strip(Styled)` and clean under `safetext.Line()`; two calls equal.
  Three seeds, and D3's input.

**Mutations,** `python3 -m tools.verify mutate`, on the final code:

| Mutation | Its failure |
| :--- | :--- |
| A1.1.1 one column a rune | `Interpret("世x\r\x1b[Cy") = ["世y"]; want [" yx"]` |
| A1.1.2 CR not returning to column 0 | `Interpret("abcdef\rXY") = ["abcdefXY"]; want ["XYcdef"]` |
| A1.1.3 HT to a multiple of 4 | `Interpret("a\tb\tc") = ["a   b   c"]; want ["a       b       c"]` |
| A1.1.4 an OSC kept as text | `Interpret("a\x1b]0;title\ab") = ["a0;titleb"]; want ["ab"]` |
| A1.1.5 `MaxRows` ignored | `Interpret("1\n2\n3\n4") = ["1" "2" "3" "4"]; want ["3" "4"]` |
| A1.1.6 the `:` form of SGR not read | `Interpret("\x1b[38:2::255:0:10mr") = [{"r" "r"}]; want one row "\x1b[0;38;2;255;0;10mr\x1b[m"` |
| A1.1.7 C1 code points read as text | `Interpret("\u009b1mx") = [{"x" "x"}]; want one row "\x1b[0;1mx\x1b[m"` |
| A1.1.8 a cluster written over left standing | `Interpret("世x\r\x1b[Cy") = ["世yx"]; want [" yx"]` |
| A1.1.9 one continuation cell only (D3) | `Interpret("ౙ్ౙ్ౙ00") = ["ౙ్ౙ్ౙ 00"]; want ["ౙ్ౙ్ౙ00"]` |
| A1.1.10 `clearCluster`'s bound read each pass (D3) | `Interpret("ౙ్ౙ్ౙx\r\x1b[Cy") = [" yx"]; want [" y x"]` |

`mutate: 10 killed, 0 survived, 0 errors`. The PLAN named six; A1.1.7 to
A1.1.10 were added for the C1 grammar, the cluster written over, and D3.
A1.1.4 is killed by `TestInterpret`, not by `FuzzInterpret` as the step
expected: the fuzz target's properties hold for an OSC's body read as
text.

**Fuzzing,** `FuzzInterpret`, 5 minutes each: on the code before the lint
fixes, 26,051,152 executions, no failure; after them, D3's failure at
112 s; after D3's fix, 35,539,438 executions, no failure.

**Benchmark,** `BenchmarkInterpret`, 1 MB of a test run's coloured output
with CR progress bars, five runs: 34.0 to 39.2 ms, 26.8 to 30.8 MB/s,
55,062,862 to 55,063,244 B and 290,099 to 290,100 allocations an
operation. `BenchmarkReplyRender`, beside it: 22,575 to 26,630 ns, 32,792
B, 200 allocations. The MADR sets no budget for `Interpret`.

**The API,** apidiff at the `Makefile`'s pin against `v2.9.0`: five
additions, `stream.DefaultInterpretCols`, `DefaultInterpretRows`,
`Interpret`, `InterpretOptions` and `InterpretRow`, and nothing else;
`make apicheck`: "against v2.9.0, 0 incompatible change(s)".

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 1347 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1232 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2578 s |

* The pre-add check of the three Go files, `make modernize`, the
  documents' checks and the identifier scan: clean.
* **Step A1.1 complete, 2026-11-01,** for the owner's commit, with
  `stream/testdata/fuzz/FuzzInterpret/06a722286caffb19`, a new file. Step
  A8.1, approved, is next.

### Step A8.1: `stream.Writer`, a plain streaming writer (0009-MADR A8)

*2026-11-01.* The owner committed Step A1.1 (`fa54b2e`) and said
"committed, proceed".

**Facts re-read:** the `Printer`'s API is `NewPrinter(o ...Option)`,
`Write(chunk string) string`, `Flush() string`, `At`, `Idle`, `SetWidth`
and `Source` (`stream/printer.go`); every printed line leaves it through
`emit`, which ends it with `\n` (`printer.go:163-168`); a prose line
wrapped at a width holds `\n` between its rows; the cleaned text holds no
`\r`, since `safetext.Text` drops a lone CR and makes CR LF one LF; `At`
builds a second `Printer` from this one's options (`printer.go:93`);
`stream.Option` is `interface{ apply(*Printer) }`, so the `Writer` passes
its options to its `Printer` and the interface does not change;
`safetext.ErrClosed` is "safetext: write after Close", the precedent for
`stream.ErrClosed`.

* **Deviation D4, 2026-11-01: the `Writer`'s defaults.**
  * **Found** reading the `Printer`'s options: 0009-MADR A8 item 2 says a
    `Writer` with no option gives "the input cleaned by `safetext.Text`,
    unstyled and unwrapped: plain", and this step's tests said the same;
    but a `Printer` lays a table out by default, and gives the cleaned
    input byte for byte only with `WithoutTableLayout` too
    (`stream/doc.go`; `options.go`, `WithoutTableLayout`).
  * **Resolutions offered:** the `Printer`'s defaults, the sentence
    corrected (recommended); a `Writer` that applies `WithoutTableLayout`
    first, with its own defaults row and a way back; or a second
    constructor for byte-for-byte output.
  * **Chosen by the owner,** 2026-11-01: "The Printer's defaults
    (Recommended)". A8 item 2 and this step's test are corrected; no row
    and no file are added.

**What changed:**

* `stream/writer.go`: `Writer`, `NewWriter`, `Write`, `WriteString`,
  `Close` and `ErrClosed` ("stream: write after Close"). Each write passes
  its input to the `Printer` and writes what is stable in one
  `io.WriteString`; a short write with no error is `io.ErrShortWrite`; the
  first error is kept and returned from then on.
* `stream/options.go`: `WithCRLF()`. `stream/printer.go`: the `crlf`
  field; `emit` writes a line's rows and its end with CR LF under it; `At`
  keeps it. `stream.Option` is unchanged: the `Writer` passes its options
  to its `Printer`.
* `stream/doc.go`: a paragraph on the `Writer`. `docs/guides/line-mode.md`,
  "Stream a reply": plain output with no region. `docs/architecture.md`:
  `stream`'s row. `docs/guides/defaults.md`: the line-end row `built`, and
  a `decided` row, the destination left open at `Close` (A8 item 5).

**Tests,** in `stream/writer_test.go`: `TestWriterCopy`, `io.Copy` of
`testdata/reply.txt` through `iotest.OneByteReader`, `iotest.HalfReader`
and a whole reader, at widths 0, 60 and 100, with and without
`DefaultMarkup`, equal to the `Printer`'s output byte for byte;
`TestWriterDefaults` (D4); `TestWriterString`; `TestCRLF`, for a
`Printer`, a `Writer` and `Printer.At`; `TestWriterError`;
`TestWriterShortWrite`; `TestWriterClose`; `FuzzWriter`, random splits of
any input equal to the `Printer`'s output for the whole; `ExampleWriter`;
`BenchmarkWriter`. Every existing `stream` test passes unchanged,
`TestReplyBudget` among them.

**Mutations,** `python3 -m tools.verify mutate`:

| Mutation | Its failure |
| :--- | :--- |
| A8.1.1 `Write`'s stable lines not written | `TestWriterCopy`: "one byte, width 0, styled false: the Writer's output differs from the Printer's" |
| A8.1.2 `Close` not flushing | the same |
| A8.1.3 the error not sticky | `TestWriterError`: "a write after the failure = 5, full; want 0 and the same error" |
| A8.1.4 `Close` closing the destination | `TestWriterClose`: "Close closed the destination 1 times" |
| A8.1.5 `WithCRLF` ignored | `TestCRLF`: "Printer: … want every line ending CR LF, and no lone CR or LF" |
| A8.1.6 a `Write` after `Close` accepted | `TestWriterClose`: "Write after Close = 1, <nil>; want 0, ErrClosed" |
| A8.1.7 a wrapped line's rows left LF | `TestCRLF`: "Printer: "A paragraph long\nenough to be wrapped\nat twenty columns.\r\n…"" |
| A8.1.8 a short write not reported | `TestWriterShortWrite`: "a short write = <nil>; want io.ErrShortWrite" |
| A8.1.9 `At` not keeping `WithCRLF` | `TestCRLF`: "Printer.At: … want every line ending CR LF" |

`mutate: 9 killed, 0 survived, 0 errors`. The step named six; A8.1.7 to
A8.1.9 were added for the wrapped rows, the short write and `At`.

**Fuzzing,** `FuzzWriter`, 5 minutes: 5,779,391 executions, no failure.

**Benchmark,** five runs: `BenchmarkWriter` 22.2 to 30.6 µs, 30,232 B and
216 allocations an operation; `BenchmarkReplyRender`, beside it, 20.6 to
24.2 µs, 32,792 B and 200 allocations.

**The API,** apidiff at the `Makefile`'s pin against `v2.9.0`: this step
adds `stream.ErrClosed`, `NewWriter`, `WithCRLF` and `Writer`, beside Step
A1.1's five; `make apicheck`: "against v2.9.0, 0 incompatible change(s)".
`TestNoTypeNameMeansTwoThings`, `TestCoresAreCharmFree` and
`TestNarrowClosures` pass.

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 956 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1341 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2615 s |

* The pre-add check of the five Go files, the documents' checks and the
  identifier scan: clean.
* **Step A8.1 complete, 2026-11-01,** for the owner's commit. Step 3,
  `frame`, is next.

### Step 3: `frame`, its scheduler, and `frame/frametea`

*2026-11-01.* The owner committed Step A8.1 (`ae933d6`) and said
"committed proceed".

**Facts re-read:** Bubble Tea's frame-rate bounds, 60 and 120
(`renderer.go:13-14` at `v2.0.10`); a `tea.Cmd` is `func() tea.Msg`;
`testing/synctest` gives `Test`, `Wait` and `Sleep`, and `Test` fails when
the bubble's goroutines deadlock at its end; `internal/enum`'s `Names`
gives an enum's `String`, `MarshalText` and `UnmarshalText`
(`inline/reflow.go`), and `internal/conformance/enums_test.go`'s `enums`
table lists each one; a core is listed in `cores`
(`internal/conformance/charmfree_test.go`) and in the depguard rule
`charmfree` (`.golangci.yml:107-131`), "Keep the two lists the same".

* **Deviation D5, 2026-11-01: the depguard rule.**
  * **Found** reading `.golangci.yml`: a core is listed in `charmfree` as
    well as in `cores`, so that a direct Bubble Tea import fails at lint;
    the step's scope named `internal/conformance/` and not `.golangci.yml`,
    and Step 4's the same.
  * **Resolutions offered:** add `"**/frame/*.go"` here and
    `"**/inputfilter/*.go"` in Step 4 (recommended); or leave lint to the
    conformance test alone, the two lists differing, a workaround.
  * **Chosen by the owner,** 2026-11-01: "Add it, Steps 3 and 4
    (Recommended)". `.golangci.yml` is added to both steps' scope rows.
* **Deviation D6, 2026-11-02: the frame message's name.**
  * **Found** by the pre-add check, on the agent's code: revive's
    `exported` rule refused `frame.FrameMsg`, "type name will be used as
    frame.FrameMsg by other packages, and that stutters; consider calling
    this Msg", the name 0009-MADR A4 gave, "a message type is named for
    what it carries". No exported type of the repository stutters, and
    none carries a revive exception.
  * **Resolutions offered:** `frame.DueMsg`, a frame that fell due, as
    `termcap.CapsMsg` names what it carries (recommended); keep `FrameMsg`
    with the repository's first `nolint:revive`; or revive's `frame.Msg`,
    which A4 refused. `python3 -m tools.verify names` over `DueMsg`, `Msg`
    and `TickMsg`: clean.
  * **Chosen by the owner,** 2026-11-02: "frame.DueMsg (Recommended)". The
    code, `docs/architecture.md`, the glossary's reserved names and this
    PLAN's steps say `DueMsg`; 0009-MADR A4's row is annotated, so that
    where the MADR says `FrameMsg`, it reads `DueMsg`. No file is added.

**What changed:**

* `frame/`, a new core: `doc.go`; `gear.go`, `Gear` (`Smooth`,
  `CatchUp`; tokens "smooth" and "catch-up") from `internal/enum`;
  `queue.go`, `Queue[T]`, a ring buffer whose zero value is empty, with
  `Push`, `Len`, `Snapshot` and `Drain`, an `iter.Seq[T]` that asks its
  `Pacer` once and leaves what an early stop does not take; `pacer.go`,
  `Pacer`, `PacerFunc`, `Release`, `Immediate`, and `Adaptive` with its
  seven options, each keeping its default at zero or less;
  `scheduler.go`, `DueMsg`, `Scheduler`, `NewScheduler`, `WithFPS`,
  `Request`, `RequestIn`, `Frames` and `Close`.
* The `Scheduler` holds one mutex. A request after which a frame falls
  due starts one goroutine with `WaitGroup.Go`; it waits on a
  `time.Timer`, is woken through a channel of capacity one when the
  deadline moves earlier, sends the frame without blocking, a frame
  waiting unread serving the request, and stops. `Close` closes `stop`,
  waits for the goroutine, and closes `Frames`.
* `frame/frametea/`, the adapter: `Wait` and `Forward` (D2), and
  `ExampleWait`, a program that re-arms `Wait` on each frame.
* `internal/conformance`: `frame` joins `cores`, and `frame.Gear` the
  `enums` table. `.golangci.yml`: `"**/frame/*.go"` joins `charmfree`
  (D5).
* `docs/architecture.md`: the package graph, the table's two rows, the
  cores' list and the tree. `docs/guides/defaults.md`: the six `frame`
  rows of Step 3 `built`. `docs/glossary.md`: `DueMsg` (D6).

**Tests:** `TestSchedulerMerges`, `TestSchedulerRate` (6 cases),
`TestRequestIn` (5), `TestSchedulerSeq`, `TestSchedulerUnread` and
`TestSchedulerClose`, each inside `testing/synctest.Test`; `TestAdaptive`
(10 tables of codex's transitions), `TestAdaptiveOptions`, `TestImmediate`
and `TestGear`; `TestQueue`, `TestQueueWraps` and `TestQueueAllocs`, a
frame of 64 items allocating at most twice, nothing an item;
`ExampleAdaptive`; and in `frametea`, `TestWait` and `TestForward`. All
pass under `-race`.

* Two test faults of the agent's, fixed: `TestSchedulerSeq` wanted
  2 s / 60 for the second frame where two intervals of a 60th are
  33.333332 ms; and two mutations first survived (below), so two tests
  were made to hold what they claimed.

**Mutations,** `python3 -m tools.verify mutate`, on the final code:

| Mutation | Its failure |
| :--- | :--- |
| 3.1 `Frames` unbuffered | `TestSchedulerMerges`: "no frame came" |
| 3.2 `Frames` of capacity 2 | `TestSchedulerUnread`: "the request made while the first waited: a frame came, {… Seq:2}" |
| 3.3 the frame-rate cap skipped | `TestSchedulerMerges`: "within the frame's interval: a frame came, {… Seq:2}" |
| 3.4 the exit hold ignored | `TestAdaptive`: "step 1 (at 10ms, depth 2, oldest 30ms): {Take:1 Gear:smooth}; want take 2 in catch-up" |
| 3.5 the severe bypass ignored | `TestAdaptive`: "step 5 (at 500ms, depth 64, oldest 0s): {Take:1 Gear:smooth}; want take 64 in catch-up" |
| 3.6 `Drain` from the tail | `TestQueue`: "Drain of 5 gave [19 19 19 19 19]; want 0 to 4" |
| 3.7 `Close` not stopping the goroutine | `TestSchedulerClose`: "Close waited 1h0m0s for the pending frame's deadline; want it stopped at once" |
| 3.8 `Seq` not raised | `TestSchedulerSeq`: "frame 1 is {… Seq:1}; want Seq 2 at 16.666666ms" |
| 3.9 a later deadline winning | `TestRequestIn`: "no frame came" |
| 3.10 a deadline moved earlier not woken | `TestRequestIn`: "no frame came" |
| 3.11 `Forward`'s stop not ending it | `TestForward`: "panic: deadlock: all goroutines in bubble are blocked" |
| 3.12 `Wait` after `Close` giving a frame | `TestWait`: "Wait after Close gave frame.DueMsg{…}; want nil" |
| 3.13 the core importing Bubble Tea | `TestCoresAreCharmFree`: "frame (GOOS=linux) links charm.land/bubbletea/v2: frame → charm.land/bubbletea/v2" |

`mutate: 13 killed, 0 survived`. The PLAN named seven; 3.9 to 3.13 were
added for the earliest deadline, the adapter and the core's closure.
3.7 and 3.10 first survived: under `synctest`, a bubble whose goroutines
all block advances its clock, so the hour's timer fired and `Close`
returned late, not never; and the second request reached the goroutine
before it first read the deadline. `TestSchedulerClose` now wants `Close`
to take no time, and `TestRequestIn` lets the goroutine block on the
first deadline before moving it. 3.7 is named for what it changes; the
PLAN's "`Close` not waiting for the goroutine" leaves it nothing to wait
for under `synctest`, and was replaced.

**The API,** apidiff at the `Makefile`'s pin against `v2.9.0`: two
packages added, `frame` and `frame/frametea`, beside Steps A1.1 and
A8.1's additions; `make apicheck`: "against v2.9.0, 0 incompatible
change(s)".

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 801 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1331 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2609 s |

* The pre-add check of the twelve Go files, `make modernize`, the
  documents' checks, `internal/conformance` and the identifier scan:
  clean.
* **Step 3 complete, 2026-11-02,** for the owner's commit. Step A1.2,
  `Coalescer`, `Demand`, `Output` and `Upgrader`, is next.

### Step A1.2: `Coalescer`, `Demand`, `Output` and `Upgrader`

*2026-11-02.* The owner committed Step 3 (`600348d`) and said "committed,
proceed".

**Facts re-read:** 0003-REPORT §8.17 gives grok's `TickDemand{None, Slow
(83 ms), Fast}`, grok's stall "after 5 s with no progress", and Kilo's
"100 ms timer" (`0003-REPORT-agent-tui-ecosystem-research.md:1983-2000`);
Bubble Tea `v2.0.10`'s `cursed_renderer.go:643` builds an
`uv.NewTerminalRenderer`, and ultraviolet's `terminal_renderer.go:111-117`,
at the version the root requires, says it writes "the minimal necessary
escape sequences to transition the terminal to the new buffer state".
Each holds. `python3 -m tools.verify names` over the step's ten type
names, D7's two among them: clean.

* **Deviation D7, 2026-11-02: the messages of `TickCmd` and
  `CoalesceCmd`.**
  * **Found** reading the step: it gives `frametea.TickCmd(d, t)` and
    `CoalesceCmd(due)`, each a `tea.Cmd`, and names no message for either;
    a command returns one, and the program switches on its type.
  * **Resolutions offered:** `frame.TickMsg{At, Demand}` and
    `frame.FlushMsg{At}` in the core, beside `DueMsg`, since they name no
    Charm type (0020-MADR D2) (recommended); the same two in `frametea`,
    against D2, which a line-mode host could not use; or `DueMsg` for
    both, with which a program cannot tell which command to re-arm.
  * **Chosen by the owner,** 2026-11-02: "Two messages in frame
    (Recommended)". The step's text, the glossary's reserved names and
    0009-MADR A1 §3 are annotated. No file is added.
* **Deviation D8, 2026-11-02: how long `Upgrader` remembers a key's
  newest generation.**
  * **Found** reading the step: "a result whose generation is older than
    the newest submitted for its key is dropped" does not say how long
    the newest is kept.
  * **Resolutions offered:** while a job for the key runs, the key
    forgotten when it ends, memory bounded by the jobs in flight
    (recommended); forever, an entry for every key ever submitted; or
    the first with an exported `Forget(key)`, which nothing asks for.
  * **Chosen by the owner,** 2026-11-02: "While a job for the key runs
    (Recommended)". The step's text is annotated; the MADR's "a stale one
    is dropped" holds as written.
* **Deviation D9, 2026-11-02: `Output` under `tea.WithOutput`.**
  * **Found** reading Bubble Tea `v2.0.10` before writing `Output`:
    * Bubble Tea finds the terminal only as `p.output.(term.File)`
      (`tty_unix.go:30`, `tty_windows.go:36`), x/term `v0.2.2`'s
      `io.ReadWriteCloser` with `Fd() uintptr` (`terminal.go:9-12`). It
      takes from it the window size (`tea.go:1051-1059`,
      `tty.go:109-126`), the resize listener (`tea.go:696`), Windows'
      virtual-terminal mode (`tty_windows.go:36-48`), and through
      colorprofile `v0.4.3` the colour profile (`env.go:34-36`). An
      `Output` that is an `io.Writer` alone gives a program a 0 by 0
      window, no resize, no colour, and no virtual-terminal mode on
      Windows.
    * Bubble Tea restores the terminal straight after its last flush
      (`tty.go:33-49`). An `Output` whose `Write` returns before its bytes
      are written lets that restore come first: on Windows the last
      sequences then reach a console whose virtual-terminal mode is off,
      and print as text.
    * 0009-MADR A1 §3 and A7 item 10 take an `io.Writer` under
      `tea.WithOutput` to be transparent; they are wrong about it. Nothing
      in the tree is affected: `Output` was unbuilt.
  * **Resolutions offered:** `Output` forwards `Fd` and `Read`, and
    `Write` waits for its own bytes up to a bound, `WithWriteWait`, 100 ms
    (recommended); forward `Fd` and `Read` and return at once, leaving the
    Windows defect, close to a workaround; or document `Output` for
    line-mode hosts alone, which drops A1's purpose for Bubble Tea.
  * **Chosen by the owner,** 2026-11-02: "File methods + bounded wait
    (Recommended)". The step's text and 0009-MADR A7 item 10 are
    annotated; `docs/guides/defaults.md` gains two rows, the wait and
    `Output`'s goroutine, which convention 12 needs and Step 1 did not
    list. 100 ms is the agent's choice, not a researched number: it is
    the coalescer's wait, long against a write to a terminal that keeps
    up and short against the 5 s stall. No file is added.

**What changed:**

* `frame/coalescer.go`: `Coalescer[T]`, `NewCoalescer`, whose nil
  `apply` panics, a programmer's error, whose nil `mergeable` merges
  every item, and whose `maxWait` keeps 100 ms at zero or less; `Push(v,
  now) (due, ok)`, a deadline when a delta starts a batch, and a barrier
  applying the batch, then itself alone; `Flush`, reusing the batch's
  slice; and `FlushMsg` (D7).
* `frame/demand.go`: `Demand` (`None`, `Slow`, `Fast`; tokens "none",
  "slow", "fast") from `internal/enum`; `Combine`, the greatest;
  `Ticks`, whose zero or negative field keeps 83 ms or a 60th of a
  second, and `Interval`, 0 for `None` and for a demand with no token;
  and `TickMsg` (D7).
* `frame/output.go`: `Output`, `NewOutput`, `OutputOption`,
  `WithStallAfter` (5 s), `WithWriteWait` (100 ms, D9), `OutputEvent`
  (`Stalled`, `Resumed`; tokens "stalled", "resumed"), `Events`, `Fd`,
  `Read`, `Close` and `ErrClosed`.
  * One goroutine, started by `WaitGroup.Go` when a write is queued and
    none is in flight, writes the pending buffer, again while more is
    pending, and stops; two buffers are swapped, so the steady state
    allocates nothing for the bytes.
  * A `Write` copies its bytes into the pending buffer and waits on that
    buffer's `done` channel or the write wait. Once a wait has run out,
    later `Write`s return at once until a write completes, so that a
    stalled terminal holds the program for one wait, not one a draw; this
    is how D9's "only a slower write lets the program go on" behaves
    across draws, and is the agent's.
  * A short write is `io.ErrShortWrite`. The first failed write's error
    is every later `Write`'s, with 0, and `Close`'s.
  * The stall timer is a `time.AfterFunc` started with each write, which
    sends `Stalled` only while that write, by its number, is in flight;
    `Resumed` follows when it completes. `Events` holds one event: an
    event still unread when its opposite comes is taken back, so a
    program that reads late reads the current state, and a send never
    blocks.
* `frame/upgrader.go`: `Upgrader[K, R]`, `NewUpgrader`, `Submit`,
  `Results` (unbuffered), `Close`, and `UpgradeMsg`. Each job is a
  goroutine started by `WaitGroup.Go`, with a context of its own; a job
  replaced, or the `Upgrader` closed, has its context cancelled and its
  result dropped, and a key is forgotten when its job ends (D8). A job
  replaced at the moment its result is read may still deliver it, and
  its `Gen` says which it is; the documentation says so.
* `frame/frametea/frametea.go`: `TickCmd`, nil for `None`, otherwise a
  `tea.Tick` returning `frame.TickMsg`; `CoalesceCmd`, a `tea.Tick` to
  the deadline returning `frame.FlushMsg`; and `WaitUpgrade`, generic,
  nil once the `Upgrader` is closed.
* `frame/doc.go`: the package comment names the new types.
* `internal/conformance/enums_test.go`: `frame.Demand` and
  `frame.OutputEvent` join `enums`.
* `docs/guides/defaults.md`: the step's four rows and D9's two `built`.
  `docs/architecture.md`: the graph's line, the two packages' rows and
  the tree.

**Tests:** `TestCoalescerModel`, every one of the 3,279 scripts of 1 to 7
steps over a delta, a barrier and a frame against a model of the batches
and the deadlines; `TestCoalescerWait` (3 cases) and `TestCoalescerNil`;
`ExampleCoalescer`; `TestCombine` (4) and `TestTicks` (8);
`TestOutputKeepsUp`, `TestOutputStalled`, `TestOutputWithdraws`,
`TestOutputErrors` (2 writers) and `TestOutputFile`;
`TestUpgraderReplaces`, `TestUpgraderStale`, `TestUpgraderGenerations`
and `TestUpgraderClose`; and in `frametea`, `TestTickCmd` (4),
`TestCoalesceCmd` and `TestWaitUpgrade`. Every driver's test runs inside
`testing/synctest.Test`; all pass under `-race`, three times over.

* Three test faults of the agent's, fixed before any mutation ran:
  `TestOutputKeepsUp` looked for no event after `Close`, when a closed
  channel gives the zero value, `Stalled`; `TestUpgraderGenerations`
  submitted a lower generation the moment the higher one's result was
  read, before that job had ended and its key was forgotten, which D8
  ignores, as it should; and `TestOutputErrors`'s writer failed every
  write, so that a mutation making the error not sticky would have
  survived, and now fails only its first.

**Mutations,** `python3 -m tools.verify mutate`, on the final code:

| Mutation | Its failure |
| :--- | :--- |
| A1.2.1 the barrier not flushing first | `TestCoalescerModel`: "\"db\" applied [[{1 true}] [{0 false}]]; want [[{0 false}] [{1 true}]]" |
| A1.2.2 `TickCmd(None)` a command | `TestTickCmd`: "TickCmd(None) is a command; want nil" |
| A1.2.3 `Output` dropping the pending buffer | `TestOutputStalled`: "writes [\"first\"]; want the first, then the rest as one" |
| A1.2.4 `Output` writing a second time while one is in flight | `TestOutputStalled`: "no event; want stalled", then the bubble's deadlock |
| A1.2.5 `Upgrader` keeping a stale result | `TestUpgraderStale`: "the replaced job returned: result {Key:a Gen:1 Result:stale}" |
| A1.2.6 `Upgrader` not cancelling the replaced job | `TestUpgraderReplaces`: "0 jobs cancelled; want the replaced one" |
| A1.2.7 `Write` returning before its bytes are written | `TestOutputStalled`: "Write(\"first\") took 0s; want 100ms" |
| A1.2.8 later `Write`s waiting while the terminal is slow | `TestOutputStalled`: "Write(\"b\") took 100ms; want 0s" |
| A1.2.9 `Fd` not forwarded | `TestOutputFile`: "Fd = 43; want the file's, 42" |
| A1.2.10 an unread event not withdrawn | `TestOutputWithdraws`: "a stall over before it was read: event stalled" |
| A1.2.11 a write's error not sticky | `TestOutputErrors`: "the next Write = 1, \<nil\>; want 0, terminal gone" |
| A1.2.12 `WithStallAfter` ignored | `TestOutputWithdraws`: "0 events waiting at the stall; want Stalled" |
| A1.2.13 a lower generation replacing the running job | `TestUpgraderGenerations`: "result {Key:a Gen:3 Result:three}; want {a 5 five}" |
| A1.2.14 a key not forgotten when its job ends | `TestUpgraderGenerations`: "no result; want {a 3 again}" |
| A1.2.15 `Close` not cancelling the jobs | `TestUpgraderClose`: "panic: deadlock: all goroutines in bubble are blocked" |
| A1.2.16 the default wait ignored | `TestCoalescerWait`: "maxWait 0s: Push gave 0s, true; want a deadline 100ms on" |
| A1.2.17 a deadline for every delta | `TestCoalescerModel`: "\"dd\" gave deadlines at steps [0 1]; want [0]" |
| A1.2.18 `Combine` the last, not the greatest | `TestCombine`: "Combine([none fast slow]) = slow; want fast" |
| A1.2.19 `Slow`'s default lost | `TestTicks`: "slow: Interval(slow) = 0s; want 83ms" |

`mutate: 19 killed, 0 survived, 0 errors`. The PLAN named six, A1.2.1
to A1.2.6; the rest hold D8, D9 and the defaults. A1.2.7 was first
written as `if true`, which left `slow` unused: the mutant did not
compile, and its "kill" was a build failure. It was replaced by
`if slow || true`, killed as above.

**The API,** apidiff at the `Makefile`'s pin against `v2.9.0`: the
packages `frame` and `frame/frametea` added, as in Step 3, with the
names above; `make apicheck`: "against v2.9.0, 0 incompatible
change(s)".

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 808 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1451 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2623 s |

* The pre-add check of the twelve Go files, the documents' checks,
  `internal/conformance` and the identifier scan, untracked files
  included: clean.
* **Step A1.2 complete, 2026-11-02,** for the owner's commit. Step 4,
  `inputfilter`, its driver, `inputfilter/inputfiltertea` and the
  workspace's extension point, is next.

### Step 4: `inputfilter`, its driver, `inputfilter/inputfiltertea`, and the extension point

*2026-11-02.* The owner committed Step A1.2 (`21eeeca`) and said
"committed, proceed".

**Facts re-read:** at Bubble Tea `v2.0.10`, `tea.MouseMsg` is an
`fmt.Stringer` with `Mouse() Mouse` (`mouse.go:46-51`), and `tea.Mouse` is
`X, Y`, `Button` and `Mod`; `paste.go` gives `PasteMsg{Content}`,
`PasteStartMsg` and `PasteEndMsg`; `tea.WithFilter` takes
`func(Model, Msg) Msg`, and the event loop runs it on every message it
receives, a program's `Send` included (`tea.go:760-769`). `local` is at
`workspace/render.go:538-556`, with the four cases. `keymaptea.FromTea`
converts a `tea.Key` field for field. `termkey.Input` is `Kind` (`InputKey`,
`InputPaste`, `InputUnknown`), `Key`, `Text`, `Raw` and `Truncated`.
codex's `paste_burst.rs`, at `4b4a5cf` (2026-11-01), is read under D11.

* **Deviation D10, 2026-11-02: the core's API.**
  * **Found** reading the step: it names `PasteBurst` with `Key`, `Tick`
    and `BurstAction`, and a `Flusher` on a channel with `Close`, and no
    type for a wheel event or the motion throttle in a core that cannot
    name `tea.Mouse`, no `BurstAction` values, no deadline a driver can
    read from `PasteBurst`, and nothing a `Flusher`'s channel carries.
  * **Resolutions offered:** the whole logic in the core, the pure
    `Wheel`, `Throttle` and `PasteBurst` and the `Flusher` over them, the
    adapter converting (recommended); the pure stages alone, the adapter
    combining them under its lock, against 0020-MADR D2; or the API
    written into the records first, as a change of its own.
  * **Chosen by the owner,** 2026-11-02: "Full core + Flusher
    (Recommended)", with the API the step's text now shows. `python3 -m
    tools.verify names` over `Scroll`, `Wheel`, `Throttle`, `PasteBurst`,
    `PasteBurstConfig`, `BurstAction`, `Flusher`, `Flush`, `Option`,
    `Filter` and `CoalescedWheelMsg`: clean.
* **Deviation D10a, 2026-11-02: who takes a timed flush.**
  * **Found** working out D10's `Flusher` before any code: a flush taken
    on the timer's goroutine and sent on the channel lets a key typed
    after a burst's idle end, while the paste is in transit through the
    program's `Send`, reach the model before it; and with no reader, the
    residual is lost, where 0009-MADR §6 has it join the next wheel event.
  * **Resolutions offered:** `Flushes() <-chan time.Time` says a flush
    fell due, of capacity one, and `Flush(now) Flush` takes it on the
    caller's goroutine (recommended); or as D10 first gave it.
  * **Chosen by the owner,** 2026-11-02: "Channel says when; caller takes
    (Recommended)". The step's text shows it.
* **Deviation D11, 2026-11-02: codex's burst, as it is.**
  * **Found** in codex's `paste_burst.rs` at `4b4a5cf`: it holds the first
    fast ASCII character and starts a burst at the second within 8 ms
    (`BeginBufferFromPending`, lines 233-242); its 3 characters
    (`PASTE_BURST_MIN_CHARS`, line 159) start one only by retro-capture,
    taking back text already in the textarea, which a filter cannot do,
    since the keys have reached the model. It also reads an Enter as a
    newline for 120 ms after a burst (`PASTE_ENTER_SUPPRESS_WINDOW`, lines
    160 and 347-351), which 0009-MADR §6 does not name.
  * **Resolutions offered:** `MinChars` 3, as the MADR says, up to two
    fast characters held, then given as keys when no third comes
    (recommended), or 2, as codex; and the Enter window left out
    (recommended), or added at 120 ms, an Enter after a flush given as a
    one-line paste.
  * **Chosen by the owner,** 2026-11-02: "3, as the MADR says
    (Recommended)" and "Leave it out (Recommended)". 0009-MADR §6 is
    annotated with codex's facts.
* **Deviation D12, 2026-11-02: `WithWheelProfile`.**
  * **Found** reading the step: its adapter lists `WithWheelProfile(n)`,
    which Step A1.3 builds and tests.
  * **Chosen by the owner,** 2026-11-02: "A1.3, where its test is
    (Recommended)". It is struck from this step's list, and A1.3 builds it
    as an `inputfilter.Option` (D10).

**What changed:**

* `inputfilter/`, a new core: `doc.go`; `wheel.go`, `Scroll`, `Wheel`
  (`Add`, `Residual`, `Due`, and an `Interval` field, 16 ms at zero or
  less) and `Throttle` (`Allow`, and `Interval`); `burst.go`,
  `PasteBurstConfig`, `BurstAction` (`Pass`, `Hold`, `Release`; tokens
  "pass", "hold", "release") from `internal/enum`, and `PasteBurst`
  (`NewPasteBurst`, `Key`, `Tick`, `Due`); `flusher.go`, `Option`,
  `WithInterval`, `WithoutWheelCoalescing`, `WithoutMotionThrottle`,
  `WithPasteBurst`, `Flush`, and `Flusher` (`NewFlusher`, `Scroll`,
  `Motion`, `Input`, `Flushes`, `Flush`, `Close`). It imports `termkey`,
  `internal/enum` and the standard library.
  * `PasteBurst`, by D11: a character typed, a key press with text,
    shifted at most, or on Windows with AltGr's Ctrl and Alt, starts or
    joins a run when it comes within the gap of the last; an Enter or a
    Tab joins a run or a burst but starts none; at `MinChars` the run is
    a burst, which takes every such key until it has been idle; a run
    that ends short is given back as its keys; any other input ends what
    is held first; a bracketed paste turns detection off for good. The
    Tab is codex's (`paste_burst.rs:4`, "`KeyCode::Char`,
    `KeyCode::Enter`, and `KeyCode::Tab`"), which §6's "codex's rules"
    take in.
  * The `Flusher` holds one mutex over the three stages. A call after
    which something falls due starts one goroutine with `WaitGroup.Go`,
    which waits on a `time.Timer`, sends the deadline on `Flushes`, of
    capacity one, without blocking, and stops; `Close` stops it at once
    and closes `Flushes`. With D10a, nothing leaves the stages but by a
    call.
* `inputfilter/inputfiltertea/`, the adapter: `Filter`, `NewFilter`, its
  `Filter` method, `Attach`, and `CoalescedWheelMsg` with `Mouse`,
  `String`, `WithMouse` and `WheelDelta`; `ExampleFilter`.
  * A wheel sum of one notch reaches the model as the
    `tea.MouseWheelMsg` it was, and of more as a `CoalescedWheelMsg`, so
    that a pane that reads only Bubble Tea's wheel still scrolls; the
    agent's reading of §6's "many wheel notches as one message".
  * Keys, releases and pastes go through `Flusher.Input` only while the
    `Filter` is attached, since a held character needs `Attach` to be
    given back; the conversions are `keymaptea.FromTea` and `ToTea`,
    which carry all six of `tea.Key`'s fields (`key.go:302-343`).
  * Bubble Tea gives a bracketed paste as `PasteStartMsg`, `PasteMsg`
    and `PasteEndMsg`; the `PasteMsg` is the `termkey.InputPaste` that
    turns detection off, so the step's "after a `tea.PasteStartMsg`" is
    held by the sequence, and its mutation by "a bracketed paste leaving
    detection on".
  * `Attach`'s one goroutine, from `Attach` to its `stop`, sends the
    drains and the flushes through the program's `Send`; `stop` closes
    the `Flusher`. It is for the program's end: a filter returns one
    message, and a send from `stop` would deadlock a `stop` called from
    `Update`, so what the `Filter` holds at `stop` ends with the program,
    and after it every message passes as it comes. A second `Attach`
    panics, a programmer's error. The first `stop` written gave back what
    the Flusher held with the next message; `TestFilterAttach` found the
    message after it held with no drain to release it, and the agent
    changed it to this before any mutation ran.
  * A message the burst is not given, such as a wheel event, passes a
    run or burst the Flusher still holds, which has not yet been given;
    the order the `Filter` keeps is that of what it gives, its held
    messages before any after them.
* `workspace/render.go`: `local` gains the case, a `tea.MouseMsg` with
  `WithMouse(tea.Mouse) tea.Msg` translated by calling it; its comment
  says so, and `docs/guides/building-workspaces.md`'s mouse list.
* `internal/conformance`: `inputfilter` joins `cores`, and
  `inputfilter.BurstAction` the `enums` table. `.golangci.yml`:
  `"**/inputfilter/*.go"` joins `charmfree` (D5).
* `docs/architecture.md`: the graph's two lines, the two packages' rows,
  the cores' list and the tree. `docs/guides/defaults.md`: the step's
  five rows `built`.

**Tests:** `TestWheelSum`, 100 notches within 16 ms, the first at once
and 99 as the residual; `TestWheelInterval` (3 intervals);
`TestWheelReverses`; `TestThrottle` (2); `TestPasteBurst`, 20 cases:
codex's, a 3-character burst, Enter and Tab inside, a burst ended by an
arrow, Windows' longer idle, typing 50 ms apart passing through, and
the rest of D11's rules; `TestPasteBurstDue`; `ExamplePasteBurst`;
inside `testing/synctest.Test`, `TestFlusherResidual`,
`TestFlusherBurst`, a key after the idle end following the paste
(D10a), `TestFlusherUnread`, `TestFlusherOptions` and
`TestFlusherClose`; in `inputfiltertea`, inside `synctest`, over a stand-in
event loop whose `Send` blocks, so that a `send` on the calling
goroutine would deadlock the bubble, `TestFilterBurst`,
`TestFilterBracketed`, `TestFilterOrder`, `TestFilterWheel`,
`TestFilterPasses` and `TestFilterAttach`; `TestCoalescedWheelMsg`; and
`TestMouseExtensionPoint` in `workspace`. All pass under `-race`, three
times over.

* Test faults of the agent's, fixed: `TestFlusherResidual` and
  `TestFlusherBurst` waited for a flush with `synctest.Wait`, which does
  not move the clock, short of its deadline, and now sleep to it; and
  `TestPasteBurst` lacked a character after a burst's idle end, so that
  a `Key` that did not first expire what is due would have survived; the
  case was added before the mutations ran.

**Mutations,** `python3 -m tools.verify mutate`, on the final code:

| Mutation | Its failure |
| :--- | :--- |
| 4.1 the direction reset removed | `TestWheelReverses`: "down, down, up held {… DY:1 …}; want DY -1" |
| 4.2 every motion sample passed | `TestThrottle`: "interval 0s: passed at [0s 1ms 2ms …]" |
| 4.3 the FIFO bypassed by new input | `TestFilterOrder`: "before the drains the model got \"paste(abc) wheel(wheeldown)\"; want the paste alone" |
| 4.4 `local` ignoring `WithMouse` | `TestMouseExtensionPoint`: "logs got []; want one coalesced wheel at (3,2) with 7 notches" |
| 4.5 a bracketed paste leaving detection on | `TestPasteBurst`: "a bracketed paste turns detection off: \"[x]\"; want \"[x] a b c\"" |
| 4.6 `Close` not stopping the goroutine | `TestFlusherClose`: "Close waited 1h0m0s for the deadline; want it stopped at once" |
| 4.7 the first notch not passing at once | `TestWheelSum`: "Due = -2562047h47m16.854775808s, true; want 16 ms after the first" |
| 4.8 the residual not taken by `Flush` | `TestFlusherResidual`: "passed 1, flushed {… HasScroll:false}; want 1, then 99" |
| 4.9 an Enter inside a burst not kept | `TestPasteBurst`: "Enter inside a burst: \"[abc] enter d\"; want \"[abc\\nd]\"" |
| 4.10 an arrow not ending a burst first | `TestPasteBurst`: "a burst ended by an arrow: \"up\"; want \"[abc] up\"" |
| 4.11 Windows' idle ignored | `TestPasteBurst`: "Windows' longer idle: \"[abc] d\"; want \"[abcd]\"" |
| 4.12 a run not given back as keys | `TestPasteBurst`: "two held, given back as keys: \"\"; want \"a b\"" |
| 4.13 a character after the idle end joining the burst | `TestPasteBurst`: "a character after the idle end starts again: \"[abcx]\"; want \"[abc] x\"" |
| 4.14 the drain not asked for again | `TestFilterOrder`: "the drains gave \"key(up)\"; want the arrow, then the wheel" |
| 4.15 the core importing Bubble Tea | `TestCoresAreCharmFree`: "inputfilter (GOOS=linux) links charm.land/bubbletea/v2: inputfilter → charm.land/bubbletea/v2" |
| 4.16 `MinChars` ignored | `TestPasteBurst`: "MinChars, Gap and Idle set: \"a b\"; want \"[ab]\"" |

`mutate: 16 killed, 0 survived, 0 errors`. The PLAN named six; its "a
`PasteStartMsg` leaving detection on" is 4.5, as above, and its "`Close`
not waiting" is 4.6, which `synctest` can observe, as Step 3's 3.7 found.
The rest hold D10, D10a and D11.

**The API,** `make apicheck`: "against v2.9.0, 0 incompatible
change(s)"; the packages `inputfilter` and `inputfilter/inputfiltertea`
added.

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 654 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1553 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2679 s |

* The pre-add check found one finding in the agent's code, errcheck on
  `_ = fl.f.Close()`, which this repository's `check-blank` refuses; it
  is the repository's reasoned `//nolint:errcheck`, as
  `stream/printer.go:119-120` has, since a `Flusher`'s `Close` always
  returns nil. Then the pre-add check of the thirteen Go files, the
  documents' checks, `internal/conformance` and the identifier scan,
  untracked files included: clean.
* **Step 4 complete, 2026-11-02,** for the owner's commit. Step A1.3, the
  input stages, measured first, is next.

### Step A1.3: the input stages, measured first

*2026-11-02.* The owner committed Step 4 (`e6947c6`) and said "committed,
proceed".

**The measurement,** before any code: a scratch module outside the tree,
requiring the tree by a `replace` and Bubble Tea `v2.0.10` at the
root's ultraviolet, wrote each input through an `os.Pipe` to a
`tea.Program` (`WithInput`, `WithOutput(io.Discard)`,
`WithoutSignalHandler`, `TERM=xterm-256color`), 100 ms after it started,
the chunks the gap apart, and quit it 300 ms after the last. Its model
kept every message but `WindowSizeMsg`, `ColorProfileMsg` and `EnvMsg`.
The same chunks went through `termkey.Decoder`, with `Flush` between
them, as a line-mode host does when a gap passes its timeout. What each
received:

* **An SGR mouse report,** `"\x1b[<0;10;5M"`, a left press at column
  10, row 5:
  * whole: tea `MouseClickMsg{X:9 Y:4 left}`; termkey
    `unknown("\x1b[<0;10;5M")`;
  * split at every byte and in each of its ten two-way splits, 10 ms
    apart: tea `MouseClickMsg{X:9 Y:4 left}`, every time;
  * 60 ms and 200 ms apart, byte by byte or split after the ESC: tea
    `key(esc) key([) key(<) key(0) key(;) key(1) key(0) key(;) key(5)
    key(shift+m)`; split after `"\x1b["`: `key(alt+[) key(<) …
    key(shift+m)`; split after 3 to 10 bytes: `uv.UnknownEvent` holding
    the head, as `"\x1b[<0;10;5"`, then a key for each byte after it,
    `key(shift+m)`. termkey gave the same keys, and `unknown(…)` for the
    same heads, at every gap.
* **A focus report,** `"\x1b[I"`: whole, tea `FocusMsg`, termkey
  `unknown("\x1b[I")`; 10 ms apart, `FocusMsg` in every split; 60 ms
  apart byte by byte, `key(esc) key([) key(shift+i)`, split after the
  ESC, `FocusMsg` in the one run, a gap near the 50 ms join, and after
  `"\x1b["`, `key(alt+[) key(shift+i)`; 200 ms apart, keys in every
  split.
* **An X10 report,** `"\x1b[M"`, a press, then the column and row each
  plus 32, row 1:
  * as the bytes, columns 94, 95, 96, 120, 200 and 223: tea
    `MouseClickMsg{X:93 Y:0}`, `{X:94 Y:0}`, `{X:95 Y:0}`, `{X:119 Y:0}`,
    `{X:199 Y:0}` and `{X:222 Y:0}`, each right;
  * as ConPTY relays it, each byte of 0x80 and over the UTF-8 of its
    value: columns 94 and 95, whose bytes are under 0x80, right; column
    96, `"\x1b[M \u0080!"`, tea `MouseClickMsg{X:161 Y:95 left} key(!)`;
    120, `{X:161 Y:119} key(!)`; 200, `{X:162 Y:135} key(!)`; 223,
    `{X:162 Y:158} key(!)`: the click misplaced, and the row a typed
    `!`. termkey: `unknown("\x1b[M \u0080") key(!)`, and so on.
* **A bracketed paste,** `"\x1b[200~a\r\nb\rc\x1b[201~"`: tea
  `PasteStartMsg PasteMsg{"a\r\nb\rc"} PasteEndMsg`; termkey
  `paste("a\r\nb\rc")`. Empty, `"\x1b[200~\x1b[201~"`: tea
  `PasteStartMsg PasteMsg{""} PasteEndMsg`; termkey `paste("")`.

**What it shows:** Bubble Tea joins a report split within its 50 ms
escape timeout and no further; it decodes X10 by the byte, so ConPTY's
re-encoding from column 96 misplaces the click and types its row; and it
keeps a paste's CR and CRLF, and gives an empty paste as an empty
`PasteMsg`. termkey, the line-mode path, does the same with its host's
flush. Bubble Tea already does none of the stages measured, so none is
dropped; and a report split after its first one or two bytes reaches the
model as `esc` or `alt+[` and keys, with no unknown event to begin from.

* **Deviation D13, 2026-11-02: the step's API.**
  * **Found** after the measurement: the step names nine stages and gives
    `PastedPath`'s signature, `ClipboardImageRequestMsg`, `Capture` and
    `Quarantine` by name, and `WithWheelProfile`; reassembly and X10
    repair yield mouse and focus reports, which `termkey.Input` does not
    carry, the core has no type for, and Bubble Tea does not export a
    decoder of.
  * **Resolutions offered:** the step in two, the stages whose shape is
    plain first, and reassembly, X10 repair, late replies and the
    typeahead after their API is reviewed (recommended); the whole API
    written into the records first, with no code; or every choice asked
    at once, and all built in one step.
  * **Chosen by the owner,** 2026-11-02: "Write the whole API first". The
    step's text now carries it, for the owner's review and commit; no
    code is written until the owner approves it. 0009-MADR A1 §6 is
    annotated, the glossary reserves the names, and
    `docs/guides/defaults.md` gains the defaults, `planned`.
    `python3 -m tools.verify names` over `Action`, `Report`,
    `ReportKind`, `MouseButton`, `Reassembler`, `ReassembleConfig`,
    `ReplyMatcher`, `ReplyMatcherFunc` and `ClipboardImageRequestMsg`:
    clean. Step 4's `BurstAction` is in no tag (`git tag --contains
    e6947c6` lists none), so its rename to `Action` owes no deprecation.
* **Deviation D14, 2026-11-02: where reassembly may begin.**
  * **Found** in the measurement: a report split after its first one or
    two bytes reaches the model as `esc` or `alt+[` and keys, with no
    unknown event; to claim it, a stage would hold every Esc, which
    0009-MADR §6 refuses, "a bare Esc flushes anything a stage holds, so
    Esc never waits". `termcaptea.Prober.IsReplyFragment`
    (`prober.go:184-238`) begins from an unknown head too.
  * **Resolutions offered:** only from an unknown head of three bytes or
    more, read through `internal/termevent` (recommended); or holding Esc
    and `alt+[` for a window, against §6.
  * **Chosen by the owner,** 2026-11-02: "Only from an unknown head
    (Recommended)". The step's API states the limits it leaves.
* **Step A1.3's API written, 2026-11-02,** a change to the records alone,
  for the owner's review and commit; its code waits for the owner's
  "proceed".

*2026-11-02.* The owner committed the API and said "committed, proceed".

* **Deviation D15, 2026-11-02: the clipboard request's control.**
  * **Found** working the API through before any code: it gives
    `inputfiltertea.WithoutClipboardImageRequest() inputfilter.Option`,
    which the adapter cannot make, since `inputfilter.Option` is opaque,
    an interface whose method only `inputfilter`'s functions satisfy
    (AGENTS.md, API convention 4; `inputfilter/flusher.go`); and the
    Flusher's `termkey.Input` cannot mark a paste as the request, so the
    adapter could not learn the setting from it either.
  * **Resolutions offered:** the option the core's, with a `Flusher`
    method, `ClipboardImageRequest(in termkey.Input) bool`, that the
    adapter and a line-mode host ask (recommended); the adapter's own
    options, with a `Core` wrapper for the core's, changing Step 4's
    `NewFilter`; or no control, the conversion listed as a floor.
  * **Chosen by the owner,** 2026-11-02: "Core option + core test
    (Recommended)". The step's text and the defaults row are corrected.

**What changed,** to the API the owner committed (`ed80a46`), with D15:

* `inputfilter/action.go`: `Action`, Step 4's `BurstAction` renamed, in
  no release; `burst.go`'s `PasteBurst` returns it.
* `inputfilter/report.go`: `Report`, `ReportKind`, `MouseButton` and
  `ParseReport`, SGR, X10 and focus, the button's bits read as
  ultraviolet's `parseMouseButton` reads them; SGR numbers over 65,536,
  a `+` sign or an empty part refused.
* `inputfilter/reassemble.go`: `ReplyMatcher`, `ReplyMatcherFunc`,
  `ReassembleConfig` and `Reassembler`. Details the API left, the
  agent's:
  * a head is an `InputUnknown` of three bytes or more: a CSI whose
    bytes after `ESC [` are all parameters or intermediates, 0x20 to
    0x3f, or an OSC, DCS or APC with no BEL or `ESC \`; a key is the
    sequence's next byte when it types one printable byte, or in a
    string sequence is Ctrl+G, BEL, or a `\` after an ESC the head ended
    with. A string reply split before its ST's ESC is not rejoined,
    since that ESC arrives as a bare Esc, which gives back what is held;
  * an X10 head's pieces must come in one read: the keys it ran into
    within 5 ms, the agent's number, so that a click at a wide
    terminal's column 162 or 163 with a key typed after it is not
    "repaired". Only a column ConPTY re-encoded is repaired; a row of 96
    or more leaves a lone byte that is no rune, which is not;
  * a held input's `Raw` is copied, since termkey's may share its
    buffer.
* `inputfilter/typeahead.go`: `Typeahead`. `inputfilter/paste.go`:
  `ClipboardImageRequestMsg`, `NormalizePaste` and `PastedPath`, which
  takes codex's quoting, `file://` decoding and drive and UNC forms, and
  grok's anchor; a `file://` URL naming another host is refused.
* `burst.go`: grok's four `PasteBurstConfig` fields and
  `PowerShellPasteBurst`, `Idle` 10 ms beside the API's values, grok's
  continue timeout (`event_loop.rs:3440`). A run now keeps the inputs it
  took, so that `LinesOnly` can give one back as its keys.
* `wheel.go`: `EventsPerNotch`; the sum holds events, and whole notches
  pass.
* `flusher.go`: the options `WithWheelProfile`, `WithoutReassembly`,
  `WithReassemblyWindow`, `WithReplies`, `WithoutX10Repair`,
  `WithoutPasteNormalization` and `WithoutClipboardImageRequest` (D15);
  the methods `Capture`, `Replay`, `Quarantine` and
  `ClipboardImageRequest` (D15). `Input` and `Flush` run the stages in the
  API's order. `Capture` holds at most 4,096 inputs, the agent's bound on
  untrusted input; `Quarantine` drops what the stages hold, and the
  wheel's sum, and its deadline holds for `Scroll` and `Motion` too.
* `inputfilter/inputfiltertea/inputfiltertea.go`:
  * a `uv.UnknownEvent` is read through `internal/termevent.Decode`, so
    the adapter imports no ultraviolet, as `termcaptea` does;
  * the message each unknown head and each suspect X10 click came as is
    kept, at most 64, and given back when its input comes back as it
    went;
  * a suspect X10 click is re-encoded with x/ansi's `EncodeMouseButton`;
  * pastes reach the stages attached or not, so that normalisation and
    the clipboard request need no `Attach`; keys and unknown heads only
    while attached, as Step 4;
  * `Capture`, `Replay`, whose messages go through the drains, and
    `Quarantine`, which drops keys, pastes, mouse and focus messages, and
    held input, and passes the rest.
* `internal/conformance`: `inputfilter.Action`, `MouseButton` and
  `ReportKind` in `enums`. `docs/architecture.md`, `docs/glossary.md`,
  `inputfilter/doc.go`: the names. `docs/guides/defaults.md`: the eight
  rows of Step A1.3 `built`.

**Tests:** `TestParseReport`, 27 reports; `TestReassembleSplits`, every
two-way split of an SGR report, a DA1, a kitty keyboard and an OSC 11
reply, through `termkey.Decoder` with a flush between, 60 ms apart;
`TestReassembleX10`, columns 95, 96, 120, 200 and 223 as ConPTY relays
them, in one read and 6 ms apart; `TestReassembleEnds`; `TestNormalizePaste`;
`TestPastedPath`, 20 cases; `TestTypeahead`; `TestPowerShellPasteBurst`, 8;
`TestWheelProfile`; `TestFlusherStages`, `TestFlusherWindow`,
`TestFlusherCapture`, `TestFlusherQuarantine` and
`TestClipboardImageRequest`; in the adapter, `TestFilterSplitReport`,
`TestFilterX10`, the four clicks the measurement found misread,
`TestFilterPastes`, `TestFilterCapture`, `TestFilterQuarantine`, and
`TestReportAgreesWithTea`, which runs a `tea.Program` over a pipe and
holds `ParseReport`, made a message, to the message Bubble Tea's own
reader gives for each of 26 reports. All pass under `-race`, three times
over.

* Test faults of the agent's, fixed: `TestReassembleX10` gave an X10
  report's pieces 60 ms apart, when they come in one read;
  `TestFlusherWindow` waited with `synctest.Wait`, which does not move
  the clock, Step 4's fault again; `TestFlusherCapture` wanted a newline
  where `show` writes `\n`. And two mutations first survived: the wheel's
  leftover events were never left over in `TestWheelProfile`, which gained
  a case that leaves two; and the X10 window in `next` was dead code,
  since `Key` gives back an expired head first, so the clause was
  removed and the mutation moved to the wait itself.

**Mutations,** `python3 -m tools.verify mutate`, on the final code:

| Mutation | Its failure |
| :--- | :--- |
| A1.3.1 reassembly skipped | `TestReassembleSplits`: "an SGR report split \"\\x1b[<\"\|\"0;10;5M\": u(\"\\x1b[<\") 0 ; 1 0 ; 5 shift+m; want u(\"\\x1b[<0;10;5M\")" |
| A1.3.2 X10 repair skipped | `TestReassembleX10`: "column 96: u(\"\\x1b[M \\u0080\") !; want one report" |
| A1.3.3 late replies passed as keys | `TestReassembleSplits`: "a DA1 reply split \"\\x1b[?\"\|\"62;22c\": u(\"\\x1b[?\") 6 2 ; 2 2 c; want" none |
| A1.3.4 Esc not flushing | `TestReassembleEnds`: "esc after a head: esc u(\"\\x1b[<0;1\"); want the head, then it" |
| A1.3.5 CR left in a paste | `TestFlusherStages`: "a paste:" the CR kept |
| A1.3.6 the window ignored | `TestReassembleEnds`: "Due 500ms; want 80ms" |
| A1.3.7 a whole reply matched and still given | `TestReassembleEnds`: "a whole reply the matcher knows: u(\"\\x1b[?62;22c\"); want it dropped" |
| A1.3.8 a Report's column off by one | `TestReportAgreesWithTea`: "tea gave tea.MouseClickMsg{X:9, Y:4, …}; ours is tea.MouseClickMsg{X:10, Y:4, …}" |
| A1.3.9 a rune of U+0080 to U+00FF not repaired | `TestReassembleX10`: "column 96: u(\"\\x1b[M \\u0080\") !; want one report" |
| A1.3.10 `Typeahead` not cutting at Esc | `TestTypeahead`: "Typeahead gave \"h i backspace shift+enter alt+enter [p] A z [q]\"" |
| A1.3.11 `Quarantine` passing input before its deadline | `TestFlusherQuarantine`: "a paste before the deadline passed" |
| A1.3.12 `PastedPath` accepting prose | `TestPastedPath`: "PastedPath(\"hello\", linux) = \"hello\", true; want \"\", false" |
| A1.3.13 `LinesOnly` ignored | `TestPowerShellPasteBurst`: "a single line given back: \"[abcd]\"; want \"a b c d\"" |
| A1.3.14 `EnterCtrlJ` ignored | `TestPowerShellPasteBurst`: "Enter then Ctrl+J: \"a enter ctrl+j\"; want \"[a\\\\n]\"" |
| A1.3.15 the wheel's leftover events dropped | `TestWheelProfile`: "9 events, 4 of them held within an interval: 2 notches; want 3, the 2 over carried" |
| A1.3.16 `WithoutClipboardImageRequest` ignored | `TestClipboardImageRequest`: "WithoutClipboardImageRequest still reports the request" |
| A1.3.17 the X10 one-read window ignored | `TestReassembleX10`: "column 96, its key 6 ms after: u(\"\\x1b[M \\x80!\"); want it as it came, not one read" |
| A1.3.18 the adapter not suspecting an X10 click | `TestFilterX10`: "column 96: the model got […MouseClickMsg{X:161, Y:95…}, KeyPressMsg{Text:\"!\"…}]" |
| A1.3.19 a head given back as nothing | `TestFilterSplitReport`: "after the window the model got []uv.Event(nil); want the head" |
| A1.3.20 `Capture` not holding | `TestFlusherCapture`: "during Capture: h u(\"\\x1b[<0;1;1M\") up …" |
| A1.3.21 `FirstGap` ignored | `TestPowerShellPasteBurst`: "the second key past 2 ms: \"[ab\\\\nc]\"; want \"a [b\\\\nc]\"" |

`mutate: 21 killed, 0 survived, 0 errors`: the PLAN's five, D13's ten,
and six for D15, the X10 window and the adapter's conversions.

**The API,** `make apicheck`: "against v2.9.0, 0 incompatible
change(s)"; `inputfilter` and `inputfilter/inputfiltertea` are in no tag.

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 527 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1093 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2665 s |

* The pre-add check's first run found eight findings, all in the
  agent's code, each fixed: `"windows"` three times (goconst), now
  `goosWindows`; `len(rest) > 0` for `rest != ""` (gocritic); two
  conversions of a rune checked to be U+00FF or under to a byte (gosec
  G115), each given the repository's reasoned `//nolint:gosec`; an
  untagged switch (staticcheck QF1002); `ansi.MouseButton(m.Button)`,
  which is already x/ansi's type (unconvert); and `messages`' `out`,
  nil at every call (unparam), dropped. The 21 mutations were run again
  on the fixed code: 21 killed. Then the pre-add check of the nineteen
  Go files, the documents' checks, `internal/conformance` and the
  identifier scan, untracked files included: clean.
* **Step A1.3 complete, 2026-11-02,** for the owner's commit. Step 5,
  `Scanner`, `Doc`, `Renderer`, `Plain`, `Buffer` and
  `stream/streamtest`, is next.

### Step 5: `Scanner`, `Doc`, `Renderer`, `Plain`, `Buffer` and `stream/streamtest`

*2026-11-03.* The owner committed Step A1.3 (`8f9d052`) and said
"committed, proceed".

* **Deviation D16, 2026-11-03: a definition in another segment.**
  * **Found** reading 0009-MADR §4 against the `Renderer` before any
    code: "When a committed segment holds one, every cached segment is
    dropped and rendered again once", but a `Renderer` is
    `Render(src, width)`, a function of the segment it is given, so a
    segment rendered again alone cannot see a definition in another, and
    the rule renders again to the same frame. crush, which §4 follows,
    holds a link-reference definition open, so that no cut falls after it
    (0003-REPORT §3.1).
  * **Resolutions offered:** each segment's input carries the definitions
    committed so far, `Plain` printing none (recommended); §4's text as it
    is, the rule recorded as without effect; or no cut after a definition,
    as crush, which breaks §4's cost bound.
  * **Chosen by the owner,** 2026-11-03: "Append definitions to each
    input (Recommended)". The step's text and 0009-MADR §4 are annotated.

**What changed,** in the step's four parts, each passing its tests before
the next:

1. `stream/scanner.go`: `Scanner`, `Scan`, `Block` and `BlockKind`, from
   `internal/enum`, sixteen kinds (tokens "blank", "paragraph",
   "continuation", "heading", "setext", "break", "fence", "code",
   "fence-end", "indented", "html", "list", "quote", "table",
   "table-row", "definition"), one for each branch the `Printer` took.
   The `Printer`'s `fence`, `html`, `paragraph` and `lazy` moved into it;
   its held line and table rows, what it prints, stay its own. `line` now
   switches on the `Block`. Every `Printer` test passed unchanged, its
   goldens byte for byte, `TestReplyBudget` within its 257 allocations,
   and `FuzzPrinter` ran 5 minutes, 9,140,231 executions, clean.
   * `Block`'s fields are the agent's, as the step left them: `Cut`, a cut
     falls before this line, told on the first line after the blank ones,
     since only that line decides it, where the step's sketch says "before
     the next line"; `Split`, a relaxed cut may fall before this line;
     `Fence`, a fence line's opening line; `Label`, a definition's label.
     The open list item is its marker's kind, a bullet's character or an
     ordered marker's delimiter, the columns before it and those its
     content begins at (CommonMark §5.2, one to four spaces after the
     marker).
2. `stream/renderer.go`: `Renderer`, `RendererFunc` and `Plain`, the
   `Printer`'s drawing with the zero `Markup`, no definition printed
   (D16). `stream/document.go`: `Doc`, `NewDoc`, `Append`, `Write`,
   `Finish`, `Reset`, `Render`, `Version`, `Finished` and `Source`.
   `stream/options.go`: `Option` gained `applyDoc`, each option a pair of
   functions, a nil one ignored and so documented; `WithRelaxAfter`,
   `WithoutInlineHoldback` and `WithoutPartialLine`. The agent's readings:
   * a segment runs from the last cut to the next, its blank lines with it;
     a fence a relaxed cut splits is closed by its run and opened again by
     its opening line; each input carries the committed definitions after
     a blank line (D16), but the tail's inside a fence or an HTML block,
     where they would be text;
   * the frame cache is the last `(width, Version)`, the segments' renders
     are kept for the two widths last used, and their join is kept with
     them, so that a frame copies its prefix once; a `Render` returns a
     string, so a frame is one copy of the document, as A1.4's
     `RenderView` will not be;
   * the holdback reads emphasis and strong runs of `*` and `_` as
     CommonMark's flanking rule does in short, a run opening only before a
     character not a space and closing only after one, `_` inside a word
     neither; `~~` exactly; a run of backticks until a run of as many; and
     a link once its `](` has come, not before; inside a fence the
     partial line is code, and shown whole;
   * `Append` after `Finish` does nothing, and `Write` returns `ErrClosed`;
     `Version` only rises, `Reset` included.
3. `stream/buffer.go`: `Buffer`, `NewBuffer`, `Write`, `Close` and
   `Doc.Pull`. `stream` now imports `frame`, a core (A7 item 9). The
   `Buffer` holds the partial line until no complete line is queued, and
   gives it to the `Doc` then, which is what hides it while lines are
   queued; the `Doc` first kept a flag of its own for that too, which the
   mutations found could never be seen, and it was removed. A `nil`
   `Pacer` is `frame.Immediate()`.
4. `stream/streamtest/`: `CheckRenderer`, forty documents from fixed
   seeds, of every block kind the `Doc` cuts by, each at 40 and 80
   columns, whole, byte by byte and in random chunks of 1 to 32 bytes:
   chunking invariance, the stated differences only, with relaxed cuts
   off, a frame and the whole source's render the same but for their
   blank lines, and `Finish` exactness; reported by `t.Errorf`. Its
   generator is unexported.
* `internal/conformance`: `stream.BlockKind` in `enums`;
  `stream/streamtest`, Charm-free, in `cores`, and `.golangci.yml`'s
  `charmfree`, by D5's rule. `docs/architecture.md`: the graph, the two
  packages' rows, the cores' list and the tree; `docs/guides/defaults.md`:
  the step's three rows `built`; `stream/doc.go`, `stream/scan.go`: the
  package comment and the scanner's.

**Tests:** `TestScanner`, 30 documents of the step's cases;
`TestScannerFields`; `TestDocCuts`, 8; `TestDocHoldback`, 14;
`TestDocFinish`; `TestDocClean`, a sequence in each of three chunks and
none in a frame or the source; `TestDocRenderError`; `TestPlain`;
`FuzzDoc`, whose 5 minutes ran 14,821,507 executions clean;
`ExampleDoc`; inside the package, `TestRelaxedCuts`, a paragraph, a
quote, a fence, an ordered list, a table and an HTML block;
`TestDefinitions`; `TestCost`, 200 KB of paragraphs, of one paragraph, of
one fence, of one list and of one table, in 20-byte chunks; and
`TestWidthBack`; `TestBufferWriters`, `TestBufferFrames`,
`TestBufferClose` and `TestPullQueued`; in `streamtest`, `TestPlain`,
`Plain` at both width methods, `TestStatedDifferences`, a Renderer that
renders each block as itself, `TestCheckRendererCatches` and
`TestDocument`. All pass under `-race`.

* Test faults of the agent's, fixed: `TestDocCuts` and
  `TestDocRenderError` ended their sources on a partial line, after
  which no cut is decided, as §4 has it; `TestPlain` wanted a table
  without its outer pipes, which `layoutTable` draws; and `TestCost`'s
  table, re-rendered whole each frame as the stated exception, took
  6.7 s, so it renders a frame each 4 KB.
* **The benchmark,** 200 KB in 20-byte chunks at 80 columns, a frame
  after each, on the agent's macOS host: `BenchmarkDoc` 0.82 s, 1.13 GB
  and 519,010 allocations; `BenchmarkWholeRender`, the source so far
  rendered whole by `Plain` at each chunk, 41.5 s, 13.4 GB and
  137,405,289. The first `BenchmarkDoc`, which joined every segment each
  frame, was 1.09 s and 1.18 GB; the join is kept since.

**Mutations,** `python3 -m tools.verify mutate`, on the final code:

| Mutation | Its failure |
| :--- | :--- |
| 5.1 a cut inside a fence | `TestDocCuts`: "a fence's blank: \"[```\\nx]\\n\\n[y\\n```]\\n\\n[z]\"" |
| 5.2 the list's next-line check skipped | `TestDocCuts`: "a loose list: \"[- a]\\n\\n[- b]\\n\\n[c]\"" |
| 5.3 a reference definition keeping the cache | `TestDefinitions`: "the first segment rendered 0 times after the definition; want once" |
| 5.4 `Finish` keeping the segmented render | `TestDocFinish`: "after Finish: \"[a]\\n\\n[b\\n\\nc]\" …; want \"[a\\n\\nb\\n\\nc **open]\"" |
| 5.5 the relaxed cut never taken | `TestRelaxedCuts`: "a paragraph: 0 segments; want the paragraph split" |
| 5.6 a relaxed cut inside a table | `TestRelaxedCuts`: "a table: segments [\"a \| b\\n--- \| ---\\n01 \| x\\n\" …]; want the table whole" |
| 5.7 a list's relaxed cut losing its number | `TestRelaxedCuts`: "an ordered list: segment 1 \"item\\n5. item…\": want it to begin with its item's number" |
| 5.8 `Append` skipping the sanitizer | `TestDocClean`: "a frame \"a \\x1b[31mred\" or the source … holds ESC" |
| 5.9 `CheckRenderer` missing a cut-dependent Renderer | `TestCheckRendererCatches`: "CheckRenderer reported nothing for a Renderer that depends on where the source was cut" |
| 5.10a the `Scanner` forgetting the lazy continuation | `TestScanner`: "a lazy continuation: list paragraph; want list continuation" |
| 5.10b the `Printer`'s `Scanner` forgetting the paragraph's continuation | `TestParagraphContinues`: "\"Foo\\n    bar\\n===\\n\"; want \"Foo\\n\\x1b[1m    bar\\x1b[0m\\n\\x1b[1m===\\x1b[0m\\n\"" |
| 5.11 the holdback skipped | `TestDocHoldback`: "strong, open: \"Hello **wor\"; want \"Hello \"" |
| 5.12 the partial line given while lines are queued | `TestPullQueued`: "after Pull 0 the frame is \"a\\npart\" …; want \"a\"" |
| 5.13 `Pull` not finishing a closed `Buffer` | `TestBufferClose`: "after the last Pull: finished false …; want finished" |
| 5.14 a line asking for no frame | `TestBufferFrames`: "a completed line asked for no frame" |
| 5.15 one width cached | `TestWidthBack`: "a segment rendered 5 times over two widths; want at most 2" |
| 5.16 the definitions not carried | `TestDefinitions`: "segment inputs [\"see [x]\\n\\n\" …]; want each to carry the definition" |
| 5.17 a failed segment's text lost | `TestDocRenderError`: "\"[good]\\n\\n[end]\"; want the failed segment as Plain draws it" |
| 5.18 `Plain` printing definitions | `TestPlain`: "Plain(\"see [x]\\n\\n[x]: https://e.x\\n\", 0) = \"see [x]\\n\\n[x]: https://e.x\\n\"" |
| 5.19 a fence split not opened again | `TestRelaxedCuts`: "a fence: segment 1 \"x = 03\\n…\": want it to open the fence with its info string and close it" |

`mutate: 20 killed, 0 survived, 0 errors`. The step named ten. Its last,
"the `Printer` built on a `Scanner` that forgets the lazy continuation
(`TestParagraphContinues`)", names a test of a paragraph's continuation,
an indented line, not the lazy continuation of a list item; it first
survived, and is 5.10b, with 5.10a for the lazy one. 5.12 first targeted
the `Doc`'s own flag, and survived, since the flag could not be seen;
the flag was removed and the mutation moved to the `Buffer`. 5.13 was
first written as the `Finish` call removed, which left `closed` unused:
the mutant did not compile, and it was rewritten.

**The API,** `make apicheck`: "against v2.9.0, 0 incompatible
change(s)": the unexported method `Option` gained is compatible, as A6
item 4 expected; `stream` adds 38 names since `v2.9.0`, Steps A1.1's and
A8.1's among them, and `stream/streamtest` is new.

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 626 s (the second run; the first failed `modernize`) |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1260 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2673 s |

* The pre-add check's first run found four findings in the agent's code,
  each fixed: a parameter named `close` twice (revive
  redefines-builtin-id), now `closer`; `render` beside `Render` (revive
  confusing-naming), now `draw`; and the holdback's untagged switch
  (staticcheck QF1002). The first macOS `gates` then failed `modernize`,
  which the pre-add check does not run and the agent had not run
  itself, as earlier steps did: `go fix` wanted `strings.SplitSeq` for
  three loops over `strings.Split`, in `streamtest.go`,
  `streamtest_test.go` and `scanner_test.go`, the agent's; the remote
  runs, started on the same code, were stopped. `go fix ./stream/...`
  applied them, changing those three files alone, and `make modernize`
  was clean on the three platforms. The 20 mutations were run again on
  the fixed code: 20 killed. The pre-add check of the eighteen Go files,
  before `go fix`, and the `gates`' `release-check`, the pre-add check
  of every file, after it; the documents' checks, `internal/conformance`
  and the identifier scan, untracked files included: clean.
* **Step 5 complete, 2026-11-03,** for the owner's commit. Step 6,
  `theme.Theme.ID` and `DocPane`, is next.

### Step 6: `theme.Theme.ID` and `DocPane`

*2026-11-03.* The owner committed Step 5 (`a9fcfaf`) and said
"committed, proceed".

* **Deviation D17, 2026-11-03: the next frame while lines stay queued.**
  * **Found** reading the step against Step 5's code, before any of its
    own: `Doc.Pull` reports that lines are still queued "so that the
    caller asks for another frame" (`stream/buffer.go:97-115`), and the
    step gives `DocPane` no scheduler to ask. `Buffer.Write` asks for a
    frame only when a line completes, and `Close` asks once, so in the
    adaptive pacer's Smooth gear, one line a frame, the rest wait for the
    next write, and after `Close` for ever. In a scratch module against
    the tree, five lines written and closed, then pulled on each frame
    with `frame.Adaptive()`: one frame came, then none in 5 s; the source
    stayed "a\n", and the `Doc` never finished.
  * **Resolutions offered:** `Pull` asks the `Buffer`'s own Scheduler for
    the next frame while lines stay queued, which a line-mode host gets
    too (recommended); or `streamtea.WithScheduler`, the pane asking when
    `Pull` reports true: one more option, and a scheduler that may not be
    the `Buffer`'s. Doing nothing leaves a stream's last lines unshown.
  * **Chosen by the owner,** 2026-11-03: "Pull asks itself
    (Recommended)". Step 5's `Pull` changes in this step; the step's
    text, Step 5's item 3 and 0009-MADR A7 item 9 are annotated.

**What changed,** in the step's three parts, each passing its tests before
the next:

1. `theme/theme.go`: `Theme.ID`, read from an unexported field that `New`
   stamps from a `sync/atomic` counter, the first ID 1; `Theme`'s and
   `ID`'s documentation say that a copy keeps its ID, that a changed theme
   is built with `New`, and that a literal's ID is 0, which a cache never
   serves from.
2. `stream/buffer.go` (D17): `Pull` asks `b`'s Scheduler for the next
   frame while lines stay queued, and its documentation says so.
3. `stream/streamtea`, new: `DocPane`, `NewDocPane`, `Update`, `View`,
   `Changed`, `Title`, `KeyContexts` and `ConsumesCommand`; `Option`,
   `WithTitle`, `WithoutFollow`, `WithBuffer` and `WithPacer`; the six
   command IDs, `DocPaneScope` and `DefaultRules`, built as `palette`'s
   are. The agent's readings, where the step leaves the shape open:
   * **The marks** have a column of their own, the pane's last, whenever
     the pane is two cells wide or more, so that a row keeps its wrap as
     the document outgrows the pane: the theme's `ScrollUp` on the first
     row while rows are above, its `ScrollDown` on the last while rows are
     below, the latter on a pane one row high, each in the theme's `Muted`
     style, their shapes carrying the meaning.
   * **What the theme keys.** A row wider than its column, which a
     renderer may draw (a long word, a table, a code line), is cut with
     the workspace's width method and ends with the theme's `Ellipsis`, as
     `glyph.Set` says the field is for. So the rows the pane keeps depend
     on the theme, and the step's key by theme ID, its literal theme never
     served, and its two mutations can be seen; drawn only at each `View`,
     the marks alone would not have made the key observable.
   * **The width method** fits each row, cut and padded, so a row is
     exactly its column under the workspace's method, whatever the `Doc`
     measured with.
   * **Scrolling:** a wheel notch is a row, as `palette`'s picker scrolls;
     a page is the pane's height; home and end are the first and the last
     page. `WithoutFollow` starts at the document's start and never
     follows, end included.
   * **`Changed`** compares what the last `View` drew with now: the `Doc`'s
     `Version`, the size from `SizeMsg`, the first row and following. A
     scroll that moves nothing, such as down at the end, an unbound key
     and a frame with no `Buffer` leave it false.
   * **`Title`** is empty without `WithTitle`. The workspace then draws an
     empty title, not the pane's ID, since `workspace/render.go:290` takes
     a `Titled` pane's title as it is, where `commands.go:343` and
     `palette/panes.go:56` fall back on an empty one: a difference in the
     workspace before this step, recorded and not changed here.

**Tests:** `TestID` and `TestIDConcurrent`, 800 `New`s on 8 goroutines
under `-race`, every ID different and none 0; `TestPullAsksNextFrame`;
and in `streamtea`, in a real workspace whose keymap is
`workspace.DefaultRules()` and `DefaultRules()`: `TestFollow`, following,
staying put when scrolled up as the document grows, following again at
the end, and `WithoutFollow`; `TestChanged`, false after an unrelated
message, a scroll down at the end, an unbound key and a frame with no
`Buffer`, true after a new version, a scroll and a size, false after each
draw; `TestPull`, a frame pulling one line by the adaptive pacer and both
by `Immediate`, then draining a closed `Buffer` to `Finish` on the frames
`Pull` asks for; `TestWheel`, a `tea.MouseWheelMsg` and an
`inputfiltertea.CoalescedWheelMsg` of four notches, both through the
workspace; `TestCommands`, each default key and a user's `k` bound to
`stream.scroll.up`, `ConsumesCommand` and `KeyContexts`; `TestTitle`;
`TestTheme`, the Unicode and ASCII ellipses, and two literal themes, ID 0
both, each drawing its own; `TestMethod`, an emoji with a skin tone padded
again when the terminal reports mode 2027; and `TestGolden`, the pane in a
bordered workspace by `Plain` across the matrix at 30 and 60 columns:
mid-stream with a partial line held before an unclosed `**`, inside a
growing table, finished, and scrolled up with both marks. The 32 goldens
were read before this record. All pass under `-race`.

* Test faults of the agent's, fixed: the first documents were paragraphs,
  which `Plain` draws a blank row apart, so the tests draw lines with an
  echoing renderer; `TestTheme` wanted "..." for the ASCII set's
  ellipsis, which is "~"; and `TestWheel`'s first notch came before any
  frame was drawn, when the workspace has no region for the mouse to hit,
  so the harness draws after building, as a program does.
* **The benchmark,** `BenchmarkView`, 24 rows of 80 columns once the rows
  are kept, five runs each on the agent's macOS host: at 100 rows a
  median of 2,484 ns, at 10,000 rows 2,738 ns, 1.10×, within the step's
  2×; 6,064 bytes and 7 allocations a `View` at both.

**Mutations,** `python3 -m tools.verify mutate`, on the final code:

| Mutation | Its failure |
| :--- | :--- |
| 6.1 `Changed` always true | `TestChanged`: "after an unrelated message: Changed true; want false" |
| 6.2 following ignoring the user's scroll | `TestFollow`: "scrolled up a row: [\"line 8 … ^\" \"line 9\" \"line 10\"]" |
| 6.3 the cache keyed without the theme ID | `TestTheme`: "a new theme, the ASCII one: \"xxxxxxxx…\"" |
| 6.4 an ID of 0 served from the cache | `TestTheme`: "a literal theme with Ellipsis \"?\": \"xxxxxxxx!\"" |
| 6.5 `ConsumesCommand` false for a scroll command | `TestCommands`: "after up: … want lines 16 to 19" and "ConsumesCommand(stream.scroll.up) false" |
| 6.6 the marks not from the glyphs | `TestGolden`: "scrolled color.utf8.30: line 2 differs", the four UTF-8 cases |
| 6.7 `Pull` not asking for the next frame (D17) | `TestPullAsksNextFrame`: "no frame in 5 s after 1: source \"a\\n\", not finished" |
| 6.8 `New` stamping one ID | `TestID`: "two News: IDs 1 and 1; want two different, neither 0" |
| 6.9 the rows fitted without the workspace's method | `TestMethod`: "under 1: … want 7 spaces of padding and the marks' column" |
| 6.10 a mouse message's `WheelDelta` ignored | `TestWheel`: "four notches up, coalesced: [\"line 7 … ^\" …]" |
| 6.11 a frame not pulling the `Buffer` | `TestPull`: "the adaptive pacer: after one frame the source is \"\"; want \"a\\n\"" |
| 6.12 a scroll at the end changing the view | `TestChanged`: "after a scroll down at the end: Changed true; want false" |

`mutate: 12 killed, 0 survived, 0 errors`. The step named six; the
others are D17's and the agent's readings'. Each was killed by its named
test's assertion, none by a build failure.

**The API,** `make apicheck`: "against v2.9.0, 0 incompatible
change(s)"; with compatible changes listed, `theme` adds
`Theme.ID` alone, as the step says, and `stream/streamtea` is new.
`stream/streamtea` is not a core: it imports `workspace` and Bubble Tea.

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 717 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1233 s |
| `remote windows` | 20 passed, 1 failed (`script-test-check_ci_filters`, D18), 1 skipped (`gofmt`) in 2809 s |

* The pre-add check of the eleven Go files and `make modernize` were clean
  at their first run, as were `internal/conformance`, the documents'
  checks, after this record too, and the identifier scan, untracked files
  included.

* **Deviation D18, 2026-11-03: D30's Windows failure, again, named.**
  * **Found** in the Windows run of every gate:
    `script-test-check_ci_filters`, "check_ci_filters_test: 41 passed, 1
    failed", the case quoted, as 0018-PLAN Step 11 made it: "FAIL the
    recipe drops the guide check: want ['readers: make docs-check does
    not run --check-guide, the reader in scripts/go-examples.py'], got
    []". It is
    [0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
    D30's failure, whose case was not known then.
  * **Rerun alone** through `remote windows`, from the same tree: passed,
    passed, then failed another case, "FAIL a pattern with no file: want
    ["tracked: CI ignores 'nothing/**/*.md', which matches no tracked
    file"], got []", each run 120 to 134 s. It passes on macOS and
    Linux.
  * **What is known:** in both cases the case's own edit of its copy
    landed, since `Tree.edit` asserts its count and a miss reads "the case
    stopped"; so `check_ci_filters.py`, run on a broken copy, printed no
    problem line, and the test's `run` keeps only those lines, not the
    exit status or the rest of the output that would say why. This step
    changed no script, workflow or `Makefile`.
  * **Resolutions offered:** record it here, close the step, and propose
    an amendment to
    [0018-MADR-python-verification-tools.md](0018-MADR-python-verification-tools.md)
    whose step makes the test report the checker's exit status and output
    when no problem line comes, then finds the cause (recommended); or
    chase it on the host now, with no bound on the time, and in `scripts/`,
    outside this PLAN.
  * **Chosen by the owner,** 2026-11-03: "Record, propose 0018 fix
    (Recommended)". The amendment is written after this step's commit, as
    its own. No file is added to this step's scope.
  * *Corrected 2026-11-03,* after the step's commit (`4413f94`): the
    agent named the wrong record. 0018-MADR A6 item 2 puts a fix to
    `scripts/check_ci_filters_test.py` or the script it tests in an
    amendment to
    [0022-MADR-align-docs-ci-and-disclosure-gates.md](0022-MADR-align-docs-ci-and-disclosure-gates.md).
    Asked again, the owner chose "0022, as 0018 A6 says (Recommended)".
    ~~an amendment to 0018-MADR~~ 0022-MADR A1 and
    [0022-PLAN-align-docs-ci-and-disclosure-gates.md](0022-PLAN-align-docs-ci-and-disclosure-gates.md)
    Phase 4 are proposed.
* **Step 6 complete, 2026-11-03,** for the owner's commit. Step A1.4,
  `RenderView`, the dialect, `Highlighter`, `FitTable` and `Bounded`, is
  next.

### Step A1.4: `RenderView`, the dialect, `Highlighter`, `FitTable` and `Bounded`

*2026-11-03.* The owner committed Step 6 (`4413f94`) and 0022-MADR A1,
proposed (`b25f461`), pushed `main`, and said "pushed, proceed".

* **Deviation D19, 2026-11-03: the step's API.**
  * **Found** reading the step against A1 before any code: the step gives
    "The API, A1's, with A4's names", and A1 gives shapes, not
    signatures. Six parts have none: how a `Renderer` reports
    `RenderView`'s rows, links and code, when `Render` returns a string;
    how a `Doc` learns the theme revision its `Highlighter`'s caches drop
    on, when A6 Q9 keeps `theme` out of `stream`; the math holdback's
    option; `Bounded`'s type and methods; `FitTable`'s signature and its
    borders, against the pipes `layoutTable` draws; and `stream`'s
    catalogue, and what "`DocPane` passes its catalogue on" passes it to.
  * **Resolutions offered:** the whole API written into the records
    first, as D13 did for Step A1.3 (recommended); the step in two, the
    parts whose shape is plain first; or each choice asked at once, and
    all built in one step.
  * **Chosen by the owner,** 2026-11-03: "Write the whole API first
    (Recommended)". The step's text now carries it, for the owner's review
    and commit; no code is written until the owner approves it. 0009-MADR
    A1 §4 is annotated, the glossary reserves the new names, and
    `docs/guides/defaults.md` gains the defaults, `planned`.
    `python3 -m tools.verify names` over `RenderView`, `ViewLink`,
    `ViewCode`, `ViewTable`, `ViewRenderer`, `Highlighter`,
    `HighlighterFunc`, `Bounded` and `TableOptions`: clean.
* **Step A1.4's API written, 2026-11-03,** a change to the records alone,
  for the owner's review and commit.

*2026-11-03.* The owner committed the API (D19) and said "committed,
proceed". Part 1, the dialect's tests and `WithMath`, passed before
part 2.

* **Deviation D20, 2026-11-03: a line-mode table is not wrapped.**
  * **Found** building part 2, `FitTable`, with the `Printer` fitting its
    tables at its width as D19 says: two tests of the tree failed.
    `stream`'s `TestWrap`, "a table is laid out and not wrapped"
    (`stream/printer_test.go:175`), one of gobble's wrapping cases; and
    `transcript`'s `TestReprintGolden` at 60 columns, whose reply holds a
    table "long enough to pass sixty columns on its own row"
    (`transcript/testdata/reply.txt:16`), drawn now as records, with
    `TestReprintBudget` at 55,478 allocations against its 49,696. Both
    hold [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)'s Context: gobble wraps prose, and "fences, indented code,
    table rows and headings" stay "unwrapped". D19 said the change
    altered the line mode's output, and did not name the decision it
    overturned.
  * **Resolutions offered:** `Plain` alone fits its tables, since it
    draws into a pane that clips a row, and the `Printer` keeps 0023's,
    since a terminal wraps a long row in its scrollback (recommended);
    both fit, 0023 amended, the two tests and the budget changed; or the
    `Printer` fits under a new opt-in option.
  * **Chosen by the owner,** 2026-11-03: "Plain fits; Printer as 0023
    (Recommended)". The `Printer` lays its tables out by `FitTable` with
    no width; `Plain`'s `Printer` passes its width. No test of the tree
    changes. D19's text, 0009-MADR A1 §4's annotation and the defaults
    row are annotated.

**What changed,** in six parts, each passing its tests before the next:

1. **The dialect and `WithMath`.** `holdback` holds an unclosed `\(` or
   `$$` under `WithMath`, through `mathSpan`. The dialect needed no code:
   `TestDialect` holds `~~x~~` struck and `~**10%**` and `~x~` not.
2. **`FitTable` and `TableOptions`** (`stream/table.go`), built from
   `layoutTable`, which they replace: a table that fits is drawn as
   `layoutTable` drew it, byte for byte, so no golden of the tree moved;
   past the width, its cells wrap, then key/value records, stacked
   records, or its source rows as they came; a box from a
   `glyph.Border`. `Plain` fits its tables at its width, and the `Printer`
   does not (D20).
3. **`Bounded`** (`stream/bounded.go`), **`Messages` and `MsgOmitted`**
   (`stream/messages.go`); `stream.Option` gains `applyBounded`, and
   `WithRules`, `WithMethod`, `WithMaxBytes`, `WithHeadLines`,
   `WithTailLines`, `WithMaxLineBytes`, `WithGlyphs` and `WithCatalog`
   apply to it.
4. **`Highlighter`, `HighlighterFunc`, `WithHighlighter`,
   `WithHighlightLimits` and `Doc.SetRevision`** (`stream/highlight.go`,
   `stream/document.go`): the `Printer` highlights each fenced line as it
   prints it; a `Doc` with `Plain` highlights through `Plain`'s `Printer`,
   scanning each render input first for the blocks past a limit, and
   keeps its tail's lines.
5. **`RenderView`, `ViewLink`, `ViewCode`, `ViewTable`, `ViewRenderer`
   and `Doc.View`** (`stream/view.go`): `Plain` is a `ViewRenderer`, its
   `Printer` recording each row's source line, the links of each prose
   line and heading, each code block and each table as it prints them;
   the `Doc` keeps a segment's view per width, as its render, and takes
   the tail's anew.
6. **`DocPane`** gives its `Doc` its theme's ID as the revision at each
   draw, before it records what it drew, and under a theme of ID 0 a new
   revision each draw, from `1<<63` up, where no `theme.New` ID reaches.

The agent's readings, where the API left the shape open:

* **Highlighting** is of fenced code alone: indented code has no info
  string, so no language. A `Highlighter` that returns a nil state is
  given the block's every line so far with each next line, its cost
  bounded by the line limit. A `Doc`'s tail keeps a fence's lines by its
  opening line's offset in the document, so a cut moves no key; a fence a
  cut commits is dropped from it.
* **A `Printer`'s block past a limit is plain from the line that passes
  it,** where D19 says "a block over any limit shows plain": a `Printer`
  prints a line once, before the block's end is known. A `Doc`'s block is
  plain whole, as D19 says, `overLimits` scanning its render input first.
* **A link is reported on a prose line or a heading,** not in a table's
  rows, whose cells `ViewTable` gives. A table's rows take its first
  line's offset, since `FitTable`'s rows are not its source's one for
  one. A reference link in a segment is reported once its definition is
  committed, which is when the segment is rendered with it (D16).
* **`Bounded` closes a head's SGR** by ending its last head line with
  `CSI m` when any head line holds a sequence; a line is cut at a
  character's start, so its cleaning makes no U+FFFD.
* **`FitTable` pads a cell by at least nothing:** a cell wider than its
  column, which correct widths never make, gives no negative
  `strings.Repeat`, so that a wrong width is a wrong row and not a panic
  (mutations A1.4.4 and A1.4.12 first panicked).

**Tests:** `TestDocMath`, `TestDialect`; `TestFitTableFits`,
`TestFitTableShrinks`, `TestFitTableFallbacks`, `TestFitTableWide`,
`TestPlainFitsTables` and `TestFitTableGolden`, the box and the Markdown
form across the matrix at 30 and 60 columns, the 8 goldens read before
this record; `TestBoundedWithin`, `TestBoundedPast`, `TestBoundedLineCap`,
`TestBoundedClean` and `TestBoundedCatalog`, `Messages()` through
`msgcat.Check`; `TestHighlightGrowing`, `TestHighlightCaches`,
`TestHighlightBudget`, `TestHighlightLimits` and `TestHighlightPrinter`;
`TestViewIsRender`, `TestViewChunking`, `TestViewTables`,
`TestViewWrappedLink`, `TestViewSplitFence` and `TestViewOtherRenderer`;
`FuzzDoc` gains the view's chunking invariance and its rows being
`Render`'s; in `streamtea`, `TestRevision`. All pass under `-race`, and
every test of the tree's other packages as before, the line mode's
goldens and allocation budgets among them.

* Faults of the agent's, found by the tests and fixed: `FitTable`'s
  source fallback rebuilt the rows from their cells, losing the
  delimiter's colons; `Plain`'s view dropped its last line break, so its
  rows did not join to `Render`'s after `Finish`; an indented block's end
  left out its last line break. Test faults: `TestFitTableFits` wanted
  the header left-aligned in a right-aligned column; the fallbacks'
  first widths fitted the table's words; `TestHighlightBudget` and
  `TestHighlightCaches` drew a second frame at the same version, which
  `Render` gives back kept; `TestViewTables`' definition was the
  document's last line, not yet committed.
* **Fuzzing,** 5 minutes each: `FuzzDoc`, 19,259,376 executions, and
  `FuzzPrinter`, 25,248,087: clean.
* **The benchmark,** `BenchmarkDoc`, 200 KB in 20-byte chunks, no
  highlighter: 1,125,834,112 bytes and 518,578 allocations an operation,
  Step 5's 1.13 GB and 519,010; 0.17 to 0.21 s an operation in three runs
  and three of one iteration each. Step 5 recorded 0.82 s, which the same
  benchmark does not reproduce on the host now; the agent cannot say how
  it was taken, and claims no speed-up.

**Mutations,** `python3 -m tools.verify mutate`, on the final code:

| Mutation | Its failure |
| :--- | :--- |
| A1.4.1 `RenderView` not cut back on a tail re-render | `TestViewWrappedLink`: "rows [\"Read [the whole\" … \"\" \"Read [the whole\" …]; want this frame's alone" |
| A1.4.2 a single `~` striking | `TestDialect`: "\"~x~\\n\": \"\\x1b[2m~x~\\x1b[0m\\n\"; want \"~x~\\n\"" |
| A1.4.3 the fence re-highlighted whole each line | `TestHighlightGrowing`: "\"line 27\" highlighted 23 times; want once" |
| A1.4.4 `FitTable` padding by byte | `TestFitTableWide`: "rows of 14 and 16 cells" |
| A1.4.5 `Bounded` counting rows instead of lines | `TestBoundedPast`: "5 lines of 200 bytes, one kept at each end: 9 omitted; want 3 lines" |
| A1.4.6 the omitted line not from the catalogue | `TestBoundedCatalog`: "[\"line 1\" \"(3 lines not shown)\" \"line 5\"]; want the catalogue's line" |
| A1.4.7 the math holdback ignored | `TestDocMath`: "an open \\(: \"the area \\\\(a^2\"; want \"the area \"" |
| A1.4.8 a link's ID by its row | `TestViewTables`: "link [a link]: \"\", false; want \"https://e.x/t\" at its offset" |
| A1.4.9 `SetRevision` keeping the segments | `TestHighlightCaches`: "after a new revision: 2 highlighted, version 4 after 3; want 4, raised" |
| A1.4.10 the closed-fence cache never dropped | `TestHighlightBudget`: "past the budget, a second frame highlighted nothing" |
| A1.4.11a a `Doc`'s whole-block limits ignored | `TestHighlightLimits`: "bytes, a Doc: … 1 highlighted; want the block plain" |
| A1.4.11b a line's highlight limits ignored | `TestHighlightLimits`: "bytes, a Printer: … want the first line highlighted and the second plain" |
| A1.4.12 `FitTable` breaking a word | `TestFitTableShrinks`: "a row of 70 cells at 60" |
| A1.4.13 a fallback skipped | `TestFitTableFallbacks`: "at 7: [\"k: alpha\" …]; want [\"k\" \"  alpha\" …]" |
| A1.4.14a `Bounded`'s floor `Text` | `TestBoundedClean`: "StatusLine keeps colour: [\"red \"]" |
| A1.4.14b `Bounded` with no floor | `TestBoundedClean`: "the floor cuts a link StatusLine refuses" |
| A1.4.15 a line's cap ignored | `TestBoundedLineCap`: "[\"abcdefghijkl\" …]; want the long lines cut at 8 bytes" |
| A1.4.16 the pane giving no revision | `TestRevision`: "a new theme: the segment drawn 0 times more; want once" |
| A1.4.17 a literal theme's revision reused | `TestRevision`: "a literal theme, two draws: the segment drawn 2 times more in all; want 3" |
| A1.4.18 `Plain` not fitting its tables (D20) | `TestPlainFitsTables`: "Plain at 20: … want records" |
| A1.4.19 a split fence's halves not joined | `TestViewSplitFence`: "code [{Info:sh Start:6 End:70 …} {Info:sh Start:70 …} …]; want one block" |

`mutate: 21 killed, 0 survived, 0 errors`, each by its test's own
assertion: no build failure and no panic among them. The step named six,
D19 eight more; A1.4.7, A1.4.11b, A1.4.16 to A1.4.19 are the readings'
and D20's. Three needed work: A1.4.1 first survived, since
`TestViewWrappedLink` checked a part a row and a carried tail's parts
fall on other rows; the test now holds the frame's rows to `Render`'s and
the parts to consecutive rows. A1.4.13, first `if true`, did not compile,
`pairs` then unused, so it counted for nothing; it is `pairs || true`.
A1.4.4 and A1.4.12 first killed by a panic, a negative `strings.Repeat`;
with the pad's clamp, by the tests' assertions.

**Lint:** the pre-add check's first run found three findings in the
agent's code, each fixed: `holdback`'s cognitive complexity, 62 and then
51 against 50, its math and its code spans now `mathSpan` and
`codeSpanEnd`; and `plain`'s `render` and `renderView`, differing from
`Render` and `RenderView` by case alone (revive confusing-naming), now
`drawWith` and `viewWith`. `make modernize` was clean, run before the
gates this time.

**The API,** `make apicheck`: "against v2.9.0, 0 incompatible
change(s)": `stream.Option`'s third unexported method is compatible, as
its second was; `stream` adds 22 names in this step, 60 since `v2.9.0`,
D19's alone.

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 657 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1066 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2709 s, `script-test-check_ci_filters` among them (D18's flake did not recur) |

* The pre-add check of the sixteen Go files, after its findings above,
  `make modernize`, `internal/conformance`, the documents' checks and the
  identifier scan, untracked files included: clean.
* **Step A1.4 complete, 2026-11-03,** for the owner's commit. Step 8, the
  examples, the guide and the root's release, is next.

### Step 8: the examples, the guide, and the root's release

*2026-11-03.* The owner committed Step A1.4 (`e0cdaa9`) and said
"committed proceed".

**The examples,** each with its `// guide:` region:

* `ExampleDoc` (`stream/document_test.go`, region `doc`) and
  `ExampleFilter` (`inputfilter/inputfiltertea/inputfiltertea_test.go`,
  region `filter`), as they were, with their regions;
* `ExampleScheduler` (`frame/example_test.go`, region `scheduler`), new: a
  line-mode host's loop with no Bubble Tea, a reply written into a
  `Buffer` from a goroutine, each frame pulled with `frame.Adaptive`;
* `ExampleDocPane` (`stream/streamtea/example_test.go`, region
  `docpane`), new: a model whose `Update` gives the workspace each
  message and re-arms `frametea.Wait` on each `frame.DueMsg`, the one
  line, and an answer streamed from a goroutine through a `Buffer` into a
  `DocPane`; the example runs Wait and gives its message to `Update` as
  Bubble Tea's loop does. Each passed three runs; the first build failed
  on the agent's function `stream`, the package's name, now `answer`.

**The guide,** `docs/guides/streaming-content.md`: feeding a `Doc` from
the event loop or from goroutines; the `Scheduler`, and why `Wait` is
re-armed; choosing a pacer; showing a `Doc` in a pane, its commands and
keys; the filter, `Attach` and paste bursts; what the stream cannot send,
and trusted links through `termsvc.Linker`; what may differ before
`Finish`; `RenderView`; highlighting; tables; `Interpret` and `Bounded`;
the line mode's use of the cores; and the defaults. Its four excerpts are
the regions, as `scripts/go-examples.py --check-guide` reads them:
"examples: 19 excerpt(s) checked", clean. markdownlint's first run found a
code span ending in a space, the agent's, reworded.

**The rows:** `docs/README.md`'s "I want to…", six rows into the guide,
and the 0009 PLAN's status, which read "in-progress (Step 1)" since Step
1, now Step 8's; `README.md`'s feature list and its "I want to…" row;
`docs/architecture.md`'s guides line; and "Migrating to `v2.10.0`" in
`docs/guides/migrating.md`, the release's notes, as the releases before
it, `v2.10.0` the name the agent proposes for the owner's tag. It names
the two changes a `v2.9.0` program can see: `theme.Theme`'s unexported
field, so an unnamed-field literal no longer compiles, and the workspace
passing on another package's mouse message that has `WithMouse` (Step
4).

**The smoke test, rehearsed** before the tag
(`scratchpad`'s `smoke210.py local`, the tree by a `replace`, in a scratch
module outside the repository, `GOWORK` empty): `./line`, a reply paced by
`frame` into a `Doc` with no Bubble Tea, 7 frames; its closure for Linux,
macOS and Windows the module itself, the root, `x/ansi`, `displaywidth`,
`uax29`, `go-colorful` and `go-runewidth`: no Charm module but `x/ansi`.
`./tui`, a document streamed through a `Buffer` into a `DocPane` of four
rows. The control, the same programs against `v2.9.0` from the proxy:
`go mod tidy` and `go build` exit 1, "module …@latest found (v2.9.0), but
does not contain package github.com/maccavelli/go-tui-lib/v2/frame".

* **Deviation D21, 2026-11-03: a finished `Doc`'s last row in a pane.**
  * **Found** in the rehearsal: the four-row pane drew lines 9 to 11 of
    twelve and an empty row, where it drew the last four while the
    document streamed. `Doc.Render` after `Finish` is the renderer's
    output byte for byte, which ends with a line break, and `DocPane`'s
    `rows()` split it at each, so the break began an empty last row; the
    text moved up a row when the stream ended. It is Step 6's, committed
    (`4413f94`).
  * **Resolutions offered:** fix it in this step, with a test and a
    mutation (recommended); or release `v2.10.0` with it recorded, and fix
    it in a `v2.10.1` step.
  * **Chosen by the owner,** 2026-11-03: "Fix it in Step 8
    (Recommended)". `rows()` drops the one line break that ends the frame
    before it splits, so a document's own blank lines stay rows. The
    step's text is annotated; its files gain `stream/streamtea/pane.go`
    and `pane_test.go`.
  * **Test:** `TestFinishedTail`, the last four lines on the pane's four
    rows while the document streams and after `Finish`. **Mutation:** 8.1,
    the break kept: `TestFinishedTail`, "finished: [\"line 10 … ^\"
    \"line 11\" \"line 12\" \"\"]; while streaming [\"line 9 … ^\" …]; want
    lines 9 to 12 both". `mutate: 1 killed`. The rehearsal again: the pane
    drew lines 8 to 11, and the smoke test passed.

**Checks:** the pre-add check of the five Go files; `make lint` ("0
issues", three systems) and `make vuln` ("No vulnerabilities found"), as
`docs/guides/releasing.md` asks before a release;
`scripts/check_ci_filters.py`, "0 problem(s)"; the documents' checks and
the identifier scan: clean.

**Gates,** with the step's files:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped in 1076 s |
| `remote linux` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 1297 s |
| `remote windows` | 21 passed, 0 failed, 1 skipped (`gofmt`) in 2724 s |

**Not done yet, and the release:** the owner's commit of the above; the
disclosure guard over the outgoing commits; the owner's push of `main`,
and CI and Docs green on it; the owner's tag, `v2.10.0` as the agent
proposes or whichever the owner names, and CI green on the tag; and the
smoke test from the proxy, `smoke210.py proxy`. The step stays open for
them, and the PLAN `in-progress`.
