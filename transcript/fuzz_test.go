package transcript_test

import (
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/transcript"
)

// FuzzTranscript: any sequence of pieces added, resizes, timeouts and idle
// calls never panics; Timeout never gives a deadline that has passed once
// HandleTimeout has seen it; and every reprint is framed as D9 orders it,
// its line breaks CR LF, the notice first when it cut, and at most
// WithMaxRows rows at its width after the notice.
func FuzzTranscript(f *testing.F) {
	// Each op is two bytes, the op and its argument: add text, a function
	// or a reply; resize; let time pass; call HandleTimeout. Both seeds
	// reach a reprint.
	f.Add([]byte("\x02\x00\x00\x05\x01\x03\x03\x10\x03\x50\x04\xff\x05\x00"), "one\ntwo **three**\n\n| a |\n|---|\n")
	f.Add([]byte("\x00\x09\x03\x0a\x03\x14\x04\x50\x05\x01\x05\x00\x02\x00\x03\x1e\x04\xff\x05\x00"), "世界 a long enough line\nand more\n")
	f.Fuzz(func(t *testing.T, ops []byte, text string) {
		const maxRows = 6
		tr := transcript.NewTranscript(transcript.WithMaxRows(maxRows), transcript.WithMaxBytes(200))
		now := time.Unix(1, 0)
		for i := 0; i+1 < len(ops); i += 2 {
			arg := int(ops[i+1])
			cut := min(arg%(len(text)+1), len(text))
			switch ops[i] % 6 {
			case 0:
				tr.AddText(text[:cut])
			case 1:
				s := text[cut:]
				tr.AddDrawn(func(w int) string { return s + strings.Repeat("-", min(max(w, 0), 50)) + "\n" })
			case 2:
				p := stream.NewPrinter(stream.WithMarkup(stream.DefaultMarkup()))
				p.Write(text)
				p.Flush()
				tr.AddReply(p)
			case 3:
				tr.Resize(arg%90, now)
			case 4:
				now = now.Add(time.Duration(arg) * 2 * time.Millisecond)
			case 5:
				out := tr.HandleTimeout(now, arg%2 == 0)
				if d, ok := tr.Timeout(); ok && !now.Before(d) {
					t.Fatalf("Timeout gives %v, passed at %v", d, now)
				}
				if out != "" {
					check(t, out, maxRows)
				}
			}
		}
	})
}

// check reads one default reprint: its framing, its CR LF line breaks, and
// its lines after the notice, each at least one row, so no more than
// maxRows of them.
func check(t *testing.T, out string, maxRows int) {
	t.Helper()
	text, framed := strings.CutPrefix(out, head)
	text, ended := strings.CutSuffix(text, tail)
	if !framed || !ended {
		t.Fatalf("reprint %q is not framed as D9 orders it", out)
	}
	if strings.Count(text, "\n") != strings.Count(text, "\r\n") {
		t.Fatalf("a line break not CR LF in %q", text)
	}
	text = strings.TrimPrefix(text, "(earlier output is not shown again)\r\n")
	if n := strings.Count(strings.TrimSuffix(text, "\r\n"), "\r\n") + 1; text != "" && n > maxRows {
		t.Fatalf("%d lines after the notice, more than %d rows: %q", n, maxRows, text)
	}
}
