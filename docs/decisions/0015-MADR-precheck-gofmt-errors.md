---
status: accepted
date: 2026-10-08
decision-makers: owner
consulted: 0013-PLAN-cli-integration-helpers.md (Step 6, the Windows run), 0014-PLAN-api-policy-gates.md (Step 5, deviation D5)
---
# Make the pre-add check fail when gofmt itself fails, and skip a tracked file the work tree no longer has

## Context and Problem Statement

`scripts/go-precheck.sh` is the one implementation of the pre-add rule.
`make pre-add-check` runs it, `make release-check` runs it over the whole
tree before a tag, and the machine-wide agent gate runs it before every
agent commit that stages Go files (`AGENTS.md`, "Pre-add checks").

Its first step is gofmt, at `scripts/go-precheck.sh:238`:

```bash
unformatted="$(gofmt -l "${mfiles[@]}")"
if [ -n "$unformatted" ]; then
```

It reads gofmt's standard output, the list of unformatted files, and not
gofmt's exit status. gofmt fails without printing a file name in two
cases. A probe on 2026-10-08 with the Go 1.27.1 toolchain's gofmt shows
both:

| `gofmt -l` on | Exit status | Standard output | Standard error |
| :--- | ---: | :--- | :--- |
| a formatted file | 0 | nothing | nothing |
| an unformatted file | 0 | the file's name | nothing |
| a missing file | 2 | nothing | `lstat missing.go: no such file or directory` |
| a file that does not parse | 2 | nothing | `broken.go:3:9: expected ')', found '{'` |
| an unformatted file and a missing one | 2 | the unformatted file's name | the `lstat` error |

So the step passes a file gofmt could not read or parse. Every other step
of the script checks its command's exit status.

The gap was seen twice:

* **On the Windows test host** (0013-PLAN-cli-integration-helpers.md,
  Step 6). The host's copy of the tree rebuilt its index from an earlier
  commit, which still tracked three test files deleted in the work tree.
  The no-list path takes its files from `git ls-files '*.go'`, which
  lists them. gofmt printed "GetFileAttributesEx launch/run_unix_test.go:
  The system cannot find the file specified", and the precheck reported
  every file clean.
* **On a scratch clone,** 2026-10-08 (H1). A Go file under `testdata`
  that does not parse, given in a file list with the examples gate
  skipped (`GO_PRECHECK_SKIP_EXAMPLES=1`). gofmt printed
  "testdata/frameworks/flag/broken.go:3:12: expected ')', found '{'", and
  the precheck printed "1 file(s) clean in 1 module(s)" and exited 0.
  * For an ordinary package, `go vet` and `go test` would fail on the
    same file.
  * But a file under `testdata` is not vetted or tested
    (0014-PLAN-api-policy-gates.md, deviation D5).
  * So only gofmt and the examples gate see it, and the examples gate
    can be skipped.

The two paths also treat a missing file differently. With a file list,
the script keeps an argument only if it is a regular file (`[ -f "$f" ]`),
so a missing file is never checked. With no list, every tracked Go file
goes to gofmt, whether or not the work tree still has it.

## Decision Drivers

* A check that cannot read its input must not pass it.
* The script stays one implementation for the three callers, with one
  meaning of "the files to check".
* No new tool, and no change to what a clean tree reports.
* The fix is proven by a test that fails on the old behaviour (`AGENTS.md`
  and the global rules: a new check is seen to fail).

## Considered Options

* Fail on gofmt's exit status, and skip a tracked file the work tree no
  longer has, as the file-list path already does
* Fail on gofmt's exit status alone, so a tracked file deleted from the
  work tree fails the no-list path
* Leave the step as it is

## Decision Outcome

Chosen option: "Fail on gofmt's exit status, and skip a tracked file the
work tree no longer has", because a file gofmt cannot parse is then
refused on both paths, and "the files to check" means the same thing on
both: the Go files the work tree has.

* **The gofmt step** runs `gofmt -l` with its standard error captured.
  * A non-zero exit status fails the step, with exit 1, and shows
    gofmt's standard error under a heading naming gofmt.
  * The list of unformatted files is reported as now. Both can appear in
    one run, as the table shows.
* **The no-list path** keeps a tracked Go file only if the work tree has
  it as a regular file, the same test the file-list path applies. A
  tracked file deleted but not yet staged is not a file to commit, so it
  is not checked.
* **A test,** `scripts/go-precheck_test.sh`, in the style of the other
  four script tests. It runs the precheck on throwaway repositories, with
  a stub golangci-lint and the network gates skipped. CI runs it.

### Consequences

* Good, because a Go file that does not parse fails the precheck on both
  paths, under `testdata` too, with gofmt's own message.
* Good, because the two paths agree on which files they check.
* Good, because a clean tree reports exactly what it reports today.
* Good, because the precheck gains a test, which none of its steps had.
* Bad, because a tracked file deleted from the work tree without
  `git rm` is now silently skipped by `make release-check`, where today
  gofmt prints an error that does not fail. `git status` still shows the
  deletion, and a commit of it goes through the file-list path.
* Neutral, because go-selfupdate-lib's `scripts/go-precheck.sh`, which
  this script was taken from, likely has the same gap. That is that
  repository's own decision.

### Confirmation

* `scripts/go-precheck_test.sh` passes, and fails when either half of the
  change is reverted on a scratch copy.
* `make release-check` and `make pre-add-check FILES=…` report a clean
  tree as before.
* On the Windows test host, the case that printed "GetFileAttributesEx …"
  no longer reaches gofmt.

## Pros and Cons of the Options

### Fail on gofmt's exit status, and skip a tracked file the work tree no longer has

* Good, because both holes close, and the paths agree.
* Bad, because an unstaged deletion is skipped rather than reported.

### Fail on gofmt's exit status alone

* Good, because it is the smallest change: one step.
* Good, because an unstaged deletion of a tracked Go file fails
  `make release-check` loudly.
* Bad, because the two paths then disagree: a missing file is ignored in a
  file list and fatal with no list.
* Bad, because a copy of the tree whose index lags its files fails until
  the index is fixed. The Windows test host's copy is one.

### Leave the step as it is

* Good, because it costs nothing.
* Bad, because a file that does not parse can pass the precheck and the
  agent's commit gate.

## More Information

* The probe and H1 are recorded in
  [0015-PLAN-precheck-gofmt-errors.md](0015-PLAN-precheck-gofmt-errors.md).
* gofmt's behaviour is that of the Go 1.27.1 toolchain this repository
  requires (`go.mod`).
