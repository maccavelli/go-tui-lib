package asktea_test

import (
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/ask/asktea"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// blank is the workspace's one pane, beneath the dialog.
type blank struct{}

func (blank) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return blank{}, nil }
func (blank) View(int, int) string                     { return "" }

// prog plays a program's event loop around a workspace, inside a synctest
// bubble: each command runs on a goroutine of its own, and settle gives
// their messages to the workspace until the bubble is idle.
type prog struct {
	w     *workspace.Workspace
	mu    sync.Mutex
	queue []tea.Msg
	seen  []tea.Msg
}

func newProg() *prog {
	g := &prog{w: workspace.New(layout.Pane{ID: "main"}, map[layout.PaneID]workspace.Pane{"main": blank{}},
		workspace.WithTheme(theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII())), workspace.WithoutBackgroundQuery())}
	g.run(g.w.Init())
	g.send(tea.WindowSizeMsg{Width: 80, Height: 24})
	g.settle()
	return g
}

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

// press sends each key, settling after each.
func (g *prog) press(keys ...tea.KeyPressMsg) {
	for _, k := range keys {
		g.send(k)
		g.settle()
	}
}

func (g *prog) answers() []asktea.AnsweredMsg {
	var out []asktea.AnsweredMsg
	for _, m := range g.seen {
		if a, ok := m.(asktea.AnsweredMsg); ok {
			out = append(out, a)
		}
	}
	return out
}

func text(s string) tea.KeyPressMsg { return tea.KeyPressMsg{Code: rune(s[0]), Text: s} }

func question() ask.Question {
	return ask.Question{
		Title: "Run make?",
		Rows: []ask.Row{{Answers: []ask.Answer{
			{ID: "y", Key: "y", Label: "allow"},
			{ID: "why", Key: "f", Label: "reject and say why", Reason: true},
		}}},
		Cancel: ask.Answer{ID: "cancel", Key: "c", Label: "cancel"},
	}
}

// TestDialog: Open shows the question in a modal overlay, drawn as the
// line mode lays it out at the overlay's width; its key answers, and the
// overlay closes with the answer sent once.
func TestDialog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg()
		d := asktea.NewDialog(g.w, question())
		g.run(d.Open())
		g.settle()
		if !slices.Contains(g.w.Overlays(), asktea.DefaultID) {
			t.Fatalf("overlays %q; want %q open", g.w.Overlays(), asktea.DefaultID)
		}
		want := ask.Layout(question(), 78, ansi.WcWidth, glyph.ASCII())
		if got := d.View(78, 10); got != strings.Join(want, "\n") {
			t.Errorf("View: %q; want the line mode's layout %q", got, want)
		}
		if got := g.w.Render(); !strings.Contains(got, "Run make?") || !strings.Contains(got, "[f] reject and say why") {
			t.Errorf("the workspace does not show the question:\n%s", got)
		}
		if c := d.Cursor(); c == nil || c.X != 2 || c.Y != len(want)-1 {
			t.Errorf("cursor %+v; want after the prompt", c)
		}
		g.press(text("x"), text("y"))
		a := g.answers()
		if len(a) != 1 || a[0].ID != asktea.DefaultID || a[0].Response.Answer.ID != "y" || a[0].Response.Canceled {
			t.Fatalf("answers %+v; want one, y", a)
		}
		if len(g.w.Overlays()) != 0 {
			t.Errorf("overlays %q open after the answer", g.w.Overlays())
		}
		g.press(text("y"))
		if len(g.answers()) != 1 {
			t.Errorf("answered again after closing: %+v", g.answers())
		}
	})
}

// TestDialogReason: a reason is typed in the overlay, its row drawn, and
// enter answers with it.
func TestDialogReason(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg()
		d := asktea.NewDialog(g.w, question(), asktea.WithID("perm"))
		g.run(d.Open())
		g.settle()
		g.press(text("f"), text("n"), text("o"))
		if got := d.View(78, 10); !strings.HasSuffix(got, "> reject and say why: no") {
			t.Errorf("View while typed: %q", got)
		}
		g.send(tea.PasteMsg{Content: " way"})
		g.settle()
		g.press(tea.KeyPressMsg{Code: tea.KeyEnter})
		a := g.answers()
		if len(a) != 1 || a[0].ID != "perm" || a[0].Response.Answer.ID != "why" || a[0].Response.Reason != "no way" {
			t.Fatalf("answers %+v; want why, \"no way\", from perm", a)
		}
	})
}

// TestDialogCancel: esc, which closes any overlay, and ctrl+c each end the
// question canceled.
func TestDialogCancel(t *testing.T) {
	for name, k := range map[string]tea.KeyPressMsg{
		"esc":    {Code: tea.KeyEscape},
		"ctrl+c": {Code: 'c', Mod: tea.ModCtrl},
	} {
		synctest.Test(t, func(t *testing.T) {
			g := newProg()
			d := asktea.NewDialog(g.w, question())
			g.run(d.Open())
			g.settle()
			g.press(k)
			a := g.answers()
			if len(a) != 1 || !a[0].Response.Canceled || a[0].Response.Answer.ID != "cancel" {
				t.Errorf("%s: answers %+v; want one, canceled", name, a)
			}
			if len(g.w.Overlays()) != 0 {
				t.Errorf("%s: overlays %q open", name, g.w.Overlays())
			}
		})
	}
}

// TestDialogWidth: WithWidth sizes the overlay, and the question is laid
// out inside its border.
func TestDialogWidth(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg()
		d := asktea.NewDialog(g.w, question(), asktea.WithWidth(24))
		g.run(d.Open())
		g.settle()
		want := ask.Layout(question(), 22, ansi.WcWidth, glyph.ASCII())
		if got := d.View(22, 10); got != strings.Join(want, "\n") {
			t.Errorf("View(22): %q; want %q", got, want)
		}
		for line := range strings.SplitSeq(d.View(22, 10), "\n") {
			if w := ansi.WcWidth.StringWidth(line); w > 22 {
				t.Errorf("%q is %d cells, in 22", line, w)
			}
		}
	})
}
