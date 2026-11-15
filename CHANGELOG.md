# Changelog

## v0.1.0 — unreleased

First release: Go 1.24 to 1.27, 46 entries.

- `SKILL.md`: ceiling rule, reference loading, twelve reflexes.
- `references/`: 46 entries over four releases, 4 of them mechanical.
  - Go 1.24: 14 entries, 1 mechanical (`stringsseq`).
  - Go 1.25: 8 entries, 1 mechanical (`waitgroupgo`).
  - Go 1.26: 11 entries, 1 mechanical (`errorsastype`).
  - Go 1.27: 13 entries, 1 mechanical (`embedlit`).
- `tools/validate`: compile gate, `go fix` reproduction for mechanical entries,
  fixpoint check for choices.
- `tools/gen`: proposal generator over `doc/next/` and `api/next/`.
- `eval/`: six tasks, scored by remaining `go fix` output.
