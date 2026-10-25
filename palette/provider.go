package palette

import (
	"context"
	"encoding/json/jsontext"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// Provider is a source of hits (0008-MADR §4).
type Provider interface {
	// Name is the provider's name, which a hit's Group defaults to and a
	// failure's row shows.
	Name() string
	// Search yields hits for a non-empty query until ctx is cancelled or
	// yield returns false. It runs off the event loop, and is cancelled
	// when the next keystroke starts a new search or the picker closes.
	Search(ctx context.Context, s Search, yield func(Hit) bool) error
}

// Discoverer is a Provider that also has hits for an empty query, such as
// recent items. It is found by type assertion.
type Discoverer interface {
	Discover(ctx context.Context, yield func(Hit) bool) error
}

// Starter is a Provider with work to do before its first search each time
// the picker opens, such as walking a tree. It is found by type assertion.
// Start runs off the event loop, once per opening, with a context that
// closing the picker cancels; searches wait for it. An error from Start is
// the provider's failure until the picker closes.
type Starter interface {
	Start(ctx context.Context) error
}

// Stopper is a Provider with something to release when the picker closes.
// It is found by type assertion, and called only for a provider that was
// used while the picker was open, once its searches have returned.
type Stopper interface {
	Stop()
}

// Prefixed is a Provider that owns a prefix: a query that starts with it
// searches this provider alone, with the prefix taken off, as ">" does for
// commands and "@" for files. A query of "?" lists the prefixes.
type Prefixed interface {
	Prefix() string
}

// Debouncer is a Provider whose search waits after a keystroke, so that a
// burst of keys searches once; the next keystroke cancels the wait.
type Debouncer interface {
	Debounce() time.Duration
}

// Search is what a provider is asked: the query's text, its matcher,
// compiled once per keystroke and shared by every provider, and the when
// values at the time the picker opened (0008-MADR §4, A2, A5.2).
type Search struct {
	Text        string
	Matcher     *fuzzy.Matcher
	WhenContext when.Context
}

// Hit is one result. Fields are added, never changed, so build one with a
// keyed literal.
type Hit struct {
	ID       string        // stable; keys recents and de-duplication
	Text     string        // the line shown
	Ranges   []fuzzy.Range // the matched ranges of Text, filled for drawn rows only
	Score    fuzzy.Score
	Group    string         // a category; the provider's name when empty
	Help     string         // the selected hit's description line
	Hint     string         // a key hint; the keymap's when empty and Command is set
	Command  command.ID     // the command the hit runs, if it is one
	Danger   command.Danger // a badge for a command's danger
	Disabled string         // non-empty: shown with this reason, and not run
	Params   []command.Param
	// Run is what accepting the hit does, on the event loop, with the
	// arguments collected for Params, or nil. A Run that blocks blocks the
	// program.
	Run func(args jsontext.Value) tea.Cmd
}

// Availability says where a provider searches: When is an expression of
// the when package, evaluated against the when values at the time the
// picker opens. An empty When is everywhere.
type Availability struct {
	When string
}
