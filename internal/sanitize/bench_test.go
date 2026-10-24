package sanitize

import (
	"strings"
	"testing"
)

// sized repeats unit as often as fits in 4 KiB: safetext's benchmark
// inputs, so that the two packages measure the same bytes
// (docs/decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md Step 5).
func sized(unit string) string { return strings.Repeat(unit, 4096/len(unit)) }

var benchInputs = []struct{ name, s string }{
	{"clean-ascii", sized("The quick brown fox jumps over the lazy dog. ")},
	{"clean-cjk-emoji", sized("日本語のテキストと絵文字😀👍🏽 ")},
	{"tool-output", sized("pkg/file.go:12:\tok\r\n\x1b[32mPASS\x1b[0m line\n")},
	{"styled", sized("\x1b[1;31mred\x1b[0m \x1b]8;;https://e.org\x1b\\link\x1b]8;;\x1b\\ ")},
	{"hostile", sized("\x1b[2J\x1b]52;c;Y2xpcA==\x07\u202e\u009d0;x\u009cA\x9bB")},
}

func BenchmarkLine(b *testing.B) {
	for _, in := range benchInputs {
		b.Run(in.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = Line(in.s)
			}
		})
	}
}

func BenchmarkStyled(b *testing.B) {
	for _, in := range benchInputs {
		b.Run(in.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = Styled(in.s)
			}
		})
	}
}

func BenchmarkHasControl(b *testing.B) {
	for _, in := range benchInputs {
		b.Run(in.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = HasControl(in.s)
			}
		})
	}
}
