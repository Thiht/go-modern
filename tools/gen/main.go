// Command gen turns the pending release notes of the Go repository into a
// proposal for the next reference file.
//
// It reads doc/next/ (the language and standard library changes being written
// for the next release) and api/next/ (the symbols the API checker will accept),
// and writes proposals/go1.<minor>.md. It never touches references/: a proposal
// is a draft that a human turns into entries and that tools/validate must prove.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"
)

type source interface {
	// List returns every file under dir, recursively, as paths relative to the
	// repository root.
	List(dir string) ([]string, error)
	Read(path string) ([]byte, error)
}

// dirSource reads a checkout, or the fixtures of the tests.
type dirSource struct{ root string }

func (d dirSource) List(dir string) ([]string, error) {
	var paths []string
	base := filepath.Join(d.root, filepath.FromSlash(dir))
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(d.root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func (d dirSource) Read(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.root, filepath.FromSlash(path)))
}

// githubSource reads golang/go through the REST API. One tree call per
// directory listing keeps the request count low enough for the anonymous quota.
type githubSource struct {
	owner, repo, ref string
	client           *http.Client
}

func (g githubSource) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "go-modern-gen")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return nil, fmt.Errorf("%s: %s: %s", url, res.Status, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(res.Body)
}

func (g githubSource) List(dir string) ([]string, error) {
	parent, name := filepath.ToSlash(filepath.Dir(dir)), filepath.Base(dir)
	raw, err := g.get(fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", g.owner, g.repo, parent, g.ref))
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Name, Sha, Type string
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("listing %s: %w", parent, err)
	}
	sha := ""
	for _, e := range entries {
		if e.Name == name && e.Type == "dir" {
			sha = e.Sha
		}
	}
	if sha == "" {
		return nil, fmt.Errorf("%s: no such directory", dir)
	}

	raw, err = g.get(fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", g.owner, g.repo, sha))
	if err != nil {
		return nil, err
	}
	var tree struct {
		Tree []struct {
			Path string
			Type string
		}
	}
	if err := json.Unmarshal(raw, &tree); err != nil {
		return nil, fmt.Errorf("tree of %s: %w", dir, err)
	}
	var paths []string
	for _, e := range tree.Tree {
		if e.Type == "blob" {
			paths = append(paths, dir+"/"+e.Path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func (g githubSource) Read(path string) ([]byte, error) {
	return g.get(fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.owner, g.repo, g.ref, path))
}

type pkg struct {
	Name    string
	Symbols []string
}

func (p pkg) Count() string {
	if len(p.Symbols) == 1 {
		return "1 new symbol"
	}
	return fmt.Sprintf("%d new symbols", len(p.Symbols))
}

type proposal struct {
	Version  string
	Owner    string
	Repo     string
	Ref      string
	Language string
	Stdlib   []struct{ Path, Body string }
	Packages []pkg
}

var issueRE = regexp.MustCompile(`\s+#\d+\s*$`)

// trimHeading drops the section heading of a release-notes fragment: the
// proposal supplies its own, and two in a row read like a mistake.
func trimHeading(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "## ") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[i+1:])
	}
	return ""
}

func groupSymbols(files map[string][]byte) []pkg {
	byPkg := map[string][]string{}
	seen := map[string]bool{}
	for _, raw := range files {
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "pkg ") {
				continue
			}
			line = issueRE.ReplaceAllString(line, "")
			line = strings.ReplaceAll(line, " ideal-int", "")
			if seen[line] {
				continue
			}
			seen[line] = true

			rest := strings.TrimPrefix(line, "pkg ")
			name, decl, ok := strings.Cut(rest, ", ")
			if !ok {
				continue
			}
			// Platform variants repeat the same declaration once per GOOS and
			// GOARCH; one entry per symbol is enough for a proposal.
			if i := strings.Index(name, " ("); i >= 0 {
				name = name[:i]
			}
			if seen[name+"\x00"+decl] {
				continue
			}
			seen[name+"\x00"+decl] = true
			byPkg[name] = append(byPkg[name], decl)
		}
	}
	var out []pkg
	for name, syms := range byPkg {
		sort.Strings(syms)
		out = append(out, pkg{name, syms})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Symbols) != len(out[j].Symbols) {
			return len(out[i].Symbols) > len(out[j].Symbols)
		}
		return out[i].Name < out[j].Name
	})
	return out
}

var page = template.Must(template.New("proposal").Parse(`# Go {{.Version}} — proposal

> Generated from ` + "`{{.Owner}}/{{.Repo}}@{{.Ref}}`" + ` by tools/gen.
> Nothing here is validated: an entry moves to ` + "`references/`" + ` only once
> tools/validate proves it. Keep what passes the filter in CONTRIBUTING.md, drop
> the rest.

## Changes to the language

{{.Language}}

## Standard library changes

{{range .Stdlib}}### {{.Path}}

{{.Body}}

{{end}}## New API symbols

{{range .Packages}}### {{.Name}}

{{range .Symbols}}- ` + "`{{.}}`" + `
{{end}}
{{end}}## Candidate packages

{{range .Packages}}- [ ] ` + "`{{.Name}}`" + ` — {{.Count}}
{{end}}`))

func run(src source, version, owner, repo, ref string) ([]byte, error) {
	paths, err := src.List("doc/next")
	if err != nil {
		return nil, err
	}
	var stdlib []struct{ Path, Body string }
	var language string
	for _, p := range paths {
		if !strings.HasSuffix(p, ".md") || strings.HasSuffix(p, "0-heading.md") {
			continue
		}
		body, err := src.Read(p)
		if err != nil {
			return nil, err
		}
		switch {
		case strings.HasSuffix(p, "/2-language.md"):
			language = trimHeading(string(body))
		case strings.Contains(p, "/6-stdlib/"):
			stdlib = append(stdlib, struct{ Path, Body string }{p, trimHeading(string(body))})
		}
	}
	if language == "" {
		return nil, fmt.Errorf("doc/next/2-language.md not found")
	}

	apiPaths, err := src.List("api/next")
	if err != nil {
		return nil, err
	}
	apiFiles := map[string][]byte{}
	for _, p := range apiPaths {
		body, err := src.Read(p)
		if err != nil {
			return nil, err
		}
		apiFiles[p] = body
	}

	var buf strings.Builder
	err = page.Execute(&buf, proposal{
		Version:  version,
		Owner:    owner,
		Repo:     repo,
		Ref:      ref,
		Language: language,
		Stdlib:   stdlib,
		Packages: groupSymbols(apiFiles),
	})
	return []byte(buf.String()), err
}

func main() {
	version := flag.String("version", "", "release to draft, for example 1.28")
	ref := flag.String("ref", "master", "branch or tag of golang/go to read")
	out := flag.String("out", "", "output file (default proposals/go1.<minor>.md)")
	checkout := flag.String("source", "", "read a local checkout instead of GitHub")
	flag.Parse()

	if *version == "" {
		fmt.Fprintln(os.Stderr, "gen: -version is required")
		os.Exit(2)
	}

	var src source = githubSource{"golang", "go", *ref, &http.Client{Timeout: 30 * time.Second}}
	if *checkout != "" {
		src = dirSource{*checkout}
	}

	body, err := run(src, *version, "golang", "go", *ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}

	if *out == "" {
		*out = filepath.Join("proposals", "go"+*version+".md")
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, body, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", *out)
}
