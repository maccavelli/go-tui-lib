package command

import (
	json "encoding/json/v2"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// ParseSlash reads a slash line, "/resize sidebar 4", into the request
// the palette, the program's own command line (ParseArgs) and an agent
// would send for it: Origin is OriginSlash, Raw the text after the name,
// and Args the JSON.
//
// Positional values fill the properties marked positional (A1's arg tag),
// in order; a positional array takes every value left. name=value fills
// any property, and repeats for an array. Single or double quotes group
// words, and a backslash in double quotes takes the next character; a
// quoted name=value is a positional value. Each value is read as its
// property's type. A command whose only argument is a string takes the
// whole tail as it. The arguments are checked against the schema, so an
// accepted line runs unless the command's own checks refuse it. A tail, or
// arguments built from it, over the registry's limit (WithMaxArgBytes) is
// an *ArgError, and the tail's size is checked before it is read.
func (r *Registry) ParseSlash(line string) (Request, error) {
	s := strings.TrimPrefix(strings.TrimLeftFunc(line, unicode.IsSpace), "/")
	name, tail := s, ""
	if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
		name, tail = s[:i], strings.TrimLeftFunc(s[i:], unicode.IsSpace)
	}
	snap := r.snap.Load()
	i, ok := snap.bySlash[name]
	if !ok {
		return Request{}, fmt.Errorf("%w: no slash command /%s", ErrUnknown, name)
	}
	e := &snap.entries[i]
	if err := r.tooLarge(len(tail)); err != nil {
		return Request{}, err
	}
	req := Request{ID: e.cmd.ID, Raw: tail, Origin: OriginSlash}
	if e.args == nil {
		return req, nil
	}
	obj, err := slashObject(e.args, tail)
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
	if err := r.tooLarge(len(req.Args)); err != nil {
		return Request{}, err
	}
	return req, nil
}

// slashObject builds the arguments object from a slash tail.
func slashObject(r *rule, tail string) (map[string]any, error) {
	obj := map[string]any{}
	if len(r.props) == 1 && len(r.props[0].r.types) == 1 && r.props[0].r.is(tString) {
		if t := strings.TrimRightFunc(tail, unicode.IsSpace); t != "" {
			obj[r.props[0].name] = t
		}
		return obj, nil
	}
	words, err := splitWords(tail)
	if err != nil {
		return nil, err
	}
	var positional []prule
	for _, p := range r.props {
		if p.r.arg {
			positional = append(positional, p)
		}
	}
	next := 0
	for _, w := range words {
		if k, v, ok := strings.Cut(w.text, "="); ok && !w.quoted {
			p := r.prop(k)
			switch {
			case p != nil:
				if err := assign(obj, k, p, v); err != nil {
					return nil, err
				}
				continue
			case !r.rest:
				return nil, &ArgError{Path: "/" + pointerToken(k), Reason: "is not a known argument"}
			}
		}
		// A positional already given by name is skipped.
		for next < len(positional) && !positional[next].r.is(tArray) && obj[positional[next].name] != nil {
			next++
		}
		if next >= len(positional) {
			if r.rest {
				continue // the words stay in Raw, for $ARGUMENTS (A6)
			}
			return nil, &ArgError{Reason: fmt.Sprintf("%q is one positional value too many", w.text)}
		}
		p := positional[next]
		if err := assign(obj, p.name, p.r, w.text); err != nil {
			return nil, err
		}
		if !p.r.is(tArray) {
			next++
		}
	}
	return obj, nil
}

// assign sets obj[name] from text, read as p's type; an array appends.
func assign(obj map[string]any, name string, p *rule, text string) error {
	path := "/" + pointerToken(name)
	if p.is(tArray) {
		var v any = text
		if p.items != nil {
			var err error
			if v, err = typed(p.items, text, path); err != nil {
				return err
			}
		}
		var list []any
		if l, ok := obj[name].([]any); ok {
			list = l
		}
		obj[name] = append(list, v)
		return nil
	}
	if _, ok := obj[name]; ok {
		return &ArgError{Path: path, Reason: "is given twice"}
	}
	v, err := typed(p, text, path)
	if err != nil {
		return err
	}
	obj[name] = v
	return nil
}

// typed reads text as a value of p's type: a number, a boolean, a string,
// or JSON for anything else.
func typed(p *rule, text, path string) (any, error) {
	switch {
	case p.is(tInteger) || p.is(tNumber):
		f, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, &ArgError{Path: path, Reason: fmt.Sprintf("%q is not a number", text)}
		}
		return f, nil
	case p.is(tBoolean):
		b, err := strconv.ParseBool(text)
		if err != nil {
			return nil, &ArgError{Path: path, Reason: fmt.Sprintf("%q is not true or false", text)}
		}
		return b, nil
	case p.is(tString) || len(p.types) == 0:
		return text, nil
	}
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return nil, &ArgError{Path: path, Reason: fmt.Sprintf("%q is not JSON: %s", text, reasonOf(err))}
	}
	return v, nil
}

// word is one word of a slash tail; quoted when any part of it was.
type word struct {
	text   string
	quoted bool
}

// splitWords splits s at spaces outside quotes.
func splitWords(s string) ([]word, error) {
	var (
		out     []word
		b       strings.Builder
		cur     word
		inWord  bool
		quote   rune
		escaped bool
	)
	for _, c := range s {
		switch {
		case escaped:
			b.WriteRune(c)
			escaped = false
		case quote != 0 && c == quote:
			quote = 0
		case quote == '"' && c == '\\':
			escaped = true
		case quote != 0:
			b.WriteRune(c)
		case c == '"' || c == '\'':
			quote, inWord, cur.quoted = c, true, true
		case unicode.IsSpace(c):
			if inWord {
				cur.text = b.String()
				out = append(out, cur)
				b.Reset()
				cur, inWord = word{}, false
			}
		default:
			b.WriteRune(c)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, &ArgError{Reason: "a quote is not closed"}
	}
	if inWord {
		cur.text = b.String()
		out = append(out, cur)
	}
	return out, nil
}
