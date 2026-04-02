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

type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

type options struct {
	coverprofile string
	threshold    float64
	over         float64
	top          int
	exclude      []string
	paths        []string
}

func main() {
	opts := parseFlags()
	os.Exit(run(opts, os.Stdout, os.Stderr))
}

func parseFlags() options {
	coverprofile := flag.String("coverprofile", "", "path to Go coverage profile (from go test -coverprofile)")
	threshold := flag.Float64("threshold", 0, "show pass/fail per function and exit 1 if any exceed this CRAP score")
	over := flag.Float64("over", 0, "only show functions with CRAP score above this value")
	top := flag.Int("top", 0, "show only the top N worst functions")
	var exclude stringSlice
	flag.Var(&exclude, "exclude", "exclude files matching glob pattern (can be repeated)")
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
		exclude:      exclude,
		paths:        paths,
	}
}

func run(opts options, stdout, stderr io.Writer) int {
	if opts.coverprofile == "" {
		fmt.Fprintln(stderr, "error: -coverprofile is required")
		fmt.Fprintln(stderr, "usage: gocrap -coverprofile coverage.out ./...")
		return 2
	}

	results, err := analyze(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	printReport(stdout, results, opts.threshold)
	return checkThreshold(stderr, results, opts.threshold)
}

func analyze(opts options) ([]FuncResult, error) {
	compStats := analyzeComplexity(opts.paths)

	covStats, err := analyzeCoverage(opts.coverprofile)
	if err != nil {
		return nil, err
	}

	modulePrefix := detectModulePrefix(covStats, compStats)
	results := joinResults(compStats, covStats, modulePrefix)

	sort.Slice(results, func(i, j int) bool {
		return results[i].CRAP > results[j].CRAP
	})

	return processResults(results, opts.exclude, opts.over, opts.top), nil
}

func printReport(w io.Writer, results []FuncResult, threshold float64) {
	if len(results) == 0 {
		fmt.Fprintln(w, "No CRAP found.")
		return
	}
	fmt.Fprint(w, formatResults(results, threshold))
	avg, total, crappy := summarize(results)
	fmt.Fprintf(w, "\nAverage CRAP: %.1f | Functions: %d | Above 30: %d\n", avg, total, crappy)
}

func checkThreshold(w io.Writer, results []FuncResult, threshold float64) int {
	if threshold <= 0 {
		return 0
	}
	exceeding := countExceeding(results, threshold)
	if exceeding > 0 {
		fmt.Fprintf(w, "\nFAIL: %d function(s) exceed CRAP threshold %.0f\n", exceeding, threshold)
		return 1
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
		if prefix, ok := findPrefix(comp.File, covStats); ok {
			return prefix
		}
	}
	return ""
}

func findPrefix(compFile string, covStats []coverageStat) (string, bool) {
	baseName := strings.TrimPrefix(compFile, "./")
	for _, cov := range covStats {
		if strings.HasSuffix(cov.File, baseName) {
			return strings.TrimSuffix(cov.File, baseName), true
		}
	}
	return "", false
}
