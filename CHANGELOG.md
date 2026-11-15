# Changelog

## v0.1.0 — unreleased

First release, Go 1.27.

- `SKILL.md`: ceiling rule, reference loading, twelve reflexes.
- `references/go1.27.md`: 13 entries, 1 mechanical (`embedlit`) and 12 choices.
- `tools/validate`: compile gate, `go fix` reproduction for mechanical entries,
  fixpoint check for choices.
- `tools/gen`: proposal generator over `doc/next/` and `api/next/`.
- `eval/`: six tasks, scored by remaining `go fix` output.
