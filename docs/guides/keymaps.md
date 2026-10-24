# Keymaps

`keymap` binds keys to command IDs
([0007-MADR](../decisions/0007-MADR-keymap-engine.md)). A `Keymap` is
built once, from defaults and layers of bindings, and does not change. A
`Dispatcher` matches keys against it along a focus path: one key, a chord
such as `ctrl+x ctrl+s`, or a leader sequence, with a timeout. The
workspace routes its keys through one, and a matched command runs through
your [command registry](commands.md).

## Bind a command

A binding is keys, a command ID, and optionally `args`, a `when`
expression and a scope:

```go
save, err := keymap.ParseSequence("ctrl+s", nil)
if err != nil { … }
program := []keymap.Binding{{
    Keys: save, Command: "app.save",
    Location: keymap.Location{Layer: keymap.ProgramLayer},
}}
km, err := keymap.New(keymap.Settings{Reserved: []string{"ctrl+c"}}, workspace.DefaultRules(), program)
if err != nil { … } // the keymap is usable; the error lists every conflict
ws := workspace.New(root, panes, workspace.WithBindings(km))
```

- **The defaults come first,** then each layer in turn: a later layer
  wins over an earlier one. `workspace.DefaultRules()` is the workspace's
  own keys; pass it as the defaults, or a program's own `[]keymap.Default`
  with it.
- **`New` always returns a keymap.** Its error joins every problem it
  found: a key bound twice in one layer (`Duplicate`), a binding that
  starts another in one scope (`Prefix`), a user's key the program
  reserves (`Reserved`), and a key the terminal cannot deliver yet
  (`NotSupported`). Each is a `keymap.BindingConflict`, naming where the
  binding came from. `Conflicts()` lists the warnings: a binding an inner
  scope or a later layer hides (`Shadow`).
- **The notation is Bubble Tea's own,** `ctrl+alt+shift+a`, one to four
  strokes separated by a space. VS Code's spelling reads too (`cmd+k`,
  `escape`, `pagedown`), and so do `return`, `del`, `option` and `win`. An
  upper-case letter means shift: `A` is `shift+a`. A stroke ending in
  `:release` or `:repeat` fires on the key's release or repeat.

The workspace runs its own commands when their keys are pressed. Any other
command a key matches comes back to your `Update` as a
`workspace.RequestMsg`, from a key and with the workspace's `when` values,
for your registry:

```go
case workspace.RequestMsg:
    return m, commandtea.Dispatch(context.Background(), m.r, msg.Request)
```

## Scopes and the focus path

A binding has a scope, `command.Global` by default. A key is resolved
along the workspace's `KeyPath()`, outermost first:

1. `global`;
2. `workspace`, the layout's keys;
3. `pane:<id>`, the focused pane, then the scopes it names as a
   `workspace.KeyContexter`, such as `kind:editor` or `mode:vim-normal`;
4. `overlay`, while any overlay is open, then `overlay:<id>` for each;
5. the modes `ws.PushMode(scope)` added, until its `pop` is called.

While a modal overlay is open, the path is `global`, `overlay`, that
overlay and its own scopes, so the layout's keys do not reach through it.

The innermost scope with a binding for the keys decides them, and outer
scopes are not tried: a pane's `g g` hides a global `g` while the pane has
focus. A binding's `when` must hold too, read from `ws.WhenContext()`:
`workspace.focusedPane`, `workspace.zoomed`, a pane's own values as a
`workspace.Contexter`, and the rest.

## Chords and the leader

- **A chord** waits for its next stroke. `Settings.Timeout`, one second
  by default, ends it; the keys typed so far, except the leader's, then go
  to the focused pane. A key that matches nothing ends it the same way, and
  is then taken on its own.
- **The leader** is `Settings.Leader`, such as `"ctrl+space"`. A binding
  writes `<leader> f`, and `New` puts the leader's keys in its place.
- **While a chord waits,** `esc` clears it, and `backspace` takes back
  its last stroke, unless a binding uses either key.
  `Settings.EscReplays` and `BackspaceReplays` make them end the chord
  and go to the pane instead.
- **A which-key panel** reads a dispatcher's `ActiveKeys`: every key
  that can come next, with its label, and either the command it completes
  or that it continues the chord.

## The user's file

Users rebind keys in VS Code's `keybindings.json` format, comments and
trailing commas allowed:

```jsonc
[
    // Save with ctrl+s, everywhere.
    { "key": "ctrl+s", "command": "app.save" },
    // Zoom with ctrl+g instead of alt+z.
    { "key": "alt+z", "command": "-workspace.zoom", "context": "workspace" },
    { "key": "ctrl+g", "command": "workspace.zoom", "context": "workspace" },
    // A chord in the editor only.
    { "key": "ctrl+k ctrl+c", "command": "editor.comment", "context": "kind:editor" },
]
```

```go
user, err := keymap.LoadVSCode(f, path, func(id command.ID) bool {
    _, ok := r.Lookup(id)
    return ok
})
km, buildErr := keymap.New(settings, workspace.DefaultRules(), program, user)
```

- **`"context"`** is the scope, `"global"` when absent.
- **A `-` before the command** removes a binding an earlier layer made:
  with `"key"`, those keys; without, every key of that command in that
  scope.
- **Errors are collected.** A bad binding is left out, and the others
  load. Each error is a `*keymap.LoadError` with the file, line, column
  and JSON Pointer, such as `keybindings.json:4:12 (/1/key)`.
- **`ExportVSCode`** writes bindings back in the same format, so a `keys`
  command can print the user's whole keymap.
- **`Schema(ids, args)`** is a JSON Schema for the file. Write it beside
  the file, so an editor completes the command IDs and checks the keys.

## Reserved keys

`Settings.Reserved` names keys only the program binds, such as
`ctrl+c`. A user's binding that starts with one is a `Reserved` conflict,
and is dropped. The workspace's defaults never bind `ctrl+c`: it reaches
your program as a key.

## What changes on an enhanced terminal

A legacy terminal cannot tell `shift+enter` from `enter`, or `ctrl+i`
from `tab`. A terminal with the Kitty keyboard protocol can, and Bubble
Tea reports what it accepted as a `tea.KeyboardEnhancementsMsg`. Until
then the keymap assumes nothing:

- **A user's binding that needs a feature** is `NotSupported` at first,
  kept aside, and bound when you rebuild with the feature.
- **A default can name alternatives,** tried in order when its own keys
  cannot be delivered: a newline on `shift+enter`, else `alt+enter`, else
  `ctrl+j`.
- **`keymap.FactsFrom(caps)`** reads `termcap`'s capabilities: whether
  releases are reported, whether shift+enter and ctrl+enter arrive apart
  from enter, whether alt+enter arrives (not on Terminal.app or iTerm2,
  whose Option key types characters by default), and whether the host,
  such as VS Code, takes `ctrl+q` and `ctrl+i`.

Rebuild when the message arrives, and give the workspace the new keymap:

```go
case tea.KeyboardEnhancementsMsg:
    km, err := m.km.Rebuild(keymap.FeaturesFrom(msg.Flags), m.facts)
    if err != nil { m.toast(err) }
    m.km = km
    m.ws.SetBindings(km)
```

On a legacy terminal, keys are read as the bindings name them before they
match (`keymap.Normalize`): `ctrl+5` is `ctrl+]`, and on Windows an AltGr
key is the text it types, for the pane, never a binding.
`Settings.GOOS` is the system the keys come from, the program's own by
default.

## Help and hints

- **The workspace's help** reads the keymap: a rebound or removed key
  shows as it is.
- **`km.KeyBinding(id, path, describer)`** is a bubbles `key.Binding` for
  your own component, and `km.Help(path, describer, ids...)` a
  `help.KeyMap`. `keymap.FromRegistry(r)` gives the commands' titles.
- **`km.Shortcut(id, path, goos, glyphs)`** is the label of a command's
  key, such as `⌃s` on macOS or `ctrl+s` elsewhere, or `""` when nothing
  binds it, so a footer drops a hint rather than showing a stale one.
- **`keymap.NewDispatcher(km).Explain(key, path, ctx)`** tells what a
  key does: the key as it arrived and as it is read, the binding that
  wins, and the ones it hides. It is the data for a "keys debug"
  command.

## Moving from `workspace.KeyMap`

`workspace.KeyMap`, `DefaultKeyMap` and `WithKeyMap` were deprecated in
`v0.11.0`, and `v0.12.0` removed them. In `v0.11.x`, `WithKeyMap` bound
each field to its command:

| Field | Command | Args | Scope |
| :--- | :--- | :--- | :--- |
| `FocusNext`, `FocusPrev` | `workspace.focus.next`, `workspace.focus.prev` | — | `workspace` |
| `FocusPane[i]` | `workspace.focus.index` | `{"index": i+1}` | `workspace` |
| `Zoom` | `workspace.zoom` | — | `workspace` |
| `ResizeLeft`, `ResizeRight`, `ResizeUp`, `ResizeDown` | `workspace.resize.direction` | `{"direction": "left"}`, and so on | `workspace` |
| `Close` | `workspace.overlay.close` | — | `overlay` |

A rebinding becomes a rule. Where you wrote:

```go
k := workspace.DefaultKeyMap()
k.Zoom.SetKeys("ctrl+g")
ws := workspace.New(root, panes, workspace.WithKeyMap(k))
```

write:

```go
g, err := keymap.ParseSequence("ctrl+g", nil)
if err != nil { … }
rebind := []keymap.Binding{
    {Command: "workspace.zoom", Scope: "workspace", Remove: true,
        Location: keymap.Location{Layer: keymap.ProgramLayer}},
    {Keys: g, Command: "workspace.zoom", Scope: "workspace",
        Location: keymap.Location{Layer: keymap.ProgramLayer}},
}
km, err := keymap.New(keymap.Settings{}, workspace.DefaultRules(), rebind)
if err != nil { … }
ws := workspace.New(root, panes, workspace.WithBindings(km))
```

The removal takes away every default key of `workspace.zoom` in the
workspace scope, and the second binding adds yours. A disabled field
becomes a removal alone.
