package kongcmd

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/maccavelli/go-tui-lib/command"
)

// property is one argument of a command's schema. It is read as
// command/cli reads it, so that the front ends agree on names, kinds and
// order.
type property struct {
	name, desc string
	types      []string
	items      string // an array's item type
	enum       []any
	required   bool
	hasDefault bool
	def        any
	min, max   *float64
	cli        struct {
		Arg         bool   `json:"arg"`
		Short       string `json:"short"`
		Placeholder string `json:"placeholder"`
		Group       string `json:"group"`
		Hidden      bool   `json:"hidden"`
	}
	kongEnum bool // Kong checks the enum; otherwise the adapter does
}

func (p *property) is(t string) bool { return slices.Contains(p.types, t) }

// JSON Schema's type names, as the shell reads them.
const (
	tString  = "string"
	tInteger = "integer"
	tNumber  = "number"
	tBoolean = "boolean"
	tObject  = "object"
	tArray   = "array"
)

// properties reads the top-level properties of schema, in document order,
// which is the order positionals take (docs/decisions/0006-MADR-command-registry.md A5).
func properties(schema command.Schema) ([]*property, error) {
	if len(bytes.TrimSpace(schema)) == 0 {
		return nil, nil
	}
	var top struct {
		Properties jsontext.Value `json:"properties"`
		Required   []string       `json:"required"`
	}
	if err := json.Unmarshal(schema, &top); err != nil {
		return nil, err
	}
	if len(top.Properties) == 0 {
		return nil, nil
	}
	d := jsontext.NewDecoder(bytes.NewReader(top.Properties))
	if _, err := d.ReadToken(); err != nil {
		return nil, err
	}
	var out []*property
	for d.PeekKind() != '}' {
		tok, err := d.ReadToken()
		if err != nil {
			return nil, err
		}
		name := tok.String()
		v, err := d.ReadValue()
		if err != nil {
			return nil, err
		}
		p, err := readProperty(name, v)
		if err != nil {
			return nil, err
		}
		p.required = slices.Contains(top.Required, name)
		out = append(out, p)
	}
	return out, nil
}

func readProperty(name string, v jsontext.Value) (*property, error) {
	var raw struct {
		Type        jsontext.Value `json:"type"`
		Description string         `json:"description"`
		Enum        []any          `json:"enum"`
		Default     jsontext.Value `json:"default"`
		Minimum     *float64       `json:"minimum"`
		Maximum     *float64       `json:"maximum"`
		Items       struct {
			Type jsontext.Value `json:"type"`
		} `json:"items"`
		CLI jsontext.Value `json:"x-cli"`
	}
	if err := json.Unmarshal(v, &raw); err != nil {
		return nil, fmt.Errorf("property %s: %w", name, err)
	}
	p := &property{name: name, desc: raw.Description, enum: raw.Enum, min: raw.Minimum, max: raw.Maximum}
	p.types = typeList(raw.Type)
	if it := typeList(raw.Items.Type); len(it) > 0 {
		p.items = it[0]
	}
	if len(raw.Default) > 0 {
		p.hasDefault = true
		if err := json.Unmarshal(raw.Default, &p.def); err != nil {
			return nil, fmt.Errorf("property %s: default: %w", name, err)
		}
	}
	if len(raw.CLI) > 0 {
		if err := json.Unmarshal(raw.CLI, &p.cli); err != nil {
			return nil, fmt.Errorf("property %s: x-cli: %w", name, err)
		}
	}
	return p, nil
}

// typeList is a schema's "type", a string or a list of strings.
func typeList(v jsontext.Value) []string {
	var one string
	if json.Unmarshal(v, &one) == nil && one != "" {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(v, &many) == nil {
		return many
	}
	return nil
}

// typeOf is p's own type, the first it names, else string.
func (p *property) typeOf() string {
	for _, t := range p.types {
		if t != "null" {
			return t
		}
	}
	return tString
}

// itemType is an array's item type, string when the schema does not say.
func itemType(p *property) string {
	if p.items != "" {
		return p.items
	}
	return tString
}

// choices is an enum's values as the shell writes them.
func choices(p *property) []string {
	out := make([]string, len(p.enum))
	for i, e := range p.enum {
		out[i] = fmt.Sprint(e)
	}
	return out
}

// goType is the Go type a property's field has in a grammar: an object,
// or an array of objects, is JSON text the adapter decodes.
func goType(t string) reflect.Type {
	switch t {
	case tInteger:
		return reflect.TypeFor[int64]()
	case tNumber:
		return reflect.TypeFor[float64]()
	case tBoolean:
		return reflect.TypeFor[bool]()
	}
	return reflect.TypeFor[string]()
}

func (p *property) goType() reflect.Type {
	if p.is(tArray) {
		return reflect.SliceOf(goType(itemType(p)))
	}
	return goType(p.typeOf())
}

// The names of the shell's own flags, which a property cannot take.
var reserved = []string{"args", "json", "yes", flagHelp, "h"}

// The keys of Kong's tags the adapter writes.
const (
	keyName  = "name"
	keyHelp  = "help"
	keyGroup = "group"
)

// flagHelp is Kong's help flag.
const flagHelp = "help"

// The custom tags the adapter reads back from Kong's model.
const (
	tagID     = "id"     // a registry command's ID
	tagDanger = "danger" // its danger
	tagNS     = "ns"     // a namespace's ID prefix
	tagSelf   = "self"   // a command that is also a parent, as its own child
	tagVerb   = "verb"   // one of the shell's verbs
)

// quote is s as a struct tag's value, with each $ doubled, because Kong
// interpolates ${…} in help, defaults and enums.
func quote(s string) string { return strconv.Quote(strings.ReplaceAll(s, "$", "$$")) }

// tag builds a struct tag from key and value pairs.
func tag(kv ...string) reflect.StructTag {
	var b strings.Builder
	for i := 0; i+1 < len(kv); i += 2 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(kv[i] + ":" + quote(kv[i+1]))
	}
	return reflect.StructTag(b.String())
}

// defaultText is a scalar default as Kong reads it in a default tag.
func defaultText(v any) (string, bool) {
	switch d := v.(type) {
	case string:
		return d, true
	case bool:
		return strconv.FormatBool(d), true
	case float64:
		return strconv.FormatFloat(d, 'f', -1, 64), true
	}
	return "", false
}

// commandType is the grammar of the registry command c: a field for each
// property, Kong-native (docs/decisions/0006-MADR-command-registry.md
// A11), then --args, --json and --yes. A property named like one of the
// shell's flags is left out, and returned as the error the command reports
// when it runs, as command/cli does.
func commandType(props []*property) (reflect.Type, error) {
	var fields []reflect.StructField
	var broken error
	for i, p := range props {
		if slices.Contains(reserved, p.name) {
			if broken == nil {
				broken = fmt.Errorf("the property %q takes the name of a flag the shell keeps", p.name)
			}
			continue
		}
		kv := []string{keyName, p.name, keyHelp, p.desc}
		if p.cli.Arg {
			kv = append(kv, "arg", "")
			if !p.required {
				kv = append(kv, "optional", "")
			}
		} else if p.required {
			kv = append(kv, "required", "")
		}
		if def, ok := defaultText(p.def); ok && p.hasDefault && !p.is(tArray) {
			kv = append(kv, "default", def)
		}
		if len(p.enum) > 0 && !p.is(tArray) && (p.required || p.hasDefault) {
			kv = append(kv, "enum", strings.Join(choices(p), ","))
			p.kongEnum = true
		}
		if s := p.cli.Short; len(s) == 1 && s != "h" && s[0] < 0x80 && !p.cli.Arg {
			kv = append(kv, "short", s)
		}
		if p.cli.Placeholder != "" {
			kv = append(kv, "placeholder", p.cli.Placeholder)
		}
		if p.cli.Hidden {
			kv = append(kv, "hidden", "")
		}
		if p.cli.Group != "" && !p.cli.Arg {
			kv = append(kv, keyGroup, p.cli.Group)
		}
		fields = append(fields, reflect.StructField{Name: "P" + strconv.Itoa(i), Type: p.goType(), Tag: tag(kv...)})
	}
	fields = append(fields,
		reflect.StructField{Name: "ShellArgs", Type: reflect.TypeFor[string](), Tag: tag(keyName, "args", keyHelp, "the arguments as one JSON object", "placeholder", "JSON")},
		reflect.StructField{Name: "ShellJSON", Type: reflect.TypeFor[bool](), Tag: tag(keyName, "json", keyHelp, "print the result's value as JSON")},
		reflect.StructField{Name: "ShellYes", Type: reflect.TypeFor[bool](), Tag: tag(keyName, "yes", keyHelp, "run a destructive command without asking")},
	)
	return reflect.StructOf(fields), broken
}

// node is one word of the tree: a registry command, a namespace of
// commands, or both.
type node struct {
	word     string
	prefix   string // the ID, or the namespace's
	cmd      *entry
	children []*node
}

func (n *node) child(word, prefix string) *node {
	for _, c := range n.children {
		if c.word == word {
			return c
		}
	}
	c := &node{word: word, prefix: prefix}
	n.children = append(n.children, c)
	return c
}

// hidden reports whether every command at and below n is hidden.
func (n *node) hidden() bool {
	if n.cmd != nil && !n.cmd.c.Hidden {
		return false
	}
	for _, c := range n.children {
		if !c.hidden() {
			return false
		}
	}
	return true
}

// group is n's category: its command's, or the one every visible command
// below it shares.
func (n *node) group() string {
	if n.cmd != nil {
		return n.cmd.c.Category
	}
	group, seen := "", false
	for _, c := range n.children {
		if c.hidden() {
			continue
		}
		g := c.group()
		if seen && g != group {
			return ""
		}
		group, seen = g, true
	}
	return group
}

// nodeType is the grammar of n: a registry command's fields, or a struct
// of its children's command fields, with n's own command as a hidden
// default child when it has both (A11).
func nodeType(n *node) (reflect.Type, error) {
	if len(n.children) == 0 {
		return n.cmd.typ, nil
	}
	var fields []reflect.StructField
	taken := map[string]bool{}
	for i, c := range n.children {
		t, err := nodeType(c)
		if err != nil {
			return nil, err
		}
		fields = append(fields, reflect.StructField{Name: "C" + strconv.Itoa(i), Type: t, Tag: cmdTag(c, taken)})
		taken[c.word] = true
	}
	if n.cmd != nil {
		if taken[n.word] {
			return nil, fmt.Errorf("kongcmd: %s has a child named like itself, and cannot be its own default child", n.cmd.c.ID)
		}
		kv := []string{"cmd", "", keyName, n.word, keyHelp, n.cmd.c.Title, "default", "withargs", "hidden", "",
			tagID, string(n.cmd.c.ID), tagDanger, n.cmd.c.Danger.String(), tagSelf, "1"}
		fields = append(fields, reflect.StructField{Name: "Self", Type: n.cmd.typ, Tag: tag(kv...)})
	}
	return reflect.StructOf(fields), nil
}

// commandTags are the tags that name n in its parent: its word, help,
// group, aliases, hiddenness, and its ID and danger or namespace.
func commandTags(n *node, taken map[string]bool) []string {
	kv := []string{keyName, n.word}
	if n.cmd != nil {
		kv = append(kv, keyHelp, n.cmd.c.Title, tagID, string(n.cmd.c.ID), tagDanger, n.cmd.c.Danger.String())
		var aliases []string
		for _, a := range n.cmd.c.Aliases {
			if a != "" && a != n.word && !taken[a] && !strings.ContainsAny(a, " \t.,") {
				aliases = append(aliases, a)
				taken[a] = true
			}
		}
		if len(aliases) > 0 {
			kv = append(kv, "aliases", strings.Join(aliases, ","))
		}
	} else {
		kv = append(kv, keyHelp, "The "+n.word+" commands")
	}
	kv = append(kv, tagNS, n.prefix)
	if g := n.group(); g != "" {
		kv = append(kv, keyGroup, g)
	}
	if n.hidden() {
		kv = append(kv, "hidden", "")
	}
	return kv
}

func cmdTag(n *node, taken map[string]bool) reflect.StructTag {
	return tag(append([]string{"cmd", ""}, commandTags(n, taken)...)...)
}

// verbType is the grammar of one of the shell's verbs.
func verbType(verb string) reflect.Type {
	var fields []reflect.StructField
	switch verb {
	case verbList:
		fields = append(fields, reflect.StructField{Name: "JSON", Type: reflect.TypeFor[bool](), Tag: tag(keyName, "json", keyHelp, "print the manifest of the commands as JSON")})
	case verbDescribe, verbSchema, verbHelp:
		fields = append(fields, reflect.StructField{Name: "Command", Type: reflect.TypeFor[[]string](), Tag: tag("arg", "", "optional", "", keyName, "command", keyHelp, "the command, by its words or its ID", "placeholder", "COMMAND")})
	case verbCompletion:
		fields = append(fields, reflect.StructField{Name: "Shell", Type: reflect.TypeFor[string](), Tag: tag("arg", "", keyName, "shell", "enum", "bash,zsh,fish,powershell", keyHelp, "the shell")})
	}
	return reflect.StructOf(fields)
}

// The verbs.
const (
	verbList       = "list"
	verbDescribe   = "describe"
	verbSchema     = "schema"
	verbHelp       = "help"
	verbCompletion = "completion"
)

var verbs = []struct{ name, help string }{
	{verbList, "List the commands you can run now"},
	{verbDescribe, "Describe a command as JSON"},
	{verbSchema, "Print a command's arguments schema"},
	{verbHelp, "Show the help of the program or of a command"},
	{verbCompletion, "Generate the completion script for a shell"},
}

// checkClashes fails a parser in which two commands of one parent share a
// name or an alias (A11): Kong itself lets the first win.
func checkClashes(k *kong.Kong) error {
	var clashes []string
	var visit func(n *kong.Node)
	visit = func(n *kong.Node) {
		seen := map[string]bool{}
		for _, c := range n.Children {
			for _, name := range append([]string{c.Name}, c.Aliases...) {
				if seen[name] {
					clashes = append(clashes, strconv.Quote(strings.TrimSpace(n.Path()+" "+name)))
				}
				seen[name] = true
			}
			visit(c)
		}
	}
	visit(k.Model.Node)
	if len(clashes) > 0 {
		return fmt.Errorf("kongcmd: two commands are named %s", strings.Join(clashes, ", "))
	}
	return nil
}
