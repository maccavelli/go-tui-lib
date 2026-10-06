package command

import (
	"regexp"
)

// placeholder is a $NAME in a command file's body.
var placeholder = regexp.MustCompile(`\$([A-Z][A-Z0-9_]*)`)

// argumentsName is the placeholder that takes the whole slash tail.
const argumentsName = "ARGUMENTS"

// placeholders is body's placeholders in order of first use, without
// ARGUMENTS.
func placeholders(body string) []string {
	var names []string
	seen := map[string]bool{argumentsName: true}
	for _, m := range placeholder.FindAllStringSubmatch(body, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	return names
}

// usesArguments reports whether body takes the whole slash tail.
func usesArguments(body string) bool {
	for _, m := range placeholder.FindAllStringSubmatch(body, -1) {
		if m[1] == argumentsName {
			return true
		}
	}
	return false
}

// expand substitutes body's placeholders, once each and in one pass, so a
// value that holds $NAME stays as it is: $ARGUMENTS with raw, and every
// other with args. Nothing else is read: $lower, $$, $(…) and !{…} stay
// as text, and nothing is run.
func expand(body string, args map[string]string, raw string) string {
	return placeholder.ReplaceAllStringFunc(body, func(m string) string {
		name := m[1:]
		if name == argumentsName {
			return raw
		}
		if v, ok := args[name]; ok {
			return v
		}
		return m
	})
}
