// Package glyph holds the glyph tables every go-tui-lib package draws from.
//
// Each glyph has a Unicode form and an ASCII twin, and each form is exactly
// one terminal cell wide, so a view laid out for one set fits the other
// (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rule 3). The
// one exception is Set.Keys, the key labels: their Unicode forms are one
// cell, and their ASCII twins are words, such as "opt+" and "up"
// (docs/decisions/0007-MADR-keymap-engine.md A8), so a view measures them.
// A package never writes a glyph literal of its own: it reads the Set it
// was given, which is Unicode() on a UTF-8 terminal and ASCII() elsewhere.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package glyph

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

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
	// Prompt starts an input line, as a text input's prompt
	// (docs/decisions/0008-MADR-command-palette.md A3 item 8).
	Prompt string
	// Mask stands for each character of a secret as it is typed.
	Mask string
	// Done, Failed, Running and Pending mark a step's or an item's state:
	// it succeeded, it failed, it is under way, or it has not started
	// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md D8).
	// Each has its own shape in both sets, so the state is not carried by
	// colour alone (rule 4).
	Done, Failed, Running, Pending string
	// Heading starts an item's first line, such as a tool call's header.
	// It may draw as Focus does; it is its own field, so that a theme that
	// changes one changes only that one.
	Heading string
	// MeterFill and MeterTrack draw a meter: its filled cells and its empty
	// ones.
	MeterFill, MeterTrack string
	// Control stands where an escape sequence was removed from untrusted
	// text, for an audit or approval view
	// (docs/decisions/0009-MADR-streaming-content-engine.md §1).
	Control string
	// Keys are the modifier and arrow labels a key binding is shown with.
	Keys KeyGlyphs
}

// KeyGlyphs are the labels of the macOS modifier keys and of the arrow
// keys. Each Unicode form is one cell; each ASCII twin is a word, which
// may be wider. A modifier's ASCII twin ends in "+", so a label is its
// modifiers and its key written together, in either set.
type KeyGlyphs struct {
	// Option, Command, Control and Shift are the macOS modifiers.
	Option, Command, Control, Shift string
	// Up, Down, Left and Right are the arrow keys.
	Up, Down, Left, Right string
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
		Prompt: "❯",
		Mask:   "•",
		Done:   "✓", Failed: "✗", Running: "●", Pending: "○",
		Heading:   "▸",
		MeterFill: "━", MeterTrack: "╌",
		Control: "␛",
		Keys: KeyGlyphs{
			Option: "⌥", Command: "⌘", Control: "⌃", Shift: "⇧",
			Up: "↑", Down: "↓", Left: "←", Right: "→",
		},
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
		Prompt: ">",
		Mask:   "*",
		Done:   "+", Failed: "x", Running: "*", Pending: "-",
		Heading:   ">",
		MeterFill: "#", MeterTrack: "-",
		Control: "^",
		Keys: KeyGlyphs{
			Option: "opt+", Command: "cmd+", Control: "ctrl+", Shift: "shift+",
			Up: "up", Down: "down", Left: "left", Right: "right",
		},
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
