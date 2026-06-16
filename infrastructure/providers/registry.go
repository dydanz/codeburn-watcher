package providers

import (
	reconciliationports "github.com/dydanz/codeburn-watcher/internal/reconciliation/ports"
)

// ProviderSpec describes how to connect to a provider billing API.
type ProviderSpec struct {
	KeyEnv     string // env var holding the API key
	URLEnv     string // optional env var overriding the default URL
	DefaultURL string
	Adapter    reconciliationports.ProviderUsagePort
}

// ProviderRegistry maps provider name → ProviderSpec.
// Used by the reconcile command to dispatch.
var ProviderRegistry = map[string]ProviderSpec{
	"anthropic": {
		KeyEnv:     "ANTHROPIC_ADMIN_KEY",
		URLEnv:     "TOKEN_MONITOR_ANTHROPIC_URL",
		DefaultURL: "https://api.anthropic.com",
		Adapter:    AnthropicUsageAdapter{},
	},
	"openai": {
		KeyEnv:     "OPENAI_ADMIN_KEY",
		URLEnv:     "TOKEN_MONITOR_OPENAI_URL",
		DefaultURL: "https://api.openai.com",
		Adapter:    OpenAIUsageAdapter{},
	},
}
