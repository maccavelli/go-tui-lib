---
status: accepted
date: 2026-10-07
decision-makers: owner
consulted: 0001-MADR-scaffold-charm-tui-library.md (§3, §6), 0005-MADR-terminal-capabilities-and-services.md (§1), 0006-MADR-command-registry.md, 0012-MADR-bring-your-own-cli.md (Decision Outcome item 7), Bubble Tea v2.0.10 and colorprofile v0.4.3 sources, gh, gum, huh, glow, mods, wish, fzf, Ink, Textual, Trogon, clig.dev, no-color.org, force-color.org
informed: pi-go
---
# Add a `launch` package that decides between a TUI and plain output, and runs the TUI on the program's own streams

## Context and Problem Statement

[0012-MADR-bring-your-own-cli.md](0012-MADR-bring-your-own-cli.md) made
go-tui-lib a TUI layer that a Go program adds to the CLI it already has.
Its Decision Outcome item 7 left the integration helpers for a later
record, giving two examples: launching the TUI from a CLI subcommand with
the caller's streams, and choosing between interactive and plain output.
This is that record.

A program on `v0.6.0` that adds a `tui` subcommand to its own CLI has to
make four choices the library does not help with:

1. **Whether a person is at a terminal.** If not, the TUI must not start.
2. **Which streams the TUI reads and draws on.** These must be the CLI's
   own streams, so that tests, SSH sessions and Cobra's `cmd.OutOrStdout()`
   all work.
3. **What to print instead, when the TUI cannot run.**
4. **Which exit status to return.**

Getting any of them wrong is costly, and the evidence below shows it.

The first program to need this states the rule for it. On 2026-10-07 the
owner said of gobble, the coding agent being built on go-tui-lib: "The
default when a user runs gobble is cli mode. To run in tui mode it should
use --tui". gobble's own records already say so:

* gobble-cli `docs/decisions/0005-MADR-v1-feature-scope.md`: the TUI is
  "Selected explicitly. It is not the default, and a TTY with no prompt
  does not start it."
* gobble-cli `docs/decisions/0008-MADR-native-cli-mode.md` D1: "The TUI stays explicit. It is never chosen by
  this resolution."

So for gobble, a terminal is never a reason to start the TUI. Only the
flag is, and the terminal decides whether the request can be met. When
it cannot be met, the owner added: "if --tui can't launch the tui for
whatever reason it should fallback to running in native cli node" (§3).
Asked whether a crash in the middle of a TUI session should fall back
too, the owner said: "scope this in. this is exactly the kind of functionality i want in gobble. graceful and seamless handling of edge cases."

### What a program gets from Bubble Tea's defaults

The probes in [More Information](#probes) ran a workspace program on
`v0.6.0` and Bubble Tea v2.0.10, from a shell with no controlling
terminal:

| Probe | Launch | What happened |
| :--- | :--- | :--- |
| P1 | `tea.NewProgram(m, tea.WithContext(ctx))`, stdin `/dev/null`, stdout a pipe | `Run` failed at once with "bubbletea: error opening TTY: … open /dev/tty: device not configured" |
| P2 | stdin and stdout passed with `WithInput`/`WithOutput`, stdin `/dev/null`, no deadline | **Never exited.** It was still running at 10 s, and stopped only when SIGTERM cancelled its context ("program was killed: context canceled") |
| P3 | same as P2, with a 2 s deadline | Exited with "program was killed: context deadline exceeded" |
| P2, P3 | (output) | 110 bytes of mode changes and terminal queries went into the stdout pipe: `ESC[?2026$p`, `ESC]11;?`, `ESC[?1049h`, …, and no frame |
| P4 | the same model, drawn once with no program (the plain path) | A 1278-byte Unicode frame with no escape sequences; 648 bytes of ASCII under `NO_COLOR=1 LC_ALL=C` |

The research agent's own run (P5, in [More Information](#probes)) also
showed what happens with a controlling terminal. With stdin redirected
from `/dev/null`, Bubble Tea opened `/dev/tty` and ran the full-screen TUI
anyway. With stdout piped, the alternate-screen and query sequences went
into the pipe while keys came from the terminal.

The source explains each result (Bubble Tea v2.0.10):

* **Output defaults to the process's standard output.** "if no output was
  set, set it to stdout" (`tea.go:624-627`).
* **Input falls back to the controlling terminal.** With no `WithInput`
  and a stdin that is not a terminal, `Run` opens the TTY itself, and
  fails if it cannot (`tea.go:1015-1026`). The upgrade guide says v2
  "always opens the TTY for input automatically".
* **Input given explicitly is read as it is.** Raw mode is entered only
  when the input is a terminal (`tty_unix.go:15-29`). A pipe at end of
  file is read and nothing more. The program waits for a quit that never
  comes, which is P2.
* **Size and resize come only from a terminal on output.** The output is
  asked for its size only when it is a terminal; otherwise the size is
  `WithWindowSize` or 0×0 (`tea.go:1050-1061`, `tty_unix.go:30-32`).
* **The colour profile comes from the output writer.** It is detected
  with `colorprofile.Detect(p.output, p.environ)` (`tea.go:1092`).
* **Signal handling.**
  * By default `Run` calls `signal.Notify` for SIGINT and SIGTERM
    (`tea.go:664`), unless `WithoutSignalHandler` is set
    (`tea.go:1029-1031`).
  * A resize listener calls `signal.Notify` for SIGWINCH whenever the
    output is a terminal (`tea.go:1146`, `signals_unix.go:17`). No
    option turns it off.
* **Panics are reported on the process's standard error.** A recovered
  panic writes "Caught panic" and the stack there, whatever `WithOutput`
  says (`tea.go:1303`, `tea.go:1328`). With `TEA_DEBUG` true it also
  creates `bubbletea-panic-<time>.log` in the working directory
  (`tea.go:1306-1314`).
* **A child process's stderr is fixed.** `tea.Exec` connects the child's
  standard error to the process's own, not to the program's
  (`exec.go:113`).
* **Terminal queries are skipped only when input is disabled**
  (`tea.go:1238-1243`). The workspace's own background query (`Init`,
  `workspace/workspace.go:300`) therefore reaches a piped stdout unless
  input is nil.
* **`WithoutRenderer` drops printed lines.** Its documentation says
  output is "plainly sent to stdout" (`options.go:97-104`), but the
  renderer it installs drops every printed line: `insertAbove` returns
  nil and writes nothing (`nil_renderer.go:21`). It also enters no raw
  mode.

Three Bubble Tea issues, all still open, show other programs hitting the
same walls:

* [#761](https://github.com/charmbracelet/bubbletea/issues/761):
  "open /dev/tty: no such device or address" when running in Docker with
  no tty. A maintainer answered: "Bubble Tea needs a tty in order to read
  input and render output".
* [#860](https://github.com/charmbracelet/bubbletea/issues/860):
  "Automatically open terminal when Stdout is not a terminal".
* [#1459](https://github.com/charmbracelet/bubbletea/issues/1459):
  "Terminal settings not restored after panics/crashes".

huh, Charm's forms library, has the same problem
([#718](https://github.com/charmbracelet/huh/issues/718), open). A form
whose every field was already given by a flag still opened `/dev/tty` and
failed "while telling Claude Code to use my CLI, but it can also occur
within any CI pipeline".

AWS's agentcore CLI fell through to its TUI when no flags were given, and
"hangs indefinitely" without a TTY
([aws/agentcore-cli#949](https://github.com/aws/agentcore-cli/pull/949),
merged). The fix was a guard before every one of its 17 TUI entry points.
The guard prints "This command requires an interactive terminal. Use
--help to see non-interactive flags." and exits 1.

### What the library gives a launching program today

The exported API of `v0.6.0`, read with `go doc -short` for each of the
ten packages, was evaluated against the four choices. These are the
gaps:

* **G1. Nothing decides whether a person is at a terminal.** A search of
  every Go file for `NO_COLOR`, `"CI"`, `IsTerminal`, `isatty`,
  `/dev/tty`, `colorprofile.Detect`, `LC_ALL` and `"LANG"` finds only two
  comments (`theme/theme.go:10`, `tuitest/tuitest.go:55`).
  * `termcap.FromEnv` names the terminal, multiplexer, editor and
    platform from the environment (`termcap/identity.go:129`). It knows
    nothing about the streams.
  * The library requires `github.com/charmbracelet/x/term` only
    indirectly (`go.mod:16`), so no package can ask whether a file is a
    terminal.
* **G2. Nothing builds a `tea.Program` on the caller's streams.** The
  workspace's example builds `tea.NewProgram(model{ws: ws})` with
  Bubble Tea's defaults. It is "compiled, not run: running it needs a
  terminal" (`workspace/example_program_test.go:35-46`). The guides
  never show `WithInput`, `WithOutput` or `WithEnvironment` used
  together.
* **G3. Nothing draws a model once, for plain output.** The research
  demo had to feed `Update` a `tea.ColorProfileMsg` and a
  `tea.WindowSizeMsg` by hand before calling `Render`. The workspace
  draws at 80×24 until a size arrives (`workspace/workspace.go:112`) and
  rebuilds its theme on a profile message (`workspace/workspace.go:428`).
* **G4. Nothing maps Bubble Tea's errors to an exit status.** `Run`
  returns `ErrInterrupted`, `ErrProgramKilled` and `ErrProgramPanic`
  (`tea.go:38-46`). The external context's error is wrapped in
  `ErrProgramKilled` (`tea.go:1164`). Each program works out 130, 124 and
  the rest for itself.
* **G5. Glyphs do not follow the locale.** The workspace starts with
  Unicode glyphs (`workspace/workspace.go:265`), and `glyph.For(utf8)`
  needs a caller that has read the locale (`docs/guides/building-workspaces.md:355`).
  No code reads `LC_ALL`, `LC_CTYPE` or `LANG`.
* **G6. The colour profile ignores most `NO_COLOR` values.**
  colorprofile reads `NO_COLOR` with `strconv.ParseBool`
  (`colorprofile@v0.4.3/env.go:116`). The probe P6 shows
  `NO_COLOR=yes` and `NO_COLOR=x` leave TrueColor in place; only a
  boolean such as `1` lowers the profile to ASCII. no-color.org says the
  variable counts "when present and not an empty string (regardless of
  its value)". colorprofile does not read `FORCE_COLOR` at all.
* **G7. The command guide prints a `Result` by hand.** Its CLI section
  ends each handler with `fmt.Println(res.Text)`
  (`docs/guides/commands.md:316-330`). That is a few lines a program
  writes once. This record leaves it there (Decision Outcome §12).

The design those gaps call for is the one each surveyed program builds
for itself; [More Information](#survey) has the survey.

* **Decide before starting.**
  * gh prompts only when stdin and stdout are both terminals:
    `return s.IsStdinTTY() && s.IsStdoutTTY()` in `CanPrompt`
    ([cli/cli](https://github.com/cli/cli/blob/trunk/pkg/iostreams/iostreams.go)).
  * clig.dev: "Only use prompts or interactive elements if stdin is an
    interactive terminal (a TTY)", and "If `--no-input` is passed, don't
    prompt or do anything interactive."
* **Keep the UI apart from the result.**
  * gum draws on standard error (`tea.WithOutput(os.Stderr)`), then
    prints the answer to standard output after `Run` returns
    ([gum v2.0.2](https://github.com/charmbracelet/gum/blob/v2.0.2/choose/command.go),
    lines 150 and 172).
  * mods does the same when stdout is a terminal, and turns input off
    when stdin is not one
    ([mods](https://github.com/charmbracelet/mods/blob/main/main.go),
    lines 85-91).
* **Map the end to the shell's conventions.** gum exits 130 on
  `tea.ErrInterrupted` and 124 on a timeout
  ([gum v2.0.2 `internal/exit`](https://github.com/charmbracelet/gum/blob/v2.0.2/internal/exit/exit.go)).

How should go-tui-lib help a program decide between its TUI and plain
output, and run the TUI on the program's own streams, while keeping the
TUI conventions in `AGENTS.md`?

## Decision Drivers

* **A program never hangs, and never takes over a terminal it was not
  given.** Without a person at a terminal, the TUI does not start (P1,
  P2, #761, agentcore#949).
* **The program owns its CLI, its streams, its signals and its screen**
  ([0012-MADR-bring-your-own-cli.md](0012-MADR-bring-your-own-cli.md);
  [0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)
  §6 rules 1 and 2). A helper takes the streams as arguments, and
  reaches no process-wide stream or signal itself.
* **The decision can be tested without a terminal,** and reports why it
  was made, so that a test or a `--debug` line can show it.
* **Works with any CLI framework,** by taking plain `io.Reader`/`io.Writer`
  values, a `context.Context` and the environment as strings. It
  imports no CLI framework (0012 Decision Outcome item 4).
* **Degrades by the standards users already set:** `NO_COLOR`,
  `CLICOLOR_FORCE`, `FORCE_COLOR`, `TERM=dumb` and the POSIX locale
  variables.
* **No module enters the build graph that is not there today**
  (0001-MADR §3).
* **Graceful and seamless.** A TUI that cannot start, or that stops
  unexpectedly, hands over to the program's CLI mode with the terminal
  as it was, and the program carries on where the TUI left off (the
  owner, quoted above).
* **Small and additive.** A minor release, no change to an existing API.

## Considered Options

* **A. A `launch` package:** a pure decision over the caller's streams
  and environment, a runner that starts Bubble Tea only on an
  interactive decision, a one-frame renderer and an exit-status mapping.
* **B. A guide only:** document the launch code each program writes, and
  ship none.
* **C. A program-options helper only:** a function that returns
  `[]tea.ProgramOption` for given streams, with no decision and no
  runner.
* **D. Plain output through Bubble Tea's `WithoutRenderer`:** always
  start a `tea.Program`, and switch the renderer off when output is not
  a terminal (Bubble Tea's `tui-daemon-combo` example).
* **E. A form generated from the registry,** in the manner of Trogon for
  Click, that lets a person fill in a command's arguments in a TUI and
  runs it.

## Decision Outcome

*Amendment A1, below, changes this outcome: native types for every
framework, the fixes for two holes in the fallback, and a colour profile
that spawns nothing. Where they differ, A1 holds.*

Chosen option: **"A. A `launch` package"**. It is the only option that
removes the hang and the terminal takeover (P1, P2) for every program,
not just for those that read a guide. It keeps the decision pure and
testable, and starts Bubble Tea only after the decision says a person
is at a terminal. It needs no module the build does not already select.

### 1. The package

A new package `launch` in the root module, at `launch/`, following the
layout rule that each capability is a top-level directory named after
its package (`AGENTS.md`). It imports `tea`, `colorprofile`, `termcap`
(for `termcap.Env`) and `github.com/charmbracelet/x/term` (§11), and no
other library package. It is not a nested module: it has no dependency
that the root does not already select.

The API below shows the shape this record decides. Doc comments, and
names finer than these, are the PLAN's to settle.

```go
package launch

// Streams are the program's own standard streams and environment. A
// Cobra command passes cmd.InOrStdin(), cmd.OutOrStdout(),
// cmd.ErrOrStderr(); main passes os.Stdin, os.Stdout, os.Stderr and
// os.Environ(). launch never reads a process-wide stream itself.
type Streams struct {
    In  io.Reader
    Out io.Writer
    Err io.Writer
    Env termcap.Env
}

// Want is the program's request, bound to its own flag.
type Want uint8

const (
    Auto        Want = iota // decide from the streams and environment
    Interactive             // e.g. --tui: skip the environment's vetoes, not TERM=dumb
    Plain                   // e.g. --plain, --no-input: never start the TUI
)

// Policy is how a program lets the TUI reach the terminal.
type Policy struct {
    Want Want
    // NoInputEnv names the program's own variable that forbids the TUI
    // when set and not empty, as gh's GH_PROMPT_DISABLED does.
    NoInputEnv string
    // UIOnErr lets the TUI draw on Err when Out is not a terminal, so that
    // `x=$(prog pick)` captures only the result (gum, mods).
    UIOnErr bool
    // OpenTTY lets the TUI read from and draw on the controlling terminal
    // when In or Out is not one (fzf, gum filter).
    OpenTTY bool
}

type Mode uint8   // ModePlain, ModeInteractive
type Reason uint8 // why the decision fell as it did (§3)
type Route uint8  // which stream the TUI uses: RouteOut, RouteErr, RouteTTY

type Decision struct {
    Mode      Mode
    Reason    Reason
    In, UI    Route              // where the TUI reads and draws (interactive only)
    Profile   colorprofile.Profile // for Out: the result, or the plain output
    UIProfile colorprofile.Profile // for the stream the TUI draws on
    UTF8      bool               // pick glyph.For(UTF8)
    Width     int                // cells; 0 when unknown (§6)
    Height    int                // rows; 0 when unknown
}

func Decide(s Streams, p Policy) Decision
func Run[M tea.Model](ctx context.Context, s Streams, d Decision, m M, opts ...tea.ProgramOption) (M, error)
func Frame(m tea.Model, width, height int, p colorprofile.Profile) string
func ExitCode(err error) int

var ErrNotStarted error // the TUI did not start: nothing drawn, the terminal as it was
var ErrCrashed error    // the TUI started, then stopped on its own: the terminal restored, the last good model returned
```

### 2. Streams come from the caller

`Streams` holds the program's streams and environment. A Cobra command's
`cmd.OutOrStdout()` is an `io.Writer` and fits as it is. An SSH session
fits the same way: wish's Cobra example sets `SetIn(sess)`, `SetOut(sess)`
and `SetErr(sess.Stderr())`
([wish v2.0.5](https://github.com/charmbracelet/wish/blob/v2.0.5/examples/cobra/main.go)).

The environment is a `termcap.Env`, the `KEY=value` form the library
already uses (`termcap/env.go`). In it the last value set wins, as in
Bubble Tea's own environment. launch never calls `os.Getenv`, so a test
and an SSH session each pass their own.

### 3. `Decide`: one pure decision, with its reason

`Decide` reads its arguments and the terminal state of the streams, and,
for the colour profiles, what `colorprofile.Detect` reads (§5). It writes
nothing. It opens the controlling terminal only when
`Policy.OpenTTY` allows it and a stream needs it. It then closes it at
once, to learn whether there is one. The rules, in order:

1. **`Want == Plain`:** plain, reason `Requested`.
2. **Unless `Want == Interactive`, the environment can veto.** A flag on
   the command line outranks the environment, as no-color.org says of
   `NO_COLOR`: "per-instance command-line arguments should override".
   The vetoes, in order:
   * `NoInputEnv` set and not empty: plain, reason `NoInputEnv`;
   * `CI` set, not empty, and neither `false` nor `0`: plain, reason
     `CI`. ci-info treats `CI=false` as "not CI"
     ([watson/ci-info](https://github.com/watson/ci-info/blob/master/index.js),
     line 18). Ink makes the same default: "Ink detects whether the
     environment is interactive based on CI detection … and
     `stdout.isTTY`"
     ([ink `render.ts`](https://github.com/vadimdemedes/ink/blob/master/src/render.ts),
     line 112). The terminal tests alone do not cover CI: the Buildkite
     agent runs each job under a pseudo-terminal unless `--no-pty` is
     given ("Do not run jobs within a pseudo terminal (default: false)",
     [buildkite/agent `clicommand/agent_start.go`](https://github.com/buildkite/agent/blob/main/clicommand/agent_start.go),
     line 578), so there stdin and stdout are terminals with no person
     at them.
3. **`TERM=dumb`:** plain, reason `DumbTerminal`, whatever `Want` says.
   It is a fact about the terminal, not a preference the flag can
   outrank. The `dumb` terminfo entry has no cursor movement at all
   (`infocmp dumb`: `am`, `cols#80`, `bel`, `cr`, `cud1`, `ind`), so a TUI
   there draws garbage. Emacs's shell mode sets it by default
   (`comint-terminfo-terminal`, "dumb",
   [emacs `lisp/comint.el`](https://github.com/emacs-mirror/emacs/blob/master/lisp/comint.el),
   line 527). clig.dev turns colour off for it, huh switches to
   accessible prompts for it
   ([huh `form.go`](https://github.com/charmbracelet/huh/blob/main/form.go),
   line 129), and colorprofile treats it as no terminal (`env.go:23`).
4. **Input.** The TUI reads `In` if `In` is a terminal. Otherwise it
   reads the controlling terminal if `OpenTTY` allows that and one can be
   opened. Otherwise the result is plain, reason `InputNotTerminal`.
5. **Drawing.** The TUI draws on `Out` if `Out` is a terminal. Otherwise
   it draws on `Err` if `UIOnErr` allows that and `Err` is a terminal.
   Otherwise it draws on the controlling terminal if `OpenTTY` allows
   that. Otherwise the result is plain, reason `OutputNotTerminal`.
6. **Otherwise:** interactive, reason `Terminal`.

The default `Policy{}` therefore asks what gh's `CanPrompt` asks: are
stdin and stdout both terminals? `UIOnErr` adds gum's and mods' layout,
and `OpenTTY` adds fzf's.

Rules 3 to 5 hold even under `Want == Interactive`. A TUI cannot run
without a terminal that can draw it, as P1 and P2 show, so `--tui` there
gives plain with the reason that says why.

**A TUI the user asked for falls back to the CLI when it cannot start,
or when it crashes.**
The owner, the same day: "if --tui can't launch the tui for whatever
reason it should fallback to running in native cli node."

* **The pattern.** A program whose default is its own CLI, as gobble's
  is, calls launch only when `--tui` was given, with
  `Want: Interactive`. If the TUI does not start, for any reason, the
  program prints one `Warning:` line on its standard error naming the
  reason. It then runs as if `--tui` had not been given. The reasons
  include:
  * a plain decision (no terminal, `TERM=dumb`);
  * a controlling terminal that will not open;
  * Bubble Tea failing before the TUI is up (§7);
  * the TUI crashing after it started: a panic Bubble Tea recovers, or
    a failure reading input (§7).
* **The CLI mode resolves as usual.** For gobble that is its line
  session, or print mode when stdin or stdout is not a terminal (gobble-cli
  `docs/decisions/0008-MADR-native-cli-mode.md` D1).
* **The warning keeps the fallback visible.** gobble's output rule puts
  diagnostics on standard error with a `Warning:` prefix and keeps
  standard output for the result (the same record, D3), so a script
  still gets the CLI mode's output. kubectl's `exec -t` does the same:
  it warns and carries on without a TTY.
* **After a crash, the CLI continues; it does not start over.** Running
  the request again could repeat work the TUI had already done: for an
  agent, a turn whose tools had already run. So the program hands the
  CLI mode what the TUI had reached, never the original input.
  * `Run` returns the last good model with `ErrCrashed` (§7), so the
    program can read that state from it.
  * For gobble the state is the session. The TUI is an ACP client of
    gobble's agent (gobble-cli
    `docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md`), and the CLI
    mode already resumes a session by id (`--session <path|id|prefix>`
    and ACP `session/load`, gobble-cli
    `docs/decisions/0008-MADR-native-cli-mode.md` D12). The CLI mode
    loads the TUI's session; it never re-sends the prompt the TUI sent.
  * A turn still running when the TUI stopped belongs to gobble's agent.
    Whether the CLI reattaches to it or cancels it is gobble's record to
    decide.
* **Only a crash falls back.** An end the user or the program chose ends
  the run, through `ExitCode`:
  * the model quitting;
  * Ctrl+C, as `tea.ErrInterrupted`;
  * the program's context ending, for example on SIGTERM, or on SIGHUP
    when the terminal goes away (gobble cancels on Interrupt, SIGTERM
    and SIGHUP; gobble-cli `docs/decisions/0008-MADR-native-cli-mode.md`
    D2).
* **Falls back once.** The CLI mode never relaunches the TUI, so a crash
  that would recur cannot loop.

`Run` reports a failure to start, a plain decision included, as an
error wrapping `ErrNotStarted`, and a crash after the start as one
wrapping `ErrCrashed`. The program makes two checks. The commands
guide's first launch example is this one:

```go
if cli.TUI {
    d := launch.Decide(s, launch.Policy{Want: launch.Interactive})
    final, err := launch.Run(ctx, s, d, newModel(sess))
    switch {
    case errors.Is(err, launch.ErrNotStarted):
        fmt.Fprintf(s.Err, "Warning: --tui unavailable (%v); using the CLI\n", err)
    case errors.Is(err, launch.ErrCrashed):
        fmt.Fprintf(s.Err, "Warning: the TUI stopped (%v); continuing in the CLI\n", err)
        sess = final.Session() // continue where the TUI was; never resend
    default:
        return exitStatus(final, err) // the user or the program ended it
    }
}
return runCLI(ctx, s, sess) // the program's own CLI mode
```

A program that would rather refuse returns `launch.ExitCode(err)`. That
is 2 for `ErrNotStarted` (§9), and for `ErrCrashed` the status of its
cause, which it wraps. agentcore's guard refuses (exit 1, naming
the non-interactive flags), and so does docker's TTY check ("cannot
attach stdin to a TTY-enabled container because stdin is not a
terminal",
[docker/cli `cli/streams/in.go`](https://github.com/docker/cli/blob/master/cli/streams/in.go),
line 74).

`Auto`, the zero value, stays for a program that starts its TUI on its
own when a terminal is there, as glow does with no arguments
([glow `main.go`](https://github.com/charmbracelet/glow/blob/master/main.go),
lines 212-262) and Pi does on a bare `pi` (Pi `main.ts:112-123`, as
gobble-cli `docs/decisions/0008-MADR-native-cli-mode.md` records it). A program with gobble's rule must set
`Want: Interactive`; `Policy{}` would start the TUI on any terminal.

The decision does not read `NO_COLOR`. Colour and interactivity are
separate questions (clig.dev; gh's `GH_FORCE_TTY` and colour rules are
separate too), so `NO_COLOR` lowers the profile (§5) and never stops the
TUI.

### 4. Terminal facts

A stream is a terminal when it has a file descriptor
(`interface{ Fd() uintptr }`) and `term.IsTerminal` says so. A stream may
also implement `interface{ IsTerminal() bool }`. That answer wins, so
that a test, or an SSH session's PTY wrapper, states it directly, much
as gh's `SetStdoutTTY` overrides do.

A file mode is not a terminal test. P7 shows `/dev/null` is a character
device (`os.ModeCharDevice` set) but not a terminal (`term.IsTerminal`
false). The standard library has no terminal test.

On Windows, `term.IsTerminal` is `GetConsoleMode` succeeding
(`x/term@v0.2.2/term_windows.go:16-19`).

* **mintty, Git Bash and MSYS2 pipes are not consoles,** so the decision
  there is plain (`InputNotTerminal` or `OutputNotTerminal`). That is
  the safe result: raw mode needs the console API, which those pipes
  lack. Docker's old advice for the same case was to run the program
  under `winpty`.
* **Ctrl+C under raw mode.** Raw mode clears `ENABLE_PROCESSED_INPUT`
  (`term_windows.go:27`), so Ctrl+C reaches the model as a key there too.

The controlling terminal is opened with Bubble Tea's `tea.OpenTTY`
(`tty.go:130`). That is `/dev/tty` read-write on Unix, and `CONIN$` and
`CONOUT$` on Windows (`ultraviolet/tty_unix.go:16`,
`ultraviolet/tty_windows.go:16-20`). Both Microsoft's console documentation and
Bubble Tea name these as the way to reach a console whose standard
handles are redirected.

### 5. Colour, for each stream

`Profile` is for `Out` and `UIProfile` is for the stream the TUI draws
on. Each is `colorprofile.Detect` of that stream, run with an environment
adjusted in two ways:

* **`NO_COLOR` set and not empty, whatever its value,** caps the profile
  at ASCII, as no-color.org defines it. This closes G6. ASCII keeps bold,
  and no-color.org says the standard covers colour only.
  theme already gives the ASCII profile that meaning
  (`theme/theme.go:10-11`).
* **`FORCE_COLOR` set, not empty, and neither `0` nor `false`,** when
  `CLICOLOR_FORCE` is unset, counts as `CLICOLOR_FORCE=1`. `NO_COLOR`
  still wins, as both standards say.
  * force-color.org: "When this variable is present and not an empty
    string (regardless of its value), it should force the addition of
    ANSI color." Rich follows it (`force_color != ""`,
    [rich `console.py`](https://github.com/Textualize/rich/blob/master/rich/console.py),
    line 966).
  * Node's supports-color reads the same variable as a level: `false`
    returns 0, and a number is that level, so `FORCE_COLOR=0` means no
    colour there
    ([chalk/supports-color `index.js`](https://github.com/chalk/supports-color/blob/main/index.js),
    lines 34-52). Excluding `0` and `false` keeps a user's Node setting
    from turning colour on, and matches how gh reads `CLICOLOR_FORCE`
    (`!= "" && != "0"`,
    [go-gh `pkg/term/env.go`](https://github.com/cli/go-gh/blob/trunk/pkg/term/env.go),
    line 167).
  * colorprofile does not read `FORCE_COLOR`; its issue asking for it is
    open ([charmbracelet/colorprofile#77](https://github.com/charmbracelet/colorprofile/issues/77)).

**What `Detect` reads besides the environment** (found while writing
the PLAN). On a terminal whose environment does not already give
TrueColor, and whose `TERM` is not `dumb`, `Detect` loads the terminfo
entry for `TERM`, and, when `TMUX` is set, runs `tmux info`, keeping the
highest of the three answers (`colorprofile@v0.4.3/env.go:44-50`,
`:214`, `:248`). Bubble Tea makes the same call when no profile is
given (`tea.go:1092`), so `Decide` does nothing a Bubble Tea program
does not already do. But `Decide` is not free of the file system or of
child processes. A test that must be the same on every host gives an
environment that returns before those reads: `COLORTERM=truecolor`, a
`NO_COLOR` value, or a non-terminal stream. Under wish, a client's
`TMUX` would make the server run `tmux info`. Such a program decides the
profile itself, from the session.

`Run` passes `UIProfile` with `tea.WithColorProfile`, so the TUI and the
program's own plain output agree. The research found the same need in
Bubble Tea's #860: colours vanished when the UI was on `/dev/tty` but the
profile was detected from a redirected stdout.

### 6. Glyphs and size

* **`UTF8`** comes from the first set, non-empty value of `LC_ALL`,
  `LC_CTYPE` and `LANG`, the precedence POSIX gives the locale
  categories. That value must name UTF-8 (`UTF-8` or `utf8`, in any case).
  * With none of them set, `UTF8` is true on Windows and false
    elsewhere.
  * Windows sets none of them. Go writes to a console as UTF-16 through
    `WriteConsole`, whatever the code page
    (`internal/poll/fd_windows.go:851` and `:880` in Go 1.27.1).
  * A POSIX system with no locale set runs in the C locale, which is not
    UTF-8. That is the case the `LC_ALL=C` CI leg tests
    (0001-MADR §5).
* **`Width` and `Height`.**
  * When the TUI's stream is a terminal, they are its size
    (`term.GetSize`).
  * For a plain decision, `Width` is `Out`'s terminal width when `Out`
    is a terminal. Otherwise it comes from `COLUMNS`, if that is a
    positive integer.
  * Otherwise both are 0. Rule 5 of `AGENTS.md` says nothing assumes a
    terminal size, so the program, not launch, picks the fallback (80
    columns is the convention: gh, glow).

### 7. `Run`: starts Bubble Tea only on an interactive decision

* **A plain decision.** `Run` returns an error wrapping `ErrNotStarted`
  and naming the decision's reason, before creating a `tea.Program`, and
  writes nothing. P1 and P2 become impossible through `Run`:
  * input is never a non-terminal (short of the `IsTerminal` override of
    §4, which a test sets);
  * Bubble Tea's implicit `/dev/tty` fallback (`tea.go:1015-1026`) is
    never reached, because `Run` always sets the input.
* **A TUI that fails to start.** On an interactive decision, `Run` also
  wraps `ErrNotStarted` around every failure before the model's `Init`
  runs. Those failures are:
  * the controlling terminal not opening;
  * Bubble Tea's own early returns: entering raw mode (`tea.go:1045`),
    reading the terminal size (`tea.go:1055`), and creating the input
    reader (`tea.go:1110`).

  Those returns all come before the renderer starts (`tea.go:1116`) and
  before `Init` (`tea.go:1127`). They do not restore the terminal.
  * On that path `Run`'s only deferred clean-up is cancelling its
    context (`tea.go:1013`).
  * `shutdown`, which restores the terminal (`tea.go:1266`), is reached
    only after the event loop.
  * So a failure after raw mode was entered would leave the terminal
    raw, and the CLI mode that follows would read keys unbuffered and
    unechoed.

  launch therefore saves the state of each terminal it hands to Bubble
  Tea (`term.GetState`, `x/term@v0.2.2/term.go:24`) before `Run`. On a
  failure before `Init` it puts that state back (`term.Restore`,
  `term.go:35`) and closes any terminal it opened. When
  `errors.Is(err, ErrNotStarted)`, the terminal is as it was, and the
  program can fall back safely.
* **How `Run` knows `Init` ran.** It wraps the model in an unexported
  type that records the call and forwards `Init`, `Update` and `View`.
  It unwraps the final model before returning it. Bubble Tea asserts no
  optional interface on a model (`tea.go:53-65`; nothing in the package
  type-asserts a `Model`), so the wrapper changes nothing it sees. A
  `tea.WithFilter` passed in `opts` would receive the wrapper, not the
  program's model, so the PLAN gives launch a filter option typed on
  `M` in its place.
* **A TUI that crashes.** `Run` wraps `ErrCrashed` around the error
  when `Init` ran and the run ended for any reason except these three:
  the model quitting (a nil error), `tea.ErrInterrupted`, or the end of
  the context the program passed (`ctx.Err()` set). The crashes are:
  * **A panic Bubble Tea recovers.**
    * One in `Init`, `Update` or `View` is recovered by `Run`'s deferred
      handler (`tea.go:1034-1041`).
    * One in a command's goroutine is recovered there
      (`tea.go:729-737`), and one in a sequence's or a batch's command
      by `recoverFromGoPanic` (`tea.go:901-948`, `:1319`).
    * Each ends in an error that wraps `tea.ErrProgramPanic`.
  * **A failure reading input,** which the input loop sends to `Run`
    (`tty.go:87-91`).

  On each path Bubble Tea's `shutdown` runs:
  * on a panic in `Init`, `Update`, `View` or a command, from
    `recoverFromPanic` (`tea.go:1299`);
  * on a panic in a sequence's or a batch's command, or an input
    failure, from the end of `Run` (`tea.go:1180`).

  Even when the program is killed, `shutdown` closes the renderer
  (`tea.go:1452-1464`), and the renderer writes the resets:
  * it leaves the alternate screen and shows the cursor;
  * it turns off bracketed paste, focus reports and the mouse modes;
  * it pops the kitty keyboard entry and resets modifyOtherKeys;
  * it clears the window title and the cursor, foreground and background
    colours it set (`cursed_renderer.go:172-245`).

  `shutdown` also restores the saved terminal modes
  (`restoreTerminalState`, `tty.go:33-38`). launch then puts back the
  state it saved before `Run`, as it does for `ErrNotStarted`.

  `Run` returns the **last good model** with `ErrCrashed`: the model
  after the last `Update` that returned, or the starting model if none
  did. Bubble Tea cannot return it after a panic in `Update`, because
  `Run`'s named result is never assigned on that path
  (`tea.go:999`, `:1034-1041`). The wrapper of "How `Run` knows `Init`
  ran" keeps it.

  Before `Run` returns, Bubble Tea writes its crash report ("Caught
  panic", then the stack) to the process's standard error (§10). launch
  cannot move it: rule 1 keeps launch away from `os.Stderr`.
* **Panics outside Bubble Tea.** A panic in a goroutine the program
  started itself, outside a Bubble Tea command, ends the process before
  anything can fall back. That is still out of scope (§12).
* **What it sets.** On an interactive decision `Run` builds the program
  with:
  * `WithContext(ctx)`;
  * `WithInput` and `WithOutput`, from the decision's routes (§3);
  * `WithEnvironment(s.Env)`;
  * `WithColorProfile(d.UIProfile)`;
  * `WithoutSignalHandler()`;
  * `WithWindowSize(d.Width, d.Height)` when the stream the TUI draws on
    is not a real terminal file, the test double case of §4.

  The caller's `opts` follow, so the caller can still add `WithFPS`, or
  replace any of these. A filter goes through launch's own option (see
  "How `Run` knows `Init` ran").
* **What it never sets.**
  * `View.AltScreen`: inline or full screen stays the program's choice,
    made in its own `View` (rule 2).
  * `WithoutCatchPanics`: Bubble Tea's recovery restores the terminal,
    which nothing else can do from outside the event loop (§10).
* **The controlling terminal it opened.** `Run` opens it for an
  `OpenTTY` route and closes it after `Run` returns. Bubble Tea's
  `shutdown` restores the terminal before that (`tea.go:1180`). If the
  terminal cannot be opened at `Run` time, `Run` returns an error
  wrapping `ErrNotStarted`, and writes nothing.
* **The result.** `Run` returns the final model as `M`, so a program
  reads its own exit choice or selection without a type assertion. The
  research demo carried an exit status out this way. If the final model
  is not an `M`, `Run` returns the zero `M` and an error saying so.
  `command.QuitRequestMsg` stays `struct{}`: the program decides what a
  quit means.

### 8. `Frame`: the model drawn once

`Frame(m, width, height, profile)` sends `m` a `tea.ColorProfileMsg`
carrying `profile`, then a `tea.WindowSizeMsg`, each through `Update`.
It returns `View().Content`. It runs no `tea.Cmd`, including the one
`Init` returns, so it sends no query and reads nothing.

This is the plain path for a TUI that has something worth printing: a
status board, a summary, a final screen. It is Ink's non-interactive
default, which writes "only the final frame", and glow's `notty` render.
A program with a better plain form, such as a `command.Result`'s `Text`
or JSON `Value`, prints that instead. launch does not choose for it.

### 9. `ExitCode`: Bubble Tea's ends as the shell expects them

| `err` | Status | Basis |
| :--- | ---: | :--- |
| `nil` | 0 | success |
| wraps `ErrNotStarted` | 2 | for a program that refuses rather than falls back (§3): the command was used where it cannot work. The standard `flag` package exits 2 on a usage error (`flag.go:1171`) |
| wraps `tea.ErrInterrupted` | 130 | 128 + SIGINT, the shell's status for Ctrl-C (TLDP, "Exit Codes With Special Meanings"); gum's `StatusAborted` |
| wraps `context.DeadlineExceeded` | 124 | GNU `timeout`'s status, and gum's `StatusTimeout` |
| wraps `context.Canceled` | 130 | under rule 2 a cancel is how the caller's interrupt arrives (`signal.NotifyContext`) |
| wraps `tea.ErrProgramPanic` | 2 | the status of an unrecovered Go panic (`runtime/panic.go`, `fatalpanic`, `exit(2)`) |
| anything else | 1 | |

A program whose cancel came from SIGTERM, and that wants 143, checks for
that before calling `ExitCode`. sysexits codes are not used: FreeBSD's
manual says the interface "has been deprecated … Its use is
discouraged".

### 10. The TUI conventions, applied to a package that starts Bubble Tea

launch is the first package that builds a `tea.Program`. The rules hold
for launch's own code with no exception, and `internal/conformance`
checks it like every other package:

* **Rule 1, output where the caller says.** launch never names
  `os.Stdout` or `os.Stderr`. It always passes `WithOutput`, so Bubble
  Tea's default output (`tea.go:624-627`) is never reached.
* **Rule 2, the caller owns the screen and the signals.**
  * launch passes `WithoutSignalHandler` every time, so SIGINT and
    SIGTERM stay the program's. Ctrl+C reaches the model as a key in raw
    mode (`tea.go:656-659`; `term_windows.go:27`).
  * A program cancels `ctx` from its own `signal.NotifyContext`.
  * launch never sets `AltScreen`.

Three behaviours belong to Bubble Tea, not to launch. Every program that
starts Bubble Tea has them, with or without launch, so the record states
them rather than granting an exception:

* **The resize listener** calls `signal.Notify(…, SIGWINCH)` when the
  output is a terminal (`signals_unix.go:17`). No option turns it off,
  and a full-screen TUI needs it.
* **A recovered panic is reported on the process's standard error,**
  and with `TEA_DEBUG` also to a file in the working directory
  (`tea.go:1303-1314`).
* **`tea.Exec` connects a child's stderr to the process's own**
  (`exec.go:113`).

A program that needs these elsewhere passes `tea.WithoutCatchPanics()`
in `opts`, and takes on restoring the terminal itself. Bubble Tea warns
that the terminal is then left "in a fairly unusable state after a
panic" (`options.go:79-82`).

### 11. One dependency made direct

`github.com/charmbracelet/x/term` becomes a direct requirement of the
root module. It is already in `go.mod` at v0.2.2, `// indirect`
(`go.mod:16`), and `go list -m all` selects that version through Bubble
Tea. Its terminal test is the one Bubble Tea itself uses
(`tty_unix.go:17`), so making it direct adds no module and no version to
the graph. `golang.org/x/term` is not in the graph, and is not added.

A depguard rule like ultraviolet's (`.golangci.yml:77-88`) lets only
`launch` import x/term. Only launch then asks the terminal for its
state, and every other package stays free of it. AGENTS.md's
Dependencies section names the package and points here.

### 12. Not in this record

* **Printing a `command.Result`.** It stays in the commands guide
  (G7). The guide gains the launch path beside it.
* **Accessible mode.** huh's accessible prompts, gh's
  `GH_ACCESSIBLE_PROMPTER` and Bubble Tea's open issue on screen readers
  ([#780](https://github.com/charmbracelet/bubbletea/issues/780)) all
  point to a later record. Until then a program maps its own
  `ACCESSIBLE`-style variable or flag to `Want: Plain`.
* **A registry-generated form (option E).** Its own later record, which
  can build on `Run`.
* **A terminal-restore guard** for panics in the program's own
  goroutines (Bubble Tea's #1459). Crashes Bubble Tea recovers are in
  scope (§7, `ErrCrashed`). It would need the terminal's saved
  state from inside the event loop, which Bubble Tea does not export.
* **Suspend (Ctrl+Z).** The model returns `tea.Suspend`, as now. Bubble
  Tea cannot suspend on Windows (`tty_windows.go:62`).
* **A pager for plain output** (gh, git, crush). That is the program's
  choice of `Out`.
* **A PTY library for tests.** 0001-MADR §3 requires a record for one.
  The interactive path is tested through the `IsTerminal` override and
  `WithWindowSize`, and checked by hand under a real terminal (see
  Confirmation).

### 13. Release

launch adds a package and changes no exported API. It is released as
`v0.7.0`.

### Consequences

* Good, because a program that calls `Decide` and then `Run` cannot hang
  on a missing terminal, cannot draw into a pipe, and cannot take over a
  terminal its streams did not give it, unless it opts in with
  `OpenTTY`.
* Good, because the decision is pure and carries its reason. A program
  can test its own CLI's TUI path with fake streams, and can say why it
  printed plainly.
* Good, because every CLI framework fits: launch sees only readers,
  writers, a context and strings.
* Good, because `NO_COLOR` behaves as its standard says, and
  `FORCE_COLOR` as its standard says except where Node reads `0` and
  `false` as off. That closes colorprofile's gaps.
* Good, because no module is added: x/term is promoted at the version
  already selected.
* Bad, because a program must cancel `ctx` on SIGTERM itself. Without
  that, `kill` ends the process the Go way (`os/signal/doc.go:38-39`: "A
  SIGHUP, SIGINT, or SIGTERM signal causes the program to exit"). The terminal
  is then left in raw mode, where Bubble Tea's default handler would have
  quit cleanly. The guide shows `signal.NotifyContext` in every example.
* Bad, because Bubble Tea's own writes to standard error on a panic, its
  `TEA_DEBUG` file and `Exec`'s stderr remain. launch can only document
  them (§10).
* Bad, because the CI veto can stop a TUI in a CI job that does give a
  terminal. The program, or its user, passes `Want: Interactive` (for
  example `--tui`) or sets `CI=false`.
* Good, because `--tui` never leaves the user without a working
  program. Where the TUI cannot start, the CLI mode runs after one
  warning line, and the terminal is as it was.
* Good, because a TUI that crashes mid-session hands over to the CLI
  mode at the point it had reached. The terminal is restored and the
  last good model is returned, and for gobble the same session
  continues.
* Bad, because Bubble Tea's crash report, the panic and its stack,
  still reaches the process's standard error before the fallback's
  `Warning:` line. Only the program can redirect it, and it is
  gobble's choice whether to.
* Bad, because a crash in code the TUI and the CLI share, such as the
  agent, can recur in the CLI mode. The fallback happens once and never
  loops, so the second crash ends the run.
* Bad, because a script that passes `--tui` with no terminal gets the
  CLI mode's output, not an error. The `Warning:` line on standard error
  is the only sign.
* Bad, because launch wraps the model to see `Init`, so a
  `tea.WithFilter` in `opts` would see the wrapper. launch's own filter
  option replaces it (§7).
* Good, because `TERM=dumb` never gets a full-screen TUI, even on
  request.
* Bad, because `Auto` is the zero value, so a program whose TUI is
  opt-in must remember `Want: Interactive`. The guide's first example
  shows it.
* Bad, because mintty and Git Bash without `winpty` get plain output
  even when a person is there (§4).
* Neutral, because `Frame` runs no commands, so a model that fills itself
  in through `Init` draws its empty state. Such a program prints its own
  plain form.

### Confirmation

The PLAN that implements this record verifies:

* **`Decide`** with a table test over fake streams (the `IsTerminal`
  override) and environments. Each rule of §3 and its order, each
  reason, `CI=false`, `NO_COLOR=yes`, `FORCE_COLOR` against `NO_COLOR`,
  `FORCE_COLOR=0` and `FORCE_COLOR=false`,
  the locale precedence of §6 on each `GOOS`, and the `COLUMNS` width.
  `TERM=dumb` gives plain under `Want: Interactive` too, while
  `NoInputEnv` and `CI` do not.
* **The guide's `--tui` example,** compiled in a scratch module:
  * run with no terminal, and again with `TERM=dumb` under a real
    terminal: one `Warning:` line on standard error naming the reason,
    then the CLI mode's output and exit status;
  * run under a real terminal: the TUI, with no warning;
  * run under a real terminal with a model that panics after its first
    `Update`: the `Warning:` line, then the CLI mode continuing from the
    last good model, and no second attempt at the TUI.
* **`Run`.**
  * On a plain decision it returns an error wrapping `ErrNotStarted`
    and naming the reason, and leaves all three writers empty.
  * A failure before `Init`, induced in a test (the PLAN names how),
    returns `ErrNotStarted`, leaves the writers empty and puts the saved
    terminal state back. A failure after `Init` does not wrap
    `ErrNotStarted`.
  * The final model comes back unwrapped, as `M`.
  * A panic in `Init`, in `Update`, in `View` and in a command each
    returns `ErrCrashed`, wrapping `tea.ErrProgramPanic`, with the last
    good model. A model quitting, a model returning `tea.Interrupt` and
    a cancelled context each do not.
  * Under a real terminal (`script` on macOS, and on the Windows test
    host), a panic in `Update` of a full-screen model leaves the
    terminal usable. The main screen is back, the cursor shows, and
    `stty -a` shows `icanon` and `echo` as before the run.
  * On a forced interactive decision over in-memory streams, a model that
    quits is returned typed.
  * A deadline gives `ExitCode` 124, and a model returning
    `tea.Interrupt` gives 130.
* **`Frame`** with tuitest goldens of a workspace across the matrix of
  rule 6.
* **`ExitCode`** against each row of §9, with wrapped errors.
* **The probes rerun with launch** (P1–P3 as a launch example program),
  on macOS and the Windows test host:
  * no terminal: a plain result, no escape bytes, exit 0, and no hang;
  * under a real terminal (`script` on macOS), the TUI starts and Ctrl+C
    exits 130;
  * with stdout piped and `UIOnErr`, the UI is on stderr and stdout
    holds only the result.
* **depguard** refuses `x/term` outside `launch`, shown by a mutation on
  a scratch copy.
* **`internal/conformance`** passes on `launch`. A mutation that writes
  `AltScreen` in launch, on a scratch copy, is caught.
* **`go mod tidy -diff`** is clean, and `go list -m all` is the same set
  of modules and versions before and after.

## Pros and Cons of the Options

### A. A `launch` package

* Good, because it removes the failure modes P1–P3 for every program
  that uses it, and the decision is testable without a terminal.
* Good, because it composes: a program can call `Decide` alone, and use
  the decision with its own `tea.NewProgram`.
* Neutral, because it promotes one indirect dependency to direct, with no
  change to the graph.
* Bad, because it is new API to keep stable, and its CI and locale rules
  are judgement calls a program may want to override. `Want` and the
  `IsTerminal` override are the escape hatches.

### B. A guide only

* Good, because it adds no API.
* Bad, because every program copies about 95 lines of launch code (the
  research demo's count: TTY test, locale, width, options, exit mapping
  and plain frame). Each copy can get the order wrong, and P2's hang
  shows the cost.
* Bad, because the library still cannot test terminal state anywhere
  (G1).

### C. A program-options helper only

* Good, because it is small: one function returning
  `[]tea.ProgramOption`.
* Bad, because it does not decide. Given non-terminal streams it builds a
  program that hangs (P2), and the plain path, exit codes and colour
  rules stay with each program.

### D. Plain output through `WithoutRenderer`

* Good, because one code path serves both modes, as in Bubble Tea's
  `tui-daemon-combo` example.
* Bad, because the renderer it installs drops `Println` and `Printf`
  (`nil_renderer.go:21`), against its own documentation.
* Bad, because it enters no raw mode (keys need Enter: Bubble Tea #1518).
* Bad, because it still needs input, so without `WithInput(nil)` it
  falls back to `/dev/tty` (`tea.go:1015-1026`), which is P1.

### E. A form generated from the registry

* Good, because a person could fill in any command's arguments without
  learning its flags (Trogon; clap-tui).
* Bad, because it is a TUI feature, not integration. It needs the launch
  path first, and its own design: argument widgets from the schema, how
  to run the command (Trogon's re-exec through `os.execvp` has open
  issues #106, #81, #70 and #39; clap-tui returns the parsed value
  in-process instead), and what to print.

## Amendments

### A1 (2026-10-07): the native integration surface

[0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md)
W1 makes this record the integration surface for every common Go CLI
framework. Its evidence is
[0014-REPORT-api-assessment-and-integration-research.md](../reports/0014-REPORT-api-assessment-and-integration-research.md).
Where this amendment and the Decision Outcome above differ, this amendment
holds. The owner accepted this record, with A1, on 2026-10-07 ("accepted, write the actionable and comprehensive plans").

#### A1.1 Two holes in the fallback, closed

* **A Loop command never waits on a program that has ended (REPORT I1).**
  * **The hole.** `command`'s loop is fixed when the registry is built
    (`command/registry.go:82-88`). `onLoop` waits for a result or for the
    caller's context (`command/dispatch.go:68-88`). Bubble Tea's `Send` is
    a no-op once the program has ended (`tea.go:1188-1197`). So after a
    fallback, every workspace command, all of which are Loop commands,
    hangs a CLI handler that passes `context.Background()`.
  * **The fix.** `command` gains a loop that can be attached and detached
    (0014-MADR W2). While a loop is attached, a Loop command runs on it.
    Once the loop's done channel closes, the command runs on the caller
    instead, exactly once.
  * **In `launch`.** `WithRegistry(r)` attaches the registry to the
    program when it starts, and detaches it on every outcome before `Run`
    returns.
* **Modes set outside Bubble Tea are undone on every outcome (REPORT I2).**
  * **The hole.** `termcap.Prober` turns on mode 2031 with raw bytes
    (`termcap/prober.go:309-311`). Only `Quit` or `Restore` turns it off
    (`:220-234`). After a crash, an interrupt or a cancel, the CLI mode's
    input would receive colour-scheme reports.
  * **The fix.** `type Restorer interface{ Restore() string }`.
    `*termcap.Prober` already satisfies it. `WithRestorer(r)` makes `Run`
    write `r.Restore()` to the stream the TUI drew on, on every outcome.
    It writes after Bubble Tea's shutdown and before the saved terminal
    state goes back. The bytes reset a mode, so writing them after a clean
    `Quit` is harmless. The planned `termmode` plugs into the same seam.
* **The program is reachable while it runs.** `OnStart(func(*tea.Program))`
  is called once the program exists, so an agent goroutine can `Send`.

#### A1.2 Native types for every framework

These replace §1's `Want`, `Policy` and `Route`, and their names follow
0014-MADR W0.3:

```go
// Mode is the program's request, bound to its own flag: auto, tui or plain.
type Mode uint8

const (
    ModeAuto  Mode = iota // decide from the streams and the environment
    ModeTUI               // the TUI was asked for: skip the environment's vetoes, not TERM=dumb
    ModePlain             // never start the TUI
)

// Each method serves the frameworks named (0014-REPORT §3.2).
func (m Mode) String() string                  // flag, pflag, urfave, ff
func (m *Mode) Set(s string) error             // flag, pflag, urfave, ff
func (m Mode) Type() string                    // pflag, cobra: "mode"
func (m Mode) Get() any                        // urfave/cli v3 requires it
func (m Mode) MarshalText() ([]byte, error)    // kong, go-arg, flag.TextVar, urfave TextFlag
func (m *Mode) UnmarshalText(b []byte) error
func (m Mode) MarshalFlag() (string, error)    // go-flags
func (m *Mode) UnmarshalFlag(s string) error

// Flags is the shared flag set. Kong embeds it (embed:""), go-flags
// groups it, ff adds it with AddStruct, go-arg embeds it. TUI is a plain
// bool because ff rejects *bool.
type Flags struct {
    Mode Mode `name:"mode" long:"mode" ff:"long=mode" default:"auto" help:"auto, tui or plain" description:"auto, tui or plain"`
    TUI  bool `name:"tui" long:"tui" ff:"long=tui" help:"start the TUI" description:"start the TUI"`
}

func (f *Flags) RegisterFlags(fs *flag.FlagSet) // stdlib; cobra through AddGoFlagSet; ff through NewFlagSetFrom
func (f Flags) Resolve() Mode                   // ModeTUI when TUI is set, otherwise Mode

// Streams, as before. StreamSource is what *cobra.Command has.
type StreamSource interface {
    InOrStdin() io.Reader
    OutOrStdout() io.Writer
    ErrOrStderr() io.Writer
}

func FromSource(src StreamSource, env termcap.Env) Streams

// Config is how a program lets the TUI reach the terminal (was Policy).
type Config struct {
    Mode       Mode
    NoInputEnv string
    UIOnErr    bool
    OpenTTY    bool
}

// Target is where the TUI reads or draws (was Route).
type Target uint8 // TargetNone, TargetStream, TargetErr, TargetTTY

// Reason is a stable text token, in termcap's style ("area.thing").
type Reason string // "launch.requested-plain", "launch.no-input-variable",
// "launch.dumb-terminal", "launch.ci", "launch.input-not-terminal",
// "launch.output-not-terminal", "launch.terminal"

type Decision struct {
    Interactive bool
    Reason      Reason
    In, UI      Target
    Profile     colorprofile.Profile
    UIProfile   colorprofile.Profile
    Glyphs      glyph.Tier // was UTF8 bool; the legacy-console tier is chosen by theme v2's rule
    Width       int
    Height      int
}

func Decide(s Streams, c Config) Decision

// Options for Run (opaque, per 0014-MADR W0.4).
type Option interface{ /* unexported */ }

func WithRegistry(r *command.Registry) Option
func WithRestorer(r Restorer) Option
func OnStart(f func(*tea.Program)) Option
func WithProgramOptions(o ...tea.ProgramOption) Option
func Filter[M tea.Model](f func(M, tea.Msg) tea.Msg) Option

func Run[M tea.Model](ctx context.Context, s Streams, d Decision, m M, opts ...Option) (M, error)

// ExitError carries a status through any framework: kong's FatalIfErrorf
// finds it with errors.As, urfave's HandleExitCoder unwrapped.
type ExitError struct {
    Code int
    Err  error
}

func (e *ExitError) Error() string
func (e *ExitError) Unwrap() error
func (e *ExitError) ExitCode() int
```

* **`ExitCode(err)`** checks `interface{ ExitCode() int }` first,
  through `errors.As`. That covers `ExitError` and `command`'s errors
  (0014-MADR W2). §9's table follows.
* **`Run` returns its errors unwrapped,** so urfave's type assertion sees
  an `ExitError` a program builds from them.
* **The kong case.** A bare `--tui` cannot come from a custom type in kong:
  kong takes boolean flags only through its own mapper. `Flags.TUI` is
  therefore a `bool`, and kong reads it natively; `negatable:""` adds
  `--no-tui`. A type holding a pointer is avoided, because kong zeroes the
  field and allocates it again (REPORT §3.2).
* **The pflag case.** pflag ignores `IsBoolFlag`, so a cobra program binds
  a bare `--tui` with `RegisterFlags` and `AddGoFlagSet`, which copies it,
  or sets `NoOptDefVal`.
* **`launch` imports more.** It imports `command` (for `WithRegistry`)
  and `glyph` (for `Tier`), as well as §1's imports. Neither of those
  imports `launch`.

#### A1.3 The colour profile spawns nothing

* **The problem.** §5 computed each profile with `colorprofile.Detect`.
  Inside tmux, that runs `tmux info` (`colorprofile@v0.4.3/env.go:248`),
  and the owner decided the library spawns no process
  ([0003-REPORT](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §11.5).
* **The fix.** A profile is `NoTTY` for a stream that is not a terminal.
  Otherwise it is `colorprofile.Env` of the adjusted environment (§5's
  `NO_COLOR` and `FORCE_COLOR` rules), which reads the environment only.
  No terminfo file and no child process.
* **What is lost.** The terminfo and tmux upgrades `Detect` adds. A
  terminal that forwards `COLORTERM` keeps TrueColor; one that does not is
  refined at run time by `termcap`, whose `TmuxQuery` returns the command
  for the program to run.
* **Bubble Tea makes no call of its own.** `Run` passes
  `tea.WithColorProfile`, so Bubble Tea's own `Detect` (`tea.go:1092`) is
  never reached either.

#### A1.4 Smaller changes

* **Terminal facts.** A stream may also implement
  `interface{ Size() (width, height int) }`, which wins over
  `term.GetSize`. A fake terminal then has a size, and §6's sizes are
  testable.
* **An epilogue.** If the final model implements
  `interface{ Epilogue() string }`, `Run` writes the epilogue to `Out`
  after the terminal is restored, on a clean end. That is where a resume
  hint goes (opencode, REPORT §4.2).
* **`launch/launchtest`:**
  * fake terminal streams with a size;
  * `Streams` builders;
  * an environment builder;
  * a `Run` driver that types keys.

  A consumer tests its own CLI integration with these.
* **Tier-1 examples.** The guide's examples for stdlib `flag`, cobra, kong
  and urfave/cli v3 are compiled and run in a scratch module on every
  release (0014-MADR, owner answer 3). ff v4, go-flags and go-arg are
  documented with REPORT §3.2's results.
* **§12, out of scope.** A result writer (`command.WriteResult`) and plain
  rendering (`workspace.RenderPlain`) leave that list for 0014-MADR W2, a
  later release.
* **What `launch` cannot do.** Bubble Tea recovers a panic itself and
  returns only `tea.ErrProgramPanic` (`tea.go:1034-1041`), so `ErrCrashed`
  cannot carry the panic value or the stack. Bubble Tea writes them to the
  process's standard error (§10).

#### A1.5 The PLAN

[0013-PLAN-cli-integration-helpers.md](0013-PLAN-cli-integration-helpers.md)
was written before this amendment. It is revised to match once A1 is
accepted, and its revision names each change.

#### A1.6 (2026-10-07, while planning): two names

* **`Mode` becomes `Choice`.**
  * `command` already exports `Mode`, for `Loop` and `Async`
    (`command/command.go:199-205`), and 0014-MADR W0.3's collision check
    would refuse a second.
  * The flag type is therefore `launch.Choice`, with `ChoiceAuto`,
    `ChoiceTUI` and `ChoicePlain`, and `Config.Mode` is `Config.Choice`.
  * The flag stays `--mode`: `Flags.Mode`, of type `Choice`.
  * Nothing else in A1.2 changes.
* **`Filter` becomes `WithFilter`,** under 0014-MADR W0.4's naming rule
  (0014-MADR A1.4).
* **`Decision` exists in two packages for now.** `command` exports one
  (`command/gate.go:11`), which 0014-MADR W4 renames to `Verdict`. The
  collision check allows both until W4 removes the old name in `v0.10.0`,
  through a dated allowlist entry.

#### A1.7 (2026-10-07, while executing the PLAN): what `colorprofile.Env` reads on Windows

* **The fact, corrected.** A1.3 says `colorprofile.Env` "reads the
  environment only". On Windows it also asks the operating system for its
  build number, when `TERM` is unset, empty or `dumb`, and `ConEmuANSI` is
  not `ON` (`colorprofile@v0.4.3/env.go:144-152`,
  `colorprofile@v0.4.3/env_windows.go:12-40`, through `windows.RtlGetNtVersionNumbers`). That
  is how a Windows console, which sets no `TERM`, gets its colours.
* **What holds.** It opens no file and starts no process, so A1.3's
  decision, and the owner's no-spawn rule, are unchanged.
* **What changes.** A test of a profile with `TERM=dumb` depends on the
  host's Windows build, unless it sets `ConEmuANSI=ON`, which answers
  from the environment. The PLAN's deviation D3 records the test that
  found this.

#### A1.8 (2026-10-07, while executing the PLAN): two corrections from the live probes

* **"The terminal is as it was" means every mode.** On macOS and the
  BSDs, the kernel sets `PENDIN` in `c_lflag` whenever canonical mode
  comes back. termios(4) calls it "retype pending input (state)".
  * A bare `term.MakeRaw` and `term.Restore`, with no launch, leave it
    set. So a state launch puts back after a crash or a failed start
    matches the saved one in every mode, and differs in that bit.
  * The PLAN's deviation D7 records the probe and the control run.
* **`Flags.TUI` carries `negatable:""`.** A1.2's tag list omits it, but
  its prose says `negatable:""` adds `--no-tui`. A program cannot add a
  tag to a field of a struct it embeds, so the tag belongs on `Flags`.
  The other frameworks ignore it. The PLAN's deviation D8 records the
  probe.

#### A1.9 (2026-10-08, while executing the PLAN): `OpenTTY` is not supported on Windows

* **The finding.** On Windows, a TUI reading the `CONIN$` launch opened
  gets its keys about one Enter late. When it ends, its read is still in
  flight, and takes the CLI mode's next line.
  * Only stdin's console reader can be cancelled there
    (`ultraviolet/cancelreader_windows.go:28-40`), as §4's facts say.
  * The PLAN's deviation D9 records the live probe, and the baseline
    without `OpenTTY`, which is correct.
* **The decision.** On Windows, `Decide` treats `Config.OpenTTY` as off,
  and never chooses `TargetTTY`.
  * A `--tui` whose stdin or stdout is redirected there gets a plain
    decision, `launch.input-not-terminal` or
    `launch.output-not-terminal`, and falls back to the CLI with its
    input intact.
  * `UIOnErr` still lets the TUI draw on a console Err.
* **What would lift it.** A cancellable reader for a console handle other
  than stdin, upstream. launch cannot make one without making `CONIN$`
  the process's stdin, a process-wide stream §10 keeps it away from.

## More Information

### Probes

Each probe was run on 2026-10-07 on macOS with go1.27.1, against
go-tui-lib `v0.6.0`, Bubble Tea v2.0.10 and colorprofile v0.4.3, from a
shell with no controlling terminal. A research agent wrote the demo
program, a workspace with stdlib `flag` subcommands `naive`, `explicit`
and `tui`, and ran it first. P1, P2, P3, P4, P6 and P7 were run again
for this record.

* **P1, `naive`** (`tea.NewProgram(m, tea.WithContext(ctx))`),
  `</dev/null | cat`: exit 1, "bubbletea: error opening TTY: bubbletea:
  could not open TTY: open /dev/tty: device not configured", after 0 s.
* **P2, `explicit`** (`WithInput(in)`, `WithOutput(out)`,
  `WithContext`, `WithEnvironment`), `</dev/null >file`: still running
  at 10 s. SIGTERM cancelled the program's `signal.NotifyContext` and
  `Run` returned "program was killed: context canceled". The agent's
  first run lasted 2 min 7 s before it was killed. The output file held
  110 bytes, which begin
  `ESC[?2026$p ESC[?2027$p ESC]11;?BEL ESC[>4m ESC[?1049h ESC[?25l ESC[?2004h`.
* **P3, P2 with `-timeout 2s`:** exit 124 (the demo's own mapping),
  "program was killed: context deadline exceeded".
* **P4, `tui`, which decides first:** with stdin and stdout not
  terminals it drew one frame through `Update` and `Render`. 1278 bytes
  of Unicode box drawing with no escapes; 648 bytes of ASCII with
  `NO_COLOR=1 LANG=C LC_ALL=C`.
* **P5, the agent's run under a pseudo-terminal (`script`),** not
  repeated here:
  * with stdin `/dev/null`, `naive` opened `/dev/tty`, ran the TUI and
    quit on a typed `q`;
  * with stdout piped to `od`, the alternate-screen and OSC 11 query
    sequences were in the pipe.
* **P6, colorprofile v0.4.3 `Env`** with `TERM=xterm-256color
  COLORTERM=truecolor`:
  * no `NO_COLOR` gives TrueColor;
  * `NO_COLOR=1` gives Ascii;
  * `NO_COLOR=yes` gives TrueColor, and so does `NO_COLOR=x`;
  * `TERM=dumb` gives NoTTY.
  * A search of the module for `FORCE_COLOR` finds nothing.
* **P7, `os.Stdin` from `/dev/null`:** `Mode()&os.ModeCharDevice != 0`
  is true, and `term.IsTerminal(fd)` is false.

### Sources read at a pinned version

All in the module cache, read for this record:

* **Bubble Tea v2.0.10:** `tea.go`, `options.go`, `tty.go`,
  `tty_unix.go`, `tty_windows.go`, `signals_unix.go`, `exec.go`,
  `nil_renderer.go`, `renderer.go`.
* **colorprofile v0.4.3:** `env.go`, `writer.go`.
* **x/term v0.2.2:** `term_windows.go`.
* **ultraviolet,** at the version the root selects: `tty_unix.go`,
  `tty_windows.go`.
* **Go 1.27.1:** `flag/flag.go`, `runtime/panic.go`,
  `internal/poll/fd_windows.go`, `os/signal/doc.go`.

### Survey

Research agents read the sources, issues and documents below. The
entries this record leans on were checked again for it with `gh` or by
fetching the page: gh's `CanPrompt`; gum's `choose/command.go`, `main.go`
and `internal/exit`; huh's `form.go`; glow's and mods' `main.go`; wish's
`bubbletea/tea.go`; ci-info's `index.js`; Ink's `render.ts`; no-color.org,
force-color.org and clig.dev; Bubble Tea #761, #860, #1459 and #1590;
huh #718; agentcore-cli #949; and, for the owner's answers, the Buildkite
agent's `agent_start.go`, supports-color's `index.js`, Rich's
`console.py`, go-gh's `env.go` and colorprofile #77.

| Project | What it does at the boundary | What this record takes |
| :--- | :--- | :--- |
| [gh](https://github.com/cli/cli/blob/trunk/pkg/iostreams/iostreams.go) | a terminal test per stream, with test overrides; prompts only when stdin and stdout are both terminals; `GH_PROMPT_DISABLED`, `GH_FORCE_TTY` | §3's default, `NoInputEnv`, the `IsTerminal` override |
| [gum](https://github.com/charmbracelet/gum) v2.0.2 | UI on stderr, result to stdout after `Run`; 130 interrupted, 124 timed out | `UIOnErr`, §9 |
| [huh](https://github.com/charmbracelet/huh) | forms on stderr by default; `TERM=dumb` forces accessible mode; #718, a hidden form still needs a TTY | the `TERM=dumb` veto; accessible mode is a later record |
| [glow](https://github.com/charmbracelet/glow) | TUI or render chosen from stdin and arguments; a `notty` style when stdout is not a terminal; width capped at 120, falling back to 80 | `Frame`, the width fallback left to the program |
| [mods](https://github.com/charmbracelet/mods) (archived 2026-03-09) | `WithInput(nil)` when stdin is not a terminal; UI on stderr when stdout is | `UIOnErr` |
| [wish](https://github.com/charmbracelet/wish) v2.0.5 | per-session input, output, environment, size and profile; refuses a session with no PTY | `Streams`, `termcap.Env`, the `IsTerminal` override |
| [fzf](https://github.com/junegunn/fzf) | candidates on stdin, UI on `/dev/tty`, result on stdout; exit 130 on Ctrl-C | `OpenTTY`, §9 |
| [Ink](https://github.com/vadimdemedes/ink) | `interactive` from CI detection and `stdout.isTTY`; non-interactive writes only the final frame | the CI veto, `Frame` |
| [Textual](https://github.com/Textualize/textual), [ratatui](https://github.com/ratatui/ratatui) | `run()` returns the app's value; restore on panic | `Run[M]` returning the final model |
| [Trogon](https://github.com/Textualize/trogon), [clap-tui](https://crates.io/crates/clap-tui) | a TUI generated from the CLI's schema; Trogon re-executes, clap-tui returns in-process | option E, a later record |
| [aws/agentcore-cli#949](https://github.com/aws/agentcore-cli/pull/949) | a guard before 17 TUI entry points; exit 1 with the non-interactive flags named | the check before Bubble Tea starts; `ErrNotStarted` |
| [clig.dev](https://clig.dev) | prompts only when stdin is a TTY; `--no-input`; no colour under `TERM=dumb` | §3 |
| [no-color.org](https://no-color.org), [force-color.org](https://force-color.org), [CLICOLOR](https://bixense.com/clicolors/) | any non-empty value counts; `NO_COLOR` wins; the command line outranks the environment | §5, `Want` over the vetoes (not over `TERM=dumb`, a capability) |

### Owner answers, 2026-10-07

Picked from options, except answers 6 and 8, which the owner stated;
the Decision Outcome states each choice. Answers 1 to 4 and 7 are the
recommendation; 5, 6 and 8 are not.

1. **The package name: `launch`.** It reads as what a program does with
   it (`launch.Decide`, `launch.Run`). The alternatives were `start`
   (`start.Run` reads awkwardly, and a `start` variable shadows it) and
   `cliterm` (a coined word, whose `cli` blurs 0012's line that the
   library ships no CLI). `tty`, `term` and `run` were not offered:
   `term` is x/term's own import name, `tty` names a device rather than a
   decision, and `run` is a common local name.
2. **The CI veto (§3): on by default,** with `CI=false` and
   `Want: Interactive` as the ways out. The alternatives were a
   `Policy` field the program sets to opt in, and no veto at all (gh).
   Both leave a TUI starting under Buildkite's pseudo-terminal.
3. **`FORCE_COLOR` (§5): honoured, except `0` and `false`.** The
   alternatives were any non-empty value (force-color.org and Rich, but
   turning colour on for Node's `FORCE_COLOR=0`), and not reading it
   (colorprofile today).
4. **x/term (§11): `launch` only,** by a depguard rule. The alternatives
   were an internal package holding the terminal test (as
   `internal/cells` holds ultraviolet), one more package for two
   functions today, and no rule (x/term has tagged releases, so
   ultraviolet's pinning reason does not apply, but nothing would stop a
   pane package from querying the terminal). A later need widens the rule
   by amendment.
5. **The API keeps `Auto`, `Interactive` and `Plain`,** with `Auto` the
   zero value. This and answers 6 and 7 followed the owner's statement
   that gobble's default is its CLI and the TUI runs only with `--tui`.
   The recommendation was different. A
   program with an opt-in TUI passes `Want: Interactive` when `--tui` is
   given. The alternatives were:
   * make the explicit request the zero value, with automatic starting
     opt-in through a `Policy.Auto` field, and drop `Plain` (the
     recommendation);
   * drop `Auto` altogether, leaving programs like glow and Pi to write
     their own checks.
6. **A `--tui` that cannot start, for any reason, falls back to the CLI
   mode** (§3). The owner first picked "refuse, exit 2", the
   recommendation, and then the same day said: "if --tui can't launch the
   tui for whatever reason it should fallback to running in native cli
   node."
   * The one `Warning:` line on standard error is this record's
     addition, following gobble's rule for diagnostics (gobble-cli
     `docs/decisions/0008-MADR-native-cli-mode.md` D3).
   * Refusing stays available to other programs through `ExitCode`.
   * The fallback needs `ErrNotStarted`'s guarantee, so `Run` restores
     the terminal on a failure before `Init` (§7).
7. **`TERM=dumb` stops the TUI even under `--tui`** (§3, rule 3). The
   alternative was to let the flag outrank it, as it outranks the other
   vetoes.
8. **A TUI that crashes mid-session falls back too, and continues**
   (§3, §7). This record had kept the fallback to failures before the
   start, because the CLI could repeat work the TUI had done. Asked, the
   owner said: "scope this in. this is exactly the kind of functionality i want in gobble. graceful and seamless handling of edge cases." The repetition is avoided by
   continuing from the last good model (for gobble, its session), not
   by restarting.
