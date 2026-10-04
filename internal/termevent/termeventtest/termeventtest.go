// Package termeventtest builds the ultraviolet events Bubble Tea passes to a
// model untranslated, as tea messages, for tests outside internal/termevent.
// A depguard rule keeps ultraviolet under internal/cells and
// internal/termevent, test files included, so a test that feeds these
// events to a model builds them here
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md A3).
//
// Each constructor builds an event termevent.Decode reads.
package termeventtest

import (
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// DarkColorScheme is a DSR 997 report of a dark scheme.
func DarkColorScheme() tea.Msg { return uv.DarkColorSchemeEvent{} }

// LightColorScheme is a DSR 997 report of a light scheme.
func LightColorScheme() tea.Msg { return uv.LightColorSchemeEvent{} }

// DeviceAttributes is a primary device attributes (DA1) reply.
func DeviceAttributes(attrs ...int) tea.Msg { return uv.PrimaryDeviceAttributesEvent(attrs) }

// SecondaryDeviceAttributes is a secondary device attributes (DA2) reply.
func SecondaryDeviceAttributes(attrs ...int) tea.Msg {
	return uv.SecondaryDeviceAttributesEvent(attrs)
}

// KittyGraphics is a Kitty graphics protocol reply with payload, "OK" on
// success.
func KittyGraphics(payload string) tea.Msg { return uv.KittyGraphicsEvent{Payload: []byte(payload)} }

// PixelSize is the window's size in pixels.
func PixelSize(w, h int) tea.Msg { return uv.PixelSizeEvent{Width: w, Height: h} }

// UnknownCsi is a CSI sequence ultraviolet did not decode.
func UnknownCsi(seq string) tea.Msg { return uv.UnknownCsiEvent(seq) }

// UnknownOsc is an OSC sequence ultraviolet did not decode.
func UnknownOsc(seq string) tea.Msg { return uv.UnknownOscEvent(seq) }

// UnknownDcs is a DCS sequence ultraviolet did not decode.
func UnknownDcs(seq string) tea.Msg { return uv.UnknownDcsEvent(seq) }

// UnknownApc is an APC sequence ultraviolet did not decode.
func UnknownApc(seq string) tea.Msg { return uv.UnknownApcEvent(seq) }

// Unknown is the bytes the reader gave up on at its escape timeout, such as
// the first part of a reply split across reads.
func Unknown(seq string) tea.Msg { return uv.UnknownEvent(seq) }
