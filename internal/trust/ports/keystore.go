package ports

import (
	"crypto/ed25519"

	"github.com/dydanz/codeburn-watcher/internal/trust"
)

// KeystorePort manages the on-disk Ed25519 keypair and keyring.
// Implemented by infrastructure/fs.FileKeystore.
type KeystorePort interface {
	// EnsureKeypair loads an existing keypair from dir, or generates and
	// persists a new one. Private key stored at mode 0600 (PKCS8 PEM),
	// public key at mode 0644 (SPKI PEM).
	EnsureKeypair(dir string) (ed25519.PrivateKey, ed25519.PublicKey, error)

	// LoadKeypair loads the existing keypair from dir.
	// Returns an error if the keypair does not exist.
	LoadKeypair(dir string) (ed25519.PrivateKey, ed25519.PublicKey, error)

	// LoadKeyring reads a JSON file mapping username → fingerprint.
	LoadKeyring(path string) (trust.Keyring, error)
}
