package main

import (
	"math"
	"path/filepath"
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

type complexityStat struct {
	FuncName   string
	File       string
	Line       int
	Complexity int
}

type coverageStat struct {
	File     string
	Line     int
	Coverage float64
}

func CRAPScore(complexity int, coveragePct float64) float64 {
	comp := float64(complexity)
	uncov := 1.0 - coveragePct/100.0
	return comp*comp*math.Pow(uncov, 3) + comp
}

func joinResults(complexity []complexityStat, coverage []coverageStat) []FuncResult {
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
		normFile := normalizePath(comp.File)
		var cov coverageStat
		var found bool
		for k, c := range covMap {
			if k.line != comp.Line {
				continue
			}
			normK := normalizePath(k.file)
			if normK == normFile || strings.HasSuffix(normK, "/"+normFile) {
				cov = c
				found = true
				break
			}
		}

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

func summarize(results []FuncResult, max float64) (avgCRAP float64, total, exceeding int) {
	if len(results) == 0 {
		return 0, 0, 0
	}

	total = len(results)
	var sum float64
	for _, r := range results {
		sum += r.CRAP
		if r.CRAP > max {
			exceeding++
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

func countExceeding(results []FuncResult, max float64) int {
	var count int
	for _, r := range results {
		if r.CRAP > max {
			count++
		}
	}
	return count
}

func normalizePath(path string) string {
	return strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
}
