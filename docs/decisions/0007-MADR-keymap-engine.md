---
status: proposed
date: 2026-10-02
decision-makers: owner
consulted: 0003-REPORT-agent-tui-ecosystem-research.md (§3, §4, §6, §7); Charm v2 APIs (bubbletea v2.0.10, bubbles v2.2.1, the ultraviolet revision in go.sum); VS Code keybindings and when-clause documentation; codex, gemini-cli and opencode keymap sources
informed: pi-go; go-core-lib
---
# Bind keys to command IDs through a context-aware keymap engine, with chords, a leader key and VS Code-format user keymaps

## Context and Problem Statement

On 2026-10-01 the owner asked for the next round of go-tui-lib work:

> i want to stay working on go-tui-lib, i want to ensure the
> bubbletea/lipgloss/charm stack is tightly integrated, coded idiomatically,
> based on go1.27.1 optimizations and standards. i want to enhance and expand
> the tui library functionality in 5 more ways that will bring benefit and
> value to the codebase functionality. i want a large, openstandards api
> surface, with a wide assortment of agentic commands available through the
> terminal UI and also the TUI. look at similar opensource projects across the
> web. dig into how they solved interesting problems and issues bringing
> functionality to reality. find ideas, features, enhancements, and
> optimizations. present 10 items to spec out and plan. future-proof is a
> focus.

Ten items were presented. On 2026-10-02 the owner answered:

> write findings into a report then follow recommendations and proceed.

The recommendation included item 4, a keymap engine. The research is in
[0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md).
The owner's standing direction also applies: build speculative API when it
is sensible, for extensibility, flexibility and idiomatic, modular design.

[0006-MADR-command-registry.md](0006-MADR-command-registry.md) makes
`command.ID` the one name for every action. It also adds the `when`
expression language. This record decides how keys reach those IDs.

Evidence (read-only, 2026-10-02):

* **What go-tui-lib has today.**
  * `workspace.KeyMap` (`workspace/keys.go`) is a struct of
    `key.Binding` fields: focus next and previous, focus pane 1 to 9,
    zoom, four resize directions and close. `Workspace.key` tests each with
    `key.Matches`, in a fixed order.
  * There are no contexts, chords, leader key or user configuration, and
    nothing detects two bindings on one key.
  * `KeyMapper` lets a pane list its bindings, but nothing reads it
    (0003-REPORT §1, finding 10).
  * The `alt+[` and `alt+]` defaults collide with escape-sequence prefixes
    on terminals without key disambiguation (0003-REPORT §1, finding 9).
    [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
    replaces them.
* **bubbles v2.2.1** (read in the module cache):
  * `key.Binding` holds keys, help text and a disabled flag. Its methods
    are `Keys`, `Help`, `Enabled`, `SetEnabled`, `SetKeys`, `SetHelp` and
    `Unbind`.
  * `key.Matches[Key fmt.Stringer](k Key, b ...Binding)` compares
    `k.String()` with each key string exactly. A binding is enabled only
    when it is not disabled and its key list is not nil.
  * `help.KeyMap` is `ShortHelp() []key.Binding` and
    `FullHelp() [][]key.Binding`. Help skips disabled bindings.
* **bubbletea v2.0.10 keys** (read in the module cache):
  * `KeyPressMsg` and `KeyReleaseMsg` are both `Key`. Its fields are `Text`,
    `Mod`, `Code`, `ShiftedCode`, `BaseCode` and `IsRepeat`. `ShiftedCode`
    and `BaseCode` come only from the Kitty keyboard protocol or the Windows
    console.
  * `Key.Keystroke()` writes modifiers in the order ctrl, alt, shift, meta,
    hyper, super, then the key name or rune. Its doc comment's example
    (`ctrl+shift+alt+a`) contradicts the code, which writes `alt+` before
    `shift+`. This record follows the code.
  * `Keystroke()` uses `BaseCode` in place of `Code` when it is set, so on
    a Kitty terminal it names the US-layout position, not the typed letter.
  * `Key.String()` returns `Text` when it is printable and not a space,
    and `Keystroke()` otherwise. `key.Matches` therefore sees `A` for
    shift+a and `?` for shift+/, but `ctrl+a` for ctrl+a.
  * Key names come from ultraviolet's table: `enter`, `tab`, `backspace`,
    `esc`, `space`, the arrows, `insert`, `delete`, `pgup`, `pgdown`,
    `home`, `end`, and `f1` to `f63`. Its parser also accepts `escape`.
  * A legacy terminal's uppercase `A` is decoded as `Code` `a`, `Mod`
    shift, `ShiftedCode` `A` and `Text` `A`.
  * Bubble Tea always requests Kitty flag 1, key disambiguation
    (`keyboardEnhancementsFlags` in `cursed_renderer.go`). A view can add
    event types, alternate keys, all keys as escape codes and associated
    text. A terminal that supports any of them answers with
    `KeyboardEnhancementsMsg{Flags}`. `KeyReleaseMsg` arrives only with
    event types.
  * `tea.Tick(d, fn)` starts a `time.Timer` when it is called and returns a
    `Cmd` that waits on it.
* **The standard user format: VS Code `keybindings.json`.**
  * Each rule has `key`, `command`, an optional `when` and optional `args`.
  * A chord is two key presses separated by a space, as in `ctrl+k ctrl+c`.
  * A `-` before the command removes a binding:
    `{"key": "tab", "command": "-jumpToNextSnippetPlaceholder"}`.
  * Rules are evaluated from bottom to top, and the first rule that matches
    both the key and the `when` clause wins.
  * Modifiers are `ctrl`, `shift`, `alt` and `cmd` on macOS, `win` on
    Windows and `meta` on Linux. Scan codes are written `[KeyA]`.
  * `when` supports `!`, `&&`, `||`, `==`, `!=`, `>`, `>=`, `<`, `<=`,
    `=~`, `in`, `not in` and parentheses.
  * Sources: <https://code.visualstudio.com/docs/configure/keybindings> and
    <https://code.visualstudio.com/api/references/when-clause-contexts>.
* **Prior art in agent TUIs** (scratch clones, 0003-REPORT §3):
  * **codex** (`config/src/tui_keymap.rs`, `tui/src/keymap.rs`):
    * named contexts: global, chat, composer, editor, the vim modes, pager,
      list and approval;
    * precedence context, then global, then defaults;
    * an empty list unbinds, and does not fall through to the defaults;
    * duplicate keys in one effective context, and keys that shadow each
      other along one focus path, are rejected with errors that name the
      config path;
    * chords have at most two strokes, and a pending chord expires after
      one second or when its context changes.
  * **gemini-cli** (`ui/key/keyBindings.ts`):
    * a `keybindings.json` that allows comments;
    * a `-command` entry removes a default binding, and removing a key that
      is not bound is an error;
    * a user binding is put first, so the UI shows it;
    * every error is collected, rather than failing on the first.
  * **opencode** (`tui/src/config/keybind.ts`, `tui/src/keymap.tsx`):
    * a leader key (default `ctrl+x`) written `<leader>q`, with a
      configurable `leader_timeout`;
    * alternatives separated by commas;
    * `"none"` or `false` to unbind;
    * an optional `event` of press or release.
  * **Elsewhere** (0003-REPORT §4 and §6):
    * Helix writes sequences as nested TOML tables;
    * lazygit writes `<disabled>` to remove a key;
    * gh-dash writes keys as ultraviolet key strings, the notation Bubble
      Tea already prints.

## Decision Drivers

* **One name per action.** A key, a palette entry, a slash command and a
  shell command all name the same `command.ID`
  ([0006-MADR-command-registry.md](0006-MADR-command-registry.md)).
* **Bubbles components keep working.** `textarea`, `viewport`, `list` and
  `help` take `key.Binding` and `help.KeyMap`. The engine must produce
  both.
* **The terminal decides what can be bound.** `ctrl+i` and `tab` are the
  same byte on a legacy terminal. Defaults must follow what the terminal
  reports.
* **Users edit a standard file,** with errors that say where the mistake
  is.
* **Conflicts are found when the keymap is built,** not when a user
  presses the key.
* **No goroutines and no wall clock.** Chord timeouts are Bubble Tea
  commands, and tests drive them without sleeping.
* **The program owns `ctrl+c`** (0001-MADR §6, rule 2). No library default
  binds it, and a program can reserve keys a user cannot take.
* **No new module.** The standard library, bubbletea and bubbles are
  enough.

## Considered Options

* **A. A `keymap` package that binds key sequences to `command.ID` in named contexts, with layered defaults, a chord matcher, a leader key, VS Code-format loading and bubbles adapters.**
* **B. Keep `key.Binding` structs per component,** and add a loader that maps a config file onto their fields.
* **C. VS Code's resolver alone:** one flat rule list, evaluated bottom to top, with focus expressed only through `when` clauses.
* **D. Each program writes its own keymap,** and go-tui-lib keeps only `workspace.KeyMap`.

## Decision Outcome

Chosen option: **"A"**, because:

* it is the only option that binds keys to the registry's IDs;
* it detects conflicts along a focus path before a key is pressed;
* it still gives bubbles components the `key.Binding` values they take.

Contexts give precedence and conflict checks that `when` alone cannot. A
`when` clause is a runtime test, so two rules guarded by different
expressions cannot be proven disjoint when the keymap is built.

### 1. The package and its imports

```text
 keymap    notation, rules, contexts, build and conflict checks, the chord
           matcher, the VS Code loader and exporter, JSON Schema, and
           bubbles adapters   → command (ID), when, bubbletea, bubbles/key,
                                bubbles/help, encoding/json/v2, jsontext
```

* `keymap` needs `command.ID`, and `Registry.Lookup` for the
  `FromRegistry` adapter of §7. It never runs a command: the program
  passes the matched ID and `args` to the registry's `Dispatch`.
* `when` is the expression language
  [0006-MADR-command-registry.md](0006-MADR-command-registry.md) defines.
  Each rule's `when` is compiled once, when the keymap is built.
* `workspace` imports `keymap` (§9). `keymap` imports nothing from
  `workspace` or `layout`.
* It has no direct `termcap` import. `Features` (§5) is built from
  `tea.KeyboardEnhancementsMsg.Flags`, which `termcap.Caps` in
  [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md)
  also carries.

### 2. Notation

The canonical notation is Bubble Tea's own `Keystroke()` form. It is what
programs already write, what gh-dash uses, and what `tea` prints in logs.

```go
// Stroke is one key press, as a binding names it.
type Stroke struct {
    Mod   tea.KeyMod // ctrl, alt, shift, meta, hyper, super; never a lock key
    Code  rune       // a tea key constant (tea.KeyEnter) or a lower-case rune
    Scan  bool       // match the US-layout position (BaseCode), as VS Code's [KeyA]
    Event Event      // Press (the default), Release, or Repeat
}
type Sequence []Stroke // one stroke, a chord, or a leader sequence; at most MaxStrokes

const MaxStrokes = 4

func ParseStroke(s string) (Stroke, error)
func ParseSequence(s string, leader Sequence) (Sequence, error) // expands <leader>
func (s Stroke) String() string    // canonical: "ctrl+alt+shift+a"
func (q Sequence) String() string  // strokes joined by one space
func (q Sequence) VSCode() string  // VS Code spelling
func (s Stroke) Matches(k tea.Key) bool
func (s Stroke) BubblesKeys() []string // every k.String() this stroke can produce
```

* **Grammar.** Modifiers and a key name joined by `+`. Strokes in a
  sequence are separated by one space. `<leader>` stands for the leader
  sequence. Input is case-insensitive for modifier and key names.
* **Output** is always canonical: modifiers in `Keystroke()` order,
  ultraviolet's key names, and lower-case runes.
* **Normalization on input:**
  * an upper-case letter `A` becomes `shift+a`, as codex accepts;
  * a shifted symbol is written as the symbol it produces, such as `?`,
    because legacy terminals report the symbol and no shift;
  * lock modifiers are refused, because they are state, not a chord.
* **Matching** (`Stroke.Matches`) ignores lock modifiers on the message.
  A stroke matches in any of these cases:
  * `Mod` and `Code` are equal;
  * for a printable stroke without shift, the message's `Text` is the
    stroke's rune and its `Mod`, less shift, is equal;
  * for a `Scan` stroke, the message's `BaseCode`, or `Code` when there is
    no `BaseCode`, is equal.
* **The VS Code mapping is lossless** for every `Sequence`:
  `ParseSequence(q.VSCode())` returns `q` unchanged. Both spellings parse.

  | Canonical | VS Code | Note |
  | :--- | :--- | :--- |
  | `ctrl`, `alt`, `shift` | `ctrl`, `alt`, `shift` | `option` is accepted as `alt` |
  | `super` | `cmd` (also `win`) | the Command or Windows key |
  | `meta` | `meta` | a distinct modifier only with the Kitty protocol |
  | `hyper` | `hyper` | no VS Code name; kept as an extension |
  | `esc` | `escape` | |
  | `pgup`, `pgdown` | `pageup`, `pagedown` | |
  | `enter`, `tab`, `space`, `backspace`, `delete`, `insert`, `home`, `end`, the arrows, `f1`… | the same | VS Code stops at `f19` |
  | `[KeyA]`, `[Digit1]`, `[Slash]`… | the same | `Scan` strokes |
  | `ctrl+k ctrl+c` | `ctrl+k ctrl+c` | a chord |
  | `<leader>` | `<leader>` | no VS Code equivalent; an extension |

### 3. Rules, layers and contexts

```go
type Context string // "global", "workspace", "pane:transcript", "kind:editor", "overlay:palette", "mode:vim-normal"
const Global Context = "global"

// Path is the focus path, outermost first. The innermost context wins.
type Path []Context

type Layer uint8 // Default < Program < User

type Rule struct {
    Keys    Sequence
    Command command.ID
    Args    json.RawMessage // passed through to the command, as VS Code's args
    When    string          // a when expression; empty means always
    Context Context         // empty means Global
    Remove  bool            // VS Code's "-command"
    Origin  Origin          // where the rule came from, for errors
}
type Origin struct {
    Layer     Layer
    File      string           // "" for code
    Pointer   jsontext.Pointer // RFC 6901, such as "/3/key"
    Line, Col int
}

type Options struct {
    Leader   string        // "" means no leader; §6
    Timeout  time.Duration // pending chord or leader; default 1s
    Features Features      // §5
    Reserved []string      // keys only the program may bind, such as "ctrl+c"
    Tick     func(time.Duration, func(time.Time) tea.Msg) tea.Cmd // default tea.Tick
}

func Build(o Options, layers ...[]Rule) (*Keymap, error)
func (k *Keymap) Rebuild(f Features) (*Keymap, error) // same layers, new features
func (k *Keymap) Lookup(p Path, ctx when.Context, q Sequence) (Match, bool)
func (k *Keymap) Bindings(id command.ID, p Path) []Sequence // primary first
func (k *Keymap) Conflicts() []Conflict                     // warnings kept after Build
func (k *Keymap) Rules() iter.Seq[Rule]                      // effective rules, in order
```

* **Layers.** Packages and programs give defaults. A program can add its
  own layer. The user's file is the last layer. A later layer overrides an
  earlier one for the same key in the same context.
* **Removal.**
  * `Remove` with keys deletes that key from that command in that context.
    Removing a key that is not bound is an error, as in gemini-cli.
  * `Remove` with no keys removes every binding of that command in that
    context. VS Code has no such form; it is an extension, and it is how
    codex's empty list is written.
* **Resolution.** For a sequence, walk the path from the innermost context
  outward. In each context, try the layers from last to first, and the
  rules from last to first (VS Code's bottom to top). The first rule whose
  keys match and whose `when` holds wins.
* **Contexts come from the focus path,** not from `when`. A pane names its
  contexts through an optional interface (§8), and a pane can push a mode,
  such as vim normal mode, as a context. `when` stays for state that a
  context cannot express, such as `inputEmpty` or `agentBusy`.
* **`Keymap` is immutable after `Build`** and safe for concurrent reads. A
  reload builds a new one.

### 4. Conflicts, checked by `Build`

```go
type Conflict struct {
    Kind    ConflictKind // Duplicate, Prefix, Reserved, Unsupported, Shadow
    Keys    Sequence
    Context Context
    Rules   []Rule // the rules involved, with their Origin
}
func (c Conflict) Error() string // "keybindings.json:12:5 (/3/key): ctrl+k: ..."
```

* **Errors.** `Build` returns every error joined with `errors.Join`, not
  only the first:
  * **Duplicate:** two rules in one context and one layer bind the same
    sequence with the same `when` text.
  * **Prefix:** along one path, a sequence is also the start of a longer
    one, such as `ctrl+x` and `ctrl+x n`. The single key could never fire
    without a timeout guess. Owner question Q3 asks about a vim-style
    fallback.
  * **Reserved:** a user rule binds a key in `Options.Reserved`.
  * **Unsupported:** a rule needs a feature the terminal lacks (§5). For a
    user rule this is an error. A default with a fallback is skipped
    silently.
* **Warnings.** **Shadow:** an inner context, or a later layer, hides a
  binding. This is usually intended, so it is reported by `Conflicts()`
  and does not fail. A `doctor` or `keys` command can list it.
* **Every message names the rule's origin,** as file, line, column and
  JSON Pointer, as codex's errors name the config path.

### 5. Features and terminal-aware defaults

```go
type Features uint8 // Disambiguate, EventTypes, AlternateKeys, AllKeysAsEscapes, AssociatedText
func FeaturesFrom(flags int) Features // from tea.KeyboardEnhancementsMsg.Flags or termcap.Caps

// A default may need a feature, and name a fallback for terminals without it.
type Default struct {
    Rule
    Needs    Features
    Fallback Sequence // used when Needs is missing; empty means no binding
}
```

* Strokes that a legacy terminal cannot tell apart need `Disambiguate`:
  `ctrl+i` against `tab`, `ctrl+m` against `enter`, `ctrl+[` against
  `esc`, `shift+enter`, and `alt+[` or `alt+]`. Release and repeat strokes
  need `EventTypes`. `Scan` strokes need `AlternateKeys`.
* Until a `KeyboardEnhancementsMsg` arrives, the features are empty, so the
  legacy-safe defaults apply. The program rebuilds the keymap when the
  message arrives. `Rebuild` keeps the layers and changes only the
  features.

### 6. Chords, the leader key and timeouts

```go
type Matcher struct{ /* keymap, pending strokes, generation */ }
func NewMatcher(k *Keymap) *Matcher
func (m *Matcher) Update(msg tea.Msg, p Path, ctx when.Context) (Result, tea.Cmd)
func (m *Matcher) SetKeymap(k *Keymap) // cancels anything pending
func (m *Matcher) Pending() Sequence   // for a "ctrl+x …" hint in the footer

type Result struct {
    Command command.ID      // "" when nothing matched
    Args    json.RawMessage
    Handled bool            // the key was consumed: matched, or started a chord
    Replay  []tea.KeyPressMsg // strokes to give back to the focused pane
}
type TimeoutMsg struct{ gen uint64 }
```

* **A prefix stroke** returns `Handled` and one command,
  `Tick(Timeout, …)`, which yields `TimeoutMsg` with the current
  generation. A later stroke either completes a sequence, extends it, or
  ends it.
* **A sequence that ends without a match** returns its strokes in
  `Replay`, except the leader, which is always swallowed. A pane then
  still gets a printable prefix such as `g` in a `g g` binding.
* **A `TimeoutMsg` with an old generation is ignored.** Each new prefix
  bumps the generation, so a late timer cannot cancel a newer chord.
* **A change of focus path cancels a pending sequence,** as in codex.
* **No goroutines.** The matcher is a state machine driven by `Update`. The
  only timer is the `tea.Tick` command, which Bubble Tea runs. Tests pass
  their own `Options.Tick` and deliver `TimeoutMsg` themselves.
* **The leader** is unset unless the program sets `Options.Leader` (owner
  question Q1). `<leader>` in a rule expands to that sequence when the
  keymap is built. A rule that names `<leader>` with no leader set is an
  error.
* **`KeyReleaseMsg`** is matched only by `Release` strokes, so a program
  that turns on event types does not see each key twice.

### 7. Bubbles adapters and help

```go
// Describer gives a command's title for help.
type Describer func(id command.ID) (title string, ok bool)

// FromRegistry describes commands through Registry.Lookup and Command.Title.
func FromRegistry(r *command.Registry) Describer

func (k *Keymap) Binding(id command.ID, p Path, d Describer) key.Binding
func (k *Keymap) Help(p Path, d Describer, ids ...command.ID) help.KeyMap
```

* **`Binding`** returns a `key.Binding` whose keys are every
  `BubblesKeys()` spelling of each one-stroke sequence bound to `id`. For
  example, `shift+a` gives both `shift+a` and `A`. `key.Matches` then
  agrees with the matcher. Its help is the primary sequence and the
  command's title.
* **A chord** is put in the key list as its full text, such as
  `ctrl+x n`. `key.Matches` never matches it, because no single key
  prints a space-separated string, but `help` still shows it.
* **`Help`** gives the short help as the primary binding of each listed
  ID, and the full help grouped by context, innermost first.
  `workspace`'s `help.KeyMap`, from
  [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md),
  delegates to it once this record lands.

### 8. The VS Code loader, the exporter and the JSON Schema

```go
func LoadVSCode(r io.Reader, file string, ids func(command.ID) bool) ([]Rule, error)
func ExportVSCode(w io.Writer, rules iter.Seq[Rule]) error
func Schema(ids []command.ID, args func(command.ID) json.RawMessage) ([]byte, error)
```

* **The format** is VS Code's `keybindings.json`: an array of
  `{key, command, when, args}`. A `-` before the command removes a binding.
  There is one optional extra member, `context`, which defaults to
  `global`.
* **JSONC.** Comments and trailing commas are accepted, as both VS Code
  and gemini-cli do. A small scanner replaces them with spaces before
  decoding, so byte offsets, and therefore line and column, are
  unchanged. Decoding uses `encoding/json/v2` with unknown members
  rejected, and `jsontext.Decoder.StackPointer` for each error's JSON
  Pointer.
* **Errors are collected:** one bad rule does not drop the rest. The
  function returns the good rules and every error, joined.
* **Unknown commands.** When `ids` is given, a command it rejects is an
  error, so a typo is reported rather than silently ignored.
* **`ExportVSCode`** writes the effective rules back in the same format.
  `LoadVSCode(ExportVSCode(r))` returns `r`. A `keys` command can then
  print the user's whole keymap.
* **`Schema`** emits a JSON Schema (draft 2020-12) using the standard
  library:
  * `command` is an enum of the IDs and their `-` forms;
  * `key` has a regular expression for the grammar;
  * `context` is a string;
  * where `args` returns a schema for an ID, an `if`/`then` pair checks
    that command's `args`.

  An editor then completes and checks the user's file, as lazygit's
  published schema does. Where the file lives is the program's choice; an
  XDG config loader is a later record (0003-REPORT §7, item 9).

### 9. Moving `workspace` onto the engine

* **Command IDs.** The workspace's actions become commands:
  * `workspace.focus.next` and `workspace.focus.prev`;
  * `workspace.focus.pane`, with `args` `{"index": n}`;
  * `workspace.zoom`;
  * `workspace.resize.left`, `.right`, `.up` and `.down`;
  * `workspace.overlay.close`, in the `overlay` context.
* **Defaults.** `workspace.DefaultRules() []keymap.Default` returns the
  defaults that
  [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  sets, unchanged, in the `workspace` context. No default is `ctrl+c`.
* **New option.** `WithBindings(*keymap.Keymap)` is the new way in. It is
  not named `WithKeymap`, because revive's `confusing-naming` rule rejects
  names that differ from `WithKeyMap` only in case.
* **Focus path.**
  * `Workspace.KeyPath() keymap.Path` returns global, workspace, the
    focused pane as `pane:<id>`, then its own contexts, and then
    `overlay:<id>` for each open overlay.
  * A pane names its contexts through the optional interface
    `KeyContexter{ KeyContexts() []keymap.Context }`, such as
    `kind:editor` or `mode:vim-normal`.
  * While a modal overlay is open, the path is global and then that
    overlay only, so the layout's bindings do not reach through it.
* **Compatibility for `v0`.**
  * `KeyMap`, `DefaultKeyMap` and `WithKeyMap` stay, marked
    `// Deprecated:`. `WithKeyMap` converts the struct into rules for the
    `workspace` context, so there is one code path inside.
  * A program that never calls either option behaves as before. Its tests
    pass unchanged.
  * Removal is a later minor release with its own record.
* **`EscConsumer` keeps its meaning.** An overlay that handles Esc itself
  still gets it before `workspace.overlay.close`.
* **Unmatched keys** go to the focused pane, or the top modal overlay, as
  today, together with any `Replay` strokes.

### 10. Versioning

`keymap` ships in a minor release after
[0006-MADR-command-registry.md](0006-MADR-command-registry.md)'s, because
it needs `command.ID` and `when`. The owner names and tags the release.
`v0` allows the deprecations in §9.

### Consequences

* Good, because one `command.ID` is reachable by key, palette, slash
  command and shell, and help always shows the key that actually works.
* Good, because conflicts, reserved keys and unsupported strokes fail when
  the keymap is built, with file, line, column and JSON Pointer.
* Good, because users edit the format VS Code users already know, with
  schema completion in their editor.
* Good, because bubbles components and `help` keep working through
  `key.Binding` and `help.KeyMap`.
* Good, because chords and the leader key are a state machine with no
  goroutine, tested with no clock.
* Neutral, because the notation is Bubble Tea's, not VS Code's. Both
  parse, and the exporter writes VS Code's spellings where the file needs
  them.
* Neutral, because `workspace.KeyMap` stays deprecated for a while. That
  leaves two ways in until it is removed.
* Bad, because `keymap` cannot land before `command` and `when`
  ([0006-MADR-command-registry.md](0006-MADR-command-registry.md)).
* Bad, because the JSONC scanner, the schema emitter and the conflict
  checks are code this repository must own and test.
* Bad, because rebuilding on `KeyboardEnhancementsMsg` can change a default
  while the program runs. The change is from a legacy-safe key to a
  better one, and help follows it.

### Confirmation

* **Notation.**
  * Property tests over random strokes show that parse and canonical
    format round-trip, and that `ParseSequence(q.VSCode())` returns `q`.
  * A table of `tea.Key` values, built as ultraviolet decodes legacy,
    Kitty and Windows input, shows which strokes match. It covers
    upper-case letters, shifted symbols, lock modifiers, `BaseCode` and
    `Scan`.
  * For every one-stroke binding, `key.Matches(msg, k.Binding(id, …))`
    agrees with the matcher.
* **Conflicts.** Each conflict kind is produced by a planted keymap, and
  its message names the file, line, column and pointer.
* **Matcher.**
  * Chords, the leader, replay, and cancellation on a path change are
    driven with a recording `Tick`.
  * A stale `TimeoutMsg` is ignored.
  * One test runs the real `tea.Tick` command inside `testing/synctest`,
    so the one-second default is checked on a fake clock.
* **Loader.**
  * Comments, trailing commas, removals, unknown commands and several
    errors in one file are tested.
  * Exporting and re-loading returns the same rules.
  * The schema's `key` pattern compiles under Go's `regexp` and accepts
    every canonical form the property tests produce.
* **Workspace.** The 0002 workspace tests pass unchanged with the
  deprecated `KeyMap`, and again through `WithBindings`.
* **Mutation proofs,** each seen failing on a scratch copy:
  * resolution walks the path outermost first;
  * a stale timeout is not ignored;
  * `Remove` of an unbound key is accepted;
  * lock modifiers are not ignored;
  * the JSONC scanner shifts offsets;
  * `BubblesKeys` drops the `Text` spelling.
* Pre-add, `-race` and the Windows test host pass. CI is green on the
  push. `go.mod` gains no requirement.

## Pros and Cons of the Options

### A. A `keymap` package on command IDs

* Good, because every binding names a registry command, so palette, help
  and keys cannot drift apart.
* Good, because contexts make conflict detection static and precedence
  predictable.
* Good, because it reads and writes a standard file format.
* Bad, because it is the largest option, and it depends on 0006.

### B. `key.Binding` structs with a config loader

* Good, because it is the smallest change, and bubbles already works this
  way.
* Bad, because a struct field is not a command ID, so the palette and the
  shell cannot share it.
* Bad, because there are no chords, no contexts and no conflict checks.
  `key.Matches` takes the first field tested.

### C. VS Code's resolver with `when` only

* Good, because it is exactly VS Code's model, with nothing invented.
* Bad, because `when` expressions are runtime tests. Two rules cannot be
  proven disjoint at build time, so conflicts surface only when a key is
  pressed.
* Bad, because every pane must publish focus context keys for the
  expressions to test, which is the focus path written as strings.

### D. Each program writes its own keymap

* Good, because there is nothing to build here.
* Bad, because pi-go and every later program would rebuild chords, config
  loading and conflict checks.
* Bad, because the palette of
  [0008-MADR-command-palette.md](0008-MADR-command-palette.md) could not
  show the key for a command.

## Owner questions

* **Q1. The leader key.** Recommended: no library default. A program sets
  `Options.Leader`, and pi-go chooses. opencode's `ctrl+x` is the
  documented suggestion. The alternative is `ctrl+x` as the library
  default, which takes a key that editors use for cut.
* **Q2. The longest sequence.** Recommended: four strokes, which allows a
  leader and a three-key sequence, as Helix does. codex allows two.
* **Q3. A key that is also a chord prefix.** Recommended: a `Prefix`
  error. The alternative is a vim-style fallback, in which the single key
  fires when the timeout expires. It is more flexible, but it makes every
  use of that key wait for the timeout.
* **Q4. `workspace.KeyMap`.** Recommended: deprecate it in this release and
  remove it in a later minor, with its own record. The alternative is to
  remove it now, which `v0` allows, and which breaks any program that
  calls `WithKeyMap`.
* **Q5. The user file format.** Recommended: VS Code's `keybindings.json`
  with one optional `context` member. The alternatives are TOML, as codex
  and Helix use, or YAML, as lazygit uses. Each needs a parser module and
  its own record.

## More Information

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md):
  §1 findings 9 and 10, §3 (codex, gemini-cli, opencode), §6 (VS Code
  keybindings, JSON Schema) and §7, item 4.
* [0002-MADR-multi-pane-workspace-layouts.md](0002-MADR-multi-pane-workspace-layouts.md)
  §3: the workspace's key routing, which §9 moves onto this engine.
* [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md):
  the default keys that §9 carries over.
* [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md):
  the workspace's `help.KeyMap`.
* [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md):
  `termcap.Caps`, a source of `Features`.
* [0006-MADR-command-registry.md](0006-MADR-command-registry.md):
  `command.ID`, `Registry.Lookup` and `Dispatch`, and `when`.
* [0008-MADR-command-palette.md](0008-MADR-command-palette.md): the
  palette, which shows each command's primary binding.
* VS Code: <https://code.visualstudio.com/docs/configure/keybindings> and
  <https://code.visualstudio.com/api/references/when-clause-contexts>.
* Kitty keyboard protocol:
  <https://sw.kovidgoyal.net/kitty/keyboard-protocol/>.
