package inputfilter

import (
	"strconv"
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// ReportKind is what a Report reports.
type ReportKind uint8

const (
	// MousePress is a button pressed. The zero value.
	MousePress ReportKind = iota
	// MouseRelease is a button released.
	MouseRelease
	// MouseMotion is the pointer moved, with a button held or none.
	MouseMotion
	// MouseWheel is a wheel turned, or a trackpad scrolled.
	MouseWheel
	// FocusIn is the terminal gaining focus.
	FocusIn
	// FocusOut is the terminal losing focus.
	FocusOut
)

var reportKindNames = enum.Names[ReportKind]{
	Pkg: pkgName, Type: "ReportKind",
	Tokens: []string{"press", "release", "motion", "wheel", "focus-in", "focus-out"},
}

// String is the kind's token.
func (k ReportKind) String() string { return reportKindNames.String(k) }

// MarshalText is the kind's token. A kind with no token is an error.
func (k ReportKind) MarshalText() ([]byte, error) { return reportKindNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *ReportKind) UnmarshalText(b []byte) error { return reportKindNames.Unmarshal(b, k) }

// MouseButton is a mouse button. Its values are ultraviolet's, which
// Bubble Tea's tea.MouseButton has, so that a program converts one value
// for value; its tokens are x/ansi's names.
type MouseButton uint8

// The buttons.
const (
	ButtonNone MouseButton = iota
	ButtonLeft
	ButtonMiddle
	ButtonRight
	WheelUp
	WheelDown
	WheelLeft
	WheelRight
	ButtonBackward
	ButtonForward
	Button10
	Button11
)

var mouseButtonNames = enum.Names[MouseButton]{
	Pkg: pkgName, Type: "MouseButton",
	Tokens: []string{
		"none", "left", "middle", "right", "wheelup", "wheeldown", "wheelleft", "wheelright",
		"backward", "forward", "button10", "button11",
	},
}

// String is the button's token.
func (b MouseButton) String() string { return mouseButtonNames.String(b) }

// MarshalText is the button's token. A button with no token is an error.
func (b MouseButton) MarshalText() ([]byte, error) { return mouseButtonNames.Marshal(b) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (b *MouseButton) UnmarshalText(t []byte) error { return mouseButtonNames.Unmarshal(t, b) }

// Report is a terminal report the stages decode: a mouse event, in SGR or
// X10 form, or a change of focus. X and Y are cells, from 0; Button and
// Mod are a mouse event's.
type Report struct {
	Kind   ReportKind
	Button MouseButton
	X, Y   int
	Mod    termkey.Mod
}

// maxCoord bounds an SGR report's numbers, which are untrusted.
const maxCoord = 1 << 16

// ParseReport reads raw as a Report: an SGR mouse report, CSI < Cb ; Cx ;
// Cy M or m; an X10 one, CSI M and three bytes, each its value plus 32;
// or a focus report, CSI I or CSI O. The button's bits are xterm's, read
// as ultraviolet reads them, so that a report reaches a program as Bubble
// Tea's own decoding would give it (ultraviolet decoder.go,
// parseMouseButton).
func ParseReport(raw string) (Report, bool) {
	switch raw {
	case "\x1b[I":
		return Report{Kind: FocusIn}, true
	case "\x1b[O":
		return Report{Kind: FocusOut}, true
	}
	if rest, ok := strings.CutPrefix(raw, "\x1b[<"); ok && rest != "" {
		final := rest[len(rest)-1]
		if final != 'M' && final != 'm' {
			return Report{}, false
		}
		var n [3]int
		parts := strings.Split(rest[:len(rest)-1], ";")
		if len(parts) != 3 {
			return Report{}, false
		}
		for i, p := range parts {
			v, err := strconv.Atoi(p)
			if err != nil || v < 0 || v > maxCoord || p == "" || p[0] == '+' {
				return Report{}, false
			}
			n[i] = v
		}
		if n[1] < 1 || n[2] < 1 {
			return Report{}, false
		}
		r, _ := button(n[0])
		return report(r, final == 'm', n[1]-1, n[2]-1), true
	}
	if rest, ok := strings.CutPrefix(raw, "\x1b[M"); ok && len(rest) == 3 {
		if rest[0] < 32 || rest[1] < 33 || rest[2] < 33 {
			return Report{}, false
		}
		r, release := button(int(rest[0]) - 32)
		return report(r, release, int(rest[1])-33, int(rest[2])-33), true
	}
	return Report{}, false
}

// button is the report's button, modifiers and motion from xterm's bits,
// and whether the bits are X10's release, button 3 among the first three.
func button(b int) (Report, bool) {
	const (
		bitShift  = 4
		bitAlt    = 8
		bitCtrl   = 16
		bitMotion = 32
		bitWheel  = 64
		bitAdd    = 128
		mask      = 3
	)
	var r Report
	if b&bitShift != 0 {
		r.Mod |= termkey.ModShift
	}
	if b&bitAlt != 0 {
		r.Mod |= termkey.ModAlt
	}
	if b&bitCtrl != 0 {
		r.Mod |= termkey.ModCtrl
	}
	release := false
	switch {
	case b&bitAdd != 0:
		r.Button = ButtonBackward + MouseButton(b&mask) //nolint:gosec // b&mask is 0 to 3
	case b&bitWheel != 0:
		r.Button = WheelUp + MouseButton(b&mask) //nolint:gosec // b&mask is 0 to 3
		r.Kind = MouseWheel
	case b&mask == mask:
		r.Button, release = ButtonNone, true
	default:
		r.Button = ButtonLeft + MouseButton(b&mask) //nolint:gosec // b&mask is 0 to 2
	}
	if b&bitMotion != 0 && r.Kind != MouseWheel {
		r.Kind = MouseMotion
	}
	return r, release
}

// report is r at x, y, its kind by ultraviolet's order: a wheel, then
// motion, then a release, then a press.
func report(r Report, release bool, x, y int) Report {
	r.X, r.Y = x, y
	if r.Kind == MousePress && release {
		r.Kind = MouseRelease
	}
	return r
}
