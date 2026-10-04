package workspace

import (
	"image/color"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
)

// followingWorkspace has unchanged Changer panes and no fixed theme, so it
// follows the terminal, and only a theme change can make its views stale.
func followingWorkspace(opts ...Option) (*Workspace, *stable) {
	main := &stable{fake: &fake{id: "main", body: "hello"}}
	w := New(layout.SidebarRight("main", "side"), map[layout.PaneID]Pane{"main": main, "side": &stable{fake: &fake{id: "side"}}}, opts...)
	w.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	return w, main
}

func TestFollowingThemeRestyles(t *testing.T) {
	w, main := followingWorkspace()
	first := w.Render()
	n := main.views
	w.Update(tea.ColorProfileMsg{Profile: colorprofile.TrueColor})
	if w.theme.Profile != colorprofile.TrueColor || !w.dirty {
		t.Fatalf("after ColorProfileMsg: profile %v, dirty %v", w.theme.Profile, w.dirty)
	}
	second := w.Render()
	if second == first {
		t.Fatal("a TrueColor profile did not restyle the frame")
	}
	if main.views == n {
		t.Fatal("a view cached before the theme change was not drawn again")
	}
	n = main.views
	w.Update(tea.BackgroundColorMsg{Color: color.White})
	if w.theme.Background != theme.Light {
		t.Fatalf("after a light background: %v", w.theme.Background)
	}
	if third := w.Render(); third == second || main.views == n {
		t.Fatal("a light background did not restyle the frame and its cached views")
	}
}

func TestFixedThemeIgnoresBothMessages(t *testing.T) {
	r := newRig(t, 80, 24) // WithTheme(asciiTheme)
	gen := r.w.themeGen
	cp := tea.ColorProfileMsg{Profile: colorprofile.TrueColor}
	bg := tea.BackgroundColorMsg{Color: color.White}
	r.w.Update(cp)
	r.w.Update(bg)
	if r.w.theme.Profile != asciiTheme.Profile || r.w.theme.Background != asciiTheme.Background || r.w.themeGen != gen {
		t.Fatalf("a fixed theme changed: profile %v, background %v, generation %d → %d", r.w.theme.Profile, r.w.theme.Background, gen, r.w.themeGen)
	}
	if !r.main.got(cp) || !r.main.got(bg) {
		t.Fatal("the panes did not get the profile and background messages")
	}
}

func TestInitAsksForTheBackground(t *testing.T) {
	ask := tea.RequestBackgroundColor()
	w, _ := followingWorkspace()
	if !slices.Contains(run(w.Init()), ask) {
		t.Fatal("Init does not ask for the background by default")
	}
	w, _ = followingWorkspace(WithoutBackgroundQuery())
	if slices.Contains(run(w.Init()), ask) {
		t.Fatal("Init asks for the background with WithoutBackgroundQuery")
	}
}

func TestThemeBuilderKeepsPalettesPerBackground(t *testing.T) {
	dark := theme.Palette{Accent: lipgloss.Color("#111111")}
	light := theme.Palette{Accent: lipgloss.Color("#eeeeee")}
	w, _ := followingWorkspace(WithThemeBuilder(func(p colorprofile.Profile, bg theme.Background) theme.Theme {
		return theme.New(p, bg, glyph.Unicode(), theme.WithPaletteFor(theme.Dark, dark), theme.WithPaletteFor(theme.Light, light))
	}))
	w.Update(tea.BackgroundColorMsg{Color: color.Black})
	if w.theme.Palette.Accent != dark.Accent {
		t.Fatalf("on a dark background: accent %v, want %v", w.theme.Palette.Accent, dark.Accent)
	}
	w.Update(tea.BackgroundColorMsg{Color: color.White})
	if w.theme.Palette.Accent != light.Accent {
		t.Fatalf("after a change to light: accent %v, want %v", w.theme.Palette.Accent, light.Accent)
	}
}

func TestSetThemeRedraws(t *testing.T) {
	w, main := followingWorkspace()
	w.Render()
	n := main.views
	w.SetTheme(theme.New(colorprofile.TrueColor, theme.Dark, glyph.ASCII()))
	if !w.dirty {
		t.Fatal("SetTheme did not mark the frame dirty")
	}
	w.Render()
	if main.views == n {
		t.Fatal("SetTheme did not redraw a cached view")
	}
}
