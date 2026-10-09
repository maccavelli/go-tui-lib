#!/usr/bin/env bash
# Fuzz every fuzz target in one package, for a fixed time each. Taken from
# go-selfupdate-lib (then go-core-lib) under docs/decisions/0002-PLAN-multi-pane-workspace-layouts.md
# Step 5 (go-selfupdate-lib docs/decisions/0004-PLAN-h2-fuzzing-and-manifest-differential.md
# Step 4). go test -fuzz takes exactly one fuzz target per
# run, so the targets run one after another.
#
# Usage: go-fuzz.sh [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PKG
#        go-fuzz.sh -a [-l] [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PATTERN...
#   -t FUZZTIME      time per target, as go test -fuzztime takes it
#                    (default 20s)
#   -z MINIMIZETIME  the cap on minimizing one input, as go test
#                    -fuzzminimizetime takes it (default 5s). Go's own
#                    default, 60s, lets the minimizer pause fuzzing for most
#                    of a short run (deviation D2 of go-selfupdate-lib
#                    docs/decisions/0004-PLAN-h2-fuzzing-and-manifest-differential.md)
#   -m MIN           the fewest targets accepted (default 4, at least 1): a
#                    loop that finds nothing must not pass
#   -a               discover: PATTERN... are package patterns, as go list
#                    takes them; every package whose test files declare a
#                    func Fuzz… is fuzzed, MIN applying to each, and the
#                    others are skipped. Finding no package is an error
#                    (docs/decisions/0014-PLAN-hardening.md Step 9, H9)
#   -l               with -a, print the packages found, one import path per
#                    line, and fuzz nothing
#
# GO names the go command (default go); the tests inject a recorder
# through it.
#
# Exit 0 when every target ran clean; on a fuzz failure, go test's status; 1
# when too few targets or packages are found, or they cannot be listed; 2 on
# a usage error. Go leaves a failing input in PKG/testdata/fuzz/<Name>/.
set -euo pipefail

usage() {
	echo "usage: go-fuzz.sh [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PKG" >&2
	echo "       go-fuzz.sh -a [-l] [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PATTERN..." >&2
	exit 2
}

GO="${GO:-go}"
fuzztime=20s
minimize=5s
min=4
discover=0
list=0
while getopts ':t:z:m:al' opt; do
	case "$opt" in
	t) fuzztime="$OPTARG" ;;
	z) minimize="$OPTARG" ;;
	m) min="$OPTARG" ;;
	a) discover=1 ;;
	l) list=1 ;;
	*) usage ;;
	esac
done
shift $((OPTIND - 1))
case "$min" in
'' | *[!0-9]* | 0) usage ;;
esac
if [ "$discover" -eq 0 ]; then
	[ "$list" -eq 0 ] && [ $# -eq 1 ] || usage
else
	[ $# -ge 1 ] || usage
fi

# fuzz_pkg PKG [DIR] fuzzes every target in PKG, one after another; DIR is
# where its corpus is, PKG itself unless given.
fuzz_pkg() {
	local pkg="$1" dir="${2:-$1}" listing line target rc
	# The listing's status is taken before anything filters it.
	if ! listing="$("$GO" test -list '^Fuzz' "$pkg")"; then
		echo "go-fuzz: cannot list the fuzz targets in $pkg" >&2
		exit 1
	fi
	local targets=()
	while IFS= read -r line; do
		case "$line" in
		Fuzz*) targets+=("$line") ;;
		esac
	done <<<"$listing"

	if [ "${#targets[@]}" -lt "$min" ]; then
		echo "go-fuzz: found ${#targets[@]} fuzz targets in $pkg, want at least $min" >&2
		exit 1
	fi

	for target in "${targets[@]}"; do
		echo "go-fuzz: $target for $fuzztime (minimizing at most $minimize)"
		rc=0
		"$GO" test -run '^$' -fuzz "^${target}\$" -fuzztime "$fuzztime" -fuzzminimizetime "$minimize" "$pkg" || rc=$?
		if [ "$rc" -ne 0 ]; then
			echo "go-fuzz: $target failed (exit $rc); Go wrote the failing input under $dir/testdata/fuzz/$target/" >&2
			exit "$rc"
		fi
	done
	echo "go-fuzz: ${#targets[@]} fuzz targets ran clean in $pkg"
}

if [ "$discover" -eq 0 ]; then
	fuzz_pkg "$1"
	exit 0
fi

# Discovery: each package's directory and test files, tab-separated, from
# one go list run whose status is taken before anything reads it.
tab="$(printf '\t')"
format='{{.ImportPath}}{{"\t"}}{{.Dir}}{{range .TestGoFiles}}{{"\t"}}{{.}}{{end}}{{range .XTestGoFiles}}{{"\t"}}{{.}}{{end}}'
if ! packages="$("$GO" list -f "$format" "$@")"; then
	echo "go-fuzz: cannot list the packages in $*" >&2
	exit 1
fi
found=()
dirs=()
while IFS="$tab" read -r -a fields; do
	[ "${#fields[@]}" -ge 3 ] || continue # no test files
	dir="${fields[1]}"
	for file in "${fields[@]:2}"; do
		if grep -q '^func Fuzz' "$dir/$file"; then
			found+=("${fields[0]}")
			dirs+=("$dir")
			break
		fi
	done
done <<<"$packages"

if [ "${#found[@]}" -eq 0 ]; then
	echo "go-fuzz: found no package with fuzz targets in $*" >&2
	exit 1
fi
if [ "$list" -eq 1 ]; then
	printf '%s\n' "${found[@]}"
	exit 0
fi
for i in "${!found[@]}"; do
	fuzz_pkg "${found[$i]}" "${dirs[$i]}"
done
echo "go-fuzz: ${#found[@]} packages ran clean"
