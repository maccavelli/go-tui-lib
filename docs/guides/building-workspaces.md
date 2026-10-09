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
    v := m.ws.View()                      // the frame, and the cursor on screen
    v.AltScreen = true                    // your choice, never the workspace's
    v.MouseMode = tea.MouseModeCellMotion // turns the built-in mouse handling on
    return v
}
```

`ws.View()` is `tea.NewView(ws.Render())` with `Cursor` set from
`ws.Cursor()`, and sets nothing else. The workspace never sets the alternate
screen, the mouse mode, focus reporting or keyboard enhancements, and never
installs a signal handler. Its mouse handling is inert until you set a mouse
mode.

The program also owns how the TUI starts. When your program has its own
command line, `launch` decides whether the TUI can run on its streams,
runs it with `launch.WithRegistry` so the workspace's commands reach the
loop, and falls back to your CLI mode when it cannot start or crashes:
see [Start the TUI from your CLI](commands.md#start-the-tui-from-your-cli).

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
called `cols:0`, `cols:1`, and so on. An unnamed split's separators are
named by position, which changes when the tree does, so they cannot be
resized.

Split names follow two rules, and `layout.Solve` returns
`layout.ErrBadSplitName` when one is broken:

- **A name is used once in a solve.** Two splits with one name would share
  their separators' IDs, so one resize would move both. Each alternative of
  a `layout.Responsive` is solved alone, so the same name may appear in
  several of its rules; that is how a resize survives a fold.
- **A name never starts with `/`.** That prefix is reserved for the
  positional IDs of unnamed splits.

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
| `Focuser` | `Focus() tea.Cmd` and `Blur()` when focus moves, for a pointer pane |
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

A `Workspace` is not safe for concurrent use. Call it from your model's
`Update` and `View`, which Bubble Tea runs on one goroutine. Commands run
elsewhere, and reach the workspace only as the messages they return.

The workspace draws a frame only when something changed: its size, the
layout or its state, focus, an overlay, the theme, the width method, or a
message delivered to a pane. Otherwise `Render` returns the last frame, at
no cost. So a pane's view changes through its `Update`, or, for a
`Changer`, whenever `Changed` reports it. A pane changed some other way,
such as by your program writing to a pointer pane directly, is drawn anew
at the next of those changes; send it a message instead.

To get a pane back as its own type, use `ws.PaneAs`, which also finds an
open overlay's pane when no pane has the ID:

```go
if log, ok := ws.PaneAs[*logPane]("logs"); ok {
    log.Clear()
}
```

`ws.Panes()` walks the panes focus moves through, in that order, and
`ws.Plan().All()` walks every placed pane with its rectangle.

## Focus

One pane has the keyboard: the focused pane, or the top modal overlay while
one is open. When that changes, the pane losing it gets
`workspace.PaneBlurMsg` and the pane gaining it gets
`workspace.PaneFocusMsg`, through `Update`. The workspace keeps the value
`Update` returns, so a pane with value semantics sees its own focus:

```go
type list struct {
    items   []string
    focused bool
}

func (l list) Update(msg tea.Msg) (workspace.Pane, tea.Cmd) {
    switch msg.(type) {
    case workspace.PaneFocusMsg:
        l.focused = true
    case workspace.PaneBlurMsg:
        l.focused = false
    }
    return l, nil
}

func (l list) View(width, height int) string { return strings.Join(l.items, "\n") }
```

The first pane is told in `ws.Init()`. Focus moves on the focus keys, a
click, `ws.Focus(id)`, hiding the focused pane, and opening or closing a
modal overlay. `ws.Focus(id)` while a modal overlay is open changes the
focused pane, which is told when the overlay closes.

A pointer pane may implement `Focuser` instead; its `Focus` and `Blur` are
called first, and it gets the messages too. Implement one or the other,
not both.

## Host a bubbles component

`workspace.Wrap` hosts a bubbles component, or any model built like one,
as a pane:

```go
ti := textinput.New()
ti.SetVirtualCursor(false) // the terminal's cursor, which ws.Cursor() places

ws := workspace.New(layout.SidebarRight("chat", "log"), map[layout.PaneID]workspace.Pane{
    "chat": workspace.Wrap(ti),
    "log":  workspace.Wrap(viewport.New()),
})
```

`Wrap` finds the component's own methods:

- `SizeMsg` calls `SetSize(width, height)`, or `SetWidth` and `SetHeight`;
- `PaneFocusMsg` calls `Focus()`, and `PaneBlurMsg` calls `Blur()`;
- the pane's cursor is the component's `Cursor()`, and its bindings are
  `Keys()`;
- every other message reaches the component's `Update`, and its `View` is
  clipped to the pane.

`OnSize`, `OnFocus`, `OnBlur`, `WithCursor` and `WithKeys` replace any of
them. Read the component back through the pane:

```go
if p, ok := ws.Pane("chat"); ok {
    value := p.(*workspace.Model[textinput.Model]).M.Value()
    _ = value
}
```

`textinput` and `textarea` draw a virtual cursor in their view by default,
and then report none. Call `SetVirtualCursor(false)` for the terminal's
cursor, as above.

## Keys and the mouse

`workspace.DefaultKeyMap()` uses alt, so tab, the arrows and alt+arrows stay
with your panes:

- `alt+.` and `alt+,` move focus, and `alt+1` to `alt+9` focus a pane;
- `alt+z` zooms the focused pane, and restores it;
- `alt+shift+arrows` move the nearest resizable separator;
- `esc` closes the top overlay.

Without an enhanced keyboard protocol a terminal sends alt+x as ESC and x.
ESC followed by `[`, `]`, `O`, `P`, `_`, `^`, `X` or `\` also starts a
control sequence, so those keys cannot be told reliably from one. No
default uses alt with them; avoid them when you rebind. (`v0.1.0` used
`alt+]` and `alt+[`.)

Rebind any of them with `SetKeys`, remove one with `Unbind`, and pass the
result with `WithKeyMap`.

The same actions are also commands, `workspace.Commands(ws)`, for a
palette, slash lines and agents: see
[the workspace's commands](commands.md#the-workspaces-commands).

The workspace is a `help.KeyMap`, and `ws.Help()` is a bubbles help model
drawn with the workspace's glyphs and styles, so a help footer is one call:

```go
footer := ws.Help().View(ws)
```

`Help` takes the short help's separator from the glyph set's `Bullet` and
its ellipsis from its `Ellipsis`, so the ASCII set draws `*` and `~`
where `help.New()` draws `•` and `…`. Call `SetWidth` on it, as on any
help model. The short help lists the bindings of whoever has the keyboard, the top
modal overlay's pane or else the focused pane, from its `KeyMapper`, then
focus-next and zoom. The full help has four columns: those bindings, focus,
layout and overlays. Disabled bindings are left out.

With a mouse mode set:

- a click focuses the pane under the pointer;
- the wheel scrolls the pane under the pointer, focused or not;
- dragging a separator resizes;
- panes receive their mouse events in their own cells.

A resize, by key, by drag or by `ws.Resize(sep, delta)`, moves the
separator from where it is drawn, as far as the panes' bounds allow. The
state keeps what moved, not what was asked for, so after holding a key or
dragging past a limit, moving back responds at once. A press that moves
nothing changes nothing. A window resize never changes the state, so a
layout squeezed by a small window comes back when the window grows.

## Overlays

```go
return m, ws.Push(workspace.Overlay{ID: "permission", Pane: dialog, Width: 40, Height: 7, Modal: true})
```

`Push` returns a `tea.Cmd` that tells the overlay its size and moves the
focus, so `Update` returns it rather than dropping it.

- **Modal.** A modal overlay takes every key and mouse event until it is
  closed, as a permission dialog or a picker does.
- **Non-modal.** A non-modal overlay, such as completion, takes only `esc`
  and the mouse events over it, so the focused editor keeps typing and
  drives it.
- **Placement.** `Anchor{Kind: workspace.BelowCursor}` places it under the
  cursor, and `Anchor{Kind: workspace.OnPane, Pane: id}` relative to a pane.
- **Size.** An overlay's pane gets `SizeMsg` with its content size when it
  opens, and again whenever a resize changes it.
- **Focus.** A modal overlay takes the keyboard: the focused pane gets
  `PaneBlurMsg`, and the overlay's pane `PaneFocusMsg`; closing it gives
  the keyboard back. A non-modal overlay changes no focus.
- **IDs.** Overlay IDs are their own namespace, apart from pane IDs, so an
  overlay may share a pane's ID. Pushing an ID that is already open
  replaces that overlay, on top. `ws.SendOverlay(id, msg)` reaches an open
  overlay only; `ws.Send(id, msg)` tries a pane first, then an overlay.
- **Closing.** `esc` or `ws.Pop()` closes it.

## Text width

Terminals disagree about how wide some characters are: an emoji joined with
zero-width joiners, or a symbol with the VS16 selector, is one width under
wcwidth and another under grapheme clustering. The workspace measures as
Bubble Tea writes, so borders after such text line up:

- it starts with `ansi.WcWidth`, as Bubble Tea's renderer does;
- when the terminal reports grapheme clustering (mode 2027), it switches to
  `ansi.GraphemeWidth`, under the same rule Bubble Tea follows; the report
  still reaches your panes;
- `WithWidthMethod(m)` fixes the method and stops it following.
  `WithWidthMethod(ansi.GraphemeWidth)` measures as `v0.1` did.

A pane that pads or truncates its own text can measure the same way with
`ws.WidthMethod()`, such as `ws.WidthMethod().StringWidth(s)`.

## Persist the layout

`ws.State()` returns what the user changed: separator moves, as applied,
hidden panes, the zoomed pane. It marshals to JSON. Store it in your settings and give it
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
  - A separator is drawn when a pane beside it has `Separators` chrome;
    one between two `None` or two `Borders` panes stays blank. Where
    separators meet, three lines take a tee (`┬ ┴ ├ ┤`) and four the cross
    (`┼`), or `+` in the ASCII glyph set.
- **Theme.** By default the workspace follows the terminal. It starts with
  ANSI256, an unknown background and Unicode glyphs, asks for the
  background in `Init`, and rebuilds its theme on each
  `tea.ColorProfileMsg` and `tea.BackgroundColorMsg`, which still reach
  your panes.
  - `WithGlyphs(g)`, `WithProfile(p)` and `WithBackground(bg)` change what
    it starts from, such as the glyph tier and profile `launch` decided.
    `WithBackground` is where it starts, not a pin: a background message
    replaces it, and `SetBackground` is the pin.
  - `WithSize(width, height)` is the size it lays out at before the first
    `tea.WindowSizeMsg` (80 × 24 by default), such as for a plain render.
  - `WithTheme(theme.New(profile, background, glyph.For(utf8)))` fixes the
    theme instead; the messages then change nothing. Its theme is the
    first, whatever the options' order, and `WithGlyphs`, `WithProfile`
    and `WithBackground` are then ignored. A `WithThemeBuilder` after it
    makes the workspace follow from that theme.
  - `WithThemeBuilder(b)` builds the first theme, and every rebuild, with
    your own builder, such as one that adds `theme.WithPaletteFor` for
    each background, so your colours survive a background change. It
    chooses its own glyphs.
  - `WithGlyphThemeBuilder(b)` is the same for a builder that is given
    the glyphs in use, `WithGlyphs`'s at first. It wins over a
    `WithThemeBuilder`.
  - `WithoutBackgroundQuery()` leaves the query out of `Init`, for a
    program that runs its own probe.
  - `ws.SetTheme(t)` replaces the theme now; a following workspace builds
    over it at the next message.
  - A palette colour may itself depend on the background or the profile:
    `theme.LightDarkColor{Light, Dark, Unknown}` and
    `theme.ProfileColor{ANSI, ANSI256, TrueColor}` are picked when the
    theme is built.

  With no colour, the focused pane is still marked by the focus glyph and a
  bold title.

## Plain output

`ws.RenderPlain(width)` writes the workspace for a pipe, a log or a screen
reader: one pane after another, each as its title and badge on one line,
its content, and a blank line. It has no borders, no colour and no escape
sequences, and no trailing spaces.

- **The order** is the focus ring's (`WithFocusRing`'s, else the tree's),
  then the visible panes the ring leaves out, such as a footer that
  refuses focus. Hidden and zero-size panes, and overlays, are left out.
- **The content** is the pane's `PlainView(width)` when it is a
  `workspace.PlainViewer`, and otherwise its `View` at `width` × its
  content height, with escape sequences removed. A pane `Wrap` hosts
  forwards its model's own `PlainView`.
- **It changes nothing:** it lays the workspace out at `width` × the
  current height on its own, and leaves the layout, the panes' sizes,
  the frame and the view cache as they were.

With `launch`, a program that decides on plain output can still build the
workspace, give it `WithSize` and the decided glyphs, and print
`RenderPlain`.
