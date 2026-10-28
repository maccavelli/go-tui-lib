package lineedittea

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/keymap/keymaptea"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// DefaultID is the pane's name in its EventMsgs, unless WithID gives
// another.
const DefaultID = "composer"

// EventMsg is what the pane's Composer gave: EditSubmit, EditEOF,
// EditInterrupt or EditExternal, never a write.
type EventMsg struct {
	ID    string
	Event lineedit.ComposerEvent
}

// tickMsg ends a key sequence's wait in the pane p.
type tickMsg struct{ p *ComposerPane }

// ComposerPane is a lineedit.Composer in a workspace pane. NewComposerPane
// builds one. Like the workspace, it is used on the program's event loop
// only.
type ComposerPane struct {
	cfg           config
	c             *lineedit.Composer
	width, height int // the pane's, from SizeMsg
}

// NewComposerPane is a pane over a Composer measured with w's width method
// and drawn with its theme's glyphs, as they are now, before WithComposer's
// options; a program builds w first, then puts the pane in it with
// Workspace.SetPane. Its region asks the terminal nothing: the workspace
// draws the pane.
func NewComposerPane(w *workspace.Workspace, o ...Option) *ComposerPane {
	p := &ComposerPane{cfg: newConfig(o)}
	opts := append([]lineedit.Option{
		lineedit.WithMethod(w.WidthMethod()), lineedit.WithGlyphs(w.Theme().Glyphs),
		lineedit.WithRegion(inline.NewRegion(inline.WithoutCursorQuery())),
	}, p.cfg.composer...)
	p.c = lineedit.NewComposer(opts...)
	return p
}

// Composer is the pane's Composer, for SetText after the external editor.
func (p *ComposerPane) Composer() *lineedit.Composer { return p.c }

// Update gives the Composer the pane's size, its keys and its pastes, and
// sends what the Composer gives as EventMsgs.
func (p *ComposerPane) Update(msg tea.Msg) (workspace.Pane, tea.Cmd) {
	now := time.Now()
	switch msg := msg.(type) {
	case workspace.SizeMsg:
		p.width, p.height = msg.Width, msg.Height
		p.c.Resize(msg.Width, now)
	case tea.KeyPressMsg:
		p.c.HandleInput(termkey.Input{Kind: termkey.InputKey, Key: keymaptea.FromTea(tea.Key(msg))}, now)
	case tea.PasteMsg:
		p.c.HandleInput(termkey.Input{Kind: termkey.InputPaste, Text: msg.Content}, now)
	case tickMsg:
		if msg.p != p {
			return p, nil
		}
		p.c.HandleTimeout(now)
	default:
		return p, nil
	}
	cmd := p.drain(now)
	return p, cmd
}

// drain takes the Composer's events: its bytes are the line mode's, and are
// dropped; the rest go to the program. A key sequence's wait is a tick.
func (p *ComposerPane) drain(now time.Time) tea.Cmd {
	var cmds []tea.Cmd
	for e := p.c.NextEvent(); e.Kind != lineedit.EditNone; e = p.c.NextEvent() {
		if e.Kind == lineedit.EditWrite {
			continue
		}
		msg := EventMsg{ID: p.cfg.id, Event: e}
		cmds = append(cmds, func() tea.Msg { return msg })
	}
	if due, ok := p.c.Timeout(); ok {
		cmds = append(cmds, tea.Tick(due.Sub(now), func(time.Time) tea.Msg { return tickMsg{p: p} }))
	}
	return tea.Batch(cmds...)
}

// window is the Composer's rows at width, at most height of them around
// the cursor's, and the cursor's row among them and its column.
func (p *ComposerPane) window(width, height int) ([]string, int, int) {
	rows, row, col := p.c.View(width)
	if height > 0 && len(rows) > height {
		from := min(max(row-height+1, 0), len(rows)-height)
		rows, row = rows[from:from+height], row-from
	}
	return rows, row, col
}

// View is the Composer's rows at the pane's size.
func (p *ComposerPane) View(width, height int) string {
	rows, _, _ := p.window(width, height)
	return strings.Join(rows, "\n")
}

// Cursor is where the Composer's cursor is, in the pane's cells.
func (p *ComposerPane) Cursor() *tea.Cursor {
	_, row, col := p.window(p.width, p.height)
	return tea.NewCursor(col, row)
}
