# Migrating between releases

A renamed name keeps working for one minor release, as a deprecated
wrapper, alias or field, and is removed in the next
([AGENTS.md, API conventions](../../AGENTS.md#api-conventions), rule 1).
`staticcheck` reports each use of one as SA1019, so a program can find
them all before the removal.

## Migrating to `v1.13.0`

`v1.13.0` follows `v0.12.0` directly. It was tagged in place of
`v0.13.0` and kept, and there is no `v0.13.0`
([0020-MADR](../decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
A4). `go get github.com/maccavelli/go-tui-lib@latest` gives it.

`v1.13.0` adds packages and fields, and removes nothing
([0020-PLAN-charm-free-cores-glyphs-and-safetext](../decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md)
Step 6). A program needs no change to build. What it shows changes where
it puts untrusted text on the screen that is not well formed, as
"Changed" lists.

### Changed

- **Invalid UTF-8 shows as U+FFFD,** one for each maximal subpart (the
  Unicode Standard, §3.9.6), where the library's output dropped it: in a
  pane's title and badge, a notification, a window title, a link's text,
  a beacon's vendor and tmux's reply. A raw byte 0x80 to 0x9F that
  continues no character is invalid UTF-8, so it shows as U+FFFD and
  opens no sequence.
- **Malformed escape sequences are read as the DEC parser reads them**
  ([0020-MADR](../decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md)
  A3):
  - a C1 control encoded as UTF-8, U+0080 to U+009F, is its 7-bit form:
    U+009B opens a CSI, U+0090 a DCS, U+009C ends a string, and the
    sequence goes whole, where only the code point was dropped before;
  - BEL ends a DCS, SOS, PM or APC, so the text after it shows;
  - text other than ASCII inside a DCS, SOS, PM or APC is part of it;
  - a character inside an OSC does not end it, even one whose UTF-8 holds
    the byte 0x9C, such as `Ü`;
  - a pane's view splits a malformed sequence as a title always did: an
    nF escape such as `ESC SP X` is removed, and the text after it stays;
  - a pane's view keeps no SGR or OSC 8 link written with a C1 control.

  Well-formed text, styles and links come out as before.

### Added

- **`safetext`:** `Clean` and `IsClean` by `Rules`, with the presets
  `Text`, `Strict`, `Line`, `Command` and `StatusLine`; `Sanitizer`, which
  cleans a stream; `SafeURL` and `URLs`. It imports only the standard
  library and `glyph`, so a line-mode CLI can use it without Bubble Tea.
  `internal/sanitize`, which cleans what the library shows, now runs on
  it.
- **`fuzzy`:** a `Matcher` that matches and ranks by grapheme cluster,
  with grades and fzf's scoring, and an `Index` for large candidate lists.
- **`glyph.Set`:** `Done`, `Failed`, `Running` and `Pending`, each its own
  shape in both sets; `Heading`; `MeterFill` and `MeterTrack`; `Control`;
  `Prompt` and `Mask`.
- **`workspace`:** `Bindings` and `Theme`, so a pane can show a command's
  key and draw as the workspace does; `CommandConsumer` and `CommandMsg`,
  for a pane or an overlay that runs some commands itself; and `Close`,
  an overlay's request to close itself.

`fuzzy`, `glyph`, `layout`, `safetext` and `when` link no Charm module but
`x/ansi`. A test holds them to that, so a program that uses only these
builds without Bubble Tea, Lip Gloss or Bubbles.

## Migrating to `v0.12.0`

`v0.12.0` removes the names `v0.11.0` deprecated
([0007-PLAN-keymap-engine](../decisions/0007-PLAN-keymap-engine.md) Step
12). A program that builds on `v0.11.x` with no SA1019 report needs no
change.

### Removed

| Removed | Use |
| :--- | :--- |
| `workspace.KeyMap` and its `Bindings` | a keymap: `keymap.New` with `workspace.DefaultRules()` as its defaults, and its `Bindings`, or `keymaptea.KeyBinding` from `v2.0.0` |
| `workspace.DefaultKeyMap` | `workspace.DefaultRules()`, the same keys |
| `workspace.WithKeyMap` | `workspace.WithBindings`, and `SetBindings` for a rebuilt keymap |

[The keymaps guide](keymaps.md#moving-from-workspacekeymap) maps each
field to its command, and writes one rebinding both ways.

## Migrating to `v0.11.0`

`v0.11.0` adds `keymap`, and moves the workspace's keys onto it
([0007-MADR](../decisions/0007-MADR-keymap-engine.md)). A program that
sets no key option needs no change: the default keys are the same.

### Deprecated

These still work in `v0.11.x`, and are removed in `v0.12.0`.

| Deprecated | Use |
| :--- | :--- |
| `workspace.KeyMap` | a keymap: `keymap.New` with `workspace.DefaultRules()` as its defaults |
| `workspace.DefaultKeyMap` | `workspace.DefaultRules()`, the same keys |
| `workspace.WithKeyMap` | `workspace.WithBindings`, and `SetBindings` for a rebuilt keymap |

Each field is now a command: `FocusNext` and `FocusPrev` are
`workspace.focus.next` and `.prev`, `FocusPane[i]` is
`workspace.focus.index` with `{"index": i+1}`, `Zoom` is `workspace.zoom`,
the four `Resize…` fields are `workspace.resize.direction` with
`{"direction": "left"}` and the rest, and `Close` is
`workspace.overlay.close` in the `overlay` scope.
[The keymaps guide](keymaps.md#moving-from-workspacekeymap) writes one
rebinding both ways.

### Changed

- **The workspace's help lists a command once.** `alt+1` to `alt+9` are
  one entry, `alt+1 pane`, where the full help listed nine, and the four
  resize keys are one, `alt+shift+left resize`. Under a modal overlay,
  the focus and zoom keys, which never reached through it, are no longer
  listed.
- **A global binding reaches a modal overlay.** A modal overlay still
  takes every key no binding on its path takes, and the layout's keys
  still do not reach through it.
- **`glyph.Set` has `Keys`,** a `KeyGlyphs` of the macOS modifier and
  arrow labels. Their ASCII twins are words, such as `opt+`: the one
  exception to a glyph's single cell.

### Added

- **`keymap`:** notation, layered keymaps with conflicts, chords and a
  leader, VS Code-format user files with a JSON Schema, terminal-aware
  defaults, labels, help, which-key data and a key explanation.
- **`workspace`:** `DefaultRules`, `WithBindings`, `SetBindings`,
  `KeyPath`, `KeyContexter`, `PushMode` and `RequestMsg`; and two
  commands, `workspace.focus.index` and `workspace.resize.direction`.

## Migrating to `v0.10.0`

`v0.10.0` removes the names `v0.9.0` deprecated
([0014-PLAN-canonicalization](../decisions/0014-PLAN-canonicalization.md)
Step 9). A program that builds on `v0.9.x` with no SA1019 report needs no
change, except for the audit log's key.

### Removed

| Removed | Use |
| :--- | :--- |
| `command.Decision` | `command.Verdict` |
| `command.Record.Decision` | `Record.Verdict` |
| `command.Request.Context`, `Invocation.Context` | `WhenContext` |
| `termcap.New` | `termcap.NewProber` |
| `layout.SidebarWidth`, `BottomHeight`, `MainSize`, `BottomSpan`, `Footer`, `Gap`, `Breakpoints`, `NoResponsive` | `layout.WithSidebarWidth`, `WithBottomHeight`, `WithMainSize`, `WithBottomSpan`, `WithFooter`, `WithGap`, `WithBreakpoints`, `WithoutResponsive` |
| `workspace.WithMouse` | `workspace.WithoutMouse()`, or nothing: the mouse is handled by default |
| `termsvc.WithGate` | `termsvc.WithNotifyFilter` |

### Changed

- **`SlogAuditor` logs the verdict under the key `verdict`,** where it was
  `decision`. A query or alert on the old key needs the new one.

## Migrating to `v0.9.0`

`v0.9.0` gives the API one name for each idea
([0014-PLAN-canonicalization](../decisions/0014-PLAN-canonicalization.md)).

### Renamed

The old names still work in `v0.9.x`, and are removed in `v0.10.0`.

| Old | New |
| :--- | :--- |
| `command.Decision` | `command.Verdict`, with the same four constants |
| `command.Record.Decision` | `Record.Verdict`; the registry sets both |
| `command.Request.Context`, `Invocation.Context` | `WhenContext`; the registry reads `WhenContext`, else `Context` |
| `termcap.New` | `termcap.NewProber` |
| `layout.SidebarWidth`, `BottomHeight`, `MainSize`, `BottomSpan`, `Footer`, `Gap`, `Breakpoints` | `layout.WithSidebarWidth`, `WithBottomHeight`, `WithMainSize`, `WithBottomSpan`, `WithFooter`, `WithGap`, `WithBreakpoints` |
| `layout.NoResponsive` | `layout.WithoutResponsive` |
| `workspace.WithMouse(false)` | `workspace.WithoutMouse()` |
| `workspace.WithMouse(true)` | nothing: the mouse is handled by default |
| `termsvc.WithGate` | `termsvc.WithNotifyFilter` |

`SlogAuditor` still logs the verdict under the key `decision` in
`v0.9.x`; the key is `verdict` from `v0.10.0`.

### Removed

These were deprecated in `v0.8.0`.

| Removed | Use |
| :--- | :--- |
| `termcaptest.RunTimeout` | `(*termcaptest.Terminal).SetRunTimeout`, or `DefaultRunTimeout` |
| `workspace.KeyFocusedPane`, `KeyZoomed`, `KeyHiddenPanes`, `KeyOverlay`, `KeyModal`, `KeyWidth`, `KeyHeight` | `workspace.FocusedPaneKey()`, `ZoomedKey()`, `HiddenPanesKey()`, `OverlayKey()`, `ModalKey()`, `WidthKey()`, `HeightKey()` |

### Changed

- **Option types are opaque.** `command.Option` and `RegistryOption`,
  `termcap.Option`, `termsvc.NotifyOption`, and `workspace.Option` and
  `WrapOption` are interfaces that only their package's functions make.
  A program's own `func(*T)` no longer compiles as one; build options with
  the `With…`, `Without…` and `On…` functions.
- **Every exported enum has text forms.** 21 enums gain `MarshalText` and
  `UnmarshalText`, so their JSON is their token, such as `"read-only"`,
  where it was a number. A value with no token prints `Type(N)`, such as
  `Mux(9)`, where some printed the number alone.
- **`termcap`'s JSON uses `encoding/json/v2`.** It no longer escapes `<`,
  `>` and `&`. Reading is stricter: a duplicate name or invalid UTF-8 is
  an error, and a name matches in its own case only. Invalid UTF-8 in a
  program's own `Caps` values is written as U+FFFD.
- **`termcap.ParseTmux` cleans its fields** of escape sequences, controls
  and invalid UTF-8, as every other source of a terminal's text is.
- **`when.Parse`'s errors are `*when.SyntaxError`,** with the offset, the
  message and any error beneath, such as a regex's. Their texts are
  unchanged.
- **`termcap`'s errors for an unknown name wrap `termcap.ErrUnknownName`,**
  with their texts unchanged.
- **`command`'s raw JSON is spelled `jsontext.Value`,** which is the type
  `json.RawMessage` names, so no code changes.

### Added

- `command.AuditorFunc`, `termsvc.BackendFunc` and `termsvc.ClipboardFunc`,
  to use a function as a hook.
- `when.SyntaxError` and `termcap.ErrUnknownName`.
