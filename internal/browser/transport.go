package browser

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

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

// NewChromeTransport creates an http.RoundTripper that mimics Chrome's TLS
// fingerprint using utls, with HTTP/2 to HTTP/1.1 fallback.
func NewChromeTransport(opts ...TransportOption) http.RoundTripper {
	o := transportOptions{}
	for _, opt := range opts {
		opt(&o)
	}

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

	if o.proxyURL != nil {
		h1.Proxy = http.ProxyURL(o.proxyURL)
	}

	return &fallbackTransport{h2: h2, h1: h1}
}

// ClientOption configures a client created by NewClient.
type ClientOption func(*clientOptions)

type clientOptions struct {
	timeout  time.Duration
	proxyURL *url.URL
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(o *clientOptions) { o.timeout = d }
}

// WithProxy sets the HTTP proxy URL for the client's transport.
func WithProxy(proxyURL string) ClientOption {
	return func(o *clientOptions) {
		if u, err := url.Parse(proxyURL); err == nil {
			o.proxyURL = u
		}
	}
}

// NewClient creates an *http.Client with Chrome TLS fingerprinting and a cookie jar.
func NewClient(opts ...ClientOption) *http.Client {
	o := clientOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	var transportOpts []TransportOption
	if o.proxyURL != nil {
		transportOpts = append(transportOpts, withProxyURL(o.proxyURL))
	}

	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Transport: NewChromeTransport(transportOpts...),
		Jar:       jar,
		Timeout:   o.timeout,
	}
}

// TransportOption configures NewChromeTransport.
type TransportOption func(*transportOptions)

type transportOptions struct {
	proxyURL *url.URL
}

func withProxyURL(u *url.URL) TransportOption {
	return func(o *transportOptions) { o.proxyURL = u }
}
