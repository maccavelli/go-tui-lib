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

- **No package yet, and no release.** The first package, such as
  `updatetea` (the Bubble Tea adapter for go-core-lib's `selfupdate`) or an
  extraction from ocp-login, is its own record.
- **The stack is Charm v2:** `charm.land/bubbletea/v2`,
  `charm.land/lipgloss/v2` and `charm.land/bubbles/v2`, with
  `github.com/charmbracelet/colorprofile` and
  `github.com/charmbracelet/x/ansi`. The v1 `github.com/charmbracelet/…`
  paths are refused by lint.
- **Go 1.27.1** is required.
- **Until the first package lands,** `make test`, `make vet`, `make lint` and
  `make vuln` fail with "no packages". That is expected.

## Documentation

Start at [docs/README.md](docs/README.md): the record index, and a table of
common tasks. [docs/architecture.md](docs/architecture.md) describes the
repository as it is now.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](docs/architecture.md) |
| know which Charm version to use, and what may be imported | [AGENTS.md, Dependencies](AGENTS.md#dependencies) |
| know the rules every TUI package follows | [AGENTS.md, TUI conventions](AGENTS.md#tui-conventions) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
