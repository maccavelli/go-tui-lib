package command

import (
	json "encoding/json/v2"
	"fmt"
	"io"
	"strings"

	"github.com/maccavelli/go-tui-lib/internal/enum"
)

// Format is how WriteResult writes a result: as text or as JSON. Its text
// form is a stable lowercase token, so a --format flag binds to it natively
// (docs/decisions/0014-MADR-native-integration-api.md W2).
type Format uint8

// The formats.
const (
	// FormatText is the result's text: Result.Text, or else Result.Value
	// as JSON, which is what CallMCP gives an agent.
	FormatText Format = iota
	// FormatJSON is Result.Value as JSON, or null.
	FormatJSON
)

var formatNames = enum.Names[Format]{
	Pkg: pkgName, Type: "Format",
	Tokens: []string{"text", "json"},
}

// String is the format's token: "text" or "json".
func (f Format) String() string { return formatNames.String(f) }

// MarshalText is the format's token. A format with no token is an error.
func (f Format) MarshalText() ([]byte, error) {
	return formatNames.Marshal(f)
}

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (f *Format) UnmarshalText(b []byte) error {
	return formatNames.Unmarshal(b, f)
}

// WriteResult writes res to w in format f, for a program's own command
// line:
//   - FormatText writes CallMCP's text, Result.Text or else Result.Value as
//     deterministic JSON, ending in a newline; an empty text writes
//     nothing;
//   - FormatJSON writes Result.Value as deterministic JSON, or null, and a
//     newline.
//
// A time.Duration in Result.Value is written in its string form
// ("1m30s"). Result.Cmd is not run: it is an effect for a Bubble Tea
// program, and a command line has none to run it.
func WriteResult(w io.Writer, res Result, f Format) error {
	var out string
	switch f {
	case FormatText:
		text, err := resultText(res)
		if err != nil {
			return fmt.Errorf("command: result: %w", err)
		}
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		out = text
	case FormatJSON:
		b, err := resultJSON(res.Value)
		if err != nil {
			return fmt.Errorf("command: result: %w", err)
		}
		out = string(b) + "\n"
	default:
		return fmt.Errorf("command: unknown Format %d", f)
	}
	if _, err := io.WriteString(w, out); err != nil {
		return fmt.Errorf("command: writing the result: %w", err)
	}
	return nil
}

// resultText is a result as text, for CallMCP and FormatText: Result.Text,
// or Result.Value as JSON when there is no text. A Value that does not
// encode is an error, whether or not there is text.
func resultText(res Result) (string, error) {
	if res.Value == nil {
		return res.Text, nil
	}
	b, err := resultJSON(res.Value)
	if err != nil {
		return "", err
	}
	if res.Text == "" {
		return string(b), nil
	}
	return res.Text, nil
}

// resultJSON is v as deterministic JSON, with durations in their string
// form. Its error is the encoder's own, as CallMCP reports it.
func resultJSON(v any) ([]byte, error) {
	return json.Marshal(v, json.Deterministic(true), json.WithMarshalers(durationMarshalers))
}
