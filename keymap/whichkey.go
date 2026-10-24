package keymap

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// ActiveKey is a key that can come next, for a which-key panel (MADR
// amendments A1 and A10).
type ActiveKey struct {
	Stroke    Stroke
	Label     string     // as Label writes the stroke
	Continues bool       // extends the sequence rather than completing it
	Command   command.ID // the command it completes; "" when Continues
}

// ActiveKeys is the keys that can come next along a path with ctx: after
// the pending sequence, or, with nothing pending on that path, the first
// strokes. Each comes once, decided as Update decides it: by the
// innermost scope with a binding whose when holds and whose keys go on
// with it, where a binding the key completes wins over one it continues.
// They come innermost scope first, in resolution order. Labels are
// Label's, on goos with g. ActiveKeys does not change the dispatcher.
func (d *Dispatcher) ActiveKeys(p Path, ctx when.Context, goos string, g glyph.Set) []ActiveKey {
	if d.k == nil {
		return nil
	}
	keys := d.pendingOn(p)
	n := len(keys)
	var out []ActiveKey
	seen := map[Stroke]bool{}
	for _, scope := range slices.Backward(p) {
		here := map[Stroke]int{} // the index in out of each stroke this scope decides
		for _, e := range d.k.scopes[scope] {
			if len(e.Keys) <= n || !e.holds(ctx) || !matchAll(e.Keys[:n], keys) {
				continue
			}
			s := e.Keys[n]
			completes := len(e.Keys) == n+1
			if i, ok := here[s]; ok {
				if completes && out[i].Continues {
					out[i].Continues, out[i].Command = false, e.Command
				}
				continue
			}
			if seen[s] {
				continue // an inner scope decides it
			}
			seen[s], here[s] = true, len(out)
			a := ActiveKey{Stroke: s, Label: s.Label(goos, g), Continues: !completes}
			if completes {
				a.Command = e.Command
			}
			out = append(out, a)
		}
	}
	return out
}

// pendingOn is the pending keys, when they were typed on p; Update would
// cancel them on another path.
func (d *Dispatcher) pendingOn(p Path) []keyEvent {
	if len(d.keys) == 0 || !slices.Equal(d.path, p) {
		return nil
	}
	return d.keys
}

// Explanation is what a key does, for a "keys debug" view (amendment A1):
// the key as it arrived and as Normalize reads it, and the bindings it
// meets after the pending sequence.
type Explanation struct {
	Raw        tea.Key
	Normalized tea.Key
	Changed    bool   // Normalize changed the key
	Stroke     Stroke // the normalised key, as a stroke
	// Matches is the deciding scope's bindings whose when holds and which
	// the key completes or continues, the one that wins first, then the
	// rest in resolution order. AltGr text on Windows meets none.
	Matches []Binding
	// Shadowed is the bindings the key also meets that cannot win: those
	// in an outer scope the deciding one hides, and those after one with
	// the same keys in their scope.
	Shadowed []Binding
}

// Explain is what k would do as the next key along a path with ctx, after
// the pending sequence when it was typed on that path, as Update reads
// it. Explain does not change the dispatcher.
func (d *Dispatcher) Explain(k tea.Key, p Path, ctx when.Context) Explanation {
	ev := d.event(k, false)
	x := Explanation{
		Raw: k, Normalized: ev.key, Changed: ev.key != k,
		Stroke: Stroke{Code: ev.key.Code, Mod: ev.key.Mod &^ locks},
	}
	if d.k == nil {
		return x
	}
	keys := append(slices.Clone(d.pendingOn(p)), ev)
	n := len(keys)
	decided := false
	for _, scope := range slices.Backward(p) {
		var here []*entry
		related := false
		for _, e := range d.k.scopes[scope] {
			m := min(len(e.Keys), n)
			if !e.holds(ctx) || !matchAll(e.Keys[:m], keys[:m]) {
				continue
			}
			related = true
			if len(e.Keys) >= n {
				here = append(here, e)
			}
		}
		if decided {
			for _, e := range here {
				x.Shadowed = append(x.Shadowed, e.Binding)
			}
			continue
		}
		if !related {
			continue
		}
		decided = true
		// The winner: the first the keys complete, else the first they
		// continue; then the rest, and a binding after one with its keys
		// is shadowed.
		win := slices.IndexFunc(here, func(e *entry) bool { return len(e.Keys) == n })
		if win > 0 {
			w := here[win]
			here = append([]*entry{w}, slices.Delete(here, win, win+1)...)
		}
		var kept []*entry
		for _, e := range here {
			if slices.ContainsFunc(kept, func(o *entry) bool { return slices.Equal(o.Keys, e.Keys) }) {
				x.Shadowed = append(x.Shadowed, e.Binding)
				continue
			}
			kept = append(kept, e)
			x.Matches = append(x.Matches, e.Binding)
		}
	}
	return x
}
