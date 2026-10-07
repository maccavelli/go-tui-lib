---
status: proposed
date: 2026-10-07
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
  cover the parsers and renderers.

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
func Line(s string) string

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
    the fuzz failure.

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

* **`const MaxSide = 4096` and `const MaxCells = 1 << 19`** (524,288
  cells, about 56 MiB of frame). For scale, an 8K display with a 6 × 12
  pixel font is about 1280 × 360, or 460,800 cells.
* **`clampSize(w, h)`:**
  1. each side goes into `[0, MaxSide]`;
  2. then, while `w*h > MaxCells`, `h` becomes `MaxCells / w`.

  Width is kept, so lines stay whole, and the frame draws top-left.
* **Used by:** the `WindowSizeMsg` case, W2's `WithSize`, and `SizeMsg` to
  panes.
* **`launch.Decide`** caps `Width` and `Height` with the same constants
  (an additive change to `launch`).
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
* **`LoadOption`** is opaque, under W0.4, with:
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
  same cap applies to a slash line's `Raw`.
* **Over the limit:** `&ArgError{Reason: "arguments are N bytes, over M"}`,
  and the audit records `Args: nil`.
* **`FuzzPrepareMask`,** over a schema with a secret field, a slice of
  secrets and a nested object. Its invariants:
  * no panic;
  * a successful `prepare` gives valid JSON;
  * `mask` gives `"***"` at every secret path, or nil;
  * no secret of four or more bytes from the input appears in `mask`'s
    output.

**Tests:** `TestArgLimit` covers `CallMCP` at 16 MiB, `Run`, slash, and
exactly at the limit.

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
    `ci.yml:101-104` drops the hand-kept list.
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
   * `README.md`'s Status names `v0.8.0`;
   * the guides name `LoadDirWith`, `WithMaxArgBytes` and the clamp where
     they describe those calls.
2. **The tag.** The owner commits, has the agent run the disclosure
   guard, pushes, and with CI green tags `v0.8.0` (annotated, `-m
   "v0.8.0"`).
3. **The consumer smoke test** against `v0.8.0` uses `command.ParseArgs`,
   `WriteResult`, `workspace.RenderPlain` and `launch.Decide`.
4. **The release notes** in the execution record:
   * W2's and W3's additions;
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

Not started.
