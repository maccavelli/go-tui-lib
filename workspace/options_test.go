package workspace

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
)

// WithGlyphs, WithProfile, WithBackground, WithSize and GlyphThemeBuilder
// (docs/decisions/0014-PLAN-component-native-forms.md Step 6).

func optionsWorkspace(opts ...Option) *Workspace {
	return New(layout.Pane{ID: "a"}, map[layout.PaneID]Pane{"a": &fake{id: "a", body: "hello"}}, opts...)
}

// facts is a theme's profile, background and whether its glyphs are ASCII.
type facts struct {
	p     colorprofile.Profile
	bg    theme.Background
	ascii bool
}

func factsOf(t theme.Theme) facts {
	return facts{t.Profile, t.Background, reflect.DeepEqual(t.Glyphs, glyph.ASCII())}
}

// built records each call of a builder.
type built struct{ calls []facts }

func (b *built) theme(p colorprofile.Profile, bg theme.Background) theme.Theme {
	b.calls = append(b.calls, facts{p, bg, false})
	return theme.New(p, bg, glyph.Unicode())
}

func (b *built) glyphTheme(p colorprofile.Profile, bg theme.Background, g glyph.Set) theme.Theme {
	ascii := reflect.DeepEqual(g, glyph.ASCII())
	b.calls = append(b.calls, facts{p, bg, ascii})
	return theme.New(p, bg, g)
}

func TestFirstThemeFollowsOptions(t *testing.T) {
	facts3 := []Option{WithGlyphs(glyph.ASCII()), WithProfile(colorprofile.TrueColor), WithBackground(theme.Dark)}
	for _, tc := range []struct {
		name string
		opts []Option
		want facts
	}{
		{"no options", nil, facts{colorprofile.ANSI256, theme.Unknown, false}},
		{"WithGlyphs", []Option{WithGlyphs(glyph.ASCII())}, facts{colorprofile.ANSI256, theme.Unknown, true}},
		{"WithProfile", []Option{WithProfile(colorprofile.ASCII)}, facts{colorprofile.ASCII, theme.Unknown, false}},
		{"WithBackground", []Option{WithBackground(theme.Light)}, facts{colorprofile.ANSI256, theme.Light, false}},
		{"all three", facts3, facts{colorprofile.TrueColor, theme.Dark, true}},
		{"all three, reversed", []Option{facts3[2], facts3[1], facts3[0]}, facts{colorprofile.TrueColor, theme.Dark, true}},
	} {
		w := optionsWorkspace(tc.opts...)
		if got := factsOf(w.theme); got != tc.want || !w.follow {
			t.Errorf("%s: first theme %+v, following %v; want %+v, following", tc.name, got, w.follow, tc.want)
		}
	}

	// A builder makes the first theme, for the starting facts, whatever the
	// order; a GlyphThemeBuilder wins over a ThemeBuilder.
	for _, order := range []string{"builders last", "builders first"} {
		var tb, gb built
		builders := []Option{WithThemeBuilder(tb.theme), WithGlyphThemeBuilder(gb.glyphTheme)}
		opts := append(append([]Option{}, facts3...), builders...)
		if order == "builders first" {
			opts = append(append([]Option{}, builders...), facts3...)
		}
		w := optionsWorkspace(opts...)
		want := facts{colorprofile.TrueColor, theme.Dark, true}
		if len(gb.calls) != 1 || gb.calls[0] != want || len(tb.calls) != 0 || factsOf(w.theme) != want {
			t.Errorf("%s: glyph builder %+v, builder %+v, theme %+v", order, gb.calls, tb.calls, factsOf(w.theme))
		}
		w.Update(tea.ColorProfileMsg{Profile: colorprofile.ANSI})
		if len(gb.calls) != 2 || gb.calls[1] != (facts{colorprofile.ANSI, theme.Dark, true}) || len(tb.calls) != 0 {
			t.Errorf("%s: a rebuild used %+v, %+v", order, gb.calls, tb.calls)
		}
	}

	var tb built
	w := optionsWorkspace(WithBackground(theme.Light), WithThemeBuilder(tb.theme))
	if len(tb.calls) != 1 || tb.calls[0] != (facts{colorprofile.ANSI256, theme.Light, false}) || w.theme.Background != theme.Light {
		t.Errorf("a ThemeBuilder's first theme: %+v", tb.calls)
	}
}

func TestWithThemeWins(t *testing.T) {
	fixed := theme.New(colorprofile.ASCII, theme.Light, glyph.ASCII())
	others := []Option{WithGlyphs(glyph.Unicode()), WithProfile(colorprofile.TrueColor), WithBackground(theme.Dark)}
	for _, opts := range [][]Option{
		append([]Option{WithTheme(fixed)}, others...),
		append(append([]Option{}, others...), WithTheme(fixed)),
	} {
		w := optionsWorkspace(opts...)
		want := facts{colorprofile.ASCII, theme.Light, true}
		if factsOf(w.theme) != want || w.profile != colorprofile.ASCII || w.bg != theme.Light || w.follow {
			t.Errorf("first theme %+v, profile %v, bg %v, following %v; want WithTheme's, fixed",
				factsOf(w.theme), w.profile, w.bg, w.follow)
		}
		w.Update(tea.BackgroundColorMsg{})
		if factsOf(w.theme) != want {
			t.Error("a background message changed a WithTheme theme")
		}
	}

	// D5: WithTheme gives the first theme; a builder after it makes the
	// workspace follow from there, and one before it does not.
	var tb built
	w := optionsWorkspace(WithTheme(fixed), WithThemeBuilder(tb.theme))
	if len(tb.calls) != 0 || factsOf(w.theme) != (facts{colorprofile.ASCII, theme.Light, true}) || !w.follow {
		t.Fatalf("WithTheme then a builder: built %+v, theme %+v, following %v", tb.calls, factsOf(w.theme), w.follow)
	}
	w.Update(tea.ColorProfileMsg{Profile: colorprofile.ANSI})
	if len(tb.calls) != 1 || tb.calls[0] != (facts{colorprofile.ANSI, theme.Light, false}) {
		t.Errorf("WithTheme then a builder, after a profile message: %+v", tb.calls)
	}
	var before built
	w = optionsWorkspace(WithThemeBuilder(before.theme), WithTheme(fixed))
	w.Update(tea.ColorProfileMsg{Profile: colorprofile.ANSI})
	if len(before.calls) != 0 || w.follow {
		t.Errorf("a builder then WithTheme: built %+v, following %v", before.calls, w.follow)
	}
}

func TestWithSizeBeforeFirstMessage(t *testing.T) {
	w := optionsWorkspace(WithTheme(asciiTheme), WithSize(40, 6))
	lines := strings.Split(w.Render(), "\n")
	if len(lines) != 6 || ansi.StringWidth(lines[0]) != 40 {
		t.Fatalf("before any WindowSizeMsg: %d lines, %d wide; want 6 by 40", len(lines), ansi.StringWidth(lines[0]))
	}
	w.Update(tea.WindowSizeMsg{Width: 30, Height: 4})
	if lines := strings.Split(w.Render(), "\n"); len(lines) != 4 || ansi.StringWidth(lines[0]) != 30 {
		t.Errorf("after a WindowSizeMsg: %d lines, %d wide; want 4 by 30", len(lines), ansi.StringWidth(lines[0]))
	}
	if w := optionsWorkspace(WithSize(-3, -1)); w.width != 0 || w.height != 0 {
		t.Errorf("WithSize(-3, -1) = %d by %d; want 0 by 0", w.width, w.height)
	}
	if w := optionsWorkspace(); w.width != fallbackWidth || w.height != fallbackHeight {
		t.Errorf("without WithSize: %d by %d", w.width, w.height)
	}
}

func TestGlyphsReachBuilder(t *testing.T) {
	var gb built
	w := optionsWorkspace(WithGlyphs(glyph.ASCII()), WithGlyphThemeBuilder(gb.glyphTheme))
	if len(gb.calls) != 1 || !gb.calls[0].ascii {
		t.Fatalf("the GlyphThemeBuilder was given %+v; want ASCII", gb.calls)
	}
	// A rebuild passes the theme's glyphs, which are the builder's choice.
	w.Update(tea.ColorProfileMsg{Profile: colorprofile.ANSI})
	if len(gb.calls) != 2 || !gb.calls[1].ascii {
		t.Errorf("a rebuild gave %+v", gb.calls)
	}
	// A ThemeBuilder chooses its own glyphs.
	var tb built
	w = optionsWorkspace(WithGlyphs(glyph.ASCII()), WithThemeBuilder(tb.theme))
	if factsOf(w.theme).ascii {
		t.Error("WithGlyphs reached a ThemeBuilder's theme")
	}
}
