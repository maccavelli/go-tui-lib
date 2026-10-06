package command

import (
	"context"
	"fmt"
	"strings"
)

// MCPPrompt is one prompt of an MCP server's prompts/list result, with
// MCP's field names.
type MCPPrompt struct {
	Name        string              `json:"name"`
	Title       string              `json:"title,omitzero"`
	Description string              `json:"description,omitzero"`
	Arguments   []MCPPromptArgument `json:"arguments,omitzero"`
	Meta        map[string]any      `json:"_meta,omitzero"`
}

// MCPPromptArgument is one of an MCP prompt's arguments.
type MCPPromptArgument struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitzero"`
	Description string `json:"description,omitzero"`
	Required    bool   `json:"required,omitzero"`
}

// PromptGetter fetches and expands one of a server's prompts, by its MCP
// name, and returns its text. The host owns the MCP client: it calls
// prompts/get and joins the messages' text
// (docs/decisions/0006-MADR-command-registry.md A6).
type PromptGetter func(ctx context.Context, name string, args map[string]string) (string, error)

// FromMCPPrompts turns a server's prompts into Prompt commands with IDs
// mcp.<server>.<name>, each name as an ID segment. A prompt's arguments
// are string properties in its order, positional, and required as MCP
// says. Running one calls get with the prompt's MCP name and returns its
// text, which Dispatch sends as a PromptMsg. They are Mutating, as every
// loaded command without a declared danger is. Load them with
// ReplaceSource(Source{MCP, server}).
func FromMCPPrompts(server string, prompts []MCPPrompt, get PromptGetter) []Command {
	out := make([]Command, 0, len(prompts))
	for _, p := range prompts {
		title := p.Title
		if title == "" {
			title = p.Name
		}
		args := make([]stringArg, len(p.Arguments))
		for i, a := range p.Arguments {
			args[i] = stringArg{name: a.Name, desc: a.Description, required: a.Required}
		}
		c := Command{
			ID:          ID("mcp." + segment(server) + "." + segment(p.Name)),
			Title:       title,
			Description: p.Description,
			Slash:       segment(p.Name),
			Kind:        Prompt,
			Danger:      Mutating,
			Source:      Source{Kind: MCP, Name: server},
			Meta:        p.Meta,
			ArgHint:     argHint(args),
		}
		if schema, err := stringArgsSchema(args, false); err == nil { // strings only; it cannot fail
			c.Args = schema
		}
		name := p.Name
		c.Handler = HandlerFunc(func(ctx context.Context, inv *Invocation) (Result, error) {
			if get == nil {
				return Result{}, fmt.Errorf("command: %s: no PromptGetter", inv.Command.ID)
			}
			values, err := decodeArgs[map[string]string](inv.Args)
			if err != nil {
				return Result{}, err
			}
			text, err := get(ctx, name, values)
			return Result{Text: text}, err
		})
		out = append(out, c)
	}
	return out
}

// stringArg is one string argument of a loaded prompt.
type stringArg struct {
	name, desc string
	required   bool
}

// stringArgsSchema is an object schema of args as positional string
// properties, in order, that allows stray slash words when rest (A5, A6).
func stringArgsSchema(args []stringArg, rest bool) (Schema, error) {
	n := &node{typ: tObject, closed: true, cli: cliInfo{Rest: rest}}
	for _, a := range args {
		n.props = append(n.props, prop{a.name, &node{typ: tString, desc: a.desc, cli: cliInfo{Arg: true}}})
		if a.required {
			n.required = append(n.required, a.name)
		}
	}
	return n.encode(true)
}

// argHint is args' names, for ACP's input hint.
func argHint(args []stringArg) string {
	names := make([]string, len(args))
	for i, a := range args {
		names[i] = a.name
	}
	return strings.Join(names, " ")
}
