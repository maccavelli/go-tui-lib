package palette

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// stage is the argument stage: a hit whose parameters the picker asks for,
// one at a time in the same overlay, before it runs (0008-MADR §6, A3 item
// 5).
type stage struct {
	hit    Hit
	params []command.Param // those asked for, in the schema's order
	at     int             // the one being asked

	names []string // the values given, in order
	vals  []any

	choices []choice       // the asked one's values, when it has a list
	shown   []row          // the choices the input matches, best first
	matcher *fuzzy.Matcher // the input's, while it is not empty
	reason  string         // why the last value was refused, until the input changes
}

// choice is one value of a list: as it is drawn, and as it is sent.
type choice struct {
	text string
	v    any
}

// asked is the parameters the stage asks for: all but the Hidden ones. It
// is false when one of those is not a string, integer, number or boolean
// and has no Enum, so the hit falls through to SelectedMsg instead.
func asked(ps []command.Param) ([]command.Param, bool) {
	var out []command.Param
	for _, p := range ps {
		if p.Hidden {
			continue
		}
		switch {
		case p.Type == "array" || p.Type == "object":
			return nil, false
		case len(p.Enum) > 0, p.Type == "string", p.Type == "integer", p.Type == "number", p.Type == "boolean":
			out = append(out, p)
		default:
			return nil, false
		}
	}
	return out, true
}

// param is the parameter being asked for.
func (st *stage) param() command.Param { return st.params[st.at] }

// begin asks for the parameter at st.at: its list of values, if it has one.
func (st *stage) begin() {
	p := st.param()
	st.choices, st.reason = nil, ""
	switch {
	case len(p.Enum) > 0:
		for _, v := range p.Enum {
			st.choices = append(st.choices, choice{text: sanitize.Line(valueText(v)), v: v})
		}
	case p.Type == "boolean":
		st.choices = []choice{{text: "true", v: true}, {text: "false", v: false}}
	}
	st.filter("")
}

// first is the row the selection starts on for an empty input: the
// default's, if the list has it.
func (st *stage) first() int {
	p := st.param()
	if !p.HasDefault {
		return 0
	}
	for i, c := range st.choices {
		if valueText(c.v) == valueText(p.Default) {
			return i
		}
	}
	return 0
}

// filter is the choices q matches, best first, or all of them, in order,
// for an empty q.
func (st *stage) filter(q string) {
	st.shown, st.matcher = nil, nil
	if q != "" {
		st.matcher = fuzzy.NewMatcher(q)
	}
	for i, c := range st.choices {
		sc := fuzzy.Score{Value: len(st.choices) - i}
		if st.matcher != nil {
			s, ok := st.matcher.Score(c.text)
			if !ok {
				continue
			}
			sc = s
		}
		st.shown = append(st.shown, row{hit: Hit{ID: c.text, Text: c.text, Score: sc}, src: i})
	}
	slices.SortStableFunc(st.shown, func(a, b row) int {
		switch {
		case better(a.hit.Score, b.hit.Score):
			return -1
		case better(b.hit.Score, a.hit.Score):
			return 1
		}
		return 0
	})
}

// give takes the asked parameter's value: the selected row of a list, or
// what was typed. It is false, with st.reason set, when the value is
// refused.
func (st *stage) give(typed string, sel int) bool {
	p := st.param()
	var v any
	present := true
	if st.choices != nil {
		if sel < 0 || sel >= len(st.shown) {
			st.reason = p.Name + ": pick one of its values"
			return false
		}
		v = st.choices[st.shown[sel].src].v
	} else {
		var why string
		v, present, why = value(p, typed)
		if why != "" {
			st.reason = p.Name + ": " + why
			return false
		}
	}
	if present {
		st.names, st.vals = append(st.names, p.Name), append(st.vals, v)
	}
	st.at++
	return true
}

// value is typed as p's value, whether there is one, and why it is
// refused: an empty input is p's default, or none when p is not required;
// a number is parsed and held to p's Min and Max.
func value(p command.Param, typed string) (any, bool, string) {
	if typed == "" {
		switch {
		case p.HasDefault:
			return p.Default, true, ""
		case p.Required:
			return nil, false, "required"
		}
		return nil, false, ""
	}
	switch p.Type {
	case "integer":
		i, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return nil, false, "not a whole number"
		}
		if why := bounds(p, float64(i)); why != "" {
			return nil, false, why
		}
		return i, true, ""
	case "number":
		f, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, false, "not a number"
		}
		if why := bounds(p, f); why != "" {
			return nil, false, why
		}
		return f, true, ""
	}
	return typed, true, ""
}

// bounds is why n is outside p's Min and Max, or "".
func bounds(p command.Param, n float64) string {
	switch {
	case p.Min != nil && n < *p.Min:
		return "at least " + strconv.FormatFloat(*p.Min, 'g', -1, 64)
	case p.Max != nil && n > *p.Max:
		return "at most " + strconv.FormatFloat(*p.Max, 'g', -1, 64)
	}
	return ""
}

// done reports whether every parameter has been asked for.
func (st *stage) done() bool { return st.at >= len(st.params) }

// args is the values given, as one JSON object in the order they were
// asked for.
func (st *stage) args() (jsontext.Value, error) {
	var b bytes.Buffer
	enc := jsontext.NewEncoder(&b)
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return nil, err
	}
	for i, name := range st.names {
		if err := enc.WriteToken(jsontext.String(name)); err != nil {
			return nil, err
		}
		if err := json.MarshalEncode(enc, st.vals[i]); err != nil {
			return nil, fmt.Errorf("argument %s: %w", name, err)
		}
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return nil, err
	}
	return jsontext.Value(bytes.TrimSpace(b.Bytes())), nil
}

// placeholder is the input's text while the parameter's value is empty:
// its placeholder or name, and its default unless it is secret.
func placeholder(p command.Param) string {
	s := p.Placeholder
	if s == "" {
		s = p.Name
	}
	if p.HasDefault && !p.Secret {
		s += " (" + valueText(p.Default) + ")"
	}
	return sanitize.Line(s)
}

// valueText is v as it is drawn: a string as itself, anything else as its
// JSON.
func valueText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
