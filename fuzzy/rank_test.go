package fuzzy

import (
	"context"
	"errors"
	"iter"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// item is a candidate with an identity, so ties can be told apart.
type item struct {
	id   int
	text string
}

func text(it item) string { return it.text }

// TestRankStableAndTotal: equal candidates come out in input order, a
// limit keeps the same head as no limit, and the order is total.
func TestRankStableAndTotal(t *testing.T) {
	m := NewMatcher("ab")
	items := []item{{0, "xab"}, {1, "ab"}, {2, "xab"}, {3, "ab"}, {4, "a_b"}, {5, "zzz"}, {6, "ab"}}
	all := m.Rank(slices.Values(items), text, 0)
	var ids []int
	for _, r := range all {
		ids = append(ids, r.Item.id)
	}
	if want := []int{1, 3, 6, 4, 0, 2}; !slices.Equal(ids, want) {
		t.Fatalf("the order is %v; want %v", ids, want)
	}
	for limit := 1; limit <= len(all); limit++ {
		head := m.Rank(slices.Values(items), text, limit)
		if !slices.Equal(head, all[:limit]) {
			t.Fatalf("limit %d keeps %v; want %v", limit, head, all[:limit])
		}
	}
	r := rand.New(rand.NewPCG(5, 6))
	for range 2000 {
		a := &entry[int]{Score: Score{Grade: Grade(r.IntN(4))}, value: r.IntN(3), size: r.IntN(3), index: r.IntN(3)}
		b := &entry[int]{Score: Score{Grade: Grade(r.IntN(4))}, value: r.IntN(3), size: r.IntN(3), index: r.IntN(3)}
		same := a.Score.Grade == b.Score.Grade && a.value == b.value && a.size == b.size && a.index == b.index
		if before(a, b) == before(b, a) && !same {
			t.Fatalf("before is not total on %+v and %+v", a, b)
		}
	}
}

// TestBoostBounded: a boost wins a near-tie within a grade, and never
// overturns a better grade or a lead of more than one consecutive-match
// bonus.
func TestBoostBounded(t *testing.T) {
	m := NewMatcher("ab")
	boost := func(on string, by int) func(string) int {
		return func(s string) int {
			if s == on {
				return by
			}
			return 0
		}
	}
	first := func(cands []string, b func(string) int) string {
		out, err := m.RankContext(context.Background(), slices.Values(cands), func(s string) string { return s }, 0, b)
		if err != nil || len(out) == 0 {
			t.Fatalf("RankContext: %v, %v", out, err)
		}
		return out[0].Item
	}
	// "xa_b" leads "xab" by 1, within Subsequence: a boost of 4 wins.
	if got := first([]string{"xa_b", "xab"}, boost("xab", 4)); got != "xab" {
		t.Errorf("a boost of 4 over a lead of 1: %q first; want xab", got)
	}
	// "xab" leads "xa__xb" by 9: no boost overturns it.
	if got := first([]string{"xab", "xa__xb"}, boost("xa__xb", 1000)); got != "xab" {
		t.Errorf("a boost of 1000 over a lead of 9: %q first; want xab", got)
	}
	// "ab" is Exact and "abc" Prefix: no boost overturns the grade.
	if got := first([]string{"abc", "ab"}, boost("abc", 1000)); got != "ab" {
		t.Errorf("a boost over a better grade: %q first; want ab", got)
	}
	// A negative boost is bounded too.
	if got := first([]string{"xab", "xa__xb"}, boost("xab", -1000)); got != "xab" {
		t.Errorf("a boost of -1000 over a lead of 9: %q first; want xab", got)
	}
}

// TestRankContextCancel: RankContext stops with ctx.Err() within
// checkEvery items of the cancel, even on an endless sequence.
func TestRankContextCancel(t *testing.T) {
	m := NewMatcher("a")
	ctx, cancel := context.WithCancel(context.Background())
	seen := 0
	endless := iter.Seq[string](func(yield func(string) bool) {
		for {
			seen++
			if seen == 1000 {
				cancel()
			}
			if !yield("abc") {
				return
			}
		}
	})
	done := make(chan error, 1)
	go func() {
		_, err := m.RankContext(ctx, endless, func(s string) string { return s }, 10, nil)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RankContext returned %v; want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RankContext did not return after its context was cancelled")
	}
	if seen > 1000+checkEvery {
		t.Errorf("it read %d items; want at most %d", seen, 1000+checkEvery)
	}
}

// cut is the cells [left, right) of s in method m: ansi.Cut under
// GraphemeWidth; under WcWidth, where x/ansi's Cut ignores left and its
// Truncate measures a ZWJ cluster unlike its StringWidth, the cut the
// workspace makes, internal/sanitize's Truncate, to right less its cut to
// left (docs/decisions/0008-MADR-command-palette.md A4).
func cut(m ansi.Method, s string, left, right int) string {
	if m == ansi.GraphemeWidth {
		return ansi.Cut(s, left, right)
	}
	upTo := sanitize.Truncate(s, right, m, "")
	before := sanitize.Truncate(s, left, m, "")
	if !strings.HasPrefix(upTo, before) {
		return "\x00not a prefix"
	}
	return upTo[len(before):]
}

// TestCells: each cell range cuts, in both width methods, exactly the text
// of its byte range.
func TestCells(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8))
	for range 2000 {
		cand, query, _ := randomCase(r, drawable)
		mt, ok := NewMatcher(query, WithCase(Insensitive)).Match(cand)
		if !ok {
			t.Fatalf("%q does not match %q", query, cand)
		}
		for _, method := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
			cells := Cells(cand, mt.Ranges, method)
			i := 0
			for _, rg := range mt.Ranges {
				if method.StringWidth(cand[rg.Start:rg.End]) == 0 {
					continue
				}
				if i >= len(cells) {
					t.Fatalf("%q, method %d: %d cell ranges for %v", cand, method, len(cells), mt.Ranges)
				}
				if got := cut(method, cand, cells[i].Start, cells[i].End); got != cand[rg.Start:rg.End] {
					t.Fatalf("%q, method %d: cells %v cut %q; the range %v is %q", cand, method, cells[i], got, rg, cand[rg.Start:rg.End])
				}
				i++
			}
			if i != len(cells) {
				t.Fatalf("%q, method %d: %d cell ranges; want %d", cand, method, len(cells), i)
			}
		}
	}
	if got := Cells("abc", []Range{{2, 3}, {0, 1}, {1, 9}}, ansi.GraphemeWidth); !slices.Equal(got, []CellRange{{2, 3}}) {
		t.Errorf("ranges out of order or outside: %v", got)
	}
}
