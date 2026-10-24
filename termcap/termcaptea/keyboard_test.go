package termcaptea

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// TestKeyboardEnhancements: Request 2 asks for event types, 0 for none,
// for each keyboard reason of termcap's views.go, and nothing more.
func TestKeyboardEnhancements(t *testing.T) {
	csiU := termcap.TmuxFacts{Known: true, Version: "3.4", ExtendedKeysFormat: "csi-u"}
	view := func(brand termcap.Brand, platform termcap.Platform, mux termcap.Mux, k termcap.Support, tmux termcap.TmuxFacts) termcap.Caps {
		var c termcap.Caps
		c.Brand.Set(brand, termcap.Environment)
		c.EnvBrand.Set(brand, termcap.Environment)
		c.Platform.Set(platform, termcap.Environment)
		c.Mux.Set(mux, termcap.Environment)
		c.KittyKeyboard.Set(k, termcap.Queried)
		c.Tmux = tmux
		return c
	}
	var unknown termcap.Caps
	unknown.Brand.Set(termcap.BrandKitty, termcap.Environment)
	cases := []struct {
		name   string
		c      termcap.Caps
		events bool
	}{
		{"kitty", view(termcap.BrandKitty, termcap.PlatformNative, termcap.NoMux, termcap.Supported, termcap.TmuxFacts{}), true},
		{"tmux, csi-u", view(termcap.BrandKitty, termcap.PlatformNative, termcap.Tmux, termcap.Supported, csiU), true},
		{termcap.ReasonLeaksReleases, view(termcap.BrandITerm2, termcap.PlatformNative, termcap.NoMux, termcap.Supported, termcap.TmuxFacts{}), false},
		{termcap.ReasonAlacrittyRelease, view(termcap.BrandAlacritty, termcap.PlatformNative, termcap.NoMux, termcap.Supported, termcap.TmuxFacts{}), false},
		{termcap.ReasonTmuxExtendedKeys, view(termcap.BrandKitty, termcap.PlatformNative, termcap.Tmux, termcap.Supported, termcap.TmuxFacts{}), false},
		{termcap.ReasonMSYSNoKitty, view(termcap.BrandMintty, termcap.PlatformNative, termcap.NoMux, termcap.Supported, termcap.TmuxFacts{}), false},
		{termcap.ReasonWSLDeadKeys, view(termcap.BrandVSCode, termcap.PlatformWSL, termcap.NoMux, termcap.Supported, termcap.TmuxFacts{}), false},
		{termcap.ReasonKittyUnsupported, view(termcap.BrandKitty, termcap.PlatformNative, termcap.NoMux, termcap.Unsupported, termcap.TmuxFacts{}), false},
		{termcap.ReasonKittyUnknown, unknown, false},
	}
	for _, c := range cases {
		got := KeyboardEnhancements(c.c)
		if got != (tea.KeyboardEnhancements{ReportEventTypes: c.events}) {
			t.Errorf("%s: %+v, want event types %v and nothing more", c.name, got, c.events)
		}
		if want := c.c.Keyboard().Request&2 != 0; got.ReportEventTypes != want {
			t.Errorf("%s: event types %v, Request %d", c.name, got.ReportEventTypes, c.c.Keyboard().Request)
		}
	}
}
