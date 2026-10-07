---
status: proposed
date: 2026-10-07
associated-madr: "0013-MADR-cli-integration-helpers.md"
---
# Implement the `launch` package with native types for every CLI framework, and release `v0.7.0`

Associated MADR: [0013-MADR-cli-integration-helpers.md](0013-MADR-cli-integration-helpers.md),
as amended by A1, under
[0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md)
W1.

## Revisions

* **2026-10-07, first draft,** committed in `0369ca7`: written for the
  MADR before A1.
* **2026-10-07, rewritten** for the MADR as accepted with A1, including
  A1.6's names (`Choice`, `WithFilter`, and `Decision` allowed in two
  packages). The
  first draft's steps are replaced, not amended. It was never approved or
  executed.

## Goal

A Go program on any common CLI framework starts go-tui-lib's TUI with
go-tui-lib's own types, and falls back to its CLI mode cleanly:

* **Its flags.** `launch.Choice` and `launch.Flags` parse `--tui` and
  `--mode` natively in stdlib `flag`, cobra/pflag, kong and urfave/cli v3,
  which are tier 1. ff v4, go-flags and go-arg are tier 2.
* **Its streams.** `launch.FromSource` takes a `*cobra.Command` as it is;
  every other framework writes one `Streams{…}` literal.
* **Its run.** `launch.Decide` decides; `launch.Run` runs the TUI only
  where it can.
  * A Loop command never waits on an ended program.
  * Modes set outside Bubble Tea are reset on every outcome.
  * A failure before the start returns `ErrNotStarted`; a crash after it
    returns `ErrCrashed`, with the last good model and the terminal as it
    was.
* **Its exit.** `launch.ExitCode` honours any error's `ExitCode() int`,
  and `launch.ExitError` carries a status through kong and urfave.
* **Its tests.** `launch/launchtest` gives fake terminals and a `Run`
  driver.

All of it ships in the root module at `v0.7.0`, with no new module in the
build, and nothing spawned.

## Scope

### Facts this PLAN starts from (2026-10-07)

| Fact | Where it was read |
| :--- | :--- |
| the root is at `v0.6.0`; the records are committed as `0369ca7` | `git tag`, `git log` |
| 0011's Steps 3–5 have not run; 0014's W0 PLAN is written beside this one | [0011-PLAN-docs-accuracy-after-v0-5-0.md](0011-PLAN-docs-accuracy-after-v0-5-0.md) |
| `github.com/charmbracelet/x/term` v0.2.2 is `// indirect` at `go.mod:16`; it exports `IsTerminal`, `GetState`, `Restore`, `GetSize` | `go.mod`; `x/term@v0.2.2/term.go:11-49` |
| `command.Mode` (`Loop`, `Async`) and `command.Decision` exist | `command/command.go:199-205`, `command/gate.go:11` |
| the registry's loop is set once by `WithLoop`; `onLoop` sends a `LoopMsg` from a goroutine and waits for a result or for `ctx` | `command/registry.go:82-88`, `command/dispatch.go:54-90` |
| Bubble Tea's `Send` is a no-op once the program has ended | `tea.go:1188-1197` |
| the prober sets mode 2031 with raw bytes and resets it only through `Quit` or `Restore` | `termcap/prober.go:220-234`, `:305-312` |
| termcap's enum text helpers are unexported generics over `~uint8` | `termcap/termcap.go:124-147` |
| `glyph` exports `Set`, `Unicode()`, `ASCII()`, `For(utf8 bool)` | `go doc ./glyph` |
| Bubble Tea's early returns (`tea.go:1045`, `:1055`, `:1110`) precede `startRenderer` (`:1116`) and `Init` (`:1127`) and do not restore | `tea.go:999-1130` |
| `Init`, `Update` and `View` run on the goroutine that calls `Run` | `tea.go:1127`, `:1153` |
| `tea.OpenTTY` returns one file as input and output on Unix; `CONIN$` and `CONOUT$` on Windows, where only stdin's reader can be cancelled | `tty.go:130`; ultraviolet `tty_unix.go:15-21`, `tty_windows.go:16-20`, `cancelreader_windows.go:28-40` |
| `colorprofile.Env` reads the environment only; `Detect` loads terminfo and may run `tmux info` | `colorprofile@v0.4.3/env.go:33-53`, `:70-72`, `:214`, `:248` |
| the framework behaviours of 0014-REPORT §3.2 (pflag ignores `IsBoolFlag` on `Var`; kong takes a bare `--tui` only into a `bool`; urfave requires `Get`; go-flags needs `UnmarshalFlag`) | 0014-REPORT §3.2, from scratch programs |
| `internal/conformance` checks rules 1 and 2 on every package | `internal/conformance/conformance_test.go:142-241` |

### Preconditions

1. **0011 is complete.** Its Steps 3–5 edit the documents Step 8 edits.
2. **0014's W0 PLAN is complete**
   ([0014-PLAN-api-policy-gates.md](0014-PLAN-api-policy-gates.md)).
   That gives this PLAN:
   * the API diff gate;
   * the collision check, whose allowlist gets this PLAN's `Decision`
     entry;
   * the conformance bans;
   * the framework-example harness, which this PLAN's guide examples join.

### In scope

| Step | Paths | Delivers |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0013-*`, `docs/README.md` | the PLAN approved |
| 2 | `internal/enum/`; `glyph/`; `command/registry.go`, `command/dispatch.go` and their tests | the three prerequisites from 0014-MADR W2 |
| 3 | `launch/doc.go`, `choice.go`, `flags.go`, `streams.go`, `decide.go`, `terminal.go`, `colour.go`, `locale.go`; tests; `go.mod`, `go.sum`; `.golangci.yml`; `AGENTS.md` | the flag types, the streams, `Decide`; x/term direct and confined |
| 4 | `launch/run.go`, `option.go`, `model.go`, `restore.go`, `errors.go`; tests | `Run` and its options |
| 5 | `launch/frame.go`, `exit.go`, `example_test.go`; tests; goldens | `Frame`, `ExitCode`, `ExitError`, the examples |
| 6 | `launch/launchtest/` | the consumer testing kit |
| 7 | scratch only; this PLAN's record | the tier-1 framework programs and the live probes |
| 8 | `internal/conformance/`, the collision allowlist; `docs/guides/commands.md`, `building-workspaces.md`, `terminal-capabilities.md`; `README.md`, `docs/README.md`, `docs/architecture.md` | conformance and the documents |
| 9 | this PLAN's record | the release `v0.7.0` |
| 10 | this PLAN, the MADRs, `docs/README.md` | close-out |

### Out of scope

* **The rest of 0014-MADR W2:**
  * `ParseArgs`, `Params`, `Complete`, `WriteResult`;
  * the exit codes on `command`'s errors, and `ErrPanicked`;
  * `workspace`'s options and `RenderPlain`, `EnvCaps`, `termsvc`'s byte
    forms.

  They are `v0.8.0`, in
  [0014-PLAN-component-native-forms.md](0014-PLAN-component-native-forms.md).
  `ExitCode` here already honours an `ExitCode()` method, so those errors
  fit when they gain one.
* **Migrating the existing enums** to `internal/enum` (0014 W4).
* **Theme v2's CP437 glyphs.** `TierLegacy` falls back to ASCII until
  then, as the owner chose
  ([0003-REPORT](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §11.5).
* **gobble's side:** its `--tui` wiring, a turn in flight at a crash, and
  its crash log.
* **`termmode`, `inline`, the `x/vt` harness:** later records.

## Implementation Steps

### Rules

1. **A step starts when the previous one is committed.** The agent
   commits on `main` only when the owner asks in that turn, with `git
   commit --no-edit`. The owner pushes and tags.
2. **Checks.**
   * **For a step that changes Go:**
     * `gofmt -l` on the changed files;
     * `make pre-add-check FILES="…"`;
     * `make lint`, all three targets;
     * with `GOWORK=off`: `go test -race -count=1 ./...`,
       `go test -shuffle=on -count=2 ./...` and `LC_ALL=C go test ./...`;
     * the tests in workspace mode;
     * `go mod tidy -diff`, `make vuln` and
       `scripts/go-modules.sh --check`;
     * `make apicheck` (0014 W0): the step's additions are compatible,
       and nothing is removed;
     * `make release-check`;
     * the Windows test host, on a copy of the tree: `make pre-add-check`,
       `make lint`, `make vuln`, and
       `GOWORK=off go test -count=3 -shuffle=on ./...`.
   * **For every step:**
     * markdownlint and the link checker on changed Markdown;
     * the citation checker;
     * the identifier scan of the diff.
3. **Each mutation runs on a scratch copy** and must fail its named test or
   check.
4. **Anything this PLAN does not say stops the step for the owner.** It is
   recorded as a dated deviation, with an amendment to the MADR if a
   decision or a fact changes.
5. **Tests are hermetic.** They never touch:
   * the process environment;
   * the process's standard streams;
   * a real terminal.

   Streams are fakes from Step 6's `launchtest`. Until Step 6, they are
   package-private fakes with the same methods. Every environment is
   explicit.

### Step 1: records

* **This PLAN, approved by the owner.**
* **`docs/README.md`:** this PLAN's row.

**Done when** the owner approves this PLAN.

### Step 2: prerequisites from 0014-MADR W2

**2a. `internal/enum`**, moving termcap's generics (`termcap/termcap.go:124-147`)
into a shared package:

```go
package enum

func Name[T ~uint8](names []string, v T) string
func Marshal[T ~uint8](pkg, kind string, names []string, v T) ([]byte, error)
func Unmarshal[T ~uint8](pkg, kind string, names []string, b []byte, v *T) error
```

* `pkg` and `kind` keep the existing error texts, "termcap: unknown Mux".
* termcap's three functions become one-line calls, and its errors and
  golden texts must not change.
* W4 migrates the other enums later.

**2b. `glyph.Tier`:**

```go
type Tier uint8

const (
    TierUnicode Tier = iota
    TierLegacy        // the legacy-console tier; ASCII until theme v2 fills it
    TierASCII
)

func (t Tier) Set() Set          // TierLegacy returns ASCII() for now
func (t Tier) String() string    // "unicode", "legacy", "ascii", through internal/enum
func (t Tier) MarshalText() ([]byte, error)
func (t *Tier) UnmarshalText([]byte) error
```

* `For(utf8 bool)` stays, documented as `TierUnicode.Set()` or
  `TierASCII.Set()`.
* A test checks that every glyph of every tier's `Set` is one cell wide.
  That test is the start of the owner's same-width rule.

**2c. The registry's loop, attachable and detachable:**

```go
// Attach makes send the registry's loop until detach is called. A Loop
// command started while attached runs on the loop; once detached, or when
// the loop does not take it, it runs on the caller instead, exactly once.
func (r *Registry) Attach(send func(tea.Msg)) (detach func())
```

* **`WithLoop(send)` stays.** It is a permanent `Attach`.
* **The registry keeps the current loop and its done channel** under its
  existing lock. `Attach` replaces both, and `detach` closes the channel
  and clears the loop. A second `detach` does nothing.
* **`onLoop`** builds the `LoopMsg` with an `atomic.Bool` claim. The
  message's `run` executes the command only if it wins the claim. `onLoop`
  then selects on three things:
  * the result;
  * the done channel: on close, if `onLoop` wins the claim, the command
    runs on the caller; if not, the loop already has it, and `onLoop`
    waits for the result;
  * `ctx`: if `onLoop` wins the claim, it returns `ctx.Err()`; if not,
    the loop is running the command, and `onLoop` waits for its result.
    Today a command whose run began just before `ctx` ended is reported
    as cancelled, and its outcome is dropped (`command/dispatch.go:71`,
    `:85-86`); the claim closes that.
* **The closure sends its outcome in a `defer`,** so a handler that
  panics on the loop still releases the caller. Bubble Tea recovers the
  panic (`tea.go:1034-1041`). Until 0014 W2's `recover` lands, the
  outcome is an error naming the panic.
* **The sending goroutine** ends when `send` returns. Bubble Tea's `Send`
  returns once the program's context ends, which `Run` does on its way
  out (`tea.go:1013`, `:1188-1197`). `detach` cannot end a `send` that
  is blocked on a program never run. So `launch` always detaches after
  `Run` returns, and the documentation of `Attach` says the same.

**Tests:**

* **`TestAttachRunsOnLoop`:** while attached, a Loop command runs inside
  the loop.
* **`TestDetachRunsOnCaller`:** after `detach`, `Run` with
  `context.Background()` returns within 1 s, with the handler run once.
* **`TestDetachRaceRunsOnce`:** 1,000 iterations under `-race`. `detach`
  fires while `send` is delivering, and the handler count is exactly one
  each time.
* **`TestCancelAfterStartKeepsOutcome`:** `ctx` ends after the loop has
  claimed the command. `Run` returns the handler's outcome, not
  `ctx.Err()`, and the audit records it.
* **`TestLoopPanicReleasesCaller`:** the handler panics on the loop, and
  `Run` returns within 1 s with an error.
* **`TestWithLoopUnchanged`:** the existing `WithLoop` tests pass
  unchanged.
* **`TestEnumNames`:** `internal/enum` round-trips, and its error texts
  are byte-for-byte termcap's.
* **`TestTierWidths`, `TestTierText`.**

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S2-1 | `detach` does not close the done channel | `TestDetachRunsOnCaller` |
| S2-2 | the claim is dropped, so both may run | `TestDetachRaceRunsOnce` |
| S2-3 | `TierLegacy.Set()` returns `Unicode()` | `TestTierText` (legacy maps to ASCII until theme v2) |
| S2-4 | `enum.Unmarshal` accepts a case-folded name | `TestEnumNames` |
| S2-5 | the `ctx` branch returns `ctx.Err()` without the claim | `TestCancelAfterStartKeepsOutcome` |
| S2-6 | the outcome is sent without `defer` | `TestLoopPanicReleasesCaller` |

**Done when** Rule 2's checks are clean, the mutations are killed, and
`make apicheck` reports only the additions `Attach`, `Tier` and its
methods.

### Step 3: the flag types, the streams and `Decide`

**API** (the MADR's A1.2, with A1.6's names):

* **`launch/doc.go`:** what the package is, what it never does (name
  `os.Stdout` or `os.Stderr`, install a signal handler, set `AltScreen`,
  spawn), the `--tui` pattern, and the stability line of 0014 W0.6:
  `Stability: stable.`

* **`launch/choice.go`:** `Choice` with `ChoiceAuto`, `ChoiceTUI` and
  `ChoicePlain`. It has every method 0014-REPORT §3.2 measured, through
  `internal/enum`:
  * `String`, `Set`, `Type`;
  * `Get`;
  * `MarshalText`, `UnmarshalText`;
  * `MarshalFlag`, `UnmarshalFlag`.

  `Type()` returns `"mode"`. The tokens are `auto`, `tui`, `plain`.
* **`launch/flags.go`:**
  * `Flags{Mode Choice; TUI bool}`, with the tags of A1.2.
  * `RegisterFlags(*flag.FlagSet)` registers `-mode` through `Var`, and
    `-tui` as a `bool`.
  * `Resolve() Choice` returns `ChoiceTUI` when `TUI` is set, and `Mode`
    otherwise.
* **`launch/streams.go`:**
  * `Streams{In io.Reader; Out, Err io.Writer; Env termcap.Env}`;
  * `StreamSource`;
  * `FromSource(src, env) Streams`.
* **`launch/decide.go`:**
  * `Config{Choice Choice; NoInputEnv string; UIOnErr, OpenTTY bool}`;
  * `Target`: `TargetNone`, `TargetStream`, `TargetErr`, `TargetTTY`;
  * the `Reason` tokens of A1.2, which are typed string constants;
  * `Decision{Interactive bool; Reason Reason; In, UI Target; Profile,
    UIProfile colorprofile.Profile; Glyphs glyph.Tier; Width, Height int}`;
  * `Decide(s Streams, c Config) Decision`.

  It calls an unexported `decide(s, c, goos, open)`.
* **The rules, in order** (§3 as amended):
  1. `ChoicePlain`;
  2. unless `ChoiceTUI`: `NoInputEnv`, then `CI` (set, not empty,
     neither `false` nor `0`);
  3. `TERM=dumb`, whatever the choice;
  4. the input target;
  5. the drawing target;
  6. interactive.

  `OpenTTY` opens the controlling terminal only when a target needs it,
  reads its size, and closes each distinct file once.
* **`launch/terminal.go`:**
  * **`isTerminal(v any)`:** the `IsTerminal() bool` override first, then
    `Fd()` with `term.IsTerminal`. Never the file mode.
  * **`size(v any)`:** the `Size() (int, int)` override first, then
    `term.GetSize` (A1.4).
* **`launch/colour.go`:**
  * **`colourEnv(env)`** applies the two rules of §5:
    * `NO_COLOR`, when set and not empty, becomes `NO_COLOR=1`;
    * `FORCE_COLOR`, when set, not empty and neither `0` nor `false`,
      adds `CLICOLOR_FORCE=1` if that is unset.
  * **`profile(terminal bool, env)`** is `NoTTY` when the stream is not
    a terminal, else `colorprofile.Env(colourEnv(env))` (A1.3).
* **`launch/locale.go`:** `tier(env, goos) glyph.Tier`.
  * The first set, non-empty value of `LC_ALL`, `LC_CTYPE`, `LANG`
    decides. A value containing `utf-8` or `utf8`, in any case, is
    `TierUnicode`; anything else is `TierASCII`.
  * With none of them set, the result is `TierUnicode` on `windows` and
    `TierASCII` elsewhere.
  * It never returns `TierLegacy` until theme v2 gives the rule.
* **Sizes (§6):**
  * Interactive: the drawing target's size.
  * Plain: `Out`'s size when `Out` is a terminal, else `COLUMNS` if it is
    a positive integer, else 0.

**The dependency:**

* **`go.mod`:** x/term moves to the direct `require` block in this
  commit, at v0.2.2. The diff of `GOWORK=off go list -m all` before and
  after is recorded, and must be empty.
* **`.golangci.yml`:** a depguard rule `xterm`, modelled on the
  `ultraviolet` rule (`.golangci.yml:77-88`):
  * `files: [$all, "!**/launch/**"]`;
  * deny `github.com/charmbracelet/x/term`;
  * the message "x/term only in launch
    (docs/decisions/0013-MADR-cli-integration-helpers.md A1,
    docs/decisions/0014-MADR-native-integration-api.md W1)".
* **`AGENTS.md`, Dependencies:** the one sentence of 0014-MADR W1.

**Tests:**

* **`TestChoiceText`:** every method, both ways, and an invalid value's
  error.
* **`TestFlagsRegister`:**
  * `-tui`, `-tui=false`, `-mode=plain` and `-mode=bogus` through
    `RegisterFlags` on a `flag.FlagSet`;
  * `Resolve`.
* **`TestFromSource`:** a fake with the three methods.
* **`TestDecideRules`:**
  * each rule, the order with two causes at once, and each `Reason`;
  * `CI=false`, `CI=0`, `CI=true`, and `CI=` (empty);
  * `NoInputEnv` named but unset, and set empty;
  * `TERM=dumb` under each choice;
  * `ChoiceTUI` skipping `NoInputEnv` and `CI`;
  * `UIOnErr`;
  * `OpenTTY` with fake openers that succeed and that fail.
* **`TestDecideOpensOnlyWhenNeeded`:** the opener is called only when
  needed, and each file it returns is closed exactly once.
* **`TestIsTerminalDevNull`:** `os.Open(os.DevNull)` is not a terminal on
  any OS (MADR P7).
* **`TestColourEnv`:**
  * `NO_COLOR` = `1`, `yes`, `x` and empty;
  * `FORCE_COLOR` = `1`, `true`, `0`, `false` and empty;
  * both together;
  * an existing `CLICOLOR_FORCE`;
  * the resulting profile.
* **`TestTierLocale`:**
  * the precedence (`LC_ALL=C` with `LANG=en_US.UTF-8` gives ASCII);
  * lower-case `utf8`;
  * none set, on `linux`, `darwin` and `windows`.
* **`TestDecideSize`:** the `Size()` override; `COLUMNS` = `100`, `0`,
  `-1` and `x`.

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S3-1 | `colourEnv` drops the `NO_COLOR` rule | `TestColourEnv` (`yes`) |
| S3-2 | `FORCE_COLOR=0` forces | `TestColourEnv` |
| S3-3 | `profile` calls `colorprofile.Detect` | `TestColourEnv`'s case with `TERM=xterm-256color`, `COLORTERM` unset, on a terminal stream: `Env` gives ANSI256, while `Detect` on a fake with no descriptor gives NoTTY |
| S3-4 | `CI=false` vetoes | `TestDecideRules` |
| S3-5 | `TERM=dumb` moves inside the `ChoiceTUI` skip | `TestDecideRules` |
| S3-6 | `isTerminal` uses `os.ModeCharDevice` | `TestIsTerminalDevNull` |
| S3-7 | `LANG` read before `LC_ALL` | `TestTierLocale` |
| S3-8 | `Choice.Get` removed (`UnmarshalFlag` likewise) | `TestChoiceText`; Step 7's urfave program (go-flags for `UnmarshalFlag`) fails to compile |
| S3-9 | the opener's files are not closed | `TestDecideOpensOnlyWhenNeeded` |
| S3-10 | a scratch copy imports x/term in `theme`, and in a `workspace` test | `make lint` fails on depguard in both; without the `xterm` rule, neither does |

**Done when** Rule 2's checks are clean, the mutations are killed, and the
module list is unchanged.

### Step 4: `Run`

**API** (§7 and A1.1–A1.4):

* **`launch/option.go`:** an opaque `Option`, with:
  * `WithRegistry(*command.Registry)`;
  * `WithRestorer(Restorer)`;
  * `OnStart(func(*tea.Program))`;
  * `WithProgramOptions(...tea.ProgramOption)`;
  * `WithFilter[M](func(M, tea.Msg) tea.Msg)`.
* **`launch/restore.go`:**
  * `Restorer`;
  * the unexported `saver` seam: `save(v any) (restore func() error, err
    error)`. The real saver uses `term.GetState` on a stream with a
    terminal descriptor, and does nothing otherwise.
* **`launch/errors.go`:**
  * `ErrNotStarted` ("launch: the TUI did not start");
  * `ErrCrashed` ("launch: the TUI stopped").
* **`launch/model.go`:**
  * `state[M]{started bool; last M}`;
  * `wrapped[M]{inner M; st *state[M]}`:
    * `Init` sets `started`, then calls `inner.Init`;
    * `Update` keeps an `M` result as `last`, and wraps it;
    * `View` forwards.
* **`launch/run.go`:** `Run[M](ctx, s, d, m, opts...) (M, error)`.
  1. **A plain decision** returns `m` and `ErrNotStarted` with the
     reason. Nothing is created or written.
  2. **Targets.** `TargetTTY` opens through the same opener seam; a
     failure returns `ErrNotStarted`.
  3. **The saved state.** `saver.save` on each target's stream.
  4. **The program** is built with:
     * `WithContext`, `WithInput`, `WithOutput`;
     * `WithEnvironment(s.Env)`;
     * `WithColorProfile(d.UIProfile)`;
     * `WithoutSignalHandler`;
     * `WithWindowSize` when the drawing stream has no terminal
       descriptor;
     * `WithFilter`'s option, if given;
     * then `WithProgramOptions`'s options.

     The model is `wrapped[M]`. The construction goes through a
     `newProgram` seam that tests replace.
  5. **Before `p.Run()`:**
     * `WithRegistry`'s `Attach(p.Send)`;
     * then each `OnStart(p)`.
  6. **After `p.Run()`, on every outcome, in this order:**
     1. the registry's `detach`;
     2. each `Restorer`'s bytes written to the drawing stream, with write
        errors ignored;
     3. on `ErrNotStarted` and `ErrCrashed`, the saved states put back;
     4. the files `TargetTTY` opened, closed once;
     5. on a clean end, the final model's `Epilogue()`, if it has one,
        written to `Out`.
  7. **The classification:**

     | Outcome | Returns |
     | :--- | :--- |
     | nil error | the unwrapped final model, nil |
     | `!started` | `m` and `ErrNotStarted` wrapping the cause |
     | `tea.ErrInterrupted`, or `ctx.Err() != nil` | the unwrapped final model, or `last` when nil, and the error unchanged |
     | otherwise | `last` and `ErrCrashed` wrapping the cause |

  8. **The final model** that is neither `wrapped[M]` nor `M` returns
     `last` and an error naming its type.

**Tests:**

* **`TestRunPlainWritesNothing`:** `newProgram` is never called, and all
  three writers stay empty.
* **`TestRunQuits`:** the model comes back typed, with a nil error.
* **`TestRunUsesTheGivenInput`:** a key arrives only through `s.In`.
* **`TestRunNotStarted`:** the `newProgram` seam fails without `Init`.
  The result wraps both errors, the restore is called, and the model is
  `m`.
* **`TestRunCrash`:** a panic in each of `Init`, `Update` after one good
  `Update`, `View`, and a command. Each gives:
  * `ErrCrashed` wrapping `tea.ErrProgramPanic`;
  * the last good model;
  * the restore called;
  * the `Restorer` written.
* **`TestRunEndsAreNotCrashes`:** `tea.Quit`, `tea.Interrupt`, a
  cancelled `ctx` and a passed deadline. None gives a sentinel or a
  saved-state restore, and each writes the `Restorer`.
* **`TestRunRegistryDetached`:**
  * a Loop command from an agent goroutine runs on the loop while the TUI
    runs;
  * after `Run` returns, the same command with `context.Background()`
    runs on the caller within 1 s;
  * this holds for a quit, a crash and a cancel.
* **`TestRunRestorerAfterCrash`:** a `termcap.Prober` that has set mode
  2031 is given as the `Restorer`. After a panic in `Update`, the drawing
  stream ends with `ansi.ResetModeLightDark`.
* **`TestRunOnStart`:** called once, with a live program; `Send` from it
  is delivered.
* **`TestRunEpilogue`:** written after the TUI's last byte on a clean end,
  and not after a crash.
* **`TestRunOpensAndClosesTTY`:** fake opener; each file is closed once,
  on success and on each failure.
* **`TestWithFilterSeesTheModel`.**
* **`TestRunSignalsStayTheProgramsUnix`** (Unix only):
  * the test registers its own `SIGINT` notification;
  * it signals itself while `Run` waits;
  * the model quits on a 300 ms timer;
  * the error is nil.

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S4-1 | a plain decision builds the program | `TestRunPlainWritesNothing` |
| S4-2 | `WithInput` left out | `TestRunUsesTheGivenInput` |
| S4-3 | `WithoutSignalHandler` left out | `TestRunSignalsStayTheProgramsUnix` |
| S4-4 | `Init` does not set `started` | `TestRunCrash`, `TestRunNotStarted` |
| S4-5 | a cancelled `ctx` is classified as a crash | `TestRunEndsAreNotCrashes` |
| S4-6 | the crash path returns the final model | `TestRunCrash` (`Update` case) |
| S4-7 | no saved-state restore on a crash | `TestRunCrash` |
| S4-8 | `ErrCrashed` wraps with `%v` | `TestRunCrash` |
| S4-9 | the registry not detached on a crash | `TestRunRegistryDetached` |
| S4-10 | the `Restorer` written only on a clean end | `TestRunRestorerAfterCrash` |
| S4-11 | `WithFilter` passes the wrapper | `TestWithFilterSeesTheModel` |
| S4-12 | the epilogue written after a crash | `TestRunEpilogue` |

**Done when** Rule 2's checks are clean, `-race` included, and the
mutations are killed.

### Step 5: `Frame`, `ExitCode`, `ExitError`, examples

* **`launch/frame.go`:** `Frame(m tea.Model, width, height int, p
  colorprofile.Profile) string`.
  1. `Update` with `tea.ColorProfileMsg{Profile: p}`.
  2. `Update` with `tea.WindowSizeMsg{Width: width, Height: height}`.
  3. `View().Content`.

  It runs no command and no `Init`.
* **`launch/exit.go`:**
  * `ExitError{Code int; Err error}`, with `Error`, `Unwrap` and
    `ExitCode`.
  * `ExitCode(err)` checks, in order:
    1. `errors.As` to `interface{ ExitCode() int }`;
    2. nil gives 0;
    3. `ErrNotStarted` gives 2;
    4. `tea.ErrInterrupted` gives 130;
    5. `context.DeadlineExceeded` gives 124;
    6. `context.Canceled` gives 130;
    7. `tea.ErrProgramPanic` gives 2;
    8. anything else gives 1.
* **`launch/example_test.go`:**
  * `ExampleDecide`, with `// Output:`;
  * `ExampleRun`: the `--tui` pattern with both fallbacks, compiled, not
    run;
  * `ExampleFlags`, with `// Output:`: `RegisterFlags` on a
    `flag.FlagSet`, then `Resolve`.

**Tests:**

* **`TestFrameGolden`:**
  * a three-pane workspace across `tuitest.Golden`, with
    `Matrix{Widths: []int{60, 100}}` at height 12;
  * the colour case is `TrueColor`, the other `ASCII`;
  * glyphs from `glyph.For(c.UTF8)` through `workspace.WithThemeBuilder`;
  * written with `-tuitest.update`, and read before commit.
* **`TestFrameRunsNoCommand`.**
* **`TestExitCode`:**
  * each row, bare and wrapped;
  * an `ExitError` inside `ErrCrashed`, where the `ExitError` wins;
  * a custom type with `ExitCode()`.

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S5-1 | `Frame` skips the profile message | `TestFrameGolden` (no-colour cases) |
| S5-2 | `Frame` runs `Init`'s command | `TestFrameRunsNoCommand` |
| S5-3 | `ExitCode` checks the interface last | `TestExitCode` (`ExitError` inside `ErrCrashed`) |
| S5-4 | `context.Canceled` maps to 1 | `TestExitCode` |

**Done when** Rule 2's checks are clean, the goldens are read, `go doc
-short ./launch` lists exactly A1.2's API with A1.6's names, and the
mutations are killed.

### Step 6: `launch/launchtest`

```go
package launchtest

// Terminal is a fake terminal stream: a reader and a writer that say they
// are a terminal, with a size.
type Terminal struct{ /* unexported */ }

func NewTerminal(width, height int) *Terminal
func (t *Terminal) Read(p []byte) (int, error)
func (t *Terminal) Write(p []byte) (int, error)
func (t *Terminal) IsTerminal() bool
func (t *Terminal) Size() (int, int)
func (t *Terminal) Type(keys string)   // queue keystrokes as raw bytes
func (t *Terminal) Close() error
func (t *Terminal) Output() string     // everything written so far

// Pipe is a stream that is not a terminal.
type Pipe struct{ /* unexported */ }

func NewPipe(input string) *Pipe

func Streams(in io.Reader, out, err io.Writer, env ...string) launch.Streams
func Env(kv ...string) termcap.Env
```

* Steps 3–5's tests move onto `launchtest` in this step, which is the
  first consumer test.
* **`ExampleTerminal`:** a CLI's `--tui` path tested end to end with a
  `Run` that quits on a typed `q`.
* **Mutation S6-1:** `Terminal.IsTerminal` returns false. `ExampleTerminal`'s
  output must differ, because the decision becomes plain.

**Done when** Rule 2's checks are clean, the moved tests pass, and S6-1 is
killed.

### Step 7: the tier-1 framework programs and the live probes (scratch)

**The framework programs.** One per tier-1 framework, in 0014 W0's
framework-example harness. Each has a `tui` path through `Decide` and
`Run` with both fallbacks, and an exit through `ExitCode`:

| Framework | `--tui` and `--mode` through | Streams |
| :--- | :--- | :--- |
| stdlib `flag` | `Flags.RegisterFlags` | `Streams{os.Stdin, os.Stdout, os.Stderr, os.Environ()}` in `main` |
| cobra | `RegisterFlags` on a `flag.FlagSet`, then `cmd.Flags().AddGoFlagSet` | `launch.FromSource(cmd, env)` |
| kong | `Flags` embedded (`embed:""`); `TUI` with `negatable:""` | a `Streams` literal from `kong.Writers` and `os.Stdin` |
| urfave/cli v3 | `&cli.GenericFlag{Name: "mode", Value: &f.Mode}`; `&cli.BoolFlag{Name: "tui"}` | a `Streams` literal from `cmd.Reader`, `cmd.Writer`, `cmd.ErrWriter` |

Each program is run with:

* `--tui`, `--mode=plain`, `--mode=bogus` and no flag, with no terminal;
* exit status 2 when a refusing variant is selected;
* kong's `FatalIfErrorf` returning an `ExitError{Code: 3}`, which exits 3.

The tier-2 programs (ff v4, go-flags, go-arg) run once and are recorded,
not gated.

**The live probes:**

* The scratch module requires the tree through a `replace`.
* Before `Run` and after a fallback, the probe prints whether
  `term.GetState` matches byte for byte.
* On macOS it runs under `script -q /dev/null`. On the Windows test host
  it runs under `ssh -tt`.

| Case | Run | Expect |
| :--- | :--- | :--- |
| L1 | `--tui`, `</dev/null \| cat` | one `Warning:` line naming the reason, then the CLI mode; exit 0; no escape byte on stdout; under 1 s |
| L2 | `--tui`, `TERM=dumb`, under a terminal | the warning names `launch.dumb-terminal`, then the CLI mode |
| L3 | `--tui`, `q` typed | the TUI at `Decide`'s size; no warning; exit 0 |
| L4 | `--tui`, a prober attached, the model panicking after its first `Update` | Bubble Tea's crash report; the warning; the CLI mode with the last good model's value; state the same; no mode 2031 reports arriving; `stty -a` with `icanon` and `echo` (macOS) |
| L5 | `--tui`, Bubble Tea patched in a scratch copy to fail in `initInputReader` | the `ErrNotStarted` warning, the CLI mode, state the same |
| L6 | `--tui`, stdout piped, `UIOnErr` | the UI on stderr; only the result on stdout |
| L7 | `--tui`, `OpenTTY`, stdin piped | the TUI on the terminal, keys from it |
| L8 | `--tui`, an agent goroutine `Run`s a workspace Loop command after the TUI quits | it returns within 1 s, run once |

* **On Windows: L1, L3, L4, L6, L7 and L8.** After L7, a line typed into
  the CLI mode must reach it, because `CONIN$` gets the reader that
  cannot be cancelled.
* **If it does not,** the step stops (Rule 4). The resolutions offered
  are:
  * `OpenTTY` unsupported on Windows, recorded in the MADR;
  * a fix that the record names.

**Done when** every gated program and case gives what is expected, and
the tier-2 programs and L7 on Windows are recorded.

### Step 8: conformance, collision allowlist, documents

* **`internal/conformance`:**
  * `launch` is scanned, which 0014 W0 makes automatic.
  * The collision allowlist gains "`Decision`: `command` (renamed
    `Verdict` by 0014 W4; old name removed in `v0.10.0`), `launch`".
* **Mutations:**
  * **S8-1:** `launch` names `os.Stderr`;
  * **S8-2:** `launch` sets `AltScreen`;
  * **S8-3:** `launch` imports `os/exec`.

  Each must fail the conformance test.
* **`docs/guides/commands.md`:** a section "Start the TUI from your CLI",
  after "Run commands from your own CLI". It holds:
  * the four tier-1 programs' essentials;
  * the fallback pattern;
  * `Decide`'s rules;
  * `UIOnErr` and `OpenTTY`;
  * `WithRegistry`, `WithRestorer` and `OnStart`;
  * `Frame` and `ExitCode`;
  * `launchtest`;
  * tier 2 in a short table;
  * three cautions: pflag's `NoOptDefVal`, cobra usage-on-error going to
    Out, and keeping streams unwrapped (0014-REPORT §3.3).

  Its examples join the framework-example harness.
* **`docs/guides/building-workspaces.md`,** "The program owns the
  program": a paragraph pointing to that section.
* **`docs/guides/terminal-capabilities.md`,** "Do not probe without
  input": `launch.Run` always sets an input, and a prober is passed
  through `WithRestorer`.
* **`README.md`:**
  * the current release is `v0.7.0`;
  * a Status bullet: "Starting the TUI from any Go CLI, since `v0.7.0`";
  * an "I want to…" row.
* **`docs/README.md`:** "I want to…" rows for the guide section and
  `launchtest`.
* **`docs/architecture.md`:**
  * the graph line: `launch → command, glyph, termcap, internal/enum;
    bubbletea, colorprofile, x/term`;
  * the table rows for `launch` and `launch/launchtest`;
  * `internal/enum` in the tree;
  * Dependencies: x/term kept to one package;
  * "What is not here": the sentence about helpers goes.

**Done when** Rule 2's checks are clean, the mutations are killed, and the
guide's examples compile and run in the harness.

### Step 9: the release

1. The owner commits, has the agent run the disclosure guard, pushes, and
   with CI green tags `v0.7.0` (annotated, `-m "v0.7.0"`).
2. The agent runs the consumer smoke test (`docs/guides/releasing.md`)
   against `v0.7.0`:
   * a program importing `launch`, `launch/launchtest` and `workspace`;
   * it runs `Decide` with no terminal, and exits through `ExitCode`;
   * `go list -m all` shows x/term at v0.2.2.
3. **Release notes** in the execution record:
   * the additions: `launch`, `launchtest`, `glyph.Tier`,
     `Registry.Attach`, `internal/enum`;
   * no removal;
   * `make apicheck`'s report.

**Done when** the tag exists and the smoke test runs.

### Step 10: close-out

* **Verification,** item by item, with output.
* **This PLAN `complete`.**
* **An amendment** to either MADR only if a fact differed.

## Verification

* **`Choice` and `Flags`** parse `--tui` and `--mode` in the four tier-1
  frameworks (Step 7). The tier-2 results are recorded.
* **`Decide`** follows §3–§6 as amended. It spawns nothing (S3-3), and
  S3-1 to S3-10 are killed.
* **`Run`:**
  * holds the fallback in every outcome;
  * detaches the registry (I1, L8) and writes the `Restorer` (I2, L4);
  * restores the saved state;
  * returns the last good model;
  * S4-1 to S4-12 are killed.
* **`Frame`, `ExitCode` and `ExitError`** follow §8, §9 and A1.2. S5-1 to
  S5-4 are killed.
* **`launchtest`** carries the package's own tests (S6-1).
* **Conformance** scans `launch` (S8-1 to S8-3).
* **The live probes L1–L8** pass on macOS, and the Windows cases pass on
  the test host.
* **x/term** is direct at v0.2.2, with an unchanged module list, and kept
  to `launch` by depguard.
* **`make apicheck`** shows additions only.
* **`v0.7.0`** is tagged, and the smoke test runs.
* **Rule 2's checks** are clean at every step, and the identifier scan
  finds nothing.

## Rollout and Rollback

* **Rollout:**
  * Steps 2 to 8 land one commit each, and `v0.7.0` is tagged after
    Step 7's probes.
  * No existing behaviour changes: `WithLoop` keeps its meaning, and
    `glyph.For` stays.
* **Rollback:**
  * **Before a push,** each step is one commit to revert. Step 3's revert
    returns x/term to `// indirect`.
  * **After `v0.7.0`,** a defect is fixed in `v0.7.1`. If `v0.7.0` must
    not be used, `retract v0.7.0` goes in that release's `go.mod`, with
    the reason. A tag is never moved or deleted.

## Execution Record

Not started.
