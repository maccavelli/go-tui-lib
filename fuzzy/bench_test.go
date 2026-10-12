package fuzzy

import (
	"fmt"
	"slices"
	"testing"
)

// titles is n command titles, as a large registry gives them.
func titles(n int) []string {
	verbs := []string{"Open", "Close", "Toggle", "Focus", "Show", "Hide", "Run", "Copy", "Move", "Rename"}
	nouns := []string{"File", "Editor", "Panel", "Terminal", "Pane", "Session", "Model", "Theme", "Layout", "Agent"}
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s: %s %s %d", nouns[i/100%10], verbs[i%10], nouns[i/10%10], i)
	}
	return out
}

func BenchmarkScoreASCII(b *testing.B) {
	m := NewMatcher("wsfoc")
	cand := "workspace.focus.next: Focus the next pane"
	b.ReportAllocs()
	for b.Loop() {
		m.Score(cand)
	}
}

func BenchmarkRank10000Titles(b *testing.B) {
	ts := titles(10_000)
	m := NewMatcher("tgpnl")
	b.ReportAllocs()
	for b.Loop() {
		m.Rank(slices.Values(ts), func(s string) string { return s }, 50)
	}
}
