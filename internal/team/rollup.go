package team

import (
	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// Rollup is a merged metrics snapshot for a named group.
type Rollup struct {
	Label    string
	Axis     string // "group" | "all"
	Members  []string
	Metrics  analytics.Metrics
	Persona  insights.Persona
	Dominant shared.Activity
}
