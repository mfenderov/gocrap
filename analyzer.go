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
	byLine := indexByLine(coverage)

	var results []FuncResult
	for _, comp := range complexity {
		cov, found := lookupCov(byLine, comp.File, comp.Line)
		var coveragePct float64
		if found {
			coveragePct = cov.Coverage
		}
		results = append(results, FuncResult{
			FuncName:   comp.FuncName,
			File:       comp.File,
			Line:       comp.Line,
			Complexity: comp.Complexity,
			Coverage:   coveragePct,
			CRAP:       CRAPScore(comp.Complexity, coveragePct),
		})
	}
	return results
}

// indexByLine groups coverage stats by declaration line, keyed by file
// within each line bucket.
func indexByLine(coverage []coverageStat) map[int]map[string]coverageStat {
	byLine := make(map[int]map[string]coverageStat, len(coverage))
	for _, c := range coverage {
		if byLine[c.Line] == nil {
			byLine[c.Line] = map[string]coverageStat{}
		}
		byLine[c.Line][c.File] = c
	}
	return byLine
}

// lookupCov finds coverage for (file, line) via the line index; ambiguous
// file matches resolve deterministically via bestPathMatch.
func lookupCov(byLine map[int]map[string]coverageStat, file string, line int) (coverageStat, bool) {
	stats := byLine[line]
	if len(stats) == 0 {
		return coverageStat{}, false
	}
	paths := make([]string, 0, len(stats))
	for p := range stats {
		paths = append(paths, p)
	}
	best, found := bestPathMatch(paths, file)
	if !found {
		return coverageStat{}, false
	}
	return stats[best], true
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

// effectiveMax returns the summary threshold: max, or the conventional
// default of 30 when no max is set.
func effectiveMax(max float64) float64 {
	if max <= 0 {
		return 30
	}
	return max
}

func normalizePath(path string) string {
	return strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
}

// bestPathMatch picks the best candidate for target among candidates.
// It normalizes candidates and target internally (via normalizePath).
// Exact normalized equality wins; else the longest suffix-matching
// candidate (most specific path); ties broken lexicographically.
// Returns the candidate in its original (non-normalized) form.
func bestPathMatch(candidates []string, target string) (string, bool) {
	normTarget := normalizePath(target)
	best := ""
	bestNorm := ""
	for _, c := range candidates {
		normC := normalizePath(c)
		if normC == normTarget {
			return c, true
		}
		if betterSuffixMatch(normC, c, bestNorm, best, normTarget) {
			best = c
			bestNorm = normC
		}
	}
	return best, best != ""
}

// betterSuffixMatch reports whether c beats best as a suffix match for
// normTarget (already normalized). normC is the normalized form of c and
// bestNorm is the normalized form of best. c must suffix-match normTarget,
// and wins on greater normalized length (more specific path); ties broken
// lexicographically on the original (non-normalized) candidate strings.
func betterSuffixMatch(normC, c, bestNorm, best, normTarget string) bool {
	if !strings.HasSuffix(normC, "/"+normTarget) {
		return false
	}
	return len(normC) > len(bestNorm) ||
		(len(normC) == len(bestNorm) && c < best)
}
