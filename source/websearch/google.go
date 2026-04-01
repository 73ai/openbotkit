package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const googleURL = "https://www.google.com/search"

var googleTimeLimits = map[string]string{
	"d": "qdr:d",
	"w": "qdr:w",
	"m": "qdr:m",
	"y": "qdr:y",
}

type Google struct {
	client  HTTPDoer
	baseURL string
}

func NewGoogle(client HTTPDoer) *Google {
	return &Google{client: client, baseURL: googleURL}
}

func (g *Google) Name() string  { return "google" }
func (g *Google) Priority() int { return 0 }

func (g *Google) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	u, err := url.Parse(g.baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	page := opts.Page
	if page <= 1 {
		page = 1
	}
	q.Set("start", fmt.Sprintf("%d", (page-1)*10))
	q.Set("hl", "en")
	if tbs, ok := googleTimeLimits[opts.TimeLimit]; ok {
		q.Set("tbs", tbs)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Engine: "google", Code: resp.StatusCode}
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find("div.g").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("a")
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		href = unwrapGoogleURL(href)
		title := strings.TrimSpace(s.Find("h3").Text())
		snippet := strings.TrimSpace(s.Find("div.VwiC3b").Text())

		if title != "" && href != "" {
			results = append(results, Result{
				Title:   title,
				URL:     href,
				Snippet: snippet,
				Source:  "google",
			})
		}
	})

	return results, nil
}

func unwrapGoogleURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Path == "/url" {
		if q := u.Query().Get("q"); q != "" {
			return q
		}
	}
	return raw
}
