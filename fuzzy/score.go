package fuzzy

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// Grade is how a candidate matched; a lower grade is a better match. Its
// text form is a stable lowercase token.
type Grade uint8

const (
	// Exact is a candidate that, or whose last path element or that
	// element's stem, equals the query. An empty query gives every
	// candidate this grade.
	Exact Grade = iota
	// Prefix is a candidate that, or whose last path element, starts with
	// the query.
	Prefix
	// Boundary is a candidate with a whole word or path segment equal to
	// the query, or where every query cluster lands on a word boundary, a
	// camelCase hump or a digit.
	Boundary
	// Subsequence is any other match.
	Subsequence
)

var gradeNames = enum.Names[Grade]{
	Pkg: pkgName, Type: "Grade",
	Tokens: []string{"exact", "prefix", "boundary", "subsequence"},
}

// String is the grade's token: "exact", "prefix", "boundary" or
// "subsequence".
func (g Grade) String() string { return gradeNames.String(g) }

// MarshalText is the grade's token. A grade with no token is an error.
func (g Grade) MarshalText() ([]byte, error) { return gradeNames.Marshal(g) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (g *Grade) UnmarshalText(b []byte) error { return gradeNames.Unmarshal(b, g) }

// Score is how well a candidate matched: its grade first, then its value
// within the grade, where higher is better.
type Score struct {
	Grade Grade
	Value int
}

// Range is the bytes s[Start:End] of a candidate. Both ends lie on grapheme
// cluster boundaries.
type Range struct{ Start, End int }

// Match is a Score with the ranges of the candidate that matched, in order,
// neither overlapping nor touching.
type Match struct {
	Score
	Ranges []Range
}
