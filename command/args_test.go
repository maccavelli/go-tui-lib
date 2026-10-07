package command

import (
	"context"
	json "encoding/json/v2"
	"slices"
	"testing"
)

type (
	// A flag the json tag makes required, with no other tag.
	requiredFlag struct {
		Name string `json:"name"`
	}
	// A positional the json tag makes optional, with no other tag.
	optionalArg struct {
		Pane string `json:"pane,omitzero" arg:""`
	}
	// A json name that is not a kebab-case spelling of the Go name.
	snakeName struct {
		MaxItems int `json:"max_items,omitzero"`
	}
)

// TestNewTakesJSONRule shows that New decides a property's name and
// whether it is required by the json tag alone, as SchemaOf does: a
// program's own CLI reads its arguments however it likes
// (docs/decisions/0012-MADR-bring-your-own-cli.md).
func TestNewTakesJSONRule(t *testing.T) {
	for name, c := range map[string]struct {
		build    func() (Command, error)
		property string
		required bool
	}{
		"a flag required by its json tag": {newOf[requiredFlag], "name", true},
		"a positional with omitzero":      {newOf[optionalArg], "pane", false},
		"a json name in another spelling": {newOf[snakeName], "max_items", false},
	} {
		c0, err := c.build()
		if err != nil {
			t.Errorf("%s: New refused it: %v", name, err)
			continue
		}
		var schema struct {
			Properties map[string]any `json:"properties"`
			Required   []string       `json:"required"`
		}
		if err := json.Unmarshal(c0.Args, &schema); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, ok := schema.Properties[c.property]; !ok {
			t.Errorf("%s: no property %q in %s", name, c.property, c0.Args)
		}
		if got := slices.Contains(schema.Required, c.property); got != c.required {
			t.Errorf("%s: %q required %v, want %v (%s)", name, c.property, got, c.required, c0.Args)
		}
	}
}

// newOf is New of A, with a run function that does nothing.
func newOf[A any]() (Command, error) {
	return New("t.cmd", "T", func(context.Context, *Invocation, A) (Result, error) { return Result{}, nil }, WithDanger(UI))
}
