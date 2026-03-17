package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	wikiOpenSearchURL = "https://en.wikipedia.org/w/api.php"
	wikiMaxSnippet    = 300
	wikiUserAgent     = "openbotkit/0.1 (https://github.com/73ai/openbotkit)"
)

type Wikipedia struct {
	client    HTTPDoer
	searchURL string
}

func NewWikipedia(client HTTPDoer) *Wikipedia {
	return &Wikipedia{client: client, searchURL: wikiOpenSearchURL}
}

func (w *Wikipedia) Name() string  { return "wikipedia" }
func (w *Wikipedia) Priority() int { return 2 }

func (w *Wikipedia) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	// Step 1: OpenSearch to find article title.
	title, pageURL, err := w.openSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	if title == "" {
		return nil, nil
	}

	// Step 2: Get extract for the article.
	extract, err := w.getExtract(ctx, title)
	if err != nil {
		return nil, err
	}
	if extract == "" {
		return nil, nil
	}

	// Skip disambiguation pages.
	if strings.Contains(extract, "may refer to:") {
		return nil, nil
	}

	snippet := extract
	runes := []rune(snippet)
	if len(runes) > wikiMaxSnippet {
		snippet = string(runes[:wikiMaxSnippet]) + "..."
	}

	return []Result{{
		Title:   title,
		URL:     pageURL,
		Snippet: snippet,
		Source:  "wikipedia",
	}}, nil
}

func (w *Wikipedia) openSearch(ctx context.Context, query string) (title, pageURL string, err error) {
	u, _ := url.Parse(w.searchURL)
	q := u.Query()
	q.Set("action", "opensearch")
	q.Set("profile", "fuzzy")
	q.Set("limit", "1")
	q.Set("search", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", wikiUserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read opensearch response: %w", err)
	}

	// OpenSearch format: ["query", ["title1"], ["desc1"], ["url1"]]
	var result []json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("parse opensearch: %w", err)
	}
	if len(result) < 4 {
		return "", "", nil
	}

	var titles []string
	if err := json.Unmarshal(result[1], &titles); err != nil || len(titles) == 0 {
		return "", "", nil
	}

	var urls []string
	if err := json.Unmarshal(result[3], &urls); err != nil || len(urls) == 0 {
		return "", "", nil
	}

	return titles[0], urls[0], nil
}

func (w *Wikipedia) getExtract(ctx context.Context, title string) (string, error) {
	u, _ := url.Parse(w.searchURL)
	q := u.Query()
	q.Set("action", "query")
	q.Set("titles", title)
	q.Set("prop", "extracts")
	q.Set("exintro", "1")
	q.Set("explaintext", "1")
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", wikiUserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wikipedia extract returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read extract response: %w", err)
	}

	var extractResp struct {
		Query struct {
			Pages map[string]struct {
				Extract string `json:"extract"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.Unmarshal(body, &extractResp); err != nil {
		return "", fmt.Errorf("parse extract: %w", err)
	}

	for _, page := range extractResp.Query.Pages {
		return strings.TrimSpace(page.Extract), nil
	}
	return "", nil
}
