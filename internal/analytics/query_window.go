package analytics

// QueryWindow defines the analysis scope: time range plus optional filters.
type QueryWindow struct {
	Days          int
	ProjectFilter string // "" = all projects
	SourceFilter  string // "" = all sources
}
