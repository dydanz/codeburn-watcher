package scheduler

import (
	"runtime"

	"github.com/dydanz/codeburn-watcher/internal/team"
)

// SchedulerPort is the common interface (mirrors team/ports for convenience).
type SchedulerPort interface {
	Install(binaryPath string, config team.DeployConfig) error
	Remove() error
}

// unsupportedScheduler is returned on non-darwin, non-linux systems.
type unsupportedScheduler struct{}

func (unsupportedScheduler) Install(_ string, _ team.DeployConfig) error {
	return nil // no-op
}

func (unsupportedScheduler) Remove() error {
	return nil
}

// NewScheduler returns the platform-appropriate scheduler.
func NewScheduler() SchedulerPort {
	switch runtime.GOOS {
	case "darwin":
		return LaunchdScheduler{}
	case "linux":
		return CronScheduler{}
	default:
		return unsupportedScheduler{}
	}
}
