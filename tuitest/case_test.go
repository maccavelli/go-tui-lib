package tuitest

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/glyph"
)

// Case's helpers and Fits
// (docs/decisions/0014-PLAN-component-native-forms.md Step 10).

func TestCaseHelpers(t *testing.T) {
	for _, c := range (Matrix{Widths: []int{80}}).Cases() {
		wantP := colorprofile.ASCII
		if c.Color {
			wantP = colorprofile.TrueColor
		}
		if c.Profile() != wantP {
			t.Errorf("%s: Profile = %v, want %v", c.Name(), c.Profile(), wantP)
		}
		if !reflect.DeepEqual(c.Glyphs(), glyph.For(c.UTF8)) {
			t.Errorf("%s: Glyphs is not glyph.For(%v)", c.Name(), c.UTF8)
		}
	}
	if (Case{UTF8: false}).Glyphs().Ellipsis != glyph.ASCII().Ellipsis {
		t.Error("an ASCII case's glyphs are not ASCII")
	}
}

func TestFits(t *testing.T) {
	var ok recorder
	Fits(&ok, "abc\n\x1b[1mabcd\x1b[m\n", 4, ansi.WcWidth)
	if len(ok.errs) != 0 {
		t.Errorf("a fitting string failed: %v", ok.errs)
	}
	var wide recorder
	Fits(&wide, "abc\nabcde\nab\nabcdef", 4, ansi.WcWidth)
	if len(wide.errs) != 2 || !strings.Contains(wide.errs[0], "line 2 is 5 cells wide, over 4") ||
		!strings.Contains(wide.errs[1], "line 4 is 6 cells wide") {
		t.Errorf("wide lines: %q", wide.errs)
	}
	// The method decides: a family emoji is wider under WcWidth than under
	// GraphemeWidth.
	family := "👨‍👩‍👧"
	w, g := ansi.WcWidth.StringWidth(family), ansi.GraphemeWidth.StringWidth(family)
	if w <= g {
		t.Fatalf("the emoji measures %d by WcWidth and %d by GraphemeWidth; the test needs the first wider", w, g)
	}
	var byGrapheme, byWc recorder
	Fits(&byGrapheme, family, g, ansi.GraphemeWidth)
	Fits(&byWc, family, g, ansi.WcWidth)
	if len(byGrapheme.errs) != 0 || len(byWc.errs) != 1 {
		t.Errorf("at %d cells: GraphemeWidth %q, WcWidth %q", g, byGrapheme.errs, byWc.errs)
	}
}
