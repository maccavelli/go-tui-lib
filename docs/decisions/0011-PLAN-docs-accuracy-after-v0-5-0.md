---
status: in-progress
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

The owner approved execution on 2026-10-07 ("proceed to 0011"). Step 1,
the records, was committed as `376106e`. The PLAN resumed after
[0012-PLAN-bring-your-own-cli.md](0012-PLAN-bring-your-own-cli.md), as its
second revision says.

### Step 2: the top-level documents

No deviation.

Each finding was re-checked against the tree at `6bf8fad` before the
edit:

* **F1 and F2:** already as their fixes say. 0012 wrote "The current
  release is `v0.6.0`" and the new opening. A12's mention was dropped with
  the check.
* **F4 and F15:** dropped (the first revision). F9's adapters table was
  dropped too; its `Registry` list stands.

| # | Re-checked | Now |
| :--- | :--- | :--- |
| F3 | `README.md:23-29` dated the cell buffer, the width measurement and the following theme to `v0.1.0`; `internal/cells` is absent from `v0.1.0` and present in `v0.2.0` | "Since `v0.2.0` it draws into one reused cell buffer, measures text as Bubble Tea writes it, and follows the terminal's theme" |
| F5 | `go list` shows ultraviolet imported by `internal/termevent` and `termeventtest`; `.golangci.yml`'s `ultraviolet` rule excludes `internal/cells` and `internal/termevent` | AGENTS.md names both, citing 0005-MADR §1 |
| F6 | AGENTS.md: "A MADR and its PLAN share the same number and the same slug", "Never reuse a number" | a lone PLAN copies the MADR's slug, several PLANs take their own (0002-PLAN-harden as the example), a REPORT about a record takes its number (0010-REPORT as the example); "Never give a new decision a number already used" |
| F7 | `go list -f '{{.Imports}}' ./workspace`: `bubbles/v2/help`, `bubbles/v2/key`, `bubbletea/v2`, `colorprofile`, `x/ansi`, no lipgloss | the graph lists those |
| F8 | `go list ./...`: `internal/cells`, `internal/conformance`, `internal/termevent`, `internal/termevent/termeventtest`, `tuitest/internal/clash` | "five internal ones"; the tree names `tuitest/internal/clash/`, whose documentation says it proves tuitest's `-update` flag does not clash |
| F9 | `go doc ./command Registry`: `NewRegistry` with `WithGate`, `WithAuditor`, `WithPrefixer`, `WithLoop`, and `CancelAll`, `Remove`, `Version` missing from the row | the row names them |
| F10 | `workspace/help.go` implements `help.KeyMap` | "a help-footer pane beyond `workspace`'s `help.KeyMap`" |
| F11 | `glyph`, `termsvc`, `tuitest` and `when` have no `testdata/golden` (the audit named three; `when` is the fourth) | "goldens under testdata/golden/ where a package renders" |
| F12 | `docs/architecture.md:120` cited "0010-REPORT §2" by number | the full filename, linked |
| F13 | 0001-MADR's A1 records facts and "The decision is unchanged"; A2 and A3 carry the owner's decisions | "accepted; A1–A3 recorded", not "accepted": A1 decides nothing |
| F14 | "know why the gates fail on an empty module" | "once failed" |
| F16 | "Decisions and their implementation plans" | "Decisions, their implementation plans, and reports" |
| F17 | "after 0009" by number | the full filename, linked |

**Checks:**

* markdownlint on the four files: 0 issues.
* The relative-link check: 0 broken. The citation checker: 247 of 247
  resolve.
* The identifier scan of the diff: no match.
* No Go file changed.

### Step 3: the guides

The owner approved it on 2026-10-07 ("proceed").

#### Deviations

* **D1 (2026-10-07): the `list` example completed.**
  * **Found:** the compile check of every guide example failed on
    `docs/guides/building-workspaces.md`'s `list` pane in "Focus". It
    defines `Update` but no `View`, so `return l, nil` fails with "list
    does not implement workspace.Pane (missing method View)". The defect
    predates this PLAN and is not among F18–F29.
  * **Asked,** with two options: complete the example, or mark it an
    excerpt and have the check supply `View`.
  * **The owner chose** to complete it, the recommendation. It gains
    `func (l list) View(width, height int) string { return
    strings.Join(l.items, "\n") }`.
  * **Files:** none beyond the guide this step already edits.

#### What changed

Each finding was re-checked against the tree at `0369ca7` before the
edit.

| # | Re-checked | Now |
| :--- | :--- | :--- |
| F18 | `commands.md`'s first example registered `layout.resize` with the slash name `resize`. `workspace.Commands` takes `resize`, and a scratch run refused it: `command: workspace.resize: slash name "resize" is taken by layout.resize`. It also passed a split name to `ws.Resize`, which takes a separator ID | a read-only `session.search` with the slash name `search`, which no workspace command takes, and a note listing the workspace's slash names. The registration line follows |
| F19 | "A1 to A13", which 0012 Step 5 wrote; 0006-MADR has A1 to A13 | unchanged |
| F20 | dropped (the first revision) | — |
| F21 | `workspace.layout.use` is appended only when `WithLayouts` gives layouts (`workspace/commands.go:51-53`) | the table row says so |
| F22 | `deliver` holds the message for tea's profile unless `TimedOut` (`termcap/prober.go:511-523`) | DA1's answer waits for the profile; the deadline does not |
| F23 | 0005-MADR has A1 to A5 | "A1 to A5" |
| F24 | the foreground query is `Gated` (`termcap/prober.go:564`) | the list names the foreground (OSC 10) |
| F25 | `LC_` + name and name are both read, the plain name last (`termcap/env.go:77-85`) | read always; the plain name wins when both are set |
| F26 | every `git tag -a` has `-m`, which 0012 Step 5 wrote | unchanged |
| F27 | the smoke test stopped at `go build` | `test -z "$(go env GOWORK)"`, `go vet ./...` and `go run .`, with a sentence on why |
| F28 | no disclosure guard; "Add a module" named `stream/glamourmd` | the guard's command after the owner's paragraph, and in the two release procedures; "Add a module" uses `<dir>` |
| F29 | `ws.Push` returns a `tea.Cmd` (`go doc ./workspace Workspace.Push`), and the example dropped it; no pointer to the workspace's commands | `return m, ws.Push(…)` with a sentence on why; a paragraph in "Keys and the mouse" links the commands guide's "The workspace's commands" |

#### Checks

* **The F18 proof,** in a scratch module that replaces the root with the
  tree. The new example, verbatim, registers beside
  `workspace.Commands(ws)`: 16 commands. It runs by its slash line
  (`/search hello limit=1` gives "1 matches") and through `Dispatch`
  ("2 matches"). The old example, in the same program, is refused as
  above, so the proof fails when the finding holds.
* **Every Go example in the guides compiles.** A scratch module holds
  each of the 22 blocks as its own `main` package, verbatim, with stubs
  only for what a block leaves undefined; the two `{ … }` placeholders
  become `{ panic(err) }`. `go vet` passes on 22 of 22. Before D1 it
  failed on the `list` block, so the check is seen to fail on a broken
  example.
* markdownlint on the four guides: 0 issues.
* The relative-link check on the four guides: 0 broken.
* The identifier scan of the diff: no match.
* No Go file, `go.mod` or `go.sum` changed.

### Step 4: the records

The owner approved it on 2026-10-07 ("Commit to main then proceed"),
after Step 3 was committed as `c79e140`.

No deviation.

#### What changed

Each finding was re-checked against the tree at `c79e140`.

| # | Re-checked | Now |
| :--- | :--- | :--- |
| F30 | 0001-PLAN's last phase is 2026-10-03; 0004-MADR's last amendment 2026-10-04 (A2); 0005-MADR's 2026-10-05 (A5); 0006-MADR's and 0010-MADR's 2026-10-07 (A13, A3). 0009's last revision was 2026-10-03, and today's A3 supersedes it | each `date` is its record's last amendment or revision. 0007-MADR, 0007-PLAN, 0008-MADR, 0009-MADR and 0009-PLAN, amended today, are `2026-10-07` |
| F31 | §9's IDs `workspace.focus.pane` and `workspace.resize.left` and its siblings are not registered (`go doc ./workspace Commands`); `glyph.Table` does not exist and `glyph.Set` has no modifier or arrow glyph (`go doc ./glyph Set`); "(§8)" names §9's `KeyContexter`; `KeyMapper` is read by `workspace/help.go:52`, shipped in `v0.2.0` (`git tag --contains`); `Dispatch` refuses a zero `Origin` (`go doc ./command Origin`); 0006-PLAN is complete | 0007-MADR A2, accepted for the corrections, with owner question Q8 on the bindings for focus by index and directional resize, open; §3's "(§8)" annotated in place; 0007-PLAN's revision names the prerequisites and Steps 7, 8 and 9 |
| F32 | the Context credits the width method to 0002-PLAN-harden, which lists it as out of scope (`:61`); it came with 0004-MADR §3; `OriginPalette`, `SurfacePalette`, `Available(c, s)`, the `Prompt` and `Forward` kinds and `PromptMsg` exist (`go doc ./command`) | 0008-MADR A1, accepted |
| F33 | `local` is at `workspace/render.go:460-478`; More Information credits the width method to 0002-PLAN-harden; 0010-PLAN is `complete`; no theme revision exists, only the workspace's unexported `themeGen` (`workspace/workspace.go:144`, `:310`) | 0009-MADR A3, accepted for the corrections, with owner question Q9 on the theme revision's source, open; 0009-PLAN's revision says Step 7's precondition is met |

`docs/README.md`'s rows for 0007, 0008 and 0009 name the new amendments
and the open questions.

#### Checks

* **markdownlint,** through renamed copies of the eleven changed records.
  * No issue in what this step wrote.
  * Ten issues remain in 0005-MADR (MD029), 0006-MADR (MD029) and
    0010-MADR (MD010, hard tabs in a code block). The same ten are in
    `HEAD`'s copies, and this step changed only those records' `date`
    lines. Completed records' bodies are out of this PLAN's scope.
  * `docs/README.md`: 0 issues.
* **The relative-link check:** two reports, both in text inside backticks
  on lines this step did not touch, which the checker reads as links:
  * 0001-PLAN:652 quotes a test's planted link to a missing file;
  * 0009-MADR:388 quotes goose's rule for a link target not yet closed.

  Nothing else is broken.
* **The citation checker:** 284 of 284 resolve.
* **The identifier scan of the diff:** no match.
* **No Go file, `go.mod` or `go.sum` changed.**
