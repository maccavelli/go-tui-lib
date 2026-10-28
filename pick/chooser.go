package pick

import (
	"cmp"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/internal/lineinput"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// queryLimit bounds the query, in bytes: room for fuzzy.MaxQuery clusters
// of one four-byte rune each, past which the matcher ignores the rest
// (docs/decisions/0023-MADR-line-mode-components.md A3.4 item 3).
const queryLimit = 4 * fuzzy.MaxQuery

// minText is the narrowest the query, or an entry's text, is cut to before
// the count, or the entry's detail, gives way, as in the palette's rows.
const minText = 8

// typing is the modifiers a key that types its text may hold.
const typing = termkey.ModShift | termkey.ModCapsLock | termkey.ModNumLock | termkey.ModScrollLock

// path is where the Chooser's keys are resolved: its own scope, then the
// composer's, whose editing commands edit the query
// (docs/decisions/0023-MADR-line-mode-components.md A5 item 4).
var path = keymap.Path{lineedit.ComposerScope, ChooserScope}

// Chooser picks one of a list of entries in an inline.Region, as an engine
// that does no I/O (docs/decisions/0023-MADR-line-mode-components.md D2,
// D5, A3.4). The host calls Resize first, with the terminal's width, which
// draws it; then HandleInput with each decoded input, HandleTimeout once
// Timeout's deadline has passed, and Cancel at the end of its input; and
// drains NextEvent after each call. Like the region, it is used on one
// goroutine.
type Chooser struct {
	orig, entries []Entry // as given, and cleaned
	method        ansi.Method
	glyphs        glyph.Set
	region        *inline.Region
	catalog       msgcat.Catalog
	rules         safetext.Rules
	keys          *keymap.Keymap
	disp          *keymap.Dispatcher
	prompt        string
	initial       string
	height        int // the list's rows, for the whole pick
	indexAbove    int
	index         *fuzzy.Index // nil for a list ranked in full
	width         int

	query    *lineinput.Line
	matches  []int // the entries that match, by index, best first
	capped   bool  // the index yielded as many candidates as it yields at most
	sel, top int   // the selected match, and the first drawn

	// pending is a key sequence the dispatcher waits on, due when its wait
	// ends, with the timeout Expire takes.
	pending bool
	due     time.Time
	expire  keymap.TimeoutMsg

	done   bool
	events []ChooserEvent
}

// NewChooser is a Chooser of entries, in the order given.
func NewChooser(entries []Entry, o ...Option) *Chooser {
	c := &Chooser{
		orig: slices.Clone(entries), glyphs: glyph.Unicode(), rules: safetext.Line(),
		height: DefaultHeight, indexAbove: DefaultIndexAbove,
	}
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
		c.prompt = c.glyphs.Prompt
	}
	c.prompt = c.clean(c.prompt)
	c.entries = make([]Entry, len(entries))
	texts := make([]string, len(entries))
	for i, e := range entries {
		e.Text, e.Detail = c.clean(e.Text), c.clean(e.Detail)
		c.entries[i], texts[i] = e, e.Text
	}
	c.height = min(c.height, len(c.entries))
	if len(c.entries) > c.indexAbove {
		c.index = fuzzy.NewIndex(texts)
	}
	c.query = lineinput.New(queryLimit, c.method)
	c.query.Insert(c.clean(c.initial))
	c.rank()
	return c
}

// clean is s under the Chooser's rules, then the floor's.
func (c *Chooser) clean(s string) string { return sanitize.Line(safetext.Clean(s, c.rules)) }

// msg is the Chooser's text for id, from its catalogue or the default.
func (c *Chooser) msg(id msgcat.MessageID, args ...any) string {
	return c.clean(msgcat.Format(c.catalog, id, Messages()[id], args...))
}

// rank matches the query against the entries' Text, from the first row: an
// empty query keeps them all, in order; a typed one scores them with one
// Matcher, through the index's candidates when there is one, best first by
// grade and then value, ties in the order given, as the palette's engine
// orders hits. Rank is not used: it puts the shorter key first on a tie
// (docs/decisions/0023-MADR-line-mode-components.md A4).
func (c *Chooser) rank() {
	c.sel, c.top, c.capped = 0, 0, false
	c.matches = c.matches[:0]
	q := c.query.Text()
	if q == "" {
		for i := range c.entries {
			c.matches = append(c.matches, i)
		}
		return
	}
	m := fuzzy.NewMatcher(q)
	var candidates iter.Seq[int]
	if c.index != nil {
		ids := slices.Sorted(c.index.Candidates(m)) // the order given, so ties keep it
		c.capped = len(ids) >= fuzzy.DefaultMaxCandidates
		candidates = slices.Values(ids)
	} else {
		candidates = func(yield func(int) bool) {
			for i := range c.entries {
				if !yield(i) {
					return
				}
			}
		}
	}
	type scored struct {
		i int
		s fuzzy.Score
	}
	var found []scored
	for i := range candidates {
		if s, ok := m.Score(c.entries[i].Text); ok {
			found = append(found, scored{i, s})
		}
	}
	slices.SortStableFunc(found, func(a, b scored) int {
		if a.s.Grade != b.s.Grade {
			return cmp.Compare(a.s.Grade, b.s.Grade)
		}
		return cmp.Compare(b.s.Value, a.s.Value) // higher is better
	})
	for _, f := range found {
		c.matches = append(c.matches, f.i)
	}
}

// Resize gives the terminal's width, in cells, and draws the pick at it:
// at once the first time, and after the region's redraw on a change.
func (c *Chooser) Resize(width int, now time.Time) {
	if c.done {
		return
	}
	c.width = width
	c.write(c.region.Resize(width, now))
	c.draw()
}

// HandleInput takes one decoded input: the cursor report the region waits
// for, a key, or a paste into the query.
func (c *Chooser) HandleInput(in termkey.Input, now time.Time) {
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
		c.edit(func() { c.query.Insert(c.clean(in.Text)) })
	case termkey.InputKey:
		c.key(in.Key, now)
	}
	if !c.done {
		c.draw()
	}
}

// key takes one key: ctrl+c cancels; the keymap's keys run their commands;
// any other edits the query.
func (c *Chooser) key(k termkey.KeyEvent, now time.Time) {
	if !k.Release && k.Keystroke() == "ctrl+c" {
		c.endCanceled()
		return
	}
	o := c.disp.Key(k, path, when.Map{})
	c.outcome(o, now)
	if !o.Handled {
		c.typed(k)
	}
}

// outcome acts on what the dispatcher made of a key or a timeout: the keys
// of a sequence that matched nothing go to the query, a wait is kept for
// Timeout, and a command runs.
func (c *Chooser) outcome(o keymap.Outcome, now time.Time) {
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

// typed gives the text k types to the query, cleaned. A key released, or
// one with a modifier besides shift and the locks, types nothing.
func (c *Chooser) typed(k termkey.KeyEvent) {
	if k.Release || c.done || k.Text == "" || k.Mod&^typing != 0 {
		return
	}
	c.edit(func() { c.query.Insert(c.clean(k.Text)) })
}

// edit runs f on the query, and ranks again when it changed the text.
func (c *Chooser) edit(f func()) {
	before := c.query.Text()
	f()
	if c.query.Text() != before {
		c.rank()
	}
}

// command runs one of the Chooser's commands, or one of lineedit's that
// edits the query; another is taken and does nothing.
func (c *Chooser) command(id command.ID) {
	switch id {
	case SelectNextID:
		c.move(1)
	case SelectPrevID:
		c.move(-1)
	case PageNextID:
		c.move(c.height)
	case PagePrevID:
		c.move(-c.height)
	case AcceptID:
		c.accept()
	case CancelID:
		c.endCanceled()
	default:
		c.edit(func() { c.query.Do(id) })
	}
}

// move moves the selection by d matches, stopping at either end, and the
// window with it, so the selection is drawn.
func (c *Chooser) move(d int) {
	if len(c.matches) == 0 {
		return
	}
	c.sel = min(max(c.sel+d, 0), len(c.matches)-1)
	if c.sel < c.top {
		c.top = c.sel
	}
	if c.sel >= c.top+c.height {
		c.top = c.sel - c.height + 1
	}
}

// accept chooses the selected entry; with nothing matched it does nothing.
func (c *Chooser) accept() {
	if len(c.matches) == 0 {
		return
	}
	i := c.matches[c.sel]
	c.finish(c.entries[i].Text, ChooserEvent{Kind: PickDone, Entry: c.orig[i]})
}

func (c *Chooser) endCanceled() {
	c.finish(c.msg(MsgCanceled), ChooserEvent{Kind: PickDone, Canceled: true})
}

// finish leaves one line in the scrollback, the prompt and echo, and gives
// e.
func (c *Chooser) finish(echo string, e ChooserEvent) {
	c.done, c.pending = true, false
	line := c.prompt + " " + echo
	if c.width > 0 {
		line = sanitize.Truncate(line, c.width, c.method, c.glyphs.Ellipsis)
	}
	c.write(c.region.Commit([]string{line}))
	c.events = append(c.events, e)
}

// Timeout is when HandleTimeout is next due: the end of the region's wait
// for a cursor report, or of a key sequence's wait, whichever is first.
func (c *Chooser) Timeout() (time.Time, bool) {
	t, ok := c.region.Timeout()
	if c.pending && (!ok || c.due.Before(t)) {
		return c.due, true
	}
	return t, ok
}

// HandleTimeout redraws by the region's re-wrap kind when the cursor report
// has not come, and ends a key sequence whose wait has passed, its keys
// given to the query.
func (c *Chooser) HandleTimeout(now time.Time) {
	if c.done {
		return
	}
	c.write(c.region.HandleTimeout(now))
	if c.pending && !now.Before(c.due) {
		c.outcome(c.disp.Expire(c.expire), now)
		c.draw()
	}
}

// Cancel ends the pick canceled, as esc does: the host's end of input, or
// a reason of its own (docs/decisions/0023-MADR-line-mode-components.md A2
// item 2).
func (c *Chooser) Cancel() {
	if !c.done {
		c.endCanceled()
	}
}

// NextEvent is the next thing the Chooser asks of its host, or PickNone.
func (c *Chooser) NextEvent() ChooserEvent {
	if len(c.events) == 0 {
		return ChooserEvent{}
	}
	e := c.events[0]
	c.events = c.events[1:]
	return e
}

// draw writes the bytes that show the pick now.
func (c *Chooser) draw() {
	rows, row, col := c.view(c.width)
	c.write(c.region.Draw(rows, row, col))
}

// view is the rows drawn at width cells, and the cursor's row and column:
// the query row, then the list's rows, blank past the matches.
func (c *Chooser) view(width int) (rows []string, row, col int) {
	m := c.method
	lead := c.prompt + " "
	if width > 0 && m.StringWidth(lead) > width/2 {
		lead = sanitize.Truncate(lead, width/2, m, c.glyphs.Ellipsis)
	}
	lw, count := m.StringWidth(lead), c.count()
	avail := 0 // the query's cells, its cursor's counted; 0 shows it all
	if width > 0 {
		avail = width - lw - 1 - m.StringWidth(count)
		if avail < minText {
			count, avail = "", max(width-lw, 1)
		}
	}
	text, qc := c.query.View(avail)
	if j := joined(m, lead, text); j > 0 && avail > 0 {
		text, qc = c.query.View(max(avail-j, 1)) // a mark that joins the lead's space
		if qc > 0 {
			qc += joined(m, lead, text)
		}
	}
	first := lead + text
	if count != "" {
		first = right(m, first, count, width)
	}
	rows = append(rows, first)
	for i := range c.height {
		at := c.top + i
		if at >= len(c.matches) {
			rows = append(rows, "")
			continue
		}
		rows = append(rows, c.row(c.entries[c.matches[at]], at == c.sel, width))
	}
	return rows, 0, lw + qc
}

// count is the selected match's place and the matches', or that there are
// none.
func (c *Chooser) count() string {
	n := len(c.matches)
	switch {
	case n == 0:
		return c.msg(MsgNoResults)
	case c.capped:
		return c.msg(MsgCountMore, c.sel+1, n)
	}
	return c.msg(MsgCount, c.sel+1, n)
}

// row is one entry in width cells, as the palette draws a row: the focus
// glyph when it is selected, or as many spaces, the Text cut to fit, and
// the Detail at the right, left out when the Text would keep fewer than
// minText cells.
func (c *Chooser) row(e Entry, selected bool, width int) string {
	m, g := c.method, c.glyphs
	marker := strings.Repeat(" ", m.StringWidth(g.Focus))
	if selected {
		marker = g.Focus
	}
	marker += " "
	text, detail := e.Text, e.Detail
	if width > 0 {
		room := width - m.StringWidth(marker)
		if detail != "" && room-1-m.StringWidth(detail) < minText {
			detail = ""
		}
		if detail != "" {
			room -= 1 + m.StringWidth(detail)
		}
		room = max(room, 0)
		text = sanitize.Truncate(text, room, m, g.Ellipsis)
		for mw := m.StringWidth(marker); room > 0 && m.StringWidth(marker+text) > mw+room; {
			room-- // a mark that joins the marker's space takes a cell of the room
			text = sanitize.Truncate(e.Text, room, m, g.Ellipsis)
		}
	}
	out := marker + text
	if detail != "" {
		out = right(m, out, detail, width)
	}
	return out
}

// joined is the cells a and b take drawn together beyond what they take
// apart: a mark with no base that starts b joins the space a ends with
// (0023-MADR A5 item 7).
func joined(m ansi.Method, a, b string) int {
	return max(m.StringWidth(a+b)-m.StringWidth(a)-m.StringWidth(b), 0)
}

// right is left, then right at the right of width cells, after at least
// one space, measured as they are drawn; right is left out when it does not
// fit. A width of 0 or less puts one space between them.
func right(m ansi.Method, left, right string, width int) string {
	if width <= 0 {
		return left + " " + right
	}
	for pad := max(width-m.StringWidth(left)-m.StringWidth(right), 1); pad >= 1; pad-- {
		if row := left + strings.Repeat(" ", pad) + right; m.StringWidth(row) <= width {
			return row
		}
	}
	return left
}

func (c *Chooser) write(s string) {
	if s != "" {
		c.events = append(c.events, ChooserEvent{Kind: PickWrite, Data: s})
	}
}
