# Architecture

`go-tui-lib` is the Go module `github.com/maccavelli/go-tui-lib`: a library
of terminal-UI packages on the Charm v2 stack. It has no binary.

## What it is

- **A library only.** Each capability is a top-level directory, with the
  package named after it. There is no root package. Helpers shared between
  packages go under `internal/`.
- **No package yet.** The module has no Go file, no requirement and no
  `go.sum`. `make test`, `make vet`, `make lint` and `make vuln` fail with
  "no packages" until the first package lands. `make pre-add-check` reports
  `no Go files to check.` and exits 0.
- **Go 1.27.1**, with no `toolchain` line.

## Tree

```text
README.md                   repository entry; links here
LICENSE                     Apache License 2.0
AGENTS.md                   rules for agents: dependencies, TUI conventions,
                            records, checks, identifiers, commits
go.mod                      the module, with no requirements
Makefile                    development targets (below)
.golangci.yml               golangci-lint configuration, with depguard
.markdownlint-cli2.jsonc    Markdown lint configuration
.gitattributes              LF line endings everywhere
.gitignore
.github/workflows/
  ci.yml                    CI
scripts/
  go-precheck.sh            the pre-add check
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  decisions/                MADR and PLAN records
```

## Dependencies

- **Named stack:** `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`,
  `charm.land/bubbles/v2`, `github.com/charmbracelet/colorprofile`,
  `github.com/charmbracelet/x/ansi` and `github.com/maccavelli/go-core-lib`.
  None is required yet. Each is added in the commit with its first import.
- **Refused by `depguard`,** in source and tests:
  - the Charm v1 paths `github.com/charmbracelet/bubbletea`, `…/lipgloss`
    and `…/bubbles`;
  - `github.com/maccavelli/mcplib`;
  - `github.com/modelcontextprotocol/go-sdk`;
  - `github.com/maccavelli/go-llmprovider-sdk`.

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`, `tidy`,
  `vuln`, `pre-add-check`, `help`.
- **`make lint`** runs `golangci-lint run -c .golangci.yml ./...` three
  times: `GOOS=linux`, `darwin` and `windows`, each with `CGO_ENABLED=0`.
- **`scripts/go-precheck.sh`** runs `gofmt` on the given Go files, the same
  three golangci-lint runs, `go vet` and `go test` on their packages, and
  `govulncheck ./...`. `make pre-add-check` runs it, and so does the
  machine-wide agent gate before an agent `git commit` that stages Go files.
- **`.golangci.yml`:**
  - 21 linters, including `depguard`;
  - `revive`'s `exported`, `package-comments` and `var-naming` rules in
    place of `golint`;
  - errcheck with `check-blank` and `check-type-assertions`;
  - gofmt and goimports as formatters;
  - test files exempt from `errcheck`, `gosec`, `unparam`, `revive`,
    `gocritic` and `goconst`, but not from `depguard`.
- **CI** (`.github/workflows/ci.yml`) runs on `ubuntu-24.04`, `macos-15` and
  `windows-2025`, with the Go version read from `go.mod`.
  - **Every OS:** `go test`.
  - **Linux and macOS:** `go test -race`.
  - **Linux also:**
    - `go test -shuffle=on -count=2` and `LC_ALL=C go test`;
    - `go vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`;
    - `go vet`, `gofmt`, `go mod tidy -diff`, and `make lint` with
      golangci-lint v2.14.0;
    - `govulncheck` v1.8.0;
    - `shellcheck` v0.11.0 (pinned by SHA-256 and first on `PATH`),
      `markdownlint-cli2` 0.23.2 and `actionlint` v1.7.12.
  - One run per ref (`concurrency`, cancel in progress). Actions are pinned
    to commit SHAs.

## What is not here

- **Any package.** The first, `updatetea` or an extraction from ocp-login,
  is its own record.
- **`make apicheck`.** It needs a release tag to compare against. It comes
  with the first release.
- **`docs/reports/` and `docs/guides/`.** Each is created by its first
  document.
- **A release workflow, Dependabot, and any tag.**
