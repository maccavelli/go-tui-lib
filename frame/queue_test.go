package frame_test

import (
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

// take is a Pacer that releases n items, in Smooth.
func take(n int) frame.Pacer {
	return frame.PacerFunc(func(frame.Snapshot, time.Time) frame.Release { return frame.Release{Take: n} })
}

// TestQueue: the zero value is empty; Drain takes from the head, in order,
// as many as the pacer says and no more than are held; an iteration that
// stops early leaves the rest; Snapshot gives the depth and the oldest
// item's wait.
func TestQueue(t *testing.T) {
	var q frame.Queue[int]
	start := time.Unix(1, 0)
	if q.Len() != 0 || q.Snapshot(start) != (frame.Snapshot{}) {
		t.Fatalf("the zero Queue: Len %d, Snapshot %+v; want empty", q.Len(), q.Snapshot(start))
	}
	for i := range 20 {
		q.Push(i, start.Add(time.Duration(i)*time.Millisecond))
	}
	if s := q.Snapshot(start.Add(30 * time.Millisecond)); s != (frame.Snapshot{Depth: 20, Oldest: 30 * time.Millisecond}) {
		t.Fatalf("Snapshot %+v; want depth 20, the oldest 30 ms", s)
	}
	if got := slices.Collect(q.Drain(take(5), start)); !slices.Equal(got, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("Drain of 5 gave %v; want 0 to 4", got)
	}
	for v := range q.Drain(take(5), start) {
		if v == 6 {
			break
		}
	}
	if got := slices.Collect(q.Drain(take(100), start)); !slices.Equal(got, []int{7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}) {
		t.Fatalf("after a stop at 6, Drain gave %v; want 7 to 19", got)
	}
	q.Push(1, start)
	if got := slices.Collect(q.Drain(take(-1), start)); len(got) != 0 || q.Len() != 1 {
		t.Fatalf("a negative take gave %v and left %d; want nothing taken", got, q.Len())
	}
	if s := q.Snapshot(start.Add(-time.Second)); s.Oldest != 0 {
		t.Fatalf("an item from the future waited %v; want 0", s.Oldest)
	}
}

// TestQueueWraps: items pushed and drained across the ring's end keep
// their order.
func TestQueueWraps(t *testing.T) {
	var q frame.Queue[int]
	now := time.Unix(1, 0)
	next, want := 0, 0
	for round := range 50 {
		for range round%7 + 1 {
			q.Push(next, now)
			next++
		}
		for v := range q.Drain(take(round%5+1), now) {
			if v != want {
				t.Fatalf("round %d: drained %d; want %d", round, v, want)
			}
			want++
		}
	}
}

// TestQueueAllocs: once the ring has grown, a frame's pushes and drain
// allocate nothing an item.
func TestQueueAllocs(t *testing.T) {
	var q frame.Queue[int]
	now := time.Unix(1, 0)
	all := frame.Immediate()
	for i := range 64 {
		q.Push(i, now)
	}
	for range q.Drain(all, now) {
	}
	allocs := testing.AllocsPerRun(100, func() {
		for i := range 64 {
			q.Push(i, now)
		}
		for range q.Drain(all, now) {
		}
	})
	if allocs > 2 {
		t.Fatalf("a frame of 64 items allocates %v times; want at most 2, none an item", allocs)
	}
}
