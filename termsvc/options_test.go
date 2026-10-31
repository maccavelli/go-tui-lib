package termsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// The options and view facts of
// docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md Step 2
// (0024-MADR D5 and A3): each default as it was, and each option or fact
// changing its one behaviour.

func over(s termcap.Support) termcap.Fact[termcap.Support] {
	return termcap.Fact[termcap.Support]{Value: s, Origin: termcap.Override}
}

// at is a Caps of brand b inside mux.
func at(b termcap.Brand, mux termcap.Mux) termcap.Caps {
	var c termcap.Caps
	c.Brand.Set(b, termcap.Environment)
	c.Mux.Set(mux, termcap.Environment)
	return c
}

func TestPointerSupport(t *testing.T) {
	with := func(c termcap.Caps, f termcap.Fact[termcap.Support]) termcap.Caps { c.PointerSupport = f; return c }
	for _, tc := range []struct {
		name, shape, want string
		c                 termcap.Caps
	}{
		{"WezTerm, unset", "pointer", "", at(termcap.BrandWezTerm, termcap.NoMux)},
		{"WezTerm, supported", "pointer", "\x1b]22;pointer\x07", with(at(termcap.BrandWezTerm, termcap.NoMux), over(termcap.Supported))},
		{"kitty in tmux, supported", "pointer", "\x1b]22;pointer\x07", with(at(termcap.BrandKitty, termcap.Tmux), over(termcap.Supported))},
		{"Ghostty resets, supported", "", "\x1b]22;default\x07", with(at(termcap.BrandGhostty, termcap.NoMux), over(termcap.Supported))},
		{"kitty, unsupported", "pointer", "", with(at(termcap.BrandKitty, termcap.NoMux), over(termcap.Unsupported))},
		{"kitty, unknown", "pointer", "", with(at(termcap.BrandKitty, termcap.NoMux), over(termcap.Unknown))},
		{"kitty, a value not queried", "pointer", "\x1b]22;pointer\x07", with(at(termcap.BrandKitty, termcap.NoMux), termcap.Fact[termcap.Support]{Value: termcap.Unsupported})},
	} {
		if got := Pointer(tc.c, tc.shape); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestProgressSupport(t *testing.T) {
	with := func(c termcap.Caps, f termcap.Fact[termcap.Support]) termcap.Caps { c.ProgressSupport = f; return c }
	for _, tc := range []struct {
		name string
		c    termcap.Caps
		want bool
	}{
		{"kitty, unset", at(termcap.BrandKitty, termcap.NoMux), false},
		{"kitty, supported", with(at(termcap.BrandKitty, termcap.NoMux), over(termcap.Supported)), true},
		{"Ghostty, unsupported", with(at(termcap.BrandGhostty, termcap.NoMux), over(termcap.Unsupported)), false},
		{"Ghostty, a value not queried", with(at(termcap.BrandGhostty, termcap.NoMux), termcap.Fact[termcap.Support]{Value: termcap.Unsupported}), true},
	} {
		if got := ProgressSupported(tc.c); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestLinkDisplayFollowsLinksSupport(t *testing.T) {
	apple := at(termcap.BrandAppleTerminal, termcap.NoMux)
	if LinkDisplay(apple) != LabelAndURL {
		t.Fatal("Apple Terminal's links are not shown with their URL")
	}
	apple.LinksSupport = over(termcap.Supported)
	kitty := at(termcap.BrandKitty, termcap.NoMux)
	kitty.LinksSupport = over(termcap.Unsupported)
	if LinkDisplay(apple) != LabelOnly || LinkDisplay(kitty) != LabelAndURL {
		t.Errorf("Apple Terminal, supported: %v; kitty, unsupported: %v", LinkDisplay(apple), LinkDisplay(kitty))
	}
}

func TestNotifyFactsPickTheProtocol(t *testing.T) {
	x := Notification{Title: "Done", Body: "b"}
	foot := caps(termcap.BrandFoot, termcap.NoMux, termcap.Unsupported)
	if got, want := seqOf(notifier(foot), x), ansi.URxvtExt("notify", "Done", "b"); got != want {
		t.Fatalf("foot: %q, want OSC 777 %q", got, want)
	}
	foot.NotifyOSC9Support = over(termcap.Supported)
	if got, want := seqOf(notifier(foot), x), ansi.Notify(osc9Text(x)); got != want {
		t.Errorf("foot with OSC 9: %q, want %q", got, want)
	}
	kitty := caps(termcap.BrandKitty, termcap.NoMux, termcap.Unsupported)
	kitty.NotifyOSC777Support = over(termcap.Supported)
	if got, want := seqOf(notifier(kitty), x), ansi.URxvtExt("notify", "Done", "b"); got != want {
		t.Errorf("kitty with OSC 777: %q, want %q", got, want)
	}
}

func TestNotifierCells(t *testing.T) {
	long := Notification{Title: strings.Repeat("t", 100), Body: strings.Repeat("b", 300)}
	for _, tc := range []struct {
		name        string
		o           []NotifyOption
		title, body int
	}{
		{"default", nil, TitleCells, BodyCells},
		{"title 10", []NotifyOption{WithTitleCells(10)}, 10, BodyCells},
		{"body 20", []NotifyOption{WithBodyCells(20)}, TitleCells, 20},
		{"0 and less keep the default", []NotifyOption{WithTitleCells(0), WithBodyCells(-1)}, TitleCells, BodyCells},
	} {
		n := notifier(termcap.Caps{}, append([]NotifyOption{WithProtocol(OSC777)}, tc.o...)...)
		_, res := prepared(n, long)
		if len(res.Notification.Title) != tc.title || len(res.Notification.Body) != tc.body {
			t.Errorf("%s: Prepare cut to %d and %d, want %d and %d", tc.name, len(res.Notification.Title), len(res.Notification.Body), tc.title, tc.body)
		}
		want := ansi.URxvtExt("notify", strings.Repeat("t", tc.title), strings.Repeat("b", tc.body))
		if seq, _ := n.Sequence(long); seq != want {
			t.Errorf("%s: Sequence %d bytes, want %d", tc.name, len(seq), len(want))
		}
	}
}

func TestWithoutBell(t *testing.T) {
	x := Notification{Title: "Done"}
	plain := caps(termcap.BrandKitty, termcap.NoMux, termcap.Unsupported) // no protocol: the bell
	if got := seqOf(notifier(plain), x); got != "\a" {
		t.Fatalf("default: %q, want the bell", got)
	}
	seq, res := prepared(notifier(plain, WithoutBell()), x)
	if seq != "" || res.Sent || res.Skipped != SkipDisabled {
		t.Errorf("WithoutBell: %q, %+v; want nothing, SkipDisabled", seq, res)
	}
	if s, skip := notifier(plain, WithoutBell()).Sequence(x); s != "" || skip != SkipDisabled {
		t.Errorf("WithoutBell's Sequence: %q, %v", s, skip)
	}
	if got := seqOf(notifier(plain, WithoutBell(), WithProtocol(Bell)), x); got != "\a" {
		t.Errorf("WithoutBell with Bell set: %q, want the bell", got)
	}
	iterm := caps(termcap.BrandITerm2, termcap.NoMux, termcap.Unsupported)
	if got, want := seqOf(notifier(iterm, WithoutBell()), x), seqOf(notifier(iterm), x); got != want {
		t.Errorf("iTerm2 under WithoutBell: %q, want OSC 9 %q", got, want)
	}
}

func TestWithCopyRoute(t *testing.T) {
	tmux := at(termcap.BrandUnknown, termcap.Tmux)
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		c    termcap.Caps
		o    []CopyOption
		want Route
	}{
		{"default", termcap.Caps{}, nil, RouteOSC52},
		{"default inside tmux", tmux, nil, RouteOSC52Tmux},
		{"wrapped, outside tmux", termcap.Caps{}, []CopyOption{WithCopyRoute(RouteOSC52Tmux)}, RouteOSC52Tmux},
		{"plain, inside tmux", tmux, []CopyOption{WithCopyRoute(RouteOSC52)}, RouteOSC52},
		{"the backend is not forced", tmux, []CopyOption{WithCopyRoute(RouteBackend)}, RouteOSC52Tmux},
		{"tmux's buffer is not forced", termcap.Caps{}, []CopyOption{WithCopyRoute(RouteTmuxBuffer)}, RouteOSC52},
		{"none is not forced", termcap.Caps{}, []CopyOption{WithCopyRoute(RouteNone)}, RouteOSC52},
		{"a clipboard still wins", tmux, []CopyOption{WithCopyRoute(RouteOSC52), WithClipboard(&clipboard{})}, RouteBackend},
	} {
		route, deliver := PrepareCopy(ctx, tc.c, "x", tc.o...)
		if got := deliver(); route != tc.want || got.Route != tc.want {
			t.Errorf("%s: %v, delivered %+v; want %v", tc.name, route, got, tc.want)
		}
	}
	if route, deliver := PrepareCopy(ctx, termcap.Caps{}, strings.Repeat("x", MaxCopyBytes+1), WithCopyRoute(RouteOSC52)); route != RouteNone || deliver().Status != Failed {
		t.Errorf("an oversized forced OSC 52: %v", route)
	}
}

func TestLinker(t *testing.T) {
	for _, u := range []string{"https://example.com", "mailto:a@example.com", "javascript:x"} {
		a, errA := Link(u, "x", "id=1")
		b, errB := Linker{}.Link(u, "x", "id=1")
		if a != b || !errors.Is(errA, errB) {
			t.Errorf("%s: Link %q, %v; Linker{} %q, %v", u, a, errA, b, errB)
		}
	}
	ssh := Linker{Schemes: []string{"ssh", "SFTP"}}
	for _, u := range []string{"ssh://host/x", "SSH://host", "sftp://host/f"} {
		if _, err := ssh.Link(u, "x"); err != nil {
			t.Errorf("%s refused: %v", u, err)
		}
	}
	for _, u := range []string{"https://example.com", "file:///tmp/x", "host"} {
		if s, err := ssh.Link(u, "x"); !errors.Is(err, ErrScheme) {
			t.Errorf("%s: %q, %v; want ErrScheme", u, s, err)
		}
	}
	if s, _ := ssh.Link("ssh://host", "h", "id=1"); s != "\x1b]8;id=1;ssh://host\x07h\x1b]8;;\x07" {
		t.Errorf("Link = %q", s)
	}
}
