---
status: proposed
date: 2026-10-02
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
whose Phases 2–6 run after `v0.1.3`. Nothing in this PLAN's steps
changes. Step 7 still waits for 0010-PLAN's tooling phases. This PLAN
stays proposed until the owner approves its execution.

## Goal

Ship `safetext`, `frame`, `inputfilter`, `stream` and `stream/glamourmd`,
so that pi-go can stream an agent's Markdown into a workspace pane:

* at O(delta) cost per chunk;
* with an exact final frame;
* with nothing from the stream reaching the terminal as a control sequence;
* with wheel floods and typed pastes coalesced before they queue.

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag the release.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0009-*`, `docs/README.md` | accept the records |
| 2 | `safetext/`; `glyph/` (`Set.Control`) | streaming sanitizer, URL check |
| 3 | `frame/` | scheduler, generic queue, drain policies |
| 4 | `inputfilter/`; `workspace/render.go` (`local`) | wheel and motion coalescing, paste bursts, mouse extension point |
| 5 | `stream/` (`Doc`, `Buffer`, `Plain`, the cut scanner) | the stable-prefix engine |
| 6 | `stream/` (`Pane`) | a workspace pane over a `Doc` |
| 7 | `stream/glamourmd/` (its own `go.mod` and `go.sum`); `go.work` | the glamour adapter, as a nested module (MADR A2) |
| 8 | `stream/example_test.go`, `docs/guides/streaming-content.md`, `docs/` | the example, the guide, close-out |
| A1.1 | `safetext/` | `Command` and `StatusLine` presets, `Policy` fields, `Interpret` (A1) |
| A1.2 | `frame/` | `Coalescer`, `Demand`, `Writer`, `Upgrader` (A1) |
| A1.3 | `inputfilter/` | the stage chain, typeahead, paste normalisation, wheel profile (A1) |
| A1.4 | `stream/` | `RenderView`, the dialect rules, `Highlighter`, `FitTable`, `Bounded` (A1) |

~~`go.mod` gains `charm.land/glamour/v2` v2.0.1, or its newest release on the
day, in Step 7 and no other step.~~ *2026-10-02 (MADR A2):* the root
`go.mod` never gains glamour. `stream/glamourmd/go.mod` requires
`charm.land/glamour/v2` v2.0.1, or its newest release on the day, and a
published root version, in Step 7. Steps 2–6 and A1.1–A1.4 add no module.
Q6–Q8 of A1 kept `Interpret`, `Writer`, `Upgrader` and `FitTable` at the
paths above.

### Out of scope

* A virtualized transcript of many `Doc`s, watermark pruning, and inline
  scrollback with OSC 133. Each is a later record (MADR option D and
  0003-REPORT §7). The owner chose an `inline` record on 2026-10-02
  (0003-REPORT §11.2).
* Syntax highlighting other than glamour's own chroma use. A1.4 adds the
  `Highlighter` interface only; a lexer adapter needs its own dependency
  record.
* Re-sending the mouse modes on focus-in, which belongs to the planned
  `termmode` record (MADR A1, "What does not change").
* Images, and the terminal services of
  [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md).
* Any change in pi-go.
* `git push` and tags, which are the owner's.

## Rules for every step

1. **Order.** Each step's package compiles, passes its tests and passes the
   pre-add gate before the next step starts.
2. **Mutation proofs.** Each step names mutations of its key invariants.
   Each is applied to a scratch copy and must make a named test fail. A
   mutation that survives, or does not compile, is replaced and recorded.
3. **Checks per step:**
   * `make pre-add-check FILES=…`;
   * `make lint`;
   * `go test -race -count=1 ./...`;
   * `LC_ALL=C go test ./...`;
   * the Windows test host on a scratch copy;
   * `go mod tidy -diff`.
4. **Conventions.** Every package follows 0001-MADR §6. In particular,
   nothing writes to `os.Stdout` or `os.Stderr`, nothing sets the alternate
   screen or installs a signal handler, and every glyph comes from `glyph`.
5. **Commit.** One commit per step, made by the owner. At the end of each
   step the agent stops with the pre-add checks passed and the step's
   evidence in the execution record, and stages and commits nothing. The
   owner commits with `git commit --no-edit` and pushes. (2026-10-02: the
   org rules forbid agent commits to `main`, and the owner chose this over
   a `feature/` branch.)

## Implementation Steps

### Step 1: records

The owner accepts the MADR, answering Q1–Q5. Record the answers, set the
MADR `accepted` and this PLAN `in-progress`, and update `docs/README.md`.
The records this one builds on
([0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
for `ansi.Method`, and the records before it in the order) are complete
first, or this step records which parts wait for them.

*2026-10-02:* the answers are recorded and the MADR is accepted. This PLAN
moves to `in-progress` when the owner approves its execution, and
`docs/README.md` is updated then.

### Step 2: `safetext`

* The API of MADR §2: `Policy`, `Text`, `Strict`, `Line`, `Clean`,
  `Filter`, `NewFilter`, `SafeURL`, `URLs`. `glyph.Set` gains `Control`
  (`␛`, ASCII `^`). The glyph package's reflection tests cover the new field
  with no change.
* **Tests:**
  * a table with one row per sequence class: CSI, OSC ended by BEL, by ST
    and by U+009C, DCS, SOS, PM, APC, nF escapes, a lone `ESC`, CAN and SUB
    aborts, `ESC` inside a sequence, each C1 code point, each C0 control,
    DEL, `\r\n`, a lone `\r`, invalid UTF-8, and bidi controls under `Text`
    and `Strict`;
  * codex's injection string, `"\x1b[2J\x1b]52;c;Y2xpcA==\x07\u009dhidden\u009c\r\n\t"`,
    cleans to `"\n\t"` under `Text`;
  * an OSC with no terminator hides at most `MaxString` bytes, and the text
    after it survives;
  * `Visible` shows exactly one `Control` glyph per removed sequence, from
    the ASCII set under `glyph.ASCII()`;
  * `SafeURL` refuses `javascript:`, a URL holding ESC, BEL or a space, and
    one holding `ESC \`;
  * `URLs` returns spans that slice cleaned text into the URLs.
* **`FuzzClean`**, over arbitrary bytes, a policy and a split index:
  * the output is valid UTF-8;
  * it holds no 0x1B, no U+0080–U+009F, and no C0 control except those the
    policy keeps;
  * `Clean(Clean(x)) == Clean(x)`;
  * `Write(x[:i])`, `Write(x[i:])`, `Close()` equals `Clean(x)`.

  The target joins `make fuzz` and the CI fuzz step.
* **Differential test.** Over generated well-formed CSI, OSC and DCS
  sequences mixed with text, `Clean` equals `ansi.Strip` followed by
  removing the C0 controls `ansi.Strip` keeps.
* **Benchmarks:** `Clean` over 1 MB of prose, and over 1 MB that is half
  escape sequences.
* **Mutations:**
  * BEL no longer ends an OSC;
  * C1 code points are passed through;
  * the filter forgets its state between writes;
  * `MaxString` is ignored;
  * `SafeURL` accepts any scheme.

### Step 3: `frame`

* The API of MADR §3: `Msg`, `Scheduler` (`New`, `WithFPS`, `WithSend`,
  `Request`, `RequestIn`, `Wait`, `Close`), `Queue[T]`, `Snapshot`,
  `Policy`, `Plan`, `Mode`, `Adaptive` and its options, `Immediate`.
* **Tests, inside `testing/synctest.Test`:**
  * 1000 `Request` calls from 10 goroutines within one frame produce one
    `Msg`;
  * two requests a frame apart produce two `Msg`s at least 1/fps apart, at
    60 and at 120 fps; `WithFPS(500)` is clamped to 120 and `WithFPS(0)`
    becomes 60;
  * `RequestIn(50ms)` followed by `Request()` delivers at once, and
    `RequestIn(50ms)` alone delivers at 50 ms;
  * `Close` makes a pending `Wait` return nil, and `synctest.Wait` then
    finds no goroutine blocked in the package;
  * `WithSend` delivers through the function and needs no `Wait`;
  * `Seq` increases by one per delivered frame.
* **Policy table.** Snapshots step through codex's transitions:
  * enter CatchUp at depth 8 and at age 120 ms;
  * no exit until depth ≤ 2 and age ≤ 40 ms have held for 250 ms;
  * no re-entry within 250 ms, except at depth 64 or age 300 ms;
  * an empty queue resets to Smooth.
* **Queue.** `Drain` releases from the head only, in order; `Snapshot`
  reports depth and the oldest age; it allocates nothing per item once it
  has grown (a `testing.AllocsPerRun` check).
* **Mutations:**
  * the `due` channel is unbuffered;
  * the frame-rate cap is skipped;
  * the exit hold is ignored;
  * the severe bypass is ignored;
  * `Drain` releases from the tail.

### Step 4: `inputfilter`, and the workspace's mouse extension point

* The API of MADR §6: `Filter`, `New` and its options, `Attach`,
  `CoalescedWheelMsg`, `PasteBurst`, `BurstAction`.
* `workspace/render.go`'s `local` gains a case for a `tea.MouseMsg` with
  `WithMouse(tea.Mouse) tea.Msg`. Its doc comment and
  `docs/guides/building-workspaces.md` say so.
* **Tests, with an injected clock:**
  * 100 wheel-down notches within 16 ms reach the filter's caller as one
    `CoalescedWheelMsg` with `DY == 100` once `Attach` flushes the
    residual, and the first notch passes at once;
  * up then down resets the sum;
  * without `Attach`, the residual joins the next wheel event, as the
    MADR documents;
  * motion is throttled to one sample per interval, and clicks, releases
    and keys pass untouched;
  * `PasteBurst` passes codex's cases: a three-character burst, Enter
    inside a burst, a burst ended by an arrow key, the slower Windows idle
    timeout, and ordinary typing at 50 ms intervals passing through;
  * a burst arrives as one `tea.PasteMsg`;
  * after a `tea.PasteStartMsg`, detection is off;
  * under interleaved keys, wheel events and drain messages, the order the
    model sees equals the input order;
  * the filter never calls `send` on the calling goroutine (a `send` that
    panics when called on the event loop's goroutine stays silent).
* **Workspace test.** A `CoalescedWheelMsg` over a pane reaches that pane
  in local coordinates. A custom mouse type with `WithMouse` does too. One
  without it is still dropped.
* **Mutations:**
  * the direction reset is removed;
  * the filter returns every motion sample;
  * the FIFO is bypassed for new input;
  * `local` ignores `WithMouse`;
  * `PasteStartMsg` does not disable detection.

### Step 5: `stream` engine

* The API of MADR §4: `Renderer`, `RendererFunc`, `Plain`, `Doc`, `New`,
  its options, `Buffer`, `NewBuffer`, and `Doc.Pull`.
* **The cut scanner** is unexported, line-oriented and table-tested on its
  own:
  * fences: backtick and tilde, longer closers, indented openers, info
    strings, and an unclosed fence;
  * list items with continuation lines, loose lists, and a new list after
    a blank line;
  * block quotes; HTML blocks of types 1–7; indented code;
  * tables; setext headings; reference and footnote definitions.
* **Property tests** over a block grammar (paragraphs, ATX and setext
  headings, lists, fences, tables, quotes, HTML blocks, reference
  definitions, long runs with no blank line) and random chunkings:
  * **Chunking invariance.** The frame after the last chunk equals the
    frame from one `Append` of the whole text, at the same width.
  * **Finish exactness.** After `Finish`, `Render(w)` equals
    `Renderer.Render(Source(), w)`.
  * **Stated differences.** With a fake renderer whose output is
    predictable, and no relaxed cut, the frame equals the one-shot frame
    once runs of blank lines are collapsed.
  * **Cost.** A counting renderer, calling `Render` after every 20-byte
    chunk of 200 KB of prose with paragraphs, 200 KB with none, one
    200 KB fence and one 200 KB list, proves:
    * every committed byte reaches the renderer once at that width;
    * no `Render` passes more than `RelaxAfter` plus the longest line of
      tail.

    A 200 KB table is the stated exception: its tail is the whole table.
  * **Resize.** Changing width and back renders no segment more than twice.
  * **Sanitizing.** No frame holds 0x1B except the styling the renderer
    added, which a renderer that adds none proves.
* **Holdback tests.** `Hello **wor` shows `"Hello "`; a closed `**world**`
  shows; an unclosed backtick run, `~~` and `[label](http…` are held; the
  partial line is hidden while complete lines are queued.
* **Buffer.** Concurrent `Write`s from 8 goroutines under `-race` lose no
  bytes and keep each writer's order. `Close` makes `Pull` call `Finish`.
  Each `Write` that completes a line calls `Request`.
* **Fuzz.** `FuzzDoc`, over arbitrary text and split points, finds no
  panic, and checks chunking invariance and Finish exactness.
* **`stream/streamtest`** (MADR A2, if accepted): the chunking-invariance
  and Finish-exactness properties live in `CheckRenderer(t, r, o...)`,
  built on `stream`'s exported API only, and `stream`'s tests call it with
  `Plain`. A test that calls it with a renderer whose output depends on
  where the source was cut must fail; that mutation is recorded.
* **Benchmarks:** streaming 200 KB in 20-byte chunks with `Plain`,
  reporting allocations, against a naive re-render of the whole source per
  chunk.
* **Mutations:**
  * a cut is allowed inside a fence;
  * the next-line check for list continuation is skipped;
  * a reference definition does not drop the cache;
  * `Finish` keeps the segmented render;
  * the relaxed cut is never taken;
  * a relaxed cut is taken inside a table;
  * a list's relaxed cut drops the item number;
  * `Append` skips the sanitizer.

### Step 6: `stream.Pane`

* The API of MADR §5: `Pane`, `NewPane`, its options, `PaneKeys`, and the
  `workspace` interfaces it implements (`Pane`, `Changer`, `Titled`,
  `KeyMapper`).
* **Tests, through a real `workspace.Workspace`:**
  * the newest line stays visible while following; scrolling up stops
    following; scrolling to the end resumes it;
  * `Changed` is false after an unrelated message, and true after a
    `Version`, size or scroll change, so the workspace does not ask for
    `View` again otherwise;
  * a `frame.Msg` pulls from an attached `Buffer`;
  * `tea.MouseWheelMsg` and `CoalescedWheelMsg` scroll by their deltas;
  * `View` is O(height): a benchmark at 10 000 lines and at 100 lines
    differs by less than 2×;
  * every binding in `PaneKeys` can be rebound or removed.
* **Goldens** across the 0001 §6 matrix at 60 and 120 columns: mid-stream
  with a partial line, mid-stream inside a table, finished, and scrolled
  up with indicators.
* **Mutations:**
  * `Changed` always returns true;
  * following ignores the user's scroll;
  * the scroll indicators do not use `glyph`.

### Step 7: `stream/glamourmd`, a nested module

*Revised 2026-10-02 for MADR amendment A2.* The original step added glamour
to the root `go.mod`; it now creates a nested module.

* **Blocked** until MADR A2 and
  [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
  are accepted, and Phases 2–5 of
  [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md)
  are complete: the per-module loop in `scripts/go-precheck.sh`, `make` and
  CI, the `go.work` check, and the per-module depguard rules.
* **Order** (0010-MADR §3). This step runs last: after Steps 2–6,
  A1.1–A1.4 and 8 have landed and the owner has tagged that root release,
  so that the adapter can require a published root version. If it ran
  before that tag existed, its `GOWORK=off` gates would fail by design, so
  it waits.
* **Documentation** for the adapter lands in this step: its package
  documentation, a section in `docs/guides/streaming-content.md` on
  `go get`ting the module and on `FromTheme`, and its `docs/README.md` and
  `docs/architecture.md` rows.
* Before the requirement is added, record in the execution record:
  * glamour's newest release and its `go.mod` (0010-REPORT §9 found v2.0.1
    on 2026-10-02);
  * the licence of each module it adds to `stream/glamourmd/go.mod`, from
    each module's licence file;
  * `GOWORK=off govulncheck ./...` in `stream/glamourmd` with the
    requirement, on a scratch copy;
  * which `ansi.StyleConfig` fields set list bullets, rules, quote bars and
    table separators, and any glyph glamour draws that no field controls
    (MADR §7).

  A finding there stops the step for the owner.
* **The module:**
  * `stream/glamourmd/go.mod`: `module
    github.com/maccavelli/go-tui-lib/stream/glamourmd`, `go 1.27.1`, and
    `require` of the root at the tagged release and of
    `charm.land/glamour/v2`, with no `replace`;
  * `go work use ./stream/glamourmd` adds it to `go.work`, in the same
    commit;
  * the package imports only the root's exported packages, never
    `internal/`.
* The API of MADR §7, as amended by A2: `New`, `Renderer.Render`,
  `WithStyle`, `FromTheme`, `WithEmoji`, `WithChromaFormatter`. `FromTheme`
  builds the `StyleConfig` from `theme` and `glyph`, with
  `CodeBlock.Chroma` nil, `CodeBlock.Theme` a built-in chroma style name,
  `Document.Margin` 0, and table separators from `glyph`. The adapter never
  calls `WithEnvironmentConfig`, `WithStylePath` or
  `WithStylesFromJSONFile`.
* **Tests:**
  * renderers are cached per (width, theme revision, emoji), at most four,
    least recently used first out;
  * concurrent `Render` calls under `-race`, at one key and at two, are
    serialized per key through its lock, and no render sees another's
    state;
  * **chroma's global registry is not touched.** One test process renders
    a code block with the dark theme, then with the light theme, and the
    code colours of each follow its own theme. A second check finds no
    style named `charm` in chroma's registry afterwards;
  * `FromTheme` under the dark, light and unknown palettes, and the ASCII
    and NoTTY profiles through `colorprofile`, gives goldens across the
    matrix with no colour where rule 4 forbids it;
  * the ASCII glyph set gives an all-ASCII render of a list, a rule, a
    quote and a table, borders included;
  * a style with `Row` and `Column` separators set renders a table without
    a panic, because `CenterSeparator` is always set;
  * the rendered output has no leading or trailing document margin;
  * the Step 5 properties run again with this renderer, through
    `streamtest.CheckRenderer` (MADR A2): chunking invariance and Finish
    exactness must hold. The "stated differences" check is run and its
    result recorded per construct, as the MADR says.
* **Mutations:**
  * the cache ignores width;
  * the cache ignores the theme revision;
  * the per-key lock is removed (caught by `-race`);
  * `FromTheme` ignores the background;
  * `CodeBlock.Chroma` is left set (the two-theme test must fail);
  * `CenterSeparator` is left unset (the table test must panic or fail).
* **Checks,** per 0010-MADR §4, in `stream/glamourmd` with `GOWORK=off`:
  `go vet`, `go test -race`, `LC_ALL=C go test`, golangci-lint for linux,
  darwin and windows, `go mod tidy -diff` and `govulncheck`. The tests also
  run once in workspace mode. The root module's gates run unchanged, and
  its `go mod tidy -diff` stays clean with no glamour line.

### Steps A1.1 to A1.4: MADR amendment A1

A1 is accepted, with Q6–Q8 answered and recorded in the MADR, so these
steps run ~~after Step 7 and before Step 8~~ after Step 6 and before Step 8
(*2026-10-02, MADR A2:* Step 7 now runs last, after the root release). Each
step follows the rules for every step above.

### Step A1.1: `safetext` presets and `Interpret`

* The API of MADR A1, §2: `Command`, `StatusLine`, and the `Policy` fields
  `Escape`, `KeepSGR` and `LinkSchemes`; `Interpret`, `InterpretOptions`
  and `Row`.
* **Tests:**
  * `Command` escapes each C0 and C1 control, DEL and each listed bidi or
    format code point, and removes nothing: a table with one row per code
    point, including a Trojan Source sample whose escaped form shows the
    real order;
  * `Command` output is one line and holds no 0x1B;
  * `StatusLine` keeps SGR, drops every other CSI, swallows DCS, SOS, PM
    and APC, keeps `ESC ( B` from painting `(B`, and keeps OSC 8 only for
    http, https and mailto with a host;
  * the zero `Policy` and the `Text`, `Strict` and `Line` presets give the
    same output as before A1 (the Step 2 tables pass unchanged);
  * `Interpret`: SGR in `;` and `:` forms, a CR-overwritten progress bar
    ending as its last state, BS, tabs, each listed CSI, an OSC dropped, a
    wide rune advancing two cells, and the row and column caps;
  * `Interpret` is deterministic: two calls over the same bytes are equal.
* **`FuzzClean`** gains the new presets and `Escape`, and `FuzzInterpret`
  checks that no output line is wider than the column cap and none holds a
  sequence other than SGR.
* **Mutations:**
  * `Command` drops a control instead of escaping it;
  * `StatusLine` keeps a non-SGR CSI;
  * `StatusLine` accepts a `javascript:` link;
  * `Interpret` advances one column per rune.

### Step A1.2: `frame` additions

* The API of MADR A1, §3: `Coalescer`, `NewCoalescer`, `Demand`, `Combine`,
  `TickCmd`, `Writer`, `Upgrader`, `UpgradeMsg`.
* **Tests, inside `testing/synctest.Test`:**
  * a barrier pushed between deltas is applied after the earlier deltas and
    before the later ones, for every interleaving of a generated sequence;
  * the maximum wait applies a batch when no frame comes;
  * `Combine` is the maximum; `TickCmd(None)` is nil; an idle program
    whose views all report `None` schedules no timer;
  * `Writer` over a writer that blocks collapses many draws into one, emits
    one stall event after the configured time and one recovery event after
    it unblocks, and loses no byte;
  * `Upgrader`: a newer job for a key replaces an older one; a result with
    an old generation is dropped; cancelling the context stops a job.
* **Mutations:**
  * the barrier does not flush first;
  * `TickCmd(None)` returns a command;
  * `Writer` sends a second frame while one is in flight;
  * `Upgrader` keeps stale results.

### Step A1.3: `inputfilter` stages

* The API of MADR A1, §6: the stage options, `Capture`, `Quarantine`,
  `ClipboardImageRequestMsg`, `PastedPath`, the new `PasteBurstConfig`
  options and `WithWheelProfile`.
* **Tests:**
  * **Split at every byte boundary.** Each SGR mouse report, focus report,
    corrupted X10 report and late probe reply is fed split at every byte
    position, and in every two-way split. Each reaches the model as exactly
    one message, or none for a swallowed reply, and no stray key.
  * a bare Esc flushes a held fragment at once;
  * `Capture` keeps text, Backspace, Shift and Alt+Enter, and pastes, and
    cuts at the first Esc; `Quarantine` drops everything until its
    deadline, then passes input;
  * CRLF and lone CR become LF inside a bracketed paste; an empty
    bracketed paste becomes `ClipboardImageRequestMsg`;
  * `PastedPath` accepts a quoted path, `file://`, a drive letter and a UNC
    path, unescapes backslashes except on Windows, and refuses prose;
  * grok's PowerShell burst cases, including Enter followed by Ctrl+J;
  * a profile of 3 events per notch turns 3 wheel events into one notch.
* **Broken input.** A deliberately truncated SGR mouse report at the end of
  the input is held, then flushed as keys by the next bare Esc, and never
  reaches the model as a mouse event.
* **Mutations:**
  * fragment reassembly is skipped;
  * the X10 repair is skipped;
  * late replies pass as keys;
  * Esc does not flush.

### Step A1.4: `stream` additions

* The API of MADR A1, §4: `Doc.View`, `RenderView`, `Highlighter`,
  `FitTable`, `Bounded`, and the dialect rules.
* **Tests:**
  * the chunking-invariance property of Step 5 also holds for `RenderView`:
    the line map, links, code blocks and tables after the last chunk equal
    those of a one-shot `Append`;
  * a link wrapped across rows keeps one ID, before and after a tail
    re-render;
  * `~**10%**` is not struck through; `~~x~~` is; an unclosed `\(` is held
    until `Finish` when math is on;
  * a counting `Highlighter` highlights each line of a growing open fence
    once; a theme revision or a width change drops its caches; inputs past
    each limit are shown plain;
  * `FitTable` keeps numbers and URLs whole, pads a row with a wide glyph to
    the exact width, and falls back to records, stacked records and source
    as the width shrinks; goldens across the 0001 §6 matrix, including the
    ASCII border set;
  * `Bounded` keeps head and tail past its limit, closes an escape cut at
    the head, and reports the same omitted count at two widths.
* **Mutations:**
  * `RenderView` is not cut back on a tail re-render;
  * a single `~` strikes through;
  * the open fence is re-highlighted whole on each line;
  * `FitTable` clips by byte;
  * `Bounded` counts omitted rows instead of lines.

### Step 8: example, guide and close-out

* `ExampleDoc_stream` streams a Markdown answer through a `Buffer` from a
  goroutine, into a `stream.Pane` in a `SidebarRight` workspace, with
  `frame.Scheduler`, and prints the finished frame. Its comment shows the
  one line that re-arms `Wait`.
* `docs/guides/streaming-content.md`:
  * feeding a `Doc` from the event loop or from goroutines;
  * the frame scheduler, and why `Wait` is re-armed;
  * choosing a drain policy;
  * the input filter, `Attach`, and paste bursts;
  * what `safetext` removes, and adding trusted links;
  * what may differ before `Finish`;
  * if A1 landed: the `Command` and `StatusLine` presets, `Interpret`, the
    coalescer and tick demand, the output `Writer`, the input stages, and
    `RenderView`.
* `docs/architecture.md` lists the ~~five~~ four root packages and their
  imports; Step 7 adds `stream/glamourmd` as a nested module
  (*2026-10-02, MADR A2*). `docs/README.md` gains rows. Release notes go in
  the execution record. The owner then tags the root release, which Step 7
  waits for.

## Verification

* Every step's mutations are killed.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`, including `FuzzClean` and `FuzzDoc`, and `FuzzInterpret`
    if A1 landed.
* ~~`go mod tidy -diff` is clean. `go.mod` adds exactly
  `charm.land/glamour/v2` as a direct requirement, and only in Step 7.~~
  *2026-10-02 (MADR A2):* `go mod tidy -diff` is clean in every module, with
  `GOWORK=off`. The root `go.mod` gains no requirement in any step, and has
  no glamour, goldmark, chroma or bluemonday line. `stream/glamourmd/go.mod`
  requires exactly the root, at a release version, and
  `charm.land/glamour/v2`, and has no `replace`.
* Every gate of 0010-MADR §4 passes for `stream/glamourmd` as well as the
  root, and `go.work` lists both.
* `internal/conformance` finds no `os.Stdout`, `os.Stderr`, `AltScreen` or
  `signal.Notify` in the new packages.
* No new package starts a goroutine outside `Scheduler`'s timer callback
  and `inputfilter`'s drain send, and, if A1 landed, `Writer`'s writer and
  `Upgrader`'s jobs. A test with `synctest` checks each.
* The identifier scan finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The steps land in order. pi-go adopts `stream.Pane`,
  `frame` and `inputfilter` under its own records. The owner tags, in the
  order 0010-MADR §3 sets:
  1. the root release that carries Steps 2–6, A1.1–A1.4 and 8;
  2. `stream/glamourmd/v0.1.0`, once Step 7's `go.mod` requires that root
     release;
  3. a consumer smoke test, in a scratch module outside the repository:
     `go get github.com/maccavelli/go-tui-lib/stream/glamourmd@v0.1.0` and
     `go build ./...`. A second scratch consumer that gets only the root
     release has no glamour line in `go.mod`, `go.sum` or `go list -m all`.

  Both results go in the execution record.
* **Rollback.** Before the push, each step is one local commit. After it,
  the packages are new and additive, so a fix goes forward in a patch
  release. If glamour's graph proves unacceptable, `stream/glamourmd` is
  not tagged again, its directory and `go.work` entry are removed, and
  `stream` keeps working with `Plain` or a program's own renderer. Its
  published tags stay, as tags are never deleted (0010-REPORT §1).

## Execution Record

None yet.
