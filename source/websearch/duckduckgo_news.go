package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
)

var vqdRe = regexp.MustCompile(`vqd="([^"]+)"`)

func (d *DuckDuckGo) News(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	if runes := []rune(query); len(runes) > maxQueryLen {
		query = string(runes[:maxQueryLen])
	}

	vqd, err := d.fetchVQD(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fetch vqd: %w", err)
	}

	u, err := url.Parse(d.newsURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("vqd", vqd)
	q.Set("o", "json")
	q.Set("noamp", "1")
	if opts.Region != "" {
		q.Set("l", opts.Region)
	}
	if opts.TimeLimit != "" {
		q.Set("df", opts.TimeLimit)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo news returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("read news response: %w", err)
	}

	var newsResp struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Excerpt string `json:"excerpt"`
			Source  string `json:"source"`
			Date    int64  `json:"date"`
			Image   string `json:"image"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &newsResp); err != nil {
		return nil, fmt.Errorf("parse news json: %w", err)
	}

	var results []Result
	for _, r := range newsResp.Results {
		if r.URL == "" {
			continue
		}
		result := Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Excerpt,
			Source:  "duckduckgo",
			Image:   r.Image,
		}
		if r.Date > 0 {
			result.Date = fmt.Sprintf("%d", r.Date)
		}
		results = append(results, result)
	}

	return results, nil
}

func (d *DuckDuckGo) fetchVQD(ctx context.Context, query string) (string, error) {
	u := d.vqdURL + "?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", chromeUserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return "", fmt.Errorf("read vqd page: %w", err)
	}

	matches := vqdRe.FindSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("vqd token not found in response")
	}

	return string(matches[1]), nil
}
