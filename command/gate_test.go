package command

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/when"
)

func mapOf(keys ...string) when.Map {
	m := when.Map{}
	for _, k := range keys {
		m[k] = when.BoolValue(true)
	}
	return m
}

// gateFunc is a Gate that counts its calls and answers with next.
type gateFunc struct {
	calls int
	next  func(inv *Invocation) (Verdict, error)
}

func (g *gateFunc) Decide(_ context.Context, inv *Invocation) (Verdict, error) {
	g.calls++
	return g.next(inv)
}

func answer(d Verdict, err error) *gateFunc {
	return &gateFunc{next: func(*Invocation) (Verdict, error) { return d, err }}
}

var (
	origins = []Origin{OriginKey, OriginMouse, OriginPalette, OriginSlash, OriginCLI, OriginAgent, OriginProgram}
	dangers = []Danger{ReadOnly, UI, Mutating, Destructive}
)

// asked is MADR §7's table: the requests the policy asks the gate about.
var asked = map[[2]uint8]bool{
	{uint8(Mutating), uint8(OriginAgent)}:    true,
	{uint8(Destructive), uint8(OriginAgent)}: true,
	{uint8(Destructive), uint8(OriginCLI)}:   true,
}

func dangerRegistry(t *testing.T, o ...RegistryOption) *Registry {
	t.Helper()
	cmds := make([]Command, len(dangers))
	for i, d := range dangers {
		cmds[i] = cmd(dangerID(d), d)
	}
	return registryOf(t, o, cmds...)
}

func dangerID(d Danger) ID { return ID(strings.ReplaceAll(d.String(), "read-only", "readonly")) }

func TestPolicy(t *testing.T) {
	gateErr := errors.New("dialog closed")
	for _, g := range []struct {
		name    string
		gate    *gateFunc
		refuses bool  // the gate's answer refuses
		err     error // the gate's error, wrapped in the refusal
	}{
		{"no gate", nil, true, nil},
		{"allow once", answer(AllowOnce, nil), false, nil},
		{"reject once", answer(RejectOnce, nil), true, nil},
		{"no decision", answer(0, nil), true, nil},
		{"a failing gate", answer(AllowOnce, gateErr), true, gateErr},
	} {
		var opts []RegistryOption
		if g.gate != nil {
			opts = append(opts, WithGate(g.gate))
		}
		r := dangerRegistry(t, opts...)
		for _, d := range dangers {
			for _, o := range origins {
				ask := asked[[2]uint8{uint8(d), uint8(o)}]
				calls := 0
				if g.gate != nil {
					calls = g.gate.calls
				}
				_, err := r.Run(t.Context(), Request{ID: dangerID(d), Origin: o})
				if refused := ask && g.refuses; refused != errors.Is(err, ErrRefused) || (!refused && err != nil) {
					t.Errorf("%s: %s from %s: %v; want refused %v", g.name, d, o, err, refused)
				}
				if g.gate != nil && (g.gate.calls > calls) != ask {
					t.Errorf("%s: %s from %s: gate asked %v, want %v", g.name, d, o, g.gate.calls > calls, ask)
				}
				if g.err != nil && ask && !errors.Is(err, g.err) {
					t.Errorf("%s: the gate's error is not wrapped: %v", g.name, err)
				}
			}
		}
	}
	r := dangerRegistry(t, WithGate(answer(AllowOnce, nil)))
	for _, c := range []struct {
		req  Request
		want error
	}{
		{Request{ID: "readonly"}, ErrRefused},
		{Request{ID: "readonly", Origin: OriginProgram + 1}, ErrRefused},
		{Request{ID: "missing", Origin: OriginKey}, ErrUnknown},
	} {
		if _, err := r.Run(t.Context(), c.req); !errors.Is(err, c.want) {
			t.Errorf("Run(%+v): %v, want %v", c.req, err, c.want)
		}
		if res := resultOf(t, collect(r.Dispatch(t.Context(), c.req))); !errors.Is(res.Err, c.want) {
			t.Errorf("Dispatch(%+v): %v, want %v", c.req, res.Err, c.want)
		}
	}
}

func TestAlways(t *testing.T) {
	var queue []Verdict
	g := &gateFunc{next: func(inv *Invocation) (Verdict, error) {
		if len(queue) == 0 {
			t.Errorf("the gate was asked about %s for %s, which it answered always", inv.Command.ID, inv.Caller)
			return RejectOnce, nil
		}
		d := queue[0]
		queue = queue[1:]
		return d, nil
	}}
	r := dangerRegistry(t, WithGate(g))
	run := func(id ID, caller string, d Verdict) error {
		t.Helper()
		if d != 0 {
			queue = append(queue, d)
		}
		_, err := r.Run(t.Context(), Request{ID: id, Origin: OriginAgent, Caller: caller})
		return err
	}
	steps := []struct {
		id      ID
		caller  string
		answer  Verdict // queued for the gate; 0 when it must not be asked
		calls   int
		refused bool
	}{
		{"mutating", "a", AllowAlways, 1, false},
		{"mutating", "a", 0, 1, false}, // remembered
		{"mutating", "b", AllowOnce, 2, false},
		{"mutating", "b", AllowOnce, 3, false}, // AllowOnce is not remembered
		{"destructive", "a", AllowOnce, 4, false},
		{"mutating", "c", RejectOnce, 5, true},
		{"mutating", "c", AllowOnce, 6, false}, // RejectOnce is not remembered
		{"mutating", "d", RejectAlways, 7, true},
		{"mutating", "d", 0, 7, true}, // remembered
	}
	for i, s := range steps {
		err := run(s.id, s.caller, s.answer)
		if g.calls != s.calls || errors.Is(err, ErrRefused) != s.refused {
			t.Errorf("step %d (%s for %s): %d gate calls, %v; want %d, refused %v", i, s.id, s.caller, g.calls, err, s.calls, s.refused)
		}
	}
	if len(queue) != 0 {
		t.Errorf("answers left unasked: %v", queue)
	}
}

// TestRequestGate: a request's own gate is asked instead of the
// registry's, its answers are not remembered, and remembered answers do
// not override it (A9).
func TestRequestGate(t *testing.T) {
	registryGate := answer(AllowAlways, nil)
	r := dangerRegistry(t, WithGate(registryGate))
	run := func(g Gate, id ID) error {
		t.Helper()
		_, err := r.Run(t.Context(), Request{ID: id, Origin: OriginCLI, Caller: "sh", Gate: g})
		return err
	}
	own := answer(RejectOnce, nil)
	if err := run(own, "destructive"); !errors.Is(err, ErrRefused) || own.calls != 1 || registryGate.calls != 0 {
		t.Errorf("the request's gate refusing: %v; it was asked %d, the registry's %d", err, own.calls, registryGate.calls)
	}
	always := answer(AllowAlways, nil)
	if err := run(always, "destructive"); err != nil || always.calls != 1 {
		t.Errorf("the request's gate allowing: %v", err)
	}
	if err := run(nil, "destructive"); err != nil || registryGate.calls != 1 {
		t.Errorf("the request gate's AllowAlways was remembered: the registry's gate was asked %d times", registryGate.calls)
	}
	// The registry's gate said AllowAlways, which is now remembered; a
	// request's own gate is still asked, and refuses.
	if err := run(own, "destructive"); !errors.Is(err, ErrRefused) || own.calls != 2 {
		t.Errorf("a remembered answer overrode the request's gate: %v", err)
	}
	if err := run(own, "ui"); err != nil || own.calls != 2 {
		t.Errorf("the request's gate was asked about a command the policy does not ask about: %v", err)
	}
}

// TestDefaultGatePolicy: without WithGatePolicy, with a nil policy, and
// with DefaultGatePolicy, the gate is asked about §7's table and nothing
// else (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D4).
func TestDefaultGatePolicy(t *testing.T) {
	for _, c := range []struct {
		name string
		opts []RegistryOption
	}{
		{"no policy", nil},
		{"a nil policy", []RegistryOption{WithGatePolicy(nil)}},
		{"DefaultGatePolicy", []RegistryOption{WithGatePolicy(DefaultGatePolicy())}},
	} {
		g := answer(AllowOnce, nil)
		r := dangerRegistry(t, append(c.opts, WithGate(g))...)
		for _, d := range dangers {
			for _, o := range origins {
				ask := asked[[2]uint8{uint8(d), uint8(o)}]
				calls := g.calls
				if _, err := r.Run(t.Context(), Request{ID: dangerID(d), Origin: o}); err != nil {
					t.Errorf("%s: %s from %s: %v", c.name, d, o, err)
				}
				if (g.calls > calls) != ask {
					t.Errorf("%s: %s from %s: gate asked %v, want %v", c.name, d, o, g.calls > calls, ask)
				}
			}
		}
	}
	for _, d := range dangers {
		for _, o := range origins {
			if got, want := DefaultGatePolicy().Asks(cmd(dangerID(d), d), o), asked[[2]uint8{uint8(d), uint8(o)}]; got != want {
				t.Errorf("DefaultGatePolicy().Asks(%s, %s) = %v, want %v", d, o, got, want)
			}
		}
	}
}

// TestGatePolicy: a program's policy decides which requests ask the gate,
// is given the command and the origin, and refuses when it panics.
func TestGatePolicy(t *testing.T) {
	destructive := GatePolicyFunc(func(c Command, _ Origin) bool { return c.Danger == Destructive })
	g := answer(RejectOnce, nil)
	r := dangerRegistry(t, WithGatePolicy(destructive), WithGate(g))
	if _, err := r.Run(t.Context(), Request{ID: "destructive", Origin: OriginKey}); !errors.Is(err, ErrRefused) || g.calls != 1 {
		t.Errorf("a destructive key request under a policy that asks for every destructive one: %v, %d gate calls; want refused by the gate", err, g.calls)
	}
	if _, err := r.Run(t.Context(), Request{ID: "mutating", Origin: OriginAgent}); err != nil || g.calls != 1 {
		t.Errorf("an agent's mutating request under the same policy: %v, %d gate calls; want it run unasked", err, g.calls)
	}

	none := dangerRegistry(t, WithGatePolicy(GatePolicyFunc(func(Command, Origin) bool { return false })))
	for _, id := range []ID{"mutating", "destructive"} {
		if _, err := none.Run(t.Context(), Request{ID: id, Origin: OriginAgent}); err != nil {
			t.Errorf("an agent's %s request under a policy that asks for nothing, with no gate: %v", id, err)
		}
	}

	quit := GatePolicyFunc(func(c Command, o Origin) bool { return c.ID == BuiltinQuit && o == OriginKey })
	g = answer(RejectOnce, nil)
	r = dangerRegistry(t, WithGatePolicy(quit), WithGate(g))
	if res := resultOf(t, collect(r.Dispatch(t.Context(), Request{ID: BuiltinQuit, Origin: OriginKey}))); !errors.Is(res.Err, ErrRefused) || g.calls != 1 {
		t.Errorf("app.quit under a policy that gates it: %v, %d gate calls", res.Err, g.calls)
	}

	g = answer(AllowOnce, nil)
	r = dangerRegistry(t, WithGatePolicy(GatePolicyFunc(func(Command, Origin) bool { panic("policy") })), WithGate(g))
	_, err := r.Run(t.Context(), Request{ID: "readonly", Origin: OriginKey})
	pe, ok := errors.AsType[*PanicError](err)
	if !errors.Is(err, ErrRefused) || !ok || pe.Value != "policy" || pe.ID != "readonly" || g.calls != 0 {
		t.Errorf("a panicking policy: %v, %d gate calls; want refused with a *PanicError, the gate unasked", err, g.calls)
	}
}

// TestForget: Forget drops one command and caller's remembered answer, and
// ForgetAll every one.
func TestForget(t *testing.T) {
	g := answer(AllowAlways, nil)
	r := dangerRegistry(t, WithGate(g))
	run := func(id ID, caller string, calls int) {
		t.Helper()
		if _, err := r.Run(t.Context(), Request{ID: id, Origin: OriginAgent, Caller: caller}); err != nil || g.calls != calls {
			t.Errorf("%s for %s: %v, %d gate calls; want %d", id, caller, err, g.calls, calls)
		}
	}
	run("mutating", "a", 1)
	run("mutating", "a", 1) // remembered
	r.Forget("mutating", "b")
	r.Forget("destructive", "a")
	run("mutating", "a", 1) // another caller's, and another command's, forgotten
	r.Forget("mutating", "a")
	run("mutating", "a", 2) // asked again, and remembered again
	run("mutating", "b", 3)
	run("destructive", "a", 4)
	run("mutating", "a", 4)
	r.ForgetAll()
	run("mutating", "a", 5)
	run("mutating", "b", 6)
	run("destructive", "a", 7)
}

func TestVerdictACPKind(t *testing.T) {
	for d, want := range map[Verdict]string{
		AllowOnce: "allow_once", AllowAlways: "allow_always", RejectOnce: "reject_once", RejectAlways: "reject_always", 0: "",
	} {
		if got := d.ACPKind(); got != want {
			t.Errorf("Verdict(%d).ACPKind() = %q, want %q", d, got, want)
		}
	}
}

// countingHandler counts the records the default logger is given.
type countingHandler struct{ n *int }

func (h countingHandler) Enabled(context.Context, slog.Level) bool  { return true }
func (h countingHandler) Handle(context.Context, slog.Record) error { *h.n++; return nil }
func (h countingHandler) WithAttrs([]slog.Attr) slog.Handler        { return h }
func (h countingHandler) WithGroup(string) slog.Handler             { return h }

func TestSlogAuditor(t *testing.T) {
	var defaults int
	old := slog.Default()
	slog.SetDefault(slog.New(countingHandler{&defaults}))
	t.Cleanup(func() { slog.SetDefault(old) })

	var buf bytes.Buffer
	r := dangerRegistry(t, WithAuditor(SlogAuditor(slog.New(slog.NewJSONHandler(&buf, nil)))))
	if _, err := r.Run(t.Context(), Request{ID: "readonly", Origin: OriginKey, Caller: "me", Args: []byte(`{"a":1}`)}); err != nil {
		t.Fatal(err)
	}
	_ = resultOf(t, collect(r.Dispatch(t.Context(), Request{ID: "destructive", Origin: OriginAgent})))
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("%d records, want one per request:\n%s", len(lines), buf.String())
	}
	var ok, refused map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &ok); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &refused); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{
		"msg": "command", "level": "INFO", "id": "readonly", "origin": "key", "caller": "me",
		"args": `{"a":1}`, "verdict": "allow_once",
	} {
		if ok[k] != want {
			t.Errorf("record %s = %v, want %v", k, ok[k], want)
		}
	}
	if refused["level"] != "WARN" || refused["verdict"] != "reject_once" || !strings.Contains(fmt.Sprint(refused["err"]), "refused") {
		t.Errorf("the refusal's record: %v", refused)
	}
	if defaults != 0 {
		t.Errorf("the default logger was given %d records", defaults)
	}
	SlogAuditor(nil).Audit(Record{ID: "x"})
}
