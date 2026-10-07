# Terminal capabilities and services

For a Bubble Tea v2 program that wants to know what its terminal can do,
over SSH and inside tmux as well as locally, and to send notifications,
clipboard writes and links that suit it. Three packages do the work:

- `termcap` learns the terminal's capabilities from one batch of queries
  that ends with DA1, and reports each fact with where it came from.
- `termsvc` turns those facts into commands and strings: notifications, the
  clipboard, links, prompt marks, titles and the activity beacon.
- `termcap/termcaptest` runs your program against scripted fake terminals,
  so all of this can be tested without a real one.

Why they are built this way is in
[0005-MADR](../decisions/0005-MADR-terminal-capabilities-and-services.md),
with its amendments A1 to A5. The examples `ExampleProber` and
`ExampleReport` compile with the package
(`go doc -all github.com/maccavelli/go-tui-lib/termcap`).

## Embed a prober

A `Prober` is not a `tea.Model`. Your model keeps one, returns its `Init`
from your `Init`, passes it every message, and quits through it:

```go
type model struct {
    probe *termcap.Prober
    ws    *workspace.Workspace
    caps  termcap.Caps
}

func newModel() *model {
    return &model{
        probe: termcap.New(),
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
  XTGETTCAP itself, and acts on the answers; the prober reads tea's
  messages instead.
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
| `KeyboardFlags(caps)` | what to put in `View.KeyboardEnhancements`: event types only where the terminal reports them cleanly |
| `caps.Keyboard()` | the Kitty fact, those enhancements, the flags the terminal accepted, and whether a binding may wait for a release |
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
| `WithDisabled()` | sends nothing; the first `tea.EnvMsg` delivers the environment's facts |
| `WithAppearanceEnv(name)` | reads light or dark from `LC_` + `name`, which a default sshd forwards, and from `name`, which wins when both are set |
| `WithAppearanceHook(f)` | asks the desktop, through your function |
| `WithConsoleHost(f)` | asks Windows whether this is the classic console |
| `WithGOOS(goos)` | reads the identity for another OS: a test, or a wish server |

Under JetBrains the prober sends nothing at all, and inside Neovim's, Vim's
or Emacs's terminal it skips the gated queries; each fact says why.

## Do not probe without input

A program run with `tea.WithInput(nil)` must not probe: tea skips its own
queries then, but the replies to the prober's would leak into the shell.
Pass `termcap.WithDisabled()`, or do not embed a prober at all.

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
// in Update: n.Update(msg), beside the prober
return m, n.Notify(termsvc.Notification{Title: "Done", Body: "3 files changed"})
```

- **`Auto`** picks OSC 99 when the terminal answered its query, then OSC 9
  or OSC 777 by brand, then the bell; inside tmux or screen the sequence is
  wrapped for passthrough.
- **Focus comes from `tea.FocusMsg` and `tea.BlurMsg`,** which need
  `View.ReportFocus`. `WhenUnfocused` sends only after a blur;
  `UnlessFocused` also sends while focus is unknown; `Always` and `Never`
  do what they say.
- **Every call delivers `NotifyResultMsg`**: `Sent`, or the `Skipped`
  reason (`disabled`, `empty`, `focused`, `focus-unknown`, `gated`,
  `failed`), and a backend's `Err`.
- **Text is cleaned:** escape sequences and controls removed, line breaks
  collapsed, cut to 80 cells for the title and 240 for the body.
  `WithGate(f)` lets you decide last, such as for completed turns only;
  `WithBackend(b)` sends through a desktop API you supply.

## Clipboard and links

- **`termsvc.Copy(caps, text)`** writes OSC 52 through tea, and inside tmux
  also wraps it for passthrough. It delivers `CopiedMsg`: `Unconfirmed`,
  because OSC 52 never replies. With `WithClipboard(b)` your clipboard
  confirms or fails. Over 100,000 bytes it fails and sends nothing.
- **`CopyPlan(caps)`** orders the routes to try: your clipboard, tmux's
  buffer (run `TmuxLoadBuffer()` with the text on standard input), OSC 52.
- **`ImageReadCommands(goos, wayland)`** lists the commands that read an
  image from the clipboard, for you to run.
- **`LinkDisplay(caps)`** says whether to show a label alone, as an OSC 8
  link, or the label with its URL, where links do not work.
  **`termsvc.Link(url, text)`** builds a link, refusing schemes other than
  http, https, file and mailto. **`LinkPolicy.Openable(url)`** decides
  whether a click may open it (http and https by default); `Open` delivers
  `OpenURLMsg`, and your program opens it.

## Titles, marks and more

| Function | For |
| :--- | :--- |
| `SanitizeTitle(s)` | `View.WindowTitle`: no controls or bidirectional marks, at most 240 characters |
| `PromptStart`, `CommandStart`, `CommandExecuted`, `CommandFinished(exit)` | OSC 133 marks in inline output; on the alternate screen they mean nothing |
| `Activity(vendor, state, t)`, `ParseActivity`, `ActivityBeacon` | an opt-in beacon for an editor that embeds the terminal, every 5 s |
| `Pointer(caps, shape)` | the mouse pointer's shape, on Ghostty and kitty outside a multiplexer |
| `ProgressSupported(caps)` | whether `View.ProgressBar` shows as a bar |

## A doctor command

`termcap.Report` writes one ASCII line per fact, then the findings: what
each reason token means and what would change it. Pass it the writer you
choose; `WithReportJSON()` writes JSON with a `schema_version`.

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
  `tea.EnvMsg`, which under wish is the SSH session's.
- **It never starts a process.** `TmuxQuery`, `TmuxLoadBuffer` and
  `ImageReadCommands` return commands for you to run.
- **It never writes to the terminal itself.** Every sequence leaves as a
  `tea.Cmd` or a string you place.
