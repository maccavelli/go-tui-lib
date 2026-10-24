package termcap

import "testing"

// The Bubble Tea cases of this file moved to termcaptea with the Prober
// (docs/decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md Step 8).

func TestAlacrittyVersionFromDA2(t *testing.T) {
	for _, c := range []struct {
		da2    []int
		events bool
	}{
		{nil, false},
		{[]int{0, 1400, 1}, false},
		{[]int{0, 1499, 1}, false},
		{[]int{0, 1500, 1}, true},
		{[]int{0, 1601, 1}, true},
	} {
		var caps Caps
		caps.Brand.Set(BrandAlacritty, Environment)
		caps.KittyKeyboard.Set(Supported, Queried)
		caps.SecondaryAttributes = c.da2
		if got := caps.Keyboard().Request&2 != 0; got != c.events {
			t.Errorf("DA2 %v: event types %v, want %v", c.da2, got, c.events)
		}
	}
}
