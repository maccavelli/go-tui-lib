package keymap

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/maccavelli/go-tui-lib/v2/command"
)

// schemaDialect is the JSON Schema dialect Schema's document names.
const schemaDialect = "https://json-schema.org/draft/2020-12/schema"

// tString is JSON Schema's name for a string.
const tString = "string"

// keyPattern is the key grammar as a regular expression, in the syntax Go
// and ECMA-262, JSON Schema's, share: one to MaxStrokes strokes separated
// by one space, each <leader>, or modifiers and a key joined by "+" with an
// optional event. ECMA-262 has no flag for case inside a pattern, so each
// letter is a class of both cases. A one-character key is any character
// but space and the controls; ParseStroke checks the rest.
var keyPattern = func() string {
	alt := func(names []string) string {
		slices.Sort(names)
		for i, n := range names {
			names[i] = caseless(n)
		}
		return "(?:" + strings.Join(names, "|") + ")"
	}
	keys := slices.Collect(maps.Keys(nameCodes))
	keys = slices.AppendSeq(keys, maps.Keys(keyAliases))
	key := "(?:" + alt(keys) + `|\[` + alt(slices.Collect(maps.Keys(scanCodes))) + `\]|[^\x00-\x20\x7f])`
	stroke := "(?:" + alt(slices.Collect(maps.Keys(modifierNames))) + `\+)*` + key +
		"(?::" + alt([]string{Release.String(), Repeat.String()}) + ")?"
	token := "(?:" + caseless(leaderName) + "|" + stroke + ")"
	return "^" + token + "(?: " + token + "){0," + strconv.Itoa(MaxStrokes-1) + "}$"
}()

// caseless is s as a pattern matching it in either case.
func caseless(s string) string {
	var b strings.Builder
	for _, r := range s {
		if lo, up := unicode.ToLower(r), unicode.ToUpper(r); lo != up {
			b.WriteString("[" + string(lo) + string(up) + "]")
		} else {
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	return b.String()
}

// The schema's parts, in the order an editor shows them.
type (
	schemaDoc struct {
		Schema string     `json:"$schema"`
		Title  string     `json:"title"`
		Type   string     `json:"type"`
		Items  schemaItem `json:"items"`
	}
	schemaItem struct {
		Type                 string       `json:"type"`
		Properties           schemaProps  `json:"properties"`
		Required             []string     `json:"required"`
		AdditionalProperties bool         `json:"additionalProperties"`
		AllOf                []schemaRule `json:"allOf"`
	}
	schemaProps struct {
		Key     schemaText `json:"key"`
		Command schemaText `json:"command"`
		When    schemaText `json:"when"`
		Args    schemaText `json:"args"`
		Context schemaText `json:"context"`
	}
	schemaText struct {
		Type        string   `json:"type,omitzero"`
		Description string   `json:"description"`
		Pattern     string   `json:"pattern,omitzero"`
		MinLength   int      `json:"minLength,omitzero"`
		Enum        []string `json:"enum,omitzero"`
	}
	schemaRule struct {
		If   schemaIf   `json:"if"`
		Then schemaThen `json:"then"`
	}
	schemaIf struct {
		Properties map[string]schemaMatch `json:"properties"`
		Required   []string               `json:"required,omitzero"`
	}
	schemaMatch struct {
		Const   string `json:"const,omitzero"`
		Pattern string `json:"pattern,omitzero"`
	}
	schemaThen struct {
		Required   []string                  `json:"required,omitzero"`
		Properties map[string]jsontext.Value `json:"properties,omitzero"`
	}
)

// Schema is a JSON Schema (draft 2020-12) for the files LoadVSCode reads,
// indented with tabs, so that an editor completes and checks a user's
// file (MADR §8). "command" is one of ids or its "-" form; "key" has the
// key grammar's pattern, and is required unless the command begins with
// "-"; and where args gives a schema for an ID, which may be nil, an
// if/then pair checks that command's "args". An args schema is an object
// or a boolean; its "$schema", if any, is left out, as JSON Schema allows
// it only at a document's root. An invalid ID or args schema is an error.
func Schema(ids []command.ID, args func(command.ID) jsontext.Value) ([]byte, error) {
	var names, removes []string
	doc := schemaDoc{
		Schema: schemaDialect, Title: "keybindings", Type: "array",
		Items: schemaItem{
			Type: "object",
			Properties: schemaProps{
				Key: schemaText{Type: tString, Pattern: keyPattern,
					Description: `The keys: up to 4 strokes separated by a space, such as "ctrl+k ctrl+s" or "<leader> f".`},
				Command: schemaText{Type: tString,
					Description: `The command's ID; a "-" before it removes a binding an earlier layer made.`},
				When:    schemaText{Type: tString, Description: "A when expression; the binding applies only while it holds."},
				Args:    schemaText{Description: "The arguments passed to the command."},
				Context: schemaText{Type: tString, MinLength: 1, Description: `The scope: "global", the default, or a pane, overlay or mode.`},
			},
			Required: []string{memberCommand},
			AllOf: []schemaRule{{
				If:   schemaIf{Properties: map[string]schemaMatch{memberCommand: {Pattern: "^[^-]"}}},
				Then: schemaThen{Required: []string{memberKey}},
			}},
		},
	}
	seen := map[command.ID]bool{}
	for _, id := range ids {
		if err := id.Valid(); err != nil {
			return nil, fmt.Errorf("keymap: Schema: %w", err)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		names, removes = append(names, string(id)), append(removes, "-"+string(id))
		if args == nil {
			continue
		}
		s, err := argsSchema(args(id))
		if err != nil {
			return nil, fmt.Errorf("keymap: Schema: the args schema of %s: %w", id, err)
		}
		if s != nil {
			doc.Items.AllOf = append(doc.Items.AllOf, schemaRule{
				If:   schemaIf{Properties: map[string]schemaMatch{memberCommand: {Const: string(id)}}, Required: []string{memberCommand}},
				Then: schemaThen{Properties: map[string]jsontext.Value{memberArgs: s}},
			})
		}
	}
	names = append(names, removes...)
	doc.Items.Properties.Command.Enum = names
	out, err := json.Marshal(doc, jsontext.WithIndent("\t"))
	if err != nil {
		return nil, fmt.Errorf("keymap: Schema: %w", err)
	}
	return append(out, '\n'), nil
}

// argsSchema is s, a schema for a command's args, without its "$schema";
// nil when s is empty.
func argsSchema(s jsontext.Value) (jsontext.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	if !s.IsValid() {
		return nil, errors.New("not valid JSON")
	}
	switch s.Kind() {
	case 't', 'f':
		return slices.Clone(s), nil
	case '{':
	default:
		return nil, fmt.Errorf("%s, not an object or a boolean", kindName(s.Kind()))
	}
	var b bytes.Buffer
	d, e := jsontext.NewDecoder(bytes.NewReader(s)), jsontext.NewEncoder(&b)
	if err := copyToken(d, e); err != nil { // {
		return nil, err
	}
	for d.PeekKind() != '}' {
		name, err := d.ReadToken()
		if err != nil {
			return nil, err
		}
		if name.String() == "$schema" {
			if err := d.SkipValue(); err != nil {
				return nil, err
			}
			continue
		}
		// The name is written first: the next read voids it.
		err = e.WriteToken(name)
		var v jsontext.Value
		if err == nil {
			v, err = d.ReadValue()
		}
		if err == nil {
			err = e.WriteValue(v)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := copyToken(d, e); err != nil { // }
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}

// copyToken reads one token from d and writes it to e.
func copyToken(d *jsontext.Decoder, e *jsontext.Encoder) error {
	t, err := d.ReadToken()
	if err == nil {
		err = e.WriteToken(t)
	}
	return err
}
