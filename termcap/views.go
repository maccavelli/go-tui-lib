package termcap

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// KeyboardCaps is the keyboard, as a program should use it.
type KeyboardCaps struct {
	Kitty Fact[Support] // the Kitty keyboard protocol, with the reason no enhancement is asked for
	// Enhancements is what KeyboardFlags returns: what to put in
	// View.KeyboardEnhancements.
	Enhancements tea.KeyboardEnhancements
	Accepted     int  // the flags the terminal last reported accepting
	Releases     bool // ReleasesReported
}

// Keyboard is the keyboard view of c.
func (c Caps) Keyboard() KeyboardCaps {
	k := KeyboardCaps{Kitty: c.KittyKeyboard, Accepted: c.KeyboardFlags, Releases: c.ReleasesReported()}
	reason := keyboardReason(c)
	if reason == "" {
		k.Enhancements.ReportEventTypes = true
	} else if k.Kitty.Reason == "" {
		k.Kitty.Reason = reason
	}
	return k
}

// KeyboardFlags is what a program should put in View.KeyboardEnhancements:
// event types only where the terminal reports them cleanly. Tea always
// asks for disambiguation as well, which a program cannot turn off through
// the View (docs/decisions/0005-MADR-terminal-capabilities-and-services.md
// A4); Keyboard's reason says when the terminal would rather have none.
func KeyboardFlags(c Caps) tea.KeyboardEnhancements { return c.Keyboard().Enhancements }

// ReleasesReported reports whether the terminal accepted key-release
// reports, so a binding may wait for a release.
func (c Caps) ReleasesReported() bool { return c.KeyboardFlags&ansi.KittyReportEventTypes != 0 }

// keyboardReason is why event types are not asked for, or "".
func keyboardReason(c Caps) string {
	brand := c.Brand.Value
	switch {
	case brand == BrandMintty || c.Platform.Value == PlatformMSYS:
		return ReasonMSYSNoKitty
	case c.Platform.Value == PlatformWSL && (brand == BrandVSCode || c.EnvBrand.Value == BrandUnknown):
		return ReasonWSLDeadKeys
	case c.KittyKeyboard.Value == Unsupported:
		return ReasonKittyUnsupported
	case c.KittyKeyboard.Value == Unknown:
		return ReasonKittyUnknown
	case brand == BrandITerm2 || brand == BrandGhostty:
		return ReasonLeaksReleases
	case brand == BrandAlacritty && !alacrittyAfter014(c):
		return ReasonAlacrittyRelease
	case c.Mux.Value == Tmux && c.Tmux.ExtendedKeysFormat != "csi-u":
		return ReasonTmuxExtendedKeys
	}
	return ""
}

// alacrittyAfter014 reports whether DA2 says Alacritty is newer than
// 0.14. Alacritty's DA2 is taken to carry its version packed as
// major*10000 + minor*100 + patch in the second field; without a DA2, it
// fails closed.
func alacrittyAfter014(c Caps) bool {
	return len(c.SecondaryAttributes) >= 2 && c.SecondaryAttributes[1] >= 1500
}

// LinkCaps is hyperlinks, as a program should use them.
type LinkCaps struct {
	OSC8 Fact[Support] // whether OSC 8 links work, from the brand, with a reason when not
}

// linkBrands show OSC 8 links.
var linkBrands = []Brand{
	BrandITerm2, BrandKitty, BrandGhostty, BrandWezTerm, BrandAlacritty, BrandFoot, BrandRio,
	BrandContour, BrandVTE, BrandKonsole, BrandTerminator, BrandWindowsTerminal, BrandVSCode,
	BrandCursor, BrandWindsurf, BrandJetBrains,
}

// Links is the hyperlink view of c.
func (c Caps) Links() LinkCaps {
	var f Fact[Support]
	brand := c.Brand.Value
	switch {
	case c.Mux.Value == Tmux && !atLeast(c.tmuxVersion(), 3, 4):
		f.SetReason(Unsupported, Heuristic, ReasonTmuxLinks)
	case brand == BrandAppleTerminal:
		f.SetReason(Unsupported, Heuristic, ReasonAppleNoOSC8)
	case brand == BrandWarp:
		f.SetReason(Unsupported, Heuristic, ReasonWarpNoOSC8)
	case slices.Contains(linkBrands, brand):
		f.Set(Supported, Heuristic)
	default:
		f.SetReason(Unknown, Heuristic, ReasonUnknownTerminal)
	}
	return LinkCaps{OSC8: f}
}

// tmuxVersion is tmux's version: from TmuxQuery when the program ran it,
// else from an XTVERSION reply of tmux.
func (c Caps) tmuxVersion() string {
	if c.Tmux.Version != "" {
		return c.Tmux.Version
	}
	if v, ok := strings.CutPrefix(c.Terminal.Value, "tmux "); ok {
		return v
	}
	return ""
}

// NotifyCaps is desktop notifications, as a program should send them.
type NotifyCaps struct {
	OSC99  Fact[Support] // the Kitty protocol, from its query
	OSC777 Fact[Support] // rxvt's notify, from the brand
	OSC9   Fact[Support] // iTerm2's sequence, from the brand
	Focus  Fact[Support] // focus reports, which WhenUnfocused needs
}

// Notifications is the notification view of c: OSC 9 on iTerm2, WezTerm
// and Warp, OSC 777 on Ghostty, VTE and foot, neither on others or inside
// Zellij.
func (c Caps) Notifications() NotifyCaps {
	n := NotifyCaps{OSC99: c.DesktopNotify, Focus: c.FocusEvents}
	brand := c.Brand.Value
	switch {
	case c.Mux.Value == Zellij:
		n.OSC777.SetReason(Unsupported, Heuristic, ReasonZellijNoForwarding)
		n.OSC9.SetReason(Unsupported, Heuristic, ReasonZellijNoForwarding)
	case brand == BrandUnknown:
		n.OSC777.SetReason(Unknown, Heuristic, ReasonUnknownTerminal)
		n.OSC9.SetReason(Unknown, Heuristic, ReasonUnknownTerminal)
	default:
		n.OSC777.Set(supportIf(slices.Contains([]Brand{BrandGhostty, BrandVTE, BrandFoot, BrandTerminator}, brand)), Heuristic)
		n.OSC9.Set(supportIf(slices.Contains([]Brand{BrandITerm2, BrandWezTerm, BrandWarp}, brand)), Heuristic)
	}
	return n
}
