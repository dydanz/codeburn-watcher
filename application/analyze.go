package application

import (
	"context"
	"fmt"
	"os"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/deepanalysis"
)

// AnalyzeQuery drives the analyze sub-command.
type AnalyzeQuery struct {
	Window analytics.QueryWindow
	LLM    bool   // --llm: pipe payload to local agent CLI
	Agent  string // --agent: override auto-detected agent
	JSON   bool   // --json: emit raw payload JSON
}

// AnalyzeHandler struct for compatibility (uses AppDeps via closure pattern).
type AnalyzeHandler struct{ Deps AppDeps }

// Handle loads events, runs deep analysis, and optionally pipes to an LLM.
func (h AnalyzeHandler) Handle(ctx context.Context, q AnalyzeQuery) error {
	evs, err := h.Deps.Reader.LoadEvents(ctx, queryFilter(q.Window))
	if err != nil {
		return err
	}

	m := h.Deps.MetricsEngine.Compute(evs)
	deep := h.Deps.Sessions.DeepAnalyze(evs)
	toolStats := h.Deps.Tools.ComputeToolStats(evs)
	deep.ToolStats = toolStats
	recs, _ := h.Deps.Recommender.Enrich(evs, m, q.Window.Days)

	if q.LLM || q.JSON {
		payload := h.Deps.Pipeline.BuildPayload(evs, m, recs, q.Window.Days)
		if q.JSON {
			return writeJSON(payload, "")
		}
		prompt := h.Deps.Pipeline.BuildPrompt(payload)
		agent, ok := h.Deps.Pipeline.DetectAgent()
		if !ok {
			return fmt.Errorf("no supported LLM agent found on PATH (claude, gemini, codex)")
		}
		os.Exit(h.Deps.Pipeline.Run(prompt, agent))
	}

	output := h.Deps.Renderer.RenderAnalysis(deep, m, recs, q.Window.Days)
	fmt.Print(output)
	return nil
}

var _ = deepanalysis.DeepAnalysis{} // ensure import
