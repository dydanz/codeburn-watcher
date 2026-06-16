package adapters

import (
	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/antigravity"
	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/claudecode"
	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/codex"
	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/copilot"
	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/cursor"
	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/geminicli"
	"github.com/dydanz/codeburn-watcher/internal/collection"
)

// AllAdapters returns the ordered list of all AgentAdapters.
// Detection order matters: first adapter to find events wins precedence.
func AllAdapters() []collection.AgentAdapter {
	return []collection.AgentAdapter{
		claudecode.NewClaudeCodeAdapter(),
		geminicli.NewGeminiCliAdapter(),
		codex.NewCodexAdapter(),
		copilot.NewCopilotAdapter(),
		cursor.NewCursorAdapter(),
		antigravity.NewAntigravityAdapter(),
	}
}
