package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCRAPScore(t *testing.T) {
	tests := []struct {
		name       string
		complexity int
		coverage   float64
		wantCRAP   float64
	}{
		{"perfect: complexity 1, 100% coverage", 1, 100.0, 1.0},
		{"simple uncovered: complexity 1, 0% coverage", 1, 0.0, 2.0},
		{"complex well-tested: complexity 10, 100% coverage", 10, 100.0, 10.0},
		{"complex untested: complexity 10, 0% coverage", 10, 0.0, 110.0},
		{"moderate: complexity 5, 50% coverage", 5, 50.0, 8.125},
		{"threshold boundary: complexity 15, 40% coverage", 15, 40.0, 63.6},
		{"zero complexity", 0, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CRAPScore(tt.complexity, tt.coverage)
			if math.Abs(got-tt.wantCRAP) > 0.01 {
				t.Errorf("CRAPScore(%d, %.1f) = %.3f, want %.3f", tt.complexity, tt.coverage, got, tt.wantCRAP)
			}
		})
	}
}

func TestFilterExcluded_Glob(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Handle", File: "agent.go", CRAP: 50},
		{FuncName: "TestHandle", File: "agent_test.go", CRAP: 40},
		{FuncName: "MockChat", File: "client_mock.go", CRAP: 10},
		{FuncName: "Chat", File: "client.go", CRAP: 5},
	}

	got := filterExcluded(results, []string{"*_test.go", "*_mock.go"})
	if len(got) != 2 {
		t.Fatalf("filterExcluded(test+mock) returned %d results, want 2", len(got))
	}
	if got[0].File != "agent.go" || got[1].File != "client.go" {
		t.Errorf("unexpected files: %q, %q", got[0].File, got[1].File)
	}
}

func TestFilterExcluded_Directory(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Handle", File: "pkg/handler.go", CRAP: 50},
		{FuncName: "Gen", File: "generated/types.go", CRAP: 10},
	}

	got := filterExcluded(results, []string{"generated/*"})
	if len(got) != 1 {
		t.Fatalf("filterExcluded(dir) returned %d results, want 1", len(got))
	}
	if got[0].FuncName != "Handle" {
		t.Errorf("got %q, want Handle", got[0].FuncName)
	}
}

func TestFilterExcluded_NoPatterns(t *testing.T) {
	results := []FuncResult{{FuncName: "A"}, {FuncName: "B"}}
	got := filterExcluded(results, nil)
	if len(got) != 2 {
		t.Errorf("filterExcluded(nil) returned %d, want 2", len(got))
	}
}

func TestSummarize(t *testing.T) {
	results := []FuncResult{
		{CRAP: 50},
		{CRAP: 10},
		{CRAP: 1},
	}

	avg, total, exceeding := summarize(results, 30)
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if exceeding != 1 {
		t.Errorf("exceeding = %d, want 1", exceeding)
	}
	if math.Abs(avg-20.33) > 0.01 {
		t.Errorf("avg = %.2f, want 20.33", avg)
	}
}

func TestSummarize_CustomMax(t *testing.T) {
	results := []FuncResult{{CRAP: 50}, {CRAP: 10}, {CRAP: 1}}
	_, _, exceeding := summarize(results, 5)
	if exceeding != 2 {
		t.Errorf("exceeding(max=5) = %d, want 2", exceeding)
	}
}

func TestSummarize_Empty(t *testing.T) {
	avg, total, exceeding := summarize(nil, 30)
	if total != 0 || exceeding != 0 || avg != 0 {
		t.Errorf("summarize(nil) = (%.1f, %d, %d), want (0, 0, 0)", avg, total, exceeding)
	}
}

func TestCountExceeding(t *testing.T) {
	results := []FuncResult{
		{FuncName: "A", CRAP: 50},
		{FuncName: "B", CRAP: 35},
		{FuncName: "C", CRAP: 10},
		{FuncName: "D", CRAP: 5},
	}

	tests := []struct {
		name string
		max  float64
		want int
	}{
		{"max 50 - none above", 50, 0},
		{"max 30 - two above", 30, 2},
		{"max 10 - two above", 10, 2},
		{"max 1 - all four above", 1, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countExceeding(results, tt.max); got != tt.want {
				t.Errorf("countExceeding(max=%.0f) = %d, want %d", tt.max, got, tt.want)
			}
		})
	}
}

func TestCountExceeding_Empty(t *testing.T) {
	if got := countExceeding(nil, 30); got != 0 {
		t.Errorf("countExceeding(nil) = %d, want 0", got)
	}
}

func TestFormatResults_Empty(t *testing.T) {
	if output := formatResults(nil, 0, false); output != "" {
		t.Errorf("expected empty string, got %q", output)
	}
}

func TestFormatResults_DefaultOnlyViolations(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Run", File: "main.go", Line: 58, Complexity: 6, Coverage: 0, CRAP: 42.0},
		{FuncName: "Parse", File: "main.go", Line: 29, Complexity: 2, Coverage: 100, CRAP: 2.0},
	}

	output := formatResults(results, 12, false)

	if !strings.Contains(output, "FAIL") {
		t.Error("expected FAIL marker for Run (CRAP 42 > max 12)")
	}
	if strings.Contains(output, "ok") {
		t.Error("expected no ok markers in default mode (only violations shown)")
	}
	if strings.Contains(output, "Parse") {
		t.Error("expected Parse to be hidden in default mode (CRAP 2 < max 12)")
	}
}

func TestFormatResults_VerboseShowsAll(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Run", File: "main.go", Line: 58, Complexity: 6, Coverage: 0, CRAP: 42.0},
		{FuncName: "Parse", File: "main.go", Line: 29, Complexity: 2, Coverage: 100, CRAP: 2.0},
	}

	output := formatResults(results, 12, true)

	if !strings.Contains(output, "FAIL") {
		t.Error("expected FAIL marker in verbose mode")
	}
	if !strings.Contains(output, "ok") {
		t.Error("expected ok markers in verbose mode")
	}
	if !strings.Contains(output, "Parse") {
		t.Error("expected Parse to be shown in verbose mode")
	}
}

func TestFormatResults_NoMax(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Run", File: "main.go", Line: 58, Complexity: 6, Coverage: 0, CRAP: 42.0},
	}

	output := formatResults(results, 0, false)

	if strings.Contains(output, "FAIL") || strings.Contains(output, "ok") {
		t.Error("expected no status markers when max is 0")
	}
}

func TestJoinResults(t *testing.T) {
	complexity := []complexityStat{
		{FuncName: "(*AgentHandler).Handle", File: "pkg/ai/agent.go", Line: 122, Complexity: 15},
		{FuncName: "NewGroqClient", File: "pkg/ai/client_groq.go", Line: 25, Complexity: 3},
		{FuncName: "Chat", File: "pkg/ai/client_groq.go", Line: 42, Complexity: 1},
	}

	coverage := []coverageStat{
		{File: "bambot/pkg/ai/agent.go", Line: 122, Coverage: 32.5},
		{File: "bambot/pkg/ai/client_groq.go", Line: 25, Coverage: 100.0},
		{File: "bambot/pkg/ai/client_groq.go", Line: 42, Coverage: 100.0},
	}

	results := joinResults(complexity, coverage)

	if len(results) != 3 {
		t.Fatalf("joinResults() returned %d results, want 3", len(results))
	}

	if results[0].FuncName != "(*AgentHandler).Handle" {
		t.Errorf("results[0].FuncName = %q, want %q", results[0].FuncName, "(*AgentHandler).Handle")
	}
	if results[0].Complexity != 15 {
		t.Errorf("results[0].Complexity = %d, want 15", results[0].Complexity)
	}
	if math.Abs(results[0].Coverage-32.5) > 0.01 {
		t.Errorf("results[0].Coverage = %.1f, want 32.5", results[0].Coverage)
	}
	if math.Abs(results[1].CRAP-3.0) > 0.01 {
		t.Errorf("results[1].CRAP = %.3f, want 3.0", results[1].CRAP)
	}
	if math.Abs(results[2].CRAP-1.0) > 0.01 {
		t.Errorf("results[2].CRAP = %.3f, want 1.0", results[2].CRAP)
	}
}

func TestFormatResultsJSON(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Run", File: "main.go", Line: 58, Complexity: 6, Coverage: 0, CRAP: 42.0},
		{FuncName: "Parse", File: "main.go", Line: 29, Complexity: 2, Coverage: 100, CRAP: 2.0},
	}

	output := formatResultsJSON(results, 12)

	if !strings.Contains(output, `"function"`) {
		t.Error("expected JSON with function field")
	}
	if !strings.Contains(output, `"crap"`) {
		t.Error("expected JSON with crap field")
	}
	if !strings.Contains(output, `"average_crap"`) {
		t.Error("expected JSON summary with average_crap")
	}
	if !strings.Contains(output, "true") {
		t.Error("expected at least one fails:true for Run (CRAP 42 > max 12)")
	}
}

func TestFormatResultsJSON_NoMax(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Run", File: "main.go", Line: 58, Complexity: 6, Coverage: 0, CRAP: 42.0},
	}

	output := formatResultsJSON(results, 0)

	if strings.Contains(output, "true") {
		t.Error("expected no fails:true when max is 0")
	}
}

func TestFormatResultsJSON_Empty(t *testing.T) {
	output := formatResultsJSON(nil, 12)

	if !strings.Contains(output, `"results"`) {
		t.Error("expected JSON with results array")
	}
	if !strings.Contains(output, `"total_functions": 0`) {
		t.Error("expected zero total_functions for empty input")
	}
}

func TestParseCoverProfile(t *testing.T) {
	input := "mode: set\nbambot/pkg/ai/agent.go:73.2,74.3 1 1\nbambot/pkg/ai/agent.go:73.2,74.3 2 0\nbambot/pkg/ai/client_groq.go:25.2,26.3 1 1\n\n"
	profile, err := parseCoverProfile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseCoverProfile() error = %v", err)
	}
	if len(profile) != 2 {
		t.Fatalf("parseCoverProfile() returned %d files, want 2", len(profile))
	}

	wantAgent := 2
	if got := len(profile["bambot/pkg/ai/agent.go"]); got != wantAgent {
		t.Errorf("profile['bambot/pkg/ai/agent.go'] has %d segments, want %d", got, wantAgent)
	}

	seg := profile["bambot/pkg/ai/agent.go"][0]
	if seg.StartLine != 73 || seg.EndLine != 74 {
		t.Errorf("segment range = (%d,%d), want (73,74)", seg.StartLine, seg.EndLine)
	}
	if seg.Statements != 1 || seg.Count != 1 {
		t.Errorf("segment (statements, count) = (%d,%d), want (1,1)", seg.Statements, seg.Count)
	}

	seg2 := profile["bambot/pkg/ai/agent.go"][1]
	if seg2.Statements != 2 || seg2.Count != 0 {
		t.Errorf("segment2 (statements, count) = (%d,%d), want (2,0)", seg2.Statements, seg2.Count)
	}
}

func TestExtractFunctions(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "calc.go")
	code := `package main

func Add(x, y int) int {
	return x + y
}

func (c *Calc) Multiply(a, b int) int {
	return a * b
}
`
	if err := os.WriteFile(src, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}

	functions, err := extractFunctions(src)
	if err != nil {
		t.Fatalf("extractFunctions() error = %v", err)
	}

	if len(functions) != 2 {
		t.Fatalf("extractFunctions() returned %d functions, want 2", len(functions))
	}

	if functions[0].Name != "Add" {
		t.Errorf("functions[0].Name = %q, want Add", functions[0].Name)
	}
	if functions[0].StartLine < 3 || functions[0].EndLine < 5 {
		t.Errorf("Add range should span lines 3-5, got %d-%d", functions[0].StartLine, functions[0].EndLine)
	}

	if functions[1].StartLine < 7 || functions[1].EndLine < 9 {
		t.Errorf("Multiply range should span around lines 7-9, got %d-%d", functions[1].StartLine, functions[1].EndLine)
	}
	if functions[1].Name != "Calc.Multiply" {
		t.Errorf("functions[1].Name = %q, want Calc.Multiply", functions[1].Name)
	}
}

func TestExtractFunctions_NoFunctions(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "empty.go")
	if err := os.WriteFile(src, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	functions, err := extractFunctions(src)
	if err != nil {
		t.Fatalf("extractFunctions() error = %v", err)
	}
	if len(functions) != 0 {
		t.Errorf("expected 0 functions, got %d", len(functions))
	}
}

func TestComputeCoverage(t *testing.T) {
	profile := map[string][]coverSegment{
		"example.com/mod/pkg/calc.go": {
			{StartLine: 3, EndLine: 5, Statements: 4, Count: 3},
			{StartLine: 6, EndLine: 8, Statements: 2, Count: 0},
			{StartLine: 10, EndLine: 12, Statements: 5, Count: 0},
		},
		"example.com/mod/pkg/util.go": {
			{StartLine: 7, EndLine: 9, Statements: 1, Count: 0},
		},
	}

	functions := []functionRange{
		{File: "pkg/calc.go", StartLine: 3, EndLine: 5},
		{File: "pkg/util.go", StartLine: 7, EndLine: 9},
		{File: "pkg/calc.go", StartLine: 5, EndLine: 11},
		{File: "pkg/missing.go", StartLine: 1, EndLine: 3},
	}

	results := computeCoverage(profile, functions)

	if len(results) != 4 {
		t.Fatalf("computeCoverage() returned %d results, want 4", len(results))
	}

	if results[0].Coverage != 100.0 {
		t.Errorf("calc.go lines 3-5 coverage = %.1f, want 100.0 (4/4 covered)", results[0].Coverage)
	}

	if results[1].Coverage != 0.0 {
		t.Errorf("util.go lines 7-9 coverage = %.1f, want 0.0 (0/1 covered)", results[1].Coverage)
	}

	want := 400.0 / 11.0
	if math.Abs(results[2].Coverage-want) > 0.01 {
		t.Errorf("calc.go lines 5-11 coverage = %.2f, want ~%.2f (4/11 covered)", results[2].Coverage, want)
	}

	if results[3].Coverage != 0.0 {
		t.Errorf("missing.go coverage = %.1f, want 0.0 (no matching file)", results[3].Coverage)
	}
}

func TestSegmentsForFile(t *testing.T) {
	profile := map[string][]coverSegment{
		"github.com/mfenderov/gocrap/analyzer.go": {
			{StartLine: 10, EndLine: 15, Statements: 3, Count: 2},
		},
		"other.com/pkg/test.go": {
			{StartLine: 1, EndLine: 3, Statements: 1, Count: 1},
		},
	}

	tests := []struct {
		name    string
		file    string
		wantLen int
	}{
		{"exact match", "analyzer.go", 1},
		{"suffix match with prefix", "gocrap/analyzer.go", 1},
		{"no match", "unknown.go", 0},
		{"match other file", "test.go", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(segmentsForFile(profile, tt.file)); got != tt.wantLen {
				t.Errorf("segmentsForFile(_, %q) = %d segments, want %d", tt.file, got, tt.wantLen)
			}
		})
	}
}
