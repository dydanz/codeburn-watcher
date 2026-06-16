package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dydanz/codeburn-watcher/internal/team"
	"github.com/dydanz/codeburn-watcher/internal/trust"
)

// MergeCommand drives the merge sub-command.
type MergeCommand struct {
	Files    []string // positional: export JSON files
	Verify   bool     // --verify: verify all signatures
	KeysFile string   // --keys: path to keyring JSON
	HTML     bool     // --html: render HTML team report
	OutFile  string   // --out: write rendered output here
	Days     int
}

// MergeHandler merges signed exports from multiple team members.
type MergeHandler struct{ Deps AppDeps }

// Handle loads, verifies (optional), dedupes, and renders the merged team report.
func (h MergeHandler) Handle(_ context.Context, cmd MergeCommand) error {
	exports, err := loadExportFiles(cmd.Files)
	if err != nil {
		return err
	}

	if cmd.Verify {
		var keyring trust.Keyring
		if cmd.KeysFile != "" {
			keyring, err = h.Deps.Keystore.LoadKeyring(cmd.KeysFile)
			if err != nil {
				return err
			}
		}
		canon := trust.CanonicalizationService{}
		for _, e := range exports {
			canonBytes, err := canon.Canonicalize(e.Payload)
			if err != nil {
				return fmt.Errorf("canonicalize %s export: %w", e.Username, err)
			}
			sig, err := trust.NewSignatureFromBase64(e.Signature)
			if err != nil {
				return fmt.Errorf("decode signature for %s: %w", e.Username, err)
			}
			result := h.Deps.Verifier.VerifyWithKeyring(
				canonBytes, sig,
				nil, // pub key parsed from export — placeholder
				keyring, e.Username,
			)
			if !result.Valid {
				return fmt.Errorf("export from %s invalid: %s", e.Username, result.Error)
			}
		}
	}

	deduped := h.Deps.ExportMerger.Dedupe(exports)
	rollups := h.Deps.ExportMerger.Rollup(deduped, team.TeamConfig{})

	days := cmd.Days
	if days == 0 {
		days = 7
	}

	if cmd.HTML {
		html := h.Deps.HtmlRenderer.RenderTeamHtml(rollups, days)
		return writeText(html, cmd.OutFile)
	}
	output := h.Deps.Renderer.RenderTeamReport(rollups, days)
	if cmd.OutFile != "" {
		return writeText(output, cmd.OutFile)
	}
	fmt.Print(output)
	return nil
}

func loadExportFiles(paths []string) ([]team.SignedExport, error) {
	var exports []team.SignedExport
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		var e team.SignedExport
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		exports = append(exports, e)
	}
	return exports, nil
}
