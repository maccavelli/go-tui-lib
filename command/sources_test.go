package command

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"slices"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// loaded is a loaded prompt command in src with id and slash name.
func loaded(id ID, slash string, aliases ...string) Command {
	return Command{
		ID: id, Title: string(id), Slash: slash, Aliases: aliases, Kind: Prompt, Danger: Mutating,
		Handler: HandlerFunc(func(context.Context, *Invocation) (Result, error) { return Result{Text: string(id)}, nil }),
	}
}

// watchMsgs runs w and returns the messages it gives, flattened.
func watchMsgs(t *testing.T, w tea.Cmd) []tea.Msg {
	t.Helper()
	ch := make(chan []tea.Msg, 1)
	go func() { ch <- collect(w) }()
	select {
	case m := <-ch:
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("Watch did not return")
		return nil
	}
}

func conflictsIn(msgs []tea.Msg) []Conflict {
	for _, m := range msgs {
		if c, ok := m.(ConflictMsg); ok {
			return c.Conflicts
		}
	}
	return nil
}

func TestSlashClash(t *testing.T) {
	r := registryOf(t, nil, cmd("builtin.review", UI, func(c *Command) { c.Slash = "review" }))
	w := r.Watch()
	got := r.ReplaceSource(userSrc, []Command{loaded("user.review", "review", "rv")})
	want := Conflict{ID: "user.review", Slash: "review", Renamed: "user:review", Holder: "builtin.review"}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("conflicts %+v, want [%+v]", got, want)
	}
	msgs := watchMsgs(t, w)
	if c := conflictsIn(msgs); len(c) != 1 || c[0] != want {
		t.Errorf("Watch sent %v, want a ConflictMsg with %+v", msgs, want)
	}
	if !slices.Contains(msgs, tea.Msg(ChangedMsg{Version: 2})) {
		t.Errorf("Watch sent %v, want ChangedMsg{2} too", msgs)
	}
	if c, ok := r.Slash("user:review"); !ok || c.ID != "user.review" {
		t.Error("/user:review does not reach the loaded command")
	}
	if c, _ := r.Slash("review"); c.ID != "builtin.review" {
		t.Error("the built-in lost /review")
	}
	if c, _ := r.Slash("rv"); c.ID != "user.review" {
		t.Error("the free alias was not kept")
	}

	// D19: a built-in registered later takes the name.
	w = r.Watch()
	if got := r.ReplaceSource(Source{Kind: Project}, []Command{loaded("project.deploy", "deploy")}); len(got) != 0 {
		t.Fatalf("conflicts %+v", got)
	}
	_ = watchMsgs(t, w)
	w = r.Watch()
	if err := r.Register(cmd("builtin.deploy", UI, func(c *Command) { c.Slash = "deploy" })); err != nil {
		t.Fatal(err)
	}
	late := Conflict{ID: "project.deploy", Slash: "deploy", Renamed: "project:deploy", Holder: "builtin.deploy"}
	if c := conflictsIn(watchMsgs(t, w)); len(c) != 1 || c[0] != late {
		t.Errorf("a late built-in: %+v, want [%+v]", c, late)
	}
	if c, _ := r.Slash("deploy"); c.ID != "builtin.deploy" {
		t.Error("the late built-in did not take /deploy")
	}
	if c, _ := r.Slash("project:deploy"); c.ID != "project.deploy" {
		t.Error("the displaced command is not at /project:deploy")
	}
	if err := r.Register(cmd("builtin.other", UI, func(c *Command) { c.Slash = "deploy" })); err == nil {
		t.Error("a built-in took another built-in's slash name")
	}

	// D22: when the renamed name is taken too, the name is dropped.
	if err := r.Register(cmd("builtin.x", UI, func(c *Command) { c.Slash, c.Aliases = "x", []string{"user:x"} })); err != nil {
		t.Fatal(err)
	}
	got = r.ReplaceSource(userSrc, []Command{loaded("user.x", "x")})
	if len(got) != 1 || got[0].Renamed != "" || got[0].Err != nil {
		t.Errorf("a double clash: %+v, want the name dropped", got)
	}
	if c, ok := r.Lookup("user.x"); !ok || c.Slash != "" {
		t.Errorf("user.x: %v, slash %q; want it loaded with no slash name", ok, c.Slash)
	}

	// WithPrefixer replaces the prefix.
	p := registryOf(t, []RegistryOption{WithPrefixer(func(s Source) string { return "u" })},
		cmd("builtin.review", UI, func(c *Command) { c.Slash = "review" }))
	if got := p.ReplaceSource(userSrc, []Command{loaded("user.review", "review")}); len(got) != 1 || got[0].Renamed != "u:review" {
		t.Errorf("WithPrefixer: %+v", got)
	}
}

func TestReplaceSource(t *testing.T) {
	r := NewRegistry()
	r.ReplaceSource(Source{Kind: Project}, []Command{loaded("project.keep", "keep")})
	r.ReplaceSource(userSrc, []Command{loaded("user.a", "a"), loaded("user.b", "b")})
	v := r.Version()
	w := r.Watch()
	if got := r.ReplaceSource(userSrc, []Command{loaded("user.b", "b"), loaded("user.c", "c")}); len(got) != 0 {
		t.Fatalf("conflicts %+v", got)
	}
	if msgs := watchMsgs(t, w); len(msgs) != 1 || msgs[0] != (ChangedMsg{Version: v + 1}) {
		t.Errorf("Watch: %v, want one ChangedMsg{%d}", msgs, v+1)
	}
	if got := ids(r.All()); !slices.Equal(got, []ID{"project.keep", "user.b", "user.c"}) {
		t.Errorf("after the swap: %v", got)
	}
	if _, ok := r.Slash("a"); ok {
		t.Error("/a outlived its command")
	}
	r.ReplaceSource(userSrc, nil)
	if got := ids(r.All()); !slices.Equal(got, []ID{"project.keep"}) {
		t.Errorf("after an empty swap: %v", got)
	}

	// Commands it cannot load are Conflicts with Err, and the rest load.
	bad := r.ReplaceSource(userSrc, []Command{
		loaded("project.sneak", "s1"), // outside the namespace
		loaded("user.ok", "ok"),       // fine
		loaded("user.ok", "ok2"),      // the same ID twice
		loaded("user.Bad", "b2"),      // malformed
		dangerless(loaded("user.nodanger", "nd")),
		loaded("user.slash", "has one"), // a bad slash name
	})
	bad = slices.DeleteFunc(bad, func(c Conflict) bool { return c.Err == nil })
	if len(bad) != 5 {
		t.Errorf("%d refusals, want 5: %+v", len(bad), bad)
	}
	if got := ids(r.All()); !slices.Contains(got, "user.ok") || slices.Contains(got, "project.sneak") {
		t.Errorf("after the refusals: %v", got)
	}
	for _, src := range []Source{{Kind: Builtin}, {Kind: MCP}} {
		if got := r.ReplaceSource(src, []Command{loaded("mcp.x.y", "y")}); len(got) != 1 || got[0].Err == nil {
			t.Errorf("%s: %+v, want a refusal", src, got)
		}
	}
	// A late source's clash with an earlier loaded one renames the later.
	got := r.ReplaceSource(Source{Kind: Plugin, Name: "p"}, []Command{loaded("plugin.p.ok", "ok")})
	if len(got) != 1 || got[0].Renamed != "plugin:p:ok" || got[0].Holder != "user.ok" {
		t.Errorf("loaded beside loaded: %+v", got)
	}
}

// dangerless is c with no danger declared.
func dangerless(c Command) Command {
	c.Danger = 0
	return c
}

func TestFromMCPPrompts(t *testing.T) {
	var asked []string
	get := func(_ context.Context, name string, args map[string]string) (string, error) {
		asked = append(asked, name+" "+args["repo"]+" "+args["label"])
		return "expanded " + name, nil
	}
	cmds := FromMCPPrompts("GitHub", []MCPPrompt{
		{Name: "github_issue", Description: "File an issue", Arguments: []MCPPromptArgument{
			{Name: "repo", Description: "the repository", Required: true}, {Name: "label"},
		}},
		{Name: "summarise", Title: "Summarise"},
		{Name: "github-issue"}, // maps to the same segment
		{Name: "___"},          // maps to nothing
	}, get)
	if len(cmds) != 4 || cmds[0].ID != "mcp.github.github-issue" || cmds[0].Slash != "github-issue" || cmds[0].Title != "github_issue" ||
		cmds[0].Kind != Prompt || cmds[0].Danger != Mutating || cmds[1].Title != "Summarise" {
		t.Fatalf("commands: %+v", cmds)
	}
	var schema struct {
		Properties map[string]map[string]any `json:"properties"`
		Required   []string                  `json:"required"`
	}
	if err := json.Unmarshal(cmds[0].Args, &schema); err != nil || !slices.Equal(schema.Required, []string{"repo"}) ||
		schema.Properties["repo"]["description"] != "the repository" || schema.Properties["label"] == nil {
		t.Errorf("schema %s: %v", cmds[0].Args, err)
	}
	r := NewRegistry()
	conflicts := r.ReplaceSource(Source{Kind: MCP, Name: "GitHub"}, cmds)
	if len(conflicts) != 2 || conflicts[0].Err == nil || conflicts[1].Err == nil {
		t.Errorf("the duplicate and the empty name: %+v", conflicts)
	}
	req, err := r.ParseSlash("/github-issue go-tui-lib bug")
	if err != nil {
		t.Fatal(err)
	}
	msgs := collect(r.Dispatch(t.Context(), req))
	var prompt PromptMsg
	for _, m := range msgs {
		if p, ok := m.(PromptMsg); ok {
			prompt = p
		}
	}
	if prompt.Text != "expanded github_issue" || !slices.Equal(asked, []string{"github_issue go-tui-lib bug"}) {
		t.Errorf("PromptMsg %q; getter asked %v", prompt.Text, asked)
	}
	if _, err := r.Run(t.Context(), Request{ID: "mcp.github.github-issue", Args: []byte(`{}`), Origin: OriginKey}); err == nil {
		t.Error("a missing required argument ran")
	}
	none := FromMCPPrompts("s", []MCPPrompt{{Name: "p"}}, nil)
	if _, err := none[0].Handler.Run(t.Context(), &Invocation{Command: none[0]}); err == nil {
		t.Error("no getter, no error")
	}
}

func TestFromACP(t *testing.T) {
	cmds := FromACP("Claude Code", []ACPCommand{
		{Name: "Plan Mode", Description: "Plan first", Input: &ACPCommandInput{Hint: "goal"}},
		{Name: "web", Description: "Search the web"},
	})
	if len(cmds) != 2 || cmds[0].ID != "acp.claude-code.plan-mode" || cmds[0].Slash != "plan-mode" ||
		cmds[0].Kind != Forward || cmds[0].ArgHint != "goal" || cmds[0].Danger != Mutating {
		t.Fatalf("commands: %+v", cmds)
	}
	r := registryOf(t, nil, cmd("builtin.web", UI, func(c *Command) { c.Slash = "web" }))
	conflicts := r.ReplaceSource(Source{Kind: ACP, Name: "Claude Code"}, cmds)
	if len(conflicts) != 1 || conflicts[0].Renamed != "acp:claude-code:web" {
		t.Errorf("conflicts: %+v", conflicts)
	}
	for line, want := range map[string]string{
		"/plan-mode step one":            "/Plan Mode step one",
		"/acp:claude-code:web go 1.27.1": "/web go 1.27.1",
	} {
		req, err := r.ParseSlash(line)
		if err != nil {
			t.Fatal(err)
		}
		var texts []string
		for _, m := range collect(r.Dispatch(t.Context(), req)) {
			if p, ok := m.(PromptMsg); ok {
				texts = append(texts, p.Text)
			}
		}
		if !slices.Equal(texts, []string{want}) {
			t.Errorf("%q sent %v, want %q", line, texts, want)
		}
	}
	if _, err := r.Run(t.Context(), Request{ID: "acp.claude-code.web", Origin: OriginAgent}); !errors.Is(err, ErrRefused) {
		t.Errorf("a forward from an agent without a gate: %v", err)
	}
}
