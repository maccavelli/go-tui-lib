---
status: proposed
date: 2026-10-02
associated-madr: "0004-MADR-integrate-charm-v2-and-go-1-27.md"
---
# Implement direct cell drawing, terminal-following width and theme, and Go 1.27 accessors (`v0.2.0`)

Associated MADR: [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)

## Goal

Ship the MADR's §§1–7 as `v0.2.0`. That means:

* `workspace` draws on a reused ultraviolet buffer behind `internal/cells`;
* it measures with the width method Bubble Tea writes with;
* its theme follows profile and background replies;
* it implements `help.KeyMap` and offers `View`, `PaneAs` and `Panes`;
* `layout.Plan` offers `All`.

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag `v0.2.0`.

**Precondition.** [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
is `complete`, so this work starts from the `v0.1.1` tree, with its cache
and focus fixes in place.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0004-*`, `docs/README.md` | accept the records; record the benchmark baseline |
| 2 | `internal/cells/`; `go.mod`; `.golangci.yml` (`depguard`); `AGENTS.md` (Dependencies) | the contained ultraviolet wrapper |
| 3 | `workspace/render.go`, `workspace/workspace.go`, `workspace/*_test.go` | direct drawing, region hit test, struct key, dirty flag, sorted broadcast |
| 4 | `workspace/`, `workspace/testdata/golden/` | the width method |
| 5 | `theme/`, `workspace/` | adaptive colours, `SetTheme`, the theme builder |
| 6 | `workspace/`, `glyph/` | `help.KeyMap`, `View`, per-pane separators, junction glyphs |
| 7 | `layout/`, `workspace/` | `Plan.All`, `Panes`, `PaneAs` |
| 8 | `README.md`, `docs/`, `docs/guides/building-workspaces.md` | documentation, release notes, close-out |

`go.mod` changes once, in Step 2: ultraviolet moves from the indirect block
to the direct block at `v0.0.0-20260811164956-006e29f97886`, the version the
graph already selects. No other requirement changes.

### Out of scope

* The terminal probe, live colour-scheme switching and terminal services:
  [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md).
* Generating help from a keymap engine:
  [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md).
* The audit's defects, which
  [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  fixes, apart from item 6, which is Step 4 here.
* Style-aware snapshots and `T.ArtifactDir` artifacts in `tuitest`, which
  are a later testing record.
* A Charm upgrade. The versions stay at lipgloss v2.0.6, bubbletea v2.0.10
  and bubbles v2.2.1.
* `git push` and tags, which are the owner's.

## Rules for every step

1. **Order.** Each step's package compiles, passes its tests and passes the
   pre-add gate before the next step starts.
2. **Mutation proofs.** Each step names mutations of its key invariants.
   Each is applied to a scratch copy and must make a named test fail. A
   mutation that survives, or does not compile, is replaced and recorded.
3. **Checks per step:**
   * `make pre-add-check FILES=…`;
   * `make lint`;
   * `go test -race -count=1 ./...`;
   * `LC_ALL=C go test ./...`;
   * the Windows test host on a scratch copy;
   * `go mod tidy -diff`.
4. **Conventions.** Every package follows 0001-MADR §6. In particular,
   nothing writes to `os.Stdout` or `os.Stderr`, nothing sets the alternate
   screen or installs a signal handler, and every glyph comes from `glyph`.
5. **Commit.** One commit per step, with `git commit --no-edit`, after the
   owner authorizes commits to `main` in that turn. The execution record
   gets each step's evidence before its commit.

Added for this PLAN:

* **Goldens are a gate.** A step that changes any existing golden file at
  the default width method stops and is recorded as a deviation. New
  golden files are read in full before they are committed.

## Implementation Steps

### Step 1: records and the baseline

* The owner accepts the MADR, answering Q1–Q4. Record the answers, set the
  MADR `accepted` and this PLAN `in-progress`, and update `docs/README.md`.
* **Baseline.** On the `v0.1.1` tree, on the macOS development host:

  ```bash
  go test -run '^$' -bench BenchmarkRender -benchmem -count=10 ./workspace \
    > <scratch>/bench-base.txt
  ```

  Keep the whole output in the execution record. If the `views` median
  differs from the audit's 1.55 ms by more than 25%, record why before
  continuing: the threshold is relative to this baseline, not the audit's.

### Step 2: `internal/cells`

* `Frame` as MADR §1: `NewFrame`, `Resize` (a no-op at the same size),
  `SetMethod`, `Method`, `Clear`, `Draw` and `Render`, over
  `uv.ScreenBuffer`, `uv.NewStyledString(s).Draw` and `uv.TrimSpace`.
* `go.mod`: ultraviolet becomes a direct requirement, at the same version.
* `.golangci.yml`: a `depguard` rule denies
  `github.com/charmbracelet/ultraviolet` in every file outside
  `internal/cells/`. AGENTS.md's Dependencies list names it, and this
  record.
* **Tests:**
  * a drawn string is clipped to its rectangle, and the cells outside it
    are untouched;
  * `Draw` clears its rectangle first, so a shorter string leaves no tail;
  * `Clear` empties the frame, and `Resize` keeps the buffer at the same
    size;
  * the same string drawn at `WcWidth` and at `GraphemeWidth` differs in
    width exactly for the emoji fixture;
  * `Render` of a frame drawn like the v0.1.1 canvas equals
    `lipgloss.Canvas.Render` for the same layers, byte for byte.
* **Mutations:**
  * `Draw` does not clear;
  * `SetMethod` is ignored;
  * `Render` does not trim.
* **depguard proof.** On a scratch copy, an ultraviolet import planted in
  `workspace` is reported, and the same import in `internal/cells` is not.

### Step 3: direct drawing

* `Render` uses the reused `Frame` (MADR §2):
  * panes, then separators, then overlays, each drawn into its rectangle;
  * `Layer` and `Compositor` are removed;
  * regions `{kind, id, rect, inner}` are recorded top first, and
    `mouseEvent` hit-tests them;
  * edge glyphs are styled once per box, and lines are walked with
    `strings.SplitSeq`;
  * the cache key is a struct, `{kind, id, width, height, focused, method,
    themeGen}`;
  * a dirty flag, set by every case in MADR §2, lets `Render` return the
    last frame;
  * `Broadcast` uses a sorted ID slice kept by `New`, `SetPane` and
    `SetLayout`.
* **Tests:**
  * every existing workspace and agent golden file is unchanged;
  * every existing mouse test passes: click focus, wheel to the pane under
    the pointer, separator drag, a modal overlay trapping the mouse;
  * an overlay over a pane wins the hit test, and a click on a border hits
    nothing inside;
  * a frame with nothing dirty is returned without any pane's `View` being
    called, and each dirty case (size, layout, state, focus, overlay push
    and pop, theme, method, a message to a non-`Changer` pane) redraws;
  * `TestRenderAllocs`: at most 650 allocations for a full frame, and at
    most 2 for a clean one.
* **Benchmark.** Re-run the Step 1 command, and compare with benchstat:
  `views` time at most 60% and bytes at most 20% of the baseline. Add
  `BenchmarkRender/truecolor`. Keep both outputs and the benchstat table in
  the execution record.
* **Mutations:**
  * the frame is not cleared between renders;
  * the hit test takes the bottom region first;
  * the dirty flag is not set on a focus change;
  * the dirty flag is not set when a non-`Changer` pane gets a message;
  * edge glyphs are styled from the wrong style when focused.

### Step 4: the width method

* `Workspace` keeps `method ansi.Method`, default `ansi.WcWidth` (MADR §3).
* `Update` switches it to `GraphemeWidth` on `tea.ModeReportMsg` for
  `ansi.ModeUnicodeCore` with `ModeReset`, `ModeSet` or
  `ModePermanentlySet`, unless `WithWidthMethod` pinned it. The message
  still reaches the panes.
* `WithWidthMethod(m)` and `WidthMethod()`.
* Every `ansi.StringWidth`, `ansi.Truncate` and `lipgloss.Width` in
  `workspace` becomes a call on `w.method`, and `Frame.SetMethod` follows.
  A method change invalidates the cache and marks the frame dirty.
* **Tests:**
  * a fixture with a zero-width-joiner family emoji and a VS16 heart, in a
    title and in a pane body, renders golden frames at both methods across
    the matrix, at 80 and 120 columns;
  * at each method, every line of the frame is exactly the frame width
    measured with that method;
  * `ModeReportMsg` for 2027 with each of the five `ModeSetting` values
    switches only on the three that Bubble Tea switches on;
  * `WithWidthMethod(ansi.GraphemeWidth)` ignores a later report;
  * a source scan finds no `ansi.StringWidth(`, `ansi.Truncate(` or
    `lipgloss.Width(` left in `workspace`'s non-test files.
* **Mutations:**
  * the switch also fires on `ModeNotRecognized`;
  * the title truncation keeps `ansi.Truncate`;
  * a method change does not invalidate the cache;
  * `Frame.SetMethod` is not called.

### Step 5: the theme

* `theme` (MADR §4): `FromDark`, `LightDarkColor`, `ProfileColor`,
  `WithPaletteFor`, and `build` resolving both colour types through
  `lipgloss.LightDark` and `lipgloss.Complete`.
* `workspace`: `SetTheme`, `ThemeBuilder`, `WithThemeBuilder`, the follow
  rules for `tea.ColorProfileMsg` and `tea.BackgroundColorMsg`, a fixed
  theme for `WithTheme`, and `WithBackgroundQuery`.
* **Tests:**
  * a `LightDarkColor` resolves to its light, dark and unknown colours on
    each background, and to the dark colour when the unknown one is nil;
  * a `ProfileColor` resolves to its ANSI, ANSI256 and TrueColor colours,
    and is not converted again;
  * the unknown-palette contrast test and the NO_COLOR and ASCII tests
    still hold;
  * a following workspace restyles the next frame on each message, and a
    view cached before it is redrawn;
  * a fixed theme ignores both messages, and both messages still reach a
    recording pane;
  * `Init` contains `tea.RequestBackgroundColor` only with
    `WithBackgroundQuery`;
  * `WithPaletteFor` survives a background change.
* **Mutations:**
  * the cache key leaves out `themeGen`;
  * the fixed theme follows `BackgroundColorMsg`;
  * `LightDarkColor` swaps light and dark;
  * a `ProfileColor` is converted again after `Complete`.

### Step 6: help, view and chrome

* `ShortHelp` and `FullHelp` as MADR §5. `View()` as MADR §5.
* Separators drawn when a neighbouring pane has `Separators` chrome, and
  junction glyphs at their meeting cells. `glyph.Set` gains the four tee
  fields in both sets.
* **Tests:**
  * `help.New().View(w)` lists the focused pane's keys first, then the
    workspace's, and leaves out disabled bindings; with a modal overlay
    open it lists the overlay's;
  * `View()` has the frame and the cursor, and `AltScreen`, `MouseMode`,
    `ReportFocus` and `KeyboardEnhancements` at their zero values;
  * goldens: a sidebar layout with `Separators` on the main pane and `None`
    on the footer; a three-way layout showing a cross and each tee; both
    glyph sets;
  * the glyph reflection tests cover the new fields: one cell, and ASCII
    in the ASCII set.
* **Mutations:**
  * `ShortHelp` puts the workspace's keys first;
  * a separator is drawn only by the global chrome;
  * the cross is drawn as a vertical line;
  * `View` sets `AltScreen`.

### Step 7: Go 1.27 accessors

* `layout.Plan.All`, `Workspace.Panes` and `Workspace.PaneAs[T]` (MADR §6).
  `render`, `resolve` and the focus ring use `Plan.All`.
* **Tests:**
  * `All` yields placed panes in `Order`, and stops when the loop breaks;
  * `Panes` yields in focus-ring order;
  * `PaneAs` returns a pane and an overlay as their concrete types, and
    false for an absent ID and for the wrong type;
  * an example, `ExampleWorkspace_PaneAs`, compiles and runs.
* **Mutations:**
  * `All` ignores the `yield` result;
  * `PaneAs` returns true for the wrong type.

### Step 8: documentation and close-out

* **`docs/guides/building-workspaces.md`:** following the theme, the width
  method and `WithWidthMethod`, `View()`, help, `PaneAs` and `Panes`.
* **Doc comments** in each changed package. `internal/cells` says why it
  exists.
* **Docs tree:** `docs/architecture.md` (the new package, the ultraviolet
  edge, the rendering path), `docs/README.md` rows, README Status.
* **Release notes for `v0.2.0`** in the execution record, naming the width
  method change and the way back.
* **Verification** as below. Mark `complete` after CI is green on the pushed
  tree. The owner tags.

## Verification

* Every step's mutations are killed.
* The benchmark gate holds: `views` time at most 60% and bytes at most 20%
  of the Step 1 baseline, by benchstat; `TestRenderAllocs` passes.
* Every golden file from `v0.1.1` is unchanged.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`.
* `go mod tidy -diff` is clean. `go.mod` requires the five modules of
  0001-MADR §3 that are in use, plus ultraviolet, at the version the graph
  selects.
* `depguard` refuses ultraviolet outside `internal/cells`, and the Charm v1
  paths everywhere.
* `internal/conformance` finds no `os.Stdout`, `os.Stderr`, `AltScreen` or
  `signal.Notify` in non-test code.
* The identifier scan of 0001-PLAN V7 finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–8 and tags `v0.2.0`. pi-go moves to
  it under its own records.
* **Rollback.** Before the push, each step is one local commit. After it, a
  `v0.2.1` fixes forward. A program hit by the width change sets
  `WithWidthMethod(ansi.GraphemeWidth)`. If ultraviolet breaks on a Charm
  upgrade, the fix is in `internal/cells` only, and the upgrade waits for
  it.

## Execution Record

None yet.
