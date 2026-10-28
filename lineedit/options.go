package lineedit

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// DefaultLimit bounds the text, in runes, as x/term and gobble bound
// theirs.
const DefaultLimit = 4096

// PasteMode is what a bracketed paste does. Its text form is a stable
// lowercase token.
type PasteMode uint8

const (
	// PasteInsert puts the paste into the text at the cursor, editable,
	// its line breaks and tabs kept. The zero PasteMode.
	PasteInsert PasteMode = iota
	// PasteSubmit puts the paste in and submits the text at once, as
	// gobble does (its 0027-MADR Q2=A).
	PasteSubmit
)

var pasteModeNames = enum.Names[PasteMode]{Pkg: pkgName, Type: "PasteMode", Tokens: []string{"insert", "submit"}}

// String is the mode's token: "insert" or "submit".
func (p PasteMode) String() string { return pasteModeNames.String(p) }

// MarshalText is the mode's token. A mode with no token is an error.
func (p PasteMode) MarshalText() ([]byte, error) { return pasteModeNames.Marshal(p) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (p *PasteMode) UnmarshalText(b []byte) error { return pasteModeNames.Unmarshal(b, p) }

// Option configures NewComposer. It is opaque: only this package's
// functions make one.
type Option interface{ apply(*Composer) }

// optionFunc is a function used as an Option.
type optionFunc func(*Composer)

func (f optionFunc) apply(c *Composer) { f(c) }

// WithPrompt sets the prompt before the text's first line: the glyphs'
// Prompt and a space by default. An empty prompt keeps it.
func WithPrompt(s string) Option { return optionFunc(func(c *Composer) { c.prompt = s }) }

// WithContinue sets the prompt before each line after a line break: the
// glyphs' Ellipsis and a space by default, as gobble's "… ". An empty one
// keeps it.
func WithContinue(s string) Option { return optionFunc(func(c *Composer) { c.cont = s }) }

// WithHeader gives a line drawn above each new prompt, read when the
// prompt is drawn, such as the model in use; an empty line draws none.
// None by default.
func WithHeader(header func() string) Option {
	return optionFunc(func(c *Composer) { c.header = header })
}

// WithHistory gives the lines up and down recall, and adds each
// submission to them. None by default.
func WithHistory(h History) Option { return optionFunc(func(c *Composer) { c.history = h }) }

// WithCompleter gives what tab completes the text with. None by default.
func WithCompleter(x Completer) Option { return optionFunc(func(c *Composer) { c.completer = x }) }

// WithKeymap resolves the Composer's keys in k, the program's keymap, in
// ComposerScope alone, so a user's file rebinds them; k holds
// DefaultRules among its defaults. Without it, or with nil, the Composer
// builds a keymap of DefaultRules alone.
func WithKeymap(k *keymap.Keymap) Option { return optionFunc(func(c *Composer) { c.keys = k }) }

// WithLimit bounds the text, in runes. Zero or less keeps DefaultLimit.
func WithLimit(n int) Option {
	return optionFunc(func(c *Composer) {
		if n > 0 {
			c.limit = n
		}
	})
}

// WithPaste sets what a bracketed paste does: PasteInsert by default.
func WithPaste(p PasteMode) Option { return optionFunc(func(c *Composer) { c.paste = p }) }

// WithExternalEditor turns on ExternalID, ctrl+x ctrl+e by default, which
// sends EditExternal for the host to edit the text in its editor. Off by
// default: the library starts no process (AGENTS.md, API conventions 9).
func WithExternalEditor() Option { return optionFunc(func(c *Composer) { c.external = true }) }

// WithClearScreen turns on ClearID, ctrl+l by default, which clears the
// screen, CSI 2 J and CSI H, and draws the Composer again at the top, as
// x/term's and gobble's ctrl+l do. Off by default
// (docs/decisions/0023-MADR-line-mode-components.md A5 item 2).
func WithClearScreen() Option { return optionFunc(func(c *Composer) { c.clear = true }) }

// WithMethod sets how the text is measured: ansi.WcWidth by default, as a
// terminal measures until it reports grapheme clustering (mode 2027), and
// ansi.GraphemeWidth after. It is the region's too, when the Composer
// builds its own.
func WithMethod(m ansi.Method) Option { return optionFunc(func(c *Composer) { c.method = m }) }

// WithGlyphs sets the glyphs the prompts default to: glyph.Unicode() by
// default, glyph.ASCII() where the terminal cannot show Unicode.
func WithGlyphs(g glyph.Set) Option { return optionFunc(func(c *Composer) { c.glyphs = g }) }

// WithRegion draws the Composer in r, the host's region, which it shares
// among its components and sets inline's options on; one engine draws a
// region at a time. Without it, or with nil, the Composer builds its own,
// by inline.NewRegion and WithMethod's method
// (docs/decisions/0023-MADR-line-mode-components.md A2 item 3).
func WithRegion(r *inline.Region) Option { return optionFunc(func(c *Composer) { c.region = r }) }

// WithRules sets how typed text, a paste, a completion, the prompts and
// the header are cleaned: by r, and then by safetext's Line rules, the
// floor; a paste keeps its line breaks and tabs, and is cleaned by r and
// then by a floor that keeps only those of the controls
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D2, 0023-MADR
// A3 item 11). safetext.Line() by default.
func WithRules(r safetext.Rules) Option { return optionFunc(func(c *Composer) { c.rules = r }) }
