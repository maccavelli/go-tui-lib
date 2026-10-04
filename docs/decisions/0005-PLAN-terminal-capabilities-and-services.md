---
status: in-progress
date: 2026-10-04
associated-madr: "0005-MADR-terminal-capabilities-and-services.md"
---
# Implement terminal capabilities and services (`termcap`, `termsvc`)

Associated MADR: [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md)

*Revised 2026-10-02.* The owner answered the MADR's Q1–Q4, and the MADR is
`accepted`. Q2 departs from the recommendation: mode 2031 is subscribed by
default, so Steps 3, 4 and 7 changed, and the original opt-in wording is
struck through where it stood. MADR amendment A1 (proposed) adds Steps A1.1
to A1.3, which run between Step 6 and Step 7 only once A1 is accepted.

*Later on 2026-10-02:* the owner accepted A1 and answered its Q5–Q7 with
the recommendations. Steps A1.1 to A1.3 are in scope, between Step 6 and
Step 7.

## Goal

Ship `termcap`, `termcap/termcaptest` and `termsvc`, so that a program
learns its terminal's capabilities with one probe that ends by DA1 or by
timeout, sees live light and dark changes by default and restores the
terminal when it quits, and sends notifications, clipboard writes, links
and prompt marks that suit that terminal, over SSH and inside tmux.

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner can tag the next minor release after `v0.2.0`.

## Scope

### Prerequisites

* [0004-PLAN-integrate-charm-v2-and-go-1-27.md](0004-PLAN-integrate-charm-v2-and-go-1-27.md)
  is `complete`, so `github.com/charmbracelet/ultraviolet` is a direct
  requirement named by a record, confined by `depguard` to
  `internal/cells`.
* [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  is `complete`, so `internal/conformance` resolves uses with `go/types`.

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0005-*`, `docs/README.md` | accept the records; the fake-terminal spike, on a scratch copy |
| 2 | `internal/termevent/`, `.golangci.yml` (the ultraviolet `depguard` rule); `termcap/` | the event decoder; `Env`, `Support`, `Origin`, `Fact`, `Mux`, `Caps`, environment facts, JSON |
| 3 | `termcap/` | `Prober`: batch, sentinel, timeout, observation, colour scheme, `Restore`, `Query`, overrides |
| 4 | `termcap/termcaptest/` | scripted fake terminals, and the integration tests that use them |
| 5 | `termcap/` | `Report` and its goldens |
| 6 | `termsvc/` | `Notifier`, `Copy`, `Link`, prompt marks, `Wrap` |
| A1.1 | `termcap/` | `FromEnv` and `Identity`, reasons, kitty flag policy, appearance chain, palette, tmux argv, legacy console (A1) |
| A1.2 | `termcap/`, `termcap/termcaptest/` | DA2, reply caps, `IsReplyFragment`, the JetBrains and editor gates (A1) |
| A1.3 | `termcap/`, `termsvc/` | `Findings`; clipboard status and plans; link display and policy; notification results; title, activity, pointer, progress (A1) |
| 7 | `README.md`, `docs/`, `docs/guides/terminal-capabilities.md` | documentation, release notes, close-out |

No module is added. `go.mod` already requires ultraviolet directly after
0004-PLAN. The only configuration change is the `depguard` rule, which
widens from `internal/cells` to `internal/cells` and `internal/termevent`
(MADR §1).

### Out of scope

* Image protocols (Kitty placeholders, Sixel, iTerm2) and an image pane.
  They are a later record, which adds `Query` values (MADR §6).
* Enabling mode 2048 (MADR owner question Q3).
* Native clipboard or notification back ends. They are hooks a program
  supplies (MADR Q4). Every command A1 adds (`TmuxQuery`,
  `TmuxLoadBuffer`, `ImageReadCommands`) is returned as an argv, never run.
* Mode plans, teardown, hand-off and Windows console helpers (the later
  `termmode` record), and inline scrollback (the later `inline` record).
* Any change to `workspace`, `theme` or `keymap`. 0004-MADR and
  [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) consume `Caps` in
  their own PLANs.
* `git push` and tags, which are the owner's.

## Rules for every step

1. **Order.** Each step's package compiles, passes its tests and passes the
   pre-add gate before the next step starts.
2. **Mutation proofs.** Each step names mutations of its key invariants.
   Each is applied to a scratch copy and must make a named test fail. A
   mutation that survives, or does not compile, is replaced and recorded.
3. **Checks per step:**
   * `make pre-add-check FILES=…`;
   * `make lint`;
   * `go test -race -count=1 ./...`;
   * `LC_ALL=C go test ./...`;
   * the Windows test host on a scratch copy;
   * `go mod tidy -diff`.
4. **Conventions.** Every package follows 0001-MADR §6. In particular,
   nothing writes to `os.Stdout` or `os.Stderr`, nothing sets the alternate
   screen or installs a signal handler, and nothing reads the process
   environment: the environment comes from `tea.EnvMsg`.
5. **Commit.** One commit per step, made by the owner. At the end of each
   step the agent stops with the pre-add checks passed and the step's
   evidence in the execution record, and stages and commits nothing. The
   owner commits with `git commit --no-edit` and pushes. (2026-10-02: the
   org rules forbid agent commits to `main`, and the owner chose this over
   a `feature/` branch.)

## Implementation Steps

### Step 1: records and the spike

* ~~The owner accepts the MADR, answering Q1–Q4. Record the answers, set the
  MADR `accepted` and this PLAN `in-progress`, and update `docs/README.md`.~~
  Done 2026-10-02 for the answers and the MADR's status. When Step 2
  starts, set this PLAN `in-progress` and its `docs/README.md` row to
  match. A1's answers are recorded in the MADR.
* **Spike, on a scratch copy, nothing committed.** Run a real `tea.Program`
  with `tea.WithInput` (a pipe), `tea.WithOutput` (a buffer the script
  reads), `tea.WithEnvironment` and `tea.WithWindowSize`. Record:
  * that a non-terminal input and output work, and which renderer options
    the fake terminal needs;
  * the order in which tea's own queries (DECRQM 2026 and 2027, the Kitty
    keyboard request) and a `tea.Raw` batch from `Init` reach the output;
  * that `uv.DarkColorSchemeEvent`, `uv.PrimaryDeviceAttributesEvent` and
    `uv.UnknownOscEvent` arrive in `Update` as themselves;
  * that `tea.Sequence(tea.Raw(seq), tea.Quit)` writes `seq` before the
    program exits;
  * for A1: how the DA2 reply arrives (a tea or ultraviolet type, or an
    unknown CSI), and whether a reply split across reads, or arriving after
    the timeout, ever reaches `Update` as key messages.

  A finding that contradicts the MADR's evidence stops the PLAN for an
  amendment.

### Step 2: the event decoder and `termcap` facts

* **`internal/termevent`:** `Decode`, `Event` and `Kind`, as MADR §1. The
  `depguard` rule in `.golangci.yml` allows ultraviolet in
  `internal/termevent` as well as `internal/cells`, and nowhere else.
* `Support`, `Origin` (strength order `NotQueried` < `Heuristic` < `Env` <
  `Query` < `Override`), `Fact[T]`, `Mux` and `Caps`, as in MADR §2, each
  with `String`.
* `Env`, a `[]string` of `KEY=value` with `Getenv` and `LookupEnv`, built
  from `tea.EnvMsg`.
* Environment facts from an `Env`: `Terminal` (TERM_PROGRAM, then TERM),
  `Mux` (TMUX, STY, ZELLIJ), `Remote` (SSH_TTY, SSH_CONNECTION).
* A merge rule: a fact is replaced only by an equal or stronger origin.
* `Caps` JSON with `omitzero`, and a round trip.
* **Tests:**
  * `Decode` maps each ultraviolet event type MADR §1 lists, including the
    four unknown-sequence events, and reports false for anything else;
  * the merge rule over every pair of origins;
  * an override is never replaced;
  * each environment variable sets its fact, with origin `Env`;
  * JSON round trip; a zero `Caps` marshals to `{}`;
  * a `go/types` walk of `termcap`'s exported API finds no ultraviolet
    type.
* **Mutations:**
  * `termcap` imports ultraviolet directly (`depguard` must report it);
  * `Decode` drops `LightColorSchemeEvent`;
  * a weaker origin replaces a stronger one;
  * `STY` is ignored;
  * `omitzero` is dropped from one field.

### Step 3: `termcap` probe

* `Prober`, `New`, `Init`, `Update`, `Caps`, `Restore`, `Quit`, `Query`
  and the options of MADR §3: `WithTimeout`, `WithQuery`,
  `WithoutHeuristic`, `WithOverride`, ~~`WithColorSchemeUpdates`~~
  `WithoutColorSchemeUpdates` (MADR Q2), `WithoutBackgroundRequest` and
  `WithDisabled`.
* The batch order of MADR §3: the Kitty keyboard request, the DECRQM set
  and DSR 996, the gated queries (wrapped for tmux where the MADR says),
  each added query, then DA1.
* The heuristic, as one function over `Env`, with a table test.
* Observation of `ModeReportMsg`, `KeyboardEnhancementsMsg`,
  `TerminalVersionMsg`, `CapabilityMsg`, `BackgroundColorMsg`,
  `ColorProfileMsg`, `tea.EnvMsg`, and the pass-through events through
  `termevent.Decode`.
* `Reply`, built for each message, with `Raw` set from an unknown-sequence
  `termevent.Event`.
* OSC 99 reply parsing from `Reply.Raw`,
  written from the Kitty specification. Nothing is copied from crush
  (FSL-1.1-MIT).
* **Tests,** feeding decoded messages:
  * the batch's exact bytes for a known environment, inside and outside
    tmux, with and without the heuristic;
  * the batch never contains DECRQM 2026, DECRQM 2027 or XTGETTCAP;
  * DA1 arriving alone marks every query sent before it `Unsupported`;
  * a reply after DA1 still updates its fact;
  * the timeout yields `TimedOut`, and `Unknown` for unanswered queries,
    run under `testing/synctest` so the 2 s default takes no wall-clock
    time;
  * `CapsMsg` is delivered exactly once, whichever comes first;
  * tea's `ModeReportMsg` for 2026 and 2027 fills `SyncOutput` and
    `GraphemeWidth`; with no report, they stay `NotQueried`;
  * DSR 997 after the probe yields `ColorSchemeMsg` and a background
    request;
  * ~~`WithColorSchemeUpdates` sets 2031 only when it is `Supported`, and
    `Restore()` then returns its reset; otherwise `Restore()` is empty;~~
    by default the prober sets 2031 only when it is `Supported`, and
    `Restore()` then returns its reset; with `WithoutColorSchemeUpdates`,
    or when 2031 is not `Supported`, it sets nothing and `Restore()` is
    empty;
  * `Quit()` yields `Restore()`'s bytes before `tea.Quit`, and is safe when
    `Restore()` is empty;
  * `WithDisabled` makes `Init` return nil;
  * an added `Query` is sent before DA1 and parses its reply;
  * an override beats a query.
* **A source test** fails on `os.Getenv`, `os.LookupEnv`, `os.Environ` or
  `os/exec` in the package.
* **Mutations:**
  * the batch includes DECRQM 2026;
  * DA1 is sent first;
  * a query with no reply stays `Unknown` after the sentinel;
  * `CapsMsg` is sent twice;
  * `Restore` omits `ResetModeLightDark`;
  * the default does not subscribe to 2031;
  * `Quit()` orders `tea.Quit` before the restore;
  * the tmux wrap is skipped;
  * the source test skips `os.LookupEnv`.

### Step 4: `termcaptest`

* `Terminal`, a scripted fake terminal that sits between a `tea.Program`'s
  output and input. It recognises queries in the output, and writes the
  replies a profile defines. It never touches a real terminal.
* Profiles: Kitty-like (every reply), xterm-like (DA1, DECRQM, OSC 11, no
  Kitty, no OSC 99), tmux (its own DA1 and XTVERSION, with and without
  passthrough to a Kitty-like outer terminal), DA1-only, and silent.
* `Run(t, model, profile)` starts the program, waits for `CapsMsg`, and
  returns the `Caps`. It is exported so consumers can test their own
  programs against the same profiles.
* **Tests:**
  * each profile yields its expected `Caps`;
  * the silent profile ends by timeout, with a short `WithTimeout`. A real
    program blocks on pipe I/O, which `testing/synctest` does not treat as
    durably blocked, so the timeout logic itself is proven in Step 3 under
    `synctest` and only the wiring is proven here;
  * the program's output contains no DECRQM 2026 or 2027 beyond tea's own;
  * a DSR 997 written mid-run reaches the model as `ColorSchemeMsg`;
  * for the Kitty-like profile, which supports 2031, the fake terminal
    sees `SetModeLightDark` after the probe and `ResetModeLightDark` before
    the program exits through `Quit()`; with `WithoutColorSchemeUpdates`
    it sees neither.
* **Mutations:**
  * the fake terminal answers out of order (the sentinel test must fail);
  * the tmux profile forwards queries without passthrough;
  * the test program quits with `tea.Quit` instead of `Quit()` (the
    reset-before-exit test must fail).

### Step 5: `Report`

* `Report(w, caps, …)` as MADR §4: one ASCII line per fact, its value and
  its origin, with the explanation for `NotQueried` facts.
* **Tests:** goldens for every `termcaptest` profile at 80 and 120 columns;
  every `Caps` field appears in the report (a reflection walk, so a new
  field is covered automatically); the output is ASCII.
* **Mutations:** the reflection walk skips the last field; an origin is
  printed as its number.

### Step 6: `termsvc`

* `Notification`, `Urgency`, `Protocol`, `Policy`, `Notifier`,
  `NewNotifier`, `Backend`, `Copy`, `Link`, `PromptStart`, `CommandStart`,
  `CommandExecuted`, `CommandFinished` and `Wrap`, as MADR §5.
* **Tests:**
  * byte-exact sequences for OSC 99 (title, body, ID, urgency), OSC 777,
    OSC 9 and the bell;
  * `Auto` picks each protocol from the matching `Caps`;
  * inside tmux, notifications and the extra OSC 52 are wrapped with
    `TmuxPassthrough`; inside screen with `ScreenPassthrough`;
  * under `WhenUnfocused`, nothing is sent between `FocusMsg` and `BlurMsg`;
    without any focus report it sends nothing; `Always` sends;
  * a `Backend` is called with the notification, and its error is returned
    as a message, not dropped;
  * C0, C1 and ESC bytes are stripped from titles, bodies, link URLs and
    link text;
  * `Link` refuses `javascript:` and `data:`, and accepts `http`, `https`,
    `file` and `mailto`;
  * the prompt marks match `ansi.FinalTerm*`.
* **A source test** as in Step 3.
* **Mutations:**
  * notifications ignore focus;
  * the strip is skipped for the body;
  * `Link` accepts any scheme;
  * the tmux wrap is skipped for notifications.

### Steps A1.1 to A1.3: amendment A1

~~**Pending A1's acceptance.** These steps run after Step 6 and before
Step 7, only once the owner accepts MADR amendment A1 and answers its
Q5–Q7. If A1 is still proposed when Step 6 is done, Step 7 closes this
PLAN without them, and A1 gets a plan of its own under this number.~~
*2026-10-02:* A1 is accepted, and Q5–Q7 took the recommendations. These
steps run after Step 6 and before Step 7. Each step follows the rules
above, mutation proofs included.

#### Step A1.1: identity and new facts

* `Brand`, `Editor`, `Identity` and `FromEnv(env, goos)`, with the
  detection order of MADR A1, and `Brand` refined from `EnvBrand` on
  Windows.
* `Fact[T].Reason`, the reason tokens as exported constants, and
  `Keyboard()`, `Links()` and `Notifications()`.
* `KeyboardFlags(c Caps) KittyFlags` and `ReleasesReported()`.
* The appearance chain: `WithAppearanceEnv(name)` (and `LC_` + name), the
  desktop hook, and `COLORFGBG`.
* `Foreground`, `Palette` and `PaletteKnown`, parsed from `Reply.Raw`, with
  the OSC 4 and OSC 10 queries sent behind the heuristic (Q5).
* `TmuxQuery`, `ParseTmux`, `TmuxFacts` and `Prober.SetTmux`.
* `LegacyConsole` and `WithConsoleHost`.
* **Tests:**
  * a table of environments, one row per brand in the detection order,
    including the traps: JetBrains with `TERM_SESSION_ID` set, tmux with
    a stale `TERM_PROGRAM`, an editor fork over SSH, and Windows with no
    terminal variables;
  * `EnvBrand` stays unknown where `Brand` is refined;
  * `KeyboardFlags` for each terminal the MADR names, and for tmux with
    and without `csi-u`;
  * the appearance chain's order, each source with its `Origin`;
    `COLORFGBG` values `0;15`, `15;0`, `default;default` and malformed;
  * `ParseTmux` on real-shaped output, on empty output and on output with
    a missing field;
  * every reason token appears in a `const` block and is unique.
* **Mutations:**
  * `TERMINAL_EMULATOR` is read after `TERM_SESSION_ID`;
  * Ghostty is given event types;
  * `LC_` + name is not read;
  * `COLORFGBG` `default` is read as dark;
  * two reason constants share a string.

#### Step A1.2: probe discipline

* DA2 in the safe set, before DA1, and `SecondaryAttributes`; the Apple
  Terminal fingerprint from DA1 and DA2.
* The 1 KiB reply cap and its reason.
* `IsReplyFragment(msg)`, from the spike's finding on split replies.
* The JetBrains gate, which sends nothing (Q7), and the editor-terminal
  gate on the gated set.
* **Tests:**
  * the batch's exact bytes now include DA2 before DA1;
  * DA1 `1;2` with DA2 `1;95;0` yields the Apple Terminal brand, and
    either alone does not;
  * a 1025-byte reply is not parsed and carries the reason;
  * under JetBrains nothing is sent (Q7), and every fact carries
    the JetBrains reason;
  * inside `NVIM` the gated queries are absent;
  * `IsReplyFragment` is true for each reply prefix the spike observed,
    and false for an ordinary `alt+[`.
* **`termcaptest`:** an Apple-Terminal-over-SSH profile and a profile that
  paints queries as text, which the JetBrains gate must leave untouched.
* **Mutations:**
  * DA2 is sent after DA1;
  * the reply cap is off by one;
  * the JetBrains gate is skipped (the painting profile must show query
    bytes on its screen).

#### Step A1.3: doctor findings and services

* `Finding`, `Disposition` and `Findings(c Caps)`; `Report` prints them;
  the JSON form gains `schema_version`.
* `CopiedMsg`, `Status`, `Route`, `CopyPlan`, `TmuxLoadBuffer`,
  `ImageReadCommands`, and the 100 KB cap.
* `LinkDisplay`, `Display`, `LinkPolicy`, `Openable` and `OpenURLMsg`.
* `NotifyResultMsg`, `SkipReason`, the brand table for `Auto`, the text
  cleaning, `WithGate`, and the `UnlessFocused` policy (Q6).
* `SanitizeTitle`, `Activity`, `ParseActivity`, `ActivityBeacon`,
  `Pointer` and `ProgressSupported`.
* **Tests:**
  * every reason token that can appear in `Caps` has a finding, checked
    by walking the constants;
  * report goldens gain the findings, at 80 and 120 columns;
  * an OSC 52 copy is `Unconfirmed`; a backend's success is `Confirmed`;
    a 100 KB + 1 payload is `Failed` and sends nothing;
  * `ImageReadCommands` for each `goos`, with and without Wayland;
  * `LinkDisplay` for each terminal the MADR names, and under tmux 3.3
    and 3.4;
  * `Openable` refuses `javascript:`, `file:` and a URL with control
    bytes;
  * `Auto` picks each protocol by brand when OSC 99 is `Unsupported`;
  * with focus unknown, `WhenUnfocused` reports `focus-unknown`;
  * a title with ESC, U+202E and 300 characters comes out clean and 240
    long; a notification body is cut by cells, not bytes, on a CJK string;
  * `ParseActivity` rejects a version 2 payload, a timestamp 6 s ahead and
    one 15 s old, and accepts one 4 s old;
  * `Pointer` is empty under tmux.
* **Mutations:**
  * `Copy` reports a terminal route `Confirmed`;
  * the title keeps U+202E;
  * the body is cut by bytes;
  * `ParseActivity` skips the version check;
  * `Openable` accepts `file:`.

### Step 7: documentation and close-out

* **`docs/guides/terminal-capabilities.md`:**
  * embedding a `Prober` beside a workspace;
  * reading `CapsMsg` and `ColorSchemeMsg`;
  * ~~`Restore()` before quitting, and why;~~ quitting through `Quit()`,
    or sending `Restore()` first, which every program must do because 2031
    is on by default (MADR Q2), and `WithoutColorSchemeUpdates` for
    programs that cannot;
  * if A1 landed: reason tokens, `FromEnv` in tests, running the returned
    tmux and clipboard commands, and clipboard delivery status;
  * not probing without input;
  * notifications and focus reports;
  * clipboard and links inside tmux;
  * a `doctor` command;
  * testing with `termcaptest`.
* **Docs tree:** `docs/architecture.md` gains the two packages and their
  imports; `docs/README.md` rows; README Status.
* **Release notes** in the execution record.
* **Verification** as below. Mark `complete` after CI is green on the pushed
  tree. The owner tags.

## Verification

* Every step's mutations are killed, A1's included when it landed.
* No example program, guide snippet or `termcaptest` program quits a
  `Prober` without `Quit()` or `Restore()`; the Step 4 reset-before-exit
  test is seen failing on a scratch copy whose test program uses
  `tea.Quit`.
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint` and `make vuln`;
  * `go test -race -count=1 ./...`, `go test -shuffle=on -count=2 ./...`
    and `LC_ALL=C go test ./...`.
* `go mod tidy -diff` is clean, and `go.mod`'s direct requirements are
  unchanged from 0004-PLAN's close-out.
* No package writes to `os.Stdout` or `os.Stderr`, sets `AltScreen`, calls
  `signal.Notify`, reads the process environment, or starts a process.
* **Real terminals.** A small example program prints `Report`, on at least
  two real terminals on different operating systems, once inside tmux and
  once over SSH. Its output is recorded in the execution record, with the
  terminals named by product only. It shows whether tmux delivers
  passthrough replies (an open MADR item), and that the shell receives no
  DSR 997 report after the example quits on a terminal with 2031. If A1
  landed, it also records the brand, the reasons and the findings, and a
  JetBrains terminal if one is available (MADR Q7).
* The identifier scan of 0001-PLAN V7 finds nothing. Nothing is copied from
  crush or any other project (MADR evidence).
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–7 and tags the release. pi-go
  adopts `termcap` under its own records.
* **Rollback.** Before the push, each step is one local commit. After it, a
  patch release fixes forward. `v0` allows an incompatible change, and the
  release notes say so. A program that hits a terminal the probe confuses
  can pass `WithDisabled`, or override the fact, without a new release.

## Execution Record

### Step 1: records and the spike (2026-10-04)

The owner said "proceed to 0005". Step 1 ran: the audit of the records
against the tree that 0002, 0010 and 0004 left, and the spike, on a scratch
module outside the repository. Nothing in the repository changed apart
from these records.

**Prerequisites.**
[0004-PLAN-integrate-charm-v2-and-go-1-27.md](0004-PLAN-integrate-charm-v2-and-go-1-27.md)
and
[0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
are `complete`, and `v0.2.0` is tagged on `0224d0e`. `go.mod` requires
ultraviolet directly, and `.golangci.yml`'s ultraviolet rule allows only
`internal/cells`.

**The spike's questions**, answered in MADR amendment A2, items 1–7:

| Question | Answer |
| :--- | :--- |
| non-terminal input and output; renderer options | work as is; tea reports `NoTTY`, so `termcaptest` sets `tea.WithColorProfile` |
| order of tea's queries and the `Init` batch | one write: DECRQM 2026 and 2027, then the batch with DA1 last; the Kitty push and request follow in the first render |
| pass-through events | `DarkColorSchemeEvent`, `LightColorSchemeEvent`, `PrimaryDeviceAttributesEvent`, `UnknownOscEvent` arrive as themselves |
| `tea.Sequence(tea.Raw(seq), tea.Quit)` | `seq` is the last write before `Run` returns, after tea's teardown |
| A1: DA2 | `uv.SecondaryDeviceAttributesEvent` `[1 95 0]`, not an unknown CSI |
| A1: split and late replies | the first part, after ultraviolet's 50 ms escape timeout, is a `uv.UnknownEvent`; the rest are `tea.KeyPressMsg`; a late reply arrives normally |

**Deviation D1 (2026-10-04): the PLAN stops for MADR amendment A2.** Item
6 contradicts A1's description of `IsReplyFragment`, and the audit found
that 0004's workspace already sends the background query that §3 puts in
the prober's batch (A2 item 8). Step 1 says a finding that contradicts the
MADR's evidence stops the PLAN for an amendment. A2 was written
`proposed`, with owner question Q8. The owner answered Q8 "The prober",
the recommendation, on 2026-10-04, and A2 is accepted. In scope, therefore:

* Step 2's `Decode` gains the DA2 and generic-unknown kinds, with a test
  row each, and a mutation that drops DA2;
* Step A1.2's `IsReplyFragment` becomes a `Prober` method with A2's
  definition; its test drives the spike's shape (an unknown event, then key
  presses up to the final byte) and an ordinary `alt+[`;
* Step 4 sets `tea.WithColorProfile` from each profile, and adds Q8's test
  of one background query;
* Step 7's guide says how a program with a workspace gets one background
  query.

This PLAN stays `proposed` until Step 2 starts, as Step 1 says.

### Step 2: the event decoder and `termcap` facts (2026-10-04)

The owner said "proceed". This PLAN is `in-progress`, and its
`docs/README.md` row with it.

**Deviation D2 (2026-10-04): a name.** MADR §2 names the `Origin` constant
`Env`, and §1 names the type `Env`; one package cannot declare both. The
agent stopped and asked. The owner picked "Origin becomes Environment",
the recommendation: the constant is `Environment`, its text stays `env`,
and the type and A1's `FromEnv` keep their names. MADR A2 records it, with
a note under §2's code.

**What was built.**

* **`internal/termevent`:** `Kind`, `Event` and `Decode`, as MADR §1 with
  A2's two kinds: `ColorScheme` (DSR 997, dark and light),
  `DeviceAttributes` (DA1), `SecondaryDeviceAttributes` (DA2),
  `KittyGraphics` (its payload in `Raw`), `PixelSize`, and `Unknown` for
  `uv.UnknownCsiEvent`, `UnknownOscEvent`, `UnknownDcsEvent`,
  `UnknownApcEvent` and the generic `uv.UnknownEvent`. `Attrs` is a copy,
  not the event's array.
* **`.golangci.yml`:** the ultraviolet `depguard` rule allows
  `internal/cells` and `internal/termevent`, and its message cites both
  records.
* **`termcap`:**
  * `termcap.go`: `Support`, `Origin` (in strength order, `NotQueried` to
    `Override`), `Fact[T]` with `Set`, the merge rule (an equal or stronger
    origin replaces; so an override is replaced only by an override), and
    `Mux`. Each enumeration has `String`, `MarshalText` and `UnmarshalText`
    by name.
  * `caps.go`: `Caps`, the fields of MADR §2, with `omitzero` JSON tags in
    snake case. `Background` (`color.Color`) and `Profile`
    (`colorprofile.Profile`) do not write themselves as text, so
    `MarshalJSON` and `UnmarshalJSON` add them, as `#rrggbb` and the
    profile's name, around the struct's own encoding.
  * `env.go`: `Env` with `LookupEnv` (the last entry wins, as in tea's
    environment) and `Getenv`, and the unexported `setEnv`, which fills
    `Terminal` (`TERM_PROGRAM`, else `TERM`), `Mux` (`TMUX`, `STY`,
    `ZELLIJ`) and `Remote` (`SSH_TTY` or `SSH_CONNECTION`) with origin
    `Environment`. The prober (Step 3) calls it; nothing else needs it, so
    it is not exported.

**Tests.** `TestDecode` (a row for each event, the split-reply shape from
the spike among them), `TestDecodeRefusesEverythingElse`,
`TestDecodeCopiesAttributes`; `TestOriginOrder`,
`TestSetOverEveryPairOfOrigins` (all 25 pairs), `TestOverrideIsNeverReplaced`,
`TestEnumNames`, `TestEnvLookup`, `TestEnvironmentFacts` (one row per
variable), `TestEnvironmentDoesNotReplaceAQuery`,
`TestZeroCapsMarshalsEmpty`, `TestFullCapsSetsEveryField` (a reflection
check that keeps the round trip's input full as fields are added),
`TestCapsJSONRoundTrip`, `TestCapsJSONRefusesBadValues`; and
`TestNoUltravioletInTheAPI`, which type-checks `termcap` from source and
walks every exported name through fields, methods, parameters, results and
type arguments.

**Mutations,** each on a scratch copy, all killed:

| Mutation | Killed by |
| :--- | :--- |
| `termcap` imports ultraviolet | `depguard`: "import 'github.com/charmbracelet/ultraviolet' is not allowed from list 'ultraviolet'" (exit 1; one issue, so `internal/termevent` is allowed) |
| `Decode` drops `LightColorSchemeEvent` | `TestDecode`: "light: … false; want … true" |
| `Decode` drops DA2 (A2) | `TestDecode`: "DA2: … false" |
| `Decode` drops `uv.UnknownEvent` (A2) | `TestDecode`: "split reply: … false" |
| a weaker origin replaces a stronger one (`o+1 < f.Origin`) | `TestSetOverEveryPairOfOrigins`: "Set from not-queried over heuristic reported true" |
| `STY` is ignored | `TestEnvironmentFacts`: "Mux = {Value:none …}, want screen" |
| `omitzero` dropped from `sixel` | `TestZeroCapsMarshalsEmpty`: `zero Caps = {"sixel":{}}` |
| an ultraviolet type in `termcap`'s API | `TestNoUltravioletInTheAPI`: "Leak: github.com/charmbracelet/ultraviolet.Event" |

**Checks.**

* `make pre-add-check FILES=…` (the seven new files): `7 file(s) clean in
  1 module(s)`.
* `make lint`: 0 issues for linux, darwin and windows. Its first run
  failed in `make modernize` on two `go fix` suggestions (`slices.Backward`
  in `LookupEnv`, a range over an integer in a test), applied by hand; the
  planted-leak mutation's anchor moved with the import, and every mutation
  was rerun after.
* `go test -race -count=1 ./...`, `LC_ALL=C go test ./...` and
  `go test -shuffle=on -count=2 ./...`, all with `GOWORK=off`: pass.
* `GOWORK=off go mod tidy -diff`: clean. `go.mod` is unchanged; no
  `go.work.sum` exists.
* Windows test host, on a copy (go1.27.1 windows/amd64): `make
  pre-add-check` (`47 file(s) clean`), `make lint` (0 issues, three
  targets), `make vuln` (no vulnerabilities), and the two packages' tests
  shuffled twice: all exit 0.
* The identifier scan of the diff finds nothing.

`internal/conformance` scans both new packages with every other; it
passes. Nothing is staged or committed: the owner commits Steps 1 and 2.

### Step 3: `termcap` probe (2026-10-04)

The owner committed and pushed Steps 1 and 2 and said "proceed".

**Deviation D3 (2026-10-04): the batch waits for the environment.** Before
any code, the agent read tea's start-up (`tea.go:1090-1135`): `tea.EnvMsg`
is sent from a goroutine just before `Init` is called, unordered with
`Init`'s command. MADR §3 has `Init` return the batch, but the batch needs
the environment for Q1's gate and the tmux wrap. The agent stopped and
asked. The owner picked "Batch on the first EnvMsg", the recommendation:
`Init` returns the deadline, and `Update` sends the batch on the first
`tea.EnvMsg`. MADR amendment A3 records it. Step 3's tests change with it:
"`WithDisabled` makes `Init` return nil" stands, and the batch tests drive
a `tea.EnvMsg` first.

**Deviation D4 (2026-10-04): a second name.** The first build of
`prober.go` failed: `termcap.go:55:2: Query redeclared in this block`.
MADR §2's `Origin` constant `Query` and §3's type `Query` clash, as `Env`
did in D2; the agent should have seen both at Step 2. It stopped and asked.
The owner picked "Origin becomes Queried", the recommendation: the
constant is `Queried`, its text stays `query`, and the type keeps its name.
MADR A3 records it.

**Deviation D5 (2026-10-04): the API test and `tea.Msg`.** With the prober
in place, Step 2's `TestNoUltravioletInTheAPI` failed:
`New.Init: github.com/charmbracelet/ultraviolet.Event`. Bubble Tea declares
`type Msg = uv.Event` (`tea.go:50`), and the walk unaliased it, so the
MADR's own `Update(msg tea.Msg)` could never pass. Step 2 did not see it,
because `termcap` named no tea type then. The agent stopped and asked. The
owner picked "Stop at other packages' aliases", the recommendation: the
walk stops at an alias declared outside ultraviolet, and still fails on any
name declared in ultraviolet. MADR A3 records it. The planted-leak mutation
is rerun, and a second one plants an alias declared in ultraviolet.

**Deviation D6 (2026-10-04): ultraviolet in `termcap`'s tests.** `make
lint` failed: `termcap/prober_test.go:13:2: import
'github.com/charmbracelet/ultraviolet' is not allowed from list
'ultraviolet'` (`depguard`). The tests build ultraviolet's pass-through
events to feed `Update`, as MADR Confirmation says, and the rule covers
test files. The agent stopped and asked. The owner picked "A termeventtest
helper", the recommendation: a test-support package,
`internal/termevent/termeventtest`, inside the tree the rule allows, builds
each event and returns it as a `tea.Msg`. `.golangci.yml` is unchanged.
MADR A3 records it. Added to Step 3's paths: `internal/termevent/termeventtest/`.

**What was built.**

* **`termcap/prober.go`:** `Prober`, `New`, `Init`, `Update`, `Caps`,
  `Restore`, `Quit`; `CapsMsg`, `ColorSchemeMsg`, `Reply`, `Query`,
  `DefaultTimeout` (2 s); and the options `WithTimeout`, `WithQuery`,
  `WithoutHeuristic`, `WithOverride`, `WithoutColorSchemeUpdates`,
  `WithoutBackgroundRequest` and `WithDisabled`.
  * `Init` returns the deadline (A3). The first `tea.EnvMsg` sets the
    environment's facts and sends one `tea.Raw`: the Kitty keyboard
    request; DECRQM 2031, 2048 and 1004; DSR 996; behind the heuristic,
    XTVERSION, the OSC 99 `p=?` query and the Kitty graphics query, the
    last two wrapped with `TmuxPassthrough` inside tmux; each added query;
    DA1. `tea.RequestBackgroundColor` goes beside it unless declined.
  * Built-in queries are `Query` values with an unexported "what silence
    means". When DA1 answers, every query sent before it with no reply and
    a fact of its own becomes `Unsupported` with origin `Queried`. A reply
    after DA1 still sets its fact.
  * Tea's own `ModeReportMsg` for 2026 and 2027 fills `SyncOutput` and
    `GraphemeWidth`; with no report they stay `NotQueried`.
  * `Dark` comes from DSR 997 when one arrived, else from OSC 11's
    `IsDark`. After the probe each DSR 997 sends `ColorSchemeMsg`, then a
    background request unless declined.
  * Mode 2031 is set once it is `Supported`, unless declined; `Restore`
    then returns its reset, and `Quit` sequences the reset before
    `tea.Quit`.
  * `WithDisabled`: `Init` is nil, nothing is written, and the first
    `tea.EnvMsg` delivers a `CapsMsg` with the environment's facts only, so
    a program that waits for one is not left waiting.
  * Overrides run on the starting facts and on every copy handed out.
  * The heuristic, `gatedAllowed(env)`: never Apple Terminal; locally,
    always; over SSH only when `TERM` names kitty, Ghostty, WezTerm,
    Alacritty, foot, Rio or Contour, or `LC_TERMINAL` is `iTerm2`.
  * The OSC 99 reply is parsed from `Reply.Raw`, written from the Kitty
    desktop-notifications specification: the metadata must carry `p=?` and
    the prober's `i=termcap`. The keys after it are not read yet.
* **`internal/termevent/termeventtest`** (D6): a constructor for each
  event `Decode` reads, returning a `tea.Msg`.
* **`termcap/source_test.go`:** fails on `os.Getenv`, `os.LookupEnv`,
  `os.Environ`, `os.ExpandEnv`, `syscall.Getenv`, `syscall.Environ`, a
  dot import of either package, or an import of `os/exec`, in the
  package's own files, under any import name.
* **`termcap/api_test.go`:** the alias rule of D5.
* **`termcap/termcap.go`:** `Queried` (D4), and a package-doc sentence on
  the prober.

**Tests** (`prober_test.go`, decoded messages fed to `Update`):
`TestBatchBytes` (exact bytes: local, inside tmux, an unknown SSH peer,
the same with `WithoutHeuristic`, Apple Terminal; one background request
each), `TestBatchNeverAsksWhatTeaAsks` (no DECRQM 2026 or 2027, no
XTGETTCAP), `TestBatchGoesOutOnceOnTheFirstEnvironment`,
`TestGatedAllowed`, `TestSentinelAloneMarksEveryQueryUnsupported`,
`TestEveryReplySupports`, `TestRepliesThatSayNo`,
`TestXTVersionOfTmuxIsAMux`, `TestReplyAfterTheSentinelStillCounts`,
`TestTimeoutEndsTheProbe` (under `testing/synctest`: the default 2 s and
a 50 ms timeout fire at exactly their length, with no wall-clock wait),
`TestCapsMsgIsDeliveredOnce` (both orders, and a second DA1),
`TestAnotherProbersTimeoutIsIgnored`, `TestTeasOwnModeReports`,
`TestColorSchemeAfterTheProbe` (with and without the background request),
`TestBackgroundSetsDarkWithoutDSR997`, `TestColorSchemeReportsSubscription`
(supported, not recognised, declined; set at most once),
`TestQuitRestoresFirst`, `TestDisabledSendsNothing`, `TestAddedQuery`,
`TestOverrideBeatsAQuery`, `TestCapsIsACopy`; and
`TestNoProcessEnvironmentOrProcesses`, `TestEveryEventDecodes`.

**Mutations,** each on a scratch copy, all 17 killed:

| Mutation | Killed by |
| :--- | :--- |
| the batch includes DECRQM 2026 | `TestBatchBytes` (every environment) |
| DA1 is sent first | `TestBatchBytes` |
| a query with no reply stays `Unknown` after the sentinel | `TestSentinelAloneMarksEveryQueryUnsupported`: "ColorSchemeReports = {Value:unknown Origin:not-queried} after DA1 alone" |
| `CapsMsg` is sent twice | `TestCapsMsgIsDeliveredOnce`: "timeout, then sentinel: 2 CapsMsg, want 1" |
| `Restore` omits `ResetModeLightDark` | `TestColorSchemeReportsSubscription`, `TestQuitRestoresFirst` |
| the default does not subscribe to 2031 | `TestColorSchemeReportsSubscription`: "default, supported: wrote \"\"" |
| `Quit` orders `tea.Quit` before the restore | `TestQuitRestoresFirst`: "Quit() = [{} {[?2031l}]" |
| the tmux wrap is skipped | `TestBatchBytes`: "inside tmux" |
| the heuristic lets Apple Terminal through | `TestGatedAllowed`, `TestBatchBytes` |
| a DSR 997 during the probe sends `ColorSchemeMsg` | `TestEveryReplySupports` |
| `os.LookupEnv` planted | `TestNoProcessEnvironmentOrProcesses`: "reads the process environment with os.LookupEnv" |
| `os.Getenv` planted through `goos "os"` | the same test: "with os.Getenv" |
| `os/exec` planted | the same test: "imports os/exec" |
| an ultraviolet type in the API (`func() uv.Event`) | `TestNoUltravioletInTheAPI`: "Leak: …ultraviolet.Event" |
| an alias declared in ultraviolet (`uv.Rectangle`) | `TestNoUltravioletInTheAPI`: "Leak: …ultraviolet.Rectangle" |
| an alias of ultraviolet declared in `termcap` | `TestNoUltravioletInTheAPI`: "Leak: …ultraviolet.Event" |
| `termeventtest.LightColorScheme` builds a dark event | `TestEveryEventDecodes` |

`depguard`, on scratch copies: ultraviolet planted in `termcap/planted.go`
and in `termcap/planted_test.go` each fail `golangci-lint` with "import
'github.com/charmbracelet/ultraviolet' is not allowed from list
'ultraviolet'" (exit 1, one issue each).

**Checks.**

* `make lint`: 0 issues for linux, darwin and windows. Its first runs
  failed twice: `depguard` on the test import (D6), and `make modernize`
  on promoted-field composite literals in `prober.go`, which `GOWORK=off
  go fix ./termcap/` applied. Every mutation was rerun after both.
* `make pre-add-check FILES=…` (the eight Go files changed or added):
  `8 file(s) clean in 1 module(s)`.
* `go test -race -count=1 ./...`, `LC_ALL=C go test ./...` and
  `go test -shuffle=on -count=2 ./...`, with `GOWORK=off`: pass.
* `GOWORK=off go mod tidy -diff`: clean; `go.mod` unchanged; no
  `go.work.sum`.
* Windows test host, on a copy (go1.27.1 windows/amd64): `make
  pre-add-check` (`52 file(s) clean`), `make lint` (0 issues, three
  targets), `make vuln` (none), and the `internal/termevent/...` and
  `termcap` tests shuffled twice: all exit 0.
* The identifier scan of the diff finds nothing.

**Not done here.** A1's facts, reasons and gates (Steps A1.1–A1.2) and
A2's `IsReplyFragment`; the fake terminal and the background-query test
of Q8 (Step 4). Nothing is staged or committed.

### Step 4: `termcaptest` (2026-10-04)

The owner committed Step 3 (`45ead16`) and said "proceed".

**Deviation D7 (2026-10-04): the background query overtook DA1.** The
first `-race -count=3` run of the new fake-terminal tests failed once:
`tmux: Dark = {Value:false Origin:not-queried}, want true`. Step 3's
prober returned `tea.Batch(tea.Raw(batch), tea.RequestBackgroundColor)`,
and `tea.Batch` runs its commands concurrently, so tea may write the OSC 11
query after the batch's DA1; its reply then arrives after the sentinel,
and `CapsMsg` goes out without it. The tmux profile has no DSR 997 to fall
back on. A defect in Step 3's code, found by Step 4's integration test.
The agent stopped and asked. The owner picked "OSC 11 in the batch, before
DA1", the recommendation: the prober writes `ansi.RequestBackgroundColor`
inside its `tea.Raw`, after DSR 996 and before the gated queries. MADR A3
records it. Added to Step 4's paths: `termcap/prober.go` and
`termcap/prober_test.go`, whose batch tests now expect the OSC 11 bytes in
the batch and no separate background command.

**What was built.**

* **`termcap/termcaptest`:**
  * `Profile`: how a fake terminal answers, field by field (DA1, DECRQM
    per mode, the Kitty keyboard query with the flags tea pushed, DSR 996,
    OSC 11, XTVERSION, OSC 99 `p=?`, the Kitty graphics query), the
    program's environment and colour profile (A2), and for a multiplexer
    its `Outer` terminal and whether `Passthrough` reaches it.
  * Profiles: `Kitty()` (every reply), `XTerm()` (DA1, DECRQM, OSC 11),
    `Tmux(passthrough bool)` (tmux's own DA1, XTVERSION, DECRQM 1004 and
    OSC 11; the OSC 99 and graphics queries reach a `Kitty()` outer
    terminal only through allowed passthrough), `DA1Only()`, `Silent()`.
  * `Terminal`: `NewTerminal`, `Write` (the program's output: it splits
    the stream into escape sequences, holding a sequence split across
    writes, and answers each complete query in order), `Send`, `Output`,
    `Sequences`, and `Run`.
  * `Run(tb, model, profile)` and `(*Terminal).Run(tb, model)`: run the
    model as a real `tea.Program` with `WithInput` (a pipe),
    `WithOutput` (the terminal), `WithEnvironment`, `WithColorProfile`,
    `WithWindowSize(80, 24)` and `WithoutSignalHandler`; wait for the
    first `CapsMsg`; send `StopMsg`, on which the model must quit through
    its prober's `Quit`; wait for the exit; return the `Caps`. Each wait is
    bounded by `RunTimeout` (10 s) and fails the test.
* **`termcap/prober.go`** (D7): the OSC 11 query is a built-in query in
  the batch, after DSR 996, unless `WithoutBackgroundRequest`.

**Tests** (`termcaptest_test.go`):

* `TestEachProfile`: Kitty, xterm, tmux with and without passthrough and
  DA1-only each yield their expected `Complete`, `Terminal`, `Mux`,
  `Profile`, `Dark`, and the six support facts the batch asks, and tea's
  `SyncOutput`.
* `TestSilentEndsByTimeout`, with `WithTimeout(100 ms)`: `TimedOut`, not
  `Complete`, unanswered facts unknown. The timeout logic itself is proven
  under `synctest` in Step 3; this proves the wiring.
* `TestOnlyTeaAsksForModes2026And2027`: in Kitty, xterm and tmux, the
  program's whole output holds each of DECRQM 2026 and 2027 once, tea's.
* `TestColorSchemeReportMidRun`: once mode 2031 is set, a DSR 997 written
  into the input reaches the model as `ColorSchemeMsg{Dark: false}`.
* `TestResetBeforeExit`: in Kitty, mode 2031 is set after the probe and
  reset before exit, the reset being the program's last sequence; with
  `WithoutColorSchemeUpdates`, neither.
* `TestOneBackgroundQueryBesideAWorkspace` (Q8): a program that embeds a
  prober beside a workspace built with `workspace.WithoutBackgroundQuery()`
  writes exactly one OSC 11 query; without that option, two. The JetBrains
  half of Q8's test ("none under the JetBrains profile") needs A1.2's
  JetBrains gate and painting profile, and lands with Step A1.2.
* `TestSeqLen`, `TestSequenceSplitAcrossWrites`: the stream splitter on
  each sequence kind, ESC-doubling inside a tmux passthrough, and a DA1
  split over three writes.

**Mutations,** each on a scratch copy, all 7 killed:

| Mutation | Killed by |
| :--- | :--- |
| the fake terminal answers DA1 ahead of the rest (others 200 ms late) | `TestEachProfile`: "kitty: … ColorSchemeReports = {Value:unsupported …}, want … supported" |
| tmux forwards queries without passthrough | `TestEachProfile`: "tmux without passthrough: DesktopNotify = {Value:supported …}" |
| the test program quits with `tea.Quit`, not `Quit()` | `TestResetBeforeExit`: "mode 2031: set, want set after the probe and reset before exit" |
| the workspace ignores `WithoutBackgroundQuery` | `TestOneBackgroundQueryBesideAWorkspace`: "2 OSC 11 queries, want 1" |
| a sequence split across writes is dropped | `TestSequenceSplitAcrossWrites`: "sequences [], want one DA1" |
| D7 reverted: the background query a separate command again | `TestBatchBytes`: "local: batch … want" |
| `WithoutBackgroundRequest` ignored | `TestBatchBytes`: "no background query: batch … want" |

Step 3's 17 mutations were rerun after D7, all killed; one anchor ("DA1
is sent first") moved with the code and was updated first.

**Checks.**

* `make lint`: 0 issues for linux, darwin and windows, after two
  findings were fixed in the code: `errcheck` on the pipe's `Close`
  (`check-blank` refuses `_ =`; the errors now fail the test), and
  `goconst` on the xterm environment string, now `xtermEnv`.
* `make pre-add-check FILES=…` (the four Go files): `4 file(s) clean in
  1 module(s)`.
* `go test -race -count=1 ./...`, `LC_ALL=C go test ./...`,
  `go test -shuffle=on -count=2 ./...`, and `go test -race -count=10
  ./termcap/...`, with `GOWORK=off`: pass. Before D7, the third of three
  race runs failed (above).
* `GOWORK=off go mod tidy -diff`: clean; `go.mod` unchanged.
* Windows test host, on a copy (go1.27.1 windows/amd64): `make
  pre-add-check` (`54 file(s) clean`), `make lint` (0 issues, three
  targets), `make vuln` (none), and the `internal/termevent/...` and
  `termcap/...` tests shuffled, three times: all exit 0.
* The identifier scan of the diff finds nothing.

**Not settled here.** The fake tmux answers a passthrough query before
its own DA1, in order. Whether real tmux delivers passthrough replies, and
when, is still the MADR's open item for the real-terminal check in
Verification: if they arrive after tmux's DA1, the facts are `Unsupported`
in `CapsMsg` and corrected by the late reply, which Step 3 tests.

### Step 5: `Report` (2026-10-04)

The owner committed Step 4 (`bf5d4d2`) and said "proceed". No deviation.

**What was built** (`termcap/report.go`).

* `Report(w io.Writer, c Caps, o ...ReportOption) error`, `ReportOption`,
  `WithReportWidth(w)` and `DefaultReportWidth` (80). MADR §4 names no
  option; the width one follows from the PLAN's goldens at 80 and 120
  columns and TUI rule 5. Zero or less does not wrap.
* One line per `Caps` field, in field order, found by reflection, so a new
  field is reported without a change here. Each line is the field's JSON
  name (snake case of the Go name for the two fields JSON writes by hand),
  padded to 22 cells, then the value, and for a `Fact` its origin in
  brackets, as `Origin`'s text.
* A fact still `NotQueried` with a zero value reads `unknown`, not "no" or
  "none", which would state a fact. Its origin carries a reason where one
  is known: tea's 2026 and 2027 ("tea asks only outside SSH and Apple
  Terminal, and not every terminal answers"), the gated queries ("asked
  only behind the heuristic"), and, when the probe timed out, "no reply
  before the timeout". A1's reason tokens refine these in Step A1.3.
* ASCII only: any byte outside printable ASCII becomes `?`, so a
  terminal's own reply, such as its XTVERSION name, cannot put a control
  sequence into a report. Long lines wrap with `ansi.Wrap` under the value
  column. One write to `w`, whose error is returned.

**Tests.** `report_test.go`: `TestReportNamesEveryField` (a reflection
walk of `Caps`, independent of `Report`'s, over a zero and a full `Caps`),
`TestReportIsASCII` (an OSC 52 sequence and CJK text planted in the
terminal's name), `TestReportValuesAndOrigins`,
`TestReportSaysWhyAFactWasNotQueried` (with and without a timeout),
`TestReportWrapsUnderTheValueColumn` (60 columns),
`TestReportReturnsTheWriteError`. `report_golden_test.go` (package
`termcap_test`, so it can use `termcaptest`): `TestReportGolden` runs each
profile, Kitty, xterm, tmux with and without passthrough, DA1-only and
silent, through a real program and writes the report at 80 and 120
columns with `tuitest.Text`, twelve files under `termcap/testdata/golden/`,
each read before this record. The report has no colour and no glyphs, so
the matrix reduces to its widths (MADR Confirmation). The Kitty flags in
`CapsMsg` depend on whether tea's first render, which pushes them, reaches
the terminal before the probe's own query; the golden test pins them to 0
with `WithOverride`, so the files are stable. `-race -count=10` on the
report tests passed.

**One wording fix after reading the goldens.** The DA1-only golden first
said tea's 2026 query was not asked "outside SSH and Apple Terminal",
though tea did ask and the terminal did not answer. The reason now ends
"and not every terminal answers", and the four affected files were
rewritten and read again.

**Mutations,** each on a scratch copy, all 6 killed:

| Mutation | Killed by |
| :--- | :--- |
| the reflection walk skips the last field | `TestReportNamesEveryField`: "18 lines for 19 fields" |
| an origin is printed as its number | `TestReportValuesAndOrigins`: "report lacks \"terminal … WezTerm 20240203 (query)\"" |
| control bytes and non-ASCII pass through | `TestReportIsASCII`: "byte 0x1b at 81 is not printable ASCII" |
| a fact never queried reads as its zero value | `TestReportSaysWhyAFactWasNotQueried`: "report lacks \"terminal … unknown (not-queried)\"" |
| continuation lines are not indented | `TestReportWrapsUnderTheValueColumn`, `TestReportGolden` |
| the timeout is not given as a reason | `TestReportSaysWhyAFactWasNotQueried`, `TestReportGolden` |

**Checks.**

* `make lint`: 0 issues for linux, darwin and windows, after three
  findings were fixed in the code: two unchecked type assertions
  (`errcheck`'s `check-type-assertions`), now comma-ok, and `revive`'s
  confusing-naming on a test helper `report` beside `Report`, renamed
  `reportOf`. The mutations were rerun after.
* `make pre-add-check FILES=…` (the three Go files): `3 file(s) clean in
  1 module(s)`.
* `go test -race -count=1 ./...`, `LC_ALL=C go test ./...`,
  `go test -shuffle=on -count=2 ./...`, with `GOWORK=off`: pass.
* `GOWORK=off go mod tidy -diff`: clean; `go.mod` unchanged.
* Windows test host, on a copy (go1.27.1 windows/amd64): `make
  pre-add-check` (`57 file(s) clean`), `make lint` (0 issues, three
  targets), `make vuln` (none), and the `internal/termevent/...` and
  `termcap/...` tests shuffled, three times, the goldens included: all
  exit 0.
* The identifier scan of the diff and the goldens finds nothing.
