---
status: in-progress
date: 2026-10-08
associated-madr: "0015-MADR-precheck-gofmt-errors.md"
---
# Implement: the pre-add check fails when gofmt fails, and skips a tracked file the work tree no longer has

Associated MADR: [0015-MADR-precheck-gofmt-errors.md](0015-MADR-precheck-gofmt-errors.md)

## Goal

`scripts/go-precheck.sh` refuses a Go file gofmt cannot read or parse, on
both paths. With no file list it checks only the tracked Go files the work
tree has, as it already does with a list. A test proves both, and CI runs
it.

## Scope

### Facts this PLAN starts from (2026-10-08)

| Fact | Where it was read |
| :--- | :--- |
| the gofmt step reads `gofmt -l`'s output, not its status | `scripts/go-precheck.sh:237-243` |
| the file-list path keeps an argument only if `[ -f "$f" ]` | `scripts/go-precheck.sh`, the argument loop |
| the no-list path takes `git ls-files '*.go'` as it is | `scripts/go-precheck.sh`, the same block |
| `gofmt -l` exits 2, with nothing on stdout, for a missing file and for one that does not parse | the probe in 0015-MADR's context |
| H1: a testdata file that does not parse passed, "1 file(s) clean", exit 0 | a scratch clone, 2026-10-08 |
| no test covers `go-precheck.sh`; the other four scripts each have one | `scripts/` |
| CI's `gates` job runs the four script tests | `.github/workflows/ci.yml` |

### In scope

| Step | Paths | Delivers |
| :--- | :--- | :--- |
| 1 | this pair, `docs/README.md` | the records |
| 2 | `scripts/go-precheck.sh`, `scripts/go-precheck_test.sh`, `.github/workflows/ci.yml`, `AGENTS.md`, `docs/architecture.md` | the fix, its test, CI, the descriptions |
| 3 | this PLAN | close-out |

### Out of scope

* go-selfupdate-lib's copy of the script.
* Any other step of the precheck: each already checks its exit status.
* A release. The scripts are not part of the module a consumer imports.

## Implementation Steps

### Rules

1. A step starts when the previous one is committed. The agent commits on
   `main` only when the owner asks, with `git commit --no-edit`. Pushes
   and tags need the owner's ask.
2. Checks for Step 2:
   * shellcheck 0.11.0 on `scripts/*.sh`, and actionlint v1.7.12;
   * `make pre-add-check` (no list) and `make release-check`;
   * `make pre-add-check FILES="<a Go file>"`;
   * the four existing script tests;
   * the Windows test host: the new test, and `make pre-add-check`;
   * markdownlint, the link check, and the identifier scan of the diff.
3. Each mutation runs on a scratch copy and must fail its named case.
4. Anything this PLAN does not say stops the step for the owner.

### Step 1: records

* This pair, and its two rows in `docs/README.md`.

**Done when** the owner approves this PLAN.

### Step 2: the fix, its test, CI and the descriptions

* **`scripts/go-precheck.sh`:**
  * **The no-list path** keeps a `git ls-files '*.go'` entry only when
    `[ -f "$f" ]`, as the file-list path does.
  * **The gofmt step:**

    ```bash
    unformatted="$(gofmt -l "${mfiles[@]}" 2>"$gofmt_err")"
    gofmt_rc=$?
    ```

    * A non-zero `gofmt_rc` shows "gofmt: failed (exit N):" and gofmt's
      standard error, indented, then `fail 1`.
    * The unformatted list is reported as now.
    * `gofmt_err` is a temporary file the script removes on exit.
  * **The header comment** says both.
* **`scripts/go-precheck_test.sh`,** in the style of
  `go-apicheck_test.sh`:
  * **The setup.** Each case is a throwaway git repository in a temporary
    directory:
    * a module with one package and a test;
    * `go.work`;
    * a copy of `scripts/go-precheck.sh` and `scripts/go-modules.sh`;
    * a stub golangci-lint (`GOLANGCI_LINT`) that exits 0;
    * `GO_PRECHECK_SKIP_VULN`, `_APICHECK` and `_EXAMPLES` set to 1.
  * **The cases:**

    | Case | Path | Expect |
    | :--- | :--- | :--- |
    | a clean tree | no list | 0, "clean" |
    | an unformatted file | file list | 1, its name under "gofmt:" |
    | a file under `testdata` that does not parse | file list | 1, "gofmt: failed", gofmt's message |
    | the same file, tracked | no list | 1, "gofmt: failed" |
    | a tracked test file deleted from the work tree | no list | 0, and no `lstat` |
    | a file list naming a missing file and a good one | file list | 0 |
* **`.github/workflows/ci.yml`:** the `gates` job gains a step
  `precheck test` running `./scripts/go-precheck_test.sh`, after the
  examples step, with a comment citing this PLAN.
* **`AGENTS.md`, "Pre-add checks":** a sentence. gofmt's own failure, a
  file it cannot read or parse, fails the check. With no list, the files
  are the tracked Go files the work tree has.
* **`docs/architecture.md`:**
  * the tree lists `go-precheck_test.sh`;
  * the precheck bullet says the same as `AGENTS.md`;
  * the CI bullet names the new step.

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| P1 | gofmt's status ignored again | the case "a file under `testdata` that does not parse" |
| P2 | the no-list path's `[ -f ]` removed | the case "a tracked test file deleted": it shows `lstat` |

P2 also fails only once P1's fix is in: with gofmt's status ignored, the
`lstat` error does not fail the run.

**Done when** Rule 2's checks are clean and P1 and P2 are killed.

### Step 3: close-out

* Verification, item by item, with output. This PLAN `complete`, and the
  MADR `accepted`.

## Verification

* The new test passes on macOS, on the Windows test host and in CI.
* P1 and P2 are killed.
* H1, run again, fails with "gofmt: failed".
* `make release-check` on the tree reports what it reported before.

## Rollout and Rollback

* **Rollout:** one commit for Step 2. Nothing a consumer imports changes,
  and no release is needed.
* **Rollback:** reverting Step 2's commit restores the old step. The test
  and its CI step go with it.

## Execution Record

### Step 1: records

The pair was committed as `5c49566` and `d33f165`. The owner approved
execution on 2026-10-08 ("committed and pushed for this repo. proceed to
fix."), choosing the MADR's recommended option.

### Step 2: the fix, its test, CI and the descriptions

No deviation.

#### What was built

* **`scripts/go-precheck.sh`:**
  * **The no-list path** keeps a `git ls-files '*.go'` entry only when
    `[ -f "$f" ]`.
  * **The gofmt step** writes gofmt's standard error to a file in a
    temporary directory, which the script removes on exit (`trap`), and
    keeps gofmt's exit status. A non-zero status shows "gofmt: failed
    (exit N):" and gofmt's message, through the script's `show`, then
    `fail 1`. The unformatted list is reported as before.
  * **The header** says both, and cites 0015-MADR.
* **`scripts/go-precheck_test.sh`:** the PLAN's six cases, 12
  assertions.
  * gofmt, `go vet` and `go test` are the real ones. golangci-lint is a
    stub that passes, and the three network gates are skipped.
  * The broken source goes in through `printf '%s\n'`, not as the
    format string (shellcheck SC2059).
* **`.github/workflows/ci.yml`:** the step `precheck test`, after
  `examples`.
* **`AGENTS.md`, "Pre-add checks":** a paragraph after the `testdata`
  one.
* **`docs/architecture.md`:** the tree, the precheck bullet and the CI
  bullet.
* **The records:** 0015-MADR is accepted, this PLAN is `in-progress`, and
  `docs/README.md`'s rows follow.

#### Checks

* **Mutations, on copies of the script, run by the test through
  `PRECHECK`.** Both killed:
  * P1 (gofmt's status ignored again): "a testdata file that does not
    parse fails: want 1, got 0", and three more failures; 7 passed, 5
    failed;
  * P2 (the no-list `[ -f ]` removed): "a deleted tracked file is
    skipped, no list: want 0, got 1", and "gofmt is not given it: a line
    holds lstat"; 10 passed, 2 failed.
* **H1 again,** on a scratch clone with the fixed script: exit 1, with
  "gofmt: failed (exit 2):" and "testdata/frameworks/flag/broken.go:3:12:
  expected ')', found '{'". Before the fix it printed "1 file(s) clean"
  and exited 0.
* **Rule 2 on macOS:**
  * shellcheck 0.11.0 on `scripts/*.sh` and actionlint v1.7.12: clean;
  * `make pre-add-check` (no list) and `make release-check`: "155 file(s)
    clean in 1 module(s) (gofmt, golangci-lint, go vet, go test, go mod
    tidy, govulncheck, apicheck, examples)". That is what the tree
    reported before;
  * `make pre-add-check FILES=launch/flags.go`: "1 file(s) clean";
  * the script tests: `go-modules_test` 17 passed, `go-fuzz_test` 12,
    `go-apicheck_test` 19, `go-examples_test` 19, and
    `go-precheck_test` 12;
  * markdownlint: 0 issues. The link check: 123 links, none broken.
* **The Windows test host,** go1.27.1 windows/amd64:
  * `go-precheck_test.sh` gave "12 passed, 0 failed", including the
    deleted-file case;
  * `make pre-add-check` gave "155 file(s) clean".
* **The identifier scan of the diff and the new test:** no match.
