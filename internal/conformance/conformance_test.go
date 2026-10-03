// Package conformance checks every package against the rules no package may
// break (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rules 1
// and 2): nothing writes to standard output or standard error, nothing logs
// through log/slog's default logger, nothing calls signal.Notify, and
// nothing sets tea.View's AltScreen.
//
// The scan type-checks each package with go/types and the source importer,
// and resolves every identifier to the object it denotes. A renamed or dot
// import, a function value or a method value names the same object, so none
// of them hides a use, and a local name that merely matches a rule is not a
// use. Test files are not scanned.
//
// Each module is scanned in its own context, from its own directory: the
// source importer resolves imports with go list in the working directory,
// and one module's build list does not load another's packages
// (docs/decisions/0010-MADR-nested-adapter-modules.md §4).
//
// Files are read for the host's GOOS. CI runs this test on Linux, macOS and
// Windows, so each operating system's files are scanned there.
package conformance

import (
	"bufio"
	"errors"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const teaPath = "charm.land/bubbletea/v2"

// violation is one breach of a rule, by file and line.
type violation struct {
	pos  token.Position
	rule string
}

// stderrLog is every function of package log that writes to standard error
// through the standard logger, or hands that logger or its writer out.
var stderrLog = []string{
	"Print", "Printf", "Println",
	"Fatal", "Fatalf", "Fatalln",
	"Panic", "Panicf", "Panicln",
	"Output", "Default", "Writer",
}

// defaultSlog is every function of package log/slog that logs through the
// default logger, or hands it out. That logger writes to standard error
// until the program replaces it, and it is the program's either way: a
// package logs through a *slog.Logger or a handler its caller passes
// (docs/decisions/0002-PLAN-harden-workspace-v0-1-1.md Step 7a).
var defaultSlog = []string{
	"Debug", "DebugContext", "Info", "InfoContext",
	"Warn", "WarnContext", "Error", "ErrorContext",
	"Log", "LogAttrs", "Default",
}

// forbidden returns the rule a use of obj breaks, or "".
func forbidden(obj types.Object) string {
	switch o := obj.(type) {
	case *types.Builtin:
		if o.Name() == "print" || o.Name() == "println" {
			return "calls the builtin " + o.Name()
		}
		return ""
	case nil:
		return ""
	}
	pkg := obj.Pkg()
	if pkg == nil || obj.Parent() != pkg.Scope() {
		return "" // not a package-level object: a field, a method or a local
	}
	name := obj.Name()
	switch pkg.Path() {
	case "os":
		if name == "Stdout" || name == "Stderr" {
			return "uses os." + name
		}
	case "fmt":
		if name == "Print" || name == "Printf" || name == "Println" {
			return "prints to standard output with fmt." + name
		}
	case "log":
		if slices.Contains(stderrLog, name) {
			return "writes to standard error with log." + name
		}
	case "log/slog":
		if slices.Contains(defaultSlog, name) {
			return "logs through the default logger with slog." + name
		}
	case "os/signal":
		if name == "Notify" {
			return "calls signal.Notify"
		}
	}
	return ""
}

// isAltScreen reports whether obj is tea.View's AltScreen field, however it
// is reached.
func isAltScreen(obj types.Object) bool {
	v, ok := obj.(*types.Var)
	if !ok || !v.IsField() || v.Name() != "AltScreen" || v.Pkg() == nil || v.Pkg().Path() != teaPath {
		return false
	}
	view, ok := v.Pkg().Scope().Lookup("View").(*types.TypeName)
	if !ok {
		return false
	}
	st, ok := view.Type().Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for f := range st.Fields() {
		if f == v {
			return true
		}
	}
	return false
}

// writesAltScreen reports whether e, as an assignment's target or an
// address taken, is tea.View's AltScreen field.
func writesAltScreen(info *types.Info, e ast.Expr) bool {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	return ok && isAltScreen(info.Uses[sel.Sel])
}

// check returns every breach in files, which info describes.
func check(fset *token.FileSet, files []*ast.File, info *types.Info) []violation {
	var out []violation
	report := func(n ast.Node, rule string) {
		out = append(out, violation{fset.Position(n.Pos()), rule})
	}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.Ident:
				if rule := forbidden(info.Uses[v]); rule != "" {
					report(v, rule)
				}
			case *ast.AssignStmt:
				for _, l := range v.Lhs {
					if writesAltScreen(info, l) {
						report(l, "sets AltScreen")
					}
				}
			case *ast.UnaryExpr:
				if v.Op == token.AND && writesAltScreen(info, v.X) {
					report(v, "takes the address of AltScreen")
				}
			case *ast.KeyValueExpr:
				if k, ok := v.Key.(*ast.Ident); ok && isAltScreen(info.Uses[k]) {
					report(v, "sets AltScreen")
				}
			}
			return true
		})
	}
	return out
}

// typeCheck parses files, which are package importPath's, and type-checks
// them with imp. src, when not nil, holds a file's source by its name.
func typeCheck(t *testing.T, fset *token.FileSet, imp types.Importer, importPath string, names []string, src map[string]string) ([]*ast.File, *types.Info) {
	t.Helper()
	var files []*ast.File
	for _, name := range names {
		var s any
		if src != nil {
			s = src[name]
		}
		f, err := parser.ParseFile(fset, name, s, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: imp}
	if _, err := conf.Check(importPath, fset, files, info); err != nil {
		t.Fatalf("type-checking %s: %v", importPath, err)
	}
	return files, info
}

// sourceImporter returns the source importer. cgo is off, as in every
// cross-target build here, so that no C toolchain is needed to read a
// dependency.
func sourceImporter(fset *token.FileSet) types.Importer {
	build.Default.CgoEnabled = false
	return importer.ForCompiler(fset, "source", nil)
}

// modulePath reads the module path from dir's go.mod.
func modulePath(t *testing.T, dir string) string {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module "); ok {
			rest = strings.TrimSpace(rest)
			if p, err := strconv.Unquote(rest); err == nil {
				return p
			}
			return rest
		}
	}
	t.Fatalf("%s/go.mod names no module", dir)
	return ""
}

// skipDir reports whether the go command ignores directory name.
func skipDir(name string) bool {
	return name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// modules returns the directory of every module under root, root first.
func modules(t *testing.T, root string) []string {
	t.Helper()
	out := []string{root}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || p == root {
			return nil
		}
		if skipDir(d.Name()) {
			return filepath.SkipDir
		}
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// scanModule type-checks every package of the module in dir, and returns
// every breach and the packages it read, as paths relative to root.
func scanModule(t *testing.T, root, dir string) ([]violation, []string) {
	t.Helper()
	t.Chdir(dir)
	mod := modulePath(t, dir)
	fset := token.NewFileSet()
	imp := sourceImporter(fset)
	var out []violation
	var read []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if p != dir {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				return filepath.SkipDir // its own module, scanned on its own
			}
		}
		bp, err := build.Default.ImportDir(p, 0)
		var none *build.NoGoError
		if errors.As(err, &none) || (err == nil && len(bp.GoFiles) == 0) {
			return nil
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		importPath := mod
		if rel != "." {
			importPath = path.Join(mod, filepath.ToSlash(rel))
		}
		names := make([]string, len(bp.GoFiles))
		for i, f := range bp.GoFiles {
			names[i] = filepath.Join(p, f)
		}
		files, info := typeCheck(t, fset, imp, importPath, names, nil)
		out = append(out, check(fset, files, info)...)
		fromRoot, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		read = append(read, filepath.ToSlash(fromRoot))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out, read
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestNoPackageOwnsTheTerminal(t *testing.T) {
	root := repoRoot(t)
	var read []string
	for _, dir := range modules(t, root) {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(filepath.ToSlash(rel), func(t *testing.T) {
			found, pkgs := scanModule(t, root, dir)
			read = append(read, pkgs...)
			for _, v := range found {
				r, _ := filepath.Rel(root, v.pos.Filename)
				t.Errorf("%s:%d %s", filepath.ToSlash(r), v.pos.Line, v.rule)
			}
		})
	}
	// Every package with non-test code must have been read.
	for _, pkg := range []string{"glyph", "layout", "theme", "tuitest", "tuitest/internal/clash", "workspace"} {
		if !slices.Contains(read, pkg) {
			t.Errorf("the scan did not read %s (it read %v)", pkg, read)
		}
	}
}

// TestScanFindsEachRule type-checks planted sources as if they were a
// package in this directory, so that their imports resolve in this module.
// Nothing is written to the tree.
func TestScanFindsEachRule(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	dir := filepath.Dir(self)
	cases := []struct {
		name string
		src  string
		want map[string]int
	}{
		{"an import alias", `import o "os"
func f() { _, _ = o.Stdout.Write(nil) }`, map[string]int{"uses os.Stdout": 1}},
		{"a renamed fmt", `import xfmt "fmt"
func f() { xfmt.Println() }`, map[string]int{"prints to standard output with fmt.Println": 1}},
		{"a dot import", `import . "fmt"
func f() { Printf("") }`, map[string]int{"prints to standard output with fmt.Printf": 1}},
		{"function and method values", `import ("fmt"; "os")
func f() { p := fmt.Print; w := os.Stderr.Write; _, _ = p, w }`, map[string]int{"prints to standard output with fmt.Print": 1, "uses os.Stderr": 1}},
		{"os.Stderr as a writer", `import ("fmt"; "os")
func f() { fmt.Fprintln(os.Stderr, "x") }`, map[string]int{"uses os.Stderr": 1}},
		{"the log package", `import "log"
func f() { log.Print("x"); log.Default().Println("x") }`, map[string]int{"writes to standard error with log.Print": 1, "writes to standard error with log.Default": 1}},
		{"the slog package", `import "log/slog"
func f() { slog.Info("x"); slog.Default().Info("x") }`, map[string]int{"logs through the default logger with slog.Info": 1, "logs through the default logger with slog.Default": 1}},
		{"a renamed slog", `import ("context"; lg "log/slog")
func f(ctx context.Context) { lg.WarnContext(ctx, "x") }`, map[string]int{"logs through the default logger with slog.WarnContext": 1}},
		{"the print builtins", `func f() { println("x"); print("x") }`, map[string]int{"calls the builtin println": 1, "calls the builtin print": 1}},
		{"signal.Notify", `import ("os"; "os/signal")
func f(c chan os.Signal) { signal.Notify(c) }`, map[string]int{"calls signal.Notify": 1}},
		{"AltScreen", `import tea "charm.land/bubbletea/v2"
type wrapped struct{ tea.View }
func f() {
	v := tea.View{}
	v.AltScreen = true
	_ = tea.View{AltScreen: true}
	var w wrapped
	w.AltScreen = true
	p := &v.AltScreen
	_ = p
	_ = v.AltScreen // a read
}`, map[string]int{"sets AltScreen": 3, "takes the address of AltScreen": 1}},
		{"names that only match", `import ("fmt"; "io"; "log"; "log/slog")
type view struct{ AltScreen bool }
// os.Stdout, fmt.Println and AltScreen in a comment.
func Println() {}
func f(os struct{ Stdout io.Writer }, l *log.Logger, sl *slog.Logger, h slog.Handler) {
	fmt.Fprintln(os.Stdout, "x")
	l.Println("x")
	sl.Info("x")
	_ = slog.New(h)
	Println()
	v := view{AltScreen: true}
	v.AltScreen = true
}`, map[string]int{}},
	}
	fset := token.NewFileSet()
	imp := sourceImporter(fset)
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			name := filepath.Join(dir, "planted"+strconv.Itoa(i)+".go")
			src := "package planted\n" + c.src + "\n"
			files, info := typeCheck(t, fset, imp, "example.com/planted"+strconv.Itoa(i), []string{name}, map[string]string{name: src})
			got := map[string]int{}
			for _, v := range check(fset, files, info) {
				got[v.rule]++
			}
			if len(got) != len(c.want) {
				t.Errorf("found %v, want %v", got, c.want)
			}
			for rule, n := range c.want {
				if got[rule] != n {
					t.Errorf("%q found %d times, want %d (all: %v)", rule, got[rule], n, got)
				}
			}
		})
	}
}
