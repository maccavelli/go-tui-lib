package command

import (
	json "encoding/json/v2"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// A program's own command line: an argument vector parsed into a request,
// a command's parameters for its flags and help, and completion
// (docs/decisions/0014-MADR-native-integration-api.md W2).

// ParseArgs reads a command line's arguments for command id, as a shell
// split them, into a request from origin o, as ParseSlash does for a slash
// line:
//   - --name=value, --name value, and -s value for a property with x-cli
//     short "s" (-s=value too); the flag names are the arguments' JSON
//     property names;
//   - a bare --name or -s sets a boolean to true, and --name=false sets
//     it false; a boolean never takes the next word;
//   - a repeated flag appends to an array, and sets anything else once;
//   - "--" ends the flags, and every word after it is positional; "-" is
//     a positional value;
//   - positional values fill the positional properties as ParseSlash
//     fills them; a command whose only argument is a string takes every
//     positional word, joined with spaces; a command that allows stray
//     words (x-cli rest) keeps them, and unknown flags, in Raw only.
//
// There is no name=value form, since the shell has already split the
// words, and no --no-name form. A word starting with "-" is a flag, so a
// negative positional number follows "--".
//
// Raw is the arguments joined with spaces, each one single-quoted when it
// needs it, so $ARGUMENTS expands to what the user typed. Args is checked
// against the schema, defaults included. An unknown id wraps ErrUnknown;
// an unknown flag, a missing or bad value, or a missing required argument
// is an *ArgError.
func (r *Registry) ParseArgs(id ID, args []string, o Origin) (Request, error) {
	snap := r.snap.Load()
	i, ok := snap.byID[id]
	if !ok {
		return Request{}, fmt.Errorf("%w: %q", ErrUnknown, id)
	}
	e := &snap.entries[i]
	req := Request{ID: id, Raw: quoteArgs(args), Origin: o}
	if e.args == nil {
		return req, nil
	}
	obj, err := flagObject(e.args, args)
	if err != nil {
		return Request{}, err
	}
	raw, err := json.Marshal(obj, json.Deterministic(true))
	if err != nil {
		return Request{}, &ArgError{Reason: reasonOf(err)}
	}
	if req.Args, err = e.args.prepare(raw); err != nil {
		return Request{}, err
	}
	return req, nil
}

// flagObject builds the arguments object from a command line.
func flagObject(r *rule, args []string) (map[string]any, error) {
	obj := map[string]any{}
	var words []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			words = append(words, args[i+1:]...)
			break
		}
		if a == "-" || !strings.HasPrefix(a, "-") {
			words = append(words, a)
			continue
		}
		flag, value, hasValue := strings.Cut(a, "=")
		name, p := r.flag(flag)
		if p == nil {
			if r.rest {
				continue // it stays in Raw, for $ARGUMENTS
			}
			return nil, &ArgError{Reason: fmt.Sprintf("%s is not a known flag", flag)}
		}
		if !hasValue {
			if p.isOnly(tBoolean) {
				value = strconv.FormatBool(true)
			} else {
				if i+1 >= len(args) {
					return nil, &ArgError{Path: "/" + pointerToken(name), Reason: flag + " needs a value"}
				}
				i++
				value = args[i]
			}
		}
		if err := assign(obj, name, p, value); err != nil {
			return nil, err
		}
	}
	if len(r.props) == 1 && len(r.props[0].r.types) == 1 && r.props[0].r.is(tString) { // as slashObject
		if len(words) > 0 {
			if err := assign(obj, r.props[0].name, r.props[0].r, strings.Join(words, " ")); err != nil {
				return nil, err
			}
		}
		return obj, nil
	}
	return obj, positionals(r, obj, words)
}

// positionals fills r's positional properties from words, as slashObject
// does.
func positionals(r *rule, obj map[string]any, words []string) error {
	var positional []prule
	for _, p := range r.props {
		if p.r.arg {
			positional = append(positional, p)
		}
	}
	next := 0
	for _, w := range words {
		// A positional already given by flag is skipped.
		for next < len(positional) && !positional[next].r.is(tArray) && obj[positional[next].name] != nil {
			next++
		}
		if next >= len(positional) {
			if r.rest {
				continue
			}
			return &ArgError{Reason: fmt.Sprintf("%q is one positional value too many", w)}
		}
		p := positional[next]
		if err := assign(obj, p.name, p.r, w); err != nil {
			return err
		}
		if !p.r.is(tArray) {
			next++
		}
	}
	return nil
}

// flag finds the property a flag names: --name by its property name, -s
// by its x-cli short.
func (r *rule) flag(f string) (string, *rule) {
	if long, ok := strings.CutPrefix(f, "--"); ok {
		return long, r.prop(long)
	}
	short := strings.TrimPrefix(f, "-")
	for _, p := range r.props {
		if p.r.short != "" && p.r.short == short {
			return p.name, p.r
		}
	}
	return "", nil
}

// isOnly reports whether r's one type, null aside, is t.
func (r *rule) isOnly(t string) bool { return r.firstType() == t && len(r.nonNull()) == 1 }

// nonNull is r's types without null.
func (r *rule) nonNull() []string {
	return slices.DeleteFunc(slices.Clone(r.types), func(t string) bool { return t == tNull })
}

// firstType is r's first type other than null, or "" when it names none.
func (r *rule) firstType() string {
	if t := r.nonNull(); len(t) > 0 {
		return t[0]
	}
	return ""
}

// quoteArgs joins args with spaces, single-quoting each one that is empty
// or holds anything but letters, digits and -_./:=@%+, ; a single quote
// inside one is written '"'"'. ParseSlash's word splitting reads it back.
func quoteArgs(args []string) string {
	var b strings.Builder
	for i, a := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		if a != "" && strings.IndexFunc(a, unsafe) < 0 {
			b.WriteString(a)
			continue
		}
		b.WriteString("'" + strings.ReplaceAll(a, "'", `'"'"'`) + "'")
	}
	return b.String()
}

// unsafe reports whether c needs quoting in a word.
func unsafe(c rune) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return false
	}
	return !strings.ContainsRune("-_./:=@%+,", c)
}

// Param is one top-level argument of a command, as a command line sees
// it: a flag, a positional value, and their help.
type Param struct {
	Name        string // the JSON property name, and the long flag's
	Type        string // the JSON Schema type, null aside: "string", "integer", …
	Items       string // an array's item type
	Description string
	Short       string // the one-dash flag name, from x-cli short
	Placeholder string // the value's name in help, from x-cli placeholder
	Group       string // the help section, from x-cli group

	Required   bool
	Positional bool // filled from positional values (x-cli arg)
	Rest       bool // a positional array: it takes every positional value left
	Hidden     bool // left out of help and completion (x-cli hidden)
	Secret     bool // writeOnly: masked in the audit
	HasDefault bool

	Default  any   // the default, when HasDefault
	Enum     []any // the allowed values; an array's are its items'
	Min, Max *float64
}

// Params is c's top-level arguments, in the schema's order. A command with
// no schema has none; a schema that does not compile is an error, since a
// Command not yet registered may carry one.
func Params(c Command) ([]Param, error) {
	if len(c.Args) == 0 {
		return nil, nil
	}
	r, err := compileRule(c.Args)
	if err != nil {
		return nil, fmt.Errorf("command: %s: args schema: %w", c.ID, err)
	}
	out := make([]Param, 0, len(r.props))
	for _, p := range r.props {
		x := p.r
		q := Param{
			Name: p.name, Type: x.firstType(), Description: x.help,
			Short: x.short, Placeholder: x.placeholder, Group: x.group,
			Required: slices.Contains(r.required, p.name), Positional: x.arg,
			Rest: x.arg && x.is(tArray), Hidden: x.hidden, Secret: x.secret,
			HasDefault: x.hasDef, Default: x.def, Enum: slices.Clone(x.enum),
			Min: x.min, Max: x.max,
		}
		if x.items != nil {
			q.Items = x.items.firstType()
			if len(q.Enum) == 0 {
				q.Enum = slices.Clone(x.items.enum)
			}
		}
		out = append(out, q)
	}
	return out, nil
}

// Complete is the completions of partial, the word being typed, for a
// command line with words args already typed:
//   - with an empty id, the IDs of the commands offered on SurfaceCLI and
//     not hidden;
//   - after a flag that takes a value, its enum values;
//   - for a partial "--name=", the flag's enum values, or true and false
//     for a boolean, each after "--name=";
//   - for any other partial starting with "-", the flags not hidden: each
//     --name, and -s for a short;
//   - otherwise, and after "--", none.
//
// They start with partial, sorted and without duplicates. Completion reads
// the commands' definitions only: it does not check a command's When, its
// gate, or whether its arguments so far are valid.
func (r *Registry) Complete(id ID, args []string, partial string) []string {
	snap := r.snap.Load()
	var out []string
	if id == "" {
		for i := range snap.entries {
			c := &snap.entries[i].cmd
			if !c.Hidden && c.surfaces()&SurfaceCLI != 0 {
				out = append(out, string(c.ID))
			}
		}
		return completions(out, partial)
	}
	i, ok := snap.byID[id]
	if !ok || snap.entries[i].args == nil || slices.Contains(args, "--") {
		return nil
	}
	ru := snap.entries[i].args
	if n := len(args); n > 0 && strings.HasPrefix(args[n-1], "-") && !strings.Contains(args[n-1], "=") {
		if _, p := ru.flag(args[n-1]); p != nil && !p.isOnly(tBoolean) {
			return completions(values(p), partial)
		}
	}
	if !strings.HasPrefix(partial, "-") {
		return nil
	}
	if f, _, ok := strings.Cut(partial, "="); ok {
		if _, p := ru.flag(f); p != nil {
			for _, v := range values(p) {
				out = append(out, f+"="+v)
			}
		}
		return completions(out, partial)
	}
	for _, p := range ru.props {
		if p.r.hidden {
			continue
		}
		out = append(out, "--"+p.name)
		if p.r.short != "" {
			out = append(out, "-"+p.r.short)
		}
	}
	return completions(out, partial)
}

// values is what a flag's value can be: its enum, an array's items' enum,
// or true and false for a boolean.
func values(p *rule) []string {
	enum := p.enum
	if p.items != nil && len(enum) == 0 {
		enum = p.items.enum
	}
	if len(enum) == 0 && p.isOnly(tBoolean) {
		return []string{strconv.FormatBool(true), strconv.FormatBool(false)}
	}
	out := make([]string, 0, len(enum))
	for _, v := range enum {
		out = append(out, fmt.Sprint(v))
	}
	return out
}

// completions is the words of out that start with partial, sorted and
// without duplicates.
func completions(out []string, partial string) []string {
	out = slices.DeleteFunc(out, func(s string) bool { return !strings.HasPrefix(s, partial) })
	slices.Sort(out)
	return slices.Compact(out)
}
