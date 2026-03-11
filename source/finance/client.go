package finance

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

const (
	cookieURL = "https://fc.yahoo.com"
	crumbURL  = "https://query2.finance.yahoo.com/v1/test/getcrumb"
	quoteURL  = "https://query1.finance.yahoo.com/v7/finance/quote"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

type Client struct {
	mu     sync.Mutex
	http   *http.Client
	crumb  string
	inited bool
}

func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		http: &http.Client{
			Transport: newChromeTransport(),
			Jar:       jar,
		},
	}
}

func (c *Client) Quote(ctx context.Context, symbols ...string) ([]Quote, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.inited {
		if err := c.initSession(ctx); err != nil {
			return nil, fmt.Errorf("init session: %w", err)
		}
	}

	quotes, err := c.fetchQuotes(ctx, symbols)
	if err == errUnauthorized {
		if err := c.initSession(ctx); err != nil {
			return nil, fmt.Errorf("refresh session: %w", err)
		}
		quotes, err = c.fetchQuotes(ctx, symbols)
	}
	return quotes, err
}

var errUnauthorized = fmt.Errorf("unauthorized")

func (c *Client) initSession(ctx context.Context) error {
	// Step 1: Get cookies from fc.yahoo.com (returns 404 body but sets A3 cookie).
	req, err := http.NewRequestWithContext(ctx, "GET", cookieURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("fetch cookies: %w", err)
	}
	resp.Body.Close()

	// Step 2: Get crumb.
	req, err = http.NewRequestWithContext(ctx, "GET", crumbURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err = c.http.Do(req)
	if err != nil {
		return fmt.Errorf("fetch crumb: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("crumb endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read crumb: %w", err)
	}

	c.crumb = strings.TrimSpace(string(body))
	if c.crumb == "" {
		return fmt.Errorf("empty crumb received")
	}
	c.inited = true
	return nil
}

func (c *Client) fetchQuotes(ctx context.Context, symbols []string) ([]Quote, error) {
	url := fmt.Sprintf("%s?symbols=%s&crumb=%s",
		quoteURL, strings.Join(symbols, ","), c.crumb)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("quote endpoint returned %d", resp.StatusCode)
	}

	var result struct {
		QuoteResponse struct {
			Result []Quote `json:"result"`
			Error  *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"quoteResponse"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if e := result.QuoteResponse.Error; e != nil {
		return nil, fmt.Errorf("yahoo api error: %s: %s", e.Code, e.Description)
	}

	return result.QuoteResponse.Result, nil
}

// fallbackTransport tries HTTP/2 first, falls back to HTTP/1.1 on error.
type fallbackTransport struct {
	h2 *http2.Transport
	h1 *http.Transport
}

func (t *fallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.h2.RoundTrip(req)
	if err == nil {
		return resp, nil
	}
	return t.h1.RoundTrip(req)
}

func newChromeTransport() http.RoundTripper {
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		host, _, _ := net.SplitHostPort(addr)
		tlsConn := utls.UClient(conn, &utls.Config{ServerName: host}, utls.HelloChrome_Auto)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, err
		}
		return tlsConn, nil
	}

	h2 := &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return dial(ctx, network, addr)
		},
	}
	h1 := &http.Transport{
		DialTLSContext: dial,
	}
	return &fallbackTransport{h2: h2, h1: h1}
}
