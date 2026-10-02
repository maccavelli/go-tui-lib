---
status: in-progress
date: 2026-10-02
associated-madr: "0002-MADR-multi-pane-workspace-layouts.md"
---
# Harden multi-pane workspaces (`v0.1.1`)

Associated MADR: [0002-MADR-multi-pane-workspace-layouts.md](0002-MADR-multi-pane-workspace-layouts.md),
amendments A1 and A2. The first plan for that MADR,
[0002-PLAN-multi-pane-workspace-layouts.md](0002-PLAN-multi-pane-workspace-layouts.md),
shipped `v0.1.0` and is complete. This plan fixes what an audit then found in
it ([0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§1).

## Goal

Fix every defect amendment A1 names, so that `v0.1.1`:

* cannot crash, show a stale or a foreign view, or mis-size an overlay;
* focuses value-type panes and bubbles models;
* resizes without a dead zone;
* has default keys that legacy terminals decode reliably;
* lets consumers keep their own `-update` flag;
* has a conformance gate that reads uses rather than names.

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag `v0.1.1`. *Amended by deviation D3: the
release that completes this PLAN is `v0.1.3`; `v0.1.1` and `v0.1.2`
both tag Steps 1–4.*

## Scope

### In scope

| Step | Paths | Finding (0003-REPORT §1) |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0002-*`, `docs/README.md` | accept amendment A1 |
| 2 | `tuitest/` | 4: the `-update` panic; 12 and 13 in this package |
| 3 | `layout/` | 8: applied resize deltas, `Separator.Resizable`; 12 and 13 in this package |
| 3a | `layout/` | MADR A2: a split name is used once per solve (found in Step 3; deviation D2) |
| 4 | `workspace/overlay.go`, `workspace/render.go`, `workspace/workspace.go` | 1, 2, 3: the crash, the cache, overlay sizes |
| 5 | `workspace/focus.go` (new), `workspace/model.go` (new), `workspace/workspace.go`, `workspace/overlay.go` | 7: focus as messages, `Wrap` and `Model` |
| 6 | `workspace/keys.go`, `workspace/workspace.go`, `workspace/*_test.go` | 8 in the workspace, 9, 12 and 13 |
| 7 | `internal/conformance/`, `Makefile`, `.github/workflows/ci.yml` | 11, and a `go fix` gate |
| 8 | `README.md`, `AGENTS.md`, `docs/` | documentation, release notes, close-out |

No module is added or removed. `go.mod` does not change. *Amended by
deviation D4: Step 5 adds `github.com/atotto/clipboard v0.1.4 // indirect`,
a dependency of bubbles' `textinput` that only its tests import.*

### Out of scope

* **The width method** (0003-REPORT §1.6). It needs the direct drawing in
  [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md),
  because the lipgloss canvas fixes its own method.
* **Render performance and the under-used Charm features** (§1.5, §1.10).
  They belong to 0004.
* `git push` and tags, which are the owner's.
* Any change in pi-go.

## Rules for every step

1. **Order.** Each step compiles, passes its tests and passes the pre-add
   gate before the next step starts.
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
5. **Regression first.** Each defect gets a test that fails on `v0.1.0`
   before its fix lands. The execution record quotes that failure.
6. **Commit.** One commit per step, made by the owner. At the end of each
   step the agent stops with the pre-add checks passed and the step's
   evidence in the execution record, and stages and commits nothing. The
   owner commits with `git commit --no-edit` and pushes. (2026-10-02: the
   org rules forbid agent commits to `main`, and the owner chose this over
   a `feature/` branch.)

## Implementation Steps

### Step 1: records

The owner accepts amendment A1 and answers Q5 and Q6. Record the answers
in A1, set it `accepted`, set this PLAN `in-progress`, and update
`docs/README.md`.

*2026-10-02:* the owner answered Q5 (`alt+.` and `alt+,`) and Q6
(`v0.1.1`) before approving execution. The answers are recorded, A1 is
`accepted`, and `docs/README.md` says so. This PLAN stays `proposed`; the
rest of this step, setting it `in-progress`, waits for that approval.

### Step 2: `tuitest`

* **The flag.**
  * Remove the `init` that registers `-update`.
  * Register `-tuitest.update` instead. A dotted name is legal for the
    `flag` package, and no consumer will define it.
  * `Golden` rewrites files when any of these holds:
    * `-tuitest.update` is set;
    * `TUITEST_UPDATE` is `1` or `true`;
    * a boolean flag named `update` exists in the test binary and is set.
* **Idioms.** `Annotate` walks lines with `strings.Lines`. Subtests that
  range over a map range over `slices.Sorted(maps.Keys(…))`.
* **Tests:**
  * a test package `tuitest/internal/clash` defines
    `var update = flag.Bool("update", false, "")`, imports `tuitest` and
    calls `Golden`. It must run without a panic. On `v0.1.0` it panics
    with `flag redefined: update`;
  * each of the three triggers rewrites a planted, stale golden file in a
    temporary copy, and no trigger leaves it untouched;
  * with no trigger, a mismatch fails and writes nothing.
* **Mutations:**
  * `init` registers `update` again;
  * the environment variable is ignored;
  * a consumer's `-update` is ignored.

### Step 3: `layout`

* **Applied deltas.** `Plan` gains `Resize map[string]int`: for each
  separator of a named split, the delta `Solve` actually applied after
  clamping. A separator with nothing applied is absent.
* **`Separator.Resizable`** is true only for a named split's separators.
  A positional ID changes with the tree, and `resizeKey` never reads one.
* **Idioms:**
  * `dropFirst` uses `slices.Backward`;
  * `State.clone` uses `maps.Copy`;
  * the map ranges in tests are sorted.
* **Tests:**
  * a delta of +1000 against a limit reports the limit in `Plan.Resize`;
  * an unnamed split's separators are not `Resizable`, and a named
    split's are;
  * the existing property and fuzz tests pass unchanged, so applied
    deltas never break tiling.
* **Mutations:**
  * `Plan.Resize` reports the requested delta;
  * `Resizable` is always true.

### Step 3a: a split name is used once per solve

Added 2026-10-02 by deviation D2, for MADR amendment A2.

* **The claim.** `Context` gains a set of the split names claimed in this
  solve, allocated on first use. `Split.Arrange` claims its `Name` first,
  before it validates sizes or arranges a child. An unnamed split claims
  nothing.
* **The error.** A name already claimed, or one that starts with `/`,
  returns `fmt.Errorf("%w: %q …", ErrBadSplitName, name)`, with the reason
  in the message. `ErrBadSplitName` joins the `var` block of `Solve`'s
  errors.
* **Docs.** `Split.Name`'s comment states the rule, the legal reuse across
  `Responsive` rules, and the reserved `/`. The comment on the presets'
  split-name constants says `SplitBottom` names a split in more than one
  rule on purpose.
* **Test helpers.** `TestSolverProperties` builds its trees through
  `uniqueNames`, as `TestAppliedResizeReproducesThePlan` does, so its random
  trees stay valid. `FuzzSolve` treats `ErrBadSplitName` as an expected
  error, beside `ErrBadSize` and `ErrDuplicatePane`, so the fuzzer still
  explores duplicate names.
* **Tests (the first three fail on `v0.1.0`, where `Solve` returns nil):**
  * a vertical split named `x` holding a horizontal split also named `x`:
    `Solve` fails, `errors.Is(err, ErrBadSplitName)`, and the message names
    `"x"`;
  * a custom `Node` that arranges one named split twice fails the same
    way;
  * a split named `/` fails;
  * a `Responsive` node with one name in two rules solves at both
    breakpoints, and a `State.Resize` for that name applies in each;
  * every preset solves at every size of `TestPresetsKeepTheMainPane`, and
    `TestPresetDiagrams` matches its `v0.1.0` golden files byte for byte;
  * a duplicate inside a subtree whose panes are all hidden does not fail,
    and fails once one of its panes is shown. This pins the "Neutral"
    consequence A2 records.
* **Mutations:**
  * the claim is skipped;
  * names are claimed from every rule of a `Responsive` node, so the
    presets fail;
  * the `/` check is skipped;
  * the set lives on the `Split` value instead of the `Context`, so a
    second solve is affected by the first.
* **Checks:** as for every step. `go mod tidy -diff` is clean; no module is
  added.

### Step 4: overlays and the view cache

* **The crash.** `overlayRect` places a `BelowCursor` overlay with a new
  `paneCursor()`. It returns the focused pane's cursor in screen cells and
  never consults overlays. `Cursor()` keeps its contract: the top modal
  overlay's cursor, else the focused pane's.
* **The cache:**
  * the key is a struct, `{kind, id}`, with `kind` one of pane or overlay;
  * the value holds width, height, focus and the view. This removes the
    `fmt.Sprintf`;
  * `SetPane` deletes the pane's entry, and `Pop` deletes the overlay's;
  * a replaced overlay is evicted.
* **Overlay IDs.**
  * `Push` with an ID already open removes that overlay, then pushes the
    new one on top.
  * `SendOverlay(id string, msg tea.Msg) tea.Cmd` reaches an overlay
    only.
  * `Send` keeps trying panes first, as documented.
* **Overlay sizes.** `resolve` keeps a size per open overlay. It sends a
  `SizeMsg` to each overlay whose content size changed.
* **Tests (each fails on `v0.1.0`):**
  * pushing a modal `BelowCursor` overlay returns, and the overlay sits
    one row under the focused pane's cursor;
  * after `SetPane`, a replacement `Changer` that reports no change is
    drawn with its own view;
  * a pane and an overlay that share an ID each show their own view;
  * after a resize from 120×40 to 60×20, an open overlay has received its
    new content size;
  * pushing an ID twice leaves one overlay with the second content;
  * after `Pop`, the cache holds no entry for the overlay.
* **Mutations:**
  * `overlayRect` calls `Cursor()` again;
  * `SetPane` keeps the entry;
  * the cache key drops `kind`;
  * `resolve` skips overlays.

### Step 5: focus as messages

* **Messages.** `PaneFocusMsg{}` and `PaneBlurMsg{}` are delivered with
  `Send`, so the value a pane's `Update` returns is kept. They are sent:
  * on `Init`, focus to the focused pane;
  * on every focus change, blur to the old pane and then focus to the new
    one;
  * when `resolve` moves focus off a hidden pane.
* **`Focuser`** is still called, before the message, so a `v0.1.0` pointer
  pane works unchanged. Its doc says to implement one or the other, not
  both.
* **Modal overlays:**
  * `Push` of a modal overlay blurs the focused pane and focuses the
    overlay's pane;
  * `Pop` blurs the popped pane, and, when no modal overlay remains,
    focuses the focused pane again;
  * a non-modal overlay changes no focus.
* **`Wrap` and `Model[M]`,** in `model.go`:

  ```go
  type Bubble[M any] interface {
      Update(tea.Msg) (M, tea.Cmd)
      View() string
  }
  type Model[M Bubble[M]] struct {
      M M // the hosted model, readable and settable by the program
      // unexported: options
  }
  func Wrap[M Bubble[M]](m M, opts ...WrapOption[M]) *Model[M]
  func (p *Model[M]) Update(msg tea.Msg) (Pane, tea.Cmd)
  func (p *Model[M]) View(width, height int) string
  func (p *Model[M]) Cursor() *tea.Cursor
  // Options, each overriding what Wrap finds by type assertion:
  func OnSize[M Bubble[M]](f func(m *M, width, height int)) WrapOption[M]
  func OnFocus[M Bubble[M]](f func(m *M) tea.Cmd) WrapOption[M]
  func OnBlur[M Bubble[M]](f func(m *M)) WrapOption[M]
  func WithCursor[M Bubble[M]](f func(m M) *tea.Cursor) WrapOption[M]
  func WithKeys[M Bubble[M]](f func(m M) []key.Binding) WrapOption[M]
  ```

  * **Size.** `SizeMsg` calls `SetSize`, or `SetWidth` and `SetHeight`, on
    `*M` when it has them. bubbles `list`, `viewport`, `textinput` and
    `textarea` have them with pointer receivers.
  * **Focus.** `PaneFocusMsg` and `PaneBlurMsg` call `Focus() tea.Cmd` and
    `Blur()` on `*M`. bubbles `textinput.go:268,275` and
    `textarea.go:799,806` have pointer receivers.
  * **Cursor.** `Cursor` calls `Cursor() *tea.Cursor` on `M`, as bubbles
    `textinput.go:916` and `textarea.go:1751` define it.
  * **View.** `View` clips `m.View()` to the size, as the workspace does
    every view.
* **Tests:**
  * a value-type pane records `PaneFocusMsg` and `PaneBlurMsg` in order
    across `Init`, a focus cycle, a click and a hidden pane;
  * a wrapped bubbles `textinput` is focused, gets typed keys, and shows
    its cursor at the pane's offset. On `v0.1.0` it is never focused;
  * a modal `Push` blurs the pane beneath, and `Pop` focuses it again;
  * a non-modal `Push` sends neither message;
  * a wrapped `viewport` receives its size;
  * each option overrides the found method.
* **Mutations:**
  * focus messages go through `Update`, but the returned value is
    dropped;
  * `Pop` does not refocus;
  * `Wrap` ignores `SetSize`;
  * a non-modal overlay blurs.

### Step 6: keys, resize and tidying

* **Keys.** `FocusNext` and `FocusPrev` take the defaults Q5 chose:
  `alt+.` and `alt+,`. `DefaultKeyMap`'s doc names the legacy
  sequence introducers that a default must avoid: `[`, `]`, `O`, `P`, `_`,
  `^`, `X` and `\`.
* **Resize:**
  * `Resize` solves with the new delta and stores the separator's entry
    from `Plan.Resize`;
  * a separator that is not `Resizable` is ignored by `Resize`, by keyboard
    resize and by a drag;
  * a later window resize does not touch the stored state.
* **Concurrency.** `Workspace`'s doc says it is not safe for concurrent
  use, and why.
* **Tests:**
  * `TestCommandsRunConcurrently` is renamed `TestCommandsAreIndependent`,
    because `Update` is serial in Bubble Tea. It runs the batch's commands
    on goroutines with `wg.Go` under `-race`;
  * holding resize right 50 times against a limit, then once left, moves
    the separator one cell;
  * a drag past the limit and back moves it at once;
  * shrinking the window and growing it again restores the layout;
  * no default binding starts with `alt+` followed by a sequence
    introducer;
  * `go fix -diff ./workspace` reports nothing.
* **Mutations:**
  * `Resize` stores the requested delta;
  * a non-`Resizable` separator is resized;
  * a default is set to `alt+[` again.

### Step 7: gates

* **Conformance by type.**
  * `internal/conformance` loads each package with `go/parser`, and
    type-checks it with `go/types` and `importer.ForCompiler(fset,
    "source", nil)`.
  * It resolves every use to its object, and fails on any of these
    outside tests:
    * the objects `os.Stdout` and `os.Stderr`;
    * the functions `fmt.Print`, `fmt.Printf` and `fmt.Println`;
    * every function of `log` that writes to standard error;
    * the `print` and `println` builtins;
    * `signal.Notify`;
    * a write to a field named `AltScreen` of `tea.View`.
  * An alias, a dot import or a method value cannot hide a use.
  * If the source importer cannot load the module's dependencies on every
    host, stop and record it under plan deviations. Do not fall back to
    name matching.
  * *Added 2026-10-02 by
    [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
    §4 (proposed):* the scan finds every module by its `go.mod`, and
    type-checks each one's packages in that module's own context, because
    one module's type checker does not load another's packages. Today the
    root is the only module, so this changes how the scan is structured,
    not what it covers.
* **A modernisation gate.** `make modernize` runs `go fix -diff ./...`
  and fails on any suggestion. `make lint` and CI's lint job run it.
* **Tests and proofs.** Each planted file is in a scratch copy, never in
  the tree. Each must fail the scan:
  * `o := os; o.Stdout.Write(…)`;
  * `import xfmt "fmt"; xfmt.Println()`;
  * `log.Print`;
  * `println`;
  * `v := tea.View{}; v.AltScreen = true`.

  A planted `for k, v := range m { out[k] = v }` must fail
  `make modernize`.
* **Mutations:**
  * the scan skips the `log` package;
  * the scan stops resolving aliases;
  * `make modernize` drops `-diff`.

### Step 8: documentation and close-out

* **`docs/guides/building-workspaces.md`:**
  * focus messages, `Wrap` and a bubbles example;
  * overlay IDs and `SendOverlay`;
  * resize behaviour, and the new default keys;
  * split names: unique in a solve, legal to reuse across `Responsive`
    rules, and never starting with `/` (MADR A2).
* **`AGENTS.md`:** the pre-add section says `-tuitest.update` or
  `TUITEST_UPDATE=1`, and names `make modernize`.
* **`docs/architecture.md`:** the conformance scan and the gate.
* **Release notes for `v0.1.3`** (deviation D3) in the execution
  record, which also say what `v0.1.1` and `v0.1.2` carry. They name the
  behaviour changes: the flag, the default keys, focus messages,
  `Push` replacing an open ID, and `Solve` failing with `ErrBadSplitName`
  for a split name used twice in one solve or starting with `/`.
* **Verification** as below. Mark `complete` after CI is green on the
  pushed tree. The owner tags.

## Verification

* Every step's regression test failed on `v0.1.0` before its fix, and the
  failure is quoted.
* Every step's mutations are killed.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint`, `make modernize` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`.
* All `v0.1.0` golden files pass byte for byte, except those the default
  keys change. Each changed file is read and listed.
* `go mod tidy -diff` is clean, and `go.mod` is unchanged.
* The identifier scan of 0001-PLAN V7 finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–8 and tags `v0.1.3`
  (deviation D3: `v0.1.1` and `v0.1.2` already tag Steps 1–4). pi-go, the one
  consumer, picks it up under its own records.
* **Rollback.** Before the push, each step is one local commit. After it,
  a consumer pins `v0.1.0`, and the defects are fixed forward in a
  `v0.1.2`. No state format changes: `layout.State` JSON from `v0.1.0`
  loads unchanged.

## Execution Record

### Fixed inputs

* **The development host:** macOS, go1.27.1, golangci-lint through
  `make lint`, govulncheck through `make pre-add-check`.
* **The Windows test host,** reached over SSH and never named in a record:
  go1.27.1 windows/amd64 in a Git Bash (MSYS2) login shell. Each run copies
  the tracked and untracked, non-ignored files of the working tree into a
  new directory under the host's `%TEMP%`, runs there, and removes it.
  Records replace the user-profile path with `<user>`.

### Deviation D1 (2026-10-02): Steps 1 and 2 hand off together

Rule 6 makes each step one commit. Step 1 changes only this PLAN's status and
its `docs/README.md` row, and Steps 1 and 2 both write this execution record,
so they cannot be split by path. They hand off as one change, and the owner
commits it once. From Step 3 on, each step hands off alone.

### Step 1: records (2026-10-02)

* The owner approved execution on 2026-10-02 ("approved to proceed").
* A1's answers were already recorded and A1 `accepted` (Step 1's note
  above). This PLAN is `in-progress`, and `docs/README.md` says so.

### Step 2: `tuitest` (2026-10-02)

**What changed.**

* **The flag.** The `init` that registered `-update` is gone. `tuitest`
  registers `-tuitest.update`. `Golden` and `Text` rewrite files when
  `-tuitest.update` is set, when `TUITEST_UPDATE` is `1` or `true` (any
  case), or when the test binary defines a boolean `-update` and it is set.
  The missing-file message names both switches.
* **`Annotate`** walks lines with `strings.Lines`. An empty string is still
  one empty line, `"  0|\n"`. The output is unchanged: the new
  `TestAnnotateLines` table also passed against `v0.1.0`'s `Annotate`, on a
  scratch copy.
* **No map iteration** exists in this package's tests, so the sorted-keys
  idiom has nothing to change here. The map ranges 0003-REPORT §1.12 names
  are in `layout`, `glyph` and `workspace`, under Steps 3 and 6.
* **New files:** `tuitest/internal/clash` (`doc.go`, `clash_test.go`, and
  its eight golden files, each read before this record was written).

**Regression first.** `tuitest/internal/clash` declares
`var update = flag.Bool("update", false, …)`, as a consumer does. On a
scratch copy of `v0.1.0` with only that package added:

```text
<tmp>/clash.test flag redefined: update
panic: <tmp>/clash.test flag redefined: update
github.com/maccavelli/go-tui-lib/tuitest/internal/clash.init()
	…/tuitest/internal/clash/clash_test.go:12 +0x40
FAIL	github.com/maccavelli/go-tui-lib/tuitest/internal/clash	0.458s
```

With the fix, the package passes.

**Tests:** `TestGoldenUpdateTriggers` (each of `-tuitest.update`,
`TUITEST_UPDATE=1`, `TUITEST_UPDATE=TRUE` and a consumer's `-update`
rewrites a planted, stale golden file), `TestGoldenWithoutTriggerWritesNothing`
(the stale file and three missing ones are reported, and nothing is
written), `TestAnnotateLines`, and the changed `TestGoldenMissingFile`.
Each test clears every switch first, so it does not depend on how `go test`
was run.

**Mutation proofs**, each on a scratch copy; none survived:

| Mutation | Killed by |
| :--- | :--- |
| `init` registers `update` again | `clash.test flag redefined: update` (panic) |
| the environment variable is ignored | `TestGoldenUpdateTriggers/TUITEST_UPDATE=1` and `=TRUE`: `the stale golden file was not rewritten` |
| a consumer's `-update` is ignored | `TestGoldenUpdateTriggers/the_consumer's_-update`: `the stale golden file was not rewritten` |

**Checks.**

* `make pre-add-check FILES=…` on the four Go files: `4 file(s) clean
  (gofmt, golangci-lint, go vet, go test, govulncheck)`.
* `make lint`: `0 issues` for linux, darwin and windows.
* `go test -race -count=1 ./...` and `LC_ALL=C go test -count=1 ./...`:
  every package `ok`.
* `go mod tidy -diff` and `go fix -diff ./tuitest/...`: no output, exit 0.
* **The Windows test host:** `go vet`, `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...` exited 0, every package `ok`.
  * The first run failed in `internal/conformance` with
    `C:\Users\<user>\…\glyph\._glyph.go:1:1: illegal character NUL`. The
    cause was the transfer, not the code: macOS `tar` had added AppleDouble
    `._*` metadata files. The tarball is now built with `COPYFILE_DISABLE=1`
    and `--no-mac-metadata`, holds no `._` entry, and the re-run passed.

**Not done in this step.** The guides and `AGENTS.md` still say `-update`.
Step 8 changes them, as planned.

### Step 3: `layout` (2026-10-02)

**What changed.**

* **`Plan.Resize map[string]int`.** For each separator that `State.Resize`
  moved, the delta `Solve` applied after clamping. A delta clamped to 0 is
  absent, and the map is nil when nothing was applied. `Split.resize` now
  takes the `Context` so it can record what it applied.
* **`Separator.Resizable`** is true for a named split's separators only.
* **Idioms:** `dropFirst` walks `slices.Backward(kept)`; `State.clone` uses
  `maps.Copy`; `TestPresetDiagrams`, `TestPresetsKeepTheMainPane` and
  `BenchmarkSolvePresets` walk the presets in sorted order
  (`presetNames`).
* **Docs:** `State.Resize` says a host stores the applied delta.

**Regression first.** On a scratch copy of `v0.1.0`:

* the new tests do not build there, because the API is new:
  `base.Resize undefined (type Plan has no field or method Resize)` and
  `s.Resizable undefined (type Separator has no field or method Resizable)`;
* the dead zone itself, with `v0.1.0`'s own API:

  ```text
  --- FAIL: TestDeadZoneV010 (0.00s)
      deadzone_test.go:17: after +1000 then -5, side stayed 20 cells (stored delta 995): the separator did not move
  ```

The workspace's use of `Plan.Resize`, which removes that dead zone for a
user, is Step 6.

**Tests:**

* `TestPlanReportsAppliedResize`: +1000 against side's Min reports the
  applied 30; storing it and moving back 5 leaves side at 25; a delta
  clamped to 0 is absent.
* `TestSeparatorResizable`: a named split's separators are `Resizable`, an
  unnamed one's are not.
* `TestAppliedResizeReproducesThePlan` (added beyond the PLAN's list): over
  2,000 random trees with random deltas on every resizable separator, the
  split stays tiled, and solving with `Plan.Resize` gives the same panes and
  the same `Plan.Resize`. That is the property Step 6 relies on.
* The existing property and fuzz tests pass unchanged, and every `v0.1.0`
  golden diagram matches byte for byte.

**Found, and not changed.** The first run of the new property test failed on
tree 0. The test's random generator names splits after the pane count, so
two splits can share a name. Two splits with one `Name` share their
separators' keys, so one `State.Resize` entry moves both, and `Plan.Resize`
keeps the last. The test now renames the tree's splits uniquely
(`uniqueNames`), and the property holds. The behaviour itself predates this
PLAN and is outside its scope: `Split.Name`'s documentation implies, but
does not state or enforce, that names are unique. It was reported to the
owner, not fixed in this step. Deviation D2 below adds the fix as Step 3a.

**Mutation proofs**, each on a scratch copy; none survived:

| Mutation | Killed by |
| :--- | :--- |
| `Plan.Resize` reports the requested delta | `Plan.Resize[row:0] = 1000, true; want the applied 30, not the asked 1000` |
| `Resizable` is always true | `unnamed split: separator /:0 Resizable = true, want false` |
| a delta clamped to 0 is still reported | `TestPlanReportsAppliedResize` (the clamped-to-0 case) |

**Checks.**

* `make pre-add-check FILES=…` on the four files: `4 file(s) clean`, and
  again on `state.go` after its doc comment: `1 file(s) clean`.
* `make lint`: `0 issues` for linux, darwin and windows.
* `go test -race -count=1 ./...` and `LC_ALL=C go test -count=1 ./...`:
  every package `ok`.
* `go mod tidy -diff` and `go fix -diff ./layout/...`: no output, exit 0.
* `FuzzSolve` for 15 s: `PASS`, about 2.9 million executions; no corpus file
  was written to the tree.
* **The Windows test host:** `go vet`, `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...` exited 0, every package `ok`.

### Deviation D2 (2026-10-02): Step 3a, unique split names

* **Found.** Step 3's property test showed that two splits with one `Name`
  share their separators' IDs (Step 3, "Found, and not changed").
* **Decision.** The owner asked for the solution to be specified, more than
  one option weighed, the most idiomatic and project-native one chosen, and
  the result added to this PLAN as its own phase. MADR amendment A2 records
  the six options and the choice. The choice is option C: `Context` claims
  each split name as it is arranged, and a second claim, or a name starting
  with `/`, fails `Solve` with `ErrBadSplitName`.
* **Scope added.** Step 3a, in `layout/` only. Step 8 gains a guide bullet
  and a release-notes item. No module, no other package.
* **Order.** Step 3a runs after Step 3 and before Step 4, so Step 6's
  resize, which finds a separator by ID, starts from unique IDs. It hands
  off alone, by rule 6.

### Step 3a: a split name is used once per solve (2026-10-02)

The owner approved Step 3a ("proceed") after committing Step 3 and the A2
records (`572ea7f`).

**What changed.**

* **`Context.claimSplit`** and the `Context.splits` set, allocated on first
  use. `Split.Arrange` claims its name before anything else. A name already
  claimed in the solve fails with
  `layout: a split name is used twice, or is reserved: "x" names two splits arranged in one layout`.
  A name starting with `/` fails with
  `…: "/0" starts with "/", which positional separator IDs use`.
* **`ErrBadSplitName`** joins `ErrDuplicatePane`, `ErrBadArea` and
  `ErrBadSize`.
* **Docs:** `Split`'s comment states the rule, the legal reuse across
  `Responsive` rules and the reserved `/`, and gives the positional form
  `"/<path>:<i>"`. The presets' constant block says `SplitBottom` is
  reused on purpose.
* **Test helpers:** `TestSolverProperties` builds through `uniqueNames`;
  `FuzzSolve` accepts `ErrBadSplitName`.

**Regression first.** The new tests use Step 3's API (`Plan.Resize`), so
they cannot compile on `v0.1.0`. They were run instead on a scratch copy of
`572ea7f` (`v0.1.0` plus Step 3), whose name handling is `v0.1.0`'s, with
one scratch-only file declaring the sentinel so the tests compile:

```text
--- FAIL: TestSplitNameUsedOncePerSolve (0.00s)
    layout_test.go:247: nested splits share a name: Solve error = <nil>, want ErrBadSplitName naming "x"
    layout_test.go:247: a custom node arranges one name twice: Solve error = <nil>, want ErrBadSplitName naming "y"
    layout_test.go:247: a name starts with /: Solve error = <nil>, want ErrBadSplitName naming "/"
    layout_test.go:247: a name starts with /0: Solve error = <nil>, want ErrBadSplitName naming "/0"
--- FAIL: TestSplitNameInHiddenSubtree (0.00s)
    layout_test.go:286: once the subtree is shown: <nil>, want ErrBadSplitName
```

`TestSplitNameReusedAcrossResponsiveRules` passed there, as it must: it
guards the reuse that option B would have broken.

**Tests:** `TestSplitNameUsedOncePerSolve` (nested splits, a custom `Node`
arranging one name twice, `/` and `/0`, and one tree solved twice);
`TestSplitNameReusedAcrossResponsiveRules` (one name in a rule and in
`Else`, solved at 120 and 80 columns, with `r:0` applied as 3 in each);
`TestSplitNameInHiddenSubtree` (no error while the duplicate's panes are
hidden; `ErrBadSplitName` once one is shown). Every preset golden diagram
matched its `v0.1.0` file byte for byte, and no golden file changed.

**Mutation proofs**, each on a scratch copy; none survived:

| Mutation | Killed by |
| :--- | :--- |
| the claim is skipped | `TestSplitNameUsedOncePerSolve` (all four cases: `Solve error = <nil>`) and `TestSplitNameInHiddenSubtree` |
| names are claimed from every rule of a `Responsive` node | `TestSplitNameReusedAcrossResponsiveRules`, `TestPresetDiagrams` and `TestPresetsKeepTheMainPane` |
| the `/` check is skipped | `TestSplitNameUsedOncePerSolve`: the `/` and `/0` cases |
| claims outlive the solve, in a package-level set | `TestSplitNameUsedOncePerSolve` (the second solve), the preset tests and `TestSolverProperties` |

The fourth mutation's first form did not compile, because it used the
package-level set without declaring it. By rule 2 it was replaced: the
mutation runner now applies more than one edit, and the declaration was
added. That form compiled and was killed as above. The runner had counted
the build failure as a kill. Every kill in the table was checked to be a
test failure, not a build failure.

**Checks.**

* `make pre-add-check FILES=…` on the four files: `4 file(s) clean`.
* `make lint`: `0 issues` for linux, darwin and windows.
* `go test -race -count=1 ./...` and `LC_ALL=C go test -count=1 ./...`:
  every package `ok`.
* `go mod tidy -diff` and `go fix -diff ./layout/...`: no output, exit 0.
* `FuzzSolve` for 15 s: `PASS`, about 3.1 million executions, with
  duplicate names now an expected error; no corpus file was written to the
  tree.
* **The Windows test host:** `go vet`, `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...` exited 0, every package `ok`.

### Step 4: overlays and the view cache (2026-10-02)

The owner committed Step 3a under an explicit, one-time authorization
(`d379c4d`) and approved Step 4 ("proceed").

**What changed.**

* **The crash.** `overlayRect` places a `BelowCursor` overlay with the new
  `paneCursor()`, the focused pane's cursor in screen cells, which never
  looks at overlays. `Cursor()` keeps its contract, and the shared logic is
  `placeCursor(p, in)`.
* **The cache.** Its key is `viewKey{kind, id}`, with `kind` `paneView` or
  `overlayView`, and its value `cached{width, height, focused, view}`. The
  `fmt.Sprintf` key is gone, with render.go's `fmt` import. `renderBox`
  takes the `viewKey`, and uses its `id` as the fallback title.
  * `SetPane` deletes the pane's entry.
  * Closing an overlay, by `Pop` or by a replacing `Push`, deletes its entry
    and its size, through the new `closeOverlay(i)`.
* **Overlay IDs.** `Push` with an ID already open closes that overlay, then
  pushes the new one on top. `SendOverlay(id string, msg tea.Msg) tea.Cmd`
  reaches an overlay only. `Send` still tries panes first, and now falls
  back to `SendOverlay`. `Overlay.ID`'s comment states the separate
  namespace and the replacement.
* **Overlay sizes.** `Workspace.osizes` holds each open overlay's last
  content size. `Push` records it as it sends the first `SizeMsg`, and
  `resolve` sends a new `SizeMsg` to each overlay whose size changed.
* **Focus, unchanged in kind.** A replacing `Push` blurs the old overlay's
  `Focuser`, as `Pop` always did. Step 5 replaces overlay focus with
  messages.

**Regression first.** On a scratch copy of `d379c4d`, whose `workspace` is
`v0.1.0`'s, with the new `overlay_test.go` and one scratch-only file giving
`SendOverlay` the old behaviour (it calls `Send`). Each test ran alone,
because the first kills the test binary:

```text
## TestModalOverlayBelowCursor
runtime: goroutine stack exceeds 1000000000-byte limit
fatal error: stack overflow
## TestSetPaneDropsTheCachedView
overlay_test.go:38: after SetPane, an unchanged replacement shows the old view:
## TestPaneAndOverlayWithOneID
overlay_test.go:52: pane viewed 1 times, overlay 0 times; frame:
## TestOverlayIsToldItsNewSize
overlay_test.go:72: overlay sizes [{98 28}], want 98x28 then 58x18
## TestPushReplacesAnOpenID
overlay_test.go:87: overlays [x y x], want [y x]: the open x replaced and moved to the top
## TestPopEvictsTheOverlay
overlay_test.go:105: after Pop the cache still holds box:qqpopped
```

**Tests** (`workspace/overlay_test.go`): a modal `BelowCursor` overlay sits
one row under the pane's cursor, and `Cursor()` is then the overlay's; an
unchanged `Changer` that replaces a pane is drawn; a pane and an overlay with
one ID each draw their own view, `SendOverlay` reaches the overlay only, and
`Send` the pane; an overlay is told 98×28, then 58×18 after a resize from
120×40 to 60×20, and not told twice; pushing `x`, `y`, `x` leaves `[y x]`,
draws the second `x`, and sizes it once; after `Pop` the cache holds nothing
for the overlay. Every existing workspace test passed unchanged, and no
golden file changed.

**Mutation proofs**, each on a scratch copy; none survived. The first two
rows are the PLAN's; the last two were added for `Push` and `Pop`:

| Mutation | Killed by |
| :--- | :--- |
| `overlayRect` calls `Cursor()` again | `TestModalOverlayBelowCursor`: `fatal error: stack overflow` |
| `SetPane` keeps the entry | `TestSetPaneDropsTheCachedView` |
| the cache key drops `kind` (overlays keyed as panes) | `TestPaneAndOverlayWithOneID` |
| `resolve` skips overlays | `overlay sizes [{98 28}], want 98x28 then 58x18` |
| `Push` does not replace an open ID | `overlays [x y x], want [y x]` |
| closing an overlay keeps its cached view | `TestPopEvictsTheOverlay` |

Each kill was a test failure on code that compiled; the first was checked to
be the runtime's stack overflow.

**Checks.**

* `make pre-add-check FILES=…` on the four files: `4 file(s) clean`.
* `make lint`: `0 issues` for linux, darwin and windows.
* `go test -race -count=1 ./...` and `LC_ALL=C go test -count=1 ./...`:
  every package `ok`.
* `go mod tidy -diff`: no output, exit 0.
* `go fix -diff ./workspace/...` exits 1 with `strings.SplitSeq`,
  `maps.Copy` and two `wg.Go` suggestions. They are the same, line for line,
  on the tree before this step: 0003-REPORT §1.13's, which Step 6 owns
  ("`go fix -diff ./workspace` reports nothing"). This step adds none.
* **The Windows test host:** `go vet`, `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...` exited 0, every package `ok`.

### Deviation D3 (2026-10-02): `v0.1.1` and `v0.1.2` tag Steps 1–4

* **Found.** After Step 4 and the 0010 records were pushed, the owner tagged
  that commit, `4253e9d`, as `v0.1.1` and then as `v0.1.2`, and pushed both
  tags. Both hold Steps 1, 2, 3, 3a and 4 of this PLAN, with the records
  of that day. Steps 5–8 are in neither. The PLAN said `v0.1.1` would carry
  Steps 1–8.
* **Evidence.** `git ls-remote --tags origin`: `refs/tags/v0.1.1^{}` and
  `refs/tags/v0.1.2^{}` are both `4253e9d`; `v0.1.0` is unchanged. The
  module proxy was not queried, because a query can make it cache a
  version.
* **Not a defect in the code.** Every step in those tags passed its gates,
  and nothing in them is broken, so neither is retracted.
* **Decision** (the owner, picked from options, 2026-10-02):
  * Both tags stay. A published tag is never moved or deleted
    ([0010-REPORT-nested-modules-and-adapter-sources.md](../reports/0010-REPORT-nested-modules-and-adapter-sources.md)
    §1, `ref/mod.md:3355-3359`), and a proxy and the checksum database may
    already hold them.
  * Steps 5–8 ship as `v0.1.3`, a patch for the reasons MADR A1 gives.
  * The two identical tags are recorded here and in `v0.1.3`'s release
    notes. No `retract` directive is added.
* **Changed.** The Goal, Step 8's release notes and Rollout name `v0.1.3`.
  MADR A1 and A2 carry a dated version note. The records that named
  `v0.1.1` as the completed hardening now name `v0.1.3`:
  [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
  and its PLAN, [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md),
  and [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md).
* **Unchanged.** Steps 5–8, and this PLAN's file name.

### Deviation D4 (2026-10-02): Step 5's `textinput` test adds an indirect requirement

* **Found.** Step 5's test of a wrapped bubbles `textinput` imports
  `charm.land/bubbles/v2/textinput`, which imports
  `github.com/atotto/clipboard`. That module is not in this module's
  `go.sum`, so the test does not build, and the Scope says `go.mod` does not
  change.
* **Evidence.** `go mod tidy` on scratch copies, one bubbles package
  imported at a time by a test file: `textinput` and `textarea` each add
  `github.com/atotto/clipboard v0.1.4 // indirect`; `viewport` adds
  nothing; `list` adds that line and `github.com/sahilm/fuzzy`. The cause
  is bubbles v2.2.1's own imports, not this PLAN's code.
* **Resolutions offered:**
  * accept the indirect line, and test the real `textinput` as planned
    (recommended);
  * test `Wrap` against `viewport` and a test model with `textinput`'s
    method set, and prove the real `textinput` in a nested test-only module
    once 0010-PLAN's tooling exists;
  * stop Step 5.
* **Decision** (the owner, picked from options, 2026-10-02): accept the
  indirect line.
* **Consequence.** `go.mod` gains `github.com/atotto/clipboard v0.1.4 //
  indirect`. Only tests import it, so no program compiles it. It still
  reaches every consumer's `go.sum` and module graph, as
  [0010-REPORT-nested-modules-and-adapter-sources.md](../reports/0010-REPORT-nested-modules-and-adapter-sources.md)
  §2 measured for any root requirement. MADR A1 names it, as AGENTS.md
  requires of every required module.
* **Scope added.** `go.mod` and `go.sum`, in Step 5.

### Step 5: focus as messages (2026-10-02)

The owner approved Step 5 ("proceed") after committing D3's records
(`831a2dc`). Deviation D4 was raised, answered and recorded before any
code was written.

**What changed.**

* **`focus.go` (new).** `PaneFocusMsg` and `PaneBlurMsg`. One rule decides
  who has the keyboard: the top modal overlay's pane, or else the focused
  pane (`focusTarget`). Every change goes through `moveFocus`, which sends
  the old target `PaneBlurMsg` and the new one `PaneFocusMsg`. Each is sent
  with `Send` (or `updateOverlay`), so the value `Update` returns is kept.
  A `Focuser` is called first, so a `v0.1.0` pointer pane works unchanged.
* **Where focus moves:** `Init` (the target gains focus); `Focus` (old
  blurs, new gains; under a modal overlay the change waits for the overlay
  to close); `resolve`, when it moves focus off a pane no longer shown;
  `Push` of a modal overlay; and `Pop` or a replacing `Push` of the overlay
  that had the keyboard. Replacing the top modal overlay blurs the old one
  and focuses the new one, with nothing beneath focused in between.
* **`model.go` (new).** `Bubble[M]`, `Model[M]`, `Wrap`, `WrapOption` and
  the options `OnSize`, `OnFocus`, `OnBlur`, `WithCursor` and `WithKeys`,
  as the PLAN gives them. `SizeMsg` calls `SetSize`, or `SetWidth` and
  `SetHeight`, on `*M`; the focus messages call `Focus` and `Blur` on `*M`;
  `Cursor` and `Keys` look on `M`, then `*M`. Every other message reaches
  the model's `Update`. `View` clips to the size.
* **Docs.** `Focuser`'s comment says to implement it or handle the
  messages, not both. `Wrap`'s comment notes that bubbles `textinput` and
  `textarea` draw a virtual cursor by default (`textinput.go:164`) and give
  no `Cursor`, unless `SetVirtualCursor(false)`.
* **`go.mod`** gains `github.com/atotto/clipboard v0.1.4 // indirect`, and
  `go.sum` its two lines: deviation D4, and nothing else.

**Behaviour changes,** for the `v0.1.3` release notes:

* panes receive `PaneFocusMsg` and `PaneBlurMsg`;
* a non-modal overlay no longer has its `Focuser` called on `Push` or on
  closing: it never has the keyboard;
* `Focus` while a modal overlay is open changes the focused pane, which is
  told only when the overlay closes.

**Regression first.** On a scratch copy of `831a2dc`, with `go mod tidy`
allowed to add D4's line there:

```text
--- FAIL: TestValueFocuserV010 (0.00s)
    focus_v010_test.go:34: a value-type Focuser was focused on a copy: the stored pane has focused=false
--- FAIL: TestTextinputV010 (0.00s)
    focus_v010_test.go:44: textinput focused false, value "": it is never focused, so it drops typed keys
```

**Tests** (`workspace/focus_test.go`):

* a value-type pane records `a focus, a blur, b focus, b blur, c focus,
  c blur, a focus` across `Init`, `FocusNext`, a click and hiding `c`, and
  the stored copies hold the matching state;
* a non-modal `Push` and its close send nothing; two stacked modal
  overlays and two `Pop`s send `a blur, dlg focus, dlg blur, dlg2 focus,
  dlg2 blur, dlg focus, dlg blur, a focus`; `Focus` under a modal overlay
  sends nothing until `Pop`;
* a pointer `Focuser` is called and also gets the message;
* a wrapped `textinput` with the terminal cursor is focused, takes typed
  keys, is given the content width, places its cursor at the pane's
  offset, and is blurred by a modal overlay;
* a wrapped `viewport` gets its content size;
* each option replaces the method `Wrap` would find.

The first run of the `textinput` test failed with
`cursor <nil>, want the textinput's <nil>`: bubbles v2's textinput uses a
virtual cursor by default. The test now calls `SetVirtualCursor(false)`, as
a program wanting the terminal cursor does; no assertion was loosened.

**Mutation proofs**, each on a scratch copy; none survived:

| Mutation | Killed by |
| :--- | :--- |
| focus messages go through `Update`, but the returned value is dropped | `TestFocusReachesValuePanes` |
| `Pop` does not refocus | `focus messages [… dlg blur], want [… dlg blur a focus]` |
| `Wrap` ignores `SetWidth` and `SetHeight` | `textinput width 0, want the content width 38`; `viewport 0x0, want 48x10` |
| a non-modal overlay takes focus | `TestModalOverlayMovesFocus` |

They ran before one lint fix in `model.go` (below), which touches none of
their anchors.

**Checks.**

* `make pre-add-check FILES=…` on the five files: first run, one gocritic
  `evalOrder` finding on `return p, p.focus()`, which mutates `p.M` while
  the results are evaluated. The call now runs first. Second run:
  `5 file(s) clean`, govulncheck included with D4's module.
* `make lint`: `0 issues` for linux, darwin and windows.
* `go test -race -count=1 ./...` and `LC_ALL=C go test -count=1 ./...`:
  every package `ok`; no golden file changed.
* `go mod tidy -diff`: no output, exit 0.
* `go fix -diff ./workspace/...`: the same four suggestions as before
  Step 4, which Step 6 owns; none new.
* **The Windows test host:** `go vet`, `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...` exited 0, every package `ok`.
