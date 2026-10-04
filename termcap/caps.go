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

	Dark       Fact[bool]           `json:"dark,omitzero"` // DSR 997, else OSC 11 luminance
	Background color.Color          `json:"-"`             // OSC 11, when it replied
	Profile    colorprofile.Profile `json:"-"`             // from tea.ColorProfileMsg
}

// caps is Caps without its methods, so the JSON methods can embed it.
type caps Caps

// capsJSON adds the two fields whose types do not write themselves as text.
type capsJSON struct {
	caps
	Background string `json:"background,omitzero"`
	Profile    string `json:"profile,omitzero"`
}

// MarshalJSON writes c with every zero field left out.
func (c Caps) MarshalJSON() ([]byte, error) {
	j := capsJSON{caps: caps(c)}
	if c.Background != nil {
		r, g, b, _ := c.Background.RGBA()
		j.Background = fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	}
	if c.Profile != colorprofile.Unknown {
		j.Profile = c.Profile.String()
	}
	return json.Marshal(j)
}

// UnmarshalJSON reads what MarshalJSON wrote.
func (c *Caps) UnmarshalJSON(b []byte) error {
	var j capsJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*c = Caps(j.caps)
	if j.Background != "" {
		var r, g, bl uint8
		if _, err := fmt.Sscanf(j.Background, "#%02x%02x%02x", &r, &g, &bl); err != nil || len(j.Background) != 7 {
			return fmt.Errorf("termcap: background %q is not #rrggbb", j.Background)
		}
		c.Background = color.RGBA{R: r, G: g, B: bl, A: 0xff}
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
