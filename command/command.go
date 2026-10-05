// Package command is a registry of commands: one definition of each action
// a program offers, which keys, a palette, slash commands, the shell and
// agents all run (docs/decisions/0006-MADR-command-registry.md).
//
// A Command names its arguments as a JSON Schema, its availability as a
// when expression, its danger, and the surfaces that offer it. A Registry
// holds the commands in immutable snapshots, so reads are lock-free and
// safe from any goroutine. Dispatch runs a command from a Bubble Tea
// program's Update and returns the effect as a tea.Cmd; Run runs one to
// completion, for the shell, agents and tests.
//
// Before a command runs, a policy decides by its danger and the request's
// origin: read-only and UI commands run for anyone, a mutating command
// from an agent and a destructive one from an agent or the shell ask the
// Gate, and with no gate they are refused. An Auditor, when set, records
// every request.
//
// Nothing here writes to the terminal or the process's streams. Output
// leaves as a Result, a tea.Msg, or a record on the caller's logger.
package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"
)

// maxIDLength is the longest ID, in bytes, so that an ID is also a valid
// MCP tool name.
const maxIDLength = 128

// ID is a command's dotted, lowercase name: "workspace.focus.next". Each
// segment matches [a-z0-9][a-z0-9-]*, and the whole ID is at most 128
// bytes.
type ID string

// Valid reports what makes id malformed, or nil.
func (id ID) Valid() error {
	if id == "" {
		return errors.New("command: empty ID")
	}
	if len(id) > maxIDLength {
		return fmt.Errorf("command: ID %q is %d bytes, more than %d", id, len(id), maxIDLength)
	}
	for seg := range id.Segments() {
		if !validSegment(seg) {
			return fmt.Errorf("command: ID %q: segment %q is not [a-z0-9][a-z0-9-]*", id, seg)
		}
	}
	return nil
}

func validSegment(seg string) bool {
	if seg == "" || seg[0] == '-' {
		return false
	}
	for i := range len(seg) {
		c := seg[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}

// Segments is id's dot-separated parts, in order.
func (id ID) Segments() iter.Seq[string] {
	return strings.SplitSeq(string(id), ".")
}

// Schema is a JSON Schema 2020-12 document.
type Schema = json.RawMessage

// Command is one action, as every surface sees it.
type Command struct {
	ID          ID
	Title       string   // "Focus next pane"; the palette and MCP title
	Description string   // one or two sentences; help, MCP and ACP
	Category    string   // grouping in the palette, help and the shell
	Slash       string   // "next"; empty means no slash command
	Aliases     []string // more slash names
	ArgHint     string   // ACP's input hint, such as "pane id"

	Kind       Kind    // Action, Prompt or Forward
	Args       Schema  // the arguments' JSON Schema
	Output     Schema  // optional schema of Result.Value (MCP outputSchema)
	When       string  // availability, a when expression; "" means always
	Scope      Scope   // Global, or a pane, overlay or mode name
	Danger     Danger  // ReadOnly, UI, Mutating or Destructive; never zero
	Idempotent bool    // MCP's idempotentHint
	OpenWorld  bool    // MCP's openWorldHint
	Surfaces   Surface // where it is offered; zero is every surface
	Mode       Mode    // Loop, on the event loop, or Async, off it
	Exclusive  bool    // a new run cancels the running one
	WhileBusy  bool    // may run while the agent streams
	Hidden     bool    // runnable, but not listed by Available

	Source  Source         // where the command came from
	Meta    map[string]any // open-ended; exported as MCP _meta
	Handler Handler
}

// surfaces is c's surfaces, with zero meaning every one.
func (c *Command) surfaces() Surface {
	if c.Surfaces == 0 {
		return AllSurfaces
	}
	return c.Surfaces
}

// Kind is what running a command does.
type Kind uint8

// The kinds. Action is the zero value.
const (
	Action  Kind = iota // runs a Go Handler
	Prompt              // expands into text for the agent, sent as PromptMsg
	Forward             // sends "/name args" to the agent as PromptMsg; needs no Handler
)

func (k Kind) String() string {
	switch k {
	case Action:
		return "action"
	case Prompt:
		return "prompt"
	case Forward:
		return "forward"
	}
	return fmt.Sprintf("Kind(%d)", uint8(k))
}

// Danger is how much a command can change, which decides who may run it
// without asking. Its zero value is "not declared", and Register refuses
// it (docs/decisions/0006-MADR-command-registry.md A4).
type Danger uint8

// The danger levels, least first.
const (
	ReadOnly    Danger = iota + 1 // changes nothing
	UI                            // changes only presentation; reversible
	Mutating                      // changes program or session state
	Destructive                   // cannot be undone, or writes outside the project
)

func (d Danger) String() string {
	switch d {
	case 0:
		return "undeclared"
	case ReadOnly:
		return "read-only"
	case UI:
		return "ui"
	case Mutating:
		return "mutating"
	case Destructive:
		return "destructive"
	}
	return fmt.Sprintf("Danger(%d)", uint8(d))
}

// Surface is a set of the places that offer a command. A Command's zero
// Surfaces is every surface.
type Surface uint8

// The surfaces.
const (
	SurfaceKey Surface = 1 << iota
	SurfacePalette
	SurfaceSlash
	SurfaceCLI
	SurfaceAgent

	AllSurfaces = SurfaceKey | SurfacePalette | SurfaceSlash | SurfaceCLI | SurfaceAgent
)

var surfaceNames = []string{"key", "palette", "slash", "cli", "agent"}

// String is the set's names joined with "|", or "none".
func (s Surface) String() string {
	if s == 0 {
		return "none"
	}
	var names []string
	for i, n := range surfaceNames {
		if s&(1<<i) != 0 {
			names = append(names, n)
		}
	}
	if rest := s &^ AllSurfaces; rest != 0 {
		names = append(names, fmt.Sprintf("%#x", uint8(rest)))
	}
	return strings.Join(names, "|")
}

// Mode is where a command runs.
type Mode uint8

// The modes. Loop is the zero value.
const (
	Loop  Mode = iota // during Dispatch, on the event loop
	Async             // inside the tea.Cmd Dispatch returns, under a context Cancel reaches
)

func (m Mode) String() string {
	switch m {
	case Loop:
		return "loop"
	case Async:
		return "async"
	}
	return fmt.Sprintf("Mode(%d)", uint8(m))
}

// Scope is where a command applies: Global, or a pane, overlay or mode
// name.
type Scope string

// Global is the scope of a command that applies everywhere.
const Global Scope = ""

func (s Scope) String() string {
	if s == Global {
		return "global"
	}
	return string(s)
}

// SourceKind is where a command came from.
type SourceKind uint8

// The source kinds. Builtin is the zero value.
const (
	Builtin SourceKind = iota // the program or the library
	User                      // the user's command files
	Project                   // the project's command files
	MCP                       // an MCP server's prompts
	ACP                       // an ACP agent's available commands
	Plugin                    // a plugin
)

var sourceKindNames = []string{"builtin", "user", "project", "mcp", "acp", "plugin"}

func (k SourceKind) String() string {
	if int(k) < len(sourceKindNames) {
		return sourceKindNames[k]
	}
	return fmt.Sprintf("SourceKind(%d)", uint8(k))
}

// Source is a command's origin: its kind, and the server, agent or
// plugin's name.
type Source struct {
	Kind SourceKind
	Name string
}

// String is the kind, then ":" and the name when there is one.
func (s Source) String() string {
	if s.Name == "" {
		return s.Kind.String()
	}
	return s.Kind.String() + ":" + s.Name
}

// Origin is what asked for a command to run. Its zero value is "not
// set", and Dispatch and Run refuse it
// (docs/decisions/0006-MADR-command-registry.md A4).
type Origin uint8

// The origins.
const (
	OriginKey     Origin = iota + 1 // a key binding
	OriginMouse                     // a click; needs SurfaceKey
	OriginPalette                   // the command palette
	OriginSlash                     // a slash command
	OriginCLI                       // the shell
	OriginAgent                     // an agent's tool call
	OriginProgram                   // the program itself; When applies, Surfaces do not
)

var originNames = []string{"unset", "key", "mouse", "palette", "slash", "cli", "agent", "program"}

func (o Origin) String() string {
	if int(o) < len(originNames) {
		return originNames[o]
	}
	return fmt.Sprintf("Origin(%d)", uint8(o))
}

// surface is the surface an origin must be offered on, or 0 when Surfaces
// do not apply.
func (o Origin) surface() Surface {
	switch o {
	case OriginKey, OriginMouse:
		return SurfaceKey
	case OriginPalette:
		return SurfacePalette
	case OriginSlash:
		return SurfaceSlash
	case OriginCLI:
		return SurfaceCLI
	case OriginAgent:
		return SurfaceAgent
	}
	return 0
}
