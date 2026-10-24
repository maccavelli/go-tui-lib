package termcap

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// Query is one probe. Programs and later records add their own with
// WithQuery; each is sent after the built-in queries and before the
// sentinel. Parse is offered every reply that arrives while the probe is
// alive, and reports whether r answered it.
type Query struct {
	Name  string
	Seq   func(env Env, mux Mux) string // the bytes to send; "" skips it
	Gated bool                          // only behind the heuristic
	Parse func(r Reply, c *Caps) bool   // true when r answered it
}

// probe is a query in the batch.
type probe struct {
	Query
	silent   func(c *Caps) // what no reply before the sentinel means; nil for nothing
	answered bool
	mode     int // the mode a DECRQM asks about, or 0

	// prefix starts the raw reply this query gets, and fact is the fact a
	// reply over maxReply marks; both only for the built-in queries whose
	// replies may run long.
	prefix string
	fact   func(*Caps) *Fact[Support]
}

// batch is every query in the batch's order, before the sentinel: the
// Kitty keyboard request, the DECRQM set and DSR 996, the OSC 11 background
// query unless declined, DA2, the gated queries, each added query, then
// DECRQM 2026 and 2027 where Bubble Tea's own rule would send them. A mode
// WithoutModeQuery names is left out. The background query is written here
// rather than sent on its own, which a host could write after the sentinel
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md A3).
func (p *Probe) batch(env Env) []*probe {
	unsupported := func(f func(*Caps) *Fact[Support]) func(*Caps) {
		return func(c *Caps) { f(c).Set(Unsupported, Queried) }
	}
	qs := []*probe{
		{Name: "kitty-keyboard", Seq: fixed(ansi.RequestKittyKeyboard), Parse: parseKittyKeyboard,
			silent: unsupported(func(c *Caps) *Fact[Support] { return &c.KittyKeyboard })},
		modeProbe("color-scheme-reports", ansi.ModeLightDark, ansi.RequestModeLightDark, true),
		modeProbe("in-band-resize", ansi.ModeInBandResize, ansi.RequestModeInBandResize, true),
		modeProbe("focus-events", ansi.ModeFocusEvent, ansi.RequestModeFocusEvent, true),
		{Name: tokenColorScheme, Seq: fixed(ansi.RequestLightDarkReport), Parse: parseColorScheme},
		{Name: "background", Seq: p.background, Parse: parseBackground},
		{Name: "secondary-attributes", Seq: fixed(ansi.RequestSecondaryDeviceAttributes), Parse: parseDA2},
		{Name: "terminal-version", Seq: fixed(ansi.RequestNameVersion), Gated: true, Parse: parseVersion},
		{Name: "desktop-notify", Seq: wrapped(osc99Query), Gated: true, Parse: parseOSC99,
			silent: unsupported(func(c *Caps) *Fact[Support] { return &c.DesktopNotify }),
			prefix: "\x1b]99;", fact: func(c *Caps) *Fact[Support] { return &c.DesktopNotify }},
		{Name: "kitty-graphics", Seq: wrapped(kittyGraphicsQuery), Gated: true, Parse: parseKittyGraphics,
			silent: unsupported(func(c *Caps) *Fact[Support] { return &c.KittyGraphics })},
		{Name: "foreground", Seq: fixed(ansi.RequestForegroundColor), Gated: true, Parse: parseForeground},
		{Name: "palette", Seq: fixed(paletteQuery), Gated: true, Parse: parsePalette, prefix: "\x1b]4;"},
	}
	for _, q := range p.added {
		qs = append(qs, &probe{Query: q})
	}
	// Bubble Tea sends these two itself, under shouldQuerySynchronizedOutput,
	// and termcaptea leaves them out; a host that is not Bubble Tea gets them
	// from the probe, under the same rule. Tea reads no silence into them,
	// and neither does the probe.
	if querySynchronizedOutput(env) {
		qs = append(qs,
			modeProbe("sync-output", ansi.ModeSynchronizedOutput, ansi.RequestModeSynchronizedOutput, false),
			modeProbe("grapheme-width", ansi.ModeUnicodeCore, ansi.RequestModeUnicodeCore, false))
	}
	return slices.DeleteFunc(qs, func(q *probe) bool { return q.mode != 0 && slices.Contains(p.hostModes, q.mode) })
}

// querySynchronizedOutput is Bubble Tea v2.0.10's
// shouldQuerySynchronizedOutput (tea.go:980-994), copied so that a host
// that is not Bubble Tea asks for modes 2026 and 2027 where tea would: not
// over SSH unless the terminal is known to answer, and not in Apple
// Terminal.
func querySynchronizedOutput(env Env) bool {
	termType := env.Getenv("TERM")
	termProg, okTermProg := env.LookupEnv("TERM_PROGRAM")
	_, okSSHTTY := env.LookupEnv("SSH_TTY")
	_, okWTSession := env.LookupEnv("WT_SESSION")

	return (!okTermProg && !okSSHTTY) ||
		okWTSession ||
		(okTermProg && !strings.Contains(termProg, "Apple") && !okSSHTTY) ||
		strings.Contains(termType, "ghostty") ||
		strings.Contains(termType, "wezterm") ||
		strings.Contains(termType, "alacritty") ||
		strings.Contains(termType, "kitty") ||
		strings.Contains(termType, "rio")
}

// background is the OSC 11 query, unless WithoutBackgroundRequest.
func (p *Probe) background(Env, Mux) string {
	if p.noBackground {
		return ""
	}
	return ansi.RequestBackgroundColor
}

// parseBackground answers OSC 11. HandleReply records the colour itself.
func parseBackground(r Reply, _ *Caps) bool { return r.Kind == ReplyColor && r.OSC == 11 }

// fixed is a query's bytes that do not depend on the environment.
func fixed(seq string) func(Env, Mux) string { return func(Env, Mux) string { return seq } }

// wrapped is a query only the outer terminal can answer: inside tmux it
// goes out through tmux's passthrough, which needs allow-passthrough.
func wrapped(seq string) func(Env, Mux) string {
	return func(_ Env, m Mux) string {
		if m == Tmux {
			return ansi.TmuxPassthrough(seq)
		}
		return seq
	}
}

// modeFact is the fact DECRPM sets for mode, or nil.
func modeFact(c *Caps, mode int) *Fact[Support] {
	switch ansi.DECMode(mode) {
	case ansi.ModeSynchronizedOutput:
		return &c.SyncOutput
	case ansi.ModeUnicodeCore:
		return &c.GraphemeWidth
	case ansi.ModeLightDark:
		return &c.ColorSchemeReports
	case ansi.ModeInBandResize:
		return &c.InBandResize
	case ansi.ModeFocusEvent:
		return &c.FocusEvents
	}
	return nil
}

// modeProbe asks with DECRQM whether the terminal knows a mode. When silent,
// no reply before the sentinel means it does not.
func modeProbe(name string, mode ansi.DECMode, seq string, silent bool) *probe {
	q := &probe{
		Name: name, Seq: fixed(seq), Parse: func(r Reply, c *Caps) bool {
			if r.Kind != ReplyMode || r.Mode != int(mode) {
				return false
			}
			modeFact(c, r.Mode).Set(supportIf(r.Setting.supported()), Queried)
			return true
		},
		mode: int(mode),
	}
	if silent {
		q.silent = func(c *Caps) { modeFact(c, int(mode)).Set(Unsupported, Queried) }
	}
	return q
}

func supportIf(ok bool) Support {
	if ok {
		return Supported
	}
	return Unsupported
}

func parseKittyKeyboard(r Reply, c *Caps) bool {
	if r.Kind != ReplyKittyKeyboard {
		return false
	}
	c.KittyKeyboard.Set(Supported, Queried)
	c.KeyboardFlags = r.Flags
	return true
}

// parseColorScheme answers DSR 996. HandleReply records the scheme itself.
func parseColorScheme(r Reply, _ *Caps) bool { return r.Kind == ReplyColorScheme }

func parseVersion(r Reply, c *Caps) bool {
	if r.Kind != ReplyVersion {
		return false
	}
	c.Terminal.Set(sanitize.Line(r.Text), Queried)
	if strings.HasPrefix(strings.ToLower(r.Text), "tmux") {
		c.Mux.Set(Tmux, Queried)
	}
	if b := brandFromVersion(r.Text); b != BrandUnknown {
		c.Brand.Set(b, Queried)
	}
	return true
}

// parseDA2 reads the secondary device attributes, which name Alacritty's
// version and, with DA1, Apple Terminal over SSH.
func parseDA2(r Reply, c *Caps) bool {
	if r.Kind != ReplySecondaryAttributes {
		return false
	}
	c.SecondaryAttributes = slices.Clone(r.Params)
	appleFingerprint(c)
	return true
}

// appleFingerprint names Apple Terminal from DA1 1;2 with DA2 1;95;0,
// which over SSH is the only sign of it.
func appleFingerprint(c *Caps) {
	if slices.Equal(c.Attributes, []int{1, 2}) && slices.Equal(c.SecondaryAttributes, []int{1, 95, 0}) {
		c.Brand.Set(BrandAppleTerminal, Queried)
	}
}

func parseForeground(r Reply, c *Caps) bool {
	if r.Kind != ReplyColor || r.OSC != 10 {
		return false
	}
	c.Foreground = r.Color
	return true
}

// paletteQuery asks for the 16 ANSI palette colours (OSC 4), so a later
// theme record can build a theme from them.
var paletteQuery = func() string {
	var b strings.Builder
	for i := range 16 {
		fmt.Fprintf(&b, "\x1b]4;%d;?\x07", i)
	}
	return b.String()
}()

// parsePalette reads one OSC 4 reply, for an index from 0 to 15.
func parsePalette(r Reply, c *Caps) bool {
	if r.Kind != ReplyColor || r.OSC != 4 || r.Index < 0 || r.Index >= len(c.Palette) {
		return false
	}
	c.Palette[r.Index] = r.Color
	c.PaletteKnown = !slices.Contains(c.Palette[:], nil)
	return true
}

// osc99ID names the probe's OSC 99 query, so its reply is told apart.
const osc99ID = "termcap"

// osc99Query asks which desktop notification features the terminal has
// (the Kitty desktop notifications protocol: metadata p=?).
var osc99Query = ansi.DesktopNotification("", "i="+osc99ID, "p=?")

// parseOSC99 reads the reply to osc99Query: OSC 99 ; i=termcap:p=? ; <keys>
// ST. Any reply means the protocol is supported; the keys are not read
// yet. The metadata is read from Raw, which says whose query it answers.
func parseOSC99(r Reply, c *Caps) bool {
	if r.Kind != ReplyNotify {
		return false
	}
	data, ok := oscData(r.Raw)
	body, cut := strings.CutPrefix(data, "99;")
	if !ok || !cut {
		return false
	}
	meta, _, _ := strings.Cut(body, ";")
	keys := strings.Split(meta, ":")
	if !slices.Contains(keys, "p=?") || !slices.Contains(keys, "i="+osc99ID) {
		return false
	}
	c.DesktopNotify.Set(Supported, Queried)
	return true
}

// kittyGraphicsQuery is the Kitty graphics protocol's support query: a
// one-pixel image with action q, which the terminal checks and does not
// store.
var kittyGraphicsQuery = ansi.KittyGraphics([]byte("AAAA"), "i=31", "s=1", "v=1", "a=q", "t=d", "f=24")

// parseKittyGraphics reads the reply to kittyGraphicsQuery: OK when the
// terminal can show images this way, an error otherwise.
func parseKittyGraphics(r Reply, c *Caps) bool {
	if r.Kind != ReplyKittyGraphics {
		return false
	}
	c.KittyGraphics.Set(supportIf(r.Text == "OK"), Queried)
	return true
}

// isDark is ultraviolet's isDarkColor (decoder.go:1781), which Bubble Tea's
// BackgroundColorMsg.IsDark calls: the HSL lightness of the colour's 8-bit
// channels is under one half, which is max + min under 255. A nil colour
// is dark, as there.
func isDark(c color.Color) bool {
	if c == nil {
		return true
	}
	r, g, b, _ := c.RGBA()
	hi := max(r>>8, g>>8, b>>8)
	lo := min(r>>8, g>>8, b>>8)
	return hi+lo < 255
}

// gatedAllowed reports whether the gated queries may go out. Some terminals
// print a query they do not understand, so they go only where the
// environment gives reason to expect an answer: not to Apple Terminal, and
// over SSH only to a terminal the environment names.
func gatedAllowed(env Env) bool {
	if env.Getenv("TERM_PROGRAM") == "Apple_Terminal" {
		return false
	}
	_, tty := env.LookupEnv("SSH_TTY")
	_, conn := env.LookupEnv("SSH_CONNECTION")
	if !tty && !conn {
		return true
	}
	if env.Getenv("LC_TERMINAL") == "iTerm2" {
		return true
	}
	term := env.Getenv("TERM")
	for _, b := range []Brand{BrandKitty, BrandGhostty, BrandWezTerm, BrandAlacritty, BrandFoot, BrandRio, BrandContour} {
		if strings.Contains(term, b.String()) {
			return true
		}
	}
	return false
}
