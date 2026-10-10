// Package conformance checks every package against the rules no package may
// break (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rules 1
// and 2): nothing writes to standard output or standard error, nothing logs
// through log/slog's default logger, nothing calls signal.Notify, and
// nothing sets tea.View's AltScreen. Library code also never reads the
// process's environment, never exits and never starts a process
// (docs/decisions/0014-MADR-native-integration-api.md W0.5), except where
// allowed lists a file. And no two public packages export a type of the
// same name, unless docs/glossary.md says the clash is deliberate (W0.3);
// and every package's documentation ends with its stability line (W0.6).
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
//
// Stability: internal.
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
	"os/exec"
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

// declared is one exported type name of a public package, by the package's
// path relative to the repository root, with the object the scan resolved.
type declared struct {
	pkg, name string
	obj       *types.TypeName
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
		switch name {
		case "Stdout", "Stderr":
			return "uses os." + name
		case "Getenv", "LookupEnv", "Environ", "ExpandEnv":
			return "reads the environment with os." + name
		case "Exit":
			return "calls os.Exit"
		case "StartProcess":
			return "starts a process with os.StartProcess"
		}
	case "syscall":
		switch name {
		case "Getenv":
			return "reads the environment with syscall.Getenv"
		case "Exec", "ForkExec", "StartProcess":
			return "starts a process with syscall." + name
		}
	case "os/exec":
		return "uses os/exec." + name
	case teaPath:
		if name == "Exec" || name == "ExecProcess" {
			return "hands the terminal to a process with tea." + name
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
			case *ast.ImportSpec:
				// A blank import of os/exec uses no identifier, so the
				// import itself is the breach.
				if p, err := strconv.Unquote(v.Path.Value); err == nil && p == "os/exec" {
					report(v, "imports os/exec")
				}
			}
			return true
		})
	}
	return out
}

// typeCheck parses files, which are package importPath's, and type-checks
// them with imp. src, when not nil, holds a file's source by its name.
func typeCheck(t *testing.T, fset *token.FileSet, imp types.Importer, importPath string, names []string, src map[string]string) ([]*ast.File, *types.Info, *types.Package) {
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
	pkg, err := conf.Check(importPath, fset, files, info)
	if err != nil {
		t.Fatalf("type-checking %s: %v", importPath, err)
	}
	return files, info, pkg
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
// every breach, the packages it read, as paths relative to root, and the
// exported type names of its public packages.
func scanModule(t *testing.T, root, dir string) ([]violation, []string, []declared) {
	t.Helper()
	t.Chdir(dir)
	mod := modulePath(t, dir)
	fset := token.NewFileSet()
	imp := sourceImporter(fset)
	var out []violation
	var read []string
	var decl []declared
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
		files, info, pkg := typeCheck(t, fset, imp, importPath, names, nil)
		out = append(out, check(fset, files, info)...)
		fromRoot, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rootRel := filepath.ToSlash(fromRoot)
		read = append(read, rootRel)
		if pkg != nil && !slices.Contains(strings.Split(rootRel, "/"), "internal") {
			for _, name := range pkg.Scope().Names() {
				if tn, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok && tn.Exported() {
					decl = append(decl, declared{pkg: rootRel, name: name, obj: tn})
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out, read, decl
}

// allowed is every breach the rules permit, by file relative to the
// repository root and by rule, never by line, so an edit cannot move an
// entry off its use. tuitest reads TUITEST_UPDATE, its golden update switch,
// and runs only under go test
// (docs/decisions/0014-PLAN-api-policy-gates.md Step 2).
var allowed = map[[2]string]bool{
	{"tuitest/tuitest.go", "reads the environment with os.Getenv"}: true,
}

// goList runs go list -f format over the packages of the module in dir,
// with GOWORK=off, and returns its non-empty lines.
func goList(t *testing.T, dir, format string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", format, "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			t.Fatalf("go list in %s: %v\n%s", dir, err, ee.Stderr)
		}
		t.Fatalf("go list in %s: %v", dir, err)
	}
	var lines []string
	for line := range strings.Lines(string(out)) {
		if line = strings.TrimRight(line, "\r\n"); strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// mustRead lists every package of the module in dir that has non-test code,
// as go list sees it for this host, as paths relative to root.
func mustRead(t *testing.T, root, dir string) []string {
	t.Helper()
	var pkgs []string
	for _, line := range goList(t, dir, "{{if .GoFiles}}{{.Dir}}{{end}}") {
		rel, err := filepath.Rel(root, strings.TrimSpace(line))
		if err != nil {
			t.Fatal(err)
		}
		pkgs = append(pkgs, filepath.ToSlash(rel))
	}
	return pkgs
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
	var read, want []string
	used := map[[2]string]bool{}
	for _, dir := range modules(t, root) {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, mustRead(t, root, dir)...)
		t.Run(filepath.ToSlash(rel), func(t *testing.T) {
			found, pkgs, _ := scanModule(t, root, dir)
			read = append(read, pkgs...)
			for _, v := range found {
				r, _ := filepath.Rel(root, v.pos.Filename)
				key := [2]string{filepath.ToSlash(r), v.rule}
				if allowed[key] {
					used[key] = true
					continue
				}
				t.Errorf("%s:%d %s", key[0], v.pos.Line, v.rule)
			}
		})
	}
	// Every package with non-test code, as go list sees it, must have been
	// read.
	if len(want) == 0 {
		t.Fatal("go list found no package with non-test code")
	}
	for _, pkg := range want {
		if !slices.Contains(read, pkg) {
			t.Errorf("the scan did not read %s (it read %v)", pkg, read)
		}
	}
	// An allowlist entry that matches nothing is stale.
	for key := range allowed {
		if !used[key] {
			t.Errorf("allowed lists %s %q, which the scan no longer finds", key[0], key[1])
		}
	}
}

// sharedNames is every exported type name that more than one public package
// may declare, with the exact packages that do, sorted, as docs/glossary.md
// explains. "*" allows any package: each package's own option type
// (docs/decisions/0014-MADR-native-integration-api.md W0.3).
var sharedNames = map[string][]string{
	"Option":  {"*"},
	"Context": {"layout", "when"},
	"Kind":    {"command", "when"},
	"Origin":  {"command", "termcap"},
	"Pane":    {"layout", "workspace"},
	// A fake terminal for tests, in each test kit
	// (docs/decisions/0013-PLAN-cli-integration-helpers.md D5).
	"Terminal": {"launch/launchtest", "termcap/termcaptest"},
}

// TestNoTypeNameMeansTwoThings fails when two public packages export a type
// of the same name, unless sharedNames holds the name and its exact
// packages (docs/decisions/0014-PLAN-api-policy-gates.md Step 3).
func TestNoTypeNameMeansTwoThings(t *testing.T) {
	root := repoRoot(t)
	by := map[string][]string{}
	n := 0
	for _, dir := range modules(t, root) {
		_, _, decl := scanModule(t, root, dir)
		for _, d := range decl {
			by[d.name] = append(by[d.name], d.pkg)
			n++
		}
	}
	if n == 0 {
		t.Fatal("the scan collected no exported type name")
	}
	for name, pkgs := range by {
		if len(pkgs) < 2 {
			continue
		}
		slices.Sort(pkgs)
		allow, ok := sharedNames[name]
		switch {
		case !ok:
			t.Errorf("%s is exported by %v: one name, one meaning (docs/glossary.md)", name, pkgs)
		case slices.Equal(allow, []string{"*"}):
		case !slices.Equal(allow, pkgs):
			t.Errorf("%s is exported by %v, and sharedNames allows %v", name, pkgs, allow)
		}
	}
	for name, allow := range sharedNames {
		if len(by[name]) < 2 {
			t.Errorf("sharedNames lists %s %v, which fewer than two public packages export", name, allow)
		}
	}
}

// stableLine and internalLine end every package's documentation: a public
// package is stable, and one under an internal directory is internal
// (AGENTS.md, "API conventions";
// docs/decisions/0014-PLAN-api-policy-gates.md Step 6).
const (
	stableLine   = `Stability: stable. Exported names change only through the deprecation policy in AGENTS.md, "API conventions".`
	internalLine = "Stability: internal."
)

// TestEveryPackageStatesStability reads each package's documentation, test
// packages' included, and fails unless it is in one file and its last
// paragraph is the package's stability line.
func TestEveryPackageStatesStability(t *testing.T) {
	root := repoRoot(t)
	n := 0
	for _, dir := range modules(t, root) {
		// The directory, then its non-test files, then its test files.
		for _, line := range goList(t, dir, "{{.Dir}}\t{{join .GoFiles \" \"}}\t{{join .TestGoFiles \" \"}}") {
			parts := strings.Split(line, "\t")
			if len(parts) != 3 {
				t.Fatalf("go list printed %q", line)
			}
			names := strings.Fields(parts[1])
			if len(names) == 0 {
				names = strings.Fields(parts[2]) // a package of tests alone
			}
			rel, err := filepath.Rel(root, parts[0])
			if err != nil {
				t.Fatal(err)
			}
			rel = filepath.ToSlash(rel)
			n++
			fset := token.NewFileSet()
			var docs []string
			for _, name := range names {
				f, err := parser.ParseFile(fset, filepath.Join(parts[0], name), nil, parser.PackageClauseOnly|parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				if f.Doc != nil {
					docs = append(docs, f.Doc.Text())
				}
			}
			want := stableLine
			if slices.Contains(strings.Split(rel, "/"), "internal") {
				want = internalLine
			}
			switch len(docs) {
			case 0:
				t.Errorf("%s has no package documentation; it must end %q", rel, want)
				continue
			case 1:
			default:
				t.Errorf("%s is documented in %d files; one holds it", rel, len(docs))
				continue
			}
			paras := strings.Split(strings.TrimSpace(docs[0]), "\n\n")
			if last := strings.Join(strings.Fields(paras[len(paras)-1]), " "); last != want {
				t.Errorf("%s: its documentation ends %q, want %q", rel, last, want)
			}
		}
	}
	if n == 0 {
		t.Fatal("go list found no package")
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
		skip string // a GOOS where the planted source cannot compile
	}{
		{"an import alias", `import o "os"
func f() { _, _ = o.Stdout.Write(nil) }`, map[string]int{"uses os.Stdout": 1}, ""},
		{"a renamed fmt", `import xfmt "fmt"
func f() { xfmt.Println() }`, map[string]int{"prints to standard output with fmt.Println": 1}, ""},
		{"a dot import", `import . "fmt"
func f() { Printf("") }`, map[string]int{"prints to standard output with fmt.Printf": 1}, ""},
		{"function and method values", `import ("fmt"; "os")
func f() { p := fmt.Print; w := os.Stderr.Write; _, _ = p, w }`, map[string]int{"prints to standard output with fmt.Print": 1, "uses os.Stderr": 1}, ""},
		{"os.Stderr as a writer", `import ("fmt"; "os")
func f() { fmt.Fprintln(os.Stderr, "x") }`, map[string]int{"uses os.Stderr": 1}, ""},
		{"the log package", `import "log"
func f() { log.Print("x"); log.Default().Println("x") }`, map[string]int{"writes to standard error with log.Print": 1, "writes to standard error with log.Default": 1}, ""},
		{"the slog package", `import "log/slog"
func f() { slog.Info("x"); slog.Default().Info("x") }`, map[string]int{"logs through the default logger with slog.Info": 1, "logs through the default logger with slog.Default": 1}, ""},
		{"a renamed slog", `import ("context"; lg "log/slog")
func f(ctx context.Context) { lg.WarnContext(ctx, "x") }`, map[string]int{"logs through the default logger with slog.WarnContext": 1}, ""},
		{"the print builtins", `func f() { println("x"); print("x") }`, map[string]int{"calls the builtin println": 1, "calls the builtin print": 1}, ""},
		{"signal.Notify", `import ("os"; "os/signal")
func f(c chan os.Signal) { signal.Notify(c) }`, map[string]int{"calls signal.Notify": 1}, ""},
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
}`, map[string]int{"sets AltScreen": 3, "takes the address of AltScreen": 1}, ""},
		{"the environment", `import "os"
func f() { _ = os.Getenv("X"); _, _ = os.LookupEnv("X"); _ = os.Environ(); _ = os.ExpandEnv("$X") }`, map[string]int{
			"reads the environment with os.Getenv": 1, "reads the environment with os.LookupEnv": 1,
			"reads the environment with os.Environ": 1, "reads the environment with os.ExpandEnv": 1}, ""},
		{"the environment through an alias and a value", `import o "os"
func f() { g := o.Getenv; _ = g }`, map[string]int{"reads the environment with os.Getenv": 1}, ""},
		{"os.Exit", `import "os"
func f() { os.Exit(1) }`, map[string]int{"calls os.Exit": 1}, ""},
		{"os.StartProcess", `import "os"
func f() { _, _ = os.StartProcess("x", nil, nil) }`, map[string]int{"starts a process with os.StartProcess": 1}, ""},
		{"os/exec", `import ("context"; "os/exec")
func f(ctx context.Context) { var c *exec.Cmd = exec.CommandContext(ctx, "x"); _ = c }`, map[string]int{
			"imports os/exec": 1, "uses os/exec.Cmd": 1, "uses os/exec.CommandContext": 1}, ""},
		{"a blank os/exec", `import _ "os/exec"`, map[string]int{"imports os/exec": 1}, ""},
		{"syscall.StartProcess and Getenv", `import "syscall"
func f() { _, _, _ = syscall.StartProcess("x", nil, nil); _, _ = syscall.Getenv("X") }`, map[string]int{
			"starts a process with syscall.StartProcess": 1, "reads the environment with syscall.Getenv": 1}, ""},
		// ForkExec does not exist on Windows (deviation D1 of
		// docs/decisions/0014-PLAN-api-policy-gates.md).
		{"syscall.ForkExec", `import "syscall"
func f() { _, _ = syscall.ForkExec("x", nil, nil) }`, map[string]int{"starts a process with syscall.ForkExec": 1}, "windows"},
		{"tea.ExecProcess", `import ("os/exec"; tea "charm.land/bubbletea/v2")
func f(c *exec.Cmd) tea.Cmd { return tea.ExecProcess(c, nil) }`, map[string]int{
			"hands the terminal to a process with tea.ExecProcess": 1, "imports os/exec": 1, "uses os/exec.Cmd": 1}, ""},
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
}`, map[string]int{}, ""},
	}
	fset := token.NewFileSet()
	imp := sourceImporter(fset)
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.skip == runtime.GOOS {
				t.Skipf("the planted source does not compile on %s", c.skip)
			}
			name := filepath.Join(dir, "planted"+strconv.Itoa(i)+".go")
			src := "package planted\n" + c.src + "\n"
			files, info, _ := typeCheck(t, fset, imp, "example.com/planted"+strconv.Itoa(i), []string{name}, map[string]string{name: src})
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
