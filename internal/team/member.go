package team

import "github.com/dydanz/codeburn-watcher/internal/trust"

// TeamMember is enrolled within a team's trust model.
// Identity: Username.
type TeamMember struct {
	Username            string
	EnrolledFingerprint trust.Fingerprint // "" = not enrolled (no keyring check)
}
