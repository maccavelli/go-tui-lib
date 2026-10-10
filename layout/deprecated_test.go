package layout

import (
	"reflect"
	"testing"
)

// TestDeprecatedOptionsStillWork: each old name does what its new name does,
// through v0.9.x (docs/decisions/0014-PLAN-canonicalization.md Step 4). It
// sits in the package, so that staticcheck does not report the deprecated
// names it uses on purpose.
func TestDeprecatedOptionsStillWork(t *testing.T) {
	for _, c := range []struct {
		name     string
		old, new PresetOption
		width    int // where the option has an effect
	}{
		{"SidebarWidth", SidebarWidth(Fixed(40)), WithSidebarWidth(Fixed(40)), 120},
		{"BottomHeight", BottomHeight(Fixed(8)), WithBottomHeight(Fixed(8)), 120},
		{"MainSize", MainSize(Fixed(50).AtLeast(10)), WithMainSize(Fixed(50).AtLeast(10)), 120},
		{"BottomSpan", BottomSpan(UnderMain), WithBottomSpan(UnderMain), 120},
		{"Footer", Footer("footer", 2), WithFooter("footer", 2), 120},
		{"Gap", Gap(0), WithGap(0), 120},
		{"Breakpoints", Breakpoints(200, 150, 50), WithBreakpoints(200, 150, 50), 120},
		{"NoResponsive", NoResponsive(), WithoutResponsive(), 80}, // the default folds below 100
	} {
		area := Rect{W: c.width, H: 40}
		def, err := Solve(SidebarRightBottom("main", "side", "logs"), area, State{})
		if err != nil {
			t.Fatal(err)
		}
		was, err := Solve(SidebarRightBottom("main", "side", "logs", c.old), area, State{})
		if err != nil {
			t.Fatal(err)
		}
		is, err := Solve(SidebarRightBottom("main", "side", "logs", c.new), area, State{})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(was, is) {
			t.Errorf("%s: the old name solves to %v, the new to %v", c.name, was.Panes, is.Panes)
		}
		if reflect.DeepEqual(is, def) {
			t.Errorf("%s: the option changed nothing at %dx40, so the case proves nothing", c.name, c.width)
		}
	}
}
