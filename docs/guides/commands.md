# Commands, agents and your CLI

For a Bubble Tea v2 program that wants one definition of each action, run
from a key, a palette, a slash line, an agent or the program's own command
line. Three packages do the work:

- `command` holds the commands: their arguments as JSON Schema, their
  danger and availability, and the registry that runs them, gates them
  and exports them to agents.
- `when` evaluates availability expressions, in VS Code's when-clause
  grammar.
- `workspace` publishes its own commands and context keys.

Why they are built this way is in
[0006-MADR](../decisions/0006-MADR-command-registry.md), with its
amendments A1 to A13. The library ships no command-line front end: your
program brings its own
([0012-MADR](../decisions/0012-MADR-bring-your-own-cli.md)).

## Define a command

`command.New` builds a command whose arguments decode into a Go struct.
The struct is the schema:

```go
type searchArgs struct {
    Query string `json:"query" arg:"" help:"the text to find" placeholder:"TEXT"`
    Limit int    `json:"limit,omitzero" help:"the most matches to list" schema:"min=1,max=200"`
    Quiet bool   `json:"quiet,omitzero" short:"q" group:"Output" help:"say less"`
}

search, err := command.New("session.search", "Search the session",
    func(ctx context.Context, inv *command.Invocation, a searchArgs) (command.Result, error) {
        hits := transcript.Find(a.Query, a.Limit) // your program's own search
        return command.Result{Text: fmt.Sprintf("%d matches", len(hits)), Value: hits}, nil
    },
    command.WithDanger(command.ReadOnly),
    command.WithSlash("search"),
    command.WithDescription("Finds text in the session's transcript."),
)
```

The slash name `search` is free: `workspace.Commands` takes `close`,
`focus`, `layout`, `next`, `panes`, `prev`, `resize`, `theme`, `toggle`
and `zoom`, and two commands with one slash name cannot both register.

- **An ID** is dotted and lowercase, each segment `[a-z0-9][a-z0-9-]*`,
  at most 128 bytes, so it is also a valid MCP tool name.
- **The tags** follow Kong's (MADR A1): `json` names the property, which
  is required unless it is a pointer, has `omitzero` or `omitempty`, or
  has a `default`; `help`, `default`, `enum:"a,b"`, `arg:""` (positional,
  in field order), `short`, `hidden`, `placeholder` and `group`; and
  `schema:"min=…,max=…,minLen=…,maxLen=…,secret"`. A scalar `enum` must
  be required or have a default.
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
  for that request only (A9); that is how your CLI's `--yes` approves
  one command (see "Run commands from your own CLI").
- **Every request is audited** when you pass `command.WithAuditor`.
  `command.SlogAuditor(logger)` writes a record per request to your
  logger, with `secret` values masked; `command.AuditorFunc` makes an
  auditor of a function.
- **A request's arguments are capped** at `command.DefaultMaxArgBytes`
  (1 MiB): its `Args` and `Raw`, a slash line's tail, and the words
  `ParseArgs` reads, each checked before anything parses it. Larger is an
  `*ArgError`, and the audit records no arguments.
  `command.WithMaxArgBytes(n)` sets another limit; below 1 keeps the
  default.
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
if err := r.Register(search); err != nil { … }
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
                ID: "workspace.zoom", Origin: command.OriginKey, WhenContext: m.ws.WhenContext(),
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
- **`Run`** runs to completion and returns the `Result`, for your CLI,
  agents and tests. Under `WithLoop` it hands a `Loop` command to your
  `Update` as a `LoopMsg` and waits, so an agent's tool call, which
  arrives on its own goroutine, never touches your model off the loop
  (A7). With `WithLoop` set, `Update` must use `Dispatch`, not `Run`.
- **A handler's panic is an error,** on every path, `Run`, `Dispatch`,
  an `Async` command and a `Loop` command on your program's loop
  included: a `*command.PanicError`, which wraps `command.ErrPanicked`
  and holds the value and the stack. Its text names the command only,
  since the value may hold a secret. Your TUI keeps running. A gate's
  panic refuses the request.
- **`WatchContext(ctx)`** is `Watch` that gives up when `ctx` ends, so a
  program that stops listening leaves no goroutine waiting.
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

`r.ParseSlash("/resize sidebar 4")` gives the `Request` the palette, the
program's own command line (`ParseArgs`) and an agent would send for it:
`OriginSlash`, the tail in `Raw`, and the arguments as JSON.

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
- **Limits.** `LoadDir` reads at most `command.DefaultMaxFileBytes`
  (256 KiB) per file, `DefaultMaxFiles` (1,000) files and
  `DefaultMaxDepth` (8) directory levels below the root. Past each, it
  returns an error wrapping `ErrFileTooLarge`, `ErrTooManyFiles` or
  `ErrTooDeep`, and the files within the limits still load.
  `command.LoadDirWith(fsys, source, command.WithMaxFileBytes(n), …)`
  sets others, with `WithMaxFiles` and `WithMaxDepth`; below 1 keeps the
  default. An expanded prompt over 1 MiB is an `*ArgError`.

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
offered everywhere but your CLI (`SurfaceCLI`), since they need the running
TUI:

| ID | Slash | Does |
| :--- | :--- | :--- |
| `workspace.focus`, `.focus.next`, `.focus.prev` | `focus`, `next`, `prev` | moves focus |
| `workspace.zoom` | `zoom` | zooms a pane, the focused one by default, or restores |
| `workspace.toggle` | `toggle` | hides or shows a pane |
| `workspace.resize` | `resize` | moves a split: `sidebar:0`, or `sidebar` when it has one separator (A8) |
| `workspace.layout.use` | `layout` | switches to a layout named in `WithLayouts`; registered only when `WithLayouts` is given |
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

## Run commands from your own CLI

go-tui-lib is a TUI layer: your program keeps its own command line, in the
standard `flag` package, Cobra, Kong, urfave/cli or anything else, and adds
the TUI beside it. A CLI handler runs a registry command by calling `Run`
with `command.OriginCLI`, so the gate and the audit trail apply as they do
to a key or an agent. `command.ArgsOf` makes the arguments from your
flags, `command.AllowIf` turns `--yes` into the request's gate, and
`command.WriteResult` prints the result:

<!-- from: testdata/frameworks/flag/main.go#runcli -->

```go
// runCLI runs a registry command from the program's own command line, and
// writes its result to w. AllowIf(yes) approves a command that asks, when
// --yes was given.
func runCLI(ctx context.Context, w io.Writer, r *command.Registry, id command.ID, args any, yes bool) error {
    raw, err := command.ArgsOf(args)
    if err != nil {
        return err
    }
    res, err := r.Run(ctx, command.Request{
        ID: id, Args: raw, Origin: command.OriginCLI, Caller: "pi", Gate: command.AllowIf(yes),
    })
    if err != nil {
        return err
    }
    return command.WriteResult(w, res, command.FormatText)
}

// runAny runs any command offered to the CLI, by its ID and its arguments
// as the shell split them: "run session.save notes", or "run
// session.save --name=notes".
func runAny(ctx context.Context, w io.Writer, r *command.Registry, args []string, yes bool, f command.Format) error {
    if len(args) == 0 {
        return &command.ArgError{Reason: "run: name a command"}
    }
    req, err := r.ParseArgs(command.ID(args[0]), args[1:], command.OriginCLI)
    if err != nil {
        return err
    }
    req.Caller, req.Gate = "pi", command.AllowIf(yes)
    res, err := r.Run(ctx, req)
    if err != nil {
        return err
    }
    return command.WriteResult(w, res, f)
}

type saveArgs struct {
    Name string `json:"name" arg:"" help:"the session's name"`
}
```

With the standard `flag` package:

<!-- from: testdata/frameworks/flag/main.go#flag -->

```go
switch args[0] {
case "save":
    fs := flag.NewFlagSet("save", flag.ExitOnError)
    yes := fs.Bool("yes", false, "approve without asking")
    _ = fs.Parse(args[1:])
    err = runCLI(ctx, os.Stdout, r, "session.save", saveArgs{Name: fs.Arg(0)}, *yes)
case "run": // pi run [--yes] [--format=json] ID ARGS…
    fs := flag.NewFlagSet("run", flag.ExitOnError)
    yes := fs.Bool("yes", false, "approve without asking")
    var format command.Format
    fs.TextVar(&format, "format", command.FormatText, "text or json")
    _ = fs.Parse(args[1:]) // stops at the ID; the rest is the command's
    err = runAny(ctx, os.Stdout, r, fs.Args(), *yes, format)
}
if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(launch.ExitCode(err))
}
```

With Cobra:

<!-- from: testdata/frameworks/cobra/main.go#cobra -->

```go
var yes bool
save := &cobra.Command{
    Use:   "save NAME",
    Short: "Save the session",
    Args:  cobra.ExactArgs(1),
    RunE: func(c *cobra.Command, a []string) error {
        return runCLI(c.Context(), c.OutOrStdout(), r, "session.save", saveArgs{Name: a[0]}, yes)
    },
}
save.Flags().BoolVar(&yes, "yes", false, "approve without asking")
```

With Kong, the registry's argument struct can be the subcommand's grammar, and
`launch.Flags` embeds beside it (see "Start the TUI from your CLI"):

<!-- from: testdata/frameworks/kong/main.go#kong -->

```go
var cli struct {
    launch.Flags `embed:""` // --mode and --tui

    Yes  bool     `help:"approve without asking"`
    Save saveArgs `cmd:"" help:"Save the session"`
    Run  struct {
        Format command.Format `default:"text" help:"text or json"`
        ID     string         `arg:"" passthrough:"" help:"the command's ID"`
        Args   []string       `arg:"" optional:"" help:"its arguments"`
    } `cmd:"" help:"Run any command by its ID"`
    Line struct{} `cmd:"" default:"1" help:"Run the line mode"`
}
kctx := kong.Parse(&cli)
switch kctx.Command() {
case "save <name>":
    kctx.FatalIfErrorf(runCLI(ctx, kctx.Stdout, r, "session.save", cli.Save, cli.Yes))
case "run <id>", "run <id> <args>":
    args := append([]string{cli.Run.ID}, cli.Run.Args...)
    kctx.FatalIfErrorf(runAny(ctx, kctx.Stdout, r, args, cli.Yes, cli.Run.Format))
}
```

With urfave/cli v3, whose root command also carries the `--tui` path (see
"Start the TUI from your CLI"):

<!-- from: testdata/frameworks/urfave/main.go#urfave -->

```go
app := &cli.Command{
    Name: "pi",
    Flags: []cli.Flag{
        &cli.GenericFlag{Name: "mode", Value: &f.Mode, Usage: "auto, tui or plain"},
        &cli.BoolFlag{Name: "tui", Destination: &f.TUI, Usage: "start the TUI"},
    },
    Action: func(ctx context.Context, c *cli.Command) error {
        s := launch.Streams{In: c.Reader, Out: c.Writer, Err: c.ErrWriter, Env: os.Environ()}
        m := session{}
        if f.Resolve() == launch.ChoiceTUI {
            final, code, fallBack := tui(ctx, s, r, m)
            if !fallBack {
                return exitWith(code)
            }
            m = final
        }
        lineMode(s, m)
        return nil
    },
    Commands: []*cli.Command{{
        Name:      "save",
        Usage:     "Save the session",
        ArgsUsage: "NAME",
        Flags:     []cli.Flag{&cli.BoolFlag{Name: "yes", Usage: "approve without asking"}},
        Action: func(ctx context.Context, c *cli.Command) error {
            return runCLI(ctx, c.Root().Writer, r, "session.save", saveArgs{Name: c.Args().First()}, c.Bool("yes"))
        },
    }, {
        Name:      "run",
        Usage:     "Run any command by its ID",
        ArgsUsage: "ID [ARGS...]",
        Flags: []cli.Flag{
            &cli.BoolFlag{Name: "yes", Usage: "approve without asking"},
            &cli.TextFlag{Name: "format", Value: &format, Usage: "text or json"},
        },
        StopOnNthArg: &one, // the flags after the ID are the command's
        Action: func(ctx context.Context, c *cli.Command) error {
            return runAny(ctx, c.Root().Writer, r, c.Args().Slice(), c.Bool("yes"), format)
        },
    }},
}
if err := app.Run(context.Background(), os.Args); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(launch.ExitCode(err))
}
```

- **The policy treats `OriginCLI` as the shell:** a `Destructive` command
  asks the gate. `AllowIf(yes)` refuses it unless `--yes` was given, so
  the run fails with `command: refused: session.save: the gate said
  reject_once`; you could ask on the terminal instead, with a
  `command.GateFunc`.
- **Only commands offered on `SurfaceCLI` run.** The workspace's own
  commands are not, since they need the running TUI.
- **`WriteResult(w, res, f)`** writes `Result.Text`, or else
  `Result.Value` as JSON, for `command.FormatText`, and `Result.Value` as
  JSON, or `null`, for `command.FormatJSON`, each with a newline. A
  `time.Duration` is written as `"1m30s"`. `Format` has text forms
  (`text`, `json`), so a `--format` flag binds to it with `flag.TextVar`,
  pflag's `TextVar` or urfave's `TextFlag`. `Result.Cmd` is a TUI's
  effect, and is not run. `ArgsOf` writes a duration the same way, which
  `New`'s schema reads back.
- **Each error carries its exit status,** through `ExitCode() int`, and
  `launch.ExitCode` reads it through any wrapping: 2 for
  `command.ErrUnknown` and a `*command.ArgError`, 1 for
  `ErrUnavailable`, 3 for `ErrRefused`, and 2 for `ErrPanicked`. Kong's
  `FatalIfErrorf` honours it too.
- **A struct shared with Kong:** Kong reads `arg`, `help`, `default`,
  `enum`, `short`, `hidden`, `placeholder` and `group` as the registry
  does. It takes a field's name from its Go name (`MaxItems` is
  `max-items`), and requiredness from its own `required:""` and
  `optional:""`, which the registry does not read. Give a shared struct
  the tags Kong needs.
- **These examples are compiled and run** before every release. Each is
  cut from a complete program under `testdata/frameworks`, which
  `make examples` builds against the library in a module of its own and
  runs: without `--yes` the save is refused, with exit status 3; with it
  the save runs; a bad flag exits with the framework's own status; `run`
  takes any command by its ID; Cobra completes it; and `--tui` with no
  terminal falls back to the line mode. The library itself imports none
  of the frameworks.

### Any command, by its ID

`runAny`, above, runs whichever command the user names: `pi run
session.save notes`, or `pi run --yes --format=json session.save
--name=notes`. `Registry.ParseArgs(id, args, origin)` reads the words the
shell split into the request a slash line or an agent would send:

- `--name=v`, `--name v`, `-s v` and `-s=v`, where `name` is the
  argument's JSON name and `s` its `short` tag; a bare `--flag` sets a
  boolean, and a repeated flag adds to a list;
- positional values fill the arguments tagged `arg`, in order; `--` ends
  the flags, so `-- -3` is a value;
- `Raw` is the words, quoted where the shell would need it, so a prompt's
  `$ARGUMENTS` sees what the user typed;
- an unknown flag, a missing value or a missing required argument is a
  `*command.ArgError`, which exits 2.

The program's own flags, `--yes` and `--format` here, come before the ID:
everything after it belongs to the command. Each framework stops at the
ID its own way: the standard `flag` package always does, pflag with
`SetInterspersed(false)`, urfave with `StopOnNthArg`, and Kong with
`passthrough` on the ID.

`command.Params(cmd)` lists a command's arguments, with their types,
help, short names, groups and whether they are positional, hidden or
secret, for a help screen of your own. `Registry.Complete(id, args,
partial)` completes a command line: the IDs offered on the CLI, then the
command's flags and their values. With Cobra:

<!-- from: testdata/frameworks/cobra/main.go#cobra-run -->

```go
var format command.Format
run := &cobra.Command{
    Use:   "run ID [ARGS...]",
    Short: "Run any command by its ID",
    Args:  cobra.MinimumNArgs(1),
    RunE: func(c *cobra.Command, a []string) error {
        return runAny(c.Context(), c.OutOrStdout(), r, a, yes, format)
    },
    // The IDs, then each command's flags and their values.
    ValidArgsFunction: func(_ *cobra.Command, a []string, partial string) ([]string, cobra.ShellCompDirective) {
        if len(a) == 0 {
            return r.Complete("", nil, partial), cobra.ShellCompDirectiveNoFileComp
        }
        return r.Complete(command.ID(a[0]), a[1:], partial), cobra.ShellCompDirectiveNoFileComp
    },
}
run.Flags().BoolVar(&yes, "yes", false, "approve without asking")
run.Flags().TextVar(&format, "format", command.FormatText, "text or json")
run.Flags().SetInterspersed(false) // the flags after the ID are the command's
```

`pi __complete run se` then answers `session.save`, and `pi __complete
run session.save --` answers `--name`. Completion reads the commands'
definitions only: it does not check a command's `When` or its gate.

## Start the TUI from your CLI

`launch` starts the TUI from your program's own command line, on its own
streams, and only where it can run
([0013-MADR](../decisions/0013-MADR-cli-integration-helpers.md)). A
program whose default is its own CLI mode asks for the TUI with `--tui`.
When the TUI cannot start, or crashes, the program says why on its
standard error and goes on in its CLI mode, from where the TUI was:

<!-- from: testdata/frameworks/flag/main.go#tui -->

```go
// tui runs the TUI for --tui. When it cannot start, or crashes, it says why
// on Err and returns the model to continue from, with fallBack true: the
// program goes on in its own CLI mode, and never repeats what the TUI did.
// An end the user or the program chose ends the program, with its status.
func tui(ctx context.Context, s launch.Streams, r *command.Registry, m session) (session, int, bool) {
    d := launch.Decide(s, launch.Config{Choice: launch.ChoiceTUI})
    final, err := launch.Run(ctx, s, d, m, launch.WithRegistry(r))
    switch {
    case errors.Is(err, launch.ErrNotStarted):
        fmt.Fprintf(s.Err, "Warning: --tui unavailable (%v); using the CLI\n", err)
        return m, 0, true
    case errors.Is(err, launch.ErrCrashed):
        fmt.Fprintf(s.Err, "Warning: the TUI stopped (%v); continuing in the CLI\n", err)
        return final, 0, true
    default:
        return final, launch.ExitCode(err), false
    }
}
```

- **`Decide`** reads the terminal state of the streams and the environment
  you pass, never the process's own, and returns a `Decision` with a
  `Reason` token. Its rules, in order:
  1. `ChoicePlain`: plain, `launch.requested-plain`.
  2. Unless `ChoiceTUI`: the variable `Config.NoInputEnv` names, set and
     not empty (`launch.no-input-variable`), then `CI` set, not empty, and
     neither `false` nor `0` (`launch.ci`). A flag outranks the
     environment, so `--tui` skips both.
  3. `TERM=dumb`, whatever the choice: `launch.dumb-terminal`.
  4. The input: `In` if it is a terminal (`launch.input-not-terminal`
     otherwise).
  5. The drawing stream: `Out` if it is a terminal
     (`launch.output-not-terminal` otherwise).
  6. Otherwise interactive, `launch.terminal`.
- **`Decision`** also carries the colour profiles for `Out` and for the
  TUI's stream, the glyph tier the locale allows, and the size. The
  profiles come from the environment, with no terminfo file read and no
  process started: `NO_COLOR` lowers them, and `FORCE_COLOR`, unless `0`
  or `false`, raises them. On Windows with no `TERM`, the system's build
  number counts too
  ([0013-MADR](../decisions/0013-MADR-cli-integration-helpers.md) A1.7).
- **`Run`** starts Bubble Tea on the decision's streams, with no signal
  handler: cancel `ctx` from your own `signal.NotifyContext`. It returns
  the final model as your type.
  - A failure before your model's `Init` wraps `ErrNotStarted`: nothing
    was drawn by your model, and the terminal is as it was.
  - A crash after it, a panic Bubble Tea recovered or a failure reading
    input, wraps `ErrCrashed`. The terminal is restored, and the model is
    the last good one, after the last `Update` that returned. Bubble Tea
    writes its own crash report to the process's standard error.
  - An interrupt (`tea.ErrInterrupted`) or a cancelled `ctx` returns
    Bubble Tea's error as it is: the user or the program ended it.
- **`Run` never sets the alternate screen;** your `View` does. A final
  model with an `Epilogue() string` method has it printed to `Out` after
  a clean end, for a resume hint.

### The flags in each framework

`launch.Flags` is `--mode` (`auto`, `tui` or `plain`, a `launch.Choice`)
and `--tui`, a plain `bool`. `Resolve` is `ChoiceTUI` when `--tui` is set,
and the mode otherwise. With the standard `flag` package:

<!-- from: testdata/frameworks/flag/main.go#flag-tui -->

```go
var f launch.Flags
global := flag.NewFlagSet("pi", flag.ExitOnError)
f.RegisterFlags(global)
_ = global.Parse(os.Args[1:])
s := launch.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Env: os.Environ()}
m := session{}
if f.Resolve() == launch.ChoiceTUI {
    final, code, fallBack := tui(ctx, s, r, m)
    if !fallBack {
        os.Exit(code)
    }
    m = final
}
```

With Cobra, `RegisterFlags` on a `flag.FlagSet` and `AddGoFlagSet` keep
`--tui` a bare boolean, and `FromSource` takes the command's streams as
they are:

<!-- from: testdata/frameworks/cobra/main.go#cobra-tui -->

```go
var f launch.Flags
root := &cobra.Command{
    Use:   "pi",
    Short: "A program with a TUI beside its CLI",
    RunE: func(c *cobra.Command, _ []string) error {
        s := launch.FromSource(c, os.Environ())
        m := session{}
        if f.Resolve() == launch.ChoiceTUI {
            final, code, fallBack := tui(c.Context(), s, r, m)
            if !fallBack {
                return exitWith(code)
            }
            m = final
        }
        lineMode(s, m)
        return nil
    },
}
fs := flag.NewFlagSet("pi", flag.ContinueOnError)
f.RegisterFlags(fs)
root.PersistentFlags().AddGoFlagSet(fs) // keeps --tui a bare boolean
```

With Kong, `launch.Flags` embeds into the grammar (above), and its
streams are the writers Kong holds. No flag means no TUI, so there is no
`--no-tui`; `--tui=false` cancels an earlier `--tui`:

<!-- from: testdata/frameworks/kong/main.go#kong-tui -->

```go
if kctx.Command() == "line" {
    s := launch.Streams{In: os.Stdin, Out: kctx.Stdout, Err: kctx.Stderr, Env: os.Environ()}
    m := session{}
    if cli.Resolve() == launch.ChoiceTUI {
        final, code, fallBack := tui(ctx, s, r, m)
        if !fallBack {
            os.Exit(code)
        }
        m = final
    }
    lineMode(s, m)
}
```

With urfave/cli v3, `--mode` is a `GenericFlag` on the `Choice`, which has
the `Get` urfave requires, and the streams are the command's own (the
root command above).

The tier-2 frameworks were run once, not on every release:

| Framework | How `launch.Flags` binds |
| :--- | :--- |
| ff v4 | `ff.NewFlagSetFrom(name, &flags)`, from the `ff` tags |
| go-flags | a `group:""` field, read through `UnmarshalFlag` |
| go-arg | embedded, read through `UnmarshalText` |

Each needs a one-line `launch.Streams{…}` literal; only a Cobra command
fits `FromSource`.

### Where the TUI reads and draws

- **`Config.UIOnErr`** lets the TUI draw on `Err` when `Out` is not a
  terminal, so `x=$(prog pick --tui)` captures only the result on `Out`.
- **`Config.OpenTTY`** lets the TUI read from and draw on the controlling
  terminal when `In` or `Out` is not one, as fzf does. It is not
  supported on Windows, where `Decide` ignores it: a read of the console
  handle launch would open cannot be cancelled, and would take the CLI's
  next line
  ([0013-MADR](../decisions/0013-MADR-cli-integration-helpers.md) A1.9).

### Commands, a prober and an agent

- **`launch.WithRegistry(r)`** attaches the registry to the program while
  the TUI runs, so an agent's `Loop` command runs on the program's loop.
  It is detached on every outcome, and a `Loop` command run after that
  runs on its caller, at once.
- **`launch.WithRestorer(p)`** with your `*termcap.Prober` writes its
  resets, such as mode 2031's, to the TUI's stream on every outcome, a
  crash included, so the CLI mode's input gets no colour-scheme reports.
- **`launch.OnStart(f)`** calls `f` with the `*tea.Program` before it
  runs, so a goroutine of yours can `Send` to it.
- **`launch.WithFilter`** is `tea.WithFilter`, given your own model;
  `WithProgramOptions` adds any other Bubble Tea option.

### Plain output and the exit status

- **`launch.Frame(m, width, height, profile)`** draws your model once, for
  a plain status board or a final screen. It sends the profile and the
  size through `Update`, runs no command, and returns the view.
- **`launch.ExitCode(err)`** is the shell's status: an error's own
  `ExitCode() int` first, such as a `launch.ExitError`; then 0 for nil, 2
  for `ErrNotStarted`, 130 for an interrupt or a cancel, 124 for a
  deadline, 2 for a panic, and 1 otherwise. `launch.ExitError` carries a
  status through Kong's `FatalIfErrorf` and urfave's exit handler, and
  `command`'s errors carry theirs (see "Run commands from your own
  CLI").

### Testing it

`launch/launchtest` gives your own tests fake streams: a `Terminal` that
says it is one, has a size and takes typed keys, a `Pipe` that is not one,
and `Streams` and `Env` builders. `ExampleTerminal` tests a `--tui` path
end to end: the decision on a fake terminal, then a run that quits on a
typed `q`.

### Cautions

- **pflag ignores `IsBoolFlag` on a custom value,** so a bare `--tui`
  bound through pflag's own `Var` asks for an argument. Use
  `RegisterFlags` and `AddGoFlagSet`, as above, or set `NoOptDefVal`.
- **Cobra prints usage on an error to `Out`,** not `Err` (cobra #1708).
  Set `SilenceUsage`, or a script that captures `Out` gets the usage.
- **Pass the streams unwrapped.** A `bufio` writer, a colour writer or an
  `io.MultiWriter` hides the descriptor: `Decide` then sees no terminal,
  unless the wrapper has its own `IsTerminal() bool`, and Bubble Tea
  enters no raw mode.

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
