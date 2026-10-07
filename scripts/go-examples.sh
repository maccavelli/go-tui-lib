#!/usr/bin/env bash
# Builds and runs the framework examples, and checks the guides' excerpts of
# them (docs/decisions/0014-MADR-native-integration-api.md W0.4;
# docs/decisions/0014-PLAN-api-policy-gates.md Step 5).
#
# The examples live under testdata/frameworks, which the go command,
# scripts/go-modules.sh and golangci-lint all skip. They are programs built on
# the standard flag package, Cobra, Kong and urfave/cli, and they build in a
# temporary module outside the repository's modules:
#   1. testdata/frameworks is copied into a temporary directory, with go.mod
#      written from go.mod.tmpl, @ROOT@ set to the repository root, and the
#      committed go.sum beside it.
#   2. With GOWORK=off: go build ./... and go vet ./....
#   3. Each line of cases.txt runs one program, as
#        <framework> <args…> => <exit> stdout|stderr <substring>
#      and fails unless the exit status matches and the stream named holds
#      the substring.
#   4. The guide check: every fenced Go block in docs/guides/*.md whose
#      nearest non-blank line above is
#        <!-- from: testdata/frameworks/<file>#<name> -->
#      must equal the lines between "// guide:<name>" and "// guide:end" in
#      that file. The region's blank lines at either end are dropped (gofmt
#      puts one before a top-level marker), its common leading tabs are
#      removed, and each remaining leading tab is written as four spaces,
#      since the guides indent with spaces (deviation D4); then the two are
#      compared byte for byte.
#
# Usage:
#   scripts/go-examples.sh                 steps 1 to 4
#   scripts/go-examples.sh --check-guide   step 4 only: no build, no network
#   scripts/go-examples.sh --update        tidy the temporary module, and
#                                          write go.mod.tmpl and go.sum back
#                                          (deviation D7); read the diff
#
# Env:
#   EXAMPLES_ROOT=<dir>          the tree to read (default: the repository);
#                                the tests point it at a throwaway one
#   GO_PRECHECK_SKIP_EXAMPLES=1  skip (offline work)
#   GO                           the go command (default go)
#
# Exit codes: 0 clean or skipped · 1 a failed case or excerpt · 2 a build,
# usage or tool failure.
set -uo pipefail

GO="${GO:-go}"

mode=all
case "${1:-}" in
"") ;;
--check-guide) mode=guide ;;
--update) mode=update ;;
*)
  echo "usage: scripts/go-examples.sh [--check-guide | --update]" >&2
  exit 2
  ;;
esac

if [ "${GO_PRECHECK_SKIP_EXAMPLES:-0}" = "1" ]; then
  echo "examples: skipped (GO_PRECHECK_SKIP_EXAMPLES=1)" >&2
  exit 0
fi

if [ -n "${EXAMPLES_ROOT:-}" ]; then
  ROOT="$(cd "$EXAMPLES_ROOT" && pwd)" || exit 2
else
  ROOT="$(git rev-parse --show-toplevel)" || exit 2
fi
SRC="$ROOT/testdata/frameworks"
GUIDES="$ROOT/docs/guides"
if [ ! -f "$SRC/go.mod.tmpl" ]; then
  echo "examples: no $SRC/go.mod.tmpl." >&2
  exit 2
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
status=0

# module: the temporary module, at $WORK/m.
module() {
  local root="$ROOT" tmpl
  # The go command on Windows needs a Windows path in a replace directive.
  command -v cygpath >/dev/null 2>&1 && root="$(cygpath -m "$ROOT")"
  cp -R "$SRC" "$WORK/m" || exit 2
  tmpl="$(cat "$SRC/go.mod.tmpl")" || exit 2
  printf '%s\n' "${tmpl//@ROOT@/$root}" >"$WORK/m/go.mod"
  rm -f "$WORK/m/go.mod.tmpl"
}

if [ "$mode" = update ]; then
  module
  if ! out="$(cd "$WORK/m" && GOWORK=off "$GO" mod tidy 2>&1)"; then
    echo "examples: go mod tidy failed:" >&2
    printf '%s\n' "$out" | sed 's/^/  /' >&2
    exit 2
  fi
  # go.mod goes back as the template: the replace line names @ROOT@ again.
  awk '/^replace github.com\/maccavelli\/go-tui-lib => / { print "replace github.com/maccavelli/go-tui-lib => @ROOT@"; next } { print }' \
    "$WORK/m/go.mod" >"$SRC/go.mod.tmpl" || exit 2
  cp "$WORK/m/go.sum" "$SRC/go.sum" || exit 2
  echo "examples: wrote testdata/frameworks/go.mod.tmpl and go.sum; read the diff before committing."
  exit 0
fi

if [ "$mode" = all ]; then
  module
  hint="  If go.mod.tmpl or go.sum is out of date: scripts/go-examples.sh --update"
  # The build writes each program into $WORK/bin: a plain `go build ./...`
  # of one main package would write it into the module.
  mkdir -p "$WORK/bin"
  for step in build vet; do
    args=(./...)
    [ "$step" = build ] && args=(-o "$WORK/bin/" ./...)
    if ! out="$(cd "$WORK/m" && GOWORK=off "$GO" "$step" "${args[@]}" 2>&1)"; then
      echo "examples: go $step failed:" >&2
      printf '%s\n' "$out" | sed 's/^/  /' >&2
      echo "$hint" >&2
      exit 2
    fi
  done
  exe="$(GOWORK=off "$GO" env GOEXE)"

  # 3. The cases.
  n=0
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    case "$line" in "" | "#"*) continue ;; esac
    if [ "${line#* => }" = "$line" ]; then
      echo "examples: cases.txt: no \" => \" in: $line" >&2
      exit 2
    fi
    read -r -a words <<<"${line%% => *}"
    read -r want stream substr <<<"${line#* => }"
    fw="${words[0]}"
    if [ ! -x "$WORK/bin/$fw$exe" ]; then
      echo "examples: cases.txt names $fw, which is not a program under testdata/frameworks." >&2
      exit 2
    fi
    case "$stream" in stdout | stderr) ;; *)
      echo "examples: cases.txt: the stream is stdout or stderr, not \"$stream\", in: $line" >&2
      exit 2
      ;;
    esac
    (cd "$WORK" && "$WORK/bin/$fw$exe" "${words[@]:1}" >"$WORK/stdout" 2>"$WORK/stderr" </dev/null)
    got=$?
    n=$((n + 1))
    if [ "$got" != "$want" ]; then
      echo "examples: $line: exit $got, want $want" >&2
      sed 's/^/  stdout: /' "$WORK/stdout" >&2
      sed 's/^/  stderr: /' "$WORK/stderr" >&2
      status=1
    elif ! grep -q -F -- "$substr" "$WORK/$stream"; then
      echo "examples: $line: $stream does not hold \"$substr\"" >&2
      sed "s/^/  $stream: /" "$WORK/$stream" >&2
      status=1
    fi
  done <"$SRC/cases.txt"
  if [ "$n" -eq 0 ]; then
    echo "examples: cases.txt holds no case." >&2
    exit 2
  fi
  echo "examples: $n case(s) run"
fi

# 4. The guide check. awk writes each referenced block to $WORK/g/<n>.block
# and an index line "<guide> <line> <ref> <n>"; a from line that no Go block
# follows is indexed with "orphan".
mkdir -p "$WORK/g"
: >"$WORK/g/index"
shopt -s nullglob
guides=("$GUIDES"/*.md)
shopt -u nullglob
if [ "${#guides[@]}" -gt 0 ]; then
  awk -v dir="$WORK/g" '
    function orphan() {
      if (pending != "") printf "%s\t%d\t%s\torphan\n", pfile, pline, pending > (dir "/index")
      pending = ""
    }
    { sub(/\r$/, "") }
    FNR == 1 { orphan(); inblock = 0 }
    inblock && /^```[[:space:]]*$/ { inblock = 0; close(body); next }
    inblock { print > body; next }
    /^<!-- from: [^ ]+ -->[[:space:]]*$/ { orphan(); pending = $3; pfile = FILENAME; pline = FNR; next }
    pending != "" && /^[[:space:]]*$/ { next }
    pending != "" && /^```go[[:space:]]*$/ {
      n++
      body = dir "/" n ".block"
      printf "" > body
      printf "%s\t%d\t%s\t%d\n", pfile, pline, pending, n > (dir "/index")
      pending = ""
      inblock = 1
      next
    }
    pending != "" { orphan() }
    END { orphan() }
  ' "${guides[@]}" || exit 2
fi

excerpts=0
while IFS=$'\t' read -r guide gline ref n; do
  where="${guide#"$ROOT"/}:$gline"
  if [ "$n" = orphan ]; then
    echo "examples: $where: <!-- from: $ref --> is not followed by a Go block." >&2
    status=1
    continue
  fi
  file="${ref%%#*}"
  name="${ref#*#}"
  if [ "$file" = "$ref" ] || [ -z "$name" ]; then
    echo "examples: $where: $ref names no region (<file>#<name>)." >&2
    status=1
    continue
  fi
  case "$file" in testdata/frameworks/*) ;; *)
    echo "examples: $where: $ref is not under testdata/frameworks." >&2
    status=1
    continue
    ;;
  esac
  if [ ! -f "$ROOT/$file" ]; then
    echo "examples: $where: $file does not exist." >&2
    status=1
    continue
  fi
  # The region, trimmed and dedented, with each leading tab as four spaces.
  # Exit 3: no start marker; 4: no end marker.
  awk -v name="$name" '
    { sub(/\r$/, "") }
    !on && $0 ~ ("^[ \t]*// guide:" name "$") { on = 1; found = 1; next }
    on && /^[ \t]*\/\/ guide:end$/ { ended = 1; exit }
    on { lines[++k] = $0 }
    END {
      if (!found) exit 3
      if (!ended) exit 4
      min = -1
      for (i = 1; i <= k; i++) {
        if (lines[i] ~ /^[ \t]*$/) continue
        match(lines[i], /^\t*/)
        if (min < 0 || RLENGTH < min) min = RLENGTH
      }
      if (min < 0) min = 0
      first = 1
      while (first <= k && lines[first] ~ /^[ \t]*$/) first++
      last = k
      while (last >= first && lines[last] ~ /^[ \t]*$/) last--
      for (i = first; i <= last; i++) {
        l = lines[i]
        if (l ~ /^[ \t]*$/) { print ""; continue }
        l = substr(l, min + 1)
        match(l, /^\t*/)
        pad = ""
        for (j = 0; j < RLENGTH; j++) pad = pad "    "
        print pad substr(l, RLENGTH + 1)
      }
    }
  ' "$ROOT/$file" >"$WORK/g/$n.region"
  rc=$?
  if [ "$rc" -eq 3 ]; then
    echo "examples: $where: $file has no region \"// guide:$name\"." >&2
    status=1
    continue
  elif [ "$rc" -eq 4 ]; then
    echo "examples: $where: $file's region $name has no \"// guide:end\"." >&2
    status=1
    continue
  elif [ "$rc" -ne 0 ]; then
    exit 2
  fi
  excerpts=$((excerpts + 1))
  if ! cmp -s "$WORK/g/$n.region" "$WORK/g/$n.block"; then
    echo "examples: $where: the excerpt differs from $ref (- the program, + the guide):" >&2
    diff -u "$WORK/g/$n.region" "$WORK/g/$n.block" | tail -n +3 | sed 's/^/  /' >&2
    status=1
  fi
done <"$WORK/g/index"
echo "examples: $excerpts excerpt(s) checked"

[ "$status" -eq 0 ] && echo "examples: clean"
exit "$status"
