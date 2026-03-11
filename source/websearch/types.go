package websearch

import "github.com/priyanshujain/openbotkit/config"

type Config struct {
	WebSearch *config.WebSearchConfig
}

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}

type SearchResult struct {
	Query    string         `json:"query"`
	Results  []Result       `json:"results"`
	Metadata SearchMetadata `json:"metadata"`
}

type SearchMetadata struct {
	Backends     []string `json:"backends"`
	SearchTimeMs int64    `json:"search_time_ms"`
	TotalResults int      `json:"total_results"`
}

type FetchResult struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	StatusCode  int    `json:"status_code"`
	Truncated   bool   `json:"truncated"`
}

type SearchOptions struct {
	MaxResults int
	Backend    string
	TimeLimit  string
	Region     string
}

type FetchOptions struct {
	Format    string
	MaxLength int
}
