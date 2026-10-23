// Package safetext makes untrusted text safe to put on a terminal: model
// output, tool output, file names, a terminal's replies
// (docs/decisions/0009-MADR-streaming-content-engine.md §2, with its
// amendments A1 and A4; docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
// D9). It imports only the standard library and glyph, so a line-mode CLI
// links no TUI framework to use it.
//
// Clean cleans a string by Rules; a Sanitizer cleans a stream, and writing
// a and then b through one gives what Clean gives for a+b. IsClean reports
// whether Clean would change a string, without cleaning it. The presets
// Text, Strict, Line, Command and StatusLine are the Rules callers
// normally use.
//
// # The grammar
//
// Escape sequences are read as ECMA-48 defines them, with the DEC parser's
// recovery rules (vt100.net's state machine, which the Charm stack's x/ansi
// also follows): CSI, OSC, DCS, SOS, PM, APC and the two- and
// three-character escapes. Every byte a sequence holds is removed, not just
// its ESC. CAN and SUB abort a sequence; an ESC inside one starts a new
// one. An OSC ends at BEL, at ST (ESC \) or at U+009C; a DCS, SOS, PM or APC
// ends at ST, and at BEL as well. A string that runs past Rules.MaxString
// bytes is abandoned, so a lost terminator hides at most that much text.
// The C1 code points U+0080 to U+009F are read as their 7-bit forms:
// U+009B is CSI, U+009D is OSC, U+009C is ST.
//
// # What survives
//
// Printable text survives. Tabs and line breaks survive as the Rules say.
// Every other C0 control, DEL and every C1 control is removed, and so are
// U+200B, U+FEFF, U+2060 to U+2064, U+00AD and the tag characters
// U+E0000 to U+E007F; the Bidi_Control set (U+061C, U+200E, U+200F,
// U+202A to U+202E, U+2066 to U+2069) is removed unless Rules.Bidi keeps
// it. Under Rules.Escape what would be removed is shown escaped instead.
// Invalid UTF-8 becomes U+FFFD, one for each maximal subpart (the Unicode
// Standard, §3.9.6), under every Rules.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package safetext
