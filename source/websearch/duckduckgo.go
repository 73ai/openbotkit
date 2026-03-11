package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	ddgURL      = "https://html.duckduckgo.com/html/"
	maxQueryLen = 499
)

type DuckDuckGo struct {
	client  *http.Client
	baseURL string
}

func NewDuckDuckGo(client *http.Client) *DuckDuckGo {
	return &DuckDuckGo{client: client, baseURL: ddgURL}
}

func (d *DuckDuckGo) Name() string  { return "duckduckgo" }
func (d *DuckDuckGo) Priority() int { return 1 }

func (d *DuckDuckGo) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	if runes := []rune(query); len(runes) > maxQueryLen {
		query = string(runes[:maxQueryLen])
	}

	form := url.Values{}
	form.Set("q", query)
	form.Set("s", "0")
	if opts.Region != "" {
		form.Set("kl", opts.Region)
	}
	if opts.TimeLimit != "" {
		form.Set("df", opts.TimeLimit)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUserAgent)
	req.Header.Set("Referer", "https://html.duckduckgo.com/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find("div.result").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("a.result__a")
		href, exists := link.Attr("href")
		if !exists {
			return
		}
		if strings.HasPrefix(href, "https://duckduckgo.com/y.js?") {
			return
		}

		title := strings.TrimSpace(link.Text())
		snippet := strings.TrimSpace(s.Find("a.result__snippet").Text())

		if title != "" && href != "" {
			results = append(results, Result{
				Title:   title,
				URL:     href,
				Snippet: snippet,
				Source:  "duckduckgo",
			})
		}
	})

	return results, nil
}
