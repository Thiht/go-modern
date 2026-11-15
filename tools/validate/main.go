// Command validate checks every entry of the reference files.
//
// An entry is a markdown section whose heading is the idiom, followed by a
// metadata line and two Go blocks: the old form first, the new form second.
//
//	## Promoted fields as literal keys
//
//	> since 1.27 · mechanical · analyzer: embedlit
//
// The second block must compile with the toolchain of the declared release.
// A mechanical entry additionally proves the rewrite: the first block compiles
// one release earlier, and `go fix` turns it into the second block, byte for
// byte after formatting. A choice entry proves the second block is already a
// `go fix` fixpoint, so a later analyzer has nothing left to say about it.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type entry struct {
	File     string
	Name     string
	Since    string
	Kind     string
	Analyzer string
	Before   string
	After    string
}

func (e entry) toolchain() string { return "go" + e.Since + ".0" }

func (e entry) previous() string {
	parts := strings.SplitN(e.Since, ".", 2)
	if len(parts) != 2 {
		return ""
	}
	var minor int
	fmt.Sscanf(parts[1], "%d", &minor)
	return fmt.Sprintf("%s.%d", parts[0], minor-1)
}

func (e entry) previousToolchain() string { return "go" + e.previous() + ".0" }

var (
	headRE = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	metaRE = regexp.MustCompile(`^>\s+since\s+(\d+\.\d+)\s+·\s+(mechanical|choice)(?:\s+·\s+analyzer:\s+([a-z0-9]+))?\s*$`)
)

func parseFile(path string) ([]entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var (
		entries []entry
		cur     *entry
		inGo    bool
		block   strings.Builder
		blocks  []string
	)
	flush := func() error {
		if cur == nil {
			return nil
		}
		switch {
		case cur.Since == "":
			return fmt.Errorf("%s: entry %q has no metadata line", path, cur.Name)
		case len(blocks) != 2:
			return fmt.Errorf("%s: entry %q has %d Go blocks, want 2", path, cur.Name, len(blocks))
		case cur.Kind == "mechanical" && cur.Analyzer == "":
			return fmt.Errorf("%s: entry %q is mechanical without analyzer", path, cur.Name)
		}
		cur.Before, cur.After = blocks[0], blocks[1]
		entries = append(entries, *cur)
		cur, blocks = nil, nil
		return nil
	}

	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case inGo:
			if strings.TrimSpace(line) == "```" {
				inGo = false
				blocks = append(blocks, block.String())
				block.Reset()
				continue
			}
			block.WriteString(line)
			block.WriteByte('\n')
		case strings.HasPrefix(line, "```go"):
			if cur == nil {
				return nil, fmt.Errorf("%s: Go block outside an entry", path)
			}
			inGo = true
		case headRE.MatchString(line):
			if err := flush(); err != nil {
				return nil, err
			}
			cur = &entry{File: path, Name: headRE.FindStringSubmatch(line)[1]}
		case metaRE.MatchString(line) && cur != nil:
			m := metaRE.FindStringSubmatch(line)
			cur.Since, cur.Kind, cur.Analyzer = m[1], m[2], m[3]
		}
	}
	if inGo {
		return nil, fmt.Errorf("%s: unterminated Go block in entry %q", path, cur.Name)
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return entries, nil
}

func withPackage(src string) string {
	if strings.HasPrefix(strings.TrimSpace(src), "package ") {
		return src
	}
	return "package entry\n\n" + src
}

func run(dir, toolchain string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOTOOLCHAIN="+toolchain,
		"GOWORK=off",
		"CGO_ENABLED=0",
	)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}

func setup(dir, since, src string) error {
	gomod := fmt.Sprintf("module entry\n\ngo %s.0\n", since)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "entry.go"), []byte(withPackage(src)), 0o644)
}

func formatted(src string) string {
	out, err := format.Source([]byte(src))
	if err != nil {
		return src
	}
	return string(out)
}

func same(a, b string) bool { return formatted(a) == formatted(b) }

func check(e entry, fixTC string) error {
	dir, err := os.MkdirTemp("", "go-modern-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	// The new form must compile on the release that introduced it.
	if err := setup(dir, e.Since, e.After); err != nil {
		return err
	}
	if out, err := run(dir, e.toolchain(), "build", "./..."); err != nil {
		return fmt.Errorf("after does not compile on go%s:\n%s", e.Since, out)
	}

	switch e.Kind {
	case "choice":
		// Nothing mechanical should be left to rewrite.
		if out, err := run(dir, fixTC, "fix", "./..."); err != nil {
			return fmt.Errorf("go fix failed on after:\n%s", out)
		}
		got, err := os.ReadFile(filepath.Join(dir, "entry.go"))
		if err != nil {
			return err
		}
		if !same(string(got), e.After) {
			return fmt.Errorf("after is not a go fix fixpoint at go%s:\n%s", e.Since, diff(e.After, string(got)))
		}
	case "mechanical":
		// The old form must compile one release earlier...
		if err := setup(dir, e.previous(), e.Before); err != nil {
			return err
		}
		if out, err := run(dir, e.previousToolchain(), "build", "./..."); err != nil {
			return fmt.Errorf("before does not compile on %s:\n%s", e.previousToolchain(), out)
		}
		// ...and go fix must produce exactly the new form.
		if err := setup(dir, e.Since, e.Before); err != nil {
			return err
		}
		if out, err := run(dir, fixTC, "fix", "./..."); err != nil {
			return fmt.Errorf("go fix failed on before:\n%s", out)
		}
		got, err := os.ReadFile(filepath.Join(dir, "entry.go"))
		if err != nil {
			return err
		}
		if !same(string(got), e.After) {
			return fmt.Errorf("go fix does not produce the after block:\n%s", diff(e.After, string(got)))
		}
	default:
		return fmt.Errorf("unknown kind %q", e.Kind)
	}
	return nil
}

func diff(want, got string) string {
	w, g := strings.Split(formatted(want), "\n"), strings.Split(formatted(got), "\n")
	var b strings.Builder
	b.WriteString("--- want\n+++ got\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		switch {
		case i >= len(g):
			fmt.Fprintf(&b, "-%s\n", w[i])
		case i >= len(w):
			fmt.Fprintf(&b, "+%s\n", g[i])
		case w[i] != g[i]:
			fmt.Fprintf(&b, "-%s\n+%s\n", w[i], g[i])
		}
	}
	return b.String()
}

func main() {
	refs := flag.String("refs", "references", "directory holding the reference files")
	only := flag.String("only", "", "only check entries whose name contains this string")
	jobs := flag.Int("jobs", 4, "entries checked in parallel")
	fixVersion := flag.String("fix", "", "release whose go fix runs the modernizers (default: the newest reference)")
	verbose := flag.Bool("v", false, "print every entry")
	flag.Parse()

	files, err := filepath.Glob(filepath.Join(*refs, "*.md"))
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "validate: no reference files under %s\n", *refs)
		os.Exit(1)
	}
	sort.Strings(files)

	var entries []entry
	for _, f := range files {
		parsed, err := parseFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "validate: %v\n", err)
			os.Exit(1)
		}
		entries = append(entries, parsed...)
	}
	if *only != "" {
		kept := entries[:0]
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Name), strings.ToLower(*only)) {
				kept = append(kept, e)
			}
		}
		entries = kept
	}
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "validate: no entry matched")
		os.Exit(1)
	}

	// The modernizers only reached go fix in Go 1.26, so an entry from an
	// older release is proved with the newest toolchain of the window, with the
	// go directive of its own release: the analyzer has to be there, and it must
	// still refuse to fire above that release.
	fix := *fixVersion
	if fix == "" {
		for _, e := range entries {
			if e.Since > fix {
				fix = e.Since
			}
		}
	}
	fixTC := "go" + fix + ".0"

	// Warm the toolchain cache sequentially: concurrent downloads race.
	seen := map[string]bool{}
	for _, e := range entries {
		for _, tc := range []string{e.toolchain(), e.previousToolchain(), fixTC} {
			if !seen[tc] {
				seen[tc] = true
				run(os.TempDir(), tc, "version")
			}
		}
	}

	type result struct {
		entry entry
		err   error
	}
	results := make([]result, len(entries))
	sem := make(chan struct{}, *jobs)
	var wg sync.WaitGroup
	for i, e := range entries {
		wg.Add(1)
		go func(i int, e entry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = result{e, check(e, fixTC)}
		}(i, e)
	}
	wg.Wait()

	failed := 0
	lastFile := ""
	for _, r := range results {
		if r.entry.File != lastFile {
			fmt.Printf("%s\n", r.entry.File)
			lastFile = r.entry.File
		}
		if r.err != nil {
			failed++
			fmt.Printf("  FAIL  %s\n", r.entry.Name)
			for _, line := range strings.Split(strings.TrimRight(r.err.Error(), "\n"), "\n") {
				fmt.Printf("        %s\n", line)
			}
			continue
		}
		if *verbose {
			tag := r.entry.Kind
			if r.entry.Analyzer != "" {
				tag += ", " + r.entry.Analyzer
			}
			fmt.Printf("  ok    %-46s (%s)\n", r.entry.Name, tag)
		}
	}
	if failed > 0 {
		fmt.Printf("\n%d of %d entries failed\n", failed, len(results))
		os.Exit(1)
	}
	fmt.Printf("\n%d entries ok\n", len(results))
}
