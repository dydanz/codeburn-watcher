package ports

import (
	"context"

	"github.com/dydanz/codeburn-watcher/internal/team"
)

// ExportTransportPort handles sending and receiving exports and configs.
type ExportTransportPort interface {
	Push(ctx context.Context, filename string, body []byte, url string) error
	FetchConfig(ctx context.Context, url string) (string, error)
}

// DeployConfigPort persists the local deploy configuration.
type DeployConfigPort interface {
	Load() (team.DeployConfig, error)
	Save(config team.DeployConfig) error
}

// SchedulerPort installs/removes the OS-level recurring job.
type SchedulerPort interface {
	Install(binaryPath string, config team.DeployConfig) error
	Remove() error
}
