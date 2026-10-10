package termcap

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"image/color"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// ReportOption configures Report.
type ReportOption func(*reportConfig)

type reportConfig struct {
	width int
	json  bool
}

// DefaultReportWidth is the width, in cells, Report wraps at.
const DefaultReportWidth = 80

// WithReportWidth wraps the report at w cells. Zero or less does not wrap.
func WithReportWidth(w int) ReportOption { return func(r *reportConfig) { r.width = w } }

// WithReportJSON writes the report as JSON instead: the schema version,
// Caps, and the findings. The width does not apply.
func WithReportJSON() ReportOption { return func(r *reportConfig) { r.json = true } }

// ReportSchemaVersion is the JSON report's schema_version. It changes only
// when a field is removed or retyped; a field added leaves it.
const ReportSchemaVersion = 1

// reportJSON is the JSON report.
type reportJSON struct {
	SchemaVersion int       `json:"schema_version"`
	Caps          Caps      `json:"caps"`
	Findings      []Finding `json:"findings"`
}

// nameWidth is the column the values start in.
const nameWidth = 22

// notQueried says why a fact no query or variable set may have stayed so.
var notQueried = map[string]string{
	"sync_output":    "tea asks only outside SSH and Apple Terminal, and not every terminal answers",
	"grapheme_width": "tea asks only outside SSH and Apple Terminal, and not every terminal answers",
	"desktop_notify": "asked only behind the heuristic",
	"kitty_graphics": "asked only behind the heuristic",
}

// Report writes c to w as one line per fact, in Caps's field order: the
// fact's name, as JSON names it; its value; and where it came from, with
// its reason token, or the reason a fact was not queried. The findings
// follow, one per token. The text is ASCII, with no colour, so it can be
// pasted into an issue. Lines longer than the width wrap under the value
// column. WithReportJSON writes the same as JSON.
func Report(w io.Writer, c Caps, o ...ReportOption) error {
	cfg := reportConfig{width: DefaultReportWidth}
	for _, f := range o {
		f(&cfg)
	}
	findings := Findings(c)
	if cfg.json {
		if findings == nil {
			findings = []Finding{}
		}
		out, err := json.Marshal(reportJSON{SchemaVersion: ReportSchemaVersion, Caps: c, Findings: findings}, jsontext.WithIndent("  "))
		if err != nil {
			return err
		}
		_, err = w.Write(append(out, '\n'))
		return err
	}
	var b strings.Builder
	v := reflect.ValueOf(c)
	t := v.Type()
	for i := range t.NumField() {
		name := fieldName(t.Field(i))
		b.WriteString(line(name, describe(name, v.Field(i), c.TimedOut), cfg.width))
	}
	if len(findings) > 0 {
		b.WriteString("\nfindings\n")
		for _, f := range findings {
			b.WriteString(line(f.Disposition.String(), printable(f.ID+": "+f.Message+" Fix: "+f.Fix), cfg.width))
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// fieldName is a field's JSON name, or its own name in snake case when JSON
// writes it by hand.
func fieldName(f reflect.StructField) string {
	if tag, _, _ := strings.Cut(f.Tag.Get("json"), ","); tag != "" && tag != "-" {
		return tag
	}
	var b strings.Builder
	for i, r := range f.Name {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// describe is a field's value, and for a Fact, its origin and its reason:
// the fact's reason token when it has one, else, for a fact never
// queried, what may have kept it so.
func describe(name string, v reflect.Value, timedOut bool) string {
	if v.Kind() != reflect.Struct || v.NumField() != 3 {
		return value(v)
	}
	origin, ok := v.Field(1).Interface().(Origin)
	if !ok {
		return value(v)
	}
	why, val := origin.String(), value(v.Field(0))
	if origin == NotQueried && v.Field(0).IsZero() {
		val = Unknown.String() // not "no" or "none", which would be facts
	}
	if token := v.Field(2).String(); token != "" {
		why += ": " + printable(token)
	} else if origin == NotQueried {
		reason := notQueried[name]
		switch {
		case timedOut && reason != "":
			reason += "; or no reply before the timeout"
		case timedOut:
			reason = "no reply before the timeout"
		}
		if reason != "" {
			why += ": " + reason
		}
	}
	return val + " (" + why + ")"
}

var colorType = reflect.TypeFor[color.Color]()

// value is one value as ASCII text; "-" when it is empty.
func value(v reflect.Value) string {
	if v.Type() == colorType {
		c, ok := v.Interface().(color.Color)
		if !ok { // a nil colour
			return "-"
		}
		r, g, b, _ := c.RGBA()
		return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	}
	if s, ok := v.Interface().(fmt.Stringer); ok {
		return printable(s.String())
	}
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return "yes"
		}
		return "no"
	case reflect.String:
		return printable(v.String())
	case reflect.Int:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Slice, reflect.Array:
		if v.Len() == 0 || v.IsZero() {
			return "-"
		}
		parts := make([]string, v.Len())
		for i := range parts {
			parts[i] = value(v.Index(i))
		}
		return strings.Join(parts, ";")
	}
	return printable(fmt.Sprint(v.Interface()))
}

// printable keeps printable ASCII, and writes anything else as '?': a
// terminal's own reply, such as its version, must not put a control
// sequence into a report.
func printable(s string) string {
	if s == "" {
		return "-"
	}
	b := []byte(s)
	for i, c := range b {
		if c < 0x20 || c > 0x7e {
			b[i] = '?'
		}
	}
	return string(b)
}

// wrapWords breaks s into lines of at most limit cells, only after a
// space, ';' or ',', so a reason token, which holds hyphens, and a colour
// stay whole; ansi's wrappers break at every hyphen (PLAN D12). A word
// longer than the line is cut.
func wrapWords(s string, limit int) []string {
	var lines []string
	var line strings.Builder
	width := 0
	flush := func() {
		lines = append(lines, strings.TrimRight(line.String(), " "))
		line.Reset()
		width = 0
	}
	for _, w := range words(s) {
		if width == 0 {
			w = strings.TrimLeft(w, " ")
		}
		if width > 0 && width+ansi.StringWidth(strings.TrimRight(w, " ")) > limit {
			flush()
			w = strings.TrimLeft(w, " ")
		}
		for ansi.StringWidth(w) > limit && ansi.StringWidth(strings.TrimRight(w, " ")) > limit {
			cut := ansi.Truncate(w, limit, "")
			line.WriteString(cut)
			flush()
			w = w[len(cut):]
		}
		line.WriteString(w)
		width += ansi.StringWidth(w)
	}
	if line.Len() > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}

// words splits s after each space, ';' and ','.
func words(s string) []string {
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] == ' ' || s[i] == ';' || s[i] == ',' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// line is one fact, wrapped at width under its value column.
func line(name, text string, width int) string {
	head := fmt.Sprintf("%-*s ", nameWidth, name)
	if width <= 0 || width <= len(head)+10 {
		return head + text + "\n"
	}
	wrapped := wrapWords(text, width-len(head))
	indent := strings.Repeat(" ", len(head))
	var b strings.Builder
	for i, l := range wrapped {
		if i == 0 {
			b.WriteString(head)
		} else {
			b.WriteString(indent)
		}
		b.WriteString(strings.TrimRight(l, " "))
		b.WriteByte('\n')
	}
	return b.String()
}
