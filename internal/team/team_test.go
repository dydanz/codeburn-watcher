package team_test

import (
	"testing"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/shared"
	"github.com/dydanz/codeburn-watcher/internal/team"
)

func TestTeamConfigParser(t *testing.T) {
	raw := `
username = alice
push_url = https://example.com/exports

[members]
backend = alice, bob
frontend = charlie
`
	cfg, err := team.TeamConfigParser{}.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if cfg.Username != "alice" {
		t.Errorf("Username = %q, want alice", cfg.Username)
	}
	if cfg.PushURL != "https://example.com/exports" {
		t.Errorf("PushURL = %q, want https://example.com/exports", cfg.PushURL)
	}
	if len(cfg.Members["backend"]) != 2 {
		t.Errorf("backend members = %v, want 2", cfg.Members["backend"])
	}
	if len(cfg.Members["frontend"]) != 1 || cfg.Members["frontend"][0] != "charlie" {
		t.Errorf("frontend members = %v, want [charlie]", cfg.Members["frontend"])
	}
}

func TestTeamConfigParser_MissingUsername(t *testing.T) {
	_, err := team.TeamConfigParser{}.Parse("push_url = https://x.com")
	if err == nil {
		t.Error("expected error for missing username")
	}
}

func TestExportFilename(t *testing.T) {
	got := team.ExportFilename("alice", "2025-01-15T10:30:00Z")
	want := "alice-2025-01-15T103000Z.json"
	if got != want {
		t.Errorf("ExportFilename = %q, want %q", got, want)
	}
}

func TestDedupe(t *testing.T) {
	merger := team.ExportMerger{}
	exports := []team.SignedExport{
		{Username: "alice", GeneratedAt: "2025-01-01T10:00:00Z"},
		{Username: "bob", GeneratedAt: "2025-01-01T09:00:00Z"},
		{Username: "alice", GeneratedAt: "2025-01-01T11:00:00Z"}, // later — should win
	}
	deduped := merger.Dedupe(exports)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 after dedupe, got %d", len(deduped))
	}
	if deduped[0].Username != "alice" {
		t.Error("alice should be first (first-seen order)")
	}
	if deduped[0].GeneratedAt != "2025-01-01T11:00:00Z" {
		t.Error("later alice export should win")
	}
}

func TestMergeMetrics(t *testing.T) {
	merger := team.ExportMerger{}
	exports := []team.SignedExport{
		{Payload: team.ExportPayload{Overall: analytics.Metrics{Events: 10, SpendTokens: 1000, CacheHitRatio: 0.4}}},
		{Payload: team.ExportPayload{Overall: analytics.Metrics{Events: 20, SpendTokens: 2000, CacheHitRatio: 0.6}}},
	}
	merged := merger.MergeMetrics(exports)
	if merged.Events != 30 {
		t.Errorf("Events = %d, want 30", merged.Events)
	}
	if merged.SpendTokens != 3000 {
		t.Errorf("SpendTokens = %d, want 3000", merged.SpendTokens)
	}
	if merged.CacheHitRatio != 0.5 {
		t.Errorf("CacheHitRatio = %f, want 0.5 (weighted avg)", merged.CacheHitRatio)
	}
}

func TestDominantActivity(t *testing.T) {
	merger := team.ExportMerger{}
	m := analytics.Metrics{
		ByActivity: map[shared.Activity]analytics.ActivityBreakdown{
			shared.ActivityCoding:      {Activity: shared.ActivityCoding, Tokens: 500},
			shared.ActivityExploration: {Activity: shared.ActivityExploration, Tokens: 300},
			shared.ActivityThinking:    {Activity: shared.ActivityThinking, Tokens: 100},
		},
	}
	got := merger.DominantActivity(m)
	if got != shared.ActivityCoding {
		t.Errorf("DominantActivity = %s, want coding", got)
	}
}
