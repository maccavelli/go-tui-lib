// Package theme turns a colour profile, a terminal background and a glyph set
// into the styles every go-tui-lib package renders with.
//
// It has three layers: a Palette of raw colours for each background, the
// semantic roles that palette fills, and the Styles built from them once. The
// output degrades instead of breaking
// (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rule 4):
//
//   - with the NoTTY profile, styles emit no escape sequence at all;
//   - with the ASCII profile (what NO_COLOR asks for on a terminal), styles
//     keep bold and drop every colour;
//   - with ANSI and ANSI256, colours are converted to the profile when the
//     theme is built;
//   - with an unknown background, the Unknown palette is used, whose
//     coloured roles clear 4.5:1 against both black and white.
//
// Colour never carries meaning alone: the focused pane's title is bold and
// marked with glyph.Set.Focus as well as coloured.
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
)

// Background is what the terminal's background is known to be.
type Background int

const (
	// Unknown is a background that has not been, or cannot be, detected.
	Unknown Background = iota
	// Dark is a dark background.
	Dark
	// Light is a light background.
	Light
)

// Palette holds a raw colour for each semantic role. A nil colour means the
// terminal's own foreground.
type Palette struct {
	Title, Body, Muted      color.Color
	Border, BorderFocus     color.Color
	Accent                  color.Color
	Success, Warning, Error color.Color
	Badge                   color.Color
}

// DarkPalette is the palette for a dark background.
func DarkPalette() Palette {
	return Palette{
		Muted:       lipgloss.Color("#8a8a8a"),
		Border:      lipgloss.Color("#626262"),
		BorderFocus: lipgloss.Color("#87afff"),
		Accent:      lipgloss.Color("#87afff"),
		Success:     lipgloss.Color("#87d787"),
		Warning:     lipgloss.Color("#ffd75f"),
		Error:       lipgloss.Color("#ff5f5f"),
		Badge:       lipgloss.Color("#d7afff"),
	}
}

// LightPalette is the palette for a light background.
func LightPalette() Palette {
	return Palette{
		Muted:       lipgloss.Color("#6c6c6c"),
		Border:      lipgloss.Color("#8a8a8a"),
		BorderFocus: lipgloss.Color("#005fd7"),
		Accent:      lipgloss.Color("#005fd7"),
		Success:     lipgloss.Color("#00875f"),
		Warning:     lipgloss.Color("#af5f00"),
		Error:       lipgloss.Color("#d70000"),
		Badge:       lipgloss.Color("#8700af"),
	}
}

// UnknownPalette is the palette for a background that is not known. Every
// colour is an xterm-256 entry, so ANSI256 conversion keeps it exactly, and
// clears 4.5:1 against both black and white. Only six entries do. No yellow
// or orange is among them, so Warning is magenta here, and relies, as every
// role does, on its text or glyph as well as its colour.
func UnknownPalette() Palette {
	return Palette{
		Muted:       lipgloss.Color("#767676"),
		Border:      lipgloss.Color("#767676"),
		BorderFocus: lipgloss.Color("#5f5fff"),
		Accent:      lipgloss.Color("#5f5fff"),
		Success:     lipgloss.Color("#00875f"),
		Warning:     lipgloss.Color("#d700af"),
		Error:       lipgloss.Color("#af5f5f"),
		Badge:       lipgloss.Color("#875fd7"),
	}
}

// PaletteFor returns the built-in palette for bg.
func PaletteFor(bg Background) Palette {
	switch bg {
	case Dark:
		return DarkPalette()
	case Light:
		return LightPalette()
	default:
		return UnknownPalette()
	}
}

// Styles are the built styles, one for each role, plus FocusTitle for the
// focused pane's title.
type Styles struct {
	Title, Body, Muted      lipgloss.Style
	Border, BorderFocus     lipgloss.Style
	Accent                  lipgloss.Style
	Success, Warning, Error lipgloss.Style
	Badge                   lipgloss.Style
	FocusTitle              lipgloss.Style
}

// BorderStyle names one of the glyph set's border styles.
type BorderStyle int

const (
	// BorderLight is a single thin line.
	BorderLight BorderStyle = iota
	// BorderRounded is a thin line with rounded corners.
	BorderRounded
	// BorderHeavy is a single thick line.
	BorderHeavy
	// BorderDouble is a double line.
	BorderDouble
)

// Theme is everything a package needs to render: the profile and background
// it was built for, the glyphs, the palette and the built styles.
type Theme struct {
	Profile    colorprofile.Profile
	Background Background
	Glyphs     glyph.Set
	Palette    Palette
	Styles     Styles
}

// Option changes how New builds a theme.
type Option func(*options)

type options struct {
	palette *Palette
}

// WithPalette replaces the built-in palette for the theme's background.
func WithPalette(p Palette) Option {
	return func(o *options) { o.palette = &p }
}

// New builds a theme for profile p, background bg and glyph set g.
func New(p colorprofile.Profile, bg Background, g glyph.Set, opts ...Option) Theme {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	pal := PaletteFor(bg)
	if o.palette != nil {
		pal = *o.palette
	}
	return Theme{Profile: p, Background: bg, Glyphs: g, Palette: pal, Styles: build(p, pal)}
}

// build makes the styles for profile p. NoTTY and below get plain styles;
// ASCII gets attributes and no colour; ANSI and above get colours converted
// to the profile.
func build(p colorprofile.Profile, pal Palette) Styles {
	attrs := p >= colorprofile.ASCII
	colours := p >= colorprofile.ANSI
	style := func(c color.Color, bold bool) lipgloss.Style {
		s := lipgloss.NewStyle()
		if attrs && bold {
			s = s.Bold(true)
		}
		if colours && c != nil {
			s = s.Foreground(p.Convert(c))
		}
		return s
	}
	return Styles{
		Title:       style(pal.Title, true),
		Body:        style(pal.Body, false),
		Muted:       style(pal.Muted, false),
		Border:      style(pal.Border, false),
		BorderFocus: style(pal.BorderFocus, true),
		Accent:      style(pal.Accent, false),
		Success:     style(pal.Success, false),
		Warning:     style(pal.Warning, false),
		Error:       style(pal.Error, false),
		Badge:       style(pal.Badge, true),
		FocusTitle:  style(pal.Accent, true),
	}
}

// Border returns the theme's glyphs for style s as a lipgloss.Border.
func (t Theme) Border(s BorderStyle) lipgloss.Border {
	var b glyph.Border
	switch s {
	case BorderRounded:
		b = t.Glyphs.Rounded
	case BorderHeavy:
		b = t.Glyphs.Heavy
	case BorderDouble:
		b = t.Glyphs.Double
	default:
		b = t.Glyphs.Light
	}
	return lipgloss.Border{
		Top: b.Top, Bottom: b.Bottom, Left: b.Left, Right: b.Right,
		TopLeft: b.TopLeft, TopRight: b.TopRight, BottomLeft: b.BottomLeft, BottomRight: b.BottomRight,
		MiddleLeft: b.MiddleLeft, MiddleRight: b.MiddleRight, Middle: b.Middle,
		MiddleTop: b.MiddleTop, MiddleBottom: b.MiddleBottom,
	}
}
