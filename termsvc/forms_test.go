package termsvc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// The byte forms and the context forms
// (docs/decisions/0014-PLAN-component-native-forms.md Step 9), read through
// Prepare and PrepareCopy (docs/decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md
// Step 9). The pairing of the commands with Sequence and CopySequence is
// termsvctea's.

// TestPrepareMatchesSequence: for each protocol and policy, Prepare's bytes
// are Sequence's, and both take the next number from one count.
func TestPrepareMatchesSequence(t *testing.T) {
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
		for _, p := range []Protocol{Auto, OSC99, OSC777, OSC9, Bell, Off} {
			for _, policy := range []Policy{Always, WhenUnfocused, UnlessFocused, Never} {
				byPrepare, bySequence := notifier(c, WithProtocol(p), WithPolicy(policy)), notifier(c, WithProtocol(p), WithPolicy(policy))
				for _, x := range xs {
					got, r := prepared(byPrepare, x)
					want, skip := bySequence.Sequence(x)
					if got != want || r.Skipped != skip || r.Sent != (skip == NotSkipped) {
						t.Errorf("protocol %v, policy %v, %+v:\n Prepare %q, %+v\nSequence %q, %v", p, policy, x, got, r, want, skip)
					}
				}
			}
		}
	}
	// One numbering: each notification without an ID takes the next number,
	// whether Prepare or Sequence encodes it, once per call.
	n := notifier(kitty, WithProtocol(OSC99))
	first, _ := n.Sequence(Notification{Title: "a"})
	second := seqOf(n, Notification{Title: "b"})
	third, _ := n.Sequence(Notification{Title: "c"})
	fourth := seqOf(n, Notification{Title: "d"})
	for i, s := range []string{first, second, third, fourth} {
		if id := "i=n" + string(rune('1'+i)) + ";"; !strings.Contains(s, id) {
			t.Errorf("notification %d: %q has no %s", i+1, s, id)
		}
	}
	// The reasons, and WithBackend ignored by Sequence.
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
			_, deliver := notifier(kitty, append(no, WithBackend(n))...).Prepare(ctx, Notification{Title: "t"})
			deliver()
			_, copied := PrepareCopy(ctx, kitty, "x", append(co, WithClipboard(c))...)
			copied()
			for what, r := range map[string]*recorder{"backend": n, "clipboard": c} {
				switch {
				case tc.want < 0 && r.has:
					t.Errorf("%s, %s: a deadline %v", tc.name, what, r.deadline.Sub(start))
				case tc.want >= 0 && (!r.has || r.deadline.Sub(start) != tc.want):
					t.Errorf("%s, %s: deadline %v (%v), want %v", tc.name, what, r.deadline.Sub(start), r.has, tc.want)
				}
			}
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
		_, r := prepared(notifier(kitty, WithBackend(blocker{}), WithBackendTimeout(20*time.Millisecond)), Notification{Title: "t"})
		if r.Sent || r.Skipped != SkipFailed || !errors.Is(r.Err, context.DeadlineExceeded) || time.Since(start) != 20*time.Millisecond {
			t.Errorf("Prepare's deliver: %+v after %v", r, time.Since(start))
		}
		_, copied := PrepareCopy(context.Background(), kitty, "x", WithClipboard(blocker{}), WithCopyTimeout(30*time.Millisecond))
		if m := copied(); m.Status != Failed || m.Route != RouteBackend || !errors.Is(m.Err, context.DeadlineExceeded) {
			t.Errorf("PrepareCopy's deliver: %+v", m)
		}
		// A caller's cancel ends it too.
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, deliver := notifier(kitty, WithBackend(blocker{})).Prepare(ctx, Notification{Title: "t"})
		if r := deliver(); r.Skipped != SkipFailed || !errors.Is(r.Err, context.Canceled) {
			t.Errorf("Prepare, cancelled: %+v", r)
		}
	})
}
