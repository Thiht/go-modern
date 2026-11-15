# Contributing

The value of this repository is that nothing in it can be wrong without the
toolchain noticing. Keep it that way.

## What qualifies

An entry must change a decision someone makes while writing code, and it must
come from one of the four releases in the window. If a reader would say "I would
never have written it the other way anyway", the entry is noise.

Excluded, whatever the release notes say:

- the toolchain, the runtime, the ports, and `GOEXPERIMENT` features;
- performance work with no API change;
- a new symbol nobody reaches for outside its package's own domain.

## Entry format

One section per idiom, in `references/go1.<minor>.md`, newest release last.

```markdown
## <Name, as you would say it>

> since 1.27 · choice

One or two sentences: when to reach for it, and the trap it avoids.

**Before**

```go
package entry

...
```

**After**

```go
package entry

...
```
```

The metadata line is machine-read: `since`, `kind`, and `analyzer` for
mechanical entries. Blocks are complete Go files, `package entry` included: the
import path of a new package is part of what the reader has to learn.

`mechanical` means `go fix` rewrites the old form into the new one. `choice`
means both compile and the newer one is better. If a modernizer exists but the
rewrite is lossy, the entry is a `choice`.

## Check your work

```bash
cd tools
go run ./validate -refs ../references -v -only <substring>
```

The validator proves, for each entry:

- the "after" block compiles on the release named by `since`;
- a mechanical entry's "before" block compiles one release earlier, and
  `go fix` turns it into the "after" block byte for byte;
- a choice entry's "after" block is already a `go fix` fixpoint.

`go fix` only grew its modernizers in Go 1.26, so the fixer runs the newest
toolchain of the window while the `go` directive stays at the entry's release.
That is what makes a 1.24 entry provable: the analyzer exists, and it still
refuses to fire above the release that owns it.

If `go fix` produces something slightly different from your after block, the
after block is wrong: copy what the toolchain writes.

## Adding a release

Read `doc/next/` in the Go repository while the release is in development, and
`api/go1.<minor>.txt` once it ships. The first says what changed and why, the
second lists every new symbol, so nothing is missed.

1. Keep what passes the filter above, drop the rest.
2. Write the before/after pairs. For a package with a mechanical modernizer,
   let `go fix` write the after block.
3. Run the validator until it is green.
4. Add the file to the coverage table of the README and to the CHANGELOG.

## Sliding the window

When a release leaves the window, delete its file and its entries. Git keeps the
history; the skill keeps its promise of four releases.

## Style

- English, imperative, no marketing.
- One idea per entry. If a feature needs two paragraphs, it is two entries.
- No code comment restating the signature.
