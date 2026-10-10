package termsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// TestBackendAndClipboardFuncs: a function used as a Backend or a
// Clipboard is called with what Notify and Copy send, and its error is
// reported (docs/decisions/0014-PLAN-canonicalization.md Step 7).
func TestBackendAndClipboardFuncs(t *testing.T) {
	kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Supported)
	var sent []Notification
	backend := BackendFunc(func(_ context.Context, x Notification) error {
		sent = append(sent, x)
		return nil
	})
	msgs := run(notifier(kitty, WithBackend(backend)).Notify(Notification{Title: "t", Body: "b"}))
	if len(sent) != 1 || sent[0].Title != "t" || sent[0].Body != "b" {
		t.Errorf("the backend was sent %+v", sent)
	}
	if len(msgs) != 1 || !msgs[0].(NotifyResultMsg).Sent {
		t.Errorf("Notify reported %+v", msgs)
	}

	refused := errors.New("no clipboard")
	var copied []string
	clip := ClipboardFunc(func(_ context.Context, text string) error {
		copied = append(copied, text)
		return refused
	})
	msgs = run(Copy(kitty, "x", WithClipboard(clip)))
	if len(copied) != 1 || copied[0] != "x" {
		t.Errorf("the clipboard was given %q", copied)
	}
	if m, ok := msgs[len(msgs)-1].(CopiedMsg); !ok || m.Status != Failed || !errors.Is(m.Err, refused) {
		t.Errorf("Copy reported %+v", msgs)
	}
}
