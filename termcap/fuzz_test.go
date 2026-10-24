package termcap

import (
	"encoding"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// Fuzz targets (docs/decisions/0014-PLAN-hardening.md Step 9, finding H9).
// The seeds are the existing tests' inputs and the replies termcaptest's
// profiles give (termcaptest.go, answerCSI and answerString), written out
// here: termcaptest imports this package. The Prober's target,
// FuzzProberReplies, moved to termcaptea with it.

// replySeeds are replies of every kind and terminator, and the ones that
// must stay unknown.
var replySeeds = []string{
	"\x1b[?62;22c", "\x1b[?c", "\x1b[>1;4000;39c", "\x1bP>|kitty(0.39.1)\x1b\\", "\x1bP>|x\a",
	"\x1b[?2026;2$y", "\x1b[?2031;5$y", "\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\", "\x1b]10;#cdd6f4\a",
	"\x1b]4;15;rgb:ff/00/80\x9c", "\x1b]11;rgb:0/0/0\x1b", "\x9d11;rgb:f/f/f\a", "\x9b?1u",
	"\x1b[?997;1n", "\x1b_Gi=31;OK\x1b\\", "\x9fGi=31;EINVAL\x9c", "\x90>|v\x9c",
	"\x1b]99;i=termcap:p=?;a=focus\x1b\\", "\x1b]11;?\a", "\x1b[" + strings.Repeat("1;", 40) + "c",
	"\x1b]4;1;" + strings.Repeat("f", 1100) + "\a", "", "x", "\x1b", "\x1b]11;rgb:0/0/0\x18",
}

// FuzzParseReply: ParseReply never panics; Raw is always the input; a
// ReplyColor holds a colour; and a ReplyMode's Setting is DECRPM's 0 to 4.
func FuzzParseReply(f *testing.F) {
	for _, s := range replySeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		r := ParseReply(raw)
		if r.Raw != raw {
			t.Fatalf("ParseReply(%q).Raw = %q", raw, r.Raw)
		}
		if r.Kind == ReplyColor && r.Color == nil {
			t.Fatalf("ParseReply(%q) is a colour with none", raw)
		}
		if r.Kind == ReplyMode && r.Setting > ModePermanentlyReset {
			t.Fatalf("ParseReply(%q) is DECRPM with Pm %d", raw, r.Setting)
		}
		if r.Kind != ReplyUnknown && len(raw) > maxReply {
			t.Fatalf("ParseReply read a reply of %d bytes as %v", len(raw), r.Kind)
		}
	})
}

// probeEnvs are the environments FuzzProbe starts a probe in.
var probeEnvs = []Env{
	{"TERM=xterm-kitty"}, {"TERM=xterm-256color", "SSH_TTY=x"}, {"TERM=tmux-256color", "TMUX=x"},
	{"TERMINAL_EMULATOR=JetBrains-JediTerm"}, {"TERM=xterm-kitty", "NVIM=x"}, {},
}

// probeOp is one step of a FuzzProbe script.
type probeOp struct {
	kind byte // 0 start, 1 reply, 2 timeout, 3 hook
	arg  byte
	raw  string
}

// script reads data as FuzzProbe's operations: each starts with its kind
// and one argument byte; a reply's argument is its length, and its bytes
// follow.
func script(data []byte) []probeOp {
	var ops []probeOp
	for len(data) >= 2 {
		op := probeOp{kind: data[0] % 4, arg: data[1]}
		data = data[2:]
		if op.kind == 1 {
			n := min(int(op.arg), len(data))
			op.raw, data = string(data[:n]), data[n:]
		}
		ops = append(ops, op)
	}
	return ops
}

// seedScript writes ops as script reads them.
func seedScript(ops ...probeOp) []byte {
	var b []byte
	for _, op := range ops {
		if op.kind == 1 {
			op.arg = byte(len(op.raw))
		}
		b = append(append(b, op.kind, op.arg), op.raw...)
	}
	return b
}

// FuzzProbe: a probe driven by any script of starts, replies, timeouts
// and hook answers, with time only rising, never panics; ends at most once;
// gives no deadline once ended; and holds only facts whose origin is one
// of Origin's.
func FuzzProbe(f *testing.F) {
	f.Add(seedScript(probeOp{kind: 0}, probeOp{kind: 1, raw: "\x1b[?1u"}, probeOp{kind: 1, raw: "\x1b[?62;4c"}))
	f.Add(seedScript(probeOp{kind: 0, arg: 1}, probeOp{kind: 1, raw: "\x1b[>1;95;0c"}, probeOp{kind: 1, raw: "\x1b[?1;2c"}))
	f.Add(seedScript(probeOp{kind: 0, arg: 2}, probeOp{kind: 2, arg: 250}, probeOp{kind: 1, raw: "\x1b[?997;2n"}))
	f.Add(seedScript(probeOp{kind: 0}, probeOp{kind: 3, arg: 0x0d}, probeOp{kind: 1, raw: "\x1b[?2031;2$y"},
		probeOp{kind: 1, raw: "\x1b[?62c"}, probeOp{kind: 1, raw: "\x1b[?997;1n"}))
	f.Add(seedScript(probeOp{kind: 0, arg: 3}, probeOp{kind: 1, raw: "\x1b]11;rgb:0/0/0\a"}))
	f.Add(seedScript(probeOp{kind: 0}, probeOp{kind: 1, raw: "\x1b[?62c"}, probeOp{kind: 1, raw: "\x1b[?62c"}, probeOp{kind: 2, arg: 255}))
	f.Fuzz(func(t *testing.T, data []byte) {
		p := NewProbe(WithGOOS("windows"), WithTimeout(time.Second),
			WithAppearanceHook(func() (bool, bool) { return true, true }),
			WithConsoleHost(func() (bool, bool) { return false, true }))
		now := time.Unix(0, 0)
		done := 0
		drain := func() {
			for e := p.NextEvent(); e.Kind != ProbeNoEvent; e = p.NextEvent() {
				if e.Kind == ProbeDone {
					done++
				}
			}
		}
		for _, op := range script(data) {
			now = now.Add(time.Duration(op.arg) * 10 * time.Millisecond)
			switch op.kind {
			case 0:
				p.Start(probeEnvs[int(op.arg)%len(probeEnvs)], now)
			case 1:
				p.HandleReply(ParseReply(op.raw), now)
			case 2:
				p.HandleTimeout(now)
			case 3:
				p.HandleHook(ProbeHook(op.arg%3), op.arg&4 != 0, op.arg&8 != 0)
			}
			drain()
			if done > 1 {
				t.Fatalf("ProbeDone %d times", done)
			}
			if _, ok := p.Timeout(); ok && done > 0 {
				t.Fatal("a deadline after ProbeDone")
			}
		}
		v := reflect.ValueOf(p.Caps())
		for i := range v.NumField() {
			if v.Field(i).Kind() != reflect.Struct {
				continue
			}
			if f := v.Field(i).FieldByName("Origin"); f.IsValid() {
				if o, ok := f.Interface().(Origin); ok && o > Override {
					t.Fatalf("%s has origin %d", v.Type().Field(i).Name, o)
				}
			}
		}
		if v := p.Caps().Terminal.Value; sanitize.Line(v) != v {
			t.Fatalf("Caps.Terminal is %q, which Line changes", v)
		}
	})
}

// FuzzParseTmux: ParseTmux never panics; Known means five tab-separated
// fields and a version; and no list holds an empty entry.
func FuzzParseTmux(f *testing.F) {
	for _, s := range []string{
		"3.4\tcsi-u\t1\tRGB,extkeys,focus,mouse\tattached,focused,UTF-8\n",
		"3.3a\txterm\toff\t\t\r\n",
		"3.4\tcsi-u\t1\tRGB",
		"\tcsi-u\t1\t\t",
		"3.4\tcsi-u\ton\t, ,a,,b\tcontrol-mode\t\n",
		"",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, out string) {
		got := ParseTmux(out)
		if got.Known {
			fields := strings.Split(strings.TrimRight(out, "\r\n"), "\t")
			if len(fields) != 5 || got.Version == "" {
				t.Fatalf("%q is Known with %d fields and version %q", out, len(fields), got.Version)
			}
		}
		for _, l := range [][]string{got.TermFeatures, got.ClientFlags} {
			for _, e := range l {
				if e == "" || strings.TrimSpace(e) != e {
					t.Fatalf("%q gives the list entry %q", out, e)
				}
			}
		}
	})
}

// textEnum is a termcap enum, through its pointer.
type textEnum interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

// FuzzEnumText: for each of termcap's enums, a name UnmarshalText takes is
// the name MarshalText gives back, and reads back to the same value.
func FuzzEnumText(f *testing.F) {
	for _, s := range []string{"supported", "unknown", "queried", "tmux", "kitty", "vscode", "wsl", "issue", "", "Supported", "7"} {
		f.Add([]byte(s))
	}
	fresh := []func() textEnum{
		func() textEnum { return new(Support) }, func() textEnum { return new(Origin) },
		func() textEnum { return new(Mux) }, func() textEnum { return new(Brand) },
		func() textEnum { return new(Editor) }, func() textEnum { return new(Platform) },
		func() textEnum { return new(ColorProfile) }, func() textEnum { return new(ReplyKind) },
		func() textEnum { return new(ModeSetting) }, func() textEnum { return new(ProbeEventKind) },
		func() textEnum { return new(ProbeHook) },
		func() textEnum { return new(Disposition) },
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		for _, mk := range fresh {
			v := mk()
			if v.UnmarshalText(b) != nil {
				continue
			}
			out, err := v.MarshalText()
			if err != nil || string(out) != string(b) {
				t.Fatalf("%T: %q reads, and writes back as %q, %v", v, b, out, err)
			}
			back := mk()
			if err := back.UnmarshalText(out); err != nil || !reflect.DeepEqual(back, v) {
				t.Fatalf("%T: %q reads back as %v, %v; want %v", v, out, back, err, v)
			}
		}
	})
}

// FuzzColorFGBG: when colorFGBG reads a value, its last field is a colour
// from 0 to 15, and dark follows Vim's rule: 0-6 and 8 are dark. When it
// does not, dark is false.
func FuzzColorFGBG(f *testing.F) {
	for _, s := range []string{"15;0", "0;15", "7;default", "0;7", "12;8", "1;2;3", "default;default", "", ";", "0;16", "0;-1", "0;+7"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, v string) {
		dark, ok := colorFGBG(v)
		if !ok {
			if dark {
				t.Fatalf("%q: not read, but dark", v)
			}
			return
		}
		n, err := strconv.Atoi(v[strings.LastIndexByte(v, ';')+1:])
		if err != nil || n < 0 || n > 15 {
			t.Fatalf("%q: read, but its background is %d, %v", v, n, err)
		}
		if want := n <= 6 || n == 8; dark != want {
			t.Fatalf("%q: dark %v, want %v for colour %d", v, dark, want, n)
		}
	})
}
