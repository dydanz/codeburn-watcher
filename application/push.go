package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/team"
)

// PushCommand drives the push sub-command.
type PushCommand struct {
	Window analytics.QueryWindow
}

// PushHandler builds and pushes a signed export.
type PushHandler struct{ Deps AppDeps }

// Handle runs a JSON report and pushes the result to the configured URL.
func (h PushHandler) Handle(ctx context.Context, cmd PushCommand) error {
	evs, err := h.Deps.Reader.LoadEvents(ctx, queryFilter(cmd.Window))
	if err != nil {
		return err
	}

	m := h.Deps.MetricsEngine.Compute(evs)
	recs, _ := h.Deps.Recommender.Enrich(evs, m, cmd.Window.Days)
	trends, movers := h.Deps.Trends.ComputeTrends(evs, cmd.Window.Days, h.Deps.MetricsEngine)
	followThrough, err := h.Deps.FollowThrough.Sync(recs, m)
	if err != nil {
		return err
	}
	persona := h.Deps.Persona.Assign(m)

	payload := team.ExportPayload{
		Days: cmd.Window.Days, Overall: m,
		Recommendations: recs, Trends: trends, ProjectMovers: movers,
		FollowThrough: followThrough, Persona: persona,
	}
	export, err := h.Deps.ExportBuilder.Build(h.Deps.Identity, h.Deps.Username, payload)
	if err != nil {
		return fmt.Errorf("build export: %w", err)
	}

	data, err := json.Marshal(export)
	if err != nil {
		return fmt.Errorf("marshal export: %w", err)
	}

	filename := team.ExportFilename(h.Deps.Username, export.GeneratedAt)

	deployCfg, err := h.Deps.DeployConfig.Load()
	if err != nil {
		return fmt.Errorf("load deploy config: %w", err)
	}
	if deployCfg.PushURL == "" {
		return fmt.Errorf("push_url not configured — run 'token-monitor init --team <url>' first")
	}

	return h.Deps.Transport.Push(ctx, filename, data, deployCfg.PushURL)
}
