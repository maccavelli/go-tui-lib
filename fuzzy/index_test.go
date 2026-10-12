package fuzzy

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
)

// ids is what ix yields for query, as candidate texts.
func ids(ix *Index, query string, opts ...Option) []string {
	var out []string
	for i := range ix.Candidates(NewMatcher(query, opts...)) {
		out = append(out, ix.At(i))
	}
	return out
}

// TestIndexStrategies: toad's three strategies, by the query's length in
// clusters (docs/decisions/0008-PLAN-command-palette.md Step 4).
func TestIndexStrategies(t *testing.T) {
	ix := NewIndex([]string{"foo.go", "src/foo_test.go", "xfoo", "bar/baz.go", "Fb.txt", "bf", "fab", "no"})
	for _, c := range []struct {
		query string
		want  []string
	}{
		// One cluster: it starts the candidate, or follows a separator;
		// those that start with it first.
		{"f", []string{"foo.go", "Fb.txt", "fab", "src/foo_test.go"}},
		{"F", []string{"foo.go", "Fb.txt", "fab", "src/foo_test.go"}},
		// Two or three: the candidate holds every one, in any order.
		{"fb", []string{"Fb.txt", "bf", "fab"}},
		{"bz.", []string{"bar/baz.go"}},
		// No term: every candidate.
		{"", []string{"foo.go", "src/foo_test.go", "xfoo", "bar/baz.go", "Fb.txt", "bf", "fab", "no"}},
	} {
		if got := ids(ix, c.query); !slices.Equal(got, c.want) {
			t.Errorf("%q yields %q; want %q", c.query, got, c.want)
		}
	}

	wide := NewIndex([]string{"привет", "опрос", "пиво", "ПР"})
	if got, want := ids(wide, "пр"), []string{"привет", "опрос", "ПР"}; !slices.Equal(got, want) {
		t.Errorf("%q yields %q; want %q", "пр", got, want)
	}

	// Four or more: at least the minimum share of the padded trigrams,
	// most shared first. "file" pads to "  f", " fi", "fil", "ile" and
	// "le ": src/file.go shares all five, profile.go and filter three, and
	// f_i_l_e one, 20%, under the default 30%.
	grams := []string{"profile.go", "f_i_l_e", "xyz", "filter", "src/file.go"}
	if got, want := ids(NewIndex(grams), "file"), []string{"src/file.go", "profile.go", "filter"}; !slices.Equal(got, want) {
		t.Errorf("file yields %q; want %q", got, want)
	}
	if got, want := ids(NewIndex(grams, WithMinOverlap(0.2)), "file"), []string{"src/file.go", "profile.go", "filter", "f_i_l_e"}; !slices.Equal(got, want) {
		t.Errorf("file at 20%% yields %q; want %q", got, want)
	}
}

// TestIndexKeepsExactAndPrefix: a candidate that equals the query, or
// whose path element starts with it, holds every padded trigram, so it
// comes before the candidates that only contain the query, and survives a
// limit they fill.
func TestIndexKeepsExactAndPrefix(t *testing.T) {
	var cands []string
	for i := range 50 {
		cands = append(cands, fmt.Sprintf("xfilex%d", i))
	}
	cands = append(cands, "lib/file.go", "file_util.go", "file")
	got := ids(NewIndex(cands, WithMaxCandidates(10)), "file")
	for _, want := range []string{"lib/file.go", "file_util.go", "file"} {
		if !slices.Contains(got, want) {
			t.Errorf("%q was dropped under a limit of 10: %q", want, got)
		}
	}
	if len(got) != 10 {
		t.Errorf("%d candidates under a limit of 10", len(got))
	}

	// Over a generated corpus: a prefix of any candidate's last element,
	// of every length, is never dropped.
	paths := corpus(rand.New(rand.NewPCG(9, 10)), 3000)
	ix := NewIndex(paths)
	r := rand.New(rand.NewPCG(11, 12))
	for range 500 {
		p := paths[r.IntN(len(paths))]
		elem := p[strings.LastIndexByte(p, '/')+1:]
		q := elem[:1+r.IntN(len(elem))]
		if !slices.Contains(ids(ix, q), p) {
			t.Fatalf("%q, a prefix of %q's last element, does not yield it", q, p)
		}
	}
}

// TestIndexUpdate: Update makes a new index, and the old one, read by many
// goroutines while the new one is, is unchanged.
func TestIndexUpdate(t *testing.T) {
	old := NewIndex([]string{"alpha.go", "beta.go", "gamma.go"}, WithMaxCandidates(50))
	before := ids(old, "go")
	next := old.Update([]string{"delta.go"}, []int{1, 9, -1})
	if old.Len() != 3 || old.At(1) != "beta.go" || !slices.Equal(ids(old, "go"), before) {
		t.Fatalf("Update changed the old index: %d, %q", old.Len(), ids(old, "go"))
	}
	if got, want := ids(next, "go"), []string{"alpha.go", "gamma.go", "delta.go"}; !slices.Equal(got, want) {
		t.Fatalf("the new index yields %q; want %q", got, want)
	}
	if next.cfg.maxCandidates != 50 {
		t.Errorf("Update dropped the options: %+v", next.cfg)
	}
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			ix := []*Index{old, next}[g%2]
			for range 200 {
				for _, q := range []string{"a", "go", "alpha", "ta.g"} {
					for i := range ix.Candidates(NewMatcher(q)) {
						_ = ix.At(i)
					}
				}
			}
		})
	}
	wg.Wait()
}

// corpus is n synthetic paths: directories and file names from word lists,
// in snake, kebab and camel case, with extensions.
func corpus(r *rand.Rand, n int) []string {
	words := []string{"api", "app", "auth", "build", "cache", "cmd", "config", "core", "data", "doc", "event", "file",
		"handler", "index", "internal", "io", "key", "layout", "log", "main", "model", "net", "pane", "parse", "path",
		"query", "render", "server", "session", "store", "stream", "test", "theme", "tool", "user", "util", "view", "work"}
	exts := []string{".go", ".md", ".json", ".py", ".ts", ".yaml", ""}
	out := make([]string, n)
	for i := range out {
		var b strings.Builder
		for range r.IntN(4) {
			b.WriteString(words[r.IntN(len(words))])
			b.WriteByte('/')
		}
		parts := make([]string, 1+r.IntN(3))
		for j := range parts {
			parts[j] = words[r.IntN(len(words))]
		}
		switch r.IntN(3) {
		case 0:
			b.WriteString(strings.Join(parts, "_"))
		case 1:
			b.WriteString(strings.Join(parts, "-"))
		default:
			b.WriteString(parts[0])
			for _, p := range parts[1:] {
				b.WriteString(strings.ToUpper(p[:1]) + p[1:])
			}
		}
		fmt.Fprintf(&b, "%d%s", i%10, exts[r.IntN(len(exts))])
		out[i] = b.String()
	}
	return out
}

// subsequence is n bytes of s, in order, chosen by r.
func subsequence(r *rand.Rand, s string, n int) string {
	at := r.Perm(len(s))[:min(n, len(s))]
	slices.Sort(at)
	var b strings.Builder
	for _, i := range at {
		b.WriteByte(s[i])
	}
	return b.String()
}

// TestIndexRecall records, on 20 000 synthetic paths, how many of a full
// scan's matches the index yields, and how many of its ten best, for
// prefixes, words and sparse subsequences. The numbers are for the
// execution record; the test fails only if an exact or a prefix match is
// lost.
func TestIndexRecall(t *testing.T) {
	paths := corpus(rand.New(rand.NewPCG(13, 14)), 20_000)
	ix := NewIndex(paths)
	r := rand.New(rand.NewPCG(15, 16))
	type tally struct{ queries, full, kept, top, topKept int }
	kinds := map[string]*tally{}
	for _, kind := range []string{"prefix", "word", "sparse-2-3", "sparse-4-6"} {
		kinds[kind] = &tally{}
		for range 100 {
			p := paths[r.IntN(len(paths))]
			elem := p[strings.LastIndexByte(p, '/')+1:]
			var q string
			switch kind {
			case "prefix":
				q = elem[:min(len(elem), 4+r.IntN(3))]
			case "word":
				q = []string{"file", "render", "session", "config", "handler"}[r.IntN(5)]
			case "sparse-2-3":
				q = subsequence(r, elem, 2+r.IntN(2))
			default:
				q = subsequence(r, p, 4+r.IntN(3))
			}
			m := NewMatcher(q)
			kept := map[int]bool{}
			for i := range ix.Candidates(m) {
				kept[i] = true
			}
			k := kinds[kind]
			k.queries++
			for i, c := range paths {
				s, ok := m.Score(c)
				if !ok {
					continue
				}
				k.full++
				if kept[i] {
					k.kept++
				}
				if (s.Grade == Exact || s.Grade == Prefix) && !kept[i] && len(kept) < DefaultMaxCandidates {
					t.Errorf("%q: the %s match %q was dropped", q, s.Grade, c)
				}
			}
			for _, rk := range m.Rank(slices.Values(seq(len(paths))), func(i int) string { return paths[i] }, 10) {
				k.top++
				if kept[rk.Item] {
					k.topKept++
				}
			}
		}
	}
	for _, kind := range []string{"prefix", "word", "sparse-2-3", "sparse-4-6"} {
		k := kinds[kind]
		t.Logf("recall %-10s %3d queries: all matches %6d of %6d (%5.1f%%); ten best %4d of %4d (%5.1f%%)",
			kind, k.queries, k.kept, k.full, pct(k.kept, k.full), k.topKept, k.top, pct(k.topKept, k.top))
	}
}

func seq(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func pct(a, b int) float64 {
	if b == 0 {
		return 100
	}
	return 100 * float64(a) / float64(b)
}
