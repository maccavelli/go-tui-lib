---
status: in-progress
date: 2026-10-06
associated-madr: "0006-MADR-command-registry.md"
---
# Implement the command registry (`when`, `command`, `command/cli`, and the Cobra and Kong front ends)

Associated MADR: [0006-MADR-command-registry.md](0006-MADR-command-registry.md)

## Revisions

* **2026-10-02.** The owner answered the MADR's Q1–Q4, and the MADR is
  `accepted`. Q4 added a Cobra front end beside `command/cli`.
* **Later on 2026-10-02.** MADR amendment A1 and
  [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
  moved the Cobra front end into the nested module `command/cobracmd`,
  without fang, added a Kong front end, `command/kongcmd`, and aligned the
  argument tags with Kong's.
* **2026-10-03.** The owner accepted A1 and approved
  [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md).
* **2026-10-05.** Rewritten after an audit of the tree at `v0.3.0`
  (`81a61de`), recorded as MADR amendment A2, at the owner's request
  ("assess the codebase for current facts … ensure the plan is accurate,
  updated, and is actionable, detailed, and deterministic"). What changed,
  and why:
  * the preconditions that 0004 and 0010 set are met, so the conditional
    and "blocked" wording is gone, and Steps 10 and 11 wait only for the
    `v0.4.0` tag;
  * A1's tags are the only tag vocabulary; §4's `doc` and `arg:"…"` are
    not built;
  * `workspace.resize` takes `split`, `workspace.layout.reset` is
    `SetState` of the zero state, `workspace.theme.set` calls a new
    `Workspace.SetBackground`, and `SchemaOf` supports `map[string]T`
    (A2, Q5);
  * the exporters target MCP 2026-07-28 (A2, Q6);
  * the registry's own commands move from Step 3 to Step 4, because
    `command.describe` takes an argument and `New[A]` arrives in Step 4;
  * each fuzz target joins `make fuzz` in the step that adds it, so CI
    fuzzes it from that step on;
  * every step now lists its files, its exported names, its tests by name,
    its golden files, its mutations, its commands and its done criteria;
  * a Step 12 closes the PLAN after both front ends, which the old Step 9
    did by reference.

  The text this revision replaces is in git at `4c6261e`.
* **2026-10-06.** MADR amendment A12 (D47, found while writing Step 11):
  Step 11a, a root change and the root release `v0.5.0`, comes before
  Step 11's module, which then requires `v0.5.0`. The Goal, the Scope
  table and Verification name the new release.

## Goal

Ship `when`, `command` and `command/cli` in the root module's `v0.4.0`,
with the workspace's built-in commands, so that one definition per action
serves keys, the palette, slash commands, help, the shell and agents; then
ship the Cobra and Kong front ends as the nested modules
`command/cobracmd` and `command/kongcmd`, each at `v0.1.0` (MADR §1–§11,
A1, A2), with the root's `v0.5.0` between them, in which `New[A]` refuses
an argument struct Kong would read differently (A12).

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner has tagged `v0.4.0`, `command/cobracmd/v0.1.0`,
`v0.5.0` and `command/kongcmd/v0.1.0`.

## Scope

### Facts this PLAN starts from (2026-10-05)

| Fact | Where it was read |
| :--- | :--- |
| Go 1.27.1; `encoding/json/v2` builds with no `GOEXPERIMENT` | `go version`; a scratch build (MADR A2, item 1) |
| the root module is at `v0.3.0`; its `go.mod` is unchanged since `v0.2.0` | `git tag`; `go.mod` |
| 0002, 0004, 0005 and 0010 are `complete` | `docs/README.md` |
| `go.work` lists `.` only; `scripts/go-modules.sh` lists the modules for CI's matrix | `go.work`; `.github/workflows/ci.yml` |
| `.golangci.yml` confines Cobra and pflag to `command/cobracmd/**`, Kong to `command/kongcmd/**` | `.golangci.yml`, the `cobra` and `kong` depguard rules |
| `internal/conformance` scans every package of every module | `internal/conformance/conformance_test.go` |
| `make fuzz` runs `scripts/go-fuzz.sh` once per package, one line each | `Makefile`, target `fuzz` |
| golden files are `tuitest.Text` or `tuitest.Golden`, rewritten with `-tuitest.update` | `tuitest/tuitest.go` |
| the workspace methods the built-ins wrap, and their signatures | `go doc ./workspace` (MADR A2, item 2) |

### In scope

| Step | Paths | Delivers | Released in |
| :--- | :--- | :--- | :--- |
| 1 | `docs/decisions/0006-*`, `docs/README.md` | the records (this revision) | — |
| 2 | `when/`, `Makefile` | the expression language | `v0.4.0` |
| 3 | `command/` | the command, the registry, dispatch, the gate, the audit trail | `v0.4.0` |
| 4 | `command/`, `Makefile` | arguments: `New[A]`, `SchemaOf`, strict decoding, slash parsing; the registry's own commands | `v0.4.0` |
| 5 | `command/` | sources: command files, MCP prompts, ACP commands, clashes | `v0.4.0` |
| 6 | `command/` | the MCP and ACP exporters and the manifest | `v0.4.0` |
| 7 | `workspace/` | `Commands`, `WhenContext`, `Contexter`, the context keys, `SetBackground` | `v0.4.0` |
| 8 | `command/cli/` | the standard-library front end | `v0.4.0` |
| 9 | `README.md`, `docs/` | documentation; the root release | `v0.4.0` |
| 10 | `command/cobracmd/`, `go.work`, docs | the Cobra front end, a nested module | `command/cobracmd/v0.1.0` |
| 11a | `command/`, `workspace/`, the `command/cli` and `command/cobracmd` test fixtures, docs | `New[A]` agrees with Kong (A12) | `v0.5.0` |
| 11 | `command/kongcmd/`, `go.work`, docs | the Kong front end, a nested module | `command/kongcmd/v0.1.0` |
| 12 | `docs/` | close-out | — |

The root `go.mod` and `go.sum` do not change in any step: every import in
Steps 2 to 9 is the standard library or `charm.land/bubbletea/v2`, which the
root already requires. The root `go.mod` never names Cobra or Kong.

### Out of scope

* The keymap engine
  ([0007-PLAN-keymap-engine.md](0007-PLAN-keymap-engine.md)) and the
  palette ([0008-PLAN-command-palette.md](0008-PLAN-command-palette.md)).
  The workspace's `KeyMap` stays as it is until 0007 moves it.
* A permission dialog: this PLAN defines `Gate` and the default refusal.
* An MCP server or an ACP transport, the MCP list envelope (`ttlMs`,
  `cacheScope`) and `subscriptions/listen`: the host's (MADR A2, Q6).
* `terminal.*` context keys from `termcap` (MADR A2, item 5).
* YAML or TOML command files (MADR Q1), fang (0010-MADR Q1), a third-party
  Kong completion module (0010-MADR Q2), and shell completion scripts from
  `command/cli`.
* Any change in pi-go.
* `git push` and tags, which are the owner's.

## Rules for every step

1. **Order.** A step starts when the previous one is committed. Its
   packages compile, its tests pass, and its checks are clean before it
   ends.
2. **Exported API.** A step adds exactly the exported names it lists. A
   name it needs and does not list stops the step for the owner, and is
   recorded as a deviation.
3. **Checks,** with their output in the execution record:
   1. `gofmt -l` on the step's Go files prints nothing;
   2. `make pre-add-check FILES="<the step's Go files>"`;
   3. `make lint`, which runs `make modernize` and golangci-lint for
      `GOOS=linux`, `darwin` and `windows`;
   4. `GOWORK=off go test -race -count=1 ./...`;
   5. `GOWORK=off go test -shuffle=on -count=2 ./...`;
   6. `LC_ALL=C GOWORK=off go test -count=1 ./...`;
   7. `GOWORK=off go mod tidy -diff`;
   8. `make fuzz FUZZTIME=20s` when the step adds a fuzz target;
   9. on the Windows test host, on a copy of the tree: `make
      pre-add-check`, `make lint`, `make vuln`, and the step's packages'
      tests with `-shuffle=on -count=3`;
   10. `markdownlint-cli2 --config .markdownlint-cli2.jsonc` and a
       relative-link check on the Markdown the step changes;
   11. the identifier scan of the diff: no hostname, account name or
       real-machine path.
4. **Mutation proofs.** Each listed mutation is applied to a scratch copy
   of the tree, never the tree itself, and the named test must fail; the
   failing line goes in the execution record. A mutation that survives, or
   does not compile, stops the step: the test is strengthened, or the
   mutation replaced, and either is recorded. After a step changes code
   that an earlier step's mutation anchors on, the earlier steps' sets run
   again.
5. **Golden files** are written with `-tuitest.update`, and every one is
   read before the step ends.
6. **Conventions** (0001-MADR §6, AGENTS.md). Nothing writes to
   `os.Stdout` or `os.Stderr`, reads the process environment, starts a
   process, sets the alternate screen or installs a signal handler; every
   glyph comes from `glyph`; output goes to the writers a caller gives.
   `internal/conformance` covers each new package with no change to it.
7. **Deviations.** Anything the PLAN does not say, or that contradicts a
   fact here, stops the step: the agent gives the evidence and the
   options, the owner picks, and the choice is recorded in this PLAN, and
   in the MADR when a decision or a fact changes, before work continues.
8. **Commits.** One commit per step, made by the owner with `git commit
   --no-edit`, unless the owner asks the agent to commit in that turn. The
   agent stages nothing otherwise.

## Implementation Steps

### Step 1: records

* MADR amendment A2 is accepted, and this revision is written. When the
  owner approves execution, this PLAN is set `in-progress`, with
  `docs/README.md`'s row, at the start of Step 2.
* **Done when:** the owner approves execution.

### Step 2: `when`

**Files:** `when/when.go` (package documentation, `Expr`, `Parse`,
`MustParse`, `Check`), `when/lex.go`, `when/parse.go`, `when/eval.go`,
`when/value.go`, `when/key.go`; tests `when/when_test.go`,
`when/fuzz_test.go`, `when/bench_test.go`; `Makefile`. (*D1: and the
regression input `when/testdata/fuzz/FuzzParse/01e90e822ccb8b53`. D3: and
`.github/workflows/ci.yml`.*)

**Exported names** (MADR §8):

```go
func Parse(src string) (Expr, error)
func MustParse(src string) Expr
type Expr struct{ /* compiled */ }
func (e Expr) Eval(c Context) bool
func (e Expr) String() string
func (e Expr) Keys() iter.Seq[string]
func Check(e Expr, known Keys) []error

type Context interface{ Value(key string) (Value, bool) }
type Kind uint8                // KindBool, KindNumber, KindString, KindList
type Value struct{ /* one of the four kinds */ }
func BoolValue(b bool) Value
func NumberValue(f float64) Value
func StringValue(s string) Value
func ListValue(l []string) Value
func (v Value) Kind() Kind
func (v Value) Bool() bool       // the truth of a key alone (§8)
func (v Value) Number() (float64, bool)
func (v Value) String() string   // canonical text
func (v Value) List() ([]string, bool)
type Map map[string]Value
func (m Map) Value(key string) (Value, bool)
func Layered(cs ...Context) Context
type Keys map[string]Kind
type Key[T bool | int | float64 | string | []string] struct{ Name, Doc string }
func NewKey[T bool | int | float64 | string | []string](name, doc string) Key[T]
func (k Key[T]) Set(m Map, v T)
func (k Key[T]) Get(c Context) (T, bool)
func (k Key[T]) Kind() Kind
```

An `int` is held as `KindNumber`. `MaxSource = 4096` and `MaxDepth = 64`
are exported constants.

**Build:** a hand-written lexer, and a recursive-descent parser of the
MADR's grammar with its precedence. A regex compiles with `regexp` at parse
time. `String` writes the canonical form: single spaces around binary
operators, parentheses only where precedence needs them, strings in single
quotes.

**Tests** (`when/when_test.go`):

| Test | Shows |
| :--- | :--- |
| `TestOperators` | every operator of §8, alone, on a `Map` |
| `TestPrecedence` | `!foo && bar` is `(!foo) && bar`; `foo \|\| bar && baz` is `foo \|\| (bar && baz)` |
| `TestUnsetKeys` | an unset key alone is false; it is unequal to every value, so `x != 'a'` is true |
| `TestIn` | `in` and `not in` over a list key, and over an unset key |
| `TestTerms` | quoted strings with spaces, barewords, integers, decimals, `true`, `false` |
| `TestNoSpacesNeeded` | `a==1&&b!=2` parses as `a == 1 && b != 2` |
| `TestLayered` | the first context that holds a key wins |
| `TestCheck` | an unknown key, and a number compared with a string, are reported |
| `TestLimits` | 4097 bytes of source, and 65 nested parentheses, are errors, not panics (*D1: and a source within 4096 bytes whose canonical form is longer*) |
| `TestRegexIsRE2` | `=~ /(a+)+$/` evaluates in linear time on a long input; a bad regex is a parse error |
| `TestStringRoundTrip` | `Parse(e.String()).String() == e.String()` over the operator table (*D2: and the two evaluate alike on every assignment of `true` and `false` to the keys*) |
| `TestTypedKeys` | `Key[T].Set` and `Get` for each `T`; `Get` of the wrong kind is false |

`FuzzParse` (`when/fuzz_test.go`): no panic; for every parsed input,
`Parse(e.String())` succeeds and gives the same `String()` (*D1: true
once `Parse` bounds the canonical form, MADR A3*); `Eval` on a
`Map` built from the fuzz input does not panic (*D2: and gives the same
result for the expression and its canonical form*). Seeds: the operator
table. Benchmarks `BenchmarkParse` and `BenchmarkEval` on a five-clause
expression.

**Makefile:** the `fuzz` target gains
`@./scripts/go-fuzz.sh -t $(FUZZTIME) -m 1 ./when`, and its help text
names `when`.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `&&` binds looser than `\|\|` | `TestPrecedence` |
| an unset key compares equal to `''` | `TestUnsetKeys` |
| the depth limit is not checked | `TestLimits` |
| `Layered` takes the last match | `TestLayered` |
| `String` drops needed parentheses | `TestStringRoundTrip` |
| `not in` is evaluated as `in` | `TestIn` |
| *D1:* the canonical form's length is not checked | `TestLimits` |

**Done when:** the checks of Rule 3 are clean, `make fuzz` covers `when`,
and every mutation is killed.

### Step 3: `command` core

**Files:** `command/command.go` (package documentation, `ID`, `Command`,
the enumerations), `command/handler.go`, `command/registry.go`,
`command/dispatch.go`, `command/gate.go`, `command/audit.go`,
`command/messages.go`; tests `command/registry_test.go`,
`command/dispatch_test.go`, `command/gate_test.go`,
`command/bench_test.go`.

**Exported names** (MADR §2, §3, §5, §7):

* `ID` with `Valid() error` and `Segments() iter.Seq[string]`;
  `Command` with the fields of §2; `Schema = json.RawMessage`.
* `Kind` (`Action`, `Prompt`, `Forward`), `Danger` (`ReadOnly`, `UI`,
  `Mutating`, `Destructive`), `Surface` (`SurfaceKey`, `SurfacePalette`,
  `SurfaceSlash`, `SurfaceCLI`, `SurfaceAgent`, and `AllSurfaces`), `Mode`
  (`Loop`, `Async`), `Scope` (a string; `Global` is `""`), `SourceKind`
  (`Builtin`, `User`, `Project`, `MCP`, `ACP`, `Plugin`), `Source`,
  `Origin` (`OriginKey`, `OriginMouse`, `OriginPalette`, `OriginSlash`,
  `OriginCLI`, `OriginAgent`, `OriginProgram`), each with `String`.
* `Handler`, `HandlerFunc`, `Invocation`, `Result`, `Request`.
* `Registry`, `NewRegistry`, `RegistryOption`, `WithGate`, `WithAuditor`,
  `WithPrefixer`, and the methods `Register`, `Remove`, `Lookup`, `Slash`,
  `All`, `Available`, `Version`, `Watch`, `Dispatch`, `Run`, `Cancel`,
  `CancelAll`. (`ReplaceSource` is Step 5's; `ParseSlash` Step 4's.)
* *D5:* `ErrUnknown`, `ErrUnavailable` and `ErrRefused`.
* `Decision` (`AllowOnce`, `AllowAlways`, `RejectOnce`, `RejectAlways`)
  with `ACPKind() string`; `Gate`; `Auditor`; `Record`; `SlogAuditor`.
* The messages `ResultMsg`, `PromptMsg`, `ChangedMsg`, `ConflictMsg`,
  `QuitRequestMsg`; `Conflict`.

**Build:**

* The snapshot is an immutable slice sorted by ID and a map from ID and
  from slash name, behind an `atomic.Pointer`. A write takes a mutex,
  copies, bumps `Version`, publishes, and closes the channel `Watch`
  waits on.
* `Dispatch` checks `When` and `Surfaces` for the request's origin and
  context, asks the policy (§7), audits, and then runs a `Loop` command at
  once, returning its `Result.Cmd` batched with a `ResultMsg`, or returns
  a `tea.Cmd` that runs an `Async` command under a context `Cancel`
  reaches. `Run` does the same synchronously, for any mode.
* *D4:* `Danger` and `Origin` start at 1; `Register` refuses a zero
  `Danger`, and `Dispatch` and `Run` a zero `Origin`. *D6:* `OriginMouse`
  needs `SurfaceKey`; `OriginProgram` is checked against `When` only.
* The policy (§7): `ReadOnly` and `UI` run for every origin; `Mutating`
  from `OriginAgent`, and `Destructive` from `OriginAgent` or `OriginCLI`,
  ask the gate, and with none are `RejectOnce`. `AllowAlways` and
  `RejectAlways` are remembered per command ID and caller.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestIDValid` | `workspace.focus.next` is valid; an upper-case letter, an empty segment, a leading `-` and 129 bytes are not |
| `TestRegisterDuplicate` | a second command with one ID is an error (*D4: and a command with no `Danger`*) |
| `TestAllSorted`, `TestAvailable` | `All` is in ID order; `Available` honours `When` and `Surfaces` (*D6: and `Dispatch` checks a click against `SurfaceKey`, and the program against `When` only*) |
| `TestDispatchLoop` | a `Loop` command runs during `Dispatch`; its `Result.Cmd` and a `ResultMsg` come back |
| `TestDispatchAsync` | an `Async` command runs only when the returned `tea.Cmd` runs |
| `TestRunMatchesDispatch` | `Run` and `Dispatch` give the same `Result` for one request |
| `TestCancel`, `TestExclusive` | a running async command sees its context cancelled |
| `TestPolicy` | every origin against every danger, without a gate and with one (*D4: a request with no origin is refused; D5: each failure is `ErrUnknown`, `ErrUnavailable` or `ErrRefused` under `errors.Is`*) |
| `TestAlways` | `AllowAlways` is remembered per command and caller; `RejectOnce` is not |
| `TestSlogAuditor` | one record per dispatch on the given logger, never the default logger |
| `TestSnapshotsUnderRace` | under `-race`, `All` and `Lookup` from many goroutines during writes see whole snapshots, and `Version` never falls |
| `TestWatch` | `Watch`'s command returns `ChangedMsg` after a change, not before |

Benchmarks: `BenchmarkLookup`, `BenchmarkAvailable` over 500 commands,
`BenchmarkDispatchLoop` of a no-op command.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| the default policy allows `Mutating` for agents | `TestPolicy` |
| `AllowAlways` is not remembered | `TestAlways` |
| `Version` is not bumped on `Remove` | `TestWatch` |
| `Exclusive` is ignored | `TestExclusive` |
| a write changes the published snapshot in place | `TestSnapshotsUnderRace` |
| `Dispatch` skips `When` | `TestAvailable` or `TestDispatchLoop` |

**Done when:** the checks are clean and every mutation is killed.

### Step 4: arguments, and the registry's own commands

**Files:** `command/args.go` (`New`, `Option` and its constructors,
`NoArgs`, `ArgError`), `command/schema.go` (`SchemaOf`, the tags),
`command/decode.go`, `command/slash.go` (`ParseSlash`),
`command/builtin.go`; tests `command/schema_test.go`,
`command/decode_test.go`, `command/slash_test.go`,
`command/builtin_test.go`, `command/fuzz_test.go`; goldens
`command/testdata/golden/schema-*.golden`; `Makefile`. (*D12: and
`.github/workflows/ci.yml`.*)

**Exported names** (MADR §4, A1, A2):

* `New[A any](id ID, title string, run func(context.Context, *Invocation, A) (Result, error), opts ...Option) (Command, error)`,
  `SchemaOf[A any]() (Schema, error)`, `NoArgs`, `ArgError{Path, Reason}`.
* `Option`, one constructor per `Command` field New does not take:
  `WithDescription`, `WithCategory`, `WithSlash(name string, aliases ...string)`,
  `WithArgHint`, `WithOutput(Schema)`, `WithWhen`, `WithScope`,
  `WithDanger`, `WithIdempotent`, `WithOpenWorld`, `WithSurfaces`,
  `WithMode`, `WithExclusive`, `WithWhileBusy`, `WithHidden`, `WithMeta(key string, v any)`.
* `(*Registry).ParseSlash(line string) (Request, error)`.

**The tags** are A1's only: `json`, `help`, `default`, `enum:"a,b"`,
`arg:""` (positional, in field order), `short`, `hidden`, `placeholder`,
`group`, and `schema:"min=…,max=…,minLen=…,maxLen=…,secret"`. `SchemaOf`
supports string, bool, the integer and float kinds, slices, nested structs,
pointers, `map[string]T` (A2), `time.Duration` (a string), and a type with
its own `JSONSchema() Schema`. It refuses a scalar `enum` field that is
neither required nor defaulted (A1), and any other type, with an error.

**Decoding:** `encoding/json/v2` with `json.RejectUnknownMembers(true)`,
then `type`, `required`, `enum`, `minimum`, `maximum`, `minLength`,
`maxLength`, `items` and `additionalProperties: false` are checked. An
`*ArgError` carries a JSON Pointer path.

**Slash parsing:** quotes group words; positionals fill `arg` fields in
field order; `name=value` fills any field; a command with one string
argument takes the whole tail. The result is the JSON `--args` would send.

**The registry's own commands,** registered by `NewRegistry`:
`command.list` (`NoArgs`, `ReadOnly`, every surface), `command.describe`
(`{id string}` positional, `ReadOnly`, every surface) and `app.quit`
(`NoArgs`, `UI`, no CLI surface), which only sends `QuitRequestMsg`.
(*D8: none has a slash name. D10: `command.list` lists what `Available`
gives the caller.*)

*D7:* `New` returns an error when no `WithDanger` is given. *D9:* `secret`
is `"writeOnly": true`; `short`, `placeholder`, `group` and `hidden` go in
one `"x-cli"` object, written only when one is set. *D11:* so does `"arg":
true`, and positionals follow the order of `properties`.

**Goldens** (`tuitest.Text`): `schema-flat`, `schema-nested`,
`schema-slices`, `schema-pointers`, `schema-enum`, `schema-bounds`,
`schema-duration`, `schema-map`, `schema-custom`, `schema-noargs`. Each is
valid JSON naming `$schema` 2020-12; `schema-noargs` is `{"type":
"object", "additionalProperties": false}` with `$schema`.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestSchemaGolden` | the ten goldens |
| `TestSchemaRefuses` | a channel field, and an enum neither required nor defaulted, are errors, not panics |
| `TestSchemaKeepsUnknownKeys` | a `schema` key Kong does not read, and the `json` name, survive in the schema (*D9: and `short`, `placeholder`, `group` and `hidden` survive in `x-cli`*) |
| `TestDecodeStrict` | an unknown member, a wrong type, a missing required field, a number out of range and a value outside its enum are each an `*ArgError` found with `errors.AsType`, naming its path |
| `TestSlashForms` | `/resize sidebar 4`, `/resize split=sidebar delta=4` and `/resize "sidebar" "4"` give the same JSON; a stray positional is an error |
| `TestSlashWholeTail` | a one-string command takes the whole tail |
| `TestAuditMasksSecrets` | a field tagged `schema:"secret"` is masked in every audit `Record` |
| `TestBuiltins` | `command.list` lists every command once; `command.describe` returns a command's ID, schema and danger; `app.quit` sends `QuitRequestMsg` and nothing else (*D8: none has a slash name; D10: `command.list` leaves out hidden, unavailable and other-surface commands*) |

`FuzzParseSlash`: no panic, and every accepted line decodes against its
command's schema. The `fuzz` target gains
`@./scripts/go-fuzz.sh -t $(FUZZTIME) -m 1 ./command`.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `omitzero` fields are marked required | `TestSchemaGolden` |
| unknown members are accepted | `TestDecodeStrict` |
| `maximum` is not checked | `TestDecodeStrict` |
| positionals fill fields in reverse order | `TestSlashForms` |
| `arg` presence is ignored, so a positional becomes a flag | `TestSlashForms` |
| `enum` is split on `\|` | `TestSchemaGolden` |
| a map field is refused | `TestSchemaGolden` |
| `secret` is not masked | `TestAuditMasksSecrets` |

**Done when:** the checks are clean, `make fuzz` covers `command`, every
golden is read, and every mutation is killed.

### Step 5: sources and loaders

**Files:** `command/source.go` (`ReplaceSource`, the prefixer),
`command/loaddir.go`, `command/frontmatter.go`, `command/expand.go`,
`command/mcpprompt.go`, `command/acp.go` (`ACPCommand`, `FromACP`); tests
`command/loaddir_test.go`, `command/frontmatter_test.go`,
`command/sources_test.go`; a test tree `command/testdata/commands/`; goldens
`command/testdata/golden/loaddir.golden`. (*Added in execution: the
regression input `command/testdata/fuzz/FuzzFrontMatter/0bb99b90c9197b64`.*)

**Exported names** (MADR §6): `LoadDir(fsys fs.FS, src Source)
([]Command, []error)`; `MCPPrompt`, `MCPPromptArgument`, `PromptGetter`,
`FromMCPPrompts`; `ACPCommand`, `ACPCommandInput`, `FromACP`;
`(*Registry).ReplaceSource(src Source, cmds []Command) []Conflict`.
(*D13: `PromptGetter` is `func(ctx context.Context, name string, args
map[string]string) (string, error)`. D15: `Conflict` gains `Err error`.*)

*Deviations D13–D22* (MADR A6): names that are not ID segments are
mapped to one (D14); a file's slash name is its path joined with `:`
(D16); a placeholder's property is named as written (D17) and is
positional in order of first use (D18); `Register` takes a slash name a
loaded command holds (D19); `Watch` returns `ConflictMsg` with
`ChangedMsg` (D20); an ACP forward keeps the agent's name in
`Meta["acp"]` (D21); a renamed name that is also taken is dropped (D22).

**Build:** the front matter is MADR §6's strict subset; `$NAME`
placeholders become required string arguments; `$ARGUMENTS` is the slash
tail; expansion is text substitution only; a loaded command with no
`danger` is `Mutating`; IDs are `user.…`, `project.…`, `mcp.<server>.…`,
`acp.<agent>.…`, `plugin.<name>.…`; a loaded slash name that a built-in
holds becomes `<prefix>:<name>` and a `Conflict`, also sent as
`ConflictMsg`.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestLoadDirGolden` | the commands of `testdata/commands/`, with subdirectories, front matter and placeholders |
| `TestFrontMatterErrors` | an unknown key, a duplicate key, a nested value, no closing `---`, a bad `danger` and a bad `when` each name the file and line |
| `TestNoFrontMatter` | a file without it is a prompt whose description is its first line |
| `TestRootRefusesSymlink` | through `os.Root.FS()` on a temporary directory, a symlink out of the root is refused |
| `TestExpand` | `$FOCUS` and `$ARGUMENTS` are substituted; `$lower`, `$$`, `!{ls}` and `$(ls)` stay as text |
| `TestDefaultDanger` | a file without `danger` is `Mutating` |
| `TestFromMCPPrompts` | `mcp.<server>.<name>` prompts whose schema has the prompt's arguments, `required` kept |
| `TestFromACP` | `acp.<agent>.<name>` forwards; running one sends `PromptMsg{Text: "/name args"}` |
| `TestSlashClash` | a loaded `/review` beside a built-in one is `/user:review`, with a `Conflict` and a `ConflictMsg` (*D19, D20, D22: also when the built-in comes later, through `Watch`, and a doubly taken name is dropped*) |
| `TestReplaceSource` | the old commands go in the same version the new ones arrive |

`FuzzFrontMatter`: no panic; every accepted input re-serialises to an
equivalent front matter.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| a built-in loses a slash clash | `TestSlashClash` |
| unknown front-matter keys are ignored | `TestFrontMatterErrors` |
| a command without `danger` defaults to `ReadOnly` | `TestDefaultDanger` |
| `ReplaceSource` keeps the old commands | `TestReplaceSource` |
| `!{…}` is expanded | `TestExpand` |

**Done when:** the checks are clean, the golden is read, and every
mutation is killed.

### Step 6: exporters

**Files:** `command/mcp.go`, `command/acpexport.go`,
`command/manifest.go`; tests `command/export_test.go`; goldens
`command/testdata/golden/mcp-tools.golden`, `mcp-call-*.golden`,
`acp-commands.golden`, `manifest.golden`.

**Exported names** (MADR §7, A2): `MCPTool`, `MCPToolAnnotations`,
`MCPCallResult`, `MCPContent`, `Manifest`, `ManifestCommand`; the methods
`MCPTools(ctx when.Context) []MCPTool`, `CallMCP(ctx context.Context, name
string, arguments json.RawMessage, caller string) MCPCallResult`,
`ACPCommands(ctx when.Context) []ACPCommand`, `Manifest() Manifest`.
(*D24: and `WithLoop(send func(tea.Msg)) RegistryOption` and `LoopMsg`
with `Run() tea.Cmd`. D25: `ACPCommands` exports slash commands only.
D26: `command.list` and `command.describe` return `ManifestCommand`.*)

**The target is MCP 2026-07-28** (A2, Q6). Before the first golden, the
step fetches `schema/2026-07-28/schema.ts` and records, in the execution
record, the field names of `Tool`, `ToolAnnotations` and
`CallToolResult`; the tests hold them as literal lists, and no test reads
the network.

* `MCPTool`: `name` (the ID), `title`, `description`, `inputSchema`,
  `outputSchema` (omitted when empty), `annotations`, `_meta`.
* `MCPCallResult`: `content`, `structuredContent` (omitted when empty),
  `isError`, and `resultType: "complete"`.
* Danger maps to annotations as MADR §2's table says.
* `MCPTools` and `ACPCommands` list in ID order, hidden commands and
  commands without `SurfaceAgent` left out.
* `CallMCP` runs as `OriginAgent`; a bad argument, an unknown name and a
  refusal are `isError: true` with a text the model can act on, never a
  Go error.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestExportGolden` | the goldens, for a catalogue of one command per kind and danger level |
| `TestMCPFieldNames` | decoded into `map[string]any`, each object's keys are a subset of 2026-07-28's, and the required ones are present |
| `TestACPFieldNames` | `name`, `description`, `input`, `hint`, and nothing else |
| `TestCallMCPErrors` | a bad argument, an unknown name and a gate refusal each give `isError: true` and a message |
| `TestCallMCPIsAgent` | the default policy applies to `CallMCP` |
| *D24:* `TestRunOnLoop` | with `WithLoop`, `Run` of a `Loop` command sends a `LoopMsg`, runs only when it is run, returns its result, and gives its `Cmd` to the program; without it, `Run` runs on its caller |
| `TestExportFilters` | hidden and non-agent commands are not exported |
| `TestManifestRoundTrip` | the manifest round-trips through JSON |

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `Destructive` exports `destructiveHint: false` | `TestExportGolden` |
| `CallMCP` runs as `OriginProgram` | `TestCallMCPIsAgent` |
| hidden commands are exported | `TestExportFilters` |
| `resultType` is left out | `TestMCPFieldNames` |
| tools are listed in insertion order | `TestExportGolden` |

**Done when:** the checks are clean, the field names are recorded, the
goldens are read, and every mutation is killed.

### Step 7: workspace integration

**Files:** `workspace/commands.go`, `workspace/context.go`,
`workspace/background.go`; tests `workspace/commands_test.go`,
`workspace/context_test.go`, `workspace/background_test.go`; goldens
`workspace/testdata/golden/commands-*`.

**Exported names** (MADR §8, §9, A2):

* `Commands(w *Workspace, o ...CommandOption) []command.Command`,
  `CommandOption`, `WithLayouts(map[string]layout.Node) CommandOption`;
* `(*Workspace).WhenContext() when.Context`; `Contexter interface{
  WhenContext() when.Context }`;
* the typed keys, as package variables: `KeyFocusedPane`
  (`workspace.focusedPane`, string), `KeyZoomed` (`workspace.zoomed`,
  bool), `KeyHiddenPanes` (`workspace.hiddenPanes`, list; *D27:
  `Plan().Hidden`*), `KeyOverlay`
  (`workspace.overlay`, string), `KeyModal` (`workspace.modal`, bool),
  `KeyWidth` (`workspace.width`, int), `KeyHeight` (`workspace.height`,
  int); and `ContextKeys() when.Keys`;
* `(*Workspace).SetBackground(bg theme.Background) tea.Cmd` (A2, Q5).

**The built-ins,** each `Loop` mode, every surface but CLI, Danger `UI`
unless marked:

| ID | Slash | Arguments | Calls |
| :--- | :--- | :--- | :--- |
| `workspace.focus` | `focus` | `pane` (`arg`) | `Focus`; a pane not in the plan is an `ArgError` |
| `workspace.focus.next`, `.prev` | `next`, `prev` | — | `FocusNext`, `FocusPrev` |
| `workspace.zoom` | `zoom` | `pane` (`arg`, `omitzero`) | `Zoom(pane)`, the focused pane when empty (*D28: a pane the layout does not know is an `ArgError`*) |
| `workspace.toggle` | `toggle` | `pane` (`arg`) | `Toggle` (*D28: likewise*) |
| `workspace.resize` | `resize` | `split` (`arg`), `delta` (`arg`, `-200..200`) | `Resize`; a split that is not a resizable separator of `Plan()` is an `ArgError` |
| `workspace.layout.use` | `layout` | `name` (`arg`, an enum of `WithLayouts`'s keys) | `SetLayout`; registered only when `WithLayouts` names a layout |
| `workspace.layout.reset` | — | — | `SetState(layout.State{})` |
| `workspace.state.get` | — | — | `State()`, as `Value`; `ReadOnly` |
| `workspace.state.set` | — | `state` (a `layout.State`) | `SetState` |
| `workspace.panes` | `panes` | — | each pane's ID, title (`Titled`, else the ID), rectangle (`Plan().Panes`), focus and hidden state (`Plan().Hidden`), in focus-ring order; `ReadOnly` |
| `workspace.overlay.close` | `close` | — | `Pop`; with no overlay, a `Result.Text` that says so |
| `workspace.theme.set` | `theme` | `background` (`arg`, `dark`, `light`, `auto`) | `SetBackground(theme.Dark)`, `(theme.Light)` or `(theme.Unknown)` |

**`SetBackground`:** a new `pinnedBg` field, and the last reported
background kept apart from the one in use. `theme.Dark` or `theme.Light`
pins it, rebuilds, and a later `tea.BackgroundColorMsg` changes only the
reported one; `theme.Unknown` unpins, rebuilds from the reported one, and
returns `tea.RequestBackgroundColor` unless `WithoutBackgroundQuery`; on a
workspace built `WithTheme` it records the choice and returns nil.

**`WhenContext`** layers the top overlay's `Contexter`, then the focused
pane's, then the workspace's keys.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestBuiltinsMatchMethods` | each built-in, dispatched through a registry, gives the same frame and `State` as its method |
| `TestPanesCommand` | every pane, visible and hidden, with its rectangle and focus (*D29: the ring, then other placed panes, then `Plan().Hidden`*) |
| `TestResizeSplit` | `sidebar` moves; a pane name or a positional split is an `ArgError` (*D30: `sidebar` means the separator `sidebar:0`, which also works; a name with several separators is an `ArgError`*) |
| `TestLayoutUseEnum` | the schema's enum is exactly `WithLayouts`'s keys, sorted; without them the command is absent |
| `TestLayoutReset` | a zoomed, resized and hidden layout returns to the zero state |
| `TestWhenContextLayers` | overlay, then focused pane, then workspace |
| `TestAgentMayZoom` | `workspace.zoom` runs for `OriginAgent` under the default policy |
| `TestKeysStillWork` | the existing key bindings still work |
| `TestSetBackgroundPins` | a pinned dark survives a light `BackgroundColorMsg` |
| `TestSetBackgroundAuto` | `Unknown` rebuilds from the reported background and returns the query, or nil with `WithoutBackgroundQuery` |
| `TestSetBackgroundFixedTheme` | on a `WithTheme` workspace the frame does not change |
| `TestCommandsGolden` | the agent-session example zoomed, restored and set light through `Dispatch`, across the matrix at 80 and 160 columns |

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `WhenContext` puts the workspace keys first | `TestWhenContextLayers` |
| `workspace.panes` leaves hidden panes out | `TestPanesCommand` |
| `workspace.focus.next` moves twice | `TestBuiltinsMatchMethods` |
| a background message overrides a pinned background | `TestSetBackgroundPins` |
| `auto` sends no query | `TestSetBackgroundAuto` |
| `workspace.resize` accepts any split | `TestResizeSplit` |

**Done when:** the checks are clean, the goldens are read, and every
mutation is killed. `workspace`'s new imports are `command` and `when`
only.

### Step 8: `command/cli`

**Files:** `command/cli/cli.go`, `command/cli/flags.go`,
`command/cli/help.go`; tests `command/cli/cli_test.go`,
`command/cli/example_test.go`; goldens
`command/cli/testdata/golden/help-*.golden`.

**Exported names** (MADR §10): `Run(ctx context.Context, r
*command.Registry, args []string, stdout, stderr io.Writer, o ...Option)
int`; `Option`, `WithName(string)`, `WithConfirm(func(prompt string)
bool)`, `WithContext(when.Context)`, `WithWidth(int)`; the exit-code
constants `ExitOK = 0`, `ExitFailed = 1`, `ExitUsage = 2`,
`ExitRefused = 3`. (*D31: and, in `command`, `Request.Gate`, with its
policy and a test, `TestRequestGate`, in `command/gate_test.go`. D32: an
object property's flag takes JSON. D33: a command whose `When` is false
exits 1. D34: `--json` prints `null` without a value.*)

**Build:** word and dotted forms; a flag per schema property, an array
repeating its flag; `--args`, `--json`, `--yes`; the verbs `list`,
`describe`, `schema`, `help`. Help is plain ASCII at the option's width
(80 by default). A shell invocation is `OriginCLI`; `Result.Cmd` is
ignored; commands without `SurfaceCLI` are neither listed nor run.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestHelpGolden` | the registry's help and one command's, at 60 and 100 columns, ASCII only |
| `TestFlagsEqualArgs` | `--delta 4` and `--args '{"delta":4}'` build the same request |
| `TestFlagKinds` | a string, integer, boolean, enum and array property each parse; a bad value exits 2 with the `ArgError` on stderr |
| `TestJSONOutput` | `--json` writes `Result.Value`; otherwise `Result.Text` |
| `TestDestructiveNeedsYes` | without `--yes` or `WithConfirm`, exit 3 |
| `TestCLISurface` | a command without `SurfaceCLI` is neither listed nor run |
| `TestWritersOnly` | everything goes to the given buffers |

`ExampleRun` builds a registry with two commands and runs `list` and one
command with `--json`.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `--yes` is not required | `TestDestructiveNeedsYes` |
| a usage error exits 1 | `TestFlagKinds` |
| hidden commands are listed | `TestHelpGolden` |
| `--json` writes `Result.Text` | `TestJSONOutput` |

**Done when:** the checks are clean, the goldens are read, and every
mutation is killed.

### Step 9: documentation, and the root release `v0.4.0`

* **`docs/guides/commands.md` (new):** defining a command and `New[A]`
  with the tags; kinds, danger and the default gate; dispatching from
  `Update`, and `Run` for the shell and tests; command files and their
  front matter; MCP prompts and ACP commands; exporting to an agent, with
  what a host still owns under MCP 2026-07-28 (the list envelope,
  `subscriptions/listen`); `when` and the context keys;
  `workspace.Commands` and `SetBackground`; `command/cli`.
* **`docs/architecture.md`:** the three packages and their imports,
  `workspace`'s two new imports, the tree. **`docs/README.md`:** "I want
  to…" rows for adding a command, letting an agent drive the TUI, and
  running commands from the shell. **`README.md`:** Status.
* **Release notes** in the execution record.
* **Checks:** Rule 3, and `make release-check`.
* **The release:** the owner commits, pushes and, with CI green, tags
  `v0.4.0`. The agent then runs the consumer smoke test of
  `docs/guides/releasing.md` against `v0.4.0`, importing `command`,
  `when` and `command/cli`, and records it.
* **Done when:** `v0.4.0` is tagged and the smoke test builds.

### Step 10: `command/cobracmd`, the Cobra front end

**Starts when** `v0.4.0` is tagged.

* **Before the first file,** record in the execution record: the newest
  `github.com/spf13/cobra` and `github.com/spf13/pflag` versions and
  their `go.mod`s, the licence of each module the new `go.mod` adds, and
  `GOWORK=off govulncheck ./...` in the new module. A finding stops the
  step for the owner. 0010-MADR §5 named Cobra v1.10.2 and pflag v1.0.10.
* **Files:** `command/cobracmd/go.mod` (`module
  github.com/maccavelli/go-tui-lib/command/cobracmd`, `go 1.27.1`,
  requiring `github.com/maccavelli/go-tui-lib v0.4.0` and Cobra, no
  `replace`), `go.sum`, `cobracmd.go`, `flags.go`, `help.go`,
  `complete.go`; `command/cobracmd/docs/docs.go`; tests, and the shared
  front-end cases in `command/cobracmd/frontend_test.go`; goldens under
  `command/cobracmd/testdata/golden/`. `go work use ./command/cobracmd`.
  `AGENTS.md` and `docs/architecture.md` drop "(planned)" from the
  module's row.
* **Exported names** (MADR A1): `New`, `Mount`, `Run`, `Option`,
  `WithName`, `WithConfirm`, `WithContext`, `WithTheme`, `WithGlyphs`,
  `WithWidth`; in `docs`, `Man(w io.Writer, root *cobra.Command, date
  time.Time) error` and `Markdown(w io.Writer, root *cobra.Command) error`.
* **Build:** MADR A1's tree, flags, completion, verbs, exit codes, help and
  rules, as written there. (*A10, D35–D40: a hidden command is in the
  tree, hidden; no `MarkFlagRequired`; no deprecation, completion
  providers or flag relations; `Run` takes a dotted ID; `Mount` adds the
  verbs and refuses a clash.*)
* **Tests:** the shared front-end cases agree with `command/cli`; no
  hidden or non-CLI command in the tree (*D35: no non-CLI command in the
  tree, and a hidden one hidden*); nested commands and groups;
  `Mount` beside a program's own commands; an enum flag refuses and
  completes; golden help across 0001-MADR §6's matrix at 60 and 100
  columns; golden completion scripts for bash, zsh, fish and PowerShell;
  golden man and Markdown pages, unchanged between two runs; `Run` reads no
  `os.Args` and writes to no standard stream; a second `New` in one
  process works and registers no completion twice.
* **Mutations:** a flag's type from the Go field instead of the schema;
  `Destructive` without `--yes`; `Run` without `SetArgs`; a group added
  after its children (*D39: survives in Cobra; replaced by "a category's
  group is never added" and "`Mount` adds a group the parent already
  has"*); depguard allowing Cobra in the root.
* **Checks:** Rule 3, in the module's directory with `GOWORK=off`, and
  the tests again in workspace mode; `scripts/go-modules.sh --check`.
* **The release:** the owner commits, pushes and tags
  `command/cobracmd/v0.1.0`; the agent runs the smoke test against it.
* **Done when:** the tag exists and the smoke test builds.

### Step 11a: `New[A]` agrees with Kong, and the root release `v0.5.0`

**Starts when** the owner approves this step (A12, D47). Step 11's
uncommitted module waits in the tree, outside `go.work`, until it is done.

* **Files:** `command/args.go` (`New` calls the check);
  `command/kongname.go` (new: Kong's name spelling and the check);
  `command/args_test.go`; `workspace/commands.go` (`zoomArg.Pane` gains
  `optional:""`, `stateArg.State` `required:""`); the test fixtures the
  check refuses, in `command`, `command/cli` and `command/cobracmd`;
  `docs/guides/commands.md` (the tags section states the rule); this
  PLAN's record.
* **Exported names:** none. `New`'s documentation states the rule.
* **Build:** for each top-level field of `A` that `SchemaOf` makes a
  property, compute requiredness by the `json` rule and by Kong's
  (`required:""` for a flag; for a positional, required unless
  `optional:""` or a default), and Kong's name (`name:""`, or the Go name
  split at case changes, joined with dashes, lowercased); a difference is
  an error from `New` naming the field and the tag to add or remove. The
  spelling is written here from Kong's documented behaviour, not copied
  from its source. `SchemaOf` is unchanged.
* **Tests** (`command/args_test.go`): `TestNewAgreesWithKong`: a required
  flag without `required:""`, an optional positional without
  `optional:""`, `required:""` beside `omitzero`, and a `json` name Kong
  spells differently are each refused with the fix named; the same
  structs with the fix, a positional with a default, a pointer flag and
  an embedded struct are accepted; `SchemaOf` of a refused struct still
  succeeds. `TestKongName`: a table of Go names, among them acronyms,
  digits and single letters (`MaxItems`, `PaneID`, `HTTPServer`, `X2`,
  `N`), and their spellings.
* **Mutations:** the required check skipped; a positional's default not
  counted; the name check skipped; the spelling not lowercasing (or not
  splitting an acronym before a word).
* **Checks:** Rule 3; `make release-check`; and `command/cobracmd`'s
  tests in workspace mode against the changed root.
* **The release:** release notes for `v0.5.0` in the record; the owner
  commits, pushes, and with CI green tags `v0.5.0`; the agent runs the
  consumer smoke test against it.
* **Done when:** `v0.5.0` is tagged and the smoke test builds.

### Step 11: `command/kongcmd`, the Kong front end

**Starts when** `v0.4.0` is tagged; it does not wait for Step 10. (*A12:
its module requires `v0.5.0`, so it is finished after Step 11a.*)

* **Before the first file,** record the newest
  `github.com/alecthomas/kong` and its `go.mod`, each licence, and
  `GOWORK=off govulncheck ./...`. 0010-MADR §5 named Kong v1.16.1.
* **Files:** `command/kongcmd/go.mod` (as Step 10's, with Kong),
  `go.sum`, `kongcmd.go`, `grammar.go`, `help.go`, `complete.go`,
  `resolver.go`; tests, the shared front-end cases in
  `command/kongcmd/frontend_test.go`; goldens under
  `command/kongcmd/testdata/golden/`. `go work use ./command/kongcmd`.
  `AGENTS.md` and `docs/architecture.md` drop "(planned)".
* **Exported names** (MADR A1): `New`, `Adapter`, `Adapter.Options`,
  `Adapter.Parser`, `Adapter.Run`, `Adapter.Resolver`, `Option`,
  `WithName`, `WithConfirm`, `WithContext`, `WithTheme`, `WithGlyphs`,
  `WithWidth`.
* **Build:** MADR A1's mounting, grammars, dispatch, groups, help,
  settings, completion and rules, as written there. (*A11, D41–D46:
  `Resolver` from `config.*` context keys; a Kong-native grammar;
  completion through `__complete`; the caller's `kong.Writers`; a
  command that is also a parent as its own hidden default child; a name
  clash fails `kong.New`.*)
* **Tests:** golden completion scripts for each shell; the shared
  front-end cases (*D42: less the requests a Kong-native grammar refuses,
  which are asserted apart*); one struct, two readers (`SchemaOf` and the Kong
  grammar agree on names, required fields, enums, defaults and positional
  order) (*D47: read directly by Kong and through the adapter, for
  structs `New` accepts; and `command`'s name spelling against Kong's
  model*); nested `cmd` selection; `--help` returns 0 and parsing stops;
  mounting beside a program's own grammar, with `handled` false for its
  commands; help at the option's width whatever `$COLUMNS` says, golden
  across the matrix at 60 and 100 columns; no standard stream and no
  `os.Args`; `Resolver` fills a flag and a command-line flag wins; no `env`
  tag unless asked.
* **Mutations:** `Exit` returns instead of panicking; dispatch from the
  first registry command instead of `ctx.Selected()`; an enum's default
  dropped; `kong.Writers` unset; depguard allowing Kong in the root.
* **Checks:** as Step 10.
* **The release:** the owner tags `command/kongcmd/v0.1.0`; the agent
  runs the smoke test against it.
* **Done when:** the tag exists and the smoke test builds.

### Step 12: close-out

* Verification, item by item, in the execution record; this PLAN set
  `complete`, with `docs/README.md`'s row.
* **Done when:** every Verification item holds.

## Verification

* Every step's mutations are killed, and each earlier set again after a
  later change moves its anchors.
* Rule 3's checks are clean at every step, on the macOS development host
  and the Windows test host.
* `make fuzz` covers `layout`, `when` and `command`, and CI's `gates` job
  runs it.
* The root `go.mod` is unchanged from `v0.3.0` through `v0.4.0`, and never
  names Cobra or Kong; `go mod tidy -diff` is clean in every module.
* Each nested `go.mod` requires a published root, `command/cobracmd`
  `v0.4.0` and `command/kongcmd` `v0.5.0` (A12), has no `replace`, and
  passes every gate of 0010-MADR §4 with `GOWORK=off` and in workspace
  mode.
* depguard keeps Cobra and pflag in `command/cobracmd`, Kong in
  `command/kongcmd`, and refuses the Charm v1 paths, mcplib, the MCP
  go-sdk and go-llmprovider-sdk everywhere.
* `internal/conformance` finds nothing in `when`, `command`, `command/cli`,
  `cobracmd` and `kongcmd`.
* The shared front-end cases pass in all three front ends.
* The MCP field names match 2026-07-28's schema, and the ACP ones the ACP
  pages, as recorded in Step 6.
* The consumer smoke test builds against `v0.4.0`,
  `command/cobracmd/v0.1.0`, `v0.5.0` and `command/kongcmd/v0.1.0`.
* The identifier scan finds nothing, and CI is green after each push.

## Rollout and Rollback

* **Rollout:** Steps 2 to 9 are pushed and `v0.4.0` is tagged; then each
  front end lands, requires `v0.4.0`, and is tagged under its prefix
  (0010-MADR §3). pi-go adopts the registry under its own records.
* **Rollback:** before a push, each step is one local commit; after it, a
  patch release fixes forward. The new packages are additive, so a
  consumer that does not import them is unaffected. Step 7 adds names to
  `workspace` and changes no existing one; removing them breaks only their
  callers. A front end is fixed forward under its own prefix while a
  consumer pins its previous tag; tags are never moved or deleted.

## Execution Record

### Step 2: `when`

#### Deviations

* **D1 (2026-10-05): the canonical form can outgrow the source limit.**
  `go test -run '^$' -fuzz=FuzzParse -fuzztime=30s ./when` failed: a
  minimized source under 4096 bytes, `A<000\…` with thousands of
  backslashes in a bareword, prints `A < '000\\…'`, "which does not
  parse: when: source of 6718 bytes, more than 4096". The canonical form
  quotes a bareword, doubles its backslashes and adds spaces around
  operators, so the PLAN's fuzz property and MADR §8's round trip were
  false at the limit. Options given: bound the canonical form too
  (recommended), or limit the source only and weaken the property to
  canonical forms that fit. **The owner picked "Bound the canonical form
  too".** `Parse` now refuses an expression whose canonical form is longer
  than `MaxSource`; recorded as MADR amendment A3. Added to the step:
  a `TestLimits` case, a seventh mutation, and the failing input as a
  regression seed in `when/testdata/fuzz/FuzzParse/`.
* **D2 (2026-10-05): a mutation survived.** "`String` drops needed
  parentheses" (`wrap`'s `n.prec() < need` made `n.prec() < need-10`)
  passed `TestStringRoundTrip`, which checked only that the canonical text
  reprints unchanged: `a && (b || !c)` printed `a && b || !c`, which is
  stable and means something else. Under Rule 4 the test was
  strengthened: `TestStringRoundTrip` also evaluates the expression and
  its re-parsed canonical form on every assignment of `true` and `false`
  to its keys, over three more sources that need parentheses; and
  `FuzzParse` fails when the two evaluate differently on its contexts.
  Seen to fail: the set below, and the same mutation fuzzed on a scratch
  copy (`-fuzz '^FuzzParse$' -fuzztime 20s`) failed on a seed in 0.07 s:
  `fuzz_test.go:51: "a && (b || !c)" prints "a && b || !c", which
  evaluates differently`.
* **D3 (2026-10-05): CI's fuzz step named only `layout`.** `make fuzz`
  runs in CI, so CI fuzzed `when` with no change; but the step's comment
  said only `layout` had fuzz targets, and on a failure it uploaded only
  `layout/testdata/fuzz/`, so a failing `when` input would be lost.
  `.github/workflows/ci.yml` was not in the step's files. Options given:
  add it to Step 2 (recommended), or leave it as a known gap. **The owner
  picked "Add ci.yml to Step 2".** The comment names both packages, and
  the artifact's `path` lists `layout/testdata/fuzz/` and
  `when/testdata/fuzz/`. `actionlint` is not installed here; the file
  parses as YAML, and the step's `path` reads back as the two lines.

#### What was built

* `when/when.go`, `lex.go`, `parse.go`, `eval.go`, `value.go`, `key.go`:
  the exported names listed above and no others, plus the constants
  `MaxSource` and `MaxDepth`. A hand-written scanner and a
  recursive-descent parser; a regex compiles with `regexp` at parse time,
  with the flags `i`, `m` and `s` written as `(?ims)`. `Parse` refuses a
  canonical form over `MaxSource` (D1). The package imports only the
  standard library.
* `when/when_test.go`: the twelve tests of the table, and
  `TestParseErrors`, `TestKeysInOrder` and `TestZeroExpr`.
  `when/fuzz_test.go`: `FuzzParse`, seeded from the operator table and
  five more sources. `when/bench_test.go`: `BenchmarkParse` and
  `BenchmarkEval`. `when/testdata/fuzz/FuzzParse/01e90e822ccb8b53`: D1's
  input.
* `Makefile`: the `fuzz` target runs `./scripts/go-fuzz.sh -t
  $(FUZZTIME) -m 1 ./when` after `./layout`, and its help text names
  `when`. `.github/workflows/ci.yml`: D3.
* Lint findings fixed on the way, before the checks below: `goconst` on
  `"true"` and `"false"` (now `trueWord` and `falseWord`), and
  `gocritic`'s `regexpSimplify` (`[0-9]` is `\d`, which RE2 limits to
  ASCII).

#### Checks (Rule 3)

1. `gofmt -l when`: no output.
2. `make pre-add-check FILES="<the nine when/*.go files>"`: exit 0,
   "go-precheck: 9 file(s) clean in 1 module(s) (gofmt, golangci-lint, go
   vet, go test, go mod tidy, govulncheck)."
3. `make lint`: exit 0; "0 issues." for `GOOS=linux`, `darwin` and
   `windows`.
4. `GOWORK=off go test -race -count=1 ./...`: exit 0, 14 packages `ok`,
   `when` among them; `internal/conformance` passes with `when` in its
   scan.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: exit 0, 14 packages
   `ok`.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: exit 0, 14 packages `ok`.
7. `GOWORK=off go mod tidy -diff`: exit 0, no output.
8. `make fuzz FUZZTIME=20s`: exit 0; "go-fuzz: 1 fuzz targets ran clean
   in ./layout", "go-fuzz: 1 fuzz targets ran clean in ./when"
   (`FuzzParse`, 5,701,018 executions). No new input was written.
9. Windows test host, go1.27.1 windows/amd64, on a copy of the tree:
   `make pre-add-check` exit 0 ("83 file(s) clean in 1 module(s)"),
   `make lint` exit 0 ("0 issues." for each target), `make vuln` exit 0
   ("No vulnerabilities found."), `GOWORK=off go test -count=3
   -shuffle=on ./when/` exit 0.
10. `markdownlint-cli2 --config .markdownlint-cli2.jsonc` on
    `docs/README.md` and this pair: "0 issues in 0 files"; the
    configuration excludes MADR and PLAN files, so only `docs/README.md`
    is linted. The relative-link check of the three files: 0 broken.
11. Identifier scan of the diff and the new files, for the local account
    name, the test host's name, home-directory paths and the employer's
    domain: no match.

Benchmarks, on the development host (`go test -run '^$' -bench .
-benchmem ./when`): `BenchmarkParse` 5278 ns/op, 11173 B/op, 69 allocs/op;
`BenchmarkEval` 110.2 ns/op, 32 B/op, 1 allocs/op.

#### Mutations (Rule 4)

Each on a scratch copy of the tree, with `go test -count=1 -run '^<Test>$'
./when`. All seven killed:

| Mutation | Applied as | Failing line |
| :--- | :--- | :--- |
| `&&` binds looser than `\|\|` | `or` loops on `tAnd` building `andNode`, `and` on `tOr` building `orNode` | `when_test.go:67: foo \|\| bar && baz read as (foo \|\| bar) && baz` |
| an unset key compares equal to `''` | `cmpNode.eval` on an unset key uses `StringValue("")` | `when_test.go:93: an unset key equals ''` |
| the depth limit is not checked | `p.depth > MaxDepth*1000` | `when_test.go:220: 65 nested parentheses parsed` |
| `Layered` takes the last match | the loop runs from the last context | `when_test.go:155: k = bottom, true; want the first context's` |
| `String` drops needed parentheses | `n.prec() < need-10` | `when_test.go:275: "!(a && b)" prints "!a && b", which differs on map[a:false b:false]` (after D2) |
| `not in` is evaluated as `in` | `return in` | `when_test.go:105: "item not in l" = true, want false` |
| D1: the canonical form's length is not checked | `n > MaxSource*100` | `when_test.go:212: a source of 4096 bytes printing longer than MaxSource: <nil>` |

### Step 3: `command` core

#### Deviations

Found before any Step 3 code was written; MADR amendment A4 records the
decisions.

* **D4 (2026-10-05): zero values.** The PLAN lists `Danger` and `Origin`
  in order but not their values. With plain `iota`, an undeclared
  `Danger` is `ReadOnly`, which an agent runs with no gate, and a request
  without an `Origin` is `OriginKey`. Options given: zero means unset and
  is refused (recommended); zero is treated as the safe level; plain
  `iota`. **The owner picked "Zero means unset; refuse it".**
* **D5 (2026-10-05): exported errors.** Step 8's exit code 3 needs
  `command/cli` to recognise a refusal, and the step lists no error.
  Options given: three sentinel errors (recommended); one error type with
  a code; none until Step 8. **The owner picked "Three sentinel
  errors"**, which adds `ErrUnknown`, `ErrUnavailable` and `ErrRefused`
  to the step's names (Rule 2).
* **D6 (2026-10-05): the surfaces of a click and of the program.** Five
  origins have a surface; `OriginMouse` and `OriginProgram` do not.
  Options given: a click as a key and the program unchecked
  (recommended); neither checked; a new `SurfaceMouse`. **The owner
  picked "Mouse as Key; Program unchecked".**

#### What was built

* `command/command.go`: the package documentation, `ID` (`Valid`,
  `Segments`), `Command`, `Schema`, and the enumerations `Kind`,
  `Danger`, `Surface` (with `AllSurfaces`), `Mode`, `Scope` (with
  `Global`), `SourceKind`, `Source` and `Origin`, each with `String`.
  `command/handler.go`: `Handler`, `HandlerFunc`, `Invocation`, `Result`,
  `Request`. `command/registry.go`: `Registry`, `NewRegistry`,
  `RegistryOption`, `WithGate`, `WithAuditor`, `WithPrefixer`, the
  errors (D5), and `Register`, `Remove`, `Lookup`, `Slash`, `All`,
  `Available`, `Version`, `Watch`. `command/dispatch.go`: `Dispatch`,
  `Run`, `Cancel`, `CancelAll`. `command/gate.go`: `Decision`
  (`ACPKind`), `Gate`. `command/audit.go`: `Auditor`, `Record`,
  `SlogAuditor`. `command/messages.go`: `ResultMsg`, `PromptMsg`,
  `ChangedMsg`, `ConflictMsg`, `QuitRequestMsg`, `Conflict`. No other
  exported name. `command` imports `when`, `bubbletea` and the standard
  library only.
* Choices inside the listed names, none of which adds a name:
  * `Decision` also starts at 1, as `Danger` and `Origin` do under D4, so a
    gate that returns the zero value refuses. This applies D4's rule to a
    third type the owner was not asked about, and is stated here so it
    can be reversed.
  * `Conflict` is `{ID, Slash, Renamed, Holder}`: the loaded command, the
    name it asked for, the name it got, and the command that holds the
    name. Step 5 fills it.
  * `Dispatch` asks the gate on the calling goroutine, in the order this
    step's Build gives; the `Gate` documentation says a gate that waits
    for the user is reached through `Run`, from the agent's goroutine.
  * A `Forward` command needs no handler and ignores one; its text is
    `/<Slash> <Raw>`, or `/<ID> <Raw>` when it has no slash name.
  * `All` includes hidden commands and `Available` leaves them out; a zero
    surface given to `Available` is any surface. `Slash` accepts a name
    with or without its leading `/`. `Register` keeps its own copies of
    `Aliases`, `Args`, `Output` and `Meta`.
  * Every request is audited, including one for an unknown ID; a record's
    `Decision` is zero when the request failed before the policy was
    asked. `Record.Args` is the request's JSON as given: masking `secret`
    fields is Step 4's (`TestAuditMasksSecrets`).
  * `WithPrefixer` stores its function, which Step 5 uses.
* Tests beyond the table: `TestRegisterCopies`, `TestStrings`,
  `TestPromptAndForward` (a `PromptMsg` for a `Prompt` and a `Forward`,
  none for an `Action`), `TestNonExclusiveRunsTogether`,
  `TestDecisionACPKind`.
* Fixed on the way: `make modernize` replaced a hand-written loop in
  `dispatch_test.go` with `slices.Contains`, and revive's
  `confusing-naming` renamed the test helper `newRegistry` to
  `registryOf`. The first `make pre-add-check` run passed with the helper
  still named `newRegistry`; `make lint` then reported it, and so did the
  next pre-add check. Why the first run passed was not investigated.
* `TestAlways`'s gate first indexed an empty answer queue, so the
  "`AllowAlways` is not remembered" mutation was killed by a panic. The
  gate now reports an unexpected question as a test error, and the
  mutation fails on that assertion.

#### Checks (Rule 3)

1. `gofmt -l command`: no output.
2. `make pre-add-check FILES="<the eleven command/*.go files>"`: exit 0,
   "go-precheck: 11 file(s) clean in 1 module(s) (gofmt, golangci-lint,
   go vet, go test, go mod tidy, govulncheck)."
3. `make lint`: exit 0; `go fix -diff` silent, and "0 issues." for
   `GOOS=linux`, `darwin` and `windows`.
4. `GOWORK=off go test -race -count=1 ./...`: exit 0, 15 packages `ok`,
   `command` among them; `internal/conformance` passes with `command` in
   its scan.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: exit 0, 15 packages
   `ok`.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: exit 0, 15 packages `ok`.
7. `GOWORK=off go mod tidy -diff`: exit 0, no output.
8. No fuzz target in this step.
9. Windows test host, go1.27.1 windows/amd64, on a copy of the tree:
   `make pre-add-check` exit 0 ("94 file(s) clean in 1 module(s)"),
   `make lint` exit 0, `make vuln` exit 0 ("No vulnerabilities found."),
   `GOWORK=off go test -count=3 -shuffle=on ./command/` exit 0.
10. `markdownlint-cli2` on `docs/README.md`: 0 issues (the configuration
    excludes MADR and PLAN files); the relative-link check of
    `docs/README.md` and this pair: 0 broken.
11. Identifier scan of the diff and `command/`, for the local account
    name, the test host's name, home-directory paths and the employer's
    domain: no match.

Benchmarks, on the development host (`go test -run '^$' -bench .
-benchmem ./command`), over 500 commands, half with a `When`:
`BenchmarkLookup` 30.10 ns/op, 0 allocs; `BenchmarkAvailable` 16093 ns/op,
0 allocs; `BenchmarkDispatchLoop` 241.9 ns/op, 560 B/op, 5 allocs/op.

#### Mutations (Rule 4)

Each on a scratch copy of the tree. All eleven killed: the step's six, and
five for D4–D6. Step 2's set was not rerun, because this step changes
nothing under `when/`.

| Mutation | Applied as | Failing line |
| :--- | :--- | :--- |
| the default policy allows `Mutating` for agents | `asks`: `case Mutating: return false` | `gate_test.go:90: no gate: mutating from agent: <nil>; want refused true` |
| `AllowAlways` is not remembered | only `RejectAlways` is stored | `gate_test.go:123: the gate was asked about mutating for a, which it answered always` |
| `Version` is not bumped on `Remove` | `Remove` stores a snapshot with the old version and channel | `registry_test.go:285: after Remove: <nil>, want ChangedMsg{2}` |
| `Exclusive` is ignored | `running.start(ctx, c.ID, false)` | `dispatch_test.go:184: the async command did not end` |
| a write changes the published snapshot in place | `Register` appends to `old.entries` (run with `-race`) | `registry_test.go:218: All: b9.c6 after b9.c6`, and the race detector's report |
| `Dispatch` skips `When` | `!e.holds(c) && req.Origin == 0` | `dispatch_test.go:45: with its When false: ran true, <nil>`; `TestAvailable` fails too |
| D4: `Register` accepts an undeclared `Danger` | `c.Danger > Destructive` only | `registry_test.go:120: no danger: registered` |
| D4: a request with no origin runs | `req.Origin > OriginProgram` only | `gate_test.go:111: Run({ID:readonly … Origin:unset …}): <nil>, want command: refused` |
| D5: a gate refusal is not `ErrRefused` | the refusal wraps `ErrUnavailable` | `gate_test.go:90: reject once: mutating from agent: command: not available: …; want refused true` |
| D6: a click is not checked against `SurfaceKey` | `case OriginKey:` alone | `registry_test.go:189: palette from mouse: <nil>, want command: not available` |
| D6: the program is checked against a surface | `case OriginAgent, OriginProgram: return SurfaceAgent` | `registry_test.go:189: palette from program: command: not available: …, want <nil>` |

#### Open for later steps

Found while building Step 3, and outside it; each will be raised as a
deviation when its step starts, unless the owner decides it sooner:

* **Agents and `Loop` commands (Steps 6 and 7).** `Run` runs a `Loop`
  command on the caller's goroutine, as this step's Build says. An agent's
  tool call arrives on its own goroutine, so `CallMCP` running a
  workspace command through `Run` would touch the workspace off the event
  loop, which the workspace does not allow (0003-REPORT §1.12).
  (*Resolved in Step 6 as D24, MADR A7: `WithLoop`.*)
* **`--yes` and the gate (Step 8).** `command/cli` receives a registry
  whose gate was fixed by `NewRegistry`, so `--yes` and `WithConfirm` have
  no way to approve a `Destructive` request from the shell for one call.
  (*Resolved in Step 8 as D31, MADR A9: `Request.Gate`.*)

### Step 4: arguments, and the registry's own commands

#### Deviations

Found before any Step 4 code was written; MADR amendment A5 records the
decisions.

* **D7 (2026-10-05): `New` without a danger.** A4 has `Register` refuse
  an undeclared `Danger` and says `New` sets one, but not what `New` does
  when its caller names none. Options given: return an error
  (recommended); default to `Mutating`. **The owner picked "Return an
  error".**
* **D8 (2026-10-05): the built-ins' slash names.** The step and MADR §9
  give `command.list`, `command.describe` and `app.quit` none. Options
  given: none (recommended); `/commands`, `/describe` and `/quit`. **The
  owner picked "None".**
* **D9 (2026-10-05): tags with no JSON Schema keyword.** Only the schema
  reaches the audit trail and the front ends, and `secret`, `short`,
  `placeholder`, `group` and `hidden` have no keyword. Options given:
  `writeOnly` and one `x-cli` object (recommended); a separate `x-`
  keyword for each; drop the four CLI-only tags. **The owner picked
  "writeOnly + one x-cli object".**
* **D10 (2026-10-05): what `command.list` lists.** Options given: what
  the caller can run now (recommended); every command not hidden; every
  command. **The owner picked "What the caller can run now".**
* **D11 (2026-10-05): positionals in the schema.** `arg:""` has no JSON
  Schema keyword either, and D9's question missed it; `ParseSlash` needs
  each positional and its order. Options given: `"arg": true` in `x-cli`,
  in the order of `properties` (recommended); one `x-cli` list of the
  positional names on the object. **The owner picked `"arg": true` in
  `x-cli`.**
* **D12 (2026-10-05): CI's fuzz step, again.** As in D3, `make fuzz`
  runs `command`'s target in CI, but the step's comment named only
  `layout` and `when`, and a failing `command` input was not uploaded.
  Options given: add `.github/workflows/ci.yml` to the step (recommended);
  leave it as a known gap. **The owner picked "Add ci.yml to Step 4".**
  The comment names all three, and the artifact's `path` gains
  `command/testdata/fuzz/`; the file parses as YAML and the path reads
  back as the three lines.

#### What was built

* `command/args.go`: `New`, `Option` and its sixteen constructors,
  `NoArgs`, `ArgError`. `command/schema.go`: `SchemaOf` and the tags.
  `command/decode.go`: the checks, defaults, masking and strict decode.
  `command/slash.go`: `ParseSlash`. `command/builtin.go`: the registry's
  own commands. No other exported name.
* Changed from Step 3: `NewRegistry` builds its first snapshot, still
  version 0, with the three commands; `Register` and `NewRegistry` share
  `addEntries`. `admit` checks a command's arguments against its schema
  after `When` and before the policy, so the gate is never asked about a
  malformed request, and passes the checked arguments, defaults filled, to
  the handler. `audit` masks with the command's schema. Step 3's
  `TestRegisterDuplicate`, `TestAllSorted`, `TestAvailable` and
  `TestSnapshotsUnderRace` counted every command; they now leave out the
  three built-ins (the `ids` helper and the race test's count).
* Choices inside the listed names, none of which adds a name:
  * A field with `default` is optional, as in Kong, which A1 aligns with:
    A1's enum rule, "required or have a default", reads that way. The
    registry fills an absent default before it checks the arguments.
  * Every command with a schema has its arguments checked, whatever its
    origin; a command without one (a `Forward`, or one built by hand) is
    not. A loaded schema is enforced for `type` (a string or a list),
    `required`, `enum`, `minimum`, `maximum`, `minLength`, `maxLength`,
    `items`, `properties` and `additionalProperties`, and nested at most
    64 deep; any other keyword is carried and ignored, as MADR §4 says.
  * The json options `omitzero`, `omitempty` and `embed` are read; any
    other (`string`, `case`, `format`) is an error. An embedded struct
    without a name is flattened, as `encoding/json/v2` embeds it.
    `[]byte` is a base64 string, an unsigned integer has `minimum` 0, a
    `time.Duration` is a string with a `pattern` and is decoded with
    `time.ParseDuration`, and `enum` on a slice applies to its items.
  * `ParseSlash` takes the line with or without its `/`. `Raw` is the
    text after the name. A quoted `name=value` is a positional value, an
    unquoted one whose name is not a property is an `ArgError`, and a
    positional already given by name is skipped. `ParseSlash` checks and
    fills the arguments too, so an accepted line runs.
  * A masked value is `"***"`. Arguments that hold a secret and cannot be
    read are recorded as none.
  * Filling a default re-encodes the arguments, so their numbers pass
    through `float64`; an integer beyond 2^53 in a request that also
    gets a default loses precision. Without a default filled, the bytes
    pass unchanged.
  * `command.list` and `command.describe` return an unexported
    `commandInfo` as `Value`, with JSON names; Step 6's `Manifest` may
    replace it.
* Tests beyond the table: `TestDefaultsFilled`, `TestLoadedSchema`,
  `TestNewNeedsDanger` (D7, and every option landing),
  `TestUnknownMembersEachLayer`.
* Found and fixed while building, before the checks below:
  * `jsontext.Token.String` panicked on a token voided by the next read;
    the property name is now copied first.
  * `SchemaOf` called a value `JSONSchema` method on a nil pointer for a
    `*T` field; a pointer now takes its element's schema. The golden
    test found it.
  * Lint: `goconst` (JSON Schema's type names are constants now),
    `unconvert` (`json.RawMessage` is `jsontext.Value` in Go 1.27, so the
    conversion did nothing), `errcheck` on three type assertions and a
    marshal, and `make modernize`'s `slices.Contains`. `go vet` refused a
    test struct with a repeated json tag; the case now repeats the name
    through an untagged field.
* `make fuzz` gained `./command`; a 60-second run of `FuzzParseSlash`
  before the checks: 9,075,320 executions, no failure.

#### Checks (Rule 3)

1. `gofmt -l command`: no output.
2. `make pre-add-check FILES="<the twenty-one command/*.go files>"`: exit
   0, "go-precheck: 21 file(s) clean in 1 module(s) (gofmt,
   golangci-lint, go vet, go test, go mod tidy, govulncheck)."
3. `make lint`: exit 0; `go fix -diff` silent, and "0 issues." for
   `GOOS=linux`, `darwin` and `windows`.
4. `GOWORK=off go test -race -count=1 ./...`: exit 0, 15 packages `ok`.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: exit 0, 15 packages
   `ok`.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: exit 0, 15 packages `ok`.
7. `GOWORK=off go mod tidy -diff`: exit 0, no output.
8. `make fuzz FUZZTIME=20s`: exit 0; "1 fuzz targets ran clean" in
   `./layout`, `./when` and `./command` (`FuzzParseSlash`, 3,211,475
   executions). No input was written.
9. Windows test host, go1.27.1 windows/amd64, on a copy of the tree,
   after the last change: `make pre-add-check` exit 0, `make lint` exit
   0, `make vuln` exit 0, `GOWORK=off go test -count=3 -shuffle=on
   ./command/` exit 0. An earlier run of the same, before
   `TestUnknownMembersEachLayer`, reported "104 file(s) clean" and "No
   vulnerabilities found."
10. `markdownlint-cli2` on `docs/README.md`: 0 issues (the configuration
    excludes MADR and PLAN files); the relative-link check of
    `docs/README.md` and this pair: 0 broken.
11. Identifier scan of the diff, `command/*.go` and the goldens, for the
    local account name, the test host's name, home-directory paths and
    the employer's domain: no match.

The ten goldens were written with `-tuitest.update` and each was read:
valid JSON naming the 2020-12 `$schema`, properties in field order,
`required` as the rules say, `writeOnly` on the secret, `x-cli` on the
tagged fields, and `schema-noargs` exactly `{"$schema": …, "type":
"object", "additionalProperties": false}`.

Benchmarks, unchanged from Step 3 within noise: `BenchmarkLookup` 29.13
ns/op; `BenchmarkAvailable` 16106 ns/op; `BenchmarkDispatchLoop` 238.2
ns/op, 560 B/op, 5 allocs/op.

#### Mutations (Rule 4)

Each on a scratch copy of the tree. All thirteen killed, run again after
the last test change. Step 3's eleven were run again too, because this
step changed `registry.go` and `dispatch.go`: all killed, with the
snapshot mutation's anchor moved into `addEntries(...)`. Step 2's set was
not rerun, because nothing under `when/` changed.

| Mutation | Applied as | Failing line |
| :--- | :--- | :--- |
| `omitzero` fields are marked required | `optional` ignores the json options | `schema_test.go:119: tuitest: schema-map …: line 19 differs` |
| unknown members are accepted | the schema check and the strict decode both accept them | `decode_test.go:56: {"name":"a","delta":1,"inner":{"x":1,"y":2}}: <nil>, want an *ArgError at "/inner/y"` |
| `maximum` is not checked | `x > *r.max+1e9` | `decode_test.go:56: {"name":"a","delta":9}: <nil>, want an *ArgError at "/delta"` |
| positionals fill fields in reverse order | each positional is prepended | `slash_test.go:44: "/resize sidebar 4": command: argument /delta: "sidebar" is not a number` |
| `arg` presence is ignored | `f.Tag.Get("arg") != ""` | `slash_test.go:44: "/resize sidebar 4": command: arguments: "sidebar" is one positional value too many` |
| `enum` is split on `\|` | `strings.SplitSeq(e, "\|")` | `schema_test.go:108: enum: … enum value "1,2,3": strconv.ParseInt: … invalid syntax` |
| a map field is refused | every map key kind is refused | `schema_test.go:108: map: command: schema: map[string]int: a map key must be a string …` |
| `secret` is not masked | `mask` returns the arguments as given | `decode_test.go:159: record 0 holds the secret: …` |
| D7: `New` accepts a command with no danger | the check never fires | `builtin_test.go:80: New without WithDanger succeeded` |
| D8: a built-in has a slash name | `app.quit` gets `/quit` | `builtin_test.go:23: app.quit: registered true, slash "quit" []; want registered with no slash name` |
| D9: `secret` is not written as `writeOnly` | the `secret` key does nothing | `decode_test.go:159: record 0 holds the secret: …` |
| D9: `x-cli` is not written | `memberIf(false, "x-cli", …)` | `schema_test.go:218: split: x-cli map[], want map[arg:true placeholder:SPLIT]` |
| D10: `command.list` lists every command | `r.All()` | `builtin_test.go:45: command.list from agent = [… hidden off shell shown], want [app.quit command.describe command.list shown]` |

The unknown-member mutation disables both layers because each covers the
other: with one disabled, `TestDecodeStrict` still passed. Run alone, the
two single-layer mutations survived; `TestUnknownMembersEachLayer` was
added, and both are now killed:
`decode_test.go:80: the schema check accepted an unknown member` and
`decode_test.go:83: the strict decode accepted an unknown member`.

### Step 5: sources and loaders

#### Deviations

Found before any Step 5 code was written; MADR amendment A6 records the
decisions. Each was asked with options and the owner picked the
recommendation on 2026-10-05; the alternatives are in A6.

* **D13: `PromptGetter`'s signature,** which §6 does not give.
  `func(ctx, name, args) (string, error)`.
* **D14: names that are not ID segments** (MCP, ACP and file names).
  Mapped to a segment; a duplicate or empty result is refused and
  reported.
* **D15: reporting a command `ReplaceSource` cannot load,** with no
  error return. `Conflict` gains `Err error`.
* **D16: a loaded file's slash name.** Its path joined with `:`,
  prefixed only on a clash.
* **D17: a placeholder's property name.** As written.
* **D18: placeholders on the slash line.** Positional, in order of first
  use.
* **D19: a built-in registered after a loaded command holds its slash
  name.** `Register` takes it, and the loaded command is renamed.
* **D20: how `ConflictMsg` reaches the host.** Through `Watch`.
* **D21: an ACP forward's name.** Kept in `Meta["acp"]`.
* **D22: a renamed slash name that is also taken.** Dropped; the command
  still loads.
* **D23: `$ARGUMENTS` and stray words** (found while building, after
  D13–D22 were answered; MADR A6 item 11). The `LoadDir` golden showed
  `notes.md`, which uses only `$ARGUMENTS`, with a schema of no
  properties, so `/notes 100` would be refused. Options given: mark such
  a file's object `"x-cli": {"rest": true}` so `ParseSlash` leaves words
  past the positionals in `Raw` (recommended); no schema for a file whose
  only placeholder is `$ARGUMENTS`. **The owner picked "x-cli rest on the
  object".**

#### What was built

* `command/source.go`: `ReplaceSource`, the namespaces and `segment`.
  `command/loaddir.go`: `LoadDir`. `command/frontmatter.go`: the front
  matter's parser, checks and writer. `command/expand.go`: placeholders and
  expansion. `command/mcpprompt.go`: `MCPPrompt`, `MCPPromptArgument`,
  `PromptGetter`, `FromMCPPrompts`. `command/acp.go`: `ACPCommand`,
  `ACPCommandInput`, `FromACP`. `Conflict` gained `Err` (D15). No other
  exported name.
* Changed from Steps 3 and 4: the write path is a `draft`, a copy of the
  snapshot's entries with its indexes. `Register`, `ReplaceSource`,
  `Remove` and `NewRegistry` all build one. An entry records whether
  `Register` added it, which is what "built-in" means for D19. A snapshot
  carries its write's conflicts and a pointer to the next version, set
  before the old version's channel closes, so `Watch` reports exactly the
  change it woke for (D20). `forwardText` reads `Meta["acp"]` (D21).
  `cliInfo` and the compiled schema gained `rest` (D23).
* Choices inside the listed names:
  * A loaded file's title is its path without `.md` (`git/commit`), unless
    its front matter has `title`. Without `description`, it is the body's
    first non-blank line, with a Markdown heading's `#`s removed.
  * Blank lines inside the front matter are allowed, and a comment line is
    an error, like any line that is not `key: value`. A value may be
    wrapped in one pair of matching quotes, which are removed, with no
    escapes. A value that starts with `[`, `{`, `|` or `>` is nested and
    refused.
  * An `arg.<NAME>` that names no placeholder in the body is an error at
    its line. `aliases` is comma-separated.
  * `LoadDir` reads User, Project and Plugin sources, skips files and
    directories whose names start with `.`, and reads only `*.md`. A file
    that is not UTF-8 is an error. The body is trimmed of surrounding
    space once, at load.
  * An MCP prompt's arguments are positional in MCP's order, as D18 has
    placeholders; its title is MCP's `title`, else its name. An ACP
    command's title is its name.
  * Loaded commands are `Mutating` unless their front matter declares a
    danger. That includes MCP prompts and ACP forwards, which have none
    to declare.
  * A command that names one slash name twice keeps it once. A loaded
    command whose slash name another loaded source holds is renamed:
    whichever arrives later is renamed.
  * `ReplaceSource` always publishes a version, even when nothing changes.
* Tests beyond the table: `TestArgumentsTakesRest` (D23) and
  `TestLoadDirSources`.
* Found while building:
  * D23, from reading the `LoadDir` golden.
  * `FuzzFrontMatter` found that a quoted value of spaces, `" "`, was
    written back unquoted, then trimmed to nothing and refused. The writer
    now quotes a value with space at either end. The failing input is
    kept as a regression seed, and it failed before the fix: that is how
    it was found.
  * Lint: `goconst` (`keySlash` and `keyHidden`), and gocritic's
    `weakCond` (`len(m) == 2` for a regex match).
  * Three test inputs or expectations were wrong and were corrected:
    * the notes body keeps its heading line;
    * `fix-bug` has one string placeholder, so Step 4's whole-tail rule
      rightly takes every word;
    * an unclosed front matter whose next line reads as a field fails at
      that line, so the test's unclosed case has fields only.

#### Checks (Rule 3)

1. `gofmt -l command`: no output.
2. `make pre-add-check FILES="<the thirty command/*.go files>"`: exit 0,
   "go-precheck: 30 file(s) clean in 1 module(s) (gofmt, golangci-lint,
   go vet, go test, go mod tidy, govulncheck)."
3. `make lint`: exit 0; `go fix -diff` silent, and "0 issues." for
   `GOOS=linux`, `darwin` and `windows`.
4. `GOWORK=off go test -race -count=1 ./...`: exit 0, 15 packages `ok`.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: exit 0, 15 packages
   `ok`.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: exit 0, 15 packages `ok`.
7. `GOWORK=off go mod tidy -diff`: exit 0, no output.
8. `make fuzz FUZZTIME=20s`: exit 0. `./layout` and `./when` ran clean,
   and "2 fuzz targets ran clean in ./command": `FuzzFrontMatter`
   3,555,935 executions, `FuzzParseSlash` 3,168,280. Before that, after
   the writer's fix, each `command` target ran clean for 60 seconds:
   10,733,913 and 8,707,766 executions.
9. Windows test host, go1.27.1 windows/amd64, on a copy of the tree:
   `make pre-add-check` exit 0 ("113 file(s) clean in 1 module(s)"),
   `make lint` exit 0, `make vuln` exit 0 ("No vulnerabilities found."),
   and `go test -count=3 -shuffle=on ./command/` printed `ok`. That
   line's `exit=` showed `grep`'s status, not the test's: the script
   placed it wrong. `TestRootRefusesSymlink` ran there with `-v`:
   `--- PASS`, not skipped, so `os.Root` refused the escaping symlink on
   Windows too.
10. `markdownlint-cli2` on `docs/README.md`: 0 issues (the configuration
    excludes MADR and PLAN files); the relative-link check of
    `docs/README.md` and this pair: 0 broken.
11. Identifier scan of the diff, `command/*.go`, the golden, the test tree
    and the fuzz input, for the local account name, the test host's name,
    home-directory paths and the employer's domain: no match.

The `loaddir` golden was written with `-tuitest.update` and read, before
and after D23: four commands in path order, with the slash names,
titles, descriptions, dangers, argument hints and schemas the rules give,
including `"x-cli": {"rest": true}` on the three files that use
`$ARGUMENTS`.

**Benchmarks.** The development host was heavily loaded during this step
(load averages 76 to 91). `BenchmarkLookup`, whose code did not change,
ran from 33 to 103 ns/op. HEAD (Step 4) and this tree were run
alternately on a scratch export. Over eight pairs of
`BenchmarkDispatchLoop`, HEAD ran 444–812 ns/op and the tree 486–1473,
with medians near 654 and 675. Over four pairs of `BenchmarkAvailable`,
HEAD ran 16878–35397 and the tree 17317–39207. No difference due to the
code is visible. Step 4's figures (238 ns/op, 16 µs) were taken on an
idle host, and these do not replace them.

#### Mutations (Rule 4)

Each on a scratch copy of the tree; every failure is an assertion, none a
compile error. All sixteen killed: the step's five, one for each of
D14–D23 (D13 has none: the getter's signature is checked by the
compiler), and one for the writer's fix.

| Mutation | Failing line |
| :--- | :--- |
| a built-in loses a slash clash | `sources_test.go:51: conflicts [], want [{ID:user.review Slash:review Renamed:user:review Holder:builtin.review …}]` |
| unknown front-matter keys are ignored | `loaddir_test.go:137: unknown.md: "", want an error naming the file and ":3:"` |
| a command without `danger` defaults to `ReadOnly` | `loaddir_test.go:182: user.x is read-only, want mutating` |
| `ReplaceSource` keeps the old commands | `sources_test.go:121: conflicts [{ID:user.b … Err:command: user.b is taken by user's command}]` |
| `!{…}` is expanded | `frontmatter_test.go:14: expand: …` (the text differs) |
| D14: a name that is not a segment is refused, not mapped | `sources_test.go:187: commands: [{ID:mcp.github. …` |
| D15: a command that cannot load is dropped silently | `sources_test.go:148: 0 refusals, want 5: []` |
| D16: a loaded file's slash name is always prefixed | `loaddir_test.go:43: tuitest: loaddir …: line 4 differs` |
| D17: placeholder properties are lowercased | `loaddir_test.go:43: tuitest: loaddir …: line 8 differs` |
| D18: placeholders are not positional | `loaddir_test.go:43: tuitest: loaddir …: line 10 differs` |
| D19: `Register` refuses a name a loaded command holds | `sources_test.go:78: command: builtin.deploy: slash name "deploy" is taken by project.deploy` |
| D20: `Watch` sends no `ConflictMsg` | `sources_test.go:55: Watch sent [{2}], want a ConflictMsg with …` |
| D21: a forward sends its slash name, not the agent's | `sources_test.go:254: "/plan-mode step one" sent [/plan-mode step one], want "/Plan Mode step one"` |
| D22: a doubly taken name is taken anyway | `sources_test.go:100: a double clash: [… Renamed:user:x …], want the name dropped` |
| D23: a `$ARGUMENTS` file refuses stray words | `loaddir_test.go:73: "/notes 100 or so": null, command: arguments: "100" is one positional value too many; want {}` |
| the writer leaves a space-ended value unquoted | `frontmatter_test.go:56: "---\ndescription:\" \"\n---" re-serialised as … which does not parse` |

Because this step changed `registry.go`, `dispatch.go` and `slash.go`,
Step 3's eleven, Step 4's thirteen and Step 4's two single-layer
mutations were run again: all killed. Two of Step 3's anchors moved onto
the `draft`. "`Version` is not bumped on `Remove`" now replaces
`r.publish(old, d)`. "A write changes the published snapshot in place"
now starts the draft's entries on the old snapshot's array,
`s.entries[:0]`, and fails `TestSnapshotsUnderRace` under the race
detector. Step 2's set was not rerun: nothing under `when/` changed.

### Step 6: exporters

#### Deviations

Found before any Step 6 code was written; MADR amendment A7 records the
decisions. Each was asked with options, and the owner picked the
recommendation on 2026-10-05.

* **D24: `Loop` commands run from off the loop** (Step 3's open item).
  `WithLoop(send)` and `LoopMsg`: `Run`, and so `CallMCP`, hands a `Loop`
  command to the program's loop and waits. Two names added to the step
  (Rule 2), and the test `TestRunOnLoop`.
* **D25: what `ACPCommands` exports.** Slash commands only, under their
  slash names.
* **D26: `ManifestCommand` and Step 4's `commandInfo`.** One exported
  type; `command.list` and `command.describe` return `ManifestCommand`.

#### MCP 2026-07-28's field names

`schema/2026-07-28/schema.ts` was fetched on 2026-10-05 from the
`modelcontextprotocol/modelcontextprotocol` repository's `main` branch:
3197 lines, SHA-256
`742750af0bb8c716e7030c4977c992b55d1adc4407e9e66997db5846baedc2cd`. The
names, as `export_test.go` holds them:

* `Tool` (`BaseMetadata`, `Icons`): `name`, `title`, `icons`,
  `description`, `inputSchema`, `outputSchema`, `annotations`, `_meta`;
  required `name` and `inputSchema`, whose `type` is `"object"`.
* `ToolAnnotations`: `title`, `readOnlyHint` (default false),
  `destructiveHint` (default true), `idempotentHint` (default false),
  `openWorldHint` (default true). The last three mean something only when
  `readOnlyHint` is false.
* `CallToolResult` (`Result`): `_meta`, `resultType` (required: `"complete"`,
  `"input_required"` or another string), `content` (required),
  `structuredContent` (any JSON value), `isError`.
* `TextContent`: `type` (`"text"`), `text`, `annotations`, `_meta`;
  required `type` and `text`.

Because `destructiveHint` and `openWorldHint` default to true, the
exporter writes every hint that applies: `readOnlyHint` and
`openWorldHint` always, and `destructiveHint` and `idempotentHint` when
`readOnlyHint` is false. `Tool.title` carries the title, so the
annotations' `title` is not written.

#### What was built

* `command/mcp.go`: `MCPTool`, `MCPToolAnnotations`, `MCPCallResult`,
  `MCPContent`, `MCPTools`, `CallMCP`. `command/acpexport.go`:
  `ACPCommands`. `command/manifest.go`: `Manifest`, `ManifestCommand`,
  `(*Registry).Manifest`. `command/registry.go`: `WithLoop`.
  `command/messages.go`: `LoopMsg` and its `Run`. `command/dispatch.go`:
  `Run` hands a `Loop` command to the loop under `WithLoop` (D24).
  `command/builtin.go`: `ManifestCommand` replaces `commandInfo` (D26).
  No other exported name.
* Choices inside the listed names:
  * A command without an arguments schema is exported with
    `{"type": "object"}`, which MCP requires and which says nothing
    false. Its arguments are not checked.
  * `CallMCP`'s text is `Result.Text`, else `Result.Value` as JSON, as MCP
    recommends for a client that reads only text; `Result.Value` is
    `structuredContent`. A value that does not marshal is an error
    result. A result with neither has an empty `content`.
  * Each error result is one sentence the model can act on: no such
    tool, not available now, refused ("Do not retry it unchanged"), bad
    arguments ("Fix them and call again"), or failed.
  * `Manifest` has `format` 1 and the registry's version, from one
    snapshot, and includes hidden commands. `ManifestCommand` adds
    `argHint`, `scope`, `idempotent`, `openWorld`, `exclusive` and
    `whileBusy` to Step 4's fields.
  * Under `WithLoop`, `Run` sends the `LoopMsg` from a new goroutine,
    because `tea.Program.Send` blocks until the program takes the
    message. A `LoopMsg` that reaches the loop after its caller's context
    ended runs nothing.
* Tests beyond the table: `TestRunOnLoop` (D24).
* Found while building:
  * `TestExportFilters` expected ACP names in name order; the export is in
    ID order (`acp.agent.plan`, `session.reset`, `user.review`,
    `view.zoom`), and the test now says so.
  * The "`Run` ignores `WithLoop`" mutation was first killed only by
    `go test`'s ten-minute timeout, because `TestRunOnLoop` waited for the
    `LoopMsg` with no deadline. Its waits now give up after five seconds,
    and the mutation fails on `export_test.go:318: Run sent no LoopMsg`.

#### Checks (Rule 3)

1. `gofmt -l command`: no output.
2. `make pre-add-check FILES="<the thirty-four command/*.go files>"`:
   exit 0, "go-precheck: 34 file(s) clean in 1 module(s) (gofmt,
   golangci-lint, go vet, go test, go mod tidy, govulncheck)."
3. `make lint`: exit 0; `go fix -diff` silent, and "0 issues." for
   `GOOS=linux`, `darwin` and `windows`.
4. `GOWORK=off go test -race -count=1 ./...`: exit 0, 15 packages `ok`.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: exit 0, 15 packages
   `ok`.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: exit 0, 15 packages `ok`.
7. `GOWORK=off go mod tidy -diff`: exit 0, no output.
8. No fuzz target in this step.
9. Windows test host, go1.27.1 windows/amd64, on a copy of the tree:
   `make pre-add-check` exit 0 ("117 file(s) clean in 1 module(s)"),
   `make lint` exit 0, `make vuln` exit 0 ("No vulnerabilities found."),
   `GOWORK=off go test -count=3 -shuffle=on ./command/` exit 0.
10. `markdownlint-cli2` on `docs/README.md`: 0 issues (the configuration
    excludes MADR and PLAN files); the relative-link check of
    `docs/README.md` and this pair: 0 broken.
11. Identifier scan of the diff, `command/*.go` and the goldens, for the
    local account name, the test host's name, home-directory paths and
    the employer's domain: no match.

The eleven goldens were written with `-tuitest.update` and each was read:

* `mcp-tools`: nine tools in ID order, without the hidden, shell-only and
  unavailable commands, with the hints MADR §2 maps.
* `acp-commands`: the four slash commands in ID order, with their hints.
* `manifest`: format 1, version 3, every command, hidden ones too.
* The seven `mcp-call-*` results: text, a value with its JSON text, a
  prompt's expansion, and errors for a failure, bad arguments, an
  unknown name and a refusal.

**Benchmarks,** on the development host, now idle (load average 2.3),
three runs each: `BenchmarkLookup` 29.58–29.64 ns/op;
`BenchmarkAvailable` 15738–15986 ns/op; `BenchmarkDispatchLoop`
234.2–238.6 ns/op, 560 B/op, 5 allocs/op. These match Step 4's figures,
which confirms that the slowdown measured during Step 5 was the host's
load.

#### Mutations (Rule 4)

Each on a scratch copy of the tree. All nine killed: the step's five and
four for D24 and D25. D26 has none, because the compiler checks the type
the built-ins return.

| Mutation | Failing line |
| :--- | :--- |
| `Destructive` exports `destructiveHint: false` | `export_test.go:103: tuitest: mcp-tools …: line 124 differs` |
| `CallMCP` runs as `OriginProgram` | `export_test.go:224: the gate saw origins [] and callers []` |
| hidden commands are exported | `export_test.go:242: MCPTools exports secret.tool` |
| `resultType` is left out | `export_test.go:157: the result of view.zoom lacks the required "resultType"` |
| tools are not listed in ID order | `export_test.go:103: tuitest: mcp-tools …: line 3 differs` |
| D24: `Run` ignores `WithLoop` | `export_test.go:318: Run sent no LoopMsg` |
| D24: `Run` returns the `Cmd` it gave the program | `export_test.go:334: Run = {… Cmd:0x…}, <nil>; want the result without its Cmd` |
| D24: a `LoopMsg` runs after its caller gave up | `export_test.go:363: a LoopMsg ran after its caller gave up` |
| D25: commands without a slash name go to ACP | `export_test.go:257: ACPCommands = [plan      reset review zoom], want [plan reset review zoom] …` |

The lines above are from the final run, after `TestRunOnLoop`'s
deadlines were added. Because this step changed `dispatch.go`,
`registry.go`, `builtin.go` and `messages.go`, Step 3's eleven, Step 4's
thirteen and two single-layer mutations, and Step 5's sixteen ran again:
all killed. Step 2's set was not rerun: nothing under `when/` changed.

### Step 7: workspace integration

#### Deviations

Found before any Step 7 code was written; MADR amendment A8 records the
decisions. Each was asked with options, and the owner picked the
recommendation on 2026-10-06.

* **D27: what `workspace.hiddenPanes` holds.** `Plan().Hidden`.
* **D28: an unknown pane for `workspace.zoom` and `workspace.toggle`.** An
  `ArgError`, as for `focus` and `resize`.
* **D29: `workspace.panes`'s order and members.** The focus ring, then
  other placed panes in tree order, then `Plan().Hidden`; panes the tree
  never mentions are left out.
* **D30 (2026-10-06): `workspace.resize`'s `split`** (found while
  building, after D27–D29; MADR A8 item 4). `layout` names separators
  `<split>:<index>`, so `Resize("sidebar", …)`, which A2's example and this
  step's test assumed, moves nothing. Options given: a separator ID, or a
  split name with exactly one resizable separator on screen
  (recommended); separator IDs only. **The owner picked "A separator ID,
  or a name with one".**

#### What was built

* `workspace/commands.go`: `Commands`, `CommandOption`, `WithLayouts`, and
  the thirteen built-ins of the step's table, each `Loop`, offered on every
  surface but the shell, `UI` or `ReadOnly` as the table says, in the
  category `Workspace`. `workspace/context.go`: `Contexter`, the seven
  typed keys, `ContextKeys`, `(*Workspace).WhenContext`.
  `workspace/background.go`: `(*Workspace).SetBackground`.
  `workspace/workspace.go`: a `pinnedBg` field beside `bg`, which now
  holds only what the terminal last reported. `rebuildTheme` builds for
  the pinned background when there is one, and a `tea.BackgroundColorMsg`
  rebuilds only when none is pinned. No other exported name. `workspace`
  newly imports `command` and `when`, and nothing else
  (`go list -deps ./workspace`).
* Choices inside the listed names:
  * `workspace.focus` refuses a pane that cannot take focus now (one that
    is not placed, or a `Focusable` pane that says no, such as a footer),
    since `Focus` would do nothing.
  * `workspace.zoom` with no pane zooms the focused one, and so restores
    the layout when the focused pane is zoomed already, as `Zoom` does.
  * `workspace.state.set` restores the previous state when the layout
    cannot use the new one, and refuses it with an `ArgError` at `/state`.
    The workspace's own `SetState` keeps the bad state and reports it
    through `Err`. `layout.State`'s `version` is required in the schema,
    as its json tag has no `omitempty`.
  * `workspace.layout.use`'s enum is set by editing the schema `New`
    emits, because the layout names are known only at run time.
  * `workspace.panes` returns each pane's `id`, `title`, `x`, `y`,
    `width`, `height`, `focused` and `hidden`, with a hidden pane's
    rectangle zero. Its text is one line per pane.
  * `workspace.overlay.close` with no overlay open returns the text "No
    overlay is open." and no error.
  * `KeyWidth` and `KeyHeight` are the workspace's size, not the
    terminal's.
* Test code: `session` in `workspace/agent_test.go` takes extra
  options, so `TestCommandsGolden` can build the agent session following
  the background. Existing callers are unchanged. Tests beyond the table:
  `TestUnknownPanes` (D28), `TestStateSetRefusesBadState` and
  `TestContextKeys`. `TestAgentMayZoom` runs the zoom twice: once through
  `CallMCP` on the caller, and once through `WithLoop`, where the agent's
  goroutine waits until the test, acting as the loop, runs the `LoopMsg`
  (A7).
* Found while building:
  * D30.
  * In `TestWhenContextLayers`, the test pane first embedded another test
    pane, whose `Update` returned the embedded value. The workspace kept
    that, so the overlay lost its `Contexter`, and the pane was not
    focusable. The test pane now stands alone.
  * D29's mutation would have survived, because in the agent preset the
    focus ring follows the tree. `TestPanesCommand` gained a case with
    `WithFocusRing` in another order.

#### Checks (Rule 3)

1. `gofmt -l workspace`: no output.
2. `make pre-add-check FILES="<the eight workspace files>"`: exit 0,
   "go-precheck: 8 file(s) clean in 1 module(s) (gofmt, golangci-lint, go
   vet, go test, go mod tidy, govulncheck)."
3. `make lint`: exit 0; `go fix -diff` silent, and "0 issues." for
   `GOOS=linux`, `darwin` and `windows`.
4. `GOWORK=off go test -race -count=1 ./...`: exit 0, 15 packages `ok`.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: exit 0, 15 packages
   `ok`.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: exit 0, 15 packages `ok`.
7. `GOWORK=off go mod tidy -diff`: exit 0, no output.
8. No fuzz target in this step.
9. Windows test host, go1.27.1 windows/amd64, on a copy of the tree:
   `make pre-add-check` exit 0 ("123 file(s) clean in 1 module(s)"),
   `make lint` exit 0, `make vuln` exit 0 ("No vulnerabilities found."),
   `GOWORK=off go test -count=3 -shuffle=on ./workspace/` exit 0.
10. `markdownlint-cli2` on `docs/README.md`: 0 issues (the configuration
    excludes MADR and PLAN files); the relative-link check of
    `docs/README.md` and this pair: 0 broken.
11. Identifier scan of the diff, the new workspace files, the tests and
    the goldens, for the local account name, the test host's name,
    home-directory paths and the employer's domain: no match.

The twenty-four `commands-*` goldens were written with `-tuitest.update`
and read:

* `commands-zoomed` (no colour, ASCII, 80) and `commands-restored` (no
  colour, ASCII, 80, and no colour, UTF-8, 160) in full.
* Every file checked for 30 lines at its width, the footer the only
  other line, which zoomed frames do not show.
* `commands-light` compared with `commands-restored`: the colour files
  differ on all 29 bordered lines, and the no-colour files are
  identical. Choosing the light background changes colour and nothing
  else (0001-MADR §6, rule 4).

#### Mutations (Rule 4)

Each on a scratch copy of the tree; every failure is an assertion. All
eleven killed: the step's six, one for each of D27–D30, and one for the
`WithTheme` rule of `SetBackground`. This step changes nothing in
`command` or `when`, so the earlier steps' sets were not rerun.

| Mutation | Failing line |
| :--- | :--- |
| `WhenContext` puts the workspace keys first | `context_test.go:39: workspace.width = 60, true; want 1` |
| `workspace.panes` leaves hidden panes out | `commands_test.go:201: panes = [session logs footer], want [session logs footer metrics]` |
| `workspace.focus.next` moves twice | `commands_test.go:146: next: the frames differ:` |
| a background message overrides a pinned background | `background_test.go:46: a light report rebuilt a pinned dark theme: [2]` |
| `auto` sends no query | `background_test.go:63: auto does not ask the terminal again` |
| `workspace.resize` accepts any split | `commands_test.go:272: split "session": <nil>, want an ArgError at /split` |
| D27: `workspace.hiddenPanes` holds `State().Hidden` | `context_test.go:64: workspace.hiddenPanes = [metrics], want Plan().Hidden [metrics logs footer]` |
| D28: zoom and toggle accept an unknown pane | `commands_test.go:282: workspace.zoom of an unknown pane: <nil>, want an ArgError at /pane` |
| D29: `workspace.panes` ignores the focus ring | `commands_test.go:226: with a focus ring: panes = [session metrics logs footer], want [logs metrics session footer]` |
| D30: a split's name is not resolved | `commands_test.go:234: command: argument /split: is not a resizable split on screen` |
| `SetBackground` changes a `WithTheme` workspace | `background_test.go:85: 0 on a WithTheme workspace returned a command` |

### Step 8: `command/cli`

#### Deviations

Found before any Step 8 code was written; MADR amendment A9 records the
decisions. Each was asked with options, and the owner picked the
recommendation on 2026-10-06.

* **D31: how `--yes` and `WithConfirm` reach the policy** (Step 3's open
  item). `command.Request` gains `Gate`, a per-request gate the policy
  asks instead of the registry's. Added to the step: `command/handler.go`,
  `command/gate.go`, and `TestRequestGate` in `command/gate_test.go`.
* **D32: an object or map property's flag.** It takes JSON.
* **D33: a command whose `When` is false.** Exit 1.
* **D34: `--json` with no value.** It prints `null`.

#### What was built

* `command/cli/cli.go`: the package documentation, `Run`, `Option`,
  `WithName`, `WithConfirm`, `WithContext`, `WithWidth`, `ExitOK`,
  `ExitFailed`, `ExitUsage`, `ExitRefused`, the verbs and the
  confirmation gate. `command/cli/flags.go`: reading a schema's
  properties in document order, and parsing flags and positionals with
  the standard `flag` package, as MADR §1 says. `command/cli/help.go`:
  plain ASCII help, wrapped with `x/ansi` at the width.
  `command/cli/example_test.go`: `ExampleRun`. In `command` (D31):
  `Request.Gate`; `decide` asks a request's gate when it has one, and the
  shared refusal logic moved into `verdict`. No other exported name.
* Choices inside the listed names:
  * `WithName`'s default is `app`: the package reads no process state,
    so it cannot take the program's name itself.
  * `Run` gathers its output, and writes stdout and then stderr once each
    when the command ends. A write that fails makes the exit code
    `ExitFailed`. This follows `termcap.Report`'s pattern, and is what
    `errcheck` asks for.
  * A command is named by the longest run of leading words that, joined
    with dots, is an ID offered on the CLI surface. The rest are its
    arguments. The dotted ID works too.
  * Flags take one or two dashes, `--name value` or `--name=value`, as
    `flag` reads them. A boolean takes no value. A short name from
    `x-cli` is a second flag. Positionals may come between flags. A
    property named like one of the shell's own flags (`args`, `json`,
    `yes`, `help`, `h`) is a usage error, not a silent clash. `--args`
    gives the object first, and flags and positionals are set over it.
  * Running with no arguments writes the registry's help to stderr and
    exits 2; `help` writes it to stdout and exits 0. `list` lists what
    `Available` gives for the CLI surface in the shell's context, and
    `list --json` the manifest of the same commands. `describe` writes a
    command's `ManifestCommand` as JSON, and `schema` its arguments
    schema, `{"type":"object"}` when it has none. Help lists every
    command offered on the CLI that is not hidden, whatever its `When`,
    grouped by category, with commands without one under `Other`, last.
  * A refused destructive command adds "pass --yes to run it" to the
    error, and the confirmation's prompt names the command and its
    danger. The caller recorded for a shell request is the program's
    name.
* Tests beyond the table: `TestExitCodes` (D33), `TestVerbs`, and
  `TestRequestGate` (D31) in `command`.
* Found while building:
  * The registry help's second usage line, one line for every verb, was
    80 columns at a 60-column width. Each verb now has its own line, and
    the command columns line up across categories.
  * Two test expectations were wrong: `--json` writes the value with its
    json tags, and the example's help wraps at the default 80 columns.
  * Lint: `errcheck` on every `Fprintf` to an `io.Writer`, hence the
    gathered output; `goconst` (type-name constants); an unused helper;
    revive's `confusing-naming` (`set` beside `Set`, now `assign`);
    `gocritic`'s `typeDefFirst`.

#### Checks (Rule 3)

1. `gofmt -l command`: no output.
2. `make pre-add-check FILES="<the five command/cli files and
   command/handler.go, gate.go, dispatch.go, gate_test.go>"`: exit 0,
   "go-precheck: 9 file(s) clean in 1 module(s) (gofmt, golangci-lint, go
   vet, go test, go mod tidy, govulncheck)."
3. `make lint`: exit 0; `go fix -diff` silent, and "0 issues." for
   `GOOS=linux`, `darwin` and `windows`.
4. `GOWORK=off go test -race -count=1 ./...`: exit 0, 16 packages `ok`,
   `command/cli` among them; `internal/conformance` passes with it in its
   scan.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: exit 0, 16 packages
   `ok`.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: exit 0, 16 packages `ok`.
7. `GOWORK=off go mod tidy -diff`: exit 0, no output.
8. No fuzz target in this step.
9. Windows test host, go1.27.1 windows/amd64, on a copy of the tree:
   `make pre-add-check` exit 0 ("128 file(s) clean in 1 module(s)"),
   `make lint` exit 0, `make vuln` exit 0 ("No vulnerabilities found."),
   `GOWORK=off go test -count=3 -shuffle=on ./command/cli/` exit 0.
10. `markdownlint-cli2` on `docs/README.md`: 0 issues (the configuration
    excludes MADR and PLAN files); the relative-link check of
    `docs/README.md` and this pair: 0 broken.
11. Identifier scan of the diff, `command/cli/` and its goldens, for the
    local account name, the test host's name, home-directory paths and
    the employer's domain: no match.

The six goldens, `help-registry`, `help-resize` and `help-echo` at 60 and
100 columns, were written with `-tuitest.update`, each was read in full,
and each line was checked to fit its width. `TestHelpGolden` also fails
on any byte outside ASCII.

#### Mutations (Rule 4)

Each on a scratch copy of the tree; every failure is an assertion. All
eight killed: the step's four and one for each of D31–D34. Because this
step changed `command/gate.go`, `dispatch.go` and `handler.go`, Step 3's
eleven, Step 4's thirteen and two single-layer mutations, Step 5's
sixteen and Step 6's nine ran again: all killed. Step 3's "`AllowAlways`
is not remembered" anchor moved into the new `decide`. Step 7's set
anchors in `workspace` only, which this step does not change.

| Mutation | Failing line |
| :--- | :--- |
| `--yes` is not required | `cli_test.go:203: without --yes: exit 0, stdout "deleted notes.txt\n", stderr ""` |
| a usage error exits 1 | `cli_test.go:181: tools echo --name x --on=maybe: exit 1, …; want exit 2 naming "argument /on"` |
| hidden commands are listed | `cli_test.go:127: tuitest: help-registry.60 …: line 21 differs` |
| `--json` writes `Result.Text` | `cli_test.go:189: --json: "\"moved sidebar\"\n"` |
| D31: a request's own gate is ignored | `gate_test.go:180: the request's gate refusing: <nil>; it was asked 0, the registry's 1` |
| D32: an object flag is not read as JSON | `cli_test.go:162: exit 2 "app: command: argument /state: is a string, not object …"` |
| D33: an unavailable command exits 2 | `cli_test.go:267: "tools later": exit 2, want 1` |
| D34: `--json` without a value prints the text | `cli_test.go:195: --json with no value: "\"hello\"\n", want null` |

### Step 9: documentation, and the root release `v0.4.0`

No deviation: the step's text named every file.

#### What was written

* `docs/guides/commands.md` (new): defining a command with `New` and the
  tags; kinds, danger, the gate and the audit; dispatching from `Update`,
  `Run`, and `WithLoop` with `LoopMsg`; availability and `when`; slash
  commands; command files and their front matter; MCP prompts and ACP
  commands, and slash-name clashes; exporting to an agent, with what an
  MCP server still owns under 2026-07-28 (the `tools/list` envelope with
  `resultType`, `ttlMs` and `cacheScope`, and
  `notifications/tools/list_changed` on a `subscriptions/listen` stream);
  the workspace's commands, `WhenContext` and `SetBackground`; and
  `command/cli`. Each claim was checked against the exported API
  (`go doc -short` of `command`, `when`, `command/cli` and the new
  `workspace` names) and the code; the envelope fields against the
  `schema.ts` recorded in Step 6.
* `docs/architecture.md`: eleven packages; `when`, `command` and
  `command/cli` in the package graph and table; `workspace`'s new imports
  and names; why imports point downward, why reads are lock-free, and why
  no protocol SDK is imported; the tree; `make fuzz`'s four targets and
  CI's three corpora; the keymap, the palette and the front ends under
  "What is not here".
* `docs/README.md`: five "I want to…" rows into the guide, and the
  MADR's row reworded from "the planned command registry".
* `README.md`: Status names `v0.4.0` and the commands, and the "I want
  to…" table links the guide.
* Fixed before the checks: the guide's example first bound `alt+z`, which
  the workspace already uses, and its shell example ran a workspace
  command, which the shell is not offered.

#### Checks

* Rule 3: `gofmt -l` silent; `make lint` exit 0 ("0 issues." for
  `GOOS=linux`, `darwin` and `windows`); `GOWORK=off go test -race`,
  `-shuffle=on -count=2` and under `LC_ALL=C`, each 16 packages `ok`;
  `go mod tidy -diff` silent; `make fuzz FUZZTIME=20s` exit 0 ("ran clean"
  in `./layout`, `./when`, and two targets in `./command`); `make vuln`
  "No vulnerabilities found."; `markdownlint-cli2` on `README.md`,
  `docs/README.md`, `docs/architecture.md` and the guide: 0 issues; the
  relative-link check of those and this pair: 0 broken; the identifier
  scan of the diff and the guide: no match.
* `make release-check`: exit 0, "go-precheck: 128 file(s) clean in 1
  module(s) (gofmt, golangci-lint, go vet, go test, go mod tidy,
  govulncheck)."
* Windows test host, go1.27.1 windows/amd64, on a copy of the tree: `make
  pre-add-check`, `make lint` and `make vuln` exit 0, and `GOWORK=off go
  test -count=3 -shuffle=on ./...` exit 0.
* `git diff v0.3.0 -- go.mod go.sum`: empty. The root module's
  requirements are unchanged from `v0.3.0`, and it names neither Cobra nor
  Kong.

#### Release notes for `v0.4.0`

`v0.4.0` adds a command registry: one definition of each action, run from
a key, a slash line, the shell or an agent. Nothing that `v0.3.0` exported
changes, and no module is added.

* **`when` (new):** availability expressions in VS Code's when-clause
  grammar: `!`, `&&`, `||`, comparisons, `=~` with RE2 regexes, and
  `in`. It parses at most 4 KiB, nests at most 64 deep, and has typed
  keys and `Check`, which reports a typo at load.
* **`command` (new):**
  * `New[A]` builds a command whose arguments are a Go struct: JSON
    Schema 2020-12 from Kong-aligned tags, checked and strictly decoded.
  * `Registry` holds the commands in lock-free snapshots, and runs them
    from `Update` (`Dispatch`) or to completion (`Run`). The default
    policy lets agents run only read-only and UI commands, and the shell
    destructive ones only when confirmed, unless the program's `Gate`
    allows.
  * It audits every request through `slog`, with secret arguments
    masked.
  * It loads Markdown command files through an `fs.FS`, MCP prompts and
    ACP commands, and renames slash-name clashes.
  * It exports the commands as MCP 2026-07-28 tools and results, ACP
    available commands, and a manifest.
  * `WithLoop` runs an agent's call on the program's event loop.
* **`command/cli` (new):** the same commands as shell subcommands:
  * words or dotted IDs, flags from the schemas, `--args`, `--json` and
    `--yes`;
  * the verbs `list`, `describe`, `schema` and `help`;
  * exit codes 0 to 3.
* **`workspace`:**
  * `Commands` publishes thirteen built-in commands: focus, zoom, toggle,
    resize, layouts, state, panes, overlay and theme.
  * `WhenContext` gives the keys their `When` reads, with the
    `Contexter` interface and typed keys.
  * `SetBackground` pins the theme's background, or returns to following
    the terminal.
* **Docs:** [guides/commands.md](../guides/commands.md).

#### The release

Left to the owner, as Rule 8 and AGENTS.md say: commit this step, push,
and with CI green tag `v0.4.0`. Then the agent runs the consumer smoke
test of [releasing.md](../guides/releasing.md#the-consumer-smoke-test)
against `v0.4.0`, importing `command`, `when` and `command/cli`, and
records it here. The step is done when the tag exists and the smoke test
builds.

**After the release (2026-10-06).** The owner committed and pushed this
step (`33974e6`), CI was green, and the owner tagged `v0.4.0` (annotated)
on it. The agent ran the releasing guide's consumer smoke test against the
tag, in a scratch module outside the repository, with no `go.work` in
effect (`go env GOWORK` empty) and `GOPROXY` at its default:

* The program imports `command`, `when` and `command/cli`. It parses a
  `when` expression and evaluates it, registers a `ReadOnly` command made
  with `command.New`, and hands its arguments to `cli.Run`.
* `go mod init`, `go get github.com/maccavelli/go-tui-lib@v0.4.0`
  ("go: downloading github.com/maccavelli/go-tui-lib v0.4.0"), `go mod
  tidy`, `go build ./...` and `go vet ./...`: each exit 0. `go.mod`
  requires `github.com/maccavelli/go-tui-lib v0.4.0`.
* Run: the `when` line printed `mode == 'shell' && !busy`, the keys
  `[mode busy]` and `eval: true`; `list` printed the three commands and
  exited 0; `greet world` printed `hello world`, and with `--json`
  `{"greeting":"hello world"}`, each exit 0; an unknown command printed
  `smoke: unknown command "nosuch"` and exited 2, the usage-error code.

The tag exists and the smoke test builds, so Step 9 is done. Steps 10 and
11 may start: each requires the published `v0.4.0`.

### Step 10: `command/cobracmd`, the Cobra front end

#### Before the first file

Read on 2026-10-06, with `GOWORK=off`, from the module proxy:

* **Versions.** `go list -m -versions`: the newest `github.com/spf13/cobra`
  is `v1.10.2` (2025-12-03) and the newest `github.com/spf13/pflag`
  `v1.0.10` (2025-09-02), the versions 0010-MADR §5 named.
* **`go.mod`s.** Cobra `v1.10.2`: `go 1.15`, requiring
  `github.com/cpuguy83/go-md2man/v2 v2.0.6`,
  `github.com/inconshreveable/mousetrap v1.1.0`,
  `github.com/spf13/pflag v1.0.9` and `go.yaml.in/yaml/v3 v3.0.4`.
  pflag `v1.0.10`: `go 1.12`, no requirement. go-md2man `v2.0.6`: `go
  1.12`, requiring `github.com/russross/blackfriday/v2 v2.1.0`.
* **Licences,** from each module's licence file: Cobra Apache-2.0;
  pflag BSD-3-Clause; mousetrap Apache-2.0; go-md2man MIT; blackfriday
  BSD-2-Clause; `go.yaml.in/yaml/v3` MIT and Apache-2.0 (its `NOTICE`).
* **`govulncheck`.** The module does not exist before its first file, so
  a scratch module outside the repository stood in for it: a program
  importing `cobra`, `cobra/doc`, `pflag`, and `command`, `when`, `theme`
  and `glyph` at `v0.4.0`, built with `go mod tidy`. `GOWORK=off
  govulncheck ./...`: "No vulnerabilities found." The check runs again in
  the module itself under Rule 3.
* **Cobra, probed** in a second scratch module: a group added after its
  child and before `Execute` works; a group never added panics at
  `Execute` ("group id 'g' is not defined for subcommand 'app leaf'"); a
  group added twice is listed twice; a command with children and no `Run`
  prints its help to stdout and returns no error, given no argument or an
  unknown one; a shorthand given to two flags panics ("unable to redefine
  'x' shorthand"); `-4` after a positional is an unknown shorthand.
  `completions.go:38` still keeps flag completions in a process-wide map,
  and `RegisterFlagCompletionFunc` refuses a second registration for one
  flag (`completions.go:178`), as A1 says.

#### Deviations

Found before any Step 10 code was written; MADR amendment A10 records the
decisions. Each was asked with options, and the owner picked the
recommendation on 2026-10-06.

* **D35: hidden commands.** In the tree, with `Hidden` set. The test
  becomes "no non-CLI command in the tree, and a hidden one is hidden,
  and runs".
* **D36: required properties.** No `MarkFlagRequired`; the registry
  reports a missing argument, exit 2.
* **D37: deprecation, completion providers, flag relations.** None is
  built; the registry has none of them.
* **D38: dotted IDs.** `Run` turns a dotted ID into its words.
* **D39: the group mutation.** It survives in Cobra, so it is replaced
  by "a category's group is never added" and "`Mount` adds a group the
  parent already has"; A1's wording is corrected.
* **D40: `Mount` and the verbs.** `Mount` adds them, and a name clash is
  an error before the parent changes.

#### What was built

* **The module.** `command/cobracmd/go.mod`: `module
  github.com/maccavelli/go-tui-lib/command/cobracmd`, `go 1.27.1`,
  requiring `github.com/maccavelli/go-tui-lib v0.4.0`,
  `github.com/spf13/cobra v1.10.2` and `github.com/spf13/pflag v1.0.10`,
  and directly `github.com/charmbracelet/colorprofile v0.4.3` and
  `github.com/charmbracelet/x/ansi v0.11.8`, which the root already
  requires (0001-MADR A3); no `replace`. `go.sum`. `go work use
  ./command/cobracmd`.
* **The package.** `cobracmd.go`: the package documentation, `Option`
  and the six options, `New`, `Mount`, `Run`, the tree, the
  annotations, the confirmation gate and the exit codes. `flags.go`:
  reading a schema's properties as `command/cli` does, copied, because a
  nested module cannot reach `command/cli`'s unexported code; and the
  pflag values. `help.go`: the verbs `list`, `describe` and `schema`, the
  help command, and help drawn with the theme. `complete.go`: the
  completion command and the completion functions. `docs/docs.go`: `Man`
  and `Markdown`.
* **Exported names,** checked with `go doc -short`: `New`, `Mount`,
  `Run`, `Option`, `WithName`, `WithConfirm`, `WithContext`,
  `WithTheme`, `WithGlyphs`, `WithWidth`; in `docs`, `Man` and
  `Markdown`. No other.
* **Tests.** `frontend_test.go`: `TestSharedFrontEndCases`, 51 command
  lines run through `command/cli` and this package, comparing the exit
  code, stdout, whether stderr is empty, and the arguments each command
  ran with. `cobracmd_test.go`: `TestTree` (no non-CLI command in the
  tree, a hidden one hidden and runnable, the annotations, `Use`,
  aliases and flags), `TestGroups`, `TestMount` (beside a program's own
  commands, by ID below the root, a clash refused with the parent
  unchanged, a group the parent has not added twice), `TestEnumFlag` (an
  enum flag refuses, and flags, positionals and `describe` complete),
  `TestHelpGolden`, `TestHelpDefaults`, `TestCompletionScripts`,
  `TestExitCodesAndNamespaces`, `TestRunsAreIndependent`,
  `TestDestructiveNeedsYes`, `TestNoProcessState` (no `os.Args`, and
  nothing on the process's stdout or stderr through a pipe),
  `TestWriteFails`, `TestSecondNew` (three trees in one process, each
  completing), `TestBrokenSchema` and `TestNilArguments`.
  `example_test.go`: `ExampleNew` and `ExampleMount`. `docs/docs_test.go`:
  `TestPagesGolden` (each page twice, equal, with no generated-by line,
  and the caller's `DisableAutoGenTag` left as it was) and
  `TestNilArguments`.
* **Goldens,** all read: 56 help files (the program, a namespace, three
  registry commands, a destructive one and a verb, each across {colour,
  no colour} × {UTF-8, ASCII} × {60, 100}); the four completion scripts;
  the man and Markdown pages of the root and of `workspace resize`. A
  script compared every colour file, with its escapes removed, against
  its no-colour twin, and every UTF-8 file against its ASCII twin: they
  differ only in the escapes and in the ellipsis, and no ASCII file has a
  non-ASCII rune.
* **Docs.** `AGENTS.md`'s module table and `docs/architecture.md`'s
  Modules section, tree and "What is not here" no longer call the module
  planned. `README.md` said "The planned Cobra, Kong and glamour
  adapters"; it now names `command/cobracmd` as built.
* **Choices inside the listed names:**
  * The exit codes are `command/cli`'s constants, which this package
    imports; it exports none of its own.
  * The defaults are `command/cli`'s: the name `app` and a width of 80,
    at least 40. The default theme is built for no terminal
    (`colorprofile.NoTTY`, an unknown background, ASCII glyphs), so help
    is plain ASCII as `command/cli`'s is. `WithGlyphs` replaces the
    theme's glyphs.
  * Help follows `command/cli`'s layout. Headings take the theme's
    title style, the left column its accent, and a destructive command's
    danger line its warning style, which also says in words that it asks
    first. A property's `x-cli` group heads its flags, as in
    `command/cli`, though Cobra itself groups only commands. A repeated
    positional ends in the glyph table's ellipsis: `…`, or `~` in ASCII.
  * The root and each namespace run: with no word they write their help
    to stderr and exit 2, and with an unknown word they exit 2, as
    `command/cli` does. Cobra's default prints help and exits 0. `New`'s
    root has its own help command for the same reason: Cobra's prints the
    program's help for an unknown topic and exits 0.
  * A namespace's short description is `The <word> commands`. It is
    hidden when every command below it is, and in a group when every
    visible command below it shares one category.
  * A slash alias becomes a Cobra alias unless it names a sibling or
    something already taken, or holds a space or a dot.
  * A property named like one of the shell's flags, or a schema that
    cannot be read, leaves the rest of the tree whole: that command alone
    exits 2 when it runs, as in `command/cli`. A short name is kept only
    when it is one ASCII letter other than `h`, and not taken.
  * An enum flag refuses a value when it is parsed, and a positional enum
    when Cobra checks the arguments; both report the `ArgError` and exit
    2.
  * `Run` writes to the caller's writers as output comes, where
    `command/cli` gathers it, so a program's own long-running command
    streams; a write that fails still makes the exit code 1. `Run` takes
    nil arguments as none, since Cobra reads `os.Args` for nil.
  * An error from a command the program built itself is exit 1. A flag
    error of a command this package built is exit 2.
  * `Mount` also refuses `help` and `completion` when the parent is a
    root, the names Cobra adds there. It sets the help and flag-error
    functions of the commands it adds, and leaves the parent's alone.
  * `docs` writes the page of the command it is given, in section 1,
    with Cobra's own link names; it sets `DisableAutoGenTag` for that
    page and puts it back.
* **Found while building:**
  * Cobra's default `completion` command keeps the writer of the run
    that creates it (`completions.go`, `out := c.OutOrStdout()`).
    `TestCompletionScripts` ran `completion bash`, then `completion zsh`,
    on one tree, and zsh's script went to the first run's buffer: stdout
    `""`. `New` turns Cobra's off (`CompletionOptions.DisableDefaultCmd`)
    and adds its own, which calls the same generators with the run's
    writer. A1's "Scripts … come from Cobra's generators, written to the
    caller's writer" holds; the package documentation tells a program
    that mounts into its own root.
  * Cobra keeps every flag's value and `Changed` between executions, so
    `--yes` in one run approved the next. `Run` clears the flags of the
    commands `New` and `Mount` built before each run.
  * The man page showed a boolean property as `--quiet[=]`; its flag's
    default now reads `false`.
  * `x/ansi.Wordwrap` breaks at a hyphen: at 60 columns, `list`'s help
    wraps `--json` as `--` and `json`. `command/cli`'s help uses the same
    wrap and does the same. It is left as it is here, so that both front
    ends wrap alike, and is open for a later record.
  * Lint: `errcheck` on `fmt.Fprintf` to the help page (now `printf` on
    its embedded `strings.Builder`); `goconst` (`bool`, `JSON`);
    `nilerr` in the arguments check, which leaves a broken command to
    `RunE` (restructured); `gofmt`'s alignment; `go fix`'s
    `errors.AsType`.

#### Checks (Rule 3)

1. `gofmt -l command/cobracmd`: silent.
2. `make pre-add-check FILES="<the 9 Go files>"` in a scratch clone with
   the step staged, because `scripts/go-modules.sh --check` reads the
   index and the agent stages nothing in the tree: exit 0, "go-precheck:
   9 file(s) clean in 1 module(s) (gofmt, golangci-lint, go vet, go test,
   go mod tidy, govulncheck)", which includes the nested module's checks
   of no `replace` and a release version of the root.
3. `make lint`: exit 0, "0 issues." for `GOOS=linux`, `darwin` and
   `windows` in each of the two modules, after `make modernize`.
4. `GOWORK=off go test -race -count=1 ./...`: the root 16 packages `ok`,
   `command/cobracmd` 2.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: the same.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: the same.
7. `GOWORK=off go mod tidy -diff`: silent in both modules. The tests
   again in workspace mode: both modules `ok`. `GOWORK=off govulncheck
   ./...` in `command/cobracmd`: "No vulnerabilities found."; `make
   vuln`: the same for both modules. `scripts/go-modules.sh --check` and
   `make release-check` in the staged clone: exit 0, "go-precheck: 137
   file(s) clean in 2 module(s)". `internal/conformance`:
   `TestNoPackageOwnsTheTerminal/command/cobracmd` passes.
8. No fuzz target was added.
9. Windows test host, on a copy of the tree, go1.27.1 windows/amd64:
   `make pre-add-check`: exit 0, "go-precheck:
   137 file(s) clean in 2 module(s)"; `make lint`: exit 0, "0 issues."
   for each module and target; `make vuln`: exit 0, "No vulnerabilities
   found." in both modules; in `command/cobracmd`, `GOWORK=off go test
   -count=3 -shuffle=on ./...`: both packages `ok`, and the tests in
   workspace mode: both `ok`.
10. `markdownlint-cli2 --config .markdownlint-cli2.jsonc` on `AGENTS.md`,
    `README.md`, `docs/README.md` and `docs/architecture.md`: 0 issues;
    the relative-link check of those and this pair: 0 broken.
11. The identifier scan of the diff and the new module, for the local
    account name, the host names and real-machine paths: no match.

#### Mutations (Rule 4)

Each on a scratch copy of the tree; every one killed. S10-1's "type from
the Go field" has no field to read here, since the package sees only the
schema: the mutation ignores the schema's type instead.

| Mutation | Test | Failing line |
| :--- | :--- | :--- |
| S10-1: every value parsed as a string, ignoring the schema's type | `TestSharedFrontEndCases` | `"workspace resize --split sidebar --delta 4": exit 2, command/cli 0` |
| S10-2: `Destructive` runs without `--yes` | `TestDestructiveNeedsYes` | `without --yes: exit 0, stdout "deleted notes.txt\n"` |
| S10-3: `Run` without `SetArgs` | `TestNoProcessState` | `list ran something else: exit 0 "hello\n"` |
| S10-4 (D39): a category's group never added | `TestGroups` | `the root's groups are []`, then Cobra's `panic: group id 'Files' is not defined for subcommand 'app files'` |
| S10-5 (D39): `Mount` adds a group the parent has | `TestMount` | `the program's group Workspace is there 2 times` |
| S10-6: `Run` clears no flag | `TestRunsAreIndependent` | `--yes outlived its run: exit 0` |
| S10-7 (D38): no dotted ID | `TestSharedFrontEndCases` | `"workspace.resize --args …": exit 2, command/cli 0` |
| S10-8 (D35): a hidden command left out | `TestTree` | `a hidden command is missing, or not hidden: <nil>` |
| S10-9 (D36): a required property `MarkFlagRequired` | `TestSharedFrontEndCases` | `"workspace resize sidebar 4": exit 1, command/cli 0` |
| S10-10 (D40): `Mount` changes the parent before a clash | `TestMount` | `a failed Mount changed the program's commands` |
| S10-11: depguard allows Cobra in the root | `golangci-lint` on a root package importing Cobra | real config: `import 'github.com/spf13/cobra' is not allowed from list 'cobra'`; with the `cobra` rule removed, no depguard finding |

No earlier step's mutation anchors moved: Step 10 changes no root Go
file.

#### The release

Left to the owner: commit this step, push, and with CI green tag
`command/cobracmd/v0.1.0`. Then the agent runs the consumer smoke test of
[releasing.md](../guides/releasing.md#the-consumer-smoke-test) against
it, and records it here. The step is done when the tag exists and the
smoke test builds.

**After the release (2026-10-06).** The owner committed this step
(`8c8de7c`), ran the disclosure guard over the outgoing commits through
the agent (exit 0, no finding), pushed, and with CI green tagged
`command/cobracmd/v0.1.0` (annotated) on `8c8de7c` and pushed the tag.
The agent ran the releasing guide's consumer smoke test against it, in a
scratch module outside the repository, with no `go.work` in effect (`go
env GOWORK` empty) and `GOPROXY` at its default:

* The program imports `command`, `command/cobracmd` and
  `command/cobracmd/docs`. It registers a `ReadOnly` command made with
  `command.New`, builds the tree with `cobracmd.New`, and runs its
  arguments with `cobracmd.Run`, or writes the root's man page with
  `docs.Man`.
* `go mod init`, `go get
  github.com/maccavelli/go-tui-lib/command/cobracmd@v0.1.0` ("go:
  downloading github.com/maccavelli/go-tui-lib/command/cobracmd v0.1.0"),
  `go mod tidy`, `go build ./...` and `go vet ./...`: each exit 0.
  `go.mod` requires `github.com/maccavelli/go-tui-lib v0.4.0` and
  `github.com/maccavelli/go-tui-lib/command/cobracmd v0.1.0`, with Cobra
  `v1.10.2` and pflag `v1.0.10` indirect.
* Run: `greet one world` printed `hello world`; `greet.one world --json`
  printed `{"greeting":"hello world"}`; `list` listed the commands;
  `completion bash` wrote the script for `smoke`; each exit 0. `nosuch`
  printed `smoke: unknown command "nosuch"` and exited 2. `man` wrote a
  page headed `.TH "SMOKE" "1" "Oct 2026"`.
* `go list -deps` of the program finds three packages of go-md2man,
  blackfriday and yaml; of `command/cobracmd` alone, none: only an
  importer of `cobracmd/docs` compiles them, as A1 says.

The tag exists and the smoke test builds, so Step 10 is done.

### Step 11: `command/kongcmd`, the Kong front end

#### Before the first file

Read on 2026-10-06, with `GOWORK=off`, from the module proxy:

* **Version.** The newest `github.com/alecthomas/kong` is `v1.16.1`
  (2026-08-09), the version 0010-MADR §5 named.
* **`go.mod`s.** Kong `v1.16.1`: `go 1.20`, requiring
  `github.com/alecthomas/assert/v2 v2.11.0` and
  `github.com/alecthomas/repr v0.5.2`, and indirectly
  `github.com/hexops/gotextdiff v1.0.3`, all three used only by Kong's
  tests. `assert/v2 v2.11.0`: `go 1.18`, requiring `repr v0.4.0` and
  `gotextdiff v1.0.3`. `repr v0.5.2`: `go 1.18`, no requirement. In a
  consumer they reach `go.sum` and nothing that is compiled.
* **Licences,** from each module's licence file: Kong MIT; assert/v2
  MIT; repr MIT; gotextdiff BSD-3-Clause.
* **`govulncheck`,** in a scratch module standing in for the new one: a
  program importing `kong`, and `command`, `command/cli`, `when`, `theme`
  and `glyph` at `v0.4.0`. `GOWORK=off govulncheck ./...`: "No
  vulnerabilities found."
* **Kong, probed** in a second scratch module: a `reflect.StructOf`
  grammar mounts as a `DynamicCommand`, and its custom `id` tag reads back
  from `ctx.Selected()`; the parse's `Path` holds each flag and positional
  given, and a resolved flag marked `Resolved`, but no default; an `arg`
  and a flag may share a name; an enum neither required nor defaulted
  fails `kong.New`; a command cannot mix positional arguments with
  children, and `default:"withargs"` on a hidden child selects it; a
  parent given alone wants a child; two commands of one name raise no
  error; `--help` under an `Exit` that panics stops the parse, and under
  one that returns the parse goes on and fails ("unknown flag --bogus");
  a slice flag splits at commas unless `sep:"none"`; a resolver's value
  loses to the command line; `-4` is an unknown flag; Kong interpolates
  `${…}` in help, defaults and enums, and `$$` is a dollar.

#### Deviations

Found before any Step 11 code was written; MADR amendment A11 records the
decisions. Each was asked with options on 2026-10-06; the owner picked
the recommendation for all but D42.

* **D41: `Resolver`'s source.** The `config.*` keys of `WithContext`'s
  context; the command line wins.
* **D42: the grammar.** Kong-native, the owner's choice over the
  recommendation (a grammar mirroring `command/cli`). The shared cases
  leave out `--args` supplying a required or positional property and a
  positional property given as its flag, and a test asserts that
  `kongcmd` refuses each with exit 2 where `command/cli` accepts it.
* **D43: completion.** Scripts that ask the program through
  `__complete`.
* **D44: writers.** The caller's `kong.Writers`; `Parser` discards until
  given them.
* **D45: a command that is also a parent.** Its own hidden default child.
* **D46: name clashes.** A `PostBuild` hook fails `kong.New`.
* **D47: one struct, two readers.** Found while writing Step 11's
  tests. Read directly, Kong and `SchemaOf` disagree on requiredness
  (Kong's `required:""` and `optional:""`) and on names Kong spells
  differently (`max_items`, `max-items`); through the adapter all five
  agree. The recommendation was to test both ways and record the gaps.
  The owner chose to teach the registry Kong's tags (MADR A12, its Q1 to
  Q4 answered as recommended): Step 11a, a root change released as
  `v0.5.0`, after which this module requires `v0.5.0`.

Step 11's code (`command/kongcmd/`: `kongcmd.go`, `grammar.go`,
`help.go`, `resolver.go`, `complete.go`, `go.mod`, `go.sum`) is written
and builds against `v0.4.0`, and has no tests yet. It stays uncommitted and
out of `go.work` until Step 11a is released.

### Step 11a: `New[A]` agrees with Kong, and the root release `v0.5.0`

No deviation: the step was written with MADR A12 and approved as
written (2026-10-06).

#### What was built

* `command/kongname.go` (new): `kongName`, Kong's spelling of a Go name,
  written from Kong's documented behaviour (split where the kind of
  character changes, an upper-case run giving its last letter to a
  lower-case word, dashes, lower case); `agreeWithKong` and its helpers,
  which walk the top-level fields as `SchemaOf`'s `structFields` does,
  flattening an embedded struct as json and Kong both do, and skip a type
  with its own `JSONSchema`.
* `command/args.go`: `New` calls the check after `SchemaOf`, and its
  documentation states the rule; no exported name was added.
  `SchemaOf` is unchanged.
* `workspace/commands.go`: `zoomArg.Pane` gains `optional:""` and
  `stateArg.State` `required:""`, the two production fields the census
  found. Neither changes a schema, a golden file or a behaviour: the
  tags only say to Kong what `json` already said.
* Test fixtures that `New` now refused, each given the tag its error
  named: `command/decode_test.go` (`strictArgs.Name`, `.Delta`,
  `args.User`, `.Token`: `required:""`), `command/export_test.go`
  (`zoomArgs.Pane`: `optional:""`), `command/cli/cli_test.go`
  (`kindsArgs.Name`: `required:""`), and
  `command/cobracmd/frontend_test.go` (`kindsArgs.Name`: `required:""`;
  `levelArgs.Words`: `optional:""`), which `command/cobracmd`'s tests
  need in workspace mode against this root. Of the census's 14 fields,
  these are 8 and the two in `workspace` 2; the other 4, in
  `command/schema_test.go`, were not refused: those structs go to
  `SchemaOf` only, which does not check.
* `command/args_test.go` (new): `TestNewAgreesWithKong` (seven structs
  refused, each error naming its fix; nine accepted, among them a
  positional with a default, a pointer flag, an embedded struct, a
  nested object and a type with its own schema; `SchemaOf` of refused
  structs still succeeds) and `TestKongName` (twelve names). Before the
  release, a scratch program had Kong v1.16.1 name fields with the same
  twelve Go names: every spelling matched. Step 11's test checks it
  again against Kong's model.
* `docs/guides/commands.md`: the rule, beside the tags.

#### Checks (Rule 3)

The parked `command/kongcmd/` (Step 11's, untracked and outside
`go.work`) makes `internal/conformance` fail in workspace mode in the
tree, which finds the directory and cannot type-check it there. Every
check below therefore ran in the tree with `GOWORK=off`, or in a scratch
clone of HEAD with this step's changes staged and without
`command/kongcmd/`, which is what CI sees.

1. `gofmt -l command workspace`: silent.
2. `make pre-add-check FILES="<the 8 Go files>"` in the clone: exit 0,
   "go-precheck: 8 file(s) clean in 2 module(s)".
3. `make lint`: exit 0, "0 issues." for the three targets in both
   modules, after `make modernize`. (First run: staticcheck's De Morgan
   suggestions, twice, and `errcheck` on a discarded error of
   `ownSchema`, now returned.)
4. `GOWORK=off go test -race -count=1 ./...`: 16 packages `ok`;
   `command/cobracmd` 2.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: the same.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: the same.
7. `GOWORK=off go mod tidy -diff`: silent in both modules; `go.mod` and
   `go.sum` unchanged. In the clone: `scripts/go-modules.sh --check` and
   `make release-check` exit 0 ("go-precheck: 139 file(s) clean in 2
   module(s)"), and the root's tests in workspace mode 16 `ok`.
   `command/cobracmd`'s tests pass in workspace mode, against this root,
   and with `GOWORK=off`, against `v0.4.0`.
8. `make fuzz FUZZTIME=10s` in the clone: exit 0.
9. Windows test host, on a copy of the tree without `command/kongcmd/`:
   go1.27.1 windows/amd64. `make pre-add-check`: exit 0, "go-precheck:
   139 file(s) clean in 2 module(s)"; `make lint`: exit 0, "0 issues."
   for each module and target; `make vuln`: exit 0, "No vulnerabilities
   found." in both modules; `GOWORK=off go test -count=3 -shuffle=on` of
   `command`, `workspace` and `command/cli`, then `command/cobracmd`'s
   tests in workspace mode: exit 0.
10. `markdownlint-cli2` on `docs/guides/commands.md`: 0 issues; the
    relative-link check: 0 broken.
11. The identifier scan of the diff and the new files: no match.

#### Mutations (Rule 4)

| Mutation | Test | Failing line |
| :--- | :--- | :--- |
| S11a-1: the required check skipped | `TestNewAgreesWithKong` | `a required flag without required: <nil>, want an error naming add required:""` |
| S11a-2: a positional's default not counted | `TestNewAgreesWithKong` | `a positional with a default: … Kong requires it, and the schema does not: add optional:""` |
| S11a-3: the name check skipped | `TestNewAgreesWithKong` | `a json name Kong spells differently: <nil>, want an error naming add name:"max_items"` |
| S11a-4: the spelling not lower-cased | `TestKongName` | `kongName("MaxItems") = "Max-Items", want "max-items"` |
| S11a-5: an acronym not split before a word | `TestKongName` | `kongName("HTTPServer") = "https-erver", want "http-server"` |
| S11a-6: `New` does not call the check | `TestNewAgreesWithKong` | `an optional positional: <nil>, want an error naming add optional:""` |

`command/args.go` and `workspace/commands.go` hold anchors of Steps 4
and 7, whose sets ran again: Step 4's 13 and Step 7's 11, all killed.

#### Release notes for `v0.5.0`

`v0.5.0` makes an argument struct read the same in Kong as in the
registry (0006-MADR A12).

* **`command`:** `New[A]` refuses an argument struct whose top-level
  fields Kong would read differently: a field the `json` tag makes
  required that Kong does not (a flag without `required:""`), or the
  reverse (a positional with `omitzero` and without `optional:""`), and a
  `json` name that is not Kong's name for the field (`name:""`, or the Go
  name with dashes: `MaxItems` is `max-items`). The error names the tag
  to add or remove. `SchemaOf` is unchanged, so output schemas need none
  of these tags.
* **`workspace`:** two argument structs gain the tags; nothing a caller
  sees changes.
* **Upgrading:** a struct that `v0.4.0` accepted may need `required:""`,
  `optional:""` or `name:""`; `New`'s error says which. No module is
  added, and nothing exported changes.

#### The release

Left to the owner: commit this step without `command/kongcmd/`, push,
and with CI green tag `v0.5.0`. Then the agent runs the consumer smoke
test against `v0.5.0`, and records it here; the step is done when the
tag exists and the smoke test builds. Step 11 then moves
`command/kongcmd`'s requirement to `v0.5.0` and resumes.

**After the release (2026-10-06).** The owner committed this step
(`d8b7ff7`), ran the disclosure guard over the outgoing commits through
the agent (exit 0, no finding), pushed, and with CI green tagged `v0.5.0`
(annotated) on `d8b7ff7` and pushed the tag. The agent ran the releasing
guide's consumer smoke test against it, in a scratch module outside the
repository, with no `go.work` in effect and `GOPROXY` at its default:

* The program imports `command`, `command/cli`, `workspace`, `layout`,
  `theme` and `glyph`. It registers `workspace.Commands` of a workspace,
  whose two argument structs gained tags in this step, beside a command
  of its own with `required:""`, and calls `command.New` once with a
  struct the check must refuse.
* `go mod init`, `go get github.com/maccavelli/go-tui-lib@v0.5.0` ("go:
  downloading github.com/maccavelli/go-tui-lib v0.5.0"), `go mod tidy`,
  `go build ./...` and `go vet ./...`: each exit 0; `go.mod` requires
  `github.com/maccavelli/go-tui-lib v0.5.0`.
* Run: the refused struct gave `command: t.refused: arguments:
  main.refused.Name: the schema requires it, and Kong does not: add
  required:"", or make it omitzero`; the workspace's thirteen commands
  and the program's registered, and `greet --name world` printed `hello
  world` and exited 0.

The tag exists and the smoke test builds, so Step 11a is done. Step 11
resumes: `command/kongcmd` requires `v0.5.0`.
