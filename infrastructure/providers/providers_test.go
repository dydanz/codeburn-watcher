package providers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dydanz/codeburn-watcher/infrastructure/providers"
)

func TestAnthropicUsageAdapter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{
					"model":                   "claude-sonnet-4-5",
					"uncached_input_tokens":   1000,
					"output_tokens":           500,
					"cache_read_input_tokens": 200,
					"cache_creation": map[string]int{
						"ephemeral_5m_input_tokens": 50,
						"ephemeral_1h_input_tokens": 25,
					},
				},
			},
			"has_more": false,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	adapter := providers.AnthropicUsageAdapter{Client: srv.Client()}
	now := time.Now().UnixMilli()
	usages, err := adapter.FetchUsage(context.Background(), srv.URL, "test-key", now-86400000, now)
	if err != nil {
		t.Fatalf("FetchUsage: %v", err)
	}
	if len(usages) != 1 {
		t.Fatalf("expected 1 usage, got %d", len(usages))
	}
	u := usages[0]
	if u.Model != "claude-sonnet-4-5" {
		t.Errorf("Model = %q", u.Model)
	}
	if u.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000", u.InputTokens)
	}
	if u.CacheCreationTokens != 75 {
		t.Errorf("CacheCreationTokens = %d, want 75 (50+25)", u.CacheCreationTokens)
	}
}

func TestOpenAIUsageAdapter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{
					"model":                "gpt-4o",
					"input_tokens":         1200, // includes 200 cached
					"input_cached_tokens":  200,
					"output_tokens":        600,
				},
			},
			"has_more": false,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	adapter := providers.OpenAIUsageAdapter{Client: srv.Client()}
	now := time.Now().UnixMilli()
	usages, err := adapter.FetchUsage(context.Background(), srv.URL, "test-key", now-86400000, now)
	if err != nil {
		t.Fatalf("FetchUsage: %v", err)
	}
	if len(usages) != 1 {
		t.Fatalf("expected 1 usage, got %d", len(usages))
	}
	u := usages[0]
	// input_tokens=1200, input_cached=200 → InputTokens=1000 (uncached)
	if u.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000 (1200-200 cached)", u.InputTokens)
	}
	if u.CacheReadTokens != 200 {
		t.Errorf("CacheReadTokens = %d, want 200", u.CacheReadTokens)
	}
	if u.OutputTokens != 600 {
		t.Errorf("OutputTokens = %d, want 600", u.OutputTokens)
	}
}

func TestProviderRegistry_Keys(t *testing.T) {
	if _, ok := providers.ProviderRegistry["anthropic"]; !ok {
		t.Error("anthropic not in registry")
	}
	if _, ok := providers.ProviderRegistry["openai"]; !ok {
		t.Error("openai not in registry")
	}
}
