// Package glyph holds the glyph tables every go-tui-lib package draws from.
//
// Each glyph has a Unicode form and an ASCII twin, and each form is exactly
// one terminal cell wide, so a view laid out for one set fits the other
// (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rule 3). A
// package never writes a glyph literal of its own: it reads the Set it was
// given, which is Unicode() on a UTF-8 terminal and ASCII() elsewhere.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package glyph

import "github.com/maccavelli/go-tui-lib/internal/enum"

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "glyph"

// Border is one border style: the same thirteen glyphs, in the same order, as
// a lipgloss.Border, so the theme package converts it field for field.
type Border struct {
	Top, Bottom, Left, Right                   string
	TopLeft, TopRight, BottomLeft, BottomRight string
	// MiddleLeft and MiddleRight join an inner rule to the frame; Middle
	// is where two inner rules cross; MiddleTop and MiddleBottom join an
	// inner column to the frame.
	MiddleLeft, MiddleRight, Middle, MiddleTop, MiddleBottom string
}

// Set is every glyph go-tui-lib draws.
type Set struct {
	// Light, Rounded, Heavy and Double are the border styles.
	Light, Rounded, Heavy, Double Border
	// SeparatorVertical and SeparatorHorizontal divide panes that have no
	// border; SeparatorCross is where two separators cross.
	SeparatorVertical, SeparatorHorizontal, SeparatorCross string
	// SeparatorTeeDown, SeparatorTeeUp, SeparatorTeeRight and
	// SeparatorTeeLeft are where one separator ends at another: a line
	// running down, up, right or left from the separator it meets.
	SeparatorTeeDown, SeparatorTeeUp, SeparatorTeeRight, SeparatorTeeLeft string
	// Focus marks the focused pane's title, so focus is visible without
	// colour (rule 4).
	Focus string
	// Ellipsis ends a truncated title or line.
	Ellipsis string
	// ScrollUp and ScrollDown show that content continues above or below;
	// ScrollThumb and ScrollTrack draw a scroll bar.
	ScrollUp, ScrollDown, ScrollThumb, ScrollTrack string
	// Bullet starts a list item.
	Bullet string
	// BadgeOpen and BadgeClose enclose a pane's badge, such as an unread
	// count.
	BadgeOpen, BadgeClose string
}

// Unicode returns the set for a UTF-8 terminal.
func Unicode() Set {
	return Set{
		Light: Border{
			Top: "─", Bottom: "─", Left: "│", Right: "│",
			TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘",
			MiddleLeft: "├", MiddleRight: "┤", Middle: "┼", MiddleTop: "┬", MiddleBottom: "┴",
		},
		Rounded: Border{
			Top: "─", Bottom: "─", Left: "│", Right: "│",
			TopLeft: "╭", TopRight: "╮", BottomLeft: "╰", BottomRight: "╯",
			MiddleLeft: "├", MiddleRight: "┤", Middle: "┼", MiddleTop: "┬", MiddleBottom: "┴",
		},
		Heavy: Border{
			Top: "━", Bottom: "━", Left: "┃", Right: "┃",
			TopLeft: "┏", TopRight: "┓", BottomLeft: "┗", BottomRight: "┛",
			MiddleLeft: "┣", MiddleRight: "┫", Middle: "╋", MiddleTop: "┳", MiddleBottom: "┻",
		},
		Double: Border{
			Top: "═", Bottom: "═", Left: "║", Right: "║",
			TopLeft: "╔", TopRight: "╗", BottomLeft: "╚", BottomRight: "╝",
			MiddleLeft: "╠", MiddleRight: "╣", Middle: "╬", MiddleTop: "╦", MiddleBottom: "╩",
		},
		SeparatorVertical: "│", SeparatorHorizontal: "─", SeparatorCross: "┼",
		SeparatorTeeDown: "┬", SeparatorTeeUp: "┴", SeparatorTeeRight: "├", SeparatorTeeLeft: "┤",
		Focus:    "▸",
		Ellipsis: "…",
		ScrollUp: "▲", ScrollDown: "▼", ScrollThumb: "┃", ScrollTrack: "│",
		Bullet:    "•",
		BadgeOpen: "[", BadgeClose: "]",
	}
}

// ASCII returns the set for a terminal that cannot be trusted with anything
// beyond ASCII. Every border style becomes the same plain box, except Double,
// whose rules are "=".
func ASCII() Set {
	box := Border{
		Top: "-", Bottom: "-", Left: "|", Right: "|",
		TopLeft: "+", TopRight: "+", BottomLeft: "+", BottomRight: "+",
		MiddleLeft: "+", MiddleRight: "+", Middle: "+", MiddleTop: "+", MiddleBottom: "+",
	}
	double := box
	double.Top, double.Bottom = "=", "="
	return Set{
		Light: box, Rounded: box, Heavy: box, Double: double,
		SeparatorVertical: "|", SeparatorHorizontal: "-", SeparatorCross: "+",
		SeparatorTeeDown: "+", SeparatorTeeUp: "+", SeparatorTeeRight: "+", SeparatorTeeLeft: "+",
		Focus:    ">",
		Ellipsis: "~",
		ScrollUp: "^", ScrollDown: "v", ScrollThumb: "#", ScrollTrack: "|",
		Bullet:    "*",
		BadgeOpen: "[", BadgeClose: "]",
	}
}

// For returns TierUnicode.Set() when utf8 is true, and TierASCII.Set()
// otherwise: Unicode() or ASCII().
func For(utf8 bool) Set {
	if utf8 {
		return TierUnicode.Set()
	}
	return TierASCII.Set()
}

// Tier is a glyph table's tier: Unicode, the legacy console's, or ASCII
// (docs/decisions/0014-MADR-native-integration-api.md W2 and A1.2). Its
// text form is a stable lowercase token, so a flag or a config file can
// name it.
type Tier uint8

// The tiers, richest first.
const (
	// TierUnicode is the Unicode table, for a UTF-8 terminal.
	TierUnicode Tier = iota
	// TierLegacy is the legacy console's table, for a console that draws
	// only its code page. Until theme v2 fills it, its Set is ASCII().
	TierLegacy
	// TierASCII is the ASCII table.
	TierASCII
)

var tierNames = enum.Names[Tier]{
	Pkg: pkgName, Type: "Tier",
	Tokens: []string{"unicode", "legacy", "ascii"},
}

// Set is the tier's glyph table. TierLegacy, and a tier with no name, give
// ASCII(), which every terminal draws.
func (t Tier) Set() Set {
	if t == TierUnicode {
		return Unicode()
	}
	return ASCII()
}

// String is the tier's token: "unicode", "legacy" or "ascii".
func (t Tier) String() string { return tierNames.String(t) }

// MarshalText is the tier's token. A tier with no token is an error.
func (t Tier) MarshalText() ([]byte, error) { return tierNames.Marshal(t) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (t *Tier) UnmarshalText(b []byte) error {
	return tierNames.Unmarshal(b, t)
}
