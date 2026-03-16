package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/fzipp/gocyclo"
)

func main() {
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
	// Normalize: gocyclo takes directory paths, not Go ./... patterns
	for i, p := range paths {
		paths[i] = strings.TrimSuffix(p, "/...")
		if paths[i] == "" {
			paths[i] = "."
		}
	}

	if *coverprofile == "" {
		fmt.Fprintln(os.Stderr, "error: -coverprofile is required")
		fmt.Fprintln(os.Stderr, "usage: gocrap -coverprofile coverage.out ./...")
		os.Exit(2)
	}

	// Get cyclomatic complexity via gocyclo library
	stats := gocyclo.Analyze(paths, nil)
	var compStats []complexityStat
	for _, s := range stats {
		compStats = append(compStats, complexityStat{
			FuncName:   s.FuncName,
			File:       s.Pos.Filename,
			Line:       s.Pos.Line,
			Complexity: s.Complexity,
		})
	}

	// Get per-function coverage via go tool cover -func
	coverOutput, err := exec.Command("go", "tool", "cover", "-func", *coverprofile).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running go tool cover -func: %v\n", err)
		os.Exit(1)
	}

	covStats, err := parseCoverFunc(string(coverOutput))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing coverage: %v\n", err)
		os.Exit(1)
	}

	// Detect module prefix from coverage output
	modulePrefix := detectModulePrefix(covStats, compStats)

	// Join and compute CRAP scores
	results := joinResults(compStats, covStats, modulePrefix)

	// Sort by CRAP score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].CRAP > results[j].CRAP
	})

	// Filter test files
	if *noTests {
		var filtered []FuncResult
		for _, r := range results {
			if !strings.HasSuffix(r.File, "_test.go") {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	// Filter by CRAP score
	if *over > 0 {
		var filtered []FuncResult
		for _, r := range results {
			if r.CRAP > *over {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	if *top > 0 && len(results) > *top {
		results = results[:*top]
	}

	// Output
	fmt.Print(formatResults(results))

	// Summary
	var totalCRAP float64
	var crappy int
	for _, r := range results {
		totalCRAP += r.CRAP
		if r.IsCrappy(30) {
			crappy++
		}
	}
	if len(results) > 0 {
		fmt.Printf("\nAverage CRAP: %.1f | Functions: %d | Above 30: %d\n",
			totalCRAP/float64(len(results)), len(results), crappy)
	}

	// Threshold check
	if *threshold > 0 {
		for _, r := range results {
			if r.IsCrappy(*threshold) {
				fmt.Fprintf(os.Stderr, "\nFAIL: %d function(s) exceed CRAP threshold %.0f\n", crappy, *threshold)
				os.Exit(1)
			}
		}
	}
}

func detectModulePrefix(covStats []coverageStat, compStats []complexityStat) string {
	if len(covStats) == 0 || len(compStats) == 0 {
		return ""
	}

	// Coverage files look like: "module/pkg/file.go"
	// Complexity files look like: "pkg/file.go" (relative)
	// Find the prefix by matching a known file
	for _, comp := range compStats {
		baseName := comp.File
		// Strip leading ./ or absolute path components
		baseName = strings.TrimPrefix(baseName, "./")

		for _, cov := range covStats {
			if strings.HasSuffix(cov.File, baseName) {
				prefix := strings.TrimSuffix(cov.File, baseName)
				return prefix
			}
		}
	}

	return ""
}
