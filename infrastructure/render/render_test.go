package render_test

import (
	"strings"
	"testing"

	"github.com/dydanz/codeburn-watcher/infrastructure/render"
	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/insights"
)

func TestFmtTokens(t *testing.T) {
	cases := []struct{ in int; want string }{
		{500, "500"},
		{1500, "1.5K"},
		{1_500_000, "1.5M"},
	}
	for _, c := range cases {
		if got := render.FmtTokens(c.in); got != c.want {
			t.Errorf("FmtTokens(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderReport_ContainsExpectedSections(t *testing.T) {
	r := render.TerminalRenderer{}
	m := analytics.Metrics{
		Events:      100,
		Sessions:    5,
		SpendTokens: 50000,
		CostUsd:     0.50,
		CacheHitRatio: 0.75,
	}
	out := r.RenderReport(m, nil, nil, nil, 7)
	if !strings.Contains(out, "Token Monitor") {
		t.Error("expected title in output")
	}
	if !strings.Contains(out, "Cache Hit") {
		t.Error("expected Cache Hit ratio in output")
	}
}

func TestRenderHtml_EscapesAmpersand(t *testing.T) {
	r := render.HtmlRenderer{}
	m := analytics.Metrics{Events: 1}
	// EnrichedRecommendation with & in label
	recs := []insights.EnrichedRecommendation{
		{Key: "cache_miss", Label: "Cache & Performance"},
	}
	out := r.RenderHtml(m, recs, nil, nil, 7)
	if strings.Contains(out, "&amp;") {
		t.Error("should not use &amp; — should use &#38; numeric entity")
	}
	if !strings.Contains(out, "&#38;") {
		t.Error("expected &#38; numeric entity for ampersand (TRD §13.5)")
	}
}

func TestBar(t *testing.T) {
	b := render.Bar(0.5, 10)
	if !strings.HasPrefix(b, "[") || !strings.HasSuffix(b, "]") {
		t.Errorf("Bar output malformed: %q", b)
	}
	runes := []rune(b)
	if len(runes) != 12 { // [ + 10 runes + ]
		t.Errorf("Bar rune count = %d, want 12", len(runes))
	}
}
