package stream_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// replyRender streams gobble-cli's benchmark reply (its
// internal/cli/render/testdata/reply.txt at 5fb6928) through a Printer
// with DefaultMarkup at 100 columns, in chunks of 64 bytes, as its
// benchReplyRender streams it through its renderer.
func replyRender(reply []byte) string {
	p := stream.NewPrinter(stream.WithMarkup(stream.DefaultMarkup()), stream.WithWidth(100))
	var out strings.Builder
	for i := 0; i < len(reply); i += 64 {
		out.WriteString(p.Write(string(reply[i:min(i+64, len(reply))])))
	}
	out.WriteString(p.Flush())
	return out.String()
}

func readReply(tb testing.TB) []byte {
	tb.Helper()
	reply, err := os.ReadFile("testdata/reply.txt")
	if err != nil {
		tb.Fatal(err)
	}
	return reply
}

// BenchmarkReplyRender: a reply as a line-mode CLI prints it while it
// streams (gobble-cli's benchReplyRender, 30 µs on macOS, its 0030-PLAN P4).
func BenchmarkReplyRender(b *testing.B) {
	reply := readReply(b)
	b.ReportAllocs()
	for b.Loop() {
		replyRender(reply)
	}
}

// chunked streams src through a Printer with DefaultMarkup at 100
// columns, in chunks of size bytes, and flushes it.
func chunked(src string, size int) string {
	p := stream.NewPrinter(stream.WithMarkup(stream.DefaultMarkup()), stream.WithWidth(100))
	var out strings.Builder
	for i := 0; i < len(src); i += size {
		out.WriteString(p.Write(src[i:min(i+size, len(src))]))
	}
	out.WriteString(p.Flush())
	return out.String()
}

// BenchmarkReplyChunks: the benchmark reply in 8-byte chunks, as a model's
// small deltas come (docs/decisions/0025-MADR-charm-free-stream-allocations.md).
func BenchmarkReplyChunks(b *testing.B) {
	reply := string(readReply(b))
	b.ReportAllocs()
	for b.Loop() {
		chunked(reply, 8)
	}
}

// longLine is one paragraph line of 4,060 bytes and its line break.
var longLine = strings.Repeat("a word or two of a long line ", 140) + "\n"

// BenchmarkLongLine: one long paragraph line in 4-byte chunks, a partial
// line until its break comes
// (docs/decisions/0025-MADR-charm-free-stream-allocations.md, Finding 2).
func BenchmarkLongLine(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		chunked(longLine, 4)
	}
}

// paragraphs is about 200 KB of prose in paragraphs of five lines.
func paragraphs() string {
	var b strings.Builder
	for i := 0; b.Len() < 200<<10; i++ {
		b.WriteString("A line of a paragraph in a streamed reply, wrapped by Plain.\n")
		if i%5 == 4 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// BenchmarkDoc: 200 KB in 20-byte chunks through a Doc with Plain at 80
// columns, a frame after each: each segment rendered once, and the tail
// each frame (docs/decisions/0009-PLAN-streaming-content-engine.md Step
// 5).
func BenchmarkDoc(b *testing.B) {
	src := paragraphs()
	b.ReportAllocs()
	for b.Loop() {
		d := stream.NewDoc(stream.Plain(ansi.WcWidth))
		for i := 0; i < len(src); i += 20 {
			d.Append(src[i:min(i+20, len(src))])
			d.Render(80)
		}
	}
}

// BenchmarkWholeRender: the same 200 KB, the whole source so far rendered
// by Plain at each chunk, as a renderer with no stable prefix does: what
// BenchmarkDoc is measured against.
func BenchmarkWholeRender(b *testing.B) {
	src := paragraphs()
	plain := stream.Plain(ansi.WcWidth)
	b.ReportAllocs()
	for b.Loop() {
		for i := 0; i < len(src); i += 20 {
			plain.Render(src[:min(i+20, len(src))], 80) //nolint:errcheck // Plain never fails
		}
	}
}

// replyAllocs is the streamed reply's budget, gobble-cli's benchReplyRender
// at ccd8aa8 (docs/decisions/0023-MADR-line-mode-components.md A9 item 5).
const replyAllocs = 257

// TestReplyBudget: the streamed reply allocates no more than gobble-cli's.
func TestReplyBudget(t *testing.T) {
	reply := readReply(t)
	if n := testing.AllocsPerRun(20, func() { replyRender(reply) }); n > replyAllocs {
		t.Errorf("a streamed reply: %.0f allocations; want at most %d", n, replyAllocs)
	}
}

// heldHeap is the heap each of n objects build makes holds once built:
// HeapAlloc after a collection, before and after building them, kept
// alive past the second read
// (docs/decisions/0025-PLAN-charm-free-stream-allocations.md rule 12).
func heldHeap(n int, build func() any) float64 {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	kept := make([]any, n)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for i := range kept {
		kept[i] = build()
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(kept)
	return (float64(after.HeapAlloc) - float64(before.HeapAlloc)) / float64(n)
}

// TestHeldHeap: what a Printer or a Doc holds once its work is done, each
// of 2,000: a kept reply, written and flushed, as a transcript keeps it;
// a Printer never written; a finished Doc, drawn once. None keeps a
// sanitizer (docs/decisions/0025-MADR-charm-free-stream-allocations.md
// D3).
func TestHeldHeap(t *testing.T) {
	reply := string(readReply(t))
	printer := func() *stream.Printer {
		return stream.NewPrinter(stream.WithMarkup(stream.DefaultMarkup()), stream.WithWidth(100))
	}
	for _, c := range []struct {
		name  string
		limit float64
		build func() any
	}{
		{"a kept reply", 3584, func() any {
			p := printer()
			p.Write(reply)
			p.Flush()
			return p
		}},
		{"a Printer never written", 3072, func() any { return printer() }},
		{"a finished Doc", 6144, func() any {
			d := stream.NewDoc(stream.Plain(ansi.WcWidth))
			d.Append(reply)
			d.Finish()
			d.Render(80)
			return d
		}},
	} {
		got := heldHeap(2000, c.build)
		t.Logf("%s holds %.0f bytes of heap", c.name, got)
		if got > c.limit {
			t.Errorf("%s holds %.0f bytes of heap; want at most %.0f", c.name, got, c.limit)
		}
	}
}

// perRun is f's allocations and bytes allocated a call, over runs calls
// after one to warm it, as transcript's perRun counts them.
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

// TestChunkBudget: the benchmark reply in 8-byte chunks allocates at most
// 176 times and 25,600 bytes, or 192 and 27,648 in a race build: no chunk
// copies the partial line again
// (docs/decisions/0025-MADR-charm-free-stream-allocations.md D2).
func TestChunkBudget(t *testing.T) {
	reply := string(readReply(t))
	maxAllocs, maxBytes := 176.0, 25600.0
	if raceBuild {
		maxAllocs, maxBytes = 192, 27648
	}
	allocs, bytes := perRun(20, func() { chunked(reply, 8) })
	t.Logf("the reply in 8-byte chunks: %.0f allocations, %.0f bytes", allocs, bytes)
	if allocs > maxAllocs || bytes > maxBytes {
		t.Errorf("the reply in 8-byte chunks: %.0f allocations and %.0f bytes; want at most %.0f and %.0f", allocs, bytes, maxAllocs, maxBytes)
	}
}

// TestLongLineBudget: a 4,060-byte line in 4-byte chunks allocates at most
// 144 times and 69,632 bytes, or 144 and 77,824 in a race build, in
// proportion to its length and not its square
// (docs/decisions/0025-MADR-charm-free-stream-allocations.md D2).
func TestLongLineBudget(t *testing.T) {
	maxAllocs, maxBytes := 144.0, 69632.0
	if raceBuild {
		maxBytes = 77824
	}
	allocs, bytes := perRun(20, func() { chunked(longLine, 4) })
	t.Logf("the long line in 4-byte chunks: %.0f allocations, %.0f bytes", allocs, bytes)
	if allocs > maxAllocs || bytes > maxBytes {
		t.Errorf("the long line in 4-byte chunks: %.0f allocations and %.0f bytes; want at most %.0f and %.0f", allocs, bytes, maxAllocs, maxBytes)
	}
}

// TestPlainAllocs: Plain draws a short document with at most 22
// allocations, and its view with at most 29: a draw's input is written
// into the Printer's Source once, not copied there again
// (docs/decisions/0025-MADR-charm-free-stream-allocations.md D2).
func TestPlainAllocs(t *testing.T) {
	src := "A first paragraph, long enough to wrap at forty columns.\n\nThen a **second** one.\n"
	p := stream.Plain(ansi.WcWidth)
	render := testing.AllocsPerRun(200, func() { _, _ = p.Render(src, 40) })
	view := testing.AllocsPerRun(200, func() { _, _ = p.(stream.ViewRenderer).RenderView(src, 40) })
	t.Logf("Plain: Render %.0f, RenderView %.0f allocations", render, view)
	if render > 22 || view > 29 {
		t.Errorf("Plain: Render %.0f and RenderView %.0f allocations; want at most 22 and 29", render, view)
	}
}
