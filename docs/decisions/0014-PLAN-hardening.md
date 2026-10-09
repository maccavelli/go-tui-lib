---
status: in-progress
date: 2026-10-09
associated-madr: "0014-MADR-native-integration-api.md"
---
# Implement W3: bounded input, one sanitizer, overflow-safe layout, no mutable package variables, wider fuzzing, and release `v0.8.0`

Associated MADR: [0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md),
workstream W3, as amended by A1. Evidence:
[0014-REPORT-api-assessment-and-integration-research.md](../reports/0014-REPORT-api-assessment-and-integration-research.md)
§2.5.

## Goal

Every input from a terminal, a file, an agent or a remote client is
bounded, sanitized where it is displayed, and fuzzed. In detail:

* **Untrusted strings** pass through one sanitizer before they reach the
  screen.
* **The workspace** bounds its view cache and its window size.
* **`LoadDir`, argument JSON and `$NAME` expansion** have limits.
* **`layout`'s arithmetic** cannot overflow.
* **No exported package variable** can change the library's behaviour.
* **`make fuzz`** finds every fuzz target by itself, and nine new targets
  cover the parsers and renderers. (Corrected 2026-10-09: ten, as Step 9's
  table and the Verification list them.)

The release is `v0.8.0`, together with W2.

## Scope

### Facts this PLAN starts from (2026-10-07)

Each probe below ran on a scratch copy and fails on today's code.

| # | Fact | Where it was read | Probe |
| :--- | :--- | :--- | :--- |
| H1 | `viewOf` caches every pane's view under `viewKey{kind, id, width, height, focused, method, themeGen}`; entries are deleted only by `forget` | `workspace/render.go:143-153`; `workspace/workspace.go:139`, `:169-176`, `:389`, `:395-397`; `workspace/overlay.go:105` | 726 entries after 500 resizes; 1326 after 200 more theme changes |
| H3 | `WindowSizeMsg` sets the size with no upper bound; the frame allocates width × height cells of 112 bytes; `clip` grows its buffer by `height * (width + 1)` | `workspace/workspace.go:409-411`; `internal/cells/cells.go:28-42`; `workspace/render.go:159` | a 2^30 × 2^30 message is accepted as-is |
| H4 | `LoadDir` walks with no count or depth limit, and reads each file whole | `command/loaddir.go:37`, `:49`, `:78` | an 8 MiB file, 5,000 files and 64 levels deep all load |
| H4+ | `expand` replaces every `$NAME` with its argument, with no cap on the result | `command/expand.go` | — |
| H5 | argument JSON is parsed whole, with no size check, from `CallMCP`, `Run` and slash lines | `command/decode.go:175-196`; `command/mcp.go:95`; `command/dispatch.go:124-130`; `command/slash.go:36,48` | `CallMCP` takes 16 MiB of arguments |
| H10 | `Ratio` computes `avail * N / D`; `share` divides by an unbounded sum of weights; `minSum` and `sum` add without bound | `layout/size.go:251-298`, `:323-329`, `:256-258`, `:398-404` | a 2^40/2^41 ratio of 16 Mi cells gives 0, not 8 Mi; two `Fill(2^40)` give 1 and 16,777,215 |
| H11 | `termcaptest.RunTimeout` is a package variable read by `Terminal.Run`; the seven `workspace.Key*` variables are read by `ContextKeys` and `WhenContext` | `termcap/termcaptest/termcaptest.go:167`, `:242-256`; `workspace/context.go:18-34`, `:36-70` | — |
| C6 | sanitizing lives in `termsvc.strip`, `clean` and `SanitizeTitle`, `osc99ID`, the pointer filter, and `command.isSpaceOrControl` | `termsvc/termsvc.go:87-117`; `termsvc/beacon.go:22-38`, `:151-156`; `termsvc/notify.go:278-286`; `command/registry.go:193` | — |
| C6 | `ParseActivity`'s vendor, `Caps.Terminal` from XTVERSION, and workspace titles and badges are not sanitized | `termsvc/beacon.go:117`; `termcap/prober.go:650`; `workspace/render.go:213-258` | an ESC and U+202E survive `ParseActivity`; a title with a newline breaks the border |
| C6 | `SanitizeTitle` keeps U+200B, U+2028, U+FEFF, U+2060, U+E0041 and U+00AD | `termsvc/beacon.go:22` | — |
| H9 | four fuzz targets exist; `make fuzz` names three packages by hand; CI uploads three corpus directories | `Makefile:94-97`; `.github/workflows/ci.yml:101-120`; `layout/layout_test.go:595`, `command/fuzz_test.go:11`, `command/frontmatter_test.go:40`, `when/fuzz_test.go:12` | a fuzz target on `clip` failed at once: `clip("\x1b", 3, 1)` gives a line 0 cells wide |
| — | `LoadDir(fsys, src)` and `NewTerminal(p)` have no options parameter; adding one would change their signatures, which `apidiff` reports as incompatible | `go doc` | — |

### Facts re-read before execution (2026-10-09)

W2 (0014-PLAN-component-native-forms) and the Go 1.27.2 floor
(0016-PLAN-go-1-27-2-for-go-2026-6604) landed after this PLAN was
written. The agent read every row above again on `e45a9f9`, and ran the
probes for H1, H3, C6 and H10 again on a scratch copy, with Go 1.27.2.
Every finding still holds. The table above keeps the 2026-10-07
readings; this one gives today's lines and what W2 added. The steps below
are annotated where it changes them.

| Fact | Today |
| :--- | :--- |
| H1 holds: two panes gave 310 cache entries after 500 resizes, and 710 after 200 more theme changes, where the bound would be 4 | `viewOf` at `workspace/render.go:143-153`; the map at `workspace/workspace.go:142` and `:317`; `forget` at `:452`, `:458` and `workspace/overlay.go:105` |
| H3 holds: a 2^30 × 2^30 `WindowSizeMsg` gives 1073741824 × 1073741824. W2's `WithSize(2^30, 2^30)` does the same | `workspace/workspace.go:472-474`; `WithSize` at `:248-250`; `internal/cells/cells.go:30`, `:38`; `clip`'s `Grow` at `workspace/render.go:160` |
| **New, from W2:** `RenderPlain(width)` lays the workspace out at `width` × the height with no cap | `workspace/plain.go:35-46` |
| H4 holds | `command/loaddir.go:37`, `:49`, `:78` |
| H5 holds | `prepare` at `command/decode.go:212`; `CallMCP` at `command/mcp.go:94`; `admit` at `command/dispatch.go:150`, its `prepare` at `:175`; `command/slash.go:48` |
| **New, from W2:** `ParseArgs` is a fourth path. It builds the arguments from the words and calls `prepare` itself, before `Run`'s `admit` | `command/cliargs.go:59` |
| H10 holds: `Ratio(2^40, 2^41)` of 16,777,216 cells gave 0, not 8,388,608; two `Fill(2^40)` gave 1 and 16,777,215 | `Percent` and `Ratio` at `layout/size.go:272-276`; `minSum` at `:255-260`; `share` at `:304`; `sum` at `:398` |
| H11 holds | `RunTimeout` at `termcap/termcaptest/termcaptest.go:170`, read at `:245` and `:256`; the seven `Key*` variables at `workspace/context.go:20-33` |
| C6 holds: `ParseActivity` kept `"ev\x1b[31m\u202eil"` as the vendor, and `SanitizeTitle` kept each of U+200B, U+2028, U+FEFF, U+2060, U+E0041 and U+00AD | `strip` and `clean` at `termsvc/termsvc.go:93`, `:117`; `SanitizeTitle` at `termsvc/beacon.go:27`; `ParseActivity` at `:97`; `Pointer` at `:146`; `osc99ID` at `termsvc/notify.go:338`; `LinkPolicy.Openable` at `termsvc/links.go:44`; `isSpaceOrControl` at `command/registry.go:225`; XTVERSION at `termcap/prober.go:650-654` |
| C6's title finding has a new shape: a title `"evil\nline"` no longer adds a line, since `clip` holds the frame to its height, but the top border stops after "evil" | `workspace/render.go:214` (`title`), `:238` (`renderBox`) |
| **New, from W2:** `RenderPlain`'s titles, through `plainTitle`, strip escapes and collapse whitespace, but kept U+202E in the probe. Its bodies, through `plainText`, strip escapes only | `workspace/plain.go:88-97`, `:99-108` |
| `clip("\x1b", 3, 1)` still gives a line 0 cells wide | `workspace/render.go:158` |
| H9: five fuzz targets now, in four packages' tests. W2 added `FuzzParseArgs`, which `make fuzz` already runs, since `command` is listed. The Makefile and CI still name three packages by hand | `layout/layout_test.go`, `command/fuzz_test.go` (two), `command/frontmatter_test.go`, `when/fuzz_test.go`; `Makefile:98-101`; `.github/workflows/ci.yml:105-124` |
| `launch.Decide` sets `Width` and `Height` from the terminal, or `COLUMNS` with no upper bound | `launch/decide.go:113`, `:191-193`, `:225-231`, `:242` |
| `LoadOption` is in `docs/glossary.md`, reserved by 0014-MADR W3 | `docs/glossary.md:44` |
| **The constants' home (owner's choice, 2026-10-09).** `launch` does not import `workspace`; it imports `command`, `glyph`, `termcap` and `internal/enum`. The owner chose, of three: a new `internal/limits` holds `MaxSide`, `MaxCells` and the clamp. `workspace` exports `MaxSide` and `MaxCells` as constants equal to them, and `launch` uses the internal package. The others: `layout` exports them, or `launch` imports `workspace` | `go list -deps ./launch` |
| The preconditions hold: 0014-PLAN-component-native-forms is `complete` (`e45a9f9`), and `scripts/apicheck.allow` lists nothing | the records |

### Preconditions

* **0014-PLAN-component-native-forms is complete.** This PLAN's `clip`
  and title changes touch the files W2's Step 7 touches, and its release
  carries W2.

### In scope

| Step | Delivers | Findings |
| :--- | :--- | :--- |
| 1 | approval | — |
| 2 | `internal/sanitize`, adopted by `termsvc`, `termcap`, `command` and `workspace` | C6 |
| 3 | the bounded view cache | H1 |
| 4 | `MaxCells`, `MaxSide` and `clampSize` | H3 |
| 5 | `LoadDirWith` and its limits; the expansion cap | H4 |
| 6 | `WithMaxArgBytes`; the argument limit on every path | H5 |
| 7 | overflow-safe `layout` arithmetic | H10 |
| 8 | per-terminal timeouts and key functions; the variables deprecated | H11 |
| 9 | fuzz discovery and nine new targets | H9 |
| 10 | the release `v0.8.0` | — |
| 11 | close-out | — |

### Out of scope

* **H2 (`recover` around handlers) and H6 (backend contexts):** done by
  W2.
* **H8 (the conformance list):** done by W0.
* **Removing H11's deprecated variables:** that happens in `v0.9.0`, by
  W4's PLAN (A1.5).

## Implementation Steps

### Rules

1. **A step starts when the previous one is committed.** The agent
   commits on `main` only when the owner asks in that turn, with `git
   commit --no-edit`. The owner pushes and tags.
2. **Checks.**
   * The Go checks of
     [0014-PLAN-component-native-forms.md](0014-PLAN-component-native-forms.md)
     Rule 2, with `make apicheck` showing additions only.
   * Each new limit is named in the release notes.
3. **Each finding's probe becomes a test** that fails before its fix. The
   execution record shows that failure output, from a scratch copy at the
   previous commit.
4. **Mutations** run on scratch copies.
5. **Anything unplanned stops the step.**

### Step 1: records

* **This PLAN, approved.**
* **Its row in `docs/README.md`.**

### Step 2: `internal/sanitize` (C6)

```go
package sanitize

// Line is one line of display text. It strips escape sequences (ansi.Strip),
// turns runs of \r \n \t and U+2028, U+2029 into one space, and drops C0, DEL,
// C1, invalid UTF-8, the Bidi_Control set, U+FEFF, U+2060-2064, U+00AD and
// tag characters (U+E0000-E007F). It keeps ZWJ, ZWNJ and variation selectors,
// which emoji need.
// (D2, 2026-10-09: it also drops U+200B.)
func Line(s string) string

// Styled is Line for styled text: it keeps each complete escape sequence and
// filters the text between them as Line does. (Added by D1, 2026-10-09.)
func Styled(s string) string

// Truncate cuts s to at most cells cells under m, adding nothing.
func Truncate(s string, cells int, m ansi.Method) string

// Token keeps only the runes allowed accepts.
func Token(s string, allowed func(rune) bool) string

// HasControl reports whether Line would change s, for validation.
func HasControl(s string) bool
```

**Adopted by:**

* **`termsvc`:**
  * `strip`, `clean` and `SanitizeTitle` call `Line`; `SanitizeTitle`
    keeps its rune cut;
  * `ParseActivity` sanitizes the vendor;
  * `osc99ID` and the pointer filter call `Token`;
  * `LinkPolicy.Openable` calls `HasControl`.
* **`termcap`:** the XTVERSION name (`prober.go:650`) and the
  environment's terminal name and version pass through `Line`. That also
  keeps W4's JSON v2 report from failing on invalid UTF-8.
* **`command`:** `isSpaceOrControl` uses `HasControl`, rejecting as
  today.
* **`workspace`:**
  * `title()` and `renderBox` flatten and truncate titles and badges with
    `Line` and `Truncate`;
  * `clip` passes each line through `Line` before truncating, which fixes
    the fuzz failure (D1, 2026-10-09: through `Styled`, which keeps the
    styling);
  * (added 2026-10-09) `plainTitle` uses `Line`, and `plainText` passes
    each line of a plain body through `Line`, so `RenderPlain` holds
    nothing `Line` drops. The `plain.*` goldens stay byte-identical.

**Tests:**

* `TestLine`, a table with every class above, kept and dropped;
* `TestTokenAndHasControl`;
* `TestActivityVendorSanitized`;
* `TestXTVersionSanitized`;
* `TestTitleWithLineBreakKeepsBorder`, a golden for a pane titled
  `"evil\nline"` with a badge `"b\r\x1b[2J‮"`;
* `TestClipSanitizes`.

The existing termsvc goldens and the notification byte tests must stay
byte-identical for clean inputs.

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S2-1 | `Line` keeps U+202E | `TestLine` |
| S2-2 | `Line` drops ZWJ | `TestLine` (an emoji family sequence) |
| S2-3 | `renderBox` uses the raw badge | `TestTitleWithLineBreakKeepsBorder` |

### Step 3: the bounded view cache (H1)

* **The map becomes `map[slotKey]cachedView`,** with
  `slotKey{kind, id, focused}` and
  `cachedView{width, height, method, themeGen, view}`.
* A hit needs every stored field to equal the request. A miss replaces
  the slot.
* `forget` deletes a pane's two slots.
* The cache is then at most 2 × (panes + overlays).

**Test:** `TestViewCacheBounded`.

1. 500 resizes, 200 theme changes and two method changes, rendering after
   each.
2. Then `len(cache) <= 2*(len(panes)+len(overlays))`.
3. The `Changer` hit path still returns the cached string when unchanged
   (`render.go:146-150`).

**Mutation S3-1:** a miss adds a slot instead of replacing it.
`TestViewCacheBounded`.

### Step 4: the window clamp (H3)

* **`internal/limits`** (added 2026-10-09, the owner's choice) holds the
  values and `Clamp(w, h)`. `workspace` exports them as below, equal to
  the internal ones.
* **`const MaxSide = 4096` and `const MaxCells = 1 << 19`** (524,288
  cells, about 56 MiB of frame). For scale, an 8K display with a 6 × 12
  pixel font is about 1280 × 360, or 460,800 cells.
* **`clampSize(w, h)`:**
  1. each side goes into `[0, MaxSide]`;
  2. then, while `w*h > MaxCells`, `h` becomes `MaxCells / w`.

  Width is kept, so lines stay whole, and the frame draws top-left.
* **Used by:** the `WindowSizeMsg` case, W2's `WithSize`, and `SizeMsg` to
  panes. Also W2's `RenderPlain`: its width goes into `[0, MaxSide]`,
  and its layout's height follows the same clamp (added 2026-10-09).
* **`launch.Decide`** caps `Width` and `Height` with the same constants,
  through `internal/limits` (an additive change to `launch`).
* **The documentation** of `Update` and the constants says so.

**Tests:**

* `TestWindowClamp`: 2^30 × 2^30, 4096 × 4096, and 100 × 0;
* `TestRenderAtClamp`: a render at the clamp completes, with the
  allocation measured by `testing.AllocsPerRun`, `t.Skip` under `-short`.

**Mutation S4-1:** no `MaxCells` step. `TestWindowClamp`.

### Step 5: `LoadDir`'s limits and the expansion cap (H4)

* **`LoadDirWith(fsys fs.FS, src Source, opts ...LoadOption) ([]Command,
  []error)`** is new. `LoadDir(fsys, src)` keeps its signature and calls
  `LoadDirWith` with the defaults. It is additive, so `apidiff` stays
  clean, and the defaults are a behaviour change, named in the release
  notes.
* **`LoadOption`** is opaque, under W0.4 (D3, 2026-10-09: the defaults are
  also exported as `DefaultMaxFileBytes`, `DefaultMaxFiles` and
  `DefaultMaxDepth`, and a value below 1 keeps the default), with:
  * `WithMaxFileBytes(n int64)`, default 256 KiB;
  * `WithMaxFiles(n int)`, default 1,000 `.md` files;
  * `WithMaxDepth(n int)`, default 8 directory levels.
* **Errors:** `ErrFileTooLarge`, `ErrTooManyFiles` and `ErrTooDeep`, each
  wrapped as `command: <path>: %w`.
* **How each limit is enforced:**
  * **Size:** `io.ReadAll(io.LimitReader(f, max+1))`, so the file's
    stated size is never trusted.
  * **Depth:** a directory past the limit gives `fs.SkipDir` and one
    error.
  * **Count:** at the limit, `fs.SkipAll` and one error.
* **The expansion cap.** `expand` stops at 1 MiB of output and returns an
  `*ArgError` naming the limit.

**Tests:** `TestLoadDirLimits`, with the three probes and the boundary
values (exactly at, and one over); `TestExpandCap`.

**Mutation S5-1:** size checked with `Stat`. A file whose `Stat` lies
(a test `fs.FS`) still passes the limit, and `TestLoadDirLimits` fails.

### Step 6: the argument limit (H5)

* **`WithMaxArgBytes(n int) RegistryOption`,** with
  `DefaultMaxArgBytes = 1 << 20`.
* **Checked in `admit`,** before `prepare` and whatever `e.args` is. The
  same cap applies to a slash line's `Raw`, and to W2's `ParseArgs`, on
  the total length of its words, before it builds or prepares anything
  (added 2026-10-09).
* **Over the limit:** `&ArgError{Reason: "arguments are N bytes, over M"}`,
  and the audit records `Args: nil`.
* **`FuzzPrepareMask`,** over a schema with a secret field, a slice of
  secrets and a nested object. Its invariants:
  * no panic;
  * a successful `prepare` gives valid JSON;
  * `mask` gives `"***"` at every secret path, or nil;
  * no secret of four or more bytes from the input appears in `mask`'s
    output.

**Tests:** `TestArgLimit` covers `CallMCP` at 16 MiB, `Run`, slash,
`ParseArgs` (added 2026-10-09), and exactly at the limit.

**Mutation S6-1:** the check runs after `prepare`. The 16 MiB case
allocates, and its time assertion of 100 ms fails.

### Step 7: overflow-safe layout (H10)

* **`mulDiv(a, b, d int) (q, r int)`,** built on `bits.Mul64` and
  `bits.Div64`. It cannot panic here, since every caller has `b <= d`.
  `Ratio`, `Percent` and `share` use it, and the remainder scaling
  `r*1000/d` uses it too.
* **`share`'s weight sum** saturates. On overflow, every weight is
  shifted right one bit until the sum fits, which keeps the proportions.
* **`minSum` and `sum`** use a saturating add.

**Tests:**

* `TestLayoutOverflow`, the probe table with exact floors: 8,388,608, and
  8,388,607 for 2^40+1 over 2^41+3, checked by hand;
* `FuzzSolve`, widened to widths up to `MaxSide` and sizes taken from the
  input.

**Mutation S7-1:** `Ratio` goes back to `avail * N / D`.
`TestLayoutOverflow`.

### Step 8: no mutable package variables (H11)

* **`termcaptest`:**
  * `const DefaultRunTimeout = 10 * time.Second`;
  * `(*Terminal) SetRunTimeout(d time.Duration) *Terminal`;
  * `Run` uses the terminal's timeout, else `RunTimeout`, else the
    default.
  * `RunTimeout` is marked `// Deprecated: use (*Terminal).SetRunTimeout`
    and is still read during `v0.8.x`.
* **`workspace`:**
  * functions `FocusedPaneKey()`, `ZoomedKey()`, `HiddenPanesKey()`,
    `OverlayKey()`, `ModalKey()`, `WidthKey()` and `HeightKey()` return
    unexported key values;
  * `ContextKeys` and `WhenContext` read only those;
  * the seven variables are marked deprecated, naming their functions.
* **Removal of both** in `v0.9.0`, in
  [0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md) (A1.5).
* **`scripts/apicheck.allow`** gains nothing, since everything is
  additive.

**Tests:**

* `TestKeysIgnoreReassignment`: reassigning `KeyZoomed` changes nothing
  the workspace publishes;
* `TestSetRunTimeout`.

**Mutation S8-1:** `WhenContext` reads `KeyZoomed`.
`TestKeysIgnoreReassignment`.

### Step 9: fuzz discovery and new targets (H9)

* **Discovery.**
  * `make fuzz` finds the packages with fuzz targets itself:
    `GOWORK=off go list -f '{{if .TestGoFiles}}{{.Dir}}{{end}}' ./...`,
    filtered to those whose tests declare `func Fuzz`. It runs
    `scripts/go-fuzz.sh -t $(FUZZTIME) -m 1` on each.
  * `scripts/go-fuzz_test.sh` gains a case where a package with no
    target is skipped.
  * CI's corpus upload becomes `**/testdata/fuzz/`, and the comment at
    `ci.yml:105-108` drops the hand-kept list (lines as of 2026-10-09).
* **The new targets:**

  | Target | Fuzzes | Invariant |
  | :--- | :--- | :--- |
  | `termcap.FuzzProberReplies` | `Prober.Update` after an `EnvMsg`, with the reply messages termeventtest builds and `TerminalVersionMsg` | no panic; replies over `maxReply` are refused; `Caps.Terminal` holds nothing `Line` would change |
  | `termcap.FuzzParseTmux` | `ParseTmux` | no panic; `Known` implies five fields and a version |
  | `termcap.FuzzEnumText` | each `UnmarshalText` | a successful unmarshal round-trips through `MarshalText` |
  | `termcap.FuzzColorFGBG` | `colorFGBG` | when ok, the colour is 0–15, and `dark` follows the rule |
  | `termsvc.FuzzSanitizers` | `ParseActivity`, `Link`, `SanitizeTitle` | no control character from `Line`'s drop set survives, outside `Link`'s own OSC 8 wrapper |
  | `layout.FuzzState` | `(*State).UnmarshalJSON` | a successful unmarshal round-trips; `Solve` on a fixed tree passes its checks |
  | `workspace.FuzzClip` | `clip` | exactly `h` lines, each `w` cells under the method, no `\r` |
  | `workspace.FuzzRender` | a workspace with fuzzed titles, badges, bodies and size | with `Borders`, every pane's corners and edges are intact |
  | `launch.FuzzDecideEnv` | `Decide` over fuzzed environments, with non-terminal streams | no panic; plain unless `OpenTTY`; sizes within `MaxCells` |
  | `internal/sanitize.FuzzLine` | `Line` | idempotent; the output holds no character from the drop set |

  Seeds come from the existing tests' inputs and termcaptest's replies
  (`termcaptest.go:387-492`).
* **CI.** Every target runs for 20 s in CI. A failure's corpus is
  uploaded and becomes a seed in a fix's commit.

**Mutation S9-1:** discovery skips `workspace`. `go-fuzz_test.sh`'s count
check fails.

### Step 10: the release `v0.8.0`

1. **The documents:**
   * `docs/architecture.md` gains `internal/sanitize` and the limits;
   * `README.md`'s Status names `v0.8.0`. Its first bullet, "The current
     release is `v0.7.0`", becomes `v0.8.0`, and the bullet W2 added
     ("for `v0.8.0` … Unreleased: on `main`") says since `v0.8.0`
     (added 2026-10-09);
   * the guides name `LoadDirWith`, `WithMaxArgBytes` and the clamp where
     they describe those calls.
2. **The tag.** The owner commits, has the agent run the disclosure
   guard, pushes, and with CI green tags `v0.8.0` (annotated, `-m
   "v0.8.0"`).
3. **The consumer smoke test** against `v0.8.0` uses `command.ParseArgs`,
   `WriteResult`, `workspace.RenderPlain` and `launch.Decide`.
4. **The release notes** in the execution record:
   * W2's and W3's additions;
   * W2's behaviour changes, which its close-out lists (added 2026-10-09):
     * a Loop handler's panic no longer crashes the TUI;
     * a `ThemeBuilder` builds the first theme;
     * `CallMCP` encodes a `time.Duration`;
     * a disabled prober gives the JetBrains reasons;
     * `termcap.Env` folds case on Windows;
   * the Go 1.27.2 floor, from 0016-MADR-go-1-27-2-for-go-2026-6604;
   * the behaviour changes, each with its value:
     * the `LoadDir` defaults;
     * the 1 MiB argument limit;
     * the expansion cap;
     * the window clamp;
     * the `Line` sanitizing;
   * the deprecations: `RunTimeout` and the `Key*` variables;
   * `make apicheck`'s report.

### Step 11: close-out

* **Verification,** item by item.
* **This PLAN `complete`,** and W2's PLAN's release noted.
* **`scripts/apicheck.allow`** is empty.

## Verification

* **Each finding's probe** fails at the commit before its fix, and passes
  after: C6, H1, H3, H4, H5, H10 and H11.
* **`internal/sanitize`** is the only sanitizer, and every display path
  uses it (S2-1 to S2-3).
* **The cache and window are bounded** (S3-1, S4-1).
* **`LoadDir`, arguments and expansion are limited** (S5-1, S6-1).
* **`layout` is overflow-safe** (S7-1).
* **No exported variable changes behaviour** (S8-1).
* **`make fuzz` finds every target,** including the ten new ones, and
  each runs (S9-1).
* **`v0.8.0` is tagged,** the smoke test runs, and `make apicheck` shows
  additions only.
* **Rule 2's checks are clean** on macOS and the Windows test host.

## Rollout and Rollback

* **Rollout:**
  * one commit per step;
  * the tag after Step 9, carrying W2 and W3;
  * the new limits are behaviour changes, each with a default chosen
    above any real use, and named in the notes.
* **Rollback:**
  * **Before the tag,** revert a step.
  * **After it,** a limit that hurts a real program is raised in
    `v0.8.1`. Every limit has an option, except the window clamp, whose
    constants change only by a record.

## Execution Record

### Step 1: records

No deviation.

* **The refresh.** On 2026-10-09 the owner asked the agent to verify this
  PLAN, after W2 and 0016 had landed. The agent read every fact again on
  `e45a9f9`. It ran the H1, H3, C6 and H10 probes again, on a scratch copy
  with Go 1.27.2. The results are in "Facts re-read before execution
  (2026-10-09)", with the amended lines in the Goal and in Steps 2, 4, 6,
  9 and 10.
  * Every finding still holds.
  * W2 added four paths the steps now cover: `RenderPlain`'s titles and
    bodies, `RenderPlain`'s and `WithSize`'s sizes, `ParseArgs`'s
    arguments, and a fifth fuzz target.
  * The Goal's "nine" new fuzz targets is ten.
* **The owner's choice:** `internal/limits` holds the window clamp, which
  `workspace` re-exports and `launch` uses.
* **No other record changes.** 0014-MADR's W3 names the findings, not
  where the constants live. The glossary already reserves `LoadOption`.
* **The approval:** "Approved to proceed", 2026-10-09.
* **This PLAN** is `in-progress`, and its row in `docs/README.md` follows.
* **Checks:**
  * markdownlint: clean, the records' `*` list markers aside;
  * the link check: none broken;
  * the identifier scan: no match.

### Step 2: `internal/sanitize` (C6)

#### Deviations

* **D1 (2026-10-09): `clip` uses `Styled`, not `Line`.**
  * **Found:** the step has `clip` pass each line through `Line` before
    truncating. `Line` begins with `ansi.Strip`, and `clip`
    (`workspace/render.go:158`) handles every pane's styled view on every
    frame. So every pane would lose its colour, and every colour golden
    would change, against the step's "byte-identical for clean inputs".
    The fuzz failure it fixes, `clip("\x1b", 3, 1)`, is a stray ESC, not
    a whole sequence.
  * **The owner's choice,** of three: `internal/sanitize` also gets
    `Styled(s)`. It keeps each complete escape sequence, decoded with
    x/ansi's parser, and filters the text between them as `Line` does,
    so stray controls, bidi controls and the rest go. `clip` uses
    `Styled`. Titles, badges and plain output use `Line`.
  * **The others:** `clip` drops lone controls itself, a second
    sanitizer outside `internal/sanitize`; or `Line` as written, which
    strips every pane's styling.
* **D2 (2026-10-09): `Line` also drops U+200B.**
  * **Found:** C6 names U+200B among the characters `SanitizeTitle`
    wrongly keeps, but the step's drop list for `Line` leaves it out, and
    it is not a Bidi_Control. The other five C6 names are covered.
  * **The owner's choice,** of two: `Line` drops U+200B. ZWJ (U+200D) and
    ZWNJ (U+200C) stay, as emoji and some scripts need them.
  * **The other:** follow the list exactly, and keep U+200B.

#### What was built

* **`internal/sanitize` (new):**
  * `Line`, `Styled` (D1), `Truncate`, `Token` and `HasControl`, over one
    `dropped` set: C0, DEL, C1, the Bidi_Control set, U+200B (D2),
    U+FEFF, U+2060-2064, U+00AD and the tag characters. Invalid UTF-8 is
    dropped, and a run of `\r`, `\n`, `\t`, U+2028 and U+2029 becomes
    one space. ZWJ, ZWNJ, variation selectors and combining marks stay.
  * `Styled` keeps a complete escape sequence introduced by ESC: a CSI
    with its final byte, an OSC with BEL or ST, DCS, APC, SOS or PM with
    ST, or ESC with its final byte. It drops a stray ESC, a sequence cut
    short, and an 8-bit C1 introducer.
  * **A fast path.** `Line` and `Styled` first check whether anything
    would change, and return the input when nothing would, with no
    allocation. Without it, `TestRenderAllocs` failed under `-race`: "a
    full 200 x 60 frame: 751 allocations, want at most 650", because
    `clip` runs `Styled` on every line of every frame. With it, a full
    frame makes 441 allocations, as on `cb65c6b`, and 648 under
    `-race`, against 646 on `cb65c6b`. The budget stands.
* **`termsvc`:**
  * `strip` is `sanitize.Line`; `clean` calls `Line`;
  * `SanitizeTitle` is `Line` plus its rune cut, and the `bidi` table is
    gone;
  * `ParseActivity` sanitizes the vendor;
  * `osc99ID` and `Pointer`'s filter call `Token`;
  * `LinkPolicy.Openable` calls `HasControl`.
* **`termcap`:**
  * the XTVERSION name (`prober.go`) passes through `Line`;
  * so do `TERM_PROGRAM` and `TERM` for `Caps.Terminal` (`env.go`);
  * so do `Identity`'s `Term`, `TermFeatures` and `Version`
    (`identity.go`).
* **`command`:** `isSpaceOrControl` is `c == ' ' || HasControl(string(c))`.
  It rejects what it rejected before. A slash name with a bidirectional
  or invisible character is now refused as well, which the step's "as
  today" did not foresee. Every existing test passes.
* **`workspace`:**
  * `title()` and `renderBox` sanitize the title and the badge with
    `Line`, and truncate as before;
  * `clip` passes each line through `Styled` (D1);
  * `plainTitle` uses `Line`, and `plainText` each line through `Line`.
  * No golden changed, the `plain.*` ones included.
* **Two expectations for dirty input changed,** as the step allows
  ("byte-identical for clean inputs"). Each now names the whole escape
  sequence `Line` removes, as a terminal reads it, where the old `strip`
  removed only its control bytes:
  * `TestActivity`: `Activity("ve;n\x1bdor", …)` now gives the vendor
    "venor", not "vendor", since `\x1bd` is ESC d;
  * `TestControlBytesAreStripped`'s link: the URL's embedded OSC 8 goes
    whole, giving "…/x", not "…/]8;;evilx". The text "te\x1bxt" gives
    "tet", since `\x1bx` is ESC x.

  The notification bytes in that test did not change.
* **Tests,** new:
  * `internal/sanitize/sanitize_test.go`: `TestLine` (20 cases),
    `TestStyled` (10, and agreement with `Line` on text), and
    `TestTokenAndHasControl`;
  * `termsvc/sanitize_test.go`: `TestActivityVendorSanitized`, which also
    covers `SanitizeTitle` over C6's six characters and U+202E, and
    `Openable`;
  * `termcap/sanitize_test.go`: `TestXTVersionSanitized`, over the reply,
    `TERM_PROGRAM`, `TERM` and `TERM_PROGRAM_VERSION`;
  * `workspace/sanitize_test.go`:
    * `TestTitleWithLineBreakKeepsBorder`: eight new `evil-title.*`
      goldens, `Borders` and `Separators`. No line is over the width,
      every bordered line is exactly the width, no U+202E survives, the
      top border ends in its corner, and `RenderPlain` is `"evil line [b
      ]\nbody text\n\n"`. The goldens were read before the commit;
    * `TestClipSanitizes`, with five cases.
  * The test files write every non-ASCII character as an escape, so
    nothing invisible hides in the source.

#### Checks

* **Rule 3: the probes fail before the fix.** The four new test files ran
  on a scratch copy of `cb65c6b`, the commit before this step, with the
  new goldens:
  * `TestXTVersionSanitized`: "Terminal = {Value:evil[2J…term"; the
    environment's terminal `"x\x1b]0;title\ay\u202e"`; `FromEnv Term
    = "xterm\u200b"`; `FromEnv Version = "1.2\x1b[31m\u202e"`;
  * `TestActivityVendorSanitized`: `ParseActivity vendor
    "ev\x1b[31m\u202eil\u200b"`. `SanitizeTitle` kept each of the six,
    and `Openable` did not refuse U+202E;
  * `TestTitleWithLineBreakKeepsBorder`: "line 0: 9 cells, want 30",
    "line 2 holds U+202E", and `the top border is broken: "╭─ ▸ evil"`;
  * `TestClipSanitizes`: `clip("\x1b", 3, 1) = "\x1b   "`, "a line 0
    cells wide, want 3".

  All four pass on the step's code.
* **Mutations,** on scratch copies of the tree. All 11 were killed by a
  failing test, none by a build failure:
  * **S2-1** (`Line` keeps U+202E): `TestLine`, Bidi_Control.
  * **S2-2** (`Line` drops ZWJ): `TestLine`, the emoji family.
  * **S2-3** (`renderBox` uses the raw badge):
    `TestTitleWithLineBreakKeepsBorder`, "line 0: 17 cells, want 30".
  * **S2-4** (D1, `Styled` drops whole sequences, added): `TestStyled`,
    the styling lost.
  * **S2-5** (D1, `Styled` keeps a stray ESC, added): `TestClipSanitizes`.
  * **S2-6** (`RenderPlain`'s title unsanitized, added): "evil line [b
    \u202e]".
  * **S2-7** (the raw vendor, added): `TestActivityVendorSanitized`.
  * **S2-8** (the raw XTVERSION name, added): `TestXTVersionSanitized`.
  * **S2-9** (D2, U+200B kept, added): `TestLine`.
  * **S2-10** and **S2-11** (each fast path passes everything, added):
    `TestLine` and `TestStyled`.

  The set was run again after the fast paths went in, all killed.
* **Rule 2 on macOS,** go1.27.2:
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the 17 Go files>`: "17 file(s) clean".
  * `make lint`: clean. On the first run, modernize wanted
    `strings.SplitSeq` in a test.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode. `-race` first failed
    `TestRenderAllocs`, which the fast path above fixed.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "173 file(s) clean … apicheck, examples". The
    new files are untracked; the `FILES` run covers them.
* **The Windows test host,** go1.27.2 windows/amd64:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with `TestRenderAllocs`: passed.

### Step 3: the bounded view cache (H1)

No deviation.

#### What was built

* **`workspace/workspace.go`:**
  * the cache is `map[slotKey]cachedView`, with `slotKey{kind, id,
    focused}` and `cachedView{width, height, method, themeGen, view}`;
  * `viewKey` is gone;
  * `forget` deletes the two slots by key, where it used to scan the
    whole map.
* **`workspace/render.go`:** `viewOf` hits only when the slot's width,
  height, method and theme generation all equal the request. A miss
  draws the view and replaces the slot. The cache therefore holds at
  most 2 × (panes + overlays) views.
* **`workspace/cache_test.go` (new):** `TestViewCacheBounded`. It runs:
  1. two panes and an overlay;
  2. 500 resizes, 200 theme changes and two method changes, with a render
     after each.

  It then checks:
  * the cache holds at most 6;
  * an unchanged `Changer` is not drawn again;
  * a resized one is, and the slot is replaced.
* **The existing cache tests pass unchanged:**
  * `TestPopEvictsTheOverlay`;
  * `TestSetPaneDropsTheCachedView`;
  * the theme restyling tests;
  * `TestRenderAllocs`: a full frame makes 441 allocations, and 646
    under `-race`, as before the step.

#### Checks

* **Rule 3: the probe fails before the fix.** `TestViewCacheBounded`, on
  a scratch copy of `d161122`, the commit before this step: "914 cached
  views, want at most 6 (2 × (panes + overlays))". It passes on the
  step's code.
* **Mutations,** on scratch copies of the tree. All four were killed by a
  failing test:
  * **S3-1** (a miss adds a slot, by putting the theme generation in the
    key): "603 cached views, want at most 6". Its first form left
    `forget`'s positional keys short of the new field and did not build.
    That is not a kill, so it was rewritten to build.
  * **S3-2** (a hit ignores the theme generation, added):
    `TestFollowingThemeRestyles`, "a view cached before the theme change
    was not drawn again", and `TestSetThemeRedraws`.
  * **S3-3** (a hit ignores the size, added): "a resized Changer was not
    drawn again".
  * **S3-4** (`forget` leaves the focused slot, added):
    `TestSetPaneDropsTheCachedView` and `TestPopEvictsTheOverlay`.
* **Rule 2 on macOS,** go1.27.2:
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the three Go files>`: "3 file(s) clean".
  * `make lint`: clean.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "178 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.2 windows/amd64:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's test, with `TestRenderAllocs` and the theme, pop and
    pane-replacement tests: passed.

### Step 4: the window clamp (H3)

No deviation.

#### What was built

* **`internal/limits` (new, the owner's choice of 2026-10-09):**
  `MaxSide = 4096`, `MaxCells = 1 << 19`, and `Clamp(w, h)`. `Clamp`
  puts each side in `[0, MaxSide]`, then cuts the height to
  `MaxCells / w` when the area is over. The width is kept.
* **`workspace`:**
  * `MaxSide` and `MaxCells` are exported, equal to `limits`', and
    documented with the rule;
  * `clampSize` is `limits.Clamp`;
  * the `WindowSizeMsg` case and `WithSize` clamp, and `Update`'s and
    `WithSize`'s documentation say so;
  * `RenderPlain` clamps the width and height it lays out at, so a plain
    view is asked for at most `MaxSide` cells, and its documentation
    says so.

  A pane's `SizeMsg` comes from the clamped layout, so it is bounded too.
  `TestWindowClamp` checks it.
* **`launch.Decide`** clamps `Width` and `Height` through `limits.Clamp`,
  on the plain path and the interactive one. `Decision`'s documentation
  says so. `launch` does not import `workspace`.
* **Tests:**
  * `workspace/clamp_test.go` (new):
    * `TestWindowClamp`: 2^30 × 2^30 and 4096 × 4096 give 4096 × 128;
      1000 × 600 gives 1000 × 524; 512 × 1024, 100 × 0 and 5000 × 10
      (to 4096 × 10); and -5 × -7 gives 0 × 0. It covers a
      `WindowSizeMsg` and `WithSize`, a pane's `SizeMsg`, and
      `RenderPlain(2^30)` through a pane that reports the width it was
      asked for. It is written in numbers, not the constants, so it also
      runs on the code before the clamp;
    * `TestClampConstants`;
    * `TestRenderAtClamp`, skipped under `-short`: a 4096 × 128 frame
      renders 128 lines, none over 4096 cells, in 1,475 allocations,
      measured by `testing.AllocsPerRun` and logged;
  * `launch/clamp_test.go` (new): `TestDecideClampsSize`, for a 2^30 ×
    2^30 fake terminal, `COLUMNS=99999999`, and a 120 × 40 terminal
    unchanged.

#### Checks

* **Rule 3: the probes fail before the fix.** `TestWindowClamp` and
  `TestDecideClampsSize`, on a scratch copy of `9675c76`, the commit
  before this step, without the two tests that name the new constants:
  * "WindowSizeMsg 1073741824 × 1073741824 gives 1073741824 ×
    1073741824, want 4096 × 128", the same for `WithSize`, and for 4096
    × 4096, 1000 × 600 and 5000 × 10;
  * "the pane was told 1073741824 × 1073741824";
  * `RenderPlain(2^30) = "a\nplain at 1073741824\n\n"`;
  * "a 2^30 × 2^30 terminal: interactive true, 1073741824 ×
    1073741824", and "COLUMNS=99999999: … 99999999 × 0".

  Both pass on the step's code. `RenderPlain`'s part of the test first
  looked only at line widths, which a short pane body never exceeds, so
  it could not see the clamp. It became the width-reporting pane above.
* **Mutations,** on scratch copies of the tree. All six were killed by a
  failing test:
  * **S4-1** (no `MaxCells` step): "WindowSizeMsg 1073741824 ×
    1073741824 gives 4096 × 4096, want 4096 × 128".
  * **S4-2** (no `MaxSide` step on the width, added): "gives 1073741824 ×
    0".
  * **S4-3** (`RenderPlain` unclamped, added): "plain at 1073741824".
  * **S4-4** (`WithSize` unclamped, added): `TestWindowClamp`.
  * **S4-5** and **S4-6** (`Decide`'s plain or interactive path
    unclamped, added): `TestDecideClampsSize`.
* **Rule 2 on macOS,** go1.27.2:
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the six Go files>`: "6 file(s) clean".
  * `make lint`: clean.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode. `TestRenderAllocs` stays at
    441.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
  * `make examples`: "clean".
  * `make release-check`: "179 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.2 windows/amd64:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with `TestRenderAtClamp` and `TestRenderAllocs`:
    passed.

### Step 5: `LoadDir`'s limits and the expansion cap (H4)

#### Deviations

* **D3 (2026-10-09): the defaults are exported, and a value below 1 keeps
  the default.**
  * **Found:**
    * The step gives the default values, but not exported names for
      them. The code exported `DefaultMaxFileBytes`, `DefaultMaxFiles`
      and `DefaultMaxDepth`, as Step 6 plans `DefaultMaxArgBytes`.
    * The step does not say what an option given a value below 1 does.
  * **The owner's choices:**
    * keep the three constants exported, so a program can show or reason
      about the limits;
    * a value below 1 keeps the default, so a limit cannot be switched
      off by accident.

    Each option's documentation says so, and `TestLoadDirWithOptions`
    checks it.
  * **The others:** unexported constants; or a value below 1 turns the
    limit off.

#### What was built

* **`command/loaddir.go`:**
  * `ErrFileTooLarge`, `ErrTooManyFiles` and `ErrTooDeep`;
  * the three defaults (D3);
  * `LoadOption`, an opaque interface, with `WithMaxFileBytes`,
    `WithMaxFiles` and `WithMaxDepth`;
  * `LoadDirWith(fsys, src, opts...)`. `LoadDir(fsys, src)` keeps its
    signature and calls it with the defaults: a behaviour change for the
    release notes.
* **How each limit is enforced:**
  * **Size:** `readLimited` opens the file and reads
    `io.LimitReader(f, limit+1)`, so `Stat` is never trusted. One byte
    over gives `ErrFileTooLarge`.
  * **Depth:** a directory with more than the limit's levels below the
    root gives one `ErrTooDeep` error, and `fs.SkipDir`.
  * **Count:** the first `.md` file past the limit gives one
    `ErrTooManyFiles` error, and `fs.SkipAll`. The files before it load.
  * Each error is wrapped as `command: <path>: …`, with the limit.
* **`command/expand.go`:** `expand` returns `(string, error)`. It writes
  through a builder, and stops before passing `maxExpansion` (1 MiB),
  with an `*ArgError`: "the expanded prompt is over 1048576 bytes". The
  handler `LoadDir` builds returns that error.
* **`command/frontmatter_test.go`:** `TestExpand` takes `expand`'s new
  error and requires it nil. Its expected text did not change.
* **`command/loadlimits_test.go` (new):**
  * `TestLoadDirLimits`, over separate file systems:
    * exactly 256 KiB loads, and one byte over is refused;
    * 8 MiB is refused, and so is 8 MiB behind an `fs.FS` whose files
      say they are one byte long;
    * exactly 1,000 files load, and of 5,000, 1,000 load with one error;
    * 8 levels down loads, and 9 and 64 levels are refused.
  * `TestLoadDirWithOptions`: each option and its error, and values
    below 1 keeping the defaults (D3);
  * `TestExpandCap`: through the registry, sixteen `$A` of 64 KiB is
    exactly 1 MiB, and one byte more is an `*ArgError` naming the limit.
    The arguments stay under Step 6's 1 MiB argument limit.

  The first and last use only names that existed before the step.

#### Checks

* **Rule 3: the probes fail before the fix.** `TestLoadDirLimits` and
  `TestExpandCap`, on a scratch copy of `bfd92a4`, the commit before this
  step:
  * "one byte over 256 KiB: 1 loaded, 0 errors", the same for 8 MiB and
    for the lying file system;
  * "5,000 files: 5000 loaded, 0 errors";
  * "9 levels down: 1 loaded" and "64 levels down: 1 loaded";
  * "one byte over 1 MiB: 1048592 bytes, <nil>".

  Both pass on the step's code.
* **Mutations,** on scratch copies of the tree. All five were killed by a
  failing test:
  * **S5-1** (the size checked with `Stat`): "8 MiB that says it is 1
    byte: 1 loaded, 0 errors". Its first form kept the length check after
    the read, so it still refused the file and survived. That was a
    mutant weaker than the step's. It was rewritten to trust `Stat`
    alone, as the step means.
  * **S5-2** (one level less deep, added): "8 levels down: 0 loaded".
  * **S5-3** (no file count, added): "5,000 files: 5000 loaded".
  * **S5-4** (no expansion cap, added): "one byte over 1 MiB: 1048592
    bytes".
  * **S5-5** (`WithMaxFiles` ignored, added): "WithMaxFiles(3): 10
    loaded".

  They were run again after the lint fixes below, all killed.
* **Rule 2 on macOS,** go1.27.2:
  * `gofmt -l`: nothing.
  * `make pre-add-check FILES=<the four Go files>`: "4 file(s) clean".
  * `make lint`: clean. On the first run, gocritic asked for the depth
    test as `strings.Count(p, "/") >= cfg.maxD`, which is equivalent, and
    revive refused a parameter named `max`, now `limit`.
  * With `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    all passed, and so did workspace mode.
  * `go mod tidy -diff`, `make vuln` and `scripts/go-modules.sh --check`
    were clean.
  * `make apicheck`: "against v0.7.1, 0 incompatible change(s)".
    `LoadDir`'s signature is unchanged.
  * `make examples`: "clean".
  * `make release-check`: "182 file(s) clean … apicheck, examples".
* **The Windows test host,** go1.27.2 windows/amd64:
  * the `FILES` and no-list `make pre-add-check`, `make lint`, `make
    vuln` and `make examples`: each exit 0;
  * `GOWORK=off go test -count=2 -shuffle=on ./...`: exit 0;
  * the step's tests, with `TestExpand` and the `LoadDir` golden and
    source tests: passed.
