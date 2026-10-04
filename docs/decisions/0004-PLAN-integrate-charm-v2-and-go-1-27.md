---
status: in-progress
date: 2026-10-04
associated-madr: "0004-MADR-integrate-charm-v2-and-go-1-27.md"
---
# Implement direct cell drawing, terminal-following width and theme, and Go 1.27 accessors (`v0.2.0`)

Associated MADR: [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)

**Revision, 2026-10-02.** The owner answered the MADR's Q1–Q4, and the
MADR is `accepted`. Q2 departs from the recommendation: `Init` asks for the
background by default, and `WithoutBackgroundQuery()` replaces
`WithBackgroundQuery()`. Step 1 and Step 5 changed to match. This PLAN is
still `proposed`, because execution is not yet approved.

**Revision, 2026-10-04: the pre-execution audit.** The owner asked for an
audit "to ensure state is known and factual" before execution. Its findings
are MADR amendment A1 (proposed), with owner question Q5 on the cache key.
This PLAN changed to match, and each change is annotated in place:

* `v0.1.6`, not `v0.1.5`, is the tree this work starts from and compares
  against (the hardening PLAN's deviation D7);
* Step 3's struct cache key and `strings.SplitSeq` in `renderBox` are
  already done; the step keeps the hardening's `{kind, id}` key and adds
  `method` and `themeGen` to the entry's check (A1, Q5);
* Step 3's dirty cases name what the hardening added: overlay messages
  through `SendOverlay`, and a replacing `Push`;
* Step 2's depguard rule takes the form 0010's rules use;
* Verification names the typed conformance scan and `make release-check`.

A measurement taken during the audit (A1) puts the baseline within 2.5% of
the audit's. Step 1 still records its own.

*Later on 2026-10-04:* the owner accepted A1, answered Q5 with "The MADR's
wider key" (not the recommendation), and approved this PLAN. Step 3's cache
bullet follows that answer. This PLAN is `in-progress`.

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
is `complete`, so this work starts from the `v0.1.5` tree, with its cache
and focus fixes in place. *Amended 2026-10-04: the `v0.1.6` tree, which
completes it (its deviation D7) and has the same Go code as `v0.1.4` and
`v0.1.5`.* (That PLAN's deviation D3: `v0.1.1` and
`v0.1.2` tag only its Steps 1–4. Its deviation D6: `v0.1.4` tags Steps
5–7a, and there is no `v0.1.3`.)

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
5. **Commit.** One commit per step, made by the owner. At the end of each
   step the agent stops with the pre-add checks passed and the step's
   evidence in the execution record, and stages and commits nothing. The
   owner commits with `git commit --no-edit` and pushes. (2026-10-02: the
   org rules forbid agent commits to `main`, and the owner chose this over
   a `feature/` branch.)

Added for this PLAN:

* **Goldens are a gate.** A step that changes any existing golden file at
  the default width method stops and is recorded as a deviation. New
  golden files are read in full before they are committed.

## Implementation Steps

### Step 1: records and the baseline

* The owner accepted the MADR on 2026-10-02, answering Q1–Q4, and the
  answers are recorded in it. When execution is approved, set this PLAN
  `in-progress` and update `docs/README.md`.
* **Baseline.** On the `v0.1.5` tree, on the macOS development host
  *(amended 2026-10-04: the `v0.1.6` tree)*:

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
  `internal/cells/`. *Amended 2026-10-04: a rule named `ultraviolet`, with
  `files` `$all` and `!**/internal/cells/**`, the form of 0010's `cobra`,
  `kong` and `glamour` rules; it covers every module.* AGENTS.md's Dependencies list names it, and this
  record.
* **Tests:**
  * a drawn string is clipped to its rectangle, and the cells outside it
    are untouched;
  * `Draw` clears its rectangle first, so a shorter string leaves no tail;
  * `Clear` empties the frame, and `Resize` keeps the buffer at the same
    size;
  * the same string drawn at `WcWidth` and at `GraphemeWidth` differs in
    width exactly for the emoji fixture;
  * `Render` of a frame drawn like the v0.1.5 *(amended: v0.1.6)* canvas equals
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
    `strings.SplitSeq`; *(amended 2026-10-04: `renderBox` already does,
    since the hardening's Step 6)*
  * the cache key is a struct, `{kind, id, width, height, focused, method,
    themeGen}`; *(2026-10-04, MADR A1, Q5 "The MADR's wider key": this key
    replaces the hardening's `viewKey{kind, id}`; `SetPane`, `Pop` and a
    replacing `Push` drop every entry with that kind and ID, so
    `TestSetPaneDropsTheCachedView`, `TestPopEvictsTheOverlay` and
    `TestPushReplacesAnOpenID` keep passing)*
  * a dirty flag, set by every case in MADR §2, lets `Render` return the
    last frame; *(amended 2026-10-04: the cases include a message
    `SendOverlay` delivers to an overlay's pane, and a `Push` that replaces
    an open overlay)*
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
    *(amended 2026-10-04: and a replacing `Push`, and a message to a
    non-`Changer` overlay)*
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
  theme for `WithTheme`, the background query in `Init` by default, and
  `WithoutBackgroundQuery`.
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
  * `Init` contains `tea.RequestBackgroundColor` by default, and does not
    with `WithoutBackgroundQuery`;
  * `WithPaletteFor` survives a background change.
* **Mutations:**
  * the cache key leaves out `themeGen`;
  * the fixed theme follows `BackgroundColorMsg`;
  * `LightDarkColor` swaps light and dark;
  * a `ProfileColor` is converted again after `Complete`;
  * `Init` leaves out the background query by default;
  * `WithoutBackgroundQuery` is ignored.

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
* Every golden file from `v0.1.5` is unchanged. *(Amended 2026-10-04:
  from `v0.1.6`, which has the same 96 files under
  `workspace/testdata/golden/`.)*
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`; *(amended
    2026-10-04: and `make release-check`, each per module)*
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`.
* `go mod tidy -diff` is clean. `go.mod` requires the five modules of
  0001-MADR §3 that are in use, plus ultraviolet, at the version the graph
  selects.
* `depguard` refuses ultraviolet outside `internal/cells`, and the Charm v1
  paths everywhere.
* `internal/conformance` finds no `os.Stdout`, `os.Stderr`, `AltScreen` or
  `signal.Notify` in non-test code. *(Amended 2026-10-04: the scan is typed,
  and also refuses `fmt.Print*`, `log`'s standard logger, `log/slog`'s
  default logger and the print builtins; `internal/cells` must pass it.)*
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

### Step 1: records and the baseline (2026-10-04)

* **Records.** Before approval, the owner asked for an audit "to ensure
  state is known and factual". It became MADR amendment A1 and this PLAN's
  revision of 2026-10-04. The owner then accepted A1, answered Q5 ("The
  MADR's wider key") and approved this PLAN. The PLAN is `in-progress`,
  and `docs/README.md` shows the MADR `accepted; A1 accepted` and this
  PLAN `in-progress`.
* **The tree.** `5f514fb`, whose Go files, `go.mod` and `go.sum` are
  identical to `v0.1.6`'s (`git diff --quiet v0.1.6 HEAD -- '*.go' go.mod
  go.sum` exits 0).
* **Baseline,** the command above, on the macOS development host:

  ```text
  goos: darwin
  goarch: arm64
  pkg: github.com/maccavelli/go-tui-lib/workspace
  cpu: Apple M1 Pro
  BenchmarkRender/views-10         	     705	   1653368 ns/op	 1741347 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     742	   1635968 ns/op	 1743034 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     722	   1641657 ns/op	 1739155 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     720	   1632069 ns/op	 1742717 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     735	   1653700 ns/op	 1743275 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     748	   1606328 ns/op	 1742876 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     745	   1648186 ns/op	 1743563 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     754	   1623907 ns/op	 1745691 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     697	   1698785 ns/op	 1744373 B/op	    1078 allocs/op
  BenchmarkRender/views-10         	     745	   1635191 ns/op	 1745190 B/op	    1078 allocs/op
  BenchmarkRender/changer-10       	     762	   1606571 ns/op	 1702143 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     775	   1623113 ns/op	 1704798 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     752	   1604682 ns/op	 1703843 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     748	   1574700 ns/op	 1703825 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     774	   1588347 ns/op	 1704319 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     768	   1563810 ns/op	 1705675 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     768	   1574254 ns/op	 1703463 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     782	   1553411 ns/op	 1703775 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     746	   1630598 ns/op	 1703943 B/op	     925 allocs/op
  BenchmarkRender/changer-10       	     720	   2141358 ns/op	 1699786 B/op	     924 allocs/op
  PASS
  ok  	github.com/maccavelli/go-tui-lib/workspace	24.828s
  ```

  benchstat: `views` 1.639 ms ± 1%, 1.662 MiB ± 0% (1.74 MB) and 1,078
  allocations; `changer` 1.597 ms ± 2%, 1.625 MiB and 925. The `views`
  median is 5.7% above the audit's 1.55 ms, inside the 25% band, so no
  explanation is needed. The last `changer` run, 2.14 ms, is an outlier
  that benchstat's median absorbs. Step 3's gate is therefore at most
  0.983 ms (60%) and 0.332 MiB (20%) for `views`.
* The benchmark wrote no `go.work.sum`.
