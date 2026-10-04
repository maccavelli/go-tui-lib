---
status: accepted
date: 2026-10-02
decision-makers: owner
consulted: the v0.1.0 audit in 0003-REPORT-agent-tui-ecosystem-research.md §1; Charm v2 sources (lipgloss v2.0.6, bubbletea v2.0.10, bubbles v2.2.1, ultraviolet at the pinned pseudo-version); the Go 1.27.1 toolchain
informed: pi-go
---
# Draw the workspace on an ultraviolet cell buffer, follow the terminal's width method and theme, and adopt Go 1.27 idioms in the public API

## Context and Problem Statement

On 2026-10-01 the owner asked:

> i want to stay working on go-tui-lib, i want to ensure the
> bubbletea/lipgloss/charm stack is tightly integrated, coded idiomatically,
> based on go1.27.1 optimizations and standards. i want to enhance and expand
> the tui library functionality in 5 more ways that will bring benefit and
> value to the codebase functionality. … future-proof is a focus.

On 2026-10-02, after the research was presented, the owner said: "write
findings into a report then follow recommendations and proceed." The
recommendation put integration with the Charm v2 stack and Go 1.27 first,
ahead of the five expansions. The audit's confirmed defects are fixed in
[0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md),
which ships across `v0.1.1` to `v0.1.5` (its deviations D3 and D6). *Amended
2026-10-04 (A1): `v0.1.6` completes it (that PLAN's deviation D7).* This record
decides what comes next: the changes that alter how `workspace` draws,
measures and styles, and the API they add.
The owner's standing direction also applies: build speculative API when it
is sensible, for extensibility, flexibility and idiomatic, modular design.

Evidence (read-only, 2026-10-02):

* **Where a frame's cost goes.** The audit profiled `BenchmarkRender`
  (200 × 60, four panes) on the macOS development host
  ([0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §1):
  * the baseline is 1.55 ms, 1.74 MB and 1091 allocations per frame;
  * 85% of the bytes come from the new 200 × 60 `lipgloss.Canvas` that
    `Render` builds every frame (`render.go:42`);
  * `NewLayer`, `NewCompositor` and the compositor's `flatten` each measure
    every pane with `lipgloss.Width`, so each frame is scanned for graphemes
    three times;
  * `renderBox` renders the left and right edge glyphs once per row;
  * the cache key is built with `fmt.Sprintf` (`render.go:79`).
* **What the audit's scratch experiment achieved.** Reusing the canvas,
  rendering edge glyphs once per box, a struct cache key, and drawing panes
  directly with ultraviolet with a hit test over the plan's rectangles took
  the frame to 0.79 ms, 173 KB and 593 allocations. In TrueColor the frame
  went from 1.92 ms to 0.97 ms and from 1.86 MB to 253 KB. Every golden
  test still passed byte for byte.

  | Step | Time/op | Bytes/op | Allocs/op |
  | :--- | :--- | :--- | :--- |
  | Baseline | 1.55 ms | 1.74 MB | 1091 |
  | Reuse canvas | 1.27 ms | 225 KB | 958 |
  | + edge glyphs once per box, struct cache key | 1.13 ms | 178 KB | 611 |
  | + direct draw, manual hit test | 0.79 ms | 173 KB | 593 |

* **lipgloss's canvas fixes the width method.** In lipgloss v2.0.6,
  `NewCanvas` sets `c.scr.Method = ansi.GraphemeWidth` (`canvas.go:27`) on an
  unexported `uv.ScreenBuffer`. Nothing in lipgloss's API changes it.
* **Bubble Tea's renderer starts elsewhere.** In bubbletea v2.0.10:
  * the renderer's buffer is `uv.NewScreenBuffer` (`cursed_renderer.go:48`),
    whose `Method` is `ansi.WcWidth` (ultraviolet `buffer.go:614-618`);
  * on `ModeReportMsg` for `ansi.ModeUnicodeCore` (mode 2027) with
    `ModeReset`, `ModeSet` or `ModePermanentlySet`, it switches to
    `ansi.GraphemeWidth` (`tea.go:802-805`);
  * it sends that query only when `shouldQuerySynchronizedOutput` holds for
    the environment (`tea.go:980-994`, `1118-1123`);
  * the same message then reaches the model's `Update` (`tea.go:880`).

  So on a terminal without mode 2027, the workspace composes with grapheme
  widths and Bubble Tea writes with wcwidth. An emoji with a zero-width
  joiner or a VS16 selector measures differently in each, and borders after
  it misalign. The fix needs the workspace to own its buffer's method.
  [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  defers this defect (audit item 6) to this record for that reason.
* **ultraviolet, as pinned.**
  * `github.com/charmbracelet/ultraviolet` is in go-tui-lib's graph today as
    an indirect requirement at `v0.0.0-20260811164956-006e29f97886`. That
    is lipgloss v2.0.6's requirement. bubbletea v2.0.10 requires an older
    pseudo-version, `v0.0.0-20260703014108-f5a850f9c2b7`, and minimum version
    selection chooses the newer.
  * It has **no tagged release**: `go list -m -versions` lists none. Every
    version is a pseudo-version of a commit.
  * Its licence is MIT (Charmbracelet, Inc.), as lipgloss's is.
  * At that version it offers `ScreenBuffer{*RenderBuffer; Method
    ansi.Method}` with `Resize`, `Clear` and `Render`; `NewStyledString(s)`
    with `Draw(Screen, Rectangle)`, which clears the area and then prints
    the string clipped to it; the `Screen` interface; and `TrimSpace`.
    `lipgloss.Layer.Draw` is exactly `uv.NewStyledString(l.content).Draw`
    (`layer.go:153-156`), and `Canvas.Render` is
    `uv.TrimSpace(c.scr.Render())` (`canvas.go:87-89`). Drawing directly is
    the same code without the layer and compositor around it.
* **Theme messages the workspace ignores today.**
  * Bubble Tea sends `ColorProfileMsg` at start (`tea.go:1098`), and again
    when an `RGB` or `Tc` capability reply upgrades the profile to
    TrueColor (`tea.go:784-791`).
  * `BackgroundColorMsg` arrives when the program runs
    `tea.RequestBackgroundColor`, and its `IsDark()` answers light or dark.
  * lipgloss v2.0.6 offers `LightDark(isDark) LightDarkFunc` and
    `Complete(profile) CompleteFunc` for colours that depend on the
    background and the profile.
  * `theme.New` builds once, and `Workspace` has no `SetTheme`. The cache key
    has no theme in it, so a new theme would show stale views.
* **Help.** bubbles v2.2.1's `help.KeyMap` is `ShortHelp() []key.Binding`
  and `FullHelp() [][]key.Binding`. `workspace.KeyMapper` exists, and
  nothing reads it (audit item 10).
* **Chrome gaps.** `glyph.Set.SeparatorCross` is never drawn, and
  `renderSeparator` reads only the workspace's chrome, ignoring
  `WithPaneChrome` (`render.go:193`).
* **Go 1.27.1.**
  * Generic methods compile: a probe with
    `func (w *W) PaneAs[T Pane](id string) (T, bool)` built and ran under
    `go 1.27.1`. An interface method with type parameters is refused
    ("interface method must have no type parameters").
  * golangci-lint v2.14.0 (built with go1.27.1) and govulncheck v1.8.0 both
    ran clean over that probe.
  * `iter.Seq2`, range-over-func and `strings.CutLast` are available.
  * Today a caller gets a pane back as the `Pane` interface and type-asserts
    it, and walks `Plan.Order` and `Plan.Panes` by hand.

## Decision Drivers

* **A frame must be cheap while an agent streams.** A transcript and a log
  tail update many times a second. Allocation, not drawing, dominates the
  frame today.
* **The workspace must measure as Bubble Tea writes.** A border that
  misaligns on one emoji is visible polish lost on every agent transcript.
* **The theme must follow the terminal.** A background or profile reply
  must restyle the frame without the program rebuilding the workspace.
* **Charm-native interfaces over our own.** Help through `help.KeyMap`,
  frames through `tea.View`, colours through `LightDark` and `Complete`.
* **Idiomatic Go 1.27 at the API.** Typed access and iterators where a
  caller now writes assertions and loops.
* **An unversioned dependency must stay contained.** ultraviolet has no
  semver promise. Its use must be small enough to adapt in one place.
* **Nothing breaks without a reason.** Existing goldens stay byte for byte
  identical at the default width method, and v0.1.x code still compiles.

## Considered Options

* **A. Draw directly on a reused `uv.ScreenBuffer`, behind one internal package; follow the width method and theme messages; add the Go 1.27 accessors.**
* **B. Stay on `lipgloss.Canvas` and `Compositor`, and only reuse the canvas and cut the allocations.**
* **C. Write our own cell buffer and ANSI parser, with no ultraviolet import.**
* **D. Leave rendering as it is, and add only the theme, help and Go 1.27 API.**

## Decision Outcome

Chosen option: **"A"**, because:

* it is the only option that fixes the width-method mismatch, since
  lipgloss's canvas cannot change its method;
* it reaches the audit's measured frame, 0.79 ms and 173 KB, where B stops at
  about 1.13 ms;
* it uses the same ultraviolet code lipgloss already runs, so it adds no new
  behaviour, only a direct edge to a module already in the graph.

### 1. ultraviolet as a direct requirement, confined to `internal/cells`

* **The requirement.** `github.com/charmbracelet/ultraviolet` moves from
  indirect to direct at the version the graph already selects,
  `v0.0.0-20260811164956-006e29f97886`. It is MIT-licensed. It is named
  here as AGENTS.md's Dependencies section requires, and AGENTS.md's list
  gains it.
* **What a pseudo-version means here.**
  * There is no semver promise. Any commit may change an exported name.
  * We never pick the version ourselves. It moves only when a lipgloss or
    bubbletea upgrade raises it, and that upgrade is already a reviewed
    change.
  * `go mod tidy -diff` stays clean, because the version is the one
    minimum version selection already chooses.
  * If upstream tags a release, the next Charm upgrade adopts it, with no
    record needed.
* **Containment.** One new internal package, `internal/cells`, is the only
  importer of ultraviolet:

  ```go
  package cells

  // Frame is a reusable cell buffer with a width method.
  type Frame struct{ /* uv.ScreenBuffer */ }
  func NewFrame(w, h int, m ansi.Method) *Frame
  func (f *Frame) Resize(w, h int)            // no-op when unchanged
  func (f *Frame) SetMethod(m ansi.Method)
  func (f *Frame) Method() ansi.Method
  func (f *Frame) Clear()
  func (f *Frame) Draw(s string, r layout.Rect) // clears r, prints s clipped to it
  func (f *Frame) Render() string             // trailing spaces trimmed
  ```

  * A `depguard` rule denies ultraviolet in every file outside
    `internal/cells`. A later package that needs it widens the rule in its
    own record.
  * An ultraviolet API change is then one file's fix.
* **Not adopted.** No ultraviolet type appears in any exported API.

### 2. Rendering

* **One frame buffer, reused.** `Workspace` keeps a `*cells.Frame`. `Render`
  clears it, and resizes it only when the size changes.
* **Direct drawing.** Each pane, separator and overlay is drawn with
  `Frame.Draw` into its rectangle, in z-order: panes, then separators,
  then overlays. `lipgloss.Layer` and `lipgloss.Compositor` are no longer
  used.
* **Hit testing over rectangles.** `Render` records a slice of
  `{kind, id, rect, inner}` regions, top first. A mouse event takes the
  first region containing the point. This is what `Compositor.Hit` did,
  without three passes of width measurement.
* **Less work per box.** The left and right edge glyphs are styled once per
  box, not per row. Lines are walked with `strings.SplitSeq`.
* **A struct cache key.** The view cache's key is
  `{kind, id, width, height, focused, method, themeGen}`, with no
  `fmt.Sprintf`. *2026-10-04 (A1, Q5): the owner kept this key, in place
  of the hardening's `{kind, id}`; dropping a pane's or an overlay's view
  drops every entry with its kind and ID.* `themeGen` rises on every theme change (§4). The 0002
  hardening rules stay: overlays and panes have separate kinds, and entries
  are dropped on `SetPane` and `Pop`.
* **An unchanged frame is returned as is.** The workspace keeps a dirty
  flag. A size, layout, state, focus, overlay, theme or method change sets
  it, and so does any message delivered to a pane that is not a `Changer`
  or reports a change. When nothing is dirty, `Render` returns the last
  frame's string without drawing.
* **`Broadcast` keeps a sorted ID slice,** rebuilt when panes are added or
  replaced, rather than sorting on every message.
* **Output is unchanged.** At the default method, every existing golden
  file stays byte for byte identical. That is a gate (§7), not a hope.

### 3. The width method

* **Default `ansi.WcWidth`,** which is what Bubble Tea's renderer starts
  with.
* **It follows Bubble Tea's own rule.** On
  `tea.ModeReportMsg{Mode: ansi.ModeUnicodeCore}` with `Value` `ModeReset`,
  `ModeSet` or `ModePermanentlySet`, the workspace switches to
  `ansi.GraphemeWidth`. Any other value leaves the method alone. The
  message is still delivered to the panes.
* **The program can fix it.** `WithWidthMethod(m ansi.Method)` sets the
  method and stops it following mode reports.
  `(*Workspace).WidthMethod() ansi.Method` reports the current one, so a
  pane can measure the same way.
* **Every measurement uses it:** the clip and pad of a view, title and
  badge truncation, separator and border lengths, and the `Sizer`
  arithmetic. Each call to `ansi.StringWidth` or `ansi.Truncate` becomes
  `w.method.StringWidth` or `w.method.Truncate`.
* **A method change** invalidates the view cache and marks the frame dirty.

### 4. A theme that follows the terminal

* **`(*Workspace).SetTheme(t theme.Theme) tea.Cmd`** replaces the theme,
  raises `themeGen` and marks the frame dirty.
* **Following, by default.** The workspace keeps a theme builder:

  ```go
  // ThemeBuilder makes the theme for a profile and a background.
  type ThemeBuilder func(p colorprofile.Profile, bg theme.Background) theme.Theme

  func WithThemeBuilder(b ThemeBuilder) Option // default: theme.New with the glyphs in use
  ```

  * On `tea.ColorProfileMsg` it rebuilds with the new profile.
  * On `tea.BackgroundColorMsg` it rebuilds with
    `theme.FromDark(msg.IsDark())`.
  * `WithTheme(t)` keeps its meaning: a fixed theme, which messages do not
    change. `WithThemeBuilder` turns following back on.
  * Both messages still reach the panes, so a pane with its own styles can
    follow too.
* **The workspace asks for the background by default.** `Init` includes
  `tea.RequestBackgroundColor`, so a following theme gets its light or dark
  answer without the program doing anything (owner question Q2).
  `WithoutBackgroundQuery()` leaves the query out, for a program that runs
  its own probe or must send nothing unasked.
  * Once `termcap` exists, the query follows 0005's gates. Some terminals
    paint a query as text (JetBrains), and an editor's `:terminal` answers
    for the editor rather than the user's terminal
    ([0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
    §8.3). Until then the query is unconditional.
  * [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md)
    adds a full probe, and live light and dark switching, on top of this.
* **`theme` gains adaptive colours,** with no Bubble Tea import:

  ```go
  func FromDark(isDark bool) Background // Dark or Light

  // LightDarkColor picks by background through lipgloss.LightDark. On an
  // Unknown background it uses Unknown, or Dark when Unknown is nil.
  type LightDarkColor struct{ Light, Dark, Unknown color.Color }

  // ProfileColor picks by profile through lipgloss.Complete.
  type ProfileColor struct{ ANSI, ANSI256, TrueColor color.Color }

  func WithPaletteFor(bg Background, p Palette) Option // one per background
  ```

  * Both types implement `color.Color`, so they fit today's `Palette`
    fields, and existing palettes keep working.
  * `build` resolves them for the theme's background and profile. A
    `ProfileColor` is not then converted again.
  * `WithPaletteFor` keeps a custom palette per background, so a rebuild
    after a background change keeps the program's colours. `WithPalette`
    stays, and means the palette for every background.

### 5. Charm-native help and view

* **`Workspace` implements `help.KeyMap`.**
  * `ShortHelp` gives the focused pane's `KeyMapper` keys, or the top modal
    overlay's, then focus-next and zoom.
  * `FullHelp` gives four columns: the pane's keys, focus, layout (zoom and
    resize), and overlays (close).
  * Disabled bindings are left out, as `help.Model` expects.
  * [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) later
    generates these from its bindings, and keeps this interface.
* **`(*Workspace).View() tea.View`** returns `tea.NewView(w.Render())` with
  `Cursor` set from `w.Cursor()`. The program sets `AltScreen`, `MouseMode`
  and the rest on the returned value. The workspace still never sets them
  (0001-MADR §6, rule 2), and a conformance test checks that `View` leaves
  them zero.

### 6. Chrome, and Go 1.27 accessors

* **Separators follow per-pane chrome.** A separator is drawn when a pane
  on either side of it has `Separators` chrome, found from the plan's
  rectangles. A separator between two `None` or two `Borders` panes stays
  blank.
* **Junctions.** Where a vertical and a horizontal separator meet, the cell
  takes `SeparatorCross` or a tee. `glyph.Set` gains `SeparatorTeeDown`,
  `SeparatorTeeUp`, `SeparatorTeeRight` and `SeparatorTeeLeft` (`┬ ┴ ├ ┤`,
  ASCII `+`). The glyph tests walk the struct by reflection, so the new
  fields are covered.
* **Typed access, as a generic method:**

  ```go
  // PaneAs returns pane, or overlay, id as T, and false when it is absent
  // or is not a T.
  func (w *Workspace) PaneAs[T Pane](id layout.PaneID) (T, bool)
  ```

  It is a method, not an interface method, which Go 1.27 refuses.
* **Iterators:**

  ```go
  // package layout: placed panes in tree order.
  func (p Plan) All() iter.Seq2[PaneID, Rect]

  // package workspace: hosted panes in focus-ring order.
  func (w *Workspace) Panes() iter.Seq2[layout.PaneID, Pane]
  ```

  `render`, `resolve` and the focus ring use `Plan.All` themselves.
* **`splitLayerID` goes away** with the compositor's string IDs. Region
  kinds are a typed constant.

### 7. Versioning

* This ships as **`v0.2.0`**: it adds API, and it changes the default width
  method from grapheme to wcwidth, which a caller can see.
* Everything public in `v0.1.5` still compiles. The release notes name the
  width change and `WithWidthMethod(ansi.GraphemeWidth)` as the way back.
  *Amended 2026-10-04 (A1): `v0.1.6`, which has the same public API.*

### Consequences

* Good, because a frame drops from about 1.55 ms and 1.74 MB to the
  audit's measured 0.79 ms and 173 KB, and an unchanged frame costs almost
  nothing.
* Good, because the workspace measures as Bubble Tea writes, on every
  terminal, with or without mode 2027.
* Good, because a theme follows the profile and background replies, and a
  program restyles with one call.
* Good, because help, views, colours and typed access use the Charm and Go
  interfaces a reader already knows.
* Neutral, because ultraviolet was already compiled into every consumer
  through lipgloss. The new edge is a direct import, not new code.
* Bad, because ultraviolet has no tagged release, so an upstream commit can
  break `internal/cells` on a Charm upgrade. Containment keeps that to one
  package.
* Bad, because the default width method changes. A program that relied on
  grapheme widths without mode 2027 sees different truncation, and must set
  `WithWidthMethod`.
* Bad, because the hit test and the dirty flag are ours now. A missed dirty
  case shows a stale frame. The tests drive each case.
* Bad, because the workspace sends a terminal query by default that the
  program did not write. A terminal that shows queries as text shows it
  until 0005's gates apply. `WithoutBackgroundQuery()` turns it off.

### Confirmation

* **The benchmark gate.**
  * **Baseline.** Before any change, `go test -run '^$' -bench
    BenchmarkRender -benchmem -count=10 ./workspace` runs on the macOS
    development host. Its output is kept in the PLAN's execution record.
  * **Threshold.** After §2, benchstat against that baseline must show, for
    `BenchmarkRender/views`, time per frame at most 60% of the baseline and
    bytes per frame at most 20% of it. The audit's experiment reached 51%
    and 10%.
  * **In CI.** Timings are not gated in CI, where they are noisy. A
    deterministic test, `TestRenderAllocs`, uses `testing.AllocsPerRun`:
    * at most 650 allocations for a full 200 × 60 frame (the audit's 593
      plus 10%);
    * at most 2 for a frame where nothing is dirty.
  * A new `BenchmarkRender/truecolor` keeps the TrueColor case visible.
* **Output.** Every existing golden file is unchanged, byte for byte, at the
  default method. *(Amended 2026-10-04, A2: except the junction cell that
  §6 adds, in four files.)*
* **Width method.** Golden frames include an emoji fixture with a zero-width
  joiner and a VS16 selector in a title and in a body, at both methods. A
  test sends `ModeReportMsg` for mode 2027 with each value and checks the
  switch, and checks that `WithWidthMethod` pins it.
* **Theme.** `Init` asks for the background by default, and not with
  `WithoutBackgroundQuery()`. A `ColorProfileMsg` and a `BackgroundColorMsg`
  restyle the next frame, and a fixed theme ignores both. A view cached before a theme
  change is redrawn after it. `LightDarkColor` and `ProfileColor` resolve
  per background and profile, and the unknown-palette contrast test still
  holds.
* **Help and view.** `help.New().View(w)` renders the focused pane's keys
  first. `View()` carries the cursor and leaves the screen fields zero.
* **Chrome.** Goldens show separators under mixed per-pane chrome, and each
  junction glyph, in Unicode and ASCII.
* **Containment.** `depguard` reports ultraviolet planted in `workspace`, in
  a scratch copy.
* **Mutation proofs,** each seen failing on a scratch copy:
  * the frame is not cleared between renders;
  * the hit test takes the bottom region first;
  * the dirty flag is not set on a focus change;
  * the method is not switched on `ModeSet`;
  * one measurement keeps `ansi.StringWidth`;
  * the cache key leaves out `themeGen`;
  * the fixed theme follows `BackgroundColorMsg`;
  * `PaneAs` returns true for the wrong type.
* Pre-add, `-race`, the Windows test host and CI on all three systems pass.

## Pros and Cons of the Options

### A. Direct drawing on a reused buffer, contained

* Good, because it fixes the width method and reaches the measured frame.
* Good, because the drawing code is lipgloss's own, without its layer
  bookkeeping.
* Neutral, because `internal/cells` is a small wrapper to keep.
* Bad, because it imports an unversioned module directly.

### B. Stay on lipgloss's canvas

* Good, because it adds no direct requirement.
* Good, because canvas reuse alone takes the frame to 1.27 ms and 225 KB.
* Bad, because the width method cannot change, so audit item 6 stays open.
* Bad, because the compositor's three width passes remain, about 0.3 ms of
  the frame.

### C. Our own cell buffer

* Good, because it has no unversioned dependency.
* Bad, because it re-implements ultraviolet's ANSI and grapheme parsing,
  which lipgloss and Bubble Tea already compile in. Two parsers would
  drift.
* Bad, because it is the largest option, and the most likely to be wrong.

### D. API only

* Good, because it is the smallest change.
* Bad, because the frame cost and the width mismatch stay. The owner asked
  for the stack to be tightly integrated and optimized.

## Owner questions

*Answered 2026-10-02* (picked from options): Q1 "Accept, in
internal/cells"; Q2 "Query in Init"; Q3 "WcWidth, follow mode 2027"; Q4
"v0.2.0". Q1, Q3 and Q4 are the recommendation. Q2 is not: the workspace
requests the background in `Init` by default, and
`WithoutBackgroundQuery()` replaces the recommended `WithBackgroundQuery()`
opt-in. §4, Consequences and Confirmation were revised to match.

* **Q1. The ultraviolet import.** Recommended: accept it, at the version the
  graph selects, confined to `internal/cells` by `depguard`. The alternative
  is option B, which leaves the width defect open.
* **Q2. The background query.** Recommended: the workspace sends no query by
  default, and `WithBackgroundQuery()` opts in. The alternative is to
  request the background in `Init` by default. That is what lipgloss's
  documentation suggests, but it is a terminal query the program did not
  ask for.
* **Q3. The default width method.** Recommended: default to `WcWidth` and
  follow mode 2027, as Bubble Tea does. The alternative is to keep
  `GraphemeWidth` as the default. That keeps v0.1 behaviour, and stays
  misaligned on terminals without mode 2027.
* **Q4. The version.** Recommended: `v0.2.0`, because the default width
  method changes. The alternative is `v0.1.2`.

## Amendments

### A1 (2026-10-04): the pre-execution audit

*Status: accepted (2026-10-04).* The owner asked, before execution, for "the audit to
ensure state is known and factual". Every claim in this record was checked
against the tree at `5f514fb` and against the pinned upstream sources.

**Confirmed, unchanged:**

* Every upstream reference holds at the pinned versions: lipgloss v2.0.6
  `canvas.go:27` (`c.scr.Method = ansi.GraphemeWidth`), `canvas.go:87-89`
  and `layer.go:153-156`; bubbletea v2.0.10 `cursed_renderer.go:48`,
  `tea.go:784-791`, `802-805`, `880`, `980-994`, `1098` and `1118-1123`;
  ultraviolet `buffer.go:614-618` (`Method: ansi.WcWidth`).
* Every API this record uses exists at those versions: `lipgloss.LightDark`
  and `lipgloss.Complete`; bubbles `help.KeyMap`; `tea.ModeReportMsg{Mode,
  Value}`, `tea.RequestBackgroundColor` and `BackgroundColorMsg.IsDark`;
  `ansi.Method`'s `StringWidth` and `Truncate` methods; ultraviolet's
  `ScreenBuffer`, `NewStyledString` with `Draw`, and `TrimSpace`.
* ultraviolet is still indirect at
  `v0.0.0-20260811164956-006e29f97886`, and `go list -m -versions` still
  lists no tag. bubbles v2.2.1, as well as bubbletea v2.0.10, requires the
  older `v0.0.0-20260703014108-f5a850f9c2b7`; minimum version selection
  takes lipgloss's.
* In `workspace` today: the canvas is still built every frame
  (`render.go:41`); edge glyphs are still styled once per row
  (`render.go:192`); `renderSeparator` still reads only the workspace's
  chrome (`render.go:201`); `SeparatorCross` is never drawn; `KeyMapper`
  is never read; `Broadcast` still sorts on every message; no
  `lipgloss.Width` call is left; `Plan.All`, `View`, `Panes`, `PaneAs`,
  `ShortHelp`, `FullHelp`, `SetTheme` and `WidthMethod` do not exist.
* **The cost is where the audit found it.** `BenchmarkRender`, ten runs on
  the macOS development host (Apple M1 Pro), `views`: 1.588 ms ± 1%,
  1.662 MiB (1.74 MB) and 1,078 allocations per frame; `changer`:
  1.534 ms, 1.624 MiB and 925. The audit measured 1.55 ms, 1.74 MB and
  1,091, so the §Confirmation thresholds, relative to the PLAN's Step 1
  baseline, still mean what they meant.

**Changed since this record was written:**

* **Releases.** `v0.1.6`, not `v0.1.5`, completes the hardening
  ([0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  deviation D7). No Go file changed after `v0.1.4`.
* **Two of §2's items are already done,** by that PLAN's Steps 4 and 6:
  the view cache has a struct key, `viewKey{kind, id}` with an entry
  `cached{width, height, focused, view}`, and `fmt.Sprintf` is gone; and
  `renderBox` walks lines with `strings.SplitSeq`. `clip` still uses
  `strings.Split`.
* **What the hardening added that §2's dirty flag must cover:**
  * focus moves through one path, which a modal overlay's push or pop also
    takes, and panes get `PaneFocusMsg` and `PaneBlurMsg`;
  * `SendOverlay` delivers to an overlay, so "a message delivered to a pane
    that is not a `Changer`" includes an overlay's pane;
  * `Push` of an open ID replaces that overlay and drops its cached view, as
    `SetPane` and `Pop` do.
* **`Wrap`'s `Model`** implements `Cursor` and `Keys`, so §5's help lists a
  wrapped bubbles component's bindings with no further work.
* **The default focus keys** are `alt+.` and `alt+,`.
* **The gates** ([0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md)):
  every gate runs per module with `GOWORK=off`; `make release-check` runs
  before a tag; depguard's per-directory rules take the form `$all` plus
  `!**/<dir>/**`, and §1's ultraviolet rule takes it too; the conformance
  scan type-checks every package, and refuses more than §5 names.

**The cache key, put to the owner.** §2 widened the cache's key to
`{kind, id, width, height, focused, method, themeGen}`. With width and
height in the key, the map gains an entry at every size a pane is drawn at,
and nothing evicts the old ones. The hardening's shape keeps one entry per
pane or overlay: the key is `{kind, id}`, and the entry records what it was
drawn under.

* **Recommended (Q5):** keep the key `{kind, id}`, and add `method` and
  `themeGen` to the entry's check beside width, height and focus. A method
  or theme change then misses every entry, as §3 and §4 require, and the
  map stays one entry per pane.
* **The alternative** is §2 as written. It works, and the map grows with
  every resize until a pane is replaced.

**Owner question for A1.**

*Answered 2026-10-04* (picked from options): Q5 "The MADR's wider key".
That is not the recommendation. §2 stands as written: the key is
`{kind, id, width, height, focused, method, themeGen}`, replacing the
hardening's `viewKey{kind, id}`. Two consequences follow, and the PLAN's
Step 3 carries both:

* `SetPane`, `Pop` and a replacing `Push` drop every entry with the pane's
  or overlay's kind and ID, so the hardening's eviction tests keep holding;
* entries for sizes no longer drawn stay until their pane or overlay is
  dropped, which the owner accepted.

* **Q5. The cache key.** Recommended: `{kind, id}`, with method and theme
  in the entry's check. The alternative is §2's wider key.

### A2 (2026-10-04): junction glyphs change four goldens

*Status: accepted (2026-10-04).* It is
[0004-PLAN-integrate-charm-v2-and-go-1-27.md](0004-PLAN-integrate-charm-v2-and-go-1-27.md)
deviation D1.

§6 draws a junction glyph where a vertical and a horizontal separator
meet. §Confirmation also says every existing golden file stays unchanged.
The two meet in `frame-separators` at 160 columns: its one meeting cell,
drawn as `─` (ASCII `-`) up to `v0.1.6`, becomes `┴` (ASCII `+`). The
owner accepted that change (picked from options, 2026-10-04). §6 stands,
and §Confirmation's rule holds for every other cell of every existing
golden file.

## More Information

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md):
  §1 the audit and its profile, §2 the Go 1.27 features, §4 the Charm
  ecosystem.
* [0002-MADR-multi-pane-workspace-layouts.md](0002-MADR-multi-pane-workspace-layouts.md)
  §3 and its amendment A1, and
  [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md):
  the workspace as built, and the fixes this record builds on.
* [0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)
  §3 the stack, §6 the conventions.
* Later records build on this one:
  [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md)
  (live colour scheme, the probe) and
  [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) (help from
  bindings).
* Upstream: lipgloss v2.0.6 `canvas.go` and `layer.go`; bubbletea v2.0.10
  `tea.go` and `cursed_renderer.go`; ultraviolet `buffer.go` and
  `styled.go` at `006e29f97886`.
