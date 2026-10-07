---
status: accepted
date: 2026-10-07
decision-makers: owner
consulted: a read-only audit of every Markdown document in the repository on 2026-10-07, in four parts (the living top-level documents, the four guides, the unexecuted records 0007 to 0009, and every record's metadata and cross-references), each finding checked against the code, `go doc`, `go list`, the tags and the records
informed: pi-go
---
# Bring the living documents and the unexecuted records up to date after `v0.5.0`, correcting records only by amendment and front matter

## Context and Problem Statement

[0006-PLAN-command-registry.md](0006-PLAN-command-registry.md) closed on
2026-10-07 with four releases: `v0.4.0`, `command/cobracmd/v0.1.0`,
`v0.5.0` and `command/kongcmd/v0.1.0`. Its amendments A1 to A12 changed
many facts that other documents state. The owner asked for every
document to be assessed and made accurate and current.

A read-only audit read all 33 tracked Markdown files. It checked their
claims against `go doc`, `go list`, the code, the `go.mod` files, the
tags, the Makefile, CI and the records. It also compiled every Go example
in the guides in a scratch module, and resolved 339 relative links and
238 citations of another record's section, amendment, deviation, step or
phase. The links and citations all resolve; the index's statuses match
every record's; every guide example compiles.

What the audit found falls into four kinds:

1. **Living documents that are wrong or stale.**
   * `README.md` names `v0.4.0` as the current release, dates three
     `v0.2.0` features to `v0.1.0`, and does not say the two adapters are
     released.
   * `AGENTS.md` lets only `internal/cells` import ultraviolet, but
     `internal/termevent` does too, and depguard allows it.
   * `docs/architecture.md` gives `workspace` an import of lipgloss it
     does not have and omits `colorprofile` and `x/ansi`; counts four
     internal packages where there are five; has no table of the two
     adapters' APIs; lists "a help footer" as not built; and cites
     `0010-REPORT §2` by bare number.
   * The commands guide's example takes the slash name `resize`, which
     `workspace.resize` holds, so `workspace.Commands` is refused when the
     guide registers both; it passes a split name where
     `Workspace.Resize` takes a separator ID; it cites amendments "A1 to
     A9"; and it does not mention the Cobra and Kong front ends.
   * The terminal capabilities guide says `CapsMsg` never comes before
     the colour profile, but the deadline delivers it regardless; and
     cites "A1 to A4" where there are five.
   * The releasing guide's `git tag -a` has no `-m`, so it opens an
     editor; its smoke test stops at `go build`; and it does not mention
     the disclosure guard AGENTS.md requires before a push.
2. **Unexecuted records whose facts went stale.** 0007's default
   bindings name commands that do not exist (`workspace.focus.pane`,
   `workspace.resize.left` and its siblings) and a `glyph.Table` that does
   not exist. 0008 and 0009 attribute the width method to a PLAN that
   puts it out of scope. 0009 cites moved line numbers and still waits on
   0010, which is complete.
3. **Record metadata.** Front matter `date` is "when the decision was
   last updated" in MADR 4 and in the `madr-and-plan-writing` skill.
   0001-MADR and 0002-MADR follow that, but five MADRs and two PLANs
   kept their first date after later amendments. 0001-MADR's index row
   does not name its amendments.
4. **A rule that the records do not follow.** AGENTS.md says a MADR and
   its PLAN share one slug and a number is never reused, but 0002 has a
   second PLAN with its own slug, and the 0001 and 0010 REPORTs share
   their MADR's number. The `madr-and-plan-writing` skill allows both.

## Decision Drivers

* A reader acts on the living documents: the README, AGENTS.md, the
  index, the architecture note and the guides must be true of the tree
  as it is.
* A record is history: its body is not rewritten (AGENTS.md, "do not
  silently rewrite historical rationale"). A correction is an amendment,
  or a front-matter field the record's own convention says to keep
  current.
* An unexecuted record will be executed: a stale fact in it costs the
  most when its PLAN runs.
* A design choice found during an audit belongs to its own record's
  owner question, not to a documentation pass.

## Considered Options

* Correct the living documents in place, the records by amendment and
  front matter, and the rule to match practice, under a new pair.
* Reopen [0006-PLAN-command-registry.md](0006-PLAN-command-registry.md)
  with a Step 13.
* Correct only the living documents, and leave the records.

## Decision Outcome

Chosen option: "Correct the living documents in place, the records by
amendment and front matter, and the rule to match practice, under a new
pair", because the pass spans records from 0001 to 0010 and AGENTS.md,
and 0006 is complete.

1. **Living documents are corrected in place,** each finding against the
   evidence the audit recorded, and checked again after the edit.
2. **The commands guide's example becomes an action the workspace does
   not have,** with its own slash name, a positional and a flag, compiled
   and registered beside `workspace.Commands` in a scratch module before
   it is written. The guide gains a section on the Cobra and Kong front
   ends.
3. **`date` is when a record was last updated,** for every MADR and
   PLAN, as MADR 4 and the skill define it: the seven stale dates move
   to their last amendment's or revision's date. Only front matter
   changes.
4. **Unexecuted records are corrected by amendment:** 0007 A2, 0008 A1,
   0009 A3, each stating the stale facts and their corrections. 0007's
   default bindings are recorded as an open owner question, to answer
   when 0007 is executed: add commands for focus by index and directional
   resize, or bind the keys to the existing IDs.
5. **AGENTS.md's Records section matches the skill:** a MADR may carry
   several PLANs, each with its own slug, a lone PLAN copying the
   MADR's; a REPORT about a record takes that record's number. The
   records stay as they are.
6. **README.md's history is made exact:** `v0.5.0` and A12's change, the
   adapters released at `v0.1.0`, and the `v0.2.0` features dated
   `v0.2.0`.
7. **Completed records' bodies are not touched.** Where a completed
   record's text is stale, as 0010-MADR A2's reference to an
   execution-record phase is, the fact stays as written.

### Consequences

* Good, because every living document is true of the tree at `v0.5.0`,
  and the three unexecuted records can be executed without first
  discovering their stale facts.
* Good, because the one rule for `date` makes a record's front matter
  say when it last changed.
* Bad, because seven records' dates move, which a reader comparing with
  git history may notice; the amendments and revisions they follow give
  the reason.
* Bad, because 0007's default bindings stay undecided until 0007 is
  executed.
* Neutral, because no Go code, no release and no API changes.

### Confirmation

[0011-PLAN-docs-accuracy-after-v0-5-0.md](0011-PLAN-docs-accuracy-after-v0-5-0.md)
lists every finding and the step that resolves it. Its close-out checks
each one again, re-runs the link and citation checkers and the example
compilation, and runs markdownlint and the identifier scan.

## Pros and Cons of the Options

### A new pair

* Good, because the pass is its own unit of work, spanning many records.
* Good, because 0006 stays closed as complete.
* Neutral, because it adds a record pair for documentation work.

### Reopen 0006 with a Step 13

* Good, because most findings trace to 0006's releases.
* Bad, because 0001, 0005, 0007 to 0010 and AGENTS.md are not 0006's,
  and a complete PLAN would go back to in-progress.

### Only the living documents

* Good, because it is smaller.
* Bad, because 0007 would be executed against command IDs that do not
  exist, and the records' metadata would stay inconsistent.

## More Information

* The findings, with the evidence for each, are tabled in
  [0011-PLAN-docs-accuracy-after-v0-5-0.md](0011-PLAN-docs-accuracy-after-v0-5-0.md).
* **Owner answers, 2026-10-07** (picked from options): a new pair; a
  different action for the example; `date` as the date last updated;
  0007's bindings recorded as an open question; AGENTS.md matched to the
  skill; README's history made exact. Each was the recommendation. The
  alternatives were: reopening 0006; keeping `layout.resize` without its
  slash name; `date` as the date first decided; deciding 0007's bindings
  now, by adding commands or by binding the existing IDs; leaving
  AGENTS.md and noting the exceptions; changing only README's release
  line.
