package application_test

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"testing"
	"time"

	"github.com/dydanz/codeburn-watcher/application"
	"github.com/dydanz/codeburn-watcher/infrastructure/render"
	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/collection"
	"github.com/dydanz/codeburn-watcher/internal/deepanalysis"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/shared"
	sharedports "github.com/dydanz/codeburn-watcher/internal/shared/ports"
	"github.com/dydanz/codeburn-watcher/internal/team"
	"github.com/dydanz/codeburn-watcher/internal/trust"
)

// --- fake ports ---

type fakeWriter struct{ events []sharedports.RawUsageEvent }

func (f *fakeWriter) WriteEvents(_ context.Context, evs []sharedports.RawUsageEvent) (int, error) {
	f.events = append(f.events, evs...)
	return len(evs), nil
}

type fakeReader struct{ events []shared.StoredEvent }

func (f *fakeReader) LoadEvents(_ context.Context, _ sharedports.QueryFilter) ([]shared.StoredEvent, error) {
	return f.events, nil
}

type stubAdapter struct {
	events []collection.UsageEvent
}

func (s stubAdapter) Source() shared.Source { return shared.SourceClaudeCode }
func (s stubAdapter) Collect(_ context.Context) ([]collection.UsageEvent, collection.CollectResult, error) {
	return s.events, collection.CollectResult{Source: shared.SourceClaudeCode, Found: len(s.events)}, nil
}

// buildTestDeps constructs AppDeps with fake ports and provided events.
func buildTestDeps(t *testing.T, storedEvents []shared.StoredEvent) application.AppDeps {
	t.Helper()
	pricing := analytics.NewPricingService()
	return application.AppDeps{
		Reader:        &fakeReader{events: storedEvents},
		Writer:        &fakeWriter{},
		Keystore:      stubKeystore{},
		Scheduler:     &stubScheduler{},
		DeployConfig:  stubDeployConfig{},
		MetricsEngine: analytics.NewMetricsEngine(pricing),
		Recommender:   insights.NewRecommendationEngine(pricing),
		Trends:        insights.TrendAnalyzer{},
		FollowThrough: insights.NewFollowThroughService(stubFTRepo{}),
		Persona:       insights.PersonaAssigner{},
		ExportBuilder: team.ExportBuilder{Canonicalizer: trust.CanonicalizationService{}},
		ExportMerger:  team.ExportMerger{},
		Sessions:      deepanalysis.SessionAnalyzer{Pricing: pricing},
		Tools:         deepanalysis.ToolAnalyzer{},
		Pipeline:      deepanalysis.LlmPipeline{},
		Verifier:      trust.VerificationService{},
		Renderer:      render.TerminalRenderer{},
		HtmlRenderer:  render.HtmlRenderer{},
	}
}

func fixedEvent(source shared.Source, activity shared.Activity, input, output int) shared.StoredEvent {
	return shared.StoredEvent{
		Source:       source,
		SessionID:    "sess1",
		EventKey:     "e" + string(activity),
		Project:      "testproj",
		Activity:     activity,
		TS:           time.Now().UTC().Format(time.RFC3339),
		InputTokens:  input,
		OutputTokens: output,
	}
}

// --- CollectHandler ---

func TestCollectHandler_Basic(t *testing.T) {
	evs := []collection.UsageEvent{
		{
			Source: shared.SourceClaudeCode, SessionID: "s1", EventKey: "ev1",
			Activity:    shared.ActivityCoding,
			InputTokens: 100, OutputTokens: 50,
			TS: time.Now().UTC().Format(time.RFC3339),
		},
	}
	w := &fakeWriter{}
	h := application.CollectHandler{
		Adapters: []collection.AgentAdapter{stubAdapter{events: evs}},
		Writer:   w,
	}
	results, err := h.Handle(context.Background(), application.CollectCommand{})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}
	if len(w.events) != 1 {
		t.Errorf("stored events = %d, want 1", len(w.events))
	}
}

func TestCollectHandler_NoAdapters(t *testing.T) {
	h := application.CollectHandler{Adapters: nil, Writer: &fakeWriter{}}
	results, err := h.Handle(context.Background(), application.CollectCommand{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// --- ReportHandler ---

func TestReportHandler_Terminal(t *testing.T) {
	events := []shared.StoredEvent{
		fixedEvent(shared.SourceClaudeCode, shared.ActivityCoding, 500, 200),
		fixedEvent(shared.SourceClaudeCode, shared.ActivityThinking, 300, 100),
	}
	deps := buildTestDeps(t, events)
	h := application.ReportHandler{Deps: deps}
	err := h.Handle(context.Background(), application.ReportQuery{
		Window: analytics.QueryWindow{Days: 7},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
}

func TestReportHandler_EmptyEvents(t *testing.T) {
	deps := buildTestDeps(t, nil)
	h := application.ReportHandler{Deps: deps}
	err := h.Handle(context.Background(), application.ReportQuery{
		Window: analytics.QueryWindow{Days: 7},
	})
	if err != nil {
		t.Fatalf("Handle with no events: %v", err)
	}
}

// --- AnalyzeHandler ---

func TestAnalyzeHandler_Basic(t *testing.T) {
	events := []shared.StoredEvent{
		fixedEvent(shared.SourceClaudeCode, shared.ActivityCoding, 1000, 400),
	}
	deps := buildTestDeps(t, events)
	h := application.AnalyzeHandler{Deps: deps}
	err := h.Handle(context.Background(), application.AnalyzeQuery{
		Window: analytics.QueryWindow{Days: 7},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
}

// --- HtmlHandler ---

func TestHtmlHandler_Personal(t *testing.T) {
	events := []shared.StoredEvent{
		fixedEvent(shared.SourceClaudeCode, shared.ActivityCoding, 600, 200),
	}
	deps := buildTestDeps(t, events)
	h := application.HtmlHandler{Deps: deps}
	// write to /dev/null via OutFile — we just ensure no error
	err := h.Handle(context.Background(), application.HtmlCommand{
		Window:  analytics.QueryWindow{Days: 7},
		OutFile: t.TempDir() + "/out.html",
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
}

func TestHtmlHandler_Team(t *testing.T) {
	deps := buildTestDeps(t, nil)
	h := application.HtmlHandler{Deps: deps}
	err := h.Handle(context.Background(), application.HtmlCommand{
		Window:  analytics.QueryWindow{Days: 7},
		OutFile: t.TempDir() + "/team.html",
		Team:    true,
	})
	if err != nil {
		t.Fatalf("Handle team: %v", err)
	}
}

// --- FingerprintHandler ---

func TestFingerprintHandler_NoKeypair(t *testing.T) {
	// No keypair on disk — expect an error, not a panic
	deps := buildTestDeps(t, nil)
	h := application.FingerprintHandler{Deps: deps}
	err := h.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error when keypair missing, got nil")
	}
}

// --- ScheduleHandler ---

// stubKeystore returns an error on LoadKeypair (no key on disk).
type stubKeystore struct{}

func (stubKeystore) EnsureKeypair(_ string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	return nil, nil, fmt.Errorf("stubKeystore: not implemented")
}
func (stubKeystore) LoadKeypair(_ string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	return nil, nil, fmt.Errorf("stubKeystore: no keypair")
}
func (stubKeystore) LoadKeyring(_ string) (trust.Keyring, error) { return nil, nil }

type stubScheduler struct {
	installed bool
	removed   bool
}

func (s *stubScheduler) Install(_ string, _ team.DeployConfig) error {
	s.installed = true
	return nil
}
func (s *stubScheduler) Remove() error { s.removed = true; return nil }

type stubDeployConfig struct{ cfg team.DeployConfig }

func (s stubDeployConfig) Load() (team.DeployConfig, error) { return s.cfg, nil }
func (s stubDeployConfig) Save(_ team.DeployConfig) error   { return nil }

type stubFTRepo struct{}

func (stubFTRepo) LoadRecords(_ context.Context) ([]insights.FollowThroughRecord, error) {
	return nil, nil
}
func (stubFTRepo) UpsertRecords(_ context.Context, _ []insights.FollowThroughRecord) error {
	return nil
}

func TestScheduleHandler_Install(t *testing.T) {
	sched := &stubScheduler{}
	deps := buildTestDeps(t, nil)
	deps.Scheduler = sched
	deps.DeployConfig = stubDeployConfig{cfg: team.DeployConfig{PushURL: "https://example.com"}}
	h := application.ScheduleHandler{Deps: deps}
	if err := h.Handle(context.Background(), application.ScheduleCommand{Remove: false}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !sched.installed {
		t.Error("expected Scheduler.Install to be called")
	}
}

func TestScheduleHandler_Remove(t *testing.T) {
	sched := &stubScheduler{}
	deps := buildTestDeps(t, nil)
	deps.Scheduler = sched
	h := application.ScheduleHandler{Deps: deps}
	if err := h.Handle(context.Background(), application.ScheduleCommand{Remove: true}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if !sched.removed {
		t.Error("expected Scheduler.Remove to be called")
	}
}
