# Line-mode components

For a CLI that writes into the terminal's own scrollback, with no
alternate screen and no Bubble Tea, and wants to ask a question, pick an
entry from a list, edit and submit a line, stream a reply's Markdown, and
print the conversation again when the window narrows. These packages do
the work:

- `inline` draws a component's live rows below the scrollback and redraws
  them in place (`Region`), and reads the input only when asked
  (`Reader`).
- `ask` asks a question with answers chosen by one key (`Asker`);
  `ask/asktea` asks the same question in a workspace overlay.
- `pick` picks one entry from a filtered list (`Chooser`);
  `palette.FromEntries` serves the same entries in the TUI's palette.
- `lineedit` is the line the user types (`Composer`), with history and
  completion; `lineedit/lineedittea` puts it in a workspace pane.
- `stream` prints a streamed reply's Markdown, each line once it can no
  longer change (`Printer`).
- `transcript` keeps what the program has shown and prints it again at the
  new width once a resize settles (`Transcript`).
- `inline/inlinetest` is a terminal's screen for tests (`Screen`).

`termkey`'s `Decoder` turns the input's bytes into keys and pastes. Each
of these packages links no Charm module but `x/ansi`; the `…tea` packages
are the Bubble Tea forms. Why they are built this way is in
[0023-MADR](../decisions/0023-MADR-line-mode-components.md), with its
amendments A1 to A11. The examples `ExampleAsker`, `ExampleChooser`,
`ExampleComposer`, `ExamplePrinter` and `ExampleTranscript` compile and
run with each package's tests
(`go doc -all github.com/maccavelli/go-tui-lib/v2/lineedit`). Each Go
block below is cut from one of them, or from the program
`testdata/frameworks/linemode`, and `make examples` fails when the two
differ.

## What is yours

The library writes nothing, reads nothing and starts nothing: each
component gives the bytes to write and asks for what it needs. These stay
the program's (0023-MADR D10):

- raw mode, and restoring the terminal's mode at the end;
- the console's VT modes on Windows;
- the terminal's size, and the resize signal or a poll;
- bracketed paste, on and off;
- the external editor's process, and the history file;
- whether to use colour and Unicode, which `termcap` helps decide;
- a cap on the width, and what a second ctrl+c means.

## The host loop

Every component is an engine that does no I/O, has no clock and starts no
goroutine (0023-MADR D2). The host:

1. calls `Resize(width, now)` first, which draws the component;
2. gives it each decoded input, in order, with `HandleInput(in, now)`;
3. calls `HandleTimeout(now)` once the deadline `Timeout()` gave has
   passed, and `Cancel()` at the end of its input;
4. after each call, drains `NextEvent()` until its zero kind: it writes
   each write event's `Data` to the terminal, in order, and takes the
   result from the done event. A kind the host does not know is ignored.

This is the loop around a `Composer`, as `ExampleComposer` runs it on a
test screen:

<!-- from: lineedit/example_test.go#run -->
<!-- from: testdata/frameworks/linemode/main.go#run -->

```go
// widthPoll is how often the loop looks at the terminal's width when
// nothing else wakes it: a host may watch SIGWINCH instead, where there is
// one.
const widthPoll = 200 * time.Millisecond

// run is a line-mode host's loop around a Composer
// (docs/guides/line-mode.md): it drains the Composer's events, writing
// what it asks; reads the input only when asked, until the Composer's
// deadline, the escape timeout or the next look at the width; decodes
// what it read and feeds each input in; and gives the Composer the width
// when it changes. It returns the lines submitted, at the end of the
// input.
func run(in *inline.Reader, out io.Writer, width func() int, c *lineedit.Composer) ([]string, error) {
    var (
        dec  termkey.Decoder
        buf  = make([]byte, 4096)
        cols = width()
        got  []string
    )
    c.Resize(cols, time.Now())
    for {
        for e := c.NextEvent(); e.Kind != lineedit.EditNone; e = c.NextEvent() {
            switch e.Kind {
            case lineedit.EditWrite:
                if _, err := io.WriteString(out, e.Data); err != nil {
                    return got, err
                }
            case lineedit.EditSubmit:
                got = append(got, e.Text)
            case lineedit.EditInterrupt:
                c.SetText("") // ctrl+c's meaning is the host's: here, the line is cleared
            case lineedit.EditEOF:
                return got, nil
            }
        }

        now := time.Now()
        wake := now.Add(widthPoll)
        if t, ok := c.Timeout(); ok && t.Before(wake) {
            wake = t
        }
        if esc := now.Add(termkey.DefaultEscTimeout); dec.Pending() && esc.Before(wake) {
            wake = esc
        }
        ctx, cancel := context.WithDeadline(context.Background(), wake)
        n, err := in.Read(ctx, buf)
        cancel()

        now = time.Now()
        ins := dec.Decode(buf[:n], nil)
        if n == 0 && dec.Pending() {
            ins = dec.Flush(ins) // the input went quiet with a sequence held
        }
        for _, k := range ins {
            c.HandleInput(k, now)
        }
        switch {
        case err == nil:
        case errors.Is(err, context.DeadlineExceeded):
            if t, ok := c.Timeout(); ok && !now.Before(t) {
                c.HandleTimeout(now)
            }
        case errors.Is(err, io.EOF):
            c.Cancel()
        default:
            return got, err
        }
        if w := width(); w != cols {
            cols = w
            c.Resize(cols, now)
        }
    }
}
```

- **One wake-up.** The loop waits for the first of three things: the
  component's own deadline (a cursor report's wait, or a key sequence's),
  the escape timeout while the decoder holds bytes, and the next look at
  the width.
- **Flush when the input goes quiet.** A lone ESC is the escape key only
  after a pause; read with the next key, it is alt and that key, so ESC
  then enter would insert a line break. `termkey.DefaultEscTimeout` is
  50 ms, as Bubble Tea's.
- **The end of the input is `Cancel`.** The `Composer` answers with
  `EditEOF`, the `Asker` and `Chooser` with their done event, canceled.
- **The width is polled here.** Windows has no SIGWINCH; on Unix a host
  may watch it instead, and wake the read by cancelling its context.
- **The `Asker` and the `Chooser` run in the same loop,** with their own
  events: `AskWrite` and `AskDone`, and `PickWrite` and `PickDone`.

## Raw mode, and what to restore

The program sets raw mode, here with `golang.org/x/term`, which the
library does not require. The region's `Restore` gives the bytes that
undo what an interrupted drawing could leave set, the cursor hidden and
mode 2026 on: `launch.Restorer`'s shape.

<!-- from: testdata/frameworks/linemode/main.go#session -->

```go
func session() ([]string, error) {
    in, out := int(os.Stdin.Fd()), int(os.Stdout.Fd())
    if !term.IsTerminal(in) || !term.IsTerminal(out) {
        return nil, errors.New("not a terminal")
    }
    old, err := term.MakeRaw(in)
    if err != nil {
        return nil, err
    }
    defer func() { _ = term.Restore(in, old) }()

    region := inline.NewRegion()
    _, _ = os.Stdout.WriteString(ansi.SetModeBracketedPaste)
    defer func() { _, _ = os.Stdout.WriteString(region.Restore() + ansi.ResetModeBracketedPaste) }()

    width := func() int {
        w, _, err := term.GetSize(out)
        if err != nil || w <= 0 {
            return 80
        }
        return w
    }
    c := lineedit.NewComposer(lineedit.WithRegion(region))
    return run(inline.NewReader(os.Stdin), os.Stdout, width, c)
}
```

## One reader, one region

- **One `Reader` for the whole program.** `inline.NewReader(os.Stdin)`
  reads only when asked, one read at a time, and never closes its input:
  on Windows a console read keeps the mode it started in, and closing
  console input while a read is pending hangs the process. The program's
  own prompts read the same `Reader`, so no second reader takes their
  keys. A context ends the wait, not the read: a read not taken by then is
  taken by the next `Read`.
- **One `Region`, shared.** Build it once, with the `inline` options you
  need, and give it to each component with `WithRegion`; one component
  draws in it at a time. Without `WithRegion`, a component builds its own.
- **Text above the live rows** goes through `region.Print(text)`: it clears
  the region, writes the text, and draws the region again below it, so a
  reply streams above the line being typed with no lock, both on the
  host's loop. Its line breaks are written as CR LF.
- **A resize follows the terminal's re-wrap.** On `Resize` the region asks
  where the cursor is (`CSI 6 n`) and takes the report out of the input
  that reaches the component's `HandleInput`. With no report within
  `inline.DefaultReportWait`, 150 ms, it redraws by `WithReflow`'s kind:

  | `Reflow` | The terminal | Seen in |
  | :--- | :--- | :--- |
  | `ReflowDown` | re-wraps each row, the cursor moving down with it | WezTerm, kitty, Terminal.app, the Windows pseudo-console |
  | `ReflowKeepRow` | re-wraps each row, the cursor staying on its screen row | iTerm2, VTE |
  | `ReflowCut` | keeps each row on one row, cut at the new width | xterm |
  | `ReflowUnknown` | not known: redrawn as `ReflowDown`, the default | |

  Set it from what you know, such as `termcap`'s identity of xterm.
  `WithoutCursorQuery` redraws by that kind at once, with no query.
- **Measure as the terminal does:** `WithMethod(ansi.WcWidth)`, the
  default, until the terminal reports grapheme clustering (mode 2027),
  and `ansi.GraphemeWidth` after, on each component and the region.

## Ask a question

A `Question` is rows of answers, each chosen by one key. The answer with
`Reason` reads a line after its key, up to enter. The `Cancel` answer's
key, ctrl+c, and the host's `Cancel()` cancel. The question and the answer,
in words, stay in the scrollback.

<!-- from: ask/example_test.go#question -->

```go
q := ask.Question{
    Title: "Run go test ./... in the repository?",
    Rows: []ask.Row{
        {Label: "Allow", Answers: []ask.Answer{
            {ID: "once", Key: "1", Label: "once"},
            {ID: "session", Key: "2", Label: "this session", Echo: "allowed for this session"},
        }},
        {Label: "Reject", Answers: []ask.Answer{
            {ID: "why", Key: "3", Label: "and say why", Echo: "rejected", Reason: true},
        }},
    },
    Cancel: ask.Answer{ID: "cancel", Key: "c", Label: "cancel"},
}
```

`ExampleAsker` asks it on a 60-column screen with ASCII glyphs, and types
`3`, `not yet` and enter. The `Response` is the answer `why` with its
reason, and the screen holds:

```text
Run go test ./... in the repository?
  Allow   [1] once  [2] this session
  Reject  [3] and say why   [c] cancel
> rejected: not yet
```

- `ask.Layout(q, width, method, glyphs)` is the layout alone, a pure
  function: a row that does not fit is broken between answers, and a word
  wider than its row is broken at the width.
- A reason is edited by `lineedit`'s editing commands, rebound through
  `WithKeymap`, and bounded by `WithReasonLimit`, `ask.DefaultReasonLimit`
  (2,048 bytes) by default.
- The Asker's own words, such as the label before a reason, come from
  `WithCatalog`; `ask.Messages()` gives their English defaults.
- In the TUI, `asktea.NewDialog(ws, q)` asks the same question in an
  overlay, drawn by the same `Asker`.

## Pick an entry

An `Entry` is what the program acts on (`ID`), what is matched and drawn
(`Text`), and what is drawn at the right and not matched (`Detail`). Typing
edits the query, and each change ranks the entries with `fuzzy`'s
matcher, best first, ties in the order given. `ExampleChooser` picks from
three models, typing `son`:

```text
> son                                       1 of 1
> anthropic/claude-sonnet-5-5                   1M
```

Enter chooses: `PickDone` carries the `Entry` as it was given, and one line
is left in the scrollback:

```text
> anthropic/claude-sonnet-5-5
```

- **Keys** are commands in the program's keymap, in `pick.ChooserScope`
  and then `lineedit.ComposerScope`: `pick.DefaultRules()` binds up and
  ctrl+p, down and ctrl+n, pgup and pgdown, enter, and esc and ctrl+g, and
  `lineedit.DefaultRules()` the query's editing keys. ctrl+c cancels,
  outside the keymap.
- **Size:** `WithHeight`, `pick.DefaultHeight` (10) rows by default;
  above `WithIndexAbove`, 5,000 entries by default, a `fuzzy.Index`
  narrows the candidates first.
- **Matches:** `WithMarkup(stream.DefaultMarkup())` underlines the cells
  the query matched; by default nothing is styled, and the focus glyph
  marks the selected row.
- **In the TUI,** `palette.FromEntries` serves the same `[]pick.Entry`,
  ranked by the same matcher.

## Edit and submit a line

The `Composer`'s events are its contract with the host:

| Event | The host |
| :--- | :--- |
| `EditWrite` | writes `Data` |
| `EditSubmit` | takes `Text`; the Composer has left it in the scrollback and drawn an empty prompt below it |
| `EditEOF` | ends: ctrl+d on an empty line, or the host's `Cancel` |
| `EditInterrupt` | decides what ctrl+c means; the text stays until `SetText` changes it |
| `EditExternal` | edits `Text` in its editor and gives it back with `SetText`; sent only `WithExternalEditor` |

- **Layout.** The text is laid out in whole words, by grapheme cluster,
  each row after the first indented by the prompt's width. alt+enter,
  shift+enter where the terminal reports it, or a trailing backslash
  before enter, starts a new line after `WithContinue`'s prompt.
- **History.** Up and down move by row, and from the first or the last
  row recall `WithHistory`'s `History`, x/term's interface. `MemHistory`
  keeps one in memory, `DefaultHistorySize` (1,000) entries, dropping an
  empty line and a repeat of the last; `WriteTo` and `ReadFrom` are its
  file form, and the file is the program's.
- **Completion.** `WithCompleter(lineedit.CompleterFunc(f))` is what tab
  completes with, as x/term's `AutoCompleteCallback`.
- **Paste.** A bracketed paste is inserted, its line breaks and tabs kept;
  `WithPaste(lineedit.PasteSubmit)` submits it at once.
- **Off by default:** ctrl+l clears the screen only `WithClearScreen`, and
  ctrl+x ctrl+e asks for the external editor only `WithExternalEditor`.
- **Keys** are commands in `lineedit.ComposerScope`, `DefaultRules()` by
  default and the program's keymap through `WithKeymap`; ctrl+c is outside
  the keymap.
- **The text** is bounded by `WithLimit`, `DefaultLimit` (4,096) runes by
  default; `WithHeader` draws a line above each new prompt.

## Stream a reply

A `Printer` takes the reply as it arrives, and gives the lines that are
stable: cleaned, styled, every Markdown marker kept, and prose wrapped at
whole words. A paragraph's line waits for the next, since an underline can
make it a heading and a delimiter row a table's header; a table waits
until it ends, then is laid out in aligned columns; a fence's lines print
as they come. Nothing printed changes later, and the output is the same
for any split of the input. `ExamplePrinter` writes four chunks at 32
columns:

```text
-- write 1
# The reply

-- write 2
A paragraph that streams in, and
is wrapped at whole words to the
width.

-- write 3
-- write 4
-- flush
| name | cells |
| ---- | ----: |
| ok   |     2 |
| wide |    10 |
```

- **Print the lines above the composer** with `region.Print(p.Write(chunk))`,
  and the rest with `region.Print(p.Flush())` at the reply's end.
- **Styling** is a `Markup`, one `Style` for each part: the zero `Markup`,
  the default, styles nothing, which is what a program without colour
  keeps, under `NO_COLOR` or an ASCII profile; `stream.DefaultMarkup()` is
  gobble's styling.
- **Width.** `WithWidth` at the start, `SetWidth` for what comes next; a
  line printed already stays as it was. `WithoutTableLayout` prints a
  table's rows as they came.
- **Cleaning** is by `WithRules`, never below `safetext.Text()`.

## Print the conversation again on a resize

A terminal re-wraps its old rows itself when the window narrows, and a
table, a diff or a code block comes out ragged. A `Transcript` keeps what
the program has shown, and once the resize settles prints it again at the
new width:

- `AddText(s)` keeps text as written, cleaned; `AddDrawn(f)` keeps a
  function of the width, drawn anew each time, such as a help listing;
  `AddReply(p)` keeps a reply, drawn again from its source.
- It keeps at most `WithMaxBytes`, 4 MiB by default, the oldest pieces
  going first, and a reprint draws the newest back to `WithMaxRows`,
  10,000 rows by default, after a notice line when anything was left out.
- `Resize(width, now)`: the first call sets the width, and a later one
  with another width starts the settle, `WithSettle`, 150 ms by default.
  `Timeout()` gives its deadline; add it to the loop's wake-up.
- `HandleTimeout(now, idle)` gives the reprint once the settle has passed
  and the host says its output is idle, at a line's start with nothing
  held; until then it gives "". Call it at each idle point.

This is the host's side of it, on the loop's goroutine, as
`ExampleTranscript` runs it on a test screen:

<!-- from: transcript/example_test.go#host -->

```go
// host is a conversation's state, on the loop's goroutine: the region the
// composer draws in, the transcript, and the reply streaming.
type host struct {
    out    io.Writer
    region *inline.Region
    c      *lineedit.Composer
    tr     *transcript.Transcript
    reply  *stream.Printer
    prompt string // the composer's WithPrompt
    cols   int
}

func (h *host) write(s string) { _, _ = io.WriteString(h.out, s) }

// submitted keeps the line the composer left in the scrollback, and
// starts the reply.
func (h *host) submitted(text string) {
    h.tr.AddText(h.prompt + text + "\n")
    h.reply = stream.NewPrinter(stream.WithMarkup(stream.DefaultMarkup()), stream.WithWidth(h.cols))
    h.tr.AddReply(h.reply)
}

// chunk prints the reply's stable lines above the composer.
func (h *host) chunk(s string) {
    if out := h.reply.Write(s); out != "" {
        h.write(h.region.Print(out))
    }
}

// done ends the reply.
func (h *host) done() {
    if out := h.reply.Flush(); out != "" {
        h.write(h.region.Print(out))
    }
}

// resize gives each part the new width.
func (h *host) resize(cols int, now time.Time) {
    h.cols = cols
    h.c.Resize(cols, now)
    h.tr.Resize(cols, now)
    if h.reply != nil {
        h.reply.SetWidth(cols)
    }
}

// idle reprints the conversation once a resize has settled, when the
// reply holds nothing back.
func (h *host) idle(now time.Time) {
    idle := h.reply == nil || h.reply.Idle()
    if out := h.tr.HandleTimeout(now, idle); out != "" {
        h.write(h.region.Clear() + out + h.region.Draw(h.c.View(h.cols)))
    }
}
```

- **Drain the Composer's events before writing your own bytes,** so they
  reach the terminal in order.
- **Around the reprint,** the region is removed with `Clear` and drawn
  again with `Draw`, from the Composer's `View`.
- **The reprint clears the scrollback** (`CSI 3 J`), and with it what came
  before the program: that is the price of rows that line up.
  `WithoutScrollbackClear` keeps the scrollback, and the old rows in it at
  their old width. kitty clears it though its terminfo leaves `E3` out,
  and iTerm2 may ask the user first.
- The notice is the catalogue's `transcript.earlier-output`, through
  `WithCatalog`; `WithNoticeStyle(stream.Dim)` dims it.

## Test on a screen

`inlinetest.NewScreen(columns, rows, method)` is a terminal's screen
for tests: write the component's bytes to it, and check its `Lines`,
`Scrollback` or `Text`, not the bytes. `Resize(columns, reflow)` narrows
it the three ways terminals do; `Answer` is given the reply to a cursor
query, so a host test feeds it back as input; `Cooked` makes a line feed
return to the row's start, as outside raw mode. The examples above, and
each package's goldens, draw on it.

## Defaults

Every default these packages apply, and its control, is in
[the defaults guide](defaults.md): the region's cursor query and
synchronized output, each component's keys, sizes and bounds, the
printer's styling and table layout, and the transcript's settle, limits
and scrollback clear. The floors that have no control are the cleaning of
untrusted text, the bounds on input, and the cursor report's protocol.
