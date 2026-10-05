package termcap

import (
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/charmbracelet/colorprofile"
)

// Caps is what is known about the terminal. A field is added, never
// repurposed: a new capability is a new field whose zero value is Unknown
// with origin NotQueried, so code that reads Caps keeps its meaning.
//
// Caps marshals to JSON with every zero field left out, so a zero Caps is
// {}. Enumerations are written by name, Background as #rrggbb and Profile by
// its name.
type Caps struct {
	Complete bool `json:"complete,omitzero"`  // the DA1 sentinel arrived
	TimedOut bool `json:"timed_out,omitzero"` // the timeout fired first

	Terminal   Fact[string] `json:"terminal,omitzero"`   // XTVERSION reply, else TERM_PROGRAM, else TERM
	Mux        Fact[Mux]    `json:"mux,omitzero"`        // TMUX, STY or ZELLIJ, or an XTVERSION of tmux
	Remote     Fact[bool]   `json:"remote,omitzero"`     // SSH_TTY or SSH_CONNECTION in tea.EnvMsg
	Attributes []int        `json:"attributes,omitzero"` // the DA1 reply

	KittyKeyboard Fact[Support] `json:"kitty_keyboard,omitzero"`
	KeyboardFlags int           `json:"keyboard_flags,omitzero"` // from tea.KeyboardEnhancementsMsg

	SyncOutput         Fact[Support] `json:"sync_output,omitzero"`          // mode 2026, observed from tea's query
	GraphemeWidth      Fact[Support] `json:"grapheme_width,omitzero"`       // mode 2027, observed from tea's query
	ColorSchemeReports Fact[Support] `json:"color_scheme_reports,omitzero"` // mode 2031
	InBandResize       Fact[Support] `json:"in_band_resize,omitzero"`       // mode 2048
	FocusEvents        Fact[Support] `json:"focus_events,omitzero"`         // mode 1004

	DesktopNotify Fact[Support] `json:"desktop_notify,omitzero"` // the OSC 99 p=? reply
	KittyGraphics Fact[Support] `json:"kitty_graphics,omitzero"` // the APC G query
	Sixel         Fact[Support] `json:"sixel,omitzero"`          // 4 in the DA1 reply

	Dark       Fact[bool]           `json:"dark,omitzero"` // the appearance chain (MADR A1)
	Background color.Color          `json:"-"`             // OSC 11, when it replied
	Profile    colorprofile.Profile `json:"-"`             // from tea.ColorProfileMsg

	// Added by MADR A1 and A4.

	Brand         Fact[Brand]    `json:"brand,omitzero"`          // EnvBrand refined, or an XTVERSION reply
	EnvBrand      Fact[Brand]    `json:"env_brand,omitzero"`      // the brand the environment names, unrefined
	Editor        Fact[Editor]   `json:"editor,omitzero"`         // an editor's embedded terminal
	Platform      Fact[Platform] `json:"platform,omitzero"`       // MSYS2 or WSL
	LegacyConsole Fact[bool]     `json:"legacy_console,omitzero"` // the classic Windows console host
	Tmux          TmuxFacts      `json:"tmux,omitzero"`           // from the program's run of TmuxQuery

	Foreground   color.Color     `json:"-"`                      // OSC 10, when it replied
	Palette      [16]color.Color `json:"-"`                      // OSC 4, indexes 0-15, each when it replied
	PaletteKnown bool            `json:"palette_known,omitzero"` // all 16 replied

	SecondaryAttributes []int `json:"secondary_attributes,omitzero"` // the DA2 reply
}

// caps is Caps without its methods, so the JSON methods can embed it.
type caps Caps

// capsJSON adds the fields whose types do not write themselves as text.
type capsJSON struct {
	caps
	Background string   `json:"background,omitzero"`
	Profile    string   `json:"profile,omitzero"`
	Foreground string   `json:"foreground,omitzero"`
	Palette    []string `json:"palette,omitzero"` // 16 entries, "" where unknown
}

// MarshalJSON writes c with every zero field left out.
func (c Caps) MarshalJSON() ([]byte, error) {
	j := capsJSON{caps: caps(c), Background: hex(c.Background), Foreground: hex(c.Foreground)}
	if c.Profile != colorprofile.Unknown {
		j.Profile = c.Profile.String()
	}
	if c.Palette != ([16]color.Color{}) {
		j.Palette = make([]string, len(c.Palette))
		for i, p := range c.Palette {
			j.Palette[i] = hex(p)
		}
	}
	return json.Marshal(j)
}

// hex is c as #rrggbb, or "" when c is nil.
func hex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// unhex reads what hex wrote; "" is nil.
func unhex(what, s string) (color.Color, error) {
	if s == "" {
		return nil, nil
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil || len(s) != 7 {
		return nil, fmt.Errorf("termcap: %s %q is not #rrggbb", what, s)
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}, nil
}

// UnmarshalJSON reads what MarshalJSON wrote.
func (c *Caps) UnmarshalJSON(b []byte) error {
	var j capsJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*c = Caps(j.caps)
	var err error
	if c.Background, err = unhex("background", j.Background); err != nil {
		return err
	}
	if c.Foreground, err = unhex("foreground", j.Foreground); err != nil {
		return err
	}
	if j.Palette != nil {
		if len(j.Palette) != len(c.Palette) {
			return fmt.Errorf("termcap: palette of %d colours, want %d", len(j.Palette), len(c.Palette))
		}
		for i, s := range j.Palette {
			if c.Palette[i], err = unhex("palette colour", s); err != nil {
				return err
			}
		}
	}
	if j.Profile != "" {
		p, ok := parseProfile(j.Profile)
		if !ok {
			return fmt.Errorf("termcap: unknown colour profile %q", j.Profile)
		}
		c.Profile = p
	}
	return nil
}

// parseProfile reads a profile's String.
func parseProfile(s string) (colorprofile.Profile, bool) {
	for p := colorprofile.Unknown; p <= colorprofile.TrueColor; p++ {
		if p.String() == s {
			return p, true
		}
	}
	return colorprofile.Unknown, false
}
