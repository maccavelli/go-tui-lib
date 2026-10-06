---
status: accepted
date: 2026-10-02
decision-makers: owner
consulted: 0003-REPORT-agent-tui-ecosystem-research.md (opencode, gemini-cli, codex, crush, toad, Textual, VS Code, k9s, lazygit, gh-dash); MCP specification 2025-11-25, and 2026-07-28 for amendment A2; Agent Client Protocol; Go 1.27.1 standard library; for amendment A1, 0010-REPORT-nested-modules-and-adapter-sources.md (Cobra v1.10.2, pflag v1.0.10, fang v2.0.1 and Kong v1.16.1 at source)
informed: pi-go
---
# Make one command registry the source of every action, for keys, palette, slash commands, the shell and agents, on open standards

## Context and Problem Statement

On 2026-10-01 the owner asked:

> i want to stay working on go-tui-lib, i want to ensure the
> bubbletea/lipgloss/charm stack is tightly integrated, coded idiomatically,
> based on go1.27.1 optimizations and standards. i want to enhance and expand
> the tui library functionality in 5 more ways that will bring benefit and
> value to the codebase functionality. i want a large, openstandards api
> surface, with a wide assortment of agentic commands available through the
> terminal UI and also the TUI. look at similar opensource projects across
> the web. dig into how they solved interesting problems and issues bringing
> functionality to reality. find ideas, features, enhancements, and
> optimizations. present 10 items to spec out and plan. future-proof is a
> focus.

On 2026-10-02, after the ten items were presented, the owner said: "write
findings into a report then follow recommendations and proceed." The
recommendation put the command registry first among the five expansions,
because the palette
([0008-MADR-command-palette.md](0008-MADR-command-palette.md)) and the keymap
([0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md)) are built on it.

The owner's standing direction also applies: build speculative API when it is
sensible, for extensibility, flexibility and idiomatic, modular design.

Today go-tui-lib has no notion of a command. `workspace.KeyMap` binds keys
straight to behaviour inside `Workspace.Update`. A program that wants a
palette, slash commands, a help screen, a shell subcommand and an agent tool
for the same action writes each one separately. This record decides the
model that every one of those surfaces reads from.

Evidence (read-only, 2026-10-02; the research is in
[0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)):

* **Every mature agent TUI converges on one registry** (REPORT §3, §4):
  * **opencode** gives commands dotted names (`command.palette.show`), a
    default binding and a description each. Slash names and aliases come
    from the same entries (`slashName`, `slashAliases`).
  * **gemini-cli's `CommandService`** merges built-in, file, MCP-prompt and
    skill loaders. A resolver keeps every command addressable by giving
    non-built-in commands a source prefix on a name clash, and reports the
    clashes as events (`packages/cli/src/services/CommandService.ts`). Its
    `SlashCommand` type also carries `altNames`, `hidden`, `autoExecute`
    and `isSafeConcurrent` (safe to run while the agent is busy).
  * **codex** declares slash commands as one enum, with `description()` and
    `supports_inline_args()`.
  * **Textual** separates stable action IDs from keys. `check_action()`
    enables an action from current state, and the palette, help and keymap
    read the same actions.
  * **VS Code** makes the command ID the unit of keybindings, menus and the
    palette, gated by a `when` clause.
* **User commands are files.** opencode (Go version) reads Markdown files
  from a user and a project directory, prefixes them `user:` and
  `project:`, turns subdirectories into name segments, and prompts for
  `$NAME` placeholders matched by `\$([A-Z][A-Z0-9_]*)`. gemini-cli reads
  TOML files. k9s and lazygit read YAML plugins with a scope, a command and
  a confirm flag.
* **The open standards the registry must speak** (fetched 2026-10-02):
  * **MCP 2025-11-25, `tools/list`** (*A2: the exporters target 2026-07-28,
    whose `Tool` is the same*). A tool is `name`, `title`,
    `description`, `icons`, `inputSchema`, `outputSchema`, `annotations`
    and `execution`. `inputSchema` defaults to JSON Schema 2020-12. A tool
    with no parameters should use
    `{"type": "object", "additionalProperties": false}`. Names should be 1
    to 128 characters of `A-Z a-z 0-9 _ - .`, so a dotted ID such as
    `admin.tools.list` is a valid name. `ToolAnnotations` holds `title`,
    `readOnlyHint` (default false), `destructiveHint` (default true),
    `idempotentHint` (default false) and `openWorldHint` (default true).
    `tools/call` returns `content`, `structuredContent` and `isError`.
    A changed list is announced by `notifications/tools/list_changed`.
  * **MCP prompts.** A `Prompt` has `name`, `title`, `description` and
    `arguments`, each a `PromptArgument` with `name`, `description` and
    `required`.
  * **ACP slash commands.** An agent sends `available_commands_update` with
    `availableCommands`, each `{name, description, input: {hint}}`. The
    client runs one by sending it as ordinary prompt text, `/name args`.
  * **ACP permissions.** A `PermissionOption` is `{optionId, name, kind}`,
    where `kind` is `allow_once`, `allow_always`, `reject_once` or
    `reject_always`. The outcome is `selected` with an `optionId`, or
    `cancelled`.
  * **VS Code `when` clauses.** Operators are `!`, `&&`, `||`, `==`, `!=`,
    `>`, `>=`, `<`, `<=`, `=~` (a `/regex/`), `in` and `not in`.
    `!foo && bar` is `(!foo) && bar`, and `foo || bar && baz` is
    `foo || (bar && baz)`.
* **Go 1.27.1 gives everything this needs in the standard library** (checked
  against `$GOROOT/api`):
  * `encoding/json/v2` is GA, with `RejectUnknownMembers` and `omitzero`,
    for strict argument decoding;
  * `reflect.Type.Fields() iter.Seq[StructField]` (Go 1.26) walks an
    argument struct to emit its schema;
  * `errors.AsType[T]` (Go 1.26) for typed argument errors;
  * generic methods (Go 1.27), `iter`, `sync/atomic.Pointer`, `regexp`
    (RE2, linear time), `io/fs` with `os.Root.FS`, `log/slog` and `flag`.
* **Charm v2** (go doc, bubbletea v2.0.10 and bubbles v2.2.1): `tea.Cmd` is
  `func() tea.Msg`, `tea.Batch` and `tea.Sequence` combine commands, and
  `help.KeyMap` is `ShortHelp() []key.Binding` and
  `FullHelp() [][]key.Binding`.
* **The first consumer.** pi-go's v1 scope
  (pi-go `docs/decisions/0005-MADR-v1-feature-scope.md`, `proposed`) lists
  slash completion, a command palette, pickers, a permission dialog and an
  ACP client. All of them need named, described, argument-checked actions.
* **What the library already offers to commands.** `Workspace` exposes
  `Focus`, `FocusNext`, `FocusPrev`, `Zoom`, `Toggle`, `Resize`,
  `SetLayout`, `SetState`, `Push`, `Pop`, `Send` and `Broadcast`, and
  `layout` has four presets. These become the first built-in commands.

## Decision Drivers

* **One definition per action.** A command is written once. The keymap,
  palette, slash completion, help, the shell and an agent all read it, so
  they cannot drift.
* **Open standards at the edges.** Arguments are JSON Schema 2020-12.
  Commands export as MCP tools and as ACP available commands. Availability
  is a VS Code-style `when` clause. Danger maps to MCP annotations and ACP
  permission kinds. No SDK is imported for any of them (AGENTS.md
  Dependencies forbids the MCP go-sdk).
* **Agents are first-class callers, and the human stays in the loop.** An
  agent can list and run TUI commands. Anything that changes more than the
  presentation goes through a permission gate, as the MCP specification
  says clients should.
* **The same command runs with and without a `tea.Program`.** A shell
  invocation has no event loop. The handler signature must serve both.
* **The workspace is not concurrency-safe.** Commands that touch it run on
  the event loop. Slow commands run off it, can be cancelled, and report
  back as messages.
* **Standard library only.** The core packages add no module (AGENTS.md
  Dependencies). The Cobra and fang adapter of §10 is the one exception,
  and it lands only under its own dependency record.
* **Extensible without changing the library.** New sources (plugins,
  skills), new surfaces (a Cobra adapter, an MCP server in the host) and
  new metadata fit without breaking callers.

## Considered Options

* **A. A standard-library `command` registry, a `when` expression package and a `command/cli` adapter, with MCP and ACP exporters.**
* **B. A command table per surface:** the keymap, palette, slash popup and CLI each keep their own list, as today.
* **C. Cobra as the registry:** a `cobra.Command` tree is the single source, and the TUI reads it (with fang for styling).
* **D. A general expression engine for `when`** (CEL or expr-lang) inside option A, instead of a small parser.

## Decision Outcome

Chosen option: **"A"**, because:

* it is the only option in which one definition serves keys, palette,
  slash, help, shell and agent;
* it speaks MCP, ACP, JSON Schema and VS Code's `when` grammar without
  importing any of their SDKs;
* its core adds no module. Cobra and fang, which option C would make the
  registry, are only an adapter over it (§10).

### 1. Packages and their dependencies

```text
 command       Command, Registry, invocation, args and JSON Schema, sources
               and loaders, gate, audit, MCP and ACP shapes
                                              → when, bubbletea, stdlib
 command/cli   run commands as shell subcommands with flags and --json
                                              → command, stdlib flag
 command/cobra the same commands as a Cobra tree, styled by fang
                                              → command, cobra, fang
                                                (after its dependency record)
 when          context-key expressions: parse, check, evaluate → stdlib only
 workspace     gains Commands(w) and WhenContext()  → command, when (new)
```

* **Imports point downward.** `when` imports only the standard library, so
  `keymap` and any non-Charm front end can use it. `command` imports `when`
  and `bubbletea` (for `tea.Cmd` and `tea.Msg` only). `workspace` imports
  `command` to publish its own commands. `command` never imports
  `workspace`.
* **Later records build on these:** `keymap`
  ([0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md)) binds keys to
  `command.ID`s, and `palette`
  ([0008-MADR-command-palette.md](0008-MADR-command-palette.md)) lists and
  runs them.

### 2. The command

```go
// ID is a dotted, lowercase name: "workspace.focus.next".
// Segments match [a-z0-9][a-z0-9-]*; the whole ID is at most 128 bytes,
// so it is also a valid MCP tool name.
type ID string
func (id ID) Valid() error
func (id ID) Segments() iter.Seq[string]

type Command struct {
    ID          ID
    Title       string   // "Focus next pane"; palette and MCP title
    Description string   // one or two sentences; help, MCP, ACP
    Category    string   // grouping in the palette, help and CLI
    Slash       string   // "next"; empty means no slash command
    Aliases     []string // more slash names
    ArgHint     string   // ACP input.hint, e.g. "pane id"

    Kind     Kind     // Action, Prompt or Forward
    Args     Schema   // JSON Schema 2020-12 for the arguments
    Output   Schema   // optional schema of Result.Value (MCP outputSchema)
    When     string   // availability, a when expression; "" means always
    Scope    Scope    // Global, or a pane, overlay or mode name
    Danger   Danger   // ReadOnly, UI, Mutating or Destructive
    Idempotent, OpenWorld bool // MCP hints
    Surfaces Surface  // Key | Palette | Slash | CLI | Agent; default all
    Mode     Mode     // Loop (on the event loop) or Async (off it)
    Exclusive  bool   // a new run cancels the running one (Textual)
    WhileBusy  bool   // may run while the agent streams (gemini-cli)
    Hidden     bool   // runnable, not listed

    Source  Source        // Builtin, User, Project, MCP, ACP, Plugin + name
    Meta    map[string]any // open-ended; exported as MCP _meta
    Handler Handler
}

type Schema = json.RawMessage
```

* **Three kinds cover the agentic sources:**
  * `Action` runs a Go `Handler`.
  * `Prompt` expands a template into text for the agent. Markdown command
    files and MCP prompts are prompts. Running one returns a `PromptMsg`,
    which the host sends to its agent.
  * `Forward` sends `/name args` to the agent as prompt text. ACP
    available commands are forwards, as the ACP specification says.
* **Danger has four levels,** chosen so each maps to the standards:

  | `Danger` | Meaning | MCP annotations | Default gate |
  | :--- | :--- | :--- | :--- |
  | `ReadOnly` | changes nothing | `readOnlyHint: true` | allowed |
  | `UI` | changes only presentation; reversible | `readOnlyHint: false`, `destructiveHint: false`, `idempotentHint` as set | allowed |
  | `Mutating` | changes program or session state | `destructiveHint: false` | agent: ask |
  | `Destructive` | cannot be undone, or writes outside the project | `destructiveHint: true` | agent and CLI: ask |

  A loaded command with no declared danger is `Mutating`, never lower.
  (*A4: the zero `Danger` is "not declared", and `Register` refuses it.*)
* **`Meta`** carries what this record does not foresee (icons, a plugin's
  own keys). It round-trips through the exporters as `_meta`.

### 3. Handlers: one signature for the TUI, the shell and agents

```go
type Handler interface {
    Run(ctx context.Context, inv *Invocation) (Result, error)
}
type HandlerFunc func(ctx context.Context, inv *Invocation) (Result, error)

type Invocation struct {
    Command Command
    Args    json.RawMessage // validated against Command.Args
    Raw     string          // the slash tail, verbatim
    Origin  Origin          // Key, Mouse, Palette, Slash, CLI, Agent, Program
    Caller  string          // e.g. the agent's or MCP client's name
    Context when.Context    // the context the command was enabled in
}

type Result struct {
    Value any     // structured output: --json, MCP structuredContent
    Text  string  // human output: CLI stdout, MCP text content, a toast
    Cmd   tea.Cmd // a follow-up effect for a TUI host; ignored by the CLI
}
```

* **The handler does the work and returns the effect.** A handler that
  needs the Bubble Tea runtime returns `Result.Cmd`. A handler that only
  computes returns `Value` and `Text`. The same handler then serves every
  surface. A command whose result is meaningless without a program is
  declared without `Surfaces&CLI`, and the CLI does not offer it.
* **Two ways to run:**

  ```go
  // TUI: call from the program's Update. Loop commands run now, on the
  // event loop, and their Result.Cmd is returned with a ResultMsg.
  // Async commands run inside the returned tea.Cmd, under a context the
  // registry can cancel.
  func (r *Registry) Dispatch(ctx context.Context, req Request) tea.Cmd

  // Shell, agents and tests: run to completion and return.
  func (r *Registry) Run(ctx context.Context, req Request) (Result, error)

  type Request struct {
      ID     ID
      Args   json.RawMessage
      Raw    string
      Origin Origin
      Caller string
      Context when.Context
  }
  ```

* **`Loop` is the default `Mode`,** because the workspace is not safe for
  concurrent use (0003-REPORT §1.12) and most UI commands are instant.
  (*A7: `Run` from off the loop hands a `Loop` command to the loop through
  `WithLoop`.*)
  `Async` is for slow work: a file search, a network call, a long prompt
  expansion.
* **Cancellation.** `Registry.Cancel(id ID)` and `CancelAll()` cancel
  running async invocations. `Exclusive` cancels the previous run of the
  same command first, because results can otherwise arrive out of order
  (Textual's `exclusive` workers).
* **Messages** the registry produces:
  * `ResultMsg{Request, Result, Err, Duration}` after every dispatch;
  * `PromptMsg{Text, Command, Request}` for prompt and forward commands;
  * `ChangedMsg{Version}` when commands are added, removed or replaced;
  * `ConflictMsg{[]Conflict}` when a load renamed a command;
  * `QuitRequestMsg{}` from `app.quit`. The program decides whether to
    quit, so `ctrl+c` and quitting stay the program's (0001-MADR §6,
    rule 2).

### 4. Arguments: JSON Schema from Go types

```go
// New builds a command whose arguments decode into A. The schema is
// emitted from A, and Run receives a decoded, validated A.
func New[A any](id ID, title string,
    run func(ctx context.Context, inv *Invocation, args A) (Result, error),
    opts ...Option) (Command, error)

// SchemaOf emits a JSON Schema 2020-12 document for a Go type.
func SchemaOf[A any]() (Schema, error)

// NoArgs is the argument type of a command without arguments:
// {"type": "object", "additionalProperties": false}.
type NoArgs struct{}

type ArgError struct{ Path, Reason string } // errors.AsType[*ArgError]
```

* **The struct is the schema.** For example:

  ```go
  type resizeArgs struct {
      Pane  layout.PaneID `json:"pane,omitzero" doc:"pane to resize; default the focused pane"`
      Delta int           `json:"delta" doc:"cells to grow; negative shrinks" arg:"min=-200,max=200,pos=0"`
  }
  ```

  * Names come from `json` tags. A field is required unless it is a
    pointer or has `omitempty` or `omitzero`.
  * `doc` is the description.
  * `arg` holds `enum=a|b`, `min`, `max`, `minLen`, `maxLen`, `default`,
    `pos=N` (slash and CLI position) and `secret` (masked in audit
    records). (*A1 replaced these tags; A5 says how the ones with no JSON
    Schema keyword are carried.*)
  * Supported types: string, bool, the integer and float kinds, slices,
    nested structs, `time.Duration` (a string), and any type with its own
    `JSONSchema() Schema` method.
* **Decoding is strict.** Arguments decode with `encoding/json/v2` and
  `RejectUnknownMembers(true)`, then the emitted keywords are checked:
  `type`, `required`, `enum`, `minimum`, `maximum`, `minLength`,
  `maxLength`, `items` and `additionalProperties: false`. A schema loaded
  from elsewhere (an MCP prompt) is carried as is. Only these keywords are
  enforced on it, and the record says so in the docs.
* **Slash arguments.** `/resize 4` and `/resize delta=4 pane=logs` both
  work. Positional values fill `pos` fields in order, `name=value` fills
  any field, and quotes group words. A command with one string argument
  takes the whole tail. The parse result is the same JSON that `--args`,
  the palette and an agent send.

### 5. The registry

```go
func NewRegistry(o ...RegistryOption) *Registry
// Options: WithGate(Gate), WithAuditor(Auditor), WithPrefixer(func(Source) string)

func (r *Registry) Register(cmds ...Command) error      // built-ins; a duplicate ID is an error
func (r *Registry) ReplaceSource(src Source, cmds []Command) []Conflict
func (r *Registry) Remove(ids ...ID)
func (r *Registry) Lookup(id ID) (Command, bool)
func (r *Registry) Slash(name string) (Command, bool)
func (r *Registry) ParseSlash(line string) (Request, error)
func (r *Registry) All() iter.Seq[Command]              // sorted by ID
func (r *Registry) Available(ctx when.Context, s Surface) iter.Seq[Command]
func (r *Registry) Version() uint64
func (r *Registry) Watch() tea.Cmd                      // next ChangedMsg
```

* **Reads are lock-free.** The registry keeps an immutable snapshot behind
  an `atomic.Pointer`. A write copies it, bumps `Version` and publishes
  it. Iterators walk one snapshot, so they never see a half-applied load.
  This is safe from any goroutine, unlike the workspace.
* **`Available`** evaluates each command's `when` against the caller's
  context and its `Surfaces`. The palette, slash popup and help call it.
  The keymap calls `Lookup`, then checks `When` itself.
* **Change notification.** `Watch` returns a command that waits for the
  next version and returns `ChangedMsg`. The host re-issues it, as with
  any Bubble Tea subscription. MCP hosts map it to
  `notifications/tools/list_changed`.

### 6. Sources, loaders and name clashes

```go
type SourceKind uint8 // Builtin, User, Project, MCP, ACP, Plugin
type Source struct{ Kind SourceKind; Name string } // Name: server, agent or plugin

// LoadDir reads Markdown command files from fsys.
func LoadDir(fsys fs.FS, src Source) ([]Command, []error)
// FromMCPPrompts turns an MCP prompts/list result into Prompt commands.
// get fetches and expands a prompt; the host owns the MCP client.
func FromMCPPrompts(server string, prompts []MCPPrompt, get PromptGetter) []Command
// FromACP turns an available_commands_update into Forward commands.
func FromACP(agent string, cmds []ACPCommand) []Command
```

* **IDs are namespaced by source, so they never clash.** A loaded command's
  ID is `user.<name>`, `project.<name>`, `mcp.<server>.<name>`,
  `acp.<agent>.<name>` or `plugin.<name>.<command>`. Built-in IDs are the
  program's and the library's. Two built-ins with one ID are a programming
  error, and `Register` returns it.
* **Slash names can clash, and built-ins win.** A loaded command whose slash
  name is taken is renamed `<prefix>:<name>` (`/user:review`,
  `/mcp:github:issue`), as gemini-cli does. Every rename is a `Conflict`,
  returned and sent as `ConflictMsg`, so a program can show it. `Prefixer`
  replaces the prefix rule. (*A6: a built-in registered later takes the
  name too; a renamed name that is also taken is dropped; `ConflictMsg`
  arrives through `Watch`; a command that cannot load is a `Conflict`
  with `Err`.*)
* **`ReplaceSource` swaps a source's whole set atomically,** because ACP
  sends the full list in each `available_commands_update` and MCP sends
  `list_changed` without a diff.
* **Markdown command files** follow opencode, with a small front matter:

  ```markdown
  ---
  description: Review the staged diff for bugs
  slash: review
  danger: read-only
  when: workspace.focusedPane == transcript
  arg.FOCUS: what to look at first
  ---
  Review the staged changes. Focus on $FOCUS. $ARGUMENTS
  ```

  * The front matter is a strict subset: one `key: value` per line, from a
    fixed set of keys (`title`, `description`, `slash`, `aliases`,
    `category`, `danger`, `when`, `hidden`, `arg.<NAME>`). An unknown key,
    a duplicate key or a nested value is an error naming the file and line.
  * `$NAME` placeholders (`\$([A-Z][A-Z0-9_]*)`) become required string
    arguments. `$ARGUMENTS` is the whole slash tail. (*A6: the property is
    named as written, and is positional in order of first use.*)
  * Subdirectories become ID segments: `git/commit.md` is `user.git.commit`
    and `/user:git:commit`. (*A6: its slash name is `git:commit`;
    `/user:git:commit` is the form a clash gives it.*)
  * The loader reads an `fs.FS`. A host passes `os.Root.FS()` for its
    user and project directories, so a symlink cannot escape them. Where
    those directories are (XDG, project root) is the host's choice.
  * Expansion is plain text substitution. There is no shell execution:
    gemini-cli's `!{…}` form is not supported, because it would let a
    shared file run commands.
* **TOML and YAML are not read,** because each needs a parser module.
  Owner question Q1.

### 7. Agents, the gate and the audit trail

```go
type Decision uint8 // AllowOnce, AllowAlways, RejectOnce, RejectAlways
func (d Decision) ACPKind() string // "allow_once", …

type Gate interface {
    // Decide is asked before a command runs, when the policy says ask.
    Decide(ctx context.Context, inv *Invocation) (Decision, error)
}
type Auditor interface{ Audit(Record) }
type Record struct {
    ID ID; Origin Origin; Caller string
    Args json.RawMessage // secret fields masked
    Decision Decision; Started time.Time; Duration time.Duration; Err error
}
func SlogAuditor(l *slog.Logger) Auditor
```

* **The default policy is safe.** `ReadOnly` and `UI` commands run for any
  origin. `Mutating` from an agent, and `Destructive` from an agent or the
  shell, ask the `Gate`. Without a gate, the answer is `RejectOnce`. A
  program opts in to agent control by installing one, typically the
  permission dialog a later record provides.
* **`AllowAlways` and `RejectAlways`** are remembered per command and
  caller for the registry's lifetime, matching ACP's option kinds. A
  program can persist them through the gate.
* **Every dispatch is audited** when an `Auditor` is set. `SlogAuditor`
  writes structured records to the caller's `slog.Logger`, never to
  stdout (0001-MADR §6, rule 1). Arguments marked `secret` are masked.
* **Exporters** make the registry an agent's tool set:

  ```go
  func (r *Registry) MCPTools(ctx when.Context) []MCPTool          // tools/list "tools"
  func (r *Registry) CallMCP(ctx context.Context, name string,
      arguments json.RawMessage, caller string) MCPCallResult      // tools/call result
  func (r *Registry) ACPCommands(ctx when.Context) []ACPCommand    // availableCommands
  func (r *Registry) Manifest() Manifest                          // the whole catalogue as JSON
  ```

  * `MCPTool` is `{name, title, description, inputSchema, outputSchema,
    annotations, _meta}` with the MCP field names. `name` is the command
    ID. `MCPCallResult` is `{content, structuredContent, isError}`. A
    gate refusal or an argument error is `isError: true` with a message
    the model can act on, as MCP recommends.
  * These are plain Go structs with `json` tags. No MCP or ACP SDK is
    imported. A host that runs an MCP server, or pi-go handing its agent
    a "ui" tool set, wires them to its own transport.
  * `Manifest` is a versioned JSON document of every command, for docs,
    shell completion and other tools.

### 8. `when`: availability as expressions

```go
func Parse(src string) (Expr, error)
func MustParse(src string) Expr
type Expr struct{ /* compiled */ }
func (e Expr) Eval(c Context) bool
func (e Expr) String() string            // canonical form
func (e Expr) Keys() iter.Seq[string]    // the keys it reads
func Check(e Expr, known Keys) []error   // unknown keys, type mismatches

type Context interface{ Value(key string) (Value, bool) }
type Value struct{ /* bool, float64, string or []string */ }
type Map map[string]Value                 // a simple Context
func Layered(cs ...Context) Context      // first match wins: the focus path

// Key is a typed, documented context key.
type Key[T bool | int | float64 | string | []string] struct{ Name, Doc string }
func NewKey[T bool | int | float64 | string | []string](name, doc string) Key[T]
func (k Key[T]) Set(m Map, v T)
func (k Key[T]) Get(c Context) (T, bool)
```

* **Grammar,** VS Code's, with its precedence (highest first):

  ```text
  expr  = or
  or    = and { "||" and }
  and   = not { "&&" not }
  not   = "!" not | cmp
  cmp   = term [ ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) term
               | "=~" regex
               | [ "not" ] "in" key ]
  term  = key | number | 'quoted string' | bareword | true | false
        | "(" expr ")"
  ```

* **Semantics.** A key alone is true when it is set and not `false`, `0`,
  `""` or empty. A missing key compares unequal to everything, so
  `x != 'a'` is true when `x` is unset, as in VS Code. `in` tests
  membership in a list key. Regexes are Go RE2, compiled at parse time,
  so evaluation is linear and cannot be made slow by a crafted pattern.
  VS Code requires spaces around comparison operators; this parser does
  not, and accepts every expression VS Code accepts.
* **Limits.** At most 4 KiB of source and 64 levels of nesting. Beyond that
  `Parse` returns an error, so a user file cannot overflow the stack.
  (*A3: and at most 4 KiB of canonical form, so every expression `Parse`
  accepts round-trips.*)
* **Typed keys** document themselves. The workspace publishes:
  * `workspace.focusedPane`, `workspace.zoomed`, `workspace.hiddenPanes`;
  * `workspace.overlay` (the top overlay's ID) and `workspace.modal`;
  * `workspace.width` and `workspace.height`.

  Later records add `terminal.*` keys from `termcap`
  ([0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md))
  and `agent.busy`. `Check` reports a typo in a user's `when` at load
  time instead of a command that never appears.
* **The focus path.** A pane can implement
  `workspace.Contexter{ WhenContext() when.Context }`.
  `Workspace.WhenContext()` layers the top overlay's, then the focused
  pane's, then the workspace's own keys. The keymap's context precedence
  is the same layering.

### 9. Built-in commands, from day one

`workspace.Commands(w *Workspace, o ...CommandOption) []command.Command`
returns the workspace's own commands, closed over `w`:

| ID | Slash | Args | Danger |
| :--- | :--- | :--- | :--- |
| `workspace.focus` | `focus` | `pane` | UI |
| `workspace.focus.next`, `workspace.focus.prev` | `next`, `prev` | — | UI |
| `workspace.zoom` | `zoom` | `pane` (default focused) | UI |
| `workspace.toggle` | `toggle` | `pane` | UI |
| `workspace.resize` | `resize` | `pane`, `delta` (*A2: `split`, `delta`*) | UI |
| `workspace.layout.use` | `layout` | `name`: an enum of the layouts given in `WithLayouts` | UI |
| `workspace.layout.reset` | — | — (*A2: `SetState` of the zero `State`*) | UI |
| `workspace.state.get` | — | — | ReadOnly |
| `workspace.state.set` | — | `state` (layout `State` JSON) | UI |
| `workspace.panes` | `panes` | — | ReadOnly |
| `workspace.overlay.close` | `close` | — | UI |
| `workspace.theme.set` | `theme` | `background`: dark, light or auto (*A2: through `SetBackground`*) | UI |

The registry itself contributes `command.list` and `command.describe`
(ReadOnly), so an agent can discover what it may call, and `app.quit`
(UI), which only sends `QuitRequestMsg`. (*A5: none has a slash name, and
`command.list` lists what the caller can run now.*)

* `workspace.panes` returns each pane's ID, title, rectangle, focus and
  hidden state as `Value`. An agent can then see the screen's structure
  before it acts.
* `workspace.theme.set` uses `Workspace.SetTheme` from
  [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md).
* The workspace's existing key bindings keep working until the keymap
  record moves them onto these IDs.
* Every built-in is `Loop` mode and `Surfaces` without `CLI`, except
  `command.list` and `command.describe`.

### 10. `command/cli`: the same commands from the shell

```go
// Run runs args (os.Args[1:] in a real program) against r, writing to
// stdout and stderr, and returns an exit code. It never calls os.Exit.
func Run(ctx context.Context, r *command.Registry, args []string,
    stdout, stderr io.Writer, o ...Option) int
// Options: WithName(prog), WithConfirm(func(prompt string) bool), WithContext(when.Context)
```

* **Shape.** ID segments become words: `prog workspace state get`, or the
  dotted ID, `prog workspace.state.get`. Each schema property becomes a
  flag (`--delta 4`). Arrays repeat the flag. `--args '{…}'` passes raw
  JSON, and `--json` prints `Result.Value` as JSON instead of
  `Result.Text`.
* **Built-in verbs:** `list` (with `--json`, the manifest), `describe <id>`,
  `schema <id>` (the JSON Schema) and `help`.
* **Exit codes:** 0 success, 1 the command failed, 2 a usage or argument
  error, 3 refused by the gate.
* **A shell invocation is `Origin: CLI`.** `Destructive` commands ask
  `WithConfirm`, and without it need `--yes`. (*A9: through a per-request
  `Request.Gate`.*) There is no event loop, so
  `Result.Cmd` is ignored, and commands without `Surfaces&CLI` are not
  listed.
* **How a program offers both.** A program such as pi-go builds one
  registry. With no subcommand it starts its `tea.Program` and dispatches
  from `Update`. With one, it calls `cli.Run`. Help, flags and JSON output
  come from the same definitions.
* **A Cobra and fang adapter ships too** (owner question Q4).
  `command/cobra` builds a `cobra.Command` tree from the registry, with the
  same word paths, flags from the schemas, `--json` and exit codes as
  `command/cli`, and fang for styled help, man pages, completion and
  version. The registry stays the source; Cobra is only a front end.
  * AGENTS.md Dependencies requires a record naming
    `github.com/spf13/cobra` and `github.com/charmbracelet/fang` before
    either is required. That dependency record is not written yet, and the
    adapter waits for it to be accepted.
  * `command/cli` stays the standard-library path. A program that does not
    import `command/cobra` never compiles Cobra or fang in.
  * `depguard` allows the two modules in `command/cobra` only.

### 11. Versioning and conventions

* This lands after `termcap`, in the next minor release after
  [0005-PLAN-terminal-capabilities-and-services.md](0005-PLAN-terminal-capabilities-and-services.md).
  The owner tags. `v0` allows API change. *A2: that release is `v0.4.0`.*
* `command`, `when` and `command/cli` render nothing except CLI help, which
  is plain ASCII text. `command/cobra`'s help is fang's, and its output
  goes to the writers Cobra is given. CLI help is golden-tested at two
  widths, and `internal/conformance` covers the new packages: they write
  only to the writers they are given.

### Consequences

* Good, because one definition drives keys, palette, slash, help, shell and
  agents, so a new command appears everywhere at once.
* Good, because commands speak MCP, ACP, JSON Schema and VS Code's `when`.
  Any agent stack can drive a go-tui-lib program without new code here.
* Good, because the default gate keeps agents to read-only and
  presentation commands until a program installs a permission gate.
* Good, because argument schemas come from Go types, so the schema, the
  validation and the handler cannot disagree.
* Good, because pi-go's user and project commands, MCP prompts and ACP
  agent commands load through one path with predictable names.
* Neutral, because `command` imports `bubbletea` for `tea.Cmd`. A
  shell-only program links Bubble Tea, which every program using this
  library already does.
* Neutral, because only a subset of JSON Schema is enforced. It is the
  subset `SchemaOf` emits, and it is documented.
* Bad, because this is three new packages and an addition to `workspace`.
  The PLAN lands them in dependency order.
* Bad, because the front-matter format is this library's own subset, not
  YAML. A user who writes YAML features gets an error naming the line.
* Bad, because the `when` parser is new code that parses user input. It is
  fuzzed and limited in size and depth.
* Bad, because the Cobra and fang adapter puts two modules, and their
  requirements, in this module's `go.mod` for every consumer, though only
  importers of `command/cobra` compile them. It also waits on a dependency
  record.

### Confirmation

* `when`:
  * table tests for every operator and the precedence examples from the VS
    Code documentation;
  * `String()` round-trips: parsing the canonical form gives the same form
    (*A3: for every expression `Parse` accepts*);
  * `FuzzParse` finds no panic, no non-round-tripping canonical form and
    no evaluation panic on random contexts;
  * the size and depth limits are errors, not crashes.
* `command`:
  * `SchemaOf` golden schemas for representative structs, each checked to
    be valid JSON with `$schema` 2020-12;
  * strict decoding rejects unknown members, wrong types, missing required
    fields and out-of-range values with an `*ArgError` naming the path;
  * the registry's iterators are stable under concurrent `ReplaceSource`
    (`-race`), and `Version` rises on every change;
  * slash-name clashes rename the loaded command and report a `Conflict`;
  * the default gate refuses an agent's `Mutating` command, and allows its
    `ReadOnly` and `UI` ones;
  * `AllowAlways` is remembered, and `RejectOnce` is not;
  * audit records mask `secret` arguments;
  * `Exclusive` cancels the previous async run.
* Exporters: golden JSON for `MCPTools`, `CallMCP` and `ACPCommands`, and a
  test that the field names match the MCP 2025-11-25 and ACP examples
  quoted in this record.
* `LoadDir`: golden commands from a test tree; each front-matter error
  names the file and line; a symlink out of the root is refused.
* `command/cli`: the golden help, flag parsing from schemas, `--json`, exit
  codes, and the `--yes` requirement.
* `command/cobra`, once its dependency record is accepted: the same requests,
  outputs and exit codes as `command/cli` for the same arguments, and
  `depguard` refusing Cobra and fang anywhere else.
* `workspace.Commands`: each built-in drives a real workspace through
  `Dispatch` and gives the same frame as the method it wraps.
* Mutation proofs for each package's key invariants, seen failing on a
  scratch copy. Pre-add, `-race`, `LC_ALL=C` and the Windows host pass. CI
  is green on the push.

## Pros and Cons of the Options

### A. `command`, `when` and `command/cli`

* Good, because every surface reads one definition.
* Good, because the standards are met at the edges, with plain structs.
* Good, because new sources and surfaces are additive: a loader, an
  exporter or an adapter.
* Bad, because it is the most code of the four, and `when` is a parser to
  maintain.

### B. A table per surface

* Good, because there is nothing new to learn.
* Bad, because the same action is described four or five times, and the
  descriptions drift.
* Bad, because there is no place to put agent access, danger or
  arguments, so the owner's agentic goal is not met.

### C. Cobra as the registry

* Good, because Cobra and fang give styled help, completion scripts and man
  pages for free.
* Bad, because it is a new module, and fang a second, each needing a record.
* Bad, because a `cobra.Command` has flags, not a JSON Schema, and no
  availability, danger, scope or async mode. The TUI would bolt those on
  through annotations.
* Bad, because it is shaped around one invocation per process, not a
  long-lived registry that changes at run time (ACP and MCP updates).

### D. CEL or expr-lang for `when`

* Good, because a mature engine would parse and evaluate more than `when`
  needs.
* Bad, because it is a new module, and a large one for a dozen operators.
* Bad, because its syntax is not VS Code's, so keybinding files written for
  VS Code's grammar would not carry over.

## Owner questions

*Answered 2026-10-02* (picked from options): Q1 "Markdown, strict front
matter"; Q2 "ReadOnly and UI free"; Q3 "workspace.Commands(w)"; Q4 "both 1
and 2", that is, the standard-library `command/cli` now and a Cobra/fang
adapter now. Q1, Q2 and Q3 are the recommendation. Q4 is not: the
recommendation deferred the adapter until a consumer asked. Decision
Drivers, Decision Outcome, §1, §10, Consequences and Confirmation were
revised to add `command/cobra`, which lands only after its own dependency
record for `github.com/spf13/cobra` and `github.com/charmbracelet/fang` is
accepted.

* **Q1. Command file format.** Recommended: Markdown with this record's
  strict front-matter subset, which needs no module. The alternatives are
  YAML front matter (a YAML module) or TOML files as gemini-cli uses (a
  TOML module), each needing a record.
* **Q2. Agent policy.** Recommended: agents may run `ReadOnly` and `UI`
  commands without asking, and must pass the gate for `Mutating` and
  `Destructive` ones; with no gate installed they are refused. The
  stricter alternative asks even for `UI` commands, so an agent cannot
  move focus or zoom without consent.
* **Q3. Where the workspace's commands live.** Recommended:
  `workspace.Commands(w)`, beside the methods they wrap. The alternative is
  a `command/builtin` package that imports `workspace`, which keeps
  `workspace` free of `command` but splits each feature across two
  packages.
* **Q4. The shell adapter.** Recommended: the standard-library
  `command/cli` now, and a Cobra or fang adapter only when a consumer asks,
  under its own record.

## Amendments

### A1 (2026-10-02): native Cobra and Kong front ends as nested modules

*Status: accepted (2026-10-03).* Its steps are Steps 4, 10 and 11 of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md).

**Found.** On 2026-10-02 the owner asked whether the library is neutral
between Cobra, Viper, Kong or no framework, and said it "needs to support,
and be fully optimized for kong". The owner then asked for:

> a native cobra module, a native kong module, each fully optimized and
> full-featured api, only the one compiled in as needed

and decided:

> Nested modules. Glamour as nested module. Go.work in repo.

[0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
decides the module layout, the release order and the gates. This amendment
decides the two front ends' APIs, and §4's tags, which as written would be
misparsed by Kong. The facts are in
[0010-REPORT-nested-modules-and-adapter-sources.md](../reports/0010-REPORT-nested-modules-and-adapter-sources.md)
§6 (Cobra and pflag), §7 (fang) and §8 (Kong).

**What changes in the decision.** §4 and §10 above are left as written;
the items below supersede them where they differ.

* **§1 and §10, the front ends.** Three front ends read one registry:

  ```text
   command/cli       the standard-library front end (root module, no module added)
   command/cobracmd  nested module github.com/maccavelli/go-tui-lib/command/cobracmd
                                             → command, theme, glyph, spf13/cobra
   command/kongcmd   nested module github.com/maccavelli/go-tui-lib/command/kongcmd
                                             → command, theme, glyph, alecthomas/kong
  ```

  * `command/cobra` becomes the nested module `command/cobracmd`, package
    `cobracmd`. The package name does not shadow `cobra` at an import site
    that uses both (0010-MADR §1).
  * `command/kongcmd`, package `kongcmd`, is new.
  * `command/cli` stays in the root module, the zero-dependency path.
  * A program imports one front end. The others, and their dependencies,
    never reach its build, its `go.sum` or its module graph (REPORT §2).
  * §10's "Cobra and fang" and its dependency record are replaced by
    0010-MADR, which names Cobra and Kong.
* **fang is not used** (0010-MADR Q1, answered 2026-10-02). fang
  queries the terminal on every help and error render, reads stdout's size,
  and writes its man page to `os.Stdout`, with no option to stop any of
  them (REPORT §7). `cobracmd` gives what fang gave, inside the rules:
  * **Charm-styled help** through `SetHelpFunc` and `SetUsageFunc`, drawn
    with the library's `theme` and `glyph` at the width the caller gives,
    across 0001-MADR §6's colour, charset and width matrix;
  * **version** through `cobra.Command.Version`, set by the program;
  * **man and Markdown pages** from an optional subpackage,
    `cobracmd/docs`, which calls `cobra/doc`'s `GenMan` and
    `GenMarkdownCustom` with the caller's writer. It sets
    `DisableAutoGenTag` and takes a fixed date, because both functions
    otherwise stamp the time (REPORT §6), and golden files would change on
    every run. md2man and YAML are compiled only by importers of
    `cobracmd/docs`.
* **§4, tags aligned with Kong.** One argument struct then drives the
  registry's schema, slash commands, the palette, agents, `command/cli`,
  Cobra flags and a Kong grammar.
  * **Why.** Kong reads bare struct tags, and makes any field with an `arg`
    tag positional, whatever the tag's value (`tag.go:260`, REPORT §8).
    §4's `arg:"min=-200,max=200,pos=0"` would therefore turn every
    constrained field into a positional argument in a Kong grammar.
  * **The vocabulary:**

    | Tag | Meaning here | Kong | Cobra and `command/cli` |
    | :--- | :--- | :--- | :--- |
    | `json:"name,omitzero"` | the property name; required unless a pointer, `omitempty` or `omitzero` | — (the adapter sets `name`) | the flag name |
    | `help:"…"` | the description; replaces `doc` | `help` | the flag's usage |
    | `default:"…"` | the default; replaces `arg:"default=…"` | `default` | the flag's default |
    | `enum:"a,b"` | the allowed values, comma-separated; replaces `arg:"enum=a\|b"` | `enum` | a custom `pflag.Value` and its completion |
    | `arg:""` | positional, in field order; replaces `arg:"pos=N"` | `arg` | a positional argument |
    | `short:"d"` | a one-letter flag | `short` | the flag's shorthand |
    | `hidden:""` | in the schema, not in help | `hidden` | `MarkHidden` |
    | `placeholder:"PANE"` | the value's name in help | `placeholder` | the usage's value name |
    | `group:"…"` | the flag's help group | `group` | not used: Cobra groups commands, not flags |
    | `schema:"min=…,max=…,minLen=…,maxLen=…,secret"` | registry-only constraints | ignored, but kept readable through `Tag.Get` (`tag.go:58-59`) | checked by the registry's decoder |

  * **Enums follow Kong's rule:** a scalar `enum` field must be required
    or have a `default` (`tag.go:326-329`). `SchemaOf` returns an error
    for one that is neither, so a struct the registry accepts is one Kong
    accepts.
  * **§4's example becomes:**

    ```go
    type resizeArgs struct {
        Delta int           `json:"delta" arg:"" help:"cells to grow; negative shrinks" schema:"min=-200,max=200"`
        Pane  layout.PaneID `json:"pane,omitzero" help:"pane to resize; default the focused pane" placeholder:"PANE"`
    }
    ```

    `Delta` is the first positional because it is the first `arg` field;
    `/resize 4` still fills it.
  * The JSON Schema, strict decoding, slash parsing and the supported types
    are unchanged.
* **`cobracmd`, the native Cobra front end.**

  ```go
  // New builds a root command holding every CLI command of r.
  func New(r *command.Registry, o ...Option) (*cobra.Command, error)
  // Mount grafts r's commands into a program's existing Cobra tree.
  func Mount(parent *cobra.Command, r *command.Registry, o ...Option) error
  // Run executes root against args with the caller's streams, and returns
  // command/cli's exit code. It never calls os.Exit.
  func Run(ctx context.Context, root *cobra.Command, args []string,
      stdin io.Reader, stdout, stderr io.Writer) int
  // Options: WithName, WithConfirm, WithContext(when.Context),
  // WithTheme(theme.Theme), WithGlyphs(glyph.Set), WithWidth(int)
  ```

  * **The tree.** A dotted ID becomes nested commands: `Use` is the last
    segment, and `Aliases` holds the command's slash aliases. The registry's
    categories become Cobra groups, added with `AddGroup` before any child,
    because Cobra panics on an undefined `GroupID` (`command.go:1205-1210`).
    `Annotations` carries the ID, the danger level and the surfaces.
    `Hidden` and `Deprecated` come from the command.
  * **Flags come from the JSON Schema** through pflag. An enum is a custom
    `pflag.Value` with `FixedCompletions`, because pflag has no enum type
    (REPORT §6). A required property is `MarkFlagRequired`. Positional
    properties set `Args`. Where the schema expresses them, mutually
    exclusive and required-together properties use
    `MarkFlagsMutuallyExclusive` and `MarkFlagsRequiredTogether`.
  * **Completion comes from the registry:** `ValidArgsFunction` and
    `RegisterFlagCompletionFunc` serve enums and the registry's completion
    providers. Scripts for bash, zsh, fish and PowerShell come from Cobra's
    generators, written to the caller's writer.
  * **The same verbs, flags and exit codes as `command/cli`:** `--args`,
    `--json`, `--yes`, `list`, `describe`, `schema`, and exit codes 0, 1, 2
    and 3. Every invocation is `Origin: CLI`, and runs `RunE` through the
    registry, so the gate and the audit trail apply as in §7.
  * **Rules** (0010-MADR §6, REPORT §6):
    * `Run` always calls `SetArgs`, `SetOut`, `SetErr` and `SetIn`, because
      Cobra otherwise reads `os.Args` and writes to `os.Stdout` and
      `os.Stderr`; it sets `SilenceErrors` and `SilenceUsage`, calls
      `ExecuteContextC`, and turns the error into an exit code.
    * It never calls `cobra.CheckErr`, and never sets Cobra's process-wide
      variables (`EnableTraverseRunHooks`, `EnablePrefixMatching`,
      `MousetrapHelpText`). The package documentation tells a Windows
      program to set `MousetrapHelpText = ""` itself if it may be started
      from Explorer, where Cobra otherwise exits.
    * Build the tree once per process. Cobra keeps flag completions in a
      process-wide map keyed by flag, with no delete, and refuses only a
      second registration for the same flag (`completions.go:38-41`,
      `178`), so every rebuild adds entries that are never freed. The
      package documentation says so.
    * A failing hidden `__complete` command writes to `os.Stderr` inside
      Cobra, which no setter reaches. It is documented, not hidden.
* **`kongcmd`, the native Kong front end.**

  ```go
  func New(r *command.Registry, o ...Option) (*Adapter, error)
  // Options returns the kong options that mount r's commands into a
  // program's own kong.New, beside its static commands.
  func (a *Adapter) Options() []kong.Option
  // Parser returns a parser of r's commands alone.
  func (a *Adapter) Parser(o ...kong.Option) (*kong.Kong, error)
  // Run parses args, runs the selected registry command, and returns its
  // exit code with handled true. When the program's own command is
  // selected, it returns the context with handled false.
  func (a *Adapter) Run(ctx context.Context, p *kong.Kong, args []string) (kctx *kong.Context, code int, handled bool)
  // Resolver gives a Kong parser values from the registry's settings.
  func (a *Adapter) Resolver() kong.Resolver
  // Options: WithName, WithConfirm, WithContext(when.Context),
  // WithTheme(theme.Theme), WithGlyphs(glyph.Set), WithWidth(int)
  ```

  * **Mounting.** Each top-level ID segment is a `kong.DynamicCommand`,
    Kong's mechanism for commands known only at run time (`options.go:100`).
    A program keeps its own static grammar and adds `a.Options()` to its
    `kong.New`.
  * **Grammars are built per command** with `reflect.StructOf` from the
    schema: scalars, enums with their default or required, slices with
    `sep`, `hidden`, `group`, and nested `cmd` fields for dotted IDs. Each
    field carries custom `id` and `danger` tags, which Kong keeps readable.
    This was run in probes: values decode, enums are checked, nested
    commands select (REPORT §8).
  * **Dispatch is the adapter's.** A `StructOf` type has no methods, so
    `kong.Context.Run` cannot find one (REPORT §8). `Run` calls `Parse`,
    then dispatches from `ctx.Selected()` through the registry, as
    `Origin: CLI`, so the gate and the audit trail apply.
  * **`Bind` and `BindTo`** give a program's own static commands the
    registry and the invocation context, so a hand-written Kong command can
    dispatch a registry command too.
  * **Groups and help.** The registry's categories become
    `kong.ExplicitGroups`. `PostBuild` sets `Hidden` and `Help` from the
    command. Help is printed by the adapter's own `kong.Help(HelpPrinter)`,
    with the library's theme and glyphs at the caller's width, because
    Kong's printer takes its width from `$COLUMNS` or an ioctl on the
    writer (REPORT §8).
  * **Settings.** `Resolver` is the Kong form of configuration, Kong's
    answer to Viper: it lets a Kong program fill flags from the values a
    later configuration record stores. No `env` tag is generated unless the
    program asks, because `env` reads the process environment.
  * **Completion** is generated in this module from the registry, for
    bash, zsh, fish and PowerShell, written to the caller's writer, with
    no further module (0010-MADR Q2, answered 2026-10-02).
  * **The same verbs, flags and exit codes as `command/cli`.**
  * **Rules** (0010-MADR §6, REPORT §8):
    * Every parser the adapter builds or mounts into gets `kong.Name`,
      because the default is `os.Args[0]`, and `kong.Writers` with the
      caller's writers.
    * Its `Exit` panics a sentinel that `Run` recovers into an exit code.
      An `Exit` that returns lets Kong keep parsing after `--help`; this
      was observed in a probe, where `--help` printed help and `Parse`
      then failed.
    * It never calls `kong.Parse`, `Fatalf` or `FatalIfErrorf`, which read
      `os.Args` or exit.
* **One conformance test for three front ends.** The same requests, run
  through `command/cli`, `cobracmd` and `kongcmd`, build the same
  `Request`, write the same `--json` output and return the same exit code.
  Each nested module carries a copy of the cases, because a test cannot
  import across modules' test files.

**What does not change.**

* Option A, the registry, the command, handlers, the gate, the audit trail,
  `when`, the loaders and the exporters.
* The answers to Q1 to Q3, and Q4's "both": the standard-library front end
  and a Cobra front end ship, now with a Kong one beside them.
* `command/cli`'s shape, verbs and exit codes, which both nested front
  ends copy.

**Consequences, changed.** The Bad bullet "the Cobra and fang adapter puts
two modules … in this module's `go.mod` for every consumer" no longer
holds: Cobra and Kong live in nested modules, and the root's `go.mod` never
names them (0010-MADR, REPORT §2). In its place:

* Bad, because a front end is released apart from the root, after the root
  release whose API it uses (0010-MADR §3).
* Bad, because the tag vocabulary of §4 changes before it ships, and its
  names are Kong's, not this library's own.

**Versioning.** `command/cli` ships in the root's release, as before.
`cobracmd` and `kongcmd` are tagged `command/cobracmd/v0.1.0` and
`command/kongcmd/v0.1.0`, each after the root release that contains Steps
1 to 9 of the PLAN, and each requires that root version (0010-MADR §3).

**Owner questions for A1.** None beyond 0010-MADR's Q1 (fang) and Q2 (Kong
completion), which this amendment follows. Both were answered on
2026-10-02 as recommended: no fang, and Kong completion generated here.

### A2 (2026-10-05): the audit after `v0.3.0`

*Status: accepted (2026-10-05).* The owner asked for the codebase to be
assessed for current facts and this record's PLAN evaluated against it, to
make the PLAN accurate and actionable. The PLAN,
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md), was revised
the same day to match.

**Found, read-only, against the tree at `v0.3.0` (`81a61de`) and Go
1.27.1.** Each item names what this record says and what holds.

1. **Go 1.27.1 (§ Context, "Go 1.27.1 gives everything").** Holds.
   `encoding/json/v2` is in `$GOROOT/api/go1.27.txt`; its sources carry
   `//go:build goexperiment.jsonv2`, and that experiment is on by default:
   a scratch program built with no `GOEXPERIMENT` decoded with
   `json.RejectUnknownMembers(true)` and refused an unknown member.
   `reflect.Type.Fields() iter.Seq[StructField]` and `reflect.Value.Fields`,
   `errors.AsType`, and `(*os.Root).FS` are in the API files. Generic
   methods are in use already (`Workspace.PaneAs[T]`).
2. **The workspace's methods (§ Context, §9).**
   * `Resize(sep string, delta int)` moves a **named split's separator**,
     not a pane (`layout.State.Resize` is keyed by split name; only a
     `Separator` with `Resizable` moves). §9's `workspace.resize` with a
     `pane` argument, and §4's and A1's `resizeArgs` example with a `Pane`
     field, do not match. The command takes `split` and `delta` (below).
   * There is no method for `workspace.layout.reset`. `SetState` of the
     zero `layout.State` is that reset: "The zero State changes nothing."
   * `workspace.theme.set` with `auto` has nothing to call. A workspace
     follows the terminal unless built `WithTheme`; `SetTheme` on a
     following workspace is rebuilt over at the next profile or background
     message, and nothing returns a workspace to following. Owner question
     Q5, below.
   * The other built-ins have their methods, with these signatures:
     `Focus(id)`, `FocusNext()`, `FocusPrev()`, `Zoom(id)` (zooms, or
     restores when `id` is zoomed or empty), `Toggle(id)`, `SetLayout(root)`,
     `State()`, `SetState(s)`, `Pop()`, `Focused()`, `Overlays()` and
     `Plan()` (rectangles in `Plan.Panes`, hidden panes in `Plan.Hidden`).
     `Push`, `Send`, `SendOverlay`, `SetPane`, `Broadcast`, `Panes`,
     `PaneAs` and `SetTheme` exist too; `layout` has the four sidebar
     presets.
   * `layout.State` holds `Resize map[string]int`. §4's supported types have
     no map, so `workspace.state.set` could not take a `State`. `SchemaOf`
     also supports `map[string]T`, as an object whose
     `additionalProperties` is `T`'s schema.
3. **MCP (§ Context, §7).** The current revision is **2026-07-28**, not
   2025-11-25 (the specification's changelog,
   <https://modelcontextprotocol.io/specification/2026-07-28/changelog>,
   fetched 2026-10-05). For this record: the `Tool` object and its
   annotations are unchanged; `inputSchema` and `outputSchema` may use any
   JSON Schema 2020-12 keyword; every result carries a required
   `resultType`, `"complete"` for an ordinary one; `tools/list` and
   `prompts/list` results carry `ttlMs` and `cacheScope`, and servers
   SHOULD list tools in a deterministic order; list-change notifications
   reach a client through `subscriptions/listen`; sessions and the
   `initialize` handshake are gone. Owner question Q6, below.
4. **ACP (§ Context, §7).** Unchanged: `AvailableCommand` is `{name,
   description, input: {hint}}`, a command runs as `/name args` prompt
   text, and `PermissionOption.kind` is `allow_once`, `allow_always`,
   `reject_once` or `reject_always`, with the outcome `selected` and an
   `optionId`, or `cancelled` (fetched 2026-10-05).
5. **`termcap` (§8).** It shipped in `v0.3.0`. `terminal.*` context keys
   are still a later record's; nothing in `termcap` imports `when`.
6. **Versioning (§11, A1).** The root's next minor is `v0.4.0`. It carries
   the PLAN's Steps 1 to 9; `command/cobracmd/v0.1.0` and
   `command/kongcmd/v0.1.0` follow it and require it.
7. **The modules (A1).** 0010's tooling is in place: `go.work` lists `.`;
   `scripts/go-modules.sh` feeds CI's test matrix, so a module added with
   `go work use` is tested on three operating systems with no workflow
   change; `.golangci.yml` already confines Cobra and pflag to
   `command/cobracmd`, and Kong to `command/kongcmd`;
   `internal/conformance` scans every package of every module it finds.
   The adapter steps need only their own directories, `go.work`, and the
   documents.

**Decision.**

* **§9, corrected to the workspace's API.** `workspace.resize` takes
  `split` (a resizable separator of the current plan; any other is an
  argument error) and `delta`. `workspace.layout.reset` is `SetState(
  layout.State{})`. §4's and A1's example becomes:

  ```go
  type resizeArgs struct {
      Split string `json:"split" arg:"" help:"the split whose separator moves" placeholder:"SPLIT"`
      Delta int    `json:"delta" arg:"" help:"cells to give the pane before it; negative takes" schema:"min=-200,max=200"`
  }
  ```

  so `/resize sidebar 4` fills both.
* **§4, maps.** `SchemaOf` supports `map[string]T`.
* **Q5, `workspace.theme.set`** (the owner, picked from options,
  2026-10-05, "Add Workspace.SetBackground", the recommendation). The
  workspace gains `SetBackground(bg theme.Background) tea.Cmd`:
  * `theme.Dark` or `theme.Light` pins the background, rebuilds the theme
    through its builder, and keeps it pinned when the terminal later
    reports another background;
  * `theme.Unknown` returns to following: the theme is rebuilt from the
    background the terminal last reported, and the command is
    `tea.RequestBackgroundColor`, unless the workspace was built
    `WithoutBackgroundQuery`;
  * on a workspace built `WithTheme`, which follows nothing, it records
    the choice and changes nothing visible.

  `workspace.theme.set` takes `background`, one of `dark`, `light` and
  `auto`, and calls it.
* **Q6, the MCP revision** (the owner, picked from options, 2026-10-05,
  "2026-07-28", the recommendation). The exporters target 2026-07-28:
  `MCPCallResult` carries `resultType: "complete"`, which an earlier
  client ignores; `MCPTools` lists in ID order; the list envelope (`ttlMs`,
  `cacheScope`) and `subscriptions/listen` are the host's, and the guide
  says so. The golden exports and the field-name test pin 2026-07-28's
  schema.
* **§11, versioning,** as found in item 6.

**Unchanged.** Option A; every other built-in; the registry, handlers,
gate, audit trail, `when`, the loaders, the ACP exporter, `command/cli`;
A1's front ends, tags and rules.

**Owner questions for A2.** *Answered 2026-10-05* (picked from options):
Q5 "Add Workspace.SetBackground"; Q6 "2026-07-28". Both are the
recommendation.

* **Q5.** What `workspace.theme.set` does, given the workspace's theme
  API. Recommended: the `SetBackground` above. The alternatives were to
  drop the command until a theme record, or to offer dark and light only
  through `SetTheme`, which a following workspace undoes at the next
  background message.
* **Q6.** Which MCP revision the exporters target. Recommended:
  2026-07-28. The alternative was to keep 2025-11-25, which a 2026-07-28
  client still reads, treating a result without `resultType` as complete.

### A3 (2026-10-05): the canonical form is bounded too

*Status: accepted (2026-10-05).* Found while executing Step 2 of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md), recorded
there as deviation D1.

**Found.** §8 says `String()` round-trips and limits the source to 4 KiB;
together they are false. The canonical form can be longer than the source
it came from: it adds spaces around operators (`a==1` prints `a == 1`),
and it quotes a bareword and doubles each backslash in it. `FuzzParse`
found a source under 4 KiB whose canonical form is 6718 bytes, which
`Parse` then refuses.

**Decided.** `Parse` refuses an expression whose canonical form is longer
than `MaxSource`, as it refuses a source that is. The error names the
canonical form's length. Every expression `Parse` accepts then
round-trips, because a canonical form prints as itself. The cost is one
`String()` per `Parse`, at load time.

**Changed.** §8's limits and §Confirmation's round-trip item, annotated
in place.

**Owner question for A3.** *Answered 2026-10-05* (picked from options):
"Bound the canonical form too", the recommendation. The alternative was to
limit the source only, and to have the round-trip property, and
`FuzzParse`, hold only for an expression whose canonical form fits.

### A4 (2026-10-05): unset danger and origin, the registry's errors, and the surfaces of a click and of the program

*Status: accepted (2026-10-05).* Found before writing Step 3 of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md), recorded
there as deviations D4, D5 and D6. §2, §3, §5 and §7 name the
enumerations, `Dispatch` and `Run` but leave three things open.

**Found.**

1. **Zero values.** If `ReadOnly` were `Danger`'s zero value, a built-in
   that forgot to declare its danger would run for an agent with no
   gate, against §7's "the default policy is safe". If `OriginKey` were
   `Origin`'s zero, a request that forgot its origin would get a user's
   rights.
2. **Errors.** §10's exit code 3, "refused by the gate", needs
   `command/cli`, another package, to tell a refusal from other errors.
   Neither §3 nor §5 names an error a caller can test.
3. **Surfaces.** `Dispatch` checks `Surfaces` against the request's
   origin. Five origins have a surface of their own; `OriginMouse` and
   `OriginProgram` have none.

**Decided.**

1. `Danger` and `Origin` start at 1, and their zero value means "not
   declared". `Register` refuses a command whose `Danger` is zero; `New`
   and the loaders set one. `Dispatch` and `Run` refuse a request whose
   `Origin` is zero. `Kind`'s zero is `Action` and `Mode`'s is `Loop`,
   the defaults §2 and §3 name, and a zero `Surfaces` is every surface
   (§2).
2. `command` exports three sentinel errors, wrapped with detail and
   tested with `errors.Is`: `ErrUnknown` (no command has the ID),
   `ErrUnavailable` (its `When` is false, or its `Surfaces` exclude the
   origin) and `ErrRefused` (the policy or the gate said no).
3. `OriginMouse` needs `SurfaceKey`, because a click is a direct binding,
   as a key is. `OriginProgram`, the program running its own command, is
   checked against `When` only.

**Changed.** §2's danger table, annotated in place.

**Owner questions for A4.** *Answered 2026-10-05* (picked from options),
each the recommendation:

* "Zero means unset; refuse it". The alternatives were to treat an unset
  `Danger` as `Mutating` and an unset `Origin` as `OriginAgent` without
  refusing, or plain `iota`, where an unset `Danger` is `ReadOnly`.
* "Three sentinel errors". The alternatives were one `*Error` type with a
  code, or no exported error until Step 8.
* "Mouse as Key; Program unchecked". The alternatives were to check
  neither against `Surfaces`, or to add a `SurfaceMouse`.

### A5 (2026-10-05): `New` without a danger, the registry's own commands, and tags with no JSON Schema keyword

*Status: accepted (2026-10-05).* Found before writing Step 4 of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md), recorded
there as deviations D7–D10.

**Found.**

1. A4 has `Register` refuse an undeclared `Danger`, and says `New` sets
   one, without saying what `New` does when its caller names none.
2. §9 and the PLAN give `command.list`, `command.describe` and `app.quit`
   no slash names, and do not say whether that is deliberate.
3. Only a command's JSON Schema reaches the audit trail, `command/cli`
   and the Cobra and Kong front ends. A1's `secret`, `short`,
   `placeholder`, `group` and `hidden` have no JSON Schema keyword, so
   without a rule they are read and lost.
4. §9 says `command.list` lets an agent discover what it may call, and
   the PLAN's test says it "lists every command once", which leaves open
   whether that means every command or the ones the caller can run.
5. A1's `arg:""` makes a property positional, and has no JSON Schema
   keyword either; slash parsing, the shell and the front ends need it and
   its order. Item 3's question named four tags and missed this one.

**Decided.**

1. `New` returns an error when no `WithDanger` is given. The loaders
   still default to `Mutating` (§6).
2. The three have no slash name. A program that wants `/quit` registers
   its own command with its own word, so no built-in word can clash with
   a program's.
3. `secret` is the standard `"writeOnly": true`, which audit masking
   reads. `short`, `placeholder`, `group` and `hidden` go in one
   extension keyword, written only when one is set:
   `"x-cli": {"short": "d", "placeholder": "PANE", "group": "g",
   "hidden": true}`. JSON Schema 2020-12 treats an unknown keyword as an
   annotation, so validators and MCP clients ignore it.
4. `command.list` returns what `Available` would for the caller: the
   commands that are not hidden, are offered on the caller's surface, and
   whose `When` holds in the caller's context. A request from the program
   is not filtered by surface, as A4 has it.
5. A positional property carries `"arg": true` in its `x-cli` object.
   Positionals are taken in the order of `properties`, which `SchemaOf`
   writes in field order and the registry reads in document order.

**Changed.** §4's tag list and §9's built-ins, annotated in place.

**Owner questions for A5.** *Answered 2026-10-05* (picked from options),
each the recommendation:

* "Return an error". The alternative was to default to `Mutating`.
* "None". The alternative was `/commands`, `/describe` and `/quit`.
* "writeOnly + one x-cli object". The alternatives were a separate `x-`
  keyword for each tag, or to drop the four CLI-only tags.
* "What the caller can run now". The alternatives were every command that
  is not hidden, or every command.
* `"arg": true` in `x-cli`. The alternative was one `x-cli` list of the
  positional names on the object.

### A6 (2026-10-05): loading, names and clashes

*Status: accepted (2026-10-05).* Found before writing Step 5 of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md), recorded
there as deviations D13–D22. §6 names the loaders and the clash rule and
leaves these open.

**Decided.**

1. **`PromptGetter`** is `func(ctx context.Context, name string, args
   map[string]string) (string, error)`. The host calls its MCP client and
   joins the result's text; `FromMCPPrompts` knows the server, so the
   getter is one per server.
2. **A name that is not an ID segment** (an MCP `github_issue`, an ACP
   `Plan Mode`, a file `Review.md`) is mapped to one: lowercased, each run
   of other characters one `-`, the ends trimmed. The slash name is that
   segment. When two names map to one segment, or a name maps to nothing,
   the later one is refused and reported.
3. **`Conflict` gains `Err error`.** A command `ReplaceSource` cannot load
   (a malformed ID, one outside the source's namespace or held by another
   source, a bad `when`, no danger) is a `Conflict` with `Err` set and
   `Renamed` empty.
4. **A loaded file's slash name** is its path with `:` between parts
   (`git/commit.md` is `git:commit`), as gemini-cli does, unless its front
   matter's `slash:` says otherwise. It becomes `<prefix>:<name>` only on
   a clash.
5. **A placeholder's property** is named as written: `$FOCUS` is `FOCUS`.
6. **Placeholders are positional,** with `"arg": true` (A5), in the order
   each first appears in the body; `name=value` still works.
7. **Built-ins win whatever the order.** `Register` takes a slash name a
   loaded command holds, and the loaded command is renamed and reported.
8. **`ConflictMsg` arrives through `Watch`.** Its command returns
   `ChangedMsg`, and with it a `ConflictMsg` when the change it woke for
   renamed or refused a command.
9. **An ACP forward keeps the agent's name** in `Meta["acp"] = {"name":
   …}`; a `Forward`'s text uses it when present, otherwise the slash name.
10. **A renamed name that is also taken is dropped:** the command loads
    without it, reported as a `Conflict` whose `Renamed` is empty.
11. **A file that uses `$ARGUMENTS` takes any slash tail.** Found while
    building, after items 1–10: such a file's schema refused words past
    its positionals, so `/notes 100` failed for a file whose only
    placeholder is `$ARGUMENTS`. Its object schema is marked `"x-cli":
    {"rest": true}`; `ParseSlash` fills the positionals and leaves any
    further words out of the arguments, in `Raw`, which `$ARGUMENTS`
    takes whole. Other commands still refuse stray words.

**Changed.** §6's clash rule and placeholder notes, annotated in place.

**Owner questions for A6.** *Answered 2026-10-05* (picked from options),
each the recommendation. The alternatives were:

1. a getter that returns MCP messages, with two more exported types;
2. refusing names that are not segments;
3. `ReplaceSource` returning an error too, or dropping silently;
4. always prefixing loaded slash names;
5. lowercased property names;
6. `name=value` only;
7. `Register` returning an error, so built-ins must come first;
8. the host building `ConflictMsg` from `ReplaceSource`'s result;
9. using `Title` as the forwarded name;
10. refusing the command;
11. no schema for a file whose only placeholder is `$ARGUMENTS`, and
    stray words refused when it has others.

### A7 (2026-10-05): `Loop` commands run from off the loop, ACP export, and one command description

*Status: accepted (2026-10-05).* Found before writing Step 6 of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md), recorded
there as deviations D24–D26. The first was found while building Step 3
and left open for this step.

**Found.**

1. §3 has `Run` serve "the shell, agents and tests", for any mode, and
   §7's `CallMCP` runs as an agent. An agent's tool call arrives on its
   own goroutine, so a `Loop` command run through `Run` touches the
   workspace off the event loop, which §3 makes the reason for `Loop`.
2. §7 names `ACPCommands` without saying which commands it exports or
   under what name. ACP's available commands are what a user types as
   `/name`.
3. §7's `Manifest` describes each command, and the registry's own
   `command.list` and `command.describe` already did so with an unexported
   type of nearly the same fields.

**Decided.**

1. **`WithLoop(send func(tea.Msg))`**, a registry option: the host passes
   `program.Send`. With it, `Run`, and so `CallMCP`, sends a `LoopMsg`
   for a `Loop` command and waits until the loop has run it, or until the
   context ends. The host's `Update` returns `msg.Run()` for a `LoopMsg`,
   which runs the handler on the loop and gives its `Result.Cmd` to the
   program; `Run` then returns the result without the `Cmd`, which has
   gone to the program already. Without `WithLoop`, `Run` runs a `Loop`
   command on its caller's goroutine, as before, which suits the shell
   and tests. With `WithLoop` set, `Update` must use `Dispatch`, not
   `Run`, or it waits for itself.
2. **`ACPCommands` exports slash commands only**, under their slash
   names, with `ArgHint` as `input.hint`. A command without a slash name
   is left out, because a user cannot type it.
3. **`ManifestCommand` is the one exported description of a command.**
   `command.list` and `command.describe` return it, and the unexported
   type is removed.

**Changed.** §3's `Loop` note, annotated in place.

**Owner questions for A7.** *Answered 2026-10-05* (picked from options),
each the recommendation. The alternatives were:

1. to document only that the host calls `CallMCP` from its loop, where a
   gate that waits for the user cannot be used, or to defer to Step 7;
2. to export every agent-surface command, by ID when it has no slash
   name;
3. to keep both types.

### A8 (2026-10-06): the workspace's hidden panes, unknown panes, and the panes list

*Status: accepted (2026-10-06).* Found before writing Step 7 of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md), recorded
there as deviations D27–D29. §8 and §9 name the key and the commands and
leave these open.

**Decided.**

1. **`workspace.hiddenPanes` is `Plan().Hidden`:** every pane the layout
   knows and is not showing, whether the user hid it, a responsive rule
   dropped it, or it was squeezed out. It matches what `workspace.panes`
   reports as hidden.
2. **A pane the layout does not know is an argument error** for
   `workspace.zoom` and `workspace.toggle`, as it is for `workspace.focus`
   and `workspace.resize`. "Known" means placed or in `Plan().Hidden`.
   The workspace's own `Zoom` and `Toggle` methods are unchanged.
3. **`workspace.panes` lists** the focus ring first, then placed panes
   outside the ring (a footer that takes no focus) in tree order, then
   `Plan().Hidden` in its order. Panes the layout tree never mentions are
   left out.
4. **`workspace.resize`'s `split`** (found while building, after items
   1–3). A2 took a split's name for what `Resize` moves, but `layout` names
   each separator `<split>:<index>`: the agent preset's is `sidebar:0`, and
   `Resize("sidebar", …)` moves nothing. `split` takes a resizable
   separator's ID on screen, or a split's name when that split has exactly
   one resizable separator on screen, so A2's `/resize sidebar 4` means
   `sidebar:0`. A name with more than one is an argument error that lists
   them.

**Owner questions for A8.** *Answered 2026-10-06* (picked from options),
each the recommendation. The alternatives were: `State().Hidden`, the
panes the user hid; an argument error for focus and resize only; tree
order with hidden panes last; for item 4, separator IDs only.

### A9 (2026-10-06): the shell's confirmation, object flags, an unavailable command, and `--json` without a value

*Status: accepted (2026-10-06).* Found before writing Step 8 of
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md), recorded
there as deviations D31–D34. The first was found while building Step 3
and left open for this step.

**Found.**

1. §10 has `Destructive` commands from the shell ask `WithConfirm` or need
   `--yes`, and §7 has the policy ask the registry's `Gate`. `cli.Run`
   receives a registry whose gate `NewRegistry` fixed, so neither
   `--yes` nor `WithConfirm` can approve one request.
2. §10 makes each schema property a flag and repeats a flag for an array,
   but says nothing of an object or map property, such as
   `workspace.state.set`'s `state`.
3. §10's exit codes are 0 success, 1 failed, 2 usage or argument error and
   3 refused; a command whose `When` is false fits none plainly.
4. §10's `--json` prints `Result.Value`; a command may return text only.

**Decided.**

1. **`command.Request` gains `Gate Gate`.** When it is set, the policy asks
   it instead of the registry's gate, for that request only, and neither
   reads nor stores a remembered `AllowAlways` or `RejectAlways`. The
   policy still decides whether to ask, and the audit still records the
   origin and the decision. `cli.Run` sets a gate that allows when
   `--yes` was given or `WithConfirm` says yes, and refuses otherwise.
   `CallMCP` never sets it.
2. **An object or map property's flag takes JSON:**
   `--state '{"zoom":"logs"}'`. Scalars and arrays of scalars stay plain.
3. **A command whose `When` is false exits 1,** as a failure: it depends
   on state, not on how it was called. A command not offered on the CLI
   surface is unknown to the shell, and exits 2.
4. **`--json` always prints `Result.Value`**, `null` when there is none,
   so a script that reads stdout always gets JSON.

**Changed.** §10's confirmation note, annotated in place.

**Owner questions for A9.** *Answered 2026-10-06* (picked from options),
each the recommendation. The alternatives were:

1. a gate carried in the context, or the CLI building its own registry;
2. object properties only through `--args`;
3. exit 2, as a usage error;
4. the text as a JSON string.

## More Information

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md):
  §3 agent TUIs, §4 frameworks and applications, §5.1 ACP, §6 keymap and
  config standards, §7 the candidates.
* [0002-MADR-multi-pane-workspace-layouts.md](0002-MADR-multi-pane-workspace-layouts.md):
  the workspace whose methods become the first commands.
* [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md):
  `SetTheme`, used by `workspace.theme.set`.
* [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md):
  `termcap`, the source of `terminal.*` context keys.
* [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) and
  [0008-MADR-command-palette.md](0008-MADR-command-palette.md): the first
  surfaces built on this registry.
* MCP tools: <https://modelcontextprotocol.io/specification/2025-11-25/server/tools>.
  MCP schema: <https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/schema/2025-11-25/schema.ts>.
* ACP slash commands: <https://agentclientprotocol.com/protocol/slash-commands>.
  ACP tool calls and permissions: <https://agentclientprotocol.com/protocol/tool-calls>.
* VS Code when clauses: <https://code.visualstudio.com/api/references/when-clause-contexts>.
* JSON Schema 2020-12: <https://json-schema.org/draft/2020-12/schema>.
* [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
  and [0010-REPORT-nested-modules-and-adapter-sources.md](../reports/0010-REPORT-nested-modules-and-adapter-sources.md):
  the nested modules, and the Cobra, fang and Kong facts amendment A1
  relies on.
