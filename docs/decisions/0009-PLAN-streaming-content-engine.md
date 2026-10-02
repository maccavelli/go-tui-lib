---
status: proposed
date: 2026-10-02
associated-madr: "0009-MADR-streaming-content-engine.md"
---
# Implement the streaming content engine

Associated MADR: [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)

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
| 7 | `stream/glamourmd/`; `go.mod`, `go.sum` | the glamour adapter |
| 8 | `stream/example_test.go`, `docs/guides/streaming-content.md`, `docs/` | the example, the guide, close-out |

`go.mod` gains `charm.land/glamour/v2` v2.0.1, or its newest release on the
day, in Step 7 and no other step. Steps 2–6 add no module.

### Out of scope

* A virtualized transcript of many `Doc`s, watermark pruning, and inline
  scrollback with OSC 133. Each is a later record (MADR option D and
  0003-REPORT §7).
* Syntax highlighting other than glamour's own chroma use.
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
5. **Commit.** One commit per step, with `git commit --no-edit`, after the
   owner authorizes commits to `main` in that turn. The execution record
   gets each step's evidence before its commit.

## Implementation Steps

### Step 1: records

The owner accepts the MADR, answering Q1–Q5. Record the answers, set the
MADR `accepted` and this PLAN `in-progress`, and update `docs/README.md`.
The records this one builds on
([0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
for `ansi.Method`, and the records before it in the order) are complete
first, or this step records which parts wait for them.

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

### Step 7: `stream/glamourmd`

* Before the requirement is added, record in the execution record:
  * glamour's newest release and its `go.mod`;
  * the licence of each module it adds to `go.mod`, from each module's
    licence file;
  * `govulncheck ./...` with the requirement on a scratch copy;
  * which `ansi.StyleConfig` fields set list bullets, rules and quote bars,
    and any glyph glamour draws that no field controls (MADR §7).

  A finding there stops the step for the owner.
* The API of MADR §7: `New`, `Renderer.Render`, `WithStyle`, `FromTheme`,
  `WithEmoji`, `WithChromaFormatter`.
* **Tests:**
  * renderers are cached per width, at most four, least recently used
    first out;
  * concurrent `Render` calls under `-race` are serialized;
  * `FromTheme` under the dark, light and unknown palettes, and the ASCII
    and NoTTY profiles through `colorprofile`, gives goldens across the
    matrix with no colour where rule 4 forbids it;
  * the ASCII glyph set gives an all-ASCII render of a list, a rule and a
    quote;
  * the Step 5 properties run again with this renderer: chunking
    invariance and Finish exactness must hold. The "stated differences"
    check is run and its result recorded per construct, as the MADR says.
* **Mutations:**
  * the cache ignores width;
  * the mutex is removed (caught by `-race`);
  * `FromTheme` ignores the background.

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
  * what may differ before `Finish`.
* `docs/architecture.md` lists the five packages and their imports.
  `docs/README.md` gains rows. Release notes go in the execution record.

## Verification

* Every step's mutations are killed.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`, including `FuzzClean` and `FuzzDoc`.
* `go mod tidy -diff` is clean. `go.mod` adds exactly
  `charm.land/glamour/v2` as a direct requirement, and only in Step 7.
* `internal/conformance` finds no `os.Stdout`, `os.Stderr`, `AltScreen` or
  `signal.Notify` in the new packages.
* No new package starts a goroutine outside `Scheduler`'s timer callback
  and `inputfilter`'s drain send. A test with `synctest` checks each.
* The identifier scan finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The steps land in order. pi-go adopts `stream.Pane`,
  `frame` and `inputfilter` under its own records. The owner tags.
* **Rollback.** Before the push, each step is one local commit. After it,
  the packages are new and additive, so a fix goes forward in a patch
  release. If glamour's graph proves unacceptable, `stream/glamourmd` is
  removed with its requirement, and `stream` keeps working with `Plain` or
  a program's own renderer.

## Execution Record

None yet.
