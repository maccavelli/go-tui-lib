package command

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/msgcat"
)

// The registry's catalogue (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md
// Step 3; 0024-MADR D9, A4).

// messageIDs is every exported msgcat.MessageID constant in the package's
// own Go files.
func messageIDs(t *testing.T) map[msgcat.MessageID]bool {
	t.Helper()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[msgcat.MessageID]bool{}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, s := range g.Specs {
				v := s.(*ast.ValueSpec)
				if sel, ok := v.Type.(*ast.SelectorExpr); !ok || sel.Sel.Name != "MessageID" {
					continue
				}
				for i, n := range v.Names {
					if lit, ok := v.Values[i].(*ast.BasicLit); ok && n.IsExported() {
						s, _ := strconv.Unquote(lit.Value)
						ids[msgcat.MessageID(s)] = true
					}
				}
			}
		}
	}
	if len(ids) == 0 {
		t.Fatal("no MessageID constant found")
	}
	return ids
}

// TestMessagesHoldEveryID: Messages has each MessageID constant the
// package declares, and nothing else; its verbs check against themselves.
func TestMessagesHoldEveryID(t *testing.T) {
	declared, table := messageIDs(t), Messages()
	for id := range declared {
		if _, ok := table[id]; !ok {
			t.Errorf("Messages has no %s", id)
		}
	}
	for id := range table {
		if !declared[id] {
			t.Errorf("Messages has %s, which no constant declares", id)
		}
	}
	if err := msgcat.Check(Messages(), Messages()); err != nil {
		t.Errorf("Check: %v", err)
	}
}

// TestCatalogMCP: each error text CallMCP gives, reworded alone, is the
// catalogue's, with the same arguments, and its default is the English.
func TestCatalogMCP(t *testing.T) {
	gate := WithGate(answer(AllowOnce, nil))
	for _, c := range []struct {
		id       msgcat.MessageID
		tool     string
		args     string
		gated    bool
		def      string
		reword   string
		reworded string
	}{
		{MsgMCPUnknown, "no.such", "{}", true,
			"There is no tool named no.such; list the tools to see which there are.", "Pas d'outil %s.", "Pas d'outil no.such."},
		{MsgMCPUnavailable, "not.now", "", true,
			`The tool not.now is not available now: command: not available: not.now: when "busy" is false`, "%[1]s indisponible : %[2]s", `not.now indisponible : command: not available: not.now: when "busy" is false`},
		{MsgMCPRefused, "session.reset", "", false,
			"The call to session.reset was refused: command: refused: mutating session.reset from agent needs a gate, and there is none. Do not retry it unchanged.",
			"%[1]s refusé : %[2]s", "session.reset refusé : command: refused: mutating session.reset from agent needs a gate, and there is none"},
		{MsgMCPArgs, "view.zoom", `{"pane":1}`, true,
			"The arguments do not fit the tool's input schema: command: argument /pane: is an integer, not string. Fix them and call again.", "Arguments : %s", "Arguments : command: argument /pane: is an integer, not string"},
		{MsgMCPFailed, "files.delete", `{"path":"x"}`, true,
			"The tool files.delete failed: no such file: x", "%[1]s a échoué : %[2]s", "files.delete a échoué : no such file: x"},
	} {
		text := func(o ...RegistryOption) string {
			if c.gated {
				o = append(o, gate)
			}
			res := catalogue(t, o...).CallMCP(t.Context(), c.tool, []byte(c.args), "agent-1")
			if !res.IsError || len(res.Content) != 1 {
				t.Fatalf("%s: %+v", c.id, res)
			}
			return res.Content[0].Text
		}
		if got := text(); got != c.def {
			t.Errorf("%s: %q, want %q", c.id, got, c.def)
		}
		if got := text(WithCatalog(msgcat.Table{c.id: c.reword})); got != c.reworded {
			t.Errorf("%s reworded: %q, want %q", c.id, got, c.reworded)
		}
	}
}

// TestCatalogDescribe: command.describe's reason and its two labels come
// from the catalogue, each alone.
func TestCatalogDescribe(t *testing.T) {
	describe := func(args string, o ...RegistryOption) (string, string) {
		res, err := catalogue(t, o...).Run(context.Background(), Request{ID: BuiltinDescribe, Args: []byte(args), Origin: OriginKey})
		if ae, ok := errors.AsType[*ArgError](err); ok {
			return "", ae.Reason
		}
		if err != nil {
			t.Fatal(err)
		}
		return res.Text, ""
	}
	if _, why := describe(`{"id":"zz"}`); why != "no command has this ID" {
		t.Errorf("the default reason: %q", why)
	}
	if _, why := describe(`{"id":"zz"}`, WithCatalog(msgcat.Table{MsgNoSuchCommand: "aucune commande"})); why != "aucune commande" {
		t.Errorf("the reason, reworded: %q", why)
	}
	plain, _ := describe(`{"id":"view.zoom"}`)
	if !strings.Contains(plain, "\ndanger: ui\n") || !strings.Contains(plain, "\narguments: {") {
		t.Fatalf("the default text:\n%s", plain)
	}
	if got, _ := describe(`{"id":"view.zoom"}`, WithCatalog(msgcat.Table{MsgDescribeDanger: "danger %s"})); got != strings.Replace(plain, "\ndanger: ui\n", "\ndanger ui\n", 1) {
		t.Errorf("the danger line, reworded:\n%s", got)
	}
	if got, _ := describe(`{"id":"view.zoom"}`, WithCatalog(msgcat.Table{MsgDescribeArguments: "args %s"})); got != strings.Replace(plain, "\narguments: {", "\nargs {", 1) {
		t.Errorf("the arguments line, reworded:\n%s", got)
	}
}
