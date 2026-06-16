package team

import (
	"fmt"
	"strings"
)

// ExportFilename returns the canonical filename for a signed export.
// Format: <username>-<generatedAt>.json with colons and dots removed.
func ExportFilename(username, generatedAt string) string {
	safe := strings.NewReplacer(":", "", ".", "").Replace(generatedAt)
	return fmt.Sprintf("%s-%s.json", username, safe)
}
