package stream_test

import (
	"os"
	"strings"
	"testing"

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
