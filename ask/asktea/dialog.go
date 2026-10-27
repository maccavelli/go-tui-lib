package asktea

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/keymap/keymaptea"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// DefaultID is the overlay's ID, unless WithID gives another.
const DefaultID = "ask"

// border is the cells an overlay's border takes, across and down.
const border = 2

// AnsweredMsg is the question's answer, from the Dialog in overlay ID.
type AnsweredMsg struct {
	ID       string
	Response ask.Response
}

// Dialog is a question in a modal overlay of a workspace. NewDialog builds
// one, and Open shows it. Like the workspace, it is used on the program's
// event loop only.
type Dialog struct {
	w     *workspace.Workspace
	q     ask.Question
	cfg   config
	a     *ask.Asker
	width int  // the content's width, from SizeMsg
	done  bool // answered: it waits to be closed
}

// NewDialog is a Dialog that asks q in w, closed.
func NewDialog(w *workspace.Workspace, q ask.Question, o ...Option) *Dialog {
	return &Dialog{w: w, q: q, cfg: newConfig(o)}
}

// Open shows the question afresh in a modal overlay as wide as the
// workspace, or WithWidth's, and as tall as the question is there, and
// returns the workspace's command for it. The question is measured with
// the workspace's width method and drawn with its theme's glyphs, before
// WithAsk's options.
func (d *Dialog) Open() tea.Cmd {
	o := append([]ask.Option{ask.WithMethod(d.w.WidthMethod()), ask.WithGlyphs(d.w.Theme().Glyphs)}, d.cfg.ask...)
	d.a, d.done = ask.NewAsker(d.q, o...), false
	width := d.cfg.width
	if width <= 0 {
		width, _ = workspace.WidthKey().Get(d.w.WhenContext())
	}
	d.width = max(width-border, 1)
	rows, _, _ := d.a.View(d.width)
	return d.w.Push(workspace.Overlay{
		ID: d.cfg.id, Pane: d, Anchor: d.cfg.anchor, Width: width, Height: len(rows) + border, Modal: true,
	})
}

// Update gives the Asker the keys and pastes, and closes the overlay with
// the answer.
func (d *Dialog) Update(msg tea.Msg) (workspace.Pane, tea.Cmd) {
	if d.a == nil || d.done {
		return d, nil
	}
	switch msg := msg.(type) {
	case workspace.SizeMsg:
		d.width = msg.Width
	case tea.KeyPressMsg:
		d.a.HandleInput(termkey.Input{Kind: termkey.InputKey, Key: keymaptea.FromTea(tea.Key(msg))}, time.Time{})
	case tea.PasteMsg:
		d.a.HandleInput(termkey.Input{Kind: termkey.InputPaste, Text: msg.Content}, time.Time{})
	case workspace.OverlayClosedMsg:
		d.a.Cancel()
	}
	cmd := d.drain()
	return d, cmd
}

// drain takes the Asker's events: its bytes are the line mode's, and are
// dropped; its answer closes the overlay and goes to the program.
func (d *Dialog) drain() tea.Cmd {
	for e := d.a.NextEvent(); e.Kind != ask.AskNone; e = d.a.NextEvent() {
		if e.Kind != ask.AskDone {
			continue
		}
		d.done = true
		id, r := d.cfg.id, e.Response
		return tea.Batch(
			func() tea.Msg { return workspace.Close(id) },
			func() tea.Msg { return AnsweredMsg{ID: id, Response: r} },
		)
	}
	return nil
}

// View is the Asker's rows at the overlay's width.
func (d *Dialog) View(width, height int) string {
	if d.a == nil {
		return ""
	}
	rows, _, _ := d.a.View(width)
	return strings.Join(rows[:min(len(rows), max(height, 0))], "\n")
}

// Cursor is where the Asker's cursor is, in the pane's cells, while the
// question is open.
func (d *Dialog) Cursor() *tea.Cursor {
	if d.a == nil || d.done {
		return nil
	}
	_, row, col := d.a.View(d.width)
	return tea.NewCursor(col, row)
}
