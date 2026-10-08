package command

import (
	"context"
	"errors"
	"testing"
)

type addArgs struct {
	Files []string `json:"files" arg:""`
	Force bool     `json:"force,omitzero" short:"f"`
}

type reviewArgs struct {
	Focus string `json:"focus,omitzero" help:"what to look at first"`
}

func slashRegistry(t testing.TB) *Registry {
	t.Helper()
	ok := func(context.Context, *Invocation, resizeArgs) (Result, error) { return Result{}, nil }
	return registryOf(t, nil,
		testCommand(t, "workspace.resize", ok, WithSlash("resize", "rs")),
		testCommand(t, "files.add", func(context.Context, *Invocation, addArgs) (Result, error) { return Result{}, nil }, WithSlash("add")),
		testCommand(t, "review", func(context.Context, *Invocation, reviewArgs) (Result, error) { return Result{}, nil }, WithSlash("review")),
		cmd("acp.agent.plan", UI, func(c *Command) { c.Kind, c.Handler, c.Slash = Forward, nil, "plan" }),
	)
}

func TestSlashForms(t *testing.T) {
	r := slashRegistry(t)
	want := `{"delta":4,"split":"sidebar"}`
	for _, line := range []string{
		"/resize sidebar 4",
		"/resize split=sidebar delta=4",
		`/resize "sidebar" "4"`,
		"/resize delta=4 sidebar",
		"/resize split=sidebar 4",
		"  /rs\tsidebar   4  ",
		"resize sidebar 4",
		`/resize 'side'bar 4`,
	} {
		req, err := r.ParseSlash(line)
		if err != nil {
			t.Errorf("%q: %v", line, err)
			continue
		}
		if string(req.Args) != want || req.ID != "workspace.resize" || req.Origin != OriginSlash {
			t.Errorf("%q: %+v, args %s; want %s", line, req, req.Args, want)
		}
	}
	if req, _ := r.ParseSlash("/resize sidebar 4"); req.Raw != "sidebar 4" {
		t.Errorf("Raw = %q", req.Raw)
	}
	for line, path := range map[string]string{
		"/resize sidebar 4 5":     "",
		"/resize bogus=1 x 1":     "/bogus",
		"/resize sidebar x":       "/delta",
		"/resize sidebar 999":     "/delta",
		"/resize sidebar":         "/delta",
		"/resize sidebar 1e999":   "/delta",
		`/resize "sidebar 4`:      "",
		"/resize delta=1 delta=2": "/delta",
		"/add a force=maybe":      "/force",
	} {
		_, err := r.ParseSlash(line)
		ae, ok := errors.AsType[*ArgError](err)
		if !ok || ae.Path != path {
			t.Errorf("%q: %v, want an *ArgError at %q", line, err, path)
		}
	}
	if _, err := r.ParseSlash("/nothing here"); !errors.Is(err, ErrUnknown) {
		t.Errorf("an unknown slash name: %v", err)
	}
	req, err := r.ParseSlash(`/add a "b c" force=true d`)
	if err != nil || string(req.Args) != `{"files":["a","b c","d"],"force":true}` {
		t.Errorf("a positional array: %s, %v", req.Args, err)
	}
	if req, err := r.ParseSlash("/plan the  next step "); err != nil || req.Args != nil || req.Raw != "the  next step " {
		t.Errorf("a command without a schema: %+v, %v", req, err)
	}
}

func TestSlashWholeTail(t *testing.T) {
	r := slashRegistry(t)
	for line, want := range map[string]string{
		`/review the "quoted" text  `: `{"focus":"the \"quoted\" text"}`,
		"/review a=b":                 `{"focus":"a=b"}`,
		"/review":                     `{}`,
		"/review   ":                  `{}`,
	} {
		req, err := r.ParseSlash(line)
		if err != nil || string(req.Args) != want {
			t.Errorf("%q: %s, %v; want %s", line, req.Args, err, want)
		}
	}
}
