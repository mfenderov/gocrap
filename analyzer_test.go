package main

import (
	"math"
	"testing"
)

func TestCRAPScore(t *testing.T) {
	tests := []struct {
		name       string
		complexity int
		coverage   float64
		wantCRAP   float64
	}{
		{
			name:       "perfect: complexity 1, 100% coverage",
			complexity: 1,
			coverage:   100.0,
			wantCRAP:   1.0,
		},
		{
			name:       "simple uncovered: complexity 1, 0% coverage",
			complexity: 1,
			coverage:   0.0,
			wantCRAP:   2.0, // 1*1*(1-0)^3 + 1 = 1 + 1 = 2
		},
		{
			name:       "complex well-tested: complexity 10, 100% coverage",
			complexity: 10,
			coverage:   100.0,
			wantCRAP:   10.0, // 100*(1-1)^3 + 10 = 0 + 10 = 10
		},
		{
			name:       "complex untested: complexity 10, 0% coverage",
			complexity: 10,
			coverage:   0.0,
			wantCRAP:   110.0, // 100*(1-0)^3 + 10 = 100 + 10 = 110
		},
		{
			name:       "moderate: complexity 5, 50% coverage",
			complexity: 5,
			coverage:   50.0,
			wantCRAP:   8.125, // 25*(0.5)^3 + 5 = 25*0.125 + 5 = 3.125 + 5 = 8.125
		},
		{
			name:       "threshold boundary: complexity 15, 40% coverage",
			complexity: 15,
			coverage:   40.0,
			wantCRAP:   63.6, // 225*(0.6)^3 + 15 = 225*0.216 + 15 = 48.6 + 15 = 63.6
		},
		{
			name:       "zero complexity",
			complexity: 0,
			coverage:   0.0,
			wantCRAP:   0.0,
		},
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
total:					(statements)	22.9%`

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

func TestProcessResults_FilterTests(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Handle", File: "agent.go", CRAP: 50},
		{FuncName: "TestHandle", File: "agent_test.go", CRAP: 40},
		{FuncName: "Chat", File: "client.go", CRAP: 5},
	}

	got := processResults(results, true, 0, 0)
	if len(got) != 2 {
		t.Fatalf("processResults(noTests=true) returned %d results, want 2", len(got))
	}
	if got[0].File != "agent.go" || got[1].File != "client.go" {
		t.Errorf("unexpected files: %q, %q", got[0].File, got[1].File)
	}
}

func TestProcessResults_FilterOver(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Handle", CRAP: 50},
		{FuncName: "Chat", CRAP: 5},
		{FuncName: "Run", CRAP: 35},
	}

	got := processResults(results, false, 30, 0)
	if len(got) != 2 {
		t.Fatalf("processResults(over=30) returned %d results, want 2", len(got))
	}
}

func TestProcessResults_Top(t *testing.T) {
	results := []FuncResult{
		{FuncName: "A", CRAP: 50},
		{FuncName: "B", CRAP: 40},
		{FuncName: "C", CRAP: 30},
	}

	got := processResults(results, false, 0, 2)
	if len(got) != 2 {
		t.Fatalf("processResults(top=2) returned %d results, want 2", len(got))
	}
	if got[0].FuncName != "A" || got[1].FuncName != "B" {
		t.Errorf("unexpected top 2: %q, %q", got[0].FuncName, got[1].FuncName)
	}
}

func TestProcessResults_Combined(t *testing.T) {
	results := []FuncResult{
		{FuncName: "Handle", File: "agent.go", CRAP: 50},
		{FuncName: "TestA", File: "agent_test.go", CRAP: 100},
		{FuncName: "Chat", File: "client.go", CRAP: 5},
		{FuncName: "Run", File: "main.go", CRAP: 35},
	}

	got := processResults(results, true, 10, 1)
	if len(got) != 1 {
		t.Fatalf("processResults(noTests+over+top) returned %d, want 1", len(got))
	}
	if got[0].FuncName != "Handle" {
		t.Errorf("got %q, want Handle", got[0].FuncName)
	}
}

func TestSummarize(t *testing.T) {
	results := []FuncResult{
		{CRAP: 50},
		{CRAP: 10},
		{CRAP: 1},
	}

	avg, total, crappy := summarize(results)
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if crappy != 1 {
		t.Errorf("crappy = %d, want 1", crappy)
	}
	if math.Abs(avg-20.33) > 0.01 {
		t.Errorf("avg = %.2f, want 20.33", avg)
	}
}

func TestSummarize_Empty(t *testing.T) {
	avg, total, crappy := summarize(nil)
	if total != 0 || crappy != 0 || avg != 0 {
		t.Errorf("summarize(nil) = (%.1f, %d, %d), want (0, 0, 0)", avg, total, crappy)
	}
}

func TestDetectModulePrefix(t *testing.T) {
	comp := []complexityStat{
		{File: "pkg/ai/agent.go", Line: 10},
	}
	cov := []coverageStat{
		{File: "bambot/pkg/ai/agent.go", Line: 10},
	}

	got := detectModulePrefix(cov, comp)
	if got != "bambot/" {
		t.Errorf("detectModulePrefix() = %q, want %q", got, "bambot/")
	}
}

func TestDetectModulePrefix_Empty(t *testing.T) {
	got := detectModulePrefix(nil, nil)
	if got != "" {
		t.Errorf("detectModulePrefix(nil, nil) = %q, want empty", got)
	}
}

func TestDetectModulePrefix_NoMatch(t *testing.T) {
	comp := []complexityStat{{File: "foo.go"}}
	cov := []coverageStat{{File: "bar.go"}}

	got := detectModulePrefix(cov, comp)
	if got != "" {
		t.Errorf("detectModulePrefix(no match) = %q, want empty", got)
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

	// Handle: complexity 15, coverage 32.5%
	// CRAP = 15^2 * (1 - 0.325)^3 + 15 = 225 * 0.307 + 15 = 69.09 + 15 = 84.09
	if results[0].FuncName != "(*AgentHandler).Handle" {
		t.Errorf("results[0].FuncName = %q, want %q", results[0].FuncName, "(*AgentHandler).Handle")
	}
	if results[0].Complexity != 15 {
		t.Errorf("results[0].Complexity = %d, want 15", results[0].Complexity)
	}
	if math.Abs(results[0].Coverage-32.5) > 0.01 {
		t.Errorf("results[0].Coverage = %.1f, want 32.5", results[0].Coverage)
	}

	// NewGroqClient: complexity 3, coverage 100% → CRAP = 3
	if math.Abs(results[1].CRAP-3.0) > 0.01 {
		t.Errorf("results[1].CRAP = %.3f, want 3.0", results[1].CRAP)
	}

	// Chat: complexity 1, coverage 100% → CRAP = 1
	if math.Abs(results[2].CRAP-1.0) > 0.01 {
		t.Errorf("results[2].CRAP = %.3f, want 1.0", results[2].CRAP)
	}
}
