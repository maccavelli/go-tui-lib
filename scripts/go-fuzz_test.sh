#!/usr/bin/env bash
# Offline tests for go-fuzz.sh, on throwaway modules in a temporary directory
# (go-selfupdate-lib docs/decisions/0004-PLAN-h2-fuzzing-and-manifest-differential.md
# Step 4; here under docs/decisions/0002-PLAN-multi-pane-workspace-layouts.md
# Step 5).
#
# The failing-target case is the one that matters: a fuzz gate that reported
# success while a target failed would be no gate at all.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FUZZ="${FUZZ:-$ROOT/scripts/go-fuzz.sh}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

PASS=0
FAIL=0

check() { # name want got
	if [ "$2" = "$3" ]; then
		echo "  ok   $1"
		PASS=$((PASS + 1))
	else
		echo "  FAIL $1: want $2, got $3"
		FAIL=$((FAIL + 1))
	fi
}

# module DIR NAME BODY: a module whose package has one test file.
module() {
	mkdir -p "$1"
	printf 'module example.com/%s\n\ngo 1.27.1\n' "$2" >"$1/go.mod"
	printf 'package %s\n\nimport "testing"\n\n%s\n' "$2" "$3" >"$1/fz_test.go"
}

# run DIR ARGS...: the script's exit status, with output in $WORK/out.
run() {
	local dir="$1"
	shift
	set +e
	(cd "$dir" && "$FUZZ" "$@") >"$WORK/out" 2>&1
	local rc=$?
	set -e
	echo "$rc"
}

module "$WORK/clean" clean '
func FuzzA(f *testing.F) {
	f.Add([]byte("a"))
	f.Fuzz(func(t *testing.T, b []byte) { _ = len(b) })
}

func FuzzB(f *testing.F) {
	f.Add("b")
	f.Fuzz(func(t *testing.T, s string) { _ = len(s) })
}'

# 1. Two clean targets, two required: both are fuzzed, and it passes.
check "clean targets pass" 0 "$(run "$WORK/clean" -t 1s -m 2 ./)"
check "both targets were fuzzed" 2 "$(grep -c '^go-fuzz: Fuzz[AB] for 1s ' "$WORK/out" || true)"

# 1b. -z reaches go test as -fuzzminimizetime, through a recording GO.
REAL_GO="$(command -v go)"
mkdir -p "$WORK/bin"
printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$*" >>"%s"\nexec "%s" "$@"\n' "$WORK/go-args" "$REAL_GO" >"$WORK/bin/go"
chmod +x "$WORK/bin/go"
rc="$(GO="$WORK/bin/go" run "$WORK/clean" -t 1s -z 3s -m 2 ./)"
check "a recorded run passes" 0 "$rc"
check "-z reaches each fuzz run" 2 "$(grep -c -- '-fuzzminimizetime 3s' "$WORK/go-args" || true)"

# 2. Fewer targets than required: refused, naming the count.
check "too few targets is refused" 1 "$(run "$WORK/clean" -t 1s -m 3 ./)"
check "the refusal names the count" 1 "$(grep -c 'found 2 fuzz targets' "$WORK/out" || true)"

# 3. THE CASE THAT MATTERS. A target that fails on any input but its seed:
#    the fuzzer finds one at once, so the script must fail and Go must leave
#    the input behind.
module "$WORK/boom" boom '
func FuzzBoom(f *testing.F) {
	f.Add([]byte("a"))
	f.Fuzz(func(t *testing.T, b []byte) {
		if string(b) != "a" {
			t.Fatalf("boom on %q", b)
		}
	})
}'
rc="$(run "$WORK/boom" -t 5s -m 1 ./)"
if [ "$rc" -ne 0 ]; then rc=nonzero; fi
check "a failing target fails the run" nonzero "$rc"
saved=0
if [ -d "$WORK/boom/testdata/fuzz/FuzzBoom" ]; then
	saved="$(find "$WORK/boom/testdata/fuzz/FuzzBoom" -type f | wc -l | tr -d ' ')"
fi
check "the failing input is saved" 1 "$saved"

# 4. Usage errors.
check "no package is a usage error" 2 "$(run "$WORK/clean")"
check "a non-numeric minimum is a usage error" 2 "$(run "$WORK/clean" -m x ./)"
check "a zero minimum is a usage error" 2 "$(run "$WORK/clean" -m 0 ./)"
check "an unknown flag is a usage error" 2 "$(run "$WORK/clean" -z ./)"
check "-l without -a is a usage error" 2 "$(run "$WORK/clean" -l ./)"
check "-a with no pattern is a usage error" 2 "$(run "$WORK/clean" -a)"

# 5. Discovery (docs/decisions/0014-PLAN-hardening.md Step 9): a module
#    with a target in a package's own tests (a), one in an external test
#    package (d), tests without a target (b) and no tests at all (c).
mkdir -p "$WORK/disc/a" "$WORK/disc/b" "$WORK/disc/c" "$WORK/disc/d"
printf 'module example.com/disc\n\ngo 1.27.1\n' >"$WORK/disc/go.mod"
for p in a b c d; do printf 'package %s\n' "$p" >"$WORK/disc/$p/$p.go"; done
printf 'package a\n\nimport "testing"\n\nfunc FuzzA(f *testing.F) {\n\tf.Add(1)\n\tf.Fuzz(func(t *testing.T, n int) {})\n}\n' >"$WORK/disc/a/a_test.go"
printf 'package b\n\nimport "testing"\n\nfunc TestB(t *testing.T) {}\n' >"$WORK/disc/b/b_test.go"
printf 'package d_test\n\nimport "testing"\n\nfunc FuzzD(f *testing.F) {\n\tf.Add(1)\n\tf.Fuzz(func(t *testing.T, n int) {})\n}\n' >"$WORK/disc/d/d_test.go"
check "discovery lists the packages with targets" 0 "$(run "$WORK/disc" -a -l ./...)"
check "the list is a and d" "example.com/disc/a example.com/disc/d" "$(tr '\n' ' ' <"$WORK/out" | sed 's/ $//')"
check "discovery fuzzes them" 0 "$(run "$WORK/disc" -a -t 1s -m 1 ./...)"
check "each target is fuzzed once" 2 "$(grep -c '^go-fuzz: Fuzz[AD] for 1s ' "$WORK/out" || true)"
check "a package with no target is skipped" 0 "$(grep -c 'example.com/disc/[bc]' "$WORK/out" || true)"
check "the run names two packages" 1 "$(grep -c '^go-fuzz: 2 packages ran clean$' "$WORK/out" || true)"
check "no package with a target is refused" 1 "$(run "$WORK/disc" -a -l ./b/... ./c/...)"
rm -rf "$WORK/boom/testdata"
rc="$(run "$WORK/boom" -a -t 5s -m 1 ./...)"
if [ "$rc" -ne 0 ]; then rc=nonzero; fi
check "a failing target found by discovery fails the run" nonzero "$rc"
named="$(sed -n 's/.*Go wrote the failing input under \(.*\)$/\1/p' "$WORK/out" | tail -1)"
check "its failure names the directory the input is in" yes "$([ -n "$named" ] && [ -d "$named" ] && echo yes || echo no)"

# 6. The repository's own fuzz targets: discovery finds exactly the
#    packages whose tracked test files declare one, so a package cannot be
#    left out of make fuzz.
mod="$(cd "$ROOT" && GOWORK=off go list -m)"
want="$(cd "$ROOT" && git ls-files --cached --others --exclude-standard '*_test.go' | grep -v '/testdata/' | xargs grep -l '^func Fuzz' |
	xargs -n1 dirname | sort -u | sed "s|^|$mod/|" | tr '\n' ' ' | sed 's/ $//')"
got="$(cd "$ROOT" && GOWORK=off "$FUZZ" -a -l ./... | sort | tr '\n' ' ' | sed 's/ $//')" || got="discovery failed"
check "the repository's packages with targets are all found" "$want" "$got"
check "the repository has fuzz targets in at least 8 packages" yes "$([ "$(echo "$got" | wc -w)" -ge 8 ] && echo yes || echo no)"

echo "go-fuzz_test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
