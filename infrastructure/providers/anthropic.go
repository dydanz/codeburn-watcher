package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/reconciliation"
)

// AnthropicUsageAdapter fetches token usage from Anthropic's usage report API.
// GET /v1/organizations/usage_report/messages
type AnthropicUsageAdapter struct {
	Client *http.Client
}

type anthropicResult struct {
	Object string `json:"object"`
	Data   []struct {
		Model              string `json:"model"`
		UncachedInputTokens int   `json:"uncached_input_tokens"`
		OutputTokens        int   `json:"output_tokens"`
		CacheReadInputTokens int  `json:"cache_read_input_tokens"`
		CacheCreation       struct {
			Ephemeral5m  int `json:"ephemeral_5m_input_tokens"`
			Ephemeral1h  int `json:"ephemeral_1h_input_tokens"`
		} `json:"cache_creation"`
	} `json:"data"`
	HasMore   bool   `json:"has_more"`
	NextPage  string `json:"next_page"`
}

// FetchUsage retrieves usage grouped by model for the given time range.
func (a AnthropicUsageAdapter) FetchUsage(ctx context.Context, baseURL, key string, startMs, endMs int64) ([]reconciliation.ProviderUsage, error) {
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	startAt := time.UnixMilli(startMs).UTC().Format("2006-01-02")
	endAt := time.UnixMilli(endMs).UTC().Format("2006-01-02")

	// aggregate by model across pages
	totals := make(map[string]*reconciliation.ProviderUsage)
	nextPage := ""

	for {
		url := fmt.Sprintf("%s/v1/organizations/usage_report/messages?starting_at=%s&ending_at=%s&bucket_width=1d&limit=31&group_by[]=model",
			baseURL, startAt, endAt)
		if nextPage != "" {
			url += "&next_page=" + nextPage
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("anthropic: build request: %w", err)
		}
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("anthropic: request: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, body)
		}

		var result anthropicResult
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("anthropic: parse response: %w", err)
		}

		for _, row := range result.Data {
			u := totals[row.Model]
			if u == nil {
				u = &reconciliation.ProviderUsage{Model: row.Model}
				totals[row.Model] = u
			}
			u.InputTokens += row.UncachedInputTokens
			u.OutputTokens += row.OutputTokens
			u.CacheReadTokens += row.CacheReadInputTokens
			u.CacheCreationTokens += row.CacheCreation.Ephemeral5m + row.CacheCreation.Ephemeral1h
		}

		if !result.HasMore {
			break
		}
		nextPage = result.NextPage
	}

	usages := make([]reconciliation.ProviderUsage, 0, len(totals))
	for _, u := range totals {
		usages = append(usages, *u)
	}
	return usages, nil
}
