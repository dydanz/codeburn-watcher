package application

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/shared"
	sharedports "github.com/dydanz/codeburn-watcher/internal/shared/ports"
)

// writeJSON marshals v to JSON and writes to path (or stdout if path is "").
func writeJSON(v any, path string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	return writeBytes(append(data, '\n'), path)
}

// writeText writes text to path (or stdout if path is "").
func writeText(text, path string) error {
	return writeBytes([]byte(text), path)
}

// writeBytes writes data to path (or stdout if path is "").
func writeBytes(data []byte, path string) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// toLocalTotals sums per-model token counts from stored events.
func toLocalTotals(events []shared.StoredEvent) map[string]int {
	totals := make(map[string]int, len(events))
	for _, e := range events {
		totals[e.Model] += e.InputTokens + e.OutputTokens + e.CacheReadTokens + e.CacheCreationTokens
	}
	return totals
}

// queryFilter converts an analytics.QueryWindow to a shared/ports.QueryFilter.
func queryFilter(w analytics.QueryWindow) sharedports.QueryFilter {
	return sharedports.QueryFilter{
		Days:          w.Days,
		ProjectFilter: w.ProjectFilter,
		SourceFilter:  w.SourceFilter,
	}
}
