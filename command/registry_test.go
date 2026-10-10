package command

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/when"
)

func noop(context.Context, *Invocation) (Result, error) { return Result{}, nil }

// cmd is a command with id and danger d and a no-op handler, changed by
// each of opts.
func cmd(id ID, d Danger, opts ...func(*Command)) Command {
	c := Command{ID: id, Title: string(id), Danger: d, Handler: HandlerFunc(noop)}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func registryOf(t testing.TB, o []RegistryOption, cmds ...Command) *Registry {
	t.Helper()
	r := NewRegistry(o...)
	if err := r.Register(cmds...); err != nil {
		t.Fatal(err)
	}
	return r
}

// collect runs c, and every command a BatchMsg it returns holds, and
// returns the messages.
func collect(c tea.Cmd) []tea.Msg {
	if c == nil {
		return nil
	}
	m := c()
	if b, ok := m.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, x := range b {
			out = append(out, collect(x)...)
		}
		return out
	}
	return []tea.Msg{m}
}

// resultOf is the one ResultMsg among msgs.
func resultOf(t *testing.T, msgs []tea.Msg) ResultMsg {
	t.Helper()
	var found []ResultMsg
	for _, m := range msgs {
		if r, ok := m.(ResultMsg); ok {
			found = append(found, r)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d ResultMsgs in %v, want 1", len(found), msgs)
	}
	return found[0]
}

// builtinIDs is the registry's own commands, which ids leaves out.
var builtinIDs = map[ID]bool{idList: true, idDescribe: true, idQuit: true}

// ids is the IDs seq yields, but the registry's own.
func ids(seq func(func(Command) bool)) []ID {
	var out []ID
	for c := range seq {
		if !builtinIDs[c.ID] {
			out = append(out, c.ID)
		}
	}
	return out
}

func TestIDValid(t *testing.T) {
	long := ID(strings.Repeat("a", 64) + "." + strings.Repeat("b", 63))
	for _, id := range []ID{"workspace.focus.next", "a", "a-b.c2", "0.x", long} {
		if err := id.Valid(); err != nil {
			t.Errorf("%q: %v", id, err)
		}
	}
	for _, id := range []ID{
		"", "Workspace.focus", "a..b", ".a", "a.", "-a", "a.-b", "a b", "a_b", "é",
		long + "c",
	} {
		if err := id.Valid(); err == nil {
			t.Errorf("%q is valid", id)
		}
	}
	if got := slices.Collect(ID("workspace.focus.next").Segments()); !slices.Equal(got, []string{"workspace", "focus", "next"}) {
		t.Errorf("Segments = %v", got)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := registryOf(t, nil, cmd("a", ReadOnly, func(c *Command) { c.Slash = "a" }))
	bad := map[string][]Command{
		"a taken ID":              {cmd("a", ReadOnly)},
		"one ID twice":            {cmd("b", ReadOnly), cmd("b", ReadOnly)},
		"a malformed ID":          {cmd("B", ReadOnly)},
		"no danger":               {cmd("c", 0)},
		"an unknown danger":       {cmd("c", Destructive+1)},
		"an unknown kind":         {cmd("c", UI, func(c *Command) { c.Kind = Forward + 1 })},
		"an action, no handler":   {cmd("c", UI, func(c *Command) { c.Handler = nil })},
		"a prompt, no handler":    {cmd("c", UI, func(c *Command) { c.Kind, c.Handler = Prompt, nil })},
		"a bad when":              {cmd("c", UI, func(c *Command) { c.When = "a &&" })},
		"a taken slash":           {cmd("c", UI, func(c *Command) { c.Slash = "a" })},
		"a taken alias":           {cmd("c", UI, func(c *Command) { c.Aliases = []string{"a"} })},
		"a slash with a space":    {cmd("c", UI, func(c *Command) { c.Slash = "a b" })},
		"a slash with its /":      {cmd("c", UI, func(c *Command) { c.Slash = "/c" })},
		"an empty alias":          {cmd("c", UI, func(c *Command) { c.Aliases = []string{""} })},
		"one good, one malformed": {cmd("c", UI), cmd("D", UI)},
	}
	for name, cmds := range bad {
		if err := r.Register(cmds...); err == nil {
			t.Errorf("%s: registered", name)
		}
	}
	if got := ids(r.All()); !slices.Equal(got, []ID{"a"}) || r.Version() != 1 {
		t.Errorf("after the refusals: %v at version %d, want [a] at 1", got, r.Version())
	}
	if err := r.Register(cmd("f", UI, func(c *Command) { c.Kind, c.Handler = Forward, nil })); err != nil {
		t.Errorf("a forward with no handler: %v", err)
	}
}

func TestAllSorted(t *testing.T) {
	r := registryOf(t, nil, cmd("c", UI), cmd("a.b", UI), cmd("a", UI, func(c *Command) { c.Hidden = true }), cmd("b", UI))
	if got := ids(r.All()); !slices.Equal(got, []ID{"a", "a.b", "b", "c"}) {
		t.Errorf("All = %v", got)
	}
}

func TestAvailable(t *testing.T) {
	r := registryOf(t, nil,
		cmd("a", UI),
		cmd("b", UI, func(c *Command) { c.When = "k" }),
		cmd("c", UI, func(c *Command) { c.When = "!k" }),
		cmd("d", UI, func(c *Command) { c.Surfaces = SurfaceCLI }),
		cmd("e", UI, func(c *Command) { c.Hidden = true }),
	)
	on, off := when.Map{"k": when.BoolValue(true)}, when.Map{}
	for _, c := range []struct {
		ctx  when.Context
		s    Surface
		want []ID
	}{
		{on, SurfacePalette, []ID{"a", "b"}},
		{off, SurfacePalette, []ID{"a", "c"}},
		{on, 0, []ID{"a", "b", "d"}},
		{off, SurfaceCLI, []ID{"a", "c", "d"}},
		{nil, SurfaceKey | SurfaceCLI, []ID{"a", "c", "d"}},
	} {
		if got := ids(r.Available(c.ctx, c.s)); !slices.Equal(got, c.want) {
			t.Errorf("Available(%v, %s) = %v, want %v", c.ctx, c.s, got, c.want)
		}
	}
	// Dispatch makes the same checks: When for every origin, and the
	// surfaces of the origin, a click needing SurfaceKey and the program
	// none (A4).
	k := registryOf(t, nil,
		cmd("key", UI, func(c *Command) { c.Surfaces = SurfaceKey }),
		cmd("palette", UI, func(c *Command) { c.Surfaces = SurfacePalette }),
		cmd("when", UI, func(c *Command) { c.When = "k" }),
	)
	for _, c := range []struct {
		id     ID
		origin Origin
		ctx    when.Context
		want   error
	}{
		{"key", OriginKey, nil, nil},
		{"key", OriginMouse, nil, nil},
		{"key", OriginPalette, nil, ErrUnavailable},
		{"palette", OriginMouse, nil, ErrUnavailable},
		{"palette", OriginProgram, nil, nil},
		{"palette", OriginAgent, nil, ErrUnavailable},
		{"when", OriginKey, on, nil},
		{"when", OriginKey, off, ErrUnavailable},
		{"when", OriginProgram, off, ErrUnavailable},
		{"hidden", OriginProgram, nil, ErrUnknown},
	} {
		res := resultOf(t, collect(k.Dispatch(t.Context(), Request{ID: c.id, Origin: c.origin, WhenContext: c.ctx})))
		if !errors.Is(res.Err, c.want) || (c.want == nil) != (res.Err == nil) {
			t.Errorf("%s from %s: %v, want %v", c.id, c.origin, res.Err, c.want)
		}
	}
}

func TestSnapshotsUnderRace(t *testing.T) {
	const batches, per = 120, 10
	r := NewRegistry()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			var last uint64
			for {
				select {
				case <-stop:
					return
				default:
				}
				v := r.Version()
				if v < last {
					t.Errorf("Version fell from %d to %d", last, v)
					return
				}
				last = v
				var prev ID
				n := 0
				for c := range r.All() {
					if prev != "" && c.ID <= prev {
						t.Errorf("All: %s after %s", c.ID, prev)
						return
					}
					prev = c.ID
					if !builtinIDs[c.ID] {
						n++
					}
				}
				if n%per != 0 {
					t.Errorf("All saw %d commands, not whole batches of %d", n, per)
					return
				}
				r.Lookup("b0.c0")
			}
		})
	}
	for b := range batches {
		batch := make([]Command, per)
		for i := range batch {
			batch[i] = cmd(ID(fmt.Sprintf("b%d.c%d", b, i)), UI)
		}
		if err := r.Register(batch...); err != nil {
			t.Fatal(err)
		}
		if b%2 == 1 {
			gone := make([]ID, per)
			for i := range gone {
				gone[i] = ID(fmt.Sprintf("b%d.c%d", b-1, i))
			}
			r.Remove(gone...)
		}
	}
	close(stop)
	wg.Wait()
	if got := r.Version(); got != batches+batches/2 {
		t.Errorf("Version = %d, want %d", got, batches+batches/2)
	}
}

// next runs w in a goroutine and returns what it sends, or nil after d.
func next(w tea.Cmd, d time.Duration) func() tea.Msg {
	ch := make(chan tea.Msg, 1)
	go func() { ch <- w() }()
	return func() tea.Msg {
		select {
		case m := <-ch:
			return m
		case <-time.After(d):
			return nil
		}
	}
}

func TestWatch(t *testing.T) {
	r := NewRegistry()
	early := next(r.Watch(), 50*time.Millisecond)
	if m := early(); m != nil {
		t.Fatalf("Watch returned %v before any change", m)
	}
	w := next(r.Watch(), 2*time.Second)
	if err := r.Register(cmd("a", UI)); err != nil {
		t.Fatal(err)
	}
	if m := w(); m != (ChangedMsg{Version: 1}) {
		t.Errorf("after Register: %v, want ChangedMsg{1}", m)
	}
	w = next(r.Watch(), 2*time.Second)
	r.Remove("a")
	if m := w(); m != (ChangedMsg{Version: 2}) {
		t.Errorf("after Remove: %v, want ChangedMsg{2}", m)
	}
	quiet := next(r.Watch(), 50*time.Millisecond)
	r.Remove("missing")
	if m := quiet(); m != nil || r.Version() != 2 {
		t.Errorf("removing nothing: %v at version %d", m, r.Version())
	}
}

func TestRegisterCopies(t *testing.T) {
	aliases := []string{"x"}
	r := registryOf(t, nil, cmd("a", UI, func(c *Command) { c.Aliases = aliases }))
	aliases[0] = "y"
	if c, _ := r.Lookup("a"); c.Aliases[0] != "x" {
		t.Error("a caller's slice reached the snapshot")
	}
	if _, ok := r.Slash("/x"); !ok {
		t.Error("Slash with its leading /")
	}
}

func TestStrings(t *testing.T) {
	for got, want := range map[fmt.Stringer]string{
		Action: "action", Prompt: "prompt", Forward: "forward", Kind(9): "Kind(9)",
		Danger(0): "undeclared", ReadOnly: "read-only", UI: "ui", Mutating: "mutating", Destructive: "destructive",
		Surface(0): "none", SurfaceKey | SurfaceAgent: "key|agent", AllSurfaces: "key|palette|slash|cli|agent",
		Loop: "loop", Async: "async", Global: "global", Scope("pane"): "pane",
		Builtin: "builtin", Plugin: "plugin", Source{Kind: MCP, Name: "gh"}: "mcp:gh", Source{}: "builtin",
		Origin(0): "unset", OriginMouse: "mouse", OriginProgram: "program", Origin(99): "Origin(99)",
	} {
		if got.String() != want {
			t.Errorf("%#v.String() = %q, want %q", got, got.String(), want)
		}
	}
}

// TestWithMaxArgBytes: the option sets the limit for every way in, and a
// value below 1 keeps DefaultMaxArgBytes
// (docs/decisions/0014-PLAN-hardening.md Step 6).
func TestWithMaxArgBytes(t *testing.T) {
	if DefaultMaxArgBytes != argLimit {
		t.Fatalf("DefaultMaxArgBytes = %d, want %d", DefaultMaxArgBytes, argLimit)
	}
	reg := func(o ...RegistryOption) *Registry {
		return registryOf(t, o, cmd("plain", UI, func(c *Command) { c.Slash = "plain" }))
	}
	r := reg(WithMaxArgBytes(10))
	for n, want := range map[int]bool{10: true, 11: false} {
		args := `"` + strings.Repeat("a", n-2) + `"`
		_, err := r.Run(t.Context(), Request{ID: "plain", Args: []byte(args), Origin: OriginKey})
		_, serr := r.ParseSlash("/plain " + strings.Repeat("a", n))
		_, perr := r.ParseArgs("plain", []string{strings.Repeat("a", n)}, OriginCLI)
		for what, err := range map[string]error{"Run": err, "ParseSlash": serr, "ParseArgs": perr} {
			if want && err != nil {
				t.Errorf("WithMaxArgBytes(10), %s of %d bytes: %v", what, n, err)
			}
			if !want {
				overLimitOf(t, what, n, 10, err)
			}
		}
	}
	for _, n := range []int{0, -1} {
		r := reg(WithMaxArgBytes(n))
		raw := strings.Repeat("a", DefaultMaxArgBytes)
		if _, err := r.Run(t.Context(), Request{ID: "plain", Raw: raw, Origin: OriginKey}); err != nil {
			t.Errorf("WithMaxArgBytes(%d), at the default: %v", n, err)
		}
		_, err := r.Run(t.Context(), Request{ID: "plain", Raw: raw + "a", Origin: OriginKey})
		overLimitOf(t, fmt.Sprintf("WithMaxArgBytes(%d), one byte over the default", n), DefaultMaxArgBytes+1, DefaultMaxArgBytes, err)
	}
}

// overLimitOf fails t unless err is an *ArgError for n bytes over limit.
func overLimitOf(t *testing.T, what string, n, limit int, err error) {
	t.Helper()
	want := fmt.Sprintf("arguments are %d bytes, over %d", n, limit)
	if _, ok := errors.AsType[*ArgError](err); !ok || !strings.Contains(err.Error(), want) {
		t.Errorf("%s: %v; want an *ArgError with %q", what, err, want)
	}
}
