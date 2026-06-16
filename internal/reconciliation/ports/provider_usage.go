package ports

import (
	"context"

	"github.com/dydanz/codeburn-watcher/internal/reconciliation"
)

// ProviderUsagePort fetches authoritative token usage from a provider's billing API.
// baseURL and key are passed per-call so the adapter is stateless.
type ProviderUsagePort interface {
	FetchUsage(ctx context.Context, baseURL, key string, startMs, endMs int64) ([]reconciliation.ProviderUsage, error)
}
