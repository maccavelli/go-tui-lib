---
status: accepted
date: 2026-11-18
decision-makers: the repository owner
consulted: "a read-only survey of the fleet's lint pins (2026-11-18), relayed by the coordinator and re-derived here at 8e019bc; golangci-lint v2.15.0 built with go1.27.4, run on a `git archive HEAD` copy; the Go module proxy's bubbletea source for `ErrProgramKilled`"
informed: gobble-cli and magic-cli-remote, the two repositories that require this module
---
# Move golangci-lint to v2.15.0, and fix the two errcheck findings in `termcaptest.Run`

## Context and Problem Statement

The owner's ruling of 2026-11-18, quoted verbatim: "standardize on
golangci-lint v2.15.0". Asked how the moves proceed in the repositories
still pinned lower, the owner chose, label quoted exactly and only as a
label: "One short record pair per repository, approved now through CI
(Recommended)". As put to the owner: each repository gets a record, the
pin moved everywhere it appears, the findings fixed in the same change,
then commit and push, with no release; the owner is asked again only if a
finding needs a real decision.

Everything else here is evidence or the author's text. Read at `main`,
`8e019bc`, clean tree:

* **E1, the pin.** golangci-lint v2.14.0 appears in `scripts/go-precheck.py:338`
  (the install hint), `.github/workflows/ci.yml:158` and
  `docs/architecture.md:522`. `Makefile:69` says `@latest`, which names no
  version. The records that mention v2.14.0 as an observation are left alone.
* **E2, which record decided it.** `0001-MADR-scaffold-charm-tui-library.md`
  chose "golangci-lint v2.14.0" for the scaffold's CI (its Decision
  Outcome and Confirmation). `0004-MADR-integrate-charm-v2-and-go-1-27.md`
  only reports that v2.14.0 ran clean over a probe; it decided nothing
  about the pin. This record supersedes 0001's decision for the version
  only.
* **E3, modules.** There is one Go module: `find` lists one `go.mod`, `go.work`
  says `use .`, and `scripts/go-modules.py` prints `.`. `make lint` lints
  it for linux, darwin and windows with `GOWORK=off`, and CI runs `make
  lint`. No module goes unlinted. `testdata/frameworks` builds in a
  temporary module that is not a lint target by design (`AGENTS.md`).
* **E4, the red.** v2.15.0 on a `git archive HEAD` copy, for each of linux,
  darwin and windows, reports the same two findings (v2.14.0 reports none):
  `termcap/termcaptest/termcaptest.go:267:3` and `:278:3`, "Error return
  value is not checked (errcheck)", both on `<-done`.
* **E5, what the findings are.** In `(*Terminal).Run`, `done` receives
  `prog.Run()`'s error. On a timeout the helper calls `prog.Kill()` and
  waits on `done`, discarding the error, and then `tb.Fatalf` reports the
  timeout. bubbletea's `Run` returns an error wrapping `tea.ErrProgramKilled`
  when killed (`tea.go`: `ErrProgramKilled`, wrapped with the cause for a
  panic or a context error). That is the expected value. Any other error
  (a failure the program hit by itself in the instant before the kill) is
  currently dropped, though the helper's doc says it "fails tb when ... the
  program fails".

## Considered Options

* **A. Move to v2.15.0, and fix the findings** by reporting an unexpected
  error from the program on the timeout path and ignoring only the expected
  kill error.
* **B. Move to v2.15.0 and discard the values** (`_ = <-done`), or add
  `//nolint` or a config exclusion.
* **C. Stay on v2.14.0.**

## Decision Outcome

Chosen option: **"A"**, the owner's ruling plus a fix that is right: the
kill error is expected and discarded on purpose, and anything else is
reported to the test. A shared helper, `reapKilled`, does it for both
timeout paths.

1. The pin is v2.15.0 in `scripts/go-precheck.py`, `.github/workflows/ci.yml`,
   `docs/architecture.md`, and `Makefile` (whose hint now names it).
2. `reapKilled(tb, name, done)` waits for `done` and calls `tb.Errorf` when
   the error is non-nil and does not wrap `tea.ErrProgramKilled`. `Run`'s
   two timeout paths call it before their `tb.Fatalf`. A test failing on a
   timeout fails as before with the same message; the only change is an
   extra line when the program also failed for another reason, so the test
   sees more, never less. No exported name or passing test's outcome changes.
3. `0001-MADR` gains one dated line pointing here. 0004 is not amended.

### Consequences

* Good, because the fleet standard holds here and the helper's documented
  promise (fail on a program failure) holds on the timeout path too.
* Good, because no `//nolint`, no exclusion and no blanket discard.
* Neutral, because there is no `CHANGELOG.md` in the repository (0027-PLAN
  starts one, not yet written), so there is no Unreleased entry to make;
  the change is not a release and no tag is cut.
* Neutral, because there is no version-agreement script in the tree: 0028's
  was a scratchpad script, not committed. None is invented here.

### Confirmation

* v2.15.0 on `git archive HEAD` reports the two findings (red); on the
  changed tree it reports 0 issues for linux, darwin and windows.
* A new test of `reapKilled` fails when the helper is replaced by a
  discard, on a scratch copy.
* The gates in the PLAN pass.

## Pros and Cons of the Options

### A. Move and fix (chosen)

* Good, because it is the owner's ruling and the fix surfaces real errors.
* Bad, because the helper gains an unexported function.

### B. Move and discard or suppress

* Good, because it is the smallest diff.
* Bad, because it hides a program failure during a timeout, and a new
  `//nolint` is refused by the ruling's scope.

### C. Stay

* Bad, because the owner ruled otherwise.

## More Information

### Decided

The owner's two rulings above. The shape of the `termcaptest` fix is the
author's application of them; it is not a choice about behaviour, because
nothing a passing or timing-out test saw before is removed.

### Not verified

* The CI run of the changed workflow, which the coordinator pushes.
* That v2.15.0 reports nothing under another GOOS than the three linted.

### Links

* [0029-PLAN-move-golangci-lint-to-v2-15-0.md](0029-PLAN-move-golangci-lint-to-v2-15-0.md)
* [0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)
