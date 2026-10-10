package command

import (
	"context"
	"testing"
)

// TestAuditorFunc: a function used as an Auditor receives the registry's
// records (docs/decisions/0014-PLAN-canonicalization.md Step 7).
func TestAuditorFunc(t *testing.T) {
	var got []Record
	r := registryOf(t, []RegistryOption{WithAuditor(AuditorFunc(func(rec Record) { got = append(got, rec) }))},
		testCommand(t, "hello", func(context.Context, *Invocation, struct{}) (Result, error) { return Result{}, nil }))
	if _, err := r.Run(t.Context(), Request{ID: "hello", Origin: OriginKey}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "hello" || got[0].Origin != OriginKey {
		t.Errorf("records %+v, want one for hello from a key", got)
	}
}
