---
status: accepted
date: 2026-11-14
decision-makers: the repository owner
consulted: "gobble-cli at 512db99, read only: docs/decisions/0056-MADR-curate-the-terminal-session-view-with-view-modes.md (D11 to D16, Q8 to Q10), 0005-MADR-v1-feature-scope.md (F9) and 0032-MADR-go-tui-lib-integration-boundary.md (D1, A1, D3, D5, D7); this repository at 855aa72 (main, one docs commit past v2.12.0); a read-only survey of v2.12.0 against gobble's F9, 2026-11-13, relayed by the coordinator; acp-go-sdk2 v1.36.0's enum tokens, gobble's pin; docs/reports/0003-REPORT-agent-tui-ecosystem-research.md §7, §8.7, §8.10, §8.12, §8.17 and §11; the owner's rulings of 2026-11-14 on Q1 to Q5 and on execution, relayed by the coordinator"
informed: gobble-cli, which asked for these components; pi-go, the library's other consumer
---
# Give agent programs a tool card, an activity row, a diff view, a status row, a plan checklist and a thought display, as Charm-free cores shaped like ACP, with a feed pane and a status pane for the full screen

## Context and Problem Statement

### The request

gobble-cli's owner ruled on 2026-11-13: "Draft a go-tui-lib record now".
gobble-cli records the ruling as
`docs/decisions/0056-MADR-curate-the-terminal-session-view-with-view-modes.md`
Q10, decision (a): the tool card, the activity row, the diff view, the
status row, the thought display and the plan checklist "are requested from
go-tui-lib, each shaped like D13's item types, and F9 waits for their
tags". That record decides nothing about this library's design; this one
does.

gobble's full-screen mode, F9 of gobble-cli
`docs/decisions/0005-MADR-v1-feature-scope.md`, does not copy what it
lacks: "Where a core behaviour is not in the selected tag yet, F9 waits
rather than copying it" (its lines 705 to 712). Its feature table (line
201) lists, among others, a "streamed Markdown transcript", "collapsible
tool cards with diffs" and "a footer with model, mode, context %, and
cost". gobble-cli `docs/decisions/0032-MADR-go-tui-lib-integration-boundary.md`
D3 has gobble's complex terminal elements built here, "each for both a
line-mode CLI … and the TUI", and its D5 makes each a request, not a copy.

### What gobble's layer will hand a renderer

gobble-cli 0056-MADR D11 to D16 put the curation of a session in a
renderer-neutral package, `internal/curate`, planned and not yet built at
`512db99` (its PLAN's P1). It imports only the standard library,
`acpclient` and `jsonx`, so it can never import this library (its D11).
Each input returns `Change{Op, ID, Item}` values, `Op` one of `Open`,
`Grow` and `Close`; an item is closed once and never changes after. Its D13
fixes the item types:

| Item | What it carries |
| :--- | :--- |
| `ToolCard` | one call: ID, ACP kind, title, status, summary, output, `Diff`s, terminal IDs, `Fold` |
| `ActivityRow` | a run of finished calls: count and failed count per kind, each call's title and diff stat, the calls' `ToolCard`s |
| `Diff` | a path, the old and new text, and `+a −d` |
| `StatusRow` (`Curator.Status()`) | thinking or not, the open `ActivityRow` so far, the running call |
| `Thought` | the thought's text, growing |
| `Plan` | the entries, whether they changed, and a `Form`: checklist or summary |
| `Reply` | the reply's text, growing, or a non-text block's name |
| `Note` | a retry or a compaction |
| `TurnEnd` | the last usage, and the turn's result |
| `Raw` | an update's params, or a notification's method and params |

* **`Fold`** is `Hidden`, `Summary`, `Tail` (the summary, the last five
  lines without the shell tool's status line, the count of lines above)
  or `Full`. Every item carries its full detail whatever its fold.
* **`Sep`** asks for a gap before an item.
* **The layer carries no words, glyphs, colour or width.** Each renderer
  phrases items with the personality's keys and lays them out.
* **Q8's ruling, (a):** "the mode sets each item's default fold … in the
  TUI any item can be opened up to the full detail it carries, so the TUI
  keeps a fold state per item."
* **Q9's ruling, (a):** a view mode applies from the next turn; no turn is
  replayed.

### What this repository has (read at `855aa72`)

| Piece | Where | What it gives the widgets |
| :--- | :--- | :--- |
| `stream.Bounded` | `stream/bounded.go:35` | live output kept to 64 KiB, then its first 20 and last 20 lines, each line at most 4 KiB (`:17-20`), the lines between counted; `Render` hard-wraps at a width (`:164`) |
| `stream.Interpret` | `stream/interpret.go:63` | captured terminal output replayed into styled rows |
| `stream.Doc`, `stream.Printer` | `stream/document.go:47`, `stream/printer.go:24` | a streamed reply's Markdown, a frame or a line at a time |
| `streamtea.DocPane` | `stream/streamtea/pane.go:26` | one `Doc` in a workspace pane; it cannot hold anything but that document |
| `stream.Style`, `stream.Markup` | `stream/style.go:13`, `:85` | SGR attributes and the eight foreground colours, the Charm-free cores' colour |
| `glyph.Set` | `glyph/glyph.go:61-70` | `Done`, `Failed`, `Running`, `Pending` (`✓ ✗ ● ○`, ASCII `+ x * -`) and `Heading` (`▸`, `>`); no fold marker and no spinner frames |
| `inline.Region` | `inline/region.go:88-276` | live rows below the scrollback, I/O-free: each method returns the bytes to write |
| `frame.Demand` | `frame/demand.go:12` | `None`, `Slow` (83 ms) and `Fast` ticks for an animation |
| `layout.WithFooter` | `layout/presets.go:87` | a footer slot in a preset |
| `transcript.Transcript` | `transcript` | the line mode's conversation, printed again on a resize |
| `msgcat` | `msgcat/msgcat.go` | the words a component shows, each with an English default a program's catalogue replaces |
| `tuitest.Golden`, `inlinetest.Screen` | `tuitest/tuitest.go:105`, `inline/inlinetest/screen.go:38` | the rendering matrix, and a terminal screen for line forms, `Cooked` included |

**Absent** (the survey; confirmed at `855aa72`): collapsible tool cards,
grouped activity rows, a diff view, a spinner or status-row component, a
plan checklist, a thought display, and a full-screen transcript able to
hold anything other than one Markdown document.
[architecture.md](../architecture.md), "What is not here", lists standard
panes and overlay widgets as records still to write.

### Prior art this repository already holds

[0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§7 names item 6, "a virtualised transcript and standard panes", and item 7,
"ACP-shaped agent widgets", neither with a record. §11.2 names them
`transcript`/`listpane` (6), `agentui` (7) and, among candidate 14,
`diffview`. §11.5 records the owner's answer of 2026-10-02 that records for
`agentui` and for `transcript` are to be written. Its findings that bear
on this record:

* **§8.12, tool-call rows have a grammar:** an icon per tool kind, then the
  title; running, waiting, done and failed as typed states; a status word
  so the coloured mark is not the only signal; output collapsed to N lines
  with an expand control.
* **§8.12, runs of calls fold under one label:** "Read 3 files, Searched 2
  patterns · 1 failed", present tense while a member runs.
* **§8.12, the boundary:** "agent widgets consume a transport-neutral
  session model and emit replies as messages. They never call a backend."
* **§8.10, diffs:** a sign column, never colour alone ("Do not copy grok's
  diffs: they have no sign column"); ANSI-16 gets a foreground only.
* **§8.7, the cell contract:** lines for a width and a mode; display modes
  collapsed, truncated, expanded; disclosure keyed by stable IDs; hidden
  output never rendered; follow the tail.
* **§8.17:** a spinner's width is its widest frame; reduced motion shows a
  static mark.

### The names

Every planned type name was checked with `python3 -m tools.verify names`.
The first draft's `agentui.Status` clashed with `termsvc.Status`
(`termsvc/clipboard.go:26`): "names: 1 clash(es) in 28 planned name(s)".
Renamed `StatusRow`, the 30 planned names of D2, each package's `Option`
among them, are "clean". `Plan` is `layout`'s
([glossary](../glossary.md)), so the plan checklist is `Checklist`;
`Row` is `ask`'s, `Block` and `Writer` are `stream`'s.

### The question

What does go-tui-lib build, in what shape, so that gobble's TUI renderer
draws `internal/curate`'s items one to one, a line-mode CLI can use the
same logic, the library stays free of ACP and of gobble, and F9's main view
is unblocked first?

## Decision Drivers

* **gobble's request, shaped like its items** (0056-MADR Q10 (a), D13),
  with F9 waiting on the tags (0005-MADR F9).
* **One implementation for both modes** (0032-MADR D3;
  [0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)'s
  first driver). The logic lives in Charm-free cores; Bubble Tea forms sit
  in `-tea` sub-packages, as `ask`/`asktea`, `lineedit`/`lineedittea` and
  `stream`/`streamtea` do.
* **ACP-shaped, not ACP-bound** (0003-REPORT §8.12). No module may be
  required without a record naming it (AGENTS.md, "Dependencies"); no ACP
  module is named, and none is proposed.
* **Never import gobble.** The library serves pi-go and any agent program.
* **The library's rules** (AGENTS.md, "TUI conventions" and "API
  conventions"): no I/O, clock, goroutine or environment read in a core;
  every glyph with an ASCII twin; colour never alone; width in cells, never
  exceeded; golden tests across the matrix; one name, one meaning;
  defaults the program can refuse (rule 12).
* **Untrusted text is a security boundary.** Titles, tool output, diffs,
  thoughts and plan steps come from a model or a tool; each is cleaned once,
  on the way in.
* **Per-item fold** (0056-MADR Q8): any item opens up to its full detail,
  and its fold survives redraws, resizes and new items.
* **Order of value:** F9's main view first, which is the tool card and the
  activity row in a scrolling view.
* **Minor releases only:** 0 incompatible changes by `make apicheck`.

## Considered Options

The record settles eight questions. Each has its options; the recommended
one is first.

* **D1, which components.**
  * **D1-A: the six requested, a note, and a minimal feed:** a scrolling
    list of these items with per-item fold, the part of candidate 6 F9's
    main view needs; the rest of candidate 6 later.
  * D1-B: the six only; the feed is candidate 6's own record.
  * D1-C: the six, and all of candidate 6 (search, selection and copy,
    bookmarks, sticky headers, hydration).
* **D2, which packages.**
  * **D2-A: `diffview` (a generic diff core), `agentui` (the agent widgets'
    cores and the feed), and `agentui/agentuitea` (the panes).**
  * D2-B: a package per component, each with its own `-tea` sub-package.
  * D2-C: one `agentui` package, the diff inside it.
* **D3, the shape of a component.**
  * **D3-A: a value core per item: data, setters, and rows for a width
    under a `Look`; the fold is the item's own; two panes for the full
    screen.**
  * D3-B: an I/O-free engine per item (`HandleInput`, `NextEvent`), as
    `ask` and `pick` are.
  * D3-C: Bubble Tea models only.
* **D4, the contract with consumers.**
  * **D4-A: types that mirror gobble's items one to one, and enums whose
    tokens are ACP's, so a consumer maps by text; no ACP import.**
  * D4-B: generic types with no ACP vocabulary; the consumer turns kinds
    and states into labels and glyphs.
  * D4-C: ACP's SDK types as the inputs.
* **D5, what a diff is made from.**
  * **D5-A: lines with their operation, as the core's input, and
    `Compute`, a bounded line diff of two texts, as a helper.**
  * D5-B: lines only.
  * D5-C: the old and new texts only, diffed inside.
* **D6, the words.**
  * **D6-A: `msgcat` messages with English defaults, one message per kind
    and number, replaced by the program's catalogue.**
  * D6-B: a phrasing hook (`Phraser`) the program implements.
  * D6-C: no words; the program supplies every label.
* **D7, the line mode.**
  * **D7-A: every core's rows usable by a line-mode CLI, plus `Spinner`,
    a one-row engine on `inline.Region` that animates the status row.**
  * D7-B: the rows only; a host drives `inline.Region` itself.
  * D7-C: full screen only, with no promise that the cores stay Charm-free.
* **D8, the release.**
  * **D8-A: two minors: `v2.13.0` with the diff view, the tool card, the
    activity row, the note and the feed; `v2.14.0` with the status row,
    the spinner, the plan checklist and the thought.**
  * D8-B: one minor, `v2.13.0`, with everything.
  * D8-C: a minor per component.

## Decision Outcome

*~~Proposed, 2026-11-14. Nothing is built before the owner accepts this
record and approves
[0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md). Five questions
are the owner's, under "Owner questions"; each recommended answer is
written into the decisions below.~~*

*Accepted 2026-11-14 by the owner, who ruled on Q1 to Q5 that day, each
the recommendation, relayed by the coordinator; each ruling is a dated
note under its question. The owner chose "Record now, build later" for
execution: nothing is built before go-llmprovider-sdk `v1.31.0` and
gobble-cli `v0.7.0` are released and the owner approves
[0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md), which stays
`proposed` until then.*

Chosen options: **D1-A, D2-A, D3-A, D4-A, D5-A, D6-A, D7-A and
D8-A**, because together they give gobble's TUI renderer a type for every
item and a view to put them in, give a line-mode CLI the same rows with no
Bubble Tea, keep ACP's vocabulary without its module, and put the main view
in the first tag. D1-B would tag the cards with nothing in this library to
show them in, so F9 would still wait; D2-B multiplies packages whose types
embed one another; D3-B gives display-only widgets an input contract they
do not need; D4-C requires an unnamed module and ties the library to one
protocol's releases; D5-C forces a second diff algorithm on any consumer
that already has one; D6-C pushes plurals and tense into every program;
D7-C breaks 0032-MADR's premise; D8-B delays the main view behind the
status row and the checklist.

The decision has nine parts, D1 to D9.

### D1. The components

| Component | Type | What it draws | Folds |
| :--- | :--- | :--- | :--- |
| Tool card | `agentui.ToolCard` | a head row (fold marker, state glyph, title, an edit's diff stat), then by fold: the summary; the last lines with the count above; or the output, the diffs and the terminal references | `Hidden`, `Summary`, `Tail`, `Full` |
| Activity row | `agentui.ActivityRow` | one row: the fold marker, a label per kind ("ran 3 commands, read 2 files"), the state, the members' titles cut to fit and `+n`; opened, each member's card under it | `Hidden`, `Summary`, `Full` |
| Diff view | `diffview.Diff` | a unified diff with a sign column, hunk headers and an optional number gutter; or its stat, `path +a −d` | (a card's fold decides) |
| Status row | `agentui.StatusRow` | one row: a spinner frame, the label (thinking, the open activity and the running call, or idle), the elapsed time, and the program's facts on the right | none |
| Plan checklist | `agentui.Checklist` | the steps with their state glyphs; or one row, `plan 2/5 · <the step in progress>` | `Hidden`, `Summary`, `Full` |
| Thought | `agentui.Thought` | one row, a label and the thought's first line cut; or the label and the text, dimmed and wrapped | `Hidden`, `Summary`, `Full` |
| Note | `agentui.Note` | a notice or a raw line, hard-wrapped, with a tone; a warning or a failure starts with a word, so colour is not alone | none |
| Feed | `agentui.Feed` | the session's items in order, scrolled by an anchor, following the tail, with a focused item and per-item fold | — |

* **The feed is in, at its minimum** (Q1). It holds `Element`s: tool
  cards, activity rows, checklists, thoughts and notes, and a stream `Doc`
  through `agentui.FromDoc`. A diff is drawn inside its card; the status
  row is drawn by the host, outside the feed.
  It keeps each element's height by version, width and look; scrolls by a
  content anchor (an element's ID and a row inside it), so a resize or a
  fold keeps the reader's place; follows the tail until scrolled up; and
  draws only what the window shows. It is bounded by an element count,
  the oldest dropped first behind a note.
* **Out of this record, left to candidate 6's own:** search, selection and
  copy, bookmarks, sticky headers, hydration of a refreshed session, and a
  generic list pane. The existing `transcript` package keeps its name and
  its line-mode meaning; the feed does not take it.
* **Hidden is reachable.** A hidden card shows when its activity row is
  opened. A hidden element of its own, such as gobble's default-mode
  `Thought`, draws nothing until the feed's reveal switch, off by default,
  draws every hidden element at `Summary`.
* **A group of one draws as its card's head row,** as gobble's line view
  draws it (0056-MADR D3).
* **A card's `Full`** draws the output when there is any and the summary
  only when there is none, as 0056-MADR D4 has it, since an agent writes the
  summary from the output; then each diff, unified; then each terminal
  reference. The output is the card's `stream.Bounded`, so `Full` is the
  full detail up to the bound the program sets.
* **A card's `Tail`** draws the summary, then the tail the program gave
  (`SetTail`), or else the output's last lines, then the count of lines
  above. gobble's layer, which owns the shell status-line recogniser
  (0056-MADR D11), gives its own tail.

### D2. The packages

| Package | Holds | Imports | Kind |
| :--- | :--- | :--- | :--- |
| `diffview` | `Diff`, `Hunk`, `Line`, `Op`, `Stat`, `Colors`; `Compute`; the unified and stat forms | `glyph`, `msgcat`, `stream` (for `Style`), `internal/enum`, `internal/sanitize`; `x/ansi` | Charm-free core |
| `agentui` | `ToolKind`, `ToolState`, `Fold`, `ToolCard`, `ActivityRow`, `StatusRow`, `Spinner`, `Checklist`, `Step`, `StepState`, `Thought`, `Note`, `Tone`, `Element`, `Foldable`, `Feed`, `Look`, `Accents`; `FromDoc` | `diffview`, `stream`, `inline`, `frame`, `glyph`, `msgcat`, `safetext`, `internal/enum`, `internal/sanitize`; `x/ansi` | Charm-free core |
| `agentui/agentuitea` | `FeedPane`, `StatusPane`, `StatusMsg` | `agentui`, `stream`, `workspace`, `keymap`, `keymap/keymaptea`, `command`, `frame`, `frame/frametea`, `theme`, `glyph`; Bubble Tea | `-tea` form |

* **`diffview` stands alone,** candidate 14's diff without an agent in
  it, so a program that shows a diff links no agent widget.
* **The cores join `cores`** in `internal/conformance`'s
  `TestCoresAreCharmFree` and depguard's `charmfree` rule, each in the
  commit that builds it
  ([0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  D11).
* **No module is added.** Every import above is in the root module's list.
* **`glyph.Set` gains,** each with an ASCII twin and one cell wide in both:
  a folded and an unfolded marker (`▸`/`▾`, `>`/`v`); the diff stat's
  minus (`−`, `-`); a separator for the members' titles (`·`, `-`); and
  spinner frames as a fixed-length array, so that `Set` stays comparable
  and `apicheck` sees an addition, not a change. A diff's sign column stays
  `+`, `-` and space in both sets: it is the unified format, and a copied
  hunk stays a patch. The legacy tier falls back to ASCII where it has no
  form (0003-REPORT §11.5).
* **The 30 names** pass `python3 -m tools.verify names`; the glossary's
  "Reserved by accepted records" gains them when this record is accepted.

### D3. The shape of a component

* **A core is a value with an ID.** `NewToolCard(id, kind, title)`,
  `NewActivityRow(id)`, `NewChecklist(id)`, `NewThought(id)`,
  `NewNote(id, text, tone)`: data in, changed by setters (`SetState`,
  `SetSummary`, `Write`, `SetTail`, `AddDiff`, `AddTerminal`, `Add`,
  `Append`, `SetSteps`, `Close`), drawn by `Rows(width int, look Look)
  []string`. Each row is at most `width` cells under the look's method,
  cut with the look's `Ellipsis`; text is wrapped only where D1 says so.
* **`Look`** is what a host may change while it runs: the glyph set, the
  width method, the `Accents` (a `stream.Style` per role: title, muted,
  success, failure, running, label, focus, and `diffview.Colors` for added,
  removed, hunk and gutter), the `msgcat.Catalog`, and a `Revision` the
  host bumps when any of them changes, as `Doc.SetRevision` takes the
  theme's ID. Zero `Accents` draw no colour: a host with `NO_COLOR` or an
  ASCII profile passes the zero value, and every state is still a glyph or
  a word.
* **The fold is the item's own.** `Foldable` is `Fold() Fold`,
  `SetFold(Fold)` and `Toggle()`. An item starts at the fold it was given
  (gobble: the mode's); `Toggle` opens it to `Full`, and closes it back to
  where it started, or to `Summary` when it started at `Full`.
* **No I/O, no clock, no goroutine.** Time comes in: `StatusRow.Row(width,
  now, look)` derives the frame and the elapsed time from `now` and the
  row's start. One goroutine drives an item, as one drives `inline.Region`.
* **Untrusted text is cleaned once, on the way in:** titles, step text and
  terminal IDs by `internal/sanitize.Line`; tool output by its
  `stream.Bounded`, whose floor is `safetext.StatusLine`; diff text and
  thoughts by `safetext`'s `Line` and `Text` rules. Cleaning is a floor,
  listed in [guides/defaults.md](../guides/defaults.md), not an option.
* **`Element`** is what the feed holds: `ID() string`, `Version() uint64`,
  `Rows(width int, look Look) []string`. The card, the activity row, the
  checklist, the thought and the note implement it, and `FromDoc(id,
  *stream.Doc)` adapts a reply.
* **Two panes for the full screen.**
  * `FeedPane`, from `NewFeedPane(ws, feed)`, is the feed in a workspace
    pane: drawn at the pane's width with the workspace's width method and
    glyphs; the look's revision from the theme's ID; scrolled, focused and
    toggled by commands in the program's keymap (`DefaultRules`, the command
    IDs, `FeedPaneScope`), and by a click on a head row; the focused item
    marked by `glyph.Focus`, not colour alone. It animates nothing.
  * `StatusPane`, from `NewStatusPane(ws)`, draws a `StatusRow` given by
    `StatusMsg`, ticking at the row's `frame.Demand` through
    `frametea.TickCmd`, in a footer slot such as `layout.WithFooter`'s.
  * Neither picks colours from the theme's Lip Gloss palette: each takes
    `Accents` from the program, as `DocPane` takes its `Doc`'s renderer, and
    draws none when the workspace's profile has no colour.
* **No pane per card, row, checklist or thought.** Each is an element of a
  feed; a checklist in a sidebar is a `FeedPane` over a feed of one.

### D4. The contract with consumers

* **One type per item.** gobble's TUI renderer maps each `Change` to one
  call, and needs no logic of its own:

| `internal/curate` item (0056-MADR D13) | go-tui-lib | `Open` | `Grow` | `Close` |
| :--- | :--- | :--- | :--- | :--- |
| `ToolCard` | `agentui.ToolCard` | `NewToolCard`, then its fields | `SetState`, `Write`, `AddDiff`, `AddTerminal`, `SetTail` | `Close` |
| `ActivityRow` | `agentui.ActivityRow`, its members `ToolCard`s | `NewActivityRow` | `Add(card)` | `Close` (past tense) |
| `Diff` | `diffview.Diff` | built from the layer's line diff (D5) | — | — |
| `StatusRow` | `agentui.StatusRow`, its open activity the feed's live `ActivityRow` | `StatusMsg` to the `StatusPane`, or `Spinner.Set` in the line mode | as `Open` | — |
| `Thought` | `agentui.Thought` | `NewThought` | `Append` | `Close` |
| `Plan` | `agentui.Checklist` | `NewChecklist`, `SetSteps`; `Form` checklist or summary as fold `Full` or `Summary` | `SetSteps` | — |
| `Reply` | `stream.Doc` through `agentui.FromDoc` (exists, `v2.12.0`) | `NewDoc` | `Append` | `Finish` |
| `Note` | `agentui.Note` | `NewNote`, tone by cause | — | — |
| `TurnEnd` | the `StatusRow`'s facts (model, mode, context, cost), phrased by the program | — | — | — |
| `Raw` | `agentui.Note`, plain tone, one line, hard-wrapped | `NewNote` | — | — |
| `Sep` | `Feed.Gap()` before the item's `Append` | — | — | — |
| `Fold` | `agentui.Fold`, by token | `SetFold` | — | — |

* **Enums share their tokens with ACP and with gobble's layer,** through
  `internal/enum`, so a consumer maps by `UnmarshalText` of the token it
  already holds:
  * `ToolKind`: `read`, `edit`, `delete`, `move`, `search`, `execute`,
    `think`, `fetch`, `switch_mode`, `other` (ACP's `ToolKind`);
  * `ToolState`: `pending`, `in_progress`, `completed`, `failed` (ACP's
    `ToolCallStatus`);
  * `StepState`: `pending`, `in_progress`, `completed` (ACP's
    `PlanEntryStatus`);
  * `Fold`: `hidden`, `summary`, `tail`, `full` (0056-MADR D13);
  * `Tone`: `plain`, `muted`, `warning`, `failure`;
  * `diffview.Op`: `context`, `insert`, `delete`.

  The tokens were read from acp-go-sdk2 `v1.36.0`, gobble's pin. A token a
  later ACP adds that this library lacks is the consumer's to map to
  `other` (a kind) or to refuse; `UnmarshalText` returns an error, never a
  guess.
* **No ACP module, no gobble import.** The library holds the tokens only
  as text. ACP's plan priority, a call's raw input and its locations are
  not drawn; a consumer orders steps and titles calls itself.
* **What is not one to one, said once here:**
  * `ActivityRow`'s per-kind counts and failed count are derived from its
    member cards, not given; a consumer that adds the layer's cards gets
    the layer's counts.
  * gobble's `Diff` carries the old and new text; `diffview.Diff` carries
    the lines. gobble maps the layer's line diff to `diffview.Line`s, so
    the stat drawn equals the layer's `+a −d` (D5).
  * `Plan`'s "changed" is the consumer's: it appends a new checklist or
    sets the steps of the one it has.
  * `TurnEnd` is no element: its words are the program's facts on the
    status row.

### D5. A diff's input

* **The core's input is lines:** `Diff{Path, Hunks}`, each `Hunk` its old
  and new start lines and its `Line`s, each `Line` an `Op`, its old and new
  line numbers and its text. `Stat` is `{Added, Removed}`, counted from the
  lines, so it cannot disagree with what is drawn.
* **`Compute(path, old, new string, opts)`** is a line diff of two texts,
  for a consumer that has only those, as an ACP `diff` content block gives
  them: Myers' O(ND) algorithm on the standard library, three lines of
  context by default (`WithContext`), and a bound (`WithComputeLimit`)
  past which the diff is one hunk that removes every old line and inserts
  every new one, which is true, never a guess. gobble does not need it; its
  layer has a line diff of its own (0056-MADR D11).
* **Drawing.** The unified form: a header row with the path; each hunk's
  `@@ -a,b +c,d @@` row; each line with its sign column and, when
  `WithGutter` asks, right-aligned old and new numbers; the text cleaned
  and cut to the width, never wrapped, so a row is one source line. Added
  and removed lines take a foreground colour only (0003-REPORT §8.10:
  ANSI-16 has no faithful tinted background). The stat form: `path +a −d`,
  one row, the path cut from its left so the file name stays.

### D6. The words

* **Every word a widget shows is a `msgcat` message** with an English
  default and a `Messages()` table, as `ask`, `pick`, `stream.Bounded` and
  `transcript` do: the activity row's label per kind in two numbers (one,
  many) and two tenses (running, done), "%d failed", `+%d`, "thinking",
  "thought", "plan %d/%d", "%d lines above", "terminal %s", the note
  labels "warning" and "error", the feed's "earlier items dropped".
* **The program's catalogue replaces them** through the look's
  `Catalog`; `msgcat.Check` holds a replacement's verbs to the defaults'.
  gobble's TUI gives a catalogue that maps these IDs to its personality's
  keys (0056-MADR D3, D13), so the line view and the TUI use one set of
  words.
* **A group of one has no label:** it draws its card's title (D1).

### D7. The line mode

* **Every core's rows serve a line-mode CLI** with no Bubble Tea: a closed
  `ActivityRow`'s one row is a group line; a `ToolCard`'s rows at its fold
  are a verbose call block or a failure's detail; `diffview`'s stat row is
  an edit's summary; a `Checklist` at `Summary` is a plan line. The host
  writes them, commits them through a region, or gives them to
  `transcript`'s `AddDrawn` so a resize reprints them.
* **`Spinner`** is the status row's line form: an engine on one
  `inline.Region` row, under
  [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)
  D2's contract. `Set(row, look, now)`, `Resize(width, now)`, `Timeout()`,
  `HandleTimeout(now)`, `Print(text)` (the text above, the row drawn again
  below), `Commit(rows)` (the row cleared and the rows left in the
  scrollback, the collapse gobble's stage 2 draws), `Clear()` and
  `Restore()` each return the bytes to write. Its region is the host's,
  by `WithRegion`, so the host chooses `WithoutCursorQuery` and the reflow
  kind; `WithoutMotion` draws the `Running` glyph in place of the frames.
  Whether a terminal is present is the host's question.
* **The feed has no line form.** The terminal's scrollback is the line
  mode's feed, and `transcript` its reprint.
* **gobble's use is gobble's decision.** gobble-cli 0032-MADR D1 and A1
  already admit into `internal/cli` a go-tui-lib package whose closure
  holds no Charm module but `x/ansi`, and its D7 makes each adoption its
  own plan step. Drawing gobble's group line or diff stat through these
  cores would also change gobble-cli 0056-MADR D9 and D16, which keep the
  group line's layout and the status row's drawing in gobble's line
  renderer, and `internal/curate` may never import this library (its D11).
  This record promises only that `diffview` and `agentui` stay Charm-free,
  by the gate.

### D8. The release

* **`v2.13.0`** (Q3): the glyph additions, `diffview`, `ToolCard`,
  `ActivityRow`, `Note`, `Feed`, `FromDoc` and `FeedPane`: everything F9's
  main view needs to show a turn's actions.
* **`v2.14.0`:** `StatusRow`, `Spinner` and `StatusPane`; `Checklist`;
  `Thought`.
* **Each is a minor:** `make apicheck` reports 0 incompatible changes
  against the tag before it. Tags are the owner's.
* **The version numbers assume `v2.12.0` is the base.** A release made in
  between moves both to the next free minors, as
  [0025-PLAN-charm-free-stream-allocations.md](0025-PLAN-charm-free-stream-allocations.md)'s
  prerequisites say.

### D9. Defaults, floors and tests

* **New defaults, each with its control** (AGENTS.md, API rule 12), listed
  in [guides/defaults.md](../guides/defaults.md) by the PLAN:
  * a card's tail lines, 5, `WithTailLines`; its output bound,
    `stream.Bounded`'s defaults, `WithOutput(stream options…)`;
  * a diff's context, 3, `WithContext`; `Compute`'s bound,
    `WithComputeLimit`; the number gutter, off, `WithGutter`;
  * the spinner's frame interval, `frame`'s `Slow` (83 ms), `WithInterval`;
    its motion, `WithoutMotion`;
  * the feed's element bound, set by the PLAN from a measured heap budget,
    `WithMaxElements`; its reveal switch, off, `SetReveal`; following the
    tail, `WithoutFollow`;
  * the panes' keys, `DefaultRules` and the command IDs, which the
    program's keymap may drop or rebind;
  * every word, by the catalogue (D6).
* **Floors:** the cleaning of D3, and the bounds on untrusted output.
* **Tests:** goldens by `tuitest.Golden` across {colour, no colour} ×
  {UTF-8, ASCII} × at least three widths for every core and fold; the
  `Spinner` on `inlinetest.Screen`, raw and `Cooked`; a fuzz target for
  `Compute` and for hostile output, titles and diff text; enum round trips;
  budgets in bytes and allocations, never time; each test seen failing on
  a planted break in a scratch copy, the failure quoted.

### Consequences

* Good, because F9's main view, a scrolling session of folded tool cards
  and activity rows with their diffs, can be drawn from one tag,
  `v2.13.0`, with no gobble copy.
* Good, because gobble's TUI renderer maps each `internal/curate` item to
  one go-tui-lib type and each `Op` to one call, with enums matched by
  token.
* Good, because a line-mode CLI can draw the same rows, and a `Spinner` on
  the region it already uses, without Bubble Tea.
* Good, because `diffview` serves any program that shows a diff.
* Good, because the library holds ACP's tokens without its module, so an
  ACP release changes nothing here unless it adds a token.
* Neutral, because a token ACP adds later is the consumer's to map until
  this library adds it in a minor.
* Neutral, because the feed covers only the part of candidate 6 F9's main
  view needs; search, copy and bookmarks wait for candidate 6's record.
* Bad, because the library takes on two packages, about thirty exported
  types, a glyph-set extension and a new kind of pane to keep stable under
  the deprecation policy.
* Bad, because `Compute` is a second diff algorithm in the fleet beside
  gobble's layer, though gobble need not call it (Q2).
* Bad, because `Full` shows output only up to the card's bound; a program
  that wants more raises it, and pays the memory.
* Bad, because F9 still waits for `v2.14.0` for its footer, checklist and
  thought, and for its own adoption steps after each tag.

### Confirmation

1. `internal/conformance`'s `TestCoresAreCharmFree` lists `diffview` and
   `agentui`, and fails on a Charm module planted in either, in a scratch
   copy, before the commit that adds them.
2. Every core and fold has goldens across the matrix, each seen failing on
   a planted break; every row of every golden fits its width by
   `tuitest.Fits`.
3. The enum tokens round-trip, and equal the ACP tokens named in D4 by a
   table test; a token removed from the table fails it.
4. A replay test feeds a scripted turn, built as gobble's D13 items would
   arrive, through the D4 mapping into a feed, and its frames match goldens
   at each `Op`; a fold toggled before a resize is kept after it.
5. The `Spinner`'s frames on `inlinetest.Screen` are one row at every
   width, each drawing one synchronized update; `Print` leaves the text
   above and the row below; `Commit` leaves the rows and none of the
   row's states in the scrollback.
6. `make apicheck` reports 0 incompatible changes for each minor.
7. The defaults inventory resolves every control D9 names.
8. The gates pass on macOS, Linux and Windows; each tag's smoke test passes
   from the proxy, and its closure for the three systems holds no Charm
   module but `x/ansi` for the two cores.

## Pros and Cons of the Options

### D1-A: the six, a note, and a minimal feed

* Good, because the first tag holds both the items and a place to show
  them, so F9's main view stops waiting.
* Good, because the feed takes the cell contract, anchors and disclosure
  from 0003-REPORT §8.7 without its search and copy.
* Bad, because the feed is a commitment candidate 6's record must build on,
  not replace.

### D1-B: the six only

* Good, because this record stays smaller, and candidate 6 is designed
  whole.
* Bad, because F9 cannot show a card until a second record and tag land;
  `DocPane` holds one Markdown document and nothing else.

### D1-C: the six and all of candidate 6

* Good, because the transcript is designed once.
* Bad, because search, copy, bookmarks and hydration are the largest part of
  candidate 6, and none blocks F9's main view.

### D2-A: `diffview`, `agentui`, `agentui/agentuitea`

* Good, because the widgets embed one another (a row holds cards, a card
  holds diffs), so one package keeps them together, while the diff, which
  embeds nothing, stands alone.
* Good, because one `-tea` sub-package holds the two panes, as
  `stream/streamtea` holds `DocPane`.
* Bad, because a program that wants only a checklist links the card's code
  too.

### D2-B: a package per component

* Good, because each program links exactly what it uses.
* Bad, because seven cores and up to seven `-tea` packages import one
  another in a chain, each with its own options, stability line and docs.

### D2-C: one `agentui`, the diff inside

* Good, because there is one import.
* Bad, because a program that shows only a diff links every agent widget.

### D3-A: value cores drawn under a `Look`

* Good, because a display-only widget needs data in and rows out; the feed
  and the panes do the input.
* Good, because a theme change is one new `Look`, not a rebuilt item.
* Neutral, because the fold lives on the item, so the consumer keeps the
  items it made.

### D3-B: an engine per item

* Good, because it matches `ask` and `pick`.
* Bad, because a card takes no keys of its own; an input contract per item
  is code with no caller.

### D3-C: Bubble Tea models only

* Bad, because a line-mode CLI could not use them, against 0032-MADR D3
  and 0020-MADR.

### D4-A: mirror the items, share ACP's tokens

* Good, because gobble's renderer maps by type and by token, with no table
  of its own.
* Good, because any ACP client, not only gobble, finds its vocabulary.
* Bad, because ACP's tokens fix this library's enum names; a renamed token
  upstream is a deprecation here.

### D4-B: generic types, no ACP vocabulary

* Good, because the library owes ACP nothing.
* Bad, because each consumer writes the same kind-to-label and
  state-to-glyph tables, and the widgets cannot phrase "ran 3 commands".

### D4-C: ACP's SDK types

* Bad, because it requires a module no record names, and ties every tag to
  the SDK's releases.

### D5-A: lines in, `Compute` as a helper

* Good, because a consumer with its own diff, gobble, draws exactly the
  lines it counted.
* Good, because a consumer with two texts, as ACP's `diff` content gives
  them, needs nothing else.
* Bad, because the library maintains a diff algorithm and its bound.

### D5-B: lines only

* Good, because there is no algorithm to maintain.
* Bad, because every ACP client without a diff of its own must find one.

### D5-C: texts only

* Bad, because gobble's stat and the drawn diff could disagree, two
  algorithms over one edit.

### D6-A: `msgcat`

* Good, because it is how every component here shows words, and
  `msgcat.Check` holds a catalogue's verbs.
* Bad, because one/many and running/done per kind is about twenty messages.

### D6-B: a `Phraser` hook

* Good, because a program writes its own grammar.
* Bad, because it is a second mechanism beside `msgcat` for the same need.

### D6-C: no words

* Bad, because every program rebuilds the plurals and tenses, and the rows
  are not usable as they come.

### D7-A: rows and `Spinner`

* Good, because a line-mode CLI gets the status row on the region it
  already uses, with the yields `inline.Region` has.
* Bad, because gobble's stage 2 (0056-MADR D10) drives its region itself
  and may not adopt it; the engine's first user may be pi-go.

### D7-B: rows only

* Good, because the library builds less.
* Bad, because each host writes the frame timing and the yield again.

### D7-C: full screen only

* Bad, because it breaks the premise of 0032-MADR and of the cores.

### D8-A: two minors

* Good, because the main view ships first, on the components that unblock
  it.
* Bad, because there are two releases, and two adoption steps in gobble.

### D8-B: one minor

* Good, because there is one release and one adoption.
* Bad, because the main view waits for the status row, the checklist and
  the thought.

### D8-C: a minor per component

* Good, because each lands as soon as it is ready.
* Bad, because the card is no use to F9 without the feed, and six releases
  cost six smoke tests and six gobble pins.

## Owner questions

Each has three options, the recommended first, with its cost. The
recommendations are written into D1 to D8; an answer other than the
recommendation amends the decision it names before the PLAN runs.

*(2026-11-14: the owner ruled on all five that day, relayed by the
coordinator, each the recommendation. Each ruling follows its options,
which stand as written; D1 to D8 are unchanged.)*

**Q1. Is the full-screen feed in this record? (D1)**

* **(a) In, at its minimum:** ordered elements, per-item fold, an anchor,
  follow-tail, a focus, a bound. Cost: a pane and a core to keep stable,
  which candidate 6's record later extends rather than replaces.
* (b) Out: candidate 6 gets its own record first. Cost: F9 waits for two
  records and two tags before it shows a card.
* (c) In, whole: search, copy, bookmarks, sticky headers and hydration too.
  Cost: the largest release here, and none of it unblocks F9.

*Ruling, 2026-11-14: "In, at a minimum". **Decision: (a)** (D1): the
minimal `Feed` and `FeedPane` are in; search, copy, bookmarks, sticky
headers and hydration are candidate 6's own record.*

**Q2. What is a diff made from? (D5)**

* **(a) Lines in, and `Compute` for two texts.** Cost: a bounded Myers diff
  to maintain, which gobble does not call.
* (b) Lines in only. Cost: every ACP client without a diff of its own
  brings one.
* (c) Texts in only. Cost: gobble's stat and the drawn diff come from two
  algorithms and may disagree.

*Ruling, 2026-11-14: "Diff lines, plus a Compute helper". **Decision: (a)**
(D5): a caller passes lines; a bounded Myers `Compute` is provided
besides.*

**Q3. How are the components released? (D8)**

* **(a) `v2.13.0` with the diff, card, row, note and feed; `v2.14.0` with
  the status row, spinner, checklist and thought.** Cost: two releases,
  two gobble adoption steps.
* (b) One `v2.13.0`. Cost: the main view waits for everything.
* (c) A minor per component. Cost: six releases, six smoke tests, six
  pins in gobble.

*Ruling, 2026-11-14: "Two minors". **Decision: (a)** (D8): `v2.13.0` holds
the PLAN's Steps 2 to 9 and `v2.14.0` its Steps 10 to 14; each tag is
still the owner's.*

**Q4. Does the line mode get a `Spinner` engine? (D7)**

* **(a) Yes, on one `inline.Region` row, with `Print` and `Commit` yields.**
  Cost: an engine gobble's stage 2 may not adopt, since gobble-cli
  0056-MADR D10 drives its region itself.
* (b) No; the status row's string only, a host animating it. Cost: every
  line-mode host writes the timing and the yields.
* (c) No line form for the status row at all. Cost: the brief's line-mode
  status row is not met.

*Ruling, 2026-11-14: "Yes, a full engine". **Decision: (a)** (D7):
`agentui.Spinner` drives one `inline.Region` row; gobble-cli 0056-MADR's
stage 2 may keep its own row.*

**Q5. Where do the words come from? (D6)**

* **(a) `msgcat`, with English defaults, one/many and running/done per
  kind.** Cost: about twenty messages to keep, and a catalogue gobble maps
  to its personality keys.
* (b) A `Phraser` hook. Cost: a second mechanism beside `msgcat`.
* (c) None; the program supplies every label. Cost: each program rebuilds
  plurals and tenses.

*Ruling, 2026-11-14: "Built-in message catalogue". **Decision: (a)** (D6):
`msgcat` messages with English defaults, one/many and running/done per
kind, which a program's catalogue overrides.*

## More Information

### Not in scope

* Candidate 6's search, selection, copy, bookmarks, sticky headers,
  hydration and generic list pane.
* The permission prompt and dialog, which are `ask` and `asktea`; questions
  and forms; URL consent; the attention policy; turn outcomes; usage bars;
  dashboards (0003-REPORT §8.12's other widgets).
* Interpreting a tool's escape codes inside a card (`stream.Interpret` is
  there for a later option); syntax highlighting of diffs; a split
  (side-by-side) diff; parsing a unified patch's text; stitching successive
  edits to one file.
* A typed "denied" state for a call (0003-REPORT §8.12): ACP's
  `ToolCallStatus` has four tokens and gobble's layer no fifth, so a later
  minor adds one when a consumer sends it.
* ACP's plan priority, a call's raw input and its locations.
* Reduced motion read from the system: the program decides, and passes
  `WithoutMotion`.
* Anything in gobble: its renderer, its adoption of each tag, and the
  amendments its records would need.

### Related records

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md):
  items 6, 7 and 14; §8.7, §8.10, §8.12, §8.17; §11.5.
* [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md):
  `stream.Bounded`, `Doc` and `DocPane`, which the card and the feed build
  on.
* [0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md):
  the cores, the closure gate and the state glyphs.
* [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md):
  the engine contract and `inline.Region`, which `Spinner` follows.
* [0024-MADR-opt-in-and-opt-out-coverage.md](0024-MADR-opt-in-and-opt-out-coverage.md):
  rule 12 and `msgcat`.
* gobble-cli `docs/decisions/0056-MADR-curate-the-terminal-session-view-with-view-modes.md`,
  `docs/decisions/0005-MADR-v1-feature-scope.md` and
  `docs/decisions/0032-MADR-go-tui-lib-integration-boundary.md`: the
  request, the wait and the boundary.
* The plan: [0026-PLAN-agent-widgets.md](0026-PLAN-agent-widgets.md).
