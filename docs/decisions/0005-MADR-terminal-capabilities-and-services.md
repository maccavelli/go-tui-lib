---
status: accepted
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

// Quit restores every mode the prober set, then quits:
// tea.Sequence(tea.Raw(p.Restore()), tea.Quit).
func (p *Prober) Quit() tea.Cmd

// Options: WithTimeout(d) (default 2 s), WithQuery(Query),
// WithoutHeuristic(), WithOverride(func(*Caps)),
// WithoutColorSchemeUpdates() (never subscribe to 2031),
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
* **Live light and dark, on by default** (owner question Q2). The prober
  writes `SetModeLightDark` once 2031 is `Supported`.
  `WithoutColorSchemeUpdates` declines it. `Restore()` then returns
  `ResetModeLightDark`. Tea does not know about this mode.
  * **Every program that embeds a `Prober` must restore before it exits.**
    It quits with `p.Quit()`, or sends
    `tea.Sequence(tea.Raw(p.Restore()), tea.Quit)` itself, or writes
    `Restore()` after `Run` returns. Without the reset, the shell receives
    DSR 997 reports after the program exits.
  * `Restore()` is empty when 2031 was never set, so calling it is always
    safe.
  * The restore bytes of the planned `termmode` record (0003-REPORT §8.4,
    §11.2) will include `ResetModeLightDark`, so a program that uses them
    on a crash path is also covered.
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
* Bad, because live light and dark is on by default, so every program that
  embeds a `Prober` must send `Restore()` before it exits, and one that
  forgets leaves DSR 997 reports to the shell. `Quit()`, the guide, the
  package documentation and the fake-terminal test that checks the reset
  is written before exit all carry that rule.
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
  * by default 2031 is set when `Supported` and not otherwise;
    `WithoutColorSchemeUpdates` never sets it;
  * `Quit()` writes `Restore()` before `tea.Quit`;
  * an override beats a query.
* **Fake-terminal tests** (`termcaptest`) run a real `tea.Program` with
  `WithInput`, `WithOutput` and `WithEnvironment`. A scripted terminal
  reads the program's output and answers in bytes, as each profile would:
  a Kitty-like terminal, an xterm-like one, tmux with and without
  passthrough, a terminal that answers only DA1, and one that answers
  nothing. For a profile that supports 2031, the fake terminal sees
  `ResetModeLightDark` before the program exits through `Quit()`.
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
  * the default does not subscribe to 2031;
  * `Quit()` sends `tea.Quit` before the restore;
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

*Answered 2026-10-02* (picked from options): Q1 "Gated"; Q2 "On by
default"; Q3 "Detect and report only". Q4 was answered by the second-pass
question "May the library spawn processes?" with "Return commands only":
the library returns an argv or a hook, and the program runs it. Q1, Q3
and Q4 are the recommendation. **Q2 departs from it.** §3 ("Live light and
dark", the options and `Quit`), Consequences and Confirmation changed to
match, and
[0005-PLAN-terminal-capabilities-and-services.md](0005-PLAN-terminal-capabilities-and-services.md)
Steps 3, 4 and 7 with them.

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

## Amendments

### A1 (2026-10-02): second-pass findings

*Status: accepted (2026-10-02).* Its steps are A1.1 to A1.3 in
[0005-PLAN-terminal-capabilities-and-services.md](0005-PLAN-terminal-capabilities-and-services.md).

**Found.** A source-level pass over the Kilo, Grok Build, opencode and codex
TUIs
([0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§8) found detection and service rules this record does not yet have. §11.1
of the report lists them against this record. Three owner answers given
with that pass bound what this record takes:

* the library returns commands and never starts a process (the same
  answer as Q4);
* terminal mode plans, teardown, hand-off and the Windows console mode
  helpers belong to a later `termmode` record, and inline scrollback to a
  later `inline` record;
* a direct `golang.org/x/sys` import may be proposed, in its own
  dependency record.

**What changes in the decision.**

* **§2, identity is a pure function of the environment** (report §8.2).
  * `termcap.FromEnv(env Env, goos string) Identity` builds the identity
    the prober starts from. `goos` is the caller's `runtime.GOOS`, and
    `env` still comes from `tea.EnvMsg`.
  * `Identity` holds a `Brand`, the raw `EnvBrand`, a version kept only
    when the brand corroborates it, the `Mux`, an embedded `Editor`
    (`NVIM`, `VIM_TERMINAL`, `INSIDE_EMACS`), `Remote`, `TERM` and
    `TERM_FEATURES`.
  * The detection order is the report's: editor-fork markers that survive
    SSH and tmux, then `TERM_PROGRAM`, then `TERMINAL_EMULATOR` (JetBrains)
    before `TERM_SESSION_ID`, then `LC_TERMINAL`, then `TERM`, then
    `TERMINATOR_UUID` before `VTE_VERSION`, and `WT_SESSION` last.
  * **Raw and refined brands stay apart.** On Windows an unknown brand is
    refined to Windows Terminal, because its default-terminal hand-off
    omits `WT_SESSION`. A decision that must not trust the guess, such as
    the legacy-console glyph tier, reads `EnvBrand`.
  * `Caps` gains `Brand`, `EnvBrand`, `Editor`, `SecondaryAttributes`
    (DA2), `LegacyConsole` and `Tmux` as new fields. Existing fields keep
    their meaning (§2, "fields are added, never repurposed").
* **§2, every fact can carry a reason** (report §8.2, §8.16).
  * `Fact[T]` gains `Reason string`: a stable dotted token, such as
    `tmux.extended-keys-off` or `terminal.jetbrains-paints-queries`, that
    says why a fact is `Unsupported`, `Unknown` or gated.
  * Tokens are API. A token is added, never renamed, and the doctor's
    finding IDs (§4 below) are the same strings.
  * Per-feature views read from `Caps` rather than the brand:
    `Keyboard() KeyboardCaps`, `Links() LinkCaps` and
    `Notifications() NotifyCaps`.
* **§2, the kitty flag policy** (report §8.5, §9).
  `KeyboardFlags(c Caps) KittyFlags` returns the flags a program should put
  in `View.KeyboardEnhancements`:
  * disambiguate when the terminal supports the protocol;
  * no event types (press, repeat, release) on iTerm2 and Ghostty, which
    leak releases for shortcuts they consume, on Alacritty 0.14 or older
    (from DA2), and in tmux unless tmux reports
    `extended-keys-format csi-u`;
  * no flags at all on mintty and MSYS2 (`TERM_PROGRAM=mintty`, any
    `MSYSTEM`), and under WSL when VS Code is detected or detection is
    inconclusive.
  * `Caps.KeyboardFlags`, from `KeyboardEnhancementsMsg`, is the ledger of
    what the terminal accepted. `ReleasesReported()` says whether a binding
    may wait for a key release, which
    [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) reads.
  * A program's environment overrides go through `WithOverride`, as for
    any fact.
* **§2, appearance has a chain** (report §8.2). `Dark` takes the strongest
  of:
  1. an override;
  2. DSR 997, then OSC 11 luminance (`Origin` `Query`);
  3. an environment variable the program names with
     `WithAppearanceEnv(name)`, which also reads `LC_` + name, because a
     default `sshd` forwards `LC_*` (`Origin` `Env`);
  4. a program-supplied desktop hook (macOS appearance, the XDG portal,
     the Windows registry), because reading those needs a process or an OS
     binding (`Origin` `Heuristic`);
  5. `COLORFGBG`, read with Vim's heuristic, where `default` means unknown
     (`Origin` `Heuristic`).
* **§2, palette facts for a generated theme** (report §8.2). `Caps` gains
  `Foreground` (OSC 10) and `Palette` (OSC 4 for indexes 0–15) with a
  `PaletteKnown` flag. They are parsed from `Reply.Raw`, because the
  decoder has no OSC 4 event. A later theme record builds a theme from
  them. Owner question Q5.
* **§2, tmux is asked through a returned command** (report §8.3).
  `TmuxQuery() []string` returns the argv of one
  `tmux display-message -p` for `extended-keys-format`, `mouse`,
  `client_termfeatures` and `client_flags`. `ParseTmux(out string)
  TmuxFacts` reads its output. The program runs the command and passes the
  result with `p.SetTmux(TmuxFacts)`. Nothing here starts it.
* **§2, the legacy console** (report §8.11, §9). `LegacyConsole` is a
  heuristic fact until a Windows record can ask the console host:
  `goos` is `windows` and `EnvBrand` is unknown (no `WT_SESSION`,
  `TERM_PROGRAM` or `WEZTERM_PANE`), because bare `cmd.exe` sets no
  terminal variables. A hook, `WithConsoleHost(func() (classic bool, ok
  bool))`, lets that later record supply the `ConsoleWindowClass` check,
  which tells classic conhost from ConPTY.
* **§3, probe discipline** (report §8.3).
  * **DA2 joins the safe set,** before DA1. Its reply gives the Alacritty
    version, and with DA1 `1;2` the Apple Terminal fingerprint
    (`1;95;0`) that identifies Terminal.app over SSH.
  * **One deadline still covers the batch** (`WithTimeout`). A reply that
    arrives after the sentinel or the timeout still updates its fact, as
    §3 already says.
  * **Reply caps.** A raw reply longer than 1 KiB is not parsed, and its
    fact gets the reason `probe.reply-too-long`.
  * **Late replies are named, not dropped.** `IsReplyFragment(msg tea.Msg)
    bool` reports a key message that matches the start of a reply the
    probe is still waiting for. The input filter of
    [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
    uses it to swallow fragments. The prober itself still never hides a
    message.
  * **Gates.** Under JetBrains, which paints queries as text, the prober
    sends nothing, and every fact comes from the environment with the
    reason `terminal.jetbrains-paints-queries` (owner question Q7). Inside
    an editor's `:terminal`, the gated set is skipped.
  * **The sole-reader rule.** `termcap` has no synchronous probe. Any
    later query that must read stdin directly runs only before
    `tea.Program` starts reading, and lives in `termmode`, not here.
* **§4, the doctor reports findings** (report §8.16).
  `Findings(c Caps) []Finding` returns
  `Finding{ID, Disposition (Issue, Recommendation), Message, Fix string}`,
  where `ID` is a reason token and `Fix` is text, never an action.
  `Report` prints them after the facts. The JSON form carries a
  `schema_version` that changes only when a field is removed or retyped.
* **§5, clipboard delivery is reported** (report §8.13).
  * `Copy` delivers `CopiedMsg{Status, Route}`, where `Status` is
    `Unconfirmed` for any terminal route (OSC 52 never replies),
    `Confirmed` or `Failed` from a program-supplied backend, and `Failed`
    for a payload over 100 KB.
  * `CopyPlan(c Caps) []Route` orders the routes: the backend, the tmux
    buffer, OSC 52, and OSC 52 wrapped for tmux.
  * `TmuxLoadBuffer() []string` returns the argv of
    `tmux load-buffer -w -`, which the program runs with the text on stdin.
  * `ImageReadCommands(goos string, wayland bool) [][]string` returns the
    clipboard-image readers in order: `osascript` on macOS, PowerShell on
    Windows and WSL, then `wl-paste` and `xclip`. It is a pure function.
* **§5, links follow the terminal** (report §8.13).
  * `LinkDisplay(c Caps) Display` is `LabelOnly` on the terminals the
    report lists and `LabelAndURL` on Apple Terminal, Warp, unknown
    terminals, any multiplexer, or tmux older than 3.4.
  * `LinkPolicy{Schemes []string}` and `Openable(url) bool` decide whether
    a click may open a link. The default is `http` and `https`. The
    program opens it, from an `OpenURLMsg`.
* **§5, notifications** (report §8.13, §8.12).
  * When OSC 99 did not answer, `Auto` picks by brand: OSC 9 for iTerm2,
    WezTerm and Warp; OSC 777 for Ghostty, VTE and foot; the bell for
    Zellij and the rest.
  * Focus is tri-state. `Notify` delivers
    `NotifyResultMsg{Sent bool; Skipped SkipReason}`, where the reasons are
    disabled, empty, focused and focus-unknown. Owner question Q6.
  * Text is cleaned before encoding: escape sequences stripped, newline
    runs collapsed to a space, controls removed, and cut by cells to 80 for
    the title and 240 for the body.
  * `WithGate(func(Notification) bool)` lets a caller apply a turn-outcome
    policy, such as Kilo's "completed root turns only". The policy itself
    belongs to the planned agent widgets, not here.
* **§5, more services, each a string or a command** (report §8.13):
  * `SanitizeTitle(s string) string` removes controls and bidi codepoints
    and caps the title at 240 characters, for `View.WindowTitle`. A
    program that is not on a terminal sets no title.
  * `Activity(vendor string, s ActivityState, t time.Time) string` writes
    `OSC 777;<vendor>;activity;1;<state>;<unix-ms>`, Kilo's versioned
    beacon. `ParseActivity(payload, now)` reads one, rejecting a wrong
    version, a timestamp more than 5 s ahead or 15 s old. `ActivityBeacon`
    is a 5 s ticker. It is opt-in.
  * `Pointer(c Caps, shape string) string` writes OSC 22 only on Ghostty
    and Kitty outside a multiplexer, and an empty OSC 22 resets it on
    Kitty.
  * `ProgressSupported(c Caps) bool` is true on Ghostty, WezTerm and
    iTerm2 3.6 or later, for a program deciding whether to set
    `View.ProgressBar`. Older iTerm2 shows OSC 9;4 as alert text.

**What does not change.**

* Option A, the two packages and `termcaptest`.
* The observer rule: nothing re-sends what tea asks.
* DA1 as the sentinel, and the timeout.
* No process is started. Every command above is returned for the program
  to run, and the source test against `os/exec` stays.
* The environment comes from `tea.EnvMsg`.
* ultraviolet stays in `internal/termevent`.
* No module is added. A direct `golang.org/x/sys` import, for the console
  host check, belongs to a later record.

**Left to other records.** Mode plans, the teardown order, restore bytes,
the DA1 pop fence, hand-off to external programs, the tmux resize sampler
and the Windows console mode helpers go to `termmode`. The per-terminal
scrollback strategy and the width-shrink model go to `inline`. Wheel
profiles go to 0009's input filter, which takes the events per notch as an
option (`WithWheelProfile`) and does not import `termcap`. The program
derives that count from `Caps.Brand` and `Caps.Mux`.
The legacy glyph tier goes to the theme and glyph record, which reads
`Caps.LegacyConsole`.

**Versioning.** Everything A1 adds is new API. If A1 is accepted before
the PLAN's Step 2 starts, it ships in the same minor release as the rest of
this record; otherwise in the next minor.

**Owner questions for A1.**

*Answered 2026-10-02* (picked from options): Q5 "Yes, behind the
heuristic"; Q6 "Don't send; add UnlessFocused"; Q7 "Send nothing".
Every answer is the recommendation, so the text above stands, and A1 is
accepted.

* **Q5. The palette query.** Should the OSC 4 and OSC 10 queries go out by
  default? Recommended: yes, behind the heuristic, as XTVERSION and OSC 99
  do, which matches the owner's choice of a background query in `Init`
  (0004-MADR Q2). The alternative is opt-in with `WithPalette()`, because
  it is 17 queries the program did not ask for.
* **Q6. Notifications without focus reports.** codex turns focus reporting
  off on Windows, so focus stays unknown there and `WhenUnfocused` never
  sends. Recommended: keep "unknown means do not send" as the default, and
  add a policy, `UnlessFocused`, that sends when blurred or unknown, for
  programs that accept the risk. The alternative is that `WhenUnfocused`
  sends when focus is unknown.
* **Q7. JetBrains.** Recommended: send nothing under JetBrains, as Grok
  does, and take every fact from the environment. The alternative is to
  send the safe set anyway and accept that the queries may be painted.
  Not verified here; the PLAN's real-terminal check covers it when a
  JetBrains terminal is available.

## More Information

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §3 (crush, opencode, codex), §4 (notcurses, yazi) and §5 (the standards
  table); for amendment A1, §8.2–§8.5, §8.13, §8.16, §9 and §11.1.
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
