#!/usr/bin/env bash
# Runs the conformance scenarios against the Go and the Python aism and
# prints, per scenario, whether each implementation passes it.
#
# Usage: test/conformance/compare.sh   (or make conformance-compare)
set -uo pipefail
cd "$(dirname "$0")/../.."
make --no-print-directory build >/dev/null
out="tmp/conformance"
mkdir -p "$out"

run() { # $1 name, $2 executable
  AISM_CLI="$2" go test -count=1 -v ./test/conformance/test/ >"$out/$1.log" 2>&1
  grep -oE -- '--- (PASS|FAIL): TestFeatures/[^ ]+' "$out/$1.log" |
    sed -E 's/--- (PASS|FAIL): TestFeatures\/(.*)/\2 \1/' | sort >"$out/$1.results"
}
run go "$PWD/bin/aism"
run python "$PWD/.venv/bin/aism"

printf '%-6s %-6s %s\n' GO PYTHON SCENARIO
join "$out/go.results" "$out/python.results" |
  awk '{ name=$1; gsub(/_/, " ", name); mark = ($2 == $3) ? "" : "  <-- differs"; printf "%-6s %-6s %s%s\n", $2, $3, name, mark }'
echo "Logs: $out/go.log, $out/python.log"
