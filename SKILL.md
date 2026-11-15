---
name: go-modern
description: Write, review or refactor Go with the idioms of the last four releases (Go 1.24 to 1.27), gated on the go directive of the target module. Use before writing new Go code, when modernising an existing package, or when a review should catch outdated patterns in the language or the standard library.
license: Apache-2.0
---

# go-modern

Coding agents reproduce the Go they were trained on. This skill gives you the
idioms of the last four releases, and one rule that overrides all of them: never
use a feature that the target module does not allow.

## 1. Find the ceiling

Read `go.mod` in the module you are editing:

	go 1.26.2

That directive is the ceiling. The `toolchain` line is not: it says which
compiler to run, not what the code may assume. If there is no `go.mod`, there is
no ceiling, and no permission to use anything in this skill.

## 2. Load only what applies

Read every `references/go1.*.md` file whose version is at or below the ceiling.
A module on 1.26 reads 1.24, 1.25 and 1.26; it ignores 1.27 entirely.

If the ceiling is below 1.24, stop: this skill adds nothing to that module.

## 3. Prefer the newer form, without asking

When a section applies, write its "after" block. Not a variant, not a mix of the
two forms, not the old form with a comment explaining the choice. If you are
unsure whether a section applies, it does not: the ceiling decides, not the
release you remember.

An entry marked `mechanical` is one the Go toolchain rewrites itself: `go fix`
produces exactly the "after" block. Everything else is `choice`, where both
forms compile and this skill exists to make you pick the newer one.

## 4. Do not invent

The references are short on purpose. If a feature is not there, the module
cannot use it. Do not fill gaps from memory: run `go doc <package>` and check
that the symbol exists, or look at the release notes.

## Twelve reflexes

These come up all day; apply them without opening a reference. The version in
parentheses introduced the newer form: skip any of them above your ceiling.

1. Loop `n` times: `for range n`, not `for i := 0; i < n; i++` (1.22).
2. Clamp: `min(max(x, lo), hi)`, not two `if` blocks (1.21).
3. Membership: `slices.Contains`, not a loop (1.21).
4. Sorting: `slices.Sort` and `slices.SortFunc`, not `sort.Slice` (1.21).
5. Map keys: `slices.Collect(maps.Keys(m))`, not a preallocated slice and a loop (1.23).
6. Split once: `strings.Cut`, `CutPrefix`, `CutSuffix`, and since 1.27 `CutLast`.
7. Wait group: `wg.Go(f)`, not `wg.Add(1)` plus a closure (1.25).
8. Test context: `t.Context()`, not `context.WithCancel` in the test (1.24).
9. Optional struct fields: `omitzero`, not `omitempty` (1.24).
10. Error typing: `errors.AsType[T](err)`, not `errors.As` plus a variable (1.26).
11. Pointer to a value: `new(v)`, not `p := new(T); *p = v` (1.26).
12. Benchmarks: `for b.Loop()`, not `for i := 0; i < b.N; i++` (1.24).

## Check yourself

- `go fix -diff ./...` must not touch the files you edited.
- `go vet ./...` and `go test ./...`. Since 1.27, `go test` runs the `stdversion`
  check and fails on a standard library symbol newer than the `go` directive.
- In a review, name the entry that justifies each change instead of saying
  "modern Go".
