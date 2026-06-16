package analytics

import "strings"

// premiumPrefixes are model name substrings that indicate a premium-tier model.
var premiumPrefixes = []string{
	"claude-opus", "gpt-4", "o1", "o3", "gemini-1.5-pro", "claude-3-opus",
	"claude-3-5-sonnet", "claude-sonnet-4", "claude-opus-4",
}

// PricingService computes costs from the static PRICES table.
// Pure — no I/O.
type PricingService struct {
	Prices []ModelPrice
}

// NewPricingService returns a PricingService loaded with the default PRICES table.
func NewPricingService() PricingService {
	return PricingService{Prices: PRICES}
}

// CostOf returns the estimated USD cost for a set of token counts.
func (p PricingService) CostOf(model string, input, output, cacheRead, cacheWrite int) Cost {
	m := strings.ToLower(model)
	for _, price := range p.Prices {
		if strings.HasPrefix(m, strings.ToLower(price.Prefix)) {
			usd := float64(input)*price.InputPerMToken/1_000_000 +
				float64(output)*price.OutputPerMToken/1_000_000 +
				float64(cacheRead)*price.CacheReadPerMToken/1_000_000 +
				float64(cacheWrite)*price.CacheWritePerMToken/1_000_000
			return Cost{Usd: usd}
		}
	}
	total := input + output + cacheRead + cacheWrite
	return Cost{Estimated: true, UnpricedTokens: total}
}

// IsPremiumModel returns true when the model is a premium/expensive tier.
func (p PricingService) IsPremiumModel(model string) bool {
	m := strings.ToLower(model)
	for _, prefix := range premiumPrefixes {
		if strings.Contains(m, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}
