---
status: proposed
date: 2026-10-02
associated-madr: "0006-MADR-command-registry.md"
---
# Implement the command registry (`command`, `when`, `command/cli`)

Associated MADR: [0006-MADR-command-registry.md](0006-MADR-command-registry.md)

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
| 4 | `command/` (args) | `New[A]`, `SchemaOf`, strict decoding, validation, slash arguments |
| 5 | `command/` (sources) | `LoadDir`, front matter, `FromMCPPrompts`, `FromACP`, clashes |
| 6 | `command/` (exporters) | `MCPTools`, `CallMCP`, `ACPCommands`, `Manifest` |
| 7 | `workspace/` | `Commands(w)`, `WhenContext()`, `Contexter`, context keys |
| 8 | `command/cli/` | shell subcommands, flags, `--json`, exit codes |
| 9 | `README.md`, `docs/`, `docs/guides/commands.md`, `Makefile` (`fuzz`) | documentation, release notes, close-out |

No module is added. `go.mod` and `go.sum` do not change. Every import is
the standard library or a module 0001-MADR §3 already names.

### Out of scope

* The keymap engine
  ([0007-PLAN-keymap-engine.md](0007-PLAN-keymap-engine.md)) and the
  palette ([0008-PLAN-command-palette.md](0008-PLAN-command-palette.md)).
  The workspace's existing `KeyMap` stays as it is until 0007 moves it.
* A permission dialog. This PLAN defines the `Gate` interface and the
  default refusal; the dialog is a later widget record.
* An MCP server or ACP transport. The exporters produce the shapes; a host
  wires them.
* Cobra or fang adapters, shell completion scripts, YAML or TOML command
  files (MADR Q1, Q4).
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

The owner accepts the MADR, answering Q1–Q4. Record the answers, set the
MADR `accepted` and this PLAN `in-progress`, and update `docs/README.md`.
If an answer changes a decision (for example YAML for Q1, which needs a
module and its own record), amend the MADR before Step 2.

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
* **Release notes** in the execution record. Mark `complete` after CI is
  green on the pushed tree. The owner tags.

## Verification

* Every step's mutations are killed.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`;
  * `make fuzz`.
* `go mod tidy -diff` is clean, and `go.mod` is unchanged by this PLAN.
* `depguard` still refuses the Charm v1 paths, mcplib, the MCP go-sdk and
  go-llmprovider-sdk in every package, the new ones included.
* `internal/conformance` covers `when`, `command` and `command/cli`, and
  finds nothing.
* The exporters' field names match the MCP 2025-11-25 and ACP pages cited
  in the MADR.
* The identifier scan of 0001-PLAN V7 finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–9 and tags the release. pi-go
  adopts the registry under its own records: its slash commands, user and
  project command files, ACP agent commands and palette all register here.
* **Rollback.** Before the push, each step is one local commit. After it, a
  patch release fixes forward. The packages are new, so a consumer that
  does not import them is unaffected. Step 7's addition to `workspace` is
  additive: removing `Commands` and `WhenContext` breaks only callers of
  those two names.

## Execution Record

None yet.
