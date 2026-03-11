package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const mojeekURL = "https://www.mojeek.com/search"

type Mojeek struct {
	client  *http.Client
	baseURL string
}

func NewMojeek(client *http.Client) *Mojeek {
	return &Mojeek{client: client, baseURL: mojeekURL}
}

func (m *Mojeek) Name() string  { return "mojeek" }
func (m *Mojeek) Priority() int { return 1 }

func (m *Mojeek) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	u, err := url.Parse(m.baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("s", "0")
	q.Set("safe", "1")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUserAgent)
	req.Header.Set("Accept", "text/html")

	if opts.Region != "" {
		arc, lb := regionToCookies(opts.Region)
		if arc != "" {
			req.AddCookie(&http.Cookie{Name: "arc", Value: arc})
		}
		if lb != "" {
			req.AddCookie(&http.Cookie{Name: "lb", Value: lb})
		}
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mojeek returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find("ul.results-standard li").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("a.ob")
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		title := strings.TrimSpace(link.Text())
		snippet := strings.TrimSpace(s.Find("p.s").Text())

		if title != "" {
			results = append(results, Result{
				Title:   title,
				URL:     href,
				Snippet: snippet,
				Source:  "mojeek",
			})
		}
	})

	return results, nil
}

func regionToCookies(region string) (arc, lb string) {
	parts := strings.SplitN(region, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return region, ""
}
