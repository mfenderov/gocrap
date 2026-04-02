# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

gocrap computes CRAP scores (Change Risk Anti-Patterns) for Go functions. It combines cyclomatic complexity (via gocyclo) with test coverage (via `go tool cover -func`) to identify risky, under-tested code.

**CRAP formula**: `complexity² × (1 - coverage/100)³ + complexity`

**Repo**: github.com/mfenderov/gocrap (public)

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run TestCRAPScore ./...

# Build
go build -o gocrap .

# Install from source
go install .

# Vet
go vet ./...

# Dogfood: run gocrap on itself (same as CI)
go test -coverprofile=coverage.out ./... && go run . -c coverage.out -max 5 -exclude '*_test.go' -exclude '*_mock.go' -v .
```

## Architecture

Flat single-package (`main`) CLI with three source files:

- **`main.go`** — CLI flags (`stringSlice` for repeated `-exclude`), orchestration (`run` → `analyze` → `printReport` → `checkMax`)
- **`analyzer.go`** — Pure domain logic: CRAP formula, coverage parser (`parseCoverFunc` → `parseCoverLine` → `parseFileLine`), result joining, filtering (`filterExcluded`), formatting (`formatResults` → `formatRow`)
- **`main_test.go`** / **`analyzer_test.go`** — Integration and unit tests

### Data pipeline

```
gocyclo.Analyze(paths) → []complexityStat
go tool cover -func    → parseCoverFunc() → []coverageStat
                         detectModulePrefix() → findPrefix() → path prefix
                         joinResults() by (file, line) → []FuncResult
                         sort → filterExcluded → formatResults → summarize
```

### Key design decisions

- `run(opts, stdout, stderr io.Writer)` accepts writers for testability
- `analyze()` extracted from `run()` to keep orchestration complexity low
- Two modes: default (only violations) and `-v` verbose (all functions) — like `go test`
- `-max` controls both display and exit code (unified concept, replaces old `-threshold` + `-over`)
- `-c` is the short form of `-coverprofile`
- `-exclude` uses `filepath.Match` globs, matching both full path and basename (because `*` doesn't match `/` in globs)
- All functions kept under CRAP 5, enforced in CI

## Testing

- **stdlib only** — uses `testing` package with `t.Errorf`/`t.Fatalf` (no testify)
- **Table-driven** — all multi-case functions use `[]struct` pattern
- **Same package** — tests in `package main`, testing unexported functions directly
- **Two test files**: `analyzer_test.go` (domain logic) and `main_test.go` (CLI integration)
- **CI dogfoods gocrap on itself** with `-max 5`

## Notable Details

- `summarize()` uses `-max` value for the "Above N" count; defaults to 30 when no max is set
- `matchesAny` checks basename too because `filepath.Match("*_test.go", "pkg/foo_test.go")` is false
- Functions with no coverage match get 0% coverage (conservative default)
- Single external dependency: `github.com/fzipp/gocyclo v0.6.0`
- No Makefile by design — all commands are one-liner `go` invocations
- No slog — CLI tool output is the product, not logs; uses fmt.Fprint to stdout/stderr
