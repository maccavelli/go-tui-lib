package theme

import (
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// luminance is the WCAG relative luminance of c.
func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	lin := func(v uint32) float64 {
		x := float64(v) / 0xffff
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

func contrast(a, b color.Color) float64 {
	la, lb := luminance(a), luminance(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

var (
	black = color.RGBA{A: 0xff}
	white = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// roles visits every role of p that has a colour, saying whether it is
// drawn as text (4.5:1) or as a border (3:1, WCAG's non-text contrast).
func roles(p Palette, visit func(name string, c color.Color, text bool)) {
	v := reflect.ValueOf(p)
	for i := range v.NumField() {
		c, _ := v.Field(i).Interface().(color.Color)
		if c == nil {
			continue
		}
		name := v.Type().Field(i).Name
		visit(name, c, name != "Border")
	}
}

func TestUnknownPaletteClearsBothBackgrounds(t *testing.T) {
	n := 0
	roles(UnknownPalette(), func(name string, c color.Color, _ bool) {
		n++
		for _, cc := range []struct {
			label string
			c     color.Color
		}{{"as written", c}, {"after ANSI256 conversion", colorprofile.ANSI256.Convert(c)}} {
			if k := contrast(cc.c, black); k < 4.5 {
				t.Errorf("Unknown.%s %s: %.2f:1 on black, want 4.5", name, cc.label, k)
			}
			if k := contrast(cc.c, white); k < 4.5 {
				t.Errorf("Unknown.%s %s: %.2f:1 on white, want 4.5", name, cc.label, k)
			}
		}
	})
	if n != 8 {
		t.Fatalf("checked %d coloured roles, want 8", n)
	}
}

func TestPalettesClearTheirBackground(t *testing.T) {
	for _, c := range []struct {
		name string
		p    Palette
		bg   color.Color
	}{{"Dark", DarkPalette(), black}, {"Light", LightPalette(), white}} {
		roles(c.p, func(role string, col color.Color, text bool) {
			floor := 3.0
			if text {
				floor = 4.5
			}
			if k := contrast(col, c.bg); k < floor {
				t.Errorf("%s.%s: %.2f:1, want %.1f", c.name, role, k, floor)
			}
		})
	}
}

// styles visits every built style.
func styles(s Styles, visit func(name string, st lipgloss.Style)) {
	v := reflect.ValueOf(s)
	for i := range v.NumField() {
		visit(v.Type().Field(i).Name, v.Field(i).Interface().(lipgloss.Style))
	}
}

func TestNoTTYIsPlain(t *testing.T) {
	styles(New(colorprofile.NoTTY, Dark, glyph.Unicode()).Styles, func(name string, st lipgloss.Style) {
		if got := st.Render("x"); got != "x" {
			t.Errorf("NoTTY %s renders %q, want plain \"x\"", name, got)
		}
	})
}

func TestASCIIKeepsBoldDropsColour(t *testing.T) {
	th := New(colorprofile.ASCII, Dark, glyph.ASCII())
	styles(th.Styles, func(name string, st lipgloss.Style) {
		if got := st.Render("x"); strings.Contains(got, "38;") || strings.Contains(got, "\x1b[3") || strings.Contains(got, "\x1b[9") {
			t.Errorf("ASCII %s renders %q, which carries colour", name, got)
		}
	})
	if got := th.Styles.FocusTitle.Render("x"); !strings.Contains(got, "\x1b[1m") {
		t.Errorf("ASCII FocusTitle renders %q, want bold", got)
	}
}

func TestColoursFollowTheProfile(t *testing.T) {
	for _, c := range []struct {
		p    colorprofile.Profile
		want string
	}{{colorprofile.TrueColor, "38;2;"}, {colorprofile.ANSI256, "38;5;"}, {colorprofile.ANSI, "\x1b[9"}} {
		got := New(c.p, Dark, glyph.Unicode()).Styles.Accent.Render("x")
		if !strings.Contains(got, c.want) {
			t.Errorf("profile %v: Accent renders %q, want %q", c.p, got, c.want)
		}
	}
}

func TestBorderComesFromTheGlyphs(t *testing.T) {
	u := New(colorprofile.TrueColor, Dark, glyph.Unicode())
	if b := u.Border(BorderRounded); b.TopLeft != "╭" || b.Top != "─" {
		t.Errorf("Unicode rounded border = %+v", b)
	}
	a := New(colorprofile.ASCII, Dark, glyph.ASCII())
	for _, s := range []BorderStyle{BorderLight, BorderRounded, BorderHeavy, BorderDouble} {
		if b := a.Border(s); b.TopLeft != "+" || b.Left != "|" {
			t.Errorf("ASCII border %d = %+v", s, b)
		}
	}
}

func TestWithPalette(t *testing.T) {
	p := DarkPalette()
	p.Accent = lipgloss.Color("#00ff00")
	th := New(colorprofile.TrueColor, Dark, glyph.Unicode(), WithPalette(p))
	if got := th.Styles.Accent.Render("x"); !strings.Contains(got, "38;2;0;255;0") {
		t.Errorf("WithPalette: Accent renders %q", got)
	}
}

// swatch renders every role, the focus marker and a border, as a package
// using the theme would.
func swatch(c tuitest.Case) string {
	th := New(c.Profile(), Unknown, c.Glyphs())
	s := th.Styles
	var b strings.Builder
	b.WriteString(th.Glyphs.Focus + " " + s.FocusTitle.Render("Focused title") + "\n")
	for _, r := range []struct {
		name string
		st   lipgloss.Style
	}{
		{"Title", s.Title}, {"Body", s.Body}, {"Muted", s.Muted}, {"Accent", s.Accent},
		{"Success", s.Success}, {"Warning", s.Warning}, {"Error", s.Error},
		{"Badge", s.Badge},
	} {
		b.WriteString(th.Glyphs.Bullet + " " + r.st.Render(r.name) + "\n")
	}
	box := lipgloss.NewStyle().Border(th.Border(BorderRounded)).BorderForeground(nil).
		Width(c.Width).Render("A pane " + th.Glyphs.Ellipsis)
	b.WriteString(s.Border.Render(box) + "\n")
	return b.String()
}

func TestSwatchGolden(t *testing.T) {
	tuitest.Golden(t, "swatch", tuitest.Matrix{Widths: []int{30, 50}}, swatch)
}
