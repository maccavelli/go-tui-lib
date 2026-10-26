---
status: in-progress
date: 2026-10-26
associated-madr: "0024-MADR-opt-in-and-opt-out-coverage.md"
---
# Implement API rule 12, its floors, the message catalogue and the defaults inventory, and close the command-and-control gaps

Associated MADR: [0024-MADR-opt-in-and-opt-out-coverage.md](0024-MADR-opt-in-and-opt-out-coverage.md)

This is the first of the MADR's two PLANs, by its answer to Q5: D1 to D4,
and D9's `msgcat` package, which
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
needs before its components draw text. D5 to D7, and the catalogue's
adoption by the existing packages, are
[0024-PLAN-terminal-text-and-convention-opt-outs.md](0024-PLAN-terminal-text-and-convention-opt-outs.md).

## Goal

A program with its own CLI can run the workspace's commands through its
own registry, set its own permission policy, forget a remembered verdict,
leave out the registry's built-in commands, make `launch`'s decision its
own way with the library's helpers, choose where the epilogue goes or
whether it is written, and bind the palette's commands under its own
names; AGENTS.md carries rule 12 and its floors; `docs/guides/defaults.md`
lists every default with its control, and a test holds it to the code;
and `msgcat` is built.

Done means every item under Verification holds, CI is green on the pushed
tree, `v2.2.0` is tagged by the owner, and the consumer smoke test passes.

## Scope

### Facts re-read before execution (2026-10-25, at `65d437b`)

| Fact | Step |
| :--- | :--- |
| `workspace.(*Workspace).run` runs a `workspace.*` command's handler with `context.Background()`; any other ID goes to a `CommandConsumer`, else back as `RequestMsg` (`workspace/bindings.go:253-274`) | 5 |
| `command.asks` is the fixed table: mutating asks for the agent, destructive for the agent or the shell (`command/gate.go:78-89`); `policy.decide` consults `Request.Gate` only where it asks, and remembers `AllowAlways` and `RejectAlways` by ID and caller (`:104-134`) | 4 |
| `NewRegistry` adds `command.list`, `command.describe` and `app.quit` (`command/registry.go:157-174`), whose IDs are unexported (`builtin.go:11-15`) | 4 |
| `Decide` reads `CI` by name only when `Choice` is not `ChoiceTUI`, and `TERM=dumb` is plain under every choice (`launch/decide.go:205-218`); `colourEnv` and `profile` are unexported (`launch/colour.go:19`, `:38`), and so is the locale's tier (`locale.go:18-31`) | 6 |
| `Run` writes `Epilogue()` to `Out` on a clean end (`launch/run.go:131-137`) | 6 |
| The picker's command IDs and its scope are unexported (`palette/keys.go:12-29`); the opening hook is the unexported `opener` (`palette/picker.go:96-100`) | 7 |
| The cores are `internal/conformance/charmfree_test.go`'s `cores` and `.golangci.yml`'s `charmfree` files; the narrow closures are `narrow`; shared type names are `conformance_test.go`'s `sharedNames` | 2 |

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0023-*`, `docs/decisions/0024-*`, `docs/README.md`, `docs/glossary.md`, `AGENTS.md` | accept both MADRs; rule 12; this PLAN `in-progress` |
| 2 | `msgcat/`, `internal/conformance/charmfree_test.go`, `.golangci.yml` | the catalogue package, a core with a narrow closure |
| 3 | `docs/guides/defaults.md`, `internal/conformance/defaults_test.go` | the inventory and `TestDefaultsInventory` |
| 4 | `command/`, `docs/guides/defaults.md` (D1 of Step 4) | `WithGatePolicy`, `GatePolicy`, `GatePolicyFunc`, `DefaultGatePolicy`, `Registry.Forget`, `ForgetAll`, `WithoutBuiltins`, `BuiltinList`, `BuiltinDescribe`, `BuiltinQuit` |
| 5 | `workspace/`, `docs/guides/defaults.md` (D1 of Step 4) | `WithoutOwnCommands` |
| 6 | `launch/`, `docs/guides/defaults.md` (D1 of Step 4) | `Config.AllowDumb`, `Config.IgnoreCI`, `ColorProfile`, `GlyphTier`, `WithoutEpilogue`, `WithEpilogueWriter` |
| 7 | `palette/`, `docs/guides/defaults.md` (D1 of Step 4) | the picker's command IDs and scope exported, `CompletionRules`, `Opener` |
| 8 | `docs/`, `README.md` | the guides, the release `v2.2.0`, the close-out |

No module is added. `msgcat` imports the standard library alone.

### Out of scope

* MADR D5 to D7, and each package's adoption of the catalogue (its
  `Messages()`, `WithCatalog` and IDs): the second PLAN.
* 0023's components: their PLAN.
* `git push` and tags, which are the owner's.

## Prerequisites

* The owner has accepted the MADR and approved this PLAN.
* `v2.1.0` is the latest tag, and `make apicheck` compares with it.

## Rules for every step

0008-PLAN's rules, which every recent PLAN has run under:

1. **Order.** Each step's packages compile, pass their tests and pass the
   gates before the next starts.
2. **Every default unchanged.** Each step adds a test that the behaviour
   without its new option is today's, and one that the option changes it.
3. **Mutation proofs,** through `python3 -m tools.verify mutate` on
   scratch copies, each making a named test fail, each failure quoted in
   the record.
4. **Gates:** `python3 -m tools.verify gates` on macOS and through
   `remote linux` and `remote windows`; `make apicheck` reports 0
   incompatible changes against `v2.1.0`.
5. **Names:** each new type name through `python3 -m tools.verify names`
   before it is written.
6. **Identifiers:** the scan of the step's diff before each handoff.
7. **Commit.** One commit per step, by the owner, after the agent stops
   with the step's evidence in the execution record; the agent stages and
   commits nothing.
8. **Deviations** stop the work, with resolutions offered; the owner's
   choice is recorded here, dated, and in the MADR when a decision changes.

## Implementation Steps

### Step 1: records

* Both MADRs `accepted`, with the owner's words; this PLAN `in-progress`;
  `docs/README.md`'s rows.
* [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)
  D12 gains a dated note: its releases are each the next minor version
  after this PLAN's `v2.2.0`, by 0024-MADR Q5, so D12's `v2.2.0` to
  `v2.5.0` read as the next four minors.
* AGENTS.md's "API conventions" gains rule 12, as 0024-MADR D1 words it,
  naming `docs/guides/defaults.md`.
* `docs/glossary.md`: `msgcat`'s `Catalog`, `CatalogFunc`, `MessageID`
  and `Table`; `command.GatePolicy` and `GatePolicyFunc`; `palette.Opener`.
* Records and AGENTS.md only.

### Step 2: `msgcat`

* `msgcat/`: `MessageID`, `Catalog`, `CatalogFunc`, `Table` (a `Catalog`),
  `Format(c Catalog, id MessageID, def string, args ...any) string` (a nil
  catalogue, or a false answer, is `def`), and `Check(defaults Table, c
  Catalog) error`, which reports each message whose fmt verbs differ from
  its default's, joined; its documentation with the stability line.
* **The gates:** `msgcat` joins `cores`, the `charmfree` files, and
  `narrow` with an empty allow list: the standard library alone.
* **Tests:** a default kept for a nil catalogue and for a missing ID; a
  translation used; explicit argument indexes reorder; `Check` names a
  translation with a missing verb, an extra one and another verb, and
  passes a matching one; `CatalogFunc` and `Table` are `Catalog`s.
* **Mutations:** `Format` ignores the catalogue; `Format` ignores a false
  answer and prints an empty string; `Check` compares the count of verbs
  only; `msgcat` imports `x/ansi`, which `TestNarrowClosures` and the
  `charmfree` rule's neighbour fail.

### Step 3: the inventory

* `docs/guides/defaults.md`: what rule 12 asks; the floors of MADR D2,
  each with its reason and its lower layer; then a table per package,
  `Behaviour | Default | Control | Status`, from the MADR's audit: every
  behaviour, its control in backticks (`workspace.WithoutMouse`,
  `command.Registry.Remove`), and a status of `built`, `planned` (with the
  PLAN that builds it) or `floor`.
* `internal/conformance/defaults_test.go`, `TestDefaultsInventory`: it
  reads the guide's tables, and resolves every control in a `built` row,
  `pkg.Name` or `pkg.Type.Method`, in the type-checked packages; one that
  does not resolve fails, naming the row. A `planned` row must name its
  PLAN by file name.
* **Tests and proofs:** on a scratch copy, a `built` row naming
  `workspace.WithoutMice` fails; a `planned` row with no PLAN fails; a
  method that does not exist fails.

### Step 4: `command`

* `GatePolicy interface{ Asks(c Command, o Origin) bool }`,
  `GatePolicyFunc`, `DefaultGatePolicy()`, today's `asks` table, and the
  option `WithGatePolicy(GatePolicy)`; a nil policy keeps the default.
  `policy.decide` asks the registry's policy where it asked `asks`.
* `Registry.Forget(id ID, caller string)` and `Registry.ForgetAll()` drop
  remembered verdicts.
* `WithoutBuiltins()`: `NewRegistry` adds none of its three commands.
  `BuiltinList`, `BuiltinDescribe` and `BuiltinQuit`, exported `ID`
  constants; the unexported names become those.
* **Tests:**
  * without the option, every danger and origin pair asks as today: a
    table test over all four dangers and seven origins;
  * a policy that asks for every destructive command puts a destructive
    key request to the gate; one that asks for nothing lets the agent's
    mutating request run unasked;
  * `AllowAlways`, then `Forget`, asks again; `ForgetAll` forgets every
    caller's;
  * `WithoutBuiltins`: `All`, `MCPTools`, `Manifest` and `Complete` hold
    none of the three; without it, all three are there;
  * the exported IDs equal the commands' IDs.
* **Mutations:** `decide` ignores the policy; `DefaultGatePolicy` asks for
  mutating key requests; `Forget` forgets nothing; `WithoutBuiltins` is
  ignored; `ForgetAll` forgets only the first caller.

### Step 5: `workspace`

* `WithoutOwnCommands()`: `run` treats a `workspace.*` command as any
  other: offered to a `CommandConsumer`, else a `RequestMsg`. The esc
  binding's offer to an `EscConsumer` stays first. A program that wants
  the commands registers `workspace.Commands(ws)` in its registry.
* **Tests:**
  * without the option, a `workspace.focus.next` key moves focus with no
    `RequestMsg`, as today;
  * with it, the key comes back as a `RequestMsg` with `OriginKey` and
    the workspace's when values; dispatched to a registry holding
    `workspace.Commands(ws)`, whose gate records what it is asked and
    whose auditor records every request, focus moves, the auditor holds
    the request, and a policy that asks for every UI command puts it to
    the gate;
  * an overlay that consumes esc still takes it under the option.
* **Mutations:** the option is ignored; the option drops the esc
  consumer's offer; the request loses its when values.

### Step 6: `launch`

* `Config.AllowDumb`: `TERM=dumb` no longer forces plain output; the rest
  of the decision runs. `Config.IgnoreCI`: `CI` no longer vetoes the
  automatic choice. Both false by default.
* `ColorProfile(s Streams) colorprofile.Profile` and `GlyphTier(s Streams)
  glyph.Tier`, the helpers `Decide` uses, exported; `Decide` calls them.
* `WithoutEpilogue()` writes no epilogue; `WithEpilogueWriter(w
  io.Writer)` writes it to `w`. `Run`'s default is today's.
* **Tests:** `Decide`'s reasons without the fields, as today; with
  `AllowDumb` and a terminal, a TUI decision; with `IgnoreCI` and `CI=1`,
  the automatic choice decides by the terminal; `ColorProfile` and
  `GlyphTier` equal what `Decide` puts in its decision, for every fixture
  `launchtest` has; the epilogue to `Out` by default, to the given writer,
  or nowhere.
* **Mutations:** `AllowDumb` ignored; `IgnoreCI` also drops the program's
  `NoInputEnv`; `Decide` stops calling the exported helpers and keeps a
  copy that drifts (a fixture that differs); `WithoutEpilogue` ignored.
* **The examples gate** (`make examples`) still passes: the framework
  examples use none of these.

### Step 7: `palette`

* The picker's command IDs exported: `OpenID`, `SelectNextID`,
  `SelectPrevID`, `PageNextID`, `PagePrevID`, `AcceptID`, `CompleteID`;
  its scope, `PickerScope`. `DefaultRules` reads them.
* `CompletionRules(id string) []keymap.Default`: the completion bindings
  in `overlay:<id>`, so a pop-up under another ID has its keys.
  `DefaultRules` uses it for `completion`.
* `Opener interface{ Opened(c when.Context) }`, exported: `Open` calls it
  on each provider that has it, as it calls `opener` today; `FromRegistry`,
  `Panes` and `Recents` implement it.
* **Tests:** `DefaultRules` unchanged, compared binding for binding with
  `v2.1.0`'s; `CompletionRules("notes")` binds in `overlay:notes`, and a
  pop-up opened `WithID("notes")` moves and accepts with them; a program's
  own wrapper of `FromRegistry` that forwards `Opened` lists the commands
  for an empty query, and one that does not lists none.
* **Mutations:** `CompletionRules` ignores its ID; `Open` stops calling
  `Opened`; `Recents` stops forwarding it.

### Step 8: documentation, the release `v2.2.0`, and the close-out

* **Guides:** `docs/guides/commands.md` gains the gate policy, forgetting,
  and a registry without built-ins; `building-workspaces.md`
  `WithoutOwnCommands`; `command-palette.md` the exported IDs,
  `CompletionRules` and `Opener`; the commands guide's launch section the
  new `Config` fields, the helpers and the epilogue options; the defaults
  guide's rows for Steps 4 to 7 move to `built`. *Amended 2026-10-26, by
  Step 4's deviation D1:* each of Steps 4 to 7 moves its own rows, and
  Step 8 checks that no row names this PLAN as `planned`.
* `docs/guides/migrating.md`: "Migrating to `v2.2.0`", additions only.
* `README.md`: the current release, and a Status bullet.
* **The release,** as 0008-PLAN Step 9 ran: the API listed by apidiff
  with compatible changes; release notes in the execution record; the
  disclosure guard; the owner's push; CI green; the owner's annotated tag;
  CI green on the tag; the consumer smoke test from the proxy, a program
  with its own registry, policy and `WithoutBuiltins`, a workspace
  `WithoutOwnCommands`, and its own launch decision, with a negative
  control seen to fail.
* This PLAN becomes `complete`, and its `docs/README.md` row says so.

## Verification

* Every step's mutations are killed, each failure quoted in the record.
* `python3 -m tools.verify gates` passes on macOS, Linux and Windows for
  every step.
* `make apicheck` reports 0 incompatible changes against `v2.1.0`, and
  `scripts/apicheck.allow` stays empty.
* `TestDefaultsInventory` passes, and every behaviour of the MADR's audit
  has a row.
* `TestCoresAreCharmFree` and `TestNarrowClosures` pass with `msgcat`.
* Without any new option, every existing test passes unchanged.
* The identifier scan finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1 to 8 and tags `v2.2.0`.
  [0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
  starts after this PLAN's close-out, by 0024-MADR Q5, and its first
  release follows this one.
* **Rollback.** Before the push, each step is one local commit. After it,
  a patch release fixes forward. Every change is an addition, and no
  default changes, so a program that sets no new option sees no change.

## Execution Record

### Step 1: records

*2026-10-25.* The owner committed the three PLANs and approved this one
("committed, proceed").

* **Both MADRs are `accepted`,** each with a dated line naming the
  owner's words, above its record of the answers.
* **0023-MADR D12** has a dated note: its four releases follow this
  PLAN's `v2.2.0`, each the next minor version.
* **AGENTS.md's "API conventions"** gains rule 12, "Defaults are the
  program's to refuse", worded as 0024-MADR D1, naming
  `docs/guides/defaults.md`, which Step 3 writes.
* **`docs/glossary.md`'s "Reserved by accepted records"** lists the type
  names both MADRs take before they are built: 0023's, by package, and
  0024's `msgcat.Catalog`, `CatalogFunc`, `MessageID` and `Table`,
  `command.GatePolicy` and `GatePolicyFunc`, and `palette.Opener`. Each
  passed `python3 -m tools.verify names` when its record was written.
* **`docs/README.md`:** both MADRs `accepted`, this PLAN `in-progress`.
* Records and AGENTS.md only: no source, test, CI or configuration.

### Step 2: `msgcat`

*2026-10-25.* The owner committed Step 1 and said to proceed.

* **`msgcat/`:** `MessageID`; `Catalog`, with `CatalogFunc` and `Table`
  (a nil `Table` keeps every default); `Format`, which takes the
  catalogue's text or the default and always formats it with
  `fmt.Sprintf`, so a literal percent sign is `%%`; and `Check`, which
  compares each translation's verbs with its default's, by argument
  number and verb letter, so explicit indexes may reorder the arguments,
  and joins one error per message, in ID order. Its documentation ends
  with the stability line.
* **The gates:** `msgcat` joins `cores` (`TestCoresAreCharmFree`), the
  `charmfree` depguard files, and `narrow` with no allowed package
  (`TestNarrowClosures`); depguard gains `msgcatstd`, strict, allowing
  `$gostd` alone, beside `termkeystd` and `safetextstd`.
* **`docs/glossary.md`:** `msgcat`'s four names move from "Reserved" to
  "Built".
* **Tests** (`msgcat/msgcat_test.go`): `TestFormat` (the default for a nil
  catalogue, a nil `Table` and an ID it lacks; a translation that
  reorders; `CatalogFunc` both ways; a literal percent); `TestCheck`
  (matching translations, among them a reorder and a `*` width, pass; a
  dropped verb, an added one, a verb where the default has none and
  another verb are each reported by ID; an empty catalogue passes);
  `TestVerbs` (eight formats, and fmt's own error marker agreeing on how
  many integers four formats read); `ExampleFormat`.
* **Mutations,** through `python3 -m tools.verify mutate` on scratch
  copies, 7 killed of 7:

  | Mutation | Killed by |
  | :--- | :--- |
  | `Format` ignores the catalogue | `TestFormat`: `reordered: "1 of 3"`; `ExampleFormat` |
  | `Format` ignores a false answer | `TestFormat`: `other ID: "%!(EXTRA int=1, int=3)"; want the default, 1 of 3` |
  | `Check` compares the count of verbs only | `TestCheck`: `t.d not reported` |
  | an explicit index does not renumber | `TestVerbs`: `verbs("%[2]s then %[1]q") = %[1]s %[2]q; want %[1]q %[2]s` |
  | a literal percent reads as a verb | `TestVerbs`: `verbs("100%% done") = %[1]%; want none` |
  | `msgcat` imports `x/ansi`: the closure | `TestNarrowClosures`, on each GOOS: `msgcat (GOOS=linux) links github.com/charmbracelet/x/ansi, which 0024-MADR D9 does not allow: msgcat → github.com/charmbracelet/x/ansi` |
  | `msgcat` imports `x/ansi`: the lint rule | `golangci-lint`: `import 'github.com/charmbracelet/x/ansi' is not allowed from list 'msgcatstd' (depguard)` |

  The lint rule's first mutation, a blank import, was killed by gofmt,
  goimports and revive's `blank-imports`, not by depguard, which did not
  report it beside them; it was rewritten as a formatted, used import,
  run again alone, and the depguard line above read from its output.
* **Deviation D1, 2026-10-25: `TestResetBeforeExit` asserted an order
  Bubble Tea does not guarantee.**
  * **Found,** by this step's gates: on `remote linux`, `lc-all-c` failed
    one test, which the gate now names
    (`0018-PLAN-python-verification-tools.md` Step 8):
    `termcaptest_test.go:181: the last sequence before exit is
    "\x1b[?2004l", want the 2031 reset`. The test is 0005-PLAN Step 4's,
    in `termcap/termcaptest`, which this step does not touch. It failed 0
    of 300 runs on macOS and 0 of 300 on the Linux host, alone.
  * **The cause,** in bubbletea v2.0.10: `tea.Raw` appends to the
    program's `outputBuf` (`tea.go:1223-1227`), which the render ticker
    flushes (`tea.go:1441-1443`) and `restoreTerminalState` flushes last
    (`tty.go:33-35`), after `stopRenderer` closes the renderer, whose
    teardown writes `CSI ?2004l` and `CSI ?25h`
    (`cursed_renderer.go:172-209`; `tea.go:1284-1288`). So the prober's
    reset comes after the teardown, unless a tick between its queueing and
    the shutdown flushed it first, as under load. A passing run's last
    sequences, printed by a scratch probe: `?25h`, `?2004l`, `?2031h`,
    `?2031l`. The reset is written either way; that it is the last
    sequence, as 0005-PLAN's test item says, is not Bubble Tea's promise.
  * **Resolutions offered:** assert what Bubble Tea guarantees, the reset
    after the set, once, and last of the prober's own sequences, with
    tea's teardown on either side (recommended); or make the order the
    library's, `termcaptest.Terminal.Run` writing the model's `Restore()`
    after tea's `Run`, as `launch.Run` does, which then tests that path
    instead; or report it upstream and keep the flaky assertion.
  * **Chosen by the owner,** 2026-10-25: "Assert what tea guarantees
    (Recommended)". `TestResetBeforeExit` now finds the last of the
    prober's sequences, its queries and mode 2031's set and reset, and
    requires the reset; the existing check of `set,reset`, once each,
    stays. 0005-PLAN gains a dated amendment. This step's scope gains
    `termcap/termcaptest/termcaptest_test.go` and
    `docs/decisions/0005-PLAN-terminal-capabilities-and-services.md`.
  * **Seen failing,** on scratch copies, 2 killed of 2: the prober
    writing DA1 after the reset (`the prober's last sequence before exit
    is "\x1b[c", want the 2031 reset`), and setting the mode again after
    it (`mode 2031: set,reset,set, want set after the probe and reset
    before exit`). Fifty runs of the changed test pass.
* **Gates:** `python3 -m tools.verify gates` over the step's 5 Go files,
  `.golangci.yml` and 4 records.
  * The first run, before D1, passed every gate on macOS (19 of 19) and
    Windows (18, `gofmt` skipped), and failed `lc-all-c` on Linux, D1's
    finding.
  * After D1: macOS 19 passed, 0 failed, in 514 s, `release-check` "331
    file(s) clean in 1 module(s)", with `apicheck` against `v2.1.0`;
    `remote linux` 18 passed, 0 failed, 1 skipped (`gofmt`, which is given
    no files there), in 786 s, `release-check` "334 file(s) clean";
    `remote windows` 18 passed, 0 failed, 1 skipped, in 1943 s, "334
    file(s) clean". `make lint` reports "0 issues" for `GOOS=linux`,
    `darwin` and `windows`. The identifier scan of the change and the
    untracked files is clean.

### Step 3: the inventory

*2026-10-26.* The owner committed Step 2 and said to proceed.

* **Deviation D1, 2026-10-26: a status for settled behaviours, and three
  gaps more.**
  * **Found,** while classifying the audit's behaviours into the step's
    three statuses: several a record already settled, with no control on
    purpose (the frecency clock, 0008-MADR A3 item 7; the boost's bound,
    0008-MADR §2; Bubble Tea asking for modes 2026 and 2027, 0020-MADR
    A2.4; tuitest's update switch, AGENTS.md; a modal overlay's clicks,
    0002-MADR), which are neither floors nor gaps; and three real gaps
    neither of the MADR's PLANs closes: focus falling back to the ring's
    first pane (`workspace/workspace.go:836-842`), the palette closing on
    every accept (`palette/picker.go:381-396`), and `?` always listing the
    prefixes (`palette/engine.go:26`).
  * **Resolutions offered:** a `decided` status citing its record, and the
    three gaps added to the second PLAN (recommended); `decided` and an
    `open` status for the gaps; or every one filed as a floor.
  * **Chosen by the owner,** 2026-10-26: "decided status; gaps to PLAN C
    (Recommended)". The MADR gains A1; the second PLAN's Step 4 gains the
    three additions, their tests and mutations.
* **`docs/guides/defaults.md`:** rule 12 and the four statuses; the
  floors, each with its reason and what a program may do instead; and an
  inventory table per package or group, 15 tables and 135 rows, from the audit,
  with every control named in backticks.
* **`internal/conformance/defaults_test.go`, `TestDefaultsInventory`:** it
  reads every table headed `Behaviour | Default | Control | Status`, and
  resolves each control, `pkg.Name`, `pkg.Type.Method` or
  `pkg.Type.Field`, through the source importer, in the module's
  type-checked packages. A `built` row names at least one control and
  every one exists; a `planned` row names an existing PLAN, and fails
  when every control it names exists already, so a row cannot stay
  `planned` once its step builds it; a `floor` row's controls exist; a
  `decided` row cites an existing record or AGENTS.md; any other status
  fails. A failure names the guide's line and the row's behaviour.
* **Its first run** found five rows wrong, which were fixed: four catalogue
  rows named `msgcat.Catalog`, which exists, where the planned control is
  each package's `WithCatalog` (`command`, `workspace`, `palette`) or
  `termcap.WithReportCatalog`; and `commandtea` was named without its
  directory, `command/commandtea`. A failure now names the guide by its
  path in the repository.
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies of the guide, 8 killed of 8:

  | Mutation | Killed by |
  | :--- | :--- |
  | a built control that does not exist | `(The mouse): built, but workspace exports no WithoutMice` |
  | a built method that does not exist | `built, but command.Registry has no exported field or method Nope` |
  | a built field that does not exist | `built, but launch.Config has no exported field or method OpenTTYs` |
  | a planned row with no PLAN | `(Its own commands, on a key): planned, and names no PLAN` |
  | a planned row whose PLAN does not exist | `planned by 0024-PLAN-no-such-plan.md, which does not exist` |
  | a planned row already built | `(The model's epilogue): planned, but every control it names exists: mark it built` |
  | a decided row with no record | `(The update switch): decided, and cites no record` |
  | a status that is none of the four | `(Letter case): status "maybe"; want built, planned, floor or decided` |

* **Names:** `workspace.FocusFallback` and `FocusFallbackFunc`, A1's two
  types: "2 planned name(s), clean". `docs/glossary.md` reserves them.
* **Gates:** `python3 -m tools.verify gates` over the step's Go file, the
  guide and 5 records: macOS 19 passed, 0 failed, in 568 s,
  `release-check` "334 file(s) clean in 1 module(s)", with `apicheck`
  against `v2.1.0`; `remote linux` 18 passed, 0 failed, 1 skipped
  (`gofmt`, which is given no files there), in 1012 s, "335 file(s)
  clean"; `remote windows` 18 passed, 0 failed, 1 skipped, in 2207 s,
  "335 file(s) clean". The identifier scan of the change and the
  untracked files is clean.

### Step 4: `command`

*2026-10-26.* The owner committed Step 3 and said to proceed.

* **Deviation D1, 2026-10-26: each step marks its own rows built.**
  * **Found,** once the code was written: `TestDefaultsInventory`, from
    Step 3, failed on `docs/guides/defaults.md:40`, `:43` and `:44` with
    "planned, but every control it names exists: mark it built". Step 3's
    test fails a `planned` row whose controls all exist, while this PLAN
    gave Steps 4 to 7 their package paths alone, and moved their rows in
    Step 8. The conflict is the PLAN's, not this step's code.
  * **Resolutions offered:** each of Steps 4 to 7 adds the guide to its
    paths and marks its own rows built, and Step 8 checks that none is
    left `planned` for this PLAN (recommended); or Steps 4 to 7 held
    uncommitted and landed as one change with the rows.
  * **Chosen by the owner,** 2026-10-26: "Each step marks its rows
    (Recommended)". The scope table's rows 4 to 7 gain
    `docs/guides/defaults.md`, and Step 8's bullet is annotated. The MADR
    is unchanged: D3 asks only that the guide match the code.
* **`command/gate.go`:** `GatePolicy`, with `Asks(c Command, o Origin)
  bool`; `GatePolicyFunc`; `DefaultGatePolicy()`, the table `asks` was,
  now `defaultAsks`; and `WithGatePolicy(p)`, where a nil `p` keeps the
  default. `policy.decide` asks the registry's policy, or the table when
  there is none. A policy's panic refuses the request with an error
  wrapping `ErrRefused` and a `*PanicError`, as a gate's does: the PLAN did
  not say, and the registry already recovers a gate's and a handler's
  panics, since `Run` is called from an agent's goroutine.
* **`command/registry.go`:** `WithoutBuiltins()`, after which
  `NewRegistry` adds none of its three commands; `Registry.Forget(id,
  caller)` and `Registry.ForgetAll()`.
* **`command/builtin.go`:** `BuiltinList`, `BuiltinDescribe` and
  `BuiltinQuit`, exported `ID` constants, replacing `idList`, `idDescribe`
  and `idQuit` there and in `builtin_test.go` and `registry_test.go`.
* **The package comment** says `WithGatePolicy` replaces the table.
* **Tests:**
  * `TestDefaultGatePolicy`: with no policy, a nil policy, and
    `DefaultGatePolicy()`, every danger and origin pair (4 by 7) asks
    exactly 0006-MADR §7's table, and `DefaultGatePolicy().Asks` equals
    it;
  * `TestGatePolicy`: a policy that asks for every destructive command
    puts a destructive key request to the gate and lets an agent's
    mutating one run unasked; one that asks for nothing runs an agent's
    mutating and destructive requests with no gate; one that names
    `BuiltinQuit` gates `app.quit`; a panicking one refuses, with a
    `*PanicError` and the gate unasked;
  * `TestForget`: an `AllowAlways` stays remembered after `Forget` of
    another caller or another command, and is asked again after its own;
    `ForgetAll` after three remembered answers asks for each again;
  * `TestWithoutBuiltins`: the IDs are `command.list`, `command.describe`
    and `app.quit`; without the option `All`, `MCPTools` and `Manifest`
    hold all three and `Complete` the two the shell is offered (`app.quit`
    is not); with it, each holds only the program's command, and the
    program may register its own `app.quit`.
* **`docs/guides/defaults.md`,** by D1: the three rows, built-ins at
  construction, which requests ask the gate and remembered verdicts, are
  `built`. `TestDefaultsInventory` passes.
* **`docs/glossary.md`:** `GatePolicy` and `GatePolicyFunc` move from
  "Reserved" to "Built", as Step 2 moved `msgcat`'s.
* **Names:** `GatePolicy` and `GatePolicyFunc`, through `python3 -m
  tools.verify names`: "2 planned name(s), clean".
* **Seen failing,** through `python3 -m tools.verify mutate` on scratch
  copies, 6 killed of 6:

  | Mutation | Killed by |
  | :--- | :--- |
  | `decide` ignores the policy | `TestGatePolicy`: `a destructive key request under a policy that asks for every destructive one: <nil>, 0 gate calls; want refused by the gate` |
  | `DefaultGatePolicy` asks for mutating key requests | `TestDefaultGatePolicy`: `DefaultGatePolicy().Asks(mutating, key) = true, want false` |
  | `Forget` forgets nothing | `TestForget`: `mutating for a: <nil>, 1 gate calls; want 2` |
  | `WithoutBuiltins` is ignored | `TestWithoutBuiltins`: `WithoutBuiltins: All holds [app.quit command.describe command.list own], want only the program's own` |
  | `ForgetAll` forgets only the first caller | `TestForget`: `mutating for b: <nil>, 5 gate calls; want 6` |
  | a policy's panic is not recovered | `TestGatePolicy`: `panic: policy [recovered, repanicked]` |

* **Gates:** `python3 -m tools.verify gates` over the step's 7 Go files,
  the guide, the glossary and 2 records: macOS 19 passed, 0 failed, in
  607 s, `release-check` "335 file(s) clean in 1 module(s)", with
  `apicheck` against `v2.1.0` reporting no incompatible change;
  `remote linux` 18 passed, 0 failed, 1 skipped (`gofmt`, which is given
  no files there), in 1428 s, "335 file(s) clean"; `remote windows` 18
  passed, 0 failed, 1 skipped, in 2234 s, "335 file(s) clean".
  `make pre-add-check` on the 7 Go files is clean. The identifier scan of
  the change is clean.
