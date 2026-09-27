# go-tui-lib

Reusable Go code for terminal interfaces, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and the rest of the [Charm](https://github.com/charmbracelet) stack.

The packages here come from other TUI-enabled Go projects, extracted while those projects are worked through. The aim is to modularize and canonicalize as much of their TUI, UX, and UI as possible: one shared implementation of each interaction pattern, used by every program that needs it.

The repository starts without packages. Code lands as each source project is taken apart and the pieces that more than one program can use are given a stable API.
