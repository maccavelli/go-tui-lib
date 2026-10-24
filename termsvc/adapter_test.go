package termsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// TestBackendAndClipboardFuncs: a function used as a Backend or a
// Clipboard is called with what a notification and a copy send, and its
// error is reported (docs/decisions/0014-PLAN-canonicalization.md Step 7).
func TestBackendAndClipboardFuncs(t *testing.T) {
	kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
	var sent []Notification
	backend := BackendFunc(func(_ context.Context, x Notification) error {
		sent = append(sent, x)
		return nil
	})
	seq, r := prepared(notifier(kitty, WithBackend(backend)), Notification{Title: "t", Body: "b"})
	if len(sent) != 1 || sent[0].Title != "t" || sent[0].Body != "b" {
		t.Errorf("the backend was sent %+v", sent)
	}
	if seq != "" || !r.Sent {
		t.Errorf("Prepare gave %q, %+v", seq, r)
	}

	refused := errors.New("no clipboard")
	var copied []string
	clip := ClipboardFunc(func(_ context.Context, text string) error {
		copied = append(copied, text)
		return refused
	})
	route, deliver := PrepareCopy(context.Background(), kitty, "x", WithClipboard(clip))
	m := deliver()
	if len(copied) != 1 || copied[0] != "x" {
		t.Errorf("the clipboard was given %q", copied)
	}
	if route != RouteBackend || m.Status != Failed || !errors.Is(m.Err, refused) {
		t.Errorf("PrepareCopy reported %v, %+v", route, m)
	}
}
