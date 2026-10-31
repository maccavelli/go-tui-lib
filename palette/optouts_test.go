package palette

import (
	"encoding/json/jsontext"
	"slices"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
)

// The picker's click, close and "?" options
// (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md Step 4;
// 0024-MADR D6, A1).

// TestWithoutClickAccept: a click selects the row under it and runs
// nothing; enter then runs it.
func TestWithoutClickAccept(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil, WithoutClickAccept())
		var r ran
		g.register(newDiscovering("items", runs(&r, 5)...))
		g.open(WithSize(40, 12)) // at (20, 6) in 80 × 24; content at (21, 7)
		g.w.Render()
		g.send(tea.MouseClickMsg{X: 25, Y: 7 + 3, Button: tea.MouseLeft}) // the third result line
		g.settle()
		if got := r.list(); len(got) != 0 || g.p.sel != 2 || len(g.w.Overlays()) != 1 {
			t.Fatalf("a click: ran %q, selection %d, overlays %v; want nothing run, 2, the picker open", got, g.p.sel, g.w.Overlays())
		}
		g.key("enter")
		if got := r.list(); !slices.Equal(got, []string{"c1"}) {
			t.Errorf("enter after the click ran %q; want c1", got)
		}
	})
}

// TestWithoutCloseOnAccept: accepting sends SelectedMsg and runs the hit,
// and the picker stays open for the next; after an argument stage, it is
// back at the query the hit was accepted from.
func TestWithoutCloseOnAccept(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil, WithoutCloseOnAccept())
		var r ran
		g.register(newDiscovering("items", runs(&r, 3)...))
		g.open()
		g.key("enter")
		g.key("down")
		g.key("enter")
		if got := r.list(); !slices.Equal(got, []string{"a1", "b1"}) {
			t.Fatalf("two accepts ran %q; want a1 and b1", got)
		}
		if n := len(filter[SelectedMsg](g.seen)); n != 2 || len(g.w.Overlays()) != 1 || g.p.done {
			t.Fatalf("SelectedMsgs %d, overlays %v, done %v; want 2, the picker open", n, g.w.Overlays(), g.p.done)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil, WithoutCloseOnAccept())
		var got []string
		h := Hit{ID: "n", Text: "set count", Score: fuzzy.Score{Grade: fuzzy.Prefix, Value: 1},
			Params: []command.Param{{Name: "count", Type: "integer", Required: true}},
			Run:    func(a jsontext.Value) tea.Cmd { got = append(got, string(a)); return nil }}
		g.register(&fake{name: "f", search: yields(h)})
		g.open()
		g.typed("set")
		g.key("enter")
		g.typed("3")
		g.key("enter")
		if !slices.Equal(got, []string{`{"count":3}`}) {
			t.Fatalf("ran %q; want count 3", got)
		}
		if g.p.args != nil || g.p.input.Value() != "set" || len(g.w.Overlays()) != 1 || len(g.texts()) != 1 {
			t.Errorf("after the stage: args %v, query %q, overlays %v, rows %q; want the query set again, the picker open", g.p.args, g.p.input.Value(), g.w.Overlays(), g.texts())
		}
	})
}

// TestWithoutPrefixList: "?" is an ordinary query, searched for by every
// provider, not the list of the prefixes.
func TestWithoutPrefixList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		for _, c := range []struct {
			name string
			opts []Option
			list bool
		}{{"default", nil, true}, {"WithoutPrefixList", []Option{WithoutPrefixList()}, false}} {
			cmds, plain := &fake{name: "commands"}, &fake{name: "plain"}
			l := newLoop(t, c.opts...)
			l.add(prefixed{cmds, ">"}, Availability{})
			l.add(plain, Availability{})
			l.e.open(nil)
			l.query(listPrefixes)
			searched := plain.searched()
			asked := len(searched) > 0 && searched[len(searched)-1] == listPrefixes
			if l.e.prefix != c.list || asked == c.list {
				t.Errorf("%s: the prefix list %v, plain searched %q; want the list %v", c.name, l.e.prefix, searched, c.list)
			}
		}
	})
}
