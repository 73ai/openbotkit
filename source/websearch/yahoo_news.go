package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func (y *Yahoo) News(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	u, err := url.Parse(y.newsURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("p", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUserAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo news returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find("div.NewsArticle").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("a")
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		if isYahooAdURL(href) {
			return
		}

		href = unwrapYahooURL(href)
		title := strings.TrimSpace(link.Text())
		snippet := strings.TrimSpace(s.Find("p").Text())
		date := strings.TrimSpace(s.Find("span.s-time").Text())
		source := strings.TrimSpace(s.Find("span.s-source").Text())
		image, _ := s.Find("img").Attr("src")

		if title != "" && href != "" {
			r := Result{
				Title:   title,
				URL:     href,
				Snippet: snippet,
				Source:  "yahoo",
				Date:    date,
				Image:   image,
			}
			if source != "" {
				r.Snippet = source + " — " + r.Snippet
			}
			results = append(results, r)
		}
	})

	return results, nil
}
