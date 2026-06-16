package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dydanz/codeburn-watcher/internal/team"
	"github.com/dydanz/codeburn-watcher/internal/trust"
)

// InitCommand drives the init sub-command.
type InitCommand struct {
	TeamConfigURL string // --team: URL to fetch team config from
}

// InitHandler bootstraps the local identity and schedule.
type InitHandler struct{ Deps AppDeps }

// Handle fetches team config, ensures keypair, prints fingerprint, runs first collect, installs scheduler.
func (h InitHandler) Handle(ctx context.Context, cmd InitCommand) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	keyDir := filepath.Join(home, ".token-monitor")

	// Fetch and save team config if URL provided
	if cmd.TeamConfigURL != "" {
		raw, err := h.Deps.Transport.FetchConfig(ctx, cmd.TeamConfigURL)
		if err != nil {
			return fmt.Errorf("fetch team config: %w", err)
		}
		cfg, err := team.TeamConfigParser{}.Parse(raw)
		if err != nil {
			return fmt.Errorf("parse team config: %w", err)
		}
		deployCfg := team.DeployConfig{
			Username: cfg.Username,
			PushURL:  cfg.PushURL,
			Members:  cfg.Members,
		}
		if err := h.Deps.DeployConfig.Save(deployCfg); err != nil {
			return fmt.Errorf("save deploy config: %w", err)
		}
	}

	// Ensure keypair
	priv, pub, err := h.Deps.Keystore.EnsureKeypair(keyDir)
	if err != nil {
		return fmt.Errorf("ensure keypair: %w", err)
	}
	fp, err := trust.FingerprintFromPublicKey(pub)
	if err != nil {
		return err
	}

	fmt.Printf("Identity ready. Share this fingerprint with your team lead:\n  %s\n\n", fp)
	_ = priv // priv stored on disk; not needed in memory after this

	// First collection run
	collector := CollectHandler{Adapters: h.Deps.Adapters, Writer: h.Deps.Writer}
	if _, err := collector.Handle(ctx, CollectCommand{}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: initial collect failed: %v\n", err)
	}

	// Install scheduler
	binaryPath, _ := os.Executable()
	deployCfg, _ := h.Deps.DeployConfig.Load()
	if err := h.Deps.Scheduler.Install(binaryPath, deployCfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: scheduler install failed: %v\n", err)
	}

	fmt.Println("Initialization complete.")
	return nil
}
