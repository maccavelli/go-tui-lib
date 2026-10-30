---
status: in-progress
date: 2026-10-30
associated-madr: "0022-MADR-align-docs-ci-and-disclosure-gates.md"
---
# Implement aligned docs-only CI and disclosure gates

Associated MADR: [0022-MADR-align-docs-ci-and-disclosure-gates.md](0022-MADR-align-docs-ci-and-disclosure-gates.md)

## Goal

A documentation-only push or pull request gets a Docs run with the common Markdown, record/link and portable disclosure gates plus this repository's Markdown-reading checks. Other changes retain full CI. The private pre-push guard remains part of publication.

## Scope

Affected files: `.github/workflows/ci.yml`, `.github/workflows/docs.yml`, `Makefile`, `scripts/check_records.py` (new), `scripts/check_ci_filters.py` (new), `scripts/check_disclosure.py` (new), their tests, `AGENTS.md`, `docs/architecture.md`, `docs/README.md`.

Repository-specific work: Keep the narrow ignore list and guide excerpt check. Add a record naming/pair/index and relative-link gate using the repository's record conventions, then run it with pinned Markdownlint and the guide check in both the full gate job and Docs. Add a portable tracked-text disclosure target to both jobs and retain the local pre-push guard. Add a Python filter contract check that compares push and pull-request ignore lists to the docs include lists and refuses to ignore `command/testdata/**/*.md`; run it in the full gate job. Update `AGENTS.md` and `docs/architecture.md`.

Boundary: Preserve full CI for command fixtures and all nonlisted paths. The user-facing records gate must cover records that Markdownlint's globs intentionally exclude.

## Implementation Steps

### Phase 1: inventory and shared checks

1. Re-read workflow triggers, job commands, Markdownlint globs, tracked Markdown files and every code or script reader of tracked Markdown. Record the docs-safe path set in this PLAN before editing filters. Include hidden agent-rule trees and test fixtures. Review required-status settings; if a filtered job would stay pending, amend this pair with a concrete migration before changing filters.
2. Add missing record/link and Markdownlint targets, reusing the current commands where they exist. Run identical pinned documentation commands in full CI and Docs. Keep Go lint, tests, vulnerability, builds and releases in full CI.
3. Implement a standard-library portable disclosure check over tracked text. It rejects real macOS, Linux and Windows user-home paths, accepts documented placeholders, prints only file, line and category, and fails on unreadable tracked input. Run the same target in full CI and Docs. Retain the private deny list and global pre-push guard locally.
4. Add fixture tests for new gates. Prove each fails on deliberately broken input in a scratch copy and passes on clean input. Record commands, statuses and redacted diagnostics. Commit the approved phase after checks pass.

### Phase 2: workflow routing and parity

1. Keep push and pull-request CI ignore sets identical and limited to audited docs-safe paths. Make Docs paths cover every ignored path and the Docs workflow itself. Mixed changes run full CI. Preserve tags, schedules, dispatch, release jobs and permissions.
2. Add or extend a Python workflow contract check. It validates actual YAML structure, compares event filters, confirms ignored tracked paths are in Docs, refuses known Markdown inputs to code tests unless their reader runs in Docs, and compares common docs-gate commands and pins, and fails when full CI loses a tag, schedule or dispatch trigger it had, or gains a `paths` filter. Integrate it in the workflow/tool test target.
3. Run actionlint and the contract check. In independent scratch copies, plant a mismatched path, a Markdown reader wrongly ignored, a missing Docs gate and a differing pin; require each to fail. Commit the approved phase after checks pass.

### Phase 3: documentation and observed rollout

1. Explain routing, docs-safe paths, common checks, local readers and the private pre-push guard in AGENTS.md and the architecture or CI guide. Verify this pair's exact filenames and current statuses in the records index; use the repository index generator if present.
2. Run `make check-records markdownlint docs-check disclosure-check`; `python3 scripts/go-examples.py --check-guide`; actionlint; scratch plants for a missing index row, broken link, MD018, real home path, divergent filters and a command fixture wrongly ignored. Capture status before filtering output. Check this pair and edited documents for real-machine identifiers. Read each edited file in full and verify links.
3. After explicit authorization to push in that same turn, observe a docs-only push and a non-doc or mixed push. Tag, schedule and dispatch routing is not waited for: GitHub evaluates no path filter for a tag push, `schedule` and `workflow_dispatch` carry none, and the contract check of phase 2, step 2 fails if a trigger is lost or a `paths` filter is added. Record run IDs, conclusions and which workflow started. Mark this PLAN complete only after the criteria below hold; otherwise record the outstanding observation. Commit at the end of each approved phase under repository rules. *Amended 2026-10-25: the wait for the next tag or scheduled run is removed, as acp-go-sdk2's 0037-PLAN-align-docs-ci-and-disclosure-gates.md did in its step 3.*

## Verification

* Both workflows pass pinned Markdownlint, numbered-record/index and relative-link validation, and portable tracked-text disclosure scanning.
* Each Markdown reader runs on changes to its input, either in Docs or full CI.
* A docs-only push starts Docs but not full CI; a non-doc or mixed push starts full CI. Tags, schedules and dispatch retain existing behavior, shown by the contract check and the workflows' structure, not by waiting for a run.
* Scratch plants fail their intended gates; filter or gate drift fails the parity test.
* Before an authorized push, the local pre-push guard checks outgoing commits. Its private deny list is never copied into CI.

## Rollout and Rollback

Land gates before relying on the filters. If routing misclassifies a change, revert the routing commit through a new reviewed commit so full CI runs for every push again. Keep the shared gates while repairing the audit. This PLAN does not authorize a push or tag.

## Execution record

### Approval, 2026-10-30

The owner: "approve 0022 and 0024, then proceed". The MADR is `accepted`, with `decision-makers` and a
dated note; this PLAN is `in-progress`. The two rows in `docs/README.md`
say so.

### Phase 1, step 1: the inventory, 2026-10-30

Read at `a088933`, read-only.

**The workflows.**

* `ci.yml` runs on `push` to `main` and to `v*` tags, and on every
  `pull_request`. Both events carry the same `paths-ignore`: `*.md`,
  `docs/**/*.md`, `.claude/**/*.md`, `.grok/**/*.md` and
  `.opencode/**/*.md`. It has no `schedule` and no `workflow_dispatch`.
  Its jobs are `modules`, `test` (three systems) and `gates`. In `gates`,
  the step "shellcheck, markdownlint, actionlint" ran
  `npx --yes markdownlint-cli2@0.23.2`.
* `docs.yml` runs on `push` to `main` and on `pull_request`, with
  `paths` equal to that list plus `.markdownlint-cli2.jsonc` and itself.
  It ran markdownlint at the same pin and
  `scripts/go-examples.py --check-guide`.
* GitHub evaluates no path filter for a tag push, so a `v*` tag runs
  full CI.

**markdownlint's globs** (`.markdownlint-cli2.jsonc`): `**/*.md`, less the
build trees and every file named like a MADR or a PLAN, so 24 files. The
records are left to the records check.

**Tracked Markdown:** 77 of 1,045 tracked files.

* 67 under `docs/`;
* `README.md` and `AGENTS.md`;
* the three agent-rule files: `.claude/rules/madr-and-plan-skill.md`,
  `.grok/rules/madr-plan-before-mutating-work.md` and
  `.opencode/rules.md`;
* five command fixtures under `command/testdata/commands/`.

**Code that reads tracked Markdown,** by a scan of every tracked
non-Markdown file for a name ending in `.md`, and of every file-reading
call in tracked Go and Python outside `testdata`:

| Reader | Reads | Runs in |
| :--- | :--- | :--- |
| `command/loaddir_test.go`, through `command.LoadDir` | `command/testdata/commands/**/*.md` | full CI, `go test` |
| `internal/conformance/defaults_test.go`, `TestDefaultsInventory` | `docs/guides/defaults.md`, and the records its rows cite under `docs/decisions/` | full CI only, until this phase |
| `scripts/go-examples.py` | `docs/guides/*.md`, against the Go regions they name | full CI's examples step; Docs, `--check-guide` |
| `markdownlint-cli2` | `**/*.md`, less the records | full CI; Docs |

Every other hit names a `.md` file only as data: a test's fixture path
(`ask/layout_test.go`, `fuzzy/example_test.go`,
`palette/providers_test.go`, `stream/printer_test.go`), a comment, or a
docstring. `tools/verify` reads no tracked Markdown. CI never runs it on
the tree.

**One reader was skipped.** `TestDefaultsInventory` reads
`docs/guides/defaults.md` and checks that the records its rows cite
exist. Both paths match `docs/**/*.md`, which CI ignores. So a docs-only
change that breaks the inventory, such as a renamed control or a removed
record, ran neither the test nor any other check of it. The gap dates
from `0024-PLAN-command-and-control-opt-outs.md`, which added the test
after 0019's filter. Step 2 closes it by running the test in Docs, the
first route the PLAN's Phase 2 step 2 allows ("unless their reader runs
in Docs"). The other route would drop `docs/decisions/**` and the guide
from CI's ignored paths, which would send nearly every record commit to
full CI.

**The docs-safe path set:** CI's five patterns, unchanged. With
`TestDefaultsInventory` and the guide check running in Docs, every reader
of a file they match runs in Docs. Left out:

* `command/testdata/**/*.md`, which `go test` reads;
* `.markdownlint-cli2.jsonc`, which Docs lists so it lints on a config
  change, but which CI does not ignore;
* every non-Markdown file.

**Required status checks:**

* `gh api repos/<owner>/go-tui-lib/branches/main/protection`: "Branch not
  protected" (404);
* the repository's rulesets and the rules for `main`: `[]` each.

No check is required, so a full-CI job that a filter skips cannot leave a
pull request pending, and no migration is needed.

### Phase 1, steps 2 to 4: the shared checks, 2026-10-30

**Deviation D1, 2026-10-30: six broken links in quoted release bodies.**

* *Found:* the first run of `scripts/check_records.py` on the tree found
  six broken links, all present before this step:
  * `0008-PLAN-command-palette.md:1548`, `:1549`;
  * `0020-PLAN-charm-free-cores-glyphs-and-safetext.md:1889`, `:3014`;
  * `0024-PLAN-command-and-control-opt-outs.md:814`, `:815`.

  Each is in a release body that the PLAN quotes word for word. The links
  are written relative to the repository root, not to the record, so
  from the record none resolves.
* *Also:* from the root, five resolve. The sixth,
  `#migrating-to-v0130`, names a section of the migration guide that
  0020-MADR A4 renamed `v1.13.0`.
* *Decision:* the owner chose "Fix links, annotate" over having the
  checker resolve a quoted link from the root.
  * Each link is now relative to its record (`../guides/…`), and the
    `v0.13.0` link uses `#migrating-to-v1130`.
  * A dated note under each quote says the published release body is
    unchanged.
* *Scope added:* the three records above.
* *MADR:* unchanged; neither a decision nor an asserted fact moves.

What changed:

* **`scripts/check_records.py`, new,** the standard library only. Over
  the files `git ls-files` lists, it checks:
  * *names:* a Markdown file in a record directory that is not a record,
    or a lookalike anywhere;
  * *placement* of each record;
  * *pairs:* a second MADR of a number, a PLAN with no MADR, a GATES with
    no PLAN;
  * *the index:* one row per record in `docs/README.md`'s "Records"
    table, whose number and kind match the file it links, and no second
    row;
  * *links:* every relative link and anchor in every tracked Markdown
    file outside `testdata`, against GitHub's anchor rules, ignoring code
    spans and fences.

  `--next` prints the next number: `0025`. Exit 1 on an error, 2 when git
  fails.
* **`scripts/check_disclosure.py`, new,** ported from acp-go-sdk2's
  `scripts/check_disclosure.py` (that repository's
  `0037-PLAN-align-docs-ci-and-disclosure-gates.md`).
  * Over every tracked file, it refuses a macOS, Linux or Windows home
    path whose account is not a placeholder, and a tracked file it cannot
    read. It prints only `FILE:LINE: CATEGORY`.
  * Its first run found 8 hits, all made-up names. Five were
    `tools/verify`'s redaction tests (four) and a failure that
    `0018-PLAN-python-verification-tools.md` quotes (one), which use four
    generic personas. The other three were a `C:\Users\...` in comments,
    two in `scripts/go-modules.py` and one in its test.
  * So the placeholders are now `user`, `u`, the four personas, and an
    account that is only dots or an ellipsis. None is a real account.
* **The tests,** in the harness the other scripts use, each report a FAIL
  line by case:
  * `scripts/check_records_test.py`, 8 cases, 33 checks;
  * `scripts/check_disclosure_test.py`, 7 cases, 10 checks.

  Each builds throwaway repositories, and its last case runs the script
  on this repository. The disclosure test builds every real-looking path
  from pieces, so it passes the check itself.

  The first run of the records test failed 2 checks, both faults in the
  test: a tree of six Markdown files was counted as five, and a tree
  without its index still had a README linking to it.
* **`Makefile`:** `MARKDOWNLINT ?= npx --yes markdownlint-cli2@0.23.2`,
  so the pin is in one place, and four targets:
  * `markdownlint`;
  * `check-records`;
  * `docs-check`: `go-examples.py --check-guide`, then
    `GOWORK=off go test -count=1 -run '^TestDefaultsInventory$'
    ./internal/conformance/`;
  * `disclosure-check`.
* **`ci.yml`:** the `gates` job's new step, `documents`, runs the two
  tests, then `make markdownlint check-records docs-check
  disclosure-check`. The old step's own markdownlint call is removed, and
  the step is renamed "shellcheck, actionlint".
* **`docs.yml`:** `actions/setup-go` at CI's pin, from `go.work`, then the
  same `make` line, in place of its two commands. Its triggers are
  unchanged; Phase 2 routes them.

**Checks,** each exit 0, with the status taken before the output was
read:

* `python3 scripts/check_records_test.py`: "33 passed, 0 failed";
* `python3 scripts/check_disclosure_test.py`: "10 passed, 0 failed";
* `make markdownlint check-records docs-check disclosure-check`:
  * "Summary: 0 issues in 0 files";
  * "check-records: 55 record(s), 72 Markdown file(s), 0 error(s)";
  * "examples: 15 excerpt(s) checked", "examples: clean";
  * `TestDefaultsInventory` "ok";
  * "check-disclosure: 1045 tracked file(s), 0 finding(s)";
* `actionlint@v1.7.12` over both workflows; `py_compile` of the four
  scripts;
* `python3 -m tools.verify scan --base HEAD`: "scan: clean".

**Seen failing,** through `python3 -m tools.verify mutate` on scratch
copies. A helper in the session's scratch directory first makes each copy
a git repository, since the copies have no `.git` and both scripts read
`git ls-files`. 28 killed of 28:

| Mutation | Killed by |
| :--- | :--- |
| 1.1 a non-record file in a record directory accepted | `FAIL docs/decisions/notes.md is misnamed` |
| 1.2 placement not checked | `FAIL a PLAN under docs/reports` |
| 1.3 a second MADR of a number accepted | `FAIL two MADRs of 0001` |
| 1.4 a PLAN with no MADR accepted | `FAIL a PLAN with no MADR` |
| 1.5 a GATES with no PLAN accepted | `FAIL a GATES with no PLAN` |
| 1.6 a record with no index row accepted | `FAIL a record with no row` |
| 1.7 an index row's number not compared | `FAIL a row's number differs` |
| 1.8 the index read outside its "Records" section | `FAIL a row under another heading is not the record's` |
| 1.9 a link to a missing file accepted | `FAIL a missing file`, `FAIL the root README`, `FAIL an agent-rule file` |
| 1.10 anchors not checked | `FAIL a missing anchor` |
| 1.11 a repeated heading's anchor not numbered | `FAIL a clean tree exits 0`: `broken anchor: ../guides/g.md#usage-1` |
| 1.12 code spans read as links | `FAIL a clean tree has no error`: ``broken link: `x.md` `` |
| 1.13 testdata Markdown checked | `FAIL a testdata file is not checked` |
| 1.14 untracked files read | `FAIL untracked files are not checked` |
| 1.15 `--next` from the lowest number | `FAIL --next after 0009: want (0, '0010'), got (0, '0002')` |
| 1.16 every account a placeholder | `FAIL findings exit 1: want 1, got 0` |
| 1.17 only bracketed accounts placeholders | `FAIL placeholders pass`, and `FAIL this repository is clean` |
| 1.18 two backslashes tried before one | `FAIL placeholders pass`: the JSON-escaped forms found as `windows-home` |
| 1.19 a home path inside a longer path found | `FAIL paths that are not home paths pass` |
| 1.20 an unreadable file skipped | `FAIL unreadable files fail`: `latin.txt` missing |
| 1.21 a binary file scanned | `FAIL unreadable files fail`: `bin.dat:1: macos-home` |
| 1.22 a finding prints its text | `FAIL the output does not hold the account` |
| 1.23 plant: MD018 in `docs/guides/line-mode.md` | `make markdownlint`: "docs/guides/line-mode.md:36:1 error MD018/no-missing-space-atx" |
| 1.24 plant: a link to `0099-MADR-gone.md` in the 0022 MADR | `make check-records`: "…0022-MADR-align-docs-ci-and-disclosure-gates.md:18: broken link: 0099-MADR-gone.md" |
| 1.25 plant: this PLAN's index row removed | `make check-records`: "docs/README.md: no row for docs/decisions/0022-PLAN-align-docs-ci-and-disclosure-gates.md" |
| 1.26 plant: the defaults guide names `workspace.WithoutMouseAtAll` | `make docs-check`: "docs/guides/defaults.md:122 (The mouse): built, but workspace exports no WithoutMouseAtAll" |
| 1.27 plant: the guide's `session`, `return 80` to `return 81` | `make docs-check`: "docs/guides/line-mode.md:167: the excerpt differs from testdata/frameworks/linemode/main.go#session" |
| 1.28 plant: a Linux home path with a made-up account in `AGENTS.md` | `make disclosure-check`: "AGENTS.md:468: linux-home", without the account |

Plants 1.26 and 1.27 are the readers that Docs now runs.

Not checked locally: what the changed steps do on GitHub, until a push.
