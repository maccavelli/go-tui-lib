// Package tuitest renders a view across the test matrix every go-tui-lib
// package uses, {colour, no colour} × {UTF-8, ASCII} × widths, and compares
// each rendering with a golden file
// (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rule 6).
//
// Golden files live under the calling package's testdata/golden/. Each line is
// stored with its width in terminal cells in front of it, so a change of width
// shows in the diff even when the text looks the same. To rewrite the files for
// the cases a test names, run the tests with -tuitest.update, or with
// TUITEST_UPDATE=1, which also works across ./..., or with the test binary's
// own boolean -update flag when it defines one.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package tuitest

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// updateFlag is tuitest's own flag. Its dotted name is one no consumer will
// define, so registering it cannot clash with a test binary's flags, as an
// -update registered here once did.
const updateFlag = "tuitest.update"

// updateEnv is the environment variable that rewrites golden files. A flag
// only reaches packages whose test binaries define it, but the environment
// reaches every package under go test ./...
const updateEnv = "TUITEST_UPDATE"

// consumerFlag is the conventional flag a consumer's test binary may define
// for its own golden files. tuitest honours it but never registers it.
const consumerFlag = "update"

// updateHint is how a failure says to write a missing golden file.
const updateHint = "run the test with -" + updateFlag + " or " + updateEnv + "=1"

var update = flag.Bool(updateFlag, false, "rewrite tuitest golden files")

// T is the part of *testing.T and *testing.B that Golden uses.
type T interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// Case is one cell of the matrix.
type Case struct {
	// Color is false for a rendering with no colour: the NO_COLOR or ASCII
	// colour profile.
	Color bool
	// UTF8 is false for a rendering restricted to ASCII glyphs.
	UTF8 bool
	// Width is the width, in cells, the view is given.
	Width int
}

// Name is the case's part of a golden file name, such as "color.utf8.80" or
// "nocolor.ascii.60".
func (c Case) Name() string {
	color, charset := "nocolor", "ascii"
	if c.Color {
		color = "color"
	}
	if c.UTF8 {
		charset = "utf8"
	}
	return color + "." + charset + "." + strconv.Itoa(c.Width)
}

// Matrix is the set of cases a test renders.
type Matrix struct {
	// Widths are the widths, in cells, each colour and charset combination
	// is rendered at. Rule 6 asks for at least two.
	Widths []int
}

// Cases returns every case: colour before no colour, UTF-8 before ASCII,
// then each width in order.
func (m Matrix) Cases() []Case {
	cases := make([]Case, 0, 4*len(m.Widths))
	for _, color := range []bool{true, false} {
		for _, utf8 := range []bool{true, false} {
			for _, w := range m.Widths {
				cases = append(cases, Case{Color: color, UTF8: utf8, Width: w})
			}
		}
	}
	return cases
}

// Golden renders every case of m and compares each with
// testdata/golden/<name>.<case>.golden, reporting every case that differs.
// When updating (see the package documentation) it writes the files instead,
// for these cases only.
func Golden(t T, name string, m Matrix, render func(Case) string) {
	t.Helper()
	compare(t, filepath.Join("testdata", "golden"), name, m, render, updating(), false)
}

// Text compares got with testdata/golden/<name>.golden, for a rendering
// that does not vary across the matrix, such as a geometry diagram.
// When updating it writes the file instead.
func Text(t T, name, got string) {
	t.Helper()
	textAt(t, filepath.Join("testdata", "golden"), name, got, updating())
}

func textAt(t T, dir, name, got string, update bool) {
	t.Helper()
	compare(t, dir, name, Matrix{Widths: []int{0}}, func(Case) string { return got }, update, true)
}

// updating reports whether golden files are to be rewritten: -tuitest.update
// is set, TUITEST_UPDATE is "1" or "true", or the test binary defines a
// boolean -update flag and it is set.
func updating() bool {
	if *update {
		return true
	}
	switch strings.ToLower(os.Getenv(updateEnv)) {
	case "1", "true":
		return true
	}
	f := flag.Lookup(consumerFlag)
	if f == nil {
		return false
	}
	g, ok := f.Value.(flag.Getter)
	if !ok {
		return false
	}
	on, ok := g.Get().(bool)
	return ok && on
}

// compare is Golden with the directory and the update switch explicit.
// Files are read and written through an os.Root on dir, so a name holding
// ".." cannot reach outside it.
func compare(t T, dir, name string, m Matrix, render func(Case) string, update, single bool) {
	t.Helper()
	if len(m.Widths) == 0 {
		t.Fatalf("tuitest: %s: the matrix has no widths", name)
		return
	}
	if update {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("tuitest: %v", err)
			return
		}
	}
	root, err := os.OpenRoot(dir)
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("tuitest: %s: no golden directory %s; "+updateHint, name, dir)
		return
	}
	if err != nil {
		t.Fatalf("tuitest: %v", err)
		return
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Errorf("tuitest: %v", err)
		}
	}()
	cases := m.Cases()
	if single {
		cases = cases[:1]
	}
	for _, c := range cases {
		file := name + "." + c.Name() + ".golden"
		if single {
			file = name + ".golden"
		}
		path := filepath.Join(dir, file)
		got := Annotate(render(c))
		if update {
			if err := root.WriteFile(file, []byte(got), 0o600); err != nil {
				t.Fatalf("tuitest: %v", err)
				return
			}
			continue
		}
		want, err := root.ReadFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("tuitest: %s: no golden file %s; "+updateHint, c.Name(), path)
			continue
		}
		if err != nil {
			t.Fatalf("tuitest: %v", err)
			return
		}
		if msg := diff(string(want), got); msg != "" {
			t.Errorf("tuitest: %s %s: %s", name, c.Name(), msg)
		}
	}
}

// Annotate puts each line's width in cells in front of it, as golden files
// store it. Every annotated line ends in a newline, whether or not s does, and
// an empty s is one empty line.
func Annotate(s string) string {
	if s == "" {
		return "  0|\n"
	}
	var b strings.Builder
	for l := range strings.Lines(s) {
		l = strings.TrimSuffix(l, "\n")
		fmt.Fprintf(&b, "%3d|%s\n", ansi.StringWidth(l), l)
	}
	return b.String()
}

// diff describes the first difference between two annotated renderings, with
// escape sequences made visible, or returns "" when they are equal.
func diff(want, got string) string {
	if want == got {
		return ""
	}
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	n := max(len(w), len(g))
	for i := range n {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return fmt.Sprintf("line %d differs\n  want %s\n  got  %s", i+1, strconv.Quote(wl), strconv.Quote(gl))
		}
	}
	return "the renderings differ"
}
