---
status: proposed
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

Not started.
