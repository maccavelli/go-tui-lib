package lineedittea_test

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/lineedit/lineedittea"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// prog plays a program's event loop around a workspace whose one pane is
// the composer, inside a synctest bubble: each command runs on a goroutine
// of its own, and settle gives their messages to the workspace until the
// bubble is idle.
type prog struct {
	w     *workspace.Workspace
	p     *lineedittea.ComposerPane
	mu    sync.Mutex
	queue []tea.Msg
	seen  []tea.Msg
}

func newProg(width, height int, o ...lineedittea.Option) *prog {
	g := &prog{}
	g.w = workspace.New(layout.Pane{ID: "main"}, map[layout.PaneID]workspace.Pane{"main": placeholder{}},
		workspace.WithTheme(theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII())), workspace.WithoutBackgroundQuery())
	g.p = lineedittea.NewComposerPane(g.w, o...)
	g.run(g.w.Init())
	g.run(g.w.SetPane("main", g.p)) // as a program puts the pane in: the workspace first, then the pane
	g.send(tea.WindowSizeMsg{Width: width, Height: height})
	g.settle()
	return g
}

// placeholder holds the pane's place until SetPane puts the pane there.
type placeholder struct{}

func (placeholder) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return placeholder{}, nil }
func (placeholder) View(int, int) string                     { return "" }

func (g *prog) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		if m := cmd(); m != nil {
			g.mu.Lock()
			g.queue = append(g.queue, m)
			g.mu.Unlock()
		}
	}()
}

func (g *prog) send(m tea.Msg) {
	g.seen = append(g.seen, m)
	g.run(g.w.Update(m))
}

func (g *prog) settle() {
	for {
		synctest.Wait()
		g.mu.Lock()
		q := g.queue
		g.queue = nil
		g.mu.Unlock()
		if len(q) == 0 {
			return
		}
		for _, m := range q {
			if b, ok := m.(tea.BatchMsg); ok {
				for _, c := range b {
					g.run(c)
				}
				continue
			}
			g.send(m)
		}
	}
}

// typed sends each character of s as a key, settling after each.
func (g *prog) typed(s string) {
	for _, r := range s {
		g.press(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func (g *prog) press(k tea.KeyPressMsg) {
	g.send(k)
	g.settle()
}

// events is the EventMsgs the program was sent.
func (g *prog) events() []lineedit.ComposerEvent {
	var out []lineedit.ComposerEvent
	for _, m := range g.seen {
		if e, ok := m.(lineedittea.EventMsg); ok {
			if e.ID != lineedittea.DefaultID {
				panic("an EventMsg from another pane: " + e.ID)
			}
			out = append(out, e.Event)
		}
	}
	return out
}

// TestComposerPane: keys and a paste reach the Composer; enter's submission
// comes to the program as one EventMsg, and the pane draws the empty
// prompt after it.
func TestComposerPane(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(40, 5)
		g.typed("hello ")
		g.send(tea.PasteMsg{Content: "world"})
		g.settle()
		if got := g.p.View(40, 5); got != "> hello world" {
			t.Errorf("drawn %q", got)
		}
		g.press(tea.KeyPressMsg{Code: tea.KeyEnter})
		ev := g.events()
		if len(ev) != 1 || ev[0].Kind != lineedit.EditSubmit || ev[0].Text != "hello world" {
			t.Fatalf("events %+v; want one EditSubmit of hello world", ev)
		}
		if got := g.p.View(40, 5); got != "> " {
			t.Errorf("after the submit: %q", got)
		}
		g.press(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if ev := g.events(); len(ev) != 2 || ev[1].Kind != lineedit.EditInterrupt {
			t.Errorf("ctrl+c: events %+v; want EditInterrupt", ev)
		}
	})
}

// TestComposerPaneWindow: a text taller than the pane is drawn from the
// rows around the cursor, and the cursor is placed among them.
func TestComposerPaneWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(40, 10)
		g.typed("the quick brown fox jumps")
		g.p.Update(workspace.SizeMsg{Width: 12, Height: 2}) // the pane's own size, which View and Cursor share
		if got := g.p.View(12, 2); got != "  brown fox\n  jumps" {
			t.Errorf("drawn %q; want the last two rows", got)
		}
		if c := g.p.Cursor(); c == nil || c.Y != 1 || c.X != 7 {
			t.Errorf("cursor %+v; want 7,1", c)
		}
	})
}

// TestComposerPaneExternal: ctrl+x ctrl+e asks for the external editor only
// WithExternalEditor, and SetText gives the text back; a lone ctrl+x waits
// the keymap's timeout, then does nothing.
func TestComposerPaneExternal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(40, 5, lineedittea.WithComposer(lineedit.WithExternalEditor()))
		g.typed("draft")
		g.press(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
		g.press(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
		ev := g.events()
		if len(ev) != 1 || ev[0].Kind != lineedit.EditExternal || ev[0].Text != "draft" {
			t.Fatalf("events %+v; want one EditExternal of draft", ev)
		}
		g.p.Composer().SetText("edited")
		if got := g.p.View(40, 5); got != "> edited" {
			t.Errorf("after SetText: %q", got)
		}
		g.press(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
		time.Sleep(1500 * time.Millisecond) // past the keymap's one-second wait
		g.settle()
		if !slices.ContainsFunc(g.seen, func(m tea.Msg) bool { return strings.Contains(strings.ToLower(typeName(m)), "tick") }) {
			t.Error("a lone ctrl+x set no tick for its wait")
		}
		g.typed("!")
		if got := g.p.View(40, 5); got != "> edited!" {
			t.Errorf("after the wait: %q", got)
		}
	})
}

// typeName is m's type, for a message the test cannot name.
func typeName(m tea.Msg) string { return fmt.Sprintf("%T", m) }
