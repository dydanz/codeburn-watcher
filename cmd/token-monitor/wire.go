package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dydanz/codeburn-watcher/application"
	infraadapters "github.com/dydanz/codeburn-watcher/infrastructure/adapters"
	"github.com/dydanz/codeburn-watcher/infrastructure/fs"
	infrahttp "github.com/dydanz/codeburn-watcher/infrastructure/http"
	"github.com/dydanz/codeburn-watcher/infrastructure/render"
	"github.com/dydanz/codeburn-watcher/infrastructure/scheduler"
	infrasqlite "github.com/dydanz/codeburn-watcher/infrastructure/sqlite"
	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/deepanalysis"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/team"
	"github.com/dydanz/codeburn-watcher/internal/trust"
)

func buildDeps() (application.AppDeps, func(), error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return application.AppDeps{}, nil, err
	}
	dbPath := filepath.Join(home, ".token-monitor", "events.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return application.AppDeps{}, nil, err
	}

	db, err := infrasqlite.Open(dbPath)
	if err != nil {
		return application.AppDeps{}, nil, fmt.Errorf("open db: %w", err)
	}
	cleanup := func() { db.Close() }

	eventStore := infrasqlite.NewEventStore(db)
	ftStore := infrasqlite.NewFollowThroughStore(db)

	keyDir := filepath.Join(home, ".token-monitor")
	keystore := fs.FileKeystore{}
	deployConfig := fs.FileDeployConfig{}

	transport := infrahttp.CompositeExportTransport{}

	pricing := analytics.NewPricingService()
	metricsEngine := analytics.NewMetricsEngine(pricing)
	recommender := insights.NewRecommendationEngine(pricing)
	followThrough := insights.NewFollowThroughService(ftStore)

	// Load signing identity (best-effort — not all commands need it)
	var identity trust.SigningIdentity
	var username string
	if priv, pub, err := keystore.LoadKeypair(keyDir); err == nil {
		if id, err := trust.NewSigningIdentity(priv, pub); err == nil {
			identity = id
		}
	}
	if cfg, err := deployConfig.Load(); err == nil {
		username = cfg.Username
	}

	exportBuilder := team.ExportBuilder{Canonicalizer: trust.CanonicalizationService{}}

	return application.AppDeps{
		Reader:        eventStore,
		Writer:        eventStore,
		Keystore:      keystore,
		Transport:     transport,
		DeployConfig:  deployConfig,
		Scheduler:     scheduler.NewScheduler(),
		Adapters:      infraadapters.AllAdapters(),
		Pricing:       pricing,
		MetricsEngine: metricsEngine,
		Recommender:   recommender,
		Trends:        insights.TrendAnalyzer{},
		FollowThrough: followThrough,
		Persona:       insights.PersonaAssigner{},
		ExportBuilder: exportBuilder,
		ExportMerger:  team.ExportMerger{},
		Sessions:      deepanalysis.SessionAnalyzer{Pricing: pricing},
		Tools:         deepanalysis.ToolAnalyzer{},
		Pipeline:      deepanalysis.LlmPipeline{},
		Verifier:      trust.VerificationService{},
		Identity:      identity,
		Username:      username,
		Renderer:      render.TerminalRenderer{},
		HtmlRenderer:  render.HtmlRenderer{},
	}, cleanup, nil
}
