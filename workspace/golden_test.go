package workspace

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// caseTheme is the theme a golden case renders with.
func caseTheme(c tuitest.Case) theme.Theme {
	p := colorprofile.ASCII
	if c.Color {
		p = colorprofile.TrueColor
	}
	return theme.New(p, theme.Unknown, glyph.For(c.UTF8))
}

// frame builds the four-pane workspace for a case, applies setup, and
// renders it at the case width and 24 rows.
func frame(c tuitest.Case, chrome Chrome, setup func(*Workspace)) string {
	panes := map[layout.PaneID]Pane{
		"main":   &fake{id: "main", title: "Session", body: "user: hello\nagent: hi there"},
		"side":   &fake{id: "side", title: "Metrics", badge: "3", body: "tokens 1200\ncost $0.01"},
		"logs":   &fake{id: "logs", title: "Logs", body: "INFO started\nWARN slow"},
		"footer": &fake{id: "footer", noFocus: true, body: "model x | mode ask | ctx 12%"},
	}
	gap := 0
	if chrome == Separators {
		gap = 1
	}
	root := layout.SidebarRightBottom("main", "side", "logs", layout.Footer("footer", 1), layout.Gap(gap))
	w := New(root, panes, WithTheme(caseTheme(c)), WithChrome(chrome), WithPaneChrome("footer", None))
	w.Update(tea.WindowSizeMsg{Width: c.Width, Height: 24})
	if setup != nil {
		setup(w)
	}
	return w.Render()
}

func TestFramesGolden(t *testing.T) {
	m := tuitest.Matrix{Widths: []int{80, 160}}
	for name, setup := range map[string]func(*Workspace){
		"focus-main": nil,
		"focus-side": func(w *Workspace) { w.Focus("side") },
		"focus-logs": func(w *Workspace) { w.Focus("logs") },
		"overlay": func(w *Workspace) {
			w.Push(Overlay{ID: "permission", Pane: &fake{id: "permission", title: "Allow bash?", body: "rm -rf build/\n[y] yes  [n] no"}, Width: 36, Height: 6, Modal: true})
		},
		"zoom": func(w *Workspace) { w.Zoom("logs") },
	} {
		tuitest.Golden(t, "frame-"+name, m, func(c tuitest.Case) string { return frame(c, Borders, setup) })
	}
	tuitest.Golden(t, "frame-separators", m, func(c tuitest.Case) string { return frame(c, Separators, nil) })
}

func TestFramesFillTheScreen(t *testing.T) {
	for _, chrome := range []Chrome{Borders, Separators, None} {
		out := frame(tuitest.Case{Color: false, UTF8: false, Width: 100}, chrome, nil)
		if rows := strings.Count(out, "\n") + 1; rows != 24 {
			t.Errorf("chrome %d: %d rows, want 24", chrome, rows)
		}
	}
}
