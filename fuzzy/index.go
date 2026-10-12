package fuzzy

import (
	"iter"
	"math"
	"slices"
	"strings"
	"unicode/utf8"
)

// The index's defaults (docs/decisions/0008-MADR-command-palette.md §3).
const (
	// DefaultMinOverlap is the share of a long query's trigrams a candidate
	// must have.
	DefaultMinOverlap = 0.3
	// DefaultMaxCandidates is how many candidates Candidates yields at
	// most.
	DefaultMaxCandidates = 2000
)

// separators are where a word of a path or a title starts, for the
// one-cluster strategy, and what a trigram treats as a space, so that the
// start and end of each path element and word earn their padded trigrams.
const separators = `/\ ._-:,;|`

// IndexOption configures NewIndex.
type IndexOption interface{ applyIndex(*indexConfig) }

type indexOptionFunc func(*indexConfig)

func (f indexOptionFunc) applyIndex(c *indexConfig) { f(c) }

type indexConfig struct {
	minOverlap    float64
	maxCandidates int
}

// WithMinOverlap sets the share, in (0, 1], of a query's padded trigrams a
// candidate must have when the query has four or more clusters. A value
// outside that range is DefaultMinOverlap.
func WithMinOverlap(f float64) IndexOption {
	return indexOptionFunc(func(c *indexConfig) { c.minOverlap = f })
}

// WithMaxCandidates sets how many candidates Candidates yields at most. n
// below 1 is DefaultMaxCandidates.
func WithMaxCandidates(n int) IndexOption {
	return indexOptionFunc(func(c *indexConfig) { c.maxCandidates = n })
}

// Index narrows a large candidate set before scoring, as toad's fuzzy index
// does (docs/decisions/0008-MADR-command-palette.md §3): a one-cluster
// query keeps the candidates that start with it or have it after a
// separator; two or three keep those that contain all of them; four or
// more keep those that share at least the minimum overlap of the query's
// padded trigrams, most shared first.
//
// The index is lossy on purpose: a sparse subsequence of a long candidate
// can share no trigram with it. A candidate that equals the query, or whose
// path element or word starts with it, contains every one of its trigrams
// and is yielded first, so it is dropped only when more than the maximum
// candidates do the same. Matching is under simple case folding; a
// Matcher's Normalizer is not applied.
//
// An Index is immutable and safe for concurrent use; Update makes the next
// one.
type Index struct {
	cands  []string
	folded []string // each candidate, case folded rune by rune
	masks  []uint64 // the folded ASCII bytes each candidate has
	starts map[rune][]int32
	grams  map[int64][]int32
	cfg    indexConfig
}

// NewIndex indexes candidates, which it keeps without copying their text.
func NewIndex(candidates []string, opts ...IndexOption) *Index {
	cfg := indexConfig{minOverlap: DefaultMinOverlap, maxCandidates: DefaultMaxCandidates}
	for _, o := range opts {
		o.applyIndex(&cfg)
	}
	if !(cfg.minOverlap > 0 && cfg.minOverlap <= 1) {
		cfg.minOverlap = DefaultMinOverlap
	}
	if cfg.maxCandidates < 1 {
		cfg.maxCandidates = DefaultMaxCandidates
	}
	return build(slices.Clone(candidates), cfg)
}

func build(cands []string, cfg indexConfig) *Index {
	if len(cands) > math.MaxInt32 {
		panic("fuzzy: an Index holds at most MaxInt32 candidates")
	}
	ix := &Index{
		cands: cands, folded: make([]string, len(cands)), masks: make([]uint64, len(cands)),
		starts: map[rune][]int32{}, grams: map[int64][]int32{}, cfg: cfg,
	}
	seen := map[int64]bool{}
	var id int32 = -1 // candidate i's id, counted beside it
	for i, c := range cands {
		id++
		f := foldString(c)
		ix.folded[i] = f
		ix.masks[i] = mask(f)
		at := true
		for _, r := range f {
			if at && !isSeparator(r) {
				ix.starts[r] = appendOnce(ix.starts[r], id)
			}
			at = isSeparator(r)
		}
		clear(seen)
		for g := range trigrams(f) {
			if !seen[g] {
				seen[g] = true
				ix.grams[g] = append(ix.grams[g], id)
			}
		}
	}
	return ix
}

// appendOnce appends id unless it is already last.
func appendOnce(ids []int32, id int32) []int32 {
	if n := len(ids); n > 0 && ids[n-1] == id {
		return ids
	}
	return append(ids, id)
}

// Len is the number of candidates.
func (ix *Index) Len() int { return len(ix.cands) }

// At is candidate i.
func (ix *Index) At(i int) string { return ix.cands[i] }

// Update returns a new Index of this one's candidates less those at the
// indexes in remove, then add, with the same options. This one is left as
// it was, so a search may go on reading it. The new index is built afresh,
// in time proportional to its size.
func (ix *Index) Update(add []string, remove []int) *Index {
	drop := make(map[int]bool, len(remove))
	for _, i := range remove {
		drop[i] = true
	}
	next := make([]string, 0, len(ix.cands)-len(drop)+len(add))
	for i, c := range ix.cands {
		if !drop[i] {
			next = append(next, c)
		}
	}
	return build(append(next, add...), ix.cfg)
}

// Candidates yields the indexes of the candidates worth scoring for q, at
// most the maximum, each once. A query with no term yields every
// candidate, up to the maximum. Of an extended query, the longest term
// that is not negated chooses them.
func (ix *Index) Candidates(q *Matcher) iter.Seq[int] {
	return func(yield func(int) bool) {
		text := q.indexTerm()
		limit := ix.cfg.maxCandidates
		var ids []int
		switch n := clusterCount(text); {
		case n == 0:
			ids = ix.all(limit)
		case n == 1:
			ids = ix.byStart(foldString(text), limit)
		case n <= 3:
			ids = ix.byContents(foldString(text), limit)
		default:
			ids = ix.byTrigrams(foldString(text), limit)
		}
		for _, i := range ids {
			if !yield(i) {
				return
			}
		}
	}
}

// indexTerm is the text of the query's longest term that is not negated.
func (m *Matcher) indexTerm() string {
	best := ""
	for _, t := range m.terms {
		if !t.inverse && len(t.units) > clusterCount(best) {
			best = t.text
		}
	}
	return best
}

// clusterCount is the number of grapheme clusters in s.
func clusterCount(s string) int {
	n := 0
	for i := 0; i < len(s); n++ {
		i += len(firstClusters(s[i:], 1))
	}
	return n
}

func (ix *Index) all(limit int) []int {
	ids := make([]int, min(limit, len(ix.cands)))
	for i := range ids {
		ids[i] = i
	}
	return ids
}

// byStart is the candidates where the query's first rune starts the
// candidate or follows a separator, then whose folded text holds all of
// it, those that start with it first.
func (ix *Index) byStart(q string, limit int) []int {
	r, _ := utf8.DecodeRuneInString(q)
	var first, rest []int
	for _, id := range ix.starts[r] {
		i := int(id)
		if !strings.Contains(ix.folded[i], q) {
			continue
		}
		if strings.HasPrefix(ix.folded[i], q) {
			first = append(first, i)
		} else {
			rest = append(rest, i)
		}
	}
	return capped(append(first, rest...), limit)
}

// byContents is the candidates whose folded text holds every rune of q:
// by the ASCII mask, then, for a rune beyond ASCII, by the text. Those
// that hold q in one piece come first, so that the limit, when it bites,
// keeps the likelier matches.
func (ix *Index) byContents(q string, limit int) []int {
	want := mask(q)
	var wide []rune
	for _, r := range q {
		if r >= utf8.RuneSelf {
			wide = append(wide, r)
		}
	}
	var whole, scattered []int
	for i, m := range ix.masks {
		if m&want != want || !containsAll(ix.folded[i], wide) {
			continue
		}
		if strings.Contains(ix.folded[i], q) {
			whole = append(whole, i)
			if len(whole) == limit {
				break
			}
		} else {
			scattered = append(scattered, i)
		}
	}
	return capped(append(whole, scattered...), limit)
}

func containsAll(s string, runes []rune) bool {
	for _, r := range runes {
		if !strings.ContainsRune(s, r) {
			return false
		}
	}
	return true
}

// byTrigrams is the candidates sharing at least the minimum overlap of
// q's padded trigrams, most shared first, then in index order.
func (ix *Index) byTrigrams(q string, limit int) []int {
	var qgrams []int64
	for g := range trigrams(q) {
		if !slices.Contains(qgrams, g) {
			qgrams = append(qgrams, g)
		}
	}
	need := int(math.Ceil(ix.cfg.minOverlap * float64(len(qgrams))))
	count := make([]uint16, len(ix.cands))
	var hit []int
	for _, g := range qgrams {
		for _, id := range ix.grams[g] {
			if count[id] == 0 {
				hit = append(hit, int(id))
			}
			count[id]++
		}
	}
	ids := slices.DeleteFunc(hit, func(i int) bool { return int(count[i]) < need })
	slices.SortStableFunc(ids, func(a, b int) int {
		if count[a] != count[b] {
			return int(count[b]) - int(count[a])
		}
		return a - b
	})
	return capped(ids, limit)
}

func capped(ids []int, limit int) []int {
	if len(ids) > limit {
		return ids[:limit]
	}
	return ids
}

// foldString is s with each rune case folded as the matcher folds it.
func foldString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(fold(r))
	}
	return b.String()
}

func isSeparator(r rune) bool { return strings.ContainsRune(separators, r) }

// mask is the set of folded ASCII bytes in s: letters and digits a bit
// each, other bytes sharing the rest. A shared bit only lets more
// candidates through.
func mask(s string) uint64 {
	var m uint64
	for i := range len(s) {
		c := s[i]
		switch {
		case c >= utf8.RuneSelf:
			continue
		case 'A' <= c && c <= 'Z':
			m |= 1 << (c - 'A')
		case '0' <= c && c <= '9':
			m |= 1 << (26 + c - '0')
		default:
			m |= 1 << (36 + c%28)
		}
	}
	return m
}

// trigrams yields the trigrams of s padded with spaces at each end, each
// separator read as a space, each trigram three runes packed into one key.
// A trigram of spaces alone is not yielded.
func trigrams(s string) iter.Seq[int64] {
	return func(yield func(int64) bool) {
		a, b := ' ', ' ' // the padding before s
		push := func(c rune) bool {
			if isSeparator(c) {
				c = ' '
			}
			blank := a == ' ' && b == ' ' && c == ' '
			g := int64(a)<<42 | int64(b)<<21 | int64(c) // a rune is at most 21 bits
			a, b = b, c
			return blank || yield(g)
		}
		for _, r := range s {
			if !push(r) {
				return
			}
		}
		push(' ') // the padding after s
	}
}
