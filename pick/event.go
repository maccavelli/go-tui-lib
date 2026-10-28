package pick

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "pick"

// ChooserEventKind is what a ChooserEvent asks of the host.
type ChooserEventKind uint8

const (
	// PickNone is no event: NextEvent has nothing more. The zero value.
	PickNone ChooserEventKind = iota
	// PickWrite asks the host to write Data to the terminal, in order.
	PickWrite
	// PickDone is the pick made, with Entry, or canceled; the Chooser takes
	// no more input.
	PickDone
)

var chooserEventKindNames = enum.Names[ChooserEventKind]{
	Pkg: pkgName, Type: "ChooserEventKind",
	Tokens: []string{"none", "write", "done"},
}

// String is the kind's token.
func (k ChooserEventKind) String() string { return chooserEventKindNames.String(k) }

// MarshalText is the kind's token. A kind with no token is an error.
func (k ChooserEventKind) MarshalText() ([]byte, error) { return chooserEventKindNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *ChooserEventKind) UnmarshalText(b []byte) error {
	return chooserEventKindNames.Unmarshal(b, k)
}

// ChooserEvent is one thing the Chooser asks of its host. A host ignores a
// kind it does not know.
type ChooserEvent struct {
	Kind     ChooserEventKind
	Data     string // PickWrite: the bytes
	Entry    Entry  // PickDone: the entry chosen, as the caller gave it
	Canceled bool   // PickDone: nothing was chosen
}
