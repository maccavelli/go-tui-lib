---
status: proposed
date: 2026-10-02
associated-madr: "0008-MADR-command-palette.md"
---
# Implement the command palette and the fuzzy matcher

Associated MADR: [0008-MADR-command-palette.md](0008-MADR-command-palette.md)

## Goal

Ship `fuzzy` and `palette`, so that a program finds and runs any command,
file, pane, theme or other item from one overlay, and drives slash and
`@path` completion with the same engine.

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag the release.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0008-*`, `docs/README.md` | accept the records |
| 2 | `fuzzy/` (matcher) | graphemes, folding, tiers, scoring, `Rank`, `Cells` |
| 3 | `fuzzy/` (index) | trigram index, recall fixture |
| 4 | `palette/` (engine) | providers, scopes, generations, streaming, de-duplication |
| 5 | `palette/` (pane) | overlay pane, rendering, keys, mouse, goldens |
| 6 | `palette/` (providers) | commands, recents, static, panes, files, completion mode |
| 7 | `palette/` (arguments, recents store) | argument prompts, `Frecency` |
| 8 | `README.md`, `docs/`, `docs/guides/command-palette.md` | documentation, release notes, close-out |

No module is added. `fuzzy` uses the standard library and `x/ansi`.
`palette` uses modules already required, and the packages of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md) and
[0007-PLAN-keymap-engine.md](0007-PLAN-keymap-engine.md).

### Out of scope

* A `.gitignore` reader (MADR Q4). The file provider takes a `PathFilter`.
* A Unicode normaliser. `Normalize` is a hook only.
* A top-centred overlay anchor (MADR §7). It needs a workspace record.
* JSON Schema shapes outside the §6 subset.
* Any change in pi-go.
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

## Implementation Steps

### Step 1: records

The owner accepts the MADR and answers Q1–Q4. Record the answers, set the
MADR `accepted` and this PLAN `in-progress`, and update `docs/README.md`.
Confirm that 0006 and 0007 are complete, since Steps 4–6 import their
packages. If either is not, stop and ask.

### Step 2: `fuzzy`, the matcher

* MADR §2: `Matcher`, `New` and its options, `Score`, `Match`, `Span`,
  `Tier`, the generic methods `Rank` and `RankContext`, `Ranked`, `Cells`.
* A short probe first confirms that Go 1.27.1 compiles generic methods on
  `*Matcher` and that `go vet` and golangci-lint v2.14.0 accept them. If a
  linter rejects them, stop and record it; the fallback is the package
  functions `fuzzy.Rank[T](m, …)`, which is a MADR amendment.
* **Tests:**
  * a table pins tier and order for command titles, paths, camelCase,
    digits, Cyrillic, CJK, combining marks and ZWJ emoji;
  * `Smart` case: a query with an upper-case letter is case-sensitive;
  * the ASCII and the grapheme paths agree on every ASCII case;
  * properties over random strings: spans lie on grapheme boundaries,
    inside the candidate, in order, and spell the query under folding;
    `Score` and `Match` agree on tier and value; ranking is total and
    stable;
  * `RankContext` returns `ctx.Err()` promptly once cancelled;
  * `Cells` gives the same ranges as `ansi.Cut` with both width methods;
  * `FuzzMatch` finds no panic and no out-of-range span, and joins the CI
    fuzz step.
* **Benchmarks** (`b.Loop`): `Score` on ASCII reports 0 allocations;
  ranking 10 000 titles is measured and recorded.
* **Mutations:**
  * tiers are ignored in the order;
  * a span ends inside a grapheme;
  * the boundary bonus is dropped;
  * ties are broken by map order;
  * `Smart` case is always insensitive.

### Step 3: `fuzzy`, the index

* MADR §3: `Index`, `NewIndex`, `With`, `Len`, `At`, `Candidates`, and the
  options.
* **Tests:**
  * an exact or prefix match is never dropped by the index;
  * one-, two- and four-grapheme queries follow toad's three strategies;
  * `With` leaves the old index unchanged, and both are readable from many
    goroutines under `-race`;
  * a fixture corpus of 20 000 synthetic paths records the index's recall
    against a full scan, for the execution record;
  * `FuzzIndex` finds no panic.
* **Benchmarks:** building the index and querying 100 000 paths, recorded.
* **Mutations:**
  * the overlap threshold is ignored;
  * padding is dropped, so prefixes lose their trigrams;
  * `With` mutates the receiver.

### Step 4: `palette`, the engine

* MADR §4: `Provider` and its optional interfaces, `Query`, `Hit`,
  `Scope`, `Register`, the generation counter, the drain command, top-k
  merge, de-duplication by `ID`, contained failures and `WithLogger`.
* **Tests, with fake providers and `testing/synctest`:**
  * a slow provider's hits for an old query never appear after a new
    keystroke;
  * hits arrive in batches of at most 64 or one frame;
  * a provider that errors or panics shows one muted row, and the others
    finish;
  * the `goroutineleak` profile shows no leaked goroutine after a search
    is cancelled, and after the palette closes;
  * scopes follow `when` expressions, and a prefix restricts the query to
    its provider; `?` lists the prefixes;
  * debounce delays a provider's search and is cancelled by the next
    keystroke;
  * two hits with one `ID` keep the better score.
* **Mutations:**
  * the generation check is removed;
  * the context is not cancelled on a new keystroke;
  * a provider panic is not recovered;
  * de-duplication keeps the first hit.

### Step 5: `palette`, the pane

* MADR §7: the overlay pane on `textinput` through `workspace.Model[M]`,
  the results window, help line and count, highlights, the selected-row
  marker, danger badges, truncation, the `palette` key context and
  `DefaultKeyMap`, `EscConsumer`, and the mouse.
* **Tests, through a real `workspace.Workspace`:**
  * the palette opens as an overlay, takes keys, and closes on Esc twice
    (clear, then close);
  * up, down, page and tab behave as MADR §7 says;
  * the wheel scrolls and a click runs the row under it;
  * only visible rows are matched for spans;
  * the cursor is the input's, offset into the overlay.
* **Golden frames** across the 0001 §6 matrix at 60 and 100 columns: an
  empty query with recents, a query with highlights, a truncated highlight,
  a danger badge, a disabled row, a provider error row, and "still
  searching".
* **Mutations:**
  * the selected row loses its glyph marker;
  * highlights use rune indexes;
  * Esc closes with a non-empty query;
  * truncation counts bytes.

### Step 6: built-in providers and completion mode

* MADR §5: `Commands`, `Recents`, `Static`, `Panes`, `Files` with
  `Walker`, `FSWalker` and `PathFilter`, and `AsCompletion` with
  `SetQuery` and `SelectedMsg`.
* **Tests:**
  * `Commands` hides commands whose `when` fails, matches aliases and slash
    names, and carries `Danger` and `Args`;
  * `Files` over an `fstest.MapFS` uses the index above its threshold and a
    full scan below it, honours `PathFilter`, and ranks by crush's tiers;
  * `Panes` focuses the chosen pane and shows a hidden one;
  * completion mode is non-modal: other keys reach the focused editor pane
    that drives it, and it emits `SelectedMsg`.
* **Mutations:**
  * `Commands` ignores `when`;
  * `FSWalker` ignores `PathFilter`;
  * completion mode is modal.

### Step 7: arguments and recents

* MADR §6: the argument stage for the schema subset, and `Frecency` with
  its versioned JSON.
* **Tests:**
  * a command with string, integer, boolean and enum arguments is prompted
    in order, honours `required` and `default`, rejects a bad integer, and
    runs with the collected JSON;
  * a schema outside the subset falls through to `SelectedMsg` with no
    arguments;
  * frecency decays with the half-life under an injected clock; `Max`
    evicts the lowest; recents boost but do not overturn a better tier;
  * a JSON round trip is exact, an unknown member and an unknown version
    are refused.
* **Mutations:**
  * the half-life is ignored;
  * `required` is not enforced;
  * an unknown version is accepted.

### Step 8: documentation and close-out

* **`docs/guides/command-palette.md`:** opening the palette; writing a
  provider and scoping it; prefixes; completion pop-ups; persisting
  recents; the file provider and its `fs.FS`.
* **Examples:** `ExamplePalette` and `ExampleMatcher_Rank`, compiled.
* **Docs tree:** `docs/architecture.md` gains both packages and their
  imports; `docs/README.md` gains rows; README Status.
* **Release notes** in the execution record. Verification as below. Mark
  `complete` after CI is green on the pushed tree. The owner tags.

## Verification

* Every step's mutations are killed.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`.
* `go mod tidy -diff` is clean, and `go.mod` gains no requirement.
* The conformance scan passes on both new packages.
* The benchmarks and the index's recall are recorded in the execution
  record.
* The identifier scan of 0001-PLAN V7 finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–8 and tags the minor release. pi-go
  adopts the palette and the completion mode under its own records.
* **Rollback.** Before the push, each step is one local commit. After it,
  a patch release fixes forward. `v0` allows an incompatible change, and
  the release notes say so.

## Execution Record

None yet.
