package application

import (
	"context"
	"fmt"
	"os"
)

// ScheduleCommand drives the schedule sub-command.
type ScheduleCommand struct {
	Remove bool // --remove: uninstall the scheduler entry
}

// ScheduleHandler installs or removes the background collection schedule.
type ScheduleHandler struct{ Deps AppDeps }

// Handle installs or removes the scheduler entry.
func (h ScheduleHandler) Handle(_ context.Context, cmd ScheduleCommand) error {
	if cmd.Remove {
		if err := h.Deps.Scheduler.Remove(); err != nil {
			return fmt.Errorf("remove schedule: %w", err)
		}
		fmt.Println("Schedule removed.")
		return nil
	}

	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	deployCfg, err := h.Deps.DeployConfig.Load()
	if err != nil {
		return fmt.Errorf("load deploy config: %w", err)
	}

	if err := h.Deps.Scheduler.Install(binaryPath, deployCfg); err != nil {
		return fmt.Errorf("install schedule: %w", err)
	}
	fmt.Println("Schedule installed.")
	return nil
}
