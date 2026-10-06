# go-tui-lib

Reusable Go code for terminal interfaces, built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Lip Gloss](https://github.com/charmbracelet/lipgloss), and the rest of the
[Charm](https://github.com/charmbracelet) stack.

The packages here come from other TUI-enabled Go projects, extracted while
those projects are worked through. The aim is to modularize and canonicalize
as much of their TUI, UX, and UI as possible: one shared implementation of
each interaction pattern, used by every program that needs it.

The repository starts without packages. Code lands as each source project is
taken apart and the pieces that more than one program can use are given a
stable API.

## Status

- **The current release is `v0.4.0`.** `v0` means the API may still
  change.
- **Multi-pane workspaces, since `v0.1.0`.** `layout` arranges panes: a main
  pane with a sidebar on either side, a bottom pane, a footer, responsive
  folds and saved state. `workspace` hosts them in a Bubble Tea program,
  with focus, resize, zoom, overlays, mouse and cursor, drawing into one
  reused cell buffer, measuring text as Bubble Tea writes it, and following
  the terminal's theme. `glyph`, `theme` and `tuitest` are the foundations
  every package uses. pi-go's agent session is the first consumer.
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
  availability in VS Code's when-clause grammar, `command/cli` runs the
  same commands from the shell, and `workspace` publishes its own.
- **The stack is Charm v2:** `charm.land/bubbletea/v2`,
  `charm.land/lipgloss/v2` and `charm.land/bubbles/v2`, with
  `github.com/charmbracelet/colorprofile` and
  `github.com/charmbracelet/x/ansi`. The v1 `github.com/charmbracelet/…`
  paths are refused by lint.
- **Go 1.27.1** is required.
- **Adapters ship as their own modules.** The planned Cobra, Kong and
  glamour adapters are nested modules with their own tags, so the root
  module never requires them.

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
| define commands once, for keys, slash lines, agents and the shell | [the commands guide](docs/guides/commands.md) |
| know which Charm version to use, and what may be imported | [AGENTS.md, Dependencies](AGENTS.md#dependencies) |
| know the rules every TUI package follows | [AGENTS.md, TUI conventions](AGENTS.md#tui-conventions) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |
| add a module, or release one | [the releasing guide](docs/guides/releasing.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
