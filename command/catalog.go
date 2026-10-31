package command

import "github.com/maccavelli/go-tui-lib/v2/msgcat"

// The registry's own words, by message ID: its built-in commands' output
// and the error text an MCP client's model reads
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9, A4).
const (
	// MsgNoSuchCommand is command.describe's reason when no command has the
	// ID asked for.
	MsgNoSuchCommand msgcat.MessageID = "command.no-such-command"
	// MsgDescribeDanger is command.describe's danger line: the danger's
	// token.
	MsgDescribeDanger msgcat.MessageID = "command.describe-danger"
	// MsgDescribeArguments is command.describe's arguments line: the
	// arguments' JSON Schema.
	MsgDescribeArguments msgcat.MessageID = "command.describe-arguments"
	// MsgMCPUnknown is CallMCP's text for a tool that does not exist: the
	// tool's name.
	MsgMCPUnknown msgcat.MessageID = "command.mcp-unknown"
	// MsgMCPUnavailable is CallMCP's text for a tool not available now: the
	// tool's name, then the error.
	MsgMCPUnavailable msgcat.MessageID = "command.mcp-unavailable"
	// MsgMCPRefused is CallMCP's text for a refused call: the tool's name,
	// then the error.
	MsgMCPRefused msgcat.MessageID = "command.mcp-refused"
	// MsgMCPArgs is CallMCP's text for arguments the schema refuses: the
	// argument error.
	MsgMCPArgs msgcat.MessageID = "command.mcp-args"
	// MsgMCPFailed is CallMCP's text for a tool that failed: the tool's
	// name, then the error.
	MsgMCPFailed msgcat.MessageID = "command.mcp-failed"
)

// Messages is the registry's words in English, its defaults, by ID: where
// a program starts its translation.
func Messages() msgcat.Table {
	return msgcat.Table{
		MsgNoSuchCommand:     "no command has this ID",
		MsgDescribeDanger:    "danger: %s",
		MsgDescribeArguments: "arguments: %s",
		MsgMCPUnknown:        "There is no tool named %s; list the tools to see which there are.",
		MsgMCPUnavailable:    "The tool %[1]s is not available now: %[2]s",
		MsgMCPRefused:        "The call to %[1]s was refused: %[2]s. Do not retry it unchanged.",
		MsgMCPArgs:           "The arguments do not fit the tool's input schema: %s. Fix them and call again.",
		MsgMCPFailed:         "The tool %[1]s failed: %[2]s",
	}
}

// WithCatalog gives the registry's own words in the program's: see
// Messages.
func WithCatalog(c msgcat.Catalog) RegistryOption {
	return registryOptionFunc(func(r *Registry) { r.catalog = c })
}

// say is the registry's text for id, from its catalogue or the default.
func (r *Registry) say(id msgcat.MessageID, args ...any) string {
	return msgcat.Format(r.catalog, id, Messages()[id], args...)
}
