package termsvc

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// run executes cmd and every command it batches, and returns the messages.
func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	v := reflect.ValueOf(msg)
	if v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
		var out []tea.Msg
		for i := range v.Len() {
			out = append(out, run(v.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// raw is the tea.Raw writes cmd makes, joined.
func raw(cmd tea.Cmd) string {
	var b strings.Builder
	for _, m := range run(cmd) {
		if r, ok := m.(tea.RawMsg); ok {
			fmt.Fprint(&b, r.Msg)
		}
	}
	return b.String()
}

func caps(terminal string, mux termcap.Mux, notify termcap.Support) termcap.Caps {
	var c termcap.Caps
	c.Terminal.Set(terminal, termcap.Queried)
	c.Mux.Set(mux, termcap.Environment)
	c.DesktopNotify.Set(notify, termcap.Queried)
	return c
}

// notifier is an Always notifier that has seen c.
func notifier(c termcap.Caps, o ...NotifyOption) *Notifier {
	n := NewNotifier(append([]NotifyOption{WithPolicy(Always)}, o...)...)
	n.Update(termcap.CapsMsg{Caps: c})
	return n
}

func TestNotificationBytes(t *testing.T) {
	kitty := caps("kitty", termcap.NoMux, termcap.Supported)
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
		if got := raw(notifier(kitty, WithProtocol(c.p)).Notify(c.x)); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestAutoPicksTheProtocol(t *testing.T) {
	cases := []struct {
		c    termcap.Caps
		want Protocol
	}{
		{caps("kitty(0.39.1)", termcap.NoMux, termcap.Supported), OSC99},
		{caps("ghostty 1.1.0", termcap.NoMux, termcap.Unsupported), OSC777},
		{caps("xterm-ghostty", termcap.NoMux, termcap.Unknown), OSC777},
		{caps("foot", termcap.NoMux, termcap.Unsupported), OSC777},
		{caps("iTerm.app", termcap.NoMux, termcap.Unsupported), OSC9},
		{caps("WezTerm 20240203", termcap.NoMux, termcap.Unsupported), OSC9},
		{caps("WarpTerminal", termcap.NoMux, termcap.Unsupported), OSC9},
		{caps("xterm-256color", termcap.NoMux, termcap.Unsupported), Bell},
		{caps("tmux 3.4", termcap.Tmux, termcap.Unsupported), Bell},
		{termcap.Caps{}, Bell},
	}
	for _, c := range cases {
		if got := notifier(c.c).chosen(); got != c.want {
			t.Errorf("%q, OSC 99 %v: Auto chose %d, want %d", c.c.Terminal.Value, c.c.DesktopNotify.Value, got, c.want)
		}
	}
}

func TestNotificationsAreWrappedInsideAMultiplexer(t *testing.T) {
	x := Notification{Title: "Done", ID: "a"}
	inner := "\x1b]99;i=a;Done\x07"
	tmux := caps("kitty", termcap.Tmux, termcap.Supported)
	if got := raw(notifier(tmux).Notify(x)); got != ansi.TmuxPassthrough(inner) {
		t.Errorf("tmux: %q, want the tmux passthrough of %q", got, inner)
	}
	screen := caps("kitty", termcap.Screen, termcap.Supported)
	if got := raw(notifier(screen).Notify(x)); got != ansi.ScreenPassthrough(inner, screenLimit) {
		t.Errorf("screen: %q, want the screen passthrough of %q", got, inner)
	}
	if got := raw(notifier(tmux, WithProtocol(Bell)).Notify(x)); got != "\a" {
		t.Errorf("tmux bell: %q, want a bare BEL, which tmux passes on", got)
	}
}

func TestWhenUnfocused(t *testing.T) {
	x := Notification{Title: "Done"}
	n := NewNotifier(WithProtocol(Bell))
	if n.Notify(x) != nil {
		t.Error("sent with no focus report")
	}
	n.Update(tea.FocusMsg{})
	if n.Notify(x) != nil {
		t.Error("sent while focused")
	}
	n.Update(tea.BlurMsg{})
	if raw(n.Notify(x)) != "\a" {
		t.Error("not sent while unfocused")
	}
	n.Update(tea.FocusMsg{})
	if n.Notify(x) != nil {
		t.Error("sent after focus came back")
	}
	if raw(NewNotifier(WithProtocol(Bell), WithPolicy(Always)).Notify(x)) != "\a" {
		t.Error("Always did not send with no focus report")
	}
	never := NewNotifier(WithProtocol(Bell), WithPolicy(Never))
	never.Update(tea.BlurMsg{})
	if never.Notify(x) != nil {
		t.Error("Never sent")
	}
}

func TestEmptyNotificationIsNotSent(t *testing.T) {
	n := notifier(termcap.Caps{}, WithProtocol(Bell))
	if n.Notify(Notification{Title: "\x1b\x07", Body: "\x9b"}) != nil {
		t.Fatal("a notification of control bytes only was sent")
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
	if msgs := run(notifier(termcap.Caps{}, WithBackend(b)).Notify(Notification{Title: "Do\x1bne", Body: "b"})); len(msgs) != 1 || msgs[0] != nil {
		t.Errorf("a backend's success sent %v, want nothing", msgs)
	}
	if len(b.got) != 1 || b.got[0].Title != "Done" {
		t.Errorf("backend got %+v, want the cleaned notification", b.got)
	}
	failing := &backend{err: errors.New("no desktop")}
	msgs := run(notifier(termcap.Caps{}, WithBackend(failing)).Notify(Notification{Title: "x"}))
	if len(msgs) != 1 {
		t.Fatalf("got %v", msgs)
	}
	if m, ok := msgs[0].(NotifyErrorMsg); !ok || !errors.Is(m.Err, failing.err) || m.Notification.Title != "x" {
		t.Errorf("a backend's error came back as %#v, want a NotifyErrorMsg", msgs[0])
	}
}

func TestControlBytesAreStripped(t *testing.T) {
	evil := "a\x1b]52;c;aGk=\x07b\x9bc\x00d\x7fe"
	got := raw(notifier(termcap.Caps{}, WithProtocol(OSC9)).Notify(Notification{Title: evil, Body: evil}))
	if want := "\x1b]9;a]52;c;aGk=bcde: a]52;c;aGk=bcde\x07"; got != want {
		t.Errorf("notification %q, want %q", got, want)
	}
	if got := strip("one\ntwo\tthree\r"); got != "one two three " {
		t.Errorf("strip turned whitespace into %q", got)
	}
	// C1 controls encoded as UTF-8, CSI and NEL among them, as well as the
	// raw 8-bit byte above.
	if got := strip("a\u009b31mb\u0085c\u0080d\u009fe"); got != "a31mbcde" {
		t.Errorf("strip left UTF-8 C1 controls: %q", got)
	}
	link, err := Link("https://example.com/\x1b]8;;evil\x07x", "te\x1bxt\x9b", "id=\x07a")
	if err != nil {
		t.Fatal(err)
	}
	if want := "\x1b]8;id=a;https://example.com/]8;;evilx\x07text\x1b]8;;\x07"; link != want {
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

func TestCopy(t *testing.T) {
	setClipboard := reflect.TypeOf(tea.SetClipboard("x")())
	count := func(msgs []tea.Msg) (n int) {
		for _, m := range msgs {
			if reflect.TypeOf(m) == setClipboard {
				n++
			}
		}
		return n
	}
	local := run(Copy(termcap.Caps{}, "hello"))
	if count(local) != 1 || len(local) != 1 {
		t.Errorf("outside tmux, Copy sent %v, want tea's OSC 52 alone", local)
	}
	var tmux termcap.Caps
	tmux.Mux.Set(termcap.Tmux, termcap.Environment)
	msgs := run(Copy(tmux, "hello"))
	if count(msgs) != 1 {
		t.Errorf("inside tmux, Copy sent %v, want tea's OSC 52", msgs)
	}
	if got, want := raw(Copy(tmux, "hello")), ansi.TmuxPassthrough(ansi.SetSystemClipboard("hello")); got != want {
		t.Errorf("inside tmux, the extra copy is %q, want %q", got, want)
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
