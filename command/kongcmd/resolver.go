package kongcmd

import (
	"github.com/alecthomas/kong"
)

// configPrefix starts the context keys Resolver reads, as VS Code offers
// its settings as config.* keys of the when-clause context.
const configPrefix = "config."

// Resolver gives a Kong parser values from the settings: a flag x takes
// the value of the key config.x in the context given with WithContext,
// unless the command line gives it
// (docs/decisions/0006-MADR-command-registry.md A11). The values are the
// keys' text, which Kong parses as it parses the command line; a list's
// items are joined with commas, which a slice flag splits. Pass it with
// kong.Resolvers.
func (a *Adapter) Resolver() kong.Resolver {
	return kong.ResolverFunc(func(_ *kong.Context, _ *kong.Path, f *kong.Flag) (any, error) {
		v, ok := a.cfg.context.Value(configPrefix + f.Name)
		if !ok {
			return nil, nil
		}
		return v.String(), nil
	})
}
