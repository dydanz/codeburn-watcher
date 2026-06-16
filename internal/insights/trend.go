package insights

import (
	"fmt"
	"sort"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// TrendVerdict is the direction of change for a metric between two windows.
type TrendVerdict string

const (
	TrendUp   TrendVerdict = "up"
	TrendDown TrendVerdict = "down"
	TrendFlat TrendVerdict = "flat"
)

// TrendRow captures one metric's trend across two windows.
type TrendRow struct {
	Key        MetricKey
	Label      string
	Current    float64
	Previous   float64
	Delta      float64
	Verdict    TrendVerdict
	FmtCurrent string
	FmtDelta   string
}

// ProjectMover captures one project's spend change between windows.
type ProjectMover struct {
	Project  string
	Current  int
	Previous int
	Delta    int
	Verdict  TrendVerdict
}

// TrendAnalyzer computes TrendRows and ProjectMovers by splitting the event window.
type TrendAnalyzer struct{}

// SplitWindow divides events into current (latter half) and previous (first half) windows.
func (TrendAnalyzer) SplitWindow(events []shared.StoredEvent, days int) (current, previous []shared.StoredEvent) {
	if len(events) == 0 || days == 0 {
		return nil, nil
	}
	cutoff := time.Now().UTC().Add(-time.Duration(days/2) * 24 * time.Hour)
	for _, ev := range events {
		ts, err := time.Parse(time.RFC3339, ev.TS)
		if err != nil {
			ts, _ = time.Parse("2006-01-02T15:04:05Z", ev.TS)
		}
		if ts.After(cutoff) {
			current = append(current, ev)
		} else {
			previous = append(previous, ev)
		}
	}
	return current, previous
}

// ComputeTrends returns metric trend rows and project movers.
func (t TrendAnalyzer) ComputeTrends(
	events []shared.StoredEvent,
	days int,
	engine analytics.MetricsEngine,
) ([]TrendRow, []ProjectMover) {
	current, previous := t.SplitWindow(events, days)
	mCur := engine.Compute(current)
	mPrev := engine.Compute(previous)

	rows := []TrendRow{
		trendRow(FindingCacheMiss, "Cache hit ratio", mCur.CacheHitRatio, mPrev.CacheHitRatio, true),
		trendRow(FindingRework, "Rework ratio", mCur.ReworkRatio, mPrev.ReworkRatio, false),
		trendRow(FindingPremiumWaste, "Premium waste share", mCur.PremiumWasteShare, mPrev.PremiumWasteShare, false),
		trendRow(FindingColdRestart, "Cold restart share", mCur.ColdRestartShare, mPrev.ColdRestartShare, false),
		trendRow(FindingBloat, "Context bloat share", mCur.ContextBloatShare, mPrev.ContextBloatShare, false),
		trendRow(FindingRetryLoop, "Retry share", mCur.RetryShare, mPrev.RetryShare, false),
	}

	movers := projectMovers(current, previous)
	return rows, movers
}

// trendRow builds a TrendRow. higherIsBetter=true means an increase is "up" (improving).
func trendRow(key MetricKey, label string, cur, prev float64, higherIsBetter bool) TrendRow {
	delta := cur - prev
	var verdict TrendVerdict
	const flatThreshold = 0.02
	switch {
	case delta > flatThreshold:
		if higherIsBetter {
			verdict = TrendUp
		} else {
			verdict = TrendDown
		}
	case delta < -flatThreshold:
		if higherIsBetter {
			verdict = TrendDown
		} else {
			verdict = TrendUp
		}
	default:
		verdict = TrendFlat
	}
	return TrendRow{
		Key: key, Label: label,
		Current: cur, Previous: prev, Delta: delta, Verdict: verdict,
		FmtCurrent: fmt.Sprintf("%.1f%%", cur*100),
		FmtDelta:   fmt.Sprintf("%+.1f%%", delta*100),
	}
}

func projectMovers(current, previous []shared.StoredEvent) []ProjectMover {
	curSpend := map[string]int{}
	prevSpend := map[string]int{}
	for _, ev := range current {
		curSpend[ev.Project] += ev.InputTokens + ev.OutputTokens
	}
	for _, ev := range previous {
		prevSpend[ev.Project] += ev.InputTokens + ev.OutputTokens
	}

	seen := map[string]bool{}
	var movers []ProjectMover
	for p := range curSpend {
		seen[p] = true
	}
	for p := range prevSpend {
		seen[p] = true
	}
	for p := range seen {
		c, v := curSpend[p], prevSpend[p]
		delta := c - v
		if delta == 0 {
			continue
		}
		verdict := TrendUp
		if delta < 0 {
			verdict = TrendDown
		}
		movers = append(movers, ProjectMover{Project: p, Current: c, Previous: v, Delta: delta, Verdict: verdict})
	}
	sort.Slice(movers, func(i, j int) bool {
		ai, aj := movers[i].Delta, movers[j].Delta
		if ai < 0 {
			ai = -ai
		}
		if aj < 0 {
			aj = -aj
		}
		return ai > aj
	})
	return movers
}
