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
	yahooSearchURL = "https://search.yahoo.com/search"
	yahooNewsURL   = "https://news.search.yahoo.com/search"
)

type Yahoo struct {
	client  HTTPDoer
	baseURL string
	newsURL string
}

func NewYahoo(client HTTPDoer) *Yahoo {
	return &Yahoo{client: client, baseURL: yahooSearchURL, newsURL: yahooNewsURL}
}

func (y *Yahoo) Name() string  { return "yahoo" }
func (y *Yahoo) Priority() int { return 1 }

func (y *Yahoo) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	u, err := url.Parse(y.baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("p", query)
	page := opts.Page
	if page <= 1 {
		page = 1
	}
	q.Set("b", fmt.Sprintf("%d", (page-1)*7+1))
	if opts.TimeLimit != "" {
		q.Set("btf", opts.TimeLimit)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Engine: "yahoo", Code: resp.StatusCode}
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find("div.algo").Each(func(_ int, s *goquery.Selection) {
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
		snippet := strings.TrimSpace(s.Find("div.compText").Text())

		if title != "" && href != "" {
			results = append(results, Result{
				Title:   title,
				URL:     href,
				Snippet: snippet,
				Source:  "yahoo",
			})
		}
	})

	return results, nil
}

func unwrapYahooURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	path := u.Path
	ruIdx := strings.Index(path, "/RU=")
	if ruIdx == -1 {
		return raw
	}
	rest := path[ruIdx+4:]
	rkIdx := strings.Index(rest, "/RK=")
	if rkIdx == -1 {
		return raw
	}
	decoded, err := url.QueryUnescape(rest[:rkIdx])
	if err != nil {
		return raw
	}
	return decoded
}

func isYahooAdURL(href string) bool {
	return strings.Contains(href, "/aclick?") || strings.Contains(href, "bing.com")
}
