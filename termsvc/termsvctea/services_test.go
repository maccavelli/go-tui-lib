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

	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termsvc"
)

type clipboard struct {
	got []string
	err error
}

func (c *clipboard) Copy(_ context.Context, text string) error {
	c.got = append(c.got, text)
	return c.err
}

// copiedOf runs cmd, and returns its CopiedMsg and whether it wrote to the
// clipboard through tea or tea.Raw.
func copiedOf(t *testing.T, cmd tea.Cmd) (termsvc.CopiedMsg, bool) {
	t.Helper()
	setClipboard := reflect.TypeOf(tea.SetClipboard("x")())
	var got []termsvc.CopiedMsg
	wrote := false
	for _, m := range run(cmd) {
		switch v := m.(type) {
		case termsvc.CopiedMsg:
			got = append(got, v)
		case tea.RawMsg:
			wrote = true
		default:
			if reflect.TypeOf(m) == setClipboard {
				wrote = true
			}
		}
	}
	if len(got) != 1 {
		t.Fatalf("%d CopiedMsg, want 1", len(got))
	}
	return got[0], wrote
}

func TestCopyStatus(t *testing.T) {
	var tmux termcap.Caps
	tmux.Mux.Set(termcap.Tmux, termcap.Environment)
	cases := []struct {
		name  string
		cmd   tea.Cmd
		want  termsvc.CopiedMsg
		wrote bool
	}{
		{"OSC 52", Copy(termcap.Caps{}, "x"), termsvc.CopiedMsg{Status: termsvc.Unconfirmed, Route: termsvc.RouteOSC52}, true},
		{"OSC 52 inside tmux", Copy(tmux, "x"), termsvc.CopiedMsg{Status: termsvc.Unconfirmed, Route: termsvc.RouteOSC52Tmux}, true},
		{"the largest payload", Copy(termcap.Caps{}, strings.Repeat("x", termsvc.MaxCopyBytes)), termsvc.CopiedMsg{Status: termsvc.Unconfirmed, Route: termsvc.RouteOSC52}, true},
		{"one byte over", Copy(termcap.Caps{}, strings.Repeat("x", termsvc.MaxCopyBytes+1)), termsvc.CopiedMsg{Status: termsvc.Failed}, false},
	}
	for _, c := range cases {
		got, wrote := copiedOf(t, c.cmd)
		if got != c.want || wrote != c.wrote {
			t.Errorf("%s: %+v, wrote %v; want %+v, wrote %v", c.name, got, wrote, c.want, c.wrote)
		}
	}

	ok := &clipboard{}
	got, wrote := copiedOf(t, Copy(tmux, "hello", termsvc.WithClipboard(ok)))
	if got != (termsvc.CopiedMsg{Status: termsvc.Confirmed, Route: termsvc.RouteBackend}) || wrote || len(ok.got) != 1 || ok.got[0] != "hello" {
		t.Errorf("a clipboard's success: %+v, wrote %v, clipboard got %q", got, wrote, ok.got)
	}
	bad := &clipboard{err: errors.New("no display")}
	if got, _ := copiedOf(t, Copy(termcap.Caps{}, "x", termsvc.WithClipboard(bad))); got.Status != termsvc.Failed || got.Route != termsvc.RouteBackend || !errors.Is(got.Err, bad.err) {
		t.Errorf("a clipboard's error: %+v", got)
	}
	if got, _ := copiedOf(t, Copy(termcap.Caps{}, strings.Repeat("x", termsvc.MaxCopyBytes+1), termsvc.WithClipboard(ok))); got.Status != termsvc.Failed || len(ok.got) != 1 {
		t.Errorf("an oversized copy reached the clipboard: %+v, %d calls", got, len(ok.got))
	}
}

func TestActivityBeacon(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()
		msg := ActivityBeacon()()
		if _, ok := msg.(termsvc.ActivityTickMsg); !ok || time.Since(began) != termsvc.ActivityInterval {
			t.Errorf("tick %T after %v, want ActivityTickMsg after %v", msg, time.Since(began), termsvc.ActivityInterval)
		}
	})
}
