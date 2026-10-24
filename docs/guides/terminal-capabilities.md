# Terminal capabilities and services

For a program that wants to know what its terminal can do, over SSH and
inside tmux as well as locally, and to send notifications, clipboard
writes and links that suit it: a Bubble Tea v2 program, or a line-mode
CLI. These packages do the work:

- `termcap` learns the terminal's capabilities from one batch of queries
  that ends with DA1, and reports each fact with where it came from. Its
  `Probe` does no I/O, and `termcap` links no Charm module but `x/ansi`,
  so a line-mode CLI drives it itself.
- `termcap/termcaptea` is the `Probe`'s host in a Bubble Tea program: its
  `Prober` sits beside your model.
- `termsvc` turns those facts into strings and decisions: notifications,
  the clipboard, links, prompt marks, titles and the activity beacon. It
  links no Charm module but `x/ansi` either.
- `termsvc/termsvctea` turns the same services into Bubble Tea commands.
- `termcap/termcaptest` runs your program against scripted fake terminals,
  so all of this can be tested without a real one.

Why they are built this way is in
[0005-MADR](../decisions/0005-MADR-terminal-capabilities-and-services.md),
with its amendments A1 to A5; the split into a core and its Bubble Tea
host is
[0020-MADR](../decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
D4 and A2. The examples `ExampleProbe` and `ExampleReport` compile with
`termcap`, and `ExampleProber` with `termcaptea`
(`go doc -all github.com/maccavelli/go-tui-lib/v2/termcap`).

## Embed a prober

A `termcaptea.Prober` is not a `tea.Model`. Your model keeps one, returns
its `Init` from your `Init`, passes it every message, and quits through
it:

```go
type model struct {
    probe *termcaptea.Prober
    ws    *workspace.Workspace
    caps  termcap.Caps
}

func newModel() *model {
    return &model{
        probe: termcaptea.NewProber(),
        // The prober asks for the background; the workspace must not ask too.
        ws: workspace.New(root, panes, workspace.WithoutBackgroundQuery()),
    }
}

func (m *model) Init() tea.Cmd { return tea.Batch(m.probe.Init(), m.ws.Init()) }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    cmds := []tea.Cmd{m.probe.Update(msg), m.ws.Update(msg)}
    switch msg := msg.(type) {
    case termcap.CapsMsg:
        m.caps = msg.Caps
    case tea.KeyPressMsg:
        if msg.String() == "ctrl+c" {
            return m, tea.Batch(append(cmds, m.probe.Quit())...)
        }
    }
    return m, tea.Batch(cmds...)
}
```

- **`Init` starts the deadline only.** The batch goes out on the first
  `tea.EnvMsg`, because it depends on the environment, and tea sends that
  message unordered with `Init`'s command (MADR A3).
- **The prober never hides a message.** Every reply still reaches your
  model and the workspace.
- **It never re-asks what tea asks.** Tea queries modes 2026 and 2027 and
  XTGETTCAP itself, and acts on the answers; the prober builds its probe
  with `termcap.WithoutModeQuery(2026, 2027)`, and reads tea's messages
  instead.
- **One background query.** The prober writes the OSC 11 query in its
  batch, before DA1. A workspace beside it asks too unless it is built with
  `workspace.WithoutBackgroundQuery()`; with both, the query goes out
  twice, and JetBrains terminals paint the workspace's copy (MADR A2, Q8).

## Quit through the prober

By default the prober subscribes to mode 2031 once the terminal supports
it, so the terminal reports light and dark changes as they happen. That
mode outlives the program, so **every program must reset it before it
exits**:

- quit with `m.probe.Quit()`, which writes the reset and then quits;
- or send `tea.Sequence(tea.Raw(m.probe.Restore()), tea.Quit)` yourself;
- or write `probe.Restore()` after `Run` returns.

`Restore()` is empty when nothing was set, so calling it is always safe. A
program that cannot do any of these passes `termcap.WithoutColorSchemeUpdates()`.
Without the reset, the shell receives DSR 997 reports after the program
exits.

## What arrives

- **`termcap.CapsMsg`, once,** when DA1 answers, or when the deadline
  (`WithTimeout`, 2 s by default) fires first. DA1's answer waits for
  tea's colour profile; the deadline does not, so a message the deadline
  sends may come before it. `Caps.Complete` and `Caps.TimedOut` say
  which.
- **`termcap.ColorSchemeMsg{Dark}`** each time the terminal reports a
  change after the probe, followed by a fresh background query, so a
  workspace that follows the theme gets the exact colour.
- **`Prober.Caps()`** at any time: a reply after DA1 still updates its fact.

Each field of `Caps` is a `Fact`: a value, its `Origin` (`not-queried`,
`heuristic`, `env`, `query`, `override`, weakest first), and a `Reason`
token when it says no. A stronger origin replaces a weaker one, and an
override is never replaced. Read the views rather than the brand:

| View | Says |
| :--- | :--- |
| `termcaptea.KeyboardEnhancements(caps)` | what to put in `View.KeyboardEnhancements`: event types only where the terminal reports them cleanly |
| `caps.Keyboard()` | the Kitty fact, the flags to ask for (`Request`: 2, event types, or 0), the flags the terminal accepted, and whether a binding may wait for a release |
| `caps.Links()` | whether OSC 8 links work |
| `caps.Notifications()` | OSC 99, OSC 777, OSC 9 and focus reports |
| `caps.ReleasesReported()` | whether key releases arrive |

Tea always asks for Kitty disambiguation, so on mintty, MSYS2 and WSL in
VS Code, where none is wanted, `Keyboard()` says why and returns no extra
features; turning the protocol off belongs to a later record (MADR A4).

## Options

| Option | Does |
| :--- | :--- |
| `WithTimeout(d)` | waits `d` for DA1; 2 s by default |
| `WithQuery(q)` | adds a query of your own, after the built-in ones and before DA1 |
| `WithoutHeuristic()` | sends the gated queries (XTVERSION, OSC 99, Kitty graphics, the foreground (OSC 10), the palette) to every terminal, Apple Terminal and unknown SSH peers included |
| `WithOverride(f)` | sets facts you know better; set them with origin `Override` |
| `WithoutColorSchemeUpdates()` | never subscribes to mode 2031 |
| `WithoutBackgroundRequest()` | never asks for the background |
| `WithDisabled()` | sends nothing; the probe ends at once with the environment's facts, which a `Prober` delivers on the first `tea.EnvMsg` |
| `WithAppearanceEnv(name)` | reads light or dark from `LC_` + `name`, which a default sshd forwards, and from `name`, which wins when both are set |
| `WithAppearanceHook(f)` | asks the desktop, through your function |
| `WithConsoleHost(f)` | asks Windows whether this is the classic console |
| `WithGOOS(goos)` | reads the identity for another OS: a test, or a wish server |
| `WithoutModeQuery(modes...)` | leaves those DECRQM queries out of the batch: the host asks for them, and passes their reports in |

Under JetBrains the prober sends nothing at all, and inside Neovim's, Vim's
or Emacs's terminal it skips the gated queries; each fact says why. A
disabled prober says why under JetBrains too.

On Windows, `termcap.Env`'s lookups ignore the case of a name, as
Windows's environment does: `Path` and `PATH` are one variable. Elsewhere
they match exactly.

## Do not probe without input

A program run with `tea.WithInput(nil)` must not probe: tea skips its own
queries then, but the replies to the prober's would leak into the shell.
Pass `termcap.WithDisabled()`, or do not embed a prober at all.

With no program at all, such as in your own command line,
`termcap.EnvCaps(env, goos, opts...)` gives the same facts a disabled
prober holds after the first `tea.EnvMsg`: the terminal's name and brand,
the multiplexer, the editor, the platform, whether the session is remote,
and light or dark from the environment. `WithAppearanceEnv` and
`WithOverride` apply; the options that only shape a probe change nothing,
the hooks are not run, and `Profile` stays unknown.

```go
caps := termcap.EnvCaps(termcap.Env(os.Environ()), runtime.GOOS,
    termcap.WithAppearanceEnv("PI_APPEARANCE"))
```

`launch.Run` always sets an input, so a program started through `launch`
can probe. Pass the prober to `launch.WithRestorer` as well: its resets,
such as mode 2031's, are then written on every outcome, a crash included
([Start the TUI from your CLI](commands.md#start-the-tui-from-your-cli)).

## Probe from a line-mode CLI

A program that runs no Bubble Tea drives a `termcap.Probe` itself. The
probe does no I/O, reads no clock and starts no goroutine: you give it
the environment, each reply and the time, and drain the events it
returns until `ProbeNoEvent`.

```go
p := termcap.NewProbe()
p.Start(termcap.Env(os.Environ()), time.Now())
for done := false; !done; {
    for e := p.NextEvent(); e.Kind != termcap.ProbeNoEvent; e = p.NextEvent() {
        switch e.Kind {
        case termcap.ProbeWrite:
            io.WriteString(tty, e.Data)
        case termcap.ProbeRunHook:
            v, ok := e.Run() // a hook you set, which may start a process
            p.HandleHook(e.Hook, v, ok)
        case termcap.ProbeDone:
            caps, done = e.Caps, true
        } // and ignore a kind you do not know
    }
    if done {
        break
    }
    deadline, _ := p.Timeout()
    raw, ok := readReply(tty, deadline) // your raw-mode reader: one whole reply
    if !ok {
        p.HandleTimeout(time.Now())
        continue
    }
    p.HandleReply(termcap.ParseReply(raw), time.Now())
}
defer io.WriteString(tty, p.Restore()) // mode 2031's reset, or ""
```

What Bubble Tea does for a TUI, the CLI does itself:

- **Probe only from raw mode, and never without input.** A reply nobody
  reads reaches the user's input line, so read every reply the batch
  asks for. DA1 comes last; its answer ends the probe, and each query
  still unanswered then is unsupported.
- **Read whole replies.** A reply can arrive split across reads, even
  inside a UTF-8 sequence. Join bytes into whole sequences before
  `ParseReply`; with the planned `termkey` decoder, do not `Flush` while
  `p.Awaiting()` is true
  ([0020-MADR](../decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  A2.4).
- **`ParseReply` takes what terminals send:** 7-bit and 8-bit
  introducers, and strings ended by BEL, ST, 0x9C, or old urxvt's bare
  ESC. Apple Terminal ends an OSC reply with BEL whatever the query used.
- **The probe asks DECRQM 2026 and 2027 itself,** under Bubble Tea's own
  rule: not over SSH, and not in Apple Terminal.
- **Write `Restore()` before you exit,** or pass
  `WithoutColorSchemeUpdates()`, as a TUI must.
- **Multiplexers.** tmux answers DA1 itself and passes a query outward
  only through its passthrough; GNU screen answers DA1 before the outer
  terminal's replies arrive, so its facts are thin.

## Drop split replies

A reply split across reads arrives as an unknown event and then as key
presses. `IsReplyFragment` recognises them; call it from a filter, once
per message, in order:

```go
prog := tea.NewProgram(m, tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
    if m.probe.IsReplyFragment(msg) {
        return nil
    }
    return msg
}))
```

## Inside tmux

tmux can say more about itself than the terminal can. The library never
runs a process, so run the command it returns, inside the same tmux, and
pass the result back:

```go
argv := termcap.TmuxQuery()              // tmux display-message -p …
out, err := exec.Command(argv[0], argv[1:]...).Output()
if err == nil {
    m.probe.SetTmux(termcap.ParseTmux(string(out)))
}
```

Event types inside tmux need `extended-keys-format csi-u`, and hyperlinks
need tmux 3.4. The OSC 99 and Kitty graphics queries go to the outer
terminal through tmux's passthrough, which needs `allow-passthrough on`.

## Notifications and focus

```go
n := termsvc.NewNotifier() // WhenUnfocused, Auto
// in Update: termsvctea.Observe(n, msg), beside the prober
return m, termsvctea.Notify(n, termsvc.Notification{Title: "Done", Body: "3 files changed"})
```

- **`Auto`** picks OSC 99 when the terminal answered its query, then OSC 9
  or OSC 777 by brand, then the bell; inside tmux or screen the sequence is
  wrapped for passthrough.
- **Focus comes from `tea.FocusMsg` and `tea.BlurMsg`,** which need
  `View.ReportFocus`, through `termsvctea.Observe`, which calls
  `n.SetFocus`; `termcap.CapsMsg` reaches `n.SetCaps` the same way. `WhenUnfocused` sends only after a blur;
  `UnlessFocused` also sends while focus is unknown; `Always` and `Never`
  do what they say.
- **Every notification delivers `NotifyResultMsg`**: `Sent`, or the `Skipped`
  reason (`disabled`, `empty`, `focused`, `focus-unknown`, `gated`,
  `failed`), and a backend's `Err`.
- **Text is cleaned:** escape sequences and controls removed, line breaks
  collapsed, cut to 80 cells for the title and 240 for the body.
  `WithNotifyFilter(f)` lets you decide last, such as for completed turns
  only; `WithBackend(b)` sends through a desktop API you supply, and
  `termsvc.BackendFunc` makes one of a function.
- **A backend runs under a deadline:** 5 s by default,
  `WithBackendTimeout(d)` to change it. `termsvctea.NotifyContext(ctx, n,
  x)` also ends it with your context; a backend that misses the deadline
  gives `failed`, with the context's error in `Err`. `termsvctea.Notify`
  is `NotifyContext` under `context.Background()`. The command decides on
  your loop and calls the backend only when it runs, off the loop.
- **From your own command line,** which runs no Bubble Tea program,
  `n.Prepare(ctx, x)` returns the bytes to write, with the same cleaning
  and numbering, and `deliver`, which runs a backend, if any, and returns
  the `NotifyResultMsg`. Write the bytes yourself, then call `deliver`;
  `n.Sequence(x)` gives the bytes alone, or the reason it would skip.
  Such a program never sees focus, so build the notifier
  `WithPolicy(termsvc.Always)`, and give it the environment's facts with
  `n.SetCaps(termcap.EnvCaps(…))`, or a `termcap.Probe`'s.

## Clipboard and links

- **`termsvctea.Copy(caps, text)`** writes OSC 52 through tea, and inside
  tmux also wraps it for passthrough. It delivers `CopiedMsg`:
  `Unconfirmed`, because OSC 52 never replies. With `WithClipboard(b)` your
  clipboard confirms or fails; `termsvc.ClipboardFunc` makes one of a
  function. Over 100,000 bytes it fails and sends nothing.
  `termsvctea.CopyContext` and `WithCopyTimeout` bound your clipboard as
  `NotifyContext` and `WithBackendTimeout` bound a backend.
- **`termsvc.CopySequence(caps, text)`** returns the bytes a copy writes,
  OSC 52 and inside tmux the same wrapped for passthrough, with the route,
  for your own command line to write. Over 100,000 bytes it returns
  `ErrCopyTooLarge`. `termsvc.PrepareCopy(ctx, caps, text, …)` gives the
  route and the `deliver` that reports it, calling your clipboard there.
- **`CopyPlan(caps)`** orders the routes to try: your clipboard, tmux's
  buffer (run `TmuxLoadBuffer()` with the text on standard input), OSC 52.
- **`ImageReadCommands(goos, wayland)`** lists the commands that read an
  image from the clipboard, for you to run.
- **`LinkDisplay(caps)`** says whether to show a label alone, as an OSC 8
  link, or the label with its URL, where links do not work.
  **`termsvc.Link(url, text)`** builds a link, refusing schemes other than
  http, https, file and mailto. **`LinkPolicy.Openable(url)`** decides
  whether a click may open it (http and https by default);
  `termsvctea.Open(policy, url)` delivers `OpenURLMsg`, and your program
  opens it.

## Titles, marks and more

| Function | For |
| :--- | :--- |
| `SanitizeTitle(s)` | `View.WindowTitle`: no controls or bidirectional marks, at most 240 characters |
| `PromptStart`, `CommandStart`, `CommandExecuted`, `CommandFinished(exit)` | OSC 133 marks in inline output; on the alternate screen they mean nothing |
| `Activity(vendor, state, t)`, `ParseActivity`, `termsvctea.ActivityBeacon` | an opt-in beacon for an editor that embeds the terminal, every 5 s |
| `Pointer(caps, shape)` | the mouse pointer's shape, on Ghostty and kitty outside a multiplexer |
| `ProgressSupported(caps)` | whether `View.ProgressBar` shows as a bar |

## A doctor command

`termcap.Report` writes one ASCII line per fact, then the findings: what
each reason token means and what would change it. Pass it the writer you
choose; `WithReportJSON()` writes JSON with a `schema_version`, 2 since
`v2.0.0`, which writes the colour profile by its token (`truecolor`).

```go
func doctor(w io.Writer, caps termcap.Caps) error {
    return termcap.Report(w, caps, termcap.WithReportWidth(100))
}
```

`termcap.Findings(caps)` returns the findings alone.

## Test with fake terminals

`termcaptest.Run(t, model, profile)` runs your model as a real
`tea.Program` against a fake terminal, waits for `CapsMsg`, sends
`termcaptest.StopMsg`, on which your model must quit through its prober,
and returns the `Caps`. The profiles are `Kitty()`, `XTerm()`,
`Tmux(passthrough)`, `AppleTerminalSSH()`, `JetBrains()`, `DA1Only()` and
`Silent()`; build your own from `Profile`. `NewTerminal(p)` keeps the
terminal, so a test can read `Output()`, `Sequences()` and `Painted()`
after the run.

For identity alone, `termcap.FromEnv(env, goos)` is a pure function of an
environment, and `WithGOOS` pins the OS a prober reads it for.

## What the library never does

- **It never reads the process environment.** The environment comes from
  `tea.EnvMsg`, which under wish is the SSH session's, or from the `Env` a
  CLI passes to `Start`.
- **It never starts a process.** `TmuxQuery`, `TmuxLoadBuffer` and
  `ImageReadCommands` return commands for you to run.
- **It never writes to the terminal itself.** Every sequence leaves as a
  `tea.Cmd`, a `ProbeWrite` event, or a string you place.
