---
status: accepted
date: 2026-10-02
decision-makers: owner
consulted: 0010-REPORT-nested-modules-and-adapter-sources.md (the Go module reference, toolchain and workspace documentation, the Go Modules wiki, precedent repositories, and the sources of Cobra v1.10.2, fang v2.0.1, Kong v1.16.1 and glamour v2.0.1); 0001-MADR-scaffold-charm-tui-library.md §3 and §6
informed: pi-go
---
# Ship the Cobra, Kong and glamour adapters as nested Go modules, released apart from the root and developed through a committed go.work, with every gate run per module and without the workspace

## Context and Problem Statement

go-tui-lib is one module today. Three planned adapters each bring a
dependency the rest of the library does not need:

* [0006-MADR-command-registry.md](0006-MADR-command-registry.md) plans a
  Cobra front end for the command registry (owner question Q4), and on
  2026-10-02 the owner asked for a Kong front end as well, each "fully
  optimized and full-featured", with "only the one compiled in as needed";
* [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
  plans a glamour Markdown renderer (owner question Q1).

As packages of the root module, each adapter's dependency would be compiled
only by programs that import it. It would still reach every consumer.
[0010-REPORT-nested-modules-and-adapter-sources.md](../reports/0010-REPORT-nested-modules-and-adapter-sources.md)
§2 measured this: a root `go.mod` requiring Kong puts
`github.com/alecthomas/kong v1.16.1/go.mod` in the `go.sum` of a consumer
that imports none of it, and Kong in that consumer's `go list -m all`. The
consumer's dependency scanners then report a module it never builds.

On 2026-10-02 the owner decided:

> Nested modules. Glamour as nested module. Go.work in repo.

Those three choices are the owner's, and this record does not reopen them.
It decides how to make them correct: the module paths and names, the
`go.work` file, the release order, the gates, and the dependencies each
module may take. The Go documentation advises against committing `go.work`
unless the module author "should ensure the individual modules are tested
and released properly" (REPORT §3). Most of this record is that assurance.

## Decision Drivers

* **A consumer gets only what it imports,** in its build, its `go.sum` and
  its module graph (REPORT §2).
* **Every gate tests what a consumer gets.** A workspace hides a missing
  requirement and an unpublished one (REPORT §3, experiments), so every
  release gate runs with `GOWORK=off`.
* **The root module's stack stays 0001-MADR §3's.** The root never requires
  an adapter or its dependencies.
* **Each adapter keeps the library's rules** (0001-MADR §6): output only to
  the caller's writers, no `os.Exit`, no terminal probe the caller did not
  make, width from the caller. REPORT §10 lists where each upstream breaks
  them.
* **Go's own idioms:** directory-prefixed tags, `v0` without a path suffix,
  permanent tags, package names that do not shadow the wrapped library
  (REPORT §1, §5).
* **Gates that cannot drift:** one script runs them locally, in the agent
  gate and in CI, as `scripts/go-precheck.sh` already does for one module.

## Considered Options

* **A. Nested modules that require only published root versions; `go.work`
  for development; every gate per module with `GOWORK=off`.**
* **B. Nested modules with a permanent local `replace` of the root, and
  root and adapter tags in lockstep on one commit,** as opentelemetry-go,
  testcontainers-go and aws-sdk-go-v2 do.
* **C. Nested modules, with CI in workspace mode only,** as grafana does.

Packages in the root module, and a separate repository per adapter, were
ruled out by the owner's decision.

## Decision Outcome

Chosen option: **"A"**, because it is the only one in which a committed
`go.work` changes nothing a consumer sees: each adapter's `go.mod` names
the root version a consumer will resolve, and every gate checks exactly
that. It follows charmbracelet/x, the Charm ecosystem's own multi-module
repository, which requires only published versions between its modules
(REPORT §5).

### 1. The modules

| Directory | Module path | Package | Requires (beyond the root) | First tag |
| :--- | :--- | :--- | :--- | :--- |
| `.` | `github.com/maccavelli/go-tui-lib` | (many) | 0001-MADR §3's stack only | `v0.x` as today |
| `command/cobracmd` | `github.com/maccavelli/go-tui-lib/command/cobracmd` | `cobracmd` | `github.com/spf13/cobra` v1.10.2 | `command/cobracmd/v0.1.0` |
| `command/kongcmd` | `github.com/maccavelli/go-tui-lib/command/kongcmd` | `kongcmd` | `github.com/alecthomas/kong` v1.16.1 | `command/kongcmd/v0.1.0` |
| `stream/glamourmd` | `github.com/maccavelli/go-tui-lib/stream/glamourmd` | `glamourmd` | `charm.land/glamour/v2` v2.0.1 | `stream/glamourmd/v0.1.0` |

* **Paths follow the packages they serve.** The front ends sit under
  `command/`, beside `command/cli`; the renderer sits under `stream/`. Each
  directory is new: `v0.1.0` published none of them, so no tagged package
  path moves (REPORT §1).
* **Names do not shadow the wrapped library.** A package named `cobra` or
  `kong` would collide with `spf13/cobra` or `alecthomas/kong` at every
  import site that uses both, as `otelhttp` and `zapgrpc` avoid (REPORT §5).
  0006-MADR's `command/cobra` becomes `command/cobracmd`.
* **Each adapter imports only the root's exported packages,** never
  `internal/`. Go allows the import (REPORT §1), but minimal version
  selection can pair an adapter release with any newer root.
* **The root never imports an adapter,** so no package cycle and no module
  cycle exists.
* **Versions are the newest on 2026-10-02** (REPORT §6, §8, §9). The PLAN
  checks again before each module's first commit, as 0009-PLAN Step 7
  already does for glamour.
* **Every module declares the root's `go` line,** `go 1.27.1`. Each
  adapter's upstream declares an older one (cobra 1.15, kong 1.20, glamour
  1.25.8; REPORT §6, §8, §9), so the root's governs, and one toolchain
  builds every module.
* **Optional subpackages stay in their module.** A subpackage that needs
  more, such as `cobra/doc` for man pages (md2man, YAML), is compiled only
  by its importers, and its requirements reach only that module's
  consumers.
* **fang is not required** (owner question Q1). It breaks rules 1, 2 and
  5 with no option to turn them off (REPORT §7). `command/cobracmd` draws
  Charm-styled help itself, and an optional subpackage writes man pages.

### 2. `go.work`

```text
go 1.27.1

use (
	.
	./command/cobracmd
	./command/kongcmd
	./stream/glamourmd
)
```

* **Committed at the root,** with `go.work.sum` when the `go` command
  writes one. Until a module exists, its `use` line is absent: each module's
  first commit adds its directory with `go work use`.
* **The `go` line equals the root's** `go 1.27.1`. The documentation
  requires it to be at least every module's (REPORT §3), and the root's is
  the highest by this record's rule that every module declares the same
  `go` line.
* **No `toolchain`, `godebug` or `replace` lines.** A `replace` in
  `go.work` would apply in development and nowhere else.
* **The workspace is for development only.** It lets an adapter change and
  a root change be built and tested together before the root is tagged. No
  release gate uses it (§4).

### 3. Requirements and the release order

* **An adapter's `go.mod` requires a published root version,** with no
  `replace`. What it names is what a consumer resolves.
* **A change that spans the root and an adapter is released in two steps:**
  1. the root change lands, and the owner tags `vX.Y.Z`;
  2. the adapter's requirement moves to `vX.Y.Z`
     (`GOWORK=off go get github.com/maccavelli/go-tui-lib@vX.Y.Z`, then
     `GOWORK=off go mod tidy`), the adapter change lands, and the owner tags
     `<dir>/vA.B.C`.

  Until step 1, the adapter's `GOWORK=off` gates fail, by design: the
  adapter does not yet build against any root a consumer can get.
* **Tags carry the directory prefix** (`command/kongcmd/v0.1.0`), are
  independent per module, start at `v0`, and are never moved or deleted
  (REPORT §1). A consumer writes `go get
  github.com/maccavelli/go-tui-lib/command/kongcmd@v0.1.0`.
* **After a tag, a consumer smoke test** runs in a scratch module outside
  the repository: `go get` the tagged module and `go build ./...`, which
  proves that the proxy resolves the prefixed tag.

### 4. Gates, per module, without the workspace

`./...` never crosses into a nested module, with or without `go.work`
(REPORT §4), so every gate loops over the modules. The list comes from
`go list -m -f '{{.Dir}}'` in workspace mode, and is checked against the
`go.mod` files in the tree, so a module missing from `go.work` fails.

| Gate | Where | Mode |
| :--- | :--- | :--- |
| `gofmt` | each module | — |
| `go vet ./...`, `go test -race ./...`, `LC_ALL=C go test ./...`, shuffle | each module | `GOWORK=off`, and again in workspace mode |
| `golangci-lint run -c <root>/.golangci.yml ./...` for linux, darwin and windows | each module | `GOWORK=off` |
| `go mod tidy -diff` | each module | `GOWORK=off` (tidy ignores the workspace anyway) |
| `govulncheck ./...` | each module | `GOWORK=off` |
| no `replace` in any `go.mod`; an adapter's root requirement is a release version, not a pseudo-version | each adapter | — |
| `go.work` lists exactly the tree's modules | root | workspace |
| fuzzing | the modules that have fuzz targets | `GOWORK=off` |

* **The root runs both ways too,** because the workspace's build list is
  the maximum over every module's requirements: an adapter's newer
  lipgloss would be what the root's tests saw in workspace mode (REPORT
  §3).
* **One implementation.** `scripts/go-precheck.sh` learns the module loop,
  and `make pre-add-check`, `make lint`, `make vuln`, the agent gate and CI
  keep calling it. Its file mode checks the modules that own the files
  given.
* **depguard, per module.** The root configuration refuses Cobra, fang,
  Kong and glamour in the root module, beside the Charm v1 paths it already
  refuses. Each adapter module may import its own dependency and no other
  adapter's.
* **The conformance scan covers every module.** Today's scan walks the
  file tree, so it already reaches nested directories. The typed scan of
  [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  Step 7 must type-check each module in its own module context, because
  one module's type checker does not load another's packages.
* **CI** runs a matrix of module × operating system. `actions/setup-go`
  reads `go-version-file: go.work`, and caches on `**/go.sum` (REPORT §4).
  `GOFLAGS=-mod=mod` is never set: workspace mode refuses it (REPORT §3).

### 5. Dependencies

AGENTS.md says no module may be required without a record that names it.
This record names:

* `github.com/spf13/cobra` (with `github.com/spf13/pflag`, and, for its
  `doc` subpackage, `github.com/cpuguy83/go-md2man/v2` and
  `go.yaml.in/yaml/v3`), in `command/cobracmd` only;
* `github.com/alecthomas/kong`, in `command/kongcmd` only;
* `charm.land/glamour/v2`, in `stream/glamourmd` only. This replaces the
  root requirement 0009-MADR Q1 named.

A module's requirement is added in the commit that adds its first import, as
AGENTS.md already says, and `go mod tidy -diff` is clean at every commit in
every module.

### 6. What each adapter must do about its upstream

The APIs are decided in 0006-MADR amendment A1 and 0009-MADR amendment A2.
This record fixes only the rules they must meet, from REPORT §6-§10:

* **Cobra:** always `SetArgs`, `SetOut`, `SetErr`, `SetIn`; `SilenceErrors`
  and `SilenceUsage`; `ExecuteContextC`, with the error returned; never
  `CheckErr`; no process-wide variable set by the adapter
  (`EnableTraverseRunHooks`, `MousetrapHelpText`), and the mousetrap's
  `os.Exit` on Windows documented for the program to turn off. A failing
  shell completion writes to `os.Stderr` inside Cobra; that is documented,
  because no setter reaches it.
* **Kong:** always `Name`, `Writers` and an `Exit` that panics a sentinel
  the adapter recovers into an exit code, because an `Exit` that returns
  lets parsing continue. Help is printed by the adapter's own
  `HelpPrinter` at the caller's width. No `env` tag unless the program asks
  for one. Shell completion scripts for bash, zsh, fish and PowerShell are
  generated in the module from the registry, to the caller's writer, with
  no further module (owner question Q2).
* **glamour:** `WithStyles` from the library's theme and glyph table,
  never the environment or a style path; no chroma style in the global
  registry; one renderer per width and theme, each behind a lock.

### 7. Versioning

* The root's next release is unaffected: the adapters are new modules, and
  the root gains no requirement.
* Each adapter starts at `v0.1.0` under its own prefix, after the root
  release whose API it uses.
* Records name an adapter's root requirement in the adapter's PLAN, so its
  compatibility is written down as well as resolved.

### Consequences

* Good, because a consumer of the root alone never sees Cobra, Kong or
  glamour in its build, `go.sum` or module graph, which was measured
  (REPORT §2).
* Good, because each adapter is released, versioned and upgraded on its own,
  and a breaking change in Cobra or Kong touches one module.
* Good, because every gate runs as a consumer would build, so the
  committed `go.work` cannot hide a missing or unpublished requirement.
* Good, because the Charm ecosystem works the same way: charmbracelet/x's
  modules require published versions and carry prefixed tags.
* Bad, because a change that spans the root and an adapter takes two
  releases, and the adapter's gates fail between them.
* Bad, because the repository goes against the Go documentation's default
  advice twice: several modules, and a committed `go.work`. This record's
  gates are the assurance the documentation asks for in that case.
* Bad, because every gate runs once per module, and CI gains a dimension.
* Bad, because the owner tags more: one tag per module release, in order.
* Neutral, because `go test ./...` at the root no longer tests everything;
  `make test` and the precheck loop over the modules instead.

### Confirmation

* **The graph.** A scratch consumer that `go get`s a tagged root and imports
  one root package has no Cobra, Kong or glamour line in `go.mod`, `go.sum`
  or `go list -m all`, as REPORT §2's `nestlib` did.
* **The gates catch what the workspace hides.** On scratch copies, each of
  these makes the precheck fail, and the failure is recorded:
  * an adapter's `require` of the root deleted;
  * an adapter requiring a root version that is not tagged;
  * a `replace` added to an adapter's `go.mod`;
  * a module's directory missing from `go.work`;
  * the root importing an adapter's dependency (depguard).
* **Lint reaches every module.** An issue planted in each module's scratch
  copy is reported by `make lint`.
* **The release order works once,** on the first adapter: the root tag,
  the requirement bump, the adapter tag and the consumer smoke test are
  recorded in that PLAN's execution record.
* CI is green for every module on Linux, macOS and Windows.

## Pros and Cons of the Options

### A. Published root versions, `go.work` for development, `GOWORK=off` gates

* Good, because the committed `go.work` changes nothing a consumer or a
  gate sees.
* Good, because it is charmbracelet/x's model, and vault and kubernetes
  commit `go.work` while running their gates with `GOWORK=off` (REPORT §5).
* Bad, because a spanning change takes two releases.

### B. A permanent local `replace` and lockstep tags

* Good, because one commit can change and release the root and an adapter
  together.
* Neutral, because consumers ignore a dependency's `replace`, so it is
  safe to publish.
* Bad, because every `GOWORK=off` gate then tests the local root, not the
  published one, which removes the check the committed `go.work` needs.
* Bad, because every adapter is re-tagged at every root release, whether
  it changed or not.

### C. Workspace-mode CI only

* Good, because it is the least CI.
* Bad, because it is exactly what the documentation warns against: CI
  selects and tests versions no consumer gets, and a missing requirement
  passes (REPORT §3).

## Owner questions

*Answered 2026-10-02:* "q1 drop fang for cobracmd, q2 kong generates its
own completions." Both are the recommendation, so §1 and §6 state them
as decided. The module layout, the glamour module and the committed
`go.work` were the owner's own decision (Context), so the record is
`accepted`.

* **Q1. fang.** fang styles Cobra's help, adds a man command and a version
  flag. It also sends an OSC 11 query on every help or error render, reads
  stdout's size, and writes its man page to `os.Stdout`, with no option to
  stop any of them (REPORT §7). Recommended: no fang. `command/cobracmd`
  draws Charm-styled help itself, through `SetHelpFunc` with the library's
  theme, glyphs and the caller's width, and an optional `docs` subpackage
  writes man and Markdown pages through `cobra/doc` to the caller's writer.
  The alternative is an opt-in `fangcmd` subpackage that wraps
  `fang.Execute` with `WithoutManpage`, its own man command, and a recorded
  exception to rules 1, 2 and 5.
* **Q2. Kong shell completion.** Kong has none of its own. The maintained
  third-party module, `github.com/jotaen/kong-completion`, calls
  `ctx.Exit(0)` and brings `posener/complete` (REPORT §8). Recommended: the
  Kong module generates completion scripts itself from the registry, for
  bash, zsh, fish and PowerShell, so it needs no other module. The
  alternative is to require `kong-completion`, which needs its own record.

## More Information

* [0010-REPORT-nested-modules-and-adapter-sources.md](../reports/0010-REPORT-nested-modules-and-adapter-sources.md):
  every fact this record relies on, with its source.
* [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md):
  the tooling, the `go.work`, the documentation, and the release procedure.
* [0006-MADR-command-registry.md](0006-MADR-command-registry.md),
  amendment A1: the Cobra and Kong front ends.
* [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md),
  amendment A2: the glamour renderer as a nested module.
* [0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)
  §3 (the root's stack) and §6 (the rules each adapter keeps).
