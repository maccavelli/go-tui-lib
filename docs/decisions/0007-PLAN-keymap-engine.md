---
status: proposed
date: 2026-10-02
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

None yet.
