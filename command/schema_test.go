package command

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/tuitest"
)

type common struct {
	Verbose bool `json:"verbose,omitzero" short:"v" group:"Output" help:"say more"`
}

type flatArgs struct {
	Split string  `json:"split" arg:"" placeholder:"SPLIT" help:"the split whose separator moves"`
	Count int     `json:"count" help:"how many"`
	Rate  float64 `json:"rate,omitzero"`
	On    bool    `json:"on,omitempty"`
	Debug bool    `json:"debug,omitzero" hidden:""`
	common
	Ignored int `json:"-"`
}

type nestedArgs struct {
	Pane struct {
		ID   string `json:"id"`
		Size int    `json:"size,omitzero"`
	} `json:"pane"`
	Named common `json:"named,omitzero"`
}

type sliceArgs struct {
	Tags  []string   `json:"tags" arg:""`
	Sizes []int      `json:"sizes,omitzero"`
	Data  []byte     `json:"data,omitzero"`
	Fixed [2]float64 `json:"fixed,omitzero"`
}

type pointerArgs struct {
	Name  *string     `json:"name"`
	N     *int        `json:"n"`
	Inner *nestedArgs `json:"inner"`
}

type enumArgs struct {
	Mode  string   `json:"mode" enum:"fast,slow"`
	Level int      `json:"level,omitzero" enum:"1,2,3" default:"2"`
	Tags  []string `json:"tags,omitzero" enum:"a,b"`
}

type boundsArgs struct {
	Delta int    `json:"delta" schema:"min=-200,max=200"`
	Name  string `json:"name" schema:"minLen=1,maxLen=8"`
	Token string `json:"token,omitzero" schema:"secret"`
	Count uint   `json:"count,omitzero"`
	Ratio uint8  `json:"ratio,omitzero" schema:"max=100"`
}

type durationArgs struct {
	Timeout time.Duration `json:"timeout" default:"30s"`
	Every   time.Duration `json:"every,omitzero"`
}

type mapArgs struct {
	Resize map[string]int    `json:"resize,omitzero"`
	Labels map[string]string `json:"labels"`
}

// colorArg is a type that gives its own schema.
type colorArg string

func (colorArg) JSONSchema() Schema {
	return Schema(`{"type":"string","pattern":"^#[0-9a-f]{6}$"}`)
}

type customArgs struct {
	Color colorArg  `json:"color"`
	Back  *colorArg `json:"back"`
}

// indent is s indented two spaces, with a final newline.
func indent(t *testing.T, s Schema) string {
	t.Helper()
	v := s
	if err := v.Indent(jsontext.WithIndent("  ")); err != nil {
		t.Fatal(err)
	}
	return string(v) + "\n"
}

func TestSchemaGolden(t *testing.T) {
	for name, of := range map[string]func() (Schema, error){
		"flat":     SchemaOf[flatArgs],
		"nested":   SchemaOf[nestedArgs],
		"slices":   SchemaOf[sliceArgs],
		"pointers": SchemaOf[pointerArgs],
		"enum":     SchemaOf[enumArgs],
		"bounds":   SchemaOf[boundsArgs],
		"duration": SchemaOf[durationArgs],
		"map":      SchemaOf[mapArgs],
		"custom":   SchemaOf[customArgs],
		"noargs":   SchemaOf[NoArgs],
	} {
		s, err := of()
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		var top map[string]any
		if err := json.Unmarshal(s, &top); err != nil {
			t.Errorf("%s: not JSON: %v", name, err)
			continue
		}
		if top["$schema"] != schemaDialect {
			t.Errorf("%s: $schema is %v", name, top["$schema"])
		}
		tuitest.Text(t, "schema-"+name, indent(t, s))
	}
}

type selfRef struct {
	Next *selfRef `json:"next"`
}

type badDefault struct {
	N int `json:"n" default:"many"`
}

func TestSchemaRefuses(t *testing.T) {
	for name, of := range map[string]func() (Schema, error){
		"a channel":        SchemaOf[struct{ C chan int }],
		"a func":           SchemaOf[struct{ F func() }],
		"an interface":     SchemaOf[struct{ X any }],
		"a complex number": SchemaOf[struct{ Z complex128 }],
		"an int-keyed map": SchemaOf[struct{ M map[int]string }],
		"a recursive type": SchemaOf[selfRef],
		"an optional enum": SchemaOf[struct {
			Mode string `json:"mode,omitzero" enum:"a,b"`
		}],
		"an optional pointer enum": SchemaOf[struct {
			Mode *string `json:"mode" enum:"a,b"`
		}],
		"the json string option": SchemaOf[struct {
			N int `json:"n,string"`
		}],
		"an unknown schema key": SchemaOf[struct {
			N int `json:"n" schema:"step=2"`
		}],
		"min on a string": SchemaOf[struct {
			S string `json:"s" schema:"min=1"`
		}],
		"minLen on an int": SchemaOf[struct {
			N int `json:"n" schema:"minLen=1"`
		}],
		"a negative maxLen": SchemaOf[struct {
			S string `json:"s" schema:"maxLen=-1"`
		}],
		"a default that is no int": SchemaOf[badDefault],
		"a default outside an enum": SchemaOf[struct {
			M string `json:"m" enum:"a,b" default:"c"`
		}],
		"a bad duration default": SchemaOf[struct {
			D time.Duration `json:"d" default:"soon"`
		}],
		"an enum on a struct": SchemaOf[struct {
			S common `json:"s" enum:"x"`
		}],
		"two fields with one name": SchemaOf[struct {
			X int
			Y int `json:"X"`
		}],
		"tags on an own schema": SchemaOf[struct {
			C colorArg `json:"c" help:"no"`
		}],
	} {
		if _, err := of(); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// resizeArgs is MADR A2's example, with every Kong tag.
type resizeArgs struct {
	Split string `json:"split" arg:"" help:"the split whose separator moves" placeholder:"SPLIT"`
	Delta int    `json:"delta" arg:"" help:"cells to give the pane before it; negative takes" schema:"min=-200,max=200"`
	Quiet bool   `json:"quiet,omitzero" short:"q" group:"Output" hidden:""`
}

func TestSchemaKeepsUnknownKeys(t *testing.T) {
	s, err := SchemaOf[resizeArgs]()
	if err != nil {
		t.Fatal(err)
	}
	var top struct {
		Properties map[string]map[string]any `json:"properties"`
		Required   []string                  `json:"required"`
	}
	if err := json.Unmarshal(s, &top); err != nil {
		t.Fatal(err)
	}
	split, delta, quiet := top.Properties["split"], top.Properties["delta"], top.Properties["quiet"]
	if split == nil || delta == nil || quiet == nil || top.Properties["Split"] != nil {
		t.Fatalf("properties are not named by their json tags: %v", top.Properties)
	}
	if delta["minimum"] != -200.0 || delta["maximum"] != 200.0 {
		t.Errorf("the schema tag Kong does not read is lost: %v", delta)
	}
	want := map[string]map[string]any{
		"split": {"arg": true, "placeholder": "SPLIT"},
		"delta": {"arg": true},
		"quiet": {"short": "q", "group": "Output", "hidden": true},
	}
	for name, cli := range want {
		got, _ := top.Properties[name]["x-cli"].(map[string]any)
		if len(got) != len(cli) {
			t.Errorf("%s: x-cli %v, want %v", name, got, cli)
			continue
		}
		for k, v := range cli {
			if got[k] != v {
				t.Errorf("%s: x-cli %s = %v, want %v", name, k, got[k], v)
			}
		}
	}
	if split["description"] != "the split whose separator moves" {
		t.Errorf("help: %v", split["description"])
	}
}
