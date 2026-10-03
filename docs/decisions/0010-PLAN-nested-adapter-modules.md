---
status: in-progress
date: 2026-10-03
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
| 6 | `AGENTS.md`, `docs/architecture.md`, `docs/guides/releasing.md` (new), `docs/README.md`, `README.md` | documentation, close-out |

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
  `go` command writes `go.work.sum`, it is committed too.
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
