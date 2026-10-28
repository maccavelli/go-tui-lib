package pick_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/pick"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// host is a line-mode host on a test screen: it resizes the Chooser to the
// screen, decodes what is typed and what the screen answers, feeds it in,
// and writes what the Chooser asks, until it is done.
type host struct {
	t       *testing.T
	scr     *inlinetest.Screen
	c       *pick.Chooser
	dec     termkey.Decoder
	now     time.Time
	replies []byte
	written []byte // every byte the Chooser asked to write
	done    *pick.ChooserEvent
	dones   int
}

func newHost(t *testing.T, columns int, entries []pick.Entry, o ...pick.Option) *host {
	t.Helper()
	h := &host{t: t, scr: inlinetest.NewScreen(columns, 40, ansi.WcWidth), now: time.Unix(1, 0)}
	h.scr.Answer = func(reply string) { h.replies = append(h.replies, reply...) }
	h.c = pick.NewChooser(entries, append([]pick.Option{pick.WithGlyphs(glyph.ASCII())}, o...)...)
	h.c.Resize(columns, h.now)
	h.drain()
	return h
}

func (h *host) drain() {
	for e := h.c.NextEvent(); e.Kind != pick.PickNone; e = h.c.NextEvent() {
		switch e.Kind {
		case pick.PickWrite:
			h.written = append(h.written, e.Data...)
			_, _ = h.scr.Write([]byte(e.Data))
		case pick.PickDone:
			h.done = &e
			h.dones++
		}
	}
}

// typeIn decodes b and feeds each input, then what the screen answered.
func (h *host) typeIn(b string) {
	for _, in := range h.dec.Flush(h.dec.Decode([]byte(b), nil)) {
		h.c.HandleInput(in, h.now)
		h.drain()
	}
	if len(h.replies) > 0 {
		r := h.replies
		h.replies = nil
		h.typeIn(string(r))
	}
}

// lines is the screen's rows, the query row first.
func (h *host) lines() []string { return h.scr.Lines() }

// models is a program's models, with their context and prices in Detail:
// 40 entries, their prices made up.
func models() []pick.Entry {
	type m struct{ id, ctx, in, out string }
	list := []m{
		{"anthropic/claude-opus-5-5", "200K", "15.00", "75.00"},
		{"anthropic/claude-sonnet-5-5", "1M", "3.00", "15.00"},
		{"anthropic/claude-haiku-5-5", "200K", "0.80", "4.00"},
		{"anthropic/claude-fable-5-1", "200K", "6.00", "30.00"},
		{"anthropic/claude-sonnet-4-5", "1M", "3.00", "15.00"},
		{"anthropic/claude-opus-4-1", "200K", "15.00", "75.00"},
		{"openai/gpt-5", "400K", "1.25", "10.00"},
		{"openai/gpt-5-mini", "400K", "0.25", "2.00"},
		{"openai/gpt-5-nano", "400K", "0.05", "0.40"},
		{"openai/gpt-4.1", "1M", "2.00", "8.00"},
		{"openai/gpt-4.1-mini", "1M", "0.40", "1.60"},
		{"openai/o3", "200K", "2.00", "8.00"},
		{"openai/o4-mini", "200K", "1.10", "4.40"},
		{"google/gemini-2.5-pro", "1M", "1.25", "10.00"},
		{"google/gemini-2.5-flash", "1M", "0.30", "2.50"},
		{"google/gemini-2.5-flash-lite", "1M", "0.10", "0.40"},
		{"google/gemma-3-27b", "128K", "0.10", "0.20"},
		{"mistral/mistral-large-2", "128K", "2.00", "6.00"},
		{"mistral/mistral-medium-3", "128K", "0.40", "2.00"},
		{"mistral/codestral-2", "256K", "0.30", "0.90"},
		{"mistral/ministral-8b", "128K", "0.10", "0.10"},
		{"meta/llama-4-maverick", "1M", "0.20", "0.60"},
		{"meta/llama-4-scout", "10M", "0.10", "0.30"},
		{"meta/llama-3.3-70b", "128K", "0.60", "0.60"},
		{"deepseek/deepseek-v3", "128K", "0.27", "1.10"},
		{"deepseek/deepseek-r1", "128K", "0.55", "2.19"},
		{"qwen/qwen3-coder", "256K", "0.40", "1.60"},
		{"qwen/qwen3-235b", "128K", "0.20", "0.60"},
		{"xai/grok-4", "256K", "3.00", "15.00"},
		{"xai/grok-3-mini", "128K", "0.30", "0.50"},
		{"cohere/command-a", "256K", "2.50", "10.00"},
		{"cohere/command-r-plus", "128K", "2.50", "10.00"},
		{"amazon/nova-pro", "300K", "0.80", "3.20"},
		{"amazon/nova-lite", "300K", "0.06", "0.24"},
		{"microsoft/phi-4", "16K", "0.07", "0.14"},
		{"ai21/jamba-large", "256K", "2.00", "8.00"},
		{"moonshot/kimi-k2", "128K", "0.60", "2.50"},
		{"zhipu/glm-4.5", "128K", "0.60", "2.20"},
		{"perplexity/sonar-pro", "200K", "3.00", "15.00"},
		{"local/ollama-person-7b", "32K", "0.00", "0.00"},
	}
	var out []pick.Entry
	for _, x := range list {
		out = append(out, pick.Entry{ID: x.id, Text: x.id, Detail: fmt.Sprintf("%s  $%s/$%s", x.ctx, x.in, x.out)})
	}
	return out
}

// sessions is a program's sessions, by title and time.
func sessions() []pick.Entry {
	titles := []string{
		"Fix the resize redraw in the inline region",
		"Plan the picker's keys",
		"Review the line-mode question's layout",
		"Why does the session list jump?",
		"Write the streamed Markdown printer",
		"Bump the toolchain floor",
		"Port the permission prompt",
		"Draft release notes for v2.3.0",
		"Explain the cursor report on row one",
		"Benchmark the fuzzy index",
		"Rename the engines' events",
		"Session restore after a crash",
		"Add a history file",
		"Trim the reason's spaces",
		"Sort sessions by time",
		"Find the flaky synctest",
	}
	var out []pick.Entry
	for i, title := range titles {
		ago := fmt.Sprintf("%d hours ago", i+1)
		if i == 0 {
			ago = "just now"
		}
		out = append(out, pick.Entry{ID: fmt.Sprintf("s%02d", i), Text: title, Detail: ago})
	}
	return out
}

// ids is the entries' IDs.
func ids(es []pick.Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.ID)
	}
	return out
}

// order is the IDs a Chooser ranks entries in for query, each found by
// moving to it and choosing it, through the keys alone.
func order(t *testing.T, entries []pick.Entry, query string, o ...pick.Option) []string {
	t.Helper()
	var out []string
	for i := 0; ; i++ {
		h := newHost(t, 100, entries, append([]pick.Option{pick.WithQuery(query)}, o...)...)
		for range i {
			h.typeIn("\x1b[B")
		}
		h.typeIn("\r")
		if h.done == nil {
			if i == 0 {
				return nil // nothing matches
			}
			t.Fatalf("%q: no choice at %d", query, i)
		}
		id := h.done.Entry.ID
		if len(out) > 0 && slices.Contains(out, id) {
			return out // down stopped at the last match
		}
		out = append(out, id)
	}
}
