package client

import (
	"fmt"
	"net/http"

	"github.com/73ai/openbotkit/internal/browser"
	"golang.org/x/time/rate"
)

type Client struct {
	session   *Session
	http      *http.Client
	endpoints map[string]Endpoint
	limiter   *rate.Limiter
	baseURL   string // override for testing; defaults to graphqlBase
}

func NewClient(session *Session, endpointsPath string) (*Client, error) {
	if session == nil {
		return nil, fmt.Errorf("session is required")
	}
	if session.AuthToken == "" {
		return nil, fmt.Errorf("auth_token is required")
	}

	endpoints, err := LoadEndpoints(endpointsPath)
	if err != nil {
		return nil, fmt.Errorf("load endpoints: %w", err)
	}

	return &Client{
		session:   session,
		http:      browser.NewClient(),
		endpoints: endpoints,
		limiter:   rate.NewLimiter(1, 3), // 1 req/sec, burst 3
	}, nil
}

// NewClientWithHTTP creates a client with a custom http.Client (for testing).
// If baseURL is non-empty, it overrides the default GraphQL base URL.
func NewClientWithHTTP(session *Session, httpClient *http.Client, endpoints map[string]Endpoint, baseURL string) *Client {
	return &Client{
		session:   session,
		http:      httpClient,
		endpoints: endpoints,
		limiter:   rate.NewLimiter(rate.Inf, 1), // no limit for tests
		baseURL:   baseURL,
	}
}

func (c *Client) graphqlBaseURL() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return graphqlBase
}
