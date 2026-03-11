package websearch

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const yandexURL = "https://yandex.com/search/site/"

type Yandex struct {
	client  *http.Client
	baseURL string
}

func NewYandex(client *http.Client) *Yandex {
	return &Yandex{client: client, baseURL: yandexURL}
}

func (y *Yandex) Name() string  { return "yandex" }
func (y *Yandex) Priority() int { return 1 }

func (y *Yandex) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	u, err := url.Parse(y.baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("text", query)
	q.Set("web", "1")
	q.Set("searchid", fmt.Sprintf("%07d", rand.IntN(10000000)))
	q.Set("p", "0")
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
		return nil, fmt.Errorf("yandex returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find("li.serp-item").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("a")
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		title := strings.TrimSpace(link.Text())
		snippet := strings.TrimSpace(s.Find("div.text-container").Text())

		if title != "" {
			results = append(results, Result{
				Title:   title,
				URL:     href,
				Snippet: snippet,
				Source:  "yandex",
			})
		}
	})

	return results, nil
}
