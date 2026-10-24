package workspace_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/command/commandtea"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// focusedMsg is the message a beacon's Focus command gives.
type focusedMsg layout.PaneID

// beacon is a pane whose Focus returns a command, so a test sees the
// effect a focus change has reach the program.
type beacon struct{ id layout.PaneID }

func (b beacon) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return b, nil }
func (b beacon) View(int, int) string                     { return string(b.id) }
func (b beacon) Focus() tea.Cmd                           { return func() tea.Msg { return focusedMsg(b.id) } }
func (b beacon) Blur()                                    {}

// messages runs cmd and every command a tea.BatchMsg it gives holds, and
// returns the messages.
func messages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	m := cmd()
	if b, ok := m.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range b {
			out = append(out, messages(c)...)
		}
		return out
	}
	if v := reflect.ValueOf(m); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
		var out []tea.Msg
		for i := range v.Len() {
			out = append(out, messages(v.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}
	return []tea.Msg{m}
}

// TestCommandEffectsReachTheProgram: a workspace command's effect, its
// Result.Cmd, comes back to the program, through commandtea.Dispatch and
// through a key bound to the command (0020-PLAN Step 10).
func TestCommandEffectsReachTheProgram(t *testing.T) {
	panes := map[layout.PaneID]workspace.Pane{}
	for _, id := range []layout.PaneID{"session", "metrics", "logs", "footer"} {
		panes[id] = beacon{id}
	}
	ws := workspace.New(agentRoot(false, layout.FullWidth), panes)
	ws.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	r := command.NewRegistry()
	if err := r.Register(workspace.Commands(ws)...); err != nil {
		t.Fatal(err)
	}

	got := messages(commandtea.Dispatch(context.Background(), r, command.Request{
		ID: "workspace.focus", Args: []byte(`{"pane":"logs"}`), Origin: command.OriginKey, WhenContext: ws.WhenContext(),
	}))
	if !slices.Contains(got, tea.Msg(focusedMsg("logs"))) {
		t.Errorf("workspace.focus through commandtea.Dispatch sent %v, want the pane's focus message", got)
	}

	was := ws.Focused()
	got = messages(ws.Update(tea.KeyPressMsg{Code: '.', Mod: tea.ModAlt}))
	if now := ws.Focused(); now == was || !slices.Contains(got, tea.Msg(focusedMsg(now))) {
		t.Errorf("alt+. moved focus %s → %s and sent %v, want the new pane's focus message", was, now, got)
	}
}
