---
date: 2026-10-02
subject: an audit of v0.1.0, the Go 1.25–1.27 changes that matter here, how open-source agent TUIs, TUI frameworks and terminal standards solve the problems go-tui-lib is about to take on, and a second, source-level pass over the Kilo, Grok Build, opencode and codex TUIs for the API surface beyond the planned records
examines: "go-tui-lib at v0.1.0 (5c57806); bubbletea v2.0.10, lipgloss v2.0.6, bubbles v2.2.1, x/ansi v0.11.8 and ultraviolet at the pinned pseudo-version; Go 1.27.1's api files and release notes; crush, codex, gemini-cli, goose, toad and opencode source; Aider, Textual, ratatui, Ink, Notcurses, lazygit, k9s, gh-dash, Posting, zellij, Helix and yazi documentation; ACP, Kitty, Contour and freedesktop specifications; second pass: Kilo-Org/kilocode at e0c27aa71f, SpaceXAI grok-build at 2bdd1d6a (monorepo 559751fd), sst/opencode at 82ea3a3a63 (v1.18.34) and openai/codex at 33aea33b39"
---
# Agent TUI ecosystem research, an audit of `v0.1.0`, and a source pass over four agent TUIs

This report records findings. It decides nothing. §1 to §7 support these
decisions:

- [0002-MADR-multi-pane-workspace-layouts.md](../decisions/0002-MADR-multi-pane-workspace-layouts.md),
  amendment A1, and its plan
  [0002-PLAN-harden-workspace-v0-1-1.md](../decisions/0002-PLAN-harden-workspace-v0-1-1.md);
- [0004-MADR-integrate-charm-v2-and-go-1-27.md](../decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md);
- [0005-MADR-terminal-capabilities-and-services.md](../decisions/0005-MADR-terminal-capabilities-and-services.md);
- [0006-MADR-command-registry.md](../decisions/0006-MADR-command-registry.md);
- [0007-MADR-keymap-engine.md](../decisions/0007-MADR-keymap-engine.md);
- [0008-MADR-command-palette.md](../decisions/0008-MADR-command-palette.md);
- [0009-MADR-streaming-content-engine.md](../decisions/0009-MADR-streaming-content-engine.md).

§8 to §11, added on 2026-10-02, support no record yet. They feed amendments
to the proposed records 0004 to 0009 and the records that items 6, 7, 9 and
10 of §7 still need.

## Why, and how

On 2026-10-01 the owner asked:

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

Three read-only passes ran on 2026-10-01:

- **An audit of `v0.1.0`.** It read every package and ran probe tests and
  benchmarks on a scratch copy. It checked Go's standard-library changes
  against `$GOROOT/api/go1.2{4,5,6,7}.txt` in the Go 1.27.1 install, and
  checked the language changes by compiling probes.
- **Agent TUIs.** Shallow clones, read at the source:
  - charmbracelet/crush at `76cc5c5` (2026-09-30);
  - openai/codex at `d25c114`;
  - google-gemini/gemini-cli at `c9096a8`;
  - block/goose at `8f4cab5`;
  - batrachianai/toad at `dd4f90e`;
  - sst/opencode at `1ddb087`;
  - opencode-ai/opencode, the archived Go version, at `73ee493`.

  Aider, Textual and ACP were read on the web.
- **Frameworks, applications and terminal standards,** read on the web,
  with the Charm v2 facts checked against the module source.

**How sure each finding is.** "Verified" means it was checked against
source, a specification or a probe. "Not re-checked" means it comes from
the researcher's prior knowledge, or from a page summary that was not
compared with the raw text. A web page read through a fetch tool is a
model's summary of that page. Every record that relies on a web finding
checks it again before relying on it.

### The second pass, 2026-10-02

On 2026-10-02 the owner asked:

> assess the kilo, grok, opencode, and codex sources in gitrepos. focus on
> their TUI implementations. i want to build this sdk with first-class
> support for as many features as possible while still remaining
> cross-platform and multi-terminal optimized. dig into those provider
> sources and find features, functionality, UI/UX/TUI design elements that
> would benefit the project to enhance and extend it's extensibility,
> flexibility, and compatibility. i want this sdk to provide a deep and wide
> api surface. be mindful of best practices and standards. update/amend
> findings to the 0003 report in docs/reports

- **Four read-only passes ran in parallel,** one per project, on local
  clones of the owner's forks (§8.1). Each pass was told what §1 to §7 and
  the records 0004 to 0009 already hold, and reported only what was new,
  what extended a record, or what contradicted one.
- **Everything in §8 was read in source.** Each pass cited the line ranges
  it opened. Sixteen of those citations, spread over the four projects, were
  re-opened afterwards, and every one matched.
- **OpenTUI's own source was not read,** because `node_modules` is not
  installed in the opencode clone. A renderer behaviour inferred from how
  opencode calls OpenTUI says "inferred from usage".
- **Nothing was run.** No binary was built, no test was run, and no terminal
  was exercised. A per-terminal claim in §9 is what the source asserts and
  works around, not a behaviour observed here.

## 1. Audit of `v0.1.0`

Every finding below is about code in the tagged `v0.1.0`. Findings 1 to 4
were confirmed with failing probe tests on a scratch copy. On 2026-10-02 the
code for findings 1, 4 and 6 was read again at `5c57806`.

| # | Finding | Where | Severity |
| :--- | :--- | :--- | :--- |
| 1 | A modal overlay anchored `BelowCursor` overflows the stack | `workspace/overlay.go:99-101`, `workspace/render.go:53-54` | crash |
| 2 | The view cache goes stale after `SetPane`, and a pane and an overlay with the same ID share a cache entry | `workspace/workspace.go:245-249`, `workspace/render.go:39,182` | wrong output |
| 3 | Overlays are never told a new size on resize | `workspace/workspace.go:500-523` | wrong output |
| 4 | tuitest's `-update` flag panics any consumer that defines its own | `tuitest/tuitest.go:25-33` | consumer panic |
| 5 | A frame allocates a new canvas and measures every pane three times | `workspace/render.go:21-44` | performance |
| 6 | Cell widths can disagree with Bubble Tea's renderer | `workspace/render.go`; lipgloss `canvas.go:27` | misaligned borders |
| 7 | `Focus` and `Blur` are lost on value-type panes; `Push` does not blur and `Pop` does not refocus | `workspace/workspace.go:50-54,366-379`, `workspace/overlay.go:47-68` | lost focus |
| 8 | Stored resize deltas grow without bound; unnamed splits write keys that are never read | `layout/state.go:70-85`, `layout/size.go:158-202` | dead zone |
| 9 | The `alt+[` and `alt+]` defaults collide with escape-sequence prefixes | `workspace/keys.go:33-34` | lost keys |
| 10 | Charm v2 features are under-used | `theme`, `workspace` | integration |
| 11 | The conformance scan matches names, not uses | `internal/conformance/conformance_test.go:58-82` | gate gap |
| 12 | Map iteration in tests; concurrency safety undocumented | tests; `workspace` | determinism |

### 1.1 Crash: a modal `BelowCursor` overlay

`overlayRect` places a `BelowCursor` overlay by calling `w.Cursor()`.
`Cursor` returns the top modal overlay's cursor, which it finds by calling
`overlayRect` for that overlay. The two call each other until the stack
overflows. The probe failed with `fatal error: stack overflow` in `Push`,
which measures the overlay to send its first size. A completion pop-up is
non-modal and does not hit this. A modal picker anchored to the cursor does.

### 1.2 The view cache

- **Stale after `SetPane`.** `SetPane` replaces the pane but keeps its cache
  entry. A replacement that implements `Changer` and reports no change
  shows the old pane's view.
- **Shared keys.** A bordered pane and an overlay both use the key
  `"box:"+id`. A pane and an overlay with the same ID draw each other's
  view.
- **No eviction.** `Pop` and a removed pane leave their entries.
- **No theme in the key.** A future theme change must clear the cache.
- The key is built with `fmt.Sprintf` on every call (`render.go:79`).

### 1.3 Overlay sizes

`resolve` walks `plan.Order`, which holds panes only. After the probe shrank
the window to 60×20, the overlay still believed it was 98×28.

### 1.4 tuitest's `-update` flag

`tuitest` registers `-update` in `init`, when no flag of that name exists.
Package `init` functions of imports run before the importing package's
variable initialisers. A consumer's `var update = flag.Bool("update", …)`
then finds the flag already defined and panics with
`flag redefined: update`. The comment above the code says it handles this
case. It does not.

### 1.5 Render performance

`BenchmarkRender` at 200×60 with four panes, on the macOS development host:

| Step | Time/op | Bytes/op | Allocs/op |
| :--- | :--- | :--- | :--- |
| `v0.1.0` | 1.55 ms | 1.74 MB | 1091 |
| Reuse the canvas | 1.27 ms | 225 KB | 958 |
| + edge glyphs rendered once per box, struct cache key | 1.13 ms | 178 KB | 611 |
| + direct ultraviolet draw, hit test on the plan's rectangles | 0.79 ms | 173 KB | 593 |
| TrueColor, `v0.1.0` → all three | 1.92 → 0.97 ms | 1.86 MB → 253 KB | 3556 → 2158 |

- **Where the time goes.** About 61% of samples are runtime overhead from
  GC starts and span allocation. 85% of bytes come from the new
  `uv.NewBuffer` that `lipgloss.NewCanvas` builds every frame.
- **Triple measurement.** `NewLayer`, `NewCompositor` and the compositor's
  flattening each measure every pane with `lipgloss.Width`, so each frame
  is scanned for graphemes three times.
- **What remains** after the three changes is ultraviolet's own cell
  drawing: `StyledString.Draw` 42%, `Canvas.Render` 30%, `Cell.Equal` 20%.
- **Next steps not measured:**
  - return the previous frame when nothing changed;
  - keep a sorted ID slice for `Broadcast`, which allocates and sorts on
    every message (`workspace.go:311-315`);
  - skip `clip` for `None` chrome.
- **The change needs a direct ultraviolet import.** ultraviolet is already
  an indirect requirement, at a pseudo-version with no tag.
  0002-MADR §1 says "No direct `ultraviolet` import", so this needs a
  record.

All golden tests passed byte for byte after the three changes.

### 1.6 Width method

- **The library measures with grapheme clustering.** `ansi.StringWidth`,
  `ansi.Truncate` and the lipgloss canvas use it, and lipgloss v2.0.6
  `canvas.go:27` sets `ansi.GraphemeWidth` on an unexported buffer.
- **Bubble Tea's renderer starts with `WcWidth`.** It switches to
  `GraphemeWidth` when a `ModeReportMsg` for mode 2027 (`ModeUnicodeCore`)
  reports reset, set or permanently set (`tea.go:802-805`).
- **Bubble Tea asks only sometimes.** It sends the 2026 and 2027 queries
  only when its environment heuristics allow (`tea.go:1118-1123`).
  `ModeReportMsg` also reaches the model's `Update`.
- **Effect.** Text with a ZWJ sequence or VS16 can be one width to the
  workspace and another to the renderer, and a border then shifts.
- **Fixing it needs the drawing in finding 5,** because the lipgloss canvas
  cannot change its method. Checked on 2026-10-02.

### 1.7 Focus and value semantics

- **The calls are lost.** `Focus()` and `Blur()` are called on whatever
  value the workspace holds. A value-type pane changes a copy, and the
  change is lost.
- **Bubbles is never focused.** bubbles v2.2.1 `textinput.(*Model).Focus`
  (`textinput.go:268`) and `textarea.(*Model).Focus` (`textarea.go:799`)
  have pointer receivers. A pane that wraps a model by value never
  satisfies `Focuser`, and is never focused.
- **Overlays.** `Push` does not blur the pane beneath, and `Pop` does not
  refocus it.
- **Suggested fix:** deliver focus and blur as messages through `Update`,
  as `SizeMsg` is, and add a generic adapter for `(M, tea.Cmd)` models.

### 1.8 Resize state

- **Unbounded deltas.** `State.WithResize` adds without limit. `Split.resize`
  clamps each solve but never writes the clamped value back. The probe
  applied +1000 and then −5: the state stored 995, and the separator did
  not move. That is a dead zone when dragging or holding a key, and it is
  persisted.
- **Keys never read.** Keyboard resize on an unnamed split writes a
  positional key (`"/0:0"`) through `sepID`. `resizeKey` reads only named
  splits.
- **Suggested fix:** `Solve` reports the deltas it applied, and the
  workspace stores those.

### 1.9 Default keys

- **Legacy encoding.** In legacy key encoding, `alt+[` is `ESC [` (the CSI
  introducer) and `alt+]` is `ESC ]` (the OSC introducer).
- **Decoding.** ultraviolet's decoder treats them as keys only when they
  arrive alone in one read, so `alt+[` followed quickly by `A` decodes as
  Up.
- **Kitty only helps where supported.** Bubble Tea asks for the Kitty
  keyboard protocol's disambiguation flag, but that helps only on terminals
  that support it.

### 1.10 Charm v2 features not used

- No `SetTheme`, and nothing handles `tea.BackgroundColorMsg` or
  `tea.ColorProfileMsg`. `lipgloss.LightDark` and `lipgloss.Complete`
  would build palettes from them.
- `KeyMapper` is never read, and `KeyMap` does not implement bubbles'
  `help.KeyMap`.
- `glyph.SeparatorCross` is never drawn. `renderSeparator` uses the
  workspace's chrome and ignores per-pane chrome (`render.go:193`).
- A helper could fill a `tea.View`'s `Content`, `Cursor` and `OnMouse`.

### 1.11 Conformance scan

It matches the identifiers `os.Stdout`, `os.Stderr`, `signal.Notify` and
`AltScreen` by name. It misses:

- `fmt.Print*`, `log.*`, and the `print` and `println` builtins;
- an aliased `os` import.

Resolving uses with `go/types`, which is in the standard library, closes
the gap.

### 1.12 Determinism and concurrency

- **Map iteration in subtests.** It is at `golden_test.go:49`,
  `layout_test.go:247,425` and `glyph_test.go:27`.
- **Undocumented concurrency.** `Render` mutates the workspace. That is
  correct on Bubble Tea's event loop, but the type does not say it is not
  safe for concurrent use.
- **A misleading test.** `TestCommandsRunConcurrently` delivers to
  `Update` serially.

### 1.13 Modernisation

- **`go fix -diff ./...` exits 1.** It suggests:
  - `slices.Backward` (`size.go:230`);
  - `maps.Copy` (`state.go:101`, `workspace.go:193`);
  - `strings.SplitSeq` (`render.go:183`);
  - `wg.Go` (`workspace_test.go:458,486`).
- `strings.Lines` would simplify `tuitest.Annotate`.
- `go vet`, `staticcheck -checks all` and `-race` were clean.
- The code already uses `b.Loop`, range over int, `min`/`max`, `slices`
  and `os.Root`.

## 2. Go 1.25 to 1.27.1

| Change | Go | Use here |
| :--- | :--- | :--- |
| Generic methods (`func (w *Workspace) PaneAs[T Pane](…)`); interface methods still cannot have type parameters. Compiled. | 1.27 | typed pane access |
| `strings.CutLast`, `bytes.CutLast` | 1.27 | splitting IDs |
| `encoding/json` backed by v2; `encoding/json/v2` and `encoding/json/jsontext` GA; `GOEXPERIMENT=nojsonv2` opts out | 1.27 | `layout.State`, config, schemas; error text may change |
| `testing/synctest.Sleep` | 1.27 | frame scheduler and leader-key timeouts |
| Size-specialised malloc for objects under 80 B | 1.27 | the ~600 small allocations left per frame |
| `goroutineleak` profile GA | 1.27 | leak checks for workers |
| `go test` runs vet's `stdversion` | 1.27 | — |
| `go fix` adds `slicesbackward`, `atomictypes`, `embedlit`, `unsafefuncs`; renames `waitgroup` to `waitgroupgo` | 1.27 | §1.13 |
| Unicode 17 | 1.27 | none: x/ansi has its own width tables |
| `new(expr)` | 1.26 | option defaults |
| `errors.AsType[T]` | 1.26 | typed errors |
| `reflect.Type.Fields()`, `Value.Fields()` iterators | 1.26 | JSON Schema from structs |
| `T.ArtifactDir()` | 1.26 | golden-mismatch artifacts |
| Green Tea GC on by default | 1.26 | small-object GC |
| `go fix` rebuilt as the modernizers, with `//go:fix inline` | 1.26 | deprecations |
| `testing/synctest` GA | 1.25 | timing tests |
| `T.Attr`, `T.Output()` | 1.25 | test diagnostics |
| `sync.WaitGroup.Go` | 1.25 | tests |
| `os.Root` file methods | 1.25 | already used by tuitest |

Sources: <https://go.dev/doc/go1.27>, <https://go.dev/doc/go1.26>,
<https://go.dev/doc/go1.25>, <https://go.dev/doc/devel/release>.

Not confirmed: the Go 1.26 notes said the Green Tea opt-out would go away
in 1.27. On 1.27.1, `GOEXPERIMENT=nogreenteagc` is still accepted, and the
1.27 notes do not mention it.

## 3. Agent TUIs

### 3.1 crush (Go, Bubble Tea v2, ultraviolet)

The closest analogue to go-tui-lib's first consumer. All verified in source.

- **Streaming Markdown with a stable prefix.** It caches the glamour
  render of a prefix known to be complete, and renders only the tail on
  each flush.
  - The cut is a blank line where no construct is open: fence, list,
    table, block quote, setext heading, HTML block or link-reference
    definition.
  - Fence parity and list state are cached at the cut, so finding the next
    cut costs O(delta).
  - A width change, or content that no longer starts with the prefix,
    drops the cache.
  - The code says two joined renders are not generally equal to one, which
    is why the cut test is cautious.
  - One mutex per renderer, because glamour is stateful.
  - [streaming_markdown.go](https://github.com/charmbracelet/crush/blob/main/internal/ui/chat/streaming_markdown.go)
- **A cached, virtualised chat list.**
  - Items are `Render(width)`, `Version() uint64` and `Finished() bool`,
    and each item's render, lines and height are cached.
  - A finished item is frozen; a version bump unfreezes it.
  - Only visible items render, from an offset index and line.
  - [list.go](https://github.com/charmbracelet/crush/blob/main/internal/ui/list/list.go),
    [item.go](https://github.com/charmbracelet/crush/blob/main/internal/ui/list/item.go)
- **Frame and cell caches.**
  - Whole frames are kept for scroll-only updates: a 3 s TTL, 32 entries,
    and GC on an idle tick.
  - Decoded `uv.ScreenBuffer`s are kept, so a hit is a cell copy with no
    ANSI re-parse.
  - [framecache.go](https://github.com/charmbracelet/crush/blob/main/internal/ui/model/framecache.go),
    [chat.go](https://github.com/charmbracelet/crush/blob/main/internal/ui/model/chat.go)
- **Input filter.**
  - A `tea.WithFilter` filter merges wheel events into one
    `CoalescedWheelMsg` and throttles motion to one sample per 16 ms, so
    keys do not queue behind mouse floods.
  - Its event broker has a lossy publish for token deltas and a blocking
    one (up to 50 ms) for terminal events.
  - [filter.go](https://github.com/charmbracelet/crush/blob/main/internal/ui/model/filter.go),
    [broker.go](https://github.com/charmbracelet/crush/blob/main/internal/pubsub/broker.go)
- **Terminal services.**
  - Clipboard: native first, then OSC 52, because OSC 52 never replies.
  - Notifications: native, OSC 99 (probed), OSC 777, bell or nothing. Off
    without focus events.
  - Kitty images with Unicode placeholders and tmux passthrough.
- **Editor and dialogs.**
  - A large paste becomes an attachment; `$EDITOR` runs at the cursor
    through `tea.ExecProcess`.
  - Permission dialog: allow, allow for session and deny, with a split or
    unified diff. It goes full screen below 77×20.
  - @-completion sorts in tiers: exact base name, prefix, path segment,
    then the rest.
  - Themes are named JSON tokens with one ordered field list.
  - Tests use golden files for every width, height and offset.

### 3.2 opencode (Go, then OpenTUI)

- **History.** opencode-ai/opencode (Go and Bubble Tea) was archived and
  continued as crush. sst/opencode had a Go TUI in `packages/tui` and
  replaced it with OpenTUI (Zig and SolidJS) at 1.0, citing performance.
  [v1.0.0](https://newreleases.io/project/github/sst/opencode/release/v1.0.0)
- **Render cache keyed by a hash of every input** (fnv64a).
  [cache.go](https://raw.githubusercontent.com/sst/opencode/v0.15.0/packages/tui/internal/components/chat/cache.go)
- **Theme JSON with a schema.**
  - `"$schema": "https://opencode.ai/theme.json"` (draft-07), with a
    `defs` palette and semantic tokens as `{dark, light}`.
  - A value is a reference, hex, an ANSI index or `"none"`.
  - [themes](https://github.com/sst/opencode/tree/dev/packages/tui/src/theme/assets),
    [theme.json](https://github.com/sst/opencode/blob/dev/packages/web/public/theme.json)
- **One registry for keys, palette and slash commands.**
  - Commands have dotted names such as `command.palette.show`.
  - Bindings have alternatives (`"ctrl+c,ctrl+d,<leader>q"`), a leader key
    with a timeout, and `false` or `"none"` to unbind.
  - Bindings are layered by mode. Slash names and aliases come from the
    same registry.
  - [keymap.tsx](https://github.com/sst/opencode/blob/dev/packages/tui/src/keymap.tsx),
    [keybind.ts](https://github.com/sst/opencode/blob/dev/packages/tui/src/config/keybind.ts)
- **Custom commands as Markdown files.**
  - User commands live in `~/.config/opencode/commands/*.md`, project
    commands in `.opencode/commands`.
  - Subdirectories become name segments, and `$NAME` placeholders prompt
    for arguments.
  - [README](https://github.com/opencode-ai/opencode#custom-commands)
- **Notifications and clipboard.**
  - It notifies only when unfocused.
  - OSC 52 is wrapped for tmux and screen, with platform clipboard tools
    as fallbacks.

### 3.3 codex (Rust, ratatui, inline viewport)

All verified in source.

- **Finished history goes into the terminal's own scrollback**, written
  above the inline viewport with DECSTBM scroll regions. Untrusted text is
  stripped of control sequences before trusted OSC 8 links are added, and a
  test injects OSC 52.
  [insert_history.rs](https://github.com/openai/codex/blob/main/codex-rs/tui/src/insert_history.rs)
- **Stream controller.**
  - Source text is committed at newlines.
  - Rendered lines split into a stable region and a mutable tail.
  - A pipe table is held in the tail until the stream ends, because a new
    row can change every column width.
  - A resize re-renders the source.
  - [controller.rs](https://github.com/openai/codex/blob/main/codex-rs/tui/src/streaming/controller.rs)
- **Two-gear drain.** Smooth mode drains one line per tick, and CatchUp
  drains the backlog. The switch uses queue depth and the oldest entry's
  age, with hysteresis.
  [chunking.rs](https://github.com/openai/codex/blob/main/codex-rs/tui/src/streaming/chunking.rs)
- **Frame scheduler.**
  - A cloneable `FrameRequester` can be called from any task.
  - A scheduler merges requests and caps drawing at 120 fps.
  - [frame_requester.rs](https://github.com/openai/codex/blob/main/codex-rs/tui/src/tui/frame_requester.rs)
- **Keymap.**
  - TOML contexts: global, chat, composer, editor, the vim modes, pager,
    list and approval.
  - Precedence is context, then global, then defaults. An empty list
    unbinds.
  - Duplicates within one focus path are rejected, with errors that name
    the config path.
  - Slash commands are an enum whose order is the pop-up order.
- **Input and approvals.**
  - `PasteBurst` is a pure state machine that finds pastes on terminals
    without bracketed paste.
  - Approvals are approved, approved for session, a policy amendment,
    denied with a reason, and abort.
  - About 900 snapshot assertions run on a vt100 parser backend.
  - A screen-reader probe sets the animation default.

### 3.4 gemini-cli (TypeScript, Ink)

- **Safe split.** `findLastSafeSplitPoint` finds the last blank line
  outside a code fence. The finished part moves to Ink `<Static>`, which
  never re-renders.
  [markdownUtilities.ts](https://github.com/google-gemini/gemini-cli/blob/main/packages/cli/src/ui/utils/markdownUtilities.ts)
- **A virtualised list.** It has height estimates, stable scrollback and a
  copy mode. A flicker detector warns when a frame is taller than the
  terminal.
- **VS Code-style keybindings.**
  - `keybindings.json` (with comments, validated) maps keys to dotted
    commands.
  - `-command` removes a default.
  - All errors are collected rather than stopping at the first.
  - [keyBindings.ts](https://github.com/google-gemini/gemini-cli/blob/main/packages/cli/src/ui/key/keyBindings.ts)
- **CommandService.**
  - Built-in, TOML file, MCP-prompt and skill loaders run in parallel.
  - A non-built-in command that clashes gets a source prefix, and the
    clash is reported as an event.
- **Tests and access.**
  - Ink output runs through `@xterm/headless` into SVG snapshots.
  - A screen-reader layout is linear.
  - gemini-cli also runs as an ACP agent.

### 3.5 Aider, goose and toad

- **Aider** renders the whole Markdown on each update. It keeps only the
  last 6 lines live, and sets the update interval to
  `min(max(render_time*10, 1/20), 2)` seconds.
  [mdstream.py](https://github.com/Aider-AI/aider/blob/main/aider/mdstream.py)
- **goose** holds back unclosed inline syntax: `push("Hello **wor")`
  returns `"Hello "`. A paste burst on Windows becomes a `[Pasted N lines]`
  chip. goose retired its own ACP TUI.
  [streaming_buffer.rs](https://github.com/block/goose/blob/main/crates/goose-cli/src/session/streaming_buffer.rs)
- **toad** (Textual, an ACP client).
  - **Pruning.** It prunes the transcript by watermark, keeping a low mark
    plus an excess.
  - **Search.** It narrows fuzzy path search with a trigram index.
  - **Danger.** It classifies shell commands as safe, unknown, dangerous
    or destructive, and returns highlight spans for the permission screen.
    Destructive means it writes outside the project root.
  - **Permissions.** Its screen groups keys as allow or reject, once or
    always, as ACP's option kinds are.
  - [conversation.py](https://github.com/batrachianai/toad/blob/main/src/toad/widgets/conversation.py)

## 4. Frameworks and applications

### 4.1 Frameworks

- **Textual.**
  - **Command palette.** It is made of `Provider`s with async `startup`,
    `search(query)` yielding scored `Hit`s, `discover()` for an empty
    query, and `shutdown`.
    - Providers attach globally or per screen. Matching is fuzzy, with a
      0–1 score and highlights.
    - A provider's exception is logged, not fatal.
    - [Command palette](https://textual.textualize.io/guide/command_palette/)
  - **Bindings and keymaps.** Bindings have stable IDs, looked up from the
    focused widget up to the app, with `check_action` for state.
    `set_keymap` remaps by ID.
    [Input](https://textual.textualize.io/guide/input/)
  - **Themes.** Semantic slots, with generated shades and contrast-checked
    text. [Design](https://textual.textualize.io/guide/design/)
  - **Workers.** `exclusive=True` cancels earlier workers, and state
    changes arrive as events.
    [Workers](https://textual.textualize.io/guide/workers/)
  - **Screens.** A screen stack, where `ModalScreen.dismiss(value)` returns
    a typed result. [Screens](https://textual.textualize.io/guide/screens/)
  - **Testing.** A headless `Pilot`, and SVG snapshots.
    [Testing](https://textual.textualize.io/guide/testing/)
- **ratatui.**
  - **Rendering.** Immediate-mode into a cell buffer, flushing the diff.
  - **Layout.** Cassowary. Its docs say conflicting constraints give a
    non-deterministic result, which supports `layout`'s deterministic
    integer solver.
  - **Testing.** Snapshot tests do not capture colour.
  - [Rendering](https://ratatui.rs/concepts/rendering/under-the-hood/),
    [Layout](https://ratatui.rs/concepts/layout/),
    [Snapshots](https://ratatui.rs/recipes/testing/snapshots/)
- **Ink.** Yoga flexbox. `<Static>` keeps permanent output above a live
  region, and `ink-testing-library` exposes the last frame.
  [Ink](https://github.com/vadimdemedes/ink)
- **Notcurses.** z-ordered planes, and pixel blitters. It queries the
  terminal at startup instead of trusting terminfo, and `notcurses-info`
  reports capabilities.
  [Notcurses](https://github.com/dankamongmen/notcurses)
- **FTXUI, tview and gocui.** Not re-checked: a DOM-and-component split, a
  goroutine-safe `QueueUpdateDraw`, and per-view bindings.

### 4.2 The Charm ecosystem

- **lipgloss v2.0.6.**
  - `Style.Hyperlink` (OSC 8).
  - `Canvas`, `Layer` and `Compositor`, with `Compositor.Hit`.
  - `LightDark`, `HasDarkBackground` and `Complete`.
  - The `table`, `list` and `tree` subpackages.
  - [lipgloss](https://pkg.go.dev/charm.land/lipgloss/v2)
- **bubblezone** marks zones with zero-width markers. Its README says v2
  may not work with the lipgloss v2 canvas or compositor. Not to be
  adopted. [bubblezone](https://github.com/lrstanley/bubblezone)
- **huh v2.** Form fields, accessible mode, and Catppuccin and Base16
  themes. [huh](https://github.com/charmbracelet/huh)
- **teatest v2.** `NewTestModel`, `Send`, `Type`, `WaitFor`, and golden
  output.
  [teatest](https://pkg.go.dev/github.com/charmbracelet/x/exp/teatest/v2)
- **x/vt.** A terminal emulator with cell access, input sending,
  scrollback and pluggable sequence handlers. It suits cell-accurate tests
  and embedded terminal panes.
  [x/vt](https://pkg.go.dev/github.com/charmbracelet/x/vt)
- **fang.** Cobra with styled help and errors, man pages, completion and
  version. [fang](https://github.com/charmbracelet/fang)
- **wish** serves a Bubble Tea program per SSH session.
  [wish](https://github.com/charmbracelet/wish)

### 4.3 Applications

- **lazygit.**
  - Keybindings by context. A key may be a list, and `<disabled>` removes
    it.
  - Custom commands with prompts.
  - A published JSON Schema for its config, and XDG paths with merged
    config files.
  - [Config](https://github.com/jesseduffield/lazygit/blob/master/docs/Config.md)
- **k9s.**
  - `:` commands with aliases, `/` filters and `?` help.
  - YAML plugins with a shortcut, scopes, a command, confirmation, and
    context variables.
  - [Plugins](https://k9scli.io/topics/plugins/),
    [Commands](https://k9scli.io/topics/commands/)
- **gh-dash.** Keybindings by view. Keys are written as ultraviolet key
  strings, and commands are Go templates over a context.
  [Keybindings](https://www.gh-dash.dev/configuration/keybindings/)
- **Posting.** `binding-id: key[,key…]` keymaps, and a palette that holds
  some features found nowhere else.
  [Keymap](https://posting.sh/guide/keymap/),
  [Palette](https://posting.sh/guide/command_palette/)
- **zellij.**
  - Sessions are serialised to KDL every second, and resurrected commands
    wait for Enter.
  - WASM plugins request permissions from about 14 kinds.
  - [Resurrection](https://zellij.dev/documentation/session-resurrection),
    [Permissions](https://zellij.dev/documentation/plugin-api-permissions)
- **Helix.**
  - TOML keymaps. Nested tables are sequences, and an array runs commands
    in order.
  - Commands are static or typable (`:write`).
  - [Remapping](https://docs.helix-editor.com/remapping.html)
- **yazi.** Image protocols in order: Kitty placeholders, iTerm2, old
  Kitty, Sixel, Überzug++ and Chafa. The choice comes from the environment.
  [Image preview](https://yazi-rs.github.io/docs/image-preview)

## 5. Terminal standards

Bubble Tea v2.0.10's public API, verified in source:

- **Messages:**
  - `KeyboardEnhancementsMsg`;
  - `BackgroundColorMsg`, `ForegroundColorMsg` and `CursorColorMsg`;
  - `ColorProfileMsg`;
  - `TerminalVersionMsg` and `CapabilityMsg`;
  - `ClipboardMsg` and `EnvMsg`;
  - `ModeReportMsg`;
  - `FocusMsg` and `BlurMsg`;
  - the `Paste*Msg` types and `RawMsg`.
- **Commands:**
  - `SetClipboard` and `ReadClipboard`, and their Primary variants;
  - `RequestTerminalVersion` and `RequestCapability`;
  - `RequestBackgroundColor` and `RequestCursorPosition`;
  - `Raw`.
- **`View` fields:**
  - `AltScreen`, `MouseMode` and `ReportFocus`;
  - `DisableBracketedPasteMode`, `WindowTitle` and `Cursor`;
  - `ForegroundColor` and `BackgroundColor`;
  - `ProgressBar`, `KeyboardEnhancements` and `OnMouse`.
- **Not in the public API:** a colour-scheme message (mode 2031), images
  and notifications.

| Standard | What it gives | Detection | Stack status |
| :--- | :--- | :--- | :--- |
| Kitty keyboard (`CSI = flags u`) | Unambiguous keys, release, repeat, alternates | `CSI ? u` then DA1 | handled: `View.KeyboardEnhancements`, `KeyboardEnhancementsMsg` |
| Kitty graphics (APC `_G`) | Images, placeholders that survive tmux | `a=q` then DA1 (not re-checked) | `x/ansi` sequences; no tea API |
| Sixel | Images | `4` in DA1 (not re-checked) | `x/ansi` helpers; no tea API |
| iTerm2 images (OSC 1337) | Images | environment or XTVERSION | `x/ansi` helpers |
| OSC 8 hyperlinks | Links | none; unsupported terminals ignore it | handled: `Style.Hyperlink`, `ansi.SetHyperlink` |
| OSC 52 clipboard | Copy, also over SSH | none reliable | handled: `SetClipboard`, `ClipboardMsg` |
| OSC 9 / 777 / 99 notifications | Desktop notifications; 99 adds IDs and actions | OSC 99 `p=?` | `x/ansi` OSC 9 and 99 helpers; no 777 helper seen; no tea API |
| OSC 9;4 progress | Taskbar or tab progress | heuristics | handled: `View.ProgressBar` |
| OSC 133 semantic prompts | Jump between and select command output | none | `x/ansi` FinalTerm helpers; no tea API |
| Mode 2026 synchronized output | Tear-free frames | DECRQM 2026 | handled, behind environment heuristics |
| Mode 2027 grapheme clustering | Correct emoji width | DECRQM 2027 | handled in the renderer; see §1.6 |
| Mode 2048 in-band resize | Resize through input, with pixels | DECRQM 2048 | ultraviolet decodes it; whether tea enables it is not re-checked |
| Mode 2031 / DSR 996 colour scheme | Live light and dark | `CSI ? 996 n` → `CSI ? 997;1 n` (dark) or `;2 n` (light) | ultraviolet decodes the events; no tea message |
| Bracketed paste (2004) | Paste apart from keys | — | handled |
| Focus events (1004) | Know when unfocused | — | handled: `ReportFocus`, `FocusMsg` |
| XTVERSION, DA1/DA2, XTGETTCAP | Identify the terminal; terminfo over SSH | queries, with DA1 as sentinel | handled |
| OSC 4 / 10 / 11 / 12 palette | Read the user's colours | `OSC 11;?` | 10–12 handled; OSC 4 not seen in the decoder |
| `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE`, `COLORTERM` | User colour choices | environment | handled by `colorprofile.Detect`; `FORCE_COLOR` not documented |

Sources:

- [bubbletea](https://pkg.go.dev/charm.land/bubbletea/v2) and
  [tea.go](https://github.com/charmbracelet/bubbletea/blob/main/tea.go);
- [ultraviolet's decoder](https://github.com/charmbracelet/ultraviolet/blob/main/decoder.go);
- [Kitty keyboard](https://sw.kovidgoyal.net/kitty/keyboard-protocol/) and
  [Kitty notifications](https://sw.kovidgoyal.net/kitty/desktop-notifications/);
- [OSC 8](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda);
- [mode 2048](https://gist.github.com/rockorager/e695fb2924d36b2bcf1fff4a3704bd83);
- [mode 2031](https://contour-terminal.org/vt-extensions/color-palette-update-notifications/);
- [colorprofile](https://pkg.go.dev/github.com/charmbracelet/colorprofile).

Across the projects read, the trend is to query the terminal first. Terminfo
is a weak fallback, especially over SSH. A capability query should always be
followed by DA1, which every terminal answers, so a missing reply can be
told from a slow one.

### 5.1 Agent Client Protocol

From [tool calls](https://agentclientprotocol.com/protocol/tool-calls):

- **Tool `kind`:** read, edit, delete, move, search, execute, think, fetch,
  switch_mode and other.
- **Tool `status`:** pending, in_progress, completed and failed.
- **Content:** content, diff `{path, oldText, newText}` and terminal
  `{terminalId}`.
- **Permission options:** allow_once, allow_always, reject_once and
  reject_always.
- **Slash commands** arrive in `available_commands_update` as
  `{name, description, input.hint}`.

Not re-checked: which Go ACP SDK is maintained. MCP prompts and resources
feed completion in crush and gemini-cli. This module may never import the
MCP SDK (AGENTS.md, Dependencies), so any MCP or ACP shape is mirrored in
its own types.

## 6. Theme, keymap and config standards

- **base16 and base24.**
  - base16 has eight greys and eight accents. base24 adds two darker
    backgrounds and six bright colours, and maps them to ANSI indexes.
  - The current YAML fields are `system`, `name`, `author`, `variant` and
    `palette`.
  - [base16](https://github.com/tinted-theming/home/blob/main/styling.md),
    [base24](https://github.com/tinted-theming/base24/blob/master/styling.md)
- **Catppuccin.** Four flavours of 26 colours, published as JSON. Not
  re-checked. [palette](https://github.com/catppuccin/palette)
- **VS Code themes.** `colors`, `tokenColors` and `semanticTokenColors`;
  importing `colors` onto semantic slots is feasible. Not re-checked.
- **VS Code keybindings.json.**
  - Entries have `key`, `command`, `when` and `args`.
  - A chord is space-separated (`"ctrl+k ctrl+c"`).
  - `-command` removes a binding.
  - `when` has `&& || ! == != =~`.
  - [Keybindings](https://code.visualstudio.com/docs/configure/keybindings)
- **JSON Schema.** lazygit publishes one with a language-server modeline.
  Emitting one from Go structs is possible with the standard library's
  `reflect`, or with a module that would need a record.
- **XDG base directories.**
  - `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`,
    `XDG_CACHE_HOME`, `XDG_RUNTIME_DIR`, and the `_DIRS` lists.
  - Relative values must be ignored.
  - [Base directories](https://specifications.freedesktop.org/basedir/latest/)

## 7. Candidates, and what the owner chose

Ten items were presented on 2026-10-01:

| # | Item | Prior art |
| :--- | :--- | :--- |
| 1 | `v0.1.1` hardening, and Charm-native and Go 1.27 integration | §1, §2 |
| 2 | A command registry, the single source of agentic commands | opencode, gemini-cli, Textual, fang |
| 3 | A command palette with async providers and fuzzy search | Textual, crush, toad |
| 4 | A keymap engine | codex, VS Code, Helix, lazygit, gh-dash |
| 5 | A streaming content engine | crush, codex, aider, goose |
| 6 | A virtualised transcript and standard panes | crush, toad, gemini-cli, codex |
| 7 | ACP-shaped agent widgets | toad, crush, codex, ACP |
| 8 | Terminal capabilities and services | notcurses, yazi, crush, §5 |
| 9 | Theme system v2 and config standards | Textual, opencode, base16, XDG |
| 10 | A testing and interaction harness | Textual, gemini-cli, codex, x/vt |

**Recommended:**

- item 1 first, because it fixes a crash and a consumer panic in a tagged
  release;
- then items 2, 3, 4, 5 and 8, which give the command surface in the shell
  and the TUI, pi-go's streaming need, and terminal adaptation.

On 2026-10-02 the owner answered: "write findings into a report then follow
recommendations and proceed."

| Item | Record |
| :--- | :--- |
| 1, defects | [0002-MADR, amendment A1](../decisions/0002-MADR-multi-pane-workspace-layouts.md#a1-2026-10-02-v011-hardening), [0002-PLAN-harden-workspace-v0-1-1.md](../decisions/0002-PLAN-harden-workspace-v0-1-1.md) |
| 1, integration | [0004-MADR-integrate-charm-v2-and-go-1-27.md](../decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) |
| 8 | [0005-MADR-terminal-capabilities-and-services.md](../decisions/0005-MADR-terminal-capabilities-and-services.md) |
| 2 | [0006-MADR-command-registry.md](../decisions/0006-MADR-command-registry.md) |
| 4 | [0007-MADR-keymap-engine.md](../decisions/0007-MADR-keymap-engine.md) |
| 3 | [0008-MADR-command-palette.md](../decisions/0008-MADR-command-palette.md) |
| 5 | [0009-MADR-streaming-content-engine.md](../decisions/0009-MADR-streaming-content-engine.md) |

The records are numbered in implementation order, so each cites only
earlier ones. Items 6, 7, 9 and 10 have no record yet. They build on these:

- the transcript on 0009's stream;
- the agent widgets on 0006's danger levels and 0005's services;
- themes on 0004's live theme wiring;
- the harness on everything.

## 8. Second pass: the Kilo, Grok Build, opencode and codex TUIs

### 8.1 The sources

| Project | Upstream | Stack | Commit read | TUI code | Citation prefix |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Kilo CLI | Kilo-Org/kilocode | TypeScript, OpenTUI; a fork of opencode | `e0c27aa71f` (2026-09-30) | `packages/tui/src`, `packages/opencode/src/kilocode` | `packages/` |
| Grok Build | SpaceXAI grok-build, synced from a monorepo | Rust, ratatui and crossterm | `2bdd1d6a` (2026-09-29), `SOURCE_REV` `559751fd` | `crates/codegen/xai-grok-pager*`, `xai-ratatui-*`, `xai-grok-markdown*` | `crates/codegen/` |
| opencode | sst/opencode, v1.18.34 | TypeScript, OpenTUI 0.4.5 (Zig and SolidJS) | `82ea3a3a63` (2026-09-30) | `packages/tui/src`, `packages/opencode/src/plugin/tui` | `packages/` |
| codex | openai/codex | Rust, ratatui and crossterm | `33aea33b39` (2026-10-01) | `codex-rs/tui`, `codex-rs/mermaid`, `codex-rs/message-history` | `codex-rs/` |

- **Citations** name the project, then a path relative to that project's
  prefix. `kilo opencode/src/kilocode/skills/display.ts:6-35` is
  `packages/opencode/src/kilocode/skills/display.ts` in the Kilo tree.
- **Kilo is a fork.** Its own code lives in `kilo`-named directories, and
  its edits to shared files carry `kilocode_change` markers (its root
  `AGENTS.md:173-212`). Every Kilo finding below is Kilo-only unless it says
  it is inherited from opencode.
- **Grok Build had not been read before.** codex and opencode were read
  in §3 at older commits (`d25c114` and `1ddb087`). §8 repeats nothing from
  §3 unless the code changed it materially, and says so where it does.
- **Each finding says where it goes:** an extension of a proposed record
  (0004 to 0009), an existing package, or a new package with no record yet.
  Go names are sketches, not decisions.

### 8.2 Terminal identity and capability facts (extends 0005)

- **Detection is a pure function of an environment map.**
  - Grok's `build_terminal_context_from_env(&HashMap)` returns one struct:
    the brand, the multiplexer, any embedded editor (`NVIM`,
    `VIM_TERMINAL`, `INSIDE_EMACS`), SSH, `TERM`, `VTE_VERSION`,
    `TERM_FEATURES`, and a version kept only when the brand corroborates
    it. (grok `xai-grok-pager-render/src/terminal/mod.rs:60-110,659-761`,
    `xai-grok-pager-render/src/terminal/embedded_editor.rs:1-37`)
  - opencode injects a frozen `TuiTerminalEnvironment{platform,
    multiplexer, displayServer}` and `TuiPaths`. No component reads
    `process.env`. (opencode `tui/src/context/runtime.tsx:3-62`)
  - **For go-tui-lib:** `termcap.FromEnv(env map[string]string, goos
    string) Context`. A pure function lets the golden matrix and `LC_ALL=C`
    runs fake tmux, SSH, Wayland or Windows.
- **Detection order encodes known traps** (grok `xai-grok-pager-render/src/terminal/mod.rs:816-907`):
  1. editor-fork markers that survive SSH and tmux (`CURSOR_TRACE_ID`, then
     `VSCODE_GIT_ASKPASS_MAIN`);
  2. `TERM_PROGRAM`;
  3. `TERMINAL_EMULATOR` for JetBrains, before `TERM_SESSION_ID`, or
     JetBrains reads as Apple Terminal;
  4. `LC_TERMINAL`, which crosses SSH;
  5. `TERM`;
  6. `TERMINATOR_UUID` before `VTE_VERSION`;
  7. `WT_SESSION` last.
- **The raw brand and the refined brand are kept apart.** On native Windows
  grok refines `Unknown` to Windows Terminal, because Windows Terminal's
  default-terminal hand-off omits `WT_SESSION`. A decision that must not
  trust the guess, such as the legacy-console glyph tier, reads the raw
  `env_brand`. (grok `xai-grok-pager-render/src/terminal/mod.rs:248-289,626-636`)
- **Capabilities are per-feature tables that return reasons, not
  booleans.**
  - Examples: `KeyboardCapabilities{cmd, opt: Native|Dropped|Unrecoverable|Unknown}`,
    `HyperlinkCapabilities{osc8: Native|HostileParser|Unsupported|Unknown}`,
    `width_shrink() -> Truncates|Rewraps`.
  - A gate returns a stable token such as `"tmux_extended_keys_off"` or
    `"unknown_no_multiplexer"`. Logs, telemetry and `/doctor` share the
    tokens. Unknown brands fail closed.
  - (grok `xai-grok-pager-render/src/terminal/mod.rs:140-182,355-417`, `xai-grok-pager-render/src/terminal/keyboard.rs:13-119`,
    `xai-grok-pager-render/src/terminal/hyperlinks.rs:10-140`)
  - **For go-tui-lib:** `Context.SkipReason(Feature) (string, bool)`.
    0005's `Fact` type gains a reason, and the 0005 doctor (§8.16) reports
    the same tokens.
- **Kilo fails closed on a missing capability:**
  `capabilities.backgroundSubagents === true` is required, and an absent
  value counts as false. (kilo `tui/src/context/sync.tsx:784-790`)
- **Light or dark is resolved through a chain that survives SSH and tmux**
  (grok `xai-grok-pager-render/src/theme/env_appearance.rs:1-67`,
  `xai-grok-pager-render/src/theme/osc11.rs:1-53`):
  1. the desktop API: macOS `AppleInterfaceStyle`, the XDG portal
     `color-scheme`, or the Windows registry;
  2. `GROK_APPEARANCE`;
  3. `LC_GROK_APPEARANCE`, named `LC_*` because a default `sshd` has
     `AcceptEnv LC_*` and so forwards it;
  4. OSC 11, sent bare first because tmux 3.2 and later answer from the
     pane, then wrapped for tmux on an 80 ms budget, and never inside an
     editor `:terminal`;
  5. `COLORFGBG`, read with Vim's heuristic (0–6 and 8 dark, 7 and 9–15
     light), labelled a guess.
- **A "system" theme is generated from the terminal's own palette**
  (opencode `tui/src/theme/index.ts:346-555`, `tui/src/context/theme.tsx:152-254`):
  - It reads 16 palette colours and the default fg and bg (OSC 4, 10 and
    11, inferred from usage).
  - The mode comes from the background's luminance, not from palette slot
    0, and a test pins that.
  - The background is transparent. A 12-step grey ramp scales from the
    background, and diff tints are ANSI green and red at alpha 0.22 dark or
    0.14 light.
  - The palette is fetched before first paint, and startup waits up to 1 s.
  - A DEC mode 2031 report (`CSI ? 997 ; 1 n` dark, `; 2 n` light) or
    `SIGUSR2` triggers a re-probe at 250 ms and 1000 ms. Refreshes are
    serialised, and an unchanged palette signature is skipped.
  - The user can lock dark or light, and the lock persists.
  - **For go-tui-lib:** `termcap.Palette{FG, BG color.Color; ANSI [16]color.Color}`
    and `theme.FromPalette` (theme v2). The 2031 report becomes a
    `tea.Msg` for 0004's live theme. The caller owns `SIGUSR2` (rule 2),
    and Windows has no `SIGUSR2`.

### 8.3 Probing discipline (extends 0005)

0005 already plans one sentinel-terminated probe. The sources add these
rules:

- **One deadline for the whole group, replies separated from typing.**
  - codex sends `CSI 6n`, `OSC 10;?`, `OSC 11;?`, `CSI ?u` and DA1 last,
    plus DA2 over SSH, under one 250 ms deadline.
  - Bytes read during the probe are replayed into the input parser with
    the colour replies removed, even inside a bracketed paste.
  - Budgets: 1 KiB for one probe, 64 KiB at startup.
  - (codex `tui/src/terminal_probe.rs:26-160,277-310`,
    `tui/src/terminal_probe/startup_replay.rs:1-62`)
- **Apple Terminal over SSH is fingerprinted by DA1 `1;2` plus DA2
  `1;95;0`.** codex then runs inline rather than in the alternate screen.
  (codex `tui/src/terminal_probe/terminal_identity.rs:1-31`)
- **A synchronous probe runs only while the program is the sole stdin
  reader.** Grok reads DA2 synchronously (500 ms, Alacritty only) before
  the event reader exists. XTVERSION is fire-and-forget, and an input filter
  armed for 5 s from the first input batch swallows its reply, which
  crossterm would otherwise surface as Alt+Shift+P plus text. (grok
  `xai-grok-pager-render/src/terminal/xtversion.rs:1-105`, `xai-grok-pager-render/src/terminal/da2.rs:1-70`,
  `xai-grok-pager-render/src/terminal/probe.rs:160-200`)
- **Probes are gated.** Grok skips them under multiplexers that intercept
  CSI and under JetBrains, which paints the query as text. Replies are
  capped at 256 bytes, with a 100 ms grace for late ones.
- **On Windows, trust the native console palette only under
  `ConsoleWindowClass`, not ConPTY.** codex prefers OSC replies and
  preserves and restores console `INPUT_RECORD`s around the probe. (codex
  `tui/src/terminal_probe/windows.rs:32-95`)
- **tmux is asked directly.** codex runs one
  `tmux display-message -p '#{extended-keys-format}\t#{mouse}'`, with
  `show-options -gqv` as the fallback. Grok reads `#{client_flags}` for
  control mode and `#{client_termfeatures}` for RGB. (codex
  `tui/src/tui/tmux.rs:1-70`; grok `xai-grok-pager-render/src/terminal/tmux_probe.rs:200-245`) The
  library spawning a process is a policy question for 0005: return the
  command, and let the caller run it.
- **Resize notifications tmux drops are recovered** by a 500 ms
  window-size sampler with generation invalidation. Ordinary repaints never
  query geometry. (codex `tui/src/tui/size_monitor.rs:1-144`)

### 8.4 Terminal modes, teardown and hand-off (new: `termmode`)

Rule 2 says the caller owns the screen. These sources show what the caller
needs from a library to own it well: data and bytes, never handlers.

- **One restore sequence that is safe in a signal handler.** Grok writes
  `?2026l ?25h ?1000l ?1002l ?1003l ?1015l ?1006l ?2004l ?1004l CSI<u ?1049l`
  with raw `write(2)`. A test pins the kitty pop before `?1049l`, because
  the kitty keyboard stack is kept per screen. (grok
  `xai-crash-handler/src/terminal.rs:35-57`)
- **Kilo's exit safety net is a longer list:** mouse modes
  `?9 ?1000 ?1001 ?1002 ?1003 ?1005 ?1006 ?1007 ?1015 ?1016`, then `?2004`,
  `?1004`, `?1l`, `ESC >`, `?66l`, `CSI > 4;0 m`, `CSI < u` (skipped when
  kitty is off), `?25h` and SGR 0. (kilo
  `opencode/src/kilocode/cli/cmd/tui/util/terminal.ts:8-56`)
- **Normal teardown has an order** (grok
  `xai-grok-pager/src/app/terminal_restore.rs:1-255`):
  1. clear the OSC 9;4 progress bar;
  2. end synchronized output;
  3. OSC 112, only if this session set the cursor colour;
  4. mouse and bracketed paste off, then focus reporting off;
  5. pop the kitty flags at most once, by atomic swap;
  6. reset the cursor shape, only if startup forced one;
  7. leave the alternate screen, or in inline mode park the cursor under
     the last live row;
  8. on Windows, restore the console mode last.
- **A mode ledger repairs a dirty exit.** Grok's `grok wrap` watches every
  CSI a wrapped child sends, keeps a bitmask of DEC modes left on and the
  net kitty push depth, and on an unclean death emits exactly the matching
  resets. A clean exit stays byte-for-byte transparent. (grok
  `xai-grok-pager/src/wrap_restore.rs:1-140`) The same ledger serves an
  embedded terminal pane (§8.12).
- **A DA1 "pop fence" stops key releases leaking into the shell.** After
  `CSI < u`, grok sends DA1 and drains stdin up to the reply (1 s, taken
  from neovim). Without it, a release such as `ESC[100;5:3u` reaches the
  shell, and fish reads it as Ctrl-D and exits. (grok
  `xai-grok-pager-render/src/terminal/pop_fence.rs:1-101`)
- **Pointer policy is per screen.** codex enables alternate scroll
  (`?1007h`) when the mouse is not captured, so the wheel still scrolls
  pickers. When it is captured, it always asks for SGR (`?1006h`), because
  on Windows legacy reports otherwise arrive as key records. Onboarding and
  the static pager leave the mouse to the terminal. (codex
  `tui/src/tui/alternate_screen.rs:1-243`)
- **Hand-off to an external program** (grok
  `xai-grok-pager/src/app/external_editor.rs:1-180`, `xai-grok-pager/src/app/event_loop.rs:379-484`;
  codex `tui/src/tui.rs:599-652`):
  1. park the stdin reader, which must confirm within 500 ms;
  2. drain the writer, for at most 750 ms, and retry later on a timeout
     rather than race the editor;
  3. pop the kitty flags and turn off focus, paste and mouse reporting;
  4. run the program with raw mode off;
  5. push back exactly the same flags;
  6. drain stray input;
  7. in inline mode, re-anchor if the cursor moved.
- **Suspend and resume.** After `fg`, codex disables then re-enables raw
  mode, because the shell may have restored termios behind the cached
  state. It then re-probes the cursor position and runs `tcflush`. opencode
  disables suspend on Windows and maps `ctrl+z` to undo instead. (codex
  `tui/src/tui/job_control.rs:23-120`; opencode `tui/src/config/index.tsx:101-111`)
- **Windows console:**
  - Clear `ENABLE_PROCESSED_INPUT` so Ctrl+C arrives as input, not as
    `CTRL_C_EVENT`. Runtimes re-apply the flag, so opencode re-enforces it
    on the next tick and polls every 100 ms. (opencode `tui/src/terminal-win32.ts:27-130`)
  - Call `FlushConsoleInputBuffer` on exit, so queued mouse and key events
    do not reach the parent shell.
  - Re-ensure `ENABLE_VIRTUAL_TERMINAL_PROCESSING` on restore, use
    input-record mode, and restore the original mode from a stack. (codex
    `tui/src/tui/windows_console.rs:1-104`)
  - Console modes, QuickEdit included, outlive the process, so snapshot
    and restore them. (grok `terminal_restore.rs:217-255`)
  - `golang.org/x/sys` is already an indirect requirement (v0.47.0). A
    direct import needs a record under the dependency rule.
- **Post-teardown output, as an epilogue.** opencode lets components set an
  epilogue that prints only after the renderer is destroyed and every
  finaliser has run. Teardown always clears the title first. (opencode
  `tui/src/context/epilogue.tsx:1-6`, `tui/src/app.tsx:340-364`) Under rule
  1 this becomes a value returned to the caller, which prints it after
  `tea.Program.Run`.
- **For go-tui-lib:** `termmode.Plan{AltScreen, MouseCapture, AltScroll bool}`
  with `Enter() string` and `Leave() string`; `termmode.RestoreSeq`;
  `termmode.Ledger{Observe([]byte); RestoreBytes() []byte}`;
  `termmode.FenceDA1(r, w, d) FenceResult`; build-tagged Windows helpers
  with no-op stubs. Each returns bytes or a `tea.Cmd`, and the caller
  writes or runs it.

### 8.5 Keyboard and input (extends 0007 and 0009)

- **Kitty flags are chosen per terminal and remembered.**
  - codex always sets disambiguate and alternate keys. It adds event types
    only when the terminal is not iTerm2 or Ghostty, which leak releases
    for shortcuts they consume, and is not tmux, unless tmux reports
    `extended-keys-format csi-u`. (codex `tui/src/tui/keyboard_modes.rs:20-145`)
  - Grok withholds event types when DA2 reports Alacritty 0.14 or older
    (packed `2401`), which sends a duplicate legacy release, so Enter
    submits twice. (grok `xai-grok-pager-render/src/terminal/kitty_keyboard.rs:1-102`)
  - Kilo turns kitty off for `TERM_PROGRAM=mintty` and any `MSYSTEM` (Git
    Bash, MSYS2), with environment overrides both ways. (kilo
    `opencode/src/kilocode/cli/cmd/tui/util/terminal.ts:8-24`)
  - codex disables enhancement under WSL when VS Code is detected or
    detection is inconclusive, because it breaks dead-key composition.
  - Grok keeps a ledger of the bits actually pushed. It answers two
    questions: are modified keys disambiguated, and may a binding wait for
    a release?
  - **For 0007:** a binding on a key release (opencode's
    `event: press|release`) degrades to a toggle or a no-op where releases
    are not reported. Grok's hold-to-talk does exactly that.
- **Defaults change by terminal** (grok `xai-grok-pager/src/actions/defaults.rs:9-68,545-890`):
  - VS Code, Cursor, Windsurf and Zed take Ctrl+Q and Ctrl+I, so quit is
    Ctrl+D there.
  - Apple Terminal never delivers Ctrl+Enter.
  - Shift+Enter arrives as a bare CR on VTE before 8200, in the xterm.js
    family, and on unknown brands with no multiplexer, ConHost included.
    Alt+Enter is preferred over SSH and under tmux before 3.3.
  - The footer shows whichever newline chord works. Cmd+Enter is never
    advertised, because many terminals bind it to fullscreen.
  - codex shows Shift+Enter only when kitty keys are on and it is bound,
    otherwise Ctrl+J. (codex `tui/src/bottom_pane/chat_composer.rs:4455-4470`)
  - **For 0007:** a default carries alternatives guarded by capability, and
    help and footers resolve the live chord.
- **Normalise legacy encodings before matching** (codex
  `tui/src/key_hint.rs:83-175`; grok `xai-ratatui-textarea/src/editor_keys.rs:5-205`):
  - a C0 byte with no modifier is Ctrl+letter, except ESC and DEL;
  - an uppercase letter with no Shift flag is Shift plus the lowercase;
  - Ctrl+] arrives as Ctrl+5, and Ctrl+\\ as Ctrl+4;
  - raw BS or DEL is Backspace whatever the modifiers;
  - on Windows, Ctrl+Alt is AltGr and types text;
  - Super+arrow matches with `contains`, so stray Meta or Hyper bits do not
    break it.
- **Recovering dropped modifiers on macOS.** For brands whose modifiers are
  marked `Dropped`, grok polls `CGEventSourceFlagsState`, which needs no
  permission, and upgrades a bare Enter, Backspace or Delete. It is useless
  over SSH. (grok `xai-grok-pager-render/src/input/keyboard_normalizer.rs:1-111`)
  It needs cgo or `purego`, so it needs a record, and CI builds with
  `CGO_ENABLED=0`.
- **Key labels per platform.** `⌥` on macOS and `alt+` elsewhere. Chords
  read `"prefix completion"`. A key-debug view shows the raw event and the
  matched actions, and after 3 s adds "your terminal is not sending that
  key". (codex `tui/src/key_hint.rs:26-30`, `tui/src/keymap_setup/debug.rs:24-60`)
  Glyph labels need ASCII twins (rule 3).
- **Which-key, driven by the keymap's pending state.** opencode's panel
  reads `getPendingSequence()` and `getActiveKeys({includeMetadata})`.
  Escape clears a pending sequence, and Backspace pops one stroke. Its
  layer outranks the app's while visible. (opencode
  `tui/src/feature-plugins/system/which-key.tsx:24-284`, `tui/src/keymap.tsx:53-290`)
  **For 0007:** the engine exposes `Pending() []Stroke` and
  `ActiveKeys() []ActiveKey`.
- **Smaller keymap rules seen in opencode** (`tui/src/keymap.tsx`,
  `tui/src/config/keybind.ts:8-34`):
  - parse aliases are separate from display aliases;
  - the leader token displays the key it is bound to;
  - a mode stack lives on layers, and a dialog pushes `"modal"` and gets
    back a pop function;
  - command visibility is `registered` or `reachable`, and the palette
    lists only reachable commands;
  - `input_paste` defaults to `{ctrl+v, preventDefault: false}`, so the
    terminal's own paste still happens.
- **Hints never go stale.** Tips are templates filled from live bindings,
  and a tip whose command is unbound drops out. Select-dialog footers and
  the diff viewer's `?` table come from the same lookup. (opencode
  `tui/src/feature-plugins/home/tips-view.tsx:120-175`; codex
  `tui/src/tooltips.rs:120-170`) **For 0007:** `keymap.Shortcut(cmd) string`
  returns empty when unbound.
- **A vim engine on a document interface.** Kilo's 879-line engine works on
  `VimDoc{text, cursor, setCursor, insert, remove, undo, redo, setSelection}`:
  motions with counts, `d c y` operators, and visual and visual-line modes.
  It returns `{handled, enteredInsert}`. (kilo
  `opencode/src/kilocode/cli/cmd/tui/component/prompt/vim.ts:1-105`) codex
  keeps undo at the composer level, 64 steps or 1 MiB, so text, chips and
  pastes restore together. (codex `tui/src/bottom_pane/chat_composer/vim_history.rs:17-80`)
- **Input-filter chain** (grok `xai-grok-pager/src/app/{csi_filter,x10_filter,xt_filter}.rs`):
  - reassemble SGR mouse and focus reports split across reads;
  - repair X10 mouse reports that ConPTY and WSL relays corrupt (bytes
    0x80 and above, columns 95 and above);
  - swallow late probe replies;
  - a bare Esc flushes anything held, so Esc never feels slow;
  - re-send the mouse DECSETs on focus-in, because some relays strip them.
  - A PTY scenario pins the X10 leak. **For 0009:** `inputfilter.Chain`,
    tested by splitting each reply at every byte boundary.
- **Typeahead.** Grok captures what the user typed during startup, keeps
  only real typing, cuts at the first Esc, and replays it into the composer
  unless a login or trust screen would swallow it. (grok
  `xai-grok-pager/src/app/event_loop.rs:41-212`) codex does the opposite
  before a security screen: it drains pending input and gives up after 1 s.
  (codex `tui/src/tui/input_boundary.rs:25-117`) Both belong in 0009's
  filter, chosen by the caller per screen.
- **Paste** (opencode `tui/src/component/prompt/index.tsx:78-87,1396-1420`;
  codex `tui/src/bottom_pane/chat_composer/paste_input.rs:1-224`;
  grok `xai-grok-pager/src/app/event_loop.rs:3160-3535`):
  - normalise CRLF and lone CR to LF, because ConPTY sends CR-only line
    ends;
  - an empty bracketed paste means "read the clipboard image", because
    Windows Terminal before 1.25 surfaces an image-only clipboard that way;
  - a pasted path is unquoted, `file://`-decoded, and accepted with drive
    letters and UNC; it is unescaped except on Windows;
  - grok tags each paste with where it came from, the terminal or X11
    PRIMARY;
  - grok detects bursts without bracketed paste (PowerShell): 2 ms for the
    first follow-up, then 10 ms, a run of at least 3 keys (8 for path-shaped
    runs on Windows), and Enter then Ctrl+J is a pasted CRLF, not a submit.
- **Wheel events per notch differ by terminal.** iTerm2, WezTerm and the
  xterm.js family send 1. Apple Terminal, kitty, Ghostty and Alacritty send
  3. Under tmux, screen or zellij it is forced to 1. Grok tells a wheel from
  a trackpad by timing. (grok `xai-grok-pager-render/src/input/mouse.rs:1-77,246-330`)
  **For 0009:** the wheel coalescer takes a profile from `termcap`.
  opencode lets the user choose macOS-style acceleration or a fixed speed.

### 8.6 Inline, scrollback and screen modes (new: `inline`)

§3.3 recorded codex's DECSTBM history insertion. The two Rust TUIs now go
much further.

- **Three presentations over one model, switchable at run time.** Grok's
  `ScreenMode{Fullscreen, Inline, Minimal}`:
  - Minimal prints each finished block once into native scrollback, and
    keeps a pinned live region for the running turn, todos, prompt and
    status.
  - **The committed frontier:** only the leading run of finished entries
    that are not waiting on the user is printed. A block waiting for a
    permission answer holds the frontier back, so its waiting form is never
    frozen into scrollback.
  - A debounced resize clears the screen and scrollback, then reprints the
    newest 4000 rows at the new width.
  - `/minimal` and `/fullscreen` switch live: park the reader, leave the
    alternate screen or clear (never `3J`, which destroys the user's own
    scrollback), toggle mouse capture, re-query the cursor, and roll back
    on failure.
  - `AltScreenMode{Auto, Always, Never}`. Auto runs inline under Zellij and
    tmux control mode.
  - (grok `xai-grok-pager-minimal/src/{lib.rs:1-60,commit.rs:1-251,reprint.rs:1-60}`,
    `xai-grok-pager/src/app/mode_switch.rs:1-196`)
- **Writing rows without scroll regions keeps soft wraps.** Grok's
  `insert_before_rows`:
  - a full-width soft-wrapped row omits `\r\n`, so the terminal marks the
    line as wrapped and native copy rejoins it;
  - a full-width hard break is written with autowrap off (`?7l … ?7h`)
    after a dummy printable, so a pending wrap cannot join the next line;
  - `CSI K` only for short rows;
  - a purge uses `CSI 2J`, `CSI 3J`, `CSI H`, not RIS, which does not clear
    scrollback in iTerm2 or Terminal.app;
  - a per-brand `Truncates|Rewraps` model estimates where the cursor lands
    after the terminal re-wraps on shrink, erring low.
  - (grok `xai-ratatui-inline/src/{scrollback.rs:8-71,resize.rs:8-149,terminal.rs:804-910}`)
- **codex picks a scrollback strategy per terminal** (codex
  `tui/src/tui/scrollback.rs:19-104`, `tui/src/resize_reflow_cap.rs:1-76`):
  - `FullScreen` under Windows Terminal, where partial scroll regions
    discard rows instead of moving them into scrollback;
  - `Zellij`, which becomes full-screen when the terminal owns line wrap;
  - `Standard` elsewhere, growing the viewport with `\r\n` instead of
    `CSI S`, which discards departing rows in QTermWidget and xterm.js;
  - DECSTBM needs two distinct rows, so a one-row insert goes full-screen;
  - replayed rows on reflow are capped at each terminal's scrollback
    default: VS Code 1000, Windows Terminal 9001, WezTerm 3500, Alacritty
    10000, otherwise 1000.
- **For go-tui-lib:** an `inline` package that returns sequences for the
  caller to write: `inline.Strategy(termcap.Context)`,
  `inline.Commit(rows []Row) string` with `Row{ANSI string; FillsWidth, SoftWraps bool}`,
  `inline.ReflowCap`, and a `Committer` that holds the frontier. It needs
  `CSI 6n`, and falls back to full screen when that times out. 0009
  considered and rejected inline scrollback as its option D. A record for
  this package would take that up again.

### 8.7 Transcript and standard panes (new: item 6 of §7)

- **Positions are content anchors, not row offsets.**
  - codex: `Position` is `Latest` or `Reading(Anchor{key, index hint, byte offset, row_bias})`.
    Prepending older pages finds the anchor again by key. A width change
    maps the same byte offset back to its new row. (codex
    `tui/src/transcript_view.rs:1-620`)
  - Grok keeps a logical anchor `{entry, logical_line, sub_rows}` for width
    rebuilds, and a structural anchor `{id, rows_into_span}` for inserts,
    dropped if the user navigated meanwhile. (grok
    `xai-grok-pager/src/scrollback/state/mod.rs:1316-1470`)
  - Kilo's webview keeps scroll state as `{bottom}` or `{anchor: rowKey, offset}`,
    in an LRU of 50. (kilo `kilo-vscode/webview-ui/src/components/chat/transcript-cache.ts:1-85`)
- **Heights are estimated first and measured only when visible.** A width
  change gives every entry a cheap estimate. Prefix sums and a binary search
  find an entry by row. Pre-measuring above the bottom waits until a resize
  settles, which profiling showed was the largest resize cost. (grok
  `xai-grok-pager/src/scrollback/state/layout.rs:12-118`, `xai-grok-pager/src/views/list_pane/layout.rs:1-80`)
- **The layout cache is bounded and invalidated by render state.** codex
  holds at most 64 entries and 8 MiB, keeping one oversized entry so a huge
  visible cell is not re-laid every frame. It clears everything when the
  theme revision, terminal fg or bg, or colour level changes. A snapshot pins
  the layouts on screen while the user reads or selects inside a cell that
  is still streaming. (codex `tui/src/transcript_view/layout.rs:1-277`,
  `tui/src/transcript_view/snapshot.rs:1-122`)
- **One source-mapped layout serves painting, hit-testing, search and
  copy.** Each codex row carries its source range, a hard-end flag, its
  prefix columns and its tab stops. Gutters and labels never enter the text.
  (codex `tui/src/transcript_view/text.rs:1-90`, `tui/src/wrapping.rs:718-760`)
  Grok's wrap returns "joiners", the exact substring dropped at each soft
  wrap, so copy rebuilds wrapped text byte for byte. (grok
  `xai-grok-pager-render/src/render/wrapping.rs:146-237`)
- **A cell or block contract** (codex `tui/src/history_cell/mod.rs:155-345`;
  grok `xai-grok-pager/src/scrollback/block.rs:40-195`):
  - required: lines for a width and a mode, and a height;
  - hints: stable height, an animation tick, groupable, selectable;
  - display modes: grok has `Collapsed | Truncated | Expanded`, plus
    `finished_display_mode` (collapse when done, expand on error);
  - codex adds a user-toggled `Raw` mode with no gutters or borders, so
    native terminal selection copies cleanly where mouse capture is off.
- **Disclosure keyed by stable activity IDs.** Hidden output is never
  rendered: its size comes from metadata. The expanded set survives
  regrouping and replay. The `+ N lines (ctrl+t to expand)` row is
  synthetic, so it is not selectable and is skipped by copy and search.
  (codex `tui/src/transcript_view/disclosure.rs:1-140`)
- **Follow the tail, and come back.** opencode sticks to the bottom until
  the user scrolls up. codex shows a clickable `New activity · ↓ Back to bottom · esc`
  pill only when the last row is off screen, choosing the first of five
  label widths that fits. A bookmark saves position, presentation and unseen
  activity, and Esc restores it exactly. (codex
  `tui/src/transcript_view/follow_control.rs:1-111`, `tui/src/transcript_view/bookmark.rs:1-46`)
- **Sticky section headers, as pure arithmetic.** The next header pushes
  the pinned one off by `clip_top`. A one-row header shrink cancels a
  one-row scroll, so each step moves the content by exactly one row. (grok
  `xai-grok-pager/src/scrollback/sticky.rs:1-140`) The palette's groups
  (0008) can use it too.
- **Search** (codex `tui/src/transcript_view/search.rs:1-734`; kilo
  `kilo-vscode/webview-ui/src/components/chat/transcript-search-text.ts:10-63`):
  - codex scans at most 8 entries or about 16 KiB per frame, then asks for
    another frame. At the oldest loaded entry it fetches a page, and scans
    only what arrived.
  - Scan windows overlap, because lowercasing can change a character's
    byte length.
  - Kilo indexes only user and assistant text, because tool output
    "dominates a long transcript by volume". It strips invisible link URLs
    so that match counts equal what is on screen.
- **The store** (kilo `tui/src/kilocode/hydration.ts:15-72`,
  `tui/src/kilocode/message-order.ts:1-22`):
  - Order by creation time with an ID tie-break. Kilo stopped trusting ID
    order because "ids wrap".
  - `hydrate(before, snapshot, live)` merges a refresh that raced live
    events: keep the snapshot except where live changed; merge ID-keyed
    arrays by ID; when one string is a prefix of the other, the longer
    wins, so streamed text never goes backwards.
  - codex's background-thread buffer coalesces adjacent deltas up to 4 KiB,
    is bounded at 256 KiB, and records any evicted server request so it can
    be replayed safely. (codex `tui/src/app/thread_event_buffer.rs:1-77`)
- **A reusable list pane** (grok `xai-grok-pager/src/views/list_pane/mod.rs:1-206`):
  - It keeps scroll, selection and layout only. Items stay with the model
    and are borrowed each frame.
  - Items give content, prefixes for each state (so the cursor is not
    shown by colour alone), a stable ID, search text and copy text.
  - Search matches are REVERSED, so they need no colour.
  - Each key family can be switched off: follow, wrap, search, filter,
    copy, visual select, goto. A disabled key is not consumed.
  - Fixed heights are O(1) and variable heights O(log n), with incremental
    extension for append-only logs of 100k lines or more.
  - One component serves a trace log, the todo list and the background
    tasks. bubbles `list` has neither wrap nor stable IDs.
- **Dock rows are granted fairly.** `grant_rows` gives each section one
  row, then a floor, then a "show N more" row (1+1+1 beats 2+2+0). The
  layout is solved once and shared by paint, cursor and hit-testing. (grok
  `xai-grok-pager/src/views/dock/layout.rs:1-283`) It fits `layout` as `GrantRows`.

### 8.8 Composer (new)

bubbles v2 `textarea` has no atomic elements, no undo and no selection.
All four projects built these.

- **Atomic elements, or chips.**
  - A side list of `{id, range, kind, display}` lies beside the raw text.
    The cursor steps over an element, and a delete takes it whole. The
    displayed text may differ from the bytes, as in `[Pasted 120 lines]`.
  - Edits shift element ranges. Grok's `inline_element(id)` turns a chip
    back into plain text as one undo step.
  - opencode re-syncs extmark positions to prompt parts after every edit,
    drops parts whose token was deleted, and expands tokens in reverse
    offset order on submit. Copying a selection expands placeholders back to
    real text.
  - (grok `xai-ratatui-textarea/src/textarea.rs:28-340`; codex
    `tui/src/bottom_pane/textarea.rs:75-200`; opencode
    `tui/src/component/prompt/index.tsx:658-733,1149-1270`)
- **Undo that merges by kind** (grok `textarea.rs:2099-2716`):
  - snapshots of text, cursor and elements, 100 deep;
  - consecutive inserts or deletes merge unless the cursor moved; a kill,
    an element or a replace starts a new step;
  - groups nest, and cancelling a group returns to its checkpoint with no
    entry left behind.
- **A pure edit core with checked plans.** `plan_command` returns a plan
  stamped with the buffer's identity and generation. Applying it after the
  buffer changed fails with `StalePlan`. The same core drives search fields
  and dialog inputs, so readline behaviour is identical everywhere. (grok
  `xai-ratatui-textarea/src/editor.rs:13-640`)
- **Word motion.** A run of separators is a single stop, so Alt+Backspace
  on `foo.bar` deletes `bar`, then `.`, then `foo`. Each CJK character is
  its own word. The single kill buffer survives submit. Ctrl+W and
  Alt+Backspace use different word classes. (codex `textarea.rs:605-742`;
  grok `editor_keys.rs`)
- **Paste chips; the thresholds differ:**

  | Project | Chip when | Again-to-expand |
  | :--- | :--- | :--- |
  | Kilo | 5 lines or 800 chars | yes |
  | opencode | 3 lines or 150 chars | — |
  | Grok | 4 lines or 10 kB | yes; Ctrl+Shift+V pastes inline |
  | codex | 1000 chars, with `#2` suffixes when sizes repeat | — |

  The threshold is a policy for the caller to set.
- **Images and attachments.** `[Image #N]` chips are renumbered after a
  delete. The clipboard's file list is preferred over raw pixels. Under WSL,
  `powershell.exe` then `pwsh` saves the image, and `C:\` maps to
  `/mnt/c`. (codex `tui/src/bottom_pane/chat_composer/attachment_state.rs:1-251`,
  `tui/src/clipboard_paste.rs:1-380`)
- **History** (opencode `tui/src/prompt/history.tsx:27-110`; codex
  `tui/src/bottom_pane/chat_composer_history.rs:1-484`,
  `message-history/src/lib.rs:1-446`):
  - Up and Down recall only when the draft is empty or still equals the
    recalled entry, so a draft is never clobbered.
  - Ctrl+R searches backwards in the footer and restores the draft on Esc.
  - codex's `history.jsonl` is mode 0600, appended with single
    `O_APPEND` writes under an advisory lock, trimmed to 80% over its cap,
    and detects a replaced file by inode.
  - opencode's JSONL stores parse tolerantly, cap, then rewrite to heal
    corruption.
- **Stash.** Ctrl+S saves text, cursor, images and chips. The border reads
  "Stashed", and the draft returns after the next send. (grok
  `xai-grok-pager/src/views/prompt_widget/mod.rs:420-445`; opencode `tui/src/prompt/stash.tsx:15-88`)
- **Queue and steer while a turn runs.** Grok queues on Enter, and Enter on
  an empty composer sends the top entry now. Steer mode slips a prompt in at
  the next safe gap. Queue entries carry `{id, version, owner, last_editor}`,
  so an edit against an old version does nothing and several clients can
  share one queue. (grok `xai-prompt-queue/src/types.rs:1-90`,
  `xai-grok-pager/src/views/queue_pane.rs:1-655`)
- **Shell mode.** `!` enters it only at offset 0 with completion closed.
  Escape, or Backspace at offset 0, leaves it. The editor declares which
  keys it captures in each state, so the keymap knows what it swallows.
  (opencode `tui/src/component/prompt/index.tsx:820-860`,
  `tui/src/prompt/traits.ts:16-28`) The mode needs a text label as well as a
  border colour (rule 4).
- **External editor** (grok `external_editor.rs:1-180`; codex
  `tui/src/external_editor.rs:1-230`; opencode `tui/src/editor.ts:12-54`):
  - `$VISUAL`, then `$EDITOR`, split with shell-word rules, and `.cmd` shims
    resolved on Windows;
  - opencode's `split(" ")` breaks quoted paths; do not copy it;
  - Grok refuses to launch while chips exist or a paste is in progress, and
    rejects the result on a non-zero exit, more than 4 MiB, invalid UTF-8,
    or a draft that changed meanwhile.
  - **For go-tui-lib:** a `tea.ExecProcess` command returning `EditedMsg` or
    `EditFailedMsg{Reason}`, over §8.4's hand-off.

### 8.9 Completion, palette and pickers (extends 0008)

- **A bash-style Tab policy** (grok `xai-grok-pager/src/views/suggestion_controller/mod.rs:168-527`):
  - one candidate: accept;
  - several sharing a longer prefix: fill the prefix;
  - otherwise: open the drop-down;
  - a stale answer, or a cursor moved other than by typing: do nothing;
  - results the provider truncated never accept or fill on their own.
  What to write is decided before the drop-down closes, and checked against
  the request's text and cursor.
- **Ghost text** is a dim italic suggestion tagged with its source
  (history, executable, path, AI). It is trimmed as typed text matches, with
  no new request. Right accepts all of it, Ctrl+Right one word. Dimming
  alone carries meaning, so it needs a no-colour cue.
- **Triggers** (opencode `tui/src/component/prompt/autocomplete.tsx:27-160`):
  `@` triggers only at the start or after whitespace, so email addresses are
  safe. `/` triggers only at offset 0. A `#12-20` suffix is a line range.
  Grok adds `@q/` for directories only and `@!q` to include hidden files.
- **Ranking.** opencode doubles the fuzzy score when the display starts with
  the query and multiplies by `1 + frecency`, where frecency is
  `frequency / (1 + days since last use)`. Grok breaks slash-command ties by
  recent use with a 7-day half-life. codex's file search reparses
  incrementally when the query only grows. (opencode
  `tui/src/prompt/frecency.tsx:10-79`; grok `xai-grok-pager/src/slash/mru.rs:1-104`; codex
  `file-search/src/lib.rs:40-610`)
- **Providers degrade, they do not crash.** When threads cannot be spawned
  (cgroup or `RLIMIT_NPROC`), grok's file search drops to browse-only, then
  to disabled. (grok `xai-fuzzy-file-search/src/lib.rs:393-460`)
- **Mention providers with URI-scheme parts.** Kilo's `@` menu has "Past
  chats", which inserts a part with `url: session:<id>` that the server
  expands into context. A failed list degrades to an empty picker. (kilo
  `tui/src/kilocode/session-mentions.ts:10-82`)
- **The select dialog** (opencode `tui/src/ui/dialog-select.tsx:23-330`):
  - An input-modality latch (keyboard or mouse) resets to keyboard
    whenever results change. This ignores the synthetic mouse-over a
    terminal reports as rows move under a still pointer.
  - `preserveSelection` keeps the cursor on the same value across live
    updates.
  - `onMove` gives a live preview, so the timeline dialog scrolls the
    transcript behind it.
  - Kilo adds list-level actions with `requiresSelection: false`, which
    work on an empty list. (kilo `tui/src/ui/dialog-select.tsx:39-70`)
- **List rows** (codex `tui/src/bottom_pane/selection_popup_common.rs:1-460`,
  `tui/src/bottom_pane/list_selection_view.rs:158-1213`):
  - match indices in bold, a shortcut, and a `disabled_reason`;
  - the description column sits at the widest name plus 2, capped at 70%,
    measured over all rows so it does not jitter while scrolling;
  - a searchable list types plain keys into the filter, and a plain list
    uses `j`/`k` and digits;
  - `require_explicit_confirmation` makes a shortcut only highlight a
    sensitive row, so Enter is still needed.
- **Paged pickers.** codex's resume picker loads pages of 25, prefetches
  with 5 rows left, discards stale pages by request token, and uses a
  tombstone so a page in flight cannot revive an archived row. (codex
  `tui/src/resume_picker.rs:92-930`, `tui/src/resume_picker/archive.rs:1-110`)

### 8.10 Rendering content (extends 0009; new `table`, `diffview`, `canvas`, `highlight`, `termimage`)

- **Side-tables survive each freeze of the stream.** Every grok render
  returns a line-to-source map, hyperlinks with IDs and column ranges, code
  blocks with their source ranges, and table copy metadata. Each is cut back
  and carried forward when the tail re-renders. Checkpoints form only at
  depth 0. Link IDs continue across tail renders. (grok
  `xai-grok-markdown/src/{streaming.rs:105-372,checkpoint.rs:1-60,output.rs:12-66}`)
  This is what makes "copy code block", link hit-testing and selection work
  while a reply is still streaming. **For 0009:** the engine returns a
  `RenderView` with these tables, not only lines.
- **Model-output dialect.** Grok counts only `~~` as strikethrough, so
  `~**10%**` stays literal, and holds a partial `\(` at a chunk edge until
  `finish()`.
- **Resumable highlighting of the open fence.** Grok keeps the parse and
  highlight states after the last committed newline, so each line is
  highlighted once: O(N) instead of O(N²), about 35 ms saved per push near
  the end of a 1000-line block. Closed fences in the tail are memoised by
  `(info, body)` under a 256 KiB budget. codex does the same with a
  `THEME_REVISION` counter that drops every cache. (grok
  `xai-grok-markdown/src/open_code_highlighter.rs:1-80`; codex
  `tui/src/render/highlight_streaming.rs:1-115`)
- **Highlighting limits and themes** (codex `tui/src/render/highlight.rs:84-775`):
  - 512 KiB, 10,000 lines and 4 KiB per line; over any of them, plain text;
  - only the first token of a fence's info string (`rust,no_run`), and
    aliases such as `golang` to `go`;
  - bat's ANSI-theme convention: alpha `0x00` means `r` is an ANSI index,
    and alpha `0x01` means the terminal default;
  - opencode styles by tree-sitter capture names (`markup.heading.1`,
    `comment.documentation`, `conceal`) with dotted fallback, and a "subtle"
    variant at reduced opacity for reasoning text. (opencode
    `tui/src/theme/index.ts:556-918`)
  - Go tree-sitter bindings need cgo. A lexer such as chroma needs a
    record, and cannot resume a saved state, so the open fence re-lexes
    only its uncommitted tail.
- **Tables at narrow widths** (codex `tui/src/markdown_render.rs:1354-1749`,
  `tui/src/markdown_render/table_key_value.rs:18-305`; grok
  `xai-grok-markdown/src/render.rs:1344-1606`):
  - codex classes columns as token-heavy, narrative or compact, and shrinks
    them in that order. It falls back to key/value records, then stacked
    records, then the raw pipe source.
  - Grok sizes from the widest cell, then from word minimums, then shares
    space in proportion to each column's want. `$145,000` and `3.14` stay
    whole, and URLs never break.
  - Grok pads or clips each row by grapheme to the exact width, so a wide
    glyph cannot leave a ghost cell.
  - Grok's copy metadata gives two selection modes: by cell, where wrapped
    fragments rejoin, and by grid, copied as TSV.
  - Kilo normalises tables with a hand-written East Asian width table. Use
    `x/ansi` instead: Kilo's table misses emoji and ZWJ sequences. (kilo
    `tui/src/util/markdown.ts:1-212`)
- **Copy as Markdown.** Each rendered codex line carries copy metadata, so
  a selection copies back as Markdown: prose re-escaped, fences one backtick
  longer than the longest run inside, and leading spaces protected. (codex
  `tui/src/markdown_copy.rs:1-557`)
- **Diffs** (codex `tui/src/diff_render.rs:17-1363`; grok
  `xai-grok-pager-diff/src/lib.rs:11-317`; opencode
  `tui/src/feature-plugins/system/diff-viewer.tsx:38-1069`; kilo
  `opencode/src/kilocode/tui/diff.ts:1-82`):
  - codex: a right-aligned number gutter, a `+`/`-` sign column, and a
    line-level background that reaches the right edge; ANSI-16 gets a
    foreground only, because a tinted ANSI-16 background is not
    representable; the light and dark palette comes from OSC 11; each hunk
    is highlighted as one block, and the result is used only if line counts
    match.
  - Grok stitches successive edits to one file in post-state coordinates,
    and refuses to merge rather than show anything untrue. A first paint
    highlights each hunk alone, and a worker later upgrades it with the
    whole file if the file on disk still matches.
  - opencode splits when wider than 120 cells (100 in the viewer). Its
    viewer has a file tree, hunk and file navigation, and mark-reviewed.
  - Kilo splits patches per file and per hunk, using hunk line counts to
    tell real `---` headers from content, and shows "No changes to review"
    for a header-only patch.
  - **Do not copy grok's diffs:** they have no sign column, so insert and
    delete show only by colour.
- **Tool output with escape codes, interpreted.** Grok runs captured PTY
  bytes through a parser into styled rows. It handles SGR in `;` and `:`
  forms, CR overwrite (progress bars), BS, tabs, and `CSI K J A B C D G`,
  and drops the rest. The limits are 50k rows and 8192 columns. It
  advances one column per character, which a cell-aware version must fix.
  (grok `xai-grok-pager-render/src/render/terminal_output.rs:1-330`) 0009's
  sanitizer strips escapes, and this interprets them.
- **Bounded live output.** codex keeps up to 1 MiB, then the first 50 and
  last 50 lines, with a per-line byte cap so output with no newlines stays
  bounded. An escape cut at the head is closed before the
  `... N bytes omitted ...` marker. Truncation counts wrapped rows but
  reports logical lines, so the count does not change with width. (codex
  `tui/src/exec_cell/live_output.rs:1-259`)
- **Diagrams and math** (grok `xai-grok-markdown/src/mermaid.rs:1-1772`;
  codex `mermaid/src/lib.rs:20-176`, `tui/src/markdown_render/math.rs:1-307`):
  - Grok's canvas cells hold an up/down/left/right bitmask. ORing edges in
    and mapping the mask to `│ ─ ┌ ┼ …` makes crossings merge on their own.
    A light Sugiyama layout places nodes. The limits are 128 nodes and 512
    edges, and anything past them shows the source.
  - codex renders flowchart, sequence, state, class and ER diagrams as
    role-tagged spans, all or nothing, only for completed fences.
  - codex lays out `\frac`, scripts and `\sum` limits in Unicode, capped at
    16 by 256 cells, and rejects `$5` and `$PATH` as math.
  - Grok renders a Mermaid PNG only on click, in a re-executed subprocess
    with a time budget, because a panic in-process cannot be contained
    against untrusted model output.
  - A switch per renderer (mermaid, math, tables, lists) keeps the source
    when one is off. (codex `config/src/tui_rendering.rs:1-31`)
  - **For go-tui-lib:** a `canvas` package that takes a glyph set, so the
    ASCII twin (`+ - |`) comes free. Neither project has one.
- **Images** (grok `xai-grok-pager-render/src/terminal/image.rs:1-621`;
  codex `tui/src/pets/image_protocol.rs:1-310`, `tui/src/pets/sixel.rs:1-230`):
  - Kitty graphics in 4096-byte chunks with `q=2` and `C=1`; transmit and
    place separately; `z=-1` keeps scrollback images under text so
    drop-downs paint over them;
  - Warp ignores placement replace, so it gets delete then transmit;
  - iTerm2 3.6 and later accepts Kitty with a local-file payload (`t=f`).
    Grok uses OSC 1337 only when `TERM_FEATURES` contains `F`;
  - Sixel for Windows Terminal, foot and mlterm, with a built-in RGB332
    encoder in codex;
  - off under tmux and Zellij, and grok also turns it off on Windows
    because ConPTY strips APC;
  - cell aspect from the pixel window size, accepted between 0.3 and 0.8,
    otherwise 0.5;
  - no partial clipping, and a fixed first frame under reduced motion.
- **Charts in cells.** Bars use eighth blocks `▁…█`, ticks are "nice"
  numbers, and the heatmap falls back to `□`/`■` on ANSI-16. (codex
  `tui/src/analytics/plot/painting.rs:1-132`) Each glyph needs an ASCII twin.

### 8.11 Theme, colour and glyphs (theme v2, item 9 of §7; extends 0004)

- **Resolve readable foregrounds against the real background.** codex's
  `contrast::foreground`:
  - in TrueColor, keep a colour that passes WCAG 4.5; otherwise blend it
    toward black or white in bounded steps, keeping as much hue as possible;
  - in 256 colours, take the nearest index from 16 to 255 that passes,
    skipping 0–15 because themes redefine them;
  - in ANSI-16 or unknown, `Reset`;
  - cache up to 512 results.
  - Secondary text blends fg into bg at 0.6 instead of DIM. "Attention"
    is yellow only on a known dark background.
  - A clippy `disallowed-methods` rule bans raw RGB and indexed colours, so
    every colour goes through these helpers.
  - (codex `tui/src/style/contrast.rs:1-151`, `tui/src/style.rs:28-270`,
    `clippy.toml:8-13`)
  - **For go-tui-lib:** `theme.Readable(pref, bg, profile)` and
    `theme.Blend`, plus a `forbidigo` rule in `.golangci.yml` banning
    literal colours outside `theme`. `go-colorful` is already an indirect
    requirement.
- **A "terminal-native" theme.** Grok's default leaves body text and every
  background at `Reset`, so the terminal's own canvas shows through,
  translucency included. Accents are named ANSI-16 only, de-emphasis is SGR
  DIM, and selection is REVERSED. The stated reason: polarity detection is
  unreliable, but a profile's default fg and bg always read against each
  other. (grok `xai-grok-pager-render/src/theme/terminal_default.rs:1-298`)
  This is the natural default under rule 4.
- **Decide light or dark before quantising** (grok
  `xai-grok-pager-render/src/theme/mod.rs:72-480`, `xai-grok-pager-render/src/theme/color_support.rs:81-266`):
  - At ANSI-16, chrome slots are pinned. Accents take the bright hue on
    dark backgrounds and the normal hue on light.
  - Themes with tinted backgrounds report that they need TrueColor. The
    picker hides them below it.
  - Syntax colours on the native palette: near-grey tokens become the
    default fg, and coloured ones map by hue to the base six, never to white,
    black or the brights, which vanish on light profiles.
  - Structural colours are pushed apart on Windows, because ConHost and
    display gamma collapse small steps.
  - The colour level is promoted to TrueColor for brands known to support
    it, because tmux, SSH and mosh strip `COLORTERM`. ConHost has had
    24-bit colour since Windows 10 1709 but does not advertise it.
- **Unfocused panes recede.** With an RGB background, each cell's fg
  blends toward the pane background. With `Reset` or a named background it
  sets DIM and removes BOLD, because many terminals ignore faint on bold.
  Named ANSI cells are never blended. (grok
  `xai-grok-pager-render/src/render/color.rs:150-297`) Focus still needs a
  non-colour cue, such as the border glyph.
- **Token taxonomy.** opencode has about 55 semantic tokens: core, status,
  text, four backgrounds, three borders, diff (with per-side line-number
  backgrounds), markdown, syntax, and a `thinkingOpacity`. Optional tokens
  fall back (`backgroundMenu` to `backgroundElement`). Reference cycles fail
  with the cycle named. (opencode `tui/src/theme/index.ts:60-299`) A theme
  v2 record needs diff and markdown tokens for the agent widgets.
- **A colour-blind preset.** Kilo ships Paul Tol's "vibrant" palette, with
  diff added teal and removed vermillion, not green and red. A custom theme
  is accepted only with `background`, `text` and `primary`, cannot shadow a
  built-in name, and one bad file never discards the others. (kilo
  `tui/src/theme/assets/colorblind.json:1-111`, `tui/src/context/theme.tsx:59-321`)
- **Identity colours.** codex hashes a thread ID with FNV into a palette
  taken from the theme. (codex `tui/src/thread_color.rs:1-83`)
- **A three-tier glyph table.** Grok returns the Unicode glyph normally. On
  legacy ConHost (raster fonts, no fallback), it returns a CP437 glyph where
  that reads better (`√ ♦ ○ • ↓`) and ASCII otherwise. Every glyph documents
  "always N columns", so spinner frames never shift layout. On Windows an
  unknown brand counts as legacy, because bare `cmd.exe` sets no terminal
  variables. (grok `xai-grok-pager-render/src/glyphs.rs:1-420`)
  - Kilo decides whether its Unicode logo will render: on Windows only under
    `WT_SESSION`, VS Code or WezTerm; ASCII under `TERM=dumb`, `ConEmuPID`
    and `ANSICON`. (kilo `opencode/src/kilocode/cli/logo.ts:35-80`)
  - **For `glyph`:** a legacy-console tier between Unicode and ASCII, a
    test that every tier has the same width, and a legacy-console row in the
    golden matrix.

### 8.12 Agent widgets (ACP-shaped, item 7 of §7)

- **The permission prompt is a staged flow** (opencode
  `tui/src/routes/session/permission.tsx:116-640`; kilo
  `opencode/src/kilocode/permission/provenance.ts:10-80`; grok
  `xai-grok-pager/src/views/permission_view.rs:1-700`):
  - Stage one offers allow once, allow always and reject, with a body per
    permission kind: a diff for edits, a path for reads, a pattern for grep.
  - "Always" confirms the exact patterns it will save. Grok lets `←`/`→`
    widen or narrow how many words of a command are remembered, stepping
    only through sizes that would persist. Dangerous commands default to the
    full command, and MCP rows toggle between tool and server scope.
  - "Reject" opens a free-text reason for the model.
  - Kilo scales the options to the risk: allow once and reject for
    config-protected files; allow and reject, never saved, for sandbox
    escalation.
  - Kilo records why a call was allowed or denied (agent, global, project,
    session, manual, default, and the winning rule), and shows it as a
    muted badge with words.
  - Kilo's interrupt key rejects at once without opening the feedback
    stage, a separate path from cancelling feedback.
  - It goes full screen at 80×24.
- **Keystrokes cannot land on Approve.** codex holds an approval prompt
  until the composer has been idle for 1 s
  (`APPROVAL_PROMPT_TYPING_IDLE_DELAY`). (codex
  `tui/src/bottom_pane/mod.rs:226,700-800`)
- **Commands are shown tamper-evident.** Kilo escapes C0 and C1 controls,
  DEL, U+200E/F, U+2028/9, U+202A–E and U+2066–9 as `\xNN` or `\uNNNN`,
  so a command cannot repaint the terminal or reorder itself
  (Trojan Source). It shows skill commands word for word, never as
  decomposed patterns, and rewrites IDN hostnames to punycode. (kilo
  `opencode/src/kilocode/skills/display.ts:6-35`,
  `opencode/src/kilocode/util/url.ts:1-38`) Grok's URL consent flags
  `xn--` hosts, and disables Accept for bad syntax, a non-http(s) scheme or
  embedded credentials. (grok `xai-grok-pager/src/views/elicitation_view/state.rs:18-163`)
  **For 0009:** `safetext.Command(s)` beside the sanitizer.
- **Prompts are deferred while streaming, in FIFO order.** codex queues
  approvals, elicitations and item start or end events while a stream is
  active. Once anything is queued, everything after it queues too, so an
  end never arrives before its begin. A request answered on another
  surface is removed without an abort. (codex
  `tui/src/chatwidget/interrupts.rs:1-80`)
- **Questions and forms.** Several questions become tabs plus a confirm
  tab. Digits select directly, and a "type your own" row is on by default.
  (opencode `tui/src/routes/session/question.tsx:22-283`) Grok builds MCP
  elicitation forms from a schema, with one typed value per field and
  errors per field and per form. (grok `xai-grok-pager/src/views/elicitation_view/state.rs`)
  0006's JSON Schema arguments can drive the same form.
- **Tool-call rows have a grammar** (opencode
  `tui/src/routes/session/index.tsx:1836-2211`; codex
  `tui/src/exec_cell/render.rs:118-800`):
  - a 2-cell icon per tool kind, then the title;
  - running, waiting on permission, done, failed and denied are distinct
    states. opencode shows denied by strikethrough, detected by matching
    error strings, which is fragile; use a typed state;
  - codex's status title (`Ran`, `Running`, `Failed (exit N)`) carries the
    meaning, so the coloured `•` is not the only signal;
  - block output collapses to N lines with an expand control.
  - Strikethrough (SGR 9) is unsupported in conhost and Apple Terminal, so
    pair it with a glyph.
- **Runs of tool calls fold under one label.** "Read 3 files, Searched 2
  patterns · 1 failed": present tense while any member runs, correct
  plurals, and a text suffix for failures. Each entry is a member, a
  finished thought that folds to nothing, a transparent member the user
  opened, or a break. (grok `xai-grok-pager/src/scrollback/state/verb_group.rs:1-499`)
- **An ACP update tracker** folds the update stream into transcript
  changes. An update that arrives before its tool call is stashed and
  merged when the call arrives. Suppressed calls still show if they fail.
  Headless reducers turn the same stream into NDJSON. (grok
  `xai-grok-pager/src/acp/tracker.rs:360-981`) Kept free of rendering, the
  CLI surface of 0006 can share its event types.
- **Attention is a policy over events** (opencode
  `tui/src/feature-plugins/system/notifications.ts:9-87`; kilo
  `tui/src/feature-plugins/system/notifications.ts:59-82`):
  - each question and permission ID fires once;
  - opencode notifies "Session done" on a busy-to-idle change when the
    run did not error;
  - Kilo notifies only when a root turn closes as `completed`. It skips
    errors, user aborts and `superseded`, a deliberate hand-off to a queued
    follow-up;
  - subagents play a sound but post no desktop notification.
- **Turn outcomes.** Kilo maps a close reason and the last finish to
  interrupted, error, limit, filtered or unexpected, each with a tone, and
  attaches the count of unfinished todos, so the banner says "stopped with N
  tasks left". (kilo `kilo-vscode/webview-ui/src/context/session-outcome.ts:1-64`)
- **Status and timing** (codex `tui/src/status_indicator_widget.rs:1-330`,
  `tui/src/history_cell/hook_cell.rs:33-60`):
  - a hook run is shown only after 300 ms, and stays at least 600 ms;
  - the elapsed timer pauses while an approval is open;
  - `(12s • esc to interrupt)` sits at a fixed position;
  - separator dates are fixed when the cell is built, so midnight cannot
    change a cached height.
- **Usage and context.** Kilo's context bar has three segments: used, which
  turns hot at 50%; reserved for output; and available. Usage groups by
  provider and model, with cost to 2–6 fraction digits and `<$0.000001` for
  anything smaller. codex shows "N% context left", or "Xk used" when the
  limit is unknown. (kilo `kilo-vscode/webview-ui/src/components/chat/ContextProgress.tsx:1-96`)
  Without colour the segments need distinct glyphs, and "hot" needs text.
- **Smaller widgets:**
  - **Kilo's activity timeline:** one bar per part, coloured by kind,
    following the newest bar only while the user is pinned to the right
    edge. In cells: `▁…█` with a letter per kind.
  - **Kilo's suggestion bar:** `→ text [Primary] [Secondary]`, keys 1 to 9
    when it blocks.
  - **Kilo's network wait:** "Waiting for network… Esc to stop", then
    "Retrying in Ns. Enter to resume now".
  - **Grok's dashboard:** rows ranked needs-input, then working, then idle;
    state by glyph shape and colour; old idle rows folded into "N more"; and
    a peek where 1 to 9 answer a pending permission. (grok
    `xai-grok-pager/src/views/dashboard/{state.rs:141-180,row.rs:19-95}`)
  - **Kilo's agent board:** `type · From → To` route headers, and "Stored
    only. Delivery and reading are not confirmed." (kilo
    `tui/src/kilocode/board-tool.tsx:28-114`)
- **Embedded terminal sessions** (grok `xai-grok-shell-terminal/src/pty_session.rs:1-790`,
  `xai-tty-utils/src/lib.rs:85-998`):
  - a 256 KB output ring with an offset that only grows; attaching
    replays the ring, marked as a replay;
  - busy detection compares `tcgetpgrp` with the shell's PID every 500
    ms, because a busy process can be silent; ConPTY has no equivalent;
  - detached children get `GIT_TERMINAL_PROMPT=0`, an empty `GPG_TTY` and
    a fixed pager, so git, ssh and pinentry never write to `/dev/tty` over
    the TUI.
  - A pane needs a VT emulator and a PTY module, each needing a record.
- **One engine, several front ends.** Kilo's CLI owns sessions and serves
  HTTP and SSE. The TUI runs it in-process, and VS Code shares one
  connection across views, filtering events per view. Every Windows spawn
  hides its window. JetBrains builds its component tree once and mutates it
  through `update(model)`. (kilo `kilo-vscode/AGENTS.md`, `kilo-jetbrains/AGENTS.md`)
  **For go-tui-lib:** agent widgets consume a transport-neutral session
  model and emit replies as messages. They never call a backend.

### 8.13 Terminal services (extends 0005)

- **Clipboard copies report whether they landed.**
  - codex: `CopyStatus{Confirmed, Unconfirmed, Pending, Busy}`, because
    terminal writes are never acknowledged ("Copy unconfirmed; /export saves
    chat"). Grok: `ClipboardDelivery{Confirmed, Unverified, Failed}`, each
    with its own toast.
  - Inside tmux, codex uses `tmux load-buffer -w -t <most recent client> -`
    rather than DCS passthrough. Raw payloads are capped at 100 KB.
  - On Linux, a lease keeps X11 or Wayland ownership alive, and the native
    write goes first to avoid a `SelectionClear` race. PRIMARY is used only
    on local X11.
  - Grok copies to every route at once: native, the tmux buffer and OSC
    52.
  - (codex `tui/src/clipboard_copy.rs:1-370`, `tui/src/clipboard_copy/tmux.rs:1-80`;
    grok `xai-grok-pager-render/src/clipboard/mod.rs:1-470`)
- **Reading images from the clipboard.** opencode tries macOS `osascript`
  (PNGf), PowerShell `Clipboard::GetImage()` on Windows and WSL, then
  `wl-paste` and `xclip`, then text. OSC 52 read is usually disabled, so
  images need native tools. (opencode `tui/src/clipboard.ts:30-125`) The
  route choice is a pure function, `CopyCommand(goos, wayland, have)`; the
  caller runs the process.
- **`grok wrap` brings OSC 52 and appearance to remote shells.** It runs a
  command in a local PTY. A streaming output filter intercepts OSC 52, plain
  or inside tmux DCS passthrough, and writes it to the local clipboard. That
  works even in terminals with no OSC 52 support, such as Apple Terminal. A
  private OSC lets a remote session request the host's clipboard image.
  (grok `xai-grok-pager/src/wrap_filter.rs:1-60`, `pty_wrap.rs:1-215`)
  **For go-tui-lib:** an `io.Writer` filter, `osc52.Filter{OnCopy}`,
  reusable by an embedded pane.
- **How links display depends on the terminal.**
  - codex shows the label only, as an OSC 8 link, on Ghostty, iTerm2,
    WezTerm, Kitty, VS Code, Alacritty, Windows Terminal, Konsole, GNOME
    Terminal and VTE.
  - It shows `label (url)` on Apple Terminal, Warp, unknown terminals, any
    multiplexer, or when stdout is not a TTY.
  - Wrapping keeps URL tokens whole, so terminals that detect links
    themselves still see one token.
  - (codex `tui/src/markdown_render/web_links.rs:59-98`, `tui/src/wrapping.rs:434-760`)
  - Grok includes link IDs in the frame diff, so a retargeted link repaints
    when its glyphs did not change. It re-issues IDs so the fragments of one
    wrapped link share one, and needs tmux 3.4 for hyperlinks. (grok
    `xai-ratatui-inline/src/terminal.rs:19-436`)
  - Kilo opens a clicked link only with no movement, no double click, no
    active selection, and after a 250 ms delay. Only http and https open.
    (kilo `tui/src/kilocode/link-interactions.ts:7-153`) The host opens it,
    and the library emits an `OpenURLMsg`.
- **Notifications by brand, gated on focus** (grok
  `xai-grok-pager/src/notifications/{protocol.rs:7-91,focus.rs:8-57}`;
  opencode `tui/src/attention.ts:60-220`):
  - OSC 9 for iTerm2, WezTerm and Warp; OSC 99 for kitty; OSC 777 for
    Ghostty, VTE and foot; BEL for Zellij and the rest;
  - focus is tri-state, and opencode returns `focus_unknown` rather than
    assume;
  - the result names why it was skipped: disabled, empty, focused,
    blurred, focus unknown, renderer destroyed;
  - text is cleaned (ANSI stripped, controls removed) and cut to 80 cells
    for the title and 240 for the body;
  - codex turns focus reporting off on Windows, so there the state stays
    unknown and an "only when unfocused" policy never fires;
  - opencode's sounds have semantic names (question, permission, error,
    done, subagent done) and packs, and fail silently. BEL is the only
    portable cue.
- **Progress.** OSC 9;4 only on Ghostty, WezTerm and iTerm2 3.6 or later,
  because older iTerm2 shows it as alert text. It is always cleared at
  teardown. (grok `xai-grok-pager/src/notifications/progress.rs:10-39`)
- **The title** (codex `tui/src/terminal_title.rs:29-185`; kilo
  `opencode/src/kilocode/cli/cmd/tui/terminal-title.ts:28-136`; grok
  `xai-grok-pager/src/notifications/title.rs:8-97`):
  - codex builds it from item IDs, ends it with BEL rather than ST,
    removes controls and bidi codepoints, caps it at 240 characters, and
    writes nothing when stdout is not a TTY;
  - Kilo prefixes a state (working, attention, finished) with an icon style
    of none, unicode or emoji. It defaults to none, because emoji titles
    render badly in conhost and some tmux status lines;
  - Grok skips unchanged writes and holds spinner frames about 264 ms,
    because Ghostty debounces title updates.
- **An activity beacon for embedders.** With `KILO_TERMINAL_ACTIVITY=1`,
  Kilo writes `OSC 777;kilo;activity;1;<state>;<unix-ms>`, where the state
  is idle, busy, retry, waiting, error or done, on change and every 5 s. Its
  VS Code host rejects the wrong version, rejects timestamps more than 5 s
  ahead or 15 s old, and expires to idle after 15 s, so a dead TUI never
  leaves a stale "busy". (kilo
  `opencode/src/kilocode/cli/cmd/tui/terminal-activity.ts:15-85`) A vendor
  tag and a version avoid clashing with urxvt's OSC 777.
- **Pointer shape (OSC 22).** A hand over links only on Ghostty and Kitty
  with no multiplexer, reset on Kitty with an empty OSC 22. (codex
  `tui/src/tui/link_pointer.rs:43-105`)
- **Copy on select, by terminal.** codex's `Auto` turns it off where the
  terminal forwards its own copy shortcut: Ghostty 1.2 and later, Kitty on
  macOS, Windows Terminal, and VS Code on Windows. It stays on for
  multiplexers and unknown terminals. Right-click paste is re-implemented,
  because capture removes the terminal's own, and is off over SSH and in VS
  Code. (codex `tui/src/local_settings.rs:143-185`)
- **Selection.** Click count picks character, word or line, within 400 ms
  on one cell. Dragging back keeps the whole word or line, and the copy key
  is Ctrl+C, Super+C or Ctrl+Shift+C. (codex `tui/src/text_selection.rs:12-76`)
  A click is not a click while text is selected, and a drag-select never
  dismisses a dialog. (opencode `tui/src/ui/dialog.tsx:21-221`)

### 8.14 Extensibility: plugins, slots and routes (new)

opencode's TUI is now built from plugins. Its built-in sidebar sections and
the diff viewer are plugins too.

- **Every registration goes through a scope.** Each plugin gets an abort
  signal, an `onDispose` list and a `track` helper. Routes, slots, event
  handlers, keymap layers, modes and sound packs all register through it.
  Disposal aborts the signal, then runs cleanups in reverse order under a
  shared 5 s budget, classifying each as ok, error or timeout. A plugin
  whose init throws is disposed at once. (opencode
  `opencode/src/plugin/tui/runtime.ts:79-651`)
  **For go-tui-lib:** a `plugin.Scope{Context(); OnDispose(); Track()}`, and
  every `Register*` in `workspace`, 0006 and 0007 returns an unregister
  function, so one teardown path covers everything.
- **Named, typed slots with composition modes.** The host declares slots
  (`home_logo`, `session_prompt_right`, `sidebar_content`, `sidebar_footer`
  and others), each with typed props. A plugin registers an order and a
  render function per slot. The modes are append (sorted by order),
  `replace` (one contribution replaces the host's children), and
  `single_winner`. A render failure is reported per plugin and slot.
  (opencode `plugin/src/tui.ts:455-517`, `tui/src/plugin/slots.tsx:25-64`,
  `tui/src/routes/home.tsx:76-91`)
  **For `workspace`:** a `Slot(name, mode, fallback)` that recovers a
  contribution's panic and renders an error cell.
- **Routes with shadowing.** A registration pushes onto a per-name stack,
  so the latest wins and unregistering restores the previous one. An unknown
  route renders "Unknown plugin route" with a way home. A route can carry a
  return route for Back. (opencode `tui/src/context/route.tsx:6-60`,
  `tui/src/component/plugin-route-missing.tsx:3-13`)
- **Status and compatibility.** Each plugin reports
  `{id, source: file|npm|internal, enabled, active}`, and a built-in can
  ship disabled. A deprecated v1 API is kept through a shim that warns once
  per name. (opencode `tui/src/plugin/command-shim.ts:7-108`) The pattern
  suits this module's own API evolution, next to `//go:fix inline`.
- **A crash screen that does not depend on the theme.** A render error
  shows a screen with a hard-coded palette per mode, since the theme may be
  what broke. It has a scrollable trace and Copy report, Restart and Quit
  actions. (opencode `tui/src/component/error-component.tsx:10-231`)
  **For `workspace`:** `Recover(func(any) tea.Model)` with a default panic
  view that has an ASCII and no-colour form.
- **An IDE bridge.** opencode scans `<home>/.claude/ide/<port>.lock` files,
  scores each by the deepest workspace folder containing the working
  directory, and speaks JSON-RPC 2.0 over a loopback websocket for selection
  and @-mention events. Zed is read from its workspace database. (opencode
  `tui/src/context/editor.ts:9-259`, `tui/src/editor-zed.ts:40-120`) The
  discovery scoring is pure. A websocket needs a record.

### 8.15 Config, state and persistence (new)

- **Settings as metadata, each value with its source.** Grok's registry
  gives each setting a kind (bool, enum with live preview and revert, int
  range, group). Every value resolves to `{value, source}` across nine
  layers (MDM, CLI, environment, user, remote and others), and the UI
  refuses an edit a higher layer overrides. (grok
  `xai-grok-config-types/src/registry.rs:1-467`, `xai-grok-config/src/resolved.rs:8-27`)
- **Edits keep the user's formatting.** Grok writes with `toml_edit` and
  never overwrites a file it cannot parse. Kilo edits `tui.jsonc` with
  `jsonc-parser` so comments survive, and coalesces reloads to one in
  flight plus one trailing. (kilo `opencode/src/kilocode/tui/config.ts:18-149`,
  `opencode/src/kilocode/cli/cmd/tui/context/tui-config-hot-reload.ts:20-49`)
  A comment-preserving JSONC writer needs a module and a record, or a small
  editor under `internal/`.
- **A JSON Schema with a drift test.** codex generates draft-07 with
  `deny_unknown_fields` and sorted keys, and a fixture test fails with "Run
  `just write-config-schema`". Audio settings ignore project layers, and
  screen ownership is fixed at launch. (codex `config-schema/src/main.rs:1-22`,
  `config/src/schema.rs:1-295`) opencode warns about unknown keys.
  **For go-tui-lib:** generate with `go generate`, and keep a golden drift
  test.
- **Display density is one vocabulary.** Kilo's `reasoning_display` is
  expanded, preview or headline, in the shared config schema, so the TUI,
  VS Code and JetBrains read the same key. A work-style choice at onboarding
  (human in the loop or autonomous) sets the permission policy and the
  densities together. (kilo `core/src/v1/config/config.ts:124-133`,
  `kilo-vscode/src/shared/work-style-presets.ts:1-80`) Each agent widget
  takes a `Density`.
- **Program-running keys come only from the user's own config.** Grok's
  status-line command cannot be set by repository or remote config. (grok
  `xai-grok-pager/src/app/status_line/command.rs:1-308`)
- **UI state.** opencode keeps `kv.json` under a cross-process lock, writes
  it atomically (temp file, then rename), queues writes so rapid sets land
  in order, and migrates legacy keys on read. (opencode
  `tui/src/context/kv.tsx:10-65`, `tui/src/util/persistence.ts:22-33`)
  Rename-over fails on Windows while the target is open.
- **An external status line** (grok `xai-grok-pager/docs/user-guide/25-status-line.md:9-142`,
  `xai-grok-status-line/src/context.rs:1-53`,
  `xai-grok-pager/src/views/status_line/sanitize.rs:13-247`):
  - modes `builtin` (ordered items, each with its own elision width),
    `command`, and `disabled`;
  - a command gets a versioned JSON payload on stdin, where unknown values
    are omitted and the version changes only when a field is removed or
    retyped;
  - runs are debounced at 300 ms, and 100 ms on resize; a running one is
    never cancelled; an error paints only after three failures;
  - the child gets `COLUMNS` and `LINES`, an empty `BASH_ENV`, its own
    process group, a 10 s timeout and a 64 KiB cap;
  - output keeps SGR only, swallows DCS, SOS, PM, APC and two-character
    escapes (so `ESC ( B` from `tput sgr0` does not paint `(B`), and keeps
    OSC 8 only for http(s) or mailto.
  - codex lets the user choose and order kebab-case status items in a
    picker with live preview. (codex `tui/src/bottom_pane/status_line_setup.rs:1-96`)
  - **For go-tui-lib:** a `statusline` package with a scheduler and a
    runner that the caller supplies, and a 0009 sanitiser profile
    `Policy{AllowSGR bool; OSC8Schemes []string}`.
- **Footers collapse by width.** `first_fitting_line(candidates, width)`
  picks the first variant that fits whole, so a shortcut is never shown
  half cut, and a key is never split from its label. (codex
  `tui/src/footer_hint.rs:1-61`) Grok's `cascade_truncate` drops the
  description, then the meta, then truncates the activity, then the type.
  (grok `xai-grok-pager-render/src/render/line_utils.rs:48-210`)

### 8.16 Diagnostics and testing (new: item 10 of §7)

- **A doctor model.** Grok's report holds facts, findings with stable
  dotted IDs (`terminal.newline-fallback`, `clipboard.delivery-unverified`,
  `terminal.tmux-truecolor`), and three-state probe results: available, no
  reply, unavailable. A finding is an issue or a recommendation, with a
  manual or an automatic fix. `grok doctor` uses an environment-only
  context so a wedged tmux cannot block it, and its JSON carries a schema
  version. It separates "Grok emits TrueColor" from "tmux forwards RGB".
  (grok `xai-grok-pager/src/diagnostics/{model.rs:7-235,probes/mod.rs:9-69}`)
  **For 0005's doctor:** the finding IDs reuse §8.2's skip-reason tokens.
- **A PTY harness with scripted scenarios** (grok
  `xai-grok-pager-pty-harness/src/{lib.rs:1-176,scripted.rs:88-370,timing.rs:1-80}`,
  `ptyctl/src/wait.rs:1-100`):
  - It runs the binary in a PTY over an emulator grid, with a mock
    inference server.
  - Steps include `wait_for_text`, `keys`, `paste`, `click_text`,
    `select_text_range`, `scroll_at`, `resize`, `assert_osc52_contains`,
    `assert_kitty_graphics` and `screenshot` (text, HTML, SVG and JSON).
  - The emulator's own replies to probes are off by default, so probe
    tests script them.
  - Frame boundaries come from the `?2026` markers. Benchmarks report p50,
    p99, max and jank, and CI fails when p99 grows 15% over a per-platform
    baseline. `idle_cost` asserts zero frames over 3 s idle.
  - Waits include "stable for N ms", and re-check only when output
    arrives. A timeout returns the screen, the cursor, the modes and the
    last 2 KB of output.
- **Probe responders.** codex's PTY tests answer `CSI 6n`, `CSI ?u` and
  OSC 10 and 11 as a terminal would. One test checks that regaining focus
  does not re-query colours. A per-thread test palette lets theme tests run
  in parallel. (codex `tui/tests/suite/focus_palette.rs:389-520`,
  `tui/src/terminal_palette.rs:97-128`)
- **A hermetic environment.** Kilo's PTY smoke test passes through only
  `PATH`, `SystemRoot`, `ComSpec`, `LANG` and `LC_*`, and points `HOME`,
  `USERPROFILE`, `APPDATA`, `LOCALAPPDATA`, `XDG_*` and `TMP*` into a
  temporary directory. Grok strips every brand, multiplexer and editor
  marker, plus `COLORFGBG`. (kilo `opencode/src/kilocode/cli/cmd/pty-smoke.ts:12-125`)
- **Matrix invariants with expected failures.** Grok replays timed wheel
  gestures per terminal profile and checks named invariants against the
  recorder's own clock, so a loaded CI host cannot fail it. Known bugs are
  expected failures, and one that starts passing fails the run. (grok
  `xai-grok-pager-pty-harness/src/scroll_matrix/invariants.rs:1-60`)
- **What the sources lack.** No project tests across colour × charset ×
  width as rule 6 does. opencode snapshots characters at one width, and
  grok has 12 text-only snapshots, so style regressions are invisible.
- **For go-tui-lib:** `tuitest/vt` on an emulator, `tuitest.Responder`,
  `tuitest.HermeticEnv`, scripted waits including `Stable(d)`, and an
  idle-frame assertion. `charmbracelet/x/vt` (§4.2) is the candidate
  emulator, and needs a record.

### 8.17 Motion, frames, i18n and accessibility

- **Tick demand.** Grok's views report `TickDemand{None, Slow (83 ms), Fast}`,
  and the maximum wins. `None` parks the loop with no wake-ups. A tick
  redraws only when an animation that was actually painted changes on that
  tick. A block waiting on the user freezes its pulsing rail solid, "paused
  on you". (grok `xai-grok-pager/src/app/app_view.rs:245-258`,
  `xai-grok-pager/src/views/dashboard/animation.rs:1-65`) **For 0009:** the frame scheduler
  takes a combined demand and returns a nil command for `None`.
- **Back-pressure from the writer.** Grok's frames go to a writer thread
  with `queued` and `written` counters. It draws only when everything
  queued is written, so draws during a busy terminal collapse into one.
  After 5 s with no progress it logs a stall once. Input and timers keep
  running when the terminal stops reading. (grok
  `xai-grok-pager-render/src/render/draw.rs:55-185`) Under rule 1 this
  is an `io.Writer` the caller passes to `tea.WithOutput`.
- **A coalescer with barriers.** Kilo's frame queue batches mergeable items
  (deltas) once per frame. Any other message first flushes the queue, so
  lifecycle events keep their order relative to deltas. A 100 ms timer
  covers stalled frames. (kilo `kilo-vscode/webview-ui/src/context/frame-queue.ts:3-76`)
- **Cheap first paint, upgraded later.** Grok's highlight and Mermaid
  workers coalesce jobs by entry ID, stamp results with a job ID and theme,
  and drop stale ones. (grok `xai-grok-pager/src/app/edit_highlight_worker.rs:1-75`)
  **For go-tui-lib:** a generic `Upgrader[K, R]` that returns a `tea.Cmd`.
- **Reduced motion, at three levels.** codex reads the OS preference: macOS
  `accessibilityDisplayShouldReduceMotion`, Windows
  `SPI_GETCLIENTAREAANIMATION`, and the XDG portal's `reduced-motion` with
  a 250 ms cap. Below a master switch, each effect has its own switch, and
  each caller names its reduced fallback (hidden, or a static bullet).
  Shimmer steps through DIM, normal and BOLD without TrueColor, and is
  static on an unknown palette. (codex `tui/src/system_motion.rs:1-83`,
  `tui/src/motion.rs:1-118`, `tui/src/shimmer.rs:1-80`) Grok has no
  reduced-motion setting. opencode and Kilo have one switch that shows a
  static `⋯`.
- **Spinner width is the widest frame, in cells.** Kilo measures every
  frame with the renderer's width method, so the layout never jitters.
  (kilo `tui/src/component/register-spinner.ts:1-60`)
- **Locale.** codex picks a 12- or 24-hour clock from `LC_ALL`, `LC_TIME`
  and `LANG`, where `C` and `POSIX` mean 24-hour. (codex
  `tui/src/clock_format.rs:1-152`) No TUI here is translated. Kilo's VS Code
  webview has 21 locales, dictionaries layered with the last winning, locale
  normalisation (`zh-Hant` to `zht`, `nb` and `nn` to `no`), RTL for `ar`
  and `fa`, and plurals through `Intl.PluralRules`. (kilo
  `kilo-vscode/webview-ui/src/context/language-utils.ts:1-100`)
- **Bidirectional text is off by default** in grok, because many terminals
  already reorder RTL text. Chrome stays in logical order, and mappings
  between painted and logical columns keep selection correct. (grok
  `xai-grok-pager-render/src/render/bidi.rs:1-80`) It needs `x/text`, and
  so a record.
- **Telemetry.** Kilo enables telemetry only for `KILO_TELEMETRY_LEVEL=all`,
  and none of the four honours `DO_NOT_TRACK`. This library has no
  telemetry, and records nothing here.

## 9. Per-terminal quirks the sources work around

What the sources assert and code around, gathered by terminal. Nothing here
was observed directly (see "The second pass").

| Terminal | Quirk | Source |
| :--- | :--- | :--- |
| Windows Terminal | Partial scroll regions discard rows; use full-screen insertion | codex |
| Windows Terminal | Default-terminal hand-off omits `WT_SESSION` | grok |
| Windows Terminal < 1.25 | An image-only clipboard arrives as an empty paste | opencode |
| ConPTY | CR-only line ends in paste; strips APC, so no Kitty images; no XTVERSION or DA2; corrupts X10 mouse at column 95 and beyond | opencode, grok |
| ConHost (legacy) | Raster fonts with no fallback: needs a CP437 or ASCII tier; 24-bit colour since Windows 10 1709 but not advertised | grok |
| Windows console | `ENABLE_PROCESSED_INPUT` turns Ctrl+C into a signal; modes outlive the process; legacy mouse arrives as key records without SGR; focus reporting off | opencode, codex, grok |
| mintty, MSYS2, Git Bash | Kitty keyboard off | Kilo |
| WSL with VS Code | Keyboard enhancement breaks dead keys | codex |
| Apple Terminal | No Ctrl+Enter; no OSC 52; no OSC 8 ("hostile parser"); needs `CSI 2J` with `CSI H`; RIS keeps scrollback; DA1 `1;2` with DA2 `1;95;0` over SSH | codex, grok |
| iTerm2 | Leaks release events for consumed shortcuts; OSC 9;4 from 3.6 only; Kitty images from 3.6; RIS keeps scrollback | codex, grok |
| Ghostty | Leaks release events; debounces title updates; copy-on-select off from 1.2 | codex, grok |
| Alacritty ≤ 0.14 | Duplicate legacy release with event types on | grok |
| kitty | Cmd arrives as Super, also over SSH | codex |
| VTE < 8200 | Shift+Enter arrives as a bare CR | grok |
| VS Code, Cursor, Windsurf, Zed | Host takes Ctrl+Q and Ctrl+I; xterm.js discards rows on `CSI S`; Shift+Enter arrives as CR | grok, codex |
| JetBrains | Paints probe queries as text; reads as Apple Terminal unless checked first | grok |
| Warp | Ignores Kitty placement replace; no OSC 8 | grok, codex |
| QTermWidget | `CSI S` discards departing rows | codex |
| tmux | Drops resize notifications; `extended-keys-format` decides kitty and Shift+Enter; hyperlinks from 3.4; passthrough needs 3.3 and `allow-passthrough on`; strips `COLORTERM`; global environment is the first client's | codex, grok |
| Zellij | Runs inline under Auto; wheel forced to 1 per notch; no images | grok, codex |
| SSH | Most brand markers lost; `LC_*` survives through `AcceptEnv`; `TERM_FEATURES` does not travel | grok |

## 10. What not to copy

- **Width by code unit or byte.** opencode truncates by UTF-16 `.length`;
  grok's `shorten_path` measures bytes and its diff wrap counts chars; Kilo
  hand-codes an East Asian width table. Use `x/ansi`.
- **A library writing to stdout.** opencode writes OSC 52 straight to
  `process.stdout`, and Kilo writes its reset list to the stdout fd. Rule 1
  forbids both.
- **Glyphs with no ASCII twin.** opencode's `▼ ▶ ┃ ⋯ △ ✱ → ← ⚙ ◈ ✓`,
  grok's shortcut arrows, and Kilo's title icons, which go straight from
  Unicode to none.
- **Meaning in colour alone.** opencode's toast variant, grok's diffs with
  no sign column, and Kilo's timeline by tool class.
- **State from error strings.** opencode detects a denied tool call by
  matching error text.
- **`$EDITOR.split(" ")`.** It breaks quoted paths.
- **No accessibility modes.** Grok has no reduced-motion, high-contrast or
  screen-reader mode.

## 11. Candidates from the second pass

The owner has not chosen among these. §7's numbering is kept, and new items
continue it.

### 11.1 Amendments to proposed records

Records 0004 to 0009 are proposed, so amending them before implementation
costs only the edit.

| Record | What the second pass adds | §8 |
| :--- | :--- | :--- |
| 0004 | render-state revision invalidating caches; live mode lock; 2031 refresh feeding the live theme; contrast resolver behind it | 8.2, 8.7, 8.11 |
| 0005 | pure `FromEnv` with raw and refined brands; per-feature caps with skip-reason tokens; shared probe deadline, input replay, sole-reader rule, late-reply swallowing; appearance chain with `LC_*` forwarding; kitty flag policy and ledger; tmux queries returned as commands; clipboard delivery status and routes; link display per terminal; notification protocol by brand, tri-state focus and skip reasons; title sanitising; activity beacon; OSC 22; progress gating; Windows console helpers | 8.2–8.5, 8.13 |
| 0006 | effects as data; press-twice confirmation; schema-driven forms | 8.12, 8.15 |
| 0007 | `Pending` and `ActiveKeys` for which-key; defaults guarded by capability, with newline alternatives; legacy-encoding normalisation; key labels per OS; key-debug view; release bindings that degrade; `Shortcut(cmd)`; vim engine on a document interface | 8.5 |
| 0008 | Tab policy and stale splices; ghost text; trigger rules and line ranges; frecency; degrading providers; modality latch; paged pickers | 8.9 |
| 0009 | barrier coalescer; tick demand; writer back-pressure; side-tables across freezes; resumable fence highlighting; narrow tables; an escape interpreter for tool output; input-filter chain; typeahead and quarantine; paste normalisation; wheel profiles; command and status-line sanitiser profiles | 8.5, 8.10, 8.17 |

### 11.2 New package candidates

| # | Candidate | What it holds | Prior art |
| :--- | :--- | :--- | :--- |
| 11 | `termmode` | mode plans, restore bytes, mode ledger, DA1 fence, hand-off, suspend, Windows console helpers | grok, codex, Kilo, opencode |
| 12 | `inline` | scrollback commit without scroll regions, per-terminal strategy, reflow cap, committed frontier | grok, codex |
| 6 | `transcript`, `listpane` | cell contract, height index, anchors, layout cache, disclosure, search, follow and bookmarks, sticky headers, hydrate | codex, grok, Kilo, opencode |
| 13 | `composer`, `composer/edit` | elements, grouped undo, kill buffer, paste chips, history, stash, queue, shell mode, external editor | all four |
| 7 | `agentui` | permission flow, idle delay, deferred prompts, questions and forms, URL consent, tool rows, verb groups, bounded output, tracker, attention policy, turn outcomes, usage, dashboard | all four |
| 14 | `textfit`, `table`, `diffview`, `canvas` | source-mapped wrap with joiners, cascade truncation, path shortening, table fitting, diffs with signs, a junction-merging canvas | codex, grok |
| 9 | theme v2 | terminal-native default, polarity-first adaptation, palette-generated theme, token taxonomy, colour-blind preset, recede, legacy glyph tier | grok, opencode, codex, Kilo |
| 15 | `plugin` | scopes, slots, routes, status, warn-once shims, crash screen | opencode |
| 16 | `statusline`, `state` | external status line, settings with sources, atomic KV, JSONL stores | grok, codex, opencode |
| 10 | `tuitest/vt`, `tuitest/pty` | emulator, responders, hermetic environment, scripted waits, idle frames, matrix invariants | grok, codex, Kilo |
| 17 | `termimage`, `chart`, `termpane` | Kitty, iTerm2 and Sixel images; cell charts; embedded terminal sessions | grok, codex |
| 18 | `motion`, `i18n` | reduced motion by OS, per-effect switches, shimmer; catalogues, plurals, locale normalisation | codex, Kilo |

### 11.3 Modules these would need

Each needs its own record under AGENTS.md, Dependencies.

| Module | For | Today |
| :--- | :--- | :--- |
| `golang.org/x/sys` | Windows console modes, `tcgetpgrp`, `tcflush` | indirect, v0.47.0 |
| `github.com/charmbracelet/x/vt` | `tuitest/vt`, `termpane` | not required |
| a PTY module, with ConPTY | `tuitest/pty`, `termpane` | not required |
| a lexer such as `github.com/alecthomas/chroma` | `highlight` | not required |
| `golang.org/x/text` | plurals, bidi | not required |
| `golang.org/x/net/idna` | punycode for URL consent; a minimal encoder under `internal/` is the alternative | not required |
| a purego or cgo binding | macOS modifier probe and reduced-motion query | not required; CI is `CGO_ENABLED=0` |
| a websocket module | IDE bridge | not required |

### 11.4 Recommended

- **Amend 0005, 0007 and 0009 first,** while they are still proposed. Most
  of the second pass lands in them, and neither `termmode` nor `inline`
  works without 0005's facts.
- **Then write records for `termmode` (11) and `transcript` (6).** Every
  later widget depends on the mode plan and on the cell contract.
- **Then `composer` (13) and `agentui` (7),** which pi-go's agent session
  needs next, and theme v2 (9), which the widgets' diff and markdown tokens
  need.
- **Then the harness (10)** and its emulator record, before `inline` (12)
  and `termpane`, which cannot be tested without it.
