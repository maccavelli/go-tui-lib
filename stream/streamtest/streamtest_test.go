package streamtest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/stream/streamtest"
)

// TestPlain: Plain keeps the Doc's contract.
func TestPlain(t *testing.T) {
	streamtest.CheckRenderer(t, stream.Plain(ansi.WcWidth))
	streamtest.CheckRenderer(t, stream.Plain(ansi.GraphemeWidth), stream.WithRelaxAfter(64))
}

// markdownish is a Renderer that renders each line as itself, a
// definition as nothing, as a Markdown renderer renders one.
var markdownish = stream.RendererFunc(func(src string, _ int) (string, error) {
	var s stream.Scanner
	var out []string
	for l := range strings.SplitSeq(src, "\n") {
		if s.Scan(l).Kind != stream.BlockDefinition {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n"), nil
})

// TestStatedDifferences: a Renderer that renders each block as itself
// keeps the contract, the stated differences the only ones
// (docs/decisions/0009-MADR-streaming-content-engine.md §4).
func TestStatedDifferences(t *testing.T) {
	streamtest.CheckRenderer(t, markdownish)
}

// failures is a testing.TB that keeps its failures.
type failures struct {
	testing.TB
	got []string
}

func (f *failures) Errorf(format string, args ...any) {
	f.got = append(f.got, fmt.Sprintf(format, args...))
}

func (f *failures) Helper() {}

// TestCheckRendererCatches: a Renderer whose output depends on where the
// source was cut, here by its input's length, is reported.
func TestCheckRendererCatches(t *testing.T) {
	cutDependent := stream.RendererFunc(func(src string, _ int) (string, error) {
		return fmt.Sprintf("%d bytes\n%s", len(src), src), nil
	})
	f := &failures{TB: t}
	streamtest.CheckRenderer(f, cutDependent)
	if len(f.got) == 0 {
		t.Fatal("CheckRenderer reported nothing for a Renderer that depends on where the source was cut")
	}
}
