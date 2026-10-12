package fuzzy

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzIndex: no candidates and query make the index panic, yield an index
// out of range or twice, or yield more than its limit; and Update leaves
// the old index as it was.
func FuzzIndex(f *testing.F) {
	for _, s := range [][2]string{
		{"foo.go\nsrc/foo_test.go\nxfoo", "f"}, {"a/b/c\nabc", "ab"}, {"привет\nопрос", "пр"},
		{"lib/file.go\nfile_util.go", "file"}, {"\n\n", ""}, {"👨‍👩‍👧/x", "👨‍👩‍👧x y"}, {"a\xffb", "\xff"},
	} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, corpus, query string) {
		cands := strings.Split(corpus, "\n")
		ix := NewIndex(cands, WithMaxCandidates(16))
		check := func(ix *Index) {
			seen := map[int]bool{}
			for i := range ix.Candidates(NewMatcher(query, WithExtended())) {
				if i < 0 || i >= ix.Len() || seen[i] {
					t.Fatalf("index %d of %d, seen %v", i, ix.Len(), seen[i])
				}
				seen[i] = true
			}
			if len(seen) > 16 {
				t.Fatalf("%d candidates over a limit of 16", len(seen))
			}
		}
		check(ix)
		next := ix.Update([]string{query}, []int{0, len(cands)})
		check(next)
		if ix.Len() != len(cands) || (len(cands) > 0 && ix.At(0) != cands[0]) {
			t.Fatal("Update changed the old index")
		}
	})
}

// FuzzMatch: no query and candidate make the matcher panic, report a range
// off a cluster boundary or outside the candidate, or disagree with Score.
func FuzzMatch(f *testing.F) {
	for _, s := range [][2]string{
		{"fb", "FooBar"}, {"file", "src/file.go"}, {"caf", "Café"},
		{"👨‍👩‍👧", "the 👨‍👩‍👧"}, {"a b", "a\r\nb"}, {"^x$ !y 'z", "xyz"}, {"\xff", "a\xffb"},
	} {
		f.Add(s[0], s[1], false)
	}
	f.Fuzz(func(t *testing.T, query, cand string, extended bool) {
		var opts []Option
		if extended {
			opts = append(opts, WithExtended())
		}
		m := NewMatcher(query, opts...)
		mt, ok := m.Match(cand)
		if s, sok := m.Score(cand); s != mt.Score || sok != ok {
			t.Fatalf("Score %+v %v, Match %+v %v", s, sok, mt.Score, ok)
		}
		if !ok || !utf8.ValidString(cand) {
			return
		}
		checkRanges(t, cand, mt.Ranges)
	})
}
