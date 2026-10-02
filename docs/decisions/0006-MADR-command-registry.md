---
status: proposed
date: 2026-10-02
decision-makers: owner
consulted: 0003-REPORT-agent-tui-ecosystem-research.md (opencode, gemini-cli, codex, crush, toad, Textual, VS Code, k9s, lazygit, gh-dash); MCP specification 2025-11-25; Agent Client Protocol; Go 1.27.1 standard library
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
  * **MCP 2025-11-25, `tools/list`.** A tool is `name`, `title`,
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
* **Standard library only.** No new module (AGENTS.md Dependencies).
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
* it adds no module.

### 1. Packages and their dependencies

```text
 command       Command, Registry, invocation, args and JSON Schema, sources
               and loaders, gate, audit, MCP and ACP shapes
                                              → when, bubbletea, stdlib
 command/cli   run commands as shell subcommands with flags and --json
                                              → command, stdlib flag
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
    records).
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
  replaces the prefix rule.
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
    arguments. `$ARGUMENTS` is the whole slash tail.
  * Subdirectories become ID segments: `git/commit.md` is `user.git.commit`
    and `/user:git:commit`.
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
| `workspace.resize` | `resize` | `pane`, `delta` | UI |
| `workspace.layout.use` | `layout` | `name`: an enum of the layouts given in `WithLayouts` | UI |
| `workspace.layout.reset` | — | — | UI |
| `workspace.state.get` | — | — | ReadOnly |
| `workspace.state.set` | — | `state` (layout `State` JSON) | UI |
| `workspace.panes` | `panes` | — | ReadOnly |
| `workspace.overlay.close` | `close` | — | UI |
| `workspace.theme.set` | `theme` | `background`: dark, light or auto | UI |

The registry itself contributes `command.list` and `command.describe`
(ReadOnly), so an agent can discover what it may call, and `app.quit`
(UI), which only sends `QuitRequestMsg`.

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
  `WithConfirm`, and without it need `--yes`. There is no event loop, so
  `Result.Cmd` is ignored, and commands without `Surfaces&CLI` are not
  listed.
* **How a program offers both.** A program such as pi-go builds one
  registry. With no subcommand it starts its `tea.Program` and dispatches
  from `Update`. With one, it calls `cli.Run`. Help, flags and JSON output
  come from the same definitions.
* **Cobra and fang are not used now.** A `command/cobra` adapter is a later
  record if a consumer asks (owner question Q4).

### 11. Versioning and conventions

* This lands after `termcap`, in the next minor release after
  [0005-PLAN-terminal-capabilities-and-services.md](0005-PLAN-terminal-capabilities-and-services.md).
  The owner tags. `v0` allows API change.
* `command`, `when` and `command/cli` render nothing except CLI help, which
  is plain ASCII text. CLI help is golden-tested at two widths, and
  `internal/conformance` covers the new packages: they write only to the
  writers they are given.

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

### Confirmation

* `when`:
  * table tests for every operator and the precedence examples from the VS
    Code documentation;
  * `String()` round-trips: parsing the canonical form gives the same form;
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
