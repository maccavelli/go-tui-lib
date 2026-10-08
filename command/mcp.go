package command

import (
	"context"
	"errors"

	"github.com/maccavelli/go-tui-lib/when"
)

// The shapes here follow MCP 2026-07-28
// (docs/decisions/0006-MADR-command-registry.md A2, Q6): a finished
// call's resultType, and a text block's type.
const (
	mcpResultComplete = "complete"
	mcpText           = "text"
)

// emptyObjectSchema is the input schema of a command that has none.
var emptyObjectSchema = Schema(`{"type":"object"}`)

// MCPTool is a command as an MCP 2026-07-28 Tool, for tools/list. Name is
// the command's ID.
type MCPTool struct {
	Name         string             `json:"name"`
	Title        string             `json:"title,omitzero"`
	Description  string             `json:"description,omitzero"`
	InputSchema  Schema             `json:"inputSchema"`
	OutputSchema Schema             `json:"outputSchema,omitzero"`
	Annotations  MCPToolAnnotations `json:"annotations"`
	Meta         map[string]any     `json:"_meta,omitzero"`
}

// MCPToolAnnotations are a tool's hints, from its Danger, Idempotent and
// OpenWorld (MADR §2). Every hint that applies is written, because MCP's
// defaults for destructiveHint and openWorldHint are true. destructiveHint
// and idempotentHint mean something only when readOnlyHint is false, and
// are left out when it is true.
type MCPToolAnnotations struct {
	ReadOnlyHint    bool  `json:"readOnlyHint"`
	DestructiveHint *bool `json:"destructiveHint,omitzero"`
	IdempotentHint  *bool `json:"idempotentHint,omitzero"`
	OpenWorldHint   bool  `json:"openWorldHint"`
}

// MCPCallResult is a tools/call result. ResultType is always "complete".
// An error is a result with IsError set and a message the model can act
// on, as MCP asks, never a protocol error.
type MCPCallResult struct {
	ResultType        string       `json:"resultType"`
	Content           []MCPContent `json:"content"`
	StructuredContent any          `json:"structuredContent,omitzero"`
	IsError           bool         `json:"isError,omitzero"`
}

// MCPContent is one text content block of a result.
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCPTools is the tools an agent may call in c: the commands that are not
// hidden, are offered on SurfaceAgent and whose When holds, in ID order.
func (r *Registry) MCPTools(c when.Context) []MCPTool {
	var tools []MCPTool
	for cmd := range r.Available(c, SurfaceAgent) {
		tools = append(tools, mcpTool(&cmd))
	}
	return tools
}

func mcpTool(c *Command) MCPTool {
	t := MCPTool{
		Name: string(c.ID), Title: c.Title, Description: c.Description,
		InputSchema: c.Args, OutputSchema: c.Output, Meta: c.Meta,
		Annotations: MCPToolAnnotations{ReadOnlyHint: c.Danger == ReadOnly, OpenWorldHint: c.OpenWorld},
	}
	if len(t.InputSchema) == 0 {
		t.InputSchema = emptyObjectSchema
	}
	if c.Danger != ReadOnly {
		destructive, idempotent := c.Danger == Destructive, c.Idempotent
		t.Annotations.DestructiveHint, t.Annotations.IdempotentHint = &destructive, &idempotent
	}
	return t
}

// CallMCP runs the tool name, a command ID, with arguments, for the agent
// or MCP client caller: as OriginAgent, so the policy and the gate apply,
// and audited. The result's text is Result.Text, or Result.Value as JSON
// when there is no text; Result.Value is the structured content. An
// unknown name, an unavailable command, bad arguments, a refusal and a
// failure are each a result with IsError set.
func (r *Registry) CallMCP(ctx context.Context, name string, arguments []byte, caller string) MCPCallResult {
	res, err := r.Run(ctx, Request{ID: ID(name), Args: arguments, Origin: OriginAgent, Caller: caller})
	if err != nil {
		return mcpError(name, err)
	}
	text, err := resultText(res)
	if err != nil {
		return mcpError(name, err)
	}
	out := MCPCallResult{ResultType: mcpResultComplete, Content: []MCPContent{}, StructuredContent: res.Value}
	if text != "" {
		out.Content = append(out.Content, MCPContent{Type: mcpText, Text: text})
	}
	return out
}

// mcpError is err as a result the model can act on.
func mcpError(name string, err error) MCPCallResult {
	var msg string
	ae, isArg := errors.AsType[*ArgError](err)
	switch {
	case errors.Is(err, ErrUnknown):
		msg = "There is no tool named " + name + "; list the tools to see which there are."
	case errors.Is(err, ErrUnavailable):
		msg = "The tool " + name + " is not available now: " + err.Error()
	case errors.Is(err, ErrRefused):
		msg = "The call to " + name + " was refused: " + err.Error() + ". Do not retry it unchanged."
	case isArg:
		msg = "The arguments do not fit the tool's input schema: " + ae.Error() + ". Fix them and call again."
	default:
		msg = "The tool " + name + " failed: " + err.Error()
	}
	return MCPCallResult{ResultType: mcpResultComplete, Content: []MCPContent{{Type: mcpText, Text: msg}}, IsError: true}
}
