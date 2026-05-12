package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type coverSegment struct {
	File       string
	StartLine  int
	StartCol   int
	EndLine    int
	EndCol     int
	Statements int
	Count      int
}

func readProfile(path string) (map[string][]coverSegment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseCoverProfile(f)
}

func parseCoverProfile(r io.Reader) (map[string][]coverSegment, error) {
	out := map[string][]coverSegment{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if skipProfileLine(line) {
			continue
		}
		segment, err := parseCoverSegment(line)
		if err != nil {
			return nil, err
		}
		out[segment.File] = append(out[segment.File], segment)
	}
	return out, scanner.Err()
}

func skipProfileLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "mode:")
}

func parseCoverSegment(line string) (coverSegment, error) {
	file, rest, err := splitFilePrefix(line)
	if err != nil {
		return coverSegment{}, err
	}
	startL, startC, endL, endC, rest, err := parseRange(rest)
	if err != nil {
		return coverSegment{}, err
	}
	statements, count, err := parseCounts(strings.Fields(rest))
	if err != nil {
		return coverSegment{}, err
	}
	return coverSegment{
		File: file, StartLine: startL, StartCol: startC,
		EndLine: endL, EndCol: endC, Statements: statements, Count: count,
	}, nil
}

func splitFilePrefix(line string) (file, rest string, err error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return "", "", fmt.Errorf("invalid coverage line: %q", line)
	}
	file, after, found := strings.Cut(fields[0], ":")
	if !found {
		return "", "", fmt.Errorf("missing file separator in %q", line)
	}
	return file, after + " " + fields[1] + " " + fields[2], nil
}

func parseRange(rest string) (startLine, startCol, endLine, endCol int, rest2 string, err error) {
	rangePart, after, found := strings.Cut(rest, " ")
	if !found {
		rangePart = rest
	}
	startPart, endPart, found := strings.Cut(rangePart, ",")
	if !found {
		return 0, 0, 0, 0, "", fmt.Errorf("missing range separator in %q", rest)
	}
	startLine, startCol, endLine, endCol, err = parsePosRange(startPart, endPart)
	if err != nil {
		return 0, 0, 0, 0, "", err
	}
	return startLine, startCol, endLine, endCol, after, nil
}

func parsePosRange(startPart, endPart string) (int, int, int, int, error) {
	startLine, startCol, err := parsePos(startPart)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	endLine, endCol, err := parsePos(endPart)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return startLine, startCol, endLine, endCol, nil
}

func parseCounts(fields []string) (int, int, error) {
	statements, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid statement count: %w", err)
	}
	count, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid execution count: %w", err)
	}
	return statements, count, nil
}

func parsePos(value string) (int, int, error) {
	linePart, colPart, found := strings.Cut(value, ".")
	if !found {
		return 0, 0, fmt.Errorf("invalid position %q", value)
	}
	line, err := strconv.Atoi(linePart)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid line: %w", err)
	}
	col, err := strconv.Atoi(colPart)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid column: %w", err)
	}
	return line, col, nil
}

func computeCoverage(profile map[string][]coverSegment, functions []functionRange) []coverageStat {
	var results []coverageStat
	for _, fn := range functions {
		cov := functionCoverage(profile, fn)
		results = append(results, coverageStat{File: fn.File, Line: fn.StartLine, Coverage: cov})
	}
	return results
}

func functionCoverage(profile map[string][]coverSegment, fn functionRange) float64 {
	segments := segmentsForFile(profile, fn.File)
	if len(segments) == 0 {
		return 0
	}
	total, covered := countStmts(segments, fn.StartLine, fn.EndLine)
	if total == 0 {
		return 0
	}
	return 100.0 * float64(covered) / float64(total)
}

func countStmts(segments []coverSegment, startLine, endLine int) (total, covered int) {
	for _, seg := range segments {
		if seg.EndLine < startLine || seg.StartLine > endLine {
			continue
		}
		total += seg.Statements
		if seg.Count > 0 {
			covered += seg.Statements
		}
	}
	return
}

func segmentsForFile(profile map[string][]coverSegment, file string) []coverSegment {
	normalized := normalizePath(file)
	if segs := profile[normalized]; len(segs) > 0 {
		return segs
	}
	return findSegmentsBySuffix(profile, normalized)
}

func findSegmentsBySuffix(profile map[string][]coverSegment, normalized string) []coverSegment {
	for candidate, segments := range profile {
		if strings.HasSuffix(candidate, "/"+normalized) || candidate == normalized {
			return segments
		}
	}
	return nil
}
