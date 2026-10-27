package palette

import (
	"context"
	"encoding/json/jsontext"
	"slices"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/command/commandtea"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/when"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// maxQuery is the most the input takes, in characters: the matcher reads
// a query's first 256 graphemes, and a grapheme is at least one character
// (0008-MADR A3 item 10).
const maxQuery = 256

// Picker is a command palette: an input, the hits its providers stream as
// the query changes, and a help line, drawn in a modal overlay of a
// workspace (0008-MADR §4, §7, A3). NewPicker builds one, Register adds its
// providers, and Open shows it.
//
// A Picker is the overlay's workspace.Pane. It takes its theme, its width
// method and its keymap from the workspace whenever it draws, so it follows
// a change of either without reopening. Like the workspace, it is used on
// the program's event loop only.
type Picker struct {
	w     *workspace.Workspace
	cfg   config
	eng   *engine
	input textinput.Model

	id            string      // the overlay's ID while open, else ""
	path          keymap.Path // the key path when it opened, for hints
	sel           int         // the selected row
	top           int         // the first result line drawn
	width, height int         // the content size, from SizeMsg
	done          bool        // a hit was accepted; it waits to be closed
	matched       int         // rows matched for their ranges, for a test
	args          *stage      // the argument stage, while a hit's parameters are asked for
}

// NewPicker returns a picker for w, closed, with no providers.
func NewPicker(w *workspace.Workspace, opts ...Option) *Picker {
	cfg := newConfig(opts)
	in := textinput.New()
	in.CharLimit = maxQuery
	in.Placeholder = cfg.placeholder
	in.SetVirtualCursor(false)
	return &Picker{w: w, cfg: cfg, eng: newEngine(cfg), input: in}
}

// Register adds a provider, which searches only where a.When holds, against
// the when values at the time the picker opens. A When that does not parse
// is an error, and the provider is not added. One added while the picker
// is open searches from the next keystroke.
func (p *Picker) Register(pr Provider, a Availability) error { return p.eng.add(pr, a) }

// Open shows the picker in a modal overlay with an empty query, and returns
// the workspace's command for it. It reads the key path and the when values
// first, so the hints, and the providers' availability, are those of where
// the user was. Opening it while it is open opens it afresh: the last
// opening's searches are cancelled and its providers stopped.
//
// A picker made WithoutInput opens non-modal instead, by default as the
// overlay "completion" under the focused pane's cursor: the pane keeps the
// keyboard and drives it with SetQuery.
func (p *Picker) Open(opts ...OverlayOption) tea.Cmd {
	o := newOverlayConfig(opts, p.cfg.noInput)
	path, ctx := p.w.KeyPath(), p.w.WhenContext()
	var cmds []tea.Cmd
	if p.id != "" && p.id != o.id {
		cmds = append(cmds, p.w.Update(workspace.Close(p.id)))
	}
	p.query()
	p.sel, p.top, p.done = 0, 0, false
	cmds = append(cmds, p.w.Push(workspace.Overlay{
		ID: o.id, Pane: p, Anchor: o.anchor, Width: o.width, Height: o.height, Modal: !p.cfg.noInput,
	}))
	p.id, p.path = o.id, path
	for _, src := range p.eng.sources {
		if op, ok := src.p.(Opener); ok {
			op.Opened(ctx)
		}
	}
	p.eng.open(ctx)
	return tea.Batch(append(cmds, p.eng.query(""))...)
}

// Opener is a provider that reads something on the event loop when the
// picker opens, before any search: the when values, which its Discover is
// not given, or the workspace's panes, which only the loop may read. Open
// calls Opened on each registered provider that has it. FromRegistry,
// Panes and Recents have it, so a program's own wrapper of one forwards
// Opened, as Recents does
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D4).
type Opener interface {
	Opened(c when.Context)
}

// OpenCommand is a command that opens this picker, for the program's
// registry: palette.open is the ID DefaultRules binds ctrl+p to. Since no
// picker is open to take it, its key comes to the program as a
// workspace.RequestMsg, and the program's Dispatch runs it. It runs on the
// event loop (command.Loop) and changes only presentation (command.UI);
// opts come after those, to add a slash name or a category. It panics when
// command.New refuses id, title or opts, as command.MustNew does: they are
// the program's fixed definitions.
func (p *Picker) OpenCommand(id command.ID, title string, opts ...command.Option) command.Command {
	o := append([]command.Option{command.WithDanger(command.UI), command.WithMode(command.Loop)}, opts...)
	return command.MustNew(id, title, func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
		return command.Result{Cmd: commandtea.Effect(p.Open())}, nil
	}, o...)
}

// KeyContexts names the picker's key scope, "kind:picker", which
// DefaultRules binds its moves in.
func (p *Picker) KeyContexts() []command.Scope { return []command.Scope{PickerScope} }

// ConsumesCommand takes the picker's own commands: the moves, accept and
// complete, so they run in the order their keys were pressed.
func (p *Picker) ConsumesCommand(id command.ID) bool {
	switch id {
	case SelectNextID, SelectPrevID, PageNextID, PagePrevID, AcceptID, CompleteID:
		return true
	}
	return false
}

// ConsumesEsc takes esc while the query is not empty, to clear it; then
// esc closes the picker, as the workspace closes any overlay. A picker
// WithoutInput never takes it, so esc closes the pop-up.
func (p *Picker) ConsumesEsc() bool { return !p.cfg.noInput && p.input.Value() != "" }

// Cursor is the input's cursor, in the picker's cells: the input is its
// first row. A picker WithoutInput has none.
func (p *Picker) Cursor() *tea.Cursor {
	if p.cfg.noInput {
		return nil
	}
	p.restyle()
	return p.input.Cursor()
}

// setQueryMsg is SetQuery's message.
type setQueryMsg struct{ q string }

// SetQuery wraps a query for a picker, which searches for it as if it had
// been typed: how an editor pane drives a completion pop-up as its "@" or
// "/" token changes, through workspace.SendOverlay or To
// (0008-MADR A3 item 6).
func SetQuery(q string) tea.Msg { return setQueryMsg{q: q} }

// SelectedMsg is a hit the user accepted, from the picker in the overlay
// ID. A picker sends it on every accept; a completion pop-up's hit is the
// pane's to use, and its Run is not called.
type SelectedMsg struct {
	ID  string // the overlay's ID
	Hit Hit
}

// Update handles the workspace's messages, the keys and commands that
// reach the picker, the mouse over it, and its own searches' batches.
func (p *Picker) Update(msg tea.Msg) (workspace.Pane, tea.Cmd) {
	switch m := msg.(type) {
	case workspace.SizeMsg:
		p.width, p.height = m.Width, m.Height
		p.scroll()
	case workspace.PaneFocusMsg:
		return p, p.input.Focus()
	case workspace.PaneBlurMsg:
		p.input.Blur()
	case workspace.OverlayClosedMsg:
		p.eng.close()
		p.id, p.args = "", nil
	case hitsMsg, startMsg:
		cmd := p.eng.update(msg)
		p.clamp()
		return p, cmd
	}
	if p.id == "" || p.done {
		return p, nil // closed, or accepted and waiting to be: no more input
	}
	cmd := p.handle(msg)
	return p, cmd
}

// handle takes what the user does while the picker is open: its own
// commands, keys, paste and the mouse.
func (p *Picker) handle(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case setQueryMsg:
		return p.set(m.q)
	case workspace.CommandMsg:
		return p.command(m.Request.ID)
	case tea.KeyPressMsg:
		if m.Code == tea.KeyEscape {
			return p.clear()
		}
		return p.edit(m)
	case tea.PasteMsg:
		return p.edit(m)
	case tea.MouseWheelMsg:
		p.wheel(m)
	case tea.MouseClickMsg:
		return p.click(m)
	}
	return nil
}

// command runs one of the picker's own commands.
func (p *Picker) command(id command.ID) tea.Cmd {
	switch id {
	case SelectNextID:
		p.move(1)
	case SelectPrevID:
		p.move(-1)
	case PageNextID:
		p.move(p.visible())
	case PagePrevID:
		p.move(-p.visible())
	case AcceptID:
		return p.accept()
	case CompleteID:
		return p.complete()
	}
	return nil
}

// clear empties a non-empty query, which esc reaches the picker for.
func (p *Picker) clear() tea.Cmd {
	if p.input.Value() == "" {
		return nil
	}
	p.input.Reset()
	return p.requery()
}

// edit gives msg to the input, and searches again when it changed the
// query.
func (p *Picker) edit(msg tea.Msg) tea.Cmd {
	before := p.input.Value()
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	if p.input.Value() == before {
		return cmd
	}
	return tea.Batch(cmd, p.requery())
}

// requery searches for the input's query, from the first row; in the
// argument stage it filters the asked parameter's values, and clears the
// last refusal.
func (p *Picker) requery() tea.Cmd {
	p.sel, p.top = 0, 0
	if p.args != nil {
		p.args.reason = ""
		p.args.filter(p.input.Value())
		if p.input.Value() == "" {
			p.sel = p.args.first()
		}
		return nil
	}
	return p.eng.query(p.input.Value())
}

// rows is the list: the providers' hits, or in the argument stage the
// asked parameter's values.
func (p *Picker) rows() []row {
	if p.args != nil {
		return p.args.shown
	}
	return p.eng.rows()
}

// listing reports whether the rows are the providers' prefixes, for "?".
func (p *Picker) listing() bool { return p.args == nil && p.eng.prefix }

// matcher is what the drawn rows are matched with for their highlights.
func (p *Picker) matcher() *fuzzy.Matcher {
	if p.args != nil {
		return p.args.matcher
	}
	return p.eng.search.Matcher
}

// selected is the selected row, if there is one.
func (p *Picker) selected() (row, bool) {
	rows := p.rows()
	if p.sel < 0 || p.sel >= len(rows) {
		return row{}, false
	}
	return rows[p.sel], true
}

// accept takes the selected hit (0008-MADR §6, A3 items 3 to 6). A hit
// with parameters to ask for opens the argument stage; one with a
// parameter the stage cannot ask for falls through to SelectedMsg, and
// nothing runs. Any other is finished. A failure's row and a disabled hit
// do nothing; the help line shows a disabled hit's reason. One of "?"'s
// rows types its prefix. In the argument stage, accept takes the asked
// parameter's value.
func (p *Picker) accept() tea.Cmd {
	if p.args != nil {
		return p.give()
	}
	r, ok := p.selected()
	switch {
	case !ok, r.failed, r.hit.Disabled != "":
		return nil
	case p.eng.prefix:
		return p.set(r.hit.Text)
	}
	if len(r.hit.Params) == 0 || p.cfg.noInput {
		return p.finish(r.hit, nil, true)
	}
	ps, ok := asked(r.hit.Params)
	switch {
	case !ok:
		return p.finish(r.hit, nil, false)
	case len(ps) == 0:
		return p.finish(r.hit, nil, true)
	}
	p.eng.reset() // the providers' searches are done with
	p.args = &stage{hit: r.hit, params: ps}
	p.ask()
	return nil
}

// ask readies the input for the stage's next parameter: empty, with its
// placeholder, masked when it is secret, and with no limit of the
// picker's, since the registry checks a value's length.
func (p *Picker) ask() {
	st := p.args
	st.begin()
	par := st.param()
	p.input.Reset()
	p.input.CharLimit = 0
	p.input.Placeholder = placeholder(par)
	p.input.EchoMode = textinput.EchoNormal
	if par.Secret {
		p.input.EchoMode = textinput.EchoPassword
	}
	p.sel, p.top = st.first(), 0
	p.scroll()
}

// query readies the input for a query: empty, with the picker's
// placeholder, and echoed.
func (p *Picker) query() {
	p.args = nil
	p.input.Reset()
	p.input.CharLimit = maxQuery
	p.input.Placeholder = p.cfg.placeholder
	p.input.EchoMode = textinput.EchoNormal
}

// give takes the asked parameter's value, and asks for the next, or runs
// the hit with every value given as one JSON object. A refused value stays
// in the input, with the reason in the help line. Values that cannot be
// encoded, which only an enum of the schema's could give, fall through to
// SelectedMsg, as a parameter the stage cannot ask for does.
func (p *Picker) give() tea.Cmd {
	st := p.args
	if !st.give(p.input.Value(), p.sel) {
		return nil
	}
	if !st.done() {
		p.ask()
		return nil
	}
	args, err := st.args()
	return p.finish(st.hit, args, err == nil)
}

// finish ends the picker with h: it records h's ID in the recents, sends
// SelectedMsg, runs h's Run with args on the event loop when run is true
// and the picker is not a completion pop-up, and returns a command that
// closes the picker, which takes no more input until it is closed.
func (p *Picker) finish(h Hit, args jsontext.Value, run bool) tea.Cmd {
	p.done = true
	if p.cfg.recents != nil {
		p.cfg.recents.Record(h.ID)
	}
	var c tea.Cmd
	if run && h.Run != nil && !p.cfg.noInput {
		c = h.Run(args)
	}
	id := p.id
	return tea.Batch(
		func() tea.Msg { return SelectedMsg{ID: id, Hit: h} },
		c,
		func() tea.Msg { return workspace.Close(id) },
	)
}

// complete makes the query the selected hit's text, under the prefix the
// query had.
func (p *Picker) complete() tea.Cmd {
	r, ok := p.selected()
	if !ok || r.failed {
		return nil
	}
	if p.args != nil || p.eng.prefix {
		return p.set(r.hit.Text)
	}
	return p.set(p.eng.used + r.hit.Text)
}

// set makes s the query, with the cursor at its end, and searches for it.
func (p *Picker) set(s string) tea.Cmd {
	p.input.SetValue(s)
	p.input.CursorEnd()
	return p.requery()
}

// move moves the selection by d rows, stopping at either end.
func (p *Picker) move(d int) {
	n := len(p.rows())
	if n == 0 {
		return
	}
	p.sel = min(max(p.sel+d, 0), n-1)
	p.scroll()
}

// clamp keeps the selection on a row after the rows change.
func (p *Picker) clamp() {
	p.sel = max(min(p.sel, len(p.rows())-1), 0)
	p.scroll()
}

// visible is how many result lines the picker draws: its height less the
// input, if it has one, and the help line.
func (p *Picker) visible() int { return max(p.height-1-p.inputRows(), 0) }

// inputRows is the rows the input takes: one, or none WithoutInput.
func (p *Picker) inputRows() int {
	if p.cfg.noInput {
		return 0
	}
	return 1
}

// scroll moves the window so the selected row's line is drawn.
func (p *Picker) scroll() {
	ls := p.lines()
	at := lineOf(ls, p.sel)
	if v := p.visible(); v > 0 && at >= 0 {
		switch {
		case at < p.top:
			p.top = at
			if at > 0 && ls[at-1].row < 0 {
				p.top-- // keep the row's group header with it
			}
		case at >= p.top+v:
			p.top = at - v + 1
		}
	}
	p.top = max(min(p.top, len(ls)-p.visible()), 0)
}

// wheel scrolls the results by a line, and keeps the selection on a line
// that is drawn.
func (p *Picker) wheel(m tea.MouseWheelMsg) {
	ls, v := p.lines(), p.visible()
	switch m.Button {
	case tea.MouseWheelUp:
		p.top--
	case tea.MouseWheelDown:
		p.top++
	default:
		return
	}
	p.top = max(min(p.top, len(ls)-v), 0)
	at := lineOf(ls, p.sel)
	switch {
	case at < p.top:
		p.sel = firstRow(ls[p.top:min(p.top+v, len(ls))], p.sel)
	case at >= p.top+v:
		p.sel = lastRow(ls[p.top:min(p.top+v, len(ls))], p.sel)
	}
}

// click runs the row under a left click; the input row and the help line
// do nothing.
func (p *Picker) click(m tea.MouseClickMsg) tea.Cmd {
	first := p.inputRows()
	if m.Button != tea.MouseLeft || m.Y < first || m.Y >= first+p.visible() {
		return nil
	}
	ls := p.lines()
	at := p.top + m.Y - first
	if at >= len(ls) || ls[at].row < 0 {
		return nil
	}
	p.sel = ls[at].row
	return p.accept()
}

// line is one result line: a row, or, with ByProvider, a group's header,
// whose row is -1.
type line struct {
	row    int
	header string
}

// lines is the result lines for the rows: each row, after a header where
// its group starts under ByProvider.
func (p *Picker) lines() []line {
	rows := p.rows()
	out := make([]line, 0, len(rows))
	group := ""
	for i, r := range rows {
		if p.cfg.grouping == ByProvider && p.args == nil && !p.eng.prefix && (i == 0 || r.hit.Group != group) {
			out = append(out, line{row: -1, header: r.hit.Group})
		}
		group = r.hit.Group
		out = append(out, line{row: i})
	}
	return out
}

// lineOf is the index of row's line, or -1.
func lineOf(ls []line, row int) int {
	for i, l := range ls {
		if l.row == row {
			return i
		}
	}
	return -1
}

// firstRow and lastRow are the first and last row among ls, or def when
// ls holds none.
func firstRow(ls []line, def int) int {
	for _, l := range ls {
		if l.row >= 0 {
			return l.row
		}
	}
	return def
}

func lastRow(ls []line, def int) int {
	for _, l := range slices.Backward(ls) {
		if l.row >= 0 {
			return l.row
		}
	}
	return def
}
