package main

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

type FuncResult struct {
	FuncName   string
	File       string
	Line       int
	Complexity int
	Coverage   float64
	CRAP       float64
}

func (f FuncResult) IsCrappy(threshold float64) bool {
	return f.CRAP >= threshold
}

type complexityStat struct {
	FuncName   string
	File       string
	Line       int
	Complexity int
}

type coverageStat struct {
	File     string
	Line     int
	FuncName string
	Coverage float64
}

func CRAPScore(complexity int, coveragePct float64) float64 {
	comp := float64(complexity)
	uncov := 1.0 - coveragePct/100.0
	return comp*comp*math.Pow(uncov, 3) + comp
}

func parseCoverFunc(output string) ([]coverageStat, error) {
	var results []coverageStat
	for _, line := range strings.Split(output, "\n") {
		if stat, ok := parseCoverLine(line); ok {
			results = append(results, stat)
		}
	}
	return results, nil
}

func parseCoverLine(line string) (coverageStat, bool) {
	line = strings.TrimSpace(line)
	if skipCoverLine(line) {
		return coverageStat{}, false
	}

	fields := strings.Fields(line)
	if len(fields) < 3 {
		return coverageStat{}, false
	}

	file, lineNum, ok := parseFileLine(fields[0])
	if !ok {
		return coverageStat{}, false
	}

	cov, ok := parseCoverage(fields[len(fields)-1])
	if !ok {
		return coverageStat{}, false
	}

	return coverageStat{File: file, Line: lineNum, FuncName: fields[1], Coverage: cov}, true
}

func skipCoverLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "total:")
}

func parseCoverage(field string) (float64, bool) {
	covStr := strings.TrimSuffix(field, "%")
	cov, err := strconv.ParseFloat(covStr, 64)
	return cov, err == nil
}

func parseFileLine(field string) (string, int, bool) {
	field = strings.TrimSuffix(field, ":")
	lastColon := strings.LastIndex(field, ":")
	if lastColon == -1 {
		return "", 0, false
	}
	lineNum, err := strconv.Atoi(field[lastColon+1:])
	if err != nil {
		return "", 0, false
	}
	return field[:lastColon], lineNum, true
}

func joinResults(complexity []complexityStat, coverage []coverageStat, modulePrefix string) []FuncResult {
	type key struct {
		file string
		line int
	}

	covMap := make(map[key]coverageStat, len(coverage))
	for _, c := range coverage {
		covMap[key{file: c.File, line: c.Line}] = c
	}

	var results []FuncResult
	for _, comp := range complexity {
		// Normalize: gocyclo uses relative paths, cover uses module-prefixed paths
		covFile := modulePrefix + comp.File
		cov, found := covMap[key{file: covFile, line: comp.Line}]

		var coveragePct float64
		if found {
			coveragePct = cov.Coverage
		}

		crapScore := CRAPScore(comp.Complexity, coveragePct)

		results = append(results, FuncResult{
			FuncName:   comp.FuncName,
			File:       comp.File,
			Line:       comp.Line,
			Complexity: comp.Complexity,
			Coverage:   coveragePct,
			CRAP:       crapScore,
		})
	}

	return results
}

func processResults(results []FuncResult, exclude []string, over float64, top int) []FuncResult {
	results = filterExcluded(results, exclude)
	results = filterOver(results, over)
	if top > 0 && len(results) > top {
		results = results[:top]
	}
	return results
}

func filterExcluded(results []FuncResult, exclude []string) []FuncResult {
	if len(exclude) == 0 {
		return results
	}
	var filtered []FuncResult
	for _, r := range results {
		if !matchesAny(r.File, exclude) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func filterOver(results []FuncResult, over float64) []FuncResult {
	if over <= 0 {
		return results
	}
	var filtered []FuncResult
	for _, r := range results {
		if r.CRAP > over {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func summarize(results []FuncResult) (avgCRAP float64, total, crappy int) {
	if len(results) == 0 {
		return 0, 0, 0
	}

	total = len(results)
	var sum float64
	for _, r := range results {
		sum += r.CRAP
		if r.IsCrappy(30) {
			crappy++
		}
	}
	avgCRAP = sum / float64(total)
	return
}

func matchesAny(file string, patterns []string) bool {
	base := filepath.Base(file)
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, file); matched {
			return true
		}
		if matched, _ := filepath.Match(p, base); matched {
			return true
		}
	}
	return false
}

func countExceeding(results []FuncResult, threshold float64) int {
	var count int
	for _, r := range results {
		if r.CRAP > threshold {
			count++
		}
	}
	return count
}

func formatResults(results []FuncResult, threshold float64) string {
	if len(results) == 0 {
		return ""
	}
	var b strings.Builder
	if threshold > 0 {
		fmt.Fprintf(&b, "%-6s %-8s %-12s %-10s %-40s %s\n", "", "CRAP", "Complexity", "Coverage", "Function", "Location")
	} else {
		fmt.Fprintf(&b, "%-8s %-12s %-10s %-40s %s\n", "CRAP", "Complexity", "Coverage", "Function", "Location")
	}
	for _, r := range results {
		formatRow(&b, r, threshold)
	}
	return b.String()
}

func formatRow(b *strings.Builder, r FuncResult, threshold float64) {
	cov := fmt.Sprintf("%.1f%%", r.Coverage)
	if threshold > 0 {
		status := "ok"
		if r.CRAP > threshold {
			status = "FAIL"
		}
		fmt.Fprintf(b, "%-6s %-8.1f %-12d %-10s %-40s %s:%d\n", status, r.CRAP, r.Complexity, cov, r.FuncName, r.File, r.Line)
	} else {
		fmt.Fprintf(b, "%-8.1f %-12d %-10s %-40s %s:%d\n", r.CRAP, r.Complexity, cov, r.FuncName, r.File, r.Line)
	}
}
