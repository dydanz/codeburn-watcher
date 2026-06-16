package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dydanz/codeburn-watcher/internal/team"
)

const configFile = ".token-monitor/config.json"

// FileDeployConfig implements team.DeployConfigPort.
// Reads/writes ~/.token-monitor/config.json.
type FileDeployConfig struct{}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, configFile), nil
}

// Load reads the deploy config from disk. Returns a zero-value if the file doesn't exist.
func (FileDeployConfig) Load() (team.DeployConfig, error) {
	path, err := configPath()
	if err != nil {
		return team.DeployConfig{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return team.DeployConfig{}, nil
	}
	if err != nil {
		return team.DeployConfig{}, fmt.Errorf("read config: %w", err)
	}
	var cfg team.DeployConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return team.DeployConfig{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Save writes the deploy config to disk, creating the directory if needed.
func (FileDeployConfig) Save(cfg team.DeployConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
