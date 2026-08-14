package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TavilyProvider struct {
	apiKey string
	client *http.Client
}

func NewTavilyProvider(apiKey string) TavilyProvider {
	return TavilyProvider{apiKey: apiKey, client: &http.Client{Timeout: 20 * time.Second}}
}

type tavilyRequest struct {
	APIKey        string `json:"api_key"`
	Query         string `json:"query"`
	SearchDepth   string `json:"search_depth"`
	MaxResults    int    `json:"max_results"`
	IncludeAnswer bool   `json:"include_answer"`
}

type tavilyResponse struct {
	Results []struct {
		URL     string  `json:"url"`
		Title   string  `json:"title"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

func (t TavilyProvider) Search(ctx context.Context, query string) ([]Result, error) {
	body, err := json.Marshal(tavilyRequest{
		APIKey:        t.apiKey,
		Query:         query,
		SearchDepth:   "basic",
		MaxResults:    8,
		IncludeAnswer: false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tavily error: %s", resp.Status)
	}

	var decoded tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(decoded.Results))
	for _, r := range decoded.Results {
		results = append(results, Result{URL: r.URL, Title: r.Title, Content: r.Content, Score: r.Score})
	}
	return results, nil
}
