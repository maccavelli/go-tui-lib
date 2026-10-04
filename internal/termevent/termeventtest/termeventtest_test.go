package termeventtest

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/internal/termevent"
)

// TestEveryEventDecodes checks that each constructor builds the event
// termevent.Decode reads, with its values.
func TestEveryEventDecodes(t *testing.T) {
	cases := []struct {
		msg  tea.Msg
		want termevent.Event
	}{
		{DarkColorScheme(), termevent.Event{Kind: termevent.ColorScheme, Dark: true}},
		{LightColorScheme(), termevent.Event{Kind: termevent.ColorScheme}},
		{DeviceAttributes(62, 4), termevent.Event{Kind: termevent.DeviceAttributes, Attrs: []int{62, 4}}},
		{SecondaryDeviceAttributes(1, 95, 0), termevent.Event{Kind: termevent.SecondaryDeviceAttributes, Attrs: []int{1, 95, 0}}},
		{KittyGraphics("OK"), termevent.Event{Kind: termevent.KittyGraphics, Raw: "OK"}},
		{PixelSize(800, 600), termevent.Event{Kind: termevent.PixelSize, W: 800, H: 600}},
		{UnknownCsi("\x1b[0n"), termevent.Event{Kind: termevent.Unknown, Raw: "\x1b[0n"}},
		{UnknownOsc("\x1b]99;;\x07"), termevent.Event{Kind: termevent.Unknown, Raw: "\x1b]99;;\x07"}},
		{UnknownDcs("\x1bP>|x\x1b\\"), termevent.Event{Kind: termevent.Unknown, Raw: "\x1bP>|x\x1b\\"}},
		{UnknownApc("\x1b_Gi=1;OK\x1b\\"), termevent.Event{Kind: termevent.Unknown, Raw: "\x1b_Gi=1;OK\x1b\\"}},
		{Unknown("\x1b[?62;"), termevent.Event{Kind: termevent.Unknown, Raw: "\x1b[?62;"}},
	}
	for _, c := range cases {
		got, ok := termevent.Decode(c.msg)
		if !ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Decode(%T) = %+v, %v; want %+v", c.msg, got, ok, c.want)
		}
	}
}
