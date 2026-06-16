package fs_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	fsinfra "github.com/dydanz/codeburn-watcher/infrastructure/fs"
	"github.com/dydanz/codeburn-watcher/internal/team"
	"github.com/dydanz/codeburn-watcher/internal/trust"
)

func TestEnsureKeypair_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	ks := fsinfra.FileKeystore{}

	priv, pub, err := ks.EnsureKeypair(dir)
	if err != nil {
		t.Fatalf("EnsureKeypair: %v", err)
	}
	if len(priv) == 0 || len(pub) == 0 {
		t.Error("expected non-empty keys")
	}
	if _, err := os.Stat(filepath.Join(dir, "signing-key.pem")); err != nil {
		t.Error("private key file not created")
	}
}

func TestEnsureKeypair_LoadsExisting(t *testing.T) {
	dir := t.TempDir()
	ks := fsinfra.FileKeystore{}

	priv1, pub1, err := ks.EnsureKeypair(dir)
	if err != nil {
		t.Fatalf("first EnsureKeypair: %v", err)
	}
	priv2, pub2, err := ks.EnsureKeypair(dir)
	if err != nil {
		t.Fatalf("second EnsureKeypair: %v", err)
	}
	if string(priv1) != string(priv2) || string(pub1) != string(pub2) {
		t.Error("second call should load existing keypair, not regenerate")
	}
}

func TestLoadKeyring(t *testing.T) {
	dir := t.TempDir()
	ring := map[string]string{"alice": "aabbccdd"}
	data, _ := json.Marshal(ring)
	path := filepath.Join(dir, "keyring.json")
	_ = os.WriteFile(path, data, 0644)

	ks := fsinfra.FileKeystore{}
	loaded, err := ks.LoadKeyring(path)
	if err != nil {
		t.Fatalf("LoadKeyring: %v", err)
	}
	if loaded["alice"] != trust.Fingerprint("aabbccdd") {
		t.Errorf("alice fingerprint = %q, want aabbccdd", loaded["alice"])
	}
}

func TestFileDeployConfig_SaveAndLoad(t *testing.T) {
	// Override home dir by writing directly to a temp path
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows compat

	cfg := fsinfra.FileDeployConfig{}
	original := team.DeployConfig{
		Username: "alice",
		PushURL:  "https://example.com",
		Members:  map[string][]string{"backend": {"alice", "bob"}},
	}
	if err := cfg.Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := cfg.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Username != "alice" {
		t.Errorf("Username = %q, want alice", loaded.Username)
	}
}
