package command

import (
	"context"
	"testing"
)

// TestCoreDispatchWithoutTea: a host that is not Bubble Tea runs
// Dispatch's Effect, and the []Effect it returns, by hand, and sees the
// effect's message, the ResultMsg and, for a Prompt, the PromptMsg; a nil
// Result.Cmd is a nil entry it skips
// (docs/decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md Step 10).
func TestCoreDispatchWithoutTea(t *testing.T) {
	type effMsg struct{}
	r := registryOf(t, nil,
		cmd("a", UI, func(c *Command) {
			c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) {
				return Result{Text: "done", Cmd: func() any { return effMsg{} }}, nil
			})
		}),
		cmd("p", UI, func(c *Command) {
			c.Kind = Prompt
			c.Handler = HandlerFunc(func(context.Context, *Invocation) (Result, error) { return Result{Text: "say hi"}, nil })
		}))

	batch, ok := r.Dispatch(t.Context(), Request{ID: "a", Origin: OriginKey})().([]Effect)
	if !ok || len(batch) != 2 || batch[0] == nil || batch[1] == nil {
		t.Fatalf("a Loop command's effect: %#v, want a batch of its Result.Cmd and a ResultMsg", batch)
	}
	if m := batch[0](); m != (effMsg{}) {
		t.Errorf("the Result.Cmd gave %v", m)
	}
	if m, ok := batch[1]().(ResultMsg); !ok || m.Result.Text != "done" || m.Err != nil {
		t.Errorf("then %+v", m)
	}

	batch, ok = r.Dispatch(t.Context(), Request{ID: "p", Origin: OriginKey})().([]Effect)
	if !ok || len(batch) != 3 || batch[0] != nil {
		t.Fatalf("a prompt's effect: %d entries, first nil %v; want 3, the first nil", len(batch), batch[0] == nil)
	}
	if _, ok := batch[1]().(ResultMsg); !ok {
		t.Error("the prompt's second entry is no ResultMsg")
	}
	if m, ok := batch[2]().(PromptMsg); !ok || m.Text != "say hi" {
		t.Errorf("the prompt's third entry %+v", m)
	}

	if m, ok := r.Dispatch(t.Context(), Request{ID: "missing", Origin: OriginKey})().(ResultMsg); !ok || m.Err == nil {
		t.Errorf("an unknown command's effect: %+v, want a ResultMsg with its error", m)
	}
}
