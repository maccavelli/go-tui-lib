package termsvctea

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

	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termsvc"
)

// The byte forms and the context forms
// (docs/decisions/0014-PLAN-component-native-forms.md Step 9), moved from
// termsvc with the commands (0020-PLAN Step 9): the commands write what
// termsvc's Sequence and CopySequence give.

func TestSequenceMatchesNotify(t *testing.T) {
	kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
	tmux := caps(termcap.BrandUnknown, termcap.Tmux, termcap.Unsupported)
	xs := []termsvc.Notification{
		{Title: "build", Body: "passed"},
		{Title: "only a title"},
		{Body: "only a body", Urgency: termsvc.Critical},
		{Title: "with id", ID: "job-7"},
		{Title: "esc \x1b[31mred\x1b[m\nline", Body: strings.Repeat("long ", 80)},
	}
	for _, c := range []termcap.Caps{kitty, tmux} {
		for _, p := range []termsvc.Protocol{termsvc.Auto, termsvc.OSC99, termsvc.OSC777, termsvc.OSC9, termsvc.Bell} {
			byNotify, bySequence := notifier(c, termsvc.WithProtocol(p)), notifier(c, termsvc.WithProtocol(p))
			for _, x := range xs {
				want := raw(Notify(byNotify, x))
				got, skip := bySequence.Sequence(x)
				if got != want || skip != termsvc.NotSkipped {
					t.Errorf("protocol %d, %+v:\n Sequence %q, %v\n   Notify %q", p, x, got, skip, want)
				}
			}
		}
	}
	// One numbering: each notification without an ID takes the next number,
	// whether Notify or Sequence encodes it.
	n := notifier(kitty, termsvc.WithProtocol(termsvc.OSC99))
	first, _ := n.Sequence(termsvc.Notification{Title: "a"})
	second := raw(Notify(n, termsvc.Notification{Title: "b"}))
	third, _ := n.Sequence(termsvc.Notification{Title: "c"})
	for i, s := range []string{first, second, third} {
		if id := "i=n" + string(rune('1'+i)); !strings.Contains(s, id) {
			t.Errorf("notification %d: %q has no %s", i+1, s, id)
		}
	}
	// The reasons, and WithBackend ignored.
	for _, tc := range []struct {
		n    *termsvc.Notifier
		x    termsvc.Notification
		want termsvc.SkipReason
	}{
		{termsvc.NewNotifier(), termsvc.Notification{Title: "t"}, termsvc.SkipFocusUnknown},
		{notifier(kitty, termsvc.WithProtocol(termsvc.Off)), termsvc.Notification{Title: "t"}, termsvc.SkipDisabled},
		{notifier(kitty, termsvc.WithPolicy(termsvc.Never)), termsvc.Notification{Title: "t"}, termsvc.SkipDisabled},
		{notifier(kitty), termsvc.Notification{Title: "\x1b[m"}, termsvc.SkipEmpty},
		{notifier(kitty, termsvc.WithNotifyFilter(func(termsvc.Notification) bool { return false })), termsvc.Notification{Title: "t"}, termsvc.SkipGated},
	} {
		if got, skip := tc.n.Sequence(tc.x); got != "" || skip != tc.want {
			t.Errorf("%+v: %q, %v; want nothing, %v", tc.x, got, skip, tc.want)
		}
	}
	b := &backend{}
	if got, skip := notifier(kitty, termsvc.WithBackend(b), termsvc.WithProtocol(termsvc.OSC9)).Sequence(termsvc.Notification{Title: "t"}); got != ansi.Notify("t") || skip != termsvc.NotSkipped || len(b.got) != 0 {
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
		route termsvc.Route
	}{{plain, termsvc.RouteOSC52}, {tmux, termsvc.RouteOSC52Tmux}} {
		for _, text := range []string{"hello", "", strings.Repeat("x", termsvc.MaxCopyBytes)} {
			cmd := Copy(tc.c, text)
			// tea writes its clipboard message as ansi.SetSystemClipboard.
			want := ansi.SetSystemClipboard(clipboardText(cmd)) + raw(cmd)
			got, route, err := termsvc.CopySequence(tc.c, text)
			if got != want || route != tc.route || err != nil {
				t.Errorf("mux %v, %d bytes: %q, %v, %v; want %q, %v", tc.c.Mux.Value, len(text), got, route, err, want, tc.route)
			}
			if m, _ := copiedOf(t, Copy(tc.c, text)); m.Route != route {
				t.Errorf("Copy's route %v, CopySequence's %v", m.Route, route)
			}
		}
	}
	got, route, err := termsvc.CopySequence(plain, strings.Repeat("x", termsvc.MaxCopyBytes+1))
	if got != "" || route != termsvc.RouteNone || !errors.Is(err, termsvc.ErrCopyTooLarge) {
		t.Errorf("one byte over: %q, %v, %v", got, route, err)
	}
}

// recorder is a backend and a clipboard that keep their context's deadline.
type recorder struct {
	deadline time.Time
	has      bool
}

func (r *recorder) Notify(ctx context.Context, _ termsvc.Notification) error {
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
			var no []termsvc.NotifyOption
			var co []termsvc.CopyOption
			if tc.opt != 0 {
				no, co = []termsvc.NotifyOption{termsvc.WithBackendTimeout(tc.opt)}, []termsvc.CopyOption{termsvc.WithCopyTimeout(tc.opt)}
			}
			n, c := &recorder{}, &recorder{}
			run(NotifyContext(ctx, notifier(kitty, append(no, termsvc.WithBackend(n))...), termsvc.Notification{Title: "t"}))
			run(CopyContext(ctx, kitty, "x", append(co, termsvc.WithClipboard(c))...))
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
		run(Notify(notifier(kitty, termsvc.WithBackend(n)), termsvc.Notification{Title: "t"}))
		run(Copy(kitty, "x", termsvc.WithClipboard(c)))
		if n.deadline.Sub(start) != 5*time.Second || c.deadline.Sub(start) != 5*time.Second {
			t.Errorf("Notify and Copy: deadlines %v and %v, want 5s", n.deadline.Sub(start), c.deadline.Sub(start))
		}
	})
}

// blocker is a backend and a clipboard that wait for their context to end.
type blocker struct{}

func (blocker) Notify(ctx context.Context, _ termsvc.Notification) error {
	<-ctx.Done()
	return ctx.Err()
}
func (blocker) Copy(ctx context.Context, _ string) error { <-ctx.Done(); return ctx.Err() }

func TestBackendTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
		start := time.Now()
		r := sent(t, Notify(notifier(kitty, termsvc.WithBackend(blocker{}), termsvc.WithBackendTimeout(20*time.Millisecond)), termsvc.Notification{Title: "t"}))
		if r.Sent || r.Skipped != termsvc.SkipFailed || !errors.Is(r.Err, context.DeadlineExceeded) || time.Since(start) != 20*time.Millisecond {
			t.Errorf("Notify: %+v after %v", r, time.Since(start))
		}
		m, _ := copiedOf(t, Copy(kitty, "x", termsvc.WithClipboard(blocker{}), termsvc.WithCopyTimeout(30*time.Millisecond)))
		if m.Status != termsvc.Failed || m.Route != termsvc.RouteBackend || !errors.Is(m.Err, context.DeadlineExceeded) {
			t.Errorf("Copy: %+v", m)
		}
		// A caller's cancel ends it too.
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		r = sent(t, NotifyContext(ctx, notifier(kitty, termsvc.WithBackend(blocker{})), termsvc.Notification{Title: "t"}))
		if r.Skipped != termsvc.SkipFailed || !errors.Is(r.Err, context.Canceled) {
			t.Errorf("NotifyContext, cancelled: %+v", r)
		}
	})
}
