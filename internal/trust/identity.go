package trust

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// SigningIdentity is the aggregate root for one machine's cryptographic identity.
// Invariant: privateKey and publicKey are a valid Ed25519 pair.
// The private key is unexported and never leaves this aggregate.
type SigningIdentity struct {
	privateKey  ed25519.PrivateKey
	publicKey   ed25519.PublicKey
	fingerprint Fingerprint
}

// NewSigningIdentity constructs a SigningIdentity after validating the keypair.
func NewSigningIdentity(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) (SigningIdentity, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return SigningIdentity{}, errors.New("trust: invalid private key length")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return SigningIdentity{}, errors.New("trust: invalid public key length")
	}
	fp, err := FingerprintFromPublicKey(publicKey)
	if err != nil {
		return SigningIdentity{}, err
	}
	return SigningIdentity{
		privateKey:  privateKey,
		publicKey:   publicKey,
		fingerprint: fp,
	}, nil
}

// Fingerprint returns the human-readable identity token for this machine.
func (id SigningIdentity) Fingerprint() Fingerprint {
	return id.fingerprint
}

// PublicKeyPEM returns the SPKI PEM representation of the public key.
// Safe to embed in exports — never reveals the private key.
func (id SigningIdentity) PublicKeyPEM() ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(id.publicKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// PublicKey returns the raw Ed25519 public key bytes.
func (id SigningIdentity) PublicKey() ed25519.PublicKey {
	return id.publicKey
}

// Sign signs payload with the private key and returns a Signature.
func (id SigningIdentity) Sign(payload CanonicalJSON) (Signature, error) {
	if len(payload) == 0 {
		return nil, errors.New("trust: cannot sign empty payload")
	}
	sig := ed25519.Sign(id.privateKey, payload)
	return Signature(sig), nil
}
