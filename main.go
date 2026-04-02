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
	max          float64
	verbose      bool
	exclude      []string
	paths        []string
}

func main() {
	opts := parseFlags()
	os.Exit(run(opts, os.Stdout, os.Stderr))
}

func parseFlags() options {
	coverprofile := flag.String("c", "", "path to Go coverage profile (from go test -coverprofile)")
	flag.StringVar(coverprofile, "coverprofile", "", "path to Go coverage profile (from go test -coverprofile)")
	max := flag.Float64("max", 0, "max allowed CRAP score — show violations and exit 1 if any exceed")
	verbose := flag.Bool("v", false, "show all functions, not just violations")
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
		max:          *max,
		verbose:      *verbose,
		exclude:      exclude,
		paths:        paths,
	}
}

func run(opts options, stdout, stderr io.Writer) int {
	if opts.coverprofile == "" {
		fmt.Fprintln(stderr, "error: -c is required")
		fmt.Fprintln(stderr, "usage: gocrap -c coverage.out -max 12 ./...")
		return 2
	}

	results, err := analyze(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	printReport(stdout, results, opts.max, opts.verbose)
	return checkMax(stderr, results, opts.max)
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

	return filterExcluded(results, opts.exclude), nil
}

func printReport(w io.Writer, results []FuncResult, max float64, verbose bool) {
	if len(results) == 0 {
		fmt.Fprintln(w, "No CRAP found.")
		return
	}
	fmt.Fprint(w, formatResults(results, max, verbose))
	threshold := max
	if threshold <= 0 {
		threshold = 30
	}
	avg, total, exceeding := summarize(results, threshold)
	fmt.Fprintf(w, "\nAverage CRAP: %.1f | Functions: %d | Above %.0f: %d\n", avg, total, threshold, exceeding)
}

func checkMax(w io.Writer, results []FuncResult, max float64) int {
	if max <= 0 {
		return 0
	}
	exceeding := countExceeding(results, max)
	if exceeding > 0 {
		fmt.Fprintf(w, "\nFAIL: %d function(s) exceed max CRAP score %.0f\n", exceeding, max)
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
