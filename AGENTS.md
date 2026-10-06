# AGENTS.md

Instructions for AI coding agents working in this repository. All agents read
this file. A repository-local `CLAUDE.md` / `.claude/rules/` / `.grok/rules/` /
`.opencode/rules.md` wins only where it is more specific than this file.

`go-tui-lib` is a shared Go library of terminal-UI code on the Charm v2 stack
(`github.com/maccavelli/go-tui-lib`). It is a library only: no packaged
binary. Each capability lives in its own top-level directory, with the
package named after the directory; there is no root package. Unexported
helpers shared between packages live under `internal/`. Requires Go 1.27.1.

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
`internal/cells` may import it, at the version lipgloss's requirement
selects, and depguard refuses it everywhere else. Any other module needs
its own MADR.

Never import the Charm v1 paths `github.com/charmbracelet/bubbletea`,
`github.com/charmbracelet/lipgloss` or `github.com/charmbracelet/bubbles`;
use the `charm.land/…/v2` path. Never import `github.com/maccavelli/mcplib`,
the MCP go-sdk (`github.com/modelcontextprotocol/go-sdk`) or
`github.com/maccavelli/go-llmprovider-sdk`. Never import fang
(`charm.land/fang/v2` or `github.com/charmbracelet/fang`), in any module.
`depguard` in `.golangci.yml` refuses each of these, so `make lint`, the
pre-add check and CI fail on them.

Each nested module's list is
`docs/decisions/0010-MADR-nested-adapter-modules.md` §5, and depguard keeps
each dependency in its one module: `github.com/spf13/cobra` and
`github.com/spf13/pflag` in `command/cobracmd` only,
`github.com/alecthomas/kong` in `command/kongcmd` only, and
`charm.land/glamour/v2` in `stream/glamourmd` only.

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
| `github.com/maccavelli/go-tui-lib` | `.` | `vX.Y.Z` |
| `github.com/maccavelli/go-tui-lib/command/cobracmd` | `command/cobracmd` | `command/cobracmd/vX.Y.Z` |
| `github.com/maccavelli/go-tui-lib/command/kongcmd` (planned) | `command/kongcmd` | `command/kongcmd/vX.Y.Z` |
| `github.com/maccavelli/go-tui-lib/stream/glamourmd` (planned) | `stream/glamourmd` | `stream/glamourmd/vX.Y.Z` |

- **`go.work` lists every module** and is committed. It is for development
  only: it builds a root change and an adapter change together before
  either is tagged. A module is added to it with `go work use ./<dir>` in
  the commit that adds the module. `scripts/go-modules.sh --check` fails
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
   truncation use `x/ansi`, and nothing assumes a terminal size.
6. **Rendering is tested across the matrix** {colour, no colour} × {UTF-8,
   ASCII} × at least two widths, with golden files under the package's
   `testdata/`. CI also runs the tests under `LC_ALL=C`.

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
  number and the same slug.
- **Next number** is the highest `NNNN` among all four kinds anywhere under
  `docs/`, plus one. Never reuse a number, never renumber an existing record,
  never leave a gap deliberately.
- Cite records by full filename, never by number alone. Cite another
  repository's record by repository and filename; a relative link cannot reach
  it.
- `docs/README.md` indexes every record. Update it in the same change.

## Pre-add checks

Before staging Go files, run:

```bash
make pre-add-check                 # every tracked Go file
make pre-add-check FILES="a.go b.go"
```

It runs `scripts/go-precheck.sh` once for each module that owns a file
given, or for every module when none is, in the module's directory:

- `gofmt` on the files;
- with `GOWORK=off`: `golangci-lint run -c <root>/.golangci.yml ./...` once
  each for `GOOS=linux`, `darwin` and `windows` with `CGO_ENABLED=0`, the
  same runs as `make lint` and CI; `go vet` and `go test` on the packages
  the files belong to; `go mod tidy -diff`; and `govulncheck ./...`
  (`GO_PRECHECK_SKIP_VULN=1` skips it offline);
- `go test` once more in workspace mode;
- for a nested module, no `replace` and a release version of the root.

It ends with `scripts/go-modules.sh --check`. `make release-check` runs it
over every module and file, before a tag. `golint` is not used: its checks
are `revive`'s `exported`, `package-comments` and `var-naming` rules in
`.golangci.yml`. A file that fails is not committed.

`internal/conformance` checks rules 1 and 2 of the TUI conventions on
every package of every module. It type-checks each package and resolves
every use, so a renamed import, a dot import or a method value hides
nothing. Outside tests it fails on `os.Stdout` and `os.Stderr`,
`fmt.Print`, `fmt.Printf` and `fmt.Println`, `log`'s standard logger,
`log/slog`'s default logger, the `print` and `println` builtins,
`signal.Notify`, and a write to `tea.View`'s `AltScreen`.

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

## Commits

`git commit --no-edit`. Never pass `-m` / `--message` / `-F`. The global
`prepare-commit-msg` hook writes the message from the staged diff. This
repository sets a local `user.name` / `user.email`; do not override it.
