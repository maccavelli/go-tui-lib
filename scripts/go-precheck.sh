#!/usr/bin/env bash
# Go pre-add checks: format, lint, vet, tests, tidiness, vulnerabilities.
#
# The single implementation of the pre-add rule in AGENTS.md, called from two
# places so they cannot drift: `make pre-add-check`, and the machine-wide agent
# gate that runs before every agent `git commit`
# (~/.agents/hooks/lib/precommit-checks.sh), which prefers this script whenever
# the repository ships one and Go files are staged.
#
# Taken from go-selfupdate-lib's (then go-core-lib's) scripts/go-precheck.sh
# under docs/decisions/0001-MADR-scaffold-charm-tui-library.md §4, unchanged
# apart from this comment and one citation that now names go-selfupdate-lib.
# go-selfupdate-lib took it from go-llmprovider-sdk, which adapted
# magic-cli-remote's.
#
# golangci-lint runs with this repository's .golangci.yml instead of golint.
# golint is archived, and CI already runs golangci-lint; a gate weaker than CI
# is not a gate. golint's own checks live on as revive's exported,
# package-comments and var-naming rules in .golangci.yml.
#
# Every module is checked on its own, in its own directory
# (docs/decisions/0010-MADR-nested-adapter-modules.md §4;
# docs/decisions/0010-PLAN-nested-adapter-modules.md Phase 3). The modules are
# the ones go.work lists (scripts/go-modules.sh). For each module:
#   1. gofmt on its files;
#   2. with GOWORK=off, so a module builds from its own go.mod and published
#      versions only: golangci-lint for linux, darwin and windows; go vet;
#      go test; go mod tidy -diff; govulncheck;
#   3. go test once more in workspace mode, against the tree's other modules;
#   4. for a module other than the root: no replace directive, and a
#      requirement of the root, if any, on a release version (vX.Y.Z).
# Then, once: scripts/go-modules.sh --check, which fails when go.work and the
# tracked go.mod files disagree; and, when no file list is given (the
# `make release-check` path), scripts/go-apicheck.sh, which fails on an
# incompatible API change since each module's previous tag that
# scripts/apicheck.allow does not list
# (docs/decisions/0014-PLAN-api-policy-gates.md Step 4). A file list skips
# it: the API belongs to the whole module, not to the files. And
# scripts/go-examples.sh, which builds and runs the framework examples under
# testdata/frameworks and checks the guides' excerpts of them, on the
# release-check path and whenever a file list names a file under
# testdata/frameworks (Step 5, deviation D5).
#
# A Go file under a testdata directory is gofmt'd, but its directory is not
# given to go vet or go test: the go command ignores testdata, and the
# framework examples there import modules no module of this repository
# requires.
#
# Usage:
#   scripts/go-precheck.sh [file.go ...]
#
# With no arguments it checks every tracked Go file, in every module; with
# arguments, only those, in the modules that own them (non-Go arguments are
# ignored, so callers can pass a whole changed-file list). The lint, tidy and
# vulnerability steps are module-scoped either way: golangci-lint analyses
# packages, not files, so narrowing it to a file list would report different
# findings than `make lint` and the two would drift.
#
# Exit codes: 0 all clear · 1 a check failed · 2 a required tool is missing.
#
# Env:
#   GOLANGCI_LINT=<path>      golangci-lint binary (default: $(go env GOPATH)/bin)
#   GO_PRECHECK_SKIP_VULN=1   skip govulncheck (offline work)
#   GO_PRECHECK_SKIP_APICHECK=1  skip the API diff gate (offline work)
#   GO_PRECHECK_SKIP_EXAMPLES=1  skip the framework examples (offline work)
set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT" || exit 1

# Whether the framework examples run: on a whole-tree run, or when a file
# under testdata/frameworks is given.
examples_run=0
[ "$#" -eq 0 ] && examples_run=1
for f in "$@"; do
  case "${f#./}" in testdata/frameworks/*) examples_run=1 ;; esac
done

# Collect the Go files to check.
files=()
if [ "$#" -gt 0 ]; then
  for f in "$@"; do
    case "$f" in
    *.go) [ -f "$f" ] && files+=("$f") ;;
    esac
  done
else
  while IFS= read -r f; do
    [ -n "$f" ] && files+=("$f")
  done < <(git ls-files '*.go')
fi

# Nothing to check is not an error: a docs-only change, or a tree with no Go
# code yet. Return before any tool runs — gofmt with no file reads stdin.
if [ "${#files[@]}" -eq 0 ]; then
  echo "go-precheck: no Go files to check."
  exit 0
fi

need() {
  command -v "$1" >/dev/null 2>&1 && return 0
  echo "go-precheck: $1 not found in PATH." >&2
  case "$1" in
  govulncheck) echo "  install: go install golang.org/x/vuln/cmd/govulncheck@v1.8.0" >&2 ;;
  esac
  return 1
}

failed=0
fail() {
  # 2 (tool missing) outranks 1 (check failed).
  [ "$failed" -lt "$1" ] && failed="$1"
}

# show TITLE OUTPUT [LINES]: report a failed step, indented.
show() {
  echo "$1:" >&2
  if [ -n "${3:-}" ]; then
    printf '%s\n' "$2" | tail -"$3" | sed 's/^/  /' >&2
  else
    printf '%s\n' "$2" | sed 's/^/  /' >&2
  fi
}

need gofmt || exit 2
need go || exit 2
GOLANGCI="${GOLANGCI_LINT:-$(go env GOPATH)/bin/golangci-lint}"
have_lint=1
if [ ! -x "$GOLANGCI" ]; then
  echo "go-precheck: golangci-lint not found at $GOLANGCI." >&2
  echo "  install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0" >&2
  fail 2
  have_lint=0
fi
have_vuln=1
if [ "${GO_PRECHECK_SKIP_VULN:-0}" = "1" ]; then
  echo "govulncheck: skipped (GO_PRECHECK_SKIP_VULN=1)" >&2
  have_vuln=0
elif ! need govulncheck; then
  fail 2
  have_vuln=0
fi

# The modules, longest directory first, so that a file belongs to the
# deepest module containing it.
if ! module_list="$("$REPO_ROOT/scripts/go-modules.sh")"; then
  echo "go-precheck: could not list the modules (scripts/go-modules.sh)." >&2
  exit 1
fi
modules=()
while IFS= read -r m; do
  [ -n "$m" ] && modules+=("$m")
done < <(printf '%s\n' "$module_list" | awk '{ print length($0) "\t" $0 }' | sort -rn | cut -f2-)
ROOT_PATH="$(GOWORK=off go list -m)"

# owner FILE: the module directory that owns FILE.
owner() {
  local m
  for m in "${modules[@]}"; do
    if [ "$m" = "." ]; then
      echo "."
      return
    fi
    case "$1" in
    "$m"/*)
      echo "$m"
      return
      ;;
    esac
  done
  echo "."
}

# govulncheck over a module. It reports *called* vulnerabilities, so it is a
# property of the whole build rather than of the edited files.
#
# Exit status 3 is govulncheck's "vulnerabilities found" and always fails. Only
# another non-zero status whose output looks like a network failure is treated
# as an unreachable database: matching those words on a status-3 run would let a
# finding whose trace names a `proxy` package or a `Timeout` function through.
vuln() {
  local out rc
  out="$(GOWORK=off govulncheck ./... 2>&1)"
  rc=$?
  if [ "$rc" -eq 3 ]; then
    show "govulncheck: vulnerabilities found" "$out" 30
    fail 1
  elif [ "$rc" -ne 0 ]; then
    if printf '%s' "$out" | grep -qiE 'no such host|connection refused|timeout|dial tcp|proxy'; then
      echo "govulncheck: could not reach the vulnerability database; skipped." >&2
    else
      show "govulncheck: failed (exit $rc)" "$out" 30
      fail 1
    fi
  fi
}

# requirements: a module other than the root has no replace directive, and
# requires the root, if at all, at a release version.
requirements() {
  local m="$1" printed ver
  if ! printed="$(GOWORK=off go mod edit -print 2>&1)"; then
    show "go mod edit ($m)" "$printed"
    fail 1
    return
  fi
  if printf '%s\n' "$printed" | grep -qE '^replace[[:space:]]'; then
    show "go.mod ($m): a replace directive (a module builds from published versions only)" \
      "$(printf '%s\n' "$printed" | awk '/^replace \(/ {r = 1} r || /^replace / {print} r && /^\)/ {r = 0}')"
    fail 1
  fi
  ver="$(printf '%s\n' "$printed" | awk -v p="$ROOT_PATH" '
    /^require \(/ { inreq = 1; next }
    inreq && /^\)/ { inreq = 0; next }
    inreq && $1 == p { print $2 }
    $1 == "require" && $2 == p { print $3 }')"
  if [ -n "$ver" ] && ! printf '%s\n' "$ver" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "go.mod ($m): requires $ROOT_PATH at $ver, which is not a release version (vX.Y.Z)." >&2
    fail 1
  fi
}

checked_files=0
checked_modules=0
# The root, then the other modules, in go.work's order.
while IFS= read -r m; do
  [ -z "$m" ] && continue
  mfiles=()
  for f in "${files[@]}"; do
    [ "$(owner "$f")" = "$m" ] && mfiles+=("$f")
  done
  [ "${#mfiles[@]}" -eq 0 ] && continue
  checked_files=$((checked_files + ${#mfiles[@]}))
  checked_modules=$((checked_modules + 1))
  echo "go-precheck: module $m (${#mfiles[@]} file(s))" >&2

  # 1. gofmt.
  unformatted="$(gofmt -l "${mfiles[@]}")"
  if [ -n "$unformatted" ]; then
    echo "gofmt: these files are not formatted (run 'gofmt -w <file>'):" >&2
    printf '%s\n' "$unformatted" | sed 's/^/  /' >&2
    fail 1
  fi

  # The packages the files belong to, relative to the module, or ./....
  if [ "$#" -gt 0 ]; then
    pkgs=()
    while IFS= read -r d; do
      [ -n "$d" ] && pkgs+=("./$d")
    done < <(for f in "${mfiles[@]}"; do
      case "/${f#./}" in */testdata/*) continue ;; esac
      d="$(dirname "$f")"
      if [ "$m" = "." ]; then echo "$d"; elif [ "$d" = "$m" ]; then echo "."; else echo "${d#"$m"/}"; fi
    done | sort -u)
  else
    pkgs=("./...")
  fi

  (
    cd "$m" || exit 1
    export GOWORK=off

    # 2. golangci-lint, with this repository's configuration: the same command
    # `make lint` runs, so a commit cannot pass a weaker check than CI applies.
    # It runs once per target the code builds for, with cgo off, because a
    # host-only run never sees a *_windows.go or *_unix.go file of another OS
    # (go-selfupdate-lib docs/decisions/0002-MADR-rehome-selfupdate-from-mcplib.md §5).
    if [ "$have_lint" = 1 ]; then
      for goos in linux darwin windows; do
        if ! lint_out="$(CGO_ENABLED=0 GOOS="$goos" "$GOLANGCI" run -c "$REPO_ROOT/.golangci.yml" ./... 2>&1)"; then
          show "golangci-lint ($m, GOOS=$goos)" "$lint_out" 40
          fail 1
        fi
      done
    fi

    # go vet and go test, over the packages the files belong to. AGENTS.md
    # requires both on the touched packages. Files under testdata alone
    # leave no package.
    if [ "${#pkgs[@]}" -gt 0 ]; then
      if ! vet_out="$(go vet "${pkgs[@]}" 2>&1)"; then
        show "go vet ($m)" "$vet_out"
        fail 1
      fi
      if ! test_out="$(go test "${pkgs[@]}" 2>&1)"; then
        show "go test ($m)" "$test_out" 40
        fail 1
      fi
    fi
    if ! tidy_out="$(go mod tidy -diff 2>&1)"; then
      show "go mod tidy -diff ($m)" "$tidy_out" 40
      fail 1
    fi
    [ "$have_vuln" = 1 ] && vuln

    # 3. go test in workspace mode, against the tree's other modules.
    if [ "${#pkgs[@]}" -gt 0 ] && ! ws_out="$(GOWORK="$REPO_ROOT/go.work" go test "${pkgs[@]}" 2>&1)"; then
      show "go test, workspace mode ($m)" "$ws_out" 40
      fail 1
    fi

    # 4. A nested module's requirements.
    [ "$m" != "." ] && requirements "$m"
    exit "$failed"
  )
  rc=$?
  [ "$rc" -gt 0 ] && fail "$rc"
done < <(printf '%s\n' "$module_list")

# Once: go.work lists exactly the tree's modules.
if ! check_out="$("$REPO_ROOT/scripts/go-modules.sh" --check 2>&1)"; then
  show "go.work" "$check_out"
  fail 1
fi

# Once, for the whole tree: no unlisted incompatible API change.
apicheck=""
if [ "$#" -eq 0 ]; then
  if [ "${GO_PRECHECK_SKIP_APICHECK:-0}" = "1" ]; then
    echo "apicheck: skipped (GO_PRECHECK_SKIP_APICHECK=1)" >&2
  else
    api_out="$("$REPO_ROOT/scripts/go-apicheck.sh" 2>&1)"
    api_rc=$?
    if [ "$api_rc" -ne 0 ]; then
      show "apicheck" "$api_out"
      fail "$api_rc"
    else
      apicheck=", apicheck"
    fi
  fi
fi

# Once: the framework examples build, run as their cases say, and match the
# guides' excerpts.
examples=""
if [ "$examples_run" = 1 ]; then
  if [ "${GO_PRECHECK_SKIP_EXAMPLES:-0}" = "1" ]; then
    echo "examples: skipped (GO_PRECHECK_SKIP_EXAMPLES=1)" >&2
  else
    ex_out="$("$REPO_ROOT/scripts/go-examples.sh" 2>&1)"
    ex_rc=$?
    if [ "$ex_rc" -ne 0 ]; then
      show "examples" "$ex_out"
      fail "$ex_rc"
    else
      examples=", examples"
    fi
  fi
fi

if [ "$failed" -eq 0 ]; then
  echo "go-precheck: $checked_files file(s) clean in $checked_modules module(s) (gofmt, golangci-lint, go vet, go test, go mod tidy, govulncheck$apicheck$examples)."
fi
exit "$failed"
