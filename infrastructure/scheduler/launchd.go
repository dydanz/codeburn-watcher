package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/dydanz/codeburn-watcher/internal/team"
)

const launchdLabel = "com.token-monitor.collect"

var plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>collect</string>
    </array>
    <key>StartInterval</key>
    <integer>600</integer>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`))

// LaunchdScheduler installs/removes a launchd job on macOS.
type LaunchdScheduler struct{}

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist"), nil
}

// Install writes the plist and calls launchctl load.
func (LaunchdScheduler) Install(binaryPath string, _ team.DeployConfig) error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir launch agents: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create plist: %w", err)
	}
	defer f.Close()
	if err := plistTmpl.Execute(f, struct{ Label, BinaryPath string }{launchdLabel, binaryPath}); err != nil {
		return fmt.Errorf("render plist: %w", err)
	}
	return exec.Command("launchctl", "load", path).Run()
}

// Remove unloads and removes the plist.
func (LaunchdScheduler) Remove() error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	return os.Remove(path)
}
