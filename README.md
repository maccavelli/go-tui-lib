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

- **Multi-pane workspaces, `v0.1.6`.** `layout` arranges panes: a main
  pane with a sidebar on either side, a bottom pane, a footer, responsive
  folds and saved state. `workspace` hosts them in a Bubble Tea program,
  with focus, resize, zoom, overlays, mouse and cursor. `glyph`, `theme`
  and `tuitest` are the foundations every package uses. pi-go's agent
  session is the first consumer. `v0` means the API may still change.
- **On `main`, for `v0.2.0`:** the workspace draws into one reused cell
  buffer and only when something changed; it measures text as Bubble Tea
  writes it; its theme follows the terminal's profile and background; it is
  a `help.KeyMap` with a `View()`; and it adds `PaneAs`, `Panes` and
  `Plan.All`. The default width method changes to wcwidth, and
  `WithWidthMethod(ansi.GraphemeWidth)` keeps `v0.1`'s.
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
| know which Charm version to use, and what may be imported | [AGENTS.md, Dependencies](AGENTS.md#dependencies) |
| know the rules every TUI package follows | [AGENTS.md, TUI conventions](AGENTS.md#tui-conventions) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |
| add a module, or release one | [the releasing guide](docs/guides/releasing.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
