package palette

import (
	"bytes"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// near reports whether a and b differ by less than a millionth.
func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// TestFrecencyDecays: a use scores 1, halved after each half-life, the
// default's or WithHalfLife's, on synctest's clock; a later use adds 1 to
// what is left; an empty ID is not kept.
func TestFrecencyDecays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := NewFrecency()
		f.Record("a")
		if s := f.Score("a"); !near(s, 1) {
			t.Fatalf("a new use scores %v; want 1", s)
		}
		time.Sleep(DefaultHalfLife)
		if s := f.Score("a"); !near(s, 0.5) {
			t.Fatalf("after one half-life: %v; want 0.5", s)
		}
		f.Record("a")
		time.Sleep(DefaultHalfLife)
		if s := f.Score("a"); !near(s, 0.75) {
			t.Fatalf("1.5 after another half-life: %v; want 0.75", s)
		}
		f.Record("")
		if s := f.Score(""); s != 0 {
			t.Fatalf("an empty ID scores %v", s)
		}

		h := NewFrecency(WithHalfLife(time.Hour))
		h.Record("b")
		time.Sleep(2 * time.Hour)
		if s := h.Score("b"); !near(s, 0.25) {
			t.Fatalf("WithHalfLife(1h), after 2h: %v; want 0.25", s)
		}
	})
}

// TestFrecencyMaxEntries: past WithMaxEntries, the lowest scored is
// dropped.
func TestFrecencyMaxEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := NewFrecency(WithMaxEntries(2))
		f.Record("a")
		f.Record("a")
		time.Sleep(time.Hour)
		f.Record("b")
		f.Record("b")
		f.Record("c")
		for id, kept := range map[string]bool{"a": true, "b": true, "c": false} {
			if got := f.Score(id) > 0; got != kept {
				t.Errorf("%s kept %v; want %v", id, got, kept)
			}
		}
	})
}

// TestRecents: an empty query lists the recent hits first, the most used
// first, then the rest in inner's order; a typed one raises a recent hit
// within its grade, and never past a better one. Recents keeps inner's
// name and prefix, and tells it the picker opened.
func TestRecents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := NewFrecency()
		f.Record("dark")
		f.Record("light")
		f.Record("light")
		items := []Item{{ID: "dark", Text: "Dark theme"}, {ID: "high", Text: "High contrast"}, {ID: "light", Text: "Light theme"}, {ID: "sepia", Text: "Sepia"}}
		r := Recents(f, Static("themes", items))
		if r.Name() != "themes" {
			t.Fatalf("name %q", r.Name())
		}
		var listed []Hit
		if err := r.(Discoverer).Discover(t.Context(), func(h Hit) bool { listed = append(listed, h); return true }); err != nil {
			t.Fatal(err)
		}
		slices.SortStableFunc(listed, byScore)
		if got := ids(listed); strings.Join(got, " ") != "light dark high sepia" {
			t.Fatalf("an empty query lists %v; want light dark high sepia", got)
		}

		g := Static("themes", items)
		plain := found(t, g, "e", when.Map{})
		boosted := found(t, r, "e", when.Map{})
		if ids(plain)[0] == "light" {
			t.Fatalf("the fixture: light is already first without recents, %v", ids(plain))
		}
		if !better(scoreOf(boosted, "light"), scoreOf(plain, "light")) || scoreOf(boosted, "light").Grade != scoreOf(plain, "light").Grade {
			t.Fatalf("light: %+v boosted, %+v plain; want a higher value in the same grade", scoreOf(boosted, "light"), scoreOf(plain, "light"))
		}
		// "sepia" is a prefix match for "se"; "light" is not, and stays below
		// it however often it was used.
		for range 50 {
			f.Record("light")
		}
		if got := ids(found(t, r, "se", when.Map{})); len(got) == 0 || got[0] != "sepia" {
			t.Fatalf("se: %v; want sepia first, by its better grade", got)
		}

		// Within one grade, boundary for "tm": "Tab manager" scores 57, "Text
		// mode" 56 and "Terminal mode" 52. Two uses win the near-tie; fifty
		// do not pass a lead of five, more than one consecutive-match bonus
		// (0008-MADR §2).
		modes := []Item{{ID: "tab", Text: "Tab manager"}, {ID: "text", Text: "Text mode"}, {ID: "term", Text: "Terminal mode"}}
		plainModes := found(t, Static("modes", modes), "tm", when.Map{})
		if got := ids(plainModes); strings.Join(got, " ") != "tab text term" || scoreOf(plainModes, "tab").Value-scoreOf(plainModes, "term").Value <= maxRecentBoost {
			t.Fatalf("the fixture: tm ranks %v, %+v", got, plainModes)
		}
		f.Record("text")
		f.Record("text")
		for range 50 {
			f.Record("term")
		}
		if got := ids(found(t, Recents(f, Static("modes", modes)), "tm", when.Map{})); strings.Join(got, " ") != "text tab term" {
			t.Fatalf("tm with recents: %v; want text tab term, the near-tie won and the clear lead kept", got)
		}

		reg, _ := gated(t)
		cmds := Recents(f, FromRegistry(reg))
		if cmds.(Prefixed).Prefix() != ">" {
			t.Fatalf("Recents over FromRegistry has prefix %q; want >", cmds.(Prefixed).Prefix())
		}
		cmds.(opener).opened(when.Map{})
		var some []Hit
		if err := cmds.(Discoverer).Discover(t.Context(), func(h Hit) bool { some = append(some, h); return true }); err != nil || len(some) == 0 {
			t.Fatalf("Recents over FromRegistry discovered %v, %v; want the commands, after the opening", ids(some), err)
		}
	})
}

// scoreOf is the score of the hit with id among hs.
func scoreOf(hs []Hit, id string) fuzzy.Score {
	for _, h := range hs {
		if h.ID == id {
			return h.Score
		}
	}
	return fuzzy.Score{Grade: 255}
}

// byScore orders hits best first, as the picker keeps them.
func byScore(a, b Hit) int {
	switch {
	case better(a.Score, b.Score):
		return -1
	case better(b.Score, a.Score):
		return 1
	}
	return 0
}

// TestFrecencyJSON: a round trip is exact, best scored first; ReadFrom
// refuses an unknown member, an unknown or missing version, more than 1
// MiB, an ID over 1024 bytes, an empty or repeated one and a score not
// above 0, and keeps the store as it was; past WithMaxEntries it keeps the
// best.
func TestFrecencyJSON(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := NewFrecency()
		f.Record("a")
		time.Sleep(time.Minute)
		f.Record("b")
		f.Record("b")
		var first bytes.Buffer
		if _, err := f.WriteTo(&first); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(first.String(), `{"version":1,"entries":[{"id":"b","score":2,"at":"2000-01-01T00:01:00Z"},{"id":"a"`) {
			t.Fatalf("written: %s", first.String())
		}
		back := NewFrecency()
		n, err := back.ReadFrom(bytes.NewReader(first.Bytes()))
		if err != nil || n != int64(first.Len()) {
			t.Fatalf("ReadFrom: %d, %v; want %d bytes", n, err, first.Len())
		}
		var second bytes.Buffer
		if _, err := back.WriteTo(&second); err != nil {
			t.Fatal(err)
		}
		if second.String() != first.String() {
			t.Fatalf("round trip:\n%s\n%s", first.String(), second.String())
		}

		long := strings.Repeat("x", maxRecentID+1)
		for name, in := range map[string]string{
			"an unknown member":  `{"version":1,"entries":[],"extra":true}`,
			"an unknown version": `{"version":2,"entries":[]}`,
			"no version":         `{"entries":[]}`,
			"an empty id":        `{"version":1,"entries":[{"id":"","score":1,"at":"2000-01-01T00:00:00Z"}]}`,
			"a long id":          `{"version":1,"entries":[{"id":"` + long + `","score":1,"at":"2000-01-01T00:00:00Z"}]}`,
			"a repeated id":      `{"version":1,"entries":[{"id":"a","score":1},{"id":"a","score":2}]}`,
			"a zero score":       `{"version":1,"entries":[{"id":"a","score":0}]}`,
			"a negative score":   `{"version":1,"entries":[{"id":"a","score":-1}]}`,
			"over 1 MiB":         `{"version":1,"entries":[]}` + strings.Repeat(" ", maxRecentsBytes),
		} {
			if _, err := back.ReadFrom(strings.NewReader(in)); err == nil {
				t.Errorf("%s: read", name)
			}
		}
		for _, in := range []string{`{"version":2,"entries":[]}`, `{"entries":[]}`} {
			if _, err := back.ReadFrom(strings.NewReader(in)); !errors.Is(err, ErrRecentsVersion) {
				t.Errorf("%s: %v; want ErrRecentsVersion", in, err)
			}
		}
		if !near(back.Score("b"), 2) {
			t.Fatalf("after refused reads, b scores %v; want 2, as read", back.Score("b"))
		}

		small := NewFrecency(WithMaxEntries(1))
		if _, err := small.ReadFrom(bytes.NewReader(first.Bytes())); err != nil {
			t.Fatal(err)
		}
		if small.Score("b") == 0 || small.Score("a") != 0 {
			t.Fatalf("WithMaxEntries(1) kept a %v, b %v; want b alone", small.Score("a"), small.Score("b"))
		}
	})
}
