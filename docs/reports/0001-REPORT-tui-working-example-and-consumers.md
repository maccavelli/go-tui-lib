---
date: 2026-10-01
subject: the evidence behind go-tui-lib's scaffold, its TUI conventions, and its first packages
examines: "ocp-login at a0d1fc4 (by pattern only); the fleet's go.mod files; pi-go records 0004 and 0005; go-core-lib records 0004; the Go module proxy"
associated-madr: "0001-MADR-scaffold-charm-tui-library.md"
---
# TUI working example, fleet consumers and the planned packages

This report records findings. It decides nothing. The decisions it supports
are [0001-MADR-scaffold-charm-tui-library.md](../decisions/0001-MADR-scaffold-charm-tui-library.md)
and [0002-MADR-multi-pane-workspace-layouts.md](../decisions/0002-MADR-multi-pane-workspace-layouts.md).

## 1. The working example: ocp-login, by pattern

ocp-login is an org-internal program, and this repository is public. It is
therefore described by its design patterns only, never by its files, and no
code is taken from it (0001-MADR amendment A2). Each finding below was
confirmed against its source at commit `a0d1fc4` by a scripted check that
reports a verdict per claim and no path. All 21 claims were confirmed.

### Stack

- Charm v2 under `charm.land`: Bubble Tea, Lip Gloss and Bubbles. It also
  uses `colorprofile` and `x/ansi`, and fang for its command layer.
- A test refuses the Charm v1 import paths.
- Go 1.26.6.
- The terminal UI is one layer, which imports no domain package. Its
  coupling to the product is in names and strings: product names, an
  environment variable for the width cap, kube glyphs, menu naming, and
  parsing of its login tool's output.

### Patterns that generalise

| Pattern | What it does |
| :--- | :--- |
| Layered theme | A raw palette, then semantic roles (title, body, muted, rule, accent, success, warning, error), then styles built once |
| Three-state background | Light, dark or unknown. The unknown palette is legible on both, and the choice is made once per run |
| Colour-profile resolution | `NO_COLOR` or a non-TTY means no colour. Otherwise the profile is detected or forced, and every write goes through a profile-aware writer |
| Width-capped measurement | One measurement per frame, with a maximum and a minimum width. Wrap and truncate go through `x/ansi`, in cells |
| Glyph table | Every glyph has a Unicode form and an ASCII twin, each one cell wide, including the glyphs Charm components draw by default |
| Rendering primitives | Rules, sections, cards, status lines and key/value rows, used instead of raw prints |
| Help footer | Hints from key bindings. Whole hints are dropped when the width runs out, then an ellipsis is shown |
| Composed frame | A pinned band and a scrolling content area, composed on a Lip Gloss canvas so content is clipped to its area |
| Motion | Spinners and progress bars, shown only when there is data to show |
| Prompts | Text, secret and confirm prompts. The secret field is masked and wipeable, and paste and cancel keys are normalised |
| Non-interactive fallback | Every interactive path has a plain path for a non-TTY or a `--no-input` run |

### How it renders

- Short-lived programs render inline (its record `0004`).
- A long-lived session region uses the alternate screen, and replays its
  record to the main screen on exit (`0011`).
- Its update command takes no alternate screen (`0024`).
- Bubble Tea's own signal handler is off. The command owns `ctrl+c` and
  exits 130.

### How it tests

- Hand-written golden files over colour × charset × width, with an update
  flag.
- Guard tests: every view degrades to ASCII, no glyph literal outside the
  table, and contrast survives ANSI-256 conversion.
- A PTY harness that replays measured terminal profiles, on Unix and
  Windows.
- A C-locale test job in CI.

### Records that govern its TUI

Cited by number, as its decisions; their content is summarised in 0001-MADR
§6:

- `0002` (one layout per frame, primitives only);
- `0003` (Charm v2);
- `0004` (inline programs);
- `0005` (footer truncation);
- `0011` (a composed frame on the alternate screen);
- `0014` (a three-state background);
- `0024` (no alternate screen for updates);
- `0027` (colour depth);
- `0033` (every glyph from the table);
- `0037` (unknown-palette contrast).

## 2. The fleet's TUI programs

| Program | Charm | Go | Note |
| :--- | :--- | :--- | :--- |
| ocp-login, ocp-login-macos | v2 (`charm.land/bubbletea/v2` v2.0.9, `lipgloss/v2` v2.0.6, `bubbles/v2` v2.2.1) | 1.26.6 | the working example |
| mcp-server-magicdev, mcp-server-magictools, mcp-server-recall, mcp-server-socratic-thinker | v1 (`github.com/charmbracelet/bubbletea` v1.3.10, `lipgloss` v1.1.0) | 1.26.6 | must move to v2 before they can share code from here |
| pi-go | v2, planned | 1.27.1, planned | the first consumer of 0002; no `go.mod` or Go code yet |

Importing go-tui-lib requires Go 1.27.1 (0001-MADR §3). Importing go-core-lib
requires it already.

## 3. pi-go, the first consumer

From pi-go's records, both `proposed`:

- **`0005-MADR-v1-feature-scope.md`.** A Charm v2 TUI on a TTY, an ACP
  client, "extracting shared widgets into go-tui-lib as they settle". The
  1.0 TUI has:
  - a streamed Markdown transcript, collapsible tool cards with diffs, and
    a multi-line editor with history;
  - a permission dialog, model, thinking and mode pickers, and a session
    picker;
  - slash and `@path` completion, `!cmd`, Esc to interrupt, steer and
    follow-up keys, and Ctrl+G for the external editor;
  - a footer with model, mode, context % and cost.

  The same record plans token, duration and cost metrics, and `slog`
  rolling logs.
- **`0004-MADR-go-module-architecture.md`.**
  - The TUI is `internal/tui`, a Charm v2 program over `acpclient`.
  - Its import rule 5 lets only `internal/tui` import `charm.land/...`, and
    only `internal/cli` import Cobra, Viper or fang.
  - Rule 8 lets only `internal/cli` import go-core-lib.
  - Neither rule names go-tui-lib, so adopting it needs a pi-go
    amendment.
- **The owner's addition (2026-10-01).** The agent session goes in a main
  pane, session metrics in a left or right sidebar, and logs in a bottom
  pane. This is 0002-MADR.

## 4. `updatetea`, as go-core-lib plans it

go-core-lib
`docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md` §1 and §4
place a Bubble Tea adapter for `selfupdate` here, "Charm, in go-tui-lib
only". Its sketch:

```go
type EventMsg struct{ Event selfupdate.Event }
type ConfirmMsg struct{ Req *selfupdate.ConfirmNeeded }
type CredentialMsg struct{ Req *selfupdate.CredentialNeeded }
type DoneMsg struct {
    Result selfupdate.Result
    Err    error
}
type BannerMsg struct {
    Record selfupdate.CheckRecord
    Err    error
}

func Listen(s *selfupdate.Stream) tea.Cmd // self-reissuing
func Answer(r *selfupdate.ConfirmNeeded, ok bool) tea.Cmd
func Supply(r *selfupdate.CredentialNeeded, secret []byte) tea.Cmd
func CheckInBackground(ctx context.Context, c *selfupdate.Checker, r selfupdate.CheckRequest,
    store selfupdate.CheckStore, maxAge time.Duration) tea.Cmd

type Model struct{ /* progress bar, spinner, masked input, confirm */ }
func New(s *selfupdate.Stream, opts ...Option) Model
// Options: WithLabels, WithBudget, WithLinger, WithStyles, WithKeyMap.
```

The same record sets three rules for it:

- the model renders inline and never takes the alternate screen;
- it shows a spinner when the total is unknown, and a bar when it is known;
- `ctrl+c` calls `Stream.Cancel`, then waits for `DoneMsg`.

The record's Phase 2 Confirmation is owed here: an example Bubble Tea
program drives a real update against the H3 test server, and `ctrl+c`
leaves the target unchanged. `Stream`, `Start` and `PromptCredential`
shipped in go-core-lib `v1.2.0`. Under 0002-MADR, `updatetea`'s model can
also be a `workspace` pane or overlay.

## 5. The stack's current releases

On the module proxy, 2026-10-01:

- `charm.land/bubbletea/v2` v2.0.10, `charm.land/lipgloss/v2` v2.0.6,
  `charm.land/bubbles/v2` v2.2.1;
- `github.com/charmbracelet/colorprofile` v0.4.3,
  `github.com/charmbracelet/x/ansi` v0.11.8;
- golangci-lint v2.14.0, govulncheck v1.8.0.

## 6. Candidates for packages

Listed, not chosen. Each is its own record.

- **Foundations** (0002-MADR): `glyph`, `theme`, `tuitest`, `layout` and
  `workspace`.
- **Standard panes:** a log tail with levels and follow mode; a key/value
  metrics view; a scrolling text and Markdown view; a transcript of
  streamed blocks.
- **Overlays:** a dialog, a picker, a command palette and a toast.
- **Chrome:** a help footer with whole-hint truncation, and a status line.
- **Input:** prompts, a masked secret field, and a multi-line editor.
- **Motion:** a spinner and a progress bar under the motion rule above.
- **Adapters:** `updatetea`.
