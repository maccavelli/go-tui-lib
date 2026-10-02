# Building multi-pane workspaces

For a Bubble Tea v2 program that wants several panes on one screen: a main
pane with a sidebar on either side, a bottom pane, a status footer, dialogs
and pop-ups over them. Two packages do the work:

- `layout` is pure geometry. It turns a tree of panes and a terminal size
  into rectangles, and holds what the user changed as JSON-ready state.
- `workspace` hosts the panes in your program. It lays them out, draws their
  chrome, routes keys and mouse events, moves focus, resizes, zooms and hides
  panes, opens overlays, and places the cursor.

Why they are built this way is in
[0002-MADR](../decisions/0002-MADR-multi-pane-workspace-layouts.md). A
complete, runnable session is `ExampleWorkspace_agentSession`
(`go doc -all github.com/maccavelli/go-tui-lib/workspace`).

## The program owns the program

The workspace is not a `tea.Model`. Your model holds it, passes it messages,
and builds the view:

```go
type model struct{ ws *workspace.Workspace }

func (m model) Init() tea.Cmd { return m.ws.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+c" {
        return m, tea.Quit // ctrl+c is yours; the workspace never binds it
    }
    return m, m.ws.Update(msg)
}

func (m model) View() tea.View {
    v := tea.NewView(m.ws.Render())
    v.Cursor = m.ws.Cursor()             // the focused pane's cursor, on screen
    v.AltScreen = true                   // your choice, never the workspace's
    v.MouseMode = tea.MouseModeCellMotion // turns the built-in mouse handling on
    return v
}
```

The workspace never sets the alternate screen, the mouse mode, focus
reporting or keyboard enhancements, and never installs a signal handler. Its
mouse handling is inert until you set a mouse mode.

## Choose a layout

Four presets build the common arrangements as ordinary `layout.Node` trees:

| Preset | Arrangement |
| :--- | :--- |
| `layout.SidebarRight(main, side)` | main pane, sidebar on the right |
| `layout.SidebarLeft(main, side)` | sidebar on the left, main pane |
| `layout.SidebarRightBottom(main, side, bottom)` | as `SidebarRight`, with a bottom pane |
| `layout.SidebarLeftBottom(main, side, bottom)` | as `SidebarLeft`, with a bottom pane |

Options change them:

- **Sizes.** `SidebarWidth(layout.Percent(30).AtLeast(24).AtMost(56))`,
  `BottomHeight(...)` and `MainSize(...)` take a claim with bounds.
- **`BottomSpan(layout.UnderMain)`** runs the bottom pane under the main pane
  only, so the sidebar keeps its full height. The default, `FullWidth`, runs
  it under both.
- **`Footer(id, rows)`** adds a status line under everything.
- **`Gap(n)`** sets the cells between panes. Use `Gap(0)` with
  `workspace.Borders`, and the default 1 with `workspace.Separators`.
- **`Breakpoints(fold, hide, hideBottom)`** sets the responsive folds:
  - below `fold` columns (default 100) the sidebar moves under the main pane;
  - below `hide` (default 70) it is hidden;
  - below `hideBottom` rows (default 16) the bottom pane is hidden.

  They measure the area above a footer. `NoResponsive()` turns them off.

For anything else, build the tree yourself:

```go
root := layout.Split{Name: "cols", Axis: layout.Horizontal, Gap: 1, Children: []layout.Child{
    {Node: layout.Pane{ID: "tree"}, Size: layout.Fixed(30).AtLeast(20)},
    {Node: layout.Pane{ID: "editor"}, Size: layout.Fill(2)},
    {Node: layout.Pane{ID: "preview"}, Size: layout.Fill(1).ShrinkFirst()},
}}
```

- **Size claims.** `Fixed`, `Percent`, `Ratio` and `Fill`, each with
  `AtLeast` and `AtMost`.
- **Shrinking.** When space runs short, children give up cells in `Shrink`
  order, lowest first (`ShrinkFirst`), down to their minimums. Then they
  hide, in the same order, rather than overlap.
- **Responsive.** `layout.Responsive{Rules: ...}` chooses a tree by size,
  with `MinWidth`, `MinHeight`, `And`, `Or` and `Not`.
- **Custom nodes.** A grid or a tab stack implements `layout.Node`, placing
  its leaves through `ctx.Place` and `ctx.Arrange`. It implements
  `layout.Leaver` so its hidden panes are reported.

Name a split (`Name: "cols"`) to make its separators resizable. They are
called `cols:0`, `cols:1`, and so on.

## Write a pane

A pane needs two methods:

```go
type Pane interface {
    Update(msg tea.Msg) (workspace.Pane, tea.Cmd)
    View(width, height int) string // exactly this many cells; extra is clipped
}
```

A pane learns its content size from `workspace.SizeMsg`, sent when it
changes. Optional interfaces add the rest:

| Interface | What it adds |
| :--- | :--- |
| `Titled` | a title in the chrome (default: the pane's ID) |
| `Badged` | a badge after the title, such as an unread count |
| `Focuser` | `Focus() tea.Cmd` and `Blur()` when focus moves |
| `Focusable` | `Focusable() bool`, false for a footer |
| `Sizer` | a minimum content size, which raises the layout minimum |
| `Cursorer` | the terminal cursor, in pane cells, while focused |
| `KeyMapper` | the pane's bindings, for a help footer |
| `Changer` | `Changed() bool`, false to reuse the last view |
| `EscConsumer` | for an overlay that handles `esc` itself |

Messages reach panes like this:

- keys go to the focused pane, after the workspace's own bindings;
- paste goes to the focused pane;
- other messages are broadcast to every pane;
- `workspace.To(id, msg)` from a command, or `ws.Send(id, msg)` directly,
  targets one pane.

## Keys and the mouse

`workspace.DefaultKeyMap()` uses alt, so tab, the arrows and alt+arrows stay
with your panes:

- `alt+]` and `alt+[` move focus, and `alt+1` to `alt+9` focus a pane;
- `alt+z` zooms the focused pane, and restores it;
- `alt+shift+arrows` move the nearest separator;
- `esc` closes the top overlay.

Rebind any of them with `SetKeys`, remove one with `Unbind`, and pass the
result with `WithKeyMap`.

With a mouse mode set:

- a click focuses the pane under the pointer;
- the wheel scrolls the pane under the pointer, focused or not;
- dragging a separator resizes;
- panes receive their mouse events in their own cells.

## Overlays

```go
ws.Push(workspace.Overlay{ID: "permission", Pane: dialog, Width: 40, Height: 7, Modal: true})
```

- **Modal.** A modal overlay takes every key and mouse event until it is
  closed, as a permission dialog or a picker does.
- **Non-modal.** A non-modal overlay, such as completion, takes only `esc`
  and the mouse events over it, so the focused editor keeps typing and
  drives it.
- **Placement.** `Anchor{Kind: workspace.BelowCursor}` places it under the
  cursor, and `Anchor{Kind: workspace.OnPane, Pane: id}` relative to a pane.
- **Closing.** `esc` or `ws.Pop()` closes it.

## Persist the layout

`ws.State()` returns what the user changed: separator moves, hidden panes,
the zoomed pane. It marshals to JSON. Store it in your settings and give it
back with `workspace.WithState(st)`. It is re-clamped at every size, so a
layout saved on a wide terminal opens safely on a narrow one. An unknown
version is refused.

## Chrome and theme

- **Chrome.**
  - `workspace.Borders` (the default) boxes each pane, with its title in the
    top edge.
  - `workspace.Separators` gives each pane a title row, and draws lines in
    the gaps.
  - `workspace.None` draws nothing.
  - `WithPaneChrome(id, chrome)` overrides one pane, such as `None` for a
    footer.
- **Theme.** Pass `workspace.WithTheme(theme.New(profile, background,
  glyph.For(utf8)))`. With no colour, the focused pane is still marked by
  the focus glyph and a bold title.
