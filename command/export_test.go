package command

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/tuitest"
	"github.com/maccavelli/go-tui-lib/when"
)

// The field names of MCP 2026-07-28's schema.ts (fetched 2026-10-05,
// SHA-256 742750af0bb8c716e7030c4977c992b55d1adc4407e9e66997db5846baedc2cd),
// held here so no test reads the network.
var (
	mcpToolFields        = []string{"name", "title", "icons", "description", "inputSchema", "outputSchema", "annotations", "_meta"}
	mcpToolRequired      = []string{"name", "inputSchema"}
	mcpAnnotationFields  = []string{"title", "readOnlyHint", "destructiveHint", "idempotentHint", "openWorldHint"}
	mcpCallResultFields  = []string{"_meta", "resultType", "content", "structuredContent", "isError"}
	mcpCallResultNeeds   = []string{"resultType", "content"}
	mcpTextContentFields = []string{"type", "text", "annotations", "_meta"}
	mcpTextContentNeeds  = []string{"type", "text"}
)

type zoomArgs struct {
	Pane string `json:"pane,omitzero" arg:"" optional:"" help:"the pane; default the focused one"`
}

type deleteArgs struct {
	Path string `json:"path" arg:"" help:"the file to delete" schema:"minLen=1"`
}

// catalogue is a registry with one command of each kind and danger, a
// hidden one, one not offered to agents and one whose When is false.
func catalogue(t testing.TB, o ...RegistryOption) *Registry {
	t.Helper()
	readDoc := mustNew(t, "doc.read", func(context.Context, *Invocation, NoArgs) (Result, error) {
		return Result{Value: map[string]any{"lines": 3}}, nil
	}, WithDanger(ReadOnly), WithDescription("Reads the document."), WithOutput(Schema(`{"type":"object","properties":{"lines":{"type":"integer"}}}`)))
	zoom := mustNew(t, "view.zoom", func(_ context.Context, _ *Invocation, a zoomArgs) (Result, error) {
		return Result{Text: "zoomed " + a.Pane}, nil
	}, WithDanger(UI), WithIdempotent(), WithSlash("zoom"), WithArgHint("pane"), WithDescription("Zooms a pane."), WithMeta("icon", "zoom"))
	reset := mustNew(t, "session.reset", func(context.Context, *Invocation, NoArgs) (Result, error) {
		return Result{Text: "reset"}, nil
	}, WithDanger(Mutating), WithSlash("reset"), WithDescription("Resets the session."))
	del := mustNew(t, "files.delete", func(_ context.Context, _ *Invocation, a deleteArgs) (Result, error) {
		return Result{}, errors.New("no such file: " + a.Path)
	}, WithDanger(Destructive), WithOpenWorld(), WithDescription("Deletes a file."))
	r := registryOf(t, o, readDoc, zoom, reset, del,
		cmd("secret.tool", UI, func(c *Command) { c.Hidden, c.Slash = true, "secret" }),
		cmd("shell.only", UI, func(c *Command) { c.Surfaces, c.Slash = SurfaceCLI, "shell" }),
		cmd("not.now", UI, func(c *Command) { c.When, c.Slash = "busy", "later" }),
	)
	cmds, errs := LoadDir(fstestFS("review.md", "---\ndescription: Review the diff\ndanger: read-only\n---\nReview, focusing on $FOCUS."), userSrc)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	r.ReplaceSource(userSrc, cmds)
	r.ReplaceSource(Source{Kind: ACP, Name: "agent"}, FromACP("agent", []ACPCommand{{Name: "plan", Description: "Plan first", Input: &ACPCommandInput{Hint: "goal"}}}))
	return r
}

// fstestFS is a file system holding one file.
func fstestFS(name, body string) fstest.MapFS {
	return fstest.MapFS{name: {Data: []byte(body)}}
}

// marshal is v as JSON, failing t on an error.
func marshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// indentJSON is v as indented JSON, with a final newline.
func indentJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		t.Fatal(err)
	}
	j := jsontext.Value(b)
	if err := j.Indent(jsontext.WithIndent("  ")); err != nil {
		t.Fatal(err)
	}
	return string(j) + "\n"
}

func TestExportGolden(t *testing.T) {
	r := catalogue(t, WithGate(answer(AllowOnce, nil)))
	tuitest.Text(t, "mcp-tools", indentJSON(t, r.MCPTools(nil)))
	tuitest.Text(t, "acp-commands", indentJSON(t, r.ACPCommands(nil)))
	tuitest.Text(t, "manifest", indentJSON(t, r.Manifest()))
	for name, call := range map[string][2]string{
		"text":    {"view.zoom", `{"pane":"logs"}`},
		"value":   {"doc.read", ""},
		"failed":  {"files.delete", `{"path":"x"}`},
		"badargs": {"view.zoom", `{"pane":1}`},
		"unknown": {"no.such", "{}"},
		"prompt":  {"user.review", `{"FOCUS":"tests"}`},
	} {
		tuitest.Text(t, "mcp-call-"+name, indentJSON(t, r.CallMCP(t.Context(), call[0], []byte(call[1]), "agent-1")))
	}
	refused := catalogue(t)
	tuitest.Text(t, "mcp-call-refused", indentJSON(t, refused.CallMCP(t.Context(), "session.reset", nil, "agent-1")))
}

// keysOf is the keys of a JSON object.
func keysOf(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func checkFields(t *testing.T, what string, m map[string]any, allowed, required []string) {
	t.Helper()
	for k := range m {
		if !slices.Contains(allowed, k) {
			t.Errorf("%s has %q, which MCP 2026-07-28 does not name", what, k)
		}
	}
	for _, k := range required {
		if _, ok := m[k]; !ok {
			t.Errorf("%s lacks the required %q", what, k)
		}
	}
}

func TestMCPFieldNames(t *testing.T) {
	r := catalogue(t, WithGate(answer(AllowOnce, nil)))
	for _, tool := range r.MCPTools(nil) {
		m := keysOf(t, marshal(t, tool))
		checkFields(t, "tool "+tool.Name, m, mcpToolFields, mcpToolRequired)
		ann, _ := m["annotations"].(map[string]any)
		checkFields(t, "annotations of "+tool.Name, ann, mcpAnnotationFields, nil)
		if in, _ := m["inputSchema"].(map[string]any); in["type"] != "object" {
			t.Errorf("tool %s: inputSchema type %v, want object", tool.Name, in["type"])
		}
	}
	for _, call := range [][2]string{{"view.zoom", "{}"}, {"doc.read", ""}, {"files.delete", `{"path":"x"}`}, {"no.such", ""}} {
		m := keysOf(t, marshal(t, r.CallMCP(t.Context(), call[0], []byte(call[1]), "")))
		checkFields(t, "the result of "+call[0], m, mcpCallResultFields, mcpCallResultNeeds)
		if m["resultType"] != "complete" {
			t.Errorf("%s: resultType %v", call[0], m["resultType"])
		}
		content, _ := m["content"].([]any)
		for _, c := range content {
			block, _ := c.(map[string]any)
			checkFields(t, "a content block of "+call[0], block, mcpTextContentFields, mcpTextContentNeeds)
		}
	}
}

func TestACPFieldNames(t *testing.T) {
	b, err := json.Marshal(catalogue(t).ACPCommands(nil))
	if err != nil {
		t.Fatal(err)
	}
	var cmds []map[string]any
	if err := json.Unmarshal(b, &cmds); err != nil {
		t.Fatal(err)
	}
	if len(cmds) == 0 {
		t.Fatal("no commands")
	}
	for _, c := range cmds {
		for k := range c {
			if k != "name" && k != "description" && k != "input" {
				t.Errorf("an ACP command has %q", k)
			}
		}
		if in, ok := c["input"].(map[string]any); ok {
			if len(in) != 1 || in["hint"] == nil {
				t.Errorf("an ACP input is %v, want only hint", in)
			}
		}
	}
}

func TestCallMCPErrors(t *testing.T) {
	r := catalogue(t)
	for call, want := range map[[2]string]string{
		{"view.zoom", `{"pane":1}`}:     "input schema",
		{"view.zoom", `{"bogus":1}`}:    "input schema",
		{"no.such", "{}"}:               "no tool named no.such",
		{"session.reset", "{}"}:         "refused",
		{"not.now", "{}"}:               "not available",
		{"files.delete", `{"path":""}`}: "input schema",
	} {
		res := r.CallMCP(t.Context(), call[0], []byte(call[1]), "agent-1")
		if !res.IsError || len(res.Content) != 1 || !strings.Contains(res.Content[0].Text, want) {
			t.Errorf("%v: %+v, want an error saying %q", call, res, want)
		}
	}
}

func TestCallMCPIsAgent(t *testing.T) {
	var seen []Origin
	var callers []string
	g := &gateFunc{next: func(inv *Invocation) (Decision, error) {
		seen, callers = append(seen, inv.Origin), append(callers, inv.Caller)
		return AllowOnce, nil
	}}
	r := catalogue(t, WithGate(g))
	if res := r.CallMCP(t.Context(), "session.reset", nil, "agent-1"); res.IsError {
		t.Fatalf("with a gate that allows: %+v", res)
	}
	if !slices.Equal(seen, []Origin{OriginAgent}) || !slices.Equal(callers, []string{"agent-1"}) {
		t.Errorf("the gate saw origins %v and callers %v", seen, callers)
	}
	if res := catalogue(t).CallMCP(t.Context(), "session.reset", nil, "agent-1"); !res.IsError {
		t.Error("a mutating tool ran for an agent with no gate")
	}
	if res := catalogue(t).CallMCP(t.Context(), "view.zoom", nil, "agent-1"); res.IsError {
		t.Errorf("a UI tool was refused for an agent: %+v", res)
	}
}

func TestExportFilters(t *testing.T) {
	r := catalogue(t)
	var tools []string
	for _, tool := range r.MCPTools(nil) {
		tools = append(tools, tool.Name)
	}
	for _, out := range []string{"secret.tool", "shell.only", "not.now"} {
		if slices.Contains(tools, out) {
			t.Errorf("MCPTools exports %s", out)
		}
	}
	if !slices.Contains(tools, "doc.read") || !slices.IsSorted(tools) {
		t.Errorf("MCPTools = %v", tools)
	}
	if busy := r.MCPTools(when.Map{"busy": when.BoolValue(true)}); !slices.ContainsFunc(busy, func(m MCPTool) bool { return m.Name == "not.now" }) {
		t.Error("a command whose When holds was not exported")
	}
	var names []string
	for _, c := range r.ACPCommands(nil) {
		names = append(names, c.Name)
	}
	// acp.agent.plan, session.reset, user.review, view.zoom.
	if want := []string{"plan", "reset", "review", "zoom"}; !slices.Equal(names, want) {
		t.Errorf("ACPCommands = %v, want %v (slash commands for agents, by ID)", names, want)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	m := catalogue(t).Manifest()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var back Manifest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, back) {
		t.Errorf("the manifest changed through JSON:\n%+v\n%+v", m, back)
	}
	ids := make([]ID, len(m.Commands))
	for i, c := range m.Commands {
		ids[i] = c.ID
	}
	if m.Format != 1 || !slices.IsSorted(ids) || !slices.Contains(ids, "secret.tool") {
		t.Errorf("format %d, IDs %v; want format 1, sorted, hidden ones too", m.Format, ids)
	}
}

// within receives from ch, failing t after five seconds.
func within[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal(what)
		var zero T
		return zero
	}
}

func TestRunOnLoop(t *testing.T) {
	type effectMsg struct{}
	var ranOn []string
	loopCmd := func(c *Command) {
		c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
			ranOn = append(ranOn, "handler")
			return Result{Text: "done", Cmd: func() tea.Msg { return effectMsg{} }}, nil
		})
	}
	sent := make(chan tea.Msg, 1)
	r := registryOf(t, []RegistryOption{WithLoop(func(m tea.Msg) { sent <- m })}, cmd("loop", UI, loopCmd),
		cmd("async", UI, func(c *Command) { c.Mode = Async; loopCmd(c) }))

	type ran struct {
		res Result
		err error
	}
	got := make(chan ran, 1)
	go func() {
		res, err := r.Run(t.Context(), Request{ID: "loop", Origin: OriginAgent})
		got <- ran{res, err}
	}()
	msg := within(t, sent, "Run sent no LoopMsg")
	lm, ok := msg.(LoopMsg)
	if !ok {
		t.Fatalf("Run sent %T, want LoopMsg", msg)
	}
	select {
	case <-got:
		t.Fatal("Run returned before the loop ran the command")
	default:
	}
	if len(ranOn) != 0 {
		t.Fatal("the handler ran before the loop ran the LoopMsg")
	}
	effect := lm.Run()
	out := within(t, got, "Run did not return after the loop ran it")
	if out.err != nil || out.res.Text != "done" || out.res.Cmd != nil {
		t.Errorf("Run = %+v, %v; want the result without its Cmd", out.res, out.err)
	}
	if effect == nil || effect() != (effectMsg{}) {
		t.Error("the LoopMsg did not give the program the command's effect")
	}

	// An Async command still runs on Run's caller.
	if res, err := r.Run(t.Context(), Request{ID: "async", Origin: OriginAgent}); err != nil || res.Cmd == nil {
		t.Errorf("an Async command: %+v, %v", res, err)
	}
	select {
	case m := <-sent:
		t.Errorf("an Async command sent %T", m)
	default:
	}

	// A caller that gives up: Run returns, and the LoopMsg then runs nothing.
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		_, err := r.Run(ctx, Request{ID: "loop", Origin: OriginAgent})
		got <- ran{err: err}
	}()
	late, _ := within(t, sent, "Run sent no LoopMsg").(LoopMsg)
	cancel()
	if out := within(t, got, "a cancelled Run did not return"); !errors.Is(out.err, context.Canceled) {
		t.Errorf("a cancelled Run: %v", out.err)
	}
	before := len(ranOn)
	if late.Run() != nil || len(ranOn) != before {
		t.Error("a LoopMsg ran after its caller gave up")
	}

	// Without WithLoop, Run runs the command on its caller.
	plain := registryOf(t, nil, cmd("loop", UI, loopCmd))
	if res, err := plain.Run(t.Context(), Request{ID: "loop", Origin: OriginAgent}); err != nil || res.Cmd == nil {
		t.Errorf("without WithLoop: %+v, %v", res, err)
	}
	var zero LoopMsg
	if zero.Run() != nil {
		t.Error("the zero LoopMsg did something")
	}
}
