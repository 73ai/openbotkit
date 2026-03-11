package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
)

const (
	defaultMaxLength = 100000
	defaultFormat    = "markdown"
	maxResponseBody  = 10 << 20 // 10 MB hard cap on response body
)

func (w *WebSearch) Fetch(ctx context.Context, rawURL string, opts FetchOptions) (*FetchResult, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, errors.New("empty URL")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q: only http and https allowed", parsed.Scheme)
	}

	if opts.MaxLength <= 0 {
		opts.MaxLength = defaultMaxLength
	}
	if opts.Format == "" {
		opts.Format = defaultFormat
	}

	rawURL = normalizeGitHubURL(rawURL)

	client := w.fetchClient()
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	result := &FetchResult{
		URL:         rawURL,
		ContentType: contentType,
		StatusCode:  resp.StatusCode,
	}

	switch {
	case strings.Contains(contentType, "text/html"), strings.Contains(contentType, "application/xhtml"):
		result.Title = extractTitle(body)
		switch opts.Format {
		case "text":
			result.Content = extractText(body)
		default:
			md, err := htmltomarkdown.ConvertString(string(body))
			if err != nil {
				result.Content = extractText(body)
			} else {
				result.Content = md
			}
		}
	case strings.Contains(contentType, "application/json"):
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err != nil {
			result.Content = string(body)
		} else {
			result.Content = pretty.String()
		}
	default:
		result.Content = string(body)
	}

	if len(result.Content) > opts.MaxLength {
		result.Content = result.Content[:opts.MaxLength] + "\n\n[Content truncated at " + fmt.Sprintf("%d", opts.MaxLength) + " characters]"
		result.Truncated = true
	}

	return result, nil
}

func normalizeGitHubURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host != "github.com" {
		return raw
	}
	parts := strings.SplitN(u.Path, "/", 5) // /user/repo/blob/branch/path
	if len(parts) < 5 || parts[3] != "blob" {
		return raw
	}
	// Rewrite: /user/repo/blob/branch/path → /user/repo/branch/path
	u.Host = "raw.githubusercontent.com"
	u.Path = strings.Join(append(parts[:3], parts[4:]...), "/")
	return u.String()
}

func extractTitle(html []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(doc.Find("title").First().Text())
}

func extractText(html []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return string(html)
	}
	return strings.TrimSpace(doc.Find("body").Text())
}

func (w *WebSearch) fetchClient() *http.Client {
	transport := &http.Transport{}

	if !w.skipSSRF {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("dns lookup for %q: %w", host, err)
			}
			for _, ip := range ips {
				if isPrivateIP(ip.IP) {
					return nil, fmt.Errorf("blocked: %s resolves to private IP %s", host, ip.IP)
				}
			}
			// Pin to first resolved IP to prevent DNS rebinding (TOCTOU).
			return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		}
	}

	timeout := 30 * time.Second
	if w.cfg.WebSearch != nil {
		if w.cfg.WebSearch.Timeout != "" {
			if d, err := time.ParseDuration(w.cfg.WebSearch.Timeout); err == nil {
				timeout = d
			}
		}
		if w.cfg.WebSearch.Proxy != "" {
			if u, err := url.Parse(w.cfg.WebSearch.Proxy); err == nil {
				transport.Proxy = http.ProxyURL(u)
			}
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
