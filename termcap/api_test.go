package termcap

import (
	"go/importer"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

const uvPath = "github.com/charmbracelet/ultraviolet"

// TestNoUltravioletInTheAPI walks every exported name of the package, and
// every type reachable from one through exported fields, methods,
// parameters and results, and fails on any type from ultraviolet, which has
// no tagged release and stays behind internal/termevent
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md §1).
//
// The walk stops at an alias another package declares: tea.Msg is
// uv.Event's alias, and it is tea's API and tea's promise (A3). An alias
// declared in ultraviolet is ultraviolet's, and fails; one declared here is
// followed.
func TestNoUltravioletInTheAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks from source")
	}
	pkg, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import("github.com/maccavelli/go-tui-lib/termcap")
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range ultravioletIn(pkg) {
		t.Error(leak)
	}
}

// ultravioletIn names each exported path in pkg that reaches an ultraviolet
// type.
func ultravioletIn(pkg *types.Package) []string {
	var leaks []string
	seen := map[types.Type]bool{}
	var walk func(t types.Type, at string)
	walk = func(t types.Type, at string) {
		if seen[t] {
			return
		}
		seen[t] = true
		switch t := t.(type) {
		case *types.Named:
			if p := t.Obj().Pkg(); p != nil && strings.HasPrefix(p.Path(), uvPath) {
				leaks = append(leaks, at+": "+t.String())
				return
			}
			for a := range t.TypeArgs().Types() {
				walk(a, at)
			}
			for m := range t.Methods() {
				if m.Exported() {
					walk(m.Type(), at+"."+m.Name())
				}
			}
			walk(t.Underlying(), at)
		case *types.Alias:
			switch p := t.Obj().Pkg(); {
			case p == pkg:
				walk(t.Rhs(), at)
			case p != nil && strings.HasPrefix(p.Path(), uvPath):
				leaks = append(leaks, at+": "+t.String())
			}
		case *types.Pointer:
			walk(t.Elem(), at)
		case *types.Slice:
			walk(t.Elem(), at)
		case *types.Array:
			walk(t.Elem(), at)
		case *types.Map:
			walk(t.Key(), at)
			walk(t.Elem(), at)
		case *types.Chan:
			walk(t.Elem(), at)
		case *types.Signature:
			for v := range t.Params().Variables() {
				walk(v.Type(), at)
			}
			for v := range t.Results().Variables() {
				walk(v.Type(), at)
			}
		case *types.Struct:
			for f := range t.Fields() {
				if f.Exported() {
					walk(f.Type(), at+"."+f.Name())
				}
			}
		case *types.Interface:
			for m := range t.Methods() {
				walk(m.Type(), at+"."+m.Name())
			}
		}
	}
	scope := pkg.Scope()
	for _, n := range scope.Names() {
		if obj := scope.Lookup(n); obj.Exported() {
			walk(obj.Type(), n)
		}
	}
	return leaks
}
