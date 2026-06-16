package analytics

import (
	"strings"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/shared"
)

const (
	bloatMinTurns   = 10  // minimum turns in a session to check for context bloat
	bloatGrowthRatio = 2.0 // late/early context ratio that flags a session as bloated
	coldRestartGap  = 5 * time.Minute
)

// sessionState accumulates per-session data in pass 1.
type sessionState struct {
	turns               int
	spendTokens         int
	inputTokens         int
	cacheReadTokens     int
	cacheCreationTokens int
	outputTokens        int
	errorTurns          int
	retryTokens         int
	coldRestartTokens   int
	byActivity          map[shared.Activity]int
	contextPerTurn      []int // input + cacheRead per turn, in order
	lastTS              time.Time
	hasError            bool
	lastTool            string
}

// MetricsEngine computes the full Metrics struct from a slice of StoredEvents.
// Two-pass: pass 1 aggregates per-session state; pass 2 derives ratios.
// Pure — no I/O.
type MetricsEngine struct {
	Pricing PricingService
}

// NewMetricsEngine returns a MetricsEngine with the default pricing table.
func NewMetricsEngine(pricing PricingService) MetricsEngine {
	return MetricsEngine{Pricing: pricing}
}

// Compute runs the two-pass algorithm and returns a Metrics snapshot.
func (e MetricsEngine) Compute(events []shared.StoredEvent) Metrics {
	if len(events) == 0 {
		return Metrics{
			ByActivity: map[shared.Activity]ActivityBreakdown{},
			ByModel:    map[string]ModelBreakdown{},
		}
	}

	// ── Pass 1: accumulate per-session and per-model/activity totals ──────────

	sessions := map[string]*sessionState{}
	sessionOrder := []string{} // preserve first-seen order

	byActivity := map[shared.Activity]int{}
	byActivityCost := map[shared.Activity]float64{}
	byModel := map[string]int{}
	byModelCost := map[string]float64{}
	modelOrder := []string{}

	var (
		totalEvents             int
		totalInputTokens        int
		totalOutputTokens       int
		totalCacheReadTokens    int
		totalCacheCreationTokens int
		totalCostUsd            float64
		costEstimated           bool
		costUnpricedTokens      int
	)

	for _, ev := range events {
		totalEvents++

		spend := ev.InputTokens + ev.OutputTokens
		totalInputTokens += ev.InputTokens
		totalOutputTokens += ev.OutputTokens
		totalCacheReadTokens += ev.CacheReadTokens
		totalCacheCreationTokens += ev.CacheCreationTokens

		cost := e.Pricing.CostOf(ev.Model, ev.InputTokens, ev.OutputTokens, ev.CacheReadTokens, ev.CacheCreationTokens)
		totalCostUsd += cost.Usd
		if cost.Estimated {
			costEstimated = true
			costUnpricedTokens += cost.UnpricedTokens
		}

		byActivity[ev.Activity] += spend
		byActivityCost[ev.Activity] += cost.Usd

		if _, seen := byModel[ev.Model]; !seen {
			modelOrder = append(modelOrder, ev.Model)
		}
		byModel[ev.Model] += spend
		byModelCost[ev.Model] += cost.Usd

		// session-level accumulation
		ss, exists := sessions[ev.SessionID]
		if !exists {
			ss = &sessionState{byActivity: map[shared.Activity]int{}}
			sessions[ev.SessionID] = ss
			sessionOrder = append(sessionOrder, ev.SessionID)
		}

		// cold restart: gap > 5 min since last turn in same session
		ts, _ := time.Parse(time.RFC3339, strings.TrimSuffix(ev.TS, "Z")+"Z")
		if !ss.lastTS.IsZero() && ts.Sub(ss.lastTS) > coldRestartGap {
			ss.coldRestartTokens += ev.InputTokens
		}
		ss.lastTS = ts

		ss.turns++
		ss.spendTokens += spend
		ss.inputTokens += ev.InputTokens
		ss.outputTokens += ev.OutputTokens
		ss.cacheReadTokens += ev.CacheReadTokens
		ss.cacheCreationTokens += ev.CacheCreationTokens
		ss.byActivity[ev.Activity] += spend
		ss.contextPerTurn = append(ss.contextPerTurn, ev.InputTokens+ev.CacheReadTokens)

		if ev.IsError {
			ss.errorTurns++
			ss.hasError = true
		}

		// rework: error turn followed by another turn = retry tokens
		if ss.hasError && !ev.IsError {
			ss.retryTokens += spend
			ss.hasError = false
		}
	}

	// ── Pass 2: derive session-level ratios and aggregate metrics ─────────────

	var (
		totalSpendTokens    int
		totalRetryTokens    int
		totalColdRestart    int
		totalBloatSessions  int
		totalLongSessions   int
		totalPremiumWaste   int
	)

	for _, ss := range sessions {
		totalSpendTokens += ss.spendTokens
		totalRetryTokens += ss.retryTokens
		totalColdRestart += ss.coldRestartTokens

		// context bloat: session has ≥ bloatMinTurns and late-half avg ≥ 2× early-half avg
		if ss.turns >= bloatMinTurns {
			totalLongSessions++
			mid := len(ss.contextPerTurn) / 2
			early := avgInt(ss.contextPerTurn[:mid])
			late := avgInt(ss.contextPerTurn[mid:])
			if early > 0 && late/early >= bloatGrowthRatio {
				totalBloatSessions++
			}
		}

		// premium waste: premium model tokens on exploration or conversation
		for _, ev := range events {
			if ev.SessionID == "" {
				continue
			}
			if e.Pricing.IsPremiumModel(ev.Model) &&
				(ev.Activity == shared.ActivityExploration || ev.Activity == shared.ActivityConversation) {
				totalPremiumWaste += ev.InputTokens + ev.OutputTokens
			}
		}
	}

	// build totals for ratio denominators
	totalAllTokens := totalInputTokens + totalCacheReadTokens + totalCacheCreationTokens
	cacheHitRatio := safeDiv(float64(totalCacheReadTokens), float64(totalAllTokens))
	reworkRatio := safeDiv(float64(totalRetryTokens), float64(totalSpendTokens))
	coldRestartShare := safeDiv(float64(totalColdRestart), float64(totalInputTokens))
	premiumWasteShare := safeDiv(float64(totalPremiumWaste), float64(totalSpendTokens))
	retryShare := safeDiv(float64(totalRetryTokens), float64(totalSpendTokens))
	contextBloatShare := safeDiv(float64(totalBloatSessions), float64(totalLongSessions))

	thinkTokens := byActivity[shared.ActivityThinking]
	codeTokens := byActivity[shared.ActivityCoding]
	thinkToCodeRatio := safeDiv(float64(thinkTokens), float64(thinkTokens+codeTokens))

	// build breakdowns
	actBreakdowns := make(map[shared.Activity]ActivityBreakdown, len(shared.Activities))
	for _, a := range shared.Activities {
		tokens := byActivity[a]
		actBreakdowns[a] = ActivityBreakdown{
			Activity: a,
			Tokens:   tokens,
			Share:    safeDiv(float64(tokens), float64(totalSpendTokens)),
			CostUsd:  byActivityCost[a],
		}
	}

	modelBreakdowns := make(map[string]ModelBreakdown, len(modelOrder))
	for _, m := range modelOrder {
		tokens := byModel[m]
		modelBreakdowns[m] = ModelBreakdown{
			Model:   m,
			Tokens:  tokens,
			Share:   safeDiv(float64(tokens), float64(totalSpendTokens)),
			CostUsd: byModelCost[m],
		}
	}

	return Metrics{
		Events:              totalEvents,
		Sessions:            len(sessions),
		SpendTokens:         totalSpendTokens,
		CostUsd:             totalCostUsd,
		CostEstimated:       costEstimated,
		CostUnpricedTokens:  costUnpricedTokens,
		CacheHitRatio:       cacheHitRatio,
		ReworkRatio:         reworkRatio,
		ThinkToCodeRatio:    thinkToCodeRatio,
		ContextBloatShare:   contextBloatShare,
		ColdRestartShare:    coldRestartShare,
		PremiumWasteShare:   premiumWasteShare,
		RetryShare:          retryShare,
		ByActivity:          actBreakdowns,
		ByModel:             modelBreakdowns,
		CacheReadTokens:     totalCacheReadTokens,
		CacheCreationTokens: totalCacheCreationTokens,
		RetryTokens:         totalRetryTokens,
		ColdRestartTokens:   totalColdRestart,
		BloatSessions:       totalBloatSessions,
		TotalLongSessions:   totalLongSessions,
	}
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func avgInt(s []int) float64 {
	if len(s) == 0 {
		return 0
	}
	sum := 0
	for _, v := range s {
		sum += v
	}
	return float64(sum) / float64(len(s))
}
