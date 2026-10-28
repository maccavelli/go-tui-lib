package lineedit

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "lineedit"

// ComposerEventKind is what a ComposerEvent asks of the host.
type ComposerEventKind uint8

const (
	// EditNone is no event: NextEvent has nothing more. The zero value.
	EditNone ComposerEventKind = iota
	// EditWrite asks the host to write Data to the terminal, in order.
	EditWrite
	// EditSubmit is the text submitted, in Text; the Composer has left it
	// in the scrollback and draws an empty prompt below it.
	EditSubmit
	// EditEOF is the end of input: ctrl+d on an empty line, or the host's
	// Cancel. The Composer takes no more input.
	EditEOF
	// EditInterrupt is ctrl+c, with the text as it is in Text; what it
	// means, such as a second press to exit, is the host's, and the text
	// stays until the host changes it with SetText.
	EditInterrupt
	// EditExternal asks the host to edit Text in an external editor and
	// give the result back with SetText. It is sent only WithExternalEditor
	// (docs/decisions/0023-MADR-line-mode-components.md D6, A5 item 3).
	EditExternal
)

var composerEventKindNames = enum.Names[ComposerEventKind]{
	Pkg: pkgName, Type: "ComposerEventKind",
	Tokens: []string{"none", "write", "submit", "eof", "interrupt", "external"},
}

// String is the kind's token.
func (k ComposerEventKind) String() string { return composerEventKindNames.String(k) }

// MarshalText is the kind's token. A kind with no token is an error.
func (k ComposerEventKind) MarshalText() ([]byte, error) { return composerEventKindNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *ComposerEventKind) UnmarshalText(b []byte) error {
	return composerEventKindNames.Unmarshal(b, k)
}

// ComposerEvent is one thing the Composer asks of its host. A host ignores
// a kind it does not know.
type ComposerEvent struct {
	Kind ComposerEventKind
	Data string // EditWrite: the bytes
	Text string // EditSubmit, EditInterrupt, EditExternal: the text
}
