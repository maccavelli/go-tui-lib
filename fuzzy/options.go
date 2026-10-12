package fuzzy

import "github.com/maccavelli/go-tui-lib/internal/enum"

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "fuzzy"

// CaseMode is how a Matcher treats letter case. Its text form is a stable
// lowercase token.
type CaseMode uint8

const (
	// Smart is case-insensitive unless the query has an upper-case letter,
	// as in fzf and Vim's smartcase. It is the zero CaseMode.
	Smart CaseMode = iota
	// Insensitive folds case always.
	Insensitive
	// Sensitive compares runes exactly.
	Sensitive
)

var caseNames = enum.Names[CaseMode]{
	Pkg: pkgName, Type: "CaseMode",
	Tokens: []string{"smart", "insensitive", "sensitive"},
}

// String is the mode's token: "smart", "insensitive" or "sensitive".
func (c CaseMode) String() string { return caseNames.String(c) }

// MarshalText is the mode's token. A mode with no token is an error.
func (c CaseMode) MarshalText() ([]byte, error) { return caseNames.Marshal(c) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (c *CaseMode) UnmarshalText(b []byte) error { return caseNames.Unmarshal(b, c) }

// Normalizer maps a rune before it is compared, as folding accents maps é
// to e. It is applied to the query and the candidate alike, before case
// folding, and must be safe for concurrent use. This package ships none: a
// Unicode normaliser would need golang.org/x/text.
type Normalizer interface {
	Normalize(r rune) rune
}

// NormalizerFunc is a function used as a Normalizer.
type NormalizerFunc func(r rune) rune

// Normalize calls f.
func (f NormalizerFunc) Normalize(r rune) rune { return f(r) }

// Option configures NewMatcher.
type Option interface{ apply(*config) }

// optionFunc is a function used as an Option.
type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// The defaults.
const (
	// DefaultMaxAlign is the longest stretch of a candidate, in grapheme
	// clusters, that a query is aligned over exactly; a longer one keeps the
	// greedy alignment.
	DefaultMaxAlign = 512
	// MaxQuery is how many grapheme clusters of a query are matched; the
	// rest are ignored, so a pasted wall of text cannot make each match
	// slow.
	MaxQuery = 256
	// defaultDelimiters are fzf's, and the Windows path separator.
	defaultDelimiters = `/\,:;|`
)

type config struct {
	mode       CaseMode
	norm       Normalizer
	delimiters string
	extended   bool
	maxAlign   int
}

// WithCase sets the case mode. The default is Smart.
func WithCase(m CaseMode) Option { return optionFunc(func(c *config) { c.mode = m }) }

// WithNormalizer maps each rune of the query and the candidates through n
// before they are compared.
func WithNormalizer(n Normalizer) Option { return optionFunc(func(c *config) { c.norm = n }) }

// WithDelimiters sets the characters after which a word earns the
// delimiter boundary bonus. The default is fzf's "/,:;|" and the backslash.
func WithDelimiters(d string) Option { return optionFunc(func(c *config) { c.delimiters = d }) }

// WithExtended turns on fzf's extended syntax: space-separated terms must
// all match, and a term may be 'exact, ^prefix, suffix$, ^equal$, or
// negated with a leading !. Without it a space is an ordinary character,
// which suits titles such as "Open Settings".
func WithExtended() Option { return optionFunc(func(c *config) { c.extended = true }) }

// WithMaxAlign sets the longest stretch, in grapheme clusters, aligned
// exactly. n below 1 is DefaultMaxAlign.
func WithMaxAlign(n int) Option { return optionFunc(func(c *config) { c.maxAlign = n }) }
