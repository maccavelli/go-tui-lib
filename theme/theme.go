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
// A palette colour may also depend on the background or the profile:
// LightDarkColor and ProfileColor are resolved when the theme is built, so a
// theme rebuilt for a new background or profile picks again
// (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §4).
//
// Colour never carries meaning alone: the focused pane's title is bold and
// marked with glyph.Set.Focus as well as coloured.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/internal/enum"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "theme"

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

var backgroundNames = enum.Names[Background]{
	Pkg: pkgName, Type: "Background",
	Tokens: []string{"unknown", "dark", "light"},
}

// String is the background's token: "unknown", "dark" or "light".
func (b Background) String() string { return backgroundNames.String(b) }

// MarshalText is the background's token. A background with no token is an
// error.
func (b Background) MarshalText() ([]byte, error) {
	return backgroundNames.Marshal(b)
}

// UnmarshalText reads a token MarshalText wrote, exactly, case included,
// and also "auto" as Unknown, the word workspace's theme command takes for
// following the terminal.
func (b *Background) UnmarshalText(t []byte) error {
	if string(t) == "auto" {
		*b = Unknown
		return nil
	}
	return backgroundNames.Unmarshal(t, b)
}

// FromDark is the background a terminal reports as dark or light, as
// tea.BackgroundColorMsg.IsDark answers.
func FromDark(isDark bool) Background {
	if isDark {
		return Dark
	}
	return Light
}

// LightDarkColor is a colour that depends on the background. A theme picks
// it through lipgloss.LightDark: Light on a light background and Dark on a
// dark one. On an Unknown background it uses Unknown, or Dark when Unknown
// is nil. A nil pick means the terminal's own foreground.
type LightDarkColor struct{ Light, Dark, Unknown color.Color }

// RGBA implements color.Color with the colour an Unknown background picks,
// so a LightDarkColor fits any Palette field.
func (c LightDarkColor) RGBA() (r, g, b, a uint32) { return rgba(c.pick(Unknown)) }

func (c LightDarkColor) pick(bg Background) color.Color {
	switch bg {
	case Dark, Light:
		return lipgloss.LightDark(bg == Dark)(c.Light, c.Dark)
	}
	if c.Unknown != nil {
		return c.Unknown
	}
	return c.Dark
}

// ProfileColor is a colour that depends on the colour profile. A theme picks
// it through lipgloss.Complete: ANSI, ANSI256 or TrueColor, as the profile
// is, and uses the pick as it is, without converting it again.
type ProfileColor struct{ ANSI, ANSI256, TrueColor color.Color }

// RGBA implements color.Color with the TrueColor colour, or the next one
// given, so a ProfileColor fits any Palette field.
func (c ProfileColor) RGBA() (r, g, b, a uint32) {
	for _, x := range []color.Color{c.TrueColor, c.ANSI256, c.ANSI} {
		if x != nil {
			return x.RGBA()
		}
	}
	return 0, 0, 0, 0
}

// rgba is c's RGBA, or zero for a nil colour.
func rgba(c color.Color) (r, g, b, a uint32) {
	if c == nil {
		return 0, 0, 0, 0
	}
	return c.RGBA()
}

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

var borderNames = enum.Names[BorderStyle]{
	Pkg: pkgName, Type: "BorderStyle",
	Tokens: []string{"light", "rounded", "heavy", "double"},
}

// String is the border style's token: "light", "rounded", "heavy" or
// "double".
func (s BorderStyle) String() string { return borderNames.String(s) }

// MarshalText is the border style's token. A style with no token is an
// error.
func (s BorderStyle) MarshalText() ([]byte, error) {
	return borderNames.Marshal(s)
}

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (s *BorderStyle) UnmarshalText(t []byte) error {
	return borderNames.Unmarshal(t, s)
}

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
	byBG    map[Background]Palette
}

// WithPalette replaces the built-in palette, for every background.
func WithPalette(p Palette) Option {
	return func(o *options) { o.palette = &p }
}

// WithPaletteFor replaces the built-in palette for background bg only. A
// program that passes one for each background keeps its colours when the
// theme is rebuilt for a new background. It wins over WithPalette for bg.
func WithPaletteFor(bg Background, p Palette) Option {
	return func(o *options) {
		if o.byBG == nil {
			o.byBG = map[Background]Palette{}
		}
		o.byBG[bg] = p
	}
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
	if bp, ok := o.byBG[bg]; ok {
		pal = bp
	}
	return Theme{Profile: p, Background: bg, Glyphs: g, Palette: pal, Styles: build(p, bg, pal)}
}

// resolve is the colour c stands for on background bg and profile p: a
// LightDarkColor picked for bg, then a ProfileColor picked for p and kept as
// it is, and any other colour converted to p.
func resolve(c color.Color, p colorprofile.Profile, bg Background) color.Color {
	if ld, ok := c.(LightDarkColor); ok {
		c = ld.pick(bg)
	}
	switch v := c.(type) {
	case nil:
		return nil
	case ProfileColor:
		return lipgloss.Complete(p)(v.ANSI, v.ANSI256, v.TrueColor)
	}
	return p.Convert(c)
}

// build makes the styles for profile p and background bg. NoTTY and below
// get plain styles; ASCII gets attributes and no colour; ANSI and above get
// colours resolved for the background and the profile.
func build(p colorprofile.Profile, bg Background, pal Palette) Styles {
	attrs := p >= colorprofile.ASCII
	colours := p >= colorprofile.ANSI
	style := func(c color.Color, bold bool) lipgloss.Style {
		s := lipgloss.NewStyle()
		if attrs && bold {
			s = s.Bold(true)
		}
		if colours {
			if rc := resolve(c, p, bg); rc != nil {
				s = s.Foreground(rc)
			}
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
