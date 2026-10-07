package command

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"strings"
	"testing"
	"time"
)

// mustNew is New, failing t on an error.
func mustNew[A any](t testing.TB, id ID, run func(context.Context, *Invocation, A) (Result, error), opts ...Option) Command {
	t.Helper()
	c, err := New(id, string(id), run, append([]Option{WithDanger(UI)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type strictArgs struct {
	Name  string `json:"name" enum:"a,b"`
	Delta int    `json:"delta" schema:"min=-5,max=5"`
	Inner struct {
		X int `json:"x"`
	} `json:"inner,omitzero"`
	Every time.Duration `json:"every,omitzero"`
}

func TestDecodeStrict(t *testing.T) {
	var got strictArgs
	r := registryOf(t, nil, mustNew(t, "strict", func(_ context.Context, _ *Invocation, a strictArgs) (Result, error) {
		got = a
		return Result{}, nil
	}))
	for args, path := range map[string]string{
		`{"name":"a","delta":1,"bogus":1}`:             "/bogus",
		`{"name":"a","delta":"x"}`:                     "/delta",
		`{"delta":1}`:                                  "/name",
		`{"name":"a","delta":9}`:                       "/delta",
		`{"name":"a","delta":-6}`:                      "/delta",
		`{"name":"c","delta":1}`:                       "/name",
		`{"name":"a","delta":1.5}`:                     "/delta",
		`{"name":"a","delta":1,"inner":{"x":1,"y":2}}`: "/inner/y",
		`{"name":"a","delta":1,"every":"soon"}`:        "/every",
		`{"name":"a","delta":1,"name":"b"}`:            "",
		`not json`:                                     "",
		`[1]`:                                          "",
		`{"name":"a","delta":1,"inner":{"x":"one"}}`:   "/inner/x",
		`{"name":"a","delta":1,"every":"1m","x~/y":1}`: "/x~0~1y",
	} {
		_, err := r.Run(t.Context(), Request{ID: "strict", Args: []byte(args), Origin: OriginKey})
		ae, ok := errors.AsType[*ArgError](err)
		if !ok || ae.Path != path {
			t.Errorf("%s: %v, want an *ArgError at %q", args, err, path)
		}
	}
	if _, err := r.Run(t.Context(), Request{ID: "strict", Args: []byte(`{"name":"b","delta":-5,"every":"1m30s"}`), Origin: OriginKey}); err != nil {
		t.Fatal(err)
	}
	if got.Name != "b" || got.Delta != -5 || got.Every != 90*time.Second {
		t.Errorf("decoded %+v", got)
	}
}

// TestUnknownMembersEachLayer: the schema check and the strict decode
// each refuse an unknown member alone, though the registry runs both.
func TestUnknownMembersEachLayer(t *testing.T) {
	s, err := SchemaOf[strictArgs]()
	if err != nil {
		t.Fatal(err)
	}
	r, err := compileRule(s)
	if err != nil {
		t.Fatal(err)
	}
	extra := []byte(`{"name":"a","delta":1,"bogus":1}`)
	if _, err := r.prepare(extra); err == nil {
		t.Error("the schema check accepted an unknown member")
	}
	if _, err := decodeArgs[strictArgs](extra); err == nil {
		t.Error("the strict decode accepted an unknown member")
	} else if ae, ok := errors.AsType[*ArgError](err); !ok || ae.Path != "/bogus" {
		t.Errorf("the strict decode's error: %v", err)
	}
}

func TestDefaultsFilled(t *testing.T) {
	type args struct {
		Mode    string        `json:"mode,omitzero" enum:"a,b" default:"b"`
		N       int           `json:"n" default:"3"`
		Timeout time.Duration `json:"timeout" default:"2s"`
	}
	var got args
	var raw string
	r := registryOf(t, nil, mustNew(t, "d", func(_ context.Context, inv *Invocation, a args) (Result, error) {
		got, raw = a, string(inv.Args)
		return Result{}, nil
	}))
	for _, in := range []string{"", "{}", " "} {
		if _, err := r.Run(t.Context(), Request{ID: "d", Args: []byte(in), Origin: OriginKey}); err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got.Mode != "b" || got.N != 3 || got.Timeout != 2*time.Second || raw != `{"mode":"b","n":3,"timeout":"2s"}` {
			t.Errorf("%q: %+v from %s", in, got, raw)
		}
	}
	if _, err := r.Run(t.Context(), Request{ID: "d", Args: []byte(`{"n":7}`), Origin: OriginKey}); err != nil || got.N != 7 {
		t.Errorf("a given value lost to its default: %+v, %v", got, err)
	}
}

func TestLoadedSchema(t *testing.T) {
	// A schema from elsewhere, carried as is: only the listed keywords are
	// enforced (MADR §4).
	r := registryOf(t, nil, cmd("loaded", UI, func(c *Command) {
		c.Args = Schema(`{"type":"object","properties":{` +
			`"q":{"type":["string","null"],"maxLength":3,"format":"ignored"},` +
			`"n":{"type":"integer","minimum":1},` +
			`"l":{"type":"array","items":{"type":"string"}},` +
			`"m":{"type":"object","additionalProperties":{"type":"boolean"}}},` +
			`"required":["n"]}`)
	}))
	for args, ok := range map[string]bool{
		`{"n":1}`: true, `{"n":1,"q":null}`: true, `{"n":1,"q":"abc"}`: true, `{"n":1,"extra":true}`: true,
		`{"n":1,"m":{"a":true}}`: true, `{"n":1,"l":["x"]}`: true,
		`{}`: false, `{"n":0}`: false, `{"n":1,"q":"abcd"}`: false, `{"n":1,"q":1}`: false,
		`{"n":1,"l":[1]}`: false, `{"n":1,"m":{"a":1}}`: false,
	} {
		_, err := r.Run(t.Context(), Request{ID: "loaded", Args: []byte(args), Origin: OriginKey})
		if (err == nil) != ok {
			t.Errorf("%s: %v, want ok %v", args, err, ok)
		}
	}
	if err := NewRegistry().Register(cmd("bad", UI, func(c *Command) { c.Args = Schema(`[1]`) })); err == nil {
		t.Error("a schema that is not an object registered")
	}
	deep := strings.Repeat(`{"items":`, maxSchemaDepth+2) + "{}" + strings.Repeat("}", maxSchemaDepth+2)
	if err := NewRegistry().Register(cmd("deep", UI, func(c *Command) { c.Args = Schema(deep) })); err == nil {
		t.Error("a schema nested past the limit registered")
	}
}

// auditLog is an Auditor that keeps every record.
type auditLog []Record

func (l *auditLog) Audit(r Record) { *l = append(*l, r) }

func TestAuditMasksSecrets(t *testing.T) {
	type args struct {
		User   string `json:"user"`
		Token  string `json:"token" schema:"secret"`
		Nested struct {
			Key string `json:"key" schema:"secret"`
		} `json:"nested,omitzero"`
		Keys []string `json:"keys,omitzero" schema:"secret"`
	}
	var log auditLog
	r := registryOf(t, []RegistryOption{WithAuditor(&log)},
		mustNew(t, "login", func(context.Context, *Invocation, args) (Result, error) { return Result{}, nil },
			WithDanger(Destructive), WithWhen("!off")),
	)
	full := `{"user":"me","token":"s3cret","nested":{"key":"s3cret"},"keys":["s3cret"]}`
	for _, req := range []Request{
		{Args: []byte(full), Origin: OriginKey},                                       // runs
		{Args: []byte(full), Origin: OriginAgent},                                     // refused
		{Args: []byte(full), Origin: OriginKey, Context: mapOf("off")},                // unavailable
		{Args: []byte(`{"user":"me","token":"s3cret","bogus":1}`), Origin: OriginKey}, // bad arguments
		{Args: []byte(`{"token":"s3cret"`), Origin: OriginKey},                        // not JSON
	} {
		req.ID = "login"
		_, _ = r.Run(t.Context(), req)
		_ = collect(r.Dispatch(t.Context(), req))
	}
	if len(log) != 10 {
		t.Fatalf("%d records, want 10", len(log))
	}
	for i, rec := range log {
		if strings.Contains(string(rec.Args), "s3cret") {
			t.Errorf("record %d holds the secret: %s", i, rec.Args)
		}
	}
	var first map[string]any
	if err := json.Unmarshal(log[0].Args, &first); err != nil {
		t.Fatal(err)
	}
	if first["user"] != "me" || first["token"] != secretMask {
		t.Errorf("the first record's arguments: %v", first)
	}
	if log[8].Args != nil {
		t.Errorf("unreadable arguments with a secret were recorded: %s", log[8].Args)
	}
}
