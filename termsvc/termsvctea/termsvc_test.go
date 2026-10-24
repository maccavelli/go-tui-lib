package termsvctea

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termsvc"
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

// caps is a terminal of brand inside mux, whose answer to the OSC 99 query
// was notify.
func caps(brand termcap.Brand, mux termcap.Mux, notify termcap.Support) termcap.Caps {
	var c termcap.Caps
	c.Brand.Set(brand, termcap.Environment)
	c.Mux.Set(mux, termcap.Environment)
	c.DesktopNotify.Set(notify, termcap.Queried)
	return c
}

// sent is the NotifyResultMsg among msgs.
func sent(t *testing.T, cmd tea.Cmd) termsvc.NotifyResultMsg {
	t.Helper()
	var got []termsvc.NotifyResultMsg
	for _, m := range run(cmd) {
		if r, ok := m.(termsvc.NotifyResultMsg); ok {
			got = append(got, r)
		}
	}
	if len(got) != 1 {
		t.Fatalf("%d NotifyResultMsg, want 1", len(got))
	}
	return got[0]
}

// notifier is an Always notifier that has seen c.
func notifier(c termcap.Caps, o ...termsvc.NotifyOption) *termsvc.Notifier {
	n := termsvc.NewNotifier(append([]termsvc.NotifyOption{termsvc.WithPolicy(termsvc.Always)}, o...)...)
	Observe(n, termcap.CapsMsg{Caps: c})
	return n
}

func TestNotificationBytes(t *testing.T) {
	kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
	cases := []struct {
		name string
		p    termsvc.Protocol
		x    termsvc.Notification
		want string
	}{
		{"OSC 99 title and body", termsvc.OSC99, termsvc.Notification{Title: "Done", Body: "3 files", ID: "build"},
			"\x1b]99;i=build:d=0;Done\x07\x1b]99;i=build:p=body;3 files\x07"},
		{"OSC 99 title only, critical", termsvc.OSC99, termsvc.Notification{Title: "Failed", ID: "t", Urgency: termsvc.Critical},
			"\x1b]99;i=t:u=2;Failed\x07"},
		{"OSC 99 body only, low", termsvc.OSC99, termsvc.Notification{Body: "fyi", ID: "t", Urgency: termsvc.Low},
			"\x1b]99;i=t:u=0:p=body;fyi\x07"},
		{"OSC 99 numbers a notification with no ID", termsvc.OSC99, termsvc.Notification{Title: "x"}, "\x1b]99;i=n1;x\x07"},
		{"OSC 99 keeps only the ID's allowed characters", termsvc.OSC99, termsvc.Notification{Title: "x", ID: "a:b;c=d"}, "\x1b]99;i=abcd;x\x07"},
		{"OSC 777", termsvc.OSC777, termsvc.Notification{Title: "Done; ok", Body: "a;b"}, "\x1b]777;notify;Done, ok;a,b\x07"},
		{"OSC 9", termsvc.OSC9, termsvc.Notification{Title: "Done", Body: "3 files"}, "\x1b]9;Done: 3 files\x07"},
		{"OSC 9 never reads as a sub-command", termsvc.OSC9, termsvc.Notification{Title: "4;1;50"}, "\x1b]9; 4;1;50\x07"},
		{"bell", termsvc.Bell, termsvc.Notification{Title: "Done"}, "\a"},
		{"off", termsvc.Off, termsvc.Notification{Title: "Done"}, ""},
	}
	for _, c := range cases {
		if got := raw(Notify(notifier(kitty, termsvc.WithProtocol(c.p)), c.x)); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

// TestFocusPolicies drives each policy through unknown, focused, blurred
// and focused-again, and checks what was sent and why not (MADR A1, Q6).
func TestFocusPolicies(t *testing.T) {
	x := termsvc.Notification{Title: "Done"}
	steps := []struct {
		name string
		msg  tea.Msg
	}{{"unknown", nil}, {"focused", tea.FocusMsg{}}, {"blurred", tea.BlurMsg{}}, {"focused again", tea.FocusMsg{}}}
	want := map[termsvc.Policy][]termsvc.SkipReason{
		termsvc.WhenUnfocused: {termsvc.SkipFocusUnknown, termsvc.SkipFocused, termsvc.NotSkipped, termsvc.SkipFocused},
		termsvc.UnlessFocused: {termsvc.NotSkipped, termsvc.SkipFocused, termsvc.NotSkipped, termsvc.SkipFocused},
		termsvc.Always:        {termsvc.NotSkipped, termsvc.NotSkipped, termsvc.NotSkipped, termsvc.NotSkipped},
		termsvc.Never:         {termsvc.SkipDisabled, termsvc.SkipDisabled, termsvc.SkipDisabled, termsvc.SkipDisabled},
	}
	for policy, reasons := range want {
		n := termsvc.NewNotifier(termsvc.WithProtocol(termsvc.Bell), termsvc.WithPolicy(policy))
		for i, s := range steps {
			if s.msg != nil {
				Observe(n, s.msg)
			}
			cmd := Notify(n, x)
			r := sent(t, cmd)
			if r.Skipped != reasons[i] || r.Sent != (reasons[i] == termsvc.NotSkipped) {
				t.Errorf("policy %d, %s: Sent %v, Skipped %q; want skipped %q", policy, s.name, r.Sent, r.Skipped, reasons[i])
			}
			if wrote := raw(cmd); (wrote == "\a") != r.Sent {
				t.Errorf("policy %d, %s: wrote %q with Sent %v", policy, s.name, wrote, r.Sent)
			}
		}
	}
}

func TestEmptyNotificationIsNotSent(t *testing.T) {
	n := notifier(termcap.Caps{}, termsvc.WithProtocol(termsvc.Bell))
	cmd := Notify(n, termsvc.Notification{Title: "\x1b\x07", Body: "\x9b\x1b[31m\n"})
	if r := sent(t, cmd); r.Sent || r.Skipped != termsvc.SkipEmpty {
		t.Fatalf("a notification of controls only: %+v", r)
	}
	if raw(cmd) != "" {
		t.Fatal("a notification of controls only wrote something")
	}
	if r := sent(t, Notify(notifier(termcap.Caps{}, termsvc.WithProtocol(termsvc.Off)), termsvc.Notification{Title: "x"})); r.Skipped != termsvc.SkipDisabled {
		t.Fatalf("protocol Off: %+v", r)
	}
}

type backend struct {
	got []termsvc.Notification
	err error
}

func (b *backend) Notify(_ context.Context, x termsvc.Notification) error {
	b.got = append(b.got, x)
	return b.err
}

func TestBackend(t *testing.T) {
	b := &backend{}
	msgs := run(Notify(notifier(termcap.Caps{}, termsvc.WithBackend(b)), termsvc.Notification{Title: "Do\x1b[1mne", Body: "b"}))
	if len(msgs) != 1 {
		t.Fatalf("a backend's notification sent %v, want its result alone, nothing written", msgs)
	}
	if r, ok := msgs[0].(termsvc.NotifyResultMsg); !ok || !r.Sent || r.Skipped != termsvc.NotSkipped {
		t.Errorf("a backend's success: %+v", msgs[0])
	}
	if len(b.got) != 1 || b.got[0].Title != "Done" {
		t.Errorf("backend got %+v, want the cleaned notification", b.got)
	}
	failing := &backend{err: errors.New("no desktop")}
	r := sent(t, Notify(notifier(termcap.Caps{}, termsvc.WithBackend(failing)), termsvc.Notification{Title: "x"}))
	if r.Sent || r.Skipped != termsvc.SkipFailed || !errors.Is(r.Err, failing.err) || r.Notification.Title != "x" {
		t.Errorf("a backend's error came back as %+v", r)
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
	if count(local) != 1 || len(local) != 2 {
		t.Errorf("outside tmux, Copy sent %v, want tea's OSC 52 and its CopiedMsg", local)
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
