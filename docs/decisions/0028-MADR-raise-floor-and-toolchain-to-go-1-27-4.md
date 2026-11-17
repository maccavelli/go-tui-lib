---
status: accepted
date: 2026-11-17
decision-makers: the repository owner
consulted: "a read-only survey of the fleet's Go pins (2026-11-17), relayed by the coordinator and re-derived here at 73d44fe; govulncheck v1.9.0 (vulnerability database updated 2026-11-11), measured 2026-11-17 for the fleet and re-run for this repository in the PLAN; the owner's rulings of 2026-11-17, relayed by the coordinator; go-selfupdate-lib's 0045-MADR-move-toolchain-to-go-1-27-4-and-floor-to-1-26-11.md, read as the model"
informed: gobble-cli and magic-cli-remote, the two repositories that require this module
---
# Raise the floor to `go 1.27.4`, with no `toolchain` line, because this module's code needs Go 1.27

## Context and Problem Statement

The owner's instruction of 2026-11-17, relayed by the coordinator and
quoted verbatim for the list it names:

> "go-llmprovider-sdk, go-tui-lib, acp-go-sdk2, gobble-cli, magic-cli-remote, go-selfupdate-lib, mcp-server-duckduckgo, mcp-server-magictools, mcp-server-recall, mcp-server-sequential-thinking, mcplib, mcp-buntdb, and mcp-server-socratic-thinker all need to have their go floor pinned to 1.26.11, and go version/toolchain pinned to 1.27.4."

"Floor" is the `go` directive in `go.mod`; "version/toolchain" is the
`toolchain` line. A read-only survey then found that four of the thirteen
use Go 1.27 features and cannot build under 1.26.11, and go-tui-lib is
one of them. The owner ruled on that finding, label quoted exactly and
only as a label: "Their floor is 1.27.4: raise go-tui-lib, leave the
other three (Recommended)". As put to the owner: for code that needs
Go 1.27 the floor and the toolchain are the same version; go-tui-lib
moves from `go 1.27.2` to `go 1.27.4` under its own record and release;
the other nine get floor 1.26.11 and toolchain 1.27.4.

**The owner asked for "version/toolchain pinned to 1.27.4".** Here the
`go` line is 1.27.4, so the `go` line itself pins that version: a `go`
line is a floor, a `toolchain` line equal to it adds nothing (E3), and
this repository's convention is a `go` line with no `toolchain` line
(E2). This record states the pin as the `go` line and gives the reason
for the form chosen in E3.

Why 1.27.4: GO-2026-6859, in `net/http`, is reported by `govulncheck`
v1.9.0 (database of 2026-11-11) under Go 1.27.3 and 1.26.10 and not under
1.27.4 and 1.26.11, measured 2026-11-17 for the fleet. The advisory is
not what forces this repository's floor to a patch of 1.27: the code
does (E4); the advisory is why the patch is 1.27.4.

Everything in this record other than the quoted statement and the quoted
label is evidence or the author's text.

**Evidence index.** Each item was read or observed on 2026-11-17 at
`main`, HEAD `73d44fe`, with a clean tree level with its upstream, unless
it says otherwise.

* **E1, the pins [read].**
  * `go.mod:3` is `go 1.27.2`; `go.work:1` is `go 1.27.2`. There is no
    `toolchain` line in either (`grep -n toolchain go.mod go.work` finds
    nothing).
  * Two live pins in files that are built as modules:
    `lineedit/testdata/xtermparity/xterm.mod:6` (`go 1.27.2`; used by
    `go run -modfile`, `lineedit/testdata/xtermparity/main.go:9`) and
    `testdata/frameworks/go.mod.tmpl:3` (`go 1.27.2`; the template
    `scripts/go-examples.py:122` writes into a temporary module).
  * Current statements: `AGENTS.md:14` ("Requires Go 1.27.2"), `AGENTS.md:207`
    (API convention 11, "today 1.27.2"), `README.md:179` ("Go 1.27.2 is
    required") and `docs/architecture.md:18` ("Go 1.27.2, with no
    `toolchain` line").
  * There are no other `go.mod` or `go.work` files in the tree (`find`),
    and no `SECURITY.md`, `CHANGELOG.md` or Makefile variable that names a
    version (`CHANGELOG.md` is 0027-PLAN's, not yet written).
* **E2, the repository's convention [read].** `docs/architecture.md:18`
  says "no `toolchain` line", and
  `0016-MADR-go-1-27-2-for-go-2026-6604.md` (Decision Outcome, option A)
  chose a `go` line with no `toolchain` line over option B (a `toolchain`
  line beside a lower `go` line), at that time to keep one version in the
  files CI reads.
* **E3, what `go mod tidy` does with a redundant `toolchain` line
  [observed in a scratch copy].** `git archive HEAD` into a scratch
  folder; `go` lines set to 1.27.4 in `go.mod` and `go.work`; `toolchain
  go1.27.4` appended to `go.mod`; then `GOTOOLCHAIN=go1.27.4 GOPROXY=off
  GOWORK=off go mod tidy` exited 0 and the `toolchain` line was gone
  (`grep -n toolchain go.mod` found nothing). `make tidy-check` runs
  `go mod tidy -diff` per module (`Makefile:90-91`), so the repository's
  own gate would fail on the line. No existing record requires one: E2's
  record chose the opposite. So the form is `go 1.27.4`, no `toolchain`
  line.
* **E4, why the floor is 1.27 [re-derived by `grep`, 2026-11-17].**
  * 45 Go files import `encoding/json/v2` or `encoding/json/jsontext`
    (27 import `encoding/json/v2`, 30 import `encoding/json/jsontext`,
    some both). The survey found that these do not build under
    Go 1.26.11 [from the survey's measurement; not re-run here].
  * `termcap/queries.go:62` builds a struct literal that sets a field
    promoted from an embedded struct, which the survey found needs Go
    1.27 [from the survey; line exists at HEAD; not re-run here].
  * `docs/reports/0014-REPORT-api-assessment-and-integration-research.md:154`
    says the same of the 1.27.1 floor: "JSON v2 and the generic method
    `PaneAs` (`workspace/workspace.go:592`) need it".
  * So there is no 1.26.11 floor for this module, and the PLAN runs
    nothing under `GOTOOLCHAIN=go1.26.11`: the survey's 1.26.11 build of
    this module fails, and the owner ruled the floor to be 1.27.4.
* **E5, CI [read].** `ci.yml:43`, `:73` and `:114` and `docs.yml:39` use
  `actions/setup-go` pinned to v7.0.0
  (`b7ad1dad31e06c5925ef5d2fc7ad053ef454303e`) with `go-version-file:
  go.work`. The setup-go README at that pin says: "If the `toolchain`
  directive is present, its version is used; otherwise, the action falls
  back to the `go` directive" (as quoted in go-selfupdate-lib's
  0018-MADR; not re-read here, and about `go.mod`, which `go.work`
  shares). `go.work` has no `toolchain` line, so after this change CI
  installs the `go` line's 1.27.4 on every job. No job builds on a lower
  Go: CI has **no floor job**, and none is needed, because the floor and
  the toolchain are the same version. That is not a gap.
* **E6, consumers [read, read-only].** `gobble-cli/go.mod:20` and
  `magic-cli-remote/go.mod:14` require
  `github.com/maccavelli/go-tui-lib/v2 v2.12.0`; both declare `go 1.27.4`
  (`go.mod:3` in each). The latest tag here is `v2.12.0`. No other
  repository the survey scanned requires this module.
* **E7, earlier Go-version records [read].**
  `0004-MADR-integrate-charm-v2-and-go-1-27.md` (Go 1.27.1, with its
  PLAN) and `0016-MADR-go-1-27-2-for-go-2026-6604.md` (Go 1.27.2, with
  its PLAN). `AGENTS.md:206-209` says the `go` line "moves only by a
  record" and "to the newest patch of a release Go still supports".

The question: what should this module's floor and toolchain be, given the
owner's instruction, the survey and E1 to E7?

## Decision Drivers

* The owner's rulings above: floor 1.27.4 for code that needs Go 1.27,
  floor and toolchain the same version.
* API convention 11 (`AGENTS.md:206-209`): the floor moves only by a
  record, to the newest patch of a supported release.
* The code needs Go 1.27 (E4), so 1.26.11 is not available as a floor.
* One source of truth for the version in the files CI reads (0016's
  driver), and no line `go mod tidy` removes (E3).
* No compatibility engineering: the owner is the only user of the fleet's
  repositories.

## Considered Options

* **A. `go 1.27.4` in `go.mod` and `go.work`, no `toolchain` line, and
  every live pin and current statement follows** (the owner's ruling).
* **B. `go 1.27.4` with `toolchain go1.27.4`.**
* **C. A floor of `go 1.26.11` with `toolchain go1.27.4`**, as the
  instruction first named for all thirteen.
* **D. Change nothing** (`go 1.27.2`).

## Decision Outcome

Chosen option: **"A"**, the owner's ruling, because the code needs Go 1.27
(E4), because a `go` line of 1.27.4 is the pin the owner asked for (a
`toolchain` line equal to it would be dropped by `go mod tidy`, E3, against
the repository's convention, E2), and because 1.27.4 is free of
GO-2026-6859.

1. **`go.mod:3` and `go.work:1`:** `go 1.27.2` becomes `go 1.27.4`. No
   `toolchain` line is added. `go mod tidy -diff` stays clean and no
   `require` or `go.sum` line moves.
2. **The live pins:** `lineedit/testdata/xtermparity/xterm.mod:6` and
   `testdata/frameworks/go.mod.tmpl:3` become `go 1.27.4`.
3. **The current statements:** `AGENTS.md:14` and `:207`, `README.md:179`
   and `docs/architecture.md:18` say 1.27.4. `AGENTS.md:207` cites this
   record by filename.
4. **CI is not edited.** Its four `setup-go` steps read `go.work` and
   install 1.27.4 (E5).
5. **History stays:** the `go 1.27.1` literals in `scripts/*_test.py`,
   which are test data; released changelog entries (there are none yet);
   and every earlier record and report. 0004-MADR and 0016-MADR each gain
   a dated amendment pointing here, and rewrite nothing.
6. **A release follows** (below). The PLAN stops before the tag.

### Consequences

* Good, because the floor is the version the code needs, and the same one
  CI, the gates and `govulncheck` use, with GO-2026-6859 absent.
* Good, because no `toolchain` line means nothing for `go mod tidy` to
  strip and one number to track.
* Bad, because a consumer must use Go 1.27.4 or later to build the next
  release. Who that touches (E6): **gobble-cli** and **magic-cli-remote**
  already declare `go 1.27.4`, so raising this floor breaks neither, and
  neither has to change its `go.mod`; each takes the new floor by moving
  its `require` to the new release, in its own record. Nothing else in
  the fleet's survey requires this module.
* Bad, because **a release is needed** before any consumer can pin a
  version whose floor excludes the affected toolchains: `v2.12.0` still
  declares `go 1.27.2`. The tag and release need the owner's word, and
  the PLAN stops before them. The next number is the owner's: the change
  adds no exported API, so a patch release (`v2.12.1`) is the natural
  one, but that is not decided here.
* Neutral, because CI has no floor job (E5), and none is wanted.
* Neutral, because `0027-PLAN-keep-docs-accurate-and-current.md` is in
  progress and will add `CHANGELOG.md`; this change has no changelog
  entry to make until that file exists, and the release's notes name the
  floor.

### Confirmation

* `go mod tidy -diff` and `make tidy-check` are clean.
* A version-agreement check, run from a scratch folder and not added to
  the tree, reads the `go` line of `go.mod` and asserts that `go.work`,
  both live pins and every current statement agree. It must pass on the
  changed tree and fail on three planted breaks in scratch copies: a
  wrong `go` line in `go.mod`, a wrong `go` line in `go.work`, and a stray
  old pin in one live pin file.
* `govulncheck ./...` under `GOTOOLCHAIN=go1.27.4 GOPROXY=off` exits 0.
* The repository's gates named in the PLAN pass.

## Pros and Cons of the Options

### A. `go 1.27.4`, no `toolchain` line (chosen)

* Good, because it is the owner's ruling and follows E2, E3 and API
  convention 11.
* Bad, because it raises every consumer's minimum by two patches (E6
  shows none is affected).

### B. `go 1.27.4` with `toolchain go1.27.4`

* Good, because it states the toolchain in the form the instruction used.
* Bad, because `go mod tidy` removes the line (E3), so `make tidy-check`
  fails, and the line adds no information.

### C. Floor `go 1.26.11`, `toolchain go1.27.4`

* Good, because it is what the instruction first named for every
  repository, and it lowers consumers' minimum.
* Bad, because the module does not build under 1.26.11 (E4), so the floor
  would be false, and a floor job could not pass.

### D. Change nothing

* Good, because nothing is tagged.
* Bad, because the owner ruled otherwise, and 1.27.2 is below the
  advisory-free 1.27.4 the fleet is moving to.

## More Information

### Decided

The owner's ruling of 2026-11-17, relayed by the coordinator. The label
is quoted exactly and only as a label: **"Their floor is 1.27.4: raise
go-tui-lib, leave the other three (Recommended)"**. Asked: four of the
thirteen cannot build under 1.26.11; the chosen option raises go-tui-lib
to floor 1.27.4 under its own record and release and leaves the other
three, and the other nine get floor 1.26.11 and toolchain 1.27.4. The
form of the pin (no `toolchain` line, E3) and the amendments to 0004 and
0016 are the author's application of that ruling, not separate rulings.

### Relation to earlier records

* [0016-MADR-go-1-27-2-for-go-2026-6604.md](0016-MADR-go-1-27-2-for-go-2026-6604.md)
  set the floor at 1.27.2. This record raises it and keeps 0016's form (a
  `go` line, no `toolchain` line). It amends 0016, dated, and supersedes
  nothing.
* [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
  adopted Go 1.27 idioms at a 1.27.1 floor. It gains a dated amendment.

### Not verified

* That the 45 files and `termcap/queries.go:62` fail under Go 1.26.11:
  the survey measured it; this record ran nothing under 1.26.11 because
  the floor is 1.27.4.
* What `actions/setup-go` v7.0.0 installs on a runner for a `go.work`
  with no `toolchain` line: the README text is quoted second-hand and no
  run has shown it.
* Whether the CI `govulncheck` (pinned older than v1.9.0) reports
  GO-2026-6859: only v1.9.0 was run.
* The release number and date, which are the owner's.

## Amendments

### A1 (2026-11-17): the owner's word for the release, and the number `v2.12.1`

*Status: accepted (2026-11-17).* The change of this record is committed
and pushed (`735a3f7` the pair, `e35e571` the change). The owner's word
for the release, relayed by the coordinator, verbatim: "you may tag and
push go-tui-lib v2.12.1 once its CI is green". The tag is the
coordinator's to make and push, not this record's author's.

* **The number is a patch,** because no exported API changes. Evidence at
  `e35e571`: `git diff --stat v2.12.0..HEAD` lists, among non-Markdown
  files, only `go.mod`, `go.work`,
  `lineedit/testdata/xtermparity/xterm.mod` and
  `testdata/frameworks/go.mod.tmpl`, each a one-line `go` change; and
  `make apicheck` prints "apicheck: .: against v2.12.0, 0 incompatible
  change(s)" and "apicheck: clean". The six commits since `v2.12.0` are
  all documents but `e35e571`: `855aa72`, `f093da6`, `fc55005`, `73d44fe`,
  `735a3f7` (Markdown) and `e35e571` (the pins). A patch has precedent
  here: `v2.6.1` and `v2.6.2`, each with a "Migrating to" section saying
  nothing exported changes.
* **CI on `e35e571` passed** (run 44167880653: `modules`, `gates` and the
  three `test` entries, on Ubuntu 24.04, Windows 2025 and macOS 15), and
  Docs passed on it. Its `setup-go` lines read `go-version-file: go.work`,
  "Acquiring 1.27.4 from https://github.com/actions/go-versions/…",
  "Successfully set up Go version 1.27.4" and `go version go1.27.4`. That
  answers the second item of "Not verified": CI installs the `go` line's
  1.27.4. The `gates` job printed "0 issues." for the linters and "No
  vulnerabilities found." for `govulncheck`. The Docs run on `735a3f7` is
  `cancelled`, as `docs.yml:22-24` says it would be: its `concurrency`
  group `docs-${{ github.ref }}` has `cancel-in-progress: true`, and the
  next push (`e35e571`, created 11 seconds later) superseded it.
* **The release carries the pins and the documents,** not code: all six
  commits above ship in the tag. The PLAN's release phase says what is
  done and in what order. The release procedure moves the current-release
  pointers before the tag, in the commit that is tagged, as `v2.12.0`
  did.

### A2 (2026-11-17): the release followed

*Status: accepted (2026-11-17).* `v2.12.1` was tagged on `6ac8355` (an
annotated, unsigned tag, object `05583ac942df089f4369842625cdca092f2b6199`)
on the owner's word quoted in A1. CI on the tag passed on Go 1.27.4,
`govulncheck` found nothing and `apicheck` reported 0 incompatible
changes against `v2.12.0`. The proxy serves `v2.12.1` with `go 1.27.4`,
where `v2.12.0` has `go 1.27.2`, and a consumer's smoke test from the proxy
passed. The "Not verified" item on the release number is settled by the
owner's word; the item on `setup-go` by A1's CI quotes. The execution
record is in
[0028-PLAN-raise-floor-and-toolchain-to-go-1-27-4.md](0028-PLAN-raise-floor-and-toolchain-to-go-1-27-4.md).

### Links

* [GO-2026-6859](https://pkg.go.dev/vuln/GO-2026-6859).
* [Go downloads](https://go.dev/dl/): `go1.27.4`.
* [0028-PLAN-raise-floor-and-toolchain-to-go-1-27-4.md](0028-PLAN-raise-floor-and-toolchain-to-go-1-27-4.md).
