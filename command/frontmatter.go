package command

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/maccavelli/go-tui-lib/when"
)

// The front matter keys that are read in more than one place.
const (
	keySlash  = "slash"
	keyHidden = "hidden"
)

// frontKeys are the front matter's keys besides arg.<NAME>
// (docs/decisions/0006-MADR-command-registry.md §6).
var frontKeys = []string{"title", "description", keySlash, "aliases", "category", "danger", "when", keyHidden}

// argKey is a front matter key that describes a placeholder.
var argKey = regexp.MustCompile(`^arg\.([A-Z][A-Z0-9_]*)$`)

// frontField is one key: value line.
type frontField struct {
	key, value string
	line       int
}

// frontMatter is a command file's front matter: a strict subset of YAML,
// one key: value per line, from a fixed set of keys.
type frontMatter struct {
	present bool // the file opens with ---
	fields  []frontField
}

func (f frontMatter) get(key string) (string, int, bool) {
	for _, x := range f.fields {
		if x.key == key {
			return x.value, x.line, true
		}
	}
	return "", 0, false
}

// lineError is a front matter error at a line, counted from 1.
type lineError struct {
	line int
	msg  string
}

func (e *lineError) Error() string { return fmt.Sprintf("line %d: %s", e.line, e.msg) }

func errAt(line int, format string, a ...any) *lineError {
	return &lineError{line, fmt.Sprintf(format, a...)}
}

// parseFrontMatter splits src into its front matter and body, and checks
// every key and value. A file that does not open with a --- line has none.
func parseFrontMatter(src string) (frontMatter, string, error) {
	lines := strings.Split(src, "\n")
	if strings.TrimSuffix(lines[0], "\r") != "---" {
		return frontMatter{}, src, nil
	}
	f := frontMatter{present: true}
	for i := 1; i < len(lines); i++ {
		line, n := strings.TrimSuffix(lines[i], "\r"), i+1
		if line == "---" {
			body := strings.Join(lines[i+1:], "\n")
			return f, body, f.check(body)
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		x, err := field(line, n)
		if err != nil {
			return frontMatter{}, "", err
		}
		if _, prev, dup := f.get(x.key); dup {
			return frontMatter{}, "", errAt(n, "%s is given again; line %d gave it first", x.key, prev)
		}
		f.fields = append(f.fields, x)
	}
	return frontMatter{}, "", errAt(1, "the front matter has no closing ---")
}

// field reads one key: value line.
func field(line string, n int) (frontField, error) {
	if line[0] == ' ' || line[0] == '\t' {
		return frontField{}, errAt(n, "a nested value; the front matter takes one key: value per line")
	}
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return frontField{}, errAt(n, "not a key: value line")
	}
	key = strings.TrimRight(key, " \t")
	if !slices.Contains(frontKeys, key) && !argKey.MatchString(key) {
		return frontField{}, errAt(n, "unknown key %q; the keys are %s and arg.<NAME>", key, strings.Join(frontKeys, ", "))
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value[:1], "[{|>") {
		return frontField{}, errAt(n, "%s has an empty or nested value; the front matter takes one key: value per line", key)
	}
	if value = unquote(value); value == "" {
		return frontField{}, errAt(n, "%s has an empty value", key)
	}
	return frontField{key, value, n}, nil
}

// unquote strips one pair of matching quotes, with no escapes.
func unquote(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}

// check checks each value: the danger, hidden, slash names, the when
// expression, and that each arg.<NAME> names a placeholder of body.
func (f frontMatter) check(body string) error {
	names := placeholders(body)
	for _, x := range f.fields {
		var err error
		switch x.key {
		case "danger":
			_, err = parseDanger(x.value)
		case keyHidden:
			if x.value != "true" && x.value != "false" {
				err = fmt.Errorf("hidden is %q, not true or false", x.value)
			}
		case keySlash:
			err = validSlash("", x.value)
		case "aliases":
			for _, a := range splitList(x.value) {
				if err = validSlash("", a); err != nil {
					break
				}
			}
		case "when":
			_, err = when.Parse(x.value)
		default:
			if m := argKey.FindStringSubmatch(x.key); len(m) == 2 && !slices.Contains(names, m[1]) {
				err = fmt.Errorf("%s describes $%s, which the body does not use", x.key, m[1])
			}
		}
		if err != nil {
			return errAt(x.line, "%s", err)
		}
	}
	return nil
}

// parseDanger reads a danger by its String form.
func parseDanger(s string) (Danger, error) {
	for d := ReadOnly; d <= Destructive; d++ {
		if d.String() == s {
			return d, nil
		}
	}
	return 0, fmt.Errorf("danger is %q, not read-only, ui, mutating or destructive", s)
}

// splitList splits a comma-separated value, trimming each item.
func splitList(v string) []string {
	var out []string
	for item := range strings.SplitSeq(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// String is f as front matter that parses back to f: --- lines around
// key: value lines, a value that would read as nested or quoted, or that
// has space at an end, put in double quotes. It is "" when f is not
// present.
func (f frontMatter) String() string {
	if !f.present {
		return ""
	}
	var b strings.Builder
	b.WriteString("---\n")
	for _, x := range f.fields {
		v := x.value
		if strings.ContainsAny(v[:1], "[{|>\"'") || strings.TrimSpace(v) != v {
			v = `"` + v + `"`
		}
		b.WriteString(x.key + ": " + v + "\n")
	}
	b.WriteString("---\n")
	return b.String()
}
