package team

import (
	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/insights"
)

// SignedExport is one developer's signed, versioned export document.
// Identity: (Username, GeneratedAt). Immutable after signing.
// Invariant: Signature is valid over Payload canonical bytes.
type SignedExport struct {
	Version     int    // always 1
	Username    string
	GeneratedAt string // ISO8601 UTC
	PublicKey   string // base64 SPKI PEM for verification
	Signature   string // base64 Ed25519 signature

	Payload ExportPayload // the signed content
}

// ExportPayload is the data portion of a SignedExport.
// All fields here are within the signing boundary.
type ExportPayload struct {
	Days            int
	Overall         analytics.Metrics
	ByProject       map[string]analytics.Metrics
	Recommendations []insights.EnrichedRecommendation
	Trends          []insights.TrendRow
	ProjectMovers   []insights.ProjectMover
	FollowThrough   []insights.FollowThroughRecord
	Persona         insights.Persona
}
