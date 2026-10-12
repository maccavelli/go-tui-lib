package fuzzy

import (
	"testing"
	"unicode/utf8"
)

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
