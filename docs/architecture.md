# Architecture

`go-tui-lib` is the repository of the Go module
`github.com/maccavelli/go-tui-lib`: a library of terminal-UI packages on
the Charm v2 stack, which a Go program stacks on its own command line
to have a TUI beside it. It has no binary. A nested adapter module may sit
beside it to carry a dependency the root must not; none is built (Modules,
below).

## What it is

- **A library only.** Each capability is a top-level directory, with the
  package named after it. There is no root package. Helpers shared between
  packages go under `internal/`.
- **Go 1.27.1**, with no `toolchain` line.
- **Ten packages,** and four internal ones, for multi-pane terminal
  workspaces, terminal capabilities and services, a command registry run
  from keys, slash lines, agents and the program's own CLI, and the
  foundations every package uses.
- **No CLI front end.** The program brings its own command line, and the
  library imports no CLI framework
  ([0012-MADR](decisions/0012-MADR-bring-your-own-cli.md)).

## Packages

```text
 workspace     Bubble Tea pane host       → layout, theme, glyph, internal/cells, command, when; bubbletea, lipgloss, bubbles/key, bubbles/help
 command       the command registry       → when; bubbletea
 when          availability expressions   → standard library
 internal/cells  the reused frame buffer  → layout; ultraviolet, x/ansi
 termsvc       terminal services          → termcap; bubbletea, x/ansi
 termcap       capabilities, the probe    → internal/termevent; bubbletea, x/ansi, colorprofile
 termcap/termcaptest  fake terminals      → termcap; bubbletea, x/ansi, colorprofile (tests)
 internal/termevent   pass-through events → ultraviolet
 internal/termevent/termeventtest  those events for tests → ultraviolet, bubbletea
 theme         palettes, roles, styles    → glyph; lipgloss, colorprofile
 glyph         Unicode and ASCII glyphs   → standard library
 layout        geometry and state         → standard library
 tuitest       golden rendering           → x/ansi (tests and examples only)
```

| Package | What it holds |
| :--- | :--- |
| `when` | `Parse` and `MustParse` → `Expr` (`Eval`, `String`, `Keys`), VS Code's when-clause grammar with RE2 regexes; `Check` against `Keys`; `Context`, `Map`, `Layered`, `Value`; typed `Key[T]` |
| `command` | `Command`, `ID`, `New[A]` and its options, `SchemaOf` (JSON Schema 2020-12 from Kong-aligned tags), `ArgError`; `Registry` (`Register`, `ReplaceSource`, `Lookup`, `Slash`, `ParseSlash`, `All`, `Available`, `Watch`, `Dispatch`, `Run`, `Cancel`); the policy, `Gate`, `Decision`, `Auditor`, `SlogAuditor`; `LoadDir`, `FromMCPPrompts`, `FromACP`; `MCPTools`, `CallMCP`, `ACPCommands`, `Manifest`; `WithLoop` and `LoopMsg`; the messages |
| `glyph` | `Set` (4 border styles, separators with a cross and four tees, focus marker, ellipsis, scroll, bullet, badge brackets), `Unicode()`, `ASCII()`, `For(utf8)`; every glyph one cell |
| `theme` | `Palette` for dark, light and unknown backgrounds; `LightDarkColor` and `ProfileColor`; `Styles`; `New(profile, background, glyphs)` with `WithPalette` and `WithPaletteFor`; `FromDark`; `Border(style)` |
| `layout` | `Rect`, `Size` (fixed, percent, ratio, fill; min, max, shrink order), `Node` (`Pane`, `Split`, `Responsive`, or a custom node), `Solve` → `Plan`; `State` (JSON); the sidebar presets |
| `workspace` | `Pane` and its optional interfaces; `Workspace` (routing, focus, chrome, resize, zoom, hide, overlays, cursor, `View`, `help.KeyMap`, the width method, a following theme, `SetBackground`, `Panes`, `PaneAs`, `WhenContext`); `Wrap`; `KeyMap`; `Commands` and the context keys |
| `internal/cells` | `Frame`: a reused cell buffer that draws strings into rectangles with a chosen width method |
| `termcap` | `Prober` (one batch of queries ended by DA1, a deadline, tea's own replies observed, mode 2031 and its reset, `IsReplyFragment`); `Caps` of `Fact`s with origin and reason; `FromEnv` and `Identity`; the keyboard, link and notification views; `TmuxQuery`; `Report` and `Findings` |
| `termcap/termcaptest` | `Terminal`, a scripted fake terminal; `Profile` and seven profiles; `Run` |
| `termsvc` | `Notifier` (OSC 99, 777, 9 or the bell; focus policies; `NotifyResultMsg`); `Copy` and `CopiedMsg`; `Link`, `LinkDisplay`, `LinkPolicy`; prompt marks; `Wrap`; `SanitizeTitle`; the activity beacon; `Pointer`; `ProgressSupported` |
| `internal/termevent` | `Decode`: the ultraviolet events tea passes through untranslated, as plain values |
| `internal/termevent/termeventtest` | those events built for tests outside `internal/termevent` |
| `tuitest` | `Golden` across {colour, no colour} × {UTF-8, ASCII} × widths; `Text` for a single file; `Annotate` |

- **`layout` has no Charm import,** so its solver can serve any front end.
- **Imports point downward.** `when` imports only the standard library,
  so the keymap and any front end can use it. `command` imports `when`
  and Bubble Tea, for `tea.Cmd` and `tea.Msg` only. `workspace` imports
  both to publish its commands and context keys; `command` never imports
  `workspace` ([0006-MADR](decisions/0006-MADR-command-registry.md)
  §1).
- **The registry reads lock-free.** Each write publishes a new immutable
  snapshot behind an `atomic.Pointer`; `Lookup`, `Available` and the
  exports read one, from any goroutine. A `Loop` command runs on the
  program's event loop, through `Dispatch` from `Update`, or through
  `WithLoop`'s `LoopMsg` when an agent's goroutine calls `Run` (A7).
- **The registry speaks no protocol.** `MCPTool`, `MCPCallResult` and
  `ACPCommand` are plain structs with MCP 2026-07-28's and ACP's field
  names; no SDK is imported, and the host owns the transport.
- **`workspace` draws each frame into one reused buffer,**
  `internal/cells.Frame`: the panes, then the separators, then the overlays,
  each into its rectangle. It records those rectangles, top first, and a
  mouse event takes the first that contains it. A frame is drawn only when
  something changed since the last; otherwise the last one is returned.
  The view cache is keyed on a pane's kind, ID, size, focus, width method
  and theme generation
  ([0004-MADR](decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) §2).
- **The workspace measures as Bubble Tea writes:** `ansi.WcWidth` by
  default, switching to `ansi.GraphemeWidth` on the terminal's mode 2027
  report, unless `WithWidthMethod` fixes it (§3).
- **`internal/cells` and `internal/termevent` are the only importers of
  ultraviolet,** which has no tagged release, so an upstream change is one
  package's fix (§1; [0005-MADR](decisions/0005-MADR-terminal-capabilities-and-services.md) §1).
  `internal/termevent/termeventtest`, under the second, builds its events
  for other packages' tests.
- **`termcap` learns, and `termsvc` acts.** The probe reads the
  environment from `tea.EnvMsg` only, never from the process; neither
  package starts a process, and commands such as `TmuxQuery` are returned
  for the program to run
  ([0005-MADR](decisions/0005-MADR-terminal-capabilities-and-services.md)).
  A source test in each refuses `os.Getenv`, `os.LookupEnv`, `os.Environ`
  and `os/exec`.
- **Nothing writes to the terminal.** No package writes to `os.Stdout` or
  `os.Stderr`, prints with `fmt.Print*` or the `print` builtins, logs
  through `log`'s standard logger or `log/slog`'s default logger, calls
  `signal.Notify` or sets `AltScreen`; `internal/conformance` checks
  this by type (see Tooling).

## Modules

0010-MADR
([0010-MADR-nested-adapter-modules.md](decisions/0010-MADR-nested-adapter-modules.md))
puts an adapter with a dependency the root must not carry in a nested
module. The Cobra and Kong front ends it planned, `command/cobracmd` and
`command/kongcmd`, were built and released at `v0.1.0`, then retired by
[0012-MADR](decisions/0012-MADR-bring-your-own-cli.md): retracted at
`v0.1.1` and removed. `stream/glamourmd` is planned:

```text
 stream/glamourmd   (planned)   → root vX.Y.Z, published; glamour
 .                  (root)      → the Charm v2 stack; never an adapter
```

- **The dependency runs one way.** An adapter requires a published root
  version, with no `replace`; the root never requires an adapter. A
  consumer of the root therefore never sees an adapter's dependency in
  its `go.sum` or module graph (0010-REPORT §2).
- **Each module has its own tags:** `vX.Y.Z` for the root,
  `<dir>/vX.Y.Z` for an adapter. [guides/releasing.md](guides/releasing.md)
  has the procedure.
- **`go.work`** lists every module, for development, and is committed;
  `go.work.sum` is ignored (0010-MADR A1). Every gate also runs
  with `GOWORK=off` (Tooling).

## Tree

```text
README.md                   repository entry; links here
LICENSE                     Apache License 2.0
AGENTS.md                   rules for agents: dependencies, TUI conventions,
                            records, checks, identifiers, commits
go.mod, go.sum              the root module and its requirements
go.work                     the workspace: every module, for development
                            (go.work.sum is ignored)
Makefile                    development targets (below)
.golangci.yml               golangci-lint configuration, with depguard
.markdownlint-cli2.jsonc    Markdown lint configuration
.gitattributes              LF line endings everywhere
.gitignore
.github/workflows/
  ci.yml                    CI
scripts/
  go-precheck.sh            the pre-add check, per module
  go-modules.sh             lists the modules; --check compares go.work
                            with the tracked go.mod files
  go-modules_test.sh        its offline test
  go-fuzz.sh                fuzzes each fuzz target of a package in turn
  go-fuzz_test.sh           its offline test
glyph/ theme/ layout/ workspace/ tuitest/ termcap/ termsvc/
when/ command/              the packages; goldens under each testdata/golden/
termcap/termcaptest/        fake terminals for tests
internal/cells/             the reused frame buffer, an ultraviolet importer
internal/termevent/         pass-through events, the other ultraviolet importer
internal/conformance/       the terminal-ownership scan (tests only)
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  decisions/                MADR and PLAN records
  reports/                  REPORT records
  guides/                   how-to guides: workspaces, terminal
                            capabilities, commands, releasing
```

## Dependencies

- **Required:** `charm.land/bubbletea/v2` v2.0.10, `charm.land/lipgloss/v2`
  v2.0.6, `charm.land/bubbles/v2` v2.2.1,
  `github.com/charmbracelet/colorprofile` v0.4.3,
  `github.com/charmbracelet/x/ansi` v0.11.8, and
  `github.com/charmbracelet/ultraviolet` at
  `v0.0.0-20260811164956-006e29f97886`, the pseudo-version lipgloss's
  requirement selects; it has no tagged release
  ([0004-MADR](decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) §1).
- **Named but not yet required:** `github.com/maccavelli/go-selfupdate-lib`
  (formerly `go-core-lib`), for `updatetea`.
- **Refused by `depguard`,** in source and tests, in every module:
  - the Charm v1 paths `github.com/charmbracelet/bubbletea`, `…/lipgloss`
    and `…/bubbles`;
  - `github.com/maccavelli/mcplib`;
  - `github.com/modelcontextprotocol/go-sdk`;
  - `github.com/maccavelli/go-llmprovider-sdk`;
  - fang: `charm.land/fang/v2` and `github.com/charmbracelet/fang`;
  - the CLI frameworks `github.com/spf13/cobra`, `github.com/spf13/pflag`
    and `github.com/alecthomas/kong`
    ([0012-MADR](decisions/0012-MADR-bring-your-own-cli.md)).
- **Kept to one module by `depguard`:** `charm.land/glamour/v2` to
  `stream/glamourmd`. The rule covers `$all` less `!**/<dir>/**`.
- **Kept to two packages by `depguard`:** `github.com/charmbracelet/ultraviolet`
  to `internal/cells` and `internal/termevent` (with
  `internal/termevent/termeventtest` beneath it), test files included.

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`,
  `modernize`, `tidy`, `tidy-check`, `vuln`, `fuzz`, `pre-add-check`,
  `release-check`, `help`.
- **Per module.** `test`, `vet`, `lint`, `modernize`, `tidy-check` and
  `vuln` run once in each module's directory with `GOWORK=off`, taking the
  list from `scripts/go-modules.sh`; a failure to list the modules fails
  the target.
- **`make lint`** runs `make modernize`, then `golangci-lint run -c
  <root>/.golangci.yml ./...` three times per module: `GOOS=linux`,
  `darwin` and `windows`, each with `CGO_ENABLED=0`.
- **`make modernize`** runs `go fix -diff ./...` for the same three
  targets, with `CGO_ENABLED=0`, and fails on any suggestion: `go fix
  -diff` exits 1 when it prints a diff.
- **`make release-check`** runs the pre-add check over every module and
  file, before a tag.
- **`internal/conformance`** is the terminal-ownership scan, run by
  `go test`.
  - It finds every module by its `go.mod`, and type-checks each module's
    packages from that module's directory with `go/types` and the source
    importer (`importer.ForCompiler(fset, "source", nil)`), which resolves
    imports with `go list` in the working directory. cgo is off for the
    scan, so no C toolchain is needed.
  - Every identifier is resolved to its object, so a renamed or dot
    import, a function value or a method value is a use, and a local name
    that only matches a rule is not.
  - Outside tests it refuses `os.Stdout` and `os.Stderr`; `fmt.Print`,
    `Printf` and `Println`; `log`'s writing functions and `log.Default`
    and `log.Writer`; `log/slog`'s `Debug`, `Info`, `Warn` and `Error`
    and their `Context` forms, `Log`, `LogAttrs` and `Default`; the
    `print` and `println` builtins; `signal.Notify`; and a write to
    `tea.View`'s `AltScreen`, however the field is reached.
  - It reads each package's files for the host's `GOOS`; CI's three
    operating systems cover the rest.
- **`make fuzz`** fuzzes each fuzz target of `layout`, `when` and
  `command` for `FUZZTIME` (default 20s): `FuzzSolve`, `FuzzParse`,
  `FuzzParseSlash` and `FuzzFrontMatter`.
- **`scripts/go-precheck.sh`** runs, for each module that owns a given
  file (every module when none is given), in the module's directory:
  `gofmt` on its files; with `GOWORK=off`, the same three golangci-lint
  runs, `go vet` and `go test` on their packages, `go mod tidy -diff` and
  `govulncheck ./...`; `go test` again in workspace mode; and, for a
  nested module, no `replace` and a release version of the root. It ends
  with `scripts/go-modules.sh --check`. `make pre-add-check` runs it, and
  so does the machine-wide agent gate before an agent `git commit` that
  stages Go files.
- **`.golangci.yml`:**
  - 21 linters, including `depguard`;
  - `revive`'s `exported`, `package-comments` and `var-naming` rules in
    place of `golint`;
  - errcheck with `check-blank` and `check-type-assertions`;
  - gofmt and goimports as formatters;
  - test files exempt from `errcheck`, `gosec`, `unparam`, `revive`,
    `gocritic` and `goconst`, but not from `depguard`.
- **Golden files** are written with
  `go test ./<pkg>/ -run <Test> -tuitest.update`, or with
  `TUITEST_UPDATE=1 go test ./...`, and read before they are committed. A
  test binary that defines its own boolean `-update` may use it instead.
- **CI** (`.github/workflows/ci.yml`) reads the Go version from `go.work`
  and caches on every module's `go.sum`. It has three jobs:
  - **`modules`** (Linux): the test of `go-modules.sh`, then
    `go-modules.sh --check`, then the module list as the matrix's input.
  - **`test (<module>, <os>)`** for each module on `ubuntu-24.04`,
    `macos-15` and `windows-2025`, in the module's directory with
    `GOWORK=off`:
    - **every OS:** `go test`, and `go test` again in workspace mode;
    - **Linux and macOS:** `go test -race`;
    - **Linux:** `go test -shuffle=on -count=2` and `LC_ALL=C go test`.
  - **`gates`** (Linux):
    - the fuzz script's test, then `make fuzz`, uploading the three
      packages' corpora as an artifact on failure;
    - `go vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`, per
      module;
    - `make vet`, `gofmt`, `make tidy-check`, and `make lint` (with
      `make modernize`) with golangci-lint v2.14.0;
    - `make vuln` with govulncheck v1.8.0;
    - `shellcheck` v0.11.0 (pinned by SHA-256 and first on `PATH`),
      `markdownlint-cli2` 0.23.2 and `actionlint` v1.7.12.
  - One run per ref (`concurrency`, cancel in progress). Actions are pinned
    to commit SHAs.

## What is not here

- **Standard panes** (log tail, metrics view, scrolling text and
  Markdown), **overlay widgets** (dialog, picker, palette, toast), a **help
  footer** and **`updatetea`.** Each is its own record.
- **The keymap engine and the command palette**
  ([0007-MADR](decisions/0007-MADR-keymap-engine.md),
  [0008-MADR](decisions/0008-MADR-command-palette.md)), which bind keys to
  command IDs and list the registry's commands. The workspace's own key
  bindings stay until the keymap moves them onto these IDs.
- **Terminal modes and inline scrollback** (`termmode`, `inline`): mode
  plans, teardown, restore bytes and the Windows console helpers, and the
  per-terminal scrollback strategy. Each is a later record; until then tea
  still asks for Kitty disambiguation everywhere
  ([0005-MADR](decisions/0005-MADR-terminal-capabilities-and-services.md) A4).
- **Image protocols** (Kitty placeholders, Sixel, iTerm2), which add
  `Query` values to `termcap`.
- **`make apicheck`.** It needs a `v1` tag to compare against, and comes
  with the `v1` record.
- **A CLI front end,** by design
  ([0012-MADR](decisions/0012-MADR-bring-your-own-cli.md)). Helpers that
  make stacking easier, such as launching the TUI from a program's own
  subcommand, or choosing interactive or plain output, are a later
  record.
- **A release workflow, Dependabot, and any tag** other than the owner's.
