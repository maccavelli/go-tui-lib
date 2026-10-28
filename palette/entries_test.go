package palette

import (
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/pick"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// entries is a line-mode picker's list: 40 models, their IDs distinct,
// context and prices in Detail, made up.
func entries() []pick.Entry {
	names := []string{
		"anthropic/claude-opus-5-5", "anthropic/claude-sonnet-5-5", "anthropic/claude-haiku-5-5", "anthropic/claude-fable-5-1",
		"anthropic/claude-sonnet-4-5", "anthropic/claude-opus-4-1", "openai/gpt-5", "openai/gpt-5-mini", "openai/gpt-5-nano",
		"openai/gpt-4.1", "openai/gpt-4.1-mini", "openai/o3", "openai/o4-mini", "google/gemini-2.5-pro", "google/gemini-2.5-flash",
		"google/gemini-2.5-flash-lite", "google/gemma-3-27b", "mistral/mistral-large-2", "mistral/mistral-medium-3",
		"mistral/codestral-2", "mistral/ministral-8b", "meta/llama-4-maverick", "meta/llama-4-scout", "meta/llama-3.3-70b",
		"deepseek/deepseek-v3", "deepseek/deepseek-r1", "qwen/qwen3-coder", "qwen/qwen3-235b", "xai/grok-4", "xai/grok-3-mini",
		"cohere/command-a", "cohere/command-r-plus", "amazon/nova-pro", "amazon/nova-lite", "microsoft/phi-4",
		"ai21/jamba-large", "moonshot/kimi-k2", "zhipu/glm-4.5", "perplexity/sonar-pro", "local/ollama-person-7b",
	}
	var out []pick.Entry
	for i, n := range names {
		out = append(out, pick.Entry{ID: fmt.Sprintf("m%02d", i), Text: n, Detail: fmt.Sprintf("%dK", 100+i)})
	}
	return out
}

// hitIDs is the IDs of the picker's rows.
func (g *prog) hitIDs() []string {
	var out []string
	for _, r := range g.p.eng.rows() {
		out = append(out, r.hit.ID)
	}
	return out
}

// TestFromEntries: an empty query lists the entries in their order; a
// typed one matches their Text and never their Detail; each hit's Hint is
// the Detail, and accepting one sends SelectedMsg with it, running
// nothing.
func TestFromEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		es := []pick.Entry{
			{ID: "a", Text: "alpha", Detail: "zulu"},
			{ID: "b", Text: "bravo\x1b[31m", Detail: "1M"},
			{ID: "c", Text: "charlie"},
		}
		p := FromEntries("models", es)
		if p.Name() != "models" {
			t.Errorf("Name = %q", p.Name())
		}
		g := newProg(t, keys(t), nil)
		g.register(p)
		g.open()
		if got := g.texts(); !slices.Equal(got, []string{"alpha", "bravo", "charlie"}) {
			t.Fatalf("an empty query lists %q", got)
		}
		for _, r := range g.p.eng.rows() {
			if want := map[string]string{"a": "zulu", "b": "1M", "c": ""}[r.hit.ID]; r.hit.Hint != want || r.hit.Run != nil {
				t.Errorf("hit %s: Hint %q, Run set %v; want %q and no Run", r.hit.ID, r.hit.Hint, r.hit.Run != nil, want)
			}
		}
		g.typed("zulu")
		if got := g.hitIDs(); len(got) != 0 {
			t.Errorf("zulu, a Detail, matched %q", got)
		}
		g.key("ctrl+u")
		g.typed("brav")
		if got := g.hitIDs(); !slices.Equal(got, []string{"b"}) {
			t.Fatalf("brav: %q", got)
		}
		g.key("enter")
		sel := filter[SelectedMsg](g.seen)
		if len(sel) != 1 || sel[0].Hit.ID != "b" || sel[0].Hit.Text != "bravo" {
			t.Errorf("accepting sent %+v; want one SelectedMsg for b, its text cleaned", sel)
		}
	})
}

// chooserIDs is the IDs a pick.Chooser ranks es in for query: each found
// by moving to it and choosing it, through the keys alone.
func chooserIDs(t *testing.T, es []pick.Entry, query string) []string {
	t.Helper()
	now := time.Unix(1, 0)
	var out []string
	for i := 0; ; i++ {
		c := pick.NewChooser(es, pick.WithQuery(query))
		c.Resize(100, now)
		for range i {
			c.HandleInput(termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: termkey.KeyDown}}, now)
		}
		c.HandleInput(termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: termkey.KeyEnter}}, now)
		var chosen *pick.Entry
		for e := c.NextEvent(); e.Kind != pick.PickNone; e = c.NextEvent() {
			if e.Kind == pick.PickDone && !e.Canceled {
				chosen = &e.Entry
			}
		}
		if chosen == nil || slices.Contains(out, chosen.ID) {
			return out // nothing matched, or down stopped at the last match
		}
		out = append(out, chosen.ID)
	}
}

// TestFromEntriesRankedLikeChooser: for each query, the palette's hits for
// FromEntries, through a picker on a workspace, are the entries a
// pick.Chooser ranks, in its order.
func TestFromEntriesRankedLikeChooser(t *testing.T) {
	es := entries()
	for _, q := range []string{"", "son", "gpt", "mini", "o", "lla", "flash", "zzqx", "4"} {
		want := chooserIDs(t, es, q)
		synctest.Test(t, func(t *testing.T) {
			g := newProg(t, keys(t), nil)
			g.register(FromEntries("models", es))
			g.open()
			g.typed(q)
			if got := g.hitIDs(); !slices.Equal(got, want) {
				t.Errorf("%q: the palette ranks %q; the Chooser %q", q, got, want)
			}
		})
	}
}
