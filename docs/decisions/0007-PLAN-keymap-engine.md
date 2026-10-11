---
status: in-progress
date: 2026-10-11
associated-madr: "0007-MADR-keymap-engine.md"
---
# Implement the keymap engine

Associated MADR: [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md)

*Revised 2026-10-02.*

* The owner answered the MADR's Q1–Q5, and the MADR is `accepted`. Q4
  departs from the recommendation: `workspace.KeyMap` is removed now, not
  deprecated. Steps 1, 7 and 8, Out of scope, and Rollback are revised to
  match.
* Steps 9–11 are added for MADR amendment A1. *Later on 2026-10-02:* the
  owner accepted A1 and answered Q6 and Q7 with the recommendations, so
  Steps 9–11 are in scope.
* This PLAN stays `proposed` until the owner approves execution.

*Revised 2026-10-07, for MADR amendment A2.* The steps it touches:

* **Prerequisites:**
  * [0006-PLAN-command-registry.md](0006-PLAN-command-registry.md) is
    complete. The names it shipped differ from the MADR's, and A2 records
    them.
  * [0005-PLAN-terminal-capabilities-and-services.md](0005-PLAN-terminal-capabilities-and-services.md)
    is complete too.
* **Step 7:**
  * the workspace's defaults bind the real IDs of A2;
  * a matched key is dispatched with `Origin: command.OriginKey`;
  * the bindings for focus by index and directional resize wait on the
    MADR's owner question Q8, which is answered before Step 7 starts.
* **Step 8:** the guides describe the bindings Q8's answer gives.
* **Step 9:** `glyph.Set` gains the modifier and arrow glyphs, each one
  cell wide with an ASCII twin, before the labels that draw them.

*Revised 2026-10-10, for MADR amendments A3 and A4, and the facts re-read
below.* A3 renames the types; A4 brings the API to AGENTS.md's "API
conventions", answers Q8, and supersedes Q4. Where a step below names an
old form, read the new one:

| Was | Is | Record |
| :--- | :--- | :--- |
| `Context`, `Global`, `Path []Context` | `command.Scope`, `command.Global`, `Path []command.Scope` | A3 |
| `Rule`, `Rule.Context` | `Binding`, `Binding.Scope` | A3 |
| `Origin` | `Location` | A3 |
| `Conflict`, `Conflict.Context` | `BindingConflict`, `BindingConflict.Scope` | A3 |
| `ConflictKind` `Unsupported` | `NotSupported` | A3 |
| `Result` | `Outcome` | A3 |
| `Options` | `Settings` | A3 |
| `Match` | `Resolved` | A3 |
| `Matcher`, `NewMatcher` | `Dispatcher`, `NewDispatcher` | A3 |
| `Build(o, layers...)` | `New(s, layers...)` | A4.3 |
| `Describer` (a function type) | an interface, with `DescriberFunc` | A4.4 |
| `Layer` constants `Default`, `Program`, `User` | `DefaultLayer`, `ProgramLayer`, `UserLayer` | A4.7 |
| `Rule.Args json.RawMessage` | `Binding.Args jsontext.Value` | A4.6 |
| `KeyMap`, `DefaultKeyMap`, `WithKeyMap` removed in Step 7 | deprecated in Step 7 (`v0.11.0`), removed in Step 12 (`v0.12.0`) | A4.2 |

The order of the steps is 1, 2 to 6, 9 to 11, 7, 8 and 12.

## Goal

Ship `keymap`, and move `workspace`'s own bindings onto it, so that:

* every key binds to a `command.ID`;
* contexts follow the focus path;
* chords and a leader key work, with tea-native timeouts;
* users can edit a VS Code-format `keybindings.json`, checked by an
  emitted JSON Schema;
* bubbles components and `help` still get `key.Binding` and
  `help.KeyMap`;
* from amendment A1: which-key data, defaults chosen by
  terminal fact, legacy-key normalisation, labels per operating system,
  live hints, and a key-debug explanation.

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag the release.

## Scope

### Facts re-read before execution (2026-10-10)

The agent read each fact this PLAN rests on again on `8f68556`, after
0014's W0 to W4 (`v0.8.0` to `v0.10.0`) and
[0017-PLAN-python-repository-scripts.md](0017-PLAN-python-repository-scripts.md).

| Fact | Today |
| :--- | :--- |
| The prerequisites are complete: 0002's hardening, 0004, 0005 and 0006; and 0014's W0 to W4 | `docs/README.md` |
| `workspace.KeyMap` (focus next and prev, `FocusPane[9]`, zoom, the four resizes, close), `DefaultKeyMap` and `WithKeyMap` hold, with 0002's keys: `alt+.`, `alt+,`, `alt+1` to `alt+9`, `alt+z`, `alt+shift` arrows, `esc` | `workspace/keys.go:28-37`; `workspace/workspace.go:311` |
| `workspace.Commands` registers the IDs A2 lists, and none takes an index or a direction | `workspace/commands.go` |
| `KeyMapper` is read by the workspace's help, and W2 added `Workspace.Help` | `go doc ./workspace` |
| `glyph.Set` has no modifier or arrow glyph; W2 added `glyph.Tier` | `go doc ./glyph Set` |
| `termcap` gives the keyboard as `Caps.Keyboard()`, a `KeyboardCaps` of `Kitty`, `Enhancements`, `Accepted` and `Releases`, and `KeyboardFlags(Caps)` | `go doc ./termcap KeyboardCaps` |
| A request carries its `when` values in `Request.WhenContext` (W4), and a key's origin is `command.OriginKey` | `command/handler.go` |
| Bubble Tea is `v2.0.10`, bubbles `v2.2.1`; `tea.Key` has `Text`, `Mod`, `Code`, `ShiftedCode`, `BaseCode` | `go list -m`; `go doc charm.land/bubbletea/v2 Key` |
| No type name A3 and A4 give is exported by another package: `Binding`, `Location`, `Settings`, `Dispatcher`, `Outcome`, `Resolved`, `BindingConflict`, `ConflictKind`, `Layer`, `Path`, `Stroke`, `Sequence`, `Event`, `Features`, `Default`, `KeyFacts`, `Alternative`, `Explanation`, `ActiveKey`, `Describer`, `Keymap` | a search of every package's types |
| **Changed:** AGENTS.md's "API conventions" bind every new exported name: enums through `internal/enum`, which the conformance test checks; typed errors; hooks as interfaces with adapters; `NewX` constructors; opaque option types; a stability line; deprecation for one minor release | `AGENTS.md`; MADR A4 |
| **Changed:** `command` and `termcap` use `encoding/json/v2`, and a raw value is a `jsontext.Value` | 0014-PLAN-canonicalization Step 7 |
| **Changed:** `make fuzz` finds every fuzz target itself, so `FuzzLoadVSCode` joins it with no list to edit | `scripts/go-fuzz.py -a` |
| **Changed:** the repository's scripts are Python; the release is `v0.11.0`, after `v0.10.0` | 0017-PLAN; `git tag` |

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0007-*`, `docs/README.md` | accept the records |
| 2 | `keymap/notation.go` and tests | `Stroke`, `Sequence`, parse, format, VS Code spelling, matching, `BubblesKeys` |
| 3 | `keymap/keymap.go`, `keymap/conflict.go` and tests | `Rule`, layers, contexts, `Build`, `Rebuild`, `Lookup`, conflicts, `Features`, `Default` |
| 4 | `keymap/matcher.go` and tests | chords, leader, timeouts, replay, release events |
| 5 | `keymap/vscode.go`, `keymap/jsonc.go`, `keymap/schema.go` and tests | loader, exporter, JSON Schema |
| 6 | `keymap/help.go` and tests | `Binding`, `Help`, `Describer` |
| 7 | `workspace/keys.go`, `workspace/workspace.go`, `workspace/*_test.go` | command IDs, `DefaultRules`, `WithBindings`, `KeyPath`, `KeyContexter`, removal of `KeyMap`, `DefaultKeyMap` and `WithKeyMap` |
| 8 | `docs/guides/`, `docs/architecture.md`, `docs/README.md`, `README.md` | documentation, release notes, close-out |
| 9 (A1) | `keymap/normalize.go`, `keymap/label.go`, `keymap/alias.go` and tests | `Normalize`, `Label`, `ParseAliases`, `DisplayAliases`, `CopyStrokes` |
| 10 (A1) | `keymap/keymap.go`, `keymap/facts.go`, `workspace/workspace.go` and tests | `KeyFacts`, `Alternative`, `Default.Alternatives`, release degrade, `Reachable`, `Shortcut`, `PushMode` |
| 11 (A1) | `keymap/matcher.go`, `keymap/explain.go` and tests | `ActiveKeys`, pending-sequence editing, `Explain` |

`go.mod` and `go.sum` gain nothing. `keymap` uses the standard library,
`bubbletea`, `bubbles/key`, `bubbles/help`, and this repository's `command`
and `when` packages. For A1 it also uses this repository's
`glyph`, and `termcap`'s fact types (Q7).

**When Steps 9–11 run.** A1 was accepted before Step 7 started, so they
run after Step 6 and before Step 7, and the workspace moves onto the
finished engine.

### Out of scope

* `command` and `when` themselves, which belong to
  [0006-PLAN-command-registry.md](0006-PLAN-command-registry.md).
* Where the user's file lives. An XDG config loader is a later record.
* A vim keymap layer for an editor pane. The `mode:` contexts allow one,
  and its engine belongs to the composer record (MADR amendment A1,
  "Placed elsewhere").
* ~~Removing the deprecated `workspace.KeyMap`, which is a later record
  (MADR Q4).~~ *2026-10-02:* the owner answered Q4 "remove now", so the
  removal is in Step 7.
* A which-key panel, and the `keys debug` view. Steps 10 and 11 give their
  data, and the components are later records.
* The macOS dropped-modifier probe, which needs cgo or `purego` (MADR
  amendment A1).
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
* For Step 10 only:
  [0005-PLAN-terminal-capabilities-and-services.md](0005-PLAN-terminal-capabilities-and-services.md)
  is complete, so the facts behind `KeyFacts` exist. This includes the
  record of Kitty flags actually pushed. If 0005's final names differ,
  this PLAN is amended before Step 10.

## Rules for every step

*2026-10-10:* rule 3's checks are now the Go checks of
[0014-PLAN-component-native-forms.md](0014-PLAN-component-native-forms.md)
Rule 2, as W4 runs them: `make pre-add-check`, `make lint`, the cross `go
vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`, `-race`,
`-shuffle=on -count=2`, `LC_ALL=C`, workspace mode, `go mod tidy -diff`,
`make vuln`, `make apicheck`, `make examples` and `make release-check`; CI's
other checks (`shellcheck`, `markdownlint-cli2`, `actionlint`, the Python
scripts' tests); and the Windows test host, with its index synced to the
tree. `make apicheck` may show only the lines this PLAN lists in
`scripts/apicheck.allow`, under a header naming it and the step.

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

## Implementation Steps

### Step 1: records

~~The owner accepts the MADR, answering Q1–Q5. Record the answers, set the
MADR `accepted` and this PLAN `in-progress`, and add the rows to
`docs/README.md`. If an answer differs from the recommendation, amend the
MADR and this PLAN before Step 2.~~

*2026-10-02: partly done.* The answers are recorded, and the MADR is
`accepted`. Q4 differed from the recommendation, and the MADR and this PLAN
are amended to match. Still to do:

* set this PLAN `in-progress` when the owner approves execution;
* ~~record the answers to A1's Q6 and Q7, and set A1 `accepted` or rejected.
  If either answer differs from its recommendation, amend Steps 9–11 before
  they start.~~ Done 2026-10-02: A1 is accepted, and both answers are the
  recommendation.

### Step 2: notation

*2026-10-10:* `Event` (`Press`, `Release`, `Repeat`) has its text forms
through `internal/enum`, and joins `internal/conformance`'s enum table
(A4.6). The package comment ends with the stability line.

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

*2026-10-10:* with A3's and A4's names: `Binding`, `Location`, `Settings`,
`New` and `Rebuild`, `Resolved`, `BindingConflict` (an error) and
`ConflictKind` (`Duplicate`, `Prefix`, `Reserved`, `NotSupported`,
`Shadow`). `Layer` (`DefaultLayer`, `ProgramLayer`, `UserLayer`) and
`ConflictKind` take `enum.Names`, and `Features` `enum.Bits`, each in the
conformance table, `Features` as a bit set. A scope is a `command.Scope`.

*2026-10-10 (D1, MADR A6):* `New(s Settings, defaults []Default, layers
...[]Binding)`; Prefix within one scope, and the innermost scope decides
along a path; `New` returns the keymap with its errors; `<leader>` expands
in `New`, so `keymap/notation.go` and its test join this step, for
`ParseSequence`'s placeholder. The tests below gain: a scope deciding
(an inner `g g` hides an outer `g`, and the reverse), and a NotSupported
binding that binds after `Rebuild`. The conflicts test plants Prefix
within one scope, and a cross-scope Shadow against Global.

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

*2026-10-10:* `Dispatcher`, `NewDispatcher` and `Outcome` (A3).
`Settings.Tick` stays a function field (A4.5).

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

*2026-10-10:* `LoadVSCode` returns `[]Binding`, and its errors are
`*LoadError`s, with the file, line, column and JSON Pointer, and `Unwrap`
(A4.6). The loader, exporter and schema use `encoding/json/v2` and
`jsontext`. `make fuzz` discovers `FuzzLoadVSCode`; there is no list to
edit.

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

*2026-10-10:* `Describer` is an interface with `DescriberFunc`, and
`FromRegistry` returns one (A4.4). A test plants a `DescriberFunc`.

*2026-10-11 (D1, MADR A7):* the method is `KeyBinding`, not `Binding`, as
A3 gave `Binding` to a type. `Help`'s full help lists the IDs given, each
once, in the column of the scope its primary sequence is bound in,
innermost first; with no IDs, short and full help list every command bound
along the path. Below, read `KeyBinding` for the method `Binding`.

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

*2026-10-10, A4 items 1 and 2:*

* **The two commands of Q8,** in `workspace/commands.go`, before the
  defaults: `workspace.focus.index` (`{"index": n}`, 1 to 9) and
  `workspace.resize.direction` (`{"direction": "left|right|up|down"}`),
  each doing what today's key does, with tests that the command and the
  key give the same layout state. `DefaultRules` binds `alt+1` … `alt+9`
  and the four `alt+shift` arrows to them.
* **`KeyMap`, `DefaultKeyMap` and `WithKeyMap` are deprecated,** not
  deleted: each marked `// Deprecated: use … (0007-MADR A4)`, and
  `WithKeyMap` converted into rules for the `workspace` context. A test
  in the package shows that a program's `WithKeyMap` still works through
  `v0.11.x`. The `go doc` test below moves to Step 12, and the mutation
  "an exported `KeyMap` type is left in place" with it.
* A matched key is dispatched with `Origin: command.OriginKey` and the
  path's `when` values in `Request.WhenContext`. `Workspace.Help` (W2)
  reads the engine.

* The command IDs, `DefaultRules`, `WithBindings`, `KeyPath` and
  `KeyContexter`, as MADR §9 gives them.
* `DefaultRules` holds MADR §9's table: `alt+.`, `alt+,`, `alt+1`–`alt+9`,
  `alt+z`, `alt+shift+` arrows and `esc` in `overlay`. These are
  0002-PLAN-harden-workspace-v0-1-1.md's keys.
* ~~`KeyMap`, `DefaultKeyMap` and `WithKeyMap` are marked `// Deprecated:`
  and converted into rules for the `workspace` context.~~ *2026-10-02 (Q4):*
  `KeyMap`, `DefaultKeyMap` and `WithKeyMap` are deleted, with every use
  inside the repository.
* `Workspace.key` asks a `keymap.Matcher` with `KeyPath()`. It keeps the
  `EscConsumer` check, and sends unmatched keys and `Replay` strokes to the
  focused pane or the top modal overlay.
* The workspace's `help.KeyMap` delegates to `keymap.Help`.
* **Tests:**
  * every existing workspace test that sets no key option passes unchanged;
  * tests that built a `KeyMap` are rewritten as rules, and the same
    behaviour holds through `WithBindings` with `DefaultRules`;
  * a golden table of `DefaultRules` (command, keys, context) pins MADR §9's
    defaults;
  * `go doc ./workspace` output, checked by a test, names no `KeyMap`,
    `DefaultKeyMap` or `WithKeyMap`;
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
  * ~~`WithKeyMap` ignores the struct;~~ *(removed with `WithKeyMap`)*
  * a default is set back to `alt+]`, which the golden table must catch;
  * an exported `KeyMap` type is left in place, which the `go doc` test must
    catch.

### Step 8: documentation and close-out

*2026-10-10:* the release is `v0.11.0`, released as W4's Step 8 did: the
documents, `docs/guides/migrating.md`'s "Migrating to `v0.11.0`" (the
deprecated `KeyMap` and its rules), the disclosure guard, the owner's push
and tag, the consumer smoke test (the deprecated names warn under
SA1019 and compile; the new ones build), and the notes. The guide's "moving
from `workspace.KeyMap`" section describes a deprecation, not a removal.

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
* **Release notes** in the execution record, under a breaking-changes
  heading, as MADR §10 lists them:
  * the removal of `KeyMap`, `DefaultKeyMap` and `WithKeyMap`;
  * their replacements;
  * the field-to-command mapping;
  * one rebinding written as a rule.
* The guide's "moving from `workspace.KeyMap`" section gives the same
  mapping, as a migration for programs that called `WithKeyMap`.
* **Verification** as below. Mark `complete` after CI is green on the pushed
  tree. The owner tags.

### Step 9 (A1): normalisation, labels and aliases

*2026-10-10:* `glyph.Set` gains the modifier and arrow glyphs first, each
one cell wide with an ASCII twin, and every `glyph.Tier` gives them; the
additions are compatible.

*2026-10-11 (D1, MADR A8):* the modifier and arrow glyphs are
`glyph.Set.Keys`, a `KeyGlyphs`. Its Unicode forms are one cell, and its
ASCII twins are A1's words, so "each one cell wide" above holds for the
Unicode forms only. `glyph`'s tests check `Keys` apart, and
`docs/architecture.md`'s `glyph` row, which says every glyph is one cell,
joins the step. With A3's names: `Settings.GOOS` and `Dispatcher.Update`.

* `Normalize`, `Stroke.Label`, `Sequence.Label`, `ParseAliases`,
  `DisplayAliases`, `CopyStrokes` and `Options.GOOS`, as MADR amendment A1
  gives them. `Matcher.Update` normalises before matching.
* **Tests:**
  * **normalisation:** a table of legacy messages, each built as
    ultraviolet decodes it, with the decoder lines cited in the test:
    * C0 bytes with no modifier, where ESC, TAB, CR and BS keep their names;
    * upper-case letters with `ctrl` and no `shift`;
    * `ctrl+5` and `ctrl+4`;
    * BS and DEL with modifiers;
    * Windows AltGr as `ctrl+alt` with printable `Text`;
    * `super` with stray `meta` and `hyper`;
  * no rule changes once `Disambiguate` is in the features, except AltGr and
    `super`;
  * **labels:** golden files for every modifier and arrow, on `darwin`,
    `linux` and `windows`, each in UTF-8 and ASCII; `<leader>` shows the
    leader's label;
  * **aliases:** every `ParseAliases` entry parses to its canonical stroke;
    `DisplayAliases` never appears in `Stroke.String()`.
* **Mutations:**
  * ESC is mapped to `ctrl+[`;
  * normalisation runs with `Disambiguate` set;
  * AltGr text is matched as a binding;
  * `Label` ignores the glyph table's ASCII twin.

### Step 10 (A1): terminal facts, alternatives, reachability and hints

*2026-10-10:* `FactsFrom(termcap.Caps)` reads `Caps.Keyboard()`: `Kitty`,
`Enhancements`, `Accepted` and `Releases`.

* `KeyFacts`, `Alternative`, `Default.Alternatives`, `Options.Facts`,
  `Rebuild` with facts, the release rule, `Keymap.Reachable`,
  `Keymap.Shortcut` and `workspace.PushMode`. `FactsFrom(termcap.Caps)`
  is added (Q7).
* `Default.Alternatives` replaces Step 3's `Default.Needs` and
  `Default.Fallback`. Step 3's feature test is rewritten over
  `Alternatives`, as a one-alternative case.
* **Tests:**
  * **alternatives:** a newline default whose preferred key is
    `shift+enter`, with alternatives `alt+enter` (needs `AltEnter`) and
    `ctrl+j`. It binds each of the three under a fact set planted from the
    terminals in 0003-REPORT §9 (VTE before 8200, Apple Terminal, tmux
    before 3.3, a VS Code-family host);
  * a default with no alternative whose needs hold is not bound, and
    `Build` reports nothing;
  * **release:** a `Release` default binds its alternative without the
    `Releases` fact; a user `Release` rule without it is `Unsupported`;
  * `Rebuild` with new facts changes the binding, and help follows;
  * **reachability:** `Reachable` excludes commands bound only in contexts
    off the path, or behind a `when` that fails;
  * **hints:** `Shortcut` is the primary label, and `""` after a `-` rule
    removes the command's last binding;
  * **modes:** `PushMode` adds the context at the end of `KeyPath`, and
    `pop` removes it; a second `pop` does nothing.
* **Mutations:**
  * alternatives are tried last first;
  * a fact named in `Facts` is ignored;
  * `Shortcut` returns the canonical string for an unbound command;
  * `pop` removes the wrong mode when two are pushed.

### Step 11 (A1): which-key data and the key-debug explanation

* `Matcher.ActiveKeys`, `Options.EscClearsPending`,
  `Options.BackspacePops` (on by default, Q6) and `Matcher.Explain`, with
  `ActiveKey` and `Explanation`.
* **Tests:**
  * after a prefix, `ActiveKeys` lists every completing and continuing
    stroke on the path, with labels; with nothing pending it lists the
    path's first strokes;
  * while pending, `esc` clears the sequence and is not replayed;
    `backspace` removes one stroke and leaves the rest pending; with both
    options off, they end the sequence as MADR §6 says;
  * `Explain` reports the raw and normalised key, the winning rule first,
    and the shadowed rules, for a planted keymap with a shadow;
  * neither method changes the matcher's state, which a test checks by
    comparing `Pending()` before and after.
* **Mutations:**
  * `backspace` clears the whole sequence;
  * `ActiveKeys` omits continuing strokes;
  * `Explain` lists the shadowed rule as the winner.

### Step 12: the removals (`v0.12.0`)

*Added 2026-10-10, A4 item 2.*

* `workspace.KeyMap`, `DefaultKeyMap`, `WithKeyMap` and `KeyMap.Bindings`
  are removed, with their deprecation test.
* `scripts/apicheck.allow` lists each removal under this step.
* **Tests:** `go doc ./workspace`, checked by a test, names no `KeyMap`,
  `DefaultKeyMap` or `WithKeyMap` (moved from Step 7).
* **Mutation:** an exported `KeyMap` type is left in place; the `go doc`
  test.
* **The release `v0.12.0`:** "Migrating to `v0.12.0`" in the migration
  guide, the disclosure guard, the owner's push and tag, the smoke test
  (the old names fail to compile), the notes, and the close-out that
  empties `scripts/apicheck.allow` and marks this PLAN `complete`.

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
* No default binding in `keymap` or `workspace` is `ctrl+c`, and
  `CopyStrokes` is never bound by `keymap` itself.
* `go doc ./workspace` lists no `KeyMap`, `DefaultKeyMap` or `WithKeyMap`.
  *2026-10-10:* after Step 12; in `v0.11.x` they are deprecated (A4.2).
* *2026-10-10:* every `keymap` enum round-trips its text, by
  `internal/conformance`'s `TestEveryEnumRoundTrips`; `make apicheck` shows
  only the lines this PLAN lists.
* For Steps 9–11:
  * labels pass the 0001-MADR §6 matrix in UTF-8 and ASCII;
  * `keymap` needs no cgo, which `CGO_ENABLED=0` builds for all three
    operating systems show;
  * `keymap` imports only what the Scope section lists.
* The identifier scan of 0001-PLAN V7 finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–8, with Steps 9–11, and tags the
  minor release after 0006's. pi-go moves its own bindings onto `keymap`
  under its own records.
* **Rollback.** Before the push, each step is one local commit. After it, a
  patch release fixes forward.
  * A program that never called `WithKeyMap` keeps 0002's keys, because
    `DefaultRules` carries them.
  * A program that called `WithKeyMap` must move to rules, or pin the
    previous minor release. `KeyMap` is not restored in a patch release.

## Execution Record

### Step 1: records

* **The amendments:** MADR A4 and this PLAN's revision of 2026-10-10,
  with the facts re-read on `8f68556`, were committed as `68d9b14`. The
  owner chose Q8's A, the deprecation of `KeyMap` over Q4, `keymap.New`,
  and a `Describer` interface with `Settings.Tick` kept, each the
  recommendation.
* **The approval:** the owner wrote "approved to proceed" on 2026-10-10,
  approving this PLAN and A4's item 7, the `Layer` constants'
  names, which is now accepted.
* **Set:** this PLAN `in-progress`; `docs/README.md`'s rows.

### Step 2: notation

#### Deviations

* **D1 (2026-10-10): three points §2 leaves open.**
  * **Found,** reading §2 against Bubble Tea `v2.0.10` and ultraviolet's
    decoder:
    * a `Stroke` has an `Event`, but §2 gives no written form for it, so
      `String()`, the VS Code spelling and this step's round-trip test
      could not be lossless for a release or repeat stroke;
    * ultraviolet gives the keypad its own codes and the main keys' names
      (`key.go`'s `keyTypeString`: `KeyKpEnter` is `enter`, `KeyKp1` is
      `1`), so §2's equal-`Code` rule would not let keypad Enter fire an
      `enter` binding, where bubbles' name-based `key.Matches` does;
    * §2's "an upper-case letter `A` becomes `shift+a`" and "input is
      case-insensitive" disagree on `Ctrl+A`.
  * **The owner's choices,** each the recommendation, recorded as MADR A5:
    a `:release` or `:repeat` suffix; a named key matches by name; an
    upper-case letter means shift, with or without other modifiers.
  * **The others:** an `event` member on the binding, as opencode; equal
    codes only; a bare letter only.
  * **Also found, needing no choice:**
    * **Caps lock on the Windows console.** ultraviolet reports a letter
      typed with caps lock on in upper case, with `ModCapsLock`
      (`decoder.go`'s `parseWin32InputKeyEvent` and `ensureKeyCase`). §2's
      "matching ignores lock modifiers" is read to undo the lock's case
      too: with `ModCapsLock`, a letter compares in lower case, and shift
      still counts.
    * **The interop test's limit.** Under caps lock, a key's `String()`
      is its upper-case `Text` (Kitty's `parseKittyKeyboard` and the
      Windows path set it), which bubbles cannot tell from shift and a
      letter. So `key.Matches` and `Stroke.Matches` agree on every case
      without a lock modifier, and differ on a caps-lock letter, where the
      stroke is right; a test pins that difference rather than leaving
      the rows out.
    * **Scan strokes,** found by the interop test: bubbles compares a
      key's typed text, so on an AZERTY `a` at the US `q` it matches
      `[KeyA]` where `Matches` rightly matches `[KeyQ]`. The table pins
      this too, and `BubblesKeys`' comment names all three limits: a
      caps-lock letter, a shifted symbol written as shift and its key,
      and a scan stroke.

#### What was built

* **`keymap/notation.go` (new),** the package's first file, with its
  comment and stability line:
  * `Event` (`Press`, `Release`, `Repeat`), with its text forms through
    `enum.Names`;
  * `Stroke`, `Sequence`, `MaxStrokes`, `ParseStroke`, `ParseSequence`,
    `String`, `VSCode`, `Matches` and `BubblesKeys`, as §2 and A5 give
    them; `ErrNotation`, which every parse error wraps, naming the input
    and its bad part;
  * the key names: a list of Bubble Tea's key constants, the main keys
    before the keypad's, each named by Bubble Tea's own `Keystroke()`, so
    no name is copied; VS Code's `escape`, `pageup` and `pagedown`, and the
    aliases `option`, `cmd` and `win`; the scan codes `[KeyA]` to
    `[KeyZ]`, `[Digit0]` to `[Digit9]` and the US punctuation;
  * `Matches` ignores the lock modifiers and a letter caps lock made
    upper case (D1); a `Repeat` stroke matches only a repeated key; a
    release is the message's type, which `Matches` does not see, and
    Step 4's dispatcher checks it.
* **`keymap/notation_test.go` (new):**
  * `TestRoundTrips`: 5,000 random canonical strokes (any modifier set,
    named and F keys, printable runes, scan codes, any event) and
    sequences of up to four read back from both spellings;
  * `TestNames`: every listed key has a name of Bubble Tea's, and a name
    reads as the main key;
  * `TestNormalization`: A5's spellings as one stroke, and the refusals
    (a lock modifier, an empty stroke, no key, a modifier twice, an
    unknown key or scan code, five strokes, `<leader>` with no leader),
    each wrapping `ErrNotation` and naming its part;
  * `TestMatches` and `TestBubblesInterop`, over sixteen keys built as
    ultraviolet `v0.0.0-20260811164956-006e29f97886` decodes them, each
    citing its `decoder.go` lines: legacy upper case, a shifted symbol,
    `alt+a` and `esc`; Kitty `shift+a`, `shift+/`, `ctrl+a`, an AZERTY key
    with `BaseCode`, keypad Enter and 1, repeated and not, and caps lock;
    the Windows console under caps lock, shift and caps lock, and num
    lock. The agent read each cited range against the source;
  * `TestEventText`.
* **`internal/conformance/enums_test.go`:** `keymap.Event` joins the enum
  table, so `TestEveryEnumRoundTrips` and `TestOutOfRangeText` cover it.

#### Checks

* **Mutations,** on scratch copies, each killed by the test the PLAN
  names:
  * **K2-1** (lock modifiers not masked): the caps-lock rows of
    `TestMatches`;
  * **K2-2** (the `Text` comparison dropped): "Kitty shift+/ typing ?:
    \"?\".Matches = false";
  * **K2-3** (`VSCode` writes `super` as `meta`): `TestRoundTrips`, a
    stroke read back as `meta`;
  * **K2-4** (`BubblesKeys` drops the text): "legacy upper-case A:
    key.Matches(\"A\", [\"shift+a\"]) = false".

  They ran again after the lint fixes below, with the same results.
* **Lint fixes, found by the first gate run:** `goconst` on the modifier
  names, spelled in the table and the parse map, now built from the
  table; and `modernize`'s `strings.SplitSeq`.
* **macOS,** go1.27.2: `make pre-add-check FILES=<the 3 files>`, "3 file(s)
  clean"; `make lint`; the cross `go vet`; `-race`, `-shuffle=on
  -count=2`, `LC_ALL=C` and workspace mode; `go mod tidy -diff` (no
  requirement added); `make vuln`; `make apicheck`, "against v0.10.0, 0
  incompatible change(s)"; `make examples`; `scripts/go-modules.py
  --check`; `make release-check`, "200 file(s) clean", the tracked files,
  which the new ones join when committed; `shellcheck`,
  `markdownlint-cli2`, `actionlint` and the five Python tests. All clean.
* **The Windows test host,** go1.27.2 windows/amd64, with the index synced:
  `make pre-add-check`, "202 file(s) clean", with the new files; `make
  lint`, `make vuln` and `make examples`, each exit 0; the step's tests
  and the conformance checks, 10 tests passed, 0 failed; `keymap` under
  `-shuffle=on -count=2`, exit 0;
  `go-fuzz_test.py`, "27 passed"; `make fuzz FUZZTIME=3s`, "8 packages ran
  clean". A first run on the code before the lint fixes was stopped, its
  processes ended and its files removed.

### Step 3: rules, layers, contexts and conflicts

#### Deviations

* **D1 (2026-10-10): four points §3 to §6 leave open.**
  * **Found,** reading §3 to §6 and A4 against Step 2's code, before any
    of this step's code was written:
    * A4.3's `New(s Settings, layers ...[]Binding)` takes no `Default`,
      which §9's `DefaultRules() []keymap.Default` returns and whose
      fallback `Rebuild` must choose again;
    * §4's Prefix is "along one path", and `New` sees no path;
    * §5's features are empty until a `KeyboardEnhancementsMsg`, so a
      user's `shift+enter` is NotSupported at every start, and a keymap
      returned only without errors leaves nothing to `Rebuild`;
    * §6 expands `<leader>` when the keymap is built, but Step 2's
      `ParseSequence` refuses it with no leader
      (`keymap/notation.go`, its comment; `TestNormalization`'s row
      `"<leader> a"`), and Step 5's loader has no leader to give.
  * **The owner's choices,** each the recommendation, recorded as MADR
    A6: a `defaults` parameter; Prefix within one scope, with the
    innermost scope deciding along a path; the keymap with its errors;
    `<leader>` kept as a placeholder and expanded in `New`.
  * **The others:** `Needs` and `Fallback` on `Binding`, or a
    `Settings.Defaults` field; a Prefix error against Global, or a
    `Check(Path)` method; `nil` on any error, or no NotSupported while
    the features are empty; expanding in the loader.
  * **Decided in the step, needing no choice,** each read from the MADR:
    * `command.Global` is a scope like any other: a path lists it, as
      §9's `KeyPath` does, and `Lookup` walks only the scopes the path
      holds;
    * a binding needs `Disambiguate` for §5's strokes (`ctrl+i`,
      `ctrl+m`, `ctrl+[`, `shift+enter`, `alt+[`, `alt+]`),
      `EventTypes` for a release or repeat stroke, and `AlternateKeys`
      for a `Scan` stroke; a default needs its `Needs` and its keys'
      own;
    * Reserved: a user binding whose keys start with a reserved
      sequence, which the program's key could then never reach;
    * a removal with a `when` removes only bindings with that `when`
      text, and one without removes any, as VS Code's `-command` does; a
      removal with keys removes a default bound to them or to its
      fallback, whole;
    * a `when` is parsed by `New`, and a bad one is an error naming the
      binding's location;
    * `Resolved` is the command, its args and the binding that won.
    * `Settings` holds `Leader`, `Features` and `Reserved`; §3's
      `Timeout` and `Tick` come with Step 4's dispatcher, which reads
      them.

#### What was built

* **`keymap/keymap.go` (new):**
  * `Path`; `Layer` (`DefaultLayer`, `ProgramLayer`, `UserLayer`) and
    `Features` (`Disambiguate`, `EventTypes`, `AlternateKeys`,
    `AllKeysAsEscapes`, `AssociatedText`, each the Kitty flag of its bit),
    with their text forms through `enum.Names` and `enum.Bits`;
    `FeaturesFrom`;
  * `Binding`, `Location` with its `String`, `Default`, `Settings` and
    `Resolved`;
  * `New(s, defaults, layers...)`, which copies what it is given, expands
    `<leader>`, parses each `when`, applies the removals, chooses each
    default's keys, refuses Reserved and NotSupported bindings, and
    returns the keymap with every error joined; `Rebuild`;
  * `Lookup`, by scope and first stroke, with the scope deciding;
    `Bindings`, primary first; `Conflicts`; `Rules`.
* **`keymap/conflict.go` (new):** `ConflictKind` (`Duplicate`, `Prefix`,
  `Reserved`, `NotSupported`, `Shadow`) with its text forms;
  `BindingConflict`, an error naming the binding at fault by location,
  keys and scope; and the checks: Duplicate and Prefix within a scope, a
  later layer's Shadow, and a scope's Shadow of a global binding.
* **`keymap/notation.go`:** `ParseSequence(s, nil)` keeps `<leader>` as a
  placeholder stroke, which writes `<leader>` in both spellings and
  matches no key (A6.4).
* **Tests:** `TestResolution`, `TestScopeDecides`, `TestRemoval`,
  `TestFeatures`, `TestFeaturesFrom`, `TestLeader`, `TestBindingsOrder`,
  `TestRules`, `TestLocationText` and `TestImmutable` (`-race`, eight
  goroutines, and the caller's slices changed after `New`) in
  `keymap/keymap_test.go`; `TestConflicts`, every error kind planted in
  one keymap with file locations, and `TestConflictText` in
  `keymap/conflict_test.go`; `TestLeaderPlaceholder`, and
  `TestNormalization`'s `"<leader> a"` refusal row replaced by a
  five-stroke one, in `keymap/notation_test.go`.
* **`BenchmarkLookup`:** 200 bindings over a five-scope path, go1.27.2 on
  an Apple M1 Pro: 127.3 to 129.9 ns/op, 0 B/op, 0 allocs/op, over three
  runs.
* **`internal/conformance/enums_test.go`:** `keymap.ConflictKind`,
  `keymap.Features` and `keymap.Layer` join the enum table, and
  `keymap.Features` the bit sets.

#### Checks

* **Mutations,** on scratch copies, each killed by the test named:
  * **K3-1** (the path walked outermost first): `TestResolution`,
    "Lookup([global workspace pane:a], \"x\") = \"g.x\", want \"ws.x\"";
  * **K3-2** (`New` returns only the first error): `TestConflicts`, one
    error where four are wanted;
  * **K3-3** (the Prefix check skipped): `TestConflicts`, no `prefix
    ctrl+x`;
  * **K3-4** (reserved keys not checked for the user layer):
    `TestConflicts`, no Reserved error, `ctrl+c a` bound and the
    program's `ctrl+c` decided away;
  * **K3-5** (A6: an inner scope's longer or shorter binding does not
    decide): `TestScopeDecides`, `g`, `h h` and `j` resolving to global
    bindings;
  * **K3-6** (A6: `Rebuild` drops the layers): `TestFeatures`, "after
    Rebuild, Bindings(indent) = [], want ctrl+i";
  * **K3-7** (a removal of an unbound key accepted, a MADR Confirmation
    mutation): `TestRemoval`, "New: <nil>";
  * **K3-8** (`<leader>` with no leader accepted): `TestLeader`;
  * **K3-9** (`keymap.Layer` left out of the conformance table):
    `TestEveryEnumRoundTrips`, "keymap.Layer is an enum the table lacks".

  K3-5 to K3-9 are added to the four this step names. All nine ran again
  after the lint fixes below, with the same results.
* **Fixes, found by the checks:**
  * the first gate run's lint: `modernize`'s `slices.Backward` in
    `Bindings`; `gocritic`'s `evalOrder` on `return k, k.build()`; and
    `gosec`'s G115 on `FeaturesFrom`'s `int` to `uint8` conversion, now
    built bit by bit;
  * the benchmark's first run: its keys include `ctrl+i` and `ctrl+m`,
    which `New` rightly refused with no features; it now gives
    `Disambiguate`.
* **macOS,** go1.27.2: `make pre-add-check FILES=<the 7 files>`, "7
  file(s) clean"; `make lint`; the cross `go vet`; `-race`, `-shuffle=on
  -count=2`, `LC_ALL=C` and workspace mode; `go mod tidy -diff` (no
  requirement added); `make vuln`; `make apicheck`, "against v0.10.0, 0
  incompatible change(s)"; `make examples`; `scripts/go-modules.py
  --check`; `make release-check`, "202 file(s) clean", the tracked
  files, which the four new ones join when committed; `shellcheck`,
  `markdownlint-cli2`, `actionlint` and the five Python tests,
  `go-fuzz_test.py` once more with an empty module cache. All clean.
  Both records, under the repository's markdownlint rules, raise only
  MD004, the `*` list markers records use.
* **The Windows test host,** go1.27.2 windows/amd64, with the index synced:
  `make pre-add-check`, "206 file(s) clean", with the new files; `make
  lint`, `make vuln` and `make examples`, each exit 0; the step's tests
  and the conformance checks, 18 tests passed, 0 failed; `keymap` under
  `-shuffle=on -count=2`, exit 0;
  `go-fuzz_test.py`, "27 passed"; `make fuzz FUZZTIME=3s`, "8 packages
  ran clean". A first run, started before the benchmark's fix, was
  stopped, its processes ended and its files removed.

### Step 4: the matcher

No deviation. As A3 names them: `Dispatcher`, `NewDispatcher` and
`Outcome`, in `keymap/matcher.go`.

#### Readings

Where §6 leaves a detail open, the step reads it so, needing no choice:

* **A key that ends a sequence** without a match is taken afresh, after
  the sequence's keys go to `Replay`: `g` then `ctrl+p` gives the palette
  and replays `g`, and `g` then `ctrl+k` replays `g` and leaves `ctrl+k`
  pending. §6's "returns its strokes" is the sequence's, not the key that
  ended it.
* **A `TimeoutMsg`** is always `Handled`, current or old: it is the
  dispatcher's own message, which no pane wants.
* **A path change, or `SetKeymap`,** drops the pending keys with no
  `Replay`: the pane they were typed for no longer has the focus.
* **A release that matches nothing** leaves a pending sequence as it is,
  so the release of a chord's first key, under event types, does not end
  the chord (§6: a program "does not see each key twice").
* **A Prefix pair** kept in one scope (A6.3): the shorter fires at once,
  with no timeout fallback, as Q3 answered.
* **Matching** walks the deciding scope's bindings with `Stroke.Matches`,
  as `Lookup` walks strokes, so the keypad, caps lock and shifted symbols
  match here as Step 2's table says.
* **`Settings`** gains `Timeout` (zero means one second) and `Tick` (nil
  means `tea.Tick`); the keymap keeps its parsed leader, which the
  dispatcher leaves out of `Replay`.

#### What was built

* **`keymap/matcher.go` (new):** `Outcome` (`Command`, `Args`, `Handled`,
  `Replay`), `TimeoutMsg`, `Dispatcher` with `NewDispatcher`, `Update`,
  `SetKeymap` and `Pending`; and the keymap's `decide`, the scope
  deciding for keys.
* **`keymap/keymap.go`:** `Settings.Timeout` and `Settings.Tick`, and the
  keymap's leader.
* **`keymap/matcher_test.go` (new),** with a recording `Tick`:
  `TestSequencesComplete` (one stroke, a chord, a leader sequence, four
  strokes, one timer per pending stroke); `TestPrefixTimeout` (the
  timeout's `Replay`, the leader left out; `Settings.Timeout` and the
  one-second default); `TestStaleTimeout`; `TestPathChange`;
  `TestSetKeymap`; `TestRelease`; `TestUnmatched`;
  `TestDispatchScopeDecides` (A6, and keypad Enter completing
  `ctrl+k enter`); and `TestRealTimer`, which runs the default
  `tea.Tick` inside `testing/synctest.Test` and sees its `TimeoutMsg`
  after one second of the fake clock.

#### Checks

* **Mutations,** on scratch copies, each killed by the test named:
  * **K4-1** (the generation not compared): `TestStaleTimeout`, the first
    stroke's timer replaying `a b`;
  * **K4-2** (a path change does not cancel): `TestPathChange`, `x` on
    another path replaying `g`;
  * **K4-3** (the leader replayed): `TestPrefixTimeout`, "replay
    [ctrl+x]";
  * **K4-4** (`Replay` drops the first stroke): `TestPrefixTimeout` and
    `TestUnmatched`, `g` not replayed;
  * **K4-5** (a press stroke matches a release): `TestRelease`, a press
    of `space` firing `talk` and a release of `a` firing `cmd.a`;
  * **K4-6** (a release that matches nothing ends the sequence):
    `TestRelease`, `ctrl+x`'s release replaying it;
  * **K4-7** (an extending stroke starts no timer):
    `TestSequencesComplete`, "a b c d, stroke 1";
  * **K4-8** (the dispatcher walks the path outermost first):
    `TestDispatchScopeDecides`, global `g` firing under `pane:a`.

  K4-5 to K4-8 are added to the four this step names.
* **macOS,** go1.27.2: `make pre-add-check FILES=<the 3 files>`, "3
  file(s) clean"; `make lint`; the cross `go vet`; `-race`, `-shuffle=on
  -count=2`, `LC_ALL=C` and workspace mode; `go mod tidy -diff` (no
  requirement added); `make vuln`; `make apicheck`, "against v0.10.0, 0
  incompatible change(s)"; `make examples`; `scripts/go-modules.py
  --check`; `make release-check`, "206 file(s) clean", the
  tracked files, which the two new ones join when committed;
  `shellcheck`, `markdownlint-cli2`, `actionlint` and the five Python
  tests. All clean. This PLAN, under the repository's markdownlint
  rules, raises only MD004.
* **The Windows test host,** go1.27.2 windows/amd64, with the index synced:
  `make pre-add-check`, "208 file(s) clean", with the new
  files; `make lint`, `make vuln` and `make examples`, each exit 0; the
  step's tests and the conformance checks, 13 tests passed, 0
  failed, `TestRealTimer` among them; `keymap` under `-shuffle=on
  -count=2`, exit 0; `go-fuzz_test.py`, "27 passed"; `make fuzz
  FUZZTIME=3s`, "8 packages ran clean".

### Step 5: the VS Code loader, exporter and schema

No deviation. With A3's and A4's names and types, in `keymap/vscode.go`,
`keymap/jsonc.go` and `keymap/schema.go`:

* `LoadVSCode(r io.Reader, file string, ids func(command.ID) bool)
  ([]Binding, error)`, its errors each a `*LoadError`;
* `ExportVSCode(w io.Writer, bindings iter.Seq[Binding]) error`;
* `Schema(ids []command.ID, args func(command.ID) jsontext.Value) ([]byte,
  error)`, where §8 wrote `json.RawMessage` (A4.6).

#### Readings

Where §8 leaves a detail open, the step reads it so, needing no choice:

* **A loaded binding is the user layer.** Its `Location` is `UserLayer`,
  the file, the element's pointer, and its line and column, so Step 3's
  Reserved check applies to the user's file. A program that loads its
  own defaults from a file sets the layer it wants.
* **Members are walked, not decoded into a struct.** The loader reads
  each binding's members with `jsontext.Decoder`, so each error has its
  member's own place and pointer (`StackPointer`), and it refuses an
  unknown member itself. A struct decode with unknown members rejected
  stops at a binding's first error and gives no member's offset.
* **A syntax error ends the file there,** as `jsontext`'s errors are
  sticky; the bindings before it are returned. A member named twice is
  not a syntax error: the file is read allowing duplicate names, and each
  binding refuses its own, so one does not end the file. `args` is read
  strictly, and a name twice inside it is an error at that name.
* **A column counts bytes,** from 1, as `go/token`'s does.
* **`args` is kept compact,** so blanked comments leave no space in it,
  and an export reads back byte for byte.
* **An empty file,** or one of only space and comments, holds no bindings
  and no error. A leading byte order mark is blanked. `"when": ""` is no
  `when`. `"context": ""` is an error, as `"global"` names the global
  scope (A3).
* **The exporter** writes VS Code's spelling, indented with tabs as VS
  Code writes the file; it leaves out an empty `when` or `args` and a
  global `context`, and writes `<leader>` as `<leader>` (A6.4).
* **The schema** requires `command`, and `key` unless the command begins
  with `-`, and refuses other members. It drops an args schema's own
  `$schema`, which JSON Schema allows only at a document's root, lists a
  repeated ID once, and refuses a malformed ID and an args schema that is
  neither an object nor a boolean.
* **The key pattern** is built from Step 2's tables, in the syntax Go's
  `regexp` and ECMA-262 share. ECMA-262 has no flag for case inside a
  pattern, so each letter is a class of both cases. A one-character key
  is any character but space and the controls, so the pattern may be
  looser than `ParseStroke`, never stricter.
* **"The workspace defaults"** of the round-trip test are MADR §9's
  table with A4.1's commands, written as bindings: `workspace`'s own
  defaults arrive in Step 7.
* **No panic.** The loader records an error it cannot meet as an error.

#### What was built

* **`keymap/jsonc.go` (new):** `blank`, the scanner. Comments and
  trailing commas become spaces, line breaks stay, strings are copied,
  and an unterminated block comment is an error where it begins.
* **`keymap/vscode.go` (new):** `LoadError` (`File`, `Line`, `Col`,
  `Pointer`, `Err`, with `Error` and `Unwrap`), `LoadVSCode` and
  `ExportVSCode`.
* **`keymap/schema.go` (new):** `Schema` and the key pattern.
* **`keymap/vscode_test.go` (new):**
  * `TestBlank`; `TestJSONC`;
  * `TestErrorPlaces`, 18 cases, each place checked against a marker
    placed by hand: among them a bad key after a block comment over two
    lines, an unknown member after a line comment, a bad `when`, an
    unknown command with `ids` set, a name twice in `args`, a column after
    non-ASCII text, a missing comma, and an unterminated comment;
  * `TestThreeErrors`; `TestLoadRemoval`, `-command` with and without
    `key`, given to `New`; `TestLoadShapes`; `TestLoadErrorText`;
  * `TestExportRoundTrip`, for the workspace defaults and a planted user
    file with `<leader>`, upper case, a scan code, a release, nested
    `args`, removals and contexts; `TestExportText`.
* **`keymap/schema_test.go` (new):** `TestSchema` (draft 2020-12, the
  command enum, the key rule, an args schema from `command.SchemaOf` with
  its `$schema` dropped, and a boolean one); `TestSchemaErrors`;
  `TestKeyPattern` (Step 2's 5000 random sequences in both spellings,
  every key, alias, modifier and scan code in both cases, and twelve
  strings that are not keys refused).
* **`keymap/fuzz_test.go` (new):** `FuzzLoadVSCode`: no panic; each error
  a `*LoadError` at a line and column inside the input; what loads,
  exports and reads back unchanged; and the key pattern accepts any input
  `ParseSequence` reads. `make fuzz` finds it.
* **`keymap/testdata/fuzz/FuzzLoadVSCode/caeb09acd3ef6f1f` (new):** the
  input the fuzzer found, kept as `command`'s and `when`'s are.

#### Checks

* **Fuzz.** A first 60-second run was clean, but ran no input for its
  last 13 seconds. No input was shown to be slow, but reading the loader
  for a cost found `position` counting lines from the start for each
  error, quadratic in a file of many errors; it now searches a table of
  line starts built once. A second run found,
  in 7 seconds, that a name twice inside `args` passed the loader and
  failed the exporter; the loader now reads `args` strictly (K5-7), the
  input is kept, and a 90-second run is clean.
* **Scale.** A 600 KB file of 200,001 bad elements loads in about 74 ms.
* **Lint and `go fix`.** `make lint` found an unchecked
  `AppendUnquote`, a `!=` on an error in a test, repeated strings and an
  `append` to another slice; each is fixed in the code. `make
  modernize`'s `go fix` turned two `errors.As` into `errors.AsType`.
* **Mutations,** on scratch copies, each killed by the tests named:
  * **K5-1** (the scanner deletes a line comment, not blanking it) and
    **K5-2** (a block comment): `TestBlank`, `TestErrorPlaces`;
  * **K5-3** (a `//` inside a string is a comment): `TestBlank`,
    `TestJSONC`;
  * **K5-4** (errors stop at the first): `TestThreeErrors`;
  * **K5-5** (unknown members are accepted): `TestErrorPlaces`,
    `TestThreeErrors`;
  * **K5-6** (a trailing comma is kept): `TestBlank`, `TestJSONC`;
  * **K5-7** (a name twice inside `args` passes): `TestErrorPlaces`,
    "a member twice in args"; its first form left a variable unused and
    did not compile, and was rewritten;
  * **K5-8** (the args schema keeps its `$schema`): `TestSchema`;
  * **K5-9** (the key pattern is case-sensitive): `TestKeyPattern`;
  * **K5-10** (a loaded binding is not the user layer): `TestJSONC`;
  * **K5-11** (a removal needs a key): `TestLoadRemoval`,
    `TestExportRoundTrip`;
  * **K5-12** (`"global"` is not the global scope): `TestLoadShapes`;
  * **K5-13** (the exporter writes the canonical spelling):
    `TestExportText`;
  * **K5-14** (the line table is off by one at a line's start):
    `TestErrorPlaces`.

  K5-1 to K5-5 are the PLAN's four, its first split by the kind of
  comment; K5-6 to K5-14 are added.
* **macOS,** go1.27.2: `make pre-add-check FILES=<the 6 files>`, "6
  file(s) clean"; `make lint`; the cross `go vet`; `-race`, `-shuffle=on
  -count=2`, `LC_ALL=C` and workspace mode; `go mod tidy -diff` (no
  requirement added); `make vuln`; `make apicheck`, "against v0.10.0, 0
  incompatible change(s)"; `make examples`; `scripts/go-modules.py
  --check`; `make release-check`, "208 file(s) clean", the
  tracked files, which the new ones join when committed; `go-fuzz.py`
  on a scratch copy, finding `FuzzLoadVSCode`; `shellcheck`,
  `markdownlint-cli2`, `actionlint` and the five Python tests. All clean.
  This PLAN, under the repository's markdownlint rules, raises only
  MD004.
* **The Windows test host,** go1.27.2 windows/amd64, with the index
  synced: `make pre-add-check`, "214 file(s) clean", with the
  new files; `make lint`, `make vuln` and `make examples`, each exit 0;
  the step's tests, `FuzzLoadVSCode`'s seeds and the conformance checks,
  17 passed, 0 failed; `keymap` under `-shuffle=on -count=2`,
  exit 0; `go-fuzz_test.py`, "27 passed"; `make fuzz FUZZTIME=3s`, "9
  packages ran clean".

### Step 6: bubbles adapters and help

#### Deviations

* **D1 (2026-10-11): §7's method name, and what the full help lists.**
  * **Found,** reading §7 against A3 and Step 3's code, before any of
    this step's code was written:
    * §7's `func (k *Keymap) Binding(id command.ID, p Path, d Describer)
      key.Binding` takes the name A3 gave the type §3 called `Rule`, and
      Step 3 built `Bindings(id, p) []Sequence` beside it: one package,
      three meanings of the name (`AGENTS.md`, "API conventions", rule
      3);
    * §7's full help is "grouped by context, innermost first", but which
      commands it lists is not said, and Step 7's workspace help depends
      on it.
  * **The owner's choices,** each the recommendation, recorded as MADR
    A7: the method is `KeyBinding`; `Help` lists the IDs given, each
    once, in its primary sequence's scope's column, and with no IDs every
    command bound along the path.
  * **The others:** `BubblesBinding`, or keeping `Binding`; a full help of
    every command bound along the path, whatever the IDs.
  * **Files:** none added to the step's scope.

#### Readings

Where §7 and A7 leave a detail open, the step reads it so, needing no
choice:

* **A release or repeat stroke** is listed in `KeyBinding` as its full
  text, as a chord is. A `key.Binding` cannot tell either from a press,
  so its `BubblesKeys` would match where the dispatcher does not.
* **`KeyBinding` agrees with the dispatcher** wherever Step 2's
  `BubblesKeys` agrees with `Matches`. The differences Step 2's D1 pinned
  (a caps-lock letter, a shifted symbol written as shift and its key, a
  scan stroke) carry over, and the test pins each.
* **A `key.Binding` has no `when`,** and §7's signature takes no
  context, so a binding with a `when` matches whether it holds or not.
* **A title** is the `Describer`'s, or the ID when it gives none or is
  nil. `FromRegistry` gives no title for a command the registry does not
  hold, or one whose `Title` is empty.
* **`Help`** is built once, as the keymap does not change, and
  `ShortHelp` and `FullHelp` return copies. A path naming a scope twice
  gives that scope one column.
* **`Bindings`** now reads an internal list of the entries it shows, so
  `Help` knows the scope of each primary sequence; what it returns is
  unchanged, and Step 3's tests pass as before.
* **The golden help model** is drawn as `workspace.Help` draws it: the
  glyph set's `Bullet` and `Ellipsis`, and the theme's styles.

#### What was built

* **`keymap/help.go` (new):** `Describer`, `DescriberFunc`,
  `FromRegistry`, `KeyBinding` and `Help`.
* **`keymap/keymap.go`:** `Bindings` over the new `shown`.
* **`keymap/help_test.go` (new):**
  * `TestKeyBindingMatches`: over Step 2's table of keys as ultraviolet
    decodes them, each stroke bound alone, `key.Matches` on its
    `KeyBinding` against the table's bubbles column, a release or repeat
    stroke never matching, and the dispatcher against its `Matches`
    column; `shift+a` gives `shift+a` and `A`;
  * `TestKeyBindingScopeDecides`: a binding an inner scope hides is
    disabled, as the dispatcher does not run it;
  * `TestKeyBindingChord`: a chord and a release stroke are enabled,
    listed as their text, and never matched; the help key is the
    primary sequence; help shows the chord;
  * `TestKeyBindingDisabled`: an unbound command, and one the user's file
    removes, give a disabled binding, which the short and full help
    leave out;
  * `TestDescriber`: a planted `DescriberFunc`, no `Describer`, and
    `FromRegistry` over a registry with a titled and an untitled command;
  * `TestHelpColumns`: the short help in the order given; the full help
    innermost first, a command bound in two scopes once, in its primary's
    column; every command with no IDs; a path naming global twice; and
    copies returned;
  * `TestHelpGolden`: the short and full help through `help.Model` across
    the 0001-MADR §6 matrix at 40 and 80 cells.
* **`keymap/testdata/golden/` (new):** the 16 golden files, written with
  `-tuitest.update` and each read: three columns innermost first; at 40
  cells both helps end in the case's ellipsis, `…` or `~`; an ASCII
  case's separator `*`. A script checked that each line, its escapes
  stripped, is no wider than its case and as wide as its annotation, and
  that an ASCII case holds only ASCII; on a copy with a `•` planted in an
  ASCII file, it failed naming that file.

#### Checks

* **A guessed number dropped.** A first draft of `TestKeyBindingMatches`
  also asserted at least 25 agreeing cases, a number not derived from the
  table, and failed at 23. Each case is pinned already, so the count was
  removed, not tuned.
* **Mutations,** on scratch copies, each killed by the tests named:
  * **K6-1** (the chord's full text is its first stroke):
    `TestKeyBindingChord`;
  * **K6-2** (`Help` groups scopes outermost first): `TestHelpColumns`,
    `TestHelpGolden`;
  * **K6-3** (a release or repeat stroke is spelled as a press):
    `TestKeyBindingMatches`, `TestKeyBindingChord`;
  * **K6-4** (an unbound command's binding is enabled):
    `TestKeyBindingDisabled`;
  * **K6-5** (the IDs given are not the only ones listed):
    `TestHelpColumns`;
  * **K6-6** (the help key is the last sequence, not the primary):
    `TestKeyBindingChord`;
  * **K6-7** (the `Describer` is not asked): `TestDescriber`;
  * **K6-8** (`FromRegistry` gives an empty title): `TestDescriber`;
  * **K6-9** (a command is listed in its last scope's column):
    `TestHelpColumns`;
  * **K6-10** (the full help is shared with the caller):
    `TestHelpColumns`.

  K6-1 and K6-2 are the PLAN's two; K6-3 to K6-10 are added.
* **macOS,** go1.27.2: `make pre-add-check FILES=<the 3 files>`, "3
  file(s) clean"; `make lint`; the cross `go vet`; `-race`, `-shuffle=on
  -count=2`, `LC_ALL=C` and workspace mode; `go mod tidy -diff` (no
  requirement added: bubbles is one); `make vuln`; `make apicheck`,
  "against v0.10.0, 0 incompatible change(s)"; `make examples`;
  `scripts/go-modules.py --check`; `make release-check`,
  "214 file(s) clean", the tracked files, which the new ones
  join when committed; `shellcheck`, `markdownlint-cli2`, `actionlint`
  and the five Python tests. All clean. `gofmt` first wrapped one line of
  the new test. This PLAN and the MADR, under the repository's
  markdownlint rules, raise only MD004.
* **The Windows test host,** go1.27.2 windows/amd64, with the index
  synced: `make pre-add-check`, "216 file(s) clean", with the
  new files; `make lint`, `make vuln` and `make examples`, each exit 0;
  the step's tests and the conformance checks, 11 passed, 0
  failed, `TestHelpGolden` among them; `keymap` under `-shuffle=on
  -count=2`, exit 0; `go-fuzz_test.py`, "27 passed"; `make fuzz
  FUZZTIME=3s`, "9 packages ran clean".

### Step 9 (A1): normalisation, labels and aliases

#### Deviations

* **D1 (2026-10-11): the key glyphs.**
  * **Found,** reading `glyph` against A1 and A2, before any of this
    step's code was written:
    * `glyph`'s package comment, and `TestEveryGlyphIsOneCell` and
      `TestTierWidths`, which walk every field of `Set`, hold each glyph
      and its ASCII twin to one cell; A1's ASCII twins are words (`opt+`,
      `cmd+`, `up`), and the step's 2026-10-10 note asked for both;
    * the type name of the option chosen, `Keys`, is `when.Keys`'s, and
      `internal/conformance` fails on a type name two public packages
      export.
  * **The owner's choices,** each the recommendation, recorded as MADR
    A8: `glyph.Set` gains `Keys`, a `KeyGlyphs`, with A1's words as its
    ASCII twins, and `glyph`'s tests check `Keys` apart; the type is
    `KeyGlyphs`.
  * **The others:** one-cell ASCII twins, such as `^` for `⌃`; a separate
    `KeyLabels` type; sharing the name `Keys` with `when`.
  * **Files added to the step's scope:** `docs/architecture.md`, whose
    `glyph` row said every glyph is one cell.

#### Readings

Where A1 leaves a detail open, the step reads it so, needing no choice:

* **Ultraviolet normalises most of it already.** It reads a C0 byte as
  ctrl and a letter, NUL as ctrl+space, BS as ctrl+h and DEL as
  backspace (`decoder.go`'s `parseControl`, :1128-1165), and Kitty's C0
  codes alike (:1318-1330). `Normalize`'s C0 and BS rules act on a key
  from another source, and the test shows both: ultraviolet's form
  unchanged, the raw code changed.
* **BS.** Ultraviolet's ctrl+h for the byte 0x08 cannot be told from
  ctrl+h, so `Normalize` leaves ctrl+h; "a raw BS" is a key whose code is
  0x08.
* **NUL** is ctrl+space, as ultraviolet reads it by default. A Windows
  console key with no code and its rune in `BaseCode` (:1856-1860) is not
  NUL.
* **An upper-case letter with ctrl and no shift** comes from
  ultraviolet's C1 path (:290-295) and from modifyOtherKeys (:1190-1196).
  A C1 byte is read as ctrl+alt and an upper-case letter, so 0x81 becomes
  ctrl+alt+shift+a, as A1's rule says, though an 8-bit terminal sends the
  same byte for ctrl+alt+a.
* **ctrl+5 and ctrl+4** keep any other modifier: ctrl+alt+5 is
  ctrl+alt+], as a legacy terminal sends it.
* **AltGr.** `Normalize` gives the text with no modifier, a lock and
  `BaseCode` kept. The dispatcher matches no binding with it, so it goes
  to the pane, and it ends a pending sequence as any unmatched key does.
  In ultraviolet only AltGr gives ctrl and alt with text
  (`parseWin32InputKeyEvent`, :2025-2045).
* **super** drops meta and hyper from the key, so a stroke naming super
  with meta or hyper does not match.
* **The dispatcher** reads each key through `Normalize` with the keymap's
  features, so the legacy rules apply until a `KeyboardEnhancementsMsg`
  and `Rebuild` give `Disambiguate`; `Replay` keeps the keys as they
  arrived.
* **Labels.** A1's macOS order, ⌃ ⌥ ⇧ ⌘, is the canonical order; meta
  and hyper, with no glyph, are words; letters stay lower case, as A8's
  example `⌃⇧k`; an event and a scan code keep their canonical text; and
  the placeholder is `<leader>`, while a keymap's bindings are expanded.
* **`DisplayAliases`** holds `pgdn` and `del`: Bubble Tea's own `pgup`
  is short already.
* **The tables** `ParseAliases`, `DisplayAliases` and `CopyStrokes` are
  functions returning a new map or slice, read-only as A1 asks.
* **The schema's key pattern** reads the parse aliases, so it accepts
  `return` and `del`.

#### What was built

* **`glyph/glyph.go`:** `KeyGlyphs`, `Set.Keys`, both tables, and the
  package comment's exception.
* **`keymap/normalize.go` (new):** `Normalize`.
* **`keymap/label.go` (new):** `ParseAliases`, `DisplayAliases`,
  `CopyStrokes`, `Stroke.Label` and `Sequence.Label`.
* **`keymap/notation.go`:** `ParseStroke` reads the parse aliases, and
  two names the tables share are constants.
* **`keymap/keymap.go`:** `Settings.GOOS`.
* **`keymap/matcher.go`:** the dispatcher normalises, matches no AltGr
  text, and replays keys as they arrived.
* **`keymap/schema.go`:** the key pattern with the aliases.
* **`docs/architecture.md`:** the `glyph` row.
* **Tests:**
  * `glyph/glyph_test.go`: the one-cell walks leave out `Keys`, and
    `TestKeyGlyphs` checks it in every tier;
  * `keymap/normalize_test.go` (new): `TestNormalize`, 31 keys each with
    its source and decoder lines, on a legacy terminal and with
    `Disambiguate`; `TestDispatchNormalizes`; `TestAltGr`;
  * `keymap/label_test.go` (new): `TestLabelGolden`, 29 sequences on
    `darwin`, `linux` and `windows` in UTF-8 and ASCII; `TestLabelASCII`;
    `TestLeaderLabel`; `TestAliases`; `TestCopyStrokes`.
* **`keymap/testdata/golden/` (6 new files):** the labels, written with
  `-tuitest.update` and each read: macOS draws `⌃⌥⇧⌘` in UTF-8 and
  `ctrl+opt+shift+cmd+` in ASCII, the others the canonical words, and
  every system `↑` or `up`.

#### Checks

* **A gap the mutations found.** K9-5 survived at first: no case had a C0
  code with a modifier. "a raw 0x01 with alt" was added, and K9-5 is
  killed.
* **A wrong comment.** `ParseAliases` first said no spelling writes an
  alias; `VSCode` writes VS Code's own, and the comment says so.
* **Lint.** `goconst` found `super` and `pgdown` three times each; they
  are constants the tables share.
* **Mutations,** on scratch copies, each killed by the tests named:
  * **K9-1** (ESC is mapped to ctrl+[): `TestNormalize`;
  * **K9-2** (normalisation runs with `Disambiguate` set):
    `TestNormalize`;
  * **K9-3** (AltGr text is matched as a binding): `TestAltGr`;
  * **K9-4** (`Label` ignores the glyph table's ASCII twin):
    `TestLabelGolden`, `TestLabelASCII`;
  * **K9-5** (a C0 code with a modifier is changed): `TestNormalize`;
  * **K9-6** (an upper-case letter with ctrl and shift is changed):
    `TestNormalize`;
  * **K9-7** (super keeps stray meta and hyper): `TestNormalize`;
  * **K9-8** (AltGr on every system): `TestNormalize`, `TestAltGr`;
  * **K9-9** (`Replay` gives the normalised key):
    `TestDispatchNormalizes`;
  * **K9-10** (the dispatcher matches the raw key):
    `TestDispatchNormalizes`;
  * **K9-11** (a parse alias is not read): `TestAliases`;
  * **K9-12** (`Label` writes the canonical name for a display alias):
    `TestLabelGolden`;
  * **K9-13** (darwin's glyphs on every system): `TestLabelGolden`;
  * **K9-14** (an ASCII key glyph is not ASCII): `TestASCIISetIsASCII`,
    `TestKeyGlyphs`;
  * **K9-15** (a Unicode key glyph is a word): `TestKeyGlyphs`.

  K9-1 to K9-4 are the PLAN's four; K9-5 to K9-15 are added.
* **macOS,** go1.27.2: `make pre-add-check FILES=<the 10 files>`, "10
  file(s) clean"; `make lint`; the cross `go vet`; `-race`, `-shuffle=on
  -count=2`, `LC_ALL=C` and workspace mode; `go mod tidy -diff` (no
  requirement added); `make vuln`; `make apicheck`, "against v0.10.0, 0
  incompatible change(s)", `Set.Keys` a compatible addition; `make
  examples`; `scripts/go-modules.py --check`; `make release-check`,
  "216 file(s) clean", the tracked files, which the new ones
  join when committed; `shellcheck`, `markdownlint-cli2`, `actionlint`
  and the five Python tests. All clean. This PLAN and the MADR, under
  the repository's markdownlint rules, raise only MD004.
* **The Windows test host,** go1.27.2 windows/amd64, with the index
  synced: `make pre-add-check`, "220 file(s) clean", with the
  new files; `make lint`, `make vuln` and `make examples`, each exit 0;
  the step's tests in `keymap` and `glyph` and the conformance checks,
  16 passed, 0 failed; `keymap` under `-shuffle=on -count=2`,
  exit 0, where `Settings.GOOS` left empty is `windows`;
  `go-fuzz_test.py`, "27 passed"; `make fuzz FUZZTIME=3s`, "9 packages
  ran clean".
