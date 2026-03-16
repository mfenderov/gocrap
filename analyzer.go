package main

import (
	"fmt"
	"math"
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
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total:") {
			continue
		}

		// Format: file:line:\tfuncName\tcoverage%
		colonIdx := strings.LastIndex(line, ":")
		if colonIdx == -1 {
			continue
		}

		// Split on tabs to get fields
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Parse file:line from first field
		fileLine := fields[0]
		// Remove trailing colon
		fileLine = strings.TrimSuffix(fileLine, ":")

		lastColon := strings.LastIndex(fileLine, ":")
		if lastColon == -1 {
			continue
		}

		file := fileLine[:lastColon]
		lineNum, err := strconv.Atoi(fileLine[lastColon+1:])
		if err != nil {
			continue
		}

		funcName := fields[1]

		// Parse coverage percentage (last field, remove %)
		covStr := strings.TrimSuffix(fields[len(fields)-1], "%")
		cov, err := strconv.ParseFloat(covStr, 64)
		if err != nil {
			continue
		}

		results = append(results, coverageStat{
			File:     file,
			Line:     lineNum,
			FuncName: funcName,
			Coverage: cov,
		})
	}

	return results, nil
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

func processResults(results []FuncResult, noTests bool, over float64, top int) []FuncResult {
	if noTests {
		var filtered []FuncResult
		for _, r := range results {
			if !strings.HasSuffix(r.File, "_test.go") && !strings.HasSuffix(r.File, "_mock.go") {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	if over > 0 {
		var filtered []FuncResult
		for _, r := range results {
			if r.CRAP > over {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	if top > 0 && len(results) > top {
		results = results[:top]
	}

	return results
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

func formatResults(results []FuncResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-8s %-12s %-10s %-40s %s\n", "CRAP", "Complexity", "Coverage", "Function", "Location"))

	for _, r := range results {
		b.WriteString(fmt.Sprintf("%-8.1f %-12d %-10s %-40s %s:%d\n",
			r.CRAP,
			r.Complexity,
			fmt.Sprintf("%.1f%%", r.Coverage),
			r.FuncName,
			r.File,
			r.Line,
		))
	}

	return b.String()
}
