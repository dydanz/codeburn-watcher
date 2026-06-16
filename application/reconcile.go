package application

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dydanz/codeburn-watcher/infrastructure/providers"
	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/reconciliation"
)

// ReconcileCommand drives the reconcile sub-command.
type ReconcileCommand struct {
	Provider string // --provider: anthropic|openai
	Days     int    // --days: look-back window
}

// ReconcileHandler fetches provider billing data and compares with local events.
type ReconcileHandler struct{ Deps AppDeps }

// Handle runs reconciliation for the given provider and window.
func (h ReconcileHandler) Handle(ctx context.Context, cmd ReconcileCommand) error {
	spec, ok := providers.ProviderRegistry[cmd.Provider]
	if !ok {
		return fmt.Errorf("unknown provider %q — supported: anthropic, openai", cmd.Provider)
	}

	apiKey := os.Getenv(spec.KeyEnv)
	if apiKey == "" {
		return fmt.Errorf("env %s not set", spec.KeyEnv)
	}
	baseURL := spec.DefaultURL
	if v := os.Getenv(spec.URLEnv); v != "" {
		baseURL = v
	}

	days := cmd.Days
	if days == 0 {
		days = 7
	}
	window := analytics.QueryWindow{Days: days}

	now := time.Now().UTC()
	startMs := now.AddDate(0, 0, -days).UnixMilli()
	endMs := now.UnixMilli()

	apiUsages, err := spec.Adapter.FetchUsage(ctx, baseURL, apiKey, startMs, endMs)
	if err != nil {
		return fmt.Errorf("fetch %s usage: %w", cmd.Provider, err)
	}

	evs, err := h.Deps.Reader.LoadEvents(ctx, queryFilter(window))
	if err != nil {
		return err
	}
	localTotals := toLocalTotals(evs)

	svc := reconciliation.ReconciliationService{}
	result := svc.Compare(cmd.Provider, window, localTotals, apiUsages)

	output := h.Deps.Renderer.RenderReconcile(cmd.Provider, result, days)
	fmt.Print(output)
	return nil
}
