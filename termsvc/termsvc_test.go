package termsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// The commands' cases moved to termsvc/termsvctea with the commands
// (docs/decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md Step 9);
// what the core decides is read here through Prepare.

// prepared is what n.Prepare gives for x: the bytes, and the result its
// deliver returns.
func prepared(n *Notifier, x Notification) (string, NotifyResultMsg) {
	seq, deliver := n.Prepare(context.Background(), x)
	return seq, deliver()
}

// seqOf is the bytes n.Prepare gives for x.
func seqOf(n *Notifier, x Notification) string {
	seq, _ := prepared(n, x)
	return seq
}

// caps is a terminal of brand inside mux, whose answer to the OSC 99 query
// was notify.
func caps(brand termcap.Brand, mux termcap.Mux, notify termcap.Support) termcap.Caps {
	var c termcap.Caps
	c.Brand.Set(brand, termcap.Environment)
	c.Mux.Set(mux, termcap.Environment)
	c.DesktopNotify.Set(notify, termcap.Queried)
	return c
}

// notifier is an Always notifier that knows c.
func notifier(c termcap.Caps, o ...NotifyOption) *Notifier {
	n := NewNotifier(append([]NotifyOption{WithPolicy(Always)}, o...)...)
	n.SetCaps(c)
	return n
}

func TestNotificationBytes(t *testing.T) {
	kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
	cases := []struct {
		name string
		p    Protocol
		x    Notification
		want string
	}{
		{"OSC 99 title and body", OSC99, Notification{Title: "Done", Body: "3 files", ID: "build"},
			"\x1b]99;i=build:d=0;Done\x07\x1b]99;i=build:p=body;3 files\x07"},
		{"OSC 99 title only, critical", OSC99, Notification{Title: "Failed", ID: "t", Urgency: Critical},
			"\x1b]99;i=t:u=2;Failed\x07"},
		{"OSC 99 body only, low", OSC99, Notification{Body: "fyi", ID: "t", Urgency: Low},
			"\x1b]99;i=t:u=0:p=body;fyi\x07"},
		{"OSC 99 numbers a notification with no ID", OSC99, Notification{Title: "x"}, "\x1b]99;i=n1;x\x07"},
		{"OSC 99 keeps only the ID's allowed characters", OSC99, Notification{Title: "x", ID: "a:b;c=d"}, "\x1b]99;i=abcd;x\x07"},
		{"OSC 777", OSC777, Notification{Title: "Done; ok", Body: "a;b"}, "\x1b]777;notify;Done, ok;a,b\x07"},
		{"OSC 9", OSC9, Notification{Title: "Done", Body: "3 files"}, "\x1b]9;Done: 3 files\x07"},
		{"OSC 9 never reads as a sub-command", OSC9, Notification{Title: "4;1;50"}, "\x1b]9; 4;1;50\x07"},
		{"bell", Bell, Notification{Title: "Done"}, "\a"},
		{"off", Off, Notification{Title: "Done"}, ""},
	}
	for _, c := range cases {
		if got := seqOf(notifier(kitty, WithProtocol(c.p)), c.x); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestAutoPicksTheProtocol(t *testing.T) {
	cases := []struct {
		c    termcap.Caps
		want Protocol
	}{
		{caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported), OSC99},
		{caps(termcap.BrandGhostty, termcap.NoMux, termcap.Supported), OSC99},
		{caps(termcap.BrandITerm2, termcap.NoMux, termcap.Supported), OSC99},
		{caps(termcap.BrandWezTerm, termcap.NoMux, termcap.Supported), OSC99},
		{caps(termcap.BrandGhostty, termcap.NoMux, termcap.Unsupported), OSC777},
		{caps(termcap.BrandVTE, termcap.NoMux, termcap.Unknown), OSC777},
		{caps(termcap.BrandFoot, termcap.NoMux, termcap.Unsupported), OSC777},
		{caps(termcap.BrandITerm2, termcap.NoMux, termcap.Unsupported), OSC9},
		{caps(termcap.BrandWezTerm, termcap.NoMux, termcap.Unsupported), OSC9},
		{caps(termcap.BrandWarp, termcap.NoMux, termcap.Unsupported), OSC9},
		{caps(termcap.BrandXTerm, termcap.NoMux, termcap.Unsupported), Bell},
		{caps(termcap.BrandKitty, termcap.NoMux, termcap.Unsupported), Bell},
		{caps(termcap.BrandITerm2, termcap.Zellij, termcap.Unsupported), Bell},
		{termcap.Caps{}, Bell},
	}
	for _, c := range cases {
		if got := notifier(c.c).chosen(); got != c.want {
			t.Errorf("%v, OSC 99 %v: Auto chose %d, want %d", c.c.Brand.Value, c.c.DesktopNotify.Value, got, c.want)
		}
	}
}

func TestNotificationsAreWrappedInsideAMultiplexer(t *testing.T) {
	x := Notification{Title: "Done", ID: "a"}
	inner := "\x1b]99;i=a;Done\x07"
	tmux := caps(termcap.BrandKitty, termcap.Tmux, termcap.Supported)
	if got := seqOf(notifier(tmux), x); got != ansi.TmuxPassthrough(inner) {
		t.Errorf("tmux: %q, want the tmux passthrough of %q", got, inner)
	}
	screen := caps(termcap.BrandKitty, termcap.Screen, termcap.Supported)
	if got := seqOf(notifier(screen), x); got != ansi.ScreenPassthrough(inner, screenLimit) {
		t.Errorf("screen: %q, want the screen passthrough of %q", got, inner)
	}
	if got := seqOf(notifier(tmux, WithProtocol(Bell)), x); got != "\a" {
		t.Errorf("tmux bell: %q, want a bare BEL, which tmux passes on", got)
	}
}

// TestFocusPolicies drives each policy through unknown, focused, blurred
// and focused-again, and checks what was prepared and why not (MADR A1,
// Q6). termsvctea's TestFocusPolicies does the same through tea's focus
// messages.
func TestFocusPolicies(t *testing.T) {
	x := Notification{Title: "Done"}
	focus := func(f bool) func(*Notifier) { return func(n *Notifier) { n.SetFocus(f) } }
	steps := []struct {
		name string
		set  func(*Notifier)
	}{{"unknown", nil}, {"focused", focus(true)}, {"blurred", focus(false)}, {"focused again", focus(true)}}
	want := map[Policy][]SkipReason{
		WhenUnfocused: {SkipFocusUnknown, SkipFocused, NotSkipped, SkipFocused},
		UnlessFocused: {NotSkipped, SkipFocused, NotSkipped, SkipFocused},
		Always:        {NotSkipped, NotSkipped, NotSkipped, NotSkipped},
		Never:         {SkipDisabled, SkipDisabled, SkipDisabled, SkipDisabled},
	}
	for policy, reasons := range want {
		n := NewNotifier(WithProtocol(Bell), WithPolicy(policy))
		for i, s := range steps {
			if s.set != nil {
				s.set(n)
			}
			seq, r := prepared(n, x)
			if r.Skipped != reasons[i] || r.Sent != (reasons[i] == NotSkipped) {
				t.Errorf("policy %d, %s: Sent %v, Skipped %q; want skipped %q", policy, s.name, r.Sent, r.Skipped, reasons[i])
			}
			if (seq == "\a") != r.Sent {
				t.Errorf("policy %d, %s: bytes %q with Sent %v", policy, s.name, seq, r.Sent)
			}
		}
	}
}

func TestEmptyNotificationIsNotSent(t *testing.T) {
	seq, r := prepared(notifier(termcap.Caps{}, WithProtocol(Bell)), Notification{Title: "\x1b\x07", Body: "\x9b\x1b[31m\n"})
	if r.Sent || r.Skipped != SkipEmpty || seq != "" {
		t.Fatalf("a notification of controls only: %q, %+v", seq, r)
	}
	if _, r := prepared(notifier(termcap.Caps{}, WithProtocol(Off)), Notification{Title: "x"}); r.Skipped != SkipDisabled {
		t.Fatalf("protocol Off: %+v", r)
	}
}

type backend struct {
	got []Notification
	err error
}

func (b *backend) Notify(_ context.Context, x Notification) error {
	b.got = append(b.got, x)
	return b.err
}

func TestBackend(t *testing.T) {
	b := &backend{}
	seq, deliver := notifier(termcap.Caps{}, WithBackend(b)).Prepare(context.Background(), Notification{Title: "Do\x1b[1mne", Body: "b"})
	if seq != "" || len(b.got) != 0 {
		t.Fatalf("a backend's notification: bytes %q, backend called %d times before deliver", seq, len(b.got))
	}
	if r := deliver(); !r.Sent || r.Skipped != NotSkipped {
		t.Errorf("a backend's success: %+v", r)
	}
	if len(b.got) != 1 || b.got[0].Title != "Done" {
		t.Errorf("backend got %+v, want the cleaned notification", b.got)
	}
	failing := &backend{err: errors.New("no desktop")}
	_, r := prepared(notifier(termcap.Caps{}, WithBackend(failing)), Notification{Title: "x"})
	if r.Sent || r.Skipped != SkipFailed || !errors.Is(r.Err, failing.err) || r.Notification.Title != "x" {
		t.Errorf("a backend's error came back as %+v", r)
	}
	gated := &backend{}
	if seq, r := prepared(notifier(termcap.Caps{}, WithBackend(gated), WithPolicy(Never)), Notification{Title: "x"}); seq != "" || r.Skipped != SkipDisabled || len(gated.got) != 0 {
		t.Errorf("a skipped notification reached the backend: %q, %+v, %d calls", seq, r, len(gated.got))
	}
}

func TestControlBytesAreStripped(t *testing.T) {
	evil := "a\x1b]52;c;aGk=\x07b\x9bc\x00d\x7fe"
	got := seqOf(notifier(termcap.Caps{}, WithProtocol(OSC9)), Notification{Title: evil, Body: evil})
	// The OSC 52 sequence goes whole. The raw byte 0x9b is CSI in its 8-bit
	// form, so it goes with the "c" that ends it, as a terminal in 8-bit
	// mode would read it; NUL and DEL go too.
	if want := "\x1b]9;abde: abde\x07"; got != want {
		t.Errorf("notification %q, want %q", got, want)
	}
	if got := strip("one\ntwo\tthree\r"); got != "one two three " {
		t.Errorf("strip turned whitespace into %q", got)
	}
	// C1 controls encoded as UTF-8 are their 7-bit forms: CSI 31m, NEL and
	// ESC @ go whole, and U+009F opens an APC, which hides the rest
	// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
	// A3.1, class 1).
	if got := strip("a\u009b31mb\u0085c\u0080d\u009fe"); got != "abcd" {
		t.Errorf("strip left UTF-8 C1 controls: %q", got)
	}
	link, err := Link("https://example.com/\x1b]8;;evil\x07x", "te\x1bxt\x9b", "id=\x07a")
	if err != nil {
		t.Fatal(err)
	}
	// sanitize.Line drops each escape sequence whole: the URL's embedded OSC
	// 8, and in the text "\x1bx", which is ESC x, as a terminal reads it
	// (docs/decisions/0014-PLAN-hardening.md Step 2). The raw 0x9b at the
	// text's end is invalid UTF-8, one U+FFFD
	// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
	// D9 S2).
	if want := "\x1b]8;id=a;https://example.com/x\x07tet\ufffd\x1b]8;;\x07"; link != want {
		t.Errorf("link %q, want %q", link, want)
	}
}

func TestLinkSchemes(t *testing.T) {
	for _, u := range []string{"http://example.com", "https://example.com", "HTTPS://example.com", "file:///tmp/x", "mailto:a@example.com"} {
		if _, err := Link(u, "x"); err != nil {
			t.Errorf("Link(%q) refused: %v", u, err)
		}
	}
	for _, u := range []string{"javascript:alert(1)", "data:text/html,x", "vbscript:x", "ftp://example.com", "example.com"} {
		if s, err := Link(u, "x"); !errors.Is(err, ErrScheme) {
			t.Errorf("Link(%q) = %q, %v; want ErrScheme", u, s, err)
		}
	}
	if s, _ := Link("https://example.com", "site"); s != "\x1b]8;;https://example.com\x07site\x1b]8;;\x07" {
		t.Errorf("Link = %q", s)
	}
}

func TestWrap(t *testing.T) {
	seq := "\x1b]9;x\x07"
	var c termcap.Caps
	if Wrap(c, seq) != seq {
		t.Error("Wrap changed a sequence outside a multiplexer")
	}
	c.Mux.Set(termcap.Zellij, termcap.Environment)
	if Wrap(c, seq) != seq {
		t.Error("Wrap changed a sequence inside Zellij, which has no passthrough")
	}
}

func TestPromptMarks(t *testing.T) {
	for got, want := range map[string]string{
		PromptStart():      ansi.FinalTermPrompt(),
		CommandStart():     ansi.FinalTermCmdStart(),
		CommandExecuted():  ansi.FinalTermCmdExecuted(),
		CommandFinished(2): ansi.FinalTermCmdFinished("2"),
	} {
		if got != want {
			t.Errorf("mark %q, want %q", got, want)
		}
	}
	if CommandFinished(0) != "\x1b]133;D;0\x07" {
		t.Errorf("CommandFinished(0) = %q", CommandFinished(0))
	}
}
