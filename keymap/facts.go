package keymap

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/internal/enum"
	"github.com/maccavelli/go-tui-lib/termcap"
)

// KeyFacts are what the terminal and its host deliver, beyond the Kitty
// features (MADR amendments A1 and A9). A default's Alternative names the
// facts its keys need.
type KeyFacts uint16

// The facts.
const (
	ShiftEnter     KeyFacts = 1 << iota // shift+enter arrives apart from enter
	AltEnter                            // alt+enter arrives
	CtrlEnter                           // ctrl+enter arrives apart from enter
	Releases                            // releases and repeats are reported, and none leak
	HostTakesCtrlQ                      // the host, such as VS Code, takes ctrl+q first
	HostTakesCtrlI                      // the host takes ctrl+i first
)

var factNames = enum.Bits[KeyFacts]{
	Pkg: pkgName, Type: "KeyFacts", None: "none",
	Tokens: []string{"shift-enter", "alt-enter", "ctrl-enter", "releases", "host-takes-ctrl-q", "host-takes-ctrl-i"},
}

// String is the set's names joined with "|", or "none". Bits with no name
// come last, in hex.
func (f KeyFacts) String() string { return factNames.String(f) }

// MarshalText is the set's names joined with "|", or "none". A bit with no
// name is an error.
func (f KeyFacts) MarshalText() ([]byte, error) { return factNames.Marshal(f) }

// UnmarshalText reads what MarshalText wrote, the names in any order.
func (f *KeyFacts) UnmarshalText(b []byte) error { return factNames.Unmarshal(b, f) }

// Alternative is keys a Default binds when its own cannot be delivered.
type Alternative struct {
	Keys  Sequence
	Needs Features // features the keys need, besides their own
	Facts KeyFacts // facts that must all hold, besides the keys' own
}

// hostBrands are the hosts that take ctrl+q and ctrl+i before the
// terminal sees them (0003-REPORT §9).
var hostBrands = []termcap.Brand{termcap.BrandVSCode, termcap.BrandCursor, termcap.BrandWindsurf, termcap.BrandZed}

// optionBrands are the macOS terminals whose Option key types characters
// until the user sets it to send Alt (amendment A9).
var optionBrands = []termcap.Brand{termcap.BrandAppleTerminal, termcap.BrandITerm2}

// FactsFrom is the facts termcap's capabilities show (amendment A9):
// Releases from Keyboard; ShiftEnter and CtrlEnter when the terminal
// accepted Kitty's disambiguation, and never CtrlEnter on Apple Terminal;
// AltEnter unless the brand is Apple Terminal or iTerm2; and the host
// facts when the brand is VS Code, Cursor, Windsurf or Zed.
func FactsFrom(c termcap.Caps) KeyFacts {
	k := c.Keyboard()
	brand := c.Brand.Value
	var f KeyFacts
	if k.Releases {
		f |= Releases
	}
	if k.Accepted&ansi.KittyDisambiguateEscapeCodes != 0 {
		f |= ShiftEnter
		if brand != termcap.BrandAppleTerminal {
			f |= CtrlEnter
		}
	}
	if !slices.Contains(optionBrands, brand) {
		f |= AltEnter
	}
	if slices.Contains(hostBrands, brand) {
		f |= HostTakesCtrlQ | HostTakesCtrlI
	}
	return f
}

// The strokes a host may take.
var (
	ctrlQ = Stroke{Code: 'q', Mod: tea.ModCtrl}
	ctrlI = Stroke{Code: 'i', Mod: tea.ModCtrl}
)

// keyFacts is the facts a sequence's strokes need, and those that bar it:
// Releases for a release or repeat stroke, and a host's taking ctrl+q or
// ctrl+i bars those strokes. Any other fact is an Alternative's to name.
func keyFacts(q Sequence) (need, bar KeyFacts) {
	for _, s := range q {
		if s.Event != Press {
			need |= Releases
		}
		press := s
		press.Event = Press
		switch press {
		case ctrlQ:
			bar |= HostTakesCtrlQ
		case ctrlI:
			bar |= HostTakesCtrlI
		}
	}
	return need, bar
}
