---
status: accepted
date: 2026-10-25
decision-makers: the repository owner
consulted: "an audit of every non-test Go file in the public packages at 65d437b, by package group: command, command/commandtea, keymap, keymap/keymaptea, launch, launch/launchtest and when; workspace, palette, theme, glyph, layout and tuitest; termcap, termcap/termcaptea, termcap/termcaptest, termsvc, termsvc/termsvctea, termkey, safetext, fuzzy and internal/sanitize"
informed: gobble-cli and pi-go, the library's consumers
---
# Let a program with its own CLI refuse, replace or opt into every behaviour the library imposes, with the floors stated and an inventory the tests hold

## Context and Problem Statement

On 2026-10-25 the owner said:

> i want to provide programs with their own cli layer maximum
> compatibility and flexibility regarding the use of the go-tui-lib API to
> improve standards and best practices adherence, while also allowing cli
> programs to manage their own custom cli layer without having to fight
> with the go-tui-lib implementation to keep their own features and
> functionality. this is best achieved by ensuring opt-out and opt-in
> coverage across the api and command and control surfaces.

[0012-MADR-bring-your-own-cli.md](0012-MADR-bring-your-own-cli.md) made
the library a TUI layer a program stacks on its own CLI.
[0013-MADR-cli-integration-helpers.md](0013-MADR-cli-integration-helpers.md)
and [0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md)
gave every package native forms: flag types, the caller's streams,
byte-returning forms, opaque `With…`, `Without…` and `On…` options, hooks
as interfaces. No record has asked, of each behaviour the library
performs on a program's behalf, whether the program can turn it off,
replace it, or must opt into it. That is this record's question.

### What was audited

Every non-test `.go` file of the public packages, at `65d437b`, in three
groups, against one rubric: for each default the library applies, each
command it registers or runs, each key it binds, each query, mode or
write it sends to the terminal, each policy it decides, each limit, each
string it shows a user, each goroutine or timer it starts, and each read
of the environment or the streams, the default with its file and line,
the API that turns it off, replaces it or turns it on, and the gap. The
three findings below were then checked against the code.

### What the library already does well

* **No package reads the process's environment, writes to `os.Stdout` or
  `os.Stderr`, installs a signal handler, exits, or starts a process**
  (AGENTS.md, TUI conventions 1 and 2, API convention 9;
  `internal/conformance`). `launch` reads variables only through the
  `Streams.Env` it is given.
* **Every option function follows the `With…`, `Without…`, `On…` rule** in
  the command, keymap, launch, when, workspace, palette, theme, glyph,
  layout and tuitest packages, with two exceptions in `termcap` and
  `termsvc` (below).
* **Most defaults already have their control:** the workspace's keys
  (`WithBindings`, `SetBindings`), its background query
  (`WithoutBackgroundQuery`), its theme (`WithTheme`, `WithThemeBuilder`,
  `WithProfile`, `WithBackground`, `WithGlyphs`), its width method
  (`WithWidthMethod`), its mouse (`WithoutMouse`); the registry's gate,
  auditor and loop (`WithGate`, `WithAuditor`, `WithLoop`); the probe's
  timeout, extra queries, overrides and hooks (`WithTimeout`,
  `WithQuery`, `WithOverride`, `WithAppearanceHook`, `WithConsoleHost`);
  the notifier's protocol, policy and backend; the palette's limit,
  recents, placeholder and logger. The keymap binds nothing itself: a
  program passes its own defaults to `keymap.New` (`keymap.go:214`).

### Where a program has to fight the library

The command-and-control surfaces:

1. **The workspace's own commands bypass the program's registry.** A key
   bound to a `workspace.*` command runs that command's handler in place,
   with `context.Background()` (`workspace/bindings.go:259-273`): no gate,
   no auditor, no middleware a program put on its registry, and no way to
   replace a `workspace.*` handler. Unbinding the key is the only escape.
2. **The registry's permission table is fixed.** `asks` decides which
   danger and origin ask the gate: a mutating command from an agent, a
   destructive one from an agent or the shell (`command/gate.go:81-89`).
   Nothing replaces the table, so no program can gate a destructive
   command from a key, the palette or a slash line, or stop its own CLI's
   destructive requests from asking. `Request.Gate` is consulted only
   where the table already asks (`gate.go:108-115`).
3. **Remembered verdicts cannot be forgotten.** `AllowAlways` and
   `RejectAlways` are kept for the registry's life, by ID and caller
   (`gate.go:116-132`); nothing lists or forgets them.
4. **Every registry holds three built-in commands,** `command.list`,
   `command.describe` and `app.quit`, added by `NewRegistry` with no
   option to leave them out (`command/registry.go:157-174`). They reach
   `MCPTools`, `Manifest` and completion. `app.quit` is `UI`, offered on
   every surface but the shell, so an agent may call it with no gate
   (`builtin.go:58`). Their IDs are unexported, so `Registry.Remove`
   takes literal strings.
5. **`launch.Decide`'s rules cannot be changed piecemeal.** `TERM=dumb`
   is plain output even under `ChoiceTUI`, and the `CI` variable is read
   by a fixed name (`launch/decide.go:208-216`). A program may build its
   own `Decision`, but the colour and glyph helpers it would need are
   unexported (`launch/colour.go:19`, `:38`; `locale.go:18-31`).
6. **`launch.Run` writes the model's `Epilogue()` to `Out`** on a clean
   end, with no option to suppress it or choose its stream
   (`launch/run.go:131-137`).
7. **The palette's own command IDs and scope are unexported,** so a
   program cannot bind them under its own names, and a completion pop-up
   opened under another ID gets no default keys (`palette/keys.go:13-29`;
   `picker.go:123`). A program that wraps `FromRegistry` or `Panes`, to
   rename it, loses its `Discover` and its pane list, because the hook
   that tells a provider the picker opened is unexported
   (`palette/picker.go:96-100`).

The terminal surfaces:

8. **The probe's batch is all or nothing.** `WithDisabled` sends nothing;
   `WithoutBackgroundRequest` and `WithoutModeQuery` drop OSC 11 and the
   DECRQMs; nothing drops `CSI ? u`, DSR 996 or DA2 alone, and nothing
   forces the gated queries off where the heuristic would send them
   (`termcap/queries.go:46-81`; `probe.go:202-216`; `options.go:28-88`).
9. **The capability views come from brand tables** with no direct
   override: OSC 8 links, OSC 9 and 777 notifications, the keyboard
   request, `LinkDisplay`, `Pointer`, `ProgressSupported`
   (`termcap/views.go:26-127`; `termsvc/links.go:42`; `beacon.go:134-159`).
   Overriding the terminal's brand changes all of them at once.
10. **`termsvc`'s limits and routes are fixed:** a notification's title
    and body cells (`termsvc.go:107-110`); `MaxCopyBytes` checked before a
    program's own clipboard backend (`clipboard.go:16`, `:130`); the
    copy's route inside tmux (`clipboard.go:144-163`); the link schemes
    (`termsvc.go:54-74`); and the bell as the last notification protocol
    under `Auto` (`notify.go:347-361`).
11. **`termcap.Env.LookupEnv` folds case by `runtime.GOOS`,** not by
    `WithGOOS` (`termcap/env.go:21`), so a server probing for a Windows
    client reads its variables with the server's rule.

The presentation surfaces:

12. **Text a user reads is fixed English:** the palette's help line,
    counts, argument reasons, failure and limit rows
    (`palette/view.go:91-220`, `args.go:133-194`, `files.go:215-220`); the
    workspace's command results and help descriptions
    (`workspace/commands.go:161`, `:358`; `help.go:20-23`); the built-in
    commands' output (`command/builtin.go:30-49`); the MCP error text an
    agent reads (`command/mcp.go:110-126`); the doctor report's findings
    (`termcap/findings.go:56-120`).
13. **Mouse handling is all or nothing** (`WithoutMouse`): click to focus,
    drag to resize, a modal overlay swallowing clicks outside it, and the
    palette's click to accept cannot be turned off one at a time
    (`workspace/render.go:488-521`; `palette/picker.go:488-500`).
14. **An overlay always has chrome:** a bordered box with a title, at
    least 3 by 3 (`workspace/render.go:71`; `overlay.go:188`).

Conventions:

15. **Two option names break the rule:** `termcap.WithDisabled` turns a
    default off under a `With…` name (`options.go:51`), and
    `termsvc.WithNotifyFilter` is a callback (`notify.go:202`).
16. **Hooks with no `…Func` adapter:** `launch.Restorer`, `when.Context`;
    and hooks given as bare functions: `command.PromptGetter`,
    `WithPrefixer`, `launch.WithFilter` and `OnStart`, `termcap`'s
    `WithOverride`, `WithAppearanceHook` and `WithConsoleHost`,
    `termsvc.WithNotifyFilter`.

### What is not a gap

Some fixed behaviour is what keeps a program, its user or its terminal
safe, or is a protocol both sides must agree on. Making it optional would
let one misconfiguration defeat the library's guarantees:

* the rules no package breaks: the environment, the streams, the signals,
  the exit, the process (`internal/conformance`);
* cleaning untrusted text where the library draws it: a pane's title,
  badge and view, a hit's text, a terminal's reply
  (`workspace/render.go:174`, `:283-355`; `palette/engine.go:411-412`;
  `termcap/env.go:50-52`), invalid UTF-8 as U+FFFD, and the invisible and
  bidirectional controls safetext removes (`safetext/escape.go:15-25`);
* bounds on untrusted input: a reply's 1 KiB (`termcap/reply.go:114`), a
  sequence's 4 KiB and a paste's 1 MiB (`termkey/decode.go:18-21`), a
  request's argument bytes (`command/registry.go:67`), a when expression's
  4096 bytes and depth 64 (`when/when.go:43-44`), the window's
  `MaxSide` and `MaxCells`, a query's 256 clusters (`fuzzy/options.go:67`);
* grammars and protocols: key names and the notation, the when grammar,
  the command ID grammar, which keybindings files and MCP tool names both
  read, the probe's DA1 sentinel;
* a command's declared danger, and a handler's panic recovered into an
  error, so a command cannot take the UI down.

### The question

What rule makes every behaviour the library performs on a program's
behalf something the program can refuse, replace or choose, which
behaviours are floors no option turns off, how does the library keep that
true as it grows, and which of the gaps above close first?

## Decision Drivers

* **The program owns its CLI and its policy** (0012-MADR). The library
  supplies defaults, never decisions a program cannot reverse.
* **The command-and-control surfaces first,** as the owner named them: the
  registry's policy, its built-ins, the workspace's own commands, the
  launch decision, the palette's commands.
* **Go idiom:** additive options over flag-laden constructors; a zero or
  nil value meaning "the default"; hooks as small interfaces with a
  `…Func` adapter; the lower layer of every convenience exported, as
  `net/http` exports `Transport` under `Client` and `RoundTripper` under
  that.
* **Compatibility:** `v2` is stable. Nothing is removed or retyped; a
  default that changes behaviour does not change in a minor release.
* **Safety holds:** no option weakens the floors.
* **The rule stays true:** a gate catches a default added without its
  control, as `internal/conformance` catches a package that reads the
  environment.
* **No option for its own sake:** a value that is a protocol, a grammar or
  a safety bound stays a constant.

## Considered Options

* **A. A rule in AGENTS.md for every behaviour the library imposes, a
  stated list of floors, an inventory of every default with its control
  that a test holds to the code, and additive APIs closing the gaps, the
  command-and-control surfaces first.**
* **B. Close the audit's gaps one by one,** with no rule and no inventory.
* **C. Make every constant an option,** every limit, string and timeout.
* **D. A single "plain" or "minimal" switch per package** that turns all
  its defaults off at once.
* **E. Document the escape hatches only:** a guide of what to bypass, and
  no API change.

## Decision Outcome

*Accepted 2026-10-25 by the owner, who committed the record with its
answers and its PLAN and said "committed, proceed".*

*The owner answered Q1 to Q6 on 2026-10-25 (Owner questions, below):
each the recommendation but Q1, where a program's text comes from one
message catalogue. D6 and D9 say so.*

Chosen option: **"A"**, because it closes the gaps a program meets now,
keeps the guarantees a program relies on, and keeps the library from
growing new gaps. B closes today's gaps and reopens them with the next
feature; C turns safety bounds and protocols into misconfigurations and
bloats every constructor; D is too coarse, since a program wants to keep
the workspace's focus keys and drop its mouse drag, not choose between
everything and nothing; E leaves the program fighting the library, with
a map.

The decision has nine parts, D1 to D9.

### D1. The rule: API convention 12

AGENTS.md's "API conventions" gains a twelfth rule:

> **Defaults are the program's to refuse.** Everything the library does
> on a program's behalf has a documented default and a control a program
> can use without forking:
>
> * a key it binds, a command it registers or runs, a query or mode it
>   sends to the terminal, a policy it applies, a goroutine or timer it
>   starts for the program, and text it shows a user: a `Without…` option,
>   a nil or zero value, or a hook that replaces it;
> * a value a program may legitimately need otherwise, such as a timeout,
>   a size, a cap on the program's own data: a `With…` option, whose zero
>   or negative value keeps the default;
> * a behaviour added after `v2.1.0` that writes to the terminal, acts on
>   the user's behalf, or runs a program's command: off until a `With…` or
>   `On…` option turns it on, unless its record says why it is on;
> * every convenience built on a lower layer: that layer exported, so a
>   program does the convenience its own way.
>
> The floors in the defaults guide, `docs/guides/defaults.md`, are the
> exceptions, each with its reason. A record that adds a default lists it
> there, with its control.

### D2. The floors

The library keeps these whatever a program sets, because each protects
the program, its user or its terminal, or is a contract with another
party. A program that needs otherwise does the work itself, through the
exported lower layer.

| Floor | Why | Lower layer a program may use instead |
| :--- | :--- | :--- |
| No read of the environment, no write to `os.Stdout` or `os.Stderr`, no signal handler, no exit, no process | the program owns its process (AGENTS.md TUI 1, 2; API 9) | its own code, with `termcap.Env` |
| Untrusted text cleaned where the library draws it | a title, a model's reply or a terminal's answer can carry escape sequences that act on the terminal | `safetext` with the program's own `Rules`, for text the program draws itself |
| Bounds on untrusted input | a hostile or broken peer must not exhaust memory or time | a `With…` option raises a bound where a legitimate use needs more, and never removes it |
| Grammars and protocols: key names, the notation, the when grammar, the command ID grammar, CommonMark, the DA1 sentinel | both sides of a file, a protocol or a keymap read them alike | none: they are the contract |
| A command's declared danger; a handler's panic recovered | the gate decides by danger; one handler must not take down the UI | `Command.Handler`, which a program writes |

A floor joins or leaves this list only by a record.

### D3. The inventory: `docs/guides/defaults.md`, held by a test

* **One guide lists every default** the library imposes: its package, the
  behaviour, the default, and its control, `Without…`, `With…`, a hook,
  or "floor" with the floor's row. It starts from this record's audit.
* **`internal/conformance` gains `TestDefaultsInventory`.** It reads the
  guide's tables, and for every control named in backticks, such as
  `workspace.WithoutMouse`, finds the exported identifier in the
  type-checked package, as the conformance test already resolves every
  use. A name that does not exist fails as stale, so a rename cannot leave
  the guide wrong.
* **A new default without its row fails review,** by rule 12; the test
  cannot find a missing row, and does not claim to.

### D4. The command-and-control surfaces first

Each is an addition; each default stays as it is.

| Gap | The addition |
| :--- | :--- |
| 1. The workspace's own commands bypass the registry | `workspace.WithoutOwnCommands()`: every matched command, `workspace.*` included, goes to a `CommandConsumer` or comes back as a `RequestMsg`, so a program that registers `workspace.Commands(ws)` in its registry runs them through its gate, auditor and middleware. The workspace still holds no registry, as 0007-MADR A11 decided. |
| 2. The permission table is fixed | `command.WithGatePolicy(GatePolicy)`, with `type GatePolicy interface{ Asks(c Command, o Origin) bool }`, `GatePolicyFunc`, and `DefaultGatePolicy()`, today's table. A program may gate every destructive command from every origin, or none from its own CLI. (`Policy` is `termsvc.Policy`.) |
| 3. Remembered verdicts cannot be forgotten | `Registry.Forget(id ID, caller string)` and `Registry.ForgetAll()`. |
| 4. Built-ins are forced | `command.WithoutBuiltins()`, and exported `BuiltinList`, `BuiltinDescribe` and `BuiltinQuit` IDs, so `Remove` and a program's own replacements name them. With `WithGatePolicy`, `app.quit` can be gated. |
| 5. `Decide`'s rules | `Config.AllowDumb` and `Config.IgnoreCI`, false by default, so today's decision stands; and `launch.ColorProfile(Streams)` and `launch.GlyphTier(Streams)` exported, the helpers `Decide` uses, so a program that builds its own `Decision` builds it as `Decide` would. |
| 6. The epilogue | `launch.WithoutEpilogue()` and `launch.WithEpilogueWriter(io.Writer)`. |
| 7. The palette's commands and providers | the picker's command IDs exported as constants; `palette.CompletionRules(id)`, the `overlay:<id>` bindings for a pop-up under another ID; and `palette.Opener`, the exported form of the hook that tells a provider the picker opened, so a program's wrapper forwards it, as `Recents` does. |

### D5. The terminal surfaces

| Gap | The addition |
| :--- | :--- |
| 8. The batch is all or nothing | `termcap.WithoutQuery(names ...string)`, by the query's `Name`, which every built-in query gets; DA1 is the protocol's sentinel and is not named. `WithoutGatedQueries()`, the inverse of `WithoutHeuristic`. `WithoutQueries()` is `WithDisabled` under the rule's name (D7). |
| 9. Views from brand tables | `Caps` gains a fact for each derived view, `LinksSupport`, `NotifySupport`, `KeyboardRequest`, `PointerSupport`, `ProgressSupport`, read first by its view, and set through `WithOverride`, so a program overrides one view without changing the brand. |
| 10. `termsvc`'s limits and routes | `WithTitleCells` and `WithBodyCells`; `MaxCopyBytes` applied only to the OSC 52 route, not to a program's `WithClipboard` backend; `WithCopyRoute(Route)`; `WithLinkSchemes`; `WithoutBell()`, so `Auto` never falls back to the bell. |
| 11. Case folding by `runtime.GOOS` | `Env.LookupEnv` folds by the probe's GOOS, `WithGOOS`'s when given: a correction, noted in the release, since it changes only a probe whose GOOS a program set to another system. |

### D6. The presentation surfaces

| Gap | The addition |
| :--- | :--- |
| 12. Fixed English text | The message catalogue, D9 (Q1): every string a package shows a user, or an agent, has an ID and an English default, and a program's `msgcat.Catalog` gives its own. |
| 13. Mouse all or nothing | `workspace.WithoutMouseFocus()` and `WithoutMouseResize()`, each one behaviour; `palette.WithoutClickAccept()`, a click selecting without accepting. |
| 14. Overlay chrome | `Overlay.Bare`, a field whose zero value is today's chrome: a bare overlay draws its pane alone. |

### D7. Conventions

* **`termcap.WithoutQueries()`** is added, and `WithDisabled` is marked
  `// Deprecated: use WithoutQueries`, kept until `v3` (API convention 1).
* **`launch.RestorerFunc` and `when.ContextFunc`** are added, the
  adapters rule 7 asks for.
* **The bare-function hooks stay as they are.** `PromptGetter`,
  `WithPrefixer`, `WithFilter`, `OnStart`, `WithOverride`,
  `WithAppearanceHook`, `WithConsoleHost` and `WithNotifyFilter` take
  functions; retyping any of them breaks `v2`. Each new hook follows rule
  7, and a `v3` may move these.

### D8. Line-mode components follow the rule from the start

[0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md)
gains D13: each of its components lists its defaults in the guide, with
their controls, before it is built. Three of its behaviours gain controls
by this record: the live region's cursor query on a resize
(`inline.WithoutCursorQuery()`, which leaves `WithReflow`'s kind), the
transcript's scrollback clear (`transcript.WithoutScrollbackClear()`,
`CSI 2 J` without `CSI 3 J`), and the rules text is cleaned by
(`WithRules(safetext.Rules)`, never below the floor's `Line`).

### D9. The message catalogue (Q1)

A program gives one catalogue, and every package that shows text reads
it. A new core package, `msgcat`, holds the shape; each package keeps its
own messages and their English defaults.

```go
package msgcat // the standard library only

// MessageID names one message, "<package>.<name>": "palette.no-results".
type MessageID string

// Catalog gives a message's text in the program's words, or false to
// keep the package's default.
type Catalog interface{ Message(id MessageID) (string, bool) }
type CatalogFunc func(id MessageID) (string, bool)

// Table is a Catalog read from a map, as a program loads one from a file.
type Table map[MessageID]string

// Format is c's text for id, or def, with args placed by fmt.
func Format(c Catalog, id MessageID, def string, args ...any) string
```

* **Each package exports its messages:** a constant per ID
  (`palette.MsgNoResults`), its defaults as `Messages() msgcat.Table`, so
  a program starts a translation from them, and a `WithCatalog(c)` option
  on each component or registry that shows text. No package keeps a
  catalogue in a package variable (0014-MADR H11): the program passes it.
* **Arguments are fmt's,** with explicit indexes where a message has more
  than one (`%[1]d of %[2]d`), so a translation may reorder them.
  `msgcat.Check(defaults, c Catalog) error` reports each message whose
  verbs differ from its default's, for a program's own test.
* **What is in it:** the palette's help line, counts, argument reasons
  and limit rows; the workspace's command results and help descriptions;
  the built-in commands' output; the MCP error text an agent reads, which
  a program may phrase for its model; the doctor report's findings; and
  0023's components' prompts, counts and notices.
* **What is not:** plural rules, dates and numbers in a locale's form.
  Each would need `golang.org/x/text`, a module a later record must name;
  a message whose form changes with its number has one ID per form where
  the package needs it.
* **The closure:** `msgcat` imports the standard library alone, and joins
  the cores and the narrow closures beside `termkey`
  ([0021-MADR-termkey-closure-gate.md](0021-MADR-termkey-closure-gate.md)).
* **A test in each package** holds its `Messages()` to its `MessageID`
  constants, both ways, and the guide's inventory lists each package's
  messages by its `Messages` function.
* **Names:** `python3 -m tools.verify names` over `Catalog`,
  `CatalogFunc`, `MessageID` and `Table`: clean, after `Map` was renamed
  (`when.Map`).

### Consequences

* Good, because a program can run the workspace's commands through its
  own registry, set its own permission policy, leave out the built-ins,
  decide its own launch rules with the library's helpers, and keep every
  string it shows in its own words.
* Good, because the floors are written down, so a program knows what the
  library will not do for it, and why.
* Good, because the inventory and its test keep the rule true as packages
  grow, and give a program one page of everything it may turn off.
* Good, because every change is an addition: `make apicheck` stays at 0
  incompatible changes, and no default changes in a minor release.
* Neutral, because `WithDisabled` and the bare-function hooks stay in
  `v2`, beside their successors or as they are.
* Good, because one catalogue, which a program loads once, translates or
  rewords every package, and a translator works from one list of IDs.
* Bad, because the API grows by about sixty names, and every message a
  package adds is an ID and a default to keep in step.
* Bad, because the catalogue has no plural rules or locale formats; a
  program that needs them formats those messages itself.
* Bad, because the inventory is one more document every record that adds
  a default must change.

### Confirmation

* `TestDefaultsInventory` passes, and fails on a scratch copy whose guide
  names a control that does not exist.
* Each addition of D4 to D7 has a test that the default is unchanged
  without it, and changed with it, each seen failing on a scratch copy.
* `workspace.WithoutOwnCommands()`, with `workspace.Commands(ws)` in a
  registry whose gate records what it is asked, sends a `workspace.*` key
  through that gate.
* `make apicheck` reports 0 incompatible changes at each release.
* A consumer smoke test builds a program with its own registry, policy,
  launch decision and text, from the proxy.

## Pros and Cons of the Options

### A. A rule, the floors, an inventory and a test, then the gaps

* Good, because it closes today's gaps and keeps new ones from opening.
* Good, because it keeps safety bounds and protocols fixed, for stated
  reasons.
* Bad, because it is the most work: a guide, a test, and the additions.

### B. Close the gaps, no rule

* Good, because it is only the additions.
* Bad, because the next default arrives without a control, as these did.

### C. Every constant an option

* Good, because nothing is fixed.
* Bad, because a program can raise a reply bound to unlimited, or change a
  grammar its keybindings file shares, and the library's guarantees end.
* Bad, because every constructor grows options nobody needs.

### D. One "minimal" switch per package

* Good, because it is one name to learn.
* Bad, because a program cannot keep one default and drop its neighbour.

### E. Document the escape hatches

* Good, because it changes no code.
* Bad, because the escape hatches are forks: re-implementing `Decide`'s
  helpers, re-registering built-ins, unbinding keys.

## Owner questions

*Answered 2026-10-25,* each picked from three options: Q1 "One message
catalogue"; Q2 "WithoutOwnCommands (Recommended)"; Q3 "A GatePolicy
interface (Recommended)"; Q4 "Stays a floor (Recommended)"; Q5 "D4, then
0023, then D5-D7 (Recommended)"; Q6 "A guide a test reads
(Recommended)". Q1 is not the recommendation: D9's catalogue replaces
the `Text` struct per package that D6 first gave.

* **Q1. User-visible text.** Recommended: a `Text` struct and `WithText`
  in each package that shows text, empty fields keeping the defaults. The
  alternatives are one message catalogue for the whole library, keyed by
  ID, which adds a localisation layer, or leaving text fixed, out of this
  record.
* **Q2. The workspace's own commands.** Recommended:
  `WithoutOwnCommands()`, every matched command offered to the program.
  The alternatives are `workspace.WithRegistry`, which 0007-MADR A11
  refused, or an `OnOwnCommand` hook that sees each before it runs.
* **Q3. The permission policy.** Recommended: a `GatePolicy` interface
  with `GatePolicyFunc` and `DefaultGatePolicy()`. The alternatives are a table option,
  `WithAsks(map[Danger][]Origin)`, or keeping the table and widening only
  `Request.Gate`.
* **Q4. The pane view filter.** Recommended: it stays a floor: a pane's
  view keeps SGR and OSC 8 only. The alternative is an opt-in per pane,
  a `TrustedViewer` interface whose view keeps an allowlist of further
  sequences, such as OSC 133's prompt marks.
* **Q5. Order with 0023.** Recommended: D4 first, then 0023's steps with
  D8 applied, then D5 to D7, each released as it lands. The alternatives
  are all of this record before 0023, or 0023 first.
* **Q6. The inventory's home.** Recommended: `docs/guides/defaults.md`,
  read by `TestDefaultsInventory`. The alternative is a Go table in
  `internal/conformance` that generates the guide.

## More Information

* The audit, at `65d437b`, by package group, read 2026-10-25. Its three
  command-and-control findings were checked against the code:
  `workspace/bindings.go:259-273`, `command/gate.go:81-89` and
  `command/registry.go:157-174`; and `launch/run.go:131-137`,
  `termcap/options.go:51`, `launch/colour.go:19-38`.
* Records here:
  [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) (A11: the
  workspace runs its own commands without a registry);
  [0012-MADR-bring-your-own-cli.md](0012-MADR-bring-your-own-cli.md);
  [0013-MADR-cli-integration-helpers.md](0013-MADR-cli-integration-helpers.md);
  [0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md)
  (W0's conventions, W2's native forms);
  [0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  (the cores, `safetext`);
  [0023-MADR-line-mode-components.md](0023-MADR-line-mode-components.md),
  which D8 amends.
* Go's standard library exports the layer under each convenience:
  `net/http.Client` over `Transport` and `RoundTripper`; `log/slog`'s
  `Logger` over `Handler`; `encoding/json/v2`'s options over
  `jsontext`.
* Names: `python3 -m tools.verify names` over the planned type names
  found one clash, `command.Policy` with `termsvc.Policy`, renamed
  `GatePolicy`; `WithCopyRoute` takes the existing `termsvc.Route`.
* No PLAN is written yet: the owner asked for the decision first.
