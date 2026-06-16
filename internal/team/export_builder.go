package team

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/trust"
)

// ExportBuilder constructs a SignedExport from a collection of domain objects.
type ExportBuilder struct {
	Canonicalizer trust.CanonicalizationService
}

// Build signs the payload with the given identity and returns a SignedExport.
// username is the caller's human-readable name (from DeployConfig).
func (b ExportBuilder) Build(identity trust.SigningIdentity, username string, payload ExportPayload) (SignedExport, error) {
	canon, err := b.Canonicalizer.Canonicalize(payload)
	if err != nil {
		return SignedExport{}, fmt.Errorf("canonicalize payload: %w", err)
	}

	sig, err := identity.Sign(canon)
	if err != nil {
		return SignedExport{}, fmt.Errorf("sign payload: %w", err)
	}

	pemBytes, err := identity.PublicKeyPEM()
	if err != nil {
		return SignedExport{}, fmt.Errorf("marshal public key: %w", err)
	}

	return SignedExport{
		Version:     1,
		Username:    username,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		PublicKey:   base64.StdEncoding.EncodeToString(pemBytes),
		Signature:   base64.StdEncoding.EncodeToString([]byte(sig)),
		Payload:     payload,
	}, nil
}
