package safetext

import "github.com/maccavelli/go-tui-lib/glyph"

// DefaultMaxString is the most bytes an OSC, DCS, SOS, PM or APC string
// may hold before it is abandoned, when Rules.MaxString is zero.
const DefaultMaxString = 64 << 10

// Rules say what survives cleaning. The zero value is the strictest:
// printable text only, everything else removed. Build a Rules with keyed
// fields: a field added later keeps the zero value's behaviour.
type Rules struct {
	// Tabs keeps \t.
	Tabs bool
	// Newlines keeps \n. \r\n becomes \n, a lone \r is removed, and U+2028
	// and U+2029 become \n.
	Newlines bool
	// Flatten turns each run of \r, \n, \t, U+2028 and U+2029 into one
	// space. It wins over Tabs, Newlines and Escape.
	Flatten bool
	// Bidi keeps the Bidi_Control set, for prose in right-to-left scripts.
	Bidi bool
	// Visible puts Glyphs.Control in place of each escape sequence it
	// removes, for an audit or approval view.
	Visible bool
	// Glyphs is where Visible's mark comes from. A Set whose Control is
	// empty, the zero Set among them, gives glyph.Unicode()'s.
	Glyphs glyph.Set
	// MaxString is the most bytes an OSC, DCS, SOS, PM or APC string may
	// hold before it is abandoned; 0 means DefaultMaxString.
	MaxString int
	// Escape shows what the rules would remove instead of removing it:
	// each such character as strconv.QuoteToASCII writes it, without the
	// quotes, such as \x1b, \n or \u202e. Escape sequences are not parsed
	// then: an ESC is a control like any other, so a sequence shows as its
	// ESC escaped and its other bytes as text. KeepSGR and Visible have no
	// effect under Escape.
	Escape bool
	// KeepSGR keeps each SGR, the CSI that sets colour and style, of at
	// most 32 parameters, and each OSC 8 hyperlink whose parameters and
	// target are printable ASCII. Every other sequence is still removed.
	KeepSGR bool
	// LinkSchemes, with KeepSGR, keeps an OSC 8 link only when its target's
	// scheme, compared without case, is one of these, and the target has a
	// host or, as mailto has, an opaque part, as RFC 3986 reads them: after
	// the scheme's ":", either "//" and an authority whose host is not
	// empty, or a part that does not begin with "/". An OSC 8 that closes a
	// link, with no target, is always kept. Empty keeps every link KeepSGR
	// keeps.
	LinkSchemes []string
}

// Text is for prose, such as a model's reply: tabs, line breaks and the
// bidirectional controls kept.
func Text() Rules { return Rules{Tabs: true, Newlines: true, Bidi: true} }

// Strict is for text shown for approval, such as a file's contents: tabs
// and line breaks kept, the bidirectional controls removed.
func Strict() Rules { return Rules{Tabs: true, Newlines: true} }

// Line is for one line: each run of line breaks and tabs one space, the
// bidirectional controls removed.
func Line() Rules { return Rules{Flatten: true} }

// Command is for a command or a URL shown for approval: nothing removed
// silently. Every control, each bidirectional and format character, and
// each escape sequence's ESC is shown escaped, on one line, so the text
// cannot repaint the terminal and a Trojan Source reorder cannot make what
// is shown differ from what runs.
func Command() Rules { return Rules{Escape: true} }

// StatusLine is for a program's output in a status row: one line, SGR
// kept, and OSC 8 links kept only for http, https and mailto targets with
// a host or an address. It is the one preset that keeps any sequence.
func StatusLine() Rules {
	return Rules{Flatten: true, KeepSGR: true, LinkSchemes: []string{"http", "https", "mailto"}}
}
