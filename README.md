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

- **The current release is `v0.7.0`.** `v0` means the API may still
  change.
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
- **The stack is Charm v2:** `charm.land/bubbletea/v2`,
  `charm.land/lipgloss/v2` and `charm.land/bubbles/v2`, with
  `github.com/charmbracelet/colorprofile` and
  `github.com/charmbracelet/x/ansi`. The v1 `github.com/charmbracelet/…`
  paths are refused by lint.
- **Go 1.27.1** is required.
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
| start the TUI from my CLI with `--tui`, and fall back when it cannot run | [Start the TUI from your CLI](docs/guides/commands.md#start-the-tui-from-your-cli) |
| know which Charm version to use, and what may be imported | [AGENTS.md, Dependencies](AGENTS.md#dependencies) |
| know the rules every TUI package follows | [AGENTS.md, TUI conventions](AGENTS.md#tui-conventions) |
| know what a type name means across the packages | [the glossary](docs/glossary.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |
| add a module, or release one | [the releasing guide](docs/guides/releasing.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
