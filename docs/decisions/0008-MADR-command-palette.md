---
status: accepted
date: 2026-10-02
decision-makers: owner
consulted: 0003-REPORT-agent-tui-ecosystem-research.md (§3, §4, §7); Textual's command palette; crush, toad and codex sources; fzf's and sahilm/fuzzy's scoring; Charm v2 APIs (bubbles v2.2.1, lipgloss v2.0.6, x/ansi v0.11.8)
informed: pi-go
---
# Find and run anything from one command palette, built on an in-repository fuzzy matcher and asynchronous, scoped providers

## Context and Problem Statement

On 2026-10-01 the owner asked for the next round of go-tui-lib work:

> i want to stay working on go-tui-lib, i want to ensure the
> bubbletea/lipgloss/charm stack is tightly integrated, coded idiomatically,
> based on go1.27.1 optimizations and standards. i want to enhance and expand
> the tui library functionality in 5 more ways that will bring benefit and
> value to the codebase functionality. i want a large, openstandards api
> surface, with a wide assortment of agentic commands available through the
> terminal UI and also the TUI. look at similar opensource projects across
> the web. dig into how they solved interesting problems and issues bringing
> functionality to reality. find ideas, features, enhancements, and
> optimizations. present 10 items to spec out and plan. future-proof is a
> focus.

On 2026-10-02, after seeing the ten items, the owner said: "write findings
into a report then follow recommendations and proceed." The recommendation
picked five expansions. This record is the third of them, the command
palette (item 3 of
[0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§7).

The owner's standing direction also applies: build speculative API when it
is sensible, for extensibility, flexibility and idiomatic, modular design.

A large command surface is only usable if a person can find a command
without knowing its name or key. pi-go also needs the same search for
`@path` completion and slash completion, and for its model, session and
theme pickers ([0002-MADR-multi-pane-workspace-layouts.md](0002-MADR-multi-pane-workspace-layouts.md),
Evidence). This record decides how go-tui-lib searches and how it shows the
results.

Evidence (read-only, 2026-10-02):

* **Textual's palette is built from providers.** A `Provider` has async
  `startup()`, `search(query)`, `discover()` and `shutdown()`. `search`
  yields scored `Hit`s with a callback and help text. `discover` yields
  unscored hits for an empty query. Providers attach to the App (global) or
  to a Screen (scoped). A provider's exception is logged, not fatal.
  Workers marked `exclusive` cancel earlier runs, because answers arrive out
  of order (0003-REPORT §4).
* **crush sorts fuzzy hits by tier.** `internal/ui/completions` filters with
  sahilm/fuzzy, then sorts stably by tier: exact base name or stem, then
  base-name prefix, then a whole path segment, then everything else
  (`namePriorityRules`, `applyNamePriorityFilter`). Read in the scratch
  clone at the commit 0003-REPORT §3 names.
* **toad narrows big candidate sets with trigrams.** `fuzzy_index.py`
  case-folds the paths and builds an inverted trigram index over each path
  padded with spaces. A one-character query keeps paths that start with it
  or have it after a `/`. A query of two or three characters keeps paths
  that contain all its characters. A longer query keeps paths that share at
  least 30% of its trigrams. At most 2000 candidates reach the scorer.
* **codex runs one search session per query.** `tui/src/file_search.rs`
  owns one `codex-file-search` session, updates its query on each
  keystroke, drops it when the query empties, and tags each session with an
  incrementing `session_token` so a late result from an old session is
  ignored. The search crate uses `nucleo` and the `ignore` crate.
* **fzf's scorer is the reference.** `src/algo/algo.go` has a greedy O(n)
  `FuzzyMatchV1` and a modified Smith-Waterman `FuzzyMatchV2` that finds the
  best-scoring alignment. Its constants are `scoreMatch` 16,
  `scoreGapStart` -3, `scoreGapExtension` -1, `bonusBoundary` 8,
  `bonusNonWord` 8, `bonusCamel123` 7, `bonusConsecutive` 4,
  `bonusFirstCharMultiplier` 2, `bonusBoundaryWhite` 10 and
  `bonusBoundaryDelimiter` 9 (read 2026-10-02). fzf is MIT-licensed
  *(licence from prior knowledge, not re-checked)*.
* **What Charm v2 already has, and where it falls short:**
  * **bubbles `list`** filters in a `tea.Cmd` with sahilm/fuzzy
    (`DefaultFilter`, `filterItems`), over one item set. Its delegate
    highlights with `lipgloss.StyleRunes`, by rune index. Importing `list`
    would add `github.com/sahilm/fuzzy` v0.1.3 to this module's graph; it
    is not in `go.sum` today.
  * **`lipgloss.StyleRunes`** indexes runes, so a highlight can split a
    grapheme cluster (an emoji with a ZWJ, a letter with a combining mark).
    **`lipgloss.StyleRanges`** takes cell ranges and cuts with `ansi.Cut`.
  * **`x/ansi`** has `FirstGraphemeCluster(s, method)`, and `Method` has
    `StringWidth` and `Cut`, so this library can walk graphemes and measure
    in cells with the method the terminal uses (the 2027 alignment in
    [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)).
  * **bubbles `textinput`** has pointer-receiver `Focus` and `Blur` and a
    value `Update(tea.Msg) (Model, tea.Cmd)`, so it is hosted through the
    `workspace.Model[M]` adapter that the 0002 hardening plan adds.
  * **`workspace.Overlay`** already places a pane above the layout:
    centred, below the cursor or on a pane, modal or not. A non-modal
    overlay takes only Esc and the mouse over it, and other keys reach the
    focused pane, which drives it. That is the shape of a completion pop-up.
* **Go 1.27.1** (0003-REPORT §2, and `$GOROOT/api/go1.27.txt`):
  * generic methods on concrete types, so `(*Matcher).Rank[T]` can rank any
    item type;
  * `encoding/json/v2` with `UnmarshalRead`, `MarshalWrite` and
    `RejectUnknownMembers`;
  * `testing/synctest` with `Sleep`, for debounce and streaming tests;
  * the `goroutineleak` profile in `runtime/pprof`, for proving that a
    cancelled search leaves no goroutine behind.

## Decision Drivers

* **One search engine for every list a user filters:** the palette, slash
  completion, `@path` completion and pickers. Ranking that differs between
  them feels broken.
* **The UI never waits for a search.** A file index of 100 000 paths, or a
  provider that asks an agent over ACP, must not stall a keystroke.
* **Stale answers never win.** A slow answer to an old query must never
  replace the answer to the current one.
* **Correct on every terminal.** Highlights never split a grapheme, widths
  are cells in the terminal's own method, and the palette reads under
  `NO_COLOR` and ASCII (0001-MADR §6, rules 3–5).
* **Extensible without changing this library.** A program adds its own
  sources (agent sessions, models, MCP prompts) as providers, scoped to
  where they make sense.
* **The library owns no files.** Recents are read and written through the
  caller's `io.Reader` and `io.Writer`, and the file provider walks an
  `fs.FS` the caller passes.
* **No new module** unless a record shows it pays for itself (AGENTS.md,
  Dependencies).

## Considered Options

* **A. Write `fuzzy` (matcher, tiers, trigram index) and `palette` (an overlay pane with asynchronous, scoped providers) in this repository.**
* **B. Use bubbles `list` and its built-in filter as the palette.**
* **C. Depend on a fuzzy-matching module** (sahilm/fuzzy, or a port of fzf's algorithm) and write only the palette here.
* **D. A synchronous palette:** providers return slices, and ranking runs inside `Update`.

## Decision Outcome

Chosen option: **"A"**, because:

* it is the only option that gives every list the same tiered ranking with
  grapheme-safe highlights;
* it keeps searches off the event loop, with cancellation and stale-result
  rejection;
* it adds no module.

### 1. Packages and their dependencies

```text
 palette   overlay pane: input, results, providers, scopes, recents,
           argument prompts       → workspace, command, when, fuzzy, theme,
                                    glyph, bubbles/textinput, bubbles/key
 fuzzy     matcher, tiers, ranking, trigram index → stdlib, x/ansi
```

* **Imports point downward.** `fuzzy` imports nothing above `x/ansi`, so a
  program can use it without Bubble Tea. `palette` imports the registry and
  `when` from
  [0006-MADR-command-registry.md](0006-MADR-command-registry.md) for its
  command provider.
* **No direct key handling of its own.** The palette's keys are bindings
  under the `palette` context, so the keymap of
  [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) can rebind them.
  Without a keymap, the palette's `DefaultKeyMap()` applies.

### 2. `fuzzy`: the matcher

```go
// Matcher is a compiled query. It is immutable and safe for concurrent use.
type Matcher struct{ /* folded query graphemes, terms, options */ }
func New(query string, opts ...Option) *Matcher
// Options: CaseMode(Smart | Insensitive | Sensitive), Normalize(func(rune) rune),
// Delimiters(string), Extended() (fzf-style terms), MaxAlign(n).

// Score is the cheap path: no positions, no allocation for ASCII input.
func (m *Matcher) Score(candidate string) (Score, bool)
// Match also returns the matched spans, for the few hits that are drawn.
func (m *Matcher) Match(candidate string) (Match, bool)

type Score struct {
    Tier  Tier // Exact, Prefix, Boundary, Subsequence; lower is better
    Value int  // alignment score within the tier; higher is better
}
type Match struct {
    Score
    Spans []Span // byte offsets, each on a grapheme boundary
}
type Span struct{ Start, End int }

// Rank scores items lazily and keeps the best limit, stable on ties.
func (m *Matcher) Rank[T any](items iter.Seq[T], key func(T) string, limit int) []Ranked[T]
func (m *Matcher) RankContext[T any](ctx context.Context, items iter.Seq[T],
    key func(T) string, limit int, boost func(T) int) ([]Ranked[T], error)
type Ranked[T any] struct{ Item T; Score Score }

// Cells turns spans into cell ranges for lipgloss.StyleRanges.
func Cells(s string, spans []Span, method ansi.Method) []CellRange
```

* **Unit of matching is the grapheme cluster.** Candidates are walked with
  `ansi.FirstGraphemeCluster`. A query grapheme matches a candidate
  grapheme when their runes are equal under simple case folding
  (`unicode.SimpleFold`). Spans therefore never split a cluster.
* **ASCII fast path.** When the query and the candidate are both ASCII, the
  matcher works on bytes with a folding table and allocates nothing.
* **Case.** `Smart` is the default: a query with an upper-case letter is
  case-sensitive, as in fzf and Vim's `smartcase`.
* **Scoring.** A greedy pass rejects non-matches in O(n). A match is then
  aligned with a Smith-Waterman-style dynamic programme that rewards
  consecutive matches, word and path boundaries, camelCase and digit
  transitions, and the first character, and penalises gaps. fzf's published
  constants are the starting point, and the tests pin the resulting order.
  Candidates longer than `MaxAlign` graphemes (default 512) keep the greedy
  alignment.
* **Tiers come before score**, as in crush:
  1. **Exact:** the candidate, its last path element or its stem equals
     the query.
  2. **Prefix:** the candidate or its last path element starts with the
     query.
  3. **Boundary:** a whole word or path segment equals the query, or every
     query grapheme lands on a boundary.
  4. **Subsequence:** any other match.
* **Rank order** is tier, then score plus the caller's `boost` (bounded by
  one consecutive-match bonus, so recency breaks near-ties without
  overturning a better match), then shorter candidate, then input order.
  The order is total, so ranking is deterministic.
* **Extended syntax**, opt-in with `Extended()`: space-separated terms all
  match (AND), and fzf's `'exact`, `^prefix`, `suffix$` and `!not` prefixes
  apply. The default treats spaces as ordinary characters, which suits
  titles such as "Open Settings".
* **`Normalize`** is a hook for folding accents. This library does not ship
  a Unicode normaliser, which would need `golang.org/x/text`.

### 3. `fuzzy`: the trigram index

```go
// Index is an immutable inverted index over a candidate set.
type Index struct{ /* folded strings, postings as sorted []uint32 */ }
func NewIndex(candidates []string, opts ...IndexOption) *Index
func (ix *Index) With(add []string, remove []int) *Index // copy-on-write
func (ix *Index) Len() int
func (ix *Index) At(i int) string
// Candidates yields the indices worth scoring for this query.
func (ix *Index) Candidates(q *Matcher) iter.Seq[int]
// IndexOptions: MinOverlap(0.3), MaxCandidates(2000).
```

* **Toad's strategy:**
  * one grapheme: candidates that start with it, or have it after a
    delimiter;
  * two or three: candidates that contain them all, by a 64-bit
    ASCII-letter mask with a slow path for other runes;
  * four or more: candidates sharing at least `MinOverlap` of the query's
    padded trigrams.
* **The index is lossy on purpose.** A sparse subsequence such as `fbz` for
  `foo/bar/baz.go` shares no trigram with it. So the index is used only
  above a size threshold that the caller picks (the file provider's
  default is 5000 candidates). Below it, every candidate is scored. Tests
  record the recall on a fixture corpus so the trade-off stays visible.
* **Immutable and copy-on-write.** A search goroutine reads an `*Index`
  without locks while a rebuild produces the next one.

### 4. `palette`: providers, scopes and streaming

```go
type Provider interface {
    Name() string
    // Search yields hits for a non-empty query until ctx is cancelled or
    // yield returns false. It runs off the event loop.
    Search(ctx context.Context, q Query, yield func(Hit) bool) error
}
// Optional, found by type assertion:
type Discoverer interface{ Discover(ctx context.Context, yield func(Hit) bool) error }
type Starter interface{ Start(ctx context.Context) error } // once, before first use
type Stopper interface{ Stop() }                           // when the palette closes
type Prefixed interface{ Prefix() string }                 // ">" commands, "@" files
type Debouncer interface{ Debounce() time.Duration }       // e.g. 30 ms for files

type Query struct {
    Text    string
    Matcher *fuzzy.Matcher // compiled once per keystroke and shared
    Context when.Context   // the context keys at open, per 0006
}

type Hit struct {
    ID      string       // stable; keys recents and de-duplication
    Text    string       // the line shown
    Spans   []fuzzy.Span // filled for drawn hits only
    Score   fuzzy.Score
    Group   string       // a category or the provider's name
    Help    string       // the selected hit's description line
    Hint    string       // a key hint, e.g. from the keymap
    Danger  command.Danger
    Disabled string      // non-empty: shown, not runnable, with this reason
    Run     tea.Cmd      // what selecting it does
    Args    json.RawMessage // a JSON Schema: prompt for arguments first (§6)
}

type Palette struct{ /* textinput via workspace.Model, results, state */ }
func New(opts ...Option) *Palette
func (p *Palette) Register(pr Provider, s Scope) // global, or scoped
func (p *Palette) Overlay(o ...OverlayOption) workspace.Overlay
// Options: WithTheme, WithGlyphs, WithKeyMap, WithRecents(*Frecency),
// WithLimit(n), WithGrouping(ByScore | ByProvider), WithPlaceholder,
// WithLogger(slog.Handler).
type Scope struct{ When string } // a 0006 `when` expression; "" is global
```

* **A Go `yield` callback, not a channel, is the provider's contract.**
  Providers are plain Go, and a provider can be written as a range-over-func
  iterator. The palette owns the goroutines and channels.
* **Streaming into Bubble Tea.** Each keystroke:
  1. increments a generation counter and cancels the previous search's
     context;
  2. compiles one `fuzzy.Matcher`;
  3. for each provider in scope (after its debounce, with a `tea.Tick`
     that carries the generation), starts a goroutine that calls `Search`
     and sends hits into a bounded channel;
  4. returns a `tea.Cmd` that waits for the first hit, drains what else is
     ready for up to one frame (16 ms) or 64 hits, and returns
     `hitsMsg{gen, hits, done}`.

  `Update` drops any `hitsMsg` whose generation is not current, merges the
  rest into a bounded top-k, and issues the next drain command until every
  provider is done. This is codex's session token and Textual's `exclusive`
  worker in Bubble Tea terms.
* **Spans only for drawn hits.** Providers return scores. The palette calls
  `Matcher.Match` only for the rows on screen.
* **Failures are contained.** A provider that returns an error, or panics,
  is recovered, logged to the caller's optional `slog.Handler`, and shown
  as one muted row naming the provider. Other providers keep streaming.
* **Scopes.** A provider's `Scope.When` is evaluated against the
  `when.Context` at open, so a provider can belong to one pane, one overlay
  or one mode. `Prefixed` providers own their prefix: `>` restricts the
  query to commands and `@` to files, as in VS Code. `?` lists the
  prefixes.
* **De-duplication.** Two hits with the same `ID` keep the better score.
* **Not a `tea.Model`.** As with `workspace`, the palette is a pane. The
  program pushes `p.Overlay()` and the workspace routes to it.

### 5. Built-in providers

* **`Commands(reg *command.Registry)`** lists the registry's commands whose
  `when` holds. It matches the title, `Category: Title`, the slash name and
  the aliases, and keeps the best. Hits carry the command's `Danger` and
  `Args`, and run through the registry's execution path, so confirmation
  policy stays in the registry
  ([0006-MADR-command-registry.md](0006-MADR-command-registry.md)).
* **`Recents(f *Frecency, inner Provider)`** puts recently and frequently
  used hits first for an empty query, and boosts them for a typed one.
* **`Static(name string, items []Item)`** covers small fixed sets: themes,
  layout presets, models, modes.
* **`Panes(w *workspace.Workspace)`** jumps focus to a pane by title, and
  lists hidden panes to show them again.
* **`Files(walk Walker, opts ...FileOption)`** builds a `fuzzy.Index` in
  `Start` and searches it:

  ```go
  type Walker interface {
      Walk(ctx context.Context, yield func(path string) bool) error
  }
  func FSWalker(fsys fs.FS, skip PathFilter) Walker
  type PathFilter interface{ Skip(path string, d fs.DirEntry) bool }
  ```

  The caller passes the `fs.FS` (for example `os.DirFS(root)`), so the
  library opens nothing itself. Ignore rules come through `PathFilter`; a
  `.gitignore` reader is a later record. Ranking uses crush's tiers on the
  last path element.
* **Completion pop-ups reuse all of this.** `palette.New(AsCompletion())`
  makes a non-modal palette with no input of its own. The editor pane sends
  it `palette.SetQuery(q)` as the `@` or `/` token changes, and it answers
  with `palette.SelectedMsg`. pi-go's slash and `@path` completion are then
  two providers, not two widgets.

### 6. Selection, arguments and recents

* **Selection.** Enter, or a click, runs the hit: the palette emits
  `SelectedMsg{Hit}`, batches `Hit.Run`, records the hit in the recents
  store, and closes. A disabled hit shows its reason and does not run.
* **Argument prompts.** A hit with an `Args` schema opens a second stage in
  the same overlay. It handles the simple, common schema shapes: an object
  whose properties are strings, integers, numbers, booleans or `enum`s,
  with `required`, `default` and `description`. It prompts for each in
  order, offers enums through the same fuzzy list, and validates before
  running. A schema outside that subset is handed to the command's own UI
  through `SelectedMsg` with no arguments.
* **Frecency.** Each use adds 1 to a score that decays with a half-life
  (default 7 days), stored as `(score, at)` and updated on read, so no
  timer runs. The store keeps the best 500 entries.

  ```go
  type Frecency struct{ /* entries; injectable clock */ }
  func NewFrecency(opts ...FrecencyOption) *Frecency // HalfLife, Max, Clock
  func (f *Frecency) Record(id string)
  func (f *Frecency) Score(id string) float64
  func (f *Frecency) WriteTo(w io.Writer) (int64, error)
  func (f *Frecency) ReadFrom(r io.Reader) (int64, error)
  ```

  The format is versioned JSON
  (`{"version":1,"entries":[{"id":…,"score":…,"at":…}]}`), decoded with
  `encoding/json/v2` and `RejectUnknownMembers`. An unknown version is
  refused, as `layout.State` refuses one.

### 7. Rendering and interaction

* **Layout.** An input row, the results, and a help line with the selected
  hit's description and a count ("12 of 340"). While providers are still
  running, the count says so in words with the `Ellipsis` glyph. Nothing
  animates.
* **Results** are a window over the top-k, so only visible rows render.
  Group headers appear with `ByProvider`.
* **Highlights** use `fuzzy.Cells` and `lipgloss.StyleRanges`, in the
  terminal's width method. Matched text is underlined as well as coloured,
  so it survives `NO_COLOR` and the ASCII profile (rule 4).
* **The selected row** carries `glyph.Set.Focus` and bold, not colour alone.
* **Danger** shows as a badge in words between `BadgeOpen` and
  `BadgeClose` (for example `[destructive]`), styled with `Warning` or
  `Error`.
* **Truncation** is in cells with the `Ellipsis` glyph (rule 5). A match
  inside the truncated part is not highlighted.
* **Keys** (the `palette` context, all rebindable): up/down and
  ctrl+p/ctrl+n move, pgup/pgdn page, enter runs, tab completes the input
  to the selected text, and Esc clears a non-empty query, then closes. The
  palette implements `workspace.EscConsumer` to get that behaviour.
* **Mouse.** The wheel scrolls the results, and a click runs a row. The
  workspace already delivers pane-local coordinates.
* **Placement.** `Overlay` centres the palette by default. A top-centred
  anchor, as VS Code uses, would need a new `workspace.AnchorKind`; that
  waits for a workspace record.

### 8. Versioning

* The packages arrive in a minor release after 0007's. The PLAN lists what
  each step adds, and the owner tags.
* pi-go adopts them under its own records.

### Consequences

* Good, because every filtered list in a program ranks the same way, with
  the same highlights and keys.
* Good, because a slow or failing provider never blocks typing or another
  provider, and a stale answer never shows.
* Good, because programs extend the palette with providers and scopes
  instead of forking it.
* Good, because `fuzzy` is usable on its own, outside Bubble Tea.
* Good, because recents and file access stay with the caller.
* Neutral, because the trigram index trades recall for speed above a
  threshold, and the tests record that trade.
* Bad, because a fuzzy scorer is subtle code to own. Its order is pinned by
  table tests and fuzzing.
* Bad, because the streaming design adds goroutines and channels. The leak
  and race tests are part of the work.
* Bad, because the argument prompts cover only a subset of JSON Schema.
  Other commands keep their own UI.

### Confirmation

* `fuzzy`:
  * table tests pin tier and order for commands, paths, camelCase, digits,
    non-Latin scripts, combining marks and ZWJ emoji;
  * property tests: every reported span is on a grapheme boundary and
    inside the candidate; spans spell the query under folding; ranking is a
    total, stable order;
  * fuzz targets (`FuzzMatch`, `FuzzIndex`) find no panic
    and no out-of-range span;
  * the index never drops an exact or prefix match, and its recall on the
    fixture corpus is recorded;
  * benchmarks with `b.Loop`: `Score` on ASCII allocates nothing; ranking
    10 000 command titles, and 100 000 paths through the index, are
    measured and recorded.
* `palette`, through real `tea` messages and fake providers:
  * a stale `hitsMsg` is dropped, and a slow provider's late hits never
    appear under a newer query (`testing/synctest`);
  * a cancelled search leaves no goroutine (the `goroutineleak` profile);
  * a panicking provider is contained and named;
  * scopes, prefixes, de-duplication, debounce, recents boost, argument
    prompts, Esc twice, mouse wheel and click;
  * golden frames across the 0001 §6 matrix at two widths, including
    danger badges, disabled rows and truncated highlights.
* Mutation proofs for the key invariants, each seen failing on a scratch
  copy. Pre-add, `-race`, `LC_ALL=C` and the Windows host pass. CI is green
  on the push.

## Pros and Cons of the Options

### A. `fuzzy` and `palette` here

* Good, because ranking, highlighting and streaming are designed together.
* Good, because the matcher walks graphemes, so highlights are correct for
  every script and emoji.
* Good, because no module is added.
* Bad, because it is the most code to write and test.

### B. bubbles `list` as the palette

* Good, because it exists, with filtering, pagination and help.
* Bad, because it filters one item set, with no providers, scopes or
  streaming.
* Bad, because its highlight indexes runes, which can split a grapheme.
* Bad, because it adds `github.com/sahilm/fuzzy` to the module graph, which
  needs a record.
* Bad, because its delegate and spinner draw glyphs that do not come from
  `glyph` (0001-MADR §6, rule 3).

### C. A fuzzy-matching module and our own palette

* Good, because the scorer would be someone else's maintenance.
* Bad, because sahilm/fuzzy reports rune indexes and has no tiers, no
  index and no cancellation. A port of fzf would be a large copy to keep in
  step.
* Bad, because it is a new module (AGENTS.md, Dependencies) for the
  smallest part of the work.

### D. A synchronous palette

* Good, because it is the simplest: no goroutines, no generations.
* Bad, because a large file set, or a provider that asks an agent, stalls
  every keystroke.
* Bad, because adding async later changes the provider contract.

## Owner questions

*Answered 2026-10-02* (picked from options): Q1 "Yes, simple schema
subset"; Q2 "Opt-in Extended()"; Q3 "ctrl+p"; Q4 "Not here; PathFilter".
Every answer is the recommendation, so the decision text is unchanged.

* **Q1. Argument prompts in this record.** Recommended: yes, for the
  simple schema subset in §6. It makes commands with arguments runnable
  from the palette, which is the "agentic commands" goal. The alternative
  is to defer them, and commands with arguments open their own UI.
* **Q2. Extended query syntax.** Recommended: ship it opt-in
  (`Extended()`), off in the palette and on in the file provider. The
  alternative is no extended syntax.
* **Q3. The default palette key.** Recommended: `ctrl+p` opens the command
  palette (Textual, Posting and VS Code), with the binding owned by the
  keymap so a program can change it. The alternative is no default, which
  leaves every program to choose.
* **Q4. A `.gitignore` reader.** Recommended: not here. The file provider
  takes a `PathFilter`, and a reader is a later record. The alternative is
  a small reader here, which is more code to own.

## More Information

* [0003-REPORT-agent-tui-ecosystem-research.md](../reports/0003-REPORT-agent-tui-ecosystem-research.md):
  §3 (crush, codex, toad), §4 (Textual, Posting, k9s), §7 (item 3).
* [0002-MADR-multi-pane-workspace-layouts.md](0002-MADR-multi-pane-workspace-layouts.md)
  §3 (overlays) and
  [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md)
  (`workspace.Model[M]`, focus messages), and
  [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
  (the width method).
* [0006-MADR-command-registry.md](0006-MADR-command-registry.md): the
  registry, `when` and `Danger`, which the command provider reads.
* [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md): the `palette`
  context and key hints.
* Sources read: Textual's command palette guide
  (`textual.textualize.io/guide/command_palette/`); crush
  `internal/ui/completions/completions.go`; toad `src/toad/fuzzy_index.py`;
  codex `codex-rs/tui/src/file_search.rs`; fzf `src/algo/algo.go`; bubbles
  v2.2.1 `list/list.go` and `list/defaultitem.go`; lipgloss v2.0.6
  `ranges.go` and `runes.go`.
