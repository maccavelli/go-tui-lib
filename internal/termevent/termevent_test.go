package termevent

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

func TestDecode(t *testing.T) {
	cases := []struct {
		name string
		msg  any
		want Event
	}{
		{"dark", uv.DarkColorSchemeEvent{}, Event{Kind: ColorScheme, Dark: true}},
		{"light", uv.LightColorSchemeEvent{}, Event{Kind: ColorScheme}},
		{"DA1", uv.PrimaryDeviceAttributesEvent{62, 4, 22}, Event{Kind: DeviceAttributes, Attrs: []int{62, 4, 22}}},
		{"DA2", uv.SecondaryDeviceAttributesEvent{1, 95, 0}, Event{Kind: SecondaryDeviceAttributes, Attrs: []int{1, 95, 0}}},
		{"kitty graphics", uv.KittyGraphicsEvent{Payload: []byte("OK")}, Event{Kind: KittyGraphics, Raw: "OK"}},
		{"pixel size", uv.PixelSizeEvent{Width: 800, Height: 600}, Event{Kind: PixelSize, W: 800, H: 600}},
		{"unknown CSI", uv.UnknownCsiEvent("\x1b[?1;2x"), Event{Kind: Unknown, Raw: "\x1b[?1;2x"}},
		{"unknown OSC", uv.UnknownOscEvent("\x1b]99;i=x:p=?;p=title\x1b\\"), Event{Kind: Unknown, Raw: "\x1b]99;i=x:p=?;p=title\x1b\\"}},
		{"unknown DCS", uv.UnknownDcsEvent("\x1bP>|tmux 3.4\x1b\\"), Event{Kind: Unknown, Raw: "\x1bP>|tmux 3.4\x1b\\"}},
		{"unknown APC", uv.UnknownApcEvent("\x1b_Gi=1;OK\x1b\\"), Event{Kind: Unknown, Raw: "\x1b_Gi=1;OK\x1b\\"}},
		// The first part of a reply split across reads, as the 0005-PLAN
		// Step 1 spike saw it.
		{"split reply", uv.UnknownEvent("\x1b[?62;"), Event{Kind: Unknown, Raw: "\x1b[?62;"}},
	}
	for _, c := range cases {
		got, ok := Decode(c.msg)
		if !ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Decode(%T) = %+v, %v; want %+v, true", c.name, c.msg, got, ok, c.want)
		}
	}
}

func TestDecodeRefusesEverythingElse(t *testing.T) {
	for _, msg := range []any{
		nil,
		"\x1b[c",
		tea.KeyPressMsg{Code: 'c', Text: "c"},
		tea.WindowSizeMsg{Width: 80, Height: 24},
		uv.WindowSizeEvent{Width: 80, Height: 24},
		uv.KeyPressEvent{Code: 'c'},
	} {
		if got, ok := Decode(msg); ok {
			t.Errorf("Decode(%T) = %+v, true; want false", msg, got)
		}
	}
}

func TestDecodeCopiesAttributes(t *testing.T) {
	src := uv.PrimaryDeviceAttributesEvent{62, 4}
	e, _ := Decode(src)
	src[0] = 1
	if e.Attrs[0] != 62 {
		t.Fatalf("Attrs shares the event's array: %v", e.Attrs)
	}
}
