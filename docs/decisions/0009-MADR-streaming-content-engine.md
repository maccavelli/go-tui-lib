---
status: proposed
date: 2026-10-02
decision-makers: owner
consulted: 0003-REPORT-agent-tui-ecosystem-research.md (§3 agent TUIs, §5 terminal standards); crush, codex, gemini-cli, goose and aider source; Charm v2 APIs (bubbletea v2.0.10, lipgloss v2.0.6, bubbles v2.2.1, x/ansi v0.11.8); charm.land/glamour/v2 v2.0.1
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

## More Information

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md):
  §3 (crush, codex, gemini-cli, goose, aider), §5 (terminal standards), §7
  (candidate 5).
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
* CommonMark 0.31.2, sections 4.5 (fenced code), 4.6 (HTML blocks), 4.7
  (link reference definitions) and 5.2 (list items):
  <https://spec.commonmark.org/0.31.2/>.
* ECMA-48, 5th edition, §5 (control functions):
  <https://ecma-international.org/publications-and-standards/standards/ecma-48/>.
  The DEC parser state machine: <https://vt100.net/emu/dec_ansi_parser>.
* Not verified: whether glamour v2 enables GFM footnotes by default (§4
  treats footnote definitions as reference definitions either way), and the
  licences of glamour's transitive dependencies, which Step 7 of the PLAN
  checks before the requirement is added.
