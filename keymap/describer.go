package keymap

import "github.com/maccavelli/go-tui-lib/v2/command"

// Describer gives a command's title for help (MADR §7, amendment A4).
type Describer interface {
	// Describe is id's title, and false when there is none.
	Describe(id command.ID) (title string, ok bool)
}

// DescriberFunc is a function that is a Describer.
type DescriberFunc func(id command.ID) (title string, ok bool)

// Describe calls f.
func (f DescriberFunc) Describe(id command.ID) (string, bool) { return f(id) }

// FromRegistry describes commands through Registry.Lookup and
// Command.Title. A command it does not hold, or one with no title, has no
// title.
func FromRegistry(r *command.Registry) Describer {
	return DescriberFunc(func(id command.ID) (string, bool) {
		c, ok := r.Lookup(id)
		if !ok || c.Title == "" {
			return "", false
		}
		return c.Title, true
	})
}
