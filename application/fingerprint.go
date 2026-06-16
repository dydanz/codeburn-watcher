package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dydanz/codeburn-watcher/internal/trust"
)

// FingerprintHandler prints the local signing identity fingerprint.
type FingerprintHandler struct{ Deps AppDeps }

// Handle loads the keypair and prints the fingerprint to stdout.
func (h FingerprintHandler) Handle(_ context.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	keyDir := filepath.Join(home, ".token-monitor")

	_, pub, err := h.Deps.Keystore.LoadKeypair(keyDir)
	if err != nil {
		return fmt.Errorf("load keypair from %s: %w — run 'token-monitor init' first", keyDir, err)
	}

	fp, err := trust.FingerprintFromPublicKey(pub)
	if err != nil {
		return fmt.Errorf("derive fingerprint: %w", err)
	}

	fmt.Printf("Fingerprint: %s\n", fp)
	return nil
}
