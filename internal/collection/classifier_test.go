package collection_test

import (
	"testing"

	"github.com/dydanz/codeburn-watcher/internal/collection"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

func TestActivityClassifier(t *testing.T) {
	c := collection.ActivityClassifier{}

	tests := []struct {
		name     string
		tools    []string
		model    string
		want     shared.Activity
	}{
		{"no tools → thinking", nil, "", shared.ActivityThinking},
		{"empty tools → thinking", []string{}, "", shared.ActivityThinking},
		{"write tool → coding", []string{"write_file"}, "", shared.ActivityCoding},
		{"edit tool → coding", []string{"str_replace_editor"}, "", shared.ActivityCoding},
		{"bash → coding", []string{"bash"}, "", shared.ActivityCoding},
		{"bash + test word → testing", []string{"bash", "go_test"}, "", shared.ActivityTesting},
		{"bash + pytest → testing", []string{"bash", "pytest"}, "", shared.ActivityTesting},
		// tools slice may include command content alongside tool names
		{"bash + git push cmd → shipping", []string{"bash", "git push origin main"}, "", shared.ActivityShipping},
		{"bash + deploy cmd → shipping", []string{"bash", "deploy.sh"}, "", shared.ActivityShipping},
		{"plan tool → exploration", []string{"todowrite"}, "", shared.ActivityExploration},
		{"read only → exploration", []string{"read_file", "grep"}, "", shared.ActivityExploration},
		{"web search → conversation", []string{"web_search"}, "", shared.ActivityConversation},
		{"mcp tool → conversation", []string{"mcp__slack__send"}, "", shared.ActivityConversation},
		{"unknown tool → exploration", []string{"unknown_tool_xyz"}, "", shared.ActivityExploration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Classify(tt.tools, tt.model)
			if got != tt.want {
				t.Errorf("Classify(%v, %q) = %q, want %q", tt.tools, tt.model, got, tt.want)
			}
		})
	}
}

func TestAllActivitiesCoveredByClassifier(t *testing.T) {
	c := collection.ActivityClassifier{}
	seen := map[shared.Activity]bool{}

	cases := [][]string{
		nil,                          // thinking
		{"write_file"},               // coding
		{"bash"},                     // coding (different path)
		{"bash", "pytest"},           // testing
		{"bash", "git push origin"},   // shipping
		{"todowrite"},                // exploration
		{"read_file"},                // exploration (different path)
		{"web_search"},               // conversation
	}

	for _, tools := range cases {
		seen[c.Classify(tools, "")] = true
	}

	for _, a := range shared.Activities {
		if !seen[a] {
			t.Errorf("no test case produces activity %q", a)
		}
	}
}
