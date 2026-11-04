---
status: accepted
date: 2026-11-01
decision-makers: owner
consulted: 0003-REPORT-agent-tui-ecosystem-research.md (§3 agent TUIs, §5 terminal standards); crush, codex, gemini-cli, goose and aider source; Charm v2 APIs (bubbletea v2.0.10, lipgloss v2.0.6, bubbles v2.2.1, x/ansi v0.11.8); charm.land/glamour/v2 v2.0.1; for amendment A2, 0010-REPORT-nested-modules-and-adapter-sources.md (§2, §9, §10)
informed: pi-go
---
# Stream agent output through a stable-prefix Markdown engine, a frame scheduler, an input filter and a safe-text sanitizer

## Context and Problem Statement

On 2026-10-01 the owner asked:

> i want to stay working on go-tui-lib, i want to ensure the
> bubbletea/lipgloss/charm stack is tightly integrated, coded idiomatically,
> based on go1.27.1 optimizations and standards. i want to enhance and expand
> the tui library functionality in 5 more ways that will bring benefit and
> value to the codebase functionality. … future-proof is a focus.

On 2026-10-02, after the research was presented, the owner said: "write
findings into a report then follow recommendations and proceed." The
recommendation named five expansions. This record is the fifth: the
streaming content engine (candidate 5 in
[0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§7).

The owner's standing direction also applies: build speculative API when it
is sensible, for extensibility, flexibility and idiomatic, modular design.

pi-go's main pane is an agent transcript. Model tokens arrive tens to
hundreds of times a second, as Markdown, from a source the program does not
trust. This record decides how go-tui-lib renders that stream: cheaply,
without flicker, without letting the stream steer the terminal, and without
the input queue falling behind.

Evidence (read-only, 2026-10-02):

* **Bubble Tea calls `View` after every message.** `bubbletea` v2.0.10
  `tea.go:888` runs `p.render(model)` after each `model.Update`. The
  renderer only flushes at its frame rate (`renderer.go:13-14`: default
  60 fps, maximum 120), but `View` itself runs once per message. A program
  that turns each token into a message re-renders its Markdown once per
  token. That is O(n) work per token and O(n²) per turn.
* **The filter sees every message, on the event loop.** `tea.go:764-768`
  applies `WithFilter`'s `func(Model, Msg) Msg` to each message before
  `Update`, and drops it when the filter returns nil. `Program.Send`
  (`tea.go:1192-1197`) writes to an unbuffered channel (`tea.go:606`), so
  a filter that called `Send` itself would deadlock.
* **Wheel events carry no magnitude.** `tea.MouseWheelMsg` is a `Mouse`
  whose `Button` is `MouseWheelUp` / `Down` / `Left` / `Right`
  (`mouse.go:34`, `113`). A fast trackpad sends one message per notch.
* **The workspace drops mouse messages it does not know.**
  `workspace/render.go:274-291` (`local`) translates only the four `tea`
  mouse types into pane cells and returns false for any other
  `tea.MouseMsg`.
* **`ansi.Strip` is not a sanitizer.** `x/ansi` v0.11.8 `width.go:11-58`
  removes escape sequences but keeps every byte the parser *executes*: BEL,
  BS, CR and the other C0 controls survive. A C1 control written as UTF-8
  (U+009B is CSI) is collected as an ordinary rune and survives too.
* **Prior art, read in source** (the report's §3 has the links):
  * **crush** (`internal/ui/chat/streaming_markdown.go`) caches the
    glamour render of a byte prefix that ends at a blank line where no
    construct is open. It renders only the tail on each flush and joins the
    two with one blank line. Its comments say two joined renders are not in
    general equal to one render, so the cut test is deliberately cautious.
    Past 2 KiB of tail with no blank line, it accepts a cut at a plain
    newline (`relaxBoundaryAfter = 2 << 10`). It locks one mutex per glamour
    renderer, because "glamour's Render is stateful and not safe for
    concurrent invocation".
  * **crush** (`internal/ui/model/filter.go`) merges wheel samples into a
    `CoalescedWheelMsg{DeltaX, DeltaY}` and throttles motion to one sample
    per 16 ms. A change of direction resets the sum. A residual delta inside
    the last 16 ms is held until the next wheel event, and lost if none
    comes.
  * **codex** (`streaming/chunking.rs`) drains queued lines in two gears.
    Smooth releases one line per tick. CatchUp releases the whole queue.
    It enters CatchUp at a depth of 8 lines or when the oldest line is
    120 ms old. It leaves when depth ≤ 2 and age ≤ 40 ms have held for
    250 ms. A further 250 ms hold blocks re-entry unless the backlog is
    severe (64 lines or 300 ms).
  * **codex** (`tui/frame_requester.rs`) gives widgets a cloneable
    `FrameRequester` with `schedule_frame()` and `schedule_frame_in(d)`. A
    scheduler task merges requests and caps draws at 120 fps.
  * **codex** (`streaming/controller.rs`) keeps a pipe table in the
    mutable tail until the stream ends, because one more row can change
    every column's width.
  * **codex** (`bottom_pane/paste_burst.rs`) recognises pastes on terminals
    without bracketed paste with a pure state machine. A burst is 3
    characters with at most 8 ms between them. It flushes after 8 ms idle,
    or 60 ms on Windows. Enter inside a burst is a newline.
  * **codex** (`insert_history.rs`) filters control characters out of
    untrusted text before adding trusted OSC 8 links. Its test injects
    `ESC[2J`, an OSC 52 clipboard write and raw C1 controls.
  * **goose** (`streaming_buffer.rs`) holds back unclosed inline syntax:
    `push("Hello **wor")` returns `"Hello "`.
  * **gemini-cli** (`markdownUtilities.ts`, `findLastSafeSplitPoint`) cuts
    at the last blank line outside a code fence, or before the fence that
    encloses the end.
* **glamour v2 is published.** `charm.land/glamour/v2` lists `v2.0.0` and
  `v2.0.1` on the module proxy. `v2.0.1` was tagged 2026-06-12 and is MIT.
  Its `go.mod` requires `charm.land/lipgloss/v2` v2.0.4, `x/ansi` v0.11.7,
  `alecthomas/chroma/v2` v2.14.0, `yuin/goldmark` v1.7.8,
  `yuin/goldmark-emoji`, `microcosm-cc/bluemonday`, `golang.org/x/text`
  and two `charmbracelet/x/exp` modules. crush builds glamour v2.0.1 with
  lipgloss v2.0.6, x/ansi v0.11.8 and the same ultraviolet pseudo-version
  this module uses, so the versions resolve together. pkg.go.dev documents
  no concurrency guarantee for `TermRenderer`, and `WithWordWrap` fixes the
  width when the renderer is built.

## Decision Drivers

* **Streaming cost is O(delta).** Appending a chunk does work in
  proportion to the chunk, not to the document. Rendering re-renders only
  the part that can still change.
* **The final frame is exactly right.** Whatever happened while streaming,
  the finished document renders exactly as a one-shot render would.
* **Chunking does not change what is drawn.** The same text, split into
  chunks in any way, draws the same frame. Cut points depend on the content
  alone, never on when a chunk arrived.
* **Untrusted text cannot steer the terminal.** Model and tool output
  cannot clear the screen, set the clipboard, change the title, open a
  link or hide text. The sanitizer works on a stream, so a sequence split
  across two chunks is still removed.
* **Input stays responsive under load.** A wheel flood or a paste typed as
  keys does not queue key presses behind it.
* **The renderer is a choice, not a dependency.** The engine works with no
  Markdown library. glamour is one adapter.
* **Bubble Tea idiom.** Everything is messages, commands and filters. No
  package holds a `*tea.Program`, and none starts a goroutine that runs
  without work to do.
* **0001 §6 holds.** No output to `os.Stdout` or `os.Stderr`, glyphs from
  `glyph`, widths in cells with `x/ansi`, and the golden matrix.

## Considered Options

* **A. Four small packages (`safetext`, `frame`, `inputfilter`, `stream`), a renderer-agnostic stable-prefix engine, and a glamour adapter subpackage `stream/glamourmd`.**
* **B. One `stream` package with glamour built in,** as crush does.
* **C. Re-render the whole document on every frame,** and rely on
  `Changer` and Bubble Tea's frame rate, with aider's adaptive delay.
* **D. Inline scrollback,** as codex does: finished lines are printed
  above the program with `tea.Println`, and only the live tail is drawn.

## Decision Outcome

Chosen option: **"A"**, because:

* it is the only option that meets every driver;
* each part is useful alone: `safetext` for any untrusted string, `frame`
  for any animation, `inputfilter` for any program with a mouse;
* the engine's committed segments are exactly what option D would print,
  so inline scrollback can be added later without redesign.

### 1. Packages and their dependencies

```text
 stream/glamourmd  glamour v2 Renderer, one TermRenderer per width
                                         → stream, theme, charm.land/glamour/v2
 stream            Doc (stable-prefix Markdown), Buffer, Pane, Plain renderer
                                         → safetext, frame, workspace, x/ansi
 inputfilter       tea.WithFilter filter: wheel coalescing, motion throttle,
                   paste bursts          → bubbletea
 frame             Scheduler (RequestFrame), Queue[T], drain Policy
                                         → bubbletea
 safetext          streaming sanitizer for untrusted text; URL check
                                         → stdlib, glyph
```

* **Imports point downward.** `safetext` imports only the standard
  library and `glyph`. `frame` and `inputfilter` import only Bubble Tea.
  Only `stream/glamourmd` imports glamour.
* **One new module.** `charm.land/glamour/v2` v2.0.1, which this record
  names (AGENTS.md, Dependencies). It enters `go.mod` in the commit that
  adds `stream/glamourmd`. A program that never imports that subpackage
  compiles none of glamour.
* **`workspace` gains one extension point** (§6), so that mouse messages
  it does not know are still routed.
* **`glyph.Set` gains `Control`**, the visible mark for a removed escape
  sequence (§2), with a Unicode form `␛` and an ASCII form `^`.

### 2. `safetext`: untrusted text in, printable text out

```go
// Policy says what survives. The zero value is the strictest: printable
// text only. The presets below are what callers normally use.
type Policy struct {
    Tabs      bool // keep \t
    Newlines  bool // keep \n; \r\n becomes \n, a lone \r is dropped
    Flatten   bool // \n and \t each become one space (wins over Tabs and Newlines)
    Bidi      bool // keep U+202A–U+202E and U+2066–U+2069
    Visible   bool // replace each removed sequence with glyph.Set.Control
    Glyphs    glyph.Set
    MaxString int // bytes an OSC/DCS/SOS/PM/APC string may run before it is abandoned; 0 means 64 KiB
}
func Text() Policy    // prose: Tabs, Newlines and Bidi
func Strict() Policy  // commands and paths shown for approval: Tabs and Newlines, no Bidi
func Line() Policy    // one line: Flatten, no Bidi

func Clean(s string, p Policy) string

// Filter is the streaming form: Clean(a+b) == Write(a)+Write(b)+Close().
type Filter struct{ /* parser state, carried between writes */ }
func NewFilter(w io.Writer, p Policy) *Filter
func (f *Filter) Write(b []byte) (int, error)
func (f *Filter) Close() error // drops an unfinished sequence; never writes it

// SafeURL accepts http, https and mailto URLs with no control byte, no
// space and no ESC or ST, so a trusted OSC 8 link cannot be broken out of.
func SafeURL(u string) (string, bool)
func URLs(s string) iter.Seq2[int, int] // byte spans of web URLs in cleaned text
```

* **The grammar is ECMA-48, with the DEC parser's recovery rules**
  (vt100.net's state machine, which `x/ansi`'s parser also follows):
  * `ESC [` starts a CSI: parameter bytes 0x30–0x3F, intermediates
    0x20–0x2F, and a final byte 0x40–0x7E.
  * `ESC ]` starts an OSC. It ends at BEL, at ST (`ESC \`) or at U+009C.
  * `ESC P` (DCS), `ESC X` (SOS), `ESC ^` (PM) and `ESC _` (APC) start a
    string. It ends at ST, and BEL is accepted as well for robustness.
  * `ESC` followed by intermediates 0x20–0x2F and a final byte 0x30–0x7E
    is an nF, Fp, Fe or Fs escape.
  * Any other byte after `ESC` is dropped together with the `ESC`.
  * CAN (0x18) and SUB (0x1A) abort any sequence. `ESC` inside a sequence
    starts a new one.
  * The C1 code points U+0080–U+009F are treated as their 7-bit forms:
    U+009B is CSI, U+009D is OSC, and so on. They never reach the output.
* **Everything a sequence contains is removed**, not just its `ESC`. codex
  removes only control characters, so `ESC[2J` shows as `[2J`. Here the
  whole sequence goes. `Visible` shows one `Control` glyph in its place
  instead, for audit and approval views.
* **What survives:** printable text, and `\n` and `\t` as the policy says.
  Every other C0 control and DEL (0x7F) is dropped. Invalid UTF-8 becomes
  U+FFFD.
* **A string sequence cannot swallow the document.** One that runs past
  `MaxString` bytes is abandoned. Its content so far is dropped and the
  parser returns to ground, so a lost terminator hides at most `MaxString`
  bytes.
* **Trusted links come after cleaning.** `SafeURL` validates a link target,
  and `URLs` finds them in cleaned text. The styling layer adds the OSC 8
  link with `lipgloss.Style.Hyperlink`, or with the link service in
  [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md).

### 3. `frame`: one frame for many requests

```go
// Msg is a due frame. Seq increases by one per delivered frame.
type Msg struct{ At time.Time; Seq uint64 }

// Scheduler merges frame requests. Its methods are safe for concurrent use.
type Scheduler struct{ /* mutex, earliest deadline, one timer, due chan (cap 1) */ }
func New(opts ...Option) *Scheduler     // WithFPS(n): default 60, clamped to 1..120 as tea.WithFPS is
func (s *Scheduler) Request()            // as soon as the frame-rate cap allows
func (s *Scheduler) RequestIn(d time.Duration) // the earliest pending deadline wins
func (s *Scheduler) Wait() tea.Cmd       // returns Msg when a frame is due
func (s *Scheduler) Close()              // a pending Wait returns nil

// Queue holds timestamped items for a Policy to release. It is not safe
// for concurrent use; it lives on the event loop.
type Queue[T any] struct{ /* ring of {T, time.Time} */ }
func (q *Queue[T]) Push(v T, at time.Time)
func (q *Queue[T]) Snapshot(now time.Time) Snapshot
func (q *Queue[T]) Drain(p Policy, now time.Time) iter.Seq[T]

type Snapshot struct{ Depth int; Oldest time.Duration }
type Policy interface{ Decide(s Snapshot, now time.Time) Plan }
type Plan struct{ Take int; Mode Mode } // Mode: Smooth or CatchUp
func Adaptive(opts ...AdaptiveOption) Policy // codex's two gears and thresholds
func Immediate() Policy                      // release everything every frame
```

* **The scheduler is a command, not a `*tea.Program`.** `Wait` returns a
  `tea.Cmd` that blocks on the `due` channel. The program re-arms it when
  each `Msg` arrives:
  `return m, tea.Batch(m.ws.Update(msg), m.frames.Wait())`.
  * This is Bubble Tea's own pattern for listening to a channel. It works
    under `teatest`, over SSH through wish, and inside `testing/synctest`.
  * The alternative, a scheduler that calls `Program.Send`, would need the
    program before the model is built, and could not be tested without
    one. `WithSend(func(tea.Msg))` is offered for programs that want
    push delivery instead; then `Wait` is not used.
* **No goroutine runs without work.** `Request` sets a deadline and arms
  one `time.Timer`. When it fires, a token goes into `due`, whose capacity
  of one merges any number of requests. Between frames, the only goroutine
  is Bubble Tea's command goroutine, parked on the channel.
* **The cap matches Bubble Tea.** The default is 60 fps, and the bounds
  1..120 are `tea.WithFPS`'s, so frames are not produced faster than the
  renderer flushes.
* **The two gears are codex's, with its numbers as defaults:**
  * Smooth releases one item per frame. CatchUp releases the whole queue.
  * CatchUp is entered at depth ≥ 8 or when the oldest item is at least
    120 ms old.
  * It is left when depth ≤ 2 and age ≤ 40 ms have held for 250 ms.
  * After leaving, re-entry waits 250 ms unless the backlog is severe
    (depth ≥ 64 or age ≥ 300 ms).
  * Every number is an `AdaptiveOption`. An empty queue resets to Smooth.
* **`Queue` is generic** (`Queue[T]`), so the same policy paces log lines
  or tool output as well as Markdown lines.

### 4. `stream`: stable-prefix Markdown

```go
// Renderer turns Markdown into styled text at a width. Implementations
// need not be safe for concurrent use; Doc calls one at a time.
type Renderer interface{ Render(src string, width int) (string, error) }
type RendererFunc func(src string, width int) (string, error)
func Plain(m ansi.Method) Renderer // no Markdown: word-wraps the text in cells

// Doc is one streamed document. It is not safe for concurrent use: it is
// owned by the event loop. Buffer is the goroutine-safe way in.
type Doc struct{ /* source, committed segments, scanner state, caches */ }
func New(r Renderer, opts ...Option) *Doc
func (d *Doc) Append(s string)              // sanitized with the Doc's safetext.Policy
func (d *Doc) Write(p []byte) (int, error)  // io.Writer form of Append
func (d *Doc) Finish()                      // no more input; the next Render is one-shot
func (d *Doc) Reset()
func (d *Doc) Render(width int) string      // cached by (width, version)
func (d *Doc) Version() uint64              // bumps on every visible change
func (d *Doc) Finished() bool
func (d *Doc) Source() string               // the cleaned source so far
func (d *Doc) Pull(b *Buffer, p frame.Policy, now time.Time) bool // true while more is queued

// Buffer collects chunks from any goroutine, timestamps whole lines, and
// requests a frame. Doc.Pull drains it on the event loop.
type Buffer struct{ /* mutex, pending bytes, frame.Queue[string] */ }
func NewBuffer(s *frame.Scheduler) *Buffer
func (b *Buffer) Write(p []byte) (int, error) // safe for concurrent use
func (b *Buffer) Close() error                // end of stream; Pull then calls Finish

// Options: WithPolicy(safetext.Policy) (default safetext.Text()),
// WithRelaxAfter(bytes) (default 2 KiB), WithInlineHoldback(bool) (default on),
// WithPartialLine(bool) (default on), WithMethod(ansi.Method).
```

* **Input is cleaned first.** `Append` passes every chunk through a
  `safetext.Filter`, which carries parser state from chunk to chunk.
  Nothing in the source, the cache or the output can hold an escape
  sequence the stream sent.
* **Cuts are found by scanning forward, once per line.**
  * Each complete line is classified once, so work is O(delta).
  * The scanner keeps the state that decides a cut:
    * the open fence's character, length and indent (CommonMark: a closer
      uses the same character, at least as many, and has no info string);
    * the open containers (list items with their content indent, block
      quotes);
    * an open HTML block of types 1–5, which ends at its own end condition,
      not at a blank line;
    * whether the current block is a table, a paragraph or indented code.
* **A cut is a blank line where nothing stays open.**
  * A candidate is a blank line outside a fence and outside an HTML block
    of types 1–5.
  * Its decision waits for the first non-blank line after it. It is
    rejected if that line continues an open list item (a marker of the
    same list, or indented to the item's content) or continues indented
    code.
  * Otherwise the candidate is committed. The segment before it is
    rendered once, cached, and never rendered again at that width.
  * Because a decision depends only on complete lines, the cut set is a
    function of the content, not of the chunking.
* **A long run with no cut gets a relaxed cut**, as in crush, so the
  tail stays small.
  * Once more than `RelaxAfter` bytes have passed with no committed cut,
    the next line boundary where a block can be split is a cut:
    * in a paragraph, at any newline;
    * in a fence, at any newline: the segment's render input closes the
      block, and the next segment's reopens it with the same info string;
    * in a list, before an item marker at the list's own indent. An ordered
      list keeps its numbers, because the next segment starts with its
      item's number from the source;
    * in a block quote, at any newline.
  * Tables and HTML blocks are never split: one more row can change every
    column, and an HTML block's end decides its meaning.
  * Relaxed cuts are counted forward from the last cut. They are a function
    of the content too.
* **What a `Render` costs.** Each committed segment is rendered once per
  width. Each `Render` also renders the tail, which holds at most
  `RelaxAfter` bytes plus one line, unless the tail is one unfinished table
  or HTML block. So the total is O(n) for the segments, plus O(tail) for
  each frame. The frame scheduler (§3) bounds how many frames there are.
* **Tables and fences are held in the tail, as codex holds them.** No cut
  falls inside a table, so a growing table stays in the part that is
  re-rendered each frame until a blank line ends it. An unclosed fence
  renders as code to the end of the document, as CommonMark specifies.
* **What the tail shows** (display only; the source is never changed):
  * the partial last line, unless `WithPartialLine(false)`. It is shown
    only when no complete lines are still queued, so text never appears
    out of order;
  * with inline holdback on, the partial line stops before an unclosed
    `**`, `*`, `__`, `_`, `~~`, an unclosed run of backticks, or a link
    whose `](…)` has not closed. This is goose's rule.
* **A width change re-renders the committed segments at the new width**,
  once and lazily, and keeps the cut set. Segments are cached per width,
  for the current and the previous width, so dragging a separator back
  and forth does not re-render the document twice per step.
* **Reference definitions reset the cache.** A link reference definition
  (`[x]: url`), or a footnote definition (`[^x]:`), can change how earlier
  text renders. When a committed segment holds one, every cached segment is
  dropped and rendered again once. That is still a function of the content.
  *Amended 2026-11-03, by
  [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
  D16:* a `Renderer` renders the segment it is given, so a segment
  rendered again alone cannot see a definition in another. Each
  segment's render input carries the definitions committed so far,
  after a blank line, which a Markdown renderer renders as nothing; and
  `Plain`, which is not one, prints no definition line.
* **`Finish` makes the frame exact.** After `Finish`, `Render(width)` is
  `Renderer.Render(Source(), width)`, byte for byte, rendered once and
  cached.
* **Before `Finish`, the frame may differ from a one-shot render in two
  stated ways, and in no other:**
  1. segments are joined with exactly one blank line, after trimming each
     render's leading and trailing blank lines (glamour adds margins);
  2. a block split by a relaxed cut renders as two adjacent blocks of the
     same kind: a paragraph is wrapped as two, and a code block, a list or
     a quote shows as two.

  Without relaxed cuts, only the first can occur.
* **`Version` and `Finished` are crush's list-item contract.** A later
  virtualized transcript record can hold `Doc`s as items, freezing those
  that are finished.
* **Errors do not lose text.** A segment the renderer fails on is shown as
  its cleaned source, wrapped with `Plain`.

### 5. `stream.Pane`: a document in a workspace pane

```go
type Pane struct{ /* *Doc, scroll offset, follow flag, cached lines */ }
func NewPane(d *Doc, opts ...PaneOption) *Pane // WithTitle, WithFollow(bool), WithKeys(PaneKeys)
func (p *Pane) Update(msg tea.Msg) (workspace.Pane, tea.Cmd)
func (p *Pane) View(width, height int) string // O(height) once lines are cached
func (p *Pane) Changed() bool                 // workspace.Changer
func (p *Pane) Title() string                 // workspace.Titled
func (p *Pane) Keys() []key.Binding           // workspace.KeyMapper
func (p *Pane) Doc() *Doc
```

* **It follows the tail.** The newest line is visible until the user
  scrolls up. Scrolling back to the end resumes following. The scroll
  indicators come from `glyph`.
* **It is a `Changer`.** `Changed` reports true only when the `Doc`'s
  `Version`, the size or the scroll offset moved. Bubble Tea's call to
  `View` after each unrelated message then costs the workspace nothing.
* **It handles** `workspace.SizeMsg`, `frame.Msg` (it calls `Pull` when a
  `Buffer` is attached), the keys in `PaneKeys` (rebindable, as every
  binding is), `tea.MouseWheelMsg`, and `inputfilter.CoalescedWheelMsg`.
* **It measures with the workspace's width method**, `ansi.Method`, which
  [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
  aligns with Bubble Tea's.

### 6. `inputfilter`: coalesce noise before it queues

```go
// Filter is passed to tea.WithFilter. It runs on the event loop only.
type Filter struct{ /* clock, wheel sums, last-sample times, paste state, FIFO */ }
func New(opts ...Option) *Filter // WithInterval(16ms), WithClock(func() time.Time),
                                 // WithPasteBurst(PasteBurstConfig), WithWheel(bool), WithMotion(bool)
func (f *Filter) Filter(m tea.Model, msg tea.Msg) tea.Msg
func (f *Filter) Attach(send func(tea.Msg)) // the program's Send; enables timed flushes

// CoalescedWheelMsg is many wheel notches as one message. The field is
// not an embedded tea.Mouse: a field named Mouse and the method Mouse
// cannot both exist.
type CoalescedWheelMsg struct{ Pointer tea.Mouse; DX, DY int }
func (m CoalescedWheelMsg) Mouse() tea.Mouse              // with String, a tea.MouseMsg
func (m CoalescedWheelMsg) String() string
func (m CoalescedWheelMsg) WithMouse(t tea.Mouse) tea.Msg // so the workspace can localize it

// PasteBurst is the pure state machine: the caller passes the time, and
// it does no I/O.
type PasteBurst struct{ /* config, held char, buffer, window */ }
func (p *PasteBurst) Key(k tea.KeyPressMsg, now time.Time) BurstAction
func (p *PasteBurst) Tick(now time.Time) (flush string, ok bool)
```

*Amended 2026-11-02, by
[0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
D10, D10a and D12:* A6 item 8 makes `inputfilter` a core, and the PLAN's
Step 4 gives its API: the pure `Wheel` over `Scroll`, `Throttle` and
`PasteBurst` over `termkey.Input`, with `Key` giving a `BurstAction`
(`Pass`, `Hold`, `Release`) and `Due` its deadline; `Flusher`, the driver
over the three, whose `Flushes` says only that a flush fell due, taken by
the caller's `Flush` on its own goroutine, so that a key typed after a
burst never overtakes it; and the options, `inputfilter`'s, which
`inputfilter/inputfiltertea`'s `NewFilter` takes. `WithWheelProfile` is
A1's, built by the PLAN's Step A1.3.

* **Wheel.** Notches are summed, and a change of direction resets the sum,
  as crush does. The first notch passes at once, and later ones at most one
  per interval, carrying the sum.
  * With `Attach`, a timer emits the residual sum after the last notch, so
    no scroll is lost.
  * Without `Attach`, the residual joins the next wheel event, which is
    crush's behaviour, and is documented as such.
* **Motion** is throttled to one sample per interval. Clicks, releases,
  keys and every other message pass untouched.
* **Paste bursts are opt-in** (owner question Q3), and need `Attach`:
  * `PasteBurst` uses codex's rules: 3 characters, 8 ms apart at most,
    flushed after 8 ms idle (60 ms on Windows), with Enter inside a burst
    as a newline;
    *annotated 2026-11-02, by the PLAN's D11:* codex at `4b4a5cf` starts a
    burst at the second fast character, holding the first
    (`paste_burst.rs:233-242`), and needs 3 only to take back typed text,
    which a filter cannot; it also reads an Enter as a newline for 120 ms
    after a burst (lines 160, 347-351). `PasteBurst` keeps 3, holding the
    characters before it, and leaves the window out;
  * a burst is delivered as `tea.PasteMsg`, so every pane that handles
    bracketed paste handles it with no change;
  * the first `tea.PasteStartMsg` turns detection off for good, because
    the terminal has proven it brackets pastes.
* **Order is preserved.** When the filter must release more than one
  message (a held character and the key that ended its window), it returns
  the first and queues the rest in a FIFO. A private drain message, sent
  from a new goroutine through `send` so the event loop never blocks on
  itself, releases the next. While the FIFO is not empty, new input joins
  its end, so nothing overtakes it.
* **The workspace routes new mouse types.** `local` (`render.go:274`)
  gains one case: a `tea.MouseMsg` with a `WithMouse(tea.Mouse) tea.Msg`
  method is translated by calling it. `CoalescedWheelMsg` uses this, and so
  can any program's own mouse message.

### 7. `stream/glamourmd`: glamour as a `Renderer`

```go
func New(opts ...Option) *Renderer // WithStyle(ansi.StyleConfig), FromTheme(theme.Theme),
                                   // WithEmoji(bool), WithChromaFormatter(string)
func (r *Renderer) Render(src string, width int) (string, error)
```

* **One `glamour.TermRenderer` per width.** `WithWordWrap` fixes a
  renderer's width, so the adapter keeps a small LRU of renderers (four
  widths) under one mutex, because glamour is stateful (crush's evidence,
  §Context).
* **Styles come from the theme.** `FromTheme` maps `theme` roles onto a
  `glamour/v2/ansi.StyleConfig`: headings to Title, code and links to
  Accent, block quotes to Muted, and so on. Light, dark and unknown
  backgrounds follow the theme. Colour is degraded by the program's
  renderer or by `colorprofile`, so 0001 §6 rule 4 holds.
* **Glyphs.** List bullets, rules and quote bars are taken from `glyph`,
  including the ASCII set (rule 3), through the `StyleConfig` fields that
  set them. Which fields those are in v2.0.1 is checked in PLAN Step 7;
  any glyph glamour draws that no field controls is recorded there.

### 8. Versioning

This record ships after the records before it in the implementation order
of [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§7. It is additive, and it is released as a minor version. The owner names
the tag.

### Consequences

* Good, because a 200 KB transcript streamed in small chunks costs O(n)
  for its committed text plus a bounded tail per frame, not O(n) per
  token, and the final frame is exact.
* Good, because what is drawn does not depend on how the network chunked
  the stream, which makes the engine testable as a pure function.
* Good, because no model or tool output can reach the terminal as a
  control sequence, even split across chunks.
* Good, because the frame scheduler, the queue and the filter serve any
  animated or mouse-heavy program, not only Markdown.
* Good, because the renderer is an interface. A program can use glamour,
  `Plain`, or its own.
* Neutral, because relaxed cuts trade a seam that `Finish` removes for
  bounded work on long runs.
* Neutral, because a single huge table or HTML block is still re-rendered
  whole on each frame until it ends, as it is in codex.
* Bad, because glamour brings goldmark, chroma, bluemonday and
  `golang.org/x/text` into the module graph, and govulncheck must stay
  clean over them.
* Bad, because the scheduler asks the program to re-arm `Wait` on each
  frame. A program that forgets stops receiving frames. The guide and the
  example show the one line, and `WithSend` removes it.
* Bad, because paste-burst detection needs `Attach`, which takes the
  program's `Send` after the program is built.

### Confirmation

* **`safetext`:**
  * a table test covering every sequence class in §2, including codex's
    injection string (`ESC[2J`, OSC 52, raw C1);
  * `FuzzClean`: the output is valid UTF-8, holds no ESC, no C0 except as
    the policy allows, and no U+0080–U+009F; `Clean` is idempotent; and
    `Write(a)`, `Write(b)`, `Close` equals `Clean(a+b)` for any split;
  * a differential test against `ansi.Strip` over generated well-formed
    sequences: the printable text is the same once the controls
    `ansi.Strip` keeps are removed.
* **`frame`,** under `testing/synctest`:
  * a thousand requests in one frame deliver one `Msg`;
  * frames never come closer than 1/fps;
  * `RequestIn` delivers at its deadline;
  * `Close` releases a pending `Wait`;
  * no goroutine is left after `Close`.

  The policy is tested as a table of snapshots against codex's documented
  transitions, including hysteresis and the severe bypass.
* **`stream`:**
  * **Chunking invariance.** For random documents from a block grammar
    (paragraphs, headings, lists, fences, tables, quotes, HTML blocks,
    reference definitions), and random splits into chunks, the frame after
    the last chunk equals the frame from one `Append`.
  * **Finish exactness.** After `Finish`, `Render` equals a one-shot render
    byte for byte.
  * **Stated differences only.** Before `Finish`, for documents with no
    relaxed cut, the frame equals the one-shot render once runs of blank
    lines are collapsed. A counting fake renderer checks this; the glamour
    adapter is measured, and any construct where it fails is recorded.
  * **Cost.** A counting renderer proves two bounds, for prose with
    paragraphs, prose with none, one long code block and one long list:
    * each committed byte is rendered once per width;
    * each `Render` passes at most `RelaxAfter` plus one line of tail to the
      renderer.
  * Goldens across the 0001 §6 matrix for `Pane` mid-stream and finished.
* **`inputfilter`:** with an injected clock, 100 notches in 16 ms reach
  the model as one message with the full sum; direction changes reset it;
  the residual is flushed with `Attach`; `PasteBurst` passes codex's cases;
  and the FIFO keeps order under interleaved input.
* **Mutation proofs** for each package's key invariant, each seen failing
  on a scratch copy. Pre-add, `-race`, `LC_ALL=C` and the Windows host
  pass. CI is green on the push.

## Pros and Cons of the Options

### A. Four packages, a renderer-agnostic engine, a glamour adapter

* Good, because each concern is testable alone, and three of the four
  packages have no Markdown in them.
* Good, because only one subpackage carries glamour's dependencies.
* Bad, because it is the most code, and the cut scanner is a small
  CommonMark block parser that must be kept right.

### B. One package with glamour built in

* Good, because it is what crush ships, and it is the least code.
* Bad, because every user of streaming text takes glamour's graph.
* Bad, because crush's cut points depend on when a flush happens, so the
  drawn frame depends on chunking, and the engine cannot be tested as a
  function of its content.
* Bad, because the frame scheduler and the input filter would hide inside
  a Markdown package.

### C. Re-render everything each frame

* Good, because there is nothing to build.
* Bad, because Bubble Tea calls `View` after every message, so cost is
  O(n) per token. aider's adaptive delay hides this by updating less often,
  which makes long answers visibly slower.

### D. Inline scrollback

* Good, because finished text is in the terminal's own scrollback, which
  the user can search, select and keep after exit.
* Bad, because it needs the inline-mode record (candidate 6 in the
  report), which the owner did not pick, and it does not work inside a
  multi-pane alternate-screen workspace.
* Neutral, because option A's committed segments are exactly what D would
  print, so D can build on A later.

## Owner questions

*Answered 2026-10-02* (picked from options): Q1 "stream/glamourmd
subpackage"; Q2 "frame.Adaptive()"; Q3 "Opt-in WithPasteBurst"; Q4 "Silent;
Policy.Visible opts in"; Q5 "Kept by Text, dropped by Strict". Every answer
is the recommendation.

* **Q1. Where the glamour adapter lives.** Recommended: the subpackage
  `stream/glamourmd` in this module, adding `charm.land/glamour/v2` v2.0.1
  to `go.mod`. The alternatives are a nested module (its own tags and
  release process, a smaller graph for the root) or no adapter at all (each
  program writes the ten lines itself).
* **Q2. The default drain policy.** Recommended: `frame.Adaptive()`, with
  codex's numbers. The alternative is `frame.Immediate()`, which shows each
  frame's text at once and can jump when a burst arrives.
* **Q3. Paste-burst detection.** Recommended: opt-in, with
  `WithPasteBurst`, turning itself off at the first bracketed paste. The
  alternative is on by default whenever `Attach` is called.
* **Q4. Removed sequences.** Recommended: removed silently by default,
  with `Policy.Visible` for approval and audit views. The alternative is
  always visible, which shows the user an attempted injection but clutters
  ordinary output.
* **Q5. Bidi controls.** Recommended: kept by `Text`, dropped by `Strict`,
  which permission dialogs and command previews use. The alternative is to
  drop them everywhere, which breaks right-to-left prose.

## Amendments

### A1 (2026-10-02): second-pass findings

*Status: accepted (2026-10-02).* Its steps are A1.1 to A1.4 of
[0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md).

**Found.** A source-level pass over the Kilo, Grok Build, opencode and
codex TUIs
([0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§8) found problems this record's packages meet and do not yet answer. The
report's §11.1 lists them against this record. On 2026-10-02 the owner chose
to amend 0005, 0007 and this record with them before anything is built.

**What changes in the decision.** Each item adds to a section above. None
removes anything from it.

* **§2, `safetext` gains two presets and an interpreter.**
  * **`Command()`** is for commands and URLs shown for approval (report
    §8.12). It removes nothing silently. Each C0 and C1 control, DEL, and
    each bidi or format code point (U+200E/F, U+2028/9, U+202A–E,
    U+2066–9) is shown escaped, as `\xNN`, `\uNNNN`, `\n`, `\r` or `\t`,
    on one line. A command then cannot repaint the terminal, and a Trojan
    Source reorder cannot make the visible text differ from what runs. This
    is Kilo's `displayCommand`.
  * **`StatusLine()`** is for an external program's output in a status row
    (report §8.15). It keeps SGR and nothing else of CSI. It swallows DCS,
    SOS, PM, APC and two-character escapes, so `ESC ( B` from `tput sgr0`
    does not paint `(B`. It keeps OSC 8 only for http, https and mailto
    targets with a host. This is grok's status-line sanitizer. It is the
    one preset that keeps any sequence, and it is never the default.
  * `Policy` gains the fields these need: `Escape bool` (show controls
    escaped, instead of removing them), `KeepSGR bool`, and
    `LinkSchemes []string`. Their zero values keep §2's behaviour.
  * **`Interpret(raw []byte, opts InterpretOptions) []Row`** turns
    captured tool output into styled rows, where `Clean` would strip it
    (report §8.10). It handles SGR in both `;` and `:` sub-parameter forms,
    CR overwrite (so a progress bar rewrites its line), BS, tabs to
    multiples of 8, and `CSI K`, `J`, `A`, `B`, `C`, `D` and `G`. It drops
    every other sequence, including OSC. Each `Row` holds a styled and a
    plain form, the plain one for copy and search. (The type is `Row`, not
    `Line`, because `Line()` is already a preset.)
    * It advances by cell width with `x/ansi`, not one column per
      character. Grok's interpreter does the latter, which §8.10 records as
      a defect.
    * It caps rows and columns (defaults 50 000 and 8192, as in grok).
    * Each call starts from a fresh state, so its result is deterministic
      and can be cached.
* **§3, `frame` gains four types.**
  * **`Coalescer[T]`** batches mergeable items once per frame, and treats
    every other item as a barrier (report §8.17, Kilo's frame queue). A
    barrier first flushes the batch, then is applied, so a lifecycle event
    keeps its order relative to the deltas around it. A maximum wait
    covers a frame that never comes (Kilo uses 100 ms).

    ```go
    func NewCoalescer[T any](apply func([]T), mergeable func(T) bool, maxWait time.Duration) *Coalescer[T]
    func (c *Coalescer[T]) Push(v T) tea.Cmd // a tick command when one is newly needed
    func (c *Coalescer[T]) Flush()
    ```

  * **`Demand`** is how often a view needs ticks: `None`, `Slow` or `Fast`
    (report §8.17, grok's tick demand). `Combine` takes the maximum.
    `TickCmd(None)` returns nil, so an idle program wakes for nothing. A
    view that paints an animation reports which animations it painted, and
    a tick redraws only when one of those changes on that tick.
    *Added 2026-11-02, by
    [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
    D7:* a tick's message is `frame.TickMsg{At, Demand}`, and the
    message of a `Coalescer`'s deadline `frame.FlushMsg{At}`, on which a
    program calls `Flush`; both are `frame`'s, and `frame/frametea`'s
    `TickCmd` and `CoalesceCmd` return them.
  * **`Writer`** wraps the output the caller passes to `tea.WithOutput`
    (report §8.17). It keeps `queued` and `written` counters and allows one
    frame in flight, so draws during a stalled terminal collapse into one.
    It reports a stall once after a configurable time with no progress
    (grok uses 5 s), and once when writes resume. It writes only to the
    writer it wraps, so rule 1 holds.
  * **`Upgrader[K, R]`** runs an expensive render off the event loop and
    returns it as a message (report §8.17). Jobs are keyed, so the newest
    job for a key replaces an older one. Each result carries a generation,
    and a stale one is dropped. A cheap render is shown until the upgrade
    arrives.

    ```go
    func (u *Upgrader[K, R]) Submit(key K, gen uint64, fn func(context.Context) R) tea.Cmd
    type UpgradeMsg[K comparable, R any] struct{ Key K; Gen uint64; Result R }
    ```

* **§4, `stream` returns side-tables with its lines.**
  * **`Doc.View(width) RenderView`** joins `Render`. A `RenderView` holds
    the lines and, for each line, its source offset; the links, with an ID,
    a row and a column range; the code blocks, with their info string and
    source and output ranges; and table copy metadata (report §8.10,
    grok's streaming renderer). Each table is cut back and carried forward
    when the tail re-renders, as the lines are. This is what lets "copy
    code block", link hit-testing and selection work while a reply still
    streams. A `Renderer` that cannot report a table returns none, and
    `Doc` derives the line map itself.
  * **Link IDs continue across tail renders,** so the fragments of one
    wrapped link keep one ID.
  * **Model-output dialect.** Only `~~` is strikethrough, so `~**10%**`
    stays literal, as grok renders it. With math rendering enabled, inline
    holdback also holds an unclosed `\(` or `$$` at a chunk edge until
    `Finish`.
  * **`Highlighter`** is an interface `Doc` and `Plain` call for code
    blocks (report §8.10):

    ```go
    type Highlighter interface {
        Highlight(lang string, lines []string, state any) (out []string, next any)
    }
    ```

    * The open fence is highlighted resumably. The state after the last
      committed newline is kept, so each line is highlighted once, which is
      O(N) instead of O(N²). A highlighter that cannot resume returns a nil
      state, and only the uncommitted tail is highlighted again.
    * Closed fences in the tail are memoised by `(info, body)` under a byte
      budget, cleared at once when it overflows.
    * A theme revision or a width change drops both caches.
    * Limits, as in codex: over 512 KiB, 10 000 lines or 4 KiB in one line,
      the code is shown plain.
    * Only the first token of an info string selects the language.
    * No lexer ships with this record. A chroma adapter needs its own
      dependency record, which the owner allowed to be proposed on
      2026-10-02. glamour's own chroma use is unchanged.
  * **`FitTable`** lays out a closed table at a width, for `Plain` and for
    a program's own renderer (report §8.10). Columns are sized from their
    widest cell, then from word minimums, with extra space shared in
    proportion to want. Numbers and URLs are never broken. When the table
    does not fit, it falls back to key/value records, then to stacked
    records, then to the pipe source, as codex does. Each row is padded or
    clipped by grapheme with `x/ansi`, so a wide glyph cannot leave a ghost
    cell. Border glyphs come from `glyph`, with their ASCII twins, so
    `stream` gains an import of `glyph`, which is downward.
  * **`Bounded`** holds live tool output (report §8.10, codex's live
    output). It keeps everything up to a byte limit, then the first and
    last lines, with a per-line byte cap so output with no newlines stays
    bounded. An escape sequence cut off at the head is closed before the
    omitted-bytes marker. Truncation counts wrapped rows and reports the
    omitted count in logical lines, so the count does not change with
    width.
  * *Specified 2026-11-03, by
    [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
    D19:* the PLAN's Step A1.4 gives the API: a `Renderer` reports its
    side-tables as a `ViewRenderer`, `Plain` one; a link's ID is its
    source offset; the theme's revision reaches a `Doc` as a number,
    `Doc.SetRevision`, so that `stream` still imports no `theme`;
    `FitTable` takes `TableOptions`, its border a `glyph.Border` or the
    Markdown form, and ~~the `Printer` and~~ `Plain` ~~fit their tables~~
    fits its tables with it (*D20:* the `Printer`'s stay unwrapped, as
    [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)
    decided); `Bounded` keeps its head and tail in lines, its floor
    `safetext.StatusLine`, its omitted line `stream`'s first message,
    `MsgOmitted`.
* **§6, `inputfilter` becomes a chain of stages.**
  * **Fragment reassembly.** SGR mouse and focus reports split across reads
    are rejoined (report §8.5).
  * **X10 repair.** X10 mouse reports that ConPTY and WSL relays corrupt,
    for columns 95 and beyond, are re-encoded so they parse as one mouse
    event and not as a mouse event plus a typed character.
  * **Reply swallowing.** A probe reply that arrives after its deadline is
    removed, not delivered as keys. The matchers come from 0005's probes.
    `inputfilter` takes them as functions, so it does not import `termcap`.
  * **A bare Esc flushes** anything a stage holds, so Esc never waits.
  * **Typeahead.** The caller chooses per screen between `Capture`, which
    keeps real typing from a window (text, Backspace, Shift or Alt+Enter,
    pastes; cut at the first Esc) and replays it, and `Quarantine`, which
    drops all input until a deadline, before a security-sensitive screen.
    Neither is on by default (report §8.5, grok and codex).
  * **Paste normalisation.** CRLF and lone CR become LF, because ConPTY
    sends CR-only line ends. An empty bracketed paste becomes
    `ClipboardImageRequestMsg`, because Windows Terminal before 1.25 sends
    an image-only clipboard that way. `PastedPath(s, goos)` recognises a
    dropped or pasted path: quoted, `file://`, drive letters and UNC, with
    backslash escapes undone except on Windows (report §8.5).
  * **Paste-burst cases.** `PasteBurstConfig` gains grok's PowerShell
    rules as options: a shorter first gap, a longer run for path-shaped
    input on Windows, and Enter followed by Ctrl+J read as a pasted CRLF.
    Detection stays opt-in (Q3).
  * **Wheel profiles.** `WithWheelProfile(eventsPerNotch int)` takes the
    count a program derives from 0005's `Caps.Brand` and `Caps.Mux`: 1 for
    iTerm2, WezTerm and the
    xterm.js family, 3 for Apple Terminal, kitty, Ghostty and Alacritty,
    and 1 under tmux, screen and zellij (report §8.5).
  * *Amended 2026-11-02, by
    [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
    Step A1.3's measurement, D13 and D14:* Bubble Tea `v2.0.10` joins a
    report split within its 50 ms escape timeout and no further, decodes
    X10 by the byte, so ConPTY's re-encoding from column 96 misplaces a
    click and types its row, and keeps a paste's CR and CRLF; it does none
    of these stages. Reassembly begins only from an unknown head of three
    bytes or more, so that a bare Esc never waits: a report split after
    one or two bytes, and any focus report, is not rejoined. The stages'
    API is the PLAN's Step A1.3: `Report`, `ParseReport`, the
    `Reassembler` with a `ReplyMatcher`, `Typeahead`, `NormalizePaste`,
    `PastedPath`, grok's PowerShell rules as `PasteBurstConfig` fields,
    and the `Flusher`'s `Capture`, `Replay` and `Quarantine`; and Step
    4's `BurstAction` is `Action`, every stage's.

**What does not change.**

* Option A, the package set and the import direction. `inputfilter` still
  imports only Bubble Tea, and `safetext` only the standard library and
  `glyph`. The one new import is `stream` → `glyph`, for `FitTable`.
* The answers to Q1 to Q5. `Text`, `Strict` and `Line` keep their meaning.
  `Command` and `StatusLine` are new presets beside them.
* The one new module stays `charm.land/glamour/v2`. A1 adds none.
* Option D stays rejected for this record. On 2026-10-02 the owner chose a
  separate `inline` record for scrollback output (report §11.2, candidate
  12), which can build on `RenderView` and on the committed segments.
* Re-sending the mouse modes on focus-in, which some relays need (report
  §8.5), sets terminal modes, so it belongs to the planned `termmode`
  record, not to this one.

**Versioning.** A1 is additive. If it is accepted before this record's
release, it ships in that minor. Otherwise it ships in the next minor.

**Owner questions for A1.**

*Answered 2026-10-02* (picked from options): Q6 "safetext"; Q7
"frame"; Q8 "stream, now". Every answer is the recommendation, so the
text above stands, and A1 is accepted.

* **Q6. Where the escape interpreter lives.** Recommended: `safetext`, as
  `Interpret`, because it shares the sanitizer's parser and states. The
  alternative is its own package, which keeps `safetext` to one job.
* **Q7. Where `Writer` and `Upgrader` live.** Recommended: `frame`, beside
  the scheduler they pace. The alternative is a new `termio` package for
  `Writer`, which `frame` would not need to import.
* **Q8. Where `FitTable` lives.** Recommended: `stream`, now, so `Plain`
  can lay out tables. The alternative is to wait for the `table` record
  (report §11.2, candidate 14), and leave `Plain` showing a table's
  source until then.

### A2 (2026-10-02): the glamour adapter as a nested module

*Status: accepted (2026-10-03).* Its plan is Step 7 of
[0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md),
as revised on 2026-10-02.

**Found.** On 2026-10-02 the owner decided:

> Nested modules. Glamour as nested module. Go.work in repo.

That supersedes the answer to Q1 ("stream/glamourmd subpackage"), which
put glamour in this module's `go.mod`.
[0010-REPORT-nested-modules-and-adapter-sources.md](../reports/0010-REPORT-nested-modules-and-adapter-sources.md)
§2 measured why it matters: a requirement of the root module reaches every
consumer's `go.sum` and `go list -m all`, even a consumer that imports none
of the packages using it. §1's "a program that never imports that
subpackage compiles none of glamour" is true, and is not enough.
[0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
decides the layout, the `go.work`, the release order and the gates. A
source read of glamour v2.0.1 (REPORT §9) also corrects and sharpens §7.

**What changes in the decision.** §1's package table, §1's "One new module"
bullet, §7's location and the Q1 answer are superseded as follows. Their
text above is kept as it was decided.

* **The module.** `stream/glamourmd` is the nested module
  `github.com/maccavelli/go-tui-lib/stream/glamourmd`, package `glamourmd`,
  with its own `go.mod`.
  * It requires `charm.land/glamour/v2` v2.0.1, the newest release on
    2026-10-02 (REPORT §9), and a published version of the root module, with
    no `replace` (0010-MADR §3).
  * The root module's `go.mod` never requires glamour. A program that does
    not `go get` the adapter has no glamour, goldmark, chroma or bluemonday
    in its build, its `go.sum` or its module graph.
  * Tags are `stream/glamourmd/vX.Y.Z`, starting at `v0.1.0`
    (0010-MADR §1, §3).
  * It imports only the root's exported packages (`stream`, `theme`,
    `glyph`), never `internal/`.
* **glamour is pure, so the adapter owns style.** v2 removed
  `WithAutoStyle` and `WithColorProfile`; it never queries the terminal
  (REPORT §9).
  * `FromTheme` builds the whole `ansi.StyleConfig` from the library's
    theme and glyph table, ASCII twins included, and passes it with
    `WithStyles`.
  * The adapter never uses `WithEnvironmentConfig`, which reads
    `GLAMOUR_STYLE`, or `WithStylePath` and `WithStylesFromJSONFile`, which
    read files. A program that wants one of glamour's built-in styles passes
    it through `WithStyle`.
  * Colour downsampling stays the caller's, through `colorprofile`, as §7
    already says. glamour emits hex and 256-colour values as given, and its
    code blocks use chroma's `terminal256` formatter by default
    (REPORT §9).
* **Code-block colours never touch chroma's global registry.** A style with
  `CodeBlock.Chroma` set registers a chroma style named `charm` once per
  process, and the first registration wins, so a later theme's code would
  keep the first theme's colours (REPORT §9). The adapter sets
  `CodeBlock.Chroma` to nil and `CodeBlock.Theme` to a built-in chroma style
  name that `FromTheme` chooses for the theme's background. The built-in
  styles that set `CodeBlock.Chroma` (dark, light, dracula, tokyo-night)
  are copied with that field cleared before use.
* **The renderer cache is keyed by width and theme.** A `TermRenderer` is
  not safe for concurrent use: every render mutates one shared
  `RenderContext` (REPORT §9). §7's four-width LRU becomes an LRU keyed by
  (width, theme revision, emoji), four entries by default, each renderer
  behind its own lock. `WithWordWrap` still fixes a renderer's width at
  construction.
* **The pane owns padding.** `Document.Margin` is 0 in every style the
  adapter builds; glamour's built-in styles use 2 (REPORT §9). §4's trim of
  leading and trailing blank lines stays.
* **Tables use the glyph table.** The adapter sets the table's row, column
  and centre separators from `glyph`, so an ASCII set gives an ASCII table;
  otherwise glamour draws `lipgloss.NormalBorder()` (REPORT §9). It always
  sets `CenterSeparator` when it sets `Row` and `Column`, because glamour
  dereferences it without a nil check in that case (`ansi/table.go:116`).
* **The rules hold** (REPORT §10): glamour writes nothing, queries nothing,
  and reads the environment only through the options the adapter never
  uses.
* **The engine's properties are exported for any renderer.** A nested
  module sees only `stream`'s exported API, so the property checks the
  PLAN runs on `Plain` move to a test-helper package in the root module,
  `stream/streamtest`, as `net/http/httptest` serves `net/http`:

  ```go
  // CheckRenderer streams generated documents through a Doc built on r,
  // in random chunks, and reports chunking invariance and Finish
  // exactness through t. It uses only stream's exported API.
  func CheckRenderer(t testing.TB, r stream.Renderer, o ...Option)
  ```

  `stream`'s own tests call it with `Plain`, and the glamour module's tests
  call it with its renderer. A program can check its own renderer the same
  way. It is test-only code that imports `testing`, so no program
  compiles it unless its tests import it.

**What does not change.**

* The `Renderer` interface in `stream`, the stable-prefix engine, `Plain`,
  and every other package of §1.
* Amendment A1.
* The API of §7: `New`, `Render`, `WithStyle`, `FromTheme`, `WithEmoji`,
  `WithChromaFormatter`.

**Versioning.** The root release that carries `stream` comes first. The
adapter's `go.mod` then requires that release, and the owner tags
`stream/glamourmd/v0.1.0` after it, in the order 0010-MADR §3 sets. The
root release itself gains no requirement.

### A3 (2026-10-07): moved lines, the width method's source, a met precondition, and the theme revision

*Status: accepted (2026-10-07)* for the corrections. The theme revision's
source is owner question Q9, open until a step keys a cache by it. Found
by the documentation audit of
[0011-MADR-docs-accuracy-after-v0-5-0.md](0011-MADR-docs-accuracy-after-v0-5-0.md),
carried out by [0011-PLAN-docs-accuracy-after-v0-5-0.md](0011-PLAN-docs-accuracy-after-v0-5-0.md), finding F33. Each fact was re-checked against the tree
on 2026-10-07.

**Found.**

1. **`local` has moved.** It is at `workspace/render.go:460-478`. The
   Context cites `:274-291`, and §6 cites `:274`.
2. **The width method's source.** More Information credits the
   width-method alignment to
   [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md).
   It came with
   [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
   §3, as
   [0008-MADR-command-palette.md](0008-MADR-command-palette.md) A1 also
   records.
3. **Step 7's precondition is met.**
   * Step 7 of [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
     waits for the tooling phases of
     [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md),
     which is complete.
   * [0012-MADR-bring-your-own-cli.md](0012-MADR-bring-your-own-cli.md)
     has since retired the two command adapters. `stream/glamourmd`
     stays planned under 0010, with its depguard rule.
4. **No "theme revision" exists.** A1 and A2 key caches by one, but
   `theme.Theme` has none. The workspace keeps an unexported generation,
   raised on every theme change (`workspace/workspace.go:144`, `:310`).

**Corrected.** Items 1 to 3, as above.

**Owner question Q9.** Open until a step keys a cache by it: where the
theme revision comes from.

* **A.** `workspace` exports its generation, and gives it to panes, for
  example in `SizeMsg` or in a message of its own.
* **B.** `theme.Theme` gains a comparable fingerprint of its palette and
  profile.
* **C.** `stream` compares the theme it was last given with the new one,
  and drops its caches on a change, with no revision at all.

### A4 (2026-10-09): names from the glossary

Names only; the design stands. Before this record is built, its types take
the names [docs/glossary.md](../glossary.md) gives, so that none collides
with a type the library exports or another record plans
([0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md) Step 2). Where this record says the old name, read the new one.

| Was | Is | Why |
| :--- | :--- | :--- |
| `safetext.Policy` | `safetext.Rules` | `termsvc.Policy`, and `frame.Policy` |
| `safetext.Filter` | `safetext.Sanitizer` | `inputfilter.Filter`; `Filter` is a message filter |
| `frame.Policy`, its `Decide` | `frame.Pacer`, its `Pace` | as above |
| `frame.Plan` | `frame.Release` | `layout.Plan` |
| `Plan.Mode` | `Release.Gear` | `command.Mode` |
| `frame.New` | `frame.NewScheduler` | `New` builds only the type named after its package |
| `frame.Msg` | ~~`frame.FrameMsg`~~ `frame.DueMsg` (*2026-11-02, [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md) D6: `FrameMsg` stutters, which revive's `exported` rule refuses; where this record says `FrameMsg`, read `DueMsg`*) | a message type is named for what it carries |
| `inputfilter.New` | `inputfilter.NewFilter` | as `frame.New` |
| `stream.New` | `stream.NewDoc` | as `frame.New` |
| `stream.Pane` | `stream.DocPane` | `layout.Pane` and `workspace.Pane` |
| glamourmd's `New`, `FromTheme` | `NewRenderer`, `WithTheme` | as `frame.New`; options are named `With…` |
| `Highlighter` | with a `HighlighterFunc` adapter | AGENTS.md's API rule for hooks |

* **`Renderer`** in `stream` and in `stream/glamourmd` stays: the
  adapter's type implements the interface. The glossary records the pair
  as the one planned deliberate collision, added to "Deliberate" and to
  `sharedNames` when the adapter is built.
* **`safetext` and `internal/sanitize`.** 0014-PLAN-hardening built
  `internal/sanitize`, which `workspace`, `termsvc`, `termcap` and
  `command` use for untrusted display text. `safetext`'s PLAN decides
  whether it wraps `internal/sanitize` or replaces it, so that the
  library keeps one sanitizer.

[0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
takes these names when it is next amended, before it runs.

### A5 (2026-10-28): `stream` becomes a Charm-free core

*Status: accepted (2026-10-28),* as
[0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)
D7, which the owner accepted, decides, with its A6 item 2:

1. **`stream` is a core** (0020-MADR D1, D11): `Doc`, `Renderer`,
   `RendererFunc`, `Plain` and the block scanner import `safetext`,
   `internal/enum`, `internal/sanitize` and `x/ansi` alone. What names
   Bubble Tea or `frame`, `Buffer`, `Doc.Pull` and `Pane`, moves to
   `stream/streamtea`, by 0020-MADR D2's split rule. §1's table, §4's
   `Doc.Pull` and §5's `Pane` read so. Nothing of `stream` is shipped, so
   nothing breaks.
2. **One block scanner, two consumers.** §4's cut scanner is also the
   line mode's: 0023's `stream.Printer`, line-stable and append-only,
   built first (0023-PLAN Step 5), and `Doc`, block-stable, re-rendered by
   a `Renderer`. The scanner is unexported until `Doc` is built on it,
   when this record's PLAN exports it as `Scanner` (0023-MADR A6 item 2).
3. **`stream` gains `Printer`, `Markup` and `Style`,** 0023-MADR D7's,
   beside `Doc`.

### A6 (2026-11-01): what is built since, the facts re-read, and four questions answered

*Status: accepted (2026-11-01),* every question answered with its
recommendation, as A1 was accepted. Proposed the same day, after the
owner asked to "assess the 0009
madr and plan pair. evaluate the current state of the codebase and
environment. ensure the madr and plan are still accurate, grounded in
facts, and comprehensive." The decision, option A, stands. This amendment
records what other records have built of it, corrects the facts that have
moved, applies the conventions written after it, and answers A3's Q9
and three questions of its own. Every fact below was re-read at `114bee8` on 2026-11-01.

1. **Built since, by other records.**

   | Of this record | Built by | Released |
   | :--- | :--- | :--- |
   | §2 `safetext`, with A1's `Command()` and `StatusLine()`, the fields `Escape`, `KeepSGR` and `LinkSchemes`, and A4's names; `glyph.Set.Control` | [0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md) D9, which also added `IsClean` and makes `internal/sanitize` wrap it | `v1.13.0` |
   | A5's split: `stream` a Charm-free core, in `TestCoresAreCharmFree` | [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md) D7 | `v2.6.0` |
   | `stream`'s `Printer`, `Markup` and `Style`, beside §4 | 0023-MADR D7, A6 to A10 | `v2.6.0` to `v2.6.2` |
   | Option D's scrollback, as a line mode: `inline`, `transcript` | 0023-MADR D3, D9, A14 | `v2.3.0` to `v2.9.0` |
   | §2's "link service" of 0005 | `termsvc.Linker`, [0024-MADR-opt-in-and-opt-out-coverage.md](0024-MADR-opt-in-and-opt-out-coverage.md) A3 | `v2.8.0` |

   **Not built:** `safetext.Interpret` (A1); `frame` and its A1 types;
   `inputfilter` and its A1 stages; `stream`'s `Doc`, `Renderer`,
   `RendererFunc`, `Plain`, `RenderView`, `Highlighter`, `FitTable` and
   `Bounded`; `stream/streamtea`'s `Buffer`, `Doc.Pull` and `DocPane`
   (A5); `stream/streamtest` (A2); `workspace`'s mouse extension point
   (§6); and `stream/glamourmd` (A2).
2. **The facts, re-read.**
   * **Bubble Tea, at the tree's `v2.0.10`:** every line §Context cites
     holds: `tea.go:888` renders after each message; `:764-768` applies the
     filter; `:606` makes the unbuffered `msgs` channel `Send`
     (`:1192-1197`) writes to; `renderer.go:13-14` is 60 and 120 fps;
     `mouse.go:113` is `MouseWheelMsg`, a `Mouse` with no magnitude.
     `v2.1.0` and `v2.1.1` have been released since. `v2.1.1` keeps each of
     these (`options.go:140` `WithFilter`, `:149` `WithFPS`, `renderer.go:13-14`,
     `tea.go:620` the channel), and coalesces no wheel or motion event. The
     tree is not moved to it here.
   * **`x/ansi`:** the tree's `v0.11.8`; `v0.11.9` is released.
   * **glamour:** `charm.land/glamour/v2` `v2.0.1` is still the newest
     release; the proxy lists `v2.0.0` and `v2.0.1`.
   * **`local`** has moved again: `workspace/render.go:538-556`, after A3
     item 1's `:460-478`. It still translates the four `tea` mouse types
     and drops every other `tea.MouseMsg`, so §6's extension point is still
     needed.
   * **The theme's revision** (Q9): `theme.Theme` has none. The workspace's
     unexported `themeGen` (`workspace/workspace.go:173`) is raised by
     `SetTheme` (`:472`) and keys its view cache (`render.go:152`).
   * **The wheel profiles' inputs** exist: `termcap.Caps.Mux`
     (`termcap/caps.go:25`) and `Caps.Brand` (`:48`).
   * **Split input is partly joined already.** ultraviolet's
     `TerminalReader`, which Bubble Tea v2 reads through, holds an
     incomplete sequence until its `EscTimeout`, 50 ms by default
     (`terminal_reader.go:27`, `:143-180`), and assembles a bracketed
     paste's text (`:288-310`). A1's fragment reassembly matters only for a
     report split by more than that; the X10 repair still stands, since
     X10 is decoded as it comes (`decoder.go:499-503`, `:1631`). Step A1.3
     measures both before it builds them.
   * **The repository's tools,** since this record: the toolchain floor is
     Go 1.27.2 ([0016-MADR-go-1-27-2-for-go-2026-6604.md](0016-MADR-go-1-27-2-for-go-2026-6604.md));
     the scripts are Python, `scripts/go-precheck.py` with a shell shim
     ([0017-MADR-python-repository-scripts.md](0017-MADR-python-repository-scripts.md));
     a step is checked with `python3 -m tools.verify`
     ([0018-MADR-python-verification-tools.md](0018-MADR-python-verification-tools.md));
     `make fuzz` finds every fuzz target itself; and `go.work` lists `.`
     alone, until the adapter joins it.
3. **A name collides.** A1's `safetext.Row`, what `Interpret` returns, is
   `ask.Row`'s name, exported since `v2.3.0` (0023). `python3 -m
   tools.verify names` over this record's 37 planned type names, with A4's,
   A5's and A6's, reports "names: safetext.Row: also in ask" and no other; the
   conformance test would fail on it. Q11 asks for the name.
4. **The API conventions written since apply** (AGENTS.md, "API
   conventions"; [0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md) W0), as they apply to every
   exported name:
   * **Hooks** (rule 7): `frame.Pacer` gains `PacerFunc`, as A4 gave
     `Highlighter` its `HighlighterFunc`; `Renderer` has `RendererFunc`.
   * **Options** (rule 4): a `Without…` option turns a default off, so
     §4's `WithInlineHoldback(bool)` and `WithPartialLine(bool)` are
     `WithoutInlineHoldback()` and `WithoutPartialLine()`; §5's
     `WithFollow(bool)` is `WithoutFollow()`; §6's `WithWheel(bool)` and
     `WithMotion(bool)` are `WithoutWheelCoalescing()` and
     `WithoutMotionThrottle()`; and §4's `WithPolicy` is `WithRules`, as
     A4 renamed the type. Option types are opaque.
   * **`stream` has one `Option`.** `Printer`'s `stream.Option`, with its
     `WithRules` and `WithMethod`, is `Doc`'s too: its unexported method
     set grows to apply to a `Doc`, and only the package makes one. `make
     apicheck` confirms the change is compatible in the step that makes it.
   * **Enums** (rule 5): `Gear`, `Demand` and `BurstAction` take their
     tokens from `internal/enum`.
   * **Defaults** (rule 12,
     [0024-MADR-opt-in-and-opt-out-coverage.md](0024-MADR-opt-in-and-opt-out-coverage.md)
     D1, D8): each default this record applies, with its control, is a row
     of `docs/guides/defaults.md`, `planned` until built:
     * `frame`: 60 fps (`WithFPS`); the adaptive gears' thresholds (each
       an `AdaptiveOption`); `Adaptive` as the default pacer (Q2; a
       program passes `Immediate`); the coalescer's maximum wait (Kilo's is 100 ms);
       `Writer`'s stall time (grok's is 5 s);
     * `inputfilter`: wheel coalescing and the motion throttle on, at
       16 ms (`WithoutWheelCoalescing`, `WithoutMotionThrottle`,
       `WithInterval`); paste bursts off (`WithPasteBurst`), with codex's
       gaps; typeahead off; the events a notch (`WithWheelProfile`);
     * `stream`: `RelaxAfter` 2 KiB; inline holdback and the partial line
       on; `safetext.Text`, never less; `Plain`'s width method; the
       highlighter's limits; `Bounded`'s limits;
     * `safetext.Interpret`: its 50,000 rows and 8,192 columns;
     * `stream/streamtea`: following the tail on (`WithoutFollow`); the
       pane's keys (`PaneKeys`);
     * `stream/glamourmd`: four cached renderers; emoji (`WithEmoji`).
   * **Text a user is shown** comes from the program's catalogue
     ([0024-MADR](0024-MADR-opt-in-and-opt-out-coverage.md) D9, `msgcat`):
     `Bounded`'s omitted-lines marker, and any text `DocPane` draws, have
     message IDs and English defaults, and a `WithCatalog`.
   * **The environment** (rule 9): `inputfilter`'s Windows timing is read
     from a `WithGOOS` option, as `termcap`'s is, defaulting to the
     platform the program was built for.
5. **The block scanner, as built, is classifiers.** `stream/scan.go` holds
   line classifiers (`fenceRun`, `closesFence`, `htmlStart`, `htmlEnds`,
   `definition`, `container` and others); the state §4 keeps, the open
   fence, the HTML block, the table, the paragraph and its lazy
   continuation, is the `Printer`'s own fields. Exporting `Scanner`
   (0023-MADR A6 item 2) moves that state into it, and builds the `Printer`
   on it, every `Printer` test, golden and fuzz target unchanged.
6. **Logic is not written twice.** `Plain` wraps with `stream`'s
   `wrapProse`, which the `Printer` wraps with, and A1's `FitTable` starts
   from its `layoutTable`, adding the fallbacks A1 names.
7. **Owner questions.** *Q9, A3's, is asked again here,* with its options
   re-read: `theme.Theme` cannot be compared, since it holds
   `theme.Styles` ("invalid operation: theme.Theme{} == theme.Theme{}
   (struct containing theme.Styles cannot be compared)", compiled in a
   scratch module against the tree), so A3's option C, a theme compared
   with the last, would need a deep comparison, and is replaced below.

   * **Q9. What keys a cache to a theme.** Recommended: `theme.New`
     stamps each theme it builds with an unexported identity, read by a
     method such as `Theme.ID() uint64`; a changed theme is always a new
     ID, and a rebuilt equal theme costs one render more. It works with or
     without a workspace, and `stream`'s core never sees `theme`.
     Alternatives: the workspace exports its `themeGen` and gives it to
     panes, which a pane outside a workspace lacks; or `theme` gains a
     fingerprint of its comparable parts, the costliest, which grows with
     `Theme`.

   * **Q10. `frame` and `inputfilter`: Bubble Tea packages, or cores with
     adapters?** §1 makes each a Bubble Tea package. 0020-MADR D2's split
     rule, which 0023 applied to `stream`, would make each a Charm-free core
     with a `<pkg>tea` adapter: the pacer, the queue, the gears, the
     coalescer and `Demand` in `frame`, with the scheduler's `tea.Cmd`, the
     frame message's delivery, `TickCmd` and `Upgrader`'s command in
     `frame/frametea`; the paste-burst machine over `termkey.Input` and the
     wheel sum in `inputfilter`, with the `tea.WithFilter` function and
     `CoalescedWheelMsg` in `inputfilter/inputfiltertea`.
     * **Recommended: cores with adapters.** A line-mode host paces a
       streamed reply and detects a paste on a terminal without bracketed
       paste with no Bubble Tea, as `lineedit` and `transcript` work; it is
       the library's direction since 0020; and the adapters call the cores,
       so nothing is written twice. Bad, because there are two more
       packages.
     * **As accepted:** two Bubble Tea packages. Fewer packages; a line-mode
       host writes its own pacing and paste detection.
   * **Q11. The name of `Interpret`'s row** (item 3). Recommended:
     `safetext.InterpretRow`, beside `InterpretOptions`. Alternative:
     `safetext.StyledRow`.
   * **Q12. The order of what is left.** Recommended: `Interpret` with
     Q11's name; `frame`; `inputfilter` and the workspace's extension
     point; `stream`'s `Scanner` and `Doc`, with `streamtest`;
     `stream/streamtea`; A1's `stream` additions; the guide and the root
     release; then the adapter. Alternative: `stream`'s `Doc` and
     `DocPane` first, the TUI's pane being pi-go's need, with `frame` and
     `inputfilter` after.
8. **Answered 2026-11-01** (picked from options): Q9 "theme.New stamps an
   ID (Recommended)"; Q10 "Both cores + adapters (Recommended)"; Q11
   "safetext.InterpretRow (Recommended)"; Q12 "Foundations first
   (Recommended)". So:
   * **`frame` and `inputfilter` are cores,** each with its `<pkg>tea`
     adapter, by 0020-MADR D2's rule: a declaration whose signature,
     fields or methods name a Charm type moves to the adapter, and every
     effect gets a core form the adapter wraps. ~~The cores take time in
     and give a deadline out, with no clock and no goroutine, as the line
     mode's engines do (0023-MADR D2): the scheduler's timer and
     `Upgrader`'s jobs are `frame/frametea`'s.~~ Each core joins
     `TestCoresAreCharmFree` in the step that builds it. §1's import
     table reads so.
   * ***Corrected 2026-11-01,*** the owner's "yes, correct the records to
     reflect the optimizations", after the agent set out the trade-off.
     The struck sentence was the agent's, not Q10's answer: the option
     chosen moved only the scheduler's `tea.Cmd`, the frame message's
     delivery, `TickCmd` and `Upgrader`'s command to the adapter.
     0020-MADR D1's boundary is a package's imports, and a core may start a
     goroutine. Each core has two layers, the second built on the first:
     * **Pure,** time in and a decision or deadline out, for a host that
       drives its own deadline loop, as `lineedit`'s and `transcript`'s
       hosts do: `frame`'s deadline and frame-rate arithmetic, `Queue`,
       `Pacer` and `PacerFunc`, `Adaptive`, `Immediate`, the gears, the
       `Coalescer`'s batching and `Demand`; `inputfilter`'s paste-burst
       machine over `termkey.Input`, the wheel sum and the motion throttle.
     * **Driven,** with the standard library's timers and goroutines:
       `frame`'s `Scheduler`, one `time.Timer` whose `Request` and
       `RequestIn` are safe for concurrent use, due frames merged into a
       channel of capacity one, and `Close`; beside it the `Coalescer`'s
       maximum wait, `Writer`'s stall detection and `Upgrader`'s jobs; and
       `inputfilter`'s driver, which emits its timed flushes, a wheel sum's
       residual and a paste burst's idle end, on a channel. A line-mode
       host selects on the channel beside its input; a stream's producers
       request frames from their own goroutines.
     * **The adapters convert, and hold no logic:** `frame/frametea` turns
       the scheduler's channel into a `tea.Cmd`, delivers `FrameMsg`, and
       holds `TickCmd` and `Upgrader`'s command;
       `inputfilter/inputfiltertea` holds the `tea.WithFilter` function,
       `CoalescedWheelMsg`, and `Attach`, which gives the driver's flushes
       to the program's `Send`.
     * **Guards:** no goroutine runs without work (§3); under
       `testing/synctest`, which Go 1.27.2 has, no goroutine is left after
       `Close`, and each driver's timing holds; a `FrameMsg` carries the
       time it fired, `At`, which is the `now` the `Pacer` is given, so the
       scheduler's clock and the host's agree; and each goroutine or timer
       a driver starts for the program is a row of
       `docs/guides/defaults.md`, its controls not building the driver and
       `Close` (AGENTS.md, API convention 12).
     * **Why:** producers on many goroutines, as a stream's network
       readers are, need one merged frame, and a line-mode host gets the
       scheduler, `Writer`'s stall detection and `Upgrader` with no Bubble
       Tea; one implementation serves both; and `synctest` makes the timers
       testable. The cost: a lifecycle to close, a mutex to keep race-free,
       and a second pattern beside the deadline-out engines.
   * **`Interpret` returns `[]InterpretRow`.**
   * **The order** is Q12's recommendation, which the PLAN's revised order
     follows.
   * **`theme.Theme` gains `ID`,** stamped by `theme.New`, before the first
     cache keyed by it; A1's and A2's "theme revision" is that ID. A3's Q9
     is closed.
9. **What does not change:** option A; the decision drivers; §2's grammar,
   as built; §3's scheduler, queue and gears; §4's cut rules, costs and
   stated differences; §6's filter; A1's additions but for Q11's name; A2's
   nested module; A4's names; A5's split.
10. **Versioning.** Each part is additive, released in a minor version of
   the root, `v2.x`. The adapter requires a published root release, and is
   tagged `stream/glamourmd/v0.1.0` after it (0010-MADR §3).

### A7 (2026-11-01): `Interpret` and the narrow closure, the pane's keys and theme, and the facts the PLAN builds on

*Status: accepted (2026-11-01),* Q13 answered with its recommendation
(item 13). Proposed the same day, after the owner asked again to
"analyze the codebase and environment, ensure the madr and plan are
accurate, up-to-date, grounded in current facts and research", and that
"the plan is actionable, detailed, and adherent to idiomatic go and best
practices and standards". It reads A6 as committed (`6365db5`), and asks
one question. Every fact was re-read on 2026-11-01.

1. **`Interpret` cannot live in `safetext`.** A1 put it there (Q6), and
   needs `x/ansi` to advance by cell width. `safetext` is a narrow core:
   it may import the standard library, `glyph` and `internal/enum` alone
   (AGENTS.md, "Dependencies";
   [0021-MADR-termkey-closure-gate.md](0021-MADR-termkey-closure-gate.md)
   A1), held by the depguard rule `safetextstd` (`.golangci.yml:165`) and
   by `TestNarrowClosures`, whose list allows `safetext` "glyph" and
   "internal/enum" only (`internal/conformance/charmfree_test.go:210`).
   0020-MADR D9 left `Interpret` out of `safetext` for that reason. Q13
   asks where it goes.
2. **The pane's keys are the keymap's.** §5's `PaneKeys`, `WithKeys` and
   `Keys() []key.Binding` predate
   [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) and 0008-MADR
   A3's keys as commands. Every pane built since binds its keys so:
   `palette` exports its command IDs and its scope, `PickerScope`
   ("kind:picker"), and `DefaultRules() []keymap.Default`, and the picker
   runs its commands as a `workspace.CommandConsumer`
   (`palette/keys.go`). So `DocPane`:
   * exports its command IDs (`streamtea.ScrollUpID`, `ScrollDownID`,
     `PageUpID`, `PageDownID`, `TopID`, `BottomID`) and its scope,
     `DocPaneScope` ("kind:docpane"), which it names as a
     `workspace.KeyContexter`;
   * gives `DefaultRules() []keymap.Default`, for the program's one keymap
     beside `workspace.DefaultRules()`: up, down, pgup, pgdown, home and
     end, none of them ctrl+c;
   * runs those commands itself, as a `workspace.CommandConsumer`.

   §5's `PaneKeys` and `WithKeys`, and A6 item 4's "the pane's keys
   (`PaneKeys`)", read so. A program rebinds or removes a key in its
   keymap, which is the control.
3. **The pane reads its workspace, as the panes built since do.**
   `NewDocPane(w *workspace.Workspace, d *stream.Doc, o ...Option)`, and
   at each draw it reads `w.Theme()` and `w.WidthMethod()`, as `palette`'s
   picker (`palette/view.go:23`, `:45`) and `lineedittea`'s
   `ComposerPane` (`lineedit/lineedittea/pane.go:47`) do. No message
   carries the theme: the workspace has none, and `Theme()` is its
   accessor. §5's "measures with the workspace's width method" holds,
   through `WidthMethod()`.
4. **`theme.Theme.ID`, A6's Q9, has two edges** the PLAN must hold:
   * `Theme`'s fields are exported, so a program can change a copy after
     `New`; the copy keeps its ID. `Theme`'s documentation says to build
     a changed theme with `New`, and `ID`'s says what a copy keeps.
   * A `Theme` built as a literal, not by `New`, has ID 0. A cache keyed
     by a theme never reuses an entry under ID 0, so a hand-built theme
     always draws right, at the cost of its cache.
5. **glamour, re-read at `v2.0.1`** (`charm.land/glamour/v2`, MIT): every
   fact A2 relies on holds. `WithWordWrap` (`glamour.go:173`) fixes a
   renderer's width; `WithStyles` (`:146`) takes the `StyleConfig`;
   `WithEmoji()` (`:208`) takes no argument, and `WithChromaFormatter`
   (`:216`) a name; `WithEnvironmentConfig` (`:122`), `WithStylePath`
   (`:129`) and `WithStylesFromJSONFile` (`:162`) are the options the
   adapter never uses. `ansi/style.go` has `Document`'s `Margin *uint`
   (`:72`), `CodeBlock`'s `Theme string` and `Chroma *Chroma` (`:78-79`),
   and the table's `CenterSeparator`, `ColumnSeparator` and
   `RowSeparator` (`:91-93`); `ansi/table.go:116` dereferences
   `CenterSeparator` with no nil check; `ansi/codeblock.go:15` names the
   chroma style "charm", which `:90` registers. Its `go.mod` states
   `go 1.25.8` and requires lipgloss `v2.0.4`, `x/ansi` `v0.11.7`,
   chroma `v2.14.0`, goldmark `v1.7.8` and an ultraviolet of 2025-12-05;
   the adapter's module selects the root's newer lipgloss `v2.0.6`,
   `x/ansi` `v0.11.8` and ultraviolet, so its tests run at those.
6. **`Doc` shares `stream.Option`, compatibly.** `apidiff`, at the
   `Makefile`'s pin, over a package whose opaque `Option` interface gains
   a second unexported method, reports "Compatible changes: - Doc: added"
   and nothing incompatible. A6 item 4's "`make apicheck` confirms" is so
   confirmed; the step still runs it.
7. **Go and the standards.** The toolchain on the development host is Go
   1.27.3; the floor stays 1.27.2. `testing/synctest` (`Test`, `Wait`,
   `Sleep`), `sync.WaitGroup.Go` and `iter` are in it, so the drivers of
   A6 item 8 are tested in a bubble with fake time, and a goroutine is
   started through a `WaitGroup` that `Close` waits on. `ansi.Method`'s
   zero value is `ansi.WcWidth`, so an options struct with a `Method`
   field measures as the terminal does by default. CommonMark 0.31.2, of
   2024-01-28, is still the newest version of the specification
   (<https://spec.commonmark.org/>), so §4's cut rules cite it unchanged.
8. **Owner question.**
   * **Q13. Where `Interpret` lives.**
     * **Recommended: `stream`,** as `stream.Interpret`,
       `stream.InterpretOptions` and `stream.InterpretRow` (Q11's name
       kept). `stream` is a core that imports `x/ansi` already, and A1's
       `Bounded`, which holds live tool output, is there too. Bad, because
       `stream` was a Markdown package, and becomes the package for
       streamed content of both kinds.
     * **Widen `safetext`'s rule** to admit `x/ansi`, by amending 0021-MADR
       A1, `safetextstd` and `TestNarrowClosures`. Keeps A1's place; every
       program that cleans a string then links `x/ansi` and its four
       modules, which the narrow rule exists to prevent.
     * **A package of its own,** a core named for it, such as `termtext`,
       allowed `x/ansi`. Keeps both packages to one job; one more package.
9. **`Buffer` and `Doc.Pull` stay in `stream`.** A5 moved them to
   `stream/streamtea` because they name `frame`, which §1 made a Bubble Tea
   package. A6 item 8 makes `frame` a core, so they name no Charm type,
   and 0020-MADR D2 keeps such a declaration in its core: `Buffer`,
   `NewBuffer(*frame.Scheduler)` and `Doc.Pull(*Buffer, frame.Pacer,
   time.Time)` are `stream`'s, which then imports `frame`, a core. A
   line-mode host feeds a `Doc` from a network goroutine through them too.
   `stream/streamtea` holds `DocPane` and its keys alone.
   *Amended 2026-11-03, by
   [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
   D17:* while lines stay queued, `Pull` asks `b`'s Scheduler for the
   next frame, so that a host or a pane with no scheduler of its own
   drains the queue a frame at a time; its result still reports it.
10. **`Writer` never drops a write.** grok's drawer skips a frame while
    the terminal is busy (0003-REPORT §8.17). A1's `Writer` is an
    `io.Writer` under `tea.WithOutput`, and Bubble Tea v2's renderer
    (`cursed_renderer.go:643`, ultraviolet's `TerminalRenderer`) writes
    "the minimal necessary escape sequences to transition the terminal to
    the new buffer state" (`terminal_renderer.go:111-117`), so a dropped
    write would leave the screen wrong. While one write is in flight, `Writer` appends the writes after
    it to one pending buffer, written whole when the first completes: many
    draws become one write, and no byte is lost, as A1's test says. A
    write's error is returned by the next `Write` and by `Close`.
    *Amended 2026-11-02, by
    [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
    D9:* an `io.Writer` under `tea.WithOutput` is not transparent. Bubble
    Tea `v2.0.10` finds the terminal only as a `term.File`
    (`tty_unix.go:30`, `tty_windows.go:36`), for the window size, the
    resize listener, Windows' virtual-terminal mode and the colour
    profile, and restores the terminal straight after its last flush
    (`tty.go:33-49`). So `frame.Output` forwards `Fd` and `Read` to what
    it wraps, and is a `term.File`; and a `Write` waits for its own bytes
    up to `WithWriteWait` (100 ms), so that on a terminal that keeps up
    Bubble Tea's last bytes land before the restore, and only a slower
    write lets the program go on.
11. **Defaults the research gives:** `Demand`'s `Slow` ticks every 83 ms
    and `Fast` at the scheduler's frame rate (grok's `TickDemand`); the
    `Coalescer`'s maximum wait is 100 ms (Kilo); `Writer` reports a stall
    after 5 s (grok). Each is an option, and a row of the defaults guide.
    *Corrected 2026-11-01, by
    [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
    D1:* the two ticks are the fields of a value, `frame.Ticks`, whose
    zero fields take the defaults, not options.
12. **What does not change:** A6, as committed, but for items 2, 3 and 9's
    readings of §5 and A5 above; option A; every other section.
13. **Answered 2026-11-01** (picked from options): Q13 "stream
    (Recommended)". `Interpret`, `InterpretOptions` and `InterpretRow` are
    `stream`'s; A1's "§2, `safetext` gains … an interpreter" and its Q6
    read so; `safetext` keeps its narrow rule.

### A8 (2026-11-01): a plain streaming writer, `stream.Writer`

*Status: accepted (2026-11-01),* Q14 answered with its recommendation
(item 8). Proposed the same day, after the owner said "we need a plain
streaming wrapper to enhance extensibility and flexibility", during
[0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
Step 1, and chose "An io.Writer around the stream (Recommended)" over one
handle over the whole engine and a pipeline of stages.

1. **The need.** `stream.Printer` gives strings: `Write(chunk) string`
   and `Flush() string` (`stream/printer.go`). A program copies each to
   its destination itself, and keeps the tail for the reply's end. Nothing
   lets the standard library's composition reach the stream: `io.Copy`
   from a reply's body, `io.MultiWriter` to a terminal and a log,
   `bufio`, a pipe, or a CI job's output.
2. **The decision: `stream.Writer`,** an `io.WriteCloser` and
   `io.StringWriter` over a `Printer`:

   ```go
   func NewWriter(dst io.Writer, o ...Option) *Writer
   func (w *Writer) Write(p []byte) (int, error)
   func (w *Writer) WriteString(s string) (int, error)
   func (w *Writer) Close() error
   ```

   * **Options** are `stream.Option`, passed to its `Printer`:
     `WithMarkup`, `WithWidth`, `WithMethod`, `WithRules`,
     `WithoutTableLayout`, and one more, `WithCRLF()`, which ends each line
     with CR LF, for a terminal in raw mode written without an
     `inline.Region`; the `Printer` honours it too. ~~With no option, the
     output is the input cleaned by `safetext.Text`, unstyled and
     unwrapped: plain.~~ *Corrected 2026-11-01, by
     [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
     D4:* with no option, the `Writer` has the `Printer`'s defaults: the
     input cleaned by `safetext.Text`, unstyled and unwrapped, a table laid
     out in aligned columns; with `WithoutTableLayout` too, the cleaned
     input byte for byte.
   * **Each `Write`** gives its bytes to the `Printer`, and writes what is
     now stable to `dst` in one `dst.Write`. The output is the same for any
     split of the input, the `Printer`'s own property.
   * **`Close`** writes what the `Printer` still holds, ending the reply,
     and does not close `dst`, which is the program's, as `gzip.Writer`'s
     `Close` leaves its writer open. A second `Close` returns nil; a
     `Write` after it returns `ErrClosed`.
   * **Errors are sticky:** once `dst` fails, every later `Write` and
     `Close` returns that error, as `bufio.Writer`'s does. A `Write` whose
     own `dst.Write` fails returns `len(p)`, since the `Printer` holds its
     bytes, and the error.
   * **Used on one goroutine,** as the `Printer` is; `io.Copy` uses one.
   * **Over the `Printer` alone.** A `Doc` re-renders its tail, and an
     append-only destination cannot take a line back, so the writer is the
     line-stable form's. A program that draws a `Doc` draws it.
3. **Extensibility, the standard library's way.** The destination is any
   `io.Writer`, so a program tees, buffers, counts, compresses or
   redirects the output with what Go already has; the styling is
   `Markup`, which is data; the cleaning is `Rules`, never below the floor.
   No new hook is added.
4. **Q14. The type's name.** `stream.Writer` and A1's `frame.Writer`, both
   exported, would be one name in two packages, which
   `TestNoTypeNameMeansTwoThings` refuses unless `sharedNames` and the
   glossary allow it, and the glossary's rule is that one name means one
   thing. `python3 -m tools.verify names` finds `stream.Writer`,
   `stream.LineWriter` and `frame.Output` free today, since `frame.Writer`
   is reserved and not built.
   * **Recommended: `stream.Writer`, and A1's `frame.Writer` becomes
     `frame.Output`,** with `NewOutput`, `OutputOption` and
     `OutputEvent`: what it wraps is a program's terminal output, given to
     `tea.WithOutput`, and `stream.Writer` reads as Go's `bufio.Writer`
     and `tabwriter.Writer` do.
   * **`stream.LineWriter`,** keeping `frame.Writer` as A1 named it.
   * **Both `Writer`,** each added to `sharedNames` and the glossary's
     "Deliberate".
5. **Defaults:** each line ends with LF (`stream.WithCRLF`); `Close`
   leaves the destination open (decided here, the destination being the
   program's).
6. **Order.** It needs the `Printer` alone, which exists: a step of its
   own, A8.1, after Step A1.1 and before Step 3.
7. **Versioning.** An addition to `stream`, released with the root's
   minor release of PLAN Step 8; `make apicheck` stays at 0 incompatible
   changes.
8. **Answered 2026-11-01** (picked from options): Q14 "stream.Writer;
   frame's → Output (Recommended)". The writer is `stream.Writer`. A1's
   `frame.Writer`, `NewWriter`, `WriterOption` and `WriterEvent`, and
   A6's and A7's mentions of `Writer` (A6 items 4 and 8, A7 items 10 and
   11), read `frame.Output`, `NewOutput`, `OutputOption` and
   `OutputEvent`; neither is built, so nothing breaks. `python3 -m
   tools.verify names` over the four: "names: 4 planned name(s), clean".

## More Information

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md):
  §3 (crush, codex, gemini-cli, goose, aider), §5 (terminal standards), §7
  (candidate 5); for A1, §8.5, §8.10, §8.12, §8.15, §8.17, §10 and §11.1.
* [0002-MADR-multi-pane-workspace-layouts.md](0002-MADR-multi-pane-workspace-layouts.md)
  §3: `Pane`, `Changer` and mouse routing, which §5 and §6 build on.
* [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md):
  the width-method alignment and the focus messages `stream.Pane` relies
  on.
* [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md):
  OSC 8 links after cleaning. Once its `Caps` reports bracketed-paste
  support, `inputfilter` can use that instead of waiting for the first
  `tea.PasteStartMsg`.
* [0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)
  §3 and §6: the stack, and the conventions every package follows.
* [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md):
  the nested-module layout, the `go.work`, the release order and the
  per-module gates amendment A2 relies on.
* [0010-REPORT-nested-modules-and-adapter-sources.md](../reports/0010-REPORT-nested-modules-and-adapter-sources.md):
  §2 (module graph pruning, measured), §9 (glamour v2.0.1's source) and §10
  (the rules, per adapter).
* CommonMark 0.31.2, sections 4.5 (fenced code), 4.6 (HTML blocks), 4.7
  (link reference definitions) and 5.2 (list items):
  <https://spec.commonmark.org/0.31.2/>.
* ECMA-48, 5th edition, §5 (control functions):
  <https://ecma-international.org/publications-and-standards/standards/ecma-48/>.
  The DEC parser state machine: <https://vt100.net/emu/dec_ansi_parser>.
* Verified since §Context was written (0010-REPORT §9): a glamour
  `TermRenderer` is not safe for concurrent use, which §7's lock already
  assumed; glamour v2 never queries the terminal; and the code-block chroma
  style is process-wide.
* Not verified: whether glamour v2 enables GFM footnotes by default (§4
  treats footnote definitions as reference definitions either way);
  `WithWordWrap(0)`'s behaviour; and the licences of glamour's transitive
  dependencies, which Step 7 of the PLAN checks before the requirement is
  added.
