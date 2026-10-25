---
status: in-progress
date: 2026-10-20
associated-madr: "0019-MADR-skip-full-ci-for-docs-only-pushes.md"
---
# Implement: a docs-only push or pull request runs a docs job instead of the full CI

Associated MADR: [0019-MADR-skip-full-ci-for-docs-only-pushes.md](0019-MADR-skip-full-ci-for-docs-only-pushes.md)
(option B, which the owner chose on 2026-10-20).

## Goal

A push or pull request that changes only the documents runs one Linux
docs job, markdownlint and the guide excerpt check, instead of
`ci.yml`'s 5 jobs. Any other change, a command fixture under
`command/testdata/` included, and every tag, runs `ci.yml` as before.

## Scope

### Facts this PLAN starts from (2026-10-20, at `e2a904b`)

| Fact | Where it was read |
| :--- | :--- |
| `ci.yml` triggers on `push` to `main` and tags `v*`, and `pull_request` | `.github/workflows/ci.yml:2-6` |
| its only Markdown check is `npx --yes markdownlint-cli2@0.23.2` | `ci.yml:198`, in the step at `ci.yml:181-199` |
| `make examples` checks the guides' excerpts; `--check-guide` does only that | `ci.yml:164-168`; `scripts/go-examples.py:32`, `:288` |
| `TestLoadDirGolden` reads `command/testdata/commands/*.md` | `command/loaddir_test.go:34-35` |
| checkout is pinned to `3d3c42e5aac5ba805825da76410c181273ba90b1` (v7.0.1) | `ci.yml:29` |
| actionlint runs on every workflow in the directory | `ci.yml:199`, no file argument |
| the repository is public; CI bills 0 ms | `gh repo view`; the timing API for runs `38921099237` and `38176803908` |
| no ruleset, no branch protection | `GET …/rulesets` is `[]`; `GET …/branches/main/protection` is 404 |
| the release guide tags "the commit CI passed" | `docs/guides/releasing.md:63-65`, `:82`, `:97`, `:110`, `:131` |

### In scope

| Step | Paths | Delivers |
| :--- | :--- | :--- |
| 1 | this pair, `docs/README.md` | the records |
| 2 | `.github/workflows/ci.yml`, `.github/workflows/docs.yml` (new) | the filters and the docs workflow |
| 3 | `docs/architecture.md`, `docs/guides/releasing.md` | the documents |
| 4 | this PLAN, the MADR's status, `docs/README.md` | the live checks, and closing |

### Out of scope

* Any change to `ci.yml`'s jobs or steps, including the `permissions:`
  block it lacks.
* CI for adapter tags (`<dir>/vX.Y.Z`), which `v*` does not match today.
* Required status checks: none exist.
* `AGENTS.md`: it names no CI job or trigger that this changes. The
  architecture page is where CI is described.
* A release, and a changelog entry: the repository keeps no
  `CHANGELOG.md`, and the change is to CI and documents only.

## Implementation Steps

### Rules

1. **A step starts when the previous one is committed.** The agent
   commits on `main` only when the owner asks, with `git commit
   --no-edit`. Pushes need the owner's ask, after the disclosure guard
   over the outgoing commits.
2. **A new check is seen failing on a scratch copy outside the
   repository before it is relied on,** through
   `python3 -m tools.verify mutate`, whose copies go to a new temporary
   directory. The mutation file lives in the session's scratch directory
   and is not committed.
3. **Anything this PLAN does not say stops the step** for the owner.

### Step 1: records

* This pair.
* Its two rows in `docs/README.md`.

**Done when** the owner approves this PLAN. The commit holds the records
only, under the bootstrap exception; the MADR becomes `accepted` and this
PLAN `in-progress` in it.

### Step 2: the filters and the docs workflow

1. **`ci.yml`:**
   * `on.push` gains, beside `branches` and `tags`:

     ```yaml
     paths-ignore: ['*.md', 'docs/**/*.md', '.claude/**/*.md', '.grok/**/*.md', '.opencode/**/*.md']
     ```

   * `on.pull_request` gains the same line.
   * A comment above them says: a change to these documents alone runs
     `docs.yml` instead, which lists the same patterns; a tag push is
     never filtered; `command/testdata/` stays out of the list because
     tests read it; and it cites the MADR.
2. **`.github/workflows/docs.yml`,** new:
   * `name: Docs`;
   * `on.push` with `branches: [main]`, and `on.pull_request`, each with
     `paths:` the five patterns above plus `'.markdownlint-cli2.jsonc'`
     and `'.github/workflows/docs.yml'`;
   * `permissions: contents: read`, and the concurrency group
     `docs-${{ github.ref }}` with `cancel-in-progress: true`;
   * one job, `docs`, on `ubuntu-24.04`, with `timeout-minutes: 10`;
   * `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1`,
     the pin at `ci.yml:29`, with `persist-credentials: false`;
   * step `markdownlint`: `npx --yes markdownlint-cli2@0.23.2`, unchanged
     from `ci.yml:198`;
   * step `guide excerpts`, `shell: bash`: `python3 --version`, then
     `python3 scripts/go-examples.py --check-guide`;
   * a header comment: what it checks, that its patterns must match
     `ci.yml`'s `paths-ignore`, and the MADR.

   Like `ci.yml`, it uses the runner image's Node and Python, with no
   setup action. The guide check needs Python 3.12, the runner's.
3. **Seen failing.** A mutation file with three entries, each run in its
   own copy:
   * `README.md`: `## Status` becomes `##Status`; `cmd`
     `["npx", "--yes", "markdownlint-cli2@0.23.2"]`. It must exit 1 and
     name `README.md:19` and MD018.
   * `docs/guides/commands.md`: in the excerpt at line 339, `approves a
     command that asks` becomes `approves any command that asks`; `cmd`
     `["python3", "scripts/go-examples.py", "--check-guide"]` with `env`
     `{"EXAMPLES_ROOT": "."}`, since a copy has no `.git`. It must exit 1
     and name `docs/guides/commands.md:`, not exit 2.
   * `command/testdata/commands/notes.md`: ~~`Summarise the notes so far`
     becomes `Summarize the notes so far`~~ `# Summarise the notes`
     becomes `# Summarize the notes` (the heading; corrected 2026-10-20);
     `test` `["./command/", "-run", "TestLoadDirGolden"]`. It must fail
     against the golden file. This is why the fixtures stay out of the
     filter.

     *Annotated 2026-10-20:* as written, this plant survives. The golden
     file records the command's description, which comes from the
     `# Summarise the notes` heading, not the body line. The Execution
     Record's Step 2 deviation entry plants the heading instead, which
     fails `TestLoadDirGolden`, and awaits the owner's confirmation.

     *Confirmed 2026-10-20:* the owner chose the heading plant ("Use the
     heading plant (Recommended)"), so the third plant is the heading
     change named above, beside the struck-through original. The
     body-line change is
     still caught by `go test ./command/`, through
     `TestArgumentsTakesRest`, not by `TestLoadDirGolden`.

   Each verdict must be KILLED, and its kept output read to confirm the
   failure is the planted one. The tree passes the first two commands,
   run as they stand.
4. **Checks:**
   * `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`, as
     `ci.yml:199`, on both workflows;
   * `python3 -m tools.verify gates --only markdownlint,actionlint`;
   * the identifier scan, `python3 -m tools.verify scan`.

   The filters themselves are proven only on GitHub, at Step 4.

**Done when** each check passes and the three plants are KILLED. The
commit holds the two workflow files and this PLAN's record of the step.

### Step 3: the documents

1. **`docs/architecture.md`:**
   * the tree listing under `.github/workflows/` gains `docs.yml`;
   * the CI bullet under "Tooling" says that a push or pull request that
     changes only the five documents' places skips `ci.yml`, that
     `docs.yml` runs markdownlint and the guide check on one Linux runner
     instead, that `command/testdata/` is never skipped, and that a tag
     push is never filtered.
2. **`docs/guides/releasing.md`:** steps 2 and 3 of "Release the root
   alone", step 2 of "Release an adapter alone", and the CI lines of
   "Release a change that spans the root and an adapter" say: when the
   commit to tag changes only documents, its push runs the docs
   workflow, and the code proof is the last full CI run, on the last
   commit that changed anything else. A root tag push then runs the full
   CI; an adapter tag runs none.
3. **Checks:** markdownlint, the guide check, and the identifier scan.
   This commit changes Markdown only, so its push is Step 4's docs-only
   test.

### Step 4: live checks and closing

1. **A push with a non-Markdown change:** the push of Step 2 changes
   workflow files, so both workflows start. Record each run's ID, its
   conclusion and its job count: "CI" 5 of 5, "Docs" 1 of 1.
2. **A docs-only push:** after the owner pushes Step 3, GitHub's run list
   for that commit (`actions/runs?head_sha=…`) must hold one run, "Docs",
   with conclusion `success`, and no "CI" run. Record its ID and wall
   time.
3. **Tags:** not exercised here. The next root release's tag runs the
   full CI; that release's PLAN records it (V4).
4. **A fixture change:** not exercised live; Step 2's patterns and the
   reference's `*` and `docs/**/*.md` rules exclude it, and the next
   commit that touches `command/testdata/commands/` runs CI, which its
   own record can note.
5. **Closing:** `### Verification, at closing`, this PLAN `complete`,
   and `docs/README.md`'s row for it. The MADR stays `accepted`.

## Verification

* **V1 (filters):** `ci.yml` has `paths-ignore` with the five patterns on
  push and pull request; `docs.yml` has the same five in `paths`, plus
  its config and itself, and runs the two checks; actionlint passes.
* **V2 (seen failing):** each of the three plants is KILLED for its
  planted reason; the tree passes markdownlint and the guide check.
  *Annotated 2026-10-20:* the third plant is the `notes.md` heading
  change, as the owner confirmed (Execution Record, Step 2 deviation).
* **V3 (live):** a docs-only push ran only "Docs"; a push with a
  non-Markdown change ran "CI", 5 of 5.
* **V4 (tags):** the next root tag ran the full CI, left to that
  release.
* **V5 (documents):** the architecture page and the release guide say
  so; markdownlint and the identifier scan are clean.

## Rollout and Rollback

* **Rollout:** Steps 2 and 3 on `main`, pushed on the owner's word.
* **Rollback:** `git revert` of Step 2's commit restores the full CI for
  every push and removes `docs.yml`; a revert of Step 3 restores the
  documents. Nothing outside the repository depends on the split.

## Execution Record

### Approval (2026-10-20)

The owner approved the pair as recommended on 2026-10-20, answering
"go-tui-lib 0019" to "Which repos' pairs do you approve as recommended
(MADR accepted, PLAN in-progress, then Phase 0/records and Phase 1 per
its plan)?". That approves option B, and Steps 1 to 3 as written; Step 4
waits for the owner's pushes.

### Step 1: records

*2026-10-20.* No deviation.

* **The MADR** is `accepted`, with "Chosen option" and the owner's words
  in its Decision Outcome; **this PLAN** is `in-progress`, with this
  record; `docs/README.md`'s two rows say so.
* **Records only,** under the bootstrap exception: this pair and
  `docs/README.md`.

### Step 2: the filters and the docs workflow

*2026-10-20.* One deviation, in item 3's third plant (below).

#### What was built

* **`.github/workflows/ci.yml`:** `on.push` and `on.pull_request` each
  gain `paths-ignore: ['*.md', 'docs/**/*.md', '.claude/**/*.md',
  '.grok/**/*.md', '.opencode/**/*.md']`. The comment above `on:` says
  that a change to these documents alone runs `docs.yml` instead, whose
  paths list the same patterns; that a tag push is never path-filtered;
  that `command/testdata/` stays out because tests read it; and cites the
  MADR. No job or step changed.
* **`.github/workflows/docs.yml`,** new, as item 2 says: `name: Docs`;
  `push` to `main` and `pull_request`, each with `paths:` the five
  patterns plus `.markdownlint-cli2.jsonc` and
  `.github/workflows/docs.yml`; `permissions: contents: read`; the
  concurrency group `docs-${{ github.ref }}`, cancelling; one job, `docs`,
  on `ubuntu-24.04`, `timeout-minutes: 10`; checkout at
  `3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1` with
  `persist-credentials: false`; step `markdownlint`
  (`npx --yes markdownlint-cli2@0.23.2`) and step `guide excerpts`
  (`shell: bash`; `python3 --version`, then
  `python3 scripts/go-examples.py --check-guide`); and the header comment.

#### Seen failing

`python3 -m tools.verify mutate <scratch>/mutations-0019.json --work
<scratch>/mut --keep`, the mutation file and copies in the session's
scratch directory, outside the repository; exit 1:

```text
## markdownlint: README.md heading without a space (MD018): KILLED (exit 1)
## guide check: drifted excerpt in docs/guides/commands.md: KILLED (exit 1)
## TestLoadDirGolden: changed command fixture notes.md: SURVIVED (exit 0)
mutate: 2 killed, 1 survived, 0 errors
```

The kept output of each:

* **The MD018 plant** (`## Status` to `##Status` in `README.md`):

  ```text
  Summary: 1 issue in 1 file
  README.md:19:1 error MD018/no-missing-space-atx No space after hash on atx style heading [Context: "##Status"]
  ```

* **The excerpt plant** (`approves a command that asks` to `approves any
  command that asks`), with `EXAMPLES_ROOT=.`; exit 1, not 2:

  ```text
  examples: docs/guides/commands.md:335: the excerpt differs from testdata/frameworks/flag/main.go#runcli (- the program, + the guide):
    @@ -1,5 +1,5 @@
     // runCLI runs a registry command from the program's own command line, and
    -// writes its result to w. AllowIf(yes) approves a command that asks, when
    +// writes its result to w. AllowIf(yes) approves any command that asks, when
  ...
  examples: 10 excerpt(s) checked
  ```

  It names line 335, the excerpt's `<!-- from: -->` line; the planted
  text is at line 339, as item 3 says.
* **The `notes.md` plant, as written** (`Summarise the notes so far` to
  `Summarize the notes so far`, `-run TestLoadDirGolden`): SURVIVED.

  ```text
  ok  	github.com/maccavelli/go-tui-lib/command	0.525s
  ```

#### Deviation, 2026-10-20: the `notes.md` plant names a line the golden file does not record

* **Found:** `TestLoadDirGolden` (`command/loaddir_test.go:35-43`)
  writes each command's fields through `describeCmd`, and
  `command/testdata/golden/loaddir.golden` holds `notes`'s
  `description: Summarise the notes`, which comes from the file's
  `# Summarise the notes` heading. The body line the plant changes is in
  no field the golden file records. The plant was mis-specified when the
  PLAN was written; the code and the MADR are as they say.
* **Not a decision change:** the MADR's facts hold (a change to a
  command fixture can fail `go test`, which only `ci.yml` runs), and its
  Confirmation, "A planted change to
  `command/testdata/commands/notes.md` fails `TestLoadDirGolden`", holds
  for the heading. Both plants below were run, on scratch copies, with
  `python3 -m tools.verify mutate <scratch>/mutations-0019-fixture.json
  --work <scratch>/mutfx --keep`; exit 0, "mutate: 2 killed, 0 survived,
  0 errors":
  * **The heading,** `# Summarise the notes` to `# Summarize the notes`,
    `-run TestLoadDirGolden`: KILLED (exit 1).

    ```text
    --- FAIL: TestLoadDirGolden (0.00s)
        loaddir_test.go:43: tuitest: loaddir color.utf8.0: line 23 differs
              want " 34|  description: Summarise the notes"
              got  " 34|  description: Summarize the notes"
    FAIL
    ```

  * **The body line, as planted, with the whole package** (`./command/`):
    KILLED (exit 1), by another test that reads the fixture:

    ```text
    --- FAIL: TestArgumentsTakesRest (0.00s)
        loaddir_test.go:77: "/notes 100 or so" expands to "# Summarise the notes\n\nSummarize the notes so far, in 100 or so words.", <nil>; want "# Summarise the notes\n\nSummarise the notes so far, in 100 or so words."
    FAIL
    ```

* **Proposed resolution, awaiting the owner's confirmation:** the third
  plant is the heading change, as in the first bullet above; item 3 is
  annotated, not rewritten. Until the owner confirms, V2's third plant is
  open.
* **Confirmed by the owner on 2026-10-20:** "Use the heading plant
  (Recommended)". The third plant is the heading change; Step 2 item 3
  is annotated to name it, with the original anchor struck through.
  V2's third plant is closed: the heading plant is KILLED for its planted
  reason, `TestLoadDirGolden` against the golden file:

  ```text
  --- FAIL: TestLoadDirGolden (0.00s)
      loaddir_test.go:43: tuitest: loaddir color.utf8.0: line 23 differs
            want " 34|  description: Summarise the notes"
            got  " 34|  description: Summarize the notes"
  FAIL
  ```

  The body-line change the PLAN first planted is caught too, by
  `TestArgumentsTakesRest`, when the whole `./command/` package runs
  (quoted above), so either kind of change to the fixture fails
  `go test`, which only `ci.yml` runs.

#### Checks

* **The tree passes both docs commands,** run as they stand:
  `npx --yes markdownlint-cli2@0.23.2`, exit 0, "Linting: 21 files",
  "Summary: 0 issues in 0 files"; `python3 scripts/go-examples.py
  --check-guide`, exit 0, "examples: 10 excerpt(s) checked",
  "examples: clean".
* **actionlint,** `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`
  from the repository root, as `ci.yml` runs it: exit 0, no output. With
  `-verbose`, it lints both files: "Found total 0 errors … for
  .github/workflows/docs.yml" and "… for .github/workflows/ci.yml".
* **`python3 -m tools.verify gates --only markdownlint,actionlint`:**
  "markdownlint: exit=0", "Summary: 0 issues in 0 files",
  "actionlint: exit=0", "gates: 2 passed, 0 failed, 0 skipped".
* **The identifier scan,** `python3 -m tools.verify scan`, over the
  change and the untracked files against `HEAD`, with the identifier
  patterns of the local config: "scan: clean (5 pattern(s))".
* **The filters themselves** are proven only on GitHub, at Step 4.

### Step 3: the documents

*2026-10-20.* No deviation.

#### What was written

* **`docs/architecture.md`:**
  * the tree, under `.github/workflows/`, gains `docs.yml`, "the
    documents' checks, for a docs-only change";
  * the CI bullet gains "Documents only": a push to `main` or a pull
    request that changes only Markdown at the root or under `docs/`,
    `.claude/`, `.grok/` or `.opencode/` skips `ci.yml` and runs
    `docs.yml` instead; Markdown under `command/testdata/` is never
    skipped; a tag push is never path-filtered; it cites the MADR;
  * a new "Docs" bullet: its triggers and paths, its one `ubuntu-24.04`
    job and two commands, that its `paths` must agree with `ci.yml`'s
    `paths-ignore`, and that a change to code and documents runs both.
* **`docs/guides/releasing.md`:**
  * "Release the root alone", step 2: a docs-only push runs Docs instead
    of CI, and the code proof is then the last full CI run on `main`, on
    the last commit that changed anything else; it cites the MADR. Step 3
    tags the commit CI passed, or after a docs-only push the commit Docs
    passed, and says the root tag push runs the full CI;
  * "Release an adapter alone", step 2: the same, and an adapter tag
    (`<dir>/vX.Y.Z`) runs no CI, so for an adapter that last full run is
    the only code proof;
  * "Release a change that spans the root and an adapter": step 1 says
    the same for the root, and that its tag push runs the full CI; step 2
    says the adapter's tag runs no CI, so its last CI run is its only code
    proof.

#### Each claim, against the workflows

* **A docs-only push runs Docs, not CI:** `ci.yml`'s `on.push` and
  `on.pull_request` ignore exactly the five patterns, and `docs.yml`'s
  `paths` hold the same five.
* **`command/testdata/` is never skipped:** none of the five patterns
  matches it; `'*.md'` does not match `/`, and the others are anchored at
  `docs/`, `.claude/`, `.grok/` and `.opencode/`.
* **A root tag runs the full CI:** `ci.yml`'s `push.tags: ['v*']`, and
  path filters are not evaluated for tag pushes (the MADR's facts).
  `docs.yml` has `branches: [main]` and no `tags`, so a tag runs no Docs.
* **An adapter tag runs no CI:** `stream/glamourmd/vA.B.C` does not
  match `v*`, and `docs.yml` takes no tag.
* **Docs runs markdownlint with `ci.yml`'s command and the guide check:**
  `docs.yml`'s two steps.

#### Checks

* `python3 -m tools.verify gates --only markdownlint,actionlint`:
  "markdownlint: exit=0", "Summary: 0 issues in 0 files",
  "actionlint: exit=0", "gates: 2 passed, 0 failed, 0 skipped".
* `python3 scripts/go-examples.py --check-guide`: exit 0, "examples: 10
  excerpt(s) checked", "examples: clean".
* The two new relative links resolve: `docs/decisions/0019-MADR-…` from
  `docs/architecture.md`, and `../decisions/0019-MADR-…` from
  `docs/guides/`.
* The tree listing's new line keeps the column of its neighbours.
* The identifier scan, `python3 -m tools.verify scan`, with the
  identifier patterns of the local config: "scan: clean (5
  pattern(s))".
* This step changes Markdown only, so its push is Step 4's docs-only
  test.

### Step 4: live checks (2026-10-25)

*Read from GitHub's run list for `main` and the tags, 2026-10-25, the last
100 runs.*

* **A push with a non-Markdown change** ran "CI", 5 of 5, and, when it
  also changed Markdown, "Docs" beside it: the first such push after Step
  2's workflows were on `main`, `63c68d6` (CI `39963257917`, 5 jobs, and
  Docs `39963257717`, both `success`); then `8be11e7`, `e0ae2bd`,
  `81177eb` and `d5509df` alike. V3's second half is met.
* **Tags (V4):** the root tags `v1.13.0` (at `8be11e7`) and `v2.0.0` (at
  `e0ae2bd`, run `40143509687`) each ran "CI" with 5 of 5 jobs and no
  "Docs", as `docs.yml` takes no tag. V4 is met.
* **A docs-only push (V3's first half): not yet seen.** No push since
  Step 2 has changed only the documents' places: `e0ae2bd`'s commit changed
  only Markdown, but it went up with code commits, and GitHub filters a
  push by everything it carries; `81177eb`'s carried
  `scripts/apicheck.allow`. The commit that holds this record changes only
  Markdown under `docs/`, so its push, made alone, is the test: one run,
  "Docs", and no "CI".
* **A fixture change:** not exercised, as the step says.
* **Not done yet:** V3's first half, and with it the closing, which
  follow that push.
