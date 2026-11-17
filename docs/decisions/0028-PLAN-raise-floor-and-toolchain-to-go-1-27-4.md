---
status: in-progress
date: 2026-11-17
associated-madr: "0028-MADR-raise-floor-and-toolchain-to-go-1-27-4.md"
---
# Implement the floor and toolchain at `go 1.27.4`

Associated MADR: [0028-MADR-raise-floor-and-toolchain-to-go-1-27-4.md](0028-MADR-raise-floor-and-toolchain-to-go-1-27-4.md)
(option A, as the owner ruled).

## Goal

* `go.mod` and `go.work` say `go 1.27.4`, with no `toolchain` line, and
  every live pin and current statement agrees.
* The gates pass, and `govulncheck ./...` under `go1.27.4` exits 0.
* A release is ready for the owner to tag. The plan stops before the tag.

## Scope

### In scope: the files and lines (numbers at `73d44fe`)

| File | Line | Change |
| :--- | :--- | :--- |
| `go.mod` | 3 | `go 1.27.2` becomes `go 1.27.4` |
| `go.work` | 1 | `go 1.27.2` becomes `go 1.27.4` |
| `lineedit/testdata/xtermparity/xterm.mod` | 6 | `go 1.27.2` becomes `go 1.27.4` (live: `go run -modfile`) |
| `testdata/frameworks/go.mod.tmpl` | 3 | `go 1.27.2` becomes `go 1.27.4` (live: the examples script's template) |
| `AGENTS.md` | 14 | "Requires Go 1.27.2" becomes "Requires Go 1.27.4" |
| `AGENTS.md` | 207 | "today 1.27.2 (0016-MADR...)" becomes "today 1.27.4 (0028-MADR...)", citing this record by filename |
| `README.md` | 179 | "Go 1.27.2" becomes "Go 1.27.4" |
| `docs/architecture.md` | 18 | "Go 1.27.2" becomes "Go 1.27.4" |
| `docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md` | after A2 | dated amendment A3, pointing here |
| `docs/decisions/0016-MADR-go-1-27-2-for-go-2026-6604.md` | end | new `## Amendments`, dated A1, pointing here |
| `docs/README.md` | records table | rows for this pair; status cells of 0004 and 0016 |

The records are staged by path; every other file is left unstaged.

### Out of scope

* The `go 1.27.1` literals in `scripts/*_test.py` (test data).
* Earlier records and reports, which keep their versions.
* `.github/workflows/*.yml`: CI reads `go-version-file: go.work`, so it
  follows with no edit (MADR E5). No floor job is added; none is needed.
* The consumers, `gobble-cli` and `magic-cli-remote` (read only).
* The tag and the release, until the owner's word.
* Any run under `GOTOOLCHAIN=go1.26.11`: there is no 1.26.11 floor here
  (MADR E4).
* `make fuzz`: `AGENTS.md` does not ask for it on every change, and no Go
  code changes.
* A `CHANGELOG.md` entry: the file does not exist yet
  (`0027-PLAN-keep-docs-accurate-and-current.md` starts it).

## Implementation Steps

1. **Preflight (read-only):** tree clean at `73d44fe`; `go mod tidy -diff`
   clean; `go version` is go1.27.4.
2. **Records:** this pair; the two amendments; the index. Check them.
3. **The change,** by one Python script with `assert` on each old
   string's count before replacing; each result read back.
4. **`go mod tidy -diff`** shows nothing; `git diff --stat` names only the
   files in the table.
5. **The version-agreement check,** a scratchpad script, on the tree and
   on three planted copies.
6. **Gates** (below).
7. **Staging:** the record files only, by path.

## Verification

* The agreement check exits 0 on the tree, and non-zero on a wrong `go`
  line in `go.mod`, a wrong `go` line in `go.work`, and a stray old pin
  in one live pin file.
* `go mod tidy -diff`, `make tidy-check`: exit 0, nothing moves.
* `make fmt`-equivalent, `make vet`, `make lint`, `make test`,
  `make apicheck`, `make examples`, `make release-check`, `make vuln` (or
  `govulncheck ./...` under `GOTOOLCHAIN=go1.27.4 GOPROXY=off`): exit 0.
* Cross builds for `linux` and `windows` with `CGO_ENABLED=0`.
* After the last record edit: `make check-records`, `make markdownlint`,
  `make docs-check`, `make disclosure-check`.
* `git grep -n '1\.27\.2'` finds only records, reports and the
  `scripts/*_test.py` literals' neighbours (they say 1.27.1).

## Rollout and Rollback

* Before a tag, revert the change with a new commit; no history is
  rewritten.
* After a tag, a fault is fixed in a new release.
* A consumer is affected only when it takes the new release; the two
  known consumers already declare `go 1.27.4` (MADR E6).
* **The release needs the owner's word.** `scripts/check-release-tag.sh`
  and the disclosure guard come first; this plan does not run them.

## Execution Record

### Start (2026-11-17)

`main` at `73d44fe`, clean, level with its upstream (no fetch).
`scripts/check_records.py --next` printed `0028`.

The `toolchain` experiment (MADR E3) ran on a scratch copy; the real tree
was not touched.

### The change (2026-11-17)

Staged: the record files only. The change files are unstaged, for the
coordinator to commit separately. One script, with `assert` on each old
string's count, made the edits and each result was read back from
`git diff`: `go.mod:3`, `go.work:1`,
`lineedit/testdata/xtermparity/xterm.mod:6` and
`testdata/frameworks/go.mod.tmpl:3` (each `go 1.27.2` to `go 1.27.4`);
`AGENTS.md:14` and `:207`, `README.md:179` and `docs/architecture.md:18`.
No `toolchain` line was added. `go mod tidy -diff` exits 0 and `go.sum` is
unchanged. History stays: the `go 1.27.1` literals in `scripts/*_test.py`
and `command/sources_test.go:239` (test data), the records and reports, and
the index rows that carry earlier records' titles.

### The version-agreement check

A scratchpad script, not added to the tree, reads the `go` line of
`go.mod` and compares `go.work`, both live pins, the four statements and
the CI workflows' `go-version-file` (each must be `go.work`), and sweeps
the tree for any other `1.27.y` outside the records, reports, test files
and index rows. Exit 0 on the tree ("0 failure(s)"). Seen to fail on
scratch copies of the changed tree:

* wrong `go` line in `go.mod` (`go 1.27.3`): exit 1, 14 failures, the
  first "`go.work`: go line 1.27.4, want 1.27.3";
* wrong `go` line in `go.work` (`go 1.27.2`): exit 1, "`go.work`: go line
  1.27.2, want 1.27.4";
* a stray old pin in `lineedit/testdata/xtermparity/xterm.mod` (`go
  1.27.2`): exit 1, "`lineedit/testdata/xtermparity/xterm.mod`: go line
  1.27.2, want 1.27.4".

### Gates (this Mac, Go 1.27.4; UTC; exit status taken before any filter)

| Gate | Start | End | Exit | Result |
| :--- | :--- | :--- | :--- | :--- |
| `go mod tidy -diff` | 19:35:35 | 19:35:35 | 0 | nothing moves |
| `make tidy-check` | 19:35:35 | 19:35:35 | 0 | |
| `gofmt -l .` empty | 19:35:35 | 19:35:36 | 0 | |
| `make vet` | 19:35:36 | 19:35:40 | 0 | |
| `govulncheck ./...`, `GOTOOLCHAIN=go1.27.4 GOPROXY=off` | 19:35:40 | 19:35:41 | 0 | "No vulnerabilities found." |
| `make lint` (`run-heavy`) | 19:38:02 | 19:39:20 | 0 | 0 issues (darwin, windows) |
| `make test` (`run-heavy`) | 19:39:57 | 19:41:47 | 0 | |
| `make check-records` | 19:38:51 | 19:38:52 | 0 | 61 records, 0 errors |
| `make markdownlint` | 19:38:55 | 19:38:59 | 0 | 0 issues |
| `make disclosure-check` | 19:38:52 | 19:38:55 | 0 | 0 findings |
| `make release-check`, first run, `GOPROXY=off` exported in error | 19:50:53 | 19:51:23 | 2 | `apicheck` could not install `apidiff` offline: my environment, not the tree; see below |
| `make release-check`, second run, default `GOPROXY` (`run-heavy`) | 20:14:01 | 20:16:10 | 0 | 516 files clean: gofmt, golangci-lint, go vet, go test, go mod tidy, govulncheck, apicheck, examples |
| `make docs-check` | 19:51:23 | 19:51:35 | 0 | |
| `GOOS=linux CGO_ENABLED=0 go build ./...` | 19:51:35 | 19:51:36 | 0 | |
| `GOOS=windows CGO_ENABLED=0 go build ./...` | 19:51:36 | 19:51:36 | 0 | |
| `govulncheck ./...`, `GOTOOLCHAIN=go1.27.4 GOPROXY=off`, after the change | 19:51:36 | 19:51:37 | 0 | "No vulnerabilities found." |

The first `release-check` failure was a deviation of mine, not of the
plan: I had exported `GOPROXY=off` for the whole shell, but the brief
confines it to `govulncheck`; the gate's `apicheck` step installs
`apidiff`, which needs the proxy. I removed the export and ran it once
more; it passed. The tree was not changed between the two runs.

Not done, and why: `make fuzz` (`AGENTS.md` does not ask for it on every
change, and no Go code changed); every run under `GOTOOLCHAIN=go1.26.11`
(there is no 1.26.11 floor, MADR E4); a race run (no Go code changed); a
CI run (nothing is pushed); the tag and the release (the owner's word).
Records edited after the markdownlint, `check-records` and disclosure runs
above: this execution record; those three checks were re-run after it.

## Amendment A1 (2026-11-17): the release phase, `v2.12.1`

The owner's word, relayed by the coordinator, verbatim: "you may tag and
push go-tui-lib v2.12.1 once its CI is green" (MADR A1). The change is
committed and pushed (`735a3f7`, `e35e571`), and CI and Docs on `e35e571`
passed. The steps are those of `docs/guides/releasing.md`, "Release the
root alone", with 0025-PLAN Step 6's release (`v2.12.0`) and 0023-PLAN's
`v2.6.1` and `v2.6.2` as the model for a patch. No nested module is
touched: the module table in `AGENTS.md` lists `stream/glamourmd` as
planned, and it has no directory.

**Pointers.** Re-derived by `grep -rn 'v2\.12\.0'` at `e35e571`:

| Where | Class | Action |
| :--- | :--- | :--- |
| `README.md:21` "The current release is `v2.12.0`" | current-release pointer | moves to `v2.12.1` |
| `docs/guides/migrating.md:10-21` "Migrating to `v2.12.0`", with `go get …@v2.12.0` | history of that release | stays; a "Migrating to `v2.12.1`" section is added above it |
| `docs/guides/streaming-content.md:47` "Since `v2.12.0`" | history (when `RenderParts` arrived) | stays |
| `README.md:170` "Since `v2.12.0`, `Doc.RenderParts`" | history | stays |
| `docs/README.md:68`, `:71`; records 0025, 0026, 0027 | records | stay (0026-PLAN's "`v2.12.0` is the latest tag" is the record's claim at its date; 0026-MADR names that a release in between changes the base) |

There is no `CHANGELOG.md` (0027-PLAN starts it); none is started here.

**Steps.**

1. **The pre-tag commit** (Markdown only): `README.md:21` and the
   "Migrating to `v2.12.1`" section (the floor is Go 1.27.4; GO-2026-6859
   is reported under 1.27.3 and not 1.27.4; no API change; a program on
   1.27.2 or 1.27.3 moves to 1.27.4). `make release-check`, `make
   docs-check`, `make check-records`, `make disclosure-check`, `make
   markdownlint` pass.
2. **The disclosure guard** over the outgoing commit; the push. The push
   changes only documents, so Docs runs and passes; the code proof is the
   full CI run on `e35e571`, the last commit that changed anything else
   (`docs/guides/releasing.md`, 0019-MADR).
3. **The tag,** annotated, with the tag name as its message, on the
   pre-tag commit Docs passed on: `git tag -a v2.12.1 -m "v2.12.1"
   <commit>`, then `git push origin v2.12.1`. There is no tag-check script
   in this repository and the tags are unsigned. Never moved or deleted.
4. **CI on the tag** runs in full (a tag is never path-filtered,
   `.github/workflows/ci.yml:14`) and passes. No workflow publishes
   anything.
5. **The consumer smoke test from the proxy:** a scratch module outside
   the repository, `GOWORK` empty, `go get
   github.com/maccavelli/go-tui-lib/v2@v2.12.1`, a program importing
   `workspace` (a package with outside dependencies), `go mod tidy`, `go
   vet`, `go build`, run; `go list -m -json …@v2.12.1` gives the tag's
   commit, and the proxy's `go.mod` for it says `go 1.27.4` (the control:
   `v2.12.0`'s says `go 1.27.2`).
6. **The close-out:** this PLAN becomes `complete` with the tag, the CI
   run and the smoke test recorded, and its index row follows. There is no
   pin commit: `v2.12.0` moved its pointers in the tagged commit, and so
   does this release.

Not done by the author of this amendment: the commit, the push, the tag,
steps 2 to 6.
