package command

import (
	"context"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
)

// NoArgs is the argument type of a command without arguments. Its schema
// is {"type": "object", "additionalProperties": false}.
type NoArgs struct{}

// ArgError is an argument that does not fit its schema: Path is a JSON
// Pointer to it ("" for the arguments as a whole), and Reason says why.
// Find it with errors.AsType[*ArgError].
type ArgError struct {
	Path   string
	Reason string
}

func (e *ArgError) Error() string {
	if e.Path == "" {
		return "command: arguments: " + e.Reason
	}
	return "command: argument " + e.Path + ": " + e.Reason
}

// ExitCode is 2, the status of a usage error
// (docs/decisions/0014-MADR-native-integration-api.md A1).
func (e *ArgError) ExitCode() int { return 2 }

// Option sets a field of the Command New builds. It is opaque: only this
// package's functions make one.
type Option interface{ apply(*Command) }

// optionFunc is a function used as an Option.
type optionFunc func(*Command)

func (f optionFunc) apply(c *Command) { f(c) }

// New builds a command whose arguments decode into A. The schema is
// SchemaOf[A], which must be an object; the registry checks arguments
// against it before the command runs, and run receives them decoded
// strictly into an A. It is an error to give no WithDanger
// (docs/decisions/0006-MADR-command-registry.md A5).
func New[A any](id ID, title string, run func(ctx context.Context, inv *Invocation, args A) (Result, error), opts ...Option) (Command, error) {
	if run == nil {
		return Command{}, fmt.Errorf("command: %s: no run function", id)
	}
	if err := id.Valid(); err != nil {
		return Command{}, err
	}
	schema, err := SchemaOf[A]()
	if err != nil {
		return Command{}, fmt.Errorf("command: %s: %w", id, err)
	}
	r, err := compileRule(schema)
	if err != nil {
		return Command{}, fmt.Errorf("command: %s: args schema: %w", id, err)
	}
	if !r.is(tObject) {
		return Command{}, fmt.Errorf("command: %s: arguments must be an object, not %v", id, r.types)
	}
	c := Command{
		ID:    id,
		Title: title,
		Args:  schema,
		Handler: HandlerFunc(func(ctx context.Context, inv *Invocation) (Result, error) {
			a, err := decodeArgs[A](inv.Args)
			if err != nil {
				return Result{}, err
			}
			return run(ctx, inv, a)
		}),
	}
	for _, o := range opts {
		o.apply(&c)
	}
	if c.Danger == 0 {
		return Command{}, errors.New("command: " + string(id) + ": no WithDanger; declare ReadOnly, UI, Mutating or Destructive")
	}
	return c, nil
}

// MustNew is New, and panics on its error: for a command whose definition
// is fixed in the program, where an error is a bug.
func MustNew[A any](id ID, title string, run func(ctx context.Context, inv *Invocation, args A) (Result, error), opts ...Option) Command {
	c, err := New(id, title, run, opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// ArgsOf encodes a as a request's arguments: deterministic JSON, with a
// time.Duration in the string form New's schema accepts ("1m30s"). It is
// how a program's own command line builds Request.Args from its flags.
func ArgsOf[A any](a A) (json.RawMessage, error) {
	b, err := jsonv2.Marshal(a, jsonv2.Deterministic(true), jsonv2.WithMarshalers(durationMarshalers))
	if err != nil {
		return nil, fmt.Errorf("command: arguments: %w", err)
	}
	return b, nil
}

// WithDescription sets Description.
func WithDescription(s string) Option { return optionFunc(func(c *Command) { c.Description = s }) }

// WithCategory sets Category.
func WithCategory(s string) Option { return optionFunc(func(c *Command) { c.Category = s }) }

// WithSlash sets the slash name and its aliases.
func WithSlash(name string, aliases ...string) Option {
	return optionFunc(func(c *Command) { c.Slash, c.Aliases = name, aliases })
}

// WithArgHint sets ArgHint.
func WithArgHint(s string) Option { return optionFunc(func(c *Command) { c.ArgHint = s }) }

// WithOutput sets Output, the schema of Result.Value.
func WithOutput(s Schema) Option { return optionFunc(func(c *Command) { c.Output = s }) }

// WithWhen sets When.
func WithWhen(expr string) Option { return optionFunc(func(c *Command) { c.When = expr }) }

// WithScope sets Scope.
func WithScope(s Scope) Option { return optionFunc(func(c *Command) { c.Scope = s }) }

// WithDanger sets Danger. New requires it.
func WithDanger(d Danger) Option { return optionFunc(func(c *Command) { c.Danger = d }) }

// WithIdempotent marks the command idempotent.
func WithIdempotent() Option { return optionFunc(func(c *Command) { c.Idempotent = true }) }

// WithOpenWorld marks the command as reaching outside the program.
func WithOpenWorld() Option { return optionFunc(func(c *Command) { c.OpenWorld = true }) }

// WithSurfaces sets Surfaces.
func WithSurfaces(s Surface) Option { return optionFunc(func(c *Command) { c.Surfaces = s }) }

// WithMode sets Mode.
func WithMode(m Mode) Option { return optionFunc(func(c *Command) { c.Mode = m }) }

// WithExclusive makes a new run cancel the running one.
func WithExclusive() Option { return optionFunc(func(c *Command) { c.Exclusive = true }) }

// WithWhileBusy lets the command run while the agent streams.
func WithWhileBusy() Option { return optionFunc(func(c *Command) { c.WhileBusy = true }) }

// WithHidden keeps the command runnable but unlisted.
func WithHidden() Option { return optionFunc(func(c *Command) { c.Hidden = true }) }

// WithMeta sets Meta[key].
func WithMeta(key string, v any) Option {
	return optionFunc(func(c *Command) {
		if c.Meta == nil {
			c.Meta = map[string]any{}
		}
		c.Meta[key] = v
	})
}
