---
status: in-progress
date: 2026-10-30
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
| 4 | `workspace/`, `palette/` | `WithoutMouseFocus`, `WithoutMouseResize`, `Overlay.Bare`, `palette.WithoutClickAccept`; by MADR A1, `WithFocusFallback`, `FocusFallback`, `FocusFallbackFunc`, `palette.WithoutCloseOnAccept`, `palette.WithoutPrefixList` |
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
  ~~`color-scheme-mode`,~~ `dsr-996`, `background`, `da2`, `xtversion`,
  `notify-query`, `kitty-graphics`, `foreground`, `palette`, and the
  DECRQM modes as `mode-<n>`); a test holds the names to a table.
  *Amended 2026-10-30, deviation D1:* `color-scheme-mode` was a slip. The
  five DECRQMs are `mode-2031`, `mode-2048`, `mode-1004`, `mode-2026` and
  `mode-2027`, so the list names each of the 14 built-ins once.
* `WithoutQuery(names ...string)` drops the named queries; an unknown name
  is an error from the first `Start`, through a `ProbeDone` reason,
  `ReasonUnknownQuery`. DA1 has no name and cannot be dropped.
* `WithoutGatedQueries()` keeps the gated queries off where the heuristic
  would send them.
* `WithoutQueries()` is today's `WithDisabled`; `WithDisabled` is marked
  `// Deprecated: use WithoutQueries`.
* `Env.LookupEnv` folds case by the probe's GOOS: `WithGOOS`'s when set.
  *Amended 2026-10-30, deviation D2:* every read the probe makes of its
  `Env` (the identity, the gating heuristic, Bubble Tea's mode rule)
  folds by the probe's GOOS. `Env.LookupEnv`, which has no probe, keeps
  `runtime.GOOS`, and its comment says so.
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
* *Added 2026-10-26, by MADR A1:* `workspace.WithFocusFallback(f)`, with
  `FocusFallback` and `FocusFallbackFunc`: the pane focus moves to when
  the focused pane hides, the ring's first by default;
  `palette.WithoutCloseOnAccept()`: accepting sends `SelectedMsg` and runs
  the hit, and the picker stays open; `palette.WithoutPrefixList()`: a
  query of `?` is an ordinary query.
* **Tests:** each default unchanged; each option drops its one behaviour
  alone; a bare overlay's golden.
* **Mutations:** `WithoutMouseFocus` also drops the drag; `Bare` ignored;
  a click still accepts; the fallback ignored; the picker closes under
  `WithoutCloseOnAccept`; `?` still lists the prefixes under
  `WithoutPrefixList`.

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

### Approval, 2026-10-30

The owner: "approve 0022 and 0024, then proceed". Both prerequisites hold:
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md) is
`complete` (`a088933`), and the owner has approved this PLAN, which is now
`in-progress`. It runs after
[0022-PLAN-align-docs-ci-and-disclosure-gates.md](0022-PLAN-align-docs-ci-and-disclosure-gates.md),
approved in the same message.

### Step 1: the probe's queries

**Deviations,** each stopped before any code, the owner's choice dated:

* **D1, 2026-10-30: the query names.**
  * *Found:* the built-in queries carry names nobody can see, such as
    `color-scheme-reports`, `secondary-attributes` and `sync-output`; no
    test, golden or guide names one.
  * The Step's list names 10 tokens and "the DECRQM modes as `mode-<n>`".
    The batch has 9 queries that are not modes and 5 DECRQMs (2031, 2048,
    1004, 2026, 2027). So `color-scheme-mode` was either a slip or a
    second name for DECRQM 2031 that breaks `mode-<n>`.
  * Once `WithoutQuery` takes them, the names are API and never renamed.
  * *Decision:* "PLAN's, all modes mode-<n>", over the list taken
    literally and over the code's names. The Step is annotated above.
  * *MADR:* unchanged; D5 names no tokens.
* **D2, 2026-10-30: `LookupEnv`'s fold.**
  * *Found:* the Step and MADR D5 gap 11 say `Env.LookupEnv` folds case by
    the probe's GOOS, but an `Env` is a `[]string` and has no probe.
  * *Decision:* "The probe's own reads", over also exporting a lookup that
    takes a GOOS:
    * every read the probe makes of its `Env` folds by its GOOS;
    * `Env.LookupEnv` keeps `runtime.GOOS`;
    * a `WithQuery`'s `Seq` still reads the `Env` it is given.
  * The Step is annotated above.
  * *MADR:* amended, A2, since D5 asserted the method changes.
* **D3, 2026-10-30: the markdownlint gate on the Linux host.**
  * *Found:* `remote linux` failed one gate of 21, `markdownlint`: "npx
    canceled due to missing packages and no YES option:
    ["markdownlint-cli2@0.24.0"]". The gate ran
    `npx --no-install markdownlint-cli2` with no version
    (`tools/verify/gates.py:136`), so npx picked the newest release, which
    that host lacks. It is not this step's fault; the same gate passed
    there for 0023-PLAN Step 7b.
  * *Decision:* "Pin the gate via make (Recommended)", over pinning the version in the gate itself.
    [0018-MADR-python-verification-tools.md](0018-MADR-python-verification-tools.md)
    A4 and
    [0018-PLAN-python-verification-tools.md](0018-PLAN-python-verification-tools.md)
    Step 9 make the gate run `make markdownlint`, as its own commit ahead
    of this step's. Linux's gates are then run again.
  * *MADR:* this record's is unchanged; 0018's has A4.

**What changed** (`termcap`):

* **The names.** The 14 built-in queries carry the tokens of D1, in the
  batch's order, and `queryNames` lists them:
  * `kitty-keyboard`, `mode-2031`, `mode-2048`, `mode-1004`, `dsr-996`,
    `background`, `da2`;
  * `xtversion`, `notify-query`, `kitty-graphics`, `foreground`,
    `palette`, `mode-2026`, `mode-2027`.

  `modeProbe` names each DECRQM `mode-<n>` from its number.
  `kitty-keyboard` and `kitty-graphics` are also `ReplyKind` tokens, so
  both are package constants, `tokenKittyKeyboard` and
  `tokenKittyGraphics`, as `tokenColorScheme` was; goconst asked for it.
* **`WithoutQuery(names ...string)`:**
  * `batch` leaves out each query named, a `WithQuery`'s by its `Name`
    too;
  * leaving out `background` also leaves out the OSC 11 that follows a
    DSR 997 report;
  * a name that is neither a built-in's nor a `WithQuery`'s, or `""`,
    makes `Start` send nothing and end at once. Each query fact is
    unknown, `NotQueried`, `ReasonUnknownQuery` (`probe.unknown-query`),
    as JetBrains' are, and the hooks still run;
  * DA1 has no name and is always sent.
* **`WithoutGatedQueries()`:** the gated queries are never sent, and it
  wins over `WithoutHeuristic`. An editor's terminal gets no
  `ReasonEditorTerminal` under it, since the option is the reason.
* **`WithoutQueries()`:** `WithDisabled`'s doc and behaviour.
  `WithDisabled` calls it, marked `// Deprecated: use WithoutQueries`.
  `EnvCaps` and every test outside the package moved to the new name, so
  staticcheck reports no SA1019.
* **The GOOS fold, D2:** `Env` gains `getenvFor(key, goos)`. `setEnv`,
  `FromEnv`, `envBrand`, `gatedAllowed` and `querySynchronizedOutput` read
  through it and `lookup(key, goos)`, the probe's GOOS. `LookupEnv`'s doc
  says it keeps `runtime.GOOS`. revive's confusing-naming rule refused
  the first name, `getenv`.
* **`ReasonUnknownQuery`'s finding:** an `Issue`, "The program left out a
  query the probe does not have, so no query was sent."
* **`docs/guides/defaults.md`:** the rows of `WithoutQueries`,
  `WithoutQuery` and `WithoutGatedQueries` are `built`.

**Tests** (`termcap/queryopts_test.go`, new):

* `TestQueryNames`: the batch's names and `queryNames` against the table.
* `TestWithoutQuery`: each of the 14 names leaves out its own query, and
  the rest is the batch byte for byte, DA1 last.
* `TestWithoutQueryReplies`: a query left out is not offered its reply, and
  is not unsupported at the sentinel; the DSR 997 follow-up goes with
  `background`.
* `TestWithoutQueryAdded`: a `WithQuery` is left out by its name.
* `TestWithoutQueryUnknownName`: `da1`, `nope`, `""` and `Kitty-Keyboard`
  each send nothing, with the reason; `""` beside an unnamed `WithQuery`
  too.
* `TestWithoutGatedQueries`: local, an SSH peer the heuristic names, and
  under `WithoutHeuristic`; no editor's reason.
* `TestWithoutQueriesIsWithDisabled`: the same events and `Caps`, locally
  and under JetBrains.
* `TestProbeReadsEnvForItsGOOS`: lowercase names, read by a Windows and a
  Linux probe on this host, and by `gatedAllowed`,
  `querySynchronizedOutput` and `FromEnv`.

Without the new options every existing test, `TestProbeBatch`'s bytes
among them, passed unchanged. The new tests passed on their first run,
since the code came first; the mutations below are their failures.

**Seen failing,** through `tools.verify mutate`, 14 killed of 14. Mutation
1.2's anchor was refreshed after the constants and run again alone:

| Mutation | Killed by |
| :--- | :--- |
| 1.1 `WithoutQuery` drops nothing | `TestWithoutQuery`: "without kitty-keyboard: wrote …" for each name |
| 1.2 `WithoutQuery` drops DA1 (`da1` a name) | `TestQueryNames`; `TestWithoutQueryUnknownName`: `"da1"` wrote the batch |
| 1.3 `WithoutGatedQueries` ignored | `TestWithoutGatedQueries`: "local: wrote …", and its other two cases |
| 1.4 the probe's `getenvFor` folds by `runtime.GOOS` | `TestProbeReadsEnvForItsGOOS`: "windows: … EnvBrand {Value:unknown …}; want remote, WezTerm" |
| 1.5 `FromEnv`'s `SSH_TTY` folds by `runtime.GOOS` | `TestProbeReadsEnvForItsGOOS`: "windows: Remote {Value:false …}" |
| 1.6 the DSR 997 follow-up sent with `background` dropped | `TestWithoutQueryReplies`: "after DSR 997, wrote \"\\x1b]11;?\\a\"" |
| 1.7 an unknown name accepted | `TestWithoutQueryUnknownName`: each of the four names wrote the batch |
| 1.8 `""` names an unnamed `WithQuery` | `TestWithoutQueryUnknownName`: "WithoutQuery(\"\") beside an unnamed query wrote …" |
| 1.9 the editor's reason under `WithoutGatedQueries` | `TestWithoutGatedQueries`: "an editor's reason under WithoutGatedQueries: {… Reason:editor.terminal-gated}" |
| 1.10 a `WithQuery`'s name not known | `TestWithoutQueryAdded`: "without the added query: \"\"" |
| 1.11 `WithDisabled` no longer `WithoutQueries` | `TestWithoutQueriesIsWithDisabled` |
| 1.12 two names swapped in `queryNames` | `TestQueryNames`: "queryNames … want …" |
| 1.13 a mode named `decrqm-<n>` | `TestQueryNames`; `TestWithoutQuery`: "without mode-2031" |
| 1.14 `ReasonUnknownQuery` without a finding | `TestEveryReasonHasAFinding`: "token \"probe.unknown-query\" has no finding" |

**Gates,** `python3 -m tools.verify gates --label p24ts1` over the step's
files:

* **macOS:** the first run failed `release-check` on its lint: goconst
  over `kitty-keyboard` and `kitty-graphics`, and revive's confusing-naming
  over `getenv`. After the fixes above, "22 passed, 0 failed, 0 skipped in
  724s"; `release-check`, "419 file(s) clean", with `apicheck` and the
  examples gate.
* **`remote linux`:**
  * the first run: "20 passed, 1 failed, 1 skipped", the failure
    `markdownlint`, deviation D3;
  * after 0018-PLAN Step 9: "21 passed, 0 failed, 1 skipped in 1128s",
    "419 file(s) clean", `markdownlint` "Summary: 0 issues in 0 files".
* **`remote windows`:** "21 passed, 0 failed, 1 skipped in 2555s", "419
  file(s) clean". It ran before 0018-PLAN Step 9, with the old
  `markdownlint` command, which that host could still run ("Summary: 0
  issues in 0 files"). That change touches no gate but `markdownlint`, so
  the run stands.
* The skip on each remote host is `gofmt`, which is given no files there.
* `make apicheck`: "against v2.6.2, 0 incompatible change(s)".
  `WithDisabled` stays.
* `tools.verify scan --base HEAD`: "scan: clean (5 pattern(s))";
  `scripts/check_records.py` and `scripts/check_disclosure.py`: clean.

**Not done here:** the guides' prose (`terminal-capabilities.md`'s option
table and its advice to pass `WithDisabled`) is Step 6's, as the PLAN
places it; only the defaults rows move in this step, by the rules.

**Step 1 complete, 2026-10-30,** for the owner's commit, after 0018-PLAN
Step 9's.
