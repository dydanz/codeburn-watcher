package deepanalysis

import (
	"sort"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

const (
	coldRestartGap  = 5 * time.Minute
	bloatMinTurns   = 10
	bloatGrowthRatio = 2.0
	topN            = 8
)

// SessionAnalyzer computes per-session behavioral statistics. Pure — no I/O.
type SessionAnalyzer struct {
	Pricing analytics.PricingService
}

// ComputeSessionStats groups events by session and builds one SessionStat per session.
func (s SessionAnalyzer) ComputeSessionStats(events []shared.StoredEvent) []SessionStat {
	type sessionAccum struct {
		project   string
		source    shared.Source
		turns     []shared.StoredEvent
		actCounts map[shared.Activity]int
	}
	sessions := make(map[string]*sessionAccum)
	order := []string{}
	for _, e := range events {
		acc, ok := sessions[e.SessionID]
		if !ok {
			acc = &sessionAccum{
				project:   e.Project,
				source:    e.Source,
				actCounts: make(map[shared.Activity]int),
			}
			sessions[e.SessionID] = acc
			order = append(order, e.SessionID)
		}
		acc.turns = append(acc.turns, e)
		acc.actCounts[e.Activity]++
	}

	stats := make([]SessionStat, 0, len(sessions))
	for _, sid := range order {
		acc := sessions[sid]
		turns := acc.turns
		if len(turns) == 0 {
			continue
		}
		stat := buildStat(sid, acc.project, acc.source, turns, acc.actCounts, s.Pricing)
		stats = append(stats, stat)
	}
	return stats
}

// DeepAnalyze returns top-N lists across several behavioral dimensions.
func (s SessionAnalyzer) DeepAnalyze(events []shared.StoredEvent) DeepAnalysis {
	stats := s.ComputeSessionStats(events)

	top := func(src []SessionStat, filter func(SessionStat) bool, less func(a, b SessionStat) bool) []SessionStat {
		var filtered []SessionStat
		for _, st := range src {
			if filter(st) {
				filtered = append(filtered, st)
			}
		}
		sort.Slice(filtered, func(i, j int) bool { return less(filtered[i], filtered[j]) })
		if len(filtered) > topN {
			filtered = filtered[:topN]
		}
		return filtered
	}

	bySpend := func(a, b SessionStat) bool { return a.SpendTokens > b.SpendTokens }
	byFix := func(a, b SessionStat) bool { return a.FixIterations > b.FixIterations }
	byCtx := func(a, b SessionStat) bool { return a.AvgContextTokens > b.AvgContextTokens }
	byGrowth := func(a, b SessionStat) bool { return a.ContextGrowth > b.ContextGrowth }
	byCold := func(a, b SessionStat) bool { return a.ColdRestartTokens > b.ColdRestartTokens }

	return DeepAnalysis{
		ExpensiveSessions:    top(stats, func(st SessionStat) bool { return true }, bySpend),
		FixLoopSessions:      top(stats, func(st SessionStat) bool { return st.FixIterations >= 2 }, byFix),
		ContextHeavySessions: top(stats, func(st SessionStat) bool { return st.Turns >= 10 }, byCtx),
		BloatTrendSessions:   top(stats, func(st SessionStat) bool { return st.ContextGrowth >= bloatGrowthRatio }, byGrowth),
		ColdRestartSessions:  top(stats, func(st SessionStat) bool { return st.ColdRestartTurns > 0 }, byCold),
	}
}

func buildStat(
	sid, project string,
	source shared.Source,
	turns []shared.StoredEvent,
	actCounts map[shared.Activity]int,
	pricing analytics.PricingService,
) SessionStat {
	var spendTok, errTurns, coldRestartTok, coldRestartTurns int
	var totalCost float64
	var prevActivity shared.Activity
	var fixIter int

	type tsEvent struct {
		ts  time.Time
		tok int
	}
	var tsSlice []tsEvent
	var prevTime time.Time

	for _, e := range turns {
		tok := e.InputTokens + e.OutputTokens + e.CacheReadTokens + e.CacheCreationTokens
		spendTok += tok
		cost := pricing.CostOf(e.Model, e.InputTokens, e.OutputTokens, e.CacheReadTokens, e.CacheCreationTokens)
		totalCost += cost.Usd
		if e.IsError {
			errTurns++
		}
		if prevActivity == shared.ActivityTesting && e.Activity == shared.ActivityCoding {
			fixIter++
		}
		prevActivity = e.Activity

		t, err := time.Parse(time.RFC3339, e.TS)
		if err == nil {
			if !prevTime.IsZero() && t.Sub(prevTime) > coldRestartGap {
				coldRestartTurns++
				coldRestartTok += tok
			}
			prevTime = t
			tsSlice = append(tsSlice, tsEvent{t, tok})
		}
	}

	// context growth: compare avg tokens in first half vs second half of session
	var contextGrowth float64
	if len(tsSlice) >= bloatMinTurns {
		mid := len(tsSlice) / 2
		var earlySum, lateSum int
		for _, ts := range tsSlice[:mid] {
			earlySum += ts.tok
		}
		for _, ts := range tsSlice[mid:] {
			lateSum += ts.tok
		}
		earlyAvg := float64(earlySum) / float64(mid)
		lateAvg := float64(lateSum) / float64(len(tsSlice)-mid)
		if earlyAvg > 0 {
			contextGrowth = lateAvg / earlyAvg
		}
	}

	// duration
	var durationMin int
	if len(tsSlice) >= 2 {
		dur := tsSlice[len(tsSlice)-1].ts.Sub(tsSlice[0].ts)
		durationMin = int(dur.Minutes())
	}

	// dominant activity
	dominant := shared.ActivityExploration
	maxCount := 0
	for act, cnt := range actCounts {
		if cnt > maxCount {
			maxCount = cnt
			dominant = act
		}
	}

	avgCtx := 0
	if len(turns) > 0 {
		avgCtx = spendTok / len(turns)
	}

	return SessionStat{
		SessionID:         sid,
		Project:           project,
		Source:            source,
		Turns:             len(turns),
		SpendTokens:       spendTok,
		CostUsd:           totalCost,
		ErrorTurns:        errTurns,
		FixIterations:     fixIter,
		AvgContextTokens:  avgCtx,
		ContextGrowth:     contextGrowth,
		ColdRestartTurns:  coldRestartTurns,
		ColdRestartTokens: coldRestartTok,
		DurationMin:       durationMin,
		Dominant:          dominant,
	}
}
