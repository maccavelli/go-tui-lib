# API assessment of `v0.6.0`, and research on how Go CLIs and agent tools integrate a TUI

This report observes and decides nothing. The decisions it informs are in
[0014-MADR-native-integration-api.md](../decisions/0014-MADR-native-integration-api.md),
and the amendment of
[0013-MADR-cli-integration-helpers.md](../decisions/0013-MADR-cli-integration-helpers.md).

## 1. Scope and method

On 2026-10-07 the owner asked for:

> a wide, broadly compatible scope to integration helpers, much like i want
> a broadly compatible, flexible, extensible, api for the go-tui-lib
> library. integration helpers should have fully native API support backing
> them.

They also asked for an assessment of the current API, research on similar
libraries and agent tools, and ideas for future-proofing, hardening and
canonicalization.

Five research passes ran in parallel, read-only:

1. The go-tui-lib API at `v0.6.0`.
2. Codex, Grok and goose, which are Rust.
3. opencode, Kilo Code and Pi, which are TypeScript.
4. The Go TUI framework landscape.
5. Go CLI frameworks' native integration points.

Scratch programs and clones lived outside the repository. The coordinator
re-read the claims this report leans on; §8 lists them. A claim marked
**(reported)** comes from a pass's reading at the commit named and was not
re-read.

| Source | Version or commit | Where |
| :--- | :--- | :--- |
| go-tui-lib | `v0.6.0`, `main` at `47e11c4` | this repository |
| Bubble Tea, colorprofile, x/term, ultraviolet | v2.0.10, v0.4.3, v0.2.2, the root's pseudo-version | the module cache |
| Codex | `823ea830c0` | openai/codex |
| Grok Build | `2bdd1d6a` | the local checkout |
| goose | `01eb01a` | aaif-goose/goose |
| opencode | `907b3bc518` | anomalyco/opencode |
| Kilo Code | `4a7c4e883f` | the local checkout |
| Pi | `312184edb` | the local fork of earendil-works/pi |
| CLI frameworks | cobra v1.10.2, pflag v1.0.10, kong v1.16.1, urfave/cli v3.14.0, ff v4.0.0-beta.1, go-flags v1.6.1, go-arg v1.6.1, flaggy v1.8.0, fang v2.0.1 | the module cache |

## 2. The go-tui-lib API at `v0.6.0`

### 2.1 Inventory

Ten public packages: `command`, `glyph`, `layout`, `termcap`,
`termcap/termcaptest`, `termsvc`, `theme`, `tuitest`, `when`, `workspace`.
`GOWORK=off go test -race ./...` passes in each.

- **Option types:**
  - five are functions over exported structs: `command.Option` and
    `RegistryOption`, `termcap.Option`, `workspace.Option`,
    `termsvc.NotifyOption`;
  - three use private configuration: `theme.Option`,
    `termsvc.CopyOption`, `termcap.ReportOption`.
- **Sentinel errors:**
  - `layout`: five;
  - `command`: three, plus `*ArgError`;
  - `termsvc`: two;
  - `when` and `termcap`: none.
- **Message types:** every one ends in `Msg`.

### 2.2 Integration gaps

Severity is **B** (blocks native integration, or breaks a promise 0013
makes), **I** (important) or **N** (nice to have). None of the fixes
breaks the existing API.

| # | Gap | Evidence | Sev |
| :--- | :--- | :--- | :--- |
| I1 | A Loop command run after the program ends never runs, and its caller waits on its context, for ever with `context.Background()` | `WithLoop` is fixed at construction (`command/registry.go:82-88`). `onLoop` hands the command to `r.loop` and waits for a result or for `ctx` (`command/dispatch.go:68-88`). Bubble Tea's `Send` is a no-op once the program has ended (`tea.go:1188-1197`). Probe: "after 2.001s: result \"\", err context deadline exceeded" with a 2 s deadline. 0013's `Run` hides the `*tea.Program`, so `WithLoop` cannot be wired at all | B |
| I2 | Mode 2031, set by `termcap.Prober`, survives a crash, an interrupt or a cancel | set with `tea.Raw(ansi.SetModeLightDark)` (`termcap/prober.go:309-311`); reset only by `Quit` or `Restore` (`:220-234`). 0013 restores the terminal's line settings only | B |
| I3 | `workspace` cannot be given glyphs, a profile or a size | the first theme is `theme.New(colorprofile.ANSI256, theme.Unknown, glyph.Unicode())` (`workspace/workspace.go:265`). A rebuild keeps the glyphs in use (`:324-334`). The fallback size is 80×24 (`:112`) | I |
| I4 | No plain, linear rendering of a pane or a workspace, for pipes and screen readers | `Pane` has only `View(width, height)` (`workspace/workspace.go:46-49`) | I |
| I5 | No writer for a `command.Result` | `CallMCP` already renders text or JSON (`command/mcp.go:94-114`); the guide prints `res.Text` with `fmt.Println` (`docs/guides/commands.md:316-330`) | I |
| I6 | Command errors carry no exit status | `ErrUnknown`, `ErrUnavailable`, `ErrRefused` (`command/registry.go:21-30`), `*ArgError` (`command/args.go:13-25`) | I |
| I7 | No entry point for an already-split argument vector | `ParseSlash` takes one line and quotes it itself (`command/slash.go:24-107`) | I |
| I8 | No framework-neutral parameter description, for flags and completion | the CLI tags live only in the schema's `x-cli` (`command/schema.go:44-70`); the reader is unexported (`command/decode.go:29-45`) | I |
| I9 | Only `termcap`'s enums have text forms, so no other enum binds to `flag.TextVar` or Kong | `termcap/termcap.go:123-145`; `theme.Background` is an `int` (`theme/theme.go:36`) | I |
| I10 | No capability facts from the environment alone, for the CLI mode | `setEnv` is unexported (`termcap/env.go:37`) | I |
| I11 | Notifications and clipboard exist only as `tea.Cmd` | `termsvc/notify.go:182-197`, `termsvc/clipboard.go:72-94` | I |
| I12 | No testing kit for a consumer's CLI integration | `termcaptest` fixes 80×24 (`termcap/termcaptest/termcaptest.go:223-230`); 0013's fakes are private | I |
| I13 | The help footer draws bubbles' default `•` and `…`, against rule 3 | `workspace/help.go:9-11` | N |
| I14 | `Watch` takes no context, and leaks after a fallback | `command/registry.go:393-404` | I |
| I15 | Each CLI writes a yes-gate and an argument marshaller | `docs/guides/commands.md:304-318` | N |
| I16 | No accessible mode | — | N now, I before v1 |

### 2.3 The proposed `launch` (0013)

| # | Finding | Sev |
| :--- | :--- | :--- |
| L1 | `Decision.UTF8` is a `bool`, while the owner chose a third, legacy-console (CP437) glyph tier ([0003-REPORT](0003-REPORT-agent-tui-ecosystem-research.md) §11.5) | I |
| L2 | `launch.Reason` is a number with prose; `termcap`'s reasons are stable dotted tokens | I |
| L3 | `Run` hides the program (I1) and has no hook to undo modes set outside Bubble Tea (I2). `Frame` takes a `tea.Model`, which `*workspace.Workspace` is not (`workspace/workspace.go:7-12`) | B / I |
| L4 | `Want` has no text form, so it cannot be a flag | N |
| L5 | `Decision` carries no background or capability facts, so a program builds its theme by hand | I |
| L6 | The model wrapper and the program builder repeat `termcaptest`'s (`termcap/termcaptest/termcaptest.go:197-257`) | N |
| L7 | Three wrong line citations in the 0013 records, since corrected | — |
| L8 | `termcap.Env` lookups are case-sensitive (`termcap/env.go:16-23`); Windows environment names are not | N |
| L9 | 0013 §12 leaves out the pieces that make the integration native: I4 to I6, I16 | I |

### 2.4 Consistency and canonicalization

| # | Finding | Evidence |
| :--- | :--- | :--- |
| C1 | **Names mean different things in different packages.** `Context` (`when/value.go:102`, `layout/layout.go:262`, the `command.Request.Context` and `Invocation.Context` fields beside a `context.Context`, `command/handler.go:35,52`); `Policy` (`termsvc/notify.go:54`); `Route` (`termsvc/clipboard.go:29`); `Origin` (`termcap`, `command/command.go:271`); `Decision` (`command/gate.go:11`); `Auto`, `Unknown`, `Kind`, `Conflict`. 0013 adds `Policy`, `Route`, `Reason`, `Decision`, `Auto`. The planned 0007 adds `keymap.Context`, `Origin` and `Conflict` (`docs/decisions/0007-MADR-keymap-engine.md:259,276,322`) | as cited |
| C2 | **Options.** `layout`'s preset options have bare names (`layout/presets.go:56-88`); `workspace` mixes `On*` and `With*` (`workspace/model.go:53-73`). Five option types expose their target struct, so a third party's option reaches internals | §2.1 |
| C3 | **Constructors.** `termcap.New` returns `*Prober` (`termcap/prober.go:191`); there is no `command.MustNew` | as cited |
| C4 | **Enum text written three ways** | `termcap/termcap.go:123-145`; `command/command.go:179,244,284`; `termsvc` |
| C5 | **One-message `tea.Cmd` helper written three times** | `command/dispatch.go:196`, `termsvc/clipboard.go:96`, `termsvc/notify.go:199` |
| C6 | **Sanitizing scattered; workspace titles and badges not sanitized** | `termsvc/termsvc.go:90-117`, `termsvc/beacon.go:27-38`, `command/registry.go:193`; `workspace/render.go:214-258` |
| C7 | **The environment in three forms:** `termcap.Env`, `[]string`, `os.Getenv` | `termcap/termcaptest/termcaptest.go:32`; `tuitest` |
| C8 | **Width measured three ways** | `workspace/workspace.go:251-256`, `tuitest/tuitest.go:215`, `termsvc/termsvc.go:115` |
| C9 | **The test-case-to-theme mapping copied four times** | `workspace/golden_test.go:16-22`, `workspace/agent_golden_test.go:29-33`, `workspace/commands_test.go:420-424`, `theme/theme_test.go:158-162` |
| C10 | **`command` mixes JSON v1 and v2** | audit, command, handler on v1; decode, mcp, schema, slash on v2 |
| C11 | **Unstructured errors** in `when`; none exported by `termcap` | `when/lex.go:47`, `when/when.go:50-66` |
| C12 | **Hooks are sometimes interfaces, sometimes funcs,** with no `Func` adapters | `command.Gate`, `Auditor`; `termsvc.Backend`, `Clipboard` |
| C13 | **Stale `--args` wording** | `command/slash.go:12-13`, `docs/guides/commands.md:171` |

### 2.5 Hardening

| # | Finding | Evidence |
| :--- | :--- | :--- |
| H1 | **The workspace's view cache is never evicted.** Keys include size and theme generation | `workspace/render.go:142-152`; deletions only at `workspace/workspace.go:389,395-397`, `workspace/overlay.go:105`. Probe: 595 entries after 500 resizes, 995 after 200 more profile changes |
| H2 | **No `recover` around command handlers;** the library has none at all | `command/dispatch.go:149-164`; a search for `recover()` outside tests finds nothing |
| H3 | **Window size is unbounded,** and under wish the client chooses it | `workspace/workspace.go:415-417`, `internal/cells/cells.go:28-42` |
| H4 | **`LoadDir` has no size, count or depth limit,** for a project directory that is untrusted | `command/loaddir.go:49,78` |
| H5 | **Agent argument JSON has no size limit and no fuzz target** | `command/decode.go:175`, `command/mcp.go:95` |
| H6 | **Notification and clipboard backends run under `context.Background()`** | `termsvc/notify.go:190`, `termsvc/clipboard.go:82` |
| H8 | **The conformance scan's must-read list is stale;** it omits `command`, `when`, `termcap`, `termsvc` and the internal packages | `internal/conformance/conformance_test.go:347` |
| H9 | **Fuzzing covers `layout`, `when` and `command` only** | `Makefile:94-97` |
| H10 | **`Ratio` overflows:** `avail * sz.N / sz.D` | `layout/size.go:274-275`. Probe: `Ratio(1<<40, 1<<41)` of 16 Mi cells gave 0, not 8 Mi |
| H11 | **Mutable package variables:** `termcaptest.RunTimeout`, the `workspace.Key*` variables | `termcap/termcaptest/termcaptest.go:167`, `workspace/context.go:18-33` |

### 2.6 Future-proofing and v1 readiness

| Item | Today | Gap |
| :--- | :--- | :--- |
| API diff gate | none until `v1` (`docs/architecture.md`, "What is not here") | a breaking change in `v0` goes unnoticed |
| Deprecation policy | none | no migration window for pi-go or gobble |
| Stability tiers | none | — |
| Option types | five expose internals (C2) | — |
| Name collisions | ten or more (C1) | grows with 0007–0009 and 0013 |
| Enum text forms | `termcap` only (I9) | — |
| Structured errors | `when` and `termcap` lack them (C11) | — |
| Input limits | partial (H1, H3–H5) | — |
| Fuzz targets | four, in three packages (H9) | — |
| Glyph choice | a `bool` (`glyph.For`, `tuitest.Case.UTF8`) | the CP437 tier needs a third value |
| Theme roles | a fixed struct (`theme/theme.go:103-109,171-178`) | a new role is a breaking change |
| Toolchain floor | `go 1.27.1`; JSON v2 and the generic method `PaneAs` (`workspace/workspace.go:592`) need it | no written policy |
| Examples | five, in `termcap` and `workspace` | — |

## 3. Go CLI frameworks: native integration without importing them

### 3.1 Versions and adoption (2026-10-07)

| Framework | Latest | Stars | Imported by (pkg.go.dev) |
| :--- | :--- | ---: | ---: |
| stdlib `flag` | go1.27.1 | — | 549,423 |
| cobra | v1.10.2 (2025-12-03) | 44,691 | 195,884 |
| pflag | v1.0.10 | 2,773 | 54,971 |
| urfave/cli | v3.14.0 (2026-10-02) | 24,287 | v3 4,439; v2 23,605; v1 21,121 |
| go-flags | v1.6.1 (2024-06-15) | 2,694 | 14,086 |
| kong | v1.16.1 (2026-08-09) | 3,185 | 3,079 |
| go-arg | v1.6.1 | 2,285 | 1,664 |
| ff | v4.0.0-beta.1, no stable v4 | 1,442 | v4 137; v3 359 |
| fang | v2.0.1 | 1,954 | v2 127 |

### 3.2 One flag type, every framework

A probe type `Mode` (`auto|tui|plain`) imported only the standard library
(`go list -deps`). Its methods were `String`, `Set`, `Type`, `Get`,
`MarshalText`, `UnmarshalText`, `UnmarshalFlag` and `MarshalFlag`. Each
framework was given it in a scratch program:

| Framework | `--mode=plain` | bare `--tui` | Glue in the program |
| :--- | :--- | :--- | :--- |
| stdlib `flag` | native (`Var`, `TextVar`) | native (`IsBoolFlag`) | none |
| pflag, cobra | native | `flag needs an argument: --tui` | one line: `NoOptDefVal = "true"`, or `AddGoFlagSet`, which copies `IsBoolFlag` |
| kong | native (TextUnmarshaler) | `expected value value but got "EOL"` on a custom type | a `bool` field with `negatable:""`; a type holding a pointer panics, because kong zeroes and reallocates the field |
| urfave/cli v3 | native (`GenericFlag`, `TextFlag`) | native (`IsBoolFlag` forwarded) | none; `Get() any` is required (`missing method Get` otherwise) |
| ff v4 | native | native | none; `AddStruct` rejects `*bool` |
| go-flags | needs `UnmarshalFlag` (`strconv.ParseUint: parsing "plain"` without it) | a `bool` field | the two methods, stdlib-only signatures |
| go-arg | native | `missing value for --tui` on a TextUnmarshaler | a `bool` field |
| flaggy | fixed types only | `*bool` only | the program calls `Set` |

- **One tagged struct in four struct parsers.** `Flags{Mode Mode; TUI
  bool}` parsed `--tui` and `--mode=plain` in kong, go-arg, go-flags and ff.
  Two cautions:
  - `TUI` must be a plain `bool`, because ff rejects `*bool`;
  - go-arg's `arg:"--mode"` tag means a positional argument to kong.
- **Streams by method set: cobra only.** `*cobra.Command` has
  `InOrStdin`, `OutOrStdout`, `ErrOrStderr` and `Context`. The others need
  a one-line literal:
  - kong (`Kong.Stdout`, `Stderr`, and no stdin) and urfave
    (`Command.Reader`, `Writer`, `ErrWriter`) expose fields;
  - ff has no streams at all.
- **Exit codes.** An error with `ExitCode() int` was honoured:
  - by kong's `FatalIfErrorf`, through `errors.As`, so a wrapped error
    exited 130;
  - by urfave's `HandleExitCoder`, only unwrapped.

  cobra, ff, stdlib, go-flags and go-arg have no convention.
- **Signals.** No CLI framework installs a handler. fang offers
  `WithNotifySignal`, which uses `signal.NotifyContext`.

### 3.3 Gotchas

- **cobra prints usage-on-error to the Out writer:** 62 bytes in the
  probe. This is cobra #1708, still open; gh does not call `SetOut`.
- **urfave's default exit handler writes to the package variable
  `cli.ErrWriter`,** not the command's.
- **Wrapping a stream** in `bufio`, a colour writer or `io.MultiWriter`
  hides its descriptor. Bubble Tea then sees no terminal: it enters no
  raw mode and reads no size (`tty_unix.go:17`, `:30`).

### 3.4 How tools launch a TUI from their CLI

| Tool | Framework | TUI launch |
| :--- | :--- | :--- |
| crush | cobra, fang | root `RunE`; `tea.WithContext(cmd.Context())`; signals from fang |
| glow | cobra | `--tui`; the TUI by default with no arguments and a non-piped stdin |
| gum | kong | the TUI on stderr, the result on stdout; 130 interrupted, 124 timed out |
| gh | cobra | its own `IOStreams{In, Out, ErrOut}`; typed exit codes |
| k9s | cobra | root `RunE` |
| lazygit, lazydocker | flaggy | always the TUI |

## 4. Agent tools: the boundary between CLI and TUI

### 4.1 Comparison

| | Codex | Grok | goose | opencode | Kilo | Pi |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| UI | ratatui, inline viewport, native scrollback | forked ratatui; fullscreen, inline, minimal | line REPL | OpenTUI | OpenTUI | pi-tui; main screen or alt screen |
| Mode choice | bare command = TUI; `exec`; `app-server` | argv only; `-p`; `agent stdio` (ACP) | `session`; `run`; `acp`, `serve`, `mcp` | TUI; `run`; `serve`, `attach` | as opencode | `resolveAppMode`: rpc, json, print, interactive |
| TTY gate | stdin and stdout, else an error; `TERM=dumb` asks over stdin and stderr | none; a raw-mode failure is the error | per feature | none for the full-screen TUI | as opencode | print unless stdin and stdout are TTYs |
| Falls back to plain | no | no | — | no | no | yes, by decision before start |
| UI's stream | stdout | stderr | stdout; approvals on stderr | stdout | stdout | stdout |
| Layering | TUI and exec are clients of app-server (JSON-RPC, in process or remote) | TUI and headless are ACP clients of one agent | in-process event stream | TUI is a client of one SDK; the transport varies | as opencode | one session event stream for every mode |
| Exit codes | 1 fatal; 128 + signal | 1, 2, 129, 130, 143 | 0 or 1 | typed `CliError`; fatal exits 1 | fatal can exit 0 (reported) | 1; 129, 143 |

Re-read by the coordinator:

- **Codex:**
  - `init` returns "stdin is not a terminal" or "stdout is not a
    terminal" (codex `codex-rs/tui/src/tui.rs:463-469`).
  - `exec` keeps stdout for the final message, and `deny(clippy::print_stdout)`
    enforces it (`codex-rs/exec/src/lib.rs:1-6`).
  - Where the final message goes is a pure function of
    `(stdout_is_terminal, stderr_is_terminal)`
    (`codex-rs/exec/src/event_processor_with_human_output.rs:516-531`).
- **Grok** points native fd 2 at `/dev/null` and draws on a duplicate of
  stderr (Grok `xai-tty-utils/src/lib.rs:1077-1110`).
- **Pi's `resolveAppMode`:** rpc, json, then print if `--print` or either
  stdin or stdout is not a TTY, then interactive (Pi
  `packages/coding-agent/src/main.ts:112-123`).
- **opencode's TUI command** reads `process.stdin.isTTY` only to merge
  piped stdin into the prompt (opencode
  `packages/opencode/src/cli/cmd/tui.ts:60`).

### 4.2 Patterns worth taking (reported unless §8 lists them)

- **One contract, several sinks.** The UI, a plain or JSON mode and a
  headless server are clients of one event and command contract:
  - Codex's app-server;
  - Grok's and goose's ACP;
  - opencode's SDK;
  - Pi's session stream.

  A command defined once reaches the slash list, the palette, the
  keybindings and the headless list (opencode's keymap registry; Grok's
  ACP `availableCommands`; Pi's extension commands).
- **Mode-aware prompts.**
  - Pi gives extensions a UI context per mode: print mode answers
    `confirm` with false, and RPC turns a dialog into a request with a
    timeout.
  - goose rejects approvals in headless Approve mode.
  - Grok's headless emitter answers permissions by policy.
- **Terminal discipline.**
  - Restore is idempotent, best-effort, keeps the first error and is
    bounded in time (Codex `restore_common`; Grok's teardown table, 2 s).
  - A "run `reset`" hint when it fails (Codex).
  - Typeahead is discarded before a security prompt (Codex) and drained
    before exit (Pi).
  - A dead output (EIO, EPIPE) exits 129 without writing restore bytes
    (Pi).
  - The second signal forces exit (Grok, Codex's app-server).
- **Leaving the session.**
  - An epilogue, such as a resume hint, goes to stdout after teardown
    (opencode).
  - Pi prints the transcript into scrollback when leaving the alternate
    screen.
  - A crash file, with the culprit extension named (Pi).
- **Rendering.**
  - An inline viewport with finished history in native scrollback, using
    a strategy per terminal (Codex; Grok's minimal mode; Pi's main
    screen).
  - Streaming markdown split into a stable prefix and a tail (Codex,
    goose, Crush).
  - Each history cell has styled lines and raw copy lines (Codex).
- **Testing.**
  - Codex: about 1,350 insta snapshots, a VT100 emulator backend, and
    openpty re-exec tests.
  - Grok: a PTY scenario harness timed by mode 2026 markers.
  - Pi: `VirtualTerminal` on xterm-headless.
  - opencode: lifecycle tests for SIGHUP teardown, no leaked listeners,
    the epilogue, and a non-zero exit on a fatal error.

### 4.3 Patterns to avoid

- A full-screen TUI started with no terminal check (opencode).
- A fatal TUI error that exits 0 (Kilo, reported).
- A terminal type bound to the process's own streams (Pi's
  `ProcessTerminal`).
- No panic hook, and a SIGTERM handler written but never wired (goose).
- Hard-coded escapes in plain output.

## 5. The Go TUI landscape (2026-10-07)

### 5.1 Adoption

| Library | Stars | Latest | Imported by |
| :--- | ---: | :--- | :--- |
| Bubble Tea | 45,318 | v2.0.10 (2026-09-24) | v1 14,989; v2 961 |
| Lip Gloss | 11,905 | v2.0.6 | v1 9,630; v2 2,560 |
| Huh | 7,197 | v2.0.3 | v2 333 |
| Glamour | 3,725 | v2.0.1 | v1 839; v2 11 |
| ultraviolet | 392 | no tags | 47 |
| tview | 14,124 | v0.42.0 (2025-08-27) | 3,836 |
| tcell | 5,223 | v3.5.0 (2026-09-11) | v2 3,555; v3 275 |
| termui | 13,591 | last tag 2019 | v3 780 |
| termdash | 3,043 | 2024-03 | 123 |
| vaxis | 117 | — | — |

- **Charm is dominant, and nothing in Go displaces it.**
  - Bubble Tea v2 shipped on 2026-02-24 under `charm.land/…`. About 6% of
    Bubble Tea's importers and 21% of Lip Gloss's have moved.
  - Every active Go agent tool found is on Charm v2: Crush, docker-agent,
    DeepSeek-Reasonix and mark3labs/kit.
  - kubectl-ai, Ollama's pickers and Plandex are on v1.
  - Charm archived mods on 2026-03-09 for Crush.
- **tcell v3** (2025-12-01) is the one modernised alternative core;
  lazygit moved to it.
- **The displacement is across languages.** opencode deleted its Go
  Bubble Tea TUI for OpenTUI (Zig and TypeScript).
- **The convergence is on main-screen rendering with native scrollback.**
  Codex, Claude Code, pi-tui, Ink's `<Static>` and ratatui's
  `Viewport::Inline` all do it. The owner already put `inline` in scope
  ([0003-REPORT](0003-REPORT-agent-tui-ecosystem-research.md) §11.5).
- **Crush's design** (reported):
  - one root model draws into ultraviolet's screen buffer;
  - components render with `Render(width)` or `Draw(scr, rect)`, and opt
    into capability interfaces;
  - dialogs stack as overlays;
  - streaming markdown re-renders after a stable prefix;
  - images use kitty placeholders with a half-block fallback.

### 5.2 Terminal protocols in Bubble Tea v2.0.10

| Protocol | Bubble Tea |
| :--- | :--- |
| Kitty keyboard | requested, and restored on exit |
| Mode 2026, synchronized output | queried only when `shouldQuerySynchronizedOutput` allows it. With `TERM_PROGRAM` or `SSH_TTY` set, that needs a known terminal in `TERM` or `WT_SESSION` (`tea.go:968-990`) |
| Mode 2027, graphemes | enabled when reported |
| OSC 52, OSC 9;4, OSC 8, window title, cursor, OSC 10/11 | through `View` fields, commands, or Lip Gloss |
| Mode 2031, dark and light | decoded, never enabled (`termcap.Prober` enables it, I2) |
| Mode 2048, in-band resize | decoded, never enabled |
| Images, notifications | not in Bubble Tea; `x/ansi` has encoders |

- **OSC 7501,** a "Program Status Protocol" for idle, working, blocked
  and done, was added to vaxis on 2026-10-06. Its spec is v0.1, and
  terminal support is unverified.
- **Risks:**
  - ultraviolet has no versions, and one consumer swaps in a fork.
  - Huh, Glamour and fang release slowly; Harmonica's last release was
    2022.

## 6. What the findings mean

- **The integration helpers can be native without importing a
  framework.** The tools are a standard-library flag type, a tagged
  struct, a method-set stream source, and an `ExitCode()` convention
  (§3).
- **0013's fallback needs two fixes first:** a registry loop that knows
  when its program is gone (I1), and a hook that undoes modes set outside
  Bubble Tea (I2).
- **The rest of the library needs native forms** of what a CLI touches:
  - glyphs, profile and size for `workspace` (I3);
  - plain rendering (I4);
  - results, exit statuses and argument vectors for `command` (I5–I8);
  - enum text (I9);
  - environment-only capabilities (I10);
  - byte forms of terminal services (I11);
  - a testing kit (I12).
- **Canonicalization is cheaper now than at `v1`:** the collisions (C1)
  grow with each planned package.
- **The hardening findings** (H1–H11) are each small and independent.
- **The landscape confirms the foundations.** Charm v2 stays; ultraviolet
  stays behind `internal/`; `inline` and the harness are the planned
  records most of the field has already built.

## 7. Not verified

- Kilo's fatal path exiting 0 is an inference from removed code.
- The OSC 7501 terminal support.
- kitty's mode 2027.
- Claude Code's renderer figures, from secondary press only.
- The pkg.go.dev counts for `x/exp/golden` and `teatest`, which may be
  counting artefacts.

## 8. Re-read by the coordinator

- **go-tui-lib:**
  - I1: `command/registry.go:82-88`, `command/dispatch.go:54-90`, and
    Bubble Tea `tea.go:1188-1197`.
  - I2: `termcap/prober.go:125-131`, `:218-236`, `:305-312`.
  - I3: `workspace/workspace.go:196-204`, `:322-336`.
  - H1: `workspace/render.go:142-152`.
  - H2: no `recover()` outside tests.
  - H8: the must-read list.
  - H10: `layout/size.go:270-277`.
- **Bubble Tea:** `tea.go:968-990`, the synchronized-output check.
- **The framework probe outputs:**
  - kong cases A, B, C1, C2 and D, and the `ExitCode` exit 130;
  - cobra's "flag needs an argument" and its 150-byte help;
  - the shared struct across kong, go-arg and go-flags;
  - the `mode` package's dependency list.
- **The agent sources:**
  - Codex `tui.rs:463-469`, `exec/src/lib.rs:1-6`,
    `event_processor_with_human_output.rs:516-531`;
  - Grok `xai-tty-utils/src/lib.rs:1077-1092`;
  - Pi `main.ts:112-123`;
  - opencode `tui.ts:60`.
- **The adoption figures:** the passes' raw GitHub and pkg.go.dev captures.
