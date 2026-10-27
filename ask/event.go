package ask

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "ask"

// AskerEventKind is what an AskerEvent asks of the host.
type AskerEventKind uint8

const (
	// AskNone is no event: NextEvent has nothing more. The zero value.
	AskNone AskerEventKind = iota
	// AskWrite asks the host to write Data to the terminal, in order.
	AskWrite
	// AskDone is the question answered, with Response; the Asker takes no
	// more input.
	AskDone
)

var askerEventKindNames = enum.Names[AskerEventKind]{
	Pkg: pkgName, Type: "AskerEventKind",
	Tokens: []string{"none", "write", "done"},
}

// String is the kind's token.
func (k AskerEventKind) String() string { return askerEventKindNames.String(k) }

// MarshalText is the kind's token. A kind with no token is an error.
func (k AskerEventKind) MarshalText() ([]byte, error) { return askerEventKindNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *AskerEventKind) UnmarshalText(b []byte) error { return askerEventKindNames.Unmarshal(b, k) }

// AskerEvent is one thing the Asker asks of its host. A host ignores a kind
// it does not know.
type AskerEvent struct {
	Kind     AskerEventKind
	Data     string   // AskWrite: the bytes
	Response Response // AskDone: the answer
}
