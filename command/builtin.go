package command

import (
	"context"
	"strings"
)

// The registry's own commands (docs/decisions/0006-MADR-command-registry.md
// §9, A5). None has a slash name, so no built-in word clashes with a
// program's.
const (
	idList     ID = "command.list"
	idDescribe ID = "command.describe"
	idQuit     ID = "app.quit"
)

// commandInfo is a command as command.list and command.describe give it.
type commandInfo struct {
	ID          ID       `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitzero"`
	Category    string   `json:"category,omitzero"`
	Slash       string   `json:"slash,omitzero"`
	Aliases     []string `json:"aliases,omitzero"`
	Kind        string   `json:"kind"`
	Danger      string   `json:"danger"`
	Surfaces    string   `json:"surfaces"`
	Mode        string   `json:"mode"`
	When        string   `json:"when,omitzero"`
	Hidden      bool     `json:"hidden,omitzero"`
	Source      string   `json:"source"`
	Args        Schema   `json:"args,omitzero"`
	Output      Schema   `json:"output,omitzero"`
}

func infoOf(c *Command) commandInfo {
	return commandInfo{
		ID: c.ID, Title: c.Title, Description: c.Description, Category: c.Category,
		Slash: c.Slash, Aliases: c.Aliases, Kind: c.Kind.String(), Danger: c.Danger.String(),
		Surfaces: c.surfaces().String(), Mode: c.Mode.String(), When: c.When, Hidden: c.Hidden,
		Source: c.Source.String(), Args: c.Args, Output: c.Output,
	}
}

// describeArgs is command.describe's argument.
type describeArgs struct {
	ID string `json:"id" arg:"" help:"the command's ID" placeholder:"ID"`
}

// builtins is the registry's own commands, closed over r.
func (r *Registry) builtins() []Command {
	list, err1 := New(idList, "List commands", func(_ context.Context, inv *Invocation, _ NoArgs) (Result, error) {
		var infos []commandInfo
		var text strings.Builder
		for c := range r.Available(inv.Context, inv.Origin.surface()) {
			infos = append(infos, infoOf(&c))
			text.WriteString(string(c.ID) + "  " + c.Title + "\n")
		}
		return Result{Value: infos, Text: text.String()}, nil
	},
		WithDescription("Lists the commands the caller can run now: not hidden, offered where it asks from, and available in its context."),
		WithDanger(ReadOnly), WithIdempotent())
	describe, err2 := New(idDescribe, "Describe a command", func(_ context.Context, _ *Invocation, a describeArgs) (Result, error) {
		c, ok := r.Lookup(ID(a.ID))
		if !ok {
			return Result{}, &ArgError{Path: "/id", Reason: "no command has this ID"}
		}
		info := infoOf(&c)
		text := string(c.ID) + "\n" + c.Title + "\n"
		if c.Description != "" {
			text += c.Description + "\n"
		}
		text += "danger: " + info.Danger + "\n"
		if len(c.Args) > 0 {
			text += "arguments: " + string(c.Args) + "\n"
		}
		return Result{Value: info, Text: text}, nil
	},
		WithDescription("Describes one command: its ID, arguments schema and danger."),
		WithArgHint("command id"), WithDanger(ReadOnly), WithIdempotent())
	quit, err3 := New(idQuit, "Quit", func(context.Context, *Invocation, NoArgs) (Result, error) {
		return Result{Cmd: msgCmd(QuitRequestMsg{})}, nil
	},
		WithDescription("Asks the program to quit; the program decides."),
		WithDanger(UI), WithSurfaces(AllSurfaces&^SurfaceCLI))
	for _, err := range []error{err1, err2, err3} {
		if err != nil {
			panic(err) // fixed definitions; this is a bug
		}
	}
	return []Command{list, describe, quit}
}
