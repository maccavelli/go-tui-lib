package frame_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

// pace is one frame of a pacer's table: the queue's state at an offset of
// the start, and the release wanted.
type pace struct {
	at     time.Duration
	depth  int
	oldest time.Duration
	take   int
	gear   frame.Gear
}

// run gives p each frame of steps in turn, and fails on the first release
// that differs from the one wanted.
func run(t *testing.T, p frame.Pacer, steps []pace) {
	t.Helper()
	start := time.Unix(1, 0)
	for i, s := range steps {
		got := p.Pace(frame.Snapshot{Depth: s.depth, Oldest: s.oldest}, start.Add(s.at))
		if got != (frame.Release{Take: s.take, Gear: s.gear}) {
			t.Fatalf("step %d (at %v, depth %d, oldest %v): %+v; want take %d in %v", i, s.at, s.depth, s.oldest, got, s.take, s.gear)
		}
	}
}

const ms = time.Millisecond

// TestAdaptive: codex's transitions, as Adaptive's defaults hold them
// (docs/decisions/0009-MADR-streaming-content-engine.md §3).
func TestAdaptive(t *testing.T) {
	for _, c := range []struct {
		name  string
		steps []pace
	}{
		{"Smooth releases one a frame", []pace{
			{0, 3, 10 * ms, 1, frame.Smooth},
			{16 * ms, 7, 119 * ms, 1, frame.Smooth},
		}},
		{"CatchUp at depth 8", []pace{
			{0, 8, 0, 8, frame.CatchUp},
		}},
		{"CatchUp at age 120 ms", []pace{
			{0, 1, 120 * ms, 1, frame.CatchUp},
		}},
		{"leaves CatchUp once calm has held 250 ms", []pace{
			{0, 8, 0, 8, frame.CatchUp},
			{10 * ms, 2, 30 * ms, 2, frame.CatchUp},  // calm from here
			{259 * ms, 2, 40 * ms, 2, frame.CatchUp}, // 249 ms of calm
			{260 * ms, 2, 40 * ms, 1, frame.Smooth},  // 250 ms
		}},
		{"calm broken starts the hold again", []pace{
			{0, 8, 0, 8, frame.CatchUp},
			{10 * ms, 2, 30 * ms, 2, frame.CatchUp},
			{200 * ms, 3, 30 * ms, 3, frame.CatchUp}, // deeper than 2
			{300 * ms, 1, 10 * ms, 1, frame.CatchUp}, // calm again from here
			{549 * ms, 1, 10 * ms, 1, frame.CatchUp},
			{550 * ms, 1, 10 * ms, 1, frame.Smooth},
		}},
		{"too old to be calm", []pace{
			{0, 8, 0, 8, frame.CatchUp},
			{10 * ms, 1, 41 * ms, 1, frame.CatchUp},
			{300 * ms, 1, 41 * ms, 1, frame.CatchUp},
		}},
		{"no re-entry within the hold but for a severe backlog", []pace{
			{0, 8, 0, 8, frame.CatchUp},
			{10 * ms, 1, 0, 1, frame.CatchUp},
			{260 * ms, 1, 0, 1, frame.Smooth},  // left at 260 ms
			{300 * ms, 10, 0, 1, frame.Smooth}, // deep, but held
			{400 * ms, 5, 299 * ms, 1, frame.Smooth},
			{500 * ms, 64, 0, 64, frame.CatchUp}, // severe by depth
		}},
		{"severe by age", []pace{
			{0, 8, 0, 8, frame.CatchUp},
			{10 * ms, 1, 0, 1, frame.CatchUp},
			{260 * ms, 1, 0, 1, frame.Smooth},
			{270 * ms, 2, 300 * ms, 2, frame.CatchUp},
		}},
		{"re-entry once the hold has passed", []pace{
			{0, 8, 0, 8, frame.CatchUp},
			{10 * ms, 1, 0, 1, frame.CatchUp},
			{260 * ms, 1, 0, 1, frame.Smooth},
			{509 * ms, 8, 0, 1, frame.Smooth},
			{510 * ms, 8, 0, 8, frame.CatchUp},
		}},
		{"an empty queue is Smooth again", []pace{
			{0, 8, 0, 8, frame.CatchUp},
			{10 * ms, 0, 0, 0, frame.Smooth},
			{20 * ms, 3, 0, 1, frame.Smooth},
		}},
	} {
		t.Run(c.name, func(t *testing.T) { run(t, frame.Adaptive(), c.steps) })
	}
}

// TestAdaptiveOptions: each number is an option, and zero or less keeps
// its default.
func TestAdaptiveOptions(t *testing.T) {
	run(t, frame.Adaptive(frame.WithCatchUpDepth(3)), []pace{{0, 3, 0, 3, frame.CatchUp}})
	run(t, frame.Adaptive(frame.WithCatchUpDepth(0)), []pace{{0, 3, 0, 1, frame.Smooth}})
	run(t, frame.Adaptive(frame.WithCatchUpAge(50*ms)), []pace{{0, 1, 50 * ms, 1, frame.CatchUp}})
	run(t, frame.Adaptive(frame.WithCatchUpAge(-1)), []pace{{0, 1, 50 * ms, 1, frame.Smooth}})
	run(t, frame.Adaptive(frame.WithSmoothDepth(5), frame.WithHold(100*ms)), []pace{
		{0, 8, 0, 8, frame.CatchUp},
		{10 * ms, 5, 0, 5, frame.CatchUp},
		{110 * ms, 5, 0, 1, frame.Smooth},
	})
	run(t, frame.Adaptive(frame.WithSmoothAge(100*ms)), []pace{
		{0, 8, 0, 8, frame.CatchUp},
		{10 * ms, 1, 100 * ms, 1, frame.CatchUp},
		{260 * ms, 1, 100 * ms, 1, frame.Smooth},
	})
	run(t, frame.Adaptive(frame.WithSevereDepth(10), frame.WithSevereAge(time.Hour)), []pace{
		{0, 8, 0, 8, frame.CatchUp},
		{10 * ms, 1, 0, 1, frame.CatchUp},
		{260 * ms, 1, 0, 1, frame.Smooth},
		{300 * ms, 10, 0, 10, frame.CatchUp},
	})
}

// TestImmediate: Immediate releases the whole queue, in CatchUp; PacerFunc
// adapts a function.
func TestImmediate(t *testing.T) {
	run(t, frame.Immediate(), []pace{{0, 5, 0, 5, frame.CatchUp}, {0, 0, 0, 0, frame.CatchUp}})
	half := frame.PacerFunc(func(s frame.Snapshot, _ time.Time) frame.Release { return frame.Release{Take: s.Depth / 2} })
	run(t, half, []pace{{0, 6, 0, 3, frame.Smooth}})
}

// TestGear: a gear's tokens.
func TestGear(t *testing.T) {
	for g, want := range map[frame.Gear]string{frame.Smooth: "smooth", frame.CatchUp: "catch-up", frame.Gear(9): "Gear(9)"} {
		if got := g.String(); got != want {
			t.Errorf("%d: %q; want %q", g, got, want)
		}
	}
}

// ExampleAdaptive: a queue of a stream's lines released a frame at a time:
// one a frame while it is shallow, all of it once it backs up.
func ExampleAdaptive() {
	var q frame.Queue[string]
	p := frame.Adaptive()
	now := time.Unix(0, 0)
	for i := range 10 {
		q.Push(fmt.Sprint("line ", i), now)
	}
	for f := range 2 {
		var got []string
		for line := range q.Drain(p, now) {
			got = append(got, line)
		}
		fmt.Println("frame", f, "released", len(got), "of", len(got)+q.Len())
		now = now.Add(time.Second / 60)
	}
	q.Push("line 10", now)
	for line := range q.Drain(p, now) {
		fmt.Println("frame 2 released", line)
	}
	// Output:
	// frame 0 released 10 of 10
	// frame 1 released 0 of 0
	// frame 2 released line 10
}
