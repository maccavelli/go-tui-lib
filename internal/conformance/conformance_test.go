// Package conformance checks every package against the rules no package may
// break (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rules 1
// and 2): nothing writes to os.Stdout or os.Stderr, nothing calls
// signal.Notify, and nothing sets AltScreen. It parses the code rather than
// searching the text, so a comment naming a rule does not trip it.
package conformance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// violation is one breach of a rule, by file and line.
type violation struct {
	pos  token.Position
	rule string
}

// scan parses every non-test Go file under root, skipping testdata, and
// returns every breach and the directories whose files it parsed.
func scan(t *testing.T, root string) ([]violation, map[string]bool) {
	t.Helper()
	fset := token.NewFileSet()
	var out []violation
	dirs := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) && path != root {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		out = append(out, check(fset, f)...)
		if rel, err := filepath.Rel(root, filepath.Dir(path)); err == nil {
			dirs[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out, dirs
}

func check(fset *token.FileSet, f *ast.File) []violation {
	var out []violation
	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.SelectorExpr:
			if x, ok := v.X.(*ast.Ident); ok {
				switch {
				case x.Name == "os" && (v.Sel.Name == "Stdout" || v.Sel.Name == "Stderr"):
					out = append(out, violation{fset.Position(v.Pos()), "writes to os." + v.Sel.Name})
				case x.Name == "signal" && v.Sel.Name == "Notify":
					out = append(out, violation{fset.Position(v.Pos()), "calls signal.Notify"})
				}
			}
			if v.Sel.Name == "AltScreen" {
				out = append(out, violation{fset.Position(v.Pos()), "sets AltScreen"})
			}
		case *ast.KeyValueExpr:
			if k, ok := v.Key.(*ast.Ident); ok && k.Name == "AltScreen" {
				out = append(out, violation{fset.Position(v.Pos()), "sets AltScreen"})
			}
		}
		return true
	})
	return out
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestNoPackageOwnsTheTerminal(t *testing.T) {
	root := repoRoot(t)
	found, dirs := scan(t, root)
	// Every package with non-test code must have been read.
	for _, pkg := range []string{"glyph", "layout", "theme", "tuitest", "workspace"} {
		if !dirs[pkg] {
			t.Errorf("the scan did not read %s (it read %v)", pkg, dirs)
		}
	}
	for _, v := range found {
		rel, _ := filepath.Rel(root, v.pos.Filename)
		t.Errorf("%s:%d %s", rel, v.pos.Line, v.rule)
	}
}

func TestScanFindsEachRule(t *testing.T) {
	src := `package p

import (
	"fmt"
	"os"
	"os/signal"
)

// AltScreen in a comment is fine.
type view struct{ AltScreen bool }

func f() {
	fmt.Fprintln(os.Stdout, "x")
	fmt.Fprintln(os.Stderr, "x")
	signal.Notify(nil)
	v := view{AltScreen: true}
	v.AltScreen = true
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, v := range check(fset, f) {
		got[v.rule]++
	}
	want := map[string]int{"writes to os.Stdout": 1, "writes to os.Stderr": 1, "calls signal.Notify": 1, "sets AltScreen": 2}
	for rule, n := range want {
		if got[rule] != n {
			t.Errorf("%q found %d times, want %d (all: %v)", rule, got[rule], n, got)
		}
	}
}
