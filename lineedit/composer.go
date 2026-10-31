package lineedit

import (
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/internal/lineinput"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// clearScreen is ctrl+l's erase, as x/term and gobble write it.
const clearScreen = "\x1b[2J\x1b[H"

// typing is the modifiers a key that types its text may hold.
const typing = termkey.ModShift | termkey.ModCapsLock | termkey.ModNumLock | termkey.ModScrollLock

// path is where the Composer's keys are resolved: its scope alone.
var path = keymap.Path{ComposerScope}

// noValues is the when values the Composer's keys are resolved with: none.
var noValues = when.Map{}

// pasteFloor is what a paste keeps at the least: printable text, line
// breaks and tabs (0023-MADR A3 item 11).
var pasteFloor = safetext.Rules{Newlines: true, Tabs: true}

// Composer is the line a user types, in an inline.Region, as an engine
// that does no I/O (docs/decisions/0023-MADR-line-mode-components.md D2,
// D6). The host calls Resize first, with the terminal's width, which draws
// the prompt; then HandleInput with each decoded input, HandleTimeout once
// Timeout's deadline has passed, and Cancel at the end of its input; and
// drains NextEvent after each call. Each submission is left in the
// scrollback, and an empty prompt drawn below it. Like the region, it is
// used on one goroutine.
type Composer struct {
	method    ansi.Method
	glyphs    glyph.Set
	region    *inline.Region
	rules     safetext.Rules
	keys      *keymap.Keymap
	disp      *keymap.Dispatcher
	prompt    string
	cont      string
	header    func() string
	history   History
	completer Completer
	limit     int
	paste     PasteMode
	external  bool
	clear     bool

	line     *lineinput.Line
	width    int
	head     string // the header above the prompt drawn now
	recalled int    // the history entry shown, or -1 for the user's own text
	own      string // the user's own text while an entry is shown
	enter    bool   // the last key submitted by enter, so a ctrl+j after it is its CR LF's LF

	// pending is a key sequence the dispatcher waits on, due when its wait
	// ends, with the timeout Expire takes.
	pending bool
	due     time.Time
	expire  keymap.TimeoutMsg

	done   bool
	events []ComposerEvent
	next   int // the first event NextEvent has not given

	// lay is the text laid out at layW cells, from the Line's edit mark;
	// view is the rows draw hands the region, reused.
	lay   laid
	layW  int
	layOK bool
	view  []string
}

// NewComposer is a Composer with an empty line.
func NewComposer(o ...Option) *Composer {
	c := &Composer{glyphs: glyph.Unicode(), rules: safetext.Line(), limit: DefaultLimit, recalled: -1}
	for _, opt := range o {
		opt.apply(c)
	}
	if c.region == nil {
		c.region = inline.NewRegion(inline.WithMethod(c.method))
	}
	if c.keys == nil {
		c.keys = defaultKeymap()
	}
	c.disp = keymap.NewDispatcher(c.keys)
	if c.prompt == "" {
		c.prompt = c.glyphs.Prompt + " "
	}
	if c.cont == "" {
		c.cont = c.glyphs.Ellipsis + " "
	}
	c.prompt, c.cont = c.clean(c.prompt), c.clean(c.cont)
	c.line = lineinput.NewRunes(c.limit, c.method)
	c.head = c.readHeader()
	return c
}

// clean is s as one line, under the Composer's rules, then the floor's.
func (c *Composer) clean(s string) string { return sanitize.Line(safetext.Clean(s, c.rules)) }

// cleanText is s with its line breaks and tabs kept, under the Composer's
// rules, then the paste's floor.
func (c *Composer) cleanText(s string) string {
	r := c.rules
	r.Flatten, r.Newlines, r.Tabs = false, true, true
	return safetext.Clean(safetext.Clean(s, r), pasteFloor)
}

func (c *Composer) readHeader() string {
	if c.header == nil {
		return ""
	}
	return c.clean(c.header())
}

// Text is the text being edited.
func (c *Composer) Text() string { return c.line.Text() }

// SetText makes s the text, cleaned with its line breaks kept and cut to
// the bound, the cursor at its end, as after the external editor. It
// draws the Composer again.
func (c *Composer) SetText(s string) {
	if c.done {
		return
	}
	c.line.SetText(c.cleanText(s))
	c.recalled, c.own = -1, ""
	c.draw()
}

// Resize gives the terminal's width, in cells, and draws the Composer at
// it: at once the first time, and after the region's redraw on a change.
func (c *Composer) Resize(width int, now time.Time) {
	if c.done {
		return
	}
	c.width = width
	c.write(c.region.Resize(width, now))
	c.draw()
}

// HandleInput takes one decoded input: the cursor report the region waits
// for, a key, or a bracketed paste.
func (c *Composer) HandleInput(in termkey.Input, now time.Time) {
	if c.done {
		return
	}
	out, took := c.region.HandleInput(in, now)
	c.write(out)
	if took {
		return
	}
	switch in.Kind {
	case termkey.InputPaste:
		c.enter = false
		c.line.Insert(c.cleanText(in.Text))
		if c.paste == PasteSubmit {
			c.submit(c.line.Text())
		}
	case termkey.InputKey:
		c.key(in.Key, now)
	}
	if !c.done {
		c.draw()
	}
}

// key takes one key: ctrl+c interrupts; the LF of a CR LF is dropped; the
// keymap's keys run their commands; any other types its text.
func (c *Composer) key(k termkey.KeyEvent, now time.Time) {
	stroke := k.Keystroke()
	if !k.Release && stroke == "ctrl+c" {
		c.enter = false
		c.events = append(c.events, ComposerEvent{Kind: EditInterrupt, Text: c.line.Text()})
		return
	}
	if c.enter && !k.Release && stroke == "ctrl+j" {
		c.enter = false // a terminal's or a pipe's CR LF is one enter, as x/term reads it
		return
	}
	c.enter = false
	o := c.disp.Key(k, path, noValues)
	c.outcome(o, now)
	if o.Command == SubmitID && stroke == "enter" {
		c.enter = true
	}
	if !o.Handled {
		c.typed(k)
	}
}

// outcome acts on what the dispatcher made of a key or a timeout: the keys
// of a sequence that matched nothing are typed, a wait is kept for
// Timeout, and a command runs.
func (c *Composer) outcome(o keymap.Outcome, now time.Time) {
	for _, k := range o.Replay {
		c.typed(k)
	}
	c.pending = o.Wait > 0
	if c.pending {
		c.due, c.expire = now.Add(o.Wait), o.Timeout
	}
	if o.Command != "" && !c.done {
		c.command(o.Command)
	}
}

// typed inserts the text a key types, cleaned; a key released, or with a
// modifier besides shift and the locks, types nothing.
func (c *Composer) typed(k termkey.KeyEvent) {
	if k.Release || c.done || k.Text == "" || k.Mod&^typing != 0 {
		return
	}
	c.line.Insert(c.clean(k.Text))
}

// command runs one of the Composer's commands; another is taken and does
// nothing.
func (c *Composer) command(id command.ID) {
	switch id {
	case UpID:
		c.up()
	case DownID:
		c.down()
	case DeleteOrEOFID:
		if len(c.line.Clusters()) == 0 {
			c.eof()
			return
		}
		c.line.Do(id)
	case SubmitID:
		text := c.line.Text()
		if strings.HasSuffix(text, `\`) {
			c.line.SetPos(len(c.line.Clusters()))
			c.line.Do(DeleteBackID)
			c.line.Insert("\n") // a trailing backslash continues the text on a new line
			return
		}
		c.submit(text)
	case NewlineID:
		c.line.Insert("\n")
	case CompleteID:
		c.complete()
	case ClearID:
		if c.clear {
			c.write(c.region.Clear())
			c.write(clearScreen)
		}
	case ExternalID:
		if c.external {
			c.events = append(c.events, ComposerEvent{Kind: EditExternal, Text: c.line.Text()})
		}
	default:
		c.line.Do(id)
	}
}

// laid is the text laid out at the Composer's width, kept between calls: it
// is laid out again only from the first row whose break read a cluster
// changed since the last call, or all of it after a change of width
// (docs/decisions/0023-MADR-line-mode-components.md A12 item 4.3).
func (c *Composer) laid() *laid {
	from, tail := c.line.Edited()
	if !c.layOK || c.layW != c.width {
		from, tail, c.layW, c.layOK = 0, 0, c.width, true
	}
	c.lay.relay(c.method, c.width, c.prompt, c.cont, c.line.Clusters(), c.line.Widths(), from, tail)
	return &c.lay
}

// up moves to the row above, the column nearest the cursor's without
// passing it, or from the first row recalls the older history entry.
func (c *Composer) up() {
	l := c.laid()
	if pl := l.places[c.line.Pos()]; pl.row > 0 {
		c.line.SetPos(nearest(l.places, pl.row-1, pl.col))
		return
	}
	if c.history == nil || c.recalled+1 >= c.history.Len() {
		return
	}
	if c.recalled == -1 {
		c.own = c.line.Text()
	}
	c.recalled++
	c.line.SetText(c.cleanText(c.history.At(c.recalled)))
}

// down moves to the row below, or from the last row recalls the newer
// history entry, and after the newest the user's own text.
func (c *Composer) down() {
	l := c.laid()
	pl := l.places[c.line.Pos()]
	if last := l.places[len(c.line.Clusters())].row; pl.row < last {
		c.line.SetPos(nearest(l.places, pl.row+1, pl.col))
		return
	}
	switch c.recalled {
	case -1:
	case 0:
		c.recalled = -1
		c.line.SetText(c.own)
	default:
		c.recalled--
		c.line.SetText(c.cleanText(c.history.At(c.recalled)))
	}
}

// complete replaces the text with the Completer's completion, cleaned as
// one line, the cursor where it says.
func (c *Composer) complete() {
	if c.completer == nil {
		return
	}
	text := c.line.Text()
	at := len(strings.Join(c.line.Clusters()[:c.line.Pos()], ""))
	line, pos, ok := c.completer.Complete(text, at)
	if !ok {
		return
	}
	pos = min(max(pos, 0), len(line))
	c.line.SetText(c.clean(line[:pos]))
	at = c.line.Pos()
	c.line.Insert(c.clean(line[pos:]))
	c.line.SetPos(at)
}

// rows is the rows that leave the text in the scrollback: the header and
// the text, the cursor at its end.
func (c *Composer) rows() []string {
	rows := c.laid().rows
	if c.head != "" {
		rows = append([]string{c.headRow()}, rows...)
	}
	return rows
}

func (c *Composer) headRow() string {
	if c.width > 0 {
		return sanitize.Truncate(c.head, c.width, c.method, c.glyphs.Ellipsis)
	}
	return c.head
}

// submit leaves the text in the scrollback, adds it to the history, gives
// it, and starts an empty one below.
func (c *Composer) submit(text string) {
	c.write(c.region.Commit(c.rows()))
	if c.history != nil && text != "" {
		c.history.Add(text)
	}
	c.events = append(c.events, ComposerEvent{Kind: EditSubmit, Text: text})
	c.line.SetText("")
	c.recalled, c.own = -1, ""
	c.head = c.readHeader()
}

// eof leaves the text in the scrollback and ends the input.
func (c *Composer) eof() {
	c.write(c.region.Commit(c.rows()))
	c.done, c.pending = true, false
	c.events = append(c.events, ComposerEvent{Kind: EditEOF})
}

// Timeout is when HandleTimeout is next due: the end of the region's wait
// for a cursor report, or of a key sequence's wait, whichever is first.
func (c *Composer) Timeout() (time.Time, bool) {
	t, ok := c.region.Timeout()
	if c.pending && (!ok || c.due.Before(t)) {
		return c.due, true
	}
	return t, ok
}

// HandleTimeout redraws by the region's re-wrap kind when the cursor report
// has not come, and ends a key sequence whose wait has passed, its keys
// typed.
func (c *Composer) HandleTimeout(now time.Time) {
	if c.done {
		return
	}
	c.write(c.region.HandleTimeout(now))
	if c.pending && !now.Before(c.due) {
		c.outcome(c.disp.Expire(c.expire), now)
		c.draw()
	}
}

// Cancel ends the input, as ctrl+d on an empty line does: the host's end
// of input, or a reason of its own
// (docs/decisions/0023-MADR-line-mode-components.md A2 item 2).
func (c *Composer) Cancel() {
	if !c.done {
		c.eof()
	}
}

// NextEvent is the next thing the Composer asks of its host, or EditNone.
func (c *Composer) NextEvent() ComposerEvent {
	if c.next == len(c.events) {
		c.events, c.next = c.events[:0], 0 // drained: the array is reused
		return ComposerEvent{}
	}
	e := c.events[c.next]
	c.events[c.next] = ComposerEvent{}
	c.next++
	return e
}

// View is the rows the Composer draws at width cells, and the cursor's row
// and column: the header, then the prompt and the text in whole words.
// It changes nothing, so a pane draws through it, as ask's Asker.View
// (docs/decisions/0023-MADR-line-mode-components.md A2 item 4).
func (c *Composer) View(width int) (rows []string, row, col int) {
	if width == c.width {
		return c.viewInto(nil, c.laid(), width)
	}
	l := layout(c.method, width, c.prompt, c.cont, c.line.Clusters(), c.line.Widths())
	return c.viewInto(nil, &l, width)
}

// viewInto appends to rows the header, cut to width, and the laid rows, the
// pad row when the cursor is below the last, and gives the cursor's row and
// column.
func (c *Composer) viewInto(rows []string, l *laid, width int) ([]string, int, int) {
	pl := l.places[c.line.Pos()]
	if c.head != "" {
		head := c.head
		if width > 0 {
			head = sanitize.Truncate(head, width, c.method, c.glyphs.Ellipsis)
		}
		rows = append(rows, head)
		pl.row++
	}
	rows = append(rows, l.rows...)
	if len(rows) == pl.row {
		rows = append(rows, l.pad)
	}
	return rows, pl.row, pl.col
}

// draw writes the bytes that show the Composer now, through a rows buffer
// it reuses: the region copies what it keeps.
func (c *Composer) draw() {
	var row, col int
	c.view, row, col = c.viewInto(c.view[:0], c.laid(), c.width)
	c.write(c.region.Draw(c.view, row, col))
}

func (c *Composer) write(s string) {
	if s != "" {
		c.events = append(c.events, ComposerEvent{Kind: EditWrite, Data: s})
	}
}
