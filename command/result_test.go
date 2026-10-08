package command

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Format and WriteResult
// (docs/decisions/0014-PLAN-component-native-forms.md Step 4).

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestWriteResult(t *testing.T) {
	value := map[string]any{"b": 2, "a": time.Minute}
	for _, tc := range []struct {
		name       string
		res        Result
		text, json string
	}{
		{"empty", Result{}, "", "null\n"},
		{"text only", Result{Text: "done"}, "done\n", "null\n"},
		{"text with its newline", Result{Text: "two\nlines\n"}, "two\nlines\n", "null\n"},
		{"value only", Result{Value: value}, `{"a":"1m0s","b":2}` + "\n", `{"a":"1m0s","b":2}` + "\n"},
		{"both", Result{Text: "done", Value: value}, "done\n", `{"a":"1m0s","b":2}` + "\n"},
		{"a duration", Result{Value: 90 * time.Second}, `"1m30s"` + "\n", `"1m30s"` + "\n"},
		{"a Cmd is not run", Result{Text: "x", Cmd: func() tea.Msg { panic("ran") }}, "x\n", "null\n"},
	} {
		for f, want := range map[Format]string{FormatText: tc.text, FormatJSON: tc.json} {
			var b strings.Builder
			if err := WriteResult(&b, tc.res, f); err != nil || b.String() != want {
				t.Errorf("%s, %s: wrote %q, %v; want %q", tc.name, f, b.String(), err, want)
			}
		}
	}
	for _, f := range []Format{FormatText, FormatJSON} {
		err := WriteResult(failingWriter{}, Result{Text: "x", Value: 1}, f)
		if err == nil || !strings.Contains(err.Error(), "disk full") {
			t.Errorf("%s to a failing writer: %v", f, err)
		}
		err = WriteResult(io.Discard, Result{Text: "x", Value: func() {}}, f)
		if err == nil || !strings.HasPrefix(err.Error(), "command: result: ") {
			t.Errorf("%s with a Value that does not encode: %v", f, err)
		}
	}
	if err := WriteResult(io.Discard, Result{}, Format(9)); err == nil {
		t.Error("an unknown Format wrote")
	}
}

func TestFormatText(t *testing.T) {
	for f, tok := range map[Format]string{FormatText: "text", FormatJSON: "json"} {
		b, err := f.MarshalText()
		var back Format
		if f.String() != tok || string(b) != tok || err != nil || back.UnmarshalText(b) != nil || back != f {
			t.Errorf("%d: %q, %q, %v, back %d", f, f.String(), b, err, back)
		}
	}
	var f Format
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.TextVar(&f, "format", FormatText, "text or json")
	if err := fs.Parse([]string{"--format=json"}); err != nil || f != FormatJSON {
		t.Errorf("--format=json: %v, %s", err, f)
	}
	if err := fs.Parse([]string{"--format=JSON"}); err == nil {
		t.Error("--format=JSON was accepted; tokens are exact")
	}
	if _, err := Format(9).MarshalText(); err == nil {
		t.Error("Format(9) has a token")
	}
}

// TestCallMCPUnchanged: CallMCP's text is WriteResult's FormatText, less
// its newline, for every call TestExportGolden's goldens hold; those
// goldens are unchanged by the move into resultText.
func TestCallMCPUnchanged(t *testing.T) {
	for _, res := range []Result{
		{Text: "zoomed logs"},
		{Value: map[string]any{"lines": 3}},
		{Text: "t", Value: []int{1, 2}},
		{Value: 90 * time.Second},
	} {
		r := registryOf(t, nil, cmd("c", ReadOnly, func(c *Command) {
			c.Handler = HandlerFunc(func(_ context.Context, _ *Invocation) (Result, error) { return res, nil })
		}))
		mcp := r.CallMCP(t.Context(), "c", nil, "agent")
		var b strings.Builder
		if err := WriteResult(&b, res, FormatText); err != nil || mcp.IsError || len(mcp.Content) != 1 ||
			mcp.Content[0].Text+"\n" != b.String() {
			t.Errorf("%+v: CallMCP %+v, WriteResult %q, %v", res, mcp, b.String(), err)
		}
	}
}
