package frame

import "time"

// defaultMaxWait is how long a Coalescer holds a batch when no frame
// comes, Kilo's (docs/reports/0003-REPORT-agent-tui-ecosystem-research.md
// §8.17).
const defaultMaxWait = 100 * time.Millisecond

// FlushMsg is a Coalescer's deadline: the batch it held for a frame has
// waited its longest, and the program calls Flush.
type FlushMsg struct {
	At time.Time // the deadline Push gave
}

// Coalescer batches mergeable items, applied a frame at a time. Any other
// item is a barrier: the batch before it is applied first, then the
// barrier alone, so that a lifecycle event keeps its order among the
// deltas around it (docs/decisions/0009-MADR-streaming-content-engine.md
// A1 §3).
//
// The caller calls Flush on each frame, and at the deadline Push gives
// when no frame comes first. A Flush at a deadline that a frame has
// served already applies a later batch early, and loses nothing. A
// Coalescer reads no clock, and is used on one goroutine.
type Coalescer[T any] struct {
	apply     func([]T)
	mergeable func(T) bool
	maxWait   time.Duration
	batch     []T
}

// NewCoalescer is a Coalescer that gives each batch to apply, and each
// barrier to apply alone. apply must not keep the slice after it returns,
// since the Coalescer reuses it, and must not call the Coalescer. A nil
// mergeable merges every item. maxWait is how long a batch waits for a
// frame: 100 ms by default, and zero or less keeps it. A nil apply
// panics.
func NewCoalescer[T any](apply func([]T), mergeable func(T) bool, maxWait time.Duration) *Coalescer[T] {
	if apply == nil {
		panic("frame: NewCoalescer with a nil apply")
	}
	if maxWait <= 0 {
		maxWait = defaultMaxWait
	}
	return &Coalescer[T]{apply: apply, mergeable: mergeable, maxWait: maxWait}
}

// Push adds v at now. A mergeable v joins the batch; when it starts one,
// due is now plus the longest wait and ok is true, and the caller arms a
// timer for due that calls Flush. Any other v is a barrier: the batch is
// applied, then v alone, and ok is false.
func (c *Coalescer[T]) Push(v T, now time.Time) (due time.Time, ok bool) {
	if c.mergeable == nil || c.mergeable(v) {
		c.batch = append(c.batch, v)
		if len(c.batch) == 1 {
			return now.Add(c.maxWait), true
		}
		return time.Time{}, false
	}
	c.Flush()
	c.batch = append(c.batch, v)
	c.Flush()
	return time.Time{}, false
}

// Flush applies the batch, if there is one, and empties it.
func (c *Coalescer[T]) Flush() {
	if len(c.batch) == 0 {
		return
	}
	b := c.batch
	c.apply(b)
	clear(b)
	c.batch = b[:0]
}
