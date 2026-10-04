package termcap

import (
	"encoding/json"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
)

func TestOriginOrder(t *testing.T) {
	want := []Origin{NotQueried, Heuristic, Environment, Query, Override}
	for i := 1; i < len(want); i++ {
		if want[i-1] >= want[i] {
			t.Fatalf("%v is not weaker than %v", want[i-1], want[i])
		}
	}
}

// TestSetOverEveryPairOfOrigins checks the merge rule: a fact is replaced
// only by an equal or stronger origin.
func TestSetOverEveryPairOfOrigins(t *testing.T) {
	for held := NotQueried; held <= Override; held++ {
		for next := NotQueried; next <= Override; next++ {
			f := Fact[string]{Value: "held", Origin: held}
			ok := f.Set("next", next)
			want := next >= held
			if ok != want {
				t.Errorf("Set from %v over %v reported %v, want %v", next, held, ok, want)
			}
			if wantV := map[bool]string{true: "next", false: "held"}[want]; f.Value != wantV {
				t.Errorf("Set from %v over %v left %q, want %q", next, held, f.Value, wantV)
			}
			if wantO := max(held, next); want && f.Origin != wantO || !want && f.Origin != held {
				t.Errorf("Set from %v over %v left origin %v", next, held, f.Origin)
			}
		}
	}
}

func TestOverrideIsNeverReplaced(t *testing.T) {
	f := Fact[Support]{}
	f.Set(Unsupported, Override)
	for o := range Override {
		if f.Set(Supported, o) || f.Value != Unsupported || f.Origin != Override {
			t.Fatalf("an override was replaced from %v: %+v", o, f)
		}
	}
}

func TestEnumNames(t *testing.T) {
	check := func(name string, got []string, want string) {
		t.Helper()
		if s := strings.Join(got, ","); s != want {
			t.Errorf("%s names: %s, want %s", name, s, want)
		}
	}
	check("Support", []string{Unknown.String(), Unsupported.String(), Supported.String()}, "unknown,unsupported,supported")
	check("Origin", []string{NotQueried.String(), Heuristic.String(), Environment.String(), Query.String(), Override.String()}, "not-queried,heuristic,env,query,override")
	check("Mux", []string{NoMux.String(), Tmux.String(), Screen.String(), Zellij.String()}, "none,tmux,screen,zellij")
	if s := Support(9).String(); s != "9" {
		t.Errorf("an unnamed Support prints %q, want its number", s)
	}
	if _, err := Support(9).MarshalText(); err == nil {
		t.Error("an unnamed Support marshalled")
	}
	var m Mux
	if err := m.UnmarshalText([]byte("byobu")); err == nil {
		t.Error("an unknown Mux name was read")
	}
}

func TestEnvLookup(t *testing.T) {
	e := Env{"TERM=xterm", "EMPTY=", "TERM=xterm-kitty", "NOEQUALS"}
	if v, ok := e.LookupEnv("TERM"); !ok || v != "xterm-kitty" {
		t.Errorf("TERM = %q, %v; want the last, xterm-kitty", v, ok)
	}
	if v, ok := e.LookupEnv("EMPTY"); !ok || v != "" {
		t.Errorf("EMPTY = %q, %v; want set and empty", v, ok)
	}
	for _, k := range []string{"NOEQUALS", "TER", "MISSING"} {
		if _, ok := e.LookupEnv(k); ok {
			t.Errorf("%s is reported set", k)
		}
	}
	if e.Getenv("MISSING") != "" {
		t.Error("Getenv of a missing key is not empty")
	}
}

func TestEnvironmentFacts(t *testing.T) {
	cases := []struct {
		env      Env
		terminal string
		mux      Mux
		remote   bool
	}{
		{Env{"TERM=xterm-256color"}, "xterm-256color", NoMux, false},
		{Env{"TERM=xterm-256color", "TERM_PROGRAM=WezTerm"}, "WezTerm", NoMux, false},
		{Env{"TERM=tmux-256color", "TMUX=/tmp/tmux-1/default,1,0"}, "tmux-256color", Tmux, false},
		{Env{"TERM=screen", "STY=1.pts-0.host"}, "screen", Screen, false},
		{Env{"TERM=xterm", "ZELLIJ=0"}, "xterm", Zellij, false},
		{Env{"TERM=xterm", "SSH_TTY=/dev/pts/1"}, "xterm", NoMux, true},
		{Env{"TERM=xterm", "SSH_CONNECTION=10.0.0.1 1 10.0.0.2 22"}, "xterm", NoMux, true},
	}
	for _, c := range cases {
		var got Caps
		got.setEnv(c.env)
		if got.Terminal != (Fact[string]{c.terminal, Environment}) {
			t.Errorf("%v: Terminal = %+v, want %q from env", c.env, got.Terminal, c.terminal)
		}
		if got.Mux != (Fact[Mux]{c.mux, Environment}) {
			t.Errorf("%v: Mux = %+v, want %v from env", c.env, got.Mux, c.mux)
		}
		if got.Remote != (Fact[bool]{c.remote, Environment}) {
			t.Errorf("%v: Remote = %+v, want %v from env", c.env, got.Remote, c.remote)
		}
	}
}

func TestEnvironmentDoesNotReplaceAQuery(t *testing.T) {
	var c Caps
	c.Terminal.Set("kitty(0.39.1)", Query)
	c.Mux.Set(Tmux, Query)
	c.setEnv(Env{"TERM=xterm-kitty"})
	if c.Terminal.Value != "kitty(0.39.1)" || c.Mux.Value != Tmux {
		t.Fatalf("the environment replaced a reply: %+v %+v", c.Terminal, c.Mux)
	}
}

func TestZeroCapsMarshalsEmpty(t *testing.T) {
	b, err := json.Marshal(Caps{})
	if err != nil || string(b) != "{}" {
		t.Fatalf("zero Caps = %s, %v; want {}", b, err)
	}
}

// full is a Caps with every field set away from its zero value. The
// reflection check below keeps it full as fields are added.
func full() Caps {
	q := func(s Support) Fact[Support] { return Fact[Support]{s, Query} }
	return Caps{
		Complete:           true,
		TimedOut:           true,
		Terminal:           Fact[string]{"WezTerm 20240203", Query},
		Mux:                Fact[Mux]{Tmux, Environment},
		Remote:             Fact[bool]{true, Environment},
		Attributes:         []int{62, 4, 22},
		KittyKeyboard:      q(Supported),
		KeyboardFlags:      1,
		SyncOutput:         q(Supported),
		GraphemeWidth:      q(Unsupported),
		ColorSchemeReports: q(Supported),
		InBandResize:       q(Unsupported),
		FocusEvents:        q(Supported),
		DesktopNotify:      Fact[Support]{Unsupported, Override},
		KittyGraphics:      Fact[Support]{Unknown, Heuristic},
		Sixel:              q(Supported),
		Dark:               Fact[bool]{true, Query},
		Background:         color.RGBA{R: 0x1e, G: 0x1f, B: 0x29, A: 0xff},
		Profile:            colorprofile.ANSI256,
	}
}

func TestFullCapsSetsEveryField(t *testing.T) {
	v := reflect.ValueOf(full())
	for i := range v.NumField() {
		if v.Field(i).IsZero() {
			t.Errorf("full() leaves %s zero; set it so the JSON round trip covers it", v.Type().Field(i).Name)
		}
	}
}

func TestCapsJSONRoundTrip(t *testing.T) {
	in := full()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"origin":"query"`, `"value":"supported"`, `"mux":{"value":"tmux","origin":"env"}`, `"background":"#1e1f29"`, `"profile":"ANSI256"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("JSON lacks %s:\n%s", want, b)
		}
	}
	var out Caps
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("%v\n%s", err, b)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip\n got %+v\nwant %+v", out, in)
	}
}

func TestCapsJSONRefusesBadValues(t *testing.T) {
	for _, s := range []string{
		`{"background":"#12345"}`,
		`{"background":"red"}`,
		`{"profile":"Sixteen"}`,
		`{"sync_output":{"value":"maybe"}}`,
		`{"sync_output":{"origin":"rumour"}}`,
	} {
		var c Caps
		if err := json.Unmarshal([]byte(s), &c); err == nil {
			t.Errorf("%s was read without an error: %+v", s, c)
		}
	}
}
