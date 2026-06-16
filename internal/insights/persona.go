package insights

import (
	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// Persona is the behavioral archetype assigned to a developer.
type Persona struct {
	ID                      string
	Label                   string
	Description             string
	GeneralRecommendations  []string
}

// PERSONAS is the ordered list evaluated by PersonaAssigner.
// First match wins (ordered threshold evaluation).
var PERSONAS = []Persona{
	{
		ID:          "firefighter",
		Label:       "Firefighter",
		Description: "High rework and retry — spending most time fixing failures.",
		GeneralRecommendations: []string{
			"Break tasks into smaller, verifiable steps.",
			"Add tests before making changes.",
			"Review error logs before retrying.",
		},
	},
	{
		ID:          "explorer",
		Label:       "Explorer",
		Description: "High exploration and thinking — lots of discovery, less delivery.",
		GeneralRecommendations: []string{
			"Set a time-box for exploration phases.",
			"Switch to a cheaper model for initial research.",
			"Commit to an approach before asking the agent to implement.",
		},
	},
	{
		ID:          "sprinter",
		Label:       "Sprinter",
		Description: "High coding share and low thinking — fast but may miss context.",
		GeneralRecommendations: []string{
			"Add a thinking/planning step before large edits.",
			"Check cache hit ratio — fast sessions often skip reuse.",
		},
	},
	{
		ID:          "surgeon",
		Label:       "Surgeon",
		Description: "Very targeted — high cache reuse, low bloat, surgical edits.",
		GeneralRecommendations: []string{
			"This is an efficient pattern. Maintain session continuity.",
			"Watch for cold-restart waste when resuming long sessions.",
		},
	},
	{
		ID:          "architect",
		Label:       "Architect",
		Description: "High thinking-to-code ratio — planning-heavy workflow.",
		GeneralRecommendations: []string{
			"Consider using a cheaper model for planning phases.",
			"Cache hit ratio matters most in long planning sessions.",
		},
	},
	{
		ID:          "balanced",
		Label:       "Balanced",
		Description: "No dominant pattern — evenly distributed across activities.",
		GeneralRecommendations: []string{
			"Review per-finding recommendations to find optimisation opportunities.",
		},
	},
}

// PersonaAssigner assigns a Persona based on Metrics.
// Ordered threshold evaluation — first match wins.
type PersonaAssigner struct{}

// Assign returns the first Persona whose criteria match the given Metrics.
func (PersonaAssigner) Assign(m analytics.Metrics) Persona {
	switch {
	case m.ReworkRatio > 0.25 || m.RetryShare > 0.20:
		return PERSONAS[0] // Firefighter
	case m.ThinkToCodeRatio > 0.60:
		return PERSONAS[4] // Architect
	case m.ByActivity[shared.ActivityExploration].Share > 0.40:
		return PERSONAS[1] // Explorer
	case m.ByActivity[shared.ActivityCoding].Share > 0.50 && m.ThinkToCodeRatio < 0.20:
		return PERSONAS[2] // Sprinter
	case m.CacheHitRatio > 0.60 && m.ContextBloatShare < 0.10:
		return PERSONAS[3] // Surgeon
	default:
		return PERSONAS[5] // Balanced
	}
}
