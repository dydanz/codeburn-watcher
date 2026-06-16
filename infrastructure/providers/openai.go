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

// OpenAIUsageAdapter fetches token usage from OpenAI's usage API.
// GET /v1/organization/usage/completions
type OpenAIUsageAdapter struct {
	Client *http.Client
}

type openaiResult struct {
	Data []struct {
		Model               string `json:"model"`
		InputTokens         int    `json:"input_tokens"`
		InputCachedTokens   int    `json:"input_cached_tokens"`
		OutputTokens        int    `json:"output_tokens"`
	} `json:"data"`
	HasMore  bool   `json:"has_more"`
	NextPage string `json:"next_page"`
}

// FetchUsage retrieves usage grouped by model.
// OpenAI's input_tokens INCLUDES cached — subtract to get uncached input.
func (a OpenAIUsageAdapter) FetchUsage(ctx context.Context, baseURL, key string, startMs, endMs int64) ([]reconciliation.ProviderUsage, error) {
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	startSec := startMs / 1000
	endSec := endMs / 1000

	totals := make(map[string]*reconciliation.ProviderUsage)
	nextPage := ""

	for {
		url := fmt.Sprintf("%s/v1/organization/usage/completions?start_time=%d&end_time=%d&bucket_width=1d&group_by=model&limit=31",
			baseURL, startSec, endSec)
		if nextPage != "" {
			url += "&next_page=" + nextPage
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("openai: build request: %w", err)
		}
		req.Header.Set("authorization", "Bearer "+key)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("openai: request: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("openai: status %d: %s", resp.StatusCode, body)
		}

		var result openaiResult
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("openai: parse response: %w", err)
		}

		for _, row := range result.Data {
			u := totals[row.Model]
			if u == nil {
				u = &reconciliation.ProviderUsage{Model: row.Model}
				totals[row.Model] = u
			}
			// input_tokens includes cached — subtract to get uncached input
			u.InputTokens += row.InputTokens - row.InputCachedTokens
			u.CacheReadTokens += row.InputCachedTokens
			u.OutputTokens += row.OutputTokens
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
