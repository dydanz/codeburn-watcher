package insights

import (
	"context"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
)

// FollowThroughStatus indicates whether a finding is being acted on.
type FollowThroughStatus string

const (
	StatusImproving  FollowThroughStatus = "improving"
	StatusRegressing FollowThroughStatus = "regressing"
	StatusResolved   FollowThroughStatus = "resolved"
)

// FollowThroughRecord is an entity: persisted, has identity (key + baseline timestamp).
type FollowThroughRecord struct {
	Key           MetricKey
	BaselineValue float64
	BaselineTS    time.Time
	CurrentValue  float64
	Status        FollowThroughStatus
}

// FollowThroughService reconciles current findings against stored records.
// Has I/O via FollowThroughRepository.
type FollowThroughService struct {
	Repo FollowThroughRepository
}

// NewFollowThroughService returns a FollowThroughService backed by repo.
func NewFollowThroughService(repo FollowThroughRepository) FollowThroughService {
	return FollowThroughService{Repo: repo}
}

// Sync upserts FollowThroughRecords for the active findings and returns the
// updated set. Newly active findings get a baseline snapshot; previously
// active findings get a current-value update and a status determination.
func (f FollowThroughService) Sync(
	recs []EnrichedRecommendation,
	m analytics.Metrics,
) ([]FollowThroughRecord, error) {
	ctx := context.Background()
	stored, err := f.Repo.LoadRecords(ctx)
	if err != nil {
		return nil, err
	}

	byKey := map[MetricKey]FollowThroughRecord{}
	for _, r := range stored {
		byKey[r.Key] = r
	}

	now := time.Now().UTC()
	var updated []FollowThroughRecord

	for _, rec := range recs {
		cur := metricValue(rec.Key, m)
		existing, ok := byKey[rec.Key]
		if !ok {
			// new finding — establish baseline
			updated = append(updated, FollowThroughRecord{
				Key:           rec.Key,
				BaselineValue: cur,
				BaselineTS:    now,
				CurrentValue:  cur,
				Status:        StatusImproving,
			})
			continue
		}
		status := StatusRegressing
		if isImproving(rec.Key, existing.BaselineValue, cur) {
			status = StatusImproving
		}
		if isResolved(rec.Key, cur) {
			status = StatusResolved
		}
		updated = append(updated, FollowThroughRecord{
			Key:           rec.Key,
			BaselineValue: existing.BaselineValue,
			BaselineTS:    existing.BaselineTS,
			CurrentValue:  cur,
			Status:        status,
		})
	}

	if err := f.Repo.UpsertRecords(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func metricValue(key MetricKey, m analytics.Metrics) float64 {
	switch key {
	case FindingCacheMiss:
		return m.CacheHitRatio
	case FindingRework:
		return m.ReworkRatio
	case FindingPremiumWaste:
		return m.PremiumWasteShare
	case FindingColdRestart:
		return m.ColdRestartShare
	case FindingBloat:
		return m.ContextBloatShare
	case FindingThinkOvercode:
		return m.ThinkToCodeRatio
	case FindingRetryLoop:
		return m.RetryShare
	case FindingHighCost:
		return m.CostUsd
	default:
		return 0
	}
}

// isImproving returns true when the metric has moved toward its goal.
func isImproving(key MetricKey, baseline, current float64) bool {
	switch key {
	case FindingCacheMiss:
		return current > baseline // higher cache hit = better
	default:
		return current < baseline // lower ratio = better for everything else
	}
}

// isResolved returns true when the metric no longer triggers the finding.
func isResolved(key MetricKey, current float64) bool {
	for _, f := range findings {
		if f.Key == key {
			return !f.predicate(analytics.Metrics{
				CacheHitRatio:     maybeSet(key, FindingCacheMiss, current),
				ReworkRatio:       maybeSet(key, FindingRework, current),
				PremiumWasteShare: maybeSet(key, FindingPremiumWaste, current),
				ColdRestartShare:  maybeSet(key, FindingColdRestart, current),
				ContextBloatShare: maybeSet(key, FindingBloat, current),
				ThinkToCodeRatio:  maybeSet(key, FindingThinkOvercode, current),
				RetryShare:        maybeSet(key, FindingRetryLoop, current),
				CostUsd:           maybeSet(key, FindingHighCost, current),
			})
		}
	}
	return false
}

func maybeSet(target, check MetricKey, v float64) float64 {
	if target == check {
		return v
	}
	return 0
}
