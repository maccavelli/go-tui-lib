---
status: proposed
date: 2026-10-07
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
    leaves out the value, because a panic value may hold a secret;
  * `Unwrap() error`, which returns `ErrPanicked`;
  * `ExitCode() int`, which returns 2.
* **`recover`** goes inside a helper around `c.Handler.Run`
  (`command/dispatch.go:160`). The helper sets `o.err` to a `*PanicError`
  holding `debug.Stack()`. `o.dur` is still set, and Async's `done` still
  runs. The same helper wraps `Gate.Decide`; a panicking gate gives a
  refusal wrapping the `PanicError`.
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
    `interface{ ExitCode() int }`;
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
  `hidden`, with `help` read from `description`.
* **`ParseArgs(id ID, args []string, o Origin) (Request, error)`:**
  * **The grammar:**
    * `--name=v`, `--name v` and `-s v`;
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
       ignored, and the documentation says so.
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
  visible pane in focus-ring order writes:
  1. its title, flattened to one line;
  2. its `PlainView(width)`, or else its `View(width, h)` with escape
     sequences stripped (`ansi.Strip`), where `h` is its laid-out height,
     with trailing blank lines dropped;
  3. a blank line.

  Overlays and hidden panes are left out. It never touches the view cache.
* **`(*Workspace) Help() help.Model`** is bubbles' model, with:
  * `ShortSeparator` and `FullSeparator` from the glyph set's `Bullet`;
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
  * JetBrains' reasons apply.
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
    goldens are wider under `StringWidth`.
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
  * exit statuses through `ExitCode`.

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

Not started.
