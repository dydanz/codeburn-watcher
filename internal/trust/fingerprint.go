package trust

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
)

// Fingerprint is the first 8 hex characters of SHA-256(DER(publicKey)).
// Serves as a stable, human-readable identity token for team enrollment.
type Fingerprint string

// ComputeFingerprint derives the fingerprint from a DER-encoded public key.
func ComputeFingerprint(pubDER []byte) Fingerprint {
	h := sha256.Sum256(pubDER)
	return Fingerprint(hex.EncodeToString(h[:])[:8])
}

// FingerprintFromPublicKey marshals the public key to DER and computes its fingerprint.
func FingerprintFromPublicKey(pub any) (Fingerprint, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	return ComputeFingerprint(der), nil
}
