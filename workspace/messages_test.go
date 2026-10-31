package workspace_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// The workspace's catalogue (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md
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
	declared, table := messageIDs(t), workspace.Messages()
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
	if err := msgcat.Check(workspace.Messages(), workspace.Messages()); err != nil {
		t.Errorf("Check: %v", err)
	}
}

// said is what running id with args says: its result's text, or its
// ArgError's reason. Through the registry unless direct, which calls the
// handler as the registry would after the schema passed, to reach a
// reason the schema's enum keeps from it.
func said(t *testing.T, cat msgcat.Catalog, id command.ID, args string, direct bool) string {
	t.Helper()
	th := theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())
	var opts []workspace.Option
	if cat != nil {
		opts = append(opts, workspace.WithCatalog(cat))
	}
	ws := session(agentRoot(false, layout.FullWidth), th, 120, 30, opts...)
	cmds := workspace.Commands(ws, workspace.WithLayouts(layouts()))
	var res command.Result
	var err error
	if direct {
		i := slices.IndexFunc(cmds, func(c command.Command) bool { return c.ID == id })
		res, err = cmds[i].Handler.Run(context.Background(), &command.Invocation{Command: cmds[i], Args: []byte(args), WhenContext: ws.WhenContext()})
	} else {
		r := command.NewRegistry()
		if err := r.Register(cmds...); err != nil {
			t.Fatal(err)
		}
		res, err = r.Run(context.Background(), command.Request{ID: id, Args: []byte(args), Origin: command.OriginAgent, WhenContext: ws.WhenContext()})
	}
	if ae, ok := errors.AsType[*command.ArgError](err); ok {
		return ae.Reason
	}
	if err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	return res.Text
}

// TestCatalogRewordsOneMessage: each result and reason, reworded alone,
// is the catalogue's text, and its default is the English one.
func TestCatalogRewordsOneMessage(t *testing.T) {
	for _, c := range []struct {
		id       msgcat.MessageID
		cmd      command.ID
		args     string
		direct   bool
		def      string
		reword   string
		reworded string
	}{
		{workspace.MsgNoPaneIndex, "workspace.focus.index", `{"index":9}`, false, "No pane 9 in the focus ring.", "Pas de volet %d.", "Pas de volet 9."},
		{workspace.MsgNoOverlay, "workspace.overlay.close", `{}`, false, "No overlay is open.", "Aucune surcouche.", "Aucune surcouche."},
		{workspace.MsgNotFocusable, "workspace.focus", `{"pane":"footer"}`, false, "is not a pane on screen that takes focus", "ne prend pas le focus", "ne prend pas le focus"},
		{workspace.MsgNotAPane, "workspace.zoom", `{"pane":"typo"}`, false, "is not a pane of the layout", "n'est pas un volet", "n'est pas un volet"},
		{workspace.MsgNotAPane, "workspace.toggle", `{"pane":"typo"}`, false, "is not a pane of the layout", "n'est pas un volet", "n'est pas un volet"},
		{workspace.MsgNotADirection, "workspace.resize.direction", `{"direction":"sideways"}`, true, "is not left, right, up or down", "n'est pas une direction", "n'est pas une direction"},
		{workspace.MsgNotASplit, "workspace.resize", `{"split":"nothing","delta":1}`, false, "is not a resizable split on screen", "n'est pas une séparation", "n'est pas une séparation"},
		{workspace.MsgNotALayout, "workspace.layout.use", `{"name":"nope"}`, true, "is not a layout's name", "n'est pas une disposition", "n'est pas une disposition"},
	} {
		if got := said(t, nil, c.cmd, c.args, c.direct); got != c.def {
			t.Errorf("%s by %s: %q, want %q", c.id, c.cmd, got, c.def)
		}
		if got := said(t, msgcat.Table{c.id: c.reword}, c.cmd, c.args, c.direct); got != c.reworded {
			t.Errorf("%s by %s, reworded: %q, want %q", c.id, c.cmd, got, c.reworded)
		}
	}
}

// TestCatalogManySplits: the reason a split's name names two separators.
func TestCatalogManySplits(t *testing.T) {
	reason := func(cat msgcat.Catalog) string {
		var opts []workspace.Option
		if cat != nil {
			opts = append(opts, workspace.WithCatalog(cat))
		}
		free := workspace.New(layout.Split{Name: "row", Axis: layout.Horizontal, Children: []layout.Child{
			{Node: layout.Pane{ID: "a"}}, {Node: layout.Pane{ID: "b"}}, {Node: layout.Pane{ID: "c"}},
		}}, map[layout.PaneID]workspace.Pane{"a": &footer{}, "b": &footer{}, "c": &footer{}}, opts...)
		free.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
		r := command.NewRegistry()
		if err := r.Register(workspace.Commands(free)...); err != nil {
			t.Fatal(err)
		}
		_, err := r.Run(context.Background(), command.Request{ID: "workspace.resize", Args: []byte(`{"split":"row","delta":1}`), Origin: command.OriginAgent, WhenContext: free.WhenContext()})
		ae, _ := errors.AsType[*command.ArgError](err)
		if ae == nil {
			t.Fatalf("no ArgError: %v", err)
		}
		return ae.Reason
	}
	if got, want := reason(nil), "has 2 separators on screen; name one of row:0, row:1"; got != want {
		t.Errorf("the default: %q, want %q", got, want)
	}
	if got, want := reason(msgcat.Table{workspace.MsgManySplits: "%[1]d séparations : %[2]s"}), "2 séparations : row:0, row:1"; got != want {
		t.Errorf("reworded: %q, want %q", got, want)
	}
}

// TestCatalogPanesMarks: workspace.panes marks the focused and the hidden
// pane in the catalogue's words.
func TestCatalogPanesMarks(t *testing.T) {
	text := func(cat msgcat.Catalog) string {
		th := theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())
		var opts []workspace.Option
		if cat != nil {
			opts = append(opts, workspace.WithCatalog(cat))
		}
		ws := session(agentRoot(false, layout.FullWidth), th, 120, 30, opts...)
		drain(ws, ws.Toggle("metrics"))
		r := command.NewRegistry()
		if err := r.Register(workspace.Commands(ws)...); err != nil {
			t.Fatal(err)
		}
		res, err := r.Run(context.Background(), command.Request{ID: "workspace.panes", Origin: command.OriginAgent, WhenContext: ws.WhenContext()})
		if err != nil {
			t.Fatal(err)
		}
		return res.Text
	}
	plain := text(nil)
	if !strings.Contains(plain, "  focused") || !strings.Contains(plain, "  hidden") {
		t.Fatalf("the default marks:\n%s", plain)
	}
	if got := text(msgcat.Table{workspace.MsgPaneFocused: "actif"}); got != strings.ReplaceAll(plain, "  focused", "  actif") {
		t.Errorf("focused, reworded:\n%s", got)
	}
	if got := text(msgcat.Table{workspace.MsgPaneHidden: "caché"}); got != strings.ReplaceAll(plain, "  hidden", "  caché") {
		t.Errorf("hidden, reworded:\n%s", got)
	}
}

// TestCatalogHelp: the workspace's commands are named in its help in the
// catalogue's words, each alone.
func TestCatalogHelp(t *testing.T) {
	descs := func(cat msgcat.Catalog) []string {
		th := theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())
		var opts []workspace.Option
		if cat != nil {
			opts = append(opts, workspace.WithCatalog(cat))
		}
		ws := session(agentRoot(false, layout.FullWidth), th, 120, 30, opts...)
		var out []string
		for _, col := range ws.FullHelp() {
			for _, b := range col {
				out = append(out, b.Help().Desc)
			}
		}
		return out
	}
	plain := descs(nil)
	for _, c := range []struct {
		id   msgcat.MessageID
		desc string
	}{
		{workspace.MsgHelpNextPane, "next pane"}, {workspace.MsgHelpPrevPane, "previous pane"}, {workspace.MsgHelpPane, "pane"},
		{workspace.MsgHelpZoom, "zoom"}, {workspace.MsgHelpResize, "resize"}, {workspace.MsgHelpClose, "close"},
	} {
		i := slices.Index(plain, c.desc)
		if i < 0 {
			t.Errorf("%s: the default help has no %q: %q", c.id, c.desc, plain)
			continue
		}
		want := slices.Clone(plain)
		want[i] = "X" + c.desc
		if got := descs(msgcat.Table{c.id: "X" + c.desc}); !slices.Equal(got, want) {
			t.Errorf("%s reworded: %q, want %q", c.id, got, want)
		}
	}
}
