package lineinput

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// whole is s cut into clusters from its start, as set does.
func whole(s string, m ansi.Method) []string {
	var out []string
	for s != "" {
		c, _ := ansi.FirstGraphemeCluster(s, m)
		out = append(out, c)
		s = s[len(c):]
	}
	return out
}

// FuzzSplice: after any insert, deletion or transpose, the clusters kept are
// the text cut again from its start, each kept width is the cluster's, the
// size is the text's, and the edit marks are at or before the first cluster
// that changed and within the clusters at the end that did not
// (0023-MADR A12 item 4.2).
func FuzzSplice(f *testing.F) {
	f.Add("👨\u200d👩\u200d👧 e\u0301\u0301 🇺🇸🇫🇷x\n\u200d👩", []byte{0, 1, 2, 3, 4, 5, 6}, "\u200d")
	f.Add("ab\ncd", []byte{3, 2, 9, 1}, "\u0301")
	f.Add("🇺🇸🇫", []byte{2, 0, 2}, "🇷")
	f.Fuzz(func(t *testing.T, text string, ops []byte, ins string) {
		if !utf8.ValidString(text) || !utf8.ValidString(ins) {
			return
		}
		// Each edit's checks read the whole text: bounded, so one input stays
		// within the fuzzer's time for it. The Line's own cost is
		// TestSpliceFlagRun's.
		if len(text) > 512 || len(ops) > 64 || len(ins) > 64 {
			return
		}
		for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
			l := NewRunes(0, m)
			l.SetText(text)
			for _, op := range ops {
				before := slices.Clone(l.Clusters())
				l.Edited()
				l.SetPos(int(op) % (len(l.Clusters()) + 1))
				edit(l, op, ins)
				checkLine(t, m, l, before)
			}
		}
	})
}

// edit is one of FuzzSplice's edits, by op.
func edit(l *Line, op byte, ins string) {
	switch op % 5 {
	case 0:
		l.Insert(ins)
	case 1:
		l.Do(DeleteBack)
	case 2:
		l.Do(DeleteForward)
	case 3:
		l.Do(DeleteWordBack)
	case 4:
		l.Do(Transpose)
	}
}

// checkLine fails t unless l, edited from before, holds what FuzzSplice says.
func checkLine(t *testing.T, m ansi.Method, l *Line, before []string) {
	t.Helper()
	got, want := l.Clusters(), whole(l.Text(), m)
	if !slices.Equal(got, want) {
		t.Fatalf("%v: clusters %q, want %q", m, got, want)
	}
	if len(l.widths) != len(got) {
		t.Fatalf("%v: %d widths for %d clusters", m, len(l.widths), len(got))
	}
	for i, c := range got {
		if l.widths[i] != m.StringWidth(c) {
			t.Fatalf("%v: width of %q kept as %d, measures %d", m, c, l.widths[i], m.StringWidth(c))
		}
	}
	if n := utf8.RuneCountInString(l.Text()); l.used != n {
		t.Fatalf("%v: used %d, text %d runes", m, l.used, n)
	}
	first := 0
	for first < min(len(before), len(got)) && before[first] == got[first] {
		first++
	}
	last := 0
	for last < min(len(before), len(got)) && before[len(before)-1-last] == got[len(got)-1-last] {
		last++
	}
	from := func() string { return strings.Join(before, "|") + " to " + strings.Join(got, "|") }
	if l.tail > last {
		t.Fatalf("%v: tail %d past the unchanged end, %d: %q", m, l.tail, last, from())
	}
	if unchanged := first == len(before) && first == len(got); l.edit > first && !unchanged {
		t.Fatalf("%v: edit mark %d after the first change, %d: %q", m, l.edit, first, from())
	}
}

// TestSpliceFlagRun: a flag letter inserted at the start of a run of them
// pairs every letter after it anew, and the splice cuts again only the last
// cluster with each one it takes in, so the cuts it makes grow with the
// run, not with its square, which cutting the whole window again at each
// step would (FuzzSplice found a run of about 2,500 letters taking 18 s).
func TestSpliceFlagRun(t *testing.T) {
	const n = 2000
	l := NewRunes(0, ansi.GraphemeWidth)
	l.SetText(strings.Repeat("\U0001F1F7", n))
	l.SetPos(0)
	cuts := 0
	defer func(f func(string, ansi.Method) (string, int)) { firstCluster = f }(firstCluster)
	counted := firstCluster
	firstCluster = func(s string, m ansi.Method) (string, int) {
		cuts++
		return counted(s, m)
	}
	l.Insert("\U0001F1FA")
	if cuts > 4*n {
		t.Errorf("one insert into a run of %d letters cut %d clusters; want at most %d", n, cuts, 4*n)
	}
	got, want := l.Clusters(), whole(l.Text(), ansi.GraphemeWidth)
	if i := slices.IndexFunc(got, func(c string) bool { return !slices.Contains(want, c) }); !slices.Equal(got, want) {
		t.Fatalf("%d clusters, want %d; the first not cut as the whole text cuts it is %d", len(got), len(want), i)
	}
}

// TestSpliceJoinsPastOne: deleting the line break before a lone ZWJ joins
// the cluster before it, the ZWJ and the emoji after it into one, which a
// window of one cluster on each side would leave as two.
func TestSpliceJoinsPastOne(t *testing.T) {
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		l := NewRunes(0, m)
		l.SetText("👨\n\u200d👩")
		before := slices.Clone(l.Clusters())
		l.SetPos(2)
		l.Do(DeleteBack)
		if got, want := l.Clusters(), whole(l.Text(), m); !slices.Equal(got, want) {
			t.Fatalf("%v: from %q, clusters %q, want %q", m, before, got, want)
		}
	}
}
