package keymap

import (
	"encoding/json/jsontext"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/when"
)

// Outcome is what the dispatcher made of a message.
type Outcome struct {
	Command command.ID     // the command a sequence completed; "" when none
	Args    jsontext.Value // its binding's args
	// Handled reports that the message was the dispatcher's: a key that
	// matched or started or extended a sequence, or a TimeoutMsg. A key
	// that is not handled goes to the focused pane.
	Handled bool
	// Replay is the keys of a sequence that ended without a match, the
	// leader left out: they go to the focused pane, before the message
	// when it is not handled.
	Replay []tea.KeyPressMsg
}

// TimeoutMsg ends a pending sequence when its timer fires. Only the timer
// of the latest stroke ends it; an older one does nothing.
type TimeoutMsg struct{ gen uint64 }

// Dispatcher matches keys against a keymap along a focus path: one key
// binding, chords, and leader sequences, with a timeout (MADR §6). It is
// a state machine driven by Update, with no goroutine: its one timer is
// the command Update returns, which Bubble Tea runs. It is not safe for
// concurrent use; the program's Update calls it.
type Dispatcher struct {
	k       *Keymap
	path    Path       // the path the pending sequence started on
	keys    []keyEvent // the pending sequence's keys
	strokes Sequence   // the same, as the bindings name them
	gen     uint64     // bumped by each stroke that pends and each cancel
}

// keyEvent is a key, pressed or released.
type keyEvent struct {
	key     tea.Key
	release bool
}

// NewDispatcher is a dispatcher over k. A nil k matches nothing.
func NewDispatcher(k *Keymap) *Dispatcher { return &Dispatcher{k: k} }

// SetKeymap replaces the keymap, as after Rebuild or a reload, and cancels
// anything pending.
func (d *Dispatcher) SetKeymap(k *Keymap) {
	d.k = k
	d.cancel()
}

// Pending is the strokes of the sequence in progress, as its bindings name
// them, for a "ctrl+x …" hint; empty when none is.
func (d *Dispatcher) Pending() Sequence { return slices.Clone(d.strokes) }

func (d *Dispatcher) cancel() {
	d.path, d.keys, d.strokes = nil, nil, nil
	d.gen++
}

// Update takes a message, the focus path and the when values:
//   - a key that completes a binding gives its command;
//   - a key that starts or extends a longer binding is Handled, and gives a
//     timer command, whose TimeoutMsg ends the sequence;
//   - a key that ends a sequence without a match gives the sequence's keys
//     in Replay, the leader left out, and is then taken afresh;
//   - a release is matched only by a Release stroke, and one that matches
//     nothing leaves a pending sequence as it is;
//   - a path that differs from the one the pending sequence started on
//     cancels it, and the key is taken afresh.
//
// Any other message gives a zero Outcome.
func (d *Dispatcher) Update(msg tea.Msg, p Path, ctx when.Context) (Outcome, tea.Cmd) {
	var ev keyEvent
	switch m := msg.(type) {
	case TimeoutMsg:
		if m.gen != d.gen || len(d.keys) == 0 {
			return Outcome{Handled: true}, nil
		}
		replay := d.replay()
		d.cancel()
		return Outcome{Handled: true, Replay: replay}, nil
	case tea.KeyPressMsg:
		ev = keyEvent{key: m.Key()}
	case tea.KeyReleaseMsg:
		ev = keyEvent{key: m.Key(), release: true}
	default:
		return Outcome{}, nil
	}
	if len(d.keys) > 0 && !slices.Equal(d.path, p) {
		d.cancel()
	}
	if d.k == nil {
		return Outcome{}, nil
	}
	if len(d.keys) == 0 {
		return d.fresh(p, ctx, ev)
	}
	keys := append(slices.Clone(d.keys), ev)
	full, longer := d.k.decide(p, ctx, keys)
	switch {
	case full != nil:
		d.cancel()
		return Outcome{Command: full.Command, Args: full.Args, Handled: true}, nil
	case longer != nil:
		d.keys, d.strokes = keys, longer.Keys[:len(keys)]
		return Outcome{Handled: true}, d.tick()
	case ev.release:
		return Outcome{}, nil
	}
	replay := d.replay()
	d.cancel()
	out, cmd := d.fresh(p, ctx, ev)
	out.Replay = replay
	return out, cmd
}

// fresh takes a key with nothing pending.
func (d *Dispatcher) fresh(p Path, ctx when.Context, ev keyEvent) (Outcome, tea.Cmd) {
	full, longer := d.k.decide(p, ctx, []keyEvent{ev})
	switch {
	case full != nil:
		return Outcome{Command: full.Command, Args: full.Args, Handled: true}, nil
	case longer != nil:
		d.path, d.keys, d.strokes = slices.Clone(p), []keyEvent{ev}, longer.Keys[:1]
		return Outcome{Handled: true}, d.tick()
	}
	return Outcome{}, nil
}

// tick bumps the generation and starts the timer for it.
func (d *Dispatcher) tick() tea.Cmd {
	d.gen++
	gen := d.gen
	timeout, tick := d.k.settings.Timeout, d.k.settings.Tick
	if timeout <= 0 {
		timeout = time.Second
	}
	if tick == nil {
		tick = tea.Tick
	}
	return tick(timeout, func(time.Time) tea.Msg { return TimeoutMsg{gen: gen} })
}

// replay is the pending keys, less the leader's and any release.
func (d *Dispatcher) replay() []tea.KeyPressMsg {
	skip := 0
	if l := d.k.leader; len(l) > 0 && hasPrefix(d.strokes, l) {
		skip = len(l)
	}
	var out []tea.KeyPressMsg
	for _, ev := range d.keys[skip:] {
		if !ev.release {
			out = append(out, tea.KeyPressMsg(ev.key))
		}
	}
	return out
}

// decide is Lookup for keys: along p, the innermost scope with a binding
// whose when holds and whose strokes match the keys, start with them, or
// are matched by their start decides them (amendment A6). It gives the
// binding the keys complete, or failing that the first longer one; or
// neither, when the deciding scope's binding is shorter, or no scope
// decides.
func (k *Keymap) decide(p Path, ctx when.Context, keys []keyEvent) (full, longer *entry) {
	for _, scope := range slices.Backward(p) {
		decided := false
		for _, e := range k.scopes[scope] {
			n := min(len(e.Keys), len(keys))
			if !matchAll(e.Keys[:n], keys[:n]) || !e.holds(ctx) {
				continue
			}
			switch {
			case len(e.Keys) == len(keys):
				return e, nil
			case len(e.Keys) > len(keys) && longer == nil:
				longer = e
			}
			decided = true
		}
		if decided {
			return nil, longer
		}
	}
	return nil, nil
}

// matchAll reports whether each key matches its stroke: a release only a
// Release stroke, and a press any other.
func matchAll(q Sequence, keys []keyEvent) bool {
	for i, s := range q {
		if keys[i].release != (s.Event == Release) || !s.Matches(keys[i].key) {
			return false
		}
	}
	return true
}
