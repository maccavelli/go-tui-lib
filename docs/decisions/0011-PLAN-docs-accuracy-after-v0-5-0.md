---
status: proposed
date: 2026-10-07
associated-madr: "0011-MADR-docs-accuracy-after-v0-5-0.md"
---
# Implement the documentation pass after `v0.5.0`

Associated MADR: [0011-MADR-docs-accuracy-after-v0-5-0.md](0011-MADR-docs-accuracy-after-v0-5-0.md)

## Revisions

* **2026-10-07, paused.** Before this PLAN was approved, the owner
  retired the library's CLI front ends:
  [0012-MADR-bring-your-own-cli.md](0012-MADR-bring-your-own-cli.md).
  This PLAN waits until
  [0012-PLAN-bring-your-own-cli.md](0012-PLAN-bring-your-own-cli.md) is
  complete, then resumes, re-scoped:
  * **Dropped:** F20's section on the Cobra and Kong front ends, and the
    mention of A12 in F1 and F20. 0012 removes what they describe.
  * **Done by 0012 Step 5:** F4 (the adapters), F9's adapters table and
    F15 (the Cobra or Kong rows).
  * **Kept:** the other findings. They are re-checked against the tree
    0012 leaves, since their line numbers move.
* **2026-10-07, resumed.**
  [0012-PLAN-bring-your-own-cli.md](0012-PLAN-bring-your-own-cli.md) is
  complete at `v0.6.0`.
  * **Resolved by 0012 Step 5,** on lines that step rewrote: F2, F19 and
    F26. Steps 2 to 4 here check them again rather than edit them.
  * **Dropped,** as the pause said: F4, F9's adapters table, F15 and F20,
    and the mention of A12 in F1.
  * **F1** becomes "the current release is `v0.6.0`", which 0012 wrote.
  * The other findings stand, against the tree at `6bf8fad`. This PLAN
    awaits the owner's approval, as before.

## Goal

Every living document is true of the tree at `v0.5.0`. The unexecuted
records 0007, 0008 and 0009 carry amendments that correct their stale
facts. Every record's front matter `date` is the date it was last
updated. AGENTS.md's Records rule matches the repository's practice.
Nothing else changes: no Go code, no release, no completed record's body.

## Scope

### Findings

From the audit of 2026-10-07 (MADR, Context). Line numbers are those of
the tree at `11d8758`. "Decided" marks a finding whose fix the owner
chose (MADR, More Information).

| # | Where | Finding | Fix | Step |
| :--- | :--- | :--- | :--- | :--- |
| F1 | `README.md:19` | the current release is `v0.4.0` | `v0.5.0`, with A12's change in the Commands bullet | 2 |
| F2 | `README.md:13-15` | "The repository starts without packages" | what the library is now (decided) | 2 |
| F3 | `README.md:21-27` | the reused cell buffer, text measured as Bubble Tea writes it and a following theme dated `v0.1.0`; they came in `v0.2.0` | a `since v0.2.0` clause (decided) | 2 |
| F4 | `README.md:46-49` | the adapters are not said to be released | `command/cobracmd` and `command/kongcmd` at `v0.1.0`; glamour planned | 2 |
| F5 | `AGENTS.md:22-25` | only `internal/cells` may import ultraviolet | `internal/cells` and `internal/termevent` (with `termeventtest`), as `.golangci.yml` and `docs/architecture.md` say | 2 |
| F6 | `AGENTS.md`, Records | one slug per MADR and PLAN; no number reused | the skill's rule: several PLANs with their own slugs; a REPORT takes its record's number (decided) | 2 |
| F7 | `docs/architecture.md:23` | `workspace` imports lipgloss; `colorprofile` and `x/ansi` missing | the imports `go list` gives | 2 |
| F8 | `docs/architecture.md:14`, the Tree | "four internal ones"; `tuitest/internal/clash` absent | five, with `tuitest/internal/clash` (tests only) | 2 |
| F9 | `docs/architecture.md:21-53` | no table of the adapters' APIs; the `Registry` list incomplete | an adapters table from `go doc`; `Remove`, `CancelAll`, `Version`, `NewRegistry` and its options in the list | 2 |
| F10 | `docs/architecture.md:285-287` | "a help footer" not built | `workspace` implements `help.KeyMap`; a help-footer pane beyond it is what is not built | 2 |
| F11 | `docs/architecture.md:150-151` | goldens "under each testdata/golden/" | where a package renders | 2 |
| F12 | `docs/architecture.md:118` | `0010-REPORT §2` cited by bare number | the full filename, linked | 2 |
| F13 | `docs/README.md:12` | 0001-MADR's row does not name its amendments | "accepted; A1–A3 accepted", the amendments' text showing them decided | 2 |
| F14 | `docs/README.md:53` | "why the gates fail on an empty module" | as history: "failed" | 2 |
| F15 | `docs/README.md:90` | Cobra or Kong rows point at a design record | the commands guide's new section | 2 |
| F16 | `docs/README.md:5` | "Decisions and their implementation plans" | decisions, plans and reports | 2 |
| F17 | `docs/README.md:66` | "after 0009" by bare number | the full filename | 2 |
| F18 | `docs/guides/commands.md:25-39,104-109` | the example's slash name `resize` makes `workspace.Commands` fail to register; `ws.Resize` gets a split name, not a separator ID | an action the workspace does not have, proved in a scratch module (decided) | 3 |
| F19 | `docs/guides/commands.md:16-17` | "A1 to A9" | A1 to A12 | 3 |
| F20 | `docs/guides/commands.md` | no Cobra or Kong front end; the opening and the shell section name `command/cli` only; "What the library never does" without Cobra's completion caveat | a section on both front ends, from their package documentation; the opening, A12's reason (Kong's grammar) and the caveat | 3 |
| F21 | `docs/guides/commands.md:285` | `workspace.layout.use` exists only with `WithLayouts` | say so | 3 |
| F22 | `docs/guides/terminal-capabilities.md:87-89` | `CapsMsg` "never before tea's colour profile" | unless the deadline fires (`termcap/prober.go:511-519`) | 3 |
| F23 | `docs/guides/terminal-capabilities.md:15-16` | "A1 to A4" | A1 to A5 | 3 |
| F24 | `docs/guides/terminal-capabilities.md:119` | `WithoutHeuristic` gates omit the foreground | add OSC 10 (`prober.go:558-565`) | 3 |
| F25 | `docs/guides/terminal-capabilities.md:124` | `LC_` + name "over SSH" | read always (`termcap/env.go:77-85`) | 3 |
| F26 | `docs/guides/releasing.md:58,75` | `git tag -a` without `-m` opens an editor | `-m` with the tag's name, as every tag has | 3 |
| F27 | `docs/guides/releasing.md`, the smoke test | stops at `go build` | `go env GOWORK` empty, `go vet ./...`, and running the program, as the 0006 records did | 3 |
| F28 | `docs/guides/releasing.md` | no disclosure guard; "Add a module" shows an existing module | the guard before a push; `<dir>` | 3 |
| F29 | `docs/guides/building-workspaces.md` | no pointer to the workspace's commands; `ws.Push(...)` drops its `tea.Cmd` | a link to the commands guide; `return m, ws.Push(...)` | 3 |
| F30 | front matter `date` of `0001-PLAN`, `0009-PLAN`, `0004-MADR`, `0005-MADR`, `0006-MADR`, `0009-MADR`, `0010-MADR` | not the date last updated | each record's last amendment's or revision's date (decided) | 4 |
| F31 | `0007-MADR`, `0007-PLAN` | §9's command IDs do not exist; `glyph.Table` is `glyph.Set`, which lacks the modifier and arrow glyphs A1 wants; "(§8)" is §9; `KeyMapper` is read since `v0.2.0`; `Dispatch` needs an `Origin`; 0006 is complete | amendment A2, the bindings an open owner question (decided) | 4 |
| F32 | `0008-MADR` | the width method attributed to `0002-PLAN-harden-workspace-v0-1-1.md`; the provider, hits and prompts predate 0006 A4 to A6 | amendment A1 | 4 |
| F33 | `0009-MADR`, `0009-PLAN` | `render.go:274` is now `460`; the width method's attribution; Step 7 waits on a complete 0010; no "theme revision" exists | amendment A3, the revision's source an open owner question | 4 |

Not changed, and why:

* `0010-MADR` A2 cites "Phase 8 of 0010-PLAN", which exists only in that
  PLAN's execution record: the citation resolves, and the record is
  complete.
* `.opencode/rules.md` checks `~/.claude/skills`: that is where the
  skill is, for OpenCode as for Claude.
* `0002-MADR` and `0001-REPORT` cite other repositories' records by
  number in their front matter: historical metadata, with the filenames
  given in the bodies.
* Follow-on short citations (`0010-MADR A1` after the full filename on
  the same page; a guide's "MADR A2" for the MADR it names at the top)
  resolve, and stay.
* `0006-PLAN`'s `date: 2026-10-07` follows the records' UTC dating.

### Out of scope

* Go code, package documentation in `.go` files, releases and tags.
* The decision 0007 must take on its default bindings, and 0009 on its
  theme revision: recorded as owner questions in their amendments.
* `git push` and tags.

## Implementation Steps

### Rules

1. A step starts when the previous one is committed. The owner commits;
   the agent stages nothing.
2. Each edit is made by a script that asserts its anchor once, or by a
   single exact edit, and the result is read back.
3. **Checks of every step:** `markdownlint-cli2 --config
   .markdownlint-cli2.jsonc` on each changed file (a record through a
   renamed copy, which the configuration's globs skip); the
   relative-link checker on each changed file; the citation checker
   (every `NNNN-KIND §/A/D/Step/Phase/Q` citation resolves) on each
   changed record; the identifier scan of the diff.
4. A fact written is checked against its source in the same step:
   `go doc`, `go list`, the code, the tags.
5. Anything the step does not cover stops it for the owner.

### Step 1: records

`docs/decisions/0011-MADR-docs-accuracy-after-v0-5-0.md`, this PLAN,
and their two rows in `docs/README.md`. **Done when** the owner approves
this PLAN.

### Step 2: the top-level documents (F1 to F17)

`README.md`, `AGENTS.md`, `docs/README.md`, `docs/architecture.md`. F9's
tables come from `go doc -short` of each package, with `GOWORK=off` in
each nested module. **Done when** the checks are clean and each finding's
line reads as its fix says.

### Step 3: the guides (F18 to F29)

* **F18:** the new example is written first in a scratch module outside
  the repository, its `go.mod` replacing the root with the working tree.
  It registers the example beside `workspace.Commands(ws)`, and runs it
  by its slash line and through `Dispatch`. Only then does it go in the
  guide.
* **F20:** the front-end section is written from `go doc` of
  `command/cobracmd`, `command/cobracmd/docs` and `command/kongcmd`, and
  from their examples. Its claims about exit codes, verbs, completion and
  the Kong-native grammar cite the package documentation, and D42 for
  the differences from `command/cli`.
* **Every Go example** in the four guides is compiled again in the
  scratch module after the edits.

**Done when** the checks are clean, the examples compile, and the F18
example registers beside the workspace's commands.

### Step 4: the records (F30 to F33)

* **F30:** front matter `date` only, each to the date of the record's
  last amendment or revision, read from the body.
* **F31:** `0007-MADR` A2, *Status: accepted (2026-10-07)* for the
  corrections, its default bindings an open owner question with the
  options the audit found; `0007-PLAN` gains a revision entry naming the
  steps A2 touches (Steps 7, 8 and 9 and the preconditions).
* **F32:** `0008-MADR` A1.
* **F33:** `0009-MADR` A3, the theme revision's source an open owner
  question; `0009-PLAN` gains a revision entry that Step 7's 0010
  precondition is met.
* `docs/README.md`'s rows for 0007, 0008 and 0009 name the new
  amendments.

**Done when** the checks are clean and the citation checker resolves
every citation in the three records.

### Step 5: close-out

Each finding again, item by item, in the execution record; the link and
citation checkers over every Markdown file; the guides' examples
compiled once more; this PLAN `complete`, with its row. **Done when**
every Verification item holds.

## Verification

* F1 to F33 each read as their fix says, with the evidence named.
* Every relative link and anchor in the repository's Markdown resolves,
  and every cross-record citation resolves.
* Every Go example in the four guides compiles against the tree, and the
  commands guide's example registers beside `workspace.Commands`.
* markdownlint is clean on every changed file, and the identifier scan
  of every step's diff finds nothing.
* No `.go` file, `go.mod`, `go.sum`, tag or completed record's body
  changed (`git diff --stat` per step).

## Rollout and Rollback

* **Rollout:** one commit per step, by the owner, pushed when the owner
  chooses. No release.
* **Rollback:** each step is one commit of Markdown only; reverting it
  restores the previous text.

## Execution Record

Not started.
