---
status: complete
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
* **2026-10-08, Step 11 added:** drop `negatable:""` from `Flags.TUI`,
  as 0013-MADR A1.10 decides. Steps 1–10 are complete. The owner approved
  Step 11 the same day, and chose `v0.7.1` for its release, so the status
  is `in-progress` again.

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

### Step 11 (2026-10-08): drop Kong's `--no-tui`

0013-MADR A1.10. One commit, released as `v0.7.1`.

* **`launch/flags.go`:**
  * `Flags.TUI`'s tag loses `negatable:""`;
  * the doc comment's clause "its negatable tag gives Kong --no-tui,
    which no other framework reads" goes.
* **`testdata/frameworks/kong/main.go`:** the comment on the embedded
  `launch.Flags` becomes `// --mode and --tui`.
* **`docs/guides/commands.md`:**
  * the Kong excerpt follows the program, which the excerpt check
    enforces;
  * the sentence "With Kong, `launch.Flags` embeds into the grammar
    (above), with `--no-tui`, …" loses "with `--no-tui`", and gains
    "`--tui=false` cancels an earlier `--tui`".
* **`testdata/frameworks/cases.txt`** gains two Kong cases:
  * `kong --no-tui => 80 stderr unknown flag --no-tui`;
  * `kong --tui --tui=false => 0 stdout line mode`.
* **The records:**
  * 0013-MADR's A1.10 becomes accepted;
  * this PLAN's execution record for Step 11;
  * `docs/README.md`'s 0013 rows.

  The history stays as written: A1.2, A1.8, D8, and Steps 7 and 9's
  records. A1.10 supersedes them.

**Mutation:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S11-1 | the tag put back | `make examples`, on the case `kong --no-tui => 80` |

**Checks:**

* Rule 2, and the Windows test host;
* `make apicheck` against `v0.7.0`: 0 incompatible changes;
* `make examples`: 26 cases.

**Release:** `v0.7.1`, the owner's choice. The owner asked the agent to
create the tag. It is an annotated tag on Step 11's commit. It is pushed
after `main`, once CI passes there, and then the consumer smoke test runs
against it.

**Done when** Rule 2's checks are clean, S11-1 is killed, and the
excerpts and cases pass.

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

### Step 1: records

The owner approved this PLAN on 2026-10-07 ("plan approved. proceed."),
after [0014-PLAN-api-policy-gates.md](0014-PLAN-api-policy-gates.md) was
completed.

* **The preconditions hold:**
  * 0011 is complete (`5c6f4bc`);
  * 0014's W0 PLAN is complete. CI run `37704939096` passed on its last
    code commit, `6663705`, and `28357ef` marks it `complete`.
* **The facts table, re-read on `28357ef`:** each fact still holds. W0's
  Step 6 added a three-line stability paragraph before each `package`
  clause, so the line numbers in files with a package comment moved down
  by three. Two of the cited spans moved:
  * termcap's generics are now at `termcap/termcap.go:127-147`;
  * `command.Mode`'s constants are at `command/command.go:204-207`.

  `command/registry.go:82-88`, `command/dispatch.go`'s `onLoop` (line 68)
  and `go.mod:16` (x/term v0.2.2, `// indirect`) are unchanged. Later
  steps cite lines as they read them.
* **`docs/README.md`:** this PLAN's row now carries the PLAN's title, and
  `in-progress`.

### Step 2: prerequisites from 0014-MADR W2

The owner approved it on 2026-10-07 ("committed, proceed"), after Step 1
was committed as `0bfc24b`. No deviation.

#### What was built

* **2a, `internal/enum`:** `Name`, `Marshal` and `Unmarshal`, with the
  PLAN's signatures. `Name` formats an unnamed value with `strconv.Itoa`,
  which gives the same text as termcap's `fmt.Sprintf("%d", v)`.
  * termcap's `name`, `marshal` and `unmarshal` stay as unexported
    one-line calls into it, so their callers are unchanged. termcap's
    tests and golden files pass unchanged.
  * The package comment ends with `Stability: internal.`, as W0's
    conformance test requires.
* **2b, `glyph.Tier`:** `TierUnicode` (the zero value), `TierLegacy` and
  `TierASCII`, with `Set`, `String`, `MarshalText` and `UnmarshalText`
  through `internal/enum`. The tokens are "unicode", "legacy" and
  "ascii", and the errors read "glyph: …".
  * `TierLegacy.Set()` is `ASCII()` until theme v2 fills it.
  * A tier with no name also gives `ASCII()`, which every terminal draws.
  * `For` is now `TierUnicode.Set()` or `TierASCII.Set()`, with the same
    result as before.
* **2c, `Registry.Attach`:**
  * The registry's loop is an `atomic.Pointer` to `{send, done}`. Run
    reads it without a lock, and `mu` serialises `Attach` and `detach`.
    `WithLoop` stores a loop whose `done` is nil, so it is never detached.
  * `detach` runs once. It clears the loop only if it is still the
    attached one, then closes `done`.
  * `onLoop` follows the PLAN's three-way select with an `atomic.Bool`
    claim. The loop's run also keeps today's check: a message it claims
    after `ctx` has ended runs nothing, and reports `ctx.Err()`.
  * **The panic path.** A `defer` recovers the panic, sends an outcome
    whose error is "command: <id> panicked on the loop: <value>", and
    panics again with the same value. Bubble Tea still recovers the
    panic as before, and the caller is released.
  * `Attach`'s documentation says to detach only after the program's
    `Run` has returned. It also says that a command run on the caller
    after detach keeps its `Result.Cmd`, which no program will run.
* **Tests:**
  * `command/loop_test.go` holds the PLAN's five loop tests, plus
    `TestWithLoopUnchanged`. That test checks that `WithLoop`'s loop is
    used on every run, and that `Attach` then `detach` leaves the
    registry with no loop. `TestRunOnLoop` and `TestDispatchLoop` pass
    unchanged.
  * `TestEnumNames` is in `internal/enum`. `TestTierWidths` and
    `TestTierText` are in `glyph`.

#### Checks

* **Mutations, each on a scratch clone,** all killed:
  * S2-1 (`detach` does not close `done`): "Run still waits on a
    detached loop";
  * S2-2 (the claim dropped): "iteration 98: the handler ran 2 times,
    want exactly 1";
  * S2-3 (`TierLegacy` gives `Unicode()`): "legacy.Set() is not the
    table the tier names";
  * S2-4 (case-folded names accepted): `Unmarshal("TMUX"): <nil>, want
    "termcap: unknown Mux \"TMUX\""`;
  * S2-5 (`ctx` wins without the claim): "Run = {…}, context canceled;
    want the handler's outcome", and the audit recorded the
    cancellation;
  * S2-6 (no `defer`): "a panic on the loop left Run waiting".
* **The API:** apidiff, run over all changes rather than only
  incompatible ones, between `v0.6.0` and the tree, lists only compatible
  changes:
  * `./command.(*Registry).Attach: added`;
  * `./glyph.Tier: added`, with its methods;
  * `TierASCII`, `TierLegacy` and `TierUnicode: added`.

  `make apicheck` (in `make release-check`) is clean.
* **Rule 2:**
  * `make pre-add-check` over the 8 changed Go files: clean;
  * `make release-check`: "128 file(s) clean … apicheck, examples";
  * `make lint`: three "0 issues.";
  * `make vuln`: "No vulnerabilities found.";
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 16 `ok` (`internal/enum` is new), and so did workspace
    mode;
  * `go mod tidy -diff` and `scripts/go-modules.sh --check`: clean.
* **The Windows test host,** go1.27.1 windows/amd64:
  * `make pre-add-check`, `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 16 `ok`;
  * the nine Step 2 tests pass.
* **The identifier scan of the diff and the new files:** no match.

### Step 3: the flag types, the streams and `Decide`

The owner approved it on 2026-10-07 ("proceed"), after Step 2 was
committed as `5b28e83`.

#### Deviations

Both were found before anything was written. Each was asked, and the owner
chose the recommendation.

* **D1 (2026-10-07): `Decision`'s collision entry lands in this step, not
  Step 8.**
  * **Found:** this step creates `launch.Decision`.
    `TestNoTypeNameMeansTwoThings` (W0 Step 3) fails as soon as two public
    packages export one name, unless `sharedNames` allows it. The PLAN
    adds the entry in Step 8, so Step 3's checks would fail. A scan of
    the public packages' exported types found no other clash: `Choice`,
    `Flags`, `StreamSource`, `Streams`, `Config`, `Target`, `Reason`,
    `Restorer` and `ExitError` are free, and `Option` is allowed
    everywhere.
  * **Chosen:** `sharedNames` gains `"Decision": {"command", "launch"}`
    in this step, with a comment that 0014's W4 removes `command`'s in
    `v0.10.0`. `docs/glossary.md` moves `Decision` from "Reserved" to
    "Deliberate" in the same change, as its rule requires. Step 8's
    collision allowlist item is then already done.
  * **Not chosen:** renaming `launch`'s type, which would amend 0013-MADR
    A1.6.
* **D2 (2026-10-07): `AGENTS.md`'s x/term sentence comes from 0013-MADR
  §11.**
  * **Found:** the PLAN says "the one sentence of 0014-MADR W1". W1 has no
    dependency sentence. 0013-MADR §11 says that "AGENTS.md's
    Dependencies section names the package and points here".
  * **Chosen:** one sentence from §11: x/term is a direct requirement at
    the version Bubble Tea selects, only `launch` may import it, and the
    depguard rule `xterm` refuses it everywhere else.
  * **Not chosen:** leaving `AGENTS.md` unchanged.
* **D3 (2026-10-07): the `FORCE_COLOR` cases are exact per OS, and the
  MADR's fact about `colorprofile.Env` is corrected (0013-MADR A1.7).**
  * **Found:** on the Windows test host, `TestColourEnv` failed with
    "FORCE_COLOR=1: profile = TrueColor, want ANSI", and the same for
    `FORCE_COLOR=true`. `make pre-add-check` and the shuffled tests
    failed with it. Every other `launch` test passed there, and all
    passed on macOS. The fault was new in this step.
  * **The cause:** with `TERM` unset, empty or `dumb`, colorprofile's
    `Env` asks Windows for its build number (`colorprofile@v0.4.3/env_windows.go:17`) and
    gives that console's profile, which `CLICOLOR_FORCE` keeps. A1.3 and
    the facts table say `Env` "reads the environment only". It opens no
    file and starts no process, so the no-spawn rule holds.
  * **Asked,** with two options: exact per OS with `ConEmuANSI=ON`, which
    colorprofile answers from the environment (`colorprofile@v0.4.3/env_windows.go:13`); or
    "at least ANSI".
  * **The owner chose** the first, the recommendation.
    * Those cases set `ConEmuANSI=ON`, and expect TrueColor on Windows
      and ANSI elsewhere.
    * The MADR gains A1.7.
    * `launch`'s two comments that said the profile comes from "the
      environment alone" now name the Windows lookup.

#### What was built

* **`launch/`:** `doc.go`, `choice.go`, `flags.go`, `streams.go`,
  `decide.go`, `terminal.go`, `colour.go` and `locale.go`, as the PLAN
  lists them, with A1.2's and A1.6's names. Choices the PLAN left open:
  * **`Target` has `String`, `MarshalText` and `UnmarshalText`**
    ("none", "stream", "err", "tty"), through `internal/enum`. This
    follows `AGENTS.md`'s API conventions, rule 5. Each `Reason`
    constant is its token.
  * **A plain decision's `UIProfile` is `NoTTY`,** since there is no TUI
    stream, and its `In` and `UI` are `TargetNone`.
  * **The controlling terminal is probed:** opened, measured, and closed
    at once, each distinct file once, the first time a target needs it.
    One that does not open, or does not close cleanly, is not used.
    errcheck refuses a discarded `Close` error, and `Decide` returns no
    error, so a close that fails makes the terminal unusable. That is
    the conservative reading.
  * **The PLAN's unexported `decide` is `decideWith`.** revive's
    `confusing-naming` refuses two functions in one file that differ only
    in case.
  * **`CI`, `FORCE_COLOR`:** "false" and "0" are compared exactly, as the
    MADR writes them.
  * **`doc.go`** ends with the full stability line, which W0's
    conformance test requires. The PLAN's "`Stability: stable.`" is its
    first sentence.
* **`go.mod`:** x/term v0.2.2 is in the direct `require` block. The
  output of `GOWORK=off go list -m all` is the same, 29 lines, before and
  after, and `go mod tidy -diff` is clean.
* **`.golangci.yml`:** the `xterm` rule, as the PLAN gives it.
* **`AGENTS.md`, Dependencies:** D2's sentence.
* **D1:** `sharedNames` has `Decision`, and `docs/glossary.md` moves
  `Decision` to "Deliberate". The glossary's other reserved `launch`
  names are now partly built. Step 8, which edits the documents, moves
  them once Steps 4 to 6 have added the rest.
* **Tests:** `launch/flags_test.go` and `launch/decide_test.go`.
  * They hold the PLAN's nine tests, and `TestTargetText` and
    `TestDecideGlyphs`. The latter checks that the operating system
    reaches the tier through `decideWith`.
  * The fakes are package-private, in `launch/fake_test.go`, until Step 6.
  * Every environment is explicit. The only file opened is
    `os.DevNull`.
  * `TestDecideOpensOnlyWhenNeeded` also covers a terminal that does not
    close cleanly.

#### Checks

* **Mutations, on scratch copies.** All killed. They were run again
  after D3's test change:
  * S3-1 (`NO_COLOR` rule dropped): "NO_COLOR=yes: profile = ANSI256,
    want Ascii";
  * S3-2 (`FORCE_COLOR=0` forces): "FORCE_COLOR=0: profile = ANSI, want
    NoTTY";
  * S3-3 (`profile` calls `Detect` on a writer with no descriptor): "no
    NO_COLOR: profile = NoTTY, want ANSI256";
  * S3-4 (`CI=false` vetoes): `TestDecideRules/CI=false`, "reason
    launch.ci";
  * S3-5 (`TERM=dumb` inside the `ChoiceTUI` skip): `TestDecideRules/
    TERM=dumb,_tui`, "reason launch.terminal";
  * S3-6 (the file mode asked): "/dev/null is a terminal";
  * S3-7 (`LANG` first): `tier(["LC_ALL=C" "LANG=en_US.UTF-8"], linux) =
    unicode, want ascii`;
  * S3-8 (`Get` removed) and S3-8b (`UnmarshalFlag` removed): the tests do
    not compile, "*Choice does not implement flag.Getter (missing method
    Get)" and the go-flags interface. Step 7's programs are the PLAN's
    second check;
  * S3-9 (files not closed): "the one file was closed 0 times, want 1".
    Its first form did not compile, because the mutation left `in`
    unused. It was rewritten to compile, and run again;
  * S3-10: `make lint` fails with depguard's "x/term only in launch" on
    both `theme/xterm_mut.go` and `workspace/xterm_mut_test.go`. With the
    `xterm` rule removed, `make lint` passes on the same copy.
* **The API:** apidiff, run over all changes, lists only additions since
  `v0.6.0`: Step 2's, and `package …/launch: added`. `make apicheck` is
  clean.
* **Rule 2 on macOS,** on the final tree:
  * `make pre-add-check` over the 12 changed Go files: clean;
  * `make release-check`: "131 file(s) clean … apicheck, examples";
  * `make lint` (three "0 issues.") and `make vuln` are clean;
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 17 `ok`, and so did workspace mode;
  * `go mod tidy -diff` and `scripts/go-modules.sh --check` are clean;
  * markdownlint over the repository: 0 issues. The link check over
    `AGENTS.md`, the glossary and this PLAN: 0 broken.
* **The Windows test host,** go1.27.1 windows/amd64, on the final tree:
  * the first run failed in `TestColourEnv` (D3); the rest of `launch`
    passed;
  * after D3, `make pre-add-check` gave "131 file(s) clean … apicheck,
    examples";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 17 `ok`;
  * `launch`'s 11 tests pass, `TestColourEnv` among them.
* **The identifier scan of the diff and the new files:** no match.

### Step 4: `Run`

The owner approved it on 2026-10-07 ("Commit to main then proceed"), when
Step 3 was committed as `cb116d0`.

#### Deviations

* **D4 (2026-10-07): a failed reset or restore is joined into `Run`'s
  error, not ignored.**
  * **Found, before anything was written:** the PLAN writes the
    `Restorer`'s bytes "with write errors ignored". This repository's
    errcheck runs with `check-blank: true` (`.golangci.yml`), so
    `_, _ = io.WriteString(…)` fails lint, and no `//nolint` directive
    exists in the repository's own code. The PLAN does not say what
    happens to an error putting a saved terminal state back.
  * **Asked,** with two options: join the error into `Run`'s, or keep
    "ignored" behind the repository's first `//nolint:errcheck`.
  * **The owner chose** the first, the recommendation.
    * A failed write of a `Restorer`'s bytes, a failed restore of a
      saved state, a failed close of the controlling terminal, and a
      failed epilogue write are each joined onto what `Run` returns with
      `errors.Join`. `errors.Is` still finds `ErrNotStarted`,
      `ErrCrashed`, `tea.ErrInterrupted` and the rest.
    * When nothing failed, `Run` returns its error as it was, so a
      program's own type assertion sees it unwrapped (A1.2).
    * On a clean end, a reset that did not land makes `Run` return the
      final model with that error alone. The terminal may still be in
      mode 2031, and `ExitCode` gives 1.

#### What was built

* **`launch/`:** `errors.go`, `restore.go`, `option.go`, `model.go` and
  `run.go`, as the PLAN lists them. Choices the PLAN left open:
  * **The seams.** `newProgram` is a package variable (`tea.NewProgram`).
    `save` is a package variable (`saveState`). The controlling terminal
    comes through the same `opener` as `Decide`, passed to an unexported
    `runWith`.
  * **A save that fails** returns `ErrNotStarted` before a program is
    built. It closes the terminal Run opened.
  * **A decision marked interactive with no input or no drawing target**
    returns `ErrNotStarted`, naming both targets.
  * **A nil `ctx`** is `context.Background()`.
  * **Several `WithRestorer`s and `OnStart`s** are kept, and run in
    order.
  * **`WithFilter[M]`** passes a message through unchanged when the model
    is not an `M`.
  * **The saved states** are saved once each when In and Out are one
    stream.
  * **The epilogue** is written only when `Out` is set.
* **Tests:** `launch/run_test.go` has the PLAN's 12 tests and one more,
  `TestRunFinalModelOfAnotherType`, for item 8 of `Run`'s steps.
  `launch/run_unix_test.go` holds `TestRunSignalsStayTheProgramsUnix`,
  with a `unix` build tag.
  * The streams are the package's fakes and `io.Pipe` readers. Each run
    has a five-second guard.
  * **What reaches the test process's standard error.** In the crash
    cases, Bubble Tea writes its own crash report ("Caught panic" and the
    stack) to the process's standard error. That is MADR §10's documented
    behaviour, which launch cannot move. It happens 8 times in a
    `go test -v ./launch` run. No test reads or writes a standard stream
    itself.

#### Checks

* **Mutations, on scratch copies,** each test process with stdin from
  `/dev/null` and in a new session, so no mutant could reach a real
  terminal. All killed:
  * S4-1 (a plain decision builds): "a plain decision built a program";
  * S4-2 (no `WithInput`): Bubble Tea fell back to the controlling
    terminal, which did not open: "launch: the TUI did not start:
    bubbletea: error opening TTY";
  * S4-3 (no `WithoutSignalHandler`): "Run = program was killed: program
    was interrupted; want the model's own quit, nil";
  * S4-4 (`Init` does not set `started`): `TestRunCrash/Init`, "err =
    launch: the TUI did not start: … want ErrCrashed".
    `TestRunNotStarted` passes under this mutation, and cannot fail: its
    program fails before `Init`, so `started` is false either way. The
    PLAN named it in error;
  * S4-5 (a cancelled `ctx` is a crash): "carries a sentinel", and
    "restored 2 saved states on an end the program chose";
  * S4-6 (the crash returns the final model): `TestRunCrash/Update,
    after a good Update`, "the model's keys are \"\", want … \"a\"";
  * S4-7 (no restore on a crash): "restored 0 saved states, want 2";
  * S4-8 (`%v`): "want ErrCrashed wrapping tea.ErrProgramPanic";
  * S4-9 (no detach on a crash): `TestRunRegistryDetached/crash`, "did
    not return within 1s";
  * S4-10 (the `Restorer` only on a clean end): "the drawing stream does
    not end with mode 2031's reset";
  * S4-11 (`WithFilter` gets the wrapper): "want x filtered out", and
    "the filter never saw the program's own model";
  * S4-12 (the epilogue after a crash): "after a crash: … Out holds
    \"…resume\"".
* **The API:** apidiff over all changes lists only additions since
  `v0.6.0`. `make apicheck` is clean.
* **Rule 2 on macOS:**
  * `make pre-add-check` over the 8 changed Go files: clean;
  * `make release-check`: "142 file(s) clean … apicheck, examples";
  * `make lint`: three "0 issues.";
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 17 `ok`, and so did workspace mode. The `Run` tests also
    passed 20 times under `-race`, with no race reported;
  * `go mod tidy -diff` and `scripts/go-modules.sh --check`: clean.
* **The Windows test host,** go1.27.1 windows/amd64:
  * `make pre-add-check` gave "142 file(s) clean … apicheck, examples";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 17 `ok`;
  * `launch`'s 24 tests pass. That is all but the Unix-only signal
    test.
* **The identifier scan of the diff and the new files:** no match.

### Step 5: `Frame`, `ExitCode`, `ExitError`, examples

The owner approved it on 2026-10-07 ("Commit to main and proceed"), when
Step 4 was committed as `0b7a9db`. No deviation.

#### What was built

* **`launch/frame.go`:** `Frame`, as the PLAN gives it.
* **`launch/exit.go`:**
  * `ExitError`, with pointer receivers as A1.2 writes them. Its `Error`
    is `Err`'s text, or "exit status <Code>" when `Err` is nil.
  * `ExitCode`, in the PLAN's order. It finds the interface with
    `errors.AsType[interface{ error; ExitCode() int }]`, since `AsType`
    takes an error type and every link of a chain is one.
* **`launch/example_test.go`:**
  * `ExampleDecide` shows the three choices on streams that are not
    terminals, with `CI=true`;
  * `ExampleFlags`;
  * `ExampleRun`, the `--tui` pattern with both fallbacks, compiled and
    not run.
* **Tests:** `launch/frame_test.go` has `TestFrameGolden`,
  `TestFrameRunsNoCommand`, `TestExitCode`, and `TestExitError`, which
  covers `ExitError`'s three methods.
  * **The golden's tree.** The first `TestFrameGolden` used
    `layout.SidebarRightBottom`. At height 12 that preset's breakpoints
    hide the bottom pane, and at width 60 the sidebar too, so the 60-cell
    frame showed one pane. The test now builds a fixed tree: a main pane
    and an 18-cell sidebar over a 3-row bottom pane. Every case shows the
    three panes. Reading the first goldens showed the problem. They were
    deleted before any commit, and written again.
  * **The 8 goldens were read:**
    * each has 12 rows at its case's width;
    * the colour cases carry TrueColor escapes;
    * the no-colour cases carry only bold, which marks the focused pane
      (MADR §5: ASCII keeps bold);
    * the ASCII cases use `+`, `-`, `|` and `>`.

#### Checks

* **Mutations, on scratch copies.** All killed:
  * S5-1 (no profile message): exactly the four no-colour cases differ,
    "frame-workspace nocolor.utf8.60: line 1 differs";
  * S5-2 (`Init`'s command run): "Frame called Init (true) or ran a
    command (true)";
  * S5-3 (the interface checked last): "an ExitError inside a crash: 2,
    want 7";
  * S5-4 (`context.Canceled` gives 1): "context.Canceled: ExitCode = 1,
    want 130".
* **`go doc -short ./launch`** lists exactly A1.2's API with A1.6's
  names:
  * `ErrNotStarted`, `ErrCrashed`, `ExitCode`, `Frame` and `Run`;
  * `Choice`, `Config`, `Decision` and `Decide`;
  * `ExitError`, `Flags`, `Option` and its five constructors;
  * `Reason`, `Restorer`, `StreamSource`, `Streams` and `FromSource`;
  * `Target`.

  apidiff over all changes lists only additions since `v0.6.0`, and
  `make apicheck` is clean.
* **Rule 2 on macOS:**
  * `make pre-add-check` over the 4 new Go files: clean;
  * `make release-check`: "149 file(s) clean … apicheck, examples";
  * `make lint`: three "0 issues.";
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 17 `ok`, and so did workspace mode;
  * `go mod tidy -diff` and `scripts/go-modules.sh --check`: clean.
* **The Windows test host,** go1.27.1 windows/amd64:
  * `make pre-add-check` gave "149 file(s) clean … apicheck, examples";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 17 `ok`;
  * `launch`'s 30 tests and examples pass, the goldens among them.
* **The identifier scan of the diff and the new files:** no match.

### Step 6: `launch/launchtest`

The owner approved it on 2026-10-07 ("Commit and proceed"), when Step 5
was committed as `d6b17bd`.

#### Deviations

Both were found before anything was written. Each was asked, and the owner
chose the recommendation.

* **D5 (2026-10-07): `Terminal` is a deliberate clash with
  `termcaptest.Terminal`.**
  * **Found:** `termcap/termcaptest` exports `Terminal`, "a scripted fake
    terminal speaking a Profile". The PLAN's `launchtest.Terminal` would
    fail `TestNoTypeNameMeansTwoThings`.
  * **Chosen:** both mean a fake terminal for tests, each in its own
    test-kit package with its own constructor. `sharedNames` gains
    `"Terminal": {"launch/launchtest", "termcap/termcaptest"}`, and
    `docs/glossary.md` gains its row in the same change.
  * **Not chosen:** renaming `launchtest`'s, which changes the PLAN's API.
* **D6 (2026-10-07): the tests that use only `launch`'s exported API move
  onto `launchtest`; the others stay internal.**
  * **Found:** `launchtest` imports `launch`, since `Streams` returns a
    `launch.Streams`. Steps 3–5's tests are in package `launch`, and Go
    refuses that import cycle in a test. They are internal because they
    use the seams: `decideWith`'s OS and opener, `runWith`, `save`,
    `newProgram`, `colourEnv`, `tier`, `isTerminal` and `size`. Only a
    `launch_test` file can import `launchtest`.
  * **Chosen:** each test that uses only the exported API moves to
    `launch/consumer_test.go`, in package `launch_test`, on `launchtest`.
    That is `launchtest`'s first consumer. The rest stay on the
    package-private fake.
  * **Not chosen:** leaving every test in place, with `launchtest`
    exercised only by its own tests and `ExampleTerminal`.

#### What was built

* **`launch/launchtest/launchtest.go`:** the PLAN's API.
  * **`Terminal`:**
    * `Read` waits, as a terminal does, until `Type` queues keys or
      `Close` is called. After `Close` it returns the queued keys, then
      `io.EOF`.
    * `Write` keeps what is written, for `Output`.
    * A second `Close` does nothing.
    * It is safe for use from several goroutines.
  * **`Pipe`** has `Read`, `Write` and `Output`, which the PLAN's sketch
    left out, so that it can stand in for In, Out or Err. It has no
    `IsTerminal`, so launch's terminal test says no.
  * The package comment ends with the stable stability line.
* **`launch/launchtest/launchtest_test.go`:** the reader waits for keys;
  `Close` ends the input; the facts; `Pipe`; `Streams` and `Env`.
* **`launch/launchtest/example_test.go`:** `ExampleTerminal`, the
  `--tui` path end to end. It shows the decision on a fake terminal, then
  a `Run` that quits on a typed q: "true launch.terminal 80 24 unicode",
  then "abq <nil> 0".
* **D6's move:**
  * `launch/consumer_test.go` (package `launch_test`, 18 tests) and
    `launch/consumer_unix_test.go` now hold the tests that use only the
    exported API, on `launchtest`:
    * `TestChoiceText`, `TestTargetText`, `TestFlagsRegister` and
      `TestFromSource`;
    * `TestRunQuits`, `TestRunUsesTheGivenInput`,
      `TestRunRegistryDetached` and `TestRunRestorerAfterCrash`;
    * `TestRunOnStart`, `TestRunEpilogue` and
      `TestWithFilterSeesTheModel`;
    * `TestRunFinalModelOfAnotherType` and
      `TestRunSignalsStayTheProgramsUnix`;
    * `TestFrameGolden`, `TestFrameRunsNoCommand`, `TestExitCode` and
      `TestExitError`.
  * `launch/flags_test.go`, `launch/frame_test.go` and
    `launch/run_unix_test.go` moved whole, and are deleted.
  * `launch/run_test.go` keeps the tests that need a seam:
    `TestRunPlainWritesNothing`, `TestRunNotStarted`, `TestRunCrash`,
    `TestRunEndsAreNotCrashes` and `TestRunOpensAndClosesTTY`.
    `launch/decide_test.go` is unchanged.
  * The golden files did not change. The moved test writes to the same
    `testdata/golden/`.
* **D5:** `sharedNames` has `Terminal`, and `docs/glossary.md` has its
  row.

#### Checks

* **Mutations, on scratch copies.** All killed:
  * S6-1 (`IsTerminal` false): `ExampleTerminal` printed "false
    launch.input-not-terminal 0 0 unicode", then " launch: the TUI did not
    start: launch.input-not-terminal 2";
  * D5's entry removed: "Terminal is exported by [launch/launchtest
    termcap/termcaptest]: one name, one meaning";
  * **the earlier mutations whose tests moved, run again** against the
    moved tests:
    * S3-8: build failure, "*launch.Choice does not implement
      flag.Getter";
    * S4-2: "the TUI did not start: … open /dev/tty";
    * S4-3: "program was interrupted";
    * S4-9: `TestRunRegistryDetached/crash`, "did not return within 1s";
    * S4-10: "does not end with mode 2031's reset";
    * S4-11: "want x filtered out";
    * S4-12: "after a crash: … Out holds";
    * S5-1: "nocolor.utf8.60: line 1 differs";
    * S5-2: "Frame called Init (true)";
    * S5-4: "context.Canceled: ExitCode = 1, want 130".
* **The API:** apidiff over all changes lists only additions since
  `v0.6.0`, now with `package …/launch/launchtest: added`.
  `go doc -short ./launch/launchtest` lists exactly the PLAN's names:
  `Env`, `Streams`, `Pipe`, `NewPipe`, `Terminal` and `NewTerminal`.
* **Rule 2 on macOS:**
  * `make pre-add-check` over the 7 changed Go files: clean;
  * `make release-check`: "150 file(s) clean … apicheck, examples";
  * `make lint`: three "0 issues.";
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 18 `ok`, and so did workspace mode. `launch/...` passed 10
    times under `-race`;
  * `go mod tidy -diff` and `scripts/go-modules.sh --check`: clean;
  * markdownlint: 0 issues. The link check: 0 broken.
* **The Windows test host,** go1.27.1 windows/amd64:
  * `make pre-add-check` gave "153 file(s) clean … apicheck, examples";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 18 `ok`;
  * `launch` and `launchtest` pass: 36 tests and examples;
  * **an artifact of the copy, and something it showed:**
    * The host's copy rebuilds its index from `d6b17bd`, which still
      tracks the three moved files, so the precheck listed
      `launch/run_unix_test.go`. gofmt then printed "GetFileAttributesEx
      launch/run_unix_test.go: The system cannot find the file
      specified".
    * The tree here has the deletions staged, so it does not happen
      there.
    * The precheck still reported the file clean. It reads gofmt's list
      of unformatted files and not gofmt's exit status, so a file gofmt
      cannot read passes. That is outside this PLAN, and is left for the
      owner.
* **The identifier scan of the diff and the new files:** no match.

### Step 7: the tier-1 framework programs and the live probes (scratch)

The owner approved it on 2026-10-07 ("Commit and proceed"), when Step 6
was committed as `6e89b0e`.

#### Deviations

Both were found while the probes ran. Each was asked, and the owner chose
the recommendation.

* **D7 (2026-10-07): "state the same" means every termios mode, less
  BSD's `PENDIN` state bit.**
  * **Found:** on macOS, L4 (a crash) and L5 (Bubble Tea failing in
    `initInputReader`) printed `same=false`. The states differed in one
    bit: `Lflag` went from `1483` to `536872395`, which adds `0x20000000`.
    That is `PENDIN`, which termios(4) describes as "retype pending input
    (state)": a state bit the kernel sets, not a mode. Every other field
    was equal: `Iflag`, `Oflag`, `Cflag`, the rest of `Lflag`, `Cc` and
    both speeds. `stty -a` showed `icanon` and `echo`.
  * **The control:** a scratch program with no launch and no Bubble Tea
    saved stdin's state with `term.GetState`, entered raw mode with
    `term.MakeRaw`, and restored it with `term.Restore`, under `script`.
    It showed the same `PENDIN` difference, with no input pending and
    with a key pending. The kernel sets the bit whenever canonical mode
    comes back, so a byte-for-byte check cannot pass on macOS.
  * **Chosen:** the probe compares every termios field, with `PENDIN`
    masked out of `Lflag`, and logs the bit it ignores. 0013-MADR gains
    A1.8 for the fact.
  * **Not chosen:** keeping byte for byte, with L4 and L5 recorded as
    failing on macOS.
* **D8 (2026-10-07): `Flags.TUI`'s tag gains `negatable:""`.**
  * **Found:** the kong program embeds `launch.Flags`, as the PLAN's
    table says. `kong --no-tui` gave "kong: error: unknown flag --no-tui"
    and exit 80. A1.2's tag list has no `negatable`, and a program cannot
    add a tag to a field of a struct it embeds. The table's "`TUI` with
    `negatable:""`" and A1.2's prose ("`negatable:""` adds `--no-tui`")
    therefore could not hold.
  * **Chosen:** `Flags.TUI`'s tag gains `negatable:""` in
    `launch/flags.go`, the one source change in this otherwise scratch
    step. The other frameworks ignore the key. apidiff does not compare
    tags. The existing tests and the scratch kong program check it.
  * **Not chosen:** correcting the table and A1.2's prose, so that a kong
    program wanting `--no-tui` would declare its own bool.
* **D9 (2026-10-08): `Config.OpenTTY` is not supported on Windows. This
  is the PLAN's first resolution for L7.**
  * **Found:** L7 on the Windows test host, in a console from `ssh -tt`,
    with `OpenTTY` and stdin piped. The TUI read from `CONIN$`, but:
    * **Keys arrived about one Enter late.**
      * W7b: `q` was typed at about 4 s, and the TUI quit at 9.9 s, when
        the next line's Enter arrived.
      * W7f: with `q` and Enter typed together, the model received
        `enter` first, and quit only at the following Enter.
    * **The CLI's next line was lost.** The CLI mode's read of the
      console got `""`, not the line typed.
      * W7, W7b and W7f: the line was gone.
      * W7c: of two lines typed, the second leaked to the shell after
        the program ended ("rc=127").
    * **The baseline is correct.** With no `OpenTTY` and stdin the
      console (W7e), `q` took effect at once (3.9 s), and the line typed
      afterwards reached the CLI ("hello").
  * **The cause the PLAN foresaw:** only stdin's console reader can be
    cancelled on Windows (ultraviolet's `cancelreader_windows.go`). A
    read of the `CONIN$` launch opened stays in flight, and takes the
    next input.
  * **No fix inside launch's rules:** cancelling it would mean making
    `CONIN$` the process's stdin during `Run`, and launch never touches a
    process-wide stream. The fix belongs upstream.
  * **Asked,** with two options:
    * `OpenTTY` unsupported on Windows;
    * refusing only its input there, which is untested.
  * **The owner chose** the first, the recommendation.
    * On `windows`, `Decide` treats `OpenTTY` as off, and never chooses
      `TargetTTY`.
    * `Config.OpenTTY`'s documentation says so.
    * 0013-MADR gains A1.9.
    * L7 is run again on Windows, to show the fallback.

#### What was built

* **In the tree:**
  * D8's tag on `Flags.TUI`;
  * D9's check in `decideWith`, and `Config.OpenTTY`'s documentation;
  * `TestDecideOpenTTYNotOnWindows`;
  * this record, and 0013-MADR A1.8 and A1.9.
* **In the scratchpad,** a module that requires the tree through a
  `replace`:
  * **A shared probe core.** Its model keeps the keys typed as its
    value, quits on q, and panics on p under `PROBE_PANIC`. It carries a
    workspace and a `termcap.Prober`, which is also its `Restorer`. It
    runs the `--tui` path with both fallbacks, and exits through
    `ExitCode`.
  * **Diagnostics go to a file** named by `PROBE_LOG`, so stderr carries
    only what a user would see. They record the decision, the run, the
    terminal state before and after (D7), `stty -a`, an agent's command
    after the TUI, and one line read from the console.
  * **The programs:**
    * tier 1: `flag`, `cobra`, `kong` and `urfave`, each bound as the
      PLAN's table says;
    * tier 2: `ff` (v4.0.0-beta.1, the latest), `goflags` (v1.6.1) and
      `goarg` (v1.6.1).
  * **For L5,** a copy of Bubble Tea v2.0.10 whose `initInputReader`
    fails, and a second module that replaces Bubble Tea with it.
  * **For D7,** a control program.

#### Checks

* **The framework programs, with no terminal:** stdin from `/dev/null`,
  stdout and stderr to files.
  * **Every program behaves the same for these:**
    * `--tui` and `--mode=tui`: one "Warning: --tui unavailable (launch:
      the TUI did not start: launch.input-not-terminal); using the CLI",
      then the CLI mode, exit 0, and no escape byte on stdout;
    * `--mode=plain` and no flag: the CLI mode, and nothing on stderr;
    * the refusing variant: exit 2.
  * **`--mode=bogus`** fails with each framework's own exit status, and
    each message names "launch: unknown Choice \"bogus\"":

    | Program | Exit status |
    | :--- | ---: |
    | `flag` | 2 |
    | `cobra` | 1 |
    | `kong` | 80 |
    | `urfave` | 1, with its usage on stdout |
    | `ff` | 2 |
    | `goflags` | 2 |
    | `goarg` | 2 |
  * **kong:**
    * `FatalIfErrorf(&ExitError{Code: 3})` exits 3;
    * after D8, `--no-tui` parses, and `--tui --no-tui` gives the CLI
      mode.
  * **Timing:** each program's first run took 420–710 ms, and every later
    run 70–90 ms. That is a newly built binary's first start, and all are
    under L1's 1 s.
* **S3-8's second check.** The urfave program, built without
  `Choice.Get`, fails to compile with "*launch.Choice does not implement
  cli.Value (missing method Get)".
  * The PLAN expected the go-flags program to fail to compile without
    `UnmarshalFlag`. It compiles: go-flags falls back to parsing the
    value as an integer, so every token fails at run time, "invalid
    argument for flag \`--mode' (expected launch.Choice):
    strconv.ParseUint: parsing \"plain\"", exit 2. The method is still
    needed. The PLAN's "fails to compile" was wrong for go-flags.
* **The live probes on macOS,** under `script -q /dev/null` at 100×30,
  with keys fed through `script`'s input. Run on the final tree:

  | Case | Result |
  | :--- | :--- |
  | L1 | one `Warning:` naming `launch.input-not-terminal`, the CLI mode, exit 0, no escape byte on stdout, 77–612 ms |
  | L2 | the warning names `launch.dumb-terminal`, then the CLI mode |
  | L3 | `flag`, `cobra`, `kong` and `urfave`: the TUI at `Decide`'s 100×30 (the model's size 100×30), no warning, exit 0 |
  | L4 | Bubble Tea's "Caught panic" report, the warning, the CLI mode with the last good value `"a"`; the state the same under D7, with only `PENDIN` differing; `stty -a` with `icanon` and `echo`. The prober's `Restore()` was empty: nothing under `script` answers its queries, so mode 2031 was never set, and no reports could arrive |
  | L5 | "Warning: --tui unavailable (launch: the TUI did not start: probe L5: initInputReader patched to fail)", the CLI mode, the state the same under D7 |
  | L6 | the UI on stderr (`ui=err`); stdout held only `result: value=""`, with no escape byte |
  | L7 | `in=tty`: the keys came from `/dev/tty`; a line typed after the TUI reached the CLI mode ("hello") |
  | L8 | the agent's `workspace.zoom` after the TUI returned at once (0–1 ms), run once, with no error |
* **D7's check seen to fail.** In a scratch copy of the tree, `Run` was
  changed to skip restoring the saved states. L5 then printed
  `same=false` with `pendin-differs=false`: a real difference, which the
  mask does not hide.
* **The live probes on Windows.**
  * **How they ran.** Under `ssh -tt`, this host ignores the command
    string and starts an interactive Git Bash in a console (ConPTY). Each
    case therefore typed its shell lines, then its keys, into that
    console.
  * **L1** ran without a console (no `-tt`, stdin from `/dev/null`,
    stdout piped): the warning, the CLI mode, exit 0, no escape byte on
    stdout.

  | Case | Result |
  | :--- | :--- |
  | L3 | the TUI at `Decide`'s 120×30 (ANSI256), exit 0 |
  | L4 | the crash report, the warning, the CLI mode with `"a"`; the console state the same byte for byte |
  | L6 | the UI on stderr; stdout held only the result, with no escape byte |
  | L7 | before D9: keys late and the CLI's line lost (D9's evidence); after D9: "Warning: --tui unavailable (launch: the TUI did not start: launch.input-not-terminal)", the CLI mode, and the line typed afterwards reached it ("hello") |
  | L8 | the agent's command returned in 1 ms, run once |
* **D9's check seen to fail.** With `goos != "windows"` dropped from
  `decideWith`, `TestDecideOpenTTYNotOnWindows` fails for both cases:
  "input on windows: true, launch.terminal, opened 1 times".
* **The tier-2 programs, recorded and not gated.** ff v4 (`NewFlagSetFrom`
  on the `ff` tags), go-flags (`Flags` as a group, `UnmarshalFlag`) and
  go-arg (`Flags` embedded, `UnmarshalText`) each parse `--tui` and
  `--mode`. Each falls back with the warning, and refuses `bogus`.
* **Rule 2,** for this step's Go changes: `launch/decide.go`,
  `launch/flags.go` and `launch/decide_test.go`.
  * **On macOS:**
    * `make pre-add-check` over the 3 files: clean;
    * `make release-check`: "155 file(s) clean … apicheck, examples";
    * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
      each gave 18 `ok`, and so did workspace mode;
    * `go mod tidy -diff` and `scripts/go-modules.sh --check`: clean;
    * markdownlint: 0 issues;
    * the citation checker resolves A1.7's and A1.9's citations, among
      them `ultraviolet/cancelreader_windows.go:28-40`.
  * **On the Windows test host:**
    * `make pre-add-check` gave "155 file(s) clean";
    * `make lint` and `make vuln` exit 0;
    * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 18 `ok`;
    * `launch` and `launchtest`: 37 tests and examples pass.
* **The identifier scan of the diff:** no match. The Windows console
  captures hold the account name in its prompt. They stay in the
  scratchpad, and every line this record quotes from them is redacted.

### Step 8: conformance, collision allowlist, documents

The owner approved it on 2026-10-08 ("Commit to main and proceed"), when
Step 7 was committed as `c680671`.

#### Deviations

* **D10 (2026-10-08): two descriptions this PLAN made stale are corrected
  here.**
  * **Found:** Step 8 lists the documents it changes. Two that it does
    not list no longer describe the tree:
    * `docs/glossary.md`, "Reserved by accepted records", still lists
      `launch`'s names as not yet built. Step 3's record said Step 8
      would move them.
    * `docs/architecture.md`'s package graph and table do not show Step
      2's additions: `internal/enum`, `glyph.Tier` and
      `Registry.Attach`. Step 8 lists only `internal/enum`'s line in the
      tree.
  * **Asked;** the owner chose to correct both in this step, the
    recommendation.
  * **Not chosen:** leaving them for a later record.

#### What was built

* **`internal/conformance`.**
  * `launch` and `launch/launchtest` are in the scan's must-read list,
    which comes from `go list` (W0).
  * The `Decision` entry has been in `sharedNames` since Step 3 (D1),
    and `Terminal` since Step 6 (D5).
* **The framework examples** (`testdata/frameworks`) gained the `--tui`
  path, so the guide's launch examples join the harness:
  * **Every program** holds the same `session` model (a stand-in that
    quits on q), the `tui` helper and `lineMode`.
    * `tui` decides with `ChoiceTUI`, runs with `WithRegistry`, and
      falls back with a warning, from the model reached, on
      `ErrNotStarted` or `ErrCrashed`.
    * `lineMode` is the program's own CLI mode.
    * `flag/main.go` marks `tui`, as the region `tui`.
  * **`flag`** parses `launch.Flags` on a global `FlagSet`, then the
    subcommand from its arguments, in the region `flag-tui`. The
    existing `flag` region now switches on `args[0]`, not `os.Args[1]`.
  * **`cobra`:** the root command's `RunE`, with `RegisterFlags` and
    `PersistentFlags().AddGoFlagSet`, and `FromSource`, in the region
    `cobra-tui`. Its exit is `launch.ExitCode(err)`.
  * **`kong`:**
    * The grammar embeds `launch.Flags`.
    * The default command, which was `TUI`, is `Line`.
    * The `--tui` path is in the region `kong-tui`.
  * **`urfave`:** the root command carries `--mode` (a `GenericFlag` on
    the `Choice`), `--tui` and the action, in the existing region. Its
    exit is `launch.ExitCode(err)`.
  * **`cases.txt`** adds three cases for each framework:
    * `--tui`: the warning names `launch.input-not-terminal`;
    * `--tui`: stdout holds "line mode";
    * `--mode=plain`: stdout holds "line mode".

    That makes 24 cases. `go.mod.tmpl` and `go.sum` needed no change.
* **`docs/guides/commands.md`:**
  * "Start the TUI from your CLI", with the pattern; `Decide`'s rules
    and `Decision`; `Run`'s outcomes; the flags in each framework, with
    the tier-2 table; `UIOnErr` and `OpenTTY` (not on Windows, A1.9);
    `WithRegistry`, `WithRestorer`, `OnStart` and `WithFilter`; `Frame`
    and `ExitCode`; `launchtest`; and the three cautions.
  * Its four code blocks come from the regions, and the excerpt check
    holds them.
  * In "Run commands from your own CLI", the `flag`, `kong` and `urfave`
    excerpts follow the programs, and the last bullet names the `--tui`
    cases.
* **The guides:**
  * `building-workspaces.md`, "The program owns the program": a
    paragraph pointing to the new section;
  * `terminal-capabilities.md`, "Do not probe without input": `Run`
    always sets an input, and a prober goes to `WithRestorer` as well.
* **`README.md`:** the current release is `v0.7.0`, a Status bullet for
  `launch`, and an "I want to…" row.
* **`docs/README.md`:** rows for the guide section and `launchtest`. The
  0013-MADR row now reads "know why `launch` decides and falls back as it
  does".
* **`docs/architecture.md`,** with D10's part:
  * the graph: `launch`, `launch/launchtest` and `internal/enum`, with
    `glyph` and `termcap` now pointing to `internal/enum`;
  * the table: rows for `launch`, `launch/launchtest` and
    `internal/enum`. `glyph`'s row names `Tier`, and `command`'s names
    `Attach`;
  * the tree: `launch/`, `launch/launchtest/` and `internal/enum/`;
  * Dependencies: x/term required, and kept to `launch` by depguard;
  * "What is not here": the sentence about helpers is gone.
* **`docs/glossary.md`,** D10's other part. `launch`'s nine names and
  `glyph.Tier` leave "Reserved", and a short section says they are built.

#### Checks

* **Mutations, on scratch clones.** All killed by
  `TestNoPackageOwnsTheTerminal`, each naming the planted file:
  * S8-1, a `launch` file writes to `os.Stderr`: "launch/zz_mutant.go:8
    uses os.Stderr";
  * S8-2, a `launch` file sets `AltScreen`: "launch/zz_mutant.go:6 sets
    AltScreen";
  * S8-3, a `launch` file imports `os/exec`: "imports os/exec", "uses
    os/exec.Command".
* **The new harness cases seen to fail.** In a clone, the `flag`
  program's fallback lost its warning. `make examples` failed with "flag
  --tui => 0 stderr launch.input-not-terminal: stderr does not hold …",
  and the `tui` excerpt differed.
* **The harness:** `make examples` gave "24 case(s) run", "9 excerpt(s)
  checked" and "clean".
* **Rule 2 on macOS:**
  * `make pre-add-check` over the 4 programs gave "4 file(s) clean … go
    vet, go test, …, examples". The examples gate ran, as D5 of
    0014-PLAN-api-policy-gates.md has it;
  * `make release-check`: "155 file(s) clean … apicheck, examples";
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 18 `ok`, and so did workspace mode;
  * `go mod tidy -diff` and `scripts/go-modules.sh --check`: clean;
  * markdownlint over the repository: 0 issues;
  * the link check over the eight changed Markdown files: 153 links,
    none broken.
* **The Windows test host,** go1.27.1 windows/amd64:
  * `make examples` gave "24 case(s) run", "9 excerpt(s) checked" and
    "clean";
  * `make pre-add-check` gave "155 file(s) clean";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 18 `ok`;
  * `launch` and `launchtest`: 37 tests and examples pass.
* **The identifier scan of the diff:** no match.

### Step 9: the release

The owner asked on 2026-10-08 for Step 8 to be committed and everything
outstanding pushed. The agent committed it as `59f1496`. The disclosure
guard passed over the six outgoing commits (`cb116d0` to `59f1496`), and
the agent pushed `main`. CI run `37792209643` passed every job on
`59f1496`: `modules`, `test` on Ubuntu, macOS and Windows, and `gates`
with `apicheck`, `examples` and the lint step. The owner then tagged
`v0.7.0` and pushed it.

* **Before the tag,** on `59f1496`:
  * `make release-check`: "155 file(s) clean … apicheck, examples";
  * `make lint`: three "0 issues.";
  * `make vuln`: "No vulnerabilities found.";
  * a dry run of the smoke test against the tree, through a `replace`,
    gave the result below.
* **The tag:** `v0.7.0` is an annotated tag on `59f1496`, with the
  message "v0.7.0".
* **The consumer smoke test** against the published `v0.7.0`, as
  `docs/guides/releasing.md` gives it. It ran in a scratch module outside
  the repository, with `go env GOWORK` empty.
  * `go get github.com/maccavelli/go-tui-lib@v0.7.0` downloaded
    `v0.7.0`, and `go mod tidy` passed.
  * The program imports `launch`, `launch/launchtest`, `workspace`,
    `layout` and `termcap`. `go vet` and `go build` passed.
  * `go run .` printed:
    * "decide: false launch.input-not-terminal";
    * "launchtest decide: launch.terminal";
    * "run: launch: the TUI did not start: launch.input-not-terminal".

    It exited 2 through `ExitCode`, as designed.
  * `go list -m all` showed `github.com/charmbracelet/x/term v0.2.2` and
    `charm.land/bubbletea/v2 v2.0.10`.
* **Release notes for `v0.7.0`:**
  * **Added:**
    * `launch`, with `Choice`, `Flags`, `Streams`, `StreamSource`,
      `FromSource`, `Config`, `Decide`, `Decision`, `Target`, `Reason`,
      `Run`, its options, `ErrNotStarted`, `ErrCrashed`, `Frame`,
      `ExitCode`, `ExitError` and `Restorer`;
    * `launch/launchtest`, with `Terminal`, `Pipe`, `Streams` and `Env`;
    * `glyph.Tier`, with `TierUnicode`, `TierLegacy`, `TierASCII` and
      `Tier.Set`;
    * `command.(*Registry).Attach`;
    * internally, `internal/enum`, which termcap's enums now use.
  * **Changed:** `github.com/charmbracelet/x/term` is a direct
    requirement, at the version Bubble Tea already selected. No module
    was added.
  * **Removed:** nothing. `make apicheck` reported "against v0.6.0, 0
    incompatible change(s)". apidiff, run over all changes, listed only
    additions.
  * **Notes:**
    * `Config.OpenTTY` is not supported on Windows (0013-MADR A1.9).
    * `Flags.TUI` carries `negatable:""`, so a Kong program that embeds
      `Flags` gets `--no-tui`.

### Step 10: close-out

The owner asked on 2026-10-08 ("v0.7.0 is pushed"). Verification, item by
item, run again on `59f1496`, which `git describe --exact-match` names
`v0.7.0`:

* **`Choice` and `Flags`** parse `--tui` and `--mode` in the four tier-1
  frameworks.
  * Step 7's scratch programs showed it, and so do the harness programs
    `make examples` runs.
  * That run gave "24 case(s) run, 9 excerpt(s) checked, clean".
  * The tier-2 results are in Step 7's record.
* **`Decide`** follows §3–§6 as amended by A1.3, A1.7 and A1.9. It
  spawns nothing (S3-3), and S3-1 to S3-10 were killed (Step 3, run again
  in Step 6 for the moved tests).
* **`Run`:**
  * it holds the fallback in every outcome;
  * it detaches the registry (L8 on both hosts) and writes the
    `Restorer` (`TestRunRestorerAfterCrash`);
  * it restores the saved state (L4 and L5, under D7);
  * it returns the last good model;
  * S4-1 to S4-12 were killed. S4-4 was caught by `TestRunCrash`;
    `TestRunNotStarted` cannot catch it.
* **`Frame`, `ExitCode` and `ExitError`** follow §8, §9 and A1.2. S5-1 to
  S5-4 were killed.
* **`launchtest`** carries the package's exported-API tests (D6), and
  S6-1 was killed.
* **Conformance** scans `launch` and `launchtest`. S8-1 to S8-3 were
  killed. `go test -v ./launch/... ./internal/conformance` gave 42
  passing tests and three `ok`.
* **The live probes L1–L8** pass on macOS. On the Windows test host, L1,
  L3, L4, L6 and L8 pass, and L7 passes as D9 decided: a clean fallback.
* **x/term** is direct at v0.2.2 (`go.mod:12`), and kept to `launch` by
  depguard's `xterm` rule. The module list (`GOWORK=off go list -m all`)
  is the same 29 lines as before Step 3.
* **`make apicheck`** gave "against v0.6.0, 0 incompatible change(s)"
  and "clean": additions only.
* **`v0.7.0`** is tagged, and the smoke test runs (Step 9).
* **Rule 2's checks** were clean at every step, on macOS and the Windows
  test host, and the identifier scan of each step's diff found nothing.

**One line of the rollout plan differed:** it said `v0.7.0` would be
tagged after Step 7's probes. It was tagged after Step 8, as the step
order requires, since Step 8's documents describe the release.

**The MADR's facts** that differed were amended during execution: A1.7
(what `colorprofile.Env` reads on Windows), A1.8 (`PENDIN`, and the
`negatable` tag) and A1.9 (`OpenTTY` on Windows). No further amendment is
needed.

**Open after this PLAN:**

* **`scripts/go-precheck.sh` ignores gofmt's own errors** (Step 6). It
  reads gofmt's list and not its exit status, so a file gofmt cannot read
  passes. That belongs to a later record.
* **A cancellable reader for a console handle other than stdin,**
  upstream in ultraviolet. It would let `OpenTTY` work on Windows
  (A1.9).

Every Verification item holds, and this PLAN is `complete`.

### Step 11: drop Kong's `--no-tui`

The owner approved it on 2026-10-08 ("proceed, commit all records, then
tag v0.7.1"). The records were committed first, as `021303a`. No
deviation.

#### What was built

* **`launch/flags.go`:** `Flags.TUI`'s tag lost `negatable:""`. The doc
  comment now says there is no `--no-tui`, and that Kong's `--tui=false`
  cancels an earlier `--tui`, citing A1.10.
* **`testdata/frameworks/kong/main.go`:** the comment on the embedded
  `launch.Flags` is `// --mode and --tui`.
* **`docs/guides/commands.md`:** the Kong excerpt follows the program.
  The Kong sentence says there is no `--no-tui`, and that `--tui=false`
  cancels an earlier `--tui`.
* **`testdata/frameworks/cases.txt`:** `kong --no-tui => 80 stderr
  unknown flag --no-tui`, and `kong --tui --tui=false => 0 stdout line
  mode`.
* **The records:**
  * 0013-MADR A1.10 is accepted;
  * this PLAN is `complete` again;
  * `docs/README.md`'s 0013 rows name A1.10 and `v0.7.1`.

#### Checks

* **S11-1, the tag put back,** in a scratch clone: `make examples` failed
  with "kong --no-tui => 80 stderr unknown flag --no-tui: exit 0, want
  80".
* **`make examples`:** "26 case(s) run", "9 excerpt(s) checked",
  "clean".
* **`make apicheck`:** "against v0.7.0, 0 incompatible change(s)".
* **Rule 2 on macOS:**
  * `make pre-add-check` over the 2 Go files: "2 file(s) clean … go vet,
    go test, …, examples";
  * `make release-check`: "155 file(s) clean … apicheck, examples";
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 18 `ok`, and so did workspace mode;
  * `go mod tidy -diff` and `scripts/go-modules.sh --check`: clean;
  * markdownlint: 0 issues.
* **The Windows test host,** go1.27.1 windows/amd64:
  * `make examples`: "26 case(s) run", "9 excerpt(s) checked", "clean";
  * `make pre-add-check`: "155 file(s) clean … apicheck, examples";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 18 `ok`;
  * `launch` and `launchtest` pass.
* **The identifier scan of the diff:** no match.

#### The release, `v0.7.1`

* **The tag:** the agent created `v0.7.1`, at the owner's request, as an
  annotated tag ("v0.7.1") on this step's commit. Its push waits for the
  owner: first `main`, then the tag once CI passes on `main`, as
  `docs/guides/releasing.md` orders it. The consumer smoke test runs
  against the published tag after that.
* **Published (2026-10-08).** The owner pushed `main` and the tag. CI run
  `37809306889` on `cad97f5` passed every job. `git ls-remote` shows
  `refs/tags/v0.7.1` on `origin`, peeling to `cad97f5`.
* **The consumer smoke test** against the published `v0.7.1`, as Step 9
  ran it for `v0.7.0`, in a scratch module with `go env GOWORK` empty:
  * `go get github.com/maccavelli/go-tui-lib@v0.7.1` downloaded `v0.7.1`;
  * `go vet` and `go build` passed;
  * `go run .` printed "decide: false launch.input-not-terminal",
    "launchtest decide: launch.terminal" and "run: launch: the TUI did
    not start: launch.input-not-terminal", and exited 2 through
    `ExitCode`, as designed;
  * `go list -m all` showed `github.com/charmbracelet/x/term v0.2.2`.
* **Release notes for `v0.7.1`:**
  * **Changed:** a Kong program that embeds `launch.Flags` no longer
    accepts `--no-tui`, which only repeated the default. `--tui=false`
    cancels an earlier `--tui`. The other frameworks are unchanged.
  * No API change: `make apicheck` against `v0.7.0` is clean.
