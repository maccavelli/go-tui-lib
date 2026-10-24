package termcap

import (
	"image/color"
	"strconv"
	"strings"
)

// xParseColor reads a colour in X11's XParseColor grammar, as a terminal
// answers OSC 10, 11 and 4 (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
// A2.4):
//
//   - rgb:R/G/B, each channel one to four hex digits, scaled to 16 bits as
//     v * 0xffff / (16^n - 1);
//   - #RGB, #RRGGBB, #RRRGGGBBB or #RRRRGGGGBBBB, each channel's digits the
//     most significant bits of 16, not scaled, so #3a7 is #3000a0007000;
//   - urxvt's rgba:R/G/B/A, read as rgb: with the alpha dropped.
//
// The result is a color.RGBA64, so a 16-bit reply keeps its precision.
// Anything else is not a colour.
func xParseColor(s string) (color.Color, bool) {
	if h, ok := strings.CutPrefix(s, "#"); ok {
		n := len(h) / 3
		if len(h)%3 != 0 || n < 1 || n > 4 {
			return nil, false
		}
		var ch [3]uint16
		for i := range ch {
			v, err := strconv.ParseUint(h[i*n:(i+1)*n], 16, 16)
			if err != nil {
				return nil, false
			}
			ch[i] = uint16(v << (16 - 4*n)) //nolint:gosec // n hex digits fit in 4n bits
		}
		return color.RGBA64{R: ch[0], G: ch[1], B: ch[2], A: 0xffff}, true
	}
	var parts []string
	switch {
	case strings.HasPrefix(s, "rgb:"):
		parts = strings.Split(s[len("rgb:"):], "/")
		if len(parts) != 3 {
			return nil, false
		}
	case strings.HasPrefix(s, "rgba:"):
		parts = strings.Split(s[len("rgba:"):], "/")
		if len(parts) != 4 {
			return nil, false
		}
	default:
		return nil, false
	}
	var ch [4]uint16
	for i, p := range parts {
		// ParseUint in base 16 takes no sign, prefix or underscore.
		v, err := strconv.ParseUint(p, 16, 16)
		if err != nil || len(p) > 4 {
			return nil, false
		}
		ch[i] = uint16(v * 0xffff / (1<<(4*len(p)) - 1)) //nolint:gosec // at most 0xffff
	}
	return color.RGBA64{R: ch[0], G: ch[1], B: ch[2], A: 0xffff}, true
}
