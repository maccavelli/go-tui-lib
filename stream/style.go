package stream

import (
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
)

// Style is a set of SGR attributes: weights, decorations and at most one of
// the eight foreground colours (docs/decisions/0023-MADR-line-mode-components.md
// D7). Its text form is its tokens joined with "|", or "none". With more than
// one colour, the lowest is used. The zero Style styles nothing.
type Style uint16

// The attributes, and the colours in SGR order, 30 to 37.
const (
	Bold Style = 1 << iota
	Dim
	Italic
	Underline
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "stream"

var styleBits = enum.Bits[Style]{
	Pkg: pkgName, Type: "Style", None: "none",
	Tokens: []string{"bold", "dim", "italic", "underline", "black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"},
}

// String is the set's tokens, joined with "|", or "none".
func (s Style) String() string { return styleBits.String(s) }

// MarshalText is the set's text. A bit with no token is an error.
func (s Style) MarshalText() ([]byte, error) { return styleBits.Marshal(s) }

// UnmarshalText reads what MarshalText wrote, the tokens in any order.
func (s *Style) UnmarshalText(b []byte) error { return styleBits.Unmarshal(b, s) }

// sgr is each attribute's SGR parameter; the first colour set wins.
var sgr = [...]struct {
	bit  Style
	code string
}{
	{Bold, "1"}, {Dim, "2"}, {Italic, "3"}, {Underline, "4"},
	{Black, "30"}, {Red, "31"}, {Green, "32"}, {Yellow, "33"},
	{Blue, "34"}, {Magenta, "35"}, {Cyan, "36"}, {White, "37"},
}

// Apply is text inside the style's SGR sequence and a reset, as a
// terminal draws it; the zero Style, or no text, leaves text as it is.
func (s Style) Apply(text string) string {
	if s == 0 || text == "" {
		return text
	}
	var codes []string
	colour := false
	for _, a := range sgr {
		if s&a.bit == 0 {
			continue
		}
		if a.bit >= Black {
			if colour {
				continue
			}
			colour = true
		}
		codes = append(codes, a.code)
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

// Markup is the Style of each part of a reply the Printer styles. It is
// data: a program builds one from its theme, or turns styling off with the
// zero Markup, which leaves the text as it is, byte for byte. Fields are
// added, never changed, so build one with a keyed literal.
type Markup struct {
	Heading        Style // an ATX or a setext heading's line, its markers kept
	Fence          Style // a fence's opening and closing marker runs
	FenceInfo      Style // the language after a fence's opening run
	Code           Style // an inline code span, its backticks kept
	Marker         Style // a list item's or a quote's marker
	Emphasis       Style // *text* and _text_
	Strong         Style // **text** and __text__
	StrongEmphasis Style // ***text*** and ___text___
	Strike         Style // ~~text~~
	Link           Style // a link's text, its brackets kept
	URL            Style // a link's destination, its parentheses kept, and a definition's
	Match          Style // a matched cell, for pick's WithMarkup
}

// DefaultMarkup is gobble-cli's styling (its render/mdstyle.go): headings
// bold, fences dim with the language in cyan, inline code dim, list and
// quote markers in cyan, emphasis by its markers, a link's text underlined
// and its destination dim (0023-MADR A6 item 1), and a match underlined.
func DefaultMarkup() Markup {
	return Markup{
		Heading: Bold, Fence: Dim, FenceInfo: Cyan, Code: Dim, Marker: Cyan,
		Emphasis: Italic, Strong: Bold, StrongEmphasis: Bold | Italic, Strike: Dim,
		Link: Underline, URL: Dim, Match: Underline,
	}
}
