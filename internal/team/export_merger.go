package team

import (
	"sort"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// ExportMerger implements merge, dedupe, and rollup logic. All methods are pure.
type ExportMerger struct{}

// Dedupe keeps the latest GeneratedAt per username; preserves first-seen insertion
// order for deterministic rendering.
func (ExportMerger) Dedupe(exports []SignedExport) []SignedExport {
	latest := make(map[string]SignedExport)
	order := []string{}
	for _, e := range exports {
		prev, seen := latest[e.Username]
		if !seen {
			order = append(order, e.Username)
		}
		if !seen || e.GeneratedAt > prev.GeneratedAt {
			latest[e.Username] = e
		}
	}
	result := make([]SignedExport, 0, len(order))
	for _, u := range order {
		result = append(result, latest[u])
	}
	return result
}

// MergeMetrics sums absolute token/event values across exports and
// weighted-averages ratio fields.
func (ExportMerger) MergeMetrics(exports []SignedExport) analytics.Metrics {
	var merged analytics.Metrics
	n := len(exports)
	if n == 0 {
		return merged
	}
	for _, e := range exports {
		m := e.Payload.Overall
		merged.Events += m.Events
		merged.Sessions += m.Sessions
		merged.SpendTokens += m.SpendTokens
		merged.CostUsd += m.CostUsd
		merged.CacheHitRatio += m.CacheHitRatio
		merged.ReworkRatio += m.ReworkRatio
		merged.ThinkToCodeRatio += m.ThinkToCodeRatio
		merged.ContextBloatShare += m.ContextBloatShare
		merged.ColdRestartShare += m.ColdRestartShare
		merged.PremiumWasteShare += m.PremiumWasteShare
		merged.RetryShare += m.RetryShare
	}
	// weighted-average ratios
	fn := float64(n)
	merged.CacheHitRatio /= fn
	merged.ReworkRatio /= fn
	merged.ThinkToCodeRatio /= fn
	merged.ContextBloatShare /= fn
	merged.ColdRestartShare /= fn
	merged.PremiumWasteShare /= fn
	merged.RetryShare /= fn
	return merged
}

// Rollup groups exports by TeamConfig.Members; one Rollup per group + one "all".
func (m ExportMerger) Rollup(exports []SignedExport, config TeamConfig) []Rollup {
	assigner := insights.PersonaAssigner{}
	rollups := make([]Rollup, 0, len(config.Members)+1)

	// per-group rollups
	groupNames := make([]string, 0, len(config.Members))
	for g := range config.Members {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	for _, groupName := range groupNames {
		members := config.Members[groupName]
		memberSet := make(map[string]bool, len(members))
		for _, u := range members {
			memberSet[u] = true
		}
		var groupExports []SignedExport
		for _, e := range exports {
			if memberSet[e.Username] {
				groupExports = append(groupExports, e)
			}
		}
		if len(groupExports) == 0 {
			continue
		}
		metrics := m.MergeMetrics(groupExports)
		rollups = append(rollups, Rollup{
			Label:    groupName,
			Axis:     "group",
			Members:  members,
			Metrics:  metrics,
			Persona:  assigner.Assign(metrics),
			Dominant: m.DominantActivity(metrics),
		})
	}

	// "all" rollup
	allMetrics := m.MergeMetrics(exports)
	allMembers := make([]string, 0, len(exports))
	for _, e := range exports {
		allMembers = append(allMembers, e.Username)
	}
	rollups = append(rollups, Rollup{
		Label:    "all",
		Axis:     "all",
		Members:  allMembers,
		Metrics:  allMetrics,
		Persona:  assigner.Assign(allMetrics),
		Dominant: m.DominantActivity(allMetrics),
	})

	return rollups
}

// DominantActivity returns the Activity with the highest token count.
// Iterates Activities slice in order; strict > so ties keep earlier activity.
func (ExportMerger) DominantActivity(m analytics.Metrics) shared.Activity {
	if len(m.ByActivity) == 0 {
		return shared.ActivityExploration
	}
	dominant := shared.Activities[0]
	bestTok := 0
	for _, act := range shared.Activities {
		if ab, ok := m.ByActivity[act]; ok && ab.Tokens > bestTok {
			bestTok = ab.Tokens
			dominant = act
		}
	}
	return dominant
}
