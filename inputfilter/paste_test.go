package inputfilter_test

import (
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// TestNormalizePaste: CR LF and a lone CR are LF; text with no CR is as it
// was.
func TestNormalizePaste(t *testing.T) {
	for in, want := range map[string]string{
		"a\r\nb\rc":   "a\nb\nc",
		"\r\r\n":      "\n\n",
		"plain\ntext": "plain\ntext",
		"":            "",
	} {
		if got := inputfilter.NormalizePaste(in); got != want {
			t.Errorf("NormalizePaste(%q) = %q; want %q", in, got, want)
		}
	}
}

// TestPastedPath: codex's cases (codex tui/src/clipboard_paste.rs:399-560),
// and prose, two paths and a URL that is not a file refused.
func TestPastedPath(t *testing.T) {
	for _, c := range []struct {
		in, goos, want string
		ok             bool
	}{
		{"file:///tmp/example.png", "linux", "/tmp/example.png", true},
		{"file:///tmp/my%20file.png", "darwin", "/tmp/my file.png", true},
		{"file:///C:/Temp/x.png", "windows", `C:\Temp\x.png`, true},
		{"file://remote/tmp/x.png", "linux", "", false},
		{`C:\Temp\example.png`, "linux", `C:\Temp\example.png`, true},
		{`/home/user/My\ File.png`, "linux", "/home/user/My File.png", true},
		{`"/home/user/My File.png"`, "linux", "/home/user/My File.png", true},
		{`'/home/user/My File.png'`, "darwin", "/home/user/My File.png", true},
		{`/home/user/a\ b.png /home/user/c.png`, "linux", "", false},
		{`'C:\Users\Alice\My File.jpeg'`, "windows", `C:\Users\Alice\My File.jpeg`, true},
		{`"C:\Users\Alice\My File.jpeg"`, "linux", `C:\Users\Alice\My File.jpeg`, true},
		{`C:\Users\Alice\My Pictures\example image.png`, "windows", `C:\Users\Alice\My Pictures\example image.png`, true},
		{`\\server\share\folder\file.jpg`, "linux", `\\server\share\folder\file.jpg`, true},
		{"~/notes.txt", "darwin", "~/notes.txt", true},
		{"  /tmp/x.png\n", "linux", "/tmp/x.png", true},
		{`/tmp/a\b`, "windows", `/tmp/a\b`, true},
		{"hello world", "linux", "", false},
		{"hello", "linux", "", false},
		{"https://example.com/a.png", "linux", "", false},
		{`/tmp/trailing\`, "linux", "", false},
	} {
		got, ok := inputfilter.PastedPath(c.in, c.goos)
		if got != c.want || ok != c.ok {
			t.Errorf("PastedPath(%q, %s) = %q, %v; want %q, %v", c.in, c.goos, got, ok, c.want, c.ok)
		}
	}
}

// TestTypeahead: grok's typing kept, text, Backspace, a Shift or Alt
// Enter, and pastes; a bare Enter, Ctrl keys, arrows and the terminal's
// reports dropped; everything from the first Esc cut
// (grok xai-grok-pager/src/app/event_loop.rs:53-124).
func TestTypeahead(t *testing.T) {
	ins := []termkey.Input{
		char("h"), char("i"),
		key(termkey.KeyBackspace, 0),
		key(termkey.KeyEnter, termkey.ModShift),
		key(termkey.KeyEnter, termkey.ModAlt),
		key(termkey.KeyEnter, 0),
		{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'c', Text: "c", Mod: termkey.ModCtrl}},
		key(termkey.KeyUp, 0),
		{Kind: termkey.InputUnknown, Raw: []byte("\x1b[<0;1;1M")},
		{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'x', Text: "x", Release: true}},
		paste("p"),
		char("A"),
		key(termkey.KeyEscape, 0),
		char("z"), paste("q"),
	}
	if got := show(inputfilter.Typeahead(ins)); got != "h i backspace shift+enter alt+enter [p] A" {
		t.Fatalf("Typeahead gave %q", got)
	}
}
