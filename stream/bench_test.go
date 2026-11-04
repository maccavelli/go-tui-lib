package stream_test

import (
	"os"
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
