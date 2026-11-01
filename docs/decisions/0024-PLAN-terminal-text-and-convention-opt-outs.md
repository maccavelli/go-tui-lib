---
status: complete
date: 2026-10-31
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

* `Caps` gains `LinksSupport`, ~~`NotifySupport`,~~ `KeyboardRequest`,
  `PointerSupport` and `ProgressSupport`, each a `Fact` its view reads
  first, set through `WithOverride`; unset, each view is today's.
  *Amended 2026-10-31, deviation D4:* `NotifySupport` is two facts,
  `NotifyOSC9Support` and `NotifyOSC777Support`. A fact is set when its
  origin is not `NotQueried`; `KeyboardRequest` is a `Fact[int]`, the
  flags, and the others are `Fact[Support]`.
* `termsvc`: `WithTitleCells(n)` and `WithBodyCells(n)`; `MaxCopyBytes`
  applied to the OSC 52 route only; `WithCopyRoute(Route)`;
  ~~`WithLinkSchemes(schemes ...string)`;~~ `WithoutBell()`.
  *Amended 2026-10-31, deviations D5 and D6:* the schemes are
  `Linker{Schemes}`, whose `Link` method `termsvc.Link` calls as
  `Linker{}.Link`; `WithCopyRoute` takes `RouteOSC52` or `RouteOSC52Tmux`,
  forcing `PrepareCopy`'s terminal route, any other value keeping the
  default and `WithClipboard` still winning; `CopySequence` is unchanged.
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
* *Amended 2026-10-31, deviations D8 and D9:* "every string they show" is
  the text made at run time, not command metadata, enum tokens or report
  labels; `palette.Search` carries the picker's catalogue to its
  providers; `termcap.Findings` stays English, and `WithReportCatalog`
  words `Report`'s findings, as text and as JSON.
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

### Step 2: the views and `termsvc`'s limits

*2026-10-31:* the owner said "0024 terminal plan, proceed".

**Deviations,** each stopped before any code, the owner's choice dated:

* **D4, 2026-10-31: one notification fact for two protocols.**
  * *Found:* `Caps.Notifications()` (`termcap/views.go:126-142`) derives
    OSC 9 and OSC 777 apart, from the brand, and `Notifier.chosen()`
    under `Auto` tries OSC 99, then OSC 9, then OSC 777
    (`termsvc/notify.go:347-361`). One `NotifySupport` fact cannot say
    which: set to `Supported` on foot, which takes OSC 777 alone, `Auto`
    would send OSC 9.
  * *Decision:* "Two facts (Recommended)", over one fact for both:
    `NotifyOSC9Support` and `NotifyOSC777Support`, each a
    `Fact[Support]` its field of `Notifications()` reads first; OSC 99
    keeps its query.
  * *MADR:* amended, A3, since D5 names `NotifySupport`.
* **D5, 2026-10-31: `WithLinkSchemes` has no host.**
  * *Found:* `Link(rawURL, text string, params ...string)`
    (`termsvc/termsvc.go:62`) takes no options, and its variadic `params`
    leave no room for any without an incompatible change. `LinkPolicy`
    (`termsvc/links.go:51`) already holds schemes, for opening, as a
    field.
  * *Decision:* "Linker{Schemes} (Recommended)", over a second link
    function with a `LinkOption` type: `Linker`, whose `Schemes` are the
    schemes `Link` accepts, http, https, file and mailto when empty, with
    a `Link` method; `termsvc.Link` is `Linker{}.Link`. `Linker` passed
    `tools.verify names`: "names: 1 planned name(s), clean".
  * *MADR:* amended, A3, since D5 names `WithLinkSchemes`.
* **D6, 2026-10-31: the routes `WithCopyRoute` takes.**
  * *Found:* `Route` has five values, and `PrepareCopy` chooses between
    two of them, `RouteOSC52` and `RouteOSC52Tmux`
    (`termsvc/clipboard.go:144-163`); `CopySequence` takes no options.
  * *Decision:* "The two OSC 52 routes (Recommended)", over also
    `RouteTmuxBuffer`: either forces `PrepareCopy`'s terminal route,
    whatever the multiplexer; any other value keeps the default;
    `WithClipboard` still wins; `CopySequence` is unchanged.
  * *MADR:* amended, A3, which says what the option takes.
* **D7, 2026-10-31: the report and the view facts.**
  * *Found:* `termcap.Report` writes a line for every `Caps` field, by
    reflection (`termcap/report.go:84-88`), and
    `TestReportNamesEveryField` (`report_test.go:21`) holds it to that.
    The six view facts, unset, would add six "not queried" lines to every
    text report and change the 16 report goldens under
    `termcap/termcaptea/testdata/golden/`, against Verification's "every
    golden and test passes unchanged". The JSON report leaves zero fields
    out already.
  * *Decision:* "Print a view fact only when set (Recommended)", over six
    new lines in every report: `Report` leaves out a view fact whose
    origin is `NotQueried`, and prints it like any fact when a program
    sets it; `TestReportNamesEveryField` names the six as its one
    exception and checks that each appears once set.
  * *MADR:* unchanged; it says nothing of the report.

No file is added to the step's scope.

**What changed:**

* `termcap`:
  * `Caps` gains `LinksSupport`, `NotifyOSC9Support`,
    `NotifyOSC777Support`, `KeyboardRequest` (a `Fact[int]`),
    `PointerSupport` and `ProgressSupport`, each written to JSON only when
    set; `viewFacts` names them.
  * `Fact` gains the unexported `given`: its origin is not `NotQueried`.
  * `Links`, `Notifications` and `Keyboard` read their facts first, once
    given; the brand is not touched.
  * `Report` leaves out a view fact not given (D7), and its doc says so.
* `termsvc`:
  * `Pointer` and `ProgressSupported` read `PointerSupport` and
    `ProgressSupport` first: `Supported` sends or says yes, any other
    value given does not; Ghostty's empty shape is still "default".
  * `WithTitleCells`, `WithBodyCells` (0 or less keeps `TitleCells` and
    `BodyCells`) and `WithoutBell` (`Auto` gives `Off`, so
    `SkipDisabled`, where it would ring) are `NotifyOption`s; `Prepare`
    and `Sequence` cut through one `cleaned`.
  * `MaxCopyBytes` bounds OSC 52 alone: `PrepareCopy` checks it after the
    clipboard branch, so a program's clipboard takes any length; its doc,
    `ErrCopyTooLarge`'s and `termsvctea.Copy`'s say so.
  * `WithCopyRoute(Route)`, a `CopyOption`: `RouteOSC52` or
    `RouteOSC52Tmux` forces the terminal route (D6).
  * `termsvctea.CopyContext` writes `RouteOSC52Tmux`'s second copy with
    `ansi.TmuxPassthrough`, not `termsvc.Wrap(c, …)`, so a forced route
    is wrapped outside the tmux the Caps name; inside it the bytes are as
    before.
  * `Linker{Schemes}` with its `Link` method; `Link` is `Linker{}.Link`.
    Schemes compare without case, as `LinkPolicy`'s do; the default set's
    behaviour is unchanged.
* `docs/guides/defaults.md`: the view facts' row lists the six and is
  `built`; `WithoutBell`, the cells, `WithCopyRoute` and the schemes,
  whose control is `termsvc.Linker`, are `built`; a floor row, "An OSC 52
  copy's length", names `MaxCopyBytes`.

**Tests:**

* New: `termcap/viewfacts_test.go` (`TestViewFactsChangeTheirViewAlone`,
  over five brands and eight settings; `TestViewFactsNotQueriedChangeNothing`;
  `TestViewFactsThroughWithOverride`, Apple Terminal by `EnvCaps`);
  `termsvc/options_test.go` (`TestPointerSupport`, `TestProgressSupport`,
  `TestLinkDisplayFollowsLinksSupport`, `TestNotifyFactsPickTheProtocol`,
  `TestNotifierCells`, `TestWithoutBell`, `TestWithCopyRoute`,
  `TestLinker`); `termsvctea`'s `TestCopyRouteForced`.
* Changed, each by the step's own rule:
  * `TestReportNamesEveryField` names the six as its exception (D7): the
    zero `Caps` reports none, `full()` all;
  * `full()` sets the six, since `TestFullCapsSetsEveryField` requires
    every field; `report-json-full.golden`, `full()`'s JSON, gains exactly
    their six entries, regenerated and read;
  * `TestPrepareCopyRoutes` and `termsvctea`'s `TestCopyStatus` asserted
    that an oversized copy never reaches the program's clipboard; the
    step reverses that rule (D5 gap 10), and they now assert it does,
    `Confirmed`.
* Without the new options and facts every other test passes unchanged,
  the 16 text-report goldens under `termcap/termcaptea/testdata/golden/`
  among them.

**Seen failing,** through `tools.verify mutate` on scratch copies, 20
killed of 20. 2.1 to 2.8 are the step's; the rest are added so that each
new test is seen to fail:

| Mutation | Killed by |
| :--- | :--- |
| 2.1 `Links` ignores `LinksSupport` | `TestViewFactsChangeTheirViewAlone`: "kitty, LinksSupport unsupported: …"; `TestViewFactsThroughWithOverride`: "links {Value:unsupported Origin:heuristic Reason:terminal.apple-no-osc8} …; want supported by override" |
| 2.2 `Notifications` ignores `NotifyOSC9Support` | `TestViewFactsChangeTheirViewAlone`: "kitty, NotifyOSC9Support: …" |
| 2.3 `Notifications` ignores `NotifyOSC777Support` | `TestViewFactsChangeTheirViewAlone`: "kitty, NotifyOSC777Support: …" |
| 2.4 `Keyboard` ignores `KeyboardRequest` | `TestViewFactsChangeTheirViewAlone`: "kitty, KeyboardRequest 0: …" |
| 2.5 `Pointer` ignores `PointerSupport` | `TestPointerSupport`: "WezTerm, supported: "", want "\x1b]22;pointer\a"" |
| 2.6 `ProgressSupported` ignores `ProgressSupport` | `TestProgressSupport`: "kitty, supported: false, want true" |
| 2.7 `WithoutBell` ignored | `TestWithoutBell`: "WithoutBell: "\a", {… Sent:true …}; want nothing, SkipDisabled" |
| 2.8 the copy bound applied to the backend | `TestPrepareCopyRoutes`: "an oversized copy did not reach the clipboard: route none, 1 calls"; `TestCopyStatus`, the same |
| 2.9 every fact counts as given, whatever its origin | `TestViewFactsNotQueriedChangeNothing`: "kitty: …"; `TestViewFactsThroughWithOverride` |
| 2.10 the report prints a view fact not given | `TestReportNamesEveryField`: "zero: 35 lines for 29 fields" |
| 2.11 the report never prints a view fact | `TestReportNamesEveryField`: "full: 29 lines for 35 fields" |
| 2.12 `WithTitleCells` ignored | `TestNotifierCells`: "title 10: Prepare cut to 80 and 240, want 10 and 240" |
| 2.13 `WithBodyCells` ignored | `TestNotifierCells`: "body 20: Prepare cut to 80 and 240, want 80 and 20" |
| 2.14 `WithCopyRoute` ignored | `TestWithCopyRoute`: "wrapped, outside tmux: osc52, …; want osc52-tmux" |
| 2.15 `WithCopyRoute` takes any route | `TestWithCopyRoute`: "the backend is not forced: backend, …; want osc52-tmux" |
| 2.16 `termsvctea` wraps by the Caps again | `TestCopyRouteForced`: "raw writes ["\x1b]52;c;eA==\a"], want ["\x1bPtmux;\x1b\x1b]52;c;eA==\a\x1b\\"]" |
| 2.17 `Linker` ignores its `Schemes` | `TestLinker`: "ssh://host/x refused: termsvc: link scheme not allowed" |
| 2.18 2.1, through `termsvc` | `TestLinkDisplayFollowsLinksSupport`: "Apple Terminal, supported: label-and-url; kitty, unsupported: label-only" |
| 2.19 2.2, through `termsvc` | `TestNotifyFactsPickTheProtocol`: "foot with OSC 9: "\x1b]777;notify;Done;b\a", want "\x1b]9;Done: b\a"" |
| 2.20 2.3, through `termsvc` | `TestNotifyFactsPickTheProtocol`: "kitty with OSC 777: "\a", want "\x1b]777;notify;Done;b\a"" |

**Gates,** `python3 -m tools.verify gates --label p24ts2` over the step's
Go files on macOS, and through `remote linux` and `remote windows`:

| Host | `gates` |
| :--- | :--- |
| macOS | 22 passed, 0 failed, 0 skipped, 808 s; `release-check` "428 file(s) clean", with `apicheck` and the examples gate |
| `remote linux` | 21 passed, 0 failed, 1 skipped, 1109 s; "430 file(s) clean", the step's two new test files among them, which a remote run takes untracked |
| `remote windows` | 21 passed, 0 failed, 1 skipped, 2627 s; "430 file(s) clean" |

* The first macOS run failed 2 of 22, both on the step's own tests:
  `modernize`, which asked `TestReportNamesEveryField` to range over
  `ct.Fields()`, and `release-check`'s lint, errorlint over `errA !=
  errB` in `TestLinker`, which `errors.Is` alone now checks. After the
  fixes, mutations 2.10, 2.11 and 2.17 were run again and killed with the
  same lines. The remote runs begun beside it were stopped and run again.
* The second macOS run failed 1 of 22, `release-check`'s lint: gosec
  G115 over `Origin(….Uint())` in `Report`, a `uint64` to `uint8`
  conversion. `Report` now asks `reflect.Value.IsZero` of the origin.
  Mutation 2.11, re-anchored there, and 2.10 were killed again with the
  same lines; `make pre-add-check` over the step's 15 Go files was clean.
  The remote runs were stopped and run a third time with the fix.
* The skip on each remote host is `gofmt`, which is given no files there.
* `make apicheck`: "against v2.7.1, 0 incompatible change(s)".
* `TestDefaultsInventory`, the documents' checks and the identifier scan:
  clean.

**Step 2 complete, 2026-10-31,** for the owner's commit.

### Step 3: the catalogue in the existing packages

*2026-10-31:* the owner committed Step 2 (`071adc0`) and said "committed
proceed".

**Deviations,** each stopped before any code, the owner's choice dated:

* **D8, 2026-10-31: which strings the catalogue covers.**
  * *Found:* the step says "a `MessageID` constant per string they show";
    MADR D6's gap 12 cites fewer. Beyond gap 12, the four packages show:
    the 15 workspace commands' and 3 built-ins' titles and descriptions,
    and their argument help, which becomes the JSON Schema an agent reads
    (`workspace/commands.go:96-372`, `command/builtin.go`); the argument
    and pane reasons the workspace's handlers and `command.describe` give
    (`workspace/commands.go:141-248`, `command/builtin.go:40`); the danger
    badge, an enum's token (`palette/view.go:96-99`); and the report's
    labels, its JSON names.
  * *Decision:* "Run-time text, not metadata (Recommended)", over also
    command metadata and over gap 12's list alone. In scope: gap 12's
    list, and the text the four packages' own code makes at run time, the
    workspace's and `describe`'s argument and pane reasons and the
    built-ins' output labels among them. Out: command titles,
    descriptions and argument help; enum tokens such as the danger badge;
    the report's field labels; and Go error values, such as `command`'s
    schema-validation reasons and the errors an MCP text quotes, which are
    errors by Go's convention, made where no catalogue is in reach.
  * *MADR:* amended, A4, since D9 says every string a package shows has an
    ID.
* **D9, 2026-10-31: the file provider's text.**
  * *Found:* the files provider makes its limit row itself,
    "only the first %d files are searched" (`palette/files.go:214-220`),
    and does not know the picker, so a picker's `WithCatalog` cannot reach
    it.
  * *Decision:* "Search carries the catalogue (Recommended)", over a
    files option: `palette.Search` gains `Catalog`, which the picker fills
    from `WithCatalog`, so the files provider, and a program's own, word
    their rows in the program's language.
  * *MADR:* amended, A4, which names the field.
* **Readings, not questions,** stated at the stop and not objected to:
  `termcap.Findings` stays English, and `WithReportCatalog` words
  `Report`'s findings, as text and as JSON; each finding has two IDs,
  `termcap.<token>` and `termcap.<token>.fix`.

No file is added to the step's scope.

**What changed,** 89 message IDs, each with its English default in its
package's `Messages()`:

* `palette` (15, `palette/messages.go`): the help line's counts,
  "searching", "no results" and the argument count; a failed provider's
  row; the argument stage's reasons, each as "<name>: <reason>"; the files
  provider's limit row and its reason. `WithCatalog`; `Search.Catalog`,
  which the engine fills, so the files provider reads it (D9); the stage
  keeps the catalogue, and `value` and `bounds` take it. Each text is one
  line, through `sanitize.Line`.
* `workspace` (16, `workspace/messages.go`): the two results, the six
  reasons its handlers give, "focused" and "hidden" in `workspace.panes`,
  and the six help names, which `helpDescs` now keys by ID. `WithCatalog`.
* `command` (8, `command/catalog.go`): `describe`'s reason and its danger
  and arguments lines; the five MCP error texts, `mcpError` now a method of
  the registry. `WithCatalog`, a `RegistryOption`. The texts are data the
  library does not draw, so they are not cut to one line.
* `termcap` (50, `termcap/messages.go`, its constants and table generated
  from `findingText`): each of the 22 findings' message and fix, as
  `termcap.<token>` and `termcap.<token>.fix`; the two for a token with no
  finding; and the four notes `Report` writes beside a fact never queried.
  `findingText` holds IDs; `Findings` is `findingsIn(c, nil)`, in English;
  `WithReportCatalog` words `Report`'s findings and notes, as text and as
  JSON. The text report stays ASCII, so each byte outside it prints as
  `?`, as `printable` always did; `WithReportCatalog`'s doc says so.
* **A reading of D8, not a question:** the notes beside a fact never
  queried are text `Report` makes at run time, not labels, so they are in.
* `docs/guides/defaults.md`: the four packages' rows are `built`, each
  naming its `Messages`, and `palette`'s `Search.Catalog` too.
* **A slip, undone:** `command/messages.go` already holds the registry's
  messages (`ResultMsg` and the rest), and was overwritten by mistake when
  the catalogue was first written; it was restored from `HEAD` before any
  test ran, and is unchanged. The catalogue is `command/catalog.go`.

**Tests,** in each package's `messages_test.go` (`command`'s
`catalog_test.go`):

* `TestMessagesHoldEveryID`: the exported `msgcat.MessageID` constants,
  read from the package's own files with `go/parser`, against
  `Messages()`, both ways; `msgcat.Check(Messages(), Messages())`; in
  `termcap`, each finding's two IDs by its token.
* One message reworded alone changes that text alone: `palette`'s frames
  and help lines, compared with the default's with the text replaced by
  one of the same width; `workspace`'s results and reasons, through the
  registry, or through the handler where the schema's enum keeps a reason
  from it (`workspace.resize.direction`, `workspace.layout.use`), its
  splits' reason, the panes' marks and its help; `command`'s five MCP texts
  and `describe`'s three; `termcap`'s findings, fixes and notes in the text
  report, the JSON report keeping the words, and `Findings` English.
* `TestSearchCarriesTheCatalog`: a provider is asked with the picker's
  catalogue, and with none without it.
* Without a catalogue, every existing test and golden passes unchanged,
  the 16 text-report goldens and `command`'s MCP goldens among them.

**Seen failing,** through `tools.verify mutate` on scratch copies, 21
killed of 21: of each kind the step names, a call site ignoring the
catalogue, `Messages()` missing an ID and a default's arguments swapped,
in each package; the rest so that each new test is seen to fail:

| Mutation | Killed by |
| :--- | :--- |
| 3.1 palette: the count ignores the catalogue | `TestCatalogRewordsOneMessage`: "count: …" |
| 3.2 palette: `Messages` misses an ID | `TestMessagesHoldEveryID`: "Messages has no palette.file-limit-reason" |
| 3.3 palette: the count's arguments swapped in its default | `TestCatalogRewordsOneMessage`: "count: the default frame holds no "1 of 3"" |
| 3.4 palette: `Search` carries no catalogue | `TestSearchCarriesTheCatalog`: "with: Search.Catalog <nil>, want map[palette.no-results:aucun]" |
| 3.5 palette: the argument stage has no catalogue | `TestCatalogArgumentReasons`: "refused: "target: required …", want "target! required …"" |
| 3.6 palette: the file limit ignores the `Search`'s catalogue | `TestCatalogFilesLimit`: "the catalogue's: {ID:files:limit Text:only the first 1 files are searched …" |
| 3.7 palette: the failed row ignores the catalogue | `TestCatalogRewordsOneMessage`: "failed: …" |
| 3.8 palette: a number's reason ignores the catalogue | `TestCatalogValueReasons`: "a number: "not a number"" |
| 3.9 workspace: `say` ignores the catalogue | `TestCatalogRewordsOneMessage`: "workspace.no-pane-index by workspace.focus.index, reworded: "No pane 9 in the focus ring.", want "Pas de volet 9.""; `TestCatalogPanesMarks`, `TestCatalogManySplits`, `TestCatalogHelp` |
| 3.10 workspace: `Messages` misses an ID | `TestMessagesHoldEveryID`: "Messages has no workspace.help-close" |
| 3.11 workspace: the splits' arguments swapped in its default | `TestCatalogManySplits`: "the default: "has row:0, row:1 separators on screen; name one of 2"" |
| 3.12 workspace: the help ignores the catalogue | `TestCatalogHelp`: "workspace.help-next-pane reworded: ["next pane" …], want ["Xnext pane" …]" |
| 3.13 workspace: a reason the schema guards ignores the catalogue | `TestCatalogRewordsOneMessage`: "workspace.not-a-direction by workspace.resize.direction, reworded: "is not left, right, up or down"" |
| 3.14 command: `say` ignores the catalogue | `TestCatalogMCP`: "command.mcp-unknown reworded: "There is no tool named no.such; …", want "Pas d'outil no.such."" |
| 3.15 command: `Messages` misses an ID | `TestMessagesHoldEveryID`: "Messages has no command.mcp-args" |
| 3.16 command: the refusal's arguments swapped in its default | `TestCatalogMCP`: "command.mcp-refused: "The call to command: refused: … was refused: session.reset. …"" |
| 3.17 command: `describe`'s danger line ignores the catalogue | `TestCatalogDescribe`: "the danger line, reworded: …" |
| 3.18 termcap: `Report`'s findings ignore the catalogue | `TestReportCatalogRewordsOneMessage`: "termcap.terminal.unknown: …"; `TestReportCatalogJSONAndASCII`: "the JSON report: …" |
| 3.19 termcap: the notes ignore the catalogue | `TestReportCatalogRewordsOneMessage`: "termcap.note-tea-asks: …" |
| 3.20 termcap: `Messages` misses an ID | `TestMessagesHoldEveryID`: "Messages has no termcap.note-timeout" |
| 3.21 termcap: a finding's message and fix swapped | `TestMessagesHoldEveryID`: "terminal.unknown: IDs termcap.terminal.unknown.fix and termcap.terminal.unknown" |

**Gates:**

* `make pre-add-check` over the step's 23 Go files, before the gates,
  first failed on two findings in `termcap`'s new code: goconst over "None
  needed." three times in `Messages`, so `noFix` is its constant again;
  and revive's confusing-naming over the unexported `findings` beside
  `Findings`, now `findingsIn`. Mutation 3.18, re-anchored there, was
  killed again with the same lines; every other anchor was checked to
  match once. The check was then clean: "23 file(s) clean".

| Host | `gates` |
| :--- | :--- |
| macOS, with the step's Go files | 22 passed, 0 failed, 0 skipped, 771 s; `release-check` "430 file(s) clean", with `apicheck` and the examples gate |
| `remote linux` | 21 passed, 0 failed, 1 skipped, 1235 s; "438 file(s) clean", the step's 8 new files among them, which a remote run takes untracked |
| `remote windows` | 21 passed, 0 failed, 1 skipped, 2614 s; "438 file(s) clean" |

* The skip on each remote host is `gofmt`, which is given no files there.
* `make apicheck`: "against v2.7.1, 0 incompatible change(s)".
* `TestDefaultsInventory`, the documents' checks and the identifier scan:
  clean.

**Step 3 complete, 2026-10-31,** for the owner's commit.

### Step 4: the mouse and the overlay

*2026-10-31:* the owner committed Step 3 (`c10b392`) and said "committed,
proceed". No deviation: the code took the step as written.
`FocusFallback` and `FocusFallbackFunc` passed `tools.verify names` again:
"names: 2 planned name(s), clean".

**Readings, not questions,** where the step is silent:

* a `FocusFallback` that names a pane outside the focus ring is ignored,
  and focus goes to the ring's first, as without one; it is given a copy
  of the ring;
* a bare overlay is at least 1 by 1, where the framed one is 3 by 3 for its
  border;
* under `WithoutCloseOnAccept`, a hit accepted through the argument stage
  leaves the picker at the query it was accepted from, which the stage now
  keeps.

**What changed:**

* `workspace`:
  * `WithoutMouseFocus()` and `WithoutMouseResize()`: `mouseEvent` skips
    the click's focus, or the separator's drag, alone; the pane still gets
    its click and the wheel.
  * `FocusFallback`, `FocusFallbackFunc` and `WithFocusFallback`, as MADR
    A1 gives them: `fallbackFrom` chooses where focus goes when the
    focused pane can take it no longer.
  * `Overlay.Bare`: `overlayInner` is the whole rectangle for a bare
    overlay and the bordered inside for any other, and its content size,
    its click region and its cursor read it; `Render` draws a bare
    overlay's view with `viewOf`, at its size, without `renderBox`.
* `palette`:
  * `WithoutClickAccept()`: a click selects its row and returns.
  * `WithoutCloseOnAccept()`: `finish` sends `SelectedMsg` and runs the
    hit, and does not close or mark the picker done; after an argument
    stage it sets the stage's `query` again.
  * `WithoutPrefixList()`: the engine lists the prefixes for "?" only
    without it.
* `docs/guides/defaults.md`: the six rows are `built`, the fallback's
  naming `FocusFallbackFunc` too.

**Tests,** `workspace/optouts_test.go` and `palette/optouts_test.go`:

* `TestMouseOptionsEachDropOneBehaviour`: a drag of 5 cells and a click on
  a pane, by default, under each option and under both; the pane gets its
  click and the wheel each time.
* `TestFocusFallback`: by default; to `logs`, given `side` and the ring
  `[main logs]`; and to the footer, outside the ring, ignored.
* `TestBareOverlay`: a 10 by 3 bare overlay is told 10 by 3, a framed one 8
  by 1; it draws otherwise; its cursor and a click on its corner are at
  (0,0) of its pane; a 1 by 1 bare overlay is 1 by 1.
  `TestBareOverlayGolden`: `overlay-bare` across the matrix at 40 and 60
  columns, eight new goldens, read: the pane's text at the overlay's
  place, no border.
* `TestWithoutClickAccept`, `TestWithoutCloseOnAccept` (two accepts, and
  an argument stage), `TestWithoutPrefixList` (default and with).
* Without the options every existing test and golden passes unchanged.

**Seen failing,** through `tools.verify mutate` on scratch copies, 15
killed of 15. 4.1 to 4.6 are the step's; the rest so that each new
assertion is seen to fail:

| Mutation | Killed by |
| :--- | :--- |
| 4.1 `WithoutMouseFocus` also drops the drag | `TestMouseOptionsEachDropOneBehaviour`: "WithoutMouseFocus: resize 0, focused false, side's clicks 1; want 5, false, 1" |
| 4.2 `Bare` ignored when drawing | `TestBareOverlay`: "the bare overlay drew as the framed one"; `TestBareOverlayGolden` |
| 4.3 a click still accepts | `TestWithoutClickAccept`: "a click: ran ["c1"], selection 2, overlays []; want nothing run, 2, the picker open" |
| 4.4 the fallback ignored | `TestFocusFallback`: "to logs: focus main after side hid, want logs" |
| 4.5 the picker closes under `WithoutCloseOnAccept` | `TestWithoutCloseOnAccept`: "two accepts ran ["a1"]; want a1 and b1" |
| 4.6 "?" still lists the prefixes under `WithoutPrefixList` | `TestWithoutPrefixList`: "WithoutPrefixList: the prefix list true, plain searched []; want the list false" |
| 4.7 `WithoutMouseResize` also drops the focus | `TestMouseOptionsEachDropOneBehaviour`: "WithoutMouseResize: resize 0, focused false, …; want 0, true, 1" |
| 4.8 `WithoutMouseResize` ignored | `TestMouseOptionsEachDropOneBehaviour`: "WithoutMouseResize: resize 5, focused true, …; want 0, true, 1" |
| 4.9 `WithoutMouseFocus` ignored | `TestMouseOptionsEachDropOneBehaviour`: "WithoutMouseFocus: resize 5, focused true, …; want 5, false, 1" |
| 4.10 a fallback outside the ring is taken | `TestFocusFallback`: "outside the ring: focus footer after side hid, want main" |
| 4.11 a bare overlay's content keeps a border | `TestBareOverlay`: "a bare overlay's sizes [{8 1}]; want the last 10x3" |
| 4.12 a bare overlay is at least 3 by 3 | `TestBareOverlay`: "a 1x1 bare overlay's sizes [{3 3}]" |
| 4.13 a bare overlay's cursor inside a border | `TestBareOverlay`: "the cursor &{Position:{X:16 Y:5} …}; want the overlay's corner (15,4)" |
| 4.14 a bare overlay's clicks inside a border | `TestBareOverlay`: "a click on the corner reached the overlay as []; want one at (0,0)" |
| 4.15 staying open, the query is not given back | `TestWithoutCloseOnAccept`: "after the stage: args <nil>, query "", overlays [palette], rows []; want the query set again, the picker open" |

**Gates:**

* `make pre-add-check` over the step's 9 Go files: "9 file(s) clean";
  `make modernize`: clean.

| Host | `gates` |
| :--- | :--- |
| macOS, with the step's Go files | 22 passed, 0 failed, 0 skipped, 844 s; `release-check` "438 file(s) clean", with `apicheck` and the examples gate |
| `remote linux` | 21 passed, 0 failed, 1 skipped, 1423 s; "440 file(s) clean", the step's two new test files among them, untracked |
| `remote windows` | 21 passed, 0 failed, 1 skipped, 2613 s; "440 file(s) clean" |

* The skip on each remote host is `gofmt`, which is given no files there.
* `make apicheck`: "against v2.7.1, 0 incompatible change(s)".
* `TestDefaultsInventory`, the documents' checks and the identifier scan:
  clean.

**Step 4 complete, 2026-10-31,** for the owner's commit.

### Step 5: the adapters

*2026-10-31:* the owner committed Step 4 (`c098544`) and said "committed,
proceed". No deviation. `RestorerFunc` and `ContextFunc` passed
`tools.verify names`: "names: 2 planned name(s), clean".

**What changed:**

* `launch/restore.go`: `RestorerFunc func() string`, whose `Restore`
  calls it.
* `when/value.go`: `ContextFunc func(key string) (Value, bool)`, whose
  `Value` calls it.
* `docs/guides/defaults.md`: the two rows are `built`. No row this PLAN
  planned is left `planned`.

**Tests:** `launch/adapter_test.go`'s `TestRestorerFunc`, the function
called by `Restore` and its bytes written last by `Run`, through a program
that fails before `Init`, as `TestRunNotStarted` runs one;
`when/adapter_test.go`'s `TestContextFunc`, the function asked by
`Value`, by an expression's `Eval` and through `Layered`.

**Seen failing,** through `tools.verify mutate` on scratch copies, 2
killed of 2:

| Mutation | Killed by |
| :--- | :--- |
| 5.1 `RestorerFunc` ignores its function | `TestRestorerFunc`: "Restore = "" after 0 calls; want <FUNC> after 1" |
| 5.2 `ContextFunc` ignores its function | `TestContextFunc`: "Value(mode) = , false; want insert, true", and its three other checks |

**Gates:**

* `make pre-add-check` over the step's 4 Go files: "4 file(s) clean".

| Host | `gates` |
| :--- | :--- |
| macOS, with the step's Go files | 22 passed, 0 failed, 0 skipped, 863 s; `release-check` "440 file(s) clean", with `apicheck` and the examples gate |
| `remote linux` | 21 passed, 0 failed, 1 skipped, 1258 s; "442 file(s) clean", the step's two new test files among them, untracked |
| `remote windows` | 21 passed, 0 failed, 1 skipped, 2664 s; "442 file(s) clean" |

* The skip on each remote host is `gofmt`, which is given no files there.
* `make apicheck`: "against v2.7.1, 0 incompatible change(s)".
* `TestDefaultsInventory`, the documents' checks and the identifier scan:
  clean.

**Step 5 complete, 2026-10-31,** for the owner's commit.

### Step 6: documentation, the release, and the close-out

*2026-10-31:* the owner committed Step 5 (`ca5fd68`) and said "committed,
proceed".

**A reading, not a question:** Step 1's additions were released in
`v2.7.0`, with 0023-PLAN Step 7c, and "Migrating to `v2.7.0`" already
names `WithDisabled`'s deprecation and the `LookupEnv` correction. So
"Migrating to `v2.8.0`" names Steps 2 to 5, and 0020-MADR A5's bound,
documented in `b25090c`.

**What changed:**

* `docs/guides/terminal-capabilities.md`: the option table names
  `WithoutQueries`, `WithoutQuery` and `WithoutGatedQueries`, with
  `WithDisabled` as its deprecated name, and the query names; "Do not
  probe without input" says `WithoutQueries`; a new "Override one view";
  `WithoutBell`, the cells, the copy's bound and route, `Linker`, the
  pointer's and progress's facts, and `WithReportCatalog`. Step 1 left
  this prose to this step.
* `docs/guides/command-palette.md`: the three `Without…` options,
  `WithCatalog`, `?` under `WithoutPrefixList`, and `Search.Catalog`.
* `docs/guides/building-workspaces.md`: the focus fallback, the two mouse
  options, `Overlay.Bare`, and `WithCatalog`.
* `docs/guides/commands.md`: `when.ContextFunc`, `command.WithCatalog`
  and `launch.RestorerFunc`.
* `docs/guides/migrating.md`: "Migrating to `v2.8.0`": Added, Changed
  (`MaxCopyBytes` bounds OSC 52 alone), Documented (0020-MADR A5).
* `README.md`: the current release `v2.8.0`, and the defaults bullet says
  what `v2.8.0` adds.
* `docs/guides/defaults.md`: no row is `planned`; every row is `built`,
  `floor` or `decided`.

**The API,** apidiff at the `Makefile`'s pin against `v2.7.1`, compatible
changes included: 120 additions and nothing else, by package `command` 10,
`launch` 1, `palette` 21, `termcap` 58, `termsvc` 5, `when` 1 and
`workspace` 24; `make apicheck`: "against v2.7.1, 0 incompatible
change(s)".

**The smoke test, rehearsed** against the tree by a `replace`, in a scratch
module outside the repository with `GOWORK` empty (`smoke28.py local`): a
probe `WithoutQuery("da2")` writes a batch with no DA2, ending with DA1; a
picker `WithCatalog` shows "nothing here" for no results; a bare overlay of
10 by 1 is told 10 by 1 and draws its text. The control, `-control`,
expecting `v2.7.1`'s behaviour, exited 1: "the batch holds DA2: false,
want true".

**Gates,** over the tree as this step leaves it, which changes Markdown
alone:

| Host | `gates` |
| :--- | :--- |
| macOS | 21 passed, 0 failed, 1 skipped, 760 s; `release-check` "442 file(s) clean", with `apicheck` and the examples gate |
| `remote linux` | 21 passed, 0 failed, 1 skipped, 939 s; "442 file(s) clean" |
| `remote windows` | 21 passed, 0 failed, 1 skipped, 2605 s; "442 file(s) clean" |

* The skip on each host is `gofmt`, which is given no Go files.
* The documents' checks, `TestDefaultsInventory` and the examples' excerpt
  check among them, and the identifier scan: clean.

**Not done here, and the release:** the owner's commit; the disclosure
guard; the owner's push and CI; the owner's annotated tag `v2.8.0` and CI
on it; the smoke test from the proxy; and the close-out: this PLAN
`complete`, the MADR's More Information recording every part built, and
the index rows. This step stays open for them, and the PLAN
`in-progress`.

**The release, `v2.8.0`:**

* The owner committed the step (`8f0ff42`). The disclosure guard, run over
  `071adc0`, `c10b392`, `c098544`, `ca5fd68` and `8f0ff42` before the
  push, exited 0 with nothing to report, and the owner pushed `main`. CI
  run 41253075496 passed: `modules`, `gates`, and `test` on macOS 15,
  Ubuntu 24.04 and Windows 2025. Docs run 41253075563 passed.
* The owner tagged `8f0ff42` `v2.8.0`, annotated, and pushed the tag. Its
  CI run, 41254648326, passed the same five jobs.
* The proxy serves `v2.8.0` at `8f0ff42`. The consumer smoke test,
  `smoke28.py proxy`, in a scratch module outside the repository with
  `GOWORK` empty, required the tag by `go get`, then ran `go mod tidy`,
  `go vet` and `go build`. The program:
  * a probe `WithoutQuery("da2")` wrote a batch with no DA2, ending with
    DA1;
  * a picker `WithCatalog` showed "nothing here" for no results;
  * a bare overlay of 10 by 1 was told 10 by 1 and drew its text.
* Its control, `-control`, expecting `v2.7.1`'s behaviour, exited 1: "the
  batch holds DA2: false, want true".
* **Step 6 complete, 2026-10-31.**
* **This PLAN is complete, 2026-10-31.** Its Verification holds: every
  step's mutations were killed, each failure quoted; the gates passed on
  macOS, Linux and Windows at every step; `make apicheck` reported 0
  incompatible changes, and `WithDisabled` stays; `TestDefaultsInventory`
  passes with no `planned` row; without a new option every golden and
  test passed unchanged; the identifier scan found nothing. Step 1 was
  released in `v2.7.0`, Steps 2 to 6 in `v2.8.0`. With 0024-PLAN's first,
  `command-and-control-opt-outs`, complete since `v2.2.0`, every part of
  the MADR is built, which its More Information records.
