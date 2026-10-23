---
status: accepted
date: 2026-10-20
decision-makers: the repository owner
consulted: "GitHub's workflow syntax reference (path filters and the filter pattern cheat sheet), read 2026-10-20; the Actions timing API for runs 38921099237 and 38176803908; go-selfupdate-lib docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md; this repository's CI at e2a904b"
informed: contributors, through docs/architecture.md and docs/guides/releasing.md
---
# Skip the full CI for a push or pull request that changes only the documents, and check them with a one-job docs workflow instead

## Context and Problem Statement

Every push to `main` and every pull request runs all of
`.github/workflows/ci.yml`. A commit that changes only Markdown gets the
same run as a code change: a record, an index row, a PLAN closing. Of the
60 commits on `main` from `d6598a3` (2026-10-09) to `e2a904b`
(2026-10-16), 20 changed only `.md` files. Pushes batch commits, so runs
are fewer than commits, but each docs-only push still runs everything.

On 2026-10-20 the owner asked for this repository to stop running its
full CI for pushes and pull requests that change only Markdown, as
go-selfupdate-lib had just done.

**Facts [observed 2026-10-20, at `e2a904b`]:**

* **The model.** go-selfupdate-lib
  `docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md` chose,
  and its PLAN carried out, `paths-ignore: ['**.md']` on `ci.yml`'s `push`
  and `pull_request` triggers, plus a one-job `docs.yml` with `paths:`
  that runs the Markdown checks `ci.yml` runs, with the same commands.
  That repository measured 0 billable ms for its public CI, and the owner
  went ahead for turnaround: its Docs run took 12 s against 18 jobs and
  15–20 minutes, with less exposure to macOS runner queues and fewer
  runs. This record restates what it relies on from there; it does not
  link it, since a relative link cannot reach another repository.
* **The triggers** (`ci.yml:2-6`): `push` to `main` and tags `v*`, and
  `pull_request`. A push to any other branch runs nothing. A newer run on
  the same ref cancels the older (`ci.yml:11-13`). `ci.yml` declares no
  `permissions:`.
* **The jobs:** `modules` (`ci.yml:24-42`, Linux), `test` (`ci.yml:44-88`,
  one leg per module and OS on `ubuntu-24.04`, `macos-15` and
  `windows-2025`) and `gates` (`ci.yml:92-208`, Linux). With one module in
  `go.work` today, that is 5 jobs.
* **Minutes.** The repository is public. The timing API reports 0
  billable ms for every job of run `38921099237` (3 Ubuntu, 1 Windows, 1
  macOS) and of run `38176803908`, a docs-only push (`d9930f8`, five
  Markdown files). This repository's CI bills no minutes, as in the model.
* **What a docs-only run costs anyway:** `38176803908` took 673 s, and
  `38921099237` 758 s, of which `gates` ran 12 minutes and the three
  `test` legs 4–6 minutes each. A docs-only push waits for a macOS and a
  Windows runner it does not need.
* **The only Markdown check in `ci.yml`** is markdownlint,
  `npx --yes markdownlint-cli2@0.23.2` at `ci.yml:198`, inside the
  `shellcheck, markdownlint, actionlint` step (`ci.yml:181-199`). There is
  no link check and no architecture-table check, unlike the model, which
  ran three. `.markdownlint-cli2.jsonc` lints `**/*.md` and excludes the
  MADR and PLAN files by name.
* **Code reads tracked Markdown here, unlike in the model:**
  * `command/testdata/commands/` holds five `.md` command files
    (`.draft.md`, `Fix_Bug.md`, `git/commit.md`, `notes.md`, `review.md`).
    `command/loaddir_test.go:35`, `:61` and `:160` load them, and
    `TestLoadDirGolden` compares the result with
    `command/testdata/golden/loaddir.golden`. A change to one of those
    files can fail `go test`, which only `ci.yml` runs.
  * `scripts/go-examples.py:288` reads every `docs/guides/*.md`: each Go
    block after a `<!-- from: testdata/frameworks/<file>#<name> -->` line
    must equal that region of the framework example. `docs/guides/commands.md`
    holds the 10 such excerpts today. `ci.yml` runs it as part of
    `make examples` (`ci.yml:164-168`). Its `--check-guide` mode
    (`scripts/go-examples.py:32`) runs only this comparison, with no build
    and no network; on the tree it reports "10 excerpt(s) checked" and
    exits 0.
  * Nothing else: no `//go:embed` in the repository names a file;
    `internal/conformance` names `docs/glossary.md` in comments and a
    message only, and keeps its list in code (`sharedNames`,
    `internal/conformance/conformance_test.go:473`); the `tools/verify`
    tests and the `.md` names in other Go tests write their own fixtures.

  So the model's `**.md` filter is unsafe here: it would skip `go test`
  for a change to a command fixture, and the guide check for a change to
  a guide's excerpt.
* **GitHub's path filters** (workflow syntax reference, read 2026-10-20):
  * "Path filters are not evaluated for pushes of tags." A tag keeps the
    full run.
  * `paths` and `paths-ignore` cannot both be used for one event; to
    exclude inside an include, `paths` takes `!` patterns, checked in
    order.
  * `*` "does not match the `/` character"; `**` matches any character;
    `docs/**/*.md` matches "A file with a `.md` suffix anywhere in the
    `docs` directory", `docs/README.md` included. The reference says
    nothing about names that begin with a dot.
  * A workflow skipped by a path filter leaves its checks "Pending",
    which blocks a pull request that requires them.
* **No check is required.** `GET repos/maccavelli/go-tui-lib/rulesets`
  returns `[]`, and `branches/main/protection` returns 404, "Branch not
  protected".
* **Releases.** The repository has 22 tags, `v0.12.0` the newest, and no
  GitHub releases. `docs/guides/releasing.md`, "Release the root alone",
  steps 2 and 3, has CI pass on the pushed `main` commit and then tags
  "the commit CI passed". `v0.11.0` was tagged on a docs-only commit,
  `2bf3da5`: its `main` push ran CI `38163689050`, and the tag push ran
  CI `38164582704`. Adapter tags (`<dir>/vX.Y.Z`) do not match `v*` and
  run no CI today; that is unchanged here.

## Decision Drivers

* A docs-only change keeps the checks `ci.yml` runs on the documents.
* A change that any test, gate or script reads keeps the full run, and so
  does every tag.
* A new kind of Markdown, or Markdown in a new place, runs the full CI
  until a record says otherwise: the filter fails towards running.
* No third-party action, and nothing that makes a skipped job read as a
  pass.
* The workflow stays simple enough that a later change cannot quietly
  disable CI for code.

## Considered Options

* **A.** `paths-ignore` on `ci.yml`'s `push` and `pull_request` triggers
  for the documents (B's list), and nothing else.
* **B.** As A, plus a new `.github/workflows/docs.yml` that runs, on the
  same documents, one Linux job: markdownlint, and the guides' excerpt
  check. The documents are named by place, not by `**.md`:
  `'*.md'`, `'docs/**/*.md'`, `'.claude/**/*.md'`, `'.grok/**/*.md'` and
  `'.opencode/**/*.md'`.
* **C.** One workflow: a first job decides from `git diff --name-only`
  whether anything but the documents changed, and every heavy job carries
  an `if:` on its output; a docs job always runs.
* **D.** `[skip ci]` in the commit message of a docs-only commit.

## Decision Outcome

Chosen option: "B", because it keeps markdownlint and the guide check
for a docs-only change at the cost of one small workflow, uses GitHub's
own filters, and leaves `ci.yml`'s jobs untouched. Naming the documents
by place keeps the command fixtures under `command/testdata/` in the full
run, and a tag is never filtered. The owner approved it as recommended on
2026-10-20, answering "go-tui-lib 0019" to "Which repos' pairs do you
approve as recommended (MADR accepted, PLAN in-progress, then Phase
0/records and Phase 1 per its plan)?".

* **`ci.yml`:** `on.push` and `on.pull_request` each gain
  `paths-ignore` with the five patterns. A push that changes anything
  else, `command/testdata/commands/*.md` included, runs as before.
* **`docs.yml`:** `push` to `main`, and `pull_request`, each with `paths:`
  the same five patterns plus `.markdownlint-cli2.jsonc` and
  `.github/workflows/docs.yml`. One `ubuntu-24.04` job runs:
  * `npx --yes markdownlint-cli2@0.23.2`, the command at `ci.yml:198`;
  * `python3 scripts/go-examples.py --check-guide`, the Markdown half of
    `make examples` (`ci.yml:164-168`). The build and run half needs Go
    and a changed example, which a docs-only change cannot be.
* **Not run there:** shellcheck, actionlint, `py_compile` and the rest of
  `gates`, which check no Markdown. A change to `docs.yml` itself runs
  `ci.yml` too, since it is not a document, so actionlint covers it.

### Consequences

* Good, because a docs-only push runs one Linux job, with no macOS or
  Windows runner to wait for.
* Good, because a code push, a fixture change and every tag run as
  before.
* Good, because a Markdown file outside the five places, such as a future
  package README or another `testdata` fixture, runs the full CI.
* Neutral, because a push that changes code and documents runs both
  workflows, so markdownlint runs twice.
* Neutral, because the release guide changes: a docs-only commit on
  `main` gets the docs workflow, not CI, so "the commit CI passed" is the
  tag's own run. Its code is the same as at the last commit whose full CI
  passed, since a docs-only commit changes none. An adapter tag runs no
  CI at all, so for an adapter that last full run is the only code proof,
  and the guide must say so.
* Bad, because the five patterns live in two files and must agree: a
  pattern in `ci.yml` that `docs.yml` lacks leaves those files unchecked.
  A comment in each names the other.
* Bad, because a required status check, if one is ever set, must not name
  a `ci.yml` job: a docs-only pull request would leave it "Pending". The
  record that sets one must say so.
* Bad, because GitHub does not document whether a pattern matches names
  that begin with a dot. `.claude/**/*.md` starts with the literal
  `.claude/`, and `command/testdata/commands/.draft.md` matches none of
  the five patterns whatever the answer, so each way the filter errs, it
  runs the full CI.

### Confirmation

* `actionlint`, as `ci.yml:199` runs it, passes on both workflows.
* The docs job's two commands fail on scratch copies with a planted
  markdownlint error and a planted excerpt drift, and pass on the tree. A
  planted change to `command/testdata/commands/notes.md` fails
  `TestLoadDirGolden`, which shows why the fixtures stay in the full run.
* After the change lands, a docs-only push to `main` shows a "Docs" run
  and no "CI" run, and a push that changes a non-Markdown file shows
  "CI". The PLAN records both run IDs.
* The next root release's tag push runs the full CI, recorded in that
  release's PLAN.

## Pros and Cons of the Options

### A. `paths-ignore` only

* Good, because it is one line per trigger.
* Bad, because a docs-only change then gets no markdownlint, and a
  drifted guide excerpt reaches `main` unchecked until the next code
  push fails `make examples`.

### B. `paths-ignore` and a docs workflow

* Good, because the documents keep their checks, and `ci.yml` keeps its
  jobs unchanged.
* Good, because both filters are GitHub's own, documented behaviour.
* Good, because naming places, not `**.md`, keeps every Markdown file a
  test reads in the full run.
* Bad, because there are two workflows to keep in step: a new place for
  documents must be named in both.
* Bad, because the model's single `'**.md'` pattern, which would be
  simpler, cannot be used here: `paths-ignore` takes no `!` exception for
  `command/testdata/`, and the `paths: ['**', '!**.md', …]` form that
  could would also skip a future `.md` a test reads.

### C. One workflow, gated jobs

* Good, because one file holds it all.
* Bad, because every heavy job gains an `if:`, and a wrong condition can
  skip CI for code with nothing to show it; a skipped job reads as neither
  pass nor fail.
* Bad, because the change-detection job runs on every push, a Linux
  runner still.

### D. `[skip ci]`

* Bad, because the global `prepare-commit-msg` hook writes the commit
  messages, and a person would have to remember it each time. It also
  skips markdownlint and the guide check.

## More Information

* The PLAN:
  [0019-PLAN-skip-full-ci-for-docs-only-pushes.md](0019-PLAN-skip-full-ci-for-docs-only-pushes.md).
* The CI being changed is described in `docs/architecture.md`'s CI bullet
  under "Tooling", and the release procedure in
  [docs/guides/releasing.md](../guides/releasing.md).
* The examples gate and its guide check are
  [0014-PLAN-api-policy-gates.md](0014-PLAN-api-policy-gates.md) Step 5.
