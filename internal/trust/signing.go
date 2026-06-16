package trust

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
)

// CanonicalJSON is the deterministic JSON bytes of an export payload.
// Produced by CanonicalizationService. This is the payload that gets signed.
type CanonicalJSON []byte

// Signature is the Ed25519 signature over CanonicalJSON.
type Signature []byte

// Base64 returns the base64-encoded signature string for embedding in exports.
func (s Signature) Base64() string {
	return base64.StdEncoding.EncodeToString(s)
}

// NewSignatureFromBase64 decodes a base64 signature string.
func NewSignatureFromBase64(b64 string) (Signature, error) {
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	return Signature(b), nil
}

// VerifyResult is the outcome of verifying a signed export.
type VerifyResult struct {
	Valid  bool
	Signer string // fingerprint of the signer, "" if invalid
	Error  string // human-readable failure reason
}

// Keyring maps username → trusted Fingerprint.
type Keyring map[string]Fingerprint

// VerificationService verifies signed exports.
type VerificationService struct{}

// Verify checks that sig is a valid Ed25519 signature of payload by pub.
// Pre-validates lengths to prevent panic from ed25519.Verify (TRD §13.5).
func (VerificationService) Verify(payload CanonicalJSON, sig Signature, pub ed25519.PublicKey) bool {
	if len(sig) != ed25519.SignatureSize {
		return false
	}
	if len(pub) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(pub, payload, sig)
}

// VerifyWithKeyring verifies a signed export and optionally checks the
// signer's fingerprint against the provided keyring.
func (v VerificationService) VerifyWithKeyring(
	payload CanonicalJSON,
	sig Signature,
	pub ed25519.PublicKey,
	keyring Keyring,
	username string,
) VerifyResult {
	if !v.Verify(payload, sig, pub) {
		return VerifyResult{Error: "signature verification failed"}
	}

	fp, err := FingerprintFromPublicKey(pub)
	if err != nil {
		return VerifyResult{Error: "failed to derive fingerprint: " + err.Error()}
	}

	if len(keyring) > 0 && username != "" {
		enrolled, ok := keyring[username]
		if !ok {
			return VerifyResult{
				Valid:  true, // signature valid but not in keyring
				Signer: string(fp),
				Error:  "username not in keyring",
			}
		}
		if enrolled != fp {
			return VerifyResult{
				Error: "fingerprint mismatch: export claims " + string(fp) + " but keyring has " + string(enrolled),
			}
		}
	}

	return VerifyResult{Valid: true, Signer: string(fp)}
}

// ErrEmptyPayload is returned when signing an empty payload.
var ErrEmptyPayload = errors.New("trust: cannot sign empty payload")
