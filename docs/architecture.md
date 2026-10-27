# Architecture

`go-tui-lib` is the repository of the Go module
`github.com/maccavelli/go-tui-lib/v2`, whose releases up to `v1.13.0`
were at `github.com/maccavelli/go-tui-lib`
([0020-MADR](decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
A4): a library of terminal-UI packages on
the Charm v2 stack, which a Go program stacks on its own command line
to have a TUI beside it. It has no binary. A nested adapter module may sit
beside it to carry a dependency the root must not; none is built (Modules,
below).

## What it is

- **A library only.** Each capability is a top-level directory, with the
  package named after it. There is no root package. Helpers shared between
  packages go under `internal/`.
- **Go 1.27.2**, with no `toolchain` line.
- **Ten packages,** and five internal ones, for multi-pane terminal
  workspaces, terminal capabilities and services, a command registry run
  from keys, slash lines, agents and the program's own CLI, and the
  foundations every package uses.
- **No CLI front end.** The program brings its own command line, and the
  library imports no CLI framework
  ([0012-MADR](decisions/0012-MADR-bring-your-own-cli.md)).

## Packages

```text
 launch        start the TUI from a CLI   → command, glyph, termcap, internal/enum, internal/limits; bubbletea, colorprofile, x/term
 launch/launchtest  fake streams for tests → launch, termcap (tests)
 palette       the command palette        → workspace, command, command/commandtea, keymap, fuzzy, layout, when, internal/enum, internal/sanitize; bubbletea, bubbles/textinput, lipgloss, x/ansi
 workspace     Bubble Tea pane host       → layout, theme, glyph, keymap, keymap/keymaptea, internal/cells, internal/enum, internal/limits, internal/sanitize, command, when; bubbletea, bubbles/key, bubbles/help, colorprofile, x/ansi
 keymap        keys to command IDs        → command, when, glyph, termcap, termkey, internal/enum; x/ansi
 keymap/keymaptea    keys in tea        → keymap, termkey, command, when; bubbletea, bubbles/key, bubbles/help
 command       the command registry       → when, internal/enum, internal/sanitize
 command/commandtea   commands in tea    → command; bubbletea
 when          availability expressions   → internal/enum
 internal/cells  the reused frame buffer  → layout; ultraviolet, x/ansi
 termsvc       terminal services          → termcap, internal/enum, internal/sanitize; x/ansi
 termsvc/termsvctea   services in tea    → termsvc, termcap, internal/teamsg; bubbletea, x/ansi
 termcap       capabilities, the probe    → internal/enum, internal/sanitize; x/ansi
 termcap/termcaptea   the probe in tea   → termcap, internal/termevent, internal/teamsg; bubbletea, x/ansi, colorprofile
 termcap/termcaptest  fake terminals      → termcap; bubbletea, x/ansi, colorprofile (tests)
 termkey       keys, and their decoder    → internal/enum; standard library
 internal/termevent   pass-through events → ultraviolet
 internal/termevent/termeventtest  those events for tests → ultraviolet, bubbletea
 theme         palettes, roles, styles    → glyph, internal/enum; lipgloss, colorprofile
 fuzzy         match and rank candidates  → internal/enum; x/ansi
 safetext      untrusted text, made safe  → glyph; standard library
 msgcat        message catalogue          → standard library
 glyph         Unicode and ASCII glyphs   → internal/enum
 internal/enum  enum names and text forms → standard library
 internal/teamsg  the one-message command → bubbletea
 internal/sanitize  untrusted display text → safetext; x/ansi
 internal/limits  the window clamp        → standard library
 layout        geometry and state         → internal/enum
 tuitest       golden rendering           → glyph; x/ansi, colorprofile (tests and examples only)
```

| Package | What it holds |
| :--- | :--- |
| `when` | `Parse` and `MustParse` → `Expr` (`Eval`, `String`, `Keys`), VS Code's when-clause grammar with RE2 regexes, or a `SyntaxError` with its offset; `Check` against `Keys`; `Context`, `Map`, `Layered`, `Value`; typed `Key[T]` |
| `command` | `Command`, `ID`, `New[A]` and `MustNew[A]` and their options, `SchemaOf` (JSON Schema 2020-12 from Kong-aligned tags), `ArgsOf`, `ArgError`; `Registry` (`NewRegistry` with `WithGate`, `WithGatePolicy`, `WithAuditor`, `WithPrefixer`, `WithLoop` and `WithoutBuiltins`; `Attach`, a loop that can be detached; `Register`, `ReplaceSource`, `Lookup`, `Slash`, `ParseSlash`, `ParseArgs`, `Complete`, `All`, `Available`, `Watch`, `Dispatch`, `Run`, `Cancel`, `CancelAll`, `Remove`, `Forget`, `ForgetAll`, `Version`); the built-in IDs, `BuiltinList`, `BuiltinDescribe` and `BuiltinQuit`; `Params` and `Param`; `Format` and `WriteResult`; the policy, `GatePolicy`, `GatePolicyFunc`, `DefaultGatePolicy`, `Gate`, `GateFunc`, `AllowIf`, `Verdict`, `Auditor`, `AuditorFunc`, `SlogAuditor`; a `Request`'s `WhenContext`; the errors, each with its exit status, and `PanicError`; `LoadDir` and `LoadDirWith`, with its limits; `WithMaxArgBytes`; `FromMCPPrompts`, `FromACP`; `MCPTools`, `CallMCP`, `ACPCommands`, `Manifest`; `WithLoop` and `LoopMsg`; `Effect`, the follow-up a host runs, and the messages |
| `command/commandtea` | `Cmd` and `Effect`, between `command.Effect` and `tea.Cmd`; `Dispatch`, `Watch`, `WatchContext`, `RunLoop`, `Attach` and `WithLoop`, the registry's forms for a program's `Update` and `Send` |
| `glyph` | `Set` (4 border styles, separators with a cross and four tees, focus marker, ellipsis, scroll, bullet, badge brackets, `Prompt` and `Mask`, the states `Done`, `Failed`, `Running` and `Pending`, `Heading`, `MeterFill` and `MeterTrack`, `Control`, and `Keys`, a `KeyGlyphs` of the macOS modifier and arrow labels), `Unicode()`, `ASCII()`, `For(utf8)`; `Tier` (`TierUnicode`, `TierLegacy`, `TierASCII`) and `Tier.Set`; every glyph one cell, except `Keys`' ASCII twins, which are words such as `opt+` and `up` |
| `fuzzy` | `Matcher`, from `NewMatcher(query)` with `WithCase`, `WithDelimiters`, `WithExtended`, `WithMaxAlign` and `WithNormalizer`: grapheme-cluster matching with smart case, a `Grade` (`Exact`, `Prefix`, `Boundary`, `Subsequence`) and a `Score`; `Match`, `Range`, `Ranked[T]`; `Cells`, a match's ranges in cells; `Index`, a lossy candidate index for large lists, with `WithMaxCandidates` and `WithMinOverlap` |
| `safetext` | `Rules` and the presets `Text`, `Strict`, `Line`, `Command` and `StatusLine`; `Clean`, `IsClean`; `Sanitizer`, a streaming `io.Writer` with `Close`; `SafeURL` and `URLs`: ECMA-48 with the DEC parser's recovery, C1 code points read as their 7-bit forms, invalid UTF-8 as one U+FFFD per maximal subpart, and the Bidi_Control set removed unless kept ([0020-MADR](decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md) D9, A3) |
| `msgcat` | `MessageID`, `Catalog`, `CatalogFunc` and `Table`, a program's message catalogue; `Format`, a message from it or its English default; `Check`, a catalogue's fmt verbs against the defaults' ([0024-MADR](decisions/0024-MADR-opt-in-and-opt-out-coverage.md) D9) |
| `launch` | `Choice` and `Flags` (`--mode`, `--tui`, bound natively in each framework); `Streams`, `StreamSource`, `FromSource`; `Config`, with `AllowDumb` and `IgnoreCI`, `Decide` → `Decision` with `Target`s and a `Reason` token; `ColorProfile` and `GlyphTier`, its helpers; `Run[M]` with `WithRegistry`, `WithRestorer`, `OnStart`, `WithProgramOptions`, `WithFilter`, `WithoutEpilogue`, `WithEpilogueWriter`; `ErrNotStarted`, `ErrCrashed`; `Frame`; `ExitCode`, `ExitError`; `Restorer` |
| `launch/launchtest` | `Terminal` (a fake terminal stream with a size and typed keys), `Pipe`, `Streams`, `Env` |
| `palette` | `Picker`, from `NewPicker(ws)` with `WithLimit`, `WithGrouping`, `WithPlaceholder`, `WithLogger`, `WithGOOS`, `WithRecents` and `WithoutInput`: an overlay pane that streams hits from its providers off the event loop, ranks them with `fuzzy`, asks for a hit's arguments, and runs it on the loop; `Register`, `Open` with `WithID`, `WithAnchor` and `WithSize`, `OpenCommand`, `DefaultRules`, `CompletionRules`, the command IDs `OpenID` to `CompleteID` and `PickerScope`, `SetQuery` and `SelectedMsg`; `Provider` and its optional `Discoverer`, `Starter`, `Stopper`, `Prefixed`, `Debouncer` and `Opener`; `Search`, `Hit`, `Availability`, `Grouping`; the providers `FromRegistry`, `Static` with `Item`, `Panes`, `Files` with `FileOption`, `Walker`, `WalkerFunc`, `FSWalker`, `PathFilter` and `PathFilterFunc`, and `Recents`; `Frecency`, with `FrecencyOption`, its JSON form and `ErrRecentsVersion` |
| `internal/enum` | `Names`, an enum's tokens, on `uint8` or `int`, and `Bits`, a bit set's, on `uint8` or `uint16`, for an enum's `String`, `MarshalText` and `UnmarshalText`; `Name`, `Marshal` and `Unmarshal` over a bare table |
| `internal/teamsg` | `Cmd`, the `tea.Cmd` that delivers one message |
| `theme` | `Palette` for dark, light and unknown backgrounds; `LightDarkColor` and `ProfileColor`; `Styles`; `New(profile, background, glyphs)` with `WithPalette` and `WithPaletteFor`; `FromDark`; `Border(style)`; text forms for `Background` and `BorderStyle` |
| `layout` | `Rect`, `Size` (fixed, percent, ratio, fill; min, max, shrink order), `Node` (`Pane`, `Split`, `Responsive`, or a custom node), `Solve` → `Plan`; `State` (JSON); the sidebar presets and their `With…` options |
| `workspace` | `Pane` and its optional interfaces, `PlainViewer` among them; `Workspace` (routing, focus, chrome, resize, zoom, hide, overlays, cursor, `View`, `RenderPlain`, `help.KeyMap` and `Help`, the width method, a following theme with `WithGlyphs`, `WithProfile`, `WithBackground` and `GlyphThemeBuilder`, `WithSize`, clamped to `MaxSide` and `MaxCells`, `SetBackground`, `Panes`, `PaneAs`, `WhenContext`); `Wrap`; its keys through a keymap: `DefaultRules`, `WithBindings`, `SetBindings`, `KeyPath`, `KeyContexter`, `PushMode`, `RequestMsg` and `WithoutOwnCommands`; `Bindings` and `Theme`, for a pane to show a key and draw alike; `CommandConsumer` and `CommandMsg`, for a pane or overlay that runs some commands itself; `Close`, an overlay's request to close, and `OverlayClosedMsg`, what an overlay is sent when it closes; `Commands` and the context keys, `FocusedPaneKey()` to `HeightKey()` |
| `keymap` | the notation: `Stroke`, `Sequence`, `ParseStroke`, `ParseSequence`, `Event`, `Normalize`, `Label`, `ParseAliases`, `DisplayAliases` and `CopyStrokes`; `Keymap`, from `New(Settings, defaults, layers…)`, with `Rebuild`, `Lookup`, `Bindings`, `Reachable`, `Shortcut`, `Conflicts` and `Rules`; `Binding`, `Default` and `Alternative`, `Layer`, `Location`, `Path`; `BindingConflict` and `ConflictKind`; `Features`, `FeaturesFrom`, `KeyFacts` and `FactsFrom`; `Shown`, the bindings help shows; `Dispatcher` (`Key` and `Expire`, over `termkey` keys; `SetKeymap`, `Pending`, `ActiveKeys`, `Explain`) with `Outcome`, whose `Wait` and `Timeout` a host times, and `TimeoutMsg`; `LoadVSCode`, `LoadError`, `ExportVSCode` and `Schema`; `Describer`, `DescriberFunc` and `FromRegistry` |
| `keymap/keymaptea` | `Update`, the `Dispatcher` driven by tea's key messages with `tea.Tick` for a pending sequence; `FromTea` and `ToTea`, between `tea.Key` and `termkey.KeyEvent`, field for field; `KeyBinding`, `Help` and `BubblesKeys`, for bubbles |
| `internal/cells` | `Frame`: a reused cell buffer that draws strings into rectangles with a chosen width method |
| `internal/sanitize` | `Line` for one line of plain text and `Styled` for a pane's view, which keeps SGR and OSC 8 only, and `HasControl`: calls of `safetext` under rules equal to their behaviour on well-formed input (0020-MADR D9 S5, A3); `Truncate`, never wider than asked; `Token` |
| `internal/limits` | `MaxSide`, `MaxCells` and `Clamp`, the window clamp `workspace` and `launch` share |
| `termcap` | `Probe`, from `NewProbe`: an I/O-free engine (one batch of queries ended by DA1, a deadline the host times, mode 2031 and its reset), driven by `Start`, `HandleReply`, `HandleTimeout` and `HandleHook`, and drained through `NextEvent`'s `ProbeEvent`s; `ParseReply`, a reply's bytes as a `Reply` of a `ReplyKind`, colours read by XParseColor's grammar at 16 bits; `ColorProfile`; `WithoutModeQuery`; `EnvCaps`, the environment's facts with no program; `Caps` of `Fact`s with origin and reason, as JSON (v2); `ErrUnknownName`; `Env`, whose lookups ignore case on Windows; `FromEnv` and `Identity`; the keyboard, link and notification views; `TmuxQuery`; `Report` and `Findings` |
| `termcap/termcaptea` | `Prober`, from `NewProber`: the `Probe`'s host in a Bubble Tea program (the batch on the first `tea.EnvMsg`, the deadline from `Init`, tea's own replies turned into `Reply`s, `CapsMsg` after tea's colour profile, `IsReplyFragment`); `KeyboardEnhancements`; `FromColorProfile` and `ToColorProfile` |
| `termcap/termcaptest` | `Terminal`, a scripted fake terminal, with `SetRunTimeout` and `Reply`; `Profile` and seven profiles; `Run` |
| `termkey` | `KeyEvent` (`Code`, `Mod`, `Text`, `ShiftedCode`, `BaseCode`, `IsRepeat`, `Release`; `Keystroke` and `String`, as `tea.Key`'s); `Mod`; the key codes, with ultraviolet's values and names; `Decoder` (`Decode`, `Pending`, `Flush`; `MaxPaste` 1 MiB and `MaxSequence` 4 KiB by default) yielding `Input`s of an `InputKind`: a key, a bracketed paste, or a sequence it does not decode, with its bytes for `termcap.ParseReply`; `DefaultEscTimeout`, 50 ms. No goroutine and no clock |
| `termsvc` | `Notifier` (OSC 99, 777, 9 or the bell; focus policies; `WithNotifyFilter`; a `Backend` or `BackendFunc`; `NotifyResultMsg`; `SetCaps` and `SetFocus`; `Prepare`, the bytes and a `deliver` that reports, or calls the backend; `Sequence`, the bytes alone; `WithBackendTimeout`); `PrepareCopy`, `CopySequence` and `CopiedMsg`, with a `Clipboard` or `ClipboardFunc`; `Link`, `LinkDisplay`, `LinkPolicy`; prompt marks; `Wrap`; `SanitizeTitle`; the activity beacon's sequence; `Pointer`; `ProgressSupported` |
| `termsvc/termsvctea` | `Notify` and `NotifyContext`, which write `Prepare`'s bytes and call a backend inside the command; `Observe`, `termcap.CapsMsg` and tea's focus messages to the `Notifier`; `Copy` and `CopyContext`, OSC 52 through tea; `Open`; `ActivityBeacon` |
| `internal/termevent` | `Decode`: the ultraviolet events tea passes through untranslated, as plain values |
| `internal/termevent/termeventtest` | those events built for tests outside `internal/termevent` |
| `tuitest` | `Golden` across {colour, no colour} × {UTF-8, ASCII} × widths; `Case.Profile` and `Case.Glyphs`; `Fits`; `Text` for a single file; `Annotate` |

- **`layout` has no Charm import,** so its solver can serve any front end.
- **The Charm-free cores** are `command`, `fuzzy`, `glyph`,
  `internal/enum`, `internal/sanitize`, `keymap`, `layout`, `msgcat`,
  `safetext`, `termcap`, `termkey`, `termsvc` and `when`: none links a Charm
  module but `x/ansi`, so a line-mode CLI can use them without Bubble Tea
  ([0020-MADR](decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  D1, D11). `internal/conformance` checks their whole import closure, and
  depguard's `charmfree` rule their own imports (Tooling).
- **Imports point downward.** `when` imports the standard library and
  `internal/enum` alone, so the keymap and any front end can use it
  ([0006-MADR](decisions/0006-MADR-command-registry.md)'s "only the
  standard library" predates the enum tables of
  [0014-MADR](decisions/0014-MADR-native-integration-api.md) W0). `command` imports `when`,
  and no Bubble Tea; `command/commandtea` holds its Bubble Tea forms.
  `workspace` imports
  both to publish its commands and context keys; `command` never imports
  `workspace` ([0006-MADR](decisions/0006-MADR-command-registry.md)
  §1). `keymap` imports `command`, `when`, `glyph` and `termcap`'s facts;
  `workspace` imports `keymap`, which never imports `workspace` or
  `layout` ([0007-MADR](decisions/0007-MADR-keymap-engine.md) §1). `keymap` imports
  `termkey` and no Bubble Tea; `keymap/keymaptea` holds its Bubble Tea forms.
  `palette` imports `workspace`, `command` and `fuzzy`, and nothing
  imports `palette` ([0008-MADR](decisions/0008-MADR-command-palette.md)
  §1).
- **The registry reads lock-free.** Each write publishes a new immutable
  snapshot behind an `atomic.Pointer`; `Lookup`, `Available` and the
  exports read one, from any goroutine. A `Loop` command runs on the
  program's event loop, through `Dispatch` from `Update`, or through
  `WithLoop`'s `LoopMsg` when an agent's goroutine calls `Run` (A7).
- **The registry speaks no protocol.** `MCPTool`, `MCPCallResult` and
  `ACPCommand` are plain structs with MCP 2026-07-28's and ACP's field
  names; no SDK is imported, and the host owns the transport.
- **`workspace` draws each frame into one reused buffer,**
  `internal/cells.Frame`: the panes, then the separators, then the overlays,
  each into its rectangle. It records those rectangles, top first, and a
  mouse event takes the first that contains it. A frame is drawn only when
  something changed since the last; otherwise the last one is returned.
  The view cache is keyed on a pane's kind, ID, size, focus, width method
  and theme generation
  ([0004-MADR](decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) §2).
- **The workspace measures as Bubble Tea writes:** `ansi.WcWidth` by
  default, switching to `ansi.GraphemeWidth` on the terminal's mode 2027
  report, unless `WithWidthMethod` fixes it (§3).
- **`internal/cells` and `internal/termevent` are the only importers of
  ultraviolet,** which has no tagged release, so an upstream change is one
  package's fix (§1; [0005-MADR](decisions/0005-MADR-terminal-capabilities-and-services.md) §1).
  `internal/termevent/termeventtest`, under the second, builds its events
  for other packages' tests.
- **`termcap` learns, and `termsvc` acts.** The probe reads the
  environment from the `Env` its host passes, `tea.EnvMsg` under
  `termcaptea`, never from the process; it does no I/O, and its host
  writes what it asks for ([0020-MADR](decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  A2). Neither
  package starts a process, and commands such as `TmuxQuery` are returned
  for the program to run
  ([0005-MADR](decisions/0005-MADR-terminal-capabilities-and-services.md)).
  A source test in each refuses `os.Getenv`, `os.LookupEnv`, `os.Environ`
  and `os/exec`.
- **Nothing writes to the terminal.** No package writes to `os.Stdout` or
  `os.Stderr`, prints with `fmt.Print*` or the `print` builtins, logs
  through `log`'s standard logger or `log/slog`'s default logger, calls
  `signal.Notify` or sets `AltScreen`; `internal/conformance` checks
  this by type (see Tooling).

## Modules

0010-MADR
([0010-MADR-nested-adapter-modules.md](decisions/0010-MADR-nested-adapter-modules.md))
puts an adapter with a dependency the root must not carry in a nested
module. The Cobra and Kong front ends it planned, `command/cobracmd` and
`command/kongcmd`, were built and released at `v0.1.0`, then retired by
[0012-MADR](decisions/0012-MADR-bring-your-own-cli.md): retracted at
`v0.1.1` and removed. `stream/glamourmd` is planned:

```text
 stream/glamourmd   (planned)   → root vX.Y.Z, published; glamour
 .                  (root)      → the Charm v2 stack; never an adapter
```

- **The dependency runs one way.** An adapter requires a published root
  version, with no `replace`; the root never requires an adapter. A
  consumer of the root therefore never sees an adapter's dependency in
  its `go.sum` or module graph
  ([0010-REPORT-nested-modules-and-adapter-sources.md](reports/0010-REPORT-nested-modules-and-adapter-sources.md)
  §2).
- **Each module has its own tags:** `vX.Y.Z` for the root,
  `<dir>/vX.Y.Z` for an adapter. [guides/releasing.md](guides/releasing.md)
  has the procedure.
- **`go.work`** lists every module, for development, and is committed;
  `go.work.sum` is ignored (0010-MADR A1). Every gate also runs
  with `GOWORK=off` (Tooling).

## Tree

```text
README.md                   repository entry; links here
LICENSE                     Apache License 2.0
AGENTS.md                   rules for agents: dependencies, TUI conventions,
                            records, checks, identifiers, commits
go.mod, go.sum              the root module and its requirements
go.work                     the workspace: every module, for development
                            (go.work.sum is ignored)
Makefile                    development targets (below)
.golangci.yml               golangci-lint configuration, with depguard
.markdownlint-cli2.jsonc    Markdown lint configuration
.gitattributes              LF line endings everywhere
.gitignore
.github/workflows/
  ci.yml                    CI
  docs.yml                  the documents' checks, for a docs-only change
scripts/
  go-precheck.py            the pre-add check, per module
  go-precheck.sh            a shim that runs it, for the machine-wide gate
  go-modules.py             lists the modules; --check compares go.work
                            with the tracked go.mod files
  go-modules_test.py        its offline test
  go-fuzz.py                fuzzes each fuzz target of a package in turn
  go-fuzz_test.py           its offline test
  go-apicheck.py            the API diff gate: incompatible changes since
                            each module's previous tag
  go-apicheck_test.py       its test, on throwaway repositories
  apicheck.allow            the incompatible changes a PLAN allows
  go-examples.py            builds and runs the framework examples, and
                            checks the guides' excerpts of them
  go-examples_test.py       its offline test
  go-precheck_test.py       the pre-add check's gofmt step and its files,
                            and the shim, on throwaway repositories
tools/verify/               python3 -m tools.verify: a step's gates, its
                            mutation proofs, the gates on another host, the
                            identifier scan, planned type names; run by a
                            person or an agent, never by make or CI; its
                            hosts come from a local config, never committed
  tests/                    its unittest suite, with fake ssh and scp
glyph/ theme/ layout/ workspace/ tuitest/ termcap/ termsvc/
when/ command/ fuzzy/       the packages; goldens under testdata/golden/
safetext/                   where a package renders
termkey/                    keys and their decoder; its corpus golden is
                            replayed by internal/termevent's differential test
msgcat/                     the message catalogue
termcap/termcaptea/         the probe's Bubble Tea host
termcap/termcaptest/        fake terminals for tests
termsvc/termsvctea/         termsvc's Bubble Tea commands
command/commandtea/         command's Bubble Tea forms
keymap/keymaptea/           keymap's Bubble Tea forms: Update, help, key.Binding
launch/                     start the TUI from a program's own CLI
palette/                    the command palette and its providers; goldens
                            under testdata/golden/
launch/launchtest/          fake streams for a program's tests
internal/enum/              enum names and text forms
internal/teamsg/            the one-message command
internal/sanitize/          the one sanitizer for untrusted display text
internal/limits/            the window clamp
internal/cells/             the reused frame buffer, an ultraviolet importer
internal/termevent/         pass-through events, the other ultraviolet importer
internal/conformance/       the conformance scan: terminal ownership,
                            environment, exit and spawn, type names,
                            stability lines, opaque options, enum text,
                            the Charm-free cores' closure (tests only)
tuitest/internal/clash/     proves tuitest's -update flag does not clash (tests only)
testdata/frameworks/        the framework examples: programs on flag, Cobra,
                            Kong and urfave/cli, built in a module of their
                            own (go.mod.tmpl), and their cases
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  glossary.md               one name, one meaning: the shared and reserved
                            type names
  decisions/                MADR and PLAN records
  reports/                  REPORT records
  guides/                   how-to guides: workspaces, terminal
                            capabilities, commands, releasing, the
                            defaults and their controls
```

## Dependencies

- **Required:** `charm.land/bubbletea/v2` v2.0.10, `charm.land/lipgloss/v2`
  v2.0.6, `charm.land/bubbles/v2` v2.2.1,
  `github.com/charmbracelet/colorprofile` v0.4.3,
  `github.com/charmbracelet/x/ansi` v0.11.8,
  `github.com/charmbracelet/x/term` v0.2.2, the version Bubble Tea selects,
  and `github.com/charmbracelet/ultraviolet` at
  `v0.0.0-20260811164956-006e29f97886`, the pseudo-version lipgloss's
  requirement selects; it has no tagged release
  ([0004-MADR](decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) §1).
- **Named but not yet required:** `github.com/maccavelli/go-selfupdate-lib`
  (formerly `go-core-lib`), for `updatetea`.
- **Refused by `depguard`,** in source and tests, in every module:
  - the Charm v1 paths `github.com/charmbracelet/bubbletea`, `…/lipgloss`
    and `…/bubbles`;
  - `github.com/maccavelli/mcplib`;
  - `github.com/modelcontextprotocol/go-sdk`;
  - `github.com/maccavelli/go-llmprovider-sdk`;
  - fang: `charm.land/fang/v2` and `github.com/charmbracelet/fang`;
  - the CLI frameworks `github.com/spf13/cobra`, `github.com/spf13/pflag`
    and `github.com/alecthomas/kong`
    ([0012-MADR](decisions/0012-MADR-bring-your-own-cli.md)).
- **Kept to one module by `depguard`:** `charm.land/glamour/v2` to
  `stream/glamourmd`. The rule covers `$all` less `!**/<dir>/**`.
- **Kept to two packages by `depguard`:** `github.com/charmbracelet/ultraviolet`
  to `internal/cells` and `internal/termevent` (with
  `internal/termevent/termeventtest` beneath it), test files included.
- **Kept to one package by `depguard`:** `github.com/charmbracelet/x/term`
  to `launch`, so only `launch` asks the terminal for its state
  ([0013-MADR](decisions/0013-MADR-cli-integration-helpers.md) §11).
- **Kept out of the Charm-free cores by `depguard`'s `charmfree`:**
  Bubble Tea, Bubbles, Lip Gloss, colorprofile, ultraviolet, `x/term` and
  `internal/teamsg`, in the cores' non-test files; `internal/conformance`
  checks what they link through other packages
  ([0020-MADR](decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  D11).
- **Kept to what their records allow by `depguard`'s `termkeystd`,
  `safetextstd` and `msgcatstd`:** `termkey` to the standard library and
  `internal/enum`, `safetext` to the standard library, `glyph` and
  `internal/enum`, and `msgcat` to the standard library, in their non-test
  files, as `internal/conformance` checks their whole closures
  ([0021-MADR](decisions/0021-MADR-termkey-closure-gate.md) N3, A1;
  [0024-MADR](decisions/0024-MADR-opt-in-and-opt-out-coverage.md) D9).

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`,
  `modernize`, `tidy`, `tidy-check`, `vuln`, `fuzz`, `apicheck`,
  `examples`, `pre-add-check`, `release-check`, `help`.
- **Per module.** `test`, `vet`, `lint`, `modernize`, `tidy-check` and
  `vuln` run once in each module's directory with `GOWORK=off`, taking the
  list from `scripts/go-modules.py`; a failure to list the modules fails
  the target.
- **`make lint`** runs `make modernize`, then `golangci-lint run -c
  <root>/.golangci.yml ./...` three times per module: `GOOS=linux`,
  `darwin` and `windows`, each with `CGO_ENABLED=0`.
- **`make modernize`** runs `go fix -diff ./...` for the same three
  targets, with `CGO_ENABLED=0`, and fails on any suggestion: `go fix
  -diff` exits 1 when it prints a diff.
- **`make release-check`** runs the pre-add check over every module and
  file, before a tag.
- **`internal/conformance`** is the conformance scan, run by `go test`.
  - It finds every module by its `go.mod`, and type-checks each module's
    packages from that module's directory with `go/types` and the source
    importer (`importer.ForCompiler(fset, "source", nil)`), which resolves
    imports with `go list` in the working directory. cgo is off for the
    scan, so no C toolchain is needed.
  - Every identifier is resolved to its object, so a renamed or dot
    import, a function value or a method value is a use, and a local name
    that only matches a rule is not.
  - Outside tests it refuses `os.Stdout` and `os.Stderr`; `fmt.Print`,
    `Printf` and `Println`; `log`'s writing functions and `log.Default`
    and `log.Writer`; `log/slog`'s `Debug`, `Info`, `Warn` and `Error`
    and their `Context` forms, `Log`, `LogAttrs` and `Default`; the
    `print` and `println` builtins; `signal.Notify`; and a write to
    `tea.View`'s `AltScreen`, however the field is reached.
  - It also refuses a read of the process's environment, `os.Exit`, and
    starting a process (`os.StartProcess`, `os/exec`, `syscall`'s `Exec`,
    `ForkExec` and `StartProcess`, `tea.Exec` and `tea.ExecProcess`).
    `allowed` lists the one exception, by file and rule: `tuitest`'s
    `TUITEST_UPDATE` switch.
  - The packages it must read come from `go list`, so a package it skips
    fails the test.
  - `TestNoTypeNameMeansTwoThings` fails when two public packages export
    a type of the same name, unless `sharedNames` and
    [glossary.md](glossary.md) allow it.
  - `TestEveryPackageStatesStability` fails when a package's
    documentation does not end with its stability line (AGENTS.md, "API
    conventions").
  - `TestCoresAreCharmFree` lists each Charm-free core's non-test import
    closure with `go list -deps`, for `GOOS=linux`, `darwin` and
    `windows`, and fails on any Charm module but `x/ansi`, naming the
    shortest import chain to it; `cores` names the packages.
  - `TestNarrowClosures` does the same for each core whose record
    allows less, `termkey` and `safetext` today, and fails on any package outside
    the standard library, the core and what `narrow` allows it, on an
    entry `cores` does not list, and on an allowance its closure no
    longer holds ([0021-MADR](decisions/0021-MADR-termkey-closure-gate.md)).
  - It reads each package's files for the host's `GOOS`; CI's three
    operating systems cover the rest.
- **`make fuzz`** fuzzes, in every module, each fuzz target of every
  package whose tests declare one, for `FUZZTIME` (default 20s):
  `scripts/go-fuzz.py -a` finds the packages, so a new target needs no
  list.
- **`scripts/go-precheck.py`** runs, for each module that owns a given
  file (every module when none is given), in the module's directory:
  `gofmt` on its files; with `GOWORK=off`, the same three golangci-lint
  runs, `go vet` and `go test` on their packages, `go mod tidy -diff` and
  `govulncheck ./...`; `go test` again in workspace mode; and, for a
  nested module, no `replace` and a release version of the root. A Go
  file under `testdata` is formatted, but not vetted or tested. gofmt's
  own failure, on a file it cannot read or parse, fails the check, and
  with no file list the files are the tracked Go files the work tree has
  ([0015-MADR](decisions/0015-MADR-precheck-gofmt-errors.md)). It ends
  with `scripts/go-modules.py --check`, then:
  - with no file list, the API diff gate;
  - with no file list, or one naming a file under `testdata/frameworks`,
    the examples gate.

  `make pre-add-check` runs it, and so does the machine-wide agent gate
  before an agent `git commit` that stages Go files.
- **`make apicheck`** runs `scripts/go-apicheck.py`. For each module it
  takes the newest release tag merged into `HEAD` that does not contain
  `HEAD`, so a tagged commit compares with the tag before it. It exports
  that tag's API and the tree's with apidiff, installed at the
  Makefile's `APIDIFF_VERSION` into a temporary directory and never added
  to `go.mod`, and judges apidiff's output against
  `scripts/apicheck.allow`. An unlisted incompatible change fails, and so
  does a listed one apidiff no longer reports. A module with no tag is
  skipped. When the tag's module path and the tree's differ only by a
  major version's `/vN`, the tag's own import paths are moved to the
  tree's before it is exported, so only real changes are reported
  ([0020-MADR](decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  A4.2).
- **`make examples`** runs `scripts/go-examples.py`:
  - it copies `testdata/frameworks` into a temporary module, whose
    `go.mod` comes from `go.mod.tmpl` with a `replace` of the root;
  - it builds and vets the programs, and runs `cases.txt`;
  - it compares each guide block marked `<!-- from: … -->` with the
    program region it names;
  - `--check-guide` does only the comparison, and `--update` rewrites
    `go.mod.tmpl` and `go.sum` from `go mod tidy`.
- **`.golangci.yml`:**
  - 21 linters, including `depguard`;
  - `revive`'s `exported`, `package-comments` and `var-naming` rules in
    place of `golint`;
  - errcheck with `check-blank` and `check-type-assertions`;
  - gofmt and goimports as formatters;
  - test files exempt from `errcheck`, `gosec`, `unparam`, `revive`,
    `gocritic` and `goconst`, but not from `depguard`.
- **Golden files** are written with
  `go test ./<pkg>/ -run <Test> -tuitest.update`, or with
  `TUITEST_UPDATE=1 go test ./...`, and read before they are committed. A
  test binary that defines its own boolean `-update` may use it instead.
- **CI** (`.github/workflows/ci.yml`) reads the Go version from `go.work`
  and caches on every module's `go.sum`. It has three jobs:
  - **`modules`** (Linux): the test of `go-modules.py`, then
    `go-modules.py --check`, then the module list as the matrix's input.
  - **`test (<module>, <os>)`** for each module on `ubuntu-24.04`,
    `macos-15` and `windows-2025`, in the module's directory with
    `GOWORK=off`:
    - **every OS:** `go test`, and `go test` again in workspace mode;
    - **Linux and macOS:** `go test -race`;
    - **Linux:** `go test -shuffle=on -count=2` and `LC_ALL=C go test`.
  - **`gates`** (Linux):
    - the fuzz script's test, then `make fuzz`, uploading every
      package's corpus (`**/testdata/fuzz/`) as an artifact on failure;
    - `go vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`, per
      module;
    - `make vet`, `gofmt`, `make tidy-check`, and `make lint` (with
      `make modernize`) with golangci-lint v2.14.0;
    - `make vuln` with govulncheck v1.8.0;
    - `apicheck`: the API diff gate's test, then `make apicheck`. The
      job's checkout fetches the whole history and its tags;
    - `examples`: the examples gate's test, then `make examples`;
    - `precheck test`: the pre-add check's own test;
    - `shellcheck` v0.11.0 (pinned by SHA-256 and first on `PATH`),
      `markdownlint-cli2` 0.23.2 and `actionlint` v1.7.12;
    - `verification tools`: `tools/` compiles, and its unittest suite
      passes, on the runner's `python3` 3.12.
  - One run per ref (`concurrency`, cancel in progress). Actions are pinned
    to commit SHAs.
  - **Documents only:** a push to `main` or a pull request that changes
    only Markdown at the root or under `docs/`, `.claude/`, `.grok/` or
    `.opencode/` skips `ci.yml` (`paths-ignore`) and runs `docs.yml`
    instead. Markdown under `command/testdata/`, which the tests read, is
    never skipped, and a tag push is never path-filtered
    ([0019-MADR-skip-full-ci-for-docs-only-pushes.md](decisions/0019-MADR-skip-full-ci-for-docs-only-pushes.md)).
- **Docs** (`.github/workflows/docs.yml`) runs on a push to `main` or a
  pull request that changes those documents, `.markdownlint-cli2.jsonc`
  or the workflow itself. One `ubuntu-24.04` job runs `markdownlint-cli2`
  0.23.2, with `ci.yml`'s command, and
  `scripts/go-examples.py --check-guide`, the guides' excerpt check. Its
  `paths` list `ci.yml`'s five `paths-ignore` patterns, and the two lists
  must agree. A change to code and documents runs both workflows.

## What is not here

- **Standard panes** (log tail, metrics view, scrolling text and
  Markdown), **overlay widgets** (dialog, toast), a help-footer pane
  beyond `workspace`'s `help.KeyMap`, and **`updatetea`.** Each is its own
  record. The command palette is `palette`, above.
- **A which-key panel** over `keymap`'s `ActiveKeys`, and **a line-mode
  picker** for a CLI without Bubble Tea, which may build on `fuzzy`
  (issue #8).
- **Terminal modes and inline scrollback** (`termmode`, `inline`): mode
  plans, teardown, restore bytes and the Windows console helpers, and the
  per-terminal scrollback strategy. Each is a later record; until then tea
  still asks for Kitty disambiguation everywhere
  ([0005-MADR](decisions/0005-MADR-terminal-capabilities-and-services.md) A4).
- **Image protocols** (Kitty placeholders, Sixel, iTerm2), which add
  `Query` values to `termcap`.
- **A CLI front end,** by design
  ([0012-MADR](decisions/0012-MADR-bring-your-own-cli.md)). `launch` starts
  the TUI from the program's own command line instead.
- **A release workflow, Dependabot, and any tag** other than the owner's.
