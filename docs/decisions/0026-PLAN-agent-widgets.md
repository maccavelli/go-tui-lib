---
status: proposed
date: 2026-11-14
associated-madr: "0026-MADR-agent-widgets.md"
---
# Implement the agent widgets: the diff view, tool card, activity row, note and feed in v2.13.0, and the status row, spinner, plan checklist and thought in v2.14.0

Associated MADR: [0026-MADR-agent-widgets.md](0026-MADR-agent-widgets.md)

*(2026-11-14, the owner's ruling "Record now, build later", relayed by the
coordinator: the MADR is accepted, with Q1 to Q5 ruled, each the
recommendation. This PLAN stays `proposed`. By the owner's ruling, building
starts only after go-llmprovider-sdk `v1.31.0` and gobble-cli `v0.7.0`
are released, and only once the owner approves this PLAN's execution.)*

*(2026-11-15, by
[0027-MADR-keep-docs-accurate-and-current.md](0027-MADR-keep-docs-accurate-and-current.md)
D5: each step documents what it adds as it lands: a doc comment on
every exported name it adds, by that record's D1; `diffview`,
`agentui` and `agentui/agentuitea` in `docs/architecture.md` from the
step that adds each; and a line under `CHANGELOG.md`'s `Unreleased`.
Steps 9 and 14 also turn `Unreleased` into the release's entry. Step
2, which rewrites `glyph/glyph.go`, also gives each exported name in
that file the comment that record's D1 asks for, since 0027-PLAN's
Phase D leaves the file to it.)*

## Goal

`diffview` and `agentui` exist as Charm-free cores, and
`agentui/agentuitea` holds `FeedPane` and `StatusPane`, as the MADR's D1 to
D9 describe, so that gobble-cli's full-screen mode can draw each
`internal/curate` item (gobble-cli
`docs/decisions/0056-MADR-curate-the-terminal-session-view-with-view-modes.md`
D13) through one go-tui-lib type, and a line-mode CLI can draw the same
rows without Bubble Tea.

Done means every item under Verification holds, CI is green on the pushed
tree and on each tag, both releases are tagged by the owner, and each
tag's smoke test passes from the proxy.

## Scope

### Facts re-read before execution (at `855aa72`)

Each is re-read at the tree its step starts on; one that has changed is a
deviation.

| Fact | Step |
| :--- | :--- |
| `glyph.Set` has `Done`, `Failed`, `Running`, `Pending` and `Heading`, no fold marker, minus, separator or spinner frames (`glyph/glyph.go:61-70`); every field is a string or a struct of strings, so `Set` is comparable | 2 |
| The Unicode states are `✓ ✗ ● ○` and `Heading` `▸` (`glyph/glyph.go:125-126`); the ASCII ones `+ x * -` and `>` (`:158-159`) | 2 |
| `stream.Bounded`'s defaults are 64 KiB, 20 head lines, 20 tail lines and 4 KiB a line (`stream/bounded.go:17-20`); `Render` hard-wraps each line at the width (`:164-172`); its options are `stream.Option`s (`stream/options.go:152-173`) | 5 |
| `stream.Style` is SGR attributes and eight foreground colours, the zero value styling nothing (`stream/style.go:13-60`) | 3, 4 |
| `msgcat.Format` and `msgcat.Check` (`msgcat/msgcat.go:40`, `:55`); `ask` gives its words by `Messages()` and `WithCatalog` (`ask/doc.go:30`, `ask/options.go:61`) | 4 |
| `inline.Region`'s `Draw`, `Print`, `Commit`, `Clear`, `Resize`, `Timeout`, `HandleTimeout` and `Restore` each return the bytes to write (`inline/region.go:88-276`) | 11 |
| `inlinetest.Screen` has `Cooked` (`inline/inlinetest/screen.go:42`) | 11 |
| `frame.Demand` and `Slow`'s 83 ms (`frame/demand.go:12`, `:48`); `frametea.TickCmd` | 10 |
| `streamtea.DocPane` is the pattern for a pane over a core: theme ID as revision, follow the tail, `DefaultRules` and the scope (`stream/streamtea/pane.go:26`) | 8 |
| `internal/conformance`'s `cores` (`internal/conformance/charmfree_test.go:18`) and depguard's `charmfree` rule (`.golangci.yml:107`) | 3, 4 |
| `sharedNames` (`internal/conformance/conformance_test.go:473`); `termsvc.Status` exists (`termsvc/clipboard.go:26`) | 1 |
| `v2.12.0` is the latest tag, and `make apicheck` compares with it | 9 |

### In scope

| Step | Files | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0026-*`, `docs/README.md`, `docs/glossary.md` | the MADR accepted with the owner's answers; this PLAN `in-progress`; the names reserved |
| 2 | `glyph/` | the new glyphs |
| 3 | `diffview/`, `internal/conformance/`, `.golangci.yml` | the diff core |
| 4 | `agentui/`, `internal/conformance/`, `.golangci.yml` | enums, `Look`, `Accents`, `Element`, `Foldable`, the messages, `Note` |
| 5 | `agentui/` | `ToolCard` |
| 6 | `agentui/` | `ActivityRow` |
| 7 | `agentui/` | `Feed`, `FromDoc` |
| 8 | `agentui/agentuitea/` | `FeedPane` |
| 9 | `docs/`, `README.md` | the guide, the release `v2.13.0` |
| 10 | `agentui/`, `agentui/agentuitea/` | `StatusRow`, `StatusPane`, `StatusMsg` |
| 11 | `agentui/` | `Spinner` |
| 12 | `agentui/` | `Checklist`, `Step`, `StepState` |
| 13 | `agentui/` | `Thought` |
| 14 | `docs/`, `README.md` | the guide, the release `v2.14.0`, the close-out |

### Out of scope

* Everything the MADR's "Not in scope" lists.
* gobble-cli: its renderer, its pins and its records. Each adoption is a
  gobble-cli plan step (gobble-cli
  `docs/decisions/0032-MADR-go-tui-lib-integration-boundary.md` D7).
* Any tag. Each is the owner's.

## Prerequisites

* The owner has accepted the MADR, answered Q1 to Q5, and approved this
  PLAN. An answer other than a recommendation amends the MADR's decision
  and this PLAN's steps first. *(2026-11-14: the MADR is accepted and Q1
  to Q5 are answered, each the recommendation, so no step changes; the
  approval of this PLAN is still to come.)*
* go-llmprovider-sdk `v1.31.0` and gobble-cli `v0.7.0` are released (the
  owner's ruling of 2026-11-14, "Record now, build later").
* `v2.12.0` is the latest tag. A release between this approval and Step 9,
  or between Step 9 and Step 14, moves the base and the name: Step 9 uses
  the next minor, and Step 14 the one after it.

## Rules for every step

[0024-PLAN-command-and-control-opt-outs.md](0024-PLAN-command-and-control-opt-outs.md)'s
rules 1 to 8, with rule 4's base the tag before the release the step
belongs to (`v2.12.0` for Steps 2 to 9, `v2.13.0` for Steps 10 to 14), and
these:

9. **Every test is seen failing on a planted break** before it is
   committed: through `python3 -m tools.verify mutate` on a scratch copy,
   or by running the new test on a scratch export of the commit before the
   step. Each step names its breaks below. The failure is quoted in the
   execution record. The working tree is never dirtied to do it.
10. **Goldens** are `tuitest.Golden` over `tuitest.Matrix{Widths: []int{24,
    60, 100}}`: {colour, no colour} × {UTF-8, ASCII} × three widths, twelve
    files per case, written with `-tuitest.update` and read in full before
    they are committed. Each test also checks every row with
    `tuitest.Fits`.
11. **Line forms** are tested on `inlinetest.Screen`, raw and `Cooked`, and
    each drawing split at `?2026l` is checked frame by frame.
12. **Budgets** hold bytes and allocations, never time, set above the
    measured value, and are seen failing twice
    ([0025-PLAN-charm-free-stream-allocations.md](0025-PLAN-charm-free-stream-allocations.md)
    rules 10 to 12).
13. **Heavy runs queue.** `go test ./...` over a module, any `-race` run,
    fuzzing, `make test`, `make lint`, `make release-check`,
    `make pre-add-check` without `FILES`, and `python3 -m tools.verify
    gates` are heavy. Before them, `pick-host` (the `available-dev-hosts`
    skill) names a host; each runs there through `run-on-host`, or on the
    session host through `run-heavy -- <command>`, one heavy run at a time
    per host; `run-heavy --status` shows the queue. The Linux runs may go
    to the WSL instance when the Linux machine is busy. A single package's
    `go test ./<pkg>/` may run on the session host. A session that
    dispatches agents never briefs two heavy runs onto one host at once.
14. **Defaults.** Each default a step adds is listed in
    `docs/guides/defaults.md` with its control in the same step, and
    `TestDefaultsInventory` resolves it (AGENTS.md, API rule 12).
15. **Names.** A type name not in the MADR's D2 is checked with
    `python3 -m tools.verify names` before it is written; a clash is a
    deviation.

## Implementation Steps

### Step 1: records

* The MADR `accepted`, with the owner's words and Q1 to Q5's answers as
  dated notes under each question; this PLAN `in-progress`;
  `docs/README.md`'s two rows. *(2026-11-14: the MADR's acceptance, its
  rulings and the index rows were recorded ahead of execution, by the
  owner's "Record now, build later"; Step 1 sets this PLAN `in-progress`,
  the index row with it, and does the rest of this step.)*
* `docs/glossary.md`, "Reserved by accepted records": `diffview`'s
  `Diff`, `Hunk`, `Line`, `Op`, `Stat` and `Colors`; `agentui`'s
  `ToolKind`, `ToolState`, `Fold`, `ToolCard`, `ActivityRow`, `StatusRow`,
  `Spinner`, `Checklist`, `Step`, `StepState`, `Thought`, `Note`, `Tone`,
  `Element`, `Foldable`, `Feed`, `Look` and `Accents`; `agentuitea`'s
  `FeedPane`, `StatusPane` and `StatusMsg`. `python3 -m tools.verify names`
  over them, its output quoted. *(2026-11-14: the reservation was made on
  the MADR's acceptance, by the repository rule and MADR D2, ahead of this
  step; Step 1 checks the glossary's rows still match D2 and quotes the
  check's output.)*
* Records only.

### Step 2: the glyphs (`v2.13.0`)

* `glyph.Set` gains `Folded` and `Unfolded` (`▸`/`▾`, `>`/`v`), `Minus`
  (`−`, `-`), `Join` (`·`, `-`) and `Spin`, a fixed-length array of frames
  (Unicode braille dots, ASCII `| / - \`), each one cell in both sets; the
  legacy tier's forms, or ASCII where it has none. Docs name each field's
  use.
* **Tests and their breaks:**
  * the existing glyph table test extended: every new glyph one cell by
    `ansi.StringWidth` in each tier; break: a two-cell Unicode form;
  * a test that `Set` is comparable (`==` compiles and holds for two
    `Unicode()` calls); break: `Spin` as a slice, which fails to compile in
    the scratch copy;
  * every ASCII form is ASCII; break: `Unfolded` as `▾` in `ASCII()`.
* `make apicheck`: 0 incompatible changes (fields added).

### Step 3: `diffview`

* `Diff{Path, Hunks}`, `Hunk`, `Line`, `Op` (`context`, `insert`,
  `delete`, by `internal/enum`), `Stat` and `Diff.Stat()`; `Compute(path,
  old, new, ...Option)`, Myers' O(ND) over lines with `WithContext` (3)
  and `WithComputeLimit` (the PLAN's measured default, past which one
  replace-all hunk); `Diff.Unified(width, ...Option) []string` and
  `Diff.StatRow(width, ...Option) string`, with `WithGlyphs`,
  `WithMethod`, `WithColors`, `WithCatalog` and `WithGutter`; `Colors`
  and `DefaultColors()`; `Messages()`.
* Text cleaned on the way in (MADR D3); a row cut, never wrapped; the stat
  row's path cut from its left.
* `diffview` joins `cores` and depguard's `charmfree` rule.
* **Tests and their breaks:**
  * `TestUnifiedGolden`, three diffs (an edit, an add-only file, a
    delete-only file) with and without the gutter, across the matrix;
    break: the sign column dropped;
  * `TestStatGolden`; break: the path cut from its right;
  * `TestStatCounts`, `Stat` from lines; break: a context line counted as
    added;
  * `TestComputeApplies`: applying `Compute`'s hunks to `old` gives `new`,
    over a table of edits; break: a hunk's new start off by one;
  * `TestComputeLimit`: past the bound, one hunk, every old line removed and
    every new one inserted; break: the bound ignored, caught by an
    allocation budget;
  * `FuzzCompute`: the apply property, and every unified row fits its
    width; break: the same off-by-one, found from the seed corpus;
  * `TestHostileText`: C0, C1, CSI, OSC and bidi controls in a line and a
    path are cleaned; break: the line's cleaning skipped;
  * `TestOpText`, the enum's round trip; break: two tokens swapped;
  * `TestCoresAreCharmFree` with `diffview` listed; break: a Lip Gloss
    import in a scratch copy, the test naming its chain.
* The defaults: context, the compute bound, the gutter, in
  `docs/guides/defaults.md`.

### Step 4: `agentui`'s foundations, and `Note`

* `ToolKind`, `ToolState`, `StepState` (declared here for its tokens),
  `Fold` and `Tone`, each with `String`, `MarshalText` and `UnmarshalText`
  from `internal/enum`, the tokens of MADR D4.
* `Look` (glyphs, method, `Accents`, `diffview.Colors`, catalogue,
  revision) and `Accents`, with `DefaultAccents()`; `Element`; `Foldable`;
  `Messages()` and the `Msg…` IDs for every word of MADR D6.
* `Note`, from `NewNote(id, text, tone)`: hard-wrapped rows, a warning or a
  failure led by its word.
* `agentui` joins `cores` and `charmfree`.
* **Tests and their breaks:**
  * `TestTokensAreACP`: each enum's tokens equal MADR D4's table, written
    out in the test; break: `in_progress` spelt `in-progress`;
  * `TestEnumText` for each enum; break: a token missing;
  * `TestMessagesCheck`: `msgcat.Check(Messages(), Messages())` passes, and
    a catalogue with a missing verb fails; break: a default's `%d` removed;
  * `TestNoteGolden`, each tone, a long line and a raw JSON line, across
    the matrix; break: the warning's word dropped, so the no-colour golden
    differs;
  * `TestCoresAreCharmFree` with `agentui` listed; break as in Step 3.

### Step 5: `ToolCard`

* `NewToolCard(id, kind, title, ...Option)`; `SetState`, `SetTitle`,
  `SetSummary`, `Write` (its `stream.Bounded`), `SetTail(lines, above)`,
  `AddDiff`, `AddTerminal`, `Close`; `Foldable`; `Version`; `Rows(width,
  look)`. Options: `WithTailLines` (5), `WithOutput(stream options…)`,
  `WithFold` (the starting fold, `Summary` by default).
* The head row and each fold as MADR D1 has them; an edit's head row ends
  with its diffs' summed stat.
* **Tests and their breaks:**
  * `TestToolCardGolden`: an execute call (output under and over the
    bound), a failed one, an edit with two diffs, a read, a call with a
    terminal reference, each at all four folds, across the matrix; break:
    `Full` drawing the summary as well as the output;
  * `TestToolCardTail`: the program's tail wins over the output's last
    lines, and the count above is drawn; break: `SetTail` ignored;
  * `TestToolCardToggle`: `Summary` opens to `Full` and closes to
    `Summary`; `Full` closes to `Summary`; break: `Toggle` closing to
    `Hidden`;
  * `TestToolCardHostile`: a title with a newline and an escape, output
    with OSC 52 and a 1 MiB line; break: the title's cleaning skipped;
  * `TestToolCardVersion`: each setter bumps the version, a draw does not;
    break: `Write` not bumping it;
  * `TestToolCardBudget`: a card's heap with 1 MiB written stays under the
    bound's budget; break: the output kept outside the `Bounded`.

### Step 6: `ActivityRow`

* `NewActivityRow(id, ...Option)`; `Add(*ToolCard)`, `Close`; `Foldable`
  (`Tail` draws as `Summary`); `Version`; `Rows(width, look)`. The label is
  each kind present, in first-seen order, in the running or done tense, one
  or many; the state is `Failed` with "%d failed" when any failed, else
  `Running` when any runs, else `Done`; the members' titles follow,
  joined by `Join`, cut to fit, the rest as `+n`; an edit member shows its
  stat. A row of one draws as its card's head row. Opened, each member's
  rows follow at its own fold, a hidden member at `Summary`.
* **Tests and their breaks:**
  * `TestActivityRowGolden`: one call; three executes; mixed kinds; one
    failed; an open row with a running member; edits with stats; each
    closed and opened, across the matrix; break: `+n` counting a shown
    title;
  * `TestActivityRowCounts`: the counts and failed count follow the
    members; break: a failed member counted done;
  * `TestActivityRowTense`: present while a member runs or the row is
    open, past once closed; break: tense from the first member only;
  * `TestActivityRowCatalog`: a catalogue replaces every label; break: a
    label formatted from its default directly.

### Step 7: `Feed` and `FromDoc`

* `NewFeed(...Option)` with `WithMaxElements` (the default set here from a
  measured heap budget, recorded) and `WithoutFollow`; `Append`, `Gap`,
  `Element(id)`; `View(width, height, look) []string`; `ScrollBy`, `Top`,
  `Bottom`, `Following`; `FocusNext`, `FocusPrev`, `Focused`, `Toggle`;
  `SetReveal`; `At(row) (id, ok)` for a click. Heights kept per element by
  version, width and look revision; the anchor an element's ID and a row
  in it; only drawn and open elements checked for a new version each
  view; the oldest dropped past the bound, behind "earlier items dropped".
* `FromDoc(id, *stream.Doc) Element`, its version the `Doc`'s.
* **Tests and their breaks:**
  * `TestFeedGolden`: a scripted turn as 0056-MADR D13's items would
    arrive (a thought hidden, three executes grouped, a failure's card at
    `Tail`, an edit row with its diff, a note, a reply) mapped by MADR D4,
    a frame after each `Change`, across the matrix; break: `Gap` ignored;
  * `TestFeedAnchor`: a fold opened above the window and a resize keep the
    first visible row's element and offset; break: the anchor kept as a
    row number;
  * `TestFeedFollow`: following until scrolled up, and again at the
    bottom; break: following lost after a resize;
  * `TestFeedReveal`: a hidden thought draws nothing, and at `Summary` with
    reveal on; break: reveal drawing it `Full`;
  * `TestFeedBound`: past the bound the oldest go, behind the note, and the
    focused element moves to the next; break: the newest dropped;
  * `TestFeedBudget`: 10,000 rows of elements, a resize re-measures only
    what is drawn (allocations bounded); break: every element re-measured.

### Step 8: `agentuitea.FeedPane`

* `NewFeedPane(ws, feed, ...Option)` with `WithID`, `WithTitle`,
  `WithAccents`, `WithCatalog`; the workspace's width method and glyphs;
  the theme's ID as the look's revision; no colour when the workspace's
  profile has none; `DefaultRules`, the command IDs (scroll up and down,
  page, top, bottom, focus next and previous, toggle, reveal) and
  `FeedPaneScope`; a click on a head row toggles it; the focused row marked
  by `glyph.Focus`; a `KeyContexter` and a `CommandConsumer`, as `DocPane`.
* **Tests and their breaks:**
  * `TestFeedPaneGolden` in a workspace, focused and not, across the matrix;
    break: the focus shown by colour alone, so the no-colour golden loses
    it;
  * `TestFeedPaneKeys`: each default rule runs its command; a program's
    keymap without them binds nothing; break: a rule's command ID wrong;
  * `TestFeedPaneClick`: a click on a head row toggles that element; break:
    the hit test off by one row;
  * `TestFeedPaneNoColor`: an ASCII profile gives zero accents; break: the
    accents kept.

### Step 9: the guide and the release `v2.13.0`

* `docs/guides/agent-widgets.md`: draw a card, a row and a diff in a line
  CLI; a feed in a pane; map an ACP client's updates by MADR D4; replace
  the words; each excerpt from a package example, checked by
  `scripts/go-examples.py --check-guide`. `docs/README.md`'s "I want to…"
  rows. `docs/architecture.md`: the packages, the cores, the tree.
  `docs/guides/migrating.md`, "Migrating to `v2.13.0`". `README.md`: the
  current release and a sentence on the widgets.
* `make markdownlint check-records docs-check disclosure-check`; the
  release gates of AGENTS.md, heavy runs by rule 13.
* **The release,** `v2.13.0` or the owner's name, as 0023-PLAN's rule 4
  and 0025-PLAN Step 6 do it: the owner's commit; the disclosure guard over
  the outgoing commits; the owner's push, CI and Docs green; the owner's
  annotated tag, CI green on it; a smoke test from the proxy, a scratch
  module with `GOWORK` empty that requires the tag, builds a feed of one
  row of two cards and an edit, prints the rows of each, and checks that
  the closure of `diffview` and `agentui` for Linux, macOS and Windows
  holds no Charm module but `x/ansi`; its control against `v2.12.0`, which
  fails to build.
* The execution record names the tag; gobble-cli's adoption is its own.

### Step 10: `StatusRow`, `StatusPane` and `StatusMsg` (`v2.14.0`)

* `StatusRow{Thinking, Running, Activity *ActivityRow, Since, Facts,
  Hint}`; `Row(width, now, look) string`; `Demand() frame.Demand`, `Slow`
  while active, `None` idle. The frame from `now − Since` by the interval;
  the facts dropped whole from the last when narrow; the label cut last.
* `agentuitea.StatusPane`, from `NewStatusPane(ws, ...Option)` with
  `WithID`, `WithAccents`, `WithCatalog`, `WithInterval` and
  `WithoutMotion`; `StatusMsg`; ticks by `frametea.TickCmd` at the row's
  demand; one row.
* **Tests and their breaks:**
  * `TestStatusRowGolden`: idle, thinking, an open activity with a running
    call, facts that fit and that do not, at three `now`s, across the
    matrix; break: a fact cut mid-word rather than dropped;
  * `TestStatusRowFrame`: the frame index from `now`; the widest frame's
    cells reserved, so the label never moves; break: the label placed after
    the frame's own width;
  * `TestStatusRowDemand`; break: `Fast` when idle;
  * `TestStatusPaneTicks`: a tick only while the demand is not `None`;
    break: a tick armed when idle;
  * `TestStatusPaneMotion`: `WithoutMotion` draws `Running`; break: the
    option ignored.

### Step 11: `Spinner`

* `NewSpinner(...Option)` with `WithRegion` (default a one-row region),
  `WithInterval` and `WithoutMotion`; `Set(row, look, now)`, `Resize`,
  `Timeout`, `HandleTimeout`, `Print`, `Commit`, `Clear`, `Restore`, each
  returning the bytes to write.
* **Tests and their breaks:**
  * `TestSpinnerScreen` on `inlinetest.Screen`, raw and `Cooked`: frames
    over time are one row at widths 24, 60 and 100; each drawing one
    synchronized update; break: a frame wider than the width;
  * `TestSpinnerPrint`: text above, the row below, in one update; break:
    the row drawn before the text;
  * `TestSpinnerCommit`: the rows in the scrollback and none of the row's
    states; break: `Commit` without clearing the row first;
  * `TestSpinnerTimeout`: the deadline is the next frame, none without
    motion; break: a deadline set with `WithoutMotion`;
  * `TestSpinnerResize`: a narrower width redraws the row cut; break:
    `Resize` not passed to the region.

### Step 12: `Checklist`

* `NewChecklist(id, ...Option)`; `SetSteps([]Step)`; `Foldable` (`Tail`
  as `Full`); `Version`; `Rows(width, look)`: `Full`, each step with its
  state glyph and its text wrapped with a hanging indent; `Summary`, one
  row, done of all and the step in progress.
* **Tests and their breaks:**
  * `TestChecklistGolden`, five steps in each state, each fold, across the
    matrix; break: the in-progress step shown by colour alone;
  * `TestChecklistSummary`: the count and the step in progress, none when
    all are done; break: pending counted done;
  * `TestChecklistHostile`: a step with controls and a newline; break: the
    cleaning skipped.

### Step 13: `Thought`

* `NewThought(id, ...Option)`; `Append`, `Close`; `Foldable`; `Version`;
  `Rows(width, look)`: `Summary`, the label ("thinking" open, "thought"
  closed) and the first line cut; `Full`, the label and the text wrapped at
  whole words in the muted accent.
* **Tests and their breaks:**
  * `TestThoughtGolden`, open and closed, each fold, a long word, across the
    matrix; break: the label dropped in `Full`, so a no-colour thought is
    indistinguishable from a reply;
  * `TestThoughtGrow`: `Append` bumps the version and extends the rows;
    break: a chunk joined without its separator.

### Step 14: the guide, the release `v2.14.0` and the close-out

* `docs/guides/agent-widgets.md` gains the status row, the spinner, the
  checklist and the thought; `docs/architecture.md`;
  `docs/guides/migrating.md`, "Migrating to `v2.14.0`"; `README.md`.
* **The release,** `v2.14.0` or the owner's name, as Step 9; the smoke
  test also draws a status row and runs a spinner's frames into a buffer;
  its control against `v2.13.0` fails to build.
* `docs/glossary.md`: the reserved names move to "Built from those
  records". The MADR's status and this PLAN's `complete`;
  `docs/README.md`.

## Verification

* Each step's tests pass, and each was seen failing on its planted break,
  the failure quoted (rule 9).
* Every golden was read in full before its commit; every row fits its
  width.
* `TestCoresAreCharmFree` lists `diffview` and `agentui` and passes; it was
  seen failing on a planted Charm import in each.
* `TestTokensAreACP` holds MADR D4's tokens.
* `TestFeedGolden`'s scripted turn draws each 0056-MADR D13 item through
  its MADR D4 type, a frame per `Change`.
* `TestDefaultsInventory` resolves every control MADR D9 names.
* `python3 -m tools.verify gates` passes on macOS and on Linux and Windows,
  at Steps 3 to 8 and 10 to 13, each through the queue of rule 13.
* `make apicheck`: 0 incompatible changes against `v2.12.0` at Steps 2 to
  9, and against `v2.13.0` at Steps 10 to 14.
* `make markdownlint check-records docs-check disclosure-check` passes at
  every step.
* The identifier scan of each step's diff is clean.
* CI is green on the pushed tree and on each tag; each smoke test passes
  from the proxy, and its control fails against the tag before.

## Rollout and Rollback

* One commit a step, by the owner (rule 7), so each step is reverted by its
  own commit.
* Before a tag, a step's commit is reverted with its tests and goldens;
  nothing else depends on it but the steps after it, which are reverted
  first.
* After a tag, every exported name stays (API convention 1); a fault is
  fixed forward in a patch release. A component found wrong in shape is a
  new record and a deprecation, never a removal within `v2`.
* Nothing persists: no file format, no state outside the process. A
  consumer that has not adopted a tag is unaffected by it.

## Execution record

Not started. ~~The owner has not accepted the MADR or approved this
PLAN.~~ *(2026-11-14:)* The owner accepted the MADR and ruled on Q1 to Q5;
this PLAN waits for go-llmprovider-sdk `v1.31.0`, gobble-cli `v0.7.0` and
the owner's approval.
