package tuitest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// recorder is a T that keeps failures instead of failing the test.
type recorder struct{ errs, fatals []string }

func (r *recorder) Helper()                   {}
func (r *recorder) Errorf(f string, a ...any) { r.errs = append(r.errs, fmt.Sprintf(f, a...)) }
func (r *recorder) Fatalf(f string, a ...any) { r.fatals = append(r.fatals, fmt.Sprintf(f, a...)) }
func (r *recorder) failed() bool              { return len(r.errs)+len(r.fatals) > 0 }
func (r *recorder) all() string               { return strings.Join(append(r.errs, r.fatals...), "\n") }
func render(c Case) string                    { return fmt.Sprintf("%s\nwidth %d\n", c.Name(), c.Width) }
func renderChanging(at string) func(Case) string {
	return func(c Case) string {
		s := render(c)
		if c.Name() == at {
			s = strings.Replace(s, "width", "Width", 1)
		}
		return s
	}
}

func TestCases(t *testing.T) {
	var names []string
	for _, c := range (Matrix{Widths: []int{60, 80}}).Cases() {
		names = append(names, c.Name())
	}
	want := []string{
		"color.utf8.60", "color.utf8.80", "color.ascii.60", "color.ascii.80",
		"nocolor.utf8.60", "nocolor.utf8.80", "nocolor.ascii.60", "nocolor.ascii.80",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("cases = %v, want %v", names, want)
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := Matrix{Widths: []int{60, 80}}
	var r recorder
	compare(&r, dir, "view", m, render, true, false)
	if r.failed() {
		t.Fatal(r.all())
	}
	if files, _ := os.ReadDir(dir); len(files) != 8 {
		t.Fatalf("update wrote %d files, want 8", len(files))
	}
	r = recorder{}
	compare(&r, dir, "view", m, render, false, false)
	if r.failed() {
		t.Fatalf("an unchanged rendering failed: %s", r.all())
	}
}

func TestGoldenReportsOneChangedCell(t *testing.T) {
	dir := t.TempDir()
	m := Matrix{Widths: []int{60, 80}}
	compare(&recorder{}, dir, "view", m, render, true, false)
	var r recorder
	compare(&r, dir, "view", m, renderChanging("nocolor.ascii.80"), false, false)
	if len(r.errs) != 1 || !strings.Contains(r.errs[0], "view nocolor.ascii.80: line 2 differs") {
		t.Fatalf("errors = %q, want one naming the case and line 2", r.errs)
	}
}

func TestGoldenComparesTheLastLine(t *testing.T) {
	dir := t.TempDir()
	m := Matrix{Widths: []int{60}}
	compare(&recorder{}, dir, "view", m, render, true, false)
	var r recorder
	compare(&r, dir, "view", m, func(c Case) string { return render(c) + "tail\n" }, false, false)
	if len(r.errs) != 4 || !strings.Contains(r.errs[0], "line 3 differs") {
		t.Fatalf("an added last line: errors = %q, want 4 naming line 3", r.errs)
	}
}

func TestAnnotateCountsCells(t *testing.T) {
	// "界" is three bytes and two cells; the escape sequence is zero cells.
	got := Annotate("a界\n\x1b[1mbold\x1b[0m\n")
	want := "  3|a界\n  4|\x1b[1mbold\x1b[0m\n"
	if got != want {
		t.Fatalf("Annotate = %q, want %q", got, want)
	}
}

func TestGoldenMissingFile(t *testing.T) {
	var r recorder
	compare(&r, t.TempDir(), "view", Matrix{Widths: []int{60}}, render, false, false)
	if len(r.errs) != 4 || !strings.Contains(r.errs[0], "run the test with -update") {
		t.Fatalf("errors = %q, want 4 asking for -update", r.errs)
	}
}

func TestGoldenUpdateTouchesOnlyItsCases(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "view.color.utf8.200.golden")
	if err := os.WriteFile(other, []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compare(&recorder{}, dir, "view", Matrix{Widths: []int{60}}, render, true, false)
	if b, _ := os.ReadFile(other); string(b) != "kept\n" {
		t.Fatalf("a case outside the matrix was rewritten: %q", b)
	}
}

func TestGoldenNeedsWidths(t *testing.T) {
	var r recorder
	compare(&r, t.TempDir(), "view", Matrix{}, render, false, false)
	if len(r.fatals) != 1 {
		t.Fatalf("an empty matrix: fatals = %q", r.fatals)
	}
}

func TestText(t *testing.T) {
	dir := t.TempDir()
	var r recorder
	textAt(&r, dir, "diagram", "ab\ncd\n", true)
	if r.failed() {
		t.Fatal(r.all())
	}
	if b, err := os.ReadFile(filepath.Join(dir, "diagram.golden")); err != nil || string(b) != "  2|ab\n  2|cd\n" {
		t.Fatalf("Text wrote %q, %v", b, err)
	}
	textAt(&r, dir, "diagram", "ab\ncd\n", false)
	if r.failed() {
		t.Fatalf("an unchanged text failed: %s", r.all())
	}
	textAt(&r, dir, "diagram", "ab\ncX\n", false)
	if len(r.errs) != 1 || !strings.Contains(r.errs[0], "line 2 differs") {
		t.Fatalf("a changed text: errors = %q", r.errs)
	}
}
