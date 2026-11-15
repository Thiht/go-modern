#!/usr/bin/env bash
# Score an agent on eval/tasks, with and without the skill installed.
# See eval/README.md for the contract on EVAL_AGENT and EVAL_AGENT_SKILL.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tasks=$root/eval/tasks
work=$(mktemp -d)
if [ -n "${EVAL_KEEP:-}" ]; then
	trap 'echo "kept $work"' EXIT
else
	trap 'rm -rf "$work"' EXIT
fi

: "${EVAL_AGENT:?set EVAL_AGENT to the baseline agent command}"
skill_cmd=${EVAL_AGENT_SKILL:-}

# score prints "<build> <hunks>": vet status, then what go fix still wants.
score() {
	local dir=$1 build=ok hunks
	if ! (cd "$dir" && GOWORK=off go vet ./... >/dev/null 2>&1); then
		build=build
	fi
	hunks=$(cd "$dir" && GOWORK=off go fix -diff ./... 2>/dev/null | grep -c '^@@' || true)
	printf '%s\t%s' "$build" "$hunks"
}

# run <task dir> <condition> <command> [setup]
run() {
	local src=$1 cond=$2 cmd=$3 setup=${4:-}
	local dir="$work/$cond/$(basename "$src")"
	mkdir -p "$dir"
	cp "$src"/go.mod "$dir"/
	(
		cd "$dir"
		# The setup command installs the skill for this condition, in whatever
		# layout the agent expects (for OpenCode: .opencode/skills/<name>).
		[ -n "$setup" ] && sh -c "$setup" >/dev/null 2>&1
		# The prompt goes on stdin; the agent writes into the scratch module.
		run_with_timeout sh -c "$cmd" <"$src/task.md" >/dev/null 2>&1 || true
	)
}

run_with_timeout() {
	if command -v timeout >/dev/null 2>&1; then
		timeout "${EVAL_TIMEOUT:-180}" "$@"
	else
		"$@"
	fi
}

printf '%-14s %-10s %-10s\n' task base skill
total_base=0 total_skill=0
fail_base=0 fail_skill=0

for src in "$tasks"/*/; do
	name=$(basename "$src")
	run "$src" base "$EVAL_AGENT" "" || true
	base=$(score "$work/base/$name")

	if [ -n "$skill_cmd" ]; then
		run "$src" skill "$skill_cmd" "${EVAL_SETUP_SKILL:-}" || true
		skl=$(score "$work/skill/$name")
	else
		skl=$'\t-'
	fi

	printf '%-14s %-10s %-10s\n' "$name" "${base//$'\t'/:}" "${skl//$'\t'/:}"
	total_base=$((total_base + ${base#*$'\t'}))
	[[ $base == build* ]] && fail_base=$((fail_base + 1))
	if [ -n "$skill_cmd" ]; then
		total_skill=$((total_skill + ${skl#*$'\t'}))
		[[ $skl == build* ]] && fail_skill=$((fail_skill + 1))
	fi
done

printf '%-14s %-10s %-10s\n' "total hunks" "$total_base" "${skill_cmd:+$total_skill}"
printf '%-14s %-10s %-10s\n' "build failures" "$fail_base" "${skill_cmd:+$fail_skill}"
