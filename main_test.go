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
	code := run("dev", options{}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "-c is required") {
		t.Errorf("stderr = %q, want error about -c flag", stderr.String())
	}
}

func TestRun_InvalidCoverprofile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run("dev", options{coverprofile: "nonexistent.out", paths: []string{"."}}, &stdout, &stderr)
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
	code := run("dev", options{coverprofile: coverprofile, paths: []string{"."}}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CRAP") {
		t.Error("expected CRAP header in output")
	}
}

func TestRun_MaxExceeded(t *testing.T) {
	coverprofile := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(coverprofile, []byte("mode: set\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run("dev", options{coverprofile: coverprofile, max: 1, paths: []string{"."}}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1 (all functions should exceed max 1)", code)
	}
	if !strings.Contains(stderr.String(), "FAIL") {
		t.Error("expected FAIL message in stderr")
	}
}

func TestRun_VerboseShowsAll(t *testing.T) {
	coverprofile := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(coverprofile, []byte("mode: set\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run("dev", options{coverprofile: coverprofile, max: 500, verbose: true, paths: []string{"."}}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Error("expected ok markers in verbose mode")
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
	os.Args = []string{"gocrap", "-c", "test.out", "-max", "12", "-v", "-version", "-exclude", "*_test.go", "-exclude", "*_mock.go", "./..."}

	opts := parseFlags()

	if opts.coverprofile != "test.out" {
		t.Errorf("coverprofile = %q, want %q", opts.coverprofile, "test.out")
	}
	if opts.max != 12 {
		t.Errorf("max = %f, want 12", opts.max)
	}
	if !opts.verbose {
		t.Error("verbose = false, want true")
	}
	if !opts.showVersion {
		t.Error("showVersion = false, want true")
	}
	if len(opts.exclude) != 2 || opts.exclude[0] != "*_test.go" || opts.exclude[1] != "*_mock.go" {
		t.Errorf("exclude = %v, want [*_test.go, *_mock.go]", opts.exclude)
	}
	if len(opts.paths) != 1 || opts.paths[0] != "." {
		t.Errorf("paths = %v, want [\".\"] (./... should be trimmed)", opts.paths)
	}
}

func TestRun_VersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run("v0.99.0-test", options{showVersion: true}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	got := stdout.String()
	if got != "v0.99.0-test\n" {
		t.Errorf("stdout = %q, want \"v0.99.0-test\\n\"", got)
	}
	if stderr.String() != "" {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}
