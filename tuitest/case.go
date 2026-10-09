package tuitest

import (
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/glyph"
)

// Profile is the colour profile a case renders with: TrueColor with colour,
// and ASCII, which carries no colour, without
// (docs/decisions/0014-MADR-native-integration-api.md W2).
func (c Case) Profile() colorprofile.Profile {
	if c.Color {
		return colorprofile.TrueColor
	}
	return colorprofile.ASCII
}

// Glyphs is the glyph set a case renders with: glyph.For(c.UTF8).
func (c Case) Glyphs() glyph.Set { return glyph.For(c.UTF8) }

// Fits fails t for each line of s wider than width cells, measured with m.
// It is opt-in, because the width method decides: a view drawn with
// ansi.WcWidth may be wider under ansi.GraphemeWidth, and the other way.
func Fits(t T, s string, width int, m ansi.Method) {
	t.Helper()
	for i, line := range strings.Split(s, "\n") {
		if w := m.StringWidth(line); w > width {
			t.Errorf("tuitest: line %d is %d cells wide, over %d: %q", i+1, w, width, line)
		}
	}
}
