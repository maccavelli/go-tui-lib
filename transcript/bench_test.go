package transcript_test

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/transcript"
)

// benchTranscript is 4 MiB of gobble-cli's benchmark reply (its
// internal/cli/render/testdata/reply.txt at 5fb6928), each a reply styled
// by DefaultMarkup, as gobble's BenchmarkReprint builds it.
func benchTranscript(tb testing.TB) *transcript.Transcript {
	tb.Helper()
	reply, err := os.ReadFile("testdata/reply.txt")
	if err != nil {
		tb.Fatal(err)
	}
	tr := transcript.NewTranscript()
	for range transcript.DefaultMaxBytes / len(reply) {
		p := stream.NewPrinter(stream.WithMarkup(stream.DefaultMarkup()), stream.WithWidth(100))
		p.Write(string(reply))
		p.Flush()
		tr.AddReply(p)
	}
	return tr
}

// BenchmarkReprint: a reprint at 60 columns of the last 10,000 rows of a
// 4 MiB transcript, the most a reprint writes (gobble-cli's 0030-MADR D5;
// its BenchmarkReprint, 9.4 ms on macOS, 29 ms on its Linux machine and
// 12 ms on its Windows laptop, 0030-PLAN P4).
func BenchmarkReprint(b *testing.B) {
	tr := benchTranscript(b)
	now := time.Unix(1, 0)
	tr.Resize(80, now)
	b.ReportAllocs()
	for b.Loop() {
		tr.Resize(61, now)
		tr.Resize(60, now)
		now = now.Add(transcript.DefaultSettle)
		if tr.HandleTimeout(now, true) == "" {
			b.Fatal("no reprint")
		}
	}
}

// The reprint's budget, gobble-cli's BenchmarkReprint at ccd8aa8
// (docs/decisions/0023-MADR-line-mode-components.md A9 item 5).
const (
	reprintAllocs = 49696
	reprintBytes  = 7274618
)

// TestReprintBudget: the benchmark's reprint allocates no more, in count or
// in bytes, than gobble-cli's.
func TestReprintBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("builds 4 MiB of replies")
	}
	tr := benchTranscript(t)
	now := time.Unix(1, 0)
	tr.Resize(80, now)
	allocs, bytes := perRun(10, func() {
		tr.Resize(61, now)
		tr.Resize(60, now)
		now = now.Add(transcript.DefaultSettle)
		if tr.HandleTimeout(now, true) == "" {
			t.Fatal("no reprint")
		}
	})
	if allocs > reprintAllocs || bytes > reprintBytes {
		t.Errorf("a reprint: %.0f allocations and %.0f bytes; want at most %d and %d", allocs, bytes, reprintAllocs, reprintBytes)
	}
}

// perRun is f's allocations and bytes allocated a call, over runs calls
// after one to warm it, counted as testing.AllocsPerRun counts them.
func perRun(runs int, f func()) (allocs, bytes float64) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	f()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range runs {
		f()
	}
	runtime.ReadMemStats(&after)
	return float64(after.Mallocs-before.Mallocs) / float64(runs), float64(after.TotalAlloc-before.TotalAlloc) / float64(runs)
}

// TestBenchTranscript: the benchmark's transcript reprints, after the
// notice, at most 10,000 rows as the terminal shows them at 60 columns, a
// line wider than 60, such as a heading kept whole, taking more than one,
// and close to that many: gobble counts line breaks, and this transcript
// the rows they take.
func TestBenchTranscript(t *testing.T) {
	if testing.Short() {
		t.Skip("builds 4 MiB of replies")
	}
	tr := benchTranscript(t)
	got := body(t, reprint(t, tr, 60))
	rest, ok := strings.CutPrefix(got, notice)
	if !ok {
		t.Fatal("no notice before a cut reprint")
	}
	rows := 0
	for l := range strings.SplitSeq(strings.TrimSuffix(rest, "\n"), "\n") {
		rows += max(1, (ansi.WcWidth.StringWidth(l)+59)/60)
	}
	if rows > transcript.DefaultMaxRows || rows < transcript.DefaultMaxRows-20 {
		t.Fatalf("%d rows at 60 columns, in %d lines; want at most %d, and close to it", rows, strings.Count(rest, "\n"), transcript.DefaultMaxRows)
	}
}
