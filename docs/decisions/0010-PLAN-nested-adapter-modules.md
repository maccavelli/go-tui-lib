---
status: in-progress
date: 2026-10-04
associated-madr: "0010-MADR-nested-adapter-modules.md"
---
# Implement the multi-module repository: go.work, per-module gates, and the release procedure

Associated MADR: [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)

## Goal

Make the repository ready for nested modules before the first one lands:

* a committed `go.work` that lists every module;
* every gate run once per module, with `GOWORK=off`, from the one
  implementation in `scripts/go-precheck.sh`;
* CI that runs a matrix of module × operating system;
* depguard rules that keep each adapter's dependency in its own module;
* documentation of the module rules, the tags and the release order.

The adapter modules themselves are created by their own plans:
`command/cobracmd` and `command/kongcmd` by
[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md) Steps 10 and
11, and `stream/glamourmd` by
[0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
Step 7. Each waits for this PLAN to be complete.

Done means every item under Verification holds, CI is green on the pushed
tree, and the gates are proven, on scratch copies, to catch each fault the
MADR's Confirmation lists.

## Scope

### In scope

| Phase | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0010-*`, `docs/README.md` | accept the records |
| 2 | `go.work`, `.gitignore` | the workspace file |
| 3 | `scripts/go-modules.sh` (new), `scripts/go-modules_test.sh` (new), `scripts/go-precheck.sh`, `Makefile` | module discovery, and every gate per module |
| 4 | `.golangci.yml` | depguard for the adapters' dependencies |
| 5 | `.github/workflows/ci.yml` | the module matrix |
| 6 | `AGENTS.md`, `docs/architecture.md`, `docs/guides/releasing.md` (new), `docs/README.md`, `README.md`; by deviation D1, `scripts/go-modules.sh` and `Makefile` | documentation, close-out; D1's Windows path fixes |
| 7 | `.gitignore`, `AGENTS.md`, `docs/architecture.md`, `docs/guides/releasing.md` | MADR A1: `go.work.sum` is ignored (added 2026-10-04) |

No module is added, and no `go.mod` changes. Until an adapter lands, the
workspace holds the root module only, and every loop runs once.

### Out of scope

* The adapters' code and their `go.mod` files (0006-PLAN Steps 10–11,
  0009-PLAN Step 7).
* Dependabot. The repository has no Dependabot configuration today, and
  its go.work support was not verified as generally available
  (0010-REPORT §4). A later record may add it.
* The typed conformance scan, which
  [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  Step 7 builds. That step takes this MADR's §4 rule: it type-checks each
  module in its own context.
* `git push` and tags, which are the owner's.

### Order

This PLAN runs after `0002-PLAN-harden-workspace-v0-1-1.md` is complete,
because both change `scripts/go-precheck.sh`, the `Makefile` and CI, and
`v0.1.5` should not wait for it. It runs before any step that creates an
adapter module.

## Rules for every phase

1. **Order.** Each phase passes its checks before the next starts.
2. **Proofs on scratch copies.** Every new check is seen to fail on a
   deliberately broken input in a scratch copy of the tree, never in the
   tree, and the failure is recorded.
3. **Checks per phase:** `make pre-add-check`, `make lint`,
   `go test -race -count=1 ./...`, `LC_ALL=C go test ./...`,
   `go mod tidy -diff`, shellcheck on changed scripts, actionlint on the
   workflow, and the Windows test host on a scratch copy.
4. **Commit.** One commit per phase, made by the owner. At the end of each
   phase the agent stops with the checks passed and the phase's evidence in
   the execution record, and stages and commits nothing.

## Implementation Phases

### Phase 1: records

The owner accepts 0010-MADR, answers Q1 and Q2, and approves this PLAN.
Record the answers in the MADR, set it `accepted` and this PLAN
`in-progress`, and update `docs/README.md`. If an answer changes the MADR,
amend 0006-MADR A1 or 0009-MADR A2 to match before Phase 2.

*2026-10-02: partly done.* The owner answered Q1 (drop fang) and Q2 (Kong
generates its own completions), both as recommended. The answers are
recorded, the MADR is `accepted`, and 0006-MADR A1 states both. This
PLAN stays `proposed` until the owner approves its execution.

*2026-10-03: done.* The owner approved this PLAN, and accepted 0006-MADR
A1 and 0009-MADR A2. This PLAN is `in-progress`. Phases 2–6 run after
0002-PLAN-harden-workspace-v0-1-1.md is complete and `v0.1.5` is tagged,
as Order says (that PLAN's deviation D6 replaced `v0.1.3`).

### Phase 2: `go.work`

* `go work init .` at the root, giving `go 1.27.1` and `use .`. No
  `toolchain`, `godebug` or `replace` line.
* `.gitignore` gains nothing for `go.work`; the file is committed. If the
  `go` command writes `go.work.sum`, it is committed too. *Superseded
  2026-10-04 by MADR A1: `go.work.sum` is ignored (Phase 7).*
* **Checks:**
  * `go env GOWORK` names the file; `go list -m` prints the root module
    only;
  * `go build ./...` and `go test ./...` pass in workspace mode, and again
    with `GOWORK=off`;
  * a scratch copy with a `go.work` whose `go` line is below the root's
    fails, as 0010-REPORT §3 says it must, and the error is recorded.

### Phase 3: module discovery and per-module gates

* **`scripts/go-modules.sh`** prints each module directory, relative to
  the root, one per line, from `go list -m -f '{{.Dir}}'` in workspace mode.
  With `--check` it compares that list with the directories of the tracked
  `go.mod` files (`git ls-files '*go.mod'`), and exits 1, naming each
  difference, when they differ.
* **`scripts/go-precheck.sh`** runs every step per module, in the module's
  directory:
  * `gofmt` on the module's files;
  * with `GOWORK=off`: `golangci-lint run -c <root>/.golangci.yml ./...`
    for linux, darwin and windows with `CGO_ENABLED=0`; `go vet ./...`;
    `go test ./...`; `go mod tidy -diff`; `govulncheck ./...` unless
    `GO_PRECHECK_SKIP_VULN=1`;
  * `go test ./...` once more in workspace mode;
  * for a module other than the root: no `replace` line in its `go.mod`, and
    its requirement of the root is a release version (`vX.Y.Z`, not a
    pseudo-version);
  * once, at the root: `scripts/go-modules.sh --check`.

  Given files, it runs the steps for the modules that own them. The
  root-only behaviour of today is the one-module case, and its output for
  the root module is unchanged apart from a module heading.
* **`Makefile`:** `test`, `vet`, `lint`, `vuln` and a new `tidy-check` loop
  over `scripts/go-modules.sh`, each with `GOWORK=off`. A new
  `release-check` runs the precheck with every module and no file list.
  `fuzz` is unchanged: only `layout` has fuzz targets.
* **`scripts/go-modules_test.sh`** builds scratch trees under a temporary
  directory, never in the repository, and asserts:
  * a root with one nested module, listed in `go.work`, gives two lines;
  * a nested `go.mod` missing from `go.work` makes `--check` exit 1 and
    name it;
  * a `go.work` entry with no `go.mod` makes `--check` exit 1.
* **Proofs, each on a scratch copy of the tree with a planted nested module
  `scratchmod/`** (requiring the root at `v0.1.0`), the precheck fails, and
  the message is recorded, when:
  * `scratchmod/go.mod`'s `require` of the root is deleted (the build fails
    with `GOWORK=off`; it passes in workspace mode, as 0010-REPORT §3
    observed);
  * it requires `v9.9.9`, which is not tagged;
  * it gains `replace github.com/maccavelli/go-tui-lib => ../`;
  * it requires a pseudo-version of the root;
  * `scratchmod` is missing from `go.work`;
  * a lint issue is planted in `scratchmod` (lint reaches it);
  * a vet issue is planted in `scratchmod`.

  The same tree with none of these faults passes.

### Phase 4: depguard

* `.golangci.yml` gains a rule refusing `github.com/spf13/cobra`,
  `github.com/spf13/pflag`, `charm.land/fang/v2`,
  `github.com/charmbracelet/fang`, `github.com/alecthomas/kong` and
  `charm.land/glamour/v2` in every file outside the module that may use
  each: Cobra and pflag in `command/cobracmd/**`, Kong in
  `command/kongcmd/**`, glamour in `stream/glamourmd/**`. fang is refused
  everywhere (0010-MADR Q1).
* depguard's `files` globs and negations are checked against the
  golangci-lint version CI pins before the rule is written, and the result
  is recorded. If depguard cannot express a per-directory allowance, each
  adapter module carries its own `.golangci.yml` that differs from the root's
  only in that rule, and a check asserts that difference.
* **Proofs, on scratch copies:** an import of Kong in `layout` fails lint;
  an import of Cobra in a planted `command/kongcmd` file fails lint; an
  import of Kong there passes.

### Phase 5: CI

* A `modules` job computes the module list with `scripts/go-modules.sh`
  and passes it as a matrix.
* The test job runs for each module × `ubuntu-24.04`, `macos-15` and
  `windows-2025`, with `working-directory` set to the module. It keeps
  today's steps (race where cgo allows, shuffle and `LC_ALL=C` on Linux)
  with `GOWORK=off`, and one `go test ./...` in workspace mode.
* The lint, tidy, vet, govulncheck, cross-vet and fuzz steps run per
  module with `GOWORK=off`. Fuzz and its corpus upload stay with `layout`.
* `actions/setup-go` reads `go-version-file: go.work` and caches with
  `cache-dependency-path: '**/go.sum'`. `GOFLAGS=-mod=mod` is never set.
* shellcheck covers the new scripts. actionlint passes.
* **Proofs:** the workflow is run once on a branch the owner pushes with a
  planted scratch module, and the run shows the matrix entry for it; the
  owner removes the branch afterwards. If the owner prefers not to push a
  branch, `act` or a local reading of the workflow with actionlint is
  recorded instead, and the first adapter's PR becomes the proof.

### Phase 6: documentation and close-out

* **`AGENTS.md`:**
  * Dependencies: the root module's list is 0001-MADR §3's; each nested
    module's is 0010-MADR §5's; depguard enforces both.
  * A "Modules" section: the module table, `go.work` and what it is for,
    the `GOWORK=off` rule for every gate, `go work use` when a module is
    added, the tag scheme, and the two-step release order.
  * Pre-add checks: the per-module loop, `make release-check`.
* **`docs/guides/releasing.md`:** the release procedure, step by step, for
  the root alone, an adapter alone, and a change that spans both, with the
  consumer smoke test.
* **`docs/architecture.md`:** the modules and how they depend on each
  other.
* **`docs/README.md`:** rows for the records, and "I want to…" rows for
  adding a module and releasing one.
* **Release notes** in the execution record. The root release that carries
  this PLAN changes no API.

### Phase 7: `go.work.sum` is ignored

*Added 2026-10-04 by MADR A1.* The PLAN was `complete`; it is
`in-progress` again until this phase is done.

* `.gitignore` gains `go.work.sum`, with a comment citing MADR A1.
* `AGENTS.md`'s Modules section, `docs/architecture.md`'s tree and
  Modules section, and `docs/guides/releasing.md`'s rules say that
  `go.work` is committed and `go.work.sum` is ignored, and why.
* **Proof, on a scratch copy:** a workspace-mode `go doc` of a dependency
  package (`charm.land/bubbles/v2/help`) writes `go.work.sum`;
  `git status --short` then shows nothing, and
  `git check-ignore -v go.work.sum` names the `.gitignore` line. Before
  the change, on the same copy, `git status` shows `?? go.work.sum`.
* **Checks:** markdownlint, the link check over the changed documents,
  `make pre-add-check`, and the identifier scan.

## Verification

* Every proof of Phases 2–4 failed on its scratch copy, and the failures
  are quoted.
* `make pre-add-check`, `make release-check`, `make lint` and `make vuln`
  pass on the macOS development host and the Windows test host.
* `go list -m` in workspace mode and `scripts/go-modules.sh --check`
  agree.
* `go mod tidy -diff` is clean with `GOWORK=off`, and no `go.mod` changed.
* shellcheck, actionlint and markdownlint pass.
* The identifier scan finds nothing.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Phases 1–6. No tag is needed: nothing a
  consumer resolves changes. The first adapter module then lands through
  its own plan, with this PLAN's gates.
* **Rollback.** Before the push, each phase is one local commit. After it,
  reverting a phase restores the single-module gates. While no nested
  module exists, removing `go.work` changes no build.

## Execution Record

### Phase 1: records (2026-10-03)

* The owner approved this PLAN, picked from options: "Approve; run after
  v0.1.3". The alternatives offered were to run it before 0002's Step 8,
  or to revise it first.
* The owner accepted 0006-MADR A1 and 0009-MADR A2 in the same message.
  Each is marked `accepted (2026-10-03)`, and the PLANs of 0006 and 0009
  carry a dated note. Neither PLAN's steps change.
* The Q1 and Q2 answers were recorded on 2026-10-02 (Phase 1 above). No
  answer changed the MADR, so neither amendment needed a change before
  Phase 2.
* `docs/README.md` shows this PLAN `in-progress`, and both amendments
  `accepted`.
* Phase 2 starts after `v0.1.5` is tagged (0002-PLAN-harden-workspace-v0-1-1.md
  deviation D6).

### Phase 2: `go.work` (2026-10-03)

The owner approved Phase 2 ("Proceed") after the hardening PLAN closed:
[0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
is `complete`, and its deviation D7 made `v0.1.6` the release that
completes it, in place of `v0.1.5`.

**What changed.** `go work init .` at the root wrote `go.work`:

```text
go 1.27.1

use .
```

No `toolchain`, `godebug` or `replace` line. The `go` command wrote no
`go.work.sum`: with one module, every checksum is in `go.sum`.
`.gitignore` ignores neither file, and is unchanged.

**Checks.**

* `go env GOWORK` names the file at the root. `go list -m` prints
  `github.com/maccavelli/go-tui-lib` only.
* `go build ./...` and `go test -count=1 ./...` exit 0 in workspace mode,
  and again with `GOWORK=off`. `go mod tidy -diff` exits 0 both ways.
* **Proof on a scratch copy.** With the `go` line lowered to `go 1.26.0`,
  `go build ./...` exits 1:

  ```text
  go: module . listed in go.work file requires go >= 1.27.1, but go.work lists go 1.26.0; to download and use go 1.27.1:
      go work use
  ```

  0010-REPORT §3's rule holds: a workspace's `go` line must be at least
  every listed module's.
* `make pre-add-check`: `29 file(s) clean`, govulncheck included, in
  workspace mode. `make lint`: `make modernize` clean and `0 issues` for
  linux, darwin and windows, in workspace mode and again with
  `GOWORK=off`.
* `go test -race -count=1 ./...` and `LC_ALL=C go test -count=1 ./...`:
  every package `ok`, the conformance scan included, which now runs
  `go list` inside the workspace.
* **The Windows test host,** with `go.work` in the copied tree: `go vet`,
  `go test -race -count=1 ./...` and `LC_ALL=C go test -count=1 ./...`
  exited 0.
* No script or workflow changed, so shellcheck and actionlint have nothing
  new to check.

**Note for Phase 5.** Until then, CI runs in workspace mode, because the
committed `go.work` is found from the checkout's root. With the root as
the only module, that builds and tests the same packages with the same
requirements as `GOWORK=off`, as the checks above show. Phase 3 sets
`GOWORK=off` in the gates, and Phase 5 in CI.

### Phase 3: module discovery and per-module gates (2026-10-03)

The owner approved Phase 3 ("Proceed").

**What changed.**

* **`scripts/go-modules.sh` (new).**
  * With no argument it prints each module directory relative to the root,
    `.` for the root, from `go list -m -f '{{.Dir}}'`. It sets `GOWORK` to
    the root's `go.work` itself, so a caller running its gates with
    `GOWORK=off` still gets every module.
  * `--check` compares that list with the directories of the tracked
    `go.mod` files. It names each module `go.work` lists without a tracked
    `go.mod`, and each tracked `go.mod` missing from `go.work`, and exits 1.
    A `go.work` entry with no `go.mod` fails in `go list` itself, and that
    message is shown.
  * It skips a tracked `go.mod` under `testdata`, or under a directory
    whose name starts with `.` or `_`, as the `go` command does. None
    exists today; the PLAN did not say, and the tests pin it.
  * Without a `go.work`, `--check` exits 1 and listing exits 2.
* **`scripts/go-precheck.sh`** runs per module, in the module's
  directory, with the modules from `go-modules.sh`:
  * `gofmt` on the module's files;
  * with `GOWORK=off`: golangci-lint with the root's `.golangci.yml` for
    linux, darwin and windows; `go vet`; `go test`; `go mod tidy -diff`;
    `govulncheck` unless `GO_PRECHECK_SKIP_VULN=1`;
  * `go test` again with `GOWORK` set to the root's `go.work`;
  * for a module other than the root: no `replace` directive, and a
    requirement of the root, if any, at `vX.Y.Z` (from
    `go mod edit -print`);
  * then, once, `go-modules.sh --check`.

  Given files, it checks the modules that own them; a file belongs to the
  deepest module containing it. Each module's output starts with
  `go-precheck: module <dir> (<n> file(s))`. The summary line now names
  `go mod tidy` and the module count:
  `go-precheck: 29 file(s) clean in 1 module(s) (gofmt, golangci-lint, go vet, go test, go mod tidy, govulncheck).`
  It runs under bash 3.2, macOS's `/bin/bash`, as well as bash 5.
* **`Makefile`.**
  * `test`, `vet`, `lint`, `vuln` and the new `tidy-check` run once per
    module, in its directory, with `GOWORK=off`. A failure to list the
    modules fails the target, rather than looping over nothing.
  * `modernize`, which Step 7 of
    [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
    added after this PLAN was written, runs per module too: it is a gate,
    and 0010-MADR §4 runs every gate per module.
  * `release-check` (new) runs the precheck over every module with no file
    list. `fuzz`, `fmt`, `tidy` and `test-sum` are unchanged.
* **`scripts/go-modules_test.sh` (new)** builds throwaway git
  repositories under `mktemp -d`. It asserts 13 things:
  * a root and a nested module in `go.work` list as two lines, even with
    `GOWORK=off` set by the caller, and `--check` passes;
  * a nested `go.mod` missing from `go.work` fails `--check`, which names
    it;
  * a `go.work` entry with no `go.mod` fails `--check`, which names it;
  * a module in `go.work` whose `go.mod` is untracked fails `--check`;
  * a `go.mod` under `testdata` or `_scratch` is skipped;
  * no `go.work` fails `--check` and listing;
  * an unknown argument exits 2.

  It passes, 13 of 13. With `go-modules.sh`'s missing-module branch
  mutated away (a scratch copy, passed in as `MODULES`), it fails:
  `FAIL a module missing from go.work fails --check: want 1, got 0`.

**Proofs.** A scratch copy of the tree gained `scratchmod/`, a module
requiring the root at `v0.1.0`, tidied with `GOWORK=off`, and listed in
`go.work`. Clean, the precheck over every module passes:
`go-precheck: 30 file(s) clean in 2 module(s) (…)`. Each fault below is
then the only one in its copy:

| Fault | The precheck fails with |
| :--- | :--- |
| `require` of the root deleted | golangci-lint, `go vet` and `go test`: `no required module provides package github.com/maccavelli/go-tui-lib/glyph`; and `go mod tidy -diff`. In workspace mode, `go build ./...` in `scratchmod` exits 0, as 0010-REPORT §3 observed |
| requires `v0.9.9`, which is not tagged | `reading github.com/maccavelli/go-tui-lib/go.mod at revision v0.9.9: unknown revision v0.9.9` |
| requires `v9.9.9`, the PLAN's example | `version "v9.9.9" invalid: should be v0 or v1, not v9`. The path has no `/v9`, so the `go` command refuses the version before any lookup; `v0.9.9` is the case that is truly untagged |
| `replace github.com/maccavelli/go-tui-lib => ../`, then tidied | `go.mod (scratchmod): a replace directive (a module builds from published versions only)` |
| a pseudo-version of the root (`go get …@823f233`, then tidied) | `requires github.com/maccavelli/go-tui-lib at v0.1.6-0.20261003154935-823f23388bbb, which is not a release version (vX.Y.Z).` |
| uses `layout.ErrBadSplitName`, which `v0.1.0` lacks, with `go.mod` and `go.sum` tidy | `./scratch.go:13:22: undefined: layout.ErrBadSplitName (typecheck)`, from golangci-lint, `go vet` and `go test` |
| missing from `go.work`, over every module | `go-modules: scratchmod/go.mod is tracked, but go.work does not list scratchmod (run 'go work use scratchmod').` |
| `func Undocumented() {}` | `exported: exported function Undocumented should have comment or be unexported (revive)`, for each GOOS |
| `fmt.Sprintf("%d", "s")` | `go vet (scratchmod)`: `fmt.Sprintf format %d has arg "s" of wrong type string`, and golangci-lint's govet |

The newer-API fault is added to the PLAN's list. It is the fault 0010-MADR
§4 exists for: `go.mod` and `go.sum` are tidy, the workspace build passes,
and only a build with `GOWORK=off` sees it. The deleted `require` does not
prove `GOWORK=off`, because `go mod tidy -diff` catches it either way.

**Mutations of the precheck,** each on the copy with its fault; each
fault then passes, so each check is what catches it:

| Check mutated away | Its fault then |
| :--- | :--- |
| `export GOWORK=off` removed | the newer-API fault passes, exit 0 |
| the `replace` check | the tidied `replace` passes, exit 0 |
| the release-version check | the tidied pseudo-version passes, exit 0 |
| `go-modules.sh --check` | the module missing from `go.work` passes, exit 0 |

With those checks in place, `make test` and `make lint` in a copy with no
`go.work` exit 2 with `go-modules: no go.work at the repository root.`

**Checks.**

* `shellcheck scripts/*.sh`: exit 0. `scripts/go-fuzz_test.sh` and
  `scripts/go-modules_test.sh`: pass.
* `make pre-add-check` (all files), `make test`, `make vet`, `make lint`
  (with `make modernize`), `make tidy-check`, `make vuln` and
  `make release-check`: exit 0, each reporting module `.`.
* `go test -race -count=1 ./...`, `LC_ALL=C go test -count=1 ./...` and
  `go mod tidy -diff`: clean.
* **The Windows test host:** `go vet`, `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...` exited 0. The scripts are bash, and CI
  runs them on Linux only.
* No workflow changed. CI runs `scripts/go-modules_test.sh` from Phase 5,
  which owns the workflow; `shellcheck scripts/*.sh` there already covers
  both new scripts.
* `AGENTS.md`'s pre-add section still describes the steps correctly for
  one module. Phase 6 documents the per-module behaviour.

### Phase 4: depguard (2026-10-03)

The owner committed Phases 2 and 3 (`32a2d74`) and approved Phase 4
("proceed"). Another session's
[0001-PLAN-scaffold-charm-tui-library.md](0001-PLAN-scaffold-charm-tui-library.md)
Phase 6 committed comment-only changes to `Makefile`, `ci.yml` and three
scripts meanwhile (`df04fc2`); this phase touched none of them.

**What depguard can express, checked at the source first.** golangci-lint
v2.14.0, the version CI pins, bundles `github.com/OpenPeeDeeP/depguard/v2`
v2.2.1 (`go version -m`).

* golangci-lint passes each rule's `files` through a replacer for
  `${base-path}` and `${config-path}` only
  (`pkg/golinters/depguard/depguard.go`, `pkg/config/placeholders.go`).
* depguard compiles each pattern with `glob.Compile(exp, '/')`. A leading
  `!` makes it a negative pattern. `$all` expands to `**/*.go`
  (`settings.go:60-84`, `internal/utils/variables.go:31-33`).
* A rule applies to a file whose absolute, slash-separated path matches a
  positive pattern and no negative one (`settings.go:133-137`,
  `depguard.go:71`). Every rule that applies is checked.
* `deny` matches by plain string prefix (`settings.go:224-247`), so
  `github.com/alecthomas/kong` also covers a module such as
  `github.com/alecthomas/kong-yaml`. Kong's companion modules belong in
  `command/kongcmd` too.

So one root `.golangci.yml` can allow each package in one directory. The
PLAN's fallback, a `.golangci.yml` per adapter module, is not needed.

**What changed.** `.golangci.yml` only:

* `forbidden` also denies `charm.land/fang/v2` and
  `github.com/charmbracelet/fang`, everywhere (0010-MADR Q1).
* Three new rules, each over `$all` less one module:

  | Rule | Files | Denies |
  | :--- | :--- | :--- |
  | `cobra` | `$all`, `!**/command/cobracmd/**` | `github.com/spf13/cobra`, `github.com/spf13/pflag` |
  | `kong` | `$all`, `!**/command/kongcmd/**` | `github.com/alecthomas/kong` |
  | `glamour` | `$all`, `!**/stream/glamourmd/**` | `charm.land/glamour/v2` |

* The patterns are unanchored. `${config-path}` would anchor them to the
  repository, but on a Windows host it would put backslashes, which the
  glob library treats as escapes, into the pattern. Lint runs on Linux
  and macOS hosts today, for all three targets.

**Proofs, on scratch copies.** Every planted import resolves: each module
was fetched at the version 0010-REPORT examined (`go get`, then
`go mod tidy`, with `GOWORK=off`). A failure can therefore come only from
lint, and each is shown to be depguard's. Each planted adapter module has a
package comment; without one, revive's `package-comments` failed first,
which an earlier run showed. golangci-lint ran in each module's directory
with the root's configuration, `GOWORK=off` and `GOOS=linux`. With the rule
mutated, each outcome flips:

| Planted | Lint | With the rule mutated |
| :--- | :--- | :--- |
| Kong in `layout` | fails: `import 'github.com/alecthomas/kong' is not allowed from list 'kong': Kong only in the command/kongcmd module (…§5) (depguard)` | Kong's deny removed: passes |
| glamour in `layout` | fails: `import 'charm.land/glamour/v2' is not allowed from list 'glamour'` | glamour's deny removed: passes |
| Kong in a `command/kongcmd` module | passes | its `!**/command/kongcmd/**` removed: fails, `not allowed from list 'kong'` |
| Kong and Cobra in `command/kongcmd` | fails: `import 'github.com/spf13/cobra' is not allowed from list 'cobra'` | Cobra's deny removed: passes |
| Kong and fang in `command/kongcmd` | fails: `import 'charm.land/fang/v2' is not allowed from list 'forbidden': fang is not used, in any module (…Q1)` | fang's deny removed: passes |
| Cobra in a `command/cobracmd` module | passes | its negation removed: fails, `not allowed from list 'cobra'` |
| glamour in a `stream/glamourmd` module | passes | its negation removed: fails, `not allowed from list 'glamour'` |

The PLAN's three proofs are rows 1, 3 and 4. The others cover glamour,
fang and the Cobra module.

**End to end.** In a scratch copy with a `command/kongcmd` module importing
Kong and Cobra, listed in `go.work`, `make lint` exits 2. It lints module
`.` clean for three targets, then fails module `command/kongcmd` for
linux, darwin and windows, each with the `cobra` list's message.

**Checks.**

* `golangci-lint config verify`: exit 0.
* `make lint` (with `make modernize`): `0 issues` for module `.` on
  linux, darwin and windows.
* `make pre-add-check`: `29 file(s) clean in 1 module(s)`.
* `go test -race -count=1 ./...`, `LC_ALL=C go test -count=1 ./...`,
  `go mod tidy -diff` and actionlint v1.7.12: clean.
* **The Windows test host:** `go vet`, `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...` exited 0.
* `AGENTS.md` names depguard's refusals under Dependencies. Phase 6 adds
  the adapters' rules there.

### Phase 5: CI (2026-10-03)

The owner committed Phase 4 (`f3ea608`) and approved Phase 5 ("proceed").
`main` has no branch protection and no rulesets
(`gh api …/branches/main/protection`: 404, "Branch not protected";
`…/rulesets`: `[]`), so renaming the jobs breaks no required check.

**What changed.** `.github/workflows/ci.yml`. The single `validate`
matrix over three operating systems becomes three jobs:

* **`modules`** (Linux) runs `scripts/go-modules_test.sh` and
  `scripts/go-modules.sh --check`, and writes the module list as JSON,
  `["."]` today, to `$GITHUB_OUTPUT` for the matrix.
* **`test`** runs for each module × `ubuntu-24.04`, `macos-15` and
  `windows-2025`, named `test (<module>, <os>)`, with
  `working-directory` set to the module and `GOWORK: 'off'` for the job. It
  keeps today's steps (`go mod download`, `go test`, race where cgo allows,
  shuffle and `LC_ALL=C` on Linux), and adds `go test` in workspace mode,
  where the step sets `GOWORK` empty so the `go` command finds `go.work`.
* **`gates`** (Linux) runs the gates that loop over the modules
  themselves, through the `Makefile` and `go-modules.sh`, so CI and
  `make pre-add-check` share one implementation, as 0010-MADR §4 asks:
  * fuzz, with its corpus upload, stays with `layout` in the root module
    (`GOWORK=off`);
  * cross `go vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`
    loops over the modules with `GOWORK=off`;
  * `make vet`, `gofmt -l .`, `make tidy-check`, `make lint` (with
    `make modernize`) and `make vuln`, each per module;
  * shellcheck over `scripts/*.sh`, which now includes both new scripts,
    markdownlint and actionlint, as before.
* Every `actions/setup-go` reads `go-version-file: go.work` and caches with
  `cache-dependency-path: '**/go.sum'`. `GOFLAGS=-mod=mod` is not set.

**Reading of the PLAN.** "The lint, tidy, vet, govulncheck, cross-vet and
fuzz steps run per module" is met by the `gates` job's per-module loops
rather than by one matrix entry per module: the `Makefile` targets already
loop, and running them in each entry would repeat every module's lint in
every entry.

**Checks.**

* An empty `GOWORK` finds `go.work` from a module's subdirectory:
  `GOWORK= go env GOWORK` in `layout/` names the root's file;
  `GOWORK=off go env GOWORK` prints `off`.
* actionlint v1.7.12: exit 0. `make pre-add-check`, `make lint`,
  `shellcheck scripts/*.sh` and `go mod tidy -diff`: clean. The Windows
  test host: `go vet`, `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...` exited 0.
* **The workflow's own commands, locally.** A script read `ci.yml` and ran
  its `run:` blocks on a scratch copy with a planted `scratchmod/`, a
  module requiring the root at `v0.1.0`, with a test, listed in `go.work`:
  * the `modules` step printed `go-modules_test: 13 passed, 0 failed` and
    `modules: [".","scratchmod"]`, and wrote
    `list=[".","scratchmod"]` to the output file;
  * each `test` step exited 0 in `.` and in `scratchmod`, with the
    job's environment;
  * the `gates` steps `cross go vet`, `vet, gofmt, tidy, modernize, lint`
    and `govulncheck` exited 0, each reporting modules `.` and
    `scratchmod`.

**The real-CI proof.** The owner chose a proof branch (picked from
options, 2026-10-03), committed and pushed Phase 5 (`ce884be`), and
authorized the agent to run the branch steps as written, committing and
pushing. The workflow runs on pushes to `main`, on tags and on pull
requests, so the branch was opened as a draft pull request.

* Branch `ci-proof-scratchmod`, commit `94b8602` on `ce884be`: the same
  `scratchmod/` module, with its tidied `go.sum`, and `go.work` gaining
  `./scratchmod`. The disclosure guard passed over the commit before the
  push. Draft pull request #1, "CI proof: nested-module matrix (do not
  merge)".
* Run 37147134663: `completed success`. Jobs: `modules`, `gates`, and
  `test (.|scratchmod, ubuntu-24.04|macos-15|windows-2025)`, all six
  `success`.
* From the run's log:
  * the `modules` step printed `go-modules_test: 13 passed, 0 failed` and
    `modules: [".","scratchmod"]`;
  * `gates` ran `go vet (module scratchmod, freebsd/amd64)`, `openbsd/amd64`
    and `linux/386`; `go fix -diff (module scratchmod, GOOS=…)` and
    `golangci-lint (module scratchmod, GOOS=…)` for linux, darwin and
    windows; and `make vet`, `make tidy-check` and `make vuln` each printed
    `== module scratchmod`.
* Cleanup, by the agent with the owner's approval: `gh pr close 1
  --delete-branch` closed the pull request unmerged and deleted the branch
  on GitHub and locally, so the separate `git branch -D` found nothing
  to delete. `git ls-remote` shows no `ci-proof-scratchmod`, and
  `main` never held `scratchmod/`.

### Deviation D1 (2026-10-03): the gates on the Windows test host

* **Found.** Phase 6's Verification runs `make pre-add-check`,
  `make release-check`, `make lint` and `make vuln` on the Windows test
  host. All four failed there. Phases 2–5 ran only `go vet` and `go test`
  on that host, never the scripts or the `Makefile`.
* **Evidence**, from a copy of the tree on the host, with account names
  redacted:
  * `git rev-parse --show-toplevel` gives
    `C:/Users/<user>/AppData/Local/Temp/go-tui-lib-probe`, and
    `go list -m -f '{{.Dir}}'` gives
    `C:\Users\<user>\AppData\Local\Temp\go-tui-lib-probe`.
    `scripts/go-modules.sh` strips the first from the second, which never
    matches, so it lists the root by its absolute path. `--check` fails:
    `go.work lists C:\Users\<user>\…, which has no tracked go.mod` and
    `./go.mod is tracked, but go.work does not list .`.
  * So `scripts/go-precheck.sh` owns no file to any listed module, and its
    module loop checks nothing; only `--check` makes it fail. This defect
    came with Phase 3.
  * `make -p` shows `GOPATH_BIN := C:\Users\<user>\go/bin`. A recipe's
    shell drops the backslashes: `C:Users<user>go/bin/golangci-lint: No
    such file or directory`, and the same for `govulncheck`. This defect
    predates this PLAN: the `Makefile` came from the 0001 scaffold, and CI
    runs `make` on Linux only.
  * `git -C "<the go list Dir>" rev-parse --show-prefix` prints an empty
    prefix for the root on that host, whatever form the path takes.
* **Resolutions offered:** fix both under this deviation (recommended);
  fix `go-modules.sh` here and the `Makefile` under its own 0001 record,
  with this Verification waiting; or stop Phase 6.
* **Decision** (the owner, picked from options, 2026-10-03): fix both here.
* **Changed.** Phase 6 also edits `scripts/go-modules.sh` (each module's
  directory from `git -C <dir> rev-parse --show-prefix`) and the
  `Makefile` (the Go tool paths with `\` turned to `/`, and the lint
  loop's root path checked on the host). The Phase 3 tests and proofs that
  cover `go-modules.sh` run again on macOS, and every `make` gate runs on
  the Windows host, before the fix and after.

### Phase 6: documentation and close-out (2026-10-03)

The owner approved Phase 6 ("proceed"). The Phase 5 record's last update
was not yet committed; it was saved as its own patch, so Phase 5 and
Phase 6 can be committed apart.

**Documentation.**

* **`docs/guides/releasing.md` (new):** the rules (a tag names one
  module; tags are never moved; an adapter requires a published root
  version; `go.work` is for development); adding a module; releasing the
  root alone, an adapter alone, and a change that spans both, in 0010-MADR
  §3's two steps; and the consumer smoke test. The smoke test, as written,
  was run against the root's `v0.1.6` in a scratch module outside the
  repository: `go get … @v0.1.6` added it, and `go build ./...` exited 0.
* **`AGENTS.md`:** Dependencies says the root's list is 0001-MADR §3's
  and each nested module's is 0010-MADR §5's, names fang as refused
  everywhere and where Cobra, pflag, Kong and glamour may be imported. A
  new Modules section gives the module table, `go.work`, `GOWORK=off`,
  `go work use`, the tags and the two-step release. Pre-add checks
  describes the per-module precheck and `make release-check`.
* **`docs/architecture.md`:** a Modules section with the dependency
  direction; the tree gains `go.work`, `go-modules.sh` and its test, and
  the releasing guide; depguard's refusals and per-module allowances; the
  `make` targets, the per-module behaviour, the precheck and CI as Phases
  2–5 left them.
* **`README.md`:** the current release, `v0.1.6`; adapters as their own
  modules; a row for adding or releasing a module.
* **`docs/README.md`:** rows for adding a module, releasing, and the
  per-module gates.
* **Links.** A link check (relative links and `#` anchors) over the five
  files found 0 broken. On a copy with a planted missing file and a
  planted missing anchor, it reported both and exited 1.

**Deviation D1, done.** On the Windows test host, before the fix:
`make pre-add-check` and `make release-check` failed on
`scripts/go-modules.sh --check`, having checked no module; `make lint` and
`make vuln` could not find their tools. The fix:

* `scripts/go-modules.sh` takes each module's directory relative to the
  repository from `git -C <dir> rev-parse --show-prefix`, after checking
  `git -C <dir> rev-parse --show-toplevel` is the repository's. It gains
  `GO`, which names the go command, as `go-fuzz.sh` does, so a test can
  inject one: this host's bash resets `PATH` in non-interactive shells, so
  a `PATH` override did not reach the script.
* `scripts/go-modules_test.sh` gains case 8: the go command spells the
  module directories differently from git. Where the host's go already
  does, as on Windows, the real go is used; elsewhere a go that reports
  them through a symlink stands in. Its first check asserts the spellings
  differ, so the case cannot pass vacuously.
* `Makefile`: `GOPATH_BIN` and `GOBIN` turn `\` into `/`, and the lint
  loop's root comes from `git rev-parse --show-toplevel` instead of `pwd`.

Proofs:

| Run | Result |
| :--- | :--- |
| macOS, `go-modules_test.sh`, fixed | `17 passed, 0 failed` |
| macOS, with only D1's `go-modules.sh` change reverted | `15 passed, 2 failed`: `listing under another spelling prints the root and sub: want .,sub, got <tmp>,<tmp>`; `--check under another spelling passes: want 0, got 1` |
| Windows, fixed | `17 passed, 0 failed` |
| Windows, with D1 reverted | `11 passed, 6 failed`, among them `listing prints the root and sub: want .,sub, got C:\Users\<user>\…\two,C:\Users\<user>\…\two\sub` |
| Windows, after the fix | `go-modules.sh` lists `.`; `--check` exit 0; `make pre-add-check`, `make release-check`, `make lint` and `make vuln` exit 0 |
| Windows, planted `scratchmod/` | `go-modules.sh` lists `.,scratchmod`; the precheck on its file prints `go-precheck: module scratchmod (1 file(s))` and passes; with `layout.ErrBadSplitName`, which `v0.1.0` lacks, it fails: `undefined: layout.ErrBadSplitName` |

The Phase 3 isolated proofs and the Phase 5 local run of the workflow were
run again on macOS with the fixed scripts; every result matched its
Phase 3 and Phase 5 record.

**Verification.**

* The proofs of Phases 2–4 failed on their scratch copies, and the
  failures are quoted in their records.
* **macOS:** `make pre-add-check`, `make release-check`, `make lint` and
  `make vuln`: exit 0. `go test -race -count=1 ./...` and
  `LC_ALL=C go test -count=1 ./...`: every package `ok`.
* **Windows test host:** the four `make` targets: exit 0, as above.
* `go list -m` in workspace mode prints `github.com/maccavelli/go-tui-lib`;
  `scripts/go-modules.sh` prints `.`, and `--check` exits 0.
* `GOWORK=off go mod tidy -diff`: exit 0. `go.mod` and `go.sum` are
  unchanged from `ce884be`.
* shellcheck over `scripts/*.sh`, actionlint v1.7.12 and markdownlint:
  clean. `git diff --check`: clean.
* The identifier scan finds nothing in the changed files.
* CI on the owner's push: see Close-out below.

### Release notes

This PLAN changes no API and no `go.mod`. It adds tooling and
documentation only, so no tag is needed (Rollout). For a reader of the
history:

* **`go.work`** at the root lists every module, for development.
* **Every gate runs per module with `GOWORK=off`:**
  * `scripts/go-precheck.sh` runs per module, adds `go mod tidy -diff`, a
    workspace-mode test, and, for a nested module, the `replace` and
    release-version checks;
  * the `Makefile` targets loop over the modules, with the new
    `tidy-check` and `release-check`;
  * `scripts/go-modules.sh` lists the modules and checks `go.work`
    against the tracked `go.mod` files.
* **depguard** keeps Cobra and pflag in `command/cobracmd`, Kong in
  `command/kongcmd` and glamour in `stream/glamourmd`, and refuses fang
  everywhere.
* **CI** has a `modules` job, a `test (<module>, <os>)` matrix and a
  `gates` job, and reads Go from `go.work`.
* **Windows:** the `Makefile` and the module scripts now work on the
  Windows test host (deviation D1); `make lint` and `make vuln` did not
  before.
* **Documentation:** the releasing guide, and the module rules in
  `AGENTS.md` and `docs/architecture.md`.

### Close-out (2026-10-04)

* The owner committed Phase 5's last record and Phase 6 together as
  `28da8ac`, and pushed it.
* CI run 37163726696 on `28da8ac`: `completed success`. Jobs: `modules`,
  `gates`, and `test (., ubuntu-24.04)`, `test (., macos-15)` and
  `test (., windows-2025)`, all `success`. The nested-module matrix itself
  was proven by Phase 5's run 37147134663.
* Every item under Verification holds, per the Phase 6 record, and CI is
  green on the pushed tree. No tag was needed (Rollout). This PLAN is
  `complete`.
* The adapter modules land through their own plans:
  [0006-PLAN-command-registry.md](0006-PLAN-command-registry.md) Steps 10
  and 11, and
  [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
  Step 7, each with this PLAN's gates.

### Amendment A1 (2026-10-04): reopened for Phase 7

* MADR A1 was accepted after the close-out above, so this PLAN is
  `in-progress` again for Phase 7, which waits for the owner's approval.
* The `go.work.sum` that the 0004 audit's `go doc` calls wrote was deleted,
  with the owner's approval. It was untracked: `git ls-files
  --error-unmatch go.work.sum` exited 1. Its six lines were `/go.mod`
  checksums of `github.com/MakeNowJust/heredoc`,
  `github.com/bits-and-blooms/bitset`, `github.com/charmbracelet/harmonica`,
  `github.com/clipperhouse/stringish`, `github.com/dustin/go-humanize` and
  `github.com/sahilm/fuzzy`.

### Phase 7: `go.work.sum` is ignored (2026-10-04)

The owner asked why the workspace exists and why `go.work.sum` would be
ignored, was given the case for each way, and answered "Ignore it for
now", which approves this phase. "For now": revisit the choice when the
first adapter module lands, since a workspace then builds against the
newest version any module requires, and CI's workspace-mode test may use
checksums no single `go.sum` holds.

**What changed.**

* **`.gitignore`** gains `go.work.sum` (line 21), with a comment: `go.work`
  is committed; `go.work.sum` is not, because ad-hoc workspace-mode
  commands write it and no gate reads it, citing MADR A1.
* **`AGENTS.md`,** Modules: the `go.work` bullet says `go.work.sum` is
  ignored, not committed, and why.
* **`docs/architecture.md`:** the Modules section and the tree say
  `go.work` is committed and `go.work.sum` ignored.
* **`docs/guides/releasing.md`:** the `go.work` rule says the same.

**Proof, on a scratch copy** committed with `HEAD`'s `.gitignore`:

* `go doc charm.land/bubbles/v2/help` in workspace mode exited 0 and wrote
  `go.work.sum`;
* with `HEAD`'s `.gitignore`, `git status --short` showed
  `?? go.work.sum`;
* with the new `.gitignore`, it showed only ` M .gitignore`, the change
  itself, and `git check-ignore -v go.work.sum` printed
  `.gitignore:21:go.work.sum	go.work.sum`.

**Checks.** markdownlint: 0 issues. The link check over `AGENTS.md`,
`docs/architecture.md` and `docs/guides/releasing.md`: 0 broken.
`make pre-add-check`: `40 file(s) clean in 1 module(s)`. `git diff
--check`: clean. The identifier scan of the changed files finds nothing.
No `go.work.sum` exists in the repository.

This PLAN returns to `complete` after the owner's push and a green CI run,
as Phase 6's close-out did: see below.

### Close-out, after Phase 7 (2026-10-04)

* At the owner's request ("Commit and push"), the agent committed Phase 7
  as `2fe6123` and pushed `main` (`edcf882..2fe6123`), after the
  disclosure guard passed.
* CI run 37184935565 on `2fe6123`: `completed success`. Jobs: `modules`,
  `gates`, and `test (., ubuntu-24.04)`, `test (., macos-15)` and
  `test (., windows-2025)`, all `success`.
* This PLAN is `complete` again. MADR A1's "for now" stands: revisit
  ignoring `go.work.sum` when the first adapter module lands.

### Phase 8: the consumer smoke test, corrected (2026-10-04)

**Deviation D2 (2026-10-04).** After the owner pushed the annotated tag
`v0.2.0` on `0224d0e` (CI run 37185968146: `modules`, `gates` and `test`
on ubuntu-24.04, macos-15 and windows-2025, all `success`), the agent ran
the releasing guide's consumer smoke test against it, importing
`workspace`. It failed at `go build ./...` with "missing go.sum entry".
The guide ran `go get` before the program existed and never ran `go mod
tidy`. Phase 6 proved the steps with `glyph`, which has no outside
dependencies, so the check was too weak to see it. The tag is sound; the
procedure was not. The owner answered "Amend commit and push", which
approves MADR amendment A2 and this phase.

**What changed.**

* **`docs/guides/releasing.md`,** the consumer smoke test: `main.go` is
  written before `go get`, and `go mod tidy` runs before `go build
  ./...`, with a paragraph saying why. The root's variant imports a
  package with outside dependencies, such as `workspace`, and says why
  `glyph` proves less.
* **0010-MADR:** §3's smoke-test bullet is annotated, and amendment A2
  records the finding and the decision.
* **`docs/README.md`:** the MADR row reads "A1, A2 accepted"; the PLAN row
  reads `in-progress` until the close-out.

**Proof,** against the published `v0.2.0`, in scratch modules outside the
repository with `GOWORK=off`, each importing one package with a blank
import:

| Order | Package | `go build ./...` |
| :--- | :--- | :--- |
| old: `go get`, write `main.go`, build | `workspace` | exit 1, "missing go.sum entry for module providing package charm.land/bubbles/v2/help" (and `key`, bubbletea, colorprofile) |
| old | `glyph` | exit 0; `go.mod` lists the root as `// indirect` |
| new: write `main.go`, `go get`, `go mod tidy`, build | `workspace` | exit 0; `go.mod` has `require github.com/maccavelli/go-tui-lib v0.2.0` |
| new, without `go mod tidy` | `workspace` | exit 1, the same "missing go.sum entry" |

The last row shows that `go mod tidy` is required, not just the order.

**Checks.** markdownlint: 0 issues. The link check over the four changed
files: 0 broken. `make pre-add-check`: `40 file(s) clean in 1 module(s)`.
`git diff --check`: clean. The identifier scan of the diff finds nothing.

This PLAN returns to `complete` after the push and a green CI run.
