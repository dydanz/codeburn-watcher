package fs

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dydanz/codeburn-watcher/internal/trust"
)

const (
	privKeyFile = "signing-key.pem"
	pubKeyFile  = "signing-key-pub.pem"
)

// FileKeystore implements trust.KeystorePort using the local filesystem.
type FileKeystore struct{}

// EnsureKeypair loads an existing keypair or generates a new one at dir.
// Private key is written with mode 0600; public key with 0644.
func (FileKeystore) EnsureKeypair(dir string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	privPath := filepath.Join(dir, privKeyFile)
	if _, err := os.Stat(privPath); err == nil {
		return FileKeystore{}.LoadKeypair(dir)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private key: %w", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal public key: %w", err)
	}

	if err := writePEM(privPath, "PRIVATE KEY", privDER, 0600); err != nil {
		return nil, nil, err
	}
	if err := writePEM(filepath.Join(dir, pubKeyFile), "PUBLIC KEY", pubDER, 0644); err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// LoadKeypair reads and parses the existing PEM keypair from dir.
func (FileKeystore) LoadKeypair(dir string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	privData, err := os.ReadFile(filepath.Join(dir, privKeyFile))
	if err != nil {
		return nil, nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(privData)
	if block == nil {
		return nil, nil, fmt.Errorf("no PEM block in private key file")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse private key: %w", err)
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("key is not Ed25519")
	}
	return priv, priv.Public().(ed25519.PublicKey), nil
}

// LoadKeyring reads a JSON file mapping username → hex fingerprint into a Keyring.
func (FileKeystore) LoadKeyring(path string) (trust.Keyring, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keyring %s: %w", path, err)
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse keyring: %w", err)
	}
	ring := make(trust.Keyring, len(raw))
	for user, fp := range raw {
		ring[user] = trust.Fingerprint(fp)
	}
	return ring, nil
}

func writePEM(path, pemType string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: pemType, Bytes: der})
}
