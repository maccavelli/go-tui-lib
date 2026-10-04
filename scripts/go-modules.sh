#!/usr/bin/env bash
# Lists this repository's Go modules, and checks go.work against the tree
# (docs/decisions/0010-MADR-nested-adapter-modules.md §2 and §4;
# docs/decisions/0010-PLAN-nested-adapter-modules.md Phase 3).
#
# Usage:
#   scripts/go-modules.sh           print each module directory, relative to
#                                   the root ("." for the root), one per line
#   scripts/go-modules.sh --check   compare the modules go.work lists with the
#                                   directories of the tracked go.mod files
#
# The list comes from `go list -m` in workspace mode, with GOWORK pointing at
# the root's go.work whatever the caller's environment says, so a caller that
# runs its gates with GOWORK=off still gets every module.
#
# --check fails, naming each difference, when go.work lists a module that has
# no tracked go.mod, or a tracked go.mod is missing from go.work. A go.mod the
# go command ignores is skipped: one under testdata, or under a directory
# whose name starts with "." or "_".
#
# GO names the go command (default go); the tests inject one through it, as
# scripts/go-fuzz_test.sh does for go-fuzz.sh.
#
# Exit codes: 0 ok · 1 check failed · 2 usage or tool error.
set -uo pipefail

GO="${GO:-go}"

ROOT="$(git rev-parse --show-toplevel)" || exit 2
cd "$ROOT" || exit 2

mode=list
case "${1:-}" in
"") ;;
--check) mode=check ;;
*)
  echo "usage: $0 [--check]" >&2
  exit 2
  ;;
esac

if [ ! -f go.work ]; then
  echo "go-modules: no go.work at the repository root." >&2
  [ "$mode" = check ] && exit 1
  exit 2
fi

# The modules go.work lists, relative to the root. git turns each
# directory into a path relative to the repository, whatever form the go
# command gives it: on Windows `go list` prints C:\Users\..., where git's
# own top level is C:/Users/... (0010-PLAN deviation D1).
if ! dirs="$(GOWORK="$ROOT/go.work" "$GO" list -m -f '{{.Dir}}' 2>&1)"; then
  echo "go-modules: go list -m failed in workspace mode:" >&2
  printf '%s\n' "$dirs" | sed 's/^/  /' >&2
  [ "$mode" = check ] && exit 1
  exit 2
fi
listed="$(
  while IFS= read -r d; do
    [ -z "$d" ] && continue
    if [ "$(git -C "$d" rev-parse --show-toplevel 2>/dev/null)" = "$ROOT" ]; then
      prefix="$(git -C "$d" rev-parse --show-prefix)"
      prefix="${prefix%/}"
      echo "${prefix:-.}"
    else
      echo "$d" # outside the repository; --check reports it
    fi
  done <<<"$dirs"
)"

if [ "$mode" = list ]; then
  printf '%s\n' "$listed"
  exit 0
fi

# The directories of the tracked go.mod files the go command would read.
tracked="$(
  git ls-files -- 'go.mod' ':(glob)**/go.mod' |
    grep -Ev '(^|/)(testdata|[._][^/]*)/' |
    while IFS= read -r f; do dirname "$f"; done
)"

status=0
while IFS= read -r d; do
  [ -z "$d" ] && continue
  echo "go-modules: go.work lists $d, which has no tracked go.mod." >&2
  status=1
done < <(comm -23 <(printf '%s\n' "$listed" | sort -u) <(printf '%s\n' "$tracked" | sort -u))
while IFS= read -r d; do
  [ -z "$d" ] && continue
  echo "go-modules: $d/go.mod is tracked, but go.work does not list $d (run 'go work use $d')." >&2
  status=1
done < <(comm -13 <(printf '%s\n' "$listed" | sort -u) <(printf '%s\n' "$tracked" | sort -u))
exit "$status"
