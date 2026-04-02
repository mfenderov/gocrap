# gocrap

Compute [CRAP scores](https://testing.googleblog.com/2011/02/this-code-is-crap.html) (Change Risk Anti-Patterns) for Go functions. Combines cyclomatic complexity with test coverage to identify risky, under-tested code.

## Install

```bash
go install github.com/mfenderov/gocrap@latest
```

## Usage

```bash
# Generate a coverage profile first
go test -coverprofile=coverage.out ./...

# Show only violations (default) — like go test
gocrap -c coverage.out -max 12 ./...

# Show all functions — like go test -v
gocrap -c coverage.out -max 12 -v ./...

# Explore without a threshold (shows everything, no pass/fail)
gocrap -c coverage.out ./...
```

### Flags

| Flag | Description |
|------|-------------|
| `-c` / `-coverprofile` | Path to coverage profile (required) |
| `-max N` | Max allowed CRAP score — show violations and exit 1 if any exceed |
| `-v` | Show all functions, not just violations |
| `-exclude PATTERN` | Exclude files matching glob pattern (can be repeated) |

### Example output

Default (`-max 12`):
```
       CRAP     Complexity   Coverage   Function                                 Location
FAIL   84.1     15           32.5%      (*AgentHandler).Handle                   pkg/ai/agent.go:122

Average CRAP: 29.4 | Functions: 3 | Above 12: 1

FAIL: 1 function(s) exceed max CRAP score 12
```

Verbose (`-max 12 -v`):
```
       CRAP     Complexity   Coverage   Function                                 Location
FAIL   84.1     15           32.5%      (*AgentHandler).Handle                   pkg/ai/agent.go:122
ok     3.0      3            100.0%     NewGroqClient                            pkg/ai/client_groq.go:25
ok     1.0      1            100.0%     Chat                                     pkg/ai/client_groq.go:42

Average CRAP: 29.4 | Functions: 3 | Above 12: 1

FAIL: 1 function(s) exceed max CRAP score 12
```

## CRAP Formula

```
CRAP(m) = complexity(m)² × (1 − coverage(m)/100)³ + complexity(m)
```

A function with complexity 1 and 100% coverage scores 1 (perfect). A function with complexity 10 and 0% coverage scores 110 (terrible). The standard "crappy" threshold is 30.

> **Tip**: TDD is the best anti-CRAP technique — writing tests first naturally keeps complexity low and coverage high.
