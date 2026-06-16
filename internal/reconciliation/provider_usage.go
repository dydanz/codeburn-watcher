package reconciliation

// ProviderUsage is one model's authoritative token counts from the billing API.
// Represents what the provider billed the org for (all machines, all users).
type ProviderUsage struct {
	Model               string
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
}

// TotalTokens returns the sum of all token types.
func (p ProviderUsage) TotalTokens() int {
	return p.InputTokens + p.OutputTokens + p.CacheReadTokens + p.CacheCreationTokens
}

// ProviderSpec describes how to connect to a provider's billing API.
// Adapter wiring lives in infrastructure/providers; this is pure config.
type ProviderSpec struct {
	Name       string
	KeyEnv     string // environment variable holding the API key
	URLEnv     string // optional environment variable overriding the default URL
	DefaultURL string
}
