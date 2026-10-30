---
status: accepted
date: 2026-10-25
decision-makers: the repository owner
consulted: "issues #7, #8, #9 and #10 of this repository's tracker, with their comments; gobble-cli at 5fb6928: docs/decisions/0027-MADR-native-line-editor-word-wrap.md, 0028-MADR-reply-word-wrap.md, 0029-MADR-permission-prompt-layout.md, 0030-MADR-resize-and-performance.md and 0032-MADR-go-tui-lib-integration-boundary.md, with their PLANs, and internal/cli; golang.org/x/term v0.47.0; reedline 0.52.1, inquire 0.9.4, liner v1.2.3 and prompt_toolkit's documentation"
informed: gobble-cli, through issues #7 to #10
---
# Give a line-mode CLI an inline question, a picker, a composer and a streamed transcript, as I/O-free Charm-free cores on one live region, with their Bubble Tea forms in sub-packages

## Context and Problem Statement

On 2026-10-25 the owner asked: "let's work through the open issues.
research the codebase, analyze the 7-10 issue requested items. determine
the most go-native, idiomatic, 1.27.3 compatible, detailed, and
comprehensive way to implement them. present findings as an madr for
review."

Issues #7 to #10 come from one decision in another repository, as #2 to
#6, #11 and #12 did
([0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)).
gobble-cli `docs/decisions/0032-MADR-go-tui-lib-integration-boundary.md`
lets gobble's native line-mode CLI import a go-tui-lib package only when
its whole import closure holds no Charm module but `x/ansi` (its D1 and
A1.1). gobble builds no copy of what go-tui-lib does: it waits, and files
what it needs here (its D3, D5). Each issue's first comment records the
A1 relaxation: the line-mode form "may measure, wrap and clean with
`x/ansi` and `internal/sanitize`", and links no Bubble Tea, Bubbles, Lip
Gloss, `colorprofile` or `ultraviolet`.

Every fact below about this repository was read on 2026-10-25 at
`65d437b`; every fact about gobble at gobble-cli `5fb6928`, with its path.

### What the four issues ask

| Issue | Asks for | gobble today | gobble's record |
| :--- | :--- | :--- | :--- |
| #7 | an inline question: a wrapped title, notes, rows of keyed options never split, a reason read after a key, the answer echoed; laid out as a pure function of the question and the width; keys read over an `io.Reader`; in the TUI too | one line, `permissionLine` (`internal/cli/permission.go:61-94`), which wraps mid-word | `0029-MADR-permission-prompt-layout.md` D1, D2, D5, D6; its PLAN's P2, the layout, is stopped to wait for #7 (`0029-PLAN-permission-prompt-layout.md:163-168`) |
| #8 | an inline filterable picker: type to filter with the palette's matcher, arrows, Enter, Escape, a few rows tall, redrawn in place, the choice left in scrollback as one line; for `/model` and sessions | `/model` answers as Markdown tables streamed as a reply (`internal/acpserver/choice.go:297-368`); sessions are chosen by `-c` or `--session`, by id or unique prefix (`internal/cli/sessions.go:126-174`); there is no `/resume` | `0032-MADR-go-tui-lib-integration-boundary.md` D4 |
| #9 | a composer: the line in whole words to the width, a multi-row line redrawn by its changed rows only, history, bracketed paste, an external editor, a resize redrawn by how the terminal re-wraps, a bytes-per-key budget; sharing stdin; never closing a Windows console handle while a read is pending | `internal/cli/editor`, about 1,800 lines | `0027-MADR-native-line-editor-word-wrap.md`, `0030-MADR-resize-and-performance.md` |
| #10 | a transcript: a reply's Markdown styled and wrapped at whole words as it streams, code, tables and long words kept whole; on a resize the conversation, bounded in bytes, drawn again at the new width once resizing settles | `internal/cli/render`, `internal/cli/transcript.go`, `internal/cli/view.go` | `0028-MADR-reply-word-wrap.md`, `0030-MADR-resize-and-performance.md` |

### What this repository has

* **The Charm-free cores** are the list `TestCoresAreCharmFree` checks
  (`internal/conformance/charmfree_test.go`): `command`, `fuzzy`,
  `glyph`, `internal/enum`, `internal/sanitize`, `keymap`, `layout`,
  `safetext`, `termcap`, `termkey`, `termsvc` and `when`. Every part a
  line-mode component needs is among them:
  * **keys:** `termkey.Decoder` reads bytes as keys, pastes and unknown
    sequences, with no goroutine and no clock; it holds a lone `ESC` and
    says so with `Pending`, and the host calls `Flush` after
    `DefaultEscTimeout`, 50 ms. It reads the legacy CSI and SS3 keys,
    xterm's modifier parameter and modifyOtherKeys, the Kitty protocol and
    bracketed paste (0020-MADR D7);
  * **bindings:** `keymap`'s `Dispatcher` takes `termkey` keys and time
    (`Key`, `Expire`), and `DefaultRules`-style defaults bind command IDs
    in scopes;
  * **text:** `safetext` and `internal/sanitize` (`Line`, `Truncate`, never
    wider than asked); `x/ansi`'s `StringWidth`, `Wordwrap`, `Wrap`,
    `Hardwrap` and `Truncate`, each with a `Wc` form, and its `Method`,
    `WcWidth` or `GraphemeWidth`;
  * **ranking:** `fuzzy.Matcher` and `Rank`, `fuzzy.Index` for large
    lists, `fuzzy.Cells` for highlights;
  * **glyphs:** `glyph.Set`, with `Prompt`, `Mask`, `Focus`, `Ellipsis`,
    `BadgeOpen` and `BadgeClose`, `Done`, `Failed`, `Running`, `Pending`;
  * **the terminal's facts:** `termcap.Probe`, an engine that does no I/O
    (0020-MADR A2), and `ParseReply`.
* **The I/O-free engine is this repository's form for a terminal
  protocol.** 0020-MADR A2.3's eight rules, for `termcap.Probe`: no I/O,
  no clock, no goroutine; time in, a deadline out; events pulled from a
  struct with a `Kind`, and a host ignores a kind it does not know; an
  event that needs the host to act only after the host opts in; fields
  only grow; no dependency type where one of the repository's own will
  do; strings out, never a shared buffer; one goroutine drives it.
* **What a line-mode CLI cannot use today:**
  * `palette` is a Bubble Tea package: `Hit.Run` is
    `func(jsontext.Value) tea.Cmd` (`palette/provider.go:87`), and its
    picker is a workspace pane;
  * `stream`, as accepted in
    [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
    §1, imports `frame`, which imports Bubble Tea, and `workspace`; it is
    not built (its PLAN is `proposed`);
  * `launch` imports `github.com/charmbracelet/x/term`
    (`launch/restore.go:20-30`, `launch/terminal.go:15-28`), which only it
    may import (depguard's `xterm` rule).
* **The library's rules** (AGENTS.md, "TUI conventions" 1 and 2 and "API
  conventions" 9; `internal/conformance`): output goes where the caller
  says; the caller owns the screen and the signals; no read of the
  process's environment, no `os.Exit`, no started process, no `os/exec`.

### What gobble does, and what a library must keep

The editor (`internal/cli/editor`):

* **Its API** is `editor.New(in io.Reader, out, errw io.Writer, opts
  Options)` and `ReadPrompt(ctx) (string, error)` (`editor.go:102`,
  `:127`). It never opens stdin. `Options` has the prompt and its
  continuation, a `History` interface of `Add`, `Len` and `At`
  (`:24-28`), a completion function, a header, the width and height as
  functions, a resize channel and a reprint function (`:40-58`).
  `MaxWidth` is 120 and `DefaultWidth` 80 (`:62-65`).
* **Stdin is read only when asked.** One goroutine reads 512-byte chunks,
  one read per request that finds nothing buffered (`reader.go:12`,
  `:49-70`, `:115-137`), because a Windows console read keeps the mode it
  started in: a read started before raw mode returned 379 bytes at the
  first Enter in 6 of 10 runs under Console Host and 3 of 10 under Windows
  Terminal, and 0 of 10 after the change
  (`0027-MADR-native-line-editor-word-wrap.md` D5's amendment,
  `:114-124`). The reader is never closed, since closing console input
  while a read is pending hangs the process (`reader.go:28`, `:45-48`);
  `TestNoCloseOnStdin` scans the files for any `.Close` call
  (`keys_test.go:107-127`).
* **Raw mode** is `golang.org/x/term`'s `MakeRaw` and `Restore`, with VT
  input enabled on Windows (`editor.go:447-457`); the line session leaves
  it for every submission (`internal/cli/line.go:202-218`).
* **Keys** are its own decoder (`buffer.go:83-116`): no SS3, no Kitty, no
  modifyOtherKeys, and no timeout for a lone `ESC`.
* **Layout** breaks only at spaces, leaves a word wider than a row whole
  (*corrected by A5 item 5:* its code cuts such a word at the row's end),
  indents continuation rows by the prompt's width, and caps the width at
  120 (`layout.go:19-88`). It measures with `golang.org/x/text/width`, so
  a combining mark, a ZWJ sequence or an emoji modifier counts one cell
  each (`layout.go:99-116`): the defect 0032-MADR A1 names, which
  `x/ansi`'s grapheme width fixes.
* **Drawing** skips unchanged rows, rewrites a changed row from its shared
  prefix, clears with `CSI K` and `CSI J`, moves with `CSI n A`, `\r` and
  `CSI n C`, and wraps every write in mode 2026 with the cursor hidden
  (`draw.go:12-17`, `:62-139`). It never erases first, because Windows
  Terminal flickered when it did (0027-MADR D3's amendment, `:89-96`).
  A key's drawing writes at most the changed text plus 40 bytes
  (0030-MADR D5; `budget_test.go:8-45`).
* **A resize** is redrawn by how the terminal re-wraps. 0030's PLAN, P1,
  narrowed 80 columns to 41 to 43: WezTerm, kitty, Terminal.app and the
  Windows pseudo-console re-wrap and move the cursor down; iTerm2 and
  xfce4-terminal (VTE) re-wrap and keep the cursor's row; xterm cuts
  (`0030-PLAN-resize-and-performance.md:143-163`). The editor asks for the
  cursor (`CSI 6 n`), waits 150 ms, takes the report out of the input
  (`editor.go:263-296`, `:364-385`), and decides cut or re-wrap from it
  (`draw.go:45-60`); with no report it falls back to reading
  `XTERM_VERSION` from the environment (`editor.go:308`). 150 ms after the
  last resize it reprints the transcript (`editor.go:318-335`).
* **History** is a file of at most 1000 entries, one per line with `\n`
  and `\` escaped, consecutive duplicates dropped, saved by an atomic
  rename at mode 0600 (`history.go:13`, `:76-105`).
* **The external editor** is `Compose(ctx, env, configured, dir, initial,
  run func(*exec.Cmd) error)`: it reads `GOBBLE_EDITOR`, the setting,
  `VISUAL` and `EDITOR`, and builds an `*exec.Cmd` the caller runs
  (`external.go:20-122`).
* **Keys bound:** the arrows and Ctrl+B and Ctrl+F; Home, End, Ctrl+A and
  Ctrl+E to the line's ends; Alt or Ctrl with the arrows, `ESC b` and
  `ESC f` by space-delimited words; Up, Down, Ctrl+P and Ctrl+N by visual
  row, then history; Backspace, Ctrl+H, Delete, Ctrl+D (end of input on an
  empty line), Ctrl+K, Ctrl+U, Ctrl+W, Ctrl+T, Ctrl+L, Tab to completion,
  Enter; a trailing `\` continues the line; a 4096-rune limit; Ctrl+C
  clears a line, and on an empty line arms a 2 s window to exit
  (`buffer.go:16`, `:204-362`; `keys.go:119-135`). There is no undo, no
  kill ring and no Shift+Enter newline. A bracketed paste is one
  submission, not editable (0027-MADR Q2=A).
* **Tests** run on `termtest.Screen`, a cell grid with pending wrap,
  scrolling, the cursor moves, `J`, `K`, DECTCEM, answered DSR, and
  `Resize(columns, ReflowDown|ReflowKeepRow|Cut)`
  (`internal/cli/term/termtest/screen.go:161-252`); a parity test against
  x/term's `Terminal` over 55 keys (`buffer_test.go:210-366`); per-key
  benchmarks with a budget test that fails past three times the baseline
  (`perf_test.go:14-32`).

The reply and the transcript (`internal/cli/render`, `transcript.go`,
`view.go`):

* **Markdown is gobble's own,** a port of goose's MarkdownBuffer, with no
  goldmark and no glamour (`render/mdstream.go:9-31`). It holds back an
  open fence, a table until a blank line, a heading until its line ends,
  an unclosed inline construct, and the incomplete last line
  (`mdstream.go:33-90`, `:154-285`).
* **It styles in place and keeps every marker:** headings bold, fences
  dim with the language in cyan, inline code dim, list and quote markers
  cyan, emphasis by its markers, from bold, dim, italic, underline and the
  eight colours; with colour off, the input comes back byte for byte
  (`render/mdstyle.go:9-198`; `term/style.go:5-47`). The colour decision
  takes an injected environment and never reads the process's
  (`term/caps.go:29-85`).
* **It wraps at whole words after styling** (0028-MADR D7, amended):
  prose at spaces, a word wider than a row whole on its own row; fences,
  indented code, table rows and headings unwrapped; list continuations
  under their text; quotes repeating `>`; no row starting with a word that
  would open a block; escapes zero-width and kept with their word; the
  output the same for any split of the input (`render/wrap.go:8-453`;
  `TestWrapAnySplit`).
* **The transcript** keeps ordered pieces, each fixed, a message, or
  drawn by a function of the width; replies keep their sanitised Markdown
  source (`transcript.go:25-55`; `view.go:495-504`). It is bounded at 4
  MiB, oldest first, and a reprint draws the newest pieces back to 10,000
  rows, after a dim "(earlier output is not shown again)" when it cut
  (`transcript.go:31-125`). A reprint waits for the view to be at a line
  start with both wrappers idle, then writes `CSI ?2026h CSI 2J CSI 3J CSI
  H`, the text, and `CSI ?2026l` (`view.go:444-487`). It clears the
  scrollback from before the program started (0030-MADR:139).

The question (`0029-MADR-permission-prompt-layout.md`):

* D1: the whole title wrapped at spaces, continuation indent 2, a long
  word whole; `Patterns:` when answers are remembered; rows `Allow [1]
  once [2] session [3] project [4] everywhere` and `Reject [5] … [8]`;
  `[f] reject and say why   [c] cancel`; notes above; the answer echoed
  after `> ` in words. D2: the other questions use a title and rows that
  never split an option. D5: the width is stderr's, at most 120. D6:
  everything sanitised.
* Keys are one byte each, no Enter; a key not shown is ignored; `c`,
  Ctrl+C or the end of input cancel; `[f]` reads a reason up to Enter,
  echoed, with Backspace, capped at 2048 bytes
  (`internal/cli/permission.go:111-179`).
* The other questions: trust (five options, `internal/cli/trust.go:166-205`),
  leaving plan mode (two), a grant (two), a personality switch (two), and
  a tool call (eight, its title cut at 2000 runes,
  `internal/acpserver/question.go:55-94`).

### What other line editors do

* **golang.org/x/term's `Terminal`** owns reading: `NewTerminal(c
  io.ReadWriter, prompt)`, a blocking `ReadLine`, and a `History`
  interface of `Add`, `Len` and `At`. Its `Write`, while a line is on the
  screen, clears the prompt and the line, writes the text, and draws the
  prompt and the line again below it (`terminal.go`, v0.47.0): printing
  above a live prompt.
* **reedline** (Nushell, 0.52.1) owns reading too (`read_line`), with a
  `History` trait and `FileBackedHistory`, a `Validator` for multi-line
  input, menus, a hinter, `with_buffer_editor(Command, PathBuf)` for an
  external editor, and an experimental `ExternalPrinter`, a bounded
  channel whose lines print above the prompt
  (<https://docs.rs/reedline/latest/reedline/struct.ExternalPrinter.html>).
* **prompt_toolkit's `patch_stdout()`** makes other output print above an
  active prompt: "This ensures that other output won't destroy the
  prompt"
  (<https://python-prompt-toolkit.readthedocs.io/en/master/pages/asking_for_input.html>).
* **inquire** (Rust, 0.9.4) has `Select`, `MultiSelect`, `Text`,
  `Confirm`, `CustomType`, `Password` and `Editor` prompts with custom
  list filters, on crossterm, termion or console
  (<https://docs.rs/inquire/latest/inquire/>).
* **liner** (Go, v1.2.3) puts the terminal in raw mode in `NewLiner`,
  prompts with `Prompt`, and reads and writes history with
  `ReadHistory(io.Reader)` and `WriteHistory(io.Writer)`, limited to 1000
  (<https://pkg.go.dev/github.com/peterh/liner>).

Each of them owns the reader, and so the terminal's mode, for as long as a
prompt is active.

### Probed here

A scratch program over this repository's `termkey` and `termcap`, on
2026-10-25:

| Bytes | `termkey.Decoder` gives | `termcap.ParseReply` gives |
| :--- | :--- | :--- |
| `CSI 12 ; 40 R`, a cursor report | `InputUnknown` | `ReplyUnknown` |
| `CSI 1 ; 5 R`, a cursor report on row 1 | the key `ctrl+f3` | `ReplyUnknown` |
| `CSI 1 ; 1 R` | the key `f3` | `ReplyUnknown` |
| `CSI ? 12 ; 40 R`, DECXCPR's report | `InputUnknown` | `ReplyUnknown` |

A cursor report on the first row reads as a modified F3, the known
ambiguity of the two encodings, and `ParseReply` has no kind for a cursor
report.

### Go

Go 1.27.3, released 2026-10-19, "includes security fixes to the
crypto/x509 package" (<https://go.dev/doc/devel/release>). Nothing here
uses `crypto/x509`. The floor stays 1.27.2 (AGENTS.md, API conventions
11; [0016-MADR-go-1-27-2-for-go-2026-6604.md](0016-MADR-go-1-27-2-for-go-2026-6604.md)),
and nothing below needs a feature newer than 1.27. Both are tested: CI
takes its version from `go.work` (`go-version-file: go.work`), the floor,
and the macOS, Linux and Windows hosts the gates run on have `go1.27.3`
(`go version`, 2026-10-25). gobble's `go.mod` says `go 1.27.3`, which a
module requiring this one at 1.27.2 may.

### The question

How does go-tui-lib give a line-mode CLI an inline question, a picker, a
composer and a streamed transcript, each linking no Charm module but
`x/ansi`, each sharing its logic with the TUI, each testable without a
terminal, and keeping what gobble measured, without reading the
environment, starting a process or owning the terminal?

## Decision Drivers

* **One implementation for both modes** (gobble's 0032-MADR D3; 0020-MADR's
  first driver). A line-mode CLI and the TUI share each component's logic.
* **The closure, not the import** (0020-MADR D1, D11). Every package a
  line-mode CLI imports joins the cores, and the gate holds it there.
* **The library's rules.** No read of the environment, no process, no
  write to `os.Stdout` or `os.Stderr`; the caller owns the screen, the
  signals and the terminal's mode.
* **What gobble measured must survive:** reading only when asked on
  Windows, never closing console input, the three re-wrap kinds, drawing
  without erasing first, the bytes-per-key budget, the settle before a
  reprint, and the reprint's bound.
* **Testable without a terminal** (issue #7): a layout is a pure
  function; a component's whole behaviour is a function of its inputs and
  the time.
* **Correct widths.** Grapheme clusters, by `x/ansi`, in the width method
  the terminal uses: `WcWidth` until it reports mode 2027, as the
  workspace measures.
* **Untrusted text is a security boundary.** Model output, tool titles,
  file names and session titles reach the screen; each is cleaned once,
  on the way in.
* **Go-native:** small packages named for what they hold, `io.Reader` and
  `io.Writer` at the edges, an interface for a hook with a `…Func`
  adapter, options that are opaque, no new module unless a record names
  it.
* **gobble's order.** The question first (its D4, issue #7), then the
  picker, the composer and the transcript (`0032-PLAN:142-153`).

## Considered Options

* **A. An I/O-free engine for each component, each its own Charm-free
  package, on one shared live-region package, with a Bubble Tea form in a
  `<pkg>tea` sub-package where the TUI needs one; the transcript on a
  Charm-free `stream` core split from 0009's.**
* **B. Blocking components that own a reader and a writer,** as x/term's
  `Terminal`, liner and gobble's editor do: `ReadLine`-style calls that
  return the answer.
* **C. Bubble Tea programs drawn inline,** without the alternate screen,
  as charm.land's `huh` forms and erikgeiser's promptkit do.
* **D. Wrap an existing Go line editor,** such as peterh/liner or
  chzyer/readline, behind this repository's API.
* **E. Option A's engines, all in one package,** `inline`, with one
  import path.
* **F. Change nothing:** gobble keeps its own.

## Decision Outcome

*Accepted 2026-10-25 by the owner, who committed the record with its
answers and its PLAN and said "committed, proceed".*

*The owner answered Q1 to Q7 on 2026-10-25 (Owner questions, below):
each the recommendation but Q5, where the TUI's picker is a palette
provider over `pick`'s entries. D1, D5 and D12 say so.*

Chosen option: **"A"**, because it is the only option that gives gobble
clean closures, shares one engine between both modes, keeps the
terminal, its mode and its reader the host's, and makes every behaviour
gobble measured a deterministic test. B reads the terminal for the length
of a prompt, which is the shape gobble had to change for Windows, and it
cannot print a streaming reply above an editor without a lock across
goroutines; C links Bubble Tea, which the boundary forbids; D brings a
module that owns stdin, measures without graphemes and reads its own
keys, beside `termkey`; E links the editor's and the picker's
dependencies into a program that asks one question; F contradicts both
repositories' records.

The decision has thirteen parts, D1 to D13.

### D1. The packages

| Package | Holds | Imports | Issue |
| :--- | :--- | :--- | :--- |
| `inline` | `Region`, the live rows below the scrollback; `Reflow`; `Reader`, the read-when-asked edge | `termkey`, `termcap`, `glyph`, `internal/enum`, `internal/sanitize`; `x/ansi` | all |
| `inline/inlinetest` | `Screen`, a terminal grid for tests, with the three re-wrap kinds | `x/ansi`; *amended:* `inline` (A1), `internal/sanitize` (A10) | all |
| `ask` | `Question`, `Row`, `Answer`, `Layout`, `Asker`, `Response` | `inline`, `lineedit`, `glyph`, `internal/sanitize`; `x/ansi` | #7 |
| `ask/asktea` | the question as a workspace overlay pane | `ask`, `workspace`; Bubble Tea | #7 |
| `pick` | `Entry`, `Chooser` | `inline`, `lineedit`, `fuzzy`, `internal/sanitize`; `x/ansi` | #8 |
| `palette` (existing) | gains `FromEntries`, the TUI's form of `pick` (Q5) | `pick`, a core, beside its imports today | #8 |
| `lineedit` | `Composer`, `History`, `MemHistory`, `Completer` | `inline`, `keymap`, `termkey`, `glyph`, `internal/sanitize`; `x/ansi` | #9 |
| `lineedit/lineedittea` | the composer as a workspace pane | `lineedit`, `keymap/keymaptea`, `workspace`; Bubble Tea | #9 |
| `stream` | 0009's `Doc`, `Renderer` and `Plain`; the block `Scanner`; `Printer`, the line-stable Markdown printer; `Markup` and `Style` | `safetext`, `internal/enum`, `internal/sanitize`; `x/ansi` | #10 |
| `stream/streamtea` | 0009's `Buffer` and `Pane` | `stream`, `frame`, `workspace`; Bubble Tea | (0009) |
| `transcript` | `Transcript`, the bounded, reprintable conversation | `stream`, `inline`, `internal/sanitize`; `x/ansi` | #10 |

* **The cores gain** `inline`, `inline/inlinetest`, `ask`, `pick`,
  `lineedit`, `stream` and `transcript`, each in the commit that builds it
  (0020-MADR D11). `TestCoresAreCharmFree` and depguard's `charmfree`
  rule hold them.
* **No module is added.** Every import above is in the root module's list
  today (AGENTS.md, Dependencies). goldmark is not taken (D8).
* **Names** were checked with `python3 -m tools.verify names` over the 31
  planned type names: "31 planned name(s), clean", after two renames, the
  composer from `Editor` (`termcap.Editor`) and the styles from `Styles`
  (`theme.Styles`). An option type in each package is `Option`, which the
  glossary and `sharedNames` allow everywhere.
* **Each package is small and stands alone** (0020-MADR's split, `termkey`
  beside `keymap`): a program that asks one question links `ask`,
  `inline` and `lineedit`, and no `fuzzy`, no `stream`.

### D2. The engine contract

Every component is an engine under 0020-MADR A2.3's rules, so that one
host loop drives any of them, and a test is a transcript of bytes and
times.

* **In:** `HandleInput(in termkey.Input, now time.Time)`, one decoded
  input, in order; `Resize(width int, now time.Time)`; `HandleTimeout(now
  time.Time)` once the deadline `Timeout() (time.Time, bool)` gave has
  passed. The host decodes with `termkey.Decoder` and calls `Flush` after
  the escape timeout.
* **Out:** `NextEvent()`, drained until its zero kind, each package's
  event a struct with a `Kind` (`AskEvent`, `PickEvent`, `EditEvent`):
  * the base kinds every host handles: write `Data` to the terminal, in
    order; and the component is done, with its result;
  * kinds that need the host to act, such as the composer's external
    editor, only after an option opts in (A2.3 rule 3); a host ignores a
    kind it does not know.
* **No I/O, no clock, no goroutine, no environment.** `now` comes in; a
  deadline goes out; bytes are strings. One goroutine drives an engine.
* **Width and method are the host's.** `Resize` gives the width in cells;
  `WithMethod(ansi.Method)` the width method, `ansi.WcWidth` by default,
  `ansi.GraphemeWidth` once `termcap` reports mode 2027, as the workspace
  measures. No component caps the width: gobble's 120 is gobble's.
* **Text is cleaned on the way in,** every caller's and every model's
  string, by `internal/sanitize.Line` or a `safetext` preset, once.
* **Glyphs and styles are the host's:** `WithGlyphs(glyph.Set)`, and an
  SGR `stream.Markup` (D7) where a component colours, so a host's
  `NO_COLOR` and its `termcap.ColorProfile` decide, read through
  `termcap.Env`, never by the library.

### D3. `inline`: the live region, and the edge

* **`Region` owns the rows below the scrollback** that a component draws
  and redraws in place, and nothing else:

  ```go
  type Region struct{ /* the rows as drawn, the cursor, the width, the reflow state */ }

  func NewRegion(o ...Option) *Region            // WithMethod, WithSynchronized, WithReflow
  func (r *Region) Draw(rows []string, row, col int) string // the bytes that show rows, the cursor at (row, col)
  func (r *Region) Print(text string) string     // text above the region, then the region again
  func (r *Region) Commit(rows []string) string  // rows left in the scrollback; the region is empty
  func (r *Region) Clear() string                // the region removed
  func (r *Region) Resize(width int, now time.Time) string // a cursor query, or the redraw
  func (r *Region) HandleInput(in termkey.Input, now time.Time) (string, bool) // a cursor report, if it is one
  func (r *Region) Timeout() (time.Time, bool)
  func (r *Region) HandleTimeout(now time.Time) string
  func (r *Region) Restore() string              // bytes that reset what it set
  ```

* **Drawing is gobble's** (0027-MADR D3's amendment, 0030-MADR D5):
  unchanged rows skipped; a changed row rewritten from its shared prefix,
  in cells, under the width method; `CSI K` for a shorter row, `CSI J`
  for rows the region no longer has; moves by `CSI n A`, `\r` and `CSI n
  C`, never an absolute position; a first draw from `\r CSI J`; every
  write wrapped in mode 2026 with the cursor hidden, unless
  `WithSynchronized(false)`; a region taller than the height given is
  windowed around the cursor's row. A test holds a one-key change to at
  most the changed text plus 40 bytes.
* **`Print` is x/term's `Terminal.Write`:** clear the region, write the
  text, draw the region again below it. A streaming reply prints above a
  composer this way, with no lock: both run on the host's loop.
* **A resize follows the terminal's re-wrap** (0030-MADR D2). `Reflow` is
  an enum, `ReflowUnknown`, `ReflowDown`, `ReflowKeepRow` and
  `ReflowCut`, tokens from `internal/enum`. On `Resize` the region writes
  `CSI 6 n` and waits, by default 150 ms, for the report; gobble's rule
  (`draw.go:45-60`) reads cut or re-wrap from it, and the region moves up
  by the rows as drawn, or as re-wrapped, and redraws. With no report it
  takes `WithReflow`'s kind, which the host sets from what it knows, such
  as `termcap`'s identity of xterm: that replaces gobble's read of
  `XTERM_VERSION`. A keep-row terminal is redrawn as a re-wrap, and `CSI
  A` clamps at the top, as in gobble (0030-PLAN:272-273).
* **The cursor report is the region's to read.** While it waits for one,
  `HandleInput` takes `CSI row ; col R` out of the input, as gobble's
  `takeCursorReport` does, including the row-1 report `termkey` reads as
  a modified F3 (Probed here); every other input goes on to the
  component. `termcap.ParseReply` gains `ReplyCursorPosition`, with `Row`
  and `Col`, an addition under 0020-MADR A2.3 rule 4, and `CSI ? row ;
  col R` (DECXCPR) is read the same.
* **`Reader` is the one part with a goroutine**, at the edge: it reads the
  host's `io.Reader` only when asked, and never closes it, for the reasons
  gobble measured on Windows (gobble's 0027-MADR D5's amendment, and its
  rule against closing console input while a read is pending):

  ```go
  func NewReader(r io.Reader) *Reader
  func (r *Reader) Read(ctx context.Context, p []byte) (int, error) // one read in flight at most; ctx ends the wait, not the read
  ```

  The program shares it: its own prompts read the same `Reader`, so no
  second reader steals keys.
* **What stays the host's:** raw mode (gobble's `golang.org/x/term`), the
  console's VT modes on Windows, the size and SIGWINCH or its polling,
  and when to write bracketed paste on and off. The region gives the
  bytes: `Restore` undoes what it set, the shape `launch.Restorer` takes.
* **`inlinetest.Screen`** is gobble's `termtest.Screen` rebuilt on `x/ansi`
  widths, by grapheme cluster, with a wide character kept whole on a
  re-wrap: pending wrap, scrolling, the moves, `J`, `K`, DECTCEM, mode
  2026, a cursor report answered, and `Resize(columns, Reflow)`. Every
  component's tests, and a program's, draw on it; its text form is what
  the goldens hold.

### D4. `ask`: the question (issue #7)

```go
type Question struct {
	Notes  []string // lines before the title: why only "once" is offered, a limit
	Title  string   // wrapped at spaces, continuation rows indented two
	After  []string // lines after the title: "Patterns: …"
	Rows   []Row
	Cancel Answer   // the key that cancels, and its label: "[c] cancel"
}
type Row struct {
	Label   string   // "Allow", "Reject", or "" for a row of its own
	Answers []Answer
}
type Answer struct {
	ID     string // the caller's
	Key    string // one key, as keymap's notation names it: "1", "f", "y"
	Label  string // "once", "reject and say why"
	Echo   string // the answer in words: "allow for this session"
	Reason bool   // a reason is read after the key, up to enter
}
type Response struct {
	Answer   Answer
	Reason   string
	Canceled bool // the cancel key, ctrl+c, or the end of input
}

func Layout(q Question, width int, m ansi.Method, g glyph.Set) []string // pure
func NewAsker(q Question, o ...Option) *Asker
```

* **`Layout` is a pure function** of the question, the width, the method
  and the glyphs, as gobble's 0029-MADR asks: the title wrapped at spaces,
  a word wider than a row left whole; continuation rows indented two;
  each row's label padded to the widest label; each answer `[k] label`,
  answers two spaces apart; a row that does not fit breaks between
  answers, never inside one, its continuation under its first answer;
  the notes, the lines after the title, the rows, then the prompt
  (`glyph.Set.Prompt`). The caller's words are drawn as given, cleaned:
  the capital and the `?` are the caller's.
* **Keys:** one key, no enter; a key no answer shows is ignored; the cancel
  key, ctrl+c or the end of input cancel. An answer with `Reason` reads a
  line through a one-row `lineedit.Composer`, echoed, with its editing
  keys, bounded at 2048 bytes by default, as gobble's is.
* **The answer stays in the scrollback:** the region commits the question
  and `> ` with the answer's `Echo`, and the reason, so the transcript
  keeps both (`internal/cli/permission.go:223-227`).
* **The TUI's form** is `ask/asktea`: an overlay pane that draws `Layout`
  at the overlay's width and drives the same `Asker`, so a permission
  question reads alike in both modes.
* **What gobble's six questions need,** trust, plan mode, a grant, a
  personality switch, a tool call and the permission question, is this
  one shape: rows of keyed answers, at most one reason.

### D5. `pick`: the picker (issue #8)

```go
type Entry struct {
	ID     string
	Text   string // matched and drawn: "anthropic/claude-sonnet-5-5", a session's title
	Detail string // drawn after the text, not matched: "1M  $3.00/$15.00", "2 hours ago"
}

func NewChooser(entries []Entry, o ...Option) *Chooser // WithHeight (8), WithPrompt, WithMethod, WithIndexAbove
```

* **It filters with `fuzzy`, the palette's matcher:** each keystroke
  compiles one `fuzzy.Matcher` and ranks with `Rank`, best first, stable
  on ties (A4: `Rank` breaks a tie by the shorter key, so the `Chooser`
  scores with `Matcher.Score` and keeps ties in the input order); above
  `WithIndexAbove` entries (5000, as `palette.Files`) it
  narrows with a `fuzzy.Index` first. Matched cells are underlined through
  `fuzzy.Cells`, in the width method, when the host's `Markup` styles.
* **Keys:** typing edits the query in a one-row `lineedit.Composer`; up,
  down, page up and page down move; enter chooses; esc cancels, as #8
  asks. The rows, `WithHeight` tall, the query and a count line are
  redrawn in place by the region; the choice is committed as one line.
* **The data is the caller's:** a program lists its models with their
  context and prices in `Detail`, or its sessions by title and time. It
  lists them all; a picker with providers that stream is the palette's
  job, in the TUI.
* **The TUI's form is a palette provider** (Q5):

  ```go
  func FromEntries(name string, entries []pick.Entry) Provider // in palette
  ```

  Each entry is one hit: its `ID` the entry's, its `Text` matched and
  drawn, its `Detail` in the hint slot, no `Run`, so accepting it sends
  `palette.SelectedMsg` with the hit, as a file hit does, and the program
  acts on the entry's ID. An empty query lists the entries in their order.
  The program gives the same `[]pick.Entry` to a `pick.Chooser` in the
  line mode and to `FromEntries` in the TUI, so one list serves both
  modes, ranked by one matcher. There is no `picktea`: the palette is the
  TUI's picker.

### D6. `lineedit`: the composer (issue #9)

```go
func NewComposer(o ...Option) *Composer
	// WithPrompt, WithContinue, WithHeader, WithHistory, WithCompleter,
	// WithKeymap, WithLimit (4096 runes), WithPaste(PasteInsert|PasteSubmit),
	// WithExternalEditor, WithMethod, WithGlyphs

type History interface { // x/term's shape, so its and gobble's plug in
	Add(entry string)
	Len() int
	At(i int) string // 0 is the newest
}
type MemHistory struct{ /* bounded, 1000 by default; consecutive duplicates dropped */ }
func (h *MemHistory) WriteTo(w io.Writer) (int64, error)  // one entry per line, \n and \ escaped
func (h *MemHistory) ReadFrom(r io.Reader) (int64, error)

type Completer interface{ Complete(line string, pos int) (string, int, bool) }
type CompleterFunc func(line string, pos int) (string, int, bool)
```

* **The buffer is grapheme clusters,** under the width method, so a
  combining mark, a ZWJ sequence or a flag is one cursor step and its true
  width: the defect gobble's 0032-MADR A1 names, fixed.
* **The layout is gobble's** (0027-MADR D2): whole words to the width,
  breaking only at spaces, a wider word left whole (A5 item 5: broken at the
  width, as gobble's code and A2 item 5 break it); continuation rows
  indented by the prompt's width; a cursor past the last column on a new
  row; drawn through the `Region`, so its diff, its budget and its re-wrap
  apply.
* **Keys are commands in the program's one keymap,**
  `lineedit.DefaultRules()`, in the scope `kind:composer`, dispatched by
  `keymap`'s `Dispatcher`, as the palette's are: gobble's bindings above
  as the defaults, and a user's `keybindings.json` rebinds them. Without
  `WithKeymap` the composer builds its defaults alone. Shift+enter, where
  the terminal reports it (Kitty or modifyOtherKeys, by `termkey`),
  inserts a newline; so does a trailing `\` before enter.
* **Events:** `EditWrite`; `EditSubmit`, with the text; `EditEOF`, ctrl+d
  on an empty line; `EditInterrupt`, ctrl+c, whose policy, such as
  gobble's second press within 2 s, is the host's; with
  `WithExternalEditor`, `EditExternal`, with the text, for the host to
  edit in a process it starts and give back with `SetText`. The library
  reads no `EDITOR` and runs nothing (AGENTS.md, API conventions 9).
* **History** is the `History` interface, x/term's shape; `MemHistory`
  is gobble's rules in memory, and its `WriteTo` and `ReadFrom` its file
  format, so the file, its path, its rename and its mode stay the host's.
* **Paste:** `PasteInsert`, the default, puts a bracketed paste into the
  buffer as text, editable; `PasteSubmit` submits it whole, gobble's
  choice (its 0027-MADR Q2=A).
* **The TUI's form** is `lineedit/lineedittea`: a workspace pane over the
  same `Composer`, its rows drawn into the pane, its keys through
  `keymaptea`, so the TUI's composer and the CLI's edit alike.
* **Not in this record:** undo and a kill ring, which gobble has not
  either; each is an addition later, with its keys in `DefaultRules`.

### D7. `stream`: a Charm-free core, amending 0009

* **0009's `stream` becomes a core.** Its `Doc`, `Renderer`,
  `RendererFunc` and `Plain` stay in `stream`, which imports `safetext`,
  `internal/enum`, `internal/sanitize` and `x/ansi` alone. What names
  Bubble Tea, `Buffer`, `Doc.Pull` and `Pane`, moves to `stream/streamtea`,
  under 0020-MADR D2's split rule. Nothing is shipped, so nothing breaks.
  [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
  gains an amendment saying so, and its PLAN is amended before it runs.
* **One block scanner, two consumers.** 0009 §4's scanner, the open fence,
  the containers, the HTML block, the table, is `stream.Scanner`, and
  serves two forms of output:
  * `Doc`, block-stable, re-rendered by a `Renderer` that may reflow a
    whole block (glamour, through `stream/glamourmd`): the TUI's pane;
  * `Printer`, line-stable and append-only, for a scrollback that cannot
    be redrawn: the line mode.
* **`Printer` prints each source line once it is stable,** as gobble's
  `render.Stream` does: a complete line outside a table and outside an
  unclosed inline construct, cleaned by a `safetext.Sanitizer`, styled by
  its `Markup`, wrapped at whole words to the width, and written once. A
  table is held until a blank line ends it, then laid out; a heading until
  its line ends; a fence's lines print as they complete.
* **Styling keeps every marker** (gobble's `mdstyle.go`): headings bold,
  fences dim with the language in a colour, inline code dim, list and
  quote markers in a colour, emphasis by its markers. `Markup` is data: a
  `Style` per role, each an SGR attribute set (bold, dim, italic,
  underline, and a colour of the eight, or none). The zero `Markup` styles
  nothing, and the output is the input, byte for byte. *Amended
  2026-10-28 by A7: but for a table's layout, which is not styling;
  with `WithoutTableLayout` too, byte for byte.*
* **Wrapping is gobble's** (0028-MADR D5 and D7, amended): after styling;
  prose at spaces, a wider word whole; fences, indented code, table rows
  and headings unwrapped; list continuations under their text; quotes
  repeating `>`; no row beginning with a word that would open a block;
  escapes zero-width and kept with their word. The output is the same for
  any split of the input, a property test as gobble's `TestWrapAnySplit`
  is.

### D8. Markdown in the line mode: the repository's own printer

* **Not glamour,** which links Lip Gloss: Charm beyond `x/ansi`.
* **Not goldmark.** goldmark (v1.8.2, MIT, no requirements of its own)
  parses a whole document into a tree. A line-stable printer cannot wait
  for a document, and a renderer that reflows restyles text the user has
  already read; gobble's users see the markers kept (its 0028-MADR). It
  would also be a module this record must name, for a parse the scanner
  already does to the depth the printer needs.
* **The printer is the scanner and a styler,** the size of gobble's
  `mdstream.go`, `mdstyle.go` and `wrap.go`, about 1,000 lines, built
  here once, for both repositories.

### D9. `transcript`: the conversation, reprinted on a resize (issue #10)

```go
func NewTranscript(o ...Option) *Transcript // WithMaxBytes (4 MiB), WithMaxRows (10 000), WithSettle (150 ms), WithMethod

func (t *Transcript) AddText(s string)                   // as written, cleaned
func (t *Transcript) AddDrawn(f func(width int) string)  // drawn anew at each width: help, a log record
func (t *Transcript) AddReply(p *stream.Printer)         // a reply, kept as its cleaned source
func (t *Transcript) Resize(width int, now time.Time)
func (t *Transcript) Timeout() (time.Time, bool)
func (t *Transcript) HandleTimeout(now time.Time, idle bool) string // the reprint, once resizing has settled and the host is idle
```

* **Bounded in bytes:** past `WithMaxBytes`, the oldest pieces go, as in
  gobble (`transcript.go:31-80`).
* **A resize reprints once it settles:** the last `Resize` sets a deadline
  `WithSettle` later; at it, when the host says the output is idle (at a
  line start, nothing held), the transcript gives `CSI ?2026h CSI 2J CSI
  3J CSI H`, the newest pieces back to `WithMaxRows` rows drawn at the
  new width, and `CSI ?2026l`, after the caller's notice line when it
  cut, as gobble writes it (`view.go:444-487`). The host then redraws its
  region.
* **It clears the terminal's scrollback,** and with it what came before
  the program, as gobble's does (0030-MADR:139): the price of a reprint
  that lines up. A terminal's own re-wrap of a styled table, a diff or a
  code block cannot line up.
* *Amended 2026-10-28 by A8 items 2 to 4:* a reply is drawn again by
  `stream.Printer.At`; `WithSynchronized` and `WithNoticeStyle` join the
  options; a reprint still due after the settle waits for the host's
  next `HandleTimeout(now, true)`.

### D10. What stays the host's, and what is not built

* **The host's:** raw mode and its restore; Windows console modes; the
  terminal's size and the resize signal or poll; bracketed paste on and
  off, from the region's bytes; the external editor's process; the
  history file; the colour and Unicode decision, through `termcap`; the
  cap on the width; a second ctrl+c's meaning.
* **Not built here:** a line-mode multi-select; mouse input in the line
  mode; Windows' win32-input-mode, which `termkey` reads as an unknown
  sequence (0020-MADR D7); undo and a kill ring (D6); a `picktea` (D5).

### D11. Tests and gates

* **Every engine is a function of its inputs and the time:** each test
  feeds bytes and times, and reads the events.
* **Goldens on `inlinetest.Screen`,** across {colour, no colour} × {UTF-8,
  ASCII} × widths 60 and 100, and gobble's 177 for the question
  (`0029-PLAN:49-60`), each line within its width under the method.
* **Fuzz targets:** `Layout` never splits an answer and never writes a row
  wider than the width; the composer and the picker never panic on any
  input stream; the printer gives the same output for any split of its
  input; `MemHistory.ReadFrom` writes back what it read.
* **Budgets:** a one-key change at most the changed text plus 40 bytes;
  per-key benchmarks with recorded baselines, as gobble's 0030-MADR D5.
  *Amended 2026-10-29 by A9 item 5:* the reprint and a streamed reply
  have budgets of allocations and bytes, gobble's numbers, tested.
* **Re-wrap tests** for each of the three kinds, with and without a
  cursor report, and keys typed while the report is awaited.
* **The Windows edge:** a test that `Reader` holds one read in flight at
  most and never calls `Close`, as gobble's `TestNoCloseOnStdin` does.
* **The gate:** each new core joins `TestCoresAreCharmFree` in the commit
  that builds it; the `<pkg>tea` packages are not cores.

### D12. Order and releases

* **In gobble's order,** each a step of its own: (1) `inline`,
  `inlinetest` and `termcap.ReplyCursorPosition`; (2) `ask` and
  `ask/asktea`, gobble's first need; (3) `pick` and
  `palette.FromEntries`; (4) `lineedit` and
  `lineedit/lineedittea`; (5) `stream`'s core, `Printer` and `Markup`,
  with 0009's amendment; (6) `transcript`.
* **Additions only,** so each release is a minor version of `v2`, with
  `make apicheck` reporting 0 incompatible changes.
* **A release after each of steps 2, 3, 4 and 6,** `v2.2.0` to `v2.5.0`,
  so gobble takes each component as it lands (Q3).
* *Note, 2026-10-25, by
  [0024-MADR-opt-in-and-opt-out-coverage.md](0024-MADR-opt-in-and-opt-out-coverage.md)
  Q5:* that record's first PLAN releases `v2.2.0` before this record's
  first step, so these four releases are each the next minor version
  after it, `v2.3.0` to `v2.6.0` if nothing else is released between.

### D13. Defaults and their controls

*Added 2026-10-25, for
[0024-MADR-opt-in-and-opt-out-coverage.md](0024-MADR-opt-in-and-opt-out-coverage.md)
D8, before this record is accepted.* Every component follows API
convention 12 from its first step: each default it applies is listed in
`docs/guides/defaults.md` with its control before it is built. Three gain
a control beyond D3 to D9:

* `inline.WithoutCursorQuery()`: a resize sends no `CSI 6 n`, and the
  region redraws by `WithReflow`'s kind alone;
* `transcript.WithoutScrollbackClear()`: a reprint writes `CSI 2 J`
  without `CSI 3 J`, keeping the terminal's scrollback, at the cost of
  the old rows above it;
* `WithRules(safetext.Rules)` on each component that cleans text, never
  weaker than `safetext.Line`, the floor of 0024-MADR D2.

The text each component shows, such as the picker's count and the
question's prompt, has a message ID and an English default, and comes
from the program's catalogue through `WithCatalog` (0024-MADR D9). The
floors stay: cleaning, the input bounds, and the protocol of the cursor
report.

### Consequences

* Good, because gobble's line-mode CLI gets all four components with
  clean closures, and can drop its own: the editor (about 1,800 lines),
  the Markdown streaming, styling and wrapping of `render` (`mdstream.go`,
  `mdstyle.go`, `wrap.go`, `table.go`), and the transcript, and need not
  build the question.
* Good, because every behaviour gobble measured becomes a deterministic
  test here, on a screen that re-wraps three ways.
* Good, because the TUI and the line mode share the question, the
  composer, a picker's entries and the Markdown scanner.
* Neutral, because `palette` imports `pick` for `FromEntries`; `pick` is a
  core, so the palette's closure gains no Charm module.
* Good, because widths are by grapheme cluster, in the terminal's method,
  where gobble's are not.
* Good, because the library still owns no terminal: no raw mode, no
  stdin, no environment, no process.
* Neutral, because 0009's `stream` changes its package graph before it is
  built; its design otherwise stands.
* Bad, because a host writes a loop: read, decode, feed an engine, drain
  its events, write, and arm a timer for its deadline. Each package's
  documentation and an example show that loop.
* Bad, because seven packages join the cores, and the conformance test
  lists them.
* Bad, because a reprint clears the terminal's scrollback.
* Bad, because a cursor report on the first row must be read by the region
  while it waits for one, since it is a modified F3 to a decoder.

### Confirmation

* `TestCoresAreCharmFree` passes with each new core listed, on Linux,
  macOS and Windows, and fails on a scratch copy that imports Bubble Tea
  into one.
* The goldens, fuzz targets, budgets and re-wrap tests of D11 pass, each
  seen failing on a scratch copy first.
* gobble's issue #7 layout, drawn by `ask.Layout` at 60, 100 and 177
  columns, matches its 0029-MADR D1 sketch.
* A consumer smoke test in a scratch module asks a question, picks an
  entry, edits and submits a line, and streams a reply, on a fake
  terminal, linking no Charm module but `x/ansi`.
* `make apicheck` reports 0 incompatible changes at each release.

## Pros and Cons of the Options

### A. I/O-free engines, a package each, on one live region

* Good, because the host owns the reader, so one reader serves the
  editor, a question asked mid-turn and a sign-in prompt, as gobble's
  `Reader` does.
* Good, because a streaming reply prints above the composer from the same
  loop, with no lock (x/term's `Write`, done as bytes).
* Good, because it is 0020-MADR A2's form, already used by `termcap.Probe`.
* Bad, because it is the most code, and each host writes its loop.

### B. Blocking components that own a reader

* Good, because `ReadLine()` is the simplest call there is.
* Bad, because a component holds the terminal for the length of a prompt:
  output from another goroutine needs a lock and a redraw inside it, as
  x/term's `Terminal.Write` takes `t.lock`.
* Bad, because a reader that reads before raw mode is what returned 379
  bytes on Windows; the fix was reading only when asked.
* Bad, because its tests need a goroutine and a pipe for every case.

### C. Bubble Tea drawn inline

* Good, because huh and promptkit already draw questions and lists so.
* Bad, because the closure holds Bubble Tea, `colorprofile` and
  `ultraviolet`: what gobble's boundary forbids.

### D. Wrap an existing Go line editor

* Good, because liner and readline exist and have users.
* Bad, because each owns stdin and raw mode (`NewLiner` "sets the terminal
  into raw mode"), measures without grapheme clusters, decodes keys beside
  `termkey`, and is a module this record would have to name.

### E. One package for all four

* Good, because there is one import path and one document.
* Bad, because a program that asks one question links the picker's
  `fuzzy`, the transcript's `stream` and the composer's keymap.
* Bad, because the types would need prefixes to stay apart in one
  package (`QuestionEvent`, `PickerEvent`).

### F. Change nothing

* Good, because nothing is built.
* Bad, because gobble's 0032-MADR D5 waits for this repository, and its
  0029-PLAN's P2 is stopped for #7.

## Owner questions

*Answered 2026-10-25,* each picked from three options: Q1 "A package
each (Recommended)"; Q2 "Own line-stable printer (Recommended)"; Q3 "A
minor per component (Recommended)"; Q4 "The program's keymap
(Recommended)"; Q5 "A palette provider"; Q6 "Amend 0009 here
(Recommended)"; Q7 "Insert, editable (Recommended)". Q5 is not the
recommendation: D5's TUI form is `palette.FromEntries`, not the palette
with a `Static` provider and no TUI form of `pick`.

* **Q1. The packages.** Recommended: D1's, one package per component on a
  shared `inline`. The alternative is option E, one package.
* **Q2. Markdown in the line mode.** Recommended: D8, the repository's own
  line-stable printer that keeps the markers. The alternatives are a
  goldmark-based renderer, which adds a module and restyles text, or none,
  leaving gobble's `render` in gobble.
* **Q3. Releases.** Recommended: a minor release after each of D12's
  steps 2, 3, 4 and 6. The alternative is one release when all six steps
  are done.
* **Q4. The composer's keys.** Recommended: commands in the program's one
  keymap, `lineedit.DefaultRules()`, as the palette's are. The
  alternative is a fixed set of keys, as gobble has today.
* **Q5. The picker in the TUI.** Recommended: the palette, with no
  `picktea`. The alternatives are a `pick/picktea` pane, or a palette
  provider over `pick`'s entries, `palette.FromEntries`, which the owner
  chose.
* **Q6. 0009's split.** Recommended: amend
  [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
  as D7 says, in this record's first PLAN. The alternative is a record of
  its own for the split, before step 5.
* **Q7. A bracketed paste in the composer.** Recommended: inserted as
  text by default, with gobble's submit-whole an option. The alternative
  is gobble's behaviour as the default.

## Amendments

### A1 (2026-10-27): the report's wait, the region's height, and the test screen's import

*Status: accepted (2026-10-27),* the owner's answers to
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 1's deviations D2 to D4, each "(Recommended)", found before that
step's code:

1. **D3's wait for the cursor report has a control,**
   `inline.WithReportWait(d time.Duration)`; zero or negative keeps
   `inline.DefaultReportWait`, 150 ms. D3 says the region waits "by
   default 150 ms", and API convention 12 asks for a `With…` option for
   a timeout a program may need otherwise, as over a slow link. D13's
   controls gain it.
2. **D3's height can change after construction:** `Region.SetHeight(rows
   int)` beside `WithHeight`, since `Resize` takes the width alone and a
   terminal's height changes too; 0 draws every row.
3. **`inline/inlinetest` imports `inline`,** in D1's table, so that
   `Screen.Resize(columns int, how inline.Reflow)` names the one `Reflow`;
   `inline`'s own tests that draw on a `Screen` are external tests. Its
   closure stays the cores' and `x/ansi`.

### A2 (2026-10-27): a line editor before `lineedit`, the host's cancel and region, the Asker's view, a word wider than a row, and the events' names

*Status: accepted (2026-10-27),* the owner's answers to
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 2's deviations D5 to D10, each "(Recommended)", found before that
step's code:

1. **`ask` and `pick` read a line through `internal/lineinput`** until
   `lineedit` is built: one row of grapheme clusters under the width
   method, bounded in bytes, with a fixed set of editing keys, drawn as a
   window around the cursor. It is a core. D1's import column for `ask`
   and `pick` reads `internal/lineinput` in place of `lineedit` until
   [0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
   Step 4, which builds `lineedit.Composer` on it and moves `ask` and
   `pick` to the `Composer` with no exported change; D4's reason keys are
   that fixed set until then, and the keymap's after.
2. **D2's contract gains `Cancel()`:** the host ends the engine's wait, at
   the end of its input or for a reason of its own, such as a request
   withdrawn. `ask` and `pick` end canceled, their echo committed as for
   the cancel key; `lineedit` sends `EditEOF`.
3. **An Asker draws in the host's region,** `ask.WithRegion(*inline.Region)`,
   which the host shares among its components and sets `inline`'s options
   on; one engine draws a region at a time. Without it the Asker builds
   `inline.NewRegion(inline.WithMethod(m))`.
4. **`Asker.View(width int) (rows []string, row, col int)`** gives the
   rows the Asker draws, and the cursor, at a width, without a change. The
   line mode draws through it, and `ask/asktea`'s `Dialog`, the overlay's
   pane, draws it at the overlay's width; the answer reaches the program
   as `asktea.AnsweredMsg`.
5. **A word wider than its whole row is broken at the width,** by
   grapheme cluster, its pieces on rows of their own with the row's
   continuation indent, as a terminal hard-wraps it; an answer too wide
   for any row is on a row of its own, broken alike. Every word and answer
   that fits a row stays whole. D4's "left whole" and D11's "never splits
   an answer" read so, and `Layout` never writes a row wider than the
   width.
6. **Each engine's event is named for its engine,** since revive's
   `exported` rule refuses a type that repeats its package's name, and
   `ask.Event` is `keymap`'s: `ask.AskerEvent` and `AskerEventKind` in
   place of D4's `AskEvent` and `AskEventKind`; `pick.ChooserEvent` and
   `ChooserEventKind` in place of D5's `PickEvent` and `PickEventKind`;
   `lineedit.ComposerEvent` and `ComposerEventKind` in place of D6's
   `EditEvent` and `EditEventKind`. D2's examples read so. The kinds'
   constants, such as `AskWrite` and `EditSubmit`, and `NextEvent` keep
   their names.

### A3 (2026-10-27): what the prior art and the standards say, for Steps 3 to 6

*Status: accepted (2026-10-27),* with the owner's answers to Q8 to Q11
below, each "(Recommended)". The owner asked, after `v2.3.0`:
"proceed to step 3. do some research on the best, most idiomatic,
compatible, standards adherent, and flexible way to implement these units
of work and improve the grounding in facts and research to update the
madr." Every fact below was read on 2026-10-27 from the primary source at
the tag, commit or revision named; a fact read from source code names the
file. The two that decide the most, CommonMark's setext rule and fzf's
default keys, were read a second time, verbatim, from the published page.
The repository's own facts were read at `699ecb4`.

#### A3.1 The picker (D5, issue #8)

**This repository.**

* `fuzzy.Matcher.Rank` "scores items by key and returns the best limit,
  best first, stable on ties", and "an empty query … matches every
  candidate with the zero Score": so an empty query keeps the input order
  with no code of its own (`go doc ./fuzzy`). *Corrected by A4 item 1:*
  its order breaks a tie by the shorter key before the input order, so
  "stable on ties" holds only between keys of one length.
* `fuzzy.Index` "is lossy on purpose", and `Candidates` yields "at most
  the maximum", `DefaultMaxCandidates`, 2000; a candidate that equals the
  query, or whose word or path element starts with it, is yielded first.
* `palette`'s engine keeps hits "by score, then by arrival"
  (`palette/engine.go:446-456`), keeps at most its limit, 200 by default,
  and keeps one hit per `ID` (`:420-444`). One provider that yields its
  entries in order is therefore ranked as `Rank` ranks them, through the
  first 200, when the IDs are distinct. *Corrected by A4 item 1:* as its
  matches are ranked by grade and value, ties in the input order, which
  `Rank` is not.
* `palette.Static` is the template for `FromEntries`: `Search` scores
  `Text` with the shared matcher, `Discover` lists every item, first best,
  so they keep their order (`palette/static.go:38-61`).
* The palette draws a row as `glyph.Set.Focus` on the selected row, spaces
  on the others, the text cut to fit, and the hint right-aligned; the hint
  is dropped when it would leave the text fewer than 8 cells (`minText`,
  `palette/view.go:16-18`, `:80-149`). Its count reads "3 of 40", or "no
  results" (`:201-238`).
* `palette.DefaultRules` binds, in `kind:picker`, up and ctrl+p, down and
  ctrl+n, pgup, pgdown, enter and tab; esc stays the workspace's; "None is
  ctrl+c, which is the program's" (`palette/keys.go:45-72`).
* `keymap.Dispatcher` is "a state machine driven by Key and Expire, with
  no goroutine and no clock"; an `Outcome`'s `Wait` is what the host times
  (`go doc ./keymap Dispatcher`, `Outcome`). An engine can fold that wait
  into its own `Timeout`.

**Prior art,** each at its latest tag: fzf v0.74.5, gum v2.0.2, huh
v2.0.3, promptkit v0.12.0, inquire v0.9.4, survey v2.3.7 (archived: "This
project is no longer maintained").

* **fzf draws inline below the cursor** with `--height`: "Display fzf
  window below the cursor with the given height instead of using the full
  screen" (`man/man1/fzf.1`). It takes the alternate screen only when full
  screen (`src/tui/light.go`, `Init`), makes room with line feeds and moves
  back by relative moves, wraps each frame in mode 2026, queries the cursor
  (`CSI 6 n`) only for its starting column and the mouse, and on a resize
  clears its rows and draws again (`src/terminal.go`, `fullRedraw`). On
  exit it erases its rows and prints the choice to stdout.
* **Layout.** `--layout`: "default: Display from the bottom of the
  screen; reverse: Display from the top of the screen; reverse-list:
  Display from the top of the screen, prompt at the bottom" (man page,
  read twice). Its README's inline example, and its shell bindings, use
  `--height … --layout reverse`: the query on top, the list below.
* **Keys.** fzf: up "ctrl-k up", down "ctrl-j down", page-up "pgup",
  page-down "pgdn", accept "enter double-click", abort "ctrl-c ctrl-g
  ctrl-q esc" (man page, AVAILABLE ACTIONS, read twice); ctrl-n and ctrl-p
  are `down-match` and `up-match`, the same as down and up unless raw mode
  is on (`src/terminal.go`, `defaultKeymap`). gum filter: down "down
  ctrl+j ctrl+n", up "up ctrl+k ctrl+p", esc quits, ctrl+c aborts, enter
  and ctrl+q submit. huh: up "up k ctrl+k ctrl+p", down "down j ctrl+j
  ctrl+n", j and k ignored while filtering. inquire: Up or Ctrl-P, Down or
  Ctrl-N, PageUp, PageDown, Home, End; Enter or Ctrl-J submits; Esc,
  Ctrl-G or Ctrl-D cancel, Ctrl-C interrupts. The keys every one of them
  shares are up, down, ctrl+p, ctrl+n, pgup, pgdown, enter, esc and
  ctrl+c; ctrl+j (down in fzf, gum and huh, submit in inquire) and ctrl+q
  (abort in fzf, submit in gum) disagree.
* **Height.** gum choose `--height` 10; huh `defaultHeight = 10`; fzf
  `--min-height` 10; survey and inquire a page of 7.
* **Order.** fzf's empty query keeps the input order (`src/matcher.go`,
  `passMerger`), and its ties break by length, then index ("Default is
  length (or equivalently length,index)"). inquire sorts with
  `sort_unstable_by_key`, so its ties are not in input order.
* **Fields shown but not matched:** fzf's `--nth` limits the search
  scope, "because fzf doesn't allow searching against the hidden fields":
  D5's `Detail`.
* **What is left after the answer.** fzf and gum leave nothing and print
  to stdout; survey leaves "? <message> <answer>"; promptkit's result
  template is the prompt and the choice; inquire leaves its prefix, the
  prompt and the answer, and "<canceled>" on a cancel.
* **Cancel and interrupt.** fzf exits 130 for both ctrl-c and esc; gum
  130 for ctrl+c and 1 for esc; inquire reports `OperationCanceled` and
  `OperationInterrupted` apart.
* **Scrolling.** fzf: "Number of screen lines to keep above or below when
  scrolling to the top or to the bottom (default: 3)", capped at half the
  height (`constrain`). The palette keeps the selected row on the screen,
  with no margin (`palette/picker.go:451-466`).
* **Large lists.** fzf stores items in chunks of 1024, scans them on
  every CPU, abandons a scan when the query changes, and caches each
  chunk's result for the next, longer query (`src/constants.go`,
  `matcher.go`, `cache.go`). None of the tools states a size at which an
  index pays: a threshold here is this repository's own number.
* **Colour never carries the selection alone.** WCAG 2.2 SC 1.4.1
  (Recommendation, 2024-12-12): "Color is not used as the only visual
  means of conveying information, indicating an action, prompting a
  response, or distinguishing a visual element." Every tool above draws
  a pointer glyph on the selected row: fzf `▌` or `>`, gum `•`, survey
  `>`, promptkit `▸`, inquire `>`. no-color.org: `NO_COLOR` "prevents the
  addition of ANSI color", and does not turn off bold or underline.
* **A prompt is never required:** "Only use prompts or interactive
  elements if stdin is an interactive terminal" and "Never require a
  prompt" (clig.dev); fzf's `--filter` is its non-interactive mode. Which
  mode to run is the host's, as D10 says.

#### A3.2 The composer (D6, issue #9)

* **GNU Readline 8.3's emacs keys** (its user manual): C-a, C-e, C-f,
  C-b; M-f and M-b, where "Words are composed of letters and digits"; M-d
  and M-DEL by the same words; C-w, unix-word-rubout, "using white space
  as a word boundary"; C-k, C-u, C-d; C-p and C-n; C-t; C-l; undo "C-_ or
  C-x C-u", "separately remembered for each line"; C-y and M-y on a kill
  ring. golang.org/x/term v0.47.0's `Terminal` binds a subset (^A ^B ^E ^F
  ^H ^K ^L ^T ^W ^N ^P ^U ^D), with no undo and no kill ring.
* **Words.** UAX #29 (Unicode 18.0.0, revision 49, 2026-09-01) names
  "cursor movement ('move to next word' control-arrow keys)" among its
  uses, and lets implementations "tailor" it; zsh's `WORDCHARS` and
  PSReadLine's `WordDelimiters` each tailor a word by a character set.
  Readline's "letters and digits" is the tailoring a standard-library
  test of a cluster's first rune (`unicode.IsLetter`, `unicode.IsDigit`)
  implements, with no module beyond the root's; a UAX #29 segmenter would
  be a module of its own (`github.com/clipperhouse/uax29/v2`, which
  `x/ansi` requires, is not on the root's list).
* **Bracketed paste** (xterm ctlseqs, patch #411, 2026-08-23): `CSI 2 0 0
  ~`, the text, `CSI 2 0 1 ~`. xterm's own note: "The terminating marker
  might be embedded in the pasted text"; KDE bug 392554 and oss-security
  (2023-10-19) show the end marker injected where a terminal did not
  filter it, and a 2025 suckless thread reports st still unfiltered.
  Readline's bracketed paste inserts each character "as if each one was
  bound to self-insert"; x/term returns `ErrPasteIndicator` so a program
  "may wish to interpret pasted data more literally". The application
  cleans the paste itself, whatever the terminal does.
* **Shift+Enter.** Under the Kitty protocol's flag 1 a modified Enter is
  `CSI 13 ; 2 u` (kitty `key_encoding.c`, master 4f7dea34); in the legacy
  encoding Enter and Shift+Enter are both `0x0d`, and Alt+Enter is `ESC
  CR`. A newline key that every terminal can send is needed beside
  Shift+Enter: alt+enter, as fish binds it, or a trailing `\`, which D6
  has.
* **History.** x/term v0.47.0's `History`: `Add(entry string)`, "a new,
  most recent entry", "It is allowed to drop any entry, including the
  entry being added"; `Len() int`; `At(idx int) string`, "Index 0 is the
  most-recently added entry and index Len()-1 is the least-recently added
  entry. If index is < 0 or >= Len(), it panics." D6's interface is that
  shape. Bounds elsewhere: x/term 100, linenoise 100, bash's `HISTSIZE`
  500, PSReadLine 4096 (its `Cmdlets.cs`); duplicates: bash's
  `ignoredups`, zsh's `HIST_IGNORE_DUPS` and linenoise drop a line equal
  to the previous one. fish writes an entry's newlines as `\n` and its
  backslashes as `\\` (`src/history/yaml_backend.rs`): D6's `MemHistory`
  file form.
* **Undo and a kill ring** are Readline's, zsh's (a ring of eight), fish's
  and PSReadLine's (ten), and not x/term's: D6 leaving them out has
  precedent.

#### A3.3 The streamed reply and the transcript (D7 to D9, issue #10)

* **CommonMark 0.31.2 (2024-01-28)** decides some lines by a later one:
  * §4.3: "A setext heading consists of one or more lines of text, not
    interrupted by a blank line, … followed by a setext heading
    underline"; example 95, `Foo`, `Bar`, `---`, is `<h2>Foo\nBar</h2>`
    (read twice, verbatim). §4.1: where a line of dashes could be a
    thematic break or an underline, "the interpretation as a setext
    heading takes precedence". "The setext heading underline cannot be a
    lazy continuation line in a list item or block quote" (examples 93,
    94).
  * §4.7: "Link reference definitions can come either before or after the
    links that use them."
  * §5.3: "A list is loose if any of its constituent list items are
    separated by blank lines": known only after the items.
  * The appendix's parsing strategy has two phases, the blocks and then
    the inlines, "Once all of the input has been parsed"; emphasis (example
    405) and code spans (example 337) may cross a line.
  * What is known on its own line: a fence's opening and its close, "of
    the same type as the code block began with … and with at least as many
    backticks or tildes" (§4.5); an unclosed fence runs to the end; an
    indented code block "cannot interrupt a paragraph" (§4.4); a lazy
    continuation line is read by looking back at the open paragraph
    (§5.1, example 233); an HTML block's start (§4.6).
* **GFM 0.29-gfm (2019-04-06) §4.10:** a table is "a single header row, a
  delimiter row separating the header from the data, and zero or more data
  rows"; "The header row must match the delimiter row in the number of
  cells. If not, a table will not be recognized"; "The table is broken at
  the first empty line, or beginning of another block-level structure"; a
  row of fewer cells is padded and a longer one cut. A header is known
  only when the second line arrives; cmark-gfm 0.29.0.gfm.13
  (`extensions/table.c`) takes the delimiter row only while a paragraph is
  open.
* **Prior art.** Codex CLI rust-v0.164.0: `MarkdownStreamCollector`
  "exposes a commit boundary at each newline", and a table's header "no
  delimiter row has followed yet. Hold back" (`codex-rs/tui/src/
  markdown_stream.rs`, `streaming/table_holdback.rs`), with a re-rendered
  tail. aider v0.86.0 re-renders the last six lines and prints the lines
  above once (`aider/mdstream.py`). gemini-cli v0.65.0 splits at the last
  blank line outside a fence (`utils/markdownUtilities.ts`). Streamdown
  v0.36.9 holds one chunk and rewrites it as a heading when an underline
  follows (`streamdown/sdlib.py`). mods v1.8.1 (archived) re-renders the
  whole reply on each chunk. None prints a setext heading's text before it
  knows the next line, except by re-rendering it.
* **The reprint.** xterm ctlseqs: "CSI Ps J … Ps = 3 ⇒ Erase Saved
  Lines". VTE and Windows Terminal v1.25.2733.0 move the page into the
  scrollback on `CSI 2 J` (`vteseq.cc`, `adaptDispatch.cpp`), so `CSI 2 J`
  comes before `CSI 3 J`, as D9 orders them. kitty v0.49.2 implements `CSI
  3 J` but leaves `E3` out of its terminfo on purpose; iTerm2 may ask the
  user before it clears. Codex resets with `CSI r`, `SGR 0`, `CSI H`,
  `CSI 2 J`, `CSI 3 J`, `CSI H`, after 75 ms of quiet; gemini-cli after
  100 ms. The synchronized-output spec (contour vt-extensions, 05050a1):
  `CSI ? 2026 h` and `l`, detected by DECRQM `CSI ? 2026 $ p`.
* **OSC 8** (the specification gist, 2026-10-20): `OSC 8 ; params ; URI
  ST`; "Simple utilities that 'just' print stuff on their standard output
  should not assign an id."
* **UAX #14** (revision 57, 2026-09-01) §3.1: "Western: spaces and hyphens
  are used to determine breaks; East Asian: lines can break anywhere,
  unless prohibited; South East Asian: line breaks require morphological
  analysis." Breaking at spaces alone, as gobble's 0028-MADR does, leaves
  a run of CJK or Thai text one word.

#### A3.4 What changes

For Step 3, by the owner's answers to Q8 and Q9 below:

1. **The layout is fzf's reverse layout:** the query row first, the prompt
   (`WithPrompt`, `glyph.Set.Prompt` by default) before it and the count
   right-aligned after it, dropped when it would leave the query fewer
   than 8 cells; then the rows, each the palette's: `glyph.Set.Focus` on
   the selected row, the `Text` cut to fit, the `Detail` right-aligned,
   dropped below 8 cells of text. The list is as tall as `WithHeight`
   rows, or the entries when fewer, for the whole question, so the region
   does not change height as the query narrows.
2. **The default height is 10 rows** (Q8), fewer when there are fewer
   entries, `WithHeight` to change it.
3. **The keys are commands in the program's keymap** (Q9):
   `pick.DefaultRules()` binds `pick.select.prev` to up and ctrl+p,
   `pick.select.next` to down and ctrl+n, `pick.page.prev` and
   `pick.page.next` to pgup and pgdown, `pick.accept` to enter, and
   `pick.cancel` to esc and ctrl+g, in `kind:chooser`. Esc and ctrl+g
   cancel, as in fzf and inquire, and the host's `Cancel` (A2 item 2)
   does too; ctrl+c cancels outside the keymap, as in `ask`, since the
   palette's bindings leave it "the program's"; ctrl+j and ctrl+q are left
   unbound, since the tools disagree on them. The query is edited by `internal/lineinput`'s fixed keys (A2 item
   1), its bound 1024 bytes, room for `fuzzy.MaxQuery` clusters of one
   four-byte rune each.
4. **The order is the matcher's:** `Rank`, its grade, its value, then the
   input order (A4 item 2: by `Matcher.Score` and a stable sort, since
   `Rank` puts the shorter key first); fzf's length tie-break is not taken, so `pick` and the
   palette order one list alike. An empty query keeps the input order. The
   selection returns to the first row when the query changes, as in fzf
   and the palette, and the window keeps it on the screen, as the
   palette's does.
5. **An index above 5000 entries,** `palette.DefaultIndexAbove`, this
   repository's number; Step 3 measures a keystroke at 5000 and at 100,000
   entries and records it. Under the index the count is of the hits it
   yields; when that is `fuzzy.DefaultMaxCandidates`, the count says so
   ("40 of 2000+").
6. **One line is left in the scrollback,** as #8 asks and the prompt
   libraries do: the prompt and the chosen entry's `Text`, or the prompt
   and the catalogue's word for a cancel. `ChooserEvent` carries the
   `Entry` or `Canceled`, as `ask.Response` does.
7. **No match is highlighted in Step 3.** D5 underlines matched cells
   "when the host's Markup styles", and `stream.Markup` is Step 5's: the
   selected row is marked by its glyph, never by colour alone. Step 5
   adds `pick.WithMarkup`.
8. **`FromEntries` is `Static`'s form over `pick.Entry`:** `Search` scores
   `Text` alone, `Discover` lists the entries in order, each hit's `Hint`
   the entry's `Detail` and `Run` nil, so accepting sends `SelectedMsg`.
   The equal ranking D5 promises is for one provider, distinct IDs, and
   the palette's limit: the first 200 hits.

For Steps 4 to 6, refinements the evidence decides, and the questions that
remain, each put to the owner before its step:

9. **`lineedit`'s words:** alt+f, alt+b, alt+d and alt+backspace move and
   delete by Readline's words, letters and digits by a cluster's first
   rune; ctrl+w deletes to whitespace, as Readline's unix-word-rubout and
   x/term do. gobble's space-delimited words become ctrl+w's alone.
10. **`lineedit`'s newline:** alt+enter inserts one, beside Shift+Enter
    where `termkey` reads it and the trailing `\`.
11. **`lineedit`'s paste** is inserted as typed text, cleaned by
    `safetext` with newlines and tabs kept, whatever the terminal filtered;
    a completion never runs on it.
12. **`MemHistory`** keeps D6's bound, 1000, drops a line equal to the
    previous one and an empty one, and writes fish's escapes, which are
    gobble's.
13. **The `Printer` holds a paragraph line until the next line** (Q10),
    or the end: an underline after it makes it a heading, and a delimiter
    row a table's header, each drawn as what it is.
14. **The `Printer`'s inline constructs** resolve over the paragraph's
    lines received so far; a delimiter still open when its line prints is
    printed as it is, and no printed line is styled again; a reference
    link is styled only when its definition came first.
15. **The reprint** writes `SGR 0` first, then D9's sequence, `CSI 2 J`
    before `CSI 3 J`. Whether it clears the scrollback by default, as
    D13 has it with `WithoutScrollbackClear`, stays so (Q11): API rule
    12's third clause lets a new terminal write be on where its record says
    why, and D9's reason, a reprint that lines up, is this record's; kitty
    and iTerm2 handle `CSI 3 J` each their own way, which the transcript's
    documentation says.
16. **Wrapping at spaces** stays D7's and gobble's, and its limit is
    stated: a run of CJK or Thai text is one word (UAX #14 §3.1), which
    the terminal wraps.

#### A3.5 Owner questions

*Answered 2026-10-27,* each picked from its options: Q8 "10 rows
(Recommended)"; Q9 "pick's own keymap commands (Recommended)"; Q10 "Hold
one line (Recommended)"; Q11 "On, WithoutScrollbackClear (Recommended)".

* **Q8. The picker's height.** Recommended: 10 rows, gum's and huh's,
  capped by the terminal. The alternative is D5's 8, or 7, survey's and
  inquire's page.
* **Q9. The picker's keys.** Recommended: commands in the program's one
  keymap, as Q4 decided for the composer: `pick.DefaultRules()` binds
  `pick.select.next`, `pick.select.prev`, `pick.page.next`,
  `pick.page.prev`, `pick.accept` and `pick.cancel` in the scope
  `kind:chooser`, to the keys A3.4 item 3 names, and `WithKeymap` takes
  the program's keymap, a user's file rebinding them; without it the
  chooser builds its defaults alone. The alternatives are the palette's
  command IDs and scope reused, so one binding moves both pickers but
  `pick`'s commands carry `palette`'s names; or a fixed set of keys, a
  `decided` row, as `ask`'s are.
* **Q10. The `Printer`'s paragraph lines** (before Step 5). Recommended:
  each held until the next line, or the end, so a setext heading and a
  table's header are drawn as what they are, at the cost of one line of
  latency in prose. The alternative is gobble's: each printed as it
  completes, an underline drawn as a dim line, a heading's text unstyled.
* **Q11. The reprint's scrollback clear** (before Step 6). Recommended:
  D13 as accepted, on by default with `WithoutScrollbackClear`, the
  record's reason a reprint that lines up. The alternative is off until
  `WithScrollbackClear`.

### A4 (2026-10-27): `Rank`'s tie order

*Status: accepted (2026-10-27),* by the owner's answer to deviation D11 of
the PLAN's Step 3, "Input order (Recommended)", found by the step's
`TestFromEntriesRankedLikeChooser` before any release.

1. **A3.1's first fact is corrected.** `fuzzy.Matcher.Rank`'s order is
   total: the grade, the value, then the shorter key, then the input order
   (`fuzzy/rank.go:23-35`, at `4341af2`). "Stable on ties", which A3.1
   quoted from its documentation, holds only between keys of one length.
   A probe at `4341af2`: for the query `gpt`, `openai/gpt-5`,
   `openai/gpt-5-mini`, `openai/gpt-5-nano`, `openai/gpt-4.1` and
   `openai/gpt-4.1-mini`, given in that order, each score `prefix 84`, and
   `Rank` returns them by length, `gpt-5`, `gpt-4.1`, `gpt-5-mini`,
   `gpt-5-nano`, `gpt-4.1-mini`, where the palette's engine, ordering by
   score and then by arrival (`palette/engine.go:448-456`), keeps the order
   given.
2. **A3.4 item 4's decision stands:** ties keep the input order, so the
   program's own order, such as a sessions list's recency, decides between
   equal matches, and `pick` and the palette order one list alike. The
   `Chooser` scores each candidate with `Matcher.Score` and sorts its
   matches stably by grade and then value, as the palette's engine orders
   hits; it does not call `Rank`.
3. **`fuzzy.Matcher.Rank`'s documentation** says its tie order, the shorter
   key and then the input order. Its behaviour is unchanged.

The options weighed were: `Rank`'s order kept, with `FromEntries`
yielding its matches in it, which would order a sessions list's ties by
title length and change an accepted decision; and a tie-order option in
`fuzzy`, new exported API for one caller.

### A5 (2026-10-27): the composer's parity reference, its clear, its editor key, and the move of `ask` and `pick`

*Status: accepted (2026-10-27),* the owner's answers to
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 4's deviations D13 to D19, each "(Recommended)", D13 to D18 found
before that step's code and D19 by its fuzzing:

1. **x/term is a recorded reference, not a requirement.** D1's "No module
   is added" stands: a generator under `lineedit/testdata/`, which the go
   command ignores, runs `golang.org/x/term` v0.47.0's `Terminal` in a
   temporary module over gobble's parity cases and writes each outcome to
   a file there; the `Composer`'s parity test compares with that file. The
   cases D6 decides otherwise are left out by name: a bracketed paste,
   which x/term returns a line at a time, and ctrl+c, which x/term ends
   the input with and D6 makes `EditInterrupt`.
2. **ctrl+l clears only after `WithClearScreen()`:** `lineedit.ClearID`,
   bound to ctrl+l in `DefaultRules`, writes `CSI 2 J` and `CSI H` and
   draws the composer again at the top, as x/term v0.47.0
   (`terminal.go:632-634`) and gobble do; without
   the option it does nothing (API convention 12's third clause).
3. **ctrl+x ctrl+e opens the external editor,** bash's
   `edit-and-execute-command (C-x C-e)`, "Invoke an editor on the current
   command line" (bash's manual page, at bash 3.2.57; the Readline user
   manual has no such command): `lineedit.ExternalID`, bound to it in
   `DefaultRules`, sends `EditExternal` only with `WithExternalEditor`, and
   does nothing without it.
4. **`ask` and `pick` move to the `Composer` in Step 4,** as A2 item 1
   says: the reason and the query are one-row composers, with no newline
   and no history, their editing keys the keymap's in the composer's
   scope after their own; `ask` gains `WithKeymap`, as `pick` has. Their
   tests and goldens show the move changes nothing drawn.
5. **A word wider than its whole row is broken at the width,** by
   grapheme cluster, its pieces on rows of their own with the
   continuation indent, as A2 item 5 breaks one in `ask`; every word that
   fits stays whole. The Context's "leaves a word wider than a row whole"
   was gobble's 0027-MADR D2 as written, not its code at `5fb6928`, which
   cuts such a word at the row's end (`internal/cli/editor/layout.go:30-31`,
   `rowEnd` at `:73-88`); D6's "a wider word left whole" followed it.
   Left whole, the region would cut the row at the width, so the rest of
   the word, and the cursor over it, could not be seen.
6. **A forward word motion stops at the end of the next word,** as
   Readline's `forward-word` does, "Move forward to the end of the next
   word. Words are composed of letters and digits." (the Readline user
   manual, `tiswww.cwru.edu/php/chet/readline/rluserman.html`, read
   2026-10-28; the page states no version), and bash and macOS's
   Option+Right with it:
   alt+f, alt+right and ctrl+right; alt+d kills to there. alt+b,
   alt+left and ctrl+left stop at the start of the current or previous
   word, and alt+backspace kills back to there. A word is a run of
   clusters whose first rune is a letter or a digit (A3 item 9). x/term's
   Alt+Right stops at the start of the next word, so the parity test
   leaves out the two of gobble's cases that tell them apart, by name.
   *Added 2026-10-28,* by D18.
7. **A row is measured as it is drawn.** A2 item 5's "`Layout` never
   writes a row wider than the width" did not hold: a mark, a variation
   selector or a joiner with no base, such as U+FE0F at a word's start, is
   0 cells measured alone, but drawn after a space it joins the space's
   cluster, and under `ansi.GraphemeWidth` `" \ufe0f"` is 2 cells (a probe
   of `x/ansi` v0.11.8). Unicode 18.0.0 calls it an isolated combining
   character, which a process "may present … as if it were a base
   character" (§3.6.1, D52), and exhibits one in isolation on U+00A0
   NO-BREAK SPACE (§2.11.4). `ask.Layout` breaks words only at a cluster
   that is a space alone, so such a mark stays in its word, measured with
   the space it joins, and its rows' widths add up as they are drawn; `pick`
   measures a row's text with the marker before it; the composer a
   line's first row with its lead. The text is unchanged. *Added
   2026-10-28,* by D19.

### A6 (2026-10-28): links styled, the scanner unexported, and the printer's limits

*Status: accepted (2026-10-28),* the owner's answers to
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 5's deviations D20 and D21, each "(Recommended)", found before that
step's code:

1. **A link's text is underlined.** D7's styling, gobble's `mdstyle.go`,
   styles no link, so A3 item 14's reference clause had nothing to apply
   to. An inline link's text, `[text](url)`, is drawn in `Markup`'s link
   style, underline by default, and its URL dim, every marker kept; a
   reference link, `[text][ref]` or `[text]`, is so styled only when its
   definition was printed before it, and a definition after it changes
   nothing printed. No OSC 8: a printer that only prints should not
   assign links (A3.3's OSC 8 note).
2. **The block scanner is unexported** until 0009's `Doc`, its second
   consumer, is built on it; 0009's amended PLAN exports it as `Scanner`
   then, with both consumers in view. D7's "one block scanner, two
   consumers" stands, and the glossary keeps `Scanner` reserved.
3. **The examples the `Printer` is tested on,** read from the
   specifications at Step 5's start (2026-10-28): CommonMark 0.31.2's
   setext headings, examples 80 to 106 (`spec.commonmark.org/0.31.2/spec.json`,
   section "Setext headings", 27 examples), example 95 `Foo`, `Bar`,
   `---` being `<h2>Foo\nBar</h2>` as A3.3 read it; and GFM 0.29-gfm's
   tables, examples 198 to 205 (cmark-gfm 0.29.0.gfm.13's
   `test/spec.txt`, "Tables (extension)", eight examples).
4. **What holding one line (Q10) does not do,** stated so the tests
   expect it: a setext heading of more than one line, example 95's, has
   only its last line drawn as the heading, the lines before it printed as
   the paragraph they were until the underline came; a line of a
   paragraph is never styled again. A table is held whole, its header
   known when the delimiter row comes, until a blank line or the start of
   another block, as GFM §4.10 ends it, its rows with or without their
   outer pipes (examples 199 and 202); gobble's `Tables` reads only rows
   that start with a pipe. An HTML block's lines (CommonMark §4.6) print as
   they come, unstyled and unwrapped, as a code line does.

### A7 (2026-10-28): a table's layout, and its control

*Status: accepted (2026-10-28),* the owner's answer to
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 5's deviation D22, "Lay out; WithoutTableLayout", found while that
step's tests were written:

1. **D7's "byte for byte" is the styler's, not the `Printer`'s.** gobble
   lays a table out before it styles (`render/table.go:76`, `Tables`,
   then `Style`), and its "byte for byte" with colour off is `Style`'s
   alone (`render/mdstyle.go:12`), so its tables are laid out with colour
   off too. D7 put both in one sentence, which a reply with a table
   cannot satisfy. *Corrected 2026-10-28 by A8 item 1: gobble lays a
   table out only when colour is on, so with colour off its tables come
   as they came.*
2. **The `Printer` lays a table out under every `Markup`,** as gobble
   does *(with colour on; A8 item 1)*: the columns as wide as their widest cell, in the width method,
   aligned as the delimiter row says. The zero `Markup` styles nothing.
3. **`stream.WithoutTableLayout` turns the layout off** (AGENTS.md "API
   conventions" rule 12: a policy applied to the text a user is shown):
   a table is still held until it ends, and its rows print as they came,
   styled by the `Markup`. With it and the zero `Markup`, the output is
   the input, byte for byte.

### A8 (2026-10-28): A7's fact corrected, and the transcript's API

*Status: accepted (2026-10-28),* the owner's answers to
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 6's deviations D23 to D26, each "(Recommended)", found before that
step's code:

1. **A7 item 1's fact is corrected (D23).** gobble styles a reply and
   lays its tables out only when colour is on: `text` runs `Tables` and
   `Style` inside `if v.caps.Out.Color`, "styled when colour is on, raw
   when it is off (D7)" (`internal/cli/view.go:269-276` at `5fb6928`),
   and its reprint's `replyAt` does the same (`:508-513`). With colour
   off, gobble prints a table as it came. D22's choice stands: the
   `Printer` lays a table out under every `Markup`, on purpose unlike
   gobble, so plain output keeps aligned columns; `WithoutTableLayout`
   gives gobble's colour-off output.
2. **`stream.Printer.At(width)` draws a reply again (D24).** *Amended
   2026-10-29 by A9 item 2: with no sanitizer, from the cleaned source.* D9's
   `AddReply(p)` keeps the `Printer`; a reprint draws `At(width)`: its
   `Source` written to a new `Printer` with its options at that width,
   then flushed. At a reprint the live `Printer` is idle, nothing held,
   so `At` gives what it would have printed at that width.
3. **The transcript's further controls (D25),** by API convention 12:
   `WithSynchronized(on bool)`, the reprint one synchronized update (mode
   2026), on by default as `inline`'s is; and the notice line, the
   catalogue's `MsgEarlierOutput` with its English default "(earlier
   output is not shown again)", through `WithCatalog`, drawn in
   `WithNoticeStyle(stream.Style)`, unstyled by default, as `stream`'s
   zero `Markup` is.
4. **The reprint waits for idle at the host's call (D26).** `Timeout`
   gives the settle deadline until it passes, and then nothing, so a host
   never waits on a passed deadline. A reprint still due is given by the
   next `HandleTimeout(now, true)`, which the host calls whenever its
   output becomes idle; it gives "" when nothing is due. This is gobble's
   `tryReprint` at each line end (`view.go:470-493`).

### A9 (2026-10-29): the reprint's and the printer's speed, and their budgets

*Status: accepted (2026-10-29),* the owner's answer to
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 6's deviation D27, "Step 6a, v2.6.1 (Recommended)", found after
`v2.6.0` was tagged, when the owner asked how gobble does it and for the
numbers to come down:

1. **The fact.** On one Apple M1 Pro, the same day, five runs each:
   `v2.6.0`'s `BenchmarkReprint` takes 12.2 to 12.3 ms, 15.6 MB and
   186,585 allocations a reprint; gobble's, at `ccd8aa8` (its
   `transcript.go`, `view.go` and `render` unchanged since `5fb6928`), 9.6
   to 11.9 ms, 7.3 MB and 49,696. A reply streamed in 64-byte chunks at 100
   columns, gobble's `benchReplyRender`, takes 33.7 µs and 636 allocations
   through `v2.6.0`'s `Printer`, and 32.9 µs and 257 through gobble's
   `render`. gobble's work is the same in kind and order; the profiles put
   the difference in four places of this repository's code:
   * `stream`'s wrapper builds each word a byte at a time by string
     concatenation (`stream/wrap.go:49`), 59% of the reprint's
     allocations, and measures each word's width three times; gobble's
     `Wrap` reuses byte buffers and swaps them (its 0030-MADR D5, as
     amended);
   * `Printer.At` builds a `NewPrinter`, so each reply drawn is cleaned
     again by two sanitizers and its `Flush` opens two more, about 6.7 KB
     each, 22% of the reprint's bytes, though `Source` is cleaned already;
     gobble's `replyAt` draws the released source and cleans nothing;
   * the row count measures every line's width (`transcript.go:238`),
     where gobble counts line breaks;
   * the line breaks are made CR LF by two copies of the whole reprint.
2. **`At` draws the source it keeps, with no sanitizer.** `Source` is the
   rules' and the floor's output, the bytes the `Printer` printed from;
   `At` gives those bytes to a new `Printer`'s line reader at the width and
   ends it, so its output is what the `Printer` printed, A8 item 2's
   promise, from bytes cleaned once. *This amends A8 item 2's "written to
   a new Printer".*
3. **A line no longer in bytes than the width takes one row.** No
   character is wider in cells than in bytes under either width method: a
   character two cells wide is at least three bytes, every other is at
   most one cell, and an escape sequence is none. The count is the same,
   and a line is measured only when it is longer in bytes than the width.
4. **The wrapper takes each word and gap as a slice of its line,**
   measures each word once, and takes a word of printable ASCII alone as
   wide as it is long, which every method measures so. Its output is
   unchanged, byte for byte.
5. **Budgets** (D11's "per-key benchmarks with recorded baselines"): the
   benchmark's reprint has a budget of allocations and of bytes, and the
   streamed reply one of allocations, gobble's numbers, held by a test on
   every host: at most 49,696 allocations and 7,274,618 bytes a reprint,
   and 257 allocations a reply. An allocation count is the same on every run, and
   `v2.6.0`'s code fails each budget. Time is recorded beside gobble's, as
   before, and is not gated: it varies by host and load, which gobble's own
   gate allows for with a slack of three times its baseline (its
   `perf_test.go`), and `v2.6.0`'s 12.3 ms passed that.
6. **A patch release, `v2.6.1`:** nothing exported changes.

### A10 (2026-10-29): a reply's parameters, counted on every path to them

*Status: accepted (2026-10-29),* the owner's answer to
[0023-PLAN-line-mode-components.md](0023-PLAN-line-mode-components.md)
Step 7's deviation D28, "Amend 0023, patch v2.6.2 (Recommended)", found
when CI ran on Step 7's push:

1. **The fact.** CI run 40969366150, on `5dd7285`, fuzzed
   `FuzzParseReply` for its 20 s and found a panic in x/ansi v0.11.8's
   parser (`parser_decode.go:200`, `index out of range [33] with length
   33`), reached through `termcap.ParseReply` (`reply.go:147`), on 37
   bytes: ESC, a space, `[`, 33 semicolons and `0`. The parser stores a
   digit at `params[paramsLen]` with no bound, once the separators have
   taken `paramsLen` to the buffer's end; v0.11.9 does the same. It reaches
   that state from ESC, any intermediate bytes 0x20 to 0x2f, then `[` or
   `P`, and from 0x9B or 0x90 as the first byte. `ParseReply`'s guard
   counts the separators only after `ESC [`, `ESC P`, 0x9B or 0x90, so an
   intermediate between ESC and `[` passes it. It panics so at `v2.2.0`,
   before this record's first step. `inline/inlinetest`'s `Screen`, built
   in Step 1, decodes what it is written with a parser of 32 parameters
   and has no guard: `ESC [`, `ESC SP [`, `ESC SP ! " [`, 0x9B, `ESC P` and
   `ESC SP P` with 40 separators and a digit each panic it, and
   `ParseReply` on the three with an intermediate. termkey decodes the CI
   input as keys, alt+space and then `[` and the semicolons, so
   `inline.Region`, which gives `ParseReply` an unknown sequence's bytes,
   was not shown to reach it; `ParseReply` is exported and takes any
   string, and `termcap/termcaptea` gives it Bubble Tea's unknown
   sequences.
2. **One count, on every path.** `internal/sanitize` counts the
   parameters x/ansi's parser would read for the sequence at a string's
   start, on each path item 1 names, and gives the bound, 32. `ParseReply`
   reads a reply with as many or more as `ReplyUnknown`, as it did for
   `ESC [`, with its parser one larger, as before; `Screen` decodes such a
   sequence with no parser and shows nothing for it, as for any sequence
   it does not draw, and the text after it is drawn. `inline/inlinetest`
   imports `internal/sanitize`, already in its closure through `inline`;
   D1's table gains it.
3. **Tests,** each seen failing on the code as it is: every path with
   the bound's separators and a digit, through `ParseReply` and through
   `Screen`, does not panic, and a reply one under the bound is still
   read; CI's input joins `FuzzParseReply`'s seeds.
4. **A patch release, `v2.6.2`:** nothing exported changes.
5. **x/ansi's parser** panicking past its buffer is upstream's to fix;
   telling charmbracelet/x is the owner's call, and this record does not.

## More Information

* Issues of this repository: #7, #8, #9 and #10, each with gobble's
  comment of 2026-10-23 on the `x/ansi` relaxation.
* gobble-cli, at `5fb6928`: `docs/decisions/0027-MADR-native-line-editor-word-wrap.md`
  (D2, D3, D5 and their amendments, Q2) and its PLAN;
  `0028-MADR-reply-word-wrap.md` (D5, D7); `0029-MADR-permission-prompt-layout.md`
  (D1, D2, D5, D6) and its PLAN (P2); `0030-MADR-resize-and-performance.md`
  (D1 to D5) and its PLAN (P1, P4, P5); `0032-MADR-go-tui-lib-integration-boundary.md`
  (D1 to D7, A1) and its PLAN; and `internal/cli`'s `editor`, `render`,
  `term`, `term/termtest`, `transcript.go`, `view.go`, `permission.go`,
  `trust.go`, `sessions.go` and `line.go`.
* Records here:
  [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md);
  [0008-MADR-command-palette.md](0008-MADR-command-palette.md) (the
  palette; A3's keys as commands);
  [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
  (§1, §4, A2), which D7 amends;
  [0016-MADR-go-1-27-2-for-go-2026-6604.md](0016-MADR-go-1-27-2-for-go-2026-6604.md);
  [0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  (D1, D2, D7, D11, A2);
  [0021-MADR-termkey-closure-gate.md](0021-MADR-termkey-closure-gate.md);
  [docs/glossary.md](../glossary.md).
* External sources, read 2026-10-25: golang.org/x/term v0.47.0,
  `terminal.go` (`Terminal.Write`) and its `History`; reedline 0.52.1,
  <https://docs.rs/reedline/latest/reedline/>; prompt_toolkit,
  <https://python-prompt-toolkit.readthedocs.io/en/master/pages/asking_for_input.html>;
  inquire 0.9.4, <https://docs.rs/inquire/latest/inquire/>; liner v1.2.3,
  <https://pkg.go.dev/github.com/peterh/liner>; goldmark v1.8.2's
  `go.mod`; the Go release history, <https://go.dev/doc/devel/release>.
* No PLAN is written yet: the owner asked for the decision first.
