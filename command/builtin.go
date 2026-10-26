package command

import (
	"context"
	"strings"
)

// The IDs of the registry's own commands
// (docs/decisions/0006-MADR-command-registry.md §9, A5), which NewRegistry
// adds unless WithoutBuiltins is given, and Remove takes out. None has a
// slash name, so no built-in word clashes with a program's.
const (
	BuiltinList     ID = "command.list"     // lists the commands the caller can run now
	BuiltinDescribe ID = "command.describe" // describes one command
	BuiltinQuit     ID = "app.quit"         // asks the program to quit
)

// describeArgs is command.describe's argument.
type describeArgs struct {
	ID string `json:"id" arg:"" help:"the command's ID" placeholder:"ID"`
}

// builtins is the registry's own commands, closed over r. Their
// definitions are fixed, so an error is a bug, and MustNew panics on it.
func (r *Registry) builtins() []Command {
	list := MustNew(BuiltinList, "List commands", func(_ context.Context, inv *Invocation, _ NoArgs) (Result, error) {
		var infos []ManifestCommand
		var text strings.Builder
		for c := range r.Available(inv.WhenContext, inv.Origin.surface()) {
			infos = append(infos, manifestCommandOf(&c))
			text.WriteString(string(c.ID) + "  " + c.Title + "\n")
		}
		return Result{Value: infos, Text: text.String()}, nil
	},
		WithDescription("Lists the commands the caller can run now: not hidden, offered where it asks from, and available in its context."),
		WithDanger(ReadOnly), WithIdempotent())
	describe := MustNew(BuiltinDescribe, "Describe a command", func(_ context.Context, _ *Invocation, a describeArgs) (Result, error) {
		c, ok := r.Lookup(ID(a.ID))
		if !ok {
			return Result{}, &ArgError{Path: "/id", Reason: "no command has this ID"}
		}
		info := manifestCommandOf(&c)
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
	quit := MustNew(BuiltinQuit, "Quit", func(context.Context, *Invocation, NoArgs) (Result, error) {
		return Result{Cmd: func() any { return QuitRequestMsg{} }}, nil
	},
		WithDescription("Asks the program to quit; the program decides."),
		WithDanger(UI), WithSurfaces(AllSurfaces&^SurfaceCLI))
	return []Command{list, describe, quit}
}
