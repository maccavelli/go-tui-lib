package command

import (
	"context"
	"encoding/json/jsontext"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/when"
)

// Handler does a command's work. The same handler serves every surface: it
// returns structured and human output, and a handler that needs the Bubble
// Tea runtime returns the effect as Result.Cmd. The registry recovers a
// handler's panic, on every path and on the program's loop too, and
// returns it as a *PanicError.
type Handler interface {
	Run(ctx context.Context, inv *Invocation) (Result, error)
}

// HandlerFunc is a function used as a Handler.
type HandlerFunc func(ctx context.Context, inv *Invocation) (Result, error)

// Run calls f.
func (f HandlerFunc) Run(ctx context.Context, inv *Invocation) (Result, error) {
	return f(ctx, inv)
}

// Invocation is what a handler is given: the command, its arguments, and
// who asked.
type Invocation struct {
	Command Command
	Args    jsontext.Value // the arguments, as JSON
	Raw     string         // the slash tail, verbatim
	Origin  Origin
	Caller  string // the agent's or MCP client's name, for example
	// WhenContext is the context the command was enabled in; never nil.
	WhenContext when.Context
	// Context is WhenContext, set to the same value through v0.9.x.
	//
	// Deprecated: use WhenContext (0014-MADR W4).
	Context when.Context
}

// Result is what a command produced.
type Result struct {
	Value any     // structured output: --json, MCP structuredContent
	Text  string  // human output: the shell's stdout, MCP text, a toast; a prompt's text
	Cmd   tea.Cmd // a follow-up effect for a TUI host; the shell ignores it
}

// Request asks the registry to run a command.
type Request struct {
	ID     ID
	Args   jsontext.Value
	Raw    string
	Origin Origin
	Caller string
	// WhenContext is the context When is evaluated in; nil is empty.
	WhenContext when.Context
	// Context is read when WhenContext is nil, through v0.9.x.
	//
	// Deprecated: use WhenContext (0014-MADR W4).
	Context when.Context
	// Gate, when set, is asked instead of the registry's gate, for this
	// request only, and its answers are not remembered: how a program's
	// own command line approves one command, from its --yes flag or a
	// prompt (docs/decisions/0006-MADR-command-registry.md A9).
	Gate Gate
}
