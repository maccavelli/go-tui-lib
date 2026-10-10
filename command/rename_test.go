package command

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/when"
)

// The old names still work through v0.9.x
// (docs/decisions/0014-PLAN-canonicalization.md Step 3). These tests sit in
// the package, so that staticcheck does not report the deprecated names
// they use on purpose.

// oldGate is a Gate written against the old name: Decide returns Decision.
type oldGate struct{ d Decision }

func (g oldGate) Decide(context.Context, *Invocation) (Decision, error) { return g.d, nil }

func TestOldDecisionStillWorks(t *testing.T) {
	for d, refused := range map[Decision]bool{AllowOnce: false, AllowAlways: false, RejectOnce: true, RejectAlways: true} {
		r := registryOf(t, []RegistryOption{WithGate(oldGate{d})}, cmd("mut", Mutating))
		_, err := r.Run(t.Context(), Request{ID: "mut", Origin: OriginAgent})
		if (err != nil) != refused || refused && !errors.Is(err, ErrRefused) {
			t.Errorf("an old Gate answering %s: %v", d.ACPKind(), err)
		}
	}
	old := GateFunc(func(context.Context, *Invocation) (Decision, error) { return AllowOnce, nil })
	r := registryOf(t, []RegistryOption{WithGate(old)}, cmd("mut", Mutating))
	if _, err := r.Run(t.Context(), Request{ID: "mut", Origin: OriginAgent}); err != nil {
		t.Errorf("a GateFunc returning Decision: %v", err)
	}
	// It compiles only while Decision and Verdict are one type.
	asVerdict := func(d Decision) Verdict { return d }
	if v := asVerdict(AllowAlways); v != AllowAlways || v.ACPKind() != "allow_always" {
		t.Errorf("a Decision of AllowAlways is %v, %q", v, v.ACPKind())
	}
}

func TestWhenContextFallsBack(t *testing.T) {
	var seen *Invocation
	r := registryOf(t, nil, cmd("on", UI, func(c *Command) {
		c.When = "on"
		c.Handler = HandlerFunc(func(_ context.Context, inv *Invocation) (Result, error) {
			seen = inv
			return Result{}, nil
		})
	}))
	for _, c := range []struct {
		name         string
		whenCtx, ctx when.Context
		runs         bool
	}{
		{"Context alone, the old field", nil, mapOf("on"), true},
		{"WhenContext alone", mapOf("on"), nil, true},
		{"both: WhenContext wins", mapOf("on"), mapOf(), true},
		{"both: an empty WhenContext still wins", when.Map{}, mapOf("on"), false},
		{"neither", nil, nil, false},
	} {
		seen = nil
		_, err := r.Run(t.Context(), Request{ID: "on", Origin: OriginKey, WhenContext: c.whenCtx, Context: c.ctx})
		if ran := err == nil; ran != c.runs {
			t.Errorf("%s: ran %v, %v; want ran %v", c.name, ran, err, c.runs)
			continue
		}
		if !c.runs {
			if !errors.Is(err, ErrUnavailable) {
				t.Errorf("%s: %v, want ErrUnavailable", c.name, err)
			}
			continue
		}
		if seen.WhenContext == nil || seen.Context == nil {
			t.Fatalf("%s: the Invocation's WhenContext %v and Context %v", c.name, seen.WhenContext, seen.Context)
		}
		w, _ := seen.WhenContext.Value("on")
		o, _ := seen.Context.Value("on")
		if w.String() != "true" || o.String() != "true" {
			t.Errorf("%s: the handler saw on = %v in WhenContext and %v in Context", c.name, w, o)
		}
	}
}

func TestRecordCarriesBoth(t *testing.T) {
	var log auditLog
	r := registryOf(t, []RegistryOption{WithAuditor(&log), WithGate(oldGate{AllowAlways})}, cmd("mut", Mutating))
	if _, err := r.Run(t.Context(), Request{ID: "mut", Origin: OriginAgent}); err != nil {
		t.Fatal(err)
	}
	_, _ = r.Run(t.Context(), Request{ID: "nope", Origin: OriginAgent})
	if len(log) != 2 {
		t.Fatalf("%d records, want 2", len(log))
	}
	if got := log[0]; got.Verdict != AllowAlways || got.Decision != AllowAlways {
		t.Errorf("a granted request: Verdict %v, Decision %v; want allow_always in both", got.Verdict, got.Decision)
	}
	if got := log[1]; got.Verdict != 0 || got.Decision != 0 {
		t.Errorf("an unknown command: Verdict %v, Decision %v; want none", got.Verdict, got.Decision)
	}
	for name, rec := range map[string]Record{
		"Verdict":           {ID: "x", Verdict: RejectOnce},
		"Decision, the old": {ID: "x", Decision: RejectOnce},
	} {
		var b bytes.Buffer
		SlogAuditor(slog.New(slog.NewTextHandler(&b, nil))).Audit(rec)
		if !strings.Contains(b.String(), "decision=reject_once") {
			t.Errorf("a Record with %s: the log line %q", name, b.String())
		}
	}
}
