package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// buildBinary compiles the token-monitor binary to a temp path and returns it.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "token-monitor")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(testRoot(), "cmd", "token-monitor")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build binary: %v\n%s", err, out)
	}
	return bin
}

func testRoot() string {
	// Walk up from cmd/token-monitor to module root.
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// mkFixtureClaudeLog writes a minimal claude-code JSONL session file.
func mkFixtureClaudeLog(t *testing.T, homeDir string, n int) {
	t.Helper()
	projDir := filepath.Join(homeDir, ".claude", "projects", "e2e-test")
	_ = os.MkdirAll(projDir, 0755)
	f, err := os.Create(filepath.Join(projDir, "session.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i := 0; i < n; i++ {
		enc.Encode(map[string]any{
			"uuid":      "evt" + string(rune('0'+i)),
			"sessionId": "sess1",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude-sonnet-4-5",
				"content": []map[string]any{
					{"type": "text", "text": "hello"},
				},
				"usage": map[string]any{
					"input_tokens":  100 * (i + 1),
					"output_tokens": 40 * (i + 1),
				},
			},
		})
	}
}

func TestE2E_CollectThenReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	bin := buildBinary(t)
	home := t.TempDir()
	dbDir := filepath.Join(home, ".token-monitor")
	_ = os.MkdirAll(dbDir, 0700)

	// Write fixture data
	mkFixtureClaudeLog(t, home, 3)

	env := append(os.Environ(), "HOME="+home)

	// collect
	collectCmd := exec.Command(bin, "collect")
	collectCmd.Env = env
	out, err := collectCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("collect: %v\n%s", err, out)
	}
	t.Logf("collect output: %s", out)

	// report
	reportCmd := exec.Command(bin, "report", "--days", "7")
	reportCmd.Env = env
	out, err = reportCmd.CombinedOutput()
	if len(out) == 0 {
		t.Error("report produced no output")
	}
	t.Logf("report output:\n%s", out)
}

func TestE2E_ReconcileRequiresProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	bin := buildBinary(t)
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".token-monitor"), 0700)

	env := append(os.Environ(), "HOME="+home)
	reconcileCmd := exec.Command(bin, "reconcile")
	reconcileCmd.Env = env
	out, err := reconcileCmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error when --provider missing, got none. output: %s", out)
	}
}

func TestE2E_HtmlOutputFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	bin := buildBinary(t)
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".token-monitor"), 0700)
	outFile := filepath.Join(home, "report.html")

	env := append(os.Environ(), "HOME="+home)
	htmlCmd := exec.Command(bin, "html", "--out", outFile)
	htmlCmd.Env = env
	out, err := htmlCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("html: %v\n%s", err, out)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if len(data) < 100 {
		t.Errorf("html output too small: %d bytes", len(data))
	}
}
