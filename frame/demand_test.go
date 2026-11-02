package frame_test

import (
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

// TestCombine: the greatest demand wins, and none is None.
func TestCombine(t *testing.T) {
	for _, c := range []struct {
		d    []frame.Demand
		want frame.Demand
	}{
		{nil, frame.None},
		{[]frame.Demand{frame.None, frame.None}, frame.None},
		{[]frame.Demand{frame.Slow, frame.None}, frame.Slow},
		{[]frame.Demand{frame.None, frame.Fast, frame.Slow}, frame.Fast},
	} {
		if got := frame.Combine(c.d...); got != c.want {
			t.Errorf("Combine(%v) = %v; want %v", c.d, got, c.want)
		}
	}
}

// TestTicks: None ticks never; Slow every 83 ms and Fast every 60th of a
// second by default, or as Ticks says; a zero or negative field keeps its
// default; a demand with no token ticks never.
func TestTicks(t *testing.T) {
	for _, c := range []struct {
		name string
		t    frame.Ticks
		d    frame.Demand
		want time.Duration
	}{
		{"none", frame.Ticks{}, frame.None, 0},
		{"none, set", frame.Ticks{Slow: time.Second, Fast: time.Second}, frame.None, 0},
		{"slow", frame.Ticks{}, frame.Slow, 83 * time.Millisecond},
		{"fast", frame.Ticks{}, frame.Fast, time.Second / 60},
		{"slow, set", frame.Ticks{Slow: 200 * time.Millisecond}, frame.Slow, 200 * time.Millisecond},
		{"fast, set", frame.Ticks{Fast: time.Second / 120}, frame.Fast, time.Second / 120},
		{"slow, negative", frame.Ticks{Slow: -time.Second}, frame.Slow, 83 * time.Millisecond},
		{"no token", frame.Ticks{}, frame.Demand(7), 0},
	} {
		if got := c.t.Interval(c.d); got != c.want {
			t.Errorf("%s: Interval(%v) = %v; want %v", c.name, c.d, got, c.want)
		}
	}
}
