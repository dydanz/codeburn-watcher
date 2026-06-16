// Package ports re-exports the FollowThroughRepository interface for
// infrastructure packages that implement it.
// The interface is defined in the parent insights package to avoid import cycles.
package ports

import "github.com/dydanz/codeburn-watcher/internal/insights"

// FollowThroughRepository is the outbound port for persisting follow-through records.
// Defined in the insights package; re-exported here for discoverability.
type FollowThroughRepository = insights.FollowThroughRepository
