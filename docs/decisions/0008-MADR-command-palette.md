---
status: accepted
date: 2026-10-11
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

## Amendments

### A1 (2026-10-07): the width method's source, and the registry after 0006 A4 to A6

*Status: accepted (2026-10-07).* Found by the documentation audit of
[0011-MADR-docs-accuracy-after-v0-5-0.md](0011-MADR-docs-accuracy-after-v0-5-0.md),
carried out by [0011-PLAN-docs-accuracy-after-v0-5-0.md](0011-PLAN-docs-accuracy-after-v0-5-0.md), finding F32. Each fact was re-checked against the tree
on 2026-10-07. The design stands. What changes is the facts it rests on.

**Found.**

1. **The width method's source.** The Context credits the width method,
   "the 2027 alignment", to
   [0002-PLAN-harden-workspace-v0-1-1.md](0002-PLAN-harden-workspace-v0-1-1.md).
   * That PLAN lists the width method as out of scope (its "Out of
     scope").
   * It came with
     [0004-MADR-integrate-charm-v2-and-go-1-27.md](0004-MADR-integrate-charm-v2-and-go-1-27.md)
     §3: the workspace's `ansi.Method`, WcWidth until the terminal reports
     mode 2027, then GraphemeWidth.
2. **§4–§6 predate 0006's amendments A4 to A6.** §5's command provider,
   §4's `Hit` and §6's argument prompts were written before
   [0006-MADR-command-registry.md](0006-MADR-command-registry.md) A4 to A6:
   * **A4.**
     * `Dispatch` and `Run` refuse a zero `Origin`.
     * A command is offered by surface, and the palette's are
       `command.OriginPalette` and `command.SurfacePalette`.
     * `Danger`'s zero value means "not declared".
   * **A5.**
     * A secret property is `"writeOnly": true`.
     * `arg`, `short`, `placeholder`, `group` and `hidden` live in one
       `"x-cli"` annotation.
   * **A6.**
     * MCP prompts load as commands of kind `Prompt`, and ACP commands as
       kind `Forward`.
     * Running one sends `command.PromptMsg` for the program to pass to
       its agent.

**Corrected.**

1. **Widths.** Spans, `Cells` and highlights measure with the workspace's
   method, as 0004-MADR §3 defines it.
2. **The command provider** lists
   `Registry.Available(ctx, command.SurfacePalette)`. Its `Hit.Run`
   dispatches with `Origin: command.OriginPalette`.
3. **`Hit.Danger`** is the command's. A hit that is not a command, such as
   a file or a pane, leaves it zero, and the palette draws no danger for
   it.
4. **Argument prompts:**
   * a `writeOnly` property is entered masked, and is never kept in the
     recents;
   * `x-cli.placeholder` is the prompt's placeholder;
   * `x-cli.hidden` skips the property.
5. **`Prompt` and `Forward` commands are listed** like any other. When
   one runs, the palette passes its `PromptMsg` through to the program.

### A2 (2026-10-09): names from the glossary

Names only; the design stands. Before this record is built, its types take
the names [docs/glossary.md](../glossary.md) gives, so that none collides
with a type the library exports or another record plans
([0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md) Step 2, with its D1). Where this record says the old name, read the
new one.

| Was | Is | Why |
| :--- | :--- | :--- |
| `fuzzy.New(query, opts...)` | `fuzzy.NewMatcher(query, opts...)`, its options named `With…` | `New` builds only the type named after its package |
| `Tier` | `Grade` | `glyph.Tier` |
| `Span`, `Hit.Spans` | `Range`, `Hit.Ranges` | `layout.Span` |
| `palette.Query`, its `Context` field | `Input`, its `WhenContext` field | `termcap.Query`; a `when.Context` is a `WhenContext` |
| `palette.Scope` | `Availability` | `command.Scope` |
| `Index.With` | `Index.Update` | `With…` names an option |
| `Palette`, `palette.New` | `Picker`, `palette.NewPicker` | `theme.Palette` |
| `Walker`, `PathFilter` | each with a `…Func` adapter, `WalkerFunc` and `PathFilterFunc` | AGENTS.md's API rule for hooks. `Provider` has two methods, and `Discoverer`, `Starter`, `Stopper`, `Prefixed` and `Debouncer` are optional capabilities found by type assertion, so none takes an adapter |

[0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) takes these
names when it is next amended, before it runs.

### A3 (2026-10-11): the palette on the built keymap, registry and API conventions

*Status: accepted (2026-10-11),* the owner's answers to Q5–Q7, each the
recommendation. Written before execution, at the owner's
request of 2026-10-11 to "evaluate, research, and determine the data for
ammending 0008 and preparing the plan". Since this record was accepted:

* 0006 and 0007 shipped: `keymap` in `v0.11.0`, and the workspace's keys
  wholly on it in `v0.12.0`
  ([0007-PLAN-keymap-engine.md](0007-PLAN-keymap-engine.md));
* AGENTS.md gained its API conventions
  ([0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md)
  W0, `v0.8.0` to `v0.10.0`);
* the toolchain moved to Go 1.27.2
  ([0016-MADR-go-1-27-2-for-go-2026-6604.md](0016-MADR-go-1-27-2-for-go-2026-6604.md)).

The design stands: `fuzzy` and `palette`, asynchronous scoped providers,
grapheme-safe highlights, no new module. What changes is how a picker takes
its keys, runs what it selects, closes, and themes itself, and the names
and shapes the conventions require. Items 1 to 3 add to `workspace`, and
item 8 to `glyph`; every addition is compatible. Q5 to Q7 below are the
owner's, and each item they decide gives the recommendation.

#### Facts re-read (2026-10-11, on `a0537d3`)

| Fact | Where |
| :--- | :--- |
| Go 1.27.2 compiles a generic method on a concrete type, such as `(*Matcher).Rank[T]`; `go vet` and golangci-lint 2.14.0 under `.golangci.yml` accept it, and apidiff reports a change to one | a scratch probe outside the repository |
| The workspace matches every key first, along `KeyPath`. It runs the commands `Commands(w)` gives itself; any other matched command goes back as `RequestMsg` | `workspace/bindings.go:194-249` |
| Under a modal overlay, `KeyPath` is global, `overlay`, `overlay:<id>`, that overlay's `KeyContexts`, then the modes. A non-modal overlay adds `overlay` and `overlay:<id>`, never its `KeyContexts` | `workspace/bindings.go:132-154` |
| An unmatched key goes to the modal overlay, or else the focused pane | `workspace/bindings.go:218-225` |
| `esc` in the `overlay` scope closes the top overlay, unless that overlay is an `EscConsumer` that consumes it | `workspace/bindings.go:231-236` |
| Bubble Tea runs each returned command in a goroutine of its own, so their messages can arrive in any order | bubbletea v2.0.10 `tea.go:708-747` |
| **Measured:** 2000 key presses arriving together, each returning a command whose message carries its number, as the workspace returns a `RequestMsg`: 94 to 169 messages reached `Update` after a later one, in five runs | a scratch program on the macOS host |
| A pane can read `(*Workspace).WidthMethod()`. It cannot read the theme, which the workspace rebuilds on `tea.ColorProfileMsg` and `tea.BackgroundColorMsg`, or the keymap, which `SetBindings` replaces | `go doc ./workspace`; `workspace/workspace.go:418-436` |
| An overlay cannot close itself: `updateOverlay(i)` writes `w.overlays[i]` after the pane's `Update` returns, so a `Pop` inside it removes the slot being written | `workspace/overlay.go:155-162` |
| `Registry.Dispatch` runs from the program's `Update`, on the calling goroutine, and runs a `Loop` command at once. A `Prompt` or `Forward` command also sends `PromptMsg` | `go doc ./command Registry.Dispatch` |
| `command.Params(c)` gives a command's top-level arguments: `Type`, `Enum`, `Required`, `HasDefault` and `Default`, `Description`, `Placeholder`, `Hidden`, `Secret`, `Min` and `Max`, `Positional` and `Rest` | `go doc ./command Param` |
| `keymap.Binding` has `When`. `Keymap` has `Shortcut(id, path, goos, glyphs)`, `KeyBinding`, `Help` and `Reachable` | `go doc ./keymap` |
| bubbles `textinput.Model` has pointer `Focus` and `Blur`, value `Update`, `Cursor() *tea.Cursor`, and draws `Prompt` (`"> "`) and `EchoCharacter` (`'*'`) by default | bubbles v2.2.1 |
| `glyph.Set` has `Focus`, `Ellipsis`, the scroll glyphs, `Bullet`, `BadgeOpen`, `BadgeClose` and `Keys`; no prompt or mask glyph | `go doc ./glyph Set` |
| `theme.Styles` has `Title`, `Body`, `Muted`, `Accent`, `Success`, `Warning`, `Error`, `Badge` and the border and title roles | `go doc ./theme Styles` |
| `internal/sanitize` has `Line`, `Styled` and `Truncate`; the workspace passes pane titles and badges through them | `workspace/render.go:174-299` |
| `internal/conformance` checks the enums' text, the shared type names, opaque options, the stability line, and that nothing reads the environment, exits or starts a process | AGENTS.md, "Pre-add checks" |
| `make fuzz` finds every fuzz target itself | `scripts/go-fuzz.py -a` |
| `runtime/pprof`'s `goroutineleak` profile reports only goroutines blocked on a primitive nothing else can reach. `testing/synctest.Test` waits for every goroutine of its bubble, and fails when those left are durably blocked | Go 1.27.2 documentation |
| `command` exports `RegistryOption`; none of the other names below is exported by another public package, `Option` aside, which `sharedNames` allows everywhere | a scan of every public package's types |

#### What changes

1. **A picker's own keys run in the picker, in the order they were
   pressed** (§1 "No direct key handling", §7 "Keys"; Q5).
   * **Found.** §1 and §7 give the palette a `DefaultKeyMap()` and a
     `palette` context. 0007 removed that style in `v0.12.0`, and A11 has
     the workspace match first. A key bound to a command the workspace does
     not own reaches the program as `RequestMsg`, through a command, and
     the measurement above shows such messages reordering: with the
     palette's moves as `RequestMsg`, a "down" and an "enter" read together
     could run the row that was selected before the move.
   * **Decided.**
     * `workspace.CommandConsumer`, beside `EscConsumer`:
       `interface{ ConsumesCommand(id command.ID) bool }`.
     * `workspace.CommandMsg{Request command.Request}`.
     * When a matched command is not the workspace's own, the workspace
       offers it before making a `RequestMsg`: to the modal overlay alone
       when one is open; otherwise to the open overlays, top first, then
       the focused pane. The first whose `ConsumesCommand(id)` is true is
       sent `CommandMsg` at once, in `Update`, as an unmatched key is. The
       request is the one `RequestMsg` would carry: `Origin:
       command.OriginKey` and the workspace's `WhenContext`. With no taker,
       `RequestMsg` as today. This extends 0007 A11 item 1, where the
       workspace runs its own commands without a registry, to the
       components it hosts.
     * A `Picker` is a `KeyContexter` naming `kind:picker`, and consumes
       its navigation commands: `palette.select.next`,
       `palette.select.prev`, `palette.page.next`, `palette.page.prev`,
       `palette.accept` and `palette.complete`.
     * `palette.DefaultRules() []keymap.Default`:
       * global: `ctrl+p` to `palette.open`, `when: "!workspace.modal"`
         (Q3);
       * `kind:picker`: `up` and `ctrl+p` to `palette.select.prev`, `down`
         and `ctrl+n` to `palette.select.next`, `pgup` and `pgdown` to the
         pages, `enter` to `palette.accept`, `tab` to `palette.complete`;
       * `overlay:completion` (item 6): `up` and `ctrl+p`, `down` and
         `ctrl+n`, and `tab` and `enter` to `palette.accept`.
       * `esc` stays the workspace's: the `Picker` is an `EscConsumer` while
         its query is not empty (§7).
     * The program builds one keymap with `workspace.DefaultRules()` and
       `palette.DefaultRules()` among its defaults, so one user file rebinds
       both. Without the palette's rules, its keys reach the input as
       text, and nothing moves the selection; the guide says so, and a test
       pins it.
     * `(*Picker).OpenCommand(id command.ID, title string, opts
       ...command.Option) command.Command` gives a `Loop`, `UI` command that
       opens that picker. The program registers it; `palette.open` is the
       ID `DefaultRules` binds. Since no picker is open to consume it, its
       key comes as `RequestMsg`, and the program's `Dispatch` runs it.
   * **Not chosen:**
     * the navigation as registry commands, through `RequestMsg`: it
       reorders, as above, and every picker would need a registry;
     * a keymap of the picker's own, resolving the raw keys it is given: a
       program's global binding of `ctrl+n` would reach past it, and the
       user's file would need a second build to reach it;
     * `workspace.WithRegistry`, which A11 refused.
   * **Found, not decided here.** Any program command bound to a key
     reaches the program as `RequestMsg`, and can reorder the same way.
     That is 0007's routing, and needs a record of its own.
2. **Theme, width and keymap are read from the workspace** (§4, §7; Q6).
   * **Found.** The workspace rebuilds its theme when the terminal reports
     its profile or background, and replaces its keymap through
     `SetBindings`. A `WithTheme` or `WithBindings` option on the palette
     would go stale at either.
   * **Decided.** `(*Workspace).Theme() theme.Theme` and
     `(*Workspace).Bindings() *keymap.Keymap`, beside `WidthMethod()`.
     `palette.NewPicker(w *workspace.Workspace, opts ...Option)` keeps `w`,
     and its `View` reads the theme, the width method and the keymap there.
     §4's `WithTheme`, `WithGlyphs` and `WithKeyMap` are dropped.
   * **Not chosen:** a message the workspace broadcasts on each change,
     which every pane would receive.
3. **An overlay closes itself with `workspace.Close(id string) tea.Msg`**
   (Q7), beside `To`. `Update` closes overlay `id` if it is open, with
   `Pop`'s focus messages, and does nothing otherwise. A `Picker` that
   accepts a hit returns a command that sends it, and takes no more input
   while it waits.
   * **Not chosen:** `SelectedMsg`, with the program popping the overlay,
     which every program would repeat.
4. **A hit runs on the event loop** (§4 `Hit.Run`, A1 item 2).
   * **Found.** A1 has a command hit dispatch with `OriginPalette`.
     `Dispatch` must run from `Update`, and §4's `Run tea.Cmd` runs on
     another goroutine.
   * **Decided.** `Hit.Run` is `func(args jsontext.Value) tea.Cmd`. The
     `Picker` calls it in its `Update` on accept, with the arguments item 5
     collects, or `nil`. `FromRegistry`'s calls `reg.Dispatch` with
     `Origin: command.OriginPalette` and the `WhenContext` captured at
     open; the registry's gate, `PromptMsg` (A1 item 5) and `ResultMsg`
     follow from it. A `Run` that blocks blocks the UI, and its
     documentation says so.
5. **Arguments come from `command.Params`** (§6, A1 item 4).
   * `Hit.Args json.RawMessage` becomes `Hit.Params []command.Param`.
     `FromRegistry` fills it from `command.Params`.
   * The argument stage prompts, in order, for a `string`, `integer`,
     `number` or `boolean` parameter, and for any with `Enum`, through the
     same fuzzy list. It applies `Required`, `Default`, `Min` and `Max`,
     shows `Description` and `Placeholder`, skips `Hidden`, and masks
     `Secret` with the `Mask` glyph.
   * Any other parameter, or an error from `Params`, falls through: the
     `Picker` sends `SelectedMsg` with no arguments, and runs nothing.
   * The collected values are one JSON object, encoded with
     `encoding/json/v2`. The recents store keeps hit IDs only, so no value
     is ever kept, secret or not.
6. **Completion mode is an option, `WithoutInput()`** (§5).
   * `AsCompletion()` is not a `With…`, `Without…` or `On…` name.
     `WithoutInput()` turns off the input row; such a picker opens
     non-modal, with the overlay ID `completion` unless `WithID` gives
     another.
   * A non-modal overlay's `KeyContexts` are not on the path (0007 A11
     item 3), so its bindings are in `overlay:completion`, by ID. A program
     that picks another ID binds its own.
   * The editor sends `palette.SetQuery(q string) tea.Msg` to it with
     `SendOverlay` or `To`. On accept the `Picker` sends `SelectedMsg` and
     closes with item 3.
7. **The API conventions** (AGENTS.md; A2's names apply first).
   * **Options** are opaque:
     * `fuzzy.Option`: `WithCase(CaseMode)`, `WithNormalizer(Normalizer)`,
       `WithDelimiters(string)`, `WithExtended()`, `WithMaxAlign(int)`;
     * `fuzzy.IndexOption`: `WithMinOverlap(float64)`,
       `WithMaxCandidates(int)`;
     * `palette.Option`: `WithGOOS(string)`, `WithRecents(*Frecency)`,
       `WithLimit(int)`, `WithGrouping(Grouping)`, `WithPlaceholder(string)`,
       `WithLogger(slog.Handler)`, `WithoutInput()`;
     * `palette.OverlayOption`: `WithID(string)`,
       `WithAnchor(workspace.Anchor)`, `WithSize(width, height int)`;
     * `palette.FrecencyOption`: `WithHalfLife(time.Duration)`,
       `WithMaxEntries(int)`;
     * `palette.FileOption`: `WithIndexAbove(int)` (default 5000) and
       `WithMaxFiles(int)`; the `PathFilter` goes to `FSWalker`, as §5
       has it.
   * **Enums** through `internal/enum`, in the conformance table:
     `fuzzy.Grade` (`exact`, `prefix`, `boundary`, `subsequence`),
     `fuzzy.CaseMode` (`smart`, `insensitive`, `sensitive`) and
     `palette.Grouping` (`score`, `provider`).
   * **Hooks:** §2's `Normalize(func(rune) rune)` becomes `Normalizer`,
     `interface{ Normalize(r rune) rune }`, with `NormalizerFunc`.
     `Walker` and `PathFilter` take their adapters (A2).
   * **Errors:** `palette.ErrRecentsVersion`, wrapped with the version, as
     `layout.ErrStateVersion` is.
   * **No clock option.** §6's injectable clock is dropped: `Frecency`
     reads `time.Now`, and its tests run inside `testing/synctest`, whose
     clock is fake.
   * **Names:** §5's `Commands(reg)` provider is `FromRegistry(reg)`, so
     `Commands` keeps meaning a package's own command list, as
     `workspace.Commands` does; and no type is `RegistryOption`, which
     `command` exports. `Hit` gains `Command command.ID`, empty for a hit
     that is not a command.
   * **Each package** ends its documentation with its stability line,
     reads no environment, and has no mutable package variable.
8. **Glyphs.** `glyph.Set` gains `Prompt` (`❯`, ASCII `>`) and `Mask` (`•`,
   ASCII `*`), each one cell in both width methods, in every tier. The
   `Picker` sets `textinput`'s `Prompt` and `EchoCharacter` from them, so
   no default bubbles glyph is drawn (AGENTS.md, TUI rule 3).
9. **Text from outside is sanitised.** A hit's `Text`, `Group`, `Help`,
   `Hint` and `Disabled`, and a provider's name, go through
   `sanitize.Line` before they are matched for drawing, measured or styled,
   so a file name or an MCP title with a control sequence or a newline
   draws as one row of its visible text. A failing provider's row names it;
   its error goes to the `slog.Handler` only.
10. **Limits.**
    * A query is matched on its first 256 graphemes; the input takes no
      more.
    * The `Picker` keeps the best `WithLimit` hits (default 200) across
      providers.
    * `Files` stops its walk at `WithMaxFiles` (default 200 000) and shows
      one muted row saying so.
    * `Frecency.ReadFrom` reads at most 1 MiB, and refuses an ID longer
      than 1024 bytes.
11. **Hints come from the keymap.** For each drawn row whose `Hit.Command`
    is set and whose `Hint` is empty, the `Picker` asks
    `Bindings().Shortcut(id, path, goos, glyphs)`, with the key path
    captured when it opened, so the hint is the key that works where the
    user was.
12. **The input is held directly.** The `Picker` holds a
    `textinput.Model`, not a `workspace.Model[M]`: it is one part of a
    pane that also draws results. It focuses and blurs the input on
    `PaneFocusMsg` and `PaneBlurMsg`, and is a `Cursorer` that offsets the
    input's cursor into the overlay.
13. **The leak check** (Confirmation). A cancelled search, and a closed
    picker, leave no goroutine: `testing/synctest.Test` fails otherwise,
    and the `goroutineleak` profile, taken at the bubble's end, reports
    none.
14. **Release.** `fuzzy`, `palette`, and the `workspace` and `glyph`
    additions ship together as `v0.13.0`. Every change is an addition, so
    `scripts/apicheck.allow` stays empty.

#### Owner questions (A3)

*Answered 2026-10-11:* "Q5 follow recommendation. Q6 follow
recommendation. Q7 follow recommendation." Every answer is the
recommendation, so items 1 to 3 stand as written.

* **Q5. How a picker takes its keys.** Recommended: item 1, the workspace
  offering a matched command to the component that consumes it. The
  alternatives are item 1's "Not chosen".
* **Q6. How a picker gets the theme, width and keymap.** Recommended: item
  2, accessors on the workspace. The alternative is a broadcast message.
* **Q7. How a picker closes.** Recommended: item 3, `workspace.Close`. The
  alternative is that the program closes it on `SelectedMsg`.

[0008-PLAN-command-palette.md](0008-PLAN-command-palette.md), revised on
2026-10-11, is written against these answers.


### A4 (2026-10-12): x/ansi's `WcWidth.Cut`, and where highlights are cut

*Status: item 1 accepted (2026-10-12), the owner's choice; item 2 open,
decided before
[0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) Step 6.*
Found in Step 3 (its D1). The Context's "`Method` has `StringWidth` and
`Cut`" holds for the names, not for `Cut`'s result under `WcWidth`.

**Found.**

1. **`ansi.WcWidth.Cut` ignores its left edge.** In x/ansi v0.11.8
   `truncate.go:35-45`, `cut` sets `truncateLeft = TruncateWc` where
   `TruncateLeftWc` is meant, so `WcWidth.Cut("abcdef", 2, 4)` is `"ab"`.
   v0.11.9, the newest release on 2026-10-12, has the same code.
   `GraphemeWidth.Cut`, and the package's `Cut`, are right. ~~x/ansi's
   `Method.Truncate` and `Method.TruncateLeft` are right under both
   methods: their composition cuts every cell range `fuzzy.Cells` gives
   back to its text, in a probe outside the repository.~~ *Corrected
   later on 2026-10-12:* under `WcWidth` they disagree too.
   `WcWidth.Truncate` counts a ZWJ family as 2 cells, and `StringWidth`
   and `TruncateLeft` count it as 6: on `"ab👨‍👩‍👧cd"`, `Truncate(s, 6)`
   keeps all 10 cells. The repository met this before:
   `internal/sanitize`'s `Truncate` cuts rune by rune under `WcWidth`, for
   it ([0014-PLAN-hardening.md](0014-PLAN-hardening.md) Step 9, D8).
2. **`lipgloss.StyleRanges` always cuts in grapheme width.** lipgloss
   v2.0.6 `ranges.go:25-33` calls the package-level `ansi.Cut` and
   `ansi.TruncateLeft`. §7's highlights "use `fuzzy.Cells` and
   `lipgloss.StyleRanges`, in the terminal's width method", which cannot
   hold while the workspace measures with `WcWidth`, its method until the
   terminal reports mode 2027: a cluster whose widths differ, such as a
   ZWJ family (6 cells under `WcWidth`, 2 under `GraphemeWidth`), shifts
   every highlight after it.

**Decided.**

1. **Step 3's oracle** (the owner, 2026-10-12): `fuzzy.Cells` is checked
   against `ansi.Cut` under `GraphemeWidth`, as planned, and under
   `WcWidth` ~~against `m.TruncateLeft(m.Truncate(s, end, ""), start, "")`,
   the composition `Cut` should make~~ *(the owner's second answer, later
   on 2026-10-12, once that composition failed on a ZWJ family)* against
   the repository's own cutter: the text of cells `[a, b)` is
   `sanitize.Truncate(s, b, m, "")` less its prefix
   `sanitize.Truncate(s, a, m, "")`, the cutter AGENTS.md's width rule
   gives the workspace. `fuzzy` itself uses none of them: `Cells` measures
   with `Method.StringWidth`.
2. **Where the palette cuts a highlighted row:** open. Step 6 does not
   start until it is decided, by an amendment of this item.

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
* For amendment A3: `workspace/bindings.go`, `workspace/overlay.go` and
  `workspace/workspace.go` at `a0537d3`; bubbletea v2.0.10 `tea.go`
  (`handleCommands`); bubbles v2.2.1 `textinput`; the `keymap` and
  `command` documentation; and
  [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) A11, which A3
  item 1 extends.
