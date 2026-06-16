package application

import (
	"context"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/team"
)

// HtmlCommand drives the html sub-command.
type HtmlCommand struct {
	Window  analytics.QueryWindow
	OutFile string // required: --out <path>
	Team    bool   // --team: render team rollup HTML
}

// HtmlHandler produces HTML reports.
type HtmlHandler struct{ Deps AppDeps }

// Handle generates the HTML dashboard and writes to OutFile.
func (h HtmlHandler) Handle(ctx context.Context, cmd HtmlCommand) error {
	evs, err := h.Deps.Reader.LoadEvents(ctx, queryFilter(cmd.Window))
	if err != nil {
		return err
	}

	if cmd.Team {
		// For team HTML, caller provides pre-merged exports via Merge command.
		// Fallback: render empty team page.
		rollups := h.Deps.ExportMerger.Rollup(nil, team.TeamConfig{})
		html := h.Deps.HtmlRenderer.RenderTeamHtml(rollups, cmd.Window.Days)
		return writeText(html, cmd.OutFile)
	}

	m := h.Deps.MetricsEngine.Compute(evs)
	recs, _ := h.Deps.Recommender.Enrich(evs, m, cmd.Window.Days)
	trends, movers := h.Deps.Trends.ComputeTrends(evs, cmd.Window.Days, h.Deps.MetricsEngine)

	html := h.Deps.HtmlRenderer.RenderHtml(m, recs, trends, movers, cmd.Window.Days)
	return writeText(html, cmd.OutFile)
}
