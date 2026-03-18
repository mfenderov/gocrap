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

# Run gocrap
gocrap -coverprofile coverage.out ./...
```

### Flags

| Flag | Description |
|------|-------------|
| `-coverprofile` | Path to coverage profile (required) |
| `-threshold N` | Show pass/fail per function, exit 1 if any exceed N (like `go test`) |
| `-over N` | Only show functions above score N |
| `-top N` | Show only the N worst functions |
| `-exclude PATTERN` | Exclude files matching glob pattern (can be repeated) |

### Example output

Without threshold:
```
CRAP     Complexity   Coverage   Function                                 Location
84.1     15           32.5%      (*AgentHandler).Handle                   pkg/ai/agent.go:122
3.0      3            100.0%     NewGroqClient                            pkg/ai/client_groq.go:25
1.0      1            100.0%     Chat                                     pkg/ai/client_groq.go:42

Average CRAP: 29.4 | Functions: 3 | Above 30: 1
```

With `-threshold 12` (like `go test`):
```
       CRAP     Complexity   Coverage   Function                                 Location
FAIL   84.1     15           32.5%      (*AgentHandler).Handle                   pkg/ai/agent.go:122
ok     3.0      3            100.0%     NewGroqClient                            pkg/ai/client_groq.go:25
ok     1.0      1            100.0%     Chat                                     pkg/ai/client_groq.go:42

Average CRAP: 29.4 | Functions: 3 | Above 30: 1

FAIL: 1 function(s) exceed CRAP threshold 12
```

## CRAP Formula

```
CRAP(m) = complexity(m)² × (1 − coverage(m)/100)³ + complexity(m)
```

A function with complexity 1 and 100% coverage scores 1 (perfect). A function with complexity 10 and 0% coverage scores 110 (terrible). The standard "crappy" threshold is 30.

> **Tip**: TDD is the best anti-CRAP technique — writing tests first naturally keeps complexity low and coverage high.
