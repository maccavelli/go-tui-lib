# go-tui-lib

Reusable Go code for terminal interfaces, built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Lip Gloss](https://github.com/charmbracelet/lipgloss), and the rest of the
[Charm](https://github.com/charmbracelet) stack.

go-tui-lib is a TUI layer for Go command-line programs. Bring your own
CLI, whether the standard `flag` package, Cobra, Kong or anything else, and
stack go-tui-lib on it: the same program then has a terminal UI beside its
native command line. The library imports no CLI framework, owns neither
the screen nor the signals, and writes only where the caller says
([0012-MADR](docs/decisions/0012-MADR-bring-your-own-cli.md)).

The packages come from other TUI-enabled Go projects, extracted while those
projects are worked through: one shared implementation of each interaction
pattern, used by every program that needs it.

## Status

- **The current release is `v2.4.0`,** at the module path
  `github.com/maccavelli/go-tui-lib/v2`
  ([0020-MADR](docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  A4.1). `v2.0.0` follows `v1.13.0`, which a `v1` user keeps; `v1.13.0` follows
  `v0.12.0`, tagged in place of `v0.13.0` (A4). Within a major version
  nothing is removed: a renamed name keeps working, deprecated, until the
  next major version, and [the migration guide](docs/guides/migrating.md)
  lists each change.
- **Multi-pane workspaces, since `v0.1.0`.** `layout` arranges panes: a main
  pane with a sidebar on either side, a bottom pane, a footer, responsive
  folds and saved state. `workspace` hosts them in a Bubble Tea program,
  with focus, resize, zoom, overlays, mouse and cursor. Since `v0.2.0` it
  draws into one reused cell buffer, measures text as Bubble Tea writes
  it, and follows the terminal's theme. `glyph`, `theme` and `tuitest` are
  the foundations every package uses. pi-go's agent session is the first
  consumer.
- **Terminal capabilities and services, since `v0.3.0`.** `termcap`
  learns the terminal's capabilities from one probe, over SSH and inside
  tmux, with a reason for every answer and a doctor report; `termsvc` sends
  notifications, clipboard writes and links that suit the terminal;
  `termcap/termcaptest` tests a program against fake terminals.
- **Commands, since `v0.4.0`.** `command` defines each action once, with
  its arguments as JSON Schema, its danger and its availability, and runs
  it from a key, a slash line or an agent, behind a gate the program
  controls; it loads command files, MCP prompts and ACP commands, and
  exports the commands to an agent as MCP tools. `when` evaluates
  availability in VS Code's when-clause grammar, and `workspace`
  publishes its own. A program's own CLI runs the same commands through
  `Registry.Run`.
- **Starting the TUI from any Go CLI, since `v0.7.0`.** `launch` decides
  whether the TUI can run on a program's own streams, with a reason; runs
  it with no signal handler; and falls back to the program's CLI mode when
  it cannot start or crashes, with the terminal as it was. Its flag types
  bind natively in the standard `flag` package, Cobra, Kong and
  urfave/cli; `launch/launchtest` tests the path with fake terminals.
- **Native forms for a program's own CLI, since `v0.8.0`.** `command`
  parses a command line (`ParseArgs`), describes and completes a command's
  arguments (`Params`, `Complete`), writes a result as text or JSON
  (`WriteResult`), gives each error an exit status, and turns a handler's
  panic into an error. `workspace` renders plain text for a pipe or a
  screen reader (`RenderPlain`) and takes its starting glyphs, profile,
  background and size as options; `termcap.EnvCaps` reads the
  environment's facts without a program; `termsvc` returns its bytes for a
  CLI to write.
- **Hardened, since `v0.8.0`.** Untrusted text, from a pane, a terminal's
  reply or a beacon, is cleaned before it reaches the screen, and a pane's
  view keeps only its styles and links. Command files, a request's
  arguments and the window have limits, and each package that parses
  input has a fuzz target.
- **One name for each idea, since `v0.9.0`.** Renamed names, whose old
  names `v0.10.0` removes; opaque option types; text forms for
  every enum; `when.SyntaxError`, `termcap.ErrUnknownName` and a `…Func`
  adapter for each hook; and `encoding/json/v2` in `command` and
  `termcap`.
- **Keymaps, since `v0.11.0`.** `keymap` binds keys to command IDs in
  scopes along a focus path, with chords, a leader, conflicts reported
  where they come from, and the user's own `keybindings.json` in VS
  Code's format, with a JSON Schema for their editor. It reads keys as a
  legacy or an enhanced terminal sends them, and picks defaults the
  terminal can deliver. The workspace's keys go through it, so a user
  rebinds them in the same file
  ([the keymaps guide](docs/guides/keymaps.md)).
- **Safe text and fuzzy matching without Bubble Tea, since `v1.13.0`.**
  `safetext` cleans untrusted text for a terminal, a string or a stream at
  a time, by rules a caller picks: escape sequences read as a terminal
  reads them, invisible and bidirectional controls removed or shown, and
  invalid UTF-8 as U+FFFD. `fuzzy` matches and ranks candidates by
  grapheme cluster. With `glyph`, `layout` and `when` they link no Charm
  module but `x/ansi`, so a line-mode CLI can use them
  ([0020-MADR](docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)).
  `glyph` gains status, heading, meter and control glyphs.
- **Terminal capabilities, commands and keymaps without Bubble Tea, since
  `v2.0.0`.** `termcap`'s probe is an engine that does no I/O, which a
  line-mode CLI drives itself; `termsvc` returns its bytes; `command`'s
  effects are plain functions; and `keymap` takes `termkey`'s keys, from
  `termkey`'s decoder or Bubble Tea's. Each links no Charm module but
  `x/ansi`, and its Bubble Tea forms are in `termcaptea`, `termsvctea`,
  `commandtea` and `keymaptea`
  ([the migration guide](docs/guides/migrating.md#migrating-to-v200)).
- **The command palette, since `v2.1.0`.** `palette` finds and runs
  anything from one overlay: the registry's commands, with their keys and
  danger, the panes, files from an `fs.FS` the program gives, and a
  program's own items, through providers that search off the event loop.
  It asks for a command's arguments before it runs it, puts recently used
  items first, and drives an editor's `@` and `/` completion pop-ups. Its
  keys are commands in the program's one keymap, run in the order they
  were pressed
  ([the command palette guide](docs/guides/command-palette.md)).
- **Defaults are the program's to refuse, since `v2.2.0`.** Each default
  the library applies has a control a program uses without forking, and
  [the defaults guide](docs/guides/defaults.md) lists them, with the few
  floors that have none. A program sets its own gate policy, forgets
  remembered answers, builds a registry without the built-ins, runs the
  workspace's own commands through its registry, makes `launch`'s decision
  with its helpers, puts the epilogue where it wants it, and binds the
  palette's commands under its own keys. `msgcat` is the message catalogue
  the packages' text will come from
  ([0024-MADR](docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md)).
- **A line-mode question, since `v2.3.0`.** `ask` asks a question inline,
  below a CLI's scrollback: rows of answers, each chosen by one key, a
  reason read after one, and the answer left in the scrollback in words.
  It draws through `inline`, which redraws a component's live rows in
  place and follows the terminal's re-wrap on a resize; both link no Charm
  module but `x/ansi`. `ask/asktea` asks the same question in a workspace
  overlay
  ([0023-MADR](docs/decisions/0023-MADR-line-mode-components.md)).
- **A line-mode picker, since `v2.4.0`.** `pick` picks one entry from a
  filtered list inline: a model with its context and prices, a session by
  title and time. The query is ranked by `fuzzy` as it is typed, the keys
  are commands in the program's keymap, and the choice is left in the
  scrollback as one line. It links no Charm module but `x/ansi`, and
  `palette.FromEntries` serves the same entries in the TUI's palette.
- **The stack is Charm v2:** `charm.land/bubbletea/v2`,
  `charm.land/lipgloss/v2` and `charm.land/bubbles/v2`, with
  `github.com/charmbracelet/colorprofile` and
  `github.com/charmbracelet/x/ansi`. The v1 `github.com/charmbracelet/…`
  paths are refused by lint.
- **Go 1.27.2** is required.
- **No CLI front end, since `v0.6.0`.** `command/cli` is gone, and the
  Cobra and Kong adapters, `command/cobracmd` and `command/kongcmd`, are
  retracted at `v0.1.1`. An adapter a TUI needs, such as the planned
  glamour one, is a nested module with its own tags, so the root module
  never requires its dependency.

## Documentation

Start at [docs/README.md](docs/README.md): the record index, and a table of
common tasks. [docs/architecture.md](docs/architecture.md) describes the
repository as it is now.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](docs/architecture.md) |
| build a main pane with a sidebar, a bottom pane and a footer | [the workspace guide](docs/guides/building-workspaces.md) |
| learn what the terminal supports, and send notifications, copies and links | [the terminal capabilities guide](docs/guides/terminal-capabilities.md) |
| define commands once, for keys, slash lines, agents and your own CLI | [the commands guide](docs/guides/commands.md) |
| find and run anything from a command palette, or complete `@` and `/` in an editor | [the command palette guide](docs/guides/command-palette.md) |
| start the TUI from my CLI with `--tui`, and fall back when it cannot run | [Start the TUI from your CLI](docs/guides/commands.md#start-the-tui-from-your-cli) |
| know which Charm version to use, and what may be imported | [AGENTS.md, Dependencies](AGENTS.md#dependencies) |
| know the rules every TUI package follows | [AGENTS.md, TUI conventions](AGENTS.md#tui-conventions) |
| refuse or change a default the library applies | [the defaults guide](docs/guides/defaults.md) |
| know what a type name means across the packages | [the glossary](docs/glossary.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |
| move to a new release's names | [the migration guide](docs/guides/migrating.md) |
| add a module, or release one | [the releasing guide](docs/guides/releasing.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
