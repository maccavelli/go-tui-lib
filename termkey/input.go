package termkey

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// InputKind is what an Input is.
type InputKind uint8

const (
	// InputKey is a key, in Key. The zero value.
	InputKey InputKind = iota
	// InputPaste is a bracketed paste, its text in Text.
	InputPaste
	// InputUnknown is a sequence the decoder does not read as a key, in
	// Raw: a terminal's reply, such as DA1 or OSC 11, which
	// termcap.ParseReply reads, or anything else a terminal sends.
	InputUnknown
)

var inputKindNames = enum.Names[InputKind]{
	Pkg: pkgName, Type: "InputKind",
	Tokens: []string{"key", "paste", "unknown"},
}

// String is the kind's token.
func (k InputKind) String() string { return inputKindNames.String(k) }

// MarshalText is the kind's token. A kind with no token is an error.
func (k InputKind) MarshalText() ([]byte, error) { return inputKindNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *InputKind) UnmarshalText(b []byte) error { return inputKindNames.Unmarshal(b, k) }

// Input is one thing the Decoder read. Only the fields its Kind names are
// set. Fields are added, never changed, so build one with a keyed literal.
type Input struct {
	Kind      InputKind
	Key       KeyEvent // InputKey
	Text      string   // InputPaste: the text, invalid UTF-8 left out
	Raw       []byte   // InputKey and InputUnknown: the bytes it came from
	Truncated bool     // a paste over MaxPaste, or a sequence over MaxSequence, cut there
}
