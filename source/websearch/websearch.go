package websearch

import (
	"context"
	"time"

	"github.com/priyanshujain/openbotkit/internal/browser"
	"github.com/priyanshujain/openbotkit/source"
	"github.com/priyanshujain/openbotkit/source/websearch/httpclient"
	"github.com/priyanshujain/openbotkit/store"
)

type WebSearch struct {
	cfg      Config
	db       *store.DB
	health   *healthTracker
	client   *httpclient.Client
	skipSSRF bool // for testing only
}

type Option func(*WebSearch)

func WithDB(db *store.DB) Option {
	return func(w *WebSearch) {
		w.db = db
	}
}

func New(cfg Config, opts ...Option) *WebSearch {
	w := &WebSearch{cfg: cfg, health: newHealthTracker()}
	for _, opt := range opts {
		opt(w)
	}
	w.client = w.buildHTTPClient()
	return w
}

func (w *WebSearch) Name() string {
	return "websearch"
}

func (w *WebSearch) Status(_ context.Context, db *store.DB) (*source.Status, error) {
	st := &source.Status{Connected: true}
	if db != nil {
		var count int64
		err := db.QueryRow("SELECT COUNT(*) FROM search_history").Scan(&count)
		if err == nil {
			st.ItemCount = count
		}
	}
	return st, nil
}

const defaultCacheTTL = 15 * time.Minute

func (w *WebSearch) cacheTTL() time.Duration {
	if w.cfg.WebSearch != nil && w.cfg.WebSearch.CacheTTL != "" {
		if d, err := time.ParseDuration(w.cfg.WebSearch.CacheTTL); err == nil {
			return d
		}
	}
	return defaultCacheTTL
}

func (w *WebSearch) configuredBackends() []string {
	if w.cfg.WebSearch != nil {
		return w.cfg.WebSearch.Backends
	}
	return nil
}

func (w *WebSearch) httpClient() *httpclient.Client {
	return w.client
}

func (w *WebSearch) buildHTTPClient() *httpclient.Client {
	var browserOpts []browser.ClientOption

	if w.cfg.WebSearch != nil {
		if w.cfg.WebSearch.Timeout != "" {
			if d, err := time.ParseDuration(w.cfg.WebSearch.Timeout); err == nil {
				browserOpts = append(browserOpts, browser.WithTimeout(d))
			}
		}
		if w.cfg.WebSearch.Proxy != "" {
			browserOpts = append(browserOpts, browser.WithProxy(w.cfg.WebSearch.Proxy))
		}
	}

	return httpclient.New(browserOpts)
}
