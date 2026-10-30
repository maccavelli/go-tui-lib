# AGENTS.md

Instructions for AI coding agents working in this repository. All agents read
this file. A repository-local `CLAUDE.md` / `.claude/rules/` / `.grok/rules/` /
`.opencode/rules.md` wins only where it is more specific than this file.

`go-tui-lib` is a shared Go library of terminal-UI code on the Charm v2 stack
(`github.com/maccavelli/go-tui-lib/v2`; `v1.13.0` was the last release at
`github.com/maccavelli/go-tui-lib`,
`docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md` A4).
It is a library only: no packaged
binary. Each capability lives in its own top-level directory, with the
package named after the directory; there is no root package. Unexported
helpers shared between packages live under `internal/`. Requires Go 1.27.2.

## Dependencies

No module may be required without a MADR in this repository that names it.
The root module's list is
`docs/decisions/0001-MADR-scaffold-charm-tui-library.md` §3, with its
amendment A3, which names the stack: `charm.land/bubbletea/v2`,
`charm.land/lipgloss/v2`, `charm.land/bubbles/v2`,
`github.com/charmbracelet/colorprofile`, `github.com/charmbracelet/x/ansi` and
`github.com/maccavelli/go-selfupdate-lib` (formerly `go-core-lib`).
`docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md` §1 adds
`github.com/charmbracelet/ultraviolet`, which has no tagged release: only
`internal/cells` and `internal/termevent` (with
`internal/termevent/termeventtest` beneath it) may import it, at the
version lipgloss's requirement selects, and depguard refuses it everywhere
else (`docs/decisions/0005-MADR-terminal-capabilities-and-services.md`
§1). `github.com/charmbracelet/x/term` is a direct requirement at the
version Bubble Tea selects, and only `launch` may import it: the depguard
rule `xterm` refuses it everywhere else
(`docs/decisions/0013-MADR-cli-integration-helpers.md` §11). `termkey`
imports the standard library and `internal/enum`, and `safetext` the
standard library, `glyph` and `internal/enum`, and nothing else: the
depguard rules `termkeystd` and `safetextstd` refuse any other import,
and `internal/conformance`'s `TestNarrowClosures` checks their closures
(`docs/decisions/0021-MADR-termkey-closure-gate.md` N1 to N4, A1). Any
other module needs its own MADR.

Never import the Charm v1 paths `github.com/charmbracelet/bubbletea`,
`github.com/charmbracelet/lipgloss` or `github.com/charmbracelet/bubbles`;
use the `charm.land/…/v2` path. Never import `github.com/maccavelli/mcplib`,
the MCP go-sdk (`github.com/modelcontextprotocol/go-sdk`) or
`github.com/maccavelli/go-llmprovider-sdk`. Never import fang
(`charm.land/fang/v2` or `github.com/charmbracelet/fang`), in any module.
`depguard` in `.golangci.yml` refuses each of these, so `make lint`, the
pre-add check and CI fail on them.

Never import a CLI framework (`github.com/spf13/cobra`,
`github.com/spf13/pflag` or `github.com/alecthomas/kong`), in any module:
go-tui-lib is a TUI layer, and a program brings its own CLI
(`docs/decisions/0012-MADR-bring-your-own-cli.md`). depguard refuses them
everywhere.

Each nested module's list is
`docs/decisions/0010-MADR-nested-adapter-modules.md` §5, and depguard keeps
each dependency in its one module: `charm.land/glamour/v2` in
`stream/glamourmd` only.

The framework examples under `testdata/frameworks` build in a temporary
module outside the repository's modules (`make examples`). That module may
require the tier-1 frameworks
`docs/decisions/0014-MADR-native-integration-api.md` names (Cobra, Kong and
urfave/cli), and `golang.org/x/term` for the line-mode example's raw mode
(`docs/decisions/0023-MADR-line-mode-components.md` A11); the library's
modules never do.

`go.mod` and `go.sum` change with the code that needs them: a requirement is
added in the commit that adds its first import, and removed in the commit that
removes its last. `go mod tidy -diff` is clean at every commit, in every
module.

## Modules

The repository holds more than one Go module
(`docs/decisions/0010-MADR-nested-adapter-modules.md`). Each has its own
`go.mod`, its own requirements and its own tags:

| Module | Directory | Tags |
| :--- | :--- | :--- |
| `github.com/maccavelli/go-tui-lib/v2` | `.` | `vX.Y.Z`, from `v2.0.0` |
| `github.com/maccavelli/go-tui-lib/stream/glamourmd` (planned) | `stream/glamourmd` | `stream/glamourmd/vX.Y.Z` |

- **`go.work` lists every module** and is committed. It is for development
  only: it builds a root change and an adapter change together before
  either is tagged. A module is added to it with `go work use ./<dir>` in
  the commit that adds the module. `scripts/go-modules.py --check` fails
  when `go.work` and the tracked `go.mod` files disagree. `go.work.sum` is
  ignored, not committed: workspace-mode commands such as `go doc` on a
  dependency write it, and no gate reads it
  (`docs/decisions/0010-MADR-nested-adapter-modules.md` A1).
- **Every gate runs per module with `GOWORK=off`,** in the module's
  directory, so that each module builds from its own `go.mod` and published
  versions only, as a consumer does. The tests run once more in workspace
  mode. `./...` never crosses into a nested module.
- **An adapter requires a published root version** (`vX.Y.Z`, never a
  pseudo-version) and has no `replace`.
- **A change that spans the root and an adapter is released in two
  steps:** the root is tagged first, then the adapter's requirement moves
  to that tag and the adapter is tagged. Tags are never moved or deleted.
  [docs/guides/releasing.md](docs/guides/releasing.md) has the procedure.

## TUI conventions

Every package follows these rules
(`docs/decisions/0001-MADR-scaffold-charm-tui-library.md` §6). A package
whose record grants an exception says so in that record.

1. **Output goes where the caller says.** A package never writes to
   `os.Stdout` or `os.Stderr` itself. It renders to a string, an
   `io.Writer` the caller passes, or a Bubble Tea view.
2. **The caller owns the screen and the signals.** A component never sets
   the alternate screen and never installs a signal handler. `ctrl+c`
   reaches the program as a key or a cancel.
3. **Every glyph comes from a glyph table** with a Unicode form and an ASCII
   twin, including the glyphs Charm components draw by default.
4. **Colour degrades, and never carries meaning alone.** Output stays
   correct under `NO_COLOR`, an ASCII colour profile and an unknown
   background.
5. **Width is the caller's, and is measured in cells.** Wrapping and
   truncation use `x/ansi`, and nothing assumes a terminal size. One rule
   says which width (0014-PLAN-canonicalization Step 7):
   - the workspace measures with its width method, `ansi.WcWidth` until
     the terminal reports grapheme clustering (mode 2027) and
     `ansi.GraphemeWidth` after, unless `WithWidthMethod` fixes it; it
     cuts with `internal/sanitize`'s `Truncate`, which never returns text
     wider than asked;
   - a test measures with `tuitest.Fits`, under its case's method;
   - `termsvc` and `termcap`'s report cut by grapheme cluster, with
     `ansi.Truncate`.
6. **Rendering is tested across the matrix** {colour, no colour} × {UTF-8,
   ASCII} × at least two widths, with golden files under the package's
   `testdata/`. CI also runs the tests under `LC_ALL=C`.

## API conventions

Every exported name follows these rules
(`docs/decisions/0014-MADR-native-integration-api.md` W0). A record that
needs an exception says so.

1. **Deprecation.** Within a major version nothing is removed or retyped.
   A renamed function, type or constant keeps its old name, as a wrapper,
   a type alias or a constant marked `// Deprecated: use X`, until the
   next major version. A renamed struct field keeps both fields until
   then, and the code reads the new field, falling back to the old. A
   removal needs a major release and a record. Release notes list every
   deprecation and every removal. Under `v0`, up to `v0.12.0`, a
   deprecated name was removed in the next minor release
   (`docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md`
   A4.2).
2. **The API diff gate.** `make apicheck` compares each module's exported
   API with its previous tag, and fails on an incompatible change that
   `scripts/apicheck.allow` does not list. The PLAN that makes such a
   change lists it there, in a block naming the PLAN. A block is allowed
   only in a PLAN whose release changes the major version, so a minor
   release has 0 incompatible changes. Across a major boundary the base's
   own import paths are moved to the tree's module path before the two
   are compared, so only real changes are reported. The release's
   close-out removes the block: once the new tag is the base, apidiff no
   longer reports the change, and a listed line it does not report fails
   as stale.
3. **Names.** One name means one thing. [docs/glossary.md](docs/glossary.md)
   lists the names several packages share on purpose, and the names
   accepted records reserve. A record checks each new exported type name
   against it. `internal/conformance` fails when two public packages
   export a type of the same name, unless the glossary and its
   `sharedNames` list both allow it.
4. **Options.** Option functions are named `With…`, `Without…` or `On…`:
   `On…` names a callback, `With…` sets a value or a provider, and
   `Without…` turns off a default. Option types are opaque: an interface
   with an unexported method, which only the package's own functions
   satisfy (0014-PLAN-canonicalization Step 5). An option type that is a
   function over an unexported struct is opaque already.
5. **Enums.** An exported enum has `String`, `MarshalText` and
   `UnmarshalText`, with stable lowercase tokens, from one table in
   `internal/enum`: `Names`, or `Bits` for a bit set
   (0014-PLAN-canonicalization Step 6). A value with no token prints
   `Type(N)`. `internal/conformance` fails on an enum whose tokens do not
   round-trip.
6. **Errors.** Sentinels are `Err…` values. A typed error has `Unwrap` when
   it wraps another. An error that ends a CLI has `ExitCode() int`.
7. **Hooks.** A hook is an interface, with a `…Func` adapter.
8. **Constructors.** `NewX` builds an `X`. `MustNewX`, which panics, exists
   only where a failure is a programming error.
9. **The environment.** Library code reads the environment only through a
   `termcap.Env` the program gives it. It never reads the process's
   environment, never exits and never starts a process. `internal/conformance`
   enforces this. Its one allowed use is `tuitest`'s `TUITEST_UPDATE`
   switch, since `tuitest` runs only under `go test`.
10. **Stability lines.** Each package's documentation ends with its
    stability line, as its last paragraph, in the file that holds the
    package comment:
    - a public package:

      ```go
      // Stability: stable. Exported names change only through the deprecation
      // policy in AGENTS.md, "API conventions".
      ```

    - a package under an `internal` directory:
      `// Stability: internal.`

    `internal/conformance` fails on a package without it.
11. **The toolchain floor.** The `go` line in each `go.mod`, today
    1.27.2 (`docs/decisions/0016-MADR-go-1-27-2-for-go-2026-6604.md`),
    moves only by a record. It moves to the newest patch of a
    release Go still supports, and the record names the features that need
    it.
12. **Defaults are the program's to refuse**
    (`docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md` D1).
    Everything the library does on a program's behalf has a documented
    default and a control a program can use without forking:
    - a key it binds, a command it registers or runs, a query or mode it
      sends to the terminal, a policy it applies, a goroutine or timer it
      starts for the program, and text it shows a user: a `Without…`
      option, a nil or zero value, or a hook that replaces it;
    - a value a program may legitimately need otherwise, such as a
      timeout, a size, a cap on the program's own data: a `With…` option,
      whose zero or negative value keeps the default;
    - a behaviour added after `v2.1.0` that writes to the terminal, acts on
      the user's behalf, or runs a program's command: off until a `With…`
      or `On…` option turns it on, unless its record says why it is on;
    - every convenience built on a lower layer: that layer exported, so a
      program does the convenience its own way.

    The floors in `docs/guides/defaults.md` are the exceptions, each with
    its reason: the process rules, the cleaning of untrusted text, the
    bounds on untrusted input, grammars and protocols, a command's
    declared danger and a handler's recovered panic. A record that adds a
    default lists it there, with its control; `internal/conformance`'s
    `TestDefaultsInventory` resolves every control the guide names.

## MADR and PLAN before mutating work

**Whenever the user asks for an MADR and a plan, load the
`madr-and-plan-writing` skill first** and follow it for authoring, naming and
review. This applies both to writing a fresh pair and to amending an existing
one.

The name is exact — it is the `name:` field of the skill. A mistyped call
returns `Unknown skill`, and an agent that proceeds without the skill writes
something shaped like a MADR while missing the required headings and the
directory rule below. Verify against the filesystem rather than memory:

```bash
ls -d ~/.claude/skills/*madr* && grep '^name:' ~/.claude/skills/*madr*/SKILL.md
```

**Read-only investigation is allowed with no pair.** Reading, searching,
`git log` / `git show` / `git diff`, and existing tests or diagnostics that
do not write the tree do not need a MADR.

**Mutating work is not.** Before the first write, name the
`docs/decisions/NNNN-MADR-*` / `docs/decisions/NNNN-PLAN-*` pair being
executed, or stop and write one.

Mutating means: creating, editing, or deleting files; staging or committing
(except the bootstrap exception below); dependency or lockfile changes;
CI / config / hook changes; builds or installers that write the tree,
`$HOME`, or a live service; generating committed artifacts.

Order:

1. Investigate (read-only).
2. Write or amend the MADR (`status: proposed` unless the owner already
   decided). Present it. Do not implement.
3. Write or amend the PLAN. Present it.
4. Mutate **only after** the owner explicitly approves execution
   (`proceed`, `execute the plan`, `do phase N`). Stay inside that PLAN.
5. Anything discovered mid-execution that is out of scope waits: amend the
   pair, re-approve, then continue. Completing a phase is not permission to
   invent the next unwritten one.

Follow-up vs greenfield:

- **Same topic** (debug, leftover phase, bug found in that plan's live
  run): amend that number. Add a PLAN phase or an amendment in the MADR. Do
  not silently rewrite historical rationale.
- **Greenfield**: next unused `NNNN`, new MADR, new PLAN, same slug. No
  mutation until that PLAN is approved.

Bootstrap exception: authoring `docs/decisions/NNNN-MADR-*`,
`docs/decisions/NNNN-PLAN-*`, this file, and the per-agent pointers
(`.claude/rules/`, `.grok/rules/`, `.opencode/rules.md`) does not require a
*prior* pair. Putting source, tests, CI, or product config in that same commit
is a violation.

`git push` and tags still need an explicit ask in the same turn.

## Records

```text
docs/decisions/NNNN-MADR-short-slug.md
docs/decisions/NNNN-PLAN-short-slug.md
docs/reports/NNNN-REPORT-short-slug.md
docs/reports/NNNN-GATES-short-slug.md
```

- `NNNN` is a zero-padded 4-digit number, one sequence across
  `docs/decisions/` and `docs/reports/`. A MADR and its PLAN share the same
  number. A lone PLAN copies the MADR's slug; a MADR implemented as several
  units of work carries several PLANs, each with a slug of its own
  (`docs/decisions/0002-PLAN-harden-workspace-v0-1-1.md`). A REPORT about
  an existing record takes that record's number
  (`docs/reports/0010-REPORT-nested-modules-and-adapter-sources.md`); a
  REPORT about nothing earlier takes the next.
- **Next number** is the highest `NNNN` among all four kinds anywhere under
  `docs/`, plus one: `python3 scripts/check_records.py --next` prints it.
  Never give a new decision a number already used, never renumber an
  existing record, never leave a gap deliberately.
- Cite records by full filename, never by number alone. Cite another
  repository's record by repository and filename; a relative link cannot reach
  it.
- `docs/README.md` indexes every record. Update it in the same change.
- `make check-records` runs `scripts/check_records.py`. It fails on a
  misnamed or misplaced record, a second MADR of a number, a PLAN with no
  MADR, a GATES with no PLAN, a record with no row in the index's
  "Records" table or a row that names another, and a broken relative link
  or anchor in any tracked Markdown file outside `testdata`
  (`docs/decisions/0022-MADR-align-docs-ci-and-disclosure-gates.md`).

## Pre-add checks

Before staging Go files, run:

```bash
make pre-add-check                 # every tracked Go file
make pre-add-check FILES="a.go b.go"
```

It runs `scripts/go-precheck.py` once for each module that owns a file
given, or for every module when none is, in the module's directory:

- `gofmt` on the files;
- with `GOWORK=off`: `golangci-lint run -c <root>/.golangci.yml ./...` once
  each for `GOOS=linux`, `darwin` and `windows` with `CGO_ENABLED=0`, the
  same runs as `make lint` and CI; `go vet` and `go test` on the packages
  the files belong to; `go mod tidy -diff`; and `govulncheck ./...`
  (`GO_PRECHECK_SKIP_VULN=1` skips it offline);
- `go test` once more in workspace mode;
- for a nested module, no `replace` and a release version of the root.

A Go file under a `testdata` directory is formatted, but not given to
`go vet` or `go test`. The go command ignores `testdata`, and the framework
examples there import modules the library does not require.

gofmt's own failure fails the check: a file it cannot read or parse makes
it exit non-zero with nothing on its output. With no file list, the files
are the tracked Go files the work tree has, so a tracked file deleted but
not yet staged is not checked
(`docs/decisions/0015-MADR-precheck-gofmt-errors.md`).

It ends with `scripts/go-modules.py --check`. Then:

- **With no file list** (`make release-check`, or `make pre-add-check`
  without `FILES`), it runs the API diff gate, `scripts/go-apicheck.py`.
  `GO_PRECHECK_SKIP_APICHECK=1` skips it offline.
- **With no file list, or one that names a file under
  `testdata/frameworks` or a Go file that starts a `// guide:` region,**
  it runs the examples gate, `scripts/go-examples.py`. That gate builds
  the framework examples in a temporary module, runs their cases, and
  compares each guide's excerpts with the regions they name, in those
  programs or in a package's examples. `GO_PRECHECK_SKIP_EXAMPLES=1`
  skips it offline.

`make release-check` runs it over every module and file, before a tag.
`golint` is not used: its checks are `revive`'s `exported`,
`package-comments` and `var-naming` rules in `.golangci.yml`. A file that
fails is not committed.

`internal/conformance` checks rules 1 and 2 of the TUI conventions on
every package of every module. It type-checks each package and resolves
every use, so a renamed import, a dot import or a method value hides
nothing. Outside tests it fails on:

- `os.Stdout` and `os.Stderr`;
- `fmt.Print`, `fmt.Printf` and `fmt.Println`;
- `log`'s standard logger and `log/slog`'s default logger;
- the `print` and `println` builtins;
- `signal.Notify`, and a write to `tea.View`'s `AltScreen`;
- a read of the process's environment (`os.Getenv`, `os.LookupEnv`,
  `os.Environ`, `os.ExpandEnv`, `syscall.Getenv`), except `tuitest`'s;
- `os.Exit`;
- starting a process: `os.StartProcess`, any use of `os/exec`,
  `syscall.Exec`, `ForkExec` and `StartProcess`, and `tea.Exec` and
  `tea.ExecProcess`.

It also fails when:

- two public packages export a type of the same name that
  [docs/glossary.md](docs/glossary.md) does not allow;
- a package's documentation does not end with its stability line ("API
  conventions").

`make apicheck` and `make examples` run the API diff gate and the examples
gate alone.

Before staging a document, run the documents' checks, the line both
workflows run ("CI routing"):

```bash
make markdownlint check-records docs-check disclosure-check
```

Before staging a workflow or a file under `scripts/`, also run
`python3 scripts/check_ci_filters.py` and the tests of each script changed.

Golden files are rewritten with `go test ./<pkg>/ -tuitest.update`, or
across packages with `TUITEST_UPDATE=1 go test ./...`, and read before they
are committed. A test binary that defines its own boolean `-update` flag
may use it instead.

The machine-wide agent gate runs the same script before every agent
`git commit` that stages Go files, and denies the commit when it fails. There
is no `git add` hook on every host; do not rely on one.

`make lint` and `make vuln` must be clean before a release-shaped change.
`make test`, `vet`, `lint`, `modernize`, `tidy-check` and `vuln` each run
once per module with `GOWORK=off`. `make lint` runs `make modernize` first, which fails on any `go fix -diff`
suggestion for `GOOS=linux`, `darwin` and `windows`; `go fix ./...`
applies them.

Cross-target commands set `CGO_ENABLED=0` explicitly: a host `go env` may
set `CGO_ENABLED=1`, and cgo cannot cross-compile to another OS here.

## Scripts

The scripts under `scripts/` are Python 3.12 or later, the standard
library only (`docs/decisions/0017-MADR-python-repository-scripts.md`):
no `pip`, no virtual environment. A script runs commands from argument
lists, never with `shell=True`. A test harness reports every failure as a
FAIL line naming its case, and goes on to the next; it never stops
silently. `make` runs them through `PYTHON` (default `python3`).

Every script is Python. The one shell file left,
`scripts/go-precheck.sh`, is a shim that runs `go-precheck.py` with the
same arguments and exit status, because the machine-wide commit gate runs
it by that name; it holds no logic, and `shellcheck` checks it.

## Tools

`tools/verify` is the verification tooling a person or an agent runs to
check a PLAN step (`docs/decisions/0018-MADR-python-verification-tools.md`),
as `python3 -m tools.verify <command>` from the repository root: `gates`
(every gate of a step, as data), `mutate` (mutation proofs on scratch
copies), `remote` (the same gates on another host), `scan` (the
identifier scan of a diff) and `names` (a record's planned type names).
Nothing in `make`, CI or the commit gate runs it; CI only tests it.

- It follows "Scripts": Python 3.12 or later, the standard library only,
  argument lists and never `shell=True`.
- **No shell scripts, for tooling or for a step's checks.** Shell runs one
  command at a time, at most with one pipe; searching and editing use the
  agent's native tools, not `grep`, `sed`, `awk` or `cut`. Logic that is
  run a second time belongs in `tools/verify`, not a scratch directory.
- **Its config is local:** the hosts and the identifier patterns are read
  from `$XDG_CONFIG_HOME/go-tui-lib/verify.json` (default
  `~/.config/go-tui-lib/verify.json`), or `--config`. It is never
  committed, and no file under `tools/` names a host, an account or a real
  path; `tools/verify/verify.example.json` shows the shape with
  placeholders.

## CI routing

`.github/workflows/ci.yml` is the full CI. `.github/workflows/docs.yml`
runs the documents' checks alone, for a push to `main` or a pull request
that changes only documents
(`docs/decisions/0019-MADR-skip-full-ci-for-docs-only-pushes.md`;
`docs/decisions/0022-MADR-align-docs-ci-and-disclosure-gates.md`).

- **The docs-safe paths** are `*.md` at the root, `docs/**/*.md`,
  `.claude/**/*.md`, `.grok/**/*.md` and `.opencode/**/*.md`. CI's `push`
  and `pull_request` ignore exactly these; Docs runs on them, on
  `.markdownlint-cli2.jsonc` and on itself. A change to any other file, or
  a mix, runs CI. A `v*` tag always runs CI: GitHub evaluates no path
  filter for a tag push.
- **Markdown that code reads is not docs-safe** unless its reader runs in
  Docs. `command/testdata/**/*.md`, which `go test` reads, stays in CI.
  `TestDefaultsInventory`, which reads `docs/guides/defaults.md` and the
  records it cites, and the guides' excerpt check run in Docs through
  `make docs-check`.
- **Both workflows run one line,**
  `make markdownlint check-records docs-check disclosure-check`. The
  `Makefile` pins `markdownlint-cli2`; no workflow names it.
- **`scripts/check_ci_filters.py`**, in CI's `gates` job, fails when:
  - CI's two ignore lists differ, or CI gains a `paths` filter;
  - CI loses its `push` to `main`, its `v*` tags or `pull_request`;
  - Docs' paths are not CI's ignored paths plus its own two;
  - an ignored pattern matches no tracked file;
  - CI ignores a file under `testdata`, or a file a reader in `READERS`
    reads while Docs does not run that reader;
  - a Go literal names an ignored document that is neither a reader's
    input nor listed in `NOT_READS`;
  - the shared line or its pins drift between the workflows.

  A new reader of a document is added to `READERS`, with its make target,
  in the change that adds it.

## Identifiers

Nothing committed carries a hostname, account name, org-internal path, or a
real-machine absolute path. Use placeholders (`<user>`, `/home/<user>/...`).

- When recording an identifier scan in a record or a commit, describe what was
  scanned for ("the local account name", "the hostname domain"); never quote
  it.
- Pushes to GitHub pass through a global pre-push disclosure guard. Before
  asking the owner to push, run the guard itself over the outgoing commits:

  ```bash
  echo "refs/heads/main $(git rev-parse HEAD) refs/heads/main $(git rev-parse origin/main)" |
    python3 ~/.global-git-hooks/github-disclosure.py pre-push origin "$(git remote get-url origin)"
  ```

- Never bypass it with `--no-verify`.
- `make disclosure-check` (`scripts/check_disclosure.py`) is the portable
  half, which both workflows run: it fails on a macOS, Linux or Windows
  home path in any tracked file whose account is not a placeholder, and
  prints only the file, line and kind. It holds no private pattern, and
  does not replace the pre-push guard.

## Commits

`git commit --no-edit`. Never pass `-m` / `--message` / `-F`. The global
`prepare-commit-msg` hook writes the message from the staged diff. This
repository sets a local `user.name` / `user.email`; do not override it.
