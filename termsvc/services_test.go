package termsvc

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

type clipboard struct {
	got []string
	err error
}

func (c *clipboard) Copy(_ context.Context, text string) error {
	c.got = append(c.got, text)
	return c.err
}

// TestPrepareCopyRoutes: each route PrepareCopy picks, what its deliver
// reports, and that a clipboard is called only by deliver.
func TestPrepareCopyRoutes(t *testing.T) {
	var tmux termcap.Caps
	tmux.Mux.Set(termcap.Tmux, termcap.Environment)
	ctx := context.Background()
	for _, c := range []struct {
		name  string
		caps  termcap.Caps
		text  string
		route Route
		want  CopiedMsg
	}{
		{"OSC 52", termcap.Caps{}, "x", RouteOSC52, CopiedMsg{Status: Unconfirmed, Route: RouteOSC52}},
		{"OSC 52 inside tmux", tmux, "x", RouteOSC52Tmux, CopiedMsg{Status: Unconfirmed, Route: RouteOSC52Tmux}},
		{"the largest payload", termcap.Caps{}, strings.Repeat("x", MaxCopyBytes), RouteOSC52, CopiedMsg{Status: Unconfirmed, Route: RouteOSC52}},
		{"one byte over", termcap.Caps{}, strings.Repeat("x", MaxCopyBytes+1), RouteNone, CopiedMsg{Status: Failed}},
		{"one byte over, inside tmux", tmux, strings.Repeat("x", MaxCopyBytes+1), RouteNone, CopiedMsg{Status: Failed}},
	} {
		route, deliver := PrepareCopy(ctx, c.caps, c.text)
		if got := deliver(); route != c.route || got != c.want {
			t.Errorf("%s: %v, %+v; want %v, %+v", c.name, route, got, c.route, c.want)
		}
		if _, r, _ := CopySequence(c.caps, c.text); r != route {
			t.Errorf("%s: CopySequence's route %v, PrepareCopy's %v", c.name, r, route)
		}
	}

	ok := &clipboard{}
	route, deliver := PrepareCopy(ctx, tmux, "hello", WithClipboard(ok))
	if route != RouteBackend || len(ok.got) != 0 {
		t.Fatalf("a clipboard: route %v, called %d times before deliver", route, len(ok.got))
	}
	if got := deliver(); got != (CopiedMsg{Status: Confirmed, Route: RouteBackend}) || len(ok.got) != 1 || ok.got[0] != "hello" {
		t.Errorf("a clipboard's success: %+v, clipboard got %q", got, ok.got)
	}
	bad := &clipboard{err: errors.New("no display")}
	if _, deliver := PrepareCopy(ctx, termcap.Caps{}, "x", WithClipboard(bad)); func() bool {
		got := deliver()
		return got.Status != Failed || got.Route != RouteBackend || !errors.Is(got.Err, bad.err)
	}() {
		t.Error("a clipboard's error was not reported")
	}
	if route, deliver := PrepareCopy(ctx, termcap.Caps{}, strings.Repeat("x", MaxCopyBytes+1), WithClipboard(ok)); route != RouteNone || deliver().Status != Failed || len(ok.got) != 1 {
		t.Errorf("an oversized copy reached the clipboard: route %v, %d calls", route, len(ok.got))
	}
}

func TestCopyPlan(t *testing.T) {
	var tmux termcap.Caps
	tmux.Mux.Set(termcap.Tmux, termcap.Environment)
	if got := CopyPlan(tmux); !reflect.DeepEqual(got, []Route{RouteBackend, RouteTmuxBuffer, RouteOSC52, RouteOSC52Tmux}) {
		t.Errorf("tmux plan %v", got)
	}
	if got := CopyPlan(termcap.Caps{}); !reflect.DeepEqual(got, []Route{RouteBackend, RouteOSC52}) {
		t.Errorf("plan %v", got)
	}
	if got := TmuxLoadBuffer(); !reflect.DeepEqual(got, []string{"tmux", "load-buffer", "-w", "-"}) {
		t.Errorf("TmuxLoadBuffer %q", got)
	}
}

func TestImageReadCommands(t *testing.T) {
	first := func(cmds [][]string) []string {
		out := make([]string, len(cmds))
		for i, c := range cmds {
			out[i] = c[0]
		}
		return out
	}
	cases := []struct {
		goos    string
		wayland bool
		want    []string
	}{
		{"darwin", false, []string{"osascript"}},
		{"darwin", true, []string{"osascript"}},
		{"windows", false, []string{"powershell"}},
		{"linux", true, []string{"wl-paste", "xclip", "powershell.exe"}},
		{"linux", false, []string{"xclip", "powershell.exe"}},
		{"freebsd", false, []string{"xclip", "powershell.exe"}},
	}
	for _, c := range cases {
		if got := first(ImageReadCommands(c.goos, c.wayland)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s, wayland %v: %q, want %q", c.goos, c.wayland, got, c.want)
		}
	}
	if cmd := ImageReadCommands("darwin", false)[0]; !strings.Contains(cmd[2], "PNGf") {
		t.Errorf("osascript reads %q, want PNG", cmd)
	}
}

func TestLinkDisplay(t *testing.T) {
	view := func(b termcap.Brand, mux termcap.Mux, tmux string) Display {
		var c termcap.Caps
		c.Brand.Set(b, termcap.Environment)
		c.Mux.Set(mux, termcap.Environment)
		c.Tmux.Version = tmux
		return LinkDisplay(c)
	}
	for _, b := range []termcap.Brand{
		termcap.BrandGhostty, termcap.BrandITerm2, termcap.BrandWezTerm, termcap.BrandKitty, termcap.BrandVSCode,
		termcap.BrandAlacritty, termcap.BrandWindowsTerminal, termcap.BrandKonsole, termcap.BrandVTE,
	} {
		if got := view(b, termcap.NoMux, ""); got != LabelOnly {
			t.Errorf("%v: %d, want the label only", b, got)
		}
	}
	for name, got := range map[string]Display{
		"Apple Terminal": view(termcap.BrandAppleTerminal, termcap.NoMux, ""),
		"Warp":           view(termcap.BrandWarp, termcap.NoMux, ""),
		"unknown":        view(termcap.BrandUnknown, termcap.NoMux, ""),
		"screen":         view(termcap.BrandKitty, termcap.Screen, ""),
		"Zellij":         view(termcap.BrandKitty, termcap.Zellij, ""),
		"tmux 3.3":       view(termcap.BrandKitty, termcap.Tmux, "3.3"),
	} {
		if got != LabelAndURL {
			t.Errorf("%s: %d, want the label and URL", name, got)
		}
	}
	if got := view(termcap.BrandKitty, termcap.Tmux, "3.4"); got != LabelOnly {
		t.Errorf("tmux 3.4 in kitty: %d, want the label only", got)
	}
}

func TestOpenable(t *testing.T) {
	var p LinkPolicy
	for _, u := range []string{"https://example.com/x", "HTTP://example.com"} {
		if !p.Openable(u) {
			t.Errorf("default policy refused %q", u)
		}
	}
	for _, u := range []string{
		"javascript:alert(1)", "file:///etc/passwd", "data:text/html,x", "mailto:a@example.com",
		"https://example.com/\x1b]8;;x", "https://example.com/\x07", "https://exa mple.com", "example.com", "",
	} {
		if p.Openable(u) {
			t.Errorf("default policy opens %q", u)
		}
	}
	files := LinkPolicy{Schemes: []string{"file"}}
	if !files.Openable("file:///tmp/x") || files.Openable("https://example.com") {
		t.Error("a policy of file only")
	}
}

func TestNotificationText(t *testing.T) {
	n := notifier(termcap.Caps{}, WithProtocol(OSC777))
	_, r := prepared(n, Notification{Title: "a\n\n  b\r\nc\x1b[1m d", Body: strings.Repeat("漢字", 200)})
	if r.Notification.Title != "a b c d" {
		t.Errorf("title %q, want line breaks collapsed and the sequence gone", r.Notification.Title)
	}
	body := r.Notification.Body
	if w := ansi.StringWidth(body); w != BodyCells {
		t.Errorf("body is %d cells, want %d", w, BodyCells)
	}
	if !utf8.ValidString(body) || utf8.RuneCountInString(body) != BodyCells/2 {
		t.Errorf("body cut to %d runes, valid %v; want %d whole characters", utf8.RuneCountInString(body), utf8.ValidString(body), BodyCells/2)
	}
	_, long := prepared(n, Notification{Title: strings.Repeat("t", 100)})
	if len(long.Notification.Title) != TitleCells {
		t.Errorf("title of %d, want %d", len(long.Notification.Title), TitleCells)
	}
}

func TestGate(t *testing.T) {
	n := notifier(termcap.Caps{}, WithProtocol(Bell), WithNotifyFilter(func(x Notification) bool { return x.Title == "done" }))
	if _, r := prepared(n, Notification{Title: "working"}); r.Sent || r.Skipped != SkipGated {
		t.Errorf("gated: %+v", r)
	}
	if seq, r := prepared(n, Notification{Title: "done"}); !r.Sent || seq != "\a" {
		t.Errorf("passed the gate: %+v", r)
	}
}

func TestSanitizeTitle(t *testing.T) {
	in := "a\x1b]0;evil\x07b\u202ec\u2066d" + strings.Repeat("x", 300)
	got := SanitizeTitle(in)
	if strings.ContainsAny(got, "\x1b\x07\u202e\u2066") {
		t.Errorf("title keeps a control or bidi mark: %q", got[:12])
	}
	if !strings.HasPrefix(got, "abcd") || utf8.RuneCountInString(got) != TitleRunes {
		t.Errorf("title %q… of %d runes, want abcd… of %d", got[:8], utf8.RuneCountInString(got), TitleRunes)
	}
	if got := SanitizeTitle("short\u200e"); got != "short" {
		t.Errorf("SanitizeTitle = %q", got)
	}
}

func TestActivity(t *testing.T) {
	at := time.UnixMilli(1_700_000_000_123)
	if got := Activity("kilo", ActivityBusy, at); got != "\x1b]777;kilo;activity;1;busy;1700000000123\x07" {
		t.Errorf("Activity = %q", got)
	}
	// "\x1bd" is a whole escape sequence, ESC d, so sanitize.Line drops it,
	// "d" and all, as a terminal reads it (docs/decisions/0014-PLAN-hardening.md
	// Step 2); then the ';' goes.
	if got := Activity("ve;n\x1bdor", ActivityDone, at); !strings.HasPrefix(got, "\x1b]777;venor;activity;") {
		t.Errorf("vendor not cleaned: %q", got)
	}
}

func TestParseActivity(t *testing.T) {
	now := time.UnixMilli(1_700_000_000_000)
	payload := func(version string, at time.Time) string {
		return "kilo;activity;" + version + ";waiting;" + strconv.FormatInt(at.UnixMilli(), 10)
	}
	cases := []struct {
		name string
		p    string
		ok   bool
	}{
		{"now", payload("1", now), true},
		{"4 s old", payload("1", now.Add(-4*time.Second)), true},
		{"just under 15 s old", payload("1", now.Add(-15*time.Second+time.Millisecond)), true},
		{"15 s old", payload("1", now.Add(-15*time.Second)), false},
		{"5 s ahead", payload("1", now.Add(5*time.Second)), true},
		{"6 s ahead", payload("1", now.Add(6*time.Second)), false},
		{"version 2", payload("2", now), false},
		{"an unknown state", "kilo;activity;1;sleeping;" + strconv.FormatInt(now.UnixMilli(), 10), false},
		{"another extension", "kilo;notify;1;busy;" + strconv.FormatInt(now.UnixMilli(), 10), false},
		{"a short payload", "kilo;activity;1;busy", false},
		{"a bad time", "kilo;activity;1;busy;soon", false},
	}
	for _, c := range cases {
		r, err := ParseActivity(c.p, now)
		if (err == nil) != c.ok {
			t.Errorf("%s: %+v, %v; want ok %v", c.name, r, err, c.ok)
		}
		if err != nil && !errors.Is(err, ErrActivity) {
			t.Errorf("%s: error %v is not ErrActivity", c.name, err)
		}
	}
	r, _ := ParseActivity(payload("1", now), now)
	if r.Vendor != "kilo" || r.State != ActivityWaiting || !r.At.Equal(now) {
		t.Errorf("report %+v", r)
	}
}

func TestPointer(t *testing.T) {
	c := func(b termcap.Brand, mux termcap.Mux) termcap.Caps {
		var caps termcap.Caps
		caps.Brand.Set(b, termcap.Environment)
		caps.Mux.Set(mux, termcap.Environment)
		return caps
	}
	cases := []struct {
		name  string
		c     termcap.Caps
		shape string
		want  string
	}{
		{"Ghostty", c(termcap.BrandGhostty, termcap.NoMux), "pointer", "\x1b]22;pointer\x07"},
		{"kitty", c(termcap.BrandKitty, termcap.NoMux), "pointer", "\x1b]22;pointer\x07"},
		{"kitty resets", c(termcap.BrandKitty, termcap.NoMux), "", "\x1b]22;\x07"},
		{"Ghostty resets", c(termcap.BrandGhostty, termcap.NoMux), "", "\x1b]22;default\x07"},
		{"a shape cleaned", c(termcap.BrandKitty, termcap.NoMux), "poi\x1bn;ter", "\x1b]22;pointer\x07"},
		{"tmux", c(termcap.BrandKitty, termcap.Tmux), "pointer", ""},
		{"Zellij", c(termcap.BrandGhostty, termcap.Zellij), "pointer", ""},
		{"WezTerm", c(termcap.BrandWezTerm, termcap.NoMux), "pointer", ""},
	}
	for _, tc := range cases {
		if got := Pointer(tc.c, tc.shape); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestProgressSupported(t *testing.T) {
	c := func(b termcap.Brand, terminal string) termcap.Caps {
		var caps termcap.Caps
		caps.Brand.Set(b, termcap.Queried)
		caps.Terminal.Set(terminal, termcap.Queried)
		return caps
	}
	for name, tc := range map[string]struct {
		c    termcap.Caps
		want bool
	}{
		"Ghostty":                {c(termcap.BrandGhostty, "ghostty 1.1"), true},
		"WezTerm":                {c(termcap.BrandWezTerm, "WezTerm"), true},
		"iTerm2 3.6.1":           {c(termcap.BrandITerm2, "iTerm2 3.6.1"), true},
		"iTerm2 4.0":             {c(termcap.BrandITerm2, "iTerm2 4.0"), true},
		"iTerm2 3.5.4":           {c(termcap.BrandITerm2, "iTerm2 3.5.4"), false},
		"iTerm2, version unseen": {c(termcap.BrandITerm2, "iTerm.app"), false},
		"kitty":                  {c(termcap.BrandKitty, "kitty(0.39.1)"), false},
	} {
		if got := ProgressSupported(tc.c); got != tc.want {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
}
