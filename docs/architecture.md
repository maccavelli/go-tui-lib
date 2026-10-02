# Architecture

`go-tui-lib` is the Go module `github.com/maccavelli/go-tui-lib`: a library
of terminal-UI packages on the Charm v2 stack. It has no binary.

## What it is

- **A library only.** Each capability is a top-level directory, with the
  package named after it. There is no root package. Helpers shared between
  packages go under `internal/`.
- **Go 1.27.1**, with no `toolchain` line.
- **Five packages,** for multi-pane terminal workspaces and the foundations
  every package uses.

## Packages

```text
 workspace     Bubble Tea pane host       → layout, theme, glyph; bubbletea, lipgloss, bubbles/key
 theme         palettes, roles, styles    → glyph; lipgloss, colorprofile
 glyph         Unicode and ASCII glyphs   → standard library
 layout        geometry and state         → standard library
 tuitest       golden rendering           → x/ansi (tests and examples only)
```

| Package | What it holds |
| :--- | :--- |
| `glyph` | `Set` (4 border styles, separators, focus marker, ellipsis, scroll, bullet, badge brackets), `Unicode()`, `ASCII()`, `For(utf8)`; every glyph one cell |
| `theme` | `Palette` for dark, light and unknown backgrounds; `Styles`; `New(profile, background, glyphs)`; `Border(style)` |
| `layout` | `Rect`, `Size` (fixed, percent, ratio, fill; min, max, shrink order), `Node` (`Pane`, `Split`, `Responsive`, or a custom node), `Solve` → `Plan`; `State` (JSON); the sidebar presets |
| `workspace` | `Pane` and its optional interfaces; `Workspace` (routing, focus, chrome, resize, zoom, hide, overlays, cursor); `KeyMap` |
| `tuitest` | `Golden` across {colour, no colour} × {UTF-8, ASCII} × widths; `Text` for a single file; `Annotate` |

- **`layout` has no Charm import,** so its solver can serve any front end.
- **`workspace` composes the frame on a Lip Gloss canvas.** Each pane is a
  layer clipped to its rectangle; separators sit at Z 1 and overlays at Z 10
  and up. The same compositor answers mouse hit tests.
- **Nothing writes to the terminal.** No package writes to `os.Stdout` or
  `os.Stderr`, calls `signal.Notify` or sets `AltScreen`;
  `internal/conformance` checks this.

## Tree

```text
README.md                   repository entry; links here
LICENSE                     Apache License 2.0
AGENTS.md                   rules for agents: dependencies, TUI conventions,
                            records, checks, identifiers, commits
go.mod, go.sum              the module and its five requirements
Makefile                    development targets (below)
.golangci.yml               golangci-lint configuration, with depguard
.markdownlint-cli2.jsonc    Markdown lint configuration
.gitattributes              LF line endings everywhere
.gitignore
.github/workflows/
  ci.yml                    CI
scripts/
  go-precheck.sh            the pre-add check
  go-fuzz.sh                fuzzes each fuzz target of a package in turn
  go-fuzz_test.sh           its offline test
glyph/ theme/ layout/ workspace/ tuitest/
                            the packages; goldens under each testdata/golden/
internal/conformance/       the terminal-ownership scan (tests only)
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  decisions/                MADR and PLAN records
  reports/                  REPORT records
  guides/                   how-to guides
```

## Dependencies

- **Required:** `charm.land/bubbletea/v2` v2.0.10, `charm.land/lipgloss/v2`
  v2.0.6, `charm.land/bubbles/v2` v2.2.1,
  `github.com/charmbracelet/colorprofile` v0.4.3 and
  `github.com/charmbracelet/x/ansi` v0.11.8.
- **Named but not yet required:** `github.com/maccavelli/go-core-lib`, for
  `updatetea`.
- **Refused by `depguard`,** in source and tests:
  - the Charm v1 paths `github.com/charmbracelet/bubbletea`, `…/lipgloss`
    and `…/bubbles`;
  - `github.com/maccavelli/mcplib`;
  - `github.com/modelcontextprotocol/go-sdk`;
  - `github.com/maccavelli/go-llmprovider-sdk`.

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`, `tidy`,
  `vuln`, `fuzz`, `pre-add-check`, `help`.
- **`make lint`** runs `golangci-lint run -c .golangci.yml ./...` three
  times: `GOOS=linux`, `darwin` and `windows`, each with `CGO_ENABLED=0`.
- **`make fuzz`** fuzzes `layout`'s fuzz target for `FUZZTIME` (default
  20s).
- **`scripts/go-precheck.sh`** runs `gofmt` on the given Go files, the same
  three golangci-lint runs, `go vet` and `go test` on their packages, and
  `govulncheck ./...`. `make pre-add-check` runs it, and so does the
  machine-wide agent gate before an agent `git commit` that stages Go files.
- **`.golangci.yml`:**
  - 21 linters, including `depguard`;
  - `revive`'s `exported`, `package-comments` and `var-naming` rules in
    place of `golint`;
  - errcheck with `check-blank` and `check-type-assertions`;
  - gofmt and goimports as formatters;
  - test files exempt from `errcheck`, `gosec`, `unparam`, `revive`,
    `gocritic` and `goconst`, but not from `depguard`.
- **Golden files** are written with `go test ./<pkg>/ -run <Test> -update`,
  and read before they are committed.
- **CI** (`.github/workflows/ci.yml`) runs on `ubuntu-24.04`, `macos-15` and
  `windows-2025`, with the Go version read from `go.mod`.
  - **Every OS:** `go test`.
  - **Linux and macOS:** `go test -race`.
  - **Linux also:**
    - `go test -shuffle=on -count=2` and `LC_ALL=C go test`;
    - the fuzz script's test, then `make fuzz`, uploading the corpus as an
      artifact on failure;
    - `go vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`;
    - `go vet`, `gofmt`, `go mod tidy -diff`, and `make lint` with
      golangci-lint v2.14.0;
    - `govulncheck` v1.8.0;
    - `shellcheck` v0.11.0 (pinned by SHA-256 and first on `PATH`),
      `markdownlint-cli2` 0.23.2 and `actionlint` v1.7.12.
  - One run per ref (`concurrency`, cancel in progress). Actions are pinned
    to commit SHAs.

## What is not here

- **Standard panes** (log tail, metrics view, scrolling text and
  Markdown), **overlay widgets** (dialog, picker, palette, toast), a **help
  footer** and **`updatetea`.** Each is its own record.
- **`make apicheck`.** It needs a `v1` tag to compare against, and comes
  with the `v1` record.
- **A release workflow, Dependabot, and any tag** other than the owner's.
