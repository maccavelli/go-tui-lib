---
status: proposed
date: 2026-10-02
associated-madr: "0007-MADR-keymap-engine.md"
---
# Implement the keymap engine

Associated MADR: [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md)

## Goal

Ship `keymap`, and move `workspace`'s own bindings onto it, so that:

* every key binds to a `command.ID`;
* contexts follow the focus path;
* chords and a leader key work, with tea-native timeouts;
* users can edit a VS Code-format `keybindings.json`, checked by an
  emitted JSON Schema;
* bubbles components and `help` still get `key.Binding` and
  `help.KeyMap`.

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag the release.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0007-*`, `docs/README.md` | accept the records |
| 2 | `keymap/notation.go` and tests | `Stroke`, `Sequence`, parse, format, VS Code spelling, matching, `BubblesKeys` |
| 3 | `keymap/keymap.go`, `keymap/conflict.go` and tests | `Rule`, layers, contexts, `Build`, `Rebuild`, `Lookup`, conflicts, `Features`, `Default` |
| 4 | `keymap/matcher.go` and tests | chords, leader, timeouts, replay, release events |
| 5 | `keymap/vscode.go`, `keymap/jsonc.go`, `keymap/schema.go` and tests | loader, exporter, JSON Schema |
| 6 | `keymap/help.go` and tests | `Binding`, `Help`, `Describer` |
| 7 | `workspace/keys.go`, `workspace/workspace.go`, `workspace/*_test.go` | command IDs, `DefaultRules`, `WithBindings`, `KeyPath`, `KeyContexter`, deprecations |
| 8 | `docs/guides/`, `docs/architecture.md`, `docs/README.md`, `README.md` | documentation, release notes, close-out |

`go.mod` and `go.sum` gain nothing. `keymap` uses the standard library,
`bubbletea`, `bubbles/key`, `bubbles/help`, and this repository's `command`
and `when` packages.

### Out of scope

* `command` and `when` themselves, which belong to
  [0006-PLAN-command-registry.md](0006-PLAN-command-registry.md).
* Where the user's file lives. An XDG config loader is a later record.
* A vim keymap layer for an editor pane. The `mode:` contexts allow one,
  and it is its own record.
* Removing the deprecated `workspace.KeyMap`, which is a later record
  (MADR Q4).
* Any change in pi-go.
* `git push` and tags, which the owner does.

## Prerequisites

* [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  is complete. Its default keys are the ones Step 7 carries over.
* [0004-PLAN-integrate-charm-v2-and-go-1-27.md](0004-PLAN-integrate-charm-v2-and-go-1-27.md)
  is complete. The workspace's `help.KeyMap` is the one Step 7 delegates.
* [0006-PLAN-command-registry.md](0006-PLAN-command-registry.md) is
  complete, so `command.ID`, `when.Parse`, `when.Context` and
  `Registry.Lookup` exist.
  If 0006's final names differ from those in the MADR, this PLAN is
  amended to match before Step 2 starts.

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

The owner accepts the MADR, answering Q1–Q5. Record the answers, set the
MADR `accepted` and this PLAN `in-progress`, and add the rows to
`docs/README.md`. If an answer differs from the recommendation, amend the
MADR and this PLAN before Step 2.

### Step 2: notation

* `Stroke`, `Sequence`, `MaxStrokes`, `Event`, `ParseStroke`,
  `ParseSequence`, `String`, `VSCode`, `Matches` and `BubblesKeys`, as
  MADR §2 gives them.
* **Tests:**
  * **round trips:** a property test over random strokes and sequences
    (every modifier set, named keys, `f1`–`f63`, printable runes, `Scan`
    strokes) shows that canonical parse and format round-trip, and that
    `ParseSequence(q.VSCode())` returns `q`;
  * **normalization:** `A` becomes `shift+a`; `Ctrl+Shift+A`,
    `shift+ctrl+a` and `ctrl+shift+a` are one stroke; `escape`, `pageup`
    and `cmd` map as the MADR's table says; lock modifiers, empty
    strokes, five-stroke sequences and unknown names are refused with a
    message naming the bad part;
  * **matching:** a table of `tea.Key` values built as ultraviolet decodes
    them, with the decoder lines cited in the test:
    * legacy upper-case letters, shifted symbols and `alt+` keys;
    * Kitty input with `ShiftedCode` and `BaseCode`;
    * Windows console input with caps lock;
  * **interop:** for each matching case, `key.Matches(msg,
    key.NewBinding(key.WithKeys(s.BubblesKeys()...)))` agrees with
    `s.Matches(msg.Key())`.
* **Mutations:**
  * lock modifiers are not masked;
  * the `Text` comparison is dropped;
  * `VSCode` writes `super` as `meta`;
  * `BubblesKeys` drops the `Text` spelling.

### Step 3: rules, layers, contexts and conflicts

* `Context`, `Global`, `Path`, `Layer`, `Rule`, `Origin`, `Options`,
  `Features`, `FeaturesFrom`, `Default`, `Build`, `Rebuild`, `Lookup`,
  `Bindings`, `Conflicts`, `Rules` and `Conflict`, as MADR §3–§5 give them.
* **Tests:**
  * **resolution:** innermost context first, later layer first, later rule
    first, and `when` filtering through the `when` package;
  * **removal:** with keys, removing an unbound key fails; with no keys,
    every binding of that command in that context goes;
  * **conflicts:** each kind (Duplicate, Prefix, Reserved, Unsupported)
    comes from a planted keymap, and `Build` returns all of them joined,
    each naming its origin; a Shadow is listed by `Conflicts()` and does
    not fail;
  * **features:** a default that needs `Disambiguate` uses its fallback
    when the features are empty, and its own keys after `Rebuild`;
  * **leader:** `<leader>` expands, and fails when no leader is set;
  * **immutability:** `Lookup` under `-race` from several goroutines on one
    `Keymap`.
* **Benchmarks:** `Lookup` on a 200-rule keymap with a five-context path.
* **Mutations:**
  * the path is walked outermost first;
  * `Build` returns only the first error;
  * the Prefix check is skipped;
  * reserved keys are not checked for the user layer.

### Step 4: the matcher

* `Matcher`, `NewMatcher`, `Update`, `SetKeymap`, `Pending`, `Result` and
  `TimeoutMsg`, as MADR §6 gives them.
* **Tests, all with a recording `Options.Tick`:**
  * a two-stroke chord, a leader sequence, and a four-stroke sequence
    complete;
  * a prefix returns `Handled` and one tick command; a `TimeoutMsg` with
    the current generation clears it and returns its `Replay`, without the
    leader;
  * a `TimeoutMsg` with an old generation does nothing;
  * a change of `Path` while pending cancels it;
  * `SetKeymap` cancels anything pending;
  * `KeyReleaseMsg` matches only `Release` strokes;
  * a key that matches nothing is not `Handled`;
  * **the real timer:** inside `testing/synctest.Test`, the command from
    the default `tea.Tick` yields `TimeoutMsg` after the one-second
    default on the fake clock, with no wall-clock wait.
* **Mutations:**
  * the generation is not compared;
  * a path change does not cancel;
  * the leader is replayed;
  * `Replay` drops the first stroke.

### Step 5: the VS Code loader, exporter and schema

* `LoadVSCode`, `ExportVSCode` and `Schema`, with the JSONC scanner, as
  MADR §8 gives them.
* **Tests:**
  * line and block comments, comments containing `"` or `//` inside
    strings, and trailing commas are accepted;
  * the scanner keeps every byte offset, so a reported line and column
    point at the bad character; a test compares them with a hand-placed
    marker;
  * an unknown member, an unknown command (with `ids` set), a bad key and
    a bad `when` each give an error with file, line, column and JSON
    Pointer; three bad rules in one file give three errors, and the good
    rules are still returned;
  * `-command` with and without `key`;
  * `LoadVSCode(ExportVSCode(rules))` returns `rules`, for the workspace
    defaults and a planted user file;
  * the schema is valid JSON with `$schema` set to draft 2020-12; its
    `command` enum holds every ID and its `-` form; its `key` pattern
    compiles with `regexp` and accepts every canonical form from Step 2's
    property test; an ID with an args schema gets an `if`/`then` pair.
* **Fuzz:** `FuzzLoadVSCode` finds no panic, and every error it returns
  has a line and column inside the input. It joins the CI fuzz step.
* **Mutations:**
  * the scanner deletes comments instead of blanking them;
  * a `//` inside a string is treated as a comment;
  * errors stop at the first;
  * unknown members are accepted.

### Step 6: bubbles adapters and help

* `Describer`, `FromRegistry`, `Binding` and `Help`, as MADR §7 gives
  them.
* **Tests:**
  * `Binding` for a one-stroke command matches with `key.Matches` exactly
    when the matcher would; its help is the primary sequence and the
    title;
  * a chord binding is enabled, shows its full text in `help`, and never
    matches a single key;
  * a removed or unbound command gives a disabled binding that `help`
    skips;
  * `Help` short and full output, through `help.Model`, rendered as
    golden files across the 0001-MADR §6 matrix.
* **Mutations:**
  * the chord's full text is replaced by its first stroke;
  * `Help` groups contexts outermost first.

### Step 7: move `workspace` onto the engine

* The command IDs, `DefaultRules`, `WithBindings`, `KeyPath` and
  `KeyContexter`, as MADR §9 gives them. `KeyMap`, `DefaultKeyMap` and
  `WithKeyMap` are marked `// Deprecated:` and converted into rules for the
  `workspace` context.
* `Workspace.key` asks a `keymap.Matcher` with `KeyPath()`. It keeps the
  `EscConsumer` check, and sends unmatched keys and `Replay` strokes to the
  focused pane or the top modal overlay.
* The workspace's `help.KeyMap` delegates to `keymap.Help`.
* **Tests:**
  * every existing workspace test passes unchanged, with no option and with
    the deprecated `WithKeyMap`;
  * the same behaviour through `WithBindings` with `DefaultRules`;
  * a user rule rebinds `workspace.zoom`, and a `-workspace.zoom` rule
    removes it;
  * `KeyPath` for a focused pane, a `KeyContexter` pane, a non-modal
    overlay and a modal overlay;
  * a chord prefix typed in an editor pane is replayed to it when the
    timeout expires;
  * `ctrl+c` is never consumed, and no default binds it;
  * golden frames are unchanged.
* **Mutations:**
  * `KeyPath` keeps the layout contexts under a modal overlay;
  * `Replay` strokes are dropped;
  * `WithKeyMap` ignores the struct.

### Step 8: documentation and close-out

* **`docs/guides/keymaps.md`:**
  * binding a command;
  * contexts and the focus path;
  * chords and the leader;
  * the user file and its schema;
  * reserved keys;
  * what changes on a Kitty terminal;
  * moving from `workspace.KeyMap`.
* **Docs tree:**
  * `docs/guides/building-workspaces.md` "keys" section;
  * `docs/architecture.md`, with `keymap` and its imports;
  * `docs/README.md` rows;
  * the README Status.
* **Release notes** in the execution record, naming the deprecations.
* **Verification** as below. Mark `complete` after CI is green on the pushed
  tree. The owner tags.

## Verification

* Every step's mutations are killed.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`.
* `go mod tidy -diff` is clean, and `go.mod` gains no requirement.
* `keymap` starts no goroutine and reads no clock outside `Options.Tick`.
  A test runs the matcher inside `testing/synctest` to show it.
* `internal/conformance` passes for `keymap`.
* No default binding in `keymap` or `workspace` is `ctrl+c`.
* The identifier scan of 0001-PLAN V7 finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–8 and tags the minor release after
  0006's. pi-go moves its own bindings onto `keymap` under its own
  records.
* **Rollback.** Before the push, each step is one local commit. After it, a
  patch release fixes forward. A program that never calls `WithBindings`
  keeps 0002's behaviour, because the deprecated `KeyMap` path stays.

## Execution Record

None yet.
