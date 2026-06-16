package trust

import (
	"bytes"
	"encoding/json"
)

// CanonicalizationService produces deterministic JSON for signing.
// Invariants (TRD §10, §13.5):
//   - Keys sorted alphabetically (encoding/json does this by default for structs)
//   - No HTML escaping (SetEscapeHTML(false))
//   - No whitespace
//   - Undefined/nil fields depend on struct tags — callers should use omitempty carefully
type CanonicalizationService struct{}

// Canonicalize marshals v to deterministic JSON bytes.
// v must be serialisable to JSON. Typically v is team.ExportPayload or similar.
func (CanonicalizationService) Canonicalize(v any) (CanonicalJSON, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "") // no whitespace
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Encode appends a trailing newline; trim it for a canonical byte sequence.
	return CanonicalJSON(bytes.TrimRight(buf.Bytes(), "\n")), nil
}
