// Package streamtest checks a stream.Renderer against what a stream.Doc
// needs of one (docs/decisions/0009-MADR-streaming-content-engine.md A2):
// CheckRenderer streams generated documents through a Doc in random
// chunks, and reports through t where the Renderer breaks the Doc's
// contract. It is built on stream's exported API alone, so a Renderer in
// another module, such as glamour's, is held to the same checks as Plain.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package streamtest

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// docs is how many generated documents CheckRenderer streams, and widths
// the widths it renders each at.
const docs = 40

var widths = []int{40, 80}

// CheckRenderer streams generated Markdown documents through a stream.Doc
// built with r and o, in three chunkings each, and reports through t, by
// Errorf, each way r breaks a Doc's contract:
//
//   - chunking invariance: every chunking of a document gives the same
//     frame, which holds when r is a function of its input;
//   - the stated differences only: with no relaxed cut, a frame before
//     Finish is r's render of the whole source but that blank lines
//     between blocks are one (0009-MADR §4), which fails for a Renderer
//     whose output depends on where the source was cut;
//   - Finish exactness: after Finish, a frame is r's render of the whole
//     source, byte for byte.
//
// A Renderer that errors is reported too. The documents and chunkings
// come from fixed seeds, so a failure repeats.
func CheckRenderer(t testing.TB, r stream.Renderer, o ...stream.Option) {
	t.Helper()
	for seed := range uint64(docs) {
		rng := rand.New(rand.NewPCG(seed, 0x5eed)) //nolint:gosec // a test's documents, not a secret
		src := document(rng)
		for _, w := range widths {
			check(t, r, o, src, w, rng)
		}
	}
}

// check holds r to a Doc's contract over src at width w.
func check(t testing.TB, r stream.Renderer, o []stream.Option, src string, w int, rng *rand.Rand) {
	t.Helper()
	var first string
	for k := range 3 {
		d := stream.NewDoc(r, o...)
		for _, c := range chunks(src, k, rng) {
			d.Append(c)
		}
		f := d.Render(w)
		if k == 0 {
			first = f
		} else if f != first {
			t.Errorf("streamtest: chunking %d of document %q at width %d gave a frame other than the whole document's:\n%s\nwant:\n%s", k, src, w, f, first)
			return
		}
		d.Finish()
		want, err := r.Render(d.Source(), w)
		if err != nil {
			t.Errorf("streamtest: Render of document %q at width %d: %v", src, w, err)
			return
		}
		if got := d.Render(w); got != want {
			t.Errorf("streamtest: after Finish, document %q at width %d gave:\n%s\nwant the Renderer's of the whole source:\n%s", src, w, got, want)
			return
		}
	}
	d := stream.NewDoc(r, append(o[:len(o):len(o)], stream.WithRelaxAfter(1<<30))...)
	d.Append(src)
	frame := d.Render(w)
	whole, err := r.Render(d.Source(), w)
	if err != nil {
		t.Errorf("streamtest: Render of document %q at width %d: %v", src, w, err)
		return
	}
	if blanks(frame) != blanks(whole) {
		t.Errorf("streamtest: document %q at width %d, with no relaxed cut, gave a frame that differs from the whole document's render by more than its blank lines:\n%s\nwant:\n%s", src, w, frame, whole)
	}
}

// chunks splits s: whole for k 0, byte by byte for k 1, and in random
// sizes of 1 to 32 bytes for k 2.
func chunks(s string, k int, rng *rand.Rand) []string {
	var out []string
	switch k {
	case 0:
		return []string{s}
	case 1:
		for i := range len(s) {
			out = append(out, s[i:i+1])
		}
		return out
	}
	for s != "" {
		n := min(1+rng.IntN(32), len(s))
		out = append(out, s[:n])
		s = s[n:]
	}
	return out
}

// blanks is s with each run of blank lines one blank line, and none at its
// start and end, a line blank when nothing but spaces is left of it with
// its sequences stripped.
func blanks(s string) string {
	var out []string
	gap := false
	for l := range strings.SplitSeq(s, "\n") {
		if strings.TrimSpace(ansi.Strip(l)) == "" {
			gap = len(out) > 0
			continue
		}
		if gap {
			out = append(out, "")
			gap = false
		}
		out = append(out, strings.TrimRight(l, " "))
	}
	return strings.Join(out, "\n")
}

// words is the vocabulary a generated paragraph is made of, inline
// Markdown among it.
var words = strings.Fields("the a stream renders each block once and caches it while the tail " +
	"grows **bold** *em* `code` ~~gone~~ [link](https://example.com) [ref][r] snake_case 1.5 “quote”")

// document is a generated Markdown document of 5 to 14 blocks, each of
// CommonMark's and GFM's kinds a Doc cuts by: paragraphs, ATX and setext
// headings, fences of both characters with a blank line inside, tight and
// loose lists, bullet and ordered, with nested items, block quotes,
// tables, thematic breaks, HTML blocks, indented code and reference
// definitions, separated by blank lines and ending with a line break.
func document(rng *rand.Rand) string {
	var b strings.Builder
	n := 5 + rng.IntN(10)
	for i := range n {
		if i > 0 {
			b.WriteString("\n")
		}
		block(&b, rng)
	}
	return b.String()
}

// sentence is 3 to 10 of the words.
func sentence(rng *rand.Rand) string {
	n := 3 + rng.IntN(8)
	w := make([]string, n)
	for i := range w {
		w[i] = words[rng.IntN(len(words))]
	}
	return strings.Join(w, " ")
}

// block writes one block of a random kind.
func block(b *strings.Builder, rng *rand.Rand) {
	switch rng.IntN(12) {
	case 0, 1, 2:
		for range 1 + rng.IntN(4) {
			b.WriteString(sentence(rng) + "\n")
		}
	case 3:
		fmt.Fprintf(b, "%s %s\n", strings.Repeat("#", 1+rng.IntN(3)), sentence(rng))
	case 4:
		fmt.Fprintf(b, "%s\n%s\n", sentence(rng), strings.Repeat("=-"[rng.IntN(2):][:1], 3))
	case 5:
		fence := []string{"```", "~~~"}[rng.IntN(2)]
		fmt.Fprintf(b, "%sgo\nx := %d\n\ny := x\n%s\n", fence, rng.IntN(100), fence)
	case 6:
		loose := rng.IntN(2) == 0
		for i := range 2 + rng.IntN(3) {
			if loose && i > 0 {
				b.WriteString("\n")
			}
			if rng.IntN(2) == 0 {
				fmt.Fprintf(b, "%d. %s\n", i+1, sentence(rng))
			} else {
				fmt.Fprintf(b, "- %s\n", sentence(rng))
			}
			if rng.IntN(3) == 0 {
				fmt.Fprintf(b, "  - %s\n", sentence(rng))
			}
		}
	case 7:
		for range 1 + rng.IntN(3) {
			b.WriteString("> " + sentence(rng) + "\n")
		}
	case 8:
		b.WriteString("| a | b |\n| --- | :-: |\n")
		for i := range 1 + rng.IntN(4) {
			fmt.Fprintf(b, "| %d | %s |\n", i, words[rng.IntN(len(words))])
		}
	case 9:
		b.WriteString([]string{"***\n", "---\n", "<div>\n" + sentence(rng) + "\n</div>\n"}[rng.IntN(3)])
	case 10:
		b.WriteString("    code " + sentence(rng) + "\n")
	default:
		b.WriteString("[r]: https://example.com/ref\n")
	}
}
