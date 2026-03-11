package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const braveURL = "https://search.brave.com/search"

type Brave struct {
	client  HTTPDoer
	baseURL string
}

func NewBrave(client HTTPDoer) *Brave {
	return &Brave{client: client, baseURL: braveURL}
}

func (b *Brave) Name() string  { return "brave" }
func (b *Brave) Priority() int { return 1 }

func (b *Brave) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", b.baseURL+"?q="+url.QueryEscape(query), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find("div[data-type='web']").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("a")
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		title := strings.TrimSpace(link.Text())
		snippet := strings.TrimSpace(s.Find("div.snippet-content").Text())
		if snippet == "" {
			snippet = strings.TrimSpace(s.Find("p.snippet-description").Text())
		}

		if title != "" {
			results = append(results, Result{
				Title:   title,
				URL:     href,
				Snippet: snippet,
				Source:  "brave",
			})
		}
	})

	return results, nil
}
