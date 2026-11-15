# Results

## 2026-11-16 — OpenCode, gemini-4-hydrogen

```bash
EVAL_AGENT='opencode run --auto "$(cat)"' \
EVAL_AGENT_SKILL='opencode run --auto "$(cat)"' \
EVAL_SETUP_SKILL='mkdir -p .opencode/skills/go-modern &&
  cp -R /Users/thibaut/code/go-modern/SKILL.md /Users/thibaut/code/go-modern/references .opencode/skills/go-modern/' \
./eval/run.sh
```

| task | base | skill |
|------|------|-------|
| bench | ok:0 | ok:0 |
| clamp | ok:0 | ok:0 |
| decode | ok:0 | ok:0 |
| envvalue | ok:0 | ok:0 |
| keys | ok:0 | ok:0 |
| withhost | ok:0 | ok:0 |

Build failures: 0 in both conditions. The skill was installed for the second
condition and absent from the first, checked in the scratch directories.

**The score does not discriminate.** `go fix -diff` only sees mechanical
rewrites, and 43 of the 47 entries are choices it cannot see. The `clamp` task
scored 0 in both conditions while producing visibly different code:

```go
// base
if n < 0 {
	return 0
}
if n > 100 {
	return 100
}
return n
```

```go
// skill
return min(max(x, 0), 100)
```

The `minmax` analyzer fires on assignments, not on early returns, so the base
looks modern to the metric.

What this run establishes: the harness works end to end. What it does not
establish: that the skill changes anything measurable. Each task needs to assert
the construct it expects instead of delegating the judgement to `go fix`.
