---
status: complete
date: 2026-10-01
associated-madr: "0001-MADR-scaffold-charm-tui-library.md"
---
# Implement the go-tui-lib Go 1.27.1 Charm v2 library scaffold

Associated MADR: [0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)

## Goal

Bring `go-tui-lib` to go-core-lib's standard: agent rules, lint and pre-add
tooling, CI, a Go 1.27.1 module, a documentation tree, and the TUI
conventions of MADR §6. Adapt it as MADR §4 and §5 say, so that the first
package's pair lands code without deciding anything about the repository.

Done means every item under Verification holds, and the execution record
carries the output.

## Scope

### In scope (this repository only)

| Path | Source | Change from source |
| :--- | :--- | :--- |
| `AGENTS.md` | go-core-lib | Intro, Dependencies and the identity sentence rewritten; a new "TUI conventions" section (MADR §6); the `TestManifestDifferential` and `make apicheck` sentences dropped; record citations point here |
| `.claude/.gitignore` | go-core-lib | none |
| `.claude/rules/madr-and-plan-skill.md` | go-core-lib | none |
| `.grok/rules/madr-plan-before-mutating-work.md` | go-core-lib | none |
| `.opencode/rules.md` | go-core-lib | none |
| `opencode.json` | go-core-lib | none |
| `.gitignore` | go-core-lib | none |
| `.markdownlint-cli2.jsonc` | go-core-lib | none |
| `LICENSE` | go-core-lib | none (Apache License 2.0, MADR §7) |
| `.gitattributes` | go-core-lib | Comment re-pointed at this record and at golden files; the `selfupdate/testdata/manifest-parity/** -text` rule removed |
| `.golangci.yml` | go-core-lib | `depguard` enabled, with the deny list of MADR §3 |
| `Makefile` | go-core-lib | Header comment; `apicheck` and `fuzz` removed |
| `scripts/go-precheck.sh` | go-core-lib | Provenance comment only; mode `0755` |
| `.github/workflows/ci.yml` | go-core-lib | MADR §4's kept steps, the `LC_ALL=C` leg, govulncheck v1.8.0 |
| `go.mod` | new | `module github.com/maccavelli/go-tui-lib`, `go 1.27.1`, nothing else |
| `README.md` | the owner's file at `ff5c504` | Its three paragraphs kept; status, links, "I want to…" and licence added |
| `docs/README.md` | new | Record index and "I want to…" table |
| `docs/architecture.md` | new | The tree and tooling as they are at Phase 3's commit |
| `docs/decisions/0001-*` | this pair | Status and execution record |
| `.git/config` | local | `user.name` and `user.email` (MADR §8, Q4); not a tracked file |

"go-core-lib" means that repository at the commit that completes its
`docs/decisions/0007-PLAN-adopt-govulncheck-v1-8.md`, read from the
sibling checkout, never from memory. That PLAN runs first, so the install
hints copied here already name `govulncheck@v1.8.0`.

### Out of scope

* Any Go source, test or `testdata/`. The first package (`updatetea`, or an
  extraction from ocp-login) is the next greenfield pair, `0002`.
* `go.sum` and any `require` line.
* `.github/dependabot.yml`, `docs/reports/`, `docs/guides/`, and any release
  workflow.
* Any change in go-core-lib, ocp-login or another fleet repository.
* `git push` and tags. Under MADR §8 the scaffold commits are pushed with the
  first package, and only when the owner asks in the same turn.

### Fixed inputs

* Toolchain `go1.27.1`; golangci-lint v2.14.0; govulncheck v1.8.0, on
  `PATH` once go-core-lib's 0007 PLAN has run;
  `markdownlint-cli2` 0.23.2, `shellcheck` and actionlint v1.7.12 (through
  `go run`).
* The hooks path resolves to `~/.global-git-hooks` (checked 2026-10-01).
* Commits land on `main`. Each commit needs the owner's explicit
  authorization for `main` in the turn it is made (the branch-gates rule).

## Implementation Steps

Every phase ends with its checks passing, then `git add` of exactly the
phase's paths, then `git commit --no-edit`. The global `prepare-commit-msg`
hook writes the message. No `-m`, `-F` or `--amend`.

### Phase 0: accept the records

1. The owner approves the MADR and this PLAN. *(Q1–Q4 answered
   2026-10-01; the MADR records them.)*
2. Set the MADR `status: accepted` and this PLAN `status: in-progress`.
   Quote the approval in the execution record.
3. Set the local identity (Q4) to the name and email the owner named,
   which are go-core-lib's local ones: copy them from go-core-lib's local
   configuration without printing either. Assert with a comparison that
   they are equal and that `git config --local --get user.name` succeeds.
4. Commit the two records alone, under the bootstrap exception.

### Phase 1: agent rules and repository hygiene

1. Copy `.claude/.gitignore`, `.claude/rules/madr-and-plan-skill.md`,
   `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`,
   `opencode.json`, `.gitignore`, `.markdownlint-cli2.jsonc` and `LICENSE`
   byte for byte. Assert each with `cmp`.
2. Write `.gitattributes` from go-core-lib's: keep `* text=auto eol=lf` and
   the binary patterns, remove the `manifest-parity` rule, and rewrite the
   comment to give this repository's reasons: Windows CI, golden files that
   hold escape sequences and exact widths, and shell scripts.
3. Write `AGENTS.md` from go-core-lib's, keeping its sections and their
   normative wording:
   * **Intro.** `go-tui-lib` (`github.com/maccavelli/go-tui-lib`): reusable
     terminal-UI code on Charm v2; a library only; one directory per
     capability; no root package; Go 1.27.1.
   * **Dependencies.** No module without a MADR here. The named stack and the
     "never" list of MADR §3, and that `depguard` enforces the list.
     `go.mod` / `go.sum` change with the code; `go mod tidy -diff` is clean
     at every commit.
   * **TUI conventions.** The six rules of MADR §6, each one sentence and a
     record citation.
   * **MADR and PLAN, and Records.** Unchanged.
   * **Pre-add checks.** Unchanged, except:
     * the `TestManifestDifferential` sentence and `make apicheck` are
       dropped;
     * one sentence: until the first package lands, `make test`, `make vet`,
       `make lint` and `make vuln` fail with "no packages" (MADR §8).
   * **Identifiers and Commits.** Unchanged, apart from the disclosure-guard
     example, which already names `main` and stays.
4. Checks:
   * `cmp` for the eight verbatim files;
   * `diff` of `AGENTS.md` and `.gitattributes` against go-core-lib's, read
     in full, showing only the changes in steps 2 and 3;
   * `markdownlint-cli2 AGENTS.md` (the config excludes records only);
   * the identifier scan (V7).
5. Commit.

### Phase 2: Go module, lint, pre-add gate and CI

1. **`go.mod`.** Write three lines: `module github.com/maccavelli/go-tui-lib`,
   a blank line, `go 1.27.1`. `go mod verify` and `go mod tidy -diff` exit
   0, `go.mod` is byte-identical after tidy, and no `go.sum` appears.
2. **`.golangci.yml`.** Copy go-core-lib's. Add `depguard` to `enable`, and
   under `settings` a `depguard` rule set that denies:
   * `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`
     and `github.com/charmbracelet/bubbles`, with a message naming the
     `charm.land/…/v2` path;
   * `github.com/maccavelli/mcplib`, `github.com/modelcontextprotocol/go-sdk`
     and `github.com/maccavelli/go-llmprovider-sdk`, with a message citing
     MADR §3.

   The rule covers test files too. Assert with `diff` that these are the only
   changes, and that `golangci-lint config verify -c .golangci.yml` exits 0.
3. **`Makefile`.** Copy go-core-lib's. Change the header comment to name
   `go-tui-lib`, and remove the `apicheck` and `fuzz` targets, their
   variables and comments, and their `.PHONY` entries. Assert with `diff`.
4. **`scripts/go-precheck.sh`.** Copy it, mode `0755`. Change only the
   provenance comment, so it says it was taken from go-core-lib's under this
   record. `shellcheck scripts/go-precheck.sh` exits 0.
5. **`.github/workflows/ci.yml`.** Copy go-core-lib's, then:
   * remove the fuzz, fuzz-corpus, API-compatibility, release-fixture,
     release-guard, release-tag-rule and workflow-contract steps;
   * remove `SELFUPDATE_REQUIRE_PYTHON` from the race step, and the
     full-history `fetch-depth`, which only the API gate needs;
   * add, on Linux, `LC_ALL=C go test -count=1 ./...` as its own step;
   * change govulncheck to `v1.8.0`;
   * keep every action SHA, the shellcheck pin and its SHA-256, and the
     concurrency block.

   Check: actionlint v1.7.12 and `shellcheck` are clean. Read the whole file
   after the edit.
6. **Expected failures, on the committed tree.** Run `make test`,
   `make vet`, `make lint` and `make vuln`, each redirected to a scratch file
   with its `$?` captured. Each must fail with a "no packages" message.
   Record the exit codes and first lines. `make pre-add-check` must exit 0
   with `go-precheck: no Go files to check.`
7. **First-fail experiment, on a scratch clone, never the tree.** Copy the
   Phase 2 working tree into the scratchpad. Plant `probe/probe.go`
   (`package probe`, with a package comment), then:
   * **(a)** add an exported function with no doc comment. `make pre-add-check`
     and `make lint` must exit non-zero, naming revive's `exported` rule;
   * **(b)** misformat the file. `make pre-add-check` must list it under
     `gofmt:`;
   * **(c)** import `github.com/charmbracelet/lipgloss` (v1).
     `make lint` must fail with depguard's message naming
     `charm.land/lipgloss/v2`;
   * **(d)** import `github.com/maccavelli/mcplib`. `make lint` must fail
     with depguard's message. Because neither module is required, record
     whether the failure is depguard's or a type-check error. Only
     depguard's counts: if the type checker fails first, require the module
     in the scratch clone and repeat;
   * **(e)** make it clean: documented, formatted, importing
     `charm.land/lipgloss/v2` at its newest release, with a passing
     `probe_test.go`. `make pre-add-check`, `make lint`, `make vet`,
     `make test` and `make vuln` exit 0, and `go mod tidy -diff` exits 0;
   * **(f)** run the same clean package with `LC_ALL=C go test ./...`; it
     exits 0.

   Record each exit code and failure line. Delete the scratch clone.
8. Commit `go.mod`, `.golangci.yml`, `Makefile`, `scripts/go-precheck.sh`
   and `.github/workflows/ci.yml`.

### Phase 3: documentation tree

1. **`README.md`.** Keep the owner's three paragraphs. Add:
   * a "Status" section: no package yet; the stack (MADR §3); the gates fail
     with "no packages" until the first package; no release;
   * a "Documentation" link to `docs/README.md`;
   * an "I want to…" table;
   * a "License" line naming the Apache License 2.0 and linking `LICENSE`.
2. **`docs/README.md`.** The record index (the 0001 MADR and PLAN, with their
   status) and the "I want to…" table, phrased as reader tasks, for example
   "know which Charm version to use", "know who owns the screen and
   ctrl+c", and "add a package".
3. **`docs/architecture.md`.** The tree and tooling as they are at this
   commit, with no history and no rationale. A "What is not here" list names
   the first package, `apicheck`, `docs/reports/` and `docs/guides/`.
4. **Checks.**
   * `markdownlint-cli2` over the three files exits 0.
   * A throwaway resolver checks every relative link and anchor in them.
     It is proven first on a scratch copy with one planted dead link and one
     bad anchor, both of which it must report.
5. Set this PLAN `status: complete` once V1–V7 hold, fill in the execution
   record, and commit Phase 3's files with this PLAN.

## Verification

* **V1. Provenance.** `cmp` passes on the eight verbatim files. `diff`
  against go-core-lib shows only the allowed changes for `AGENTS.md`,
  `.gitattributes`, `.golangci.yml`, `Makefile`, `scripts/go-precheck.sh`
  and `ci.yml`.
* **V2. Module.** `go.mod` is exactly three lines. `go mod verify` and
  `go mod tidy -diff` exit 0. There is no `go.sum`.
* **V3. Honest gates.** Phase 2 step 6 behaves as stated.
* **V4. Gates proven.** Phase 2 step 7 (a)–(f) behave as stated, with the
  output recorded.
* **V5. Tooling.** `golangci-lint config verify`, `shellcheck scripts/*.sh`
  and actionlint exit 0.
* **V6. Markdown.** markdownlint-cli2 is clean over every non-record
  Markdown file, and every relative link resolves.
* **V7. Identifiers.** The committed tree contains none of these:
  * the local account name, the hostname, or a real-machine absolute path;
  * ocp-login's org-internal module host.

  The scan describes what it looked for and never quotes it.

## Rollout and Rollback

* **Rollout.** None until the first package. The commits stay local (MADR
  §8). The owner pushes them with that package.
* **Rollback.** Each phase is one commit, and nothing outside this
  repository changes. While unpushed, a phase is undone only on the owner's
  explicit ask.
* **Consumers.** None yet. The first consumer's record states the Go 1.27.1
  requirement (MADR Consequences).

## Execution Record

### Phase 0: accept the records (2026-10-01)

* **Approval.** The owner answered Q1–Q4 (MADR "Owner questions"), then
  approved the pair: "approved, proceed, commit to main". The MADR is
  `accepted` and this PLAN `in-progress`.
* **Order.** go-core-lib's
  `docs/decisions/0007-PLAN-adopt-govulncheck-v1-8.md` ran first, and its
  commit `ef05dfe` is the go-core-lib source for Phases 1 and 2. All four
  development hosts now run govulncheck v1.8.0.
* **Identity.** `user.name` and `user.email` were copied from go-core-lib's
  local configuration without being printed. A comparison showed them
  equal to the identity the owner named, and
  `git config --local --get user.name` succeeds.
* Committed as `bcfab0b`, the two records alone.

### Phase 1: agent rules and repository hygiene (2026-10-01)

* **Verbatim.** `cmp` passed on all eight files: `.claude/.gitignore`,
  `.claude/rules/madr-and-plan-skill.md`,
  `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`,
  `opencode.json`, `.gitignore`, `.markdownlint-cli2.jsonc` and `LICENSE`.
* **`.gitattributes`.** The diff against go-core-lib shows only:
  * the fixture bullet, now about golden files;
  * the citation, now this record's §4;
  * the `manifest-parity` rule, removed.
* **`AGENTS.md`.** The diff, read in full, shows only step 3's changes:
  * the intro;
  * Dependencies, with the named stack, the "never" list and `depguard`;
  * the new "TUI conventions" section;
  * the `TestManifestDifferential` sentence, dropped;
  * the citation of go-core-lib's 0002 §5, dropped;
  * the "no packages" sentence.
* **markdownlint.** `markdownlint-cli2 AGENTS.md` lints every non-record
  file the config globs, so it also read `README.md`. It exited 1 on two
  MD013 findings in `README.md` lines 3 and 5 (235 and 289 characters).
  * Those lines are the owner's file at `ff5c504`, untouched by Phase 1.
    `AGENTS.md` had no finding.
  * The commit (`a65b2c8`) went ahead because the command chain did not
    gate on that exit status. That was an execution slip: from Phase 2
    on, every check gated its commit.
  * Phase 3 re-wrapped the README, and markdownlint is clean (V6).
* **Identifiers.** The staged diff had no hit for the local account name,
  a real-machine home path, either development hostname or its domain,
  the owner's email, or ocp-login's org-internal module host.

### Phase 2: Go module, lint, pre-add gate and CI (2026-10-01)

* **`go.mod`** is the three lines. `go mod verify` printed `all modules
  verified`, and `go mod tidy -diff` exited 0. `cmp` against the three
  lines passed, and there is no `go.sum` (V2).
* **`.golangci.yml`.** The diff against go-core-lib is `depguard` in
  `enable`, plus a `forbidden` rule (`list-mode: lax`, `files: $all`)
  denying the six paths of MADR §3, each with a message citing it.
  `golangci-lint config verify` exited 0 with golangci-lint v2.14.0.
* **`Makefile`.** The diff is the header, the `.PHONY` list, the removed
  `apicheck` and `fuzz` targets, and one citation (D1).
* **`scripts/go-precheck.sh`.** Mode `0755`. The diff is the provenance
  comment and one citation (D1). `shellcheck` exited 0.
* **`.github/workflows/ci.yml`.**
  * Kept, from go-core-lib: checkout, setup-go, `go mod download`,
    `go test`, `-race`, `-shuffle`, cross `go vet`, the vet / gofmt /
    tidy / lint step, govulncheck, and the shellcheck / markdownlint /
    actionlint step. The action SHAs, the shellcheck SHA-256 and
    `concurrency` are unchanged.
  * Removed: the seven selfupdate-specific steps, the full-history
    `fetch-depth`, and `SELFUPDATE_REQUIRE_PYTHON`.
  * Added: `go test LC_ALL=C` on Linux.
  * govulncheck is `v1.8.0`, the value go-core-lib's 0007 already
    committed.
  * Citations of go-core-lib's records name that repository (D1).
  * The first build of the file attached each step's comment to the
    step before it, which the full read caught. The splitter was fixed
    and the file rebuilt and read again. actionlint v1.7.12 exited 0, and
    the YAML parses to 11 steps.
* **Expected failures, on the working tree** (V3):

  | Command | rc | First line |
  | :--- | :--- | :--- |
  | `make test` | 2 | `go: warning: "./..." matched no packages` / `no packages to test` |
  | `make vet` | 2 | `no packages to vet` |
  | `make lint` | 2 | `context loading failed: no go files to analyze` (each GOOS) |
  | `make vuln` | 2 | `govulncheck: no packages matched the provided patterns` |
  | `make pre-add-check` | 0 | `go-precheck: no Go files to check.` |

  `make` exits 2 when a recipe fails.
* **First-fail experiment, on a scratch copy, then deleted** (V4):

  | Case | Command | rc | Failure line |
  | :--- | :--- | :--- | :--- |
  | (a) undocumented export | `make pre-add-check`, `make lint` | 2, 2 | `exported: exported function Exported should have comment or be unexported (revive)` |
  | (b) misformatted | `make pre-add-check` | 2 | `gofmt: these files are not formatted` / `probe/probe.go` |
  | (c) v1 lipgloss, unrequired | `make lint` | 2 | `could not import github.com/charmbracelet/lipgloss … (typecheck)`, not depguard, so the case was repeated |
  | (c) v1 lipgloss, required (v1.1.0) | `make lint` | 2 | `import 'github.com/charmbracelet/lipgloss' is not allowed from list 'forbidden': Charm v1: use charm.land/lipgloss/v2 (…§3) (depguard)` |
  | (d) mcplib, required (v1.6.0) | `make lint` | 2 | `import 'github.com/maccavelli/mcplib' is not allowed from list 'forbidden': this module never imports mcplib (…§3) (depguard)` |
  | (e) clean, `charm.land/lipgloss/v2` v2.0.6 | pre-add, lint, vet, test, vuln, `tidy -diff` | 0 each | `go-precheck: 2 file(s) clean (…)` |
  | (f) clean, `LC_ALL=C go test ./...` | | 0 | `ok  github.com/maccavelli/go-tui-lib/probe` |

  In (e), the first `go mod tidy -diff` exited 1. The experiment had run
  `go get` without the `go mod tidy` that follows it, which left lipgloss
  `// indirect`. That is the gate refusing an untidy `go.mod`. After
  `go mod tidy`, every gate in (e) and (f) exited 0. The pre-add gate does
  not run `tidy -diff`; CI does.
* Committed as `5530ac9`.

**Deviation D1 (2026-10-01): citations of go-core-lib records.**

* **Found.** Copied text cited `docs/decisions/0002-…`, `0003-…` and
  `0004-…` records that exist only in go-core-lib:
  * the `Makefile` lint comment;
  * one comment in `scripts/go-precheck.sh`;
  * six citations in `ci.yml`.

  Kept as they were, those citations would resolve to nothing here.
* **Resolution.** Each now names go-core-lib and the record's full
  filename, as AGENTS.md "Records" requires for another repository's
  record. Steps 3 and 4 allowed only the header and provenance comments
  to change, so this is recorded here.
  * The provenance comment of `go-precheck.sh` says so.
  * No behaviour changed, and the MADR is unaffected.

### Phase 3: documentation tree (2026-10-01)

* **`README.md`.** The owner's three paragraphs and title are kept word
  for word. A comparison with whitespace normalised found all four blocks
  of `ff5c504`. The lines are re-wrapped to clear MD013. Added: Status,
  Documentation, "I want to…" and License.
* **`docs/README.md`** holds the record index and ten "I want to…" rows;
  **`docs/architecture.md`** the tree, dependencies, tooling, and "What is
  not here".
* **Checks** (V6):
  * markdownlint-cli2 0.23.2 linted 5 files with 0 issues;
  * the link resolver found 0 broken links over the three files;
  * on a scratch copy with a planted missing record and a planted bad
    anchor, it reported both and exited 1.
* **Not run here.** Under MADR §8 the scaffold is not pushed, so CI has
  not run on it. Its first run is with the first package.
* V1–V7 hold, so this PLAN is `complete`.
