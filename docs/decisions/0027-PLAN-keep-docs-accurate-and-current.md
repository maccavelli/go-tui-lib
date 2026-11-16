---
status: in-progress
date: 2026-11-15
associated-madr: "0027-MADR-keep-docs-accurate-and-current.md"
---
# Implement accurate, comprehensive and current documents: verify the survey, fix what is wrong, describe what is missing, comment every exported name, start `CHANGELOG.md`, and bring the index current

Associated MADR: [0027-MADR-keep-docs-accurate-and-current.md](0027-MADR-keep-docs-accurate-and-current.md)

*Provenance: drafted by the strongest model at high effort; the survey it
starts from was made by a mainstream model at medium effort; the rulings
are the owner's, of 2026-11-14, relayed by the coordinator. This PLAN is
`proposed`: the owner has not approved it or said when it starts. On
2026-11-14 the owner ruled on the MADR's Q1 to Q5 and Q7, and the
coordinator applied the owner's ruling on the global rules file to Q6;
the phases below follow them, and no question is open.*

*(2026-11-15: the owner approved this PLAN and its MADR, relayed by the
coordinator, by choosing "Approve, one phase at a time". The MADR is
`accepted` and this PLAN `in-progress`. Phase A starts on 2026-11-15;
each later phase starts only on the owner's word after the phase before
it ends.)*

## Goal

Every living document of this repository is true of the tree, every
exported name has a doc comment, and every package, command, flag,
environment variable, settings key and make target is described in a
README, guide or reference section (MADR D1). `CHANGELOG.md` exists and
`AGENTS.md` names it (D2). `docs/README.md` indexes what was added. No
gate, CI job or check script is added or changed (D3), and no tag waits
for this work (D4).

Done means every phase's acceptance criteria hold at the closing commit,
the close-out's evidence is quoted in the execution record, and this PLAN
is `complete`.

## Scope

### Findings at `f093da6`

From the survey and from the MADR's own checks (MADR, "What the survey
found" and "What this record checked"). "Verified" means this record or the
coordinator confirmed it against the tree; "lead" means it is still to be
confirmed in Phase A; "refuted" means the claim did not hold.

| # | Where | Finding | Status | Phase |
| :--- | :--- | :--- | :--- | :--- |
| L1 | `docs/architecture.md:19` | "Thirty-one packages": `go list` gives 37 public; "ten internal" is right | verified | B |
| L2 | `docs/architecture.md:31` | `launch` lacks `command/commandtea` | verified | B |
| L3 | `docs/architecture.md:47` | `inline/inlinetest` lacks `internal/sanitize` | verified | B |
| L4 | `docs/architecture.md:48` | `palette` lacks `msgcat` | verified | B |
| L5 | `docs/architecture.md:49` | `workspace` lacks `command/commandtea` and `msgcat` | verified | B |
| L6 | `docs/architecture.md:52` | `command` lacks `msgcat` | verified | B |
| L7 | `docs/architecture.md:58` | `termcap` lacks `msgcat` | verified | B |
| L8 | `docs/architecture.md:70` | `internal/lineinput` lacks `internal/sanitize` | verified | B |
| L9 | `docs/architecture.md:560-561` | "What is not here" lists a scrolling Markdown pane; `stream/streamtea.DocPane` is one | lead | A, then B |
| L10 | `AGENTS.md:4` | `CLAUDE.md` absent | refuted: the sentence is conditional | — |
| L11 | `stream/glamourmd` in five files | named as if built | refuted for `docs/guides/releasing.md:6`, `docs/glossary.md:194-196`, `AGENTS.md:84` and `docs/architecture.md:199-202`, which say planned; the rest of `releasing.md` is read in context in Phase A | A |
| L12 | (none) | `lineedit/testdata/xtermparity` named in no living document, only in `lineedit/parity_test.go`, its own `main.go` and `0023-PLAN-line-mode-components.md` | verified | C |
| L13 | `docs/architecture.md:376-379` | `test-sum`, `fmt`, `tidy` and `help` named, not described; `all` not named | verified; a defect by the owner's Q4 ruling | C |
| L14 | (none) | the `Makefile` variables `GOLANGCI_LINT`, `GOVULNCHECK` and `GOTESTSUM` named in no document | verified | C |
| L15 | (none) | about 34 upper-case literals in `termcap`, `termsvc`, `launch`, `theme` and `tuitest` that look like environment variable names, named in no document | lead | A, then C |
| L16 | survey B2, B3 | exported names in no prose, by word search | reclassified: not a defect by D1 alone; input to Phase A's inventory | A |
| L17 | non-test Go files | exported names without a doc comment, by a line-based count: 3 methods, 451 fields, 42 interface methods without their own comment; 331 constants and 4 variables with a block comment only (MADR) | lead, for sizing | D |
| L18 | repository root | no `CHANGELOG.md` | verified | E |
| L19 | `AGENTS.md:147-149` | "Release notes list every deprecation and every removal" names no place | verified | E |
| L20 | `docs/reports/0014-REPORT-api-assessment-and-integration-research.md` | no front matter `date:`, where the other three reports have one | verified | F |
| L21 | `docs/guides/migrating.md` | a section for 21 of 39 tags, `v0.9.0` to `v2.12.0` | verified | E |

*(2026-11-15, Phase A: every row is settled, each with its evidence, in
"Findings (Phase A)" below: L1 to L8 re-confirmed; L9 verified, a wrong
statement; L10 refuted, as above; L11 refuted for every file; L12 to L14
verified; L15 verified, with the variables corrected; L16 not a defect,
its use recorded; L17 still a lead, which Phase D's measurement settles;
L18 to L21 verified. The Status column above is as it was written.)*

### In scope

| Phase | Files | What |
| :--- | :--- | :--- |
| A | `docs/decisions/0027-*`, `docs/decisions/0026-PLAN-agent-widgets.md` (a dated note), `docs/README.md` (statuses) | on approval, the statuses and 0026-PLAN's note; then every lead settled, every living document read in full, the feature inventory, the full defect list |
| B | `docs/architecture.md` first, then the files Phase A's wrong rows name | wrong statements |
| C | the guides, `docs/architecture.md`, `README.md`, as Phase A's missing rows name | missing feature descriptions |
| D | non-test `.go` files outside `testdata`, as Phase D's measurement names, except `glyph/glyph.go` | doc comments on exported names |
| E | `CHANGELOG.md`, `AGENTS.md`, `docs/guides/releasing.md`, `docs/guides/migrating.md`, `README.md`, `docs/architecture.md` (Tree) | the changelog and where it is named |
| F | `docs/README.md`, `docs/reports/0014-REPORT-api-assessment-and-integration-research.md` (front matter only), `docs/architecture.md` (Tree) | the index current |
| G | this PLAN, the MADR, `docs/README.md` | the close-out |

*(2026-11-15, by the owner's ruling on QA1, "Phase D, as its own group":
Phase D's row also takes the five package comments F18 to F22 name, in
`internal/cells/cells.go`, `internal/termevent/termevent.go`,
`fuzzy/doc.go`, `when/when.go` and `workspace/workspace.go`, as a package
group of their own. Its scope is doc comments on exported names and these
five package comments; the row above is as it was written.)*

### Out of scope

* Any new gate, CI job or check script, and any change to an existing one
  (MADR D3). A one-off audit lives outside the repository and is never
  committed.
* Code behaviour and the exported API: Phase D changes comments only.
* Any release, tag or `git push`.
* The bodies of records. A stale fact in an unexecuted record
  (0009-PLAN Step 7, 0022-PLAN Phase 4, 0026) is listed by Phase A for the
  owner, and corrected only by that record's own amendment.
* Building `diffview`, `agentui` or `agentui/agentuitea`, which is
  [0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md)'s; and
  `stream/glamourmd`.
* Other repositories.

## Prerequisites

* The owner has approved this PLAN and said when it starts. The MADR's
  questions are all answered (Q1 to Q5 and Q7 ruled, Q6 applied,
  2026-11-14), so no phase waits on a question.
* No release step is in progress (the order below). At `f093da6` none is:
  0026-PLAN is `proposed`.

## Order against 0026 and the next tag

* **The next tag** is `v2.13.0`, 0026-PLAN Step 9 (0026-MADR D8, from the
  base `v2.12.0`). 0026 waits for go-llmprovider-sdk `v1.31.0`, gobble-cli
  `v0.7.0` and the owner's approval of its PLAN.
* **No tag waits for this PLAN** (R4). Each release's own PLAN fixes what
  the release makes false and documents what it ships, before its tag:
  0026's Step 9 for `v2.13.0`, its Step 14 for `v2.14.0`.
* **Every finding here is an older gap,** present at `v2.12.0`; this PLAN
  lands them. By the owner's Q7 ruling, "Phases that change only Markdown
  run now, one at a time, and pause at a phase boundary if a release step
  starts. Doc-comment phases that edit Go files wait where code is about to
  be rewritten." Phases A, B, C, E, F and G change only Markdown and run
  now, in that order, one at a time; Phase E may run before B or C.
* **Phase D and the code 0026 rewrites.** Of the code files that exist
  at `f093da6`, 0026-PLAN's build edits `glyph/glyph.go` (Step 2), tests
  under `internal/conformance/` and `.golangci.yml`; its other code files
  are new. Only `glyph/glyph.go` holds exported names Phase D would comment
  (`glyph.Border`'s fields, by the approximate count). Phase D therefore
  runs now for every other file, after Phase A, and leaves
  `glyph/glyph.go` to 0026's Step 2, whose docs step comments that file's
  exported names as it rewrites it (Phase A's note in 0026-PLAN says so).
* **A release step in progress pauses this PLAN** at the next phase
  boundary. After its tag, the next phase starts by re-reading Phase A's
  list against the new tree: a row the release fixed is marked so, a moved
  line is renumbered, and a changed fact is a deviation.
* **`CHANGELOG.md` before 0026 Step 9.** Phase E should land first. If
  Step 9 comes first, it creates `CHANGELOG.md` with its `v2.13.0` entry in
  the shape MADR D2 sets; Phase E then adds the rest and does not rewrite
  that entry.
* **One PLAN at a time on the shared files:** `docs/architecture.md`,
  `docs/README.md`, `README.md`, `docs/guides/migrating.md`,
  `CHANGELOG.md`. No Go file is shared: Phase D leaves `glyph/glyph.go`
  to 0026's Step 2.

## The docs step in every later brief

By the mandate (MADR D5), every later brief for code work in this
repository carries this step, executed in the same phase as the change,
and the phase's execution record shows it:

1. a doc comment on each exported name added or changed, by MADR D1 and
   Q3's answer;
2. a description of each package, command, flag, environment variable,
   settings key or make target added or changed, where its reader reads,
   by D1 and the owner's Q4 ruling;
3. every statement the change makes false corrected, in the living
   documents and the package comments;
4. a line under `CHANGELOG.md`'s `Unreleased`;
5. `docs/README.md`'s rows for a new guide or section.

A phase with nothing to document says so, and why. 0026's build documents
`diffview`, `agentui` and `agentui/agentuitea` in the step that adds each
(Phase A records this in 0026-PLAN).

## Rules for every phase

1. **One phase, one purpose, one commit** (Phase D: one commit per package
   group). `git commit --no-edit`, the global hook writing the message;
   made by the owner, or by the executing agent where the owner's approval
   of the phase says so. No push and no tag.
2. **Stop condition.** An unexpected state, or the same step failing
   twice, ends the task: the executor returns the state and its evidence,
   and does not improvise or try a third time.
3. **Deviations** stop the phase, with resolutions offered that leave the
   tree correct, never workarounds; the owner's choice is recorded in this
   PLAN, dated, and in the MADR when a decision or a stated fact changes.
4. **A fact written is checked against its source in the same phase:** the
   code, `go doc`, `go list` with `GOWORK=off`, the `Makefile`, the
   workflows, the tags.
5. **One-off audits** (MADR D3) live in a scratch directory outside the
   repository and are never committed. Each is seen failing on a
   deliberately broken copy (a scratch export, `git archive HEAD`
   extracted outside the tree), never on the working tree, before its
   result is trusted; the failure is quoted in the execution record.
6. **Exit statuses** are captured before any filter: output is redirected
   to a scratch file, `$?` recorded, then the file read in full.
7. **The documents' checks** before every commit: `make check-records
   docs-check disclosure-check`, each exit 0; markdownlint over the
   non-record Markdown, by `make markdownlint` where the network is
   allowed or by a local `markdownlint-cli2` at the `Makefile`'s pin
   (`markdownlint-cli2 --version` prints `v0.23.2`) with the repository's
   configuration, ending `Summary: 0 error(s)`. A record is linted through
   a renamed copy, since the configuration's globs skip records.

   *(2026-11-15, by the owner's ruling on QA3, "Compare against the base
   copy", relayed by the coordinator: a record is linted as a renamed copy
   of it as the phase starts it (the base, the committed file) and a
   renamed copy of it as the phase leaves it, both with the repository's
   own configuration and the `Makefile`'s pin, alike, and the phase adds no
   issue: the counts by rule after are no higher than before, and no rule
   appears that was not there. The summary is read as markdownlint-cli2
   prints it, `Summary: N issues in M files`, not as `Summary: 0
   error(s)`; the base's older issues stay, and are not this phase's to
   fix. Each copy's name carries neither `MADR` nor `PLAN`, which the
   configuration's globs skip, or the run lints no file. The non-record
   Markdown is still linted as it is, and ends `Summary: 0 issues`. This
   resolves deviation A-2; the text above is as it was written.)*
8. **Identifiers.** `python3 -m tools.verify scan` over the phase's diff,
   clean, before the commit; nothing written carries a hostname, an
   account name or a real path.
9. **Heavy runs queue.** This PLAN needs none: `make pre-add-check
   FILES="…"` and `make docs-check` may run on the session host. Any heavy
   run (`make test`, `make release-check`, `make pre-add-check` without
   `FILES`, `python3 -m tools.verify gates`, `-race`, fuzzing) goes
   through `pick-host` and `run-on-host`, or `run-heavy -- <command>` on
   the session host, one at a time per host.

## Implementation Steps

### Phase A: verify the leads, read in full, list every defect

*(2026-11-14: Phase 0, which was to record the owner's answers, has
nothing left to record now that every question is answered in the MADR.
What remained of it, the statuses on approval and the dated note in
0026-PLAN, is step 0 of this phase, in the same records-only commit.)*

**Tier:** strongest, high effort (judgment on every claim); step 0 is
given text.

**Files read:** `README.md`, `AGENTS.md`, `docs/README.md`,
`docs/architecture.md`, `docs/glossary.md`, the ten files under
`docs/guides/`, every package comment (the file holding each `// Package`
comment), `.claude/rules/`, `.grok/rules/`, `.opencode/rules.md`, the
`Makefile`, the `--help` of every script under `scripts/` and of
`python3 -m tools.verify`, and `testdata/frameworks/cases.txt`. **Files
written:** this PLAN, the MADR's front matter, the note in 0026-PLAN, and
`docs/README.md`'s rows.

0. **On the owner's approval:**
   * the MADR: `status: accepted`, `date` of the approval;
   * this PLAN: `status: in-progress`; the italic note under the title
     records the approval and when it starts;
   * [0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md): a dated
     note after its italic ruling note, worded:
     *(DATE, by
     [0027-MADR-keep-docs-accurate-and-current.md](0027-MADR-keep-docs-accurate-and-current.md)
     D5: each step documents what it adds as it lands: a doc comment on
     every exported name it adds, by that record's D1; `diffview`,
     `agentui` and `agentui/agentuitea` in `docs/architecture.md` from the
     step that adds each; and a line under `CHANGELOG.md`'s `Unreleased`.
     Steps 9 and 14 also turn `Unreleased` into the release's entry. Step
     2, which rewrites `glyph/glyph.go`, also gives each exported name in
     that file the comment that record's D1 asks for, since 0027-PLAN's
     Phase D leaves the file to it.)*
   * `docs/README.md`: the two 0027 rows' statuses; 0026-PLAN's row
     unchanged.
1. Record the state: `git rev-parse --short HEAD`, `git describe --tags
   --abbrev=0`, `git status --short` (empty). Commits since `f093da6`
   that touch a living document are listed, and their effect on L1 to L21
   noted.
2. Settle L9, L11, L15 and L16 with the evidence named; re-confirm L1 to
   L8 with the import comparison below.
3. Read each file above in full, in the order listed, and check every
   factual claim (a name, path, number, default, behaviour, version, link
   target, command line) against its source (rule 4). Each file read is
   listed with its `wc -l`.
4. Build the inventory of features, each with the place that describes it
   or "none": the packages (`go list ./...`); the command IDs the library
   registers or exports; the flags (`launch`'s, `tuitest`'s, the scripts'
   and `tools.verify`'s); the environment variables the library reads
   through a `termcap.Env`, and those the scripts and `make` read; the
   settings keys (command files' front matter, `keybindings.json`, the
   layout state's JSON, the palette's recents, `tools/verify`'s
   configuration); the `make` targets and variables; the `testdata`
   programs and data the docs never mention.
5. Write the full list into this PLAN under "Findings (Phase A)": one row
   per defect, with its file and line, class (wrong, missing, stale),
   what the document says, the evidence, the fix, and the phase. Two
   further classes are listed and not fixed by this PLAN: **record**, a
   stale fact in an unexecuted record, for its own amendment; and
   **decision**, a finding whose fix is a choice, for the owner.
6. Stop for the owner with the list. Phases B to F execute against it; a
   decision row is answered first.

**Verification.**

* The import comparison: a scratch script reading `GOWORK=off go list -f
  '{{.ImportPath}}|{{join .Imports " "}}' ./...` and the diagram in
  `docs/architecture.md`, printing each row whose in-repository or Charm
  imports differ. Expected at `f093da6`: exactly the seven rows L2 to L8,
  and `37` public packages. Seen failing first on a scratch copy of
  `docs/architecture.md` with one more import removed from a row: it
  prints that row.
* Every lead in the table has a status; every file in "Files read"
  appears in the read list with its line count.
* `python3 -B scripts/check_records.py`: exit 0, `check-records: 61
  record(s), … 0 error(s)`.
* `git diff --cached --stat`: this PLAN, the MADR, 0026-PLAN and
  `docs/README.md`, nothing else; 0026-PLAN's diff is the note alone.

**Acceptance.** The statuses and 0026-PLAN's note are recorded; every lead
is verified or refuted with evidence; every listed file was read in full;
every inventory item has a place or a "missing" row; the owner has the
list.

### Phase B: wrong statements

**Tier:** mainstream, medium effort (corrections from a verified list).

**Files:** `docs/architecture.md` first, then each file Phase A's wrong
rows name, one file at a time.

1. `docs/architecture.md`, by the owner's Q2 ruling ("Restate without
   counts"): the count removed and the section restated, the diagram
   keeping each package's purpose and its external modules, the
   in-repository layering stated as the rules `depguard` and
   `internal/conformance` enforce; L1 to L8 go with the restatement. L9 as
   Phase A settled it.
2. Each other wrong row, in the order Phase A lists them.
3. Each edit is made once, exactly, and read back.

**Verification.**

* The import comparison of Phase A, reduced to external modules, against
  the edited file: the diagram's external modules match `go list`, and no
  in-repository import is listed. No number of packages appears in "What
  it is" (a search of `docs/architecture.md` for "Thirty" and for a
  number before "packages" finds nothing).
* Each layering rule the section states names the `depguard` rule or the
  `internal/conformance` test that enforces it, and that rule or test
  exists in `.golangci.yml` or the test file.
* Each wrong row's line reads as its fix, quoted in the execution record.
* Rule 7's checks; `git diff --stat`: Markdown files only, each named in
  Phase A's wrong rows.

**Acceptance.** Every wrong row is fixed or, where it became a decision,
answered by the owner and then fixed; no other text changed.

### Phase C: missing feature descriptions

**Tier:** mainstream, medium effort (descriptions from a clear outline:
Phase A's rows give the place and the facts).

**Files:** as Phase A's missing rows name. Expected among them, by the
leads: `docs/architecture.md`'s Tooling (L13's targets and L14's
variables) and its Tree (L12's `xtermparity`); a reference list of the
environment variables `termcap` and `launch` consult, in
`docs/guides/terminal-capabilities.md` (L15); any command, flag or
settings key Phase A's inventory found undescribed.

1. Each description says what the feature does and its default, from its
   source (rule 4), in the place Phase A named, by the owner's Q4 ruling
   ("Row plus a sentence each"), which includes a reference list of the
   environment variables the library reads.
2. A description that holds a Go excerpt takes it from a package example
   or a framework example through a `<!-- from: … -->` block, as the
   guides do.
3. Each section a reader would look for is recorded in the execution
   record as an index row owed to Phase F.

**Verification.**

* The inventory of Phase A run again (a scratch script over the same
  sources): every item has a place. Expected: no "none". Seen failing
  first on a scratch copy of the documents with one added description
  removed.
* Rule 7's checks, `make docs-check` included for any new excerpt.
* `git diff --stat`: Markdown files only.

**Acceptance.** Every missing row is described where Phase A said; the
inventory has no item without a place.

*(2026-11-15, by the owner's ruling on QA2, "No; `Messages()` and doc
comments cover them": the message IDs, each package's `Msg…` constants
whose English defaults its `Messages()` returns, are not settings keys
under MADR D1 and need no reference list. In Phase A's inventory (step 4)
they are not an item, so the inventory's "no item without a place" holds
for them without a row in Phase C: a program reaches them through each
package's `Messages()` and `msgcat`, and their doc comments are Phase D's.
No row, section or index entry is owed for them; the acceptance above is as
it was written.)*

### Phase D: doc comments on exported names

**Tier:** mainstream, medium effort (each comment written from the code).

**Files:** the non-test `.go` files outside `testdata` that the
measurement names, in every module, except `glyph/glyph.go`, which 0026's
Step 2 rewrites and comments (the order above).

*(2026-11-15, by the owner's ruling on QA1, "Phase D, as its own group":
the five package comments F18 to F22 are corrected in this phase, as a
package group of their own, one commit under rule 1: `internal/cells/cells.go:2-6`,
`internal/termevent/termevent.go:7-8`, `fuzzy/doc.go:21`,
`when/when.go:29-30` and `workspace/workspace.go:16-18`, each by the fix its
row gives. The group is a Go commit, so it runs the full CI, as the
other groups do. Its files are named by those rows and are added to the
"Files" above: the measurement of step 1 lists exported names without a
comment, so it does not name a package comment that is present and
wrong. Step 3 lists the group among the others before the first edit; step
4's comment-only check, and step 5's per-group checks (`gofmt -l`, the
comment-only diff, `make pre-add-check FILES="…"` and `make docs-check`),
apply to it unchanged, each fact it writes checked against `go list` with
`GOWORK=off` and the code (rule 4). The measurement's "0 names" does not
test this group; its acceptance is that each of the five rows' lines
reads as its fix, quoted in the execution record.)*

*(2026-11-15, by the owner's ruling on QA2: the measurement's definition
already counts the message IDs. Each `Msg…` constant is an exported
top-level declaration, in a block comment or with its own, in a public
package, so step 1 names one that has no comment and the definition above
needs no change; no separate list of them is made.)*

**The definition** is MADR D1's, from the owner's Q3 ruling: every
exported top-level declaration and method its own doc comment, in every
package; every exported struct field and interface method of a type in a
public package, or of a type users configure or receive, a doc comment, or
its name in its type's doc comment. A type under `internal/` counts as one
users configure or receive when a public package exposes it (an alias, or
an exported signature or field that uses it). Not ruled, and applied as
D1 proposes: a grouped constant or variable is covered by a block comment
that describes the members as a set; a trailing line comment serves a
field or interface method.

1. **The measurement.** A one-off Go program in a scratch directory
   outside the repository, standard library only (`go/parser`,
   `go/ast`), run with `go run` there. For each package `GOWORK=off go
   list ./...` lists, it parses the non-test files and prints each
   exported name without a comment by the definition above, as
   `file:line kind name`, a count per package and in total, and the
   `internal/` types it treats as exposed. Not committed.
2. **Prove it** on a scratch export (rule 5): with one struct field's
   comment, one method's doc comment and one constant block's comment
   removed, it names exactly those (the block's members, for the block);
   and with a field's comment removed where its type's comment names that
   field, it names nothing. The failure is quoted.
3. **Size it:** run it on the tree; record the per-package counts; set the
   package groups, each one commit, and list them in the execution record
   before the first edit. A starting split, adjusted to the counts:
   `command` and `command/commandtea`; `termcap`, `termcap/…`, `termsvc`
   and `termsvc/termsvctea`; `termkey`, `inputfilter`, `keymap`,
   `lineedit`, `pick` and `ask`; the rest, `internal/` included and
   `glyph/glyph.go` left out. By the approximate count at `f093da6`: three
   methods; up to about 266 fields and interface methods in the public
   packages with no comment at all, fewer where a type's comment names
   them (not measured); and about 331 grouped constants and 4 variables to
   judge against their block comments.
4. **Write the comments,** group by group. Each comment is the Go
   convention (it begins with the name) and states what the code does,
   read from the code, its tests and its record; nothing is guessed. A
   name whose meaning cannot be established is listed in the execution
   record and left; the phase is not accepted until the owner rules on
   each. A comment inside a `// guide:` region changes that guide's
   excerpt in the same commit.
5. Per group: rule 7's checks and the checks below, then the commit.

**Verification, per group.**

* `gofmt -l <files>`: no output.
* A comment-only diff: a scratch script over `git diff --cached -U0 --
  '*.go'` that fails on any added or removed line that is neither blank
  nor a `//` comment. Expected: `0 non-comment lines`. Seen failing first
  on a scratch export with one code line changed.
* `make pre-add-check FILES="<the group's files>"`: exit 0.
* `make docs-check`: exit 0.
* The measurement, over the group's packages: `0` names.

**Verification, at the end.** The measurement over every package: total
`0` outside `glyph/glyph.go`, quoted, and that file's count recorded for
0026's Step 2.

**Acceptance.** The measurement reports 0 under the definition above
outside `glyph/glyph.go`; every commit's diff is comments only; every
group's checks passed.

*(2026-11-15, by the owner's ruling on QA1, "Phase D, as its own group":
the acceptance also holds for the package group of F18 to F22: each of the
five lines reads as its row's fix, quoted, and that commit's diff is
comments only with its checks passed. The text above is as it was
written.)*

### Phase E: `CHANGELOG.md` and the release-notes sentence

**Tier:** mainstream, medium effort.

**Files:** `CHANGELOG.md` (new), `AGENTS.md`, `docs/guides/releasing.md`,
`docs/guides/migrating.md`, `README.md`, `docs/architecture.md` (Tree).

1. `CHANGELOG.md`: a short opening (what it lists, that `Unreleased`
   collects each change in the phase that makes it, that the release step
   renames it), then `## Unreleased`, then one section for the latest
   tag, `## v2.12.0 (YYYY-MM-DD)` with the tag's date from `git for-each-ref
   --format='%(taggerdate:short)' refs/tags/v2.12.0`, in the sections of
   MADR D2, from `docs/guides/migrating.md`'s `v2.12.0` section; then a
   single pointer to `docs/guides/migrating.md` for older releases (the
   owner's Q1 ruling). Nothing older is copied. No link names the hosting
   account.
2. `AGENTS.md`, API convention 1: "Release notes list every deprecation
   and every removal" names `CHANGELOG.md`, a factual correction. No
   docs-step section is added (Q6, as applied).
3. `docs/guides/releasing.md`: before the tag, `Unreleased` becomes the
   version's entry, in the release's commit.
4. `docs/guides/migrating.md`'s opening and `README.md`'s release line link
   `CHANGELOG.md`, dividing the work by the owner's Q5 ruling
   ("Changelog lists, guide migrates"): the guide's opening says that from
   the next release it keeps only what a program must change, and links
   `CHANGELOG.md` for the rest; its existing sections stay as they are.
5. `docs/architecture.md`'s Tree gains `CHANGELOG.md`.
6. The index rows owed (`CHANGELOG.md` in "I want to…") are recorded for
   Phase F.

**Verification.**

* `git describe --tags --abbrev=0` and the first version heading under
  `## Unreleased` name the same tag (`v2.12.0` at `f093da6`, unless 0026
  has tagged since).
* A scratch comparison of the backticked names under the `### Deprecated`
  and `### Removed` headings of `docs/guides/migrating.md`'s `v2.12.0`
  section with `CHANGELOG.md`'s `v2.12.0` section: none missing (at
  `f093da6` that section has neither heading, so the comparison is empty).
  Seen failing first on a scratch copy given one deprecation the changelog
  lacks.
* `CHANGELOG.md` has exactly one released section and one link to
  `docs/guides/migrating.md` below it.
* A search of `AGENTS.md` for `CHANGELOG.md`: the convention 1 line, and
  no new section.
* Rule 7's checks; `python3 scripts/check_ci_filters.py`: exit 0 (a
  root `*.md` is docs-safe; nothing in the workflows changes).

**Acceptance.** `CHANGELOG.md` exists in D2's shape: `Unreleased`,
`v2.12.0` and one pointer; `AGENTS.md`'s release-notes sentence names it;
the release guide moves `Unreleased` before a tag.

### Phase F: index currency

**Tier:** lightweight (the rows are given by Phases C and E).

**Files:** `docs/README.md`, `docs/architecture.md` (Tree, if a guide was
added), `docs/reports/0014-REPORT-api-assessment-and-integration-research.md`
(front matter only).

1. "I want to…" rows for each row owed by Phases C and E.
2. L20: the report gains front matter with `date:` the date its body
   states ("On 2026-10-07"), confirmed against `git log --diff-filter=A
   --format=%cs --` that file; front matter only, as 0011-MADR's rule
   allows.
3. The Records table's statuses for 0026 and 0027 match their front
   matter.

**Verification.**

* `python3 -B scripts/check_records.py`: exit 0, 0 errors.
* Each row owed appears in `docs/README.md`, each link resolving (the same
  check).
* Rule 7's checks; `git diff --stat`: the files above.

**Acceptance.** Every new document and section a reader looks for is
reachable from `docs/README.md`; every record's row matches its status.

### Phase G: close-out

**Tier:** mainstream, medium effort.

1. Re-read each row of L1 to L21 and of "Findings (Phase A)" at the
   closing tree; quote the line or the evidence.
2. Re-run Phase B's external-module comparison and count search, the
   inventory, the doc-comment measurement and Phase E's comparison; quote
   each output.
3. Rule 7's checks, each exit status recorded.
4. This PLAN `complete`; its row in `docs/README.md`.

**Acceptance.** Every finding reads as fixed, or is a record or decision
row the owner has dispositioned; every one-off audit reports nothing
missing; the documents' checks pass.

## Phases at a glance

| Phase | Scope | Tier | Acceptance |
| :--- | :--- | :--- | :--- |
| A | on approval, statuses and 0026-PLAN's note; verify leads, read every living doc, inventory, defect list | strongest, high | statuses and note recorded; every lead settled; every doc read; list with the owner |
| B | wrong statements, `docs/architecture.md` first | mainstream, medium | every wrong row fixed; Markdown only |
| C | missing feature descriptions | mainstream, medium | inventory has no item without a place |
| D | doc comments on exported names, `glyph/glyph.go` left to 0026 Step 2 | mainstream, medium | measurement 0 outside `glyph/glyph.go`; comment-only diffs |
| E | `CHANGELOG.md` (`Unreleased`, `v2.12.0`, one pointer), `AGENTS.md`'s sentence | mainstream, medium | changelog in D2's shape; AGENTS names it |
| F | index currency | lightweight | every new doc and section reachable; statuses match |
| G | close-out | mainstream, medium | every finding re-read; audits report nothing missing |

*(2026-11-15, by the owner's ruling on QA1, "Phase D, as its own group":
row D also covers the five package comments F18 to F22, as a package group
of their own, with their lines quoted fixed as well as the measurement at 0
outside `glyph/glyph.go`. Row B is unchanged: no package comment is in it.
The rows above are as they were written.)*

## Verification

* Each phase's verification holds, quoted in the execution record with its
  exit status.
* Each one-off audit was seen failing on a broken scratch copy before it
  was trusted, the failure quoted.
* No `.go` change other than comments (Phase D's comment-only check); no
  change under `.github/`, `scripts/`, `tools/` or the `Makefile`.
* The documents' checks pass at every commit; the identifier scan is clean
  for every phase.

## Rollout and Rollback

* **Rollout:** one commit per phase (Phase D per group), no release. The
  owner pushes when the owner chooses; the documents-only commits run
  Docs, Phase D's run the full CI.
* **Rollback:** each commit reverts on its own. A Markdown-only commit's
  revert restores the previous text; a Phase D commit is comments only, so
  its revert changes no behaviour or API. Phase E's revert removes
  `CHANGELOG.md`; a later release step that wrote an entry into it is
  reverted first, or the entry kept by hand.
* Nothing persists outside the repository.

## What this PLAN does not do

* Add or change any gate, CI job, check script, `Makefile` target or
  workflow; commit any audit tool.
* Change code behaviour, the exported API, a golden file or a test.
* Tag, release or push.
* Rewrite a record's body, or correct an unexecuted record other than by
  listing its stale fact for the owner (0026-PLAN's dated note in Phase A
  is a note, not a correction).
* Build anything 0026 decides, or `stream/glamourmd`; comment
  `glyph/glyph.go`, which 0026's Step 2 rewrites.
* Add a docs-step section to `AGENTS.md` (Q6, as applied).
* Touch another repository.

## Findings (Phase A)

*(2026-11-15, Phase A, at `f093da6` with this pair staged. Each line
number is the file's at that tree. Classes: **wrong**, a statement the
tree contradicts; **stale**, true once and no longer, or partial where it
reads as whole; **missing**, a feature D1 asks a description of, with
none; **record** and **decision** as step 5 defines them. A stale
statement in Markdown is corrected in Phase B, as a wrong one is; the
changelog's are Phase E's.)*

### The leads, settled

| Lead | Settled | Evidence |
| :--- | :--- | :--- |
| L1 | verified, wrong | the import comparison prints `rows: 45; packages: 47; public: 37; internal: 10`; `docs/architecture.md:19` says "Thirty-one packages" |
| L2 to L8 | verified, wrong; no other row differs | the import comparison prints exactly seven rows, `launch (line 31)`, `inline/inlinetest (line 47)`, `palette (line 48)`, `workspace (line 49)`, `command (line 52)`, `termcap (line 58)` and `internal/lineinput (line 70)`, each lacking only the imports L2 to L8 name; no external module differs in any row |
| L9 | verified, wrong | `docs/architecture.md:560-561` lists "scrolling text and Markdown" among the panes not here; `stream/streamtea/pane.go:17-26` is a `stream.Doc` in a workspace pane that scrolls and follows its tail, added in `4413f94` (2026-11-03) and released in `v2.10.0`, while the sentence last changed in `bfe819d` (2026-10-25). The log tail and the metrics view are still not here; `ask/asktea.Dialog` is a question in an overlay, not a dialog widget, so "dialog" stands |
| L10 | refuted | `AGENTS.md:4-5`: "A repository-local `CLAUDE.md` … wins only where it is more specific", a condition |
| L11 | refuted, in every file | read in context: `docs/guides/releasing.md:6-7` says "the planned `stream/glamourmd`", and its other uses (`:24-28`, `:130-131`, `:151-157`, `:200-204`) are that procedure's examples; the `glamour` depguard rule `docs/guides/releasing.md:55-57` cites exists (`.golangci.yml:69-76`); `README.md:182-183` says "the planned glamour one" and `docs/guides/streaming-content.md:57-58` "the planned glamour adapter's"; `AGENTS.md:84`, `docs/architecture.md:199-202` and `docs/glossary.md:194-196` as the table says |
| L12 | verified, missing | `lineedit/testdata/xtermparity/main.go:1-9`, a program with its own `xterm.mod` and `xterm.sum`, writes `lineedit/testdata/xterm-parity.json`, which `lineedit/parity_test.go:11` reads; neither is named in the Tree (`docs/architecture.md:219-327`) or any other living document |
| L13 | verified, missing | `docs/architecture.md:376-379` names `test-sum`, `fmt`, `tidy` and `help` and describes none; `all` (`Makefile:27`, which runs `help`) is named nowhere. For the description: `fmt` and `tidy` run at the root only, not per module (`Makefile:55-56`, `:87-88`), and `test-sum` runs gotestsum over `./...` at the root, without `GOWORK=off` (`:48-53`) |
| L14 | verified, missing | `Makefile:10-12`: `GOLANGCI_LINT` defaults to `golangci-lint` in `GOPATH`'s `bin`; `GOVULNCHECK` and `GOTESTSUM` to the first found in `GOBIN`, `GOPATH`'s `bin`, then `PATH` |
| L15 | verified, missing; the lead corrected | F26 and F27: `termcap` reads 30 names through its `Env`; `launch` 9 by name, the one `Config.NoInputEnv` names, and `colorprofile`'s through it; `tuitest` one, which is described; `termsvc` and `theme` read none (no `Env` lookup and no such literal in their non-test files); `AAAA` (`termcap/queries.go:300`), `ANSI` and `ANSI256` (`termcap/colorprofile.go:33`) are not variables |
| L16 | not a defect (D1) | the survey's B2 and B3 names were a prompt for the inventory only; no row below comes from a name's absence from prose |
| L17 | still a lead | Phase A does not measure doc comments; Phase D's `go/ast` measurement, proved on a scratch export, settles it before its first edit |
| L18 | verified, missing | `git ls-files CHANGELOG.md` prints nothing |
| L19 | verified, stale | `AGENTS.md:147-149` |
| L20 | verified, stale | the report opens with its `#` title, where the other three open with front matter holding `date:`; its body says "On 2026-10-07" (`docs/reports/0014-REPORT-api-assessment-and-integration-research.md:10`) |
| L21 | verified | `git tag` lists 39 tags; `docs/guides/migrating.md` has 21 `## Migrating to` sections. Not a defect by Q1: Phase E's pointer covers the older tags |

### The defects

| # | Where | Class | The document says | Evidence | Fix | Phase |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| F1 | `docs/architecture.md:19` | wrong | "Thirty-one packages" | L1 | removed with the restatement (D6) | B |
| F2 | `docs/architecture.md:31` | wrong | `launch`'s imports, without `command/commandtea` | L2 | the in-repository imports leave the diagram (D6) | B |
| F3 | `docs/architecture.md:47` | wrong | `inline/inlinetest`'s, without `internal/sanitize` | L3 | as F2 | B |
| F4 | `docs/architecture.md:48` | wrong | `palette`'s, without `msgcat` | L4 | as F2 | B |
| F5 | `docs/architecture.md:49` | wrong | `workspace`'s, without `command/commandtea` and `msgcat` | L5 | as F2 | B |
| F6 | `docs/architecture.md:52` | wrong | `command`'s, without `msgcat` | L6 | as F2 | B |
| F7 | `docs/architecture.md:58` | wrong | `termcap`'s, without `msgcat` | L7 | as F2 | B |
| F8 | `docs/architecture.md:70` | wrong | `internal/lineinput`'s, without `internal/sanitize` | L8 | as F2 | B |
| F9 | `docs/architecture.md:560-561` | wrong | a scrolling text and Markdown pane is not here | L9 | name `stream/streamtea`'s `DocPane` as that pane; keep the log tail, the metrics view, the dialog and toast and the help-footer pane | B |
| F10 | `docs/architecture.md:442-444` | wrong | `TestNarrowClosures` covers "`termkey` and `safetext` today" | `narrow` holds `termkey`, `safetext` and `msgcat` (`internal/conformance/charmfree_test.go:211-215`) | name `msgcat`, as `docs/architecture.md:366-372` does | B |
| F11 | `docs/glossary.md:41` | wrong | `stream.Scanner` reserved, "unexported until `Doc` is built on it" | exported (`stream/scanner.go:110`); `Doc` built in `v2.10.0` (`docs/guides/migrating.md:119-120`) | move it to "Built from those records" | B |
| F12 | `docs/glossary.md:42` | wrong | `workspace.FocusFallback` and `FocusFallbackFunc` reserved | built in `v2.8.0` (`workspace/workspace.go:378`; `docs/guides/migrating.md:234-235`) | as F11 | B |
| F13 | `docs/glossary.md:154-155` | wrong | 0009-MADR "is accepted but unbuilt" | its packages are built, `v2.10.0` (`docs/guides/migrating.md:93-143`; `docs/README.md:30`) | built through `v2.10.0`, `stream/glamourmd` still planned | B |
| F14 | `docs/glossary.md:175-190` | stale | 0009's names, under "Names the planned records must not take", which "its PLAN builds" | each name resolves to a declaration in its package (the name check) | list them as built, each with its one meaning | B |
| F15 | `docs/guides/defaults.md:165` | wrong | "folds at 100, 70 and 16 columns" | the third is rows (`layout/presets.go:94-97`; `docs/guides/building-workspaces.md:76-79`) | "16 rows" | B |
| F16 | `docs/guides/line-mode.md:27-28` | stale | 0023-MADR "with its amendments A1 to A11" | A12 to A15 exist (`docs/decisions/0023-MADR-line-mode-components.md:1676`, `:1956`, `:2037`, `:2184`), and the guide describes A14's and A15's behaviour (`docs/guides/line-mode.md:411-421`) | "A1 to A15" | B |
| F17 | `AGENTS.md:34-39` | stale | the narrow cores are `termkey` and `safetext`, by `termkeystd`, `safetextstd` and `TestNarrowClosures` | `msgcat` is a third (`.golangci.yml:182`, `msgcatstd`; `internal/conformance/charmfree_test.go:214`) | name `msgcat` and `msgcatstd`, a factual correction (Q6) | B |
| F18 | `internal/cells/cells.go:2-6` | wrong | the only package that imports ultraviolet, refused anywhere else | `internal/termevent` and `internal/termevent/termeventtest` import it (`go list`), as `.golangci.yml:80` allows and `docs/architecture.md:354-356` says | one of the two importers, with `internal/termevent` | F39 |
| F19 | `internal/termevent/termevent.go:7-8` | wrong | it and `internal/cells` are the only importers | `internal/termevent/termeventtest` imports it too (`go list`; `docs/architecture.md:62-63`) | add `termeventtest`, beneath it | F39 |
| F20 | `fuzzy/doc.go:21` | wrong | "imports only the standard library and x/ansi" | `go list`: `internal/enum` too | the standard library, `internal/enum` and x/ansi | F39 |
| F21 | `when/when.go:29-30` | wrong | "The package imports only the standard library." | `go list`: `internal/enum` too, as `docs/architecture.md:135-139` says | as F20, less x/ansi | F39 |
| F22 | `workspace/workspace.go:16-18` | stale | the optional interfaces add a title, a badge, focus callbacks, a minimum size, a cursor, help bindings and change tracking | also `EscConsumer` (`workspace/workspace.go:101`), `KeyContexter` and `CommandConsumer` (`workspace/bindings.go:39`, `:60`), `Contexter` (`workspace/context.go:11`) and `PlainViewer` (`workspace/plain.go:21`) | name them, or "among them" | F39 |
| F23 | `docs/architecture.md:219-327` (Tree) | missing | — | L12 | a row for `lineedit/testdata/xtermparity` and `xterm-parity.json`: what writes the reference, and its command (`lineedit/testdata/xtermparity/main.go:9`) | C |
| F24 | `docs/architecture.md:376-379` (Tooling) | missing | — | L13 | a sentence each for `test-sum`, `fmt`, `tidy`, `help` and `all` | C |
| F25 | `docs/architecture.md` (Tooling) | missing | — | L14 | a sentence each for `GOLANGCI_LINT`, `GOVULNCHECK` and `GOTESTSUM`, with its default | C |
| F26 | `docs/guides/terminal-capabilities.md` | missing | — | `termcap` reads, through its `Env`: `TERM`, `TERM_PROGRAM`, `TERM_PROGRAM_VERSION`, `TERM_FEATURES`, `TERM_SESSION_ID`, `TERMINAL_EMULATOR`, `LC_TERMINAL`, `LC_TERMINAL_VERSION`, `COLORFGBG`, `TMUX`, `STY`, `ZELLIJ`, `NVIM`, `VIM_TERMINAL`, `INSIDE_EMACS`, `MSYSTEM`, `WSL_DISTRO_NAME`, `WSL_INTEROP`, `CURSOR_TRACE_ID`, `VSCODE_GIT_ASKPASS_MAIN`, `SSH_TTY`, `SSH_CONNECTION`, `KITTY_WINDOW_ID`, `GHOSTTY_RESOURCES_DIR`, `WEZTERM_PANE`, `ALACRITTY_WINDOW_ID`, `KONSOLE_VERSION`, `VTE_VERSION`, `TERMINATOR_UUID` and `WT_SESSION` (`termcap/env.go:58-60`, `:94`; `termcap/identity.go:143-271`; `termcap/queries.go:113-116`, `:332-343`); no living document names one, but `TERM` for `launch` (`docs/guides/commands.md:677`). `WithAppearanceEnv`'s pair is described (`docs/guides/terminal-capabilities.md:139`) | the reference list Q4 asks for: each name, and what `termcap` learns from it | C |
| F27 | `docs/guides/commands.md:669-698` | missing | — | `launch` reads `COLUMNS`, a plain decision's width when `Out` is no terminal (`launch/decide.go:209-213`, `:261-267`), `LC_ALL`, `LC_CTYPE` and `LANG`, the glyph tier (`launch/locale.go:15-35`), and `CLICOLOR_FORCE`, with `COLORTERM` through `colorprofile.Env` (`launch/colour.go:11-38`); the section names `CI`, `TERM`, `NO_COLOR` and `FORCE_COLOR` only | each with its rule, here or in F26's list | C |
| F28 | `docs/architecture.md` (Tooling) | missing | — | `GO`, the go command, in four scripts (`scripts/go-apicheck.py:40`, `scripts/go-fuzz.py:30`, `scripts/go-modules.py:23`, `scripts/go-examples.py:345`); `APIDIFF` and `APICHECK_ALLOW` (`scripts/go-apicheck.py:37-38`); `EXAMPLES_ROOT` (`scripts/go-examples.py:44`); `GOLANGCI_LINT` for the pre-add check (`scripts/go-precheck.py:71`); named in no document | a sentence each, with its default | C |
| F29 | `docs/architecture.md` (Tooling) | missing | — | `--root`, the tree to check (`scripts/check_records.py:230`, `scripts/check_disclosure.py:112`, `scripts/check_ci_filters.py:397`); `go-fuzz.py`'s `-t`, `-z`, `-m` and `-l`, and its one-package form (`scripts/go-fuzz.py:11-28`), where the documents give `-a` and `FUZZTIME` only (`docs/architecture.md:449-452`) | a sentence each, with its default | C |
| F30 | `docs/architecture.md` (Tooling) | missing | — | `tools.verify`'s flags: `gates` `--label`, `--out`, `--only`, `--skip`, `--no-history`, `--list`; `mutate` `--work`, `--keep`; `remote` `--label`; `scan` `--base` (`tools/verify/__main__.py:87-111`); `AGENTS.md:444-450` names the commands and `:458-461` `--config` | a sentence each, with its default | C |
| F31 | `docs/architecture.md` (Tooling) | missing | — | the config's keys, `hosts` with each host's `kind`, `ssh`, `path_prepend` and `python`, and `identifiers` (`tools/verify/verify.example.json`), described as "the hosts and the identifier patterns" (`AGENTS.md:458-463`) | a sentence each | C |
| F32 | `docs/guides/keymaps.md:92-127` | missing | — | the file's `when` and `args` members (`keymap/vscode.go:325-329`); the section describes `key`, `command`, `context` and the `-` removal | a sentence each | C |
| F33 | `docs/guides/building-workspaces.md:414-420` | missing | — | the state's JSON members `version`, `resize`, `hidden` and `zoom` (`layout/state.go:20-34`) | a sentence each | C |
| F34 | `docs/guides/commands.md:337-348` | missing | — | `workspace.Commands` also returns `workspace.focus.index` and `workspace.resize.direction` (`workspace/commands.go:49`, `:157`, `:174`), with no slash name; only `docs/guides/building-workspaces.md:283-284` names them | a row each | C |
| F35 | `docs/guides/line-mode.md` | missing | — | `lineedit`'s 22 command IDs (`lineedit/keys.go:14-35`) and `pick`'s 6 (`pick/keys.go:15-20`): the guide names their scopes and some keys, no ID; `docs/guides/streaming-content.md:187-194` is the shape such a reference takes | a table each: the ID, its default keys, what it does | C |
| F36 | repository root | missing | no `CHANGELOG.md` | L18 | Phase E, step 1 | E |
| F37 | `AGENTS.md:147-149` | stale | "Release notes list every deprecation and every removal" | L19 | names `CHANGELOG.md` | E |
| F38 | `docs/reports/0014-REPORT-api-assessment-and-integration-research.md:1` | stale | no front matter | L20 | `date: 2026-10-07`, front matter only | F |
| F39 | F18 to F22 | decision | — | they are package comments in Go files: Phase B's acceptance is Markdown only, and Phase D's scope is the doc comments its measurement names | see "Questions for the owner (Phase A)" | owner |
| F40 | the message IDs | decision | — | each package's `Msg…` IDs are the keys of a program's catalogue; the guides point at each `Messages()` (`docs/guides/building-workspaces.md:307-308`, `docs/guides/commands.md:314-316`) and list none; step 4's list of settings keys does not name them | see "Questions for the owner (Phase A)" | owner |

*(2026-11-15, the owner's rulings on QA1 and QA2, relayed by the
coordinator. F18 to F22: the phase is D, as a package group of its own
(QA1, "Phase D, as its own group"), not B, which the Phase column above
gave as F39. F39: answered by that ruling; it is no longer a decision row.
F40: answered by QA2, "No; `Messages()` and doc comments cover them": the
message IDs are not settings keys under MADR D1, no Phase C row is owed,
and their doc comments are Phase D's; it is no longer a decision row. The
rows are as they were written.)*

**Totals.** 40 rows: wrong 18 (F1 to F13, F15, F18 to F21), stale 6
(F14, F16, F17, F22, F37, F38), missing 14 (F23 to F36), decision 2 (F39,
F40), record 0. By phase: B 17 (F1 to F17), C 13 (F23 to F35), E 2 (F36,
F37), F 1 (F38), and 7 for the owner first (F39, F40, and F18 to F22,
whose phase F39 settles). Phase D's own count is its measurement's (L17).

*(2026-11-15, after QA1 and QA2: of the 7 "for the owner first", F18 to
F22 go to Phase D as a package group, which makes Phase D's 5 rows beside
its measurement's own count; F39 and F40 are answered, with no row left
for the owner. The 40 rows and the class counts are unchanged, and B 17, C
13, E 2 and F 1 are unchanged. The totals above are as they were written.)*

**Record rows.** None. The bodies of the unexecuted records (0009-PLAN
Step 7, 0022-PLAN Phase 4, 0026) are outside Phase A's "Files read", and
were not read; no stale fact in them was met on the way.

### Questions for the owner (Phase A)

**QA1 (F39). Which phase corrects the five package comments, F18 to F22?**

* **(a) Phase D, as a package group of its own,** under Phase D's
  comment-only check and its full CI. Cost: Phase D's scope widens from
  "doc comments on exported names" to package comments too, by a dated
  amendment of this PLAN; the five wait for Phase D.
* (b) Phase B, which then edits Go comments beside Markdown. Cost: B's
  acceptance ("Markdown files only") and its Docs-only CI change, and a B
  commit runs the full CI.
* (c) Leave them to the next code change in each package, by D5's docs
  step. Cost: four wrong statements stay in `go doc` for an unknown time.

Recommended: (a). Doing nothing leaves `go doc` saying that one package
imports ultraviolet where three do, and that `fuzzy` and `when` import
less than they do.

*Answer, 2026-11-15, the owner's ruling, relayed by the coordinator:
"Phase D, as its own group". That is option (a). It follows that the five
package comments F18 to F22 are corrected in Phase D, as a package group of
their own, under its comment-only check and its full CI; Phase D's scope is
widened to them by the dated annotations under "In scope", Phase D's
files, its acceptance and "Phases at a glance"; Phase B stays Markdown
only; F18 to F22 and F39 are annotated in "Findings (Phase A)". No
amendment of the MADR is needed (see the execution record).*

**QA2 (F40). Are the message IDs settings keys under D1?**

* **(a) No:** a program sets them through `msgcat`, each package's
  `Messages()` lists them with their English defaults, and their doc
  comments are Phase D's. Cost: a reader of the guides reaches each ID
  through `go doc`, not a page.
* (b) Yes: a reference list of every package's IDs and defaults in
  `docs/guides/defaults.md` or a new guide. Cost: about seventy rows that
  repeat `Messages()` and drift when a message is added.

Recommended: (a). Doing nothing leaves Phase C's acceptance ("every item
has a place") open to either reading.

*Answer, 2026-11-15, the owner's ruling, relayed by the coordinator: "No;
`Messages()` and doc comments cover them". That is option (a). It follows
that the message IDs are not settings keys under MADR D1: Phase C owes no
reference list for them, and a program reaches each through `msgcat` and
that package's `Messages()`, with the doc comments Phase D's definition
already asks of each `Msg…` constant. Phase C's acceptance and F40 are
annotated.*

**QA3 (deviation A-2). How does rule 7 lint a record?**

* **(a) Against its base:** a renamed copy of the record before the phase
  and one after it, linted alike, and no issue the phase adds; the
  summary read as markdownlint-cli2 prints it. Cost: a base copy each
  time, and a record's older issues stay.
* (b) A renamed copy with MD004 off, ending `Summary: 0 issues`. Cost:
  the copy's configuration differs from the repository's by one rule, and
  0026-PLAN's older MD029 still fails it.
* (c) Not at all: the configuration skips records by design, and `make
  check-records` covers their names, index and links. Cost: a record's
  Markdown errors go unseen.

Recommended: (a). Doing nothing leaves rule 7 unmeetable by any phase
that edits a record, and each would record this deviation again.

*Answer, 2026-11-15, the owner's ruling, relayed by the coordinator:
"Compare against the base copy". That is option (a). It follows that rule 7
lints a record as a renamed copy before the phase and one after it, alike,
and the phase adds no issue, the summary read as markdownlint-cli2 prints
it; rule 7 is annotated, and deviation A-2 is resolved.*

## Execution record

Not started. The owner has not approved this PLAN or said when it starts.
The MADR's Q1 to Q5 and Q7 are ruled and Q6 applied (2026-11-14); no
question is open.
*(2026-11-15: started; the entries below supersede this paragraph.)*

### 2026-11-15: approval

The owner's approval, relayed by the coordinator: "Approve, one phase at a
time". The MADR is `accepted` and this PLAN `in-progress`; Phase A runs
now, and the PLAN pauses after each phase for the owner's word.

### 2026-11-15: Phase A

**Provenance.** Run by a strongest-tier model at high effort, as the
phase's row prices it, on the coordinator's brief. Base: `f093da6`
(`main`, level with `origin/main`), with this pair and its two index rows
staged and not committed.

**Step 0.** The MADR's front matter `accepted`, dated 2026-11-15; this
PLAN `in-progress`, dated, with a dated note under the title;
[0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md) the dated note
after its ruling note, worded as step 0 gives it, and nothing else; the
two 0027 rows of `docs/README.md` `accepted` and `in-progress`, 0026's
unchanged.

**Step 1.** `git rev-parse --short HEAD`: `f093da6`. `git describe
--tags --abbrev=0`: `v2.12.0`. `git status --short`: the three staged
files of this pair, where the step expects nothing (deviation A-1). `git
log --oneline f093da6..HEAD`: empty, so no commit since `f093da6` touches
a living document, and L1 to L21 stand as surveyed.

**Step 2.** The leads, in "Findings (Phase A)" above. The import
comparison, over `GOWORK=off go list` with each package's import path and
imports, and the diagram's rows:

```text
DIFF    command (line 52): lacks in-repo ['msgcat']; names extra in-repo []; lacks external []; names extra external []
DIFF    inline/inlinetest (line 47): lacks in-repo ['internal/sanitize']; names extra in-repo []; lacks external []; names extra external []
NO ROW  internal/conformance
DIFF    internal/lineinput (line 70): lacks in-repo ['internal/sanitize']; names extra in-repo []; lacks external []; names extra external []
DIFF    launch (line 31): lacks in-repo ['command/commandtea']; names extra in-repo []; lacks external []; names extra external []
DIFF    palette (line 48): lacks in-repo ['msgcat']; names extra in-repo []; lacks external []; names extra external []
DIFF    termcap (line 58): lacks in-repo ['msgcat']; names extra in-repo []; lacks external []; names extra external []
NO ROW  tuitest/internal/clash
DIFF    workspace (line 49): lacks in-repo ['command/commandtea', 'msgcat']; names extra in-repo []; lacks external []; names extra external []
rows: 45; packages: 47; public: 37; internal: 10; differing rows: 7
```

Exactly L2 to L8, and 37 public packages, as expected. The two `NO ROW`
packages hold tests only, and the Tree (`docs/architecture.md:304-308`)
and Tooling (`docs/architecture.md:410-448`) describe them; the package table's 45 rows are
the other 45 packages.

**Step 3.** Every file read in full, in the order listed, with `wc -l`:
`README.md` 214; `AGENTS.md` 528; `docs/README.md` 170;
`docs/architecture.md` 577; `docs/glossary.md` 215; the ten guides,
`building-workspaces.md` 495, `command-palette.md` 311, `commands.md`
877, `defaults.md` 408, `keymaps.md` 237, `line-mode.md` 526,
`migrating.md` 938, `releasing.md` 226, `streaming-content.md` 322 and
`terminal-capabilities.md` 417; the package comments of all 47 packages,
47 files, 1,055 lines of comment (`internal/conformance`'s in its test
file); `.claude/rules/madr-and-plan-skill.md` 28,
`.grok/rules/madr-plan-before-mutating-work.md` 29, `.opencode/rules.md`
27 and `opencode.json` 4; the `Makefile` 154; the `--help` of
`check_records.py`, `check_disclosure.py`, `check_ci_filters.py`,
`go-modules.py`, `go-examples.py` and `go-fuzz.py`, and of `python3 -m
tools.verify` and its five commands; `testdata/frameworks/cases.txt` 77.
`go-apicheck.py`, `go-precheck.py` and the shim `go-precheck.sh` have no
`--help`: an argument runs the gate, so their usage docstrings were read
instead.

Each claim was checked against its source:

* **Names.** A scratch check resolved every backticked Go name in those
  Markdown files against a scratch dump of every exported declaration,
  method, field and interface method (`go/parser`, non-test files). 199
  spans did not resolve; each was read in context, and each is a test or
  example function, another module's name (`tea.View`'s fields, Cobra's,
  pflag's, urfave's, x/term's), a name a record reserves for 0026, or a
  removed name the text says was removed. None is a defect.
* **Numbers and defaults.** Each default a guide states was compared with
  its constant: the workspace's 80 by 24, `MaxSide` and `MaxCells`, the
  folds 100, 70 and 16; the registry's 1 MiB, the command files' 256 KiB,
  1,000 files and 8 levels, the 128-byte ID, `when`'s 4,096 bytes and
  depth 64; the exit statuses of `command` and `launch`; the palette's 200
  hits, 72 by 16, 5,000 and 200,000 paths, 7 days and 500 recents, 1 MiB
  and 1,024 bytes; the keymap's one second; the line mode's 50 ms, 150 ms,
  2,048 bytes, 10 rows, 4,096 runes, 1,000 entries, 4 MiB, 10,000 rows and
  150 ms; the stream's 2 KiB, 512 KiB, 10,000 lines, 4 KiB, 64 KiB, 20 and
  20 lines, 50,000 rows and 8,192 columns; the frame's 60 fps, 8 and 120
  ms, 2, 40 ms and 250 ms, 64 and 300 ms, 100 ms, 83 ms, 5 s and 100 ms;
  the input filter's 16 ms and 500 ms; `termcap`'s 2 s and 80 columns;
  `termsvc`'s 80 and 240 cells, 5 s, 100,000 bytes, 240 characters, 5 s,
  and its link schemes; `fuzzy`'s 512, 0.3, 2,000 and 256. Each matched
  but F15's.
* **Versions and pins.** The requirements in `docs/architecture.md:331-338`
  equal `go.mod`'s; `go 1.27.2`; 21 linters; golangci-lint v2.14.0,
  govulncheck v1.8.0, shellcheck v0.11.0 and actionlint v1.7.12, the
  runners and the five `paths-ignore` patterns equal `ci.yml`'s; the
  Charm-free cores equal `cores`; the depguard rules named exist.
* **Records' statuses.** A scratch comparison of each record's front
  matter `status` with its `docs/README.md` row: `rows: 61; mismatches:
  0`.
* **Links.** `make check-records`, below.
* **Examples.** `cases.txt` against `docs/guides/commands.md:571-578`,
  and the guides' excerpts by `make docs-check`, below.

**Step 4. The inventory,** from the scratch dump of the API, the
`Makefile`, the scripts' and `tools.verify`'s argument parsers, and a scan
of the non-test Go files for environment variables:

| Kind | Items | Described where | Without a place |
| :--- | :--- | :--- | :--- |
| packages | 47: 37 public, 10 internal | each a diagram and table row in `docs/architecture.md` and a package comment; the two test-only packages in its Tree and Tooling | none (F18 to F22 are wrong statements in five package comments) |
| command IDs | 59: `command`'s 3, `workspace`'s 15, `palette`'s 7, `pick`'s 6, `lineedit`'s 22, `stream/streamtea`'s 6 | `docs/guides/commands.md:186-193` and `:337-348`; `docs/guides/command-palette.md:27-37`; `docs/guides/streaming-content.md:187-194` | F34, F35 |
| flags | `launch`'s `--mode`, `--tui`; `-tuitest.update`; the scripts'; `tools.verify`'s | `docs/guides/commands.md:716-720`; `AGENTS.md:410`; `docs/architecture.md:255-256`, `:449-452`, `:492-493`; `AGENTS.md:89-90`, `:444-461` | F29, F30 |
| environment variables | the library's: `termcap` 30; `launch` 9, `Config.NoInputEnv`'s and `colorprofile`'s; `tuitest`'s `TUITEST_UPDATE`; the scripts': `GO_PRECHECK_SKIP_*`, `APIDIFF_VERSION`, `XDG_CONFIG_HOME`, `GO`, `APIDIFF`, `APICHECK_ALLOW`, `EXAMPLES_ROOT`, `GOLANGCI_LINT` | `docs/guides/commands.md:669-693` (`CI`, `TERM`, `NO_COLOR`, `FORCE_COLOR`); `AGENTS.md:341`, `:359`, `:365`, `:410-412`, `:459`; `docs/architecture.md:474` | F26, F27, F28 |
| settings keys | command files' front matter, 9 keys; `keybindings.json`, 5 members; the layout state, 4; the recents' JSON; `tools/verify`'s config | `docs/guides/commands.md:254-258`; `docs/guides/keymaps.md:94-123`; `docs/guides/command-palette.md:232-235` | F31, F32, F33; F40 asks about the message IDs |
| `make` targets | 20, `all` among them | `docs/architecture.md:380-409`, `:449-493`; `AGENTS.md:326-371`, `:419-423` | F24 |
| `make` variables | 8 set with `?=` | `FILES` (`AGENTS.md:330`), `PYTHON` (`:435`), `MARKDOWNLINT` (`docs/architecture.md:384`), `FUZZTIME` (`:450`), `APIDIFF_VERSION` (`:474`) | F25 |
| `testdata` programs and data | `testdata/frameworks`' five programs, `cases.txt` and `go.mod.tmpl`; `lineedit/testdata/xtermparity` and `xterm-parity.json` | `docs/architecture.md:309-312`; `docs/guides/commands.md:571-578`; `docs/guides/line-mode.md:31-34` | F23 |

The tests' fixtures and fuzz corpora under the packages' `testdata` are
inputs to their tests, not features, and are not inventoried.

**Step 5.** "Findings (Phase A)" above: 40 rows, each assigned.

**Step 6.** The list goes to the owner with this entry; QA1 to QA3 are
answered before Phases B to F run against it.

**Verification.**

* The import comparison, as quoted. Seen failing first on a scratch copy
  of `docs/architecture.md` with `stream` removed from the
  `stream/streamtest` row: it printed `DIFF    stream/streamtest (line
  39): lacks in-repo ['stream']; …`, eight differing rows and exit 1.
* The scratch API dump, seen failing first on a scratch export with
  `ScrollUpID` removed from `stream/streamtea/keys.go` and `ZELLIJ`
  lower-cased in `termcap/identity.go`: its command-ID and
  environment lines lost exactly those two, and nothing else changed but
  the moved line numbers.
* The name check, seen failing first on a scratch file naming
  `stream.NoSuchName` and `workspace.Workspace.NoSuchMethod` beside three
  real names: `unresolved: 2`, exit 1.
* The status comparison, seen failing first on a scratch copy of `docs/`
  with 0025-PLAN's status changed to `in-progress`: one `MISMATCH` line,
  naming that record, `front=in-progress` and its row's cell, `rows: 59;
  mismatches: 1`, exit 1. On the staged tree: `rows: 61; mismatches: 0`,
  exit 0.
* Every lead has a status, and every file in "Files read" is in step 3's
  list with its line count.
* `python3 -B scripts/check_records.py`: `check-records: 61 record(s), 79
  Markdown file(s), 0 error(s)`, exit 0.
* The documents' checks, after the last edit: `make check-records`
  (`check-records: 61 record(s), 79 Markdown file(s), 0 error(s)`), `make
  docs-check` (`examples: 20 excerpt(s) checked`, `examples: clean`, `ok
  github.com/maccavelli/go-tui-lib/v2/internal/conformance`) and `make
  disclosure-check` (`check-disclosure: 1208 tracked file(s), 0
  finding(s)`), each exit 0. markdownlint by the local
  `markdownlint-cli2` v0.23.2 with the repository's configuration, over
  the non-record Markdown: `Linting: 25 files`, `Summary: 0 issues in 0
  files`, exit 0. Over renamed copies of the three records this phase
  edits: exit 1, `Summary: 327 issues in 3 files`, 320 of them MD004 (the
  `*` list marker the records use) and 7 MD029 in 0026-PLAN, which a copy
  of 0026-PLAN at `HEAD` shows too (7 MD029, 115 MD004); no other rule
  fires (deviation A-2). The same run over a copy given two extra blank
  lines, a heading ending in a full stop and trailing spaces reported
  MD012, MD022, MD026 and MD009 for them. `python3 -m tools.verify scan
  --base HEAD`, with the local config's identifier patterns: `scan: clean
  (5 pattern(s))`, exit 0.
* A scratch check of every `path:line` this entry and "Findings (Phase
  A)" cite, against the staged tree, seen failing first on a planted
  out-of-range citation: every citation in range.
* `git diff --cached --stat`: this PLAN, the MADR, 0026-PLAN and
  `docs/README.md`, nothing else; 0026-PLAN's diff is the note alone.

**Deviations.**

* **A-1 (2026-11-15).** Step 1 expects an empty `git status --short`; it
  showed this pair and its index rows, staged and not committed, as the
  coordinator's brief said before the phase began. The phase adds to that
  staged set. No other effect.
* **A-2 (2026-11-15).** Rule 7 expects markdownlint to end `Summary: 0
  error(s)`, a record linted through a renamed copy. markdownlint-cli2
  v0.23.2 prints `Summary: N issues in M files`; and the repository's
  configuration pins the `-` list marker (MD004), where every record uses
  `*`, as the records' own convention has it, so a record's copy never
  lints clean. Found at this phase's checks; the phase's edits add no
  issue (above). Not resolved here: it is QA3, and rule 7 stands as
  written until the owner rules.

  *(2026-11-15, resolved: the owner ruled on QA3, "Compare against the
  base copy", and rule 7 is annotated to say how a record is linted. The
  entry above is as it was written. No file's scope changed.)*

**No heavy run.** Phase A ran none: `make docs-check` runs one test of
one package, which rule 9 allows on the session host.

**Not verified.**

* Whether `docs/guides/*.md`'s statements about other programs hold:
  Bubble Tea's own queries and its 50 ms, terminals' behaviours (kitty,
  iTerm2, JetBrains, Apple Terminal, tmux, ConPTY), the measured
  benchmarks in `docs/guides/migrating.md`, and gobble-cli's names. Each
  was read; none was run.
* The behaviour each guide describes beyond the names and numbers above:
  read against the code's doc comments, not by running it.
* The doc-comment counts (L17), which Phase D measures.
* The bodies of the records, outside the list.

**Files.** `docs/decisions/0027-PLAN-keep-docs-accurate-and-current.md`,
`docs/decisions/0027-MADR-keep-docs-accurate-and-current.md` (front
matter), `docs/decisions/0026-PLAN-agent-widgets.md` (the note),
`docs/README.md` (the two 0027 rows).

Staged, not committed; the coordinator commits and pushes after the
disclosure guard; the plan pauses until the owner says to continue.

### 2026-11-15: the owner's answers to QA1, QA2 and QA3

**Provenance.** Written by a mainstream-tier model at medium effort, on
the coordinator's brief, from the owner's three rulings, relayed by the
coordinator. A records-only change: no phase ran. Base: `fc55005` (`main`,
clean).

**The answers,** each a label the owner chose from the options of
"Questions for the owner (Phase A)". QA1 (F39): "Phase D, as its own
group", option (a). QA2 (F40): "No; `Messages()` and doc comments cover
them", option (a). QA3 (A-2): "Compare against the base copy", option (a).

**Annotated, each by a dated note that leaves the text as written:** the
answer under each of QA1, QA2 and QA3 in "Questions for the owner (Phase
A)"; rule 7 (QA3); the "In scope" table, Phase D's row and Phase D's
files, steps and acceptance (QA1, and QA2 for the measurement, which
already counts the `Msg…` constants as exported declarations); Phase C's
acceptance and inventory (QA2); "Phases at a glance" (QA1); the defects
table's rows F18 to F22, F39 and F40, and the totals line, whose "7 for the
owner first" becomes none (QA1, QA2); deviation A-2 (QA3, resolved);
`docs/README.md`'s row for this PLAN.

**The MADR needed no amendment.** The three answers change no decision and
contradict no fact it asserts. D1 asks a doc comment on every exported
name and a description of every package, command, flag, environment
variable, settings key and make target: QA2 rules that the message IDs fall
under the first (their constants' doc comments) and are not a settings key,
which D1 does not define, so D1 is applied, not changed. QA1 places five
package comments in a phase, which D1 and the MADR's "Neutral" consequence
(Phase D adds comments to Go files and runs the full CI) already allow. QA3
is a rule of this PLAN's, not a decision of the MADR. The MADR is
untouched.

**Verification.** Rule 7 as ruled, for each record edited: a renamed copy
of it at `fc55005` (the base) and one as left, linted alike by
markdownlint-cli2 `v0.23.2` with the repository's own configuration. Only
this PLAN is edited among the records. Base: `Summary: 86 issues in 1
file`, all 86 MD004 (the `*` list marker). After: the same, 86 MD004 and no
other rule, so the edit adds no issue. `make check-records`, `make
docs-check` and `make disclosure-check`: each exit 0 after the last edit.
Every `path:line` the new notes cite is one of Phase A's rows, checked
against the staged files, and no note sits inside a quotation.

**No question is open.** The plan stays paused until the owner's word for
the next phase.
