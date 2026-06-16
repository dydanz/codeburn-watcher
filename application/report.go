package application

import (
	"context"
	"fmt"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/team"
)

// ReportQuery drives the report sub-command.
type ReportQuery struct {
	Window  analytics.QueryWindow
	JSON    bool   // --json: emit signed ExportV1
	OutFile string // --out: write to file instead of stdout
}

// ReportHandler produces the main usage report.
type ReportHandler struct{ Deps AppDeps }

// Handle computes metrics, recommendations, trends, and optional signed export.
func (h ReportHandler) Handle(ctx context.Context, q ReportQuery) error {
	evs, err := h.Deps.Reader.LoadEvents(ctx, queryFilter(q.Window))
	if err != nil {
		return err
	}

	m := h.Deps.MetricsEngine.Compute(evs)
	recs, _ := h.Deps.Recommender.Enrich(evs, m, q.Window.Days)
	trends, movers := h.Deps.Trends.ComputeTrends(evs, q.Window.Days, h.Deps.MetricsEngine)
	followThrough, err := h.Deps.FollowThrough.Sync(recs, m)
	if err != nil {
		return err
	}
	persona := h.Deps.Persona.Assign(m)

	if q.JSON {
		payload := team.ExportPayload{
			Days:            q.Window.Days,
			Overall:         m,
			Recommendations: recs,
			Trends:          trends,
			ProjectMovers:   movers,
			FollowThrough:   followThrough,
			Persona:         persona,
		}
		export, err := h.Deps.ExportBuilder.Build(h.Deps.Identity, h.Deps.Username, payload)
		if err != nil {
			return fmt.Errorf("build export: %w", err)
		}
		return writeJSON(export, q.OutFile)
	}

	output := h.Deps.Renderer.RenderReport(m, recs, trends, movers, q.Window.Days)
	return writeText(output, q.OutFile)
}
