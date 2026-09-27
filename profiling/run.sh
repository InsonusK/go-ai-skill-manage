#!/usr/bin/env bash
# Profiles `aism sync` on the skills of profiling/ai-skill*.yaml.
#
# Usage: profiling/run.sh [local|github]
#   local  (default) -- skills from a local clone (profiling/sources/ai-skills,
#                        cloned once), no network during the runs;
#   github           -- ai-skill.yaml as is: the repository is downloaded on
#                        every run.
#
# Each mode runs sync twice: "cold" into empty targets (every skill is
# created) and "warm" into the targets the cold run wrote (every skill is
# updated). Per run it keeps, in profiling/out/<mode>/:
#   <run>.cpu.prof, <run>.mem.prof -- CPU and heap profiles;
#   <run>.log                      -- stdout and stderr of aism;
# and prints the wall time, the memory totals and the top CPU functions.
set -euo pipefail

cd "$(dirname "$0")/.."
mode="${1:-local}"
repo="https://github.com/InsonusK/ai-skills.git"
branch="master"

case "$mode" in
local)
  config="profiling/ai-skill.local.yaml"
  targets=(profiling/out/local/.agents profiling/out/local/.claude)
  if [ ! -d profiling/sources/ai-skills/.git ]; then
    echo "cloning $repo ($branch) into profiling/sources/ai-skills"
    git clone --quiet --depth 1 --branch "$branch" "$repo" profiling/sources/ai-skills
  fi
  ;;
github)
  config="profiling/ai-skill.yaml"
  targets=(profiling/.agents profiling/.claude)
  ;;
*)
  echo "usage: $0 [local|github]" >&2
  exit 2
  ;;
esac

out="profiling/out/$mode"
mkdir -p "$out"
make --no-print-directory build >/dev/null
rm -rf "${targets[@]}"

for run in cold warm; do
  log="$out/$run.log"
  start=$(date +%s%N)
  status=0
  ./bin/aism --profile \
    --profile-output "$out/$run.cpu.prof" \
    --mem-profile-output "$out/$run.mem.prof" \
    sync --config "$config" >"$log" 2>&1 || status=$?
  end=$(date +%s%N)
  echo "== $mode / $run: exit $status, $(( (end - start) / 1000000 )) ms"
  grep -E "^(Synced|Dry run)" "$log" || true
  grep -o 'msg=memory.*' "$log" || true
  go tool pprof -top -nodecount=15 "$out/$run.cpu.prof" 2>/dev/null | sed -n '1,25p'
  if [ "$status" -ne 0 ]; then
    # The profile above covers what ran before the failure (e.g. loading and
    # validating the skills); the warm run needs the cold run's targets.
    echo "aism failed, see $log" >&2
    exit "$status"
  fi
done
