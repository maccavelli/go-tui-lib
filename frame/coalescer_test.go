package frame_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

// event is a test's item: a delta, mergeable, or a barrier.
type event struct {
	n       int
	barrier bool
}

func isDelta(e event) bool { return !e.barrier }

// recorder collects what a Coalescer applies, a copy of each batch.
type recorder struct{ batches [][]event }

func (r *recorder) apply(b []event) { r.batches = append(r.batches, slices.Clone(b)) }

// play runs script on c, and flushes at its end: 'd' pushes a delta, 'b'
// a barrier, and 'f' is a frame, which flushes. It gives the steps whose
// push gave a deadline.
func play(c *frame.Coalescer[event], script string, now time.Time) (deadlines []int) {
	for i, step := range script {
		switch step {
		case 'd', 'b':
			if _, ok := c.Push(event{n: i, barrier: step == 'b'}, now); ok {
				deadlines = append(deadlines, i)
			}
		case 'f':
			c.Flush()
		}
	}
	c.Flush()
	return deadlines
}

// model is what a script applies: each run of deltas, ended by a frame, a
// barrier or the end, as one batch, and each barrier alone, in order; and
// the steps whose delta starts a run, which give a deadline.
func model(script string) (batches [][]event, deadlines []int) {
	var run []event
	end := func() {
		if len(run) > 0 {
			batches = append(batches, run)
			run = nil
		}
	}
	for i, step := range script {
		switch step {
		case 'd':
			if len(run) == 0 {
				deadlines = append(deadlines, i)
			}
			run = append(run, event{n: i})
		case 'b':
			end()
			batches = append(batches, []event{{n: i, barrier: true}})
		case 'f':
			end()
		}
	}
	end()
	return batches, deadlines
}

// scripts is every script of length 1 to n over "dbf".
func scripts(n int) []string {
	out := []string{""}
	var all []string
	for range n {
		var next []string
		for _, s := range out {
			for _, step := range "dbf" {
				next = append(next, s+string(step))
			}
		}
		all = append(all, next...)
		out = next
	}
	return all
}

// TestCoalescerModel: for every interleaving of deltas, barriers and
// frames up to 7 steps, a barrier is applied after the deltas before it and
// before those after it, alone; each run of deltas is one batch; and a
// deadline is given when a delta starts a batch, and only then
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §3).
func TestCoalescerModel(t *testing.T) {
	now := time.Unix(1, 0)
	for _, s := range scripts(7) {
		var r recorder
		c := frame.NewCoalescer(r.apply, isDelta, 0)
		deadlines := play(c, s, now)
		wantBatches, wantDeadlines := model(s)
		if fmt.Sprint(r.batches) != fmt.Sprint(wantBatches) {
			t.Fatalf("%q applied %v; want %v", s, r.batches, wantBatches)
		}
		if !slices.Equal(deadlines, wantDeadlines) {
			t.Fatalf("%q gave deadlines at steps %v; want %v", s, deadlines, wantDeadlines)
		}
	}
}

// TestCoalescerWait: the deadline is the push's time plus the longest wait,
// 100 ms by default and for zero or less, or the one given.
func TestCoalescerWait(t *testing.T) {
	now := time.Unix(1, 0)
	for _, c := range []struct {
		maxWait, want time.Duration
	}{
		{0, 100 * time.Millisecond},
		{-time.Second, 100 * time.Millisecond},
		{30 * time.Millisecond, 30 * time.Millisecond},
	} {
		var r recorder
		co := frame.NewCoalescer(r.apply, isDelta, c.maxWait)
		due, ok := co.Push(event{}, now)
		if !ok || due.Sub(now) != c.want {
			t.Errorf("maxWait %v: Push gave %v, %v; want a deadline %v on", c.maxWait, due.Sub(now), ok, c.want)
		}
		if _, ok := co.Push(event{n: 1}, now.Add(time.Millisecond)); ok {
			t.Errorf("maxWait %v: a second delta gave a deadline", c.maxWait)
		}
	}
}

// TestCoalescerNil: a nil mergeable merges every item; a nil apply panics;
// a Flush with nothing batched applies nothing.
func TestCoalescerNil(t *testing.T) {
	var r recorder
	c := frame.NewCoalescer(r.apply, nil, 0)
	c.Flush()
	if len(r.batches) != 0 {
		t.Fatalf("an empty Flush applied %v", r.batches)
	}
	play(c, "dbd", time.Unix(1, 0))
	if len(r.batches) != 1 || len(r.batches[0]) != 3 {
		t.Fatalf("a nil mergeable applied %v; want one batch of 3", r.batches)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("NewCoalescer with a nil apply did not panic")
		}
	}()
	frame.NewCoalescer[event](nil, nil, 0)
}

// A Coalescer between a stream's deltas and a view: deltas are batched for
// the next frame, and a lifecycle event applies the deltas before it first.
func ExampleCoalescer() {
	var shown []string
	c := frame.NewCoalescer(func(b []string) {
		shown = append(shown, strings.Join(b, ""))
	}, func(s string) bool { return !strings.HasPrefix(s, "[") }, 0)
	now := time.Unix(0, 0)
	c.Push("Hel", now)
	c.Push("lo", now)
	c.Push("[done]", now) // a barrier: "Hello" is applied first
	c.Push("!", now)
	c.Flush() // the next frame
	fmt.Printf("%q\n", shown)
	// Output: ["Hello" "[done]" "!"]
}
