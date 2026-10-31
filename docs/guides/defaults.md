# Defaults, and how to refuse them

Everything go-tui-lib does on a program's behalf has a default and a
control a program can use without forking: a `Without…` option, a nil or
zero value, a hook, or the lower layer the convenience is built on
(AGENTS.md, API conventions, rule 12;
[0024-MADR](../decisions/0024-MADR-opt-in-and-opt-out-coverage.md)). This
page lists each, by package.

Each row's status is one of:

- **built**: the control exists, as named;
- **planned**: a PLAN builds it, as named;
- **floor**: the library keeps it whatever a program sets, for the reason
  in the table of floors below; a control named there is the lower layer a
  program may use instead;
- **decided**: a record chose to give no control, for the reason it
  gives.

`internal/conformance`'s `TestDefaultsInventory` reads every table below
headed `Behaviour | Default | Control | Status`, and fails on a control
that does not exist, a `planned` row whose control already does, and a
record that is not there.

## The floors

| Floor | Why | What a program may do instead |
| :--- | :--- | :--- |
| No read of the environment, no write to `os.Stdout` or `os.Stderr`, no signal handler, no exit, no process | the program owns its process (AGENTS.md, TUI conventions 1 and 2; API conventions 9) | its own code, reading through `termcap.Env` |
| Untrusted text cleaned where the library draws it | a title, a model's reply or a terminal's answer can carry sequences that act on the terminal | `safetext` with its own `Rules`, for text it draws itself |
| Bounds on untrusted input | a hostile or broken peer must not exhaust memory or time | raise a bound through its `With…` option, where one exists; never remove it |
| Grammars and protocols | both sides of a file, a protocol or a keymap read them alike | none: they are the contract |
| A command's declared danger; a handler's panic recovered | the gate decides by danger; one handler must not take the UI down | write its own `Command.Handler` |

## command

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| The built-in commands, after construction | `command.list`, `command.describe` and `app.quit` in every registry | `command.Registry.Remove` | built |
| The built-in commands, at construction | added by `NewRegistry` | `command.WithoutBuiltins`, `command.BuiltinList`, `command.BuiltinDescribe`, `command.BuiltinQuit` | built |
| The gate | none: a request the policy asks about is refused | `command.WithGate`, `command.GateFunc`, `command.AllowIf` | built |
| A request's own gate | asked where the policy asks | `command.Request.Gate` | built |
| Which requests ask the gate | a mutating command from an agent; a destructive one from an agent or the shell | `command.WithGatePolicy`, `command.GatePolicy`, `command.GatePolicyFunc`, `command.DefaultGatePolicy` | built |
| Remembered verdicts | kept for the registry's life | `command.Registry.Forget`, `command.Registry.ForgetAll` | built |
| The auditor | none | `command.WithAuditor`, `command.AuditorFunc`, `command.SlogAuditor` | built |
| A request's argument bytes | 1 MiB | `command.WithMaxArgBytes` | built |
| A loaded command's slash name when taken | `prefix:name` | `command.WithPrefixer` | built |
| A command directory's limits | 256 KiB a file, 1000 files, 8 levels | `command.WithMaxFileBytes`, `command.WithMaxFiles`, `command.WithMaxDepth` | built |
| Where a `Loop` command runs | on the calling goroutine, until a loop is attached | `command.WithLoop`, `command.Registry.Attach` | built |
| Where a command is offered | every surface, when none is named | `command.WithSurfaces` | built |
| A handler's panic | recovered into an error | `command.Command.Handler` | floor |
| A command's danger | required | `command.WithDanger` | floor |
| The ID, slash and argument grammars | fixed | `command.ArgsOf`, `command.Request` | floor |
| Exit statuses of its errors | `ErrUnknown` 2, `ErrUnavailable` 1, `ErrRefused` 3 | `launch.ExitError` | built |
| The built-in commands' text, and the MCP error text | English | `command.WithCatalog` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |

## command/commandtea

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| How long `Watch` waits | until the program ends | `command/commandtea.WatchContext` | built |

## keymap

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| Default bindings | none: the program passes its own | `keymap.New` | built |
| A pending sequence's timeout | 1 s | `keymap.Settings.Timeout` | built |
| Esc and backspace while a sequence is pending | esc clears it, backspace takes back a stroke | `keymap.Settings.EscReplays`, `keymap.Settings.BackspaceReplays` | built |
| Keys a user may not bind | none | `keymap.Settings.Reserved` | built |
| Key labels | the system's glyphs and the display aliases | `keymap.DisplayAliases`, `glyph.Set` | built |
| Help text for a command | its ID | `keymap.Describer`, `keymap.DescriberFunc` | built |
| How a legacy terminal's keys are read | normalised as the terminal sends them | `keymap.Settings.Features` | floor |
| Strokes in a sequence | at most 4 | `keymap.ParseSequence` | floor |
| Bubble Tea's timer for a pending sequence | `keymaptea.Update` starts one | `keymap.Dispatcher.Key`, `keymap.Dispatcher.Expire` | built |

## launch

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| A program's own variable that forbids the TUI | none | `launch.Config.NoInputEnv` | built |
| `CI` forbids the automatic choice | read by name | `launch.ChoiceTUI` | built |
| `CI`, without forcing the TUI | read by name | `launch.Config.IgnoreCI` | built |
| `TERM=dumb` | plain output, under every choice | `launch.Config.AllowDumb` | built |
| Building a decision as `Decide` does | `Decide` sets the profiles and the glyph tier | `launch.ColorProfile`, `launch.GlyphTier` | built |
| The controlling terminal | not opened | `launch.Config.OpenTTY` | built |
| Drawing on standard error | not | `launch.Config.UIOnErr` | built |
| Bubble Tea's program options | the streams, the profile, the size, no signal handler | `launch.WithProgramOptions` | built |
| Bubble Tea's signal handler | off | `launch.WithProgramOptions` | floor |
| The model's epilogue | written to `Out` on a clean end | `launch.WithoutEpilogue`, `launch.WithEpilogueWriter` | built |
| Modes to reset at the end | none | `launch.WithRestorer` | built |
| A registry for the run | none | `launch.WithRegistry` | built |
| A message filter | none | `launch.WithFilter` | built |
| A hook when the program starts | none | `launch.OnStart` | built |
| The flags' names | `--mode` and `--tui` | `launch.Choice` | built |
| Exit statuses | Bubble Tea's ends as the shell expects | `launch.ExitError` | built |
| An adapter for a restorer | none | `launch.RestorerFunc` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |

## when

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| The grammar, and its 4096 bytes and depth 64 | VS Code's when clauses | `when.Parse` | floor |
| Where the values come from | a map or layers | `when.Map`, `when.Layered` | built |
| An adapter for a context | none | `when.ContextFunc` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |

## workspace

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| Its keys | `workspace.DefaultRules` | `workspace.WithBindings`, `workspace.Workspace.SetBindings` | built |
| Its own commands, on a key | run in place, past the program's registry | `workspace.WithoutOwnCommands` | built |
| Esc closes the top overlay | always | `workspace.EscConsumer` | built |
| Its commands for a registry | given only when asked | `workspace.Commands` | built |
| The background colour query | sent at start | `workspace.WithoutBackgroundQuery` | built |
| The theme | follows the terminal's profile and background | `workspace.WithTheme`, `workspace.WithThemeBuilder`, `workspace.WithProfile`, `workspace.WithBackground`, `workspace.WithGlyphs`, `workspace.Workspace.SetTheme` | built |
| The width method | WcWidth until mode 2027 | `workspace.WithWidthMethod` | built |
| The window clamp | `MaxSide` and `MaxCells` | `workspace.WithSize` | floor |
| Chrome | rounded borders | `workspace.WithChrome`, `workspace.WithPaneChrome`, `workspace.WithBorder` | built |
| An overlay's chrome | a bordered box with a title | `workspace.Overlay.Bare` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| Pane titles, badges and views | cleaned | `safetext.Clean` | floor |
| The mouse | on | `workspace.WithoutMouse` | built |
| Click to focus, drag to resize, one at a time | on with the mouse | `workspace.WithoutMouseFocus`, `workspace.WithoutMouseResize` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| Focus | the first pane of the ring | `workspace.WithFocus`, `workspace.WithFocusRing`, `workspace.Focusable` | built |
| Focus when the focused pane hides | the ring's first pane | `workspace.WithFocusFallback`, `workspace.FocusFallback` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| A help footer | the workspace's bindings and the focused pane's | `workspace.Workspace.Bindings` | built |
| Text it shows: results, help | English | `workspace.WithCatalog` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| A modal overlay takes clicks outside it | always | `workspace.Overlay.Modal` | decided (0002-MADR-multi-pane-workspace-layouts.md) |

## palette

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| Its keys | `palette.DefaultRules` | `palette.DefaultRules` | built |
| Its commands' IDs and scope | `palette.open` and the picker's moves, in `kind:picker` | `palette.OpenID`, `palette.SelectNextID`, `palette.PickerScope` | built |
| A completion pop-up's keys under another ID | bound in `overlay:completion` only | `palette.CompletionRules` | built |
| A wrapper of a built-in provider | `Open` tells each provider that has `Opened` | `palette.Opener` | built |
| The overlay | `palette`, 72 by 16, centred | `palette.WithID`, `palette.WithAnchor`, `palette.WithSize` | built |
| Hits kept | 200 | `palette.WithLimit` | built |
| A provider's error | dropped | `palette.WithLogger` | built |
| Key hints' system | the running one | `palette.WithGOOS` | built |
| The input's placeholder | none | `palette.WithPlaceholder` | built |
| Recents | not recorded | `palette.WithRecents` | built |
| A completion pop-up | not | `palette.WithoutInput` | built |
| The argument stage | for a hit with parameters | `palette.Hit.Params` | built |
| A line-mode picker's entries | a provider of them, its hits with no `Run` | `palette.FromEntries` | built |
| A click on a row | accepts it | `palette.WithoutClickAccept` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| Accepting a hit | closes the picker | `palette.WithoutCloseOnAccept` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| A query of `?` | lists the prefixes | `palette.WithoutPrefixList` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| Text it shows | English | `palette.WithCatalog` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| Hit text | cleaned | `safetext.Clean` | floor |
| The file provider's limits | an index above 5000 paths, a walk of 200 000 | `palette.WithIndexAbove`, `palette.WithMaxFiles` | built |
| Recents' half-life and size | 7 days, 500 | `palette.WithHalfLife`, `palette.WithMaxEntries` | built |
| Recents' clock | `time.Now` | `palette.Frecency.Score` | decided (0008-MADR-command-palette.md) |
| A recent hit's boost | at most one consecutive-match bonus | `palette.Recents` | decided (0008-MADR-command-palette.md) |
| A query's length | 256 characters | `palette.Picker.Open` | floor |

## theme, glyph and layout

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| The palettes | dark, light and unknown | `theme.WithPalette`, `theme.WithPaletteFor` | built |
| The styles | from the palette | `theme.Theme.Styles` | built |
| The glyphs | Unicode or ASCII | `glyph.Set`, `glyph.Unicode`, `glyph.ASCII` | built |
| The presets' sizes and folds | a 30% sidebar, folds at 100, 70 and 16 columns | `layout.WithoutResponsive` | built |
| A layout state's version | 1 | `layout.State` | floor |

## tuitest

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| The update switch | `-tuitest.update`, `TUITEST_UPDATE`, or a test's own `-update` | `tuitest.Golden` | decided (AGENTS.md, Pre-add checks) |

## termcap and termcap/termcaptea

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| The probe | sends its batch | `termcap.WithDisabled` | built |
| The probe, by the rule's name | sends its batch | `termcap.WithoutQueries` | built |
| The background query, and the DECRQMs | sent | `termcap.WithoutBackgroundRequest`, `termcap.WithoutModeQuery` | built |
| One query of the batch | sent | `termcap.WithoutQuery` | built |
| A query of the program's own | none | `termcap.WithQuery` | built |
| The gated queries | sent where the heuristic allows | `termcap.WithoutHeuristic` | built |
| The gated queries, off | sent where the heuristic allows | `termcap.WithoutGatedQueries` | built |
| Mode 2031 | set when supported | `termcap.WithoutColorSchemeUpdates` | built |
| The probe's timeout | 2 s | `termcap.WithTimeout` | built |
| What the environment says | read from the `Env` given | `termcap.WithOverride`, `termcap.WithAppearanceEnv`, `termcap.WithAppearanceHook`, `termcap.WithConsoleHost` | built |
| The system the identity is read for | the running one | `termcap.WithGOOS` | built |
| One capability view, apart from the brand | from the brand's table | `termcap.Caps.LinksSupport` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| A reply's length | 1 KiB | `termcap.ParseReply` | floor |
| The DA1 sentinel | sent last | `termcap.Probe.Start` | floor |
| The report | 80 columns, text | `termcap.WithReportWidth`, `termcap.WithReportJSON` | built |
| The report's findings | English | `termcap.WithReportCatalog` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| tmux's facts | not asked | `termcap.TmuxQuery`, `termcap.ParseTmux`, `termcap.Probe.SetTmux` | built |
| Modes 2026 and 2027 in a Bubble Tea program | asked by Bubble Tea, not the prober | `termcaptea.NewProber` | decided (0020-MADR-charm-free-cores-glyphs-and-safetext.md) |

## termsvc

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| A notification's protocol | OSC 99, 9, 777, then the bell | `termsvc.WithProtocol` | built |
| The bell as the last protocol | under `Auto` | `termsvc.WithoutBell` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| When a notification is sent | while unfocused | `termsvc.WithPolicy` | built |
| A notification backend | none | `termsvc.WithBackend`, `termsvc.BackendFunc`, `termsvc.WithBackendTimeout` | built |
| A notification filter | none | `termsvc.WithNotifyFilter` | built |
| A notification's title and body | 80 and 240 cells | `termsvc.WithTitleCells`, `termsvc.WithBodyCells` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| The clipboard | OSC 52 | `termsvc.WithClipboard`, `termsvc.WithCopyTimeout` | built |
| The copy's route | by the terminal and the multiplexer | `termsvc.WithCopyRoute` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| A link's schemes | http, https, file, mailto | `termsvc.WithLinkSchemes` | planned (0024-PLAN-terminal-text-and-convention-opt-outs.md) |
| Text it writes | cleaned | `safetext.Clean` | floor |

## termkey

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| A paste's and a sequence's bounds | 1 MiB and 4 KiB | `termkey.Decoder.MaxPaste`, `termkey.Decoder.MaxSequence` | built |
| The escape timeout | the host's, 50 ms suggested | `termkey.Decoder.Flush` | built |
| How keys are read | xterm's, the Kitty protocol's and bracketed paste | `termkey.Decoder.Decode` | floor |

## safetext

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| What survives | by the `Rules` given | `safetext.Rules` | built |
| Invisible and bidirectional controls, invalid UTF-8 | removed, or U+FFFD | `safetext.Rules` | floor |
| A kept OSC 8's length | at most 2,048 bytes; a longer one is removed | `safetext.Rules` | floor |

## fuzzy

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| Letter case | smart | `fuzzy.WithCase` | built |
| Word delimiters | slash, backslash, comma, colon, semicolon, the vertical bar and space | `fuzzy.WithDelimiters` | built |
| The alignment bound | 512 | `fuzzy.WithMaxAlign` | built |
| fzf's extended syntax | off | `fuzzy.WithExtended` | built |
| A rune normaliser | none | `fuzzy.WithNormalizer` | built |
| The index's overlap and candidates | 0.3 and 2000 | `fuzzy.WithMinOverlap`, `fuzzy.WithMaxCandidates` | built |
| A query's length | 256 clusters | `fuzzy.NewMatcher` | floor |
| A boost's bound | one consecutive-match bonus | `fuzzy.Matcher.RankContext` | decided (0008-MADR-command-palette.md) |

## ask and ask/asktea

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| How text is measured | `ansi.WcWidth` | `ask.WithMethod` | built |
| The glyphs, the prompt among them | Unicode | `ask.WithGlyphs` | built |
| The region the question is drawn in | its own, measured as the question is | `ask.WithRegion` | built |
| A reason's bound | 2048 bytes | `ask.WithReasonLimit`, `ask.DefaultReasonLimit` | built |
| Its own words: the label before a reason, the reason's echo, a cancel's word | English | `ask.WithCatalog`, `ask.Messages` | built |
| How the question's text and a reason are cleaned | `safetext.Line`, never less | `ask.WithRules` | built |
| What cancels | the cancel answer's key, ctrl+c, and the host's `Cancel` | `ask.Question`, `ask.Asker.HandleInput` | decided (0023-MADR-line-mode-components.md D4) |
| A reason's editing keys | `lineedit.DefaultRules`, in `kind:composer` | `ask.WithKeymap`, `lineedit.DefaultRules` | built |
| A word or an answer wider than its whole row | broken at the width | `ask.Layout` | decided (0023-MADR-line-mode-components.md A2) |
| The overlay: its ID, place and width | `ask/asktea.DefaultID`, centred, the workspace's width | `ask/asktea.WithID`, `ask/asktea.WithAnchor`, `ask/asktea.WithWidth` | built |
| The overlay's question: its method and glyphs | the workspace's | `ask/asktea.WithAsk` | built |
| Esc in the overlay | closes it, the question canceled | `workspace.EscConsumer` | decided (0008-MADR-command-palette.md) |

## pick

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| How text is measured | `ansi.WcWidth` | `pick.WithMethod` | built |
| The glyphs: the prompt, the focus marker and the ellipsis | Unicode | `pick.WithGlyphs` | built |
| The prompt | the glyphs' | `pick.WithPrompt` | built |
| The query it starts with | empty | `pick.WithQuery` | built |
| The list's height | 10 rows, fewer when there are fewer entries | `pick.WithHeight`, `pick.DefaultHeight` | built |
| An index before ranking | above 5000 entries | `pick.WithIndexAbove`, `pick.DefaultIndexAbove` | built |
| The region it is drawn in | its own, measured as the entries are | `pick.WithRegion` | built |
| Its keys | `pick.DefaultRules`, alone | `pick.WithKeymap`, `pick.DefaultRules` | built |
| Its commands' IDs and scope | `pick.select.next` and the others, in `kind:chooser` | `pick.SelectNextID`, `pick.ChooserScope` | built |
| Its own words: the count, no results and a cancel's word | English | `pick.WithCatalog`, `pick.Messages` | built |
| How the entries' text, the prompt and the query are cleaned | `safetext.Line`, never less | `pick.WithRules` | built |
| What cancels besides its keys | ctrl+c, outside the keymap, and the host's `Cancel` | `pick.Chooser.Cancel` | decided (0023-MADR-line-mode-components.md D4) |
| The query's editing keys | `lineedit.DefaultRules`, in `kind:composer` after `kind:chooser` | `pick.WithKeymap`, `lineedit.DefaultRules` | built |
| Equal matches | in the order given | `pick.NewChooser` | decided (0023-MADR-line-mode-components.md A4) |
| A query's length | 1024 bytes | `pick.NewChooser` | floor |
| The matched cells styled | none: the zero `stream.Markup` | `pick.WithMarkup`, `stream.DefaultMarkup` | built |

## lineedit and lineedit/lineedittea

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| How text is measured | `ansi.WcWidth` | `lineedit.WithMethod` | built |
| The glyphs the prompts default to | Unicode | `lineedit.WithGlyphs` | built |
| The prompt and the continuation prompt | the glyphs' `Prompt` and `Ellipsis`, each with a space | `lineedit.WithPrompt`, `lineedit.WithContinue` | built |
| A header above each prompt | none | `lineedit.WithHeader` | built |
| The history | none | `lineedit.WithHistory`, `lineedit.MemHistory` | built |
| A `MemHistory`'s bound | 1000 entries | `lineedit.MemHistory.Max`, `lineedit.DefaultHistorySize` | built |
| A `MemHistory`'s empty entries and repeats | dropped | `lineedit.History` | decided (0023-MADR-line-mode-components.md A3) |
| Completion | none | `lineedit.WithCompleter`, `lineedit.CompleterFunc` | built |
| Its keys | `lineedit.DefaultRules`, alone | `lineedit.WithKeymap`, `lineedit.DefaultRules` | built |
| Its commands' IDs and scope | `lineedit.submit` and the others, in `kind:composer` | `lineedit.SubmitID`, `lineedit.ComposerScope` | built |
| shift+enter's newline | bound where the keymap's features say the terminal sends it | `keymap.Settings` | built |
| The text's bound | 4096 runes | `lineedit.WithLimit`, `lineedit.DefaultLimit` | built |
| A bracketed paste | inserted, editable | `lineedit.WithPaste`, `lineedit.PasteSubmit` | built |
| The external editor, on ctrl+x ctrl+e | off | `lineedit.WithExternalEditor` | built |
| ctrl+l clearing the screen | off | `lineedit.WithClearScreen` | built |
| The region it is drawn in | its own, measured as the text is | `lineedit.WithRegion` | built |
| How typed text, a completion, the prompts and the header are cleaned | `safetext.Line`, never less | `lineedit.WithRules` | built |
| A paste, an entry recalled and a text set | cleaned by the rules with line breaks and tabs kept, never below printable text, line breaks and tabs | `lineedit.WithRules` | floor |
| ctrl+c | `EditInterrupt`, outside the keymap, the text kept | `lineedit.ComposerEvent` | decided (0023-MADR-line-mode-components.md D6) |
| The pane's name in its events | `lineedit/lineedittea.DefaultID` | `lineedit/lineedittea.WithID` | built |
| The pane's composer: its method and glyphs | the workspace's | `lineedit/lineedittea.WithComposer` | built |

## stream

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| How text is measured | `ansi.WcWidth` | `stream.WithMethod` | built |
| Styling | none: the zero `Markup`; gobble's is `DefaultMarkup` | `stream.WithMarkup`, `stream.DefaultMarkup` | built |
| Wrapping prose | none | `stream.WithWidth`, `stream.Printer.SetWidth` | built |
| A table's layout | its columns as wide as their widest cell, aligned as its delimiter row says | `stream.WithoutTableLayout` | built |
| How the reply is cleaned | `safetext.Text`, never less | `stream.WithRules` | built |
| A paragraph's line | held until the next line comes, or the reply ends | `stream.Printer.Flush` | decided (0023-MADR-line-mode-components.md A3, Q10) |
| A table | held until a blank line or another block ends it | `stream.Printer.Flush` | decided (0023-MADR-line-mode-components.md D7, A6) |
| A link | its text in the `Link` style and its destination in the `URL` style, no OSC 8 | `stream.Markup` | decided (0023-MADR-line-mode-components.md A6) |

## transcript

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| What it keeps | 4 MiB of pieces, the oldest dropped past it | `transcript.WithMaxBytes`, `transcript.DefaultMaxBytes` | built |
| A reprint's rows | the newest 10,000, as the terminal shows them at the width | `transcript.WithMaxRows`, `transcript.DefaultMaxRows` | built |
| The wait after the last resize | 150 ms | `transcript.WithSettle`, `transcript.DefaultSettle` | built |
| How a row's cells are counted | `ansi.WcWidth` | `transcript.WithMethod` | built |
| The reprint clearing the scrollback | `CSI 3 J` written, after `CSI 2 J` | `transcript.WithoutScrollbackClear` | built |
| The reprint as one synchronized update | mode 2026 set and reset | `transcript.WithSynchronized` | built |
| How `AddText`'s text and the notice are cleaned | `safetext.Text`, never less | `transcript.WithRules` | built |
| The notice when a reprint left output out | "(earlier output is not shown again)", unstyled | `transcript.WithCatalog`, `transcript.Messages`, `transcript.WithNoticeStyle` | built |
| When a reprint is made | once the last resize has settled and the host says its output is idle | `transcript.Transcript.HandleTimeout` | decided (0023-MADR-line-mode-components.md D9, A8) |

## inline and inline/inlinetest

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| How text is measured | `ansi.WcWidth` | `inline.WithMethod` | built |
| Each drawing one synchronized update, the cursor hidden | on | `inline.WithSynchronized` | built |
| A cursor query on a resize | `CSI 6 n` sent | `inline.WithoutCursorQuery` | built |
| The wait for the cursor report | 150 ms | `inline.WithReportWait`, `inline.DefaultReportWait` | built |
| How the terminal re-wraps, when the report does not say | a re-wrap | `inline.WithReflow` | built |
| The rows shown | every row | `inline.WithHeight`, `inline.Region.SetHeight` | built |
| A row wider than the width | cut to it | `inline.Region.Resize` | floor |
| The cursor report, while a resize asks for one | taken from the input, the first row's too; a late one dropped | `termcap.ParseReply` | floor |
| Reading the input | only when asked, one read at a time, never closed | `inline.NewReader` | decided (0023-MADR-line-mode-components.md D3) |

## msgcat

| Behaviour | Default | Control | Status |
| :--- | :--- | :--- | :--- |
| A message's text | its package's English default | `msgcat.Catalog`, `msgcat.Table`, `msgcat.CatalogFunc` | built |
