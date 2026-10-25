package palette

import (
	"context"
	"encoding/json/jsontext"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/command/commandtea"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/when"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// drain runs cmd and every command in its batches, in order, on this
// goroutine, and returns their messages.
func drain(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	m := cmd()
	if b, ok := m.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range b {
			out = append(out, drain(c)...)
		}
		return out
	}
	if m == nil {
		return nil
	}
	return []tea.Msg{m}
}

// found is the hits p yields for text under c, best first.
func found(t *testing.T, p Provider, text string, c when.Context) []Hit {
	t.Helper()
	var out []Hit
	if err := p.Search(t.Context(), Search{Text: text, Matcher: fuzzy.NewMatcher(text), WhenContext: c}, func(h Hit) bool {
		out = append(out, h)
		return true
	}); err != nil {
		t.Fatal(err)
	}
	slices.SortStableFunc(out, func(a, b Hit) int {
		switch {
		case better(a.Score, b.Score):
			return -1
		case better(b.Score, a.Score):
			return 1
		}
		return 0
	})
	return out
}

func ids(hs []Hit) []string {
	var out []string
	for _, h := range hs {
		out = append(out, h.ID)
	}
	return out
}

type dropArgs struct {
	Name string `json:"name" help:"the database to drop"`
}

// gated is a registry of four commands, whose gate records what it is
// asked and allows it: file.open, a UI command with a category, a slash
// name and an alias; db.drop, destructive, with an argument; edit.only,
// available only while an editor is focused; key.only, offered on keys
// alone; and say.hi, a prompt.
func gated(t *testing.T) (*command.Registry, *[]*command.Invocation) {
	t.Helper()
	var mu sync.Mutex
	var asked []*command.Invocation
	reg := command.NewRegistry(command.WithGate(command.GateFunc(func(_ context.Context, inv *command.Invocation) (command.Verdict, error) {
		mu.Lock()
		asked = append(asked, inv)
		mu.Unlock()
		return command.AllowOnce, nil
	})))
	ok := func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
		return command.Result{Text: "done"}, nil
	}
	say := command.MustNew("say.hi", "Say hi", func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
		return command.Result{Text: "say hi"}, nil
	}, command.WithDanger(command.ReadOnly))
	say.Kind = command.Prompt
	if err := reg.Register(
		command.MustNew("file.open", "Open file", ok, command.WithDanger(command.UI), command.WithCategory("File"),
			command.WithSlash("open", "o"), command.WithDescription("Open a file by its path")),
		command.MustNew("db.drop", "Drop database", func(context.Context, *command.Invocation, dropArgs) (command.Result, error) {
			return command.Result{Text: "dropped"}, nil
		}, command.WithDanger(command.Destructive)),
		command.MustNew("edit.only", "Format document", ok, command.WithDanger(command.UI), command.WithWhen("editor.focused")),
		command.MustNew("key.only", "Open the key menu", ok, command.WithDanger(command.UI), command.WithSurfaces(command.SurfaceKey)),
		say,
	); err != nil {
		t.Fatal(err)
	}
	return reg, &asked
}

// TestFromRegistry: a command matches by its title, "Category: Title", its
// slash name and its aliases, keeping the best; one whose when fails at
// open, or that is not offered on the palette, is left out; a hit carries
// the command's ID, Danger, description and parameters.
func TestFromRegistry(t *testing.T) {
	reg, _ := gated(t)
	p := FromRegistry(reg)
	none := when.Map{}

	hs := found(t, p, "o", none)
	if len(hs) == 0 || hs[0].ID != "file.open" || hs[0].Score.Grade != fuzzy.Exact {
		t.Fatalf("o: %v, best %+v; want file.open, exact by its alias", ids(hs), hs)
	}
	if hs := found(t, p, "open", none); len(hs) == 0 || hs[0].ID != "file.open" || hs[0].Score.Grade != fuzzy.Exact {
		t.Fatalf("open: %v; want file.open, exact by its slash name", ids(hs))
	}
	if hs := found(t, p, "file: open", none); len(hs) != 1 || hs[0].Text != "File: Open file" || hs[0].Help != "Open a file by its path" {
		t.Fatalf("file: open: %+v; want file.open drawn as File: Open file", hs)
	}
	for _, q := range []string{"format", "key menu"} {
		if hs := found(t, p, q, none); len(hs) != 0 {
			t.Errorf("%s: %v; want none", q, ids(hs))
		}
	}
	if hs := found(t, p, "format", when.Map{"editor.focused": when.BoolValue(true)}); len(ids(hs)) != 1 {
		t.Errorf("format with an editor focused: %v; want edit.only", ids(hs))
	}
	hs = found(t, p, "drop", none)
	if len(hs) != 1 || hs[0].Command != "db.drop" || hs[0].Danger != command.Destructive ||
		len(hs[0].Params) != 1 || hs[0].Params[0].Name != "name" {
		t.Errorf("drop: %+v; want db.drop with its Danger and its one parameter", hs)
	}
}

// TestRegistryRun: running a command hit dispatches it through the
// registry with OriginPalette and the when values it was found under, so a
// ResultMsg follows, a prompt's PromptMsg reaches the program, and the
// registry's policy decides a destructive command: from the palette it runs
// unasked, while the same command from the agent is put to the gate
// (0006-MADR A4; 0008-PLAN Step 7 deviation D2).
func TestRegistryRun(t *testing.T) {
	reg, asked := gated(t)
	p := FromRegistry(reg)
	at := when.Map{"x": when.BoolValue(true)}

	run := func(q string) []tea.Msg {
		t.Helper()
		hs := found(t, p, q, at)
		if len(hs) == 0 {
			t.Fatalf("%s: no hit", q)
		}
		var args jsontext.Value
		if len(hs[0].Params) > 0 {
			args = jsontext.Value(`{"name":"x"}`) // what Step 8's argument stage collects
		}
		return drain(hs[0].Run(args))
	}
	res := filter[command.ResultMsg](run("open"))
	if len(res) != 1 || res[0].Err != nil || res[0].Request.Origin != command.OriginPalette || res[0].Request.WhenContext == nil {
		t.Fatalf("open: %+v; want one ResultMsg from the palette", res)
	}
	if v, ok := res[0].Request.WhenContext.Value("x"); !ok || v.Kind() != when.KindBool || !v.Bool() {
		t.Errorf("the request's when values are not those it was found under")
	}
	if pm := filter[command.PromptMsg](run("say")); len(pm) != 1 || pm[0].Text != "say hi" {
		t.Errorf("say: %+v; want its PromptMsg", pm)
	}
	n := len(*asked)
	res = filter[command.ResultMsg](run("drop"))
	if len(res) != 1 || res[0].Err != nil || res[0].Result.Text != "dropped" || res[0].Request.Origin != command.OriginPalette {
		t.Fatalf("drop from the palette: %+v; want it run by the registry", res)
	}
	if len(*asked) != n {
		t.Fatalf("drop from the palette asked the gate %d times; the policy asks only the agent's and the shell's", len(*asked)-n)
	}
	agent := filter[command.ResultMsg](drain(commandtea.Dispatch(t.Context(), reg, command.Request{
		ID: "db.drop", Origin: command.OriginAgent, Args: jsontext.Value(`{"name":"x"}`),
	})))
	if len(*asked) != n+1 || (*asked)[n].Command.ID != "db.drop" || len(agent) != 1 {
		t.Errorf("drop from the agent: the gate was asked %d times, results %+v; want once", len(*asked)-n, agent)
	}
}

// TestRegistryDiscover: through a picker, ">" alone lists the commands
// available on the palette where it opened: the registry's own built-in
// commands and gated's, less the one whose when fails and the one offered
// on keys alone.
func TestRegistryDiscover(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		reg, _ := gated(t)
		g.register(FromRegistry(reg))
		g.open()
		g.typed(">")
		var got []string
		for _, r := range g.p.eng.rows() {
			got = append(got, r.hit.ID)
		}
		var want []string
		for c := range reg.Available(when.Map{}, command.SurfacePalette) {
			want = append(want, string(c.ID))
		}
		if !slices.Equal(got, want) || !slices.Contains(got, "db.drop") || slices.Contains(got, "edit.only") || slices.Contains(got, "key.only") {
			t.Errorf("> lists %q; want %q", got, want)
		}
	})
}

// TestStatic: an empty query lists the items in their order; a typed one
// matches their text; accepting one runs its Run.
func TestStatic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		var ran []string
		item := func(id, text string) Item {
			return Item{ID: id, Text: text, Run: func() tea.Cmd { ran = append(ran, id); return nil }}
		}
		g.register(Static("themes", []Item{item("dark", "Dark theme"), item("light", "Light theme"), item("hc", "High contrast")}))
		g.open()
		if got := g.texts(); !slices.Equal(got, []string{"Dark theme", "Light theme", "High contrast"}) {
			t.Fatalf("an empty query lists %q", got)
		}
		g.typed("light")
		if got := g.texts(); len(got) == 0 || got[0] != "Light theme" {
			t.Fatalf("light: %q", got)
		}
		g.key("enter")
		if !slices.Equal(ran, []string{"light"}) {
			t.Errorf("enter ran %q; want light", ran)
		}
	})
}

// titled is a pane with a title.
type titled struct {
	pane
	title string
}

func (p *titled) Title() string { return p.title }
func (p *titled) Update(m tea.Msg) (workspace.Pane, tea.Cmd) {
	p.msgs = append(p.msgs, m)
	return p, nil
}

// TestPanes: the panes provider lists the focus ring's panes by title,
// then the hidden; accepting one focuses it, and accepting a hidden one
// shows it and focuses it.
func TestPanes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		panes := map[layout.PaneID]workspace.Pane{
			"main": &titled{title: "Session"}, "side": &titled{title: "Metrics"}, "logs": &titled{title: "Logs"},
		}
		root := layout.SidebarRightBottom("main", "side", "logs", layout.WithGap(0))
		g := newProgOn(t, root, panes, keys(t), nil)
		g.run(g.w.Toggle("logs"))
		g.settle()
		g.register(Panes(g.w))
		g.open()
		if got := g.texts(); !slices.Equal(got, []string{"Session", "Metrics", "Logs"}) {
			t.Fatalf("the panes are %q", got)
		}
		g.typed("metrics")
		g.key("enter")
		if f := g.w.Focused(); f != "side" || len(g.w.Overlays()) != 0 {
			t.Fatalf("accepting Metrics: focus %q, overlays %v; want side", f, g.w.Overlays())
		}
		g.open()
		g.typed("logs")
		g.key("enter")
		if f, hidden := g.w.Focused(), g.w.Plan().Hidden; f != "logs" || slices.Contains(hidden, "logs") {
			t.Errorf("accepting the hidden Logs: focus %q, hidden %v; want logs, shown", f, hidden)
		}
	})
}

// tree is a file system for the file provider.
var tree = fstest.MapFS{
	"README.md":                  {},
	"main.go":                    {},
	"cmd/main.go":                {},
	"internal/main/run.go":       {},
	"docs/maintenance.md":        {},
	"docs/a/b/domain.txt":        {},
	"vendor/x/main.go":           {},
	"node_modules/pkg/index.js":  {},
	"internal/main/main_test.go": {},
}

// skipVendored skips vendor and node_modules.
var skipVendored = PathFilterFunc(func(p string, d fs.DirEntry) bool {
	return d.IsDir() && (p == "vendor" || p == "node_modules")
})

// started is the files provider after its Start.
func startedFiles(t *testing.T, opts ...FileOption) *files {
	t.Helper()
	f := Files(FSWalker(tree, skipVendored), opts...).(*files)
	if err := f.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	return f
}

// TestFiles: FSWalker honours its PathFilter; the paths are ranked by
// crush's tiers on the last element; a walk above WithIndexAbove is
// searched through an index, and one below in full, with the same best
// hits; WithMaxFiles stops the walk and adds its muted row.
func TestFiles(t *testing.T) {
	f := startedFiles(t)
	if w := f.at.Load(); slices.ContainsFunc(w.paths, func(p string) bool {
		return strings.HasPrefix(p, "vendor/") || strings.HasPrefix(p, "node_modules/")
	}) || len(w.paths) != 7 {
		t.Fatalf("walked %q; want the seven files outside vendor and node_modules", w.paths)
	}
	hs := found(t, f, "main", nil)
	grades := map[string]fuzzy.Grade{}
	for _, h := range hs {
		grades[h.Text] = h.Score.Grade
	}
	want := map[string]fuzzy.Grade{
		"main.go": fuzzy.Exact, "cmd/main.go": fuzzy.Exact, "internal/main/main_test.go": fuzzy.Prefix,
		"docs/maintenance.md": fuzzy.Prefix, "internal/main/run.go": fuzzy.Boundary, "docs/a/b/domain.txt": fuzzy.Subsequence,
	}
	for p, g := range want {
		if got, ok := grades[p]; !ok || got != g {
			t.Errorf("main: %s graded %v, %v; want %v", p, got, ok, g)
		}
	}
	if got := ids(hs)[:2]; !slices.Contains(got, "file:main.go") || !slices.Contains(got, "file:cmd/main.go") {
		t.Errorf("main: the best two are %q; want the two main.go", got)
	}
	if hs[0].Run != nil {
		t.Error("a path hit has a Run")
	}

	small, big := startedFiles(t, WithIndexAbove(100)), startedFiles(t, WithIndexAbove(3))
	if small.at.Load().ix != nil || big.at.Load().ix == nil {
		t.Fatalf("an index for 7 paths above 100: %v; above 3: %v", small.at.Load().ix != nil, big.at.Load().ix != nil)
	}
	if a, b := ids(found(t, small, "main.go", nil)), ids(found(t, big, "main.go", nil)); len(a) < 2 || !slices.Equal(a[:2], b[:2]) {
		t.Errorf("main.go: in full %q, through the index %q", a, b)
	}

	capped := startedFiles(t, WithMaxFiles(2))
	if w := capped.at.Load(); len(w.paths) != 2 || !w.stopped {
		t.Fatalf("WithMaxFiles(2) walked %d, stopped %v", len(w.paths), w.stopped)
	}
	hs = found(t, capped, "zzz", nil)
	if len(hs) != 1 || hs[0].Disabled == "" || !strings.Contains(hs[0].Text, "2 files") {
		t.Errorf("past the limit: %+v; want the muted limit row alone", hs)
	}
}

// TestFilesInPicker: through a picker, "@" restricts the query to the
// files, which search once their debounce ends, after the walk.
func TestFilesInPicker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		g.register(Files(FSWalker(tree, skipVendored)))
		g.register(Static("other", []Item{{ID: "m", Text: "main menu"}}))
		g.open()
		g.typed("@main")
		if got := g.texts(); len(got) != 0 {
			t.Fatalf("before the debounce ends: rows %q", got)
		}
		time.Sleep(filesDebounce)
		g.settle()
		got := g.texts()
		if len(got) < 2 || !slices.Contains(got[:2], "main.go") || !slices.Contains(got[:2], "cmd/main.go") || slices.Contains(got, "main menu") {
			t.Errorf("@main: rows %q; want the files alone, the two main.go first", got)
		}
	})
}

// editor is a focused pane that records its keys and binds enter itself.
type editor struct{ pane }

func (e *editor) Update(m tea.Msg) (workspace.Pane, tea.Cmd) {
	e.msgs = append(e.msgs, m)
	return e, nil
}

// TestCompletion: a picker WithoutInput opens non-modal, as "completion",
// so other keys reach the focused editor; SetQuery searches; down moves and
// tab or enter accepts, in overlay:completion, sending SelectedMsg, running
// nothing and closing it; the editor's own enter is shadowed only while it
// is open; esc closes it.
func TestCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ed := &editor{}
		km := keys(t, []keymap.Binding{user(t, "pane:main", "enter", "editor.submit", false)})
		g := newProgOn(t, layout.Pane{ID: "main"}, map[layout.PaneID]workspace.Pane{"main": ed}, km, nil, WithoutInput())
		var r ran
		var items []Item
		for _, h := range runs(&r, 4) {
			run := h.Run
			items = append(items, Item{ID: h.ID, Text: h.Text, Run: func() tea.Cmd { return run(nil) }})
		}
		g.register(Static("paths", items))
		g.open()
		if o := g.w.Overlays(); !slices.Equal(o, []string{"completion"}) {
			t.Fatalf("overlays %v; want completion", o)
		}
		if modal, _ := workspace.ModalKey().Get(g.w.WhenContext()); modal {
			t.Fatal("the completion pop-up is modal")
		}
		if first, _, _ := strings.Cut(g.view(), "\n"); !strings.Contains(first, "a1") {
			t.Fatalf("the pop-up's first line is %q; want its first row, with no input", first)
		}
		g.key("x")
		if keys := filter[tea.KeyPressMsg](ed.msgs); len(keys) != 1 || keys[0].Text != "x" {
			t.Fatalf("the editor got %v; want the x", keys)
		}
		g.run(g.w.SendOverlay("completion", SetQuery("c")))
		g.settle()
		if got := g.texts(); len(got) == 0 || got[0] != "c1" {
			t.Fatalf("SetQuery(c): rows %q", got)
		}
		g.run(g.w.SendOverlay("completion", SetQuery("")))
		g.settle()
		g.key("down")
		if g.p.sel != 1 {
			t.Fatalf("down: selection %d; want 1", g.p.sel)
		}
		g.key("enter")
		sel := filter[SelectedMsg](g.seen)
		if len(sel) != 1 || sel[0].ID != "completion" || sel[0].Hit.ID != "b1" {
			t.Fatalf("enter sent %+v; want SelectedMsg for b1 from completion", sel)
		}
		if got := r.list(); len(got) != 0 || len(g.w.Overlays()) != 0 || len(g.requests()) != 0 {
			t.Fatalf("enter: ran %q, overlays %v, requests %v; want nothing run, closed, the editor's enter shadowed", got, g.w.Overlays(), g.requests())
		}
		g.key("enter")
		if rs := g.requests(); len(rs) != 1 || rs[0].Request.ID != "editor.submit" {
			t.Fatalf("enter with the pop-up closed: %v; want editor.submit", rs)
		}
		g.open()
		g.key("tab")
		if sel := filter[SelectedMsg](g.seen); len(sel) != 2 || sel[1].Hit.ID != "a1" {
			t.Fatalf("tab sent %+v; want a second SelectedMsg, for a1", sel)
		}
		g.open()
		g.key("esc")
		if o := g.w.Overlays(); len(o) != 0 {
			t.Errorf("esc left %v", o)
		}
	})
}

// TestSelectedMsg: a picker's accept sends SelectedMsg as well as running
// the hit.
func TestSelectedMsg(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		var r ran
		g.register(newDiscovering("items", runs(&r, 2)...))
		g.open()
		g.key("enter")
		if sel := filter[SelectedMsg](g.seen); len(sel) != 1 || sel[0].ID != "palette" || sel[0].Hit.ID != "a1" || !slices.Equal(r.list(), []string{"a1"}) {
			t.Errorf("enter: SelectedMsg %+v, ran %q; want both, for a1", sel, r.list())
		}
	})
}
