package workspace_test

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// counter is a pane that counts the messages it gets.
type counter struct{ n int }

func (c *counter) Update(tea.Msg) (workspace.Pane, tea.Cmd) {
	c.n++
	return c, nil
}

func (c *counter) View(int, int) string { return strconv.Itoa(c.n) }

// PaneAs returns a hosted pane as its own type, with no type assertion at
// the call site.
func ExampleWorkspace_PaneAs() {
	ws := workspace.New(layout.Pane{ID: "count"}, map[layout.PaneID]workspace.Pane{"count": &counter{}})
	ws.Send("count", "tick")
	ws.Send("count", "tick")
	if c, ok := ws.PaneAs[*counter]("count"); ok {
		fmt.Println("count:", c.n)
	}
	_, ok := ws.PaneAs[*counter]("missing")
	fmt.Println("missing:", ok)
	// Output:
	// count: 2
	// missing: false
}
