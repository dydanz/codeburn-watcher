package scheduler

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dydanz/codeburn-watcher/internal/team"
)

const cronMarker = "# TOKEN_MONITOR"

// CronScheduler installs/removes a cron job on Linux.
type CronScheduler struct{}

// Install adds a cron entry to run "collect" every 10 minutes.
func (CronScheduler) Install(binaryPath string, _ team.DeployConfig) error {
	existing, err := currentCrontab()
	if err != nil {
		existing = ""
	}
	entry := fmt.Sprintf("*/10 * * * * %s collect %s", binaryPath, cronMarker)
	if strings.Contains(existing, cronMarker) {
		return nil // already installed
	}
	return setCrontab(strings.TrimRight(existing, "\n") + "\n" + entry + "\n")
}

// Remove removes the TOKEN_MONITOR cron entry.
func (CronScheduler) Remove() error {
	existing, err := currentCrontab()
	if err != nil {
		return nil
	}
	var filtered []string
	for line := range strings.SplitSeq(existing, "\n") {
		if !strings.Contains(line, cronMarker) {
			filtered = append(filtered, line)
		}
	}
	return setCrontab(strings.Join(filtered, "\n"))
}

func currentCrontab() (string, error) {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func setCrontab(content string) error {
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = bytes.NewBufferString(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("crontab -: %w: %s", err, out)
	}
	return nil
}
