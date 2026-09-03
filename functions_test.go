package main

import (
	"os"
	"path/filepath"
	"testing"
)

// gocyclo (complexity pass) skips dot-dirs, "_"-prefixed dirs, testdata and
// vendor. The coverage pass must apply the same rules, otherwise stale copies
// of the codebase (e.g. .worktrees/) pollute the line-indexed coverage join
// and produce non-deterministic 0% coverage attribution.
func TestExtractAllFunctions_SkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	writeFile := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("main.go", "package main\n\nfunc real() {}\n")
	writeFile(filepath.Join(".worktrees", "old-copy", "main.go"), "package main\n\nfunc staleWorktree() {}\n")
	writeFile(filepath.Join(".idea", "scratch", "main.go"), "package main\n\nfunc staleDotDir() {}\n")
	writeFile(filepath.Join("_hidden", "main.go"), "package main\n\nfunc staleUnderscore() {}\n")
	writeFile(filepath.Join("testdata", "fixture.go"), "package main\n\nfunc staleTestdata() {}\n")
	writeFile(filepath.Join("vendor", "dep", "dep.go"), "package dep\n\nfunc staleVendor() {}\n")

	fns, err := extractAllFunctions([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(fns) != 1 || fns[0].Name != "real" {
		var names []string
		for _, fn := range fns {
			names = append(names, fn.Name)
		}
		t.Errorf("extractAllFunctions = %v, want only [real]", names)
	}
}
