package theme

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
)

// same reports whether a and b are the same colour, nil included.
func same(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// accent is the Accent style's foreground for a theme whose palette has c
// as its Accent.
func accent(p colorprofile.Profile, bg Background, c color.Color) color.Color {
	return New(p, bg, glyph.Unicode(), WithPalette(Palette{Accent: c})).Styles.Accent.GetForeground()
}

func TestFromDark(t *testing.T) {
	if FromDark(true) != Dark || FromDark(false) != Light {
		t.Fatal("FromDark does not map dark to Dark and light to Light")
	}
}

func TestLightDarkColorPicksByBackground(t *testing.T) {
	light, dark, unknown := lipgloss.Color("#d70000"), lipgloss.Color("#87d787"), lipgloss.Color("#5f5fff")
	c := LightDarkColor{Light: light, Dark: dark, Unknown: unknown}
	for bg, want := range map[Background]color.Color{Light: light, Dark: dark, Unknown: unknown} {
		if got := accent(colorprofile.TrueColor, bg, c); !same(got, want) {
			t.Errorf("background %d: %v, want %v", bg, got, want)
		}
	}
	noUnknown := LightDarkColor{Light: light, Dark: dark}
	if got := accent(colorprofile.TrueColor, Unknown, noUnknown); !same(got, dark) {
		t.Errorf("Unknown background with no Unknown colour: %v, want the dark one %v", got, dark)
	}
	if !same(noUnknown, dark) || !same(c, unknown) {
		t.Error("RGBA is not the colour an Unknown background picks")
	}
}

func TestProfileColorPicksByProfileAndIsNotConvertedAgain(t *testing.T) {
	// Each field is a TrueColor value, so a conversion to ANSI or ANSI256
	// after the pick would change it.
	ansi, ansi256, tc := lipgloss.Color("#123456"), lipgloss.Color("#654321"), lipgloss.Color("#abcdef")
	c := ProfileColor{ANSI: ansi, ANSI256: ansi256, TrueColor: tc}
	for p, want := range map[colorprofile.Profile]color.Color{
		colorprofile.ANSI: ansi, colorprofile.ANSI256: ansi256, colorprofile.TrueColor: tc,
	} {
		if got := accent(p, Dark, c); !same(got, want) {
			t.Errorf("profile %v: %v, want %v, unconverted", p, got, want)
		}
	}
	// Without colour the pick is not used at all.
	if got := accent(colorprofile.ASCII, Dark, c); got != nil && !same(got, lipgloss.NoColor{}) {
		t.Errorf("the ASCII profile coloured a ProfileColor: %v", got)
	}
	if !same(c, tc) || !same(ProfileColor{ANSI: ansi}, ansi) || !same(ProfileColor{}, color.RGBA{}) {
		t.Error("RGBA is not the TrueColor colour, or the next one given")
	}
}

func TestAdaptiveColoursNest(t *testing.T) {
	// A LightDarkColor may hold ProfileColors: the background picks first,
	// then the profile.
	darkTC := lipgloss.Color("#00ff00")
	c := LightDarkColor{Light: lipgloss.Color("#ff0000"), Dark: ProfileColor{ANSI: lipgloss.Color("#0000ff"), TrueColor: darkTC}}
	if got := accent(colorprofile.TrueColor, Dark, c); !same(got, darkTC) {
		t.Fatalf("a ProfileColor inside a LightDarkColor: %v, want %v", got, darkTC)
	}
}

func TestWithPaletteForKeepsAPalettePerBackground(t *testing.T) {
	d := Palette{Accent: lipgloss.Color("#111111")}
	l := Palette{Accent: lipgloss.Color("#eeeeee")}
	every := Palette{Accent: lipgloss.Color("#777777")}
	opts := []Option{WithPalette(every), WithPaletteFor(Dark, d), WithPaletteFor(Light, l)}
	for bg, want := range map[Background]color.Color{Dark: d.Accent, Light: l.Accent, Unknown: every.Accent} {
		th := New(colorprofile.TrueColor, bg, glyph.Unicode(), opts...)
		if !same(th.Palette.Accent, want) {
			t.Errorf("background %d: accent %v, want %v", bg, th.Palette.Accent, want)
		}
	}
}
