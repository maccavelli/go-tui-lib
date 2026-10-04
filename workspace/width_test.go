package workspace

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// The family emoji joins three emoji with zero-width joiners; the heart is
// U+2764 with the VS16 selector. wcwidth and grapheme widths disagree on
// both (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §3).
const family, heart = "\U0001F468\u200D\U0001F469\u200D\U0001F467", "\u2764\uFE0F"

// emojiPanes are panes with the fixture in titles and bodies. The side
// pane's title is long enough to be truncated.
func emojiPanes() map[layout.PaneID]Pane {
	return map[layout.PaneID]Pane{
		"main": &fake{id: "main", title: "Family " + family, body: "user: " + family + " hello\nagent: " + heart + " hi there"},
		"side": &fake{id: "side", title: heart + " hearts, and a title longer than the sidebar", body: heart + heart + heart + " three\n" + family + family + " two"},
		"logs": &fake{id: "logs", title: "Logs", body: "INFO " + heart + " started"},
	}
}

func emojiFrame(c tuitest.Case, m ansi.Method) string {
	w := New(layout.SidebarRightBottom("main", "side", "logs", layout.Gap(0)), emojiPanes(),
		WithTheme(caseTheme(c)), WithWidthMethod(m))
	w.Update(tea.WindowSizeMsg{Width: c.Width, Height: 24})
	return w.Render()
}

func TestWidthMethodGolden(t *testing.T) {
	m := tuitest.Matrix{Widths: []int{80, 120}}
	tuitest.Golden(t, "width-wcwidth", m, func(c tuitest.Case) string { return emojiFrame(c, ansi.WcWidth) })
	tuitest.Golden(t, "width-grapheme", m, func(c tuitest.Case) string { return emojiFrame(c, ansi.GraphemeWidth) })
}

// wholeLines fails unless every line of frame is exactly width cells,
// measured with m.
func wholeLines(t *testing.T, frame string, m ansi.Method, width int) {
	t.Helper()
	for i, line := range strings.Split(frame, "\n") {
		if got := m.StringWidth(line); got != width {
			t.Fatalf("method %v, width %d: line %d is %d cells: %q", m, width, i, got, ansi.Strip(line))
		}
	}
}

func TestEveryLineIsTheFrameWidth(t *testing.T) {
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		for _, width := range []int{80, 120} {
			for _, utf8 := range []bool{true, false} {
				wholeLines(t, emojiFrame(tuitest.Case{Color: true, UTF8: utf8, Width: width}, m), m, width)
			}
		}
	}
}

func TestModeReportSwitchesAsBubbleTeaDoes(t *testing.T) {
	values := map[ansi.ModeSetting]bool{
		ansi.ModeNotRecognized:    false,
		ansi.ModeSet:              true,
		ansi.ModeReset:            true,
		ansi.ModePermanentlySet:   true,
		ansi.ModePermanentlyReset: false,
	}
	for v, switches := range values {
		r := newRig(t, 80, 24)
		if r.w.WidthMethod() != ansi.WcWidth {
			t.Fatalf("the default method is %v, want WcWidth", r.w.WidthMethod())
		}
		msg := tea.ModeReportMsg{Mode: ansi.ModeUnicodeCore, Value: v}
		r.w.Update(msg)
		want := ansi.WcWidth
		if switches {
			want = ansi.GraphemeWidth
		}
		if got := r.w.WidthMethod(); got != want {
			t.Errorf("mode 2027 reported %v: method %v, want %v", v, got, want)
		}
		if !r.main.got(msg) {
			t.Errorf("mode 2027 reported %v: the panes did not get the message", v)
		}
	}
	// Another mode leaves the method alone.
	r := newRig(t, 80, 24)
	r.w.Update(tea.ModeReportMsg{Mode: ansi.ModeSynchronizedOutput, Value: ansi.ModeSet})
	if r.w.WidthMethod() != ansi.WcWidth {
		t.Fatal("a mode 2026 report switched the width method")
	}
}

func TestWithWidthMethodPins(t *testing.T) {
	// A report only ever switches to GraphemeWidth, so pinning WcWidth is
	// the case that shows the pin.
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		r := newRig(t, 80, 24, WithWidthMethod(m))
		r.w.Update(tea.ModeReportMsg{Mode: ansi.ModeUnicodeCore, Value: ansi.ModeSet})
		if got := r.w.WidthMethod(); got != m {
			t.Errorf("pinned to %v, a report changed it to %v", m, got)
		}
	}
}

func TestMethodChangeRedrawsCachedViews(t *testing.T) {
	// Unchanged Changers, so only the method can make their views stale.
	panes := map[layout.PaneID]Pane{}
	for id, p := range emojiPanes() {
		panes[id] = &stable{fake: p.(*fake)}
	}
	w := New(layout.SidebarRightBottom("main", "side", "logs", layout.Gap(0)), panes, WithTheme(caseTheme(tuitest.Case{Color: true, UTF8: true})))
	w.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	wholeLines(t, w.Render(), ansi.WcWidth, 80)
	w.Update(tea.ModeReportMsg{Mode: ansi.ModeUnicodeCore, Value: ansi.ModeSet})
	if !w.dirty {
		t.Fatal("a method change did not mark the frame dirty")
	}
	wholeLines(t, w.Render(), ansi.GraphemeWidth, 80)
}

// TestNoFixedWidthMeasurement scans the package's source: every width is
// measured with the workspace's method.
func TestNoFixedWidthMeasurement(t *testing.T) {
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	banned := map[string]bool{"ansi.StringWidth": true, "ansi.Truncate": true, "lipgloss.Width": true}
	fset := token.NewFileSet()
	scanned := 0
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if x, ok := sel.X.(*ast.Ident); ok && banned[x.Name+"."+sel.Sel.Name] {
						t.Errorf("%s: %s.%s measures with a fixed method", fset.Position(call.Pos()), x.Name, sel.Sel.Name)
					}
				}
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("no source file was scanned")
	}
}
