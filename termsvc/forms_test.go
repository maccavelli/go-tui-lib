package termsvc

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// The byte forms and the context forms
// (docs/decisions/0014-PLAN-component-native-forms.md Step 9).

func TestSequenceMatchesNotify(t *testing.T) {
	kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
	tmux := caps(termcap.BrandUnknown, termcap.Tmux, termcap.Unsupported)
	xs := []Notification{
		{Title: "build", Body: "passed"},
		{Title: "only a title"},
		{Body: "only a body", Urgency: Critical},
		{Title: "with id", ID: "job-7"},
		{Title: "esc \x1b[31mred\x1b[m\nline", Body: strings.Repeat("long ", 80)},
	}
	for _, c := range []termcap.Caps{kitty, tmux} {
		for _, p := range []Protocol{Auto, OSC99, OSC777, OSC9, Bell} {
			byNotify, bySequence := notifier(c, WithProtocol(p)), notifier(c, WithProtocol(p))
			for _, x := range xs {
				want := raw(byNotify.Notify(x))
				got, skip := bySequence.Sequence(x)
				if got != want || skip != NotSkipped {
					t.Errorf("protocol %d, %+v:\n Sequence %q, %v\n   Notify %q", p, x, got, skip, want)
				}
			}
		}
	}
	// One numbering: each notification without an ID takes the next number,
	// whether Notify or Sequence encodes it.
	n := notifier(kitty, WithProtocol(OSC99))
	first, _ := n.Sequence(Notification{Title: "a"})
	second := raw(n.Notify(Notification{Title: "b"}))
	third, _ := n.Sequence(Notification{Title: "c"})
	for i, s := range []string{first, second, third} {
		if id := "i=n" + string(rune('1'+i)); !strings.Contains(s, id) {
			t.Errorf("notification %d: %q has no %s", i+1, s, id)
		}
	}
	// The reasons, and WithBackend ignored.
	for _, tc := range []struct {
		n    *Notifier
		x    Notification
		want SkipReason
	}{
		{NewNotifier(), Notification{Title: "t"}, SkipFocusUnknown},
		{notifier(kitty, WithProtocol(Off)), Notification{Title: "t"}, SkipDisabled},
		{notifier(kitty, WithPolicy(Never)), Notification{Title: "t"}, SkipDisabled},
		{notifier(kitty), Notification{Title: "\x1b[m"}, SkipEmpty},
		{notifier(kitty, WithNotifyFilter(func(Notification) bool { return false })), Notification{Title: "t"}, SkipGated},
	} {
		if got, skip := tc.n.Sequence(tc.x); got != "" || skip != tc.want {
			t.Errorf("%+v: %q, %v; want nothing, %v", tc.x, got, skip, tc.want)
		}
	}
	b := &backend{}
	if got, skip := notifier(kitty, WithBackend(b), WithProtocol(OSC9)).Sequence(Notification{Title: "t"}); got != ansi.Notify("t") || skip != NotSkipped || len(b.got) != 0 {
		t.Errorf("with a backend: %q, %v, backend got %v", got, skip, b.got)
	}
}

// clipboardText is the text of the tea.SetClipboard message cmd makes.
func clipboardText(cmd tea.Cmd) string {
	setClipboard := reflect.TypeOf(tea.SetClipboard("x")())
	for _, m := range run(cmd) {
		if reflect.TypeOf(m) == setClipboard {
			return reflect.ValueOf(m).String()
		}
	}
	return ""
}

func TestCopySequence(t *testing.T) {
	plain := caps(termcap.BrandUnknown, termcap.NoMux, termcap.Unsupported)
	tmux := caps(termcap.BrandUnknown, termcap.Tmux, termcap.Unsupported)
	for _, tc := range []struct {
		c     termcap.Caps
		route Route
	}{{plain, RouteOSC52}, {tmux, RouteOSC52Tmux}} {
		for _, text := range []string{"hello", "", strings.Repeat("x", MaxCopyBytes)} {
			cmd := Copy(tc.c, text)
			// tea writes its clipboard message as ansi.SetSystemClipboard.
			want := ansi.SetSystemClipboard(clipboardText(cmd)) + raw(cmd)
			got, route, err := CopySequence(tc.c, text)
			if got != want || route != tc.route || err != nil {
				t.Errorf("mux %v, %d bytes: %q, %v, %v; want %q, %v", tc.c.Mux.Value, len(text), got, route, err, want, tc.route)
			}
			if m, _ := copiedOf(t, Copy(tc.c, text)); m.Route != route {
				t.Errorf("Copy's route %v, CopySequence's %v", m.Route, route)
			}
		}
	}
	got, route, err := CopySequence(plain, strings.Repeat("x", MaxCopyBytes+1))
	if got != "" || route != RouteNone || !errors.Is(err, ErrCopyTooLarge) {
		t.Errorf("one byte over: %q, %v, %v", got, route, err)
	}
}

// recorder is a backend and a clipboard that keep their context's deadline.
type recorder struct {
	deadline time.Time
	has      bool
}

func (r *recorder) Notify(ctx context.Context, _ Notification) error {
	r.deadline, r.has = ctx.Deadline()
	return nil
}

func (r *recorder) Copy(ctx context.Context, _ string) error {
	r.deadline, r.has = ctx.Deadline()
	return nil
}

func TestBackendDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
		start := time.Now()
		for _, tc := range []struct {
			name string
			opt  time.Duration // 0: the default
			ctx  time.Duration // 0: none on the caller's context
			want time.Duration // -1: no deadline
		}{
			{"the default", 0, 0, 5 * time.Second},
			{"a timeout of its own", time.Second, 0, time.Second},
			{"the caller's sooner deadline", time.Second, 200 * time.Millisecond, 200 * time.Millisecond},
			{"the timeout sooner than the caller's", time.Second, time.Minute, time.Second},
			{"no timeout, the caller's", -1, 3 * time.Second, 3 * time.Second},
			{"no timeout, no deadline", -1, 0, -1},
		} {
			ctx := t.Context()
			if tc.ctx > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tc.ctx)
				defer cancel()
			}
			var no []NotifyOption
			var co []CopyOption
			if tc.opt != 0 {
				no, co = []NotifyOption{WithBackendTimeout(tc.opt)}, []CopyOption{WithCopyTimeout(tc.opt)}
			}
			n, c := &recorder{}, &recorder{}
			run(notifier(kitty, append(no, WithBackend(n))...).NotifyContext(ctx, Notification{Title: "t"}))
			run(CopyContext(ctx, kitty, "x", append(co, WithClipboard(c))...))
			for what, r := range map[string]*recorder{"backend": n, "clipboard": c} {
				switch {
				case tc.want < 0 && r.has:
					t.Errorf("%s, %s: a deadline %v", tc.name, what, r.deadline.Sub(start))
				case tc.want >= 0 && (!r.has || r.deadline.Sub(start) != tc.want):
					t.Errorf("%s, %s: deadline %v (%v), want %v", tc.name, what, r.deadline.Sub(start), r.has, tc.want)
				}
			}
		}
		// Notify and Copy are the context forms under context.Background().
		n, c := &recorder{}, &recorder{}
		run(notifier(kitty, WithBackend(n)).Notify(Notification{Title: "t"}))
		run(Copy(kitty, "x", WithClipboard(c)))
		if n.deadline.Sub(start) != 5*time.Second || c.deadline.Sub(start) != 5*time.Second {
			t.Errorf("Notify and Copy: deadlines %v and %v, want 5s", n.deadline.Sub(start), c.deadline.Sub(start))
		}
	})
}

// blocker is a backend and a clipboard that wait for their context to end.
type blocker struct{}

func (blocker) Notify(ctx context.Context, _ Notification) error { <-ctx.Done(); return ctx.Err() }
func (blocker) Copy(ctx context.Context, _ string) error         { <-ctx.Done(); return ctx.Err() }

func TestBackendTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
		start := time.Now()
		r := sent(t, notifier(kitty, WithBackend(blocker{}), WithBackendTimeout(20*time.Millisecond)).Notify(Notification{Title: "t"}))
		if r.Sent || r.Skipped != SkipFailed || !errors.Is(r.Err, context.DeadlineExceeded) || time.Since(start) != 20*time.Millisecond {
			t.Errorf("Notify: %+v after %v", r, time.Since(start))
		}
		m, _ := copiedOf(t, Copy(kitty, "x", WithClipboard(blocker{}), WithCopyTimeout(30*time.Millisecond)))
		if m.Status != Failed || m.Route != RouteBackend || !errors.Is(m.Err, context.DeadlineExceeded) {
			t.Errorf("Copy: %+v", m)
		}
		// A caller's cancel ends it too.
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		r = sent(t, notifier(kitty, WithBackend(blocker{})).NotifyContext(ctx, Notification{Title: "t"}))
		if r.Skipped != SkipFailed || !errors.Is(r.Err, context.Canceled) {
			t.Errorf("NotifyContext, cancelled: %+v", r)
		}
	})
}
