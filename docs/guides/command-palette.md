# The command palette

`palette` finds and runs anything from one overlay: commands, files,
panes, themes or a program's own items
([0008-MADR](../decisions/0008-MADR-command-palette.md)). Providers search
off the event loop and stream their hits back; the `Picker` draws them in
an overlay of a [workspace](building-workspaces.md), ranked by
[`fuzzy`](#ranking), and runs the one the user accepts on the event loop.
The same picker, without its input, is an editor pane's completion pop-up.

## Open the palette

A program makes one picker for its workspace, registers its providers,
and registers the command that opens it:

```go
km, err := keymap.New(keymap.Settings{}, append(workspace.DefaultRules(), palette.DefaultRules()...))
if err != nil { … }
ws := workspace.New(root, panes, workspace.WithBindings(km))

p := palette.NewPicker(ws)
if err := p.Register(palette.FromRegistry(reg), palette.Availability{}); err != nil { … }
if err := p.Register(palette.Panes(ws), palette.Availability{}); err != nil { … }
if err := reg.Register(p.OpenCommand("palette.open", "Command palette")); err != nil { … }
```

- **One keymap holds both sets of keys.** `palette.DefaultRules()` binds
  `ctrl+p` to `palette.open` while no modal overlay is open, and the
  picker's own keys in its scope, `kind:picker`: `up` and `ctrl+p`, `down`
  and `ctrl+n`, `pgup` and `pgdown`, `enter` to accept and `tab` to
  complete. The user's `keybindings.json` rebinds them all, as it rebinds
  the workspace's ([the keymaps guide](keymaps.md)). Without the
  palette's rules nothing moves the selection: the keys reach the input as
  text.
- **`ctrl+p` comes to your `Update`** as a `workspace.RequestMsg`, since
  no picker is open to take it, and your registry runs `OpenCommand`'s
  command, which opens the picker:

  ```go
  case workspace.RequestMsg:
      return m, commandtea.Dispatch(context.Background(), m.reg, msg.Request)
  ```

  A program that opens the picker some other way returns `p.Open()`.
- **The picker's moves run in the order their keys were pressed.** It is
  a `workspace.CommandConsumer`: the workspace hands it `CommandMsg` for
  its own commands at once, in `Update`, where a `RequestMsg` would go
  through a command and could arrive after a later key.
- **`esc`** clears a query that is not empty; on an empty one it closes
  the picker, as it closes any overlay.

The picker reads the theme, the width method and the keymap from the
workspace whenever it draws, so it follows a change of any of them without
being reopened. Its options: `WithLimit` (200 hits by default),
`WithGrouping(palette.ByProvider)`, `WithPlaceholder`, `WithLogger` for
providers' errors, `WithGOOS` for key hints, `WithRecents` and
`WithoutInput`. `Open` takes `WithID`, `WithAnchor` and `WithSize`.

`ExamplePicker` in `palette/example_test.go` runs this end to end: it
opens a picker over three themes, types a query, moves the selection and
accepts it.

## The built-in providers

| Provider | What it lists | Prefix |
| :--- | :--- | :--- |
| `FromRegistry(reg)` | the registry's commands available on `command.SurfacePalette` where the picker opened, by title, "Category: Title", slash name or alias; run through `commandtea.Dispatch` with `command.OriginPalette` | `>` |
| `Static(name, items)` | a small fixed set of `Item`s: themes, layout presets, models, modes | none |
| `Panes(ws)` | the focus ring's panes, which accepting focuses, and the hidden ones, which it shows | none |
| `Files(walk, opts...)` | the paths a `Walker` yields, ranked by their last element | `@` |

A command hit carries the command's `Danger`, drawn as a badge, and its
key in the hint slot, the binding that works where the user was. It runs
through the registry, so the registry's gate and its `PromptMsg` and
`ResultMsg` follow as from any other surface: from the palette, as from a
key, a person is at the keyboard, and the gate is asked only as
[0006-MADR](../decisions/0006-MADR-command-registry.md) A4 has it.

`Register` takes an `Availability`, whose `When` is a
[when expression](commands.md) read once when the picker opens: a provider
whose `When` does not hold there does not search until the picker opens
again.

## Prefixes

A query that starts with a provider's prefix searches that provider
alone, with the prefix taken off: `>open` searches the commands for
"open", and `@main` the files. A query of `?` lists the prefixes, and
accepting one types it. `tab` completes the selected hit's text under the
query's prefix.

## Write a provider

A provider has a name and a search:

```go
type bookmarks struct{ store *Store }

func (bookmarks) Name() string { return "bookmarks" }

func (b bookmarks) Search(ctx context.Context, s palette.Search, yield func(palette.Hit) bool) error {
    for _, bm := range b.store.All() {
        if err := ctx.Err(); err != nil {
            return err
        }
        if score, ok := s.Matcher.Score(bm.Title); ok {
            if !yield(palette.Hit{ID: "bookmark:" + bm.ID, Text: bm.Title, Score: score, Run: open(bm)}) {
                return nil
            }
        }
    }
    return nil
}
```

- **`Search` runs off the event loop.** Each keystroke cancels the last
  search's context and starts another; return when `ctx` is done or
  `yield` returns false. A panic or an error is the provider's failure,
  drawn as one muted row naming it, and its error goes to `WithLogger`'s
  handler, never the screen.
- **`s.Matcher` is compiled once per keystroke** and shared by every
  provider; score with it so that hits from different providers rank
  together. `s.WhenContext` is the when values where the picker opened.
- **A hit's `ID`** de-duplicates hits across providers, keeping the best
  score, and keys the recents. Its text fields are made one line of
  display text before they are drawn, so a title with an escape sequence
  or a newline draws as one row.
- **`Run`** is what accepting the hit does, on the event loop, with the
  arguments [the argument stage](#arguments) collected, or nil. A `Run`
  that blocks blocks the program; return a `tea.Cmd` for the slow part.
- **Optional capabilities,** found by type assertion: `Discoverer`, hits
  for an empty query; `Starter`, work before the first search of each
  opening, such as a walk; `Stopper`, a release when the picker closes;
  `Prefixed`; and `Debouncer`, a wait after a keystroke, so a burst of keys
  searches once.

The picker closes by any means, `enter`, `esc`, `workspace.Close` or a
program's `Pop`, through the workspace, which sends it
`workspace.OverlayClosedMsg`: its searches are cancelled, and its
providers stopped once they return.

## Arguments

A hit whose `Params` hold parameters asks for them in the same overlay
before it runs. `FromRegistry` fills `Params` from the command's schema
(`command.Params`).

- **In order,** one at a time: each `string`, `integer`, `number` and
  `boolean` parameter, and each with an `enum`. A boolean or an enum is
  chosen from a fuzzy list of its values, which starts on the default.
- **Checked before it runs:** an empty value is the default, or nothing
  when the parameter is optional, and a required one with no default is
  refused; so is an integer or a number that does not parse, or that is
  outside its `minimum` and `maximum`. The reason is in the help line, and
  the value stays to be fixed. The registry checks the rest of the schema,
  such as a pattern or a length, when the command runs.
- **A hidden parameter** (`hidden:""` on the field, `x-cli.hidden` in the
  schema) is skipped, and **a secret one** (`schema:"secret"`, `writeOnly`)
  is drawn with the `Mask` glyph. No value is ever kept in the recents.
- **The values reach `Run`** as one JSON object, in the order they were
  asked for.
- **A parameter it cannot ask for,** an array or an object, sends
  `palette.SelectedMsg` with the hit and runs nothing, so the program can
  open its own form.

`esc` clears a value, and on an empty one closes the picker without
running anything.

## Recents

A `Frecency` remembers how recently and how often hits were used, by ID
only:

```go
recent := palette.NewFrecency() // WithHalfLife, 7 days; WithMaxEntries, 500
p := palette.NewPicker(ws, palette.WithRecents(recent))
if err := p.Register(palette.Recents(recent, palette.FromRegistry(reg)), palette.Availability{}); err != nil { … }
```

`WithRecents` records each hit the picker accepts. `Recents` wraps a
provider so that its recent hits come first for an empty query, the most
used first, and win a near-tie for a typed one: a recent hit gains at
most one consecutive-match bonus, so it never passes a clearly better
match. Register the wrapped provider in place of the plain one.

The program keeps the store between runs; the palette opens no file:

```go
if f, err := os.Open(path); err == nil {
    _, err := recent.ReadFrom(f) // refuses another version: errors.Is(err, palette.ErrRecentsVersion)
    f.Close()
    …
}
…
var b bytes.Buffer
if _, err := recent.WriteTo(&b); err == nil { … } // write b to path
```

The stored form is versioned JSON,
`{"version":1,"entries":[{"id":…,"score":…,"at":…}]}`. `ReadFrom` reads
at most 1 MiB, refuses an unknown member, an ID over 1024 bytes or a
repeated one, and leaves the store as it was when it refuses.

## Files

`Files` searches the paths a `Walker` yields. `FSWalker` walks an `fs.FS`
the program gives, so the library opens nothing itself, and a
`PathFilter` leaves paths out:

```go
skip := palette.PathFilterFunc(func(p string, d fs.DirEntry) bool {
    return d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules")
})
files := palette.Files(palette.FSWalker(os.DirFS(root), skip), palette.WithMaxFiles(100_000))
```

A directory the filter skips is skipped whole, and one that cannot be
read is passed over. The walk runs once each time the picker opens, off
the event loop. Above
`WithIndexAbove` paths (5000 by default) it builds a `fuzzy.Index`, which
narrows the candidates before they are scored. It stops at `WithMaxFiles`
(200 000 by default), and a muted row says so. A path ranks by its last
element: equal to the query, or its stem equal; starting with it; a whole
element of the path equal to it; then any other match. A file hit has no
`Run`: the program takes the path from `SelectedMsg`. There is no
`.gitignore` reader; give a `PathFilter` that reads one.

## Completion pop-ups

A picker made `WithoutInput` is a completion pop-up for an editor pane:
non-modal, so the pane keeps the keyboard, under the pane's cursor, as the
overlay `completion`, with no input row of its own.

```go
complete := palette.NewPicker(ws, palette.WithoutInput())
if err := complete.Register(palette.Files(walker), palette.Availability{}); err != nil { … }
```

- **The pane opens it** by returning `complete.Open()` when its `@` or `/`
  token starts, and sends it `palette.SetQuery(token)` as the token
  changes. A pane returns a command that gives the message; the workspace
  delivers it to every pane and open overlay, and every open picker takes
  it. The program can send it to one overlay with
  `ws.SendOverlay("completion", palette.SetQuery(token))`.
- **Its keys** are `palette.DefaultRules()`' `overlay:completion`
  bindings: `up` and `ctrl+p`, `down` and `ctrl+n`, and `tab` and `enter`
  to accept. While it is open they shadow the editor's own `enter`; every
  other key reaches the editor. A pop-up opened `WithID` under another ID
  needs its own bindings in `overlay:<id>`.
- **Accepting a hit** sends `palette.SelectedMsg{ID: "completion", Hit: …}`
  and closes the pop-up. It runs nothing, and asks for no arguments: the
  pane inserts `Hit.Text`. `esc` closes it.

## Any pane can consume a command

`workspace.CommandConsumer` is not the palette's alone. A pane, or an
overlay's content, that runs some commands itself says so:

```go
func (e *editor) ConsumesCommand(id command.ID) bool { return id == "editor.indent" }
```

A key bound to `editor.indent` then reaches the editor as
`workspace.CommandMsg`, in `Update`, while it has the keyboard: the top
modal overlay alone when one is open, else the open overlays, top first,
then the focused pane. A command nobody consumes comes to the program as
`RequestMsg`, as before.

## Ranking

`fuzzy` ranks by grapheme cluster, in four grades, then by fzf's scores
within a grade: an exact match, a prefix, a whole word or path element,
then any other match that holds the query's letters in order
([0008-MADR](../decisions/0008-MADR-command-palette.md) §2).
`ExampleMatcher_Rank` in `fuzzy/example_test.go` ranks paths that way.
`fuzzy` links no Charm module but `x/ansi`, so a line-mode CLI can rank
with it without Bubble Tea.
