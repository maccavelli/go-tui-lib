# Streaming content

A model's reply arrives a few bytes at a time, often faster than a terminal
can draw it, from a goroutine that is not the program's. These packages
draw it as it comes, without drawing it twice or out of order, and keep
the input that arrives meanwhile from flooding the event loop
([0009-MADR](../decisions/0009-MADR-streaming-content-engine.md)):

| Package | What it does | Links |
| :--- | :--- | :--- |
| `stream` | `Doc`, a streamed Markdown document drawn a stable prefix at a time; `Buffer`, the way in from other goroutines; `Printer`, the line mode's printer; `Bounded` and `Interpret`, tool output | no Charm module but `x/ansi` |
| `frame` | the `Scheduler`, which merges requests for a frame; the pacers, which release queued lines a frame at a time | the standard library |
| `inputfilter` | the wheel, motion and paste bursts, coalesced; split reports joined | the standard library |
| `stream/streamtea` | `DocPane`, a `Doc` in a workspace pane | Bubble Tea |
| `frame/frametea` | the `Scheduler`'s frames as Bubble Tea commands | Bubble Tea |
| `inputfilter/inputfiltertea` | `Filter`, for `tea.WithFilter` | Bubble Tea |

The cores serve a line-mode program as they serve a TUI: only the `…tea`
packages name Bubble Tea.

## Feed a Doc

A `Doc` is one document. Give it chunks with `Append` (or `Write`), draw
it with `Render(width)` when a frame is due, and end it with `Finish`:

<!-- from: stream/document_test.go#doc -->

```go
d := stream.NewDoc(stream.Plain(ansi.WcWidth))
for _, chunk := range []string{"A first para", "graph.\n\nThen a **sec", "ond** one."} {
    d.Append(chunk)
    fmt.Printf("%q\n", d.Render(40))
}
d.Finish()
fmt.Printf("%q\n", d.Render(40))
```

- **A frame is cheap.** A blank line where nothing stays open is a cut:
  the text before it is rendered once a width and kept, and only the tail
  after the last cut is rendered each frame. A tail that goes 2 KiB with
  no cut is cut where its block can be split (`WithRelaxAfter`).
- **A frame need not be copied.** `Render`'s frame is one new string each
  version, the kept part copied into it with the tail. A host that draws
  the frame's last rows itself, in a pane or an inline region, takes the
  two parts with `RenderParts(width)`: the committed part, and the tail,
  neither copied; `Render` is the two joined by a blank line. Since
  `v2.12.0`
  ([0025-MADR-charm-free-stream-allocations.md](../decisions/0025-MADR-charm-free-stream-allocations.md)
  D1).
- **The partial last line shows,** up to an emphasis, a strikethrough, a
  code span or a link it opens and has not closed: `Then a **sec` shows
  as far as `Then a`. `WithoutInlineHoldback` shows it whole, `WithoutPartialLine`
  hides it, and `WithMath` holds an unclosed `\(` or `$$` too.
- **`Version`** rises with every change to what `Render` gives, so a view
  that keeps its last frame asks it whether to draw again.
- **The `Renderer` is yours.** `NewDoc(nil)` draws with `Plain`, which
  wraps prose and lays tables out and styles nothing; a program passes
  its own, such as the planned glamour adapter's.

<!-- from: stream/document_test.go#parts -->

```go
d := stream.NewDoc(stream.Plain(ansi.WcWidth))
d.Append("A first paragraph.\n\nThen a second, whose first line is whole,\nand whose next is still stream")
prefix, tail := d.RenderParts(40)
fmt.Printf("%q\n%q\n", prefix, tail)
```

The prefix is `"A first paragraph."`, committed once the second
paragraph's first line came, and the tail the second paragraph, wrapped at
40. The prefix is drawn anew, not only grown, when a reference definition
is committed, at a width the `Doc` no longer caches (it keeps two), and
after `SetRevision` or `Reset`; after `Finish`, it is the whole frame.

A `Doc` is used on one goroutine. A reply that comes from a network
reader's goroutine goes through a `Buffer`: its `Write` is safe from any
goroutine, it queues whole lines with the time each came, and it asks its
`Scheduler` for a frame when a line completes; `Close` ends the reply. On
each frame, `Doc.Pull(b, pacer, now)` appends the lines the pacer
releases, and the partial line once no complete one waits, so text never
shows out of order; while lines still wait, it asks for the next frame
itself.

## Pace frames with the Scheduler

The `Scheduler` merges every request for a frame into one, at most 60 a
second (`WithFPS`): `Request` and `RequestIn` are safe from any goroutine,
and the frames come on `Frames()`, a channel of capacity one. Between
frames no goroutine of its own runs. `Close` stops it.

A line-mode host selects on `Frames()` beside its input and its timers:

<!-- from: frame/example_test.go#scheduler -->

```go
s := frame.NewScheduler()
defer s.Close()
b := stream.NewBuffer(s) // a completed line asks s for a frame
d := stream.NewDoc(nil)
pacer := frame.Adaptive()
go answer(b)
for !d.Finished() {
    due := <-s.Frames()      // a host selects on Frames beside its input and its timers
    d.Pull(b, pacer, due.At) // the lines due; another frame is asked for while more wait
    _ = d.Render(40)         // draw the frame
}
```

A Bubble Tea program waits for a frame with `frametea.Wait(s)`, a command
that returns the next `frame.DueMsg`. A command runs once, so the program
re-arms it on each frame: that is the one line in `Update` below. Or
`frametea.Forward(s, p.Send)` sends every frame from a goroutine of its
own until its `stop`.

## Choose a pacer

A pacer decides how many queued lines a frame releases.

- **`frame.Adaptive()`,** codex's two gears: `Smooth` releases one line a
  frame, so a reply reads as it is typed; `CatchUp` releases the whole
  queue once it is 8 lines deep or its oldest line 120 ms old, and eases
  back once it stays shallow and young. Every number is an option. It
  keeps its state from frame to frame, so each queue has its own.
- **`frame.Immediate()`** releases everything each frame: the text as
  fast as it comes.
- **A `frame.PacerFunc`** is your own rule over the queue's `Snapshot`.

`Pull` with a nil pacer is `Immediate`; a `DocPane` uses a new `Adaptive`
unless `WithPacer` gives another.

## Show a Doc in a pane

`streamtea.NewDocPane(w, d, …)` is a `Doc` in a workspace pane. Built
`WithBuffer(b)`, it pulls the `Buffer` on each `frame.DueMsg` the
workspace gives it:

<!-- from: stream/streamtea/example_test.go#docpane -->

```go
// model is a program whose workspace shows an answer as it streams in.
type model struct {
    w      *workspace.Workspace
    frames *frame.Scheduler
}

func (m model) Init() tea.Cmd { return tea.Batch(m.w.Init(), frametea.Wait(m.frames)) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    cmd := m.w.Update(msg) // a frame.DueMsg reaches the DocPane, which pulls its Buffer
    if _, ok := msg.(frame.DueMsg); ok {
        cmd = tea.Batch(cmd, frametea.Wait(m.frames)) // the one line: wait for the next frame
    }
    return m, cmd
}

func (m model) View() tea.View { return tea.NewView(m.w.Render()) }

// answer writes the answer into the Buffer from a goroutine of its own, as
// a network reader does, and closes it at the end.
func answer(b *stream.Buffer, chunks []string) {
    for _, c := range chunks {
        _, _ = b.Write([]byte(c))
    }
    _ = b.Close()
}
```

- **It follows the tail** until the user scrolls up, and again once they
  scroll back to the end; `WithoutFollow` keeps it where it was put.
- **It draws with the workspace's theme and width method,** read at each
  draw: `ScrollUp` and `ScrollDown` in a column of their own mark rows
  above and below, and a row too wide ends with the theme's `Ellipsis`.
  It gives its `Doc` the theme's ID as the revision (`Doc.SetRevision`),
  so a renderer or a highlighter that draws with the theme draws again
  when it changes.
- **It is a `Changer`:** a message that changes nothing it shows leaves
  the frame as it was.
- **The wheel scrolls it** a row a notch, from a `tea.MouseWheelMsg` or
  any mouse message with `WheelDelta`, such as `inputfiltertea`'s
  `CoalescedWheelMsg`.

Its keys are commands, in the `"kind:docpane"` scope, which the pane
names as a `KeyContexter` and runs itself as a `CommandConsumer`. Add
`streamtea.DefaultRules()` to the program's one keymap, beside
`workspace.DefaultRules()`, and a user file rebinds them as any key:

| Command | Default key | Moves |
| :--- | :--- | :--- |
| `stream.scroll.up` | up | a row up |
| `stream.scroll.down` | down | a row down |
| `stream.page.up` | pgup | a page, the pane's height, up |
| `stream.page.down` | pgdown | a page down |
| `stream.top` | home | to the document's start |
| `stream.bottom` | end | to its end, which the pane then follows |

## Filter input noise

A wheel turned hard, or a mouse moved, sends Bubble Tea far more messages
than a frame can use, each one redrawing. `inputfiltertea.NewFilter`,
given to `tea.WithFilter`, sums the wheel and thins motion to one message
every 16 ms before they queue:

<!-- from: inputfilter/inputfiltertea/inputfiltertea_test.go#filter -->

```go
f := inputfiltertea.NewFilter(inputfilter.WithPasteBurst(inputfilter.PasteBurstConfig{}))
p := tea.NewProgram(nil, tea.WithFilter(f.Filter))
stop := f.Attach(p.Send) // the timed flushes and the drains reach the program through its Send
defer stop()
```

- **`Attach` is needed** for anything a stage holds to be given back on
  time: a timed flush and the messages after a burst reach the program
  through its `Send`. Keys reach the stages only while it is attached.
- **Paste bursts** are off by default. On a terminal that does not
  bracket a paste, `WithPasteBurst(PasteBurstConfig{})` tells a paste from
  typing by the keys' timing, codex's way, and gives it as one
  `tea.PasteMsg`; `PowerShellPasteBurst()` adds grok's rules for
  PowerShell.
- **On by default:** a paste's CR and CRLF made LF; an empty bracketed
  paste, which Windows Terminal sends for an image, given as
  `inputfilter.ClipboardImageRequestMsg`; a mouse report or reply split
  past the reader's 50 ms joined; an X10 report ConPTY re-encoded
  repaired. Each has its `Without…` option.
- **Before a sensitive screen,** `Quarantine(until)` drops input until a
  deadline, and `Capture` and `Replay` keep the user's typing for the
  next input, cut at the first Esc.

A line-mode host uses the same stages through `inputfilter.NewFlusher`.

## What the stream cannot send

A `Doc`, a `Printer` and a `Bounded` clean what the stream sends before
anything keeps it: by `WithRules`, then by `safetext.Text`, the floor. So
no escape sequence the model sent reaches the screen: not a colour, not a
cursor move, not a window title, and not an OSC 8 link. Tabs, line breaks
and the bidirectional controls prose needs are kept. The styling a frame
holds is the renderer's.

A link the program trusts it makes itself. `Doc.View` gives each link's
destination and cells; `termsvc.Linker{}.Link(url, text)` writes an OSC 8
hyperlink for the schemes it allows, http, https, file and mailto by
default, and refuses any other.

## What may differ before Finish

Until `Finish`, a frame may differ from one render of the whole text in
two stated ways, and no other:

1. the parts are joined by exactly one blank line, each trimmed of its
   blank lines at its start and end;
2. a block a relaxed cut split shows as two blocks of its kind.

After `Finish`, a frame is the renderer's of the whole text, byte for
byte. `streamtest.CheckRenderer(t, r)` holds your renderer to that: the
same frame for any chunking, no other difference, and `Finish` exact.

## Copy, links and selection

`Doc.View(width)` is `Render`'s frame with what a program needs to act on
it while it streams:

- **`Lines` and `Offsets`:** each row, and the source offset of the line
  it was drawn from, for selection;
- **`Links`:** each link's part on each row, its `URL`, its cells, and an
  `ID`, the source offset of its `[`, the same in every frame and at
  every width;
- **`Code`:** each code block's info string and its body's source,
  `src[c.Start:c.End]`, for "copy code block", and its rows;
- **`Tables`:** each table's cells as the source has them.

`Plain` reports all of them. Another renderer reports them by being a
`ViewRenderer`; one that is not gives the rows and their offsets alone.

## Highlight code

`WithHighlighter(h)` highlights fenced code by your `Highlighter`, its
language the info string's first word. A `Printer` highlights each line
as it prints it; a `Doc` whose renderer is `Plain` highlights its blocks,
each line of a growing fence once, from the state the line before it left.
Past 512 KiB, 10,000 lines or 4 KiB in a line, a block shows plain
(`WithHighlightLimits`). When what it highlights with changes, such as
the theme, `Doc.SetRevision` draws the document again; a `DocPane` does
this itself.

## Tables

`FitTable(rows, TableOptions{Width, Method, Border})` lays a table out at
a width: its columns shrink toward their longest word, its cells wrap at
their spaces, and no word, number or URL is broken. Too wide still, it is
key/value records, then stacked records, then its source. `Plain` fits
its tables to its width; a `Printer`'s tables are never wrapped, since
the terminal wraps a long row in its scrollback.

## Tool output

- **`Interpret(raw, InterpretOptions{})`** replays captured terminal
  output, a progress bar's carriage returns and a tool's colours among
  it, into rows, each `Styled` and `Plain`, every sequence but colour and
  the moves it applies dropped.
- **`Bounded`** holds live output within bounds: all of it up to 64 KiB,
  then its first and last 20 lines, each line cut at 4 KiB, the lines
  between counted by `Omitted` and shown as one line,
  `stream.MsgOmitted`, which `WithCatalog` translates. It cleans by
  `safetext.Text` unless `WithRules` says otherwise, and never less than
  `safetext.StatusLine`, so `WithRules(safetext.StatusLine())` keeps a
  tool's colours.

## In the line mode

A line-mode program, with no Bubble Tea, uses the cores directly: the
`Scheduler` in its loop's select, a `Printer` for the scrollback, which
prints each line once it can no longer change, and `inputfilter`'s
`Flusher` for its input. [The line-mode guide](line-mode.md) shows that
loop.

## Defaults

Every default here and its control, the pacer's numbers, the filter's
stages, the `Doc`'s holdback and relaxed cut, the highlight limits and
`Bounded`'s bounds, is in [the defaults guide](defaults.md), under
`stream`, `stream/streamtea`, `frame` and `inputfilter`.
