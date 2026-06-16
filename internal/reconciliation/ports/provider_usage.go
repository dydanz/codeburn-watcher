package ports

import (
	"context"

	"github.com/dydanz/codeburn-watcher/internal/reconciliation"
)

// ProviderUsagePort fetches authoritative token usage from a provider's billing API.
type ProviderUsagePort interface {
	FetchUsage(ctx context.Context, days int) ([]reconciliation.ProviderUsage, error)
}
