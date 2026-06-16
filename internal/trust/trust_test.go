package trust_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/dydanz/codeburn-watcher/internal/trust"
)

func generateKeypair(t *testing.T) (ed25519.PrivateKey, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return priv, pub
}

func TestSigningIdentityRoundTrip(t *testing.T) {
	priv, pub := generateKeypair(t)
	id, err := trust.NewSigningIdentity(priv, pub)
	if err != nil {
		t.Fatalf("NewSigningIdentity: %v", err)
	}

	payload := trust.CanonicalJSON(`{"hello":"world"}`)
	sig, err := id.Sign(payload)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	v := trust.VerificationService{}
	if !v.Verify(payload, sig, pub) {
		t.Error("Verify returned false for a valid signature")
	}
}

func TestVerifyRejectsAlteredPayload(t *testing.T) {
	priv, pub := generateKeypair(t)
	id, _ := trust.NewSigningIdentity(priv, pub)

	sig, _ := id.Sign(trust.CanonicalJSON(`{"a":1}`))
	v := trust.VerificationService{}
	if v.Verify(trust.CanonicalJSON(`{"a":2}`), sig, pub) {
		t.Error("Verify should reject a signature over altered payload")
	}
}

func TestVerifyRejectsBadSigLength(t *testing.T) {
	_, pub := generateKeypair(t)
	v := trust.VerificationService{}
	// short sig must not panic
	if v.Verify(trust.CanonicalJSON(`{}`), trust.Signature([]byte{1, 2, 3}), pub) {
		t.Error("Verify should reject sig with wrong length")
	}
}

func TestVerifyRejectsBadPubKeyLength(t *testing.T) {
	priv, pub := generateKeypair(t)
	id, _ := trust.NewSigningIdentity(priv, pub)
	sig, _ := id.Sign(trust.CanonicalJSON(`{}`))

	v := trust.VerificationService{}
	// short pub must not panic
	if v.Verify(trust.CanonicalJSON(`{}`), sig, ed25519.PublicKey([]byte{1, 2, 3})) {
		t.Error("Verify should reject pub key with wrong length")
	}
}

func TestFingerprintLength(t *testing.T) {
	_, pub := generateKeypair(t)
	fp, err := trust.FingerprintFromPublicKey(pub)
	if err != nil {
		t.Fatalf("FingerprintFromPublicKey: %v", err)
	}
	if len(string(fp)) != 8 {
		t.Errorf("Fingerprint length = %d, want 8", len(string(fp)))
	}
}

func TestCanonicalizationNoHTMLEscape(t *testing.T) {
	c := trust.CanonicalizationService{}
	v := map[string]string{"url": "https://example.com?a=1&b=2"}
	canon, err := c.Canonicalize(v)
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	// SetEscapeHTML(false): '&' must remain as literal '&', not JSON-escaped to &.
	got := string(canon)
	if strings.Contains(got, "\\u0026") {
		t.Errorf("SetEscapeHTML not effective, got \\u0026 in output: %s", got)
	}
	if !strings.Contains(got, "&") {
		t.Errorf("Canonicalize should preserve literal &, got: %s", got)
	}
}

func TestSignatureBase64RoundTrip(t *testing.T) {
	priv, pub := generateKeypair(t)
	id, _ := trust.NewSigningIdentity(priv, pub)
	payload := trust.CanonicalJSON(`{"test":true}`)
	sig, _ := id.Sign(payload)

	b64 := sig.Base64()
	decoded, err := trust.NewSignatureFromBase64(b64)
	if err != nil {
		t.Fatalf("NewSignatureFromBase64: %v", err)
	}

	v := trust.VerificationService{}
	if !v.Verify(payload, decoded, pub) {
		t.Error("round-tripped signature should verify")
	}
}

