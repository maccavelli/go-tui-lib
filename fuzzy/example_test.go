package fuzzy_test

import (
	"fmt"
	"slices"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
)

// Ranking paths for a query: a candidate whose last path element equals
// the query is exact, one whose last element starts with it is a prefix
// match, one with a whole path element equal to it is a boundary match,
// and the rest that hold its letters in order are subsequence matches.
// A candidate that does not hold them is left out.
func ExampleMatcher_Rank() {
	paths := []string{
		"docs/guides/commands.md",
		"cmd/main.go",
		"main.go",
		"internal/main/run.go",
		"README.md",
		"maintainers.txt",
		"email/index.go",
	}
	m := fuzzy.NewMatcher("main")
	for _, r := range m.Rank(slices.Values(paths), func(p string) string { return p }, 0) {
		fmt.Printf("%-12s %s\n", r.Score.Grade, r.Item)
	}
	// Output:
	// exact        main.go
	// exact        cmd/main.go
	// prefix       maintainers.txt
	// boundary     internal/main/run.go
	// subsequence  email/index.go
}
