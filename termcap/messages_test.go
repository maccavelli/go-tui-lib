package termcap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/msgcat"
)

// The report's catalogue (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md
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
// package declares, and nothing else; its verbs check against themselves;
// and every finding names two of them.
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
	for tok, f := range findingText {
		if f.message != msgcat.MessageID("termcap."+tok) || f.fix != msgcat.MessageID("termcap."+tok+".fix") {
			t.Errorf("%s: IDs %s and %s", tok, f.message, f.fix)
		}
	}
}

// reportWith is c's text report, unwrapped, under o.
func reportWith(t *testing.T, c Caps, o ...ReportOption) string {
	return reportOf(t, c, append([]ReportOption{WithReportWidth(0)}, o...)...)
}

// TestReportCatalogRewordsOneMessage: a finding's message, its fix and
// each note, reworded alone, change the text report by that text alone.
func TestReportCatalogRewordsOneMessage(t *testing.T) {
	timedOut := Caps{TimedOut: true}
	plain := reportWith(t, timedOut)
	for _, c := range []struct {
		id       msgcat.MessageID
		reword   string
		from, to string
	}{
		{MsgUnknownTerminal, "Nobody names the terminal.", "Nothing names the terminal, so capabilities that depend on it are off.", "Nobody names the terminal."},
		{MsgUnknownTerminalFix, "Name it.", "Set TERM_PROGRAM, or let LC_TERMINAL through SSH.", "Name it."},
		{MsgNoteTeaAsks, "tea asks", "tea asks only outside SSH and Apple Terminal, and not every terminal answers", "tea asks"},
		{MsgNoteOrTimeout, "%s, or timed out", "; or no reply before the timeout", ", or timed out"},
		{MsgNoteTimeout, "timed out", "(not-queried: no reply before the timeout)", "(not-queried: timed out)"},
	} {
		if !strings.Contains(plain, c.from) {
			t.Fatalf("%s: the default report holds no %q:\n%s", c.id, c.from, plain)
		}
		got := reportWith(t, timedOut, WithReportCatalog(msgcat.Table{c.id: c.reword}))
		if want := strings.ReplaceAll(plain, c.from, c.to); got != want {
			t.Errorf("%s:\n got %s\nwant %s", c.id, got, want)
		}
	}
}

// TestReportCatalogJSONAndASCII: the JSON report's findings take the
// catalogue's words as they are; the text report prints each byte
// outside ASCII as ?; Findings stays English.
func TestReportCatalogJSONAndASCII(t *testing.T) {
	cat := msgcat.Table{MsgUnknownTerminal: "Rien ne nomme le terminal, désolé."}
	if out := reportOf(t, Caps{}, WithReportJSON(), WithReportCatalog(cat)); !strings.Contains(out, `"message": "Rien ne nomme le terminal, désolé."`) {
		t.Errorf("the JSON report:\n%s", out)
	}
	if out := reportWith(t, Caps{}, WithReportCatalog(cat)); !strings.Contains(out, "Rien ne nomme le terminal, d??sol??.") {
		t.Errorf("the text report:\n%s", out)
	}
	for _, f := range Findings(Caps{}) {
		if f.ID == ReasonUnknownTerminal && f.Message != Messages()[MsgUnknownTerminal] {
			t.Errorf("Findings: %q", f.Message)
		}
	}
}
