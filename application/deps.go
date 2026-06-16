package application

import (
	"github.com/dydanz/codeburn-watcher/infrastructure/render"
	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/collection"
	"github.com/dydanz/codeburn-watcher/internal/deepanalysis"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/team"
	"github.com/dydanz/codeburn-watcher/internal/trust"
	"github.com/dydanz/codeburn-watcher/internal/trust/ports"
	sharedports "github.com/dydanz/codeburn-watcher/internal/shared/ports"
	teamports "github.com/dydanz/codeburn-watcher/internal/team/ports"
	schedulerinfra "github.com/dydanz/codeburn-watcher/infrastructure/scheduler"
)

// AppDeps holds all application-layer dependencies wired together.
type AppDeps struct {
	// Ports
	Reader       sharedports.EventReader
	Writer       sharedports.EventWriter
	Keystore     ports.KeystorePort
	Transport    teamports.ExportTransportPort
	DeployConfig teamports.DeployConfigPort
	Scheduler    schedulerinfra.SchedulerPort

	// Domain services
	Adapters     []collection.AgentAdapter
	Classifier   collection.ActivityClassifier
	Pricing      analytics.PricingService
	MetricsEngine analytics.MetricsEngine
	Recommender  insights.RecommendationEngine
	Trends       insights.TrendAnalyzer
	FollowThrough insights.FollowThroughService
	Persona      insights.PersonaAssigner
	ExportBuilder team.ExportBuilder
	ExportMerger  team.ExportMerger
	Sessions     deepanalysis.SessionAnalyzer
	Tools        deepanalysis.ToolAnalyzer
	Pipeline     deepanalysis.LlmPipeline
	Verifier     trust.VerificationService
	Identity     trust.SigningIdentity
	Username     string // from DeployConfig

	// Renderers
	Renderer     render.TerminalRenderer
	HtmlRenderer render.HtmlRenderer
}
