package workspace

import (
	"context"
	json "encoding/json/v2"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
)

// CommandOption configures Commands.
type CommandOption func(*commandConfig)

type commandConfig struct {
	layouts map[string]layout.Node
}

// WithLayouts names the layouts workspace.layout.use switches between. The
// command is registered only when at least one is named.
func WithLayouts(layouts map[string]layout.Node) CommandOption {
	return func(c *commandConfig) { c.layouts = maps.Clone(layouts) }
}

// Commands returns the workspace's own commands, closed over w, for a
// command.Registry (docs/decisions/0006-MADR-command-registry.md §9, A2,
// A8): focus and its cycle, zoom, toggle, resize, layouts, the layout
// state, the panes, the top overlay and the theme's background. Each runs
// on the event loop (command.Loop), on every surface but the shell, and
// changes only presentation (command.UI) unless it only reads
// (command.ReadOnly). A pane or split the workspace does not have is a
// *command.ArgError. The workspace's key bindings keep working beside
// them.
func Commands(w *Workspace, o ...CommandOption) []command.Command {
	var cfg commandConfig
	for _, f := range o {
		f(&cfg)
	}
	b := builder{w: w}
	cmds := []command.Command{
		b.focus(), b.next(), b.prev(), b.zoom(), b.toggle(), b.resize(),
		b.reset(), b.stateGet(), b.stateSet(), b.panes(), b.closeOverlay(), b.themeSet(),
	}
	if len(cfg.layouts) > 0 {
		cmds = append(cmds, b.layoutUse(cfg.layouts))
	}
	if b.err != nil {
		panic(b.err) // the definitions are fixed; this is a bug
	}
	return cmds
}

// builder makes the commands and keeps the first error.
type builder struct {
	w   *Workspace
	err error
}

// opts are the options every workspace command has, then extra.
func (b *builder) opts(slash string, d command.Danger, desc string, extra ...command.Option) []command.Option {
	o := []command.Option{
		command.WithDanger(d), command.WithDescription(desc), command.WithCategory("Workspace"),
		command.WithMode(command.Loop), command.WithSurfaces(command.AllSurfaces &^ command.SurfaceCLI),
	}
	if slash != "" {
		o = append(o, command.WithSlash(slash))
	}
	return append(o, extra...)
}

func newCommand[A any](b *builder, id command.ID, title string, run func(A) (command.Result, error), opts []command.Option) command.Command {
	c, err := command.New(id, title, func(_ context.Context, _ *command.Invocation, a A) (command.Result, error) {
		return run(a)
	}, opts...)
	if err != nil && b.err == nil {
		b.err = err
	}
	return c
}

// cmdResult runs a method that takes nothing, returning its command.
func cmdResult(c func() tea.Cmd) func(command.NoArgs) (command.Result, error) {
	return func(command.NoArgs) (command.Result, error) { return command.Result{Cmd: c()}, nil }
}

type paneArg struct {
	Pane string `json:"pane" arg:"" help:"the pane's ID" placeholder:"PANE"`
}

type zoomArg struct {
	Pane string `json:"pane,omitzero" arg:"" help:"the pane to zoom; the focused pane when empty" placeholder:"PANE"`
}

type resizeArg struct {
	Split string `json:"split" arg:"" help:"the split whose separator moves" placeholder:"SPLIT"`
	Delta int    `json:"delta" arg:"" help:"cells to give the pane before it; negative takes" schema:"min=-200,max=200"`
}

type layoutArg struct {
	Name string `json:"name" arg:"" help:"the layout's name" placeholder:"NAME"`
}

type stateArg struct {
	State layout.State `json:"state" help:"the layout state: zoom, hidden panes and resized splits"`
}

type themeArg struct {
	Background string `json:"background" arg:"" enum:"dark,light,auto" help:"the background to build the theme for; auto follows the terminal"`
}

// known reports whether the layout knows pane id: placed or hidden.
func (w *Workspace) known(id layout.PaneID) bool {
	_, placed := w.plan.Panes[id]
	return placed || slices.Contains(w.plan.Hidden, id)
}

func paneError(reason string) error {
	return &command.ArgError{Path: "/pane", Reason: reason}
}

func (b *builder) focus() command.Command {
	return newCommand(b, "workspace.focus", "Focus a pane", func(a paneArg) (command.Result, error) {
		if !b.w.focusable(layout.PaneID(a.Pane)) {
			return command.Result{}, paneError("is not a pane on screen that takes focus")
		}
		return command.Result{Cmd: b.w.Focus(layout.PaneID(a.Pane))}, nil
	}, b.opts("focus", command.UI, "Moves focus to a pane on screen.", command.WithArgHint("pane")))
}

func (b *builder) next() command.Command {
	return newCommand(b, "workspace.focus.next", "Focus the next pane", cmdResult(b.w.FocusNext),
		b.opts("next", command.UI, "Moves focus to the next pane of the focus ring."))
}

func (b *builder) prev() command.Command {
	return newCommand(b, "workspace.focus.prev", "Focus the previous pane", cmdResult(b.w.FocusPrev),
		b.opts("prev", command.UI, "Moves focus to the previous pane of the focus ring."))
}

func (b *builder) zoom() command.Command {
	return newCommand(b, "workspace.zoom", "Zoom a pane", func(a zoomArg) (command.Result, error) {
		id := layout.PaneID(a.Pane)
		if id == "" {
			id = b.w.focus
		}
		if !b.w.known(id) {
			return command.Result{}, paneError("is not a pane of the layout")
		}
		return command.Result{Cmd: b.w.Zoom(id)}, nil
	}, b.opts("zoom", command.UI, "Gives a pane the whole area, or restores the layout when it is zoomed already.",
		command.WithArgHint("pane")))
}

func (b *builder) toggle() command.Command {
	return newCommand(b, "workspace.toggle", "Hide or show a pane", func(a paneArg) (command.Result, error) {
		if !b.w.known(layout.PaneID(a.Pane)) {
			return command.Result{}, paneError("is not a pane of the layout")
		}
		return command.Result{Cmd: b.w.Toggle(layout.PaneID(a.Pane))}, nil
	}, b.opts("toggle", command.UI, "Hides a pane, or shows it when it is hidden.", command.WithArgHint("pane")))
}

func (b *builder) resize() command.Command {
	return newCommand(b, "workspace.resize", "Move a split", func(a resizeArg) (command.Result, error) {
		sep, err := b.w.separator(a.Split)
		if err != nil {
			return command.Result{}, err
		}
		return command.Result{Cmd: b.w.Resize(sep, a.Delta)}, nil
	}, b.opts("resize", command.UI, "Moves a named split's separator by delta cells.", command.WithArgHint("split delta")))
}

// separator resolves workspace.resize's split: a resizable separator's ID
// on screen, such as sidebar:0, or the name of a split with exactly one
// resizable separator on screen, such as sidebar
// (docs/decisions/0006-MADR-command-registry.md A8).
func (w *Workspace) separator(split string) (string, error) {
	var named []string
	for _, s := range w.plan.Separators {
		if !s.Resizable {
			continue
		}
		if s.ID == split {
			return s.ID, nil
		}
		if i := strings.LastIndexByte(s.ID, ':'); i >= 0 && s.ID[:i] == split {
			named = append(named, s.ID)
		}
	}
	switch len(named) {
	case 0:
		return "", &command.ArgError{Path: "/split", Reason: "is not a resizable split on screen"}
	case 1:
		return named[0], nil
	}
	return "", &command.ArgError{Path: "/split", Reason: "has " + strconv.Itoa(len(named)) + " separators on screen; name one of " + strings.Join(named, ", ")}
}

func (b *builder) layoutUse(layouts map[string]layout.Node) command.Command {
	c := newCommand(b, "workspace.layout.use", "Use a layout", func(a layoutArg) (command.Result, error) {
		root, ok := layouts[a.Name]
		if !ok {
			return command.Result{}, &command.ArgError{Path: "/name", Reason: "is not a layout's name"}
		}
		return command.Result{Cmd: b.w.SetLayout(root)}, nil
	}, b.opts("layout", command.UI, "Switches to a named layout.", command.WithArgHint("name")))
	schema, err := withEnum(c.Args, "name", slices.Sorted(maps.Keys(layouts)))
	if err != nil && b.err == nil {
		b.err = err
	}
	c.Args = schema
	return c
}

// withEnum is schema with property's enum set to values.
func withEnum(schema command.Schema, property string, values []string) (command.Schema, error) {
	var s map[string]any
	if err := json.Unmarshal(schema, &s); err != nil {
		return nil, err
	}
	props, ok := s["properties"].(map[string]any)
	p, isMap := props[property].(map[string]any)
	if !ok || !isMap {
		return nil, fmt.Errorf("workspace: the schema has no property %q", property)
	}
	p["enum"] = values
	return json.Marshal(s, json.Deterministic(true))
}

func (b *builder) reset() command.Command {
	return newCommand(b, "workspace.layout.reset", "Reset the layout", func(command.NoArgs) (command.Result, error) {
		return command.Result{Cmd: b.w.SetState(layout.State{})}, nil
	}, b.opts("", command.UI, "Restores the layout: no zoom, no hidden panes, no moved splits."))
}

func (b *builder) stateGet() command.Command {
	return newCommand(b, "workspace.state.get", "Get the layout state", func(command.NoArgs) (command.Result, error) {
		return command.Result{Value: b.w.State()}, nil
	}, b.opts("", command.ReadOnly, "Returns the layout state: zoom, hidden panes and moved splits."))
}

func (b *builder) stateSet() command.Command {
	return newCommand(b, "workspace.state.set", "Set the layout state", func(a stateArg) (command.Result, error) {
		old := b.w.State()
		cmd := b.w.SetState(a.State)
		if err := b.w.Err(); err != nil {
			b.w.SetState(old)
			return command.Result{}, &command.ArgError{Path: "/state", Reason: err.Error()}
		}
		return command.Result{Cmd: cmd}, nil
	}, b.opts("", command.UI, "Replaces the layout state; a state the layout cannot use is refused and the old one kept."))
}

// paneInfo is one pane as workspace.panes describes it.
type paneInfo struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Focused bool   `json:"focused"`
	Hidden  bool   `json:"hidden"`
}

func (b *builder) panes() command.Command {
	return newCommand(b, "workspace.panes", "List the panes", func(command.NoArgs) (command.Result, error) {
		infos := b.w.paneInfos()
		var text strings.Builder
		for _, p := range infos {
			fmt.Fprintf(&text, "%s  %s  %d,%d %dx%d", p.ID, p.Title, p.X, p.Y, p.Width, p.Height)
			if p.Focused {
				text.WriteString("  focused")
			}
			if p.Hidden {
				text.WriteString("  hidden")
			}
			text.WriteString("\n")
		}
		return command.Result{Value: infos, Text: text.String()}, nil
	}, b.opts("panes", command.ReadOnly, "Lists every pane of the layout, shown and hidden, with its title, rectangle and focus."))
}

// paneInfos is the panes in A8's order: the focus ring, the other placed
// panes in tree order, then the hidden ones.
func (w *Workspace) paneInfos() []paneInfo {
	var ids []layout.PaneID
	ids = append(ids, w.focusRing()...)
	for _, id := range w.plan.Order {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	ids = append(ids, w.plan.Hidden...)
	out := make([]paneInfo, 0, len(ids))
	for _, id := range ids {
		r, placed := w.plan.Panes[id]
		title := string(id)
		if t, ok := w.panes[id].(Titled); ok && t.Title() != "" {
			title = t.Title()
		}
		out = append(out, paneInfo{
			ID: string(id), Title: title, X: r.X, Y: r.Y, Width: r.W, Height: r.H,
			Focused: id == w.focus, Hidden: !placed,
		})
	}
	return out
}

func (b *builder) closeOverlay() command.Command {
	return newCommand(b, "workspace.overlay.close", "Close the overlay", func(command.NoArgs) (command.Result, error) {
		if len(b.w.overlays) == 0 {
			return command.Result{Text: "No overlay is open."}, nil
		}
		return command.Result{Cmd: b.w.Pop()}, nil
	}, b.opts("close", command.UI, "Closes the top overlay."))
}

// backgrounds maps workspace.theme.set's argument to a background.
var backgrounds = map[string]theme.Background{"dark": theme.Dark, "light": theme.Light, "auto": theme.Unknown}

func (b *builder) themeSet() command.Command {
	return newCommand(b, "workspace.theme.set", "Set the background", func(a themeArg) (command.Result, error) {
		return command.Result{Cmd: b.w.SetBackground(backgrounds[a.Background])}, nil
	}, b.opts("theme", command.UI, "Builds the theme for a dark or light background, or follows the terminal's again.",
		command.WithArgHint("dark|light|auto")))
}
