# go-modern

A skill for coding agents that makes them write Go the way the module allows,
not the way their training data did.

Models learn Go from a corpus that is years behind the current release, and even
when they know a newer idiom they default to the older one because it is more
frequent in that corpus. Asking for "modern Go" in general terms does not fix
it. Concrete before/after pairs, gated on a version, do.

## Install

```bash
npx skills add Thiht/go-modern
```

The CLI installs the skill into the agent of your choice (Claude Code, Codex,
OpenCode, Cursor and the rest). Without the CLI, copy `SKILL.md` and
`references/` into your agent's skills directory as `go-modern/`.

## How it works

1. The agent reads the `go` directive of the module it is editing. That is the
   ceiling; the `toolchain` line is not.
2. It reads the reference file of every release at or below the ceiling.
3. It writes the modern form, and never a feature above the ceiling.

Every entry carries the release that introduced it, so a module on 1.25 can
never be handed a 1.27 feature.

## Coverage

| Release | Entries |
|---------|---------|
| Go 1.27 | 13 |
| Go 1.26 | 11 |
| Go 1.25 | 8 |
| Go 1.24 | 14 |

The counts differ because the releases do: 1.27 was mostly runtime and tooling,
while 1.24 shipped a long list of library additions. Padding a thin release with
entries nobody would act on would make the whole file less credible.

The window slides: when Go 1.28 ships, 1.24 leaves and 1.28 joins.

## What is checked

- Every "after" block compiles with the toolchain of the release it claims.
- Every "mechanical" entry is reproduced exactly by `go fix`, so the reference
  cannot drift from what the toolchain actually does. The modernizers only
  reached `go fix` in Go 1.26, so the check runs the newest toolchain of the
  window, with the `go` directive of the entry's own release.
- Every "choice" entry is already a `go fix` fixpoint.

`eval/` holds the behavioural side: the same tasks run with and without the
skill, scored by how much `go fix` still has to say.

## Scope

Language changes and standard library APIs that alter a decision you make while
writing code. Not the toolchain, not the runtime, not the ports, not
`GOEXPERIMENT` features.

## Relation to go-modern-guidelines

JetBrains publishes [go-modern-guidelines](https://github.com/JetBrains/go-modern-guidelines),
which covers Go 1.0 to 1.27 and ships a CLI that injects the rules that match
your module. It is broader.

This repository is narrower on purpose: four releases, one file per release, no
binary to install, and every example proved by the compiler. The two can be
installed side by side; if they disagree, the compiler wins.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Licenced under Apache-2.0.
