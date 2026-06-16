package deepanalysis_test

import (
	"fmt"
	"testing"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/deepanalysis"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

func makeEvent(sid, project string, src shared.Source, act shared.Activity, input, output int, isErr bool, ts string) shared.StoredEvent {
	return shared.StoredEvent{
		SessionID:    sid,
		Project:      project,
		Source:       src,
		Activity:     act,
		InputTokens:  input,
		OutputTokens: output,
		Tools:        []string{"Read"},
		IsError:      isErr,
		TS:           ts,
	}
}

func TestComputeSessionStats_Basic(t *testing.T) {
	svc := deepanalysis.SessionAnalyzer{Pricing: analytics.PricingService{}}
	events := []shared.StoredEvent{
		makeEvent("s1", "proj", shared.SourceClaudeCode, shared.ActivityCoding, 100, 50, false, "2025-01-01T10:00:00Z"),
		makeEvent("s1", "proj", shared.SourceClaudeCode, shared.ActivityTesting, 200, 100, true, "2025-01-01T10:01:00Z"),
	}
	stats := svc.ComputeSessionStats(events)
	if len(stats) != 1 {
		t.Fatalf("expected 1 session, got %d", len(stats))
	}
	st := stats[0]
	if st.Turns != 2 {
		t.Errorf("Turns = %d, want 2", st.Turns)
	}
	if st.ErrorTurns != 1 {
		t.Errorf("ErrorTurns = %d, want 1", st.ErrorTurns)
	}
	if st.SpendTokens != 450 {
		t.Errorf("SpendTokens = %d, want 450", st.SpendTokens)
	}
}

func TestComputeSessionStats_FixIterations(t *testing.T) {
	svc := deepanalysis.SessionAnalyzer{Pricing: analytics.PricingService{}}
	// testing → coding = 1 fix iteration
	events := []shared.StoredEvent{
		makeEvent("s1", "p", shared.SourceClaudeCode, shared.ActivityTesting, 100, 50, false, "2025-01-01T10:00:00Z"),
		makeEvent("s1", "p", shared.SourceClaudeCode, shared.ActivityCoding, 100, 50, false, "2025-01-01T10:01:00Z"),
		makeEvent("s1", "p", shared.SourceClaudeCode, shared.ActivityTesting, 100, 50, false, "2025-01-01T10:02:00Z"),
		makeEvent("s1", "p", shared.SourceClaudeCode, shared.ActivityCoding, 100, 50, false, "2025-01-01T10:03:00Z"),
	}
	stats := svc.ComputeSessionStats(events)
	if stats[0].FixIterations != 2 {
		t.Errorf("FixIterations = %d, want 2", stats[0].FixIterations)
	}
}

func TestDeepAnalyze_TopN(t *testing.T) {
	svc := deepanalysis.SessionAnalyzer{Pricing: analytics.PricingService{}}
	var events []shared.StoredEvent
	// create 10 sessions, each with 1 event
	for i := range 10 {
		events = append(events, makeEvent(
			fmt.Sprintf("s%d", i), "p", shared.SourceClaudeCode,
			shared.ActivityCoding, (i+1)*100, 50, false,
			fmt.Sprintf("2025-01-01T10:%02d:00Z", i),
		))
	}
	da := svc.DeepAnalyze(events)
	if len(da.ExpensiveSessions) > 8 {
		t.Errorf("ExpensiveSessions should be capped at 8, got %d", len(da.ExpensiveSessions))
	}
	// most expensive session should be first
	if da.ExpensiveSessions[0].SpendTokens < da.ExpensiveSessions[len(da.ExpensiveSessions)-1].SpendTokens {
		t.Error("ExpensiveSessions not sorted descending by SpendTokens")
	}
}

func TestToolAnalyzer(t *testing.T) {
	ta := deepanalysis.ToolAnalyzer{}
	events := []shared.StoredEvent{
		{Tools: []string{"Read", "Write"}, IsError: false, InputTokens: 100},
		{Tools: []string{"Read"}, IsError: true, InputTokens: 200},
		{Tools: []string{"Bash"}, IsError: false, InputTokens: 50},
	}
	stats := ta.ComputeToolStats(events)
	byTool := make(map[string]deepanalysis.ToolStat)
	for _, s := range stats {
		byTool[s.Tool] = s
	}
	readStat := byTool["read"]
	if readStat.Turns != 2 {
		t.Errorf("Read.Turns = %d, want 2", readStat.Turns)
	}
	if readStat.ErrorTurns != 1 {
		t.Errorf("Read.ErrorTurns = %d, want 1", readStat.ErrorTurns)
	}
	if readStat.ErrorRate != 0.5 {
		t.Errorf("Read.ErrorRate = %f, want 0.5", readStat.ErrorRate)
	}
}
