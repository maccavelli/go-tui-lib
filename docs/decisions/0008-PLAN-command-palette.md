---
status: in-progress
date: 2026-10-25
associated-madr: "0008-MADR-command-palette.md"
---
# Implement the command palette and the fuzzy matcher

Associated MADR: [0008-MADR-command-palette.md](0008-MADR-command-palette.md)

**Revision, 2026-10-02.** The owner answered the MADR's Q1–Q4, each with
the recommendation, and the MADR is `accepted`. Only Step 1 changed. This
PLAN is still `proposed`, because execution is not yet approved.

**Revision, 2026-10-11, for MADR amendments A2 and A3.** Nothing had run.
The steps are rewritten against the tree at `a0537d3`, after 0006 and
0007, and against A3's recommended answers to Q5–Q7:

* A2's names, and A3's, replace the MADR's first ones throughout.
* A new Step 2 makes the `workspace` and `glyph` additions A3 items 1–3
  and 8 need, before any palette code. The old Steps 2–8 are now Steps
  3–9.
* The old Step 5's `DefaultKeyMap` and `palette` context are replaced by
  A3 item 1: `palette.DefaultRules`, the `kind:picker` scope, and commands
  the `Picker` consumes.
* The Rules for every step take 0007-PLAN's current gates.
* Step 2's generic-method probe ran on 2026-10-11 (A3, facts), so the old
  Step 2's probe and fallback are gone.
* If the owner answers Q5, Q6 or Q7 otherwise, Steps 2, 6 and 7 are
  amended before Step 2 starts. *Later on 2026-10-11:* the owner answered
  each with the recommendation, and approved execution ("committed,
  proceed").

**Note, 2026-10-23, for
[0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)
Q7.** Steps 1 to 4 stand. Steps 5 to 9 wait until
[0020-PLAN-charm-free-cores-glyphs-and-safetext.md](0020-PLAN-charm-free-cores-glyphs-and-safetext.md)
is `complete`, and `palette` is then written against `commandtea` and
`keymaptea`, the Bubble Tea forms of `command` and `keymap` that 0020
moves out of the cores. `fuzzy` ships in `v0.13.0` by 0020-PLAN Step 6,
so Step 9's release is the next minor release after `v0.14.0`, and its
migration section names that version. Nothing else here changes until
this PLAN is next amended, before Step 5.

**Note, 2026-10-24, for 0020-MADR A4.** The release that carries Steps 1
to 4 was tagged `v1.13.0`, in place of `v0.13.0`, and kept; there is no
`v0.13.0`. Where this PLAN says `v0.13.0`, read `v1.13.0`. Step 9's
release follows 0020's release 2, whose version 0020-MADR A4.1 decides.

**Revision, 2026-10-25, for MADR amendment A5, accepted the same day with
both recommendations ("committed, proceed with rcommendations").** Steps 1 to 4
stand. Steps 5 to 9 take `v2.0.0`'s names and the release `v2.1.0`, A5.2's
`Search` for A2's `Input`, and A5.3's cut in Step 6; each change is
annotated where it falls. Step 5 is approved.

## Goal

Ship `fuzzy` and `palette`, so that a program finds and runs any command,
file, pane, theme or other item from one overlay, and drives slash and
`@path` completion with the same engine, with keys that the program's one
keymap binds and that run in the order they were pressed.

Done means every item under Verification holds, CI is green on the pushed
tree, ~~`v0.13.0`~~ `v2.1.0` (A5.1) is tagged by the owner, and the
consumer smoke test passes.

## Scope

### Facts re-read before execution (2026-10-11)

MADR A3's table is the full list, read on `a0537d3`. The ones a step rests
on directly:

| Fact | Step |
| :--- | :--- |
| `workspace.run` sends any matched command that is not its own as `RequestMsg`; `deliver` sends an unmatched key to the modal overlay or the focused pane | 2, 6 |
| A non-modal overlay's `KeyContexts` are not on `KeyPath`; its `overlay:<id>` is | 2, 7 |
| `updateOverlay(i)` writes the slot after the pane's `Update`, so an overlay cannot `Pop` itself | 2 |
| No accessor for the workspace's theme or keymap; `WidthMethod()` exists | 2, 6 |
| Bubble Tea runs each command in its own goroutine; measured reordering of 94–169 in 2000 | 2, 6 |
| Generic methods compile and lint on Go 1.27.2 | 3 |
| `command.Params`, `Registry.Dispatch` (from `Update`), `Available(ctx, SurfacePalette)`, `OriginPalette` | 7, 8 |
| `keymap.Keymap.Shortcut`, `Binding.When`, `keymap.Default` | 6 |
| `glyph.Set` has no prompt or mask glyph | 2 |
| `make fuzz` finds new fuzz targets itself | 3, 4, 8 |

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0008-*`, `docs/decisions/0007-MADR-keymap-engine.md`, `docs/README.md`, `docs/glossary.md` | accept A3; this PLAN `in-progress`; 0007's pointer; the glossary |
| 2 | `workspace/bindings.go`, `workspace/overlay.go`, `workspace/workspace.go`, `glyph/` and tests | `CommandConsumer`, `CommandMsg`, `Close`, `Theme()`, `Bindings()`; `glyph.Set.Prompt` and `Mask` |
| 3 | `fuzzy/` (matcher) and `internal/conformance/enums_test.go` | graphemes, folding, grades, scoring, `Rank`, `RankContext`, `Cells` |
| 4 | `fuzzy/` (index) | trigram index, recall fixture |
| 5 | `palette/` (engine) | providers, availability, generations, streaming, de-duplication |
| 6 | `palette/` (pane and keys) | `Picker`, overlay, rendering, `DefaultRules`, consumed commands, mouse, goldens |
| 7 | `palette/` (providers and completion) | `FromRegistry`, `Recents`, `Static`, `Panes`, `Files`, `WithoutInput` |
| 8 | `palette/` (arguments and recents) | argument stage from `command.Params`, `Frecency` |
| 9 | `README.md`, `docs/`, `docs/guides/command-palette.md` | documentation, release `v0.13.0`, close-out |

No module is added. `fuzzy` uses the standard library and `x/ansi`.
`palette` uses `bubbletea`, `bubbles/textinput`, `lipgloss`, `x/ansi`,
and this repository's `workspace`, `command`, `when`, `keymap`, `theme`,
`glyph`, `fuzzy` and `internal/sanitize`, `internal/enum`.

### Out of scope

* A `.gitignore` reader (MADR Q4). `FSWalker` takes a `PathFilter`.
* A Unicode normaliser. `Normalizer` is a hook only.
* A top-centred overlay anchor (MADR §7). It needs a workspace record.
* Parameters outside A3 item 5's types, which fall through to
  `SelectedMsg`.
* The reordering of a program's own key commands through `RequestMsg`
  (A3 item 1, "Found, not decided here"). It is 0007's routing, for a
  record of its own.
* A key help footer for the picker. `Picker` is not a `KeyMapper`.
* Any change in pi-go.
* `git push` and tags, which are the owner's.

## Prerequisites

* [0006-PLAN-command-registry.md](0006-PLAN-command-registry.md) and
  [0007-PLAN-keymap-engine.md](0007-PLAN-keymap-engine.md) are complete
  (`docs/README.md`), and `v0.12.0` is tagged. Met on 2026-10-11.
* `a0537d3`, 0007's close-out, is pushed, so `make apicheck` compares with
  `v0.12.0` and `scripts/apicheck.allow` is empty. The agent checks both
  before Step 2.
* The owner has answered Q5–Q7 of MADR A3.

## Rules for every step

1. **Order.** Each step's packages compile, pass their tests and pass the
   gates before the next step starts.
2. **Mutation proofs.** Each step names mutations of its key invariants.
   Each is applied to a scratch copy, never the tree, and must make a named
   test fail. A mutation that survives, or does not compile, is replaced
   and recorded. The record quotes each failure line.
3. **Checks per step,** with output to a scratch file and the exit status
   captured before any filter:
   * `make pre-add-check FILES=…` on the step's Go files;
   * `make lint`;
   * the cross `go vet` for `freebsd/amd64`, `openbsd/amd64` and
     `linux/386`, with `CGO_ENABLED=0`;
   * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`,
     `LC_ALL=C go test ./...`, and the tests in workspace mode;
   * `go mod tidy -diff`, `make vuln`, `make apicheck`, `make examples`
     and `make release-check`;
   * CI's other checks: `shellcheck`, `markdownlint-cli2`, `actionlint`,
     and the Python scripts' tests (`scripts/*_test.py`);
   * the Windows test host, on a scratch copy with its index synced to the
     tree; and, where the step's tests are slow or racy, the idle Linux
     host the owner named, under `-race`.
   * `make apicheck` must report no incompatible change: every change in
     this PLAN is an addition.

   *2026-10-15, for
   [0018-MADR-python-verification-tools.md](0018-MADR-python-verification-tools.md)
   A2:* these checks are met by `python3 -m tools.verify gates`'s default
   list, which runs each check once. `release-check` covers
   `pre-add-check`, `make lint`'s golangci-lint, `go mod tidy -diff`,
   workspace mode, the modules check, `make apicheck` and `make examples`;
   `modernize` covers the rest of `make lint`. The gates that repeat them
   are run by name, with `--only`, when a step needs one alone.
4. **Conventions.** Every package follows AGENTS.md's TUI and API
   conventions. In particular: nothing writes to `os.Stdout` or
   `os.Stderr`, sets the alternate screen, installs a signal handler,
   reads the environment, exits or starts a process; every glyph comes
   from `glyph`; widths are cells in the workspace's method; options are
   opaque; enums go through `internal/enum`; each package ends its
   documentation with its stability line.
5. **Identifiers.** The identifier scan runs before each handoff, over the
   step's diff: no hostname, account name, org-internal path or
   real-machine absolute path.
6. **Commit.** One commit per step, made by the owner. At the end of each
   step the agent stops with the gates passed and the step's evidence in
   the execution record, and stages and commits nothing. The owner commits
   with `git commit --no-edit` and pushes. (2026-10-02: the org rules
   forbid agent commits to `main`, and the owner chose this over a
   `feature/` branch.)
7. **Deviations.** A finding outside a step stops the work: the agent
   offers resolutions, and records the owner's choice as a dated entry
   here, and in the MADR when a decision or fact changes, before going on.

## Implementation Steps

### Step 1: records

* The owner's answers to Q5–Q7 go into MADR A3, which becomes `accepted`;
  a departure from a recommendation amends A3 and this PLAN first.
* This PLAN becomes `in-progress`; `docs/README.md`'s rows follow.
* [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) gains A12, a
  pointer only: A11 item 1 is extended by 0008-MADR A3 item 1.
* `docs/glossary.md`:
  * "Names the planned records must not take" records A3's renames
    (`Commands` provider → `FromRegistry`, `AsCompletion` →
    `WithoutInput`) and that `RegistryOption` is `command`'s;
  * `Grade`, `Hit`, `Picker`, `Availability`, `Frecency`, `Input` and
    `CommandMsg` are entered when their steps build them, in those steps.
* Records only: no source, test, CI or configuration change in this
  commit.

### Step 2: the workspace and glyph additions

* **`workspace`** (A3 items 1–3):
  * `CommandConsumer` and `CommandMsg`;
  * `run`: a matched command that is not the workspace's own is offered
    to the modal overlay alone, or else to the open overlays top first,
    then the focused pane; the first consumer is sent `CommandMsg` through
    `updateOverlay` or `Send`; with none, `RequestMsg` as today. The
    `EscConsumer` case is unchanged;
  * `Close(id string) tea.Msg`, handled in `Update` like `Pop` for that
    overlay, and a no-op for an ID not open;
  * `(*Workspace).Theme() theme.Theme` and
    `(*Workspace).Bindings() *keymap.Keymap`.
* **`glyph`** (A3 item 8): `Set.Prompt` and `Set.Mask`, in `Unicode()`,
  `ASCII()` and every tier, each one cell under `ansi.WcWidth` and
  `ansi.GraphemeWidth`.
* **Tests:**
  * a modal overlay that consumes `x.y` gets `CommandMsg` for a key bound
    to `x.y`, with `OriginKey` and the workspace's `WhenContext`, and no
    `RequestMsg` is returned;
  * with no modal overlay: a non-modal overlay that consumes beats the
    focused pane; the top overlay beats a lower one; a focused pane that
    consumes gets it; none consuming gives `RequestMsg`;
  * a modal overlay that does not consume: `RequestMsg`, and the focused
    pane underneath is not offered;
  * **order:** the consumer holds each `CommandMsg` when `Update`
    returns, before any command it returned runs; ten such keys reach it
    in the order they were sent;
  * the workspace's own commands are still run first, and `esc` still
    reaches an `EscConsumer`;
  * `Close` closes the named overlay and sends `Pop`'s focus messages;
    `Close` of an ID not open changes nothing; `Close` returned from an
    overlay's own `Update` leaves the other overlays intact;
  * `Theme()` follows `SetTheme` and a `tea.BackgroundColorMsg`;
    `Bindings()` follows `SetBindings`, and is `DefaultRules`' keymap by
    default;
  * every existing workspace golden and key test passes unchanged;
  * `glyph`: `Prompt` and `Mask` are one cell in both methods, non-empty,
    and ASCII in `ASCII()`; the glyph table test covers them.
* **Mutations:**
  * the offer skips overlays and goes to the focused pane only;
  * the modal overlay's refusal falls through to the pane beneath;
  * `CommandMsg` is sent as a command, not delivered at once (the order
    test);
  * `Close` closes the top overlay whatever the ID;
  * `Mask` is two cells in `Unicode()`.

### Step 3: `fuzzy`, the matcher

* MADR §2 with A2 and A3: `Matcher`, `NewMatcher` and `Option` (`WithCase`,
  `WithNormalizer`, `WithDelimiters`, `WithExtended`, `WithMaxAlign`),
  `CaseMode`, `Normalizer` and `NormalizerFunc`, `Score`, `Grade`, `Match`,
  `Range`, `Rank[T]`, `RankContext[T]`, `Ranked[T]`, `CellRange`, `Cells`.
  The query is matched on its first 256 graphemes (A3 item 10).
* `Grade` and `CaseMode` through `internal/enum`, and in
  `internal/conformance/enums_test.go`'s table.
* **Tests:**
  * a table pins grade and order for command titles, paths, camelCase,
    digits, Cyrillic, CJK, combining marks and ZWJ emoji;
  * `Smart` case: a query with an upper-case letter is case-sensitive;
  * the ASCII and the grapheme paths agree on every ASCII case;
  * properties over random strings: ranges lie on grapheme boundaries,
    inside the candidate, in order, and spell the query under folding;
    `Score` and `Match` agree on grade and value; ranking is total and
    stable;
  * the `boost` never lifts a hit above a better grade, or above one
    consecutive-match bonus;
  * extended syntax: AND terms, `'exact`, `^prefix`, `suffix$`, `!not`;
    without `WithExtended`, a space is a character;
  * `WithNormalizer` folds through the hook, and `NormalizerFunc` adapts a
    function;
  * a 10 000-grapheme query matches as its first 256;
  * `RankContext` returns `ctx.Err()` promptly once cancelled;
  * `Cells` gives the same ranges as `ansi.Cut` under both width methods;
    *2026-10-12 (D1, MADR A4):* as `ansi.Cut` under `GraphemeWidth`, and
    ~~as `m.TruncateLeft(m.Truncate(s, end, ""), start, "")`~~ under
    `WcWidth`, whose `Cut` ignores its left edge upstream, as
    `internal/sanitize`'s `Truncate` cuts: `Truncate(s, b)` less its prefix
    `Truncate(s, a)` (D1, second answer);
  * `FuzzMatch` finds no panic and no out-of-range range, and `make fuzz`
    picks it up.
* **Benchmarks** (`b.Loop`): `Score` on ASCII reports 0 allocations;
  ranking 10 000 titles is measured and recorded.
* **Mutations:**
  * grades are ignored in the order;
  * a range ends inside a grapheme;
  * the boundary bonus is dropped;
  * ties are broken by map order;
  * `Smart` case is always insensitive;
  * the 256-grapheme cap is removed (the long-query test's time bound).

### Step 4: `fuzzy`, the index

* MADR §3 with A2: `Index`, `NewIndex`, `IndexOption` (`WithMinOverlap`,
  `WithMaxCandidates`), `Update`, `Len`, `At`, `Candidates`.
* **Tests:**
  * an exact or prefix match is never dropped by the index;
  * one-, two- and four-grapheme queries follow toad's three strategies;
  * `Update` leaves the old index unchanged, and both are read from many
    goroutines under `-race`;
  * a fixture corpus of 20 000 synthetic paths records the index's recall
    against a full scan, for the execution record;
  * `FuzzIndex` finds no panic.
* **Benchmarks:** building the index and querying 100 000 paths, recorded.
* **Mutations:**
  * the overlap threshold is ignored;
  * padding is dropped, so prefixes lose their trigrams;
  * `Update` changes the receiver.

### Step 5: `palette`, the engine

* MADR §4 with A2 and A3: `Provider` and its optional `Discoverer`,
  `Starter`, `Stopper`, `Prefixed` and `Debouncer`; ~~`Input`~~ `Search`
  (A5.2) (`Text`, `Matcher`, `WhenContext`); `Hit` (`ID`, `Text`, `Ranges`, `Score`,
  `Group`, `Help`, `Hint`, `Command`, `Danger`, `Disabled`, `Params`,
  `Run func(jsontext.Value) tea.Cmd`); `Availability`; `Grouping` through
  `internal/enum` and the conformance table; the generation counter, the
  drain command, the top-k merge under `WithLimit` (default 200),
  de-duplication by `ID`, contained failures, and `WithLogger`.
* The engine is internal to the `Picker`, which Step 6 makes; this step
  tests it through an unexported seam and fake providers.
* **Tests, with fake providers inside `testing/synctest`:**
  * a slow provider's hits for an old query never appear after a new
    keystroke;
  * hits arrive in batches of at most 64 or one frame;
  * a provider that errors or panics shows one muted row naming it; its
    error reaches the `slog.Handler` and not the screen; the others
    finish;
  * **leaks:** after a search is cancelled, and after the picker closes,
    `synctest.Test` returns, so no goroutine of the bubble is left
    blocked; the `goroutineleak` profile, taken at the bubble's end,
    reports none;
  * availability follows `when` expressions against the context at open,
    a prefix restricts the query to its provider, and `?` lists the
    prefixes;
  * debounce delays a provider's search, and the next keystroke cancels
    it;
  * two hits with one `ID` keep the better score;
  * the top-k never holds more than the limit.
* **Mutations:**
  * the generation check is removed;
  * the context is not cancelled on a new keystroke;
  * a provider panic is not recovered;
  * de-duplication keeps the first hit;
  * the drain loop ignores the 64-hit bound.

### Step 6: `palette`, the pane and its keys

*2026-10-12 (D1, MADR A4 item 2):* `lipgloss.StyleRanges` cuts in grapheme
width whatever the workspace measures with. Where the palette cuts a
highlighted row is decided by an amendment of MADR A4 item 2 before this
step starts, and this step is amended to match. *2026-10-25, MADR A5.3,
accepted:* the palette cuts it itself, piece by piece, with
`internal/sanitize`'s `Truncate` in `w.WidthMethod()`, and renders each
piece with its style; `lipgloss.StyleRanges` is not used. The step's
tests gain: each highlight covers the cells `fuzzy.Cells` gives, under
`WcWidth` and `GraphemeWidth`, on a row holding a ZWJ family before a
match; and its mutations: the cut in grapheme width under `WcWidth`.

* MADR §7 with A3 items 1–4, 9, 11 and 12:
  * `NewPicker(w, opts...)`, `Register`, `Open(...OverlayOption)`
    (`WithID`, `WithAnchor`, `WithSize`), capturing `w.KeyPath()` and
    `w.WhenContext()` before it pushes;
  * the `Picker` as a `workspace.Pane`, `KeyContexter` (`kind:picker`),
    `CommandConsumer`, `EscConsumer` and `Cursorer`;
  * the `textinput.Model` held directly, with `Prompt` and the placeholder
    from `glyph` and the options;
  * the results window, help line and count, highlights with `fuzzy.Cells`
    and ~~`lipgloss.StyleRanges`~~ the repository's cutter (A5.3) in
    `w.WidthMethod()`, the selected-row
    marker, danger badges, `Disabled` rows, truncation with `Ellipsis`
    through `internal/sanitize`'s `Truncate`;
  * `sanitize.Line` on every hit's text fields and provider names;
  * hints from `w.Bindings().Shortcut` for drawn rows with a `Command` and
    no `Hint`;
  * `DefaultRules()`, the navigation commands consumed, and
    `OpenCommand`;
  * accept: `Hit.Run(args)` on the loop, the recents record, and
    `workspace.Close`; a disabled hit shows its reason and does not run;
  * the mouse: the wheel scrolls, a click runs a row.
* **Tests, through a real `workspace.Workspace` with
  `workspace.DefaultRules()` and `palette.DefaultRules()`:**
  * `ctrl+p` comes back as `RequestMsg` for `palette.open`; dispatching
    `OpenCommand`'s command opens the picker; with a non-picker modal
    overlay open, `ctrl+p` is not matched (`!workspace.modal`);
  * inside the picker, `ctrl+p` and `ctrl+n` move, and reach no
    `RequestMsg`; `up`, `down`, `pgup`, `pgdown`, `tab` and `enter` do
    what MADR §7 says; printable keys reach the input;
  * **order:** "down, down, enter" in one burst runs the third row;
  * `esc` clears a non-empty query, then closes;
  * a user binding in `kind:picker` rebinds a move, and the default key
    no longer moves;
  * without `palette.DefaultRules()`, `down` reaches the input, and the
    selection stays;
  * the theme follows a `tea.BackgroundColorMsg` without reopening;
  * a hit text with an ESC sequence and a newline draws as one row of its
    visible text;
  * the hint is the key from the path at open, and follows `SetBindings`;
  * only visible rows are matched for ranges;
  * the cursor is the input's, offset into the overlay;
  * the wheel scrolls, and a click runs the row under it.
* **Golden frames** across the matrix {colour, no colour} × {UTF-8,
  ASCII} at 60 and 100 columns, measured with `tuitest.Fits`: an empty
  query with recents, a query with highlights, a truncated highlight, a
  danger badge, a disabled row, a provider error row, "still searching",
  and a hint.
* **Mutations:**
  * the selected row loses its glyph marker;
  * highlights use rune indexes;
  * `esc` closes with a non-empty query;
  * truncation counts bytes;
  * `ConsumesCommand` returns false (the in-picker `ctrl+p` test);
  * `sanitize.Line` is skipped (the ESC-sequence test);
  * the hint uses the overlay's path, not the one at open.

### Step 7: built-in providers and completion mode

* MADR §5 with A3 items 4–6:
  * `FromRegistry(reg)`: `Available(ctx, command.SurfacePalette)`; matches
    the title, `Category: Title`, the slash name and the aliases, keeping
    the best; sets `Command`, `Danger` and `Params`; its `Run` calls
    ~~`Dispatch`~~ `commandtea.Dispatch(ctx, reg, req)` (A5.1) with
    `Origin: command.OriginPalette`;
  * `Recents`, `Static` with `Item`, `Panes(w)`;
  * `Files(walk, opts...)` with `FileOption` (`WithIndexAbove`,
    `WithMaxFiles`), `Walker`, `WalkerFunc`, `FSWalker`, `PathFilter`,
    `PathFilterFunc`;
  * `WithoutInput()`, `SetQuery`, `SelectedMsg`, and the
    `overlay:completion` bindings.
* **Tests:**
  * `FromRegistry` hides commands whose `when` fails at open, and commands
    not offered on `SurfacePalette`; matches aliases and slash names;
    carries `Danger` and `Params`;
  * running a command hit sends a `ResultMsg` whose request has
    `OriginPalette`; a `Prompt` command's `PromptMsg` reaches the program;
    a `Destructive` command goes through the registry's gate;
  * `Files` over an `fstest.MapFS` uses the index above its threshold and
    a full scan below it, honours `PathFilter`, ranks by crush's tiers,
    and stops at `WithMaxFiles` with its row;
  * `Panes` focuses the chosen pane and shows a hidden one;
  * completion mode is non-modal: other keys reach the focused editor
    pane; `down` and `tab` in `overlay:completion` move and accept;
    accepting sends `SelectedMsg` and closes the overlay; the editor's
    own `enter` binding is shadowed only while the pop-up is open.
* **Mutations:**
  * `FromRegistry` ignores `when`;
  * `FromRegistry` dispatches with `OriginKey`;
  * `FSWalker` ignores `PathFilter`;
  * completion mode is modal;
  * `WithMaxFiles` is ignored.

### Step 8: arguments and recents

* MADR §6 with A3 items 5, 7 and 10: the argument stage from
  `command.Params`; `Frecency`, `NewFrecency`, `FrecencyOption`
  (`WithHalfLife`, `WithMaxEntries`), `Record`, `Score`, `WriteTo`,
  `ReadFrom`, and `ErrRecentsVersion`.
* **Tests:**
  * a command with string, integer, boolean and enum parameters is
    prompted in order; `Required` and `Default` hold; a bad integer, and
    one outside `Min` and `Max`, are refused with the reason in the help
    line; it runs with the collected JSON object;
  * a `Hidden` parameter is skipped; a `Secret` one is drawn with the
    `Mask` glyph, and its value appears in no golden, log line or
    recents entry;
  * an `array` or `object` parameter, and a `Params` error, fall through to
    `SelectedMsg` with no arguments, and nothing runs;
  * frecency decays with the half-life under `synctest`'s clock;
    `WithMaxEntries` evicts the lowest; recents boost but do not overturn
    a better grade;
  * a JSON round trip is exact; an unknown member, an unknown version
    (`errors.Is(err, ErrRecentsVersion)`), input over 1 MiB and an ID over
    1024 bytes are refused;
  * `FuzzFrecencyRead` finds no panic, and a successful read always
    writes back what it read.
* **Mutations:**
  * the half-life is ignored;
  * `Required` is not enforced;
  * an unknown version is accepted;
  * the 1 MiB limit is removed;
  * a secret value is drawn unmasked.

### Step 9: documentation, the release and the close-out

* **`docs/guides/command-palette.md`:** opening the palette and binding
  its keys (`palette.DefaultRules`, `OpenCommand`, one keymap); writing a
  provider and its availability; prefixes; completion pop-ups and their
  ID; persisting recents; the file provider and its `fs.FS`; what
  `CommandConsumer` lets any pane do.
* **Examples,** compiled and run: `ExamplePicker` and
  `ExampleMatcher_Rank`.
* **Docs tree:** `docs/architecture.md` gains both packages and their
  imports; `docs/guides/building-workspaces.md` gains `CommandConsumer`,
  `Close`, `Theme` and `Bindings`; `docs/glossary.md` its final entries;
  `docs/README.md` rows; README Status.
* **The release, ~~`v0.13.0`~~ `v2.1.0` (A5.1),** as 0007-PLAN's Steps 8
  and 12 ran, and 0020-PLAN's Step 13 for the `/v2` path:
  * `docs/guides/migrating.md` gains "Migrating to `v2.1.0`": additions
    only;
  * release notes in the execution record;
  * the disclosure guard over the outgoing commits, then the owner's push;
  * CI green on `main`, then the owner's annotated tag, and CI green on
    the tag;
  * the consumer smoke test, in a scratch module outside the repository
    with `GOWORK` empty: `go get github.com/maccavelli/go-tui-lib/v2@v2.1.0`
    from the proxy, build a program
    that opens the palette over a registry and runs a command through it,
    under staticcheck, with a negative control seen to fail.
* This PLAN becomes `complete`, and its `docs/README.md` row says so.

## Verification

* Every step's mutations are killed, each failure quoted in the record.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint`, `make vuln`, `make apicheck`,
    `make examples` and `make release-check`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`,
    `LC_ALL=C go test ./...`, and the tests in workspace mode;
  * the cross `go vet` for `freebsd/amd64`, `openbsd/amd64` and
    `linux/386`;
  * `make fuzz`.
* `go mod tidy -diff` is clean, and `go.mod` gains no requirement.
* `make apicheck` reports no incompatible change against ~~`v0.12.0`~~
  `v2.0.0` (A5.1), and `scripts/apicheck.allow` stays empty.
* `internal/conformance` passes for `fuzzy` and `palette`: enums, names,
  options, stability lines, and no output, environment, exit or process.
* No default binding in `palette` is `ctrl+c`.
* The benchmarks and the index's recall are in the execution record.
* The identifier scan finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–9 and tags ~~`v0.13.0`~~ `v2.1.0`
  (A5.1). pi-go adopts
  the palette and the completion mode under its own records.
* **Rollback.** Before the push, each step is one local commit. After it,
  a patch release fixes forward. Every change is an addition, so a program
  that does not use `fuzzy` or `palette` sees no change, except that a
  matched key is offered to a pane or overlay implementing
  `CommandConsumer`, which none did before `v0.13.0`.

## Execution Record

### Step 1: records

*2026-10-11.* The owner committed the revised records as `d297371`,
answered Q5–Q7 ("Q5 follow recommendation. Q6 follow recommendation. Q7
follow recommendation.") and approved execution ("committed, proceed").

* **MADR A3** is `accepted`, with the answers under its owner questions;
  the MADR's date is 2026-10-11.
* **This PLAN** is `in-progress`.
* **[0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md)** gains A12, a
  pointer to A3 item 1, which extends its A11 item 1.
* **`docs/glossary.md`:** "Names the planned records must not take"
  records A3's two renames and `command`'s `RegistryOption`. The types
  Steps 2 to 8 build are entered by those steps.
* **`docs/README.md`:** 0007-MADR's row reads A4–A12 accepted; 0008-MADR's
  A3 accepted; this PLAN's row `in-progress`.
* **Records only:** no source, test, CI or configuration file changes in
  this step.
* **Prerequisites for Step 2,** checked: `scripts/apicheck.allow` is its
  header alone. **Not met:** `a0537d3`, 0007's close-out, is not on
  `origin/main` (`04182bf` after a fetch), so until it is pushed `make
  apicheck` compares with `v0.11.0`. Step 2 waits for the owner's push of
  `a0537d3`, `d297371` and this step.

*Met, 2026-10-11:* the owner pushed; `origin/main` is `d9930f8`, and `make
apicheck` reports "against v0.12.0, 0 incompatible change(s)", clean.

### Step 2: the workspace and glyph additions

*2026-10-11.* The owner approved ("committed, pushed, proceed"), and asked
during the step for the gates on the idle Linux host as well.

* **`workspace`:**
  * `CommandConsumer` and `CommandMsg` (`workspace/bindings.go`);
  * `run` offers a matched command that is not the workspace's own
    before making a `RequestMsg`, through `offer`: the top modal overlay
    alone; else the open overlays top first (`slices.Backward`), then the
    focused pane; delivered at once through `updateOverlay` or `Send`;
  * `Close(id) tea.Msg`, handled in `Update` by `closeOverlay`, a no-op for
    an ID not open (`workspace/overlay.go`, `workspace/workspace.go`);
  * `(*Workspace).Theme()` and `(*Workspace).Bindings()`.
* **`glyph`:** `Set.Prompt` (`❯`, `>`) and `Set.Mask` (`•`, `*`).
* **`docs/glossary.md`:** `CommandConsumer` and `CommandMsg` under "One
  meaning each".
* **Tests:**
  * `workspace/consumer_test.go`: `TestModalConsumer`, `TestConsumerOrder`,
    `TestModalConsumerRefuses`, `TestConsumedInKeyOrder`,
    `TestOwnCommandsFirst`, `TestClose`, `TestThemeAndBindings`;
  * `glyph/glyph_test.go`: the walks count 70 glyphs (was 68);
    `TestOneCellInBothMethods` measures every glyph under `ansi.WcWidth`
    and `ansi.GraphemeWidth`; `TestPromptAndMask`;
  * every existing workspace golden and key test passes unchanged.
* **Mutations,** each on a scratch copy, each killed:

  | Mutation | Killed by |
  | :--- | :--- |
  | the offer skips the overlays (`slices.Backward(w.overlays[:0])`) | `TestConsumerOrder`: "top 0, low 0, pane 1; want the top overlay alone" |
  | a modal overlay's refusal falls through to the panes beneath | `TestModalConsumerRefuses`: "the pane beneath a modal overlay was given [...]" and "got []; want one RequestMsg" |
  | `CommandMsg` is returned as a command, not delivered at once | `TestConsumedInKeyOrder`: "after key 0 the consumer holds []; want [x.a] in order" |
  | `Close` closes the top overlay whatever the ID | `TestClose`: "Close of an ID not open: [a b], cmd false" |
  | `Mask` is two cells in `Unicode()` | `TestEveryGlyphIsOneCell`: `Unicode.Mask = "••" is 2 cells, want 1`, and the two other width tests |

* **Found and fixed during the step,** in the step's own files:
  * `gofmt` realigned the two new fields in `Unicode()` and `ASCII()`.
  * `make modernize` suggested `slices.Backward` for the overlay loop in
    `offer`; applied, and the first mutation's anchor moved with it.
  * golangci-lint 2.14.0 failed on all three `GOOS` with a panic in
    go-critic v0.15.0's `evalOrder` checker
    (`evalOrder_checker.go:83`, `hasPtrRecv`), on the test fake's
    `return c, c.on(m)`: it takes the function field `on` for a method
    and reads a receiver it has not got. The fake calls `c.on(m)` before
    the return, with a comment saying why. The library's code was not the
    cause.
  * The first Linux run's copy had no `.git`, so `make pre-add-check`,
    `lint`, `tidy-check` and `vuln`, which find the modules with git,
    failed there ("Stopping at filesystem boundary"). The script now makes
    an index of the copy first, as the Windows script does.
* **Gates, macOS** (load average near 55 from another job's mutation run):
  gofmt clean; `make pre-add-check FILES=…` "6 file(s) clean"; `make lint`
  0; the cross `go vet` 0; `-race` 0; `-shuffle=on -count=2` 0; `LC_ALL=C`
  0; workspace mode 0; `go mod tidy -diff` 0; `make vuln` "No
  vulnerabilities found"; `scripts/go-modules.py --check` 0; `make
  apicheck` "against v0.12.0, 0 incompatible change(s)", clean; `make
  examples` clean; `make release-check` "226 file(s) clean". `shellcheck`,
  `actionlint`, `markdownlint-cli2` and the five `scripts/*_test.py` exit
  0.
* **Gates, the idle Linux host** (Go 1.27.2 linux/amd64, golangci-lint
  2.14.0): gofmt clean; `make pre-add-check FILES=…` "6 file(s) clean";
  `make lint` 0; the cross `go vet` 0; `-race` 0, 21 packages ok,
  `internal/conformance` in 112 s; `-shuffle=on -count=2` 0; `LC_ALL=C` 0;
  workspace mode 0; `make tidy-check` 0; `make vuln` "No vulnerabilities
  found". The API diff, the examples and `release-check` need the tags,
  and ran on macOS and Windows.
* **Gates, the Windows test host** (Go 1.27.2 windows/amd64, a copy of the
  tree with its history from a bundle, HEAD `d9930f8`, the index synced):
  `make pre-add-check` "227 file(s) clean" (apicheck and examples
  included; the count has the new test file, which the index tracks);
  `make lint` "0 issues."; `make vuln` "No vulnerabilities found"; `make
  examples` clean; the step's 14 tests pass; `workspace` under
  `-shuffle=on -count=2` 0; `go-fuzz_test.py` 27 passed; `make fuzz
  FUZZTIME=3s` "9 packages ran clean".
  * A first run on the host, and one on the Linux host, were stopped and
    restarted once the lint and modernize findings above were fixed, since
    their copies predated the fix.
* **Identifier scan** of the step's diff: nothing.

### Step 3: `fuzzy`, the matcher

*In progress, 2026-10-12.* The package and its tests are written, and the
fuzzy and conformance tests pass, under `-race` too. D1 (above, in the
step, and MADR A4) changed `TestCells`' oracle twice, each time by the
owner's answer.

*Paused, 2026-10-12.* The owner ruled out the bash scripts this PLAN's
gates had run through, and approved
[0018-PLAN-python-verification-tools.md](0018-PLAN-python-verification-tools.md).
This step's gates, and its mutation proofs, which were killed before the
lint fixes changed the code, run again through `tools/verify`: locally
after 0018's Step 2, and on the two other hosts after its Step 4. The
step's record is written then.

*macOS, 2026-10-12:* the owner committed the step's code as `79de568`
before its gates ran. `python3 -m tools.verify gates --label p8s3` over
its 14 files: "gates: 25 passed, 0 failed, 0 skipped" (0018-PLAN Step 2's
acceptance). The mutation proofs, and the Linux and Windows gates, are
still to run.

*Mutations, 2026-10-12,* through `python3 -m tools.verify mutate` on the
committed code, each on a copy: "mutate: 9 killed, 0 survived, 0 errors".
The range-end anchor moved with the lint fixes (`End: us[pos[j]].end`).

| Mutation | Killed by |
| :--- | :--- |
| grades are ignored in the order | `TestOrder/paths` and `/ZWJ_emoji`: `"file" ranks ["file" "file_test.go" … "src/file.go" …]` |
| a range ends inside a grapheme | `TestProperties`: `in "п" the range {0 1} is out of place` |
| the boundary bonus is dropped | `TestOrder/camelCase` and `/digits`: `"fb" ranks ["fooBar" "FooBar" "fab" "f_b" "foobar"]` |
| ties are broken against input order | `TestRankStableAndTotal`: "the order is [6 3 1 4 2 0]; want [1 3 6 4 0 2]" |
| Smart case is always insensitive | `TestSmartCase`: `"Fb" in "fooBar": matched true; want false` |
| the 256-cluster cap is removed | `TestLongQuery`: "the query keeps 10000 clusters; want 256" |
| the boost is unbounded | `TestBoostBounded`: `a boost of 1000 over a lead of 9: "xa__xb" first` |
| `Cells` counts bytes | `TestCells`: `cells {13 32} cut … the range {28 47} is …` |
| `fuzzy.Grade` is missing from the conformance enum table | `TestEveryEnumRoundTrips`: "fuzzy.Grade is an enum the table lacks" |

The Linux and Windows gates wait for 0018-PLAN Step 4.

*Linux and Windows, 2026-10-12,* through `python3 -m tools.verify remote`
over the same 14 files, nothing skipped: on each host "gates: 25 passed,
0 failed, 0 skipped", with `pre-add-check` "14 file(s) clean", `-race`,
the cross `go vet`, `LC_ALL=C`, workspace mode, `apicheck` "against
v0.12.0, 0 incompatible change(s)", `examples` clean and `release-check`
"240 file(s) clean". The first Linux run failed 7 gates on a stray `.git`
in that host's temporary directory, which the owner had removed
(0018-PLAN Step 4, D1).

*Done, 2026-10-12.* Step 3's code is `79de568`; its gates have passed on
macOS, the Linux host and the Windows host, and its nine mutations are
killed. `FuzzMatch` ran clean for 30 s (868 271 executions) before the
lint fixes, and is not run again here: `make fuzz` is not one of the
gates, and CI's fuzz step runs it on the push. The benchmarks are recorded
at Step 9, with the index's.

### Step 4: `fuzzy`, the index

*2026-10-12.* The owner approved ("committed, proceed") after
`8c34cff`.

* **`fuzzy/index.go`:** `Index`, `NewIndex`, `IndexOption`
  (`WithMinOverlap`, default 0.3; `WithMaxCandidates`, default 2000),
  `Update`, `Len`, `At`, `Candidates`, and the constants
  `DefaultMinOverlap` and `DefaultMaxCandidates`. The matcher's terms keep
  their text, which `Candidates` reads (`fuzzy/matcher.go`).
  * **One cluster:** candidates where it starts the text or follows a
    separator (`/\ ._-:,;|`), those that start with it first.
  * **Two or three:** candidates holding every rune, by a 64-bit mask of
    folded ASCII bytes, and by the folded text for a rune beyond ASCII;
    those holding the query in one piece first.
  * **Four or more:** candidates sharing at least the minimum share of
    the query's trigrams, padded with spaces, a separator read as a space,
    most shared first. A candidate equal to the query, or whose path
    element or word starts with it, holds every one of them, so it comes
    before candidates that only contain the query.
  * **An extended query** is narrowed by its longest term that is not
    negated. Folding is the matcher's simple case folding; a `Normalizer`
    is not applied, and the type's documentation says so.
  * **`Update`** returns a new index of the kept candidates and the added
    ones, with the same options, built afresh; the old one is untouched.
  * Posting IDs are `int32` from a counter, and trigram keys `int64`, so
    no conversion narrows; an index of more than `MaxInt32` candidates
    panics.
* **Tests** (`fuzzy/index_test.go`): `TestIndexStrategies` (each strategy,
    case folding, Cyrillic, the 30% and 20% thresholds),
    `TestIndexKeepsExactAndPrefix` (three exact or prefix matches survive a
    limit of 10 filled by fifty containing ones; and 500 prefixes of a
    3000-path corpus each yield their path), `TestIndexUpdate` (the old
    index unchanged; both read by eight goroutines under `-race`),
    `TestIndexRecall`. `FuzzIndex` (`fuzzy/fuzz_test.go`): no index out of
    range or yielded twice, never more than the limit, and `Update` leaves
    the old index alone; 30 s clean, 914 395 executions.
* **Recall** on 20 000 synthetic paths, 100 queries of each kind, against
  a full scan, at the default options:

  | Queries | All matches kept | Ten best kept |
  | :--- | :--- | :--- |
  | prefixes of a last element, 4 to 6 bytes | 104 962 of 135 342 (77.6%) | 998 of 998 (100%) |
  | whole words | 178 128 of 191 184 (93.2%) | 1000 of 1000 (100%) |
  | sparse subsequences of a last element, 2 or 3 bytes | 113 069 of 355 735 (31.8%) | 564 of 1000 (56.4%) |
  | sparse subsequences of a path, 4 to 6 bytes | 5 881 of 44 699 (13.2%) | 190 of 932 (20.4%) |

  No exact or prefix match was lost. The two- and three-cluster rows lose
  to the limit, not the filter: about 3 500 candidates hold the runes and
  2000 are kept. Putting candidates that hold the query in one piece
  first raised their ten-best share from 54.0% to 56.4%; the rest of a
  sparse query's best matches are scattered, and only scoring tells them
  apart. The four-to-six row is the trigram filter's trade, which MADR §3
  names. Step 7's file provider uses the index only above 5000
  candidates, and its options are tuned there.
* **Benchmarks:** they run (`-benchtime 1x`, all five); their figures are
  taken at Step 9 on a quiet host. This one's load average was 51.
* **Mutations,** through `python3 -m tools.verify mutate`, each killed:

  | Mutation | Killed by |
  | :--- | :--- |
  | the overlap threshold is ignored | `TestIndexStrategies`: `file yields [… "f_i_l_e"]` |
  | padding is dropped, so starts and ends lose their trigrams | `TestIndexStrategies`, and `TestIndexKeepsExactAndPrefix`: `"lib/file.go" was dropped under a limit of 10` |
  | `Update` changes the receiver | `TestIndexUpdate`: "Update changed the old index" |
  | the candidate limit is ignored | `TestIndexKeepsExactAndPrefix`: "53 candidates under a limit of 10" |

* **Gates,** through `tools.verify` over the step's five Go files, nothing
  skipped: macOS, the Linux host and the Windows host each "gates: 25
  passed, 0 failed, 0 skipped", with `pre-add-check` "5 file(s) clean",
  `apicheck` "against v0.12.0, 0 incompatible change(s)", and
  `release-check` clean (240 tracked files on macOS; 242 on the hosts,
  whose copies' index tracks the two new files).
* **Docs:** `docs/glossary.md` gains `Index` and `IndexOption` under `fuzzy`.

### Step 5: `palette`, the engine

*2026-10-25.* The owner accepted MADR A5 with both recommendations and
approved this step ("committed, proceed with rcommendations").

* **`palette`** is new, a Bubble Tea package
  (`palette/doc.go`, with its stability line):
  * `palette/provider.go`: `Provider` (`Name`, `Search(ctx, Search, yield)`),
    the optional `Discoverer`, `Starter`, `Stopper`, `Prefixed` and
    `Debouncer`; `Search` (`Text`, `Matcher`, `WhenContext`, A5.2); `Hit`
    with the twelve fields this step lists, `Run func(jsontext.Value)
    tea.Cmd` among them; `Availability` (`When`).
  * `palette/options.go`: `Grouping` (`ByScore`, `ByProvider`; tokens
    `score` and `provider`) through `internal/enum`'s `Names`;
    `DefaultLimit` (200); `Option`, an interface with an unexported method;
    `WithLimit`, `WithGrouping` and `WithLogger(slog.Handler)`.
  * `palette/engine.go`, unexported: the engine Step 6's `Picker` wraps.
    Each keystroke cancels the last generation's context, counts a new
    generation, compiles one `fuzzy.Matcher`, and starts each target
    provider's search on a goroutine, or a `tea.Tick` carrying the
    generation for a provider with a debounce. Searches send into a
    channel of 64. The drain command waits for the first event, then takes
    what arrives within 16 ms, up to 64 hits, and returns
    `hitsMsg{gen, events}`. `update` drops a batch or a debounce of another
    generation, merges hits into a top-k kept in order under the limit,
    de-duplicates by `ID`, and issues the next drain while searches run.
    A search's error or panic is logged to the `slog.Handler` and is one
    row naming the provider.
* **Choices made within MADR §4.** None changes a decision; they are
  listed for Step 6, which builds on them:
  1. `Start` runs once per opening of the picker, on a goroutine of its
     own, with a context that closing cancels; each search waits for it
     or for its own cancellation. The first version used `sync.Once`,
     which held a second search on a mutex behind a slow `Start`,
     deaf to its own cancellation. `TestNoLeaks` hung on it: synctest does
     not count a mutex wait as durable, so `synctest.Wait` never returned.
  2. `Stop` is called off the event loop, after the opening's searches and
     `Start`s have returned, and only for a provider used in that opening.
     A panic in it is logged.
  3. A drain also returns as soon as every search it waits for has ended,
     so the last batch does not wait out its frame.
  4. Without a prefix, every available provider searches, prefixed ones
     too. A prefix belongs to an available provider only.
  5. An empty query asks the discoverers alone. `Search` is never called
     with empty text.
  6. A hit with an empty `ID` is never de-duplicated.
  7. Failure rows follow the hits, outside the limit, in the order the
     failures arrived. `?`'s rows carry the prefix as `Text` and the
     provider's name as `Group`.
  8. Registration returns a `When` that does not parse as an error, and
     reads `Prefix` and `Debounce` once. `Register`'s exported signature is
     Step 6's.
* **Tests** (`palette/engine_test.go`), each inside `testing/synctest`
  through a loop that plays Bubble Tea's, with fake providers:
  `TestStaleHitsNeverAppear`; `TestBatches` (`64`: 200 hits arrive in
  batches of at most 64; `frame`: a lone hit within 16 ms);
  `TestFailuresAreContained` (an error, a panic and a failed `Start`: three
  rows naming the providers, none showing the error, each error in the log,
  and the good provider's hits); `TestNoLeaks` (the keystroke cancels the
  last search, closing cancels a `Start` still running and stops the used
  provider and not the unused one; `synctest.Test` returns, and the
  `goroutineleak` profile, taken after it, reports `total 0`);
  `TestAvailability`; `TestPrefixes`; `TestDebounce`; `TestDuplicates`;
  `TestLimit` (checked after every message); `TestGrouping`;
  `TestStartAndDiscover`. `go test -race -count=200 ./palette/` is clean.
* **Conformance:** `TestEveryEnumRoundTrips` failed on the new enum
  ("palette.Grouping is an enum the table lacks") before the table gained
  `palette.Grouping`.
* **Mutations,** through `python3 -m tools.verify mutate` on scratch
  copies, 21 killed of 21. The PLAN's five come first:

  | Mutation | Killed by |
  | :--- | :--- |
  | `update` ignores a batch's generation | `TestStaleHitsNeverAppear`: `rows ["a" "b"]; want only b` |
  | a keystroke does not cancel the last search | `TestNoLeaks`: "the search for a was not cancelled by the next keystroke" |
  | a provider's panic is not recovered | `TestFailuresAreContained`: the test binary panics, `boom` |
  | de-duplication keeps the first hit | `TestDuplicates`: `x:subsequence/1`, want `x:exact/5` |
  | the drain ignores the 64-hit bound | `TestBatches`: `batches [200]` |
  | a stale debounce still launches | `TestDebounce`: `40ms in, searched ["ab"]` |
  | the kept hits are not cut to the limit | `TestLimit`: "13 hits kept; the limit is 5" |
  | closing does not cancel the opening's context | `TestNoLeaks`: synctest's "deadlock: main bubble goroutine has exited but blocked goroutines remain" |
  | closing stops a provider that was not used | `TestNoLeaks`: "stopped 1 and 1 times" |
  | every search starts the provider afresh | `TestStartAndDiscover`: "started 2 times in one opening" |
  | availability is ignored | `TestAvailability`: `editor searched ["x" "y"]` |
  | a prefix is not taken off the query | `TestPrefixes`: `commands asked ">ab"` |
  | `?` searches instead of listing the prefixes | `TestPrefixes`: `? lists []` |
  | the drain ignores its frame | `TestBatches`: "after 16ms: rows []" |
  | a failure is not logged | `TestFailuresAreContained`: the log lacks `provider=bad` |
  | a hit's group does not default to its provider | `TestGrouping`: `b1's group ""` |
  | `ByProvider` orders by score | `TestGrouping`: `provider: rows ["b1" "a1" "b2" "a2"]` |
  | an empty query skips the discoverers | `TestStartAndDiscover`: `an empty query: rows []` |
  | the drain waits out its frame after every search ended | `TestDuplicates`: `rows []` |
  | the enums table lacks `palette.Grouping` | `TestEveryEnumRoundTrips` |
  | a goroutine leaks beside `TestNoLeaks` | `noLeaks`: "leaked goroutines" |

  *2026-10-25:* the first run left "closing does not cancel the opening's
  context" alive. Closing also cancels the generation's context, so only
  a running `Start` could tell the two apart. `TestNoLeaks` gained a
  provider whose `Start` runs until closing, and the mutation is now
  killed.
* **Names:** `python3 -m tools.verify names` over the eleven exported
  types: "11 planned name(s), clean". `docs/glossary.md` gains two
  `palette` rows.
* **Gates,** through `tools.verify` on the final tree: macOS, over the
  step's six Go files and four records, "gates: 19 passed, 0 failed, 0
  skipped"; the Linux host and the Windows host each "gates: 18 passed,
  0 failed, 1 skipped", the skip being `gofmt`, which a remote run is given
  no files for. `release-check` is clean on each: 309 tracked files on
  macOS, where `palette/` is untracked, and 314 on the hosts, whose
  copies' index tracks its five files. `make lint` reports "0 issues" for
  `GOOS=linux`, `darwin` and `windows`. The identifier scan of the change
  and the untracked files is clean.
* **Not done here, by the PLAN:** `Ranges` are filled for drawn rows, and
  failure rows are drawn muted, by Step 6's pane. `Register`, `Open` and
  the `Picker` are also Step 6's.
