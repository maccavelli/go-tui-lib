package command

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// An argument struct reads the same in the registry and in Kong
// (docs/decisions/0006-MADR-command-registry.md A12). New checks each
// top-level field of A that becomes a property: its requiredness by the
// json rule against Kong's, and its json name against Kong's name for it.
// Kong is not imported; its rules are these:
//
//   - a flag is required only with required:"";
//   - a positional, arg:"", is required unless it has optional:"" or a
//     default;
//   - a field is named by its name:"" tag, or else by kongName of its Go
//     name.

// kongName spells a Go field name as Kong names a flag: split into words
// where the kind of character changes (lower case, upper case, digit,
// other), an upper-case run before a lower-case word giving the word its
// last letter, then joined with dashes and lower-cased. MaxItems is
// max-items, PaneID is pane-id, HTTPServer is http-server.
func kongName(goName string) string {
	var words [][]rune
	kind := -1
	for _, r := range goName {
		k := runeKind(r)
		if k == kind {
			words[len(words)-1] = append(words[len(words)-1], r)
			continue
		}
		// An upper-case run ending where a lower-case word starts gives
		// that word its first letter: Max is one word, HTTPServer two.
		switch last := len(words) - 1; {
		case k == kindLower && kind == kindUpper && len(words[last]) == 1:
			words[last] = append(words[last], r)
		case k == kindLower && kind == kindUpper:
			run := words[last]
			words[last] = run[:len(run)-1]
			words = append(words, []rune{run[len(run)-1], r})
		default:
			words = append(words, []rune{r})
		}
		kind = k
	}
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = string(w)
	}
	return strings.ToLower(strings.Join(parts, "-"))
}

// The kinds of character kongName splits between.
const (
	kindLower = iota
	kindUpper
	kindDigit
	kindOther
)

func runeKind(r rune) int {
	switch {
	case unicode.IsLower(r):
		return kindLower
	case unicode.IsUpper(r):
		return kindUpper
	case unicode.IsDigit(r):
		return kindDigit
	}
	return kindOther
}

// agreeWithKong checks the top-level fields of the argument type t, which
// SchemaOf has accepted. A type with its own JSONSchema method names its
// properties itself, and is not checked.
func agreeWithKong(t reflect.Type) error {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	if _, own, err := ownSchema(t); own || err != nil {
		return err
	}
	return agreeFields(t)
}

// agreeFields checks t's fields as structFields reads them, an embedded
// struct's fields among its own, as both json and Kong flatten it.
func agreeFields(t reflect.Type) error {
	for f := range t.Fields() {
		tag := f.Tag.Get("json")
		if tag == "-" || (!f.IsExported() && !f.Anonymous) {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if f.Anonymous && name == "" {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				if err := agreeFields(ft); err != nil {
					return err
				}
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		if err := agreeField(t, f, name, opts); err != nil {
			return err
		}
	}
	return nil
}

// agreeField checks one field: its requiredness, then its name.
func agreeField(t reflect.Type, f reflect.StructField, name, opts string) error {
	where := fmt.Sprintf("%s.%s", t, f.Name)
	omit := strings.Contains(","+opts+",", ",omitzero,") || strings.Contains(","+opts+",", ",omitempty,")
	_, hasDefault := f.Tag.Lookup("default")
	_, arg := f.Tag.Lookup("arg")
	_, required := f.Tag.Lookup("required")
	_, optional := f.Tag.Lookup("optional")
	if required && optional {
		return fmt.Errorf("%s: required:\"\" and optional:\"\" together", where)
	}
	byJSON := !omit && !hasDefault && f.Type.Kind() != reflect.Pointer
	byKong := required
	if arg {
		byKong = !optional && !hasDefault
	}
	switch {
	case byJSON && !byKong && arg:
		return fmt.Errorf("%s: the schema requires it, and Kong does not: remove optional:\"\", or make it omitzero", where)
	case byJSON && !byKong:
		return fmt.Errorf("%s: the schema requires it, and Kong does not: add required:\"\", or make it omitzero", where)
	case !byJSON && byKong && arg:
		return fmt.Errorf("%s: Kong requires it, and the schema does not: add optional:\"\"", where)
	case !byJSON && byKong:
		return fmt.Errorf("%s: Kong requires it, and the schema does not: remove required:\"\"", where)
	}
	kong := f.Tag.Get("name")
	if kong == "" {
		kong = kongName(f.Name)
	}
	if kong != name {
		return fmt.Errorf("%s: the schema names it %q, and Kong %q: add name:%q", where, name, kong, name)
	}
	return nil
}
