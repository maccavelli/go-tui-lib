package termsvctea

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termsvc"
)

// TestBackendOffTheLoop: NotifyContext returns its command without calling
// the backend, and the command delivers Sent once the backend, which waits
// for the test, returns. So a slow backend never holds the program's loop.
func TestBackendOffTheLoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var calls atomic.Int32
		b := termsvc.BackendFunc(func(context.Context, termsvc.Notification) error {
			calls.Add(1)
			<-release
			return nil
		})
		n := notifier(termcap.Caps{}, termsvc.WithBackend(b))
		cmds := make(chan tea.Cmd, 1)
		go func() { cmds <- NotifyContext(t.Context(), n, termsvc.Notification{Title: "t"}) }()
		synctest.Wait()
		var cmd tea.Cmd
		select {
		case cmd = <-cmds:
		default:
			close(release)
			t.Fatal("NotifyContext waited for the backend")
		}
		if c := calls.Load(); c != 0 {
			close(release)
			t.Fatalf("the backend ran %d times before the command did", c)
		}
		msgs := make(chan tea.Msg, 1)
		go func() { msgs <- cmd() }()
		synctest.Wait()
		if calls.Load() != 1 || len(msgs) != 0 {
			close(release)
			t.Fatalf("the command: %d calls, %d results before the backend returned", calls.Load(), len(msgs))
		}
		close(release)
		if r, ok := (<-msgs).(termsvc.NotifyResultMsg); !ok || !r.Sent {
			t.Errorf("the command delivered %+v, want Sent", r)
		}
	})
}

// TestNotifyWritesThenReports: through the terminal, the bytes are written
// before the result is delivered.
func TestNotifyWritesThenReports(t *testing.T) {
	msgs := run(Notify(notifier(termcap.Caps{}, termsvc.WithProtocol(termsvc.Bell)), termsvc.Notification{Title: "t"}))
	if len(msgs) != 2 {
		t.Fatalf("Notify sent %v, want the write, then the result", msgs)
	}
	if w, ok := msgs[0].(tea.RawMsg); !ok || w.Msg != "\a" {
		t.Errorf("first %+v, want the bell", msgs[0])
	}
	if r, ok := msgs[1].(termsvc.NotifyResultMsg); !ok || !r.Sent {
		t.Errorf("then %+v, want Sent", msgs[1])
	}
}

// TestObserve: termcap.CapsMsg, tea.FocusMsg and tea.BlurMsg reach the
// notifier, and nothing else changes it.
func TestObserve(t *testing.T) {
	n := termsvc.NewNotifier(termsvc.WithProtocol(termsvc.Bell))
	x := termsvc.Notification{Title: "t"}
	for _, s := range []struct {
		name string
		msg  tea.Msg
		want termsvc.SkipReason
	}{
		{"nothing seen", nil, termsvc.SkipFocusUnknown},
		{"a key", tea.KeyPressMsg{Code: 'a'}, termsvc.SkipFocusUnknown},
		{"focus", tea.FocusMsg{}, termsvc.SkipFocused},
		{"a resize", tea.WindowSizeMsg{Width: 80, Height: 24}, termsvc.SkipFocused},
		{"blur", tea.BlurMsg{}, termsvc.NotSkipped},
	} {
		if s.msg != nil {
			Observe(n, s.msg)
		}
		if _, skip := n.Sequence(x); skip != s.want {
			t.Errorf("after %s: %v, want %v", s.name, skip, s.want)
		}
	}
	auto := termsvc.NewNotifier(termsvc.WithPolicy(termsvc.Always))
	if seq, _ := auto.Sequence(x); seq != "\a" {
		t.Fatalf("Auto with no caps: %q, want the bell", seq)
	}
	var kitty termcap.Caps
	kitty.Brand.Set(termcap.BrandKitty, termcap.Environment)
	kitty.DesktopNotify.Set(termcap.Supported, termcap.Queried)
	Observe(auto, termcap.CapsMsg{Caps: kitty})
	if seq, _ := auto.Sequence(x); !strings.HasPrefix(seq, "\x1b]99;") {
		t.Errorf("Auto after CapsMsg: %q, want OSC 99", seq)
	}
}

// TestOpen: the command delivers OpenURLMsg for a URL the policy opens,
// and a refused URL gets no command (the Open half of termsvc's
// TestOpenable).
func TestOpen(t *testing.T) {
	var p termsvc.LinkPolicy
	if msgs := run(Open(p, "https://example.com")); len(msgs) != 1 || msgs[0] != (termsvc.OpenURLMsg{URL: "https://example.com"}) {
		t.Errorf("Open sent %v", msgs)
	}
	if Open(p, "javascript:x") != nil {
		t.Error("Open returned a command for a refused URL")
	}
}
