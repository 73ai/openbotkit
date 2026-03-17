package httpclient

import (
	"net/http"

	"github.com/73ai/openbotkit/internal/browser"
)

type Client struct {
	inner   *http.Client
	profile headerProfile
	limiter *hostRateLimiter
}

type Option func(*Client)

func WithBrowserClient(c *http.Client) Option {
	return func(cl *Client) { cl.inner = c }
}

func New(browserOpts []browser.ClientOption, opts ...Option) *Client {
	c := &Client{
		profile: randomProfile(),
		limiter: newHostRateLimiter(),
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.inner == nil {
		c.inner = browser.NewClient(browserOpts...)
	}
	return c
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(req.Context(), req.URL.Host); err != nil {
		return nil, err
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.profile.UserAgent)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", c.profile.Accept)
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", c.profile.AcceptLanguage)
	}

	return c.inner.Do(req)
}
