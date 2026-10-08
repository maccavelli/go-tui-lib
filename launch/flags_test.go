package launch

import (
	"encoding"
	"flag"
	"io"
	"testing"
)

// The interfaces each framework asks of a flag value
// (docs/reports/0014-REPORT-api-assessment-and-integration-research.md §3.2).
var (
	_ flag.Value               = new(Choice)
	_ flag.Getter              = new(Choice)
	_ encoding.TextMarshaler   = ChoiceAuto
	_ encoding.TextUnmarshaler = new(Choice)
	_ interface {
		flag.Value
		Type() string // pflag
	} = new(Choice)
	_ interface {
		MarshalFlag() (string, error)
		UnmarshalFlag(string) error
	} = new(Choice) // go-flags
)

func TestChoiceText(t *testing.T) {
	for _, c := range []struct {
		choice Choice
		text   string
	}{{ChoiceAuto, "auto"}, {ChoiceTUI, "tui"}, {ChoicePlain, "plain"}} {
		if got := c.choice.String(); got != c.text {
			t.Errorf("String() = %q, want %q", got, c.text)
		}
		if got := c.choice.Type(); got != "mode" {
			t.Errorf("Type() = %q, want \"mode\"", got)
		}
		if got := c.choice.Get(); got != c.choice {
			t.Errorf("Get() = %v, want %v", got, c.choice)
		}
		b, err := c.choice.MarshalText()
		if err != nil || string(b) != c.text {
			t.Errorf("MarshalText() = %q, %v", b, err)
		}
		s, err := c.choice.MarshalFlag()
		if err != nil || s != c.text {
			t.Errorf("MarshalFlag() = %q, %v", s, err)
		}
		for name, read := range map[string]func(*Choice, string) error{
			"Set":           (*Choice).Set,
			"UnmarshalText": func(v *Choice, s string) error { return v.UnmarshalText([]byte(s)) },
			"UnmarshalFlag": (*Choice).UnmarshalFlag,
		} {
			v := Choice(9)
			if err := read(&v, c.text); err != nil || v != c.choice {
				t.Errorf("%s(%q) = %v, %v", name, c.text, v, err)
			}
		}
	}
	if ChoiceAuto != 0 {
		t.Error("ChoiceAuto is not the zero Choice")
	}

	// An invalid value: refused, with launch's error, and v left alone.
	for name, read := range map[string]func(*Choice, string) error{
		"Set":           (*Choice).Set,
		"UnmarshalText": func(v *Choice, s string) error { return v.UnmarshalText([]byte(s)) },
		"UnmarshalFlag": (*Choice).UnmarshalFlag,
	} {
		v := ChoicePlain
		if err := read(&v, "TUI"); err == nil || err.Error() != `launch: unknown Choice "TUI"` || v != ChoicePlain {
			t.Errorf("%s(\"TUI\") = %v, left %s", name, err, v)
		}
	}
	if _, err := Choice(9).MarshalText(); err == nil || err.Error() != "launch: Choice 9 has no name" {
		t.Errorf("Choice(9).MarshalText(): %v", err)
	}
	if _, err := Choice(9).MarshalFlag(); err == nil {
		t.Error("Choice(9).MarshalFlag() gave no error")
	}
}

func TestTargetText(t *testing.T) {
	for i, want := range []string{"none", "stream", "err", "tty"} {
		v := Target(i)
		b, err := v.MarshalText()
		if v.String() != want || err != nil || string(b) != want {
			t.Errorf("Target(%d) = %q, %q, %v; want %q", i, v.String(), b, err, want)
		}
		var back Target
		if err := back.UnmarshalText(b); err != nil || back != v {
			t.Errorf("UnmarshalText(%q) = %v, %v", b, back, err)
		}
	}
	if TargetNone != 0 {
		t.Error("TargetNone is not the zero Target")
	}
}

func TestFlagsRegister(t *testing.T) {
	for _, c := range []struct {
		args    []string
		want    Choice
		wantErr bool
	}{
		{nil, ChoiceAuto, false},
		{[]string{"-tui"}, ChoiceTUI, false},
		{[]string{"--tui"}, ChoiceTUI, false},
		{[]string{"-tui=false"}, ChoiceAuto, false},
		{[]string{"-mode=plain"}, ChoicePlain, false},
		{[]string{"-mode", "tui"}, ChoiceTUI, false},
		// --tui wins over --mode, whatever the order.
		{[]string{"-mode=plain", "-tui"}, ChoiceTUI, false},
		{[]string{"-tui", "-mode=plain"}, ChoiceTUI, false},
		{[]string{"-mode=bogus"}, ChoiceAuto, true},
	} {
		var f Flags
		fs := flag.NewFlagSet("prog", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f.RegisterFlags(fs)
		err := fs.Parse(c.args)
		if (err != nil) != c.wantErr {
			t.Errorf("%q: Parse = %v, want an error: %v", c.args, err, c.wantErr)
			continue
		}
		if got := f.Resolve(); !c.wantErr && got != c.want {
			t.Errorf("%q: Resolve() = %s, want %s", c.args, got, c.want)
		}
	}

	// The bare -tui is a boolean flag, so it takes no value.
	var f Flags
	fs := flag.NewFlagSet("prog", flag.ContinueOnError)
	f.RegisterFlags(fs)
	if bf, ok := fs.Lookup("tui").Value.(interface{ IsBoolFlag() bool }); !ok || !bf.IsBoolFlag() {
		t.Error("-tui is not a boolean flag")
	}
	if got := fs.Lookup("mode").DefValue; got != "auto" {
		t.Errorf("-mode's default is %q, want \"auto\"", got)
	}
}

// source is what a *cobra.Command has.
type source struct{ in, out, err *fake }

func (s source) InOrStdin() io.Reader   { return s.in }
func (s source) OutOrStdout() io.Writer { return s.out }
func (s source) ErrOrStderr() io.Writer { return s.err }

func TestFromSource(t *testing.T) {
	src := source{pipe(), terminal(80, 24), pipe()}
	s := FromSource(src, []string{"TERM=xterm"})
	if s.In != src.in || s.Out != src.out || s.Err != src.err {
		t.Error("FromSource did not take the source's three streams")
	}
	if got := s.Env.Getenv("TERM"); got != "xterm" {
		t.Errorf("FromSource's Env: TERM = %q", got)
	}
}
