# Commands, agents and the shell

For a Bubble Tea v2 program that wants one definition of each action, run
from a key, a palette, a slash line, the shell or an agent. Four packages
do the work:

- `command` holds the commands: their arguments as JSON Schema, their
  danger and availability, and the registry that runs them, gates them
  and exports them to agents.
- `when` evaluates availability expressions, in VS Code's when-clause
  grammar.
- `command/cli` runs the same commands as shell subcommands.
- `workspace` publishes its own commands and context keys.

Why they are built this way is in
[0006-MADR](../decisions/0006-MADR-command-registry.md), with its
amendments A1 to A9. `ExampleRun` in `command/cli` compiles with the
package (`go doc -all github.com/maccavelli/go-tui-lib/command/cli`).

## Define a command

`command.New` builds a command whose arguments decode into a Go struct.
The struct is the schema:

```go
type resizeArgs struct {
    Split string `json:"split" arg:"" help:"the split whose separator moves" placeholder:"SPLIT"`
    Delta int    `json:"delta" arg:"" help:"cells to give the pane before it" schema:"min=-200,max=200"`
    Quiet bool   `json:"quiet,omitzero" short:"q" group:"Output" help:"say less"`
}

resize, err := command.New("layout.resize", "Move a split",
    func(ctx context.Context, inv *command.Invocation, a resizeArgs) (command.Result, error) {
        return command.Result{Text: "moved " + a.Split, Cmd: ws.Resize(a.Split, a.Delta)}, nil
    },
    command.WithDanger(command.UI),
    command.WithSlash("resize"),
    command.WithDescription("Moves a named split's separator."),
)
```

- **An ID** is dotted and lowercase, each segment `[a-z0-9][a-z0-9-]*`,
  at most 128 bytes, so it is also a valid MCP tool name.
- **The tags** follow Kong's (MADR A1): `json` names the property, which
  is required unless it is a pointer, has `omitzero` or `omitempty`, or
  has a `default`; `help`, `default`, `enum:"a,b"`, `arg:""` (positional,
  in field order), `short`, `hidden`, `placeholder` and `group`; and
  `schema:"min=…,max=…,minLen=…,maxLen=…,secret"`. A scalar `enum` must
  be required or have a default.
- **The struct reads the same in Kong,** since `v0.5.0` (A12). Kong
  requires a flag only with `required:""`, and a positional unless it has
  `optional:""` or a default, and it names a field by `name:""` or else
  its Go name spelled with dashes (`MaxItems` is `max-items`). `New`
  refuses a top-level field where these disagree with the `json` tag, and
  its error names the tag to add or remove: a required flag takes
  `required:""`, an `omitzero` positional `optional:""`, and
  `json:"max_items"` takes `name:"max_items"`. `SchemaOf` does not check,
  so an output type needs none of these.
- **The schema** is JSON Schema 2020-12. `secret` is written as
  `"writeOnly": true`; `arg`, `short`, `placeholder`, `group` and `hidden`
  go in one `"x-cli"` object, which JSON Schema treats as an annotation
  (A5). `command.SchemaOf[T]()` gives the schema alone.
- **Supported types:** string, bool, the integer and float kinds, slices
  and arrays, structs (an embedded one is flattened), pointers,
  `map[string]T`, `time.Duration` (a string such as `"1m30s"`), `[]byte`
  (base64), and a type with its own `JSONSchema() Schema`. Anything else
  is an error from `New`.
- **`WithDanger` is required.** `New` returns an error without it, and
  `Register` refuses a command whose danger is not declared (A4, A5).
- **The handler gets checked arguments.** The registry checks them
  against the schema, fills defaults, and refuses an unknown member, a
  wrong type, a missing field or a value out of range with a
  `*command.ArgError` whose `Path` is a JSON Pointer. Then `New` decodes
  them strictly into your struct.

`command.NoArgs` is the argument type of a command without arguments.

## Kinds, danger and the gate

| `Danger` | Means | Runs without asking for |
| :--- | :--- | :--- |
| `ReadOnly` | changes nothing | everyone |
| `UI` | changes only presentation | everyone |
| `Mutating` | changes program or session state | everyone but an agent |
| `Destructive` | cannot be undone, or writes outside the project | everyone but an agent or the shell |

- **When the policy asks,** it asks the registry's `Gate`
  (`command.WithGate`), typically your permission dialog. Without one, the
  request is refused. `AllowAlways` and `RejectAlways` are remembered per
  command and caller. A `Request` with its own `Gate` is asked instead,
  for that request only (A9); that is how the shell's `--yes` works.
- **Every request is audited** when you pass `command.WithAuditor`.
  `command.SlogAuditor(logger)` writes a record per request to your
  logger, with `secret` values masked.
- **Three kinds.** An `Action` runs its handler. A `Prompt` expands into
  text for your agent and sends a `PromptMsg`. A `Forward` sends
  `/name args` to the agent, as ACP's available commands are run.
- **A failure is one of three errors,** tested with `errors.Is`:
  `command.ErrUnknown`, `command.ErrUnavailable` (its `When` is false, or
  it is not offered where it was asked from) and `command.ErrRefused`.
  An argument error is a `*command.ArgError`.

## Run commands from your program

```go
r := command.NewRegistry(command.WithGate(dialog), command.WithLoop(program.Send))
if err := r.Register(resize); err != nil { … }
if err := r.Register(workspace.Commands(ws)...); err != nil { … }

func (m *model) Init() tea.Cmd { return tea.Batch(m.ws.Init(), m.r.Watch()) }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case command.LoopMsg: // an agent's command, run here on the loop
        return m, msg.Run()
    case command.ChangedMsg: // the commands changed; listen again
        return m, m.r.Watch()
    case command.ConflictMsg: // a loaded command was renamed or refused
        m.toast(msg.Conflicts)
    case command.QuitRequestMsg: // app.quit; quitting stays yours
        return m, tea.Quit
    case command.ResultMsg:
        if msg.Err != nil { m.toast(msg.Err) }
    case tea.KeyPressMsg:
        if msg.String() == "f2" {
            return m, m.r.Dispatch(context.Background(), command.Request{
                ID: "workspace.zoom", Origin: command.OriginKey, Context: m.ws.WhenContext(),
            })
        }
    }
    return m, m.ws.Update(msg)
}
```

- **`Dispatch`** is for `Update`. A `Loop` command, the default mode,
  runs at once, and `Dispatch` returns its `Result.Cmd` with a
  `ResultMsg`. An `Async` command runs inside the returned `tea.Cmd`,
  under a context `Cancel` and `CancelAll` reach; `WithExclusive` makes a
  new run cancel the last.
- **`Run`** runs to completion and returns the `Result`, for the shell,
  agents and tests. Under `WithLoop` it hands a `Loop` command to your
  `Update` as a `LoopMsg` and waits, so an agent's tool call, which
  arrives on its own goroutine, never touches your model off the loop
  (A7). With `WithLoop` set, `Update` must use `Dispatch`, not `Run`.
- **Reads are lock-free.** `Lookup`, `Slash`, `All` and `Available` read
  one snapshot, and are safe from any goroutine.
- **The registry's own commands,** registered by `NewRegistry`:
  `command.list` and `command.describe`, so an agent can find what it may
  call, and `app.quit`, which only sends `QuitRequestMsg`. None has a
  slash name, so none clashes with yours (A5).

## Availability

A command's `When` is an expression over context keys:

```text
workspace.focusedPane == transcript && !workspace.modal
workspace.width >= 100 || workspace.zoomed
logs in workspace.hiddenPanes
mode =~ /^insert/i
```

- **The grammar** is VS Code's: `!`, `&&`, `||`, parentheses, `==`, `!=`,
  `<`, `<=`, `>`, `>=`, `=~` with an RE2 regex, and `in` and `not in` a
  list key. Spaces around operators are optional. A key alone is true
  when it is set and not false, 0, `""` or empty; an unset key equals
  nothing.
- **`when.Check(expr, keys)`** reports an unknown key or a type mismatch,
  so a typo in a user's file is an error at load, not a command that never
  appears. `workspace.ContextKeys()` gives the workspace's keys.
- **A source is at most 4 KiB, and so is its canonical form,** and it
  nests at most 64 deep (A3).
- **`Available(context, surface)`** lists the commands a palette, a slash
  popup or help shows: not hidden, offered on that surface, with `When`
  true.

## Slash commands

`r.ParseSlash("/resize sidebar 4")` gives the `Request` the palette,
`--args` and an agent would send for it: `OriginSlash`, the tail in `Raw`,
and the arguments as JSON.

- Positional values fill the `arg` properties in order; a positional array
  takes the rest. `name=value` sets any property, and repeats for an
  array. Quotes group words.
- A command whose only argument is a string takes the whole tail.
- An accepted line has been checked against the schema, so it runs unless
  the handler refuses it.

## Command files

`command.LoadDir(fsys, source)` reads Markdown command files, as opencode
writes them:

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

- **Pass `os.Root.FS()`** for the user's and the project's directories,
  so a symlink cannot reach outside them. Where they are is your choice.
- **The front matter is a strict subset of YAML:** one `key: value` per
  line, from `title`, `description`, `slash`, `aliases` (comma separated),
  `category`, `danger`, `when`, `hidden` and `arg.<NAME>`. An unknown or
  repeated key, a nested value, or a bad danger, slash name or `when` is
  an error naming the file and line.
- **`$NAME`** is a required, positional string argument named as written,
  in order of first use. **`$ARGUMENTS`** is the whole slash tail, so a
  file that uses it accepts any words after its positionals (A6).
  Expansion substitutes text and runs nothing.
- **IDs and slash names come from the path:** `git/commit.md` from the
  user's directory is `user.git.commit`, typed `/git:commit`. A file
  without `danger` is `Mutating`.

## MCP prompts and ACP commands

```go
cmds := command.FromMCPPrompts("github", prompts, func(ctx context.Context, name string, args map[string]string) (string, error) {
    return myClient.GetPromptText(ctx, name, args) // your MCP client
})
conflicts := r.ReplaceSource(command.Source{Kind: command.MCP, Name: "github"}, cmds)

forwards := command.FromACP("claude", update.AvailableCommands)
r.ReplaceSource(command.Source{Kind: command.ACP, Name: "claude"}, forwards)
```

- **`ReplaceSource` swaps a source's whole set in one version,** because
  ACP sends its full list in each update and MCP says only that its list
  changed.
- **Names that are not ID segments are mapped:** `github_issue` is
  `github-issue`, `Plan Mode` is `plan-mode` (A6). An ACP forward still
  sends the agent its own name.
- **Built-ins win slash names.** A loaded command whose name is taken
  becomes `/<source>:<name>`, `/mcp:github:issue`, or loses the name when
  that is taken too. A built-in registered later takes the name back.
  Each rename, and each command that could not load, is a `Conflict`,
  returned and sent through `Watch` as a `ConflictMsg`.

## Let an agent drive the program

```go
tools := r.MCPTools(ws.WhenContext())                          // tools/list's tools
res := r.CallMCP(ctx, call.Name, call.Arguments, client.Name) // a tools/call result
```

- **`MCPTools`** gives MCP 2026-07-28 `Tool` objects for the commands an
  agent may run now. The ID is the tool's name, and every annotation that
  applies is written, because MCP's defaults for `destructiveHint` and
  `openWorldHint` are true.
- **`CallMCP`** runs a tool as `OriginAgent`, so the policy and your gate
  apply, and returns a `CallToolResult` with `resultType: "complete"`. An
  unknown tool, bad arguments, a refusal or a failure is `isError: true`
  with a sentence the model can act on, never a protocol error.
- **What your MCP server still owns** under 2026-07-28:
  - the `tools/list` envelope: `tools`, `resultType: "complete"`, and the
    required `ttlMs` and `cacheScope` (`"private"` suits a program's own
    tools), with `nextCursor` if you page;
  - `notifications/tools/list_changed` on a `subscriptions/listen` stream
    whose client set `toolsListChanged`: send it on each `ChangedMsg` from
    `Watch`;
  - the transport, sessions and authorization.
- **`ACPCommands`** gives the commands as an ACP agent's available
  commands: those with a slash name, under it.
- **`Manifest`** is the whole catalogue as one JSON document, for docs and
  completion.

## The workspace's commands

`workspace.Commands(ws, workspace.WithLayouts(layouts))` returns the
workspace's own commands, closed over it, each run on the event loop and
offered everywhere but the shell:

| ID | Slash | Does |
| :--- | :--- | :--- |
| `workspace.focus`, `.focus.next`, `.focus.prev` | `focus`, `next`, `prev` | moves focus |
| `workspace.zoom` | `zoom` | zooms a pane, the focused one by default, or restores |
| `workspace.toggle` | `toggle` | hides or shows a pane |
| `workspace.resize` | `resize` | moves a split: `sidebar:0`, or `sidebar` when it has one separator (A8) |
| `workspace.layout.use` | `layout` | switches to a layout named in `WithLayouts` |
| `workspace.layout.reset` | — | no zoom, nothing hidden, no moved splits |
| `workspace.state.get`, `.state.set` | — | reads or replaces the layout state |
| `workspace.panes` | `panes` | every pane, shown and hidden, with title, rectangle and focus |
| `workspace.overlay.close` | `close` | closes the top overlay |
| `workspace.theme.set` | `theme` | `dark`, `light`, or `auto` to follow the terminal |

- **`ws.WhenContext()`** layers the top overlay's keys, if its content is
  a `workspace.Contexter`, over the focused pane's, over the workspace's:
  `workspace.focusedPane`, `.zoomed`, `.hiddenPanes`, `.overlay`,
  `.modal`, `.width` and `.height`.
- **`ws.SetBackground(theme.Light)`** pins the background the theme is
  built for, and a later report from the terminal does not change it;
  `theme.Unknown` follows the terminal again and asks it. A workspace
  built `WithTheme` keeps its theme.
- **A pane or split the workspace does not have** is a
  `*command.ArgError`. The workspace's key bindings keep working beside
  the commands.

## Run commands from the shell

```go
func main() {
    r := buildRegistry()
    if len(os.Args) > 1 {
        os.Exit(cli.Run(context.Background(), r, os.Args[1:], os.Stdout, os.Stderr,
            cli.WithName("pi"), cli.WithConfirm(askOnTTY)))
    }
    runTUI(r)
}
```

- **Words or IDs:** `pi session save` or `pi session.save`. The
  workspace's own commands are not offered here: they need the running
  TUI. Each argument is a flag, `--delta 4`; an array repeats it, and an object
  takes JSON (A9). Positional arguments come in order. `--args` takes the
  whole object as JSON, and `--json` prints the result's value, `null`
  when there is none.
- **Verbs:** `list` (`--json` for the manifest), `describe`, `schema` and
  `help`. Help is plain ASCII at `WithWidth` (80 by default).
- **A destructive command** asks `WithConfirm`, or without it needs
  `--yes`. Only commands offered on `SurfaceCLI` are listed or run, and
  `Result.Cmd` is ignored: there is no event loop.
- **Exit codes:** `cli.ExitOK` 0; `cli.ExitFailed` 1, a failure or a
  command whose `When` is false; `cli.ExitUsage` 2, an unknown command or
  flag or a bad argument; `cli.ExitRefused` 3.
- **`cli.Run` writes only to the writers you pass**, each once when the
  command ends, and never exits the process.

## What the library never does

- **Run a shell command, or read a file you did not give it.** Command
  files are read through the `fs.FS` you pass, and expansion only
  substitutes text.
- **Own the screen, the signals or the quit.** `app.quit` sends
  `QuitRequestMsg`, and your program decides.
- **Speak a protocol.** `MCPTool`, `MCPCallResult` and `ACPCommand` are
  plain structs with the protocols' field names; no MCP or ACP SDK is
  imported, and your server or client owns the transport.
- **Write to the terminal or the default logger.** Output goes to the
  writers, the logger and the program you give it.
