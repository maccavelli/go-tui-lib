package termsvc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// processEnv is every function that reads the process's environment. The
// environment comes from tea.EnvMsg, which under wish is the SSH session's
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md).
var processEnv = map[string][]string{
	"os":      {"Getenv", "LookupEnv", "Environ", "ExpandEnv"},
	"syscall": {"Getenv", "Environ"},
}

// TestNoProcessEnvironmentOrProcesses fails on a read of the process's
// environment, or an import of os/exec, in the package's own files. The
// library returns commands; it never starts a process (MADR Q4).
func TestNoProcessEnvironmentOrProcesses(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range violations(fset, f) {
			t.Error(v)
		}
	}
}

// violations names each forbidden import and call in f.
func violations(fset *token.FileSet, f *ast.File) []string {
	var out []string
	local := map[string]string{} // the file's name for each package, to its path
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		at := fset.Position(imp.Pos()).String()
		if path == "os/exec" {
			out = append(out, at+": imports os/exec")
		}
		if _, ok := processEnv[path]; !ok {
			continue
		}
		name := path
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == "." {
			out = append(out, at+": dot-imports "+path)
		}
		local[name] = path
	}
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		path, ok := local[x.Name]
		if !ok {
			return true
		}
		for _, fn := range processEnv[path] {
			if sel.Sel.Name == fn {
				out = append(out, fset.Position(sel.Pos()).String()+": reads the process environment with "+path+"."+fn)
			}
		}
		return true
	})
	return out
}
