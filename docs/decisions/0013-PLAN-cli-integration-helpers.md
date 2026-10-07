---
status: proposed
date: 2026-10-07
associated-madr: "0013-MADR-cli-integration-helpers.md"
---
# Implement the `launch` package: decide, run with fallback, draw once, map the exit, release `v0.7.0`

Associated MADR: [0013-MADR-cli-integration-helpers.md](0013-MADR-cli-integration-helpers.md)

## Revisions

* **2026-10-07: to be revised before approval.** The MADR's amendment A1
  ([0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md)
  W1) changes what this PLAN builds. Once A1 is accepted, this PLAN is
  revised in place. Until then its steps describe the record before A1,
  and it is not to be executed.

  **Changes the revision must make:**

  * **The types:**
    * `Mode`, `Flags`, `StreamSource`, `FromSource`, `Config`, `Target`,
      the `Reason` tokens, `Decision.Glyphs`, the opaque `Option` and
      `ExitError`;
    * `Run` taking `...Option`.
  * **New pieces:**
    * `WithRegistry` and `command`'s detachable loop;
    * `WithRestorer`, `OnStart`, `Filter` as an `Option`, the epilogue,
      and the `Size()` override;
    * `launch/launchtest`.
  * **The colour profile:** from `colorprofile.Env`, not `Detect`. Rule 5's
    note on terminfo and tmux reads goes.
  * **Prerequisites,** each from 0014-MADR W2, in this PLAN or before it:
    * `glyph.Tier`;
    * `command`'s loop attach and detach;
    * enum text for `Mode` through `internal/enum`.
  * **Tests and mutations for each addition,** including:
    * a Loop command after the program ends running on the caller exactly
      once;
    * a `Restorer`'s bytes written on every outcome;
    * each tier-1 framework parsing `--tui` and `--mode`;
    * `ExitCode` honouring an `ExitCode()` method first.
  * **Step 6's probes:**
    * the tier-1 frameworks' examples;
    * L4 with a prober attached, checking mode 2031 is reset after the
      crash.

## Goal

A Go program that keeps its own CLI can start go-tui-lib's TUI from that
CLI safely:

* **It decides first.** `launch.Decide` says whether the TUI can run
  here, why, on which streams, with which colour profiles and glyphs, and
  at what size.
* **It runs only where it can.** `launch.Run` starts Bubble Tea only on
  an interactive decision, on the program's own streams, and never
  installs a signal handler.
* **It falls back cleanly.** `Run` reports a TUI that did not start as
  `ErrNotStarted`, and one that crashed after starting as `ErrCrashed`.
  In both cases the terminal is as it was, so the program can carry on
  in its CLI mode. After a crash it also gets the last good model.
* **It covers the plain path and the end.** `launch.Frame` draws a model
  once for plain output, and `launch.ExitCode` maps the end to the
  shell's statuses.

All of it is in the root module at `v0.7.0`, with no new module in the
build.

## Scope

### Facts this PLAN starts from (2026-10-07)

| Fact | Where it was read |
| :--- | :--- |
| the root is at `v0.6.0`; `main` is `47e11c4`, level with `origin/main` | `git tag`, `git status -sb` |
| 0011 Step 2 is committed as `47e11c4`; Steps 3–5 have not run | `git log`; [0011-PLAN-docs-accuracy-after-v0-5-0.md](0011-PLAN-docs-accuracy-after-v0-5-0.md) |
| Bubble Tea v2.0.10, colorprofile v0.4.3, x/ansi v0.11.8 are required; `github.com/charmbracelet/x/term` v0.2.2 is `// indirect` | `go.mod` |
| x/term v0.2.2 exports `IsTerminal`, `MakeRaw`, `GetState`, `SetState`, `Restore`, `GetSize`, `ReadPassword` | `x/term@v0.2.2/term.go:11-49` |
| no package of the library reads a terminal's state, `NO_COLOR`, `CI` or the locale | `grep -rn` of every Go file (0013-MADR G1) |
| `tea.OpenTTY` opens `/dev/tty` once and returns it as both input and output on Unix; on Windows it opens `CONIN$` and `CONOUT$` | `tty.go:130`; ultraviolet `tty_unix.go:15-21`, `ultraviolet/tty_windows.go:16-20` |
| on Windows the input reader is cancellable only when the input's descriptor is the process's stdin; any other file, `CONIN$` included, gets the fallback reader | ultraviolet `cancelreader_windows.go:28-40` |
| `Run`'s early returns at `tea.go:1045`, `:1055` and `:1110` come before `startRenderer` (`:1116`) and `Init` (`:1127`), and call no `shutdown` | `tea.go:999-1130` |
| `Init`, `Update` and `View` are called on the goroutine that calls `Run` (`:1127`; the event loop at `:1153`) | `tea.go` |
| `colorprofile.Detect` loads terminfo and, under `TMUX`, runs `tmux info`, unless the environment already gives TrueColor or `NO_COLOR` parses true, or the writer is not a terminal | `colorprofile@v0.4.3/env.go:33-53`, `:214`, `:248` |
| `internal/conformance` scans every package of every module, and asserts a fixed list was read: `glyph`, `layout`, `theme`, `tuitest`, `tuitest/internal/clash`, `workspace` | `internal/conformance/conformance_test.go:328-353` |
| tuitest renders `Golden(t, name, Matrix{Widths}, render func(Case) string)` over {colour} × {UTF-8} × widths into `testdata/golden/` | `go doc ./tuitest` |
| CI runs Linux, macOS and Windows; `-race` off Windows; `-shuffle` and `LC_ALL=C` on Linux | `.github/workflows/ci.yml:46-82` |
| the ultraviolet depguard rule is the model for a one-package rule: `$all` less `!**/<dir>/**` | `.golangci.yml:77-88` |

### Preconditions

1. **0011 is complete first.** Its Steps 3–5 edit `docs/guides/commands.md`,
   `README.md`, `AGENTS.md`, `docs/architecture.md` and `docs/README.md`,
   the files this PLAN's Step 5 edits too. Running both PLANs at once
   would interleave two PLANs' changes in one file.
2. **The owner accepts the MADR.** Its status goes from `proposed` to
   `accepted` in Step 1.

### In scope

| Step | Paths | Delivers | Released in |
| :--- | :--- | :--- | :--- |
| 1 | `docs/decisions/0013-*`, `docs/README.md` | the records accepted and indexed | — |
| 2 | `launch/doc.go`, `launch/decide.go`, `launch/terminal.go`, `launch/colour.go`, `launch/locale.go`, their tests; `go.mod`, `go.sum`; `.golangci.yml`; `AGENTS.md` Dependencies | `Streams`, `Want`, `Policy`, `Mode`, `Reason`, `Route`, `Decision`, `Decide`; x/term direct and confined | `v0.7.0` |
| 3 | `launch/run.go`, `launch/model.go`, `launch/errors.go`, `launch/filter.go`, their tests | `Run`, `ErrNotStarted`, `ErrCrashed`, `Filter` | `v0.7.0` |
| 4 | `launch/frame.go`, `launch/exit.go`, their tests and goldens, `launch/example_test.go` | `Frame`, `ExitCode`, the examples | `v0.7.0` |
| 5 | `internal/conformance/conformance_test.go`; `docs/guides/commands.md`, `docs/guides/building-workspaces.md`, `docs/guides/terminal-capabilities.md`; `README.md`, `docs/README.md`, `docs/architecture.md` | conformance reads `launch`; the documents | `v0.7.0` |
| 6 | scratch only; this PLAN's record | the live probes on macOS and the Windows test host | — |
| 7 | this PLAN's record | the release and its smoke test | `v0.7.0` |
| 8 | this PLAN, the MADR, `docs/README.md` | close-out | — |

### Out of scope

* **gobble's side:**
  * its `--tui` flag and its fallback wiring;
  * reattaching to or cancelling a turn that was running when the TUI
    stopped;
  * sending Bubble Tea's crash report to a log.

  Each belongs to gobble-cli's own records. The MADR (§3) names them.
* **Accessible mode, a registry-generated form, suspend, a pager,** and
  a guard for panics in the program's own goroutines: the MADR's §12.
* **`termmode` and `inline`:** the mode plans and Windows console
  helpers that `docs/architecture.md` lists as later records. launch
  restores only the state it saved before `Run`.
* **A PTY library** (0001-MADR §3). Real-terminal checks use `script` on
  macOS and `ssh -tt` to the Windows test host, in Step 6.

## Implementation Steps

### Rules

1. **A step starts when the previous one is committed.** The agent
   commits on `main` only when the owner asks in that turn, with `git
   commit --no-edit`. The owner pushes and tags.
2. **Checks.**
   * **For a step that changes Go:**
     * `gofmt -l` on the changed files;
     * `make pre-add-check FILES="…"`;
     * `make lint`, which runs `make modernize` first, for all three
       targets;
     * with `GOWORK=off`: `go test -race -count=1 ./...`,
       `go test -shuffle=on -count=2 ./...` and
       `LC_ALL=C go test ./...`;
     * the tests in workspace mode;
     * `go mod tidy -diff`, `make vuln` and
       `scripts/go-modules.sh --check`;
     * `make release-check`;
     * the Windows test host on a copy of the tree: `make pre-add-check`,
       `make lint`, `make vuln`, and
       `GOWORK=off go test -count=3 -shuffle=on ./...`.
   * **For every step:**
     * markdownlint and the link checker on changed Markdown;
     * the citation checker on changed records;
     * the identifier scan of the diff.
3. **Each mutation runs on a scratch copy of the tree,** and must fail
   its named test or check. The tree is never dirtied to prove a test.
4. **Anything this PLAN does not say stops the step for the owner.** It
   is recorded as a dated deviation, and the MADR is amended if a
   decision or a fact changes.
5. **Tests are hermetic.** No test reads the process's environment, its
   standard streams or a real terminal. Each passes `Streams` with an
   explicit `Env`, and fake streams through the `IsTerminal` override.
   Where a test needs a colour profile, it gives an environment that
   returns before colorprofile's terminfo and tmux reads:
   `COLORTERM=truecolor`, `NO_COLOR=1`, or a stream that is not a
   terminal (Facts).

### Step 1: records

* **The MADR's status becomes `accepted`,** dated, by the owner's word.
* **This PLAN**, `proposed` until approved.
* **`docs/README.md`:** the 0013 MADR row's status, and a row for this
  PLAN.

**Done when** the owner accepts the MADR and approves this PLAN.

### Step 2: `Decide`, and x/term made direct

**Files and API.** These are the MADR's §1 to §6, with the names left
open there settled here:

* **`launch/doc.go`:** the package documentation.
  * What the package is: deciding between a TUI and plain output, and
    running the TUI on the program's own streams.
  * What it never does: name `os.Stdout` or `os.Stderr`, install a
    signal handler, or set `AltScreen`.
  * The opt-in pattern of the MADR's §3, as prose.
* **`launch/decide.go`:**
  * `Streams{In io.Reader; Out, Err io.Writer; Env termcap.Env}`.
  * `Want`: `Auto` (the zero value), `Interactive`, `Plain`.
  * `Policy{Want Want; NoInputEnv string; UIOnErr, OpenTTY bool}`.
  * `Mode`: `ModePlain` (the zero value), `ModeInteractive`.
  * `Route`: `RouteNone` (the zero value); `RouteStream` (`In` for
    input, `Out` for drawing); `RouteErr` (drawing only); `RouteTTY`.
  * `Reason`, with a `String` method whose text names the cause
    without a variable's value:

    | Constant | `String()` |
    | :--- | :--- |
    | `ReasonRequested` | "plain output was requested" |
    | `ReasonNoInputEnv` | "the no-input variable is set" |
    | `ReasonDumbTerminal` | "TERM is dumb" |
    | `ReasonCI` | "CI is set" |
    | `ReasonInputNotTerminal` | "input is not a terminal" |
    | `ReasonOutputNotTerminal` | "output is not a terminal" |
    | `ReasonTerminal` | "a terminal is present" |

  * `Decision{Mode; Reason; In, UI Route; Profile, UIProfile
    colorprofile.Profile; UTF8 bool; Width, Height int}`.
  * `func Decide(s Streams, p Policy) Decision`. It calls an unexported
    `decide(s, p, goos string, open func() (in, out *os.File, err
    error))`, so a test can choose the `GOOS` and the opener. `Decide`
    passes `runtime.GOOS` and `tea.OpenTTY`.
  * The rules, in the MADR's order (§3):
    1. `Plain`;
    2. unless `Interactive`, `NoInputEnv`, then `CI` (set, not empty,
       neither `false` nor `0`);
    3. `TERM=dumb`, whatever `Want` says;
    4. the input route;
    5. the drawing route;
    6. otherwise interactive.
  * With `OpenTTY`, `decide` opens the controlling terminal only when a
    route needs it. It reads the terminal's size, then closes each
    distinct file once. On Unix one file serves as both input and
    output (Facts).
* **`launch/terminal.go`:**
  * `isTerminal(v any) bool`: the `interface{ IsTerminal() bool }`
    override first; otherwise `interface{ Fd() uintptr }` with
    `term.IsTerminal`. It never consults the file mode (MADR §4, P7).
  * `size(v any) (w, h int, ok bool)`, using `term.GetSize`.
* **`launch/colour.go`:**
  * `colourEnv(env termcap.Env) []string` returns a copy with two
    changes:
    * `NO_COLOR=1` appended when `NO_COLOR` is set and not empty, so
      colorprofile's `ParseBool` reads it as true;
    * `CLICOLOR_FORCE=1` appended when `FORCE_COLOR` is set, not empty,
      and neither `0` nor `false`, and `CLICOLOR_FORCE` is unset.
  * `profile(w io.Writer, terminal bool, env []string)
    colorprofile.Profile`. It calls `colorprofile.Detect(w, env)`. For
    a stream the override calls a terminal but that has no terminal
    descriptor (a test double, an SSH session), it adds `TTY_FORCE=1`
    (colorprofile `env.go:131`).
* **`launch/locale.go`:** `utf8Locale(env termcap.Env, goos string)
  bool`.
  * It reads the first set, non-empty value of `LC_ALL`, `LC_CTYPE`,
    `LANG`.
  * It is true when that value contains `utf-8` or `utf8` in any case.
  * With none of them set, it is true on `windows` and false elsewhere.
* **Size (MADR §6).**
  * Interactive: the size of the stream the TUI draws on.
  * Plain: `Out`'s width when `Out` is a terminal, else `COLUMNS` if it
    is a positive integer, else 0. `Height` is 0.

**The dependency.**

* **`go.mod`:** `github.com/charmbracelet/x/term v0.2.2` moves to the
  direct `require` block, in this step's commit, the one that adds its
  first import. Its version must not change.
  `GOWORK=off go list -m all` must list the same modules and versions as
  before; the diff of the two lists is recorded.
* **`.golangci.yml`:** a depguard rule `xterm`, modelled on
  `ultraviolet`. The rule:
  * covers `files: [$all, "!**/launch/**"]`;
  * denies `github.com/charmbracelet/x/term`;
  * gives the message "x/term only in launch
    (docs/decisions/0013-MADR-cli-integration-helpers.md §11)".
* **`AGENTS.md`, Dependencies:** one sentence. `launch` alone may import
  `github.com/charmbracelet/x/term`, which Bubble Tea already selects;
  depguard refuses it elsewhere (0013-MADR §11).

**Tests** (`launch/decide_test.go`, `launch/terminal_test.go`,
`launch/colour_test.go`, `launch/locale_test.go`):

* **`TestDecideRules`, a table test** over a fake stream type with
  `IsTerminal()`, covering:
  * each rule;
  * the order of the rules, with two causes present at once;
  * each `Reason`;
  * `CI=false`, `CI=0`, `CI=true`, and `CI=` (empty);
  * `NoInputEnv` named but unset, and set empty;
  * `TERM=dumb` under each `Want`;
  * `Interactive` skipping `NoInputEnv` and `CI`;
  * `UIOnErr` with `Out` piped and `Err` a terminal;
  * `OpenTTY` with a fake opener that succeeds, and with one that fails.
* **`TestDecideOpensOnlyWhenNeeded`:**
  * the fake opener is not called when both streams are terminals, or
    when `OpenTTY` is false;
  * each file it returns is closed exactly once.
* **`TestIsTerminalDevNull`:** `os.Open(os.DevNull)` is not a terminal,
  on every OS. This is P7 as a test.
* **`TestColourEnv`:**
  * `NO_COLOR` = `1`, `yes`, `x` and empty;
  * `FORCE_COLOR` = `1`, `true`, `0`, `false` and empty;
  * `FORCE_COLOR` with `CLICOLOR_FORCE` already set;
  * `NO_COLOR` with `FORCE_COLOR`;
  * the profile through `Detect`, for the cases that return before
    terminfo (Rule 5).
* **`TestUTF8Locale`:**
  * the precedence (`LC_ALL=C` with `LANG=en_US.UTF-8` is false);
  * `utf8` spelled in lower case;
  * none set, on `linux`, `darwin` and `windows`.
* **`TestDecideSize`:** a plain decision with `COLUMNS` = `100`, `0`,
  `-1` and `x`. A fake terminal has no descriptor and so no size; the
  terminal-size path is checked under a real terminal in Step 6 (L3).

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S2-1 | `colourEnv` drops the `NO_COLOR` normalisation | `TestColourEnv` (`NO_COLOR=yes`) |
| S2-2 | `FORCE_COLOR=0` forces colour | `TestColourEnv` |
| S2-3 | `CI=false` vetoes | `TestDecideRules` |
| S2-4 | the `TERM=dumb` rule moves inside the `Interactive` skip | `TestDecideRules` |
| S2-5 | `isTerminal` tests `os.ModeCharDevice` instead | `TestIsTerminalDevNull` |
| S2-6 | `LANG` read before `LC_ALL` | `TestUTF8Locale` |
| S2-7 | the drawing rule skips `UIOnErr` | `TestDecideRules` |
| S2-8 | the opener's files are not closed | `TestDecideOpensOnlyWhenNeeded` |
| S2-9 | a scratch copy imports x/term in `theme`, and again in a test file of `workspace` | `make lint` fails on depguard in both; with the `xterm` rule removed, neither does |

**Done when** Rule 2's checks are clean, S2-1 to S2-9 are killed, and
the module list is unchanged.

### Step 3: `Run`, with fallback

**Files and API:**

* **`launch/errors.go`:**
  * `var ErrNotStarted = errors.New("launch: the TUI did not start")`;
  * `var ErrCrashed = errors.New("launch: the TUI stopped")`.
  * A plain decision's error is
    `fmt.Errorf("%w: %s", ErrNotStarted, d.Reason)`. A Bubble Tea
    failure before `Init` is `fmt.Errorf("%w: %w", ErrNotStarted, err)`.
    A crash is `fmt.Errorf("%w: %w", ErrCrashed, err)`, so `errors.Is`
    still finds `tea.ErrProgramPanic` through it.
* **`launch/model.go`:**
  * `type state[M tea.Model] struct{ started bool; last M }`.
  * `type wrapped[M tea.Model] struct{ inner M; st *state[M] }`, with
    these methods:
    * `Init` sets `st.started`, then calls `inner.Init()`;
    * `Update` calls `inner.Update(msg)`. If the result is an `M`, it
      stores it in `st.last` and returns a `wrapped` around it.
      Otherwise it returns the result unwrapped, which `Run` reports;
    * `View` returns `inner.View()`.
  * All three run on `Run`'s goroutine (Facts), so `state` has no
    lock; `-race` checks it.
* **`launch/filter.go`:** `func Filter[M tea.Model](f func(M, tea.Msg)
  tea.Msg) tea.ProgramOption`. It returns `tea.WithFilter` around a
  function that unwraps a `wrapped[M]` before calling `f`. Its
  documentation says a plain `tea.WithFilter` in `opts` would receive
  launch's wrapper (MADR §7).
* **`launch/run.go`:** `func Run[M tea.Model](ctx context.Context, s
  Streams, d Decision, m M, opts ...tea.ProgramOption) (M, error)`.
  1. **A plain decision:** return `m` and the `ErrNotStarted` error.
     Nothing is created or written.
  2. **Routes:** take the input and the output from `d.In` and `d.UI`.
     For `RouteTTY`, call `tea.OpenTTY` (through the same seam as
     `decide`'s opener). On failure, return the `ErrNotStarted` error.
  3. **Save the terminal state:** for each route's stream, through an
     unexported seam `saver` with `save(v any) (restore func() error,
     err error)`. The real saver calls `term.GetState` on a stream with
     a terminal descriptor, and returns a restore that does nothing for
     any other stream. A test's saver records each call, so the restore
     paths are tested with fake streams.
  4. **Build the program** with the MADR §7 options, in this order:
     * `WithContext(ctx)`;
     * `WithInput(in)` and `WithOutput(ui)`;
     * `WithEnvironment(s.Env)`;
     * `WithColorProfile(d.UIProfile)`;
     * `WithoutSignalHandler()`;
     * `WithWindowSize(d.Width, d.Height)` when the drawing stream has
       no terminal descriptor;
     * then `opts`.

     The model is `wrapped[M]{inner: m, st: &state[M]{last: m}}`. The
     construction goes through a seam `newProgram`, which tests replace.
  5. **`p.Run()`, then classify:**

     | Outcome | Returns |
     | :--- | :--- |
     | `err == nil` | the unwrapped final model, nil |
     | `!st.started` | `st.last` and the `ErrNotStarted` error; the saved states restored |
     | `errors.Is(err, tea.ErrInterrupted)` or `ctx.Err() != nil` | the unwrapped final model, or `st.last` when that is nil, and `err` unchanged |
     | otherwise | `st.last`, the `ErrCrashed` error; the saved states restored |

  6. **Close the files** `RouteTTY` opened, each once.
  7. **Check the model's type:** if the final model is neither a
     `wrapped[M]` nor an `M`, return `st.last` and an error naming the
     type.

**Tests** (`launch/run_test.go`; a Unix-only `launch/run_unix_test.go`):

* **In-memory streams.** Run's tests use a fake input, a type wrapping
  an `io.Pipe` reader with `IsTerminal() bool { return true }`, and a
  fake output wrapping a `bytes.Buffer` with the same override.
  `Decide` then gives an interactive decision, and Bubble Tea, finding
  no terminal descriptor, enters no raw mode (Bubble Tea's
  `tty_unix.go:15-28`). The demo's
  `echo q | explicit` probe showed Bubble Tea reading a piped key
  (0013-MADR, More Information).
* **`TestRunPlainWritesNothing`:** a plain decision returns
  `ErrNotStarted` naming the reason. The `newProgram` seam is never
  called, and all three writers stay empty.
* **`TestRunQuits`:** a model that quits on the key `q` returns typed
  with a nil error.
* **`TestRunUsesTheGivenInput`:** the key arrives only through `s.In`.
  If `WithInput` were left out, Bubble Tea would read the process's
  stdin, and the test would end on its deadline.
* **`TestRunNotStarted`:** the `newProgram` seam returns a program
  whose `Run` returns an error without calling `Init`, as Bubble Tea's
  early returns do (Facts). The result wraps both `ErrNotStarted` and the cause, the
  `saver`'s restore is called, and the model returned is `m`.
* **`TestRunCrash`:** a panic in `Init`, in `Update` after a first good
  `Update`, in `View`, and in a command's goroutine. In each case:
  * the error wraps `ErrCrashed` and `tea.ErrProgramPanic`;
  * the model is the last good one;
  * the `saver`'s restore is called.
* **`TestRunEndsAreNotCrashes`:** each of these ends gives neither
  sentinel, and no restore:
  * a model returning `tea.Quit`;
  * a model returning `tea.Interrupt` (`tea.ErrInterrupted`);
  * a cancelled `ctx`;
  * a `ctx` past its deadline.
* **`TestRunOpensAndClosesTTY`:** `RouteTTY` through a fake opener;
  each file is closed once after `Run`, on success and on each failure.
* **`TestFilterSeesTheModel`:** `Filter[M]` receives an `M`, not the
  wrapper.
* **`TestRunSignalsStayTheProgramsUnix`** (Unix only):
  * The test installs its own `signal.Notify` for `SIGINT`, so that the
    process survives.
  * It sends itself `SIGINT` while `Run` is waiting.
  * The model quits on a timer `Cmd` 300 ms later, and the test
    expects a nil error. If launch left Bubble Tea's handler on, Bubble
    Tea would turn the signal into `ErrInterrupted` (`tea.go:664`).
  * The test stops its own notification at the end.

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S3-1 | a plain decision builds the program anyway | `TestRunPlainWritesNothing` |
| S3-2 | `WithInput` left out | `TestRunUsesTheGivenInput` |
| S3-3 | `WithoutSignalHandler` left out | `TestRunSignalsStayTheProgramsUnix` |
| S3-4 | `Init` does not set `started` | `TestRunCrash`, `TestRunNotStarted` |
| S3-5 | a cancelled `ctx` classified as a crash | `TestRunEndsAreNotCrashes` |
| S3-6 | the crash path returns the final model, not `st.last` | `TestRunCrash` (`Update` case) |
| S3-7 | no restore on a crash | `TestRunCrash` |
| S3-8 | `ErrCrashed` wraps with `%v` instead of `%w` | `TestRunCrash` (`ErrProgramPanic` not found) |
| S3-9 | `Filter` passes the wrapper | `TestFilterSeesTheModel` |

**Done when** Rule 2's checks are clean, `-race` included, and S3-1 to
S3-9 are killed.

### Step 4: `Frame`, `ExitCode`, examples

**Files and API:**

* **`launch/frame.go`:** `func Frame(m tea.Model, width, height int, p
  colorprofile.Profile) string`.
  1. It sends `m` a `tea.ColorProfileMsg{Profile: p}` through `Update`.
  2. It sends a `tea.WindowSizeMsg{Width: width, Height: height}`
     through `Update`.
  3. It returns `View().Content`.

  It runs no `tea.Cmd`, and does not call `Init` (MADR §8).
* **`launch/exit.go`:** `func ExitCode(err error) int`, the MADR's §9
  table in that order:

  | Error | Status |
  | :--- | ---: |
  | nil | 0 |
  | `ErrNotStarted` | 2 |
  | `tea.ErrInterrupted` | 130 |
  | `context.DeadlineExceeded` | 124 |
  | `context.Canceled` | 130 |
  | `tea.ErrProgramPanic` | 2 |
  | anything else | 1 |

  A wrapped `ErrCrashed` resolves through its cause.
* **`launch/example_test.go`:**
  * `ExampleDecide`, with an `// Output:` line: fake streams, a piped
    output, and the printed `Mode` and `Reason`.
  * `ExampleRun`: the MADR's §3 `--tui` pattern with both fallbacks.
    It is compiled, not run, as `workspace`'s program example is
    (`workspace/example_program_test.go:35-46`).

**Tests:**

* **`TestFrameGolden`:** a three-pane `workspace` across `tuitest.Golden`
  with `Matrix{Widths: []int{60, 100}}` and a height of 12, in
  `launch/testdata/golden/`. For the colour case the profile is
  `TrueColor`; otherwise `ASCII`. Glyphs come from
  `glyph.For(c.UTF8)` through `workspace.WithThemeBuilder`. The goldens
  are written with `-tuitest.update` and read before they are committed
  (`AGENTS.md`).
* **`TestFrameRunsNoCommand`:** a model whose `Init` and `Update` return
  a command that would panic. `Frame` returns without running either.
* **`TestExitCode`:** each row, bare and wrapped, including
  `ErrCrashed` around `ErrProgramPanic` (2) and `ErrCrashed` around a
  read error (1).

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S4-1 | `Frame` skips the profile message | `TestFrameGolden` (no-colour cases) |
| S4-2 | `Frame` runs `Init`'s command | `TestFrameRunsNoCommand` |
| S4-3 | `context.Canceled` maps to 1 | `TestExitCode` |
| S4-4 | `ErrNotStarted` checked after `ErrProgramPanic`, with a wrapped error holding both | `TestExitCode` |

**Done when** Rule 2's checks are clean, the goldens are read, `go doc
-short ./launch` lists exactly the MADR's API and this PLAN's names, and
S4-1 to S4-4 are killed.

### Step 5: conformance and the documents

* **`internal/conformance/conformance_test.go`:** `launch` joins the
  list of packages the scan must have read (Facts).
* **Mutations:**
  * **S5-1:** a scratch copy of `launch/run.go` names `os.Stderr`.
    `TestNoPackageOwnsTheTerminal` must fail on `launch`.
  * **S5-2:** a scratch copy sets `AltScreen` on a view in launch.
    It must fail in the same way.
  * **S5-3:** a scratch copy removes `launch` from the read list and
    skips the directory in the scan. The test must fail on the list.
* **`docs/guides/commands.md`:** a section "Start the TUI from your CLI",
  after "Run commands from your own CLI". It holds:
  * the MADR's §3 `--tui` example with both fallbacks;
  * `Decide`'s rules as a short list;
  * where the TUI draws (`UIOnErr`, `OpenTTY`);
  * `Frame` for plain output;
  * `ExitCode` for a program that refuses;
  * that `Streams` comes from Cobra's `cmd.InOrStdin()` and the like,
    from Kong's bound streams, or from `os.Stdin` in `main`, and that
    the program cancels `ctx` on its own signals.

  Its examples are compiled in a scratch module outside the repository,
  as 0012's were.
* **`docs/guides/building-workspaces.md`, "The program owns the
  program":** one paragraph pointing to that section for starting the
  program.
* **`docs/guides/terminal-capabilities.md`, "Do not probe without
  input":** a sentence. `launch.Run` always sets an input, so a prober
  under it may probe.
* **`README.md`:**
  * the current release is `v0.7.0`;
  * a Status bullet: "Starting the TUI from a program's CLI, since
    `v0.7.0`", naming `Decide`, `Run`'s fallbacks, `Frame` and
    `ExitCode`;
  * an "I want to…" row.
* **`docs/README.md`:** an "I want to…" row for the new guide section.
* **`docs/architecture.md`:**
  * the package graph line: `launch  start a TUI from a CLI →
    termcap; bubbletea, colorprofile, x/term`;
  * a table row naming the API;
  * the tree line;
  * Dependencies: x/term required and "Kept to one package by
    `depguard`";
  * "What is not here": the "later record" sentence about helpers goes.
    The CLI front end stays refused, with 0012.

**Done when** Rule 2's checks are clean, S5-1 to S5-3 are killed, every
guide example compiles, and a grep of the living documents for "later
record" next to "helpers" finds none.

### Step 6: live probes (scratch only)

The tests of Steps 2 to 4 cannot hold a real terminal, so this step
checks the behaviour that needs one, on a scratch module that requires
the tree through a `replace`. Nothing here is committed. The output goes
in this PLAN's record.

* **The probe program:** standard-library `flag` with a `--tui` flag; a
  workspace model; the MADR's §3 pattern; a CLI mode that prints "cli"
  and the session value it was handed. Before `Run` and after the
  fallback it prints whether the terminal state is the same
  (`term.GetState` before and after, compared byte for byte), so the
  check works on both systems.
* **The cases, on macOS** under `script -q /dev/null` for a pseudo-terminal:

  | Case | Run | Expect |
  | :--- | :--- | :--- |
  | L1 | `--tui`, no terminal (plain shell, `</dev/null \| cat`) | one `Warning:` line naming the reason, then `cli`, exit 0, no escape bytes on stdout, under 1 s |
  | L2 | `--tui` under `script`, `TERM=dumb` | the warning names `TERM is dumb`, then `cli` |
  | L3 | `--tui` under `script`, `q` typed | the TUI drawn at the pseudo-terminal's size (`Decide`'s `Width` and `Height`), no warning, exit 0 |
  | L4 | `--tui` under `script`, the model panicking after its first `Update` | Bubble Tea's crash report on stderr; the warning; `cli` with the last good model's session value; state the same; `stty -a` showing `icanon` and `echo` |
  | L5 | `--tui` under `script`, with Bubble Tea patched in a scratch copy to fail in `initInputReader` | the `ErrNotStarted` warning, `cli`, state the same, `stty -a` as before |
  | L6 | `--tui` with stdout piped and `UIOnErr`, under `script` | the UI on stderr, only the result on stdout |
  | L7 | `--tui`, `OpenTTY`, stdin piped, under `script` | the TUI on the terminal, keys from it |

  L5's patched Bubble Tea is a copy of the module cache's v2.0.10 under
  the scratch directory, with `initInputReader` returning an error. It
  is wired in by a `replace` in the scratch module only.
* **The cases on the Windows test host,** in a copy of the probe, with
  `ssh -tt` for a console: L1, L3, L4 and L6. L7 is run as well. On
  Windows `CONIN$` gets the fallback input reader, which cannot be
  cancelled (Facts), so after L7's TUI ends the probe checks that a line
  typed into the CLI mode reaches it.
  * **If it does not,** the step stops (Rule 4). The resolutions to
    offer the owner are: `OpenTTY` unsupported on Windows, recorded in
    the MADR; or a fix in launch that the record names.

**Done when** each case gives what the table expects, and L7 on Windows
is recorded either way.

### Step 7: the release

1. The owner commits, has the agent run the disclosure guard
   (`AGENTS.md`, Identifiers), pushes, and with CI green tags `v0.7.0`
   (annotated, `-m "v0.7.0"`).
2. The agent runs the consumer smoke test
   (`docs/guides/releasing.md`) against `v0.7.0`, from a scratch module
   outside the repository:
   * `go get github.com/maccavelli/go-tui-lib@v0.7.0`;
   * a program importing `launch` and `workspace` that runs `Decide`,
     prints the reason with no terminal, and exits by `ExitCode`;
   * `go mod tidy`, `go build`, a run with stdout piped;
   * `go list -m all` listing `github.com/charmbracelet/x/term v0.2.2`.
3. **Release notes** in the execution record: the new package, its API,
   the x/term promotion, and that nothing existing changed.

**Done when** the tag exists and the smoke test builds and runs.

### Step 8: close-out

* **Verification,** item by item, with output.
* **This PLAN `complete`,** with its row in `docs/README.md`.
* **The MADR,** amended only if Steps 2–7 found a fact that differs.

## Verification

* **`Decide` follows the MADR's §3 to §6.**
  * `TestDecideRules`, `TestColourEnv`, `TestUTF8Locale`, `TestDecideSize`
    and `TestIsTerminalDevNull` pass on Linux, macOS and Windows.
  * S2-1 to S2-8 are killed.
* **x/term is direct at v0.2.2.** `go list -m all` is unchanged, and
  depguard refuses x/term outside `launch` (S2-9).
* **`Run` follows the MADR's §7.**
  * It starts nothing on a plain decision.
  * It sets the input, the output, the environment and the profile.
  * It installs no signal handler.
  * It returns `ErrNotStarted` before `Init`, and `ErrCrashed` with the
    last good model after it.
  * It restores the saved state on both.
  * It leaves the user's own ends alone.
  * S3-1 to S3-9 are killed.
* **`Frame` and `ExitCode` follow the MADR's §8 and §9.** The goldens
  are committed and read, and S4-1 to S4-4 are killed.
* **`internal/conformance` reads and passes `launch`.** S5-1 to S5-3 are
  killed.
* **The live probes L1 to L7** give what Step 6 expects, on macOS and
  the Windows test host.
* **The guide's examples compile.**
* **Rule 2's checks are clean at every step,** on macOS and the Windows
  test host, and CI is green after each push.
* **`v0.7.0` is tagged,** and the consumer smoke test builds and runs.
* **The identifier scan of every step's diff finds nothing.**

## Rollout and Rollback

* **Rollout:** Steps 2 to 5 land on `main` one commit each, and
  `v0.7.0` is tagged after Step 6. The package is new, so no existing
  program changes behaviour until it imports `launch`.
* **Rollback:**
  * **Before a push,** each step is one commit to revert. Step 2's
    revert moves x/term back to `// indirect`.
  * **After `v0.7.0`,** removing `launch` is a breaking change, made only
    by a new record.
  * **A defect found after the tag** is fixed in `v0.7.1`. If `v0.7.0`
    must not be used, a `retract v0.7.0` directive with the reason goes
    in that release (`docs/guides/releasing.md`, "Retire a module",
    gives the directive's form). A tag is never moved or deleted.

## Execution Record

Not started.
