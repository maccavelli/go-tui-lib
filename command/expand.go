package command

import (
	"fmt"
	"regexp"
	"strings"
)

// maxExpansion is the most a command file's expansion writes: 1 MiB
// (docs/decisions/0014-PLAN-hardening.md Step 5).
const maxExpansion = 1 << 20

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
// as text, and nothing is run. An expansion over maxExpansion bytes stops
// there, with an *ArgError naming the limit.
func expand(body string, args map[string]string, raw string) (string, error) {
	var b strings.Builder
	last := 0
	for _, loc := range placeholder.FindAllStringIndex(body, -1) {
		name := body[loc[0]+1 : loc[1]]
		v, ok := args[name]
		switch {
		case name == argumentsName:
			v = raw
		case !ok:
			v = body[loc[0]:loc[1]]
		}
		if b.Len()+(loc[0]-last)+len(v) > maxExpansion {
			return "", over()
		}
		b.WriteString(body[last:loc[0]])
		b.WriteString(v)
		last = loc[1]
	}
	if b.Len()+len(body)-last > maxExpansion {
		return "", over()
	}
	b.WriteString(body[last:])
	return b.String(), nil
}

// over is expand's error for an expansion past maxExpansion.
func over() error {
	return &ArgError{Reason: fmt.Sprintf("the expanded prompt is over %d bytes", maxExpansion)}
}
