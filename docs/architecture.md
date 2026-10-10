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
- **Go 1.27.2**, with no `toolchain` line.
- **Ten packages,** and five internal ones, for multi-pane terminal
  workspaces, terminal capabilities and services, a command registry run
  from keys, slash lines, agents and the program's own CLI, and the
  foundations every package uses.
- **No CLI front end.** The program brings its own command line, and the
  library imports no CLI framework
  ([0012-MADR](decisions/0012-MADR-bring-your-own-cli.md)).

## Packages

```text
 launch        start the TUI from a CLI   → command, glyph, termcap, internal/enum, internal/limits; bubbletea, colorprofile, x/term
 launch/launchtest  fake streams for tests → launch, termcap (tests)
 workspace     Bubble Tea pane host       → layout, theme, glyph, internal/cells, internal/enum, internal/limits, internal/sanitize, command, when; bubbletea, bubbles/key, bubbles/help, colorprofile, x/ansi
 command       the command registry       → when, internal/enum, internal/sanitize, internal/teamsg; bubbletea
 when          availability expressions   → internal/enum
 internal/cells  the reused frame buffer  → layout; ultraviolet, x/ansi
 termsvc       terminal services          → termcap, internal/enum, internal/sanitize, internal/teamsg; bubbletea, x/ansi
 termcap       capabilities, the probe    → internal/termevent, internal/enum, internal/sanitize, internal/teamsg; bubbletea, x/ansi, colorprofile
 termcap/termcaptest  fake terminals      → termcap; bubbletea, x/ansi, colorprofile (tests)
 internal/termevent   pass-through events → ultraviolet
 internal/termevent/termeventtest  those events for tests → ultraviolet, bubbletea
 theme         palettes, roles, styles    → glyph, internal/enum; lipgloss, colorprofile
 glyph         Unicode and ASCII glyphs   → internal/enum
 internal/enum  enum names and text forms → standard library
 internal/teamsg  the one-message command → bubbletea
 internal/sanitize  untrusted display text → x/ansi
 internal/limits  the window clamp        → standard library
 layout        geometry and state         → internal/enum
 tuitest       golden rendering           → glyph; x/ansi, colorprofile (tests and examples only)
```

| Package | What it holds |
| :--- | :--- |
| `when` | `Parse` and `MustParse` → `Expr` (`Eval`, `String`, `Keys`), VS Code's when-clause grammar with RE2 regexes, or a `SyntaxError` with its offset; `Check` against `Keys`; `Context`, `Map`, `Layered`, `Value`; typed `Key[T]` |
| `command` | `Command`, `ID`, `New[A]` and `MustNew[A]` and their options, `SchemaOf` (JSON Schema 2020-12 from Kong-aligned tags), `ArgsOf`, `ArgError`; `Registry` (`NewRegistry` with `WithGate`, `WithAuditor`, `WithPrefixer` and `WithLoop`; `Attach`, a loop that can be detached; `Register`, `ReplaceSource`, `Lookup`, `Slash`, `ParseSlash`, `ParseArgs`, `Complete`, `All`, `Available`, `Watch`, `WatchContext`, `Dispatch`, `Run`, `Cancel`, `CancelAll`, `Remove`, `Version`); `Params` and `Param`; `Format` and `WriteResult`; the policy, `Gate`, `GateFunc`, `AllowIf`, `Verdict`, `Auditor`, `AuditorFunc`, `SlogAuditor`; a `Request`'s `WhenContext`; the errors, each with its exit status, and `PanicError`; `LoadDir` and `LoadDirWith`, with its limits; `WithMaxArgBytes`; `FromMCPPrompts`, `FromACP`; `MCPTools`, `CallMCP`, `ACPCommands`, `Manifest`; `WithLoop` and `LoopMsg`; the messages |
| `glyph` | `Set` (4 border styles, separators with a cross and four tees, focus marker, ellipsis, scroll, bullet, badge brackets), `Unicode()`, `ASCII()`, `For(utf8)`; `Tier` (`TierUnicode`, `TierLegacy`, `TierASCII`) and `Tier.Set`; every glyph one cell |
| `launch` | `Choice` and `Flags` (`--mode`, `--tui`, bound natively in each framework); `Streams`, `StreamSource`, `FromSource`; `Config`, `Decide` → `Decision` with `Target`s and a `Reason` token; `Run[M]` with `WithRegistry`, `WithRestorer`, `OnStart`, `WithProgramOptions`, `WithFilter`; `ErrNotStarted`, `ErrCrashed`; `Frame`; `ExitCode`, `ExitError`; `Restorer` |
| `launch/launchtest` | `Terminal` (a fake terminal stream with a size and typed keys), `Pipe`, `Streams`, `Env` |
| `internal/enum` | `Names`, an enum's tokens, and `Bits`, a bit set's, for an enum's `String`, `MarshalText` and `UnmarshalText`, on `uint8` or `int`; `Name`, `Marshal` and `Unmarshal` over a bare table |
| `internal/teamsg` | `Cmd`, the `tea.Cmd` that delivers one message |
| `theme` | `Palette` for dark, light and unknown backgrounds; `LightDarkColor` and `ProfileColor`; `Styles`; `New(profile, background, glyphs)` with `WithPalette` and `WithPaletteFor`; `FromDark`; `Border(style)`; text forms for `Background` and `BorderStyle` |
| `layout` | `Rect`, `Size` (fixed, percent, ratio, fill; min, max, shrink order), `Node` (`Pane`, `Split`, `Responsive`, or a custom node), `Solve` → `Plan`; `State` (JSON); the sidebar presets and their `With…` options |
| `workspace` | `Pane` and its optional interfaces, `PlainViewer` among them; `Workspace` (routing, focus, chrome, resize, zoom, hide, overlays, cursor, `View`, `RenderPlain`, `help.KeyMap` and `Help`, the width method, a following theme with `WithGlyphs`, `WithProfile`, `WithBackground` and `GlyphThemeBuilder`, `WithSize`, clamped to `MaxSide` and `MaxCells`, `SetBackground`, `Panes`, `PaneAs`, `WhenContext`); `Wrap`; `KeyMap`; `Commands` and the context keys, `FocusedPaneKey()` to `HeightKey()` |
| `internal/cells` | `Frame`: a reused cell buffer that draws strings into rectangles with a chosen width method |
| `internal/sanitize` | `Line` for one line of plain text and `Styled` for a pane's view, which keeps SGR and OSC 8 only; `Truncate`, never wider than asked; `Token`; `HasControl` |
| `internal/limits` | `MaxSide`, `MaxCells` and `Clamp`, the window clamp `workspace` and `launch` share |
| `termcap` | `Prober`, from `NewProber` (one batch of queries ended by DA1, a deadline, tea's own replies observed, mode 2031 and its reset, `IsReplyFragment`); `EnvCaps`, the environment's facts with no program; `Caps` of `Fact`s with origin and reason, as JSON (v2); `ErrUnknownName`; `Env`, whose lookups ignore case on Windows; `FromEnv` and `Identity`; the keyboard, link and notification views; `TmuxQuery`; `Report` and `Findings` |
| `termcap/termcaptest` | `Terminal`, a scripted fake terminal, with `SetRunTimeout`; `Profile` and seven profiles; `Run` |
| `termsvc` | `Notifier` (OSC 99, 777, 9 or the bell; focus policies; `WithNotifyFilter`; a `Backend` or `BackendFunc`; `NotifyResultMsg`; `Sequence`, its bytes for a CLI; `NotifyContext` and `WithBackendTimeout`); `Copy`, `CopyContext`, `CopySequence` and `CopiedMsg`, with a `Clipboard` or `ClipboardFunc`; `Link`, `LinkDisplay`, `LinkPolicy`; prompt marks; `Wrap`; `SanitizeTitle`; the activity beacon; `Pointer`; `ProgressSupported` |
| `internal/termevent` | `Decode`: the ultraviolet events tea passes through untranslated, as plain values |
| `internal/termevent/termeventtest` | those events built for tests outside `internal/termevent` |
| `tuitest` | `Golden` across {colour, no colour} × {UTF-8, ASCII} × widths; `Case.Profile` and `Case.Glyphs`; `Fits`; `Text` for a single file; `Annotate` |

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
  its `go.sum` or module graph
  ([0010-REPORT-nested-modules-and-adapter-sources.md](reports/0010-REPORT-nested-modules-and-adapter-sources.md)
  §2).
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
  go-modules.py             lists the modules; --check compares go.work
                            with the tracked go.mod files
  go-modules_test.py        its offline test
  go-fuzz.py                fuzzes each fuzz target of a package in turn
  go-fuzz_test.py           its offline test
  go-apicheck.sh            the API diff gate: incompatible changes since
                            each module's previous tag
  go-apicheck_test.sh       its test, on throwaway repositories
  apicheck.allow            the incompatible changes a PLAN allows
  go-examples.sh            builds and runs the framework examples, and
                            checks the guides' excerpts of them
  go-examples_test.sh       its offline test
  go-precheck_test.sh       the pre-add check's gofmt step and its files,
                            on throwaway repositories
glyph/ theme/ layout/ workspace/ tuitest/ termcap/ termsvc/
when/ command/              the packages; goldens under testdata/golden/
                            where a package renders
termcap/termcaptest/        fake terminals for tests
launch/                     start the TUI from a program's own CLI
launch/launchtest/          fake streams for a program's tests
internal/enum/              enum names and text forms
internal/teamsg/            the one-message command
internal/sanitize/          the one sanitizer for untrusted display text
internal/limits/            the window clamp
internal/cells/             the reused frame buffer, an ultraviolet importer
internal/termevent/         pass-through events, the other ultraviolet importer
internal/conformance/       the conformance scan: terminal ownership,
                            environment, exit and spawn, type names,
                            stability lines, opaque options, enum text
                            (tests only)
tuitest/internal/clash/     proves tuitest's -update flag does not clash (tests only)
testdata/frameworks/        the framework examples: programs on flag, Cobra,
                            Kong and urfave/cli, built in a module of their
                            own (go.mod.tmpl), and their cases
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  glossary.md               one name, one meaning: the shared and reserved
                            type names
  decisions/                MADR and PLAN records
  reports/                  REPORT records
  guides/                   how-to guides: workspaces, terminal
                            capabilities, commands, releasing
```

## Dependencies

- **Required:** `charm.land/bubbletea/v2` v2.0.10, `charm.land/lipgloss/v2`
  v2.0.6, `charm.land/bubbles/v2` v2.2.1,
  `github.com/charmbracelet/colorprofile` v0.4.3,
  `github.com/charmbracelet/x/ansi` v0.11.8,
  `github.com/charmbracelet/x/term` v0.2.2, the version Bubble Tea selects,
  and `github.com/charmbracelet/ultraviolet` at
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
- **Kept to one package by `depguard`:** `github.com/charmbracelet/x/term`
  to `launch`, so only `launch` asks the terminal for its state
  ([0013-MADR](decisions/0013-MADR-cli-integration-helpers.md) §11).

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`,
  `modernize`, `tidy`, `tidy-check`, `vuln`, `fuzz`, `apicheck`,
  `examples`, `pre-add-check`, `release-check`, `help`.
- **Per module.** `test`, `vet`, `lint`, `modernize`, `tidy-check` and
  `vuln` run once in each module's directory with `GOWORK=off`, taking the
  list from `scripts/go-modules.py`; a failure to list the modules fails
  the target.
- **`make lint`** runs `make modernize`, then `golangci-lint run -c
  <root>/.golangci.yml ./...` three times per module: `GOOS=linux`,
  `darwin` and `windows`, each with `CGO_ENABLED=0`.
- **`make modernize`** runs `go fix -diff ./...` for the same three
  targets, with `CGO_ENABLED=0`, and fails on any suggestion: `go fix
  -diff` exits 1 when it prints a diff.
- **`make release-check`** runs the pre-add check over every module and
  file, before a tag.
- **`internal/conformance`** is the conformance scan, run by `go test`.
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
  - It also refuses a read of the process's environment, `os.Exit`, and
    starting a process (`os.StartProcess`, `os/exec`, `syscall`'s `Exec`,
    `ForkExec` and `StartProcess`, `tea.Exec` and `tea.ExecProcess`).
    `allowed` lists the one exception, by file and rule: `tuitest`'s
    `TUITEST_UPDATE` switch.
  - The packages it must read come from `go list`, so a package it skips
    fails the test.
  - `TestNoTypeNameMeansTwoThings` fails when two public packages export
    a type of the same name, unless `sharedNames` and
    [glossary.md](glossary.md) allow it.
  - `TestEveryPackageStatesStability` fails when a package's
    documentation does not end with its stability line (AGENTS.md, "API
    conventions").
  - It reads each package's files for the host's `GOOS`; CI's three
    operating systems cover the rest.
- **`make fuzz`** fuzzes, in every module, each fuzz target of every
  package whose tests declare one, for `FUZZTIME` (default 20s):
  `scripts/go-fuzz.py -a` finds the packages, so a new target needs no
  list.
- **`scripts/go-precheck.sh`** runs, for each module that owns a given
  file (every module when none is given), in the module's directory:
  `gofmt` on its files; with `GOWORK=off`, the same three golangci-lint
  runs, `go vet` and `go test` on their packages, `go mod tidy -diff` and
  `govulncheck ./...`; `go test` again in workspace mode; and, for a
  nested module, no `replace` and a release version of the root. A Go
  file under `testdata` is formatted, but not vetted or tested. gofmt's
  own failure, on a file it cannot read or parse, fails the check, and
  with no file list the files are the tracked Go files the work tree has
  ([0015-MADR](decisions/0015-MADR-precheck-gofmt-errors.md)). It ends
  with `scripts/go-modules.py --check`, then:
  - with no file list, the API diff gate;
  - with no file list, or one naming a file under `testdata/frameworks`,
    the examples gate.

  `make pre-add-check` runs it, and so does the machine-wide agent gate
  before an agent `git commit` that stages Go files.
- **`make apicheck`** runs `scripts/go-apicheck.sh`. For each module it
  takes the newest release tag merged into `HEAD` that does not contain
  `HEAD`, so a tagged commit compares with the tag before it. It exports
  that tag's API and the tree's with apidiff, installed at the
  Makefile's `APIDIFF_VERSION` into a temporary directory and never added
  to `go.mod`, and judges apidiff's output against
  `scripts/apicheck.allow`. An unlisted incompatible change fails, and so
  does a listed one apidiff no longer reports. A module with no tag is
  skipped.
- **`make examples`** runs `scripts/go-examples.sh`:
  - it copies `testdata/frameworks` into a temporary module, whose
    `go.mod` comes from `go.mod.tmpl` with a `replace` of the root;
  - it builds and vets the programs, and runs `cases.txt`;
  - it compares each guide block marked `<!-- from: … -->` with the
    program region it names;
  - `--check-guide` does only the comparison, and `--update` rewrites
    `go.mod.tmpl` and `go.sum` from `go mod tidy`.
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
  - **`modules`** (Linux): the test of `go-modules.py`, then
    `go-modules.py --check`, then the module list as the matrix's input.
  - **`test (<module>, <os>)`** for each module on `ubuntu-24.04`,
    `macos-15` and `windows-2025`, in the module's directory with
    `GOWORK=off`:
    - **every OS:** `go test`, and `go test` again in workspace mode;
    - **Linux and macOS:** `go test -race`;
    - **Linux:** `go test -shuffle=on -count=2` and `LC_ALL=C go test`.
  - **`gates`** (Linux):
    - the fuzz script's test, then `make fuzz`, uploading every
      package's corpus (`**/testdata/fuzz/`) as an artifact on failure;
    - `go vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`, per
      module;
    - `make vet`, `gofmt`, `make tidy-check`, and `make lint` (with
      `make modernize`) with golangci-lint v2.14.0;
    - `make vuln` with govulncheck v1.8.0;
    - `apicheck`: the API diff gate's test, then `make apicheck`. The
      job's checkout fetches the whole history and its tags;
    - `examples`: the examples gate's test, then `make examples`;
    - `precheck test`: the pre-add check's own test;
    - `shellcheck` v0.11.0 (pinned by SHA-256 and first on `PATH`),
      `markdownlint-cli2` 0.23.2 and `actionlint` v1.7.12.
  - One run per ref (`concurrency`, cancel in progress). Actions are pinned
    to commit SHAs.

## What is not here

- **Standard panes** (log tail, metrics view, scrolling text and
  Markdown), **overlay widgets** (dialog, picker, palette, toast), a
  help-footer pane beyond `workspace`'s `help.KeyMap`, and **`updatetea`.**
  Each is its own record.
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
- **A CLI front end,** by design
  ([0012-MADR](decisions/0012-MADR-bring-your-own-cli.md)). `launch` starts
  the TUI from the program's own command line instead.
- **A release workflow, Dependabot, and any tag** other than the owner's.
