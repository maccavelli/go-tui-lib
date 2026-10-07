// Package termevent turns the ultraviolet events Bubble Tea passes to a
// model untranslated into plain values. Tea translates the events it knows
// into its own message types and returns every other event as its
// ultraviolet type, so reading a colour-scheme report, a device-attributes
// reply or a sequence nothing decoded needs ultraviolet's types.
//
// ultraviolet has no tagged release. This package and internal/cells are the
// only ones that import it, so an upstream change is one file's fix, and no
// ultraviolet type reaches an exported API
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md §1 and
// amendment A2). A depguard rule refuses the import anywhere else.
//
// Stability: internal.
package termevent

import (
	"slices"

	uv "github.com/charmbracelet/ultraviolet"
)

// Kind is what an Event reports.
type Kind uint8

const (
	// ColorScheme is a DSR 997 report. Dark says which scheme.
	ColorScheme Kind = iota + 1
	// DeviceAttributes is the primary device attributes (DA1) reply, in
	// Attrs.
	DeviceAttributes
	// SecondaryDeviceAttributes is the secondary device attributes (DA2)
	// reply, in Attrs.
	SecondaryDeviceAttributes
	// KittyGraphics is a Kitty graphics protocol reply. Raw is its payload,
	// "OK" on success.
	KittyGraphics
	// PixelSize is the window's size in pixels, in W and H.
	PixelSize
	// Unknown is a sequence ultraviolet did not decode, in Raw: an unknown
	// CSI, OSC, DCS or APC sequence, or the bytes the reader gave up on at
	// its escape timeout. The last may be the first part of a reply split
	// across reads, whose remaining bytes arrive as key presses.
	Unknown
)

// Event is a decoded pass-through event. Only the fields its Kind names are
// set.
type Event struct {
	Kind  Kind
	Dark  bool   // ColorScheme
	Attrs []int  // DeviceAttributes, SecondaryDeviceAttributes
	W, H  int    // PixelSize
	Raw   string // Unknown: the bytes; KittyGraphics: the payload
}

// Decode turns an event tea passed through untranslated into an Event, or
// reports false for any other message.
func Decode(msg any) (Event, bool) {
	switch e := msg.(type) {
	case uv.DarkColorSchemeEvent:
		return Event{Kind: ColorScheme, Dark: true}, true
	case uv.LightColorSchemeEvent:
		return Event{Kind: ColorScheme}, true
	case uv.PrimaryDeviceAttributesEvent:
		return Event{Kind: DeviceAttributes, Attrs: slices.Clone([]int(e))}, true
	case uv.SecondaryDeviceAttributesEvent:
		return Event{Kind: SecondaryDeviceAttributes, Attrs: slices.Clone([]int(e))}, true
	case uv.KittyGraphicsEvent:
		return Event{Kind: KittyGraphics, Raw: string(e.Payload)}, true
	case uv.PixelSizeEvent:
		return Event{Kind: PixelSize, W: e.Width, H: e.Height}, true
	case uv.UnknownCsiEvent:
		return Event{Kind: Unknown, Raw: string(e)}, true
	case uv.UnknownOscEvent:
		return Event{Kind: Unknown, Raw: string(e)}, true
	case uv.UnknownDcsEvent:
		return Event{Kind: Unknown, Raw: string(e)}, true
	case uv.UnknownApcEvent:
		return Event{Kind: Unknown, Raw: string(e)}, true
	case uv.UnknownEvent:
		return Event{Kind: Unknown, Raw: string(e)}, true
	}
	return Event{}, false
}
