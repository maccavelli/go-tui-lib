package palette

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// The picker's catalogue (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md
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

// frameOf is the picker's frame for query over providers, once settled.
func frameOf(t *testing.T, query string, providers []Provider, opts ...Option) string {
	g := newProg(t, keys(t), nil, opts...)
	g.send(tea.WindowSizeMsg{Width: 62, Height: 14})
	for _, p := range providers {
		g.register(p)
	}
	g.open(WithSize(62, 12))
	g.typed(query)
	time.Sleep(frame)
	g.settle()
	out := g.view()
	g.send(workspace.Close("palette")) // ends a search still running
	g.settle()
	return out
}

// TestCatalogRewordsOneMessage: a Table that rewords one message, to text
// of the same width, changes that text in the frame and nothing else.
func TestCatalogRewordsOneMessage(t *testing.T) {
	three := func() []Provider {
		return []Provider{&fake{name: "f", search: yields(hit("xa", fuzzy.Prefix, 3), hit("xb", fuzzy.Prefix, 2), hit("xc", fuzzy.Prefix, 1))}}
	}
	slow := func() []Provider {
		return []Provider{&fake{name: "slow", search: stalled(hit("logs", fuzzy.Prefix, 2), hit("lock", fuzzy.Prefix, 1))}}
	}
	cases := []struct {
		name      string
		query     string
		providers func() []Provider
		id        msgcat.MessageID
		reword    string
		from, to  string
	}{
		{"count", "x", three, MsgCount, "%[1]d de %[2]d", "1 of 3", "1 de 3"},
		{"no results", "zz", func() []Provider { return []Provider{&fake{name: "f", search: yields()}} }, MsgNoResults, "pas trouvé", "no results", "pas trouvé"},
		{"searching with rows", "lo", slow, MsgCountSearching, "%[1]d de %[2]d, cherchant", "1 of 2, searching", "1 de 2, cherchant"},
		{"searching", "lo", func() []Provider { return []Provider{&fake{name: "slow", search: stalled()}} }, MsgSearching, "cherchant", "searching", "cherchant"},
		{"failed", "x", func() []Provider { return append(three(), failing("files")) }, MsgFailed, "%s broken", "files failed", "files broken"},
	}
	synctest.Test(t, func(t *testing.T) {
		for _, c := range cases {
			plain := frameOf(t, c.query, c.providers())
			if !strings.Contains(plain, c.from) {
				t.Fatalf("%s: the default frame holds no %q:\n%s", c.name, c.from, plain)
			}
			got := frameOf(t, c.query, c.providers(), WithCatalog(msgcat.Table{c.id: c.reword}))
			if want := strings.ReplaceAll(plain, c.from, c.to); got != want {
				t.Errorf("%s:\n got %s\nwant %s", c.name, got, want)
			}
		}
	})
}

// TestCatalogArgumentReasons: the argument stage's count and each reason,
// reworded one at a time, change the help line by that text alone.
func TestCatalogArgumentReasons(t *testing.T) {
	// help is the help line after keys, under opts.
	help := func(t *testing.T, keys2 []string, opts ...Option) string {
		reg, _ := deploying(t)
		g := newProg(t, keys(t), nil, opts...)
		g.reg = reg
		g.register(FromRegistry(reg))
		g.open()
		g.typed("deploy")
		g.key("enter")
		for _, k := range keys2 {
			if len(k) > 1 && k[0] == '=' {
				g.typed(k[1:])
				continue
			}
			g.key(k)
		}
		return g.help()
	}
	cases := []struct {
		name     string
		keys     []string
		id       msgcat.MessageID
		reword   string
		from, to string
	}{
		{"the count", nil, MsgArgumentCount, "argument %[1]d de %[2]d", "argument 1 of 5", "argument 1 de 5"},
		{"refused", []string{"enter"}, MsgRefused, "%[1]s! %[2]s", "target: required", "target! required"},
		{"required", []string{"enter"}, MsgRequired, "demandé!", "target: required", "target: demandé!"},
		{"not a whole number", []string{"=web", "enter", "=x", "enter"}, MsgNotWholeNumber, "pas entier du tout", "count: not a whole number", "count: pas entier du tout"},
		{"at least", []string{"=web", "enter", "=0", "enter"}, MsgAtLeast, "au moins %s", "count: at least 1", "count: au moins 1"},
		{"at most", []string{"=web", "enter", "=12", "enter"}, MsgAtMost, "au plus %s", "count: at most 9", "count: au plus 9"},
	}
	synctest.Test(t, func(t *testing.T) {
		for _, c := range cases {
			plain := help(t, c.keys)
			if !strings.Contains(plain, c.from) {
				t.Fatalf("%s: the default help line holds no %q: %q", c.name, c.from, plain)
			}
			got := help(t, c.keys, WithCatalog(msgcat.Table{c.id: c.reword}))
			if want := strings.ReplaceAll(plain, c.from, c.to); got != want {
				t.Errorf("%s: %q, want %q", c.name, got, want)
			}
		}
	})
}

// TestCatalogValueReasons: a list's and a number's reasons come from the
// catalogue too.
func TestCatalogValueReasons(t *testing.T) {
	cat := msgcat.Table{MsgPickOne: "choisissez une valeur", MsgNotNumber: "pas un nombre"}
	if _, _, why := value(command.Param{Name: "ratio", Type: "number"}, "x", cat); why != "pas un nombre" {
		t.Errorf("a number: %q", why)
	}
	st := &stage{params: []command.Param{{Name: "mode", Type: "string", Required: true, Enum: []any{"fast"}}}, cat: cat}
	st.choices = []choice{{text: "fast", v: "fast"}}
	if st.give("", -1) || st.reason != "mode: choisissez une valeur" {
		t.Errorf("a list: %q", st.reason)
	}
}

// TestSearchCarriesTheCatalog: a provider is asked with the picker's
// WithCatalog in its Search, and with none without it.
func TestSearchCarriesTheCatalog(t *testing.T) {
	cat := msgcat.Table{MsgNoResults: "aucun"}
	for _, c := range []struct {
		name string
		opts []Option
		want msgcat.Catalog
	}{{"with", []Option{WithCatalog(cat)}, cat}, {"without", nil, nil}} {
		synctest.Test(t, func(t *testing.T) {
			var got []msgcat.Catalog
			seen := func(_ context.Context, q Search, _ func(Hit) bool) error {
				got = append(got, q.Catalog)
				return nil
			}
			frameOf(t, "x", []Provider{&fake{name: "f", search: seen}}, c.opts...)
			if len(got) == 0 {
				t.Fatalf("%s: the provider was not asked", c.name)
			}
			for _, g := range got {
				if !reflect.DeepEqual(g, c.want) {
					t.Errorf("%s: Search.Catalog %v, want %v", c.name, g, c.want)
				}
			}
		})
	}
}

// TestCatalogFilesLimit: the files provider words its limit row from the
// Search's catalogue.
func TestCatalogFilesLimit(t *testing.T) {
	f := startedFiles(t, WithMaxFiles(1))
	limit := func(c msgcat.Catalog) Hit {
		var out Hit
		_ = f.Search(t.Context(), Search{Text: "x", Matcher: fuzzy.NewMatcher("x"), Catalog: c}, func(h Hit) bool {
			if h.ID == "files:limit" {
				out = h
			}
			return true
		})
		return out
	}
	if h := limit(nil); h.Text != "only the first 1 files are searched" || h.Disabled != "the file limit" {
		t.Fatalf("the default: %+v", h)
	}
	h := limit(msgcat.Table{MsgFileLimit: "seuls les %d premiers", MsgFileLimitReason: "la limite"})
	if h.Text != "seuls les 1 premiers" || h.Disabled != "la limite" {
		t.Errorf("the catalogue's: %+v", h)
	}
}
