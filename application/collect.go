package application

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/collection"
	"github.com/dydanz/codeburn-watcher/internal/shared"
	sharedevents "github.com/dydanz/codeburn-watcher/internal/shared/events"
	sharedports "github.com/dydanz/codeburn-watcher/internal/shared/ports"
)

// CollectCommand restricts collection to one source when non-empty.
type CollectCommand struct {
	Source shared.Source
}

// CollectHandler orchestrates the collection flow across all adapters.
type CollectHandler struct {
	Adapters []collection.AgentAdapter
	Writer   sharedports.EventWriter
}

// Handle runs all adapters (fail-soft) and writes events to the store.
func (h CollectHandler) Handle(ctx context.Context, cmd CollectCommand) ([]sharedevents.UsageEventsCollected, error) {
	var allRaw []sharedports.RawUsageEvent
	var results []sharedevents.UsageEventsCollected

	for _, adapter := range h.Adapters {
		evs, result, err := adapter.Collect(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s collect failed: %v\n", result.Source, err)
			continue
		}
		if cmd.Source != "" && result.Source != cmd.Source {
			continue
		}
		for _, e := range evs {
			allRaw = append(allRaw, usageEventToRaw(e))
		}
		results = append(results, sharedevents.UsageEventsCollected{
			Source:     result.Source,
			Found:      result.Found,
			Inserted:   result.Inserted,
			OccurredAt: time.Now(),
		})
	}

	if len(allRaw) > 0 {
		if _, err := h.Writer.WriteEvents(ctx, allRaw); err != nil {
			return nil, fmt.Errorf("writing events: %w", err)
		}
	}
	return results, nil
}

func usageEventToRaw(e collection.UsageEvent) sharedports.RawUsageEvent {
	return sharedports.RawUsageEvent{
		Source:              string(e.Source),
		SessionID:           e.SessionID,
		EventKey:            e.EventKey,
		Project:             e.Project,
		Branch:              e.Branch,
		Model:               e.Model,
		Tools:               e.Tools,
		Activity:            string(e.Activity),
		TS:                  e.TS,
		InputTokens:         e.InputTokens,
		OutputTokens:        e.OutputTokens,
		CacheReadTokens:     e.CacheReadTokens,
		CacheCreationTokens: e.CacheCreationTokens,
		IsError:             e.IsError,
	}
}
