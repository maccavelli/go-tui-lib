package streamtest

import (
	"math/rand/v2"
	"strings"
	"testing"
)

// TestDocument: a generated document ends with a line break, and the same
// seed gives the same document, so that a failure repeats.
func TestDocument(t *testing.T) {
	a := document(rand.New(rand.NewPCG(7, 7))) //nolint:gosec // a test's documents
	b := document(rand.New(rand.NewPCG(7, 7))) //nolint:gosec // a test's documents
	if a != b || !strings.HasSuffix(a, "\n") || strings.Count(a, "\n") < 5 {
		t.Fatalf("document gave %q and %q; want one document, ending a line", a, b)
	}
}
