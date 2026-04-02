package main

import (
	"math"
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

func TestFuncResult_IsCrappy(t *testing.T) {
	tests := []struct {
		name      string
		crap      float64
		threshold float64
		want      bool
	}{
		{"below threshold", 10.0, 30.0, false},
		{"at threshold", 30.0, 30.0, true},
		{"above threshold", 42.0, 30.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FuncResult{CRAP: tt.crap}
			if got := f.IsCrappy(tt.threshold); got != tt.want {
				t.Errorf("FuncResult{CRAP: %.1f}.IsCrappy(%.1f) = %v, want %v", tt.crap, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestParseCoverFunc(t *testing.T) {
	input := `bambot/pkg/ai/agent.go:73:	NewAgentHandler		75.0%
bambot/pkg/ai/agent.go:287:	executeToolLoop		76.9%
bambot/pkg/ai/client_groq.go:25:	NewGroqClient		100.0%
total:					(statements)	22.9%
badline
nocolon	func	notpercent
file.go:10:	funcName	abc%`

	results, err := parseCoverFunc(input)
	if err != nil {
		t.Fatalf("parseCoverFunc() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("parseCoverFunc() returned %d results, want 3", len(results))
	}

	want := []struct {
		file     string
		line     int
		funcName string
		coverage float64
	}{
		{"bambot/pkg/ai/agent.go", 73, "NewAgentHandler", 75.0},
		{"bambot/pkg/ai/agent.go", 287, "executeToolLoop", 76.9},
		{"bambot/pkg/ai/client_groq.go", 25, "NewGroqClient", 100.0},
	}

	for i, w := range want {
		if results[i].File != w.file {
			t.Errorf("result[%d].File = %q, want %q", i, results[i].File, w.file)
		}
		if results[i].Line != w.line {
			t.Errorf("result[%d].Line = %d, want %d", i, results[i].Line, w.line)
		}
		if results[i].FuncName != w.funcName {
			t.Errorf("result[%d].FuncName = %q, want %q", i, results[i].FuncName, w.funcName)
		}
		if math.Abs(results[i].Coverage-w.coverage) > 0.01 {
			t.Errorf("result[%d].Coverage = %.1f, want %.1f", i, results[i].Coverage, w.coverage)
		}
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

func TestDetectModulePrefix(t *testing.T) {
	comp := []complexityStat{{File: "pkg/ai/agent.go", Line: 10}}
	cov := []coverageStat{{File: "bambot/pkg/ai/agent.go", Line: 10}}

	got := detectModulePrefix(cov, comp)
	if got != "bambot/" {
		t.Errorf("detectModulePrefix() = %q, want %q", got, "bambot/")
	}
}

func TestDetectModulePrefix_Empty(t *testing.T) {
	if got := detectModulePrefix(nil, nil); got != "" {
		t.Errorf("detectModulePrefix(nil, nil) = %q, want empty", got)
	}
}

func TestDetectModulePrefix_NoMatch(t *testing.T) {
	comp := []complexityStat{{File: "foo.go"}}
	cov := []coverageStat{{File: "bar.go"}}
	if got := detectModulePrefix(cov, comp); got != "" {
		t.Errorf("detectModulePrefix(no match) = %q, want empty", got)
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
		{File: "bambot/pkg/ai/agent.go", Line: 122, FuncName: "Handle", Coverage: 32.5},
		{File: "bambot/pkg/ai/client_groq.go", Line: 25, FuncName: "NewGroqClient", Coverage: 100.0},
		{File: "bambot/pkg/ai/client_groq.go", Line: 42, FuncName: "Chat", Coverage: 100.0},
	}

	results := joinResults(complexity, coverage, "bambot/")

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
