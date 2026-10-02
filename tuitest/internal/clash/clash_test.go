package clash

import (
	"flag"
	"testing"

	"github.com/maccavelli/go-tui-lib/tuitest"
)

// update is the consumer's own flag. In v0.1.0, tuitest registered -update in
// its init, which runs first, and this declaration panicked.
var update = flag.Bool("update", false, "rewrite this package's golden files")

func TestGoldenBesideOwnUpdateFlag(t *testing.T) {
	_ = update
	tuitest.Golden(t, "clash", tuitest.Matrix{Widths: []int{10, 20}}, func(c tuitest.Case) string {
		return c.Name() + "\n"
	})
}
