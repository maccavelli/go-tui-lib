---
status: accepted
date: 2026-11-15
decision-makers: the repository owner
consulted: "a read-only documentation survey of this repository at f093da6 (main, two docs-only commits past v2.12.0) by a mainstream-tier agent at medium effort, 2026-11-14, relayed by the coordinator; the coordinator's re-check of the survey's row A1; this record's own read-only checks at f093da6 (go list, a line-based count of doc comments, searches of the living documents); the owner's rulings of 2026-11-14, relayed by the coordinator"
informed: the other Go repositories the owner's mandate covers, each of which takes its own record
---
# Keep go-tui-lib's documents accurate, comprehensive and current: fix the gaps found at `v2.12.0`, give every exported name a doc comment and every feature a description, start a `CHANGELOG.md`, and hold the line with a docs step in every brief rather than a new check

## Context and Problem Statement

### The mandate

On 2026-11-14 the owner set a mandate for every related repository,
relayed by the coordinator:

> "Docs across all related repos should be accurate, comprehensive and
> kept up to date. When they fall behind they are hard to fix"

The owner then ruled on what "comprehensive" requires, on a changelog, on
enforcement and on how documentation fixes order against a release (the
four rulings are under Considered Options, with the options as offered).
This record applies them to this repository; each other repository takes
its own record.

### What the survey found

A read-only survey of this repository at `f093da6` (branch `main`, the tree
clean, the last tag `v2.12.0`, with two docs-only commits after it) ran on
2026-11-14, by a mainstream-tier agent at medium effort. It reports ten
statements that are wrong, three things missing and three stale:

* **Wrong.** `docs/architecture.md:19` says "Thirty-one packages"; the
  survey counts 37 public package directories. Seven rows of the package
  diagram in `docs/architecture.md` (lines 31 to 70) omit an import the
  code has. Two rows name paths that do not exist: `CLAUDE.md` in
  `AGENTS.md:4`, and the planned `stream/glamourmd`.
* **Missing.** `lineedit/testdata/xtermparity` is named in no living
  document. Many exported names appear in no document and no package
  comment, by word search: `termkey` 155 of 163, `termcap` 143 of 204,
  `termsvc` 43 of 98, `command` 41 of 151, `stream` 27 of 83, and smaller
  counts in fifteen other packages.
* **Stale.** There is no `CHANGELOG.md`. `AGENTS.md:147-149` says "Release
  notes list every deprecation and every removal" without naming where
  release notes live. `docs/reports/0014-REPORT-api-assessment-and-integration-research.md`
  has no front matter `date:`, where the other three reports have one.

**The survey's limits.** It is evidence of where to look, not a verified
list. Its name coverage is by word presence in the source and the
documents, not by type resolution, and it is a different measure from the
owner's standard: it did not measure exported names that lack a doc
comment. It read `README.md`, `AGENTS.md`, `docs/README.md` and
`docs/architecture.md` in full, but scanned the ten guides (4,757 lines)
and the 46 package comments by script only, and read none of the records'
bodies, the agent rule files or the framework examples. It did not run
markdownlint or `TestDefaultsInventory`.

### What this record checked (read-only, at `f093da6`)

* **The count and the imports.** `go list ./...` with `GOWORK=off` lists 47
  packages: 37 public and 10 under an `internal` directory, two of those
  (`internal/conformance` and `tuitest/internal/clash`) holding tests only.
  "Thirty-one packages" is wrong; "ten internal ones" is right. Comparing
  each diagram row with the package's non-test imports confirms the seven
  rows exactly, and finds no other row that differs: `launch` lacks
  `command/commandtea`; `inline/inlinetest` lacks `internal/sanitize`;
  `palette` lacks `msgcat`; `workspace` lacks `command/commandtea` and
  `msgcat`; `command` lacks `msgcat`; `termcap` lacks `msgcat`;
  `internal/lineinput` lacks `internal/sanitize`. 7 of the diagram's 45
  rows have drifted since they were written.
* **Two "wrong" rows do not hold.** `AGENTS.md:4` says a repository-local
  `CLAUDE.md` "wins only where it is more specific", a conditional that
  claims no such file exists. `docs/guides/releasing.md:6` calls
  `stream/glamourmd` "the planned `stream/glamourmd`", and
  `docs/glossary.md:194-196` says it is added "when the adapter is built".
* **Doc comments, approximately.** A line-based count over the non-test Go
  files (not type-resolved; the PLAN's Phase D measures with `go/ast`) finds,
  in the public packages: every exported function (451) and type (369) has
  its own doc comment; 589 of 592 methods do (`command`'s `ArgError.Error`,
  `PanicError.Error` and `Scope.String` do not); 333 of 664 constants and 19
  of 23 variables have their own comment, the rest sharing their group's
  block comment (153 of them `termkey`'s key codes); 134 of 585 exported
  struct fields and 7 of 49 interface methods have one. Of the fields,
  interface methods and methods without their own comment, 230 carry a
  trailing line comment and 266 carry none. The top-level completeness is
  explained by the linter already in place: `.golangci.yml` enables
  `revive`'s `exported` rule with no exclusion preset, and that rule checks
  top-level declarations, not fields, interface methods or each member of
  a commented group.
* **Release notes.** `docs/guides/migrating.md` has a section for 21 of the
  39 tags (`v0.9.0` to `v2.12.0`), under the headings Added, Changed,
  Deprecated, Removed, Fixed and Documented; the earlier tags and the
  retired adapters' four tags have none. The tag annotations carry only the
  tag's name. Whether the hosting service holds release notes was not
  checked.
* **Feature leads.** The `Makefile` targets `test-sum`, `fmt`, `tidy` and
  `help` are named only in the target list at `docs/architecture.md:376-379`,
  undescribed, and the override variables `GOLANGCI_LINT`, `GOVULNCHECK`
  and `GOTESTSUM` are named in no document. About 34 upper-case string
  literals in `termcap`, `termsvc`, `launch`, `theme` and `tuitest` that
  look like environment variable names (`WT_SESSION`, `TMUX`, `ZELLIJ`,
  `COLORFGBG`, `CLICOLOR_FORCE` among them) are named in no document; not
  every literal is a variable the library reads (`AAAA` is not).
* **One new lead.** `docs/architecture.md:560-563`, "What is not here",
  lists "standard panes (… scrolling text and Markdown)", while
  `stream/streamtea.DocPane` is a scrolling Markdown pane. Whether the
  sentence means a different pane is for Phase A of the PLAN to settle.

### The question

How does this repository's documentation become accurate and comprehensive
by the owner's standard, and stay so, with no new check, without holding
a release, and in an order that does not collide with
[0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md), whose first
release, `v2.13.0`, is the next tag?

## Decision Drivers

* **The mandate:** accurate, comprehensive and kept up to date, because
  documentation that falls behind is hard to fix.
* **The owner's four rulings**, which this record applies and does not
  reopen.
* **A wrong statement costs more than a missing one:** an agent or a
  consumer acts on it. `docs/architecture.md` is the first page the index
  sends a reader to ("see what is in this repository today").
* **No new check,** so the docs must be fixed by people and agents, and
  kept fixed by the work that changes the code: a fact written in a form
  that drifts silently should be avoided where the owner agrees.
* **Releases are not held** for gaps a release did not cause.
* **Records are history.** A record's body is not rewritten; an
  unexecuted record's stale fact is the subject of its own amendment
  ([0011-MADR-docs-accuracy-after-v0-5-0.md](0011-MADR-docs-accuracy-after-v0-5-0.md)).
* **The evidence is partly unverified,** so the first phase verifies it and
  reads in full what the survey only scanned, before any fix.

## Considered Options

The options are the ones offered to the owner on 2026-11-14, in the words
offered, grouped by ruling.

**R1. What "comprehensive" requires.**

* "Doc comment plus feature docs"
* "Every name in a reference page"
* "Feature docs only"

**R2. A changelog.**

* "CHANGELOG.md in every repo"
* "Keep each repo's current practice"
* "Release-notes reports everywhere"

**R3. Enforcement.**

* "Fix the docs, no new checks"
* "Add checks to each gate"
* "Checks in the three release repos only"

**R4. Release ordering.**

* "Release-caused fixes before each tag"
* "All fixes before each tag"
* "Tags first, docs after"

## Decision Outcome

Chosen options, each the owner's ruling of 2026-11-14, relayed by the
coordinator: R1 "Doc comment plus feature docs", R2 "CHANGELOG.md in every
repo", R3 "Fix the docs, no new checks" and R4 "Release-caused fixes
before each tag", because together they make the documentation complete
at the level a Go library's reader uses (`go doc` for a name, a guide or
the architecture note for a feature), give each release one place that
says what changed, and keep the cost of currency inside the work that
causes it, without a gate to maintain and without holding a tag.

This record stays `proposed` until the owner answers the open questions
below; each recommendation is written into D1 to D6, and an answer other
than a recommendation amends the decision it names before the PLAN runs.
*(2026-11-14: the owner ruled on Q1, Q3 and Q7, relayed by the
coordinator, and the coordinator applied the owner's ruling on the global
rules file to Q6; each is annotated under its question, and D1, D2, D4 and
D5 follow them. Q2, Q4 and Q5 are open.)*
*(2026-11-14, later: the owner answered Q2, Q4 and Q5, relayed by the
coordinator, each the recommendation; each is annotated under its
question, and D1, D2 and D6 follow them. No question is open. This record
stays `proposed` until the owner approves the PLAN and says when it
starts.)*

### D1. The standard

In the owner's words: every exported name has a doc comment, and every
package, command, flag, environment variable, settings key and make
target is described in a README, guide or reference. Names are not all
listed in prose.

* **A doc comment on every exported name,** in non-test files outside
  `testdata`, as the owner ruled on Q3 (2026-11-14, "Fields of public and
  contract types"): "Top-level names, plus exported fields and interface
  methods of public packages and of the types users configure or receive
  (wire and config structs). A field counts as documented when its type's
  comment names it." Applied here:
  * every exported top-level declaration and method has its own doc
    comment, in every package, `internal/` included;
  * every exported struct field and interface method of a type in a public
    package, and of a type users configure or receive wherever it is
    declared, has a doc comment or is named in its type's doc comment;
  * a field or interface method of a type under `internal/` that users
    neither configure nor receive needs none;
  * **not ruled, this record's proposal:** a member of a grouped `const`
    or `var` block is covered by the block's comment when that comment
    describes the members as a set, and otherwise needs its own; and a
    trailing line comment on a field or interface method serves as its
    comment. The owner's word on either governs.
* **A description of every feature,** in a README, a guide or a reference
  section (`docs/architecture.md`'s package table and Tooling are
  reference sections). What counts as a description, as the owner ruled
  on Q4 (2026-11-14, "Row plus a sentence each"): a package by its row in
  `docs/architecture.md` and its package comment; each command, flag,
  environment variable, settings key and make target by at least a
  sentence saying what it does and its default, in the guide or reference
  section a reader of that feature reads. That adds a reference list of
  the environment variables the library reads, and descriptions of the
  undescribed make targets and variables.
* **The survey's word-search coverage is not the standard.** A name
  missing from the prose is not a defect by itself ("Names are not all
  listed in prose"); its doc comment and its feature's description are
  what is checked.

### D2. The changelog

* **`CHANGELOG.md` at the repository root,** one file, updated in the same
  phase as the change it records: an `Unreleased` section that each
  change adds to, renamed to the version in the release step before the
  tag.
* **Its sections** are the ones `docs/guides/migrating.md` already uses:
  Added, Changed, Deprecated, Removed, Fixed and Documented, and Security
  when there is one; a release heading is the version and its tag's date,
  with no link that names the hosting account. Every deprecation and every
  removal is listed, as API convention 1 requires of release notes.
* **How far back it starts,** as the owner ruled on Q1 (2026-11-14,
  "Latest tag forward"): an `Unreleased` section, one section for the
  latest tag, `v2.12.0`, then a single pointer to where older notes live,
  `docs/guides/migrating.md`. Nothing older is copied.
* **How it divides the work with `docs/guides/migrating.md`,** as the
  owner ruled on Q5 (2026-11-14, "Changelog lists, guide migrates"): from
  the next release on, `CHANGELOG.md` says what each release changed and
  the migration guide keeps only what a program must change (renames,
  deprecations, recipes), each linking the other. The guide's existing
  sections stay as they are.
* **`AGENTS.md`'s release-notes sentence** (API convention 1) names
  `CHANGELOG.md`.

### D3. No new check

* **The PLAN adds no gate, no CI job and no check script,** and changes no
  existing one. The checks already in place keep running as they are:
  `make check-records` (names, index and links), `make docs-check` (the
  guides' excerpts and `TestDefaultsInventory`), `revive`'s `exported`
  rule, and `internal/conformance`'s stability lines.
* **A one-off audit is verification, not a check.** A phase may run a
  command or a scratch program that proves its own work (the doc-comment
  measurement, the import comparison, the feature inventory). It lives
  outside the repository, is never committed, and its output is quoted in
  the PLAN's execution record. Before its result is trusted, it is seen
  failing on a deliberately broken copy, never on the working tree.

### D4. Release ordering, against `0026` and the next tag

* **Before a tag,** the release's own PLAN fixes every statement the
  release makes false and documents what it ships, `CHANGELOG.md` entry
  included. For the next two tags that is
  [0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md)'s Step 9
  (`v2.13.0`: `diffview`, `agentui`, `agentui/agentuitea`'s `FeedPane`)
  and Step 14 (`v2.14.0`), per its MADR's D8.
* **Older gaps,** everything this record lists, all present at `v2.12.0`,
  are this record's PLAN's, and never hold a tag. A phase of this PLAN
  does not run while a release step is in progress; it resumes after the
  tag, re-checking its list against the new tree first.
* **Now,** no release is in progress: 0026-PLAN is `proposed`, and waits
  for go-llmprovider-sdk `v1.31.0`, gobble-cli `v0.7.0` and the owner's
  approval. As the owner ruled on Q7 (2026-11-14, "Yes, docs-only phases
  now"): "Phases that change only Markdown run now, one at a time, and
  pause at a phase boundary if a release step starts. Doc-comment phases
  that edit Go files wait where code is about to be rewritten."
* **The Go file 0026 rewrites.** Of the code files that exist at
  `f093da6`, 0026-PLAN's build edits `glyph/glyph.go` (Step 2, `glyph.Set`'s
  new fields), tests under `internal/conformance/` and `.golangci.yml`; its
  other code files are new, and its document edits are its own docs
  steps. Only `glyph/glyph.go` holds exported names
  the doc-comment phase would edit (`glyph.Border`'s fields, by the
  approximate count). That phase therefore leaves `glyph/glyph.go` to
  0026's Step 2, whose docs step comments the file's exported names as it
  rewrites it, and runs now for every other file.
* **`CHANGELOG.md` before Step 9.** This PLAN's changelog phase is meant to
  land before 0026's Step 9, so that Step 9 adds an entry to a file that
  exists. If Step 9 comes first, it creates `CHANGELOG.md` with its own
  entry, in the shape D2 sets, as part of documenting what `v2.13.0`
  ships; the changelog phase then adds the rest.

### D5. A docs step in every later brief

Every later brief for code work in this repository carries a docs step,
by the mandate, executed in the same phase as the change:

1. a doc comment on each exported name the change adds or changes (D1);
2. a description of each package, command, flag, environment variable,
   settings key or make target it adds or changes, in the README, guide or
   reference section its reader reads (D1);
3. every statement the change makes false corrected, in the living
   documents and the package comments;
4. its line under `CHANGELOG.md`'s `Unreleased` (D2);
5. `docs/README.md`'s rows for a new guide or section.

A phase with nothing to document says so, with the reason, in its
execution record. `0026`'s build documents `diffview`, `agentui` and
`agentui/agentuitea` as each lands, in the step that adds it, not only at
Steps 9 and 14; the PLAN adds a dated note to
[0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md) saying so.
`AGENTS.md` does not state this step as a rule (Q6, as the coordinator
applied the owner's ruling; the annotation under Q6): the step lives in
the coordinator's skill and in each brief. Factual corrections to
`AGENTS.md` stay in scope, the release-notes sentence naming
`CHANGELOG.md` among them.

### D6. `docs/architecture.md`'s count and import lists

As the owner ruled on Q2 (2026-11-14, "Restate without counts"), the
section is restated so that it cannot go stale: no count, and the diagram
keeps each package's purpose and the external modules it imports, while
its layering inside the repository is stated as the rules that hold it
(imports point downward; the Charm-free cores; ultraviolet's two
importers), which `depguard` and `internal/conformance` already enforce.
The wrong count and the seven wrong import rows go with the restatement.

### Consequences

* Good, because the first page a reader is sent to becomes true of the
  tree, and the rest of the living documents are read in full against the
  code for the first time since
  [0011-MADR-docs-accuracy-after-v0-5-0.md](0011-MADR-docs-accuracy-after-v0-5-0.md).
* Good, because `go doc` answers for every exported name, and each
  feature a program configures has one place that describes it.
* Good, because a release has one file that says what it changed.
* Good, because no tag waits for this work, and `0026`'s releases document
  only what they ship.
* Neutral, because no Go behaviour, API or release changes; Phase D adds
  comments to Go files, so those commits run the full CI rather than
  Docs.
* Bad, because nothing automated will report the next gap: currency
  depends on each brief carrying its docs step and each executor doing
  it. The owner chose this over a check.
* Bad, because the doc-comment phase is large: by the approximate count,
  under the owner's Q3 ruling, three methods and up to about 266 fields and
  interface methods in the public packages (fewer where a type's comment
  already names them, which was not measured), and a judgment on about
  331 grouped constants and 4 variables; each comment must be written from
  the code, not guessed.
* Bad, because `docs/architecture.md`, `README.md`, `docs/README.md` and
  `CHANGELOG.md` are edited by this PLAN and by 0026's release steps, so
  the two must not run on those files at once.
* Bad, because the survey's leads are partly unverified until Phase A ends,
  so the size of Phases B and C is not known in advance.

### Confirmation

There is no new check (R3), so confirmation is by reading, at points the
work already passes through:

1. **This PLAN's close-out** re-reads each finding of its Phase A list at
   the closing commit and quotes the evidence; re-runs the doc-comment
   measurement (0 names missing under Q3's definition) and the feature
   inventory (every item with its place); and quotes the existing checks'
   output (`make check-records`, `make docs-check`, `make
   disclosure-check`, markdownlint), each exit status recorded.
2. **At every release step,** `CHANGELOG.md`'s newest version heading is
   the tag about to be made, and its entry lists every deprecation and
   removal the release's PLAN made; the release PLAN's execution record
   quotes it.
3. **In every later PLAN,** the execution record shows its docs step: the
   files and doc comments touched, or "nothing to document" and why.
4. **A reader** can test any one claim: `go doc` on a name shows a
   comment; `git describe --tags --abbrev=0` names `CHANGELOG.md`'s newest
   released version.

## Pros and Cons of the Options

These are this record's reading of the options as offered; the owner's
reasons were not relayed beyond the choices.

### R1: "Doc comment plus feature docs" (chosen)

* Good, because it matches how a Go library is read: `go doc` for a name,
  prose for a feature.
* Good, because the top-level half is already nearly met (the `revive`
  rule), so the cost is concentrated in fields, interface methods and
  grouped constants.
* Bad, because "every exported name" needs a definition (Q3) before it can
  be measured.

### R1: "Every name in a reference page"

* Good, because a reader of the documents alone would find every name.
* Bad, because about 2,100 exported top-level names and methods, by the
  approximate count, would be listed in prose beside `go doc`, which is
  the kind of list that falls behind.

### R1: "Feature docs only"

* Good, because it is the smallest.
* Bad, because a field or constant without a comment stays unexplained
  where a programmer looks first.

### R2: "CHANGELOG.md in every repo" (chosen)

* Good, because a consumer finds what a release changed in the file every
  Go repository is expected to have, and `AGENTS.md`'s "release notes"
  gets a place.
* Bad, because it overlaps the migration guide's per-release sections
  until Q5 divides them.

### R2: "Keep each repo's current practice"

* Good, because nothing changes here: the migration guide already serves.
* Bad, because `AGENTS.md`'s "release notes" stays without a place, and
  practice differs across the repositories the mandate covers.

### R2: "Release-notes reports everywhere"

* Good, because a report per release fits the records' numbering.
* Bad, because a reader would look for a release's changes among
  decision records, and every release would take a record number.

### R3: "Fix the docs, no new checks" (chosen)

* Good, because no gate is added to maintain, and no release waits on a
  docs linter's false positive.
* Bad, because a new gap is found only by a docs step or a reader.

### R3: "Add checks to each gate"

* Good, because drift would fail a gate the day it happens.
* Bad, because each check (doc comments on fields, feature inventories,
  counts) is code to maintain in every repository.

### R3: "Checks in the three release repos only"

* Good, because the checks would sit where releases happen.
* Neutral, because whether this repository is one of the three was not
  stated with the option.
* Bad, because the other repositories would rely on discipline alone, and
  the rules would differ by repository.

### R4: "Release-caused fixes before each tag" (chosen)

* Good, because a tag never makes the docs newly wrong, and never waits
  for an old gap.
* Bad, because old gaps live on after a tag until this PLAN lands them.

### R4: "All fixes before each tag"

* Good, because every tag would ship with the docs current.
* Bad, because the next tag, `v2.13.0`, would wait for this whole PLAN.

### R4: "Tags first, docs after"

* Good, because releases are never slowed.
* Bad, because each tag would ship docs it had made wrong.

## Owner questions

Each has options, the recommended first, with its cost. Q1, Q3 and Q7 are
ruled and Q6 applied (2026-11-14), each annotated below its options, which
stand as written; Q2, Q4 and Q5 were answered later the same day, and
none is open.

**Q1. How far back does `CHANGELOG.md` start? (D2)**

* **(a) From the last tag forward:** an `Unreleased` section and a
  `v2.12.0` entry, then a pointer to `docs/guides/migrating.md` for
  `v0.9.0` to `v2.11.0` and to the records for the tags before. Cost:
  a reader of `CHANGELOG.md` follows a link for older releases.
* (b) Backfill from the migration guide: one entry for each of its 21
  sections, `v0.9.0` to `v2.12.0`, the older tags pointed at the records.
  Cost: about twenty entries written from existing text, which duplicate
  the guide unless Q5 (a) also moves its Added lists.
* (c) Backfill every tag, 39 of them including `v0.1.0` to `v0.8.0` and the
  retired adapters', from the records and the history. Cost: the largest,
  for releases no consumer pins.

*Ruling, 2026-11-14, the owner's, relayed by the coordinator: "Latest tag
forward": "An Unreleased section and one section for the latest tag, then
a single pointer to where older notes live (tui's migration guide, acp's
release-notes reports, magic's tags and records). Least work, nothing
duplicated; older releases are one link away." Rejected: "Plus one line
per older tag"; "Full backfill". **Decision: (a)**, with the single
pointer the ruling names, to `docs/guides/migrating.md` (D2).*

**Q2. Does `docs/architecture.md`'s package section stay hand-written? (D6)**

* **(a) Restate it so it cannot go stale silently:** no count; the
  diagram keeps each package's purpose and the external modules it
  imports, and its in-repository layering is stated as the rules
  `depguard` and `internal/conformance` hold. Cost: a reader asking what
  `palette` imports inside the repository runs `go list` instead of
  reading the diagram.
* (b) Keep the count and every import, corrected, and re-compare with
  `go list` in each release's docs step. Cost: the same drift returns
  whenever a step forgets; 7 of 45 rows drifted since they were written.
* (c) Generate the diagram from `go list` with a committed script, re-run
  at each release. Cost: a new script to maintain, against the spirit of
  R3, and the diagram still drifts between runs.

*Ruling, 2026-11-14, the owner's, relayed by the coordinator: "Restate
without counts": "Keep each package's purpose and the external modules it
uses. State the in-repository layering as the rules the linter and
conformance tests already enforce, instead of listing every import.
Nothing left to drift; a reader wanting one package's internal imports
runs go list." Rejected: "Keep, corrected, re-compared"; "Generate from go
list". **Decision: (a)** (D6).*

**Q3. Which exported names need a doc comment, and what serves as one? (D1)**

* **(a) Every exported top-level declaration and method its own comment;
  a grouped constant or variable covered by its block's comment where that
  comment describes the members as a set, else its own; every exported
  struct field and interface method a doc comment or a trailing line
  comment.** Cost, approximately: three methods, the fields and interface
  methods with no comment at all (about 266 in the public packages and 11
  under `internal/`), and a judgment on each commented block (about 331
  constants and 4 variables).
* (b) Every exported identifier its own doc comment, each constant, field
  and interface method included. Cost: about 830 comments, among them 153
  for `termkey`'s key codes, much of it repeating the name.
* (c) Top-level declarations and methods only, as `revive`'s `exported`
  rule checks. Cost: three comments, and fields and interface methods
  stay undocumented.

*Ruling, 2026-11-14, the owner's, relayed by the coordinator: "Fields of
public and contract types": "Top-level names, plus exported fields and
interface methods of public packages and of the types users configure or
receive (wire and config structs). A field counts as documented when its
type's comment names it." Rejected: "Top-level names only"; "Every
exported identifier". **Decision:** the owner's words, which differ from
(a): a type's comment naming a field documents it; fields and interface
methods under `internal/` need comments only on types users configure or
receive. Grouped constants and trailing line comments were not ruled; (a)'s
treatment of them stays this record's proposal (D1).*

**Q4. What counts as a feature's description? (D1)**

* **(a) A package by its `docs/architecture.md` row and its package
  comment; each command, flag, environment variable, settings key and
  make target by at least a sentence, with its default, in the guide or
  reference section its reader reads.** Cost: a reference list of the
  environment variables `termcap` and `launch` consult, and descriptions
  for the undescribed make targets and variables.
* (b) As (a), and a guide section for every public package. Cost: new
  guide text for the packages no guide covers today, `glyph`, `safetext`,
  `msgcat`, `termkey` and `tuitest` among them.
* (c) The package comment and `make help` count as the reference. Cost:
  a reader of the README and guides never meets the environment variables
  or the make variables.

*Ruling, 2026-11-14, the owner's, relayed by the coordinator: "Row plus a
sentence each": "A package is described by its architecture row and its
package comment. Every other feature gets at least one sentence, with its
default, in the guide or reference section its reader would look in. Adds
a reference list of the environment variables the library reads and
descriptions for the undescribed make targets and variables." Rejected:
"Plus a guide per package"; "Package comment and make help". **Decision:
(a)** (D1).*

**Q5. How do `CHANGELOG.md` and `docs/guides/migrating.md` divide the work? (D2)**

* **(a) `CHANGELOG.md` says what each release changed; from the next
  release, the migration guide keeps only what a program must change
  (renames, deprecations, recipes), each linking the other.** Cost: the
  guide's existing sections stay as they are, so the split begins at the
  next release.
* (b) Both keep full lists. Cost: two lists of every change, kept in step
  by hand, the drift the mandate warns of.
* (c) `CHANGELOG.md` is a one-line index of releases, each linking the
  guide's section. Cost: the changelog says little, and the release
  notes still live in a guide.

*Ruling, 2026-11-14, the owner's, relayed by the coordinator: "Changelog
lists, guide migrates": "From the next release, the changelog says what
each release changed and the migration guide keeps only what a program
must change (renames, deprecations, recipes). Each links the other. The
guide's existing sections stay as they are." Rejected: "Both keep full
lists"; "Changelog as an index". **Decision: (a)** (D2).*

**Q6. Does `AGENTS.md` state the docs step as a rule? (D5)**

* **(a) Yes: a short "Documentation" section stating the standard (D1),
  the docs step (D5), `CHANGELOG.md` in the same phase (D2) and the release
  ordering (D4); text an agent reads, not a check.** Cost: one more
  section in a long file.
* (b) No: only the release-notes sentence names `CHANGELOG.md`; the rest
  lives in this record and in each brief. Cost: an agent not briefed by
  the coordinator never sees the step.
* (c) Yes, and restated in the per-agent pointer files. Cost: three
  copies, against those files' own rule not to restate `AGENTS.md`.

*2026-11-14: not put to the owner. On the same question for the global
rules file the owner ruled "Skill only": the docs rule lives in the
coordinator skill and in briefs, not in a rules file. The coordinator
applied that ruling to this repository's files and told the owner so; the
owner has not ruled on per-repository files specifically, and a contrary
word from the owner governs. **Applied: (b)** (D5): no docs-step section
in `AGENTS.md`; factual corrections stay in scope, the release-notes
sentence naming `CHANGELOG.md` among them.*

**Q7. Does this PLAN run now, or after `v2.13.0`? (D4)**

* **(a) Now, while no release is in progress, pausing at a phase boundary
  for any release step that starts.** Cost: if 0026 starts mid-way, a
  phase waits, and its list is re-checked after the tag.
* (b) After `v2.13.0` is tagged, reading the ruling's "land after the tag"
  literally. Cost: 0026 waits on two releases elsewhere, so the known
  wrong statements stay for an unknown time, and 0026's Step 9 edits a
  wrong `docs/architecture.md`.
* (c) Phases 0, A and E now (records, the defect list, the changelog);
  the rest after `v2.13.0`. Cost: two passes, and the wrong statements
  stay until the second.

*Ruling, 2026-11-14, the owner's, relayed by the coordinator: "Yes,
docs-only phases now": "Phases that change only Markdown run now, one at a
time, and pause at a phase boundary if a release step starts. Doc-comment
phases that edit Go files wait where code is about to be rewritten."
Rejected: "As ruled: after the tag"; "Defect list and changelog now, rest
after". **Decision: (a)** (D4); the doc-comment phase leaves
`glyph/glyph.go`, which 0026's Step 2 rewrites, to that step.*

## More Information

### What is not verified

* The survey's guides (4,757 lines in ten files), `docs/glossary.md` beyond
  the lines it sampled, and the 46 package comments were scanned by script,
  not read; this record read none of them in full either. Phase A reads
  them.
* The doc-comment counts are a line-based approximation, not type-resolved,
  and a struct field declared with several names counts once.
* The environment-variable list is a literal scan, not a list of what the
  library reads.
* Whether the hosting service holds release notes for any tag.
* The bodies of the unexecuted records (0009-PLAN Step 7, 0022-PLAN Phase
  4, 0026) were not audited for stale facts.
* markdownlint and `TestDefaultsInventory` were not run by the survey; this
  record ran markdownlint on its own two files only.

### Not in scope

* Any new gate, CI job or check script (R3).
* Code behaviour, the exported API, any release or tag, and `git push`.
* The bodies of records; an unexecuted record's stale fact is listed for
  the owner and amended in its own record.
* `stream/glamourmd` and every other planned package.
* The other repositories the mandate covers; each takes its own record.

### Related records

* [0011-MADR-docs-accuracy-after-v0-5-0.md](0011-MADR-docs-accuracy-after-v0-5-0.md):
  the last documentation pass, and its rule that records are corrected by
  amendment and front matter only.
* [0022-MADR-align-docs-ci-and-disclosure-gates.md](0022-MADR-align-docs-ci-and-disclosure-gates.md):
  the documents' checks this record leaves as they are.
* [0024-MADR-opt-in-and-opt-out-coverage.md](0024-MADR-opt-in-and-opt-out-coverage.md):
  `docs/guides/defaults.md` and `TestDefaultsInventory`.
* [0026-MADR-agent-widgets.md](0026-MADR-agent-widgets.md) and
  [0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md): the next
  releases, and the packages they document as they land.
* The plan: [0027-PLAN-keep-docs-accurate-and-current.md](0027-PLAN-keep-docs-accurate-and-current.md).
