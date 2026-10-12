package fuzzy

import (
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// termKind is how an extended term matches.
type termKind uint8

const (
	termFuzzy  termKind = iota // a subsequence, aligned
	termExact                  // 'term: a contiguous run
	termPrefix                 // ^term
	termSuffix                 // term$
	termEqual                  // ^term$
)

// term is one compiled part of a query: the whole query, or one of the
// extended syntax's space-separated terms.
type term struct {
	kind    termKind
	inverse bool // !term: the candidate must not match
	units   []unit
}

// Matcher is a compiled query. It is immutable after NewMatcher and safe
// for concurrent use.
type Matcher struct {
	query      string
	terms      []term
	sensitive  bool
	norm       Normalizer
	delimiters string
	maxAlign   int
	// clusters gives each multi-rune cluster of the query, folded, its key.
	clusters map[string]int64
	// scratch holds a *scratch per goroutine at work, so that matching an
	// ASCII candidate allocates nothing once warm.
	scratch sync.Pool
}

// scratch is one match's working memory.
type scratch struct {
	units  []unit
	buf    []byte
	pos    []int
	fwd    []int
	bwd    []int
	cells  []int // the alignment's scores
	runs   []int // the bonus each alignment's run of consecutive clusters began with
	from   []int // where each alignment came from
	ranges []Range
}

// NewMatcher compiles query. Only its first MaxQuery grapheme clusters are
// matched. An empty query, or an extended one with no term, matches every
// candidate with the zero Score and no ranges.
func NewMatcher(query string, opts ...Option) *Matcher {
	cfg := config{delimiters: defaultDelimiters, maxAlign: DefaultMaxAlign}
	for _, o := range opts {
		o.apply(&cfg)
	}
	if cfg.maxAlign < 1 {
		cfg.maxAlign = DefaultMaxAlign
	}
	query = firstClusters(query, MaxQuery)
	m := &Matcher{
		query: query, norm: cfg.norm, delimiters: cfg.delimiters, maxAlign: cfg.maxAlign,
		clusters: map[string]int64{},
	}
	switch cfg.mode {
	case Sensitive:
		m.sensitive = true
	case Smart:
		m.sensitive = strings.IndexFunc(query, unicode.IsUpper) >= 0
	}
	m.scratch.New = func() any { return new(scratch) }
	var buf []byte
	compile := func(text string, kind termKind, inverse bool) {
		if text == "" {
			return
		}
		m.terms = append(m.terms, term{kind: kind, inverse: inverse, units: m.prepare(nil, text, &buf, true, false)})
	}
	if !cfg.extended {
		compile(query, termFuzzy, false)
		return m
	}
	for f := range strings.FieldsSeq(query) {
		inverse := strings.HasPrefix(f, "!")
		if inverse {
			f = f[1:]
		}
		kind := termFuzzy
		switch {
		case strings.HasPrefix(f, "'"):
			kind, f = termExact, f[1:]
		case strings.HasPrefix(f, "^") && len(f) > 2 && strings.HasSuffix(f, "$"):
			kind, f = termEqual, f[1:len(f)-1]
		case strings.HasPrefix(f, "^"):
			kind, f = termPrefix, f[1:]
		case len(f) > 1 && strings.HasSuffix(f, "$"):
			kind, f = termSuffix, f[:len(f)-1]
		case inverse:
			kind = termExact // fzf: !term is an inverse exact match
		}
		compile(f, kind, inverse)
	}
	return m
}

// firstClusters is s cut after its first n grapheme clusters.
func firstClusters(s string, n int) string {
	if len(s) <= n {
		return s
	}
	i := 0
	for range n {
		if i >= len(s) {
			return s
		}
		c, _ := ansi.FirstGraphemeCluster(s[i:], ansi.GraphemeWidth)
		if c == "" {
			c = s[i : i+1]
		}
		i += len(c)
	}
	return s[:i]
}

// Query is the query the matcher matches: the one NewMatcher was given,
// cut to MaxQuery grapheme clusters.
func (m *Matcher) Query() string { return m.query }

// Score reports whether candidate matches, and how well, without the
// ranges. On an ASCII candidate it allocates nothing.
func (m *Matcher) Score(candidate string) (Score, bool) {
	s, ok, _ := m.run(candidate, false, false)
	return s, ok
}

// Match is Score with the ranges that matched, for the candidates that are
// drawn.
func (m *Matcher) Match(candidate string) (Match, bool) {
	s, ok, r := m.run(candidate, true, false)
	return Match{Score: s, Ranges: r}, ok
}

// run matches candidate; with ranges it also returns the matched ranges,
// and slow walks an ASCII candidate's clusters the general way.
func (m *Matcher) run(candidate string, ranges, slow bool) (Score, bool, []Range) {
	if len(m.terms) == 0 {
		return Score{}, true, nil
	}
	sc, ok := m.scratch.Get().(*scratch)
	if !ok {
		sc = new(scratch)
	}
	defer m.scratch.Put(sc)
	sc.units = m.prepare(sc.units[:0], candidate, &sc.buf, false, slow)
	us := sc.units
	var total Score
	sc.ranges = sc.ranges[:0]
	for i := range m.terms {
		t := &m.terms[i]
		s, ok := m.matchTerm(t, us, sc)
		if t.inverse {
			if ok {
				return Score{}, false, nil
			}
			continue
		}
		if !ok {
			return Score{}, false, nil
		}
		total.Grade = max(total.Grade, s.Grade)
		total.Value += s.Value
		if ranges {
			sc.ranges = appendRanges(sc.ranges, us, sc.pos)
		}
	}
	if !ranges {
		return total, true, nil
	}
	return total, true, mergeRanges(sc.ranges)
}

// appendRanges adds the byte ranges of the clusters at positions pos, runs
// of adjacent clusters joined.
func appendRanges(out []Range, us []unit, pos []int) []Range {
	for i := 0; i < len(pos); {
		j := i
		for j+1 < len(pos) && pos[j+1] == pos[j]+1 {
			j++
		}
		out = append(out, Range{Start: us[pos[i]].start, End: us[pos[j]].end})
		i = j + 1
	}
	return out
}

// mergeRanges sorts ranges and joins those that overlap or touch, into a
// new slice.
func mergeRanges(rs []Range) []Range {
	if len(rs) == 0 {
		return nil
	}
	out := slices.Clone(rs)
	slices.SortFunc(out, func(a, b Range) int { return a.Start - b.Start })
	n := 0
	for _, r := range out[1:] {
		if r.Start <= out[n].End {
			out[n].End = max(out[n].End, r.End)
			continue
		}
		n++
		out[n] = r
	}
	return out[:n+1]
}

// matchTerm matches one term, leaving its positions in sc.pos.
func (m *Matcher) matchTerm(t *term, us []unit, sc *scratch) (Score, bool) {
	q := t.units
	if len(q) > len(us) {
		return Score{}, false
	}
	switch t.kind {
	case termExact:
		return m.contiguous(q, us, sc, 0, len(us)-len(q))
	case termPrefix:
		return m.contiguous(q, us, sc, 0, 0)
	case termSuffix:
		return m.contiguous(q, us, sc, len(us)-len(q), len(us)-len(q))
	case termEqual:
		if len(q) != len(us) {
			return Score{}, false
		}
		return m.contiguous(q, us, sc, 0, 0)
	}
	return m.fuzzy(q, us, sc)
}

// contiguous finds q as a run starting at from to last, the best-scoring
// one, with its grade.
func (m *Matcher) contiguous(q, us []unit, sc *scratch, from, last int) (Score, bool) {
	bestV, bestAt := 0, -1
	for at := from; at <= last; at++ {
		if !equalAt(us, at, q) {
			continue
		}
		sc.pos = runOf(sc.pos[:0], at, len(q))
		if v := scorePositions(us, sc.pos); bestAt < 0 || v > bestV {
			bestV, bestAt = v, at
		}
	}
	if bestAt < 0 {
		return Score{}, false
	}
	sc.pos = runOf(sc.pos[:0], bestAt, len(q))
	return Score{Grade: grade(q, us, sc.pos), Value: bestV}, true
}

// runOf appends the positions at, at+1, … of n clusters.
func runOf(pos []int, at, n int) []int {
	for i := range n {
		pos = append(pos, at+i)
	}
	return pos
}

// equalAt reports whether q's keys are us's, starting at at.
func equalAt(us []unit, at int, q []unit) bool {
	if at < 0 || at+len(q) > len(us) {
		return false
	}
	for i := range q {
		if us[at+i].key != q[i].key {
			return false
		}
	}
	return true
}

// fuzzy aligns q as a subsequence of us: exactly over the stretch it can
// lie in, or greedily when that is longer than maxAlign.
func (m *Matcher) fuzzy(q, us []unit, sc *scratch) (Score, bool) {
	// The forward greedy match gives each query cluster its earliest
	// position, and the backward one its latest.
	sc.fwd = sc.fwd[:0]
	j := 0
	for i := range q {
		for j < len(us) && us[j].key != q[i].key {
			j++
		}
		if j == len(us) {
			return Score{}, false
		}
		sc.fwd = append(sc.fwd, j)
		j++
	}
	sc.bwd = slices.Grow(sc.bwd[:0], len(q))[:len(q)]
	j = len(us) - 1
	for i := len(q) - 1; i >= 0; i-- {
		for us[j].key != q[i].key {
			j--
		}
		sc.bwd[i] = j
		j--
	}
	lo, hi := sc.fwd[0], sc.bwd[len(q)-1]
	if hi-lo+1 > m.maxAlign {
		m.greedy(q, us, sc)
	} else {
		align(q, us, sc, lo, hi)
	}
	return Score{Grade: grade(q, us, sc.pos), Value: scorePositions(us, sc.pos)}, true
}

// greedy is fzf's V1 alignment: the forward match's end, then the tightest
// start from there backwards, then forward again from that start.
func (m *Matcher) greedy(q, us []unit, sc *scratch) {
	j := sc.fwd[len(q)-1]
	for i, u := range slices.Backward(q) {
		for us[j].key != u.key {
			j--
		}
		if i > 0 {
			j--
		}
	}
	sc.pos = sc.pos[:0]
	for i := range q {
		for us[j].key != q[i].key {
			j++
		}
		sc.pos = append(sc.pos, j)
		j++
	}
}

// negInf is below every reachable score.
const negInf = -1 << 30

// align is the exact alignment over us[lo:hi+1], a Smith-Waterman-style
// dynamic programme with fzf's bonuses and affine gaps, leaving the best
// positions in sc.pos. Query cluster i is only placed between its forward
// and backward greedy positions.
func align(q, us []unit, sc *scratch, lo, hi int) {
	w := hi - lo + 1
	n := len(q) * w
	sc.cells = slices.Grow(sc.cells[:0], n)[:n]
	sc.runs = slices.Grow(sc.runs[:0], n)[:n]
	sc.from = slices.Grow(sc.from[:0], n)[:n]
	for i := range q {
		row, prev := i*w, (i-1)*w
		gapBest, gapFrom := negInf, -1 // best of row i-1 at k ≤ j-2, with its gap
		for j := lo; j <= hi; j++ {
			c := j - lo
			if i > 0 {
				if gapBest > negInf {
					gapBest += scoreGapExtension
				}
				if c >= 2 {
					if v := sc.cells[prev+c-2]; v > negInf && v+scoreGapStart > gapBest {
						gapBest, gapFrom = v+scoreGapStart, j-2
					}
				}
			}
			sc.cells[row+c] = negInf
			if j < sc.fwd[i] || j > sc.bwd[i] || us[j].key != q[i].key {
				continue
			}
			b := us[j].bonus
			if i == 0 {
				sc.cells[row+c], sc.runs[row+c], sc.from[row+c] = scoreMatch+b*bonusFirstCharMultiple, b, -1
				continue
			}
			best, run, from := negInf, b, -1
			if c > 0 && sc.cells[prev+c-1] > negInf {
				gain, r := consecutive(b, sc.runs[prev+c-1])
				best, run, from = sc.cells[prev+c-1]+scoreMatch+gain, r, j-1
			}
			if gapBest > negInf {
				if s := gapBest + scoreMatch + b; s > best {
					best, run, from = s, b, gapFrom
				}
			}
			sc.cells[row+c], sc.runs[row+c], sc.from[row+c] = best, run, from
		}
	}
	last := (len(q) - 1) * w
	at := -1
	for c := range w {
		if sc.cells[last+c] > negInf && (at < 0 || sc.cells[last+c] > sc.cells[last+at]) {
			at = c
		}
	}
	sc.pos = slices.Grow(sc.pos[:0], len(q))[:len(q)]
	j := lo + at
	for i := len(q) - 1; i >= 0; i-- {
		sc.pos[i] = j
		j = sc.from[i*w+j-lo]
	}
}

// consecutive is the bonus for a cluster with bonus b that follows the
// previous query cluster directly, in a run that began with bonus run, and
// the bonus the run carries on with: fzf's rule, where a stronger boundary
// starts a new run.
func consecutive(b, run int) (gain, carried int) {
	if b >= bonusBoundary && b > run {
		return b, b
	}
	return max(b, bonusConsecutive, run), run
}

// scorePositions is the value of placing the query at pos, by the same
// rules align uses, so that the greedy alignment and the exact one are
// scored alike.
func scorePositions(us []unit, pos []int) int {
	v, run := 0, 0
	for i, p := range pos {
		b := us[p].bonus
		switch {
		case i == 0:
			v += scoreMatch + b*bonusFirstCharMultiple
			run = b
		case p == pos[i-1]+1:
			gain, r := consecutive(b, run)
			v += scoreMatch + gain
			run = r
		default:
			v += scoreGapStart + (p-pos[i-1]-2)*scoreGapExtension + scoreMatch + b
			run = b
		}
	}
	return v
}

// grade is the grade of q placed at pos in us.
func grade(q, us []unit, pos []int) Grade {
	elem := lastElement(us)
	stem := stemEnd(us, elem)
	switch {
	case len(q) == len(us) && equalAt(us, 0, q),
		len(q) == len(us)-elem && equalAt(us, elem, q),
		len(q) == stem-elem && equalAt(us, elem, q):
		return Exact
	case equalAt(us, 0, q), equalAt(us, elem, q):
		return Prefix
	case wholeWord(q, us):
		return Boundary
	}
	for _, p := range pos {
		if us[p].bonus < bonusCamel123 {
			return Subsequence
		}
	}
	return Boundary
}

// pathSep reports whether u is a path separator.
func pathSep(u unit) bool { return u.key == '/' || u.key == '\\' }

// lastElement is where the last path element of us starts.
func lastElement(us []unit) int {
	for i, u := range slices.Backward(us) {
		if pathSep(u) {
			return i + 1
		}
	}
	return 0
}

// stemEnd is where the element starting at elem ends, less its last
// extension: before its last dot, unless that dot starts it.
func stemEnd(us []unit, elem int) int {
	for i := len(us) - 1; i > elem; i-- {
		if us[i].key == '.' {
			return i
		}
	}
	return len(us)
}

// wholeWord reports whether q equals a whole word of us, a maximal run of
// word clusters, or a whole path segment.
func wholeWord(q, us []unit) bool {
	word := func(u unit) bool { return u.class > charDelimiter }
	for _, inside := range []func(unit) bool{word, func(u unit) bool { return !pathSep(u) }} {
		for at := 0; at < len(us); {
			if !inside(us[at]) {
				at++
				continue
			}
			end := at
			for end < len(us) && inside(us[end]) {
				end++
			}
			if end-at == len(q) && equalAt(us, at, q) {
				return true
			}
			at = end
		}
	}
	return false
}
