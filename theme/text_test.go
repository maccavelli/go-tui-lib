package theme

import (
	"flag"
	"io"
	"testing"
)

// The enums' text forms
// (docs/decisions/0014-PLAN-component-native-forms.md Step 10).

func TestBackgroundText(t *testing.T) {
	for b, tok := range map[Background]string{Unknown: "unknown", Dark: "dark", Light: "light"} {
		got, err := b.MarshalText()
		var back Background = 9
		if b.String() != tok || string(got) != tok || err != nil || back.UnmarshalText(got) != nil || back != b {
			t.Errorf("%d: %q, %q, %v, back %d", b, b.String(), got, err, back)
		}
	}
	b := Dark
	if err := b.UnmarshalText([]byte("auto")); err != nil || b != Unknown {
		t.Errorf(`"auto" = %d, %v; want Unknown`, b, err)
	}
	for _, in := range []string{"Dark", "LIGHT", "", "1", "Auto"} {
		b = Light
		if err := b.UnmarshalText([]byte(in)); err == nil || b != Light {
			t.Errorf("%q: %d, %v; want an error and Light kept", in, b, err)
		}
	}
	for _, v := range []Background{3, -1} {
		if _, err := v.MarshalText(); err == nil {
			t.Errorf("Background(%d) has a token", v)
		}
	}
	if Background(-1).String() != "Background(-1)" {
		t.Errorf("Background(-1) = %q", Background(-1).String())
	}
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.TextVar(&b, "background", Unknown, "dark, light or auto")
	if err := fs.Parse([]string{"--background=light"}); err != nil || b != Light {
		t.Errorf("--background=light: %d, %v", b, err)
	}
}

func TestBorderStyleText(t *testing.T) {
	for s, tok := range map[BorderStyle]string{
		BorderLight: "light", BorderRounded: "rounded", BorderHeavy: "heavy", BorderDouble: "double",
	} {
		got, err := s.MarshalText()
		var back BorderStyle = 9
		if s.String() != tok || string(got) != tok || err != nil || back.UnmarshalText(got) != nil || back != s {
			t.Errorf("%d: %q, %q, %v, back %d", s, s.String(), got, err, back)
		}
	}
	s := BorderHeavy
	for _, in := range []string{"Rounded", "auto", "", "2"} {
		if err := s.UnmarshalText([]byte(in)); err == nil || s != BorderHeavy {
			t.Errorf("%q: %d, %v; want an error and heavy kept", in, s, err)
		}
	}
	if _, err := BorderStyle(4).MarshalText(); err == nil || err.Error() != "theme: BorderStyle 4 has no name" {
		t.Errorf("BorderStyle(4): %v", err)
	}
}
