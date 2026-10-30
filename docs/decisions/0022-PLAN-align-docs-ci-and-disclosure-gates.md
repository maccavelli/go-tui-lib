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

Phase 1 was committed by the owner as `dfce61e`.

### Phase 2: workflow routing and parity, 2026-10-30

*Step 1, the routing,* needed no change. CI's `push` and `pull_request`
already carry the same `paths-ignore`, the docs-safe set of step 1. Docs'
`paths` are that set, `.markdownlint-cli2.jsonc` and `docs.yml`. Neither
workflow's triggers, concurrency, permissions or jobs change in this
phase. CI has no `schedule` and no `workflow_dispatch` to keep.

What changed:

* **`scripts/check_ci_filters.py`, new:** the contract check, ported from
  acp-go-sdk2's `scripts/check_ci_filters.py`. It reads both workflows as
  YAML, the `Makefile`'s recipes and the tracked files, and checks five
  parts:
  * *`filters`:* CI's two ignore lists are equal and not empty, and CI has
    no `paths` filter. CI keeps each trigger in `CI_TRIGGERS`: `push` to
    `main`, `push` to `v*` tags, and `pull_request`.
  * *`docs`:* Docs' two lists are equal, and are CI's ignored paths plus
    `DOCS_EXTRA`.
  * *`tracked`:* each ignored pattern matches a tracked file, and Docs
    runs on every tracked file CI ignores.
  * *`readers`:*
    * no ignored file is under a `testdata` directory, so
      `command/testdata/**/*.md` stays in full CI;
    * for each of `READERS` (`TestDefaultsInventory` and the guide
      check) whose input CI ignores, Docs runs its make target, and the
      target's recipe runs the reader;
    * every tracked file a Go file names by a literal path, as a string
      or as `filepath.Join`'s literal parts, is not ignored, is read by
      one of `READERS`, or is in `NOT_READS`.
  * *`gates`* and *`pins`:* both workflows run the four document targets
    through make, neither names `markdownlint-cli2` itself, and the
    `Makefile` pins it once. Every action Docs uses, CI uses at the same
    pin, and Docs has `contents: read`.

  `NOT_READS` lists, each with its reason, the four Go literals that name
  an ignored document without reading it:
  * `fuzzy/example_test.go`'s two candidate strings;
  * a key of `palette/providers_test.go`'s in-memory file system;
  * the name `TestDefaultsInventory` looks for in a row's status.

  A line there that no longer matches fails as stale. The first run,
  before `NOT_READS` existed, reported exactly those four.
* **The YAML reader** is acp-go-sdk2's, for the subset the workflows use.
  It refuses anchors, aliases, tags, flow mappings, document markers,
  duplicate keys and tabs in indentation.
  * PyYAML 6.0.3, which this host has, served once, from the session's
    scratch directory, as an oracle and not a dependency.
  * On both workflows, the reader's value equalled `yaml.safe_load`'s,
    with YAML 1.1's `on: true` key read as `on`, and PyYAML's booleans and
    integers written as the reader keeps them, as strings.
* **The globs** follow GitHub's: `**/` also matches no directory, so
  `.opencode/**/*.md` matches `.opencode/rules.md`.
* **`scripts/check_ci_filters_test.py`, new:** 12 cases, 42 checks.
  * It copies the tracked files once into a scratch git repository, and
    each case copies that and breaks one part.
  * It also covers the reader, the globs, the refused YAML and this
    repository itself.
  * Every check passed on its first run; the mutations below are its
    failures.
* **`ci.yml`:** the `gates` step "shellcheck, actionlint, CI filters" runs
  the test and the check after actionlint. The header comment names the
  check. **`docs.yml`:** its header comment says that the check holds its
  paths to CI's.

**Checks,** each exit 0:

* `python3 scripts/check_ci_filters.py`: "check-ci-filters: 0
  problem(s)";
* `python3 scripts/check_ci_filters_test.py`: "42 passed, 0 failed";
* `actionlint@v1.7.12` over both workflows; `py_compile` of the two
  scripts;
* `scripts/check_records.py`: 0 errors; `scripts/check_disclosure.py`: 0
  findings;
* `python3 -m tools.verify scan --base HEAD`: "scan: clean".

**Seen failing,** through `python3 -m tools.verify mutate` on scratch
copies, each made a git repository first, as in Phase 1. 22 killed of 22:

| Mutation | Killed by |
| :--- | :--- |
| 2.1 the `pull_request` list not compared | `FAIL lists differ: exit 1: want 1, got 0` |
| 2.2 a `paths` filter accepted | `FAIL a paths filter` |
| 2.3 the triggers not checked | `FAIL tags lost`, `FAIL the branch lost`, `FAIL pull_request lost` |
| 2.4 Docs' `push` and `pull_request` not compared | `FAIL Docs' lists differ` |
| 2.5 Docs' paths not compared with CI's | `FAIL Docs misses an ignored pattern` |
| 2.6 a pattern with no file accepted | `FAIL a pattern with no file` |
| 2.7 an ignored file Docs misses accepted | `FAIL and its files` |
| 2.8 `testdata` may be ignored | `FAIL testdata ignored` |
| 2.9 a reader's target not looked for in Docs | `FAIL Docs without docs-check` |
| 2.10 a reader's recipe not read | `FAIL the recipe drops the inventory`, `FAIL the recipe drops the guide check` |
| 2.11 `filepath.Join`'s parts not read | `FAIL a Join names one` |
| 2.12 a stale `NOT_READS` line accepted | `FAIL a stale NOT_READS line` |
| 2.13 `markdownlint-cli2` in a workflow accepted | `FAIL CI runs markdownlint itself` |
| 2.14 the `Makefile`'s pin not counted | `FAIL the pin twice` |
| 2.15 Docs' action pins not compared | `FAIL another checkout pin` |
| 2.16 Docs' permissions not checked | `FAIL Docs may write` |
| 2.17 a duplicate key accepted | `FAIL a duplicate key is refused: want 2, got 0` |
| 2.18 `**/` needs a directory | `FAIL the copy is clean`: "tracked: CI ignores '.opencode/**/*.md', which matches no tracked file" |
| 2.19 plant: CI's `pull_request` list loses `.opencode/**/*.md` | "filters: CI's push and pull_request paths-ignore differ, or are empty" |
| 2.20 plant: `command/testdata/**/*.md` added to CI's lists and Docs' | "readers: CI ignores command/testdata/commands/notes.md, under a testdata directory go test reads", one line for each of the 5 fixtures |
| 2.21 plant: Docs drops `disclosure-check` from its make line | "gates: Docs does not run make disclosure-check" |
| 2.22 plant: Docs sets up Go at another pin | "pins: Docs uses actions/setup-go@0a2b…, and CI ['actions/setup-go@b7ad…']" |

The four plants are the ones step 3 names: a mismatched path, a Markdown
reader wrongly ignored, a missing Docs gate and a differing pin. Each was
applied to the real workflows, and the contract check alone was run.

Not checked locally: how GitHub routes a push. Phase 3, step 3 observes
it after an authorized push.

Phase 2 was committed by the owner as `4ff2e22`.

### Phase 3, steps 1 and 2: documentation and the final checks, 2026-10-30

What changed:

* **`AGENTS.md`:**
  * "Records": `scripts/check_records.py --next` prints the next number,
    and `make check-records`, with what it fails on;
  * "Pre-add checks": before staging a document, the documents' line,
    `make markdownlint check-records docs-check disclosure-check`. Before
    staging a workflow or a script, also `scripts/check_ci_filters.py` and
    the tests of each script changed;
  * a new section, "CI routing":
    * the docs-safe paths, and that any other file, a mix or a `v*` tag
      runs CI;
    * why Markdown that code reads is not docs-safe unless its reader runs
      in Docs: `command/testdata/**/*.md` stays in CI, and
      `TestDefaultsInventory` and the excerpt check run in Docs;
    * the one line both workflows run;
    * what `scripts/check_ci_filters.py` fails on, and that a new reader
      joins `READERS` in its own change;
  * "Identifiers": `make disclosure-check` is the portable half and does
    not replace the pre-push guard. It is placed after "Never bypass it
    with `--no-verify`", so that line's "it" still means the guard, the
    fault acp-go-sdk2's
    `0037-PLAN-align-docs-ci-and-disclosure-gates.md` recorded.
* **`docs/architecture.md`:**
  * the tree names the three scripts and their tests;
  * "Tooling":
    * the four make targets, and what each runs;
    * `scripts/check_ci_filters.py` and its YAML reader;
    * the `gates` job's `documents` step and the CI-filters check;
    * Docs' Go setup and shared line;
    * the local pre-push guard, which no workflow has.
* **The records index:** the pair's rows name both files exactly, with
  `accepted` and `in-progress`, matching each file's front matter. The
  index is written by hand; `scripts/check_records.py` checks its rows
  and generates none.

Both files were read in full after the edit. No fault was found.

**Checks,** each exit 0, with the status taken before any output was
read:

* `make check-records markdownlint docs-check disclosure-check`:
  * "check-records: 55 record(s), 72 Markdown file(s), 0 error(s)";
  * "Summary: 0 issues in 0 files";
  * "examples: 15 excerpt(s) checked", "examples: clean";
  * `TestDefaultsInventory` "ok";
  * "check-disclosure: 1051 tracked file(s), 0 finding(s)";
* `python3 scripts/go-examples.py --check-guide`: "15 excerpt(s) checked",
  "clean";
* `actionlint@v1.7.12`; `python3 scripts/check_ci_filters.py`: "0
  problem(s)";
* `python3 -m tools.verify scan --base HEAD`, over this phase's change,
  this pair included: "scan: clean (5 pattern(s))", the identifier
  patterns of the local, uncommitted config.

**Planted breaks,** the six step 2 names. Each was applied to the tree as
it now stands, in its own scratch copy made a git repository, by
`python3 -m tools.verify mutate --keep`. The target was then run in each
copy, its status taken before its output was read. Six of six failed as
required:

| Break | Target | What failed |
| :--- | :--- | :--- |
| the 0024 terminal PLAN's index row removed | `make check-records`, exit 2 | "docs/README.md: no row for docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md" |
| a link to `guides/gone.md#nowhere` in `docs/architecture.md` | `make check-records`, exit 2 | "docs/architecture.md:540: broken link: guides/gone.md#nowhere" |
| `#CI routing` above the new section in `AGENTS.md` | `make markdownlint`, exit 2 | "AGENTS.md:465:1 error MD018/no-missing-space-atx" |
| a Windows home path with a made-up account in `docs/architecture.md` | `make disclosure-check`, exit 2 | "docs/architecture.md:540: windows-home", without the account |
| Docs' `pull_request` paths lose `*.md` | `scripts/check_ci_filters.py`, exit 1 | "docs: Docs' push and pull_request paths differ" |
| `command/**/*.md` added to CI's lists and Docs' | `scripts/check_ci_filters.py`, exit 1 | "readers: CI ignores command/testdata/commands/notes.md, under a testdata directory go test reads", one line for each of the 5 fixtures |

~~Step 3 is not done. It observes GitHub's routing after a push, which needs
the owner's authorization in that turn. Until it is recorded, this PLAN
stays `in-progress`.~~ Step 3 follows.

### Phase 3, step 3: the observed rollout, 2026-10-30

The owner committed steps 1 and 2 as `eb862cf` and pushed `main`. The push
was made before the agent ran the disclosure guard over it. Run afterwards,
over `a088933..eb862cf`, the guard exited 0 with nothing to report. Before
that commit, the agent had run it over `a088933..4ff2e22`, also exit 0.

| Push | Changes | Workflows started | Runs and conclusions |
| :--- | :--- | :--- | :--- |
| `a088933..eb862cf` | mixed: the workflows, `Makefile`, `scripts/`, `AGENTS.md`, records and docs | CI and Docs | CI 41103202698, "completed success", jobs `modules`, `gates` and `test` on Windows 2025, Ubuntu 24.04 and macOS 15; Docs 41103202654, "completed success", job `docs` |
| this record alone | docs-only: this PLAN | to be observed | |

In the mixed push's logs (`gh run view --log`):

* CI's `gates` job printed:
  * "check_records_test: 33 passed, 0 failed" and
    "check_disclosure_test: 10 passed, 0 failed";
  * from the shared line: "Summary: 0 issues in 0 files",
    "check-records: 55 record(s), 72 Markdown file(s), 0 error(s)",
    "examples: 15 excerpt(s) checked", `TestDefaultsInventory` "ok", and
    "check-disclosure: 1051 tracked file(s), 0 finding(s)";
  * "check_ci_filters_test: 42 passed, 0 failed" and "check-ci-filters: 0
    problem(s)".
* Docs printed the same five lines from the shared line, the
  inventory's `go test` among them.

So both workflows run the documents' checks on GitHub, and Docs' Go setup
and `Makefile` work on its runner.
