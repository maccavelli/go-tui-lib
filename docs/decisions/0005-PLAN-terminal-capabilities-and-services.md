---
status: proposed
date: 2026-10-02
associated-madr: "0005-MADR-terminal-capabilities-and-services.md"
---
# Implement terminal capabilities and services (`termcap`, `termsvc`)

Associated MADR: [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md)

*Revised 2026-10-02.* The owner answered the MADR's Q1–Q4, and the MADR is
`accepted`. Q2 departs from the recommendation: mode 2031 is subscribed by
default, so Steps 3, 4 and 7 changed, and the original opt-in wording is
struck through where it stood. MADR amendment A1 (proposed) adds Steps A1.1
to A1.3, which run between Step 6 and Step 7 only once A1 is accepted.

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
| A1.1 | `termcap/` | `FromEnv` and `Identity`, reasons, kitty flag policy, appearance chain, palette, tmux argv, legacy console (pending A1) |
| A1.2 | `termcap/`, `termcap/termcaptest/` | DA2, reply caps, `IsReplyFragment`, the JetBrains and editor gates (pending A1) |
| A1.3 | `termcap/`, `termsvc/` | `Findings`; clipboard status and plans; link display and policy; notification results; title, activity, pointer, progress (pending A1) |
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
5. **Commit.** One commit per step, with `git commit --no-edit`, after the
   owner authorizes commits to `main` in that turn. The execution record
   gets each step's evidence before its commit.

## Implementation Steps

### Step 1: records and the spike

* ~~The owner accepts the MADR, answering Q1–Q4. Record the answers, set the
  MADR `accepted` and this PLAN `in-progress`, and update `docs/README.md`.~~
  Done 2026-10-02 for the answers and the MADR's status. When Step 2
  starts, set this PLAN `in-progress` and its `docs/README.md` row to
  match. If A1 has been answered by then, record those answers in the MADR
  first.
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

**Pending A1's acceptance.** These steps run after Step 6 and before
Step 7, only once the owner accepts MADR amendment A1 and answers its
Q5–Q7. If A1 is still proposed when Step 6 is done, Step 7 closes this
PLAN without them, and A1 gets a plan of its own under this number. Each
step follows the rules above, mutation proofs included.

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
  the OSC 4 and OSC 10 queries placed as Q5 decides.
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
* The JetBrains gate as Q7 decides, and the editor-terminal gate on the
  gated set.
* **Tests:**
  * the batch's exact bytes now include DA2 before DA1;
  * DA1 `1;2` with DA2 `1;95;0` yields the Apple Terminal brand, and
    either alone does not;
  * a 1025-byte reply is not parsed and carries the reason;
  * under JetBrains the batch matches Q7's answer, and every fact carries
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
  cleaning, `WithGate`, and the `UnlessFocused` policy if Q6 adds it.
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

None yet.
