package frame

import (
	"iter"
	"time"
)

// Queue holds items, each with the time it came, for a Pacer to release in
// order. Its zero value is an empty queue. It is used on one goroutine.
type Queue[T any] struct {
	ring []queued[T] // a ring buffer of the items, oldest at head
	head int
	n    int
}

// queued is one item and the time it came.
type queued[T any] struct {
	v  T
	at time.Time
}

// Push adds v, which came at at, behind the items held.
func (q *Queue[T]) Push(v T, at time.Time) {
	if q.n == len(q.ring) {
		q.grow()
	}
	q.ring[(q.head+q.n)%len(q.ring)] = queued[T]{v, at}
	q.n++
}

// grow doubles the ring, at least to 8, keeping the items in order.
func (q *Queue[T]) grow() {
	ring := make([]queued[T], max(8, 2*len(q.ring)))
	for i := range q.n {
		ring[i] = q.ring[(q.head+i)%len(q.ring)]
	}
	q.ring, q.head = ring, 0
}

// Len is how many items are held.
func (q *Queue[T]) Len() int { return q.n }

// Snapshot is a queue's state, as a Pacer reads it.
type Snapshot struct {
	Depth  int           // items held
	Oldest time.Duration // how long the oldest has waited; 0 when none has
}

// Snapshot is the queue's depth and its oldest item's wait at now.
func (q *Queue[T]) Snapshot(now time.Time) Snapshot {
	if q.n == 0 {
		return Snapshot{}
	}
	return Snapshot{Depth: q.n, Oldest: max(now.Sub(q.ring[q.head].at), 0)}
}

// Drain releases the items p allows at now, oldest first, removing each
// as it is given. p is asked once, when the iteration starts; the items an
// iteration that stops early does not take stay held.
func (q *Queue[T]) Drain(p Pacer, now time.Time) iter.Seq[T] {
	return func(yield func(T) bool) {
		r := p.Pace(q.Snapshot(now), now)
		for range min(max(r.Take, 0), q.n) {
			e := q.ring[q.head]
			q.ring[q.head] = queued[T]{} // the item is the caller's now
			q.head = (q.head + 1) % len(q.ring)
			q.n--
			if !yield(e.v) {
				return
			}
		}
	}
}
