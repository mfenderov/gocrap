package main

import (
	"flag"
	"fmt"
	"io"
	"os"
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
	json         bool
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
	json := flag.Bool("json", false, "output results as JSON")
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
		json:         *json,
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

	printReport(stdout, results, opts.max, opts.verbose, opts.json)
	return checkMax(stderr, results, opts.max)
}

func analyze(opts options) ([]FuncResult, error) {
	compStats := analyzeComplexity(opts.paths)

	profile, err := readProfile(opts.coverprofile)
	if err != nil {
		return nil, err
	}

	sourceFiles, err := findSourceFiles(opts.paths)
	if err != nil {
		return nil, err
	}

	var functions []functionRange
	for _, f := range sourceFiles {
		fns, err := extractFunctions(f)
		if err != nil {
			return nil, err
		}
		functions = append(functions, fns...)
	}

	covStats := computeCoverage(profile, functions)
	results := joinResults(compStats, covStats)

	sort.Slice(results, func(i, j int) bool {
		return results[i].CRAP > results[j].CRAP
	})

	return filterExcluded(results, opts.exclude), nil
}

func printReport(w io.Writer, results []FuncResult, max float64, verbose, json bool) {
	if json {
		fmt.Fprint(w, formatResultsJSON(results, max))
		return
	}
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
