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
| 3 | `workspace/render.go`, `workspace/workspace.go`, `workspace/*_test.go`; also `workspace/overlay.go` (recorded in Step 3) | direct drawing, region hit test, struct key, dirty flag, sorted broadcast |
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

### Step 2: `internal/cells` (2026-10-04)

The owner committed Step 1 (`506a0fc`) and the 0010 records (`db18f09`),
and approved this step ("Proceed to 0004").

**What changed.**

* **`internal/cells/cells.go` (new).** `Frame` over a `uv.ScreenBuffer`:
  `NewFrame(w, h, m)`, `Resize` (a no-op at the same size; a negative size
  is 0), `Width`, `Height`, `SetMethod`, `Method`, `Clear`, `Draw(s, r)` and
  `Render`. `Draw` is `uv.NewStyledString(s).Draw` into `uv.Rect(r)`, which
  clears the rectangle and prints the string clipped to it; `Render` is
  `uv.TrimSpace` of the buffer's render, as lipgloss's `Canvas.Render` is.
  `Width` and `Height` are additions to MADR §1's list, which the tests and
  Step 3 need. The package comment says why the package exists.
* **`go.mod`.** `go mod tidy` moved
  `github.com/charmbracelet/ultraviolet v0.0.0-20260811164956-006e29f97886`
  from the indirect block to the direct one. The version is unchanged, and
  `go.sum` did not change.
* **`.golangci.yml`.** A depguard rule, `ultraviolet`, with `files` `$all`
  and `!**/internal/cells/**`, denies the module everywhere else, in every
  module, in the form of 0010's rules.
* **`AGENTS.md`,** Dependencies: names ultraviolet, this record's §1, and
  that only `internal/cells` may import it.

**Tests** (`internal/cells/cells_test.go`, new):

* a string drawn into a 4 × 1 rectangle of a frame filled with `x` is
  clipped to it, and every other cell keeps its `x`;
* drawing `ab` over `abcdef` in the same rectangle leaves no tail;
* `Resize` to the same size keeps the contents, `Clear` empties the frame,
  and a new or a negative size takes effect;
* the zero-width-joiner family emoji and the VS16 heart push a following
  `|` by exactly `m.StringWidth` of the fixture at each method, and the
  two methods differ for both emoji but not for ASCII;
* four layers like the workspace's (two bordered, coloured boxes with the
  emoji and Japanese text, a bold separator column, and a background-filled
  overlay across both) render byte for byte as lipgloss's
  `NewCanvas` and `NewCompositor` render them, at `GraphemeWidth`, the
  method lipgloss's canvas fixes; and no rendered row ends in a space.

The first run of the method test failed on its own fixture: `ab` contains
a `b`, so the gap was measured from the wrong one. It now measures from
the last `b`; only spaces lie between that and the `|`. The test file's
emoji constant had been written with literal joiner and selector
characters, which staticcheck's ST1018 reported; it now uses `\u`
escapes, with the same code points.

**Mutation proofs**, each on a scratch copy; none survived:

| Mutation | Killed by |
| :--- | :--- |
| `Draw` does not clear (it copies only the drawn string's non-empty cells) | `after a shorter draw: "xxabcdefxx", want "xxab    xx"` |
| `SetMethod` is ignored | `Method() is 0 after SetMethod(1)` |
| `Render` does not trim | `Render differs from lipgloss's canvas` |

**depguard proof,** on a scratch copy with `import uv
"github.com/charmbracelet/ultraviolet"` planted in `workspace/zz_uv.go`
and in `internal/cells/zz_uv.go`: golangci-lint on `./workspace/` exits 1
with `import 'github.com/charmbracelet/ultraviolet' is not allowed from
list 'ultraviolet': ultraviolet only in internal/cells (…§1) (depguard)`;
on `./internal/cells/` it exits 0, with no depguard report.

**Checks.**

* `make pre-add-check FILES=…` on the two new files: `2 file(s) clean in
  1 module(s)`, govulncheck included.
* `make lint` (with `make modernize`): `0 issues` for linux, darwin and
  windows.
* `go test -race -count=1 ./...` and `LC_ALL=C go test -count=1 ./...`:
  every package `ok`; the conformance scan type-checks `internal/cells`.
* `GOWORK=off go mod tidy -diff`: exit 0. No `go.work.sum` was written.
* **The Windows test host:** `make pre-add-check` and `make release-check` (`31 file(s) clean in 1 module(s)`, tests and govulncheck included), `make lint` (`0 issues` for linux, darwin and windows) and `make vuln` (`No vulnerabilities found.`) exited 0; `go-modules.sh --check` exited 0.

### Step 3: direct drawing (2026-10-04)

The owner committed Step 2 (`e5dc9d0`) and approved this step ("Proceed").

**What changed.**

* **`workspace/render.go`.**
  * `Render` draws into one reused `cells.Frame`: it resizes it only when
    the size changes, clears it, and draws each pane, then each separator,
    then each overlay into its rectangle with `Frame.Draw`.
    `lipgloss.Layer`, `lipgloss.Compositor` and `lipgloss.NewCanvas` are
    gone from `workspace`.
  * Each rectangle drawn is recorded as a region `{kind, id, rect,
    inner}`; the list is reversed so the top comes first, and `mouseEvent`
    takes the first region containing the pointer (`hit`). Region kinds are
    a typed constant, and `layerID`, `splitLayerID`, `w.hits` and `w.inner`
    are gone.
  * When nothing is dirty and no shown `Changer` reports a change,
    `Render` returns the last frame without drawing.
  * `view` keys the cache on `viewKey{kind, id, width, height, focused,
    method, themeGen}` (MADR A1, Q5). `method` is `ansi.GraphemeWidth`
    and `themeGen` 0 until Steps 4 and 5 make them move.
  * `renderBox` styles the left and right edges once per box, and writes
    each row's pieces into one pre-sized builder. `clip` walks lines with
    `strings.SplitSeq` into one pre-sized builder and pads from a constant
    run of spaces, with no allocation per line. Both were needed for the
    allocation gate (below).
* **`workspace/workspace.go`.** The new fields (`frame`, `method`,
  `themeGen`, `regions`, `last`, `dirty`, `ids`); `forget(kind, id)`,
  which drops every cached view of a pane or an overlay at every size;
  `touched(p)`, which marks the frame dirty after a message reaches a pane
  that is not a `Changer`; the dirty flag set by `New`, `resolve` (size,
  layout, state, zoom, hide, resize, `SetPane`) and `Focus`; `Broadcast`
  walking `ids`, kept sorted by `New` and `SetPane` (`SetLayout` adds no
  pane, so it keeps none). `Pane`'s comment says that a pane's view changes
  through its `Update`, or, for a `Changer`, when `Changed` says so.
* **`workspace/overlay.go`.** `Push` marks the frame dirty;
  `removeOverlay`, which `Pop` and a replacing `Push` use, calls `forget`
  and marks it dirty; `updateOverlay` calls `touched`. This file was not
  in Step 3's paths, but the step's own text requires it: `Pop` and a
  replacing `Push` drop every cached entry (Q5), and an overlay message
  and push are dirty cases. The Scope table is annotated.

**Two behaviours, recorded for the release notes.**

* A pane that is not a `Changer` is redrawn after a message reaches it,
  not on every `Render`. A program that writes to a pointer pane directly,
  outside its `Update`, sees the change at the next message or layout
  change. MADR §2 decided this; `Pane`'s comment now says it.
  `TestPanesAreClipped` changed a pane's body directly and then rendered,
  so it now delivers a message after the change, and asserts that the new
  body was drawn: without that assertion it passed against the old frame.
* A `Changer` reporting a change is redrawn even with no message, because
  `Render` asks each shown `Changer`, as `TestChangerSkipsTheView`
  requires. MADR §2 named only messages; asking costs no allocation.

**Tests.**

* Every existing workspace and agent golden file passes unchanged: no file
  under `workspace/testdata/` changed. Every existing mouse test passes.
* `workspace/render_test.go` (new):
  * a clean frame equals the last one and asks no pane for its view;
  * on a workspace of unchanged `Changer`s, each case marks the frame
    dirty: size, layout, state, focus, overlay push, overlay pop, a
    replacing push, a message to a non-`Changer` pane, a message to a
    non-`Changer` overlay; a message to an unchanged `Changer` marks
    nothing and asks for no view. Theme and method are Steps 5 and 4;
  * hiding every pane leaves an empty frame;
  * a click inside an overlay over `main` reaches the overlay at (0, 0)
    and not `main`; a click on the overlay's border reaches neither, and
    focus stays;
  * `TestRenderAllocs`: a full 200 × 60 frame takes 440 allocations (at
    most 650), and an unchanged one 0 (at most 2).
* `BenchmarkRender` marks the frame dirty each iteration, so it measures a
  full frame as `v0.1.6` drew every frame; without that it would measure
  the returned last frame and compare nothing with the baseline. It gains
  `truecolor`, a TrueColor theme with Unicode glyphs.

The first allocation run measured 702 for a full frame. A profile put 93
allocations per frame in `renderBox`'s per-row concatenation and 93 in
`clip`'s per-line padding and join; writing both into one builder took it
to 440.

**Benchmark,** the Step 1 command, on the macOS development host:

  ```text
  goos: darwin
  goarch: arm64
  pkg: github.com/maccavelli/go-tui-lib/workspace
  cpu: Apple M1 Pro
  BenchmarkRender/views-10         	    1492	    863537 ns/op	  151131 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1465	    860938 ns/op	  150884 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1401	    867234 ns/op	  150520 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1515	    850384 ns/op	  151365 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1404	    866753 ns/op	  151563 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1495	    850089 ns/op	  150924 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1417	    881236 ns/op	  151211 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1440	    847756 ns/op	  150585 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1366	    861917 ns/op	  151726 B/op	     440 allocs/op
  BenchmarkRender/views-10         	    1458	    815736 ns/op	  150912 B/op	     440 allocs/op
  BenchmarkRender/changer-10       	    1521	    794889 ns/op	  136810 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	    1508	    797043 ns/op	  137039 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	    1485	   1244691 ns/op	  136787 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	    1195	   2278979 ns/op	  137254 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	    1225	   1237002 ns/op	  137063 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	    1191	   1329663 ns/op	  137205 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	    1381	   1860360 ns/op	  137206 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	     337	   3317503 ns/op	  141447 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	     810	   1287729 ns/op	  137661 B/op	     437 allocs/op
  BenchmarkRender/changer-10       	    1148	   1599780 ns/op	  137533 B/op	     437 allocs/op
  BenchmarkRender/truecolor-10     	     883	   1222004 ns/op	  224990 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	     820	   1356163 ns/op	  224819 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	    1249	    981110 ns/op	  224248 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	    1143	    991677 ns/op	  225125 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	    1215	    980015 ns/op	  223952 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	    1225	    984352 ns/op	  224522 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	    1221	    994156 ns/op	  224668 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	    1221	    977211 ns/op	  224723 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	    1239	    980048 ns/op	  225110 B/op	    1741 allocs/op
  BenchmarkRender/truecolor-10     	    1242	   1054055 ns/op	  224689 B/op	    1741 allocs/op
  PASS
  ok  	github.com/maccavelli/go-tui-lib/workspace	41.272s
  ```

  benchstat against Step 1's baseline:

  ```text
  goos: darwin
  goarch: arm64
  pkg: github.com/maccavelli/go-tui-lib/workspace
  cpu: Apple M1 Pro
    │ bench-base.txt │  bench-step3.txt  │
    │  sec/op  │  sec/op  vs base  │
  Render/views-10  1638.8µ ± 1%  861.4µ ±  2%  -47.44% (p=0.000 n=10)
  Render/changer-10  1.597m ± 2%  1.309m ± 74%  ~ (p=0.247 n=10)
  Render/truecolor-10  988.0µ ± 24%
  geomean  1.618m  1.037m  -34.36%

    │ bench-base.txt │  bench-step3.txt  │
    │  B/op  │  B/op  vs base  │
  Render/views-10  1702.3Ki ± 0%  147.5Ki ± 0%  -91.34% (p=0.000 n=10)
  Render/changer-10  1663.9Ki ± 0%  134.0Ki ± 0%  -91.95% (p=0.000 n=10)
  Render/truecolor-10  219.4Ki ± 0%
  geomean  1.644Mi  163.1Ki  -91.65%

    │ bench-base.txt │  bench-step3.txt  │
    │  allocs/op  │  allocs/op  vs base  │
  Render/views-10  1078.0 ± 0%  440.0 ± 0%  -59.18% (p=0.000 n=10)
  Render/changer-10  925.0 ± 0%  437.0 ± 0%  -52.76% (p=0.000 n=10)
  Render/truecolor-10  1.741k ± 0%
  geomean  998.6  694.3  -56.09%
  ```

* **The gate holds:** `views` 861.4 µs, 52.6% of 1.639 ms (at most 60%),
  and 147.5 KiB, 8.7% of 1.662 MiB (at most 20%); 440 allocations from
  1,078.
* `changer` is noisy (± 74%, runs from 0.79 ms to 3.32 ms) while its bytes
  and allocations are steady; the host was busy during part of the run.
  It is not gated. `truecolor`, new, is 0.988 ms, 219.4 KiB and 1,741
  allocations: TrueColor styles produce more SGR sequences per cell.

**Mutation proofs**, each on a scratch copy; none survived:

| Mutation | Killed by |
| :--- | :--- |
| the frame is not cleared between renders | `with every pane hidden the frame still shows "+- > main ---…"` |
| the hit test takes the bottom region first | `a click inside the overlay: overlay got [], main got [left]` |
| the dirty flag is not set on a focus change | `TestEachDirtyCaseRedraws/focus: focus did not mark the frame dirty` |
| the dirty flag is not set when a non-`Changer` pane gets a message | the pane and overlay message cases; `TestPanesAreClipped: the new body was not drawn` |
| edge glyphs are styled from the wrong style when focused | `TestFramesGolden: frame-focus-main color.utf8.80: line 2 differs`, and the other colour variants |
| `forget` drops no cached view | `TestSetPaneDropsTheCachedView`; `TestPopEvictsTheOverlay: after Pop the cache still holds {1 qqpopped 28 4 true 1 0}` |
| `Render` does not ask `Changer`s | `TestChangerSkipsTheView: a changed Changer was viewed 1 times` |
| `Render` leaves the frame dirty | `TestCleanFrameSkipsEveryView: an unchanged frame asked for 3 views`; `TestRenderAllocs: an unchanged frame: 440 allocations` |

The last three are added to the PLAN's five, for the eviction, the
`Changer` poll and the flag's reset.

**Checks.**

* `make pre-add-check FILES=…` on the five Go files: the first run failed
  on gofmt, a blank line left at the end of `workspace.go` where
  `layerID` and `splitLayerID` were removed; after `gofmt -w`,
  `5 file(s) clean in 1 module(s)`, govulncheck included.
* `make lint` (with `make modernize`): `0 issues` for linux, darwin and
  windows.
* `go test -race -count=1 ./...`, `LC_ALL=C go test -count=1 ./...` and
  `go test -shuffle=on -count=2 ./...`: every package `ok`.
* `GOWORK=off go mod tidy -diff`: exit 0. No `go.work.sum` was written.
* **The Windows test host:** a first run, started before the gofmt fix, failed `make pre-add-check`, `make release-check` and `make lint` on the same `workspace\\workspace.go:676:1: File is not properly formatted (gofmt)`. After the fix: `make pre-add-check` and `make release-check` (`32 file(s) clean in 1 module(s)`), `make lint` (`0 issues` for linux, darwin and windows) and `make vuln` (`No vulnerabilities found.`) exited 0; `go-modules.sh --check` exited 0.

### Step 4: the width method (2026-10-04)

The owner committed Step 3 (`1553112`) and approved this step ("commit
then proceed").

**What changed.**

* **`workspace/workspace.go`.**
  * `method` defaults to `ansi.WcWidth`, what Bubble Tea's renderer starts
    with; `pinned` records `WithWidthMethod`.
  * `Update` passes a `tea.ModeReportMsg` to `followMode`, then broadcasts
    it as before, so the panes still get it. `followMode` switches to
    `ansi.GraphemeWidth` when the report is for `ansi.ModeUnicodeCore` with
    `ModeReset`, `ModeSet` or `ModePermanentlySet`, bubbletea v2.0.10's rule
    (`tea.go:802-805`), unless the method is pinned.
  * `WithWidthMethod(m)` sets and pins the method; `WidthMethod()` reports
    it. `setMethod` marks the frame dirty on a change; every cached view
    then misses, because its key holds the method.
* **`workspace/render.go`.** `clip` takes the method, and `title`,
  `renderBox` and `renderSeparator` measure and truncate with `w.method`.
  `Render` calls `Frame.SetMethod(w.method)` before drawing. No
  `ansi.StringWidth`, `ansi.Truncate` or `lipgloss.Width` call is left in
  `workspace`'s non-test files.
* **`workspace/model.go`.** `Model.View` returns the hosted model's view
  as it is. It used to clip it with `clip`, which now needs the
  workspace's method, which a `Model` cannot know; the workspace clips
  every view to its pane anyway, so frames do not change. Its doc comment
  says so. This is a reading of the step, recorded here: the source scan
  leaves no fixed-method clip for `Model` to call.

**Goldens.** Every existing golden file passes unchanged at the new default
method: none under `workspace/testdata/` changed, since the existing
fixtures hold no character whose wcwidth and grapheme widths differ.
Sixteen new files, `width-wcwidth` and `width-grapheme` across
{colour, no colour} × {UTF-8, ASCII} × {80, 120}, were written with
`-tuitest.update` and read before commit:

* the no-colour files were read in full. At 80 columns the sidebar folds
  under the main pane, so its long title fits; at 120 it is truncated,
  "longer th…" at wcwidth and "longer t…" at grapheme widths, because the
  heart is one cell in the first and two in the second;
* each colour file's text, with its SGR sequences stripped, equals its
  no-colour twin's, and each carries colour (88 SGR sequences at 80
  columns, 139 at 120);
* `tuitest` writes each line's width measured in graphemes, so the
  wcwidth files show 76, 81 or 83 on lines that are exactly 80 cells in
  wcwidth, which `TestEveryLineIsTheFrameWidth` asserts.

**Tests** (`workspace/width_test.go`, new):

* the golden frames above, with the zero-width-joiner family emoji and the
  VS16 heart in titles and bodies;
* at each method, at 80 and 120 columns, in both glyph sets, every line of
  the frame is exactly the frame width measured with that method;
* `ModeReportMsg` for mode 2027 with each of the five `ModeSetting` values
  switches only on `ModeSet`, `ModeReset` and `ModePermanentlySet`, and
  every report reaches the panes; a mode 2026 report switches nothing;
* `WithWidthMethod` pins the method against a later report. A report only
  ever switches to `GraphemeWidth`, so the PLAN's case, pinning
  `GraphemeWidth`, cannot show the pin; the test pins `WcWidth` as well,
  which can;
* on unchanged `Changer`s, a method switch marks the frame dirty, and the
  next frame's lines are exact at the new method, so no cached view or
  frame kept the old one;
* a source scan of the package's non-test files finds no call to
  `ansi.StringWidth`, `ansi.Truncate` or `lipgloss.Width`.

The test file's emoji constant was again written with literal joiner and
selector characters, and was corrected to `\u` escapes before any run that
is recorded here.

**Mutation proofs**, each on a scratch copy; none survived:

| Mutation | Killed by |
| :--- | :--- |
| the switch also fires on `ModeNotRecognized` | `mode 2027 reported 0: method 1, want 0` |
| the title truncation keeps `ansi.Truncate` | `tuitest: width-wcwidth nocolor.utf8.120: line 1 differs`, and the other three 120-column wcwidth files |
| a method change does not invalidate the cache | `method 1, width 80: line 1 is 76 cells` |
| `Frame.SetMethod` is not called | `method 1, width 80: line 0 is 76 cells` |

**Checks.**

* `make pre-add-check FILES=…` on the four Go files: `4 file(s) clean in
  1 module(s)`, govulncheck included.
* `make lint` (with `make modernize`): `0 issues` for linux, darwin and
  windows.
* `go test -race -count=1 ./...`, `LC_ALL=C go test -count=1 ./...` and
  `go test -shuffle=on -count=2 ./...`: every package `ok`.
* `GOWORK=off go mod tidy -diff`: exit 0.
* **`go.work.sum`.** It appeared once during this step, with the same six
  lines as in 0010-MADR A1. It came from `go doc` calls on dependency
  packages in workspace mode, not from the build: after deleting it,
  `go build ./...`, `go vet ./workspace/`, `go test ./workspace/` and
  `go test ./...` each wrote nothing. It was deleted again; `go doc` runs
  with `GOWORK=off` from here, and 0010-PLAN Phase 7 ignores the file.
* **The Windows test host:** `make pre-add-check` and `make release-check` (`33 file(s) clean in 1 module(s)`, the width goldens and tests included), `make lint` (`0 issues` for linux, darwin and windows) and `make vuln` (`No vulnerabilities found.`) exited 0; `go-modules.sh --check` exited 0.
