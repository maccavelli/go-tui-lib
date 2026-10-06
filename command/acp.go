package command

// metaACP is the Meta key that keeps an ACP command's own name, which a
// forward sends back to the agent.
const metaACP = "acp"

// ACPCommand is one of an ACP agent's available commands, as its
// available_commands_update sends it, with ACP's field names.
type ACPCommand struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Input       *ACPCommandInput `json:"input,omitzero"`
}

// ACPCommandInput is an ACP command's unstructured input: a hint shown
// where the user types its arguments.
type ACPCommandInput struct {
	Hint string `json:"hint"`
}

// FromACP turns an agent's available commands into Forward commands with
// IDs acp.<agent>.<name>, each name as an ID segment. Running one sends
// "/<name> <args>" to the agent as a PromptMsg, with the agent's own name,
// kept in Meta["acp"]. They are Mutating, as every loaded command without
// a declared danger is. Load them with ReplaceSource(Source{ACP, agent}).
func FromACP(agent string, cmds []ACPCommand) []Command {
	out := make([]Command, 0, len(cmds))
	for _, a := range cmds {
		seg := segment(a.Name)
		c := Command{
			ID:          ID("acp." + segment(agent) + "." + seg),
			Title:       a.Name,
			Description: a.Description,
			Slash:       seg,
			Kind:        Forward,
			Danger:      Mutating,
			Source:      Source{Kind: ACP, Name: agent},
			Meta:        map[string]any{metaACP: map[string]any{"name": a.Name}},
		}
		if a.Input != nil {
			c.ArgHint = a.Input.Hint
		}
		out = append(out, c)
	}
	return out
}
