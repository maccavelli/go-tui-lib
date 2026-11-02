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
* **The adapter** (`inputfilter/inputfiltertea`):

  ```go
  func NewFilter(o ...Option) *Filter // WithInterval(16 ms), WithoutWheelCoalescing,
  	// WithoutMotionThrottle, WithPasteBurst(inputfilter.PasteBurstConfig), WithWheelProfile(n)
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
  notch, 1 by default.
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
3. **`Buffer` and `Doc.Pull`,** in `stream` (A7 item 9):

   ```go
   func NewBuffer(s *frame.Scheduler) *Buffer           // safe for concurrent use
   func (b *Buffer) Write(p []byte) (int, error)          // whole lines timestamped; a line requests a frame
   func (b *Buffer) Close() error                        // the stream's end: Pull then calls Finish
   func (d *Doc) Pull(b *Buffer, p frame.Pacer, now time.Time) bool // true while more is queued
   ```

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
