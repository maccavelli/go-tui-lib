package inputfilter_test

import (
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// x10 is an X10 report of button byte cb at 1-based column col and row
// row.
func x10(cb byte, col, row int) string {
	return "\x1b[M" + string([]byte{cb + 32, byte(32 + col), byte(32 + row)})
}

// TestParseReport: SGR and X10 mouse reports and focus reports, read by
// ultraviolet's rules (decoder.go parseMouseButton); and what is not one
// refused.
func TestParseReport(t *testing.T) {
	type r = inputfilter.Report
	for _, c := range []struct {
		raw  string
		want r
		ok   bool
	}{
		{"\x1b[<0;10;5M", r{Kind: inputfilter.MousePress, Button: inputfilter.ButtonLeft, X: 9, Y: 4}, true},
		{"\x1b[<0;10;5m", r{Kind: inputfilter.MouseRelease, Button: inputfilter.ButtonLeft, X: 9, Y: 4}, true},
		{"\x1b[<2;1;1M", r{Kind: inputfilter.MousePress, Button: inputfilter.ButtonRight}, true},
		{"\x1b[<32;3;4M", r{Kind: inputfilter.MouseMotion, Button: inputfilter.ButtonLeft, X: 2, Y: 3}, true},
		{"\x1b[<35;3;4M", r{Kind: inputfilter.MouseMotion, Button: inputfilter.ButtonNone, X: 2, Y: 3}, true},
		{"\x1b[<64;5;6M", r{Kind: inputfilter.MouseWheel, Button: inputfilter.WheelUp, X: 4, Y: 5}, true},
		{"\x1b[<65;5;6M", r{Kind: inputfilter.MouseWheel, Button: inputfilter.WheelDown, X: 4, Y: 5}, true},
		{"\x1b[<67;5;6M", r{Kind: inputfilter.MouseWheel, Button: inputfilter.WheelRight, X: 4, Y: 5}, true},
		{"\x1b[<128;1;1M", r{Kind: inputfilter.MousePress, Button: inputfilter.ButtonBackward}, true},
		{"\x1b[<131;1;1M", r{Kind: inputfilter.MousePress, Button: inputfilter.Button11}, true},
		{"\x1b[<28;300;200M", r{Kind: inputfilter.MousePress, Button: inputfilter.ButtonLeft, X: 299, Y: 199,
			Mod: termkey.ModShift | termkey.ModAlt | termkey.ModCtrl}, true},
		{x10(0, 96, 1), r{Kind: inputfilter.MousePress, Button: inputfilter.ButtonLeft, X: 95}, true},
		{x10(3, 10, 2), r{Kind: inputfilter.MouseRelease, Button: inputfilter.ButtonNone, X: 9, Y: 1}, true},
		{x10(32, 10, 2), r{Kind: inputfilter.MouseMotion, Button: inputfilter.ButtonLeft, X: 9, Y: 1}, true},
		{x10(65, 223, 223), r{Kind: inputfilter.MouseWheel, Button: inputfilter.WheelDown, X: 222, Y: 222}, true},
		{"\x1b[I", r{Kind: inputfilter.FocusIn}, true},
		{"\x1b[O", r{Kind: inputfilter.FocusOut}, true},
		{"\x1b[<0;10;5", r{}, false},
		{"\x1b[<0;10M", r{}, false},
		{"\x1b[<0;0;5M", r{}, false},
		{"\x1b[<0;+1;5M", r{}, false},
		{"\x1b[<0;;5M", r{}, false},
		{"\x1b[<0;99999999;5M", r{}, false},
		{"\x1b[<0;10;5X", r{}, false},
		{"\x1b[M ", r{}, false},
		{"\x1b[?62;22c", r{}, false},
		{"abc", r{}, false},
	} {
		got, ok := inputfilter.ParseReport(c.raw)
		if ok != c.ok || got != c.want {
			t.Errorf("ParseReport(%q) = %+v, %v; want %+v, %v", c.raw, got, ok, c.want, c.ok)
		}
	}
}
