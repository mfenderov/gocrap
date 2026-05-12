package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func formatResults(results []FuncResult, max float64, verbose bool) string {
	if len(results) == 0 {
		return ""
	}
	var b strings.Builder
	for _, r := range results {
		formatRow(&b, r, max, verbose)
	}
	if b.Len() == 0 {
		return ""
	}
	var out strings.Builder
	writeHeader(&out, max)
	out.WriteString(b.String())
	return out.String()
}

func writeHeader(b *strings.Builder, max float64) {
	if max > 0 {
		fmt.Fprintf(b, "%-6s %-8s %-12s %-10s %-40s %s\n", "", "CRAP", "Complexity", "Coverage", "Function", "Location")
	} else {
		fmt.Fprintf(b, "%-8s %-12s %-10s %-40s %s\n", "CRAP", "Complexity", "Coverage", "Function", "Location")
	}
}

func formatRow(b *strings.Builder, r FuncResult, max float64, verbose bool) {
	cov := fmt.Sprintf("%.1f%%", r.Coverage)
	if max <= 0 {
		fmt.Fprintf(b, "%-8.1f %-12d %-10s %-40s %s:%d\n", r.CRAP, r.Complexity, cov, r.FuncName, r.File, r.Line)
		return
	}
	status := "ok"
	if r.CRAP > max {
		status = "FAIL"
	}
	if !verbose && status == "ok" {
		return
	}
	fmt.Fprintf(b, "%-6s %-8.1f %-12d %-10s %-40s %s:%d\n", status, r.CRAP, r.Complexity, cov, r.FuncName, r.File, r.Line)
}

type jsonOutput struct {
	Results []jsonResult `json:"results"`
	Summary jsonSummary  `json:"summary"`
}

type jsonResult struct {
	Function   string  `json:"function"`
	File       string  `json:"file"`
	Line       int     `json:"line"`
	Complexity int     `json:"complexity"`
	Coverage   float64 `json:"coverage"`
	CRAP       float64 `json:"crap"`
	Fails      bool    `json:"fails"`
}

type jsonSummary struct {
	AverageCRAP    float64 `json:"average_crap"`
	TotalFunctions int     `json:"total_functions"`
	AboveThreshold int     `json:"above_threshold"`
	Threshold      float64 `json:"threshold"`
}

func formatResultsJSON(results []FuncResult, max float64) string {
	out := jsonOutput{
		Results: make([]jsonResult, len(results)),
	}
	for i, r := range results {
		out.Results[i] = jsonResult{
			Function:   r.FuncName,
			File:       r.File,
			Line:       r.Line,
			Complexity: r.Complexity,
			Coverage:   r.Coverage,
			CRAP:       r.CRAP,
			Fails:      isFailing(r.CRAP, max),
		}
	}

	threshold := max
	if threshold <= 0 {
		threshold = 30
	}
	avg, total, exceeding := summarize(results, threshold)
	out.Summary = jsonSummary{
		AverageCRAP:    avg,
		TotalFunctions: total,
		AboveThreshold: exceeding,
		Threshold:      threshold,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

func isFailing(crap, max float64) bool {
	return max > 0 && crap > max
}
