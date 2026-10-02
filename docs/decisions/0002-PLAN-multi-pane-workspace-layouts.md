---
status: in-progress
date: 2026-10-01
associated-madr: "0002-MADR-multi-pane-workspace-layouts.md"
---
# Implement multi-pane workspaces (`v0.1.0`)

Associated MADR: [0002-MADR-multi-pane-workspace-layouts.md](0002-MADR-multi-pane-workspace-layouts.md)

## Goal

Ship `tuitest`, `glyph`, `theme`, `layout` and `workspace` as `v0.1.0`, so
that pi-go can host its agent session in a main pane, metrics in a left or
right sidebar, logs in a bottom pane, and a footer, using the MADR's presets
or its own trees.

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag `v0.1.0`.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0002-*`, `docs/README.md` | accept the records |
| 2 | `tuitest/` | golden matrix helper |
| 3 | `glyph/` | glyph tables |
| 4 | `theme/` | palettes, roles, styles |
| 5 | `layout/`; `Makefile` (`fuzz`), `scripts/go-fuzz.sh` and its test (from go-core-lib), `.github/workflows/ci.yml` (the fuzz step) | solver, responsive rules, presets, state, and fuzzing in CI |
| 6 | `workspace/`; `internal/conformance/` (a test-only package) | pane host, and the source scan for 0001 §6 rules 1 and 2 |
| 7 | `workspace/example_test.go`, `workspace/testdata/golden/agent-*` | the agent-session example |
| 8 | `README.md`, `docs/`, `docs/guides/building-workspaces.md` | documentation, release notes, close-out |

`go.mod` and `go.sum` gain each requirement in the step that adds its first
import: `charm.land/lipgloss/v2` and `github.com/charmbracelet/colorprofile`
in Step 4, `charm.land/bubbletea/v2` and `charm.land/bubbles/v2` in Step 6,
and `github.com/charmbracelet/x/ansi` in Step 2. Each enters at its newest
release, which on 2026-10-01 was lipgloss v2.0.6, bubbletea v2.0.10, bubbles
v2.2.1, colorprofile v0.4.3 and x/ansi v0.11.8.

### Out of scope

* Standard panes (log tail, metrics, scrolling text), overlay helpers
  (dialog, picker, palette, toast), a help footer and `updatetea`. Each is
  its own record (MADR §1).
* Any change in pi-go, ocp-login or go-core-lib.
* `apicheck`, which comes with the `v1` record.
* `git push` and tags. The owner pushes this PLAN's commits and the
  scaffold's together (0001-MADR §8), and tags `v0.1.0`.

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

## Implementation Steps

### Step 1: records

The owner accepts the MADR, answering Q1–Q4. Record the answers, set the
MADR `accepted` and this PLAN `in-progress`, and update `docs/README.md`. If
Q2 asks for re-wording, re-word 0001-MADR's ocp-login evidence to patterns
here too, as an amendment, before any push.

### Step 2: `tuitest`

* `Matrix{Widths []int}` × {`colour`, `nocolor`} × {`utf8`, `ascii`};
  `Golden(t, name, render func(Case) string)`; an `-update` flag.
* Each golden line is prefixed with its cell width (`x/ansi`), so width
  drift shows in the diff.
* **Tests:** a planted render that changes one cell fails, with a diff
  naming the case and the line. `-update` rewrites only the named cases.
* **Mutations:**
  * the comparison ignores the last line;
  * the width prefix uses bytes.

### Step 3: `glyph`

* `Set` with `Unicode()` and `ASCII()`; border sets (light, rounded, heavy,
  double, ASCII) as `lipgloss`-free structs of strings; separators, focus
  marker, ellipsis, scroll indicators.
* **Tests:**
  * every `ASCII()` field is ASCII;
  * every field of both sets is exactly one cell;
  * no field is empty. These checks walk the struct by reflection, so a new
    field is covered automatically.
* **Mutations:**
  * a two-cell Unicode glyph;
  * a non-ASCII rune in the ASCII set;
  * the reflection walk skips the last field.

### Step 4: `theme`

* Palettes for light, dark and unknown backgrounds. Roles: Title, Body,
  Muted, Border, BorderFocus, Accent, Success, Warning, Error and Badge.
* `New(profile colorprofile.Profile, bg Background, g glyph.Set) Theme`,
  holding built `Styles` and the `lipgloss.Border` sets made from `g`.
* **Tests:**
  * under `NoTTY` / `Ascii` profiles and `NO_COLOR`, rendered styles carry
    no colour sequence and keep structure (bold, borders);
  * the unknown palette is legible on both backgrounds: every foreground
    role clears 4.5:1 against black and white in sRGB;
  * golden swatches across the matrix.
* **Mutations:**
  * the ASCII profile keeps colour;
  * one unknown-palette role drops below the contrast floor.

### Step 5: `layout`

* The API of MADR §2: `Rect`, `Axis`, `Size` and its constructors and
  bounds, `Node`, `Pane`, `Split`, `Child`, `Responsive`, `Rule`, the
  conditions, `Solve`, `Plan`, `Separator`, `State` (JSON, versioned), and
  the four presets with their options.
* **Property tests** over random trees (depth ≤ 4, ≤ 8 panes) and areas
  (1..300 × 1..100):
  * leaves and gaps tile each split exactly;
  * no rectangle escapes the area;
  * Min and Max hold whenever the area allows them;
  * shrinking follows `Shrink` order, then hides;
  * the same input gives the same plan.
* **Golden diagrams.** Each preset is drawn as a box diagram at 60, 80, 120
  and 200 columns and at 20 and 40 rows, showing the responsive folds and
  both bottom spans.
* **State.** Resize deltas are clamped at every size. A state saved at 200
  columns solves at 60 without error. A JSON round trip is exact, and an
  unknown version is refused.
* **Fuzz.** `FuzzSolve`, over encoded trees and sizes, finds no panic and no
  escaping rectangle. The fuzz target joins the CI fuzz step that this step
  adds, mirroring go-core-lib's (`make fuzz`, 20 s per target on Linux).
* **Benchmarks:** solving each preset at 200 × 60.
* **Mutations:**
  * remainders are dropped, not distributed;
  * Max is ignored;
  * the shrink order is reversed;
  * a responsive rule's condition is inverted;
  * `State` deltas are not clamped.

### Step 6: `workspace`

* The API of MADR §3: `Pane` and the optional interfaces, `New`, `Update`,
  `Render`, `Cursor`, `State`, the control methods, `Overlay`, `KeyMap`
  (every binding rebindable or removable) and the options.
* **Tests, through real `tea` messages and a recording fake pane:**
  * `WindowSizeMsg` re-solves the layout, and each pane gets its size;
  * focus cycles in ring order, skipping non-focusable and hidden panes;
  * keys reach only the focused pane, or only the top modal overlay;
  * `ctrl+c` is never consumed;
  * a mouse click focuses the pane under it, the wheel reaches the pane
    under the pointer (not the focused one), and the coordinates are
    pane-local;
  * a separator drag changes `State`, and the next `Render` reflects it;
  * zoom shows one pane at the full area and restores; toggle hides and
    shows;
  * Esc pops the top overlay unless the overlay handles it;
  * the cursor is offset by the pane's rectangle, and is nil when the
    focused pane has none or is hidden;
  * a `Changer` pane that reports no change is not asked to `View` again;
  * a pane painting past its rectangle is clipped;
  * under `NO_COLOR`, the focused pane is still marked (glyph and bold).
* **Golden frames** across the matrix at 80 and 160 columns:
  * focus on each pane;
  * an open modal overlay;
  * a zoomed pane.
* **Race.** `Send`, `Broadcast` and `Render` under `-race`, with commands
  executed concurrently as Bubble Tea runs them.
* **Benchmarks:** `Render` of three panes and a footer at 200 × 60, with and
  without `Changer`.
* **Conformance.** `internal/conformance` scans every non-test Go file
  for `os.Stdout`, `os.Stderr`, `AltScreen` and `signal.Notify`, and
  fails on any.
* **Mutations:**
  * keys go to every pane;
  * the conformance scan skips one package;
  * the wheel goes to the focused pane;
  * the cursor is not offset;
  * `Changer` is ignored;
  * the overlay does not trap keys;
  * `ctrl+c` is consumed.

### Step 7: the agent-session example

* `ExampleWorkspace_agentSession` builds the four presets with fake panes:
  * a transcript that streams lines;
  * a metrics key/value view (tokens, cost, context %, latency);
  * a log tail;
  * a footer.

  It drives a resize and a focus change, and prints the frame.
* Golden scenarios at 80, 120 and 200 columns across the matrix, with the
  sidebar on the right and then on the left, the bottom pane in both spans,
  and the folds below the breakpoints.
* These fakes live in the example and the tests. They are not the standard
  panes, which are later records.

### Step 8: documentation and close-out

* **`docs/guides/building-workspaces.md`:**
  * choosing a preset;
  * writing a `Pane` and its optional interfaces;
  * a custom `Node`;
  * responsive rules;
  * persisting `State`;
  * overlays;
  * what the program owns: alternate screen, mouse mode, `ctrl+c`.
* **Doc comments:** `doc.go` in each package.
* **Docs tree:** `docs/architecture.md`, with the packages, their imports and
  the tooling; `docs/README.md` rows; README Status.
* **Release notes for `v0.1.0`** in the execution record.
* **Verification** as below. Mark `complete` after CI is green on the pushed
  tree. The owner tags.

## Verification

* Every step's mutations are killed.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`.
* `go mod tidy -diff` is clean, and `go.mod` requires exactly the five
  modules MADR §1 names.
* `depguard` refuses the Charm v1 paths in every package.
* No package writes to `os.Stdout` / `os.Stderr`, sets `AltScreen`, or
  calls `signal.Notify`. A test scans the sources for each.
* The identifier scan of 0001-PLAN V7 finds nothing. Nothing is copied from
  ocp-login (MADR §6).
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes the scaffold and Steps 1–8 together, and
  tags `v0.1.0`. pi-go adopts it under its own records.
* **Rollback.** Before the push, each step is one local commit. After it, a
  `v0.1.1` fixes forward. `v0` allows an incompatible change, and the
  release notes say so.

## Execution Record

### Step 1: records (2026-10-01)

* **Approval.** The owner answered Q1–Q4 with the recommendations and
  approved the PLAN (quoted in the MADR). Commits to `main` were
  authorized for the turn.
* **Q2's re-wording ran first,** as 0001-PLAN Phase 4 (`9c770c4`):
  0001-MADR amendment A2, the pattern-level report, and this MADR's
  working-example bullet.
* **State.** The MADR is `accepted`, and this PLAN is `in-progress`.
  `docs/README.md` indexes both.

### Step 2: `tuitest` (2026-10-01)

**What changed.**

* **`tuitest`.**
  * The types are `Case` (`Color`, `UTF8`, `Width`) with `Name()`, and
    `Matrix{Widths}` with `Cases()`, in a fixed order.
  * `Golden(t, name, m, render)` compares each case with
    `testdata/golden/<name>.<case>.golden`, reporting every differing case
    with its first differing line, escapes quoted. `Annotate` puts each
    line's cell width in front.
  * `-update` rewrites only the named cases. The flag is registered only if
    the test binary has none.
  * `Golden` takes a small `T` interface, so `*testing.T` and
    `*testing.B` both work and the package's own tests can record failures.
* **`go.mod`** requires `github.com/charmbracelet/x/ansi` v0.11.8, its
  first import.

**Lint, fixed at the source.** golangci-lint found five problems:

* `revive` confusing-naming on `golden` / `Golden`: renamed to `compare`;
* gosec G301 and G306 on the permissions: the directory is `0o750` and the
  files `0o600`;
* gosec G304 on a variable path: files are now read and written through an
  `os.Root` on the golden directory, so a name holding `..` cannot escape it;
* errcheck `check-blank` on `_ = root.Close()`: the close error is reported
  through `t.Errorf`.

**Tests:** case order and names, a round trip, one changed cell reported
with case and line, an added last line, cell-width annotation of a wide
rune and an escape, a missing golden file, an update that leaves other cases
alone, and an empty matrix.

**Mutation proofs**; none survived:

| Mutation | Killed by |
| :--- | :--- |
| the comparison ignores the last line | `an added last line: errors = [… the renderings differ …]` |
| the width prefix counts bytes | `Annotate = "  4\|a界\n 12\|…", want "  3\|a界\n  4\|…"` |
| update rewrites every case of the name | `a case outside the matrix was rewritten: ""` |

The third mutation's first anchor stopped matching after the `os.Root`
change. It was re-anchored and then killed.

**Checks.**

* `make pre-add-check` reported 2 files clean.
* `make lint` passed, as did `go test -race`, `LC_ALL=C go test` and
  `go mod tidy -diff`.
* The Windows test host passed `go vet` and `go test -race`.

### Step 3: `glyph` (2026-10-01)

**What changed.**

* **`glyph`** (standard library only).
  * `Border` holds lipgloss's thirteen fields, in its order.
  * `Set` holds four border styles (Light, Rounded, Heavy, Double), pane
    separators, the focus marker, ellipsis, scroll indicators and bar,
    bullet, and badge brackets: 64 glyphs.
  * `Unicode()`, `ASCII()` and `For(utf8)`. The ASCII set draws every
    border as `+-|`, except Double, whose rules are `=`. The ASCII ellipsis
    is `~`, one cell, where `...` would be three.

**Tests.**

* A reflection walk checks every field of both sets: non-empty, one cell
  (`x/ansi`, in the test only) and one rune. It asserts that it walked 64,
  so a skipped field fails.
* The ASCII set is printable ASCII.
* `For` chooses by `utf8`, and the Unicode border styles are distinct.

**Mutation proofs**; none survived:

| Mutation | Killed by |
| :--- | :--- |
| a two-cell Unicode glyph | `Unicode.Focus = "▸▸" is 2 cells, want 1` |
| a non-ASCII rune in the ASCII set | `ASCII.Ellipsis = "…" holds U+2026, which is not printable ASCII` |
| the reflection walk skips the last field | `ASCII: walked 59 glyphs, want 64` |
| an empty glyph | `Unicode.BadgeClose is empty` |

**Checks.**

* `make pre-add-check` reported 2 files clean.
* `make lint` passed, as did `go test -race`, `LC_ALL=C go test` and
  `go mod tidy -diff`.
* The Windows test host passed `go vet` and `go test -race`.
