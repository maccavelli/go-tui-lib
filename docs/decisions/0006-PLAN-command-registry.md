---
status: proposed
date: 2026-10-02
associated-madr: "0006-MADR-command-registry.md"
---
# Implement the command registry (`command`, `when`, `command/cli`)

Associated MADR: [0006-MADR-command-registry.md](0006-MADR-command-registry.md)

**Revision, 2026-10-02.** The owner answered the MADR's Q1–Q4, and the
MADR is `accepted`. Q4 departs from the recommendation: a Cobra and fang
adapter, `command/cobra`, ships as well as `command/cli`. It needs its own
dependency record first, so it is Step 10, blocked on that record. Step 1,
the scope tables and Verification changed to match. This PLAN is still
`proposed`, because execution is not yet approved.

**Revision, later on 2026-10-02.** MADR amendment A1 (proposed) and
[0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
(proposed) move the Cobra front end into the nested module
`command/cobracmd`, without fang, and add a Kong front end, the nested
module `command/kongcmd`. A1 also aligns §4's tags with Kong's. Step 4,
Step 10, the new Step 11, the scope tables, Out of scope, Verification and
Rollout changed to match. Steps 10 and 11 wait for 0010's acceptance and
for its Phases 2–5, and run only after the root release that contains
Steps 1 to 9.

**Revision, 2026-10-03.** The owner accepted MADR amendment A1, and
approved [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md),
whose Phases 2–6 run after `v0.1.5`. Nothing in this PLAN's steps changes.
Steps 10 and 11 still wait for 0010-PLAN's Phases 2–5. This PLAN stays
`proposed`, because its execution is not yet approved.

## Goal

Ship `when`, `command` and `command/cli`, and the workspace's built-in
commands, so that one definition per action serves keys, the palette, slash
commands, help, the shell and agents (MADR §1–§10).

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag the release.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0006-*`, `docs/README.md` | accept the records |
| 2 | `when/` | expression parser, evaluator, typed keys, layering, fuzz target |
| 3 | `command/` (core) | `ID`, `Command`, `Registry`, `Dispatch` and `Run`, messages, gate, audit |
| 4 | `command/` (args) | `New[A]`, `SchemaOf`, strict decoding, validation, slash arguments; the Kong-aligned tags of MADR A1 once A1 is accepted |
| 5 | `command/` (sources) | `LoadDir`, front matter, `FromMCPPrompts`, `FromACP`, clashes |
| 6 | `command/` (exporters) | `MCPTools`, `CallMCP`, `ACPCommands`, `Manifest` |
| 7 | `workspace/` | `Commands(w)`, `WhenContext()`, `Contexter`, context keys |
| 8 | `command/cli/` | shell subcommands, flags, `--json`, exit codes |
| 9 | `README.md`, `docs/`, `docs/guides/commands.md`, `Makefile` (`fuzz`) | documentation, release notes, close-out |
| ~~10~~ | ~~`command/cobra/`; `go.mod`, `go.sum`; `.golangci.yml` (`depguard`); `AGENTS.md` (Dependencies)~~ | ~~the Cobra and fang adapter, blocked on its dependency record~~ (replaced below, MADR A1) |
| 10 | `command/cobracmd/` (its own `go.mod` and `go.sum`), `command/cobracmd/docs/`; `go.work` | the nested Cobra module, blocked as the revision note says |
| 11 | `command/kongcmd/` (its own `go.mod` and `go.sum`); `go.work` | the nested Kong module, blocked as the revision note says |

Steps 1–9 add no module. `go.mod` and `go.sum` do not change in them, and
every import is the standard library or a module 0001-MADR §3 already
names. The root `go.mod` never gains Cobra or Kong. Step 10's module
requires `github.com/spf13/cobra`, and Step 11's
`github.com/alecthomas/kong`, each in its own `go.mod` only, at the
versions 0010-MADR §1 names and the step re-checks.

### Out of scope

* The keymap engine
  ([0007-PLAN-keymap-engine.md](0007-PLAN-keymap-engine.md)) and the
  palette ([0008-PLAN-command-palette.md](0008-PLAN-command-palette.md)).
  The workspace's existing `KeyMap` stays as it is until 0007 moves it.
* A permission dialog. This PLAN defines the `Gate` interface and the
  default refusal; the dialog is a later widget record.
* An MCP server or ACP transport. The exporters produce the shapes; a host
  wires them.
* ~~Writing the Cobra and fang dependency record. Step 10 waits for it.~~
  0010-MADR is that record, for Cobra and Kong.
* The multi-module tooling (`go.work`, the per-module gates, CI and the
  release procedure). It is
  [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md)'s,
  and Steps 10 and 11 wait for it.
* fang (0010-MADR Q1: dropped). Styled help, version and man pages come
  from `cobracmd` itself (MADR A1).
* A third-party Kong completion module, such as `kong-completion`
  (0010-MADR Q2: `kongcmd` generates its own).
* Shell completion scripts from `command/cli`, and YAML or TOML command
  files (MADR Q1). ~~Completion comes through fang in Step 10.~~
  Completion comes from `cobracmd` and `kongcmd` (Steps 10 and 11).
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
5. **Commit.** One commit per step, made by the owner. At the end of each
   step the agent stops with the pre-add checks passed and the step's
   evidence in the execution record, and stages and commits nothing. The
   owner commits with `git commit --no-edit` and pushes. (2026-10-02: the
   org rules forbid agent commits to `main`, and the owner chose this over
   a `feature/` branch.)

## Implementation Steps

### Step 1: records

The owner accepted the MADR on 2026-10-02, answering Q1–Q4, and the
answers are recorded in it. Q4 added `command/cobra`, which is Step 10.
When execution is approved, set this PLAN `in-progress` and update
`docs/README.md`.

### Step 2: `when`

* The API of MADR §8: `Parse`, `MustParse`, `Expr` with `Eval`, `String`
  and `Keys`, `Check`, `Context`, `Value`, `Map`, `Layered`, and the typed
  `Key[T]` with `NewKey`, `Set` and `Get`.
* A hand-written lexer and recursive-descent parser for the MADR's
  grammar. Regexes compile with `regexp` at parse time. The source limit
  is 4 KiB and the nesting limit 64.
* **Tests:**
  * every operator, alone and combined;
  * the VS Code precedence examples: `!foo && bar` and
    `foo || bar && baz`;
  * unset keys: falsy alone, unequal to everything, so `x != 'a'` holds;
  * `in` and `not in` over a list key;
  * quoted strings with spaces, barewords, numbers, `true` and `false`;
  * `Layered` takes the first context that has the key;
  * `Check` reports an unknown key and a number compared with a string;
  * a 4 KiB + 1 source and 65 nested parentheses are errors.
* **Fuzz.** `FuzzParse`: no panic; for every parsed input,
  `Parse(e.String())` succeeds and gives the same `String()`; `Eval` on a
  random `Map` does not panic. The target joins `make fuzz`.
* **Benchmarks:** `Parse` and `Eval` of a five-clause expression.
* **Mutations:**
  * `&&` binds looser than `||`;
  * an unset key compares equal to `""`;
  * the depth limit is not checked;
  * `Layered` takes the last match;
  * `String()` drops parentheses.

### Step 3: `command` core

* The types of MADR §2, §3, §5 and §7 except the exporters: `ID` and its
  validation, `Command`, `Kind`, `Danger`, `Surface`, `Mode`, `Scope`,
  `Source`, `Handler`, `HandlerFunc`, `Invocation`, `Result`, `Request`,
  `Registry` with its options and methods, `Decision`, `Gate`, `Auditor`,
  `Record`, `SlogAuditor`, and the messages `ResultMsg`, `PromptMsg`,
  `ChangedMsg`, `ConflictMsg` and `QuitRequestMsg`.
* The registry snapshot is an immutable sorted slice and an index map
  behind `atomic.Pointer`. Writes take a mutex, copy, bump `Version` and
  publish. `Watch` waits on a channel closed at each publish.
* The registry's own commands: `command.list`, `command.describe` and
  `app.quit`.
* **Tests:**
  * `ID.Valid` accepts `workspace.focus.next` and refuses an upper-case
    letter, an empty segment, a leading dash and 129 bytes;
  * `Register` refuses a duplicate ID;
  * `All` is sorted, and `Available` honours `When` and `Surfaces`;
  * `Dispatch` of a `Loop` command runs it at once and returns its
    `Result.Cmd` with a `ResultMsg`; an `Async` command runs only when the
    returned `tea.Cmd` runs;
  * `Run` gives the same `Result` as `Dispatch` for the same request;
  * `Cancel` and `Exclusive` cancel a running async command, seen through
    its context;
  * the default policy: for `Origin: Agent`, `ReadOnly` and `UI` run,
    `Mutating` is refused without a gate and asks the gate when there is
    one; for `Origin: CLI`, `Destructive` asks;
  * `AllowAlways` is remembered per command and caller; `RejectOnce` is
    not;
  * `SlogAuditor` writes one record per dispatch to the given logger;
  * `app.quit` sends `QuitRequestMsg` and nothing else;
  * under `-race`, `All` and `Lookup` from many goroutines while
    `ReplaceSource` runs see whole snapshots only, and `Version` never
    falls;
  * `Watch` returns `ChangedMsg` after a change, and not before.
* **Benchmarks:** `Lookup`, `Available` over 500 commands, and `Dispatch`
  of a no-op `Loop` command.
* **Mutations:**
  * the default gate allows `Mutating` for agents;
  * `AllowAlways` is not remembered;
  * `Version` is not bumped on `Remove`;
  * `Exclusive` is ignored;
  * a write mutates the published snapshot in place.

### Step 4: arguments

* `New[A]`, `SchemaOf[A]`, `NoArgs`, `ArgError`, and the `doc` and `arg`
  tags of MADR §4.
* The schema walk uses `reflect.Type.Fields()`. Decoding uses
  `encoding/json/v2` with `RejectUnknownMembers(true)`, then checks the
  keywords the MADR lists. `ArgError` carries a JSON Pointer path.
* `Registry.ParseSlash`: tokenises with quotes, fills `pos` fields in
  order and `name=value` pairs by name, and gives a one-string-argument
  command the whole tail. It produces the same JSON as `--args` would.
* **Tests:**
  * golden schemas (`testdata/golden/schema-*.json`) for a flat struct,
    nested structs, slices, pointers, enums, bounds, `time.Duration`, a
    custom `JSONSchema()` type and `NoArgs`; each has `$schema` 2020-12
    and parses as JSON;
  * `NoArgs` emits `{"type": "object", "additionalProperties": false}`,
    as MCP recommends;
  * strict decoding refuses an unknown member, a wrong type, a missing
    required field, an out-of-range number and a value outside its enum,
    each with an `*ArgError` found by `errors.AsType` and naming its path;
  * `/resize 4`, `/resize delta=4 pane=logs` and `/resize "4"` decode to
    the same arguments; a stray positional is an error;
  * an unsupported field type (a channel) is an error from `SchemaOf`,
    not a panic.
* **Mutations:**
  * `omitzero` fields are marked required;
  * unknown members are accepted;
  * `maximum` is not checked;
  * positional values fill fields in reverse order.
* **Once MADR A1 is accepted,** the tags are A1's vocabulary instead of
  `doc` and `arg:"…"`: `help`, `default`, `enum:"a,b"`, `arg:""` for
  positionals in field order, `short`, `hidden`, `placeholder`, `group`,
  and `schema:"min=…,max=…,minLen=…,maxLen=…,secret"`.
  * **More tests:**
    * `SchemaOf` returns an error for a scalar `enum` field that is neither
      required nor defaulted, as Kong would;
    * a `schema` key Kong does not know, and the `json` name, survive in
      the emitted schema;
    * golden schemas are regenerated for the new tags and read before they
      are committed.
  * The test that one struct means the same thing to the registry and to a
    Kong grammar built from it needs Kong, so it lives in Step 11's module.
  * **More mutations:**
    * a field's `arg` presence is ignored, so a positional becomes a flag;
    * `enum` is split on `|` instead of `,`.

### Step 5: sources and loaders

* `SourceKind`, `LoadDir`, the front-matter parser, `$NAME` and
  `$ARGUMENTS` expansion, `MCPPrompt` and `PromptGetter`,
  `FromMCPPrompts`, `ACPCommand`, `FromACP`, `ReplaceSource`, `Conflict`
  and the default prefixer (MADR §6).
* **Tests,** over a `testing/fstest.MapFS` tree and an `os.Root` on a
  temporary directory:
  * golden commands from a tree with subdirectories, front matter and
    placeholders;
  * each front-matter error (unknown key, duplicate key, nested value, no
    closing `---`, bad `danger`, bad `when`) names the file and line;
  * a file with no front matter is a prompt with its first line as the
    description;
  * a symlink out of the root is refused through `os.Root.FS()`;
  * `$FOCUS` becomes a required string argument; expansion substitutes
    it and `$ARGUMENTS`, and leaves `$lower` and `$$` alone;
  * nothing in a command file is executed: a body containing `!{ls}` and
    `$(ls)` expands to that text unchanged;
  * a loaded command with no `danger` is `Mutating`;
  * MCP prompts become `mcp.<server>.<name>` prompt commands whose schema
    has the prompt's arguments, `required` respected;
  * ACP commands become `acp.<agent>.<name>` forwards; running one sends
    `PromptMsg{Text: "/name args"}`;
  * a loaded slash name equal to a built-in's is renamed `user:name`, and
    a `Conflict` is returned and sent as `ConflictMsg`;
  * `ReplaceSource` removes the source's old commands in the same
    version that adds the new ones.
* **Fuzz.** `FuzzFrontMatter`: no panic, and every accepted input
  re-serialises to an equivalent front matter.
* **Mutations:**
  * a built-in loses a slash clash;
  * unknown front-matter keys are ignored;
  * a loaded command without `danger` defaults to `ReadOnly`;
  * `ReplaceSource` keeps the old commands.

### Step 6: exporters

* `MCPTool`, `MCPToolAnnotations`, `MCPCallResult`, `MCPContent`,
  `ACPCommand`, `Manifest`, and the registry methods `MCPTools`,
  `CallMCP`, `ACPCommands` and `Manifest` (MADR §7).
* The danger table of MADR §2 maps to `readOnlyHint`,
  `destructiveHint`, `idempotentHint` and `openWorldHint`. `Meta`
  exports as `_meta`.
* **Tests:**
  * golden JSON for a catalogue of one command per kind and danger level;
  * the field names are exactly MCP 2025-11-25's (`name`, `title`,
    `description`, `inputSchema`, `outputSchema`, `annotations`,
    `content`, `structuredContent`, `isError`) and ACP's (`name`,
    `description`, `input`, `hint`), checked by decoding the golden JSON
    into `map[string]any` and comparing key sets;
  * `CallMCP` with bad arguments, an unknown name or a gate refusal gives
    `isError: true` and a text message, never a Go error;
  * `CallMCP` runs as `Origin: Agent`, so the default gate applies;
  * hidden commands and commands without `Surfaces&Agent` are not
    exported;
  * `Manifest` round-trips through JSON.
* **Mutations:**
  * `Destructive` exports `destructiveHint: false`;
  * `CallMCP` runs as `Origin: Program`, bypassing the gate;
  * hidden commands are exported.

### Step 7: workspace integration

* `workspace.Commands(w, o...)` with `WithLayouts(map[string]layout.Node)`,
  the commands of MADR §9, `Workspace.WhenContext()`, the optional
  `Contexter` interface, and the typed keys `workspace.focusedPane`,
  `workspace.zoomed`, `workspace.hiddenPanes`, `workspace.overlay`,
  `workspace.modal`, `workspace.width` and `workspace.height`.
* `workspace.theme.set` calls `SetTheme` from
  [0004-PLAN-integrate-charm-v2-and-go-1-27.md](0004-PLAN-integrate-charm-v2-and-go-1-27.md).
  If that PLAN has not landed, this step stops and asks (Rule 1).
* **Tests, through real `tea` messages:**
  * each built-in, dispatched through a registry, gives the same frame and
    `State` as calling the method it wraps;
  * `workspace.panes` returns every visible pane's ID, title, rectangle
    and focus, and the hidden ones as hidden;
  * `workspace.layout.use` offers only the names given in `WithLayouts`,
    as a schema enum;
  * `WhenContext` layers the top overlay, then the focused pane's
    `Contexter`, then the workspace keys;
  * an agent may run `workspace.zoom` (UI) under the default gate;
  * the existing key bindings still work.
* **Golden frames:** the agent-session example zoomed and restored through
  `Dispatch`, across the matrix at 80 and 160 columns.
* **Mutations:**
  * `WhenContext` puts the workspace keys before the focused pane's;
  * `workspace.panes` omits hidden panes;
  * `workspace.focus.next` moves focus twice.

### Step 8: `command/cli`

* `cli.Run` and its options (MADR §10): word and dotted forms, flags from
  the schema, `--args`, `--json`, `--yes`, the verbs `list`, `describe`,
  `schema` and `help`, and the exit codes.
* **Tests:**
  * golden help for the registry and for one command, at 60 and 100
    columns, all ASCII;
  * `--delta 4` and `--args '{"delta": 4}'` give the same request;
  * a string, integer, boolean, enum and array property each parse, and a
    bad value exits 2 with the `ArgError` on stderr;
  * `--json` writes `Result.Value`; without it, `Result.Text`;
  * a `Destructive` command exits 3 without `--yes` or a confirm callback;
  * commands without `Surfaces&CLI` are neither listed nor runnable;
  * nothing is written to `os.Stdout` or `os.Stderr`: the conformance scan
    covers the package, and the tests pass `bytes.Buffer`s.
* **Example:** `ExampleRun` builds a registry with two commands and runs
  `list` and one command with `--json`.
* **Mutations:**
  * `--yes` is not required for `Destructive`;
  * a usage error exits 1;
  * hidden commands are listed.

### Step 9: documentation and close-out

* **`docs/guides/commands.md`:**
  * defining a command, and `New[A]` with tags;
  * kinds, danger levels and the default gate;
  * dispatching from `Update`, and `Run` for the shell and tests;
  * user and project command files, and their front matter;
  * MCP prompts and ACP commands;
  * exporting to an agent;
  * `when` expressions and context keys;
  * `command/cli`.
* **Doc comments** in each package's main file, as 0002-PLAN Step 8
  recorded.
* **Docs tree:** `docs/architecture.md` gains the three packages and
  their imports; `docs/README.md` rows, including "I want to…" rows for
  adding a command, letting an agent drive the TUI, and running commands
  from the shell; README Status.
* **`Makefile`:** the `fuzz` target gains `./when` and `./command`.
* **Release notes** in the execution record. Steps 1–9 can be released
  without ~~Step 10~~ Steps 10 and 11, and must be, because those modules
  require the root release that contains Steps 1–9 (0010-MADR §3). Mark
  `complete` only after ~~Step 10~~ Steps 10 and 11 too, and after CI is
  green on the pushed tree. The owner tags.

### Step 10: `command/cobracmd`, the nested Cobra module (blocked)

*Revised 2026-10-02 for MADR amendment A1 and 0010-MADR.* The original
text is kept, struck through, below the new one.

* **Blocked** until all hold:
  * MADR A1 and 0010-MADR are accepted;
  * [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md)'s
    Phases 2–5 are complete, so the gates loop over modules;
  * the owner has tagged the root release that contains Steps 1–9.
* **Before the first commit,** record in the execution record Cobra's and
  pflag's newest versions and their `go.mod`, the licence of each module
  the new `go.mod` adds, and `govulncheck ./...` in the new module with
  `GOWORK=off`. A finding there stops the step for the owner.
* **What lands** (MADR A1, `cobracmd`):
  * `command/cobracmd/go.mod`: `module
    github.com/maccavelli/go-tui-lib/command/cobracmd`, `go 1.27.1`,
    requiring the tagged root release and `github.com/spf13/cobra`, with no
    `replace`; `go work use ./command/cobracmd` adds it to `go.work`;
  * `New`, `Mount`, `Run` and the options; the tree from dotted IDs,
    groups added before children, annotations, `Hidden`, `Deprecated`;
  * flags from each schema through pflag: an enum `pflag.Value` with
    `FixedCompletions`, `MarkFlagRequired`, positional `Args`, flag
    groups where the schema expresses them;
  * completion from the registry, and scripts for bash, zsh, fish and
    PowerShell written to the caller's writer;
  * `--args`, `--json`, `--yes`, `list`, `describe`, `schema` and the exit
    codes of `command/cli`;
  * help through `SetHelpFunc` and `SetUsageFunc`, with `theme` and `glyph`
    at the option's width;
  * `command/cobracmd/docs`: `GenMan` and `GenMarkdownCustom` to the
    caller's writer, with `DisableAutoGenTag` and a fixed date;
  * depguard, per 0010-MADR §4: Cobra is allowed in this module only, and
    no other adapter's dependency is.
* **Tests:**
  * the shared front-end cases (MADR A1): for the same arguments,
    `cobracmd` and `command/cli` build the same `Request`, write the same
    `--json` output and return the same exit code;
  * commands without `Surfaces&CLI`, and hidden ones, are not in the tree;
  * a dotted ID becomes nested commands, and a category becomes a group;
  * `Mount` grafts the registry into a program's existing tree beside its
    own commands, and both run;
  * an enum flag refuses a value outside the enum, and completes the enum's
    values;
  * golden help across 0001-MADR §6's matrix, at 60 and 100 columns;
  * golden completion scripts for the four shells;
  * golden man and Markdown pages, unchanged between two runs;
  * `Run` never reads `os.Args` and never writes to `os.Stdout` or
    `os.Stderr`: a test runs it with empty buffers and a planted
    `os.Args`, and the conformance scan covers the module;
  * a second `New` in one process builds a working tree, and registers no
    completion twice for any one flag, so Cobra's refusal of a repeated
    registration never fires.
* **Mutations:**
  * a flag's type is taken from the Go field instead of the schema;
  * `Destructive` runs without `--yes`;
  * `Run` does not call `SetArgs`, so `os.Args` is read;
  * a group is added after its children (Cobra panics);
  * depguard allows Cobra in the root module.
* **Checks:** the per-module gates of 0010-MADR §4, with `GOWORK=off`, and
  again in workspace mode.

*Superseded 2026-10-02 by the text above (MADR A1); the original, struck
through:*

* ~~**Blocked** until a dependency record naming `github.com/spf13/cobra`
  and `github.com/charmbracelet/fang` is accepted (AGENTS.md
  Dependencies).~~
* ~~**What lands** (MADR §10): `command/cobra` builds a `*cobra.Command`
  tree from a registry; fang wraps the root for styled help, man pages,
  completion and version; `go.mod` and `go.sum` gain the two modules;
  `depguard` allows Cobra and fang in `command/cobra/`; AGENTS.md's
  Dependencies list names them.~~
* ~~**Tests:** the same requests, outputs and exit codes as
  `command/cli`; no hidden or non-CLI commands; output only to Cobra's
  writers; golden help at 60 and 100 columns, with and without colour.~~
* ~~**Mutations:** a flag's type from the Go field; `Destructive` without
  `--yes`; `depguard` allows Cobra in `command`.~~

### Step 11: `command/kongcmd`, the nested Kong module (blocked)

Added 2026-10-02 for MADR amendment A1.

* **Blocked** on the same three conditions as Step 10. Steps 10 and 11
  are independent of each other.
* **Before the first commit,** record Kong's newest version and its
  `go.mod`, the licence of each module the new `go.mod` adds, and
  `govulncheck ./...` in the new module with `GOWORK=off`.
* **What lands** (MADR A1, `kongcmd`):
  * `command/kongcmd/go.mod`: `module
    github.com/maccavelli/go-tui-lib/command/kongcmd`, `go 1.27.1`,
    requiring the tagged root release and `github.com/alecthomas/kong`,
    with no `replace`; `go work use ./command/kongcmd`;
  * `New`, `Adapter.Options`, `Adapter.Parser`, `Adapter.Run`,
    `Adapter.Resolver` and the options;
  * one `kong.DynamicCommand` per top-level ID segment, each grammar built
    with `reflect.StructOf` from the schema, with `id` and `danger` tags;
  * dispatch from `ctx.Selected()` through the registry, as `Origin: CLI`;
  * `kong.Name`, `kong.Writers`, and an `Exit` that panics a sentinel
    `Run` recovers;
  * `kong.ExplicitGroups`, `PostBuild`, and a `kong.Help` printer with
    `theme` and `glyph` at the option's width;
  * completion scripts for bash, zsh, fish and PowerShell, generated from
    the registry and written to the caller's writer (0010-MADR Q2);
  * depguard, per 0010-MADR §4.
* **Tests:**
  * golden completion scripts for each shell, from one registry; adding a
    command or an enum value changes each script; no script is written to
    `os.Stdout`, and no module beyond Kong is required;
  * the shared front-end cases, as in Step 10;
  * **one struct, two readers:** for representative argument structs in
    A1's tags, `SchemaOf` and a Kong grammar built from the same struct
    agree on names, required fields, enums, defaults and the order of
    positionals;
  * a dotted ID selects through nested `cmd` fields, and runs the right
    registry command;
  * `--help` prints help and returns exit code 0, and `Parse` does not
    continue after it;
  * mounted into a program's own grammar, the program's static commands
    and the registry's commands both parse, and `Run` reports `handled`
    false for the program's own;
  * help at the option's width, whatever `$COLUMNS` says; golden help
    across 0001-MADR §6's matrix, at 60 and 100 columns;
  * nothing is written to `os.Stdout` or `os.Stderr`, and `os.Args` is never
    read: empty buffers, a planted `os.Args`, and the conformance scan;
  * `Resolver` fills a flag from a value given to it, and a command-line
    flag still wins;
  * no generated field carries an `env` tag unless an option asks for one.
* **Mutations:**
  * `Exit` returns instead of panicking, so parsing continues after
    `--help` (the `--help` test must fail);
  * dispatch reads the first registry command instead of `ctx.Selected()`;
  * an enum's default is dropped from the grammar (Kong refuses it);
  * `kong.Writers` is not set;
  * depguard allows Kong in the root module.
* **Checks:** as in Step 10.

## Verification

* Every step's mutations are killed.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`.
* `go mod tidy -diff` is clean. `go.mod` is unchanged by Steps 1–9, and
  ~~Step 10 adds only Cobra and fang, at the versions their record pins.~~
  the root's `go.mod` never names Cobra or Kong.
* ~~`depguard` allows Cobra and fang in `command/cobra/` only.~~
* For Steps 10 and 11, every gate of 0010-MADR §4 passes in each nested
  module with `GOWORK=off`, and again in workspace mode: vet, `-race`,
  `LC_ALL=C`, lint for three operating systems, `go mod tidy -diff` and
  govulncheck. Each nested `go.mod` has no `replace`, and requires a
  tagged root release.
* `depguard` allows Cobra in `command/cobracmd` only, and Kong in
  `command/kongcmd` only.
* `depguard` still refuses the Charm v1 paths, mcplib, the MCP go-sdk and
  go-llmprovider-sdk in every package, the new ones included.
* `internal/conformance` covers `when`, `command`, `command/cli`,
  ~~`command/cobra`~~ `command/cobracmd` and `command/kongcmd`, and finds
  nothing.
* The shared front-end cases pass in `command/cli`, `cobracmd` and
  `kongcmd`.
* The exporters' field names match the MCP 2025-11-25 and ACP pages cited
  in the MADR.
* The identifier scan of 0001-PLAN V7 finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–9 and tags the release. ~~Step 10
  follows in a later release once its dependency record is accepted.~~
  Steps 10 and 11 follow, in the order 0010-MADR §3 sets:
  1. the root release that contains Steps 1–9 is tagged;
  2. each nested module requires that release, lands, and is tagged under
     its prefix, `command/cobracmd/v0.1.0` and `command/kongcmd/v0.1.0`;
  3. a consumer smoke test, in a scratch module outside the repository,
     runs `go get` on each tagged module and `go build ./...`, and its
     output goes in the execution record.

  pi-go adopts the registry under its own records: its slash commands, user
  and project command files, ACP agent commands and palette all register
  here.
* **Rollback.** Before the push, each step is one local commit. After it, a
  patch release fixes forward. The packages are new, so a consumer that
  does not import them is unaffected. Step 7's addition to `workspace` is
  additive: removing `Commands` and `WhenContext` breaks only callers of
  those two names.
* **Rolling back a front end.** `cobracmd` and `kongcmd` are released apart
  from the root, so a fault in one is fixed forward under its own prefix,
  and a consumer pins its previous tag meanwhile. Tags are never moved or
  deleted (0010-MADR §3).

## Execution Record

None yet.
