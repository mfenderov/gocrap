package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/fzipp/gocyclo"
)

type options struct {
	coverprofile string
	threshold    float64
	over         float64
	top          int
	noTests      bool
	paths        []string
}

func main() {
	opts := parseFlags()
	os.Exit(run(opts, os.Stdout, os.Stderr))
}

func parseFlags() options {
	coverprofile := flag.String("coverprofile", "", "path to Go coverage profile (from go test -coverprofile)")
	threshold := flag.Float64("threshold", 0, "fail if any function exceeds this CRAP score")
	over := flag.Float64("over", 0, "only show functions with CRAP score above this value")
	top := flag.Int("top", 0, "show only the top N worst functions")
	noTests := flag.Bool("no-tests", false, "exclude test files (*_test.go)")
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for i, p := range paths {
		paths[i] = strings.TrimSuffix(p, "/...")
		if paths[i] == "" {
			paths[i] = "."
		}
	}

	return options{
		coverprofile: *coverprofile,
		threshold:    *threshold,
		over:         *over,
		top:          *top,
		noTests:      *noTests,
		paths:        paths,
	}
}

func run(opts options, stdout, stderr io.Writer) int {
	if opts.coverprofile == "" {
		fmt.Fprintln(stderr, "error: -coverprofile is required")
		fmt.Fprintln(stderr, "usage: gocrap -coverprofile coverage.out ./...")
		return 2
	}

	compStats := analyzeComplexity(opts.paths)

	covStats, err := analyzeCoverage(opts.coverprofile)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	modulePrefix := detectModulePrefix(covStats, compStats)
	results := joinResults(compStats, covStats, modulePrefix)

	sort.Slice(results, func(i, j int) bool {
		return results[i].CRAP > results[j].CRAP
	})

	results = processResults(results, opts.noTests, opts.over, opts.top)

	fmt.Fprint(stdout, formatResults(results))

	avg, total, crappy := summarize(results)
	if total > 0 {
		fmt.Fprintf(stdout, "\nAverage CRAP: %.1f | Functions: %d | Above 30: %d\n", avg, total, crappy)
	}

	if opts.threshold > 0 {
		exceeding := countExceeding(results, opts.threshold)
		if exceeding > 0 {
			fmt.Fprintf(stderr, "\nFAIL: %d function(s) exceed CRAP threshold %.0f\n", exceeding, opts.threshold)
			return 1
		}
	}

	return 0
}

func analyzeComplexity(paths []string) []complexityStat {
	stats := gocyclo.Analyze(paths, nil)
	result := make([]complexityStat, len(stats))
	for i, s := range stats {
		result[i] = complexityStat{
			FuncName:   s.FuncName,
			File:       s.Pos.Filename,
			Line:       s.Pos.Line,
			Complexity: s.Complexity,
		}
	}
	return result
}

func analyzeCoverage(coverprofile string) ([]coverageStat, error) {
	output, err := exec.Command("go", "tool", "cover", "-func", coverprofile).Output()
	if err != nil {
		return nil, fmt.Errorf("running go tool cover -func: %w", err)
	}
	return parseCoverFunc(string(output))
}

func detectModulePrefix(covStats []coverageStat, compStats []complexityStat) string {
	if len(covStats) == 0 || len(compStats) == 0 {
		return ""
	}

	for _, comp := range compStats {
		baseName := strings.TrimPrefix(comp.File, "./")
		for _, cov := range covStats {
			if strings.HasSuffix(cov.File, baseName) {
				return strings.TrimSuffix(cov.File, baseName)
			}
		}
	}

	return ""
}
