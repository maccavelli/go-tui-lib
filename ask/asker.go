package ask

import (
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

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

// typing is the modifiers a key that types its text may hold.
const typing = termkey.ModShift | termkey.ModCapsLock | termkey.ModNumLock | termkey.ModScrollLock

// reasonPath is where a reason's keys are resolved: lineedit's scope,
// whose editing commands edit it and whose submit gives it
// (docs/decisions/0023-MADR-line-mode-components.md A5 item 4).
var reasonPath = keymap.Path{lineedit.ComposerScope}

// Asker asks one question in an inline.Region, as an engine that does no
// I/O (docs/decisions/0023-MADR-line-mode-components.md D2, D4). The host
// calls Resize first, with the terminal's width, which draws the question;
// then HandleInput with each decoded input, HandleTimeout once Timeout's
// deadline has passed, and Cancel at the end of its input; and drains
// NextEvent after each call. Like the region, it is used on one goroutine.
type Asker struct {
	orig, q Question // as given, and cleaned
	method  ansi.Method
	glyphs  glyph.Set
	region  *inline.Region
	catalog msgcat.Catalog
	rules   safetext.Rules
	limit   int
	width   int
	keys    *keymap.Keymap
	disp    *keymap.Dispatcher
	// reading is the answer whose reason is read, cleaned, and chosen
	// the caller's; line is the reason.
	reading, chosen *Answer
	line            *lineinput.Line
	// pending is a key sequence the dispatcher waits on, due when its wait
	// ends, with the timeout Expire takes.
	pending bool
	due     time.Time
	expire  keymap.TimeoutMsg
	done    bool
	resp    Response
	events  []AskerEvent
}

// NewAsker is an Asker for q.
func NewAsker(q Question, o ...Option) *Asker {
	a := &Asker{orig: q, glyphs: glyph.Unicode(), limit: DefaultReasonLimit, rules: safetext.Line()}
	for _, opt := range o {
		opt.apply(a)
	}
	if a.region == nil {
		a.region = inline.NewRegion(inline.WithMethod(a.method))
	}
	if a.keys == nil {
		k, err := keymap.New(keymap.Settings{}, lineedit.DefaultRules())
		if err != nil {
			panic(err) // the defaults are fixed: an error is a bug
		}
		a.keys = k
	}
	a.disp = keymap.NewDispatcher(a.keys)
	a.q = Question{Notes: a.cleanAll(q.Notes), Title: a.clean(q.Title), After: a.cleanAll(q.After), Cancel: a.cleanAnswer(q.Cancel)}
	for _, r := range q.Rows {
		c := Row{Label: a.clean(r.Label)}
		for _, ans := range r.Answers {
			c.Answers = append(c.Answers, a.cleanAnswer(ans))
		}
		a.q.Rows = append(a.q.Rows, c)
	}
	return a
}

// clean is s under the Asker's rules, then the floor's.
func (a *Asker) clean(s string) string { return sanitize.Line(safetext.Clean(s, a.rules)) }

func (a *Asker) cleanAll(ss []string) []string {
	var out []string
	for _, s := range ss {
		out = append(out, a.clean(s))
	}
	return out
}

func (a *Asker) cleanAnswer(ans Answer) Answer {
	ans.Key, ans.Label, ans.Echo = a.clean(ans.Key), a.clean(ans.Label), a.clean(ans.Echo)
	return ans
}

// msg is the Asker's text for id, from its catalogue or the default.
func (a *Asker) msg(id msgcat.MessageID, args ...any) string {
	return a.clean(msgcat.Format(a.catalog, id, Messages()[id], args...))
}

// Resize gives the terminal's width, in cells, and draws the question at
// it: at once the first time, and after the region's redraw on a change.
func (a *Asker) Resize(width int, now time.Time) {
	if a.done {
		return
	}
	a.width = width
	a.write(a.region.Resize(width, now))
	a.draw()
}

// HandleInput takes one decoded input: the cursor report the region waits
// for, a key, or a paste into a reason. A key no answer shows is ignored.
func (a *Asker) HandleInput(in termkey.Input, now time.Time) {
	if a.done {
		return
	}
	out, took := a.region.HandleInput(in, now)
	a.write(out)
	if took {
		return
	}
	switch in.Kind {
	case termkey.InputPaste:
		if a.reading != nil {
			a.line.Insert(a.clean(in.Text))
			a.draw()
		}
	case termkey.InputKey:
		a.key(in.Key, now)
	}
}

func (a *Asker) key(k termkey.KeyEvent, now time.Time) {
	if k.Release {
		return
	}
	if k.Keystroke() == "ctrl+c" {
		a.endCanceled()
		return
	}
	if a.reading != nil {
		o := a.disp.Key(k, reasonPath, when.Map{})
		a.outcome(o, now)
		if !o.Handled {
			a.typed(k)
		}
		if !a.done {
			a.draw()
		}
		return
	}
	if matches(a.q.Cancel.Key, k) {
		a.endCanceled()
		return
	}
	for i, r := range a.q.Rows {
		for j := range r.Answers {
			if !matches(r.Answers[j].Key, k) {
				continue
			}
			chosen := a.orig.Rows[i].Answers[j]
			if !chosen.Reason {
				a.finish(Response{Answer: chosen})
				return
			}
			a.reading, a.chosen = &a.q.Rows[i].Answers[j], &chosen
			a.line = lineinput.New(a.limit, a.method)
			a.draw()
			return
		}
	}
}

// outcome acts on what the dispatcher made of a reason's key or a
// timeout: the keys of a sequence that matched nothing are typed, a wait is
// kept for Timeout, lineedit's submit gives the reason, and its editing
// commands edit it; another command does nothing.
func (a *Asker) outcome(o keymap.Outcome, now time.Time) {
	for _, k := range o.Replay {
		a.typed(k)
	}
	a.pending = o.Wait > 0
	if a.pending {
		a.due, a.expire = now.Add(o.Wait), o.Timeout
	}
	switch {
	case o.Command == "" || a.done:
	case o.Command == lineedit.SubmitID:
		a.finish(Response{Answer: *a.chosen, Reason: strings.TrimSpace(a.line.Text())})
	default:
		a.line.Do(o.Command)
	}
}

// typed inserts the text a key types into the reason, cleaned.
func (a *Asker) typed(k termkey.KeyEvent) {
	if k.Release || a.done || k.Text == "" || k.Mod&^typing != 0 {
		return
	}
	a.line.Insert(a.clean(k.Text))
}

// Timeout is when HandleTimeout is next due: the end of the region's wait
// for a cursor report, or of a key sequence's wait in a reason, whichever
// is first.
func (a *Asker) Timeout() (time.Time, bool) {
	t, ok := a.region.Timeout()
	if a.pending && (!ok || a.due.Before(t)) {
		return a.due, true
	}
	return t, ok
}

// HandleTimeout redraws by the region's re-wrap kind when the cursor
// report has not come, and ends a reason's key sequence whose wait has
// passed, its keys typed.
func (a *Asker) HandleTimeout(now time.Time) {
	if a.done {
		return
	}
	a.write(a.region.HandleTimeout(now))
	if a.pending && !now.Before(a.due) {
		a.outcome(a.disp.Expire(a.expire), now)
		if !a.done {
			a.draw()
		}
	}
}

// Cancel ends the question canceled, as its cancel key does: the host's
// end of input, or a reason of its own, such as a request withdrawn
// (docs/decisions/0023-MADR-line-mode-components.md A2 item 2).
func (a *Asker) Cancel() {
	if !a.done {
		a.endCanceled()
	}
}

func (a *Asker) endCanceled() { a.finish(Response{Answer: a.orig.Cancel, Canceled: true}) }

// NextEvent is the next thing the Asker asks of its host, or AskNone.
func (a *Asker) NextEvent() AskerEvent {
	if len(a.events) == 0 {
		return AskerEvent{}
	}
	e := a.events[0]
	a.events = a.events[1:]
	return e
}

// View is the rows the Asker draws at width cells, and the cursor's row
// and column: the question and its prompt, the reason after it while one
// is read, and once answered, the rows left in the scrollback. It changes
// nothing (docs/decisions/0023-MADR-line-mode-components.md A2 item 4).
func (a *Asker) View(width int) (rows []string, row, col int) {
	if a.done {
		rows = a.final(width)
		last := len(rows) - 1
		col = a.method.StringWidth(rows[last])
		if width > 0 {
			col = min(col, width-1)
		}
		return rows, last, col
	}
	rows = Layout(a.q, width, a.method, a.glyphs)
	last := len(rows) - 1
	lead := a.glyphs.Prompt + " "
	if a.reading == nil {
		col = a.method.StringWidth(lead)
		if width > 0 {
			col = min(col, width-1)
		}
		return rows, last, col
	}
	lead += a.msg(MsgReasonPrompt, a.reading.Label)
	if width > 0 && a.method.StringWidth(lead) > width/2 {
		lead = sanitize.Truncate(lead, width/2, a.method, a.glyphs.Ellipsis)
	}
	lw, avail := a.method.StringWidth(lead), 0
	if width > 0 {
		avail = max(width-a.method.StringWidth(lead), 1)
	}
	text, c := a.line.View(avail)
	rows[last] = lead + text
	return rows, last, lw + c
}

// final is the rows left in the scrollback: the question, and the answer
// in words after the prompt.
func (a *Asker) final(width int) []string {
	rows := Layout(a.q, width, a.method, a.glyphs)
	rows = rows[:len(rows)-1]
	lead := a.glyphs.Prompt + " "
	return append(rows, flow(a.method, width, lead, words(a.echo()), a.method.StringWidth(lead))...)
}

// echo is the answer in words: its echo, or its label; with its reason.
func (a *Asker) echo() string {
	if a.resp.Canceled {
		c := a.q.Cancel
		switch {
		case c.Echo != "":
			return c.Echo
		case c.Label != "":
			return c.Label
		}
		return a.msg(MsgCanceled)
	}
	e := a.clean(a.resp.Answer.Echo)
	if e == "" {
		e = a.clean(a.resp.Answer.Label)
	}
	if a.resp.Answer.Reason {
		return a.msg(MsgReasonEcho, e, a.resp.Reason)
	}
	return e
}

// finish leaves the question and its answer in the scrollback, and gives
// the answer.
func (a *Asker) finish(r Response) {
	a.resp, a.done, a.pending = r, true, false
	a.write(a.region.Commit(a.final(a.width)))
	a.events = append(a.events, AskerEvent{Kind: AskDone, Response: r})
}

// draw writes the bytes that show the Asker's rows now.
func (a *Asker) draw() {
	rows, row, col := a.View(a.width)
	a.write(a.region.Draw(rows, row, col))
}

func (a *Asker) write(s string) {
	if s != "" {
		a.events = append(a.events, AskerEvent{Kind: AskWrite, Data: s})
	}
}
