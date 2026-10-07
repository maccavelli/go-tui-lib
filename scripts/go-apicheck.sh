#!/usr/bin/env bash
# Fails on an incompatible change to a module's exported API since the
# module's previous release tag
# (docs/decisions/0014-MADR-native-integration-api.md W0.2;
# docs/decisions/0014-PLAN-api-policy-gates.md Step 4).
#
# For each module scripts/go-modules.sh lists, with GOWORK=off:
#   1. The base is the newest <prefix>vX.Y.Z tag merged into HEAD that does
#      not contain HEAD, so a run on a tagged commit compares with the tag
#      before it. The prefix is empty for the root and "<dir>/" for a nested
#      module. A module with no such tag has never been released, and is
#      skipped.
#   2. apidiff exports the base's API, from `git archive <tag>`, and the
#      tree's, each in module mode.
#   3. apidiff compares the two, printing incompatible changes only. It exits
#      0 even when it prints one, so its output is judged line by line
#      against scripts/apicheck.allow: a line the file does not list fails,
#      and so does a listed line apidiff no longer prints, which is stale.
#
# scripts/apicheck.allow holds "<module dir><TAB><apidiff line>" entries,
# and "#" comments. Each block names the PLAN that allows its changes; a
# release's close-out removes them.
#
# Usage: scripts/go-apicheck.sh
#
# Env:
#   APIDIFF_VERSION              the golang.org/x/exp version apidiff runs at
#                                (default: the Makefile's pin)
#   APIDIFF=<path>               an apidiff binary to use instead (tests)
#   APICHECK_ALLOW=<path>        the allow file (default scripts/apicheck.allow)
#   GO_PRECHECK_SKIP_APICHECK=1  skip (offline work)
#   GO                           the go command (default go)
#
# Exit codes: 0 clean or skipped · 1 an unlisted or stale line · 2 a tool or
# git failure.
set -uo pipefail

GO="${GO:-go}"
HERE="$(cd "$(dirname "$0")" && pwd)"

if [ "${GO_PRECHECK_SKIP_APICHECK:-0}" = "1" ]; then
  echo "apicheck: skipped (GO_PRECHECK_SKIP_APICHECK=1)" >&2
  exit 0
fi

ROOT="$(git rev-parse --show-toplevel)" || exit 2
cd "$ROOT" || exit 2

ALLOW="${APICHECK_ALLOW:-$ROOT/scripts/apicheck.allow}"
if [ ! -f "$ALLOW" ]; then
  echo "apicheck: no allow file at $ALLOW." >&2
  exit 2
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# apidiff, at the Makefile's pin unless a binary is given.
if [ -z "${APIDIFF:-}" ]; then
  if [ -z "${APIDIFF_VERSION:-}" ]; then
    APIDIFF_VERSION="$(sed -n 's/^APIDIFF_VERSION[[:space:]]*?=[[:space:]]*//p' "$HERE/../Makefile" | head -1)"
  fi
  if [ -z "${APIDIFF_VERSION:-}" ]; then
    echo "apicheck: APIDIFF_VERSION is not set, and the Makefile names none." >&2
    exit 2
  fi
  if ! out="$(GOBIN="$WORK/bin" GOWORK=off "$GO" install "golang.org/x/exp/cmd/apidiff@$APIDIFF_VERSION" 2>&1)"; then
    echo "apicheck: cannot install apidiff@$APIDIFF_VERSION:" >&2
    printf '%s\n' "$out" | sed 's/^/  /' >&2
    exit 2
  fi
  APIDIFF="$WORK/bin/apidiff"
  [ -x "$APIDIFF.exe" ] && APIDIFF="$APIDIFF.exe"
fi

# The allow file's entries, without comments or blank lines, sorted.
grep -v -e '^#' -e '^[[:space:]]*$' "$ALLOW" | tr -d '\r' | LC_ALL=C sort -u >"$WORK/allowed"

if ! modules="$("$HERE/go-modules.sh")"; then
  echo "apicheck: scripts/go-modules.sh failed." >&2
  exit 2
fi

status=0
i=0
: >"$WORK/printed"
while IFS= read -r m; do
  [ -z "$m" ] && continue
  prefix=""
  [ "$m" != "." ] && prefix="$m/"
  tag="$(git tag --merged HEAD --no-contains HEAD -l "${prefix}v[0-9]*.[0-9]*.[0-9]*" --sort=-v:refname | head -1)"
  if [ -z "$tag" ]; then
    echo "apicheck: $m: no previous tag, skipped"
    continue
  fi
  if ! mod="$(cd "$m" && GOWORK=off "$GO" list -m 2>&1)"; then
    echo "apicheck: $m: go list -m failed: $mod" >&2
    exit 2
  fi

  # Numbered, not named after the module: "base-." would end in a dot, which
  # Windows strips from a path, so a native program could not enter it.
  i=$((i + 1))
  base="$WORK/base$i"
  mkdir -p "$base"
  if [ "$m" = "." ]; then
    git archive "$tag" | tar -x -C "$base" || exit 2
  else
    git archive "$tag" -- "$m" | tar -x -C "$base" || exit 2
  fi
  if ! out="$(cd "$base/$m" && GOWORK=off "$APIDIFF" -m -w "$WORK/old.api" "$mod" 2>&1)"; then
    echo "apicheck: $m: exporting $tag's API failed:" >&2
    printf '%s\n' "$out" | sed 's/^/  /' >&2
    exit 2
  fi
  if ! out="$(cd "$m" && GOWORK=off "$APIDIFF" -m -w "$WORK/new.api" "$mod" 2>&1)"; then
    echo "apicheck: $m: exporting the tree's API failed:" >&2
    printf '%s\n' "$out" | sed 's/^/  /' >&2
    exit 2
  fi
  if ! "$APIDIFF" -m -incompatible "$WORK/old.api" "$WORK/new.api" >"$WORK/diff" 2>"$WORK/diff.err"; then
    echo "apicheck: $m: apidiff failed:" >&2
    sed 's/^/  /' "$WORK/diff.err" >&2
    exit 2
  fi
  n=0
  while IFS= read -r line; do
    line="${line%$'\r'}"
    [ -z "$line" ] && continue
    printf '%s\t%s\n' "$m" "$line" >>"$WORK/printed"
    n=$((n + 1))
  done <"$WORK/diff"
  echo "apicheck: $m: against $tag, $n incompatible change(s)"
done < <(printf '%s\n' "$modules")

LC_ALL=C sort -u "$WORK/printed" -o "$WORK/printed"
unlisted="$(LC_ALL=C comm -23 "$WORK/printed" "$WORK/allowed")"
stale="$(LC_ALL=C comm -13 "$WORK/printed" "$WORK/allowed")"
if [ -n "$unlisted" ]; then
  echo "apicheck: incompatible changes that scripts/apicheck.allow does not list:" >&2
  printf '%s\n' "$unlisted" | sed 's/^/  /' >&2
  status=1
fi
if [ -n "$stale" ]; then
  echo "apicheck: scripts/apicheck.allow lists changes apidiff no longer reports (stale):" >&2
  printf '%s\n' "$stale" | sed 's/^/  /' >&2
  status=1
fi
[ "$status" -eq 0 ] && echo "apicheck: clean"
exit "$status"
