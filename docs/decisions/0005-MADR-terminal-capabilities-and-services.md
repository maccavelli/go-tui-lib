---
status: proposed
date: 2026-10-02
decision-makers: owner
consulted: 0003-REPORT-agent-tui-ecosystem-research.md; Charm v2 sources (bubbletea v2.0.10, ultraviolet at the commit go.sum pins, x/ansi v0.11.8); crush's capability layer as a design reference
informed: pi-go; go-core-lib
---
# Detect terminal capabilities with one sentinel-terminated probe that observes Bubble Tea's own queries, and offer notifications, clipboard, links and prompt marks as commands built from the result

## Context and Problem Statement

On 2026-10-01 the owner asked:

> i want to stay working on go-tui-lib, i want to ensure the
> bubbletea/lipgloss/charm stack is tightly integrated, coded idiomatically,
> based on go1.27.1 optimizations and standards. i want to enhance and expand
> the tui library functionality in 5 more ways that will bring benefit and
> value to the codebase functionality. i want a large, openstandards api
> surface, with a wide assortment of agentic commands available through the
> terminal UI and also the TUI. look at similar opensource projects across
> the web. dig into how they solved interesting problems and issues bringing
> functionality to reality. find ideas, features, enhancements, and
> optimizations. present 10 items to spec out and plan. future-proof is a
> focus.

On 2026-10-02, after the ten items were presented, the owner said: "write
findings into a report then follow recommendations and proceed." The
recommendation named this record's subject, terminal capabilities and
services, as one of the five expansions. It is the base that lets the
keymap, the theme and the agent widgets adapt to the terminal they run in.

The owner's standing direction also applies: build speculative API when it
is sensible, for extensibility, flexibility and idiomatic, modular design.

An agent session needs to know things about its terminal that no
environment variable reliably says, especially over SSH and inside tmux:

* whether keys are unambiguous (the Kitty keyboard protocol);
* whether the background is dark, and whether it will tell the program when
  that changes;
* whether a desktop notification, a clipboard write or a hyperlink will
  work, and in which dialect.

Every agent TUI studied writes its own detection, and each gets a different
part wrong. This record decides how go-tui-lib detects capabilities, and
how it turns them into terminal services.

Evidence (read-only, 2026-10-02, from module sources unless marked):

* **What Bubble Tea v2.0.10 already queries, and acts on.**
  * **Modes 2026 and 2027.** At startup, before `Init`, `Program.Run` sends
    DECRQM for synchronized output (2026) and grapheme clustering (2027)
    (`tea.go`, `RequestModeSynchronizedOutput + RequestModeUnicodeCore`).
    It does so only when `shouldQuerySynchronizedOutput` passes. That
    heuristic skips SSH sessions (`SSH_TTY`) and Apple Terminal, except for
    Windows Terminal and a list of known terminals.
  * **It acts on the replies.** On `ModeReportMsg` for 2026 it turns on
    synchronized updates. For 2027 it switches the renderer to
    `ansi.GraphemeWidth`.
  * **It acts on `CapabilityMsg`.** An XTGETTCAP reply of `RGB` or `Tc`
    upgrades the colour profile to TrueColor and sends a
    `ColorProfileMsg`.
  * **Kitty keyboard.** The renderer pushes the Kitty keyboard flags and
    writes `RequestKittyKeyboard` on the first render and whenever
    `View.KeyboardEnhancements` or `AltScreen` changes
    (`cursed_renderer.go`). The event loop has no case for
    `KeyboardEnhancementsMsg`, so a second reply changes nothing.
  * **On request only.** Background colour (`RequestBackgroundColor`),
    XTVERSION (`RequestTerminalVersion`), XTGETTCAP (`RequestCapability`),
    the clipboard and the cursor position. Tea never sends DA1, DSR 996,
    DECRQM 2031 or 2048, or an OSC 99 query.
  * **Not enabled.** Neither mode 2031 (colour-scheme reports) nor mode 2048
    (in-band resize) appears anywhere in tea's sources.
* **Every reply reaches the model.** The event loop handles its special
  cases and then always calls `model.Update(msg)`, except for quit,
  interrupt, suspend and batches. `translateInputEvent` maps the
  ultraviolet events it knows to tea types:
  `ModeReportMsg`, `KeyboardEnhancementsMsg`, `TerminalVersionMsg`,
  `CapabilityMsg`, `BackgroundColorMsg`, `FocusMsg`, `BlurMsg` and others.
  **Every other event is passed through as its ultraviolet type.** So
  `uv.DarkColorSchemeEvent`, `uv.LightColorSchemeEvent`,
  `uv.PrimaryDeviceAttributesEvent`, `uv.KittyGraphicsEvent`,
  `uv.PixelSizeEvent` and `uv.UnknownOscEvent` arrive in `Update` as
  themselves. Reading them needs a direct ultraviolet import.
* **What ultraviolet decodes.** DSR 997 becomes `DarkColorSchemeEvent` (`;1`)
  or `LightColorSchemeEvent` (`;2`). The in-band resize report `CSI 48;…t`
  becomes a `MultiEvent` of `WindowSizeEvent` and `PixelSizeEvent`, and the
  terminal reader flattens `MultiEvent`s. OSC 10, 11, 12 and 52 have their
  own events. Any other OSC reply, including OSC 99, is an
  `UnknownOscEvent` holding the raw bytes.
* **What x/ansi v0.11.8 provides.**
  * **Queries:** `RequestPrimaryDeviceAttributes`, `RequestNameVersion`,
    `RequestKittyKeyboard`, `RequestLightDarkReport` (DSR 996),
    `RequestModeLightDark` (2031), `RequestModeInBandResize` (2048),
    `RequestModeFocusEvent`, `DECRQM(m)`, `KittyGraphics(...)`,
    `XTGETTCAP(...)`.
  * **Modes:** `SetModeLightDark`, `ResetModeLightDark`,
    `SetModeInBandResize`, `ResetModeInBandResize`.
  * **Services:** `DesktopNotification(payload, metadata...)` (OSC 99),
    `Notify` (OSC 9), `URxvtExt` (OSC 777, the `notify` extension),
    `SetHyperlink` / `ResetHyperlink` (OSC 8), `SetSystemClipboard`
    (OSC 52), `FinalTerm`, `FinalTermPrompt`, `FinalTermCmdStart`,
    `FinalTermCmdExecuted` and `FinalTermCmdFinished` (OSC 133).
  * **Passthrough:** `TmuxPassthrough(seq)` (DCS `tmux;` with ESC doubled;
    tmux needs `allow-passthrough on`) and `ScreenPassthrough(seq, limit)`.
  * **Progress:** `SetProgressBar` and its variants (OSC 9;4). Tea already
    renders these from `View.ProgressBar`.
* **Links survive the compositor.** An ultraviolet `Cell` carries a `Link`,
  and `StyledString` reads OSC 8 into it. A hyperlink inside a workspace
  pane survives composition on the canvas.
* **Tea's commands for the rest.** `tea.Raw(seq)` writes a sequence through
  the program's output. `tea.SetClipboard` and `tea.ReadClipboard` write
  and read OSC 52. `tea.EnvMsg` carries the program's environment, which
  under wish is the SSH session's, not the server's. `tea.WithInput`,
  `WithOutput`, `WithEnvironment` and `WithWindowSize` let a test run a
  real program against scripted bytes.
* **Prior art** (0003-REPORT-agent-tui-ecosystem-research.md §3 and §5):
  * **crush** queries DA1, modifyOtherKeys, mode 1004 and OSC 99 always.
    It sends XTVERSION, the pixel size and a Kitty graphics query only to
    terminals that pass an environment heuristic, and wraps the Kitty query
    for tmux. It reads OSC 99 replies from `uv.UnknownOscEvent`.
    Notifications fall back from native to OSC 99, then OSC 777, then the
    bell, and are disabled without focus events. crush is FSL-1.1-MIT, so
    it is a design reference only.
  * **opencode** wraps OSC 52 for tmux and screen, and notifies only while
    unfocused.
  * **notcurses** queries the terminal at startup rather than trusting
    terminfo, and ships `notcurses-info`. **yazi** reports its chosen image
    driver with `ya env`. Both are the "doctor" pattern.
  * **Standards:** DA1 as the sentinel that ends a probe (the Kitty keyboard
    protocol's own detection advice); DSR 996/997 and mode 2031 (Contour's
    VT extension, also implemented by Ghostty, Kitty, tmux and VTE, per the
    report); OSC 99 `p=?` (Kitty desktop notifications).

## Decision Drivers

* **Never fight Bubble Tea.** Tea already queries, and acts on, modes 2026
  and 2027 and XTGETTCAP `RGB`. A second copy of those queries would change
  tea's renderer where tea chose not to, for example turning on synchronized
  output over SSH. The probe must observe tea's queries, not repeat them.
* **Query, then fall back.** Environment variables are the weakest evidence
  over SSH and in tmux. A reply is the strongest. Every fact records where
  it came from, so the doctor report can say why.
* **The probe always ends.** A terminal that ignores a query sends nothing,
  so "no reply" is only knowable when something after it replies. DA1 is
  answered by practically every terminal and multiplexer. A timeout covers
  the rest.
* **Per program, not per process.** Under wish there is one `tea.Program`
  per SSH session. Nothing is global. The environment comes from
  `tea.EnvMsg`, never `os.Getenv`.
* **The library writes nothing itself** (0001-MADR §6, rule 1). Every
  sequence leaves as a `tea.Cmd` or a string the caller places.
* **Leave the terminal as it was found.** A mode the library sets, such as
  2031, must have a reset the program can send before it exits.
* **Testable without a terminal.** Decoded messages, and a scripted fake
  terminal speaking bytes through a real `tea.Program`.
* **Extensible.** A later image record must add its own queries (Sixel
  geometry, iTerm2) without changing this package.

## Considered Options

* **A. `termcap`, an observer-prober with a DA1 sentinel and typed facts
  with provenance, plus `termsvc`, services built as commands from those
  facts.**
* **B. Environment heuristics only:** read `TERM`, `TERM_PROGRAM`, `TMUX`,
  `SSH_TTY` and friends, with no queries.
* **C. A complete query layer** that sends every query itself, tea's
  included, and owns all the replies.
* **D. Leave detection to each program.** Document the tea messages and the
  x/ansi helpers.

## Decision Outcome

Chosen option: **"A"**, because:

* it is the only option that learns the truth over SSH and in tmux without
  overriding the decisions tea already makes;
* it ends deterministically;
* it adds no module beyond the direct ultraviolet import that
  [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
  names.

### 1. Packages and their dependencies

```text
 termsvc    notifications, clipboard, hyperlinks, prompt marks, passthrough
                                            → termcap, bubbletea, x/ansi
 termcap    probe, typed Caps with provenance, CapsMsg, ColorSchemeMsg,
            Report (doctor)                 → bubbletea, x/ansi, colorprofile,
                                              internal/termevent
 termcap/termcaptest  scripted fake terminals for tests → bubbletea, stdlib
 internal/termevent   ultraviolet pass-through events → plain values
                                            → ultraviolet
```

* **ultraviolet stays contained.** 0004-MADR §1 confines ultraviolet to
  `internal/cells` with a `depguard` rule, keeps every ultraviolet type out
  of exported API, and lets a later record widen the rule. This record
  widens it by one package, `internal/termevent`. It holds one function:

  ```go
  // Decode turns an event tea passed through untranslated into a plain
  // value, or reports false.
  func Decode(msg any) (Event, bool)
  type Event struct {
      Kind  Kind   // ColorScheme, DeviceAttributes, KittyGraphics, PixelSize, Unknown
      Dark  bool   // ColorScheme
      Attrs []int  // DeviceAttributes
      W, H  int    // PixelSize
      Raw   string // Unknown: the sequence's bytes
  }
  ```

  It type-switches on `uv.DarkColorSchemeEvent`, `uv.LightColorSchemeEvent`,
  `uv.PrimaryDeviceAttributesEvent`, `uv.KittyGraphicsEvent`,
  `uv.PixelSizeEvent`, and the unknown-sequence events
  (`uv.UnknownCsiEvent`, `UnknownOscEvent`, `UnknownDcsEvent`,
  `UnknownApcEvent`), whose bytes a later query can parse. An ultraviolet change is
  then one file's fix, as 0004-MADR intends. This record adds no module.
* **No ultraviolet type in `termcap`'s API.** The environment is
  `termcap.Env`, a `[]string` of `KEY=value` with `Getenv` and `LookupEnv`,
  converted from `tea.EnvMsg`.
* Neither package imports `workspace`. A program feeds them messages beside
  its workspace. `keymap`
  ([0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md)) reads
  `termcap.Caps`, and `termcap` does not import it. `theme` and `workspace`
  follow `tea.BackgroundColorMsg` themselves (0004-MADR §2), and this probe
  only asks for it.

### 2. `termcap`: the facts

```go
// Support is what is known about one capability.
type Support uint8 // Unknown, Unsupported, Supported

// Origin is where a fact came from, strongest last.
type Origin uint8 // NotQueried, Heuristic, Env, Query, Override

// Fact is a value with its provenance, so a report can say why.
type Fact[T any] struct {
    Value  T
    Origin Origin
}

type Mux uint8 // NoMux, Tmux, Screen, Zellij

type Caps struct {
    Complete bool // the DA1 sentinel arrived
    TimedOut bool // the timeout fired first

    Terminal   Fact[string] // XTVERSION reply, else TERM_PROGRAM, else TERM
    Mux        Fact[Mux]    // TMUX / STY / ZELLIJ, or an XTVERSION of tmux
    Remote     Fact[bool]   // SSH_TTY or SSH_CONNECTION in tea.EnvMsg
    Attributes []int        // the DA1 reply

    KittyKeyboard Fact[Support]
    KeyboardFlags int // from KeyboardEnhancementsMsg

    SyncOutput         Fact[Support] // 2026, observed from tea's query
    GraphemeWidth      Fact[Support] // 2027, observed from tea's query
    ColorSchemeReports Fact[Support] // 2031
    InBandResize       Fact[Support] // 2048
    FocusEvents        Fact[Support] // 1004

    DesktopNotify Fact[Support] // OSC 99 p=? reply
    KittyGraphics Fact[Support] // APC G query, for the later image record
    Sixel         Fact[Support] // 4 in the DA1 reply

    Dark       Fact[bool]  // DSR 997, else OSC 11 luminance
    Background color.Color // OSC 11, when it replied
    Profile    colorprofile.Profile // from tea.ColorProfileMsg
}

type CapsMsg struct{ Caps Caps }         // once, when the probe ends
type ColorSchemeMsg struct{ Dark bool }  // each DSR 997, after the probe
```

* **`Fact` is generic.** The provenance rule is written once, and an
  override (`Origin` `Override`) always wins. A program can force a fact,
  from a flag or a config file.
* **`Caps` marshals to JSON** with `omitzero` tags, for a `--doctor --json`
  output and for bug reports.
* **Fields are added, never repurposed.** A new capability is a new field
  whose zero value is `Unknown`, so older code reading `Caps` keeps its
  meaning.

### 3. `termcap`: the probe

```go
type Prober struct{ /* caps, pending queries, options */ }

func New(o ...Option) *Prober
func (p *Prober) Init() tea.Cmd             // the batch, then the timeout
func (p *Prober) Update(msg tea.Msg) tea.Cmd // observes; never consumes
func (p *Prober) Caps() Caps
func (p *Prober) Restore() string           // resets every mode it set

// Query is one probe. Programs and later records add their own.
type Query struct {
    Name  string
    Seq   func(env Env, mux Mux) string    // "" skips it
    Gated bool                            // only behind the heuristic
    Parse func(r Reply, c *Caps) bool     // true when r answered it
}

// Reply is a message as a query sees it: the tea message, and for any
// sequence ultraviolet did not decode, its raw bytes.
type Reply struct {
    Msg tea.Msg
    Raw string // an unknown CSI, OSC, DCS or APC reply; "" otherwise
}

// Options: WithTimeout(d) (default 2 s), WithQuery(Query),
// WithoutHeuristic(), WithOverride(func(*Caps)),
// WithColorSchemeUpdates() (subscribe to 2031 when supported),
// WithoutBackgroundRequest(), WithDisabled().
```

* **Embedding.** The program keeps a `*Prober` beside its workspace,
  returns `p.Init()` from `Init`, and passes every message to
  `p.Update(msg)`, as it does for `Workspace.Update`. The prober never
  hides a message from the program.
* **The batch** is one `tea.Raw` write, in this order:
  1. `RequestKittyKeyboard`. Tea's own request goes out on the first render,
     which may come after the sentinel, so the prober asks again. Tea
     ignores the extra `KeyboardEnhancementsMsg`.
  2. DECRQM for 2031, 2048 and 1004, and DSR 996.
  3. XTVERSION, the OSC 99 `p=?` query and the Kitty graphics query, **only
     behind the heuristic** (the crush and tea pattern: skip Apple Terminal
     and unknown SSH peers, because some terminals print queries they do
     not understand). Inside tmux, the OSC 99 and Kitty graphics queries
     are wrapped with `TmuxPassthrough`, because only the outer terminal can
     answer them.
  4. Each `WithQuery` sequence.
  5. **DA1 last**, as the sentinel.

  `tea.RequestBackgroundColor` runs in the same batch. Tea does not act on
  the reply. A workspace that follows the theme rebuilds it from that reply
  (0004-MADR §2), which leaves asking to the program: embedding the prober
  is that choice, and `WithoutBackgroundRequest()` declines it.
* **What it never sends.** DECRQM 2026 and 2027, and XTGETTCAP `RGB` or
  `Tc`, because tea acts on those replies. Their facts come from watching
  tea's own `ModeReportMsg` and `CapabilityMsg`. When tea skipped the query
  (its SSH gate), the fact stays `Unknown` with origin `NotQueried`.
* **How it ends.** Terminals answer in order. When the DA1 reply arrives,
  every query sent before it that has no reply becomes `Unsupported`, with
  origin `Query`. If the timeout fires first, the prober marks
  `TimedOut`, keeps what it has, and leaves the rest `Unknown`. Either way,
  it returns a command that delivers `CapsMsg` once.
* **After it ends.** It keeps observing:
  * DSR 997 becomes `ColorSchemeMsg`, followed by a fresh
    `tea.RequestBackgroundColor`, so a workspace following the theme gets
    the exact colour;
  * `ColorProfileMsg` updates `Profile`.
* **Live light and dark.** `WithColorSchemeUpdates` writes
  `SetModeLightDark` once 2031 is `Supported`. `Restore()` then returns
  `ResetModeLightDark`. Tea does not know about this mode. The program sends
  `tea.Sequence(tea.Raw(p.Restore()), tea.Quit)` to quit, or writes
  `Restore()` itself after `Run` returns. Without the reset, the shell would
  receive DSR 997 reports after the program exits.
* **In-band resize (2048)** is detected and reported, not enabled. Tea
  resizes from SIGWINCH, and enabling 2048 would need the same reset
  discipline for a gain this record does not need (owner question Q3).
* **No input, no probe.** A program run with `tea.WithInput(nil)` must not
  call `Init`. Tea skips its own queries in that case, but `tea.Raw` does
  not, and the replies would leak into the shell. The guide says so, and
  `New` takes `WithDisabled()` for programs that decide at run time.

### 4. `termcap`: the doctor

```go
func Report(w io.Writer, c Caps, o ...ReportOption) error
```

* One line per fact: the name, the value, and the origin ("query", "env",
  "not queried: tea skips 2026 over SSH", "override").
* It writes to the `io.Writer` it is given (rule 1). A program's
  `doctor` command, or the command registry's `terminal.doctor`
  ([0006-MADR-command-registry.md](0006-MADR-command-registry.md)), calls
  it. ASCII only, so it can be pasted into an issue.

### 5. `termsvc`: services as commands

```go
// Notifications.
type Notification struct {
    Title, Body string
    ID          string  // groups updates to one notification (OSC 99)
    Urgency     Urgency // Low, Normal, Critical
}
type Protocol uint8 // Auto, OSC99, OSC777, OSC9, Bell, Off
type Policy uint8   // WhenUnfocused (default), Always, Never

type Notifier struct{ /* caps, focus, protocol, policy, hook */ }
func NewNotifier(o ...NotifyOption) *Notifier
func (n *Notifier) Update(msg tea.Msg) tea.Cmd // CapsMsg, FocusMsg, BlurMsg
func (n *Notifier) Notify(x Notification) tea.Cmd // nil when the policy says no

// A program-supplied native notifier (for example a desktop API). The
// library never starts a process itself.
type Backend interface {
    Notify(ctx context.Context, x Notification) error
}

// Clipboard, links, marks, passthrough.
func Copy(c termcap.Caps, text string) tea.Cmd
func Link(url, text string, params ...string) (string, error)
func PromptStart() string; func CommandStart() string
func CommandExecuted() string; func CommandFinished(exit int) string
func Wrap(c termcap.Caps, seq string) string // tmux or screen passthrough
```

* **Notifications.** `Auto` picks OSC 99 when `DesktopNotify` is
  `Supported`. Otherwise it picks OSC 777 on terminals the heuristic knows
  to support it, then OSC 9, then the bell. Inside tmux the sequence is
  wrapped with `Wrap`.
  * Under `WhenUnfocused`, nothing is sent while the terminal has focus.
    Focus comes from `FocusMsg` and `BlurMsg`, which need the program's
    `View.ReportFocus`. Without focus reports, `WhenUnfocused` behaves as
    `Never` unless the program chooses `Always`, which is crush's choice.
  * Title and body are stripped of C0, C1 and ESC bytes before encoding.
    An agent's text must not smuggle a sequence into the terminal. This is
    a small unexported function here. The `safetext` package of
    [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
    may later absorb it.
* **Clipboard.** `Copy` returns `tea.SetClipboard(text)`. Inside tmux it
  also sends a passthrough-wrapped OSC 52, as opencode does, for tmux
  configurations without `set-clipboard`. OSC 52 never replies, so a copy
  cannot be confirmed. A native clipboard is a program-supplied hook, not a
  process the library starts.
* **Links.** `Link` refuses schemes other than `http`, `https`, `file` and
  `mailto`, strips control bytes from both parts, and returns
  `SetHyperlink` … `ResetHyperlink`. Lip Gloss's `Style.Hyperlink` remains
  the styling path. `Link` is the validating one.
* **Prompt marks (OSC 133)** are strings for inline output, where they let
  the terminal jump between commands. On the alternate screen they mean
  nothing, and the documentation says so.
* **Progress (OSC 9;4)** is already `tea.View.ProgressBar`. `termsvc` adds no
  wrapper.

### 6. Versioning

The two packages land after 0004-MADR's `v0.2.0`, in the next minor release.
The tag is the owner's. Image protocols (Kitty placeholders, Sixel, iTerm2)
are a later record that adds `Query` values and a pane, and needs no change
here.

### Consequences

* Good, because every consumer learns the same facts the same way, over SSH
  and in tmux, and can show the user why in one report.
* Good, because tea's own decisions stand. Nothing here re-asks what tea
  asked.
* Good, because the probe always ends, by DA1 or by timeout, and the result
  arrives as one message.
* Good, because a later record extends detection through `Query` and `Caps`
  fields, without an incompatible change.
* Good, because notifications, clipboard and links carry the sanitising
  and passthrough rules that each agent TUI otherwise rewrites.
* Neutral, because live light and dark needs the program to send
  `Restore()`. The guide and the package documentation carry that rule.
* Bad, because the heuristic gate is a list that ages. It is one function,
  overridable with `WithoutHeuristic`, and every gated fact is reported as
  `NotQueried` rather than `Unsupported`.
* Bad, because `termcap` depends on ultraviolet's event types, which are
  not covered by a semver release. They are read in one internal file,
  `internal/termevent`, under 0004-MADR's containment rule, and tea pins the
  version.
* Bad, because `depguard`'s ultraviolet rule widens from one internal
  package to two.
* Bad, because OSC 777 and OSC 9 have no query. Choosing them relies on the
  heuristic, which can be wrong.

### Confirmation

* **`internal/termevent`** has a table test per ultraviolet event type, and
  the `depguard` rule reports ultraviolet planted in `termcap`.
* **No ultraviolet type is exported.** A `go/types` test walks `termcap`'s
  and `termsvc`'s exported API and fails on any type from ultraviolet.
* **Unit tests** feed decoded messages (`tea` and `uv` types) into
  `Prober.Update`:
  * a full reply set yields `Supported` facts with origin `Query`;
  * DA1 alone marks every earlier query `Unsupported`;
  * the timeout yields `TimedOut` and `Unknown` facts;
  * tea's `ModeReportMsg` for 2026 and 2027 fills `SyncOutput` and
    `GraphemeWidth`;
  * the batch never contains DECRQM 2026 or 2027, or XTGETTCAP;
  * `CapsMsg` is delivered exactly once;
  * DSR 997 after the probe yields `ColorSchemeMsg` and a background
    request;
  * an override beats a query.
* **Fake-terminal tests** (`termcaptest`) run a real `tea.Program` with
  `WithInput`, `WithOutput` and `WithEnvironment`. A scripted terminal
  reads the program's output and answers in bytes, as each profile would:
  a Kitty-like terminal, an xterm-like one, tmux with and without
  passthrough, a terminal that answers only DA1, and one that answers
  nothing.
* **Golden output** of `Report` for each profile. The report is ASCII
  text with no colour, so the matrix reduces to its widths.
* **termsvc:** byte-exact sequences for each protocol; passthrough
  wrapping inside tmux; no notification while focused under
  `WhenUnfocused`; control bytes stripped from titles, bodies and links; a
  `javascript:` link refused.
* **Mutation proofs,** each seen failing on a scratch copy:
  * `termcap` imports ultraviolet directly;
  * the batch includes DECRQM 2026;
  * DA1 is sent first, not last;
  * a missing reply stays `Unknown` after the sentinel;
  * `CapsMsg` is sent twice;
  * `Restore` omits `ResetModeLightDark`;
  * notifications ignore focus;
  * the control-byte strip is skipped.
* **Conformance:** `internal/conformance` (as hardened by
  [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md))
  covers rules 1 and 2 in both packages. A source test in each package also
  fails on any use of `os.Getenv`, `os.LookupEnv`, `os.Environ` or
  `os/exec`, because the environment must come from `tea.EnvMsg` and no
  process is started.

## Pros and Cons of the Options

### A. Observer-prober plus services

* Good, because it combines replies, tea's own observations and the
  environment, and labels each fact.
* Good, because the `Query` hook keeps the package open.
* Bad, because it needs ultraviolet's event types to see the pass-through
  events, in a second internal package.
* Bad, because a fake terminal for tests is real work.

### B. Environment heuristics only

* Good, because it is synchronous and needs no messages.
* Bad, because over SSH `TERM_PROGRAM` is usually missing, and in tmux `TERM`
  names tmux, not the terminal. Both cases matter to pi-go.
* Bad, because it cannot see live light and dark, or OSC 99.

### C. A complete query layer

* Good, because one component would own every query.
* Bad, because re-sending DECRQM 2026, 2027 or XTGETTCAP `RGB` changes tea's
  renderer, for example enabling synchronized output over SSH, which tea
  deliberately avoids.
* Bad, because it duplicates logic tea maintains and will change.

### D. Leave it to each program

* Good, because there is nothing to build.
* Bad, because the owner asked for a large open-standards surface, and
  every consumer would rediscover the sentinel, the SSH gate and the
  passthrough rules.

## Owner questions

* **Q1. Gated queries.** Should XTVERSION, OSC 99 and the Kitty graphics
  query be sent only behind the environment heuristic? Recommended: yes,
  as crush and tea do. `WithoutHeuristic` sends them always. The safe
  queries (DECRQM, DSR 996, the Kitty keyboard request, DA1) are always
  sent.
* **Q2. Live light and dark.** Should mode 2031 be subscribed by default
  when supported? Recommended: no, opt in with `WithColorSchemeUpdates`,
  because it needs the program to send `Restore()` before exiting. The
  initial DSR 996 query, which sets nothing, is always sent.
* **Q3. In-band resize (2048).** Should `termcap` offer to enable it?
  Recommended: detect and report only in this record. Enabling it needs the
  same reset discipline, and tea's SIGWINCH path already works.
* **Q4. Native fallbacks.** Should the library start `pbcopy`, `wl-copy`,
  `notify-send` or similar? Recommended: no. Native clipboard and
  notifications are hooks the program supplies. Over SSH a native call
  would act on the remote host.

## More Information

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §3 (crush, opencode, codex), §4 (notcurses, yazi) and §5 (the standards
  table).
* [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md):
  the ultraviolet import and its containment rule, which this record widens
  to `internal/termevent`, and the workspace's theme following
  `tea.BackgroundColorMsg`, the reply this probe requests.
* [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md): default bindings
  chosen from `Caps.KittyKeyboard`.
* [0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)
  §6: the conventions both packages follow.
* Standards: the Kitty keyboard protocol and desktop notifications
  (sw.kovidgoyal.net), Contour's colour-palette update notifications
  (DSR 996/997, mode 2031), the in-band resize gist (mode 2048), and the
  OSC 8 hyperlink specification. Each is linked in the report.
* **Not verified here:**
  * that tmux delivers replies to passthrough queries back to the pane (crush
    relies on it for its Kitty graphics query);
  * which terminals implement OSC 777 `notify`;
  * terminals' OSC 52 size limits.

  PLAN Step 1 records what the fake-terminal spike shows. The first two are
  checked on real terminals in Verification.
