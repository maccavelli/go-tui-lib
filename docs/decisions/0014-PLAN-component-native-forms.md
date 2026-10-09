---
status: in-progress
date: 2026-10-08
associated-madr: "0014-MADR-native-integration-api.md"
---
# Implement W2: native forms in `command`, `workspace`, `termcap`, `termsvc`, `theme` and `tuitest`

Associated MADR: [0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md),
workstream W2, as amended by A1.

## Goal

A program's own CLI uses each package natively, without a TUI running:

* **`command`:**
  * parses an argument vector, describes a command's parameters, and
    completes them;
  * writes a result as text or JSON;
  * maps its errors to exit statuses;
  * survives a panicking handler;
  * builds arguments and gates in one call;
  * stops watching on a context.
* **`workspace`:**
  * takes glyphs, profile, background and size;
  * renders plainly, for a pipe or a screen reader;
  * draws its help with the library's glyphs.
* **`termcap`:** gives the environment's facts without a program, and
  reads Windows environment names without regard to case.
* **`termsvc`:**
  * returns a notification's or a copy's bytes, for a CLI to write;
  * runs its backends under a context with a timeout.
* **`theme`:** its enums have text forms.
* **`tuitest`:** maps a case to its profile and glyphs, and can assert
  widths.

Everything is additive, and released in `v0.8.0` with W3.

## Scope

### Facts this PLAN starts from (2026-10-07)

| Fact | Where it was read |
| :--- | :--- |
| 0013's PLAN delivers `internal/enum`, `glyph.Tier` and `Registry.Attach` in `v0.7.0` | [0013-PLAN-cli-integration-helpers.md](0013-PLAN-cli-integration-helpers.md), Step 2 |
| `CallMCP` renders `Text`, or `Value` as deterministic JSON v2 | `command/mcp.go:99-108` |
| `execute` runs every path's handler with no `recover` | `command/dispatch.go:149-164`; no `recover()` outside tests |
| the sentinels are `errors.New` values; `*ArgError` exists | `command/registry.go:21-30`, `command/args.go:13-26` |
| `ParseSlash` parses one line: positional values and `name=value`; the compiled rule keeps only `x-cli` `arg` and `rest` | `command/slash.go:24-106`; `command/decode.go:104-123` |
| `SchemaOf` emits `x-cli` `short`, `placeholder`, `group` and `hidden`, never compiled | `command/schema.go:104-111` |
| the retired `command/cli` had a flag-style parser and result printer | `git show 60ed13e^:command/cli/flags.go` (239-293), `cli.go` (197-216) |
| JSON v2 has no default representation for `time.Duration`; the decoder has its own duration unmarshalers | a probe; `command/decode.go:412-423` |
| `Watch` blocks on the registry's change channel with no context | `command/registry.go:389-404` |
| `Gate` is `Decide(ctx, *Invocation) (Decision, error)`; the guide writes a `yesGate` and `json.Marshal`s arguments | `command/gate.go:43-45`; `docs/guides/commands.md:304-330` |
| three built-in commands panic on a construction error | `command/builtin.go:24-62` |
| the workspace's first theme is fixed at `theme.New(colorprofile.ANSI256, theme.Unknown, glyph.Unicode())`; `ThemeBuilder` takes no glyphs; the builder is not used for the first theme | `workspace/workspace.go:187`, `:261-292` |
| `Pane` has `Update` and `View(w, h)`, plus nine optional interfaces; `Model[M]` forwards some of them | `workspace/workspace.go:46-90`; `workspace/model.go:98-160` |
| bubbles v2.2.1's `help.New()` draws `" • "` and `"…"`; `glyph.Set` has `Bullet` and `Ellipsis` | bubbles `help/help.go:48-62`; `glyph/glyph.go:36,41` |
| `setEnv` is unexported; `WithDisabled` gives only the environment's facts | `termcap/env.go:31-87`; `termcap/prober.go:80-84` |
| `termcap.Env` lookups are case-sensitive | `termcap/env.go:16-23` |
| the notifier's encoder is a stateful method; `Notify` and `Copy` call their backends under `context.Background()` | `termsvc/notify.go:182-197`, `:236-276`; `termsvc/clipboard.go:72-94` |
| `theme.Background` and `BorderStyle` are `int`s without text forms; `workspace`'s theme command spells `Unknown` as `"auto"` | `theme/theme.go:35-43`, `:181`; `workspace/commands.go:115`, `:327` |
| four tests map a `tuitest.Case` to TrueColor or ASCII and `glyph.For(c.UTF8)` | `workspace/golden_test.go:17-23`, `agent_golden_test.go:29-33`, `commands_test.go:420-424`; `theme/theme_test.go:157-162` |
| 12 workspace goldens are drawn with WcWidth and are wider than their nominal width under `ansi.StringWidth` | the `width-wcwidth.*` goldens, measured |

### Facts re-read before execution (2026-10-08)

`v0.7.0` and `v0.7.1` shipped after this PLAN was written. Before
execution the agent read every row above again on `7b4bded`. The design
facts hold, apart from the ones this table corrects. Where only a line
moved, the citation in the table above is still the 2026-10-07 reading,
and this table gives the line today. The steps below cite today's lines.

| Fact | Today |
| :--- | :--- |
| **Corrected.** Eight workspace goldens, not 12, are wider than their nominal width under `ansi.StringWidth`: the eight `width-wcwidth.*` goldens, 83 cells at 80 and 121 at 120, and none under `ansi.WcWidth`. The other four of the twelve measured are `tuitest`'s own `clash` fixtures, which are wider under both methods on purpose. | measured over every `testdata/golden` directory with both methods: `workspace` 8 of 152; `tuitest/internal/clash` 4 of 8; `command`, `launch`, `layout`, `theme` and `termcap` none |
| **New since the PLAN.** 0013 Step 2c put a guard on the loop. `onLoop`'s closure recovers a Loop handler's panic, sends the caller an error that names the panic's value ("command: <id> panicked on the loop: <value>"), and panics again, so that Bubble Tea recovers it and the TUI crashes. 0013's PLAN recorded it as the behaviour "until 0014 W2's `recover` lands". `TestLoopPanicReleasesCaller` asserts both halves. | `command/dispatch.go:103-146`, the guard at `:111-119`; `command/loop_test.go:184-209`; [0013-PLAN-cli-integration-helpers.md](0013-PLAN-cli-integration-helpers.md) Step 2 |
| **New since the PLAN.** `launch` imports `command` (`WithRegistry`), so a `command` test that calls `launch.ExitCode` lives in the external test package `command_test`. Every `command` test file today is in package `command`. | `launch/option.go:31`; `go list -deps ./launch` |
| **New since the PLAN.** `launch.ExitCode` honours an error's own `ExitCode() int` first, through the error chain. | `launch/exit.go:48` |
| **New since the PLAN.** `launch` reads `termcap.Env` (`NO_COLOR`, `FORCE_COLOR`, `CLICOLOR_FORCE`, `CI`, `TERM`, `COLUMNS`, the program's `NoInputEnv` and the locale), so Step 8's case-folding reaches `launch`'s decision on Windows too. colorprofile's own reading of the list `launch` gives it does not change. | `launch/colour.go:19-38`, `launch/locale.go:18-20`, `launch/decide.go:202-242` |
| **Corrected.** Step 5's `ParseArgs` is a method on `*Registry`, as `ParseSlash` is and as 0014-MADR W2 names it (`Registry.ParseArgs`): it needs the registry to find the command. Step 5 as written dropped the receiver. | `command/slash.go:24`; 0014-MADR W2's table |
| moved: `execute` runs the handler at `command/dispatch.go:218`, in `execute` at `:207-222` | `command/dispatch.go` |
| moved: the sentinels at `command/registry.go:19-30`; `ArgError` at `command/args.go:16-26` | |
| moved: the compiled `x-cli` at `command/decode.go:108-123`, inside `scalars` at `:90-124` | |
| moved: `Watch` at `command/registry.go:406-418` | |
| moved: the built-ins at `command/builtin.go:23-63`, the panic at `:60` | |
| moved: the first theme at `workspace/workspace.go:268`, in `New` at `:264-295`; `ThemeBuilder` at `:190`; `Pane` at `:49`; `Model[M]`'s forwarding at `workspace/model.go:79-162` | |
| moved: the notifier's encoder at `termsvc/notify.go:238-277` | |
| moved: `theme.Background` at `theme/theme.go:39-48`, `BorderStyle` at `:184-195` | |
| moved: `make fuzz` at `Makefile:98-101` | |
| unchanged: `CallMCP`'s rule; `ParseSlash`; `cliInfo`; `durationUnmarshalers`; `Gate`; the guide's `yesGate`; `setEnv`, `WithDisabled` and `Env.LookupEnv`; `Notify` and `Copy` under `context.Background()`; `workspace/commands.go:115`, `:327`; the four mapping sites; termcaptest's seven profiles; bubbles' `help.New()` | as cited above |
| no clash: `Format`, `Param`, `PanicError`, `PlainViewer` and `GlyphThemeBuilder` are in `docs/glossary.md`, reserved by 0014-MADR W2. `GateFunc` is not a type name elsewhere. No public package exports any of the new function or method names yet | `docs/glossary.md:39-43`; a search of the non-test Go files |
| the preconditions hold: 0013's PLAN is complete, `v0.7.0` and `v0.7.1` are tagged, W0's gates are in force, and `scripts/apicheck.allow` lists nothing | [0014-PLAN-api-policy-gates.md](0014-PLAN-api-policy-gates.md); `git tag` |

### Preconditions

* **0013's PLAN is complete and `v0.7.0` is tagged.** This PLAN uses
  `internal/enum`, `glyph.Tier` and `Registry.Attach`.
* **The gates of 0014-PLAN-api-policy-gates are in force:**
  * `make apicheck` must show additions only;
  * the conformance bans and the collision check apply.

### In scope

| Step | Package | Delivers |
| :--- | :--- | :--- |
| 1 | — | approval |
| 2 | `command` | `ErrPanicked`, `PanicError`, `recover`; `ExitCode()` on the errors |
| 3 | `command` | `ArgsOf`, `MustNew`, `GateFunc`, `AllowIf`, `WatchContext` |
| 4 | `command` | `Format`, `WriteResult` |
| 5 | `command` | the compiled `x-cli` tags; `ParseArgs`; `Param`, `Params`; `Registry.Complete` |
| 6 | `workspace` | `WithGlyphs`, `WithProfile`, `WithBackground`, `WithSize`, `GlyphThemeBuilder`; the first theme through the builder |
| 7 | `workspace` | `PlainViewer`, `RenderPlain`, `Help` |
| 8 | `termcap` | `EnvCaps`; case-folding on Windows |
| 9 | `termsvc` | `Sequence`, `CopySequence`, `ErrCopyTooLarge`; `NotifyContext`, `CopyContext`; backend timeouts |
| 10 | `theme`, `internal/enum`, `tuitest` | enum text for `Background` and `BorderStyle`; `Case.Profile`, `Case.Glyphs`, `Fits` |
| 11 | documents | the guides, `docs/architecture.md`, `README.md`, the framework examples |
| 12 | — | close-out; the release is W3's PLAN's last step |

### Out of scope

* **Migrating the other 20-odd enums** to `internal/enum` (W4).
* **Renames and opaque options** (W4).
* **Limits on arguments, `LoadDir`, the window and the view cache**
  (W3). `WithSize` here passes through W3's `clampSize` once that lands,
  in the same release.
* **A screen-reader mode,** beyond `RenderPlain`'s linear output (W5).

## Implementation Steps

### Rules

1. **A step starts when the previous one is committed.** The agent
   commits on `main` only when the owner asks in that turn, with `git
   commit --no-edit`.
2. **Checks.**
   * **For a step that changes Go:**
     * `gofmt -l`;
     * `make pre-add-check FILES="…"`;
     * `make lint`;
     * with `GOWORK=off`: `-race`, `-shuffle=on -count=2` and `LC_ALL=C`;
     * workspace mode;
     * `go mod tidy -diff`, `make vuln`, `scripts/go-modules.sh --check`;
     * `make apicheck`, which must show additions only;
     * `make examples`;
     * `make release-check`;
     * the Windows test host.
   * **For every step:** markdownlint, the link and citation checkers,
     and the identifier scan.
3. **Each mutation runs on a scratch copy** and must fail its named test.
4. **Anything unplanned stops the step,** recorded as a dated deviation.
5. **No existing golden changes** unless the step says so, and goldens
   are read before commit.

### Step 1: records

* **This PLAN, approved.**
* **Its row in `docs/README.md`.**

### Step 2: panics and exit statuses in `command`

* **`ErrPanicked`.** `var ErrPanicked error = …`, and
  `type PanicError struct{ ID ID; Value any; Stack []byte }` with:
  * `Error() string`, which gives "command: <id>: handler panicked" and
    leaves out the value, because a panic value may hold a secret. A
    gate's panic gives "command: <id>: gate panicked", through an
    unexported field (D1, 2026-10-08);
  * `Unwrap() error`, which returns `ErrPanicked`;
  * `ExitCode() int`, which returns 2.
* **`recover`** goes inside a helper around `c.Handler.Run`
  (`command/dispatch.go:160`). The helper sets `o.err` to a `*PanicError`
  holding `debug.Stack()`. `o.dur` is still set, and Async's `done` still
  runs. The same helper wraps `Gate.Decide`; a panicking gate gives a
  refusal wrapping the `PanicError`.
* **The loop's guard (added 2026-10-08).** A Loop handler's panic is
  recovered inside `execute`, like every other handler's. So the loop
  never sees it, and the TUI keeps running. The caller gets the
  `*PanicError`, and the value is not in its text. That is the end state
  0013's PLAN named, and 0014-MADR W2 decides "`recover` around every
  handler".
  * `onLoop`'s guard (`command/dispatch.go:111-119`) is removed with its
    comment. With the handler recovered inside `execute`, nothing it
    wraps can panic.
  * `TestLoopPanicReleasesCaller` is rewritten for the new end state. The
    caller is still released within a second. Its error is a
    `*PanicError` for `loop`, and the loop goroutine recovers `nil`.
  * Before Step 2, a panicking Loop handler under `launch.Run` crashed
    the TUI with `ErrCrashed`. After it, `Run` keeps going, and the
    command's caller gets the error. The release notes say so (Step 11
    and W3's release step).
* **The audit** records the `PanicError`. `SlogAuditor` logs its `Error()`
  only; the value and the stack stay in the error for the program.
* **Exit statuses (A1):**
  * the sentinels become `var ErrUnknown error = &codedError{msg, 2}`,
    `ErrUnavailable` with 1 and `ErrRefused` with 3;
  * `codedError` is an unexported pointer type with `Error()` and
    `ExitCode()`;
  * the declared type stays `error`, so `apidiff` sees no change and
    `errors.Is` keeps working by identity;
  * `(*ArgError).ExitCode()` returns 2.
* **Tests:**
  * `TestHandlerPanicIsAnError`, for Loop through `Dispatch`, Async
    inside its command, `Run`, and `CallMCP`;
  * `TestGatePanicRefuses`;
  * `TestPanicErrorHidesValue`;
  * `TestExitCodes`: each error bare and wrapped, through
    `launch.ExitCode` and through `errors.As` to
    `interface{ ExitCode() int }`. It is in the external package
    `command_test`, because `launch` imports `command` (added
    2026-10-08). A panicking gate's refusal gives 3, the status of the
    refusal, which comes first in its chain;
  * `TestSentinelsStillCompare`.
* **Mutations:**
  * **S2-1:** no `recover`. `TestHandlerPanicIsAnError` crashes the test
    binary, which counts as failing.
  * **S2-2:** `Error()` includes the value. `TestPanicErrorHidesValue`.
  * **S2-3:** `ErrRefused`'s status is 1. `TestExitCodes`.

### Step 3: `ArgsOf`, `MustNew`, `GateFunc`, `AllowIf`, `WatchContext`

* **`ArgsOf[A any](a A) (json.RawMessage, error)`** is deterministic JSON
  v2. Its duration marshaler mirrors `durationUnmarshalers`, so a
  `time.Duration` field marshals to the schema's string form.
* **`MustNew[A any](id ID, title string, run func(ctx context.Context,
  inv *Invocation, args A) (Result, error), opts ...Option) Command`** is
  `New[A]`'s signature (`go doc ./command New`), and panics on error. The three built-ins (`command/builtin.go:24-62`) use
  it.
* **`type GateFunc func(context.Context, *Invocation) (Decision, error)`**
  with a `Decide` method. **`AllowIf(ok bool) Gate`** gives `AllowOnce`
  when `ok`, and `RejectOnce` otherwise.
* **`WatchContext(ctx) tea.Cmd`** returns a nil message when `ctx` ends.
  `Watch()` is `WatchContext(context.Background())`.
* **Tests:**
  * `TestArgsOfDuration`: the duration's string, accepted by `New[A]`'s
    rule;
  * `TestArgsOfDeterministic`;
  * `TestMustNewPanics`;
  * `TestAllowIf`;
  * `TestGateFunc`;
  * `TestWatchContextEnds`: returns within 100 ms of cancel, and has no
    goroutine left (a `runtime.NumGoroutine` check around it).
* **Mutations:**
  * **S3-1:** `ArgsOf` uses the default duration form.
    `TestArgsOfDuration`.
  * **S3-2:** `WatchContext` ignores `ctx`. `TestWatchContextEnds`.

### Step 4: `Format` and `WriteResult`

* **`type Format uint8`:** `FormatText` and `FormatJSON`, with text forms
  `text` and `json` through `internal/enum`, so `--format` binds natively.
* **`WriteResult(w io.Writer, res Result, f Format) error`:**
  * **`FormatText`** writes `CallMCP`'s rule, `Text` or else `Value` as
    deterministic JSON, with a trailing newline when it lacks one.
  * **`FormatJSON`** writes `Value` as deterministic JSON, or `null`, and
    a newline.
  * The duration marshaler applies.
  * `Result.Cmd` is ignored, and the documentation says why.
  * The rule moves into an unexported `resultText`, which `CallMCP` calls,
    so its goldens must not change.
* **Tests:**
  * `TestWriteResult`: each format, with `Text` only, `Value` only, both,
    a duration, and a failing writer;
  * `TestCallMCPUnchanged`: the existing MCP goldens.
* **Mutation S4-1:** `FormatText` adds no newline. `TestWriteResult`.

### Step 5: `ParseArgs`, `Params`, `Complete`

* **The rule compiles all of `x-cli`.** The rule
  (`command/decode.go:104-123`) adds `short`, `placeholder`, `group` and
  `hidden`, with `help` read from `description`. Those five are read
  leniently (D4, 2026-10-08).
* **`ParseArgs(id ID, args []string, o Origin) (Request, error)`,**
  a method on `*Registry` (corrected 2026-10-08; see the facts re-read
  before execution):
  * **The grammar:**
    * `--name=v`, `--name v` and `-s v`, and `-s=v` (D3, 2026-10-08);
    * a bare `--name` for a boolean;
    * `--` ends the flags;
    * positional values and `rest` as in `ParseSlash`;
    * no `name=value` form, because the shell already splits;
    * no `--no-x` (W5 may add it).
  * Flag names are the JSON property names.
  * **`Raw`** is the arguments joined with single quotes where needed, so
    `$ARGUMENTS` expansion sees what the user typed.
  * **Errors:** `ErrUnknown` for an unknown ID; `*ArgError` naming the
    flag for a bad value, an unknown flag, or a missing required one.
  * **The code** follows the retired parser
    (`git show 60ed13e^:command/cli/flags.go`) over the compiled rule.
* **`type Param struct`** has:
  * `Name`, `Type`, `Items`, `Description`, `Short`, `Placeholder`,
    `Group`;
  * `Required`, `Positional`, `Rest`, `Hidden`, `Secret`, `HasDefault`;
  * `Default any`, `Enum []any`, `Min` and `Max *float64`.
* **`Params(c Command) ([]Param, error)`** takes the top-level properties,
  in schema order. It returns an error for a schema that does not
  compile, since an unregistered `Command` may carry one.
* **`(*Registry) Complete(id ID, args []string, partial string) []string`:**
  * with an empty `id`, the IDs offered on `SurfaceCLI`;
  * otherwise flag names, a flag's enum values, or `true`/`false`;
  * sorted and de-duplicated;
  * availability is not consulted, and the documentation says so.
* **`FuzzParseArgs`** checks:
  * no panic;
  * a successful parse prepares;
  * `Raw` round-trips through the same parse.

  It joins `make fuzz` (`Makefile:94-97`, `command` already listed).
* **Tests:**
  * `TestParseArgs`, a table over every grammar form and every error;
  * `TestParseArgsMatchesSlash`: the same arguments by `ParseArgs` and by
    `ParseSlash` give the same prepared JSON, where both grammars can
    express them;
  * `TestParams`: a struct with every tag;
  * `TestComplete`.
* **Mutations:**
  * **S5-1:** `--` does not end the flags. `TestParseArgs`.
  * **S5-2:** a missing required flag passes. `TestParseArgs`.
  * **S5-3:** `Complete` lists IDs not on `SurfaceCLI`. `TestComplete`.

### Step 6: the workspace's options

* **The options:**
  * `WithGlyphs(glyph.Set)`, `WithProfile(colorprofile.Profile)`,
    `WithBackground(theme.Background)` and `WithSize(width, height int)`;
  * `type GlyphThemeBuilder func(colorprofile.Profile, theme.Background,
    glyph.Set) theme.Theme`, with `WithGlyphThemeBuilder(b)`.
* **The rules:**
  1. **They set the starting facts.**
     * `WithBackground` is the starting background, not a pin;
       `SetBackground` stays the pin.
     * `WithSize` is the size before the first `WindowSizeMsg`. Once W3
       lands it passes through `clampSize`.
  2. **The first theme is built after every option has applied.**
     * `WithTheme` wins over everything: the other options are then
       ignored, and the documentation says so. (D5, 2026-10-08: it wins
       for the first theme and the starting facts. Whether the workspace
       then follows is still decided by the last of `WithTheme` and the
       two builder options, as today.)
     * Otherwise the first theme comes from the `GlyphThemeBuilder`, else
       the `ThemeBuilder`, else `theme.New(profile, bg, glyphs)`.
     * Rebuilds use the same choice.
  3. **A `ThemeBuilder` chooses its own glyphs, as today.**
     `WithGlyphs` reaches only the default builder and a
     `GlyphThemeBuilder`.
* **Tests:**
  * `TestFirstThemeFollowsOptions`: each combination, including option
    order;
  * `TestWithThemeWins`;
  * `TestWithSizeBeforeFirstMessage`;
  * `TestGlyphsReachBuilder`.

  The existing goldens and `background_test.go`'s build counts stay
  unchanged.
* **Mutation S6-1:** the first theme ignores `WithGlyphs`.
  `TestFirstThemeFollowsOptions`.

### Step 7: `RenderPlain` and `Help`

* **`type PlainViewer interface{ PlainView(width int) string }`.**
  `Model[M]` forwards it.
* **`(*Workspace) RenderPlain(width int) string`** lays the workspace out
  at `width` × the current height (or the fallback), then for each
  visible pane in focus-ring order writes (D6, 2026-10-08: every visible
  pane, the ring's first and then the rest in tree order):
  1. its title, flattened to one line;
  2. its `PlainView(width)`, or else its `View(width, h)` with escape
     sequences stripped (`ansi.Strip`), where `h` is its laid-out height,
     with trailing blank lines dropped;
  3. a blank line.

  Overlays and hidden panes are left out. It never touches the view cache.
* **`(*Workspace) Help() help.Model`** is bubbles' model, with:
  * `ShortSeparator` and `FullSeparator` from the glyph set's `Bullet`
    (D7, 2026-10-08: `FullSeparator` keeps bubbles' four spaces);
  * `Ellipsis` from its `Ellipsis`;
  * styles from the theme.
* **Tests:**
  * `TestRenderPlainGolden` across the tuitest matrix (new goldens,
    `plain-*`);
  * `TestRenderPlainNoEscapes`;
  * `TestRenderPlainOrder`;
  * `TestHelpUsesGlyphs`: the ASCII glyphs give no `•` or `…`.
* **Mutations:**
  * **S7-1:** `RenderPlain` keeps escapes. `TestRenderPlainNoEscapes`.
  * **S7-2:** `Help` uses `help.New()`. `TestHelpUsesGlyphs`.

### Step 8: `EnvCaps`, and case-folding on Windows

* **`EnvCaps(env Env, goos string, opts ...Option) Caps`** gives exactly
  the facts of a `WithDisabled` prober after the environment message:
  * the appearance variable and the overrides come through the existing
    options;
  * options that only concern probing are ignored, as documented;
  * `Profile` stays unknown;
  * JetBrains' reasons apply. (D9, 2026-10-08: a disabled prober sets
    them too.)
* **`Env.LookupEnv`** matches names regardless of case when the running
  `GOOS` is `windows`. An unexported `lookup(key, goos)` carries the
  rule, so it is tested on every host.
* **Tests:**
  * `TestEnvCapsMatchesDisabledProber`, over termcaptest's seven
    profiles;
  * `TestLookupFoldsOnWindows`;
  * `TestLookupExactElsewhere`.
* **Mutation S8-1:** `EnvCaps` skips the JetBrains reasons.
  `TestEnvCapsMatchesDisabledProber`.

### Step 9: `termsvc`'s byte forms and contexts

* **The byte forms:**
  * `(*Notifier) Sequence(x Notification) (string, SkipReason)` is the
    same encoder `Notify` uses, its sequence number included;
  * `CopySequence(c termcap.Caps, text string) (string, Route, error)`
    wraps the sequence for tmux as `Copy` does, and returns
    `ErrCopyTooLarge` over `MaxCopyBytes`.
* **The context forms:**
  * `(*Notifier) NotifyContext(ctx, x) tea.Cmd` and
    `CopyContext(ctx, c, text, opts...) tea.Cmd`;
  * `WithBackendTimeout(d)` and `WithCopyTimeout(d)`, each with a 5 s
    default;
  * a backend's `DeadlineExceeded` gives `Failed`, with `Err` wrapping it;
  * `Notify` and `Copy` call the context forms with
    `context.Background()`.
* **The documentation** says a CLI passes `WithPolicy(Always)` and feeds
  `EnvCaps` through `Update(termcap.CapsMsg{…})`, since it never sees
  focus.
* **Tests:**
  * `TestSequenceMatchesNotify`;
  * `TestCopySequence`;
  * `TestBackendDeadline`, with a backend that records its context's
    deadline;
  * `TestBackendTimeout`, with a backend that blocks on `ctx`.
* **Mutation S9-1:** the backend gets `context.Background()`.
  `TestBackendDeadline`.

### Step 10: `theme`, `internal/enum` and `tuitest`

* **`internal/enum`'s constraint** widens to `~uint8 | ~int`, an internal
  change.
* **`theme.Background`** gets `String`, `MarshalText` and
  `UnmarshalText`, with tokens `unknown`, `dark` and `light`.
  `UnmarshalText` also reads `auto` as `Unknown`, which
  `workspace.theme.set` accepts today.
* **`theme.BorderStyle`** gets the same, with its four tokens.
* **`tuitest`:**
  * `(Case) Profile() colorprofile.Profile` gives TrueColor for colour
    and ASCII without;
  * `(Case) Glyphs() glyph.Set` gives `glyph.For(c.UTF8)`;
  * `Fits(t T, s string, width int, m ansi.Method)` fails on any line
    wider than `width` under `m`. It is opt-in, because the WcWidth
    goldens are wider under `StringWidth` (eight of them, as re-measured
    on 2026-10-08).
  * The four mapping sites use `Profile` and `Glyphs`, with their goldens
    byte-identical.
* **Tests:**
  * `TestBackgroundText`;
  * `TestBorderStyleText`;
  * `TestCaseHelpers`;
  * `TestFits`, which passes on a fitting string, fails on a wide one,
    and respects the method.
* **Mutation S10-1:** `Case.Profile` gives ANSI256 for colour. Every
  migrated golden fails.

### Step 11: documents and examples

* **`docs/guides/commands.md`:**
  * `AllowIf` replaces `yesGate`;
  * `ArgsOf` and `WriteResult` replace `json.Marshal` and `fmt.Println`;
  * a section on `ParseArgs`, a generic `run <id> …` subcommand, and
    completion with cobra's `ValidArgsFunction` through `Complete`;
  * exit statuses through `ExitCode`;
  * a handler's panic is an `ErrPanicked` error, a Loop handler's
    included, and no longer crashes the TUI (added 2026-10-08).

  The excerpts are tied to the framework-example harness, whose programs
  gain these forms and their cases.
* **`docs/guides/building-workspaces.md`:** the options, `RenderPlain`,
  `Help`.
* **`docs/guides/terminal-capabilities.md`:** `EnvCaps`, `Sequence` and
  `CopySequence` for a CLI.
* **`docs/architecture.md`:** the package table rows.
* **`README.md`:** the Status bullet for `v0.8.0`.

**Done when** `make examples` passes, with the new cases.

### Step 12: close-out

* **Verification,** item by item.
* **This PLAN `complete`.**
* **The release** is the last step of
  [0014-PLAN-hardening.md](0014-PLAN-hardening.md), which follows.

## Verification

* **`command`:**
  * a panicking handler or gate is an error (Step 2);
  * every error has its A1 status (Step 2);
  * `ArgsOf`, `MustNew`, `AllowIf`, `GateFunc` and `WatchContext` work
    (Step 3);
  * `WriteResult` matches `CallMCP` (Step 4);
  * `ParseArgs`, `Params` and `Complete` work, and `FuzzParseArgs` runs
    (Step 5).
* **`workspace`:**
  * the options set the first theme and size (Step 6);
  * `RenderPlain` is linear and plain;
  * `Help` uses the glyphs (Step 7).
* **`termcap`:** `EnvCaps` equals a disabled prober's facts, and lookups
  fold on Windows (Step 8).
* **`termsvc`:** the byte forms match, and backends see a deadline
  (Step 9).
* **`theme` and `tuitest`:** the enum text forms work, the case helpers
  leave the goldens byte-identical, and `Fits` works (Step 10).
* **The rest:**
  * every mutation fails as named;
  * `make apicheck` shows additions only;
  * the guides' examples run;
  * Rule 2's checks are clean on macOS and the Windows test host.

## Rollout and Rollback

* **Rollout:** one commit per step. Released with W3 as `v0.8.0`.
* **Rollback:**
  * **Before the tag,** revert a step's commit.
  * **After it,** a defect is fixed in `v0.8.1`, and the tag is never
    moved.

## Execution Record

### Step 1: records

No deviation.

* **The refresh.** On 2026-10-08 the owner asked the agent to "Verify w2
  update related docs as needed, then proceed." The agent read every fact
  in the Scope table again on `7b4bded`. It measured the golden widths
  again, with a scratch program outside the tree, using x/ansi `v0.11.8`,
  the version `go.mod` requires. The results are in "Facts re-read before
  execution (2026-10-08)", with the amended lines in Steps 2, 5, 10 and
  11.
  * Two facts were wrong: the golden count, and `ParseArgs`'s missing
    receiver.
  * One fact is new: 0013's loop guard, which Step 2 now replaces.
* **No other record changes.** 0014-MADR's W2 already decides `recover`
  around every handler, and names `Registry.ParseArgs`. 0013's PLAN
  already names W2's `recover` as the end of its loop guard. Neither
  needs an amendment. [0014-PLAN-hardening.md](0014-PLAN-hardening.md)
  cites none of the facts that changed.
* **The approval** is the same message's "then proceed".
* **This PLAN** is `in-progress`, and its row in `docs/README.md` follows.

### Step 2: panics and exit statuses in `command`

#### Deviations

* **D1 (2026-10-08): a gate's panic says "gate".**
  * **Found:** the step fixes `PanicError`'s text as "command: <id>:
    handler panicked". It also reports a panicking gate with the same
    type. A gate's refusal would then read "command: refused: <id>: the
    gate failed: command: <id>: handler panicked". That names the
    handler, which never ran, in the error, the audit log and the MCP
    message.
  * **The owner's choice,** of three: `PanicError` gets an unexported
    field that is set when the gate panicked, and its text is then
    "command: <id>: gate panicked". The handler's text stays as
    planned. No exported name changes, so `apidiff` sees nothing.
    `TestGatePanicRefuses` checks the text.
  * **The others:** one neutral "command: <id>: panicked" for both,
    which changes the handler's planned text; or "handler panicked" for
    both, which misreports a gate's panic.
* **D2 (2026-10-08): three framework cases exit 3 on a refusal.**
  * **Found:** with `ErrRefused`'s status in place, `make examples`
    failed: "examples: cobra save notes => 1 stderr refused:
    session.save: exit 3, want 1", and the same for kong and urfave. Those
    examples pass the error through `launch.ExitCode`, which honours the
    new status. The flag example exits 1 itself (`os.Exit(1)`, which
    the commands guide quotes), so its case still passed. The gate passed
    on `4857ea0`.
  * **The owner's choice,** of two: `testdata/frameworks/cases.txt`
    joins Step 2. The three cases expect 3, and the file's header says
    why. The flag example and its case wait for Step 11, which moves
    the guide's exits to `ExitCode` as planned.
  * **The other:** move the flag example and its guide excerpt to
    `launch.ExitCode` now, which pulls part of Step 11 into this step.

#### What was built

* **`command/registry.go`:**
  * `ErrUnknown`, `ErrUnavailable` and `ErrRefused` are declared
    `error` and hold a `*codedError` with statuses 2, 1 and 3. Their texts
    are unchanged.
  * The new `ErrPanicked` holds status 2.
  * `codedError` has `Error` and `ExitCode`.
* **`command/panic.go` (new):**
  * `PanicError{ID, Value, Stack}` with `Error`, `Unwrap` and `ExitCode`.
    Its unexported `gate` field gives D1's text.
  * `runHandler` and `askGate` recover a panic into a `*PanicError` with
    `debug.Stack()`.
* **`command/args.go`:** `(*ArgError).ExitCode()` returns 2.
* **`command/dispatch.go`:**
  * `execute` calls `runHandler`, so Dispatch, Async, Run and CallMCP
    all get the error.
  * `onLoop`'s guard is gone, as the facts re-read before execution
    recorded. Its closure sends `execute`'s outcome, which now always
    arrives.
* **`command/gate.go`:** both of `decide`'s gate calls go through
  `askGate`. `verdict` wraps the `*PanicError` in the refusal.
* **The documentation:** `Handler` and `Gate` say what a panic becomes.
  The sentinels' block cites 0014-MADR A1 and `launch.ExitCode`.
* **The audit:** unchanged. `SlogAuditor` already logs `Err.Error()`
  only, and `TestPanicErrorHidesValue` proves the value stays out.
* **Tests:**
  * `command/panic_test.go` (new) has `TestHandlerPanicIsAnError`, with
    subtests for Loop through `Dispatch`, Async inside its command, `Run`
    in both modes, and `CallMCP`. It also has `TestGatePanicRefuses`,
    `TestPanicErrorHidesValue` (the error, the slog audit line and
    `CallMCP`'s text) and `TestSentinelsStillCompare`.
  * `command/exitcode_test.go` (new, package `command_test`) has
    `TestExitCodes`. Each bare and wrapped error goes through
    `launch.ExitCode` and `errors.As`. That covers the four sentinels,
    `*ArgError` and `*PanicError`, and five errors from real requests:
    an unknown ID, no origin, bad arguments, a handler's panic and a
    gate's panic.
  * `command/loop_test.go`: `TestLoopPanicReleasesCaller` is rewritten
    as the facts re-read before execution said.
* **`testdata/frameworks/cases.txt`:** D2.

#### Checks

* **Mutations,** on scratch copies of the tree. All six were killed:
  * **S2-1** (no `recover` around the handler):
    `TestHandlerPanicIsAnError/Loop_through_Dispatch` failed, and the
    test binary died with "panic: s3cr3t-panic-value [recovered,
    repanicked]".
  * **S2-2** (`Error()` includes the value): `TestPanicErrorHidesValue`
    gave "Error() holds the panic's value", the same for the audit log,
    and the same for CallMCP.
  * **S2-3** (`ErrRefused`'s status is 1): `TestExitCodes` gave
    "ErrRefused: launch.ExitCode(command: refused) = 1, want 3", and the
    same for the wrapped error and for "no origin".
  * **S2-4** (D1: a gate's panic says "handler"):
    `TestGatePanicRefuses` gave `PanicError = "command: p: handler
    panicked", s3cr3t-panic-value`.
  * **S2-5** (no `recover` around the gate): `TestGatePanicRefuses`
    died with the panic.
  * **S2-6** (the loop's panic passed on again):
    `TestLoopPanicReleasesCaller` gave "the loop recovered command:
    loop: handler panicked; want no panic".
* **Rule 2 on macOS:**
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the nine Go files>`: "9 file(s) clean".
  * `make lint`: clean. On the first run, errorlint refused a `!=`
    between two sentinels in `TestSentinelsStillCompare`, and it now
    uses `errors.Is`.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` ("No vulnerabilities found.") and
    `scripts/go-modules.sh --check` were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)". The
    sentinels' declared type was already `error`.
  * `make examples`: "clean", after D2. Before D2 it failed as D2
    records.
  * `make release-check`: "155 file(s) clean … apicheck, examples". The
    three new files are untracked, so the no-list run does not see them.
    The `FILES` run above covers them.
* **The Windows test host,** go1.27.1 windows/amd64, with the working
  tree and a bundle of `main` and its tags:
  * `make pre-add-check FILES=<the nine>`: "9 file(s) clean".
  * `make pre-add-check`: "155 file(s) clean".
  * `make lint`: "0 issues".
  * `make vuln`: clean.
  * `make examples`: "clean".
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: 18 ok.
  * The six panic and exit-status tests passed.

### Step 3: `ArgsOf`, `MustNew`, `GateFunc`, `AllowIf`, `WatchContext`

No deviation.

#### What was built

* **`command/args.go`:**
  * `MustNew[A]` has `New[A]`'s signature and panics with `New`'s error.
  * `ArgsOf[A](a A) (json.RawMessage, error)` marshals with JSON v2,
    `Deterministic(true)` and the new `durationMarshalers`. Its error
    is prefixed "command: arguments: ".
* **`command/decode.go`:** `durationMarshalers`, beside
  `durationUnmarshalers`, writes `d.String()`.
* **`command/builtin.go`:** the three built-ins use `MustNew`, and the
  loop that panicked on their errors is gone.
* **`command/gate.go`:** `GateFunc` with `Decide`, and `AllowIf(ok)`,
  which answers `AllowOnce` or `RejectOnce`.
* **`command/registry.go`:** `WatchContext(ctx)` waits on the change
  channel or `ctx.Done()`, and returns nil on the latter. `Watch()` is
  `WatchContext(context.Background())`.
* **Tests,** in `command/forms_test.go` (new):
  * `TestArgsOfDuration`: `{"name":"x","wait":"1m30s"}`, which a
    `MustNew` command's rule accepts and decodes back to 90 s;
  * `TestArgsOfDeterministic`: a five-key map, sorted and identical over
    20 runs, and an error for a func;
  * `TestMustNewPanics`;
  * `TestAllowIf`: both answers, and both through `Request.Gate` on a
    Destructive command from the CLI;
  * `TestGateFunc`;
  * `TestWatchContextEnds`. It returns nil within 100 ms of cancel, and
    `runtime.NumGoroutine()` falls back to its count before. A change
    still wakes it.
* **A test helper renamed.** revive's confusing-naming rule refused the
  exported `MustNew` beside the test helper `mustNew` in
  `command/decode_test.go`. The helper is now `testCommand`, in its 12
  uses across `decode_test.go`, `export_test.go` and `slash_test.go`.
  No assertion changed.

#### Checks

* **Mutations,** on scratch copies of the tree. All six were killed:
  * **S3-1a** (no duration marshaler, JSON v2's default):
    `TestArgsOfDuration` gave `json: cannot marshal from Go
    time.Duration within "/wait": no default representation`.
  * **S3-1b** (integer nanoseconds, JSON v1's form): `ArgsOf =
    {"name":"x","wait":90000000000}`, and "New's rule refused ArgsOf's
    arguments: command: argument /wait: is an integer, not string".
  * **S3-2** (`WatchContext` ignores `ctx`): "WatchContext did not
    return within 100 ms of cancel".
  * **S3-3** (`AllowIf` inverted, added): `TestAllowIf` failed on both
    answers.
  * **S3-4** (`MustNew` swallows the error, added): "MustNew did not
    panic".
  * **S3-5** (`Deterministic(false)`, added): `ArgsOf =
    {"b":2,"a":1,"c":3,"e":5,"d":4}; want sorted members`.
* **Rule 2 on macOS:**
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the nine Go files>`: "9 file(s) clean".
  * `make lint`: clean after the rename. Before it, the one finding was
    "command/args.go:83:6: confusing-naming: Method 'MustNew' differs
    only by capitalization to function 'mustNew' in
    command/decode_test.go".
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "158 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.1 windows/amd64, with the working
  tree and a bundle of `main` and its tags:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with `TestBuiltins`: passed.

### Step 4: `Format` and `WriteResult`

No deviation.

#### What was built

* **`command/result.go` (new):**
  * `Format`, with `FormatText` (the zero value) and `FormatJSON`. Its
    `String`, `MarshalText` and `UnmarshalText` go through
    `internal/enum`, with the tokens `text` and `json`.
  * `WriteResult(w, res, f)`:
    * **`FormatText`** writes `resultText`, with a newline added when
      the text lacks one. An empty text writes nothing.
    * **`FormatJSON`** writes `Value` as JSON, or `null`, and a newline.
    * An encoding error is wrapped as "command: result: …", and a write
      error as "command: writing the result: …".
    * An unknown `Format` is an error.
    * The documentation says why `Result.Cmd` is not run.
  * `resultText` is `CallMCP`'s rule, moved here: `Text`, or else `Value`
    as JSON. A `Value` that does not encode is an error even when there
    is text, as before.
  * `resultJSON` is deterministic JSON v2 with `durationMarshalers`. It
    returns the encoder's own error, so `CallMCP`'s message for one is
    unchanged.
* **`command/mcp.go`:** `CallMCP` calls `resultText`, and the file no
  longer imports JSON v2.
* **One consequence of the shared encoder.** A `time.Duration` in
  `Result.Value` used to make `CallMCP` an error result, because JSON v2
  has no default form for it. Now it is written as "1m30s". No golden
  holds a duration, and each existing golden is byte-identical.
  `Result.Value` still goes into `structuredContent` untouched, for the
  MCP server's own encoder.
* **Tests,** in `command/result_test.go` (new):
  * `TestWriteResult`: both formats over an empty result, `Text` only
    (with and without its newline), `Value` only, both, a bare duration
    and a `Cmd` that panics if run. Also a failing writer, a `Value` that
    does not encode, and an unknown `Format`.
  * `TestFormatText`: the tokens round-trip, `flag.TextVar` binds
    `--format=json`, and `--format=JSON` is refused.
  * `TestCallMCPUnchanged`: for four results, `CallMCP`'s text plus a
    newline equals `WriteResult`'s `FormatText` output. The step names
    the MCP goldens for this test; `TestExportGolden` already holds
    them, unchanged, and this test adds the equality.

#### Checks

* **Mutations,** on scratch copies of the tree. All six were killed:
  * **S4-1** (`FormatText` adds no newline): `TestWriteResult` gave
    `text only, text: wrote "done", <nil>; want "done\n"`, and four more.
  * **S4-2** (`Value`'s JSON wins over `Text`, added): for "both" as
    text, it wrote the value's JSON where it wanted `"done\n"`.
  * **S4-3** (no duration marshaler, added): `command: result: json:
    cannot marshal from Go time.Duration within "/a": no default
    representation`, and five more.
  * **S4-4** (`CallMCP` does not use `resultText`, added):
    `TestCallMCPUnchanged` gave empty `Content` for the `Value`-only
    results.
  * **S4-5** (`FormatJSON` writes no newline, added): `empty, json:
    wrote "null", <nil>; want "null\n"`, and six more.
  * **S4-6** (a write error is dropped, added): `text to a failing
    writer: <nil>`.
* **Rule 2 on macOS:**
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the three Go files>`: "3 file(s) clean".
  * `make lint`: clean.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "159 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.1 windows/amd64, with the working
  tree and a bundle of `main` and its tags:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with `TestExportGolden`: passed.

### Step 5: `ParseArgs`, `Params`, `Complete`

#### Deviations

* **D3 (2026-10-08): `-s=value` is accepted.**
  * **Found:** the grammar lists `-s v` for a short flag, and
    `--name=v` for a long one. The standard library's flag package
    accepts `-s=v` too, and so did the retired parser this step follows.
  * **The owner's choice,** of two: keep `-s=v`, documented in
    `ParseArgs`'s comment and tested in `TestParseArgs`.
  * **The other:** only `-s v`, with `-s=v` an unknown flag.
* **D4 (2026-10-08): the annotations are read leniently.**
  * **Found:** compiling all of `x-cli` means the rule reads
    `description`, `short`, `placeholder`, `group` and `hidden`. Read
    strictly, as `arg`, `rest`, `enum` and the rest are, a loaded MCP or
    ACP schema with one of those of the wrong type (`"description": 5`)
    would stop compiling, though it compiles today, and its source would
    lose the command.
  * **The owner's choice,** of two: those five are read leniently. A
    `description` or `x-cli` that does not decode leaves out every
    annotation, and an `x-cli` member that does not decode leaves out
    `x-cli`'s four. The schema still compiles, and `arg` and `rest` stay
    strict as before. `TestParams` checks it, with a schema holding
    `"description": 5` and `"short": 7`.
  * **The other:** strict, so such a schema is refused.
* **Recorded, not a deviation: `Complete` leaves hidden commands out.**
  The step says "the IDs offered on `SurfaceCLI`". The code also leaves
  out a command with `Hidden`, which `Command` documents as "runnable,
  but not listed by Available". The owner asked why hidden commands would
  be wanted there, and they are not.

#### What was built

* **`command/decode.go`:**
  * The rule gains `help`, `short`, `placeholder`, `group` and `hidden`,
    and `annotations` reads them (D4).
  * `arg` and `rest` are read strictly, as before.
* **`command/cliargs.go` (new):**
  * **`(*Registry) ParseArgs(id, args, o)`,** whose grammar is in its
    documentation:
    * `--name=v`, `--name v`, `-s v`, and `-s=v` (D3);
    * a bare boolean flag, and `--name=false`. A boolean never takes
      the next word;
    * repeats append to an array;
    * `--` ends the flags, and `-` is a positional value;
    * positional values are filled as `slashObject` fills them. A
      command whose only argument is a string takes the words joined
      with spaces, as `ParseSlash` takes its tail;
    * under `x-cli rest`, an unknown flag and a positional value too
      many stay in `Raw` only, as stray slash words do.

    `Raw` is `quoteArgs(args)`: POSIX single quoting, with `'"'"'` for a
    quote inside, which `splitWords` reads back. Errors:
    * an unknown ID wraps `ErrUnknown`;
    * an unknown flag gives "--x is not a known flag";
    * a flag without its value gives "/name: --name needs a value";
    * a bad value, and a missing required argument, give the
      `*ArgError`s `assign` and `prepare` give.
  * **`Param` and `Params(c)`:**
    * the fields the step lists;
    * `Rest` means a positional array, which takes every positional value
      left;
    * an array's `Enum` is its items' when it has none of its own.
  * **`(*Registry) Complete(id, args, partial)`:**
    * the IDs (above);
    * after a value-taking flag, its enum values;
    * for `--name=`, its values, or `true` and `false` for a boolean;
    * for another `-` partial, the visible `--name`s and `-s`s;
    * nothing for a positional, after `--`, or for an unknown command.

    Sorted and de-duplicated. The documentation says completion reads
    definitions only.
* **Tests:**
  * **`command/cliargs_test.go` (new):**
    * `TestParseArgs`: 24 accepted forms and 14 errors. A duration
      whose text is wrong passes `ParseArgs` and is refused by `Run` as
      an `*ArgError` for `/wait`: the rule does not read a schema's
      pattern, as for a slash line;
    * `TestParseArgsRaw`;
    * `TestParseArgsMatchesSlash`, over seven pairs;
    * `TestParams`: a struct with every tag, a command with no schema,
      a bad schema, and D4's schema;
    * `TestComplete`, with 17 cases.
  * **`command/fuzz_test.go`:** `FuzzParseArgs`, over four commands. It
    checks that nothing panics, and that accepted arguments come out the
    same when prepared again. It also checks that `Raw` splits back into
    the same words, which parse to the same request. Splitting reads
    runes, so the `Raw` check skips input that is not valid UTF-8.
    `scripts/go-fuzz.sh` finds it with no change to the Makefile: "3 fuzz
    targets ran clean in ./command".

#### Checks

* **The fuzz target** ran for 60 s with `-fuzz`: 1,261,071 executions,
  no failure.
* **Mutations,** on scratch copies of the tree. All nine were killed:
  * **S5-1** (`--` does not end the flags): "-- is not a known flag".
  * **S5-2** (a missing required argument passes): `deploy []: <nil>;
    want an error with "/target: is required"`.
  * **S5-3** (`Complete` lists IDs not on `SurfaceCLI`): the list gained
    `app.quit` and `keys.only`.
  * **S5-4** (a boolean takes the next word, added): "--verbose needs a
    value", and `"h" is not true or false`. It was run again after the
    lint fix below, with the same result.
  * **S5-5** (`Raw` quotes nothing, added): `TestParseArgsRaw` and
    `FuzzParseArgs` failed.
  * **S5-6** (hidden flags offered, added): `--token` and `--quiet`
    appeared.
  * **S5-7** (the rule drops `short`, added): "-n is not a known flag".
  * **S5-8** (D4 made strict, added): `TestParams` died on the schema.
  * **S5-9** (the one-string rule takes one word, added):
    `TestParseArgs` and `TestParseArgsMatchesSlash` failed.
* **Rule 2 on macOS:**
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the four Go files>`: "4 file(s) clean".
  * `make lint`: clean. On the first run, goconst counted four `"true"`
    literals in the package, two of them new, and unused found a test
    type left over. The new code now uses `strconv.FormatBool`, which
    leaves `frontmatter.go` and `loaddir.go` untouched, and the type is
    gone.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "161 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.1 windows/amd64, with the working
  tree and a bundle of `main` and its tags:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests and the slash tests, with `FuzzParseArgs`'s seeds:
    passed.

### Step 6: the workspace's options

#### Deviations

* **D5 (2026-10-08): `WithTheme` fixes the first theme, not whether the
  workspace follows.**
  * **Found:** rule 2 says "`WithTheme` wins over everything: the other
    options are then ignored". Today `WithTheme` and `WithThemeBuilder`
    are decided by order: the last decides whether the workspace follows
    the terminal (`workspace/workspace.go:198-208`).
    `WithTheme(t)` then `WithThemeBuilder(b)` means "start from `t`, then
    rebuild with `b`". `workspace/commands_test.go:429` (through
    `session`, `workspace/agent_test.go:130`) and
    `launch/consumer_test.go:523-524` use it, and the `commands-light.*`
    goldens and `launch`'s `Frame` golden depend on it. Read literally,
    the rule would turn the builder off and change them. apidiff cannot
    see that.
  * **The owner's choice,** of two: `WithTheme` fixes the first theme and
    the starting profile, background and glyphs, in any order.
    `WithGlyphs`, `WithProfile` and `WithBackground` are ignored for it.
    Whether the workspace then follows stays as today: the last of
    `WithTheme`, `WithThemeBuilder` and `WithGlyphThemeBuilder` decides.
    No golden changes, and the options' documentation says so.
  * **The other:** the literal rule, which changes the `v0.7` behaviour
    and those goldens, and needs a release note.

#### What was built

* **`workspace/workspace.go`:**
  * `GlyphThemeBuilder`, and the options `WithGlyphThemeBuilder`,
    `WithGlyphs`, `WithProfile`, `WithBackground` and `WithSize`.
  * New fields: `gbuilder`, `themed` and `glyphs`.
  * `New` starts from ANSI256, Unknown and Unicode glyphs, applies the
    options, and then builds the first theme:
    * under `WithTheme`, the starting profile, background and glyphs are
      the theme's (D5);
    * otherwise `build(w.glyphs)` makes it: the `GlyphThemeBuilder`,
      else the `ThemeBuilder`, else `theme.New`.
  * `rebuildTheme` calls `build` with the theme's glyphs, so a rebuild
    keeps the glyphs in use, as the default builder always did.
  * `WithSize` sets the size before the first `WindowSizeMsg`, with a
    negative size made 0, as that message's handling does.
  * The option comments say which options reach which builder, and D5's
    rule.
* **One change a program can see.** A `ThemeBuilder` now makes the first
  theme too, so it runs once in `New`. Before, the first theme was
  always `theme.New(ANSI256, Unknown, Unicode)`, and the builder ran only
  on the first message. This is the step's "the first theme through the
  builder". No golden changed, and neither did `background_test.go`'s
  counts, which compare builds before and after a message.
* **Tests,** in `workspace/options_test.go` (new):
  * `TestFirstThemeFollowsOptions`: no options; each fact alone; all
    three, in both orders; both builders, before and after the facts,
    with the `GlyphThemeBuilder` winning on the first theme and on a
    rebuild; and a `ThemeBuilder`'s first theme.
  * `TestWithThemeWins`: the facts are ignored in both orders, and a
    message does not rebuild. D5 is checked both ways: a builder after
    `WithTheme` follows from it, and a builder before it does not.
  * `TestWithSizeBeforeFirstMessage`: 40 by 6 before the message, 30 by
    4 after it, a negative size, and the default.
  * `TestGlyphsReachBuilder`: a `GlyphThemeBuilder` is given ASCII from
    `WithGlyphs`, at first and on a rebuild. A `ThemeBuilder`'s theme
    does not get them.

#### Checks

* **Mutations,** on scratch copies of the tree. All seven were killed:
  * **S6-1** (the first theme ignores `WithGlyphs`):
    `TestFirstThemeFollowsOptions` gave "WithGlyphs: first theme
    {p:4 bg:0 ascii:false}", and the same for "all three".
  * **S6-2** (`WithTheme` does not give the first theme, added):
    `TestWithThemeWins` failed.
  * **S6-3** (a `ThemeBuilder` wins, added): `TestFirstThemeFollowsOptions`
    gave "glyph builder [], builder [{p:5 bg:1 ascii:false}]".
  * **S6-4** (`WithSize` does nothing, added): "24 lines, 80 wide; want 6
    by 40".
  * **S6-5** (D5 reversed, added): "WithTheme then a builder: built [], …,
    following false".
  * **S6-6** (a rebuild passes the starting glyphs, added):
    `TestGlyphsReachBuilder` failed.
  * **S6-7** (the starting facts not taken from `WithTheme`'s theme,
    added): "profile TrueColor, bg 1".
* **Rule 2 on macOS:**
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the two Go files>`: "2 file(s) clean".
  * `make lint`: clean.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode. No golden changed.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "163 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.1 windows/amd64, with the working
  tree and a bundle of `main` and its tags:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with the background and golden tests: passed.

### Step 7: `RenderPlain` and `Help`

#### Deviations

* **D6 (2026-10-08): `RenderPlain` writes every visible pane.**
  * **Found:** the step says "each visible pane in focus-ring order". The
    focus ring leaves out a pane whose `Focusable()` is false, such as a
    status footer (`workspace/workspace.go:670-680`). Read literally, a
    plain render would drop that pane's text.
  * **The owner's choice,** of two: every placed pane with a non-empty
    area. The focus ring's panes come first, in its order (`WithFocusRing`'s,
    else the tree's), then the visible panes it leaves out, in tree order.
    Hidden and zero-size panes and overlays are left out.
  * **The other:** the focus ring only.
* **D7 (2026-10-08): `Help`'s `FullSeparator` stays four spaces.**
  * **Found:** the step takes both separators from the glyph set's
    `Bullet`. bubbles' `FullSeparator` default is four spaces, which hold
    no glyph (bubbles v2.2.1 `help/help.go:96`). Only `ShortSeparator`
    (" • ") and `Ellipsis` ("…") draw anything outside ASCII, as the fact
    table records.
  * **The owner's choice,** of two: `ShortSeparator` is " " + `Bullet` +
    " ", `Ellipsis` is the glyph set's, and `FullSeparator` stays as
    bubbles draws it.
  * **The other:** a bullet between the full help's columns too, which
    changes bubbles' layout.
* **D8 (2026-10-08): the step is paused for GO-2026-6604.**
  * **Found:** the step's code was complete, and its first gate run
    passed `make vuln`. The second run, after a lint fix, failed. That
    afternoon the vulnerability database had added GO-2026-6604
    (CVE-2026-56857, published 2026-10-08T22:31Z): "Root.Mkdir(All) can
    follow junctions out of the root on Windows in os", in every Go 1.27
    release before 1.27.2. govulncheck traces it to `tuitest.compare`
    calling `os.Root.WriteFile` (`tuitest/tuitest.go:187`), which W2 does
    not touch.
  * **Not this step's doing:** `make vuln` and `make release-check` fail
    on `main` as well. The commit gate runs the same precheck, so the
    step cannot be committed until it passes.
  * **The owner's choice,** of two: a record of its own first,
    [0016-MADR-go-1-27-2-for-go-2026-6604.md](0016-MADR-go-1-27-2-for-go-2026-6604.md),
    then this step resumes, with its code as it stands, uncommitted.
  * **The other:** a deviation step in this PLAN. That would mix a
    security patch into W2 and its release.
  * **Not offered,** being workarounds: excluding the advisory from
    govulncheck, or dropping `os.Root` from `tuitest`.
  * **What followed:**
    * The owner committed the step's code with the 0016 records, as
      `c71b4ba`, before the advisory was cleared.
    * 0016-PLAN-go-1-27-2-for-go-2026-6604 then moved both hosts to
      go1.27.2 and the floor to `go 1.27.2`, in `970e14e`.
    * Under that toolchain, every gate below passes, `make vuln`
      included.

    So this step resumes only to record what it built, with no change
    to its code.

#### What was built

* **`workspace/plain.go` (new):**
  * **`PlainViewer`.**
  * **`RenderPlain(width)`:**
    * it solves a layout of its own at `width` × the current height (80 ×
      24 before any size), with `layout.Solve`, and changes nothing in
      the workspace;
    * the panes come in `plainOrder`: the focus ring's first, then the
      rest in tree order (D6);
    * a pane's title and badge are flattened to one line, without
      escapes;
    * the body is its `PlainView(width)`, or else its `View(width, content
      height)`. Either way it is stripped of escapes, trailing spaces and
      trailing blank lines;
    * a blank line follows each pane, and a width of 0 or less gives "".
  * **`Help()`:** `help.New()` with `ShortSeparator` " " + `Bullet` + " ",
    the glyph set's `Ellipsis`, and the theme's styles: `Body` for keys,
    `Muted` for the rest. `FullSeparator` keeps bubbles' four spaces (D7).
* **`workspace/model.go`:** `Model[M].PlainView`. It gives the hosted
  model's `PlainView` when `*M` has one, which includes a value
  receiver's, and otherwise its `View()` stripped of escapes.
* **One rule beyond the step's text:** trailing spaces are dropped from
  each line, as well as trailing blank lines. A pane's view is padded to
  its width, and plain output is for a pipe or a screen reader.
* **Tests,** in `workspace/plain_test.go` (new):
  * **`TestRenderPlainGolden`:** the eight new `plain.*` goldens, over
    the tuitest matrix at 40 and 80. They were read before the commit:
    * at 40 the layout hides the sidebar;
    * the overlay is absent, and the footer comes last;
    * the output is the same in every colour and glyph case, as plain
      text must be.
  * **`TestRenderPlainNoEscapes`:** in colour, no escape and no trailing
    space. The view cache, the plan and the frame are unchanged. A
    wrapped bubble with no `PlainView` gives "s\nred line\n\n", and
    width 0 gives "".
  * **`TestRenderPlainOrder`:** tree order with the footer last;
    `WithFocusRing` first; a hidden pane left out; and a zoomed pane
    alone.
  * **`TestHelpUsesGlyphs`:** the ASCII set draws " * " and "~", and no
    "•" or "…", in the short, the truncated and the full help. The
    Unicode set draws " • ". `FullSeparator` is four spaces.

#### Checks

* **Mutations,** on scratch copies of the tree. All eight were killed on
  the final code:
  * **S7-1** (`RenderPlain` keeps escapes): "an escape sequence in
    …\x1b[1muser: hello\x1b[m…".
  * **S7-2** (`Help` uses `help.New()`): `the short help "alt+. next
    pane • alt+z zoom" has no " * "`.
  * **S7-3** (D6: the ring's order ignored, added): `TestRenderPlainOrder`
    failed for `WithFocusRing`.
  * **S7-4** (D6: the focus ring only, added): the golden lost the
    footer.
  * **S7-5** (trailing blank lines kept, added): the golden differs.
  * **S7-6** (`Model` does not forward the bubble's `PlainView`, added):
    the golden lost "(plain, www)".
  * **S7-7** (`RenderPlain` lays out the workspace itself, added): "RenderPlain
    changed the cache, the plan or the frame".
  * **S7-8** (bubbles' ellipsis kept, added): the truncated help ended in
    "…", not "~".

  On the first run, S7-6 removed only `Model`'s value-receiver check, and
  survived. That mutant was equivalent: `*M`'s method set holds `M`'s, so
  the pointer check still forwarded. The redundant check was removed from
  the code, and S7-6 was rewritten to remove the forward. It was then
  killed.
* **Rule 2 on macOS:**
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the three Go files>`: "3 file(s) clean".
  * `make lint`: clean. On the first run, modernize wanted
    `strings.SplitSeq` in a test loop, and it has it.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff` and `scripts/go-modules.sh --check` were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make vuln` failed on go1.27.1 with GO-2026-6604 (D8). On go1.27.2,
    in 0016's Step 3 run over the same tree, it gave "No vulnerabilities
    found." `make release-check` gave "166 file(s) clean … apicheck,
    examples".
* **The Windows test host:** 0016's Step 3 run, on go1.27.2, covered this
  step's code. It ran `make pre-add-check` (166 files), `make lint`,
  `make vuln`, `make examples` and `GOWORK=off go test -count=2
  -shuffle=on ./...`, each exit 0.

### Step 8: `EnvCaps`, and case-folding on Windows

#### Deviations

* **D9 (2026-10-08): a disabled prober sets the JetBrains reasons.**
  * **Found:** the step asks for `EnvCaps` to give "exactly the facts of
    a `WithDisabled` prober", and also says "JetBrains' reasons apply". A
    disabled prober never sets them: `start` returns for `p.disabled`
    (`termcap/prober.go:324-326`) before the JetBrains block (`:327-333`).
    So the two cannot both hold. No test or golden covers a disabled
    prober in JetBrains; the report goldens probe.
  * **The owner's choice,** of three: the JetBrains block moves before the
    disabled return. A disabled prober in JetBrains then marks the seven
    query facts Unknown, with `NotQueried` and `ReasonJetBrainsPaints`.
    `EnvCaps` is a disabled prober by construction, and the test asserts
    the reasons for the JetBrains profile itself, so mutation S8-1 can
    fail it.
  * **The others:** `EnvCaps` adds the reasons on its own, with the test
    allowing that one difference; or no JetBrains reasons, dropping the
    line and S8-1.

#### What was built

* **`termcap/envcaps.go` (new):** `EnvCaps(env, goos, opts...)`. It
  builds a prober from `opts` plus `WithDisabled()` and `WithGOOS(goos)`,
  so `goos` wins over a `WithGOOS` in `opts`. It calls `start(env)`,
  whose command it does not run, and returns `Caps()`. It is a disabled
  prober by construction. The documentation lists:
  * the options that apply: `WithAppearanceEnv` and `WithOverride`;
  * those that change nothing, which only shape a probe;
  * the hooks it does not run;
  * the unknown `Profile`, and the JetBrains reasons.
* **`termcap/prober.go`:** the JetBrains block runs before the disabled
  return (D9), and `WithDisabled`'s documentation says so.
* **`termcap/env.go`:**
  * `LookupEnv` is `lookup(key, runtime.GOOS)`;
  * `lookup` matches a name exactly, or, on `windows`, with
    `strings.EqualFold`;
  * the last match wins, as before, and an entry without "=" after the
    name is skipped;
  * `Getenv` follows, and so do `launch`'s reads of the environment.
* **Tests:**
  * `termcap/envcaps_test.go` (new, package `termcap_test`, since
    `termcaptest` imports `termcap`):
    * `TestEnvCapsMatchesDisabledProber` checks `reflect.DeepEqual` with
      a disabled prober's `Caps()` after `tea.EnvMsg`. It covers the
      seven termcaptest profiles, on linux, darwin and windows, with no
      options, with the appearance variable and an override, and with
      probe-only options and a contrary `WithGOOS`;
    * it then asserts the seven JetBrains facts (unknown, `NotQueried`,
      `ReasonJetBrainsPaints`), the appearance variable and override
      themselves, and an unprobed Kitty's empty `KittyKeyboard`.
  * `termcap/lookup_test.go` (new): `TestLookupFoldsOnWindows` and
    `TestLookupExactElsewhere`. They cover the last-wins rule, an empty
    value, an entry with no "=", a prefix of a name, and `Getenv` under
    the running GOOS.

#### Checks

* **Mutations,** on scratch copies of the tree. All six were killed by a
  failing test:
  * **S8-1** (D9 reversed: no JetBrains reasons when disabled):
    "JetBrains KittyKeyboard = {Value:unknown Origin:not-queried
    Reason:}; want … ReasonJetBrainsPaints", and the other six facts.
  * **S8-2** (no folding on Windows, added): `windows "PATH" = "",
    false`.
  * **S8-3** (folding everywhere, added): `linux "PATH" = "C:\bin",
    true; want "", false`.
  * **S8-4** (`EnvCaps` drops its options, added): "kitty, linux,
    appearance and override" did not match.
  * **S8-5** (`EnvCaps` ignores `goos`, added): "kitty, windows, no
    options" did not match.
  * **S8-6** (the first of a repeated name wins, added): `windows
    "Term" = "xterm"; want "dumb-later"`.

  S8-6's first two mutants did not compile: one left `slices` unused, the
  other ranged `slices.Values` with two variables. A build failure is not
  a kill, so it was rewritten until it compiled, and then it failed the
  test.
* **Rule 2 on macOS,** go1.27.2:
  * `gofmt -l`: nothing, after `gofmt -w` on the new test.
  * `make pre-add-check FILES=<the five Go files>`: "5 file(s) clean".
  * `make lint`: clean.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` ("No vulnerabilities found.") and
    `scripts/go-modules.sh --check` were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "166 file(s) clean … apicheck, examples". The
    three new files are untracked; the `FILES` run covers them.
* **The Windows test host,** go1.27.2 windows/amd64, where `LookupEnv`
  folds for real:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with `termcap`'s disabled, report and env tests:
    passed. The filter matched no `launch` test there; `launch`'s tests,
    which read the environment through `Getenv`, ran in the shuffled
    suite above.

### Step 9: `termsvc`'s byte forms and contexts

No deviation.

#### What was built

* **`termsvc/notify.go`:**
  * **`WithBackendTimeout(d)`,** with a 5 s default
    (`defaultBackendTimeout`); `d` of 0 or less leaves only the caller's
    context.
  * **`NotifyContext(ctx, x)`:** a backend is called under `bounded(ctx,
    d)`, and an error, its deadline's included, gives `SkipFailed` with
    the error in `Err`. `Notify` is `NotifyContext(context.Background(),
    x)`.
  * **`Sequence(x) (string, SkipReason)`:** it cleans `x` and returns
    `bytesFor(x)`. `NotifyContext`'s terminal path uses the same
    `bytesFor`, so the two share one encoder and one numbering.
    `Sequence` ignores `WithBackend`, as documented: its bytes are the
    terminal's.
  * `skip` takes whether the bytes are the terminal's. That replaces the
    old `n.backend == nil` test, with the same result on each path.
  * The `Notifier` documentation says what a CLI does: `WithPolicy(Always)`,
    `termcap.EnvCaps` through `Update(termcap.CapsMsg{…})`, and
    `Sequence`'s bytes written by the program.
* **`termsvc/clipboard.go`:**
  * `ErrCopyTooLarge`;
  * `WithCopyTimeout(d)`, with the same default;
  * `CopyContext(ctx, c, text, opts...)`, with `Copy` as
    `CopyContext(context.Background(), …)`;
  * `CopySequence(c, text) (string, Route, error)`: `ansi.SetSystemClipboard(text)`,
    the bytes `tea.SetClipboard` makes Bubble Tea write (bubbletea
    v2.0.10 `tea.go:821-822`), and inside tmux the same again through
    `Wrap`. Over `MaxCopyBytes` it gives "", `RouteNone` and
    `ErrCopyTooLarge`.
* **Kept as it was:** `Copy`'s too-large result is still
  `CopiedMsg{Status: Failed}` with no `Err`. The agent first gave it
  `ErrCopyTooLarge` too, which the step does not ask for.
  `TestCopyStatus` (`termsvc/services_test.go:72`) asserts no `Err`, so
  that change was taken back, and the test stands unchanged.
  `ErrCopyTooLarge`'s documentation says Copy reports the payload without
  it.
* **Tests,** in `termsvc/forms_test.go` (new):
  * `TestSequenceMatchesNotify`:
    * `Sequence` equals `Notify`'s bytes over two terminals, five
      protocols and five notifications, with escapes, a long body, an
      ID and an urgency;
    * one numbering across `Sequence` and `Notify` (n1, n2, n3);
    * each `SkipReason`, and `WithBackend` ignored.
  * `TestCopySequence`: it equals what `Copy` writes, plain and inside
    tmux, for "hello", "" and exactly `MaxCopyBytes`, with the same
    route. One byte over gives `ErrCopyTooLarge`.
  * `TestBackendDeadline`, under `testing/synctest`: the deadline both a
    backend and a clipboard see:
    * the 5 s default, and a timeout of their own;
    * the caller's sooner deadline, and their own timeout when it is
      sooner;
    * no timeout with the caller's deadline, and neither;
    * `Notify` and `Copy` on the 5 s default.
  * `TestBackendTimeout`, under `testing/synctest`:
    * a backend blocking on `ctx` gives `SkipFailed` with
      `DeadlineExceeded` after exactly 20 ms;
    * a clipboard gives `Failed` on `RouteBackend`;
    * a cancelled caller gives `Canceled`.

#### Checks

* **Mutations,** on scratch copies of the tree. All eight were killed by
  a failing test, none by a build failure:
  * **S9-1** (the backend gets `context.Background()`): "the default,
    backend: deadline … (false), want 5s".
  * **S9-2** (the clipboard gets `context.Background()`, added): the same
    for the clipboard.
  * **S9-3** (`Sequence` numbers apart, added): `notification 2:
    "\x1b]99;i=n1;b\a" has no i=n2`.
  * **S9-4** (no tmux wrap, added): `TestCopySequence` failed for tmux.
  * **S9-5** (no default timeout, added): "the default, backend: … want
    5s".
  * **S9-6** (`Sequence` honours `WithBackend`, added): "with a backend"
    got no bytes.
  * **S9-7** (no size limit, added): "one byte over" returned bytes.
  * **S9-8** (the caller's context dropped, added): "the caller's sooner
    deadline, backend: deadline 1s (true), want 200ms".

  Three needed a second form:
  * S9-1 and S9-2 first did not compile, because the bounded `ctx` was
    left unused, so they were rewritten to drop it.
  * S9-3 first survived. `Notify` now encodes through the same
    `bytesFor`, so breaking the numbering broke both sides alike. The
    test gained the explicit numbering check, and S9-3 then failed it.
* **Rule 2 on macOS,** go1.27.2:
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the three Go files>`: "3 file(s) clean".
  * `make lint`: clean. On the first run, revive's confusing-naming
    refused the unexported `sequence` beside `Sequence`, and it is now
    `bytesFor`. The mutations were run again on the renamed code, all
    killed.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` ("No vulnerabilities found.") and
    `scripts/go-modules.sh --check` were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "169 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.2 windows/amd64:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with `termsvc`'s notify and copy tests: passed.

### Step 10: `theme`, `internal/enum` and `tuitest`

No deviation.

#### What was built

* **`internal/enum`:**
  * a `Value` constraint, `~uint8 | ~int`, used by `Name`, `Marshal` and
    `Unmarshal`;
  * an `index` helper that also refuses a negative value, which an `int`
    enum can hold. A negative value has no name, and prints as its
    number.
* **`theme/theme.go`:**
  * `Background`'s `String`, `MarshalText` and `UnmarshalText`, with
    `unknown`, `dark` and `light`. `UnmarshalText` also reads `auto` as
    `Unknown`.
  * `BorderStyle`'s, with `light`, `rounded`, `heavy` and `double`.
  * No non-test code formats either type with `%v`, so adding `String`
    changes no output. No golden changed.
* **`tuitest/case.go` (new):** `Case.Profile` (TrueColor or ASCII),
  `Case.Glyphs` (`glyph.For(c.UTF8)`), and `Fits(t, s, width, m)`.
  `Fits` reports each line over `width` under `m`, by line number.
* **The four mapping sites** use `Profile` and `Glyphs`:
  * `workspace/golden_test.go`'s `caseTheme`;
  * `workspace/agent_golden_test.go`;
  * `workspace/commands_test.go`;
  * `theme/theme_test.go`'s `swatch`.

  Two of them drop their now-unused imports. Every golden is
  byte-identical: `git status` shows none changed.
* **Tests:**
  * `internal/enum/enum_test.go`: `TestEnumInt`, with names over `int`,
    and -1, -100 and 2 as having no name;
  * `theme/text_test.go` (new):
    * `TestBackgroundText`: the tokens, `auto`, five refused spellings
      that leave the value alone, -1 and 3, and `flag.TextVar`;
    * `TestBorderStyleText`;
  * `tuitest/case_test.go` (new):
    * `TestCaseHelpers`;
    * `TestFits`: a fitting string with escapes, two wide lines reported
      by number, and a family emoji that fits under `GraphemeWidth` and
      not under `WcWidth`. It uses the package's existing `recorder`.

#### Checks

* **Mutations,** on scratch copies of the tree. All seven were killed by
  a failing test:
  * **S10-1** (`Case.Profile` gives ANSI256 for colour): on a scratch
    copy, the full list of failures was `TestFramesGolden`,
    `TestChromeGolden`, `TestWidthMethodGolden`, `TestAgentSessionGolden`
    and `TestCommandsGolden` in `workspace`, `TestSwatchGolden` in
    `theme`, and `TestCaseHelpers`. That is every golden test the four
    migrated sites feed.
  * **S10-2** (`Case.Glyphs` inverted, added): the same goldens.
  * **S10-3** (`auto` refused, added): `"auto" = 1, theme: unknown
    Background "auto"`.
  * **S10-4** (`Fits` ignores the method, added): the emoji case.
  * **S10-5** (`Fits` fails an exact fit, added): "a fitting string
    failed".
  * **S10-6** (a negative value indexes `names`, added): `TestEnumInt`
    panicked, "index out of range [-1]".
  * **S10-7** (the border tokens out of order, added):
    `TestBorderStyleText`.

  They were run again after the lint fix below, all killed.
* **Rule 2 on macOS,** go1.27.2:
  * `gofmt -l`: nothing, after `gofmt -w` on two files (an import block's
    blank line, and a test's alignment).
  * `make pre-add-check FILES=<the ten Go files>`: "10 file(s) clean".
  * `make lint`: clean. On the first run, staticcheck ST1023 asked for
    `b := Dark` in a test.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "170 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.2 windows/amd64:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with the golden tests of `theme`, `tuitest` and
    `workspace`: passed.
