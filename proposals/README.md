# Proposals

Working directory for `tools/gen`. It reads the pending release notes of the Go
repository and writes `go1.<minor>.md` here.

The files are gitignored: a proposal is a scratchpad, not documentation. Once
the entries pass `tools/validate` and the filter of `CONTRIBUTING.md`, they move
to `references/`.
