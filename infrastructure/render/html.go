package render

import (
	"fmt"
	"strings"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/shared"
	"github.com/dydanz/codeburn-watcher/internal/team"
)

// HtmlRenderer produces self-contained dark-theme HTML dashboards.
// Uses custom esc() with numeric HTML entities (not html.EscapeString — TRD §13.5).
type HtmlRenderer struct{}

// esc escapes HTML special characters using numeric entities.
// Does NOT use html.EscapeString to preserve exact character control.
func esc(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			sb.WriteString("&#38;")
		case '<':
			sb.WriteString("&#60;")
		case '>':
			sb.WriteString("&#62;")
		case '"':
			sb.WriteString("&#34;")
		case '\'':
			sb.WriteString("&#39;")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

const htmlHead = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Token Monitor</title>
<style>
body{background:#0d1117;color:#c9d1d9;font-family:monospace;margin:2rem}
h1,h2{color:#58a6ff}
table{border-collapse:collapse;width:100%;margin:1rem 0}
th{background:#161b22;text-align:left;padding:8px 12px;border-bottom:1px solid #30363d}
td{padding:6px 12px;border-bottom:1px solid #21262d}
.ok{color:#3fb950}.warn{color:#d29922}.breach{color:#f85149}
.bar{display:inline-block;background:#238636;height:8px;border-radius:2px}
</style>
</head>
<body>
`

const htmlFoot = `</body></html>`

// RenderHtml produces the main usage dashboard HTML.
func (HtmlRenderer) RenderHtml(
	m analytics.Metrics,
	recs []insights.EnrichedRecommendation,
	trends []insights.TrendRow,
	movers []insights.ProjectMover,
	days int,
) string {
	var sb strings.Builder
	sb.WriteString(htmlHead)
	sb.WriteString(fmt.Sprintf("<h1>Token Monitor — Last %d days</h1>\n", days))

	// Overview
	sb.WriteString("<h2>Overview</h2>\n<table><tr>")
	sb.WriteString(fmt.Sprintf("<th>Events</th><th>Sessions</th><th>Tokens</th><th>Cost</th></tr><tr>"))
	sb.WriteString(fmt.Sprintf("<td>%s</td><td>%d</td><td>%s</td><td>%s</td></tr></table>\n",
		esc(FmtTokens(m.Events)), m.Sessions, esc(FmtTokens(m.SpendTokens)), esc(FmtCost(m.CostUsd))))

	// Ratios
	sb.WriteString("<h2>Key Ratios</h2>\n<table><tr><th>Metric</th><th>Value</th><th>Bar</th></tr>\n")
	type kv struct{ label string; val float64 }
	ratios := []kv{
		{"Cache Hit", m.CacheHitRatio},
		{"Rework", m.ReworkRatio},
		{"Think/Code", m.ThinkToCodeRatio},
		{"Context Bloat", m.ContextBloatShare},
		{"Cold Restart", m.ColdRestartShare},
		{"Premium Waste", m.PremiumWasteShare},
		{"Retry Share", m.RetryShare},
	}
	for _, r := range ratios {
		pct := r.val * 100
		barW := int(pct)
		if barW > 100 {
			barW = 100
		}
		sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%.1f%%</td><td><span class="bar" style="width:%dpx"></span></td></tr>`+"\n",
			esc(r.label), pct, barW))
	}
	sb.WriteString("</table>\n")

	// Activity breakdown
	if len(m.ByActivity) > 0 {
		sb.WriteString("<h2>Activity Breakdown</h2>\n<table><tr><th>Activity</th><th>Tokens</th><th>Share</th></tr>\n")
		for _, act := range shared.Activities {
			if ab, ok := m.ByActivity[act]; ok && ab.Tokens > 0 {
				sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%.1f%%</td></tr>\n",
					esc(string(act)), esc(FmtTokens(ab.Tokens)), ab.Share*100))
			}
		}
		sb.WriteString("</table>\n")
	}

	// Recommendations
	if len(recs) > 0 {
		sb.WriteString("<h2>Recommendations</h2>\n<table><tr><th>#</th><th>Finding</th><th>Label</th><th>Savings/mo</th></tr>\n")
		for i, r := range recs {
			sb.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td><td>%s</td><td>%s</td></tr>\n",
				i+1, esc(string(r.Key)), esc(r.Label), esc(FmtCost(r.Evidence.SavingsUsd))))
		}
		sb.WriteString("</table>\n")
	}

	// Trends
	if len(trends) > 0 {
		sb.WriteString("<h2>Trends</h2>\n<table><tr><th>Metric</th><th>Previous</th><th>Current</th><th>Delta</th><th>Verdict</th></tr>\n")
		for _, t := range trends {
			cls := "ok"
			if t.Verdict == insights.TrendDown {
				cls = "warn"
			}
			sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td class="%s">%s</td></tr>`+"\n",
				esc(t.Label), esc(t.FmtCurrent), esc(FmtPct(t.Current)), esc(t.FmtDelta), cls, esc(string(t.Verdict))))
		}
		sb.WriteString("</table>\n")
	}

	sb.WriteString(htmlFoot)
	return sb.String()
}

// RenderTeamHtml produces the team rollup HTML dashboard.
func (HtmlRenderer) RenderTeamHtml(rollups []team.Rollup, days int) string {
	var sb strings.Builder
	sb.WriteString(htmlHead)
	sb.WriteString(fmt.Sprintf("<h1>Team Report — Last %d days</h1>\n", days))
	sb.WriteString("<table><tr><th>Group</th><th>Members</th><th>Tokens</th><th>Cost</th><th>Persona</th></tr>\n")
	for _, r := range rollups {
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			esc(r.Label), len(r.Members),
			esc(FmtTokens(r.Metrics.SpendTokens)),
			esc(FmtCost(r.Metrics.CostUsd)),
			esc(r.Persona.Label)))
	}
	sb.WriteString("</table>\n")
	sb.WriteString(htmlFoot)
	return sb.String()
}
