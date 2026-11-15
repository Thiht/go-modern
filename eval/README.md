# Eval

The validator proves that the references are true. It does not prove that the
skill changes what a model writes. This directory measures that.

Each task is a small module with a `go.mod` at a target version and a prose
prompt. The same prompt runs twice: with the agent as you normally use it, and
with the same agent having the skill installed. The score is the number of
hunks `go fix -diff` still wants to apply to the result: fewer is better, and
zero means the agent wrote code the toolchain has nothing to say about.

## Run it

```bash
EVAL_AGENT='claude -p' \
EVAL_AGENT_SKILL='claude -p' \
./eval/run.sh
```

Both variables hold a single shell command. The difference between the two is
whatever makes the skill visible to the agent. With OpenCode, the setup command
copies it into the project layout the CLI reads:

```bash
EVAL_AGENT='opencode run --auto "$(cat)"' \
EVAL_AGENT_SKILL='opencode run --auto "$(cat)"' \
EVAL_SETUP_SKILL='mkdir -p .opencode/skills/go-modern &&
  cp -R ~/code/go-modern/SKILL.md ~/code/go-modern/references .opencode/skills/go-modern/' \
./eval/run.sh
```

Without `EVAL_AGENT_SKILL`, only the baseline runs. `EVAL_TIMEOUT` caps each
agent invocation, in seconds; the default is 180.

## Reading the score

```
task           base       skill
clamp          2          0
decode         build      0
```

`build` means the agent produced code that does not compile. A task where the
baseline already scores zero measures nothing; replace it.

## Limits

Six tasks and one run are a smoke test, not a study. Models are
non-deterministic: run each condition three times before drawing a conclusion,
and change one variable at a time. Do not tune `SKILL.md` on a single number.
