package fuzzy

import (
	"fmt"
	"math/rand/v2"
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

func BenchmarkNewIndex100000Paths(b *testing.B) {
	paths := corpus(rand.New(rand.NewPCG(17, 18)), 100_000)
	b.ReportAllocs()
	for b.Loop() {
		NewIndex(paths)
	}
}

// BenchmarkIndexedRank100000Paths is a four-cluster query through the
// index, scoring only the candidates it yields; BenchmarkFullRank is the
// same query over every path.
func BenchmarkIndexedRank100000Paths(b *testing.B) {
	paths := corpus(rand.New(rand.NewPCG(17, 18)), 100_000)
	ix := NewIndex(paths)
	m := NewMatcher("sesstor")
	b.ReportAllocs()
	for b.Loop() {
		m.Rank(ix.Candidates(m), ix.At, 50)
	}
}

func BenchmarkFullRank100000Paths(b *testing.B) {
	paths := corpus(rand.New(rand.NewPCG(17, 18)), 100_000)
	m := NewMatcher("sesstor")
	b.ReportAllocs()
	for b.Loop() {
		m.Rank(slices.Values(paths), func(s string) string { return s }, 50)
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
