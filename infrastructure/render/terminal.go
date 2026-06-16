package render

import (
	"fmt"
	"strings"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/deepanalysis"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/reconciliation"
	"github.com/dydanz/codeburn-watcher/internal/shared"
	"github.com/dydanz/codeburn-watcher/internal/team"
)

// TerminalRenderer produces ANSI terminal output. No port interface — called directly.
type TerminalRenderer struct{}

// RenderReport renders the main usage report.
func (TerminalRenderer) RenderReport(
	m analytics.Metrics,
	recs []insights.EnrichedRecommendation,
	trends []insights.TrendRow,
	movers []insights.ProjectMover,
	days int,
) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n╔══════════════════════════════════════╗\n"))
	sb.WriteString(fmt.Sprintf("║   Token Monitor — Last %d days        ║\n", days))
	sb.WriteString(fmt.Sprintf("╚══════════════════════════════════════╝\n\n"))

	sb.WriteString(fmt.Sprintf("  Events:   %s   Sessions: %d   Cost: %s\n",
		FmtTokens(m.Events), m.Sessions, FmtCost(m.CostUsd)))
	sb.WriteString(fmt.Sprintf("  Tokens:   %s\n\n", FmtTokens(m.SpendTokens)))

	// Ratios
	sb.WriteString("  ── Key Ratios ──────────────────────────\n")
	sb.WriteString(fmt.Sprintf("  Cache Hit    %s  %s\n", FmtPct(m.CacheHitRatio), Bar(m.CacheHitRatio, 20)))
	sb.WriteString(fmt.Sprintf("  Rework       %s  %s\n", FmtPct(m.ReworkRatio), Bar(m.ReworkRatio, 20)))
	sb.WriteString(fmt.Sprintf("  Think/Code   %s\n", FmtPct(m.ThinkToCodeRatio)))
	sb.WriteString(fmt.Sprintf("  Context Bloat %s\n", FmtPct(m.ContextBloatShare)))
	sb.WriteString(fmt.Sprintf("  Cold Restart  %s\n\n", FmtPct(m.ColdRestartShare)))

	// Activity breakdown
	if len(m.ByActivity) > 0 {
		sb.WriteString("  ── Activity Breakdown ──────────────────\n")
		for _, act := range shared.Activities {
			if ab, ok := m.ByActivity[act]; ok && ab.Tokens > 0 {
				sb.WriteString(fmt.Sprintf("  %-14s %s  %s\n",
					string(act), FmtTokens(ab.Tokens), Bar(ab.Share, 15)))
			}
		}
		sb.WriteString("\n")
	}

	// Recommendations
	if len(recs) > 0 {
		sb.WriteString("  ── Recommendations ─────────────────────\n")
		for i, r := range recs {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, r.Key, r.Label))
			sb.WriteString(fmt.Sprintf("     Potential savings: %s/mo\n", FmtCost(r.Evidence.SavingsUsd)))
		}
		sb.WriteString("\n")
	}

	// Project movers
	if len(movers) > 0 {
		sb.WriteString("  ── Rising Projects ─────────────────────\n")
		for _, m := range movers {
			sb.WriteString(fmt.Sprintf("  %-20s %s  %+d tokens\n", m.Project, m.Verdict, m.Delta))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderTrend renders the trend table.
func (TerminalRenderer) RenderTrend(trends []insights.TrendRow, days int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n  ── Trend (last %d days) ─────────────────\n", days))
	header := []string{"Metric", "Early", "Late", "Change", "Verdict"}
	cols := []int{20, 10, 10, 10, 12}
	rows := make([][]string, 0, len(trends))
	for _, t := range trends {
		rows = append(rows, []string{
			t.Label, FmtPct(t.Previous), FmtPct(t.Current),
			t.FmtDelta, string(t.Verdict),
		})
	}
	sb.WriteString(Table(header, rows, cols))
	return sb.String()
}

// RenderEnrichedRecs renders the full recommendations table.
func (TerminalRenderer) RenderEnrichedRecs(recs []insights.EnrichedRecommendation) string {
	var sb strings.Builder
	sb.WriteString("\n  ── All Recommendations ─────────────────\n")
	for i, r := range recs {
		sb.WriteString(fmt.Sprintf("  %2d. [%-20s] %-30s  %s/mo\n",
			i+1, r.Key, r.Label, FmtCost(r.Evidence.SavingsUsd)))
	}
	return sb.String()
}

// RenderTeamReport renders the team rollup table.
func (TerminalRenderer) RenderTeamReport(rollups []team.Rollup, days int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n  ── Team Report (last %d days) ───────────\n", days))
	header := []string{"Group", "Members", "Tokens", "Cost", "Persona"}
	cols := []int{18, 8, 10, 12, 14}
	rows := make([][]string, 0, len(rollups))
	for _, r := range rollups {
		rows = append(rows, []string{
			r.Label,
			fmt.Sprintf("%d", len(r.Members)),
			FmtTokens(r.Metrics.SpendTokens),
			FmtCost(r.Metrics.CostUsd),
			r.Persona.Label,
		})
	}
	sb.WriteString(Table(header, rows, cols))
	return sb.String()
}

// RenderAnalysis renders the deep analysis output.
func (TerminalRenderer) RenderAnalysis(
	deep deepanalysis.DeepAnalysis,
	m analytics.Metrics,
	recs []insights.EnrichedRecommendation,
	days int,
) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n  ── Deep Analysis (last %d days) ─────────\n\n", days))

	if len(deep.ExpensiveSessions) > 0 {
		sb.WriteString("  Top Sessions by Cost:\n")
		for _, s := range deep.ExpensiveSessions {
			sb.WriteString(fmt.Sprintf("    %-20s  %s  %d turns\n",
				s.Project, FmtTokens(s.SpendTokens), s.Turns))
		}
		sb.WriteString("\n")
	}

	if len(deep.FixLoopSessions) > 0 {
		sb.WriteString("  Fix-Loop Sessions (≥2 testing→coding):\n")
		for _, s := range deep.FixLoopSessions {
			sb.WriteString(fmt.Sprintf("    %-20s  %d fix iterations\n", s.Project, s.FixIterations))
		}
		sb.WriteString("\n")
	}

	if len(deep.ToolStats) > 0 {
		sb.WriteString("  Top Tool Error Rates:\n")
		header := []string{"Tool", "Turns", "Errors", "ErrorRate"}
		cols := []int{20, 8, 8, 12}
		rows := make([][]string, 0, len(deep.ToolStats))
		for _, t := range deep.ToolStats {
			if t.ErrorTurns == 0 {
				continue
			}
			rows = append(rows, []string{
				t.Tool,
				fmt.Sprintf("%d", t.Turns),
				fmt.Sprintf("%d", t.ErrorTurns),
				FmtPct(t.ErrorRate),
			})
		}
		sb.WriteString(Table(header, rows, cols))
	}
	return sb.String()
}

// RenderReconcile renders the reconciliation result.
func (TerminalRenderer) RenderReconcile(provider string, result reconciliation.ReconciliationResult, days int) string {
	var sb strings.Builder
	status := "✓ OK"
	if result.Breach {
		status = "✗ BREACH"
	}
	sb.WriteString(fmt.Sprintf("\n  ── Reconciliation: %s (last %d days) ── %s\n\n", provider, days, status))

	header := []string{"Model", "Local", "API", "Coverage", "Verdict"}
	cols := []int{28, 10, 10, 12, 20}
	rows := make([][]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		rows = append(rows, []string{
			row.Model,
			FmtTokens(row.LocalTokens),
			FmtTokens(row.ApiTokens),
			FmtPct(row.Coverage),
			string(row.Verdict),
		})
	}
	sb.WriteString(Table(header, rows, cols))
	return sb.String()
}
