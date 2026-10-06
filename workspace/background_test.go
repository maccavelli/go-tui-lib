package workspace_test

import (
	"fmt"
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// built records the backgrounds a following workspace builds its theme for.
type built struct{ bgs []theme.Background }

func (b *built) builder(p colorprofile.Profile, bg theme.Background) theme.Theme {
	b.bgs = append(b.bgs, bg)
	return theme.New(p, bg, glyph.ASCII())
}

func (b *built) last() theme.Background { return b.bgs[len(b.bgs)-1] }

func following(b *built, opts ...workspace.Option) *workspace.Workspace {
	ws := workspace.New(layout.Pane{ID: "a"}, map[layout.PaneID]workspace.Pane{"a": &transcript{}},
		append([]workspace.Option{workspace.WithThemeBuilder(b.builder)}, opts...)...)
	ws.Update(tea.WindowSizeMsg{Width: 30, Height: 5})
	return ws
}

func TestSetBackgroundPins(t *testing.T) {
	var b built
	ws := following(&b)
	if cmd := ws.SetBackground(theme.Dark); cmd != nil {
		t.Error("pinning dark returned a command")
	}
	if b.last() != theme.Dark {
		t.Fatalf("pinned dark built %v", b.last())
	}
	n := len(b.bgs)
	ws.Update(tea.BackgroundColorMsg{Color: color.White})
	if len(b.bgs) != n || b.last() != theme.Dark {
		t.Errorf("a light report rebuilt a pinned dark theme: %v", b.bgs[n:])
	}
	if cmd := ws.SetBackground(theme.Light); cmd != nil || b.last() != theme.Light {
		t.Errorf("pinning light: %v", b.bgs)
	}
}

func TestSetBackgroundAuto(t *testing.T) {
	var b built
	ws := following(&b)
	ws.Update(tea.BackgroundColorMsg{Color: color.Black}) // the terminal is dark
	ws.SetBackground(theme.Light)
	cmd := ws.SetBackground(theme.Unknown)
	if b.last() != theme.Dark {
		t.Errorf("auto rebuilt for %v, want the reported dark", b.last())
	}
	if cmd == nil || fmt.Sprintf("%T", cmd()) != fmt.Sprintf("%T", tea.RequestBackgroundColor()) {
		t.Error("auto does not ask the terminal again")
	}
	ws.Update(tea.BackgroundColorMsg{Color: color.White})
	if b.last() != theme.Light {
		t.Error("after auto, a light report did not rebuild the theme")
	}

	var quiet built
	q := following(&quiet, workspace.WithoutBackgroundQuery())
	q.SetBackground(theme.Dark)
	if cmd := q.SetBackground(theme.Unknown); cmd != nil {
		t.Error("auto asked the terminal under WithoutBackgroundQuery")
	}
}

func TestSetBackgroundFixedTheme(t *testing.T) {
	ws := workspace.New(layout.Pane{ID: "a"}, map[layout.PaneID]workspace.Pane{"a": &transcript{}},
		workspace.WithTheme(theme.New(colorprofile.TrueColor, theme.Dark, glyph.ASCII())))
	ws.Update(tea.WindowSizeMsg{Width: 30, Height: 5})
	before := ws.Render()
	for _, bg := range []theme.Background{theme.Light, theme.Unknown, theme.Dark} {
		if cmd := ws.SetBackground(bg); cmd != nil {
			t.Errorf("%v on a WithTheme workspace returned a command", bg)
		}
		if ws.Render() != before {
			t.Errorf("%v changed a WithTheme workspace's frame", bg)
		}
	}
}
