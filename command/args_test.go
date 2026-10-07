package command

import (
	"context"
	"strings"
	"testing"
)

// build is New of A, with a run function that does nothing.
func build[A any]() error {
	_, err := New("t.cmd", "T", func(context.Context, *Invocation, A) (Result, error) { return Result{}, nil }, WithDanger(UI))
	return err
}

type (
	requiredFlag struct {
		Name string `json:"name"`
	}
	requiredFlagFixed struct {
		Name string `json:"name" required:""`
	}
	optionalArg struct {
		Pane string `json:"pane,omitzero" arg:""`
	}
	optionalArgFixed struct {
		Pane string `json:"pane,omitzero" arg:"" optional:""`
	}
	requiredOmitted struct {
		Name string `json:"name,omitzero" required:""`
	}
	requiredOptionalArg struct {
		Pane string `json:"pane" arg:"" optional:""`
	}
	snakeName struct {
		MaxItems int `json:"max_items,omitzero"`
	}
	snakeNameFixed struct {
		MaxItems int `json:"max_items,omitzero" name:"max_items"`
	}
	bothTags struct {
		Name string `json:"name" required:"" optional:""`
	}
	argWithDefault struct {
		Mode string `json:"mode" arg:"" enum:"a,b" default:"a"`
	}
	pointerFlag struct {
		N *int `json:"n"`
	}
	embedded struct {
		requiredFlagFixed
		Extra bool `json:"extra,omitzero"`
	}
	embeddedWrong struct {
		requiredFlag
	}
	nestedRequired struct {
		Inner struct {
			Name string `json:"name"`
		} `json:"inner,omitzero"`
	}
	ownSchemaArgs struct {
		Whatever string `json:"x_y"`
	}
)

func (ownSchemaArgs) JSONSchema() Schema {
	return Schema(`{"type":"object","properties":{"x_y":{"type":"string"}}}`)
}

func TestNewAgreesWithKong(t *testing.T) {
	for name, c := range map[string]struct {
		build func() error
		fix   string // the tag the error names; "" for a struct New accepts
	}{
		"a required flag without required":    {build[requiredFlag], `add required:""`},
		"an optional positional":              {build[optionalArg], `add optional:""`},
		"required beside omitzero":            {build[requiredOmitted], `remove required:""`},
		"optional on a required positional":   {build[requiredOptionalArg], `remove optional:""`},
		"a json name Kong spells differently": {build[snakeName], `add name:"max_items"`},
		"required and optional together":      {build[bothTags], `required:"" and optional:"" together`},
		"an embedded struct's field":          {build[embeddedWrong], `add required:""`},
		"a required flag, fixed":              {build[requiredFlagFixed], ""},
		"an optional positional, fixed":       {build[optionalArgFixed], ""},
		"a json name, fixed":                  {build[snakeNameFixed], ""},
		"a positional with a default":         {build[argWithDefault], ""},
		"a pointer flag":                      {build[pointerFlag], ""},
		"an embedded struct, fixed":           {build[embedded], ""},
		"a nested object's fields":            {build[nestedRequired], ""},
		"a type with its own schema":          {build[ownSchemaArgs], ""},
		"no arguments":                        {build[NoArgs], ""},
	} {
		err := c.build()
		switch {
		case c.fix == "" && err != nil:
			t.Errorf("%s: %v", name, err)
		case c.fix != "" && (err == nil || !strings.Contains(err.Error(), c.fix)):
			t.Errorf("%s: %v, want an error naming %s", name, err, c.fix)
		}
	}
	// SchemaOf is unchanged: it still describes a struct New refuses.
	for name, f := range map[string]func() (Schema, error){
		"requiredFlag": SchemaOf[requiredFlag], "optionalArg": SchemaOf[optionalArg], "snakeName": SchemaOf[snakeName],
	} {
		if _, err := f(); err != nil {
			t.Errorf("SchemaOf[%s]: %v", name, err)
		}
	}
}

func TestKongName(t *testing.T) {
	for goName, want := range map[string]string{
		"N":          "n",
		"Name":       "name",
		"MaxItems":   "max-items",
		"PaneID":     "pane-id",
		"HTTPServer": "http-server",
		"ID":         "id",
		"X2":         "x-2",
		"Base64Data": "base-64-data",
		"AB":         "ab",
		"ABc":        "a-bc",
		"Max_Items":  "max-_-items",
		"Ünïcode":    "ünïcode",
	} {
		if got := kongName(goName); got != want {
			t.Errorf("kongName(%q) = %q, want %q", goName, got, want)
		}
	}
}
