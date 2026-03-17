package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_NoCoverprofile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(options{}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "-coverprofile is required") {
		t.Errorf("stderr = %q, want error about coverprofile", stderr.String())
	}
}

func TestRun_InvalidCoverprofile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(options{coverprofile: "nonexistent.out", paths: []string{"."}}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestRun_Integration(t *testing.T) {
	coverprofile := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(coverprofile, []byte("mode: set\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run(options{coverprofile: coverprofile, paths: []string{"."}}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CRAP") {
		t.Error("expected CRAP header in output")
	}
}

func TestRun_ThresholdExceeded(t *testing.T) {
	coverprofile := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(coverprofile, []byte("mode: set\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run(options{coverprofile: coverprofile, threshold: 1, paths: []string{"."}}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1 (all functions should exceed threshold 1)", code)
	}
	if !strings.Contains(stderr.String(), "FAIL") {
		t.Error("expected FAIL message in stderr")
	}
}

func TestParseFlags(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"gocrap", "-coverprofile", "test.out", "-threshold", "12", "-no-tests", "./..."}

	opts := parseFlags()

	if opts.coverprofile != "test.out" {
		t.Errorf("coverprofile = %q, want %q", opts.coverprofile, "test.out")
	}
	if opts.threshold != 12 {
		t.Errorf("threshold = %f, want 12", opts.threshold)
	}
	if !opts.noTests {
		t.Error("noTests = false, want true")
	}
	if len(opts.paths) != 1 || opts.paths[0] != "." {
		t.Errorf("paths = %v, want [\".\"] (./... should be trimmed)", opts.paths)
	}
}
