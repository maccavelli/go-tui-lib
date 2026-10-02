---
status: accepted
date: 2026-10-01
decision-makers: owner
consulted: pi-go records 0004 and 0005 (the first consumer); go-core-lib 0004-MADR (the updatetea adapter); Charm v2 APIs (lipgloss v2.0.6, bubbletea v2.0.10, bubbles v2.2.1)
informed: pi-go; ocp-login; go-core-lib
---
# Build multi-pane terminal workspaces from a pure layout solver and a Bubble Tea pane host, with pi-go's agent session as the first consumer

## Context and Problem Statement

On 2026-10-01 the owner widened go-tui-lib's scope:

> expand the go-tui-lib scope to include support for multi-layout terminal
> windows, eg: main pane on left, narrow pane on right. main pane on right,
> narrow pane on left, and both of those with an included bottom pane. eg: an
> agent session in the main pane, the session metrics in the right or left
> sidebar, and logs in the bottom pane. i want this to be extensible,
> flexibile, future-proof, and provide supporting functionality to build
> professional, polished, and optimized TUI/UI/UX environments for a variety
> of uses and applications. its first consumer will be pi-go, as the primary
> agent session TUI.

The owner's standing direction also applies: build speculative API when it is
sensible, for extensibility, flexibility and idiomatic, modular design.

[0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)
§1 already puts layout and components in scope, and its §6 conventions
govern every package. This record decides the first packages: what they are,
how they divide the work, and the API's shape.

Evidence (read-only, 2026-10-01):

* **What the first consumer needs.**
  * **pi-go's TUI.** pi-go `docs/decisions/0005-MADR-v1-feature-scope.md`
    (`proposed`) lists:
    * a streamed Markdown transcript, collapsible tool cards with diffs, and
      a multi-line editor with history;
    * a permission dialog, model, thinking, mode and session pickers, and
      slash and `@path` completion;
    * Esc to interrupt, and a footer with model, mode, context % and cost.

    It decides "Charm v2, extracting shared widgets into go-tui-lib as they
    settle".
  * **pi-go's metrics and logs.** It plans token and duration metrics, cost
    estimates and structured logs (`slog` rolling files), which the owner's
    sidebar and bottom pane would show.
  * **pi-go's structure.** pi-go `docs/decisions/0004-MADR-go-module-architecture.md`
    (`proposed`) puts the TUI in `internal/tui`, an ACP client on
    `charm.land/…/v2`. Its import rule 5 lets only `internal/tui` import
    `charm.land/...`. pi-go has no `go.mod` or Go code yet.
* **What Charm v2 already provides** (from `go doc` on the released
  modules):
  * **Compositing (`lipgloss` v2.0.6).**
    * `Canvas` is a cell buffer that composes drawables in order and clips
      to its size.
    * `Layer` is a positioned block with an `ID`, `X`, `Y` and `Z`, and
      nested layers.
    * `Compositor` flattens a layer tree, renders it, and answers
      `Hit(x, y) LayerHit` with the layer's ID and bounds. That is exactly
      mouse hit-testing for panes and overlays.
    * `Border` is a struct of 13 glyph strings, so borders can come from a
      glyph table.
    * `Width` and `Height` measure in cells.
  * **The program (`bubbletea` v2.0.10).**
    * `Model.View() View`, where `View` carries `Content`, `Cursor` (a screen
      position, shape and blink), `AltScreen`, `MouseMode`, `ReportFocus`,
      `KeyboardEnhancements`, `WindowTitle` and `OnMouse`.
    * Input arrives as `WindowSizeMsg`, `KeyPressMsg`, `MouseClickMsg`,
      `MouseMotionMsg`, `MouseReleaseMsg`, `MouseWheelMsg`, `FocusMsg` /
      `BlurMsg` and `PasteMsg`.
  * **Scrolling (`bubbles` v2.2.1).** `viewport.Model` scrolls, pages, wheels,
    highlights and soft-wraps.
* **The working example.** ocp-login composes its interactive frame on a
  Lip Gloss canvas with layers at absolute rows, so content is clipped to
  its area, which joined strings cannot do (its `0011`). Its long-lived
  region uses the alternate screen, and its short-lived steps render
  inline. It is described by pattern only
  (0001-MADR amendment A2).
* **Prior art outside Go Charm** (design references only, nothing to import):
  * ratatui's `Layout`, with `Length` / `Min` / `Max` / `Percentage` /
    `Ratio` / `Fill` constraints;
  * tview's `Flex` and `Grid`;
  * lazygit's box layout, with weights and conditional children by width;
  * tmux and zellij, with pane splits, zoom, and keyboard and mouse resize.

  They converge on four ideas:
  * a tree of splits with per-child size constraints;
  * a solver that turns the tree and a size into rectangles;
  * focus that moves between leaves;
  * responsive rules that change the tree as the terminal shrinks.

## Decision Drivers

* **The owner's four layouts are presets, not the design.** Main and
  sidebar in either order, each with or without a bottom pane, must be
  ordinary values built from public parts. A program can then build a fifth
  without go-tui-lib's help.
* **Geometry is testable without a terminal.** Size allocation is where
  layouts go wrong: off-by-one rows, panes that vanish at 80 columns, a
  sidebar that eats the main pane. It must be pure, deterministic and
  property-tested.
* **The host is a good Bubble Tea citizen.** It embeds in a program's model,
  takes the program's messages, and never owns the screen, the signals or
  the process (0001-MADR §6, rules 1 and 2).
* **A polished feel needs host-level work that every program would
  otherwise repeat:**
  * focus that is visible without colour;
  * borders and titles from the glyph table;
  * keyboard and mouse resize;
  * zoom, collapse and hide;
  * overlays for dialogs and pickers;
  * the real terminal cursor in the focused editor;
  * mouse wheel to the pane under the pointer.
* **Performance under streaming.** An agent transcript and a log tail update
  many times a second. A pane whose content did not change must not be
  re-rendered.
* **Charm stays out of the geometry,** so the solver also serves a
  non-Bubble-Tea front end and outlives a Charm API change.
* **No new module.** Everything is built on the stack 0001-MADR §3 names.

## Considered Options

* **A. Two layers, `layout` (pure geometry) and `workspace` (a Bubble Tea pane host), on new foundation packages `glyph`, `theme` and `tuitest`, with the four layouts as presets.**
* **B. One package:** a workspace that hard-codes the sidebar and bottom-pane arrangements.
* **C. Adopt a community layout library** (for example a flexbox port) and build on it.
* **D. Leave layout to each program,** with only `lipgloss.JoinHorizontal` / `JoinVertical` helpers here.

## Decision Outcome

Chosen option: **"A"**, because:

* it is the only option in which the owner's four layouts are data rather
  than code;
* it keeps the part most likely to be wrong, geometry, pure and provable;
* it adds no module.

### 1. Packages and their dependencies

```text
 workspace     Bubble Tea pane host: panes, focus, routing, chrome, resize,
               zoom, overlays, cursor          → bubbletea, lipgloss, bubbles/key
 theme         semantic colours and styles; light, dark, unknown; NO_COLOR
                                               → lipgloss, colorprofile
 glyph         Unicode and ASCII glyph tables, borders included → stdlib
 layout        pure geometry: tree, constraints, solver, responsive rules,
               presets, serializable state     → stdlib only
 tuitest       golden rendering across the 0001 §6 matrix → stdlib, x/ansi
```

* **Imports point downward.** `workspace` imports `layout`, `glyph` and
  `theme`. `theme` imports `glyph` (for borders). `layout` imports nothing
  outside the standard library. `tuitest` is for tests and examples only.
* **No direct `ultraviolet` import.** `Canvas`, `Layer` and `Compositor`
  are reached through lipgloss, and `Hit` returns `image.Rectangle`.
* **Later packages build on these and are each their own record:**
  * standard panes: a log tail, a key/value metrics view, scrolling text;
  * overlay helpers: dialog, picker, command palette, toast;
  * a help footer;
  * `updatetea`.

### 2. `layout`: the geometry

```go
type Rect struct{ X, Y, W, H int }
type Axis uint8 // Horizontal (side by side), Vertical (stacked)

// Size is a child's claim on its split's main axis.
type Size struct {
    Kind     SizeKind // Fixed, Percent, Ratio, Fill
    N, D     int      // Fixed: N cells; Percent: N; Ratio: N/D; Fill: weight N
    Min, Max int      // cells; Max 0 means unbounded
    Shrink   int      // order in which children give up cells, lowest first
}
func Fixed(n int) Size; func Percent(p int) Size; func Ratio(n, d int) Size; func Fill(w int) Size
func (s Size) AtLeast(n int) Size; func (s Size) AtMost(n int) Size; func (s Size) ShrinkFirst() Size

// Node is a layout tree node. Split and Pane are the built-in kinds; a
// program can implement Node for its own arrangement (a grid, a stack).
type Node interface {
    Arrange(area Rect, ctx *Context) error // places leaves through ctx
}
type PaneID string
type Pane struct{ ID PaneID }
type Split struct {
    Axis     Axis
    Children []Child // each a Node with its Size
    Gap      int     // cells between children: separators or borders
}
type Child struct{ Node Node; Size Size }

// Responsive picks the first rule whose condition holds for the area.
type Responsive struct{ Rules []Rule; Else Node }
type Rule struct{ When Condition; Use Node }
func MinWidth(n int) Condition; func MinHeight(n int) Condition // and And, Or

func Solve(root Node, area Rect, st State) (Plan, error)
type Plan struct {
    Panes      map[PaneID]Rect // only visible panes
    Separators []Separator     // drawable, and draggable, boundaries
    Hidden     []PaneID        // dropped by a rule or by State
}
```

* **The solver is integer and deterministic.** Fixed and percentage claims
  are met first. The rest is shared by ratio, then by fill weight, with
  remainders distributed largest-first, so the parts always sum to the area.
  Min and Max are honoured. When the area is too small, children give up
  cells in `Shrink` order, and then hide, rather than overlap or overflow.
  `Solve` never returns a rectangle outside `area`.
* **Responsive rules** choose a different tree by size. For example: below
  100 columns the sidebar folds under the main pane; below 70 it hides;
  below 16 rows the bottom pane hides. The program writes the rules, or
  takes a preset's defaults.
* **`State` is the user's adjustments, as data:**
  * resize deltas per separator;
  * hidden panes and the zoomed pane;
  * a version number.

  It marshals to JSON, so a program can persist it, as pi-go's settings
  would. `Solve` applies it, and clamps it again at every size.
* **Presets** are functions returning ordinary trees:

  ```go
  func SidebarRight(main, side PaneID, o ...PresetOption) Node
  func SidebarLeft(main, side PaneID, o ...PresetOption) Node
  func SidebarRightBottom(main, side, bottom PaneID, o ...PresetOption) Node
  func SidebarLeftBottom(main, side, bottom PaneID, o ...PresetOption) Node
  // Options: SidebarWidth(Size), BottomHeight(Size),
  // BottomSpan(FullWidth | UnderMain), Footer(PaneID, rows),
  // Breakpoints(...), NoResponsive().
  ```

  * The bottom pane can span the full width or sit under the main pane
    only.
  * An optional footer row serves a status line such as pi-go's model,
    mode, context % and cost.
  * Default breakpoints are part of each preset and can be replaced.

### 3. `workspace`: the pane host

```go
// Pane is the one required interface: a Bubble Tea-style component that
// renders into the size it is given.
type Pane interface {
    Update(msg tea.Msg) (Pane, tea.Cmd)
    View(width, height int) string
}
// Optional capabilities, found by type assertion (as go-core-lib's
// selfupdate does for ReleaseLister):
type Titled interface{ Title() string }            // chrome title
type Badged interface{ Badge() string }            // e.g. unread log count
type Focuser interface{ Focus() tea.Cmd; Blur() }   // and Focusable() bool
type Sizer interface{ MinSize() (w, h int) }        // feeds layout Min
type Cursorer interface{ Cursor() *tea.Cursor }     // pane-local; host offsets it
type KeyMapper interface{ Keys() []key.Binding }    // for help footers
type Changer interface{ Changed() bool }            // skip re-render when false

type Workspace struct{ /* layout, panes by ID, focus, state, overlays */ }
func New(root layout.Node, panes map[layout.PaneID]Pane, o ...Option) *Workspace
func (w *Workspace) Update(msg tea.Msg) tea.Cmd
func (w *Workspace) Render() string          // the composed frame
func (w *Workspace) Cursor() *tea.Cursor     // focused pane's, in screen cells
func (w *Workspace) State() layout.State     // to persist
// Control: Focus(id), FocusNext/Prev, Toggle(id), Zoom(id), Resize(sep, delta),
// SetLayout(node), Push(Overlay)/Pop, Send(id, msg) → tea.Cmd, Broadcast(msg).
// Options: WithKeyMap, WithTheme, WithGlyphs, WithChrome(Borders|Separators|None),
// WithFocusRing(order), WithState(layout.State), WithMouse(bool).
```

* **The program owns the program.** `Workspace` is not a `tea.Model`. The
  program's `Update` calls `w.Update(msg)`, and its `View` puts
  `w.Render()` and `w.Cursor()` into its own `tea.View`. The program sets
  `AltScreen`, `MouseMode`, `ReportFocus` and keyboard enhancements: the
  workspace never does (0001 §6, rule 2). A workspace in the main buffer
  works; one on the alternate screen is the usual choice for an agent
  session, and it is the program's to make.
* **Routing.**
  * `WindowSizeMsg` re-solves the layout, and each pane is told its new
    size.
  * Keys go to the workspace's own bindings first: focus cycling, focus by
    number, resize, zoom, toggle, and overlay close. Every binding can be
    rebound or removed. Other keys go to the focused pane, or to the top
    overlay when one is open.
  * Mouse events are hit-tested on the composed layers. A click focuses,
    the wheel scrolls the pane under the pointer, and a drag on a separator
    resizes. Coordinates are translated to the pane.
  * `ctrl+c` is never consumed: it reaches the program, which decides
    (0001 §6, rule 2).
  * `Send` targets one pane, and `Broadcast` reaches all of them.
* **Chrome.**
  * Borders, separators and titles come from `glyph` and `theme`.
  * The focused pane is marked by a glyph and bold title as well as colour,
    so it stays visible under `NO_COLOR` (rule 4).
  * Titles truncate in cells, never mid-glyph (rule 5).
* **Composition.** Each visible pane renders into its rectangle on a
  `lipgloss.Canvas`, as a `Layer` with the pane's ID. Overlays are layers
  above. The canvas clips, so a pane cannot paint outside its rectangle.
* **Cursor.** The focused pane's `Cursorer` position is offset by the pane's
  rectangle. pi-go's multi-line editor then shows the real terminal cursor,
  which IME and screen readers need.
* **Optimization.** A pane implementing `Changer` that reports no change
  reuses its last render. Re-solving happens only on a size, layout or state
  change. The frame is composed once per `Render`, and Bubble Tea v2's cell
  renderer writes only changed cells.
* **Overlays.**
  * `Push(Overlay{ID, Pane, Anchor, Size, Modal})` places a pane above the
    layout: centred, or anchored to a pane or the cursor, as a completion
    pop-up is.
  * A modal overlay takes all keys until popped, which is the shape of
    pi-go's permission dialog and pickers.
  * Esc pops the top overlay, unless the overlay handles Esc itself.

### 4. `glyph`, `theme` and `tuitest`

* **`glyph`** is a `Set` with a Unicode and an ASCII form of every glyph
  the packages draw: borders (light, rounded, heavy, double, ASCII),
  separators, the focus marker, ellipsis, scroll indicators and badges.
  Every glyph is one cell. Tests prove that each ASCII form is ASCII and
  that every form is one cell wide (rule 3).
* **`theme`** layers raw palettes, semantic roles and built styles:
  * the roles are Title, Body, Muted, Border, BorderFocus, Accent, Success,
    Warning, Error and Badge;
  * `Theme` holds `Styles` for a colour profile and a background (light,
    dark or unknown);
  * the unknown palette is legible on both backgrounds;
  * `NO_COLOR` and the ASCII profile drop colour and keep structure
    (rule 4).
* **`tuitest`** renders a function across {colour, no colour} × {UTF-8,
  ASCII} × the widths a test names. It compares against golden files under
  the package's `testdata/golden/`, with an `-update` flag and a cell-width
  prefix per line (rule 6).

### 5. Versioning and the first consumer

* The first release is **`v0.1.0`**. `v0` allows API change while pi-go
  builds its TUI on it (owner question Q1). `apicheck` (0001 §5) arrives
  with the `v1` record.
* pi-go adopts it under its own records. Its import rule 5 must name
  `github.com/maccavelli/go-tui-lib`, which only pi-go can decide.
* The scaffold commits are pushed with this first package set, as
  0001 §8 says.

### 6. Provenance

go-tui-lib is a public Apache-2.0 repository. ocp-login's module lives on an
org-internal host. Its code is a design reference only, and nothing is copied
from it. Every package here is written from the public Charm APIs and this
record (owner question Q2).

### Consequences

* Good, because the owner's four layouts, and any fifth, are values a program
  can read, change, persist and test.
* Good, because the solver can be proven with property tests:
  * the parts tile the area;
  * Min and Max hold;
  * output is deterministic;
  * no rectangle escapes;
  * shrinking follows `Shrink` order.
* Good, because pi-go gets focus, resize, zoom, overlays, cursor and mouse
  routing without writing them, and so does every later consumer.
* Good, because `layout` has no Charm import and can serve another
  framework.
* Neutral, because `glyph`, `theme` and `tuitest` arrive before any
  consumer asked for them by name. Every later package needs them under
  0001 §6.
* Bad, because the first release is five packages. The PLAN lands them in
  dependency order, each proven before the next.
* Bad, because overlays, focus and mouse routing are a real event system.
  Their tests are the larger part of the work.
* Bad, because pi-go must amend its import rule before it can use this.

### Confirmation

* `layout`:
  * property tests over random trees and sizes prove the solver's
    invariants;
  * a fuzz target finds no panic and no escaping rectangle;
  * golden diagrams show each preset at 60, 80, 120 and 200 columns,
    including its responsive folds.
* `workspace`:
  * tests drive real `tea` messages: a resize, key focus cycling, a mouse
    click, a wheel event, a separator drag, zoom, toggle, a modal overlay,
    and the cursor offset;
  * a pane reporting no change is not re-rendered;
  * `ctrl+c` is never consumed.
* An agent-session example (main transcript, metrics sidebar on the right
  and then the left, logs bottom pane, footer) renders through `tuitest`
  across the 0001 §6 matrix at three widths.
* Mutation proofs for each package's key invariants. Pre-add, `-race` and
  the Windows host pass. CI is green on the push.

## Pros and Cons of the Options

### A. `layout` + `workspace` on foundations

* Good, because geometry, hosting and styling can each change without the
  others.
* Good, because every arrangement is data, and a custom `Node` extends it.
* Bad, because it is the largest first release.

### B. One hard-coded workspace

* Good, because the four layouts would ship soonest.
* Bad, because a fifth layout, a grid or a stacked tab set means changing
  the library.
* Bad, because geometry tests would need a terminal.

### C. A community layout library

* Good, because it would save writing a solver.
* Bad, because it is a new module, which needs its own record (0001 §3).
* Bad, because none found offers responsive rules, persisted state and hit
  regions together.
* Bad, because a Charm v1 dependency (most such libraries) is refused by
  `depguard`.

### D. Layout left to each program

* Good, because there is nothing to build.
* Bad, because it fails the owner's request. pi-go and every later program
  would rewrite focus, resize and overlays.

## Owner questions

*Answered 2026-10-01: "1. ship as v0.1.0 first. 2. design reference no
code. 3. both. 4. yes, on, inactive unless app turns it on. approved to
proceed."* On Q2's second part, the owner chose "Patterns only": this
repository describes ocp-login by pattern, not by file (0001-MADR
amendment A2). Every answer is the recommendation.

* **Q1. First version.** Recommended: `v0.1.0`, with `v1` once pi-go's TUI
  ships. The alternative is `v1.0.0` now, which freezes the API before its
  first consumer has used it.
* **Q2. ocp-login provenance.** Recommended: ocp-login is a design reference
  only, and no code is copied. A related question: may go-tui-lib's
  records, and the planned 0001 report, describe ocp-login's internal design
  at file level? Its module lives on an org-internal host and this
  repository is public. Recommended: describe patterns, not files, and
  re-word 0001-MADR's evidence to match before the first push.
* **Q3. The bottom pane.** Recommended: both spans, full width (the default)
  and under the main pane only, as a preset option.
* **Q4. Mouse.** Recommended: mouse support is on in the workspace and
  inert until the program sets `MouseMode`. The alternative is to leave it
  out of `v0.1.0`.

## More Information

* [0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md):
  §1 scope, §3 stack, §6 conventions, §8 push.
* pi-go `docs/decisions/0004-MADR-go-module-architecture.md` and
  `docs/decisions/0005-MADR-v1-feature-scope.md`: the first consumer.
* go-core-lib `docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md`
  §4: `updatetea`, a later package that will be a `workspace` pane or
  overlay as well as a stand-alone model.
