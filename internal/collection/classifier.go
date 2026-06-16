package collection

import (
	"regexp"
	"strings"

	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// tool name sets used by the 9-step classification chain.
var (
	writeTools = map[string]bool{
		"write_file": true, "edit_file": true, "create_file": true,
		"str_replace_editor": true, "str_replace_based_edit_tool": true,
		"multiedit": true, "insert_edit_into_file": true,
		"text_editor_20250429": true, "text_editor_20241022": true,
		"computer_20250124": true,
	}
	shellTools = map[string]bool{
		"bash": true, "execute_command": true, "run_command": true,
		"shell": true, "terminal": true, "execute_bash": true,
		"computer": true,
	}
	readTools = map[string]bool{
		"read_file": true, "view_file": true, "cat": true,
		"list_directory": true, "ls": true, "find_files": true,
		"glob": true, "grep": true, "search_files": true,
		"codebase_search": true, "file_search": true,
	}
	planTools = map[string]bool{
		"todowrite": true, "todoupdate": true, "task": true,
		"plan": true,
	}
	interactiveTools = map[string]bool{
		"web_search": true, "browser_action": true, "web_fetch": true,
		"mcp__": true, // prefix-matched below
	}

	testRE = regexp.MustCompile(`(?i)(test|spec|jest|pytest|go test|cargo test|rspec|mocha|vitest|bun test)`)
	shipRE = regexp.MustCompile(`(?i)(git (commit|push|tag)|npm (publish|pack)|cargo publish|deploy|release|docker (build|push)|kubectl apply)`)
)

// ActivityClassifier classifies a turn into one of the six Activities.
// Stateless and pure — no I/O. Instantiate once and reuse.
type ActivityClassifier struct{}

// norm lowercases and trims a tool name.
func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// Classify implements the 9-step priority chain from TRD §3.
//
//  1. No tools used → Thinking
//  2. Shell tool + test regex on any tool name → Testing
//  3. Shell tool + ship regex on any tool name → Shipping
//  4. Shell tool (no regex match) → Coding
//  5. Write/edit tool present → Coding
//  6. Plan tool present → Exploration
//  7. Read-only tools only → Exploration
//  8. Interactive/web tool → Conversation
//  9. Fallback → Exploration
func (ActivityClassifier) Classify(tools []string, _ string) shared.Activity {
	if len(tools) == 0 {
		return shared.ActivityThinking
	}

	normed := make([]string, len(tools))
	for i, t := range tools {
		normed[i] = norm(t)
	}

	hasShell, hasWrite, hasPlan, hasRead, hasInteractive := false, false, false, false, false
	joined := strings.Join(normed, " ")

	for _, t := range normed {
		switch {
		case shellTools[t]:
			hasShell = true
		case writeTools[t]:
			hasWrite = true
		case planTools[t]:
			hasPlan = true
		case readTools[t]:
			hasRead = true
		case interactiveTools[t], strings.HasPrefix(t, "mcp__"):
			hasInteractive = true
		}
	}

	switch {
	case hasShell && testRE.MatchString(joined):
		return shared.ActivityTesting
	case hasShell && shipRE.MatchString(joined):
		return shared.ActivityShipping
	case hasShell:
		return shared.ActivityCoding
	case hasWrite:
		return shared.ActivityCoding
	case hasPlan:
		return shared.ActivityExploration
	case hasRead && !hasInteractive:
		return shared.ActivityExploration
	case hasInteractive:
		return shared.ActivityConversation
	default:
		return shared.ActivityExploration
	}
}
