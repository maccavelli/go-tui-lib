package fuzzy

import (
	"context"
	"iter"
	"slices"
)

// Ranked is one ranked item and its score.
type Ranked[T any] struct {
	Item  T
	Score Score
}

// entry is a ranked item with what breaks its ties.
type entry[T any] struct {
	Ranked[T]
	value int // the score's value plus the bounded boost
	size  int // the key's length in bytes: shorter ranks first
	index int // its place in the input: earlier ranks first
}

// before is the rank order, which is total: grade, then value with the
// boost, then the shorter key, then the earlier item.
func before[T any](a, b *entry[T]) bool {
	switch {
	case a.Score.Grade != b.Score.Grade:
		return a.Score.Grade < b.Score.Grade
	case a.value != b.value:
		return a.value > b.value
	case a.size != b.size:
		return a.size < b.size
	}
	return a.index < b.index
}

// maxBoost bounds a boost to one consecutive-match bonus, so that recency
// breaks near-ties without overturning a better match.
const maxBoost = bonusConsecutive

// checkEvery is how many items RankContext scores between looks at its
// context.
const checkEvery = 64

// Rank scores items by key and returns the best limit, best first, stable
// on ties. A limit below 1 keeps every match.
func (m *Matcher) Rank[T any](items iter.Seq[T], key func(T) string, limit int) []Ranked[T] {
	out, err := m.RankContext(context.Background(), items, key, limit, nil)
	if err != nil {
		return nil // context.Background is never done
	}
	return out
}

// RankContext is Rank with boost, which adds to each match's value within
// its grade, bounded to ±4 (one consecutive-match bonus), so that a recent
// or frequent item wins a near-tie and never a clearly better match. It
// stops with ctx.Err() soon after ctx is done. boost may be nil.
func (m *Matcher) RankContext[T any](ctx context.Context, items iter.Seq[T], key func(T) string, limit int, boost func(T) int) ([]Ranked[T], error) {
	var kept []*entry[T]
	index := 0
	for item := range items {
		if index%checkEvery == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		k := key(item)
		s, ok := m.Score(k)
		index++
		if !ok {
			continue
		}
		e := &entry[T]{Item: item, Score: s, value: s.Value, size: len(k), index: index}
		if boost != nil {
			e.value += min(max(boost(item), -maxBoost), maxBoost)
		}
		kept = keep(kept, e, limit)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	slices.SortFunc(kept, func(a, b *entry[T]) int {
		if before(a, b) {
			return -1
		}
		return 1
	})
	out := make([]Ranked[T], len(kept))
	for i, e := range kept {
		out[i] = e.Ranked
	}
	return out, nil
}

// keep adds e to kept, a max-heap on the rank order whose root is the worst
// entry kept, holding at most limit entries when limit is at least 1.
func keep[T any](kept []*entry[T], e *entry[T], limit int) []*entry[T] {
	if limit < 1 || len(kept) < limit {
		kept = append(kept, e)
		for i := len(kept) - 1; i > 0; {
			p := (i - 1) / 2
			if !before(kept[p], kept[i]) {
				break
			}
			kept[p], kept[i] = kept[i], kept[p]
			i = p
		}
		return kept
	}
	if !before(e, kept[0]) {
		return kept
	}
	kept[0] = e
	for i := 0; ; {
		worst, l, r := i, 2*i+1, 2*i+2
		if l < len(kept) && before(kept[worst], kept[l]) {
			worst = l
		}
		if r < len(kept) && before(kept[worst], kept[r]) {
			worst = r
		}
		if worst == i {
			return kept
		}
		kept[i], kept[worst] = kept[worst], kept[i]
		i = worst
	}
}
