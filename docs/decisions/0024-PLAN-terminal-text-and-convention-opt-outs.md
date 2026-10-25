---
status: proposed
date: 2026-10-25
associated-madr: "0024-MADR-opt-in-and-opt-out-coverage.md"
---
# Close the terminal, text and convention gaps, and adopt the message catalogue

Associated MADR: [0024-MADR-opt-in-and-opt-out-coverage.md](0024-MADR-opt-in-and-opt-out-coverage.md)

The second of the MADR's two PLANs: D5, D6 and D7, and D9's catalogue in
the packages that show text. It runs after
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md),
by the MADR's answer to Q5.

## Goal

A program can drop any one of the probe's queries, force the gated ones
off, override one capability view without changing the terminal's brand,
set `termsvc`'s cell limits, copy route and link schemes, keep `Auto`
from ringing the bell, give every string the existing packages show in its
own words through one catalogue, keep the workspace's focus click while
dropping its drag, open a bare overlay, and select in the palette with a
click without accepting; every hook has its `…Func` adapter; and
`termcap`'s off switch has its rule's name.

Done means every item under Verification holds, CI is green on the pushed
tree, the release is tagged by the owner, and its smoke test passes.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `termcap/` | `WithoutQuery`, `WithoutGatedQueries`, `WithoutQueries` (and `WithDisabled` deprecated); a `Name` for every built-in query; `Env.LookupEnv` folding by the probe's GOOS |
| 2 | `termcap/`, `termsvc/` | the view facts in `Caps`; `WithTitleCells`, `WithBodyCells`, `WithCopyRoute`, `WithLinkSchemes`, `WithoutBell`; `MaxCopyBytes` on the OSC 52 route only |
| 3 | `palette/`, `workspace/`, `command/`, `termcap/` | each package's message IDs, `Messages()` and `WithCatalog` |
| 4 | `workspace/`, `palette/` | `WithoutMouseFocus`, `WithoutMouseResize`, `Overlay.Bare`, `palette.WithoutClickAccept` |
| 5 | `launch/`, `when/` | `RestorerFunc`, `ContextFunc` |
| 6 | `docs/`, `README.md` | the guides, the inventory's rows, the release, the close-out |

### Out of scope

* Retyping the bare-function hooks of MADR D7, which needs `v3`.
* Plural rules and locale formats (MADR D9).
* `git push` and tags, which are the owner's.

## Prerequisites

* [0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
  is `complete`, or the owner moves this PLAN ahead of its remaining
  steps.
* The owner has approved this PLAN.

## Rules for every step

0024-PLAN-command-and-control-opt-outs's rules 1 to 8. Each new option's
row in `docs/guides/defaults.md` moves from `planned` to `built` in its
step.

## Implementation Steps

### Step 1: the probe's queries

* Every built-in query's `Name` is a stable token, checked or set here (`kitty-keyboard`,
  `color-scheme-mode`, `dsr-996`, `background`, `da2`, `xtversion`,
  `notify-query`, `kitty-graphics`, `foreground`, `palette`, and the
  DECRQM modes as `mode-<n>`); a test holds the names to a table.
* `WithoutQuery(names ...string)` drops the named queries; an unknown name
  is an error from the first `Start`, through a `ProbeDone` reason,
  `ReasonUnknownQuery`. DA1 has no name and cannot be dropped.
* `WithoutGatedQueries()` keeps the gated queries off where the heuristic
  would send them.
* `WithoutQueries()` is today's `WithDisabled`; `WithDisabled` is marked
  `// Deprecated: use WithoutQueries`.
* `Env.LookupEnv` folds case by the probe's GOOS: `WithGOOS`'s when set.
* **Tests:** the batch unchanged without the options, byte for byte; each
  name drops its query alone; gated queries off under
  `WithoutGatedQueries` with an SSH environment that would send them;
  `WithDisabled` and `WithoutQueries` give the same probe; a Windows GOOS
  on a Linux host folds case.
* **Mutations:** `WithoutQuery` drops nothing; it drops DA1;
  `WithoutGatedQueries` ignored; the fold reads `runtime.GOOS`.

### Step 2: the views and `termsvc`'s limits

* `Caps` gains `LinksSupport`, `NotifySupport`, `KeyboardRequest`,
  `PointerSupport` and `ProgressSupport`, each a `Fact` its view reads
  first, set through `WithOverride`; unset, each view is today's.
* `termsvc`: `WithTitleCells(n)` and `WithBodyCells(n)`; `MaxCopyBytes`
  applied to the OSC 52 route only; `WithCopyRoute(Route)`;
  `WithLinkSchemes(schemes ...string)`; `WithoutBell()`.
* **Tests:** each view unchanged with no fact set, and changed by its fact
  alone; each limit and route default unchanged, and changed by its
  option; a program's clipboard backend copies past `MaxCopyBytes`.
* **Mutations:** a view ignores its fact; `WithoutBell` ignored; the copy
  bound applied to the backend.

### Step 3: the catalogue in the existing packages

* `palette`, `workspace`, `command` and `termcap` each gain an exported
  `MessageID` constant per string they show, `Messages() msgcat.Table`
  with the English defaults, and `WithCatalog(msgcat.Catalog)`: on the
  picker, the workspace, the registry (for the built-ins' output and the
  MCP error text), and the report (`WithReportCatalog`, beside
  `WithReportWidth`).
* Each string's call site reads `msgcat.Format(c, id, default, args...)`.
* **Tests:** without a catalogue, every golden unchanged; a `Table` that
  rewords one message changes that message alone; each package's
  `Messages()` holds every `MessageID` constant it exports, both ways;
  `msgcat.Check(Messages(), Messages())` passes.
* **Mutations:** a call site ignores the catalogue; `Messages()` misses an
  ID; an argument index swapped in a default.

### Step 4: the mouse and the overlay

* `workspace.WithoutMouseFocus()`, `WithoutMouseResize()`: each turns off
  one behaviour, the wheel and the panes' own mouse events kept.
* `Overlay.Bare`: the zero value is today's chrome; a bare overlay draws
  its pane alone, at its size.
* `palette.WithoutClickAccept()`: a click selects the row.
* **Tests:** each default unchanged; each option drops its one behaviour
  alone; a bare overlay's golden.
* **Mutations:** `WithoutMouseFocus` also drops the drag; `Bare` ignored;
  a click still accepts.

### Step 5: the adapters

* `launch.RestorerFunc` and `when.ContextFunc`, each with its test.
* **Mutations:** each adapter ignores its function.

### Step 6: documentation, the release, and the close-out

* The guides for each addition; `docs/guides/defaults.md` with every row
  `built` or `floor`; `docs/guides/migrating.md`'s section, naming
  `WithDisabled`'s deprecation and the `LookupEnv` correction.
* **The release,** as 0024-PLAN-command-and-control-opt-outs's Step 8,
  the next minor version, its smoke test a program that drops one query,
  rewords one palette message and opens a bare overlay.
* This PLAN becomes `complete`; with the first PLAN complete, the MADR's
  every part is built, which its More Information records.

## Verification

* Every step's mutations are killed, each failure quoted in the record.
* `python3 -m tools.verify gates` passes on macOS, Linux and Windows for
  every step.
* `make apicheck` reports 0 incompatible changes; `WithDisabled` stays.
* `TestDefaultsInventory` passes with no `planned` row left from this
  MADR.
* Without any new option, every golden and test passes unchanged.
* The identifier scan finds nothing.

## Rollout and Rollback

* **Rollout.** One release after Step 6.
* **Rollback.** Before the push, each step is one local commit. After it,
  a patch release fixes forward. The one correction that changes
  behaviour, `LookupEnv`'s fold, touches only a probe whose GOOS a program
  set to another system.

## Execution Record
