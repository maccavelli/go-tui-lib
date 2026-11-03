---
status: accepted
date: 2026-10-30
decision-makers: the repository owner
---
# Align docs-only CI and disclosure gates

## Context and Problem Statement

This repository needs a focused Docs run for documentation-only pushes and pull requests. The earlier local decision is 0019-MADR-skip-full-ci-for-docs-only-pushes.md. This new decision adds a common quality and disclosure contract across the seven repositories.

Observed in this checkout on 2026-10-25: `ci.yml` ignores only root Markdown, `docs/**/*.md` and three agent-rule trees; command test fixtures remain on full CI. `docs.yml` runs Markdownlint and `scripts/go-examples.py --check-guide`. There is no CI record/link gate or portable disclosure scan. The prior PLAN is still in progress.

Across the repositories, the common documentation contract is pinned Markdownlint, numbered-record and relative-link validation, a portable disclosure check for real home paths in tracked text, and any repository-specific check that reads Markdown. The owner's global pre-push disclosure guard uses private deny patterns and remains mandatory before a real push. Go lint, race, vulnerability, build and release checks remain in full CI.

## Decision Drivers

* A documentation-only change has a successful, visible Docs run.
* A Markdown file read by code must run that reader check in Docs or full CI.
* Non-document changes, tags, schedules and manual runs keep full CI behavior.
* Both workflows use the same documentation commands and pins.
* CI disclosure checks work without private host data; the pre-push guard still protects publication.

## Considered Options

* Keep the current workflow split and existing checks.
* Align the documentation gates and add a portable disclosure check, with a path and gate parity test.
* Detect changed paths inside one full workflow and conditionally skip expensive jobs.

## Decision Outcome

*Accepted 2026-10-30 by the owner: "approve 0022 and 0024, then proceed".*

Chosen option: "Align the documentation gates and add a portable disclosure check, with a path and gate parity test", because it gives both workflows the same common documentation checks, covers local Markdown readers, and makes filter drift observable. This is proposed for the owner; it does not approve CI implementation.

### Consequences

* Good, because documentation-only changes can complete after focused checks.
* Good, because records, links, Markdown style and generic disclosure faults are checked consistently.
* Neutral, because a mixed change may start both workflows.
* Bad, because private identity patterns remain enforceable only by the local pre-push guard; CI has narrower coverage.
* Bad, because path-filtered full-CI jobs may be pending for a required pull-request check; required-status settings need review before rollout.

### Confirmation

The paired [0022-PLAN-align-docs-ci-and-disclosure-gates.md](0022-PLAN-align-docs-ci-and-disclosure-gates.md) requires failing planted inputs for each new gate, static workflow checks, and observed Docs-only and full-CI runs after an approved push. Local checks alone do not prove GitHub path-filter behavior.

## Pros and Cons of the Options

### Keep the current workflow split and existing checks

* Good, because it requires little work.
* Bad, because the Docs job omits at least one common gate.

### Align the documentation gates and add a portable disclosure check, with a path and gate parity test

* Good, because it establishes the same baseline while preserving local Markdown readers.
* Bad, because it adds a maintained scanner and parity test.

### Detect changed paths inside one full workflow and conditionally skip expensive jobs

* Good, because a single workflow can expose one stable status.
* Bad, because conditional jobs add control logic and can obscure whether documents passed.

## More Information

Keep the narrow ignore list and guide excerpt check. Add a record naming/pair/index and relative-link gate using the repository's record conventions, then run it with pinned Markdownlint and the guide check in both the full gate job and Docs. Add a portable tracked-text disclosure target to both jobs and retain the local pre-push guard. Add a Python filter contract check that compares push and pull-request ignore lists to the docs include lists and refuses to ignore `command/testdata/**/*.md`; run it in the full gate job. Update `AGENTS.md` and `docs/architecture.md`.

This fleet alignment decision cites 0019-MADR-skip-full-ci-for-docs-only-pushes.md without rewriting its history. The implementation is [0022-PLAN-align-docs-ci-and-disclosure-gates.md](0022-PLAN-align-docs-ci-and-disclosure-gates.md).

*Built 2026-10-30* by that PLAN, now `complete`. Its Confirmation holds: each
new gate was seen failing on planted input, the workflows pass actionlint and
the contract check, and a docs-only push (Docs 41109199104) and a mixed push
(CI 41103202698, Docs 41103202654) routed as decided.

### A1 (2026-11-03): a contract-check case that sees no problem says why

*Status: proposed (2026-11-03),* from the owner's choice "Record, propose
0018 fix (Recommended)" in [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md) Step 6, deviation D18, and their answer
the same day, "0022, as 0018 A6 says (Recommended)": [0018-MADR-python-verification-tools.md](0018-MADR-python-verification-tools.md) A6 item 2 puts
a fix to `scripts/check_ci_filters_test.py` or the script it tests here.
To be implemented by [0022-PLAN-align-docs-ci-and-disclosure-gates.md](0022-PLAN-align-docs-ci-and-disclosure-gates.md) Phase 4.

**Found.**

1. **A case fails now and then on the Windows host, and only there.**
   `script-test-check_ci_filters` failed one case in [0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md) Step 7d (D30),
   its case not named then; passed ten runs of 0018-PLAN Step 11, which
   made a failure name its case; then failed in 0009-PLAN Step 6's run
   of every gate, "FAIL the recipe drops the guide check: want
   ['readers: make docs-check does not run --check-guide, the reader in
   scripts/go-examples.py'], got []", and, rerun alone on the same tree,
   passed twice and failed a third time, "FAIL a pattern with no file:
   want ["tracked: CI ignores 'nothing/**/*.md', which matches no tracked
   file"], got []". It passes on macOS and Linux.
2. **What the failures share:** a case broke its copy, and the checker's
   run gave no problem. The copy was broken: `Tree.edit` asserts that its
   text occurs, and a miss would read "the case stopped".
3. **Why the cause is not known:** the test's `run` keeps the lines that
   start "check-ci-filters: " and do not end "problem(s)", and nothing
   else. `check_ci_filters.py` prints every error it handles on such a
   line, so `[]` is either a run that found nothing in a broken copy, or
   a run that ended before its summary, such as an exception it does not
   catch, whose traceback and exit status `run` drops. The case of D30
   and both of D18 could be either.

**Considered.**

* **A. The test tells the two apart.** `run` keeps the exit status and
  the whole output, and refuses a run the checker did not complete: one
  with no "check-ci-filters: N problem(s)" line, or whose status does not
  agree with N. The case then stops with the status and the output's
  last lines, and the gate is run on the Windows host until it fails
  again, so that the cause is seen before anything is fixed.
* **B. A case that gets nothing runs again.** It would hide the fault,
  whichever it is: a workaround.
* **C. The gate skips the Windows host.** The same, for one host.

**Decided (proposed):** "A", because the next failure then names its
cause, and B and C remove the evidence.

1. **`run` refuses an incomplete run,** and its message gives, on one
   line, the exit status and the last 10 lines of the output, joined, so
   that `case`, which prints the last line of a traceback, prints them in
   the case's FAIL line, and the run goes on to the next case, as
   AGENTS.md, "Scripts", requires.
2. **The flake is then looked into:** the gate runs alone on the Windows
   host until a run fails or twenty have passed, and what the failure
   names is recorded.
3. **The cause and its fix are the owner's to choose:** they are recorded
   with their evidence and offered as a deviation of Phase 4, and no fix
   is made before one is picked.
4. **What is not changed:** the contract the checker holds, the
   workflows, CI's and Docs' commands, and every other gate.

