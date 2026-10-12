package fuzzy

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// ranked is the candidates that match query, in rank order.
func ranked(m *Matcher, cands []string) []string {
	var out []string
	for _, r := range m.Rank(slices.Values(cands), func(s string) string { return s }, 0) {
		out = append(out, r.Item)
	}
	return out
}

// TestOrder pins grade and order for command titles, paths, camelCase,
// digits, Cyrillic, CJK, combining marks and ZWJ emoji
// (docs/decisions/0008-PLAN-command-palette.md Step 3).
func TestOrder(t *testing.T) {
	for _, c := range []struct {
		name, query string
		cands, want []string
		grades      []Grade
	}{
		{
			name: "paths", query: "file",
			cands:  []string{"profile.go", "fiddle", "my_file.txt", "filesystem.go", "file_test.go", "src/file.go", "file", "fil"},
			want:   []string{"file", "src/file.go", "file_test.go", "filesystem.go", "my_file.txt", "fiddle", "profile.go"},
			grades: []Grade{Exact, Exact, Prefix, Prefix, Boundary, Subsequence, Subsequence},
		},
		{
			name: "camelCase", query: "fb",
			cands:  []string{"foobar", "fab", "fooBar", "FooBar", "f_b"},
			want:   []string{"f_b", "fooBar", "FooBar", "fab", "foobar"},
			grades: []Grade{Boundary, Boundary, Boundary, Subsequence, Subsequence},
		},
		{
			name: "digits", query: "v2",
			cands:  []string{"version2", "v1.2", "vv22", "v2"},
			want:   []string{"v2", "v1.2", "vv22", "version2"},
			grades: []Grade{Exact, Boundary, Boundary, Boundary},
		},
		{
			name: "command titles", query: "open",
			cands:  []string{"Reopen Closed Editor", "Open File", "Open", "File: Open Recent", "Toggle Panel"},
			want:   []string{"Open", "Open File", "File: Open Recent", "Reopen Closed Editor"},
			grades: []Grade{Exact, Prefix, Boundary, Subsequence},
		},
		{
			name: "Cyrillic", query: "пр",
			cands:  []string{"опрос", "Привет", "Пр"},
			want:   []string{"Пр", "Привет", "опрос"},
			grades: []Grade{Exact, Prefix, Subsequence},
		},
		{
			name: "CJK", query: "管理",
			cands:  []string{"文件管理器", "管 理", "管理"},
			want:   []string{"管理", "管 理", "文件管理器"},
			grades: []Grade{Exact, Boundary, Subsequence},
		},
		{
			name: "combining mark", query: "caf",
			cands:  []string{"Café", "café", "cafe"},
			want:   []string{"cafe", "café", "Café"},
			grades: []Grade{Prefix, Prefix, Prefix},
		},
		{
			// e and e with its combining acute are different clusters.
			name: "cluster, not rune", query: "cafe",
			cands:  []string{"Café", "café", "cafe"},
			want:   []string{"cafe"},
			grades: []Grade{Exact},
		},
		{
			name: "ZWJ emoji", query: "👨‍👩‍👧",
			cands:  []string{"👨 alone", "👨‍👩‍👧 family", "the 👨‍👩‍👧"},
			want:   []string{"👨‍👩‍👧 family", "the 👨‍👩‍👧"},
			grades: []Grade{Prefix, Boundary},
		},
		{
			name: "part of a ZWJ cluster", query: "👨",
			cands:  []string{"👨‍👩‍👧 family", "👨 alone"},
			want:   []string{"👨 alone"},
			grades: []Grade{Prefix},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := NewMatcher(c.query)
			if got := ranked(m, c.cands); !slices.Equal(got, c.want) {
				t.Fatalf("%q ranks %q; want %q", c.query, got, c.want)
			}
			for i, s := range c.want {
				if sc, _ := m.Score(s); sc.Grade != c.grades[i] {
					t.Errorf("%q in %q is %s; want %s", c.query, s, sc.Grade, c.grades[i])
				}
			}
		})
	}
}

// TestSmartCase: a query with an upper-case letter is case-sensitive under
// Smart, the default; Insensitive and Sensitive hold whatever the query.
func TestSmartCase(t *testing.T) {
	for _, c := range []struct {
		query string
		opts  []Option
		cand  string
		want  bool
	}{
		{"fb", nil, "FooBar", true},
		{"fb", nil, "fooBar", true},
		{"Fb", nil, "fooBar", false},
		{"FB", nil, "FooBar", true},
		{"Fb", []Option{WithCase(Insensitive)}, "fooBar", true},
		{"fb", []Option{WithCase(Sensitive)}, "FooBar", false},
		{"ПР", nil, "привет", false},
		{"пр", nil, "ПРИВЕТ", true},
		// The Kelvin sign folds with k and K.
		{"k", nil, "K", true},
	} {
		if _, ok := NewMatcher(c.query, c.opts...).Score(c.cand); ok != c.want {
			t.Errorf("%q in %q: matched %v; want %v", c.query, c.cand, ok, c.want)
		}
	}
}

// TestExtended: with WithExtended, terms all match and their prefixes
// apply; without it a space is a character.
func TestExtended(t *testing.T) {
	cands := []string{"cmd/foo/main.go", "foo_test.go", "bar/foo.go", "main.go", "README"}
	for _, c := range []struct {
		query string
		want  []string
	}{
		{"foo go", []string{"bar/foo.go", "foo_test.go", "cmd/foo/main.go"}},
		{"'oo.", []string{"bar/foo.go"}},
		{"^foo", []string{"foo_test.go"}},
		{"main.go$", []string{"main.go", "cmd/foo/main.go"}},
		{"^main.go$", []string{"main.go"}},
		{"go$ !test", []string{"main.go", "bar/foo.go", "cmd/foo/main.go"}},
		{"!^cmd go$", []string{"main.go", "bar/foo.go", "foo_test.go"}},
		{"   ", cands},
	} {
		if got := ranked(NewMatcher(c.query, WithExtended()), cands); !slices.Equal(sorted(got), sorted(c.want)) {
			t.Errorf("extended %q matches %q; want %q", c.query, got, c.want)
		}
	}
	if got := ranked(NewMatcher("foo go"), cands); len(got) != 0 {
		t.Errorf("without WithExtended, %q matched %q", "foo go", got)
	}
	if _, ok := NewMatcher("open settings").Score("Open Settings"); !ok {
		t.Error("without WithExtended, a title with its space does not match")
	}
}

func sorted(s []string) []string { return slices.Sorted(slices.Values(s)) }

// TestNormalizer: the hook folds what the matcher compares, and
// NormalizerFunc adapts a function.
func TestNormalizer(t *testing.T) {
	strip := NormalizerFunc(func(r rune) rune {
		if r == 'é' {
			return 'e'
		}
		return r
	})
	if _, ok := NewMatcher("cafe").Score("café"); ok {
		t.Fatal("cafe matched café with no normalizer")
	}
	if _, ok := NewMatcher("cafe", WithNormalizer(strip)).Score("café"); !ok {
		t.Error("cafe did not match café through the normalizer")
	}
	if _, ok := NewMatcher("café", WithNormalizer(strip)).Score("cafe"); !ok {
		t.Error("the normalizer was not applied to the query")
	}
}

// TestLongQuery: a query is matched on its first MaxQuery clusters, so a
// 10 000-cluster query matches as its first 256, and quickly.
func TestLongQuery(t *testing.T) {
	m := NewMatcher(strings.Repeat("a", 10_000))
	if n := len([]rune(m.Query())); n != MaxQuery {
		t.Fatalf("the query keeps %d clusters; want %d", n, MaxQuery)
	}
	if _, ok := m.Score(strings.Repeat("a", 300)); !ok {
		t.Error("a 10 000-cluster query does not match 300 a's as its first 256")
	}
	if got := firstClusters(strings.Repeat("👨‍👩‍👧", 300), 256); got != strings.Repeat("👨‍👩‍👧", 256) {
		t.Error("a query is not cut on a cluster boundary")
	}
}

// TestEmptyQuery: an empty query matches everything with the zero score.
func TestEmptyQuery(t *testing.T) {
	m := NewMatcher("")
	for _, s := range []string{"", "anything"} {
		if mt, ok := m.Match(s); !ok || mt.Score != (Score{}) || mt.Ranges != nil {
			t.Errorf("the empty query on %q: %+v, %v", s, mt, ok)
		}
	}
}

// TestMaxAlign: past WithMaxAlign the greedy alignment is kept, and it
// still spells the query; within it the exact one wins.
func TestMaxAlign(t *testing.T) {
	cand := "a" + strings.Repeat("x", 40) + "b c_ab"
	exact, _ := NewMatcher("ab").Match(cand)
	greedy, _ := NewMatcher("ab", WithMaxAlign(4)).Match(cand)
	if exact.Value <= greedy.Value {
		t.Errorf("the exact alignment scores %d, the greedy %d; want the exact higher", exact.Value, greedy.Value)
	}
	if got := spell(cand, greedy.Ranges); got != "ab" {
		t.Errorf("the greedy alignment spells %q", got)
	}
}

// spell is the text of s's ranges, joined.
func spell(s string, rs []Range) string {
	var b strings.Builder
	for _, r := range rs {
		b.WriteString(s[r.Start:r.End])
	}
	return b.String()
}

// atoms are whole grapheme clusters, which stay clusters when joined.
var atoms = []string{
	"a", "b", "A", "B", "x", "1", "_", "/", " ", ".", "ß", "п", "П", "é",
	"é", "👨‍👩‍👧", "🇫🇷", "文", "\r\n",
}

// drawable are the atoms a row can draw: CR LF, a control cluster with no
// width, is not one, and no cell range can say which side of it a
// highlight ends.
var drawable = slices.DeleteFunc(slices.Clone(atoms), func(a string) bool { return a == "\r\n" })

// randomCase is a candidate of random atoms from set, and a query of some
// of them, in order.
func randomCase(r *rand.Rand, set []string) (cand, query string, picked []string) {
	n := 1 + r.IntN(20)
	parts := make([]string, n)
	for i := range parts {
		parts[i] = set[r.IntN(len(set))]
	}
	for _, p := range parts {
		if r.IntN(3) == 0 {
			picked = append(picked, p)
		}
	}
	if len(picked) == 0 {
		picked = parts[:1]
	}
	return strings.Join(parts, ""), strings.Join(picked, ""), picked
}

// boundaries is the set of s's cluster boundaries.
func boundaries(s string) map[int]bool {
	out := map[int]bool{0: true, len(s): true}
	for i := 0; i < len(s); {
		c, _ := ansi.FirstGraphemeCluster(s[i:], ansi.GraphemeWidth)
		i += len(c)
		out[i] = true
	}
	return out
}

// TestProperties: over random strings, every range lies on cluster
// boundaries inside the candidate, in order, neither overlapping nor
// touching; the ranges spell the query under folding; and Score and Match
// agree.
func TestProperties(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 3000 {
		cand, query, _ := randomCase(r, atoms)
		m := NewMatcher(query, WithCase(Insensitive))
		mt, ok := m.Match(cand)
		if !ok {
			t.Fatalf("%q does not match %q, of which it is a subsequence", query, cand)
		}
		if s, _ := m.Score(cand); s != mt.Score {
			t.Fatalf("%q in %q: Score %+v, Match %+v", query, cand, s, mt.Score)
		}
		checkRanges(t, cand, mt.Ranges)
		want := m.prepare(nil, query, new([]byte), false, false)
		got := m.prepare(nil, spell(cand, mt.Ranges), new([]byte), false, false)
		if !slices.EqualFunc(got, want, func(a, b unit) bool { return a.key == b.key }) {
			t.Fatalf("%q in %q: the ranges %v spell %q", query, cand, mt.Ranges, spell(cand, mt.Ranges))
		}
		if v := rescore(m, cand, mt.Ranges); v != mt.Value {
			t.Fatalf("%q in %q: value %d, its ranges rescore %d", query, cand, mt.Value, v)
		}
	}
}

// checkRanges fails unless rs lie on cand's cluster boundaries, inside it,
// in order, apart.
func checkRanges(t *testing.T, cand string, rs []Range) {
	t.Helper()
	b := boundaries(cand)
	end := -1
	for _, rg := range rs {
		if rg.Start <= end || rg.End <= rg.Start || rg.End > len(cand) || !b[rg.Start] || !b[rg.End] {
			t.Fatalf("in %q the range %v is out of place among %v", cand, rg, rs)
		}
		end = rg.End
	}
}

// rescore is the value of the clusters rs cover, placed one by one.
func rescore(m *Matcher, cand string, rs []Range) int {
	us := m.prepare(nil, cand, new([]byte), false, false)
	var pos []int
	for i, u := range us {
		for _, rg := range rs {
			if u.start >= rg.Start && u.end <= rg.End {
				pos = append(pos, i)
			}
		}
	}
	return scorePositions(us, pos)
}

// TestASCIIPathAgrees: an ASCII candidate's bytes give the same score and
// ranges as walking its clusters.
func TestASCIIPathAgrees(t *testing.T) {
	const alpha = "abcABC019 _-/.:xyzXYZ"
	r := rand.New(rand.NewPCG(3, 4))
	pick := func(n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = alpha[r.IntN(len(alpha))]
		}
		return string(b)
	}
	for range 3000 {
		cand, query := pick(1+r.IntN(30)), pick(1+r.IntN(4))
		for _, m := range []*Matcher{NewMatcher(query), NewMatcher(query, WithExtended())} {
			fs, fok, fr := m.run(cand, true, false)
			ss, sok, sr := m.run(cand, true, true)
			if fs != ss || fok != sok || !slices.Equal(fr, sr) {
				t.Fatalf("%q in %q: bytes %+v %v %v, clusters %+v %v %v", query, cand, fs, fok, fr, ss, sok, sr)
			}
		}
	}
}

// TestScoreAllocatesNothing: the cheap path allocates nothing for ASCII
// once warm.
func TestScoreAllocatesNothing(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector drops sync.Pool entries")
	}
	m := NewMatcher("wsfoc")
	cand := "workspace.focus.next: Focus the next pane"
	if n := testing.AllocsPerRun(100, func() { m.Score(cand) }); n != 0 {
		t.Errorf("Score allocates %v times per call; want 0", n)
	}
}

// TestGradeText: each grade and case mode prints and reads its token.
func TestGradeText(t *testing.T) {
	for g, want := range map[Grade]string{Exact: "exact", Prefix: "prefix", Boundary: "boundary", Subsequence: "subsequence", Grade(9): "Grade(9)"} {
		if g.String() != want {
			t.Errorf("Grade(%d) is %q; want %q", g, g.String(), want)
		}
	}
	var c CaseMode
	if err := c.UnmarshalText([]byte("sensitive")); err != nil || c != Sensitive {
		t.Errorf("UnmarshalText(sensitive): %v, %v", c, err)
	}
	if Smart != 0 {
		t.Error("Smart is not the zero CaseMode")
	}
}
