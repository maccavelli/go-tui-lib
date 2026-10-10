package conformance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"
)

// TestOptionsAreOpaque: no program can make its own option of the six
// option types that were functions over exported structs: a func(*T)
// literal assigned to each fails to type-check, and so does a type of its
// own with an apply method, or an Apply one, while the library's own option
// type-checks
// (docs/decisions/0014-PLAN-canonicalization.md Step 5, 0014-MADR W0.4).
// The sources are planted, and type-checked as the scan type-checks a
// package, so an import that fails cannot pass for opacity.
func TestOptionsAreOpaque(t *testing.T) {
	mod := modulePath(t, repoRoot(t))
	header := `package planted

import (
	tea "charm.land/bubbletea/v2"

	"` + mod + `/command"
	"` + mod + `/termcap"
	"` + mod + `/termsvc"
	"` + mod + `/workspace"
)

type bubble struct{}

func (bubble) Update(tea.Msg) (bubble, tea.Cmd) { return bubble{}, nil }
func (bubble) View() string                     { return "" }

var (
	_ = command.UI
	_ = termcap.Supported
	_ = termsvc.Bell
	_ workspace.Pane
	_ tea.Cmd
)
`
	cases := []struct {
		typ, target, library string
	}{
		{"command.Option", "*command.Command", "command.WithDanger(command.UI)"},
		{"command.RegistryOption", "*command.Registry", "command.WithMaxArgBytes(1)"},
		{"termcap.Option", "*termcap.Prober", "termcap.WithGOOS(\"linux\")"},
		{"termsvc.NotifyOption", "*termsvc.Notifier", "termsvc.WithProtocol(termsvc.Bell)"},
		{"workspace.Option", "*workspace.Workspace", "workspace.WithoutMouse()"},
		{"workspace.WrapOption[bubble]", "*workspace.Model[bubble]", "workspace.OnSize[bubble](nil)"},
	}
	fset := token.NewFileSet()
	imp := sourceImporter(fset)
	check := func(src string) error {
		name := filepath.Join(t.TempDir(), "planted.go")
		f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		_, err = (&types.Config{Importer: imp}).Check("example.com/planted", fset, []*ast.File{f}, nil)
		return err
	}
	for _, c := range cases {
		if err := check(header + "var _ " + c.typ + " = " + c.library + "\n"); err != nil {
			t.Errorf("%s: the library's own option does not type-check: %v", c.typ, err)
			continue
		}
		for what, src := range map[string]string{
			"a func literal": "var _ " + c.typ + " = func(" + c.target + ") {}\n",
			"a type of its own": "type mine struct{}\n\nfunc (mine) apply(" + c.target + ") {}\nfunc (mine) Apply(" + c.target + ") {}\n\n" +
				"var _ " + c.typ + " = mine{}\n",
		} {
			err := check(header + src)
			if err == nil || !strings.Contains(err.Error(), "does not implement") {
				t.Errorf("%s, %s: %v; want it refused as not implementing the type", c.typ, what, err)
			}
		}
	}
}
