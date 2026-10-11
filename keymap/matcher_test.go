package keymap

import (
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/command"
)

// clock records each timer the dispatcher starts; a timer's command
// returns its TimeoutMsg at once.
type clock struct{ timers []time.Duration }

func (c *clock) tick(d time.Duration, fn func(time.Time) tea.Msg) tea.Cmd {
	c.timers = append(c.timers, d)
	return func() tea.Msg { return fn(time.Time{}) }
}

// press and release are the messages for a stroke's key.
func press(t *testing.T, s string) tea.KeyPressMsg {
	t.Helper()
	st := mustStroke(t, s)
	return tea.KeyPressMsg{Code: st.Code, Mod: st.Mod}
}

func release(t *testing.T, s string) tea.KeyReleaseMsg {
	t.Helper()
	return tea.KeyReleaseMsg(press(t, s))
}

// replayed is the keys of a Replay, as strokes.
func replayed(o Outcome) string {
	var out []string
	for _, k := range o.Replay {
		out = append(out, k.Key().Keystroke())
	}
	return fmt.Sprint(out)
}

var global = Path{command.Global}

func dispatcher(t *testing.T, s Settings, bs ...Binding) (*Dispatcher, *clock) {
	t.Helper()
	c := &clock{}
	if s.Tick == nil {
		s.Tick = c.tick
	}
	return NewDispatcher(mustNew(t, s, nil, bs)), c
}

// TestSequencesComplete: a one-stroke binding, a chord, a leader sequence
// and a four-stroke sequence complete, each stroke before the last pending
// with one timer.
func TestSequencesComplete(t *testing.T) {
	find := bind(t, ProgramLayer, command.Global, "<leader> f", "find")
	find.Args = []byte(`{"q":1}`)
	d, c := dispatcher(t, Settings{Leader: "ctrl+x"},
		bind(t, ProgramLayer, command.Global, "x", "cut"),
		bind(t, ProgramLayer, command.Global, "ctrl+k ctrl+c", "comment"),
		find,
		bind(t, ProgramLayer, command.Global, "a b c d", "four"),
	)
	for _, c2 := range []struct {
		keys string
		want command.ID
		args string
	}{
		{"x", "cut", ""}, {"ctrl+k ctrl+c", "comment", ""}, {"ctrl+x f", "find", `{"q":1}`}, {"a b c d", "four", ""},
	} {
		q := seq(t, c2.keys)
		before := len(c.timers)
		for i, s := range q {
			out, cmd := d.Update(press(t, s.String()), global, nil)
			if i < len(q)-1 {
				if !out.Handled || out.Command != "" || cmd == nil || d.Pending().String() != q[:i+1].String() {
					t.Fatalf("%s, stroke %d: %+v, cmd %v, pending %q; want it pending with a timer", c2.keys, i, out, cmd != nil, d.Pending())
				}
				continue
			}
			if out.Command != c2.want || string(out.Args) != c2.args || !out.Handled || cmd != nil || len(d.Pending()) != 0 {
				t.Errorf("%s: %+v, cmd %v, pending %q; want %s", c2.keys, out, cmd != nil, d.Pending(), c2.want)
			}
		}
		if got := len(c.timers) - before; got != len(q)-1 {
			t.Errorf("%s started %d timers, want %d", c2.keys, got, len(q)-1)
		}
	}
}

// TestPrefixTimeout: a prefix starts a timer of Settings.Timeout, one second
// by default; its TimeoutMsg ends the sequence and gives back its keys,
// the leader left out.
func TestPrefixTimeout(t *testing.T) {
	d, c := dispatcher(t, Settings{Leader: "ctrl+x", Timeout: 250 * time.Millisecond},
		bind(t, ProgramLayer, command.Global, "g g", "top"),
		bind(t, ProgramLayer, command.Global, "<leader> g h", "help"),
	)
	for _, tc := range []struct{ keys, replay string }{
		{"g", "[g]"}, {"ctrl+x", "[]"}, {"ctrl+x g", "[g]"},
	} {
		var cmd tea.Cmd
		for _, s := range seq(t, tc.keys) {
			_, cmd = d.Update(press(t, s.String()), global, nil)
		}
		msg := cmd()
		if _, ok := msg.(TimeoutMsg); !ok {
			t.Fatalf("%s: the timer gave %T, want TimeoutMsg", tc.keys, msg)
		}
		out, next := d.Update(msg, global, nil)
		if !out.Handled || out.Command != "" || next != nil || replayed(out) != tc.replay || len(d.Pending()) != 0 {
			t.Errorf("%s, then the timeout: %+v, replay %s, pending %q; want replay %s", tc.keys, out, replayed(out), d.Pending(), tc.replay)
		}
	}
	if c.timers[0] != 250*time.Millisecond {
		t.Errorf("the timer is %v, want Settings.Timeout", c.timers[0])
	}
	d2, c2 := dispatcher(t, Settings{}, bind(t, ProgramLayer, command.Global, "g g", "top"))
	d2.Update(press(t, "g"), global, nil)
	if !slices.Equal(c2.timers, []time.Duration{time.Second}) {
		t.Errorf("with no Timeout, the timers are %v, want one second", c2.timers)
	}
}

// TestStaleTimeout: only the latest stroke's timer ends a sequence; an
// older one, or one whose sequence completed, does nothing.
func TestStaleTimeout(t *testing.T) {
	d, _ := dispatcher(t, Settings{}, bind(t, ProgramLayer, command.Global, "a b c", "abc"))
	_, first := d.Update(press(t, "a"), global, nil)
	_, second := d.Update(press(t, "b"), global, nil)
	if out, _ := d.Update(first(), global, nil); !out.Handled || len(out.Replay) != 0 || d.Pending().String() != "a b" {
		t.Fatalf("the first stroke's timer: %+v, pending %q; want nothing done", out, d.Pending())
	}
	if out, _ := d.Update(second(), global, nil); replayed(out) != "[a b]" || len(d.Pending()) != 0 {
		t.Fatalf("the latest timer: %+v, pending %q; want a b replayed", out, d.Pending())
	}
	d.Update(press(t, "a"), global, nil)
	_, late := d.Update(press(t, "b"), global, nil)
	if out, _ := d.Update(press(t, "c"), global, nil); out.Command != "abc" {
		t.Fatalf("a b c: %+v", out)
	}
	if out, _ := d.Update(late(), global, nil); !out.Handled || len(out.Replay) != 0 {
		t.Errorf("a timer after its sequence completed: %+v; want nothing done", out)
	}
}

// TestPathChange: a key on another path cancels a pending sequence, and is
// taken afresh; an equal path in a new slice does not.
func TestPathChange(t *testing.T) {
	d, _ := dispatcher(t, Settings{},
		bind(t, ProgramLayer, pane, "g g", "top"),
		bind(t, ProgramLayer, command.Global, "x", "cut"),
	)
	p := Path{command.Global, pane}
	_, timer := d.Update(press(t, "g"), p, nil)
	out, cmd := d.Update(press(t, "x"), global, nil)
	if out.Command != "cut" || len(out.Replay) != 0 || cmd != nil || len(d.Pending()) != 0 {
		t.Fatalf("x on another path: %+v, pending %q; want cut, the g dropped", out, d.Pending())
	}
	if out, _ := d.Update(timer(), global, nil); len(out.Replay) != 0 {
		t.Errorf("the cancelled sequence's timer replayed %s", replayed(out))
	}
	d.Update(press(t, "g"), p, nil)
	if out, _ := d.Update(press(t, "g"), slices.Clone(p), nil); out.Command != "top" {
		t.Errorf("g g on an equal path: %+v; want top", out)
	}
}

// TestSetKeymap: a new keymap cancels anything pending.
func TestSetKeymap(t *testing.T) {
	d, _ := dispatcher(t, Settings{}, bind(t, ProgramLayer, command.Global, "g g", "top"))
	_, timer := d.Update(press(t, "g"), global, nil)
	d.SetKeymap(mustNew(t, Settings{}, nil, []Binding{bind(t, ProgramLayer, command.Global, "g", "go")}))
	if len(d.Pending()) != 0 {
		t.Fatalf("pending %q after SetKeymap", d.Pending())
	}
	if out, _ := d.Update(timer(), global, nil); len(out.Replay) != 0 {
		t.Errorf("the old sequence's timer replayed %s", replayed(out))
	}
	if out, _ := d.Update(press(t, "g"), global, nil); out.Command != "go" {
		t.Errorf("g on the new keymap: %+v; want go", out)
	}
}

// TestRelease: a release is matched only by a Release stroke, a press never
// is, and a release that matches nothing leaves a pending sequence as it is.
func TestRelease(t *testing.T) {
	d, _ := dispatcher(t, Settings{Features: EventTypes},
		bind(t, ProgramLayer, command.Global, "space:release", "talk"),
		bind(t, ProgramLayer, command.Global, "a", "cmd.a"),
		bind(t, ProgramLayer, command.Global, "ctrl+x f", "find"),
	)
	for _, c := range []struct {
		msg  tea.Msg
		want command.ID
	}{
		{press(t, "space"), ""}, {release(t, "space"), "talk"}, {press(t, "a"), "cmd.a"}, {release(t, "a"), ""},
	} {
		if out, _ := d.Update(c.msg, global, nil); out.Command != c.want || out.Handled != (c.want != "") {
			t.Errorf("%T %v: %+v; want %q", c.msg, c.msg, out, c.want)
		}
	}
	d.Update(press(t, "ctrl+x"), global, nil)
	if out, cmd := d.Update(release(t, "ctrl+x"), global, nil); out.Handled || cmd != nil || d.Pending().String() != "ctrl+x" {
		t.Fatalf("ctrl+x released while pending: %+v, pending %q; want it left pending", out, d.Pending())
	}
	if out, _ := d.Update(press(t, "f"), global, nil); out.Command != "find" {
		t.Errorf("ctrl+x, its release, f: %+v; want find", out)
	}
}

// TestUnmatched: a key that matches nothing is not handled; one that ends a
// sequence gives the sequence's keys back, and is taken afresh.
func TestUnmatched(t *testing.T) {
	d, _ := dispatcher(t, Settings{},
		bind(t, ProgramLayer, command.Global, "g g", "top"),
		bind(t, ProgramLayer, command.Global, "ctrl+p", "palette"),
		bind(t, ProgramLayer, command.Global, "ctrl+k ctrl+c", "comment"),
	)
	if out, cmd := d.Update(press(t, "z"), global, nil); out.Handled || out.Command != "" || out.Replay != nil || cmd != nil {
		t.Errorf("z: %+v, cmd %v; want a zero Outcome", out, cmd != nil)
	}
	for _, c := range []struct {
		second  string
		want    command.ID
		handled bool
		pending string
	}{
		{"z", "", false, ""}, {"ctrl+p", "palette", true, ""}, {"ctrl+k", "", true, "ctrl+k"},
	} {
		d.SetKeymap(d.k)
		d.Update(press(t, "g"), global, nil)
		out, cmd := d.Update(press(t, c.second), global, nil)
		if out.Command != c.want || out.Handled != c.handled || replayed(out) != "[g]" ||
			d.Pending().String() != c.pending || (cmd != nil) != (c.pending != "") {
			t.Errorf("g %s: %+v, replay %s, pending %q; want %q, handled %v, g replayed",
				c.second, out, replayed(out), d.Pending(), c.want, c.handled)
		}
	}
	if out, cmd := d.Update(tea.WindowSizeMsg{Width: 80}, global, nil); out.Handled || cmd != nil {
		t.Errorf("a WindowSizeMsg: %+v", out)
	}
	if out, _ := NewDispatcher(nil).Update(press(t, "g"), global, nil); out.Handled {
		t.Errorf("a dispatcher with no keymap handled g: %+v", out)
	}
}

// TestDispatchScopeDecides: the dispatcher lets the innermost scope decide,
// as Lookup does (amendment A6), and a keypad key fires its main key's
// binding (amendment A5).
func TestDispatchScopeDecides(t *testing.T) {
	d, _ := dispatcher(t, Settings{},
		bind(t, ProgramLayer, command.Global, "g", "g.g"),
		bind(t, ProgramLayer, pane, "g g", "pane.gg"),
		bind(t, ProgramLayer, command.Global, "ctrl+k enter", "run"),
	)
	p := Path{command.Global, pane}
	if out, _ := d.Update(press(t, "g"), p, nil); !out.Handled || out.Command != "" || d.Pending().String() != "g" {
		t.Fatalf("g under pane:a: %+v; want it pending for g g", out)
	}
	if out, _ := d.Update(press(t, "g"), p, nil); out.Command != "pane.gg" {
		t.Errorf("g g under pane:a: %+v", out)
	}
	if out, _ := d.Update(press(t, "g"), global, nil); out.Command != "g.g" {
		t.Errorf("g on the global path: %+v", out)
	}
	d.Update(press(t, "ctrl+k"), global, nil)
	if out, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyKpEnter}, global, nil); out.Command != "run" {
		t.Errorf("ctrl+k, keypad enter: %+v; want run", out)
	}
}

// TestRealTimer: the default timer is tea.Tick, and its TimeoutMsg comes
// after one second, on synctest's fake clock.
func TestRealTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := NewDispatcher(mustNew(t, Settings{}, nil, []Binding{bind(t, ProgramLayer, command.Global, "g g", "top")}))
		_, cmd := d.Update(press(t, "g"), global, nil)
		start := time.Now()
		msg := cmd()
		if waited := time.Since(start); waited != time.Second {
			t.Errorf("the timer fired after %v, want one second", waited)
		}
		if out, _ := d.Update(msg, global, nil); replayed(out) != "[g]" {
			t.Errorf("the real timer's message: %+v; want g replayed", out)
		}
	})
}
