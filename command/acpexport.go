package command

import "github.com/maccavelli/go-tui-lib/when"

// ACPCommands is the registry's commands as an ACP agent's available
// commands, for an available_commands_update: the commands an agent may
// run in c, as MCPTools has them, that have a slash name, under it, in ID
// order. ArgHint is input.hint. A command without a slash name is left
// out, because a user cannot type it
// (docs/decisions/0006-MADR-command-registry.md A7).
func (r *Registry) ACPCommands(c when.Context) []ACPCommand {
	var out []ACPCommand
	for cmd := range r.Available(c, SurfaceAgent) {
		if cmd.Slash == "" {
			continue
		}
		a := ACPCommand{Name: cmd.Slash, Description: cmd.Description}
		if cmd.ArgHint != "" {
			a.Input = &ACPCommandInput{Hint: cmd.ArgHint}
		}
		out = append(out, a)
	}
	return out
}
