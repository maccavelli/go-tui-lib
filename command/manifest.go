package command

// manifestFormat is the Manifest's format, raised when a field changes
// meaning or goes away.
const manifestFormat = 1

// Manifest is the whole catalogue as one JSON document, for docs, shell
// completion and other tools: every command, hidden ones too, from one
// version of the registry.
type Manifest struct {
	Format   int               `json:"format"`  // the document's format, 1
	Version  uint64            `json:"version"` // the registry's Version
	Commands []ManifestCommand `json:"commands"`
}

// ManifestCommand describes one command: its names, its arguments and
// output schemas, and how it may run. command.list and command.describe
// return it too (docs/decisions/0006-MADR-command-registry.md A7).
type ManifestCommand struct {
	ID          ID       `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitzero"`
	Category    string   `json:"category,omitzero"`
	Slash       string   `json:"slash,omitzero"`
	Aliases     []string `json:"aliases,omitzero"`
	ArgHint     string   `json:"argHint,omitzero"`
	Kind        string   `json:"kind"`
	Danger      string   `json:"danger"`
	Surfaces    string   `json:"surfaces"`
	Mode        string   `json:"mode"`
	When        string   `json:"when,omitzero"`
	Scope       string   `json:"scope"`
	Idempotent  bool     `json:"idempotent,omitzero"`
	OpenWorld   bool     `json:"openWorld,omitzero"`
	Exclusive   bool     `json:"exclusive,omitzero"`
	WhileBusy   bool     `json:"whileBusy,omitzero"`
	Hidden      bool     `json:"hidden,omitzero"`
	Source      string   `json:"source"`
	Args        Schema   `json:"args,omitzero"`
	Output      Schema   `json:"output,omitzero"`
}

func manifestCommandOf(c *Command) ManifestCommand {
	return ManifestCommand{
		ID: c.ID, Title: c.Title, Description: c.Description, Category: c.Category,
		Slash: c.Slash, Aliases: c.Aliases, ArgHint: c.ArgHint, Kind: c.Kind.String(),
		Danger: c.Danger.String(), Surfaces: c.surfaces().String(), Mode: c.Mode.String(),
		When: c.When, Scope: c.Scope.String(), Idempotent: c.Idempotent, OpenWorld: c.OpenWorld,
		Exclusive: c.Exclusive, WhileBusy: c.WhileBusy, Hidden: c.Hidden,
		Source: c.Source.String(), Args: c.Args, Output: c.Output,
	}
}

// Manifest is every command, in ID order, with the version they are from.
func (r *Registry) Manifest() Manifest {
	s := r.snap.Load()
	m := Manifest{Format: manifestFormat, Version: s.version, Commands: make([]ManifestCommand, len(s.entries))}
	for i := range s.entries {
		m.Commands[i] = manifestCommandOf(&s.entries[i].cmd)
	}
	return m
}
