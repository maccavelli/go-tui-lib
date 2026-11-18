---
status: in-progress
date: 2026-11-18
associated-madr: "0029-MADR-move-golangci-lint-to-v2-15-0.md"
---
# Implement the golangci-lint v2.15.0 move

Associated MADR: [0029-MADR-move-golangci-lint-to-v2-15-0.md](0029-MADR-move-golangci-lint-to-v2-15-0.md)
(option A). The owner approved this scope in advance (2026-11-18).

## Goal

golangci-lint v2.15.0 is the pin everywhere it is stated, and `make lint`
reports 0 issues with it on linux, darwin and windows.

## Scope

| File | Change |
| --- | --- |
| `scripts/go-precheck.py` | install hint `@v2.14.0` becomes `@v2.15.0` |
| `.github/workflows/ci.yml` | `go install ...@v2.14.0` becomes `@v2.15.0` |
| `docs/architecture.md` | "golangci-lint v2.14.0" becomes "v2.15.0" |
| `Makefile` | install hint `@latest` becomes `@v2.15.0` |
| `termcap/termcaptest/termcaptest.go` | `reapKilled`, used on both timeout paths |
| `termcap/termcaptest/termcaptest_test.go` | a test of `reapKilled` |
| `docs/decisions/0001-MADR-...` | one dated line pointing to the MADR |
| `docs/README.md` | index rows for 0029 |

History stays: every record and report that names v2.14.0.

## Implementation Steps

1. Record the red: v2.15.0 on `git archive HEAD`, three GOOS.
2. Write the test of `reapKilled`; see it fail against a discarding stub.
3. Add `reapKilled`; replace both `<-done`.
4. Move the pins; add the index rows and the 0001 line.
5. Run the gates.

## Verification

`make lint` (with the v2.15.0 binary), gofmt, `go vet`, the precheck
wrapper, `go mod tidy -diff`, the full test suite and `-race` for the
package, `make check-records`, `make markdownlint`, `actionlint`.

## Rollout and Rollback

Commit and push by the coordinator; no tag. Rollback is a revert.

## Execution record

2026-11-18, steps 1 to 5 done, uncommitted at the time of writing:

* Red: v2.15.0 on `git archive HEAD`, linux, darwin and windows, each
  exit 1 with `termcap/termcaptest/termcaptest.go:267:3` and `:278:3`
  (errcheck) and nothing else.
* `TestReapKilled` failed against a discarding stub ("unexpected: reported
  [], want [...]"), then passed with `reapKilled`.
* Green: `make lint` with v2.15.0 printed "0 issues." for each of the
  three GOOS (exit 0); `make vet`, `make tidy-check`, `make check-records`,
  `make markdownlint`, `make disclosure-check`, `make docs-check`,
  `actionlint` and `scripts/go-precheck.sh` on the two changed Go files
  exit 0; `go test -count=1 ./...` and `go test -race -count=1 ./...`
  exit 0 (through `run-heavy`).
* Not done: no `CHANGELOG.md` entry (the file does not exist), no tag, no
  version-agreement script (the repository has none).
