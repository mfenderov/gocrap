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
		if line == "" || strings.HasPrefix(line, "mode:") {
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

func parseCoverSegment(line string) (coverSegment, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return coverSegment{}, fmt.Errorf("invalid coverage line: %q", line)
	}

	file, rest, found := strings.Cut(fields[0], ":")
	if !found {
		return coverSegment{}, fmt.Errorf("missing file separator in %q", line)
	}

	rangePart, stmtPart, found := strings.Cut(rest, " ")
	if !found {
		rangePart = rest
		stmtPart = fields[1]
	}

	startPart, endPart, found := strings.Cut(rangePart, ",")
	if !found {
		return coverSegment{}, fmt.Errorf("missing range separator in %q", line)
	}

	startLine, startCol, err := parsePos(startPart)
	if err != nil {
		return coverSegment{}, err
	}
	endLine, endCol, err := parsePos(endPart)
	if err != nil {
		return coverSegment{}, err
	}

	statements, err := strconv.Atoi(stmtPart)
	if err != nil {
		return coverSegment{}, fmt.Errorf("invalid statement count: %w", err)
	}

	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return coverSegment{}, fmt.Errorf("invalid execution count: %w", err)
	}

	return coverSegment{
		File: file, StartLine: startLine, StartCol: startCol,
		EndLine: endLine, EndCol: endCol, Statements: statements, Count: count,
	}, nil
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
		segments := segmentsForFile(profile, fn.File)
		if len(segments) == 0 {
			results = append(results, coverageStat{File: fn.File, Line: fn.StartLine, Coverage: 0})
			continue
		}

		total := 0
		covered := 0
		for _, seg := range segments {
			if seg.EndLine < fn.StartLine || seg.StartLine > fn.EndLine {
				continue
			}
			total += seg.Statements
			if seg.Count > 0 {
				covered += seg.Statements
			}
		}

		var coverage float64
		if total > 0 {
			coverage = 100.0 * float64(covered) / float64(total)
		}

		results = append(results, coverageStat{File: fn.File, Line: fn.StartLine, Coverage: coverage})
	}
	return results
}

func segmentsForFile(profile map[string][]coverSegment, file string) []coverSegment {
	normalized := normalizePath(file)
	if segments := profile[normalized]; len(segments) > 0 {
		return segments
	}
	for candidate, segments := range profile {
		if strings.HasSuffix(candidate, "/"+normalized) || candidate == normalized {
			return segments
		}
	}
	return nil
}
