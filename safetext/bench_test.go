package safetext_test

import (
	"io"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/safetext"
)

// sized repeats unit as often as fits in 4 KiB, the inputs of gobble's
// measurements on issue #12 (0020-MADR, "The sanitizer").
func sized(unit string) string { return strings.Repeat(unit, 4096/len(unit)) }

// benchInputs are the five inputs, each built from fixed strings, with no
// randomness, so every run measures the same bytes.
var benchInputs = []struct {
	name, s string
	clean   bool // clean under Text and Line alike
}{
	{"clean-ascii", sized("The quick brown fox jumps over the lazy dog. "), true},
	{"clean-cjk-emoji", sized("日本語のテキストと絵文字😀👍🏽 "), true},
	{"tool-output", sized("pkg/file.go:12:\tok\r\n\x1b[32mPASS\x1b[0m line\n"), false},
	{"styled", sized("\x1b[1;31mred\x1b[0m \x1b]8;;https://e.org\x1b\\link\x1b]8;;\x1b\\ "), false},
	{"hostile", sized("\x1b[2J\x1b]52;c;Y2xpcA==\x07\u202e\u009d0;x\u009cA\x9bB"), false},
}

// TestAllocations is 0020-MADR D9's bar: clean input allocates nothing;
// input that needs cleaning, once; IsClean, nothing; a Sanitizer's Write,
// once warm, nothing.
func TestAllocations(t *testing.T) {
	for _, in := range benchInputs {
		for name, r := range map[string]safetext.Rules{
			"Text": safetext.Text(), "Line": safetext.Line(), "Strict": safetext.Strict(), "StatusLine": safetext.StatusLine(),
		} {
			want := 1.0
			if safetext.Clean(in.s, r) == in.s {
				want = 0 // clean under these rules, as styled is under StatusLine
			}
			if got := testing.AllocsPerRun(100, func() { _ = safetext.Clean(in.s, r) }); got != want {
				t.Errorf("Clean(%s, %s): %v allocations, want %v", in.name, name, got, want)
			}
			if got := testing.AllocsPerRun(100, func() { _ = safetext.IsClean(in.s, r) }); got != 0 {
				t.Errorf("IsClean(%s, %s): %v allocations, want 0", in.name, name, got)
			}
		}
		s := safetext.NewSanitizer(io.Discard, safetext.Text())
		p := []byte(in.s)
		_, _ = s.Write(p)
		if got := testing.AllocsPerRun(100, func() { _, _ = s.Write(p) }); got != 0 {
			t.Errorf("Sanitizer.Write(%s), warm: %v allocations, want 0", in.name, got)
		}
	}
}

func BenchmarkClean(b *testing.B) {
	for _, in := range benchInputs {
		for _, preset := range []struct {
			name  string
			rules safetext.Rules
		}{
			{"Text", safetext.Text()}, {"Line", safetext.Line()}, {"StatusLine", safetext.StatusLine()},
			// internal/sanitize's Styled, as Step 5 of 0020-PLAN maps it.
			{"Styled", safetext.Rules{Flatten: true, KeepSGR: true}},
		} {
			b.Run(in.name+"/"+preset.name, func(b *testing.B) {
				b.SetBytes(int64(len(in.s)))
				b.ReportAllocs()
				for b.Loop() {
					_ = safetext.Clean(in.s, preset.rules)
				}
			})
		}
	}
}

func BenchmarkIsClean(b *testing.B) {
	for _, in := range benchInputs {
		b.Run(in.name, func(b *testing.B) {
			b.SetBytes(int64(len(in.s)))
			b.ReportAllocs()
			for b.Loop() {
				_ = safetext.IsClean(in.s, safetext.Line())
			}
		})
	}
}

// BenchmarkLarge is 0009-PLAN Step 2's pair: 1 MiB of prose, and 1 MiB
// that is half escape sequences.
func BenchmarkLarge(b *testing.B) {
	prose := strings.Repeat("A line of ordinary prose, with punctuation, numbers 123 and 日本語.\n", 1<<20/72)
	half := strings.Repeat("\x1b[38;5;196mword\x1b[0m ", 1<<20/22)
	for name, s := range map[string]string{"prose": prose, "half-escapes": half} {
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(s)))
			b.ReportAllocs()
			for b.Loop() {
				_ = safetext.Clean(s, safetext.Text())
			}
		})
	}
}

func BenchmarkSanitizer(b *testing.B) {
	for _, in := range benchInputs {
		b.Run(in.name, func(b *testing.B) {
			s := safetext.NewSanitizer(io.Discard, safetext.Text())
			p := []byte(in.s)
			b.SetBytes(int64(len(p)))
			b.ReportAllocs()
			for b.Loop() {
				_, _ = s.Write(p)
			}
		})
	}
}
