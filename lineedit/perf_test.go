//go:build perf

package lineedit_test

import (
	"runtime"
	"slices"
	"testing"
	"time"
)

// keyDrawBaseline is gobble-cli's editor's BenchmarkKeyDraw, by GOOS: its
// internal/cli/editor/perf_test.go at 6c5eb02, which its 0030-MADR D5
// holds to three times (docs/decisions/0023-MADR-line-mode-components.md
// A12 item 7).
var keyDrawBaseline = map[string]time.Duration{
	"darwin":  1500 * time.Nanosecond,
	"linux":   5000 * time.Nanosecond,
	"windows": 1400 * time.Nanosecond,
}

// perfSlack is how many times its baseline the key may take, gobble's.
const perfSlack = 3

// perfRuns is how many times each benchmark runs; its median is the one
// recorded.
const perfRuns = 5

// TestPerfKeys: each key benchmark's median of five runs, logged with its
// range and allocations; issue #13's key, KeyDraw, within three times
// gobble's baseline for this GOOS. The perf gate runs it, and CI does not,
// since time varies by host and load.
func TestPerfKeys(t *testing.T) {
	for _, kc := range keyCases {
		times := make([]time.Duration, perfRuns)
		var r testing.BenchmarkResult
		for i := range times {
			r = testing.Benchmark(func(b *testing.B) { benchKeys(b, kc) })
			times[i] = time.Duration(r.NsPerOp())
		}
		slices.Sort(times)
		med := times[perfRuns/2]
		t.Logf("%s: %v per op, median of %d, %v to %v; %d allocs, %d B", kc.name, med, perfRuns, times[0], times[perfRuns-1], r.AllocsPerOp(), r.AllocedBytesPerOp())
		if kc.name != "KeyDraw" {
			continue
		}
		base, ok := keyDrawBaseline[runtime.GOOS]
		if !ok {
			t.Logf("KeyDraw: no baseline for %s", runtime.GOOS)
			continue
		}
		t.Logf("KeyDraw: budget %v per op, %d times gobble's %v", perfSlack*base, perfSlack, base)
		if med > perfSlack*base {
			t.Errorf("KeyDraw: %v per op, more than %d times its baseline %v", med, perfSlack, base)
		}
	}
}
