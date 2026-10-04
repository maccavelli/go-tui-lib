package termcap

import (
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

type reportConfig struct{ width int }

// DefaultReportWidth is the width, in cells, Report wraps at.
const DefaultReportWidth = 80

// WithReportWidth wraps the report at w cells. Zero or less does not wrap.
func WithReportWidth(w int) ReportOption { return func(r *reportConfig) { r.width = w } }

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
// the reason a fact was not queried. The text is ASCII, with no colour, so
// it can be pasted into an issue. Lines longer than the width wrap under
// the value column.
func Report(w io.Writer, c Caps, o ...ReportOption) error {
	cfg := reportConfig{width: DefaultReportWidth}
	for _, f := range o {
		f(&cfg)
	}
	var b strings.Builder
	v := reflect.ValueOf(c)
	t := v.Type()
	for i := range t.NumField() {
		name := fieldName(t.Field(i))
		b.WriteString(line(name, describe(name, v.Field(i), c.TimedOut), cfg.width))
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

// line is one fact, wrapped at width under its value column.
func line(name, text string, width int) string {
	head := fmt.Sprintf("%-*s ", nameWidth, name)
	if width <= 0 || width <= len(head)+10 {
		return head + text + "\n"
	}
	wrapped := strings.Split(ansi.Wrap(text, width-len(head), ";,"), "\n")
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
